package service

import "testing"

func TestIsContentPolicyViolation(t *testing.T) {
	cases := []struct {
		text string
		want bool
	}{
		{"Generated video rejected by content moderation.", true},
		{"The generated images appear to be unsafe. Try modifying the prompts or the seeds.", true},
		{"非常抱歉，该提示可能违反了我们的内容政策。如果你认为此判断有误，请重试或修改提示语。", true},
		{"非常抱歉，生成的图片可能违反了关于与第三方内容相似性的防护限制。如果你认为此判断有误，请重试或修改提示语。", true},
		{"非常抱歉，生成的图片可能违反了关于暴力内容的防护限制。如果你认为此判断有误，请重试或修改提示语。", true},
		{"非常抱歉，生成的图片可能违反了我们的内容政策。如果你认为此判断有误，请重试或修改提示语。", true},
		{"非常抱歉，该提示可能违反了关于裸露、色情或情色内容的防护限制。", true},
		{"非常抱歉，生成的图片可能违反了关于裸露、色情或情色内容的防护限制。", true},
		{"status_code=400, I can't create an image with that level of explicit sexualization or erotic focus.", true},
		{"parse image json: unexpected end of JSON input", true},
		{"unexpected end of JSON input", true},
		{"invalid character 'e' looking for beginning of value", true},
		{"upstream returned unrecognized message", false},
		{"No available channel for model gpt-image-2", false},
		{"download image failed: connection refused", false},
	}

	for _, tc := range cases {
		if got := IsContentPolicyViolation(tc.text); got != tc.want {
			t.Fatalf("IsContentPolicyViolation(%q) = %v, want %v", tc.text, got, tc.want)
		}
	}
}

func TestStripLogArtifacts(t *testing.T) {
	raw := "The generated images appear to be unsafe... [truncated, original_length=1200, limit=512]"
	if got := stripLogArtifacts(raw); got != "The generated images appear to be unsafe" {
		t.Fatalf("stripLogArtifacts() = %q", got)
	}
}

func TestStripStatusCodePrefix(t *testing.T) {
	raw := "status_code=400, 非常抱歉，生成的图片可能违反了关于裸露、色情或情色内容的防护限制。"
	if got := stripStatusCodePrefix(raw); got != "非常抱歉，生成的图片可能违反了关于裸露、色情或情色内容的防护限制。" {
		t.Fatalf("stripStatusCodePrefix() = %q", got)
	}
}

func TestNormalizeClientErrorMessageUnified(t *testing.T) {
	cases := []struct {
		name          string
		raw           string
		preferChinese bool
		want          string
	}{
		{
			name: "content_policy_en",
			raw:  "Generated video rejected by content moderation.",
			want: ContentPolicyMessageEN,
		},
		{
			name: "content_policy_zh",
			raw:  "The generated images appear to be unsafe.",
			preferChinese: true,
			want: ContentPolicyMessageZH,
		},
		{
			name: "upstream_unavailable_en",
			raw:  "Upstream service temporarily unavailable",
			want: UpstreamUnavailableMessageEN,
		},
		{
			name: "upstream_unavailable_zh",
			raw:  "bad response status code 502",
			preferChinese: true,
			want: UpstreamUnavailableMessageZH,
		},
		{
			name: "timeout_en",
			raw:  "status_code=524, The origin web server did not return a complete response within the 120-second Proxy Read Timeout window.",
			want: TimeoutMessageEN,
		},
		{
			name: "timeout_zh",
			raw:  "任务超时（30分钟）",
			preferChinese: true,
			want: TimeoutMessageZH,
		},
		{
			name: "missing_reference_zh",
			raw:  "我目前还没有看到你说的「图片1」，因此无法基于它生成多个视角与景别。请你把参考图片（图片1）上传一下。",
			preferChinese: true,
			want: MissingReferenceMessageZH,
		},
		{
			name: "parse_error_as_content_policy",
			raw:  "invalid character 'e' looking for beginning of value",
			want: ContentPolicyMessageEN,
		},
		{
			name: "gulie_prompt_policy_zh",
			raw:  "非常抱歉，该提示可能违反了我们的内容政策。如果你认为此判断有误，请重试或修改提示语。",
			preferChinese: true,
			want: ContentPolicyMessageZH,
		},
		{
			name: "gulie_third_party_similarity_zh",
			raw:  "非常抱歉，生成的图片可能违反了关于与第三方内容相似性的防护限制。如果你认为此判断有误，请重试或修改提示语。",
			preferChinese: true,
			want: ContentPolicyMessageZH,
		},
		{
			name: "geek2_unsafe_en",
			raw:  "The generated images appear to be unsafe. Try modifying the prompts or the seeds.",
			want: ContentPolicyMessageEN,
		},
		{
			name: "upstream_safety_policy_en",
			raw:  "status_code=400, The model output was blocked by the upstream safety policy.",
			want: ContentPolicyMessageEN,
		},
		{
			name: "upstream_safety_policy_zh",
			raw:  "status_code=400, The model output was blocked by the upstream safety policy.",
			preferChinese: true,
			want: ContentPolicyMessageZH,
		},
		{
			name: "upstream_do_request_failed_timeout_en",
			raw:  "upstream error: do request failed",
			want: TimeoutMessageEN,
		},
		{
			name: "upstream_do_request_failed_timeout_zh",
			raw:  "upstream error: do request failed",
			preferChinese: true,
			want: TimeoutMessageZH,
		},
		{
			name: "upstream_capacity_en",
			raw:  "No capacity available for model gemini-3.1-flash-image on the server",
			want: UpstreamUnavailableMessageEN,
		},
		{
			name: "upstream_capacity_zh",
			raw:  "No capacity available for model gemini-3.1-flash-image on the server",
			preferChinese: true,
			want: UpstreamUnavailableMessageZH,
		},
		{
			name: "size_limit_passthrough",
			raw:  "gpt-image 最长边须 ≤3840（收到 4096×4096）",
			want: "gpt-image 最长边须 ≤3840（收到 4096×4096）",
		},
		{
			name: "leonardo_reference_download_zh",
			raw:  `All cookies failed. cookie#8: leonardo: download https://tmp.cangyuansuanli.cn/temp/video-refs/x: OK`,
			preferChinese: true,
			want: ReferenceMaterialMessageZH,
		},
		{
			name: "leonardo_audio_upload_en",
			raw:  "All cookies failed. cookie#8: leonardo: UploadImage: originalFilename is required for audio uploads",
			want: ReferenceMaterialMessageEN,
		},
		{
			name: "leonardo_pool_depleted_zh",
			raw:  "All cookies failed. cookie#5: depleted (auto-disabled)",
			preferChinese: true,
			want: PoolUnavailableMessageZH,
		},
		{
			name: "leonardo_generic_failure_en",
			raw:  "All cookies failed.",
			want: GenerationFailedMessageEN,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := NormalizeClientErrorMessageForLang(tc.preferChinese, tc.raw)
			if got != tc.want {
				t.Fatalf("NormalizeClientErrorMessageForLang() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestNormalizeClientErrorMessageContentPolicy(t *testing.T) {
	raw := "Generated video rejected by content moderation."
	if got := NormalizeClientErrorMessage(nil, raw); got != ContentPolicyMessageEN {
		t.Fatalf("NormalizeClientErrorMessage(nil) = %q", got)
	}
}
