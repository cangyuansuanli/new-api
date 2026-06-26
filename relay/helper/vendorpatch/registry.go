package vendorpatch

import (
	"strings"

	"github.com/QuantumNous/new-api/dto"
)

// ImagePatcher applies vendor-specific adjustments to OpenAI-format image requests
// before they are converted and sent upstream.
type ImagePatcher interface {
	Match(originModel string) bool
	Apply(request *dto.ImageRequest) (ImageTransformResult, error)
}

var imagePatchers []ImagePatcher

func registerImagePatcher(patcher ImagePatcher) {
	imagePatchers = append(imagePatchers, patcher)
}

// ApplyImage runs the first registered patcher that matches originModel.
func ApplyImage(originModel string, request *dto.ImageRequest) (ImageTransformResult, error) {
	for _, patcher := range imagePatchers {
		if patcher.Match(originModel) {
			return patcher.Apply(request)
		}
	}
	return ImageTransformResult{}, nil
}

func matchModelPrefix(originModel, prefix string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(originModel)), strings.ToLower(prefix))
}
