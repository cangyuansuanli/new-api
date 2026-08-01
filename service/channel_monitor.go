package service

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"gorm.io/gorm"
)

const (
	channelMonitorMinIntervalSeconds = 60
	channelMonitorMaxIntervalSeconds = 86400
	channelMonitorHistoryDays        = 45
	channelMonitorWorkerConcurrency  = 4
	channelMonitorImageFreshness     = 30 * time.Minute
	channelMonitorVideoFreshness     = 24 * time.Hour
	channelMonitorPassiveQueryLimit  = 100
)

var ErrChannelMonitorMediaProbeDisabled = errors.New("billable media probes are disabled")
var ErrChannelMonitorAlreadyRunning = errors.New("channel monitor is already running")
var ErrChannelMonitorDisabled = errors.New("channel monitor is disabled")

type ChannelMonitorProbeOutcome struct {
	Status       string
	LatencyMs    *int
	HTTPStatus   int
	ErrorCode    string
	ErrorMessage string
}

type ChannelMonitorProbeFunc func(context.Context, *model.ChannelMonitor, *model.Channel, string) ChannelMonitorProbeOutcome

type ChannelMonitorModelStat struct {
	Model          string                         `json:"model"`
	LatestStatus   string                         `json:"latest_status"`
	LatestLatency  *int                           `json:"latest_latency_ms"`
	Availability   *float64                       `json:"availability"`
	AverageLatency *int                           `json:"average_latency_ms"`
	Observed       int                            `json:"observed_checks"`
	Operational    int                            `json:"operational_checks"`
	LatestChecked  *int64                         `json:"latest_checked_at"`
	Timeline       []*ChannelMonitorTimelinePoint `json:"timeline,omitempty"`
}

type ChannelMonitorTimelinePoint struct {
	Status    string `json:"status"`
	LatencyMs *int   `json:"latency_ms"`
	CheckedAt int64  `json:"checked_at"`
}

type ChannelMonitorView struct {
	ID              int64                      `json:"id"`
	Name            string                     `json:"name"`
	Provider        string                     `json:"provider"`
	ProbeKind       string                     `json:"probe_kind"`
	Enabled         bool                       `json:"enabled"`
	Visible         bool                       `json:"visible"`
	IntervalSeconds int                        `json:"interval_seconds"`
	PrimaryModel    string                     `json:"primary_model"`
	Primary         *ChannelMonitorModelStat   `json:"primary"`
	ExtraModels     []*ChannelMonitorModelStat `json:"extra_models"`
	WindowDays      int                        `json:"window_days"`
}

type AdminChannelMonitorView struct {
	*ChannelMonitorView
	ChannelID     int      `json:"channel_id"`
	ChannelName   string   `json:"channel_name"`
	Group         string   `json:"group"`
	JitterSeconds int      `json:"jitter_seconds"`
	ExtraModels   []string `json:"extra_model_names"`
}

type ChannelMonitorRuntimeSummary struct {
	Enabled          bool `json:"enabled"`
	VisibleMonitors  int  `json:"visible_monitors"`
	ObservedMonitors int  `json:"observed_monitors"`
	Operational      int  `json:"operational"`
	Degraded         int  `json:"degraded"`
	Unavailable      int  `json:"unavailable"`
	Unknown          int  `json:"unknown"`
}

const (
	ChannelMonitorCategoryText  = "text"
	ChannelMonitorCategoryImage = "image"
	ChannelMonitorCategoryVideo = "video"
)

// PublicChannelMonitorItem is the user-facing availability contract. It must
// not contain channel identifiers, internal model names, or sample counts.
type PublicChannelMonitorItem struct {
	Name            string   `json:"name"`
	Category        string   `json:"category"`
	LatestStatus    string   `json:"latest_status"`
	Availability    *float64 `json:"availability"`
	AverageLatency  *int     `json:"average_latency_ms"`
	LatestCheckedAt *int64   `json:"latest_checked_at"`
}

type PublicChannelMonitorSummary struct {
	Enabled     bool `json:"enabled"`
	Total       int  `json:"total"`
	Operational int  `json:"operational"`
	Degraded    int  `json:"degraded"`
	Unavailable int  `json:"unavailable"`
	Unknown     int  `json:"unknown"`
}

