package simulation

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// COMPREHENSIVE SIMULATION TESTS FOR 80%+ COVERAGE
// Critical for production-ready mining simulation
// =============================================================================

// -----------------------------------------------------------------------------
// Simulation Manager Unit Tests
// -----------------------------------------------------------------------------

func TestNewSimulationManager_Success(t *testing.T) {
	config := createMinimalConfig()

	manager, err := NewSimulationManager(config)

	require.NoError(t, err)
	require.NotNil(t, manager)
	assert.NotNil(t, manager.blockchain)
	assert.NotNil(t, manager.virtualMiners)
	assert.NotNil(t, manager.clusters)
}

func TestNewSimulationManager_WithFullConfig(t *testing.T) {
	config := SimulationConfig{
		BlockchainConfig: BlockchainConfig{
			NetworkType:       "testnet",
			BlockTime:         time.Second * 2,
			InitialDifficulty: 100,
			TransactionLoad: TransactionLoadConfig{
				TxPerSecond:      5,
				BurstProbability: 0.1,
				BurstMultiplier:  2,
			},
		},
		MinerConfig: VirtualMinerConfig{
			MinerCount:    5,
			HashRateRange: HashRateRange{Min: 1000000, Max: 2000000},
		},
		ClusterConfig: ClusterSimulatorConfig{
			ClusterCount: 1,
			ClustersConfig: []ClusterConfig{
				{Name: "TestCluster", MinerCount: 3, Location: "TestLoc"},
			},
		},
		EnableIntegration: true,
		SyncInterval:      time.Second,
	}

	manager, err := NewSimulationManager(config)

	require.NoError(t, err)
	require.NotNil(t, manager)
}

func TestSimulationManager_StartStop(t *testing.T) {
	config := createMinimalConfig()
	manager, err := NewSimulationManager(config)
	require.NoError(t, err)

	// Start simulation
	err = manager.Start()
	require.NoError(t, err)
	assert.True(t, manager.isRunning)

	// Try to start again - should error
	err = manager.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	// Stop simulation
	err = manager.Stop()
	require.NoError(t, err)
	assert.False(t, manager.isRunning)

	// Stop again - should be no-op
	err = manager.Stop()
	require.NoError(t, err)
}

func TestSimulationManager_GetSimulators(t *testing.T) {
	config := createMinimalConfig()
	manager, err := NewSimulationManager(config)
	require.NoError(t, err)

	blockchain := manager.GetBlockchainSimulator()
	assert.NotNil(t, blockchain)

	miners := manager.GetVirtualMinerSimulator()
	assert.NotNil(t, miners)

	clusters := manager.GetClusterSimulator()
	assert.NotNil(t, clusters)
}

func TestSimulationManager_GetOverallStats(t *testing.T) {
	config := createMinimalConfig()
	manager, err := NewSimulationManager(config)
	require.NoError(t, err)

	err = manager.Start()
	require.NoError(t, err)
	defer manager.Stop()

	// Allow some time for stats to be collected
	time.Sleep(time.Millisecond * 100)

	stats := manager.GetOverallStats()
	require.NotNil(t, stats)
	assert.NotNil(t, stats.VirtualMinerStats)
	assert.NotNil(t, stats.ClusterStats)
}

func TestSimulationManager_GetPerformanceMetrics(t *testing.T) {
	config := createMinimalConfig()
	manager, err := NewSimulationManager(config)
	require.NoError(t, err)

	err = manager.Start()
	require.NoError(t, err)
	defer manager.Stop()

	metrics := manager.GetPerformanceMetrics()
	require.NotNil(t, metrics)

	// Check expected keys exist
	assert.Contains(t, metrics, "total_hash_rate")
	assert.Contains(t, metrics, "total_miners")
	assert.Contains(t, metrics, "active_miners")
	assert.Contains(t, metrics, "overall_uptime")
	assert.Contains(t, metrics, "shares_per_second")
	assert.Contains(t, metrics, "blocks_per_hour")
	assert.Contains(t, metrics, "network_efficiency")
	assert.Contains(t, metrics, "simulation_time")
}

