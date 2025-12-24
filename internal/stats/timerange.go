// Package stats provides statistics calculation utilities for the mining pool
package stats

import (
	"time"
)

// TimeRangeService provides time range parsing and interval calculation
type TimeRangeService interface {
	// ParseRange converts a time range string to duration
	ParseRange(rangeStr string) time.Duration

	// GetInterval returns the appropriate grouping interval for a duration
	GetInterval(duration time.Duration) string

	// GetPostgresInterval returns the interval string for PostgreSQL queries
	GetPostgresInterval(rangeStr string) string

	// GetDateTrunc returns the date_trunc precision for PostgreSQL
	GetDateTrunc(rangeStr string) string

	// GetExpectedDataPoints returns expected number of data points for a range
	GetExpectedDataPoints(rangeStr string) int
}

// timeRangeServiceImpl implements TimeRangeService
type timeRangeServiceImpl struct{}

// NewTimeRangeService creates a new time range service
func NewTimeRangeService() TimeRangeService {
	return &timeRangeServiceImpl{}
}

// ParseRange converts a time range string to duration
func (s *timeRangeServiceImpl) ParseRange(rangeStr string) time.Duration {
	switch rangeStr {
	case "1h":
		return time.Hour
	case "6h":
		return 6 * time.Hour
	case "24h":
		return 24 * time.Hour
	case "7d":
		return 7 * 24 * time.Hour
	case "30d":
		return 30 * 24 * time.Hour
	case "3m":
		return 90 * 24 * time.Hour
	case "6m":
		return 180 * 24 * time.Hour
	case "1y":
		return 365 * 24 * time.Hour
	case "all":
		return 10 * 365 * 24 * time.Hour // 10 years as "all time"
	default:
		return 24 * time.Hour // Default to 24 hours
	}
}

// GetInterval returns the appropriate grouping interval for a duration
func (s *timeRangeServiceImpl) GetInterval(duration time.Duration) string {
	switch {
	case duration <= time.Hour:
		return "5 minutes"
	case duration <= 6*time.Hour:
		return "15 minutes"
	case duration <= 24*time.Hour:
		return "hour"
	case duration <= 7*24*time.Hour:
		return "6 hours"
	case duration <= 90*24*time.Hour:
		return "day"
	case duration <= 180*24*time.Hour:
		return "week"
	case duration <= 365*24*time.Hour:
		return "week"
	default:
		return "month"
	}
}

// GetPostgresInterval returns the interval string for PostgreSQL queries
func (s *timeRangeServiceImpl) GetPostgresInterval(rangeStr string) string {
	switch rangeStr {
	case "1h":
		return "1 hour"
	case "6h":
		return "6 hours"
	case "24h":
		return "24 hours"
	case "7d":
		return "7 days"
	case "30d":
		return "30 days"
	case "3m":
		return "90 days"
	case "6m":
		return "180 days"
	case "1y":
		return "365 days"
	case "all":
		return "3650 days" // 10 years
	default:
		return "24 hours"
	}
}

// GetDateTrunc returns the date_trunc precision for PostgreSQL
func (s *timeRangeServiceImpl) GetDateTrunc(rangeStr string) string {
	switch rangeStr {
	case "1h":
		return "minute"
	case "6h":
		return "hour"
	case "24h":
		return "hour"
	case "7d":
		return "hour"
	case "30d":
		return "day"
	case "3m":
		return "day"
	case "6m":
		return "week"
	case "1y":
		return "week"
	case "all":
		return "month"
	default:
		return "hour"
	}
}

// GetExpectedDataPoints returns expected number of data points for a range
func (s *timeRangeServiceImpl) GetExpectedDataPoints(rangeStr string) int {
	switch rangeStr {
	case "1h":
		return 12 // 5-minute intervals
	case "6h":
		return 24 // 15-minute intervals
	case "24h":
		return 24 // hourly intervals
	case "7d":
		return 28 // 6-hour intervals
	case "30d":
		return 30 // daily intervals
	case "3m":
		return 90 // daily intervals
	case "6m":
		return 26 // weekly intervals
	case "1y":
		return 52 // weekly intervals
	case "all":
		return 120 // monthly intervals (10 years)
	default:
		return 24
	}
}
