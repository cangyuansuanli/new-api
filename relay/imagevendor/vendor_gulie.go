package imagevendor

import (
	"fmt"
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
	return strings.HasPrefix(name, "cy-img1-") || strings.HasPrefix(name, "gulie-")
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
	if strings.EqualFold(strings.TrimSpace(request.Size), "auto") {
		request.Size = ""
	}
	return result, nil
}

func stripGulieUnsupportedFields(request *dto.ImageRequest) {
	request.Quality = ""
	request.Background = nil
	request.Moderation = nil
	request.OutputFormat = nil
	request.OutputCompression = nil
}
