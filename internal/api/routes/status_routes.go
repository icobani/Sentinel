package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/sentinel/sentinel/internal/api/controllers"
)

// InitStatusRoutes registers status and service-related routes
func InitStatusRoutes(router *gin.RouterGroup, ctrl *controllers.StatusController) {
	router.GET("/status", ctrl.GetStatus)
	router.POST("/service/restart", ctrl.RestartService)
	router.GET("/config/pending", ctrl.GetPendingConfig)
}
