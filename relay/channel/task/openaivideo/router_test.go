package openaivideo

import (
	"testing"

	"github.com/QuantumNous/new-api/relay/channel/task/manjusora"
	"github.com/QuantumNous/new-api/relay/channel/task/seedance"
	"github.com/QuantumNous/new-api/relay/channel/task/sora"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

func TestRouterAdaptor_PickDelegate(t *testing.T) {
	r := NewRouterAdaptor().(*RouterAdaptor)

	cases := []struct {
		origin   string
		upstream string
		want     string
	}{
		{"manju-openai-sora2", "sora2", "manju"},
		{"cy-sd4-seedance-2.0", "seedance-2.0", "seedance"},
		{"cy-sd2-seedance-2.0", "manxue-2.0", "seedance"},
		{"sora-2", "sora-2", "sora"},
		{"cy-sd1-seedance-2.0-mini-480p", "Seedance-2.0-480p", "sora"},
	}
	for _, tc := range cases {
		info := &relaycommon.RelayInfo{
			OriginModelName:   tc.origin,
			UpstreamModelName: tc.upstream,
		}
		d := r.pick(info)
		switch tc.want {
		case "manju":
			if _, ok := d.(*manjusora.TaskAdaptor); !ok {
				t.Fatalf("%s: expected manju adaptor", tc.origin)
			}
		case "seedance":
			if _, ok := d.(*seedance.TaskAdaptor); !ok {
				t.Fatalf("%s: expected seedance adaptor", tc.origin)
			}
		default:
			if _, ok := d.(*sora.TaskAdaptor); !ok {
				t.Fatalf("%s: expected sora adaptor", tc.origin)
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
