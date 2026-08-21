package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveInternalModelNamePrefersSingleMarketplaceActiveAlias(t *testing.T) {
	modelPublicRegistryMu.Lock()
	modelPublicRegistryData = modelPublicRegistry{
		internalSet: map[string]struct{}{
			"adobe-firefly-gpt-image-2-4k": {},
			"cy-yf-gpt-image-2-4k":         {},
		},
		internalToPublic: map[string]string{
			"adobe-firefly-gpt-image-2-4k": "gpt-image-2-4k",
			"cy-yf-gpt-image-2-4k":         "gpt-image-2-4k",
		},
		publicToInternals: map[string][]string{
			"gpt-image-2-4k": {
				"adobe-firefly-gpt-image-2-4k",
				"cy-yf-gpt-image-2-4k",
			},
		},
		marketplaceActiveInternal: map[string]bool{
			"cy-yf-gpt-image-2-4k": true,
		},
	}
	modelPublicRegistryReady = true
	modelPublicRegistryMu.Unlock()

	internal, clientPublic, err := ResolveInternalModelName("gpt-image-2-4k")
	require.NoError(t, err)
	require.Equal(t, "cy-yf-gpt-image-2-4k", internal)
	require.Equal(t, "gpt-image-2-4k", clientPublic)
}

func TestResolveInternalModelNamePublicAliasWinsOverRoutingAlias(t *testing.T) {
	modelPublicRegistryMu.Lock()
	modelPublicRegistryData = modelPublicRegistry{
		internalSet: map[string]struct{}{
			"adobe-firefly-gpt-image-2-4k": {},
			"cy-yf-gpt-image-2-4k":         {},
		},
		internalToPublic: map[string]string{
			"adobe-firefly-gpt-image-2-4k": "gpt-image-2-4k",
			"cy-yf-gpt-image-2-4k":         "gpt-image-2-4k",
		},
		publicToInternals: map[string][]string{
			"gpt-image-2-4k": {
				"adobe-firefly-gpt-image-2-4k",
				"cy-yf-gpt-image-2-4k",
			},
		},
		routingPublicToInternal: map[string]string{
			"gpt-image-2-4k": "cy-yf-gpt-image-2-4k",
		},
		marketplaceActiveInternal: map[string]bool{
			"adobe-firefly-gpt-image-2-4k": true,
		},
	}
	modelPublicRegistryReady = true
	modelPublicRegistryMu.Unlock()

	internal, clientPublic, err := ResolveInternalModelName("gpt-image-2-4k")
	require.NoError(t, err)
	require.Equal(t, "adobe-firefly-gpt-image-2-4k", internal)
	require.Equal(t, "gpt-image-2-4k", clientPublic)
}
