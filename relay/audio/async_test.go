package audio

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

func TestIsAsyncRequestDefaultsTrue(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"gemini-music","prompt":"upbeat BGM"}`)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/audio/generations", nil)
	c.Request.Header.Set("Content-Type", "application/json")
	storage, err := common.CreateBodyStorage(body)
	if err != nil {
		t.Fatalf("CreateBodyStorage: %v", err)
	}
	c.Set(common.KeyBodyStorage, storage)
	if !IsAsyncRequest(c) {
		t.Fatal("expected default async=true when field omitted")
	}
}

func TestIsAsyncRequestExplicitFalse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"gemini-music","prompt":"upbeat BGM","async":false}`)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/audio/generations", nil)
	c.Request.Header.Set("Content-Type", "application/json")
	storage, err := common.CreateBodyStorage(body)
	if err != nil {
		t.Fatalf("CreateBodyStorage: %v", err)
	}
	c.Set(common.KeyBodyStorage, storage)
	if IsAsyncRequest(c) {
		t.Fatal("expected async=false to disable task enqueue")
	}
}

func TestNormalizeAsyncGenerationBody(t *testing.T) {
	out := normalizeAsyncGenerationBody([]byte(`{"model":"gemini-music","prompt":"x","async":true,"stream":true}`))
	var raw map[string]any
	if err := common.Unmarshal(out, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := raw["async"]; ok {
		t.Fatal("async should be stripped for worker replay")
	}
	if raw["response_format"] != "url" {
		t.Fatalf("response_format = %#v, want url", raw["response_format"])
	}
	if raw["stream"] != false {
		t.Fatalf("stream = %#v, want false", raw["stream"])
	}
}
