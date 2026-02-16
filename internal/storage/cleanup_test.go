package storage

import (
	"testing"
	"time"
)

func TestParseRetention(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Duration
	}{
		{
			name:     "empty string defaults to 30 days",
			input:    "",
			expected: 30 * 24 * time.Hour,
		},
		{
			name:     "30 days format",
			input:    "30d",
			expected: 30 * 24 * time.Hour,
		},
		{
			name:     "7 days format",
			input:    "7d",
			expected: 7 * 24 * time.Hour,
		},
		{
			name:     "1 day format",
			input:    "1d",
			expected: 24 * time.Hour,
		},
		{
			name:     "24 hours Go duration format",
			input:    "24h",
			expected: 24 * time.Hour,
		},
		{
			name:     "720 hours Go duration format",
			input:    "720h",
			expected: 720 * time.Hour,
		},
		{
			name:     "case insensitive days",
			input:    "30D",
			expected: 30 * 24 * time.Hour,
		},
		{
			name:     "with whitespace",
			input:    "  30d  ",
			expected: 30 * 24 * time.Hour,
		},
		{
			name:     "invalid format returns zero",
			input:    "invalid",
			expected: 0,
		},
		{
			name:     "negative days returns zero",
			input:    "-5d",
			expected: 0,
		},
		{
			name:     "zero days returns zero",
			input:    "0d",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseRetention(tt.input)
			if result != tt.expected {
				t.Errorf("parseRetention(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}