func IsChannelMonitorEnabled() bool {
	common.OptionMapRWMutex.RLock()
	value, ok := common.OptionMap["ChannelMonitorEnabled"]
	common.OptionMapRWMutex.RUnlock()
	return ok && strings.EqualFold(strings.TrimSpace(value), "true")
}

func NormalizeChannelMonitorWindow(days int) int {
	switch days {
	case 7, 15, 30:
		return days
	default:
		return 7
	}
}

func NormalizeChannelMonitorModels(primary string, extras []string) (string, []string, error) {
	primary = strings.TrimSpace(primary)
	if primary == "" {
		return "", nil, errors.New("primary_model is required")
	}
	seen := map[string]struct{}{primary: {}}
	normalized := make([]string, 0, len(extras))
	for _, candidate := range extras {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if _, exists := seen[candidate]; exists {
			continue
		}
		seen[candidate] = struct{}{}
		normalized = append(normalized, candidate)
	}
	if len(normalized) > 8 {
		return "", nil, errors.New("extra_models cannot contain more than 8 models")
	}
	return primary, normalized, nil
}

func EncodeChannelMonitorExtraModels(models []string) (string, error) {
	data, err := common.Marshal(models)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func DecodeChannelMonitorExtraModels(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}
	var models []string
	if err := common.Unmarshal([]byte(raw), &models); err != nil {
		return []string{}
	}
	return models
}

func ValidateChannelMonitor(monitor *model.ChannelMonitor) error {
	if monitor == nil {
		return errors.New("channel monitor is required")
	}
	if monitor.ChannelID <= 0 {
		return errors.New("channel_id is required")
	}
	if strings.TrimSpace(monitor.Name) == "" {
		return errors.New("name is required")
	}
	if len([]rune(monitor.Name)) > 128 {
		return errors.New("name cannot exceed 128 characters")
	}
	if strings.TrimSpace(monitor.PrimaryModel) == "" {
		return errors.New("primary_model is required")
	}
	if monitor.IntervalSeconds < channelMonitorMinIntervalSeconds || monitor.IntervalSeconds > channelMonitorMaxIntervalSeconds {
		return fmt.Errorf("interval_seconds must be between %d and %d", channelMonitorMinIntervalSeconds, channelMonitorMaxIntervalSeconds)
	}
	if monitor.JitterSeconds < 0 || monitor.JitterSeconds > monitor.IntervalSeconds/2 {
		return errors.New("jitter_seconds must be between 0 and half of interval_seconds")
	}
	switch monitor.ProbeKind {
	case model.ChannelMonitorProbeTextActive, model.ChannelMonitorProbeMediaPassive:
	default:
		return errors.New("probe_kind must be text_active or media_passive")
	}
	return nil
}

func IsBillableMediaMonitorTarget(channelType int, modelName string) bool {
	switch channelType {
	case constant.ChannelTypeMidjourney,
		constant.ChannelTypeMidjourneyPlus,
		constant.ChannelTypeSunoAPI,
		constant.ChannelTypeKling,
		constant.ChannelTypeJimeng,
		constant.ChannelTypeDoubaoVideo,
		constant.ChannelTypeVidu:
		return true
	}
	name := strings.ToLower(strings.TrimSpace(modelName))
	mediaMarkers := []string{
		"gpt-image", "dall-e", "dalle", "seedream", "flux", "imagen", "image-", "-image",
		"firefly", "banana", "stable-diffusion", "sdxl", "recraft", "ideogram",
		"seedance", "video", "veo", "sora", "kling", "vidu", "hailuo", "minimax-hailuo",
		"wan2", "wan-", "luma", "runway", "grok-video", "cogvideo", "hunyuan-video", "pixverse",
		"cy-sd1-omni", "cy-sd4-minimax",
	}
	for _, marker := range mediaMarkers {
		if strings.Contains(name, marker) {
			return true
		}
	}
	return false
}

func ResolveChannelMonitorProbeKind(channelType int, primary string, extras []string) string {
	models := append([]string{primary}, extras...)
	for _, modelName := range models {
		if !IsBillableMediaMonitorTarget(channelType, modelName) {
			return model.ChannelMonitorProbeTextActive
		}
	}
	return model.ChannelMonitorProbeMediaPassive
}

