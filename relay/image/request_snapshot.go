package image

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

const ImageRequestSnapshotVersion = 1

type RequestSnapshotKind string

const (
	RequestSnapshotGenerationJSON RequestSnapshotKind = "image.generation.json"
	RequestSnapshotEditMultipart  RequestSnapshotKind = "image.edit.multipart"
	RequestSnapshotLegacyChatJSON RequestSnapshotKind = "image.legacy-chat.json"
)

// RequestSnapshot is the only durable envelope written by new image tasks.
// Body and Multipart differ because JSON and uploaded-file requests have
// different replay needs, but version/kind/method/path/content type are stable.
type RequestSnapshot struct {
	Version     int                 `json:"version"`
	Kind        RequestSnapshotKind `json:"kind"`
	Method      string              `json:"method"`
	Path        string              `json:"path"`
	ContentType string              `json:"content_type"`
	Body        json.RawMessage     `json:"body,omitempty"`
	Multipart   *EditPayload        `json:"multipart,omitempty"`
}

func NewJSONRequestSnapshot(kind RequestSnapshotKind, path string, body []byte) ([]byte, error) {
	if kind != RequestSnapshotGenerationJSON && kind != RequestSnapshotLegacyChatJSON {
		return nil, fmt.Errorf("unsupported JSON image snapshot kind %q", kind)
	}
	var decoded any
	if err := common.Unmarshal(body, &decoded); err != nil {
		return nil, fmt.Errorf("invalid image snapshot JSON: %w", err)
	}
	snapshot := RequestSnapshot{
		Version:     ImageRequestSnapshotVersion,
		Kind:        kind,
		Method:      http.MethodPost,
		Path:        normalizeImageSnapshotPath(path),
		ContentType: "application/json",
		Body:        append(json.RawMessage(nil), body...),
	}
	if err := snapshot.validate(); err != nil {
		return nil, err
	}
	return common.Marshal(snapshot)
}

func NewEditRequestSnapshot(payload EditPayload) ([]byte, error) {
	snapshot := RequestSnapshot{
		Version:     ImageRequestSnapshotVersion,
		Kind:        RequestSnapshotEditMultipart,
		Method:      http.MethodPost,
		Path:        "/v1/images/edits",
		ContentType: "multipart/form-data",
		Multipart:   &payload,
	}
	return common.Marshal(snapshot)
}

// DecodeRequestSnapshot reads the versioned envelope and converts pre-v1 task
// rows in memory. Compatibility never writes the legacy shape back to storage.
func DecodeRequestSnapshot(data []byte, legacyPath string) (RequestSnapshot, error) {
	if len(data) == 0 {
		return RequestSnapshot{}, fmt.Errorf("empty request snapshot")
	}
	var snapshot RequestSnapshot
	if err := common.Unmarshal(data, &snapshot); err == nil && snapshot.Version != 0 {
		if err := snapshot.validate(); err != nil {
			return RequestSnapshot{}, err
		}
		return snapshot, nil
	}

	path := normalizeImageSnapshotPath(legacyPath)
	snapshot = RequestSnapshot{
		Method: http.MethodPost,
		Path:   path,
	}
	switch {
	case strings.Contains(path, "/edits"):
		var payload EditPayload
		if err := common.Unmarshal(data, &payload); err != nil {
			return RequestSnapshot{}, fmt.Errorf("decode legacy edit snapshot: %w", err)
		}
		snapshot.Kind = RequestSnapshotEditMultipart
		snapshot.ContentType = "multipart/form-data"
		snapshot.Multipart = &payload
	case strings.Contains(path, "/chat/completions"):
		snapshot.Kind = RequestSnapshotLegacyChatJSON
		snapshot.ContentType = "application/json"
		snapshot.Body = append(json.RawMessage(nil), data...)
	default:
		snapshot.Kind = RequestSnapshotGenerationJSON
		snapshot.ContentType = "application/json"
		snapshot.Body = append(json.RawMessage(nil), data...)
	}
	return snapshot, nil
}

func (snapshot RequestSnapshot) validate() error {
	if snapshot.Version != ImageRequestSnapshotVersion {
		return fmt.Errorf("unsupported image request snapshot version %d", snapshot.Version)
	}
	if snapshot.Method != http.MethodPost {
		return fmt.Errorf("unsupported image snapshot method %q", snapshot.Method)
	}
	switch snapshot.Kind {
	case RequestSnapshotGenerationJSON:
		if snapshot.Path != "/v1/images/generations" || snapshot.ContentType != "application/json" || len(snapshot.Body) == 0 || snapshot.Multipart != nil {
			return fmt.Errorf("invalid generation request snapshot")
		}
	case RequestSnapshotEditMultipart:
		if snapshot.Path != "/v1/images/edits" || snapshot.ContentType != "multipart/form-data" || snapshot.Multipart == nil || len(snapshot.Body) != 0 {
			return fmt.Errorf("invalid edit request snapshot")
		}
	case RequestSnapshotLegacyChatJSON:
		if snapshot.Path != "/v1/chat/completions" || snapshot.ContentType != "application/json" || len(snapshot.Body) == 0 || snapshot.Multipart != nil {
			return fmt.Errorf("invalid legacy chat request snapshot")
		}
	default:
		return fmt.Errorf("unsupported image request snapshot kind %q", snapshot.Kind)
	}
	return nil
}

func normalizeImageSnapshotPath(path string) string {
	path = strings.TrimSpace(path)
	switch {
	case strings.Contains(path, "/chat/completions"):
		return "/v1/chat/completions"
	case strings.Contains(path, "/edits"):
		return "/v1/images/edits"
	default:
		return "/v1/images/generations"
	}
}
