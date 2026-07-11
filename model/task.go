package model

import (
	"bytes"
	"context"
	"database/sql/driver"
	"encoding/json"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	commonRelay "github.com/QuantumNous/new-api/relay/common"

	"gorm.io/gorm"
)

type TaskStatus string

func (t TaskStatus) ToVideoStatus() string {
	var status string
	switch t {
	case TaskStatusQueued, TaskStatusSubmitted:
		status = dto.VideoStatusQueued
	case TaskStatusInProgress:
		status = dto.VideoStatusInProgress
	case TaskStatusSuccess:
		status = dto.VideoStatusCompleted
	case TaskStatusFailure:
		status = dto.VideoStatusFailed
	default:
		status = dto.VideoStatusUnknown // Default fallback
	}
	return status
}

const (
	TaskStatusNotStart   TaskStatus = "NOT_START"
	TaskStatusSubmitted             = "SUBMITTED"
	TaskStatusQueued                = "QUEUED"
	TaskStatusInProgress            = "IN_PROGRESS"
	TaskStatusFailure               = "FAILURE"
	TaskStatusSuccess               = "SUCCESS"
	TaskStatusUnknown               = "UNKNOWN"
)

type Task struct {
	ID         int64                 `json:"id" gorm:"primary_key;AUTO_INCREMENT"`
	CreatedAt  int64                 `json:"created_at" gorm:"index"`
	UpdatedAt  int64                 `json:"updated_at"`
	TaskID     string                `json:"task_id" gorm:"type:varchar(191);index"` // 第三方id，不一定有/ song id\ Task id
	Platform   constant.TaskPlatform `json:"platform" gorm:"type:varchar(30);index"` // 平台
	UserId     int                   `json:"user_id" gorm:"index"`
	Group      string                `json:"group" gorm:"type:varchar(50)"` // 修正计费用
	ChannelId  int                   `json:"channel_id" gorm:"index"`
	Quota      int                   `json:"quota"`
	Action     string                `json:"action" gorm:"type:varchar(40);index"` // 任务类型, song, lyrics, description-mode
	Status     TaskStatus            `json:"status" gorm:"type:varchar(20);index"` // 任务状态
	FailReason string                `json:"fail_reason"`
	SubmitTime int64                 `json:"submit_time" gorm:"index"`
	StartTime  int64                 `json:"start_time" gorm:"index"`
	FinishTime int64                 `json:"finish_time" gorm:"index"`
	Progress   string                `json:"progress" gorm:"type:varchar(20);index"`
	Properties Properties            `json:"properties" gorm:"type:json"`
	Username   string                `json:"username,omitempty" gorm:"-"`
	// 禁止返回给用户，内部可能包含key等隐私信息
	PrivateData TaskPrivateData `json:"-" gorm:"column:private_data;type:json"`
	Data        json.RawMessage `json:"data" gorm:"type:json"`
}

func (t *Task) SetData(data any) {
	b, _ := common.Marshal(data)
	t.Data = json.RawMessage(b)
}

func (t *Task) GetData(v any) error {
	return common.Unmarshal(t.Data, &v)
}

type Properties struct {
	Input             string `json:"input"`
	UpstreamModelName string `json:"upstream_model_name,omitempty"`
	OriginModelName   string `json:"origin_model_name,omitempty"`
	ClientModelName   string `json:"client_model_name,omitempty"`
	TaskKind          string `json:"task_kind,omitempty"`
}

func (m *Properties) Scan(val interface{}) error {
	bytesValue, _ := val.([]byte)
	if len(bytesValue) == 0 {
		*m = Properties{}
		return nil
	}
	return common.Unmarshal(bytesValue, m)
}

func (m Properties) Value() (driver.Value, error) {
	if m == (Properties{}) {
		return nil, nil
	}
	return common.Marshal(m)
}

