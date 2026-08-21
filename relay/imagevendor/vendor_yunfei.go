package imagevendor

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
)

const yunfeiGPTImage4KOrigin = "cy-yf-gpt-image-2-4k"

func init() {
	register(Descriptor{
		Name:              "yunfei-gpt-image-2-4k",
		Match:             matchYunfeiGPTImage4KModel,
		MatchRelay:        matchYunfeiGPTImage4KRelay,
		PatchRequest:      patchYunfeiGPTImage4KRequest,
		ValidateRequest:   validateYunfeiGPTImage4KRequest,
		PatchRelayRequest: patchYunfeiGPTImage4KRelayRequest,
		Rehost: RehostPolicy{
			AcceptUpstreamURL:      true,
			AsyncPreferURLResponse: true,
		},
	})
	register(Descriptor{
		Name:  "yunfei-gemini-banana",
		Match: IsYunfeiBananaOriginModel,
		Rehost: RehostPolicy{
			AcceptUpstreamURL: true,
		},
	})
}

// IsYunfeiBananaOriginModel reports Yunfei Gemini Banana internal models.
func IsYunfeiBananaOriginModel(originModel string) bool {
	name := normalizeOriginModel(originModel)
	return strings.HasPrefix(name, "cy-yf-gemini-banana")
}

func matchYunfeiGPTImage4KModel(originModel string) bool {
	return normalizeOriginModel(originModel) == yunfeiGPTImage4KOrigin
}

func matchYunfeiGPTImage4KRelay(info *relaycommon.RelayInfo) bool {
	return info != nil && matchYunfeiGPTImage4KModel(info.OriginModelName)
}

func validateYunfeiGPTImage4KRequest(_ *gin.Context, info *relaycommon.RelayInfo, request *dto.ImageRequest) error {
	if request == nil || info == nil || !matchYunfeiGPTImage4KModel(info.OriginModelName) {
		return nil
	}
	if err := validateFixedResolutionRequest(nil, info.OriginModelName, ImageResolution4K, true, request); err != nil {
		return err
	}
	_, err := resolveYunfeiGPTImage4KSize(request.Size)
	return err
}

func patchYunfeiGPTImage4KRequest(originModel string, request *dto.ImageRequest) (RequestPatchResult, error) {
	if !matchYunfeiGPTImage4KModel(originModel) {
		return RequestPatchResult{}, nil
	}
	return patchYunfeiGPTImage4KBody(request)
}

func patchYunfeiGPTImage4KRelayRequest(info *relaycommon.RelayInfo, request *dto.ImageRequest) (RequestPatchResult, error) {
	if info == nil || !matchYunfeiGPTImage4KModel(info.OriginModelName) {
		return RequestPatchResult{}, nil
	}
	return patchYunfeiGPTImage4KBody(request)
}

func patchYunfeiGPTImage4KBody(request *dto.ImageRequest) (RequestPatchResult, error) {
	if request == nil {
		return RequestPatchResult{}, fmt.Errorf("yunfei image patch: request is nil")
	}
	size, err := resolveYunfeiGPTImage4KSize(request.Size)
	if err != nil {
		return RequestPatchResult{}, err
	}
	request.Size = size
	request.N = common.GetPointer(uint(1))
	request.Stream = common.GetPointer(false)
	request.Quality = normalizeYunfeiGPTImageQuality(request.Quality)
	stripYunfeiUnsupportedFields(request)
	return RequestPatchResult{
		OutboundBodyChanged: true,
		SyncSizeToMultipart: true,
		SyncQualityToMultipart: true,
	}, nil
}

func normalizeYunfeiGPTImageQuality(quality string) string {
	switch strings.ToLower(strings.TrimSpace(quality)) {
	case "high", "hd", "4k":
		return "high"
	default:
		// Yunfei upstream defaults to medium when quality is omitted.
		return "medium"
	}
}

func resolveYunfeiGPTImage4KSize(size string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(size))
	if looksLikeExactImageSize(normalized) {
		if err := ValidateGPTImageExactSize(normalized, ImageResolution4K); err != nil {
			return "", err
		}
		return normalized, nil
	}
	if normalized == "" || normalized == "auto" {
		normalized = "1:1"
	}
	if suffix := "-" + strings.ToLower(ImageResolution4K); strings.HasSuffix(normalized, suffix) {
		normalized = strings.TrimSuffix(normalized, suffix)
	}
	if err := ValidateGPTImageAspectRatio(normalized); err != nil {
		return "", err
	}
	widthRatio, heightRatio, err := parseYunfeiGPTImageRatio(normalized)
	if err != nil {
		return "", err
	}
	const maxPixels int64 = 8_294_400
	maxScale := 3840 / max(widthRatio, heightRatio)
	for scale := maxScale; scale > 0; scale-- {
		width := widthRatio * scale
		height := heightRatio * scale
		pixels := int64(width) * int64(height)
		if width%16 == 0 && height%16 == 0 && pixels >= 655_360 && pixels <= maxPixels {
			return fmt.Sprintf("%dx%d", width, height), nil
		}
	}
	return "", fmt.Errorf("cannot resolve ratio %q within the 4K pixel budget", size)
}

func parseYunfeiGPTImageRatio(value string) (int, int, error) {
	parts := strings.Split(strings.TrimSpace(value), ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("aspect ratio must use WIDTH:HEIGHT")
	}
	width, widthErr := strconv.Atoi(strings.TrimSpace(parts[0]))
	height, heightErr := strconv.Atoi(strings.TrimSpace(parts[1]))
	if widthErr != nil || heightErr != nil {
		return 0, 0, fmt.Errorf("aspect ratio must use positive WIDTH:HEIGHT values")
	}
	return width, height, nil
}

func stripYunfeiUnsupportedFields(request *dto.ImageRequest) {
	request.Background = nil
	request.Moderation = nil
	request.OutputFormat = nil
	request.OutputCompression = nil
}
