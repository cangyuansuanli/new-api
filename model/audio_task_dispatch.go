package model

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"gorm.io/gorm"
)

var ErrAudioTaskQueueFull = errors.New("audio task queue is at capacity")

const audioTaskAdmissionLockKey int64 = 0x617564696f5f7175

const audioTaskDispatchLeaderLockKey int64 = 0x617564696f7363

var audioTaskAdmissionLocalMu sync.Mutex
var audioTaskDispatchLeaderLocalMu sync.Mutex

// AudioTaskDispatchLeadership owns the single durable audio-queue repair scanner.
type AudioTaskDispatchLeadership struct {
	conn     *sql.Conn
	local    bool
	release  sync.Once
	dbEngine string
}

func TryAcquireAudioTaskDispatchLeadership(ctx context.Context) (*AudioTaskDispatchLeadership, bool, error) {
	if !common.UsingPostgreSQL && !common.UsingMySQL {
		if !audioTaskDispatchLeaderLocalMu.TryLock() {
			return nil, false, nil
		}
		return &AudioTaskDispatchLeadership{local: true, dbEngine: "local"}, true, nil
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return nil, false, err
	}
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		return nil, false, err
	}
	leader := &AudioTaskDispatchLeadership{conn: conn}
	acquired := false
	if common.UsingPostgreSQL {
		leader.dbEngine = "postgresql"
		err = conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", audioTaskDispatchLeaderLockKey).Scan(&acquired)
	} else {
		leader.dbEngine = "mysql"
		var won int
		err = conn.QueryRowContext(ctx, "SELECT GET_LOCK(?, 0)", "new_api_audio_task_dispatch_leader").Scan(&won)
		acquired = won == 1
	}
	if err != nil || !acquired {
		_ = conn.Close()
		return nil, false, err
	}
	return leader, true, nil
}

func (leadership *AudioTaskDispatchLeadership) Check(ctx context.Context) error {
	if leadership == nil {
		return errors.New("audio task dispatch leadership is nil")
	}
	if leadership.local {
		return nil
	}
	return leadership.conn.PingContext(ctx)
}

