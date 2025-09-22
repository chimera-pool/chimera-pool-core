package simulation

import (
	"fmt"
	"sync"
	"time"
)

// SimulationManager coordinates all simulation components
type SimulationManager struct {
	blockchain      BlockchainSimulator
	virtualMiners   VirtualMinerSimulator
	clusters        ClusterSimulator
	isRunning       bool
	mutex           sync.RWMutex
	stopChan        chan struct{}
	stats           *OverallSimulationStats
}

// OverallSimulationStats provides comprehensive simulation statistics
type OverallSimulationStats struct {
	// Blockchain stats
	BlockchainStats NetworkStats
	
	// Virtual miner stats
	VirtualMinerStats *SimulationStats
	
	// Cluster stats
	ClusterStats *OverallClusterStats
	
	// Combined metrics
	TotalHashRate     uint64
	TotalMiners       uint32
	TotalActiveMiners uint32
	OverallUptime     float64
	SimulationTime    time.Duration
	
	// Performance metrics
	SharesPerSecond   float64
	BlocksPerHour     float64
	NetworkEfficiency float64
}

// SimulationConfig defines overall simulation configuration
type SimulationConfig struct {
	BlockchainConfig BlockchainConfig
	MinerConfig      VirtualMinerConfig
	ClusterConfig    ClusterSimulatorConfig
	EnableIntegration bool
	SyncInterval     time.Duration
}

// NewSimulationManager creates a new simulation manager
func NewSimulationManager(config SimulationConfig) (*SimulationManager, error) {
	// Create blockchain simulator
	blockchain, err := NewBlockchainSimulator(config.BlockchainConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create blockchain simulator: %w", err)
	}

	// Create virtual miner simulator
	virtualMiners, err := NewVirtualMinerSimulator(config.MinerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create virtual miner simulator: %w", err)
	}

	// Create cluster simulator
	clusters, err := NewClusterSimulator(config.ClusterConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create cluster simulator: %w", err)
	}

	manager := &SimulationManager{
		blockchain:    blockchain,
		virtualMiners: virtualMiners,
		clusters:      clusters,
		stopChan:      make(chan struct{}),
		stats:         &OverallSimulationStats{},
	}

	return manager, nil
}

// Start begins all simulation components
func (sm *SimulationManager) Start() error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if sm.isRunning {
		return fmt.Errorf("simulation is already running")
	}

	// Start blockchain simulator
	if err := sm.blockchain.Start(); err != nil {
		return fmt.Errorf("failed to start blockchain simulator: %w", err)
	}

	// Start virtual miner simulator
	if err := sm.virtualMiners.Start(); err != nil {
		sm.blockchain.Stop()
		return fmt.Errorf("failed to start virtual miner simulator: %w", err)
	}

	// Start cluster simulator
	if err := sm.clusters.Start(); err != nil {
		sm.blockchain.Stop()
		sm.virtualMiners.Stop()
		return fmt.Errorf("failed to start cluster simulator: %w", err)
	}

	sm.isRunning = true

	// Start coordination and statistics collection
	go sm.coordinateSimulation()
	go sm.collectStatistics()

	return nil
}

// Stop halts all simulation components
func (sm *SimulationManager) Stop() error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if !sm.isRunning {
		return nil
	}

	sm.isRunning = false
	close(sm.stopChan)

	// Stop all simulators
	sm.blockchain.Stop()
	sm.virtualMiners.Stop()
	sm.clusters.Stop()

	return nil
}

// GetOverallStats returns comprehensive simulation statistics
func (sm *SimulationManager) GetOverallStats() *OverallSimulationStats {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	sm.updateOverallStats()
	return sm.stats
}

// GetBlockchainSimulator returns the blockchain simulator
func (sm *SimulationManager) GetBlockchainSimulator() BlockchainSimulator {
	return sm.blockchain
}

// GetVirtualMinerSimulator returns the virtual miner simulator
func (sm *SimulationManager) GetVirtualMinerSimulator() VirtualMinerSimulator {
	return sm.virtualMiners
}

// GetClusterSimulator returns the cluster simulator
func (sm *SimulationManager) GetClusterSimulator() ClusterSimulator {
	return sm.clusters
}

