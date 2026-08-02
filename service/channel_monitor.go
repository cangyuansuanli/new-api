package service

import (
	"context"
	"crypto/sha256"
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
	"github.com/QuantumNous/new-api/pkg/cachex"
	"github.com/samber/hot"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
)

const (
	channelMonitorMinIntervalSeconds  = 60
	channelMonitorMaxIntervalSeconds  = 86400
	channelMonitorHistoryDays         = 45
	channelMonitorWorkerConcurrency   = 4
	channelMonitorImageFreshness      = 30 * time.Minute
	channelMonitorVideoFreshness      = 24 * time.Hour
	channelMonitorPassiveQueryLimit   = 100
	channelMonitorPublicTimelineLimit = 48
	channelMonitorPublicBucketSize    = 30 * time.Minute
	channelMonitorCarryLookback       = 24 * time.Hour
	channelMonitorPublicTextRefresh   = 5 * time.Minute
	channelMonitorPublicRefresh       = 30 * time.Minute
	channelMonitorPublicCacheTTL      = 2 * time.Hour
	channelMonitorPublicCacheNS       = "new-api:channel_monitor_public:v4"
	channelMonitorMediaCacheTTL       = 49 * time.Hour
	channelMonitorMediaCacheNS        = "new-api:channel_monitor_media:v1"
)

var ErrChannelMonitorMediaProbeDisabled = errors.New("billable media probes are disabled")
var ErrChannelMonitorAlreadyRunning = errors.New("channel monitor is already running")
var ErrChannelMonitorDisabled = errors.New("channel monitor is disabled")

type publicChannelMonitorCacheItem struct {
	Items   []*PublicChannelMonitorItem  `json:"items"`
	Summary *PublicChannelMonitorSummary `json:"summary"`
}

type channelMonitorMediaCacheItem struct {
	Cursor int64         `json:"cursor"`
	Tasks  []*model.Task `json:"tasks"`
}

var (
	publicChannelMonitorCacheOnce sync.Once
	publicChannelMonitorCache     *cachex.HybridCache[publicChannelMonitorCacheItem]
	publicChannelMonitorFlight    singleflight.Group
	channelMonitorMediaCacheOnce  sync.Once
	channelMonitorMediaCache      *cachex.HybridCache[channelMonitorMediaCacheItem]
	channelMonitorMediaFlight     singleflight.Group
)

func getPublicChannelMonitorCache() *cachex.HybridCache[publicChannelMonitorCacheItem] {
	publicChannelMonitorCacheOnce.Do(func() {
		publicChannelMonitorCache = cachex.NewHybridCache(cachex.HybridCacheConfig[publicChannelMonitorCacheItem]{
			Namespace: cachex.Namespace(channelMonitorPublicCacheNS),
			Redis:     common.RDB,
			RedisEnabled: func() bool {
				return common.RedisEnabled && common.RDB != nil
			},
			RedisCodec: cachex.JSONCodec[publicChannelMonitorCacheItem]{},
			Memory: func() *hot.HotCache[string, publicChannelMonitorCacheItem] {
				return hot.NewHotCache[string, publicChannelMonitorCacheItem](hot.LRU, 8).
					WithTTL(channelMonitorPublicCacheTTL).
					WithJanitor().
					Build()
			},
		})
	})
	return publicChannelMonitorCache
}

func getChannelMonitorMediaCache() *cachex.HybridCache[channelMonitorMediaCacheItem] {
	channelMonitorMediaCacheOnce.Do(func() {
		channelMonitorMediaCache = cachex.NewHybridCache(cachex.HybridCacheConfig[channelMonitorMediaCacheItem]{
			Namespace: cachex.Namespace(channelMonitorMediaCacheNS),
			Redis:     common.RDB,
			RedisEnabled: func() bool {
				return common.RedisEnabled && common.RDB != nil
			},
			RedisCodec: cachex.JSONCodec[channelMonitorMediaCacheItem]{},
			Memory: func() *hot.HotCache[string, channelMonitorMediaCacheItem] {
				return hot.NewHotCache[string, channelMonitorMediaCacheItem](hot.LRU, 8).
					WithTTL(channelMonitorMediaCacheTTL).
					WithJanitor().
					Build()
			},
		})
	})
	return channelMonitorMediaCache
}

func InvalidatePublicChannelMonitorCache() {
	if err := getPublicChannelMonitorCache().Purge(); err != nil {
		common.SysError(fmt.Sprintf("public channel monitor cache purge failed: %v", err))
	}
}

func SchedulePublicChannelMonitorRefresh() {
	go RefreshPublicChannelMonitorSnapshots()
}

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
	Carried   bool   `json:"carried,omitempty"`
}

