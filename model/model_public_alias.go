package model

import (
	"errors"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

type ModelPublicAlias struct {
	Id           int            `json:"id" gorm:"primaryKey;autoIncrement"`
	InternalName string         `json:"internal_name" gorm:"size:255;not null;uniqueIndex:uk_model_public_alias_internal"`
	PublicName   string         `json:"public_name" gorm:"size:255;not null;uniqueIndex:uk_model_public_alias_public"`
	CreatedTime  int64          `json:"created_time" gorm:"bigint"`
	UpdatedTime  int64          `json:"updated_time" gorm:"bigint"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}

func (alias *ModelPublicAlias) Insert() error {
	now := common.GetTimestamp()
	alias.CreatedTime = now
	alias.UpdatedTime = now
	return DB.Create(alias).Error
}

func (alias *ModelPublicAlias) Update() error {
	alias.UpdatedTime = common.GetTimestamp()
	return DB.Model(alias).Where("id = ?", alias.Id).Updates(map[string]interface{}{
		"internal_name": alias.InternalName,
		"public_name":   alias.PublicName,
		"updated_time":  alias.UpdatedTime,
	}).Error
}

func GetAllModelPublicAliases() ([]ModelPublicAlias, error) {
	var aliases []ModelPublicAlias
	err := DB.Order("internal_name asc").Find(&aliases).Error
	return aliases, err
}

func GetModelPublicAliasByID(id int) (*ModelPublicAlias, error) {
	var alias ModelPublicAlias
	err := DB.Where("id = ?", id).First(&alias).Error
	if err != nil {
		return nil, err
	}
	return &alias, nil
}

func IsModelPublicAliasDuplicated(id int, internalName, publicName string) (bool, error) {
	if internalName == "" && publicName == "" {
		return false, nil
	}
	var count int64
	tx := DB.Model(&ModelPublicAlias{})
	if internalName != "" {
		tx = tx.Where("internal_name = ? AND id <> ?", internalName, id)
	} else {
		tx = tx.Where("public_name = ? AND id <> ?", publicName, id)
	}
	if err := tx.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func DeleteModelPublicAlias(id int) error {
	result := DB.Delete(&ModelPublicAlias{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("record not found")
	}
	return nil
}