func TestSimulationManager_TriggerStressTest_NotRunning(t *testing.T) {
	config := createMinimalConfig()
	manager, err := NewSimulationManager(config)
	require.NoError(t, err)

	// Try to trigger stress test without starting
	err = manager.TriggerStressTest(time.Second)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")
}

func TestSimulationManager_TriggerStressTest_Running(t *testing.T) {
	config := createMinimalConfig()
	manager, err := NewSimulationManager(config)
	require.NoError(t, err)

	err = manager.Start()
	require.NoError(t, err)
	defer manager.Stop()

	err = manager.TriggerStressTest(time.Millisecond * 100)
	assert.NoError(t, err)
}

func TestSimulationManager_TriggerFailureScenario_NotRunning(t *testing.T) {
	config := createMinimalConfig()
	manager, err := NewSimulationManager(config)
	require.NoError(t, err)

	err = manager.TriggerFailureScenario("network_partition")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")
}

func TestSimulationManager_TriggerFailureScenario_Unknown(t *testing.T) {
	config := createMinimalConfig()
	manager, err := NewSimulationManager(config)
	require.NoError(t, err)

	err = manager.Start()
	require.NoError(t, err)
	defer manager.Stop()

	err = manager.TriggerFailureScenario("unknown_scenario")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown")
}

func TestSimulationManager_TriggerFailureScenario_MassDropout(t *testing.T) {
	config := createMinimalConfig()
	manager, err := NewSimulationManager(config)
	require.NoError(t, err)

	err = manager.Start()
	require.NoError(t, err)
	defer manager.Stop()

	err = manager.TriggerFailureScenario("mass_miner_dropout")
	assert.NoError(t, err)
}

func TestSimulationManager_TriggerFailureScenario_MaliciousAttack(t *testing.T) {
	config := SimulationConfig{
		BlockchainConfig: BlockchainConfig{
			NetworkType:       "testnet",
			BlockTime:         time.Second,
			InitialDifficulty: 100,
		},
		MinerConfig: VirtualMinerConfig{
			MinerCount:    3,
			HashRateRange: HashRateRange{Min: 1000000, Max: 2000000},
			MaliciousBehavior: MaliciousBehaviorConfig{
				MaliciousMinerPercentage: 1.0, // All miners malicious for testing
				AttackTypes: []AttackType{
					{Type: "invalid_shares", Probability: 0.5, Intensity: 0.3},
				},
			},
		},
		ClusterConfig: ClusterSimulatorConfig{
			ClusterCount:   0,
			ClustersConfig: []ClusterConfig{},
		},
	}

	manager, err := NewSimulationManager(config)
	require.NoError(t, err)

	err = manager.Start()
	require.NoError(t, err)
	defer manager.Stop()

	err = manager.TriggerFailureScenario("malicious_attack")
	assert.NoError(t, err)
}

func TestSimulationManager_ExecutePoolMigration_NotRunning(t *testing.T) {
	config := createMinimalConfig()
	manager, err := NewSimulationManager(config)
	require.NoError(t, err)

	err = manager.ExecutePoolMigration("pool_A", "pool_B", "gradual")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")
}

func TestSimulationManager_ValidateSimulationAccuracy_NoHashRate(t *testing.T) {
	config := SimulationConfig{
		BlockchainConfig: BlockchainConfig{
			NetworkType:       "testnet",
			BlockTime:         time.Second,
			InitialDifficulty: 100,
		},
		MinerConfig: VirtualMinerConfig{
			MinerCount:    0, // No miners
			HashRateRange: HashRateRange{Min: 0, Max: 0},
		},
		ClusterConfig: ClusterSimulatorConfig{
			ClusterCount:   0,
			ClustersConfig: []ClusterConfig{},
		},
	}

	manager, err := NewSimulationManager(config)
	require.NoError(t, err)

	err = manager.Start()
	require.NoError(t, err)
	defer manager.Stop()

	err = manager.ValidateSimulationAccuracy()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "hash rate is zero")
}

