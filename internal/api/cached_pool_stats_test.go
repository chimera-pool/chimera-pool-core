package api

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// CACHED POOL STATS READER TESTS (TDD)
// =============================================================================

func TestDefaultCachedPoolStatsConfig(t *testing.T) {
	config := DefaultCachedPoolStatsConfig()

	require.NotNil(t, config)
	assert.Equal(t, 10*time.Second, config.TTL)
	assert.Equal(t, "chimera:", config.KeyPrefix)
}

func TestCachedPoolStatsReader_ImplementsInterface(t *testing.T) {
	// Verify CachedPoolStatsReader implements PoolStatsReader
	var _ PoolStatsReader = (*CachedPoolStatsReader)(nil)
}

func TestCachedPoolStatsReader_GetCacheStats(t *testing.T) {
	// Create a mock reader for testing cache stats
	mockReader := &mockPoolStatsReader{
		stats: &PoolStatsData{
			TotalMiners:   100,
			TotalHashrate: 21.5e12,
			BlocksFound:   50,
		},
	}

	// Create cached reader with nil redis (will fail cache ops but still work)
	cached := &CachedPoolStatsReader{
		delegate: mockReader,
		ttl:      10 * time.Second,
		key:      "test:pool:stats",
	}

	// Initial stats should be zero
	stats := cached.GetCacheStats()
	assert.Equal(t, int64(0), stats["hits"])
	assert.Equal(t, int64(0), stats["misses"])
	assert.Equal(t, float64(0), stats["hit_rate"])
}

func TestCachedHashrateStatsReader_ImplementsInterface(t *testing.T) {
	// Verify CachedHashrateStatsReader implements HashrateStatsReader
	var _ HashrateStatsReader = (*CachedHashrateStatsReader)(nil)
}

func TestHashrateDataPoint_Struct(t *testing.T) {
	now := time.Now()
	point := HashrateDataPoint{
		Timestamp: now,
		Hashrate:  21.5e12,
	}

	assert.Equal(t, now, point.Timestamp)
	assert.Equal(t, 21.5e12, point.Hashrate)
}

// =============================================================================
// MOCK IMPLEMENTATIONS
// =============================================================================

type mockPoolStatsReader struct {
	stats *PoolStatsData
	err   error
}

func (m *mockPoolStatsReader) GetStats() (*PoolStatsData, error) {
	return m.stats, m.err
}

type mockHashrateStatsReader struct {
	data []HashrateDataPoint
	err  error
}

func (m *mockHashrateStatsReader) GetHashrateHistory(rangeStr string) ([]HashrateDataPoint, error) {
	return m.data, m.err
}
