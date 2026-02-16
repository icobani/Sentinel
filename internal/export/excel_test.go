package export

import (
	"testing"
	"time"

	"github.com/sentinel/sentinel/internal/storage"
)

func TestExportLogsToExcel(t *testing.T) {
	tests := []struct {
		name         string
		watcherName  string
		events       []storage.Event
		webhookLogs  map[uint]*storage.WebhookLog
		wantErr      bool
		expectedRows int // header + data rows
	}{
		{
			name:         "empty events",
			watcherName:  "test-watcher",
			events:       []storage.Event{},
			webhookLogs:  map[uint]*storage.WebhookLog{},
			wantErr:      false,
			expectedRows: 1, // only header
		},
		{
			name:        "single event without webhook log",
			watcherName: "uploads",
			events: []storage.Event{
				{
					ID:        1,
					CreatedAt: time.Now(),
					EventType: "created",
					FileName:  "test.txt",
					FilePath:  "/data/test.txt",
					FileSize:  1024,
				},
			},
			webhookLogs:  map[uint]*storage.WebhookLog{},
			wantErr:      false,
			expectedRows: 2, // header + 1 data row
		},
		{
			name:        "single event with webhook log",
			watcherName: "reports",
			events: []storage.Event{
				{
					ID:        1,
					CreatedAt: time.Now(),
					EventType: "modified",
					FileName:  "report.pdf",
					FilePath:  "/data/reports/report.pdf",
					FileSize:  2048576,
				},
			},
			webhookLogs: map[uint]*storage.WebhookLog{
				1: {
					ID:           1,
					EventID:      1,
					StatusCode:   200,
					Success:      true,
					RetryCount:   0,
					ResponseTime: 150,
				},
			},
			wantErr:      false,
			expectedRows: 2, // header + 1 data row
		},
		{
			name:        "multiple events with mixed webhook logs",
			watcherName: "config-sync",
			events: []storage.Event{
				{
					ID:        1,
					CreatedAt: time.Now(),
					EventType: "created",
					FileName:  "config.yaml",
					FilePath:  "/config/config.yaml",
					FileSize:  512,
				},
				{
					ID:        2,
					CreatedAt: time.Now(),
					EventType: "deleted",
					FileName:  "old-config.yaml",
					FilePath:  "/config/old-config.yaml",
					FileSize:  256,
				},
				{
					ID:        3,
					CreatedAt: time.Now(),
					EventType: "renamed",
					FileName:  "settings.json",
					FilePath:  "/config/settings.json",
					OldPath:   "/config/old-settings.json",
					FileSize:  768,
				},
			},
			webhookLogs: map[uint]*storage.WebhookLog{
				1: {
					ID:           1,
					EventID:      1,
					StatusCode:   200,
					Success:      true,
					RetryCount:   0,
					ResponseTime: 100,
				},
				3: {
					ID:           2,
					EventID:      3,
					StatusCode:   500,
					Success:      false,
					Error:        "Internal Server Error",
					RetryCount:   3,
					ResponseTime: 5000,
				},
			},
			wantErr:      false,
			expectedRows: 4, // header + 3 data rows
		},
		{
			name:        "special characters in watcher name",
			watcherName: "test/watcher:name*with?chars[brackets]",
			events: []storage.Event{
				{
					ID:        1,
					CreatedAt: time.Now(),
					EventType: "created",
					FileName:  "test.txt",
					FilePath:  "/data/test.txt",
					FileSize:  100,
				},
			},
			webhookLogs:  map[uint]*storage.WebhookLog{},
			wantErr:      false,
			expectedRows: 2,
		},
		{
			name:        "long watcher name (> 31 chars)",
			watcherName: "this-is-a-very-long-watcher-name-that-exceeds-the-limit",
			events: []storage.Event{
				{
					ID:        1,
					CreatedAt: time.Now(),
					EventType: "created",
					FileName:  "test.txt",
					FilePath:  "/data/test.txt",
					FileSize:  100,
				},
			},
			webhookLogs:  map[uint]*storage.WebhookLog{},
			wantErr:      false,
			expectedRows: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := ExportLogsToExcel(tt.watcherName, tt.events, tt.webhookLogs)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExportLogsToExcel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				defer file.Close()

				// Verify sheet exists
				sheetName := file.GetSheetName(0)
				if sheetName == "" {
					t.Error("Expected sheet name to be set")
				}

				// Verify sheet name is valid (max 31 chars)
				if len(sheetName) > 31 {
					t.Errorf("Sheet name too long: %d chars (max 31)", len(sheetName))
				}

				// Verify rows exist
				rows, err := file.GetRows(sheetName)
				if err != nil {
					t.Errorf("Failed to get rows: %v", err)
					return
				}

				if len(rows) != tt.expectedRows {
					t.Errorf("Expected %d rows, got %d", tt.expectedRows, len(rows))
				}

				// Verify header row
				if len(rows) > 0 {
					headerRow := rows[0]
					expectedHeaders := []string{
						"ID",
						"Event Type",
						"File Name",
						"File Path",
						"File Size",
						"Old Path",
						"Created At",
						"Webhook Status",
						"Webhook Error",
						"Retry Count",
						"Response Time (ms)",
					}

					if len(headerRow) != len(expectedHeaders) {
						t.Errorf("Expected %d columns, got %d", len(expectedHeaders), len(headerRow))
					}

					for i, expected := range expectedHeaders {
						if i < len(headerRow) && headerRow[i] != expected {
							t.Errorf("Column %d: expected %q, got %q", i, expected, headerRow[i])
						}
					}
				}
			}
		})
	}
}

