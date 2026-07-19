package clienterror

import "testing"

func TestNormalizeAdobe_PromptUnsafeChinese(t *testing.T) {
	raw := `video poll failed: 451 {"error_code":"prompt_unsafe","message":"The provided prompt is considered unsafe and it cannot be used to generate content."}`
	msg, ok := normalizeAdobe(true, raw)
	if !ok {
		t.Fatalf("normalizeAdobe(%q) = ok false", raw)
	}
	if msg != ContentPolicyMessageZH {
		t.Fatalf("normalizeAdobe(zh) = %q, want %q", msg, ContentPolicyMessageZH)
	}
}

func TestNormalizeAdobe_PromptUnsafeEnglish(t *testing.T) {
	raw := `The provided prompt is considered unsafe and it cannot be used to generate content.`
	msg, ok := normalizeAdobe(false, raw)
	if !ok {
		t.Fatalf("normalizeAdobe(%q) = ok false", raw)
	}
	if msg != ContentPolicyMessageEN {
		t.Fatalf("normalizeAdobe(en) = %q, want %q", msg, ContentPolicyMessageEN)
	}
}
