package clienterror

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/gin-gonic/gin"
)

func init() {
	Register(normalizeSD5)
	Register(normalizeTransport)
	RegisterRaw(normalizeCommon)
	RegisterRaw(normalizeOmni)
	Register(normalizeLeonardoRelay)
	RegisterRaw(normalizeAdobe)
	RegisterRaw(normalizeGrok)
	RegisterRaw(normalizeManju)
	RegisterRaw(normalizeChatVideo)
	RegisterRaw(normalizeDefaultVideo)
}

func PreferChineseClient(c *gin.Context) bool {
	if c == nil {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(c.GetHeader("X-Cangyuan-Client")), "infinite-canvas") {
		return true
	}
	lang := strings.ToLower(strings.TrimSpace(c.GetHeader("Accept-Language")))
	return strings.HasPrefix(lang, "zh")
}

func ContentPolicyMessage(c *gin.Context) string {
	if PreferChineseClient(c) {
		return ContentPolicyMessageZH
	}
	return ContentPolicyMessageEN
}

// NormalizeClientErrorMessage is the single entry for translating upstream/provider
// errors into client-facing copy. Vendor rules register via Register() in each file.
func NormalizeClientErrorMessage(c *gin.Context, raw string) string {
	return NormalizeError(c, requestErrorContext(c, raw))
}

// NormalizeClientErrorMessageForLang applies pre-processing then runs registered normalizers.
func NormalizeClientErrorMessageForLang(preferChinese bool, raw string) string {
	return NormalizeErrorForLang(preferChinese, ErrorContext{Raw: raw})
}

// NormalizeError is the single entry for both raw and structured upstream
// errors. Channel-specific context rules run before the generic raw rules.
func NormalizeError(c *gin.Context, failure ErrorContext) string {
	if strings.TrimSpace(failure.Model) == "" && c != nil {
		failure.Model = common.GetContextKeyString(c, constant.ContextKeyOriginalModel)
	}
	return NormalizeErrorForLang(PreferChineseClient(c), failure)
}

func NormalizeErrorForLang(preferChinese bool, failure ErrorContext) string {
	if len(failure.Payload) == 0 && failure.Raw != "" {
		failure.Payload = []byte(failure.Raw)
	}
	failure.Raw = stripLogArtifacts(failure.Raw)
	failure.Raw = stripStatusCodePrefix(failure.Raw)
	failure.Raw = unwrapUpstreamErrorText(failure.Raw)
	if failure.Raw == "" {
		if failure.ErrorCode == "" && failure.ErrorType == "" && len(failure.Payload) == 0 {
			return failure.Raw
		}
	}
	if msg, ok := runNormalizers(preferChinese, failure); ok {
		return msg
	}
	return failure.Raw
}

func NormalizeTaskErrorMessage(c *gin.Context, taskErr *dto.TaskError) {
	if taskErr == nil || (taskErr.Message == "" && taskErr.Code == "") {
		return
	}
	failure := requestErrorContext(c, taskErr.Message)
	failure.ErrorCode = taskErr.Code
	failure.HTTPStatus = taskErr.StatusCode
	taskErr.Message = NormalizeError(c, failure)
}

func NormalizeOpenAIImageJobError(c *gin.Context, job *dto.OpenAIImageJob) {
	NormalizeOpenAIImageJobErrorWithContext(c, job, ErrorContext{})
}

func NormalizeOpenAIImageJobErrorWithContext(c *gin.Context, job *dto.OpenAIImageJob, failure ErrorContext) {
	if job == nil || job.Error == nil || job.Error.Message == "" {
		return
	}
	failure.Raw = job.Error.Message
	job.Error.Message = NormalizeError(c, failure)
}

func NormalizeOpenAIVideoResponse(c *gin.Context, data []byte) []byte {
	return NormalizeOpenAIVideoResponseWithContext(c, data, ErrorContext{})
}

func NormalizeOpenAIVideoResponseWithContext(c *gin.Context, data []byte, failure ErrorContext) []byte {
	if IsLeonardoWeb2APIRelayModel(failure.Model) {
		return data
	}
	var payload map[string]any
	if err := common.Unmarshal(data, &payload); err != nil {
		return data
	}
	if errObj, ok := payload["error"].(map[string]any); ok {
		if msg, ok := errObj["message"].(string); ok && msg != "" {
			failure.Raw = msg
			errObj["message"] = NormalizeError(c, failure)
		}
	}
	if reason, ok := payload["fail_reason"].(string); ok && reason != "" {
		failure.Raw = reason
		payload["fail_reason"] = NormalizeError(c, failure)
	}
	out, err := common.Marshal(payload)
	if err != nil {
		return data
	}
	return out
}

func requestErrorContext(c *gin.Context, raw string) ErrorContext {
	failure := ErrorContext{Raw: raw}
	if c != nil {
		failure.Model = common.GetContextKeyString(c, constant.ContextKeyOriginalModel)
	}
	return failure
}
