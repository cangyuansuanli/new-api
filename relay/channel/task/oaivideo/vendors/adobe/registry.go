package adobe

import (
	"net/url"
	"strings"
)

// IsRelay identifies Adobe by channel identity first. Model prefixes are a
// fallback for callers that resolve a vendor before channel metadata exists.
func IsRelay(originModel, upstreamModel string, channelID int, baseURL string) bool {
	if channelID == 75 || isAdobeBaseURL(baseURL) {
		return true
	}
	for _, model := range []string{originModel, upstreamModel} {
		name := strings.ToLower(strings.TrimSpace(model))
		for _, prefix := range []string{
			"adobe-", "adobe/", "adobe2api-", "adobe2api/",
			"firefly-sora", "firefly-veo", "firefly-kling", "firefly-seedance",
		} {
			if strings.HasPrefix(name, prefix) {
				return true
			}
		}
	}
	return false
}

func isAdobeBaseURL(raw string) bool {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return false
	}
	if strings.Contains(raw, "adobe2api") || strings.Contains(raw, "gongju") {
		return true
	}
	u, err := url.Parse(raw)
	return err == nil && u.Host == "45.67.221.45:6001"
}
