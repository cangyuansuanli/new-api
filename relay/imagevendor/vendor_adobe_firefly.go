package imagevendor

import (
	"strings"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
)

func init() {
	register(Descriptor{
		Name:  "adobe-firefly",
		Match: IsAdobeFireflyOriginModel,
		Rehost: RehostPolicy{
			AcceptUpstreamURL:      true,
			AsyncPreferURLResponse: true,
		},
		ValidateRequest: validateAdobeFireflyRequest,
	})
}

func validateAdobeFireflyRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.ImageRequest) error {
	return ValidateFixedResolutionSKU(c, info.OriginModelName, request)
}

// FixedResolutionSKU returns the resolution encoded in an Adobe sellable image
// SKU. An xK suffix used by another channel is not the same product contract.
func FixedResolutionSKU(originModel string) (string, bool) {
	name := normalizeOriginModel(originModel)
	if !strings.HasPrefix(name, "adobe-firefly-") {
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

// ValidateFixedResolutionSKU rejects structured parameters that attempt to buy
// one Adobe SKU while requesting another. Prompt text is deliberately ignored.
func ValidateFixedResolutionSKU(c *gin.Context, originModel string, request *dto.ImageRequest) error {
	skuResolution, fixed := FixedResolutionSKU(originModel)
	if !fixed || request == nil {
		return nil
	}
	isGPTImage := strings.Contains(normalizeOriginModel(originModel), "gpt-image")
	return validateFixedResolutionRequest(c, originModel, skuResolution, isGPTImage, request)
}

// IsAdobeFireflyOriginModel matches the internal Adobe2API image SKU family.
// Adobe2API returns Adobe presigned URLs; workers must always rehost them to R2.
func IsAdobeFireflyOriginModel(originModel string) bool {
	return strings.HasPrefix(normalizeOriginModel(originModel), "adobe-firefly-")
}
