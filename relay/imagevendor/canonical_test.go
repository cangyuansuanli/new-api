package imagevendor

import "testing"

func TestCanonicalImageOptions(t *testing.T) {
	for input, want := range map[string]string{
		"16:9":      "16:9",
		"1024x1024": "1:1",
		"1920x1080": "16:9",
		"auto":      "",
	} {
		if got := CanonicalAspectRatio(input); got != want {
			t.Fatalf("CanonicalAspectRatio(%q) = %q, want %q", input, got, want)
		}
	}
	for input, want := range map[string]string{
		"low":    "1K",
		"medium": "2K",
		"high":   "4K",
		"auto":   "1K",
	} {
		if got := CanonicalResolution(input); got != want {
			t.Fatalf("CanonicalResolution(%q) = %q, want %q", input, got, want)
		}
	}
}
