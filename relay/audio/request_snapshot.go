package audio

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

const requestSnapshotVersion = 1

type requestSnapshot struct {
	Version     int             `json:"version"`
	Kind        string          `json:"kind"`
	Method      string          `json:"method"`
	Path        string          `json:"path"`
	ContentType string          `json:"content_type"`
	Body        json.RawMessage `json:"body"`
}

func NewJSONRequestSnapshot(path string, body []byte) ([]byte, error) {
	if len(body) == 0 {
		return nil, fmt.Errorf("empty audio snapshot body")
	}
	snapshot := requestSnapshot{
		Version:     requestSnapshotVersion,
		Kind:        "audio.generation.json",
		Method:      http.MethodPost,
		Path:        normalizeSnapshotPath(path),
		ContentType: "application/json",
		Body:        append(json.RawMessage(nil), body...),
	}
	return common.Marshal(snapshot)
}

func decodeRequestSnapshot(data []byte, legacyPath string) (requestSnapshot, error) {
	if len(data) == 0 {
		return requestSnapshot{}, fmt.Errorf("empty request snapshot")
	}
	var snapshot requestSnapshot
	if err := common.Unmarshal(data, &snapshot); err == nil && snapshot.Version != 0 {
		if len(snapshot.Body) == 0 {
			return requestSnapshot{}, fmt.Errorf("empty audio snapshot body")
		}
		return snapshot, nil
	}
	path := legacyPath
	if path == "" {
		path = "/v1/audio/generations"
	}
	return requestSnapshot{
		Version:     requestSnapshotVersion,
		Kind:        "audio.generation.json",
		Method:      http.MethodPost,
		Path:        normalizeSnapshotPath(path),
		ContentType: "application/json",
		Body:        append(json.RawMessage(nil), data...),
	}, nil
}

func normalizeSnapshotPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/v1/audio/generations"
	}
	return path
}

func buildHTTPRequestForAudioTask(ctx context.Context, task *model.Task) (*http.Request, error) {
	snapshot, err := decodeRequestSnapshot(task.PrivateData.RequestSnapshot, task.PrivateData.RequestPath)
	if err != nil {
		return nil, err
	}
	body := normalizeAsyncGenerationBody(snapshot.Body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, snapshot.Path, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func normalizeAsyncGenerationBody(body []byte) []byte {
	var raw map[string]json.RawMessage
	if err := common.Unmarshal(body, &raw); err != nil {
		return body
	}
	delete(raw, "async")
	raw["stream"] = json.RawMessage("false")
	if _, ok := raw["response_format"]; !ok {
		raw["response_format"] = json.RawMessage("\"url\"")
	}
	out, err := common.Marshal(raw)
	if err != nil {
		return body
	}
	return out
}
