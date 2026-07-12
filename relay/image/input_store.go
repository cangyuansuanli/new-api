package image

import (
	"context"
	"errors"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
)

func EditSnapshotObjectKeys(snapshot []byte) []string {
	if len(snapshot) == 0 {
		return nil
	}
	var payload EditPayload
	if err := common.Unmarshal(snapshot, &payload); err != nil {
		return nil
	}
	objectKeys := make([]string, 0, len(payload.Files))
	for _, file := range payload.Files {
		if file.ObjectKey != "" {
			objectKeys = append(objectKeys, file.ObjectKey)
		}
	}
	return objectKeys
}

func CleanupEditSnapshotInputs(snapshot []byte) error {
	var cleanupErrors []error
	for _, objectKey := range EditSnapshotObjectKeys(snapshot) {
		if err := service.DeleteImageTaskInput(context.Background(), objectKey); err != nil {
			cleanupErrors = append(cleanupErrors, err)
		}
	}
	return errors.Join(cleanupErrors...)
}
