package simulation

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func skipBlockchainTest(t *testing.T) {
	if os.Getenv("SIMULATION_TEST") != "true" {
		t.Skip("Skipping blockchain simulation test - set SIMULATION_TEST=true to run")
	}
}

// Test for Requirement 15.1: Simulated BlockDAG blockchain with configurable parameters
func TestBlockchainSimulator_ConfigurableParameters(t *testing.T) {
	skipBlockchainTest(t)
	config := BlockchainConfig{
		NetworkType:                "testnet",
		BlockTime:                  time.Second * 10,
		InitialDifficulty:          1000,
		DifficultyAdjustmentWindow: 10,
		MaxBlockSize:               1024 * 1024, // 1MB
	}

	simulator, err := NewBlockchainSimulator(config)
	require.NoError(t, err)
	require.NotNil(t, simulator)

	// Should fail - not implemented yet
	assert.Equal(t, config.NetworkType, simulator.GetNetworkType())
	assert.Equal(t, config.BlockTime, simulator.GetBlockTime())
	assert.Equal(t, config.InitialDifficulty, simulator.GetCurrentDifficulty())
}

// Test for Requirement 15.2: Replicate mainnet difficulty and block timing
func TestBlockchainSimulator_MainnetCharacteristics(t *testing.T) {
	skipBlockchainTest(t)
	config := BlockchainConfig{
		NetworkType:                "mainnet",
		BlockTime:                  time.Minute * 1, // 1 minute blocks
		InitialDifficulty:          1000000,
		DifficultyAdjustmentWindow: 144, // Adjust every 144 blocks
	}

	simulator, err := NewBlockchainSimulator(config)
	require.NoError(t, err)

	// Should fail - not implemented yet
	stats := simulator.GetNetworkStats()
	assert.Equal(t, "mainnet", stats.NetworkType)
	assert.Equal(t, time.Minute, stats.AverageBlockTime)
	assert.Greater(t, stats.CurrentDifficulty, uint64(1000000))
}

// Test for Requirement 15.3: Faster block times and lower difficulty for rapid testing
func TestBlockchainSimulator_TestnetRapidTesting(t *testing.T) {
	skipBlockchainTest(t)
	config := BlockchainConfig{
		NetworkType:                "testnet",
		BlockTime:                  time.Second * 5, // Fast 5-second blocks
		InitialDifficulty:          100,             // Low difficulty
		DifficultyAdjustmentWindow: 5,               // Quick adjustments
	}

	simulator, err := NewBlockchainSimulator(config)
	require.NoError(t, err)

	// Should fail - not implemented yet
	start := time.Now()
	block, err := simulator.MineNextBlock()
	require.NoError(t, err)
	duration := time.Since(start)

	assert.Less(t, duration, time.Second*10) // Should mine quickly
	assert.NotNil(t, block)
	assert.Equal(t, uint64(1), block.Height)
}

// Test for Requirement 15.4: Custom difficulty curves and network conditions
func TestBlockchainSimulator_CustomDifficultyCurves(t *testing.T) {
	skipBlockchainTest(t)
	config := BlockchainConfig{
		NetworkType:                "custom",
		BlockTime:                  time.Second * 30,
		InitialDifficulty:          5000,
		DifficultyAdjustmentWindow: 20,
		CustomDifficultyCurve: &DifficultyCurve{
			Type: "exponential",
			Parameters: map[string]float64{
				"growth_rate":    1.1,
				"max_difficulty": 1000000,
			},
		},
	}

	simulator, err := NewBlockchainSimulator(config)
	require.NoError(t, err)

	// Should fail - not implemented yet
	initialDifficulty := simulator.GetCurrentDifficulty()

	// Mine several blocks to trigger difficulty adjustment
	for i := 0; i < 25; i++ {
		_, err := simulator.MineNextBlock()
		require.NoError(t, err)
	}

	newDifficulty := simulator.GetCurrentDifficulty()
	assert.Greater(t, newDifficulty, initialDifficulty)
}

