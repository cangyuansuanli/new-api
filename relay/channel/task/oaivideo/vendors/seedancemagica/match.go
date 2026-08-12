package seedancemagica

import (
	"strings"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

const (
	Model720p           = "cy-sd7-seedance-2.0-720p"
	Model1080p          = "cy-sd7-seedance-2.0-1080p"
	UpstreamStandard    = "seedance-2.0"
	UpstreamReference   = "seedance-2.0-reference"
	ChannelUpstreamSlug = UpstreamStandard // New-API 渠道统一填 seedance-2.0
)

func IsRelay(originModel, upstreamModel string) bool {
	origin := strings.ToLower(strings.TrimSpace(originModel))
	upstream := strings.ToLower(strings.TrimSpace(upstreamModel))
	if upstream != ChannelUpstreamSlug {
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

// resolveUpstreamModel picks reference vs standard from request payload.
// 多参（参考图/视频/音频）走 seedance-2.0-reference；单图 i2v / 纯文生走 seedance-2.0。
func resolveUpstreamModel(req relaycommon.TaskSubmitReq) string {
	if len(req.ReferenceVideos) > 0 || len(req.ReferenceAudios) > 0 || len(req.Images) > 1 {
		return UpstreamReference
	}
	return UpstreamStandard
}
