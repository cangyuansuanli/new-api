package common

import (
	"encoding/json"
	"testing"

	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/require"
)

func TestRelayInfoGetFinalRequestRelayFormatPrefersExplicitFinal(t *testing.T) {
	info := &RelayInfo{
		RelayFormat:             types.RelayFormatOpenAI,
		RequestConversionChain:  []types.RelayFormat{types.RelayFormatOpenAI, types.RelayFormatClaude},
		FinalRequestRelayFormat: types.RelayFormatOpenAIResponses,
	}

	require.Equal(t, types.RelayFormat(types.RelayFormatOpenAIResponses), info.GetFinalRequestRelayFormat())
}

func TestRelayInfoGetFinalRequestRelayFormatFallsBackToConversionChain(t *testing.T) {
	info := &RelayInfo{
		RelayFormat:            types.RelayFormatOpenAI,
		RequestConversionChain: []types.RelayFormat{types.RelayFormatOpenAI, types.RelayFormatClaude},
	}

	require.Equal(t, types.RelayFormat(types.RelayFormatClaude), info.GetFinalRequestRelayFormat())
}

func TestRelayInfoGetFinalRequestRelayFormatFallsBackToRelayFormat(t *testing.T) {
	info := &RelayInfo{
		RelayFormat: types.RelayFormatGemini,
	}

	require.Equal(t, types.RelayFormat(types.RelayFormatGemini), info.GetFinalRequestRelayFormat())
}

func TestRelayInfoGetFinalRequestRelayFormatNilReceiver(t *testing.T) {
	var info *RelayInfo
	require.Equal(t, types.RelayFormat(""), info.GetFinalRequestRelayFormat())
}

func TestTaskSubmitReqUnmarshalInputReferenceString(t *testing.T) {
	var req TaskSubmitReq
	require.NoError(t, json.Unmarshal([]byte(`{
		"prompt": "test",
		"model": "omni-fast",
		"input_reference": "https://example.com/a.jpg"
	}`), &req))
	require.Equal(t, "https://example.com/a.jpg", req.InputReference)
	require.True(t, req.HasImage())
}

func TestTaskSubmitReqUnmarshalInputReferenceArray(t *testing.T) {
	var req TaskSubmitReq
	require.NoError(t, json.Unmarshal([]byte(`{
		"prompt": "test",
		"model": "omni-fast",
		"input_reference": ["https://example.com/a.jpg", "https://example.com/b.jpg"]
	}`), &req))
	require.Empty(t, req.InputReference)
	require.Equal(t, []string{
		"https://example.com/a.jpg",
		"https://example.com/b.jpg",
	}, req.Images)
	require.True(t, req.HasImage())
}

func TestTaskSubmitReqUnmarshalReferenceObject(t *testing.T) {
	var req TaskSubmitReq
	require.NoError(t, json.Unmarshal([]byte(`{
		"prompt": "test",
		"model": "grok-video-1.5",
		"image": {"url": "https://example.com/a.jpg"}
	}`), &req))
	require.Equal(t, "https://example.com/a.jpg", req.Image)
	require.Equal(t, []string{"https://example.com/a.jpg"}, req.Images)
}

func TestTaskSubmitReqUnmarshalInputReferenceObject(t *testing.T) {
	var req TaskSubmitReq
	require.NoError(t, json.Unmarshal([]byte(`{
		"prompt": "test",
		"model": "grok-video-1.5",
		"input_reference": {"url": "https://example.com/a.jpg"}
	}`), &req))
	require.Equal(t, "https://example.com/a.jpg", req.InputReference)
	require.Equal(t, []string{"https://example.com/a.jpg"}, req.Images)
}

func TestTaskSubmitReqNormalizesAllImageAliases(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"image string", `{"prompt":"test","image":"https://example.com/ref.jpg"}`},
		{"image object", `{"prompt":"test","image":{"url":"https://example.com/ref.jpg"}}`},
		{"image_url", `{"prompt":"test","image_url":"https://example.com/ref.jpg"}`},
		{"images", `{"prompt":"test","images":["https://example.com/ref.jpg"]}`},
		{"image_urls", `{"prompt":"test","image_urls":["https://example.com/ref.jpg"]}`},
		{"reference_images", `{"prompt":"test","reference_images":[{"url":"https://example.com/ref.jpg"}]}`},
		{"reference_image_urls", `{"prompt":"test","reference_image_urls":["https://example.com/ref.jpg"]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req TaskSubmitReq
			require.NoError(t, json.Unmarshal([]byte(tt.body), &req))
			require.Equal(t, []string{"https://example.com/ref.jpg"}, req.Images)
		})
	}
}

func TestTaskSubmitReqCombinesAndDeduplicatesImageAliases(t *testing.T) {
	var req TaskSubmitReq
	require.NoError(t, json.Unmarshal([]byte(`{
		"prompt":"test",
		"images":["https://example.com/a.jpg","https://example.com/b.jpg"],
		"image_url":"https://example.com/a.jpg",
		"reference_image_urls":["https://example.com/c.jpg"]
	}`), &req))
	require.Equal(t, []string{
		"https://example.com/a.jpg",
		"https://example.com/b.jpg",
		"https://example.com/c.jpg",
	}, req.Images)
}
