// Package health provides multi-chain node health monitoring with automatic recovery.
// Designed following Interface Segregation Principle (ISP) for maximum flexibility
// and testability across different blockchain implementations.
package health

import (
	"context"
	"time"
)

// =============================================================================
// Core Health Check Interfaces (ISP-Compliant)
// =============================================================================

// RPCChecker validates basic RPC connectivity to a blockchain node.
type RPCChecker interface {
	// CheckRPCConnectivity verifies the node is responding to RPC calls.
	// Returns nil if healthy, error with details if not.
	CheckRPCConnectivity(ctx context.Context) error
}

// SyncChecker monitors blockchain synchronization status.
type SyncChecker interface {
	// GetSyncProgress returns sync progress as a float64 (0.0 to 1.0).
	// A value >= 0.9999 typically indicates fully synced.
	GetSyncProgress(ctx context.Context) (float64, error)

	// IsInitialBlockDownload returns true if node is in IBD mode.
	IsInitialBlockDownload(ctx context.Context) (bool, error)
}

// BlockTemplateChecker validates mining block template generation.
type BlockTemplateChecker interface {
	// CheckBlockTemplateGeneration attempts to generate a block template.
	// This is critical for mining pools - if this fails, miners can't work.
	CheckBlockTemplateGeneration(ctx context.Context) error
}

// MempoolChecker monitors transaction mempool health.
type MempoolChecker interface {
	// GetMempoolInfo returns current mempool statistics.
	GetMempoolInfo(ctx context.Context) (*MempoolInfo, error)
}

// ChainDiagnostics provides comprehensive chain-specific diagnostics.
type ChainDiagnostics interface {
	// RunDiagnostics performs a full health check and returns detailed results.
	RunDiagnostics(ctx context.Context) (*NodeDiagnostics, error)

	// GetChainName returns the blockchain identifier (e.g., "litecoin", "blockdag").
	GetChainName() string
}

// NodeHealthChecker combines all health check capabilities.
// Implementations should embed the specific interfaces they support.
type NodeHealthChecker interface {
	RPCChecker
	SyncChecker
	BlockTemplateChecker
	ChainDiagnostics
}

// =============================================================================
// Recovery Action Interfaces
// =============================================================================

// ContainerRestarter can restart Docker containers.
type ContainerRestarter interface {
	// RestartContainer restarts a container by name.
	// Returns nil on success, error on failure.
	RestartContainer(ctx context.Context, containerName string) error

	// GetContainerStatus returns the current status of a container.
	GetContainerStatus(ctx context.Context, containerName string) (ContainerStatus, error)
}

// AlertNotifier sends alerts to operators.
type AlertNotifier interface {
	// SendAlert sends an alert message to configured channels.
	SendAlert(ctx context.Context, alert Alert) error
}

// FailoverManager handles failover to backup nodes.
type FailoverManager interface {
	// TriggerFailover switches to a backup node.
	TriggerFailover(ctx context.Context, fromNode, toNode string) error

	// GetActiveNode returns the currently active node identifier.
	GetActiveNode(ctx context.Context) (string, error)
}

// RecoveryAction combines all recovery capabilities.
type RecoveryAction interface {
	ContainerRestarter
	AlertNotifier
}

// =============================================================================
// Miner Monitoring Interfaces
// =============================================================================

// MinerConnectionMonitor tracks active miner connections.
type MinerConnectionMonitor interface {
	// GetActiveMiners returns the count of currently connected miners.
	GetActiveMiners(ctx context.Context) (int, error)

	// GetMinerStats returns detailed miner statistics.
	GetMinerStats(ctx context.Context) (*MinerStats, error)
}

// ShareRateMonitor tracks share submission rates.
type ShareRateMonitor interface {
	// GetShareRate returns shares per minute for the last interval.
	GetShareRate(ctx context.Context) (float64, error)

	// GetShareStats returns detailed share statistics.
	GetShareStats(ctx context.Context) (*ShareStats, error)
}

// =============================================================================
// Health Monitor Orchestration
// =============================================================================

// HealthRule defines a condition and action for health monitoring.
type HealthRule interface {
	// Evaluate checks if the rule condition is met.
	// Returns true if action should be taken.
	Evaluate(ctx context.Context, diagnostics *NodeDiagnostics) bool

	// GetAction returns the recovery action to take.
	GetAction() RecoveryActionType

	// GetDescription returns a human-readable description of the rule.
	GetDescription() string
}