type mediaTaskFailureClass string

const (
	mediaTaskSuccess        mediaTaskFailureClass = "success"
	mediaTaskChannelFailure mediaTaskFailureClass = "channel_failure"
	mediaTaskExcluded       mediaTaskFailureClass = "excluded"
	mediaTaskPlatform       mediaTaskFailureClass = "platform"
	mediaTaskConfiguration  mediaTaskFailureClass = "configuration"
	mediaTaskUnknown        mediaTaskFailureClass = "unknown"
)

func classifyMediaTask(task *model.Task) mediaTaskFailureClass {
	if task == nil {
		return mediaTaskUnknown
	}
	if task.Status == model.TaskStatusSuccess {
		return mediaTaskSuccess
	}
	if task.Status != model.TaskStatusFailure {
		return mediaTaskUnknown
	}
	reason := strings.ToLower(strings.TrimSpace(task.FailReason))
	if reason == "" {
		return mediaTaskUnknown
	}

	excludedMarkers := []string{
		"content policy", "content moderation", "content review", "policy violation",
		"appear to be unsafe", "considered unsafe", "prompt_unsafe", "video_unsafe",
		"请求无法用于生成", "安全政策", "内容审核", "未通过平台", "敏感内容", "真人",
		"prompt is required", "prompt or reference image is required", "reference image rejected",
		"reference images exceed", "at most", "requires exactly one reference image",
		"invalid reference", "invalid image", "unsupported image format",
		"duration must", "resolution must", "aspect ratio", "invalid parameter",
	}
	if containsChannelMonitorMarker(reason, excludedMarkers) {
		return mediaTaskExcluded
	}

	platformMarkers := []string{
		"r2", "rehost", "result upload", "upload result", "object storage",
		"lease lost", "lease expired", "worker", "local queue", "queue timeout",
		"download result", "stream copy", "转存", "对象存储", "队列", "工作节点",
	}
	if containsChannelMonitorMarker(reason, platformMarkers) {
		return mediaTaskPlatform
	}

	configurationMarkers := []string{
		"unsupported endpoint", "unsupported model", "unsupported modality", "not supported by channel",
		"model mapping", "no available channel", "未配置", "不支持的模型", "不支持此端点",
	}
	if containsChannelMonitorMarker(reason, configurationMarkers) {
		return mediaTaskConfiguration
	}

	channelMarkers := []string{
		"bad response status 502", "bad response status 503", "bad response status 504",
		"bad response status code 502", "bad response status code 503", "bad response status code 504",
		"http 403 unauthorized", "unauthorized", "credential", "refresh unavailable",
		"upstream timeout", "upstream request failed", "upstream submit failed", "service unavailable",
		"rate limit", "too many requests", "all cookies failed", "unexpected eof",
		"connection refused", "connection reset", "tls handshake timeout", "context deadline exceeded",
		"生成超时", "服务不可用", "上游超时", "上游请求失败", "上游提交失败", "凭证不可用",
	}
	if containsChannelMonitorMarker(reason, channelMarkers) {
		return mediaTaskChannelFailure
	}
	return mediaTaskUnknown
}

func containsChannelMonitorMarker(value string, markers []string) bool {
	for _, marker := range markers {
		if strings.Contains(value, marker) {
			return true
		}
	}
	return false
}

func isVideoChannelMonitorTarget(channelType int, modelName string) bool {
	switch channelType {
	case constant.ChannelTypeKling, constant.ChannelTypeJimeng, constant.ChannelTypeDoubaoVideo, constant.ChannelTypeVidu:
		return true
	}
	name := strings.ToLower(strings.TrimSpace(modelName))
	return containsChannelMonitorMarker(name, []string{
		"seedance", "video", "veo", "sora", "kling", "vidu", "hailuo", "minimax-hailuo",
		"wan2", "wan-", "luma", "runway", "grok-video", "cogvideo", "hunyuan-video", "pixverse",
		"cy-sd1-omni", "cy-sd4-minimax",
	})
}