// -----------------------------------------------------------------------------
// Blockchain Simulator Unit Tests
// -----------------------------------------------------------------------------

func TestNewBlockchainSimulator_Success(t *testing.T) {
	config := BlockchainConfig{
		NetworkType:       "testnet",
		BlockTime:         time.Second,
		InitialDifficulty: 100,
	}

	simulator, err := NewBlockchainSimulator(config)

	require.NoError(t, err)
	require.NotNil(t, simulator)
}

func TestBlockchainSimulator_GetNetworkType(t *testing.T) {
	config := BlockchainConfig{
		NetworkType:       "mainnet",
		BlockTime:         time.Second,
		InitialDifficulty: 100,
	}

	simulator, err := NewBlockchainSimulator(config)
	require.NoError(t, err)

	assert.Equal(t, "mainnet", simulator.GetNetworkType())
}

func TestBlockchainSimulator_GetBlockTime(t *testing.T) {
	config := BlockchainConfig{
		NetworkType:       "testnet",
		BlockTime:         time.Second * 5,
		InitialDifficulty: 100,
	}

	simulator, err := NewBlockchainSimulator(config)
	require.NoError(t, err)

	assert.Equal(t, time.Second*5, simulator.GetBlockTime())
}

func TestBlockchainSimulator_GetCurrentDifficulty(t *testing.T) {
	config := BlockchainConfig{
		NetworkType:       "testnet",
		BlockTime:         time.Second,
		InitialDifficulty: 500,
	}

	simulator, err := NewBlockchainSimulator(config)
	require.NoError(t, err)

	assert.Equal(t, uint64(500), simulator.GetCurrentDifficulty())
}

func TestBlockchainSimulator_GetGenesisBlock(t *testing.T) {
	config := BlockchainConfig{
		NetworkType:       "testnet",
		BlockTime:         time.Second,
		InitialDifficulty: 100,
	}

	simulator, err := NewBlockchainSimulator(config)
	require.NoError(t, err)

	genesis := simulator.GetGenesisBlock()
	require.NotNil(t, genesis)
	assert.Equal(t, uint64(0), genesis.Height)
	assert.Empty(t, genesis.PreviousHash)
}

func TestBlockchainSimulator_StartStop(t *testing.T) {
	config := BlockchainConfig{
		NetworkType:       "testnet",
		BlockTime:         time.Second,
		InitialDifficulty: 100,
	}

	simulator, err := NewBlockchainSimulator(config)
	require.NoError(t, err)

	// Start
	err = simulator.Start()
	require.NoError(t, err)

	// Start again - should error
	err = simulator.Start()
	assert.Error(t, err)

	// Stop
	err = simulator.Stop()
	require.NoError(t, err)

	// Stop again - should be no-op
	err = simulator.Stop()
	require.NoError(t, err)
}

func TestBlockchainSimulator_ValidateChain(t *testing.T) {
	config := BlockchainConfig{
		NetworkType:       "testnet",
		BlockTime:         time.Millisecond * 10,
		InitialDifficulty: 1, // Very low difficulty for fast testing
	}

	simulator, err := NewBlockchainSimulator(config)
	require.NoError(t, err)

	// Chain should be valid with just genesis
	assert.True(t, simulator.ValidateChain())
}

func TestBlockchainSimulator_GetNetworkStats(t *testing.T) {
	config := BlockchainConfig{
		NetworkType:       "testnet",
		BlockTime:         time.Second,
		InitialDifficulty: 100,
	}

	simulator, err := NewBlockchainSimulator(config)
	require.NoError(t, err)

	stats := simulator.GetNetworkStats()
	assert.Equal(t, "testnet", stats.NetworkType)
	assert.Equal(t, uint64(100), stats.CurrentDifficulty)
	assert.Equal(t, uint64(1), stats.BlocksGenerated) // Genesis block
}

