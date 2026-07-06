package openai

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

var manjuMarkdownHTTPImageRE = regexp.MustCompile(`!\[[^\]]*\]\((https?://[^)]+)\)`)

const (
	defaultManjuBananaPollInterval = 3 * time.Second
	defaultManjuBananaPollTimeout  = 180 * time.Second
)

// IsManjuBananaOriginModel：Manju Gemini Banana 渠道（manjuapi 上游）走 chat/completions 异步任务适配。
func IsManjuBananaOriginModel(originModel string) bool {
	name := strings.ToLower(strings.TrimSpace(originModel))
	return strings.HasPrefix(name, "manju-gemini-banana")
}

// AdaptManjuBananaChatCompletionResponse 将上游异步任务或 URL 出图响应转为下游同步 data URI Markdown。
func AdaptManjuBananaChatCompletionResponse(ctx context.Context, info *relaycommon.RelayInfo, responseBody []byte) ([]byte, *types.NewAPIError) {
	if info == nil || len(responseBody) == 0 || !gjson.ValidBytes(responseBody) {
		return responseBody, nil
	}

	status := strings.ToLower(strings.TrimSpace(gjson.GetBytes(responseBody, "status").String()))
	if isManjuUpstreamTaskPending(status) {
		polled, pollErr := pollManjuBananaTask(ctx, info, responseBody)
		if pollErr != nil {
			return nil, pollErr
		}
		responseBody = polled
	}

	normalized, normErr := normalizeManjuBananaChatBody(ctx, responseBody)
	if normErr != nil {
		return nil, normErr
	}
	return stripManjuUpstreamTaskFields(normalized), nil
}

func isManjuUpstreamTaskPending(status string) bool {
	switch status {
	case "running", "queued", "pending", "in_progress", "processing":
		return true
	default:
		return false
	}
}

func isManjuUpstreamTaskSucceeded(status string) bool {
	switch status {
	case "succeeded", "success", "completed", "done":
		return true
	default:
		return false
	}
}

func isManjuUpstreamTaskFailed(status string) bool {
	switch status {
	case "failed", "failure", "error", "cancelled", "canceled":
		return true
	default:
		return false
	}
}

func manjuBananaPollInterval() time.Duration {
	if v := strings.TrimSpace(os.Getenv("MANJU_BANANA_POLL_INTERVAL")); v != "" {
		if sec, err := strconv.Atoi(v); err == nil && sec > 0 {
			return time.Duration(sec) * time.Second
		}
	}
	return defaultManjuBananaPollInterval
}

func manjuBananaPollTimeout() time.Duration {
	if v := strings.TrimSpace(os.Getenv("MANJU_BANANA_POLL_TIMEOUT")); v != "" {
		if sec, err := strconv.Atoi(v); err == nil && sec > 0 {
			return time.Duration(sec) * time.Second
		}
	}
	return defaultManjuBananaPollTimeout
}

func pollManjuBananaTask(ctx context.Context, info *relaycommon.RelayInfo, createBody []byte) ([]byte, *types.NewAPIError) {
	pollURL := strings.TrimSpace(gjson.GetBytes(createBody, "poll_url").String())
	taskID := strings.TrimSpace(gjson.GetBytes(createBody, "task_id").String())
	if pollURL == "" && taskID != "" {
		base := strings.TrimRight(strings.TrimSpace(info.ChannelBaseUrl), "/")
		if base != "" {
			pollURL = base + "/api/tasks/" + taskID
		}
	}
	if pollURL == "" {
		return nil, types.NewOpenAIError(fmt.Errorf("upstream returned async task without poll_url"), types.ErrorCodeBadResponse, http.StatusBadGateway)
	}

	deadline := time.Now().Add(manjuBananaPollTimeout())
	interval := manjuBananaPollInterval()
	for {
		if ctx.Err() != nil {
			return nil, types.NewOpenAIError(ctx.Err(), types.ErrorCodeDoRequestFailed, http.StatusGatewayTimeout)
		}
		if time.Now().After(deadline) {
			return nil, types.NewOpenAIError(fmt.Errorf("manju image task timed out after %s", manjuBananaPollTimeout()), types.ErrorCodeBadResponse, http.StatusGatewayTimeout)
		}

		body, err := fetchManjuPollURL(ctx, info, pollURL)
		if err != nil {
			return nil, types.NewOpenAIError(err, types.ErrorCodeDoRequestFailed, http.StatusBadGateway)
		}
		if !gjson.ValidBytes(body) {
			return nil, types.NewOpenAIError(fmt.Errorf("invalid manju poll response"), types.ErrorCodeBadResponse, http.StatusBadGateway)
		}

		status := strings.ToLower(strings.TrimSpace(gjson.GetBytes(body, "status").String()))
		if isManjuUpstreamTaskSucceeded(status) {
			return body, nil
		}
		if isManjuUpstreamTaskFailed(status) {
			reason := strings.TrimSpace(gjson.GetBytes(body, "fail_reason").String())
			if reason == "" {
				reason = strings.TrimSpace(gjson.GetBytes(body, "error").String())
			}
			if reason == "" {
				reason = "upstream image task failed"
			}
			return nil, types.NewOpenAIError(fmt.Errorf("%s", reason), types.ErrorCodeBadResponse, http.StatusBadGateway)
		}

		select {
		case <-ctx.Done():
			return nil, types.NewOpenAIError(ctx.Err(), types.ErrorCodeDoRequestFailed, http.StatusGatewayTimeout)
		case <-time.After(interval):
		}
	}
}

