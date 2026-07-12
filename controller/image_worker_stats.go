package controller

import (
	"net/http"

	"github.com/QuantumNous/new-api/relay/image"
	"github.com/gin-gonic/gin"
)

func GetImageWorkerStats(c *gin.Context) {
	stats, err := image.GetWorkerStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    stats,
	})
}
