package seedanceleonardo

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
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
	_ = writer.WriteField("model", mini8sModel)
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

func TestValidateRequestRejectsLeonardoReferenceLimits(t *testing.T) {
	tests := []struct {
		name    string
		fields  map[string][]string
		code    string
		message string
	}{
		{
			name: "five images across aliases",
			fields: map[string][]string{
				"image_url":            {"https://example.com/1.jpg"},
				"reference_image_urls": {"https://example.com/2.jpg", "https://example.com/3.jpg", "https://example.com/4.jpg", "https://example.com/5.jpg"},
			},
			code:    "reference_images_limit_exceeded",
			message: "reference images exceed Leonardo limit (5/4)",
		},
		{
			name:    "four videos",
			fields:  map[string][]string{"reference_videos": {"v1", "v2", "v3", "v4"}},
			code:    "reference_videos_limit_exceeded",
			message: "reference videos exceed Leonardo limit (4/3)",
		},
		{
			name:    "two audios",
			fields:  map[string][]string{"reference_audios": {"a1", "a2"}},
			code:    "reference_audios_limit_exceeded",
			message: "reference audios exceed Leonardo limit (2/1)",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := multipartContextWithFields(t, "8", test.fields)
			info := &relaycommon.RelayInfo{OriginModelName: "cy-sd4-seedance-2.0"}
			taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(c, info)
			if taskErr == nil {
				t.Fatal("expected reference limit error")
			}
			if taskErr.StatusCode != 400 || taskErr.Code != test.code || taskErr.Message != test.message {
				t.Fatalf("unexpected task error: %+v", taskErr)
			}
		})
	}
}

func TestValidateJSONRequestRejectsLeonardoReferenceLimits(t *testing.T) {
	tests := []struct {
		name    string
		body    map[string]interface{}
		code    string
		message string
	}{
		{
			name: "five images across public aliases",
			body: map[string]interface{}{
				"image_url":            "i1",
				"images":               []string{"i2", "i3"},
				"reference_image_urls": []string{"i4", "i5"},
			},
			code:    "reference_images_limit_exceeded",
			message: "reference images exceed Leonardo limit (5/4)",
		},
		{
			name:    "four videos",
			body:    map[string]interface{}{"reference_videos": []string{"v1", "v2", "v3", "v4"}},
			code:    "reference_videos_limit_exceeded",
			message: "reference videos exceed Leonardo limit (4/3)",
		},
		{
			name:    "two audios",
			body:    map[string]interface{}{"reference_audios": []string{"a1", "a2"}},
			code:    "reference_audios_limit_exceeded",
			message: "reference audios exceed Leonardo limit (2/1)",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.body["model"] = "cy-sd4-seedance-2.0"
			test.body["prompt"] = "test"
			raw, err := json.Marshal(test.body)
			if err != nil {
				t.Fatal(err)
			}
			gin.SetMode(gin.TestMode)
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", bytes.NewReader(raw))
			c.Request.Header.Set("Content-Type", "application/json")
			info := &relaycommon.RelayInfo{OriginModelName: "cy-sd4-seedance-2.0"}
			taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(c, info)
			if taskErr == nil || taskErr.StatusCode != 400 || taskErr.Code != test.code || taskErr.Message != test.message {
				t.Fatalf("unexpected task error: %+v", taskErr)
			}
		})
	}
}

func TestValidateMultipartJSONArrayReferenceLimits(t *testing.T) {
	c := multipartContextWithFields(t, "8", map[string][]string{
		"reference_videos": {`["v1","v2","v3","v4"]`},
	})
	info := &relaycommon.RelayInfo{OriginModelName: "cy-sd4-seedance-2.0"}
	taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(c, info)
	if taskErr == nil || taskErr.Code != "reference_videos_limit_exceeded" {
		t.Fatalf("unexpected task error: %+v", taskErr)
	}
}

func TestValidateRequestAllowsLeonardoReferenceLimits(t *testing.T) {
	c := multipartContextWithFields(t, "8", map[string][]string{
		"reference_image_urls": {"i1", "i2", "i3", "i4"},
		"reference_videos":     {"v1", "v2", "v3"},
		"reference_audios":     {"a1"},
	})
	info := &relaycommon.RelayInfo{OriginModelName: "cy-sd4-seedance-2.0"}
	if taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(c, info); taskErr != nil {
		t.Fatalf("limits should be accepted: %+v", taskErr)
	}
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
	if IsRelay("cy-sd1-seedance-2.0-720p") {
		t.Fatal("cy-sd1 must not match leonardo")
	}
}
