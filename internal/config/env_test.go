package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetEnv(t *testing.T) {
	t.Run("returns env value when set", func(t *testing.T) {
		os.Setenv("TEST_VAR", "test_value")
		defer os.Unsetenv("TEST_VAR")

		result := GetEnv("TEST_VAR", "default")
		assert.Equal(t, "test_value", result)
	})

	t.Run("returns default when not set", func(t *testing.T) {
		os.Unsetenv("TEST_VAR_UNSET")

		result := GetEnv("TEST_VAR_UNSET", "default_value")
		assert.Equal(t, "default_value", result)
	})
}

func TestGetEnvInt(t *testing.T) {
	t.Run("returns int value when set", func(t *testing.T) {
		os.Setenv("TEST_INT", "42")
		defer os.Unsetenv("TEST_INT")

		result := GetEnvInt("TEST_INT", 0)
		assert.Equal(t, 42, result)
	})

	t.Run("returns default on invalid int", func(t *testing.T) {
		os.Setenv("TEST_INT_INVALID", "not_a_number")
		defer os.Unsetenv("TEST_INT_INVALID")

		result := GetEnvInt("TEST_INT_INVALID", 100)
		assert.Equal(t, 100, result)
	})

	t.Run("returns default when not set", func(t *testing.T) {
		result := GetEnvInt("TEST_INT_UNSET", 50)
		assert.Equal(t, 50, result)
	})
}

func TestGetEnvInt64(t *testing.T) {
	t.Run("returns int64 value when set", func(t *testing.T) {
		os.Setenv("TEST_INT64", "9223372036854775807")
		defer os.Unsetenv("TEST_INT64")

		result := GetEnvInt64("TEST_INT64", 0)
		assert.Equal(t, int64(9223372036854775807), result)
	})

	t.Run("returns default when not set", func(t *testing.T) {
		result := GetEnvInt64("TEST_INT64_UNSET", 123456789)
		assert.Equal(t, int64(123456789), result)
	})
}

func TestGetEnvFloat64(t *testing.T) {
	t.Run("returns float value when set", func(t *testing.T) {
		os.Setenv("TEST_FLOAT", "3.14159")
		defer os.Unsetenv("TEST_FLOAT")

		result := GetEnvFloat64("TEST_FLOAT", 0)
		assert.InDelta(t, 3.14159, result, 0.00001)
	})

	t.Run("returns default when not set", func(t *testing.T) {
		result := GetEnvFloat64("TEST_FLOAT_UNSET", 2.71828)
		assert.InDelta(t, 2.71828, result, 0.00001)
	})
}

func TestGetEnvBool(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected bool
	}{
		{"true lowercase", "true", true},
		{"True mixed", "True", true},
		{"TRUE uppercase", "TRUE", true},
		{"1", "1", true},
		{"false lowercase", "false", false},
		{"False mixed", "False", false},
		{"0", "0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("TEST_BOOL", tt.envValue)
			defer os.Unsetenv("TEST_BOOL")

			result := GetEnvBool("TEST_BOOL", !tt.expected)
			assert.Equal(t, tt.expected, result)
		})
	}

	t.Run("returns default when not set", func(t *testing.T) {
		result := GetEnvBool("TEST_BOOL_UNSET", true)
		assert.True(t, result)
	})
}

func TestGetEnvDuration(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected time.Duration
	}{
		{"seconds", "30s", 30 * time.Second},
		{"minutes", "5m", 5 * time.Minute},
		{"hours", "2h", 2 * time.Hour},
		{"complex", "1h30m", 90 * time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("TEST_DURATION", tt.envValue)
			defer os.Unsetenv("TEST_DURATION")

			result := GetEnvDuration("TEST_DURATION", 0)
			assert.Equal(t, tt.expected, result)
		})
	}

	t.Run("returns default on invalid duration", func(t *testing.T) {
		os.Setenv("TEST_DURATION_INVALID", "not_a_duration")
		defer os.Unsetenv("TEST_DURATION_INVALID")

		result := GetEnvDuration("TEST_DURATION_INVALID", 10*time.Second)
		assert.Equal(t, 10*time.Second, result)
	})
}

func TestMustGetEnv(t *testing.T) {
	t.Run("returns value when set", func(t *testing.T) {
		os.Setenv("TEST_MUST", "required_value")
		defer os.Unsetenv("TEST_MUST")

		result := MustGetEnv("TEST_MUST")
		assert.Equal(t, "required_value", result)
	})

	t.Run("panics when not set", func(t *testing.T) {
		os.Unsetenv("TEST_MUST_UNSET")

		assert.Panics(t, func() {
			MustGetEnv("TEST_MUST_UNSET")
		})
	})
}

func TestGetEnvSlice(t *testing.T) {
	t.Run("returns slice from comma-separated value", func(t *testing.T) {
		os.Setenv("TEST_SLICE", "a,b,c")
		defer os.Unsetenv("TEST_SLICE")

		result := GetEnvSlice("TEST_SLICE", nil)
		assert.Equal(t, []string{"a", "b", "c"}, result)
	})

	t.Run("trims whitespace", func(t *testing.T) {
		os.Setenv("TEST_SLICE_SPACES", "a , b , c")
		defer os.Unsetenv("TEST_SLICE_SPACES")

		result := GetEnvSlice("TEST_SLICE_SPACES", nil)
		assert.Equal(t, []string{"a", "b", "c"}, result)
	})

	t.Run("returns default when not set", func(t *testing.T) {
		defaultSlice := []string{"default1", "default2"}
		result := GetEnvSlice("TEST_SLICE_UNSET", defaultSlice)
		assert.Equal(t, defaultSlice, result)
	})
}
