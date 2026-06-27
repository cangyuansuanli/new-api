package controller

import (
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relay"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

func RelayOpenAIImageGenerations(c *gin.Context) {
	if relay.IsAsyncImageRequest(c) {
		RelayImageTaskSubmit(c)
		return
	}
	Relay(c, types.RelayFormatOpenAIImage)
}

func RelayOpenAIImageEdits(c *gin.Context) {
	if relay.IsAsyncImageRequest(c) {
		RelayImageTaskSubmit(c)
		return
	}
	Relay(c, types.RelayFormatOpenAIImage)
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
	if taskErr := relay.RelayImageTaskFetch(c, relayInfo.RelayMode); taskErr != nil {
		respondTaskError(c, taskErr)
	}
}

func RelayImageTaskSubmit(c *gin.Context) {
	relayMode := c.GetInt("relay_mode")
	if relayMode == 0 {
		if strings.Contains(c.Request.URL.Path, "/edits") {
			relayMode = relayconstant.RelayModeImagesEdits
		} else {
			relayMode = relayconstant.RelayModeImagesGenerations
		}
	}

	request, err := helper.GetAndValidateRequest(c, types.RelayFormatOpenAIImage)
	if err != nil {
		respondTaskError(c, service.TaskErrorWrapper(err, "invalid_request", http.StatusBadRequest))
		return
	}

	relayInfo, err := relaycommon.GenRelayInfo(c, types.RelayFormatOpenAIImage, request, nil)
	if err != nil {
		respondTaskError(c, service.TaskErrorWrapper(err, "gen_relay_info_failed", http.StatusInternalServerError))
		return
	}
	relayInfo.RelayMode = relayMode
	publicTaskID := model.GenerateTaskID()
	action := constant.TaskActionImageGenerate
	if relayMode == relayconstant.RelayModeImagesEdits {
		action = constant.TaskActionImageEdit
	}
	relayInfo.TaskRelayInfo = &relaycommon.TaskRelayInfo{
		PublicTaskID: publicTaskID,
		Action:       action,
	}

	var taskErr *dto.TaskError
	defer func() {
		if taskErr != nil && relayInfo.Billing != nil {
			relayInfo.Billing.Refund(c)
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

		snapshot, requestPath, snapErr := snapshotAsyncImageRequest(c, relayMode)
		if snapErr != nil {
			taskErr = service.TaskErrorWrapper(snapErr, "snapshot_request_failed", http.StatusBadRequest)
			break
		}

		task := model.InitTask(constant.TaskPlatformImage, relayInfo)
		task.TaskID = publicTaskID
		task.Action = action
		task.Status = model.TaskStatusSubmitted
		task.Progress = "10%"
		task.Quota = relayInfo.PriceData.Quota
		task.Properties.TaskKind = constant.TaskKindImage
		if meta := request.GetTokenCountMeta(); meta != nil {
			task.Properties.Input = strings.TrimSpace(meta.CombineText)
		}
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
			taskErr = service.TaskErrorWrapper(insertErr, "insert_task_failed", http.StatusInternalServerError)
			break
		}

		relay.EnqueueImageAsyncTask(task.TaskID)
		job := task.ToOpenAIImageJob(relay.ImageJobObjectForPathExported(requestPath))
		c.JSON(http.StatusOK, job)
		return
	}

	if taskErr != nil {
		respondTaskError(c, taskErr)
	}
}

func snapshotAsyncImageRequest(c *gin.Context, relayMode int) ([]byte, string, error) {
	if relayMode == relayconstant.RelayModeImagesEdits {
		body, err := relay.SnapshotAsyncImageEditRequest(c)
		return body, "/v1/images/edits", err
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
