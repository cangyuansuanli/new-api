package service

import (
	"context"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupChannelMonitorTest(t *testing.T) {
	t.Helper()
	InvalidatePublicChannelMonitorCache()
	require.NoError(t, getChannelMonitorMediaCache().Purge())
	require.NoError(t, model.DB.AutoMigrate(
		&model.Model{},
		&model.ChannelMonitor{},
		&model.ChannelMonitorResult{},
		&model.Task{},
	))
	require.NoError(t, model.DB.Exec("DELETE FROM tasks").Error)
	require.NoError(t, model.DB.Exec("DELETE FROM channel_monitor_results").Error)
	require.NoError(t, model.DB.Exec("DELETE FROM channel_monitors").Error)
	require.NoError(t, model.DB.Exec("DELETE FROM channels").Error)
	t.Cleanup(func() {
		model.DB.Exec("DELETE FROM tasks")
		model.DB.Exec("DELETE FROM channel_monitor_results")
		model.DB.Exec("DELETE FROM channel_monitors")
		model.DB.Exec("DELETE FROM channels")
	})
}

func createMonitorFixture(t *testing.T, channelType int, modelName string) *model.ChannelMonitor {
	t.Helper()
	channel := &model.Channel{
		Name:   "monitor-fixture",
		Type:   channelType,
		Key:    "sensitive-test-key",
		Status: common.ChannelStatusEnabled,
		Models: modelName,
		Group:  "default",
	}
	require.NoError(t, model.DB.Create(channel).Error)
	monitor := &model.ChannelMonitor{
		ChannelID:       channel.Id,
		Name:            "Public monitor",
		PrimaryModel:    modelName,
		ExtraModelsJSON: "[]",
		ProbeKind:       ResolveChannelMonitorProbeKind(channelType, modelName, nil),
		IntervalSeconds: 300,
		JitterSeconds:   0,
		Enabled:         true,
		Visible:         true,
	}
	require.NoError(t, model.CreateChannelMonitor(monitor))
	return monitor
}

func TestRunChannelMonitorNeverActivelyProbesBillableMedia(t *testing.T) {
	setupChannelMonitorTest(t)
	monitor := createMonitorFixture(t, constant.ChannelTypeOpenAI, "gpt-image-1")
	probeCalls := 0

	results, err := RunChannelMonitor(context.Background(), monitor.ID, func(
		context.Context,
		*model.ChannelMonitor,
		*model.Channel,
		string,
	) ChannelMonitorProbeOutcome {
		probeCalls++
		return ChannelMonitorProbeOutcome{Status: model.ChannelMonitorStatusOperational}
	})

	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, 0, probeCalls)
	assert.Equal(t, model.ChannelMonitorStatusUnknown, results[0].Status)
	assert.Equal(t, "passive_no_recent_sample", results[0].ErrorCode)
	stored, err := model.ListChannelMonitorResults(monitor.ID, 0)
	require.NoError(t, err)
	require.Len(t, stored, 1)
	assert.Equal(t, model.ChannelMonitorStatusUnknown, stored[0].Status)
}

func TestBillableMediaTargetsAreNeverActivelyProbed(t *testing.T) {
	modelCases := []string{
		"imagen-4-ultra",
		"adobe-firefly-gpt-image-2-1k",
		"stable-diffusion-3.5-large",
		"sdxl-1.0",
		"gemini-banana-pro-4k",
		"cy-sd1-omni-fast-no-water",
		"cy-sd4-minimax-h3-2k",
	}
	for _, modelName := range modelCases {
		t.Run(modelName, func(t *testing.T) {
			setupChannelMonitorTest(t)
			monitor := createMonitorFixture(t, constant.ChannelTypeOpenAI, modelName)
			assert.Equal(t, model.ChannelMonitorProbeMediaPassive, monitor.ProbeKind)

			probeCalls := 0
			results, err := RunChannelMonitor(context.Background(), monitor.ID, func(
				context.Context,
				*model.ChannelMonitor,
				*model.Channel,
				string,
			) ChannelMonitorProbeOutcome {
				probeCalls++
				return ChannelMonitorProbeOutcome{Status: model.ChannelMonitorStatusOperational}
			})

			require.NoError(t, err)
			require.Len(t, results, 1)
			assert.Zero(t, probeCalls)
			assert.Equal(t, model.ChannelMonitorStatusUnknown, results[0].Status)
		})
	}

	channelCases := []int{
		constant.ChannelTypeMidjourney,
		constant.ChannelTypeSunoAPI,
		constant.ChannelTypeKling,
		constant.ChannelTypeJimeng,
		constant.ChannelTypeDoubaoVideo,
		constant.ChannelTypeVidu,
	}
	for _, channelType := range channelCases {
		t.Run(constant.GetChannelTypeName(channelType), func(t *testing.T) {
			assert.True(t, IsBillableMediaMonitorTarget(channelType, "text-looking-model"))
			assert.Equal(t, model.ChannelMonitorProbeMediaPassive, ResolveChannelMonitorProbeKind(channelType, "text-looking-model", nil))
		})
	}
}

func TestOmniMediaModelsUseVideoFreshness(t *testing.T) {
	assert.True(t, IsBillableMediaMonitorTarget(constant.ChannelTypeOpenAI, "cy-sd1-omni-fast"))
	assert.True(t, isVideoChannelMonitorTarget(constant.ChannelTypeOpenAI, "cy-sd1-omni-v2v-no-water"))
}

func TestMinimaxMediaModelsUseVideoFreshness(t *testing.T) {
	assert.True(t, IsBillableMediaMonitorTarget(constant.ChannelTypeMidjourneyPlus, "cy-sd4-minimax-h3-2k"))
	assert.True(t, isVideoChannelMonitorTarget(constant.ChannelTypeMidjourneyPlus, "cy-sd4-minimax-h3-2k"))
}

