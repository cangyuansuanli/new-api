package dto

const (
	AudioJobStatusQueued     = "queued"
	AudioJobStatusInProgress = "in_progress"
	AudioJobStatusCompleted  = "completed"
	AudioJobStatusFailed     = "failed"
)

type OpenAIAudioJob struct {
	ID        string              `json:"id"`
	Object    string              `json:"object"`
	Model     string              `json:"model,omitempty"`
	Status    string              `json:"status"`
	Progress  string              `json:"progress,omitempty"`
	CreatedAt int64               `json:"created_at"`
	Data      []AudioGenerationData `json:"data,omitempty"`
	Error     *OpenAIAudioJobError  `json:"error,omitempty"`
}

type OpenAIAudioJobError struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

func NewOpenAIAudioJob(object string) *OpenAIAudioJob {
	return &OpenAIAudioJob{
		Object: object,
		Status: AudioJobStatusQueued,
	}
}