// HealthMonitorService is the main orchestrator interface.
type HealthMonitorService interface {
	// Start begins health monitoring for all registered nodes.
	Start(ctx context.Context) error

	// Stop gracefully stops health monitoring.
	Stop(ctx context.Context) error

	// RegisterNode adds a node to be monitored.
	RegisterNode(name string, checker NodeHealthChecker, containerName string) error

	// UnregisterNode removes a node from monitoring.
	UnregisterNode(name string) error

	// GetHealthStatus returns the current health status of all nodes.
	GetHealthStatus(ctx context.Context) (map[string]*NodeHealth, error)

	// AddRule adds a health rule to the monitor.
	AddRule(rule HealthRule)

	// GetStats returns monitoring statistics.
	GetStats() *MonitorStats
}

// =============================================================================
// Data Types
// =============================================================================

// HealthStatus represents the overall health state of a node.
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusUnknown   HealthStatus = "unknown"
)

// RecoveryActionType defines types of recovery actions.
type RecoveryActionType string

const (
	RecoveryActionNone     RecoveryActionType = "none"
	RecoveryActionRestart  RecoveryActionType = "restart"
	RecoveryActionAlert    RecoveryActionType = "alert"
	RecoveryActionFailover RecoveryActionType = "failover"
)

// ContainerStatus represents Docker container state.
type ContainerStatus string

const (
	ContainerStatusRunning    ContainerStatus = "running"
	ContainerStatusStopped    ContainerStatus = "stopped"
	ContainerStatusRestarting ContainerStatus = "restarting"
	ContainerStatusUnknown    ContainerStatus = "unknown"
)

// AlertSeverity represents the severity level of an alert.
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityCritical AlertSeverity = "critical"
)

// MempoolInfo contains mempool statistics.
type MempoolInfo struct {
	Size          int     `json:"size"`
	Bytes         int64   `json:"bytes"`
	Usage         int64   `json:"usage"`
	MaxMempool    int64   `json:"max_mempool"`
	MempoolMinFee float64 `json:"mempool_min_fee"`
}

// NodeDiagnostics contains comprehensive node health information.
type NodeDiagnostics struct {
	ChainName            string        `json:"chain_name"`
	Timestamp            time.Time     `json:"timestamp"`
	RPCConnected         bool          `json:"rpc_connected"`
	RPCLatency           time.Duration `json:"rpc_latency"`
	RPCError             string        `json:"rpc_error,omitempty"`
	SyncProgress         float64       `json:"sync_progress"`
	IsIBD                bool          `json:"is_ibd"`
	BlockHeight          int64         `json:"block_height"`
	BlockTemplateOK      bool          `json:"block_template_ok"`
	BlockTemplateError   string        `json:"block_template_error,omitempty"`
	BlockTemplateLatency time.Duration `json:"block_template_latency"`
	Mempool              *MempoolInfo  `json:"mempool,omitempty"`
	ChainSpecificErrors  []string      `json:"chain_specific_errors,omitempty"`
}

// NodeHealth represents the current health state of a monitored node.
type NodeHealth struct {
	Name             string           `json:"name"`
	ContainerName    string           `json:"container_name"`
	Status           HealthStatus     `json:"status"`
	LastCheck        time.Time        `json:"last_check"`
	LastHealthy      time.Time        `json:"last_healthy"`
	ConsecutiveFails int              `json:"consecutive_fails"`
	TotalChecks      int64            `json:"total_checks"`
	TotalFailures    int64            `json:"total_failures"`
	TotalRestarts    int64            `json:"total_restarts"`
	LastRestart      time.Time        `json:"last_restart,omitempty"`
	LastDiagnostics  *NodeDiagnostics `json:"last_diagnostics,omitempty"`
	RestartsThisHour int              `json:"restarts_this_hour"`
	CooldownUntil    time.Time        `json:"cooldown_until,omitempty"`
}

// Alert represents an alert to be sent to operators.
type Alert struct {
	Severity    AlertSeverity `json:"severity"`
	Title       string        `json:"title"`
	Message     string        `json:"message"`
	NodeName    string        `json:"node_name"`
	Timestamp   time.Time     `json:"timestamp"`
	ActionTaken string        `json:"action_taken,omitempty"`
}

