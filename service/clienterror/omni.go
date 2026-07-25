package clienterror

import "strings"

// Omni / oairegbox (cy-sd1-omni-*). Upstream: Gemini Veo via oairegbox relay.
// Product notes: infinite-canvas/docs/dev/models/oairegbox-omni-video.md §七

func normalizeOmni(preferChinese bool, raw string) (string, bool) {
	if IsOmniContentReviewError(raw) {
		return localized(preferChinese, ContentPolicyMessageZH, ContentPolicyMessageEN), true
	}
	return "", false
}

func IsOmniContentReviewError(text string) bool {
	text = strings.TrimSpace(stripStatusCodePrefix(text))
	if text == "" {
		return false
	}
	lower := strings.ToLower(text)
	return strings.Contains(lower, "didn't pass content review") ||
		strings.Contains(lower, "did not pass content review")
}
