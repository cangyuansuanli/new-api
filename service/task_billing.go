package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

// IsPerRequestTaskBilling reports models billed once per generation regardless of duration.
func IsPerRequestTaskBilling(modelName string) bool {
	if common.StringsContains(constant.TaskPricePatches, modelName) {
		return true
	}
	return billing_setting.GetBillingMode(modelName) == billing_setting.BillingModePerRequest
}

// ShouldApplyTaskOtherRatio decides whether a billing ratio key should affect pre-charge quota.
func ShouldApplyTaskOtherRatio(modelName, ratioKey string) bool {
	if ratioKey == "seconds" && IsPerRequestTaskBilling(modelName) {
		return false
	}
	return true
}

// ShouldTaskPerCallBilling 判断任务是否按固定次价计费（完成后不再按 usage.seconds 差额结算）。
// 配置了 ModelPrice 且带 seconds 倍率的视频模型视为按秒单价，完成后再结算。
func ShouldTaskPerCallBilling(modelName string, usePrice bool, otherRatios map[string]float64) bool {
	if IsPerRequestTaskBilling(modelName) {
		return true
	}
	if !usePrice {
		return false
	}
	if seconds, ok := otherRatios["seconds"]; ok && seconds > 0 {
		return false
	}
	return true
}

// LogTaskConsumption 记录任务消费日志和统计信息（仅记录，不涉及实际扣费）。
// 实际扣费已由 BillingSession（PreConsumeBilling + SettleBilling）完成。
func LogTaskConsumption(c *gin.Context, info *relaycommon.RelayInfo) {
	tokenName := c.GetString("token_name")
	logContent := fmt.Sprintf("操作 %s", info.Action)
	// 支持任务仅按次计费
	if common.StringsContains(constant.TaskPricePatches, info.OriginModelName) {
		logContent = fmt.Sprintf("%s，按次计费", logContent)
	} else {
		if len(info.PriceData.OtherRatios) > 0 {
			var contents []string
			for key, ra := range info.PriceData.OtherRatios {
				if 1.0 != ra {
					contents = append(contents, fmt.Sprintf("%s: %.2f", key, ra))
				}
			}
			if len(contents) > 0 {
				logContent = fmt.Sprintf("%s, 计算参数：%s", logContent, strings.Join(contents, ", "))
			}
		}
	}
	other := make(map[string]interface{})
	other["is_task"] = true
	other["request_path"] = c.Request.URL.Path
	other["model_price"] = info.PriceData.ModelPrice
	if info.PriceData.ModelRatio > 0 {
		other["model_ratio"] = info.PriceData.ModelRatio
	}
	other["group_ratio"] = info.PriceData.GroupRatioInfo.GroupRatio
	if info.PriceData.GroupRatioInfo.HasSpecialRatio {
		other["user_group_ratio"] = info.PriceData.GroupRatioInfo.GroupSpecialRatio
	}
	if info.IsModelMapped {
		other["is_model_mapped"] = true
		other["upstream_model_name"] = info.UpstreamModelName
	}
	model.RecordConsumeLog(c, info.UserId, model.RecordConsumeLogParams{
		ChannelId: info.ChannelId,
		ModelName: info.OriginModelName,
		TokenName: tokenName,
		Quota:     info.PriceData.Quota,
		Content:   logContent,
		TokenId:   info.TokenId,
		Group:     info.UsingGroup,
		Other:     other,
	})
	model.UpdateUserUsedQuotaAndRequestCount(info.UserId, info.PriceData.Quota)
	model.UpdateChannelUsedQuota(info.ChannelId, info.PriceData.Quota)
}

// ---------------------------------------------------------------------------
// 异步任务计费辅助函数
// ---------------------------------------------------------------------------

