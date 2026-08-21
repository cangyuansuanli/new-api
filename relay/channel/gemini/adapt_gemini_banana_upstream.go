package gemini

// cy-yf-gemini-banana-* 向上适配：OpenAI Image 请求转为 generateContent body。
//
// 文生图 / 图生图统一走：
//   POST /v1beta/models/{upstream}:generateContent

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/relay/channel/openai"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/imagevendor"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/gin-gonic/gin"
)

// IsYunfeiGeminiBananaUpstreamImage 判断 cy-yf-gemini-banana-* 是否走 generateContent 出图。
func IsYunfeiGeminiBananaUpstreamImage(info *relaycommon.RelayInfo) bool {
	if info == nil || info.ChannelMeta == nil {
		return false
	}
	if !imagevendor.IsYunfeiBananaOriginModel(info.OriginModelName) {
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
	return imagevendor.CanonicalAspectRatio(size)
}

func resolveGeminiBananaImageSize(quality string) string {
	return imagevendor.CanonicalResolution(quality)
}
