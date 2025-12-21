package equipment

import (
	"context"
	"time"
)

// EquipmentStatus represents the current state of mining equipment
type EquipmentStatus string

const (
	StatusOnline      EquipmentStatus = "online"
	StatusOffline     EquipmentStatus = "offline"
	StatusMining      EquipmentStatus = "mining"
	StatusIdle        EquipmentStatus = "idle"
	StatusError       EquipmentStatus = "error"
	StatusMaintenance EquipmentStatus = "maintenance"
)

// EquipmentType categorizes the mining hardware
type EquipmentType string

const (
	TypeASIC         EquipmentType = "asic"
	TypeGPU          EquipmentType = "gpu"
	TypeCPU          EquipmentType = "cpu"
	TypeFPGA         EquipmentType = "fpga"
	TypeOfficialX30  EquipmentType = "blockdag_x30"
	TypeOfficialX100 EquipmentType = "blockdag_x100"
)

// Equipment represents a single piece of mining hardware
type Equipment struct {
	ID         string          `json:"id"`
	UserID     string          `json:"user_id"`
	Name       string          `json:"name"`
	Type       EquipmentType   `json:"type"`
	Status     EquipmentStatus `json:"status"`
	WorkerName string          `json:"worker_name"`
	IPAddress  string          `json:"ip_address,omitempty"`

	// Hardware specs
	Model           string `json:"model,omitempty"`
	Manufacturer    string `json:"manufacturer,omitempty"`
	FirmwareVersion string `json:"firmware_version,omitempty"`

	// Performance metrics
	CurrentHashrate float64 `json:"current_hashrate"`
	AverageHashrate float64 `json:"average_hashrate"`
	MaxHashrate     float64 `json:"max_hashrate"`
	Efficiency      float64 `json:"efficiency"`  // Hashrate per watt
	PowerUsage      float64 `json:"power_usage"` // Watts
	Temperature     float64 `json:"temperature"` // Celsius
	FanSpeed        int     `json:"fan_speed"`   // RPM or percentage

	// Network metrics
	Latency        float64   `json:"latency"`         // ms
	ConnectionType string    `json:"connection_type"` // stratum_v1, stratum_v2
	LastSeen       time.Time `json:"last_seen"`
	Uptime         int64     `json:"uptime"` // seconds

	// Mining stats
	SharesAccepted int64   `json:"shares_accepted"`
	SharesRejected int64   `json:"shares_rejected"`
	SharesStale    int64   `json:"shares_stale"`
	BlocksFound    int     `json:"blocks_found"`
	TotalEarnings  float64 `json:"total_earnings"`

	// Payout configuration
	PayoutSplits []PayoutSplit `json:"payout_splits,omitempty"`

	// Timestamps
	RegisteredAt time.Time `json:"registered_at"`
	LastUpdated  time.Time `json:"last_updated"`

	// Error tracking
	LastError   string     `json:"last_error,omitempty"`
	LastErrorAt *time.Time `json:"last_error_at,omitempty"`
	ErrorCount  int        `json:"error_count"`
}

// PayoutSplit defines how earnings from equipment are distributed
type PayoutSplit struct {
	ID            string    `json:"id"`
	EquipmentID   string    `json:"equipment_id"`
	WalletAddress string    `json:"wallet_address"`
	Percentage    float64   `json:"percentage"`      // 0-100
	Label         string    `json:"label,omitempty"` // User-friendly name
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
}

// EquipmentFilter for querying equipment
type EquipmentFilter struct {
	UserID   string          `json:"user_id,omitempty"`
	Status   EquipmentStatus `json:"status,omitempty"`
	Type     EquipmentType   `json:"type,omitempty"`
	IsOnline *bool           `json:"is_online,omitempty"`
}

// EquipmentStats aggregates statistics across equipment
type EquipmentStats struct {
	TotalEquipment      int     `json:"total_equipment"`
	OnlineCount         int     `json:"online_count"`
	OfflineCount        int     `json:"offline_count"`
	MiningCount         int     `json:"mining_count"`
	ErrorCount          int     `json:"error_count"`
	TotalHashrate       float64 `json:"total_hashrate"`
	AverageLatency      float64 `json:"average_latency"`
	TotalPowerUsage     float64 `json:"total_power_usage"`
	TotalEarnings       float64 `json:"total_earnings"`
	TotalSharesAccepted int64   `json:"total_shares_accepted"`
	TotalSharesRejected int64   `json:"total_shares_rejected"`
}

