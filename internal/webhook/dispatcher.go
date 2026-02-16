package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/sentinel/sentinel/internal/storage"
	"github.com/sentinel/sentinel/internal/ws"
)

// Dispatcher manages webhook delivery
type Dispatcher struct {
	client     *http.Client
	eventQueue chan *dispatchJob
	workers    int
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	mu         sync.RWMutex
	running    bool
}

// dispatchJob represents a webhook dispatch job
type dispatchJob struct {
	event   *storage.Event
	watcher *storage.Watcher
}

// WebhookPayload represents the JSON payload sent to webhooks
type WebhookPayload struct {
	WatcherID   uint      `json:"watcher_id"`
	WatcherName string    `json:"watcher_name"`
	Event       string    `json:"event"`
	FileName    string    `json:"file_name"`
	FilePath    string    `json:"file_path"`
	FileSize    int64     `json:"file_size"`
	OldPath     string    `json:"old_path,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// NewDispatcher creates a new webhook dispatcher
// If workers is not provided (0 or negative), defaults to 5 workers
func NewDispatcher(workers ...int) *Dispatcher {
	workerCount := 5 // default worker count
	if len(workers) > 0 && workers[0] > 0 {
		workerCount = workers[0]
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Dispatcher{
		client: &http.Client{
			Timeout: 30 * time.Second, // default, overridden per watcher
		},
		eventQueue: make(chan *dispatchJob, 100),
		workers:    workerCount,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start starts the dispatcher workers
func (d *Dispatcher) Start() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.running {
		return fmt.Errorf("dispatcher already running")
	}

	d.running = true

	// Start worker goroutines
	for i := 0; i < d.workers; i++ {
		d.wg.Add(1)
		go d.worker(i)
	}

	slog.Info("Webhook dispatcher started", "workers", d.workers)
	return nil
}

// Stop stops the dispatcher gracefully
func (d *Dispatcher) Stop() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.running {
		return nil
	}

	d.running = false
	d.cancel()
	close(d.eventQueue)
	d.wg.Wait()

	slog.Info("Webhook dispatcher stopped")
	return nil
}

// IsRunning returns whether the dispatcher is running
func (d *Dispatcher) IsRunning() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.running
}

// Dispatch queues an event for webhook delivery
func (d *Dispatcher) Dispatch(event *storage.Event, w *storage.Watcher) error {
	d.mu.RLock()
	running := d.running
	d.mu.RUnlock()

	if !running {
		return fmt.Errorf("dispatcher not running")
	}

	job := &dispatchJob{
		event:   event,
		watcher: w,
	}

	select {
	case d.eventQueue <- job:
		slog.Debug("Event queued for webhook dispatch",
			"event_id", event.ID,
			"watcher", w.Name,
			"file", event.FileName)
		return nil
	case <-d.ctx.Done():
		return fmt.Errorf("dispatcher shutting down")
	default:
		slog.Warn("Webhook queue full, dropping event",
			"event_id", event.ID,
			"watcher", w.Name)
		return fmt.Errorf("webhook queue full")
	}
}

// worker processes webhook dispatch jobs
func (d *Dispatcher) worker(id int) {
	defer d.wg.Done()

	slog.Debug("Webhook worker started", "worker_id", id)

	for {
		select {
		case <-d.ctx.Done():
			slog.Debug("Webhook worker stopping", "worker_id", id)
			return

		case job, ok := <-d.eventQueue:
			if !ok {
				slog.Debug("Webhook worker stopped", "worker_id", id)
				return
			}

			d.processJob(job)
		}
	}
}

// processJob processes a single webhook dispatch job
func (d *Dispatcher) processJob(job *dispatchJob) {
	startTime := time.Now()

	// Parse timeout
	timeout := 10 * time.Second // default
	if job.watcher.Timeout != "" {
		if parsedTimeout, err := time.ParseDuration(job.watcher.Timeout); err == nil {
			timeout = parsedTimeout
		} else {
			slog.Warn("Invalid timeout duration, using default",
				"watcher", job.watcher.Name,
				"timeout", job.watcher.Timeout,
				"error", err)
		}
	}

	// Parse retry config
	maxRetries := job.watcher.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3 // default
	}

	backoff := 2 * time.Second // default
	if job.watcher.RetryBackoff != "" {
		if parsedBackoff, err := time.ParseDuration(job.watcher.RetryBackoff); err == nil {
			backoff = parsedBackoff
		}
	}

	// Create payload
	payload := WebhookPayload{
		WatcherID:   job.watcher.ID,
		WatcherName: job.watcher.Name,
		Event:       job.event.EventType,
		FileName:    job.event.FileName,
		FilePath:    job.event.FilePath,
		FileSize:    job.event.FileSize,
		OldPath:     job.event.OldPath,
		Timestamp:   job.event.CreatedAt,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		slog.Error("Failed to marshal webhook payload",
			"event_id", job.event.ID,
			"error", err)
		d.saveWebhookLog(job, 0, false, fmt.Sprintf("failed to marshal payload: %v", err), 0, "", string(payloadBytes), 0)
		return
	}

	// Send with retry
	statusCode, success, errMsg, responseBody, retryCount := SendWithRetry(
		d.ctx,
		job.watcher.WebhookURL,
		payloadBytes,
		job.watcher.WebhookHeaders,
		timeout,
		maxRetries,
		backoff,
	)

	responseTime := time.Since(startTime).Milliseconds()

	// Save log
	d.saveWebhookLog(job, statusCode, success, errMsg, responseTime, responseBody, string(payloadBytes), retryCount)

	// Broadcast webhook result via WebSocket
	hub := ws.GetHub()
	hub.BroadcastWebhookResult(ws.WebhookResultData{
		WatcherID:     job.watcher.ID,
		WatcherName:   job.watcher.Name,
		Event:         job.event.EventType,
		FileName:      job.event.FileName,
		FilePath:      job.event.FilePath,
		FileSize:      job.event.FileSize,
		Timestamp:     job.event.CreatedAt.Format(time.RFC3339),
		WebhookStatus: statusCode,
		WebhookError:  errMsg,
		RetryCount:    retryCount,
		MaxRetries:    maxRetries,
	})

	if success {
		slog.Info("Webhook delivered successfully",
			"event_id", job.event.ID,
			"watcher", job.watcher.Name,
			"status", statusCode,
			"duration_ms", responseTime)
	} else {
		slog.Error("Webhook delivery failed",
			"event_id", job.event.ID,
			"watcher", job.watcher.Name,
			"status", statusCode,
			"error", errMsg,
			"duration_ms", responseTime)
	}
}

// saveWebhookLog saves webhook delivery log to database
func (d *Dispatcher) saveWebhookLog(job *dispatchJob, statusCode int, success bool, errMsg string, responseTime int64, responseBody, requestPayload string, retryCount int) {
	// Truncate response body if too large (keep first 10KB)
	if len(responseBody) > 10240 {
		responseBody = responseBody[:10240] + "... (truncated)"
	}

	// Truncate error message if too large
	if len(errMsg) > 1000 {
		errMsg = errMsg[:1000] + "... (truncated)"
	}

	webhookLog := &storage.WebhookLog{
		EventID:        job.event.ID,
		WatcherID:      job.watcher.ID,
		StatusCode:     statusCode,
		Success:        success,
		Error:          errMsg,
		RetryCount:     retryCount,
		ResponseTime:   responseTime,
		RequestPayload: requestPayload,
		ResponseBody:   responseBody,
	}

	if err := storage.DB.Create(webhookLog).Error; err != nil {
		slog.Error("Failed to save webhook log",
			"event_id", job.event.ID,
			"error", err)
	}
}

// SendTestWebhook sends a test webhook with mock data
func (d *Dispatcher) SendTestWebhook(w *storage.Watcher) (int, bool, string, error) {
	// Parse timeout
	timeout := 10 * time.Second
	if w.Timeout != "" {
		if parsedTimeout, err := time.ParseDuration(w.Timeout); err == nil {
			timeout = parsedTimeout
		}
	}

	// Create test payload
	payload := WebhookPayload{
		WatcherID:   w.ID,
		WatcherName: w.Name,
		Event:       "created",
		FileName:    "test-file.txt",
		FilePath:    "/test/path/test-file.txt",
		FileSize:    1024,
		Timestamp:   time.Now(),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return 0, false, "", fmt.Errorf("failed to marshal test payload: %w", err)
	}

	// Send without retry (single attempt for testing)
	req, err := http.NewRequestWithContext(d.ctx, "POST", w.WebhookURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return 0, false, "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Sentinel-Webhook/1.0")

	// Parse and set custom headers
	if w.WebhookHeaders != "" {
		var headers map[string]string
		if err := json.Unmarshal([]byte(w.WebhookHeaders), &headers); err == nil {
			for key, value := range headers {
				req.Header.Set(key, value)
			}
		}
	}

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return 0, false, err.Error(), err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 10240))
	success := resp.StatusCode >= 200 && resp.StatusCode < 300

	var errMsg string
	if !success {
		errMsg = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	slog.Info("Test webhook sent",
		"watcher", w.Name,
		"url", w.WebhookURL,
		"status", resp.StatusCode,
		"success", success)

	return resp.StatusCode, success, errMsg, nil
}
