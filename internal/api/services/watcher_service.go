package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/sentinel/sentinel/internal/config"
	"github.com/sentinel/sentinel/internal/storage"
	"github.com/sentinel/sentinel/internal/watcher"
	"github.com/sentinel/sentinel/internal/webhook"
	"github.com/sentinel/sentinel/internal/ws"
	"gorm.io/gorm"
)

// Sentinel error types for controller to map to HTTP status
var (
	ErrNotFound   = fmt.Errorf("not found")
	ErrConflict   = fmt.Errorf("conflict")
	ErrBadRequest = fmt.Errorf("bad request")
)

// CreateWatcherRequest represents the request body for creating a watcher
type CreateWatcherRequest struct {
	Name         string         `json:"name" binding:"required"`
	Path         string         `json:"path" binding:"required"`
	Mode         string         `json:"mode"`
	Recursive    bool           `json:"recursive"`
	PollInterval string         `json:"poll_interval"`
	Filters      FilterRequest  `json:"filters"`
	Webhook      WebhookRequest `json:"webhook" binding:"required"`
}

// FilterRequest represents filter configuration
type FilterRequest struct {
	Include []string `json:"include"`
	Exclude []string `json:"exclude"`
}

// WebhookRequest represents webhook configuration
type WebhookRequest struct {
	URL        string            `json:"url" binding:"required"`
	Headers    map[string]string `json:"headers"`
	Timeout    string            `json:"timeout"`
	Retry      RetryRequest      `json:"retry"`
	AttachFile bool              `json:"attach_file"`
}

// RetryRequest represents retry configuration
type RetryRequest struct {
	MaxAttempts int    `json:"max_attempts"`
	Backoff     string `json:"backoff"`
}

// WatcherResponse represents watcher data with status
type WatcherResponse struct {
	ID           uint           `json:"id"`
	Name         string         `json:"name"`
	Path         string         `json:"path"`
	Mode         string         `json:"mode"`
	Recursive    bool           `json:"recursive"`
	PollInterval string         `json:"poll_interval,omitempty"`
	Filters      FilterRequest  `json:"filters"`
	Webhook      WebhookRequest `json:"webhook"`
	IsRunning    bool           `json:"is_running"`
	CreatedAt    string         `json:"created_at"`
	UpdatedAt    string         `json:"updated_at"`
}

// ValidatePathRequest represents the request for path validation
type ValidatePathRequest struct {
	Path string `json:"path" binding:"required"`
}

// TestWebhookResponse represents the response for webhook test
type TestWebhookResponse struct {
	Success    bool   `json:"success"`
	StatusCode int    `json:"status_code,omitempty"`
	Error      string `json:"error,omitempty"`
}

// WatcherService handles watcher business logic
type WatcherService struct {
	DB      *gorm.DB
	Manager *watcher.Manager
	Hub     *ws.Hub
}

// NewWatcherService creates a new watcher service
func NewWatcherService(db *gorm.DB, manager *watcher.Manager, hub *ws.Hub) *WatcherService {
	return &WatcherService{
		DB:      db,
		Manager: manager,
		Hub:     hub,
	}
}

// ValidatePath validates a directory path
func (s *WatcherService) ValidatePath(path string) (map[string]interface{}, error) {
	pathInfo, err := watcher.ValidatePath(path)
	if err != nil {
		return map[string]interface{}{
			"valid":   false,
			"message": err.Error(),
		}, nil
	}

	message := "Directory exists and is accessible"
	if pathInfo.IsNetwork {
		message += " (network share detected - poll mode recommended)"
	}
	if !pathInfo.Writable {
		message += " (read-only)"
	}

	return map[string]interface{}{
		"valid":      true,
		"message":    message,
		"is_network": pathInfo.IsNetwork,
		"readable":   pathInfo.Readable,
		"writable":   pathInfo.Writable,
	}, nil
}

