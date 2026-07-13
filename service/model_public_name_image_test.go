package service

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestExtractInboundModelNameNormalizesJSONImageRequestWithBareMultipartHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := `{"model":"gpt-image-2-2k","prompt":"test","async":true}`
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "multipart/form-data")

	modelName, source, err := extractInboundModelName(c)
	require.NoError(t, err)
	require.Equal(t, "gpt-image-2-2k", modelName)
	require.Equal(t, "json", source)
	require.Equal(t, "application/json", c.Request.Header.Get("Content-Type"))
}

func TestExtractInboundModelNameKeepsInvalidMultipartImageRequestRejected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", strings.NewReader("--missing-boundary\r\n"))
	c.Request.Header.Set("Content-Type", "multipart/form-data")

	_, _, err := extractInboundModelName(c)
	require.ErrorContains(t, err, "multipart boundary not found")
}