// resolveTokenKey 通过 TokenId 运行时获取令牌 Key（用于 Redis 缓存操作）。
// 如果令牌已被删除或查询失败，返回空字符串。
func resolveTokenKey(ctx context.Context, tokenId int, taskID string) string {
	token, err := model.GetTokenById(tokenId)
	if err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("获取令牌 key 失败 (tokenId=%d, task=%s): %s", tokenId, taskID, err.Error()))
		return ""
	}
	return token.Key
}

// taskIsSubscription 判断任务是否通过订阅计费。
func taskIsSubscription(task *model.Task) bool {
	return task.PrivateData.BillingSource == BillingSourceSubscription && task.PrivateData.SubscriptionId > 0
}

// taskAdjustFunding 调整任务的资金来源（钱包或订阅），delta > 0 表示扣费，delta < 0 表示退还。
func taskAdjustFunding(task *model.Task, delta int) error {
	if taskIsSubscription(task) {
		return model.PostConsumeUserSubscriptionDelta(task.PrivateData.SubscriptionId, int64(delta))
	}
	if delta > 0 {
		return model.DecreaseUserQuota(task.UserId, delta, false)
	}
	return model.IncreaseUserQuota(task.UserId, -delta, false)
}

// taskAdjustTokenQuota 调整任务的令牌额度，delta > 0 表示扣费，delta < 0 表示退还。
// 需要通过 resolveTokenKey 运行时获取 key（不从 PrivateData 中读取）。
func taskAdjustTokenQuota(ctx context.Context, task *model.Task, delta int) {
	if task.PrivateData.TokenId <= 0 || delta == 0 {
		return
	}
	tokenKey := resolveTokenKey(ctx, task.PrivateData.TokenId, task.TaskID)
	if tokenKey == "" {
		return
	}
	var err error
	if delta > 0 {
		err = model.DecreaseTokenQuota(task.PrivateData.TokenId, tokenKey, delta)
	} else {
		err = model.IncreaseTokenQuota(task.PrivateData.TokenId, tokenKey, -delta)
	}
	if err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("调整令牌额度失败 (delta=%d, task=%s): %s", delta, task.TaskID, err.Error()))
	}
}

// taskBillingOther 从 task 的 BillingContext 构建日志 Other 字段。
func taskBillingOther(task *model.Task) map[string]interface{} {
	other := make(map[string]interface{})
	if bc := task.PrivateData.BillingContext; bc != nil {
		other["model_price"] = bc.ModelPrice
		if bc.ModelRatio > 0 {
			other["model_ratio"] = bc.ModelRatio
		}
		other["group_ratio"] = bc.GroupRatio
		if len(bc.OtherRatios) > 0 {
			for k, v := range bc.OtherRatios {
				other[k] = v
			}
		}
	}
	props := task.Properties
	if props.UpstreamModelName != "" && props.UpstreamModelName != props.OriginModelName {
		other["is_model_mapped"] = true
		other["upstream_model_name"] = props.UpstreamModelName
	}
	return other
}

// taskModelName 从 BillingContext 或 Properties 中获取模型名称。
func taskModelName(task *model.Task) string {
	if bc := task.PrivateData.BillingContext; bc != nil && bc.OriginModelName != "" {
		return bc.OriginModelName
	}
	return task.Properties.OriginModelName
}

// upstreamRefundableTaskFailureMarkers Geek2 等上游确认不扣费的失败（生成前/生成后审查拦截）。
var upstreamRefundableTaskFailureMarkers = []string{
	"unexpected end of json input",
	"appear to be unsafe",
}

// nonRefundableTaskFailureMarkers 上游内容审核/策略类失败：上游已扣费且不退款，我们也不应退还预扣额度。
var nonRefundableTaskFailureMarkers = []string{
	"content moderation",
	"content policy",
	"content violates",
	"content_policy_violation",
	"usage guidelines",
	"safety_check",
	"moderation_blocked",
}

// nonRefundableUpstreamErrorCodes OpenAI/Geek2 等内容策略类 error.code，命中时不退预扣费。
var nonRefundableUpstreamErrorCodes = []string{
	"content_policy_violation",
	"moderation_blocked",
}

