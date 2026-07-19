package seedanceleonardo

import "strings"

const mini8sModel = "cy-sd4-seedance-2.0-mini-8s"

func IsRelay(originModel string) bool {
	origin := strings.ToLower(strings.TrimSpace(originModel))
	return strings.HasPrefix(origin, "cy-sd4-seedance")
}
