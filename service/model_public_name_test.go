package service

import "testing"

func TestStripChannelRegistrationPrefix(t *testing.T) {
	tests := map[string]string{
		"gz-seedance-pro-720p-k":  "seedance-pro-720p-k",
		"yunwu-sora-2":            "sora-2",
		"oairegbox-omni-fast":     "omni-fast",
		"119337-grok-video":       "grok-video",
		"gpt-4o":                  "gpt-4o",
	}
	for input, want := range tests {
		if got := StripChannelRegistrationPrefix(input); got != want {
			t.Fatalf("%s => %s, want %s", input, got, want)
		}
	}
}

func TestResolveInternalModelNameRegistry(t *testing.T) {
	modelPublicRegistryMu.Lock()
	modelPublicRegistryData = modelPublicRegistry{
		internalSet: map[string]struct{}{
			"gz-seedance-pro-720p-k": {},
			"yunwu-sora-2":           {},
		},
		publicToInternals: map[string][]string{
			"seedance-pro-720p-k": {"gz-seedance-pro-720p-k"},
			"sora-2":              {"yunwu-sora-2"},
		},
		internalToPublic: map[string]string{
			"gz-seedance-pro-720p-k": "seedance-pro-720p-k",
			"yunwu-sora-2":           "sora-2",
		},
	}
	modelPublicRegistryReady = true
	modelPublicRegistryMu.Unlock()

	internal, public, err := ResolveInternalModelName("seedance-pro-720p-k")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if internal != "gz-seedance-pro-720p-k" || public != "seedance-pro-720p-k" {
		t.Fatalf("got %s %s", internal, public)
	}
}
