package poolmanager

import (
	"context"
	"time"
)

// PoolStatus represents the current status of the mining pool
type PoolStatus string

const (
	PoolStatusStopped  PoolStatus = "stopped"
	PoolStatusStarting PoolStatus = "starting"
	PoolStatusRunning  PoolStatus = "running"
	PoolStatusStopping PoolStatus = "stopping"
	PoolStatusError    PoolStatus = "error"
)

// PoolManagerConfig contains configuration for the pool manager
type PoolManagerConfig struct {
	StratumAddress string `json:"stratum_address"`
	MaxMiners      int    `json:"max_miners"`
	BlockReward    int64  `json:"block_reward"`
}

// Share represents a mining share submission (simplified version)
type Share struct {
	ID         int64     `json:"id"`
	MinerID    int64     `json:"miner_id"`
	UserID     int64     `json:"user_id"`
	JobID      string    `json:"job_id"`
	Nonce      string    `json:"nonce"`
	Hash       string    `json:"hash"`
	Difficulty float64   `json:"difficulty"`
	IsValid    bool      `json:"is_valid"`
	Timestamp  time.Time `json:"timestamp"`
}

// ShareProcessingResult represents the result of share processing
type ShareProcessingResult struct {
	Success        bool   `json:"success"`
	ProcessedShare *Share `json:"processed_share"`
	Error          string `json:"error"`
}

// ShareStatistics represents share processing statistics
type ShareStatistics struct {
	TotalShares     int64     `json:"total_shares"`
	ValidShares     int64     `json:"valid_shares"`
	InvalidShares   int64     `json:"invalid_shares"`
	TotalDifficulty float64   `json:"total_difficulty"`
	LastUpdated     time.Time `json:"last_updated"`
}