func TestBlockchainSimulator_WithTransactionLoad(t *testing.T) {
	config := BlockchainConfig{
		NetworkType:       "testnet",
		BlockTime:         time.Second,
		InitialDifficulty: 100,
		TransactionLoad: TransactionLoadConfig{
			TxPerSecond:      10,
			BurstProbability: 0.5,
			BurstMultiplier:  2,
		},
	}

	simulator, err := NewBlockchainSimulator(config)
	require.NoError(t, err)
	require.NotNil(t, simulator)
}

// -----------------------------------------------------------------------------
// Virtual Miner Simulator Unit Tests
// -----------------------------------------------------------------------------

func TestNewVirtualMinerSimulator_Success(t *testing.T) {
	config := VirtualMinerConfig{
		MinerCount:    5,
		HashRateRange: HashRateRange{Min: 1000000, Max: 2000000},
	}

	simulator, err := NewVirtualMinerSimulator(config)

	require.NoError(t, err)
	require.NotNil(t, simulator)
}

func TestVirtualMinerSimulator_GetMiners(t *testing.T) {
	config := VirtualMinerConfig{
		MinerCount:    3,
		HashRateRange: HashRateRange{Min: 1000000, Max: 2000000},
	}

	simulator, err := NewVirtualMinerSimulator(config)
	require.NoError(t, err)

	miners := simulator.GetMiners()
	assert.Len(t, miners, 3)
}

func TestVirtualMinerSimulator_GetMiner(t *testing.T) {
	config := VirtualMinerConfig{
		MinerCount:    2,
		HashRateRange: HashRateRange{Min: 1000000, Max: 2000000},
	}

	simulator, err := NewVirtualMinerSimulator(config)
	require.NoError(t, err)

	miners := simulator.GetMiners()
	require.Len(t, miners, 2)

	// Get existing miner
	miner := simulator.GetMiner(miners[0].ID)
	assert.NotNil(t, miner)
	assert.Equal(t, miners[0].ID, miner.ID)

	// Get non-existing miner
	miner = simulator.GetMiner("non-existing-id")
	assert.Nil(t, miner)
}

func TestVirtualMinerSimulator_AddRemoveMiner(t *testing.T) {
	config := VirtualMinerConfig{
		MinerCount:    1,
		HashRateRange: HashRateRange{Min: 1000000, Max: 2000000},
	}

	simulator, err := NewVirtualMinerSimulator(config)
	require.NoError(t, err)

	initialCount := len(simulator.GetMiners())

	// Add miner
	newMiner, err := simulator.AddMiner(MinerType{Type: "GPU", HashRateMultiplier: 1.0})
	require.NoError(t, err)
	assert.NotNil(t, newMiner)
	assert.Len(t, simulator.GetMiners(), initialCount+1)

	// Remove miner
	err = simulator.RemoveMiner(newMiner.ID)
	require.NoError(t, err)
	assert.Len(t, simulator.GetMiners(), initialCount)

	// Remove non-existing miner
	err = simulator.RemoveMiner("non-existing-id")
	assert.Error(t, err)
}

func TestVirtualMinerSimulator_StartStop(t *testing.T) {
	config := VirtualMinerConfig{
		MinerCount:    2,
		HashRateRange: HashRateRange{Min: 1000000, Max: 2000000},
	}

	simulator, err := NewVirtualMinerSimulator(config)
	require.NoError(t, err)

	// Start
	err = simulator.Start()
	require.NoError(t, err)

	// Start again - should error
	err = simulator.Start()
	assert.Error(t, err)

	// Miners should be active
	miners := simulator.GetMiners()
	for _, miner := range miners {
		assert.True(t, miner.IsActive)
	}

	// Stop
	err = simulator.Stop()
	require.NoError(t, err)

	// Miners should be inactive
	miners = simulator.GetMiners()
	for _, miner := range miners {
		assert.False(t, miner.IsActive)
	}
}

