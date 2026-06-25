package model

import (
	"errors"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

const (
	ModelUiParamCapabilityVideo = "video"
	ModelUiParamCapabilityImage = "image"
)

type ModelUiParamRegistry struct {
	Id                 int            `json:"id" gorm:"primaryKey;autoIncrement"`
	Capability         string         `json:"capability" gorm:"size:16;not null;uniqueIndex:uk_model_ui_param_registry_capability"`
	DefaultProfileId   string         `json:"default_profile_id" gorm:"size:128;not null"`
	CapabilityFallback string         `json:"capability_fallback" gorm:"type:text;not null;default:'[]'"`
	PollDefaults       string         `json:"poll_defaults" gorm:"type:text;not null;default:'{}'"`
	UpdatedTime        int64          `json:"updated_time" gorm:"bigint"`
	DeletedAt          gorm.DeletedAt `json:"-" gorm:"index"`
}

func (item *ModelUiParamRegistry) Insert() error {
	if item.Capability != ModelUiParamCapabilityVideo && item.Capability != ModelUiParamCapabilityImage {
		return errors.New("invalid capability")
	}
	now := common.GetTimestamp()
	item.UpdatedTime = now
	return DB.Create(item).Error
}

func (item *ModelUiParamRegistry) Update() error {
	if item.Capability != ModelUiParamCapabilityVideo && item.Capability != ModelUiParamCapabilityImage {
		return errors.New("invalid capability")
	}
	item.UpdatedTime = common.GetTimestamp()
	return DB.Model(item).Where("id = ?", item.Id).Updates(map[string]interface{}{
		"default_profile_id":  item.DefaultProfileId,
		"capability_fallback": item.CapabilityFallback,
		"poll_defaults":       item.PollDefaults,
		"updated_time":        item.UpdatedTime,
	}).Error
}

func GetModelUiParamRegistryByCapability(capability string) (*ModelUiParamRegistry, error) {
	var item ModelUiParamRegistry
	err := DB.Where("capability = ?", capability).First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func GetAllModelUiParamRegistries() ([]ModelUiParamRegistry, error) {
	var items []ModelUiParamRegistry
	err := DB.Order("capability asc").Find(&items).Error
	return items, err
}
