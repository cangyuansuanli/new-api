package clienterror

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Leonardo / Seedance cy-sd4 pool. Upstream: leonardo-web2api/
// Client-facing copy for cy-sd4 is produced by leonardo-web2api; NewAPI passes through.

var (
	leonardoReferenceImagesLimitRe         = regexp.MustCompile(`(?i)reference images exceed leonardo limit \((\d+)/(\d+)\)`)
	leonardoReferenceVideosLimitRe         = regexp.MustCompile(`(?i)reference videos exceed leonardo limit \((\d+)/(\d+)\)`)
	leonardoReferenceAudiosLimitRe         = regexp.MustCompile(`(?i)reference audios exceed leonardo limit \((\d+)/(\d+)\)`)
	leonardoReferenceVideosTotalDurationRe = regexp.MustCompile(`(?i)reference videos total duration ([0-9.]+)s exceeds leonardo limit \((\d+) s\)`)
	leonardoReferenceAudioDurationRe       = regexp.MustCompile(`(?i)reference audio duration ([0-9.]+)s exceeds leonardo limit \((\d+) s\)`)
	leonardoReferenceVideoDurationRangeRe  = regexp.MustCompile(`(?i)reference video duration ([0-9.]+)s not in (\d+)-(\d+) s range`)
	cookieFailurePartRe                    = regexp.MustCompile(`(?i)cookie#(\d+):\s*([^|]+)`)
	insufficientCreditsDetailRe            = regexp.MustCompile(`(?i)insufficient credits \(need (\d+), have (\d+)\)`)
)

func normalizeLeonardo(preferChinese bool, raw string) (string, bool) {
	if strings.Contains(raw, PoolDepletedMessageZH) {
		return localized(preferChinese, PoolDepletedMessageZH, PoolDepletedMessageEN), true
	}
	if msg, ok := humanizeLeonardoCookiePoolFailure(preferChinese, raw); ok {
		return msg, true
	}
	if IsLeonardoInsufficientCreditsForJobError(raw) {
		return localized(preferChinese, InsufficientCreditsForJobMessageZH, InsufficientCreditsForJobMessageEN), true
	}
	if IsLeonardoPoolDepletedError(raw) {
		return localized(preferChinese, PoolDepletedMessageZH, PoolDepletedMessageEN), true
	}
	if IsLeonardoReferenceDurationTooLongError(raw) {
		return localized(preferChinese, ReferenceDurationTooLongZH, ReferenceDurationTooLongEN), true
	}
	if IsLeonardoPoolReferenceMaterialError(raw) {
		return localized(preferChinese, ReferenceMaterialMessageZH, ReferenceMaterialMessageEN), true
	}
	if msg, ok := humanizeLeonardoReferenceLimitError(preferChinese, raw); ok {
		return msg, true
	}
	if IsLeonardoPoolInvalidRequestError(raw) {
		return localized(preferChinese, InvalidRequestMessageZH, InvalidRequestMessageEN), true
	}
	if msg, ok := humanizeLeonardoPoolCapacityError(preferChinese, raw); ok {
		return msg, true
	}
	if isGenerationFailureWithoutDetail(raw) {
		return localized(preferChinese, GenerationFailedNoDetailZH, GenerationFailedNoDetailEN), true
	}
	if detail, noDetail := extractLeonardoUpstreamFailureDetail(raw); noDetail {
		return localized(preferChinese, GenerationFailedNoDetailZH, GenerationFailedNoDetailEN), true
	} else if detail != "" {
		if IsRealFaceReferenceError(detail) {
			return localized(preferChinese, ReferenceRealFaceMessageZH, ReferenceRealFaceMessageEN), true
		}
		if IsContentPolicyViolation(detail) {
			return localized(preferChinese, ContentPolicyMessageZH, ContentPolicyMessageEN), true
		}
		if msg, ok := humanizeUpstreamUnavailableError(preferChinese, detail); ok {
			return msg, true
		}
		return localized(preferChinese, GenerationFailedMessageZH, GenerationFailedMessageEN), true
	}
	if IsLeonardoPoolGenerationFailed(raw) {
		return localized(preferChinese, GenerationFailedMessageZH, GenerationFailedMessageEN), true
	}
	return "", false
}