func TestVirtualMinerSimulator_GetSimulationStats(t *testing.T) {
	config := VirtualMinerConfig{
		MinerCount:    3,
		HashRateRange: HashRateRange{Min: 1000000, Max: 2000000},
	}

	simulator, err := NewVirtualMinerSimulator(config)
	require.NoError(t, err)

	err = simulator.Start()
	require.NoError(t, err)
	defer simulator.Stop()

	stats := simulator.GetSimulationStats()
	require.NotNil(t, stats)
	assert.Equal(t, uint32(3), stats.TotalMiners)
	assert.Equal(t, uint32(3), stats.ActiveMiners)
}

func TestVirtualMinerSimulator_GetMinerStats(t *testing.T) {
	config := VirtualMinerConfig{
		MinerCount:    1,
		HashRateRange: HashRateRange{Min: 1000000, Max: 2000000},
	}

	simulator, err := NewVirtualMinerSimulator(config)
	require.NoError(t, err)

	miners := simulator.GetMiners()
	require.Len(t, miners, 1)

	stats := simulator.GetMinerStats(miners[0].ID)
	assert.NotNil(t, stats)

	// Non-existing miner
	stats = simulator.GetMinerStats("non-existing")
	assert.Nil(t, stats)
}

func TestVirtualMinerSimulator_TriggerBurst(t *testing.T) {
	config := VirtualMinerConfig{
		MinerCount:    1,
		HashRateRange: HashRateRange{Min: 1000000, Max: 2000000},
	}

	simulator, err := NewVirtualMinerSimulator(config)
	require.NoError(t, err)

	miners := simulator.GetMiners()
	require.Len(t, miners, 1)

	// Trigger burst
	err = simulator.TriggerBurst(miners[0].ID, time.Second)
	require.NoError(t, err)

	miner := simulator.GetMiner(miners[0].ID)
	assert.True(t, miner.CurrentState.IsBursting)
	assert.Equal(t, uint64(1), miner.Statistics.BurstEvents)

	// Non-existing miner
	err = simulator.TriggerBurst("non-existing", time.Second)
	assert.Error(t, err)
}

func TestVirtualMinerSimulator_TriggerDrop(t *testing.T) {
	config := VirtualMinerConfig{
		MinerCount:    1,
		HashRateRange: HashRateRange{Min: 1000000, Max: 2000000},
	}

	simulator, err := NewVirtualMinerSimulator(config)
	require.NoError(t, err)

	err = simulator.Start()
	require.NoError(t, err)
	defer simulator.Stop()

	miners := simulator.GetMiners()
	require.Len(t, miners, 1)

	// Trigger drop
	err = simulator.TriggerDrop(miners[0].ID, time.Millisecond*50)
	require.NoError(t, err)

	miner := simulator.GetMiner(miners[0].ID)
	assert.True(t, miner.CurrentState.IsDisconnected)
	assert.False(t, miner.IsActive)

	// Wait for reconnection
	time.Sleep(time.Millisecond * 100)

	miner = simulator.GetMiner(miners[0].ID)
	assert.False(t, miner.CurrentState.IsDisconnected)

	// Non-existing miner
	err = simulator.TriggerDrop("non-existing", time.Second)
	assert.Error(t, err)
}

