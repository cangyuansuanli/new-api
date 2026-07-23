package registry

import "testing"

func TestResolve(t *testing.T) {
	cases := []struct {
		origin   string
		upstream string
		want     Vendor
	}{
		{"manju-openai-sora2", "sora2", VendorManju},
		{"cy-sd1-seedance-2.0-fast-720p", "Seedance-2.0-720p", VendorSeedanceOairegbox},
		{"cy-sd4-seedance-2.0", "seedance-2.0", VendorSeedanceLeonardo},
		{"cy-sd5-seedance-2.0", "cy-sd5-seedance-2.0", VendorSD5},
		{"cy-sd5-seedance-2.0-fast", "cy-sd5-seedance-2.0-fast", VendorSD5},
		{"cy-sd2-seedance-2.0", "manxue-2.0", VendorSeedanceTengda},
		{"tengd-seedance-2.0", "manxue-2.0", VendorSeedanceTengda},
		{"cy-vid2-sora-2", "cy-vid2-sora-2", VendorChat},
		{"cy-sd1-grok-video", "grok-imagine-video", VendorChat},
		{"cy-sd1-omni-fast", "omni-fast", VendorOmniI2V},
		{"cy-sd1-omni-v2v", "omni-fast-v2v", VendorOmniV2V},
		{"cy-gv1-grok-video", "grok-image-video", VendorGrok},
		{"cy-gv1-grok-video-1.5", "grok-video-1.5", VendorGrok},
		{"cy-gv1-grok-video", "grok-imagine-video", VendorGeeknowGrok},
		{"cy-gv1-grok-video-1.5", "grok-imagine-video-1.5-preview", VendorGeeknowGrok},
		{"sora-2", "sora-2", VendorSora},
		{"grok-video", "grok-video", VendorSora},
	}
	for _, tc := range cases {
		if got := Resolve(tc.origin, tc.upstream); got != tc.want {
			t.Fatalf("Resolve(%q,%q)=%q want %q", tc.origin, tc.upstream, got, tc.want)
		}
	}
}

func TestResolveWithChannelKeepsSD5SeparateFromAdobe(t *testing.T) {
	if got := ResolveWithChannel("cy-sd5-seedance-2.0", "cy-sd5-seedance-2.0", 86, "http://45.67.221.45:6002"); got != VendorSD5 {
		t.Fatalf("SD5 route = %q, want %q", got, VendorSD5)
	}
	if got := ResolveWithChannel("adobe-sora2", "sora2", 75, "http://45.67.221.45:6001"); got != VendorAdobe {
		t.Fatalf("Adobe route = %q, want %q", got, VendorAdobe)
	}
}

func TestResolve_CySd1NotTengda(t *testing.T) {
	if Resolve("cy-sd1-seedance-2.0-720p", "manxue-2.0") != VendorSeedanceOairegbox {
		t.Fatal("cy-sd1 should resolve to seedance-oairegbox even with tengda upstream name")
	}
}
