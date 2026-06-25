package controller

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

func GetModelUiParamsRegistry(c *gin.Context) {
	capability := strings.TrimSpace(c.Param("capability"))
	if capability != model.ModelUiParamCapabilityVideo && capability != model.ModelUiParamCapabilityImage {
		common.ApiErrorMsg(c, "invalid capability")
		return
	}
	data, err := service.BuildModelUiParamRegistryJSON(capability)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, data)
}

func GetModelUiParamRegistrySettings(c *gin.Context) {
	capability := strings.TrimSpace(c.Param("capability"))
	if capability != model.ModelUiParamCapabilityVideo && capability != model.ModelUiParamCapabilityImage {
		common.ApiErrorMsg(c, "invalid capability")
		return
	}
	item, err := model.GetModelUiParamRegistryByCapability(capability)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, item)
}

func UpdateModelUiParamRegistrySettings(c *gin.Context) {
	capability := strings.TrimSpace(c.Param("capability"))
	if capability != model.ModelUiParamCapabilityVideo && capability != model.ModelUiParamCapabilityImage {
		common.ApiErrorMsg(c, "invalid capability")
		return
	}
	item, err := model.GetModelUiParamRegistryByCapability(capability)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var payload model.ModelUiParamRegistry
	if err := c.ShouldBindJSON(&payload); err != nil {
		common.ApiError(c, err)
		return
	}
	if strings.TrimSpace(payload.DefaultProfileId) != "" {
		item.DefaultProfileId = strings.TrimSpace(payload.DefaultProfileId)
	}
	if strings.TrimSpace(payload.CapabilityFallback) != "" {
		item.CapabilityFallback = payload.CapabilityFallback
	}
	if capability == model.ModelUiParamCapabilityVideo && strings.TrimSpace(payload.PollDefaults) != "" {
		item.PollDefaults = payload.PollDefaults
	}
	if err := item.Update(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, item)
}

func GetAllModelUiParamProfiles(c *gin.Context) {
	capability := strings.TrimSpace(c.Query("capability"))
	items, err := model.GetAllModelUiParamProfiles(capability)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, items)
}

func CreateModelUiParamProfile(c *gin.Context) {
	var item model.ModelUiParamProfile
	if err := c.ShouldBindJSON(&item); err != nil {
		common.ApiError(c, err)
		return
	}
	item.ProfileId = strings.TrimSpace(item.ProfileId)
	item.Note = strings.TrimSpace(item.Note)
	if item.ProfileId == "" {
		common.ApiErrorMsg(c, "profile_id is required")
		return
	}
	if item.Capability != model.ModelUiParamCapabilityVideo && item.Capability != model.ModelUiParamCapabilityImage {
		common.ApiErrorMsg(c, "invalid capability")
		return
	}
	matchTokens, err := service.ParseJSONStringArray(item.Match)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := service.ValidateModelUiParamMatchTokens(matchTokens); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	dup, err := model.IsModelUiParamProfileDuplicated(0, item.Capability, item.ProfileId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if dup {
		common.ApiErrorMsg(c, "profile_id already exists for this capability")
		return
	}
	if strings.TrimSpace(item.Params) == "" {
		item.Params = "{}"
	}
	if strings.TrimSpace(item.Match) == "" {
		item.Match = "[]"
	}
	if err := item.Insert(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, item)
}

func UpdateModelUiParamProfile(c *gin.Context) {
	var item model.ModelUiParamProfile
	if err := c.ShouldBindJSON(&item); err != nil {
		common.ApiError(c, err)
		return
	}
	if item.Id <= 0 {
		common.ApiErrorMsg(c, "id is required")
		return
	}
	item.ProfileId = strings.TrimSpace(item.ProfileId)
	item.Note = strings.TrimSpace(item.Note)
	if item.ProfileId == "" {
		common.ApiErrorMsg(c, "profile_id is required")
		return
	}
	matchTokens, err := service.ParseJSONStringArray(item.Match)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := service.ValidateModelUiParamMatchTokens(matchTokens); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	dup, err := model.IsModelUiParamProfileDuplicated(item.Id, item.Capability, item.ProfileId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if dup {
		common.ApiErrorMsg(c, "profile_id already exists for this capability")
		return
	}
	if strings.TrimSpace(item.Params) == "" {
		item.Params = "{}"
	}
	if strings.TrimSpace(item.Match) == "" {
		item.Match = "[]"
	}
	if err := item.Update(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, item)
}

func DeleteModelUiParamProfile(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "invalid id")
		return
	}
	if err := model.DeleteModelUiParamProfile(id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func ReorderModelUiParamProfiles(c *gin.Context) {
	var payload struct {
		Capability string                              `json:"capability"`
		Items      []model.ModelUiParamProfileReorderItem `json:"items"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		common.ApiError(c, err)
		return
	}
	if payload.Capability != model.ModelUiParamCapabilityVideo && payload.Capability != model.ModelUiParamCapabilityImage {
		common.ApiErrorMsg(c, "invalid capability")
		return
	}
	if len(payload.Items) == 0 {
		common.ApiErrorMsg(c, "items is required")
		return
	}
	if err := model.ReorderModelUiParamProfiles(payload.Capability, payload.Items); err != nil {
		common.ApiError(c, err)
		return
	}
	c.Status(http.StatusOK)
	common.ApiSuccess(c, nil)
}

func PreviewModelUiParamMatch(c *gin.Context) {
	capability := strings.TrimSpace(c.Query("capability"))
	modelName := strings.TrimSpace(c.Query("model_name"))
	if capability != model.ModelUiParamCapabilityVideo && capability != model.ModelUiParamCapabilityImage {
		common.ApiErrorMsg(c, "invalid capability")
		return
	}
	if modelName == "" {
		common.ApiErrorMsg(c, "model_name is required")
		return
	}
	out, err := service.PreviewModelUiParamMatch(capability, modelName)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, out)
}
