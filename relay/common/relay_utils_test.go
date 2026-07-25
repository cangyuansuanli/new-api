package common

import (
	"bytes"
	"mime/multipart"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/gin-gonic/gin"
)

func TestValidateMultipartDirectStoresNormalizedRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("model", "adobe-sora2")
	_ = writer.WriteField("prompt", "a city at night")
	_ = writer.WriteField("seconds", "8")
	_ = writer.WriteField("size", "1280x720")
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/videos", &body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	info := &RelayInfo{}
	if taskErr := ValidateMultipartDirect(c, info); taskErr != nil {
		t.Fatalf("ValidateMultipartDirect: %v", taskErr)
	}
	req, err := GetTaskRequest(c)
	if err != nil {
		t.Fatalf("GetTaskRequest: %v", err)
	}
	if req.Model != "adobe-sora2" || req.Prompt != "a city at night" || req.Duration != 8 {
		t.Fatalf("normalized request = %#v", req)
	}
	if req.Seconds != "8" {
		t.Fatalf("seconds = %q, want normalized alias 8", req.Seconds)
	}
}

func TestValidateMultipartDirectReadsDurationField(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("model", "cy-sd1-seedance-2.0-4k")
	_ = writer.WriteField("prompt", "fifteen second video")
	_ = writer.WriteField("duration", "15")
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/videos", &body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	if taskErr := ValidateMultipartDirect(c, &RelayInfo{}); taskErr != nil {
		t.Fatalf("ValidateMultipartDirect: %v", taskErr)
	}
	req, err := GetTaskRequest(c)
	if err != nil {
		t.Fatalf("GetTaskRequest: %v", err)
	}
	if req.Duration != 15 {
		t.Fatalf("duration = %d, want 15", req.Duration)
	}
	if req.Seconds != "15" {
		t.Fatalf("seconds = %q, want normalized alias 15", req.Seconds)
	}
}

func TestValidateMultipartDirectNormalizesImageURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("model", "seedance-2.0-fast")
	_ = writer.WriteField("prompt", "test")
	_ = writer.WriteField("image_url", "https://example.com/ref.jpg")
	_ = writer.Close()

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/videos", &body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	if taskErr := ValidateMultipartDirect(c, &RelayInfo{}); taskErr != nil {
		t.Fatalf("ValidateMultipartDirect: %v", taskErr)
	}
	req, err := GetTaskRequest(c)
	if err != nil {
		t.Fatalf("GetTaskRequest: %v", err)
	}
	if len(req.Images) != 1 || req.Images[0] != "https://example.com/ref.jpg" {
		t.Fatalf("normalized images = %#v", req.Images)
	}
}

func TestValidateMultipartDirectRejectsConflictingDurationAliases(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/videos", bytes.NewBufferString(`{"model":"grok-video","prompt":"test","duration":15,"seconds":4}`))
	c.Request.Header.Set("Content-Type", "application/json")

	taskErr := ValidateMultipartDirect(c, &RelayInfo{})
	if taskErr == nil {
		t.Fatal("expected conflicting duration aliases to be rejected")
	}
	if taskErr.Code != "invalid_duration" {
		t.Fatalf("error code = %q, want invalid_duration", taskErr.Code)
	}
}

func TestValidateMultipartDirectDetectsArrayReferenceFiles(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("model", "grok-video")
	_ = writer.WriteField("prompt", "test")
	file, _ := writer.CreateFormFile("input_reference[]", "reference.png")
	_, _ = file.Write([]byte("png"))
	_ = writer.Close()

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/videos", &body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	info := &RelayInfo{}
	if taskErr := ValidateMultipartDirect(c, info); taskErr != nil {
		t.Fatalf("unexpected error: %v", taskErr)
	}
	if info.Action != constant.TaskActionGenerate {
		t.Fatalf("action = %q, want generate", info.Action)
	}
}
