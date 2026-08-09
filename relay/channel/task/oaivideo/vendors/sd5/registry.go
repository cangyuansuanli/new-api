package sd5

import "strings"

const modelPrefix = "cy-sd5-seedance-2.0"

func IsRelay(originModel, upstreamModel string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(originModel)), modelPrefix)
}
