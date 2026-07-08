package model

import (
	"testing"
)

func TestApplySubmittedStatusFromUpstreamData_Queued(t *testing.T) {
	task := &Task{}
	ApplySubmittedStatusFromUpstreamData(task, []byte(`{"status":"queued","id":"task_x"}`))
	if task.Status != TaskStatusQueued {
		t.Fatalf("expected QUEUED, got %s", task.Status)
	}
	if task.Progress != "20%" {
		t.Fatalf("expected 20%%, got %s", task.Progress)
	}
}

func TestApplySubmittedStatusFromUpstreamData_DefaultSubmitted(t *testing.T) {
	task := &Task{}
	ApplySubmittedStatusFromUpstreamData(task, []byte(`{"id":"task_x"}`))
	if task.Status != TaskStatusSubmitted {
		t.Fatalf("expected SUBMITTED, got %s", task.Status)
	}
}