func humanizeLeonardoReferenceLimitError(preferChinese bool, raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}

	if m := leonardoReferenceImagesLimitRe.FindStringSubmatch(raw); len(m) == 3 {
		current, _ := strconv.Atoi(m[1])
		max, _ := strconv.Atoi(m[2])
		if preferChinese {
			return fmt.Sprintf("参考图最多 %d 张，当前 %d 张，请减少后重试。", max, current), true
		}
		return fmt.Sprintf("At most %d reference images allowed; you provided %d. Please remove extras and retry.", max, current), true
	}
	if m := leonardoReferenceVideosLimitRe.FindStringSubmatch(raw); len(m) == 3 {
		current, _ := strconv.Atoi(m[1])
		max, _ := strconv.Atoi(m[2])
		if preferChinese {
			return fmt.Sprintf("参考视频最多 %d 段，当前 %d 段，请减少后重试。", max, current), true
		}
		return fmt.Sprintf("At most %d reference videos allowed; you provided %d. Please remove extras and retry.", max, current), true
	}
	if m := leonardoReferenceAudiosLimitRe.FindStringSubmatch(raw); len(m) == 3 {
		current, _ := strconv.Atoi(m[1])
		max, _ := strconv.Atoi(m[2])
		if preferChinese {
			return fmt.Sprintf("参考音频最多 %d 段，当前 %d 段，请减少后重试。", max, current), true
		}
		return fmt.Sprintf("At most %d reference audio clip allowed; you provided %d. Please remove extras and retry.", max, current), true
	}
	if m := leonardoReferenceVideosTotalDurationRe.FindStringSubmatch(raw); len(m) == 3 {
		current, _ := strconv.ParseFloat(m[1], 64)
		max, _ := strconv.Atoi(m[2])
		if preferChinese {
			return fmt.Sprintf("参考视频总时长最多 %d 秒，当前 %.1f 秒，请缩短后重试。", max, current), true
		}
		return fmt.Sprintf("Total reference video duration must be at most %ds; yours is %.1fs. Please shorten and retry.", max, current), true
	}
	if m := leonardoReferenceAudioDurationRe.FindStringSubmatch(raw); len(m) == 3 {
		current, _ := strconv.ParseFloat(m[1], 64)
		max, _ := strconv.Atoi(m[2])
		if preferChinese {
			return fmt.Sprintf("参考音频时长最多 %d 秒，当前 %.1f 秒，请缩短后重试。", max, current), true
		}
		return fmt.Sprintf("Reference audio must be at most %ds; yours is %.1fs. Please shorten and retry.", max, current), true
	}
	if m := leonardoReferenceVideoDurationRangeRe.FindStringSubmatch(raw); len(m) == 4 {
		current, _ := strconv.ParseFloat(m[1], 64)
		minSec, _ := strconv.Atoi(m[2])
		maxSec, _ := strconv.Atoi(m[3])
		if preferChinese {
			return fmt.Sprintf("单条参考视频时长须在 %d–%d 秒之间，当前 %.1f 秒，请调整后重试。", minSec, maxSec, current), true
		}
		return fmt.Sprintf("Each reference video must be %d–%ds; one clip is %.1fs. Please adjust and retry.", minSec, maxSec, current), true
	}

	lower := strings.ToLower(raw)
	if strings.Contains(lower, "multimodal references cannot be combined with start/end frame inputs") {
		if preferChinese {
			return "多模态参考（图/视频/音频）与首尾帧不能同时使用，请只保留一种方式。", true
		}
		return "Multimodal references (images/videos/audio) cannot be combined with start/end frames. Use one mode only.", true
	}
	if strings.Contains(lower, "multimodal video/audio references require at least one reference image") {
		if preferChinese {
			return "使用参考视频或音频时，至少需要 1 张参考图。", true
		}
		return "At least one reference image is required when using reference video or audio.", true
	}

	return "", false
}

