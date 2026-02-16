package controllers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/sentinel/sentinel/internal/api/services"
)

// APIResponse represents a standard API response
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// PaginatedResponse represents a paginated response structure
type PaginatedResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// mapErrorToStatus maps service-level errors to HTTP status codes
func mapErrorToStatus(err error) int {
	if errors.Is(err, services.ErrNotFound) {
		return http.StatusNotFound
	}
	if errors.Is(err, services.ErrConflict) {
		return http.StatusConflict
	}
	if errors.Is(err, services.ErrBadRequest) {
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}

// parseID parses a string ID to uint
func parseID(idStr string) (uint, error) {
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint(id), nil
}
