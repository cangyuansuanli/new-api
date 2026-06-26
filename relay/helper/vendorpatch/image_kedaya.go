package vendorpatch

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/dto"
)

const (
	kedayaMaxEdge  = 1024
	kedayaSizeStep = 16
)

var kedayaSizePattern = regexp.MustCompile(`(?i)^(\d+)x(\d+)$`)

type kedayaImagePatcher struct{}

func (kedayaImagePatcher) Match(originModel string) bool {
	return matchModelPrefix(originModel, "kedaya-")
}

func (kedayaImagePatcher) Apply(request *dto.ImageRequest) (ImageTransformResult, error) {
	if request == nil {
		return ImageTransformResult{}, fmt.Errorf("kedaya image patch: request is nil")
	}

	stripKedayaUnsupportedFields(request)

	originalSize := strings.TrimSpace(request.Size)
	result := ImageTransformResult{SuppressQualityLog: true}
	if originalSize == "" || strings.EqualFold(originalSize, "auto") {
		request.Size = ""
		return result, nil
	}

	width, height, ok := parsePixelSize(originalSize)
	if !ok {
		return ImageTransformResult{}, fmt.Errorf("kedaya image patch: invalid size %q", originalSize)
	}

	upstreamWidth, upstreamHeight := mapKedayaUpstreamSize(width, height)
	request.Size = fmt.Sprintf("%dx%d", upstreamWidth, upstreamHeight)
	request.Prompt = appendKedayaSizeHint(request.Prompt, width, height)
	result.LogSize = originalSize
	return result, nil
}

func init() {
	registerImagePatcher(kedayaImagePatcher{})
}

func stripKedayaUnsupportedFields(request *dto.ImageRequest) {
	request.Quality = ""
	request.Background = nil
	request.Moderation = nil
	request.OutputFormat = nil
	request.OutputCompression = nil
}

func parsePixelSize(size string) (width, height int, ok bool) {
	match := kedayaSizePattern.FindStringSubmatch(strings.TrimSpace(size))
	if match == nil {
		return 0, 0, false
	}
	width, errW := parsePositiveInt(match[1])
	height, errH := parsePositiveInt(match[2])
	if errW != nil || errH != nil {
		return 0, 0, false
	}
	return width, height, true
}

func parsePositiveInt(raw string) (int, error) {
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("invalid integer %q", raw)
	}
	return parsed, nil
}

func alignKedayaDimension(value int) int {
	if value <= 0 {
		return kedayaSizeStep
	}
	aligned := int(math.Ceil(float64(value)/float64(kedayaSizeStep))) * kedayaSizeStep
	if aligned > kedayaMaxEdge {
		return kedayaMaxEdge
	}
	return aligned
}

func mapKedayaUpstreamSize(width, height int) (int, int) {
	maxSide := max(width, height)
	if maxSide <= kedayaMaxEdge {
		return alignKedayaDimension(width), alignKedayaDimension(height)
	}

	scale := float64(kedayaMaxEdge) / float64(maxSide)
	scaledWidth := int(math.Round(float64(width) * scale))
	scaledHeight := int(math.Round(float64(height) * scale))
	scaledWidth = alignKedayaDimension(scaledWidth)
	scaledHeight = alignKedayaDimension(scaledHeight)

	if scaledWidth > kedayaMaxEdge || scaledHeight > kedayaMaxEdge {
		maxScaled := max(scaledWidth, scaledHeight)
		rescale := float64(kedayaMaxEdge) / float64(maxScaled)
		scaledWidth = alignKedayaDimension(int(math.Round(float64(scaledWidth) * rescale)))
		scaledHeight = alignKedayaDimension(int(math.Round(float64(scaledHeight) * rescale)))
	}
	return scaledWidth, scaledHeight
}

func appendKedayaSizeHint(prompt string, width, height int) string {
	hint := fmt.Sprintf("尺寸：%d*%d", width, height)
	trimmedPrompt := strings.TrimSpace(prompt)
	if trimmedPrompt == "" {
		return hint
	}
	if strings.Contains(trimmedPrompt, hint) {
		return trimmedPrompt
	}
	return trimmedPrompt + "\n" + hint
}
