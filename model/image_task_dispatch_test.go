package model

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/require"
)

func newQueuedImageTask(id string) *Task {
	return &Task{
		TaskID:     id,
		Platform:   constant.TaskPlatformImage,
		Status:     TaskStatusQueued,
		Properties: Properties{TaskKind: constant.TaskKindImage},
	}
}

func TestClaimImageAsyncTaskSingleWinner(t *testing.T) {
	truncateTables(t)
	task := newQueuedImageTask("task_image_claim")
	insertTask(t, task)

	claimed, won, err := ClaimImageAsyncTask(task.TaskID, "node-a", time.Minute)
	require.NoError(t, err)
	require.True(t, won)
	require.Equal(t, "node-a", claimed.LeaseOwner)
	require.Equal(t, 1, claimed.Attempt)

	_, won, err = ClaimImageAsyncTask(task.TaskID, "node-b", time.Minute)
	require.NoError(t, err)
	require.False(t, won)
}

func TestClaimImageAsyncTaskReclaimsExpiredLease(t *testing.T) {
	truncateTables(t)
	task := newQueuedImageTask("task_image_reclaim")
	task.Status = TaskStatusInProgress
	task.LeaseOwner = "dead-node"
	task.LeaseExpiresAt = time.Now().Add(-time.Minute).Unix()
	insertTask(t, task)

	claimed, won, err := ClaimImageAsyncTask(task.TaskID, "node-b", time.Minute)
	require.NoError(t, err)
	require.True(t, won)
	require.Equal(t, "node-b", claimed.LeaseOwner)
	require.Equal(t, 1, claimed.Attempt)
}

func TestRenewImageAsyncTaskLeaseRequiresOwner(t *testing.T) {
	truncateTables(t)
	task := newQueuedImageTask("task_image_renew")
	insertTask(t, task)
	claimed, won, err := ClaimImageAsyncTask(task.TaskID, "node-a", time.Second)
	require.NoError(t, err)
	require.True(t, won)

	ok, err := RenewImageAsyncTaskLease(claimed.TaskID, "node-b", time.Minute)
	require.NoError(t, err)
	require.False(t, ok)
	ok, err = RenewImageAsyncTaskLease(claimed.TaskID, "node-a", time.Minute)
	require.NoError(t, err)
	require.True(t, ok)
}

func TestUpdateImageTaskWithLeaseRejectsOldOwner(t *testing.T) {
	truncateTables(t)
	task := newQueuedImageTask("task_image_finish_lease")
	insertTask(t, task)
	claimed, won, err := ClaimImageAsyncTask(task.TaskID, "node-a", time.Minute)
	require.NoError(t, err)
	require.True(t, won)

	claimed.Status = TaskStatusSuccess
	claimed.LeaseOwner = ""
	won, err = UpdateImageTaskWithLease(claimed, "node-b")
	require.NoError(t, err)
	require.False(t, won)
	won, err = UpdateImageTaskWithLease(claimed, "node-a")
	require.NoError(t, err)
	require.True(t, won)
}

func TestGetClaimableImageAsyncTaskIDsExcludesOtherKinds(t *testing.T) {
	truncateTables(t)
	imageTask := newQueuedImageTask("task_image_ready")
	insertTask(t, imageTask)
	videoTask := newQueuedImageTask("task_video_ready")
	videoTask.Properties.TaskKind = "video"
	insertTask(t, videoTask)

	ids := GetClaimableImageAsyncTaskIDs(10, time.Now().Unix())
	require.Equal(t, []string{imageTask.TaskID}, ids)
}

func TestGetClaimableImageAsyncTaskIDsPrioritizesSyncWaiters(t *testing.T) {
	truncateTables(t)
	normal := newQueuedImageTask("task_image_normal")
	insertTask(t, normal)
	priority := newQueuedImageTask("task_image_priority")
	priority.Priority = 100
	insertTask(t, priority)

	ids := GetClaimableImageAsyncTaskIDs(10, time.Now().Unix())
	require.Equal(t, []string{priority.TaskID, normal.TaskID}, ids)
}

func TestImageTaskChannelLaneIsolation(t *testing.T) {
	truncateTables(t)
	regular := newQueuedImageTask("task_image_regular_lane")
	regular.ChannelId = 77
	regular.UserId = 7
	insertTask(t, regular)
	adobe := newQueuedImageTask("task_image_adobe_lane")
	adobe.ChannelId = 75
	adobe.UserId = 7
	insertTask(t, adobe)

	now := time.Now().Unix()
	require.Equal(t, []string{adobe.TaskID}, GetClaimableImageAsyncTaskIDsForChannels(10, now, []int{75}, true))
	require.Equal(t, []string{regular.TaskID}, GetClaimableImageAsyncTaskIDsForChannels(10, now, []int{75}, false))

	global, perUser, err := CountActiveImageTasksForChannels(7, []int{75}, true)
	require.NoError(t, err)
	require.Equal(t, int64(1), global)
	require.Equal(t, int64(1), perUser)

	_, won, err := ClaimImageAsyncTaskForChannels(adobe.TaskID, "regular-worker", time.Minute, []int{75}, false)
	require.NoError(t, err)
	require.False(t, won)
	claimed, won, err := ClaimImageAsyncTaskForChannels(adobe.TaskID, "adobe-worker", time.Minute, []int{75}, true)
	require.NoError(t, err)
	require.True(t, won)
	require.Equal(t, adobe.TaskID, claimed.TaskID)
}

