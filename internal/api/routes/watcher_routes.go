package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/sentinel/sentinel/internal/api/controllers"
)

// InitWatcherRoutes registers all watcher-related routes
func InitWatcherRoutes(router *gin.RouterGroup, ctrl *controllers.WatcherController) {
	watchers := router.Group("/watchers")
	{
		watchers.POST("/validate-path", ctrl.ValidatePath)
		watchers.GET("", ctrl.GetWatchers)
		watchers.POST("", ctrl.CreateWatcher)
		watchers.GET("/:id", ctrl.GetWatcher)
		watchers.PUT("/:id", ctrl.UpdateWatcher)
		watchers.DELETE("/:id", ctrl.DeleteWatcher)
		watchers.POST("/:id/start", ctrl.StartWatcher)
		watchers.POST("/:id/stop", ctrl.StopWatcher)
		watchers.POST("/:id/restart", ctrl.RestartWatcher)
		watchers.POST("/:id/test", ctrl.TestWebhook)
	}
}
