package model

import (
	"time"

	"github.com/QuantumNous/new-api/constant"
	"gorm.io/gorm"
)

// GetClaimableImageAsyncTaskIDs returns durable image jobs that are ready to
// run. Multiple nodes may observe the same ids; ClaimImageAsyncTask is the
// atomic boundary that elects exactly one worker.
func GetClaimableImageAsyncTaskIDs(limit int, now int64) []string {
	if limit <= 0 {
		return nil
	}
	var tasks []*Task
	err := DB.Select("task_id", "user_id", "priority", "properties").
		Where("platform = ?", constant.TaskPlatformImage).
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
	statuses := []TaskStatus{TaskStatusSubmitted, TaskStatusQueued, TaskStatusInProgress}
	query := DB.Model(&Task{}).
		Where("platform = ? AND status IN ?", constant.TaskPlatformImage, statuses)
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
	if taskID == "" || owner == "" || leaseDuration <= 0 {
		return nil, false, nil
	}
	now := time.Now().Unix()
	leaseUntil := now + int64(leaseDuration/time.Second)
	result := DB.Model(&Task{}).
		Where("task_id = ?", taskID).
		Where("platform = ?", constant.TaskPlatformImage).
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
