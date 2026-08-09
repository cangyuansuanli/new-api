package seqnode

import "strings"

const (
	upstreamImagineVideo   = "grok-imagine-video"
	upstreamImagineVideo15 = "grok-imagine-video-1.5"
)

func IsRelay(originModel, upstreamModel string, channelID int) bool {
	if channelID != 106 {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(upstreamModel), upstreamImagineVideo) || strings.EqualFold(strings.TrimSpace(upstreamModel), upstreamImagineVideo15) || strings.HasPrefix(strings.ToLower(strings.TrimSpace(originModel)), "cy-gv2-grok-video")
}