// TestWebhook tests a webhook by sending a mock request
func (s *WatcherService) TestWebhook(id uint) (*TestWebhookResponse, error) {
	// Load watcher from database
	var w storage.Watcher
	if err := s.DB.First(&w, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("watcher not found: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("failed to load watcher: %w", err)
	}

	// Create dispatcher and test webhook
	dispatcher := webhook.NewDispatcher()
	statusCode, success, errMsg, err := dispatcher.SendTestWebhook(&w)

	if err != nil {
		return &TestWebhookResponse{
			Success:    false,
			StatusCode: statusCode,
			Error:      err.Error(),
		}, nil
	}

	return &TestWebhookResponse{
		Success:    success,
		StatusCode: statusCode,
		Error:      errMsg,
	}, nil
}

// CreateWatcher creates a new watcher
func (s *WatcherService) CreateWatcher(req CreateWatcherRequest) (*WatcherResponse, error) {
	// Path traversal check
	if strings.Contains(req.Path, "..") {
		return nil, fmt.Errorf("invalid path: path traversal not allowed: %w", ErrBadRequest)
	}

	// Set defaults
	if req.Mode == "" {
		req.Mode = "watch"
	}

	// Validate mode
	if req.Mode != "watch" && req.Mode != "poll" {
		return nil, fmt.Errorf("invalid mode (must be 'watch' or 'poll'): %w", ErrBadRequest)
	}

	// Check if watcher with same name already exists
	exists, err := storage.WatcherExists(req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to check watcher existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("watcher with name '%s' already exists: %w", req.Name, ErrConflict)
	}

	// Validate path
	_, err = watcher.ValidatePath(req.Path)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %s: %w", err.Error(), ErrBadRequest)
	}

	// Encode filters as JSON
	filtersJSON, err := json.Marshal(req.Filters)
	if err != nil {
		return nil, fmt.Errorf("failed to encode filters: %w", err)
	}

	// Encode headers as JSON
	headersJSON, err := json.Marshal(req.Webhook.Headers)
	if err != nil {
		return nil, fmt.Errorf("failed to encode headers: %w", err)
	}

	// Create watcher in database
	w := storage.Watcher{
		Name:           req.Name,
		Path:           req.Path,
		Mode:           req.Mode,
		Recursive:      req.Recursive,
		PollInterval:   req.PollInterval,
		Filters:        string(filtersJSON),
		WebhookURL:     req.Webhook.URL,
		WebhookHeaders: string(headersJSON),
		Timeout:        req.Webhook.Timeout,
		MaxRetries:     req.Webhook.Retry.MaxAttempts,
		RetryBackoff:   req.Webhook.Retry.Backoff,
		AttachFile:     req.Webhook.AttachFile,
	}

	if err := storage.CreateWatcher(&w); err != nil {
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}

	// Add to config file
	configWatcher := config.WatcherConfig{
		Name:         req.Name,
		Path:         req.Path,
		Mode:         req.Mode,
		Recursive:    req.Recursive,
		PollInterval: req.PollInterval,
		Filters: config.FilterConfig{
			Include: req.Filters.Include,
			Exclude: req.Filters.Exclude,
		},
		Webhook: config.WebhookConfig{
			URL:        req.Webhook.URL,
			Headers:    req.Webhook.Headers,
			Timeout:    req.Webhook.Timeout,
			AttachFile: req.Webhook.AttachFile,
			Retry: config.RetryConfig{
				MaxAttempts: req.Webhook.Retry.MaxAttempts,
				Backoff:     req.Webhook.Retry.Backoff,
			},
		},
	}

	if err := config.AddWatcherToConfig(configWatcher); err != nil {
		slog.Error("Failed to add watcher to config file", "error", err)
		// Don't fail the request, just log the error
	} else {
		// Broadcast config change via WebSocket
		s.Hub.BroadcastConfigChange(ws.ConfigChangeData{
			Message:   fmt.Sprintf("Watcher '%s' created - restart required to apply changes", req.Name),
			Timestamp: time.Now().Format(time.RFC3339),
		})
	}

	// Convert to response format
	response := watcherToResponse(&w, false)

	return &response, nil
}

// GetWatchers retrieves all watchers
func (s *WatcherService) GetWatchers() ([]WatcherResponse, error) {
	watchers, err := storage.GetAllWatchers()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve watchers: %w", err)
	}

	// Get running status
	runningStatus := s.Manager.GetStatus()

	// Convert to response format
	responses := make([]WatcherResponse, len(watchers))
	for i, w := range watchers {
		isRunning := runningStatus[w.ID]
		responses[i] = watcherToResponse(&w, isRunning)
	}

	return responses, nil
}

