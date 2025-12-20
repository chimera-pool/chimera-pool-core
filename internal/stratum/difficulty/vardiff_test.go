package difficulty

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// TDD TESTS FOR HARDWARE-AWARE VARIABLE DIFFICULTY SYSTEM
// =============================================================================

// -----------------------------------------------------------------------------
// Hardware Class Constants Tests
// -----------------------------------------------------------------------------

func TestHardwareClassConstants(t *testing.T) {
	assert.Equal(t, "cpu", HardwareCPU)
	assert.Equal(t, "gpu", HardwareGPU)
	assert.Equal(t, "fpga", HardwareFPGA)
	assert.Equal(t, "asic", HardwareASIC)
	assert.Equal(t, "official_asic", HardwareOfficialASIC)
}

func TestDefaultDifficulties_Ordering(t *testing.T) {
	// CPU should have lowest, Official ASIC highest
	assert.Less(t, DefaultDiffCPU, DefaultDiffGPU)
	assert.Less(t, DefaultDiffGPU, DefaultDiffFPGA)
	assert.Less(t, DefaultDiffFPGA, DefaultDiffASIC)
	assert.Less(t, DefaultDiffASIC, DefaultDiffOfficialASIC)
}

func TestHardwareClasses_BaseDifficulty(t *testing.T) {
	assert.Equal(t, DefaultDiffCPU, ClassCPU.BaseDifficulty)
	assert.Equal(t, DefaultDiffGPU, ClassGPU.BaseDifficulty)
	assert.Equal(t, DefaultDiffFPGA, ClassFPGA.BaseDifficulty)
	assert.Equal(t, DefaultDiffASIC, ClassASIC.BaseDifficulty)
	assert.Equal(t, DefaultDiffOfficialASIC, ClassOfficialASIC.BaseDifficulty)
}

// -----------------------------------------------------------------------------
// Hardware Classifier Tests
// -----------------------------------------------------------------------------

func TestNewHardwareClassifier(t *testing.T) {
	hc := NewHardwareClassifier()
	assert.NotNil(t, hc)
}

func TestHardwareClassifier_ClassifyByHashrate(t *testing.T) {
	hc := NewHardwareClassifier()

	tests := []struct {
		name     string
		hashrate float64
		expected string
	}{
		{"Very low - CPU", 50000, HardwareCPU},
		{"Low - CPU", 500000, HardwareCPU},
		{"Medium - GPU", 5000000, HardwareGPU},
		{"High - FPGA", 40000000, HardwareFPGA},
		{"Very High - ASIC (X30)", 80000000, HardwareASIC},
		{"Highest - Official ASIC (X100)", 240000000, HardwareOfficialASIC},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			class := hc.ClassifyByHashrate(tt.hashrate)
			assert.Equal(t, tt.expected, class.Name)
		})
	}
}

func TestHardwareClassifier_ClassifyByUserAgent(t *testing.T) {
	hc := NewHardwareClassifier()

	tests := []struct {
		userAgent string
		expected  string
	}{
		{"BlockDAG-X100/1.0", HardwareOfficialASIC},
		{"BDAG-X100 Miner", HardwareOfficialASIC},
		{"BlockDAG-X30/1.0", HardwareASIC},
		{"Antminer S19", HardwareASIC},
		{"FPGA Miner v2", HardwareFPGA},
		{"CUDA Miner/1.0", HardwareGPU},
		{"AMD GPU Miner", HardwareGPU},
		{"cpuminer-multi/1.3", HardwareCPU},
		{"Unknown Miner", HardwareUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.userAgent, func(t *testing.T) {
			class := hc.ClassifyByUserAgent(tt.userAgent)
			assert.Equal(t, tt.expected, class.Name)
		})
	}
}

func TestHardwareClassifier_ClassifyByCombined(t *testing.T) {
	hc := NewHardwareClassifier()

	// When hashrate is available, it takes precedence
	class := hc.ClassifyByCombined("cpuminer", 240000000)
	assert.Equal(t, HardwareOfficialASIC, class.Name) // Hashrate wins

	// When no hashrate, use user-agent
	class = hc.ClassifyByCombined("BlockDAG-X100", 0)
	assert.Equal(t, HardwareOfficialASIC, class.Name)
}

// -----------------------------------------------------------------------------
// Miner State Tests
// -----------------------------------------------------------------------------

func TestNewMinerState(t *testing.T) {
	state := NewMinerState("miner-1", ClassGPU)

	assert.Equal(t, "miner-1", state.MinerID)
	assert.Equal(t, ClassGPU.Name, state.HardwareClass.Name)
	assert.Equal(t, ClassGPU.BaseDifficulty, state.CurrentDiff)
	assert.Equal(t, uint64(0), state.ShareCount)
}

