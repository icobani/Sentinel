package export

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/sentinel/sentinel/internal/storage"
	"github.com/xuri/excelize/v2"
)

// ExportLogsToExcel generates an Excel file from event logs with associated webhook data
// Note: Caller is responsible for closing the returned file
func ExportLogsToExcel(watcherName string, events []storage.Event, webhookLogs map[uint]*storage.WebhookLog) (*excelize.File, error) {
	f := excelize.NewFile()

	// Create sheet name from watcher name (max 31 chars, Excel limitation)
	sheetName := sanitizeSheetName(watcherName)

	// Rename default sheet to watcher name
	defaultSheet := f.GetSheetName(0)
	if err := f.SetSheetName(defaultSheet, sheetName); err != nil {
		return nil, fmt.Errorf("failed to set sheet name: %w", err)
	}

	// Define header style with bold font and blue background
	headerStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:  true,
			Size:  12,
			Color: "#FFFFFF",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Pattern: 1,
			Color:   []string{"#4472C4"}, // Blue background
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create header style: %w", err)
	}

	// Define data row style (with borders)
	dataStyle, err := f.NewStyle(&excelize.Style{
		Border: []excelize.Border{
			{Type: "left", Color: "D3D3D3", Style: 1},
			{Type: "top", Color: "D3D3D3", Style: 1},
			{Type: "bottom", Color: "D3D3D3", Style: 1},
			{Type: "right", Color: "D3D3D3", Style: 1},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create data style: %w", err)
	}

	// Set column headers
	headers := []string{
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

	for col, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, 1)
		if err := f.SetCellValue(sheetName, cell, header); err != nil {
			return nil, fmt.Errorf("failed to set header value: %w", err)
		}
		if err := f.SetCellStyle(sheetName, cell, cell, headerStyle); err != nil {
			return nil, fmt.Errorf("failed to set header style: %w", err)
		}
	}

	// Set column widths for better readability
	columnWidths := map[string]float64{
		"A": 8,  // ID
		"B": 15, // Event Type
		"C": 25, // File Name
		"D": 50, // File Path
		"E": 12, // File Size
		"F": 25, // Old Path
		"G": 20, // Created At
		"H": 15, // Webhook Status
		"I": 40, // Webhook Error
		"J": 12, // Retry Count
		"K": 18, // Response Time
	}

	for col, width := range columnWidths {
		if err := f.SetColWidth(sheetName, col, col, width); err != nil {
			return nil, fmt.Errorf("failed to set column width: %w", err)
		}
	}

	// Populate data rows
	for i, event := range events {
		row := i + 2 // Start from row 2 (after header)

		// ID
		idCell, _ := excelize.CoordinatesToCellName(1, row)
		if err := f.SetCellValue(sheetName, idCell, event.ID); err != nil {
			return nil, fmt.Errorf("failed to set ID value: %w", err)
		}
		if err := f.SetCellStyle(sheetName, idCell, idCell, dataStyle); err != nil {
			return nil, fmt.Errorf("failed to set cell style: %w", err)
		}

		// Event Type
		eventTypeCell, _ := excelize.CoordinatesToCellName(2, row)
		if err := f.SetCellValue(sheetName, eventTypeCell, event.EventType); err != nil {
			return nil, fmt.Errorf("failed to set event type: %w", err)
		}
		if err := f.SetCellStyle(sheetName, eventTypeCell, eventTypeCell, dataStyle); err != nil {
			return nil, fmt.Errorf("failed to set cell style: %w", err)
		}

		// File Name
		fileNameCell, _ := excelize.CoordinatesToCellName(3, row)
		if err := f.SetCellValue(sheetName, fileNameCell, event.FileName); err != nil {
			return nil, fmt.Errorf("failed to set file name: %w", err)
		}
		if err := f.SetCellStyle(sheetName, fileNameCell, fileNameCell, dataStyle); err != nil {
			return nil, fmt.Errorf("failed to set cell style: %w", err)
		}

		// File Path
		filePathCell, _ := excelize.CoordinatesToCellName(4, row)
		if err := f.SetCellValue(sheetName, filePathCell, event.FilePath); err != nil {
			return nil, fmt.Errorf("failed to set file path: %w", err)
		}
		if err := f.SetCellStyle(sheetName, filePathCell, filePathCell, dataStyle); err != nil {
			return nil, fmt.Errorf("failed to set cell style: %w", err)
		}

		// File Size (human-readable)
		sizeCell, _ := excelize.CoordinatesToCellName(5, row)
		fileSizeStr := formatFileSize(event.FileSize)
		if err := f.SetCellValue(sheetName, sizeCell, fileSizeStr); err != nil {
			return nil, fmt.Errorf("failed to set file size: %w", err)
		}
		if err := f.SetCellStyle(sheetName, sizeCell, sizeCell, dataStyle); err != nil {
			return nil, fmt.Errorf("failed to set cell style: %w", err)
		}

		// Old Path
		oldPathCell, _ := excelize.CoordinatesToCellName(6, row)
		if err := f.SetCellValue(sheetName, oldPathCell, event.OldPath); err != nil {
			return nil, fmt.Errorf("failed to set old path: %w", err)
		}
		if err := f.SetCellStyle(sheetName, oldPathCell, oldPathCell, dataStyle); err != nil {
			return nil, fmt.Errorf("failed to set cell style: %w", err)
		}

		// Created At (formatted as string for consistency)
		dateCell, _ := excelize.CoordinatesToCellName(7, row)
		createdAtStr := event.CreatedAt.Format("2006-01-02 15:04:05")
		if err := f.SetCellValue(sheetName, dateCell, createdAtStr); err != nil {
			return nil, fmt.Errorf("failed to set date value: %w", err)
		}
		if err := f.SetCellStyle(sheetName, dateCell, dateCell, dataStyle); err != nil {
			return nil, fmt.Errorf("failed to set cell style: %w", err)
		}

		// Webhook data (if available)
		if webhookLog, ok := webhookLogs[event.ID]; ok {
			// Webhook Status
			statusCell, _ := excelize.CoordinatesToCellName(8, row)
			if err := f.SetCellValue(sheetName, statusCell, webhookLog.StatusCode); err != nil {
				return nil, fmt.Errorf("failed to set webhook status: %w", err)
			}
			if err := f.SetCellStyle(sheetName, statusCell, statusCell, dataStyle); err != nil {
				return nil, fmt.Errorf("failed to set cell style: %w", err)
			}

			// Webhook Error
			errorCell, _ := excelize.CoordinatesToCellName(9, row)
			if err := f.SetCellValue(sheetName, errorCell, webhookLog.Error); err != nil {
				return nil, fmt.Errorf("failed to set webhook error: %w", err)
			}
			if err := f.SetCellStyle(sheetName, errorCell, errorCell, dataStyle); err != nil {
				return nil, fmt.Errorf("failed to set cell style: %w", err)
			}

			// Retry Count
			retryCell, _ := excelize.CoordinatesToCellName(10, row)
			if err := f.SetCellValue(sheetName, retryCell, webhookLog.RetryCount); err != nil {
				return nil, fmt.Errorf("failed to set retry count: %w", err)
			}
			if err := f.SetCellStyle(sheetName, retryCell, retryCell, dataStyle); err != nil {
				return nil, fmt.Errorf("failed to set cell style: %w", err)
			}

			// Response Time
			responseTimeCell, _ := excelize.CoordinatesToCellName(11, row)
			if err := f.SetCellValue(sheetName, responseTimeCell, webhookLog.ResponseTime); err != nil {
				return nil, fmt.Errorf("failed to set response time: %w", err)
			}
			if err := f.SetCellStyle(sheetName, responseTimeCell, responseTimeCell, dataStyle); err != nil {
				return nil, fmt.Errorf("failed to set cell style: %w", err)
			}
		} else {
			// No webhook data available - set empty values with borders
			for col := 8; col <= 11; col++ {
				cell, _ := excelize.CoordinatesToCellName(col, row)
				if err := f.SetCellValue(sheetName, cell, ""); err != nil {
					return nil, fmt.Errorf("failed to set empty cell: %w", err)
				}
				if err := f.SetCellStyle(sheetName, cell, cell, dataStyle); err != nil {
					return nil, fmt.Errorf("failed to set cell style: %w", err)
				}
			}
		}
	}

	// Freeze the first row (header)
	if err := f.SetPanes(sheetName, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	}); err != nil {
		return nil, fmt.Errorf("failed to freeze header row: %w", err)
	}

	slog.Info("Excel file generated successfully",
		"watcher_name", watcherName,
		"event_count", len(events),
		"sheet_name", sheetName,
	)

	return f, nil
}

