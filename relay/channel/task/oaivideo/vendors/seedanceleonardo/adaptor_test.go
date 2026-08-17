package seedanceleonardo

import (
	"bytes"
	"encoding/json"
	"io"
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
	if !IsRelay("cy-sd4-seedance-2.0", "seedance-2.0") {
		t.Fatal("expected leonardo relay")
	}
	if !IsRelay("cy-sd4-minimax-h3-2k", "hailuo-03") {
		t.Fatal("expected minimax h3 relay")
	}
	if !IsRelay("cy-sd4-minimax-h3-768p", "hailuo-03") || !IsRelay("cy-sd4-minimax-h3-4k", "hailuo-03") {
		t.Fatal("expected all minimax3 resolution SKUs to use leonardo relay")
	}
	if !IsRelay("cy-sd4-happyhouse-1.0", "happy-horse") || !IsRelay("cy-sd4-happyhouse-1.1", "happy-horse-1.1") {
		t.Fatal("expected happyhouse relay")
	}
	if IsRelay("cy-sd4-happyhouse", "happy-horse") {
		t.Fatal("happyhouse family without a version must not match")
	}
	if IsRelay("cy-sd1-seedance-2.0-720p", "seedance-2.0") {
		t.Fatal("cy-sd1 must not match leonardo")
	}
	if IsRelay("cy-sd4-seedance-2.0", "seedance-2.0-fast") || IsRelay("cy-sd4-minimax-h3-4k", "seedance-2.5") {
		t.Fatal("mismatched internal/upstream pair must not use leonardo relay")
	}
}

func TestEstimateBilling_Seedance25UsesSecondsOnly(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("task_request", relaycommon.TaskSubmitReq{Duration: 4})
	adaptor := &TaskAdaptor{}

	got := adaptor.EstimateBilling(c, &relaycommon.RelayInfo{OriginModelName: seedance25Model480})
	if got["seconds"] != 4 || len(got) != 1 {
		t.Fatalf("unexpected 2.5 billing ratios: %#v", got)
	}
	if got := adaptor.EstimateBilling(c, &relaycommon.RelayInfo{OriginModelName: "cy-sd4-seedance-2.0"}); got != nil {
		t.Fatalf("2.0 billing must remain unchanged, got %#v", got)
	}
}

func TestEstimateBilling_Seedance25ReferenceVideoUsesSecondsOnly(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("task_request", relaycommon.TaskSubmitReq{Duration: 10, ReferenceVideos: []string{"https://cdn.example/ref.mp4"}})
	adaptor := &TaskAdaptor{}

	got := adaptor.EstimateBilling(c, &relaycommon.RelayInfo{OriginModelName: seedance25Model480})
	if got["seconds"] != 10 || len(got) != 1 {
		t.Fatalf("unexpected 480p reference billing ratios: %#v", got)
	}
	got = adaptor.EstimateBilling(c, &relaycommon.RelayInfo{OriginModelName: seedance25Model720})
	if got["seconds"] != 10 || len(got) != 1 {
		t.Fatalf("unexpected 720p reference billing ratios: %#v", got)
	}
}

func TestEstimateBilling_Seedance25DefaultsToEightSeconds(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("task_request", relaycommon.TaskSubmitReq{})
	got := (&TaskAdaptor{}).EstimateBilling(c, &relaycommon.RelayInfo{OriginModelName: seedance25Model720})
	if got["seconds"] != 8 {
		t.Fatalf("default seconds = %v, want 8", got["seconds"])
	}
}

func TestBuildRequestBody_FixedSKUForcesJSONResolution(t *testing.T) {
	for _, tc := range []struct {
		model      string
		upstream   string
		requested  string
		resolution string
	}{
		{seedance25Model480, "seedance-2.5", "720p", "480p"},
		{seedance25Model720, "seedance-2.5", "480p", "720p"},
		{minimax3Model768, "hailuo-03", "4k", "768p"},
		{minimax3Model2K, "hailuo-03", "768p", "2k"},
		{minimax3Model4K, "hailuo-03", "2k", "4k"},
	} {
		t.Run(tc.model, func(t *testing.T) {
			body := []byte(`{"model":"ignored","prompt":"test","duration":4,"resolution":"` + tc.requested + `"}`)
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("POST", "/v1/videos", bytes.NewReader(body))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Set("task_request", relaycommon.TaskSubmitReq{Duration: 4, Resolution: tc.requested})

			reader, err := (&TaskAdaptor{}).BuildRequestBody(c, &relaycommon.RelayInfo{
				OriginModelName: tc.model,
				ChannelMeta: &relaycommon.ChannelMeta{
					UpstreamModelName: tc.upstream,
				},
			})
			if err != nil {
				t.Fatalf("build body: %v", err)
			}
			raw, err := io.ReadAll(reader)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			var got map[string]interface{}
			if err := json.Unmarshal(raw, &got); err != nil {
				t.Fatalf("unmarshal body: %v", err)
			}
			if got["resolution"] != tc.resolution {
				t.Fatalf("resolution = %v, want %s", got["resolution"], tc.resolution)
			}
		})
	}
}

func TestBuildRequestBody_FixedSKUForcesMultipartResolution(t *testing.T) {
	c := multipartContextWithFields(t, "4", map[string][]string{"resolution": {"720p"}})
	c.Set("task_request", relaycommon.TaskSubmitReq{Duration: 4, Resolution: "720p"})
	reader, err := (&TaskAdaptor{}).BuildRequestBody(c, &relaycommon.RelayInfo{
		OriginModelName: seedance25Model480,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "seedance-2.5",
		},
	})
	if err != nil {
		t.Fatalf("build body: %v", err)
	}
	raw, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	req := httptest.NewRequest("POST", "/v1/videos", bytes.NewReader(raw))
	req.Header.Set("Content-Type", c.Request.Header.Get("Content-Type"))
	if err := req.ParseMultipartForm(1 << 20); err != nil {
		t.Fatalf("parse output: %v", err)
	}
	if got := req.FormValue("resolution"); got != "480p" {
		t.Fatalf("resolution = %q, want 480p", got)
	}
}

func TestBuildRequestBody_Minimax4KForcesMultipartResolution(t *testing.T) {
	c := multipartContextWithFields(t, "5", map[string][]string{"resolution": {"768p"}})
	c.Set("task_request", relaycommon.TaskSubmitReq{Duration: 5, Resolution: "768p"})
	reader, err := (&TaskAdaptor{}).BuildRequestBody(c, &relaycommon.RelayInfo{
		OriginModelName: minimax3Model4K,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "hailuo-03",
		},
	})
	if err != nil {
		t.Fatalf("build body: %v", err)
	}
	raw, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	req := httptest.NewRequest("POST", "/v1/videos", bytes.NewReader(raw))
	req.Header.Set("Content-Type", c.Request.Header.Get("Content-Type"))
	if err := req.ParseMultipartForm(1 << 20); err != nil {
		t.Fatalf("parse output: %v", err)
	}
	if got := req.FormValue("resolution"); got != "4k" {
		t.Fatalf("resolution = %q, want 4k", got)
	}
}

func TestBuildUpstreamBody_HappyHouseUsesDuration(t *testing.T) {
	out := buildUpstreamBody(map[string]interface{}{
		"prompt":     "test",
		"seconds":    float64(4),
		"resolution": "720p",
	}, "happy-horse-1.1", 4, nil)
	if out["model"] != "happy-horse-1.1" || out["duration"] != 4 {
		t.Fatalf("unexpected model/duration: %v", out)
	}
	if _, ok := out["seconds"]; ok {
		t.Fatalf("seconds must not be sent upstream: %v", out)
	}
}
