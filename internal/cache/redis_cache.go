package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// =============================================================================
// REDIS CACHE IMPLEMENTATION
// Implements all cache interfaces with Redis backend
// =============================================================================

// RedisCache implements the Cache interface using Redis
type RedisCache struct {
	client *redis.Client
	config *CacheConfig
	keys   *CacheKeys
}

// NewRedisCache creates a new Redis cache instance
func NewRedisCache(config *CacheConfig) (*RedisCache, error) {
	if config == nil {
		config = DefaultCacheConfig()
	}

	client := redis.NewClient(&redis.Options{
		Addr:         config.RedisAddr,
		Password:     config.RedisPassword,
		DB:           config.RedisDB,
		PoolSize:     50, // Connection pool size
		MinIdleConns: 10, // Minimum idle connections
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolTimeout:  4 * time.Second,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisCache{
		client: client,
		config: config,
		keys:   NewCacheKeys(config.KeyPrefix),
	}, nil
}

// Close closes the Redis connection
func (c *RedisCache) Close() error {
	return c.client.Close()
}

// =============================================================================
// CacheReader Implementation
// =============================================================================

// Get retrieves a value from cache
func (c *RedisCache) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil // Cache miss - not an error
	}
	return val, err
}

// Exists checks if a key exists in cache
func (c *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	result, err := c.client.Exists(ctx, key).Result()
	return result > 0, err
}

// =============================================================================
// CacheWriter Implementation
// =============================================================================

// Set stores a value in cache with TTL
func (c *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}

// Delete removes a key from cache
func (c *RedisCache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

// =============================================================================
// CacheInvalidator Implementation
// =============================================================================

// DeletePattern removes all keys matching a pattern
func (c *RedisCache) DeletePattern(ctx context.Context, pattern string) error {
	iter := c.client.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		if err := c.client.Del(ctx, iter.Val()).Err(); err != nil {
			return err
		}
	}
	return iter.Err()
}

// Flush clears all cache entries with the configured prefix
func (c *RedisCache) Flush(ctx context.Context) error {
	return c.DeletePattern(ctx, c.config.KeyPrefix+"*")
}

// =============================================================================
// PoolStatsCache Implementation
// =============================================================================

// GetPoolStats retrieves cached pool statistics
func (c *RedisCache) GetPoolStats(ctx context.Context) (*PoolStats, error) {
	data, err := c.Get(ctx, c.keys.PoolStats())
	if err != nil || data == nil {
		return nil, err
	}

	var stats PoolStats
	if err := json.Unmarshal(data, &stats); err != nil {
		return nil, err
	}
	return &stats, nil
}

// SetPoolStats caches pool statistics
func (c *RedisCache) SetPoolStats(ctx context.Context, stats *PoolStats) error {
	stats.CachedAt = time.Now()
	data, err := json.Marshal(stats)
	if err != nil {
		return err
	}
	return c.Set(ctx, c.keys.PoolStats(), data, c.config.PoolStatsTTL)
}

// GetHashrateHistory retrieves cached hashrate history
func (c *RedisCache) GetHashrateHistory(ctx context.Context, range_ string) ([]HashratePoint, error) {
	data, err := c.Get(ctx, c.keys.HashrateHistory(range_))
	if err != nil || data == nil {
		return nil, err
	}

	var points []HashratePoint
	if err := json.Unmarshal(data, &points); err != nil {
		return nil, err
	}
	return points, nil
}

// SetHashrateHistory caches hashrate history
func (c *RedisCache) SetHashrateHistory(ctx context.Context, range_ string, data []HashratePoint) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return c.Set(ctx, c.keys.HashrateHistory(range_), jsonData, c.config.HashrateTTL)
}

// InvalidatePoolStats removes all pool stats from cache
func (c *RedisCache) InvalidatePoolStats(ctx context.Context) error {
	return c.DeletePattern(ctx, c.config.KeyPrefix+"pool:*")
}

// =============================================================================
// UserStatsCache Implementation
// =============================================================================

// GetUserStats retrieves cached user statistics
func (c *RedisCache) GetUserStats(ctx context.Context, userID int64) (*UserStats, error) {
	key := fmt.Sprintf("%suser:%d:stats", c.config.KeyPrefix, userID)
	data, err := c.Get(ctx, key)
	if err != nil || data == nil {
		return nil, err
	}

	var stats UserStats
	if err := json.Unmarshal(data, &stats); err != nil {
		return nil, err
	}
	return &stats, nil
}

// SetUserStats caches user statistics
func (c *RedisCache) SetUserStats(ctx context.Context, userID int64, stats *UserStats) error {
	stats.CachedAt = time.Now()
	data, err := json.Marshal(stats)
	if err != nil {
		return err
	}
	key := fmt.Sprintf("%suser:%d:stats", c.config.KeyPrefix, userID)
	return c.Set(ctx, key, data, c.config.UserStatsTTL)
}

// InvalidateUserStats removes user stats from cache
func (c *RedisCache) InvalidateUserStats(ctx context.Context, userID int64) error {
	key := fmt.Sprintf("%suser:%d:*", c.config.KeyPrefix, userID)
	return c.DeletePattern(ctx, key)
}

// =============================================================================
// HELPER METHODS
// =============================================================================

// GetClient returns the underlying Redis client for advanced operations
func (c *RedisCache) GetClient() *redis.Client {
	return c.client
}

// Stats returns cache statistics
func (c *RedisCache) Stats(ctx context.Context) (map[string]interface{}, error) {
	info, err := c.client.Info(ctx, "stats", "memory").Result()
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"info": info,
	}, nil
}

// HealthCheck checks if Redis is healthy
func (c *RedisCache) HealthCheck(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}