type TaskPrivateData struct {
	Key            string `json:"key,omitempty"`
	UpstreamTaskID string `json:"upstream_task_id,omitempty"` // 上游真实 task ID
	ResultURL      string `json:"result_url,omitempty"`       // 任务成功后的结果 URL（视频地址等）
	RequestPath    string `json:"request_path,omitempty"`
	RequestSnapshot []byte `json:"request_snapshot,omitempty"`
	ImageResultURLs []string `json:"image_result_urls,omitempty"`
	// 计费上下文：用于异步退款/差额结算（轮询阶段读取）
	BillingSource  string              `json:"billing_source,omitempty"`  // "wallet" 或 "subscription"
	SubscriptionId int                 `json:"subscription_id,omitempty"` // 订阅 ID，用于订阅退款
	TokenId        int                 `json:"token_id,omitempty"`        // 令牌 ID，用于令牌额度退款
	BillingContext *TaskBillingContext `json:"billing_context,omitempty"` // 计费参数快照（用于轮询阶段重新计算）
}

// TaskBillingContext 记录任务提交时的计费参数，以便轮询阶段可以重新计算额度。
type TaskBillingContext struct {
	ModelPrice      float64            `json:"model_price,omitempty"`       // 模型单价
	GroupRatio      float64            `json:"group_ratio,omitempty"`       // 分组倍率
	ModelRatio      float64            `json:"model_ratio,omitempty"`       // 模型倍率
	OtherRatios     map[string]float64 `json:"other_ratios,omitempty"`      // 附加倍率（时长、分辨率等）
	OriginModelName   string             `json:"origin_model_name,omitempty"` // 模型名称，必须为OriginModelName
	UpstreamModelName string             `json:"upstream_model_name,omitempty"` // 上游模型名（轮询计费路由）
	PerCallBilling    bool               `json:"per_call_billing,omitempty"`  // 按次计费：跳过轮询阶段的差额结算
}

// GetUpstreamTaskID 获取上游真实 task ID（用于与 provider 通信）
// 旧数据没有 UpstreamTaskID 时，TaskID 本身就是上游 ID
func (t *Task) GetUpstreamTaskID() string {
	if t.PrivateData.UpstreamTaskID != "" {
		return t.PrivateData.UpstreamTaskID
	}
	return t.TaskID
}

// GetResultURL 获取任务结果 URL（视频地址等）
// 新数据存在 PrivateData.ResultURL 中；旧数据回退到 FailReason（历史兼容）
func (t *Task) GetResultURL() string {
	if t.PrivateData.ResultURL != "" {
		return t.PrivateData.ResultURL
	}
	return t.FailReason
}

// ReleaseRequestSnapshot drops the async request body snapshot after a task finishes.
func (t *Task) ReleaseRequestSnapshot() {
	t.PrivateData.RequestSnapshot = nil
}

// GenerateTaskID 生成对外暴露的 task_xxxx 格式 ID
func GenerateTaskID() string {
	key, _ := common.GenerateRandomCharsKey(32)
	return "task_" + key
}

func (p *TaskPrivateData) Scan(val interface{}) error {
	bytesValue, _ := val.([]byte)
	if len(bytesValue) == 0 {
		return nil
	}
	return common.Unmarshal(bytesValue, p)
}

func (p TaskPrivateData) Value() (driver.Value, error) {
	b, err := common.Marshal(p)
	if err != nil {
		return nil, err
	}
	if len(b) <= 2 {
		return nil, nil
	}
	return b, nil
}

// SyncTaskQueryParams 用于包含所有搜索条件的结构体，可以根据需求添加更多字段
type SyncTaskQueryParams struct {
	Platform       constant.TaskPlatform
	ChannelID      string
	TaskID         string
	UserID         string
	Action         string
	Status         string
	StartTimestamp int64
	EndTimestamp   int64
	UserIDs        []int
}

