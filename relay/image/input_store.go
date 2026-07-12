package image

import (
	"context"
	"errors"

	"github.com/QuantumNous/new-api/service"
)

func EditSnapshotObjectKeys(snapshot []byte) []string {
	if len(snapshot) == 0 {
		return nil
	}
	decoded, err := DecodeRequestSnapshot(snapshot, "/v1/images/edits")
	if err != nil || decoded.Kind != RequestSnapshotEditMultipart || decoded.Multipart == nil {
		return nil
	}
	objectKeys := make([]string, 0, len(decoded.Multipart.Files))
	for _, file := range decoded.Multipart.Files {
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
