package storage

import (
	"fmt"
)

// CreateWatcher creates a new watcher in the database
func CreateWatcher(watcher *Watcher) error {
	if err := DB.Create(watcher).Error; err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	return nil
}

// GetWatcherByID retrieves a watcher by ID
func GetWatcherByID(id uint) (*Watcher, error) {
	var watcher Watcher
	if err := DB.First(&watcher, id).Error; err != nil {
		return nil, fmt.Errorf("failed to get watcher: %w", err)
	}
	return &watcher, nil
}

// GetWatcherByName retrieves a watcher by name
func GetWatcherByName(name string) (*Watcher, error) {
	var watcher Watcher
	if err := DB.Where("name = ?", name).First(&watcher).Error; err != nil {
		return nil, fmt.Errorf("failed to get watcher by name: %w", err)
	}
	return &watcher, nil
}

// GetAllWatchers retrieves all watchers
func GetAllWatchers() ([]Watcher, error) {
	var watchers []Watcher
	if err := DB.Find(&watchers).Error; err != nil {
		return nil, fmt.Errorf("failed to get watchers: %w", err)
	}
	return watchers, nil
}

// UpdateWatcher updates an existing watcher
func UpdateWatcher(watcher *Watcher) error {
	if err := DB.Save(watcher).Error; err != nil {
		return fmt.Errorf("failed to update watcher: %w", err)
	}
	return nil
}

// DeleteWatcher deletes a watcher by ID
func DeleteWatcher(id uint) error {
	// First delete all associated webhook logs
	if err := DeleteWebhookLogsByWatcherID(id); err != nil {
		return fmt.Errorf("failed to delete associated webhook logs: %w", err)
	}

	// Then delete all associated events
	if err := DeleteEventsByWatcherID(id); err != nil {
		return fmt.Errorf("failed to delete associated events: %w", err)
	}

	// Finally delete the watcher
	if err := DB.Delete(&Watcher{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete watcher: %w", err)
	}
	return nil
}

// WatcherExists checks if a watcher with the given name exists
func WatcherExists(name string) (bool, error) {
	var count int64
	if err := DB.Model(&Watcher{}).Where("name = ?", name).Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check watcher existence: %w", err)
	}
	return count > 0, nil
}

// GetWatcherCount returns the total number of watchers
func GetWatcherCount() (int64, error) {
	var count int64
	if err := DB.Model(&Watcher{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count watchers: %w", err)
	}
	return count, nil
}
