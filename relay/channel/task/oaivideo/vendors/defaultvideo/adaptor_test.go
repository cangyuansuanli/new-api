package defaultvideo

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

func TestParseTaskResult_GZFormat(t *testing.T) {
	adaptor := &TaskAdaptor{}

	t.Run("running", func(t *testing.T) {
		result, err := adaptor.ParseTaskResult([]byte(`{"id":"vid1","status":"running","videoUrl":null,"error":null}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Status != model.TaskStatusInProgress {
			t.Fatalf("expected IN_PROGRESS, got %s", result.Status)
		}
	})

	t.Run("succeeded with videoUrl", func(t *testing.T) {
		result, err := adaptor.ParseTaskResult([]byte(`{"id":"vid1","status":"succeeded","videoUrl":"https://example.com/a.mp4","error":null}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Status != model.TaskStatusSuccess {
			t.Fatalf("expected SUCCESS, got %s", result.Status)
		}
		if result.Url != "https://example.com/a.mp4" {
			t.Fatalf("expected video url, got %q", result.Url)
		}
	})

	t.Run("failed with string error", func(t *testing.T) {
		result, err := adaptor.ParseTaskResult([]byte(`{"id":"vid1","status":"failed","videoUrl":null,"error":"content policy violation"}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Status != model.TaskStatusFailure {
			t.Fatalf("expected FAILURE, got %s", result.Status)
		}
		if result.Reason != "content policy violation" {
			t.Fatalf("expected error reason, got %q", result.Reason)
		}
	})

	t.Run("grok moderation without status", func(t *testing.T) {
		body := []byte(`{"code":"Client specified an invalid argument","error":"Generated video rejected by content moderation.","id":"task_upstream","task_id":"task_upstream","model":"grok-image-video"}`)
		result, err := adaptor.ParseTaskResult(body)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Status != model.TaskStatusFailure {
			t.Fatalf("expected FAILURE, got %s", result.Status)
		}
		if result.Reason != "Generated video rejected by content moderation." {
			t.Fatalf("expected moderation reason, got %q", result.Reason)
		}
	})

	t.Run("omni prefers data url over relative video_url", func(t *testing.T) {
		body := []byte(`{
			"id":"task_upstream",
			"status":"completed",
			"video_url":"/v1/videos/vid-4444bf370600/content",
			"data":[{"url":"https://download-2.oaibox.xyz/v1/videos/task_abc/content"}]
		}`)
		result, err := adaptor.ParseTaskResult(body)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Status != model.TaskStatusSuccess {
			t.Fatalf("expected SUCCESS, got %s", result.Status)
		}
		want := "https://download-2.oaibox.xyz/v1/videos/task_abc/content"
		if result.Url != want {
			t.Fatalf("expected %q, got %q", want, result.Url)
		}
	})
}

func TestParseTaskResult_OpenAIFormat(t *testing.T) {
	adaptor := &TaskAdaptor{}

	result, err := adaptor.ParseTaskResult([]byte(`{"id":"vid1","status":"completed","usage":{"seconds":8}}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != model.TaskStatusSuccess {
		t.Fatalf("expected SUCCESS, got %s", result.Status)
	}
	if result.CompletionTokens != 8 {
		t.Fatalf("expected 8 seconds, got %d", result.CompletionTokens)
	}
}

func TestAdjustBillingOnComplete_OAIREGBoxFallbackToRequestedSeconds(t *testing.T) {
	adaptor := &TaskAdaptor{}
	task := &model.Task{
		Quota: 400000,
		Properties: model.Properties{
			OriginModelName: "cy-sd1-seedance-2.0-mini-480p",
		},
		Data: []byte(`{"created_at":1783265344,"status":"completed","video_url":"https://example.com/a.mp4","data":[{"url":"https://example.com/a.mp4"}]}`),
		PrivateData: model.TaskPrivateData{
			BillingContext: &model.TaskBillingContext{
				ModelPrice:      0.2,
				GroupRatio:      1,
				OtherRatios:     map[string]float64{"seconds": 4, "size": 1},
				OriginModelName: "cy-sd1-seedance-2.0-mini-480p",
			},
		},
	}

	got := adaptor.AdjustBillingOnComplete(task, &relaycommon.TaskInfo{Status: model.TaskStatusSuccess})
	want := int(0.2 * float64(common.QuotaPerUnit) * 4)
	if got != want {
		t.Fatalf("got %d want %d", got, want)
	}
}

func TestAdjustBillingOnComplete_PrefersUpstreamUsageSeconds(t *testing.T) {
	adaptor := &TaskAdaptor{}
	task := &model.Task{
		Quota: 200000,
		Data:  []byte(`{"status":"completed","usage":{"seconds":8}}`),
		PrivateData: model.TaskPrivateData{
			BillingContext: &model.TaskBillingContext{
				ModelPrice:  0.2,
				GroupRatio:  1,
				OtherRatios: map[string]float64{"seconds": 4},
			},
		},
	}

	got := adaptor.AdjustBillingOnComplete(task, &relaycommon.TaskInfo{Status: model.TaskStatusSuccess})
	want := int(0.2 * float64(common.QuotaPerUnit) * 8)
	if got != want {
		t.Fatalf("got %d want %d", got, want)
	}
}

func TestUsageSecondsFromTaskData(t *testing.T) {
	tests := []struct {
		name string
		data string
		want int
	}{
		{
			name: "usage object",
			data: `{"status":"completed","usage":{"seconds":6}}`,
			want: 6,
		},
		{
			name: "top level seconds string",
			data: `{"status":"completed","seconds":"5"}`,
			want: 5,
		},
		{
			name: "oairegbox without seconds",
			data: `{"status":"completed","video_url":"https://example.com/a.mp4"}`,
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := usageSecondsFromTaskData([]byte(tt.data)); got != tt.want {
				t.Fatalf("usageSecondsFromTaskData() = %d, want %d", got, tt.want)
			}
		})
	}
}