func passiveMediaStatus(effective []*model.Task, isVideo bool) string {
	if len(effective) == 0 {
		return model.ChannelMonitorStatusUnknown
	}
	consecutiveFailures := 0
	for _, task := range effective {
		if classifyMediaTask(task) != mediaTaskChannelFailure {
			break
		}
		consecutiveFailures++
	}
	if consecutiveFailures >= 3 {
		return model.ChannelMonitorStatusUnavailable
	}
	if isVideo || len(effective) < 5 {
		if classifyMediaTask(effective[0]) == mediaTaskSuccess {
			return model.ChannelMonitorStatusOperational
		}
		return model.ChannelMonitorStatusDegraded
	}

	recent := effective[:5]
	successes := 0
	for _, task := range recent {
		if classifyMediaTask(task) == mediaTaskSuccess {
			successes++
		}
	}
	if classifyMediaTask(recent[0]) == mediaTaskSuccess && successes >= 4 {
		return model.ChannelMonitorStatusOperational
	}
	if successes == 0 {
		return model.ChannelMonitorStatusUnavailable
	}
	return model.ChannelMonitorStatusDegraded
}

func buildPassiveMediaStat(channel *model.Channel, modelName string, includeTimeline bool) (*ChannelMonitorModelStat, error) {
	isVideo := isVideoChannelMonitorTarget(channel.Type, modelName)
	freshness := channelMonitorImageFreshness
	if isVideo {
		freshness = channelMonitorVideoFreshness
	}
	tasks, err := model.ListRecentMediaTasks(channel.Id, modelName, time.Now().Add(-freshness).Unix(), channelMonitorPassiveQueryLimit)
	if err != nil {
		return nil, err
	}
	effective := make([]*model.Task, 0, len(tasks))
	for _, task := range tasks {
		classification := classifyMediaTask(task)
		if classification == mediaTaskSuccess || classification == mediaTaskChannelFailure {
			effective = append(effective, task)
		}
	}

	stat := &ChannelMonitorModelStat{Model: modelName, LatestStatus: passiveMediaStatus(effective, isVideo)}
	if len(effective) == 0 {
		return stat, nil
	}
	checkedAt := effective[0].UpdatedAt
	stat.LatestChecked = &checkedAt
	availabilityWindow := 20
	if isVideo {
		availabilityWindow = 5
	}
	if len(effective) < availabilityWindow {
		availabilityWindow = len(effective)
	}
	for _, task := range effective[:availabilityWindow] {
		stat.Observed++
		if classifyMediaTask(task) == mediaTaskSuccess {
			stat.Operational++
		}
	}
	availability := float64(stat.Operational) * 100 / float64(stat.Observed)
	stat.Availability = &availability

	if includeTimeline {
		timelineLimit := 24
		if len(effective) < timelineLimit {
			timelineLimit = len(effective)
		}
		stat.Timeline = make([]*ChannelMonitorTimelinePoint, 0, timelineLimit)
		for i := timelineLimit - 1; i >= 0; i-- {
			status := model.ChannelMonitorStatusDegraded
			if classifyMediaTask(effective[i]) == mediaTaskSuccess {
				status = model.ChannelMonitorStatusOperational
			}
			stat.Timeline = append(stat.Timeline, &ChannelMonitorTimelinePoint{Status: status, CheckedAt: effective[i].UpdatedAt})
		}
	}
	return stat, nil
}

func ChannelMonitorModels(monitor *model.ChannelMonitor) []string {
	if monitor == nil {
		return nil
	}
	models := []string{strings.TrimSpace(monitor.PrimaryModel)}
	models = append(models, DecodeChannelMonitorExtraModels(monitor.ExtraModelsJSON)...)
	return models
}

