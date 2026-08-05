package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPricingSnapshotReady(t *testing.T) {
	previous := currentPricingSnapshot.Load()
	t.Cleanup(func() {
		currentPricingSnapshot.Store(previous)
		pricingCacheInvalid.Store(false)
		lastPricingRefreshTry.Store(0)
	})

	currentPricingSnapshot.Store(nil)
	require.False(t, PricingSnapshotReady(time.Minute))

	currentPricingSnapshot.Store(&pricingSnapshot{
		pricing:     []Pricing{{ModelName: "ready-model"}},
		refreshedAt: time.Now().Add(-2 * time.Minute),
	})
	require.False(t, PricingSnapshotReady(time.Minute))
	require.True(t, PricingSnapshotReady(5*time.Minute))
}

func TestInvalidationPreservesPublishedSnapshot(t *testing.T) {
	previous := currentPricingSnapshot.Load()
	t.Cleanup(func() {
		currentPricingSnapshot.Store(previous)
		pricingCacheInvalid.Store(false)
		lastPricingRefreshTry.Store(0)
	})

	snapshot := &pricingSnapshot{
		pricing:     []Pricing{{ModelName: "stable-model"}},
		refreshedAt: time.Now(),
	}
	currentPricingSnapshot.Store(snapshot)
	pricingRefreshLock.Lock()
	t.Cleanup(pricingRefreshLock.Unlock)

	InvalidatePricingCache()

	require.Same(t, snapshot, currentPricingSnapshot.Load())
	require.True(t, pricingCacheInvalid.Load())
	require.Equal(t, "stable-model", GetPricing()[0].ModelName)
}
