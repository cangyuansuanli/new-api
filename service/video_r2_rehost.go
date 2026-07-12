package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const maxGeneratedVideoBytes = 256 * 1024 * 1024

func extensionForVideoMime(mimeType string, sourceURL string) string {
	switch strings.ToLower(strings.TrimSpace(strings.Split(mimeType, ";")[0])) {
	case "video/webm":
		return ".webm"
	case "video/quicktime":
		return ".mov"
	default:
		if ext := extensionFromURLPath(sourceURL); ext != "" {
			return ext
		}
		return ".mp4"
	}
}

func extensionFromURLPath(rawURL string) string {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return ""
	}
	path := strings.ToLower(u.Path)
	switch {
	case strings.HasSuffix(path, ".webm"):
		return ".webm"
	case strings.HasSuffix(path, ".mov"):
		return ".mov"
	case strings.HasSuffix(path, ".mp4"):
		return ".mp4"
	default:
		return ""
	}
}

func buildGeneratedVideoObjectKey(userID int, taskID string, ext string) string {
	ext = strings.TrimSpace(ext)
	if ext == "" {
		ext = ".mp4"
	}
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return fmt.Sprintf("gen-videos/%d/%s%s", userID, taskID, ext)
}

func isOurUserCDNURL(rawURL string) bool {
	cfg := getR2Config()
	if cfg == nil {
		return false
	}
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return false
	}
	base, err := url.Parse(cfg.PublicBase)
	if err != nil {
		return false
	}
	return strings.EqualFold(u.Host, base.Host)
}

func isVideoProxyURL(rawURL string) bool {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return false
	}
	server := strings.TrimRight(strings.TrimSpace(system_setting.ServerAddress), "/")
	if server == "" {
		return false
	}
	return strings.HasPrefix(rawURL, server+"/v1/videos/") && strings.HasSuffix(rawURL, "/content")
}

// VideoURLNeedsRehost reports whether an upstream mp4 URL should be copied to R2_USER_BUCKET.
func VideoURLNeedsRehost(rawURL string) bool {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" || strings.HasPrefix(rawURL, "data:") {
		return false
	}
	if isOurUserCDNURL(rawURL) || isVideoProxyURL(rawURL) {
		return false
	}
	return getR2Config() != nil
}

func UploadGeneratedVideoBytes(ctx context.Context, userID int, taskID string, data []byte, mimeType, sourceURL string) (*R2UploadResult, error) {
	cfg := getR2Config()
	if cfg == nil {
		return nil, fmt.Errorf("R2 not configured")
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("empty video data")
	}
	if mimeType == "" {
		mimeType = "video/mp4"
	}
	client, err := newR2S3Client(cfg)
	if err != nil {
		return nil, err
	}
	ext := extensionForVideoMime(mimeType, sourceURL)
	objectKey := buildGeneratedVideoObjectKey(userID, taskID, ext)
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

func UploadGeneratedVideoFromURL(ctx context.Context, userID int, taskID, videoURL string) (*R2UploadResult, error) {
	videoURL = strings.TrimSpace(videoURL)
	if videoURL == "" {
		return nil, fmt.Errorf("empty video url")
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
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, videoURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download video failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("download video HTTP %d: %s", resp.StatusCode, string(body))
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, int64(maxGeneratedVideoBytes)+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxGeneratedVideoBytes {
		return nil, fmt.Errorf("video exceeds %dMB upload limit", maxGeneratedVideoBytes/(1024*1024))
	}
	mimeType := resp.Header.Get("Content-Type")
	if mimeType == "" || mimeType == "application/octet-stream" {
		mimeType = "video/mp4"
	}
	uploaded, err := UploadGeneratedVideoBytes(ctx, userID, taskID, data, mimeType, videoURL)
	if err != nil {
		return nil, err
	}
	ext := extensionForVideoMime(mimeType, videoURL)
	if duration, probeErr := common.GetAudioDuration(ctx, bytes.NewReader(data), ext); probeErr == nil && duration > 0 {
		uploaded.DurationSeconds = int(math.Round(duration))
	}
	return uploaded, nil
}

func patchVideoURLInTaskData(data []byte, publicURL string) ([]byte, error) {
	if len(data) == 0 || strings.TrimSpace(publicURL) == "" {
		return data, nil
	}
	prefix := ""
	if gjson.GetBytes(data, "data").IsObject() {
		prefix = "data."
	}
	out := data
	var err error
	for _, path := range []string{prefix + "video_url", prefix + "result_url", prefix + "data.0.url", prefix + "data.0.video_url"} {
		out, err = sjson.SetBytes(out, path, publicURL)
		if err != nil {
			return data, err
		}
	}
	return out, nil
}

func patchVideoUsageSecondsInTaskData(data []byte, seconds int) ([]byte, error) {
	if len(data) == 0 || seconds <= 0 {
		return data, nil
	}
	prefix := ""
	if gjson.GetBytes(data, "data").IsObject() {
		prefix = "data."
	}
	return sjson.SetBytes(data, prefix+"usage.seconds", seconds)
}

// RehostVideoTaskResult copies upstream video to R2 and returns CDN URL plus patched task data.
func RehostVideoTaskResult(ctx context.Context, userID int, channelID int, taskID, upstreamURL string, taskData []byte) (string, []byte, error) {
	if !VideoURLNeedsRehost(upstreamURL) {
		return upstreamURL, taskData, nil
	}
	uploaded, err := UploadGeneratedVideoFromURL(ctx, userID, taskID, upstreamURL)
	if err != nil {
		return upstreamURL, taskData, err
	}
	patched, err := patchVideoURLInTaskData(taskData, uploaded.PublicURL)
	if err != nil {
		return uploaded.PublicURL, taskData, err
	}
	patched, err = patchVideoUsageSecondsInTaskData(patched, uploaded.DurationSeconds)
	if err != nil {
		return uploaded.PublicURL, taskData, err
	}
	return uploaded.PublicURL, patched, nil
}
