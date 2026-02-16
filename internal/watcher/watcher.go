package watcher

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/sentinel/sentinel/internal/storage"
)

// FSWatcher implements fsnotify-based file system watching for local directories
type FSWatcher struct {
	id              uint
	name            string
	path            string
	recursive       bool
	includePatterns []string
	excludePatterns []string
	debounceTime    time.Duration
	watcher         *fsnotify.Watcher
	eventChan       chan *FileEvent
	stopChan        chan struct{}
	wg              sync.WaitGroup
	mu              sync.RWMutex
	running         bool
	lastEvents      map[string]time.Time // for debouncing
	eventsMu        sync.Mutex
}

// NewFSWatcher creates a new fsnotify-based watcher
func NewFSWatcher(w *storage.Watcher, debounceTime time.Duration) (*FSWatcher, error) {
	if debounceTime == 0 {
		debounceTime = 500 * time.Millisecond
	}

	fw := &FSWatcher{
		id:           w.ID,
		name:         w.Name,
		path:         w.Path,
		recursive:    w.Recursive,
		debounceTime: debounceTime,
		eventChan:    make(chan *FileEvent, 100),
		stopChan:     make(chan struct{}),
		lastEvents:   make(map[string]time.Time),
	}

	// Parse filters from JSON
	include, exclude, err := ParseFiltersFromWatcher(w)
	if err != nil {
		return nil, fmt.Errorf("failed to parse filters: %w", err)
	}
	fw.includePatterns = include
	fw.excludePatterns = exclude

	return fw, nil
}

// Start starts the watcher
func (fw *FSWatcher) Start() error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if fw.running {
		return fmt.Errorf("watcher already running")
	}

	// Create fsnotify watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}

	fw.watcher = watcher

	// Add path to watch
	if err := fw.addWatchPath(fw.path); err != nil {
		watcher.Close()
		return fmt.Errorf("failed to add watch path: %w", err)
	}

	fw.running = true

	// Start event processing goroutine
	fw.wg.Add(1)
	go fw.processEvents()

	// Start debounce map cleanup goroutine
	fw.wg.Add(1)
	go fw.cleanupDebounceMap()

	slog.Info("FSWatcher started", "id", fw.id, "name", fw.name, "path", fw.path)
	return nil
}

// Stop stops the watcher
func (fw *FSWatcher) Stop() error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if !fw.running {
		return nil
	}

	fw.running = false
	close(fw.stopChan)

	if fw.watcher != nil {
		fw.watcher.Close()
	}

	fw.wg.Wait()

	slog.Info("FSWatcher stopped", "id", fw.id, "name", fw.name)
	return nil
}

// IsRunning returns whether the watcher is running
func (fw *FSWatcher) IsRunning() bool {
	fw.mu.RLock()
	defer fw.mu.RUnlock()
	return fw.running
}

// GetEventChan returns the event channel
func (fw *FSWatcher) GetEventChan() <-chan *FileEvent {
	return fw.eventChan
}

// addWatchPath adds a path and optionally its subdirectories to the watcher
func (fw *FSWatcher) addWatchPath(path string) error {
	if err := fw.watcher.Add(path); err != nil {
		return err
	}

	slog.Debug("Added watch path", "path", path)

	// If recursive, add all subdirectories
	if fw.recursive {
		return filepath.Walk(path, func(walkPath string, info os.FileInfo, err error) error {
			if err != nil {
				slog.Warn("Error walking path", "path", walkPath, "error", err)
				return nil // continue walking
			}

			if info.IsDir() && walkPath != path {
				if err := fw.watcher.Add(walkPath); err != nil {
					slog.Warn("Failed to add subdirectory", "path", walkPath, "error", err)
					return nil // continue walking
				}
				slog.Debug("Added watch subdirectory", "path", walkPath)
			}

			return nil
		})
	}

	return nil
}

// processEvents processes fsnotify events
func (fw *FSWatcher) processEvents() {
	defer fw.wg.Done()
	defer close(fw.eventChan)

	slog.Info("processEvents goroutine started", "watcher", fw.name, "id", fw.id)

	for {
		select {
		case <-fw.stopChan:
			slog.Info("processEvents: stopChan closed, exiting", "watcher", fw.name)
			return

		case event, ok := <-fw.watcher.Events:
			if !ok {
				slog.Info("processEvents: Events channel closed, exiting", "watcher", fw.name)
				return
			}

			fw.handleFSNotifyEvent(event)

		case err, ok := <-fw.watcher.Errors:
			if !ok {
				slog.Info("processEvents: Errors channel closed, exiting", "watcher", fw.name)
				return
			}
			slog.Error("FSWatcher error", "id", fw.id, "name", fw.name, "error", err)
		}
	}
}

