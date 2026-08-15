package service

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
)

func TestResolveInternalModelNameUsesRoutingAliasWithoutReplacingDisplayPublic(t *testing.T) {
	modelPublicRegistryMu.Lock()
	modelPublicRegistryData = modelPublicRegistry{
		internalSet: map[string]struct{}{
			"cy-sd7-seedance-2.0-720p": {},
			"cy-sd4-seedance-2.0":      {},
		},
		internalToPublic: map[string]string{
			"cy-sd7-seedance-2.0-720p": "sd7-seedance-2.0-720p",
			"cy-sd4-seedance-2.0":      "sd4-seedance-2.0",
		},
		publicToInternals: map[string][]string{
			"sd7-seedance-2.0-720p": {"cy-sd7-seedance-2.0-720p"},
			"sd4-seedance-2.0":      {"cy-sd4-seedance-2.0"},
		},
		routingPublicToInternal: map[string]string{
			"seedance-2.0": "cy-sd7-seedance-2.0-720p",
		},
	}
	modelPublicRegistryReady = true
	modelPublicRegistryMu.Unlock()

	internal, clientPublic, err := ResolveInternalModelName("seedance-2.0")
	if err != nil {
		t.Fatalf("resolve seedance-2.0: %v", err)
	}
	if internal != "cy-sd7-seedance-2.0-720p" || clientPublic != "seedance-2.0" {
		t.Fatalf("routing resolve = (%q, %q)", internal, clientPublic)
	}
	if got := ToPublicModelName("cy-sd7-seedance-2.0-720p"); got != "sd7-seedance-2.0-720p" {
		t.Fatalf("display public unchanged: %q", got)
	}
}

func TestAppendRoutingAliasPricingAddsLegacyPublicName(t *testing.T) {
	modelPublicRegistryMu.Lock()
	modelPublicRegistryData = modelPublicRegistry{
		internalToPublic: map[string]string{
			"cy-sd7-seedance-2.0-720p": "sd7-seedance-2.0-720p",
		},
		routingPublicToInternal: map[string]string{
			"seedance-2.0": "cy-sd7-seedance-2.0-720p",
		},
	}
	modelPublicRegistryReady = true
	modelPublicRegistryMu.Unlock()

	source := []model.Pricing{{
		ModelName:   "cy-sd7-seedance-2.0-720p",
		ModelPrice:  3.9,
		Description: "Seedance 720p",
	}}
	out := AppendRoutingAliasPricing(source)
	if len(out) != 2 {
		t.Fatalf("expected 2 pricing rows, got %d", len(out))
	}
	if out[1].ModelName != "seedance-2.0" || out[1].ModelPrice != 3.9 {
		t.Fatalf("routing pricing row = %#v", out[1])
	}
}
