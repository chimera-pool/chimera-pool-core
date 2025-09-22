package testutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupTestDatabase(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDatabase(t)
	require.NotNil(t, testDB)
	require.NotNil(t, testDB.DB)
	require.NotEmpty(t, testDB.URL)

	// Test database connection
	err := testDB.DB.Ping()
	assert.NoError(t, err)

	// Test basic query
	var version string
	err = testDB.DB.QueryRow("SELECT version()").Scan(&version)
	assert.NoError(t, err)
	assert.Contains(t, version, "PostgreSQL")
}

func TestSetupTestRedis(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testRedis := SetupTestRedis(t)
	require.NotNil(t, testRedis)
	require.NotNil(t, testRedis.Client)
	require.NotEmpty(t, testRedis.URL)

	// Test Redis connection
	pong, err := testRedis.Client.Ping(testRedis.Client.Context()).Result()
	assert.NoError(t, err)
	assert.Equal(t, "PONG", pong)

	// Test basic operations
	err = testRedis.Client.Set(testRedis.Client.Context(), "test_key", "test_value", 0).Err()
	assert.NoError(t, err)

	val, err := testRedis.Client.Get(testRedis.Client.Context(), "test_key").Result()
	assert.NoError(t, err)
	assert.Equal(t, "test_value", val)
}

func TestBenchmarkHelper(t *testing.T) {
	helper := NewBenchmarkHelper()
	require.NotNil(t, helper)

	// Test that benchmark runs without error
	counter := 0
	helper.Run("test_benchmark", func() {
		counter++
	})

	// Should have run warmup + iterations
	expectedRuns := helper.warmup + helper.iterations
	assert.Equal(t, expectedRuns, counter)
}