func TestGenerateFileName(t *testing.T) {
	tests := []struct {
		name        string
		watcherName string
		wantPrefix  string
		wantSuffix  string
	}{
		{
			name:        "simple name",
			watcherName: "uploads",
			wantPrefix:  "sentinel_uploads_",
			wantSuffix:  ".xlsx",
		},
		{
			name:        "name with spaces",
			watcherName: "network reports",
			wantPrefix:  "sentinel_network_reports_",
			wantSuffix:  ".xlsx",
		},
		{
			name:        "name with special characters",
			watcherName: "config/sync:test",
			wantPrefix:  "sentinel_config_sync_test_",
			wantSuffix:  ".xlsx",
		},
		{
			name:        "uppercase name",
			watcherName: "UPLOADS",
			wantPrefix:  "sentinel_uploads_",
			wantSuffix:  ".xlsx",
		},
		{
			name:        "empty name",
			watcherName: "",
			wantPrefix:  "sentinel_export_",
			wantSuffix:  ".xlsx",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename := GenerateFileName(tt.watcherName)

			if filename[:len(tt.wantPrefix)] != tt.wantPrefix {
				t.Errorf("GenerateFileName() prefix = %q, want %q", filename[:len(tt.wantPrefix)], tt.wantPrefix)
			}

			if filename[len(filename)-len(tt.wantSuffix):] != tt.wantSuffix {
				t.Errorf("GenerateFileName() suffix = %q, want %q", filename[len(filename)-len(tt.wantSuffix):], tt.wantSuffix)
			}

			// Verify timestamp format is present (YYYY-MM-DD_HHMMSS)
			if len(filename) < len(tt.wantPrefix)+len(tt.wantSuffix)+15 {
				t.Errorf("GenerateFileName() = %q, seems too short to contain timestamp", filename)
			}
		})
	}
}

func TestSanitizeSheetName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple name",
			input: "uploads",
			want:  "uploads",
		},
		{
			name:  "name with invalid characters",
			input: "test/watcher:name*with?chars[brackets]",
			want:  "testwatchernamewithcharsbracket", // Truncated to 31 chars
		},
		{
			name:  "name exceeding 31 chars",
			input: "this-is-a-very-long-watcher-name-that-exceeds-the-limit",
			want:  "this-is-a-very-long-watcher-nam", // Truncated to 31 chars
		},
		{
			name:  "empty name",
			input: "",
			want:  "Logs",
		},
		{
			name:  "name with only invalid characters",
			input: "///***???",
			want:  "Logs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeSheetName(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeSheetName() = %q, want %q", got, tt.want)
			}

			// Verify length constraint
			if len(got) > 31 {
				t.Errorf("sanitizeSheetName() returned name longer than 31 chars: %d", len(got))
			}
		})
	}
}

