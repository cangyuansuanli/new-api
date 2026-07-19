package seedancetengda

import "strings"

const upstreamModel = "manxue-2.0"

func IsRelay(originModel, upstreamModelName string) bool {
	origin := strings.ToLower(strings.TrimSpace(originModel))
	if !strings.HasPrefix(origin, "cy-sd2-seedance") && !strings.HasPrefix(origin, "tengd-seedance") {
		return false
	}
	if strings.TrimSpace(upstreamModelName) == "" {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(upstreamModelName), upstreamModel)
}
