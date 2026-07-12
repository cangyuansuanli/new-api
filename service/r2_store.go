package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type R2Config struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	PublicBase      string
	Endpoint        string
}

type R2UploadResult struct {
	PublicURL       string
	ObjectKey       string
	Bytes           int64
	MimeType        string
	DurationSeconds int
}

func resolveS3Endpoint(accountID string) string {
	if endpoint := strings.TrimSpace(os.Getenv("S3_ENDPOINT")); endpoint != "" {
		return strings.TrimRight(endpoint, "/")
	}
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return ""
	}
	return fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)
}

func getR2Config() *R2Config {
	accountID := strings.TrimSpace(os.Getenv("R2_ACCOUNT_ID"))
	accessKeyID := strings.TrimSpace(os.Getenv("R2_ACCESS_KEY_ID"))
	secretAccessKey := strings.TrimSpace(os.Getenv("R2_SECRET_ACCESS_KEY"))
	// 生图结果与临时参考素材共用 R2_USER_BUCKET（uers-assets / tmp 域名），勿用 pro 运营桶。
	bucket := strings.TrimSpace(os.Getenv("R2_USER_BUCKET"))
	publicBase := strings.TrimRight(strings.TrimSpace(os.Getenv("R2_USER_PUBLIC_BASE_URL")), "/")
	endpoint := resolveS3Endpoint(accountID)
	if endpoint == "" || accessKeyID == "" || secretAccessKey == "" || bucket == "" || publicBase == "" {
		return nil
	}
	return &R2Config{
		AccountID:       accountID,
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
		Bucket:          bucket,
		PublicBase:      publicBase,
		Endpoint:        endpoint,
	}
}

func newR2S3Client(cfg *R2Config) (*s3.Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("r2 config is nil")
	}
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("s3 endpoint is empty")
	}
	return s3.New(s3.Options{
		Region:       "auto",
		BaseEndpoint: aws.String(cfg.Endpoint),
		Credentials:  credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		UsePathStyle: true,
	}), nil
}

func extensionForImageMime(mimeType string) string {
	switch strings.ToLower(strings.TrimSpace(mimeType)) {
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	default:
		return ".png"
	}
}

func buildGeneratedImageObjectKey(userID int, taskID string, index int, mimeType string) string {
	return fmt.Sprintf("gen-images/%d/%s/%d%s", userID, taskID, index, extensionForImageMime(mimeType))
}

func publicURLForObject(cfg *R2Config, objectKey string) string {
	return cfg.PublicBase + "/" + strings.TrimPrefix(objectKey, "/")
}

var (
	imageR2LimiterOnce      sync.Once
	imageR2Limiter          chan struct{}
	imageTransferURLPattern = regexp.MustCompile(`https?://[^\s"']+`)
)

func redactImageTransferError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s", imageTransferURLPattern.ReplaceAllString(err.Error(), "[upstream-url-redacted]"))
}

func acquireImageR2Slot(ctx context.Context) (func(), error) {
	imageR2LimiterOnce.Do(func() {
		imageR2Limiter = make(chan struct{}, common.GetEnvOrDefault("IMAGE_R2_MAX_CONCURRENT", 16))
	})
	select {
	case imageR2Limiter <- struct{}{}:
		return func() { <-imageR2Limiter }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func UploadGeneratedImageBytes(ctx context.Context, userID int, taskID string, index int, data []byte, mimeType string) (*R2UploadResult, error) {
	release, err := acquireImageR2Slot(ctx)
	if err != nil {
		return nil, err
	}
	defer release()
	return uploadGeneratedImageReader(ctx, userID, taskID, index, bytes.NewReader(data), int64(len(data)), mimeType)
}

func uploadGeneratedImageReader(ctx context.Context, userID int, taskID string, index int, body io.Reader, size int64, mimeType string) (*R2UploadResult, error) {
	objectKey := buildGeneratedImageObjectKey(userID, taskID, index, mimeType)
	return uploadImageReaderToObjectKey(ctx, objectKey, body, size, mimeType)
}

func uploadImageReaderToObjectKey(ctx context.Context, objectKey string, body io.Reader, size int64, mimeType string) (*R2UploadResult, error) {
	cfg := getR2Config()
	if cfg == nil {
		return nil, fmt.Errorf("R2 not configured")
	}
	if size <= 0 {
		return nil, fmt.Errorf("empty image data")
	}
	if mimeType == "" {
		mimeType = "image/png"
	}
	client, err := newR2S3Client(cfg)
	if err != nil {
		return nil, err
	}
	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(cfg.Bucket),
		Key:           aws.String(objectKey),
		Body:          body,
		ContentLength: aws.Int64(size),
		ContentType:   aws.String(mimeType),
	})
	if err != nil {
		return nil, fmt.Errorf("r2 put object failed: %w", err)
	}
	return &R2UploadResult{
		PublicURL: publicURLForObject(cfg, objectKey),
		ObjectKey: objectKey,
		Bytes:     size,
		MimeType:  mimeType,
	}, nil
}

