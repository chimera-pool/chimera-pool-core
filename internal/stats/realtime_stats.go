package stats

import (
	"context"
	"database/sql"
	"time"
)

// PoolSnapshot represents a point-in-time snapshot of pool statistics
type PoolSnapshot struct {
	Timestamp      time.Time `json:"timestamp"`
	TotalHashrate  float64   `json:"totalHashrate"`
	ActiveMiners   int       `json:"activeMiners"`
	ValidShares    int64     `json:"validShares"`
	InvalidShares  int64     `json:"invalidShares"`
	AcceptanceRate float64   `json:"acceptanceRate"`
	BlocksFound    int       `json:"blocksFound"`
}

// MinerSnapshot represents a point-in-time snapshot of a miner's statistics
type MinerSnapshot struct {
	Timestamp     time.Time `json:"timestamp"`
	MinerID       int64     `json:"minerId"`
	Username      string    `json:"username"`
	Hashrate      float64   `json:"hashrate"`
	ValidShares   int64     `json:"validShares"`
	InvalidShares int64     `json:"invalidShares"`
	Efficiency    float64   `json:"efficiency"`
}

// HashrateBucket represents hashrate data for a time bucket
type HashrateBucket struct {
	Time          time.Time `json:"time"`
	TotalHashrate float64   `json:"totalHashrate"`
	AvgHashrate   float64   `json:"avgHashrate"`
	ActiveUsers   int       `json:"activeUsers"`
}

// SharesBucket represents shares data for a time bucket
type SharesBucket struct {
	Time           time.Time `json:"time"`
	ValidShares    int64     `json:"validShares"`
	InvalidShares  int64     `json:"invalidShares"`
	TotalShares    int64     `json:"totalShares"`
	AcceptanceRate float64   `json:"acceptanceRate"`
}

// MinersBucket represents miner activity for a time bucket
type MinersBucket struct {
	Time         time.Time `json:"time"`
	ActiveMiners int       `json:"activeMiners"`
	UniqueUsers  int       `json:"uniqueUsers"`
	NewMiners    int       `json:"newMiners"`
}

// IRealtimeStatsReader provides read-only access to real-time pool statistics (ISP)
type IRealtimeStatsReader interface {
	GetCurrentPoolSnapshot(ctx context.Context) (*PoolSnapshot, error)
	GetHashrateHistory(ctx context.Context, timeRange string) ([]HashrateBucket, error)
	GetSharesHistory(ctx context.Context, timeRange string) ([]SharesBucket, error)
	GetMinersHistory(ctx context.Context, timeRange string) ([]MinersBucket, error)
}

// IRealtimeMinerStatsReader provides read-only access to real-time miner statistics (ISP)
type IRealtimeMinerStatsReader interface {
	GetMinerSnapshot(ctx context.Context, minerID int64) (*MinerSnapshot, error)
	GetMinerHashrateHistory(ctx context.Context, minerID int64, timeRange string) ([]HashrateBucket, error)
}

// IStatsAggregator aggregates statistics from raw data (ISP)
type IStatsAggregator interface {
	CalculateHashrateFromShares(ctx context.Context, windowMinutes int) (float64, error)
	CalculateMinerHashrate(ctx context.Context, minerID int64, windowMinutes int) (float64, error)
}

// RealtimeStatsService implements real-time statistics retrieval
type RealtimeStatsService struct {
	db           *sql.DB
	timeRangeSvc TimeRangeService
}

// NewRealtimeStatsService creates a new real-time stats service
func NewRealtimeStatsService(db *sql.DB) *RealtimeStatsService {
	return &RealtimeStatsService{
		db:           db,
		timeRangeSvc: NewTimeRangeService(),
	}
}

// GetCurrentPoolSnapshot returns the current pool statistics
func (s *RealtimeStatsService) GetCurrentPoolSnapshot(ctx context.Context) (*PoolSnapshot, error) {
	snapshot := &PoolSnapshot{
		Timestamp: time.Now(),
	}

	// Get hashrate from recent shares (last 10 minutes for accuracy)
	err := s.db.QueryRowContext(ctx, `
		SELECT 
			COALESCE(SUM(difficulty) * 4294967296 / 600, 0) as hashrate
		FROM shares 
		WHERE timestamp > NOW() - INTERVAL '10 minutes' AND is_valid = true
	`).Scan(&snapshot.TotalHashrate)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Get active miners count
	err = s.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT miner_id) 
		FROM shares 
		WHERE timestamp > NOW() - INTERVAL '10 minutes'
	`).Scan(&snapshot.ActiveMiners)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Get share counts for last hour
	err = s.db.QueryRowContext(ctx, `
		SELECT 
			COALESCE(COUNT(*) FILTER (WHERE is_valid = true), 0),
			COALESCE(COUNT(*) FILTER (WHERE is_valid = false), 0)
		FROM shares 
		WHERE timestamp > NOW() - INTERVAL '1 hour'
	`).Scan(&snapshot.ValidShares, &snapshot.InvalidShares)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	total := snapshot.ValidShares + snapshot.InvalidShares
	if total > 0 {
		snapshot.AcceptanceRate = float64(snapshot.ValidShares) / float64(total) * 100
	}

	// Get blocks found
	err = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM blocks
	`).Scan(&snapshot.BlocksFound)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	return snapshot, nil
}

