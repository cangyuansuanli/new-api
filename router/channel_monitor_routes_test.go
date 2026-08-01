package router

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestChannelMonitorRoutesRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	SetApiRouter(engine)
	routes := make(map[string]struct{})
	for _, route := range engine.Routes() {
		routes[route.Method+" "+route.Path] = struct{}{}
	}
	expected := []string{
		"GET /api/channel-monitors",
		"GET /api/channel-monitors/:id/status",
		"GET /api/channel-monitors/admin",
		"POST /api/channel-monitors/admin",
		"PUT /api/channel-monitors/admin/settings",
		"PUT /api/channel-monitors/admin/:id",
		"DELETE /api/channel-monitors/admin/:id",
		"POST /api/channel-monitors/admin/:id/run",
	}
	for _, route := range expected {
		_, exists := routes[route]
		assert.True(t, exists, "missing route %s", route)
	}
}