func TestSanitizeFileName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple name",
			input: "uploads",
			want:  "uploads",
		},
		{
			name:  "name with spaces",
			input: "network reports",
			want:  "network_reports",
		},
		{
			name:  "name with special characters",
			input: "test/file:name*with?chars",
			want:  "test_file_name_with_chars",
		},
		{
			name:  "uppercase name",
			input: "UPLOADS",
			want:  "uploads",
		},
		{
			name:  "name with consecutive underscores",
			input: "test___file___name",
			want:  "test_file_name",
		},
		{
			name:  "name with leading/trailing underscores",
			input: "_test_file_",
			want:  "test_file",
		},
		{
			name:  "empty name",
			input: "",
			want:  "export",
		},
		{
			name:  "name with only special characters",
			input: "///***???",
			want:  "export",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeFileName(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeFileName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{
			name:  "zero bytes",
			bytes: 0,
			want:  "0 B",
		},
		{
			name:  "bytes only",
			bytes: 500,
			want:  "500 B",
		},
		{
			name:  "kilobytes",
			bytes: 1024,
			want:  "1.00 KB",
		},
		{
			name:  "megabytes",
			bytes: 1024 * 1024,
			want:  "1.00 MB",
		},
		{
			name:  "gigabytes",
			bytes: 1024 * 1024 * 1024,
			want:  "1.00 GB",
		},
		{
			name:  "terabytes",
			bytes: 1024 * 1024 * 1024 * 1024,
			want:  "1.00 TB",
		},
		{
			name:  "fractional KB",
			bytes: 1536,
			want:  "1.50 KB",
		},
		{
			name:  "fractional MB",
			bytes: 2 * 1024 * 1024,
			want:  "2.00 MB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatFileSize(tt.bytes)
			if got != tt.want {
				t.Errorf("formatFileSize() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestExcelFileStructure verifies the Excel file has correct structure
func TestExcelFileStructure(t *testing.T) {
	// Create sample data
	events := []storage.Event{
		{
			ID:        1,
			CreatedAt: time.Date(2026, 2, 14, 10, 30, 0, 0, time.UTC),
			EventType: "created",
			FileName:  "report.pdf",
			FilePath:  "/data/reports/report.pdf",
			FileSize:  1048576, // 1 MB
		},
	}

	webhookLogs := map[uint]*storage.WebhookLog{
		1: {
			ID:           1,
			EventID:      1,
			StatusCode:   200,
			Success:      true,
			RetryCount:   0,
			ResponseTime: 150,
		},
	}

	file, err := ExportLogsToExcel("test-watcher", events, webhookLogs)
	if err != nil {
		t.Fatalf("Failed to create Excel file: %v", err)
	}
	defer file.Close()

	sheetName := file.GetSheetName(0)

	// Verify specific cell values
	tests := []struct {
		cell  string
		value string
	}{
		{"A1", "ID"},
		{"B1", "Event Type"},
		{"C1", "File Name"},
		{"A2", "1"},
		{"B2", "created"},
		{"C2", "report.pdf"},
		{"E2", "1.00 MB"},
		{"H2", "200"},
		{"J2", "0"},
		{"K2", "150"},
	}

	for _, tt := range tests {
		cellValue, err := file.GetCellValue(sheetName, tt.cell)
		if err != nil {
			t.Errorf("Failed to get cell %s: %v", tt.cell, err)
			continue
		}

		if cellValue != tt.value {
			t.Errorf("Cell %s = %q, want %q", tt.cell, cellValue, tt.value)
		}
	}

	// Verify freeze panes is set
	panes, err := file.GetPanes(sheetName)
	if err != nil {
		t.Errorf("Failed to get panes: %v", err)
	} else if !panes.Freeze {
		t.Error("Expected freeze panes to be enabled")
	}
}

// BenchmarkExportLogsToExcel measures performance of Excel generation
func BenchmarkExportLogsToExcel(b *testing.B) {
	// Create sample data with 100 events
	events := make([]storage.Event, 100)
	webhookLogs := make(map[uint]*storage.WebhookLog)

	for i := 0; i < 100; i++ {
		events[i] = storage.Event{
			ID:        uint(i + 1),
			CreatedAt: time.Now(),
			EventType: "created",
			FileName:  "file.txt",
			FilePath:  "/data/file.txt",
			FileSize:  1024,
		}

		webhookLogs[uint(i+1)] = &storage.WebhookLog{
			ID:           uint(i + 1),
			EventID:      uint(i + 1),
			StatusCode:   200,
			Success:      true,
			RetryCount:   0,
			ResponseTime: 100,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		file, err := ExportLogsToExcel("benchmark-watcher", events, webhookLogs)
		if err != nil {
			b.Fatalf("Failed to create Excel file: %v", err)
		}
		file.Close()
	}
}
