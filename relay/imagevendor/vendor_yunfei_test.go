package imagevendor

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/require"
)

func TestIsYunfeiBananaOriginModel(t *testing.T) {
	require.True(t, IsYunfeiBananaOriginModel("cy-yf-gemini-banana-pro"))
	require.True(t, IsYunfeiBananaOriginModel("cy-yf-gemini-banana-flash"))
	require.False(t, IsYunfeiBananaOriginModel("manju-gemini-banana-pro-4k"))
	require.False(t, IsYunfeiBananaOriginModel("adobe-firefly-nano-banana-pro-4k"))
}

func TestPatchYunfeiGPTImage4KRequest(t *testing.T) {
	request := &dto.ImageRequest{
		Prompt: "test",
		Size:   "16:9",
		N:      uintPtr(2),
	}
	result, err := patchYunfeiGPTImage4KRequest(yunfeiGPTImage4KOrigin, request)
	require.NoError(t, err)
	require.True(t, result.OutboundBodyChanged)
	require.NotNil(t, request.N)
	require.Equal(t, uint(1), *request.N)
	require.NotEmpty(t, request.Size)
	require.NotContains(t, request.Size, ":")
}

func TestPatchYunfeiGPTImage4KRequestSkipsOtherModels(t *testing.T) {
	request := &dto.ImageRequest{N: uintPtr(2)}
	result, err := patchYunfeiGPTImage4KRequest("cy-img1-gpt-image-2", request)
	require.NoError(t, err)
	require.False(t, result.OutboundBodyChanged)
	require.Equal(t, uint(2), *request.N)
}

func TestResolveYunfeiRehostPolicy(t *testing.T) {
	policy := ResolveRehostPolicy(yunfeiGPTImage4KOrigin)
	require.True(t, policy.AcceptUpstreamURL)
	require.True(t, policy.AsyncPreferURLResponse)

	policy = ResolveRehostPolicy("cy-yf-gemini-banana-pro")
	require.True(t, policy.AcceptUpstreamURL)
}

func TestValidateYunfeiGPTImage4KRequestRejectsOverBudgetSize(t *testing.T) {
	info := &relaycommon.RelayInfo{OriginModelName: yunfeiGPTImage4KOrigin}
	err := validateYunfeiGPTImage4KRequest(nil, info, &dto.ImageRequest{
		Size: "8192x8192",
	})
	require.Error(t, err)
}

func uintPtr(v uint) *uint {
	return &v
}
