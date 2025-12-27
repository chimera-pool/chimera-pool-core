// Package recovery provides network-aware automatic recovery for mining pool services.
// Designed following Interface Segregation Principle (ISP) for maximum flexibility.
// This system addresses the gap where services stay down after internet restoration.
package recovery

import (
	"context"
	"time"
)

// =============================================================================
// NETWORK WATCHDOG INTERFACES (ISP-Compliant)
// =============================================================================

// NetworkChecker detects internet connectivity status
type NetworkChecker interface {
	// IsOnline returns true if internet connectivity is available
	IsOnline(ctx context.Context) bool

	// GetLatency returns the network latency to external hosts
	GetLatency(ctx context.Context) (time.Duration, error)
}

// NetworkStateObserver receives notifications about network state changes
type NetworkStateObserver interface {
	// OnNetworkRestored is called when internet connectivity is restored
	OnNetworkRestored(ctx context.Context)

	// OnNetworkLost is called when internet connectivity is lost
	OnNetworkLost(ctx context.Context)
}

// NetworkWatchdog monitors network connectivity and notifies observers
type NetworkWatchdog interface {
	// Start begins network monitoring
	Start(ctx context.Context) error

	// Stop gracefully stops monitoring
	Stop(ctx context.Context) error

	// RegisterObserver adds an observer to receive network state changes
	RegisterObserver(observer NetworkStateObserver)

	// IsOnline returns current network status
	IsOnline() bool

	// GetStats returns watchdog statistics
	GetStats() *NetworkWatchdogStats
}

// =============================================================================
// SERVICE RECOVERY INTERFACES (ISP-Compliant)
// =============================================================================

// ServiceChecker checks if a service is healthy
type ServiceChecker interface {
	// IsHealthy returns true if the service is responding correctly
	IsHealthy(ctx context.Context) bool

	// GetServiceName returns the service identifier
	GetServiceName() string
}

// ServiceRestarter can restart a service
type ServiceRestarter interface {
	// Restart restarts the service
	Restart(ctx context.Context) error

	// GetServiceName returns the service identifier
	GetServiceName() string
}

// ServiceRecoverable combines checking and restarting capabilities
type ServiceRecoverable interface {
	ServiceChecker
	ServiceRestarter
}

// =============================================================================
// RECOVERY ORCHESTRATOR INTERFACES
// =============================================================================

// RecoveryOrchestrator coordinates the recovery of all services after network restoration
type RecoveryOrchestrator interface {
	// Start begins the orchestrator
	Start(ctx context.Context) error

	// Stop gracefully stops the orchestrator
	Stop(ctx context.Context) error

	// RegisterService adds a service to be managed
	RegisterService(service ServiceRecoverable, priority int) error

	// TriggerRecovery manually triggers recovery sequence
	TriggerRecovery(ctx context.Context) error

	// GetRecoveryStatus returns current recovery status
	GetRecoveryStatus() *RecoveryStatus

	// ResetRestartCounters resets all restart counters (called on network restore)
	ResetRestartCounters()
}

// RecoveryPolicy defines rules for service recovery
type RecoveryPolicy interface {
	// ShouldRestart returns true if service should be restarted
	ShouldRestart(service ServiceRecoverable, stats *ServiceRecoveryStats) bool

	// GetRetryDelay returns delay before next restart attempt
	GetRetryDelay(attemptNumber int) time.Duration

	// GetMaxRetries returns maximum restart attempts
	GetMaxRetries() int
}

// =============================================================================
// ALERT AND NOTIFICATION INTERFACES
// =============================================================================

// RecoveryAlertSender sends alerts about recovery events
type RecoveryAlertSender interface {
	// SendRecoveryStarted notifies that recovery has begun
	SendRecoveryStarted(ctx context.Context, reason string) error

	// SendRecoveryComplete notifies that recovery is complete
	SendRecoveryComplete(ctx context.Context, services []string, duration time.Duration) error

	// SendRecoveryFailed notifies that recovery failed
	SendRecoveryFailed(ctx context.Context, service string, err error) error

	// SendNetworkStateChange notifies about network state changes
	SendNetworkStateChange(ctx context.Context, online bool) error
}

// =============================================================================
// DATA TYPES
// =============================================================================

// NetworkState represents the current network connectivity state
type NetworkState string

const (
	NetworkStateOnline   NetworkState = "online"
	NetworkStateOffline  NetworkState = "offline"
	NetworkStateUnknown  NetworkState = "unknown"
	NetworkStateRestored NetworkState = "restored" // Just came back online
)

// RecoveryState represents the current recovery process state
type RecoveryState string

const (
	RecoveryStateIdle       RecoveryState = "idle"
	RecoveryStateInProgress RecoveryState = "in_progress"
	RecoveryStateComplete   RecoveryState = "complete"
	RecoveryStateFailed     RecoveryState = "failed"
)

// NetworkWatchdogStats contains network monitoring statistics
type NetworkWatchdogStats struct {
	StartTime           time.Time     `json:"start_time"`
	CurrentState        NetworkState  `json:"current_state"`
	LastStateChange     time.Time     `json:"last_state_change"`
	TotalOutages        int64         `json:"total_outages"`
	TotalOutageDuration time.Duration `json:"total_outage_duration"`
	LongestOutage       time.Duration `json:"longest_outage"`
	LastOnline          time.Time     `json:"last_online"`
	LastOffline         time.Time     `json:"last_offline"`
	CheckCount          int64         `json:"check_count"`
	AvgLatency          time.Duration `json:"avg_latency"`
}

