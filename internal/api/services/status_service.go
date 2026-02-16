package services

import (
	"fmt"
	"log/slog"
	"runtime"
	"time"

	"github.com/sentinel/sentinel/internal/config"
	"github.com/sentinel/sentinel/internal/storage"
	"github.com/sentinel/sentinel/internal/watcher"
	"gorm.io/gorm"
)

// Package-level variable to track service start time
var startTime = time.Now()

// StatusData contains system status information
type StatusData struct {
	Status             string    `json:"status"`
	Version            string    `json:"version"`
	BuildTime          string    `json:"build_time"`
	Uptime             string    `json:"uptime"`
	UptimeSeconds      float64   `json:"uptime_seconds"`
	StartTime          time.Time `json:"start_time"`
	GoVersion          string    `json:"go_version"`
	NumGoroutines      int       `json:"num_goroutines"`
	DatabaseStatus     string    `json:"database_status"`
	ConfigLoaded       bool      `json:"config_loaded"`
	WatcherCount       int64     `json:"watcher_count"`
	WatchersTotal      int64     `json:"watchers_total"`
	WatchersActive     int       `json:"watchers_active"`
	EventsToday        int64     `json:"events_today"`
	WebhookSuccessRate float64   `json:"webhook_success_rate"`
	PendingRestart     bool      `json:"pending_restart"`
}

// StatusService handles system status operations
type StatusService struct {
	DB      *gorm.DB
	Manager *watcher.Manager
}

// NewStatusService creates a new StatusService instance
func NewStatusService(db *gorm.DB, manager *watcher.Manager) *StatusService {
	return &StatusService{
		DB:      db,
		Manager: manager,
	}
}

// GetStatus retrieves current system status information
func (s *StatusService) GetStatus() *StatusData {
	// Check database connection
	dbStatus := "disconnected"
	var watcherCount int64 = 0
	var eventsToday int64 = 0
	var webhookSuccessRate float64 = 0

	if s.DB != nil {
		sqlDB, err := s.DB.DB()
		if err == nil {
			if err := sqlDB.Ping(); err == nil {
				dbStatus = "connected"

				// Get watcher count
				if err := s.DB.Model(&storage.Watcher{}).Count(&watcherCount).Error; err != nil {
					slog.Warn("Failed to count watchers", "error", err)
				}

				// Get events today
				if count, err := storage.GetEventCountToday(); err == nil {
					eventsToday = count
				}

				// Get webhook success rate
				if rate, err := storage.GetWebhookSuccessRate(); err == nil {
					webhookSuccessRate = rate
				}
			}
		}
	}

	// Get active watcher count from manager
	activeWatcherCount := 0
	if s.Manager != nil {
		activeWatcherCount = s.Manager.GetActiveWatcherCount()
	}

	// Calculate uptime
	uptime := time.Since(startTime)

	return &StatusData{
		Status:             "running",
		Version:            config.Version,
		BuildTime:          config.BuildTime,
		Uptime:             uptime.String(),
		UptimeSeconds:      uptime.Seconds(),
		StartTime:          startTime,
		GoVersion:          runtime.Version(),
		NumGoroutines:      runtime.NumGoroutine(),
		DatabaseStatus:     dbStatus,
		ConfigLoaded:       config.AppConfig != nil,
		WatcherCount:       watcherCount,
		WatchersTotal:      watcherCount,
		WatchersActive:     activeWatcherCount,
		EventsToday:        eventsToday,
		WebhookSuccessRate: webhookSuccessRate,
		PendingRestart:     config.IsPendingRestart(),
	}
}

// RestartService restarts all watchers and clears pending restart flag
func (s *StatusService) RestartService() error {
	slog.Info("Service restart requested")

	// Restart all watchers
	if err := s.Manager.RestartAll(); err != nil {
		return fmt.Errorf("failed to restart all watchers: %w", err)
	}

	// Clear pending restart flag
	config.SetPendingRestart(false)

	return nil
}

// GetPendingConfig returns pending restart status and appropriate message
func (s *StatusService) GetPendingConfig() (bool, string) {
	pending := config.IsPendingRestart()

	var message string
	if pending {
		message = "Configuration changes require service restart"
	} else {
		message = "No pending configuration changes"
	}

	return pending, message
}
