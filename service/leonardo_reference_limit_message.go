package service

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	leonardoReferenceImagesLimitRe = regexp.MustCompile(`(?i)reference images exceed leonardo limit \((\d+)/(\d+)\)`)
	leonardoReferenceVideosLimitRe = regexp.MustCompile(`(?i)reference videos exceed leonardo limit \((\d+)/(\d+)\)`)
	leonardoReferenceAudiosLimitRe = regexp.MustCompile(`(?i)reference audios exceed leonardo limit \((\d+)/(\d+)\)`)
	leonardoReferenceVideosTotalDurationRe = regexp.MustCompile(`(?i)reference videos total duration ([0-9.]+)s exceeds leonardo limit \((\d+) s\)`)
	leonardoReferenceAudioDurationRe = regexp.MustCompile(`(?i)reference audio duration ([0-9.]+)s exceeds leonardo limit \((\d+) s\)`)
	leonardoReferenceVideoDurationRangeRe = regexp.MustCompile(`(?i)reference video duration ([0-9.]+)s not in (\d+)-(\d+) s range`)
)

// HumanizeLeonardoReferenceLimitError maps Leonardo multimodal limit errors to client-friendly copy.
func HumanizeLeonardoReferenceLimitError(preferChinese bool, raw string) (string, bool) {
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

func humanizeLeonardoGenerationFailureDetail(preferChinese bool, detail string) (string, bool) {
	lower := strings.ToLower(strings.TrimSpace(detail))
	if strings.Contains(lower, "upstream rejected the job with no output") {
		if preferChinese {
			return "Leonardo 上游拒绝了该任务（无任何输出），建议缩短提示词或减少参考素材后重试。", true
		}
		return "Leonardo rejected the job with no output. Try a shorter prompt or fewer references.", true
	}
	return "", false
}
