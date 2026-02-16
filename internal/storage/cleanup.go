package storage

import (
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	retentionDuration time.Duration
	lastCleanupTime   time.Time
	cleanupMu         sync.Mutex
	cleanupInterval   = 1 * time.Hour // Don't run cleanup more often than this
)

// InitCleanup configures the retention duration from config
func InitCleanup(retention string) {
	retentionDuration = parseRetention(retention)
	slog.Info("Database retention configured", "retention", retention, "duration", retentionDuration.String())
}

// CleanupIfNeeded runs cleanup only if enough time has passed since the last one.
// Called after each event insert - safe to call frequently, throttled internally.
func CleanupIfNeeded() {
	if retentionDuration <= 0 {
		return
	}

	cleanupMu.Lock()
	if time.Since(lastCleanupTime) < cleanupInterval {
		cleanupMu.Unlock()
		return
	}
	lastCleanupTime = time.Now()
	cleanupMu.Unlock()

	// Delete old webhook logs first (foreign key dependency)
	webhookCount, err := DeleteOldWebhookLogs(retentionDuration)
	if err != nil {
		slog.Error("Failed to cleanup old webhook logs", "error", err)
	}

	// Delete old events
	eventCount, err := DeleteOldEvents(retentionDuration)
	if err != nil {
		slog.Error("Failed to cleanup old events", "error", err)
	}

	if webhookCount > 0 || eventCount > 0 {
		slog.Info("Database cleanup completed", "events_deleted", eventCount, "webhook_logs_deleted", webhookCount)
	}
}

// parseRetention parses retention string like "30d", "7d", "24h" into time.Duration
func parseRetention(s string) time.Duration {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 30 * 24 * time.Hour // default 30 days
	}

	// Try standard Go duration first (e.g., "720h", "24h")
	if d, err := time.ParseDuration(s); err == nil {
		return d
	}

	// Handle "Xd" format (days)
	if strings.HasSuffix(s, "d") {
		daysStr := strings.TrimSuffix(s, "d")
		if days, err := strconv.Atoi(daysStr); err == nil && days > 0 {
			return time.Duration(days) * 24 * time.Hour
		}
	}

	return 0
}
