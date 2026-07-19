package geeknowgrok

import "strings"

const (
	upstreamImagineVideo       = "grok-imagine-video"
	upstreamImagineVideo15Prev = "grok-imagine-video-1.5-preview"
)

func IsRelay(_ string, upstreamModel string) bool {
	switch strings.ToLower(strings.TrimSpace(upstreamModel)) {
	case upstreamImagineVideo, upstreamImagineVideo15Prev:
		return true
	default:
		return false
	}
}

func isImagine15Preview(originModel, upstreamModel string) bool {
	upstream := strings.ToLower(strings.TrimSpace(upstreamModel))
	if upstream == upstreamImagineVideo15Prev {
		return true
	}
	return strings.Contains(strings.ToLower(strings.TrimSpace(originModel)), "1.5")
}
