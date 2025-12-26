package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// INTERFACE COMPLIANCE TESTS (TDD)
// =============================================================================

func TestDefaultCacheConfig(t *testing.T) {
	config := DefaultCacheConfig()

	require.NotNil(t, config)
	assert.Equal(t, 10*time.Second, config.PoolStatsTTL)
	assert.Equal(t, 30*time.Second, config.HashrateTTL)
	assert.Equal(t, 15*time.Second, config.UserStatsTTL)
	assert.Equal(t, 60*time.Second, config.MinerLocationsTTL)
	assert.Equal(t, "redis:6379", config.RedisAddr)
	assert.Equal(t, 1, config.RedisDB)
	assert.Equal(t, "chimera:", config.KeyPrefix)
}

func TestCacheKeys_PoolStats(t *testing.T) {
	keys := NewCacheKeys("chimera:")

	assert.Equal(t, "chimera:pool:stats", keys.PoolStats())
}

func TestCacheKeys_HashrateHistory(t *testing.T) {
	keys := NewCacheKeys("chimera:")

	assert.Equal(t, "chimera:pool:hashrate:24h", keys.HashrateHistory("24h"))
	assert.Equal(t, "chimera:pool:hashrate:7d", keys.HashrateHistory("7d"))
}

func TestCacheKeys_MinerLocations(t *testing.T) {
	keys := NewCacheKeys("chimera:")

	assert.Equal(t, "chimera:miners:locations", keys.MinerLocations())
}

func TestCacheKeys_ShareStats(t *testing.T) {
	keys := NewCacheKeys("chimera:")

	assert.Equal(t, "chimera:pool:shares:24h", keys.ShareStats("24h"))
}

func TestPoolStats_Struct(t *testing.T) {
	now := time.Now()
	stats := &PoolStats{
		TotalHashrate:    21.5e12, // 21.5 TH/s
		ActiveMiners:     150,
		ActiveWorkers:    320,
		BlocksFound24h:   5,
		TotalBlocksFound: 1250,
		PoolFee:          0.01,
		MinPayout:        0.001,
		LastBlockTime:    now.Add(-2 * time.Hour),
		NetworkDiff:      1.5e10,
		CachedAt:         now,
	}

	assert.Equal(t, 21.5e12, stats.TotalHashrate)
	assert.Equal(t, 150, stats.ActiveMiners)
	assert.Equal(t, 320, stats.ActiveWorkers)
	assert.Equal(t, 5, stats.BlocksFound24h)
	assert.Equal(t, 1250, stats.TotalBlocksFound)
	assert.Equal(t, 0.01, stats.PoolFee)
}

func TestHashratePoint_Struct(t *testing.T) {
	now := time.Now()
	point := HashratePoint{
		Timestamp: now,
		Hashrate:  21.5e12,
	}

	assert.Equal(t, now, point.Timestamp)
	assert.Equal(t, 21.5e12, point.Hashrate)
}

func TestUserStats_Struct(t *testing.T) {
	now := time.Now()
	stats := &UserStats{
		UserID:        123,
		Hashrate:      1.5e12,
		SharesValid:   50000,
		SharesInvalid: 50,
		PendingPayout: 0.0025,
		TotalPaid:     1.25,
		WorkerCount:   3,
		CachedAt:      now,
	}

	assert.Equal(t, int64(123), stats.UserID)
	assert.Equal(t, 1.5e12, stats.Hashrate)
	assert.Equal(t, int64(50000), stats.SharesValid)
	assert.Equal(t, 3, stats.WorkerCount)
}

// =============================================================================
// INTERFACE IMPLEMENTATION VERIFICATION
// =============================================================================

// Verify interface segregation - each interface is independently usable
func TestInterfaceSegregation(t *testing.T) {
	t.Run("CacheReader is independent", func(t *testing.T) {
		var _ CacheReader = (*mockCacheReader)(nil)
	})

	t.Run("CacheWriter is independent", func(t *testing.T) {
		var _ CacheWriter = (*mockCacheWriter)(nil)
	})

	t.Run("CacheInvalidator is independent", func(t *testing.T) {
		var _ CacheInvalidator = (*mockCacheInvalidator)(nil)
	})

	t.Run("Cache combines all interfaces", func(t *testing.T) {
		var _ Cache = (*mockCache)(nil)
	})

	t.Run("PoolStatsCache is specialized", func(t *testing.T) {
		var _ PoolStatsCache = (*mockPoolStatsCache)(nil)
	})
}

// =============================================================================
// MOCK IMPLEMENTATIONS FOR INTERFACE VERIFICATION
// =============================================================================

type mockCacheReader struct{}

func (m *mockCacheReader) Get(_ context.Context, _ string) ([]byte, error)  { return nil, nil }
func (m *mockCacheReader) Exists(_ context.Context, _ string) (bool, error) { return false, nil }

type mockCacheWriter struct{}

func (m *mockCacheWriter) Set(_ context.Context, _ string, _ []byte, _ time.Duration) error {
	return nil
}
func (m *mockCacheWriter) Delete(_ context.Context, _ string) error { return nil }

type mockCacheInvalidator struct{}

func (m *mockCacheInvalidator) DeletePattern(_ context.Context, _ string) error { return nil }
func (m *mockCacheInvalidator) Flush(_ context.Context) error                   { return nil }

type mockCache struct {
	mockCacheReader
	mockCacheWriter
	mockCacheInvalidator
}

type mockPoolStatsCache struct{}

func (m *mockPoolStatsCache) GetPoolStats(_ context.Context) (*PoolStats, error) { return nil, nil }
func (m *mockPoolStatsCache) SetPoolStats(_ context.Context, _ *PoolStats) error { return nil }
func (m *mockPoolStatsCache) GetHashrateHistory(_ context.Context, _ string) ([]HashratePoint, error) {
	return nil, nil
}
func (m *mockPoolStatsCache) SetHashrateHistory(_ context.Context, _ string, _ []HashratePoint) error {
	return nil
}
func (m *mockPoolStatsCache) InvalidatePoolStats(_ context.Context) error { return nil }
