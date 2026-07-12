package image

// EditFile 异步 edits 任务快照中的 multipart 文件。
type EditFile struct {
	Field       string `json:"field"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Data        []byte `json:"data,omitempty"` // legacy snapshots
	URL         string `json:"url,omitempty"`
	ObjectKey   string `json:"object_key,omitempty"`
}

// EditPayload 异步 edits 任务快照。
type EditPayload struct {
	Fields map[string]string `json:"fields"`
	Files  []EditFile        `json:"files"`
}
