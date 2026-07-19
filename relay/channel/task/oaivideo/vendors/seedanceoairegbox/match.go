package seedanceoairegbox

import "strings"

func IsRelay(originModel string) bool {
	origin := strings.ToLower(strings.TrimSpace(originModel))
	return strings.HasPrefix(origin, "cy-sd1-seedance")
}
