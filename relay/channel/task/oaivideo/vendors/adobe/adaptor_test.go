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
	if !IsRelay("adobe-sora2", "sora2", 75, "") {
		t.Fatal("Adobe channel should be recognized after model mapping")
	}
	if !IsRelay("sora2", "sora2", 0, "https://adobe2api.example.test") {
		t.Fatal("Adobe base URL should be recognized")
	}
	if IsRelay("sora-2", "sora-2", 0, "https://api.openai.com") {
		t.Fatal("regular OpenAI Sora should not be recognized as Adobe")
	}
}

func TestBuildRequestBodyUsesAdobeStrictVideoSchema(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := `{"model":"adobe-veo31-ref","prompt":"a cat","duration":6,"aspect_ratio":"16x9","resolution":"1080p","generate_audio":true,"size":"bad","seed":42}`
	c := gin.CreateTestContextOnly(httptest.NewRecorder(), gin.New())
	c.Request = httptest.NewRequest("POST", "/v1/videos", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("task_request", relaycommon.TaskSubmitReq{Model: "adobe-veo31-ref", Prompt: "a cat", Duration: 6})

	reader, err := (&TaskAdaptor{}).BuildRequestBody(c, &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "veo31-ref"},
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
	if payload["model"] != "veo31-ref" || payload["prompt"] != "a cat" {
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
