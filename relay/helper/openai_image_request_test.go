package helper

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/QuantumNous/new-api/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// TestGetAndValidOpenAIImageRequestMultipartStream verifies multipart image
// edit parsing: the stream field is parsed and validated, and the request body
// stays replayable for the upstream request.
func TestGetAndValidOpenAIImageRequestMultipartStream(t *testing.T) {
	gin.SetMode(gin.TestMode)

	newContext := func(t *testing.T, streamValue string, withImage bool) (*gin.Context, string) {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		require.NoError(t, writer.WriteField("model", "gpt-image-1"))
		require.NoError(t, writer.WriteField("prompt", "edit this image"))
		require.NoError(t, writer.WriteField("stream", streamValue))
		if withImage {
			part, err := writer.CreateFormFile("image", "input.png")
			require.NoError(t, err)
			_, err = part.Write([]byte("fake image"))
			require.NoError(t, err)
		}
		require.NoError(t, writer.Close())
		originalBody := body.String()

		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", &body)
		c.Request.Header.Set("Content-Type", writer.FormDataContentType())
		return c, originalBody
	}

	t.Run("valid stream value keeps body replayable", func(t *testing.T) {
		c, originalBody := newContext(t, "true", true)

		req, err := GetAndValidOpenAIImageRequest(c, relayconstant.RelayModeImagesEdits)
		require.NoError(t, err)
		require.NotNil(t, req.Stream)
		require.True(t, *req.Stream)
		require.True(t, req.IsStream(c))

		bodyAfterValidation, err := io.ReadAll(c.Request.Body)
		require.NoError(t, err)
		require.Equal(t, originalBody, string(bodyAfterValidation))

		form, err := common.ParseMultipartFormReusable(c)
		require.NoError(t, err)
		require.Equal(t, "true", url.Values(form.Value).Get("stream"))
		require.Len(t, form.File["image"], 1)
	})

	t.Run("invalid stream value is rejected", func(t *testing.T) {
		c, _ := newContext(t, "notabool", false)

		_, err := GetAndValidOpenAIImageRequest(c, relayconstant.RelayModeImagesEdits)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid stream value")
	})
}

func TestGetAndValidOpenAIImageRequestAcceptsJSONWithBareMultipartContentType(t *testing.T) {
	body := `{"model":"gpt-image-2-2k","prompt":"edit","size":"2048x2048","async":true,"imageUrls":["https://cdn.example.com/a.png"]}`
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "multipart/form-data")

	req, err := GetAndValidOpenAIImageRequest(c, relayconstant.RelayModeImagesGenerations)
	require.NoError(t, err)
	require.Equal(t, "application/json", c.Request.Header.Get("Content-Type"))
	require.Equal(t, "gpt-image-2-2k", req.Model)
	var imageURLs []string
	require.NoError(t, json.Unmarshal(req.Extra["imageUrls"], &imageURLs))
	require.Equal(t, []string{"https://cdn.example.com/a.png"}, imageURLs)
}

func TestGetAndValidOpenAIImageRequestRejectsActualMultipartWithoutBoundary(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", bytes.NewBufferString("--missing-boundary\r\n"))
	c.Request.Header.Set("Content-Type", "multipart/form-data")

	_, err := GetAndValidOpenAIImageRequest(c, relayconstant.RelayModeImagesEdits)
	require.EqualError(t, err, "multipart Content-Type must include a boundary")
}

func TestGetAndValidOpenAIImageRequestPreservesRepeatedURLReferences(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "gpt-image-2"))
	require.NoError(t, writer.WriteField("prompt", "edit these images"))
	require.NoError(t, writer.WriteField("image", "https://cdn.example.com/a.png"))
	require.NoError(t, writer.WriteField("image", "https://cdn.example.com/b.png"))
	require.NoError(t, writer.WriteField("mask", "https://cdn.example.com/mask.png"))
	require.NoError(t, writer.Close())

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", &body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())

	req, err := GetAndValidOpenAIImageRequest(c, relayconstant.RelayModeImagesEdits)
	require.NoError(t, err)
	var images []string
	require.NoError(t, common.Unmarshal(req.Image, &images))
	require.Equal(t, []string{"https://cdn.example.com/a.png", "https://cdn.example.com/b.png"}, images)
	var mask string
	require.NoError(t, common.Unmarshal(req.Mask, &mask))
	require.Equal(t, "https://cdn.example.com/mask.png", mask)
}