// GetWatcher retrieves a single watcher by ID
func (s *WatcherService) GetWatcher(id uint) (*WatcherResponse, error) {
	// Get watcher from database
	w, err := storage.GetWatcherByID(id)
	if err != nil {
		return nil, wrapStorageError(err, "watcher not found")
	}

	// Check if running
	isRunning := s.Manager.IsRunning(id)

	// Convert to response format
	response := watcherToResponse(w, isRunning)

	return &response, nil
}

// UpdateWatcher updates an existing watcher
func (s *WatcherService) UpdateWatcher(id uint, req CreateWatcherRequest) (*WatcherResponse, error) {
	// Get existing watcher
	w, err := storage.GetWatcherByID(id)
	if err != nil {
		return nil, wrapStorageError(err, "watcher not found")
	}

	oldName := w.Name

	// Path traversal check
	if strings.Contains(req.Path, "..") {
		return nil, fmt.Errorf("invalid path: path traversal not allowed: %w", ErrBadRequest)
	}

	// Set defaults
	if req.Mode == "" {
		req.Mode = "watch"
	}

	// Validate mode
	if req.Mode != "watch" && req.Mode != "poll" {
		return nil, fmt.Errorf("invalid mode (must be 'watch' or 'poll'): %w", ErrBadRequest)
	}

	// Check if name changed and new name already exists
	if req.Name != oldName {
		exists, err := storage.WatcherExists(req.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to check watcher existence: %w", err)
		}
		if exists {
			return nil, fmt.Errorf("watcher with name '%s' already exists: %w", req.Name, ErrConflict)
		}
	}

	// Validate path
	_, err = watcher.ValidatePath(req.Path)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %s: %w", err.Error(), ErrBadRequest)
	}

	// Encode filters as JSON
	filtersJSON, err := json.Marshal(req.Filters)
	if err != nil {
		return nil, fmt.Errorf("failed to encode filters: %w", err)
	}

	// Encode headers as JSON
	headersJSON, err := json.Marshal(req.Webhook.Headers)
	if err != nil {
		return nil, fmt.Errorf("failed to encode headers: %w", err)
	}

	// Update watcher fields
	w.Name = req.Name
	w.Path = req.Path
	w.Mode = req.Mode
	w.Recursive = req.Recursive
	w.PollInterval = req.PollInterval
	w.Filters = string(filtersJSON)
	w.WebhookURL = req.Webhook.URL
	w.WebhookHeaders = string(headersJSON)
	w.Timeout = req.Webhook.Timeout
	w.MaxRetries = req.Webhook.Retry.MaxAttempts
	w.RetryBackoff = req.Webhook.Retry.Backoff
	w.AttachFile = req.Webhook.AttachFile

	if err := storage.UpdateWatcher(w); err != nil {
		return nil, fmt.Errorf("failed to update watcher: %w", err)
	}

	// Update config file
	configWatcher := config.WatcherConfig{
		Name:         req.Name,
		Path:         req.Path,
		Mode:         req.Mode,
		Recursive:    req.Recursive,
		PollInterval: req.PollInterval,
		Filters: config.FilterConfig{
			Include: req.Filters.Include,
			Exclude: req.Filters.Exclude,
		},
		Webhook: config.WebhookConfig{
			URL:        req.Webhook.URL,
			Headers:    req.Webhook.Headers,
			Timeout:    req.Webhook.Timeout,
			AttachFile: req.Webhook.AttachFile,
			Retry: config.RetryConfig{
				MaxAttempts: req.Webhook.Retry.MaxAttempts,
				Backoff:     req.Webhook.Retry.Backoff,
			},
		},
	}

	if err := config.UpdateWatcherInConfig(oldName, configWatcher); err != nil {
		slog.Error("Failed to update watcher in config file", "error", err)
		// Don't fail the request, just log the error
	} else {
		// Broadcast config change via WebSocket
		s.Hub.BroadcastConfigChange(ws.ConfigChangeData{
			Message:   fmt.Sprintf("Watcher '%s' updated - restart required to apply changes", req.Name),
			Timestamp: time.Now().Format(time.RFC3339),
		})
	}

	// Check if running
	isRunning := s.Manager.IsRunning(id)

	// Convert to response format
	response := watcherToResponse(w, isRunning)

	return &response, nil
}

