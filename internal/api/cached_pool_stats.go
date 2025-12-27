package api

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// =============================================================================
// CACHED POOL STATS READER - ISP-COMPLIANT DECORATOR
// Implements PoolStatsReader interface with Redis caching
// =============================================================================

// CachedPoolStatsReader wraps a PoolStatsReader with Redis caching
type CachedPoolStatsReader struct {
	delegate PoolStatsReader
	redis    *redis.Client
	ttl      time.Duration
	key      string
	mu       sync.RWMutex
	stats    struct {
		hits   int64
		misses int64
	}
}

// CachedPoolStatsConfig configures the cached reader
type CachedPoolStatsConfig struct {
	TTL       time.Duration
	KeyPrefix string
}

// DefaultCachedPoolStatsConfig returns sensible defaults
func DefaultCachedPoolStatsConfig() *CachedPoolStatsConfig {
	return &CachedPoolStatsConfig{
		TTL:       10 * time.Second, // Cache for 10 seconds
		KeyPrefix: "chimera:",
	}
}

// NewCachedPoolStatsReader creates a new cached pool stats reader
func NewCachedPoolStatsReader(
	delegate PoolStatsReader,
	redisClient *redis.Client,
	config *CachedPoolStatsConfig,
) *CachedPoolStatsReader {
	if config == nil {
		config = DefaultCachedPoolStatsConfig()
	}

	return &CachedPoolStatsReader{
		delegate: delegate,
		redis:    redisClient,
		ttl:      config.TTL,
		key:      config.KeyPrefix + "pool:stats",
	}
}

// GetStats returns pool statistics with caching
func (c *CachedPoolStatsReader) GetStats() (*PoolStatsData, error) {
	ctx := context.Background()

	// Try cache first
	cached, err := c.redis.Get(ctx, c.key).Bytes()
	if err == nil && len(cached) > 0 {
		var stats PoolStatsData
		if err := json.Unmarshal(cached, &stats); err == nil {
			c.mu.Lock()
			c.stats.hits++
			c.mu.Unlock()
			return &stats, nil
		}
	}

	// Cache miss - fetch from delegate
	c.mu.Lock()
	c.stats.misses++
	c.mu.Unlock()

	stats, err := c.delegate.GetStats()
	if err != nil {
		return nil, err
	}

	// Cache the result
	data, err := json.Marshal(stats)
	if err == nil {
		if cacheErr := c.redis.Set(ctx, c.key, data, c.ttl).Err(); cacheErr != nil {
			log.Printf("[CachedPoolStats] Failed to cache: %v", cacheErr)
		}
	}

	return stats, nil
}

// Invalidate removes the cached stats
func (c *CachedPoolStatsReader) Invalidate() error {
	return c.redis.Del(context.Background(), c.key).Err()
}

// GetCacheStats returns cache hit/miss statistics
func (c *CachedPoolStatsReader) GetCacheStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	total := c.stats.hits + c.stats.misses
	hitRate := float64(0)
	if total > 0 {
		hitRate = float64(c.stats.hits) / float64(total) * 100
	}

	return map[string]interface{}{
		"hits":     c.stats.hits,
		"misses":   c.stats.misses,
		"hit_rate": hitRate,
		"ttl":      c.ttl.String(),
	}
}

// =============================================================================
// CACHED HASHRATE STATS READER
// =============================================================================

// CachedHashrateStatsReader caches hashrate history data
type CachedHashrateStatsReader struct {
	delegate HashrateStatsReader
	redis    *redis.Client
	ttl      time.Duration
	prefix   string
	mu       sync.RWMutex
	stats    struct {
		hits   int64
		misses int64
	}
}

// HashrateStatsReader interface for hashrate data
type HashrateStatsReader interface {
	GetHashrateHistory(rangeStr string) ([]HashrateDataPoint, error)
}

// HashrateDataPoint represents a hashrate data point
type HashrateDataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Hashrate  float64   `json:"hashrate"`
}

// NewCachedHashrateStatsReader creates a cached hashrate reader
func NewCachedHashrateStatsReader(
	delegate HashrateStatsReader,
	redisClient *redis.Client,
	ttl time.Duration,
) *CachedHashrateStatsReader {
	return &CachedHashrateStatsReader{
		delegate: delegate,
		redis:    redisClient,
		ttl:      ttl,
		prefix:   "chimera:pool:hashrate:",
	}
}

// GetHashrateHistory returns hashrate history with caching
func (c *CachedHashrateStatsReader) GetHashrateHistory(rangeStr string) ([]HashrateDataPoint, error) {
	ctx := context.Background()
	key := c.prefix + rangeStr

	// Try cache first
	cached, err := c.redis.Get(ctx, key).Bytes()
	if err == nil && len(cached) > 0 {
		var data []HashrateDataPoint
		if err := json.Unmarshal(cached, &data); err == nil {
			c.mu.Lock()
			c.stats.hits++
			c.mu.Unlock()
			return data, nil
		}
	}

	// Cache miss
	c.mu.Lock()
	c.stats.misses++
	c.mu.Unlock()

	data, err := c.delegate.GetHashrateHistory(rangeStr)
	if err != nil {
		return nil, err
	}

	// Cache the result
	jsonData, err := json.Marshal(data)
	if err == nil {
		if cacheErr := c.redis.Set(ctx, key, jsonData, c.ttl).Err(); cacheErr != nil {
			log.Printf("[CachedHashrateStats] Failed to cache %s: %v", key, cacheErr)
		}
	}

	return data, nil
}

// GetCacheStats returns cache statistics
func (c *CachedHashrateStatsReader) GetCacheStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	total := c.stats.hits + c.stats.misses
	hitRate := float64(0)
	if total > 0 {
		hitRate = float64(c.stats.hits) / float64(total) * 100
	}

	return map[string]interface{}{
		"hits":     c.stats.hits,
		"misses":   c.stats.misses,
		"hit_rate": hitRate,
	}
}
