package service

import (
	"os"
	"strings"
	"testing"
)

func TestAudioURLNeedsRehost(t *testing.T) {
	t.Setenv("R2_ACCOUNT_ID", "acc")
	t.Setenv("R2_ACCESS_KEY_ID", "key")
	t.Setenv("R2_SECRET_ACCESS_KEY", "secret")
	t.Setenv("R2_USER_BUCKET", "user-bucket")
	t.Setenv("R2_USER_PUBLIC_BASE_URL", "https://tmp.cangyuansuanli.cn")

	if !AudioURLNeedsRehost("https://download-2.oaibox.xyz/v1/audio/aud-xxxx/content") {
		t.Fatal("upstream audio url should need rehost")
	}
	if AudioURLNeedsRehost("https://tmp.cangyuansuanli.cn/gen-audio/1/task_x.mp3") {
		t.Fatal("our cdn url should not rehost")
	}
	if AudioURLNeedsRehost("data:audio/mpeg;base64,abc") {
		t.Fatal("data url should not rehost")
	}
}

func TestAudioURLNeedsRehostWithoutR2(t *testing.T) {
	for _, key := range []string{"R2_ACCOUNT_ID", "R2_ACCESS_KEY_ID", "R2_SECRET_ACCESS_KEY", "R2_USER_BUCKET", "R2_USER_PUBLIC_BASE_URL"} {
		os.Unsetenv(key)
	}
	if AudioURLNeedsRehost("https://download-2.oaibox.xyz/v1/audio/aud-xxxx/content") {
		t.Fatal("without R2 config should skip rehost")
	}
}

func TestPatchAudioURLInTaskData(t *testing.T) {
	in := []byte(`{"result_url":"https://download.example.com/a.mp3","data":[{"url":"https://download.example.com/a.mp3"}]}`)
	out, err := patchAudioURLInTaskData(in, "https://tmp.cangyuansuanli.cn/gen-audio/1/task_x.mp3")
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, `"result_url":"https://tmp.cangyuansuanli.cn/gen-audio/1/task_x.mp3"`) {
		t.Fatalf("result_url not patched: %s", s)
	}
	if !strings.Contains(s, `"url":"https://tmp.cangyuansuanli.cn/gen-audio/1/task_x.mp3"`) {
		t.Fatalf("nested url not patched: %s", s)
	}
}

func TestPatchAudioURLInTaskDataEmpty(t *testing.T) {
	out, err := patchAudioURLInTaskData(nil, "https://tmp.cangyuansuanli.cn/gen-audio/1/task_x.mp3")
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, `"result_url":"https://tmp.cangyuansuanli.cn/gen-audio/1/task_x.mp3"`) {
		t.Fatalf("empty input should create result_url: %s", s)
	}
}
