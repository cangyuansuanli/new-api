package clienterror

import "testing"

func TestNormalizeSD5TaskFailureByStructuredCode(t *testing.T) {
	tests := []struct {
		name string
		code string
		raw  string
		want string
	}{
		{"overloaded", "submission_overloaded", "system under load", "SD5 上游负载过高或提交超时，请稍后重试。"},
		{"real face", "reference_image_privacy_error", "provider detail", ReferenceRealFaceMessageZH},
		{"unsafe video", "video_unsafe", "provider detail", ContentPolicyMessageZH},
		{"unsafe prompt", "prompt_unsafe", "provider detail", ContentPolicyMessageZH},
		{"url rejected", "bad_request", "Cannot fetch content from the provided URL. Status: URL_REJECTED", "参考素材链接无法被上游访问，请确认链接可公开下载后重新提交。"},
		{"video format", "image_not_processable", "provider detail", "参考视频无法处理，文件可能损坏或格式不受支持，请重新编码或更换素材。"},
		{"private address", "invalid_stored_request", "Seedance video reference URL resolves to a private address", "参考素材链接指向内网或私有地址，上游无法访问，请改用公开 HTTPS 链接。"},
		{"reference 404", "reference_fetch_error", "Seedance image reference returned status 404", "参考素材链接已失效或返回 404，请重新上传后提交。"},
		{"unknown submission", "submission_unknown", "provider detail", "SD5 提交结果未能确认，任务已终止且不会自动重试，请重新提交。"},
		{"timeout", "upstream_timeout", "provider detail", TimeoutMessageZH},
		{"webm", "validation_error", "Unsupported content type: video/webm", "SD5 当前不支持 WebM 参考视频，请转换为 MP4 后重试。"},
		{"prompt limit", "validation_error", "Validation error for field 'prompt': String should have at most 2500 characters", "提示词超过 2500 字符，请缩短后重新提交。"},
		{"access", "access_error", "Adobe HTTP 403", "SD5 上游账号权限异常，请稍后重试或联系管理员。"},
		{"quota", "quota_exhausted", "Adobe HTTP 403", "SD5 号池可用额度已耗尽，请联系管理员补充额度。"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeErrorForLang(true, ErrorContext{
				Model:     "cy-sd5-seedance-2.0",
				Raw:       tt.raw,
				ErrorType: tt.code,
			})
			if got != tt.want {
				t.Fatalf("NormalizeErrorForLang() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeSD5ParsesPersistedErrorFieldsInChannelRule(t *testing.T) {
	failure := ErrorContext{
		Model:   "cy-sd5-seedance-2.0-fast",
		Payload: []byte(`{"error":"system under load","error_type":"submission_overloaded","error_status":408}`),
	}
	if got, want := NormalizeErrorForLang(true, failure), "SD5 上游负载过高或提交超时，请稍后重试。"; got != want {
		t.Fatalf("NormalizeErrorForLang() = %q, want %q", got, want)
	}
}

func TestNormalizeSD5PayloadCodeOverridesGenericTaskErrorCode(t *testing.T) {
	failure := ErrorContext{
		Model:     "cy-sd5-seedance-2.0",
		Raw:       `{"error":"system under load","error_type":"submission_overloaded"}`,
		ErrorCode: "fail_to_fetch_task",
	}
	if got, want := NormalizeErrorForLang(true, failure), "SD5 上游负载过高或提交超时，请稍后重试。"; got != want {
		t.Fatalf("NormalizeErrorForLang() = %q, want %q", got, want)
	}
}

func TestNormalizeTaskFailureDoesNotApplySD5RulesToOtherChannels(t *testing.T) {
	failure := ErrorContext{Model: "other-model", Raw: "system under load", ErrorType: "submission_overloaded"}
	if got := NormalizeErrorForLang(true, failure); got != failure.Raw {
		t.Fatalf("NormalizeErrorForLang() = %q, want raw %q", got, failure.Raw)
	}
}

func TestNormalizeSD5SubmissionOnlyErrorsStayChannelScoped(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{"Seedance media video/audio references require at least one image reference", "使用参考视频或音频时，至少需要同时提供 1 张参考图。"},
		{"invalid request body: http: request body too large", "请求体超过 64 MB，请压缩或减少参考素材后重试。"},
	}
	for _, tt := range tests {
		failure := ErrorContext{Model: "cy-sd5-seedance-2.0", Raw: tt.raw}
		if got := NormalizeErrorForLang(true, failure); got != tt.want {
			t.Fatalf("NormalizeErrorForLang(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}