func TestMediaMonitorViewIsUnknownBeforeFirstObservation(t *testing.T) {
	setupChannelMonitorTest(t)
	monitor := createMonitorFixture(t, constant.ChannelTypeOpenAI, "gpt-image-1")

	view, err := BuildChannelMonitorView(monitor, 7, true)
	require.NoError(t, err)
	assert.Equal(t, model.ChannelMonitorStatusUnknown, view.Primary.LatestStatus)
	assert.Nil(t, view.Primary.Availability)
	assert.Zero(t, view.Primary.Observed)
	require.Len(t, view.Primary.Timeline, channelMonitorPublicTimelineLimit)
	for _, point := range view.Primary.Timeline {
		assert.Equal(t, model.ChannelMonitorStatusUnknown, point.Status)
		assert.True(t, point.Carried)
	}
}

func TestRunChannelMonitorPersistsTextProbeResult(t *testing.T) {
	setupChannelMonitorTest(t)
	monitor := createMonitorFixture(t, constant.ChannelTypeOpenAI, "gpt-4o-mini")
	latency := 137

	results, err := RunChannelMonitor(context.Background(), monitor.ID, func(
		_ context.Context,
		_ *model.ChannelMonitor,
		channel *model.Channel,
		modelName string,
	) ChannelMonitorProbeOutcome {
		assert.Equal(t, "sensitive-test-key", channel.Key)
		assert.Equal(t, "gpt-4o-mini", modelName)
		return ChannelMonitorProbeOutcome{
			Status:     model.ChannelMonitorStatusOperational,
			LatencyMs:  &latency,
			HTTPStatus: 200,
		}
	})

	require.NoError(t, err)
	require.Len(t, results, 1)
	stored, err := model.ListChannelMonitorResults(monitor.ID, 0)
	require.NoError(t, err)
	require.Len(t, stored, 1)
	assert.Equal(t, model.ChannelMonitorStatusOperational, stored[0].Status)
	assert.Equal(t, 137, *stored[0].LatencyMs)
	assert.Equal(t, 200, stored[0].HTTPStatus)
}

func TestRunChannelMonitorUsesEnabledReplacementInSameGroup(t *testing.T) {
	setupChannelMonitorTest(t)
	monitor := createMonitorFixture(t, constant.ChannelTypeOpenAI, "gpt-5.6-sol")
	configured, err := model.GetChannelById(monitor.ChannelID, false)
	require.NoError(t, err)
	configured.Group = "LLM-GPT-pro"
	configured.Status = common.ChannelStatusManuallyDisabled
	require.NoError(t, model.DB.Save(configured).Error)
	replacement := &model.Channel{
		Name: "replacement", Type: constant.ChannelTypeOpenAI, Key: "replacement-key",
		Status: common.ChannelStatusEnabled, Models: "gpt-5.6-sol", Group: "LLM-GPT-pro",
	}
	require.NoError(t, model.DB.Create(replacement).Error)

	results, err := RunChannelMonitor(context.Background(), monitor.ID, func(
		_ context.Context,
		_ *model.ChannelMonitor,
		channel *model.Channel,
		_ string,
	) ChannelMonitorProbeOutcome {
		assert.Equal(t, replacement.Id, channel.Id)
		assert.Equal(t, "replacement-key", channel.Key)
		return ChannelMonitorProbeOutcome{Status: model.ChannelMonitorStatusOperational}
	})

	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, model.ChannelMonitorStatusOperational, results[0].Status)
}

func TestTextGroupMonitorStopsAfterFirstOperationalChannel(t *testing.T) {
	setupChannelMonitorTest(t)
	channels := []*model.Channel{
		{Name: "first", Type: constant.ChannelTypeOpenAI, Key: "first-key", Status: common.ChannelStatusEnabled, Models: "gpt-5.6-sol", Group: "LLM-GPT-pro"},
		{Name: "second", Type: constant.ChannelTypeOpenAI, Key: "second-key", Status: common.ChannelStatusEnabled, Models: "gpt-5.6-sol", Group: "LLM-GPT-pro"},
		{Name: "third", Type: constant.ChannelTypeOpenAI, Key: "third-key", Status: common.ChannelStatusEnabled, Models: "gpt-5.6-sol", Group: "LLM-GPT-pro"},
	}
	for _, channel := range channels {
		require.NoError(t, model.DB.Create(channel).Error)
	}
	monitor := &model.ChannelMonitor{
		Scope: model.ChannelMonitorScopeText, Target: "LLM-GPT-pro", Name: "GPT pro",
		PrimaryModel: "gpt-5.6-sol", ExtraModelsJSON: "[]", ProbeKind: model.ChannelMonitorProbeTextActive,
		IntervalSeconds: 300, Enabled: true, Visible: true,
	}
	require.NoError(t, model.CreateChannelMonitor(monitor))
	probeCalls := 0

	results, err := RunChannelMonitor(context.Background(), monitor.ID, func(
		_ context.Context,
		_ *model.ChannelMonitor,
		_ *model.Channel,
		_ string,
	) ChannelMonitorProbeOutcome {
		probeCalls++
		if probeCalls == 1 {
			return ChannelMonitorProbeOutcome{Status: model.ChannelMonitorStatusUnavailable}
		}
		return ChannelMonitorProbeOutcome{Status: model.ChannelMonitorStatusOperational}
	})

	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, 2, probeCalls)
	assert.Equal(t, model.ChannelMonitorStatusUnavailable, results[0].Status)
	assert.Equal(t, model.ChannelMonitorStatusOperational, results[1].Status)
	view, err := BuildChannelMonitorView(monitor, 7, true)
	require.NoError(t, err)
	assert.Equal(t, model.ChannelMonitorStatusOperational, view.Primary.LatestStatus)
	assert.Equal(t, 1, view.Primary.Observed)
}

