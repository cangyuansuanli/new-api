package seedanceleonardo

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http/httptest"
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
)

func multipartContext(t *testing.T, duration string) *gin.Context {
	return multipartContextWithFields(t, duration, nil)
}

func multipartContextWithFields(t *testing.T, duration string, fields map[string][]string) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("model", "cy-sd4-seedance-2.0")
	_ = writer.WriteField("prompt", "test")
	if duration != "" {
		_ = writer.WriteField("duration", duration)
	}
	for key, values := range fields {
		for _, value := range values {
			_ = writer.WriteField(key, value)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/videos", &body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	return c
}

func TestValidateRequestDoesNotEnforceLeonardoReferenceLimits(t *testing.T) {
	c := multipartContextWithFields(t, "8", map[string][]string{
		"reference_videos": {"v1", "v2", "v3", "v4"},
	})
	info := &relaycommon.RelayInfo{OriginModelName: "cy-sd4-seedance-2.0"}
	if taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(c, info); taskErr != nil {
		t.Fatalf("expected upstream to own reference limits, got: %+v", taskErr)
	}
}

func TestBuildUpstreamBody_CanonicalOnly(t *testing.T) {
	in := map[string]interface{}{
		"prompt": "test",
		"reference_image_urls": []interface{}{
			"https://example.com/a.jpg",
			"https://example.com/b.jpg",
		},
		"generate_audio": true,
	}
	out := buildUpstreamBody(in, "seedance-2.0", 5, []string{
		"https://example.com/a.jpg",
		"https://example.com/b.jpg",
	})
	raw, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(raw) == "" {
		t.Fatal("empty body")
	}
	if out["audio"] != true {
		t.Fatalf("expected audio from generate_audio, got %v", out["audio"])
	}
	refs, ok := out["reference_image_urls"].([]interface{})
	if !ok || len(refs) != 2 {
		t.Fatalf("expected two reference images, got %v", out["reference_image_urls"])
	}
}

func TestBuildUpstreamBody_UsesNormalizedReferenceImages(t *testing.T) {
	out := buildUpstreamBody(map[string]interface{}{
		"prompt": "test",
	}, "seedance-2.0-fast", 8, []string{"https://example.com/reference.jpg"})
	refs, ok := out["reference_image_urls"].([]interface{})
	if !ok || len(refs) != 1 || refs[0] != "https://example.com/reference.jpg" {
		t.Fatalf("expected normalized input to become a reference image, got %v", out["reference_image_urls"])
	}
}

func TestIsRelay(t *testing.T) {
	if !IsRelay("cy-sd4-seedance-2.0") {
		t.Fatal("expected leonardo relay")
	}
	if !IsRelay("cy-sd4-minimax-h3-2k") {
		t.Fatal("expected minimax h3 relay")
	}
	if IsRelay("cy-sd1-seedance-2.0-720p") {
		t.Fatal("cy-sd1 must not match leonardo")
	}
}
