package cache

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"
)

// =============================================================================
// CACHED DATA PROVIDER
// Wraps data fetching with cache-aside pattern
// =============================================================================

// DataFetcher is a function that fetches fresh data
type DataFetcher[T any] func(ctx context.Context) (T, error)

// CachedProvider provides cached data with automatic refresh
type CachedProvider[T any] struct {
	cache      Cache
	key        string
	ttl        time.Duration
	fetcher    DataFetcher[T]
	mu         sync.RWMutex
	lastFetch  time.Time
	fetchCount int64
	hitCount   int64
	missCount  int64
}

// NewCachedProvider creates a new cached data provider
func NewCachedProvider[T any](cache Cache, key string, ttl time.Duration, fetcher DataFetcher[T]) *CachedProvider[T] {
	return &CachedProvider[T]{
		cache:   cache,
		key:     key,
		ttl:     ttl,
		fetcher: fetcher,
	}
}

// Get retrieves data from cache or fetches fresh data
func (p *CachedProvider[T]) Get(ctx context.Context) (T, error) {
	var zero T

	// Try cache first
	data, err := p.cache.Get(ctx, p.key)
	if err == nil && data != nil {
		p.mu.Lock()
		p.hitCount++
		p.mu.Unlock()

		var result T
		if err := unmarshalJSON(data, &result); err == nil {
			return result, nil
		}
	}

	// Cache miss - fetch fresh data
	p.mu.Lock()
	p.missCount++
	p.mu.Unlock()

	result, err := p.fetcher(ctx)
	if err != nil {
		return zero, err
	}

	// Store in cache
	jsonData, err := marshalJSON(result)
	if err == nil {
		if cacheErr := p.cache.Set(ctx, p.key, jsonData, p.ttl); cacheErr != nil {
			log.Printf("[Cache] Failed to cache %s: %v", p.key, cacheErr)
		}
	}

	p.mu.Lock()
	p.fetchCount++
	p.lastFetch = time.Now()
	p.mu.Unlock()

	return result, nil
}

// Invalidate removes the cached data
func (p *CachedProvider[T]) Invalidate(ctx context.Context) error {
	return p.cache.Delete(ctx, p.key)
}

// Stats returns cache statistics
func (p *CachedProvider[T]) Stats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	total := p.hitCount + p.missCount
	hitRate := float64(0)
	if total > 0 {
		hitRate = float64(p.hitCount) / float64(total) * 100
	}

	return map[string]interface{}{
		"key":        p.key,
		"ttl":        p.ttl.String(),
		"hits":       p.hitCount,
		"misses":     p.missCount,
		"fetches":    p.fetchCount,
		"hit_rate":   hitRate,
		"last_fetch": p.lastFetch,
	}
}

// =============================================================================
// POOL STATS CACHED SERVICE
// =============================================================================

// PoolStatsFetcher defines how to fetch pool stats from the database
type PoolStatsFetcher func(ctx context.Context) (*PoolStats, error)

// CachedPoolStatsService provides cached pool statistics
type CachedPoolStatsService struct {
	cache   PoolStatsCache
	fetcher PoolStatsFetcher
	mu      sync.RWMutex
	stats   struct {
		hits   int64
		misses int64
	}
}

// NewCachedPoolStatsService creates a new cached pool stats service
func NewCachedPoolStatsService(cache PoolStatsCache, fetcher PoolStatsFetcher) *CachedPoolStatsService {
	return &CachedPoolStatsService{
		cache:   cache,
		fetcher: fetcher,
	}
}

// GetPoolStats retrieves pool stats with caching
func (s *CachedPoolStatsService) GetPoolStats(ctx context.Context) (*PoolStats, error) {
	// Try cache first
	cached, err := s.cache.GetPoolStats(ctx)
	if err == nil && cached != nil {
		s.mu.Lock()
		s.stats.hits++
		s.mu.Unlock()
		return cached, nil
	}

	// Cache miss
	s.mu.Lock()
	s.stats.misses++
	s.mu.Unlock()

	// Fetch fresh data
	stats, err := s.fetcher(ctx)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if cacheErr := s.cache.SetPoolStats(ctx, stats); cacheErr != nil {
		log.Printf("[CachedPoolStats] Failed to cache: %v", cacheErr)
	}

	return stats, nil
}

// InvalidateCache invalidates the pool stats cache
func (s *CachedPoolStatsService) InvalidateCache(ctx context.Context) error {
	return s.cache.InvalidatePoolStats(ctx)
}

// GetCacheStats returns cache statistics
func (s *CachedPoolStatsService) GetCacheStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := s.stats.hits + s.stats.misses
	hitRate := float64(0)
	if total > 0 {
		hitRate = float64(s.stats.hits) / float64(total) * 100
	}

	return map[string]interface{}{
		"hits":     s.stats.hits,
		"misses":   s.stats.misses,
		"hit_rate": hitRate,
	}
}

// =============================================================================
// JSON HELPERS
// =============================================================================

func marshalJSON(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func unmarshalJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
