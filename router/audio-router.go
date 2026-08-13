package router

import (
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"

	"github.com/gin-gonic/gin"
)

func SetAudioRouter(router *gin.Engine) {
	audioV1Router := router.Group("/v1")
	audioV1Router.Use(middleware.CORS())
	audioV1Router.Use(middleware.RouteTag("relay"))
	audioV1Router.Use(middleware.TokenAuth(), middleware.PublicModelName(), middleware.Distribute())
	{
		audioV1Router.GET("/audio/generations/:task_id", controller.RelayAudioTaskFetch)
	}
}