func TestMediaAggregationExcludesUnlistedModelsAcrossEnabledChannels(t *testing.T) {
	setupChannelMonitorTest(t)
	enabled := &model.Channel{
		Name: "enabled-media", Type: constant.ChannelTypeOpenAI, Key: "enabled-key",
		Status: common.ChannelStatusEnabled,
		Models: "adobe-firefly-nano-banana-1k,adobe-firefly-nano-banana2-2k", Group: "default",
	}
	require.NoError(t, model.DB.Create(enabled).Error)
	disabledModel := &model.Model{
		ModelName: "adobe-firefly-nano-banana-1k", NameRule: model.NameRuleExact, Status: 0,
	}
	require.NoError(t, disabledModel.Insert())
	monitor := &model.ChannelMonitor{
		Scope: model.ChannelMonitorScopeImage, Name: "Images", ProbeKind: model.ChannelMonitorProbeMediaPassive,
		ExtraModelsJSON: "[]", IntervalSeconds: 1800, Enabled: true, Visible: true,
	}
	require.NoError(t, model.CreateChannelMonitor(monitor))

	items, err := buildPublicChannelMonitorItems([]*model.ChannelMonitor{monitor}, 7, true)

	require.NoError(t, err)
	byName := make(map[string]*PublicChannelMonitorItem)
	for _, item := range items {
		byName[item.Name] = item
	}
	assert.NotContains(t, byName, "adobe-firefly-nano-banana-1k")
	assert.Contains(t, byName, "adobe-firefly-nano-banana2-2k")
}

func TestPublicMonitorCacheMissReturnsWithoutAggregation(t *testing.T) {
	setupChannelMonitorTest(t)
	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	previousEnabled := common.OptionMap["ChannelMonitorEnabled"]
	common.OptionMap["ChannelMonitorEnabled"] = "true"
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap["ChannelMonitorEnabled"] = previousEnabled
		common.OptionMapRWMutex.Unlock()
	})
	require.NoError(t, getPublicChannelMonitorCache().Purge())

	items, summary, err := ListPublicChannelMonitorViews(7)

	require.NoError(t, err)
	assert.Empty(t, items)
	assert.True(t, summary.Enabled)
}

func TestRunChannelMonitorProbesTextAndSkipsMediaInMixedMonitor(t *testing.T) {
	setupChannelMonitorTest(t)
	monitor := createMonitorFixture(t, constant.ChannelTypeOpenAI, "gpt-4o-mini")
	extraModels, err := EncodeChannelMonitorExtraModels([]string{"gpt-image-1"})
	require.NoError(t, err)
	monitor.ExtraModelsJSON = extraModels
	monitor.ProbeKind = ResolveChannelMonitorProbeKind(constant.ChannelTypeOpenAI, monitor.PrimaryModel, []string{"gpt-image-1"})
	require.NoError(t, model.UpdateChannelMonitor(monitor))
	probeCalls := 0

	results, err := RunChannelMonitor(context.Background(), monitor.ID, func(
		_ context.Context,
		_ *model.ChannelMonitor,
		_ *model.Channel,
		modelName string,
	) ChannelMonitorProbeOutcome {
		probeCalls++
		assert.Equal(t, "gpt-4o-mini", modelName)
		return ChannelMonitorProbeOutcome{Status: model.ChannelMonitorStatusOperational}
	})

	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, 1, probeCalls)
	assert.Equal(t, model.ChannelMonitorStatusOperational, results[0].Status)
	assert.Equal(t, model.ChannelMonitorStatusUnknown, results[1].Status)
}

func createMediaTaskFixture(t *testing.T, monitor *model.ChannelMonitor, status model.TaskStatus, reason string, updatedAt int64) {
	t.Helper()
	require.NoError(t, model.DB.Create(&model.Task{
		CreatedAt:  updatedAt,
		UpdatedAt:  updatedAt,
		TaskID:     model.GenerateTaskID(),
		Platform:   constant.TaskPlatformImage,
		ChannelId:  monitor.ChannelID,
		Status:     status,
		FailReason: reason,
		Properties: model.Properties{ClientModelName: monitor.PrimaryModel},
	}).Error)
}

func TestRecentMediaTasksMatchAllPersistedModelNames(t *testing.T) {
	setupChannelMonitorTest(t)
	monitor := createMonitorFixture(t, constant.ChannelTypeOpenAI, "seedance-2.0")
	now := time.Now().Unix()
	properties := []model.Properties{
		{ClientModelName: monitor.PrimaryModel},
		{OriginModelName: monitor.PrimaryModel},
		{UpstreamModelName: monitor.PrimaryModel},
	}
	for index, taskProperties := range properties {
		require.NoError(t, model.DB.Create(&model.Task{
			CreatedAt:  now,
			UpdatedAt:  now - int64(index),
			TaskID:     model.GenerateTaskID(),
			Platform:   constant.TaskPlatformImage,
			ChannelId:  monitor.ChannelID,
			Status:     model.TaskStatusSuccess,
			Properties: taskProperties,
		}).Error)
	}
	require.NoError(t, model.DB.Create(&model.Task{
		CreatedAt: now, UpdatedAt: now, TaskID: model.GenerateTaskID(),
		Platform: constant.TaskPlatformImage, ChannelId: monitor.ChannelID + 1,
		Status:     model.TaskStatusSuccess,
		Properties: model.Properties{ClientModelName: monitor.PrimaryModel},
	}).Error)

	tasks, err := model.ListRecentMediaTasks(monitor.ChannelID, monitor.PrimaryModel, now-60, 100)

	require.NoError(t, err)
	assert.Len(t, tasks, 3)
}

