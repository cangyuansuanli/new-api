package ratio_setting

import "testing"

func TestConfiguredCompletionRatioOverridesBuiltInFallback(t *testing.T) {
	original := CompletionRatio2JSONString()
	t.Cleanup(func() {
		if err := UpdateCompletionRatioByJSONString(original); err != nil {
			t.Fatalf("restore completion ratios: %v", err)
		}
	})

	if err := UpdateCompletionRatioByJSONString(`{"gpt-5.6-luna":6}`); err != nil {
		t.Fatalf("update completion ratios: %v", err)
	}

	if got := GetCompletionRatio("gpt-5.6-luna"); got != 6 {
		t.Fatalf("GetCompletionRatio() = %v, want 6", got)
	}
	if got := GetCompletionRatioInfo("gpt-5.6-luna"); got.Ratio != 6 || got.Locked {
		t.Fatalf("GetCompletionRatioInfo() = %+v, want ratio 6 unlocked", got)
	}
}

func TestUnconfiguredCompletionRatioUsesBuiltInFallback(t *testing.T) {
	original := CompletionRatio2JSONString()
	t.Cleanup(func() {
		if err := UpdateCompletionRatioByJSONString(original); err != nil {
			t.Fatalf("restore completion ratios: %v", err)
		}
	})

	if err := UpdateCompletionRatioByJSONString(`{}`); err != nil {
		t.Fatalf("clear completion ratios: %v", err)
	}

	if got := GetCompletionRatio("gpt-5.6-unconfigured"); got != 8 {
		t.Fatalf("GetCompletionRatio() = %v, want fallback 8", got)
	}
	if got := GetCompletionRatioInfo("gpt-5.6-unconfigured"); got.Ratio != 8 || !got.Locked {
		t.Fatalf("GetCompletionRatioInfo() = %+v, want fallback ratio 8 locked", got)
	}
}

func TestConfiguredCompletionRatioOverridesNonGPTBuiltInFallback(t *testing.T) {
	original := CompletionRatio2JSONString()
	t.Cleanup(func() {
		if err := UpdateCompletionRatioByJSONString(original); err != nil {
			t.Fatalf("restore completion ratios: %v", err)
		}
	})

	if err := UpdateCompletionRatioByJSONString(`{"claude-sonnet-4-test":7}`); err != nil {
		t.Fatalf("update completion ratios: %v", err)
	}

	if got := GetCompletionRatio("claude-sonnet-4-test"); got != 7 {
		t.Fatalf("GetCompletionRatio() = %v, want 7", got)
	}
	if got := GetCompletionRatioInfo("claude-sonnet-4-test"); got.Ratio != 7 || got.Locked {
		t.Fatalf("GetCompletionRatioInfo() = %+v, want ratio 7 unlocked", got)
	}
}
