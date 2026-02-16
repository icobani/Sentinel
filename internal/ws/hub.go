package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
)

// MessageType represents the type of WebSocket message
type MessageType string

const (
	MessageTypeFileEvent     MessageType = "file_event"
	MessageTypeWebhookResult MessageType = "webhook_result"
	MessageTypeWatcherStatus MessageType = "watcher_status"
	MessageTypeConfigChange  MessageType = "config_change"
)

// Message represents a WebSocket message
type Message struct {
	Type MessageType `json:"type"`
	Data interface{} `json:"data"`
}

// FileEventData represents file event message data
type FileEventData struct {
	WatcherID   uint   `json:"watcher_id"`
	WatcherName string `json:"watcher_name"`
	Event       string `json:"event"`
	FileName    string `json:"file_name"`
	FilePath    string `json:"file_path"`
	FileSize    int64  `json:"file_size"`
	OldPath     string `json:"old_path,omitempty"`
	Timestamp   string `json:"timestamp"`
}

// WebhookResultData represents webhook result message data
type WebhookResultData struct {
	WatcherID     uint   `json:"watcher_id"`
	WatcherName   string `json:"watcher_name"`
	Event         string `json:"event"`
	FileName      string `json:"file_name"`
	FilePath      string `json:"file_path"`
	FileSize      int64  `json:"file_size"`
	Timestamp     string `json:"timestamp"`
	WebhookStatus int    `json:"webhook_status"`
	WebhookError  string `json:"webhook_error,omitempty"`
	RetryCount    int    `json:"retry_count"`
	MaxRetries    int    `json:"max_retries"`
}

// WatcherStatusData represents watcher status change message data
type WatcherStatusData struct {
	WatcherID   uint   `json:"watcher_id"`
	WatcherName string `json:"watcher_name"`
	Status      string `json:"status"` // "started", "stopped", "error"
	Message     string `json:"message,omitempty"`
	Timestamp   string `json:"timestamp"`
}

// ConfigChangeData represents config change message data
type ConfigChangeData struct {
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

// Hub maintains the set of active clients and broadcasts messages to the clients
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Inbound messages from the clients
	broadcast chan []byte

	// Register requests from the clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Mutex for thread-safe access to clients map
	mu sync.RWMutex

	// Context for graceful shutdown
	ctx    context.Context
	cancel context.CancelFunc
}

var (
	globalHub *Hub
	hubOnce   sync.Once
)

// GetHub returns the singleton hub instance
func GetHub() *Hub {
	hubOnce.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		globalHub = &Hub{
			broadcast:  make(chan []byte, 256),
			register:   make(chan *Client),
			unregister: make(chan *Client),
			clients:    make(map[*Client]bool),
			ctx:        ctx,
			cancel:     cancel,
		}
		go globalHub.Run()
	})
	return globalHub
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	slog.Info("WebSocket hub starting")

	for {
		select {
		case <-h.ctx.Done():
			slog.Info("WebSocket hub shutting down")
			// Close all client connections
			h.mu.Lock()
			for client := range h.clients {
				close(client.send)
				delete(h.clients, client)
			}
			h.mu.Unlock()
			return

		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			slog.Debug("Client registered", "total_clients", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				slog.Debug("Client unregistered", "total_clients", len(h.clients))
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			clientCount := len(h.clients)
			h.mu.RUnlock()

			slog.Debug("Broadcasting message", "client_count", clientCount, "message_size", len(message))

			// Collect slow clients to avoid lock upgrade deadlock
			h.mu.RLock()
			var slowClients []*Client
			for client := range h.clients {
				select {
				case client.send <- message:
					// Message sent successfully
				default:
					// Client's send channel is full, mark for cleanup
					slowClients = append(slowClients, client)
				}
			}
			h.mu.RUnlock()

			// Clean up slow clients using unregister channel to avoid double-close
			for _, client := range slowClients {
				h.unregister <- client
				slog.Warn("Client send buffer full, unregistering")
			}
		}
	}
}

// BroadcastMessage broadcasts a message to all connected clients
func (h *Hub) BroadcastMessage(msgType MessageType, data interface{}) error {
	msg := Message{
		Type: msgType,
		Data: data,
	}

	jsonData, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// Non-blocking send to avoid blocking caller goroutines
	select {
	case h.broadcast <- jsonData:
		return nil
	default:
		slog.Warn("WebSocket broadcast channel full, dropping message", "type", msgType)
		return fmt.Errorf("broadcast channel full")
	}
}

// BroadcastFileEvent broadcasts a file event to all connected clients
func (h *Hub) BroadcastFileEvent(data FileEventData) {
	if err := h.BroadcastMessage(MessageTypeFileEvent, data); err != nil {
		slog.Error("Failed to broadcast file event", "error", err)
	}
}

// BroadcastWebhookResult broadcasts a webhook result to all connected clients
func (h *Hub) BroadcastWebhookResult(data WebhookResultData) {
	if err := h.BroadcastMessage(MessageTypeWebhookResult, data); err != nil {
		slog.Error("Failed to broadcast webhook result", "error", err)
	}
}

// BroadcastWatcherStatus broadcasts a watcher status change to all connected clients
func (h *Hub) BroadcastWatcherStatus(data WatcherStatusData) {
	if err := h.BroadcastMessage(MessageTypeWatcherStatus, data); err != nil {
		slog.Error("Failed to broadcast watcher status", "error", err)
	}
}

// BroadcastConfigChange broadcasts a config change notification to all connected clients
func (h *Hub) BroadcastConfigChange(data ConfigChangeData) {
	if err := h.BroadcastMessage(MessageTypeConfigChange, data); err != nil {
		slog.Error("Failed to broadcast config change", "error", err)
	}
}

// GetClientCount returns the number of connected clients
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// Register registers a new client with the hub
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister unregisters a client from the hub
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// Stop stops the hub and closes all connections
func (h *Hub) Stop() {
	if h.cancel != nil {
		h.cancel()
	}
}
