package imagevendor

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
)

func init() {
	preferUpstreamURL := common.GetEnvOrDefaultBool("IMAGE_GULIE_UPSTREAM_URL_ENABLED", true)
	register(Descriptor{
		Name:         "gulie-gpt-image",
		Match:        matchGulieGPTImageModel,
		PatchRequest: patchGulieImageRequest,
		Rehost: RehostPolicy{
			AcceptUpstreamURL:      true,
			PreferUpstreamB64JSON:  !preferUpstreamURL,
			AsyncPreferURLResponse: preferUpstreamURL,
		},
	})
}

func matchGulieGPTImageModel(originModel string) bool {
	name := normalizeOriginModel(originModel)
	if strings.HasPrefix(name, "cy-img1-") || strings.HasPrefix(name, "gulie-") {
		return true
	}
	// cy-img2- 默认走 Geek2 4K；仅 2K 经济档挂在 Gulie 渠道 72
	return name == "cy-img2-gpt-image-2-2k"
}

func patchGulieImageRequest(originModel string, request *dto.ImageRequest) (RequestPatchResult, error) {
	if !matchGulieGPTImageModel(originModel) {
		return RequestPatchResult{}, nil
	}
	if request == nil {
		return RequestPatchResult{}, fmt.Errorf("gulie image patch: request is nil")
	}

	stripGulieUnsupportedFields(request)
	request.Stream = common.GetPointer(false)

	result := RequestPatchResult{SuppressQualityLog: true}
	if normalizeOriginModel(originModel) == "cy-img2-gpt-image-2-2k" {
		sanitizeGulie2KImageRequest(request)
	} else if strings.EqualFold(strings.TrimSpace(request.Size), "auto") {
		request.Size = ""
	}
	return result, nil
}

// IsGulie2KImageModel：Gulie 渠道 72 固定 2K 经济档，上游不接受客户自选分辨率。
func IsGulie2KImageModel(originModel string) bool {
	return normalizeOriginModel(originModel) == "cy-img2-gpt-image-2-2k"
}

// Gulie2KUpstreamFormStripKeys：multipart 转发前须从表单删除的分辨率相关字段。
func Gulie2KUpstreamFormStripKeys() []string {
	return []string{
		"quality",
		"image_size",
		"output_resolution",
		"resolution",
		"imageSize",
		"outputResolution",
	}
}

// sanitizeGulie2KImageRequest 剥离客户分辨率参数，仅保留画幅比例；2K 档位由上游模型固定。
func sanitizeGulie2KImageRequest(request *dto.ImageRequest) {
	if request == nil {
		return
	}
	request.Quality = ""
	request.Size = normalizeGulie2KAspectSize(request.Size)
	stripGulie2KExtraResolution(request)
}

func stripGulie2KExtraResolution(request *dto.ImageRequest) {
	if len(request.Extra) == 0 {
		return
	}
	for _, key := range Gulie2KUpstreamFormStripKeys() {
		delete(request.Extra, key)
	}
}

func normalizeGulie2KAspectSize(size string) string {
	trimmed := strings.TrimSpace(size)
	if trimmed == "" || strings.EqualFold(trimmed, "auto") {
		return ""
	}
	lower := strings.ToLower(trimmed)
	for _, suffix := range []string{"-4k", "-2k", "-1k"} {
		if strings.HasSuffix(lower, suffix) {
			candidate := strings.TrimSpace(trimmed[:len(trimmed)-len(suffix)])
			if ratio := normalizePureAspectRatio(candidate); ratio != "" {
				return ratio
			}
		}
	}
	if ratio := normalizePureAspectRatio(trimmed); ratio != "" {
		return ratio
	}
	switch lower {
	case "1024x1024", "2048x2048", "4096x4096", "2880x2880":
		return "1:1"
	case "1536x1024", "2048x1152", "3840x2160":
		return "3:2"
	case "1024x1536", "1152x2048", "2160x3840":
		return "2:3"
	case "4k", "2k", "1k", "high", "hd", "medium", "low", "standard":
		return ""
	}
	if ratio := aspectRatioFromPixelSize(trimmed); ratio != "" {
		return ratio
	}
	return ""
}

func normalizePureAspectRatio(value string) string {
	parts := strings.Split(strings.TrimSpace(value), ":")
	if len(parts) != 2 {
		return ""
	}
	width, errW := strconv.Atoi(strings.TrimSpace(parts[0]))
	height, errH := strconv.Atoi(strings.TrimSpace(parts[1]))
	if errW != nil || errH != nil || width <= 0 || height <= 0 {
		return ""
	}
	divisor := gcdInt(width, height)
	return fmt.Sprintf("%d:%d", width/divisor, height/divisor)
}

func aspectRatioFromPixelSize(size string) string {
	width, height, ok := parsePixelSize(size)
	if !ok {
		return ""
	}
	divisor := gcdInt(width, height)
	return fmt.Sprintf("%d:%d", width/divisor, height/divisor)
}

func parsePixelSize(size string) (int, int, bool) {
	parts := strings.Split(strings.ToLower(strings.TrimSpace(size)), "x")
	if len(parts) != 2 {
		return 0, 0, false
	}
	width, errW := strconv.Atoi(strings.TrimSpace(parts[0]))
	height, errH := strconv.Atoi(strings.TrimSpace(parts[1]))
	if errW != nil || errH != nil || width <= 0 || height <= 0 {
		return 0, 0, false
	}
	return width, height, true
}

func gcdInt(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

func stripGulieUnsupportedFields(request *dto.ImageRequest) {
	request.Quality = ""
	request.Background = nil
	request.Moderation = nil
	request.OutputFormat = nil
	request.OutputCompression = nil
}