func IsLeonardoPoolReferenceMaterialError(text string) bool {
	lower := strings.ToLower(stripStatusCodePrefix(text))
	patterns := []string{
		"leonardo: download",
		"upload reference",
		"upload start_frame",
		"upload end_frame",
		"uploadaudio",
		"uploadvideo",
		"leonardo: uploadimage:",
		"originalfilename",
		"media upload failed",
		"uploaded media processing timeout",
	}
	return containsAny(lower, text, patterns...)
}

func IsLeonardoReferenceDurationTooLongError(text string) bool {
	lower := strings.ToLower(stripStatusCodePrefix(text))
	return strings.Contains(lower, "duration_too_long") ||
		strings.Contains(lower, "reference video or audio exceeds the model's duration limit")
}

func IsLeonardoPoolInvalidRequestError(text string) bool {
	lower := strings.ToLower(stripStatusCodePrefix(text))
	patterns := []string{
		"reference video duration",
		"reference audio duration",
		"exceed leonardo limit",
		"requires start_frame",
		"multimodal references cannot",
		"unsupported aspect_ratio",
		"is not supported by",
	}
	return containsAny(lower, text, patterns...)
}

func IsLeonardoInsufficientCreditsForJobError(text string) bool {
	lower := strings.ToLower(stripStatusCodePrefix(text))
	return strings.Contains(lower, "insufficient credits")
}

func IsLeonardoPoolDepletedError(text string) bool {
	lower := strings.ToLower(stripStatusCodePrefix(text))
	patterns := []string{
		"no active cookie",
		"depleted (auto-disabled)",
		"token balance is empty",
	}
	return containsAny(lower, text, patterns...)
}

func IsLeonardoPoolCapacityError(text string) bool {
	lower := strings.ToLower(stripStatusCodePrefix(text))
	patterns := []string{
		"dynamic proxy fetch failed",
		"auth_expired",
		"failed to fetch token",
		"failed to resolve token",
		"model overloaded",
		"busy (max in-flight)",
		"cooldown (generation recently failed)",
	}
	return containsAny(lower, text, patterns...)
}

func IsLeonardoPoolGenerationFailed(text string) bool {
	text = strings.TrimSpace(stripStatusCodePrefix(text))
	if text == "" {
		return false
	}
	lower := strings.ToLower(text)
	return strings.Contains(lower, "all cookies failed") ||
		strings.Contains(lower, "video generation failed") ||
		strings.Contains(lower, "leonardo: video generation failed") ||
		strings.Contains(lower, "leonardo: generation failed") ||
		strings.Contains(lower, "generation_failed")
}

func extractLeonardoUpstreamFailureDetail(raw string) (detail string, noDetail bool) {
	lower := strings.ToLower(strings.TrimSpace(raw))
	if !strings.Contains(lower, "leonardo:") || !strings.Contains(lower, "generation failed") {
		return "", false
	}
	if strings.Contains(lower, "upstream returned no detail") {
		return "", true
	}
	if strings.Contains(lower, "no output and no failure detail") {
		return "", true
	}
	if idx := strings.Index(raw, "):"); idx >= 0 {
		if d := strings.TrimSpace(raw[idx+2:]); d != "" {
			return d, false
		}
	}
	return "", false
}

func isGenerationFailureWithoutDetail(raw string) bool {
	lower := strings.ToLower(strings.TrimSpace(raw))
	return strings.Contains(lower, "upstream returned no detail") ||
		strings.Contains(lower, "without a specific provider reason") ||
		strings.Contains(lower, "upstream rejected the job with no output") ||
		strings.Contains(lower, "no output and no failure detail")
}

