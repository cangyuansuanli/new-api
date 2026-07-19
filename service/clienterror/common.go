package clienterror

import "strings"

// Cross-channel rules. Upstream sources: adobe2api, relay layers, generic HTTP/proxy errors.
// See coverage.md for per-vendor gaps.

func normalizeCommon(preferChinese bool, raw string) (string, bool) {
	if IsRealFaceReferenceError(raw) {
		return localized(preferChinese, ReferenceRealFaceMessageZH, ReferenceRealFaceMessageEN), true
	}
	if IsContentPolicyViolation(raw) {
		return localized(preferChinese, ContentPolicyMessageZH, ContentPolicyMessageEN), true
	}
	if IsTimeoutError(raw) {
		return localized(preferChinese, TimeoutMessageZH, TimeoutMessageEN), true
	}
	if msg, ok := humanizeUpstreamUnavailableError(preferChinese, raw); ok {
		return msg, true
	}
	if IsMissingReferenceError(raw) {
		return localized(preferChinese, MissingReferenceMessageZH, MissingReferenceMessageEN), true
	}
	if msg, ok := humanizeReferenceByteLimitError(preferChinese, raw); ok {
		return msg, true
	}
	if msg, ok := humanizePromptLengthError(preferChinese, raw); ok {
		return msg, true
	}
	if msg, ok := humanizeTooManyImagesError(preferChinese, raw); ok {
		return msg, true
	}
	if IsReferenceMaterialError(raw) {
		return localized(preferChinese, ReferenceMaterialMessageZH, ReferenceMaterialMessageEN), true
	}
	if IsGenericInvalidRequestError(raw) {
		return localized(preferChinese, InvalidRequestMessageZH, InvalidRequestMessageEN), true
	}
	return "", false
}

func IsRealFaceReferenceError(text string) bool {
	text = strings.TrimSpace(stripStatusCodePrefix(text))
	if text == "" {
		return false
	}
	lower := strings.ToLower(text)
	patterns := []string{
		"real face",
		"real human face",
		"human face",
		"human faces",
		"faces in reference",
		"face in reference",
		"face detected in reference",
		"realistic human",
		"real person",
		"identifiable person",
		"public figure",
		"celebrity likeness",
		"likeness of",
		"deepfake",
		"reference image rejected",
		"真实人脸",
		"真人脸",
		"含有人脸",
		"包含人脸",
		"参考图含真人",
		"参考素材含真人",
		"真人素材",
		"真人参考",
	}
	return containsAny(lower, text, patterns...)
}

func IsContentPolicyViolation(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	lower := strings.ToLower(text)

	patterns := []string{
		"content moderation",
		"content policy",
		"content_policy",
		"content_policy_violation",
		"content_moderation_failed",
		"appear to be unsafe",
		"provided prompt is considered unsafe",
		"cannot be used to generate content",
		"prompt_unsafe",
		"unsafe content",
		"policy violation",
		"sensitive_words",
		"sensitive words detected",
		"sexualization",
		"sexualized",
		"erotic focus",
		"erotic",
		"exposed cleavage",
		"prompt_blocked",
		"blocked by the upstream safety policy",
		"upstream safety policy",
		"model output was blocked",
		"generated video rejected by content moderation",
		"the generated images appear to be unsafe",
		"modifying the prompts or the seeds",
		"unexpected end of json input",
		"invalid character",
		"looking for beginning of value",
		"parse image json",
		"图片内容不合规",
		"内容审核",
		"未通过平台内容审核",
		"参考图未通过",
		"平台内容审核",
		"审核失败",
		"可识别真人肖像",
		"敏感内容",
		"内容政策",
		"该提示可能违反了",
		"生成的图片可能违反了",
		"第三方内容相似",
		"裸露",
		"色情",
		"情色",
		"暴力内容",
		"防护限制",
	}

	for _, pattern := range patterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}

	if strings.Contains(text, "非常抱歉") {
		return strings.Contains(text, "内容政策") ||
			strings.Contains(text, "裸露") ||
			strings.Contains(text, "色情") ||
			strings.Contains(text, "情色") ||
			strings.Contains(text, "暴力") ||
			strings.Contains(text, "防护限制") ||
			strings.Contains(text, "第三方")
	}

	return false
}

func IsUpstreamUnavailableError(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	lower := strings.ToLower(stripStatusCodePrefix(text))
	patterns := []string{
		"upstream service temporarily unavailable",
		"upstream is temporarily unavailable",
		"video service is temporarily unavailable",
		"upstream request failed",
		"no capacity available",
		"capacity available for model",
		"model overloaded",
		"bad response status code 502",
		"bad response status code 503",
		"bad response status code 504",
		"connection reset by peer",
		"connection refused",
		"download image failed",
		"rehost upstream image url",
		"no active tokens available",
		"all available tokens are invalid or expired",
	}
	if containsAny(lower, text, patterns...) {
		return true
	}
	if strings.HasPrefix(text, "status_code=502") ||
		strings.HasPrefix(text, "status_code=503") ||
		strings.HasPrefix(text, "status_code=504") {
		return true
	}
	return false
}

func IsTimeoutError(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	lower := strings.ToLower(stripStatusCodePrefix(text))
	if containsAny(lower, text, "proxy read timeout", "timed out", "timeout", "任务超时", "生图超时", "do request failed", "upstream error: do request failed", "context deadline exceeded", "client.timeout", "video generation timeout") {
		return true
	}
	return strings.HasPrefix(text, "status_code=524")
}

func IsMissingReferenceError(text string) bool {
	text = strings.TrimSpace(stripStatusCodePrefix(text))
	if text == "" {
		return false
	}
	lower := strings.ToLower(text)
	if strings.Contains(text, "图片1") &&
		(strings.Contains(text, "上传") || strings.Contains(text, "没有看到") || strings.Contains(text, "还没有看到") || strings.Contains(text, "引用")) {
		return true
	}
	if strings.Contains(text, "参考图") &&
		(strings.Contains(text, "上传") || strings.Contains(text, "还没有看到") || strings.Contains(text, "没有看到")) {
		return true
	}
	if strings.Contains(lower, "reference image") &&
		(strings.Contains(lower, "upload") || strings.Contains(lower, "don't have") || strings.Contains(lower, "do not have")) {
		return true
	}
	if strings.Contains(lower, "image is required") {
		return true
	}
	return false
}

func IsReferenceMaterialError(text string) bool {
	lower := strings.ToLower(stripStatusCodePrefix(text))
	patterns := []string{
		"failed to fetch image_url",
		"failed to fetch audio_url",
		"failed to fetch reference_video",
		"only http/https or data url",
		"image_url is empty",
		"audio_url is empty",
		"reference_video is empty",
		"invalid image for video",
		"invalid data url image",
		"invalid base64 image",
		"mask could not be loaded",
		"invalid mask or input image",
		"reference material could not be processed",
	}
	return containsAny(lower, text, patterns...)
}

func IsGenericInvalidRequestError(text string) bool {
	lower := strings.ToLower(stripStatusCodePrefix(text))
	patterns := []string{
		"unsupported parameters for video model",
		"entity not found",
		"entity name is ambiguous",
		"entity has no",
		"mask is only supported",
		"mask requires an input image",
		"mask must be",
		"mask and input image",
		"unsupported image type",
		"model and prompt are required",
		"prompt is required",
		"duration must be an integer",
		"duration is required",
		"requests must use application/json",
		"request parameters are invalid",
	}
	return containsAny(lower, text, patterns...)
}
