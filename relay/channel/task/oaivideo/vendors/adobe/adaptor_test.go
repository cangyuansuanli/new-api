package adobe

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	basecommon "github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
)

func TestIsRelayUsesChannelIdentityWhenModelIsMapped(t *testing.T) {
	if !IsRelay("cy-adobe-veo-3.1", "veo-3.1", 0, "https://adobe2api.example.test") {
		t.Fatal("Adobe base URL should be recognized")
	}
	if IsRelay("sora-2", "sora-2", 0, "https://api.openai.com") {
		t.Fatal("regular OpenAI Sora should not be recognized as Adobe")
	}
	if !IsRelay("cy-sd5-seedance-2.0", "seedance-2.0", 0, "") {
		t.Fatal("channel 86 should use the unified Adobe vendor")
	}
}

func TestAdobeModelListUsesCurrentContractNames(t *testing.T) {
	for _, model := range []string{"cy-adobe-veo-3.1", "cy-adobe-veo-3.1-fast", "cy-adobe-kling-3.0", "cy-adobe-kling-3.0-omni", "cy-adobe-gemini-omni-flash"} {
		if !IsRelay(model, strings.TrimPrefix(model, "cy-adobe-"), 0, "") {
			t.Fatalf("current Adobe model %q should be recognized", model)
		}
	}
	if !IsRelay("cy-sd5-seedance-2.0", "seedance-2.0", 75, "") {
		t.Fatal("SD5 Seedance should use the unified Adobe vendor")
	}
}

func TestBuildRequestBodyUsesAdobeStrictVideoSchema(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := `{"model":"veo-3.1-fast","prompt":"a cat","duration":6,"aspect_ratio":"16x9","resolution":"1080p","generate_audio":true,"size":"bad","seed":42}`
	c := gin.CreateTestContextOnly(httptest.NewRecorder(), gin.New())
	c.Request = httptest.NewRequest("POST", "/v1/videos", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("task_request", relaycommon.TaskSubmitReq{Model: "veo-3.1-fast", Prompt: "a cat", Duration: 6})

	reader, err := (&TaskAdaptor{}).BuildRequestBody(c, &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "veo-3.1-fast"},
	})
	if err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := basecommon.Unmarshal(out, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["model"] != "veo-3.1-fast" || payload["prompt"] != "a cat" {
		t.Fatalf("unexpected required fields: %#v", payload)
	}
	if payload["aspect_ratio"] != "16:9" {
		t.Fatalf("aspect ratio was not normalized: %#v", payload["aspect_ratio"])
	}
	if _, exists := payload["seed"]; exists {
		t.Fatal("unsupported seed leaked into strict Adobe request")
	}
	if _, exists := payload["size"]; exists {
		t.Fatal("UI-only size leaked into strict Adobe request")
	}
}

func TestBuildRequestBodyValidatesModelSpecificCapabilities(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := `{"model":"gemini-omni-flash","prompt":"test","duration":10,"seed":1,"reference_videos":["v.mp4"]}`
	c := gin.CreateTestContextOnly(httptest.NewRecorder(), gin.New())
	c.Request = httptest.NewRequest("POST", "/v1/videos", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("task_request", relaycommon.TaskSubmitReq{Model: "gemini-omni-flash", Prompt: "test", Duration: 10, ReferenceVideos: []string{"v.mp4"}})
	_, err := (&TaskAdaptor{}).BuildRequestBody(c, &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "gemini-omni-flash"}})
	if err == nil || !strings.Contains(err.Error(), "does not support video or audio") {
		t.Fatalf("error = %v", err)
	}
}

func TestVeo31AllowsThreeAssetReferences(t *testing.T) {
	c := gin.CreateTestContextOnly(httptest.NewRecorder(), gin.New())
	c.Request = httptest.NewRequest("POST", "/v1/videos", strings.NewReader(`{"model":"veo-3.1","prompt":"test","duration":8,"reference_mode":"asset","images":["a","b","c"]}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("task_request", relaycommon.TaskSubmitReq{Model: "veo-3.1", Prompt: "test", Duration: 8, Images: []string{"a", "b", "c"}})
	if _, err := (&TaskAdaptor{}).BuildRequestBody(c, &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "veo-3.1"}}); err != nil {
		t.Fatal(err)
	}
}

func TestVeoFastRejectsAssetReferences(t *testing.T) {
	c := gin.CreateTestContextOnly(httptest.NewRecorder(), gin.New())
	c.Request = httptest.NewRequest("POST", "/v1/videos", strings.NewReader(`{"model":"veo-3.1-fast","prompt":"test","duration":8,"reference_mode":"asset","images":["a"]}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("task_request", relaycommon.TaskSubmitReq{Model: "veo-3.1-fast", Prompt: "test", Duration: 8, Images: []string{"a"}})
	if _, err := (&TaskAdaptor{}).BuildRequestBody(c, &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "veo-3.1-fast"}}); err == nil {
		t.Fatal("expected asset references to be rejected")
	}
}

func TestGeminiOmniAllowsThreeToTenSecondsAndSingleFirstFrame(t *testing.T) {
	c := gin.CreateTestContextOnly(httptest.NewRecorder(), gin.New())
	c.Request = httptest.NewRequest("POST", "/v1/videos", strings.NewReader(`{"model":"gemini-omni-flash","prompt":"test","duration":6,"first_image_url":"first.png"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("task_request", relaycommon.TaskSubmitReq{Model: "gemini-omni-flash", Prompt: "test", Duration: 6, FirstImageUrl: "first.png"})
	if _, err := (&TaskAdaptor{}).BuildRequestBody(c, &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "gemini-omni-flash"}}); err != nil {
		t.Fatal(err)
	}
}

func TestAdobeUsesTypedSubmitAndSucceededResponse(t *testing.T) {
	url, err := (&TaskAdaptor{}).BuildRequestURL(&relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{ChannelBaseUrl: "https://adobe.example.test/"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if url != "https://adobe.example.test/v1/videos/generations" {
		t.Fatalf("unexpected submit URL: %s", url)
	}
	result, err := (&TaskAdaptor{}).ParseTaskResult([]byte(`{"object":"video.generation","id":"vid_1","status":"succeeded","progress":100.0,"video_url":"https://example.test/out.mp4"}`))
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "SUCCESS" || result.Url != "https://example.test/out.mp4" {
		t.Fatalf("unexpected succeeded result: %+v", result)
	}
}