func humanizeLeonardoPoolCapacityError(preferChinese bool, raw string) (string, bool) {
	if !IsLeonardoPoolCapacityError(raw) {
		return "", false
	}
	lower := strings.ToLower(stripStatusCodePrefix(raw))

	switch {
	case strings.Contains(lower, "busy (max in-flight)"):
		if preferChinese {
			return "当前并发已满，请稍后重试。", true
		}
		return "Concurrency limit reached. Please retry later.", true
	case strings.Contains(lower, "cooldown (generation recently failed)"):
		if preferChinese {
			return "服务刚失败正在冷却，请稍后重试。", true
		}
		return "Service is in post-failure cooldown. Please retry later.", true
	case strings.Contains(lower, "dynamic proxy fetch failed"):
		if preferChinese {
			return "网络代理获取失败，请稍后重试或联系管理员。", true
		}
		return "Proxy fetch failed. Retry later or contact an administrator.", true
	case strings.Contains(lower, "auth_expired"):
		if preferChinese {
			return "服务鉴权已过期，请联系管理员。", true
		}
		return "Service authentication expired. Please contact an administrator.", true
	case strings.Contains(lower, "failed to fetch token") || strings.Contains(lower, "failed to resolve token"):
		if preferChinese {
			return "服务鉴权失败，请稍后重试或联系管理员。", true
		}
		return "Service authentication failed. Retry later or contact an administrator.", true
	case strings.Contains(lower, "model overloaded"):
		if preferChinese {
			return "模型过载，请稍后重试。", true
		}
		return "The model is overloaded. Please retry later.", true
	}

	return localized(preferChinese, PoolUnavailableMessageZH, PoolUnavailableMessageEN), true
}

func humanizeLeonardoCookiePoolFailure(preferChinese bool, raw string) (string, bool) {
	lower := strings.ToLower(strings.TrimSpace(raw))
	if !strings.Contains(lower, "all cookies failed") {
		return "", false
	}

	parts := parseCookieFailureParts(raw)
	if len(parts) == 0 {
		return localized(preferChinese, PoolUnavailableMessageZH, PoolUnavailableMessageEN), true
	}

	if single, ok := singleCookieFailureCategory(parts); ok {
		switch single {
		case "depleted":
			return localized(preferChinese, PoolDepletedMessageZH, PoolDepletedMessageEN), true
		case "insufficient":
			return localized(preferChinese, InsufficientCreditsForJobMessageZH, InsufficientCreditsForJobMessageEN), true
		case "busy":
			return humanizeLeonardoPoolCapacityError(preferChinese, "busy (max in-flight)")
		case "cooldown":
			return humanizeLeonardoPoolCapacityError(preferChinese, "cooldown (generation recently failed)")
		case "proxy":
			return humanizeLeonardoPoolCapacityError(preferChinese, "dynamic proxy fetch failed")
		case "auth":
			return humanizeLeonardoPoolCapacityError(preferChinese, "auth_expired")
		}
	}

	summary, hasInsufficient := summarizeCookieFailureReasons(preferChinese, parts)
	if preferChinese {
		msg := "视频生成失败：" + summary + "。"
		if hasInsufficient {
			msg += " 若因积分不足，可缩短视频秒数、降低分辨率，或改用 480p/经济档模型后再试。"
		}
		return msg, true
	}
	msg := "Video generation failed: " + summary + "."
	if hasInsufficient {
		msg += " If credits are insufficient, try a shorter duration, lower resolution, or an economy model (e.g. 480p)."
	}
	return msg, true
}

type cookieFailurePart struct {
	id     string
	reason string
}

func parseCookieFailureParts(raw string) []cookieFailurePart {
	matches := cookieFailurePartRe.FindAllStringSubmatch(raw, -1)
	if len(matches) == 0 {
		return nil
	}
	out := make([]cookieFailurePart, 0, len(matches))
	for _, m := range matches {
		if len(m) != 3 {
			continue
		}
		out = append(out, cookieFailurePart{
			id:     strings.TrimSpace(m[1]),
			reason: strings.TrimSpace(m[2]),
		})
	}
	return out
}

