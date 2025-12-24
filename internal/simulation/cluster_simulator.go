package simulation

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// clusterSimulator implements the ClusterSimulator interface
type clusterSimulator struct {
	config            ClusterSimulatorConfig
	clusters          map[string]*Cluster
	migrations        map[string]*MigrationPlan
	migrationProgress map[string]*MigrationProgress
	isRunning         bool
	stopChan          chan struct{}
	mutex             sync.RWMutex
	overallStats      *OverallClusterStats
	startTime         time.Time
}

// NewClusterSimulator creates a new cluster simulator
func NewClusterSimulator(config ClusterSimulatorConfig) (ClusterSimulator, error) {
	simulator := &clusterSimulator{
		config:            config,
		clusters:          make(map[string]*Cluster),
		migrations:        make(map[string]*MigrationPlan),
		migrationProgress: make(map[string]*MigrationProgress),
		stopChan:          make(chan struct{}),
		overallStats:      &OverallClusterStats{},
	}

	// Generate clusters based on configuration
	err := simulator.generateClusters()
	if err != nil {
		return nil, fmt.Errorf("failed to generate clusters: %w", err)
	}

	return simulator, nil
}

// Start begins the cluster simulation
func (cs *clusterSimulator) Start() error {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	if cs.isRunning {
		return fmt.Errorf("simulator is already running")
	}

	cs.isRunning = true
	cs.startTime = time.Now()

	// Activate all clusters
	for _, cluster := range cs.clusters {
		cluster.IsActive = true
		cluster.LastSyncTime = time.Now()

		// Start all miners in the cluster
		for _, miner := range cluster.Miners {
			miner.IsActive = true
			miner.CurrentState.LastSeen = time.Now()
		}
	}

	// Start simulation goroutines
	go cs.simulateClusterBehaviors()
	go cs.simulateFailures()
	go cs.simulateCoordination()
	go cs.updateStatistics()

	// Elect initial leaders if configured
	cs.performLeaderElection()

	return nil
}

// Stop halts the cluster simulation
func (cs *clusterSimulator) Stop() error {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	if !cs.isRunning {
		return nil
	}

	cs.isRunning = false
	close(cs.stopChan)

	// Deactivate all clusters
	for _, cluster := range cs.clusters {
		cluster.IsActive = false
		for _, miner := range cluster.Miners {
			miner.IsActive = false
		}
	}

	return nil
}

// GetClusters returns all clusters
func (cs *clusterSimulator) GetClusters() []*Cluster {
	cs.mutex.RLock()
	defer cs.mutex.RUnlock()

	// Return deep COPIES to prevent race conditions
	clusters := make([]*Cluster, 0, len(cs.clusters))
	for _, cluster := range cs.clusters {
		clusters = append(clusters, cs.copyCluster(cluster))
	}
	return clusters
}

// GetCluster returns a specific cluster by ID
func (cs *clusterSimulator) GetCluster(id string) *Cluster {
	cs.mutex.RLock()
	defer cs.mutex.RUnlock()

	cluster := cs.clusters[id]
	if cluster == nil {
		return nil
	}
	// Return a deep COPY to prevent race conditions
	return cs.copyCluster(cluster)
}

// copyCluster creates a deep copy of a Cluster
func (cs *clusterSimulator) copyCluster(c *Cluster) *Cluster {
	if c == nil {
		return nil
	}

	copy := &Cluster{
		ID:                 c.ID,
		Name:               c.Name,
		Location:           c.Location,
		Coordinator:        c.Coordinator,
		FarmType:           c.FarmType,
		PowerLimit:         c.PowerLimit,
		CurrentPowerUsage:  c.CurrentPowerUsage,
		IsActive:           c.IsActive,
		IsLeader:           c.IsLeader,
		IsBackup:           c.IsBackup,
		IsInFailure:        c.IsInFailure,
		CurrentPool:        c.CurrentPool,
		LastSyncTime:       c.LastSyncTime,
		FailoverConfig:     c.FailoverConfig,
		CoordinationConfig: c.CoordinationConfig,
	}

	// Deep copy miners slice (just copy IDs and basic info, not full miner data)
	if c.Miners != nil {
		copy.Miners = make([]*VirtualMiner, len(c.Miners))
		for i, m := range c.Miners {
			if m != nil {
				copy.Miners[i] = &VirtualMiner{
					ID:          m.ID,
					Type:        m.Type,
					HashRate:    m.HashRate,
					IsActive:    m.IsActive,
					IsMalicious: m.IsMalicious,
					Location:    m.Location,
				}
				if m.Statistics != nil {
					copy.Miners[i].Statistics = &MinerStatistics{
						TotalShares:   m.Statistics.TotalShares,
						ValidShares:   m.Statistics.ValidShares,
						InvalidShares: m.Statistics.InvalidShares,
					}
				}
			}
		}
	}

	if c.Statistics != nil {
		copy.Statistics = &ClusterStatistics{
			MinerCount:       c.Statistics.MinerCount,
			ActiveMiners:     c.Statistics.ActiveMiners,
			TotalHashRate:    c.Statistics.TotalHashRate,
			AverageHashRate:  c.Statistics.AverageHashRate,
			TotalShares:      c.Statistics.TotalShares,
			ValidShares:      c.Statistics.ValidShares,
			InvalidShares:    c.Statistics.InvalidShares,
			UptimePercentage: c.Statistics.UptimePercentage,
			PowerEfficiency:  c.Statistics.PowerEfficiency,
			FailoverEvents:   c.Statistics.FailoverEvents,
			SyncEvents:       c.Statistics.SyncEvents,
			MigrationEvents:  c.Statistics.MigrationEvents,
			LastFailureTime:  c.Statistics.LastFailureTime,
			LastRecoveryTime: c.Statistics.LastRecoveryTime,
			IsActive:         c.Statistics.IsActive,
			IsInFailure:      c.Statistics.IsInFailure,
		}
	}

	return copy
}

