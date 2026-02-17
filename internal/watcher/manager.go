package watcher

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/sentinel/sentinel/internal/storage"
	"github.com/sentinel/sentinel/internal/webhook"
	"github.com/sentinel/sentinel/internal/ws"
)

// WatcherInstance represents a running watcher instance
type WatcherInstance struct {
	watcher    interface{} // either *FSWatcher or *Poller
	eventChan  <-chan *FileEvent
	cancelChan chan struct{}
	wg         sync.WaitGroup
}

// Manager manages all watchers
type Manager struct {
	instances    map[uint]*WatcherInstance
	reservations map[uint]struct{} // Track IDs being initialized to prevent races
	mu           sync.RWMutex
	eventHandler EventHandler
	debounceTime time.Duration
	dispatcher   *webhook.Dispatcher
}

// EventHandler is called when a file event occurs
type EventHandler func(event *FileEvent)

var (
	globalManager *Manager
	managerOnce   sync.Once
)

// GetManager returns the singleton manager instance
func GetManager() *Manager {
	managerOnce.Do(func() {
		globalManager = &Manager{
			instances:    make(map[uint]*WatcherInstance),
			reservations: make(map[uint]struct{}),
			debounceTime: 500 * time.Millisecond,
			dispatcher:   webhook.NewDispatcher(),
		}
	})
	return globalManager
}

// SetEventHandler sets the event handler function
func (m *Manager) SetEventHandler(handler EventHandler) {
	m.eventHandler = handler
}

// SetDebounceTime sets the debounce time for all watchers
func (m *Manager) SetDebounceTime(duration time.Duration) {
	m.debounceTime = duration
}

// GetDispatcher returns the webhook dispatcher
func (m *Manager) GetDispatcher() *webhook.Dispatcher {
	return m.dispatcher
}

// StartWatcher starts a single watcher by ID
func (m *Manager) StartWatcher(id uint) error {
	// Load watcher from database (outside mutex to avoid long lock)
	var w storage.Watcher
	if err := storage.DB.First(&w, id).Error; err != nil {
		return fmt.Errorf("failed to load watcher: %w", err)
	}

	// Reserve the slot to prevent concurrent initialization races
	m.mu.Lock()
	if _, exists := m.instances[id]; exists {
		m.mu.Unlock()
		return fmt.Errorf("watcher %d is already running", id)
	}
	if _, reserved := m.reservations[id]; reserved {
		m.mu.Unlock()
		return fmt.Errorf("watcher %d is already being initialized", id)
	}
	// Set reservation to prevent other goroutines from starting this watcher
	m.reservations[id] = struct{}{}
	m.mu.Unlock()

	// Ensure cleanup with defer
	var instance *WatcherInstance
	var err error
	defer func() {
		m.mu.Lock()
		if err != nil {
			// Cleanup reservation on error
			delete(m.reservations, id)
		} else {
			// Move from reservation to instance
			delete(m.reservations, id)
			m.instances[id] = instance
		}
		m.mu.Unlock()
	}()

	// Create watcher instance based on mode (outside mutex to avoid long lock)
	instance, err = m.createWatcherInstance(&w)
	if err != nil {
		return fmt.Errorf("failed to create watcher instance: %w", err)
	}

	// Start event consumer
	instance.wg.Add(1)
	go m.consumeEvents(id, instance)

	// Broadcast watcher status change via WebSocket
	hub := ws.GetHub()
	hub.BroadcastWatcherStatus(ws.WatcherStatusData{
		WatcherID:   w.ID,
		WatcherName: w.Name,
		Status:      "started",
		Message:     fmt.Sprintf("Watcher '%s' started successfully", w.Name),
		Timestamp:   time.Now().Format(time.RFC3339),
	})

	slog.Info("Watcher started", "id", id, "name", w.Name, "mode", w.Mode)
	return nil
}