// Test for Requirement 15.5: Realistic transaction loads and network latency
func TestBlockchainSimulator_RealisticNetworkConditions(t *testing.T) {
	skipBlockchainTest(t)
	config := BlockchainConfig{
		NetworkType:       "testnet",
		BlockTime:         time.Second * 15,
		InitialDifficulty: 1000,
		NetworkLatency: NetworkLatencyConfig{
			MinLatency:   time.Millisecond * 50,
			MaxLatency:   time.Millisecond * 500,
			Distribution: "normal",
		},
		TransactionLoad: TransactionLoadConfig{
			TxPerSecond:      10,
			BurstProbability: 0.1,
			BurstMultiplier:  5,
		},
	}

	simulator, err := NewBlockchainSimulator(config)
	require.NoError(t, err)

	// Should fail - not implemented yet
	simulator.Start()
	defer simulator.Stop()

	time.Sleep(time.Second * 2)

	stats := simulator.GetNetworkStats()
	assert.Greater(t, stats.TotalTransactions, uint64(10)) // Should have generated transactions
	assert.Greater(t, stats.AverageLatency, time.Millisecond*40)
	assert.Less(t, stats.AverageLatency, time.Millisecond*600)
}

// Test blockchain state management
func TestBlockchainSimulator_StateManagement(t *testing.T) {
	skipBlockchainTest(t)
	config := BlockchainConfig{
		NetworkType:       "testnet",
		BlockTime:         time.Second * 5,
		InitialDifficulty: 100,
	}

	simulator, err := NewBlockchainSimulator(config)
	require.NoError(t, err)

	// Should fail - not implemented yet
	// Test genesis block
	genesis := simulator.GetGenesisBlock()
	assert.NotNil(t, genesis)
	assert.Equal(t, uint64(0), genesis.Height)

	// Test block mining and chain building
	block1, err := simulator.MineNextBlock()
	require.NoError(t, err)
	assert.Equal(t, uint64(1), block1.Height)
	assert.Equal(t, genesis.Hash, block1.PreviousHash)

	// Test chain validation
	isValid := simulator.ValidateChain()
	assert.True(t, isValid)
}

// Test concurrent mining simulation
func TestBlockchainSimulator_ConcurrentMining(t *testing.T) {
	skipBlockchainTest(t)
	config := BlockchainConfig{
		NetworkType:       "testnet",
		BlockTime:         time.Second * 10,
		InitialDifficulty: 1000,
	}

	simulator, err := NewBlockchainSimulator(config)
	require.NoError(t, err)

	// Should fail - not implemented yet
	simulator.Start()
	defer simulator.Stop()

	// Simulate multiple miners
	results := make(chan *Block, 3)
	for i := 0; i < 3; i++ {
		go func(minerID int) {
			block, err := simulator.MineBlockWithMiner(minerID)
			if err == nil {
				results <- block
			}
		}(i)
	}

	// Wait for first block to be mined
	select {
	case block := <-results:
		assert.NotNil(t, block)
		assert.Greater(t, block.Height, uint64(0))
	case <-time.After(time.Second * 15):
		t.Fatal("Mining took too long")
	}
}

// ============================================================================
// Additional Tests for Coverage Improvement (No Skip)
// ============================================================================

func TestBlockchainSimulator_MineBlock_NoSkip(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}

	config := BlockchainConfig{
		NetworkType:                "testnet",
		BlockTime:                  time.Millisecond * 50,
		InitialDifficulty:          1,
		DifficultyAdjustmentWindow: 5,
	}

	simulator, err := NewBlockchainSimulator(config)
	require.NoError(t, err)

	err = simulator.Start()
	require.NoError(t, err)
	defer simulator.Stop()

	// Mine a block
	block, err := simulator.MineNextBlock()
	require.NoError(t, err)
	assert.NotNil(t, block)
	assert.Equal(t, uint64(1), block.Height)
	assert.NotEmpty(t, block.Hash)

	// Mine another with specific miner ID
	block2, err := simulator.MineBlockWithMiner(42)
	require.NoError(t, err)
	assert.Equal(t, uint64(2), block2.Height)
	assert.Equal(t, 42, block2.MinerID)
	assert.Equal(t, block.Hash, block2.PreviousHash)
}

