package controller

import (
	"net/http"

	openai "github.com/QuantumNous/new-api/relay/channel/openai"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	adobevideo "github.com/QuantumNous/new-api/relay/video/adobe"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

// RelayOpenAIChatCompletions handles POST /v1/chat/completions entry routing.
func RelayOpenAIChatCompletions(c *gin.Context) {
	if adobevideo.IsDeprecatedChatRequest(c) {
		adobevideo.SetDeprecatedChatHeaders(c)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Adobe 视频模型请使用 POST /v1/videos 提交任务，并通过 GET /v1/videos/{task_id} 轮询结果。",
				"type":    "invalid_request_error",
				"code":    "adobe_video_use_videos_api",
			},
		})
		return
	}
	if openai.IsAiclubChatRequest(c) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "该模型是生图模型，不支持 /v1/chat/completions。请使用 POST /v1/images/generations；图生图请使用 POST /v1/images/edits。异步提交后请使用返回的 task id 查询结果。",
				"type":    "invalid_request_error",
				"code":    "image_model_use_images_api",
			},
		})
		return
	}
	if openai.IsAsyncChatImageRequest(c) {
		openai.SetChatImageDeprecationHeaders(c)
		c.Set("relay_mode", relayconstant.RelayModeChatCompletions)
		RelayImageTaskSubmit(c)
		return
	}
	if openai.IsLegacyChatImageRequest(c) {
		openai.SetChatImageDeprecationHeaders(c)
	}
	if openai.IsLegacyChatAudioRequest(c) {
		openai.SetChatAudioDeprecationHeaders(c)
	}
	Relay(c, types.RelayFormatOpenAI)
}
