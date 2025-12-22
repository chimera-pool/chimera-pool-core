package api

import (
	"database/sql"
	"errors"
	"time"
)

// =============================================================================
// POOL SERVICE IMPLEMENTATIONS
// ISP-compliant services for pool statistics and data
// =============================================================================

// -----------------------------------------------------------------------------
// Pool Stats Provider Implementation
// -----------------------------------------------------------------------------

// DBPoolStatsProvider implements pool statistics from database
type DBPoolStatsProvider struct {
	db *sql.DB
}

// NewDBPoolStatsProvider creates a new database pool stats provider
func NewDBPoolStatsProvider(db *sql.DB) *DBPoolStatsProvider {
	return &DBPoolStatsProvider{db: db}
}

// GetPoolStats returns overall pool statistics
func (p *DBPoolStatsProvider) GetPoolStats() (*PoolStats, error) {
	var totalMiners, blocksFound, totalShares, validShares int64
	var totalHashrate float64

	p.db.QueryRow("SELECT COUNT(*) FROM miners WHERE is_active = true").Scan(&totalMiners)
	p.db.QueryRow("SELECT COUNT(*) FROM blocks").Scan(&blocksFound)
	p.db.QueryRow("SELECT COALESCE(SUM(hashrate), 0) FROM miners WHERE is_active = true").Scan(&totalHashrate)
	p.db.QueryRow("SELECT COUNT(*) FROM shares").Scan(&totalShares)
	p.db.QueryRow("SELECT COUNT(*) FROM shares WHERE is_valid = true").Scan(&validShares)

	var lastBlockTime time.Time
	p.db.QueryRow("SELECT COALESCE(MAX(timestamp), NOW()) FROM blocks").Scan(&lastBlockTime)

	return &PoolStats{
		TotalHashrate:     totalHashrate,
		ConnectedMiners:   totalMiners,
		TotalShares:       totalShares,
		ValidShares:       validShares,
		BlocksFound:       blocksFound,
		LastBlockTime:     lastBlockTime,
		NetworkHashrate:   0, // Would come from blockchain client
		NetworkDifficulty: 0, // Would come from blockchain client
		PoolFee:           1.0,
	}, nil
}

// GetRealTimeStats returns real-time pool statistics
func (p *DBPoolStatsProvider) GetRealTimeStats() (*RealTimeStats, error) {
	stats, err := p.GetPoolStats()
	if err != nil {
		return nil, err
	}

	return &RealTimeStats{
		CurrentHashrate:   stats.TotalHashrate,
		AverageHashrate:   stats.TotalHashrate,
		ActiveMiners:      stats.ConnectedMiners,
		SharesPerSecond:   0,
		LastBlockFound:    stats.LastBlockTime,
		NetworkDifficulty: stats.NetworkDifficulty,
		PoolEfficiency:    100.0,
	}, nil
}

// GetBlockMetrics returns block-related metrics
func (p *DBPoolStatsProvider) GetBlockMetrics() (*BlockMetrics, error) {
	var totalBlocks, blocksLast24h, blocksLast7d, orphanBlocks int64
	var totalRewards, lastBlockReward float64

	p.db.QueryRow("SELECT COUNT(*) FROM blocks").Scan(&totalBlocks)
	p.db.QueryRow("SELECT COUNT(*) FROM blocks WHERE timestamp > NOW() - INTERVAL '24 hours'").Scan(&blocksLast24h)
	p.db.QueryRow("SELECT COUNT(*) FROM blocks WHERE timestamp > NOW() - INTERVAL '7 days'").Scan(&blocksLast7d)
	p.db.QueryRow("SELECT COUNT(*) FROM blocks WHERE status = 'orphan'").Scan(&orphanBlocks)
	p.db.QueryRow("SELECT COALESCE(SUM(reward), 0) FROM blocks WHERE status = 'confirmed'").Scan(&totalRewards)
	p.db.QueryRow("SELECT COALESCE(reward, 0) FROM blocks ORDER BY height DESC LIMIT 1").Scan(&lastBlockReward)

	orphanRate := float64(0)
	if totalBlocks > 0 {
		orphanRate = float64(orphanBlocks) / float64(totalBlocks) * 100
	}

	return &BlockMetrics{
		TotalBlocks:      totalBlocks,
		BlocksLast24h:    blocksLast24h,
		BlocksLast7d:     blocksLast7d,
		AverageBlockTime: 0, // Would need calculation
		LastBlockReward:  lastBlockReward,
		TotalRewards:     totalRewards,
		OrphanBlocks:     orphanBlocks,
		OrphanRate:       orphanRate,
	}, nil
}