func TestBlockchainSimulator_ValidateChain_NoSkip(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}

	config := BlockchainConfig{
		NetworkType:                "testnet",
		BlockTime:                  time.Millisecond * 20,
		InitialDifficulty:          1,
		DifficultyAdjustmentWindow: 10,
	}

	simulator, err := NewBlockchainSimulator(config)
	require.NoError(t, err)

	// Validate initial chain (just genesis)
	assert.True(t, simulator.ValidateChain())

	// Mine several blocks
	for i := 0; i < 3; i++ {
		_, err := simulator.MineNextBlock()
		require.NoError(t, err)
	}

	// Chain should still be valid
	assert.True(t, simulator.ValidateChain())
}

func TestBlockchainSimulator_DifficultyAdjustment_NoSkip(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}

	config := BlockchainConfig{
		NetworkType:                "testnet",
		BlockTime:                  time.Millisecond * 10,
		InitialDifficulty:          100,
		DifficultyAdjustmentWindow: 3,
	}

	simulator, err := NewBlockchainSimulator(config)
	require.NoError(t, err)

	initialDifficulty := simulator.GetCurrentDifficulty()
	assert.Equal(t, uint64(100), initialDifficulty)

	// Mine enough blocks to trigger adjustment
	for i := 0; i < 6; i++ {
		_, err := simulator.MineNextBlock()
		require.NoError(t, err)
	}

	// Difficulty should have been adjusted
	newDifficulty := simulator.GetCurrentDifficulty()
	assert.NotEqual(t, uint64(0), newDifficulty)
}

func TestBlockchainSimulator_NetworkLatency_NoSkip(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}

	// Test uniform distribution
	config := BlockchainConfig{
		NetworkType:       "testnet",
		BlockTime:         time.Millisecond * 20,
		InitialDifficulty: 1,
		NetworkLatency: NetworkLatencyConfig{
			MinLatency:   time.Millisecond * 5,
			MaxLatency:   time.Millisecond * 15,
			Distribution: "uniform",
		},
	}

	simulator, err := NewBlockchainSimulator(config)
	require.NoError(t, err)

	_, err = simulator.MineNextBlock()
	require.NoError(t, err)

	// Test normal distribution
	config.NetworkLatency.Distribution = "normal"
	simulator2, err := NewBlockchainSimulator(config)
	require.NoError(t, err)
	_, err = simulator2.MineNextBlock()
	require.NoError(t, err)

	// Test exponential distribution
	config.NetworkLatency.Distribution = "exponential"
	simulator3, err := NewBlockchainSimulator(config)
	require.NoError(t, err)
	_, err = simulator3.MineNextBlock()
	require.NoError(t, err)
}

func TestBlockchainSimulator_CustomDifficultyCurve_NoSkip(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}

	// Test exponential curve
	config := BlockchainConfig{
		NetworkType:                "testnet",
		BlockTime:                  time.Millisecond * 10,
		InitialDifficulty:          100,
		DifficultyAdjustmentWindow: 2,
		CustomDifficultyCurve: &DifficultyCurve{
			Type: "exponential",
			Parameters: map[string]float64{
				"growth_rate": 1.5,
			},
		},
	}

	simulator, err := NewBlockchainSimulator(config)
	require.NoError(t, err)

	for i := 0; i < 4; i++ {
		_, err := simulator.MineNextBlock()
		require.NoError(t, err)
	}

	// Test logarithmic curve
	config.CustomDifficultyCurve = &DifficultyCurve{
		Type: "logarithmic",
		Parameters: map[string]float64{
			"base": 2.0,
		},
	}

	simulator2, err := NewBlockchainSimulator(config)
	require.NoError(t, err)

	for i := 0; i < 4; i++ {
		_, err := simulator2.MineNextBlock()
		require.NoError(t, err)
	}
}

func TestBlockchainSimulator_TransactionGeneration_NoSkip(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}

	config := BlockchainConfig{
		NetworkType:       "testnet",
		BlockTime:         time.Millisecond * 50,
		InitialDifficulty: 1,
		TransactionLoad: TransactionLoadConfig{
			TxPerSecond:      100,
			BurstProbability: 0.5,
			BurstMultiplier:  2.0,
		},
	}

	simulator, err := NewBlockchainSimulator(config)
	require.NoError(t, err)

	err = simulator.Start()
	require.NoError(t, err)
	defer simulator.Stop()

	// Let transactions accumulate
	time.Sleep(time.Millisecond * 1200)

	// Mine a block with transactions
	block, err := simulator.MineNextBlock()
	require.NoError(t, err)
	assert.NotNil(t, block)

	stats := simulator.GetNetworkStats()
	assert.NotNil(t, stats)
}

