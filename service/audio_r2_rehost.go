package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const maxGeneratedAudioBytes = 128 * 1024 * 1024

func extensionForAudioMime(mimeType, sourceURL string) string {
	switch strings.ToLower(strings.TrimSpace(strings.Split(mimeType, ";")[0])) {
	case "audio/mpeg", "audio/mp3":
		return ".mp3"
	case "audio/wav", "audio/x-wav":
		return ".wav"
	case "audio/mp4", "audio/x-m4a":
		return ".m4a"
	case "audio/ogg":
		return ".ogg"
	case "audio/flac":
		return ".flac"
	default:
		if ext := extensionFromURLPath(sourceURL); ext != "" {
			return ext
		}
		return ".mp3"
	}
}

func buildGeneratedAudioObjectKey(userID int, taskID string, ext string) string {
	ext = strings.TrimSpace(ext)
	if ext == "" {
		ext = ".mp3"
	}
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return fmt.Sprintf("gen-audio/%d/%s%s", userID, taskID, ext)
}

// AudioURLNeedsRehost reports whether an upstream audio URL should be copied to R2_USER_BUCKET.
func AudioURLNeedsRehost(rawURL string) bool {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" || strings.HasPrefix(rawURL, "data:") {
		return false
	}
	if isOurUserCDNURL(rawURL) {
		return false
	}
	return getR2Config() != nil
}

func UploadGeneratedAudioBytes(ctx context.Context, userID int, taskID string, data []byte, mimeType, sourceURL string) (*R2UploadResult, error) {
	cfg := getR2Config()
	if cfg == nil {
		return nil, fmt.Errorf("R2 not configured")
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("empty audio data")
	}
	if mimeType == "" {
		mimeType = "audio/mpeg"
	}
	client, err := newR2S3Client(cfg)
	if err != nil {
		return nil, err
	}
	ext := extensionForAudioMime(mimeType, sourceURL)
	objectKey := buildGeneratedAudioObjectKey(userID, taskID, ext)
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

func UploadGeneratedAudioFromURL(ctx context.Context, userID int, taskID, audioURL string, sourceHeaders http.Header) (*R2UploadResult, error) {
	audioURL = strings.TrimSpace(audioURL)
	if audioURL == "" {
		return nil, fmt.Errorf("empty audio url")
	}
	client := &http.Client{
		Timeout:   600 * time.Second,
		Transport: GetHttpClient().Transport,
	}
	if client.Transport == nil {
		client = GetHttpClient()
		if client == nil {
			client = &http.Client{Timeout: 600 * time.Second}
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, audioURL, nil)
	if err != nil {
		return nil, err
	}
	for name, values := range sourceHeaders {
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download audio failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("download audio HTTP %d: %s", resp.StatusCode, string(body))
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, int64(maxGeneratedAudioBytes)+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxGeneratedAudioBytes {
		return nil, fmt.Errorf("audio exceeds %dMB upload limit", maxGeneratedAudioBytes/(1024*1024))
	}
	mimeType := resp.Header.Get("Content-Type")
	if mimeType == "" || mimeType == "application/octet-stream" {
		mimeType = "audio/mpeg"
	}
	return UploadGeneratedAudioBytes(ctx, userID, taskID, data, mimeType, audioURL)
}

func patchAudioURLInTaskData(data []byte, publicURL string) ([]byte, error) {
	if len(data) == 0 || strings.TrimSpace(publicURL) == "" {
		if len(data) == 0 {
			out, err := sjson.SetBytes([]byte("{}"), "result_url", publicURL)
			if err != nil {
				return nil, err
			}
			return sjson.SetBytes(out, "data.0.url", publicURL)
		}
		return data, nil
	}
	prefix := ""
	if gjson.GetBytes(data, "data").IsObject() {
		prefix = "data."
	}
	out := data
	var err error
	for _, path := range []string{prefix + "result_url", prefix + "url", prefix + "data.0.url"} {
		out, err = sjson.SetBytes(out, path, publicURL)
		if err != nil {
			return data, err
		}
	}
	return out, nil
}

// RehostAudioTaskResult copies upstream audio to R2 and returns CDN URL plus patched task data.
func RehostAudioTaskResult(ctx context.Context, userID int, taskID, upstreamURL string, taskData []byte, sourceHeaders http.Header) (string, []byte, error) {
	if !AudioURLNeedsRehost(upstreamURL) {
		return upstreamURL, taskData, nil
	}
	uploaded, err := UploadGeneratedAudioFromURL(ctx, userID, taskID, upstreamURL, sourceHeaders)
	if err != nil {
		return "", taskData, err
	}
	patched, err := patchAudioURLInTaskData(taskData, uploaded.PublicURL)
	if err != nil {
		return uploaded.PublicURL, taskData, err
	}
	return uploaded.PublicURL, patched, nil
}

// RehostSyncAudioURL rehosts a synchronous audio generation result for client delivery.
func RehostSyncAudioURL(ctx context.Context, userID int, storeID, upstreamURL string, sourceHeaders http.Header) (string, error) {
	publicURL, _, err := RehostAudioTaskResult(ctx, userID, storeID, upstreamURL, nil, sourceHeaders)
	return publicURL, err
}
