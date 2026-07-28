package clienterror

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

var (
	referenceByteLimitRe = regexp.MustCompile(
		`(?i)(?:reference[_ ]?)?(image|video|audio|mask|entity(?:[_ ]image)?|\w+) too large, max (\d+)\s*mb`,
	)
	promptLengthLimitRe = regexp.MustCompile(
		`(?i)prompt length exceeds the maximum allowed length of (\d+)`,
	)
	promptExceedsCharsRe = regexp.MustCompile(`(?i)prompt exceeds (\d+) characters`)
	tooManyImagesRe      = regexp.MustCompile(`(?i)too many images, max (\d+)`)
)

func localized(preferChinese bool, zh, en string) string {
	if preferChinese {
		return zh
	}
	return en
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

func unwrapUpstreamErrorText(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}
	if msg := extractMessageFromJSON(raw); msg != "" {
		return msg
	}
	if idx := strings.Index(raw, "{"); idx >= 0 {
		if msg := extractMessageFromJSON(raw[idx:]); msg != "" {
			return msg
		}
	}
	return raw
}

func extractMessageFromJSON(raw string) string {
	var payload map[string]any
	if err := common.Unmarshal([]byte(raw), &payload); err != nil {
		return ""
	}
	if detail, ok := payload["detail"].(string); ok && strings.TrimSpace(detail) != "" {
		return strings.TrimSpace(detail)
	}
	if msg, ok := payload["message"].(string); ok && strings.TrimSpace(msg) != "" {
		return strings.TrimSpace(msg)
	}
	if msg, ok := payload["msg"].(string); ok && strings.TrimSpace(msg) != "" {
		return strings.TrimSpace(msg)
	}
	switch errValue := payload["error"].(type) {
	case map[string]any:
		if msg, ok := errValue["message"].(string); ok && strings.TrimSpace(msg) != "" {
			return strings.TrimSpace(msg)
		}
	case string:
		if strings.TrimSpace(errValue) != "" {
			return strings.TrimSpace(errValue)
		}
	}
	return ""
}

func humanizeReferenceByteLimitError(preferChinese bool, raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	m := referenceByteLimitRe.FindStringSubmatch(raw)
	if len(m) != 3 {
		return "", false
	}
	maxMB, err := strconv.Atoi(m[2])
	if err != nil || maxMB <= 0 {
		return "", false
	}
	maxBytes := int64(maxMB) * (1 << 20)
	if preferChinese {
		return common.FormatReferenceByteLimitMessageZH(m[1], maxBytes), true
	}
	return common.FormatReferenceByteLimitMessageEN(m[1], maxBytes), true
}

func humanizePromptLengthError(preferChinese bool, raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	if m := promptLengthLimitRe.FindStringSubmatch(raw); len(m) == 2 {
		limit, err := strconv.Atoi(m[1])
		if err == nil && limit > 0 {
			return formatPromptLengthLimitMessage(preferChinese, limit), true
		}
	}
	if m := promptExceedsCharsRe.FindStringSubmatch(raw); len(m) == 2 {
		limit, err := strconv.Atoi(m[1])
		if err == nil && limit > 0 {
			return formatPromptLengthLimitMessage(preferChinese, limit), true
		}
	}
	return "", false
}

func humanizeTooManyImagesError(preferChinese bool, raw string) (string, bool) {
	m := tooManyImagesRe.FindStringSubmatch(strings.TrimSpace(raw))
	if len(m) != 2 {
		return "", false
	}
	max, err := strconv.Atoi(m[1])
	if err != nil || max <= 0 {
		return "", false
	}
	if preferChinese {
		return "参考图最多 " + strconv.Itoa(max) + " 张，请减少后重试。", true
	}
	return "At most " + strconv.Itoa(max) + " reference images allowed. Please remove extras and retry.", true
}

func formatPromptLengthLimitMessage(preferChinese bool, limit int) string {
	if preferChinese {
		return "提示词超过上限（" + strconv.Itoa(limit) + " 字符），请缩短后再试"
	}
	return "Prompt exceeds the maximum length of " + strconv.Itoa(limit) + " characters. Please shorten it and retry."
}

func isGenericTaskFailed(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "task failed", "upstream returned empty status", "upstream returned unrecognized message":
		return true
	default:
		return false
	}
}
