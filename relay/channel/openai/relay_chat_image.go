package openai

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relay/imagevendor"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

var (
	chatImageMarkdownDataImageRE = regexp.MustCompile(`!\[[^\]]*\]\((data:image/[^;)]+;base64,([^)]+))\)`)
	chatImageMarkdownHTTPImageRE = regexp.MustCompile(`!\[[^\]]*\]\((https?://[^)]+)\)`)
)

// IsChatImageModel：Flash Image 等走 chat upstream、下游 Image API 的出图模型。
// Manju Gemini Banana 图生图见 ManjuBananaUsesChatCompletionsUpstream。
func IsChatImageModel(model string) bool {
	if imagevendor.IsManjuBananaOriginModel(model) {
		return false
	}
	name := strings.ToLower(strings.TrimSpace(model))
	if strings.Contains(name, "banana") {
		return true
	}
	if strings.Contains(name, "flash-image") {
		return true
	}
	if strings.Contains(name, "flash-lite-image") {
		return true
	}
	return false
}

func chatImageGetRequestURL(info *relaycommon.RelayInfo) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(info.ChannelBaseUrl), "/")
	if base == "" {
		return "/v1/chat/completions", nil
	}
	return base + "/v1/chat/completions", nil
}

func resolveChatImageUpstreamModel(info *relaycommon.RelayInfo, request dto.ImageRequest) string {
	if info != nil && info.ChannelMeta != nil && strings.TrimSpace(info.UpstreamModelName) != "" {
		return strings.TrimSpace(info.UpstreamModelName)
	}
	if strings.TrimSpace(request.Model) != "" {
		return strings.TrimSpace(request.Model)
	}
	if info != nil && strings.TrimSpace(info.OriginModelName) != "" {
		return strings.TrimSpace(info.OriginModelName)
	}
	return ""
}