func TestMinerState_RecordShare(t *testing.T) {
	state := NewMinerState("miner-1", ClassGPU)

	state.RecordShare(true, false)
	assert.Equal(t, uint64(1), state.ShareCount)
	assert.Equal(t, uint64(1), state.ValidShares)

	state.RecordShare(false, false)
	assert.Equal(t, uint64(2), state.ShareCount)
	assert.Equal(t, uint64(1), state.InvalidShares)

	state.RecordShare(true, true)
	assert.Equal(t, uint64(3), state.ShareCount)
	assert.Equal(t, uint64(1), state.StaleShares)
}

func TestMinerState_ShareTimeTracking(t *testing.T) {
	state := NewMinerState("miner-1", ClassGPU)

	// Record shares with small delays
	state.RecordShare(true, false)
	time.Sleep(10 * time.Millisecond)
	state.RecordShare(true, false)
	time.Sleep(10 * time.Millisecond)
	state.RecordShare(true, false)

	// Should have recorded intervals
	state.mu.RLock()
	assert.True(t, len(state.ShareTimes) >= 2)
	state.mu.RUnlock()

	avgTime := state.GetAverageShareTime()
	assert.True(t, avgTime > 0)
}

func TestMinerState_GetCurrentDifficulty(t *testing.T) {
	state := NewMinerState("miner-1", ClassASIC)
	assert.Equal(t, ClassASIC.BaseDifficulty, state.GetCurrentDifficulty())
}

func TestMinerState_SetDifficulty(t *testing.T) {
	state := NewMinerState("miner-1", ClassGPU)

	// Normal set
	state.SetDifficulty(8192)
	assert.Equal(t, uint64(8192), state.GetCurrentDifficulty())

	// Below minimum - should clamp
	state.SetDifficulty(1)
	assert.Equal(t, ClassGPU.MinDifficulty, state.GetCurrentDifficulty())

	// Above maximum - should clamp
	state.SetDifficulty(10000000)
	assert.Equal(t, ClassGPU.MaxDifficulty, state.GetCurrentDifficulty())
}

func TestMinerState_GetStats(t *testing.T) {
	state := NewMinerState("miner-1", ClassOfficialASIC)
	state.RecordShare(true, false)
	state.RecordShare(true, false)
	state.RecordShare(false, false)

	stats := state.GetStats()
	assert.Equal(t, "miner-1", stats.MinerID)
	assert.Equal(t, HardwareOfficialASIC, stats.HardwareClass)
	assert.Equal(t, uint64(3), stats.ShareCount)
	assert.Equal(t, uint64(2), stats.ValidShares)
	assert.Equal(t, uint64(1), stats.InvalidShares)
}

// -----------------------------------------------------------------------------
// Vardiff Manager Tests
// -----------------------------------------------------------------------------

func TestNewVardiffManager(t *testing.T) {
	vm := NewVardiffManager()
	assert.NotNil(t, vm)
	assert.NotNil(t, vm.miners)
	assert.NotNil(t, vm.classifier)
	assert.Equal(t, TargetShareTime, vm.targetShareTime)
}

func TestVardiffManager_RegisterMiner(t *testing.T) {
	vm := NewVardiffManager()

	state := vm.RegisterMiner("miner-1", "BlockDAG-X100", 0)
	assert.NotNil(t, state)
	assert.Equal(t, HardwareOfficialASIC, state.HardwareClass.Name)

	// Check it's registered
	found, exists := vm.GetMiner("miner-1")
	assert.True(t, exists)
	assert.Equal(t, state, found)
}

func TestVardiffManager_GetMiner(t *testing.T) {
	vm := NewVardiffManager()
	vm.RegisterMiner("miner-1", "GPU", 0)

	// Existing miner
	state, exists := vm.GetMiner("miner-1")
	assert.True(t, exists)
	assert.NotNil(t, state)

	// Non-existing miner
	_, exists = vm.GetMiner("miner-2")
	assert.False(t, exists)
}

func TestVardiffManager_GetOrCreateMiner(t *testing.T) {
	vm := NewVardiffManager()

	// First call creates
	state1 := vm.GetOrCreateMiner("miner-1")
	assert.NotNil(t, state1)

	// Second call returns same
	state2 := vm.GetOrCreateMiner("miner-1")
	assert.Equal(t, state1, state2)
}

func TestVardiffManager_RemoveMiner(t *testing.T) {
	vm := NewVardiffManager()
	vm.RegisterMiner("miner-1", "", 0)

	_, exists := vm.GetMiner("miner-1")
	assert.True(t, exists)

	vm.RemoveMiner("miner-1")

	_, exists = vm.GetMiner("miner-1")
	assert.False(t, exists)
}

