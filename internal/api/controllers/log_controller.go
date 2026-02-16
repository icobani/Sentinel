package controllers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sentinel/sentinel/internal/api/services"
)

// LogController handles HTTP requests for log operations
type LogController struct {
	service *services.LogService
}

// NewLogController creates a new LogController instance
func NewLogController(service *services.LogService) *LogController {
	return &LogController{
		service: service,
	}
}

// GetLogs handles GET /api/v1/logs - returns paginated logs for all watchers
func (ctrl *LogController) GetLogs(c *gin.Context) {
	var params services.LogQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Invalid query parameters: " + err.Error(),
		})
		return
	}

	result, err := ctrl.service.GetLogs(params)
	if err != nil {
		status := mapErrorToStatus(err)
		c.JSON(status, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, PaginatedResponse{
		Success: true,
		Data:    result,
	})
}

// GetWatcherLogs handles GET /api/v1/watchers/:id/logs - returns paginated logs for a specific watcher
func (ctrl *LogController) GetWatcherLogs(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Invalid watcher ID",
		})
		return
	}

	var params services.LogQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Invalid query parameters: " + err.Error(),
		})
		return
	}

	result, err := ctrl.service.GetWatcherLogs(id, params)
	if err != nil {
		status := mapErrorToStatus(err)
		c.JSON(status, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, PaginatedResponse{
		Success: true,
		Data:    result,
	})
}

// GetWatcherStats handles GET /api/v1/watchers/:id/stats - returns webhook success/error statistics
func (ctrl *LogController) GetWatcherStats(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Invalid watcher ID",
		})
		return
	}

	stats, err := ctrl.service.GetWatcherStats(id)
	if err != nil {
		status := mapErrorToStatus(err)
		c.JSON(status, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    stats,
	})
}

// ExportLogs handles GET /api/v1/watchers/:id/logs/export - exports logs as Excel file
func (ctrl *LogController) ExportLogs(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Invalid watcher ID",
		})
		return
	}

	var params services.LogQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Invalid query parameters: " + err.Error(),
		})
		return
	}

	data, filename, err := ctrl.service.ExportLogs(id, params)
	if err != nil {
		status := mapErrorToStatus(err)
		c.JSON(status, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// Set headers for file download
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Length", strconv.Itoa(len(data)))

	// Send file data
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}

// DeleteLogs handles DELETE /api/v1/logs - bulk delete events by IDs
func (ctrl *LogController) DeleteLogs(c *gin.Context) {
	var req struct {
		IDs []uint `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Invalid request: " + err.Error(),
		})
		return
	}

	count, err := ctrl.service.DeleteLogs(req.IDs)
	if err != nil {
		status := mapErrorToStatus(err)
		c.JSON(status, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    gin.H{"deleted": count},
	})
}
