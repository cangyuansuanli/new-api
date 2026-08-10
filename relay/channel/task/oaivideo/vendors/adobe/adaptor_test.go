package adobe

import (
	"io"
	"net/http/httptest"
	"reflect"
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

func TestSD5DefaultsReferenceImagesToMediaMode(t *testing.T) {
	images := []string{"i1", "i2", "i3", "i4", "i5", "i6", "i7", "i8", "i9"}
	payload, err := buildAdobeTestPayload(t, `{"model":"seedance-2.0","prompt":"test","duration":4,"images":["i1","i2","i3","i4","i5","i6","i7","i8","i9"]}`, relaycommon.TaskSubmitReq{
		Model: "seedance-2.0", Prompt: "test", Duration: 4, Images: images,
	})
	if err != nil {
		t.Fatal(err)
	}
	if payload["reference_mode"] != "media" {
		t.Fatalf("reference_mode = %#v", payload["reference_mode"])
	}
	if got := stringListFromPayload(t, payload["images"]); !reflect.DeepEqual(got, images) {
		t.Fatalf("images = %#v", got)
	}
}

func TestSD5AllowsNineImagesAndThreeSourceMedia(t *testing.T) {
	images := []string{"i1", "i2", "i3", "i4", "i5", "i6", "i7", "i8", "i9"}
	for _, testCase := range []struct {
		name   string
		videos []string
		audios []string
	}{
		{name: "three videos", videos: []string{"v1", "v2", "v3"}},
		{name: "three audios", audios: []string{"a1", "a2", "a3"}},
		{name: "mixed sources", videos: []string{"v1", "v2"}, audios: []string{"a1"}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			payload, err := buildAdobeTestPayload(t, `{"model":"seedance-2.0-fast","prompt":"test","duration":4,"images":["i1","i2","i3","i4","i5","i6","i7","i8","i9"]}`, relaycommon.TaskSubmitReq{
				Model: "seedance-2.0-fast", Prompt: "test", Duration: 4, Images: images,
				ReferenceVideos: testCase.videos, ReferenceAudios: testCase.audios,
			})
			if err != nil {
				t.Fatal(err)
			}
			if payload["reference_mode"] != "media" {
				t.Fatalf("reference_mode = %#v", payload["reference_mode"])
			}
		})
	}
}

func TestSD5ReferenceLimits(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		request relaycommon.TaskSubmitReq
		want    string
	}{
		{
			name:    "ten images",
			body:    `{"model":"seedance-2.0","prompt":"test","duration":4,"reference_mode":"media","images":["1","2","3","4","5","6","7","8","9","10"]}`,
			request: relaycommon.TaskSubmitReq{Model: "seedance-2.0", Prompt: "test", Duration: 4, Images: []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"}},
			want:    "at most 9 reference images",
		},
		{
			name:    "four source items",
			body:    `{"model":"seedance-2.0","prompt":"test","duration":4,"reference_mode":"media","images":["i"],"reference_videos":["v1","v2"],"reference_audios":["a1","a2"]}`,
			request: relaycommon.TaskSubmitReq{Model: "seedance-2.0", Prompt: "test", Duration: 4, Images: []string{"i"}, ReferenceVideos: []string{"v1", "v2"}, ReferenceAudios: []string{"a1", "a2"}},
			want:    "at most 3 items combined",
		},
		{
			name:    "thirteen total assets",
			body:    `{"model":"seedance-2.0","prompt":"test","duration":4,"reference_mode":"media","images":["1","2","3","4","5","6","7","8","9"],"reference_videos":["v1","v2","v3"],"reference_audios":["a1"]}`,
			request: relaycommon.TaskSubmitReq{Model: "seedance-2.0", Prompt: "test", Duration: 4, Images: []string{"1", "2", "3", "4", "5", "6", "7", "8", "9"}, ReferenceVideos: []string{"v1", "v2", "v3"}, ReferenceAudios: []string{"a1"}},
			want:    "at most 12 total reference assets",
		},
		{
			name:    "media without image",
			body:    `{"model":"seedance-2.0","prompt":"test","duration":4,"reference_mode":"media","reference_videos":["v1"]}`,
			request: relaycommon.TaskSubmitReq{Model: "seedance-2.0", Prompt: "test", Duration: 4, ReferenceVideos: []string{"v1"}},
			want:    "require at least one image",
		},
		{
			name:    "frame mixed with source media",
			body:    `{"model":"seedance-2.0","prompt":"test","duration":4,"reference_mode":"frame","images":["first","last"],"reference_videos":["v1"]}`,
			request: relaycommon.TaskSubmitReq{Model: "seedance-2.0", Prompt: "test", Duration: 4, Images: []string{"first", "last"}, ReferenceVideos: []string{"v1"}},
			want:    "cannot be combined",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := buildAdobeTestPayload(t, testCase.body, testCase.request)
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("error = %v, want containing %q", err, testCase.want)
			}
		})
	}
}

func TestSD5FrameModeUsesFirstAndLastFrames(t *testing.T) {
	payload, err := buildAdobeTestPayload(t, `{"model":"seedance-2.0","prompt":"test","duration":4,"reference_mode":"frame","images":["first","last"]}`, relaycommon.TaskSubmitReq{
		Model: "seedance-2.0", Prompt: "test", Duration: 4, Images: []string{"first", "last"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if payload["reference_mode"] != "frame" || payload["first_image_url"] != "first" || payload["last_image_url"] != "last" {
		t.Fatalf("frame payload = %#v", payload)
	}
	if _, exists := payload["images"]; exists {
		t.Fatalf("frame images leaked into payload: %#v", payload)
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

func buildAdobeTestPayload(t *testing.T, body string, request relaycommon.TaskSubmitReq) (map[string]any, error) {
	t.Helper()
	c := gin.CreateTestContextOnly(httptest.NewRecorder(), gin.New())
	c.Request = httptest.NewRequest("POST", "/v1/videos", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("task_request", request)
	reader, err := (&TaskAdaptor{}).BuildRequestBody(c, &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: request.Model},
	})
	if err != nil {
		return nil, err
	}
	out, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	payload := map[string]any{}
	if err := basecommon.Unmarshal(out, &payload); err != nil {
		t.Fatal(err)
	}
	return payload, nil
}

func stringListFromPayload(t *testing.T, value any) []string {
	t.Helper()
	raw, ok := value.([]any)
	if !ok {
		t.Fatalf("value is not a list: %#v", value)
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		text, ok := item.(string)
		if !ok {
			t.Fatalf("list item is not a string: %#v", item)
		}
		out = append(out, text)
	}
	return out
}
