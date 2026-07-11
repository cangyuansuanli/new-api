package imagevendor

import (
	"encoding/json"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	"github.com/gin-gonic/gin"
)

func TestValidateFixedResolutionSKUIgnoresPromptText(t *testing.T) {
	request := &dto.ImageRequest{Prompt: "render a 4K cinematic poster"}
	if err := ValidateFixedResolutionSKU(nil, "adobe-firefly-nano-banana-pro-1k", request); err != nil {
		t.Fatalf("prompt text must not select the protocol resolution: %v", err)
	}
}

func TestValidateFixedResolutionSKURejectsStructuredBypass(t *testing.T) {
	tests := []dto.ImageRequest{
		{Extra: map[string]json.RawMessage{"image_size": json.RawMessage(`"4K"`)}},
		{Extra: map[string]json.RawMessage{"output_resolution": json.RawMessage(`"4K"`)}},
		{Extra: map[string]json.RawMessage{"metadata": json.RawMessage(`{"resolution":"4K"}`)}},
		{Extra: map[string]json.RawMessage{"extra_body": json.RawMessage(`{"google":{"image_config":{"image_size":"4K"}}}`)}},
		{Size: "3840x2160"},
		{Size: "16:9-4k"},
		{Quality: "high"},
	}
	for index := range tests {
		err := ValidateFixedResolutionSKU(nil, "adobe-firefly-nano-banana-pro-1k", &tests[index])
		if err == nil || !strings.Contains(err.Error(), "fixed 1K SKU") {
			t.Fatalf("case %d: expected fixed-SKU rejection, got %v", index, err)
		}
	}
}

func TestValidateFixedResolutionSKUAllowsMatchingOrOmittedResolution(t *testing.T) {
	for _, request := range []dto.ImageRequest{
		{},
		{Size: "2048x1152"},
		{Quality: "medium"},
		{Extra: map[string]json.RawMessage{"image_size": json.RawMessage(`"2K"`)}},
	} {
		request := request
		if err := ValidateFixedResolutionSKU(nil, "adobe-firefly-nano-banana-pro-2k", &request); err != nil {
			t.Fatalf("matching request rejected: %v", err)
		}
	}
}

func TestValidateFixedResolutionSKURejectsMultipleImages(t *testing.T) {
	n := uint(2)
	err := ValidateFixedResolutionSKU(nil, "adobe-firefly-gpt-image-2-4k", &dto.ImageRequest{N: &n})
	if err == nil || !strings.Contains(err.Error(), "n=1") {
		t.Fatalf("expected n rejection, got %v", err)
	}
}

func TestValidateFixedResolutionSKURejectsMultipartBypass(t *testing.T) {
	c, _ := gin.CreateTestContext(nil)
	c.Request = &http.Request{MultipartForm: &multipart.Form{Value: map[string][]string{
		"output_resolution": {"4K"},
	}}}
	err := ValidateFixedResolutionSKU(c, "adobe-firefly-gpt-image-2-1k", &dto.ImageRequest{})
	if err == nil || !strings.Contains(err.Error(), "fixed 1K SKU") {
		t.Fatalf("expected multipart resolution rejection, got %v", err)
	}
}

func TestFixedResolutionSKUDoesNotCaptureExistingNonAdobeProducts(t *testing.T) {
	for _, model := range []string{
		"go2api-gpt-image-2-1k",
		"cy-img2-gpt-image-2-2k",
		"cy-img2-gpt-image-2-4k",
		"geek2-gpt-image-2-4k",
		"manju-gemini-banana-pro-4k",
	} {
		if tier, ok := FixedResolutionSKU(model); ok {
			t.Fatalf("existing non-Adobe model %q captured as %s", model, tier)
		}
	}
}