func InitTask(platform constant.TaskPlatform, relayInfo *commonRelay.RelayInfo) *Task {
	properties := Properties{}
	privateData := TaskPrivateData{}
	if relayInfo == nil {
		relayInfo = &commonRelay.RelayInfo{}
	}
	if relayInfo.ChannelMeta != nil {
		if relayInfo.ChannelMeta.ChannelType == constant.ChannelTypeGemini ||
			relayInfo.ChannelMeta.ChannelType == constant.ChannelTypeVertexAi {
			privateData.Key = relayInfo.ChannelMeta.ApiKey
		}
	}
	if relayInfo.UpstreamModelName != "" {
		properties.UpstreamModelName = relayInfo.UpstreamModelName
	}
	if relayInfo.OriginModelName != "" {
		properties.OriginModelName = relayInfo.OriginModelName
	}
	if relayInfo.ClientModelName != "" {
		properties.ClientModelName = relayInfo.ClientModelName
	}

	// 使用预生成的公开 ID（如果有），否则新生成
	taskID := ""
	if relayInfo.TaskRelayInfo != nil && relayInfo.TaskRelayInfo.PublicTaskID != "" {
		taskID = relayInfo.TaskRelayInfo.PublicTaskID
	} else {
		taskID = GenerateTaskID()
	}

	channelId := 0
	if relayInfo.ChannelMeta != nil {
		channelId = relayInfo.ChannelMeta.ChannelId
	}

	t := &Task{
		TaskID:      taskID,
		UserId:      relayInfo.UserId,
		Group:       relayInfo.UsingGroup,
		SubmitTime:  time.Now().Unix(),
		Status:      TaskStatusNotStart,
		Progress:    "0%",
		ChannelId:   channelId,
		Platform:    platform,
		Properties:  properties,
		PrivateData: privateData,
	}
	return t
}

// ApplySubmittedStatusFromUpstreamData 视频任务提交成功后设置初始状态（避免长期显示 NOT_START）。
func ApplySubmittedStatusFromUpstreamData(task *Task, taskData []byte) {
	if task == nil {
		return
	}
	task.Status = TaskStatusSubmitted
	task.Progress = "10%"
	if len(taskData) == 0 {
		return
	}
	var payload map[string]any
	if err := common.Unmarshal(taskData, &payload); err != nil {
		return
	}
	status, _ := payload["status"].(string)
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "queued", "pending":
		task.Status = TaskStatusQueued
		task.Progress = "20%"
	case "processing", "in_progress", "running":
		task.Status = TaskStatusInProgress
		task.Progress = "30%"
	}
}

// taskListPrivateDataSelectExpr returns a dialect-specific SQL expression that
// projects only result_url from private_data, avoiding multi-MB request_snapshot reads.
func taskListPrivateDataSelectExpr() string {
	switch {
	case common.UsingPostgreSQL:
		return `jsonb_build_object('result_url', COALESCE(private_data->>'result_url', ''))::json`
	case common.UsingMySQL:
		return `JSON_OBJECT('result_url', COALESCE(JSON_UNQUOTE(JSON_EXTRACT(private_data, '$.result_url')), ''))`
	default:
		return `json_object('result_url', COALESCE(json_extract(private_data, '$.result_url'), ''))`
	}
}

// taskPrivateDataWithoutSnapshotExpr projects private_data without request_snapshot,
// avoiding multi-MB reads on status fetch and unfinished-task polling.
func taskPrivateDataWithoutSnapshotExpr() string {
	switch {
	case common.UsingPostgreSQL:
		// private_data 列类型为 json（非 jsonb），COALESCE 两侧须同类型；先 cast 再 jsonb 减键。
		return `(COALESCE(private_data::jsonb, '{}'::jsonb) - 'request_snapshot')::json`
	case common.UsingMySQL:
		return `JSON_REMOVE(COALESCE(private_data, JSON_OBJECT()), '$.request_snapshot')`
	default:
		return `json_remove(COALESCE(private_data, '{}'), '$.request_snapshot')`
	}
}

func taskFetchSelectClause() string {
	return strings.Join([]string{
		"id", "created_at", "updated_at", "task_id", "platform", "user_id",
		commonGroupCol,
		"channel_id", "quota", "action", "status", "fail_reason",
		"submit_time", "start_time", "finish_time", "progress",
		"properties", "data",
		taskPrivateDataWithoutSnapshotExpr() + " AS private_data",
	}, ", ")
}

func applyTaskFetchSelect(query *gorm.DB) *gorm.DB {
	return query.Select(taskFetchSelectClause())
}

func taskListSelectClause(includeChannelID bool) string {
	cols := []string{
		"id", "created_at", "updated_at", "task_id", "platform", "user_id",
		commonGroupCol,
	}
	if includeChannelID {
		cols = append(cols, "channel_id")
	}
	cols = append(cols,
		"quota", "action", "status", "fail_reason",
		"submit_time", "start_time", "finish_time", "progress",
		"properties", "data",
		taskListPrivateDataSelectExpr()+" AS private_data",
	)
	return strings.Join(cols, ", ")
}

