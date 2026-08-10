package seedanceleonardo

import "strings"

const (
	mini8sModel        = "cy-sd4-seedance-2.0-mini-8s"
	seedance25Model480 = "cy-sd4-seedance-2.5-480p"
	seedance25Model720 = "cy-sd4-seedance-2.5-720p"
)

func seedance25Resolution(originModel string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(originModel)) {
	case seedance25Model480:
		return "480p", true
	case seedance25Model720:
		return "720p", true
	default:
		return "", false
	}
}

func IsRelay(originModel string) bool {
	origin := strings.ToLower(strings.TrimSpace(originModel))
	return strings.HasPrefix(origin, "cy-sd4-seedance") ||
		strings.HasPrefix(origin, "cy-sd4-minimax-h3") ||
		strings.HasPrefix(origin, "cy-sd4-happyhouse-")
}
