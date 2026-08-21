package imagevendor

import (
	"strings"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
)

func init() {
	register(Descriptor{
		Name:  "aiclub",
		Match: IsAiclubOriginModel,
		Rehost: RehostPolicy{
			AcceptUpstreamURL:      true,
			AsyncPreferURLResponse: true,
		},
		ReferenceInput:  ReferenceInputURLJSON,
		ValidateRequest: validateAiclubRequest,
	})
}

func validateAiclubRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.ImageRequest) error {
	if info == nil || request == nil || !IsAiclubOriginModel(info.OriginModelName) {
		return nil
	}
	return ValidateAiclubFixedResolutionSKU(c, info.OriginModelName, request)
}

// IsAiclubOriginModel matches the internal Aiclub image SKU family.
func IsAiclubOriginModel(originModel string) bool {
	return strings.HasPrefix(normalizeOriginModel(originModel), "cy-ac-")
}

// AiclubFixedResolutionSKU returns the resolution encoded in a cy-ac-* sellable SKU.
func AiclubFixedResolutionSKU(originModel string) (string, bool) {
	name := normalizeOriginModel(originModel)
	if !strings.HasPrefix(name, "cy-ac-") {
		return "", false
	}
	if !strings.Contains(name, "gpt-image") && !strings.Contains(name, "nano-banana") {
		return "", false
	}
	for _, candidate := range []string{ImageResolution1K, ImageResolution2K, ImageResolution4K} {
		if strings.HasSuffix(name, "-"+strings.ToLower(candidate)) {
			return candidate, true
		}
	}
	return "", false
}

// ValidateAiclubFixedResolutionSKU rejects parameters that bypass the billed SKU tier.
func ValidateAiclubFixedResolutionSKU(c *gin.Context, originModel string, request *dto.ImageRequest) error {
	skuResolution, fixed := AiclubFixedResolutionSKU(originModel)
	if !fixed || request == nil {
		return nil
	}
	isGPTImage := strings.Contains(normalizeOriginModel(originModel), "gpt-image")
	return validateFixedResolutionRequest(c, originModel, skuResolution, isGPTImage, request)
}