func RunChannelMonitor(ctx context.Context, monitorID int64, probe ChannelMonitorProbeFunc) ([]*model.ChannelMonitorResult, error) {
	monitor, err := model.GetChannelMonitorByID(monitorID)
	if err != nil {
		return nil, err
	}
	channel, err := model.GetChannelById(monitor.ChannelID, true)
	if err != nil {
		return nil, err
	}
	models := ChannelMonitorModels(monitor)
	results := make([]*model.ChannelMonitorResult, 0, len(models))
	for _, modelName := range models {
		if err := ctx.Err(); err != nil {
			return results, err
		}
		outcome := ChannelMonitorProbeOutcome{Status: model.ChannelMonitorStatusUnknown}
		if IsBillableMediaMonitorTarget(channel.Type, modelName) {
			stat, passiveErr := buildPassiveMediaStat(channel, modelName, false)
			if passiveErr != nil {
				return results, passiveErr
			}
			outcome.Status = stat.LatestStatus
			outcome.ErrorCode = "passive_observed"
			if stat.LatestChecked == nil {
				outcome.ErrorCode = "passive_no_recent_sample"
			}
		} else if monitor.ProbeKind == model.ChannelMonitorProbeTextActive {
			if probe == nil {
				return results, errors.New("channel monitor probe is not configured")
			}
			outcome = probe(ctx, monitor, channel, modelName)
		}
		if outcome.Status == "" {
			outcome.Status = model.ChannelMonitorStatusUnknown
		}
		if len(outcome.ErrorMessage) > 512 {
			outcome.ErrorMessage = outcome.ErrorMessage[:512]
		}
		result := &model.ChannelMonitorResult{
			MonitorID:    monitor.ID,
			Model:        modelName,
			Status:       outcome.Status,
			LatencyMs:    outcome.LatencyMs,
			HTTPStatus:   outcome.HTTPStatus,
			ErrorCode:    outcome.ErrorCode,
			ErrorMessage: outcome.ErrorMessage,
			CheckedAt:    time.Now().Unix(),
		}
		if err := model.CreateChannelMonitorResult(result); err != nil {
			return results, err
		}
		results = append(results, result)
	}
	return results, nil
}

func ClaimAndRunChannelMonitor(ctx context.Context, monitorID int64, probe ChannelMonitorProbeFunc) ([]*model.ChannelMonitorResult, error) {
	if !IsChannelMonitorEnabled() {
		return nil, ErrChannelMonitorDisabled
	}
	monitor, err := model.GetChannelMonitorByID(monitorID)
	if err != nil {
		return nil, err
	}
	now := time.Now().Unix()
	claimed, err := model.ClaimChannelMonitor(
		monitorID,
		now,
		now+int64(monitor.IntervalSeconds),
		now+int64((2*time.Minute)/time.Second),
		true,
	)
	if err != nil {
		return nil, err
	}
	if !claimed {
		return nil, ErrChannelMonitorAlreadyRunning
	}
	defer func() { _ = model.ReleaseChannelMonitorLease(monitorID) }()
	return RunChannelMonitor(ctx, monitorID, probe)
}

func buildChannelMonitorModelStat(modelName string, results []*model.ChannelMonitorResult, includeTimeline bool) *ChannelMonitorModelStat {
	stat := &ChannelMonitorModelStat{Model: modelName, LatestStatus: model.ChannelMonitorStatusUnknown}
	var totalLatency int64
	var latencySamples int
	modelResults := make([]*model.ChannelMonitorResult, 0)
	for _, result := range results {
		if result.Model != modelName {
			continue
		}
		modelResults = append(modelResults, result)
		if stat.LatestChecked == nil || result.CheckedAt >= *stat.LatestChecked {
			checked := result.CheckedAt
			stat.LatestChecked = &checked
			stat.LatestStatus = result.Status
			stat.LatestLatency = result.LatencyMs
		}
		if result.Status == model.ChannelMonitorStatusOperational ||
			result.Status == model.ChannelMonitorStatusDegraded ||
			result.Status == model.ChannelMonitorStatusUnavailable {
			stat.Observed++
			if result.Status == model.ChannelMonitorStatusOperational {
				stat.Operational++
			}
		}
		if result.LatencyMs != nil {
			totalLatency += int64(*result.LatencyMs)
			latencySamples++
		}
	}
	if stat.Observed > 0 {
		availability := float64(stat.Operational) * 100 / float64(stat.Observed)
		stat.Availability = &availability
	}
	if latencySamples > 0 {
		average := int(totalLatency / int64(latencySamples))
		stat.AverageLatency = &average
	}
	if includeTimeline {
		sort.Slice(modelResults, func(i, j int) bool { return modelResults[i].CheckedAt < modelResults[j].CheckedAt })
		if len(modelResults) > 24 {
			modelResults = modelResults[len(modelResults)-24:]
		}
		stat.Timeline = make([]*ChannelMonitorTimelinePoint, 0, len(modelResults))
		for _, result := range modelResults {
			stat.Timeline = append(stat.Timeline, &ChannelMonitorTimelinePoint{
				Status: result.Status, LatencyMs: result.LatencyMs, CheckedAt: result.CheckedAt,
			})
		}
	}
	return stat
}

