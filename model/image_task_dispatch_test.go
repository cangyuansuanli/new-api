package model

import (
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

func TestFairImageTaskIDsRoundRobinsUsersWithinPriority(t *testing.T) {
	tasks := []*Task{
		{TaskID: "u1-a", UserId: 1, Priority: 0, Properties: Properties{TaskKind: constant.TaskKindImage}},
		{TaskID: "u1-b", UserId: 1, Priority: 0, Properties: Properties{TaskKind: constant.TaskKindImage}},
		{TaskID: "u2-a", UserId: 2, Priority: 0, Properties: Properties{TaskKind: constant.TaskKindImage}},
		{TaskID: "u2-b", UserId: 2, Priority: 0, Properties: Properties{TaskKind: constant.TaskKindImage}},
	}
	require.Equal(t, []string{"u1-a", "u2-a", "u1-b", "u2-b"}, fairImageTaskIDs(tasks, 4))
}
