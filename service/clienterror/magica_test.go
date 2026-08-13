package clienterror

import "testing"

func TestSanitizeMagicaClientMessageStripsUpstreamHint(t *testing.T) {
	raw := "参考图 必须是公网 HTTPS 链接；不支持 base64/data URL（Magica 上游请求体上限约 4MB）"
	got := sanitizeMagicaClientMessage(raw)
	if got != "参考图必须是公网 HTTPS 链接，不支持 base64 或本地 data URL" {
		t.Fatalf("got %q", got)
	}
}

func TestNormalizeMagicaRelaySanitizesLegacyReferenceError(t *testing.T) {
	raw := "参考图 必须是公网 HTTPS 链接；不支持 base64/data URL（Magica 上游请求体上限约 4MB）"
	got, ok := normalizeMagicaRelay(true, ErrorContext{Model: "cy-sd7-seedance-2.0-720p", Raw: raw})
	if !ok {
		t.Fatal("expected normalization")
	}
	if got != "参考图必须是公网 HTTPS 链接，不支持 base64 或本地 data URL" {
		t.Fatalf("got %q", got)
	}
}
