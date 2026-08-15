package model

import (
	"errors"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

// ModelRoutingAlias maps a legacy neutral public model name to an internal
// ability name for inbound routing only. It does not replace the display
// public name registered in model_public_aliases.
type ModelRoutingAlias struct {
	Id           int            `json:"id" gorm:"primaryKey;autoIncrement"`
	PublicName   string         `json:"public_name" gorm:"size:255;not null;uniqueIndex:uk_model_routing_alias_public"`
	InternalName string         `json:"internal_name" gorm:"size:255;not null;index:idx_model_routing_alias_internal"`
	Note         string         `json:"note" gorm:"size:255"`
	CreatedTime  int64          `json:"created_time" gorm:"bigint"`
	UpdatedTime  int64          `json:"updated_time" gorm:"bigint"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}

func (alias *ModelRoutingAlias) Insert() error {
	now := common.GetTimestamp()
	alias.CreatedTime = now
	alias.UpdatedTime = now
	return DB.Create(alias).Error
}

func (alias *ModelRoutingAlias) Update() error {
	alias.UpdatedTime = common.GetTimestamp()
	return DB.Model(alias).Where("id = ?", alias.Id).Updates(map[string]interface{}{
		"public_name":   alias.PublicName,
		"internal_name": alias.InternalName,
		"note":          alias.Note,
		"updated_time":  alias.UpdatedTime,
	}).Error
}

func GetAllModelRoutingAliases() ([]ModelRoutingAlias, error) {
	var aliases []ModelRoutingAlias
	err := DB.Order("public_name asc").Find(&aliases).Error
	return aliases, err
}

func GetModelRoutingAliasByID(id int) (*ModelRoutingAlias, error) {
	var alias ModelRoutingAlias
	err := DB.Where("id = ?", id).First(&alias).Error
	if err != nil {
		return nil, err
	}
	return &alias, nil
}

func IsModelRoutingAliasDuplicated(id int, publicName string) (bool, error) {
	if publicName == "" {
		return false, nil
	}
	var count int64
	tx := DB.Model(&ModelRoutingAlias{}).Where("public_name = ? AND id <> ?", publicName, id)
	if err := tx.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func DeleteModelRoutingAlias(id int) error {
	result := DB.Delete(&ModelRoutingAlias{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("record not found")
	}
	return nil
}
