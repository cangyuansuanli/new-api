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

var ErrImageTaskQueueFull = errors.New("image task queue is at capacity")

const imageTaskAdmissionLockKey int64 = 0x696d6167655f7175

const imageTaskDispatchLeaderLockKey int64 = 0x696d675f7363616e

var imageTaskAdmissionLocalMu sync.Mutex
var imageTaskDispatchLeaderLocalMu sync.Mutex

// ImageTaskDispatchLeadership owns the single durable-queue repair scanner.
// PostgreSQL/MySQL leadership is tied to a dedicated database session, so the
// lock is released automatically if the process or connection dies.
type ImageTaskDispatchLeadership struct {
	conn     *sql.Conn
	local    bool
	release  sync.Once
	dbEngine string
}

func TryAcquireImageTaskDispatchLeadership(ctx context.Context) (*ImageTaskDispatchLeadership, bool, error) {
	if !common.UsingPostgreSQL && !common.UsingMySQL {
		if !imageTaskDispatchLeaderLocalMu.TryLock() {
			return nil, false, nil
		}
		return &ImageTaskDispatchLeadership{local: true, dbEngine: "local"}, true, nil
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return nil, false, err
	}
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		return nil, false, err
	}
	leader := &ImageTaskDispatchLeadership{conn: conn}
	acquired := false
	if common.UsingPostgreSQL {
		leader.dbEngine = "postgresql"
		err = conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", imageTaskDispatchLeaderLockKey).Scan(&acquired)
	} else {
		leader.dbEngine = "mysql"
		var won int
		err = conn.QueryRowContext(ctx, "SELECT GET_LOCK(?, 0)", "new_api_image_task_dispatch_leader").Scan(&won)
		acquired = won == 1
	}
	if err != nil || !acquired {
		_ = conn.Close()
		return nil, false, err
	}
	return leader, true, nil
}

func (leadership *ImageTaskDispatchLeadership) Check(ctx context.Context) error {
	if leadership == nil {
		return errors.New("image task dispatch leadership is nil")
	}
	if leadership.local {
		return nil
	}
	return leadership.conn.PingContext(ctx)
}