// StopWatcher stops a single watcher by ID
func (m *Manager) StopWatcher(id uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	instance, exists := m.instances[id]
	if !exists || instance == nil {
		return fmt.Errorf("watcher %d is not running or not fully initialized", id)
	}

	// Stop the watcher
	if err := m.stopWatcherInstance(instance); err != nil {
		slog.Error("Error stopping watcher", "id", id, "error", err)
	}

	// Wait for event consumer to finish
	instance.wg.Wait()

	// Remove from instances
	delete(m.instances, id)

	// Broadcast watcher status change via WebSocket
	// Load watcher for broadcasting status (best effort)
	var w storage.Watcher
	if err := storage.DB.First(&w, id).Error; err == nil {
		hub := ws.GetHub()
		hub.BroadcastWatcherStatus(ws.WatcherStatusData{
			WatcherID:   w.ID,
			WatcherName: w.Name,
			Status:      "stopped",
			Message:     fmt.Sprintf("Watcher '%s' stopped", w.Name),
			Timestamp:   time.Now().Format(time.RFC3339),
		})
	}

	slog.Info("Watcher stopped", "id", id)
	return nil
}

// RestartWatcher restarts a single watcher
func (m *Manager) RestartWatcher(id uint) error {
	// Stop if running
	if m.IsRunning(id) {
		if err := m.StopWatcher(id); err != nil {
			return fmt.Errorf("failed to stop watcher: %w", err)
		}
	}

	// Start
	if err := m.StartWatcher(id); err != nil {
		return fmt.Errorf("failed to start watcher: %w", err)
	}

	return nil
}

// StartAll starts all watchers from database
func (m *Manager) StartAll() error {
	var watchers []storage.Watcher
	if err := storage.DB.Find(&watchers).Error; err != nil {
		return fmt.Errorf("failed to load watchers: %w", err)
	}

	var startErrors []error
	for _, w := range watchers {
		if err := m.StartWatcher(w.ID); err != nil {
			slog.Error("Failed to start watcher", "id", w.ID, "name", w.Name, "error", err)
			startErrors = append(startErrors, err)
		}
	}

	if len(startErrors) > 0 {
		return fmt.Errorf("failed to start %d watcher(s)", len(startErrors))
	}

	slog.Info("All watchers started", "count", len(watchers))
	return nil
}

// StopAll stops all running watchers
func (m *Manager) StopAll() error {
	m.mu.Lock()
	ids := make([]uint, 0, len(m.instances))
	for id := range m.instances {
		ids = append(ids, id)
	}
	m.mu.Unlock()

	var stopErrors []error
	for _, id := range ids {
		if err := m.StopWatcher(id); err != nil {
			slog.Error("Failed to stop watcher", "id", id, "error", err)
			stopErrors = append(stopErrors, err)
		}
	}

	if len(stopErrors) > 0 {
		return fmt.Errorf("failed to stop %d watcher(s)", len(stopErrors))
	}

	slog.Info("All watchers stopped")
	return nil
}

// RestartAll restarts all watchers
func (m *Manager) RestartAll() error {
	if err := m.StopAll(); err != nil {
		slog.Warn("Error stopping watchers during restart", "error", err)
	}

	// Small delay to ensure clean shutdown
	time.Sleep(100 * time.Millisecond)

	if err := m.StartAll(); err != nil {
		return fmt.Errorf("failed to start watchers: %w", err)
	}

	return nil
}

// IsRunning checks if a watcher is running
func (m *Manager) IsRunning(id uint) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	instance, exists := m.instances[id]
	// Check for placeholder (nil instance) - not actually running yet
	return exists && instance != nil
}

// GetRunningWatchers returns IDs of all running watchers
func (m *Manager) GetRunningWatchers() []uint {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids := make([]uint, 0, len(m.instances))
	for id := range m.instances {
		ids = append(ids, id)
	}
	return ids
}

// GetStatus returns status of all watchers
func (m *Manager) GetStatus() map[uint]bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := make(map[uint]bool)
	for id := range m.instances {
		status[id] = true
	}
	return status
}