// ConvertImageRequestForChatImage 将 Image API 请求转为 upstream chat/completions body。
func ConvertImageRequestForChatImage(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (any, error) {
	messages, err := buildChatImageMessages(c, info, request)
	if err != nil {
		return nil, err
	}
	extraBody, err := buildChatImageExtraBody(request)
	if err != nil {
		return nil, err
	}

	modelName := resolveChatImageUpstreamModel(info, request)
	if modelName == "" {
		return nil, fmt.Errorf("model is required")
	}

	chatReq := dto.GeneralOpenAIRequest{
		Model:    modelName,
		Messages: messages,
		Stream:   common.GetPointer(false),
	}
	if extraBody != nil {
		raw, err := common.Marshal(extraBody)
		if err != nil {
			return nil, err
		}
		chatReq.ExtraBody = raw
	}
	return chatReq, nil
}

func buildChatImageExtraBody(request dto.ImageRequest) (map[string]any, error) {
	imageConfig := map[string]string{}
	if aspect := resolveChatImageAspectRatio(request.Size); aspect != "" {
		imageConfig["aspect_ratio"] = aspect
	}
	if imageSize := resolveChatImageSize(request.Quality); imageSize != "" {
		imageConfig["image_size"] = imageSize
	}
	if len(imageConfig) == 0 && len(request.Extra) == 0 {
		return nil, nil
	}
	extra := map[string]any{}
	if raw, ok := request.Extra["extra_body"]; ok && len(raw) > 0 {
		if err := common.Unmarshal(raw, &extra); err != nil {
			return nil, err
		}
	}
	if len(imageConfig) > 0 {
		google, _ := extra["google"].(map[string]any)
		if google == nil {
			google = map[string]any{}
		}
		cfg, _ := google["image_config"].(map[string]any)
		if cfg == nil {
			cfg = map[string]any{}
		}
		for k, v := range imageConfig {
			if _, exists := cfg[k]; !exists {
				cfg[k] = v
			}
		}
		google["image_config"] = cfg
		extra["google"] = google
	}
	if len(extra) == 0 {
		return nil, nil
	}
	return extra, nil
}

func resolveChatImageAspectRatio(size string) string {
	value := strings.TrimSpace(size)
	if value == "" || strings.EqualFold(value, "auto") {
		return ""
	}
	if strings.Contains(value, ":") {
		return value
	}
	return ""
}

func resolveChatImageSize(quality string) string {
	value := strings.ToLower(strings.TrimSpace(quality))
	switch value {
	case "high", "hd", "4k":
		return "4K"
	case "medium", "2k":
		return "2K"
	case "low", "standard", "1k":
		return "1K"
	default:
		return ""
	}
}

func buildChatImageMessages(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) ([]dto.Message, error) {
	prompt := strings.TrimSpace(request.Prompt)
	hasRefs := hasChatImageReferenceInput(c, info, request)
	if !hasRefs {
		if prompt == "" {
			return nil, fmt.Errorf("prompt is required")
		}
		return []dto.Message{{Role: "user", Content: prompt}}, nil
	}

	contentParts := make([]dto.MediaContent, 0, 8)
	if prompt != "" {
		contentParts = append(contentParts, dto.MediaContent{Type: "text", Text: prompt})
	}

	refURLs, err := collectChatImageReferenceURLs(c, request)
	if err != nil {
		return nil, err
	}
	for _, refURL := range refURLs {
		contentParts = append(contentParts, dto.MediaContent{
			Type: "image_url",
			ImageUrl: &dto.MessageImageUrl{
				Url: refURL,
			},
		})
	}

	if len(contentParts) == 0 {
		return nil, fmt.Errorf("prompt or reference image is required")
	}
	return []dto.Message{{Role: "user", Content: contentParts}}, nil
}

func hasChatImageReferenceInput(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) bool {
	if len(parseJSONStringList(request.Image)) > 0 || len(parseJSONStringList(request.Images)) > 0 {
		return true
	}
	if info != nil && info.RelayMode == relayconstant.RelayModeImagesEdits && c != nil && c.Request != nil {
		if err := ensureMultipartFormParsed(c); err == nil && c.Request.MultipartForm != nil {
			if files, ok := c.Request.MultipartForm.File["image"]; ok && len(files) > 0 {
				return true
			}
			if files, ok := c.Request.MultipartForm.File["image[]"]; ok && len(files) > 0 {
				return true
			}
			for fieldName, files := range c.Request.MultipartForm.File {
				if strings.HasPrefix(fieldName, "image[") && len(files) > 0 {
					return true
				}
			}
			if maskFiles, ok := c.Request.MultipartForm.File["mask"]; ok && len(maskFiles) > 0 {
				return true
			}
		}
	}
	return len(parseJSONStringList(request.Mask)) > 0
}

func ensureMultipartFormParsed(c *gin.Context) error {
	if c == nil || c.Request == nil || c.Request.MultipartForm != nil {
		return nil
	}
	if !strings.Contains(c.Request.Header.Get("Content-Type"), "multipart/form-data") {
		return nil
	}
	form, err := common.ParseMultipartFormReusable(c)
	if err != nil {
		return err
	}
	c.Request.MultipartForm = form
	c.Request.PostForm = form.Value
	return nil
}

func collectChatImageReferenceURLs(c *gin.Context, request dto.ImageRequest) ([]string, error) {
	var urls []string
	urls = append(urls, parseJSONStringList(request.Image)...)
	urls = append(urls, parseJSONStringList(request.Images)...)

	if c != nil && c.Request != nil {
		if err := ensureMultipartFormParsed(c); err != nil {
			return nil, err
		}
		if c.Request.MultipartForm != nil {
			for _, key := range []string{"image", "image[]"} {
				for _, fh := range c.Request.MultipartForm.File[key] {
					dataURI, err := multipartFileToDataURI(fh)
					if err != nil {
						return nil, err
					}
					urls = append(urls, dataURI)
				}
			}
			for _, key := range []string{"image[0]", "image[1]", "image[2]", "image[3]"} {
				for _, fh := range c.Request.MultipartForm.File[key] {
					dataURI, err := multipartFileToDataURI(fh)
					if err != nil {
						return nil, err
					}
					urls = append(urls, dataURI)
				}
			}
			if maskFiles, ok := c.Request.MultipartForm.File["mask"]; ok {
				for _, fh := range maskFiles {
					dataURI, err := multipartFileToDataURI(fh)
					if err != nil {
						return nil, err
					}
					urls = append(urls, dataURI)
				}
			}
		}
	}

	maskURLs := parseJSONStringList(request.Mask)
	urls = append(urls, maskURLs...)
	return urls, nil
}

func parseJSONStringList(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var single string
	if err := common.Unmarshal(raw, &single); err == nil && strings.TrimSpace(single) != "" {
		return []string{strings.TrimSpace(single)}
	}
	var list []string
	if err := common.Unmarshal(raw, &list); err == nil {
		out := make([]string, 0, len(list))
		for _, item := range list {
			if v := strings.TrimSpace(item); v != "" {
				out = append(out, v)
			}
		}
		return out
	}
	return nil
}

func multipartFileToDataURI(fh *multipart.FileHeader) (string, error) {
	file, err := fh.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 20<<20))
	if err != nil {
		return "", err
	}
	mimeType := detectImageMimeType(fh.Filename)
	if ct := fh.Header.Get("Content-Type"); strings.HasPrefix(ct, "image/") {
		mimeType = ct
	}
	return fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(data)), nil
}

