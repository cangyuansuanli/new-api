package gemini

// Gemini Banana 向上适配（渠道 58/71：New API 系上游 api.0lll0.cn / 157.254.18.68）。
//
// 文生图 / 图生图（含 multipart 参考图）统一走：
//   POST /v1beta/models/gemini-3-pro-image-preview:generateContent
//
// public 名 manju-gemini-banana-*，与 #70 Manju OpenAI 适配（adapt_manju_banana.go）对称，
// 但上游协议为 Gemini generateContent + inlineData，而非 chat/completions。

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/relay/channel/openai"
	"github.com/QuantumNous/new-api/relay/imagevendor"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/gin-gonic/gin"
)

// IsGeminiBananaUpstreamImage 判断是否走 Gemini Banana 向上适配（渠道 58/71）。
func IsGeminiBananaUpstreamImage(info *relaycommon.RelayInfo) bool {
	if info == nil || info.ChannelMeta == nil {
		return false
	}
	if !imagevendor.IsManjuBananaOriginModel(info.OriginModelName) {
		return false
	}
	return model_setting.IsGeminiModelSupportImagine(info.UpstreamModelName)
}

// ConvertGeminiBananaImageRequest 将 OpenAI Image 请求转为上游 generateContent body。
func ConvertGeminiBananaImageRequest(c *gin.Context, request dto.ImageRequest) (*dto.GeminiChatRequest, error) {
	prompt := strings.TrimSpace(request.Prompt)

	parts := make([]dto.GeminiPart, 0, 4)
	if c != nil {
		referenceImages, err := openai.CollectImageEditReferenceDataURIs(c, request)
		if err != nil {
			return nil, err
		}
		for _, dataURI := range referenceImages {
			mimeType, base64Data, err := service.DecodeBase64FileData(dataURI)
			if err != nil {
				return nil, fmt.Errorf("decode reference image: %w", err)
			}
			if _, ok := geminiSupportedMimeTypes[strings.ToLower(mimeType)]; !ok {
				return nil, fmt.Errorf("mime type is not supported by Gemini: %s", mimeType)
			}
			parts = append(parts, dto.GeminiPart{
				InlineData: &dto.GeminiInlineData{
					MimeType: mimeType,
					Data:     base64Data,
				},
			})
		}
	}
	if prompt != "" {
		parts = append(parts, dto.GeminiPart{Text: prompt})
	}
	if len(parts) == 0 {
		return nil, errors.New("prompt or reference image is required")
	}

	geminiRequest := &dto.GeminiChatRequest{
		Contents: []dto.GeminiChatContent{{
			Role:  "user",
			Parts: parts,
		}},
		GenerationConfig: dto.GeminiChatGenerationConfig{
			ResponseModalities: []string{"TEXT", "IMAGE"},
		},
		SafetySettings: buildGeminiSafetySettings(),
	}

	imageConfig := map[string]any{}
	if aspect := resolveGeminiBananaImageAspectRatio(request.Size); aspect != "" {
		imageConfig["aspectRatio"] = aspect
	}
	if imageSize := resolveGeminiBananaImageSize(request.Quality); imageSize != "" {
		imageConfig["imageSize"] = imageSize
	}
	if len(imageConfig) > 0 {
		imageConfigBytes, err := common.Marshal(imageConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal image_config: %w", err)
		}
		geminiRequest.GenerationConfig.ImageConfig = imageConfigBytes
	}
	return geminiRequest, nil
}

func resolveGeminiBananaImageAspectRatio(size string) string {
	value := strings.TrimSpace(size)
	if value == "" || strings.EqualFold(value, "auto") {
		return ""
	}
	if strings.Contains(value, ":") {
		return value
	}
	return ""
}

func resolveGeminiBananaImageSize(quality string) string {
	switch strings.ToLower(strings.TrimSpace(quality)) {
	case "high", "hd", "4k":
		return "4K"
	case "medium", "2k":
		return "2K"
	case "low", "standard", "1k", "auto":
		return "1K"
	default:
		return ""
	}
}
