package image

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
)

func TestDowngradeImageSize4KTo2K(t *testing.T) {
	tests := []struct {
		size string
		want string
		ok   bool
	}{
		{size: "3840x2160", want: "2048x1152", ok: true},
		{size: "2160x3840", want: "1152x2048", ok: true},
		{size: "2880x2880", want: "2048x2048", ok: true},
		{size: "16:9-4k", want: "16:9-2k", ok: true},
		{size: "9:16-4k", want: "9:16-2k", ok: true},
		{size: "2560x1440", want: "2048x1152", ok: true},
		{size: "1024x1024", want: "", ok: false},
		{size: "2048x2048", want: "", ok: false},
		{size: "1536x1024", want: "", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.size, func(t *testing.T) {
			got, ok := downgradeImageSize4KTo2K(tt.size)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestApplyChannelImageSizeDowngrade(t *testing.T) {
	request := &dto.ImageRequest{Size: "3840x2160"}

	if ApplyChannelImageSizeDowngrade(71, request) {
		t.Fatal("expected no downgrade for channel 71")
	}
	if request.Size != "3840x2160" {
		t.Fatalf("channel 71 size changed: %q", request.Size)
	}

	if !ApplyChannelImageSizeDowngrade(channelImageSize4KDowngradeID, request) {
		t.Fatal("expected downgrade for channel 72")
	}
	if request.Size != "2048x1152" {
		t.Fatalf("size = %q, want 2048x1152", request.Size)
	}
}
