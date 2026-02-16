package watcher

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sentinel/sentinel/internal/storage"
)

// Poller implements polling-based file system watching for network shares
type Poller struct {
	id              uint
	name            string
	path            string
	recursive       bool
	includePatterns []string
	excludePatterns []string
	pollInterval    time.Duration
	debounceTime    time.Duration
	eventChan       chan *FileEvent
	stopChan        chan struct{}
	wg              sync.WaitGroup
	mu              sync.RWMutex
	running         bool
	fileStates      map[string]*FileState // tracks file hashes and mod times
	statesMu        sync.Mutex
}

// FileState tracks the state of a file for change detection
type FileState struct {
	Path    string
	Size    int64
	ModTime time.Time
	Hash    string
	IsDir   bool
}

// NewPoller creates a new polling-based watcher
func NewPoller(w *storage.Watcher, pollInterval, debounceTime time.Duration) (*Poller, error) {
	if pollInterval == 0 {
		pollInterval = 5 * time.Second
	}
	if debounceTime == 0 {
		debounceTime = 500 * time.Millisecond
	}

	p := &Poller{
		id:           w.ID,
		name:         w.Name,
		path:         w.Path,
		recursive:    w.Recursive,
		pollInterval: pollInterval,
		debounceTime: debounceTime,
		eventChan:    make(chan *FileEvent, 100),
		stopChan:     make(chan struct{}),
		fileStates:   make(map[string]*FileState),
	}

	// Parse filters from JSON
	include, exclude, err := ParseFiltersFromWatcher(w)
	if err != nil {
		return nil, fmt.Errorf("failed to parse filters: %w", err)
	}
	p.includePatterns = include
	p.excludePatterns = exclude

	return p, nil
}

// Start starts the poller
func (p *Poller) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return fmt.Errorf("poller already running")
	}

	// Initial scan to build file state map
	if err := p.scanDirectory(); err != nil {
		return fmt.Errorf("initial scan failed: %w", err)
	}

	p.running = true

	// Start polling goroutine
	p.wg.Add(1)
	go p.poll()

	slog.Info("Poller started", "id", p.id, "name", p.name, "path", p.path, "interval", p.pollInterval)
	return nil
}

// Stop stops the poller
func (p *Poller) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return nil
	}

	p.running = false
	close(p.stopChan)
	p.wg.Wait()

	slog.Info("Poller stopped", "id", p.id, "name", p.name)
	return nil
}

// IsRunning returns whether the poller is running
func (p *Poller) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}

// GetEventChan returns the event channel
func (p *Poller) GetEventChan() <-chan *FileEvent {
	return p.eventChan
}

// poll continuously polls the directory for changes
func (p *Poller) poll() {
	defer p.wg.Done()
	defer close(p.eventChan)

	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopChan:
			return

		case <-ticker.C:
			if err := p.checkForChanges(); err != nil {
				slog.Error("Poller scan error", "id", p.id, "name", p.name, "error", err)
			}
		}
	}
}

// scanDirectory performs initial scan to build file state map
func (p *Poller) scanDirectory() error {
	p.statesMu.Lock()
	defer p.statesMu.Unlock()

	// Clear existing states
	p.fileStates = make(map[string]*FileState)

	walkFunc := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			slog.Warn("Error accessing path during scan", "path", path, "error", err)
			return nil // continue walking
		}

		// Skip if not matching filters
		if !info.IsDir() && !p.shouldProcessFile(path) {
			return nil
		}

		// Create file state
		state := &FileState{
			Path:    path,
			Size:    info.Size(),
			ModTime: info.ModTime(),
			IsDir:   info.IsDir(),
		}

		// Calculate hash for files (not directories)
		if !info.IsDir() {
			hash, err := calculateFileHash(path)
			if err != nil {
				slog.Warn("Failed to calculate hash during scan", "path", path, "error", err)
				// Continue without hash
			} else {
				state.Hash = hash
			}
		}

		p.fileStates[path] = state

		// If not recursive and this is a subdirectory, skip it
		if !p.recursive && info.IsDir() && path != p.path {
			return filepath.SkipDir
		}

		return nil
	}

	return filepath.Walk(p.path, walkFunc)
}