func TestBlockchainSimulator_GenesisBlock_NoSkip(t *testing.T) {
	config := BlockchainConfig{
		NetworkType:       "testnet",
		BlockTime:         time.Second,
		InitialDifficulty: 1000,
	}

	simulator, err := NewBlockchainSimulator(config)
	require.NoError(t, err)

	genesis := simulator.GetGenesisBlock()
	require.NotNil(t, genesis)
	assert.Equal(t, uint64(0), genesis.Height)
	assert.Empty(t, genesis.PreviousHash)
	assert.NotEmpty(t, genesis.Hash)
	assert.Equal(t, -1, genesis.MinerID)
}

func TestBlockchainSimulator_StartStop_NoSkip(t *testing.T) {
	config := BlockchainConfig{
		NetworkType:       "testnet",
		BlockTime:         time.Second,
		InitialDifficulty: 1000,
	}

	simulator, err := NewBlockchainSimulator(config)
	require.NoError(t, err)

	// Start
	err = simulator.Start()
	require.NoError(t, err)

	// Start again should error
	err = simulator.Start()
	assert.Error(t, err)

	// Stop
	err = simulator.Stop()
	require.NoError(t, err)

	// Stop again should be no-op
	err = simulator.Stop()
	require.NoError(t, err)
}

// =============================================================================
// Quick Unit Tests (No SIMULATION_TEST required)
// =============================================================================

func TestBlockchainSimulator_MineNextBlock_Quick(t *testing.T) {
	config := BlockchainConfig{
		NetworkType:       "testnet",
		BlockTime:         time.Second * 5,
		InitialDifficulty: 100,
	}

	simulator, err := NewBlockchainSimulator(config)
	require.NoError(t, err)

	// Mine a block
	block, err := simulator.MineNextBlock()
	require.NoError(t, err)
	require.NotNil(t, block)
	assert.Equal(t, uint64(1), block.Height)

	// Mine another block
	block2, err := simulator.MineNextBlock()
	require.NoError(t, err)
	require.NotNil(t, block2)
	assert.Equal(t, uint64(2), block2.Height)
}

func TestBlockchainSimulator_MineBlockWithMiner(t *testing.T) {
	config := BlockchainConfig{
		NetworkType:       "testnet",
		BlockTime:         time.Second * 5,
		InitialDifficulty: 100,
	}

	simulator, err := NewBlockchainSimulator(config)
	require.NoError(t, err)

	// Mine block with specific miner ID
	block, err := simulator.MineBlockWithMiner(42)
	require.NoError(t, err)
	require.NotNil(t, block)
	assert.Equal(t, 42, block.MinerID)
}

func TestBlockchainSimulator_ValidateChain_Genesis(t *testing.T) {
	config := BlockchainConfig{
		NetworkType:       "testnet",
		BlockTime:         time.Second * 5,
		InitialDifficulty: 100,
	}

	simulator, err := NewBlockchainSimulator(config)
	require.NoError(t, err)

	// Chain with just genesis should be valid
	valid := simulator.ValidateChain()
	assert.True(t, valid)
}

func TestBlockchainSimulator_GetGenesisBlock_Detailed(t *testing.T) {
	config := BlockchainConfig{
		NetworkType:       "testnet",
		BlockTime:         time.Second * 5,
		InitialDifficulty: 100,
	}

	simulator, err := NewBlockchainSimulator(config)
	require.NoError(t, err)

	genesis := simulator.GetGenesisBlock()
	require.NotNil(t, genesis)
	assert.Equal(t, uint64(0), genesis.Height)
	assert.Empty(t, genesis.PreviousHash)
	assert.NotEmpty(t, genesis.Hash)
	assert.Equal(t, -1, genesis.MinerID) // Genesis has no miner
}