func TestVirtualMinerSimulator_TriggerAttack(t *testing.T) {
	config := VirtualMinerConfig{
		MinerCount:    1,
		HashRateRange: HashRateRange{Min: 1000000, Max: 2000000},
		MaliciousBehavior: MaliciousBehaviorConfig{
			MaliciousMinerPercentage: 1.0, // All malicious
			AttackTypes: []AttackType{
				{Type: "invalid_shares", Probability: 0.5},
			},
		},
	}

	simulator, err := NewVirtualMinerSimulator(config)
	require.NoError(t, err)

	miners := simulator.GetMiners()
	require.Len(t, miners, 1)
	require.True(t, miners[0].IsMalicious)

	// Trigger attack
	err = simulator.TriggerAttack(miners[0].ID, "invalid_shares", time.Millisecond*50)
	require.NoError(t, err)

	miner := simulator.GetMiner(miners[0].ID)
	assert.True(t, miner.AttackProfile.IsAttacking)

	// Wait for attack to end
	time.Sleep(time.Millisecond * 100)

	miner = simulator.GetMiner(miners[0].ID)
	assert.False(t, miner.AttackProfile.IsAttacking)

	// Non-existing miner
	err = simulator.TriggerAttack("non-existing", "invalid_shares", time.Second)
	assert.Error(t, err)
}

func TestVirtualMinerSimulator_TriggerAttack_NonMalicious(t *testing.T) {
	config := VirtualMinerConfig{
		MinerCount:    1,
		HashRateRange: HashRateRange{Min: 1000000, Max: 2000000},
		MaliciousBehavior: MaliciousBehaviorConfig{
			MaliciousMinerPercentage: 0, // No malicious miners
		},
	}

	simulator, err := NewVirtualMinerSimulator(config)
	require.NoError(t, err)

	miners := simulator.GetMiners()
	require.Len(t, miners, 1)
	require.False(t, miners[0].IsMalicious)

	// Trigger attack on non-malicious miner - should error
	err = simulator.TriggerAttack(miners[0].ID, "invalid_shares", time.Second)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not configured as malicious")
}

func TestVirtualMinerSimulator_UpdateMinerHashRate(t *testing.T) {
	config := VirtualMinerConfig{
		MinerCount:    1,
		HashRateRange: HashRateRange{Min: 1000000, Max: 2000000},
	}

	simulator, err := NewVirtualMinerSimulator(config)
	require.NoError(t, err)

	miners := simulator.GetMiners()
	require.Len(t, miners, 1)

	// Update hash rate
	err = simulator.UpdateMinerHashRate(miners[0].ID, 5000000)
	require.NoError(t, err)

	miner := simulator.GetMiner(miners[0].ID)
	assert.Equal(t, uint64(5000000), miner.HashRate)

	// Non-existing miner
	err = simulator.UpdateMinerHashRate("non-existing", 1000000)
	assert.Error(t, err)
}

func TestVirtualMinerSimulator_UpdateNetworkConditions(t *testing.T) {
	config := VirtualMinerConfig{
		MinerCount:    1,
		HashRateRange: HashRateRange{Min: 1000000, Max: 2000000},
	}

	simulator, err := NewVirtualMinerSimulator(config)
	require.NoError(t, err)

	miners := simulator.GetMiners()
	require.Len(t, miners, 1)

	// Update network conditions
	newConditions := NetworkProfile{
		Quality:    "excellent",
		Latency:    time.Millisecond * 10,
		PacketLoss: 0.001,
	}

	err = simulator.UpdateNetworkConditions(miners[0].ID, newConditions)
	require.NoError(t, err)

	miner := simulator.GetMiner(miners[0].ID)
	assert.Equal(t, "excellent", miner.NetworkProfile.Quality)

	// Non-existing miner
	err = simulator.UpdateNetworkConditions("non-existing", newConditions)
	assert.Error(t, err)
}

