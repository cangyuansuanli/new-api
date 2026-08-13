package controller

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relay/audio"
	audiovendor "github.com/QuantumNous/new-api/relay/audiovendor"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

func RelayOpenAIAudioGenerations(c *gin.Context) {
	if audio.IsAsyncRequest(c) {
		RelayAudioTaskSubmit(c)
		return
	}
	Relay(c, types.RelayFormatOpenAIAudioGeneration)
}

func RelayAudioTaskFetch(c *gin.Context) {
	relayInfo, err := relaycommon.GenRelayInfo(c, types.RelayFormatTask, nil, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, &dto.TaskError{
			Code:       "gen_relay_info_failed",
			Message:    err.Error(),
			StatusCode: http.StatusInternalServerError,
		})
		return
	}
	if taskErr := audio.FetchTask(c, relayInfo.RelayMode); taskErr != nil {
		respondTaskError(c, taskErr)
	}
}

func RelayAudioTaskSubmit(c *gin.Context) {
	request, err := helper.GetAndValidAudioGenerationRequest(c)
	if err != nil {
		respondTaskError(c, service.TaskErrorWrapper(err, "invalid_request", http.StatusBadRequest))
		return
	}

	relayInfo, err := relaycommon.GenRelayInfo(c, types.RelayFormatOpenAIAudioGeneration, request, nil)
	if err != nil {
		respondTaskError(c, service.TaskErrorWrapper(err, "gen_relay_info_failed", http.StatusInternalServerError))
		return
	}
	relayInfo.RelayMode = relayconstant.RelayModeAudioGenerations
	relayInfo.InitChannelMeta(c)
	if err := audiovendor.ValidateRequest(c, relayInfo, request); err != nil {
		respondTaskError(c, service.TaskErrorWrapper(err, "invalid_request", http.StatusBadRequest))
		return
	}

	publicTaskID := model.GenerateTaskID()
	action := constant.TaskActionAudioGenerate
	relayInfo.TaskRelayInfo = &relaycommon.TaskRelayInfo{
		PublicTaskID: publicTaskID,
		Action:       action,
	}

	meta := request.GetTokenCountMeta()
	userId := c.GetInt("id")
	if meta != nil {
		relaycommon.StorePromptInput(c, meta.CombineText)
		if setting.ShouldCheckPromptSensitiveForUser(userId, setting.SensitivePromptScopeAudio) {
			if taskErr := service.TaskErrorIfSensitivePrompt(c, meta.CombineText, setting.SensitivePromptScopeAudio); taskErr != nil {
				respondTaskError(c, taskErr)
				return
			}
		}
	}

	var taskErr *dto.TaskError
	defer func() {
		if taskErr != nil {
			service.MaybeRefundBilling(c, action, relayInfo.Billing, taskErr.Message, nil)
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

		originModelName := relayInfo.OriginModelName
		relayInfo.UpstreamModelName = originModelName
		if mapErr := helper.ModelMappedHelper(c, relayInfo, nil); mapErr != nil {
			taskErr = service.TaskErrorWrapperLocal(mapErr, "model_mapping_failed", http.StatusBadRequest)
			break
		}

		if admissionErr := enforceAudioTaskAdmission(c, userId); admissionErr != nil {
			taskErr = admissionErr
			break
		}

		priceData, err := helper.ModelPriceHelperPerCall(c, relayInfo)
		if err != nil {
			taskErr = service.TaskErrorWrapper(err, "model_price_error", http.StatusBadRequest)
			break
		}
		relayInfo.PriceData = priceData

		if relayInfo.Billing == nil && !relayInfo.PriceData.FreeModel {
			relayInfo.ForcePreConsume = true
			if apiErr := service.PreConsumeBilling(c, relayInfo.PriceData.Quota, relayInfo); apiErr != nil {
				taskErr = service.TaskErrorFromAPIError(apiErr)
				break
			}
		}

		snapshot, requestPath, snapErr := snapshotAsyncAudioRequest(c)
		if snapErr != nil {
			taskErr = service.TaskErrorWrapper(snapErr, "snapshot_request_failed", http.StatusBadRequest)
			break
		}

		task := model.InitTask(constant.TaskPlatformAudio, relayInfo)
		task.TaskID = publicTaskID
		task.Action = action
		task.Status = model.TaskStatusQueued
		task.Progress = "20%"
		task.Quota = relayInfo.PriceData.Quota
		task.Properties.TaskKind = constant.TaskKindAudio
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

		globalLimit, perUserLimit := audioTaskAdmissionLimits()
		if insertErr := model.InsertAudioTaskWithAdmission(task, globalLimit, perUserLimit); insertErr != nil {
			if errors.Is(insertErr, model.ErrAudioTaskQueueFull) {
				c.Header("Retry-After", "5")
				taskErr = service.TaskErrorWrapperLocal(insertErr, "audio_queue_full", http.StatusTooManyRequests)
				break
			}
			taskErr = service.TaskErrorWrapper(insertErr, "insert_task_failed", http.StatusInternalServerError)
			break
		}

		audio.EnqueueTask(task.TaskID)
		job := task.ToOpenAIAudioJob(audio.JobObjectForPath(requestPath))
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

func enforceAudioTaskAdmission(c *gin.Context, userID int) *dto.TaskError {
	global, perUser, err := model.CountActiveAudioTasks(userID)
	if err != nil {
		return service.TaskErrorWrapper(err, "audio_queue_status_failed", http.StatusInternalServerError)
	}
	globalLimit, perUserLimit := audioTaskAdmissionLimits()
	if (globalLimit > 0 && global >= globalLimit) || (perUserLimit > 0 && perUser >= perUserLimit) {
		c.Header("Retry-After", "5")
		return service.TaskErrorWrapperLocal(
			fmt.Errorf("audio queue is at capacity; retry later"),
			"audio_queue_full",
			http.StatusTooManyRequests,
		)
	}
	return nil
}

func audioTaskAdmissionLimits() (globalLimit, perUserLimit int64) {
	return int64(common.GetEnvOrDefault("AUDIO_ASYNC_MAX_QUEUED_GLOBAL", 500)),
		int64(common.GetEnvOrDefault("AUDIO_ASYNC_MAX_QUEUED_PER_USER", 50))
}

func snapshotAsyncAudioRequest(c *gin.Context) ([]byte, string, error) {
	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return nil, "", err
	}
	body, err := storage.Bytes()
	if err != nil {
		return nil, "", err
	}
	snapshot, err := audio.NewJSONRequestSnapshot("/v1/audio/generations", body)
	return snapshot, "/v1/audio/generations", err
}
