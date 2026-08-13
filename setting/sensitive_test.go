package setting

import "testing"

func TestShouldCheckPromptSensitiveForUser_WhitelistSkipsLocalBlock(t *testing.T) {
	prevGlobal := LocalSensitivePromptBlockEnabled
	prevEnabled := CheckSensitiveEnabled
	prevPrompt := CheckSensitiveOnPromptEnabled
	prevWhitelist := SensitiveReviewWhitelistUserIds
	t.Cleanup(func() {
		LocalSensitivePromptBlockEnabled = prevGlobal
		CheckSensitiveEnabled = prevEnabled
		CheckSensitiveOnPromptEnabled = prevPrompt
		SensitiveReviewWhitelistUserIds = prevWhitelist
	})

	LocalSensitivePromptBlockEnabled = true
	CheckSensitiveEnabled = true
	CheckSensitiveOnPromptEnabled = true
	SensitiveReviewWhitelistUserIds = map[int]struct{}{42: {}}

	if !ShouldCheckPromptSensitiveForUser(1, SensitivePromptScopeImage) {
		t.Fatal("expected non-whitelist user to be checked when global block enabled")
	}
	if ShouldCheckPromptSensitiveForUser(42, SensitivePromptScopeImage) {
		t.Fatal("expected whitelist user to skip local check")
	}

	LocalSensitivePromptBlockEnabled = false
	if ShouldCheckPromptSensitiveForUser(1, SensitivePromptScopeImage) {
		t.Fatal("expected non-whitelist user to skip check when global block disabled")
	}
	if ShouldCheckPromptSensitiveForUser(42, SensitivePromptScopeImage) {
		t.Fatal("expected whitelist user to skip local check when global block disabled")
	}
}

func TestShouldCheckPromptSensitiveForUser_AudioScopeMatchesImage(t *testing.T) {
	prevGlobal := LocalSensitivePromptBlockEnabled
	prevEnabled := CheckSensitiveEnabled
	prevPrompt := CheckSensitiveOnPromptEnabled
	t.Cleanup(func() {
		LocalSensitivePromptBlockEnabled = prevGlobal
		CheckSensitiveEnabled = prevEnabled
		CheckSensitiveOnPromptEnabled = prevPrompt
	})

	LocalSensitivePromptBlockEnabled = true
	CheckSensitiveEnabled = true
	CheckSensitiveOnPromptEnabled = true

	if !ShouldCheckPromptSensitiveForUser(1, SensitivePromptScopeAudio) {
		t.Fatal("expected audio scope to use local prompt block")
	}
}

func TestShouldCheckPromptSensitiveForUser_NonMediaScopeNeverChecked(t *testing.T) {
	prevGlobal := LocalSensitivePromptBlockEnabled
	prevEnabled := CheckSensitiveEnabled
	prevPrompt := CheckSensitiveOnPromptEnabled
	prevWhitelist := SensitiveReviewWhitelistUserIds
	t.Cleanup(func() {
		LocalSensitivePromptBlockEnabled = prevGlobal
		CheckSensitiveEnabled = prevEnabled
		CheckSensitiveOnPromptEnabled = prevPrompt
		SensitiveReviewWhitelistUserIds = prevWhitelist
	})

	LocalSensitivePromptBlockEnabled = true
	CheckSensitiveEnabled = true
	CheckSensitiveOnPromptEnabled = true
	SensitiveReviewWhitelistUserIds = map[int]struct{}{}

	// 文本/视频等非媒体 scope 恒为 false；上游负责审查。
	if ShouldCheckPromptSensitiveForUser(1, SensitivePromptScope(-1)) {
		t.Fatal("expected unknown scope to skip local check")
	}
}

func TestSensitiveReviewWhitelistFromString(t *testing.T) {
	prev := SensitiveReviewWhitelistUserIds
	t.Cleanup(func() {
		SensitiveReviewWhitelistUserIds = prev
	})

	SensitiveReviewWhitelistFromString("1\n2, 3\t4")
	if !IsSensitiveReviewWhitelistUser(1) || !IsSensitiveReviewWhitelistUser(2) || !IsSensitiveReviewWhitelistUser(3) || !IsSensitiveReviewWhitelistUser(4) {
		t.Fatalf("unexpected whitelist: %#v", SensitiveReviewWhitelistUserIds)
	}
	if IsSensitiveReviewWhitelistUser(5) {
		t.Fatal("user 5 should not be whitelisted")
	}
}
