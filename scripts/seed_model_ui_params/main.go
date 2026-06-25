package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
)

type seedRegistry struct {
	DefaultId          string                   `json:"defaultId"`
	CapabilityFallback []string                 `json:"capabilityFallback"`
	Poll               map[string]interface{}   `json:"poll"`
	Profiles           []map[string]interface{} `json:"profiles"`
}

func main() {
	force := flag.Bool("force", false, "replace existing profiles for each capability")
	flag.Parse()

	common.IsMasterNode = true
	if err := model.InitDB(); err != nil {
		fmt.Fprintf(os.Stderr, "init db: %v\n", err)
		os.Exit(1)
	}

	scriptDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "cwd: %v\n", err)
		os.Exit(1)
	}
	seedDir := filepath.Join(scriptDir, "scripts", "seed_data")
	if _, err := os.Stat(seedDir); err != nil {
		seedDir = filepath.Join(scriptDir, "seed_data")
	}

	capabilities := []struct {
		capability string
		filename   string
	}{
		{model.ModelUiParamCapabilityVideo, "model_ui_params_video.json"},
		{model.ModelUiParamCapabilityImage, "model_ui_params_image.json"},
	}

	for _, item := range capabilities {
		path := filepath.Join(seedDir, item.filename)
		if err := seedCapability(item.capability, path, *force); err != nil {
			fmt.Fprintf(os.Stderr, "seed %s: %v\n", item.capability, err)
			os.Exit(1)
		}
		fmt.Printf("seeded model ui params: %s\n", item.capability)
	}
}

func seedCapability(capability, path string, force bool) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var doc seedRegistry
	if err := json.Unmarshal(raw, &doc); err != nil {
		return err
	}

	existingProfiles, err := model.GetAllModelUiParamProfiles(capability)
	if err != nil {
		return err
	}
	if len(existingProfiles) > 0 && !force {
		fmt.Printf("skip %s: %d profiles already exist (use -force)\n", capability, len(existingProfiles))
		return nil
	}

	registry, err := model.GetModelUiParamRegistryByCapability(capability)
	if err != nil {
		registry = &model.ModelUiParamRegistry{
			Capability:         capability,
			DefaultProfileId:   doc.DefaultId,
			CapabilityFallback: service.MustJSONString(doc.CapabilityFallback, "[]"),
			PollDefaults:       service.MustJSONString(doc.Poll, "{}"),
		}
		if err := registry.Insert(); err != nil {
			return err
		}
	} else {
		registry.DefaultProfileId = doc.DefaultId
		registry.CapabilityFallback = service.MustJSONString(doc.CapabilityFallback, "[]")
		if capability == model.ModelUiParamCapabilityVideo {
			registry.PollDefaults = service.MustJSONString(doc.Poll, "{}")
		}
		if err := registry.Update(); err != nil {
			return err
		}
	}

	if force && len(existingProfiles) > 0 {
		for _, profile := range existingProfiles {
			if err := model.DeleteModelUiParamProfile(profile.Id); err != nil {
				return err
			}
		}
	}

	for index, profileDoc := range doc.Profiles {
		row, err := profileDocToRow(capability, profileDoc, index*10)
		if err != nil {
			return err
		}
		if err := row.Insert(); err != nil {
			return err
		}
	}
	return nil
}

func profileDocToRow(capability string, doc map[string]interface{}, sortOrder int) (*model.ModelUiParamProfile, error) {
	profileId, _ := doc["id"].(string)
	if strings.TrimSpace(profileId) == "" {
		return nil, fmt.Errorf("profile missing id at sort_order %d", sortOrder)
	}

	match := service.MustJSONString(doc["match"], "[]")
	params := service.MustJSONString(doc["params"], "{}")

	row := &model.ModelUiParamProfile{
		Capability: capability,
		ProfileId:  profileId,
		Match:      match,
		SortOrder:  sortOrder,
		Params:     params,
		OptionRules: "[]",
		Hints:       "[]",
		Poll:          "{}",
		ReferenceLimits: "{}",
	}

	if capability == model.ModelUiParamCapabilityVideo {
		if apiMode, ok := doc["apiMode"].(string); ok {
			row.ApiMode = apiMode
		}
		if requires, ok := doc["requiresReferenceMedia"].(bool); ok {
			row.RequiresReferenceMedia = requires
		}
		if pollStatus, ok := doc["pollStatus"].(string); ok {
			row.PollStatus = pollStatus
		}
		if poll, ok := doc["poll"]; ok {
			row.Poll = service.MustJSONString(poll, "{}")
		}
		if limits, ok := doc["referenceLimits"]; ok {
			row.ReferenceLimits = service.MustJSONString(limits, "{}")
		}
		if rules, ok := doc["optionRules"]; ok {
			row.OptionRules = service.MustJSONString(rules, "[]")
		}
		if hints, ok := doc["hints"]; ok {
			row.Hints = service.MustJSONString(hints, "[]")
		}
	}

	return row, nil
}