// AddCluster adds a new cluster to the simulation
func (cs *clusterSimulator) AddCluster(config ClusterConfig) (*Cluster, error) {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	cluster, err := cs.createCluster(config)
	if err != nil {
		return nil, err
	}

	cs.clusters[cluster.ID] = cluster

	if cs.isRunning {
		cluster.IsActive = true
		cluster.LastSyncTime = time.Now()
		for _, miner := range cluster.Miners {
			miner.IsActive = true
			miner.CurrentState.LastSeen = time.Now()
		}
	}

	return cluster, nil
}

// RemoveCluster removes a cluster from the simulation
func (cs *clusterSimulator) RemoveCluster(id string) error {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	if _, exists := cs.clusters[id]; !exists {
		return fmt.Errorf("cluster %s not found", id)
	}

	delete(cs.clusters, id)
	return nil
}

// TriggerClusterFailure simulates a cluster failure
func (cs *clusterSimulator) TriggerClusterFailure(clusterID string, duration time.Duration) error {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	cluster := cs.clusters[clusterID]
	if cluster == nil {
		return fmt.Errorf("cluster %s not found", clusterID)
	}

	cluster.IsInFailure = true
	cluster.IsActive = false
	cluster.Statistics.FailoverEvents++
	cluster.Statistics.LastFailureTime = time.Now()

	// Deactivate all miners in the cluster
	for _, miner := range cluster.Miners {
		miner.IsActive = false
		miner.CurrentState.IsDisconnected = true
	}

	// Trigger failover if configured
	if cluster.FailoverConfig.AutoFailover && len(cluster.FailoverConfig.BackupClusters) > 0 {
		go cs.executeFailover(clusterID, duration)
	} else {
		// Schedule recovery
		go cs.scheduleRecovery(clusterID, duration)
	}

	return nil
}

// TriggerNetworkPartition simulates network partition between clusters
func (cs *clusterSimulator) TriggerNetworkPartition(clusterIDs []string, duration time.Duration) error {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	// Mark clusters as partitioned
	for _, clusterID := range clusterIDs {
		cluster := cs.clusters[clusterID]
		if cluster != nil {
			// Increase network latency significantly
			for _, miner := range cluster.Miners {
				miner.NetworkProfile.Latency *= 10
				miner.NetworkProfile.PacketLoss = 0.5 // 50% packet loss
			}
		}
	}

	// Schedule partition recovery
	go func() {
		time.Sleep(duration)
		cs.mutex.Lock()
		defer cs.mutex.Unlock()

		for _, clusterID := range clusterIDs {
			cluster := cs.clusters[clusterID]
			if cluster != nil {
				// Restore normal network conditions
				for _, miner := range cluster.Miners {
					miner.NetworkProfile.Latency /= 10
					miner.NetworkProfile.PacketLoss = 0.01 // Restore to 1%
				}
			}
		}
	}()

	return nil
}