func (leadership *ImageTaskDispatchLeadership) Release() {
	if leadership == nil {
		return
	}
	leadership.release.Do(func() {
		if leadership.local {
			imageTaskDispatchLeaderLocalMu.Unlock()
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if leadership.dbEngine == "postgresql" {
			var released bool
			_ = leadership.conn.QueryRowContext(ctx, "SELECT pg_advisory_unlock($1)", imageTaskDispatchLeaderLockKey).Scan(&released)
		} else {
			var released int
			_ = leadership.conn.QueryRowContext(ctx, "SELECT RELEASE_LOCK(?)", "new_api_image_task_dispatch_leader").Scan(&released)
		}
		_ = leadership.conn.Close()
	})
}

type ImageTaskStatus struct {
	Status     TaskStatus
	FailReason string
}

func GetImageTaskStatus(userID int, taskID string) (ImageTaskStatus, bool, error) {
	if taskID == "" {
		return ImageTaskStatus{}, false, nil
	}
	var status ImageTaskStatus
	err := DB.Model(&Task{}).
		Select("status", "fail_reason").
		Where("user_id = ? AND task_id = ?", userID, taskID).
		Take(&status).Error
	exists, err := RecordExist(err)
	return status, exists, err
}

// GetClaimableImageAsyncTaskIDs returns durable image jobs that are ready to
// run. Multiple nodes may observe the same ids; ClaimImageAsyncTask is the
// atomic boundary that elects exactly one worker.
func GetClaimableImageAsyncTaskIDs(limit int, now int64) []string {
	return GetClaimableImageAsyncTaskIDsForChannels(limit, now, nil, false)
}

// GetClaimableImageAsyncTaskIDsForChannels returns claimable image tasks from
// one channel lane. include=true selects only channelIDs; include=false
// excludes them. A nil exclusion keeps the historical all-channel behavior.
func GetClaimableImageAsyncTaskIDsForChannels(limit int, now int64, channelIDs []int, include bool) []string {
	if limit <= 0 {
		return nil
	}
	var tasks []*Task
	query := DB.Select("task_id", "user_id", "priority", "properties").
		Where("platform = ?", constant.TaskPlatformImage)
	query = filterImageTaskChannels(query, channelIDs, include)
	err := query.
		Where("status IN ? OR (status = ? AND ((lease_owner != '' AND lease_expires_at < ?) OR (lease_owner = '' AND updated_at < ?)))",
			[]TaskStatus{TaskStatusSubmitted, TaskStatusQueued}, TaskStatusInProgress, now, now-600).
		Order("priority DESC, id ASC").
		Limit(limit * 8).
		Find(&tasks).Error
	if err != nil {
		return nil
	}
	return fairImageTaskIDs(tasks, limit)
}

func fairImageTaskIDs(tasks []*Task, limit int) []string {
	type userQueue struct {
		userID int
		ids    []string
	}
	priorityOrder := make([]int, 0)
	queuesByPriority := make(map[int][]*userQueue)
	queueIndex := make(map[int]map[int]*userQueue)
	for _, task := range tasks {
		if task == nil || task.TaskID == "" || task.Properties.TaskKind != constant.TaskKindImage {
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

func CountActiveImageTasks(userID int) (global int64, perUser int64, err error) {
	return CountActiveImageTasksForChannels(userID, nil, false)
}

// InsertImageTaskWithAdmission serializes the final capacity check and insert.
// PostgreSQL and MySQL share a database-level lock across every API process,
// closing the COUNT-then-INSERT race at the durable database boundary. SQLite
// deployments use an in-process lock; SQLite itself permits only one writer.
func InsertImageTaskWithAdmission(task *Task, globalLimit, perUserLimit int64) error {
	if task == nil {
		return errors.New("image task is nil")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		release, err := lockImageTaskAdmission(tx)
		if err != nil {
			return err
		}
		defer release()

		statuses := []TaskStatus{TaskStatusSubmitted, TaskStatusQueued, TaskStatusInProgress}
		query := tx.Model(&Task{}).
			Where("platform = ? AND status IN ?", constant.TaskPlatformImage, statuses)
		var global int64
		if err := query.Count(&global).Error; err != nil {
			return err
		}
		if globalLimit > 0 && global >= globalLimit {
			return ErrImageTaskQueueFull
		}

		if perUserLimit > 0 {
			var perUser int64
			if err := query.Where("user_id = ?", task.UserId).Count(&perUser).Error; err != nil {
				return err
			}
			if perUser >= perUserLimit {
				return ErrImageTaskQueueFull
			}
		}
		return tx.Create(task).Error
	})
}

func lockImageTaskAdmission(tx *gorm.DB) (func(), error) {
	if common.UsingPostgreSQL {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", imageTaskAdmissionLockKey).Error; err != nil {
			return nil, err
		}
		return func() {}, nil
	}
	if common.UsingMySQL {
		const lockName = "new_api_image_task_admission"
		var acquired int
		if err := tx.Raw("SELECT GET_LOCK(?, 10)", lockName).Scan(&acquired).Error; err != nil {
			return nil, err
		}
		if acquired != 1 {
			return nil, fmt.Errorf("timed out acquiring image task admission lock")
		}
		return func() {
			_ = tx.Exec("SELECT RELEASE_LOCK(?)", lockName).Error
		}, nil
	}

	imageTaskAdmissionLocalMu.Lock()
	return imageTaskAdmissionLocalMu.Unlock, nil
}

// CountActiveImageTasksForChannels counts active tasks inside one channel
// lane. It mirrors GetClaimableImageAsyncTaskIDsForChannels so admission and
// dispatch always use the same isolation boundary.
func CountActiveImageTasksForChannels(userID int, channelIDs []int, include bool) (global int64, perUser int64, err error) {
	statuses := []TaskStatus{TaskStatusSubmitted, TaskStatusQueued, TaskStatusInProgress}
	query := DB.Model(&Task{}).
		Where("platform = ? AND status IN ?", constant.TaskPlatformImage, statuses)
	query = filterImageTaskChannels(query, channelIDs, include)
	if err = query.Count(&global).Error; err != nil {
		return 0, 0, err
	}
	if userID > 0 {
		err = query.Where("user_id = ?", userID).Count(&perUser).Error
	}
	return global, perUser, err
}

// ClaimImageAsyncTask atomically leases an image job to one worker node. A
// stale IN_PROGRESS task may be reclaimed after its lease expires.
func ClaimImageAsyncTask(taskID, owner string, leaseDuration time.Duration) (*Task, bool, error) {
	return ClaimImageAsyncTaskForChannels(taskID, owner, leaseDuration, nil, false)
}

// ClaimImageAsyncTaskForChannels atomically claims a task only when it belongs
// to the caller's channel lane. This guards against stale Redis notifications
// crossing lanes during rolling deployments.
func ClaimImageAsyncTaskForChannels(taskID, owner string, leaseDuration time.Duration, channelIDs []int, include bool) (*Task, bool, error) {
	if taskID == "" || owner == "" || leaseDuration <= 0 {
		return nil, false, nil
	}
	now := time.Now().Unix()
	leaseUntil := now + int64(leaseDuration/time.Second)
	query := DB.Model(&Task{}).
		Where("task_id = ?", taskID).
		Where("platform = ?", constant.TaskPlatformImage)
	query = filterImageTaskChannels(query, channelIDs, include)
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
	if task.Properties.TaskKind != constant.TaskKindImage {
		ReleaseImageAsyncTaskLease(taskID, owner)
		return nil, false, nil
	}
	return &task, true, nil
}

func filterImageTaskChannels(query *gorm.DB, channelIDs []int, include bool) *gorm.DB {
	if len(channelIDs) == 0 {
		if include {
			return query.Where("1 = 0")
		}
		return query
	}
	if include {
		return query.Where("channel_id IN ?", channelIDs)
	}
	return query.Where("channel_id NOT IN ?", channelIDs)
}

func UpdateImageTaskWithLease(task *Task, owner string) (bool, error) {
	if task == nil || owner == "" {
		return false, nil
	}
	result := DB.Model(task).
		Where("status = ? AND lease_owner = ?", TaskStatusInProgress, owner).
		Select("*").
		Updates(task)
	return result.RowsAffected > 0, result.Error
}

func RenewImageAsyncTaskLease(taskID, owner string, leaseDuration time.Duration) (bool, error) {
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

func ReleaseImageAsyncTaskLease(taskID, owner string) error {
	if taskID == "" || owner == "" {
		return nil
	}
	return DB.Model(&Task{}).
		Where("task_id = ? AND lease_owner = ?", taskID, owner).
		Updates(map[string]any{"lease_owner": "", "lease_expires_at": 0}).Error
}
