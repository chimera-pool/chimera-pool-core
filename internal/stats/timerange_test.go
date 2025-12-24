package stats

import (
	"testing"
	"time"
)

func TestParseTimeRange(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Duration
	}{
		{"1 hour", "1h", time.Hour},
		{"6 hours", "6h", 6 * time.Hour},
		{"24 hours", "24h", 24 * time.Hour},
		{"7 days", "7d", 7 * 24 * time.Hour},
		{"30 days", "30d", 30 * 24 * time.Hour},
		{"3 months", "3m", 90 * 24 * time.Hour},
		{"6 months", "6m", 180 * 24 * time.Hour},
		{"1 year", "1y", 365 * 24 * time.Hour},
		{"all time", "all", 10 * 365 * 24 * time.Hour}, // 10 years as "all"
		{"default for invalid", "invalid", 24 * time.Hour},
		{"empty string", "", 24 * time.Hour},
	}

	service := NewTimeRangeService()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.ParseRange(tt.input)
			if result != tt.expected {
				t.Errorf("ParseRange(%s) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetInterval(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"1 hour - 5 min intervals", time.Hour, "5 minutes"},
		{"6 hours - 15 min intervals", 6 * time.Hour, "15 minutes"},
		{"24 hours - 1 hour intervals", 24 * time.Hour, "hour"},
		{"7 days - 6 hour intervals", 7 * 24 * time.Hour, "6 hours"},
		{"30 days - 1 day intervals", 30 * 24 * time.Hour, "day"},
		{"3 months - 1 day intervals", 90 * 24 * time.Hour, "day"},
		{"6 months - 1 week intervals", 180 * 24 * time.Hour, "week"},
		{"1 year - 1 week intervals", 365 * 24 * time.Hour, "week"},
		{"10 years - 1 month intervals", 10 * 365 * 24 * time.Hour, "month"},
	}

	service := NewTimeRangeService()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.GetInterval(tt.duration)
			if result != tt.expected {
				t.Errorf("GetInterval(%v) = %s, expected %s", tt.duration, result, tt.expected)
			}
		})
	}
}

func TestGetPostgresInterval(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"1 hour", "1h", "1 hour"},
		{"6 hours", "6h", "6 hours"},
		{"24 hours", "24h", "24 hours"},
		{"7 days", "7d", "7 days"},
		{"30 days", "30d", "30 days"},
		{"3 months", "3m", "90 days"},
		{"6 months", "6m", "180 days"},
		{"1 year", "1y", "365 days"},
		{"all time", "all", "3650 days"},
	}

	service := NewTimeRangeService()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.GetPostgresInterval(tt.input)
			if result != tt.expected {
				t.Errorf("GetPostgresInterval(%s) = %s, expected %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetDateTrunc(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"1 hour - minute", "1h", "minute"},
		{"6 hours - hour", "6h", "hour"},
		{"24 hours - hour", "24h", "hour"},
		{"7 days - hour", "7d", "hour"},
		{"30 days - day", "30d", "day"},
		{"3 months - day", "3m", "day"},
		{"6 months - week", "6m", "week"},
		{"1 year - week", "1y", "week"},
		{"all time - month", "all", "month"},
	}

	service := NewTimeRangeService()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.GetDateTrunc(tt.input)
			if result != tt.expected {
				t.Errorf("GetDateTrunc(%s) = %s, expected %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetExpectedDataPoints(t *testing.T) {
	tests := []struct {
		name  string
		input string
		min   int
		max   int
	}{
		{"1 hour", "1h", 10, 15},
		{"6 hours", "6h", 20, 30},
		{"24 hours", "24h", 20, 30},
		{"7 days", "7d", 25, 35},
		{"30 days", "30d", 25, 35},
		{"3 months", "3m", 80, 100},
		{"6 months", "6m", 20, 30},
		{"1 year", "1y", 50, 55},
		{"all time", "all", 100, 130},
	}

	service := NewTimeRangeService()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.GetExpectedDataPoints(tt.input)
			if result < tt.min || result > tt.max {
				t.Errorf("GetExpectedDataPoints(%s) = %d, expected between %d and %d", tt.input, result, tt.min, tt.max)
			}
		})
	}
}
