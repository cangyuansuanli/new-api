package controller

import (
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

func GetAllModelPublicAliases(c *gin.Context) {
	aliases, err := model.GetAllModelPublicAliases()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, aliases)
}

func CreateModelPublicAlias(c *gin.Context) {
	var alias model.ModelPublicAlias
	if err := c.ShouldBindJSON(&alias); err != nil {
		common.ApiError(c, err)
		return
	}
	alias.InternalName = strings.TrimSpace(alias.InternalName)
	alias.PublicName = strings.TrimSpace(alias.PublicName)
	if alias.InternalName == "" || alias.PublicName == "" {
		common.ApiErrorMsg(c, "internal_name and public_name are required")
		return
	}
	dup, err := model.IsModelPublicAliasDuplicated(0, alias.InternalName, "")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if dup {
		common.ApiErrorMsg(c, "internal_name already exists")
		return
	}
	dup, err = model.IsModelPublicAliasDuplicated(0, "", alias.PublicName)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if dup {
		common.ApiErrorMsg(c, "public_name already exists")
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

func UpdateModelPublicAlias(c *gin.Context) {
	var alias model.ModelPublicAlias
	if err := c.ShouldBindJSON(&alias); err != nil {
		common.ApiError(c, err)
		return
	}
	if alias.Id <= 0 {
		common.ApiErrorMsg(c, "id is required")
		return
	}
	alias.InternalName = strings.TrimSpace(alias.InternalName)
	alias.PublicName = strings.TrimSpace(alias.PublicName)
	if alias.InternalName == "" || alias.PublicName == "" {
		common.ApiErrorMsg(c, "internal_name and public_name are required")
		return
	}
	dup, err := model.IsModelPublicAliasDuplicated(alias.Id, alias.InternalName, "")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if dup {
		common.ApiErrorMsg(c, "internal_name already exists")
		return
	}
	dup, err = model.IsModelPublicAliasDuplicated(alias.Id, "", alias.PublicName)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if dup {
		common.ApiErrorMsg(c, "public_name already exists")
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

func DeleteModelPublicAlias(c *gin.Context) {
	id := common.String2Int(c.Param("id"))
	if id == 0 {
		common.ApiErrorMsg(c, "invalid id")
		return
	}
	if err := model.DeleteModelPublicAlias(id); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := refreshModelPublicRegistry(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func refreshModelPublicRegistry() error {
	return service.RefreshModelPublicNameRegistry()
}
