package clienterror

import "strings"

// Magica / cy-sd7 pool. Upstream: magica-web2api/

func IsMagicaWeb2APIRelayModel(model string) bool {
	model = strings.ToLower(strings.TrimSpace(model))
	return strings.HasPrefix(model, "cy-sd7-seedance-")
}

func normalizeMagicaRelay(preferChinese bool, failure ErrorContext) (string, bool) {
	if !IsMagicaWeb2APIRelayModel(failure.Model) {
		return "", false
	}
	if msg, ok := normalizeMagicaPool(preferChinese, failure.Raw); ok {
		return msg, true
	}
	if cleaned := sanitizeMagicaClientMessage(failure.Raw); cleaned != strings.TrimSpace(failure.Raw) {
		return cleaned, true
	}
	return "", false
}

func normalizeMagicaPool(preferChinese bool, raw string) (string, bool) {
	raw = sanitizeMagicaClientMessage(raw)
	if strings.Contains(raw, PoolDepletedMessageZH) {
		return localized(preferChinese, PoolDepletedMessageZH, PoolDepletedMessageEN), true
	}
	if IsLeonardoInsufficientCreditsForJobError(raw) {
		return localized(preferChinese, InsufficientCreditsForJobMessageZH, InsufficientCreditsForJobMessageEN), true
	}
	if msg, ok := humanizeMagicaKeyPoolFailure(preferChinese, raw); ok {
		return msg, true
	}
	if strings.Contains(strings.ToLower(raw), "no active api key") {
		return localized(preferChinese, PoolDepletedMessageZH, PoolDepletedMessageEN), true
	}
	if IsContentPolicyViolation(raw) {
		return localized(preferChinese, ContentPolicyMessageZH, ContentPolicyMessageEN), true
	}
	if msg, ok := humanizeMagicaPayloadTooLarge(preferChinese, raw); ok {
		return msg, true
	}
	return "", false
}

func humanizeMagicaPayloadTooLarge(preferChinese bool, raw string) (string, bool) {
	lower := strings.ToLower(raw)
	if !strings.Contains(lower, "payload_too_large") &&
		!strings.Contains(lower, "entity too large") &&
		!strings.Contains(lower, "请求体过大") {
		return "", false
	}
	if preferChinese {
		return "参考素材过大，请先将图片/视频上传到 OSS，只传公网 HTTPS 链接，勿使用 base64 或本地 data URL。", true
	}
	return "Reference media is too large. Upload assets to OSS and pass public HTTPS URLs only.", true
}

func sanitizeMagicaClientMessage(raw string) string {
	raw = strings.TrimSpace(raw)
	replacements := []struct{ old, new string }{
		{"参考图 必须是公网 HTTPS 链接；不支持 base64/data URL（Magica 上游请求体上限约 4MB）", "参考图必须是公网 HTTPS 链接，不支持 base64 或本地 data URL"},
		{"参考视频 必须是公网 HTTPS 链接；不支持 base64/data URL（Magica 上游请求体上限约 4MB）", "参考视频必须是公网 HTTPS 链接，不支持 base64 或本地 data URL"},
		{"参考音频 必须是公网 HTTPS 链接；不支持 base64/data URL（Magica 上游请求体上限约 4MB）", "参考音频必须是公网 HTTPS 链接，不支持 base64 或本地 data URL"},
		{"参考素材请求体过大：请使用公网 HTTPS 链接，勿传 base64/data URL（Magica 上游限制约 4MB）", "参考素材过大，请使用公网 HTTPS 链接，勿传 base64 或本地 data URL"},
	}
	for _, r := range replacements {
		raw = strings.ReplaceAll(raw, r.old, r.new)
	}
	if i := strings.Index(raw, "（Magica"); i >= 0 {
		if j := strings.Index(raw[i:], "）"); j >= 0 {
			raw = strings.TrimSpace(raw[:i] + raw[i+j+len("）"):])
		}
	}
	return raw
}

func humanizeMagicaKeyPoolFailure(preferChinese bool, raw string) (string, bool) {
	lower := strings.ToLower(strings.TrimSpace(raw))
	if !strings.Contains(lower, "all api keys failed") {
		return "", false
	}
	if IsLeonardoInsufficientCreditsForJobError(raw) {
		return localized(preferChinese, InsufficientCreditsForJobMessageZH, InsufficientCreditsForJobMessageEN), true
	}
	if strings.Contains(lower, "balance below minimum") || strings.Contains(lower, "insufficient credits") {
		return localized(preferChinese, InsufficientCreditsForJobMessageZH, InsufficientCreditsForJobMessageEN), true
	}
	if msg, ok := humanizeMagicaPayloadTooLarge(preferChinese, raw); ok {
		return msg, true
	}
	return localized(preferChinese, PoolUnavailableMessageZH, PoolUnavailableMessageEN), true
}
