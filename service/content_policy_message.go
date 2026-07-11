package service

import (
	"encoding/json"
	"strings"

	"github.com/QuantumNous/new-api/dto"
	"github.com/gin-gonic/gin"
)

const (
	ContentPolicyMessageZH = "您的提示词或参考素材触发了上游内容审查，请修改后重新提交。"
	ContentPolicyMessageEN = "Your prompt or reference material was rejected by upstream content moderation. Please revise it and submit again."

	UpstreamUnavailableMessageZH = "上游服务暂时不可用，请稍后重试。"
	UpstreamUnavailableMessageEN = "Upstream service temporarily unavailable, please retry later."

	TimeoutMessageZH = "生成超时，请稍后重试。"
	TimeoutMessageEN = "Generation timed out, please retry later."

	MissingReferenceMessageZH = "参考图未正确传递，请重新上传后重试。"
	MissingReferenceMessageEN = "Reference image was not delivered correctly, please re-upload and retry."

	ReferenceMaterialMessageZH = "参考素材处理失败，请重新上传后重试。"
	ReferenceMaterialMessageEN = "Reference material could not be processed, please re-upload and retry."

	GenerationFailedMessageZH = "视频生成失败，请稍后重试。"
	GenerationFailedMessageEN = "Video generation failed, please retry later."

	GenerationFailedNoDetailZH = "Leonardo 上游生成失败且未提供具体原因；参考素材已成功提交，这不代表素材数量超限，请调整提示词或素材后重试。"
	GenerationFailedNoDetailEN = "Leonardo upstream generation failed without a specific reason. The references were accepted; this does not indicate a reference-count limit. Adjust the prompt or source material and retry."

	InvalidRequestMessageZH = "请求参数不符合要求，请检查后重试。"
	InvalidRequestMessageEN = "Request parameters are invalid, please check and retry."

	PoolUnavailableMessageZH = "视频服务暂时不可用，请稍后重试。"
	PoolUnavailableMessageEN = "Video service is temporarily unavailable, please retry later."
)

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
	return localizedClientMessage(c, ContentPolicyMessageZH, ContentPolicyMessageEN)
}

func localizedClientMessage(c *gin.Context, zh, en string) string {
	if PreferChineseClient(c) {
		return zh
	}
	return en
}

func IsContentPolicyViolation(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	lower := strings.ToLower(text)

	patterns := []string{
		"content moderation",
		"content policy",
		"content_policy",
		"content_policy_violation",
		"content_moderation_failed",
		"appear to be unsafe",
		"unsafe content",
		"policy violation",
		"sensitive_words",
		"sensitive words detected",
		"sexualization",
		"sexualized",
		"erotic focus",
		"erotic",
		"exposed cleavage",
		"prompt_blocked",
		"blocked by the upstream safety policy",
		"upstream safety policy",
		"model output was blocked",
		"generated video rejected by content moderation",
		"the generated images appear to be unsafe",
		"modifying the prompts or the seeds",
		"unexpected end of json input",
		"invalid character",
		"looking for beginning of value",
		"parse image json",
		"图片内容不合规",
		"内容审核",
		"未通过平台内容审核",
		"参考图未通过",
		"内容政策",
		"该提示可能违反了",
		"生成的图片可能违反了",
		"第三方内容相似",
		"裸露",
		"色情",
		"情色",
		"暴力内容",
		"防护限制",
	}

	for _, pattern := range patterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}

	if strings.Contains(text, "非常抱歉") {
		return strings.Contains(text, "内容政策") ||
			strings.Contains(text, "裸露") ||
			strings.Contains(text, "色情") ||
			strings.Contains(text, "情色") ||
			strings.Contains(text, "暴力") ||
			strings.Contains(text, "防护限制") ||
			strings.Contains(text, "第三方")
	}

	return false
}

func stripLogArtifacts(text string) string {
	text = strings.TrimSpace(text)
	if idx := strings.Index(text, "... [truncated"); idx != -1 {
		text = strings.TrimSpace(text[:idx])
	}
	if idx := strings.Index(text, "[truncated"); idx != -1 {
		text = strings.TrimSpace(text[:idx])
	}
	return text
}

func stripStatusCodePrefix(text string) string {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "status_code=") {
		return text
	}
	if idx := strings.Index(text, ", "); idx != -1 {
		return strings.TrimSpace(text[idx+2:])
	}
	return text
}

func containsAny(lower, text string, patterns ...string) bool {
	for _, pattern := range patterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	for _, pattern := range patterns {
		if strings.Contains(text, pattern) {
			return true
		}
	}
	return false
}

func IsUpstreamUnavailableError(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	lower := strings.ToLower(stripStatusCodePrefix(text))
	patterns := []string{
		"upstream service temporarily unavailable",
		"upstream request failed",
		"no capacity available",
		"capacity available for model",
		"bad response status code 502",
		"bad response status code 503",
		"bad response status code 504",
		"connection reset by peer",
		"connection refused",
		"download image failed",
		"rehost upstream image url",
	}
	if containsAny(lower, text, patterns...) {
		return true
	}
	if strings.HasPrefix(text, "status_code=502") ||
		strings.HasPrefix(text, "status_code=503") ||
		strings.HasPrefix(text, "status_code=504") {
		return true
	}
	return false
}

func IsTimeoutError(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	lower := strings.ToLower(stripStatusCodePrefix(text))
	if containsAny(lower, text, "proxy read timeout", "timed out", "timeout", "任务超时", "生图超时", "do request failed", "upstream error: do request failed", "context deadline exceeded", "client.timeout") {
		return true
	}
	return strings.HasPrefix(text, "status_code=524")
}