// MinerStats contains miner connection statistics.
type MinerStats struct {
	ActiveMiners   int       `json:"active_miners"`
	TotalConnected int       `json:"total_connected"`
	TotalHashrate  float64   `json:"total_hashrate"`
	AvgHashrate    float64   `json:"avg_hashrate"`
	LastConnection time.Time `json:"last_connection"`
	LastDisconnect time.Time `json:"last_disconnect"`
}

// ShareStats contains share submission statistics.
type ShareStats struct {
	SharesPerMinute float64   `json:"shares_per_minute"`
	ValidShares     int64     `json:"valid_shares"`
	InvalidShares   int64     `json:"invalid_shares"`
	StaleShares     int64     `json:"stale_shares"`
	LastShare       time.Time `json:"last_share"`
	AcceptanceRate  float64   `json:"acceptance_rate"`
}

// MonitorStats contains overall monitoring statistics.
type MonitorStats struct {
	StartTime      time.Time              `json:"start_time"`
	TotalChecks    int64                  `json:"total_checks"`
	TotalRestarts  int64                  `json:"total_restarts"`
	TotalAlerts    int64                  `json:"total_alerts"`
	NodesMonitored int                    `json:"nodes_monitored"`
	CheckInterval  time.Duration          `json:"check_interval"`
	NodeStats      map[string]*NodeHealth `json:"node_stats"`
}

// =============================================================================
// Configuration Types
// =============================================================================

// HealthMonitorConfig contains configuration for the health monitor.
type HealthMonitorConfig struct {
	// CheckInterval is how often to check node health.
	CheckInterval time.Duration `json:"check_interval" yaml:"check_interval"`

	// MaxRestartsPerHour limits restarts to prevent loops.
	MaxRestartsPerHour int `json:"max_restarts_per_hour" yaml:"max_restarts_per_hour"`

	// RestartCooldown is the minimum time between restarts.
	RestartCooldown time.Duration `json:"restart_cooldown" yaml:"restart_cooldown"`

	// ConsecutiveFailuresBeforeRestart is how many failures trigger a restart.
	ConsecutiveFailuresBeforeRestart int `json:"consecutive_failures_before_restart" yaml:"consecutive_failures_before_restart"`

	// RPCTimeout is the timeout for RPC calls.
	RPCTimeout time.Duration `json:"rpc_timeout" yaml:"rpc_timeout"`

	// EnableAutoRestart enables automatic container restarts.
	EnableAutoRestart bool `json:"enable_auto_restart" yaml:"enable_auto_restart"`

	// EnableAlerts enables alert notifications.
	EnableAlerts bool `json:"enable_alerts" yaml:"enable_alerts"`

	// AlertWebhookURL is the webhook URL for alerts (Discord/Slack).
	AlertWebhookURL string `json:"alert_webhook_url" yaml:"alert_webhook_url"`
}

// DefaultHealthMonitorConfig returns sensible defaults.
func DefaultHealthMonitorConfig() *HealthMonitorConfig {
	return &HealthMonitorConfig{
		CheckInterval:                    30 * time.Second,
		MaxRestartsPerHour:               10, // Increased from 3 - allows recovery from transient issues
		RestartCooldown:                  60 * time.Second,
		ConsecutiveFailuresBeforeRestart: 3,
		RPCTimeout:                       10 * time.Second,
		EnableAutoRestart:                true,
		EnableAlerts:                     true,
	}
}

// LitecoinNodeConfig contains Litecoin-specific configuration.
type LitecoinNodeConfig struct {
	RPCURL      string `json:"rpc_url" yaml:"rpc_url"`
	RPCUser     string `json:"rpc_user" yaml:"rpc_user"`
	RPCPassword string `json:"rpc_password" yaml:"rpc_password"`
	Container   string `json:"container" yaml:"container"`
}

// BlockDAGNodeConfig contains BlockDAG-specific configuration.
type BlockDAGNodeConfig struct {
	RPCURL        string `json:"rpc_url" yaml:"rpc_url"`
	WalletAddress string `json:"wallet_address" yaml:"wallet_address"`
	Container     string `json:"container" yaml:"container"`
}
