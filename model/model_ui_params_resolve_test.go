package model

import "testing"

func TestProfileToDocumentImageHints(t *testing.T) {
	doc, err := profileToDocument(ModelUiParamProfile{
		ProfileId:   "image-tpl-test",
		Capability:  ModelUiParamCapabilityImage,
		Params:      `{"quality":{"enabled":false}}`,
		Hints:       `[{"text":"示例 hint"}]`,
		OptionRules: "[]",
	})
	if err != nil {
		t.Fatalf("profileToDocument() error = %v", err)
	}
	hints, ok := doc["hints"].([]interface{})
	if !ok || len(hints) != 1 {
		t.Fatalf("hints = %#v, want one hint", doc["hints"])
	}
}

func TestProfileToDocumentImageApiMode(t *testing.T) {
	doc, err := profileToDocument(ModelUiParamProfile{
		ProfileId:  "image-tpl-aspect-count-extended",
		Capability: ModelUiParamCapabilityImage,
		ApiMode:    "chat-completions",
		Params:     `{"quality":{"enabled":true}}`,
		Hints:      "[]",
	})
	if err != nil {
		t.Fatalf("profileToDocument() error = %v", err)
	}
	if doc["apiMode"] != "chat-completions" {
		t.Fatalf("apiMode = %#v, want chat-completions", doc["apiMode"])
	}
}

func TestResolveProfileDocumentRequiresExplicitProfileID(t *testing.T) {
	profiles := map[string]ModelUiParamProfile{
		"default-video": {
			ProfileId:  "default-video",
			Capability: ModelUiParamCapabilityVideo,
			Params:     `{"resolution":{"enabled":true}}`,
			Hints:      "[]",
		},
	}
	registry := &ModelUiParamRegistry{DefaultProfileId: "default-video"}

	doc, err := resolveProfileDocument(ModelUiParamCapabilityVideo, "", profiles, registry)
	if err != nil {
		t.Fatalf("resolveProfileDocument() error = %v", err)
	}
	if doc != nil {
		t.Fatalf("empty profileID doc = %#v, want nil", doc)
	}

	doc, err = resolveProfileDocument(ModelUiParamCapabilityVideo, "default-video", profiles, registry)
	if err != nil {
		t.Fatalf("resolveProfileDocument() error = %v", err)
	}
	if doc == nil || doc["id"] != "default-video" {
		t.Fatalf("explicit profile doc = %#v", doc)
	}
}

func TestProfileToDocumentVideoRoutingFields(t *testing.T) {
	doc, err := profileToDocument(ModelUiParamProfile{
		ProfileId:      "video-tpl-seedance-480p-async",
		Capability:     ModelUiParamCapabilityVideo,
		ApiMode:        "videos-json-async",
		PayloadBuilder: "seedance-flat",
		ValidationKey:  "seedance-oairegbox",
		Params:         `{"resolution":{"enabled":true}}`,
		Hints:          "[]",
	})
	if err != nil {
		t.Fatalf("profileToDocument() error = %v", err)
	}
	if doc["payloadBuilder"] != "seedance-flat" {
		t.Fatalf("payloadBuilder = %#v, want seedance-flat", doc["payloadBuilder"])
	}
	if doc["validationKey"] != "seedance-oairegbox" {
		t.Fatalf("validationKey = %#v, want seedance-oairegbox", doc["validationKey"])
	}
}

func TestApplyImagePollDefaults(t *testing.T) {
	registry := &ModelUiParamRegistry{
		PollDefaults: `{"images-json-async":{"delayMs":5000,"maxAttempts":72},"images-edits-async":{"delayMs":5000,"maxAttempts":72}}`,
	}
	doc := map[string]interface{}{
		"id":      "image-tpl-aspect-count-basic",
		"apiMode": "images-json-async",
	}
	applyImagePollDefaults(doc, registry)
	poll, ok := doc["poll"].(map[string]interface{})
	if !ok {
		t.Fatalf("poll = %#v, want map", doc["poll"])
	}
	if poll["delayMs"] != float64(5000) || poll["maxAttempts"] != float64(72) {
		t.Fatalf("poll = %#v", poll)
	}
}