// ServiceRecoveryStats tracks recovery statistics for a service
type ServiceRecoveryStats struct {
	ServiceName      string        `json:"service_name"`
	TotalRestarts    int64         `json:"total_restarts"`
	RestartsThisHour int           `json:"restarts_this_hour"`
	LastRestart      time.Time     `json:"last_restart"`
	LastHealthy      time.Time     `json:"last_healthy"`
	ConsecutiveFails int           `json:"consecutive_fails"`
	CurrentlyHealthy bool          `json:"currently_healthy"`
	AvgRestartTime   time.Duration `json:"avg_restart_time"`
	LastRecoveryTime time.Duration `json:"last_recovery_time"`
}

// RecoveryStatus represents the overall recovery status
type RecoveryStatus struct {
	State             RecoveryState                    `json:"state"`
	StartedAt         time.Time                        `json:"started_at,omitempty"`
	CompletedAt       time.Time                        `json:"completed_at,omitempty"`
	Reason            string                           `json:"reason,omitempty"`
	ServicesRecovered []string                         `json:"services_recovered,omitempty"`
	ServicesFailed    []string                         `json:"services_failed,omitempty"`
	ServiceStats      map[string]*ServiceRecoveryStats `json:"service_stats"`
	TotalRecoveries   int64                            `json:"total_recoveries"`
}

// =============================================================================
// CONFIGURATION
// =============================================================================

// NetworkWatchdogConfig configures the network watchdog
type NetworkWatchdogConfig struct {
	// CheckInterval is how often to check network connectivity
	CheckInterval time.Duration `json:"check_interval" yaml:"check_interval"`

	// CheckHosts are external hosts to ping for connectivity checks
	CheckHosts []string `json:"check_hosts" yaml:"check_hosts"`

	// Timeout for each connectivity check
	Timeout time.Duration `json:"timeout" yaml:"timeout"`

	// ConsecutiveFailuresBeforeOffline is how many failures before declaring offline
	ConsecutiveFailuresBeforeOffline int `json:"consecutive_failures_before_offline" yaml:"consecutive_failures_before_offline"`

	// ConsecutiveSuccessesBeforeOnline is how many successes before declaring online
	ConsecutiveSuccessesBeforeOnline int `json:"consecutive_successes_before_online" yaml:"consecutive_successes_before_online"`
}

// DefaultNetworkWatchdogConfig returns sensible defaults
func DefaultNetworkWatchdogConfig() *NetworkWatchdogConfig {
	return &NetworkWatchdogConfig{
		CheckInterval: 5 * time.Second,
		CheckHosts: []string{
			"8.8.8.8:53",        // Google DNS
			"1.1.1.1:53",        // Cloudflare DNS
			"208.67.222.222:53", // OpenDNS
		},
		Timeout:                          3 * time.Second,
		ConsecutiveFailuresBeforeOffline: 3,
		ConsecutiveSuccessesBeforeOnline: 2,
	}
}

// RecoveryOrchestratorConfig configures the recovery orchestrator
type RecoveryOrchestratorConfig struct {
	// MaxRestartsPerHour per service
	MaxRestartsPerHour int `json:"max_restarts_per_hour" yaml:"max_restarts_per_hour"`

	// BaseRetryDelay is the initial delay between restart attempts
	BaseRetryDelay time.Duration `json:"base_retry_delay" yaml:"base_retry_delay"`

	// MaxRetryDelay caps the exponential backoff
	MaxRetryDelay time.Duration `json:"max_retry_delay" yaml:"max_retry_delay"`

	// ServiceStartupTimeout is how long to wait for a service to become healthy
	ServiceStartupTimeout time.Duration `json:"service_startup_timeout" yaml:"service_startup_timeout"`

	// ResetCountersOnNetworkRestore resets restart counters when network comes back
	ResetCountersOnNetworkRestore bool `json:"reset_counters_on_network_restore" yaml:"reset_counters_on_network_restore"`

	// EnableAutoRecovery automatically triggers recovery on network restore
	EnableAutoRecovery bool `json:"enable_auto_recovery" yaml:"enable_auto_recovery"`

	// RecoveryOrder defines service restart order (by priority)
	// Lower priority number = starts first
	RecoveryOrder []string `json:"recovery_order" yaml:"recovery_order"`
}

// DefaultRecoveryOrchestratorConfig returns sensible defaults
func DefaultRecoveryOrchestratorConfig() *RecoveryOrchestratorConfig {
	return &RecoveryOrchestratorConfig{
		MaxRestartsPerHour:            10,
		BaseRetryDelay:                5 * time.Second,
		MaxRetryDelay:                 5 * time.Minute,
		ServiceStartupTimeout:         2 * time.Minute,
		ResetCountersOnNetworkRestore: true, // KEY: Reset counters when network returns!
		EnableAutoRecovery:            true,
		RecoveryOrder: []string{
			"redis",                // 1. Cache/session store
			"postgres",             // 2. Database
			"litecoind",            // 3. Blockchain node
			"chimera-pool-api",     // 4. API server
			"chimera-pool-stratum", // 5. Mining stratum
			"chimera-pool-web",     // 6. Web frontend
			"nginx",                // 7. Reverse proxy
		},
	}
}

// ServiceConfig defines configuration for a recoverable service
type ServiceConfig struct {
	Name          string        `json:"name" yaml:"name"`
	ContainerName string        `json:"container_name" yaml:"container_name"`
	HealthURL     string        `json:"health_url" yaml:"health_url"`
	Priority      int           `json:"priority" yaml:"priority"` // Lower = starts first
	Critical      bool          `json:"critical" yaml:"critical"` // Must be healthy for pool to operate
	DependsOn     []string      `json:"depends_on" yaml:"depends_on"`
	StartupDelay  time.Duration `json:"startup_delay" yaml:"startup_delay"`
}