func UploadImageTaskInput(ctx context.Context, userID int, taskID string, index int, body io.Reader, size int64, mimeType string) (*R2UploadResult, error) {
	if size <= 0 || size > constantMaxImageTaskInputBytes {
		return nil, fmt.Errorf("image task input size must be between 1 and %d bytes", constantMaxImageTaskInputBytes)
	}
	release, err := acquireImageR2Slot(ctx)
	if err != nil {
		return nil, err
	}
	defer release()
	extension := extensionForImageMime(mimeType)
	objectKey := fmt.Sprintf("image-task-inputs/%d/%s/%d%s", userID, taskID, index, extension)
	return uploadImageReaderToObjectKey(ctx, objectKey, body, size, mimeType)
}

func DeleteImageTaskInput(ctx context.Context, objectKey string) error {
	objectKey = strings.TrimSpace(objectKey)
	if objectKey == "" {
		return nil
	}
	cfg := getR2Config()
	if cfg == nil {
		return fmt.Errorf("R2 not configured")
	}
	client, err := newR2S3Client(cfg)
	if err != nil {
		return err
	}
	_, err = client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(cfg.Bucket),
		Key:    aws.String(objectKey),
	})
	return err
}

func OpenImageTaskInput(ctx context.Context, objectKey string) (io.ReadCloser, error) {
	objectKey = strings.TrimSpace(objectKey)
	if objectKey == "" {
		return nil, fmt.Errorf("empty image task input object key")
	}
	release, err := acquireImageR2Slot(ctx)
	if err != nil {
		return nil, err
	}
	cfg := getR2Config()
	if cfg == nil {
		release()
		return nil, fmt.Errorf("R2 not configured")
	}
	client, err := newR2S3Client(cfg)
	if err != nil {
		release()
		return nil, err
	}
	result, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(cfg.Bucket),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		release()
		return nil, fmt.Errorf("r2 get image task input failed: %w", err)
	}
	return &imageTaskInputBody{ReadCloser: result.Body, release: release}, nil
}

type imageTaskInputBody struct {
	io.ReadCloser
	releaseOnce sync.Once
	release     func()
}

func (body *imageTaskInputBody) Close() error {
	err := body.ReadCloser.Close()
	body.releaseOnce.Do(body.release)
	return err
}

func UploadGeneratedImageFromURL(ctx context.Context, userID int, taskID string, index int, imageURL string) (*R2UploadResult, error) {
	release, err := acquireImageR2Slot(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	client := &http.Client{
		Timeout:   300 * time.Second,
		Transport: GetHttpClient().Transport,
	}
	if client.Transport == nil {
		client = GetHttpClient()
		if client == nil {
			client = &http.Client{Timeout: 300 * time.Second}
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return nil, redactImageTransferError(err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download image failed: %w", redactImageTransferError(err))
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		message := imageTransferURLPattern.ReplaceAllString(string(body), "[upstream-url-redacted]")
		return nil, fmt.Errorf("download image HTTP %d: %s", resp.StatusCode, message)
	}
	if resp.ContentLength > constantMaxGeneratedImageBytes {
		return nil, fmt.Errorf("generated image exceeds %d bytes", constantMaxGeneratedImageBytes)
	}
	tmp, err := os.CreateTemp("", "new-api-image-transfer-*")
	if err != nil {
		return nil, err
	}
	defer func() {
		tmp.Close()
		os.Remove(tmp.Name())
	}()
	written, err := io.Copy(tmp, io.LimitReader(resp.Body, int64(constantMaxGeneratedImageBytes)+1))
	if err != nil {
		return nil, err
	}
	if written > constantMaxGeneratedImageBytes {
		return nil, fmt.Errorf("generated image exceeds %d bytes", constantMaxGeneratedImageBytes)
	}
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	mimeType := resp.Header.Get("Content-Type")
	if mimeType == "" || mimeType == "application/octet-stream" {
		mimeType = "image/png"
	}
	return uploadGeneratedImageReader(ctx, userID, taskID, index, tmp, written, mimeType)
}

const constantMaxGeneratedImageBytes = 32 * 1024 * 1024
const constantMaxImageTaskInputBytes = 20 * 1024 * 1024