// TriggerStressTest triggers a comprehensive stress test
func (sm *SimulationManager) TriggerStressTest(duration time.Duration) error {
	if !sm.isRunning {
		return fmt.Errorf("simulation is not running")
	}

	// Trigger high load on blockchain
	// This would involve increasing transaction load and mining difficulty

	// Trigger burst mining on virtual miners
	miners := sm.virtualMiners.GetMiners()
	for i, miner := range miners {
		if i%3 == 0 { // Every third miner
			sm.virtualMiners.TriggerBurst(miner.ID, duration/2)
		}
	}

	// Trigger some cluster failures
	clusters := sm.clusters.GetClusters()
	for i, cluster := range clusters {
		if i%4 == 0 { // Every fourth cluster
			sm.clusters.TriggerClusterFailure(cluster.ID, duration/3)
		}
	}

	return nil
}

// TriggerFailureScenario triggers various failure scenarios
func (sm *SimulationManager) TriggerFailureScenario(scenario string) error {
	if !sm.isRunning {
		return fmt.Errorf("simulation is not running")
	}

	switch scenario {
	case "network_partition":
		clusters := sm.clusters.GetClusters()
		if len(clusters) >= 2 {
			clusterIDs := []string{clusters[0].ID, clusters[1].ID}
			return sm.clusters.TriggerNetworkPartition(clusterIDs, time.Minute*5)
		}

	case "mass_miner_dropout":
		miners := sm.virtualMiners.GetMiners()
		for i, miner := range miners {
			if i%2 == 0 { // Every second miner
				sm.virtualMiners.TriggerDrop(miner.ID, time.Minute*2)
			}
		}

	case "coordinator_failure":
		clusters := sm.clusters.GetClusters()
		if len(clusters) > 0 {
			return sm.clusters.TriggerCoordinatorFailure(clusters[0].Coordinator, time.Minute*3)
		}

	case "malicious_attack":
		miners := sm.virtualMiners.GetMiners()
		for _, miner := range miners {
			if miner.IsMalicious {
				sm.virtualMiners.TriggerAttack(miner.ID, "invalid_shares", time.Minute*10)
			}
		}

	default:
		return fmt.Errorf("unknown failure scenario: %s", scenario)
	}

	return nil
}

// ExecutePoolMigration executes a coordinated pool migration
func (sm *SimulationManager) ExecutePoolMigration(sourcePool, targetPool string, strategy string) error {
	if !sm.isRunning {
		return fmt.Errorf("simulation is not running")
	}

	// Find clusters in source pool
	clusters := sm.clusters.GetClusters()
	clusterIDs := make([]string, 0)
	
	for _, cluster := range clusters {
		if cluster.CurrentPool == sourcePool {
			clusterIDs = append(clusterIDs, cluster.ID)
		}
	}

	if len(clusterIDs) == 0 {
		return fmt.Errorf("no clusters found in source pool %s", sourcePool)
	}

	// Create migration plan
	plan := MigrationPlan{
		SourcePool:        sourcePool,
		TargetPool:        targetPool,
		ClusterIDs:        clusterIDs,
		Strategy:          strategy,
		StartTime:         time.Now().Add(time.Second * 5),
		EstimatedDuration: time.Minute * 10,
	}

	return sm.clusters.ExecuteMigration(plan)
}

// Private methods

func (sm *SimulationManager) coordinateSimulation() {
	ticker := time.NewTicker(time.Second * 10)
	defer ticker.Stop()

	for {
		select {
		case <-sm.stopChan:
			return
		case <-ticker.C:
			sm.performCoordination()
		}
	}
}

func (sm *SimulationManager) performCoordination() {
	// Coordinate between different simulation components
	// This could involve:
	// - Synchronizing blockchain state with mining activities
	// - Coordinating cluster behaviors with individual miner behaviors
	// - Adjusting simulation parameters based on performance

	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// Get current blockchain difficulty
	blockchainStats := sm.blockchain.GetNetworkStats()
	
	// Adjust miner behavior based on blockchain state
	if blockchainStats.CurrentDifficulty > 0 {
		// Could adjust miner hash rates or behavior patterns
		// based on blockchain difficulty
	}

	// Coordinate cluster and individual miner activities
	clusterStats := sm.clusters.GetOverallStats()
	minerStats := sm.virtualMiners.GetSimulationStats()
	
	// Balance load between clusters and individual miners
	totalHashRate := clusterStats.TotalHashRate + minerStats.TotalHashRate
	if totalHashRate > 0 {
		// Could implement load balancing logic here
	}
}

func (sm *SimulationManager) collectStatistics() {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	for {
		select {
		case <-sm.stopChan:
			return
		case <-ticker.C:
			sm.updateOverallStats()
		}
	}
}

