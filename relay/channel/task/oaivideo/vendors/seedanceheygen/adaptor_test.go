package seedanceheygen

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
)

func TestIsRelayRequiresExactModelPair(t *testing.T) {
	if !IsRelay(Model720p, UpstreamModel) || !IsRelay(Model1080p, UpstreamModel) {
		t.Fatal("expected both exact product pairs to match")
	}
	if IsRelay(Model720p, "seedance-2.0-fast") || IsRelay("cy-sd4-seedance-2.0", UpstreamModel) {
		t.Fatal("mismatched model pair must not match")
	}
}

func TestBuildRequestBodyForcesProductResolutionAndWhitelistsFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := `{"model":"seedance-2.0-1080p","prompt":" test ","seconds":"4","resolution":"720p","aspect_ratio":"16:9","image_urls":["https://img/1.png","https://img/2.png"],"reference_videos":["https://video/1.mp4"],"reference_audios":["https://audio/1.mp3"],"avatar_id":"must-not-pass"}`
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/videos", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	info := &relaycommon.RelayInfo{
		OriginModelName: Model1080p,
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{},
		ChannelMeta:     &relaycommon.ChannelMeta{UpstreamModelName: UpstreamModel},
	}
	a := &TaskAdaptor{}
	if taskErr := a.ValidateRequestAndSetAction(c, info); taskErr != nil {
		t.Fatalf("validate: %#v", taskErr)
	}
	reader, err := a.BuildRequestBody(c, info)
	if err != nil {
		t.Fatalf("build body: %v", err)
	}
	encoded, _ := io.ReadAll(reader)
	var got map[string]any
	if err := common.Unmarshal(encoded, &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got["resolution"] != "1080p" || got["model"] != UpstreamModel || got["duration"].(float64) != 4 {
		t.Fatalf("unexpected forced fields: %#v", got)
	}
	if _, exists := got["avatar_id"]; exists {
		t.Fatal("unknown client field leaked upstream")
	}
	if _, exists := got["image_url"]; exists {
		t.Fatal("multiple normalized images must use reference_image_urls")
	}
}

func TestResolveTaskResultSourceIncludesBearer(t *testing.T) {
	source := (&TaskAdaptor{}).ResolveTaskResultSource("https://example.com/heygen-api/", "video_7", "secret")
	if source.URL != "https://example.com/heygen-api/v1/videos/video_7/content" {
		t.Fatalf("unexpected URL %q", source.URL)
	}
	if source.Headers.Get("Authorization") != "Bearer secret" {
		t.Fatalf("unexpected auth header %q", source.Headers.Get("Authorization"))
	}
}