// TriggerCoordinatorFailure simulates coordinator failure
func (cs *clusterSimulator) TriggerCoordinatorFailure(coordinatorID string, duration time.Duration) error {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	// Find clusters with this coordinator
	affectedClusters := make([]*Cluster, 0)
	for _, cluster := range cs.clusters {
		if cluster.Coordinator == coordinatorID {
			affectedClusters = append(affectedClusters, cluster)
		}
	}

	if len(affectedClusters) == 0 {
		return fmt.Errorf("no clusters found with coordinator %s", coordinatorID)
	}

	// Mark clusters as having coordinator failure
	for _, cluster := range affectedClusters {
		cluster.IsLeader = false
		cluster.Statistics.FailoverEvents++
	}

	// Schedule coordinator recovery and re-election
	go func() {
		time.Sleep(duration)
		cs.mutex.Lock()
		defer cs.mutex.Unlock()

		// Trigger leader re-election
		clusterIDs := make([]string, len(affectedClusters))
		for i, cluster := range affectedClusters {
			clusterIDs[i] = cluster.ID
		}
		cs.performLeaderElectionForClusters(clusterIDs)
	}()

	return nil
}

// ExecuteMigration executes a pool migration plan
func (cs *clusterSimulator) ExecuteMigration(plan MigrationPlan) error {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	// Validate plan
	for _, clusterID := range plan.ClusterIDs {
		if cs.clusters[clusterID] == nil {
			return fmt.Errorf("cluster %s not found", clusterID)
		}
	}

	// Generate plan ID if not provided
	if plan.ID == "" {
		plan.ID = fmt.Sprintf("migration_%d", time.Now().UnixNano())
	}

	plan.Status = "planned"
	cs.migrations[plan.ID] = &plan

	// Initialize progress tracking
	totalMiners := uint32(0)
	for _, clusterID := range plan.ClusterIDs {
		cluster := cs.clusters[clusterID]
		totalMiners += uint32(len(cluster.Miners))
	}

	progress := &MigrationProgress{
		PlanID:      plan.ID,
		TotalMiners: totalMiners,
		Status:      "in_progress",
	}
	cs.migrationProgress[plan.ID] = progress

	// Execute migration based on strategy
	go cs.executeMigrationPlan(&plan)

	return nil
}

// GetMigrationProgress returns migration progress
func (cs *clusterSimulator) GetMigrationProgress(sourcePool, targetPool string) *MigrationProgress {
	cs.mutex.RLock()
	defer cs.mutex.RUnlock()

	// Find migration by source and target pools
	for _, progress := range cs.migrationProgress {
		plan := cs.migrations[progress.PlanID]
		if plan != nil && plan.SourcePool == sourcePool && plan.TargetPool == targetPool {
			// Return a COPY to prevent race conditions
			errorsCopy := make([]string, len(progress.Errors))
			copy(errorsCopy, progress.Errors)
			return &MigrationProgress{
				PlanID:                 progress.PlanID,
				TotalMiners:            progress.TotalMiners,
				MigratedMiners:         progress.MigratedMiners,
				FailedMiners:           progress.FailedMiners,
				ProgressPercent:        progress.ProgressPercent,
				EstimatedTimeRemaining: progress.EstimatedTimeRemaining,
				Status:                 progress.Status,
				Errors:                 errorsCopy,
			}
		}
	}

	return nil
}

// CancelMigration cancels an ongoing migration
func (cs *clusterSimulator) CancelMigration(planID string) error {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	plan := cs.migrations[planID]
	if plan == nil {
		return fmt.Errorf("migration plan %s not found", planID)
	}

	plan.Status = "cancelled"

	progress := cs.migrationProgress[planID]
	if progress != nil {
		progress.Status = "cancelled"
	}

	return nil
}

// GetOverallStats returns overall cluster statistics
func (cs *clusterSimulator) GetOverallStats() *OverallClusterStats {
	// Use write lock because calculateOverallStats modifies cs.overallStats
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	cs.calculateOverallStats()

	// Deep copy the map to prevent race conditions
	geoDist := make(map[string]uint32, len(cs.overallStats.GeographicalDistribution))
	for k, v := range cs.overallStats.GeographicalDistribution {
		geoDist[k] = v
	}

	// Return a COPY to prevent race conditions
	return &OverallClusterStats{
		TotalClusters:            cs.overallStats.TotalClusters,
		ActiveClusters:           cs.overallStats.ActiveClusters,
		TotalMiners:              cs.overallStats.TotalMiners,
		ActiveMiners:             cs.overallStats.ActiveMiners,
		TotalHashRate:            cs.overallStats.TotalHashRate,
		AverageHashRate:          cs.overallStats.AverageHashRate,
		TotalPowerUsage:          cs.overallStats.TotalPowerUsage,
		PowerEfficiency:          cs.overallStats.PowerEfficiency,
		UptimePercentage:         cs.overallStats.UptimePercentage,
		FailoverEvents:           cs.overallStats.FailoverEvents,
		MigrationEvents:          cs.overallStats.MigrationEvents,
		GeographicalDistribution: geoDist,
	}
}

