package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

type TaskSetting struct {
	// RetentionDays controls automatic cleanup of finished async tasks.
	// 0 disables scheduled cleanup.
	RetentionDays int `json:"retention_days"`
}

var taskSetting = TaskSetting{
	RetentionDays: 30,
}

func init() {
	config.GlobalConfig.Register("task_setting", &taskSetting)
}

func GetTaskSetting() TaskSetting {
	return taskSetting
}
