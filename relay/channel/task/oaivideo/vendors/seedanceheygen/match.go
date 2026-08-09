package seedanceheygen

import "strings"

const (
	Model720p     = "cy-sd6-seedance-2.0-720p"
	Model1080p    = "cy-sd6-seedance-2.0-1080p"
	UpstreamModel = "seedance-2.0"
)

func IsRelay(originModel, upstreamModel string) bool {
	origin := strings.ToLower(strings.TrimSpace(originModel))
	upstream := strings.ToLower(strings.TrimSpace(upstreamModel))
	if upstream != UpstreamModel {
		return false
	}
	return origin == Model720p || origin == Model1080p
}

func resolutionForModel(originModel string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(originModel)) {
	case Model720p:
		return "720p", true
	case Model1080p:
		return "1080p", true
	default:
		return "", false
	}
}