// GetClusterStats returns statistics for a specific cluster
func (cs *clusterSimulator) GetClusterStats(clusterID string) *ClusterStatistics {
	// Use write lock because calculateClusterStats modifies cluster.Statistics
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	cluster := cs.clusters[clusterID]
	if cluster == nil {
		return nil
	}

	cs.calculateClusterStats(cluster)

	// Return a COPY to prevent race conditions
	return &ClusterStatistics{
		MinerCount:       cluster.Statistics.MinerCount,
		ActiveMiners:     cluster.Statistics.ActiveMiners,
		TotalHashRate:    cluster.Statistics.TotalHashRate,
		AverageHashRate:  cluster.Statistics.AverageHashRate,
		TotalShares:      cluster.Statistics.TotalShares,
		ValidShares:      cluster.Statistics.ValidShares,
		InvalidShares:    cluster.Statistics.InvalidShares,
		UptimePercentage: cluster.Statistics.UptimePercentage,
		PowerEfficiency:  cluster.Statistics.PowerEfficiency,
		FailoverEvents:   cluster.Statistics.FailoverEvents,
		MigrationEvents:  cluster.Statistics.MigrationEvents,
		SyncEvents:       cluster.Statistics.SyncEvents,
	}
}

// GetGeographicalDistribution returns geographical distribution of clusters
func (cs *clusterSimulator) GetGeographicalDistribution() map[string]uint32 {
	cs.mutex.RLock()
	defer cs.mutex.RUnlock()

	distribution := make(map[string]uint32)
	for _, cluster := range cs.clusters {
		distribution[cluster.Location]++
	}

	return distribution
}

// ElectLeader elects a leader among specified clusters
func (cs *clusterSimulator) ElectLeader(clusterIDs []string) (string, error) {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	return cs.performLeaderElectionForClusters(clusterIDs), nil
}

// SynchronizeClusters synchronizes specified clusters
func (cs *clusterSimulator) SynchronizeClusters(clusterIDs []string) error {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	syncTime := time.Now()
	for _, clusterID := range clusterIDs {
		cluster := cs.clusters[clusterID]
		if cluster != nil {
			cluster.LastSyncTime = syncTime
			cluster.Statistics.SyncEvents++
		}
	}

	return nil
}

// UpdateClusterConfig updates cluster configuration
func (cs *clusterSimulator) UpdateClusterConfig(clusterID string, config ClusterConfig) error {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	cluster := cs.clusters[clusterID]
	if cluster == nil {
		return fmt.Errorf("cluster %s not found", clusterID)
	}

	// Update cluster configuration
	cluster.Name = config.Name
	cluster.Location = config.Location
	cluster.Coordinator = config.Coordinator
	cluster.FarmType = config.FarmType
	cluster.PowerLimit = config.PowerLimit
	cluster.FailoverConfig = config.FailoverConfig
	cluster.CoordinationConfig = config.CoordinationConfig

	return nil
}

// UpdateMinerDistribution updates the number of miners in a cluster
func (cs *clusterSimulator) UpdateMinerDistribution(clusterID string, minerCount int) error {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	cluster := cs.clusters[clusterID]
	if cluster == nil {
		return fmt.Errorf("cluster %s not found", clusterID)
	}

	currentCount := len(cluster.Miners)

	if minerCount > currentCount {
		// Add miners
		for i := currentCount; i < minerCount; i++ {
			miner := cs.createMinerForCluster(cluster)
			cluster.Miners = append(cluster.Miners, miner)
		}
	} else if minerCount < currentCount {
		// Remove miners
		cluster.Miners = cluster.Miners[:minerCount]
	}

	return nil
}

// Private helper methods

func (cs *clusterSimulator) generateClusters() error {
	for i, config := range cs.config.ClustersConfig {
		cluster, err := cs.createCluster(config)
		if err != nil {
			return fmt.Errorf("failed to create cluster %d: %w", i, err)
		}
		cs.clusters[cluster.ID] = cluster
	}
	return nil
}