// GetHashrateHistory returns hashrate history for the given time range
func (s *RealtimeStatsService) GetHashrateHistory(ctx context.Context, timeRange string) ([]HashrateBucket, error) {
	_ = s.timeRangeSvc.ParseRange(timeRange) // duration for reference
	interval := s.timeRangeSvc.GetPostgresInterval(timeRange)
	truncUnit := s.timeRangeSvc.GetDateTrunc(timeRange)

	query := `
		SELECT 
			date_trunc($1, timestamp) as time_bucket,
			COALESCE(SUM(difficulty) * 4294967296 / EXTRACT(EPOCH FROM $2::interval), 0) as total_hashrate,
			COUNT(DISTINCT miner_id) as active_miners
		FROM shares 
		WHERE timestamp > NOW() - $2::interval AND is_valid = true
		GROUP BY time_bucket
		ORDER BY time_bucket ASC
	`

	rows, err := s.db.QueryContext(ctx, query, truncUnit, interval)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var buckets []HashrateBucket
	for rows.Next() {
		var bucket HashrateBucket
		var activeMiners int
		if err := rows.Scan(&bucket.Time, &bucket.TotalHashrate, &activeMiners); err != nil {
			continue
		}
		bucket.ActiveUsers = activeMiners
		bucket.AvgHashrate = bucket.TotalHashrate / float64(max(activeMiners, 1))
		buckets = append(buckets, bucket)
	}

	return buckets, nil
}

// GetSharesHistory returns shares history for the given time range
func (s *RealtimeStatsService) GetSharesHistory(ctx context.Context, timeRange string) ([]SharesBucket, error) {
	_ = s.timeRangeSvc.ParseRange(timeRange) // duration for reference
	interval := s.timeRangeSvc.GetPostgresInterval(timeRange)
	truncUnit := s.timeRangeSvc.GetDateTrunc(timeRange)

	query := `
		SELECT 
			date_trunc($1, timestamp) as time_bucket,
			COUNT(*) FILTER (WHERE is_valid = true) as valid_shares,
			COUNT(*) FILTER (WHERE is_valid = false) as invalid_shares,
			COUNT(*) as total_shares
		FROM shares 
		WHERE timestamp > NOW() - $2::interval
		GROUP BY time_bucket
		ORDER BY time_bucket ASC
	`

	rows, err := s.db.QueryContext(ctx, query, truncUnit, interval)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var buckets []SharesBucket
	for rows.Next() {
		var bucket SharesBucket
		if err := rows.Scan(&bucket.Time, &bucket.ValidShares, &bucket.InvalidShares, &bucket.TotalShares); err != nil {
			continue
		}
		if bucket.TotalShares > 0 {
			bucket.AcceptanceRate = float64(bucket.ValidShares) / float64(bucket.TotalShares) * 100
		}
		buckets = append(buckets, bucket)
	}

	return buckets, nil
}

// GetMinersHistory returns miner activity history for the given time range
func (s *RealtimeStatsService) GetMinersHistory(ctx context.Context, timeRange string) ([]MinersBucket, error) {
	_ = s.timeRangeSvc.ParseRange(timeRange) // duration for reference
	interval := s.timeRangeSvc.GetPostgresInterval(timeRange)
	truncUnit := s.timeRangeSvc.GetDateTrunc(timeRange)

	query := `
		SELECT 
			date_trunc($1, timestamp) as time_bucket,
			COUNT(DISTINCT miner_id) as active_miners,
			COUNT(DISTINCT user_id) as unique_users
		FROM shares 
		WHERE timestamp > NOW() - $2::interval
		GROUP BY time_bucket
		ORDER BY time_bucket ASC
	`

	rows, err := s.db.QueryContext(ctx, query, truncUnit, interval)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var buckets []MinersBucket
	for rows.Next() {
		var bucket MinersBucket
		if err := rows.Scan(&bucket.Time, &bucket.ActiveMiners, &bucket.UniqueUsers); err != nil {
			continue
		}
		buckets = append(buckets, bucket)
	}

	return buckets, nil
}

// CalculateHashrateFromShares calculates hashrate from recent shares
func (s *RealtimeStatsService) CalculateHashrateFromShares(ctx context.Context, windowMinutes int) (float64, error) {
	var hashrate float64
	query := `
		SELECT COALESCE(SUM(difficulty) * 4294967296 / ($1 * 60), 0)
		FROM shares 
		WHERE timestamp > NOW() - ($1 || ' minutes')::interval AND is_valid = true
	`
	err := s.db.QueryRowContext(ctx, query, windowMinutes).Scan(&hashrate)
	return hashrate, err
}

// CalculateMinerHashrate calculates a specific miner's hashrate
func (s *RealtimeStatsService) CalculateMinerHashrate(ctx context.Context, minerID int64, windowMinutes int) (float64, error) {
	var hashrate float64
	query := `
		SELECT COALESCE(SUM(difficulty) * 4294967296 / ($1 * 60), 0)
		FROM shares 
		WHERE miner_id = $2 AND timestamp > NOW() - ($1 || ' minutes')::interval AND is_valid = true
	`
	err := s.db.QueryRowContext(ctx, query, windowMinutes, minerID).Scan(&hashrate)
	return hashrate, err
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
