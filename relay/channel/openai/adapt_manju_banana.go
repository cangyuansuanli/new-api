package openai

// Manju Gemini Banana 适配层（对齐 Manju OpenAI 兼容 API 文档）。
//
// 文生图：buildManjuBananaImageBody → 上游 POST /v1/images/generations（model=UpstreamModelName）
// 图生图（edits / 带参考图）：ConvertImageRequestForChatImage → POST /v1/chat/completions + image_url
//
// Legacy chat 响应（含 async poll）：
//   manjuBananaAdaptIfNeeded → AdaptManjuBananaChatCompletionResponse

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

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relay/imagevendor"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

var manjuMarkdownHTTPImageRE = regexp.MustCompile(`!\[[^\]]*\]\((https?://[^)]+)\)`)

const (
	defaultManjuBananaPollInterval = 3 * time.Second
	defaultManjuBananaPollTimeout  = 180 * time.Second
)

// BuildManjuBananaImageGenerationBody 将 OpenAI Image 请求转为 Manju 上游 /v1/images/generations body。
func BuildManjuBananaImageGenerationBody(originModel string, request dto.ImageRequest) map[string]any {
	info := &relaycommon.RelayInfo{
		OriginModelName: originModel,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: request.Model,
		},
	}
	body, _ := buildManjuBananaImageBody(nil, info, request)
	return body
}

// ManjuBananaUsesChatCompletionsUpstream 判断 Manju 图生图是否应走 chat/completions + image_url。
func ManjuBananaUsesChatCompletionsUpstream(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) bool {
	if info != nil && info.RelayMode == constant.RelayModeImagesEdits {
		return true
	}
	if hasManjuBananaReferenceInputFromRequest(request) {
		return true
	}
	if c != nil && c.Request != nil && !isJSONRequest(c) {
		if err := ensureMultipartFormParsed(c); err == nil && hasManjuBananaMultipartReference(c) {
			return true
		}
	}
	return false
}

// ManjuBananaUsesChatCompletionsUpstreamFromInfo 供 GetRequestURL / DoResponse 等无完整 request 副本时使用。
func ManjuBananaUsesChatCompletionsUpstreamFromInfo(info *relaycommon.RelayInfo) bool {
	if info == nil {
		return false
	}
	if info.RelayMode == constant.RelayModeImagesEdits {
		return true
	}
	req, ok := info.Request.(*dto.ImageRequest)
	if !ok || req == nil {
		return false
	}
	return hasManjuBananaReferenceInputFromRequest(*req)
}

func hasManjuBananaReferenceInputFromRequest(request dto.ImageRequest) bool {
	if len(parseJSONStringList(request.Image)) > 0 || len(parseJSONStringList(request.Images)) > 0 {
		return true
	}
	return len(parseJSONStringList(request.Mask)) > 0
}

func hasManjuBananaMultipartReference(c *gin.Context) bool {
	if c == nil || c.Request == nil || c.Request.MultipartForm == nil {
		return false
	}
	mf := c.Request.MultipartForm
	for _, key := range []string{"image", "image[]"} {
		if files, ok := mf.File[key]; ok && len(files) > 0 {
			return true
		}
	}
	for fieldName, files := range mf.File {
		if strings.HasPrefix(fieldName, "image[") && len(files) > 0 {
			return true
		}
	}
	if maskFiles, ok := mf.File["mask"]; ok && len(maskFiles) > 0 {
		return true
	}
	return false
}

func resolveManjuBananaUpstreamModel(info *relaycommon.RelayInfo, request dto.ImageRequest) string {
	if info != nil && info.ChannelMeta != nil && strings.TrimSpace(info.UpstreamModelName) != "" {
		return strings.TrimSpace(info.UpstreamModelName)
	}
	if strings.TrimSpace(request.Model) != "" {
		return strings.TrimSpace(request.Model)
	}
	if info != nil {
		return strings.TrimSpace(info.OriginModelName)
	}
	return ""
}