func (cs *clusterSimulator) createCluster(config ClusterConfig) (*Cluster, error) {
	id := fmt.Sprintf("cluster_%s_%d", config.Name, time.Now().UnixNano())

	cluster := &Cluster{
		ID:                 id,
		Name:               config.Name,
		Location:           config.Location,
		Coordinator:        config.Coordinator,
		FarmType:           config.FarmType,
		PowerLimit:         config.PowerLimit,
		IsBackup:           config.IsBackup,
		CurrentPool:        config.CurrentPool,
		FailoverConfig:     config.FailoverConfig,
		CoordinationConfig: config.CoordinationConfig,
		Statistics:         &ClusterStatistics{},
	}

	// Generate miners for the cluster
	miners := make([]*VirtualMiner, 0, config.MinerCount)
	for i := 0; i < config.MinerCount; i++ {
		miner := cs.createMinerForCluster(cluster)
		miners = append(miners, miner)
	}
	cluster.Miners = miners

	return cluster, nil
}

func (cs *clusterSimulator) createMinerForCluster(cluster *Cluster) *VirtualMiner {
	// Find cluster config to get hash rate range
	var hashRateRange HashRateRange
	for _, config := range cs.config.ClustersConfig {
		if config.Name == cluster.Name {
			hashRateRange = config.HashRateRange
			break
		}
	}

	if hashRateRange.Min == 0 {
		hashRateRange = HashRateRange{Min: 1000000, Max: 10000000}
	}

	// Calculate hash rate
	baseHashRate := hashRateRange.Min +
		uint64(rand.Float64()*float64(hashRateRange.Max-hashRateRange.Min))

	id := fmt.Sprintf("miner_%s_%d_%d", cluster.ID, time.Now().UnixNano(), rand.Intn(10000))

	miner := &VirtualMiner{
		ID:       id,
		Type:     cluster.FarmType,
		HashRate: baseHashRate,
		Location: cluster.Location,
		PerformanceProfile: &PerformanceProfile{
			PowerConsumption: cs.calculatePowerConsumption(cluster.FarmType, baseHashRate),
			EfficiencyRating: cs.calculateEfficiency(cluster.FarmType),
			FailureRate:      0.01,
			Temperature:      20.0 + rand.Float64()*40.0,
			FanSpeed:         uint32(1000 + rand.Intn(2000)),
		},
		NetworkProfile: cs.createNetworkProfileForCluster(cluster),
		CurrentState: &MinerState{
			LastSeen: time.Now(),
		},
		Statistics: &MinerStatistics{
			LastShareTime: time.Now(),
		},
	}

	return miner
}

func (cs *clusterSimulator) createNetworkProfileForCluster(cluster *Cluster) *NetworkProfile {
	// Find cluster config to get network latency
	var networkLatency time.Duration
	for _, config := range cs.config.ClustersConfig {
		if config.Name == cluster.Name {
			networkLatency = config.NetworkLatency
			break
		}
	}

	if networkLatency == 0 {
		networkLatency = time.Millisecond * 50
	}

	// Add some jitter
	jitter := time.Duration(rand.Float64() * float64(networkLatency) * 0.2)
	actualLatency := networkLatency + jitter

	return &NetworkProfile{
		Quality:    "good",
		Latency:    actualLatency,
		PacketLoss: 0.01,
		Jitter:     time.Millisecond * time.Duration(5+rand.Intn(15)),
		Bandwidth:  uint64(1000000 + rand.Intn(9000000)), // 1-10 Mbps
	}
}

func (cs *clusterSimulator) calculatePowerConsumption(farmType string, hashRate uint64) uint32 {
	switch farmType {
	case "ASIC":
		return uint32(hashRate / 1000000 * 100) // ~100W per MH/s for ASIC
	case "GPU":
		return uint32(hashRate / 1000000 * 300) // ~300W per MH/s for GPU
	case "CPU":
		return uint32(hashRate / 1000000 * 500) // ~500W per MH/s for CPU
	default:
		return uint32(hashRate / 1000000 * 200) // Default
	}
}

func (cs *clusterSimulator) calculateEfficiency(farmType string) float64 {
	switch farmType {
	case "ASIC":
		return 0.95
	case "GPU":
		return 0.85
	case "CPU":
		return 0.70
	default:
		return 0.80
	}
}

func (cs *clusterSimulator) simulateClusterBehaviors() {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	for {
		select {
		case <-cs.stopChan:
			return
		case <-ticker.C:
			cs.processClusterBehaviors()
		}
	}
}

func (cs *clusterSimulator) processClusterBehaviors() {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	for _, cluster := range cs.clusters {
		if !cluster.IsActive {
			continue
		}

		// Update power usage
		totalPower := uint32(0)
		for _, miner := range cluster.Miners {
			if miner.IsActive {
				totalPower += miner.PerformanceProfile.PowerConsumption
			}
		}
		cluster.CurrentPowerUsage = totalPower

		// Check power limits
		if cluster.PowerLimit > 0 && totalPower > cluster.PowerLimit {
			cs.handlePowerLimitExceeded(cluster)
		}

		// Simulate miner behaviors within cluster
		for _, miner := range cluster.Miners {
			if miner.IsActive {
				cs.simulateMinerInCluster(miner, cluster)
			}
		}
	}
}