func TestPassiveVideoAggregatesCurrentBucketObservations(t *testing.T) {
	setupChannelMonitorTest(t)
	monitor := createMonitorFixture(t, constant.ChannelTypeOpenAI, "seedance-2.0")
	bucketStart := time.Now().Truncate(channelMonitorPublicBucketSize)
	observationTime := bucketStart.Add(-5 * time.Minute)
	createMediaTaskFixture(t, monitor, model.TaskStatusSuccess, "", observationTime.Unix())
	createMediaTaskFixture(t, monitor, model.TaskStatusFailure, "upstream returned failed with no output and no failure detail", observationTime.Add(-time.Minute).Unix())
	createMediaTaskFixture(t, monitor, model.TaskStatusFailure, "bad response status 504", observationTime.Add(-2*time.Minute).Unix())

	view, err := BuildChannelMonitorView(monitor, 7, true)

	require.NoError(t, err)
	assert.Equal(t, model.ChannelMonitorProbeMediaPassive, view.ProbeKind)
	assert.Equal(t, model.ChannelMonitorStatusDegraded, view.Primary.LatestStatus)
	assert.Equal(t, 1, view.Primary.Observed)
	assert.Zero(t, view.Primary.Operational)
	require.NotNil(t, view.Primary.Availability)
	assert.InDelta(t, 0, *view.Primary.Availability, 0.001)
	require.Len(t, view.Primary.Timeline, channelMonitorPublicTimelineLimit)
	assert.Equal(t, model.ChannelMonitorStatusDegraded, view.Primary.Timeline[len(view.Primary.Timeline)-1].Status)
}

func TestPassiveVideoThreeRecentChannelFailuresAreUnavailable(t *testing.T) {
	setupChannelMonitorTest(t)
	monitor := createMonitorFixture(t, constant.ChannelTypeOpenAI, "seedance-2.0")
	now := time.Now()
	for index := 0; index < 3; index++ {
		createMediaTaskFixture(t, monitor, model.TaskStatusFailure, "upstream request failed: unexpected EOF", now.Add(-time.Duration(index)*time.Minute).Unix())
	}
	createMediaTaskFixture(t, monitor, model.TaskStatusSuccess, "", now.Add(-4*time.Minute).Unix())

	view, err := BuildChannelMonitorView(monitor, 7, true)

	require.NoError(t, err)
	assert.Equal(t, model.ChannelMonitorStatusUnavailable, view.Primary.LatestStatus)
}

func TestPassiveImageUsesRecentFiveEffectiveObservations(t *testing.T) {
	setupChannelMonitorTest(t)
	monitor := createMonitorFixture(t, constant.ChannelTypeOpenAI, "gpt-image-2")
	bucketStart := time.Now().Truncate(channelMonitorPublicBucketSize)
	observationTime := bucketStart.Add(-5 * time.Minute)
	statuses := []model.TaskStatus{
		model.TaskStatusSuccess,
		model.TaskStatusSuccess,
		model.TaskStatusSuccess,
		model.TaskStatusFailure,
		model.TaskStatusFailure,
	}
	for index, status := range statuses {
		reason := ""
		if status == model.TaskStatusFailure {
			reason = "bad response status code 502"
		}
		createMediaTaskFixture(t, monitor, status, reason, observationTime.Add(-time.Duration(index)*time.Minute).Unix())
	}

	view, err := BuildChannelMonitorView(monitor, 7, false)

	require.NoError(t, err)
	assert.Equal(t, model.ChannelMonitorStatusDegraded, view.Primary.LatestStatus)
	require.NotNil(t, view.Primary.Availability)
	assert.InDelta(t, 0, *view.Primary.Availability, 0.001)
}

func TestClassifyMediaTaskSeparatesFailureOwnership(t *testing.T) {
	cases := []struct {
		name   string
		reason string
		want   mediaTaskFailureClass
	}{
		{name: "moderation", reason: "The generated images appear to be unsafe", want: mediaTaskExcluded},
		{name: "invalid input", reason: "prompt or reference image is required", want: mediaTaskExcluded},
		{name: "platform", reason: "R2 result upload timeout", want: mediaTaskPlatform},
		{name: "configuration", reason: "unsupported endpoint for model", want: mediaTaskConfiguration},
		{name: "upstream", reason: "bad response status code 504", want: mediaTaskChannelFailure},
		{name: "credential", reason: "adobe http 403 unauthorized", want: mediaTaskChannelFailure},
		{name: "unknown", reason: "upstream returned failed with no output and no failure detail", want: mediaTaskUnknown},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			task := &model.Task{Status: model.TaskStatusFailure, FailReason: testCase.reason}
			assert.Equal(t, testCase.want, classifyMediaTask(task))
		})
	}
}

