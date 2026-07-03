package service

import (
	"context"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/dto"
)

func TestImageModelUsesURLRehostOnlyFor4K(t *testing.T) {
	if !ImageModelUsesURLRehost("geek2-gpt-image-2-4k") {
		t.Fatal("expected 4k model to need url rehost")
	}
	if ImageModelUsesURLRehost("Gulie-gpt-image-2") {
		t.Fatal("non-4k model should not need sync url rehost")
	}
}

func TestImageAsyncAcceptsUpstreamURL(t *testing.T) {
	if !ImageAsyncAcceptsUpstreamURL("geek2-gpt-image-2-4k") {
		t.Fatal("expected 4k async to accept upstream url")
	}
	if !ImageAsyncAcceptsUpstreamURL("cy-img1-gpt-image-2") {
		t.Fatal("expected gulie async to accept upstream url")
	}
	if !ImageAsyncAcceptsUpstreamURL("Gulie-gpt-image-2") {
		t.Fatal("expected gulie async to accept upstream url")
	}
	if ImageAsyncAcceptsUpstreamURL("go2api-gpt-image-2-1k") {
		t.Fatal("generic non-gulie model should still require b64_json in async worker")
	}
}

func TestRewriteLoopbackUpstreamImageURL(t *testing.T) {
	got := RewriteLoopbackUpstreamImageURL("https://api.example.com", "http://127.0.0.1:3001/files/a.png")
	if !strings.HasPrefix(got, "https://api.example.com:3001/files/a.png") {
		t.Fatalf("rewrite = %q", got)
	}
	if RewriteLoopbackUpstreamImageURL("", "https://cdn.example.com/a.png") != "https://cdn.example.com/a.png" {
		t.Fatal("empty channel base should keep original url")
	}
}

func TestRehostSyncImageResponseBodySkipsNon4K(t *testing.T) {
	body := []byte(`{"created":1,"data":[{"url":"https://upstream.example/a.png"}]}`)
	out, err := RehostSyncImageResponseBody(context.Background(), 1, "Gulie-gpt-image-2", "https://api.example.com", body)
	if err != nil {
		t.Fatalf("RehostSyncImageResponseBody: %v", err)
	}
	if string(out) != string(body) {
		t.Fatalf("non-4k should passthrough body unchanged")
	}
}

func TestRehostImageDataURLsRequiresR2For4KURL(t *testing.T) {
	images := []dto.ImageData{{Url: "https://upstream.example/a.png"}}
	_, err := RehostImageDataURLs(context.Background(), 1, "task_test", "https://api.example.com", "geek2-gpt-image-2-4k", images)
	if err == nil || !strings.Contains(err.Error(), "R2 not configured") {
		t.Fatalf("expected R2 not configured error, got %v", err)
	}
}
