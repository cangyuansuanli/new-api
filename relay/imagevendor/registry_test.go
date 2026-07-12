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
	if !policy2k.AcceptUpstreamURL || !policy2k.AsyncPreferURLResponse {
		t.Fatalf("gulie 2k policy = %+v", policy2k)
	}
}

func TestResolveRehostPolicy4K(t *testing.T) {
	policy := ResolveRehostPolicy("geek2-gpt-image-2-4k")
	if !policy.AcceptUpstreamURL || !policy.AsyncPreferURLResponse {
		t.Fatalf("4k policy = %+v", policy)
	}
	if policy.PreferUpstreamB64JSON {
		t.Fatal("4k should not prefer upstream b64")
	}
}

func TestResolveRehostPolicyManjuBanana(t *testing.T) {
	for _, model := range []string{
		"manju-gemini-banana-pro-1/2k",
		"manju-gemini-banana-flash-lite",
	} {
		policy := ResolveRehostPolicy(model)
		if !policy.AcceptUpstreamURL {
			t.Fatalf("%s: expected AcceptUpstreamURL", model)
		}
		if policy.PreferUpstreamB64JSON || policy.AsyncPreferURLResponse {
			t.Fatalf("%s: policy should be url-only rehost, got %+v", model, policy)
		}
	}
}

func TestResolveRehostPolicyManjuBanana4KUsesLargeURLRule(t *testing.T) {
	policy := ResolveRehostPolicy("manju-gemini-banana-pro-4k")
	if !policy.AcceptUpstreamURL || !policy.AsyncPreferURLResponse {
		t.Fatalf("manju 4k should match large-url rule, got %+v", policy)
	}
}

func TestResolveRehostPolicyDefault(t *testing.T) {
	policy := ResolveRehostPolicy("go2api-gpt-image-2-1k")
	if policy.AcceptUpstreamURL || policy.PreferUpstreamB64JSON || policy.AsyncPreferURLResponse {
		t.Fatalf("internal model should use default policy, got %+v", policy)
	}
}

func TestImageModelUsesURLRehost(t *testing.T) {
	if !ImageModelUsesURLRehost("geek2-gpt-image-2-4k") {
		t.Fatal("expected 4k model to need url rehost")
	}
	if !ImageModelUsesURLRehost("flux-pro-2") {
		t.Fatal("expected flux-pro-2 to need url rehost")
	}
	if !ImageModelUsesURLRehost("Gulie-gpt-image-2") {
		t.Fatal("gulie should prefer upstream url for internal R2 transfer")
	}
}

func TestImageAsyncAcceptsUpstreamURL(t *testing.T) {
	if !ImageAsyncAcceptsUpstreamURL("geek2-gpt-image-2-4k") {
		t.Fatal("expected 4k async to accept upstream url")
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
	if ImageAsyncAcceptsUpstreamURL("go2api-gpt-image-2-1k") {
		t.Fatal("internal prefixed model should still require b64_json in async worker")
	}
}

func TestImageSyncPreferUpstreamB64JSON(t *testing.T) {
	if ImageSyncPreferUpstreamB64JSON("cy-img1-gpt-image-2") {
		t.Fatal("gulie should keep upstream url internal and rehost it to R2")
	}
	if ImageSyncPreferUpstreamB64JSON("geek2-gpt-image-2-4k") {
		t.Fatal("4k should not prefer upstream b64")
	}
	if ImageSyncPreferUpstreamB64JSON("manju-gemini-banana-pro-4k") {
		t.Fatal("manju banana should not prefer upstream b64")
	}
}