type ChannelMonitorView struct {
	ID              int64                      `json:"id"`
	Name            string                     `json:"name"`
	Provider        string                     `json:"provider"`
	ProbeKind       string                     `json:"probe_kind"`
	Scope           string                     `json:"scope"`
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
	Target        string   `json:"target"`
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

type ChannelMonitorTextTarget struct {
	Group  string   `json:"group"`
	Models []string `json:"models"`
}

const (
	ChannelMonitorCategoryText  = "text"
	ChannelMonitorCategoryImage = "image"
	ChannelMonitorCategoryVideo = "video"
)

// PublicChannelMonitorItem is the user-facing availability contract. It must
// not contain channel identifiers, internal model names, sample counts, or
// per-observation timestamps and latency.
type PublicChannelMonitorItem struct {
	Name           string                               `json:"name"`
	Category       string                               `json:"category"`
	LatestStatus   string                               `json:"latest_status"`
	Availability   *float64                             `json:"availability"`
	AverageLatency *int                                 `json:"average_latency_ms"`
	Timeline       []*PublicChannelMonitorTimelinePoint `json:"timeline"`
}

type PublicChannelMonitorTimelinePoint struct {
	Status  string `json:"status"`
	Carried bool   `json:"carried,omitempty"`
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

func NormalizeChannelMonitorScope(scope string) (string, error) {
	switch strings.TrimSpace(scope) {
	case model.ChannelMonitorScopeText, model.ChannelMonitorScopeImage, model.ChannelMonitorScopeVideo, model.ChannelMonitorScopeMedia:
		return strings.TrimSpace(scope), nil
	default:
		return "", errors.New("scope must be text, image, or video")
	}
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
	if strings.TrimSpace(monitor.Name) == "" {
		return errors.New("name is required")
	}
	if len([]rune(monitor.Name)) > 128 {
		return errors.New("name cannot exceed 128 characters")
	}
	if monitor.Scope == model.ChannelMonitorScopeText && strings.TrimSpace(monitor.PrimaryModel) == "" {
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

func ValidateTextChannelMonitorTarget(group string, modelName string) error {
	channels, err := model.GetEnabledChannels(false)
	if err != nil {
		return err
	}
	for _, channel := range channels {
		if channelSupportsMonitorGroup(channel, strings.TrimSpace(group)) && channelSupportsMonitorModel(channel, modelName) && !IsBillableMediaMonitorTarget(channel.Type, modelName) {
			return nil
		}
	}
	return errors.New("no enabled channel in the group supports the selected text model")
}

func channelMonitorAcceptsCategory(monitor *model.ChannelMonitor, category string) bool {
	if monitor == nil {
		return false
	}
	switch monitor.Scope {
	case model.ChannelMonitorScopeText:
		return category == ChannelMonitorCategoryText
	case model.ChannelMonitorScopeImage:
		return category == ChannelMonitorCategoryImage
	case model.ChannelMonitorScopeVideo:
		return category == ChannelMonitorCategoryVideo
	case "":
		return true
	default:
		return category == ChannelMonitorCategoryImage || category == ChannelMonitorCategoryVideo
	}
}

func disabledChannelMonitorModels() (map[string]struct{}, error) {
	return model.GetDisabledExactModelNames()
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

func passiveMediaStatus(effective []*model.Task, _ bool) string {
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
	successes := 0
	for _, task := range effective {
		if classifyMediaTask(task) == mediaTaskSuccess {
			successes++
		}
	}
	if successes*100 >= len(effective)*80 {
		return model.ChannelMonitorStatusOperational
	}
	if successes == 0 {
		return model.ChannelMonitorStatusUnavailable
	}
	return model.ChannelMonitorStatusDegraded
}

func taskMatchesChannelMonitorModel(task *model.Task, modelName string) bool {
	if task == nil {
		return false
	}
	modelName = strings.TrimSpace(modelName)
	publicName := ToPublicModelName(modelName)
	for _, taskModelName := range []string{
		task.Properties.ClientModelName,
		task.Properties.OriginModelName,
		task.Properties.UpstreamModelName,
	} {
		taskModelName = strings.TrimSpace(taskModelName)
		if taskModelName != "" && (taskModelName == modelName || ToPublicModelName(taskModelName) == publicName) {
			return true
		}
	}
	return false
}

func channelMonitorModelAliases(modelName string) []string {
	aliases := []string{strings.TrimSpace(modelName), ToPublicModelName(modelName)}
	seen := make(map[string]struct{}, len(aliases))
	result := make([]string, 0, len(aliases))
	for _, alias := range aliases {
		if alias == "" {
			continue
		}
		if _, exists := seen[alias]; exists {
			continue
		}
		seen[alias] = struct{}{}
		result = append(result, alias)
	}
	return result
}

func buildPassiveMediaStatFromTasks(channel *model.Channel, modelName string, tasks []*model.Task, includeTimeline bool) *ChannelMonitorModelStat {
	return buildPassiveMediaStatFromTasksAt(channel, modelName, tasks, includeTimeline, time.Now())
}

func buildPassiveMediaStatFromTasksAt(channel *model.Channel, modelName string, tasks []*model.Task, includeTimeline bool, now time.Time) *ChannelMonitorModelStat {
	isVideo := isVideoChannelMonitorTarget(channel.Type, modelName)
	bucketSeconds := int64(channelMonitorPublicBucketSize / time.Second)
	timelineEnd := ((now.Unix() / bucketSeconds) + 1) * bucketSeconds
	timelineStart := timelineEnd - int64(channelMonitorPublicTimelineLimit)*bucketSeconds
	lookbackStart := timelineStart - int64(channelMonitorCarryLookback/time.Second)
	effective := make([]*model.Task, 0, len(tasks))
	for _, task := range tasks {
		if task.UpdatedAt < lookbackStart || task.UpdatedAt >= timelineEnd || !taskMatchesChannelMonitorModel(task, modelName) {
			continue
		}
		classification := classifyMediaTask(task)
		if classification == mediaTaskSuccess || classification == mediaTaskChannelFailure {
			effective = append(effective, task)
		}
	}
	sort.SliceStable(effective, func(i, j int) bool { return effective[i].UpdatedAt > effective[j].UpdatedAt })

	stat := &ChannelMonitorModelStat{Model: modelName, LatestStatus: passiveMediaStatus(effective, isVideo)}
	if len(effective) == 0 {
		if includeTimeline {
			stat.Timeline = make([]*ChannelMonitorTimelinePoint, 0, channelMonitorPublicTimelineLimit)
			for index := 0; index < channelMonitorPublicTimelineLimit; index++ {
				stat.Timeline = append(stat.Timeline, &ChannelMonitorTimelinePoint{
					Status: model.ChannelMonitorStatusUnknown, CheckedAt: timelineStart + int64(index)*bucketSeconds, Carried: true,
				})
			}
		}
		return stat
	}
	checkedAt := effective[0].UpdatedAt
	stat.LatestChecked = &checkedAt
	buckets := make([][]*model.Task, channelMonitorPublicTimelineLimit)
	baseline := make([]*model.Task, 0)
	for _, task := range effective {
		if task.UpdatedAt < timelineStart {
			baseline = append(baseline, task)
			continue
		}
		index := int((task.UpdatedAt - timelineStart) / bucketSeconds)
		if index >= 0 && index < len(buckets) {
			buckets[index] = append(buckets[index], task)
		}
	}
	carriedStatus := passiveMediaStatus(baseline, isVideo)
	stat.Timeline = make([]*ChannelMonitorTimelinePoint, 0, channelMonitorPublicTimelineLimit)
	for index, bucket := range buckets {
		status := carriedStatus
		carried := true
		if len(bucket) > 0 {
			status = passiveMediaStatus(bucket, isVideo)
			carriedStatus = status
			carried = false
			stat.Observed++
			if status == model.ChannelMonitorStatusOperational {
				stat.Operational++
			}
		}
		stat.Timeline = append(stat.Timeline, &ChannelMonitorTimelinePoint{
			Status: status, CheckedAt: timelineStart + int64(index)*bucketSeconds, Carried: carried,
		})
	}
	stat.LatestStatus = stat.Timeline[len(stat.Timeline)-1].Status
	if stat.Observed > 0 {
		availability := float64(stat.Operational) * 100 / float64(stat.Observed)
		stat.Availability = &availability
	}
	if !includeTimeline {
		stat.Timeline = nil
	}
	return stat
}

func buildPassiveMediaStat(channel *model.Channel, modelName string, includeTimeline bool) (*ChannelMonitorModelStat, error) {
	isVideo := isVideoChannelMonitorTarget(channel.Type, modelName)
	freshness := channelMonitorImageFreshness
	if isVideo {
		freshness = channelMonitorVideoFreshness
	}
	since := time.Now().Add(-freshness - channelMonitorCarryLookback).Unix()
	tasks, err := model.ListRecentMediaTasks(channel.Id, modelName, since, channelMonitorPassiveQueryLimit)
	if err != nil {
		return nil, err
	}
	return buildPassiveMediaStatFromTasks(channel, modelName, tasks, includeTimeline), nil
}

func ChannelMonitorModels(monitor *model.ChannelMonitor) []string {
	if monitor == nil {
		return nil
	}
	models := []string{strings.TrimSpace(monitor.PrimaryModel)}
	models = append(models, DecodeChannelMonitorExtraModels(monitor.ExtraModelsJSON)...)
	return models
}

func channelMonitorModelsForChannel(monitor *model.ChannelMonitor, channel *model.Channel) []string {
	if monitor == nil || channel == nil || monitor.Scope == "" || monitor.Scope == model.ChannelMonitorScopeText {
		return ChannelMonitorModels(monitor)
	}
	disabled, err := disabledChannelMonitorModels()
	if err != nil {
		return nil
	}
	models := make([]string, 0)
	for _, modelName := range channel.GetModels() {
		modelName = strings.TrimSpace(modelName)
		if _, isDisabled := disabled[modelName]; isDisabled {
			continue
		}
		category := publicChannelMonitorCategory(channel.Type, modelName)
		if category != "" && channelMonitorAcceptsCategory(monitor, category) {
			models = append(models, modelName)
		}
	}
	return models
}

func RunChannelMonitor(ctx context.Context, monitorID int64, probe ChannelMonitorProbeFunc) ([]*model.ChannelMonitorResult, error) {
	monitor, err := model.GetChannelMonitorByID(monitorID)
	if err != nil {
		return nil, err
	}
	if monitor.Scope == model.ChannelMonitorScopeText {
		return runTextGroupChannelMonitor(ctx, monitor, probe)
	}
	if monitor.ChannelID <= 0 {
		return []*model.ChannelMonitorResult{}, nil
	}
	channel, err := model.GetChannelById(monitor.ChannelID, true)
	if err != nil {
		return nil, err
	}
	channel, err = resolveActiveTextMonitorChannel(monitor, channel)
	if err != nil {
		return nil, err
	}
	return runLegacyChannelMonitor(ctx, monitor, channel, probe)
}

func runTextGroupChannelMonitor(ctx context.Context, monitor *model.ChannelMonitor, probe ChannelMonitorProbeFunc) ([]*model.ChannelMonitorResult, error) {
	if probe == nil {
		return nil, errors.New("channel monitor probe is not configured")
	}
	channels, err := model.GetEnabledChannels(true)
	if err != nil {
		return nil, err
	}
	targetGroup := strings.TrimSpace(monitor.Target)
	if targetGroup == "" && monitor.ChannelID > 0 {
		legacyChannel, legacyErr := model.GetChannelById(monitor.ChannelID, false)
		if legacyErr != nil {
			return nil, legacyErr
		}
		groups := legacyChannel.GetGroups()
		if len(groups) > 0 {
			targetGroup = groups[0]
		}
	}
	results := make([]*model.ChannelMonitorResult, 0)
	roundCheckedAt := time.Now().Unix()
	for _, channel := range channels {
		if !channelSupportsMonitorGroup(channel, targetGroup) || !channelSupportsMonitorModel(channel, monitor.PrimaryModel) {
			continue
		}
		if err := ctx.Err(); err != nil {
			return results, err
		}
		outcome := probe(ctx, monitor, channel, monitor.PrimaryModel)
		if outcome.Status == "" {
			outcome.Status = model.ChannelMonitorStatusUnknown
		}
		if len(outcome.ErrorMessage) > 512 {
			outcome.ErrorMessage = outcome.ErrorMessage[:512]
		}
		result := &model.ChannelMonitorResult{
			MonitorID: monitor.ID, ChannelID: channel.Id, Model: monitor.PrimaryModel,
			Status: outcome.Status, LatencyMs: outcome.LatencyMs, HTTPStatus: outcome.HTTPStatus,
			ErrorCode: outcome.ErrorCode, ErrorMessage: outcome.ErrorMessage, CheckedAt: roundCheckedAt,
		}
		if err := model.CreateChannelMonitorResult(result); err != nil {
			return results, err
		}
		results = append(results, result)
		if outcome.Status == model.ChannelMonitorStatusOperational {
			break
		}
	}
	return results, nil
}

func channelSupportsMonitorGroup(channel *model.Channel, targetGroup string) bool {
	for _, group := range channel.GetGroups() {
		if group == targetGroup {
			return true
		}
	}
	return false
}

func runLegacyChannelMonitor(ctx context.Context, monitor *model.ChannelMonitor, channel *model.Channel, probe ChannelMonitorProbeFunc) ([]*model.ChannelMonitorResult, error) {
	models := channelMonitorModelsForChannel(monitor, channel)
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
			ChannelID:    channel.Id,
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

func resolveActiveTextMonitorChannel(monitor *model.ChannelMonitor, configured *model.Channel) (*model.Channel, error) {
	if monitor == nil || configured == nil || monitor.ProbeKind != model.ChannelMonitorProbeTextActive || configured.Status == common.ChannelStatusEnabled {
		return configured, nil
	}
	configuredGroups := make(map[string]struct{})
	for _, group := range configured.GetGroups() {
		configuredGroups[group] = struct{}{}
	}
	channels, err := model.GetEnabledChannels(true)
	if err != nil {
		return nil, err
	}
	for _, channel := range channels {
		if !channelSupportsMonitorModel(channel, monitor.PrimaryModel) {
			continue
		}
		for _, group := range channel.GetGroups() {
			if _, matches := configuredGroups[group]; matches {
				return channel, nil
			}
		}
	}
	return configured, nil
}

func channelSupportsMonitorModel(channel *model.Channel, modelName string) bool {
	wanted := ToPublicModelName(modelName)
	for _, candidate := range channel.GetModels() {
		if candidate == modelName || ToPublicModelName(candidate) == wanted {
			return true
		}
	}
	return false
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
	rounds := make(map[int64][]*model.ChannelMonitorResult)
	legacyResults := make([]*model.ChannelMonitorResult, 0)
	for _, result := range results {
		if result.Model != modelName {
			continue
		}
		if result.ChannelID <= 0 {
			legacyResults = append(legacyResults, result)
		} else {
			rounds[result.CheckedAt] = append(rounds[result.CheckedAt], result)
		}
		if result.LatencyMs != nil {
			totalLatency += int64(*result.LatencyMs)
			latencySamples++
		}
	}
	roundResults := make([]*model.ChannelMonitorResult, 0, len(rounds))
	if len(legacyResults) > 0 && len(rounds) == 0 {
		roundResults = append(roundResults, legacyResults...)
	}
	for checkedAt, channelResults := range rounds {
		successes := 0
		failures := 0
		for _, result := range channelResults {
			switch result.Status {
			case model.ChannelMonitorStatusOperational:
				successes++
			case model.ChannelMonitorStatusUnavailable:
				failures++
			default:
				if result.Status == model.ChannelMonitorStatusDegraded {
					successes++
					failures++
				}
			}
		}
		status := model.ChannelMonitorStatusUnknown
		switch {
		case successes > 0:
			status = model.ChannelMonitorStatusOperational
		case failures > 0:
			status = model.ChannelMonitorStatusUnavailable
		}
		roundResults = append(roundResults, &model.ChannelMonitorResult{Model: modelName, Status: status, CheckedAt: checkedAt})
	}
	sort.Slice(roundResults, func(i, j int) bool { return roundResults[i].CheckedAt < roundResults[j].CheckedAt })
	for _, result := range roundResults {
		checked := result.CheckedAt
		stat.LatestChecked = &checked
		stat.LatestStatus = result.Status
		if result.Status == model.ChannelMonitorStatusOperational || result.Status == model.ChannelMonitorStatusDegraded || result.Status == model.ChannelMonitorStatusUnavailable {
			stat.Observed++
			if result.Status == model.ChannelMonitorStatusOperational {
				stat.Operational++
			}
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
		if len(roundResults) > channelMonitorPublicTimelineLimit {
			roundResults = roundResults[len(roundResults)-channelMonitorPublicTimelineLimit:]
		}
		stat.Timeline = make([]*ChannelMonitorTimelinePoint, 0, len(roundResults))
		for _, result := range roundResults {
			stat.Timeline = append(stat.Timeline, &ChannelMonitorTimelinePoint{
				Status: result.Status, LatencyMs: result.LatencyMs, CheckedAt: result.CheckedAt,
			})
		}
	}
	return stat
}

func BuildChannelMonitorView(monitor *model.ChannelMonitor, windowDays int, includeTimeline bool) (*ChannelMonitorView, error) {
	var channel *model.Channel
	var err error
	if monitor.ChannelID > 0 {
		channel, err = model.GetChannelById(monitor.ChannelID, false)
		if err != nil {
			return nil, err
		}
	}
	windowDays = NormalizeChannelMonitorWindow(windowDays)
	results, err := model.ListChannelMonitorResults(monitor.ID, time.Now().Add(-time.Duration(windowDays)*24*time.Hour).Unix())
	if err != nil {
		return nil, err
	}
	view := &ChannelMonitorView{
		ID: monitor.ID, Name: monitor.Name,
		Provider:  "",
		ProbeKind: monitor.ProbeKind, Scope: monitor.Scope, Enabled: monitor.Enabled, Visible: monitor.Visible,
		IntervalSeconds: monitor.IntervalSeconds, PrimaryModel: monitor.PrimaryModel, WindowDays: windowDays,
	}
	view.Primary = buildChannelMonitorModelStat(monitor.PrimaryModel, results, includeTimeline)
	if channel != nil {
		view.Provider = constant.GetChannelTypeName(channel.Type)
	}
	if channel != nil && IsBillableMediaMonitorTarget(channel.Type, monitor.PrimaryModel) {
		view.Primary, err = buildPassiveMediaStat(channel, monitor.PrimaryModel, includeTimeline)
		if err != nil {
			return nil, err
		}
	}
	for _, modelName := range DecodeChannelMonitorExtraModels(monitor.ExtraModelsJSON) {
		stat := buildChannelMonitorModelStat(modelName, results, false)
		if channel != nil && IsBillableMediaMonitorTarget(channel.Type, modelName) {
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
	timeline            []*ChannelMonitorTimelinePoint
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

func channelMonitorMediaCacheKey(targets map[int][]string) string {
	channelIDs := make([]int, 0, len(targets))
	for channelID := range targets {
		channelIDs = append(channelIDs, channelID)
	}
	sort.Ints(channelIDs)
	var identity strings.Builder
	for _, channelID := range channelIDs {
		modelNames := append([]string(nil), targets[channelID]...)
		sort.Strings(modelNames)
		fmt.Fprintf(&identity, "%d:%s;", channelID, strings.Join(modelNames, ","))
	}
	return fmt.Sprintf("targets:%x", sha256.Sum256([]byte(identity.String())))
}

func mergeChannelMonitorMediaTasks(existing, fresh []*model.Task, cutoff int64) []*model.Task {
	byID := make(map[int64]*model.Task, len(existing)+len(fresh))
	for _, task := range append(existing, fresh...) {
		if task != nil && task.UpdatedAt >= cutoff {
			byID[task.ID] = task
		}
	}
	merged := make([]*model.Task, 0, len(byID))
	for _, task := range byID {
		merged = append(merged, task)
	}
	sort.SliceStable(merged, func(i, j int) bool {
		if merged[i].UpdatedAt != merged[j].UpdatedAt {
			return merged[i].UpdatedAt > merged[j].UpdatedAt
		}
		return merged[i].ID > merged[j].ID
	})
	return merged
}

func listCachedChannelMonitorMediaTasks(targets map[int][]string, cutoff int64) ([]*model.Task, error) {
	cacheKey := channelMonitorMediaCacheKey(targets)
	cache := getChannelMonitorMediaCache()
	result, err, _ := channelMonitorMediaFlight.Do(cacheKey, func() (any, error) {
		cached, found, cacheErr := cache.Get(cacheKey)
		if cacheErr != nil {
			common.SysError(fmt.Sprintf("channel monitor media cache get failed: %v", cacheErr))
			found = false
		}
		since := cutoff
		if found && cached.Cursor > since {
			since = cached.Cursor - 1
		}
		fresh, cursor, queryErr := model.ListRecentChannelsMediaTasks(targets, since, 200000)
		if queryErr != nil {
			return channelMonitorMediaCacheItem{}, queryErr
		}
		if cursor < cached.Cursor {
			cursor = cached.Cursor
		}
		item := channelMonitorMediaCacheItem{
			Cursor: cursor,
			Tasks:  mergeChannelMonitorMediaTasks(cached.Tasks, fresh, cutoff),
		}
		if cacheErr := cache.SetWithTTL(cacheKey, item, channelMonitorMediaCacheTTL); cacheErr != nil {
			common.SysError(fmt.Sprintf("channel monitor media cache set failed: %v", cacheErr))
		}
		return item, nil
	})
	if err != nil {
		return nil, err
	}
	return result.(channelMonitorMediaCacheItem).Tasks, nil
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
	aggregate.timeline = append(aggregate.timeline, stat.Timeline...)
	aggregate.observed += stat.Observed
	aggregate.operational += stat.Operational
	if stat.AverageLatency != nil {
		aggregate.latencyTotal += int64(*stat.AverageLatency)
		aggregate.latencyMeasurements++
	}
}

func buildPublicChannelMonitorTimeline(points []*ChannelMonitorTimelinePoint, now time.Time) []*PublicChannelMonitorTimelinePoint {
	bucketSeconds := int64(channelMonitorPublicBucketSize / time.Second)
	timelineEnd := ((now.Unix() / bucketSeconds) + 1) * bucketSeconds
	timelineStart := timelineEnd - int64(channelMonitorPublicTimelineLimit)*bucketSeconds
	type bucketState struct {
		status  string
		carried bool
		set     bool
	}
	buckets := make([]bucketState, channelMonitorPublicTimelineLimit)
	baselineStatus := model.ChannelMonitorStatusUnknown
	baselineCheckedAt := int64(0)
	for _, point := range points {
		if point == nil || point.CheckedAt >= timelineEnd {
			continue
		}
		if point.CheckedAt < timelineStart {
			if point.CheckedAt >= baselineCheckedAt {
				baselineStatus = point.Status
				baselineCheckedAt = point.CheckedAt
			}
			continue
		}
		index := int((point.CheckedAt - timelineStart) / bucketSeconds)
		bucket := &buckets[index]
		if !bucket.set || (bucket.carried && !point.Carried) ||
			(bucket.carried == point.Carried && publicStatusPriority(point.Status) > publicStatusPriority(bucket.status)) {
			bucket.status = point.Status
			bucket.carried = point.Carried
			bucket.set = true
		}
	}
	timeline := make([]*PublicChannelMonitorTimelinePoint, 0, channelMonitorPublicTimelineLimit)
	carriedStatus := baselineStatus
	for _, bucket := range buckets {
		if bucket.set {
			carriedStatus = bucket.status
			timeline = append(timeline, &PublicChannelMonitorTimelinePoint{Status: bucket.status, Carried: bucket.carried})
			continue
		}
		timeline = append(timeline, &PublicChannelMonitorTimelinePoint{Status: carriedStatus, Carried: true})
	}
	return timeline
}

func buildPublicTextChannelMonitorTimeline(points []*ChannelMonitorTimelinePoint) []*PublicChannelMonitorTimelinePoint {
	filtered := make([]*ChannelMonitorTimelinePoint, 0, len(points))
	for _, point := range points {
		if point != nil {
			filtered = append(filtered, point)
		}
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].CheckedAt < filtered[j].CheckedAt })
	if len(filtered) > channelMonitorPublicTimelineLimit {
		filtered = filtered[len(filtered)-channelMonitorPublicTimelineLimit:]
	}
	timeline := make([]*PublicChannelMonitorTimelinePoint, 0, channelMonitorPublicTimelineLimit)
	for len(timeline)+len(filtered) < channelMonitorPublicTimelineLimit {
		timeline = append(timeline, &PublicChannelMonitorTimelinePoint{
			Status:  model.ChannelMonitorStatusUnknown,
			Carried: true,
		})
	}
	for _, point := range filtered {
		timeline = append(timeline, &PublicChannelMonitorTimelinePoint{Status: point.Status})
	}
	return timeline
}

type publicChannelMonitorSource struct {
	channel     *model.Channel
	mediaModels []string
}

func buildPublicChannelMonitorItems(monitors []*model.ChannelMonitor, windowDays int, discoverTextGroups bool) ([]*PublicChannelMonitorItem, error) {
	aggregates := make(map[string]*publicChannelMonitorAggregate)
	sources := make([]*publicChannelMonitorSource, 0, len(monitors))
	mediaTargets := make(map[int][]string)
	textGroups := make(map[string]struct{})
	mediaScopes := make(map[string]bool)
	disabledModels, err := disabledChannelMonitorModels()
	if err != nil {
		return nil, err
	}
	if discoverTextGroups {
		enabledChannels, err := model.GetEnabledChannels(false)
		if err != nil {
			return nil, err
		}
		for _, channel := range enabledChannels {
			hasTextModel := false
			for _, modelName := range channel.GetModels() {
				if _, disabled := disabledModels[strings.TrimSpace(modelName)]; disabled {
					continue
				}
				if !IsBillableMediaMonitorTarget(channel.Type, modelName) {
					hasTextModel = true
					break
				}
			}
			if !hasTextModel {
				continue
			}
			for _, group := range channel.GetGroups() {
				textGroups[group] = struct{}{}
				mergePublicChannelMonitorStat(aggregates, group, ChannelMonitorCategoryText, &ChannelMonitorModelStat{
					LatestStatus: model.ChannelMonitorStatusUnknown,
				})
			}
		}
	}
	for _, monitor := range monitors {
		if monitor.Scope == model.ChannelMonitorScopeImage || monitor.Scope == model.ChannelMonitorScopeMedia {
			mediaScopes[ChannelMonitorCategoryImage] = true
		}
		if monitor.Scope == model.ChannelMonitorScopeVideo || monitor.Scope == model.ChannelMonitorScopeMedia {
			mediaScopes[ChannelMonitorCategoryVideo] = true
		}
		if monitor.Scope == "" && monitor.ProbeKind == model.ChannelMonitorProbeMediaPassive {
			mediaScopes[ChannelMonitorCategoryImage] = true
			mediaScopes[ChannelMonitorCategoryVideo] = true
		}
		if monitor.Scope == model.ChannelMonitorScopeText || (monitor.Scope == "" && monitor.ProbeKind == model.ChannelMonitorProbeTextActive) {
			view, viewErr := BuildChannelMonitorView(monitor, windowDays, true)
			if viewErr != nil {
				return nil, viewErr
			}
			groups := []string{strings.TrimSpace(monitor.Target)}
			if groups[0] == "" && monitor.ChannelID > 0 {
				legacyChannel, legacyErr := model.GetChannelById(monitor.ChannelID, false)
				if legacyErr == nil {
					groups = legacyChannel.GetGroups()
				}
			}
			for _, group := range groups {
				_, enabled := textGroups[group]
				if !discoverTextGroups || enabled {
					mergePublicChannelMonitorStat(aggregates, group, ChannelMonitorCategoryText, view.Primary)
				}
			}
		}
	}
	enabledChannels, err := model.GetEnabledChannels(false)
	if err != nil {
		return nil, err
	}
	for _, channel := range enabledChannels {
		mediaModels := make([]string, 0)
		for _, modelName := range channel.GetModels() {
			modelName = strings.TrimSpace(modelName)
			if _, disabled := disabledModels[modelName]; disabled {
				continue
			}
			if !IsBillableMediaMonitorTarget(channel.Type, modelName) {
				continue
			}
			category := publicChannelMonitorCategory(channel.Type, modelName)
			if category == "" {
				continue
			}
			if !mediaScopes[category] {
				continue
			}
			mediaModels = append(mediaModels, modelName)
		}
		sources = append(sources, &publicChannelMonitorSource{channel: channel, mediaModels: mediaModels})
		if len(mediaModels) > 0 {
			for _, modelName := range mediaModels {
				mediaTargets[channel.Id] = append(mediaTargets[channel.Id], channelMonitorModelAliases(modelName)...)
			}
		}
	}

	mediaTasks := make([]*model.Task, 0)
	if len(mediaTargets) > 0 {
		mediaTasks, err = listCachedChannelMonitorMediaTasks(
			mediaTargets,
			time.Now().Add(-channelMonitorVideoFreshness-channelMonitorCarryLookback).Unix(),
		)
		if err != nil {
			return nil, err
		}
	}
	mediaTasksByChannel := make(map[int][]*model.Task)
	for _, task := range mediaTasks {
		mediaTasksByChannel[task.ChannelId] = append(mediaTasksByChannel[task.ChannelId], task)
	}

	for _, source := range sources {
		for _, modelName := range source.mediaModels {
			category := publicChannelMonitorCategory(source.channel.Type, modelName)
			stat := buildPassiveMediaStatFromTasks(source.channel, modelName, mediaTasksByChannel[source.channel.Id], true)
			mergePublicChannelMonitorStat(
				aggregates,
				ToPublicModelName(modelName),
				category,
				stat,
			)
		}
	}

	now := time.Now()
	items := make([]*PublicChannelMonitorItem, 0, len(aggregates))
	for _, aggregate := range aggregates {
		if aggregate.item.Category == ChannelMonitorCategoryText {
			aggregate.item.Timeline = buildPublicTextChannelMonitorTimeline(aggregate.timeline)
		} else {
			aggregate.item.Timeline = buildPublicChannelMonitorTimeline(aggregate.timeline, now)
		}
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
	windowDays = NormalizeChannelMonitorWindow(windowDays)
	cacheKey := fmt.Sprintf("window:%d", windowDays)
	cache := getPublicChannelMonitorCache()
	if item, found, cacheErr := cache.Get(cacheKey); cacheErr == nil && found {
		return item.Items, item.Summary, nil
	} else if cacheErr != nil {
		common.SysError(fmt.Sprintf("public channel monitor cache get failed: %v", cacheErr))
	}
	return []*PublicChannelMonitorItem{}, &PublicChannelMonitorSummary{Enabled: IsChannelMonitorEnabled()}, nil
}

func listPublicChannelMonitorViewsUncached(windowDays int) ([]*PublicChannelMonitorItem, *PublicChannelMonitorSummary, error) {
	monitors, err := model.ListChannelMonitors(true, true)
	if err != nil {
		return nil, nil, err
	}
	items, err := buildPublicChannelMonitorItems(monitors, windowDays, true)
	if err != nil {
		return nil, nil, err
	}
	return items, summarizePublicChannelMonitorItems(items), nil
}

func listPublicTextChannelMonitorViewsUncached(windowDays int) ([]*PublicChannelMonitorItem, error) {
	monitors, err := model.ListChannelMonitors(true, true)
	if err != nil {
		return nil, err
	}
	textMonitors := make([]*model.ChannelMonitor, 0, len(monitors))
	for _, monitor := range monitors {
		if monitor.Scope == model.ChannelMonitorScopeText ||
			(monitor.Scope == "" && monitor.ProbeKind == model.ChannelMonitorProbeTextActive) {
			textMonitors = append(textMonitors, monitor)
		}
	}
	return buildPublicChannelMonitorItems(textMonitors, windowDays, true)
}

func BuildPublicChannelMonitorViews(monitor *model.ChannelMonitor, windowDays int) ([]*PublicChannelMonitorItem, error) {
	if monitor == nil {
		return nil, errors.New("channel monitor is required")
	}
	return buildPublicChannelMonitorItems([]*model.ChannelMonitor{monitor}, windowDays, false)
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
		channelName := ""
		group := monitor.Target
		if monitor.ChannelID > 0 {
			channel, channelErr := model.GetChannelById(monitor.ChannelID, false)
			if channelErr != nil && !errors.Is(channelErr, gorm.ErrRecordNotFound) {
				return nil, nil, channelErr
			}
			if channel != nil {
				channelName = channel.Name
				if group == "" {
					group = channel.Group
				}
			}
		}
		views = append(views, &AdminChannelMonitorView{
			ChannelMonitorView: view,
			ChannelID:          monitor.ChannelID,
			ChannelName:        channelName,
			Group:              group,
			Target:             monitor.Target,
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

func ListChannelMonitorTextTargets() ([]*ChannelMonitorTextTarget, error) {
	channels, err := model.GetEnabledChannels(false)
	if err != nil {
		return nil, err
	}
	disabled, err := disabledChannelMonitorModels()
	if err != nil {
		return nil, err
	}
	modelsByGroup := make(map[string]map[string]struct{})
	for _, channel := range channels {
		for _, modelName := range channel.GetModels() {
			modelName = strings.TrimSpace(modelName)
			if _, isDisabled := disabled[modelName]; isDisabled || IsBillableMediaMonitorTarget(channel.Type, modelName) {
				continue
			}
			for _, group := range channel.GetGroups() {
				if modelsByGroup[group] == nil {
					modelsByGroup[group] = make(map[string]struct{})
				}
				modelsByGroup[group][modelName] = struct{}{}
			}
		}
	}
	targets := make([]*ChannelMonitorTextTarget, 0, len(modelsByGroup))
	for group, modelSet := range modelsByGroup {
		models := make([]string, 0, len(modelSet))
		for modelName := range modelSet {
			models = append(models, modelName)
		}
		sort.Strings(models)
		targets = append(targets, &ChannelMonitorTextTarget{Group: group, Models: models})
	}
	sort.Slice(targets, func(i, j int) bool { return targets[i].Group < targets[j].Group })
	return targets, nil
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
		go RefreshPublicChannelMonitorSnapshots()
		go func() {
			ticker := time.NewTicker(15 * time.Second)
			textRefreshTicker := time.NewTicker(channelMonitorPublicTextRefresh)
			publicRefreshTicker := time.NewTicker(channelMonitorPublicRefresh)
			cleanupTicker := time.NewTicker(6 * time.Hour)
			defer ticker.Stop()
			defer textRefreshTicker.Stop()
			defer publicRefreshTicker.Stop()
			defer cleanupTicker.Stop()
			runDueChannelMonitors(probe)
			for {
				select {
				case <-ticker.C:
					runDueChannelMonitors(probe)
				case <-textRefreshTicker.C:
					RefreshPublicTextChannelMonitorSnapshots()
				case <-publicRefreshTicker.C:
					RefreshPublicChannelMonitorSnapshots()
				case <-cleanupTicker.C:
					_, _ = model.DeleteChannelMonitorResultsBefore(time.Now().Add(-channelMonitorHistoryDays * 24 * time.Hour).Unix())
				}
			}
		}()
	})
}

func RefreshPublicTextChannelMonitorSnapshots() {
	if !IsChannelMonitorEnabled() {
		return
	}
	_, _, _ = publicChannelMonitorFlight.Do("refresh", func() (any, error) {
		for _, windowDays := range []int{7, 15, 30} {
			cacheKey := fmt.Sprintf("window:%d", windowDays)
			cached, found, err := getPublicChannelMonitorCache().Get(cacheKey)
			if err != nil || !found {
				continue
			}
			textItems, err := listPublicTextChannelMonitorViewsUncached(windowDays)
			if err != nil {
				common.SysLog(fmt.Sprintf("channel monitor: refresh public text snapshot for %d days failed: %v", windowDays, err))
				continue
			}
			items := make([]*PublicChannelMonitorItem, 0, len(cached.Items)+len(textItems))
			for _, item := range cached.Items {
				if item.Category != ChannelMonitorCategoryText {
					items = append(items, item)
				}
			}
			items = append(items, textItems...)
			sort.Slice(items, func(i, j int) bool {
				if items[i].Category != items[j].Category {
					return items[i].Category < items[j].Category
				}
				return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
			})
			updated := publicChannelMonitorCacheItem{Items: items, Summary: summarizePublicChannelMonitorItems(items)}
			if err := getPublicChannelMonitorCache().SetWithTTL(cacheKey, updated, channelMonitorPublicCacheTTL); err != nil {
				common.SysLog(fmt.Sprintf("channel monitor: store public text snapshot for %d days failed: %v", windowDays, err))
			}
		}
		return nil, nil
	})
}

func RefreshPublicChannelMonitorSnapshots() {
	if !IsChannelMonitorEnabled() {
		return
	}
	_, _, _ = publicChannelMonitorFlight.Do("refresh", func() (any, error) {
		for _, windowDays := range []int{7, 15, 30} {
			items, summary, err := listPublicChannelMonitorViewsUncached(windowDays)
			if err != nil {
				common.SysLog(fmt.Sprintf("channel monitor: refresh public snapshot for %d days failed: %v", windowDays, err))
				continue
			}
			cacheKey := fmt.Sprintf("window:%d", windowDays)
			item := publicChannelMonitorCacheItem{Items: items, Summary: summary}
			if err := getPublicChannelMonitorCache().SetWithTTL(cacheKey, item, channelMonitorPublicCacheTTL); err != nil {
				common.SysLog(fmt.Sprintf("channel monitor: store public snapshot for %d days failed: %v", windowDays, err))
			}
		}
		return nil, nil
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
