package stats

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRealtimeStatsService_GetCurrentPoolSnapshot(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewRealtimeStatsService(db)
	ctx := context.Background()

	t.Run("returns pool snapshot with real data", func(t *testing.T) {
		// Mock hashrate query
		mock.ExpectQuery("SELECT.*SUM\\(difficulty\\).*FROM shares").
			WillReturnRows(sqlmock.NewRows([]string{"hashrate"}).AddRow(50000000000.0))

		// Mock active miners query
		mock.ExpectQuery("SELECT COUNT\\(DISTINCT miner_id\\).*FROM shares").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

		// Mock share counts query
		mock.ExpectQuery("SELECT.*COUNT.*FILTER.*FROM shares").
			WillReturnRows(sqlmock.NewRows([]string{"valid", "invalid"}).AddRow(1000, 10))

		// Mock blocks query
		mock.ExpectQuery("SELECT COUNT.*FROM blocks").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

		snapshot, err := service.GetCurrentPoolSnapshot(ctx)
		require.NoError(t, err)
		assert.NotNil(t, snapshot)
		assert.Equal(t, 50000000000.0, snapshot.TotalHashrate)
		assert.Equal(t, 5, snapshot.ActiveMiners)
		assert.Equal(t, int64(1000), snapshot.ValidShares)
		assert.Equal(t, int64(10), snapshot.InvalidShares)
		assert.InDelta(t, 99.0, snapshot.AcceptanceRate, 0.1)
		assert.Equal(t, 3, snapshot.BlocksFound)
	})

	t.Run("handles zero shares gracefully", func(t *testing.T) {
		mock.ExpectQuery("SELECT.*SUM\\(difficulty\\).*FROM shares").
			WillReturnRows(sqlmock.NewRows([]string{"hashrate"}).AddRow(0.0))
		mock.ExpectQuery("SELECT COUNT\\(DISTINCT miner_id\\).*FROM shares").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectQuery("SELECT.*COUNT.*FILTER.*FROM shares").
			WillReturnRows(sqlmock.NewRows([]string{"valid", "invalid"}).AddRow(0, 0))
		mock.ExpectQuery("SELECT COUNT.*FROM blocks").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		snapshot, err := service.GetCurrentPoolSnapshot(ctx)
		require.NoError(t, err)
		assert.Equal(t, 0.0, snapshot.TotalHashrate)
		assert.Equal(t, 0, snapshot.ActiveMiners)
		assert.Equal(t, 0.0, snapshot.AcceptanceRate)
	})
}

func TestRealtimeStatsService_GetHashrateHistory(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewRealtimeStatsService(db)
	ctx := context.Background()

	t.Run("returns hashrate history buckets", func(t *testing.T) {
		now := time.Now()
		mock.ExpectQuery("SELECT.*date_trunc.*FROM shares").
			WithArgs("hour", "24 hours").
			WillReturnRows(sqlmock.NewRows([]string{"time_bucket", "total_hashrate", "active_miners"}).
				AddRow(now.Add(-2*time.Hour), 45000000000.0, 3).
				AddRow(now.Add(-1*time.Hour), 50000000000.0, 4).
				AddRow(now, 55000000000.0, 5))

		buckets, err := service.GetHashrateHistory(ctx, "24h")
		require.NoError(t, err)
		assert.Len(t, buckets, 3)
		assert.Equal(t, 45000000000.0, buckets[0].TotalHashrate)
		assert.Equal(t, 5, buckets[2].ActiveUsers)
	})

	t.Run("handles empty result", func(t *testing.T) {
		mock.ExpectQuery("SELECT.*date_trunc.*FROM shares").
			WithArgs("minute", "1 hour").
			WillReturnRows(sqlmock.NewRows([]string{"time_bucket", "total_hashrate", "active_miners"}))

		buckets, err := service.GetHashrateHistory(ctx, "1h")
		require.NoError(t, err)
		assert.Len(t, buckets, 0)
	})
}

