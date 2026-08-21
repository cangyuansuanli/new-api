package imagevendor

import "testing"

func TestIsAiclubOriginModel(t *testing.T) {
	if !IsAiclubOriginModel("cy-ac-gpt-image-2-4k") {
		t.Fatal("expected aiclub gpt model")
	}
	if !IsAiclubOriginModel("cy-ac-nano-banana-pro-1k") {
		t.Fatal("expected aiclub banana model")
	}
	if IsAiclubOriginModel("adobe-firefly-gpt-image-2-4k") {
		t.Fatal("adobe model should not match aiclub")
	}
}

func TestAiclubFixedResolutionSKU(t *testing.T) {
	for _, tc := range []struct {
		model string
		tier  string
	}{
		{"cy-ac-gpt-image-2-1k", ImageResolution1K},
		{"cy-ac-gpt-image-2-2k", ImageResolution2K},
		{"cy-ac-gpt-image-2-4k", ImageResolution4K},
		{"cy-ac-nano-banana-pro-4k", ImageResolution4K},
		{"cy-ac-nano-banana2-2k", ImageResolution2K},
	} {
		tier, ok := AiclubFixedResolutionSKU(tc.model)
		if !ok || tier != tc.tier {
			t.Fatalf("%s: got %q ok=%v want %q", tc.model, tier, ok, tc.tier)
		}
	}
}

func TestResolveRehostPolicyAiclub(t *testing.T) {
	policy := ResolveRehostPolicy("cy-ac-gpt-image-2-4k")
	if !policy.AcceptUpstreamURL || !policy.AsyncPreferURLResponse {
		t.Fatalf("aiclub policy = %+v", policy)
	}
	if !ImageAsyncAcceptsUpstreamURL("cy-ac-nano-banana-pro-2k") {
		t.Fatal("expected async url rehost for aiclub banana")
	}
}
