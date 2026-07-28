package clienterror

import (
	"encoding/json"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

type sd5FailurePayload struct {
	Error       json.RawMessage `json:"error"`
	Message     string          `json:"message"`
	ErrorCode   string          `json:"error_code"`
	ErrorType   string          `json:"error_type"`
	ErrorStatus int             `json:"error_status"`
}

type sd5NestedError struct {
	Message string `json:"message"`
	Code    string `json:"code"`
	Type    string `json:"type"`
}

func normalizeSD5(preferChinese bool, failure ErrorContext) (string, bool) {
	if !isSD5TaskFailure(failure) {
		return "", false
	}

	code, raw := parseSD5Failure(failure)
	lower := strings.ToLower(raw)

	switch code {
	case "submission_overloaded":
		return sd5Localized(preferChinese,
			"SD5 上游负载过高或提交超时，请稍后重试。",
			"The SD5 upstream is overloaded or the submission timed out. Please retry later."), true
	case "reference_image_privacy_error":
		return localized(preferChinese, ReferenceRealFaceMessageZH, ReferenceRealFaceMessageEN), true
	case "video_unsafe", "prompt_unsafe", "policy_error":
		return localized(preferChinese, ContentPolicyMessageZH, ContentPolicyMessageEN), true
	case "bad_request":
		if strings.Contains(lower, "cannot fetch content from the provided url") ||
			strings.Contains(lower, "url_rejected") {
			return sd5Localized(preferChinese,
				"参考素材链接无法被上游访问，请确认链接可公开下载后重新提交。",
				"The upstream could not access a reference URL. Make sure it is publicly downloadable and submit again."), true
		}
		return localized(preferChinese, InvalidRequestMessageZH, InvalidRequestMessageEN), true
	case "image_not_processable":
		return sd5Localized(preferChinese,
			"参考视频无法处理，文件可能损坏或格式不受支持，请重新编码或更换素材。",
			"The reference video could not be processed. It may be corrupt or use an unsupported format; re-encode or replace it."), true
	case "invalid_stored_request":
		if strings.Contains(lower, "private address") {
			return sd5Localized(preferChinese,
				"参考素材链接指向内网或私有地址，上游无法访问，请改用公开 HTTPS 链接。",
				"The reference URL resolves to a private address. Use a publicly accessible HTTPS URL."), true
		}
		return localized(preferChinese, InvalidRequestMessageZH, InvalidRequestMessageEN), true
	case "reference_fetch_error":
		if strings.Contains(lower, "status 404") {
			return sd5Localized(preferChinese,
				"参考素材链接已失效或返回 404，请重新上传后提交。",
				"A reference URL returned 404 or expired. Re-upload the source media and submit again."), true
		}
		return localized(preferChinese, ReferenceMaterialMessageZH, ReferenceMaterialMessageEN), true
	case "submission_unknown":
		return sd5Localized(preferChinese,
			"SD5 提交结果未能确认，任务已终止且不会自动重试，请重新提交。",
			"The SD5 submission result could not be confirmed. The task was stopped and will not retry automatically; please submit again."), true
	case "upstream_timeout":
		return localized(preferChinese, TimeoutMessageZH, TimeoutMessageEN), true
	case "validation_error":
		if strings.Contains(lower, "video/webm") {
			return sd5Localized(preferChinese,
				"SD5 当前不支持 WebM 参考视频，请转换为 MP4 后重试。",
				"SD5 does not currently support WebM reference video. Convert it to MP4 and retry."), true
		}
		if strings.Contains(lower, "prompt") && strings.Contains(lower, "2500") {
			return sd5Localized(preferChinese,
				"提示词超过 2500 字符，请缩短后重新提交。",
				"The prompt exceeds 2500 characters. Shorten it and submit again."), true
		}
		return localized(preferChinese, InvalidRequestMessageZH, InvalidRequestMessageEN), true
	case "access_error", "authentication_error":
		return sd5Localized(preferChinese,
			"SD5 上游账号权限异常，请稍后重试或联系管理员。",
			"The SD5 upstream account has an authorization problem. Retry later or contact an administrator."), true
	case "quota_exhausted", "taste_exhausted":
		return sd5Localized(preferChinese,
			"SD5 号池可用额度已耗尽，请联系管理员补充额度。",
			"The SD5 pool has exhausted its available credits. Contact an administrator to replenish it."), true
	}

	// Submission-time errors do not always carry a provider-specific code.
	// Keep their raw matching inside the SD5 boundary instead of common.go.
	switch {
	case strings.Contains(lower, "media video/audio references require at least one image reference"):
		return sd5Localized(preferChinese,
			"使用参考视频或音频时，至少需要同时提供 1 张参考图。",
			"At least one reference image is required when using reference video or audio."), true
	case strings.Contains(lower, "request body too large"):
		return sd5Localized(preferChinese,
			"请求体超过 64 MB，请压缩或减少参考素材后重试。",
			"The request body exceeds 64 MB. Compress or remove reference media and retry."), true
	case strings.Contains(lower, "system under load"):
		return sd5Localized(preferChinese,
			"SD5 上游负载过高，请稍后重试。",
			"The SD5 upstream is overloaded. Please retry later."), true
	}

	return "", false
}

// parseSD5Failure owns the SD5/Adobe2API response schema. In particular, the
// public video response may expose its provider error_code as error_type.
func parseSD5Failure(failure ErrorContext) (code, raw string) {
	code = strings.TrimSpace(failure.ErrorType)
	if code == "" {
		code = strings.TrimSpace(failure.ErrorCode)
	}
	raw = strings.TrimSpace(failure.Raw)
	if len(failure.Payload) == 0 {
		return strings.ToLower(code), raw
	}

	var payload sd5FailurePayload
	if err := common.Unmarshal(failure.Payload, &payload); err != nil {
		return strings.ToLower(code), raw
	}
	if providerCode := strings.TrimSpace(payload.ErrorCode); providerCode != "" {
		code = providerCode
	} else if providerType := strings.TrimSpace(payload.ErrorType); providerType != "" {
		code = providerType
	}
	if message := strings.TrimSpace(payload.Message); message != "" {
		raw = message
	}
	if len(payload.Error) > 0 && string(payload.Error) != "null" {
		switch common.GetJsonType(payload.Error) {
		case "string":
			var message string
			if err := common.Unmarshal(payload.Error, &message); err == nil && strings.TrimSpace(message) != "" {
				raw = strings.TrimSpace(message)
			}
		case "object":
			var nested sd5NestedError
			if err := common.Unmarshal(payload.Error, &nested); err == nil {
				if message := strings.TrimSpace(nested.Message); message != "" {
					raw = message
				}
				if nestedCode := strings.TrimSpace(nested.Code); nestedCode != "" {
					code = nestedCode
				} else if nestedType := strings.TrimSpace(nested.Type); nestedType != "" {
					code = nestedType
				}
			}
		}
	}
	return strings.ToLower(code), raw
}

func isSD5TaskFailure(failure ErrorContext) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(failure.Model)), "cy-sd5-seedance-2.0")
}

func sd5Localized(preferChinese bool, zh, en string) string {
	if preferChinese {
		return zh
	}
	return en
}
