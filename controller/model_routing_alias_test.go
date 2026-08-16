package controller

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type modelRoutingAliasResponse struct {
	Success bool                    `json:"success"`
	Message string                  `json:"message"`
	Data    model.ModelRoutingAlias `json:"data"`
}

func performModelRoutingAliasRequest(t *testing.T, method, path, body string, handler gin.HandlerFunc) modelRoutingAliasResponse {
	t.Helper()
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(method, path, strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	if id := ctx.Param("id"); id == "" && strings.HasPrefix(path, "/api/model_routing_aliases/") {
		ctx.Params = gin.Params{{Key: "id", Value: strings.TrimPrefix(path, "/api/model_routing_aliases/")}}
	}
	handler(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response modelRoutingAliasResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	return response
}

func TestCreateModelRoutingAliasValidatesAndRefreshesRegistry(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.Create(&model.Ability{
		Group: "default", Model: "internal-video", ChannelId: 1, Enabled: true,
	}).Error)

	response := performModelRoutingAliasRequest(
		t,
		http.MethodPost,
		"/api/model_routing_aliases/",
		`{"public_name":" legacy-video ","internal_name":" internal-video ","note":" primary "}`,
		CreateModelRoutingAlias,
	)
	require.True(t, response.Success)
	require.Equal(t, "legacy-video", response.Data.PublicName)
	require.Equal(t, "internal-video", response.Data.InternalName)
	require.Equal(t, "primary", response.Data.Note)

	internal, clientPublic, err := service.ResolveInternalModelName("legacy-video")
	require.NoError(t, err)
	require.Equal(t, "internal-video", internal)
	require.Equal(t, "legacy-video", clientPublic)
}

func TestCreateModelRoutingAliasRejectsInvalidNames(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.Create(&model.Ability{
		Group: "default", Model: "internal-video", ChannelId: 1, Enabled: true,
	}).Error)
	require.NoError(t, db.Create(&model.ModelPublicAlias{
		InternalName: "internal-video", PublicName: "marketplace-video",
	}).Error)

	conflict := performModelRoutingAliasRequest(
		t,
		http.MethodPost,
		"/api/model_routing_aliases/",
		`{"public_name":"marketplace-video","internal_name":"internal-video"}`,
		CreateModelRoutingAlias,
	)
	require.False(t, conflict.Success)
	require.Equal(t, "public_name conflicts with a model public alias", conflict.Message)

	missingTarget := performModelRoutingAliasRequest(
		t,
		http.MethodPost,
		"/api/model_routing_aliases/",
		`{"public_name":"legacy-missing","internal_name":"missing-internal"}`,
		CreateModelRoutingAlias,
	)
	require.False(t, missingTarget.Success)
	require.Equal(t, "internal_name does not exist in abilities", missingTarget.Message)
}

func TestCreateModelPublicAliasRejectsRoutingName(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.Create(&model.ModelRoutingAlias{
		PublicName: "shared-name", InternalName: "internal-one",
	}).Error)

	response := performModelRoutingAliasRequest(
		t,
		http.MethodPost,
		"/api/model_public_aliases/",
		`{"internal_name":"internal-two","public_name":"shared-name"}`,
		CreateModelPublicAlias,
	)
	require.False(t, response.Success)
	require.Equal(t, "public_name conflicts with a model routing alias", response.Message)
}

func TestUpdateModelPublicAliasVisibility(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.Create(&model.Ability{
		Group: "default", Model: "cy-channel-image", ChannelId: 1, Enabled: true,
	}).Error)
	alias := model.ModelPublicAlias{
		InternalName: "cy-channel-image",
		PublicName:   "channel-image",
	}
	require.NoError(t, alias.Insert())

	response := performModelRoutingAliasRequest(
		t,
		http.MethodPut,
		"/api/model_public_aliases/",
		`{"id":`+strconv.Itoa(alias.Id)+`,"internal_name":"cy-channel-image","public_name":"channel-image","hidden_from_marketplace":true}`,
		UpdateModelPublicAlias,
	)
	require.True(t, response.Success)

	stored, err := model.GetModelPublicAliasByID(alias.Id)
	require.NoError(t, err)
	require.True(t, stored.HiddenFromMarketplace)
	require.False(t, service.IsModelPublicModelVisible(alias.InternalName))

	internal, clientName, err := service.ResolveInternalModelName(alias.PublicName)
	require.NoError(t, err)
	require.Equal(t, alias.InternalName, internal)
	require.Equal(t, alias.PublicName, clientName)
}

func TestUpdateAndDeleteModelRoutingAlias(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.Create(&[]model.Ability{
		{Group: "default", Model: "internal-one", ChannelId: 1, Enabled: true},
		{Group: "default", Model: "internal-two", ChannelId: 1, Enabled: true},
	}).Error)
	alias := model.ModelRoutingAlias{PublicName: "legacy-video", InternalName: "internal-one"}
	require.NoError(t, alias.Insert())

	updated := performModelRoutingAliasRequest(
		t,
		http.MethodPut,
		"/api/model_routing_aliases/",
		`{"id":`+strconv.Itoa(alias.Id)+`,"public_name":"legacy-video","internal_name":"internal-two","note":"switched"}`,
		UpdateModelRoutingAlias,
	)
	require.True(t, updated.Success)
	require.Equal(t, "internal-two", updated.Data.InternalName)

	deleted := performModelRoutingAliasRequest(
		t,
		http.MethodDelete,
		"/api/model_routing_aliases/"+strconv.Itoa(alias.Id),
		"",
		DeleteModelRoutingAlias,
	)
	require.True(t, deleted.Success)
	_, err := model.GetModelRoutingAliasByID(alias.Id)
	require.Error(t, err)
}
