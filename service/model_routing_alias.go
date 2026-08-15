package service

import (
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
)

// AppendRoutingAliasPricing adds legacy neutral public names (e.g. seedance-2.0)
// to the pricing list when their routing target internal is already listed.
// The appended rows copy billing/UI metadata from the target internal model.
func AppendRoutingAliasPricing(pricing []model.Pricing) []model.Pricing {
	registry := getModelPublicRegistry()
	if len(registry.routingPublicToInternal) == 0 || len(pricing) == 0 {
		return pricing
	}

	byInternal := make(map[string]model.Pricing, len(pricing))
	seenPublic := make(map[string]struct{}, len(pricing))
	for _, item := range pricing {
		byInternal[item.ModelName] = item
		public := ToPublicModelName(item.ModelName)
		if public != "" {
			seenPublic[public] = struct{}{}
		}
	}

	out := pricing
	for public, internal := range registry.routingPublicToInternal {
		if _, exists := seenPublic[public]; exists {
			continue
		}
		source, ok := byInternal[internal]
		if !ok {
			continue
		}
		clone := source
		clone.ModelName = public
		out = append(out, clone)
		seenPublic[public] = struct{}{}
	}
	return out
}

// AppendRoutingAliasOpenAIModels adds legacy routing public names to /v1/models
// when the routing target internal is already visible to the user.
func AppendRoutingAliasOpenAIModels(models []dto.OpenAIModels, enabledInternals []string) []dto.OpenAIModels {
	registry := getModelPublicRegistry()
	if len(registry.routingPublicToInternal) == 0 {
		return models
	}

	enabled := make(map[string]struct{}, len(enabledInternals))
	for _, internal := range enabledInternals {
		enabled[internal] = struct{}{}
	}
	seenPublic := make(map[string]struct{}, len(models))
	byPublic := make(map[string]dto.OpenAIModels, len(models))
	for _, item := range models {
		seenPublic[item.Id] = struct{}{}
		byPublic[item.Id] = item
	}

	out := models
	for public, internal := range registry.routingPublicToInternal {
		if _, exists := seenPublic[public]; exists {
			continue
		}
		if _, ok := enabled[internal]; !ok {
			continue
		}
		template, ok := byPublic[ToPublicModelName(internal)]
		if !ok {
			continue
		}
		clone := template
		clone.Id = public
		out = append(out, clone)
		seenPublic[public] = struct{}{}
	}
	return out
}
