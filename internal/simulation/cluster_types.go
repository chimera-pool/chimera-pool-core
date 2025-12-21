package simulation

import (
	"time"
)

// ClusterSimulatorConfig defines configuration for cluster simulation
type ClusterSimulatorConfig struct {
	ClusterCount           int
	ClustersConfig         []ClusterConfig
	GeographicalSimulation GeographicalConfig
	FailureSimulation      FailureSimulationConfig
	MigrationConfig        MigrationConfig
}

// ClusterConfig defines configuration for a single cluster
type ClusterConfig struct {
	Name               string
	MinerCount         int
	Location           string
	Coordinator        string
	HashRateRange      HashRateRange
	NetworkLatency     time.Duration
	FarmType           string // "ASIC", "GPU", "CPU", "Mixed"
	PowerLimit         uint32 // Watts
	IsBackup           bool
	CurrentPool        string
	FailoverConfig     FailoverConfig
	CoordinationConfig CoordinationConfig
}

// GeographicalConfig defines geographical simulation parameters
type GeographicalConfig struct {
	EnableLatencySimulation bool
	EnableTimezoneEffects   bool
	EnableRegionalFailures  bool
	RegionalLatencyMap      map[string]time.Duration
}

// FailureSimulationConfig defines failure simulation parameters
type FailureSimulationConfig struct {
	EnableClusterFailures     bool
	EnableNetworkPartitions   bool
	EnableCoordinatorFailures bool
	FailureRate               float64
	RecoveryTimeRange         DurationRange
}

// MigrationConfig defines pool migration parameters
type MigrationConfig struct {
	EnableCoordinatedMigration bool
	MigrationStrategies        []MigrationStrategy
	DefaultStrategy            string
}

// FailoverConfig defines failover behavior for a cluster
type FailoverConfig struct {
	BackupClusters []string
	FailureRate    float64
	RecoveryTime   time.Duration
	AutoFailover   bool
}

// CoordinationConfig defines cluster coordination parameters
type CoordinationConfig struct {
	SyncInterval   time.Duration
	LeaderElection bool
	ConsensusType  string // "raft", "pbft", "simple"
}

// MigrationStrategy defines how pool migrations are executed
type MigrationStrategy struct {
	Type           string // "gradual", "immediate", "scheduled"
	Duration       time.Duration
	BatchSize      int
	RollbackOnFail bool
}

// Cluster represents a mining cluster
type Cluster struct {
	ID                 string
	Name               string
	Location           string
	Coordinator        string
	Miners             []*VirtualMiner
	FarmType           string
	PowerLimit         uint32
	CurrentPowerUsage  uint32
	IsActive           bool
	IsLeader           bool
	IsBackup           bool
	IsInFailure        bool
	CurrentPool        string
	LastSyncTime       time.Time
	FailoverConfig     FailoverConfig
	CoordinationConfig CoordinationConfig
	Statistics         *ClusterStatistics
}

// ClusterStatistics tracks cluster performance statistics
type ClusterStatistics struct {
	MinerCount       uint32
	ActiveMiners     uint32
	TotalHashRate    uint64
	AverageHashRate  uint64
	TotalShares      uint64
	ValidShares      uint64
	InvalidShares    uint64
	UptimePercentage float64
	PowerEfficiency  float64 // Hash/Watt
	FailoverEvents   uint64
	SyncEvents       uint64
	MigrationEvents  uint64
	LastFailureTime  time.Time
	LastRecoveryTime time.Time
	IsActive         bool // Whether cluster is currently active
	IsInFailure      bool // Whether cluster is in failure state
}

// MigrationPlan defines a pool migration plan
type MigrationPlan struct {
	ID                string
	SourcePool        string
	TargetPool        string
	ClusterIDs        []string
	Strategy          string
	StartTime         time.Time
	EstimatedDuration time.Duration
	Status            string // "planned", "in_progress", "completed", "failed"
}

// MigrationProgress tracks migration progress
type MigrationProgress struct {
	PlanID                 string
	TotalMiners            uint32
	MigratedMiners         uint32
	FailedMiners           uint32
	ProgressPercent        float64
	EstimatedTimeRemaining time.Duration
	Status                 string
	Errors                 []string
}

// OverallClusterStats provides statistics across all clusters
type OverallClusterStats struct {
	TotalClusters            uint32
	ActiveClusters           uint32
	TotalMiners              uint32
	ActiveMiners             uint32
	TotalHashRate            uint64
	AverageHashRate          uint64
	TotalPowerUsage          uint32
	PowerEfficiency          float64
	UptimePercentage         float64
	FailoverEvents           uint64
	MigrationEvents          uint64
	GeographicalDistribution map[string]uint32
}

// ClusterSimulator interface defines cluster simulation capabilities
type ClusterSimulator interface {
	// Lifecycle management
	Start() error
	Stop() error

	// Cluster management
	GetClusters() []*Cluster
	GetCluster(id string) *Cluster
	AddCluster(config ClusterConfig) (*Cluster, error)
	RemoveCluster(id string) error

	// Failure simulation
	TriggerClusterFailure(clusterID string, duration time.Duration) error
	TriggerNetworkPartition(clusterIDs []string, duration time.Duration) error
	TriggerCoordinatorFailure(coordinatorID string, duration time.Duration) error

	// Migration management
	ExecuteMigration(plan MigrationPlan) error
	GetMigrationProgress(sourcePool, targetPool string) *MigrationProgress
	CancelMigration(planID string) error

	// Statistics and monitoring
	GetOverallStats() *OverallClusterStats
	GetClusterStats(clusterID string) *ClusterStatistics
	GetGeographicalDistribution() map[string]uint32

	// Coordination
	ElectLeader(clusterIDs []string) (string, error)
	SynchronizeClusters(clusterIDs []string) error

	// Configuration
	UpdateClusterConfig(clusterID string, config ClusterConfig) error
	UpdateMinerDistribution(clusterID string, minerCount int) error
}
