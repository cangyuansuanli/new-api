package middleware

import (
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
)

func CanvasTrustSecretAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !setting.CanvasTrustConfigured() {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "canvas trust is not configured",
			})
			c.Abort()
			return
		}

		secret := strings.TrimSpace(c.GetHeader("X-Canvas-Trust-Secret"))
		if secret == "" {
			secret = strings.TrimSpace(c.GetHeader("Authorization"))
			secret = strings.TrimPrefix(secret, "Bearer ")
		}
		if secret == "" || secret != setting.CanvasTrustSecret {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "unauthorized",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