func applyTaskListSelect(query *gorm.DB, includeChannelID bool) *gorm.DB {
	return query.Select(taskListSelectClause(includeChannelID))
}

func TaskGetAllUserTask(userId int, startIdx int, num int, queryParams SyncTaskQueryParams) []*Task {
	var tasks []*Task
	var err error

	// 初始化查询构建器
	query := DB.Where("user_id = ?", userId)

	if queryParams.TaskID != "" {
		query = query.Where("task_id = ?", queryParams.TaskID)
	}
	if queryParams.Action != "" {
		query = query.Where("action = ?", queryParams.Action)
	}
	if queryParams.Status != "" {
		query = query.Where("status = ?", queryParams.Status)
	}
	if queryParams.Platform != "" {
		query = query.Where("platform = ?", queryParams.Platform)
	}
	if queryParams.StartTimestamp != 0 {
		// 假设您已将前端传来的时间戳转换为数据库所需的时间格式，并处理了时间戳的验证和解析
		query = query.Where("submit_time >= ?", queryParams.StartTimestamp)
	}
	if queryParams.EndTimestamp != 0 {
		query = query.Where("submit_time <= ?", queryParams.EndTimestamp)
	}

	// 列表页只需 result_url，避免读取 private_data.request_snapshot 等大字段。
	err = applyTaskListSelect(query, false).Order("id desc").Limit(num).Offset(startIdx).Find(&tasks).Error
	if err != nil {
		return nil
	}

	return tasks
}

func TaskGetAllTasks(startIdx int, num int, queryParams SyncTaskQueryParams) []*Task {
	var tasks []*Task
	var err error

	// 初始化查询构建器
	query := DB

	// 添加过滤条件
	if queryParams.ChannelID != "" {
		query = query.Where("channel_id = ?", queryParams.ChannelID)
	}
	if queryParams.Platform != "" {
		query = query.Where("platform = ?", queryParams.Platform)
	}
	if queryParams.UserID != "" {
		query = query.Where("user_id = ?", queryParams.UserID)
	}
	if len(queryParams.UserIDs) != 0 {
		query = query.Where("user_id in (?)", queryParams.UserIDs)
	}
	if queryParams.TaskID != "" {
		query = query.Where("task_id = ?", queryParams.TaskID)
	}
	if queryParams.Action != "" {
		query = query.Where("action = ?", queryParams.Action)
	}
	if queryParams.Status != "" {
		query = query.Where("status = ?", queryParams.Status)
	}
	if queryParams.StartTimestamp != 0 {
		query = query.Where("submit_time >= ?", queryParams.StartTimestamp)
	}
	if queryParams.EndTimestamp != 0 {
		query = query.Where("submit_time <= ?", queryParams.EndTimestamp)
	}

	// 列表页只需 result_url，避免读取 private_data.request_snapshot 等大字段。
	err = applyTaskListSelect(query, true).Order("id desc").Limit(num).Offset(startIdx).Find(&tasks).Error
	if err != nil {
		return nil
	}

	return tasks
}

func GetTimedOutUnfinishedTasks(cutoffUnix int64, limit int) []*Task {
	var tasks []*Task
	err := DB.Where("progress != ?", "100%").
		Where("status NOT IN ?", []string{TaskStatusFailure, TaskStatusSuccess}).
		Where("submit_time < ?", cutoffUnix).
		Order("submit_time").
		Limit(limit).
		Find(&tasks).Error
	if err != nil {
		return nil
	}
	return tasks
}

func GetAllUnFinishSyncTasks(limit int) []*Task {
	var tasks []*Task
	var err error
	// get all tasks progress is not 100%
	err = applyTaskFetchSelect(DB.Where("progress != ?", "100%").
		Where("status != ?", TaskStatusFailure).
		Where("status != ?", TaskStatusSuccess)).
		Limit(limit).Order("id").Find(&tasks).Error
	if err != nil {
		return nil
	}
	return tasks
}

func GetPendingImageAsyncTasks(limit int) []*Task {
	return getPendingAsyncTasksByKind(constant.TaskKindImage, limit)
}

