package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sentinel/sentinel/internal/api/services"
)

// WatcherController handles HTTP requests for watcher operations
type WatcherController struct {
	service *services.WatcherService
}

// NewWatcherController creates a new WatcherController instance
func NewWatcherController(service *services.WatcherService) *WatcherController {
	return &WatcherController{
		service: service,
	}
}

// ValidatePath validates a directory path
func (ctrl *WatcherController) ValidatePath(c *gin.Context) {
	var req services.ValidatePathRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	result, err := ctrl.service.ValidatePath(req.Path)
	if err != nil {
		c.JSON(mapErrorToStatus(err), APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    result,
	})
}

// TestWebhook tests a watcher's webhook configuration
func (ctrl *WatcherController) TestWebhook(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	result, err := ctrl.service.TestWebhook(id)
	if err != nil {
		c.JSON(mapErrorToStatus(err), APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// Return TestWebhookResponse directly (not wrapped in APIResponse)
	c.JSON(http.StatusOK, result)
}

// CreateWatcher creates a new watcher
func (ctrl *WatcherController) CreateWatcher(c *gin.Context) {
	var req services.CreateWatcherRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	result, err := ctrl.service.CreateWatcher(req)
	if err != nil {
		c.JSON(mapErrorToStatus(err), APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, APIResponse{
		Success: true,
		Data:    result,
	})
}

// GetWatchers retrieves all watchers
func (ctrl *WatcherController) GetWatchers(c *gin.Context) {
	result, err := ctrl.service.GetWatchers()
	if err != nil {
		c.JSON(mapErrorToStatus(err), APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    result,
	})
}

// GetWatcher retrieves a specific watcher by ID
func (ctrl *WatcherController) GetWatcher(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	result, err := ctrl.service.GetWatcher(id)
	if err != nil {
		c.JSON(mapErrorToStatus(err), APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    result,
	})
}

// UpdateWatcher updates an existing watcher
func (ctrl *WatcherController) UpdateWatcher(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	var req services.CreateWatcherRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	result, err := ctrl.service.UpdateWatcher(id, req)
	if err != nil {
		c.JSON(mapErrorToStatus(err), APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    result,
	})
}

// DeleteWatcher deletes a watcher by ID
func (ctrl *WatcherController) DeleteWatcher(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	err = ctrl.service.DeleteWatcher(id)
	if err != nil {
		c.JSON(mapErrorToStatus(err), APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"message": "Watcher deleted successfully",
		},
	})
}

// StartWatcher starts a watcher
func (ctrl *WatcherController) StartWatcher(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	err = ctrl.service.StartWatcher(id)
	if err != nil {
		c.JSON(mapErrorToStatus(err), APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"message": "Watcher started successfully",
		},
	})
}

// StopWatcher stops a watcher
func (ctrl *WatcherController) StopWatcher(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	err = ctrl.service.StopWatcher(id)
	if err != nil {
		c.JSON(mapErrorToStatus(err), APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"message": "Watcher stopped successfully",
		},
	})
}

// RestartWatcher restarts a watcher
func (ctrl *WatcherController) RestartWatcher(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	err = ctrl.service.RestartWatcher(id)
	if err != nil {
		c.JSON(mapErrorToStatus(err), APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"message": "Watcher restarted successfully",
		},
	})
}