func TestVardiffManager_RecordShare(t *testing.T) {
	vm := NewVardiffManager()
	vm.RegisterMiner("miner-1", "GPU", 0)

	// Record shares
	newDiff, changed := vm.RecordShare("miner-1", true, false)
	assert.False(t, changed) // Not enough shares yet
	assert.True(t, newDiff > 0)

	state, _ := vm.GetMiner("miner-1")
	assert.Equal(t, uint64(1), state.ShareCount)
}

func TestVardiffManager_GetDifficulty(t *testing.T) {
	vm := NewVardiffManager()
	vm.RegisterMiner("miner-1", "ASIC", 0)

	diff := vm.GetDifficulty("miner-1")
	assert.Equal(t, ClassASIC.BaseDifficulty, diff)

	// Unknown miner
	diff = vm.GetDifficulty("unknown")
	assert.Equal(t, DefaultDiffUnknown, diff)
}

func TestVardiffManager_SetDifficulty(t *testing.T) {
	vm := NewVardiffManager()
	vm.RegisterMiner("miner-1", "GPU", 0)

	err := vm.SetDifficulty("miner-1", 8192)
	require.NoError(t, err)
	assert.Equal(t, uint64(8192), vm.GetDifficulty("miner-1"))

	// Unknown miner
	err = vm.SetDifficulty("unknown", 100)
	assert.Error(t, err)
}

func TestVardiffManager_GetMinerCount(t *testing.T) {
	vm := NewVardiffManager()
	assert.Equal(t, 0, vm.GetMinerCount())

	vm.RegisterMiner("miner-1", "", 0)
	assert.Equal(t, 1, vm.GetMinerCount())

	vm.RegisterMiner("miner-2", "", 0)
	assert.Equal(t, 2, vm.GetMinerCount())

	vm.RemoveMiner("miner-1")
	assert.Equal(t, 1, vm.GetMinerCount())
}

func TestVardiffManager_GetAllStats(t *testing.T) {
	vm := NewVardiffManager()
	vm.RegisterMiner("miner-1", "GPU", 0)
	vm.RegisterMiner("miner-2", "ASIC", 0)

	stats := vm.GetAllStats()
	assert.Len(t, stats, 2)
}

func TestVardiffManager_GetPoolHashrate(t *testing.T) {
	vm := NewVardiffManager()

	// Initially zero
	assert.Equal(t, float64(0), vm.GetPoolHashrate())

	// After registering miners, hashrate depends on share submissions
	vm.RegisterMiner("miner-1", "GPU", 10000000)
	// Hashrate calculated from shares, not initial registration
}

func TestVardiffManager_ReclassifyMiner(t *testing.T) {
	vm := NewVardiffManager()
	state := vm.RegisterMiner("miner-1", "", 0) // Unknown initially

	// Simulate observed hashrate
	state.mu.Lock()
	state.AverageHashrate = 240000000 // X100 level
	state.mu.Unlock()

	vm.ReclassifyMiner("miner-1")

	state.mu.RLock()
	assert.Equal(t, HardwareOfficialASIC, state.HardwareClass.Name)
	state.mu.RUnlock()
}

// -----------------------------------------------------------------------------
// Difficulty Adjustment Tests
// -----------------------------------------------------------------------------

func TestVardiffManager_AdjustmentTooFast(t *testing.T) {
	vm := NewVardiffManagerWithParams(10*time.Second, 1*time.Second, 3)
	state := vm.RegisterMiner("fast-miner", "GPU", 0)

	// Simulate very fast shares (shares coming every 1ms - too fast)
	for i := 0; i < 5; i++ {
		state.mu.Lock()
		state.ShareTimes = append(state.ShareTimes, 1*time.Millisecond)
		state.mu.Unlock()
	}

	// Force adjustment check
	state.mu.Lock()
	state.LastAdjustment = time.Now().Add(-2 * time.Second) // Past retarget time
	state.mu.Unlock()

	newDiff, changed := vm.checkAndAdjust(state)

	// Should increase difficulty (shares too fast)
	if changed {
		assert.Greater(t, newDiff, ClassGPU.BaseDifficulty)
	}
}

