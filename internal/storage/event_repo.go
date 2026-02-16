package storage

import (
	"fmt"
	"time"
)

// CreateEvent creates a new event in the database
func CreateEvent(event interface{}) error {
	// Support both FileEvent from watcher and Event model
	var dbEvent Event

	switch e := event.(type) {
	case *Event:
		dbEvent = *e
	case Event:
		dbEvent = e
	default:
		// Try to handle FileEvent from watcher package
		// We use reflection-like approach through interface{}
		return DB.Create(event).Error
	}

	if err := DB.Create(&dbEvent).Error; err != nil {
		return fmt.Errorf("failed to create event: %w", err)
	}

	// Cleanup old records if retention period has passed (throttled)
	CleanupIfNeeded()

	return nil
}

// GetEventByID retrieves an event by ID
func GetEventByID(id uint) (*Event, error) {
	var event Event
	if err := DB.Preload("Watcher").First(&event, id).Error; err != nil {
		return nil, fmt.Errorf("failed to get event: %w", err)
	}
	return &event, nil
}

// GetEvents retrieves events with pagination and filters
func GetEvents(page, limit int, watcherID uint, eventType, search string, from, to *time.Time) ([]Event, int64, error) {
	var events []Event
	var total int64

	query := DB.Model(&Event{})

	// Apply filters
	if watcherID > 0 {
		query = query.Where("watcher_id = ?", watcherID)
	}

	if eventType != "" {
		query = query.Where("event_type = ?", eventType)
	}

	if search != "" {
		query = query.Where("file_name LIKE ?", "%"+search+"%")
	}

	if from != nil {
		query = query.Where("created_at >= ?", from)
	}

	if to != nil {
		query = query.Where("created_at <= ?", to)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count events: %w", err)
	}

	// Apply pagination
	offset := (page - 1) * limit
	if err := query.Preload("Watcher").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&events).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get events: %w", err)
	}

	return events, total, nil
}

// GetEventsByWatcherID retrieves events for a specific watcher with pagination
func GetEventsByWatcherID(watcherID uint, page, limit int) ([]Event, int64, error) {
	var events []Event
	var total int64

	query := DB.Model(&Event{}).Where("watcher_id = ?", watcherID)

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count events: %w", err)
	}

	// Apply pagination
	offset := (page - 1) * limit
	if err := query.Preload("Watcher").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&events).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get events: %w", err)
	}

	return events, total, nil
}

// GetRecentEvents retrieves the most recent events
func GetRecentEvents(limit int) ([]Event, error) {
	var events []Event
	if err := DB.Preload("Watcher").
		Order("created_at DESC").
		Limit(limit).
		Find(&events).Error; err != nil {
		return nil, fmt.Errorf("failed to get recent events: %w", err)
	}
	return events, nil
}

// DeleteEvent deletes an event by ID
func DeleteEvent(id uint) error {
	if err := DB.Delete(&Event{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete event: %w", err)
	}
	return nil
}

// DeleteEventsByWatcherID deletes all events for a specific watcher
func DeleteEventsByWatcherID(watcherID uint) error {
	if err := DB.Where("watcher_id = ?", watcherID).Delete(&Event{}).Error; err != nil {
		return fmt.Errorf("failed to delete events: %w", err)
	}
	return nil
}

// GetEventCount returns the total number of events
func GetEventCount() (int64, error) {
	var count int64
	if err := DB.Model(&Event{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count events: %w", err)
	}
	return count, nil
}

// GetEventCountByWatcherID returns the number of events for a specific watcher
func GetEventCountByWatcherID(watcherID uint) (int64, error) {
	var count int64
	if err := DB.Model(&Event{}).Where("watcher_id = ?", watcherID).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count events: %w", err)
	}
	return count, nil
}

// GetEventCountByType returns the number of events by type
func GetEventCountByType(eventType string) (int64, error) {
	var count int64
	if err := DB.Model(&Event{}).Where("event_type = ?", eventType).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count events by type: %w", err)
	}
	return count, nil
}

// GetEventStats returns event statistics
func GetEventStats(watcherID uint) (map[string]int64, error) {
	stats := make(map[string]int64)

	query := DB.Model(&Event{})
	if watcherID > 0 {
		query = query.Where("watcher_id = ?", watcherID)
	}

	var results []struct {
		EventType string
		Count     int64
	}

	if err := query.Select("event_type, COUNT(*) as count").
		Group("event_type").
		Find(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to get event stats: %w", err)
	}

	for _, r := range results {
		stats[r.EventType] = r.Count
	}

	return stats, nil
}

// GetEventCountToday returns the number of events created today
func GetEventCountToday() (int64, error) {
	var count int64

	// Get start of today
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	if err := DB.Model(&Event{}).
		Where("created_at >= ?", startOfDay).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count events today: %w", err)
	}

	return count, nil
}

// DeleteOldEvents deletes events older than the specified duration
func DeleteOldEvents(olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	result := DB.Where("created_at < ?", cutoff).Delete(&Event{})
	if result.Error != nil {
		return 0, fmt.Errorf("failed to delete old events: %w", result.Error)
	}
	return result.RowsAffected, nil
}

// DeleteEventsByIDs deletes events by their IDs and their associated webhook logs
func DeleteEventsByIDs(ids []uint) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	// Delete associated webhook logs first
	DB.Where("event_id IN ?", ids).Delete(&WebhookLog{})
	// Delete events
	result := DB.Where("id IN ?", ids).Delete(&Event{})
	if result.Error != nil {
		return 0, fmt.Errorf("failed to delete events: %w", result.Error)
	}
	return result.RowsAffected, nil
}