// GetActiveWatcherCount returns the number of currently running watchers
func (m *Manager) GetActiveWatcherCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.instances)
}

// createWatcherInstance creates a watcher instance based on configuration
func (m *Manager) createWatcherInstance(w *storage.Watcher) (*WatcherInstance, error) {
	var eventChan <-chan *FileEvent
	var watcherImpl interface{}

	mode := w.Mode
	if mode == "" {
		mode = "watch"
	}

	switch mode {
	case "watch":
		fw, err := NewFSWatcher(w, m.debounceTime)
		if err != nil {
			return nil, fmt.Errorf("failed to create FSWatcher: %w", err)
		}
		if err := fw.Start(); err != nil {
			return nil, fmt.Errorf("failed to start FSWatcher: %w", err)
		}
		watcherImpl = fw
		eventChan = fw.GetEventChan()

	case "poll":
		// Parse poll interval
		pollInterval := 5 * time.Second
		if w.PollInterval != "" {
			if pi, err := time.ParseDuration(w.PollInterval); err == nil {
				pollInterval = pi
			}
		}

		poller, err := NewPoller(w, pollInterval, m.debounceTime)
		if err != nil {
			return nil, fmt.Errorf("failed to create Poller: %w", err)
		}
		if err := poller.Start(); err != nil {
			return nil, fmt.Errorf("failed to start Poller: %w", err)
		}
		watcherImpl = poller
		eventChan = poller.GetEventChan()

	default:
		return nil, fmt.Errorf("unknown watcher mode: %s", mode)
	}

	return &WatcherInstance{
		watcher:    watcherImpl,
		eventChan:  eventChan,
		cancelChan: make(chan struct{}),
	}, nil
}

// stopWatcherInstance stops a watcher instance
func (m *Manager) stopWatcherInstance(instance *WatcherInstance) error {
	close(instance.cancelChan)

	switch w := instance.watcher.(type) {
	case *FSWatcher:
		return w.Stop()
	case *Poller:
		return w.Stop()
	default:
		return fmt.Errorf("unknown watcher type")
	}
}

// consumeEvents consumes events from a watcher and processes them
func (m *Manager) consumeEvents(watcherID uint, instance *WatcherInstance) {
	defer instance.wg.Done()

	for {
		select {
		case <-instance.cancelChan:
			return

		case event, ok := <-instance.eventChan:
			if !ok {
				return
			}

			// Save event to database
			dbEvent, err := m.saveEvent(event)
			if err != nil {
				slog.Error("Failed to save event", "watcher_id", watcherID, "error", err)
				continue
			}

			// Load watcher from database for webhook config
			var w storage.Watcher
			if err := storage.DB.First(&w, watcherID).Error; err != nil {
				slog.Error("Failed to load watcher for webhook dispatch", "watcher_id", watcherID, "error", err)
			} else {
				// Broadcast file event via WebSocket
				hub := ws.GetHub()
				hub.BroadcastFileEvent(ws.FileEventData{
					WatcherID:   w.ID,
					WatcherName: w.Name,
					Event:       dbEvent.EventType,
					FileName:    dbEvent.FileName,
					FilePath:    dbEvent.FilePath,
					FileSize:    dbEvent.FileSize,
					OldPath:     dbEvent.OldPath,
					Timestamp:   dbEvent.CreatedAt.Format(time.RFC3339),
				})

				// Dispatch webhook if URL is configured
				// Note: Dispatch() is already async, no need for 'go' keyword
				if w.WebhookURL != "" {
					m.dispatchWebhook(&w, dbEvent)
				}
			}

			// Call event handler if set
			if m.eventHandler != nil {
				m.eventHandler(event)
			}
		}
	}
}

