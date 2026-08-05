package service

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
)

func TestPromptSensitiveRejection_ImageScopeOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	prevEnabled := setting.CheckSensitiveEnabled
	prevPrompt := setting.CheckSensitiveOnPromptEnabled
	prevLocalBlock := setting.LocalSensitivePromptBlockEnabled
	prevWords := append([]string(nil), setting.SensitiveWords...)
	t.Cleanup(func() {
		setting.CheckSensitiveEnabled = prevEnabled
		setting.CheckSensitiveOnPromptEnabled = prevPrompt
		setting.LocalSensitivePromptBlockEnabled = prevLocalBlock
		setting.SensitiveWords = prevWords
	})

	setting.CheckSensitiveEnabled = true
	setting.CheckSensitiveOnPromptEnabled = true
	setting.LocalSensitivePromptBlockEnabled = true
	setting.SensitiveWords = []string{"雷管炸弹", "换脸"}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/v1/images/generations", nil)

	rejected, apiErr := PromptSensitiveRejection(c, "生成圆柱体雷管炸弹", setting.SensitivePromptScopeImage)
	if !rejected || apiErr == nil {
		t.Fatalf("expected sensitive rejection for image scope")
	}
	if got := apiErr.Error(); got != ContentPolicyMessageEN {
		t.Fatalf("PromptSensitiveRejection() = %q, want %q", got, ContentPolicyMessageEN)
	}

	rejected, apiErr = PromptSensitiveRejection(c, "生成一只鸟异兽", setting.SensitivePromptScopeImage)
	if rejected || apiErr != nil {
		t.Fatalf("expected no rejection for safe prompt")
	}
}

func TestPromptSensitiveRejection_TextScopeSkipsLocalBlock(t *testing.T) {
	gin.SetMode(gin.TestMode)
	prevEnabled := setting.CheckSensitiveEnabled
	prevPrompt := setting.CheckSensitiveOnPromptEnabled
	prevLocalBlock := setting.LocalSensitivePromptBlockEnabled
	prevWords := append([]string(nil), setting.SensitiveWords...)
	t.Cleanup(func() {
		setting.CheckSensitiveEnabled = prevEnabled
		setting.CheckSensitiveOnPromptEnabled = prevPrompt
		setting.LocalSensitivePromptBlockEnabled = prevLocalBlock
		setting.SensitiveWords = prevWords
	})

	setting.CheckSensitiveEnabled = true
	setting.CheckSensitiveOnPromptEnabled = true
	setting.LocalSensitivePromptBlockEnabled = true
	setting.SensitiveWords = []string{"雷管炸弹"}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/v1/chat/completions", nil)

	rejected, apiErr := PromptSensitiveRejection(c, "生成圆柱体雷管炸弹", setting.SensitivePromptScope(-1))
	if rejected || apiErr != nil {
		t.Fatal("text/video paths must not be locally blocked; upstream handles review")
	}
}

func TestTaskErrorIfSensitivePrompt(t *testing.T) {
	gin.SetMode(gin.TestMode)
	prevEnabled := setting.CheckSensitiveEnabled
	prevPrompt := setting.CheckSensitiveOnPromptEnabled
	prevLocalBlock := setting.LocalSensitivePromptBlockEnabled
	prevWords := append([]string(nil), setting.SensitiveWords...)
	t.Cleanup(func() {
		setting.CheckSensitiveEnabled = prevEnabled
		setting.CheckSensitiveOnPromptEnabled = prevPrompt
		setting.LocalSensitivePromptBlockEnabled = prevLocalBlock
		setting.SensitiveWords = prevWords
	})

	setting.CheckSensitiveEnabled = true
	setting.CheckSensitiveOnPromptEnabled = true
	setting.LocalSensitivePromptBlockEnabled = true
	setting.SensitiveWords = []string{"锁定人脸"}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/v1/images/generations", nil)
	c.Request.Header.Set("X-Cangyuan-Client", "infinite-canvas")

	taskErr := TaskErrorIfSensitivePrompt(c, "需要锁定人脸信息")
	if taskErr == nil {
		t.Fatal("expected task error")
	}
	if !taskErr.LocalError {
		t.Fatal("expected local task error")
	}
	if taskErr.Message != ContentPolicyMessageZH {
		t.Fatalf("TaskErrorIfSensitivePrompt() = %q, want %q", taskErr.Message, ContentPolicyMessageZH)
	}
}

func TestPromptSensitiveRejection_DisabledByLocalBlockSwitch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	prevLocalBlock := setting.LocalSensitivePromptBlockEnabled
	prevWords := append([]string(nil), setting.SensitiveWords...)
	prevWhitelist := setting.SensitiveReviewWhitelistUserIds
	t.Cleanup(func() {
		setting.LocalSensitivePromptBlockEnabled = prevLocalBlock
		setting.SensitiveWords = prevWords
		setting.SensitiveReviewWhitelistUserIds = prevWhitelist
	})

	setting.LocalSensitivePromptBlockEnabled = false
	setting.SensitiveReviewWhitelistUserIds = map[int]struct{}{}
	setting.SensitiveWords = []string{"雷管炸弹"}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	rejected, apiErr := PromptSensitiveRejection(c, "生成圆柱体雷管炸弹", setting.SensitivePromptScopeImage)
	if rejected || apiErr != nil {
		t.Fatalf("expected no local rejection when LocalSensitivePromptBlockEnabled=false")
	}
}

func TestPromptSensitiveRejection_WhitelistSkipsLocalBlock(t *testing.T) {
	gin.SetMode(gin.TestMode)
	prevGlobal := setting.LocalSensitivePromptBlockEnabled
	prevEnabled := setting.CheckSensitiveEnabled
	prevPrompt := setting.CheckSensitiveOnPromptEnabled
	prevWhitelist := setting.SensitiveReviewWhitelistUserIds
	prevWords := append([]string(nil), setting.SensitiveWords...)
	t.Cleanup(func() {
		setting.LocalSensitivePromptBlockEnabled = prevGlobal
		setting.CheckSensitiveEnabled = prevEnabled
		setting.CheckSensitiveOnPromptEnabled = prevPrompt
		setting.SensitiveReviewWhitelistUserIds = prevWhitelist
		setting.SensitiveWords = prevWords
	})

	setting.LocalSensitivePromptBlockEnabled = true
	setting.CheckSensitiveEnabled = true
	setting.CheckSensitiveOnPromptEnabled = true
	setting.SensitiveReviewWhitelistUserIds = map[int]struct{}{7: {}}
	setting.SensitiveWords = []string{"雷管炸弹"}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/v1/images/generations", nil)
	c.Set("id", 7)

	rejected, apiErr := PromptSensitiveRejection(c, "生成圆柱体雷管炸弹", setting.SensitivePromptScopeImage)
	if rejected || apiErr != nil {
		t.Fatal("expected whitelist user to bypass local sensitive check for image")
	}
}