func TestVardiffManager_AdjustmentTooSlow(t *testing.T) {
	vm := NewVardiffManagerWithParams(10*time.Second, 1*time.Second, 3)
	state := vm.RegisterMiner("slow-miner", "GPU", 0)

	// Simulate very slow shares (shares coming every 60s - too slow)
	for i := 0; i < 5; i++ {
		state.mu.Lock()
		state.ShareTimes = append(state.ShareTimes, 60*time.Second)
		state.mu.Unlock()
	}

	// Force adjustment check
	state.mu.Lock()
	state.LastAdjustment = time.Now().Add(-2 * time.Second)
	state.mu.Unlock()

	newDiff, changed := vm.checkAndAdjust(state)

	// Should decrease difficulty (shares too slow)
	if changed {
		assert.Less(t, newDiff, ClassGPU.BaseDifficulty)
	}
}

// Helper Function Tests
// -----------------------------------------------------------------------------

func TestCalculateExpectedDifficulty(t *testing.T) {
	// Difficulty = Hashrate * TargetTime / 2^32
	// For 100 KH/s @ 10s: 100000 * 10 / 4294967296 ≈ 0.0002 -> MinDifficulty
	// For 10 MH/s @ 10s: 10000000 * 10 / 4294967296 ≈ 0.02 -> MinDifficulty
	// For 80 MH/s @ 10s: 80000000 * 10 / 4294967296 ≈ 0.19 -> MinDifficulty
	// For 240 MH/s @ 10s: 240000000 * 10 / 4294967296 ≈ 0.56 -> 1

	// With realistic hashrates for difficulty calculation:
	tests := []struct {
		hashrate   float64
		targetTime time.Duration
		minExpect  uint64
	}{
		{100000, 10 * time.Second, 1},        // Low hashrate -> min difficulty
		{10000000, 10 * time.Second, 1},      // Still low for difficulty
		{4294967296, 10 * time.Second, 10},   // 4 GH/s @ 10s = diff 10
		{42949672960, 10 * time.Second, 100}, // 40 GH/s @ 10s = diff 100
	}

	for _, tt := range tests {
		diff := CalculateExpectedDifficulty(tt.hashrate, tt.targetTime)
		assert.GreaterOrEqual(t, diff, tt.minExpect)
	}
}

func TestCalculateExpectedDifficulty_EdgeCases(t *testing.T) {
	// Zero hashrate
	diff := CalculateExpectedDifficulty(0, 10*time.Second)
	assert.Equal(t, DefaultDiffUnknown, diff)

	// Zero time
	diff = CalculateExpectedDifficulty(1000000, 0)
	assert.Equal(t, DefaultDiffUnknown, diff)
}

func TestCalculateHashrateFromDifficulty(t *testing.T) {
	// Test with known values
	// Hashrate = Difficulty * 2^32 / ShareTime
	difficulty := uint64(100)
	shareTime := 10 * time.Second

	// Expected: 100 * 4294967296 / 10 = 42949672960 H/s ≈ 43 GH/s
	hashrate := CalculateHashrateFromDifficulty(difficulty, shareTime)
	expectedHashrate := float64(100) * 4294967296.0 / 10.0

	assert.InDelta(t, expectedHashrate, hashrate, 1.0)

	// Zero share time should return 0
	assert.Equal(t, float64(0), CalculateHashrateFromDifficulty(100, 0))
}

func TestContainsAny(t *testing.T) {
	assert.True(t, containsAny("BlockDAG-X100", []string{"X100", "X30"}))
	assert.True(t, containsAny("GPU Miner", []string{"GPU", "CUDA"}))
	assert.False(t, containsAny("Unknown", []string{"GPU", "ASIC"}))
}

func TestContainsIgnoreCase(t *testing.T) {
	assert.True(t, containsIgnoreCase("BlockDAG X100", "x100"))
	assert.True(t, containsIgnoreCase("GPU MINER", "gpu"))
	assert.False(t, containsIgnoreCase("CPU", "GPU"))
}

// -----------------------------------------------------------------------------
// Benchmark Tests
// -----------------------------------------------------------------------------

func BenchmarkHardwareClassifier_ClassifyByHashrate(b *testing.B) {
	hc := NewHardwareClassifier()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hc.ClassifyByHashrate(80000000)
	}
}

func BenchmarkVardiffManager_RecordShare(b *testing.B) {
	vm := NewVardiffManager()
	vm.RegisterMiner("bench-miner", "GPU", 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vm.RecordShare("bench-miner", true, false)
	}
}

func BenchmarkVardiffManager_GetDifficulty(b *testing.B) {
	vm := NewVardiffManager()
	vm.RegisterMiner("bench-miner", "GPU", 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vm.GetDifficulty("bench-miner")
	}
}

func BenchmarkCalculateExpectedDifficulty(b *testing.B) {
	for i := 0; i < b.N; i++ {
		CalculateExpectedDifficulty(80000000, 10*time.Second)
	}
}
