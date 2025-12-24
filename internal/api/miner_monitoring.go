package api

import (
	"database/sql"
	"fmt"
	"time"
)

// =============================================================================
// MINER MONITORING SERVICE
// Comprehensive miner monitoring for admin troubleshooting and analysis
// =============================================================================

// -----------------------------------------------------------------------------
// Data Types
// -----------------------------------------------------------------------------

// MinerDetail provides comprehensive miner information for admin monitoring
type MinerDetail struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Username  string    `json:"username"`
	Name      string    `json:"name"`
	Address   string    `json:"address"`
	Hashrate  float64   `json:"hashrate"`
	IsActive  bool      `json:"is_active"`
	LastSeen  time.Time `json:"last_seen"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Connection info
	ConnectionDuration string  `json:"connection_duration"`
	UptimePercent      float64 `json:"uptime_percent"`

	// Share statistics
	ShareStats ShareStats `json:"share_stats"`

	// Performance metrics
	Performance PerformanceMetrics `json:"performance"`

	// Recent activity
	RecentShares []ShareDetail `json:"recent_shares"`
}

// ShareStats contains share statistics for a miner
type ShareStats struct {
	TotalShares     int64   `json:"total_shares"`
	ValidShares     int64   `json:"valid_shares"`
	InvalidShares   int64   `json:"invalid_shares"`
	StaleShares     int64   `json:"stale_shares"`
	AcceptanceRate  float64 `json:"acceptance_rate"`
	Last24Hours     int64   `json:"last_24_hours"`
	LastHour        int64   `json:"last_hour"`
	AvgDifficulty   float64 `json:"avg_difficulty"`
	TotalDifficulty float64 `json:"total_difficulty"`
}

// PerformanceMetrics contains performance analysis data
type PerformanceMetrics struct {
	EffectiveHashrate    float64 `json:"effective_hashrate"`
	ReportedHashrate     float64 `json:"reported_hashrate"`
	HashrateDelta        float64 `json:"hashrate_delta"`
	SharesPerMinute      float64 `json:"shares_per_minute"`
	AvgShareTime         float64 `json:"avg_share_time_seconds"`
	Efficiency           float64 `json:"efficiency_percent"`
	EstimatedDailyShares int64   `json:"estimated_daily_shares"`
}

// ShareDetail provides individual share information
type ShareDetail struct {
	ID         int64     `json:"id"`
	Difficulty float64   `json:"difficulty"`
	IsValid    bool      `json:"is_valid"`
	Nonce      string    `json:"nonce"`
	Hash       string    `json:"hash"`
	Timestamp  time.Time `json:"timestamp"`
	TimeSince  string    `json:"time_since"`
}

// UserMinerSummary provides an overview of a user's miners
type UserMinerSummary struct {
	UserID         int64          `json:"user_id"`
	Username       string         `json:"username"`
	Email          string         `json:"email"`
	TotalMiners    int            `json:"total_miners"`
	ActiveMiners   int            `json:"active_miners"`
	InactiveMiners int            `json:"inactive_miners"`
	TotalHashrate  float64        `json:"total_hashrate"`
	TotalShares24h int64          `json:"total_shares_24h"`
	AcceptanceRate float64        `json:"acceptance_rate"`
	Miners         []MinerSummary `json:"miners"`
}

// MinerSummary provides a brief overview of a miner
type MinerSummary struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Address      string    `json:"address"`
	Hashrate     float64   `json:"hashrate"`
	IsActive     bool      `json:"is_active"`
	LastSeen     time.Time `json:"last_seen"`
	Shares24h    int64     `json:"shares_24h"`
	ValidPercent float64   `json:"valid_percent"`
}

// ConnectionEvent tracks miner connection history
type ConnectionEvent struct {
	Timestamp time.Time `json:"timestamp"`
	EventType string    `json:"event_type"` // connect, disconnect, authorize, error
	Details   string    `json:"details"`
	IPAddress string    `json:"ip_address"`
}

// -----------------------------------------------------------------------------
// Service Interface (ISP)
// -----------------------------------------------------------------------------

// MinerMonitoringService provides miner monitoring capabilities
type MinerMonitoringService interface {
	// GetUserMinerSummary returns an overview of all miners for a user
	GetUserMinerSummary(userID int64) (*UserMinerSummary, error)

	// GetMinerDetail returns comprehensive details for a specific miner
	GetMinerDetail(minerID int64) (*MinerDetail, error)

	// GetAllMinersForAdmin returns all miners with summary info for admin view
	GetAllMinersForAdmin(page, limit int, search string, activeOnly bool) ([]*MinerSummary, int64, error)

	// GetMinerShareHistory returns share history for a miner
	GetMinerShareHistory(minerID int64, limit int) ([]ShareDetail, error)
}

// -----------------------------------------------------------------------------
// Implementation
// -----------------------------------------------------------------------------

// DBMinerMonitoringService implements MinerMonitoringService with database
type DBMinerMonitoringService struct {
	db *sql.DB
}

// NewMinerMonitoringService creates a new miner monitoring service
func NewMinerMonitoringService(db *sql.DB) *DBMinerMonitoringService {
	return &DBMinerMonitoringService{db: db}
}

// GetUserMinerSummary returns an overview of all miners for a user
func (s *DBMinerMonitoringService) GetUserMinerSummary(userID int64) (*UserMinerSummary, error) {
	// Get user info
	var summary UserMinerSummary
	err := s.db.QueryRow(`
		SELECT id, username, email 
		FROM users WHERE id = $1`,
		userID,
	).Scan(&summary.UserID, &summary.Username, &summary.Email)
	if err != nil {
		return nil, err
	}

	// Get all miners for this user with their stats
	rows, err := s.db.Query(`
		SELECT 
			m.id, m.name, COALESCE(m.address::text, ''), m.hashrate, m.is_active, m.last_seen,
			COALESCE(s.shares_24h, 0) as shares_24h,
			COALESCE(s.valid_percent, 0) as valid_percent
		FROM miners m
		LEFT JOIN LATERAL (
			SELECT 
				COUNT(*) as shares_24h,
				CASE WHEN COUNT(*) > 0 
					THEN (COUNT(*) FILTER (WHERE is_valid = true)::float / COUNT(*)::float) * 100 
					ELSE 0 
				END as valid_percent
			FROM shares 
			WHERE miner_id = m.id AND timestamp > NOW() - INTERVAL '24 hours'
		) s ON true
		WHERE m.user_id = $1
		ORDER BY m.is_active DESC, m.last_seen DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var miner MinerSummary
		err := rows.Scan(
			&miner.ID, &miner.Name, &miner.Address, &miner.Hashrate,
			&miner.IsActive, &miner.LastSeen, &miner.Shares24h, &miner.ValidPercent,
		)
		if err != nil {
			continue
		}
		summary.Miners = append(summary.Miners, miner)
		summary.TotalMiners++
		if miner.IsActive {
			summary.ActiveMiners++
		} else {
			summary.InactiveMiners++
		}
		summary.TotalHashrate += miner.Hashrate
		summary.TotalShares24h += miner.Shares24h
	}

	// Calculate overall acceptance rate
	if summary.TotalShares24h > 0 {
		var totalValid int64
		for _, m := range summary.Miners {
			totalValid += int64(float64(m.Shares24h) * m.ValidPercent / 100)
		}
		summary.AcceptanceRate = float64(totalValid) / float64(summary.TotalShares24h) * 100
	}

	return &summary, nil
}

