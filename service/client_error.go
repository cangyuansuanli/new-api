package service

import (
	"strings"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	ce "github.com/QuantumNous/new-api/service/clienterror"
	"github.com/gin-gonic/gin"
)

const (
	ContentPolicyMessageZH = ce.ContentPolicyMessageZH
	ContentPolicyMessageEN = ce.ContentPolicyMessageEN

	UpstreamUnavailableMessageZH = ce.UpstreamUnavailableMessageZH
	UpstreamUnavailableMessageEN = ce.UpstreamUnavailableMessageEN

	TimeoutMessageZH = ce.TimeoutMessageZH
	TimeoutMessageEN = ce.TimeoutMessageEN

	MissingReferenceMessageZH = ce.MissingReferenceMessageZH
	MissingReferenceMessageEN = ce.MissingReferenceMessageEN

	ReferenceMaterialMessageZH = ce.ReferenceMaterialMessageZH
	ReferenceMaterialMessageEN = ce.ReferenceMaterialMessageEN
	ReferenceDurationTooLongZH = ce.ReferenceDurationTooLongZH
	ReferenceDurationTooLongEN = ce.ReferenceDurationTooLongEN

	ReferenceRealFaceMessageZH = ce.ReferenceRealFaceMessageZH
	ReferenceRealFaceMessageEN = ce.ReferenceRealFaceMessageEN

	GenerationFailedMessageZH = ce.GenerationFailedMessageZH
	GenerationFailedMessageEN = ce.GenerationFailedMessageEN

	GenerationFailedNoDetailZH = ce.GenerationFailedNoDetailZH
	GenerationFailedNoDetailEN = ce.GenerationFailedNoDetailEN

	InvalidRequestMessageZH = ce.InvalidRequestMessageZH
	InvalidRequestMessageEN = ce.InvalidRequestMessageEN

	PoolUnavailableMessageZH = ce.PoolUnavailableMessageZH
	PoolUnavailableMessageEN = ce.PoolUnavailableMessageEN
)

func PreferChineseClient(c *gin.Context) bool { return ce.PreferChineseClient(c) }

func ContentPolicyMessage(c *gin.Context) string { return ce.ContentPolicyMessage(c) }

func NormalizeClientErrorMessage(c *gin.Context, raw string) string {
	return ce.NormalizeClientErrorMessage(c, raw)
}

func NormalizeClientErrorMessageForLang(preferChinese bool, raw string) string {
	return ce.NormalizeClientErrorMessageForLang(preferChinese, raw)
}

func NormalizeTaskErrorMessage(c *gin.Context, taskErr *dto.TaskError) {
	ce.NormalizeTaskErrorMessage(c, taskErr)
}

func NormalizeOpenAIImageJobError(c *gin.Context, job *dto.OpenAIImageJob) {
	ce.NormalizeOpenAIImageJobError(c, job)
}

func NormalizeOpenAIVideoResponse(c *gin.Context, data []byte) []byte {
	return ce.NormalizeOpenAIVideoResponse(c, data)
}

func IsContentPolicyViolation(text string) bool { return ce.IsContentPolicyViolation(text) }

func IsRealFaceReferenceError(text string) bool { return ce.IsRealFaceReferenceError(text) }

func clientErrorContextFromTask(task *model.Task) ce.ErrorContext {
	if task == nil {
		return ce.ErrorContext{}
	}
	modelName := task.Properties.OriginModelName
	if modelName == "" && task.PrivateData.BillingContext != nil {
		modelName = task.PrivateData.BillingContext.OriginModelName
	}
	if modelName == "" {
		modelName = task.Properties.UpstreamModelName
	}
	if modelName == "" && task.PrivateData.BillingContext != nil {
		modelName = task.PrivateData.BillingContext.UpstreamModelName
	}
	return ce.ErrorContext{
		Model:   modelName,
		Raw:     task.FailReason,
		Payload: task.Data,
	}
}

// NormalizeTaskFailure renders a stored task failure without changing the
// persisted raw reason used by logging and billing.
func NormalizeTaskFailure(c *gin.Context, task *model.Task) string {
	if task == nil {
		return ""
	}
	ctx := clientErrorContextFromTask(task)
	if ce.IsLeonardoWeb2APIRelayModel(ctx.Model) {
		if raw := strings.TrimSpace(task.FailReason); raw != "" {
			return raw
		}
	}
	return ce.NormalizeError(c, ctx)
}

func NormalizeOpenAIImageTaskJobError(c *gin.Context, task *model.Task, job *dto.OpenAIImageJob) {
	ce.NormalizeOpenAIImageJobErrorWithContext(c, job, clientErrorContextFromTask(task))
}

func NormalizeOpenAIVideoTaskResponse(c *gin.Context, task *model.Task, data []byte) []byte {
	return ce.NormalizeOpenAIVideoResponseWithContext(c, data, clientErrorContextFromTask(task))
}
