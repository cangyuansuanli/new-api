package seedancehuabu

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	oaivideo "github.com/QuantumNous/new-api/relay/channel/task/oaivideo/shared"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
)

func TestIsRelayRequiresExactModelPair(t *testing.T) {
	if !IsRelay(ModelStandard, UpstreamStandard) {
		t.Fatal("expected standard pair")
	}
	if !IsRelay(ModelFast, UpstreamFast) {
		t.Fatal("expected fast pair")
	}
	if IsRelay(ModelStandard, UpstreamFast) {
		t.Fatal("mismatched upstream must not match")
	}
	if IsRelay("sora-2", "sora-2") {
		t.Fatal("unrelated model must not match")
	}
}

func TestBuildRequestBodyMapsPublicAssetFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := `{"model":"cy-sd8-seedance-2.0","prompt":"test prompt","duration":10,"aspect_ratio":"16:9","reference_image_urls":["https://img/1.png","https://img/2.png"],"reference_videos":["https://vid/1.mp4"],"reference_audios":["https://aud/1.mp3"]}`
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/videos", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	info := &relaycommon.RelayInfo{
		OriginModelName: ModelStandard,
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{},
		ChannelMeta:     &relaycommon.ChannelMeta{UpstreamModelName: UpstreamStandard},
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
	if got["model"] != UpstreamStandard || got["size"] != "16:9" || got["duration"] != float64(10) {
		t.Fatalf("unexpected core fields: %#v", got)
	}
	images, ok := got["image"].([]any)
	if !ok || len(images) != 2 {
		t.Fatalf("expected image array under image, got %#v", got["image"])
	}
	if _, ok := got["images"]; ok {
		t.Fatal("upstream must not receive images field")
	}
}

func TestBuildUpstreamBodyFastOmitsVideoAndAudio(t *testing.T) {
	got := buildUpstreamBody(
		map[string]any{"prompt": "hello", "reference_videos": []string{"https://v/1.mp4"}, "reference_audios": []string{"https://a/1.mp3"}},
		ModelFast, UpstreamFast, 10, nil, []string{"https://v/1.mp4"}, []string{"https://a/1.mp3"},
	)
	if _, ok := got["videos"]; ok {
		t.Fatal("fast model must not send videos")
	}
	if _, ok := got["audios"]; ok {
		t.Fatal("fast model must not send audios")
	}
}

func TestValidateRequestRejectsFastReferenceVideos(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := `{"model":"cy-sd8-seedance-2.0-fast","prompt":"test","duration":10,"reference_videos":["https://vid/1.mp4"]}`
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/videos", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	info := &relaycommon.RelayInfo{OriginModelName: ModelFast}
	if taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(c, info); taskErr == nil {
		t.Fatal("expected fast model to reject reference videos")
	}
}

func TestValidateRequestRejectsInvalidDuration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := `{"model":"cy-sd8-seedance-2.0","prompt":"test","duration":8}`
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/videos", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	info := &relaycommon.RelayInfo{OriginModelName: ModelStandard}
	if taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(c, info); taskErr == nil {
		t.Fatal("expected invalid duration error")
	}
}

func TestParseTaskResultUsesResultURL(t *testing.T) {
	resp := []byte(`{"id":"task_1","status":"completed","result_url":"https://cdn.example.com/video.mp4"}`)
	a := &TaskAdaptor{}
	result, err := a.ParseTaskResult(resp)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if result.Status != model.TaskStatusSuccess {
		t.Fatalf("status = %q", result.Status)
	}
	if result.Url != "https://cdn.example.com/video.mp4" {
		t.Fatalf("url = %q", result.Url)
	}
}

func TestExtractSd8VideoURL(t *testing.T) {
	resp := []byte(`{"status":"completed","result_url":"https://cdn.example.com/video.mp4"}`)
	res, err := oaivideo.ParseResponseTask(resp)
	if err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if got := extractSd8VideoURL(resp, res); got != "https://cdn.example.com/video.mp4" {
		t.Fatalf("url = %q", got)
	}
}

func TestResolveTaskResultSourcePrefersDirectResultURL(t *testing.T) {
	resp := []byte(`{"id":"task_upstream","status":"completed","result_url":"https://v3-default.douyin.com/video.mp4"}`)
	a := &TaskAdaptor{}
	if _, err := a.ParseTaskResult(resp); err != nil {
		t.Fatalf("parse: %v", err)
	}
	source := a.ResolveTaskResultSource("https://huabu-admin.example", "task_upstream", "secret")
	if source == nil {
		t.Fatal("expected source")
	}
	if source.URL != "" {
		t.Fatalf("direct result_url should skip /content override, got %q", source.URL)
	}
	if source.Headers.Get("Authorization") != "Bearer secret" {
		t.Fatalf("unexpected auth header %q", source.Headers.Get("Authorization"))
	}
}

func TestResolveTaskResultSourceFallsBackToContentEndpoint(t *testing.T) {
	resp := []byte(`{"id":"task_upstream","status":"completed"}`)
	a := &TaskAdaptor{}
	if _, err := a.ParseTaskResult(resp); err != nil {
		t.Fatalf("parse: %v", err)
	}
	source := a.ResolveTaskResultSource("https://huabu-admin.example", "task_upstream", "secret")
	if source == nil {
		t.Fatal("expected source")
	}
	want := "https://huabu-admin.example/v1/videos/task_upstream/content"
	if source.URL != want {
		t.Fatalf("missing result_url should use /content, got %q", source.URL)
	}
}
