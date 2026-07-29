package imagevendor

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/require"
)

func greek2APITestRelay(channelID int, model string) *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		OriginModelName: model,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelId: channelID,
		},
	}
}

func TestGreek2APIPatchConvertsRatiosAtOutboundBoundary(t *testing.T) {
	tests := []struct {
		model string
		size  string
		want  string
	}{
		{"cy-img2-gpt-image-2-1k", "16:9", "1280x720"},
		{"cy-img2-gpt-image-2-2k", "16:9", "2560x1440"},
		{"cy-img2-gpt-image-2-4k", "16:9", "3840x2160"},
		{"cy-img2-gpt-image-2-1k", "auto", "1024x1024"},
		{"cy-img2-gpt-image-2-2k", "2048x2048", "2048x2048"},
	}
	for _, test := range tests {
		request := &dto.ImageRequest{Size: test.size}
		result, err := ApplyRequestPatch(greek2APITestRelay(greek2APIChannelID, test.model), request)
		require.NoError(t, err)
		require.Equal(t, test.want, request.Size)
		require.True(t, result.OutboundBodyChanged)
		require.True(t, result.SyncSizeToMultipart)
	}
}

func TestGreek2APIPatchRequiresChannel48(t *testing.T) {
	request := &dto.ImageRequest{Size: "16:9"}
	result, err := ApplyRequestPatch(greek2APITestRelay(72, "cy-img2-gpt-image-2-2k"), request)
	require.NoError(t, err)
	require.Equal(t, "16:9", request.Size)
	require.False(t, result.OutboundBodyChanged)
}

func TestGreek2APIValidationRejectsTierBypass(t *testing.T) {
	info := greek2APITestRelay(greek2APIChannelID, "cy-img2-gpt-image-2-1k")
	err := ValidateRequest(nil, info, &dto.ImageRequest{Size: "2048x2048"})
	require.ErrorContains(t, err, "1K")
}

func TestGreek2APIValidationIgnoresSameModelOnAnotherChannel(t *testing.T) {
	info := greek2APITestRelay(72, "cy-img2-gpt-image-2-1k")
	require.NoError(t, ValidateRequest(nil, info, &dto.ImageRequest{Size: "2048x2048"}))
}