func fetchManjuPollURL(ctx context.Context, info *relaycommon.RelayInfo, pollURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pollURL, nil)
	if err != nil {
		return nil, err
	}
	if info != nil && strings.TrimSpace(info.ApiKey) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(info.ApiKey))
	}
	req.Header.Set("Accept", "application/json")

	client := service.GetHttpClient()
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("manju poll HTTP %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

func normalizeManjuBananaChatBody(ctx context.Context, body []byte) ([]byte, *types.NewAPIError) {
	content := gjson.GetBytes(body, "choices.0.message.content").String()
	if strings.Contains(content, "data:image/") {
		return ensureManjuBananaFinishReason(body), nil
	}

	imageURL := extractManjuImageURL(body, content)
	if imageURL == "" {
		return body, nil
	}

	markdown, err := imageURLToDataURIMarkdown(ctx, imageURL)
	if err != nil {
		return nil, types.NewOpenAIError(fmt.Errorf("convert manju image url: %w", err), types.ErrorCodeBadResponse, http.StatusBadGateway)
	}

	patched, err := sjson.SetBytes(body, "choices.0.message.content", markdown)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponse, http.StatusInternalServerError)
	}
	return ensureManjuBananaFinishReason(patched), nil
}

func ensureManjuBananaFinishReason(body []byte) []byte {
	if gjson.GetBytes(body, "choices.0.finish_reason").String() != "" {
		return body
	}
	patched, err := sjson.SetBytes(body, "choices.0.finish_reason", "stop")
	if err != nil {
		return body
	}
	return patched
}

func extractManjuImageURL(body []byte, content string) string {
	if match := manjuMarkdownHTTPImageRE.FindStringSubmatch(content); len(match) > 1 {
		return strings.TrimSpace(match[1])
	}
	for _, path := range []string{
		"result_url",
		"download_url",
		"url",
		"image_url",
		"data.url",
		"data.image_url",
		"data.result_url",
		"data.download_url",
		"result.url",
		"result.image_url",
		"result.result_url",
	} {
		if u := strings.TrimSpace(gjson.GetBytes(body, path).String()); u != "" && strings.HasPrefix(u, "http") {
			return u
		}
	}
	if data := gjson.GetBytes(body, "data"); data.IsArray() {
		for _, item := range data.Array() {
			for _, key := range []string{"url", "image_url"} {
				if u := strings.TrimSpace(item.Get(key).String()); u != "" && strings.HasPrefix(u, "http") {
					return u
				}
			}
		}
	}
	return ""
}

func imageURLToDataURIMarkdown(ctx context.Context, imageURL string) (string, error) {
	data, mimeType, err := downloadImageBytes(ctx, imageURL)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(mimeType, "image/") {
		mimeType = http.DetectContentType(data)
	}
	if !strings.HasPrefix(mimeType, "image/") {
		mimeType = "image/jpeg"
	}
	encoded := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("![image](data:%s;base64,%s)", mimeType, encoded), nil
}

func downloadImageBytes(ctx context.Context, imageURL string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return nil, "", err
	}
	client := service.GetHttpClient()
	if client == nil {
		client = &http.Client{Timeout: 120 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, "", fmt.Errorf("download image HTTP %d: %s", resp.StatusCode, string(body))
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, "", err
	}
	mimeType := strings.TrimSpace(resp.Header.Get("Content-Type"))
	if idx := strings.Index(mimeType, ";"); idx >= 0 {
		mimeType = strings.TrimSpace(mimeType[:idx])
	}
	return data, mimeType, nil
}

func stripManjuUpstreamTaskFields(body []byte) []byte {
	result := body
	for _, key := range []string{
		"task_id",
		"poll_url",
		"status",
		"progress",
		"detail_url",
		"download_url",
		"result_url",
		"final_url",
		"url",
		"image_url",
		"image_urls",
		"result",
		"data",
	} {
		if !gjson.GetBytes(result, key).Exists() {
			continue
		}
		next, err := sjson.DeleteBytes(result, key)
		if err == nil {
			result = next
		}
	}
	if gjson.GetBytes(result, "object").String() == "" {
		if patched, err := sjson.SetBytes(result, "object", "chat.completion"); err == nil {
			result = patched
		}
	}
	return result
}

func manjuBananaAdaptIfNeeded(ctx context.Context, info *relaycommon.RelayInfo, responseBody []byte) ([]byte, *types.NewAPIError) {
	if !IsManjuBananaOriginModel(info.OriginModelName) {
		return responseBody, nil
	}
	return AdaptManjuBananaChatCompletionResponse(ctx, info, responseBody)
}
