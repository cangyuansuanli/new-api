package audio

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/helper"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

// ExecuteTaskUpstream replays a queued audio task against upstream chat/completions.
func ExecuteTaskUpstream(ctx context.Context, task *model.Task) (string, error) {
	channel, err := model.GetChannelById(task.ChannelId, true)
	if err != nil {
		return "", err
	}
	cache, err := model.GetUserCache(task.UserId)
	if err != nil {
		return "", err
	}

	req, err := buildHTTPRequestForAudioTask(ctx, task)
	if err != nil {
		return "", err
	}
	req = req.WithContext(ctx)
	defer req.Body.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	defer cleanupTaskRequestContext(c)
	cache.WriteContext(c)
	c.Set("id", task.UserId)

	group := task.Group
	if group == "" {
		group, _ = model.GetUserGroup(task.UserId, false)
	}
	c.Set("group", group)

	if apiErr := setupAudioTaskChannelContext(c, channel, task.Properties.OriginModelName, task.PrivateData.Key); apiErr != nil {
		return "", apiErr.Err
	}
	c.Set("relay_mode", relayconstant.RelayModeAudioGenerations)

	request, err := helper.GetAndValidAudioGenerationRequest(c)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(request.ResponseFormat) == "" {
		request.ResponseFormat = "url"
	}
	request.Stream = common.GetPointer(false)

	relayInfo, err := relaycommon.GenRelayInfo(c, types.RelayFormatOpenAIAudioGeneration, request, nil)
	if err != nil {
		return "", err
	}
	relayInfo.InitChannelMeta(c)
	if relayInfo.TaskRelayInfo == nil {
		relayInfo.TaskRelayInfo = &relaycommon.TaskRelayInfo{}
	}
	relayInfo.TaskRelayInfo.PublicTaskID = task.TaskID
	relayInfo.IsStream = false
	relayInfo.SkipConsumeQuota = true

	if apiErr := Helper(c, relayInfo); apiErr != nil {
		return "", apiErr.Err
	}

	return parseCapturedAudioURL(w)
}

func parseCapturedAudioURL(w *httptest.ResponseRecorder) (string, error) {
	if w.Code != http.StatusOK {
		return "", fmt.Errorf("upstream audio generation failed with status %d", w.Code)
	}
	var resp dto.AudioGenerationResponse
	if err := common.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		return "", err
	}
	if len(resp.Data) == 0 || strings.TrimSpace(resp.Data[0].URL) == "" {
		return "", fmt.Errorf("audio response has no url")
	}
	return strings.TrimSpace(resp.Data[0].URL), nil
}

func cleanupTaskRequestContext(c *gin.Context) {
	common.CleanupBodyStorage(c)
}

func setupAudioTaskChannelContext(c *gin.Context, channel *model.Channel, modelName, keyOverride string) *types.NewAPIError {
	if channel == nil {
		return types.NewError(fmt.Errorf("channel is nil"), types.ErrorCodeGetChannelFailed, types.ErrOptionWithSkipRetry())
	}
	c.Set("original_model", modelName)
	common.SetContextKey(c, constant.ContextKeyChannelId, channel.Id)
	common.SetContextKey(c, constant.ContextKeyChannelName, channel.Name)
	common.SetContextKey(c, constant.ContextKeyChannelType, channel.Type)
	common.SetContextKey(c, constant.ContextKeyChannelCreateTime, channel.CreatedTime)
	common.SetContextKey(c, constant.ContextKeyChannelSetting, channel.GetSetting())
	common.SetContextKey(c, constant.ContextKeyChannelOtherSetting, channel.GetOtherSettings())
	common.SetContextKey(c, constant.ContextKeyChannelParamOverride, channel.GetParamOverride())
	common.SetContextKey(c, constant.ContextKeyChannelHeaderOverride, channel.GetHeaderOverride())
	if channel.OpenAIOrganization != nil && *channel.OpenAIOrganization != "" {
		common.SetContextKey(c, constant.ContextKeyChannelOrganization, *channel.OpenAIOrganization)
	}
	common.SetContextKey(c, constant.ContextKeyChannelAutoBan, channel.GetAutoBan())
	common.SetContextKey(c, constant.ContextKeyChannelModelMapping, channel.GetModelMapping())
	common.SetContextKey(c, constant.ContextKeyChannelStatusCodeMapping, channel.GetStatusCodeMapping())
	key := strings.TrimSpace(keyOverride)
	index := 0
	var newAPIError *types.NewAPIError
	if key == "" {
		key, index, newAPIError = channel.GetNextEnabledKey()
		if newAPIError != nil {
			return newAPIError
		}
	}
	if channel.ChannelInfo.IsMultiKey && keyOverride == "" {
		common.SetContextKey(c, constant.ContextKeyChannelIsMultiKey, true)
		common.SetContextKey(c, constant.ContextKeyChannelMultiKeyIndex, index)
	} else {
		common.SetContextKey(c, constant.ContextKeyChannelIsMultiKey, false)
	}
	common.SetContextKey(c, constant.ContextKeyChannelKey, key)
	common.SetContextKey(c, constant.ContextKeyChannelBaseUrl, channel.GetBaseURL())
	common.SetContextKey(c, constant.ContextKeySystemPromptOverride, false)
	common.SetContextKey(c, constant.ContextKeyOriginalModel, modelName)
	return nil
}
