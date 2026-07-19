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
	t.Helper()
	gin.SetMode(gin.TestMode)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("model", mini8sModel)
	_ = writer.WriteField("prompt", "test")
	if duration != "" {
		_ = writer.WriteField("duration", duration)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/videos", &body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	return c
}

func TestValidateRequestRejectsMini8sDurationOverEight(t *testing.T) {
	for _, duration := range []string{"9", "15"} {
		c := multipartContext(t, duration)
		info := &relaycommon.RelayInfo{OriginModelName: mini8sModel}
		if taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(c, info); taskErr == nil {
			t.Fatalf("expected duration %s to be rejected", duration)
		}
	}
}

func TestValidateRequestAcceptsMini8sDurationAtMostEight(t *testing.T) {
	for _, duration := range []string{"", "4", "8"} {
		c := multipartContext(t, duration)
		info := &relaycommon.RelayInfo{OriginModelName: mini8sModel}
		if taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(c, info); taskErr != nil {
			t.Fatalf("duration %s should be accepted: %v", duration, taskErr)
		}
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
	out := buildUpstreamBody(in, "seedance-2.0", 5)
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

func TestIsRelay(t *testing.T) {
	if !IsRelay("cy-sd4-seedance-2.0") {
		t.Fatal("expected leonardo relay")
	}
	if IsRelay("cy-sd1-seedance-2.0-720p") {
		t.Fatal("cy-sd1 must not match leonardo")
	}
}
