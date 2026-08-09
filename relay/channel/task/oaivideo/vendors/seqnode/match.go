package seqnode

import "strings"

const (
	upstreamImagineVideo   = "grok-imagine-video"
	upstreamImagineVideo15 = "grok-imagine-video-1.5"
	originImagineVideo     = "cy-gv2-grok-video"
	originImagineVideo15   = "cy-gv2-grok-video-1.5"
)

func IsRelay(originModel, upstreamModel string, _ int) bool {
	origin := strings.ToLower(strings.TrimSpace(originModel))
	upstream := strings.ToLower(strings.TrimSpace(upstreamModel))
	switch origin {
	case originImagineVideo:
		return upstream == upstreamImagineVideo
	case originImagineVideo15:
		return upstream == upstreamImagineVideo15
	default:
		return false
	}
}
