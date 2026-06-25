package service

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/model"
)

var channelPrefixTokens = []string{
	"119337-", "aini-", "byte-", "ctlove-", "czeq-", "go2api-",
	"gz-", "happyhorse-", "niming-", "oairegbox-", "yunwu-", "zeabur-",
}

func ValidateModelUiParamMatchTokens(tokens []string) error {
	for _, token := range tokens {
		trimmed := strings.TrimSpace(token)
		if trimmed == "" {
			continue
		}
		lower := strings.ToLower(trimmed)
		for _, prefix := range channelPrefixTokens {
			if strings.HasPrefix(lower, prefix) {
				return errors.New("match tokens must use public names without channel prefix: " + trimmed)
			}
		}
	}
	return nil
}

func ParseJSONStringArray(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{}, nil
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func MustJSONString(value interface{}, fallback string) string {
	if value == nil {
		return fallback
	}
	b, err := json.Marshal(value)
	if err != nil {
		return fallback
	}
	return string(b)
}

func BuildModelUiParamRegistryJSON(capability string) (map[string]interface{}, error) {
	registry, err := model.GetModelUiParamRegistryByCapability(capability)
	if err != nil {
		return nil, err
	}
	profiles, err := model.GetAllModelUiParamProfiles(capability)
	if err != nil {
		return nil, err
	}

	var capabilityFallback []string
	if err := json.Unmarshal([]byte(registry.CapabilityFallback), &capabilityFallback); err != nil {
		capabilityFallback = []string{}
	}

	profileDocs := make([]map[string]interface{}, 0, len(profiles))
	for _, profile := range profiles {
		doc, err := profileRowToDocument(profile)
		if err != nil {
			return nil, err
		}
		profileDocs = append(profileDocs, doc)
	}

	out := map[string]interface{}{
		"defaultId":          registry.DefaultProfileId,
		"capabilityFallback": capabilityFallback,
		"profiles":           profileDocs,
	}

	if capability == model.ModelUiParamCapabilityVideo {
		var pollDefaults map[string]interface{}
		if err := json.Unmarshal([]byte(registry.PollDefaults), &pollDefaults); err != nil {
			pollDefaults = map[string]interface{}{}
		}
		out["poll"] = pollDefaults
	}

	return out, nil
}

func profileRowToDocument(profile model.ModelUiParamProfile) (map[string]interface{}, error) {
	doc := map[string]interface{}{
		"id": profile.ProfileId,
	}

	var match []string
	if err := json.Unmarshal([]byte(profile.Match), &match); err != nil {
		return nil, err
	}
	doc["match"] = match

	var params map[string]interface{}
	if err := json.Unmarshal([]byte(profile.Params), &params); err != nil {
		return nil, err
	}
	doc["params"] = params

	if profile.Capability == model.ModelUiParamCapabilityVideo {
		if profile.ApiMode != "" {
			doc["apiMode"] = profile.ApiMode
		}
		if profile.RequiresReferenceMedia {
			doc["requiresReferenceMedia"] = true
		}
		if profile.PollStatus != "" {
			doc["pollStatus"] = profile.PollStatus
		}
		if strings.TrimSpace(profile.Poll) != "" && profile.Poll != "{}" {
			var poll map[string]interface{}
			if err := json.Unmarshal([]byte(profile.Poll), &poll); err != nil {
				return nil, err
			}
			if len(poll) > 0 {
				doc["poll"] = poll
			}
		}
		if strings.TrimSpace(profile.ReferenceLimits) != "" && profile.ReferenceLimits != "{}" {
			var limits map[string]interface{}
			if err := json.Unmarshal([]byte(profile.ReferenceLimits), &limits); err != nil {
				return nil, err
			}
			if len(limits) > 0 {
				doc["referenceLimits"] = limits
			}
		}
		if strings.TrimSpace(profile.OptionRules) != "" && profile.OptionRules != "[]" {
			var rules []interface{}
			if err := json.Unmarshal([]byte(profile.OptionRules), &rules); err != nil {
				return nil, err
			}
			if len(rules) > 0 {
				doc["optionRules"] = rules
			}
		}
		if strings.TrimSpace(profile.Hints) != "" && profile.Hints != "[]" {
			var hints []interface{}
			if err := json.Unmarshal([]byte(profile.Hints), &hints); err != nil {
				return nil, err
			}
			if len(hints) > 0 {
				doc["hints"] = hints
			}
		}
	}

	return doc, nil
}

type ModelUiParamMatchPreview struct {
	ModelName       string   `json:"model_name"`
	MatchedProfile  string   `json:"matched_profile,omitempty"`
	MatchedProfiles []string `json:"matched_profiles,omitempty"`
	Collision       bool     `json:"collision"`
}

func PreviewModelUiParamMatch(capability, modelName string) (*ModelUiParamMatchPreview, error) {
	registry, err := model.GetModelUiParamRegistryByCapability(capability)
	if err != nil {
		return nil, err
	}
	profiles, err := model.GetAllModelUiParamProfiles(capability)
	if err != nil {
		return nil, err
	}

	modelName = strings.ToLower(strings.TrimSpace(modelName))
	var matched []string
	for _, profile := range profiles {
		if profile.ProfileId == registry.DefaultProfileId {
			continue
		}
		tokens, err := ParseJSONStringArray(profile.Match)
		if err != nil {
			return nil, err
		}
		for _, token := range tokens {
			if token == "" {
				continue
			}
			if strings.Contains(modelName, strings.ToLower(token)) {
				matched = append(matched, profile.ProfileId)
				break
			}
		}
	}

	out := &ModelUiParamMatchPreview{
		ModelName:       modelName,
		MatchedProfiles: matched,
		Collision:       len(matched) > 1,
	}
	if len(matched) > 0 {
		out.MatchedProfile = matched[0]
	}
	return out, nil
}