func TestBuildChannelMonitorViewAggregatesSelectedWindow(t *testing.T) {
	setupChannelMonitorTest(t)
	monitor := createMonitorFixture(t, constant.ChannelTypeOpenAI, "gpt-4o-mini")
	now := time.Now()
	latencyFast, latencySlow := 100, 300
	fixtures := []*model.ChannelMonitorResult{
		{MonitorID: monitor.ID, Model: monitor.PrimaryModel, Status: model.ChannelMonitorStatusOperational, LatencyMs: &latencyFast, CheckedAt: now.Add(-time.Hour).Unix()},
		{MonitorID: monitor.ID, Model: monitor.PrimaryModel, Status: model.ChannelMonitorStatusDegraded, LatencyMs: &latencySlow, CheckedAt: now.Add(-2 * time.Hour).Unix()},
		{MonitorID: monitor.ID, Model: monitor.PrimaryModel, Status: model.ChannelMonitorStatusOperational, LatencyMs: &latencyFast, CheckedAt: now.Add(-8 * 24 * time.Hour).Unix()},
	}
	for _, result := range fixtures {
		require.NoError(t, model.CreateChannelMonitorResult(result))
	}

	sevenDay, err := BuildChannelMonitorView(monitor, 7, true)
	require.NoError(t, err)
	require.NotNil(t, sevenDay.Primary.Availability)
	assert.InDelta(t, 50, *sevenDay.Primary.Availability, 0.001)
	assert.Equal(t, 2, sevenDay.Primary.Observed)
	assert.Equal(t, 200, *sevenDay.Primary.AverageLatency)
	assert.Len(t, sevenDay.Primary.Timeline, 2)

	fifteenDay, err := BuildChannelMonitorView(monitor, 15, false)
	require.NoError(t, err)
	require.NotNil(t, fifteenDay.Primary.Availability)
	assert.InDelta(t, 66.666, *fifteenDay.Primary.Availability, 0.01)
	assert.Equal(t, 3, fifteenDay.Primary.Observed)
}

func TestBuildChannelMonitorViewCountsUnavailableAsObserved(t *testing.T) {
	setupChannelMonitorTest(t)
	monitor := createMonitorFixture(t, constant.ChannelTypeOpenAI, "gpt-4o-mini")
	now := time.Now().Unix()
	require.NoError(t, model.CreateChannelMonitorResult(&model.ChannelMonitorResult{
		MonitorID: monitor.ID, Model: monitor.PrimaryModel,
		Status: model.ChannelMonitorStatusOperational, CheckedAt: now - 1,
	}))
	require.NoError(t, model.CreateChannelMonitorResult(&model.ChannelMonitorResult{
		MonitorID: monitor.ID, Model: monitor.PrimaryModel,
		Status: model.ChannelMonitorStatusUnavailable, CheckedAt: now,
	}))

	view, err := BuildChannelMonitorView(monitor, 7, false)

	require.NoError(t, err)
	assert.Equal(t, model.ChannelMonitorStatusUnavailable, view.Primary.LatestStatus)
	assert.Equal(t, 2, view.Primary.Observed)
	assert.Equal(t, 1, view.Primary.Operational)
	require.NotNil(t, view.Primary.Availability)
	assert.InDelta(t, 50, *view.Primary.Availability, 0.001)
}

func TestChannelMonitorPublicViewContainsNoCredentialsOrRawErrors(t *testing.T) {
	setupChannelMonitorTest(t)
	monitor := createMonitorFixture(t, constant.ChannelTypeOpenAI, "gpt-4o-mini")
	require.NoError(t, model.CreateChannelMonitorResult(&model.ChannelMonitorResult{
		MonitorID:    monitor.ID,
		Model:        monitor.PrimaryModel,
		Status:       model.ChannelMonitorStatusDegraded,
		ErrorCode:    "upstream_error",
		ErrorMessage: "raw upstream response containing sensitive-test-key",
		CheckedAt:    time.Now().Unix(),
	}))

	view, err := BuildChannelMonitorView(monitor, 7, true)
	require.NoError(t, err)
	payload, err := common.Marshal(view)
	require.NoError(t, err)
	assert.NotContains(t, string(payload), "sensitive-test-key")
	assert.NotContains(t, string(payload), "raw upstream response")
	assert.NotContains(t, string(payload), "channel_id")
	assert.NotContains(t, string(payload), "channel_name")
	assert.NotContains(t, string(payload), "group")
	assert.NotContains(t, string(payload), "http_status")
	assert.NotContains(t, string(payload), "error_code")
}

func TestPublicMonitorListExcludesDisabledMonitors(t *testing.T) {
	setupChannelMonitorTest(t)
	monitor := createMonitorFixture(t, constant.ChannelTypeOpenAI, "gpt-4o-mini")
	monitor.Enabled = false
	require.NoError(t, model.UpdateChannelMonitor(monitor))

	views, summary, err := ListChannelMonitorViews(7, true)

	require.NoError(t, err)
	assert.Empty(t, views)
	assert.Zero(t, summary.VisibleMonitors)
	assert.Zero(t, summary.ObservedMonitors)

	adminViews, _, err := ListAdminChannelMonitorViews(7)
	require.NoError(t, err)
	require.Len(t, adminViews, 1)
	assert.False(t, adminViews[0].Enabled)
}

func TestPublicChannelMonitorViewsExcludeDisabledChannels(t *testing.T) {
	setupChannelMonitorTest(t)
	monitor := createMonitorFixture(t, constant.ChannelTypeOpenAI, "gemini-banana-pro-4k")
	channel, err := model.GetChannelById(monitor.ChannelID, false)
	require.NoError(t, err)
	channel.Status = common.ChannelStatusManuallyDisabled
	require.NoError(t, model.DB.Save(channel).Error)

	items, summary, err := ListPublicChannelMonitorViews(7)

	require.NoError(t, err)
	assert.Empty(t, items)
	assert.Zero(t, summary.Total)
}

