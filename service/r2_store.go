package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

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
	PublicURL string
	ObjectKey string
	Bytes     int64
	MimeType  string
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

func UploadGeneratedImageBytes(ctx context.Context, userID int, taskID string, index int, data []byte, mimeType string) (*R2UploadResult, error) {
	cfg := getR2Config()
	if cfg == nil {
		return nil, fmt.Errorf("R2 not configured")
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("empty image data")
	}
	if mimeType == "" {
		mimeType = "image/png"
	}
	client, err := newR2S3Client(cfg)
	if err != nil {
		return nil, err
	}
	objectKey := buildGeneratedImageObjectKey(userID, taskID, index, mimeType)
	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(cfg.Bucket),
		Key:         aws.String(objectKey),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(mimeType),
	})
	if err != nil {
		return nil, fmt.Errorf("r2 put object failed: %w", err)
	}
	return &R2UploadResult{
		PublicURL: publicURLForObject(cfg, objectKey),
		ObjectKey: objectKey,
		Bytes:     int64(len(data)),
		MimeType:  mimeType,
	}, nil
}

func UploadGeneratedImageFromURL(ctx context.Context, userID int, taskID string, index int, imageURL string) (*R2UploadResult, error) {
	client := &http.Client{
		Timeout: 300 * time.Second,
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
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download image failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("download image HTTP %d: %s", resp.StatusCode, string(body))
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, int64(constantMaxGeneratedImageBytes)))
	if err != nil {
		return nil, err
	}
	mimeType := resp.Header.Get("Content-Type")
	if mimeType == "" || mimeType == "application/octet-stream" {
		mimeType = "image/png"
	}
	return UploadGeneratedImageBytes(ctx, userID, taskID, index, data, mimeType)
}

const constantMaxGeneratedImageBytes = 32 * 1024 * 1024
