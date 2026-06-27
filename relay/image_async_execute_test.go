package relay

import (
	"encoding/base64"
	"testing"

	"github.com/QuantumNous/new-api/dto"
)

func TestDecodeImageDataItemDetectsJPEGFromB64(t *testing.T) {
	// minimal JPEG header bytes
	jpeg := []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 0x4a, 0x46, 0x49, 0x46}
	data, mime, err := decodeImageDataItem(dto.ImageData{B64Json: base64.StdEncoding.EncodeToString(jpeg)})
	if err != nil {
		t.Fatalf("decodeImageDataItem: %v", err)
	}
	if mime != "image/jpeg" {
		t.Fatalf("mime = %q, want image/jpeg", mime)
	}
	if len(data) != len(jpeg) {
		t.Fatalf("data len = %d, want %d", len(data), len(jpeg))
	}
}
