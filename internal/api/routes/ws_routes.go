package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/sentinel/sentinel/internal/api/controllers"
)

// InitWSRoutes registers WebSocket routes
// Note: WebSocket is registered on the root engine, not under /api/v1
func InitWSRoutes(router *gin.Engine, ctrl *controllers.WSController) {
	router.GET("/ws", ctrl.HandleWS)
}
