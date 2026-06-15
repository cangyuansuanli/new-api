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

type canvasTrustUserRequest struct {
	UserID int `json:"user_id"`
}

type canvasTrustTokensRequest struct {
	UserID int `json:"user_id"`
	Start  int `json:"start"`
	Size   int `json:"size"`
}

type canvasTrustTokenKeyRequest struct {
	UserID  int `json:"user_id"`
	TokenID int `json:"token_id"`
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

func GetCanvasTrustUserSelf(c *gin.Context) {
	var request canvasTrustUserRequest
	_ = c.ShouldBindJSON(&request)
	if request.UserID <= 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "invalid user id",
		})
		return
	}
	profile, err := service.GetCanvasTrustUserProfile(request.UserID)
	if err != nil {
		commonCanvasTrustError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    profile,
	})
}

func ListCanvasTrustTokens(c *gin.Context) {
	var request canvasTrustTokensRequest
	_ = c.ShouldBindJSON(&request)
	if request.UserID <= 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "invalid user id",
		})
		return
	}
	items, total, err := service.ListCanvasTrustUserTokens(request.UserID, request.Start, request.Size)
	if err != nil {
		commonCanvasTrustError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"items": items,
			"total": total,
		},
	})
}

func GetCanvasTrustTokenKey(c *gin.Context) {
	var request canvasTrustTokenKeyRequest
	_ = c.ShouldBindJSON(&request)
	if request.UserID <= 0 || request.TokenID <= 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "invalid request",
		})
		return
	}
	key, err := service.GetCanvasTrustTokenKey(request.UserID, request.TokenID)
	if err != nil {
		commonCanvasTrustError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"key": key,
		},
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
	case service.ErrCanvasTrustUser:
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "invalid user",
		})
	default:
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
	}
}
