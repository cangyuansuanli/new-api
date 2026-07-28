package image

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdobeDirectChannelIDsAreConfigurable(t *testing.T) {
	t.Setenv("IMAGE_ASYNC_ADOBE_CHANNEL_IDS", "75, 81;75 invalid")
	require.Equal(t, []int{75, 81}, AdobeDirectChannelIDs())
	require.True(t, IsAdobeDirectChannel(75))
	require.True(t, IsAdobeDirectChannel(81))
	require.False(t, IsAdobeDirectChannel(77))
}

func TestEffectiveImageWorkerConcurrencyFoldsLegacyAdobeLane(t *testing.T) {
	t.Setenv("IMAGE_ASYNC_MAX_CONCURRENT", "32")
	t.Setenv("IMAGE_ASYNC_ADOBE_MAX_CONCURRENT", "14")
	require.Equal(t, 46, effectiveImageWorkerConcurrency())
}

func TestImageWorkerConfigUsesUnifiedQueueDefaults(t *testing.T) {
	t.Setenv("IMAGE_ASYNC_MAX_CONCURRENT", "16")
	t.Setenv("IMAGE_ASYNC_ADOBE_MAX_CONCURRENT", "8")
	cfg := loadImageWorkerConfig()
	require.Equal(t, 24, cfg.concurrency)
	require.Equal(t, 96, cfg.queueCapacity)
	require.Equal(t, 48, cfg.dispatchBatch)
}