// OpenaiChatImageHandler 将 upstream chat 出图响应转为 OpenAI ImageResponse。
func OpenaiChatImageHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	defer service.CloseResponseBodyGracefully(resp)

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}

	var simpleResponse dto.OpenAITextResponse
	if err := common.Unmarshal(responseBody, &simpleResponse); err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	if oaiError := simpleResponse.GetOpenAIError(); oaiError != nil && oaiError.Type != "" {
		return nil, types.WithOpenAIError(*oaiError, resp.StatusCode)
	}

	adaptedBody, adaptErr := manjuBananaAdaptIfNeeded(c.Request.Context(), info, responseBody)
	if adaptErr != nil {
		return nil, adaptErr
	}
	responseBody = adaptedBody

	images, err := imageDataFromChatImageContent(extractChatImageMessageContent(responseBody))
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponse, http.StatusBadGateway)
	}

	rehosted, err := service.RehostImageDataForClient(c.Request.Context(), c.GetInt("id"), "", info.ChannelBaseUrl, info.OriginModelName, images, info.ImageClientWantsURL)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponse, http.StatusBadGateway)
	}

	imageResp := dto.ImageResponse{
		Created: time.Now().Unix(),
		Data:    rehosted,
	}
	out, err := common.Marshal(imageResp)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	service.IOCopyBytesGracefully(c, resp, out)

	usage := simpleResponse.Usage
	if usage.TotalTokens == 0 {
		usage = dto.Usage{TotalTokens: 1, PromptTokens: 1}
	}
	normalizeOpenAIUsage(&usage)
	applyUsagePostProcessing(info, &usage, responseBody)
	return &usage, nil
}

func extractChatImageMessageContent(body []byte) string {
	var payload struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := common.Unmarshal(body, &payload); err != nil {
		return ""
	}
	if len(payload.Choices) == 0 {
		return ""
	}
	return payload.Choices[0].Message.Content
}

func imageDataFromChatImageContent(content string) ([]dto.ImageData, error) {
	if match := chatImageMarkdownDataImageRE.FindStringSubmatch(content); len(match) > 2 {
		return []dto.ImageData{{B64Json: match[2]}}, nil
	}
	if match := chatImageMarkdownHTTPImageRE.FindStringSubmatch(content); len(match) > 1 {
		return []dto.ImageData{{Url: strings.TrimSpace(match[1])}}, nil
	}
	trimmed := strings.TrimSpace(content)
	if strings.HasPrefix(trimmed, "data:image/") {
		comma := strings.Index(trimmed, ",")
		if comma > 0 {
			return []dto.ImageData{{B64Json: trimmed[comma+1:]}}, nil
		}
	}
	return nil, fmt.Errorf("chat image response has no image markdown")
}

// IsLegacyChatImageRequest 兼容期：POST /chat/completions 的 chat 出图请求（含 sync/async）。
func IsLegacyChatImageRequest(c *gin.Context) bool {
	if c == nil || c.Request == nil || c.Request.Method != http.MethodPost {
		return false
	}
	if !strings.HasSuffix(c.Request.URL.Path, "/chat/completions") {
		return false
	}
	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return false
	}
	body, err := storage.Bytes()
	if err != nil || len(body) == 0 {
		return false
	}
	var probe struct {
		Model string `json:"model"`
	}
	if err := common.Unmarshal(body, &probe); err != nil {
		return false
	}
	return IsChatImageModel(probe.Model)
}

// IsAsyncChatImageRequest 兼容期：POST /chat/completions + async 的 chat 出图请求。
func IsAsyncChatImageRequest(c *gin.Context) bool {
	if c == nil || c.Request == nil || c.Request.Method != http.MethodPost {
		return false
	}
	if !strings.HasSuffix(c.Request.URL.Path, "/chat/completions") {
		return false
	}
	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return false
	}
	body, err := storage.Bytes()
	if err != nil || len(body) == 0 {
		return false
	}
	var probe struct {
		Async *bool  `json:"async"`
		Model string `json:"model"`
	}
	if err := common.Unmarshal(body, &probe); err != nil {
		return false
	}
	return probe.Async != nil && *probe.Async && IsChatImageModel(probe.Model)
}

// NormalizeAsyncLegacyChatImageBody 兼容期：旧 async task 快照中的 chat body。
func NormalizeAsyncLegacyChatImageBody(body []byte) ([]byte, error) {
	var raw map[string]json.RawMessage
	if err := common.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	delete(raw, "async")
	raw["stream"] = json.RawMessage("false")
	return common.Marshal(raw)
}

// ParseLegacyChatImageResponse 解析旧 chat/completions 出图响应。
func ParseLegacyChatImageResponse(body []byte) ([]dto.ImageData, *dto.Usage, error) {
	if len(body) == 0 {
		return nil, nil, fmt.Errorf("empty chat image response")
	}
	var payload struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage dto.Usage `json:"usage"`
	}
	if err := common.Unmarshal(body, &payload); err != nil {
		return nil, nil, fmt.Errorf("parse chat image json: %w", err)
	}
	images, err := imageDataFromChatImageContent(extractChatImageMessageContent(body))
	if err != nil {
		return nil, nil, err
	}
	usage := payload.Usage
	if usage.TotalTokens == 0 {
		usage = dto.Usage{TotalTokens: 1, PromptTokens: 1}
	}
	return images, &usage, nil
}

// SetChatImageDeprecationHeaders 标记 chat 出图兼容路径。
func SetChatImageDeprecationHeaders(c *gin.Context) {
	if c == nil {
		return
	}
	c.Header("Deprecation", "true")
	c.Header("Link", `</v1/images/generations>; rel="successor-version"`)
}
