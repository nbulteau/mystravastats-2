package helpers

import (
	"testing"
)

func TestFormatDate(t *testing.T) {
	tests := []struct {
		name     string
		dateStr  string
		expected string
	}{
		{
			name:     "Valid date",
			dateStr:  "2024-07-22T10:30:00Z",
			expected: "Mon 22 July 2024 - 10:30",
		},
		{
			name:     "Invalid date",
			dateStr:  "invalid-date",
			expected: "",
		},
		{
			name:     "Empty date string",
			dateStr:  "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDate(tt.dateStr)
			if result != tt.expected {
				t.Errorf("formatDate(%s) = %s, expected %s", tt.dateStr, result, tt.expected)
			}
		})
	}
}

func TestFormatSeconds(t *testing.T) {
	tests := []struct {
		name     string
		seconds  int
		expected string
	}{
		{
			name:     "Zero seconds",
			seconds:  0,
			expected: "00s",
		},
		{
			name:     "Seconds only",
			seconds:  59,
			expected: "59s",
		},
		{
			name:     "Minutes and seconds",
			seconds:  125,
			expected: "02m 05s",
		},
		{
			name:     "Hours, minutes, and seconds",
			seconds:  3725,
			expected: "01h 02m 05s",
		},
		{
			name:     "Hours only",
			seconds:  3600,
			expected: " 1h",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatSeconds(tt.seconds)
			if result != tt.expected {
				t.Errorf("FormatSeconds(%d) = %s, expected %s", tt.seconds, result, tt.expected)
			}
		})
	}
}

func TestFormatSecondsFloat(t *testing.T) {
	tests := []struct {
		name     string
		seconds  float64
		expected string
	}{
		{
			name:     "Zero seconds",
			seconds:  0.0,
			expected: "0'00",
		},
		{
			name:     "Seconds with hundredths",
			seconds:  5.55,
			expected: "0'05",
		},
		{
			name:     "Minutes and seconds",
			seconds:  125.75,
			expected: "2'05",
		},
		{
			name:     "Rounding up seconds",
			seconds:  59.99,
			expected: "0'59",
		},
		{
			name:     "Rounding up to next minute",
			seconds:  59.995,
			expected: "1'00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatSecondsFloat(tt.seconds)
			if result != tt.expected {
				t.Errorf("FormatSecondsFloat(%f) = %s, expected %s", tt.seconds, result, tt.expected)
			}
		})
	}
}
