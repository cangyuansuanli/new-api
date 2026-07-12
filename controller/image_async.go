package controller

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	openai "github.com/QuantumNous/new-api/relay/channel/openai"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relay/image"
	"github.com/QuantumNous/new-api/relay/imagevendor"
	adobevideo "github.com/QuantumNous/new-api/relay/video/adobe"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

func RelayOpenAIImageGenerations(c *gin.Context) {
	if image.IsAsyncRequest(c) {
		RelayImageTaskSubmit(c)
		return
	}
	if shouldRunSyncImageViaQueue(c) {
		relaySyncImageViaQueue(c)
		return
	}
	Relay(c, types.RelayFormatOpenAIImage)
}

func RelayOpenAIImageEdits(c *gin.Context) {
	if image.IsAsyncRequest(c) {
		RelayImageTaskSubmit(c)
		return
	}
	if shouldRunSyncImageViaQueue(c) {
		relaySyncImageViaQueue(c)
		return
	}
	Relay(c, types.RelayFormatOpenAIImage)
}

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
	if openai.IsAsyncChatImageRequest(c) {
		openai.SetChatImageDeprecationHeaders(c)
		c.Set("relay_mode", relayconstant.RelayModeChatCompletions)
		RelayImageTaskSubmit(c)
		return
	}
	if openai.IsLegacyChatImageRequest(c) {
		openai.SetChatImageDeprecationHeaders(c)
	}
	Relay(c, types.RelayFormatOpenAI)
}

func RelayImageTaskFetch(c *gin.Context) {
	relayInfo, err := relaycommon.GenRelayInfo(c, types.RelayFormatTask, nil, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, &dto.TaskError{
			Code:       "gen_relay_info_failed",
			Message:    err.Error(),
			StatusCode: http.StatusInternalServerError,
		})
		return
	}
	if taskErr := image.FetchTask(c, relayInfo.RelayMode); taskErr != nil {
		respondTaskError(c, taskErr)
	}
}

