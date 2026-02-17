package storage

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// Watcher represents a file system watcher configuration
type Watcher struct {
	ID             uint           `gorm:"primarykey" json:"id"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
	Name           string         `gorm:"uniqueIndex;not null" json:"name"`
	Path           string         `gorm:"not null" json:"path"`
	Mode           string         `gorm:"not null;default:watch" json:"mode"` // watch or poll
	Recursive      bool           `gorm:"default:true" json:"recursive"`
	PollInterval   string         `json:"poll_interval,omitempty"`  // e.g., "5s"
	Filters        string         `gorm:"type:text" json:"filters"` // JSON-encoded filters
	WebhookURL     string         `gorm:"not null" json:"webhook_url"`
	WebhookHeaders string         `gorm:"type:text" json:"webhook_headers"` // JSON-encoded headers
	Timeout        string         `json:"timeout"`                          // e.g., "10s"
	MaxRetries     int            `gorm:"default:3" json:"max_retries"`
	RetryBackoff   string         `json:"retry_backoff"` // e.g., "2s"
	AttachFile     bool           `gorm:"default:false" json:"attach_file"`
}

// Event represents a file system event
type Event struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	WatcherID uint           `gorm:"index;not null" json:"watcher_id"`
	Watcher   Watcher        `gorm:"foreignKey:WatcherID" json:"-"`
	EventType string         `gorm:"index;not null" json:"event_type"` // created, modified, deleted, renamed
	FileName  string         `gorm:"index;not null" json:"file_name"`
	FilePath  string         `gorm:"not null" json:"file_path"`
	FileSize  int64          `json:"file_size"`
	OldPath   string         `json:"old_path,omitempty"` // for rename events
}

// MarshalJSON customizes JSON output to include timestamp and watcher_name fields
func (e Event) MarshalJSON() ([]byte, error) {
	type EventAlias Event
	return json.Marshal(&struct {
		EventAlias
		Timestamp   time.Time `json:"timestamp"`
		WatcherName string    `json:"watcher_name,omitempty"`
	}{
		EventAlias:  EventAlias(e),
		Timestamp:   e.CreatedAt,
		WatcherName: e.Watcher.Name,
	})
}

// WebhookLog represents a webhook delivery attempt
type WebhookLog struct {
	ID             uint           `gorm:"primarykey" json:"id"`
	CreatedAt      time.Time      `json:"created_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
	EventID        uint           `gorm:"index;not null" json:"event_id"`
	Event          Event          `gorm:"foreignKey:EventID" json:"-"`
	WatcherID      uint           `gorm:"index;not null" json:"watcher_id"`
	StatusCode     int            `gorm:"index" json:"status_code"`
	Success        bool           `gorm:"index" json:"success"`
	Error          string         `gorm:"type:text" json:"error,omitempty"`
	RetryCount     int            `json:"retry_count"`
	ResponseTime   int64          `json:"response_time"` // milliseconds
	RequestPayload string         `gorm:"type:text" json:"request_payload"`
	ResponseBody   string         `gorm:"type:text" json:"response_body,omitempty"`
}
