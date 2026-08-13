package model

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/require"
)

func newQueuedAudioTask(id string) *Task {
	return &Task{
		TaskID:     id,
		Platform:   constant.TaskPlatformAudio,
		Status:     TaskStatusQueued,
		Properties: Properties{TaskKind: constant.TaskKindAudio},
	}
}

func TestClaimAudioAsyncTaskSingleWinner(t *testing.T) {
	truncateTables(t)
	task := newQueuedAudioTask("task_audio_claim")
	insertTask(t, task)

	claimed, won, err := ClaimAudioAsyncTask(task.TaskID, "node-a", time.Minute)
	require.NoError(t, err)
	require.True(t, won)
	require.Equal(t, "node-a", claimed.LeaseOwner)
	require.Equal(t, 1, claimed.Attempt)

	_, won, err = ClaimAudioAsyncTask(task.TaskID, "node-b", time.Minute)
	require.NoError(t, err)
	require.False(t, won)
}

func TestGetClaimableAudioAsyncTaskIDsExcludesImageTasks(t *testing.T) {
	truncateTables(t)
	audioTask := newQueuedAudioTask("task_audio_ready")
	insertTask(t, audioTask)
	imageTask := newQueuedImageTask("task_image_ready")
	insertTask(t, imageTask)

	ids := GetClaimableAudioAsyncTaskIDs(10, time.Now().Unix())
	require.Equal(t, []string{audioTask.TaskID}, ids)
}

func TestInsertAudioTaskWithAdmissionEnforcesGlobalLimit(t *testing.T) {
	truncateTables(t)
	insertTask(t, newQueuedAudioTask("task_audio_active"))

	candidate := newQueuedAudioTask("task_audio_candidate")
	err := InsertAudioTaskWithAdmission(candidate, 1, 0)
	require.ErrorIs(t, err, ErrAudioTaskQueueFull)
}

func TestClaimImageAsyncTaskRejectsAudioKind(t *testing.T) {
	truncateTables(t)
	task := newQueuedImageTask("task_audio_on_image_platform")
	task.Properties.TaskKind = constant.TaskKindAudio
	insertTask(t, task)

	_, won, err := ClaimImageAsyncTask(task.TaskID, "node-a", time.Minute)
	require.NoError(t, err)
	require.False(t, won)
}
