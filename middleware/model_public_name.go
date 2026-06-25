package middleware

import (
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

func PublicModelName() func(c *gin.Context) {
	return func(c *gin.Context) {
		if !service.ModelPublicNameEnabled() {
			c.Next()
			return
		}
		if err := service.ApplyPublicModelTranslation(c); err != nil {
			abortWithOpenAiMessage(c, http.StatusBadRequest, i18n.T(c, i18n.MsgDistributorInvalidRequest, map[string]any{"Error": err.Error()}))
			return
		}
		c.Next()
	}
}

func PublicModelNameForRetrieve() func(c *gin.Context) {
	return func(c *gin.Context) {
		if !service.ModelPublicNameEnabled() {
			c.Next()
			return
		}
		modelID := c.Param("model")
		if modelID == "" {
			c.Next()
			return
		}
		internal, clientPublic, err := service.ResolveInternalModelName(modelID)
		if err != nil {
			openAIError := types.OpenAIError{
				Message: err.Error(),
				Type:    "invalid_request_error",
				Param:   "model",
				Code:    "model_not_found",
			}
			c.JSON(http.StatusOK, gin.H{"error": openAIError})
			c.Abort()
			return
		}
		service.SetClientModelNameContext(c, clientPublic)
		common.SetContextKey(c, constant.ContextKeyOriginalModel, internal)
		c.Next()
	}
}
