package service

import (
	"os"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/setting/system_setting"
)

func TestResolveStoredVideoResultURLRecoversSelfReferentialProxy(t *testing.T) {
	previous := system_setting.ServerAddress
	system_setting.ServerAddress = "https://ai.cangyuansuanli.cn"
	t.Cleanup(func() { system_setting.ServerAddress = previous })

	stored := "https://ai.cangyuansuanli.cn/v1/videos/task_public/content"
	body := []byte(`{"code":"success","data":{"task_id":"task_upstream","result_url":"https://vidgen.x.ai/video.mp4","data":{"video":{"url":"https://vidgen.x.ai/video.mp4"}}}}`)
	if got := ResolveStoredVideoResultURL(stored, body); got != "https://vidgen.x.ai/video.mp4" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveStoredVideoResultURLKeepsValidStoredURL(t *testing.T) {
	stored := "https://tmp.cangyuansuanli.cn/gen-videos/1/task_public.mp4"
	body := []byte(`{"result_url":"https://vidgen.x.ai/video.mp4"}`)
	if got := ResolveStoredVideoResultURL(stored, body); got != stored {
		t.Fatalf("got %q", got)
	}
}

func TestVideoURLNeedsRehost(t *testing.T) {
	t.Setenv("R2_ACCOUNT_ID", "acc")
	t.Setenv("R2_ACCESS_KEY_ID", "key")
	t.Setenv("R2_SECRET_ACCESS_KEY", "secret")
	t.Setenv("R2_USER_BUCKET", "user-bucket")
	t.Setenv("R2_USER_PUBLIC_BASE_URL", "https://tmp.cangyuansuanli.cn")

	if !VideoURLNeedsRehost("https://cdn.leonardo.ai/users/u/videos/a.mp4") {
		t.Fatal("leonardo url should need rehost")
	}
	if VideoURLNeedsRehost("https://tmp.cangyuansuanli.cn/gen-videos/1/task_x.mp4") {
		t.Fatal("our cdn url should not rehost")
	}
	if VideoURLNeedsRehost("data:video/mp4;base64,abc") {
		t.Fatal("data url should not rehost")
	}
	if VideoURLNeedsRehost("") {
		t.Fatal("empty should not rehost")
	}
}

func TestVideoURLNeedsRehostWithoutR2(t *testing.T) {
	for _, key := range []string{"R2_ACCOUNT_ID", "R2_ACCESS_KEY_ID", "R2_SECRET_ACCESS_KEY", "R2_USER_BUCKET", "R2_USER_PUBLIC_BASE_URL"} {
		os.Unsetenv(key)
	}
	if VideoURLNeedsRehost("https://cdn.leonardo.ai/users/u/videos/a.mp4") {
		t.Fatal("without R2 config should skip rehost")
	}
}

func TestPatchVideoURLInTaskData(t *testing.T) {
	in := []byte(`{"id":"video_1","video_url":"https://cdn.leonardo.ai/a.mp4","data":[{"url":"https://cdn.leonardo.ai/a.mp4","video_url":"https://cdn.leonardo.ai/a.mp4"}]}`)
	out, err := patchVideoURLInTaskData(in, "https://tmp.cangyuansuanli.cn/gen-videos/1/task_x.mp4")
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, `"video_url":"https://tmp.cangyuansuanli.cn/gen-videos/1/task_x.mp4"`) {
		t.Fatalf("video_url not patched: %s", s)
	}
	if !strings.Contains(s, `"url":"https://tmp.cangyuansuanli.cn/gen-videos/1/task_x.mp4"`) {
		t.Fatalf("nested url not patched: %s", s)
	}
}

func TestPatchVideoUsageSecondsInTaskData(t *testing.T) {
	in := []byte(`{"id":"video_1","status":"completed","video_url":"https://example.com/a.mp4"}`)
	out, err := patchVideoUsageSecondsInTaskData(in, 15)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `"usage":{"seconds":15}`) {
		t.Fatalf("usage seconds not patched: %s", out)
	}
}

func TestPatchVideoEnvelopeKeepsDataObject(t *testing.T) {
	in := []byte(`{"code":"success","data":{"task_id":"task_x","status":"SUCCESS","result_url":"https://upstream.example/a.mp4"}}`)
	cdn := "https://tmp.cangyuansuanli.cn/gen-videos/1/task_x.mp4"
	out, err := patchVideoURLInTaskData(in, cdn)
	if err != nil {
		t.Fatal(err)
	}
	out, err = patchVideoUsageSecondsInTaskData(out, 4)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, `"data":{"task_id"`) || !strings.Contains(s, `"result_url":"`+cdn+`"`) {
		t.Fatalf("envelope data was corrupted: %s", s)
	}
	if !strings.Contains(s, `"usage":{"seconds":4}`) {
		t.Fatalf("nested usage seconds not patched: %s", s)
	}
}

func TestBuildGeneratedVideoObjectKey(t *testing.T) {
	if got := buildGeneratedVideoObjectKey(1, "task_abc", ".mp4"); got != "gen-videos/1/task_abc.mp4" {
		t.Fatalf("got=%q", got)
	}
}
