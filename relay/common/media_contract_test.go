package common

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestMediaContractVideoCanonicalFields(t *testing.T) {
	t.Parallel()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	path := filepath.Join(filepath.Dir(file), "..", "..", "testdata", "media-contract", "canonical-payloads.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read contract: %v", err)
	}
	var doc struct {
		Video struct {
			CanonicalJSON map[string]any `json:"canonical_json"`
		} `json:"video"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal contract: %v", err)
	}

	body, err := json.Marshal(doc.Video.CanonicalJSON)
	if err != nil {
		t.Fatalf("marshal canonical body: %v", err)
	}
	var req TaskSubmitReq
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatalf("TaskSubmitReq unmarshal: %v", err)
	}
	if req.Model == "" || strings.TrimSpace(req.GetPrompt()) == "" {
		t.Fatalf("model/prompt required: %#v", req)
	}
	if len(req.Images) == 0 {
		t.Fatalf("reference_image_urls should normalize to Images: %#v", req)
	}
	if req.RequestedDurationSeconds() != 8 {
		t.Fatalf("duration = %d, want 8", req.RequestedDurationSeconds())
	}
}
