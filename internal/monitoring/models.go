package monitoring

import (
	"time"
	"github.com/google/uuid"
)

// Metric represents a monitoring metric
type Metric struct {
	Name      string            `json:"name"`
	Value     float64           `json:"value"`
	Labels    map[string]string `json:"labels"`
	Timestamp time.Time         `json:"timestamp"`
	Type      string            `json:"type"` // counter, gauge, histogram, summary
}

// Alert represents a monitoring alert
type Alert struct {
	ID          uuid.UUID         `json:"id" db:"id"`
	Name        string            `json:"name" db:"name"`
	Description string            `json:"description" db:"description"`
	Severity    string            `json:"severity" db:"severity"` // info, warning, error, critical
	Status      string            `json:"status" db:"status"`     // active, resolved, silenced
	Labels      map[string]string `json:"labels" db:"labels"`
	CreatedAt   time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at" db:"updated_at"`
	ResolvedAt  *time.Time        `json:"resolved_at" db:"resolved_at"`
}

// AlertRule represents an alerting rule
type AlertRule struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Query       string    `json:"query" db:"query"`
	Condition   string    `json:"condition" db:"condition"` // >, <, >=, <=, ==, !=
	Threshold   float64   `json:"threshold" db:"threshold"`
	Duration    string    `json:"duration" db:"duration"` // 5m, 10m, 1h
	Severity    string    `json:"severity" db:"severity"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// Dashboard represents a monitoring dashboard
type Dashboard struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	Config      string    `json:"config" db:"config"` // JSON configuration
	IsPublic    bool      `json:"is_public" db:"is_public"`
	CreatedBy   uuid.UUID `json:"created_by" db:"created_by"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// PerformanceMetrics represents system performance metrics
type PerformanceMetrics struct {
	Timestamp       time.Time `json:"timestamp"`
	CPUUsage        float64   `json:"cpu_usage"`
	MemoryUsage     float64   `json:"memory_usage"`
	DiskUsage       float64   `json:"disk_usage"`
	NetworkIn       float64   `json:"network_in"`
	NetworkOut      float64   `json:"network_out"`
	ActiveMiners    int       `json:"active_miners"`
	TotalHashrate   float64   `json:"total_hashrate"`
	SharesPerSecond float64   `json:"shares_per_second"`
	BlocksFound     int       `json:"blocks_found"`
	Uptime          float64   `json:"uptime"`
}

// MinerMetrics represents individual miner metrics
type MinerMetrics struct {
	MinerID         uuid.UUID `json:"miner_id"`
	Timestamp       time.Time `json:"timestamp"`
	Hashrate        float64   `json:"hashrate"`
	SharesSubmitted int64     `json:"shares_submitted"`
	SharesAccepted  int64     `json:"shares_accepted"`
	SharesRejected  int64     `json:"shares_rejected"`
	LastSeen        time.Time `json:"last_seen"`
	IsOnline        bool      `json:"is_online"`
	Difficulty      float64   `json:"difficulty"`
	Earnings        float64   `json:"earnings"`
}

// PoolMetrics represents pool-wide metrics
type PoolMetrics struct {
	Timestamp         time.Time `json:"timestamp"`
	TotalHashrate     float64   `json:"total_hashrate"`
	ActiveMiners      int       `json:"active_miners"`
	TotalShares       int64     `json:"total_shares"`
	ValidShares       int64     `json:"valid_shares"`
	InvalidShares     int64     `json:"invalid_shares"`
	BlocksFound       int       `json:"blocks_found"`
	NetworkDifficulty float64   `json:"network_difficulty"`
	PoolDifficulty    float64   `json:"pool_difficulty"`
	NetworkHashrate   float64   `json:"network_hashrate"`
	PoolEfficiency    float64   `json:"pool_efficiency"`
	Luck              float64   `json:"luck"`
}

// AlertChannel represents notification channels for alerts
type AlertChannel struct {
	ID       uuid.UUID         `json:"id" db:"id"`
	Name     string            `json:"name" db:"name"`
	Type     string            `json:"type" db:"type"` // email, slack, discord, webhook
	Config   map[string]string `json:"config" db:"config"`
	IsActive bool              `json:"is_active" db:"is_active"`
}

// Notification represents a sent notification
type Notification struct {
	ID        uuid.UUID `json:"id" db:"id"`
	AlertID   uuid.UUID `json:"alert_id" db:"alert_id"`
	ChannelID uuid.UUID `json:"channel_id" db:"channel_id"`
	Status    string    `json:"status" db:"status"` // sent, failed, pending
	SentAt    time.Time `json:"sent_at" db:"sent_at"`
	Error     string    `json:"error" db:"error"`
}