func BuildChannelMonitorView(monitor *model.ChannelMonitor, windowDays int, includeTimeline bool) (*ChannelMonitorView, error) {
	channel, err := model.GetChannelById(monitor.ChannelID, false)
	if err != nil {
		return nil, err
	}
	windowDays = NormalizeChannelMonitorWindow(windowDays)
	results, err := model.ListChannelMonitorResults(monitor.ID, time.Now().Add(-time.Duration(windowDays)*24*time.Hour).Unix())
	if err != nil {
		return nil, err
	}
	view := &ChannelMonitorView{
		ID: monitor.ID, Name: monitor.Name,
		Provider:  constant.GetChannelTypeName(channel.Type),
		ProbeKind: monitor.ProbeKind, Enabled: monitor.Enabled, Visible: monitor.Visible,
		IntervalSeconds: monitor.IntervalSeconds, PrimaryModel: monitor.PrimaryModel, WindowDays: windowDays,
	}
	view.Primary = buildChannelMonitorModelStat(monitor.PrimaryModel, results, includeTimeline)
	if IsBillableMediaMonitorTarget(channel.Type, monitor.PrimaryModel) {
		view.Primary, err = buildPassiveMediaStat(channel, monitor.PrimaryModel, includeTimeline)
		if err != nil {
			return nil, err
		}
	}
	for _, modelName := range DecodeChannelMonitorExtraModels(monitor.ExtraModelsJSON) {
		stat := buildChannelMonitorModelStat(modelName, results, false)
		if IsBillableMediaMonitorTarget(channel.Type, modelName) {
			stat, err = buildPassiveMediaStat(channel, modelName, false)
			if err != nil {
				return nil, err
			}
		}
		view.ExtraModels = append(view.ExtraModels, stat)
	}
	if view.ExtraModels == nil {
		view.ExtraModels = []*ChannelMonitorModelStat{}
	}
	return view, nil
}

func ListChannelMonitorViews(windowDays int, visibleOnly bool) ([]*ChannelMonitorView, *ChannelMonitorRuntimeSummary, error) {
	monitors, err := model.ListChannelMonitors(visibleOnly, true)
	if err != nil {
		return nil, nil, err
	}
	summary := &ChannelMonitorRuntimeSummary{Enabled: IsChannelMonitorEnabled(), VisibleMonitors: len(monitors)}
	views := make([]*ChannelMonitorView, 0, len(monitors))
	for _, monitor := range monitors {
		view, err := BuildChannelMonitorView(monitor, windowDays, true)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return nil, nil, err
		}
		views = append(views, view)
		switch view.Primary.LatestStatus {
		case model.ChannelMonitorStatusOperational:
			summary.ObservedMonitors++
			summary.Operational++
		case model.ChannelMonitorStatusDegraded:
			summary.ObservedMonitors++
			summary.Degraded++
		case model.ChannelMonitorStatusUnavailable:
			summary.ObservedMonitors++
			summary.Unavailable++
		default:
			summary.Unknown++
		}
	}
	return views, summary, nil
}

type publicChannelMonitorAggregate struct {
	item                *PublicChannelMonitorItem
	observed            int
	operational         int
	latencyTotal        int64
	latencyMeasurements int
}

func publicChannelMonitorCategory(channelType int, modelName string) string {
	if isVideoChannelMonitorTarget(channelType, modelName) {
		return ChannelMonitorCategoryVideo
	}
	if channelType == constant.ChannelTypeSunoAPI {
		return ""
	}
	return ChannelMonitorCategoryImage
}

func publicStatusPriority(status string) int {
	switch status {
	case model.ChannelMonitorStatusOperational:
		return 4
	case model.ChannelMonitorStatusDegraded:
		return 3
	case model.ChannelMonitorStatusUnavailable:
		return 2
	default:
		return 1
	}
}

