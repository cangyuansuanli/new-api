package router

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel/task/oaivideo/registry"
	"github.com/QuantumNous/new-api/relay/channel/task/oaivideo/vendors/chatvideo"
	"github.com/QuantumNous/new-api/relay/channel/task/oaivideo/vendors/defaultvideo"
	"github.com/QuantumNous/new-api/relay/channel/task/oaivideo/vendors/geeknowgrok"
	"github.com/QuantumNous/new-api/relay/channel/task/oaivideo/vendors/grok"
	"github.com/QuantumNous/new-api/relay/channel/task/oaivideo/vendors/manju"
	"github.com/QuantumNous/new-api/relay/channel/task/oaivideo/vendors/sd5"
	"github.com/QuantumNous/new-api/relay/channel/task/oaivideo/vendors/seedance"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

func TestRouterAdaptor_DelegateFor(t *testing.T) {
	r := NewRouterAdaptor().(*RouterAdaptor)

	cases := []struct {
		origin   string
		upstream string
		want     string
	}{
		{"manju-openai-sora2", "sora2", "manju"},
		{"cy-sd4-seedance-2.0", "seedance-2.0", "seedance"},
		{"cy-sd5-seedance-2.0-fast", "cy-sd5-seedance-2.0-fast", "sd5"},
		{"cy-sd2-seedance-2.0", "manxue-2.0", "seedance"},
		{"cy-vid2-sora-2", "cy-vid2-sora-2", "chat"},
		{"cy-gv1-grok-video-1.5", "grok-video-1.5", "grok"},
		{"cy-gv1-grok-video", "grok-imagine-video", "geeknow-grok"},
		{"cy-gv1-grok-video-1.5", "grok-imagine-video-1.5-preview", "geeknow-grok"},
		{"sora-2", "sora-2", "default"},
		{"cy-sd1-seedance-2.0-mini-480p", "Seedance-2.0-480p", "seedance"},
	}
	for _, tc := range cases {
		info := &relaycommon.RelayInfo{
			OriginModelName: tc.origin,
		}
		if tc.upstream != "" {
			info.ChannelMeta = &relaycommon.ChannelMeta{UpstreamModelName: tc.upstream}
		}
		d := r.delegateFor(info)
		switch tc.want {
		case "manju":
			if _, ok := d.(*manju.TaskAdaptor); !ok {
				t.Fatalf("%s: expected manju adaptor", tc.origin)
			}
		case "seedance":
			if _, ok := d.(*seedance.TaskAdaptor); !ok {
				t.Fatalf("%s: expected seedance adaptor", tc.origin)
			}
		case "chat":
			if _, ok := d.(*chatvideo.TaskAdaptor); !ok {
				t.Fatalf("%s: expected chat video adaptor", tc.origin)
			}
		case "grok":
			if _, ok := d.(*grok.TaskAdaptor); !ok {
				t.Fatalf("%s: expected Grok adaptor", tc.origin)
			}
		case "geeknow-grok":
			if _, ok := d.(*geeknowgrok.TaskAdaptor); !ok {
				t.Fatalf("%s: expected Geeknow Grok adaptor", tc.origin)
			}
		case "sd5":
			if _, ok := d.(*sd5.TaskAdaptor); !ok {
				t.Fatalf("%s: expected SD5 adaptor", tc.origin)
			}
		default:
			if _, ok := d.(*defaultvideo.TaskAdaptor); !ok {
				t.Fatalf("%s: expected default adaptor", tc.origin)
			}
		}
	}
}

func TestRouterAdaptor_ParseTaskResult_ManjuBody(t *testing.T) {
	r := NewRouterAdaptor()
	body := []byte(`{"id":"sora2-abc","platform":"sora2","status":"failed","message":"审核失败"}`)
	result, err := r.ParseTaskResult(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Reason != "审核失败" {
		t.Fatalf("expected manju parser, got reason %q", result.Reason)
	}
}

func TestRouterAdaptor_ParseTaskResultForTask_Completed(t *testing.T) {
	r := NewRouterAdaptor().(*RouterAdaptor)
	body := []byte(`{"id":"task_x","status":"completed","progress":100,"video_url":"https://example.com/a.mp4"}`)
	task := &model.Task{
		Properties: model.Properties{OriginModelName: "cy-sd1-seedance-2.0-fast-720p"},
	}
	result, err := r.ParseTaskResultForTask(task, body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != model.TaskStatusSuccess {
		t.Fatalf("expected SUCCESS, got %s", result.Status)
	}
	if result.Url != "https://example.com/a.mp4" {
		t.Fatalf("unexpected url %q", result.Url)
	}
}

func TestRouterAdaptor_NilSafeAdjustBilling(t *testing.T) {
	var r *RouterAdaptor
	if r.AdjustBillingOnComplete(&model.Task{}, &relaycommon.TaskInfo{}) != 0 {
		t.Fatal("nil receiver should return 0")
	}
}

func TestRouterAdaptor_VendorResolveMatchesPick(t *testing.T) {
	origin := "cy-sd1-seedance-2.0-fast-720p"
	if registry.Resolve(origin, "") != registry.VendorSeedance {
		t.Fatal("registry should classify cy-sd1 as seedance")
	}
}
