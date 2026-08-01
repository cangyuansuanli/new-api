package model

import (
	"errors"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

const (
	ChannelMonitorStatusOperational = "operational"
	ChannelMonitorStatusDegraded    = "degraded"
	ChannelMonitorStatusUnavailable = "unavailable"
	ChannelMonitorStatusUnknown     = "unknown"

	ChannelMonitorProbeTextActive   = "text_active"
	ChannelMonitorProbeMediaPassive = "media_passive"
)

// ChannelMonitor stores monitoring policy separately from channel credentials.
// A monitor references an existing channel and never persists a copy of its key.
type ChannelMonitor struct {
	ID              int64  `json:"id" gorm:"primaryKey;autoIncrement"`
	ChannelID       int    `json:"channel_id" gorm:"not null;uniqueIndex"`
	Name            string `json:"name" gorm:"type:varchar(128);not null"`
	PrimaryModel    string `json:"primary_model" gorm:"type:varchar(191);not null"`
	ExtraModelsJSON string `json:"-" gorm:"column:extra_models;type:text"`
	ProbeKind       string `json:"probe_kind" gorm:"type:varchar(32);not null;default:'text_active'"`
	IntervalSeconds int    `json:"interval_seconds" gorm:"not null;default:300"`
	JitterSeconds   int    `json:"jitter_seconds" gorm:"not null;default:30"`
	Enabled         bool   `json:"enabled" gorm:"not null;index"`
	Visible         bool   `json:"visible" gorm:"not null;index"`
	NextProbeAt     int64  `json:"-" gorm:"not null;default:0;index"`
	LeaseExpiresAt  int64  `json:"-" gorm:"not null;default:0;index"`
	CreatedAt       int64  `json:"created_at" gorm:"not null"`
	UpdatedAt       int64  `json:"updated_at" gorm:"not null"`
}

type ChannelMonitorResult struct {
	ID           int64  `json:"id" gorm:"primaryKey;autoIncrement"`
	MonitorID    int64  `json:"monitor_id" gorm:"not null;index:idx_monitor_model_checked,priority:1;index:idx_monitor_checked,priority:1"`
	Model        string `json:"model" gorm:"type:varchar(191);not null;index:idx_monitor_model_checked,priority:2"`
	Status       string `json:"status" gorm:"type:varchar(32);not null;index"`
	LatencyMs    *int   `json:"latency_ms"`
	HTTPStatus   int    `json:"http_status"`
	ErrorCode    string `json:"error_code,omitempty" gorm:"type:varchar(96)"`
	ErrorMessage string `json:"-" gorm:"type:varchar(512)"`
	CheckedAt    int64  `json:"checked_at" gorm:"not null;index:idx_monitor_model_checked,priority:3,sort:desc;index:idx_monitor_checked,priority:2,sort:desc"`
}

func CreateChannelMonitor(monitor *ChannelMonitor) error {
	if monitor == nil {
		return errors.New("channel monitor is required")
	}
	now := time.Now().Unix()
	monitor.CreatedAt = now
	monitor.UpdatedAt = now
	return DB.Create(monitor).Error
}

func UpdateChannelMonitor(monitor *ChannelMonitor) error {
	if monitor == nil || monitor.ID <= 0 {
		return errors.New("valid channel monitor is required")
	}
	monitor.UpdatedAt = time.Now().Unix()
	monitor.NextProbeAt = 0
	monitor.LeaseExpiresAt = 0
	return DB.Model(&ChannelMonitor{}).
		Where("id = ?", monitor.ID).
		Select("channel_id", "name", "primary_model", "extra_models", "probe_kind", "interval_seconds", "jitter_seconds", "enabled", "visible", "next_probe_at", "lease_expires_at", "updated_at").
		Updates(monitor).Error
}

func ClaimChannelMonitor(id int64, now int64, nextProbeAt int64, leaseExpiresAt int64, ignoreSchedule bool) (bool, error) {
	query := DB.Model(&ChannelMonitor{}).
		Where("id = ? AND enabled = ? AND lease_expires_at <= ?", id, true, now)
	if !ignoreSchedule {
		query = query.Where("next_probe_at <= ?", now)
	}
	result := query.Updates(map[string]any{
		"next_probe_at":    nextProbeAt,
		"lease_expires_at": leaseExpiresAt,
	})
	return result.RowsAffected == 1, result.Error
}

func ReleaseChannelMonitorLease(id int64) error {
	return DB.Model(&ChannelMonitor{}).Where("id = ?", id).Update("lease_expires_at", 0).Error
}

func GetChannelMonitorByID(id int64) (*ChannelMonitor, error) {
	var monitor ChannelMonitor
	if err := DB.First(&monitor, id).Error; err != nil {
		return nil, err
	}
	return &monitor, nil
}

func GetChannelMonitorByChannelID(channelID int) (*ChannelMonitor, error) {
	var monitor ChannelMonitor
	if err := DB.Where("channel_id = ?", channelID).First(&monitor).Error; err != nil {
		return nil, err
	}
	return &monitor, nil
}

func ListChannelMonitors(visibleOnly bool, enabledOnly bool) ([]*ChannelMonitor, error) {
	query := DB.Model(&ChannelMonitor{})
	if visibleOnly {
		query = query.Where("visible = ?", true)
	}
	if enabledOnly {
		query = query.Where("enabled = ?", true)
	}
	var monitors []*ChannelMonitor
	if err := query.Order("id ASC").Find(&monitors).Error; err != nil {
		return nil, err
	}
	return monitors, nil
}

func DeleteChannelMonitor(id int64) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("monitor_id = ?", id).Delete(&ChannelMonitorResult{}).Error; err != nil {
			return err
		}
		result := tx.Delete(&ChannelMonitor{}, id)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

func CreateChannelMonitorResult(result *ChannelMonitorResult) error {
	if result == nil {
		return errors.New("channel monitor result is required")
	}
	result.Model = strings.TrimSpace(result.Model)
	if result.CheckedAt == 0 {
		result.CheckedAt = time.Now().Unix()
	}
	return DB.Create(result).Error
}

func ListChannelMonitorResults(monitorID int64, since int64) ([]*ChannelMonitorResult, error) {
	var results []*ChannelMonitorResult
	err := DB.Where("monitor_id = ? AND checked_at >= ?", monitorID, since).
		Order("checked_at ASC, id ASC").
		Find(&results).Error
	return results, err
}

func ListRecentChannelMonitorResults(monitorID int64, modelName string, limit int) ([]*ChannelMonitorResult, error) {
	if limit <= 0 || limit > 200 {
		limit = 24
	}
	query := DB.Where("monitor_id = ?", monitorID)
	if strings.TrimSpace(modelName) != "" {
		query = query.Where("model = ?", strings.TrimSpace(modelName))
	}
	var results []*ChannelMonitorResult
	err := query.Order("checked_at DESC, id DESC").Limit(limit).Find(&results).Error
	return results, err
}

func GetLatestChannelMonitorResult(monitorID int64, modelName string) (*ChannelMonitorResult, error) {
	var result ChannelMonitorResult
	err := DB.Where("monitor_id = ? AND model = ?", monitorID, strings.TrimSpace(modelName)).
		Order("checked_at DESC, id DESC").
		First(&result).Error
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func DeleteChannelMonitorResultsBefore(cutoff int64) (int64, error) {
	result := DB.Where("checked_at < ?", cutoff).Delete(&ChannelMonitorResult{})
	return result.RowsAffected, result.Error
}

// ListRecentMediaTasks returns terminal tasks for one channel/model without
// reading task payloads or private data. Model names are matched at all three
// persisted naming boundaries so public aliases and upstream names both work.
func ListRecentMediaTasks(channelID int, modelName string, since int64, limit int) ([]*Task, error) {
	modelName = strings.TrimSpace(modelName)
	if channelID <= 0 || modelName == "" {
		return []*Task{}, nil
	}
	if limit <= 0 || limit > 200 {
		limit = 100
	}

	var modelPredicate string
	switch {
	case common.UsingPostgreSQL:
		modelPredicate = `(COALESCE(properties->>'client_model_name', '') = ? OR COALESCE(properties->>'origin_model_name', '') = ? OR COALESCE(properties->>'upstream_model_name', '') = ?)`
	case common.UsingMySQL:
		modelPredicate = `(COALESCE(JSON_UNQUOTE(JSON_EXTRACT(properties, '$.client_model_name')), '') = ? OR COALESCE(JSON_UNQUOTE(JSON_EXTRACT(properties, '$.origin_model_name')), '') = ? OR COALESCE(JSON_UNQUOTE(JSON_EXTRACT(properties, '$.upstream_model_name')), '') = ?)`
	default:
		modelPredicate = `(COALESCE(json_extract(properties, '$.client_model_name'), '') = ? OR COALESCE(json_extract(properties, '$.origin_model_name'), '') = ? OR COALESCE(json_extract(properties, '$.upstream_model_name'), '') = ?)`
	}

	var tasks []*Task
	err := DB.Model(&Task{}).
		Select("id", "updated_at", "channel_id", "platform", "action", "status", "fail_reason", "properties").
		Where("channel_id = ?", channelID).
		Where("status IN ?", []TaskStatus{TaskStatusSuccess, TaskStatusFailure}).
		Where("updated_at >= ?", since).
		Where(modelPredicate, modelName, modelName, modelName).
		Order("updated_at DESC, id DESC").
		Limit(limit).
		Find(&tasks).Error
	return tasks, err
}
