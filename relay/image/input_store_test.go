package image

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestEditSnapshotObjectKeysExcludesLegacyInlineData(t *testing.T) {
	snapshot, err := common.Marshal(EditPayload{Files: []EditFile{
		{ObjectKey: "image-task-inputs/1/task/0.png"},
		{Data: []byte("legacy")},
	}})
	require.NoError(t, err)
	require.Equal(t, []string{"image-task-inputs/1/task/0.png"}, EditSnapshotObjectKeys(snapshot))
}