// IsUpstreamRefundableTaskFailure 判断任务失败是否属于上游确认不扣费、我们需退预扣的场景。
func IsUpstreamRefundableTaskFailure(reason string) bool {
	lower := strings.ToLower(strings.TrimSpace(reason))
	if lower == "" {
		return false
	}
	for _, marker := range upstreamRefundableTaskFailureMarkers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

// IsNonRefundableTaskFailure 判断任务失败是否属于上游不退款的策略/审核类错误。
func IsNonRefundableTaskFailure(reason string) bool {
	if IsUpstreamRefundableTaskFailure(reason) {
		return false
	}
	lower := strings.ToLower(strings.TrimSpace(reason))
	if lower == "" {
		return false
	}
	for _, marker := range nonRefundableTaskFailureMarkers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

// IsNonRefundableUpstreamErrorCode 判断上游 error.code 是否属于策略/审核类（通常已计费）。
func IsNonRefundableUpstreamErrorCode(code string) bool {
	lower := strings.ToLower(strings.TrimSpace(code))
	if lower == "" || lower == "<nil>" {
		return false
	}
	for _, marker := range nonRefundableUpstreamErrorCodes {
		if lower == marker || strings.Contains(lower, marker) {
			return true
		}
	}
	for _, marker := range nonRefundableTaskFailureMarkers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func upstreamErrorFieldsFromBody(raw map[string]any) (message, code string) {
	errVal, ok := raw["error"]
	if !ok {
		return "", ""
	}
	switch v := errVal.(type) {
	case string:
		return v, ""
	case map[string]any:
		if msg, ok := v["message"].(string); ok {
			message = msg
		}
		if c, ok := v["code"].(string); ok {
			code = c
		} else if v["code"] != nil {
			code = fmt.Sprintf("%v", v["code"])
		}
	}
	return message, code
}

// ShouldRefundTaskOnFailure 决定异步/同步失败时是否退还预扣额度。
func ShouldRefundTaskOnFailure(reason string, responseBody []byte) bool {
	if IsUpstreamRefundableTaskFailure(reason) {
		return true
	}
	if IsNonRefundableTaskFailure(reason) {
		return false
	}
	if len(responseBody) == 0 {
		return true
	}
	var raw map[string]any
	if err := common.Unmarshal(responseBody, &raw); err != nil {
		return true
	}
	msg, code := upstreamErrorFieldsFromBody(raw)
	if IsUpstreamRefundableTaskFailure(msg) {
		return true
	}
	if IsNonRefundableUpstreamErrorCode(code) || IsNonRefundableTaskFailure(msg) {
		return false
	}
	if errVal, ok := raw["error"]; ok {
		switch v := errVal.(type) {
		case string:
			if IsUpstreamRefundableTaskFailure(v) {
				return true
			}
			if IsNonRefundableTaskFailure(v) {
				return false
			}
		case map[string]any:
			if m, ok := v["message"].(string); ok {
				if IsUpstreamRefundableTaskFailure(m) {
					return true
				}
				if IsNonRefundableTaskFailure(m) {
					return false
				}
			}
		}
	}
	return true
}

// ShouldRefundRelayError 决定同步 Relay 失败时是否退还 BillingSession 预扣费。
func ShouldRefundRelayError(apiErr *types.NewAPIError) bool {
	if apiErr == nil {
		return false
	}
	oai := apiErr.ToOpenAIError()
	reason := strings.TrimSpace(oai.Message)
	if reason == "" {
		reason = strings.TrimSpace(apiErr.Error())
	}
	code := strings.TrimSpace(fmt.Sprintf("%v", oai.Code))
	if IsUpstreamRefundableTaskFailure(reason) {
		return true
	}
	if IsNonRefundableUpstreamErrorCode(code) || IsNonRefundableTaskFailure(reason) {
		return false
	}
	var body []byte
	if code != "" || reason != "" {
		body, _ = common.Marshal(map[string]any{
			"error": map[string]any{
				"code":    code,
				"message": reason,
			},
		})
	}
	return ShouldRefundTaskOnFailure(reason, body)
}

// ShouldRefundTaskError 决定 Task 提交接口失败时是否退还 BillingSession 预扣费。
func ShouldRefundTaskError(taskErr *dto.TaskError) bool {
	if taskErr == nil {
		return false
	}
	return ShouldRefundTaskOnFailure(taskErr.Message, nil)
}

// MaybeRefundBilling 在失败时按策略退还 BillingSession 预扣费（同步/异步提交共用）。
func MaybeRefundBilling(c *gin.Context, billing relaycommon.BillingSettler, reason string, responseBody []byte) {
	if billing == nil {
		return
	}
	if ShouldRefundTaskOnFailure(reason, responseBody) {
		billing.Refund(c)
		return
	}
	logger.LogInfo(c, fmt.Sprintf("skip billing refund for non-refundable error: %s", reason))
}

// RefundTaskQuota 统一的任务失败退款逻辑。
// 当异步任务失败时，将预扣的 quota 退还给用户（支持钱包和订阅），并退还令牌额度。
func RefundTaskQuota(ctx context.Context, task *model.Task, reason string) {
	quota := task.Quota
	if quota == 0 {
		return
	}
	if IsNonRefundableTaskFailure(reason) {
		logger.LogInfo(ctx, fmt.Sprintf("Task %s failed with non-refundable error, skip refund: %s", task.TaskID, reason))
		return
	}

	// 1. 退还资金来源（钱包或订阅）
	if err := taskAdjustFunding(task, -quota); err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("退还资金来源失败 task %s: %s", task.TaskID, err.Error()))
		return
	}

	// 2. 退还令牌额度
	taskAdjustTokenQuota(ctx, task, -quota)

	// 3. 记录日志
	other := taskBillingOther(task)
	other["task_id"] = task.TaskID
	other["reason"] = reason
	model.RecordTaskBillingLog(model.RecordTaskBillingLogParams{
		UserId:    task.UserId,
		LogType:   model.LogTypeRefund,
		Content:   "",
		ChannelId: task.ChannelId,
		ModelName: taskModelName(task),
		Quota:     quota,
		TokenId:   task.PrivateData.TokenId,
		Group:     task.Group,
		Other:     other,
	})
}

func recordTaskBillingConsumeLog(ctx context.Context, task *model.Task, quota int, reason string, preConsumedQuota int) {
	if quota <= 0 {
		return
	}
	other := taskBillingOther(task)
	other["task_id"] = task.TaskID
	other["pre_consumed_quota"] = preConsumedQuota
	other["actual_quota"] = quota
	other["is_task"] = true
	model.RecordTaskBillingLog(model.RecordTaskBillingLogParams{
		UserId:    task.UserId,
		LogType:   model.LogTypeConsume,
		Content:   reason,
		ChannelId: task.ChannelId,
		ModelName: taskModelName(task),
		Quota:     quota,
		TokenId:   task.PrivateData.TokenId,
		Group:     task.Group,
		Other:     other,
	})
	model.UpdateUserUsedQuotaAndRequestCount(task.UserId, quota)
	model.UpdateChannelUsedQuota(task.ChannelId, quota)
}

// RecalculateTaskQuota 通用的异步差额结算。
// actualQuota 是任务完成后的实际应扣额度，与预扣额度 (task.Quota) 做差额结算。
// reason 用于日志记录（例如 "token重算" 或 "adaptor调整"）。
func RecalculateTaskQuota(ctx context.Context, task *model.Task, actualQuota int, reason string) {
	if actualQuota <= 0 {
		return
	}
	preConsumedQuota := task.Quota
	quotaDelta := actualQuota - preConsumedQuota

	if quotaDelta == 0 {
		logger.LogInfo(ctx, fmt.Sprintf("任务 %s 预扣费准确（%s，%s）",
			task.TaskID, logger.LogQuota(actualQuota), reason))
		recordTaskBillingConsumeLog(ctx, task, actualQuota, reason, preConsumedQuota)
		return
	}

	logger.LogInfo(ctx, fmt.Sprintf("任务 %s 差额结算：delta=%s（实际：%s，预扣：%s，%s）",
		task.TaskID,
		logger.LogQuota(quotaDelta),
		logger.LogQuota(actualQuota),
		logger.LogQuota(preConsumedQuota),
		reason,
	))

	// 调整资金来源
	if err := taskAdjustFunding(task, quotaDelta); err != nil {
		logger.LogError(ctx, fmt.Sprintf("差额结算资金调整失败 task %s: %s", task.TaskID, err.Error()))
		return
	}

	// 调整令牌额度
	taskAdjustTokenQuota(ctx, task, quotaDelta)

	task.Quota = actualQuota

	var logType int
	var logQuota int
	if quotaDelta > 0 {
		logType = model.LogTypeConsume
		logQuota = quotaDelta
		model.UpdateUserUsedQuotaAndRequestCount(task.UserId, quotaDelta)
		model.UpdateChannelUsedQuota(task.ChannelId, quotaDelta)
	} else {
		logType = model.LogTypeRefund
		logQuota = -quotaDelta
	}
	other := taskBillingOther(task)
	other["task_id"] = task.TaskID
	other["pre_consumed_quota"] = preConsumedQuota
	other["actual_quota"] = actualQuota
	model.RecordTaskBillingLog(model.RecordTaskBillingLogParams{
		UserId:    task.UserId,
		LogType:   logType,
		Content:   reason,
		ChannelId: task.ChannelId,
		ModelName: taskModelName(task),
		Quota:     logQuota,
		TokenId:   task.PrivateData.TokenId,
		Group:     task.Group,
		Other:     other,
	})
}

// RecalculateTaskQuotaByTokens 根据实际 token 消耗重新计费（异步差额结算）。
// 当任务成功且返回了 totalTokens 时，根据模型倍率和分组倍率重新计算实际扣费额度，
// 与预扣费的差额进行补扣或退还。支持钱包和订阅计费来源。
func RecalculateTaskQuotaByTokens(ctx context.Context, task *model.Task, totalTokens int) {
	if totalTokens <= 0 {
		return
	}

	modelName := taskModelName(task)

	// 获取模型价格和倍率
	modelRatio, hasRatioSetting, _ := ratio_setting.GetModelRatio(modelName)
	// 只有配置了倍率(非固定价格)时才按 token 重新计费
	if !hasRatioSetting || modelRatio <= 0 {
		return
	}

	// 获取用户和组的倍率信息
	group := task.Group
	if group == "" {
		user, err := model.GetUserById(task.UserId, false)
		if err == nil {
			group = user.Group
		}
	}
	if group == "" {
		return
	}

	groupRatio := ratio_setting.GetGroupRatio(group)
	userGroupRatio, hasUserGroupRatio := ratio_setting.GetGroupGroupRatio(group, group)

	var finalGroupRatio float64
	if hasUserGroupRatio {
		finalGroupRatio = userGroupRatio
	} else {
		finalGroupRatio = groupRatio
	}

	// 计算 OtherRatios 乘积（视频折扣、时长等）
	otherMultiplier := 1.0
	if bc := task.PrivateData.BillingContext; bc != nil {
		for _, r := range bc.OtherRatios {
			if r != 1.0 && r > 0 {
				otherMultiplier *= r
			}
		}
	}

	// 计算实际应扣费额度: totalTokens * modelRatio * groupRatio * otherMultiplier
	actualQuota := int(float64(totalTokens) * modelRatio * finalGroupRatio * otherMultiplier)

	reason := fmt.Sprintf("token重算：tokens=%d, modelRatio=%.2f, groupRatio=%.2f, otherMultiplier=%.4f", totalTokens, modelRatio, finalGroupRatio, otherMultiplier)
	RecalculateTaskQuota(ctx, task, actualQuota, reason)
}