func (cs *clusterSimulator) handlePowerLimitExceeded(cluster *Cluster) {
	// Temporarily reduce power by deactivating some miners
	excessPower := cluster.CurrentPowerUsage - cluster.PowerLimit
	powerReduced := uint32(0)

	for _, miner := range cluster.Miners {
		if miner.IsActive && powerReduced < excessPower {
			miner.IsActive = false
			powerReduced += miner.PerformanceProfile.PowerConsumption
		}
	}
}

func (cs *clusterSimulator) simulateMinerInCluster(miner *VirtualMiner, cluster *Cluster) {
	// Simulate coordinated behavior within cluster
	// Miners in the same cluster tend to have similar behavior patterns

	// Simulate share submission
	baseShareRate := float64(miner.HashRate) / 10000000.0 // shares per second

	if rand.Float64() < baseShareRate {
		miner.CurrentState.SharesSubmitted++
		miner.Statistics.TotalShares++
		miner.Statistics.ValidShares++
		miner.Statistics.LastShareTime = time.Now()
	}

	miner.CurrentState.LastSeen = time.Now()
}

func (cs *clusterSimulator) simulateFailures() {
	if !cs.config.FailureSimulation.EnableClusterFailures {
		return
	}

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-cs.stopChan:
			return
		case <-ticker.C:
			cs.processRandomFailures()
		}
	}
}

func (cs *clusterSimulator) processRandomFailures() {
	cs.mutex.RLock()
	failureRate := cs.config.FailureSimulation.FailureRate
	cs.mutex.RUnlock()

	if failureRate == 0 {
		return
	}

	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	for _, cluster := range cs.clusters {
		if cluster.IsActive && !cluster.IsInFailure && rand.Float64() < failureRate/60 { // Per minute
			// Trigger random failure
			duration := time.Minute * time.Duration(5+rand.Intn(25)) // 5-30 minutes
			go func(clusterID string) {
				cs.TriggerClusterFailure(clusterID, duration)
			}(cluster.ID)
		}
	}
}

func (cs *clusterSimulator) simulateCoordination() {
	ticker := time.NewTicker(time.Second * 10)
	defer ticker.Stop()

	for {
		select {
		case <-cs.stopChan:
			return
		case <-ticker.C:
			cs.processCoordination()
		}
	}
}

func (cs *clusterSimulator) processCoordination() {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	// Group clusters by coordinator
	coordinatorGroups := make(map[string][]*Cluster)
	for _, cluster := range cs.clusters {
		if cluster.IsActive && cluster.CoordinationConfig.SyncInterval > 0 {
			coordinatorGroups[cluster.Coordinator] = append(coordinatorGroups[cluster.Coordinator], cluster)
		}
	}

	// Process coordination for each group
	for _, clusters := range coordinatorGroups {
		cs.processCoordinatorGroup(clusters)
	}
}

func (cs *clusterSimulator) processCoordinatorGroup(clusters []*Cluster) {
	if len(clusters) == 0 {
		return
	}

	syncTime := time.Now()

	// Check if synchronization is needed
	for _, cluster := range clusters {
		if time.Since(cluster.LastSyncTime) >= cluster.CoordinationConfig.SyncInterval {
			cluster.LastSyncTime = syncTime
			cluster.Statistics.SyncEvents++
		}
	}

	// Check if leader election is needed
	hasLeader := false
	for _, cluster := range clusters {
		if cluster.IsLeader {
			hasLeader = true
			break
		}
	}

	if !hasLeader && len(clusters) > 0 && clusters[0].CoordinationConfig.LeaderElection {
		clusterIDs := make([]string, len(clusters))
		for i, cluster := range clusters {
			clusterIDs[i] = cluster.ID
		}
		cs.performLeaderElectionForClusters(clusterIDs)
	}
}

func (cs *clusterSimulator) performLeaderElection() {
	// Group clusters by coordinator for leader election
	coordinatorGroups := make(map[string][]string)
	for _, cluster := range cs.clusters {
		if cluster.CoordinationConfig.LeaderElection {
			coordinatorGroups[cluster.Coordinator] = append(coordinatorGroups[cluster.Coordinator], cluster.ID)
		}
	}

	for _, clusterIDs := range coordinatorGroups {
		if len(clusterIDs) > 1 {
			cs.performLeaderElectionForClusters(clusterIDs)
		}
	}
}

