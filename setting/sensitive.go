package setting

import (
	"strconv"
	"strings"
)

var CheckSensitiveEnabled = true
var CheckSensitiveOnPromptEnabled = true

// LocalSensitivePromptBlockEnabled 本地敏感词前置拦截（仅生图）：关闭后不检查词表、直接转发上游。
var LocalSensitivePromptBlockEnabled = true

// SensitiveReviewWhitelistUserIds 审查白名单用户（仅生图）：跳过本地词表前置拦截；生图上游内容审查拒绝时仍扣费。
var SensitiveReviewWhitelistUserIds = map[int]struct{}{}

// SensitivePromptScope 本地词表前置拦截的作用域。
// 文本/视频不做本地拦截，审查交给上游；仅生图走本地词表 + 审查白名单。
type SensitivePromptScope int

const (
	SensitivePromptScopeImage SensitivePromptScope = iota
)

//var CheckSensitiveOnCompletionEnabled = true

// StopOnSensitiveEnabled 如果检测到敏感词，是否立刻停止生成，否则替换敏感词
var StopOnSensitiveEnabled = true

// StreamCacheQueueLength 流模式缓存队列长度，0表示无缓存
var StreamCacheQueueLength = 0

// SensitiveWords 敏感词
// var SensitiveWords []string
var SensitiveWords = []string{
	"test_sensitive",
}

func SensitiveWordsToString() string {
	return strings.Join(SensitiveWords, "\n")
}

func SensitiveWordsFromString(s string) {
	SensitiveWords = []string{}
	sw := strings.Split(s, "\n")
	for _, w := range sw {
		w = strings.TrimSpace(w)
		if w != "" {
			SensitiveWords = append(SensitiveWords, w)
		}
	}
}

func SensitiveReviewWhitelistToString() string {
	if len(SensitiveReviewWhitelistUserIds) == 0 {
		return ""
	}
	ids := make([]string, 0, len(SensitiveReviewWhitelistUserIds))
	for id := range SensitiveReviewWhitelistUserIds {
		ids = append(ids, strconv.Itoa(id))
	}
	return strings.Join(ids, "\n")
}

func SensitiveReviewWhitelistFromString(s string) {
	SensitiveReviewWhitelistUserIds = map[int]struct{}{}
	for _, part := range strings.FieldsFunc(s, func(r rune) bool {
		return r == '\n' || r == ',' || r == ' ' || r == '\t'
	}) {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.Atoi(part)
		if err != nil || id <= 0 {
			continue
		}
		SensitiveReviewWhitelistUserIds[id] = struct{}{}
	}
}

func IsSensitiveReviewWhitelistUser(userId int) bool {
	if userId <= 0 {
		return false
	}
	_, ok := SensitiveReviewWhitelistUserIds[userId]
	return ok
}

func promptSensitiveBaseEnabled() bool {
	return CheckSensitiveEnabled && CheckSensitiveOnPromptEnabled
}

func ShouldCheckPromptSensitive(scope SensitivePromptScope) bool {
	return ShouldCheckPromptSensitiveForUser(0, scope)
}

// ShouldCheckPromptSensitiveForUser 白名单用户跳过本地词表拦截（仅 SensitivePromptScopeImage）。
func ShouldCheckPromptSensitiveForUser(userId int, scope SensitivePromptScope) bool {
	if scope != SensitivePromptScopeImage {
		return false
	}
	if !promptSensitiveBaseEnabled() {
		return false
	}
	if IsSensitiveReviewWhitelistUser(userId) {
		return false
	}
	return LocalSensitivePromptBlockEnabled
}

//func ShouldCheckCompletionSensitive() bool {
//	return CheckSensitiveEnabled && CheckSensitiveOnCompletionEnabled
//}
