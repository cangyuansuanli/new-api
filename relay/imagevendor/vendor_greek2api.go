package imagevendor

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
)

const greek2APIChannelID = 48

func init() {
	register(Descriptor{
		Name:       "greek2api-gpt-image-2",
		Match:      matchGreek2APIGPTImageModel,
		MatchRelay: matchGreek2APIRelay,
		Rehost: RehostPolicy{
			AcceptUpstreamURL:      true,
			AsyncPreferURLResponse: true,
		},
		ValidateRequest:   validateGreek2APIRequest,
		PatchRelayRequest: patchGreek2APIRequest,
	})
}

func matchGreek2APIGPTImageModel(originModel string) bool {
	_, ok := greek2APIResolutionTier(originModel)
	return ok
}

func matchGreek2APIRelay(info *relaycommon.RelayInfo) bool {
	return info != nil && info.ChannelMeta != nil && info.ChannelId == greek2APIChannelID && matchGreek2APIGPTImageModel(info.OriginModelName)
}

func greek2APIResolutionTier(originModel string) (string, bool) {
	name := normalizeOriginModel(originModel)
	const prefix = "cy-img2-gpt-image-2-"
	if !strings.HasPrefix(name, prefix) {
		return "", false
	}
	switch strings.TrimPrefix(name, prefix) {
	case "1k":
		return ImageResolution1K, true
	case "2k":
		return ImageResolution2K, true
	case "4k":
		return ImageResolution4K, true
	default:
		return "", false
	}
}

func validateGreek2APIRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.ImageRequest) error {
	if request == nil {
		return nil
	}
	tier, ok := greek2APIResolutionTier(info.OriginModelName)
	if !ok {
		return nil
	}
	if err := validateFixedResolutionRequest(c, info.OriginModelName, tier, true, request); err != nil {
		return err
	}
	_, err := resolveGreek2APISize(tier, request.Size)
	return err
}

func patchGreek2APIRequest(info *relaycommon.RelayInfo, request *dto.ImageRequest) (RequestPatchResult, error) {
	if request == nil {
		return RequestPatchResult{}, fmt.Errorf("greek2api image patch: request is nil")
	}
	tier, ok := greek2APIResolutionTier(info.OriginModelName)
	if !ok {
		return RequestPatchResult{}, nil
	}
	size, err := resolveGreek2APISize(tier, request.Size)
	if err != nil {
		return RequestPatchResult{}, err
	}
	request.Size = size
	return RequestPatchResult{
		OutboundBodyChanged: true,
		SyncSizeToMultipart: true,
	}, nil
}

func resolveGreek2APISize(tier, size string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(size))
	if looksLikeExactImageSize(normalized) {
		if err := ValidateGPTImageExactSize(normalized, tier); err != nil {
			return "", err
		}
		return normalized, nil
	}
	if normalized == "" || normalized == "auto" || normalized == strings.ToLower(tier) {
		normalized = "1:1"
	}
	if suffix := "-" + strings.ToLower(tier); strings.HasSuffix(normalized, suffix) {
		normalized = strings.TrimSuffix(normalized, suffix)
	}
	if err := ValidateGPTImageAspectRatio(normalized); err != nil {
		return "", err
	}
	widthRatio, heightRatio, err := parseGreek2APIRatio(normalized)
	if err != nil {
		return "", err
	}
	maxPixels := map[string]int64{
		ImageResolution1K: 1_048_576,
		ImageResolution2K: 4_194_304,
		ImageResolution4K: 8_294_400,
	}[tier]
	maxScale := 3840 / max(widthRatio, heightRatio)
	for scale := maxScale; scale > 0; scale-- {
		width := widthRatio * scale
		height := heightRatio * scale
		pixels := int64(width) * int64(height)
		if width%16 == 0 && height%16 == 0 && pixels >= 655_360 && pixels <= maxPixels {
			return fmt.Sprintf("%dx%d", width, height), nil
		}
	}
	return "", fmt.Errorf("cannot resolve ratio %q within the %s pixel budget", size, tier)
}

func parseGreek2APIRatio(value string) (int, int, error) {
	parts := strings.Split(strings.TrimSpace(value), ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("aspect ratio must use WIDTH:HEIGHT")
	}
	width, widthErr := strconv.Atoi(strings.TrimSpace(parts[0]))
	height, heightErr := strconv.Atoi(strings.TrimSpace(parts[1]))
	if widthErr != nil || heightErr != nil || width <= 0 || height <= 0 {
		return 0, 0, fmt.Errorf("aspect ratio must use positive WIDTH:HEIGHT values")
	}
	return width, height, nil
}
