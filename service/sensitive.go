package service

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

func CheckSensitiveMessages(messages []dto.Message) ([]string, error) {
	if len(messages) == 0 {
		return nil, nil
	}

	for _, message := range messages {
		arrayContent := message.ParseContent()
		for _, m := range arrayContent {
			if m.Type == "image_url" {
				// TODO: check image url
				continue
			}
			// 检查 text 是否为空
			if m.Text == "" {
				continue
			}
			if ok, words := SensitiveWordContains(m.Text); ok {
				return words, errors.New("sensitive words detected")
			}
		}
	}
	return nil, nil
}

func CheckSensitiveText(text string) (bool, []string) {
	return SensitiveWordContains(text)
}

// PromptSensitiveRejection 本地敏感词前置拦截（仅生图）：命中则直接拒绝，不转发上游、不预扣费。
func PromptSensitiveRejection(c *gin.Context, text string, scope setting.SensitivePromptScope) (bool, *types.NewAPIError) {
	userId := 0
	if c != nil {
		userId = c.GetInt("id")
	}
	if !setting.ShouldCheckPromptSensitiveForUser(userId, scope) {
		return false, nil
	}
	contains, words := CheckSensitiveText(text)
	if !contains {
		return false, nil
	}
	if c != nil {
		logger.LogWarn(c, fmt.Sprintf("user sensitive words detected: %s", strings.Join(words, ", ")))
	}
	return true, types.NewErrorWithStatusCode(
		fmt.Errorf("%s", ContentPolicyMessage(c)),
		types.ErrorCodeSensitiveWordsDetected,
		http.StatusBadRequest,
	)
}

// TaskErrorIfSensitivePrompt 生图任务提交前的敏感词拦截。
func TaskErrorIfSensitivePrompt(c *gin.Context, text string) *dto.TaskError {
	if rejected, apiErr := PromptSensitiveRejection(c, text, setting.SensitivePromptScopeImage); rejected {
		taskErr := TaskErrorFromAPIError(apiErr)
		taskErr.LocalError = true
		return taskErr
	}
	return nil
}

// SensitiveWordContains 是否包含敏感词，返回是否包含敏感词和敏感词列表
func SensitiveWordContains(text string) (bool, []string) {
	if len(setting.SensitiveWords) == 0 {
		return false, nil
	}
	if len(text) == 0 {
		return false, nil
	}
	checkText := strings.ToLower(text)
	return AcSearch(checkText, setting.SensitiveWords, true)
}

// SensitiveWordReplace 敏感词替换，返回是否包含敏感词和替换后的文本
func SensitiveWordReplace(text string, returnImmediately bool) (bool, []string, string) {
	if len(setting.SensitiveWords) == 0 {
		return false, nil, text
	}
	checkText := strings.ToLower(text)
	m := getOrBuildAC(setting.SensitiveWords)
	hits := m.MultiPatternSearch([]rune(checkText), returnImmediately)
	if len(hits) > 0 {
		words := make([]string, 0, len(hits))
		var builder strings.Builder
		builder.Grow(len(text))
		lastPos := 0

		for _, hit := range hits {
			pos := hit.Pos
			word := string(hit.Word)
			builder.WriteString(text[lastPos:pos])
			builder.WriteString("**###**")
			lastPos = pos + len(word)
			words = append(words, word)
		}
		builder.WriteString(text[lastPos:])
		return true, words, builder.String()
	}
	return false, nil, text
}
