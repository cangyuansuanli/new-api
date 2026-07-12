package service

import (
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

type taskAwarePollingAdaptorStub struct {
	result *relaycommon.TaskInfo
}

func (s *taskAwarePollingAdaptorStub) Init(*relaycommon.RelayInfo) {}

func (s *taskAwarePollingAdaptorStub) FetchTask(string, string, map[string]any, string) (*http.Response, error) {
	return nil, nil
}

func (s *taskAwarePollingAdaptorStub) ParseTaskResult([]byte) (*relaycommon.TaskInfo, error) {
	return &relaycommon.TaskInfo{}, nil
}

func (s *taskAwarePollingAdaptorStub) ParseTaskResultForTask(*model.Task, []byte) (*relaycommon.TaskInfo, error) {
	return s.result, nil
}

func (s *taskAwarePollingAdaptorStub) AdjustBillingOnComplete(*model.Task, *relaycommon.TaskInfo) int {
	return 0
}

func TestParseVideoPollingResultPrefersVendorNormalization(t *testing.T) {
	wantURL := "https://vidgen.x.ai/video.mp4"
	adaptor := &taskAwarePollingAdaptorStub{result: &relaycommon.TaskInfo{
		Status: model.TaskStatusSuccess,
		Url:    wantURL,
	}}
	task := &model.Task{TaskID: "task_public"}
	body := []byte(`{"code":"success","data":{"task_id":"task_upstream","status":"SUCCESS","result_url":"https://vidgen.x.ai/video.mp4"}}`)

	result, err := parseVideoPollingResult(adaptor, task, body)
	if err != nil {
		t.Fatalf("parseVideoPollingResult: %v", err)
	}
	if result.Status != model.TaskStatusSuccess || result.Url != wantURL {
		t.Fatalf("vendor-normalized result was not preferred: %#v", result)
	}
}

func TestParseVideoPollingResultFallsBackToGenericEnvelope(t *testing.T) {
	adaptor := &taskAwarePollingAdaptorStub{result: &relaycommon.TaskInfo{}}
	task := &model.Task{TaskID: "task_public"}
	body := []byte(`{"code":"success","data":{"task_id":"task_upstream","status":"SUCCESS","progress":"100%","result_url":"https://example.com/video.mp4"}}`)

	result, err := parseVideoPollingResult(adaptor, task, body)
	if err != nil {
		t.Fatalf("parseVideoPollingResult: %v", err)
	}
	if result.Status != model.TaskStatusSuccess || result.TaskID != "task_upstream" || result.Url != "https://example.com/video.mp4" {
		t.Fatalf("generic fallback failed: %#v", result)
	}
}
