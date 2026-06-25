package model

import (
	"encoding/json"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

type UiParamResolveContext struct {
	VideoProfiles map[string]ModelUiParamProfile
	ImageProfiles map[string]ModelUiParamProfile
	VideoRegistry *ModelUiParamRegistry
	ImageRegistry *ModelUiParamRegistry
}

func LoadUiParamResolveContext() (*UiParamResolveContext, error) {
	ctx := &UiParamResolveContext{
		VideoProfiles: make(map[string]ModelUiParamProfile),
		ImageProfiles: make(map[string]ModelUiParamProfile),
	}

	videoProfiles, err := GetAllModelUiParamProfiles(ModelUiParamCapabilityVideo)
	if err != nil {
		return nil, err
	}
	for _, profile := range videoProfiles {
		ctx.VideoProfiles[profile.ProfileId] = profile
	}

	imageProfiles, err := GetAllModelUiParamProfiles(ModelUiParamCapabilityImage)
	if err != nil {
		return nil, err
	}
	for _, profile := range imageProfiles {
		ctx.ImageProfiles[profile.ProfileId] = profile
	}

	videoRegistry, err := GetModelUiParamRegistryByCapability(ModelUiParamCapabilityVideo)
	if err == nil {
		ctx.VideoRegistry = videoRegistry
	}
	imageRegistry, err := GetModelUiParamRegistryByCapability(ModelUiParamCapabilityImage)
	if err == nil {
		ctx.ImageRegistry = imageRegistry
	}

	return ctx, nil
}

func ResolveVideoUiParams(meta *Model, ctx *UiParamResolveContext) map[string]interface{} {
	if ctx == nil {
		return nil
	}
	profileID := ""
	if meta != nil {
		profileID = strings.TrimSpace(meta.VideoProfileId)
	}
	doc, err := resolveProfileDocument(ModelUiParamCapabilityVideo, profileID, ctx.VideoProfiles, ctx.VideoRegistry)
	if err != nil || doc == nil {
		return nil
	}
	applyVideoPollDefaults(doc, ctx.VideoRegistry)
	return doc
}

func ResolveImageUiParams(meta *Model, ctx *UiParamResolveContext) map[string]interface{} {
	if ctx == nil {
		return nil
	}
	profileID := ""
	if meta != nil {
		profileID = strings.TrimSpace(meta.ImageProfileId)
	}
	doc, err := resolveProfileDocument(ModelUiParamCapabilityImage, profileID, ctx.ImageProfiles, ctx.ImageRegistry)
	if err != nil || doc == nil {
		return nil
	}
	return doc
}

func resolveProfileDocument(
	capability, profileID string,
	profiles map[string]ModelUiParamProfile,
	registry *ModelUiParamRegistry,
) (map[string]interface{}, error) {
	if profileID != "" {
		if profile, ok := profiles[profileID]; ok {
			return profileToDocument(profile)
		}
	}
	if registry != nil {
		if profile, ok := profiles[registry.DefaultProfileId]; ok {
			return profileToDocument(profile)
		}
	}
	for _, profile := range profiles {
		return profileToDocument(profile)
	}
	return nil, nil
}

func profileToDocument(profile ModelUiParamProfile) (map[string]interface{}, error) {
	doc := map[string]interface{}{
		"id": profile.ProfileId,
	}

	var params map[string]interface{}
	if err := json.Unmarshal([]byte(profile.Params), &params); err != nil {
		return nil, err
	}
	doc["params"] = params

	if profile.Capability == ModelUiParamCapabilityVideo {
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

func applyVideoPollDefaults(doc map[string]interface{}, registry *ModelUiParamRegistry) {
	if doc == nil || registry == nil {
		return
	}
	if _, ok := doc["poll"]; ok {
		return
	}
	apiMode, _ := doc["apiMode"].(string)
	if apiMode == "" {
		if id, _ := doc["id"].(string); strings.HasPrefix(id, "gz-video") {
			apiMode = "videos-json-gz"
		} else {
			apiMode = "videos-form"
		}
	}
	var pollDefaults map[string]map[string]interface{}
	if err := json.Unmarshal([]byte(registry.PollDefaults), &pollDefaults); err != nil {
		return
	}
	if poll, ok := pollDefaults[apiMode]; ok && len(poll) > 0 {
		doc["poll"] = poll
	}
}

func BindModelsToProfile(capability, profileID string, matchTokens []string) error {
	if capability != ModelUiParamCapabilityVideo && capability != ModelUiParamCapabilityImage {
		return nil
	}
	profileID = strings.TrimSpace(profileID)
	if profileID == "" || len(matchTokens) == 0 {
		return nil
	}

	var models []Model
	if err := DB.Find(&models).Error; err != nil {
		return err
	}

	now := common.GetTimestamp()
	for _, item := range models {
		name := strings.ToLower(strings.TrimSpace(item.ModelName))
		if name == "" {
			continue
		}
		matched := false
		for _, token := range matchTokens {
			token = strings.ToLower(strings.TrimSpace(token))
			if token != "" && strings.Contains(name, token) {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}
		updates := map[string]interface{}{"updated_time": now}
		if capability == ModelUiParamCapabilityVideo {
			updates["video_profile_id"] = profileID
		} else {
			updates["image_profile_id"] = profileID
		}
		if err := DB.Model(&Model{}).Where("id = ?", item.Id).Updates(updates).Error; err != nil {
			return err
		}
	}
	return nil
}
