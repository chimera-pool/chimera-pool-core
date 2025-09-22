//go:build integration
// +build integration

package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisIntegration(t *testing.T) {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		t.Skip("REDIS_URL not set, skipping Redis integration test")
	}

	// Parse Redis URL and create client
	opt, err := redis.ParseURL(redisURL)
	require.NoError(t, err)

	client := redis.NewClient(opt)
	defer client.Close()

	ctx := context.Background()

	// Test connection
	pong, err := client.Ping(ctx).Result()
	require.NoError(t, err)
	assert.Equal(t, "PONG", pong)

	t.Run("BasicOperations", func(t *testing.T) {
		// Set a key
		err := client.Set(ctx, "test_key", "test_value", time.Minute).Err()
		assert.NoError(t, err)

		// Get the key
		val, err := client.Get(ctx, "test_key").Result()
		assert.NoError(t, err)
		assert.Equal(t, "test_value", val)

		// Delete the key
		err = client.Del(ctx, "test_key").Err()
		assert.NoError(t, err)

		// Verify key is deleted
		_, err = client.Get(ctx, "test_key").Result()
		assert.Equal(t, redis.Nil, err)
	})

	t.Run("HashOperations", func(t *testing.T) {
		hashKey := "test_hash"

		// Set hash fields
		err := client.HSet(ctx, hashKey, map[string]interface{}{
			"field1": "value1",
			"field2": "value2",
		}).Err()
		assert.NoError(t, err)

		// Get hash field
		val, err := client.HGet(ctx, hashKey, "field1").Result()
		assert.NoError(t, err)
		assert.Equal(t, "value1", val)

		// Get all hash fields
		fields, err := client.HGetAll(ctx, hashKey).Result()
		assert.NoError(t, err)
		assert.Equal(t, map[string]string{
			"field1": "value1",
			"field2": "value2",
		}, fields)

		// Cleanup
		err = client.Del(ctx, hashKey).Err()
		assert.NoError(t, err)
	})

	t.Run("ListOperations", func(t *testing.T) {
		listKey := "test_list"

		// Push items to list
		err := client.LPush(ctx, listKey, "item1", "item2", "item3").Err()
		assert.NoError(t, err)

		// Get list length
		length, err := client.LLen(ctx, listKey).Result()
		assert.NoError(t, err)
		assert.Equal(t, int64(3), length)

		// Pop item from list
		item, err := client.LPop(ctx, listKey).Result()
		assert.NoError(t, err)
		assert.Equal(t, "item3", item) // LIFO order

		// Cleanup
		err = client.Del(ctx, listKey).Err()
		assert.NoError(t, err)
	})
}

func main() {
	// This allows the test to be run as a standalone program
	// for integration testing in Docker
	testing.Main(func(pat, str string) (bool, error) { return true, nil },
		[]testing.InternalTest{
			{
				Name: "TestRedisIntegration",
				F:    TestRedisIntegration,
			},
		},
		[]testing.InternalBenchmark{},
		[]testing.InternalExample{},
	)
}