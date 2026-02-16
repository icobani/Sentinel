package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/sentinel/sentinel/internal/api/controllers"
)

// InitLogRoutes registers all log-related routes
func InitLogRoutes(router *gin.RouterGroup, ctrl *controllers.LogController) {
	// Combined logs endpoint
	router.GET("/logs", ctrl.GetLogs)
	router.DELETE("/logs", ctrl.DeleteLogs)

	// Watcher-specific log endpoints (under /watchers/:id)
	watchers := router.Group("/watchers")
	{
		watchers.GET("/:id/logs", ctrl.GetWatcherLogs)
		watchers.GET("/:id/logs/export", ctrl.ExportLogs)
		watchers.GET("/:id/stats", ctrl.GetWatcherStats)
	}
}