func singleCookieFailureCategory(parts []cookieFailurePart) (string, bool) {
	if len(parts) == 0 {
		return "", false
	}
	category := classifyCookieFailureReason(parts[0].reason)
	for _, part := range parts[1:] {
		if classifyCookieFailureReason(part.reason) != category {
			return "", false
		}
	}
	return category, true
}

func classifyCookieFailureReason(reason string) string {
	lower := strings.ToLower(strings.TrimSpace(reason))
	switch {
	case strings.Contains(lower, "depleted (auto-disabled)"), strings.Contains(lower, "token balance is empty"):
		return "depleted"
	case strings.Contains(lower, "insufficient credits"):
		return "insufficient"
	case strings.Contains(lower, "busy (max in-flight)"):
		return "busy"
	case strings.Contains(lower, "cooldown"):
		return "cooldown"
	case strings.Contains(lower, "dynamic proxy"):
		return "proxy"
	case strings.Contains(lower, "auth_expired"), strings.Contains(lower, "failed to fetch token"), strings.Contains(lower, "failed to resolve token"):
		return "auth"
	default:
		return "other"
	}
}

func summarizeCookieFailureReasons(preferChinese bool, parts []cookieFailurePart) (summary string, hasInsufficient bool) {
	seen := make(map[string]struct{})
	labels := make([]string, 0, len(parts))
	for _, part := range parts {
		category := classifyCookieFailureReason(part.reason)
		if category == "insufficient" {
			hasInsufficient = true
		}
		if _, ok := seen[category]; ok {
			continue
		}
		seen[category] = struct{}{}
		if label := cookieFailureCategoryLabel(preferChinese, category, part.reason); label != "" {
			labels = append(labels, label)
		}
	}
	return strings.Join(labels, "；"), hasInsufficient
}

func cookieFailureCategoryLabel(preferChinese bool, category, reason string) string {
	if preferChinese {
		switch category {
		case "depleted":
			return "额度耗尽"
		case "insufficient":
			if detail := formatInsufficientCreditsDetail(preferChinese, reason); detail != "" {
				return "积分不足（" + detail + "）"
			}
			return "积分不足"
		case "busy":
			return "并发已满"
		case "cooldown":
			return "冷却中"
		case "proxy":
			return "网络代理失败"
		case "auth":
			return "鉴权失败"
		default:
			return "生成失败"
		}
	}
	switch category {
	case "depleted":
		return "credits depleted"
	case "insufficient":
		if detail := formatInsufficientCreditsDetail(preferChinese, reason); detail != "" {
			return "insufficient credits (" + detail + ")"
		}
		return "insufficient credits"
	case "busy":
		return "concurrency limit reached"
	case "cooldown":
		return "cooldown active"
	case "proxy":
		return "proxy fetch failed"
	case "auth":
		return "authentication failed"
	default:
		return "generation failed"
	}
}

func formatInsufficientCreditsDetail(preferChinese bool, reason string) string {
	m := insufficientCreditsDetailRe.FindStringSubmatch(reason)
	if len(m) != 3 {
		return ""
	}
	if preferChinese {
		return "需 " + m[1] + " 积分，剩余 " + m[2]
	}
	return "need " + m[1] + ", have " + m[2]
}

// IsLeonardoWeb2APIRelayModel reports NewAPI internal models routed to leonardo-web2api.
func IsLeonardoWeb2APIRelayModel(model string) bool {
	model = strings.ToLower(strings.TrimSpace(model))
	return strings.HasPrefix(model, "cy-sd4-seedance") || strings.HasPrefix(model, "cy-sd4-minimax-h3")
}

func normalizeLeonardoRelay(preferChinese bool, failure ErrorContext) (string, bool) {
	if IsLeonardoWeb2APIRelayModel(failure.Model) {
		return "", false
	}
	return normalizeLeonardo(preferChinese, failure.Raw)
}
