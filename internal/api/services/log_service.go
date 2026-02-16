package services

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/sentinel/sentinel/internal/export"
	"github.com/sentinel/sentinel/internal/storage"
	"gorm.io/gorm"
)

// LogQueryParams represents query parameters for log retrieval
type LogQueryParams struct {
	Page      int    `form:"page"`
	Limit     int    `form:"limit"`
	WatcherID uint   `form:"watcher_id"`
	EventType string `form:"event_type"`
	Search    string `form:"search"`
	From      string `form:"from"`
	To        string `form:"to"`
	FromTime  *time.Time
	ToTime    *time.Time
}

// PaginatedData represents paginated response data
type PaginatedData struct {
	Items      interface{} `json:"items"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	Limit      int         `json:"limit"`
	TotalPages int         `json:"total_pages"`
}

// LogService handles log business logic
type LogService struct {
	DB *gorm.DB
}

// NewLogService creates a new log service
func NewLogService(db *gorm.DB) *LogService {
	return &LogService{
		DB: db,
	}
}

// GetLogs retrieves logs with pagination and filters
func (s *LogService) GetLogs(params LogQueryParams) (*PaginatedData, error) {
	// Set defaults
	if params.Page < 1 {
		params.Page = 1
	}
	if params.Limit < 1 {
		params.Limit = 20
	}

	// Parse date filters
	if err := s.parseDateFilters(&params); err != nil {
		return nil, fmt.Errorf("invalid date format: %w", ErrBadRequest)
	}

	return s.queryLogs(params)
}

// GetWatcherLogs retrieves logs for a specific watcher with pagination and filters
func (s *LogService) GetWatcherLogs(watcherID uint, params LogQueryParams) (*PaginatedData, error) {
	// Check if watcher exists
	if _, err := storage.GetWatcherByID(watcherID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("watcher not found: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("failed to retrieve watcher: %w", err)
	}

	// Set defaults
	if params.Page < 1 {
		params.Page = 1
	}
	if params.Limit < 1 {
		params.Limit = 20
	}

	// Override watcher ID from URL parameter
	params.WatcherID = watcherID

	// Parse date filters
	if err := s.parseDateFilters(&params); err != nil {
		return nil, fmt.Errorf("invalid date format: %w", ErrBadRequest)
	}

	return s.queryLogs(params)
}

// GetWatcherStats retrieves webhook statistics for a specific watcher
func (s *LogService) GetWatcherStats(watcherID uint) (*storage.WebhookStats, error) {
	// Check if watcher exists
	if _, err := storage.GetWatcherByID(watcherID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("watcher not found: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("failed to retrieve watcher: %w", err)
	}

	// Get webhook stats
	stats, err := storage.GetWebhookStats(watcherID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve webhook stats: %w", err)
	}

	return stats, nil
}

// ExportLogs exports logs to Excel format
func (s *LogService) ExportLogs(watcherID uint, params LogQueryParams) ([]byte, string, error) {
	// Check if watcher exists
	watcher, err := storage.GetWatcherByID(watcherID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", fmt.Errorf("watcher not found: %w", ErrNotFound)
		}
		return nil, "", fmt.Errorf("failed to retrieve watcher: %w", err)
	}

	// Parse date filters
	if err := s.parseDateFilters(&params); err != nil {
		return nil, "", fmt.Errorf("invalid date format: %w", ErrBadRequest)
	}

	// Set high limit to export all matching events (max 10000)
	params.Page = 1
	params.Limit = 10000
	params.WatcherID = watcherID

	// Query events
	events, _, err := storage.GetEvents(
		params.Page,
		params.Limit,
		params.WatcherID,
		params.EventType,
		params.Search,
		params.FromTime,
		params.ToTime,
	)
	if err != nil {
		return nil, "", fmt.Errorf("failed to retrieve events for export: %w", err)
	}

	// Fetch webhook logs for each event
	webhookLogs := make(map[uint]*storage.WebhookLog)
	for _, event := range events {
		logs, err := storage.GetWebhookLogsByEventID(event.ID)
		if err != nil {
			slog.Warn("Failed to get webhook logs for event", "event_id", event.ID, "error", err)
			continue
		}
		if len(logs) > 0 {
			// Use the most recent webhook log
			webhookLogs[event.ID] = &logs[0]
		}
	}

	// Generate Excel file
	f, err := export.ExportLogsToExcel(watcher.Name, events, webhookLogs)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate Excel file: %w", err)
	}
	defer f.Close()

	// Write to buffer
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, "", fmt.Errorf("failed to write Excel file to buffer: %w", err)
	}

	// Generate filename
	filename := export.GenerateFileName(watcher.Name)

	slog.Info("Exported logs to Excel",
		"watcher_id", watcherID,
		"watcher_name", watcher.Name,
		"event_count", len(events),
		"filename", filename,
	)

	return buf.Bytes(), filename, nil
}

// queryLogs is a private helper method to avoid code duplication
func (s *LogService) queryLogs(params LogQueryParams) (*PaginatedData, error) {
	// Query events from database
	events, total, err := storage.GetEvents(
		params.Page,
		params.Limit,
		params.WatcherID,
		params.EventType,
		params.Search,
		params.FromTime,
		params.ToTime,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve events: %w", err)
	}

	// Calculate total pages
	totalPages := int(total) / params.Limit
	if int(total)%params.Limit > 0 {
		totalPages++
	}

	return &PaginatedData{
		Items:      events,
		Total:      total,
		Page:       params.Page,
		Limit:      params.Limit,
		TotalPages: totalPages,
	}, nil
}

// DeleteLogs deletes events by their IDs
func (s *LogService) DeleteLogs(ids []uint) (int64, error) {
	if len(ids) == 0 {
		return 0, fmt.Errorf("no IDs provided: %w", ErrBadRequest)
	}
	count, err := storage.DeleteEventsByIDs(ids)
	if err != nil {
		return 0, fmt.Errorf("failed to delete events: %w", err)
	}
	return count, nil
}

// parseDateFilters parses the from/to date strings into time.Time pointers
func (s *LogService) parseDateFilters(params *LogQueryParams) error {
	// Parse from date
	if params.From != "" {
		fromTime, err := time.Parse(time.RFC3339, params.From)
		if err != nil {
			// Try alternative format (2006-01-02)
			fromTime, err = time.Parse("2006-01-02", params.From)
			if err != nil {
				return fmt.Errorf("invalid 'from' date format (expected RFC3339 or YYYY-MM-DD): %w", err)
			}
		}
		params.FromTime = &fromTime
	}

	// Parse to date
	if params.To != "" {
		toTime, err := time.Parse(time.RFC3339, params.To)
		if err != nil {
			// Try alternative format (2006-01-02)
			toTime, err = time.Parse("2006-01-02", params.To)
			if err != nil {
				return fmt.Errorf("invalid 'to' date format (expected RFC3339 or YYYY-MM-DD): %w", err)
			}
			// Set to end of day if only date was provided
			toTime = toTime.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		}
		params.ToTime = &toTime
	}

	return nil
}
