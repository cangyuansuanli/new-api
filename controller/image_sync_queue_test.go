package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func imageSyncQueueTestContext(t *testing.T, body string, contentType string) *gin.Context {
	t.Helper()
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", contentType)
	storage, err := common.CreateBodyStorage([]byte(body))
	require.NoError(t, err)
	c.Set(common.KeyBodyStorage, storage)
	t.Cleanup(func() { storage.Close() })
	return c
}

func TestSyncImageRequestsB64JSON(t *testing.T) {
	c := imageSyncQueueTestContext(t, `{"model":"gpt-image-2","response_format":"b64_json"}`, "application/json")
	require.True(t, syncImageRequestsB64JSON(c))
}

func TestSyncImageDefaultResponseKeepsLegacyContract(t *testing.T) {
	c := imageSyncQueueTestContext(t, `{"model":"gpt-image-2"}`, "application/json")
	require.False(t, syncImageRequestsB64JSON(c))
	require.False(t, shouldRunSyncImageViaQueue(c))
}

func TestSyncImageURLResponseCanUseQueue(t *testing.T) {
	c := imageSyncQueueTestContext(t, `{"model":"gpt-image-2","response_format":"url"}`, "application/json")
	require.True(t, shouldRunSyncImageViaQueue(c))
}
