package sd5

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	basecommon "github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
)

func TestIsRelayUsesSD5ModelIdentityWithoutMapping(t *testing.T) {
	if !IsRelay("cy-sd5-seedance-2.0-fast", "cy-sd5-seedance-2.0-fast") {
		t.Fatal("SD5 model should select the dedicated vendor without model mapping")
	}
	if IsRelay("adobe-sora2", "sora2") {
		t.Fatal("Adobe Sora should not select the SD5 vendor")
	}
}

func TestBuildRequestBodyPreservesSeedance933References(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := `{"model":"cy-sd5-seedance-2.0-fast","prompt":"test","duration":4,"aspect_ratio":"16x9","resolution":"480p","seed":0,"generate_audio":false,"reference_mode":"media","images":["i1"],"reference_videos":["v1","v2","v3"],"reference_audios":["a1","a2","a3"]}`
	c := gin.CreateTestContextOnly(httptest.NewRecorder(), gin.New())
	c.Request = httptest.NewRequest("POST", "/v1/videos", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	var taskRequest relaycommon.TaskSubmitReq
	if err := basecommon.Unmarshal([]byte(body), &taskRequest); err != nil {
		t.Fatal(err)
	}
	if taskRequest.Seed == nil || *taskRequest.Seed != 0 {
		t.Fatalf("decoded seed = %#v, want explicit zero", taskRequest.Seed)
	}
	c.Set("task_request", taskRequest)

	reader, err := (&TaskAdaptor{}).BuildRequestBody(c, &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "cy-sd5-seedance-2.0-fast"},
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
	if payload["model"] != "cy-sd5-seedance-2.0-fast" {
		t.Fatalf("model name should pass through unchanged: %#v", payload)
	}
	if payload["aspect_ratio"] != "16:9" || payload["reference_mode"] != "media" {
		t.Fatalf("SD5 request normalization failed: %#v", payload)
	}
	if payload["duration"] != float64(4) || payload["resolution"] != "480p" || payload["generate_audio"] != false {
		t.Fatalf("SD5 scalar parameters were not preserved: %#v", payload)
	}
	if seedValue, ok := payload["seed"].(float64); !ok || seedValue != 0 {
		t.Fatalf("seed = %#v, want explicit zero", payload["seed"])
	}
	if got, ok := payload["reference_videos"].([]any); !ok || len(got) != 3 {
		t.Fatalf("reference videos were not preserved: %#v", payload)
	}
	if got, ok := payload["reference_audios"].([]any); !ok || len(got) != 3 {
		t.Fatalf("reference audios were not preserved: %#v", payload)
	}
	if got, ok := payload["images"].([]any); !ok || len(got) != 1 || got[0] != "i1" {
		t.Fatalf("reference images were not preserved: %#v", payload)
	}
}

func TestSD5UsesTypedSubmitAndSucceededResponse(t *testing.T) {
	url, err := (&TaskAdaptor{}).BuildRequestURL(&relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{ChannelBaseUrl: "http://45.67.221.45:6002/"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if url != "http://45.67.221.45:6002/v1/videos/generations" {
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
