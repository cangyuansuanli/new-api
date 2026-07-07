package gemini

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

func imageDataFromGeminiGenerateContent(response *dto.GeminiChatResponse) ([]dto.ImageData, error) {
	if response == nil {
		return nil, errors.New("empty response from Gemini API")
	}
	images := make([]dto.ImageData, 0, 2)
	for _, candidate := range response.Candidates {
		for _, part := range candidate.Content.Parts {
			if part.InlineData == nil || !strings.HasPrefix(part.InlineData.MimeType, "image") {
				continue
			}
			images = append(images, dto.ImageData{B64Json: part.InlineData.Data})
		}
	}
	if len(images) == 0 {
		return nil, errors.New("generateContent response has no image data")
	}
	return images, nil
}

// GeminiGenerateContentImageHandler 将 generateContent 出图响应转为 OpenAI ImageResponse。
func GeminiGenerateContentImageHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	service.CloseResponseBodyGracefully(resp)

	var geminiResponse dto.GeminiChatResponse
	if err := common.Unmarshal(responseBody, &geminiResponse); err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	if len(geminiResponse.Candidates) == 0 {
		if geminiResponse.PromptFeedback != nil && geminiResponse.PromptFeedback.BlockReason != nil {
			return nil, types.NewOpenAIError(
				fmt.Errorf("request blocked by Gemini API: %s", *geminiResponse.PromptFeedback.BlockReason),
				types.ErrorCodePromptBlocked,
				http.StatusBadRequest,
			)
		}
		return nil, types.NewOpenAIError(errors.New("empty response from Gemini API"), types.ErrorCodeEmptyResponse, http.StatusInternalServerError)
	}

	images, err := imageDataFromGeminiGenerateContent(&geminiResponse)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponse, http.StatusBadGateway)
	}

	rehosted, err := service.RehostImageDataForClient(c.Request.Context(), c.GetInt("id"), "", info.ChannelBaseUrl, info.OriginModelName, images, info.ImageClientWantsURL)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponse, http.StatusBadGateway)
	}

	imageResp := dto.ImageResponse{
		Created: common.GetTimestamp(),
		Data:    rehosted,
	}
	out, err := common.Marshal(imageResp)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	service.IOCopyBytesGracefully(c, resp, out)

	usage := buildUsageFromGeminiMetadata(geminiResponse.UsageMetadata, info.GetEstimatePromptTokens())
	return &usage, nil
}
