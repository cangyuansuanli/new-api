package relay

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

func TestApplySyncImageUpstreamB64Override(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)

	info := &relaycommon.RelayInfo{OriginModelName: "gpt-image-2"}
	request := &dto.ImageRequest{ResponseFormat: "url"}

	applySyncImageUpstreamB64Override(c, info, request)

	if !info.ImageClientWantsURL {
		t.Fatal("expected ImageClientWantsURL")
	}
	if request.ResponseFormat != "b64_json" {
		t.Fatalf("response_format = %q, want b64_json", request.ResponseFormat)
	}

	info2 := &relaycommon.RelayInfo{OriginModelName: "geek2-gpt-image-2-4k"}
	request2 := &dto.ImageRequest{ResponseFormat: "url"}
	applySyncImageUpstreamB64Override(c, info2, request2)
	if info2.ImageClientWantsURL {
		t.Fatal("4k model should keep upstream url response")
	}
	if request2.ResponseFormat != "url" {
		t.Fatalf("4k response_format = %q, want url", request2.ResponseFormat)
	}
}

func TestImageSyncPreferUpstreamB64JSON(t *testing.T) {
	if !service.ImageSyncPreferUpstreamB64JSON("gpt-image-2") {
		t.Fatal("expected gpt-image-2")
	}
	if service.ImageSyncPreferUpstreamB64JSON("geek2-gpt-image-2-4k") {
		t.Fatal("4k should not prefer upstream b64 for client url path")
	}
}
