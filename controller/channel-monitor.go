package controller

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type channelMonitorRequest struct {
	ChannelID       int      `json:"channel_id"`
	Scope           string   `json:"scope"`
	Target          string   `json:"target"`
	Name            string   `json:"name"`
	PrimaryModel    string   `json:"primary_model"`
	ExtraModels     []string `json:"extra_models"`
	IntervalSeconds int      `json:"interval_seconds"`
	JitterSeconds   int      `json:"jitter_seconds"`
	Enabled         *bool    `json:"enabled"`
	Visible         *bool    `json:"visible"`
}

type channelMonitorSettingsRequest struct {
	Enabled bool `json:"enabled"`
}

func decodeChannelMonitorRequest(c *gin.Context) (*channelMonitorRequest, bool) {
	var request channelMonitorRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		common.ApiError(c, err)
		return nil, false
	}
	return &request, true
}

func applyChannelMonitorRequest(monitor *model.ChannelMonitor, request *channelMonitorRequest, creating bool) error {
	scope := strings.TrimSpace(request.Scope)
	legacyScope := scope == "" && monitor.Scope == ""
	if scope == "" {
		scope = monitor.Scope
	}
	if scope == "" {
		scope = model.ChannelMonitorScopeText
	}
	scope, err := service.NormalizeChannelMonitorScope(scope)
	if err != nil {
		return err
	}
	primary := ""
	extras := []string{}
	if scope == model.ChannelMonitorScopeText || legacyScope {
		primary, extras, err = service.NormalizeChannelMonitorModels(request.PrimaryModel, request.ExtraModels)
		if err != nil {
			return err
		}
	}
	extraModelsJSON, err := service.EncodeChannelMonitorExtraModels(extras)
	if err != nil {
		return err
	}
	var channel *model.Channel
	if legacyScope {
		channel, err = model.GetChannelById(request.ChannelID, false)
		if err != nil {
			return err
		}
	}
	monitor.ChannelID = 0
	monitor.Target = strings.TrimSpace(request.Target)
	if legacyScope {
		probeKind := service.ResolveChannelMonitorProbeKind(channel.Type, primary, extras)
		if probeKind == model.ChannelMonitorProbeMediaPassive {
			scope = model.ChannelMonitorScopeMedia
		}
		monitor.ChannelID = request.ChannelID
	}
	monitor.Scope = scope
	monitor.Name = strings.TrimSpace(request.Name)
	if monitor.Name == "" {
		switch scope {
		case model.ChannelMonitorScopeText:
			monitor.Name = monitor.Target
		case model.ChannelMonitorScopeImage:
			monitor.Name = "Image aggregation"
		case model.ChannelMonitorScopeVideo:
			monitor.Name = "Video aggregation"
		default:
			monitor.Name = "Media aggregation"
		}
	}
	monitor.PrimaryModel = primary
	monitor.ExtraModelsJSON = extraModelsJSON
	if scope == model.ChannelMonitorScopeText {
		if monitor.Target == "" {
			return errors.New("target group is required")
		}
		if err := service.ValidateTextChannelMonitorTarget(monitor.Target, primary); err != nil {
			return err
		}
		monitor.ProbeKind = model.ChannelMonitorProbeTextActive
	} else {
		monitor.ProbeKind = model.ChannelMonitorProbeMediaPassive
	}
	monitor.IntervalSeconds = request.IntervalSeconds
	monitor.JitterSeconds = request.JitterSeconds
	if creating {
		if monitor.IntervalSeconds == 0 {
			monitor.IntervalSeconds = 300
		}
		if monitor.JitterSeconds == 0 {
			monitor.JitterSeconds = 30
		}
		monitor.Enabled = true
		monitor.Visible = true
	}
	if request.Enabled != nil {
		monitor.Enabled = *request.Enabled
	}
	if request.Visible != nil {
		monitor.Visible = *request.Visible
	}
	if err := service.ValidateChannelMonitor(monitor); err != nil {
		return err
	}
	exists, err := model.ChannelMonitorTargetExists(monitor.ID, monitor.Scope, monitor.Target)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("a monitor for this target already exists")
	}
	return nil
}

func parseChannelMonitorID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		common.ApiError(c, errors.New("invalid channel monitor id"))
		return 0, false
	}
	return id, true
}

func AdminListChannelMonitors(c *gin.Context) {
	windowDays, _ := strconv.Atoi(c.Query("window_days"))
	views, summary, err := service.ListAdminChannelMonitorViews(windowDays)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	textTargets, err := service.ListChannelMonitorTextTargets()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"items": views, "summary": summary, "text_targets": textTargets})
}

func AdminCreateChannelMonitor(c *gin.Context) {
	request, ok := decodeChannelMonitorRequest(c)
	if !ok {
		return
	}
	monitor := &model.ChannelMonitor{}
	if err := applyChannelMonitorRequest(monitor, request, true); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.CreateChannelMonitor(monitor); err != nil {
		common.ApiError(c, err)
		return
	}
	service.InvalidatePublicChannelMonitorCache()
	service.SchedulePublicChannelMonitorRefresh()
	common.ApiSuccess(c, monitor)
}

