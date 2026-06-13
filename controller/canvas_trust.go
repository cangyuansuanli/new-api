package controller

import (
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
)

type canvasTrustVerifyRequest struct {
	Token string `json:"token"`
}

func CreateCanvasTrustToken(c *gin.Context) {
	if !setting.CanvasTrustConfigured() {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "canvas trust is not configured",
		})
		return
	}

	userID := c.GetInt("id")
	token, err := service.CreateCanvasTrustToken(userID)
	if err != nil {
		commonCanvasTrustError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"token":     token,
			"canvasUrl": setting.CanvasBaseURL,
			"expiresIn": setting.CanvasTrustTokenTTL,
		},
	})
}

func VerifyCanvasTrustToken(c *gin.Context) {
	var request canvasTrustVerifyRequest
	_ = c.ShouldBindJSON(&request)
	token := strings.TrimSpace(request.Token)
	if token == "" {
		token = strings.TrimSpace(c.Query("token"))
	}

	user, err := service.VerifyCanvasTrustToken(token)
	if err != nil {
		commonCanvasTrustError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    user,
	})
}

func commonCanvasTrustError(c *gin.Context, err error) {
	switch err {
	case service.ErrCanvasTrustDisabled:
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": err.Error(),
		})
	case service.ErrCanvasTrustInvalid:
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "invalid or expired trust token",
		})
	default:
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
	}
}
