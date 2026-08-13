package common

import "testing"

func TestInferReferenceMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		req        TaskSubmitReq
		explicit   string
		allowMedia bool
		want       string
	}{
		{
			name:       "explicit mode preserved",
			req:        TaskSubmitReq{Images: []string{"a"}},
			explicit:   "media",
			allowMedia: true,
			want:       "media",
		},
		{
			name:       "first last frame urls",
			req:        TaskSubmitReq{FirstImageUrl: "a", LastImageUrl: "b"},
			allowMedia: true,
			want:       "frame",
		},
		{
			name:       "media refs",
			req:        TaskSubmitReq{Images: []string{"a"}, ReferenceVideos: []string{"v"}},
			allowMedia: true,
			want:       "media",
		},
		{
			name:       "two ordinary images stay media references",
			req:        TaskSubmitReq{Images: []string{"a", "b"}},
			allowMedia: true,
			want:       "media",
		},
		{
			name:       "single image asset",
			req:        TaskSubmitReq{Images: []string{"a"}},
			allowMedia: false,
			want:       "asset",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := InferReferenceMode(tc.req, tc.explicit, tc.allowMedia); got != tc.want {
				t.Fatalf("InferReferenceMode() = %q, want %q", got, tc.want)
			}
		})
	}
}