func getPendingAsyncTasksByKind(kind string, limit int) []*Task {
	if limit <= 0 {
		return nil
	}
	// Async workers need the durable request_snapshot to replay the request after
	// a process restart. Do not reuse GetAllUnFinishSyncTasks here: that query
	// intentionally projects request_snapshot out of private_data for polling
	// and status reads.
	var all []*Task
	if err := DB.Where("progress != ?", "100%").
		Where("status != ?", TaskStatusFailure).
		Where("status != ?", TaskStatusSuccess).
		Limit(limit * 8).
		Order("id").
		Find(&all).Error; err != nil {
		return nil
	}
	if len(all) == 0 {
		return nil
	}
	out := make([]*Task, 0, limit)
	for _, task := range all {
		if task == nil || task.Properties.TaskKind != kind {
			continue
		}
		out = append(out, task)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func GetByOnlyTaskId(taskId string) (*Task, bool, error) {
	if taskId == "" {
		return nil, false, nil
	}
	var task *Task
	var err error
	err = DB.Where("task_id = ?", taskId).First(&task).Error
	exist, err := RecordExist(err)
	if err != nil {
		return nil, false, err
	}
	return task, exist, err
}

func GetByTaskId(userId int, taskId string) (*Task, bool, error) {
	if taskId == "" {
		return nil, false, nil
	}
	var task *Task
	var err error
	err = DB.Where("user_id = ? and task_id = ?", userId, taskId).
		First(&task).Error
	exist, err := RecordExist(err)
	if err != nil {
		return nil, false, err
	}
	return task, exist, err
}

// GetByTaskIdForFetch loads a task for read-only status/result endpoints without
// reading private_data.request_snapshot.
func GetByTaskIdForFetch(userId int, taskId string) (*Task, bool, error) {
	if taskId == "" {
		return nil, false, nil
	}
	var task *Task
	err := applyTaskFetchSelect(DB.Where("user_id = ? and task_id = ?", userId, taskId)).
		First(&task).Error
	exist, err := RecordExist(err)
	if err != nil {
		return nil, false, err
	}
	return task, exist, err
}

func GetByTaskIds(userId int, taskIds []any) ([]*Task, error) {
	if len(taskIds) == 0 {
		return nil, nil
	}
	var task []*Task
	var err error
	err = DB.Where("user_id = ? and task_id in (?)", userId, taskIds).
		Find(&task).Error
	if err != nil {
		return nil, err
	}
	return task, nil
}

// GetByTaskIdsForFetch is the batch variant of GetByTaskIdForFetch.
func GetByTaskIdsForFetch(userId int, taskIds []any) ([]*Task, error) {
	if len(taskIds) == 0 {
		return nil, nil
	}
	var tasks []*Task
	err := applyTaskFetchSelect(DB.Where("user_id = ? and task_id in (?)", userId, taskIds)).
		Find(&tasks).Error
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

func (Task *Task) Insert() error {
	var err error
	err = DB.Create(Task).Error
	return err
}

type taskSnapshot struct {
	Status     TaskStatus
	Progress   string
	StartTime  int64
	FinishTime int64
	FailReason string
	ResultURL  string
	Data       json.RawMessage
}

func (s taskSnapshot) Equal(other taskSnapshot) bool {
	return s.Status == other.Status &&
		s.Progress == other.Progress &&
		s.StartTime == other.StartTime &&
		s.FinishTime == other.FinishTime &&
		s.FailReason == other.FailReason &&
		s.ResultURL == other.ResultURL &&
		bytes.Equal(s.Data, other.Data)
}

func (t *Task) Snapshot() taskSnapshot {
	return taskSnapshot{
		Status:     t.Status,
		Progress:   t.Progress,
		StartTime:  t.StartTime,
		FinishTime: t.FinishTime,
		FailReason: t.FailReason,
		ResultURL:  t.PrivateData.ResultURL,
		Data:       t.Data,
	}
}

func (Task *Task) Update() error {
	var err error
	err = DB.Save(Task).Error
	return err
}

// UpdateQuota persists a terminal task's final quota without rewriting its
// other fields. Billing settlement updates user/token balances separately, so
// keeping this write narrow avoids overwriting a concurrent task snapshot.
func (Task *Task) UpdateQuota(quota int) error {
	return DB.Model(&Task{}).Where("id = ?", Task.ID).Update("quota", quota).Error
}

// UpdateWithStatus performs a conditional UPDATE guarded by fromStatus (CAS).
// Returns (true, nil) if this caller won the update, (false, nil) if
// another process already moved the task out of fromStatus.
//
// Uses Model().Select("*").Updates() instead of Save() because GORM's Save
// falls back to INSERT ON CONFLICT when the WHERE-guarded UPDATE matches
// zero rows, which silently bypasses the CAS guard.
func (t *Task) UpdateWithStatus(fromStatus TaskStatus) (bool, error) {
	result := DB.Model(t).Where("status = ?", fromStatus).Select("*").Updates(t)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

// TaskBulkUpdate performs an unconditional bulk UPDATE by upstream task_id strings.
// Same caveats as TaskBulkUpdateByID — no CAS guard.
func TaskBulkUpdate(taskIds []string, params map[string]any) error {
	if len(taskIds) == 0 {
		return nil
	}
	return DB.Model(&Task{}).
		Where("task_id in (?)", taskIds).
		Updates(params).Error
}

// TaskBulkUpdateByID performs an unconditional bulk UPDATE by primary key IDs.
// WARNING: This function has NO CAS (Compare-And-Swap) guard — it will overwrite
// any concurrent status changes. DO NOT use in billing/quota lifecycle flows
// (e.g., timeout, success, failure transitions that trigger refunds or settlements).
// For status transitions that involve billing, use Task.UpdateWithStatus() instead.
func TaskBulkUpdateByID(ids []int64, params map[string]any) error {
	if len(ids) == 0 {
		return nil
	}
	return DB.Model(&Task{}).
		Where("id in (?)", ids).
		Updates(params).Error
}

type TaskQuotaUsage struct {
	Mode  string  `json:"mode"`
	Count float64 `json:"count"`
}

// TaskCountAllTasks returns total tasks that match the given query params (admin usage)
func TaskCountAllTasks(queryParams SyncTaskQueryParams) int64 {
	var total int64
	query := DB.Model(&Task{})
	if queryParams.ChannelID != "" {
		query = query.Where("channel_id = ?", queryParams.ChannelID)
	}
	if queryParams.Platform != "" {
		query = query.Where("platform = ?", queryParams.Platform)
	}
	if queryParams.UserID != "" {
		query = query.Where("user_id = ?", queryParams.UserID)
	}
	if len(queryParams.UserIDs) != 0 {
		query = query.Where("user_id in (?)", queryParams.UserIDs)
	}
	if queryParams.TaskID != "" {
		query = query.Where("task_id = ?", queryParams.TaskID)
	}
	if queryParams.Action != "" {
		query = query.Where("action = ?", queryParams.Action)
	}
	if queryParams.Status != "" {
		query = query.Where("status = ?", queryParams.Status)
	}
	if queryParams.StartTimestamp != 0 {
		query = query.Where("submit_time >= ?", queryParams.StartTimestamp)
	}
	if queryParams.EndTimestamp != 0 {
		query = query.Where("submit_time <= ?", queryParams.EndTimestamp)
	}
	_ = query.Count(&total).Error
	return total
}

// TaskCountAllUserTask returns total tasks for given user
func TaskCountAllUserTask(userId int, queryParams SyncTaskQueryParams) int64 {
	var total int64
	query := DB.Model(&Task{}).Where("user_id = ?", userId)
	if queryParams.TaskID != "" {
		query = query.Where("task_id = ?", queryParams.TaskID)
	}
	if queryParams.Action != "" {
		query = query.Where("action = ?", queryParams.Action)
	}
	if queryParams.Status != "" {
		query = query.Where("status = ?", queryParams.Status)
	}
	if queryParams.Platform != "" {
		query = query.Where("platform = ?", queryParams.Platform)
	}
	if queryParams.StartTimestamp != 0 {
		query = query.Where("submit_time >= ?", queryParams.StartTimestamp)
	}
	if queryParams.EndTimestamp != 0 {
		query = query.Where("submit_time <= ?", queryParams.EndTimestamp)
	}
	_ = query.Count(&total).Error
	return total
}

// DeleteOldTasks removes finished tasks submitted before targetTimestamp.
// Only SUCCESS/FAILURE rows are deleted so in-flight tasks are preserved.
func DeleteOldTasks(ctx context.Context, targetTimestamp int64, limit int) (int64, error) {
	if targetTimestamp <= 0 || limit <= 0 {
		return 0, nil
	}

	var total int64
	for {
		if err := ctx.Err(); err != nil {
			return total, err
		}

		result := DB.Where("submit_time < ?", targetTimestamp).
			Where("status IN ?", []TaskStatus{TaskStatusSuccess, TaskStatusFailure}).
			Limit(limit).
			Delete(&Task{})
		if result.Error != nil {
			return total, result.Error
		}

		total += result.RowsAffected
		if result.RowsAffected < int64(limit) {
			break
		}
	}

	return total, nil
}

// StripFinishedTaskRequestSnapshots clears request_snapshot from finished tasks.
func StripFinishedTaskRequestSnapshots(ctx context.Context, maxUpdates int) (int64, error) {
	if maxUpdates <= 0 {
		return 0, nil
	}

	var updated int64
	var lastID int64
	for updated < int64(maxUpdates) {
		if err := ctx.Err(); err != nil {
			return updated, err
		}

		var batch []*Task
		err := DB.Select("id", "private_data").
			Where("status IN ?", []TaskStatus{TaskStatusSuccess, TaskStatusFailure}).
			Where("id > ?", lastID).
			Order("id ASC").
			Limit(32).
			Find(&batch).Error
		if err != nil {
			return updated, err
		}
		if len(batch) == 0 {
			break
		}

		for _, task := range batch {
			lastID = task.ID
			if len(task.PrivateData.RequestSnapshot) == 0 {
				continue
			}
			task.ReleaseRequestSnapshot()
			if err := DB.Model(&Task{}).Where("id = ?", task.ID).Update("private_data", task.PrivateData).Error; err != nil {
				return updated, err
			}
			updated++
			if updated >= int64(maxUpdates) {
				break
			}
		}
	}

	return updated, nil
}

// ToOpenAIVideo 构建 Video DTO；Model 为 internal 名。
// 对外响应请 marshal 后使用 service.PatchClientFacingModelJSONFromTask。
func (t *Task) ToOpenAIVideo() *dto.OpenAIVideo {
	openAIVideo := dto.NewOpenAIVideo()
	openAIVideo.ID = t.TaskID
	openAIVideo.Status = t.Status.ToVideoStatus()
	openAIVideo.Model = t.Properties.OriginModelName
	openAIVideo.SetProgressStr(t.Progress)
	openAIVideo.CreatedAt = t.CreatedAt
	openAIVideo.CompletedAt = t.UpdatedAt
	openAIVideo.SetMetadata("url", t.GetResultURL())
	return openAIVideo
}

// ToOpenAIImageJob 构建 Image Job DTO；Model 为 internal 名。
// 对外响应请 marshal 后使用 service.PatchClientFacingModelJSONFromTask。
func (t *Task) ToOpenAIImageJob(object string) *dto.OpenAIImageJob {
	job := dto.NewOpenAIImageJob(object)
	job.ID = t.TaskID
	job.Model = t.Properties.OriginModelName
	job.CreatedAt = t.CreatedAt
	switch t.Status {
	case TaskStatusSuccess:
		job.Status = dto.ImageJobStatusCompleted
		job.Progress = "100%"
	case TaskStatusFailure:
		job.Status = dto.ImageJobStatusFailed
		job.Progress = "100%"
		if t.FailReason != "" {
			job.Error = &dto.OpenAIImageError{Message: t.FailReason}
		}
	case TaskStatusInProgress:
		job.Status = dto.ImageJobStatusInProgress
		job.Progress = t.Progress
	default:
		job.Status = dto.ImageJobStatusQueued
		job.Progress = t.Progress
	}
	if len(t.PrivateData.ImageResultURLs) > 0 {
		for _, url := range t.PrivateData.ImageResultURLs {
			job.Data = append(job.Data, dto.ImageData{Url: url})
		}
	} else if t.GetResultURL() != "" {
		job.Data = []dto.ImageData{{Url: t.GetResultURL()}}
	}
	return job
}