func RelayImageTaskSubmit(c *gin.Context) {
	relayMode := c.GetInt("relay_mode")
	if relayMode == 0 {
		if strings.Contains(c.Request.URL.Path, "/chat/completions") {
			relayMode = relayconstant.RelayModeChatCompletions
		} else if strings.Contains(c.Request.URL.Path, "/edits") {
			relayMode = relayconstant.RelayModeImagesEdits
		} else {
			relayMode = relayconstant.RelayModeImagesGenerations
		}
	}

	var request dto.Request
	var relayFormat types.RelayFormat
	if relayMode == relayconstant.RelayModeChatCompletions {
		textReq, err := helper.GetAndValidateTextRequest(c, relayMode)
		if err != nil {
			respondTaskError(c, service.TaskErrorWrapper(err, "invalid_request", http.StatusBadRequest))
			return
		}
		request = textReq
		relayFormat = types.RelayFormatOpenAI
	} else {
		imgReq, err := helper.GetAndValidateRequest(c, types.RelayFormatOpenAIImage)
		if err != nil {
			respondTaskError(c, service.TaskErrorWrapper(err, "invalid_request", http.StatusBadRequest))
			return
		}
		request = imgReq
		relayFormat = types.RelayFormatOpenAIImage
	}

	relayInfo, err := relaycommon.GenRelayInfo(c, relayFormat, request, nil)
	if err != nil {
		respondTaskError(c, service.TaskErrorWrapper(err, "gen_relay_info_failed", http.StatusInternalServerError))
		return
	}
	relayInfo.RelayMode = relayMode
	if imageRequest, ok := request.(*dto.ImageRequest); ok {
		if err := imagevendor.ValidateFixedResolutionSKU(c, relayInfo.OriginModelName, imageRequest); err != nil {
			respondTaskError(c, service.TaskErrorWrapper(err, "invalid_request", http.StatusBadRequest))
			return
		}
	}
	publicTaskID := model.GenerateTaskID()
	action := constant.TaskActionImageGenerate
	if relayMode == relayconstant.RelayModeImagesEdits {
		action = constant.TaskActionImageEdit
	}
	relayInfo.TaskRelayInfo = &relaycommon.TaskRelayInfo{
		PublicTaskID: publicTaskID,
		Action:       action,
	}

	meta := request.GetTokenCountMeta()
	userId := c.GetInt("id")
	if taskErr := enforceImageTaskAdmission(c, userId); taskErr != nil {
		respondTaskError(c, taskErr)
		return
	}
	needSensitiveCheck := setting.ShouldCheckPromptSensitiveForUser(userId)
	if meta != nil {
		relaycommon.StorePromptInput(c, meta.CombineText)
		if needSensitiveCheck {
			if taskErr := service.TaskErrorIfSensitivePrompt(c, meta.CombineText); taskErr != nil {
				respondTaskError(c, taskErr)
				return
			}
		}
	}

	var taskErr *dto.TaskError
	defer func() {
		if taskErr != nil {
			service.MaybeRefundBilling(c, relayInfo.Billing, taskErr.Message, nil)
		}
	}()

	retryParam := &service.RetryParam{
		Ctx:        c,
		TokenGroup: relayInfo.TokenGroup,
		ModelName:  relayInfo.OriginModelName,
		Retry:      common.GetPointer(0),
	}

	for ; retryParam.GetRetry() <= common.RetryTimes; retryParam.IncreaseRetry() {
		channel, channelErr := getChannel(c, relayInfo, retryParam)
		if channelErr != nil {
			taskErr = service.TaskErrorWrapperLocal(channelErr.Err, "get_channel_failed", http.StatusInternalServerError)
			break
		}
		addUsedChannel(c, channel.Id)
		relayInfo.InitChannelMeta(c)

		// 与视频任务对齐：在创建 task 前应用渠道 model_mapping，便于 task.Properties
		// 与异步结算日志正确记录 is_model_mapped / upstream_model_name。
		originModelName := relayInfo.OriginModelName
		relayInfo.UpstreamModelName = originModelName
		if mapErr := helper.ModelMappedHelper(c, relayInfo, nil); mapErr != nil {
			taskErr = service.TaskErrorWrapperLocal(mapErr, "model_mapping_failed", http.StatusBadRequest)
			break
		}

		priceData, err := helper.ModelPriceHelperPerCall(c, relayInfo)
		if err != nil {
			taskErr = service.TaskErrorWrapper(err, "model_price_error", http.StatusBadRequest)
			break
		}
		relayInfo.PriceData = priceData
		if imageReq, ok := request.(*dto.ImageRequest); ok && imageReq.N != nil {
			relayInfo.PriceData.AddOtherRatio("n", float64(*imageReq.N))
			if !relayInfo.PriceData.UsePrice {
				relayInfo.PriceData.Quota = int(float64(relayInfo.PriceData.Quota) * float64(*imageReq.N))
			}
		}

		if relayInfo.Billing == nil && !relayInfo.PriceData.FreeModel {
			relayInfo.ForcePreConsume = true
			if apiErr := service.PreConsumeBilling(c, relayInfo.PriceData.Quota, relayInfo); apiErr != nil {
				taskErr = service.TaskErrorFromAPIError(apiErr)
				break
			}
		}

		snapshot, requestPath, snapErr := snapshotAsyncImageRequest(c, relayMode, publicTaskID)
		if snapErr != nil {
			taskErr = service.TaskErrorWrapper(snapErr, "snapshot_request_failed", http.StatusBadRequest)
			break
		}

		task := model.InitTask(constant.TaskPlatformImage, relayInfo)
		task.TaskID = publicTaskID
		task.Action = action
		task.Status = model.TaskStatusQueued
		task.Progress = "20%"
		if c.GetBool("image_sync_wait") {
			task.Priority = 100
		}
		task.Quota = relayInfo.PriceData.Quota
		task.Properties.TaskKind = constant.TaskKindImage
		task.Properties.Input = relaycommon.PromptInputFromContext(c)
		task.PrivateData.Key = relayInfo.ApiKey
		task.PrivateData.RequestPath = requestPath
		task.PrivateData.RequestSnapshot = snapshot
		task.PrivateData.BillingSource = relayInfo.BillingSource
		task.PrivateData.SubscriptionId = relayInfo.SubscriptionId
		task.PrivateData.TokenId = relayInfo.TokenId
		task.PrivateData.BillingContext = &model.TaskBillingContext{
			ModelPrice:      relayInfo.PriceData.ModelPrice,
			GroupRatio:      relayInfo.PriceData.GroupRatioInfo.GroupRatio,
			ModelRatio:      relayInfo.PriceData.ModelRatio,
			OtherRatios:     relayInfo.PriceData.OtherRatios,
			OriginModelName: relayInfo.OriginModelName,
			PerCallBilling:  service.ShouldTaskPerCallBilling(relayInfo.OriginModelName, relayInfo.PriceData.UsePrice, relayInfo.PriceData.OtherRatios),
		}

		if insertErr := task.Insert(); insertErr != nil {
			_ = image.CleanupEditSnapshotInputs(snapshot)
			taskErr = service.TaskErrorWrapper(insertErr, "insert_task_failed", http.StatusInternalServerError)
			break
		}

		image.EnqueueTask(task.TaskID)
		job := task.ToOpenAIImageJob(image.JobObjectForPath(requestPath))
		if public := service.ClientFacingModelFromTask(task); public != "" {
			job.Model = public
		}
		c.JSON(http.StatusOK, job)
		return
	}

	if taskErr != nil {
		respondTaskError(c, taskErr)
	}
}

func enforceImageTaskAdmission(c *gin.Context, userID int) *dto.TaskError {
	global, perUser, err := model.CountActiveImageTasks(userID)
	if err != nil {
		return service.TaskErrorWrapper(err, "image_queue_status_failed", http.StatusInternalServerError)
	}
	globalLimit := int64(common.GetEnvOrDefault("IMAGE_ASYNC_MAX_QUEUED_GLOBAL", 2000))
	perUserLimit := int64(common.GetEnvOrDefault("IMAGE_ASYNC_MAX_QUEUED_PER_USER", 200))
	if c.GetBool("image_sync_wait") {
		globalLimit = int64(common.GetEnvOrDefault("IMAGE_SYNC_MAX_BACKLOG", 64))
	}
	if (globalLimit > 0 && global >= globalLimit) || (perUserLimit > 0 && perUser >= perUserLimit) {
		c.Header("Retry-After", "5")
		return service.TaskErrorWrapperLocal(
			fmt.Errorf("image queue is at capacity; retry later"),
			"image_queue_full",
			http.StatusTooManyRequests,
		)
	}
	return nil
}

func snapshotAsyncImageRequest(c *gin.Context, relayMode int, taskID string) ([]byte, string, error) {
	if relayMode == relayconstant.RelayModeImagesEdits {
		body, err := image.SnapshotEditRequest(c, taskID)
		return body, "/v1/images/edits", err
	}
	if relayMode == relayconstant.RelayModeChatCompletions {
		storage, err := common.GetBodyStorage(c)
		if err != nil {
			return nil, "", err
		}
		body, err := storage.Bytes()
		if err != nil {
			return nil, "", err
		}
		return body, "/v1/chat/completions", nil
	}
	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return nil, "", err
	}
	body, err := storage.Bytes()
	if err != nil {
		return nil, "", err
	}
	return body, "/v1/images/generations", nil
}
