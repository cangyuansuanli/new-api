package service

import (
	"fmt"
	"io"
	"strings"
	"testing"
)

func TestRedactImageTransferErrorRemovesUpstreamURL(t *testing.T) {
	err := redactImageTransferError(fmt.Errorf(`Get "https://oss5.yunfei.best/generated/a.png?token=secret": timeout`))
	if strings.Contains(err.Error(), "yunfei.best") || strings.Contains(err.Error(), "token=secret") {
		t.Fatalf("upstream URL leaked: %v", err)
	}
	if !strings.Contains(err.Error(), "[upstream-url-redacted]") {
		t.Fatalf("missing redaction marker: %v", err)
	}
}

func TestImageTaskInputBodyReleasesSlotOnce(t *testing.T) {
	releases := 0
	body := &imageTaskInputBody{
		ReadCloser: io.NopCloser(strings.NewReader("image")),
		release:    func() { releases++ },
	}
	if err := body.Close(); err != nil {
		t.Fatal(err)
	}
	if err := body.Close(); err != nil {
		t.Fatal(err)
	}
	if releases != 1 {
		t.Fatalf("releases = %d, want 1", releases)
	}
}