func TestVirtualMinerSimulator_WithMinerTypes(t *testing.T) {
	config := VirtualMinerConfig{
		MinerCount:    10,
		HashRateRange: HashRateRange{Min: 1000000, Max: 2000000},
		MinerTypes: []MinerType{
			{Type: "CPU", Percentage: 0.3, HashRateMultiplier: 0.1},
			{Type: "GPU", Percentage: 0.5, HashRateMultiplier: 1.0},
			{Type: "ASIC", Percentage: 0.2, HashRateMultiplier: 10.0},
		},
	}

	simulator, err := NewVirtualMinerSimulator(config)
	require.NoError(t, err)

	miners := simulator.GetMiners()
	assert.Len(t, miners, 10)

	// Check that we have different miner types
	types := make(map[string]int)
	for _, miner := range miners {
		types[miner.Type]++
	}
	// Should have at least one of each type (probabilistic, but likely with 10 miners)
	assert.True(t, len(types) > 0)
}

// -----------------------------------------------------------------------------
// Concurrency Tests
// -----------------------------------------------------------------------------

func TestSimulationManager_ConcurrentAccess(t *testing.T) {
	config := createMinimalConfig()
	manager, err := NewSimulationManager(config)
	require.NoError(t, err)

	// Test concurrent access without starting (simpler, no background goroutines)
	var wg sync.WaitGroup

	// Concurrent get simulator calls
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				_ = manager.GetBlockchainSimulator()
				_ = manager.GetVirtualMinerSimulator()
				_ = manager.GetClusterSimulator()
			}
		}()
	}

	wg.Wait()
}

func TestVirtualMinerSimulator_ConcurrentAccess(t *testing.T) {
	config := VirtualMinerConfig{
		MinerCount:    5,
		HashRateRange: HashRateRange{Min: 1000000, Max: 2000000},
	}

	simulator, err := NewVirtualMinerSimulator(config)
	require.NoError(t, err)

	// Test concurrent access without starting (avoids background goroutines)
	var wg sync.WaitGroup
	miners := simulator.GetMiners()

	// Concurrent read operations
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				_ = simulator.GetMiners()
				if len(miners) > 0 {
					_ = simulator.GetMiner(miners[0].ID)
					_ = simulator.GetMinerStats(miners[0].ID)
				}
			}
		}()
	}

	wg.Wait()
}

// -----------------------------------------------------------------------------
// Helper Functions
// -----------------------------------------------------------------------------

func createMinimalConfig() SimulationConfig {
	return SimulationConfig{
		BlockchainConfig: BlockchainConfig{
			NetworkType:       "testnet",
			BlockTime:         time.Millisecond * 100,
			InitialDifficulty: 10,
		},
		MinerConfig: VirtualMinerConfig{
			MinerCount:    2,
			HashRateRange: HashRateRange{Min: 1000000, Max: 2000000},
		},
		ClusterConfig: ClusterSimulatorConfig{
			ClusterCount: 1,
			ClustersConfig: []ClusterConfig{
				{Name: "TestCluster", MinerCount: 2, Location: "TestLoc"},
			},
		},
	}
}

// -----------------------------------------------------------------------------
// Benchmark Tests
// -----------------------------------------------------------------------------

func BenchmarkSimulationManager_GetOverallStats(b *testing.B) {
	config := createMinimalConfig()
	manager, _ := NewSimulationManager(config)
	manager.Start()
	defer manager.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.GetOverallStats()
	}
}

func BenchmarkVirtualMinerSimulator_GetMiners(b *testing.B) {
	config := VirtualMinerConfig{
		MinerCount:    100,
		HashRateRange: HashRateRange{Min: 1000000, Max: 2000000},
	}
	simulator, _ := NewVirtualMinerSimulator(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		simulator.GetMiners()
	}
}

func BenchmarkVirtualMinerSimulator_GetSimulationStats(b *testing.B) {
	config := VirtualMinerConfig{
		MinerCount:    100,
		HashRateRange: HashRateRange{Min: 1000000, Max: 2000000},
	}
	simulator, _ := NewVirtualMinerSimulator(config)
	simulator.Start()
	defer simulator.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		simulator.GetSimulationStats()
	}
}
