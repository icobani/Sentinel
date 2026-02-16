package storage

import (
	"fmt"
	"time"
)

// CreateWebhookLog creates a new webhook log entry
func CreateWebhookLog(log *WebhookLog) error {
	if err := DB.Create(log).Error; err != nil {
		return fmt.Errorf("failed to create webhook log: %w", err)
	}
	return nil
}

// GetWebhookLogByID retrieves a webhook log by ID
func GetWebhookLogByID(id uint) (*WebhookLog, error) {
	var log WebhookLog
	if err := DB.Preload("Event").First(&log, id).Error; err != nil {
		return nil, fmt.Errorf("failed to get webhook log: %w", err)
	}
	return &log, nil
}

// GetWebhookLogsByEventID retrieves all webhook logs for a specific event
func GetWebhookLogsByEventID(eventID uint) ([]WebhookLog, error) {
	var logs []WebhookLog
	if err := DB.Where("event_id = ?", eventID).Order("created_at DESC").Find(&logs).Error; err != nil {
		return nil, fmt.Errorf("failed to get webhook logs by event ID: %w", err)
	}
	return logs, nil
}

// GetWebhookLogsByWatcherID retrieves webhook logs for a specific watcher with pagination
func GetWebhookLogsByWatcherID(watcherID uint, page, limit int) ([]WebhookLog, int64, error) {
	var logs []WebhookLog
	var total int64

	query := DB.Model(&WebhookLog{}).Where("watcher_id = ?", watcherID)

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count webhook logs: %w", err)
	}

	// Apply pagination
	offset := (page - 1) * limit
	if err := query.Preload("Event").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get webhook logs: %w", err)
	}

	return logs, total, nil
}

// GetRecentWebhookLogs retrieves the most recent webhook logs
func GetRecentWebhookLogs(limit int) ([]WebhookLog, error) {
	var logs []WebhookLog
	if err := DB.Preload("Event").
		Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error; err != nil {
		return nil, fmt.Errorf("failed to get recent webhook logs: %w", err)
	}
	return logs, nil
}

// GetWebhookStats returns webhook statistics for a watcher
func GetWebhookStats(watcherID uint) (*WebhookStats, error) {
	var stats WebhookStats

	query := DB.Model(&WebhookLog{})
	if watcherID > 0 {
		query = query.Where("watcher_id = ?", watcherID)
	}

	// Total count
	if err := query.Count(&stats.Total).Error; err != nil {
		return nil, fmt.Errorf("failed to count total webhook logs: %w", err)
	}

	// Success count
	if err := query.Where("success = ?", true).Count(&stats.SuccessCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count successful webhook logs: %w", err)
	}

	// Failure count
	stats.FailureCount = stats.Total - stats.SuccessCount

	// Calculate success rate
	if stats.Total > 0 {
		stats.SuccessRate = float64(stats.SuccessCount) / float64(stats.Total) * 100
	}

	// Average response time
	var avgTime struct {
		Avg float64
	}
	if err := query.Select("AVG(response_time) as avg").Where("success = ?", true).Scan(&avgTime).Error; err == nil {
		stats.AvgResponseTime = int64(avgTime.Avg)
	}

	return &stats, nil
}

// GetWebhookLogsByDateRange retrieves webhook logs within a date range
func GetWebhookLogsByDateRange(watcherID uint, from, to time.Time, page, limit int) ([]WebhookLog, int64, error) {
	var logs []WebhookLog
	var total int64

	query := DB.Model(&WebhookLog{})

	if watcherID > 0 {
		query = query.Where("watcher_id = ?", watcherID)
	}

	query = query.Where("created_at >= ? AND created_at <= ?", from, to)

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count webhook logs: %w", err)
	}

	// Apply pagination
	offset := (page - 1) * limit
	if err := query.Preload("Event").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get webhook logs: %w", err)
	}

	return logs, total, nil
}

// DeleteWebhookLogsByEventID deletes all webhook logs for a specific event
func DeleteWebhookLogsByEventID(eventID uint) error {
	if err := DB.Where("event_id = ?", eventID).Delete(&WebhookLog{}).Error; err != nil {
		return fmt.Errorf("failed to delete webhook logs: %w", err)
	}
	return nil
}

// DeleteWebhookLogsByWatcherID deletes all webhook logs for a specific watcher
func DeleteWebhookLogsByWatcherID(watcherID uint) error {
	if err := DB.Where("watcher_id = ?", watcherID).Delete(&WebhookLog{}).Error; err != nil {
		return fmt.Errorf("failed to delete webhook logs: %w", err)
	}
	return nil
}

// DeleteOldWebhookLogs deletes webhook logs older than the specified duration
func DeleteOldWebhookLogs(olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	result := DB.Where("created_at < ?", cutoff).Delete(&WebhookLog{})
	if result.Error != nil {
		return 0, fmt.Errorf("failed to delete old webhook logs: %w", result.Error)
	}
	return result.RowsAffected, nil
}

// WebhookStats represents webhook statistics
type WebhookStats struct {
	Total           int64   `json:"total"`
	SuccessCount    int64   `json:"success_count"`
	FailureCount    int64   `json:"failure_count"`
	SuccessRate     float64 `json:"success_rate"`
	AvgResponseTime int64   `json:"avg_response_time"` // milliseconds
}

// GetWebhookSuccessRate returns the overall webhook success rate as a percentage
func GetWebhookSuccessRate() (float64, error) {
	var total int64
	var successCount int64

	// Count total webhook logs
	if err := DB.Model(&WebhookLog{}).Count(&total).Error; err != nil {
		return 0, fmt.Errorf("failed to count total webhook logs: %w", err)
	}

	// If no logs, return 0%
	if total == 0 {
		return 0, nil
	}

	// Count successful webhook logs
	if err := DB.Model(&WebhookLog{}).Where("success = ?", true).Count(&successCount).Error; err != nil {
		return 0, fmt.Errorf("failed to count successful webhook logs: %w", err)
	}

	// Calculate success rate
	successRate := float64(successCount) / float64(total) * 100

	return successRate, nil
}
