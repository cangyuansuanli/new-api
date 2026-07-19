package clienterror

import "testing"

func TestIsContentPolicyViolation_AdobePromptUnsafe(t *testing.T) {
	raw := `video poll failed: 451 {"error_code":"prompt_unsafe","message":"The provided prompt is considered unsafe and it cannot be used to generate content."}`
	if !IsContentPolicyViolation(raw) {
		t.Fatalf("IsContentPolicyViolation(%q) = false, want true", raw)
	}
}

func TestNormalizeClientErrorMessageForLang_AdobePromptUnsafe(t *testing.T) {
	raw := `video poll failed: 451 {"error_code":"prompt_unsafe","message":"The provided prompt is considered unsafe and it cannot be used to generate content."}`

	if got := NormalizeClientErrorMessageForLang(true, raw); got != ContentPolicyMessageZH {
		t.Fatalf("NormalizeClientErrorMessageForLang(zh) = %q, want %q", got, ContentPolicyMessageZH)
	}
	if got := NormalizeClientErrorMessageForLang(false, raw); got != ContentPolicyMessageEN {
		t.Fatalf("NormalizeClientErrorMessageForLang(en) = %q, want %q", got, ContentPolicyMessageEN)
	}
}

func TestNormalizeClientErrorMessageForLang_AdobePromptUnsafeMessageOnly(t *testing.T) {
	raw := "The provided prompt is considered unsafe and it cannot be used to generate content."

	if got := NormalizeClientErrorMessageForLang(true, raw); got != ContentPolicyMessageZH {
		t.Fatalf("NormalizeClientErrorMessageForLang(zh, message only) = %q, want %q", got, ContentPolicyMessageZH)
	}
}