func IsMissingReferenceError(text string) bool {
	text = strings.TrimSpace(stripStatusCodePrefix(text))
	if text == "" {
		return false
	}
	lower := strings.ToLower(text)
	if strings.Contains(text, "图片1") &&
		(strings.Contains(text, "上传") || strings.Contains(text, "没有看到") || strings.Contains(text, "还没有看到") || strings.Contains(text, "引用")) {
		return true
	}
	if strings.Contains(text, "参考图") &&
		(strings.Contains(text, "上传") || strings.Contains(text, "还没有看到") || strings.Contains(text, "没有看到")) {
		return true
	}
	if strings.Contains(lower, "reference image") &&
		(strings.Contains(lower, "upload") || strings.Contains(lower, "don't have") || strings.Contains(lower, "do not have")) {
		return true
	}
	return false
}

func IsLeonardoPoolReferenceMaterialError(text string) bool {
	lower := strings.ToLower(stripStatusCodePrefix(text))
	patterns := []string{
		"leonardo: download",
		"upload reference",
		"upload start_frame",
		"upload end_frame",
		"originalfilename",
		"leonardo: uploadimage:",
		"media upload failed",
		"uploaded media processing timeout",
		"reference material could not be processed",
	}
	return containsAny(lower, text, patterns...)
}

func IsLeonardoPoolInvalidRequestError(text string) bool {
	lower := strings.ToLower(stripStatusCodePrefix(text))
	patterns := []string{
		"reference video duration",
		"reference audio duration",
		"exceed leonardo limit",
		"requires start_frame",
		"multimodal references cannot",
		"request parameters are invalid",
	}
	return containsAny(lower, text, patterns...)
}

func IsLeonardoPoolCapacityError(text string) bool {
	lower := strings.ToLower(stripStatusCodePrefix(text))
	patterns := []string{
		"no active cookie",
		"depleted (auto-disabled)",
		"token balance is empty",
		"insufficient credits",
		"dynamic proxy fetch failed",
		"auth_expired",
		"failed to fetch token",
	}
	return containsAny(lower, text, patterns...)
}

func IsLeonardoPoolGenerationFailed(text string) bool {
	text = strings.TrimSpace(stripStatusCodePrefix(text))
	if text == "" {
		return false
	}
	lower := strings.ToLower(text)
	return strings.Contains(lower, "all cookies failed") ||
		strings.Contains(lower, "video generation failed") ||
		strings.Contains(lower, "leonardo: video generation failed") ||
		strings.Contains(lower, "leonardo: generation failed") ||
		strings.Contains(lower, "generation_failed")
}

func extractLeonardoUpstreamFailureDetail(raw string) (detail string, noDetail bool) {
	lower := strings.ToLower(strings.TrimSpace(raw))
	if !strings.Contains(lower, "leonardo:") || !strings.Contains(lower, "generation failed") {
		return "", false
	}
	if strings.Contains(lower, "upstream returned no detail") {
		return "", true
	}
	// leonardo: video generation failed (FAILED): reason from upstream
	if idx := strings.Index(raw, "):"); idx >= 0 {
		if d := strings.TrimSpace(raw[idx+2:]); d != "" {
			return d, false
		}
	}
	return "", false
}

func NormalizeClientErrorMessage(c *gin.Context, raw string) string {
	return NormalizeClientErrorMessageForLang(PreferChineseClient(c), raw)
}

func NormalizeClientErrorMessageForLang(preferChinese bool, raw string) string {
	raw = stripLogArtifacts(raw)
	raw = stripStatusCodePrefix(raw)
	if raw == "" {
		return raw
	}
	if IsContentPolicyViolation(raw) {
		if preferChinese {
			return ContentPolicyMessageZH
		}
		return ContentPolicyMessageEN
	}
	if IsTimeoutError(raw) {
		if preferChinese {
			return TimeoutMessageZH
		}
		return TimeoutMessageEN
	}
	if IsUpstreamUnavailableError(raw) {
		if preferChinese {
			return UpstreamUnavailableMessageZH
		}
		return UpstreamUnavailableMessageEN
	}
	if IsMissingReferenceError(raw) {
		if preferChinese {
			return MissingReferenceMessageZH
		}
		return MissingReferenceMessageEN
	}
	if IsLeonardoPoolReferenceMaterialError(raw) {
		if preferChinese {
			return ReferenceMaterialMessageZH
		}
		return ReferenceMaterialMessageEN
	}
	if msg, ok := HumanizeLeonardoReferenceLimitError(preferChinese, raw); ok {
		return msg
	}
	if IsLeonardoPoolInvalidRequestError(raw) {
		if preferChinese {
			return InvalidRequestMessageZH
		}
		return InvalidRequestMessageEN
	}
	if IsLeonardoPoolCapacityError(raw) {
		if preferChinese {
			return PoolUnavailableMessageZH
		}
		return PoolUnavailableMessageEN
	}
	if detail, noDetail := extractLeonardoUpstreamFailureDetail(raw); noDetail {
		if preferChinese {
			return GenerationFailedNoDetailZH
		}
		return GenerationFailedNoDetailEN
	} else if detail != "" {
		if IsContentPolicyViolation(detail) {
			if preferChinese {
				return ContentPolicyMessageZH
			}
			return ContentPolicyMessageEN
		}
		if msg, ok := humanizeLeonardoGenerationFailureDetail(preferChinese, detail); ok {
			return msg
		}
		return detail
	}
	if IsLeonardoPoolGenerationFailed(raw) {
		if preferChinese {
			return GenerationFailedMessageZH
		}
		return GenerationFailedMessageEN
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