func AdminUpdateChannelMonitor(c *gin.Context) {
	id, ok := parseChannelMonitorID(c)
	if !ok {
		return
	}
	monitor, err := model.GetChannelMonitorByID(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	request, ok := decodeChannelMonitorRequest(c)
	if !ok {
		return
	}
	if err := applyChannelMonitorRequest(monitor, request, false); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.UpdateChannelMonitor(monitor); err != nil {
		common.ApiError(c, err)
		return
	}
	service.InvalidatePublicChannelMonitorCache()
	service.SchedulePublicChannelMonitorRefresh()
	common.ApiSuccess(c, monitor)
}

func AdminDeleteChannelMonitor(c *gin.Context) {
	id, ok := parseChannelMonitorID(c)
	if !ok {
		return
	}
	if err := model.DeleteChannelMonitor(id); err != nil {
		common.ApiError(c, err)
		return
	}
	service.InvalidatePublicChannelMonitorCache()
	service.SchedulePublicChannelMonitorRefresh()
	common.ApiSuccess(c, nil)
}

func AdminRunChannelMonitor(c *gin.Context) {
	id, ok := parseChannelMonitorID(c)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Minute)
	defer cancel()
	results, err := service.ClaimAndRunChannelMonitor(ctx, id, ProbeChannelMonitor)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	service.InvalidatePublicChannelMonitorCache()
	service.SchedulePublicChannelMonitorRefresh()
	common.ApiSuccess(c, gin.H{"results": results})
}

func AdminUpdateChannelMonitorSettings(c *gin.Context) {
	var request channelMonitorSettingsRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.UpdateOption("ChannelMonitorEnabled", strconv.FormatBool(request.Enabled)); err != nil {
		common.ApiError(c, err)
		return
	}
	service.InvalidatePublicChannelMonitorCache()
	service.SchedulePublicChannelMonitorRefresh()
	common.ApiSuccess(c, gin.H{"enabled": request.Enabled})
}

func ListChannelMonitorStatus(c *gin.Context) {
	if !service.IsChannelMonitorEnabled() {
		common.ApiSuccess(c, gin.H{
			"items":   []service.PublicChannelMonitorItem{},
			"summary": service.PublicChannelMonitorSummary{Enabled: false},
		})
		return
	}
	windowDays, _ := strconv.Atoi(c.Query("window_days"))
	views, summary, err := service.ListPublicChannelMonitorViews(windowDays)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"items": views, "summary": summary})
}

func GetChannelMonitorStatus(c *gin.Context) {
	if !service.IsChannelMonitorEnabled() {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "channel monitor is disabled"})
		return
	}
	id, ok := parseChannelMonitorID(c)
	if !ok {
		return
	}
	monitor, err := model.GetChannelMonitorByID(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if !monitor.Visible || !monitor.Enabled {
		common.ApiError(c, gorm.ErrRecordNotFound)
		return
	}
	windowDays, _ := strconv.Atoi(c.Query("window_days"))
	views, err := service.BuildPublicChannelMonitorViews(monitor, windowDays)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"items": views})
}

// ProbeChannelMonitor adapts the existing channel test path to the monitor
// runner. Media targets are handled passively in the service and never reach
// this active-probe function.
func ProbeChannelMonitor(ctx context.Context, monitor *model.ChannelMonitor, channel *model.Channel, modelName string) service.ChannelMonitorProbeOutcome {
	if service.IsBillableMediaMonitorTarget(channel.Type, modelName) {
		return service.ChannelMonitorProbeOutcome{
			Status:       model.ChannelMonitorStatusUnknown,
			ErrorCode:    "media_passive_only",
			ErrorMessage: service.ErrChannelMonitorMediaProbeDisabled.Error(),
		}
	}
	if monitor.ProbeKind != model.ChannelMonitorProbeTextActive {
		return service.ChannelMonitorProbeOutcome{Status: model.ChannelMonitorStatusUnknown, ErrorCode: "active_probe_disabled"}
	}
	if err := ctx.Err(); err != nil {
		return service.ChannelMonitorProbeOutcome{Status: model.ChannelMonitorStatusUnknown, ErrorCode: "context_cancelled", ErrorMessage: err.Error()}
	}
	testUserID, err := resolveChannelTestUserID(nil)
	if err != nil {
		return service.ChannelMonitorProbeOutcome{Status: model.ChannelMonitorStatusDegraded, ErrorCode: "test_user_unavailable", ErrorMessage: err.Error()}
	}
	startedAt := time.Now()
	result := testChannelWithContext(ctx, channel, testUserID, modelName, "", false)
	latency := int(time.Since(startedAt).Milliseconds())
	outcome := service.ChannelMonitorProbeOutcome{Status: model.ChannelMonitorStatusOperational, LatencyMs: &latency, HTTPStatus: http.StatusOK}
	if result.localErr != nil {
		outcome.Status = model.ChannelMonitorStatusDegraded
		outcome.ErrorCode = "local_probe_error"
		outcome.ErrorMessage = common.MaskSensitiveInfo(result.localErr.Error())
		outcome.HTTPStatus = 0
		return outcome
	}
	if result.newAPIError != nil {
		outcome.Status = model.ChannelMonitorStatusDegraded
		outcome.ErrorCode = string(result.newAPIError.GetErrorCode())
		outcome.ErrorMessage = result.newAPIError.MaskSensitiveError()
		outcome.HTTPStatus = result.newAPIError.StatusCode
		return outcome
	}
	return outcome
}
