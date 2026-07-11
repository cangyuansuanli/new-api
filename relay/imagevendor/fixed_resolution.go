package imagevendor

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/dto"
	"github.com/gin-gonic/gin"
)

const (
	ImageResolution1K = "1K"
	ImageResolution2K = "2K"
	ImageResolution4K = "4K"
)

// FixedResolutionSKU returns the resolution encoded in an Adobe sellable image
// SKU. This is deliberately scoped to Adobe's internal namespace: an xK suffix
// used by another vendor is not the same product contract.
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
// one resolution SKU while requesting another. Prompt text is deliberately not
// inspected: writing "4K" in a prompt cannot override Adobe2API's image_size.
func ValidateFixedResolutionSKU(c *gin.Context, originModel string, request *dto.ImageRequest) error {
	skuResolution, fixed := FixedResolutionSKU(originModel)
	if !fixed || request == nil {
		return nil
	}
	if request.N != nil && *request.N > 1 {
		return fmt.Errorf("model %s only supports n=1", originModel)
	}

	hints := make([]resolutionHint, 0, 8)
	appendResolutionHint(&hints, "size", request.Size, false)
	appendResolutionHint(&hints, "quality", request.Quality, false)
	collectResolutionHintsFromRaw(&hints, "extra_fields", request.ExtraFields)
	for key, raw := range request.Extra {
		collectResolutionHintsFromRaw(&hints, key, raw)
	}
	if c != nil && c.Request != nil && c.Request.MultipartForm != nil {
		for key, values := range c.Request.MultipartForm.Value {
			for _, value := range values {
				appendResolutionHint(&hints, key, value, isStrictResolutionKey(key))
			}
		}
	}

	for _, hint := range hints {
		if hint.invalid {
			return fmt.Errorf("invalid image resolution in %s: %q", hint.source, hint.raw)
		}
		if hint.tier != "" && hint.tier != skuResolution {
			return fmt.Errorf("model %s is a fixed %s SKU, but %s requests %s", originModel, skuResolution, hint.source, hint.tier)
		}
	}
	return nil
}

type resolutionHint struct {
	source  string
	raw     string
	tier    string
	invalid bool
}

func collectResolutionHintsFromRaw(hints *[]resolutionHint, source string, raw json.RawMessage) {
	if len(raw) == 0 {
		return
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return
	}
	if scalar, ok := scalarString(value); ok {
		appendResolutionHint(hints, source, scalar, isStrictResolutionKey(source))
	}
	collectResolutionHints(hints, source, value)
}

func collectResolutionHints(hints *[]resolutionHint, source string, value any) {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			childSource := source + "." + key
			if scalar, ok := scalarString(child); ok {
				appendResolutionHint(hints, childSource, scalar, isStrictResolutionKey(key))
			}
			collectResolutionHints(hints, childSource, child)
		}
	case []any:
		for index, child := range typed {
			collectResolutionHints(hints, fmt.Sprintf("%s[%d]", source, index), child)
		}
	}
}

func appendResolutionHint(hints *[]resolutionHint, source, value string, strict bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	key := source
	if dot := strings.LastIndex(key, "."); dot >= 0 {
		key = key[dot+1:]
	}
	key = normalizeResolutionKey(key)

	var tier string
	var ok bool
	switch key {
	case "quality":
		tier, ok = resolutionFromQuality(value)
	case "size", "imagesize", "outputresolution", "resolution":
		tier, ok = classifyStructuredResolution(value)
	default:
		return
	}
	if ok {
		*hints = append(*hints, resolutionHint{source: source, raw: value, tier: tier})
		return
	}
	if strict && !isVideoResolution(value) {
		*hints = append(*hints, resolutionHint{source: source, raw: value, invalid: true})
	}
}

func classifyStructuredResolution(value string) (string, bool) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "1k":
		return ImageResolution1K, true
	case "2k":
		return ImageResolution2K, true
	case "4k":
		return ImageResolution4K, true
	case "auto":
		return "", false
	}
	for _, tier := range []string{"1k", "2k", "4k"} {
		if strings.HasSuffix(normalized, "-"+tier) || strings.HasSuffix(normalized, "_"+tier) {
			return strings.ToUpper(tier), true
		}
	}
	parts := strings.Split(normalized, "x")
	if len(parts) != 2 {
		return "", false
	}
	width, widthErr := strconv.Atoi(strings.TrimSpace(parts[0]))
	height, heightErr := strconv.Atoi(strings.TrimSpace(parts[1]))
	if widthErr != nil || heightErr != nil || width <= 0 || height <= 0 {
		return "", false
	}
	maxEdge := width
	if height > maxEdge {
		maxEdge = height
	}
	switch {
	case maxEdge <= 1024:
		return ImageResolution1K, true
	case maxEdge <= 2048:
		return ImageResolution2K, true
	default:
		return ImageResolution4K, true
	}
}

func resolutionFromQuality(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "low", "standard", "1k":
		return ImageResolution1K, true
	case "medium", "2k":
		return ImageResolution2K, true
	case "high", "hd", "4k":
		return ImageResolution4K, true
	default:
		return "", false
	}
}

func normalizeResolutionKey(key string) string {
	key = strings.ToLower(strings.TrimSpace(key))
	key = strings.ReplaceAll(key, "_", "")
	key = strings.ReplaceAll(key, "-", "")
	return key
}

func isStrictResolutionKey(key string) bool {
	switch normalizeResolutionKey(key) {
	case "imagesize", "outputresolution":
		return true
	default:
		return false
	}
}

func isVideoResolution(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "480p", "540p", "720p", "1080p", "1440p", "2160p":
		return true
	default:
		return false
	}
}

func scalarString(value any) (string, bool) {
	switch typed := value.(type) {
	case string:
		return typed, true
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64), true
	default:
		return "", false
	}
}
