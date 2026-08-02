package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relay/image"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSnapshotAsyncImageRequestSupportsJSONEdits(t *testing.T) {
	body := `{"model":"gpt-image-2-2k","prompt":"edit","images":["https://cdn.example.com/reference.png"],"async":true}`
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	storage, err := common.CreateBodyStorage([]byte(body))
	require.NoError(t, err)
	c.Set(common.KeyBodyStorage, storage)
	t.Cleanup(func() { require.NoError(t, storage.Close()) })

	snapshot, path, err := snapshotAsyncImageRequest(c, relayconstant.RelayModeImagesEdits, "task_json_edit")
	require.NoError(t, err)
	require.Equal(t, "/v1/images/edits", path)
	decoded, err := image.DecodeRequestSnapshot(snapshot, path)
	require.NoError(t, err)
	require.Equal(t, image.RequestSnapshotEditJSON, decoded.Kind)
	require.JSONEq(t, body, string(decoded.Body))
}
