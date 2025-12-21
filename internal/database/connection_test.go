package database

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// skipIfNoDatabase skips integration tests when no database is available
func skipIfNoDatabase(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test - set INTEGRATION_TEST=true to run")
	}
}

// getTestConfig returns a database config for testing
func getTestConfig() *Config {
	return &Config{
		Host:     getEnvOrDefault("TEST_DB_HOST", "localhost"),
		Port:     5432,
		Database: getEnvOrDefault("TEST_DB_NAME", "chimera_pool_test"),
		Username: getEnvOrDefault("TEST_DB_USER", "chimera"),
		Password: getEnvOrDefault("TEST_DB_PASSWORD", "test_password"),
		SSLMode:  "disable",
		MaxConns: 10,
		MinConns: 2,
	}
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func TestDatabaseConnection(t *testing.T) {
	skipIfNoDatabase(t)

	t.Run("CreateConnectionPool", func(t *testing.T) {
		config := getTestConfig()

		pool, err := NewConnectionPool(config)
		require.NoError(t, err, "should create connection pool without error")
		require.NotNil(t, pool, "connection pool should not be nil")

		defer pool.Close()
	})

	t.Run("ConnectionPoolHealthCheck", func(t *testing.T) {
		config := getTestConfig()

		pool, err := NewConnectionPool(config)
		require.NoError(t, err)
		defer pool.Close()

		// Test health check
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		healthy := pool.HealthCheck(ctx)
		assert.True(t, healthy, "connection pool should be healthy")
	})

	t.Run("ConnectionPoolStats", func(t *testing.T) {
		config := getTestConfig()

		pool, err := NewConnectionPool(config)
		require.NoError(t, err)
		defer pool.Close()

		stats := pool.Stats()
		assert.GreaterOrEqual(t, stats.MaxConns, int32(2), "should have at least min connections")
		assert.LessOrEqual(t, stats.OpenConns, stats.MaxConns, "open connections should not exceed max")
	})

	t.Run("BasicQuery", func(t *testing.T) {
		config := getTestConfig()

		pool, err := NewConnectionPool(config)
		require.NoError(t, err)
		defer pool.Close()

		// Test basic query
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var result int
		err = pool.QueryRow(ctx, "SELECT 1").Scan(&result)
		require.NoError(t, err, "should execute basic query")
		assert.Equal(t, 1, result, "query should return expected result")
	})

	t.Run("TransactionSupport", func(t *testing.T) {
		config := getTestConfig()

		pool, err := NewConnectionPool(config)
		require.NoError(t, err)
		defer pool.Close()

		// Test transaction
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		tx, err := pool.Begin(ctx)
		require.NoError(t, err, "should begin transaction")

		// Test rollback
		err = tx.Rollback(ctx)
		require.NoError(t, err, "should rollback transaction")
	})
}

func TestMigrations(t *testing.T) {
	skipIfNoDatabase(t)

	t.Run("RunMigrations", func(t *testing.T) {
		config := getTestConfig()

		err := RunMigrations(config, "../../migrations")
		require.NoError(t, err, "should run migrations without error")
	})

	t.Run("MigrationStatus", func(t *testing.T) {
		config := getTestConfig()

		status, err := GetMigrationStatus(config)
		require.NoError(t, err, "should get migration status")
		assert.NotNil(t, status, "migration status should not be nil")
	})
}

// Note: All types and functions are implemented in connection.go and database.go