func TestPublicChannelMonitorViewsIncludeEnabledTextGroupsWithoutMonitor(t *testing.T) {
	setupChannelMonitorTest(t)
	channel := &model.Channel{
		Name: "dynamic-text", Type: constant.ChannelTypeOpenAI, Key: "sensitive-test-key",
		Status: common.ChannelStatusEnabled, Models: "gpt-5.6-sol", Group: "LLM-GPT-快速,LLM-GPT-plus",
	}
	require.NoError(t, model.DB.Create(channel).Error)

	items, summary, err := ListPublicChannelMonitorViews(7)

	require.NoError(t, err)
	require.Len(t, items, 2)
	assert.Equal(t, 2, summary.Total)
	assert.Equal(t, 2, summary.Unknown)
	assert.Equal(t, "LLM-GPT-plus", items[0].Name)
	assert.Equal(t, ChannelMonitorCategoryText, items[0].Category)
	assert.Equal(t, model.ChannelMonitorStatusUnknown, items[0].LatestStatus)
	require.Len(t, items[0].Timeline, channelMonitorPublicTimelineLimit)
}

func TestPassiveMediaMatchesPublicAliasOfInternalModel(t *testing.T) {
	now := time.Date(2026, 8, 1, 22, 17, 0, 0, time.UTC)
	channel := &model.Channel{Id: 75, Type: constant.ChannelTypeOpenAI}
	modelPublicRegistryMu.Lock()
	previousRegistry := modelPublicRegistryData
	modelPublicRegistryData.internalToPublic = map[string]string{
		"adobe-firefly-nano-banana-1k": "nano-banana-1k",
	}
	modelPublicRegistryMu.Unlock()
	t.Cleanup(func() {
		modelPublicRegistryMu.Lock()
		modelPublicRegistryData = previousRegistry
		modelPublicRegistryMu.Unlock()
	})
	tasks := []*model.Task{{
		ID: 1, UpdatedAt: now.Add(-5 * time.Minute).Unix(), Status: model.TaskStatusSuccess,
		Properties: model.Properties{ClientModelName: "nano-banana-1k"},
	}}

	stat := buildPassiveMediaStatFromTasksAt(channel, "adobe-firefly-nano-banana-1k", tasks, true, now)

	assert.Equal(t, model.ChannelMonitorStatusOperational, stat.LatestStatus)
	assert.Equal(t, 1, stat.Observed)
	assert.Equal(t, 1, stat.Operational)
}

func TestPublicChannelMonitorViewsGroupTextAndExposePublicMediaModels(t *testing.T) {
	setupChannelMonitorTest(t)
	textMonitor := createMonitorFixture(t, constant.ChannelTypeOpenAI, "internal-gpt-primary")
	textChannel, err := model.GetChannelById(textMonitor.ChannelID, false)
	require.NoError(t, err)
	textChannel.Group = "LLM-GPT-pro, shared"
	textChannel.Models = "internal-gpt-primary,internal-gpt-extra,cy-img1-seedream-4"
	require.NoError(t, model.DB.Save(textChannel).Error)
	textLatency := 90
	require.NoError(t, model.CreateChannelMonitorResult(&model.ChannelMonitorResult{
		MonitorID: textMonitor.ID, Model: textMonitor.PrimaryModel,
		Status: model.ChannelMonitorStatusOperational, LatencyMs: &textLatency, CheckedAt: time.Now().Unix(),
	}))
	for index := 1; index <= 55; index++ {
		require.NoError(t, model.CreateChannelMonitorResult(&model.ChannelMonitorResult{
			MonitorID: textMonitor.ID, Model: textMonitor.PrimaryModel,
			Status: model.ChannelMonitorStatusOperational, LatencyMs: &textLatency,
			CheckedAt: time.Now().Add(-time.Duration(index) * time.Minute).Unix(),
		}))
	}

	mediaMonitor := createMonitorFixture(t, constant.ChannelTypeOpenAI, "cy-img1-gpt-image-2")
	mediaChannel, err := model.GetChannelById(mediaMonitor.ChannelID, false)
	require.NoError(t, err)
	mediaChannel.Models = "cy-img1-gpt-image-2,cy-img1-gpt-image-2-edit,cy-sd4-seedance-2.0"
	require.NoError(t, model.DB.Save(mediaChannel).Error)

	modelPublicRegistryMu.Lock()
	previousRegistry := modelPublicRegistryData
	modelPublicRegistryData.channelPrefixes = []string{"cy-img1-", "cy-sd4-"}
	modelPublicRegistryData.internalToPublic = map[string]string{
		"cy-img1-seedream-4":       "seedream-4",
		"cy-img1-gpt-image-2":      "gpt-image-2",
		"cy-img1-gpt-image-2-edit": "gpt-image-2-edit",
		"cy-sd4-seedance-2.0":      "seedance-2.0",
	}
	modelPublicRegistryMu.Unlock()
	t.Cleanup(func() {
		modelPublicRegistryMu.Lock()
		modelPublicRegistryData = previousRegistry
		modelPublicRegistryMu.Unlock()
	})

	items, summary, err := listPublicChannelMonitorViewsUncached(7)
	require.NoError(t, err)
	require.Len(t, items, 6)
	assert.Equal(t, 6, summary.Total)

	byKey := make(map[string]*PublicChannelMonitorItem)
	for _, item := range items {
		byKey[item.Category+":"+item.Name] = item
	}
	assert.Equal(t, model.ChannelMonitorStatusOperational, byKey["text:LLM-GPT-pro"].LatestStatus)
	assert.Equal(t, model.ChannelMonitorStatusOperational, byKey["text:shared"].LatestStatus)
	require.Len(t, byKey["text:LLM-GPT-pro"].Timeline, channelMonitorPublicTimelineLimit)
	assert.Equal(t, model.ChannelMonitorStatusOperational, byKey["text:LLM-GPT-pro"].Timeline[len(byKey["text:LLM-GPT-pro"].Timeline)-1].Status)
	assert.Contains(t, byKey, "image:gpt-image-2")
	assert.Contains(t, byKey, "image:gpt-image-2-edit")
	assert.Contains(t, byKey, "image:seedream-4")
	assert.Contains(t, byKey, "video:seedance-2.0")
	assert.NotContains(t, byKey, "text:internal-gpt-primary")
	assert.NotContains(t, byKey, "text:internal-gpt-extra")
	payload, err := common.Marshal(items)
	require.NoError(t, err)
	assert.NotContains(t, string(payload), "cy-img1-")
	assert.NotContains(t, string(payload), "cy-sd4-")
	assert.NotContains(t, string(payload), "internal-gpt")
	assert.NotContains(t, string(payload), "observed_checks")
	assert.NotContains(t, string(payload), "operational_checks")
	assert.NotContains(t, string(payload), "latest_checked_at")
	assert.NotContains(t, string(payload), "checked_at")
	assert.NotContains(t, string(payload), `"latency_ms":`)
}

