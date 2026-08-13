package model

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/require"
)

func TestPendingAudioAsyncTasksKeepRequestSnapshot(t *testing.T) {
	truncateTables(t)

	snapshot := []byte(`{"model":"gemini-music","prompt":"a beat"}`)
	task := &Task{
		UserId:     7,
		TaskID:     "task_pending_audio_snapshot",
		Platform:   constant.TaskPlatformAudio,
		Status:     TaskStatusInProgress,
		Progress:   "30%",
		SubmitTime: time.Now().Unix(),
		Properties: Properties{TaskKind: constant.TaskKindAudio},
		PrivateData: TaskPrivateData{
			RequestSnapshot: snapshot,
		},
		Data: json.RawMessage(`{}`),
	}
	insertTask(t, task)

	pending := GetPendingAudioAsyncTasks(1)
	require.Len(t, pending, 1)
	require.Equal(t, snapshot, pending[0].PrivateData.RequestSnapshot)
}