// GetMinerDetail returns comprehensive details for a specific miner
func (s *DBMinerMonitoringService) GetMinerDetail(minerID int64) (*MinerDetail, error) {
	var detail MinerDetail

	// Get miner info with user
	err := s.db.QueryRow(`
		SELECT 
			m.id, m.user_id, u.username, m.name, 
			COALESCE(m.address::text, ''), m.hashrate, m.is_active, 
			m.last_seen, m.created_at, m.updated_at
		FROM miners m
		JOIN users u ON m.user_id = u.id
		WHERE m.id = $1`,
		minerID,
	).Scan(
		&detail.ID, &detail.UserID, &detail.Username, &detail.Name,
		&detail.Address, &detail.Hashrate, &detail.IsActive,
		&detail.LastSeen, &detail.CreatedAt, &detail.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Calculate connection duration
	if detail.IsActive {
		duration := time.Since(detail.LastSeen)
		if duration < time.Hour {
			detail.ConnectionDuration = "Active now"
		} else {
			detail.ConnectionDuration = formatDuration(time.Since(detail.CreatedAt))
		}
	} else {
		detail.ConnectionDuration = "Disconnected"
	}

	// Calculate uptime (simplified - based on activity in last 24h)
	var activeMinutes int64
	s.db.QueryRow(`
		SELECT COUNT(DISTINCT DATE_TRUNC('minute', timestamp))
		FROM shares
		WHERE miner_id = $1 AND timestamp > NOW() - INTERVAL '24 hours'`,
		minerID,
	).Scan(&activeMinutes)
	detail.UptimePercent = float64(activeMinutes) / 1440.0 * 100 // 1440 minutes in 24 hours

	// Get share statistics
	err = s.db.QueryRow(`
		SELECT 
			COALESCE(COUNT(*), 0) as total,
			COALESCE(COUNT(*) FILTER (WHERE is_valid = true), 0) as valid,
			COALESCE(COUNT(*) FILTER (WHERE is_valid = false), 0) as invalid,
			COALESCE(AVG(difficulty), 0) as avg_diff,
			COALESCE(SUM(difficulty), 0) as total_diff,
			COALESCE(COUNT(*) FILTER (WHERE timestamp > NOW() - INTERVAL '24 hours'), 0) as last_24h,
			COALESCE(COUNT(*) FILTER (WHERE timestamp > NOW() - INTERVAL '1 hour'), 0) as last_hour
		FROM shares
		WHERE miner_id = $1`,
		minerID,
	).Scan(
		&detail.ShareStats.TotalShares,
		&detail.ShareStats.ValidShares,
		&detail.ShareStats.InvalidShares,
		&detail.ShareStats.AvgDifficulty,
		&detail.ShareStats.TotalDifficulty,
		&detail.ShareStats.Last24Hours,
		&detail.ShareStats.LastHour,
	)
	if err != nil {
		return nil, err
	}

	// Calculate acceptance rate
	if detail.ShareStats.TotalShares > 0 {
		detail.ShareStats.AcceptanceRate = float64(detail.ShareStats.ValidShares) / float64(detail.ShareStats.TotalShares) * 100
	}

	// Calculate performance metrics
	detail.Performance.ReportedHashrate = detail.Hashrate

	// Calculate effective hashrate from shares (simplified)
	if detail.ShareStats.Last24Hours > 0 {
		// Effective hashrate = (shares * difficulty) / time
		detail.Performance.SharesPerMinute = float64(detail.ShareStats.LastHour) / 60.0
		detail.Performance.AvgShareTime = 60.0 / detail.Performance.SharesPerMinute
		if detail.Performance.SharesPerMinute > 0 {
			detail.Performance.EffectiveHashrate = detail.Performance.SharesPerMinute * detail.ShareStats.AvgDifficulty * 4294967296 / 60 // GH/s approximation
		}
		detail.Performance.EstimatedDailyShares = detail.ShareStats.LastHour * 24
	}

	// Calculate efficiency
	if detail.Performance.ReportedHashrate > 0 && detail.Performance.EffectiveHashrate > 0 {
		detail.Performance.Efficiency = (detail.Performance.EffectiveHashrate / detail.Performance.ReportedHashrate) * 100
	}

	detail.Performance.HashrateDelta = detail.Performance.EffectiveHashrate - detail.Performance.ReportedHashrate

	// Get recent shares
	detail.RecentShares, _ = s.GetMinerShareHistory(minerID, 20)

	return &detail, nil
}

// GetAllMinersForAdmin returns all miners with summary info for admin view
func (s *DBMinerMonitoringService) GetAllMinersForAdmin(page, limit int, search string, activeOnly bool) ([]*MinerSummary, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	offset := (page - 1) * limit

	var totalCount int64
	var rows *sql.Rows
	var err error

	baseQuery := `
		SELECT 
			m.id, m.name, COALESCE(m.address::text, ''), 
			CASE 
				WHEN m.hashrate > 0 THEN m.hashrate
				ELSE COALESCE(calc.calculated_hashrate, 0)
			END as hashrate,
			m.is_active, m.last_seen,
			COALESCE(s.shares_24h, 0) as shares_24h,
			COALESCE(s.valid_percent, 0) as valid_percent
		FROM miners m
		LEFT JOIN LATERAL (
			SELECT 
				COUNT(*) as shares_24h,
				CASE WHEN COUNT(*) > 0 
					THEN (COUNT(*) FILTER (WHERE is_valid = true)::float / COUNT(*)::float) * 100 
					ELSE 0 
				END as valid_percent
			FROM shares 
			WHERE miner_id = m.id AND timestamp > NOW() - INTERVAL '24 hours'
		) s ON true
		LEFT JOIN LATERAL (
			SELECT 
				(COALESCE(SUM(difficulty), 0) * 4294967296.0 / 300.0) as calculated_hashrate
			FROM shares 
			WHERE miner_id = m.id AND timestamp > NOW() - INTERVAL '5 minutes' AND is_valid = true
		) calc ON true`

	if search != "" {
		searchPattern := "%" + search + "%"
		if activeOnly {
			s.db.QueryRow("SELECT COUNT(*) FROM miners WHERE (name ILIKE $1) AND is_active = true", searchPattern).Scan(&totalCount)
			rows, err = s.db.Query(baseQuery+" WHERE (m.name ILIKE $1) AND m.is_active = true ORDER BY m.last_seen DESC LIMIT $2 OFFSET $3", searchPattern, limit, offset)
		} else {
			s.db.QueryRow("SELECT COUNT(*) FROM miners WHERE name ILIKE $1", searchPattern).Scan(&totalCount)
			rows, err = s.db.Query(baseQuery+" WHERE m.name ILIKE $1 ORDER BY m.is_active DESC, m.last_seen DESC LIMIT $2 OFFSET $3", searchPattern, limit, offset)
		}
	} else {
		if activeOnly {
			s.db.QueryRow("SELECT COUNT(*) FROM miners WHERE is_active = true").Scan(&totalCount)
			rows, err = s.db.Query(baseQuery+" WHERE m.is_active = true ORDER BY m.last_seen DESC LIMIT $1 OFFSET $2", limit, offset)
		} else {
			s.db.QueryRow("SELECT COUNT(*) FROM miners").Scan(&totalCount)
			rows, err = s.db.Query(baseQuery+" ORDER BY m.is_active DESC, m.last_seen DESC LIMIT $1 OFFSET $2", limit, offset)
		}
	}

	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var miners []*MinerSummary
	for rows.Next() {
		var miner MinerSummary
		err := rows.Scan(
			&miner.ID, &miner.Name, &miner.Address, &miner.Hashrate,
			&miner.IsActive, &miner.LastSeen, &miner.Shares24h, &miner.ValidPercent,
		)
		if err != nil {
			continue
		}
		miners = append(miners, &miner)
	}

	return miners, totalCount, nil
}

// GetMinerShareHistory returns share history for a miner
func (s *DBMinerMonitoringService) GetMinerShareHistory(minerID int64, limit int) ([]ShareDetail, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := s.db.Query(`
		SELECT id, difficulty, is_valid, nonce, hash, timestamp
		FROM shares
		WHERE miner_id = $1
		ORDER BY timestamp DESC
		LIMIT $2`,
		minerID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shares []ShareDetail
	for rows.Next() {
		var share ShareDetail
		err := rows.Scan(&share.ID, &share.Difficulty, &share.IsValid, &share.Nonce, &share.Hash, &share.Timestamp)
		if err != nil {
			continue
		}
		share.TimeSince = formatDuration(time.Since(share.Timestamp))
		shares = append(shares, share)
	}

	return shares, nil
}

// formatDuration formats a duration into a human-readable string
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	} else if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%dh %dm ago", int(d.Hours()), int(d.Minutes())%60)
	} else {
		days := int(d.Hours()) / 24
		hours := int(d.Hours()) % 24
		return fmt.Sprintf("%dd %dh ago", days, hours)
	}
}