func mergePublicChannelMonitorStat(
	aggregates map[string]*publicChannelMonitorAggregate,
	name string,
	category string,
	stat *ChannelMonitorModelStat,
) {
	name = strings.TrimSpace(name)
	if name == "" || stat == nil {
		return
	}
	key := category + "\x00" + name
	aggregate, exists := aggregates[key]
	if !exists {
		aggregate = &publicChannelMonitorAggregate{item: &PublicChannelMonitorItem{
			Name: name, Category: category, LatestStatus: model.ChannelMonitorStatusUnknown,
		}}
		aggregates[key] = aggregate
	}
	if publicStatusPriority(stat.LatestStatus) > publicStatusPriority(aggregate.item.LatestStatus) {
		aggregate.item.LatestStatus = stat.LatestStatus
	}
	if stat.LatestChecked != nil && (aggregate.item.LatestCheckedAt == nil || *stat.LatestChecked > *aggregate.item.LatestCheckedAt) {
		checkedAt := *stat.LatestChecked
		aggregate.item.LatestCheckedAt = &checkedAt
	}
	aggregate.observed += stat.Observed
	aggregate.operational += stat.Operational
	if stat.AverageLatency != nil {
		aggregate.latencyTotal += int64(*stat.AverageLatency)
		aggregate.latencyMeasurements++
	}
}

func buildPublicChannelMonitorItems(monitors []*model.ChannelMonitor, windowDays int) ([]*PublicChannelMonitorItem, error) {
	aggregates := make(map[string]*publicChannelMonitorAggregate)
	for _, monitor := range monitors {
		channel, err := model.GetChannelById(monitor.ChannelID, false)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return nil, err
		}

		if monitor.ProbeKind == model.ChannelMonitorProbeTextActive && !IsBillableMediaMonitorTarget(channel.Type, monitor.PrimaryModel) {
			view, err := BuildChannelMonitorView(monitor, windowDays, false)
			if err != nil {
				return nil, err
			}
			for _, group := range channel.GetGroups() {
				mergePublicChannelMonitorStat(aggregates, group, ChannelMonitorCategoryText, view.Primary)
			}
		}

		for _, modelName := range channel.GetModels() {
			if !IsBillableMediaMonitorTarget(channel.Type, modelName) {
				continue
			}
			category := publicChannelMonitorCategory(channel.Type, modelName)
			if category == "" {
				continue
			}
			stat, err := buildPassiveMediaStat(channel, modelName, false)
			if err != nil {
				return nil, err
			}
			mergePublicChannelMonitorStat(
				aggregates,
				ToPublicModelName(modelName),
				category,
				stat,
			)
		}
	}

	items := make([]*PublicChannelMonitorItem, 0, len(aggregates))
	for _, aggregate := range aggregates {
		if aggregate.observed > 0 {
			availability := float64(aggregate.operational) * 100 / float64(aggregate.observed)
			aggregate.item.Availability = &availability
		}
		if aggregate.latencyMeasurements > 0 {
			latency := int(aggregate.latencyTotal / int64(aggregate.latencyMeasurements))
			aggregate.item.AverageLatency = &latency
		}
		items = append(items, aggregate.item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Category != items[j].Category {
			return items[i].Category < items[j].Category
		}
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
	return items, nil
}

func summarizePublicChannelMonitorItems(items []*PublicChannelMonitorItem) *PublicChannelMonitorSummary {
	summary := &PublicChannelMonitorSummary{Enabled: IsChannelMonitorEnabled(), Total: len(items)}
	for _, item := range items {
		switch item.LatestStatus {
		case model.ChannelMonitorStatusOperational:
			summary.Operational++
		case model.ChannelMonitorStatusDegraded:
			summary.Degraded++
		case model.ChannelMonitorStatusUnavailable:
			summary.Unavailable++
		default:
			summary.Unknown++
		}
	}
	return summary
}

func ListPublicChannelMonitorViews(windowDays int) ([]*PublicChannelMonitorItem, *PublicChannelMonitorSummary, error) {
	monitors, err := model.ListChannelMonitors(true, true)
	if err != nil {
		return nil, nil, err
	}
	items, err := buildPublicChannelMonitorItems(monitors, windowDays)
	if err != nil {
		return nil, nil, err
	}
	return items, summarizePublicChannelMonitorItems(items), nil
}

func BuildPublicChannelMonitorViews(monitor *model.ChannelMonitor, windowDays int) ([]*PublicChannelMonitorItem, error) {
	if monitor == nil {
		return nil, errors.New("channel monitor is required")
	}
	return buildPublicChannelMonitorItems([]*model.ChannelMonitor{monitor}, windowDays)
}

func ListAdminChannelMonitorViews(windowDays int) ([]*AdminChannelMonitorView, *ChannelMonitorRuntimeSummary, error) {
	monitors, err := model.ListChannelMonitors(false, false)
	if err != nil {
		return nil, nil, err
	}
	summary := &ChannelMonitorRuntimeSummary{Enabled: IsChannelMonitorEnabled(), VisibleMonitors: len(monitors)}
	views := make([]*AdminChannelMonitorView, 0, len(monitors))
	for _, monitor := range monitors {
		view, err := BuildChannelMonitorView(monitor, windowDays, true)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return nil, nil, err
		}
		channel, err := model.GetChannelById(monitor.ChannelID, false)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return nil, nil, err
		}
		views = append(views, &AdminChannelMonitorView{
			ChannelMonitorView: view,
			ChannelID:          monitor.ChannelID,
			ChannelName:        channel.Name,
			Group:              channel.Group,
			JitterSeconds:      monitor.JitterSeconds,
			ExtraModels:        DecodeChannelMonitorExtraModels(monitor.ExtraModelsJSON),
		})
		switch view.Primary.LatestStatus {
		case model.ChannelMonitorStatusOperational:
			summary.ObservedMonitors++
			summary.Operational++
		case model.ChannelMonitorStatusDegraded:
			summary.ObservedMonitors++
			summary.Degraded++
		case model.ChannelMonitorStatusUnavailable:
			summary.ObservedMonitors++
			summary.Unavailable++
		default:
			summary.Unknown++
		}
	}
	return views, summary, nil
}