// EquipmentMetricsHistory for time-series data
type EquipmentMetricsHistory struct {
	EquipmentID    string    `json:"equipment_id"`
	Timestamp      time.Time `json:"timestamp"`
	Hashrate       float64   `json:"hashrate"`
	Temperature    float64   `json:"temperature"`
	PowerUsage     float64   `json:"power_usage"`
	Latency        float64   `json:"latency"`
	SharesAccepted int64     `json:"shares_accepted"`
	SharesRejected int64     `json:"shares_rejected"`
}

// === ISP Interfaces ===

// EquipmentReader provides read-only access to equipment data
type EquipmentReader interface {
	GetEquipment(ctx context.Context, equipmentID string) (*Equipment, error)
	GetEquipmentByWorker(ctx context.Context, userID, workerName string) (*Equipment, error)
	ListUserEquipment(ctx context.Context, userID string) ([]Equipment, error)
	ListEquipment(ctx context.Context, filter EquipmentFilter) ([]Equipment, error)
	GetUserEquipmentStats(ctx context.Context, userID string) (*EquipmentStats, error)
	GetPoolEquipmentStats(ctx context.Context) (*EquipmentStats, error)
}

// EquipmentWriter provides write access to equipment data
type EquipmentWriter interface {
	CreateEquipment(ctx context.Context, equipment *Equipment) error
	UpdateEquipment(ctx context.Context, equipment *Equipment) error
	DeleteEquipment(ctx context.Context, equipmentID string) error
	SetEquipmentName(ctx context.Context, equipmentID, name string) error
	SetEquipmentStatus(ctx context.Context, equipmentID string, status EquipmentStatus) error
}

// EquipmentMonitor provides real-time monitoring capabilities
type EquipmentMonitor interface {
	UpdateMetrics(ctx context.Context, equipmentID string, hashrate, temperature, power, latency float64) error
	RecordShare(ctx context.Context, equipmentID string, accepted bool, stale bool) error
	RecordError(ctx context.Context, equipmentID string, errorMsg string) error
	GetMetricsHistory(ctx context.Context, equipmentID string, from, to time.Time) ([]EquipmentMetricsHistory, error)
	GetRealtimeMetrics(ctx context.Context, equipmentID string) (*Equipment, error)
}

// PayoutSplitManager handles multi-wallet payout configuration
type PayoutSplitManager interface {
	GetPayoutSplits(ctx context.Context, equipmentID string) ([]PayoutSplit, error)
	SetPayoutSplits(ctx context.Context, equipmentID string, splits []PayoutSplit) error
	AddPayoutSplit(ctx context.Context, split *PayoutSplit) error
	UpdatePayoutSplit(ctx context.Context, split *PayoutSplit) error
	RemovePayoutSplit(ctx context.Context, splitID string) error
	ValidatePayoutSplits(splits []PayoutSplit) error
}

// EquipmentNotifier sends alerts about equipment status changes
type EquipmentNotifier interface {
	NotifyStatusChange(ctx context.Context, equipment *Equipment, oldStatus, newStatus EquipmentStatus) error
	NotifyError(ctx context.Context, equipment *Equipment, errorMsg string) error
	NotifyOffline(ctx context.Context, equipment *Equipment, offlineDuration time.Duration) error
	NotifyPerformanceDrop(ctx context.Context, equipment *Equipment, expectedHashrate, actualHashrate float64) error
}

// EquipmentService combines all equipment management capabilities
type EquipmentService interface {
	EquipmentReader
	EquipmentWriter
	EquipmentMonitor
	PayoutSplitManager
}

// AdminEquipmentService provides admin-only equipment operations
type AdminEquipmentService interface {
	EquipmentService
	ListAllEquipment(ctx context.Context, page, pageSize int) ([]Equipment, int, error)
	GetGlobalStats(ctx context.Context) (*EquipmentStats, error)
	ForceDisconnect(ctx context.Context, equipmentID string) error
	BanEquipment(ctx context.Context, equipmentID string, reason string) error
}
