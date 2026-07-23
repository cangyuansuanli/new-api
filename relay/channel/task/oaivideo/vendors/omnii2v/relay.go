package omnii2v

import "strings"

func IsRelay(originModel, upstreamModel string) bool {
	upstream := strings.ToLower(strings.TrimSpace(upstreamModel))
	if strings.Contains(upstream, "omni-fast-v2v") {
		return false
	}
	if strings.Contains(upstream, "omni-fast") {
		return true
	}
	origin := strings.ToLower(strings.TrimSpace(originModel))
	if strings.Contains(origin, "omni-v2v") {
		return false
	}
	if strings.Contains(origin, "omni-fast") {
		return true
	}
	return strings.Contains(origin, "cy-sd1-omni") && !strings.Contains(origin, "v2v")
}