func TestPublicChannelMonitorViewsPreferOperationalFallback(t *testing.T) {
	aggregates := make(map[string]*publicChannelMonitorAggregate)
	checkedEarly := int64(100)
	checkedLate := int64(200)
	mergePublicChannelMonitorStat(aggregates, "gpt-image-2", ChannelMonitorCategoryImage, &ChannelMonitorModelStat{
		LatestStatus: model.ChannelMonitorStatusUnavailable, LatestChecked: &checkedLate,
		Observed: 1, Timeline: []*ChannelMonitorTimelinePoint{{Status: model.ChannelMonitorStatusUnavailable, CheckedAt: checkedLate}},
	})
	mergePublicChannelMonitorStat(aggregates, "gpt-image-2", ChannelMonitorCategoryImage, &ChannelMonitorModelStat{
		LatestStatus: model.ChannelMonitorStatusOperational, LatestChecked: &checkedEarly,
		Observed: 1, Operational: 1,
		Timeline: []*ChannelMonitorTimelinePoint{{Status: model.ChannelMonitorStatusOperational, CheckedAt: checkedEarly}},
	})

	aggregate := aggregates[ChannelMonitorCategoryImage+"\x00gpt-image-2"]
	require.NotNil(t, aggregate)
	assert.Equal(t, model.ChannelMonitorStatusOperational, aggregate.item.LatestStatus)
	require.Len(t, aggregate.timeline, 2)
	assert.Equal(t, checkedLate, aggregate.timeline[0].CheckedAt)
	assert.Equal(t, checkedEarly, aggregate.timeline[1].CheckedAt)
}

func TestBuildPassiveMediaStatFromTasksPreservesFreshnessAndModelBoundaries(t *testing.T) {
	now := time.Now()
	channel := &model.Channel{Id: 1, Type: constant.ChannelTypeOpenAI}
	tasks := []*model.Task{
		{
			UpdatedAt: now.Unix(), Status: model.TaskStatusSuccess,
			Properties: model.Properties{OriginModelName: "gpt-image-2"},
		},
		{
			UpdatedAt: now.Add(-time.Hour).Unix(), Status: model.TaskStatusFailure,
			FailReason: "bad response status 502",
			Properties: model.Properties{OriginModelName: "gpt-image-2"},
		},
		{
			UpdatedAt: now.Unix(), Status: model.TaskStatusFailure,
			FailReason: "bad response status 502",
			Properties: model.Properties{OriginModelName: "other-image"},
		},
	}

	stat := buildPassiveMediaStatFromTasks(channel, "gpt-image-2", tasks, true)

	assert.Equal(t, model.ChannelMonitorStatusOperational, stat.LatestStatus)
	assert.Equal(t, 2, stat.Observed)
	assert.Equal(t, 1, stat.Operational)
	require.Len(t, stat.Timeline, channelMonitorPublicTimelineLimit)
}

func TestBuildPassiveMediaTimelineUsesFixedBucketsAndCarriesState(t *testing.T) {
	now := time.Date(2026, 8, 1, 22, 17, 0, 0, time.UTC)
	channel := &model.Channel{Id: 1, Type: constant.ChannelTypeOpenAI}
	tasks := []*model.Task{
		{
			UpdatedAt: now.Add(-5 * time.Minute).Unix(), Status: model.TaskStatusSuccess,
			Properties: model.Properties{OriginModelName: "gpt-image-2"},
		},
		{
			UpdatedAt: now.Add(-35 * time.Minute).Unix(), Status: model.TaskStatusFailure,
			FailReason: "bad response status 502",
			Properties: model.Properties{OriginModelName: "gpt-image-2"},
		},
		{
			UpdatedAt: now.Add(-40 * time.Minute).Unix(), Status: model.TaskStatusSuccess,
			Properties: model.Properties{OriginModelName: "gpt-image-2"},
		},
	}

	stat := buildPassiveMediaStatFromTasksAt(channel, "gpt-image-2", tasks, true, now)

	require.Len(t, stat.Timeline, channelMonitorPublicTimelineLimit)
	latest := stat.Timeline[len(stat.Timeline)-1]
	previous := stat.Timeline[len(stat.Timeline)-2]
	carried := stat.Timeline[len(stat.Timeline)-3]
	assert.Equal(t, model.ChannelMonitorStatusOperational, latest.Status)
	assert.False(t, latest.Carried)
	assert.Equal(t, model.ChannelMonitorStatusDegraded, previous.Status)
	assert.False(t, previous.Carried)
	assert.Equal(t, model.ChannelMonitorStatusUnknown, carried.Status)
	assert.True(t, carried.Carried)
	assert.Equal(t, 2, stat.Observed)
	assert.Equal(t, 1, stat.Operational)
	assert.InDelta(t, 50, *stat.Availability, 0.001)
}