func TestRealtimeStatsService_GetSharesHistory(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewRealtimeStatsService(db)
	ctx := context.Background()

	t.Run("returns shares history with acceptance rate", func(t *testing.T) {
		now := time.Now()
		mock.ExpectQuery("SELECT.*date_trunc.*FROM shares").
			WithArgs("hour", "24 hours").
			WillReturnRows(sqlmock.NewRows([]string{"time_bucket", "valid_shares", "invalid_shares", "total_shares"}).
				AddRow(now.Add(-1*time.Hour), 950, 50, 1000).
				AddRow(now, 990, 10, 1000))

		buckets, err := service.GetSharesHistory(ctx, "24h")
		require.NoError(t, err)
		assert.Len(t, buckets, 2)
		assert.Equal(t, int64(950), buckets[0].ValidShares)
		assert.InDelta(t, 95.0, buckets[0].AcceptanceRate, 0.1)
		assert.InDelta(t, 99.0, buckets[1].AcceptanceRate, 0.1)
	})
}

func TestRealtimeStatsService_GetMinersHistory(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewRealtimeStatsService(db)
	ctx := context.Background()

	t.Run("returns miner activity history", func(t *testing.T) {
		now := time.Now()
		mock.ExpectQuery("SELECT.*date_trunc.*FROM shares").
			WithArgs("hour", "24 hours").
			WillReturnRows(sqlmock.NewRows([]string{"time_bucket", "active_miners", "unique_users"}).
				AddRow(now.Add(-1*time.Hour), 10, 8).
				AddRow(now, 12, 10))

		buckets, err := service.GetMinersHistory(ctx, "24h")
		require.NoError(t, err)
		assert.Len(t, buckets, 2)
		assert.Equal(t, 10, buckets[0].ActiveMiners)
		assert.Equal(t, 8, buckets[0].UniqueUsers)
	})
}

func TestRealtimeStatsService_CalculateHashrateFromShares(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewRealtimeStatsService(db)
	ctx := context.Background()

	t.Run("calculates hashrate from difficulty sum", func(t *testing.T) {
		// 100,000 difficulty in 10 minutes = 100000 * 2^32 / 600 ≈ 715 GH/s
		mock.ExpectQuery("SELECT COALESCE\\(SUM\\(difficulty\\)").
			WithArgs(10).
			WillReturnRows(sqlmock.NewRows([]string{"hashrate"}).AddRow(715000000000.0))

		hashrate, err := service.CalculateHashrateFromShares(ctx, 10)
		require.NoError(t, err)
		assert.Equal(t, 715000000000.0, hashrate)
	})
}

func TestIRealtimeStatsReader_Interface(t *testing.T) {
	// Verify RealtimeStatsService implements the interface
	var _ IRealtimeStatsReader = (*RealtimeStatsService)(nil)
}

func TestIStatsAggregator_Interface(t *testing.T) {
	// Verify RealtimeStatsService implements the aggregator interface
	var _ IStatsAggregator = (*RealtimeStatsService)(nil)
}

func TestPoolSnapshot_Fields(t *testing.T) {
	snapshot := PoolSnapshot{
		Timestamp:      time.Now(),
		TotalHashrate:  50000000000.0,
		ActiveMiners:   5,
		ValidShares:    1000,
		InvalidShares:  10,
		AcceptanceRate: 99.0,
		BlocksFound:    3,
	}

	assert.NotZero(t, snapshot.Timestamp)
	assert.Equal(t, 50000000000.0, snapshot.TotalHashrate)
	assert.Equal(t, 5, snapshot.ActiveMiners)
	assert.Equal(t, int64(1000), snapshot.ValidShares)
	assert.Equal(t, int64(10), snapshot.InvalidShares)
	assert.Equal(t, 99.0, snapshot.AcceptanceRate)
	assert.Equal(t, 3, snapshot.BlocksFound)
}