// User represents a user account
type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// JWTClaims represents JWT token claims
type JWTClaims struct {
	UserID    int64     `json:"user_id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// PoolManagerStatus represents the overall status of the pool manager
type PoolManagerStatus struct {
	Status            PoolStatus        `json:"status"`
	ConnectedMiners   int               `json:"connected_miners"`
	TotalShares       int64             `json:"total_shares"`
	ValidShares       int64             `json:"valid_shares"`
	ComponentHealth   ComponentHealth   `json:"component_health"`
	LastUpdated       time.Time         `json:"last_updated"`
}

// ComponentHealth represents the health status of all components
type ComponentHealth struct {
	StratumServer   HealthStatus `json:"stratum_server"`
	ShareProcessor  HealthStatus `json:"share_processor"`
	AuthService     HealthStatus `json:"auth_service"`
	PayoutService   HealthStatus `json:"payout_service"`
}

// HealthStatus represents the health status of a component
type HealthStatus struct {
	Status      string    `json:"status"`
	LastCheck   time.Time `json:"last_check"`
	ErrorCount  int       `json:"error_count"`
	LastError   string    `json:"last_error,omitempty"`
}

// PoolStatistics represents comprehensive pool statistics
type PoolStatistics struct {
	TotalMiners       int               `json:"total_miners"`
	ActiveMiners      int               `json:"active_miners"`
	TotalHashrate     float64           `json:"total_hashrate"`
	SharesPerSecond   float64           `json:"shares_per_second"`
	BlocksFound       int64             `json:"blocks_found"`
	LastBlockTime     time.Time         `json:"last_block_time"`
	ShareStatistics   ShareStatistics   `json:"share_statistics"`
	ComponentHealth   ComponentHealth   `json:"component_health"`
}

// Interfaces for dependency injection

// StratumServerInterface defines the interface for Stratum server operations
type StratumServerInterface interface {
	Start() error
	Stop() error
	GetConnectionCount() int
	GetAddress() string
}

// ShareProcessorInterface defines the interface for share processing operations
type ShareProcessorInterface interface {
	ProcessShare(share *Share) ShareProcessingResult
	GetStatistics() ShareStatistics
}

// AuthServiceInterface defines the interface for authentication operations
type AuthServiceInterface interface {
	ValidateJWT(token string) (*JWTClaims, error)
	LoginUser(username, password string) (*User, string, error)
}

// PayoutServiceInterface defines the interface for payout operations
type PayoutServiceInterface interface {
	ProcessBlockPayout(ctx context.Context, blockID int64) error
	CalculateEstimatedPayout(ctx context.Context, userID int64, estimatedBlockReward int64) (int64, error)
}

// New types for comprehensive component coordination

// MinerConnection represents a connected miner
type MinerConnection struct {
	ID       string `json:"id"`
	Address  string `json:"address"`
	Username string `json:"username"`
}

// JobTemplate represents a mining job template
type JobTemplate struct {
	ID         string  `json:"id"`
	PrevHash   string  `json:"prev_hash"`
	Difficulty float64 `json:"difficulty"`
}

// MiningWorkflow represents a complete mining workflow
type MiningWorkflow struct {
	MinerConnection *MinerConnection `json:"miner_connection"`
	AuthToken       string           `json:"auth_token"`
	JobTemplate     *JobTemplate     `json:"job_template"`
}

// MiningWorkflowResult represents the result of a mining workflow
type MiningWorkflowResult struct {
	Success       bool              `json:"success"`
	SharesProcessed int             `json:"shares_processed"`
	BlocksFound   int               `json:"blocks_found"`
	PayoutsIssued int               `json:"payouts_issued"`
	Errors        []string          `json:"errors"`
	ComponentHealth *ComponentHealth `json:"component_health"`
}

// BlockDiscovery represents a discovered block
type BlockDiscovery struct {
	BlockHash   string    `json:"block_hash"`
	BlockHeight int64     `json:"block_height"`
	Difficulty  float64   `json:"difficulty"`
	Reward      int64     `json:"reward"`
	FoundBy     string    `json:"found_by"`
	Timestamp   time.Time `json:"timestamp"`
}

// ComponentHealthReport represents a comprehensive health report
type ComponentHealthReport struct {
	OverallHealth   string                     `json:"overall_health"`
	ComponentHealth ComponentHealth            `json:"component_health"`
	Recommendations []string                   `json:"recommendations"`
	Metrics         map[string]interface{}     `json:"metrics"`
	Timestamp       time.Time                  `json:"timestamp"`
}

// Advanced coordination types for enhanced TDD testing

// AdvancedWorkflowConfig represents configuration for advanced workflow coordination
type AdvancedWorkflowConfig struct {
	EnableDetailedMetrics         bool `json:"enable_detailed_metrics"`
	EnablePerformanceOptimization bool `json:"enable_performance_optimization"`
	EnableAdvancedErrorRecovery   bool `json:"enable_advanced_error_recovery"`
}

// AdvancedWorkflowMetrics represents detailed metrics from advanced workflow coordination
type AdvancedWorkflowMetrics struct {
	ProcessingEfficiency    float64           `json:"processing_efficiency"`
	ComponentResponseTimes  map[string]time.Duration `json:"component_response_times"`
	ErrorRecoveryCount      int               `json:"error_recovery_count"`
	OptimizationApplied     []string          `json:"optimization_applied"`
	DetailedMetrics         map[string]interface{} `json:"detailed_metrics"`
}

// ComponentFailureScenario represents a component failure scenario for testing
type ComponentFailureScenario struct {
	FailedComponent  string `json:"failed_component"`
	FailureType      string `json:"failure_type"`
	RecoveryStrategy string `json:"recovery_strategy"`
}

// ErrorRecoveryResult represents the result of error recovery coordination
type ErrorRecoveryResult struct {
	RecoverySuccessful bool          `json:"recovery_successful"`
	RecoveryTime       time.Duration `json:"recovery_time"`
	ActionsPerformed   []string      `json:"actions_performed"`
	ComponentsAffected []string      `json:"components_affected"`
}

// PerformanceOptimizationConfig represents configuration for performance optimization
type PerformanceOptimizationConfig struct {
	TargetLatency       time.Duration `json:"target_latency"`
	TargetThroughput    int           `json:"target_throughput"`
	EnableCaching       bool          `json:"enable_caching"`
	EnableLoadBalancing bool          `json:"enable_load_balancing"`
}

// PerformanceOptimizationResult represents the result of performance optimization
type PerformanceOptimizationResult struct {
	AchievedLatency     time.Duration `json:"achieved_latency"`
	AchievedThroughput  int           `json:"achieved_throughput"`
	OptimizationsApplied []string     `json:"optimizations_applied"`
	PerformanceGain     float64       `json:"performance_gain"`
}

// RealTimeMetricsConfig represents configuration for real-time metrics
type RealTimeMetricsConfig struct {
	UpdateInterval   time.Duration `json:"update_interval"`
	EnablePredictive bool          `json:"enable_predictive"`
	EnableAlerting   bool          `json:"enable_alerting"`
	MetricsRetention time.Duration `json:"metrics_retention"`
}

// RealTimeMetricsData represents real-time metrics data
type RealTimeMetricsData struct {
	ActiveMetrics     []string                   `json:"active_metrics"`
	MetricsValues     map[string]interface{}     `json:"metrics_values"`
	PredictiveData    map[string]interface{}     `json:"predictive_data"`
	AlertsTriggered   []string                   `json:"alerts_triggered"`
	LastUpdated       time.Time                  `json:"last_updated"`
}

// LoadBalancingConfig represents configuration for load balancing
type LoadBalancingConfig struct {
	Strategy            string        `json:"strategy"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	MaxLoadPerInstance  float64       `json:"max_load_per_instance"`
	EnableAutoScaling   bool          `json:"enable_auto_scaling"`
}

// LoadBalancingResult represents the result of load balancing coordination
type LoadBalancingResult struct {
	BalancingActive    bool                       `json:"balancing_active"`
	ActiveInstances    int                        `json:"active_instances"`
	LoadDistribution   map[string]float64         `json:"load_distribution"`
	HealthStatus       map[string]string          `json:"health_status"`
	ScalingActions     []string                   `json:"scaling_actions"`
}