// GenerateFileName creates a filename for the Excel export
// Format: sentinel_{watcher_name}_{date}_{time}.xlsx
func GenerateFileName(watcherName string) string {
	// Sanitize watcher name for filename
	sanitized := sanitizeFileName(watcherName)

	// Format: sentinel_uploads_2025-02-14_103001.xlsx
	timestamp := time.Now().Format("2006-01-02_150405")
	return fmt.Sprintf("sentinel_%s_%s.xlsx", sanitized, timestamp)
}

// sanitizeSheetName sanitizes a string for use as an Excel sheet name
// Excel sheet names must be <= 31 chars and cannot contain: \ / ? * [ ]
func sanitizeSheetName(name string) string {
	// Remove invalid characters
	replacer := strings.NewReplacer(
		"\\", "",
		"/", "",
		"?", "",
		"*", "",
		"[", "",
		"]", "",
		":", "",
	)
	sanitized := replacer.Replace(name)

	// Trim to max 31 characters
	if len(sanitized) > 31 {
		sanitized = sanitized[:31]
	}

	// If empty after sanitization, use default
	if sanitized == "" {
		sanitized = "Logs"
	}

	return sanitized
}

// sanitizeFileName sanitizes a string for use in a filename
func sanitizeFileName(name string) string {
	// Replace spaces and special characters with underscores
	replacer := strings.NewReplacer(
		" ", "_",
		"\\", "_",
		"/", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	sanitized := replacer.Replace(name)

	// Convert to lowercase
	sanitized = strings.ToLower(sanitized)

	// Remove consecutive underscores
	for strings.Contains(sanitized, "__") {
		sanitized = strings.ReplaceAll(sanitized, "__", "_")
	}

	// Trim underscores from start and end
	sanitized = strings.Trim(sanitized, "_")

	// If empty after sanitization, use default
	if sanitized == "" {
		sanitized = "export"
	}

	return sanitized
}

// formatFileSize formats a file size in bytes to a human-readable string
func formatFileSize(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}

	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f TB", float64(bytes)/float64(TB))
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