func (sm *SimulationManager) updateOverallStats() {
	// Collect stats from all components
	blockchainStats := sm.blockchain.GetNetworkStats()
	minerStats := sm.virtualMiners.GetSimulationStats()
	clusterStats := sm.clusters.GetOverallStats()

	// Calculate combined metrics
	totalHashRate := minerStats.TotalHashRate + clusterStats.TotalHashRate
	totalMiners := minerStats.TotalMiners + clusterStats.TotalMiners
	totalActiveMiners := minerStats.ActiveMiners + clusterStats.ActiveMiners

	// Calculate overall uptime
	overallUptime := 0.0
	if totalMiners > 0 {
		minerWeight := float64(minerStats.TotalMiners) / float64(totalMiners)
		clusterWeight := float64(clusterStats.TotalMiners) / float64(totalMiners)
		overallUptime = minerWeight*minerStats.UptimePercentage + clusterWeight*clusterStats.UptimePercentage
	}

	// Calculate performance metrics
	sharesPerSecond := 0.0
	if minerStats.SimulationTime > 0 {
		sharesPerSecond = float64(minerStats.TotalShares) / minerStats.SimulationTime.Seconds()
	}

	blocksPerHour := 0.0
	if blockchainStats.AverageBlockTime > 0 {
		blocksPerHour = float64(time.Hour) / float64(blockchainStats.AverageBlockTime)
	}

	networkEfficiency := 0.0
	if blockchainStats.TotalTransactions > 0 && blockchainStats.BlocksGenerated > 0 {
		networkEfficiency = float64(blockchainStats.TotalTransactions) / float64(blockchainStats.BlocksGenerated)
	}

	// Update overall stats
	sm.stats = &OverallSimulationStats{
		BlockchainStats:   blockchainStats,
		VirtualMinerStats: minerStats,
		ClusterStats:      clusterStats,
		TotalHashRate:     totalHashRate,
		TotalMiners:       totalMiners,
		TotalActiveMiners: totalActiveMiners,
		OverallUptime:     overallUptime,
		SimulationTime:    minerStats.SimulationTime,
		SharesPerSecond:   sharesPerSecond,
		BlocksPerHour:     blocksPerHour,
		NetworkEfficiency: networkEfficiency,
	}
}

// Utility functions for testing and validation

// ValidateSimulationAccuracy validates that the simulation is producing realistic results
func (sm *SimulationManager) ValidateSimulationAccuracy() error {
	stats := sm.GetOverallStats()

	// Validate hash rate consistency
	if stats.TotalHashRate == 0 {
		return fmt.Errorf("total hash rate is zero")
	}

	// Validate uptime is reasonable
	if stats.OverallUptime < 50.0 {
		return fmt.Errorf("overall uptime too low: %.2f%%", stats.OverallUptime)
	}

	// Validate blockchain is producing blocks
	if stats.BlockchainStats.BlocksGenerated == 0 {
		return fmt.Errorf("no blocks generated")
	}

	// Validate miners are submitting shares
	if stats.VirtualMinerStats.TotalShares == 0 && stats.ClusterStats.TotalMiners > 0 {
		return fmt.Errorf("no shares submitted despite active miners")
	}

	return nil
}

// GetPerformanceMetrics returns detailed performance metrics
func (sm *SimulationManager) GetPerformanceMetrics() map[string]interface{} {
	stats := sm.GetOverallStats()
	
	return map[string]interface{}{
		"total_hash_rate":     stats.TotalHashRate,
		"total_miners":        stats.TotalMiners,
		"active_miners":       stats.TotalActiveMiners,
		"overall_uptime":      stats.OverallUptime,
		"shares_per_second":   stats.SharesPerSecond,
		"blocks_per_hour":     stats.BlocksPerHour,
		"network_efficiency":  stats.NetworkEfficiency,
		"simulation_time":     stats.SimulationTime.Seconds(),
		"blockchain_blocks":   stats.BlockchainStats.BlocksGenerated,
		"blockchain_txs":      stats.BlockchainStats.TotalTransactions,
		"miner_burst_events":  stats.VirtualMinerStats.TotalBurstEvents,
		"miner_drop_events":   stats.VirtualMinerStats.TotalDropEvents,
		"cluster_failovers":   stats.ClusterStats.FailoverEvents,
		"cluster_migrations":  stats.ClusterStats.MigrationEvents,
	}
}