// handleFSNotifyEvent handles a single fsnotify event
func (fw *FSWatcher) handleFSNotifyEvent(event fsnotify.Event) {
	// Ignore CHMOD-only events (no actual file content change)
	if event.Op == fsnotify.Chmod {
		return
	}

	slog.Debug("Raw fsnotify event", "watcher", fw.name, "op", event.Op.String(), "file", event.Name)

	// Determine event type first (before filter/debounce)
	var eventType string
	switch {
	case event.Has(fsnotify.Create):
		eventType = "created"
	case event.Has(fsnotify.Write):
		eventType = "modified"
	case event.Has(fsnotify.Remove):
		eventType = "deleted"
	case event.Has(fsnotify.Rename):
		eventType = "renamed"
	default:
		return
	}

	// Check if we should process this file
	if !fw.shouldProcessFile(event.Name) {
		return
	}

	// Debounce events using path+type as key (so different event types aren't debounced together)
	debounceKey := event.Name + ":" + eventType
	if !fw.shouldEmitEvent(debounceKey) {
		return
	}

	// If it's a new directory and recursive mode, add it to watch
	if eventType == "created" {
		if info, err := os.Stat(event.Name); err == nil && info.IsDir() && fw.recursive {
			if addErr := fw.addWatchPath(event.Name); addErr != nil {
				slog.Error("Failed to add new directory to watch",
					"path", event.Name,
					"watcher_id", fw.id,
					"error", addErr)
			} else {
				slog.Info("Added new directory to watch", "path", event.Name, "watcher_id", fw.id)
			}
		}
	}

	// Get file info
	var fileSize int64
	if info, err := os.Stat(event.Name); err == nil {
		if !info.IsDir() {
			fileSize = info.Size()
		}
	}

	fileEvent := &FileEvent{
		WatcherID: fw.id,
		EventType: eventType,
		FilePath:  event.Name,
		FileName:  filepath.Base(event.Name),
		FileSize:  fileSize,
		Timestamp: time.Now(),
	}

	// Send event to channel
	select {
	case fw.eventChan <- fileEvent:
		slog.Debug("Event emitted", "watcher", fw.name, "type", eventType, "file", event.Name)
	default:
		slog.Warn("Event channel full, dropping event", "watcher", fw.name, "file", event.Name)
	}
}

// shouldProcessFile checks if a file should be processed based on filters (case-insensitive)
func (fw *FSWatcher) shouldProcessFile(path string) bool {
	fileName := strings.ToLower(filepath.Base(path))

	// Check exclude patterns first
	for _, pattern := range fw.excludePatterns {
		if matched, _ := filepath.Match(strings.ToLower(pattern), fileName); matched {
			return false
		}
	}

	// If no include patterns specified, include all (that didn't match exclude)
	if len(fw.includePatterns) == 0 {
		return true
	}

	// Check include patterns
	for _, pattern := range fw.includePatterns {
		if matched, _ := filepath.Match(strings.ToLower(pattern), fileName); matched {
			return true
		}
	}

	return false
}

// shouldEmitEvent checks if enough time has passed since last event for this file (debouncing)
func (fw *FSWatcher) shouldEmitEvent(path string) bool {
	fw.eventsMu.Lock()
	defer fw.eventsMu.Unlock()

	lastTime, exists := fw.lastEvents[path]
	now := time.Now()

	if exists && now.Sub(lastTime) < fw.debounceTime {
		return false
	}

	fw.lastEvents[path] = now
	return true
}

// cleanupDebounceMap periodically cleans up old entries from the debounce map to prevent memory leaks
func (fw *FSWatcher) cleanupDebounceMap() {
	defer fw.wg.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-fw.stopChan:
			return
		case <-ticker.C:
			fw.eventsMu.Lock()
			now := time.Now()
			// Remove entries older than 1 hour
			for path, lastTime := range fw.lastEvents {
				if now.Sub(lastTime) > 1*time.Hour {
					delete(fw.lastEvents, path)
				}
			}
			fw.eventsMu.Unlock()
			slog.Debug("Cleaned up debounce map", "watcher_id", fw.id)
		}
	}
}

// ValidatePath validates if a path exists, is accessible, and returns path information
func ValidatePath(path string) (*PathInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("path does not exist: %s", path)
		}
		if os.IsPermission(err) {
			return nil, fmt.Errorf("permission denied: %s", path)
		}
		return nil, fmt.Errorf("failed to access path: %w", err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", path)
	}

	// Check if directory is readable
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsPermission(err) {
			return nil, fmt.Errorf("directory is not readable: %w", err)
		}
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}
	_ = entries // Avoid unused variable warning

	// Try to check if we can watch it
	writable := false
	testWatcher, err := fsnotify.NewWatcher()
	if err == nil {
		defer testWatcher.Close()
		if err := testWatcher.Add(path); err != nil {
			return nil, fmt.Errorf("directory cannot be watched: %w", err)
		}

		// Optionally test write permission (non-intrusive)
		testFile := filepath.Join(path, ".sentinel_test")
		f, writeErr := os.Create(testFile)
		if writeErr == nil {
			f.Close()
			os.Remove(testFile)
			writable = true
		}
	}

	// Detect if it's a network share (basic heuristic)
	isNetwork := isNetworkPath(path)

	pathInfo := &PathInfo{
		Path:      path,
		Exists:    true,
		IsDir:     true,
		Readable:  true,
		Writable:  writable,
		IsNetwork: isNetwork,
		Size:      info.Size(),
		ModTime:   info.ModTime(),
	}

	return pathInfo, nil
}

// isNetworkPath detects if a path is a network share
func isNetworkPath(path string) bool {
	// Windows UNC path
	if strings.HasPrefix(path, "\\\\") || strings.HasPrefix(path, "//") {
		return true
	}

	// Check if it's a mounted network drive (basic check)
	// On Unix systems, network mounts might be in /mnt, /net, etc.
	lower := strings.ToLower(path)
	if strings.HasPrefix(lower, "/mnt/") || strings.HasPrefix(lower, "/net/") {
		return true
	}

	return false
}

// PathInfo contains information about a path
type PathInfo struct {
	Path      string    `json:"path"`
	Exists    bool      `json:"exists"`
	IsDir     bool      `json:"is_dir"`
	Readable  bool      `json:"readable"`
	Writable  bool      `json:"writable"`
	IsNetwork bool      `json:"is_network"`
	Size      int64     `json:"size"`
	ModTime   time.Time `json:"mod_time"`
}

// FileEvent represents a file system event
type FileEvent struct {
	WatcherID uint
	EventType string // created, modified, deleted, renamed
	FilePath  string
	FileName  string
	FileSize  int64
	OldPath   string // for rename events
	Timestamp time.Time
}
