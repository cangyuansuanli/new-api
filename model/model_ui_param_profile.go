package model

import (
	"errors"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

type ModelUiParamProfile struct {
	Id                     int            `json:"id" gorm:"primaryKey;autoIncrement"`
	Capability             string         `json:"capability" gorm:"size:16;not null;uniqueIndex:uk_model_ui_param_profile_cap_id,priority:1"`
	ProfileId              string         `json:"profile_id" gorm:"size:128;not null;uniqueIndex:uk_model_ui_param_profile_cap_id,priority:2"`
	Match                  string         `json:"match" gorm:"type:text;not null;default:'[]'"`
	SortOrder              int            `json:"sort_order" gorm:"not null;default:0"`
	ApiMode                string         `json:"api_mode" gorm:"size:32"`
	RequiresReferenceMedia bool           `json:"requires_reference_media" gorm:"default:false;not null"`
	Poll                   string         `json:"poll" gorm:"type:text;not null;default:'{}'"`
	PollStatus             string         `json:"poll_status" gorm:"size:16"`
	ReferenceLimits        string         `json:"reference_limits" gorm:"type:text;not null;default:'{}'"`
	Params                 string         `json:"params" gorm:"type:text;not null;default:'{}'"`
	OptionRules            string         `json:"option_rules" gorm:"type:text;not null;default:'[]'"`
	Hints                  string         `json:"hints" gorm:"type:text;not null;default:'[]'"`
	Note                   string         `json:"note" gorm:"size:512"`
	CreatedTime            int64          `json:"created_time" gorm:"bigint"`
	UpdatedTime            int64          `json:"updated_time" gorm:"bigint"`
	DeletedAt              gorm.DeletedAt `json:"-" gorm:"index"`
}

func (item *ModelUiParamProfile) Insert() error {
	if item.Capability != ModelUiParamCapabilityVideo && item.Capability != ModelUiParamCapabilityImage {
		return errors.New("invalid capability")
	}
	if item.ProfileId == "" {
		return errors.New("profile_id is required")
	}
	now := common.GetTimestamp()
	item.CreatedTime = now
	item.UpdatedTime = now
	return DB.Create(item).Error
}

func (item *ModelUiParamProfile) Update() error {
	if item.Capability != ModelUiParamCapabilityVideo && item.Capability != ModelUiParamCapabilityImage {
		return errors.New("invalid capability")
	}
	if item.ProfileId == "" {
		return errors.New("profile_id is required")
	}
	item.UpdatedTime = common.GetTimestamp()
	return DB.Model(item).Where("id = ?", item.Id).Updates(map[string]interface{}{
		"profile_id":               item.ProfileId,
		"match":                    item.Match,
		"sort_order":               item.SortOrder,
		"api_mode":                 item.ApiMode,
		"requires_reference_media": item.RequiresReferenceMedia,
		"poll":                     item.Poll,
		"poll_status":              item.PollStatus,
		"reference_limits":         item.ReferenceLimits,
		"params":                   item.Params,
		"option_rules":             item.OptionRules,
		"hints":                    item.Hints,
		"note":                     item.Note,
		"updated_time":             item.UpdatedTime,
	}).Error
}

func GetAllModelUiParamProfiles(capability string) ([]ModelUiParamProfile, error) {
	var items []ModelUiParamProfile
	tx := DB.Order("sort_order asc, id asc")
	if capability != "" {
		tx = tx.Where("capability = ?", capability)
	}
	err := tx.Find(&items).Error
	return items, err
}

func GetModelUiParamProfileByID(id int) (*ModelUiParamProfile, error) {
	var item ModelUiParamProfile
	err := DB.Where("id = ?", id).First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func IsModelUiParamProfileDuplicated(id int, capability, profileId string) (bool, error) {
	if capability == "" || profileId == "" {
		return false, nil
	}
	var count int64
	err := DB.Model(&ModelUiParamProfile{}).
		Where("capability = ? AND profile_id = ? AND id <> ?", capability, profileId, id).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func DeleteModelUiParamProfile(id int) error {
	result := DB.Delete(&ModelUiParamProfile{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("record not found")
	}
	return nil
}

type ModelUiParamProfileReorderItem struct {
	Id        int `json:"id"`
	SortOrder int `json:"sort_order"`
}

func ReorderModelUiParamProfiles(capability string, items []ModelUiParamProfileReorderItem) error {
	if capability == "" {
		return errors.New("capability is required")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		now := common.GetTimestamp()
		for _, item := range items {
			if item.Id <= 0 {
				continue
			}
			if err := tx.Model(&ModelUiParamProfile{}).
				Where("id = ? AND capability = ?", item.Id, capability).
				Updates(map[string]interface{}{
					"sort_order":   item.SortOrder,
					"updated_time": now,
				}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
