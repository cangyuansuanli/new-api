package image

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
)

func TestApplySyncImageUpstreamURLRehostPolicy(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)

	info := &relaycommon.RelayInfo{OriginModelName: "cy-img1-gpt-image-2"}
	request := &dto.ImageRequest{ResponseFormat: "url"}

	applySyncImageUpstreamB64Override(c, info, request)

	if !info.ImageClientWantsURL {
		t.Fatal("expected ImageClientWantsURL")
	}
	if request.ResponseFormat != "url" {
		t.Fatalf("response_format = %q, want url", request.ResponseFormat)
	}

	info2 := &relaycommon.RelayInfo{OriginModelName: "geek2-gpt-image-2-4k"}
	request2 := &dto.ImageRequest{ResponseFormat: "url"}
	applySyncImageUpstreamB64Override(c, info2, request2)
	if !info2.ImageClientWantsURL {
		t.Fatal("4k model should rehost upstream url response")
	}
	if request2.ResponseFormat != "url" {
		t.Fatalf("4k response_format = %q, want url", request2.ResponseFormat)
	}
}
