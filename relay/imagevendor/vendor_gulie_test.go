package imagevendor

import (
	"encoding/json"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/require"
)

func TestPatchGulieImageRequest(t *testing.T) {
	request := &dto.ImageRequest{
		Model:             "gpt-image-2",
		Prompt:            "test",
		Size:              "1536x1024",
		Quality:           "high",
		Background:        json.RawMessage(`"transparent"`),
		OutputFormat:      json.RawMessage(`"png"`),
		OutputCompression: json.RawMessage(`80`),
		Moderation:        json.RawMessage(`"low"`),
	}

	result, err := patchGulieImageRequest("cy-img1-gpt-image-2", request)
	require.NoError(t, err)
	require.True(t, result.SuppressQualityLog)
	require.Equal(t, "1536x1024", request.Size)
	require.Empty(t, request.Quality)
	require.Nil(t, request.Background)
	require.Nil(t, request.OutputFormat)
	require.Nil(t, request.OutputCompression)
	require.Nil(t, request.Moderation)
	require.NotNil(t, request.Stream)
	require.False(t, *request.Stream)
}

func TestPatchGulieImageRequestSkipsNonGulieInternal(t *testing.T) {
	request := &dto.ImageRequest{Quality: "high", Size: "1024x1024"}
	result, err := patchGulieImageRequest("go2api-gpt-image-2-1k", request)
	require.NoError(t, err)
	require.False(t, result.SuppressQualityLog)
	require.Equal(t, "high", request.Quality)
}

func TestPatchGulieImageRequestSkipsCyImg2FourK(t *testing.T) {
	request := &dto.ImageRequest{Quality: "high", Size: "3840x2160"}
	result, err := patchGulieImageRequest("cy-img2-gpt-image-2-4k", request)
	require.NoError(t, err)
	require.False(t, result.SuppressQualityLog)
	require.Equal(t, "high", request.Quality)
}

func TestPatchGulieImageRequestAutoSize(t *testing.T) {
	request := &dto.ImageRequest{Size: "auto"}
	_, err := patchGulieImageRequest("gulie-gpt-image-2", request)
	require.NoError(t, err)
	require.Empty(t, request.Size)
}

func TestApplyRequestPatchKedaya(t *testing.T) {
	request := &dto.ImageRequest{
		Model:  "kedaya-gpt-image-2",
		Prompt: "cat",
		Size:   "800x600",
	}
	result, err := ApplyRequestPatch(&relaycommon.RelayInfo{OriginModelName: "kedaya-gpt-image-2"}, request)
	require.NoError(t, err)
	require.True(t, result.SuppressQualityLog)
	require.Equal(t, "800x600", result.LogSize)
	require.Contains(t, request.Prompt, "尺寸：800*600")
}
