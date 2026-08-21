package imagevendor

import "testing"

func TestIsManjuBananaOriginModel(t *testing.T) {
	if !IsManjuBananaOriginModel("manju-gemini-banana-2.0-1/2k") {
		t.Fatal("expected manju model")
	}
	if IsManjuBananaOriginModel("byte-gemini-banana-2.0") {
		t.Fatal("byte model should not match")
	}
}

func TestResolveRehostPolicyGulie(t *testing.T) {
	policy := ResolveRehostPolicy("cy-img1-gpt-image-2")
	if !policy.AcceptUpstreamURL || !policy.AsyncPreferURLResponse {
		t.Fatalf("gulie policy = %+v", policy)
	}
	if policy.PreferUpstreamB64JSON {
		t.Fatal("gulie should not prefer upstream b64 response")
	}
	policy2k := ResolveRehostPolicy("cy-img2-gpt-image-2-2k")
	if policy2k.AcceptUpstreamURL || policy2k.AsyncPreferURLResponse {
		t.Fatalf("retired cy-img2 2k should use default policy, got %+v", policy2k)
	}
}

func TestResolveRehostPolicyYunfeiGPTImage4K(t *testing.T) {
	policy := ResolveRehostPolicy("cy-yf-gpt-image-2-4k")
	if !policy.AcceptUpstreamURL || !policy.AsyncPreferURLResponse {
		t.Fatalf("yunfei 4k policy = %+v", policy)
	}
	if policy.PreferUpstreamB64JSON {
		t.Fatal("yunfei 4k should not prefer upstream b64")
	}
}

func TestResolveRehostPolicyManjuBanana(t *testing.T) {
	for _, model := range []string{
		"manju-gemini-banana-pro-1/2k",
		"manju-gemini-banana-flash-lite",
		"manju-gemini-banana-pro-4k",
	} {
		policy := ResolveRehostPolicy(model)
		if !policy.AcceptUpstreamURL {
			t.Fatalf("%s: expected AcceptUpstreamURL", model)
		}
		if policy.PreferUpstreamB64JSON || policy.AsyncPreferURLResponse {
			t.Fatalf("%s: policy should be sync url rehost only, got %+v", model, policy)
		}
	}
}

func TestResolveRehostPolicyAdobeFirefly(t *testing.T) {
	for _, model := range []string{
		"adobe-firefly-gpt-image-2-2k",
		"Adobe-Firefly-Nano-Banana-Pro-1K",
	} {
		policy := ResolveRehostPolicy(model)
		if !policy.AcceptUpstreamURL || !policy.AsyncPreferURLResponse {
			t.Fatalf("%s: expected URL rehost policy, got %+v", model, policy)
		}
		if policy.PreferUpstreamB64JSON {
			t.Fatalf("%s: should not prefer upstream b64", model)
		}
	}
}

func TestResolveRehostPolicyDefault(t *testing.T) {
	for _, model := range []string{
		"go2api-gpt-image-2-1k",
		"geek2-gpt-image-2-4k",
		"flux-pro-2",
	} {
		policy := ResolveRehostPolicy(model)
		if policy.AcceptUpstreamURL || policy.PreferUpstreamB64JSON || policy.AsyncPreferURLResponse {
			t.Fatalf("%s should use default policy, got %+v", model, policy)
		}
	}
}

func TestImageModelUsesURLRehost(t *testing.T) {
	if !ImageModelUsesURLRehost("cy-yf-gpt-image-2-4k") {
		t.Fatal("expected yunfei 4k model to need url rehost")
	}
	if !ImageModelUsesURLRehost("Gulie-gpt-image-2") {
		t.Fatal("gulie should prefer upstream url for internal R2 transfer")
	}
	if ImageModelUsesURLRehost("geek2-gpt-image-2-4k") {
		t.Fatal("retired catch-all models should not use url rehost")
	}
}

func TestImageAsyncAcceptsUpstreamURL(t *testing.T) {
	if !ImageAsyncAcceptsUpstreamURL("cy-yf-gpt-image-2-4k") {
		t.Fatal("expected yunfei 4k async to accept upstream url")
	}
	if !ImageAsyncAcceptsUpstreamURL("cy-img1-gpt-image-2") {
		t.Fatal("expected gulie async to accept upstream url")
	}
	if !ImageAsyncAcceptsUpstreamURL("Gulie-gpt-image-2") {
		t.Fatal("expected gulie async to accept upstream url")
	}
	if !ImageAsyncAcceptsUpstreamURL("manju-gemini-banana-pro-1/2k") {
		t.Fatal("expected manju banana async to accept upstream url")
	}
	if !ImageAsyncAcceptsUpstreamURL("adobe-firefly-gpt-image-2-2k") {
		t.Fatal("expected adobe firefly async to accept upstream url")
	}
	if ImageAsyncAcceptsUpstreamURL("go2api-gpt-image-2-1k") {
		t.Fatal("internal prefixed model should still require b64_json in async worker")
	}
	if ImageAsyncAcceptsUpstreamURL("flux-pro-2") {
		t.Fatal("flux without vendor registration should not accept upstream url")
	}
}

func TestImageSyncPreferUpstreamB64JSON(t *testing.T) {
	if ImageSyncPreferUpstreamB64JSON("cy-img1-gpt-image-2") {
		t.Fatal("gulie should keep upstream url internal and rehost it to R2")
	}
	if ImageSyncPreferUpstreamB64JSON("cy-yf-gpt-image-2-4k") {
		t.Fatal("yunfei 4k should not prefer upstream b64")
	}
	if ImageSyncPreferUpstreamB64JSON("manju-gemini-banana-pro-4k") {
		t.Fatal("manju banana should not prefer upstream b64")
	}
}
