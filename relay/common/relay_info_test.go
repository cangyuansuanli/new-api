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
