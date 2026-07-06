package openai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

func TestIsManjuBananaOriginModel(t *testing.T) {
	if !IsManjuBananaOriginModel("manju-gemini-banana-2.0-1/2k") {
		t.Fatal("expected manju model")
	}
	if IsManjuBananaOriginModel("byte-gemini-banana-2.0") {
		t.Fatal("byte model should not match")
	}
}

func TestAdaptManjuBananaChatCompletionResponseAsyncPoll(t *testing.T) {
	var polls int
	imageBody := []byte("fakejpeg")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/files/test.jpg"):
			w.Header().Set("Content-Type", "image/jpeg")
			_, _ = w.Write(imageBody)
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/api/tasks/"):
			polls++
			if polls < 2 {
				_, _ = w.Write([]byte(`{"status":"running","task_id":"gemini-img-test"}`))
				return
			}
			_, _ = w.Write([]byte(`{"status":"succeeded","result_url":"` + srv.URL + `/files/test.jpg"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	createBody := []byte(`{
		"status":"running",
		"task_id":"gemini-img-test",
		"poll_url":"` + srv.URL + `/api/tasks/gemini-img-test",
		"choices":[{"message":{"role":"assistant","content":""}}]
	}`)
	info := &relaycommon.RelayInfo{
		OriginModelName: "manju-gemini-banana-pro-4k",
		ChannelMeta: relaycommon.ChannelMeta{
			ApiKey:         "sk-test",
			ChannelBaseUrl: srv.URL,
		},
	}

	out, adaptErr := AdaptManjuBananaChatCompletionResponse(context.Background(), info, createBody)
	if adaptErr != nil {
		t.Fatalf("adapt: %v", adaptErr)
	}
	content := string(out)
	if !strings.Contains(content, "data:image/jpeg;base64,") {
		t.Fatalf("expected data uri markdown, got %s", content)
	}
	if strings.Contains(content, "poll_url") || strings.Contains(content, "task_id") {
		t.Fatalf("upstream task fields should be stripped: %s", content)
	}
	if polls < 2 {
		t.Fatalf("expected polling, got %d", polls)
	}
}

func TestAdaptManjuBananaChatCompletionResponseURLMarkdown(t *testing.T) {
	imageBody := []byte("pngbytes")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(imageBody)
	}))
	defer srv.Close()

	body := []byte(`{
		"choices":[{"message":{"role":"assistant","content":"![Generated Image](` + srv.URL + `/img.png)"}}],
		"task_id":"x",
		"poll_url":"` + srv.URL + `/poll"
	}`)
	info := &relaycommon.RelayInfo{OriginModelName: "manju-gemini-banana-2.0-1/2k"}
	out, err := AdaptManjuBananaChatCompletionResponse(context.Background(), info, body)
	if err != nil {
		t.Fatalf("adapt: %v", err)
	}
	if !strings.Contains(string(out), "data:image/png;base64,") {
		t.Fatalf("expected png data uri, got %s", string(out))
	}
}