func TestCountActiveImageTasks(t *testing.T) {
	truncateTables(t)
	first := newQueuedImageTask("task_image_count_1")
	first.Platform = constant.TaskPlatformImage
	first.UserId = 7
	insertTask(t, first)
	second := newQueuedImageTask("task_image_count_2")
	second.Platform = constant.TaskPlatformImage
	second.UserId = 8
	insertTask(t, second)

	global, perUser, err := CountActiveImageTasks(7)
	require.NoError(t, err)
	require.Equal(t, int64(2), global)
	require.Equal(t, int64(1), perUser)
}

func TestInsertImageTaskWithAdmissionEnforcesGlobalLimit(t *testing.T) {
	truncateTables(t)
	insertTask(t, newQueuedImageTask("task_image_existing"))
	candidate := newQueuedImageTask("task_image_rejected_global")

	err := InsertImageTaskWithAdmission(candidate, 1, 0)
	require.ErrorIs(t, err, ErrImageTaskQueueFull)
	var count int64
	require.NoError(t, DB.Model(&Task{}).Where("task_id = ?", candidate.TaskID).Count(&count).Error)
	require.Zero(t, count)
}

func TestInsertImageTaskWithAdmissionEnforcesPerUserLimit(t *testing.T) {
	truncateTables(t)
	existing := newQueuedImageTask("task_image_existing_user")
	existing.UserId = 7
	insertTask(t, existing)
	candidate := newQueuedImageTask("task_image_rejected_user")
	candidate.UserId = 7

	err := InsertImageTaskWithAdmission(candidate, 10, 1)
	require.ErrorIs(t, err, ErrImageTaskQueueFull)
}

func TestInsertImageTaskWithAdmissionAcceptsWithinLimits(t *testing.T) {
	truncateTables(t)
	candidate := newQueuedImageTask("task_image_accepted")
	candidate.UserId = 7

	require.NoError(t, InsertImageTaskWithAdmission(candidate, 10, 2))
	var count int64
	require.NoError(t, DB.Model(&Task{}).Where("task_id = ?", candidate.TaskID).Count(&count).Error)
	require.Equal(t, int64(1), count)
}

func TestInsertImageTaskWithAdmissionConcurrentLimit(t *testing.T) {
	truncateTables(t)
	const (
		attempts = 50
		limit    = 10
	)
	var wg sync.WaitGroup
	errs := make(chan error, attempts)
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			task := newQueuedImageTask(fmt.Sprintf("task_image_concurrent_%d", i))
			task.UserId = 7
			errs <- InsertImageTaskWithAdmission(task, limit, limit)
		}(i)
	}
	wg.Wait()
	close(errs)

	accepted := 0
	for err := range errs {
		if err == nil {
			accepted++
			continue
		}
		require.ErrorIs(t, err, ErrImageTaskQueueFull)
	}
	require.Equal(t, limit, accepted)
	var count int64
	require.NoError(t, DB.Model(&Task{}).Where("user_id = ?", 7).Count(&count).Error)
	require.Equal(t, int64(limit), count)
}

func TestImageTaskDispatchLeadershipSingleLocalLeader(t *testing.T) {
	leader, acquired, err := TryAcquireImageTaskDispatchLeadership(context.Background())
	require.NoError(t, err)
	require.True(t, acquired)
	require.NoError(t, leader.Check(context.Background()))

	second, acquired, err := TryAcquireImageTaskDispatchLeadership(context.Background())
	require.NoError(t, err)
	require.False(t, acquired)
	require.Nil(t, second)

	leader.Release()
	replacement, acquired, err := TryAcquireImageTaskDispatchLeadership(context.Background())
	require.NoError(t, err)
	require.True(t, acquired)
	replacement.Release()
}

func TestGetImageTaskStatusLoadsOnlyStatusFields(t *testing.T) {
	truncateTables(t)
	task := newQueuedImageTask("task_image_status")
	task.UserId = 7
	task.Status = TaskStatusFailure
	task.FailReason = "upstream failed"
	insertTask(t, task)

	status, exists, err := GetImageTaskStatus(7, task.TaskID)
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, TaskStatusFailure, status.Status)
	require.Equal(t, "upstream failed", status.FailReason)

	_, exists, err = GetImageTaskStatus(8, task.TaskID)
	require.NoError(t, err)
	require.False(t, exists)
}

func TestFairImageTaskIDsRoundRobinsUsersWithinPriority(t *testing.T) {
	tasks := []*Task{
		{TaskID: "u1-a", UserId: 1, Priority: 0, Properties: Properties{TaskKind: constant.TaskKindImage}},
		{TaskID: "u1-b", UserId: 1, Priority: 0, Properties: Properties{TaskKind: constant.TaskKindImage}},
		{TaskID: "u2-a", UserId: 2, Priority: 0, Properties: Properties{TaskKind: constant.TaskKindImage}},
		{TaskID: "u2-b", UserId: 2, Priority: 0, Properties: Properties{TaskKind: constant.TaskKindImage}},
	}
	require.Equal(t, []string{"u1-a", "u2-a", "u1-b", "u2-b"}, fairImageTaskIDs(tasks, 4))
}
