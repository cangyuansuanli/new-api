package clienterror

import (
	"net/http"
	"testing"
)

func TestNormalizeTransportTooManyRequests(t *testing.T) {
	failure := ErrorContext{Raw: "provider rate limit", HTTPStatus: http.StatusTooManyRequests}
	if got := NormalizeErrorForLang(true, failure); got != UpstreamSaturatedMessageZH {
		t.Fatalf("NormalizeErrorForLang(zh) = %q, want %q", got, UpstreamSaturatedMessageZH)
	}
	if got := NormalizeErrorForLang(false, failure); got != UpstreamSaturatedMessageEN {
		t.Fatalf("NormalizeErrorForLang(en) = %q, want %q", got, UpstreamSaturatedMessageEN)
	}
}
