package service

import "testing"

func TestHumanizeLeonardoReferenceLimitError(t *testing.T) {
	cases := []struct {
		name          string
		raw           string
		preferChinese bool
		want          string
	}{
		{
			name: "images_zh",
			raw:  "reference images exceed Leonardo limit (5/4)",
			preferChinese: true,
			want: "参考图最多 4 张，当前 5 张，请减少后重试。",
		},
		{
			name: "images_en",
			raw:  "All cookies failed. cookie#3: reference images exceed Leonardo limit (6/4)",
			want: "At most 4 reference images allowed; you provided 6. Please remove extras and retry.",
		},
		{
			name: "videos_zh",
			raw:  "reference videos exceed Leonardo limit (4/3)",
			preferChinese: true,
			want: "参考视频最多 3 段，当前 4 段，请减少后重试。",
		},
		{
			name: "audios_en",
			raw:  "reference audios exceed Leonardo limit (2/1)",
			want: "At most 1 reference audio clip allowed; you provided 2. Please remove extras and retry.",
		},
		{
			name: "video_total_duration_zh",
			raw:  "reference videos total duration 16.0s exceeds Leonardo limit (15 s)",
			preferChinese: true,
			want: "参考视频总时长最多 15 秒，当前 16.0 秒，请缩短后重试。",
		},
		{
			name: "audio_duration_en",
			raw:  "reference audio duration 16.5s exceeds Leonardo limit (15 s)",
			want: "Reference audio must be at most 15s; yours is 16.5s. Please shorten and retry.",
		},
		{
			name: "video_clip_duration_zh",
			raw:  "reference video duration 2.0s not in 4-15 s range",
			preferChinese: true,
			want: "单条参考视频时长须在 4–15 秒之间，当前 2.0 秒，请调整后重试。",
		},
		{
			name: "mixed_mode_en",
			raw:  "multimodal references cannot be combined with start/end frame inputs",
			want: "Multimodal references (images/videos/audio) cannot be combined with start/end frames. Use one mode only.",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := HumanizeLeonardoReferenceLimitError(tc.preferChinese, tc.raw)
			if !ok {
				t.Fatalf("expected match for %q", tc.raw)
			}
			if got != tc.want {
				t.Fatalf("HumanizeLeonardoReferenceLimitError() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestHumanizeLeonardoReferenceLimitErrorNoMatch(t *testing.T) {
	if msg, ok := HumanizeLeonardoReferenceLimitError(true, "request parameters are invalid"); ok {
		t.Fatalf("unexpected match: %q", msg)
	}
}
