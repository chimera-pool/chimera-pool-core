// Package config provides unified configuration utilities for the Chimeria Pool.
// This eliminates duplicate getEnv functions across multiple packages.
package config

import (
	"os"
	"strconv"
	"time"
)

// GetEnv returns the value of an environment variable or a default value.
// This is the canonical implementation - use this instead of local getEnv functions.
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetEnvInt returns an integer environment variable or a default value.
func GetEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// GetEnvInt64 returns an int64 environment variable or a default value.
func GetEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// GetEnvFloat64 returns a float64 environment variable or a default value.
func GetEnvFloat64(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			return floatVal
		}
	}
	return defaultValue
}

// GetEnvBool returns a boolean environment variable or a default value.
func GetEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

// GetEnvDuration returns a duration environment variable or a default value.
// Accepts formats like "30s", "5m", "1h".
func GetEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// MustGetEnv returns the value of an environment variable or panics if not set.
// Use sparingly - only for truly required configuration.
func MustGetEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic("required environment variable not set: " + key)
	}
	return value
}

// GetEnvSlice returns a slice from a comma-separated environment variable.
func GetEnvSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		// Simple split - for more complex parsing, use a dedicated library
		result := []string{}
		for _, item := range splitAndTrim(value, ",") {
			if item != "" {
				result = append(result, item)
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	return defaultValue
}

// splitAndTrim splits a string and trims whitespace from each part.
func splitAndTrim(s, sep string) []string {
	parts := []string{}
	start := 0
	for i := 0; i < len(s); i++ {
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			part := trimSpace(s[start:i])
			parts = append(parts, part)
			start = i + len(sep)
			i += len(sep) - 1
		}
	}
	parts = append(parts, trimSpace(s[start:]))
	return parts
}

// trimSpace removes leading and trailing whitespace.
func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}
