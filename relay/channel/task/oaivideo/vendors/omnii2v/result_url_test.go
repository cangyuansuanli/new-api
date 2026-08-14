package omnii2v

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	oaivideo "github.com/QuantumNous/new-api/relay/channel/task/oaivideo/shared"
)

func TestExtractOmniI2VVideoURL(t *testing.T) {
	t.Run("prefers absolute vid video_url over task data url", func(t *testing.T) {
		res, err := oaivideo.ParseResponseTask([]byte(`{
			"status":"completed",
			"video_url":"https://download-2.oaibox.xyz/v1/videos/vid-4444bf370600/content",
			"data":[{"url":"https://download-2.oaibox.xyz/v1/videos/task_abc/content"}]
		}`))
		if err != nil {
			t.Fatalf("ParseResponseTask: %v", err)
		}
		want := "https://download-2.oaibox.xyz/v1/videos/vid-4444bf370600/content"
		if got := extractOmniI2VVideoURL(res); got != want {
			t.Fatalf("expected %q, got %q", want, got)
		}
	})

	t.Run("prefers data url when video_url is relative", func(t *testing.T) {
		res, err := oaivideo.ParseResponseTask([]byte(`{
			"status":"completed",
			"video_url":"/v1/videos/vid-4444bf370600/content",
			"data":[{"url":"https://download-2.oaibox.xyz/v1/videos/task_abc/content"}]
		}`))
		if err != nil {
			t.Fatalf("ParseResponseTask: %v", err)
		}
		want := "https://download-2.oaibox.xyz/v1/videos/task_abc/content"
		if got := extractOmniI2VVideoURL(res); got != want {
			t.Fatalf("expected %q, got %q", want, got)
		}
	})

	t.Run("keeps task data url when no vid video_url", func(t *testing.T) {
		res, err := oaivideo.ParseResponseTask([]byte(`{
			"status":"completed",
			"data":[{"url":"https://download-2.oaibox.xyz/v1/videos/task_abc/content"}]
		}`))
		if err != nil {
			t.Fatalf("ParseResponseTask: %v", err)
		}
		want := "https://download-2.oaibox.xyz/v1/videos/task_abc/content"
		if got := extractOmniI2VVideoURL(res); got != want {
			t.Fatalf("expected %q, got %q", want, got)
		}
	})
}

func TestParseTaskResult_OmniI2VVideoURL(t *testing.T) {
	adaptor := &TaskAdaptor{}
	body := []byte(`{
		"id":"task_upstream",
		"status":"completed",
		"video_url":"https://download-2.oaibox.xyz/v1/videos/vid-ready123/content",
		"data":[{"url":"https://download-2.oaibox.xyz/v1/videos/task_abc/content"}]
	}`)
	result, err := adaptor.ParseTaskResult(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != model.TaskStatusSuccess {
		t.Fatalf("expected SUCCESS, got %s", result.Status)
	}
	want := "https://download-2.oaibox.xyz/v1/videos/vid-ready123/content"
	if result.Url != want {
		t.Fatalf("expected %q, got %q", want, result.Url)
	}
}