// DeleteWatcher deletes a watcher
func (s *WatcherService) DeleteWatcher(id uint) error {
	// Get watcher to retrieve name
	w, err := storage.GetWatcherByID(id)
	if err != nil {
		return wrapStorageError(err, "watcher not found")
	}

	watcherName := w.Name

	// Stop watcher if running
	if s.Manager.IsRunning(id) {
		if err := s.Manager.StopWatcher(id); err != nil {
			slog.Error("Failed to stop watcher during deletion", "id", id, "error", err)
		}
	}

	// Delete from database
	if err := storage.DeleteWatcher(id); err != nil {
		return fmt.Errorf("failed to delete watcher: %w", err)
	}

	// Remove from config file
	if err := config.RemoveWatcherFromConfig(watcherName); err != nil {
		slog.Error("Failed to remove watcher from config file", "error", err)
		// Don't fail the request, just log the error
	} else {
		// Broadcast config change via WebSocket
		s.Hub.BroadcastConfigChange(ws.ConfigChangeData{
			Message:   fmt.Sprintf("Watcher '%s' deleted - restart required to apply changes", watcherName),
			Timestamp: time.Now().Format(time.RFC3339),
		})
	}

	return nil
}

// StartWatcher starts a single watcher
func (s *WatcherService) StartWatcher(id uint) error {
	// Check if watcher exists
	if _, err := storage.GetWatcherByID(id); err != nil {
		return wrapStorageError(err, "watcher not found")
	}

	// Start watcher
	if err := s.Manager.StartWatcher(id); err != nil {
		return fmt.Errorf("failed to start watcher: %w", err)
	}

	return nil
}

// StopWatcher stops a single watcher
func (s *WatcherService) StopWatcher(id uint) error {
	// Check if watcher exists
	if _, err := storage.GetWatcherByID(id); err != nil {
		return wrapStorageError(err, "watcher not found")
	}

	// Stop watcher
	if err := s.Manager.StopWatcher(id); err != nil {
		return fmt.Errorf("failed to stop watcher: %w", err)
	}

	return nil
}

// RestartWatcher restarts a single watcher
func (s *WatcherService) RestartWatcher(id uint) error {
	// Check if watcher exists
	if _, err := storage.GetWatcherByID(id); err != nil {
		return wrapStorageError(err, "watcher not found")
	}

	// Restart watcher
	if err := s.Manager.RestartWatcher(id); err != nil {
		return fmt.Errorf("failed to restart watcher: %w", err)
	}

	return nil
}

// wrapStorageError wraps a storage error with the appropriate sentinel error
func wrapStorageError(err error, notFoundMsg string) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("%s: %w", notFoundMsg, ErrNotFound)
	}
	return fmt.Errorf("database error: %w", err)
}

// watcherToResponse converts a storage.Watcher to WatcherResponse
func watcherToResponse(w *storage.Watcher, isRunning bool) WatcherResponse {
	// Parse filters
	var filters FilterRequest
	if w.Filters != "" {
		if err := json.Unmarshal([]byte(w.Filters), &filters); err != nil {
			slog.Warn("Failed to unmarshal filters", "watcher_id", w.ID, "error", err)
		}
	}

	// Parse headers
	var headers map[string]string
	if w.WebhookHeaders != "" {
		if err := json.Unmarshal([]byte(w.WebhookHeaders), &headers); err != nil {
			slog.Warn("Failed to unmarshal webhook headers", "watcher_id", w.ID, "error", err)
		}
	}

	return WatcherResponse{
		ID:           w.ID,
		Name:         w.Name,
		Path:         w.Path,
		Mode:         w.Mode,
		Recursive:    w.Recursive,
		PollInterval: w.PollInterval,
		Filters:      filters,
		Webhook: WebhookRequest{
			URL:        w.WebhookURL,
			Headers:    headers,
			Timeout:    w.Timeout,
			AttachFile: w.AttachFile,
			Retry: RetryRequest{
				MaxAttempts: w.MaxRetries,
				Backoff:     w.RetryBackoff,
			},
		},
		IsRunning: isRunning,
		CreatedAt: w.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: w.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
