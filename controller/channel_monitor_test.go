package controller

import (
	"bytes"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupChannelMonitorControllerTest(t *testing.T) {
	t.Helper()
	originalDB := model.DB
	originalLogDB := model.LOG_DB
	gin.SetMode(gin.TestMode)
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(
		&model.Channel{},
		&model.Model{},
		&model.ChannelMonitor{},
		&model.ChannelMonitorResult{},
		&model.Task{},
		&model.Option{},
	))
	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	originalEnabled, hadOriginalEnabled := common.OptionMap["ChannelMonitorEnabled"]
	common.OptionMap["ChannelMonitorEnabled"] = "false"
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		if hadOriginalEnabled {
			common.OptionMap["ChannelMonitorEnabled"] = originalEnabled
		} else {
			delete(common.OptionMap, "ChannelMonitorEnabled")
		}
		common.OptionMapRWMutex.Unlock()
		model.DB = originalDB
		model.LOG_DB = originalLogDB
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
}

func TestListChannelMonitorStatusReturnsEmptyWhenDisabled(t *testing.T) {
	setupChannelMonitorControllerTest(t)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("GET", "/api/channel-monitors", nil)

	ListChannelMonitorStatus(ctx)

	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"enabled":false`)
	assert.Contains(t, recorder.Body.String(), `"items":[]`)
}

func TestAdminCreateMediaMonitorUsesPassiveProbe(t *testing.T) {
	setupChannelMonitorControllerTest(t)
	channel := &model.Channel{
		Name: "media-channel", Type: constant.ChannelTypeOpenAI,
		Key: "sensitive-key", Status: common.ChannelStatusEnabled,
		Models: "gpt-image-1", Group: "default",
	}
	require.NoError(t, model.DB.Create(channel).Error)
	body := fmt.Sprintf(`{
		"channel_id": %d,
		"name": "Images",
		"primary_model": "gpt-image-1",
		"extra_models": [],
		"interval_seconds": 300,
		"jitter_seconds": 30,
		"enabled": true,
		"visible": true
	}`, channel.Id)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("POST", "/api/channel-monitors/admin", bytes.NewBufferString(body))

	AdminCreateChannelMonitor(ctx)

	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"success":true`)
	monitor, err := model.GetChannelMonitorByChannelID(channel.Id)
	require.NoError(t, err)
	assert.Equal(t, model.ChannelMonitorProbeMediaPassive, monitor.ProbeKind)
	assert.NotContains(t, recorder.Body.String(), "sensitive-key")
}

func TestAdminCreateGlobalImageMonitorDoesNotRequireChannelOrModel(t *testing.T) {
	setupChannelMonitorControllerTest(t)
	body := `{
		"scope": "image",
		"target": "",
		"channel_id": 0,
		"name": "Images",
		"primary_model": "",
		"extra_models": [],
		"interval_seconds": 1800,
		"jitter_seconds": 0,
		"enabled": true,
		"visible": true
	}`
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("POST", "/api/channel-monitors/admin", bytes.NewBufferString(body))

	AdminCreateChannelMonitor(ctx)

	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"success":true`)
	var monitor model.ChannelMonitor
	require.NoError(t, model.DB.Where("scope = ?", model.ChannelMonitorScopeImage).First(&monitor).Error)
	assert.Zero(t, monitor.ChannelID)
	assert.Empty(t, monitor.PrimaryModel)
	assert.Equal(t, model.ChannelMonitorProbeMediaPassive, monitor.ProbeKind)
}

func TestGetChannelMonitorStatusHidesDisabledMonitor(t *testing.T) {
	setupChannelMonitorControllerTest(t)
	common.OptionMapRWMutex.Lock()
	common.OptionMap["ChannelMonitorEnabled"] = "true"
	common.OptionMapRWMutex.Unlock()
	channel := &model.Channel{
		Name: "text-channel", Type: constant.ChannelTypeOpenAI,
		Key: "sensitive-key", Status: common.ChannelStatusEnabled,
		Models: "gpt-4o-mini", Group: "default",
	}
	require.NoError(t, model.DB.Create(channel).Error)
	monitor := &model.ChannelMonitor{
		ChannelID: channel.Id, Name: "Paused monitor", PrimaryModel: "gpt-4o-mini",
		ExtraModelsJSON: "[]", ProbeKind: model.ChannelMonitorProbeTextActive,
		IntervalSeconds: 300, Enabled: false, Visible: true,
	}
	require.NoError(t, model.CreateChannelMonitor(monitor))
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(monitor.ID)}}
	ctx.Request = httptest.NewRequest("GET", fmt.Sprintf("/api/channel-monitors/%d/status", monitor.ID), nil)

	GetChannelMonitorStatus(ctx)

	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"success":false`)
	assert.NotContains(t, recorder.Body.String(), `"Paused monitor"`)
}

func TestListChannelMonitorStatusUsesPublicContract(t *testing.T) {
	setupChannelMonitorControllerTest(t)
	common.OptionMapRWMutex.Lock()
	common.OptionMap["ChannelMonitorEnabled"] = "true"
	common.OptionMapRWMutex.Unlock()
	channel := &model.Channel{
		Name: "internal-channel", Type: constant.ChannelTypeOpenAI,
		Key: "sensitive-key", Status: common.ChannelStatusEnabled,
		Models: "internal-gpt-primary,internal-gpt-extra", Group: "LLM-GPT-pro",
	}
	require.NoError(t, model.DB.Create(channel).Error)
	monitor := &model.ChannelMonitor{
		ChannelID: channel.Id, Name: "Internal monitor", PrimaryModel: "internal-gpt-primary",
		ExtraModelsJSON: `["internal-gpt-extra"]`, ProbeKind: model.ChannelMonitorProbeTextActive,
		IntervalSeconds: 300, Enabled: true, Visible: true,
	}
	require.NoError(t, model.CreateChannelMonitor(monitor))
	require.NoError(t, model.CreateChannelMonitorResult(&model.ChannelMonitorResult{
		MonitorID: monitor.ID, Model: monitor.PrimaryModel, Status: model.ChannelMonitorStatusOperational,
		HTTPStatus: 200, ErrorCode: "should_not_leak", ErrorMessage: "sensitive-upstream-error", CheckedAt: time.Now().Unix(),
	}))
	service.RefreshPublicChannelMonitorSnapshots()

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("GET", "/api/channel-monitors", nil)
	ListChannelMonitorStatus(ctx)

	body := recorder.Body.String()
	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, body, `"name":"LLM-GPT-pro"`)
	assert.Contains(t, body, `"category":"text"`)
	assert.Contains(t, body, `"timeline":[`)
	assert.Contains(t, body, `{"status":"operational"}]`)
	for _, privateValue := range []string{
		"observed_checks", "operational_checks", "channel_id", "channel_name",
		"internal-gpt-primary", "internal-gpt-extra", "Internal monitor", "internal-channel",
		"latest_checked_at", `"checked_at":`, `"latency_ms":`, "http_status", "error_code",
		"sensitive-key", "sensitive-upstream-error",
	} {
		assert.NotContains(t, body, privateValue)
	}
}
