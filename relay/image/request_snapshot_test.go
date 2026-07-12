package image

import (
	"context"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

type mediaContractFixture struct {
	Version int                 `json:"version"`
	Cases   []mediaContractCase `json:"cases"`
}

type mediaContractCase struct {
	Name         string              `json:"name"`
	SnapshotKind RequestSnapshotKind `json:"snapshot_kind"`
	Method       string              `json:"method"`
	ClientPath   string              `json:"client_path"`
	APIPath      string              `json:"api_path"`
	ContentType  string              `json:"content_type"`
	ClientBody   map[string]any      `json:"client_body"`
	WorkerBody   map[string]any      `json:"worker_body"`
	ClientFields map[string]string   `json:"client_fields"`
	WorkerFields map[string]string   `json:"worker_fields"`
	Files        []mediaContractFile `json:"files"`
}

type mediaContractFile struct {
	Field string `json:"field"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	Size  int    `json:"size"`
}

func loadMediaContractFixture(t *testing.T) mediaContractFixture {
	t.Helper()
	data, err := os.ReadFile("../../contracts/media-request-snapshots.json")
	require.NoError(t, err)
	var fixture mediaContractFixture
	require.NoError(t, common.Unmarshal(data, &fixture))
	require.Equal(t, ImageRequestSnapshotVersion, fixture.Version)
	return fixture
}

func TestMediaRequestContractSnapshots(t *testing.T) {
	fixture := loadMediaContractFixture(t)
	for _, contract := range fixture.Cases {
		contract := contract
		t.Run(contract.Name, func(t *testing.T) {
			require.Equal(t, http.MethodPost, contract.Method)
			switch contract.SnapshotKind {
			case RequestSnapshotGenerationJSON:
				assertGenerationContract(t, contract)
			case RequestSnapshotEditMultipart:
				assertEditContract(t, contract)
			default:
				t.Fatalf("unsupported contract kind %q", contract.SnapshotKind)
			}
		})
	}
}

func assertGenerationContract(t *testing.T, contract mediaContractCase) {
	clientBody, err := common.Marshal(contract.ClientBody)
	require.NoError(t, err)
	encoded, err := NewJSONRequestSnapshot(contract.SnapshotKind, contract.APIPath, clientBody)
	require.NoError(t, err)
	decoded, err := DecodeRequestSnapshot(encoded, "")
	require.NoError(t, err)
	require.Equal(t, contract.APIPath, decoded.Path)
	require.Equal(t, contract.ContentType, decoded.ContentType)
	var storedBody map[string]any
	require.NoError(t, common.Unmarshal(decoded.Body, &storedBody))
	require.Equal(t, contract.ClientBody, storedBody)

	task := contractTask(encoded, contract.APIPath)
	req, _, err := buildHTTPRequestForImageTask(context.Background(), task)
	require.NoError(t, err)
	defer req.Body.Close()
	body, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	var replayed map[string]any
	require.NoError(t, common.Unmarshal(body, &replayed))
	require.Equal(t, contract.WorkerBody, replayed)
}

func assertEditContract(t *testing.T, contract mediaContractCase) {
	files := make([]EditFile, 0, len(contract.Files))
	for _, file := range contract.Files {
		files = append(files, EditFile{
			Field:       file.Field,
			Filename:    file.Name,
			ContentType: file.Type,
			Data:        make([]byte, file.Size),
		})
	}
	encoded, err := NewEditRequestSnapshot(EditPayload{Fields: contract.ClientFields, Files: files})
	require.NoError(t, err)
	decoded, err := DecodeRequestSnapshot(encoded, "")
	require.NoError(t, err)
	require.Equal(t, contract.APIPath, decoded.Path)
	require.Equal(t, contract.ContentType, decoded.ContentType)
	require.Equal(t, contract.ClientFields, decoded.Multipart.Fields)

	task := contractTask(encoded, contract.APIPath)
	req, _, err := buildHTTPRequestForImageTask(context.Background(), task)
	require.NoError(t, err)
	defer req.Body.Close()
	require.NoError(t, req.ParseMultipartForm(1<<20))
	actualFields := make(map[string]string, len(req.MultipartForm.Value))
	for key, values := range req.MultipartForm.Value {
		if len(values) > 0 {
			actualFields[key] = values[0]
		}
	}
	require.Equal(t, contract.WorkerFields, actualFields)
	for _, expected := range contract.Files {
		actual := req.MultipartForm.File[expected.Field]
		require.Len(t, actual, 1)
		require.Equal(t, expected.Name, actual[0].Filename)
		require.Equal(t, expected.Type, actual[0].Header.Get("Content-Type"))
		require.Equal(t, int64(expected.Size), actual[0].Size)
	}
}

func contractTask(snapshot []byte, path string) *model.Task {
	return &model.Task{
		TaskID: "task_contract",
		Properties: model.Properties{
			OriginModelName: "go2api-gpt-image-2-1k",
		},
		PrivateData: model.TaskPrivateData{
			RequestPath:     path,
			RequestSnapshot: snapshot,
		},
	}
}

func TestDecodeRequestSnapshotSupportsLegacyRows(t *testing.T) {
	generation, err := DecodeRequestSnapshot([]byte(`{"model":"legacy"}`), "/v1/images/generations")
	require.NoError(t, err)
	require.Equal(t, RequestSnapshotGenerationJSON, generation.Kind)
	require.Zero(t, generation.Version)

	legacyEdit, err := common.Marshal(EditPayload{Fields: map[string]string{"model": "legacy"}})
	require.NoError(t, err)
	edit, err := DecodeRequestSnapshot(legacyEdit, "/v1/images/edits")
	require.NoError(t, err)
	require.Equal(t, RequestSnapshotEditMultipart, edit.Kind)
	require.Zero(t, edit.Version)

	chat, err := DecodeRequestSnapshot([]byte(`{"model":"legacy-chat"}`), "/v1/chat/completions")
	require.NoError(t, err)
	require.Equal(t, RequestSnapshotLegacyChatJSON, chat.Kind)
}

func TestDecodeRequestSnapshotRejectsUnknownVersion(t *testing.T) {
	_, err := DecodeRequestSnapshot([]byte(`{"version":2,"kind":"image.generation.json","method":"POST","path":"/v1/images/generations","content_type":"application/json","body":{}}`), "")
	require.ErrorContains(t, err, "unsupported image request snapshot version")
}
