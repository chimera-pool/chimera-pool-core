package simulation

import "time"

// ============================================================================
// Interface Segregation Principle (ISP) Compliant Interfaces
// ============================================================================
// These smaller, focused interfaces allow clients to depend only on the
// methods they actually use, following SOLID principles.
// ============================================================================

// --- Lifecycle Interfaces ---

// Startable represents components that can be started
type Startable interface {
	Start() error
}

// Stoppable represents components that can be stopped
type Stoppable interface {
	Stop() error
}

// LifecycleManager combines start and stop capabilities
type LifecycleManager interface {
	Startable
	Stoppable
}

// --- Blockchain Segregated Interfaces ---

// BlockchainReader provides read-only access to blockchain state
type BlockchainReader interface {
	GetNetworkType() string
	GetBlockTime() time.Duration
	GetCurrentDifficulty() uint64
	GetGenesisBlock() *Block
	GetNetworkStats() NetworkStats
}

// BlockchainMiner provides mining capabilities
type BlockchainMiner interface {
	MineNextBlock() (*Block, error)
	MineBlockWithMiner(minerID int) (*Block, error)
}

// BlockchainValidator provides chain validation
type BlockchainValidator interface {
	ValidateChain() bool
}

// --- Virtual Miner Segregated Interfaces ---

// MinerReader provides read-only access to miners
type MinerReader interface {
	GetMiners() []*VirtualMiner
	GetMiner(id string) *VirtualMiner
}

// MinerWriter provides miner management capabilities
type MinerWriter interface {
	AddMiner(config MinerType) (*VirtualMiner, error)
	RemoveMiner(id string) error
}

// MinerStatsProvider provides miner statistics
type MinerStatsProvider interface {
	GetSimulationStats() *SimulationStats
	GetMinerStats(id string) *MinerStatistics
}

// MinerBehaviorController provides behavior control capabilities
type MinerBehaviorController interface {
	TriggerBurst(minerID string, duration time.Duration) error
	TriggerDrop(minerID string, duration time.Duration) error
	TriggerAttack(minerID string, attackType string, duration time.Duration) error
}

// MinerConfigurator provides configuration capabilities
type MinerConfigurator interface {
	UpdateMinerHashRate(minerID string, hashRate uint64) error
	UpdateNetworkConditions(minerID string, conditions NetworkProfile) error
}

// --- Cluster Segregated Interfaces ---

// ClusterReader provides read-only access to clusters
type ClusterReader interface {
	GetClusters() []*Cluster
	GetCluster(id string) *Cluster
}

// ClusterWriter provides cluster management capabilities
type ClusterWriter interface {
	AddCluster(config ClusterConfig) (*Cluster, error)
	RemoveCluster(id string) error
}

// ClusterStatsProvider provides cluster statistics
type ClusterStatsProvider interface {
	GetOverallStats() *OverallClusterStats
	GetClusterStats(clusterID string) *ClusterStatistics
	GetGeographicalDistribution() map[string]uint32
}

// ClusterFailureSimulator provides failure simulation capabilities
type ClusterFailureSimulator interface {
	TriggerClusterFailure(clusterID string, duration time.Duration) error
	TriggerNetworkPartition(clusterIDs []string, duration time.Duration) error
	TriggerCoordinatorFailure(coordinatorID string, duration time.Duration) error
}

// ClusterMigrationManager provides migration capabilities
type ClusterMigrationManager interface {
	ExecuteMigration(plan MigrationPlan) error
	GetMigrationProgress(sourcePool, targetPool string) *MigrationProgress
	CancelMigration(planID string) error
}

// ClusterCoordinator provides coordination capabilities
type ClusterCoordinator interface {
	ElectLeader(clusterIDs []string) (string, error)
	SynchronizeClusters(clusterIDs []string) error
}

// ClusterConfigurator provides configuration capabilities
type ClusterConfigurator interface {
	UpdateClusterConfig(clusterID string, config ClusterConfig) error
	UpdateMinerDistribution(clusterID string, minerCount int) error
}

// --- Composite Interfaces (for convenience) ---

// ReadOnlyBlockchain combines all read-only blockchain capabilities
type ReadOnlyBlockchain interface {
	BlockchainReader
	BlockchainValidator
}

// FullBlockchain combines all blockchain capabilities
type FullBlockchain interface {
	LifecycleManager
	BlockchainReader
	BlockchainMiner
	BlockchainValidator
}

// ReadOnlyMinerSimulator combines all read-only miner capabilities
type ReadOnlyMinerSimulator interface {
	MinerReader
	MinerStatsProvider
}

// FullMinerSimulator combines all miner simulation capabilities
type FullMinerSimulator interface {
	LifecycleManager
	MinerReader
	MinerWriter
	MinerStatsProvider
	MinerBehaviorController
	MinerConfigurator
}

// ReadOnlyClusterSimulator combines all read-only cluster capabilities
type ReadOnlyClusterSimulator interface {
	ClusterReader
	ClusterStatsProvider
}

// FullClusterSimulator combines all cluster simulation capabilities
type FullClusterSimulator interface {
	LifecycleManager
	ClusterReader
	ClusterWriter
	ClusterStatsProvider
	ClusterFailureSimulator
	ClusterMigrationManager
	ClusterCoordinator
	ClusterConfigurator
}

// --- Simulation Manager Interfaces ---

// SimulationStatsProvider provides overall simulation statistics
type SimulationStatsProvider interface {
	GetOverallStats() *OverallSimulationStats
	GetPerformanceMetrics() map[string]interface{}
}

// SimulationComponentProvider provides access to simulation components
type SimulationComponentProvider interface {
	GetBlockchainSimulator() BlockchainSimulator
	GetVirtualMinerSimulator() VirtualMinerSimulator
	GetClusterSimulator() ClusterSimulator
}

// SimulationController provides simulation control capabilities
type SimulationController interface {
	TriggerStressTest(duration time.Duration) error
	TriggerFailureScenario(scenario string) error
	ExecutePoolMigration(sourcePool, targetPool string, strategy string) error
}

// RunningChecker provides running state check
type RunningChecker interface {
	IsRunning() bool
}
