package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

func TestRelayOpenAIChatCompletionsRejectsAiclubImageModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"cy-ac-gpt-image-2-1k","messages":[]}`))
	c.Request.Header.Set("Content-Type", "application/json")
	defer common.CleanupBodyStorage(c)

	RelayOpenAIChatCompletions(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{"image_model_use_images_api", "/v1/images/generations", "/v1/images/edits", "生图模型"} {
		if !strings.Contains(body, want) {
			t.Fatalf("response missing %q: %s", want, body)
		}
	}
}
