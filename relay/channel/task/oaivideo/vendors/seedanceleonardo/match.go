package seedanceleonardo

import "strings"

const (
	seedance20Model     = "cy-sd4-seedance-2.0"
	seedance20FastModel = "cy-sd4-seedance-2.0-fast"
	seedance20MiniModel = "cy-sd4-seedance-2.0-mini"
	mini8sModel         = "cy-sd4-seedance-2.0-mini-8s"
	seedance25Model480  = "cy-sd4-seedance-2.5-480p"
	seedance25Model720  = "cy-sd4-seedance-2.5-720p"
	minimax3Model768    = "cy-sd4-minimax-h3-768p"
	minimax3Model2K     = "cy-sd4-minimax-h3-2k"
	minimax3Model4K     = "cy-sd4-minimax-h3-4k"
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

func fixedResolution(originModel string) (string, bool) {
	if resolution, ok := seedance25Resolution(originModel); ok {
		return resolution, true
	}
	switch strings.ToLower(strings.TrimSpace(originModel)) {
	case minimax3Model768:
		return "768p", true
	case minimax3Model2K:
		return "2k", true
	case minimax3Model4K:
		return "4k", true
	default:
		return "", false
	}
}

func IsRelay(originModel, upstreamModel string) bool {
	origin := strings.ToLower(strings.TrimSpace(originModel))
	upstream := strings.ToLower(strings.TrimSpace(upstreamModel))
	switch origin {
	case seedance20Model:
		return upstream == "seedance-2.0"
	case seedance20FastModel:
		return upstream == "seedance-2.0-fast"
	case seedance20MiniModel, mini8sModel:
		return upstream == "seedance-2.0-mini"
	case seedance25Model480, seedance25Model720:
		return upstream == "seedance-2.5"
	case minimax3Model768, minimax3Model2K, minimax3Model4K:
		return upstream == "hailuo-03"
	case "cy-sd4-happyhouse-1.0":
		return upstream == "happy-horse"
	case "cy-sd4-happyhouse-1.1":
		return upstream == "happy-horse-1.1"
	default:
		return false
	}
}
