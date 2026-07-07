package image

import (
	"math"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/dto"
)

// channelImageSize4KDowngradeID：对该渠道发往上游前，将 4K size 自动降为 2K。
const channelImageSize4KDowngradeID = 72

var imageSize4KTo2KDirect = map[string]string{
	"3840x2160": "2048x1152",
	"2160x3840": "1152x2048",
	"2880x2880": "2048x2048",
	"4096x4096": "2048x2048",
	"16:9-4k":   "16:9-2k",
	"9:16-4k":   "9:16-2k",
	"1:1-4k":    "1:1-2k",
	"4k":        "2k",
}

// ApplyChannelImageSizeDowngrade 对指定渠道将 4K 分辨率 size 降为 2K；返回是否发生改写。
func ApplyChannelImageSizeDowngrade(channelID int, request *dto.ImageRequest) bool {
	if channelID != channelImageSize4KDowngradeID || request == nil {
		return false
	}
	downgraded, ok := downgradeImageSize4KTo2K(request.Size)
	if !ok {
		return false
	}
	request.Size = downgraded
	return true
}

func downgradeImageSize4KTo2K(size string) (string, bool) {
	trimmed := strings.TrimSpace(size)
	if trimmed == "" {
		return "", false
	}
	if mapped, ok := imageSize4KTo2KDirect[strings.ToLower(trimmed)]; ok {
		return mapped, true
	}
	if mapped, ok := imageSize4KTo2KDirect[trimmed]; ok {
		return mapped, true
	}
	if !isImageSize4K(trimmed) {
		return "", false
	}
	if scaled, ok := scaleImagePixelSizeTo2K(trimmed); ok {
		return scaled, true
	}
	return "", false
}

func isImageSize4K(size string) bool {
	trimmed := strings.TrimSpace(size)
	normalized := strings.ToLower(trimmed)
	switch normalized {
	case "4k":
		return true
	case "3840x2160", "2160x3840", "2880x2880", "4096x4096":
		return true
	case "16:9-4k", "9:16-4k", "1:1-4k":
		return true
	}
	width, height, ok := parseImagePixelSize(trimmed)
	if !ok {
		return false
	}
	maxEdge := width
	if height > maxEdge {
		maxEdge = height
	}
	return maxEdge > 2048
}

func scaleImagePixelSizeTo2K(size string) (string, bool) {
	width, height, ok := parseImagePixelSize(size)
	if !ok {
		return "", false
	}
	maxEdge := width
	if height > maxEdge {
		maxEdge = height
	}
	if maxEdge <= 2048 {
		return "", false
	}
	scale := float64(2048) / float64(maxEdge)
	scaledWidth := int(math.Round(float64(width) * scale))
	scaledHeight := int(math.Round(float64(height) * scale))
	if scaledWidth < 1 || scaledHeight < 1 {
		return "", false
	}
	return formatImagePixelSize(scaledWidth, scaledHeight), true
}

func parseImagePixelSize(size string) (int, int, bool) {
	parts := strings.Split(strings.ToLower(strings.TrimSpace(size)), "x")
	if len(parts) != 2 {
		return 0, 0, false
	}
	width, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || width <= 0 {
		return 0, 0, false
	}
	height, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil || height <= 0 {
		return 0, 0, false
	}
	return width, height, true
}

func formatImagePixelSize(width, height int) string {
	return strconv.Itoa(width) + "x" + strconv.Itoa(height)
}