// checkForChanges scans directory and compares with previous state
func (p *Poller) checkForChanges() error {
	currentStates := make(map[string]*FileState)

	// Scan current directory state
	walkFunc := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			slog.Warn("Error accessing path", "path", path, "error", err)
			return nil
		}

		// Skip if not matching filters
		if !info.IsDir() && !p.shouldProcessFile(path) {
			return nil
		}

		// Create file state
		state := &FileState{
			Path:    path,
			Size:    info.Size(),
			ModTime: info.ModTime(),
			IsDir:   info.IsDir(),
		}

		// Calculate hash for files
		if !info.IsDir() {
			hash, err := calculateFileHash(path)
			if err != nil {
				slog.Warn("Failed to calculate hash", "path", path, "error", err)
			} else {
				state.Hash = hash
			}
		}

		currentStates[path] = state

		// If not recursive and this is a subdirectory, skip it
		if !p.recursive && info.IsDir() && path != p.path {
			return filepath.SkipDir
		}

		return nil
	}

	if err := filepath.Walk(p.path, walkFunc); err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	p.statesMu.Lock()
	defer p.statesMu.Unlock()

	// Detect new and modified files
	for path, currentState := range currentStates {
		oldState, existed := p.fileStates[path]

		// Skip directories for event emission
		if currentState.IsDir {
			continue
		}

		if !existed {
			// New file created
			p.emitEvent(&FileEvent{
				WatcherID: p.id,
				EventType: "created",
				FilePath:  path,
				FileName:  filepath.Base(path),
				FileSize:  currentState.Size,
				Timestamp: time.Now(),
			})
		} else if p.isFileModified(oldState, currentState) {
			// File modified
			p.emitEvent(&FileEvent{
				WatcherID: p.id,
				EventType: "modified",
				FilePath:  path,
				FileName:  filepath.Base(path),
				FileSize:  currentState.Size,
				Timestamp: time.Now(),
			})
		}
	}

	// Detect deleted files
	for path, oldState := range p.fileStates {
		if _, exists := currentStates[path]; !exists {
			// Skip directories for event emission
			if oldState.IsDir {
				continue
			}

			// File deleted
			p.emitEvent(&FileEvent{
				WatcherID: p.id,
				EventType: "deleted",
				FilePath:  path,
				FileName:  filepath.Base(path),
				FileSize:  oldState.Size,
				Timestamp: time.Now(),
			})
		}
	}

	// Update file states
	p.fileStates = currentStates

	return nil
}

// isFileModified checks if a file has been modified
func (p *Poller) isFileModified(oldState, newState *FileState) bool {
	// Check size first (fast)
	if oldState.Size != newState.Size {
		return true
	}

	// Check modification time
	if !oldState.ModTime.Equal(newState.ModTime) {
		// If hashes are available, use them for accurate comparison
		if oldState.Hash != "" && newState.Hash != "" {
			return oldState.Hash != newState.Hash
		}
		return true
	}

	// If hash is available, compare hashes
	if oldState.Hash != "" && newState.Hash != "" {
		return oldState.Hash != newState.Hash
	}

	return false
}

// shouldProcessFile checks if a file should be processed based on filters (case-insensitive)
func (p *Poller) shouldProcessFile(path string) bool {
	fileName := strings.ToLower(filepath.Base(path))

	// Check exclude patterns first
	for _, pattern := range p.excludePatterns {
		if matched, _ := filepath.Match(strings.ToLower(pattern), fileName); matched {
			return false
		}
	}

	// If no include patterns specified, include all (that didn't match exclude)
	if len(p.includePatterns) == 0 {
		return true
	}

	// Check include patterns
	for _, pattern := range p.includePatterns {
		if matched, _ := filepath.Match(strings.ToLower(pattern), fileName); matched {
			return true
		}
	}

	return false
}

// emitEvent sends an event to the event channel
func (p *Poller) emitEvent(event *FileEvent) {
	select {
	case p.eventChan <- event:
		slog.Debug("Event emitted", "watcher", p.name, "type", event.EventType, "file", event.FilePath)
	default:
		slog.Warn("Event channel full, dropping event", "watcher", p.name, "file", event.FilePath)
	}
}

// calculateFileHash calculates SHA256 hash of a file
func calculateFileHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file for hashing: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("failed to calculate file hash: %w", err)
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}
