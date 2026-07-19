package clienterror

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Adobe2API / cy-sd5 / adobe-direct image+video. Upstream: adobe2api/
//   - core/media_limits.py
//   - api/routes/generation.py
//   - app.py
// new-api relay validation: relay/channel/openai/adapt_adobe2api.go

var adobeReferenceCountRe = regexp.MustCompile(`(?i)reference images exceed (\d+)`)

func normalizeAdobe(preferChinese bool, raw string) (string, bool) {
	lower := strings.ToLower(stripStatusCodePrefix(raw))

	if strings.Contains(lower, "mask supports exactly one") {
		if preferChinese {
			return "蒙版仅支持上传 1 个文件或 URL。", true
		}
		return "Mask supports exactly one file or URL.", true
	}
	if strings.Contains(lower, "mask is only supported for adobe gpt image") {
		if preferChinese {
			return "当前模型不支持蒙版，请更换为支持蒙版的图像模型。", true
		}
		return "Mask is not supported by this model. Please use a model that supports masking.", true
	}
	if m := adobeReferenceCountRe.FindStringSubmatch(raw); len(m) == 2 {
		max, _ := strconv.Atoi(m[1])
		if max > 0 {
			if preferChinese {
				return fmt.Sprintf("参考图最多 %d 张，请减少后重试。", max), true
			}
			return fmt.Sprintf("At most %d reference images allowed. Please remove extras and retry.", max), true
		}
	}
	if strings.Contains(lower, "entities in one prompt must belong to the same adobe account") {
		if preferChinese {
			return "同一提示词中的主体必须属于同一账号，请调整主体选择。", true
		}
		return "Entities in one prompt must belong to the same account.", true
	}
	if strings.Contains(lower, "pillow is required for video image preprocessing") {
		if preferChinese {
			return "参考图预处理失败，请更换图片格式后重试。", true
		}
		return "Reference image preprocessing failed. Please try a different image format.", true
	}
	if strings.Contains(lower, "prompt_unsafe") ||
		strings.Contains(lower, "provided prompt is considered unsafe") ||
		strings.Contains(lower, "cannot be used to generate content") {
		return localized(preferChinese, ContentPolicyMessageZH, ContentPolicyMessageEN), true
	}

	return "", false
}
