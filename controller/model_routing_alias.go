package controller

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

func GetAllModelRoutingAliases(c *gin.Context) {
	aliases, err := model.GetAllModelRoutingAliases()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, aliases)
}

func validateModelRoutingAlias(c *gin.Context, alias *model.ModelRoutingAlias) bool {
	alias.PublicName = strings.TrimSpace(alias.PublicName)
	alias.InternalName = strings.TrimSpace(alias.InternalName)
	alias.Note = strings.TrimSpace(alias.Note)
	if alias.PublicName == "" || alias.InternalName == "" {
		common.ApiErrorMsg(c, "public_name and internal_name are required")
		return false
	}

	duplicated, err := model.IsModelRoutingAliasDuplicated(alias.Id, alias.PublicName)
	if err != nil {
		common.ApiError(c, err)
		return false
	}
	if duplicated {
		common.ApiErrorMsg(c, "public_name already exists")
		return false
	}

	duplicated, err = model.IsModelPublicAliasDuplicated(0, "", alias.PublicName)
	if err != nil {
		common.ApiError(c, err)
		return false
	}
	if duplicated {
		common.ApiErrorMsg(c, "public_name conflicts with a model public alias")
		return false
	}

	targetExists, err := model.ModelRoutingAliasTargetExists(alias.InternalName)
	if err != nil {
		common.ApiError(c, err)
		return false
	}
	if !targetExists {
		common.ApiErrorMsg(c, "internal_name does not exist in abilities")
		return false
	}
	return true
}

func CreateModelRoutingAlias(c *gin.Context) {
	var alias model.ModelRoutingAlias
	if err := c.ShouldBindJSON(&alias); err != nil {
		common.ApiError(c, err)
		return
	}
	alias.Id = 0
	if !validateModelRoutingAlias(c, &alias) {
		return
	}
	if err := alias.Insert(); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := refreshModelPublicRegistry(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, alias)
}

func UpdateModelRoutingAlias(c *gin.Context) {
	var alias model.ModelRoutingAlias
	if err := c.ShouldBindJSON(&alias); err != nil {
		common.ApiError(c, err)
		return
	}
	if alias.Id <= 0 {
		common.ApiErrorMsg(c, "id is required")
		return
	}
	if _, err := model.GetModelRoutingAliasByID(alias.Id); err != nil {
		common.ApiError(c, err)
		return
	}
	if !validateModelRoutingAlias(c, &alias) {
		return
	}
	if err := alias.Update(); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := refreshModelPublicRegistry(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, alias)
}

func DeleteModelRoutingAlias(c *gin.Context) {
	id := common.String2Int(c.Param("id"))
	if id == 0 {
		common.ApiErrorMsg(c, "invalid id")
		return
	}
	if err := model.DeleteModelRoutingAlias(id); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := refreshModelPublicRegistry(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}
