package imagevendor

import "strings"

func init() {
	register(Descriptor{
		Name:  "adobe-firefly",
		Match: IsAdobeFireflyOriginModel,
		Rehost: RehostPolicy{
			AcceptUpstreamURL:      true,
			AsyncPreferURLResponse: true,
		},
	})
}

// IsAdobeFireflyOriginModel matches the internal Adobe2API image SKU family.
// Adobe2API returns Adobe presigned URLs; workers must always rehost them to R2.
func IsAdobeFireflyOriginModel(originModel string) bool {
	return strings.HasPrefix(normalizeOriginModel(originModel), "adobe-firefly-")
}
