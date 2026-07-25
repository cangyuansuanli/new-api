package clienterror

import (
	"encoding/json"
	"strings"

	"github.com/QuantumNous/new-api/dto"
	"github.com/gin-gonic/gin"
)

func init() {
	Register(normalizeCommon)
	Register(normalizeOmni)
	Register(normalizeLeonardo)
	Register(normalizeAdobe)
	Register(normalizeGrok)
	Register(normalizeManju)
	Register(normalizeChatVideo)
	Register(normalizeDefaultVideo)
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
	return NormalizeClientErrorMessageForLang(PreferChineseClient(c), raw)
}

// NormalizeClientErrorMessageForLang applies pre-processing then runs registered normalizers.
func NormalizeClientErrorMessageForLang(preferChinese bool, raw string) string {
	raw = stripLogArtifacts(raw)
	raw = stripStatusCodePrefix(raw)
	raw = unwrapUpstreamErrorText(raw)
	if raw == "" {
		return raw
	}
	if msg, ok := runNormalizers(preferChinese, raw); ok {
		return msg
	}
	return raw
}

func NormalizeTaskErrorMessage(c *gin.Context, taskErr *dto.TaskError) {
	if taskErr == nil || taskErr.Message == "" {
		return
	}
	taskErr.Message = NormalizeClientErrorMessage(c, taskErr.Message)
}

func NormalizeOpenAIImageJobError(c *gin.Context, job *dto.OpenAIImageJob) {
	if job == nil || job.Error == nil || job.Error.Message == "" {
		return
	}
	job.Error.Message = NormalizeClientErrorMessage(c, job.Error.Message)
}

func NormalizeOpenAIVideoResponse(c *gin.Context, data []byte) []byte {
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return data
	}
	if errObj, ok := payload["error"].(map[string]any); ok {
		if msg, ok := errObj["message"].(string); ok && msg != "" {
			errObj["message"] = NormalizeClientErrorMessage(c, msg)
		}
	}
	if reason, ok := payload["fail_reason"].(string); ok && reason != "" {
		payload["fail_reason"] = NormalizeClientErrorMessage(c, reason)
	}
	out, err := json.Marshal(payload)
	if err != nil {
		return data
	}
	return out
}
