package clienterror

import "testing"

func TestNormalizeOmni_ContentReviewWithoutReferenceMedia(t *testing.T) {
	raw := "This request didn't pass content review (e.g. an identifiable real person, unsafe content, or protected IP). Retrying or switching accounts won't help."

	if !IsOmniContentReviewError(raw) {
		t.Fatalf("IsOmniContentReviewError(%q) = false, want true", raw)
	}
	if IsRealFaceReferenceError(raw) {
		t.Fatalf("IsRealFaceReferenceError(%q) = true, want false", raw)
	}
	if got := NormalizeClientErrorMessageForLang(true, raw); got != ContentPolicyMessageZH {
		t.Fatalf("NormalizeClientErrorMessageForLang(zh) = %q, want %q", got, ContentPolicyMessageZH)
	}
	if got := NormalizeClientErrorMessageForLang(false, raw); got != ContentPolicyMessageEN {
		t.Fatalf("NormalizeClientErrorMessageForLang(en) = %q, want %q", got, ContentPolicyMessageEN)
	}
}

func TestNormalizeOmni_ContentReviewDirect(t *testing.T) {
	msg, ok := normalizeOmni(true, "This request did not pass content review.")
	if !ok || msg != ContentPolicyMessageZH {
		t.Fatalf("normalizeOmni = (%q, %v), want (%q, true)", msg, ok, ContentPolicyMessageZH)
	}
}
