package audio

import (
	"bytes"
	"errors"
	"io"
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

func FetchTask(c *gin.Context, relayMode int) *dto.TaskError {
	if relayMode != relayconstant.RelayModeAudioFetchByID {
		return service.TaskErrorWrapperLocal(errors.New("invalid_relay_mode"), "invalid_relay_mode", http.StatusBadRequest)
	}
	respBody, taskErr := audioFetchByIDRespBodyBuilder(c)
	if taskErr != nil {
		return taskErr
	}
	if len(respBody) == 0 {
		respBody = []byte("{\"id\":\"\",\"object\":\"audio.generation\",\"status\":\"queued\"}")
	}
	c.Writer.Header().Set("Content-Type", "application/json")
	_, err := io.Copy(c.Writer, bytes.NewBuffer(respBody))
	if err != nil {
		return service.TaskErrorWrapper(err, "copy_response_body_failed", http.StatusInternalServerError)
	}
	return nil
}

func audioFetchByIDRespBodyBuilder(c *gin.Context) (respBody []byte, taskResp *dto.TaskError) {
	taskID := c.Param("task_id")
	if taskID == "" {
		taskID = c.GetString("task_id")
	}
	userID := c.GetInt("id")
	originTask, exist, err := model.GetByTaskIdForFetch(userID, taskID)
	if err != nil {
		taskResp = service.TaskErrorWrapper(err, "get_task_failed", http.StatusInternalServerError)
		return
	}
	if !exist {
		taskResp = service.TaskErrorWrapperLocal(errors.New("task_not_exist"), "task_not_exist", http.StatusBadRequest)
		return
	}
	job := originTask.ToOpenAIAudioJob(JobObjectForPath(c.Request.URL.Path))
	respBody, err = common.Marshal(job)
	if err != nil {
		taskResp = service.TaskErrorWrapper(err, "marshal_response_failed", http.StatusInternalServerError)
		return
	}
	respBody, err = service.PatchClientFacingModelJSONFromTask(originTask, respBody)
	if err != nil {
		taskResp = service.TaskErrorWrapper(err, "marshal_response_failed", http.StatusInternalServerError)
	}
	return
}