var (
	channelMonitorRunnerOnce sync.Once
	channelMonitorInFlight   sync.Map
	channelMonitorWorkers    = make(chan struct{}, channelMonitorWorkerConcurrency)
)

func StartChannelMonitorRunner(probe ChannelMonitorProbeFunc) {
	channelMonitorRunnerOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}
		go func() {
			ticker := time.NewTicker(15 * time.Second)
			cleanupTicker := time.NewTicker(6 * time.Hour)
			defer ticker.Stop()
			defer cleanupTicker.Stop()
			runDueChannelMonitors(probe)
			for {
				select {
				case <-ticker.C:
					runDueChannelMonitors(probe)
				case <-cleanupTicker.C:
					_, _ = model.DeleteChannelMonitorResultsBefore(time.Now().Add(-channelMonitorHistoryDays * 24 * time.Hour).Unix())
				}
			}
		}()
	})
}

func runDueChannelMonitors(probe ChannelMonitorProbeFunc) {
	if !IsChannelMonitorEnabled() {
		return
	}
	monitors, err := model.ListChannelMonitors(false, true)
	if err != nil {
		common.SysLog("channel monitor: list enabled monitors failed: " + err.Error())
		return
	}
	for _, monitor := range monitors {
		select {
		case channelMonitorWorkers <- struct{}{}:
		default:
			return
		}
		jitter := 0
		if monitor.JitterSeconds > 0 {
			jitter = rand.Intn(monitor.JitterSeconds*2+1) - monitor.JitterSeconds
		}
		now := time.Now().Unix()
		claimed, err := model.ClaimChannelMonitor(
			monitor.ID,
			now,
			now+int64(monitor.IntervalSeconds+jitter),
			now+int64((2*time.Minute)/time.Second),
			false,
		)
		if err != nil || !claimed {
			<-channelMonitorWorkers
			continue
		}
		if _, loaded := channelMonitorInFlight.LoadOrStore(monitor.ID, struct{}{}); loaded {
			_ = model.ReleaseChannelMonitorLease(monitor.ID)
			<-channelMonitorWorkers
			continue
		}
		go func(id int64) {
			defer func() {
				<-channelMonitorWorkers
				channelMonitorInFlight.Delete(id)
				_ = model.ReleaseChannelMonitorLease(id)
			}()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			if _, err := RunChannelMonitor(ctx, id, probe); err != nil {
				common.SysLog(fmt.Sprintf("channel monitor %d failed: %v", id, err))
			}
		}(monitor.ID)
	}
}
