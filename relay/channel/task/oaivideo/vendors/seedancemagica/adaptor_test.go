package seedancemagica

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
)

func TestIsRelayRequiresUnifiedUpstream(t *testing.T) {
	if !IsRelay(Model720p, ChannelUpstreamSlug) {
		t.Fatal("expected 720p pair")
	}
	if IsRelay(Model720p, UpstreamReference) {
		t.Fatal("channel upstream must be seedance-2.0")
	}
}

func TestBuildRequestBodyForcesResolutionAndReferenceRouting(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := `{"model":"cy-sd7-seedance-2.0-1080p","prompt":" test @Image1","seconds":"5","reference_image_urls":["https://img/1.png","https://img/2.png"]}`
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/videos", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	info := &relaycommon.RelayInfo{
		OriginModelName: Model1080p,
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{},
		ChannelMeta:     &relaycommon.ChannelMeta{UpstreamModelName: ChannelUpstreamSlug},
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
	if got["resolution"] != "1080p" || got["model"] != UpstreamReference {
		t.Fatalf("unexpected forced fields: %#v", got)
	}
}

func TestBuildRequestBodyStandardWhenNoReference(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := `{"model":"cy-sd7-seedance-2.0-720p","prompt":"plain text","seconds":"5"}`
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/videos", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	info := &relaycommon.RelayInfo{
		OriginModelName: Model720p,
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{},
		ChannelMeta:     &relaycommon.ChannelMeta{UpstreamModelName: ChannelUpstreamSlug},
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
	if got["model"] != UpstreamStandard {
		t.Fatalf("expected standard upstream, got %#v", got)
	}
}

func TestValidateRequestRejectsFrameInputs(t *testing.T) {
	for _, body := range []string{
		`{"model":"cy-sd7-seedance-2.0-720p","prompt":"test","duration":5,"first_image_url":"https://img/first.png"}`,
		`{"model":"cy-sd7-seedance-2.0-720p","prompt":"test","duration":5,"first_image_url":"https://img/first.png","last_image_url":"https://img/last.png"}`,
		`{"model":"cy-sd7-seedance-2.0-720p","prompt":"test","duration":5,"first_image_url":"https://img/first.png","last_image_url":"https://img/last.png","reference_videos":["https://vid/ref.mp4"]}`,
	} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("POST", "/v1/videos", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		info := &relaycommon.RelayInfo{OriginModelName: Model720p, TaskRelayInfo: &relaycommon.TaskRelayInfo{}}
		if taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(c, info); taskErr == nil {
			t.Fatalf("expected frame inputs to be rejected for %s", body)
		}
	}
}