// saveEvent saves a file event to the database and returns the saved event
func (m *Manager) saveEvent(event *FileEvent) (*storage.Event, error) {
	dbEvent := storage.Event{
		WatcherID: event.WatcherID,
		EventType: event.EventType,
		FileName:  event.FileName,
		FilePath:  event.FilePath,
		FileSize:  event.FileSize,
		OldPath:   event.OldPath,
	}

	if err := storage.DB.Create(&dbEvent).Error; err != nil {
		return nil, fmt.Errorf("failed to save event to database: %w", err)
	}

	slog.Debug("Event saved to database", "id", dbEvent.ID, "watcher_id", event.WatcherID, "type", event.EventType)
	return &dbEvent, nil
}

// dispatchWebhook dispatches a webhook for an event
func (m *Manager) dispatchWebhook(watcher *storage.Watcher, event *storage.Event) {
	err := m.dispatcher.Dispatch(event, watcher)
	if err != nil {
		slog.Warn("Webhook dispatch failed",
			"watcher_id", watcher.ID,
			"event_id", event.ID,
			"error", err)
	} else {
		slog.Debug("Webhook queued for dispatch",
			"watcher_id", watcher.ID,
			"event_id", event.ID)
	}
}

// ParseFiltersFromWatcher parses filter configuration from watcher
func ParseFiltersFromWatcher(w *storage.Watcher) (include []string, exclude []string, err error) {
	if w.Filters == "" {
		return nil, nil, nil
	}

	var filters struct {
		Include []string `json:"include"`
		Exclude []string `json:"exclude"`
	}

	if err := json.Unmarshal([]byte(w.Filters), &filters); err != nil {
		return nil, nil, fmt.Errorf("failed to parse filters JSON: %w", err)
	}

	return filters.Include, filters.Exclude, nil
}

// LoadWatchersFromConfig loads watchers from config and saves to database
func LoadWatchersFromConfig(watchers []WatcherConfig) error {
	for _, wc := range watchers {
		// Check if watcher already exists
		var existing storage.Watcher
		result := storage.DB.Where("name = ?", wc.Name).First(&existing)

		// Encode filters as JSON
		filtersJSON, err := json.Marshal(wc.Filters)
		if err != nil {
			return fmt.Errorf("failed to encode filters for %s: %w", wc.Name, err)
		}

		// Encode webhook headers as JSON
		headersJSON, err := json.Marshal(wc.Webhook.Headers)
		if err != nil {
			return fmt.Errorf("failed to encode headers for %s: %w", wc.Name, err)
		}

		w := storage.Watcher{
			Name:           wc.Name,
			Path:           wc.Path,
			Mode:           wc.Mode,
			Recursive:      wc.Recursive,
			PollInterval:   wc.PollInterval,
			Filters:        string(filtersJSON),
			WebhookURL:     wc.Webhook.URL,
			WebhookHeaders: string(headersJSON),
			Timeout:        wc.Webhook.Timeout,
			MaxRetries:     wc.Webhook.Retry.MaxAttempts,
			RetryBackoff:   wc.Webhook.Retry.Backoff,
			AttachFile:     wc.Webhook.AttachFile,
		}

		if result.Error != nil {
			// Create new
			if err := storage.DB.Create(&w).Error; err != nil {
				return fmt.Errorf("failed to create watcher %s: %w", wc.Name, err)
			}
			slog.Info("Created watcher from config", "name", wc.Name)
		} else {
			// Update existing
			w.ID = existing.ID
			if err := storage.DB.Save(&w).Error; err != nil {
				return fmt.Errorf("failed to update watcher %s: %w", wc.Name, err)
			}
			slog.Info("Updated watcher from config", "name", wc.Name)
		}
	}

	return nil
}

// WatcherConfig represents watcher configuration from YAML
type WatcherConfig struct {
	Name         string
	Path         string
	Mode         string
	Recursive    bool
	PollInterval string
	Filters      struct {
		Include []string
		Exclude []string
	}
	Webhook struct {
		URL        string
		Headers    map[string]string
		Timeout    string
		AttachFile bool
		Retry      struct {
			MaxAttempts int
			Backoff     string
		}
	}
}
