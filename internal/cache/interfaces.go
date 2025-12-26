package cache

import (
	"context"
	"time"
)

// =============================================================================
// ISP-COMPLIANT CACHE INTERFACES
// Each interface is small and focused on a single responsibility
// =============================================================================

// CacheReader handles cache read operations
type CacheReader interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Exists(ctx context.Context, key string) (bool, error)
}

// CacheWriter handles cache write operations
type CacheWriter interface {
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

// CacheInvalidator handles cache invalidation
type CacheInvalidator interface {
	DeletePattern(ctx context.Context, pattern string) error
	Flush(ctx context.Context) error
}

// Cache combines read and write operations (full cache interface)
type Cache interface {
	CacheReader
	CacheWriter
	CacheInvalidator
}

// PoolStatsCache is a specialized cache for pool statistics
// ISP: Only exposes methods needed for pool stats caching
type PoolStatsCache interface {
	GetPoolStats(ctx context.Context) (*PoolStats, error)
	SetPoolStats(ctx context.Context, stats *PoolStats) error
	GetHashrateHistory(ctx context.Context, range_ string) ([]HashratePoint, error)
	SetHashrateHistory(ctx context.Context, range_ string, data []HashratePoint) error
	InvalidatePoolStats(ctx context.Context) error
}

// UserStatsCache is a specialized cache for user statistics
type UserStatsCache interface {
	GetUserStats(ctx context.Context, userID int64) (*UserStats, error)
	SetUserStats(ctx context.Context, userID int64, stats *UserStats) error
	InvalidateUserStats(ctx context.Context, userID int64) error
}

// =============================================================================
// CACHE DATA TYPES
// =============================================================================

// PoolStats represents cached pool statistics
type PoolStats struct {
	TotalHashrate    float64   `json:"total_hashrate"`
	ActiveMiners     int       `json:"active_miners"`
	ActiveWorkers    int       `json:"active_workers"`
	BlocksFound24h   int       `json:"blocks_found_24h"`
	TotalBlocksFound int       `json:"total_blocks_found"`
	PoolFee          float64   `json:"pool_fee"`
	MinPayout        float64   `json:"min_payout"`
	LastBlockTime    time.Time `json:"last_block_time"`
	NetworkDiff      float64   `json:"network_diff"`
	CachedAt         time.Time `json:"cached_at"`
}

// HashratePoint represents a single hashrate data point
type HashratePoint struct {
	Timestamp time.Time `json:"timestamp"`
	Hashrate  float64   `json:"hashrate"`
}

// UserStats represents cached user statistics
type UserStats struct {
	UserID        int64     `json:"user_id"`
	Hashrate      float64   `json:"hashrate"`
	SharesValid   int64     `json:"shares_valid"`
	SharesInvalid int64     `json:"shares_invalid"`
	PendingPayout float64   `json:"pending_payout"`
	TotalPaid     float64   `json:"total_paid"`
	WorkerCount   int       `json:"worker_count"`
	CachedAt      time.Time `json:"cached_at"`
}

// =============================================================================
// CACHE CONFIGURATION
// =============================================================================

// CacheConfig holds cache configuration
type CacheConfig struct {
	// TTL settings
	PoolStatsTTL      time.Duration `json:"pool_stats_ttl"`      // Default: 10s
	HashrateTTL       time.Duration `json:"hashrate_ttl"`        // Default: 30s
	UserStatsTTL      time.Duration `json:"user_stats_ttl"`      // Default: 15s
	MinerLocationsTTL time.Duration `json:"miner_locations_ttl"` // Default: 60s

	// Redis settings
	RedisAddr     string `json:"redis_addr"`
	RedisPassword string `json:"redis_password"`
	RedisDB       int    `json:"redis_db"`

	// Key prefixes
	KeyPrefix string `json:"key_prefix"` // Default: "chimera:"
}

// DefaultCacheConfig returns sensible defaults
func DefaultCacheConfig() *CacheConfig {
	return &CacheConfig{
		PoolStatsTTL:      10 * time.Second, // Refresh pool stats every 10s
		HashrateTTL:       30 * time.Second, // Hashrate history cached 30s
		UserStatsTTL:      15 * time.Second, // User stats every 15s
		MinerLocationsTTL: 60 * time.Second, // Miner locations every 60s
		RedisAddr:         "redis:6379",
		RedisDB:           1, // Use DB 1 for cache (DB 0 for sessions)
		KeyPrefix:         "chimera:",
	}
}

// =============================================================================
// CACHE KEY HELPERS
// =============================================================================

// CacheKeys provides standardized cache key generation
type CacheKeys struct {
	prefix string
}

// NewCacheKeys creates a new CacheKeys helper
func NewCacheKeys(prefix string) *CacheKeys {
	return &CacheKeys{prefix: prefix}
}

// PoolStats returns the key for pool stats
func (k *CacheKeys) PoolStats() string {
	return k.prefix + "pool:stats"
}

// HashrateHistory returns the key for hashrate history
func (k *CacheKeys) HashrateHistory(range_ string) string {
	return k.prefix + "pool:hashrate:" + range_
}

// UserStats returns the key for user stats
func (k *CacheKeys) UserStats(userID int64) string {
	return k.prefix + "user:" + string(rune(userID)) + ":stats"
}

// MinerLocations returns the key for miner locations
func (k *CacheKeys) MinerLocations() string {
	return k.prefix + "miners:locations"
}

// ShareStats returns the key for share stats
func (k *CacheKeys) ShareStats(range_ string) string {
	return k.prefix + "pool:shares:" + range_
}
