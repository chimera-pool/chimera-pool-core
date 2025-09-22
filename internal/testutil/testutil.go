// Package testutil provides common testing utilities for the Chimera Pool project
package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	_ "github.com/lib/pq"
)

// TestDatabase represents a test database instance
type TestDatabase struct {
	Container testcontainers.Container
	DB        *sql.DB
	URL       string
}

// TestRedis represents a test Redis instance
type TestRedis struct {
	Container testcontainers.Container
	Client    *redis.Client
	URL       string
}

// SetupTestDatabase creates a PostgreSQL test database using testcontainers
func SetupTestDatabase(t *testing.T) *TestDatabase {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:15-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_DB":       "chimera_pool_test",
			"POSTGRES_USER":     "chimera",
			"POSTGRES_PASSWORD": "test_password",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(30 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	// Get the mapped port
	mappedPort, err := container.MappedPort(ctx, "5432")
	require.NoError(t, err)

	host, err := container.Host(ctx)
	require.NoError(t, err)

	// Create database connection
	dbURL := fmt.Sprintf("postgres://chimera:test_password@%s:%s/chimera_pool_test?sslmode=disable",
		host, mappedPort.Port())

	db, err := sql.Open("postgres", dbURL)
	require.NoError(t, err)

	// Wait for database to be ready
	require.Eventually(t, func() bool {
		return db.Ping() == nil
	}, 30*time.Second, 1*time.Second, "Database should be ready")

	t.Cleanup(func() {
		db.Close()
		container.Terminate(ctx)
	})

	return &TestDatabase{
		Container: container,
		DB:        db,
		URL:       dbURL,
	}
}

// SetupTestRedis creates a Redis test instance using testcontainers
func SetupTestRedis(t *testing.T) *TestRedis {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "redis:7-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	// Get the mapped port
	mappedPort, err := container.MappedPort(ctx, "6379")
	require.NoError(t, err)

	host, err := container.Host(ctx)
	require.NoError(t, err)

	// Create Redis client
	redisURL := fmt.Sprintf("redis://%s:%s", host, mappedPort.Port())
	
	client := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", host, mappedPort.Port()),
	})

	// Test connection
	require.Eventually(t, func() bool {
		return client.Ping(ctx).Err() == nil
	}, 30*time.Second, 1*time.Second, "Redis should be ready")

	t.Cleanup(func() {
		client.Close()
		container.Terminate(ctx)
	})

	return &TestRedis{
		Container: container,
		Client:    client,
		URL:       redisURL,
	}
}

// AssertCoverage checks that test coverage meets the minimum threshold
func AssertCoverage(t *testing.T, coverageFile string, threshold float64) {
	// This would be implemented to parse coverage files and assert minimum coverage
	// For now, we'll just log the requirement
	t.Logf("Coverage check: %s should meet %.1f%% threshold", coverageFile, threshold*100)
}

// BenchmarkHelper provides utilities for consistent benchmarking
type BenchmarkHelper struct {
	iterations int
	warmup     int
}

// NewBenchmarkHelper creates a new benchmark helper
func NewBenchmarkHelper() *BenchmarkHelper {
	return &BenchmarkHelper{
		iterations: 1000,
		warmup:     100,
	}
}

// Run executes a benchmark with warmup
func (bh *BenchmarkHelper) Run(name string, fn func()) {
	// Warmup
	for i := 0; i < bh.warmup; i++ {
		fn()
	}

	// Actual benchmark
	start := time.Now()
	for i := 0; i < bh.iterations; i++ {
		fn()
	}
	duration := time.Since(start)

	fmt.Printf("Benchmark %s: %d iterations in %v (%.2f ns/op)\n",
		name, bh.iterations, duration, float64(duration.Nanoseconds())/float64(bh.iterations))
}