func (leadership *AudioTaskDispatchLeadership) Release() {
	if leadership == nil {
		return
	}
	leadership.release.Do(func() {
		if leadership.local {
			audioTaskDispatchLeaderLocalMu.Unlock()
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if leadership.dbEngine == "postgresql" {
			var released bool
			_ = leadership.conn.QueryRowContext(ctx, "SELECT pg_advisory_unlock($1)", audioTaskDispatchLeaderLockKey).Scan(&released)
		} else {
			var released int
			_ = leadership.conn.QueryRowContext(ctx, "SELECT RELEASE_LOCK(?)", "new_api_audio_task_dispatch_leader").Scan(&released)
		}
		_ = leadership.conn.Close()
	})
}

func isAudioAsyncTask(task *Task) bool {
	if task == nil || task.Platform != constant.TaskPlatformAudio {
		return false
	}
	kind := task.Properties.TaskKind
	return kind == "" || kind == constant.TaskKindAudio
}

func audioAsyncPlatformQuery(query *gorm.DB) *gorm.DB {
	return query.Where("platform = ?", constant.TaskPlatformAudio)
}

// GetClaimableAudioAsyncTaskIDs returns durable audio jobs ready to run.
func GetClaimableAudioAsyncTaskIDs(limit int, now int64) []string {
	if limit <= 0 {
		return nil
	}
	var tasks []*Task
	query := DB.Select("task_id", "user_id", "priority", "platform", "properties")
	query = audioAsyncPlatformQuery(query)
	err := query.
		Where("status IN ? OR (status = ? AND ((lease_owner != '' AND lease_expires_at < ?) OR (lease_owner = '' AND updated_at < ?)))",
			[]TaskStatus{TaskStatusSubmitted, TaskStatusQueued}, TaskStatusInProgress, now, now-600).
		Order("priority DESC, id ASC").
		Limit(limit * 8).
		Find(&tasks).Error
	if err != nil {
		return nil
	}
	return fairAudioTaskIDs(tasks, limit)
}

func fairAudioTaskIDs(tasks []*Task, limit int) []string {
	type userQueue struct {
		userID int
		ids    []string
	}
	priorityOrder := make([]int, 0)
	queuesByPriority := make(map[int][]*userQueue)
	queueIndex := make(map[int]map[int]*userQueue)
	for _, task := range tasks {
		if task == nil || task.TaskID == "" || !isAudioAsyncTask(task) {
			continue
		}
		if _, exists := queuesByPriority[task.Priority]; !exists {
			priorityOrder = append(priorityOrder, task.Priority)
			queueIndex[task.Priority] = make(map[int]*userQueue)
		}
		queue := queueIndex[task.Priority][task.UserId]
		if queue == nil {
			queue = &userQueue{userID: task.UserId}
			queueIndex[task.Priority][task.UserId] = queue
			queuesByPriority[task.Priority] = append(queuesByPriority[task.Priority], queue)
		}
		queue.ids = append(queue.ids, task.TaskID)
	}

	ids := make([]string, 0, limit)
	for _, priority := range priorityOrder {
		queues := queuesByPriority[priority]
		for round := 0; len(ids) < limit; round++ {
			added := false
			for _, queue := range queues {
				if round >= len(queue.ids) {
					continue
				}
				ids = append(ids, queue.ids[round])
				added = true
				if len(ids) >= limit {
					break
				}
			}
			if !added {
				break
			}
		}
	}
	return ids
}

func CountActiveAudioTasks(userID int) (global int64, perUser int64, err error) {
	statuses := []TaskStatus{TaskStatusSubmitted, TaskStatusQueued, TaskStatusInProgress}
	query := DB.Model(&Task{}).
		Where("platform = ? AND status IN ?", constant.TaskPlatformAudio, statuses)
	if err = query.Count(&global).Error; err != nil {
		return 0, 0, err
	}
	if userID > 0 {
		err = query.Where("user_id = ?", userID).Count(&perUser).Error
	}
	return global, perUser, err
}

// InsertAudioTaskWithAdmission serializes the final capacity check and insert.
func InsertAudioTaskWithAdmission(task *Task, globalLimit, perUserLimit int64) error {
	if task == nil {
		return errors.New("audio task is nil")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		release, err := lockAudioTaskAdmission(tx)
		if err != nil {
			return err
		}
		defer release()

		statuses := []TaskStatus{TaskStatusSubmitted, TaskStatusQueued, TaskStatusInProgress}
		query := tx.Model(&Task{}).Where("platform = ? AND status IN ?", constant.TaskPlatformAudio, statuses)
		var global int64
		if err := query.Count(&global).Error; err != nil {
			return err
		}
		if globalLimit > 0 && global >= globalLimit {
			return ErrAudioTaskQueueFull
		}

		if perUserLimit > 0 {
			var perUser int64
			if err := query.Where("user_id = ?", task.UserId).Count(&perUser).Error; err != nil {
				return err
			}
			if perUser >= perUserLimit {
				return ErrAudioTaskQueueFull
			}
		}
		return tx.Create(task).Error
	})
}

func lockAudioTaskAdmission(tx *gorm.DB) (func(), error) {
	if common.UsingPostgreSQL {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", audioTaskAdmissionLockKey).Error; err != nil {
			return nil, err
		}
		return func() {}, nil
	}
	if common.UsingMySQL {
		const lockName = "new_api_audio_task_admission"
		var acquired int
		if err := tx.Raw("SELECT GET_LOCK(?, 10)", lockName).Scan(&acquired).Error; err != nil {
			return nil, err
		}
		if acquired != 1 {
			return nil, fmt.Errorf("timed out acquiring audio task admission lock")
		}
		return func() {
			_ = tx.Exec("SELECT RELEASE_LOCK(?)", lockName).Error
		}, nil
	}

	audioTaskAdmissionLocalMu.Lock()
	return audioTaskAdmissionLocalMu.Unlock, nil
}