// -----------------------------------------------------------------------------
// Block Data Types
// -----------------------------------------------------------------------------

// BlockInfo represents a mined block
type BlockInfo struct {
	ID        int64     `json:"id"`
	Height    int64     `json:"height"`
	Hash      string    `json:"hash"`
	Reward    float64   `json:"reward"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

// MinerLocation represents a miner's geographic location
type MinerLocation struct {
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lng"`
	Hashrate  float64 `json:"hashrate"`
	UserID    int64   `json:"user_id,omitempty"`
	IsPublic  bool    `json:"is_public,omitempty"`
}

// -----------------------------------------------------------------------------
// Block Reader Implementation
// -----------------------------------------------------------------------------

// DBBlockReader reads block data from database
type DBBlockReader struct {
	db *sql.DB
}

// NewDBBlockReader creates a new database block reader
func NewDBBlockReader(db *sql.DB) *DBBlockReader {
	return &DBBlockReader{db: db}
}

// GetRecentBlocks returns recent blocks
func (r *DBBlockReader) GetRecentBlocks(limit int) ([]*BlockInfo, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	rows, err := r.db.Query(
		"SELECT id, height, hash, reward, status, timestamp FROM blocks ORDER BY height DESC LIMIT $1",
		limit,
	)
	if err != nil {
		return nil, errors.New("failed to fetch blocks")
	}
	defer rows.Close()

	var blocks []*BlockInfo
	for rows.Next() {
		var b BlockInfo
		err := rows.Scan(&b.ID, &b.Height, &b.Hash, &b.Reward, &b.Status, &b.Timestamp)
		if err != nil {
			continue
		}
		blocks = append(blocks, &b)
	}

	return blocks, nil
}

// -----------------------------------------------------------------------------
// Miner Location Reader Implementation
// -----------------------------------------------------------------------------

// DBMinerLocationReader reads miner locations from database
type DBMinerLocationReader struct {
	db *sql.DB
}

// NewDBMinerLocationReader creates a new database miner location reader
func NewDBMinerLocationReader(db *sql.DB) *DBMinerLocationReader {
	return &DBMinerLocationReader{db: db}
}

// GetPublicLocations returns public miner locations
func (r *DBMinerLocationReader) GetPublicLocations() ([]*MinerLocation, error) {
	rows, err := r.db.Query(
		"SELECT latitude, longitude, hashrate FROM miner_locations WHERE is_public = true",
	)
	if err != nil {
		return nil, errors.New("failed to fetch locations")
	}
	defer rows.Close()

	var locations []*MinerLocation
	for rows.Next() {
		var l MinerLocation
		err := rows.Scan(&l.Latitude, &l.Longitude, &l.Hashrate)
		if err != nil {
			continue
		}
		locations = append(locations, &l)
	}

	return locations, nil
}

// GetAllLocations returns all miner locations (admin only)
func (r *DBMinerLocationReader) GetAllLocations() ([]*MinerLocation, error) {
	rows, err := r.db.Query(
		"SELECT latitude, longitude, hashrate, user_id, is_public FROM miner_locations",
	)
	if err != nil {
		return nil, errors.New("failed to fetch locations")
	}
	defer rows.Close()

	var locations []*MinerLocation
	for rows.Next() {
		var l MinerLocation
		err := rows.Scan(&l.Latitude, &l.Longitude, &l.Hashrate, &l.UserID, &l.IsPublic)
		if err != nil {
			continue
		}
		locations = append(locations, &l)
	}

	return locations, nil
}

// GetLocationStats returns miner location statistics
func (r *DBMinerLocationReader) GetLocationStats() (int64, error) {
	var count int64
	err := r.db.QueryRow("SELECT COUNT(*) FROM miner_locations WHERE is_public = true").Scan(&count)
	if err != nil {
		return 0, errors.New("failed to fetch location stats")
	}
	return count, nil
}

// =============================================================================
// POOL SERVICE FACTORY
// =============================================================================

// PoolServices holds all pool-related service implementations
type PoolServices struct {
	StatsProvider  PoolStatsProvider
	BlockReader    *DBBlockReader
	LocationReader *DBMinerLocationReader
}

// NewPoolServices creates all pool services
func NewPoolServices(db *sql.DB) *PoolServices {
	return &PoolServices{
		StatsProvider:  NewDBPoolStatsProvider(db),
		BlockReader:    NewDBBlockReader(db),
		LocationReader: NewDBMinerLocationReader(db),
	}
}
