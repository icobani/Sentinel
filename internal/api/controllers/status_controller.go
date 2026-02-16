package controllers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sentinel/sentinel/internal/api/services"
)

// StatusController handles system status-related HTTP requests
type StatusController struct {
	service *services.StatusService
}

// NewStatusController creates a new StatusController instance
func NewStatusController(service *services.StatusService) *StatusController {
	return &StatusController{
		service: service,
	}
}

// GetStatus retrieves current system status information
// GET /api/v1/status
func (ctrl *StatusController) GetStatus(c *gin.Context) {
	statusData := ctrl.service.GetStatus()

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    statusData,
	})
}

// RestartService restarts all watchers and clears pending restart flag
// POST /api/v1/service/restart
func (ctrl *StatusController) RestartService(c *gin.Context) {
	if err := ctrl.service.RestartService(); err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"message":      "Service restarted successfully",
			"restarted_at": time.Now().Format(time.RFC3339),
		},
	})
}

// GetPendingConfig retrieves pending configuration restart status
// GET /api/v1/config/pending
func (ctrl *StatusController) GetPendingConfig(c *gin.Context) {
	pending, message := ctrl.service.GetPendingConfig()

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"pending_restart": pending,
			"message":         message,
		},
	})
}