// ClaimAudioAsyncTask atomically leases an audio job to one worker node.
func ClaimAudioAsyncTask(taskID, owner string, leaseDuration time.Duration) (*Task, bool, error) {
	if taskID == "" || owner == "" || leaseDuration <= 0 {
		return nil, false, nil
	}
	now := time.Now().Unix()
	leaseUntil := now + int64(leaseDuration/time.Second)
	query := DB.Model(&Task{}).Where("task_id = ?", taskID)
	query = audioAsyncPlatformQuery(query)
	result := query.
		Where("status IN ? OR (status = ? AND ((lease_owner != '' AND lease_expires_at < ?) OR (lease_owner = '' AND updated_at < ?)))",
			[]TaskStatus{TaskStatusSubmitted, TaskStatusQueued}, TaskStatusInProgress, now, now-600).
		Updates(map[string]any{
			"status":           TaskStatusInProgress,
			"progress":         "30%",
			"start_time":       gorm.Expr("CASE WHEN start_time = 0 THEN ? ELSE start_time END", now),
			"lease_owner":      owner,
			"lease_expires_at": leaseUntil,
			"attempt":          gorm.Expr("attempt + ?", 1),
			"updated_at":       now,
		})
	if result.Error != nil {
		return nil, false, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, false, nil
	}
	var task Task
	if err := DB.Where("task_id = ? AND lease_owner = ?", taskID, owner).First(&task).Error; err != nil {
		return nil, false, err
	}
	if !isAudioAsyncTask(&task) {
		ReleaseAudioAsyncTaskLease(taskID, owner)
		return nil, false, nil
	}
	return &task, true, nil
}

func UpdateAudioTaskWithLease(task *Task, owner string) (bool, error) {
	if task == nil || owner == "" {
		return false, nil
	}
	result := DB.Model(task).
		Where("status = ? AND lease_owner = ?", TaskStatusInProgress, owner).
		Select("*").
		Updates(task)
	return result.RowsAffected > 0, result.Error
}

func RenewAudioAsyncTaskLease(taskID, owner string, leaseDuration time.Duration) (bool, error) {
	if taskID == "" || owner == "" || leaseDuration <= 0 {
		return false, nil
	}
	now := time.Now().Unix()
	result := DB.Model(&Task{}).
		Where("task_id = ? AND status = ? AND lease_owner = ?", taskID, TaskStatusInProgress, owner).
		Updates(map[string]any{
			"lease_expires_at": now + int64(leaseDuration/time.Second),
			"updated_at":       now,
		})
	return result.RowsAffected > 0, result.Error
}

func ReleaseAudioAsyncTaskLease(taskID, owner string) error {
	if taskID == "" || owner == "" {
		return nil
	}
	return DB.Model(&Task{}).
		Where("task_id = ? AND lease_owner = ?", taskID, owner).
		Updates(map[string]any{"lease_owner": "", "lease_expires_at": 0}).Error
}

// GetPendingAudioAsyncTasks returns in-flight audio async tasks with request snapshots.
func GetPendingAudioAsyncTasks(limit int) []*Task {
	if limit <= 0 {
		return nil
	}
	var all []*Task
	query := DB.Where("progress != ?", "100%").
		Where("status != ?", TaskStatusFailure).
		Where("status != ?", TaskStatusSuccess)
	query = audioAsyncPlatformQuery(query)
	if err := query.Limit(limit * 8).Order("id").Find(&all).Error; err != nil {
		return nil
	}
	if len(all) == 0 {
		return nil
	}
	out := make([]*Task, 0, limit)
	for _, task := range all {
		if task == nil || !isAudioAsyncTask(task) {
			continue
		}
		out = append(out, task)
		if len(out) >= limit {
			break
		}
	}
	return out
}