// buildManjuBananaImageBody 构建上游文生图 JSON body（无参考图时使用）。
func buildManjuBananaImageBody(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (map[string]any, error) {
	originModel := ""
	if info != nil {
		originModel = info.OriginModelName
	}
	body := map[string]any{
		"model":  resolveManjuBananaUpstreamModel(info, request),
		"prompt": request.Prompt,
	}
	if aspect := resolveManjuBananaAspectRatio(request.Size); aspect != "" {
		body["aspect_ratio"] = aspect
	}
	if resolution := resolveManjuBananaOutputResolution(originModel, request.Quality); resolution != "" {
		body["output_resolution"] = resolution
	}
	if request.N != nil && *request.N > 0 {
		body["n"] = *request.N
	}
	body["stream"] = false
	return body, nil
}

// ConvertManjuBananaImageRequest 按 Manju 文档路由文生图或图生图上游格式。
func ConvertManjuBananaImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (any, error) {
	if ManjuBananaUsesChatCompletionsUpstream(c, info, request) {
		chatReq := request
		chatReq.Model = resolveManjuBananaUpstreamModel(info, request)
		return ConvertImageRequestForChatImage(c, info, chatReq)
	}
	return buildManjuBananaImageBody(c, info, request)
}

func applyManjuBananaReferenceFields(body map[string]any, c *gin.Context, request dto.ImageRequest) error {
	images, mask, err := collectManjuBananaReferenceImages(c, request)
	if err != nil {
		return err
	}
	switch len(images) {
	case 0:
	case 1:
		body["image"] = images[0]
	default:
		body["images"] = images
	}
	if mask != "" {
		body["mask"] = mask
	}
	return nil
}

func collectManjuBananaReferenceImages(c *gin.Context, request dto.ImageRequest) (images []string, mask string, err error) {
	images = append(images, parseJSONStringList(request.Image)...)
	images = append(images, parseJSONStringList(request.Images)...)

	// JSON 文生图/图生图（image/images 在 body 里）无需解析 multipart；强行 ParseMultipartFormReusable 会在
	// Content-Type: application/json 时触发 "multipart boundary not found"。
	if c != nil && c.Request != nil && !isJSONRequest(c) {
		mf := c.Request.MultipartForm
		if mf == nil {
			mf, err = common.ParseMultipartFormReusable(c)
			if err != nil {
				return nil, "", err
			}
			c.Request.MultipartForm = mf
			c.Request.PostForm = mf.Value
		}
		if mf != nil {
			for _, key := range []string{"image", "image[]"} {
				for _, fh := range mf.File[key] {
					dataURI, convErr := multipartFileToDataURI(fh)
					if convErr != nil {
						return nil, "", convErr
					}
					images = append(images, dataURI)
				}
			}
			for fieldName, files := range mf.File {
				if strings.HasPrefix(fieldName, "image[") && len(files) > 0 {
					for _, fh := range files {
						dataURI, convErr := multipartFileToDataURI(fh)
						if convErr != nil {
							return nil, "", convErr
						}
						images = append(images, dataURI)
					}
				}
			}
			if maskFiles, ok := mf.File["mask"]; ok && len(maskFiles) > 0 {
				mask, err = multipartFileToDataURI(maskFiles[0])
				if err != nil {
					return nil, "", err
				}
			}
		}
	}

	maskURLs := parseJSONStringList(request.Mask)
	if mask == "" && len(maskURLs) > 0 {
		mask = maskURLs[0]
	}
	return images, mask, nil
}

func resolveManjuBananaAspectRatio(size string) string {
	value := strings.TrimSpace(size)
	if value == "" || strings.EqualFold(value, "auto") {
		return ""
	}
	if strings.Contains(value, ":") {
		return value
	}
	switch strings.ToLower(value) {
	case "1024x1024":
		return "1:1"
	case "1536x1024":
		return "3:2"
	case "1024x1536":
		return "2:3"
	case "1792x1024", "1920x1080":
		return "16:9"
	case "1024x1792", "1080x1920":
		return "9:16"
	default:
		return ""
	}
}

func resolveManjuBananaOutputResolution(originModel, quality string) string {
	name := strings.ToLower(strings.TrimSpace(originModel))
	switch {
	case strings.HasSuffix(name, "-4k"):
		return "4K"
	case strings.Contains(name, "flash-lite"):
		return "1K"
	}
	value := strings.ToLower(strings.TrimSpace(quality))
	switch value {
	case "high", "hd", "4k":
		return "4K"
	case "medium", "2k":
		return "2K"
	case "low", "standard", "1k", "1/2k":
		return "1K"
	default:
		return "1K"
	}
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

// manjuBananaAdaptIfNeeded 仅用于 Legacy chat/completions 响应；Image API 主路径不经此函数。
func manjuBananaAdaptIfNeeded(ctx context.Context, info *relaycommon.RelayInfo, responseBody []byte) ([]byte, *types.NewAPIError) {
	if !imagevendor.IsManjuBananaOriginModel(info.OriginModelName) {
		return responseBody, nil
	}
	return AdaptManjuBananaChatCompletionResponse(ctx, info, responseBody)
}