func (cs *clusterSimulator) performLeaderElectionForClusters(clusterIDs []string) string {
	if len(clusterIDs) == 0 {
		return ""
	}

	// Clear existing leaders
	for _, clusterID := range clusterIDs {
		cluster := cs.clusters[clusterID]
		if cluster != nil {
			cluster.IsLeader = false
		}
	}

	// Elect new leader (simple random selection for simulation)
	leaderID := clusterIDs[rand.Intn(len(clusterIDs))]
	leader := cs.clusters[leaderID]
	if leader != nil {
		leader.IsLeader = true
	}

	return leaderID
}

func (cs *clusterSimulator) executeFailover(clusterID string, duration time.Duration) {
	time.Sleep(time.Second * 2) // Brief delay before failover

	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	cluster := cs.clusters[clusterID]
	if cluster == nil {
		return
	}

	// Activate backup clusters
	for _, backupID := range cluster.FailoverConfig.BackupClusters {
		backup := cs.clusters[backupID]
		if backup != nil && backup.IsBackup {
			backup.IsActive = true
			for _, miner := range backup.Miners {
				miner.IsActive = true
				miner.CurrentState.LastSeen = time.Now()
			}
		}
	}

	// Schedule recovery
	go cs.scheduleRecovery(clusterID, duration)
}

func (cs *clusterSimulator) scheduleRecovery(clusterID string, duration time.Duration) {
	time.Sleep(duration)

	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	cluster := cs.clusters[clusterID]
	if cluster != nil {
		cluster.IsInFailure = false
		cluster.IsActive = true
		cluster.Statistics.LastRecoveryTime = time.Now()

		// Reactivate miners
		for _, miner := range cluster.Miners {
			miner.IsActive = true
			miner.CurrentState.IsDisconnected = false
			miner.CurrentState.LastSeen = time.Now()
		}

		// Deactivate backup clusters if they were activated
		for _, backupID := range cluster.FailoverConfig.BackupClusters {
			backup := cs.clusters[backupID]
			if backup != nil && backup.IsBackup {
				backup.IsActive = false
				for _, miner := range backup.Miners {
					miner.IsActive = false
				}
			}
		}
	}
}

func (cs *clusterSimulator) executeMigrationPlan(plan *MigrationPlan) {
	cs.mutex.Lock()
	plan.Status = "in_progress"

	progress := cs.migrationProgress[plan.ID]
	if progress == nil {
		cs.mutex.Unlock()
		return
	}
	strategy := plan.Strategy
	cs.mutex.Unlock()

	switch strategy {
	case "immediate":
		cs.executeImmediateMigration(plan, progress)
	case "gradual":
		cs.executeGradualMigration(plan, progress)
	case "scheduled":
		cs.executeScheduledMigration(plan, progress)
	default:
		cs.executeGradualMigration(plan, progress) // Default to gradual
	}
}

func (cs *clusterSimulator) executeImmediateMigration(plan *MigrationPlan, progress *MigrationProgress) {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	for _, clusterID := range plan.ClusterIDs {
		cluster := cs.clusters[clusterID]
		if cluster != nil {
			cluster.CurrentPool = plan.TargetPool
			cluster.Statistics.MigrationEvents++
			progress.MigratedMiners += uint32(len(cluster.Miners))
		}
	}

	progress.ProgressPercent = 100.0
	progress.Status = "completed"
	plan.Status = "completed"
}

func (cs *clusterSimulator) executeGradualMigration(plan *MigrationPlan, progress *MigrationProgress) {
	batchSize := 10 // Default batch size
	if len(plan.ClusterIDs) > 0 {
		// Find a strategy with batch size
		for _, strategy := range cs.config.MigrationConfig.MigrationStrategies {
			if strategy.Type == "gradual" && strategy.BatchSize > 0 {
				batchSize = strategy.BatchSize
				break
			}
		}
	}

	batches := progress.TotalMiners / uint32(batchSize)
	if batches == 0 {
		batches = 1
	}
	interval := plan.EstimatedDuration / time.Duration(batches)
	if interval == 0 {
		interval = time.Millisecond * 100
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-cs.stopChan:
			return
		case <-ticker.C:
			cs.mutex.Lock()

			if plan.Status == "cancelled" {
				cs.mutex.Unlock()
				return
			}

			migrated := cs.migrateBatch(plan, batchSize)
			progress.MigratedMiners += uint32(migrated)
			progress.ProgressPercent = float64(progress.MigratedMiners) / float64(progress.TotalMiners) * 100.0

			if progress.MigratedMiners >= progress.TotalMiners {
				progress.Status = "completed"
				plan.Status = "completed"
				cs.mutex.Unlock()
				return
			}

			cs.mutex.Unlock()
		}
	}
}

