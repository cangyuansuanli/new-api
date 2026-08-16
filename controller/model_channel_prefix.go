package controller

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

func GetAllModelChannelPrefixes(c *gin.Context) {
	items, err := model.GetAllModelChannelPrefixes()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, items)
}

func CreateModelChannelPrefix(c *gin.Context) {
	var item model.ModelChannelPrefix
	if err := c.ShouldBindJSON(&item); err != nil {
		common.ApiError(c, err)
		return
	}
	item.Note = strings.TrimSpace(item.Note)
	item.Prefix = model.NormalizeModelChannelPrefix(item.Prefix)
	if item.Prefix == "" {
		common.ApiErrorMsg(c, "prefix is required")
		return
	}
	dup, err := model.IsModelChannelPrefixDuplicated(0, item.Prefix)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if dup {
		common.ApiErrorMsg(c, "prefix already exists")
		return
	}
	if err := item.Insert(); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := refreshModelPublicRegistry(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, item)
}

func UpdateModelChannelPrefix(c *gin.Context) {
	var item model.ModelChannelPrefix
	if err := c.ShouldBindJSON(&item); err != nil {
		common.ApiError(c, err)
		return
	}
	if item.Id <= 0 {
		common.ApiErrorMsg(c, "id is required")
		return
	}
	item.Note = strings.TrimSpace(item.Note)
	item.Prefix = model.NormalizeModelChannelPrefix(item.Prefix)
	if item.Prefix == "" {
		common.ApiErrorMsg(c, "prefix is required")
		return
	}
	dup, err := model.IsModelChannelPrefixDuplicated(item.Id, item.Prefix)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if dup {
		common.ApiErrorMsg(c, "prefix already exists")
		return
	}
	if err := item.Update(); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := refreshModelPublicRegistry(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, item)
}

func DeleteModelChannelPrefix(c *gin.Context) {
	id := common.String2Int(c.Param("id"))
	if id == 0 {
		common.ApiErrorMsg(c, "invalid id")
		return
	}
	if err := model.DeleteModelChannelPrefix(id); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := refreshModelPublicRegistry(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func GetModelPublicNameRegistryStatus(c *gin.Context) {
	collisions, missingAliases, ready := service.GetModelPublicNameRegistryStatus()
	common.ApiSuccess(c, gin.H{
		"ready":           ready,
		"collisions":      collisions,
		"missing_aliases": missingAliases,
	})
}
