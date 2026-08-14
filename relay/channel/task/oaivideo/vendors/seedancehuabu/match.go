package seedancehuabu

import "strings"

const (
	ModelStandard      = "cy-sd8-seedance-2.0"
	ModelFast          = "cy-sd8-seedance-2.0-fast"
	UpstreamStandard   = "sd2.0-933"
	UpstreamFast       = "sd-2.0-fast-v1"
	PublicStandard     = "sd8-seedance-2.0"
	PublicFast         = "sd8-seedance-2.0-fast"
)

// IsRelay matches only explicit internal → upstream model pairs from model_mapping.
// Routing must not depend on channel ID, base URL, or vendor priority.
func IsRelay(originModel, upstreamModel string) bool {
	origin := strings.ToLower(strings.TrimSpace(originModel))
	upstream := strings.ToLower(strings.TrimSpace(upstreamModel))
	switch origin {
	case ModelStandard:
		return upstream == UpstreamStandard
	case ModelFast:
		return upstream == UpstreamFast
	default:
		return false
	}
}