func (cs *clusterSimulator) executeScheduledMigration(plan *MigrationPlan, progress *MigrationProgress) {
	// Wait until scheduled start time
	if plan.StartTime.After(time.Now()) {
		time.Sleep(time.Until(plan.StartTime))
	}

	// Execute as immediate migration
	cs.executeImmediateMigration(plan, progress)
}

func (cs *clusterSimulator) migrateBatch(plan *MigrationPlan, batchSize int) int {
	migrated := 0

	for _, clusterID := range plan.ClusterIDs {
		cluster := cs.clusters[clusterID]
		if cluster != nil && cluster.CurrentPool == plan.SourcePool {
			// Migrate miners in batches
			minersToMigrate := batchSize
			if minersToMigrate > len(cluster.Miners) {
				minersToMigrate = len(cluster.Miners)
			}

			// For simplicity, migrate entire cluster at once
			cluster.CurrentPool = plan.TargetPool
			cluster.Statistics.MigrationEvents++
			migrated += len(cluster.Miners)
			break // Only migrate one cluster per batch
		}
	}

	return migrated
}

func (cs *clusterSimulator) updateStatistics() {
	ticker := time.NewTicker(time.Second * 10)
	defer ticker.Stop()

	for {
		select {
		case <-cs.stopChan:
			return
		case <-ticker.C:
			cs.calculateAllStats()
		}
	}
}

func (cs *clusterSimulator) calculateAllStats() {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	// Calculate stats for each cluster
	for _, cluster := range cs.clusters {
		cs.calculateClusterStats(cluster)
	}

	// Calculate overall stats
	cs.calculateOverallStats()
}

func (cs *clusterSimulator) calculateClusterStats(cluster *Cluster) {
	stats := cluster.Statistics

	stats.MinerCount = uint32(len(cluster.Miners))
	stats.ActiveMiners = 0
	stats.TotalHashRate = 0
	stats.TotalShares = 0
	stats.ValidShares = 0
	stats.InvalidShares = 0

	for _, miner := range cluster.Miners {
		if miner.IsActive {
			stats.ActiveMiners++
			stats.TotalHashRate += miner.HashRate
		}
		stats.TotalShares += miner.Statistics.TotalShares
		stats.ValidShares += miner.Statistics.ValidShares
		stats.InvalidShares += miner.Statistics.InvalidShares
	}

	if stats.ActiveMiners > 0 {
		stats.AverageHashRate = stats.TotalHashRate / uint64(stats.ActiveMiners)
		stats.UptimePercentage = float64(stats.ActiveMiners) / float64(stats.MinerCount) * 100.0
	}

	if cluster.CurrentPowerUsage > 0 {
		stats.PowerEfficiency = float64(stats.TotalHashRate) / float64(cluster.CurrentPowerUsage)
	}
}

func (cs *clusterSimulator) calculateOverallStats() {
	stats := cs.overallStats

	stats.TotalClusters = uint32(len(cs.clusters))
	stats.ActiveClusters = 0
	stats.TotalMiners = 0
	stats.ActiveMiners = 0
	stats.TotalHashRate = 0
	stats.TotalPowerUsage = 0
	stats.FailoverEvents = 0
	stats.MigrationEvents = 0
	stats.GeographicalDistribution = make(map[string]uint32)

	for _, cluster := range cs.clusters {
		if cluster.IsActive {
			stats.ActiveClusters++
		}

		stats.TotalMiners += uint32(len(cluster.Miners))
		stats.ActiveMiners += cluster.Statistics.ActiveMiners
		stats.TotalHashRate += cluster.Statistics.TotalHashRate
		stats.TotalPowerUsage += cluster.CurrentPowerUsage
		stats.FailoverEvents += cluster.Statistics.FailoverEvents
		stats.MigrationEvents += cluster.Statistics.MigrationEvents
		stats.GeographicalDistribution[cluster.Location]++
	}

	if stats.ActiveMiners > 0 {
		stats.AverageHashRate = stats.TotalHashRate / uint64(stats.ActiveMiners)
		stats.UptimePercentage = float64(stats.ActiveMiners) / float64(stats.TotalMiners) * 100.0
	}

	if stats.TotalPowerUsage > 0 {
		stats.PowerEfficiency = float64(stats.TotalHashRate) / float64(stats.TotalPowerUsage)
	}
}