func TestBuildPublicTextTimelineUsesOneBarPerProbeRound(t *testing.T) {
	points := make([]*ChannelMonitorTimelinePoint, 0, 55)
	for index := 0; index < 55; index++ {
		status := model.ChannelMonitorStatusOperational
		if index == 54 {
			status = model.ChannelMonitorStatusUnavailable
		}
		points = append(points, &ChannelMonitorTimelinePoint{
			Status: status, CheckedAt: int64(index * 300),
		})
	}

	timeline := buildPublicTextChannelMonitorTimeline(points)

	require.Len(t, timeline, channelMonitorPublicTimelineLimit)
	for _, point := range timeline[:len(timeline)-1] {
		assert.Equal(t, model.ChannelMonitorStatusOperational, point.Status)
		assert.False(t, point.Carried)
	}
	assert.Equal(t, model.ChannelMonitorStatusUnavailable, timeline[len(timeline)-1].Status)
	assert.False(t, timeline[len(timeline)-1].Carried)
}

func TestBuildPublicTextTimelinePadsOnlyMissingProbeRounds(t *testing.T) {
	points := []*ChannelMonitorTimelinePoint{
		{Status: model.ChannelMonitorStatusOperational, CheckedAt: 300},
		{Status: model.ChannelMonitorStatusUnavailable, CheckedAt: 600},
	}

	timeline := buildPublicTextChannelMonitorTimeline(points)

	require.Len(t, timeline, channelMonitorPublicTimelineLimit)
	for _, point := range timeline[:channelMonitorPublicTimelineLimit-2] {
		assert.Equal(t, model.ChannelMonitorStatusUnknown, point.Status)
		assert.True(t, point.Carried)
	}
	assert.Equal(t, model.ChannelMonitorStatusOperational, timeline[channelMonitorPublicTimelineLimit-2].Status)
	assert.False(t, timeline[channelMonitorPublicTimelineLimit-2].Carried)
	assert.Equal(t, model.ChannelMonitorStatusUnavailable, timeline[channelMonitorPublicTimelineLimit-1].Status)
	assert.False(t, timeline[channelMonitorPublicTimelineLimit-1].Carried)
}

func TestBuildPassiveMediaTimelineCarriesLastKnownStatusWithoutCountingBucket(t *testing.T) {
	now := time.Date(2026, 8, 1, 22, 17, 0, 0, time.UTC)
	channel := &model.Channel{Id: 1, Type: constant.ChannelTypeOpenAI}
	tasks := []*model.Task{{
		UpdatedAt: now.Add(-35 * time.Minute).Unix(), Status: model.TaskStatusSuccess,
		Properties: model.Properties{OriginModelName: "gpt-image-2"},
	}}

	stat := buildPassiveMediaStatFromTasksAt(channel, "gpt-image-2", tasks, true, now)

	latest := stat.Timeline[len(stat.Timeline)-1]
	assert.Equal(t, model.ChannelMonitorStatusOperational, latest.Status)
	assert.True(t, latest.Carried)
	assert.Equal(t, model.ChannelMonitorStatusOperational, stat.LatestStatus)
	assert.Equal(t, 1, stat.Observed)
	assert.Equal(t, 1, stat.Operational)
}

func TestBuildPassiveMediaTimelineUsesPreWindowBaseline(t *testing.T) {
	now := time.Date(2026, 8, 1, 22, 17, 0, 0, time.UTC)
	channel := &model.Channel{Id: 1, Type: constant.ChannelTypeOpenAI}
	tasks := []*model.Task{{
		UpdatedAt: now.Add(-25 * time.Hour).Unix(), Status: model.TaskStatusSuccess,
		Properties: model.Properties{OriginModelName: "gpt-image-2"},
	}}

	stat := buildPassiveMediaStatFromTasksAt(channel, "gpt-image-2", tasks, true, now)

	require.Len(t, stat.Timeline, channelMonitorPublicTimelineLimit)
	for _, point := range stat.Timeline {
		assert.Equal(t, model.ChannelMonitorStatusOperational, point.Status)
		assert.True(t, point.Carried)
	}
	assert.Zero(t, stat.Observed)
	assert.Nil(t, stat.Availability)
}

func TestMergeChannelMonitorMediaTasksDeduplicatesAndPrunes(t *testing.T) {
	existing := []*model.Task{
		{ID: 1, UpdatedAt: 100, FailReason: "old"},
		{ID: 2, UpdatedAt: 200},
	}
	fresh := []*model.Task{
		{ID: 2, UpdatedAt: 200, FailReason: "refreshed"},
		{ID: 3, UpdatedAt: 300},
	}

	merged := mergeChannelMonitorMediaTasks(existing, fresh, 150)

	require.Len(t, merged, 2)
	assert.Equal(t, int64(3), merged[0].ID)
	assert.Equal(t, int64(2), merged[1].ID)
	assert.Equal(t, "refreshed", merged[1].FailReason)
}

func TestChannelMonitorLeasePreventsDuplicateClaimsAndExpires(t *testing.T) {
	setupChannelMonitorTest(t)
	monitor := createMonitorFixture(t, constant.ChannelTypeOpenAI, "gpt-4o-mini")
	now := time.Now().Unix()

	claimed, err := model.ClaimChannelMonitor(monitor.ID, now, now+300, now+120, false)
	require.NoError(t, err)
	assert.True(t, claimed)

	claimed, err = model.ClaimChannelMonitor(monitor.ID, now, now+300, now+120, false)
	require.NoError(t, err)
	assert.False(t, claimed)

	claimed, err = model.ClaimChannelMonitor(monitor.ID, now+121, now+421, now+241, true)
	require.NoError(t, err)
	assert.True(t, claimed)
}
