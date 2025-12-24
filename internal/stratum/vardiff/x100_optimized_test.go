package vardiff

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestX100OptimizedConfig(t *testing.T) {
	config := X100OptimizedConfig()

	t.Run("has correct target share time", func(t *testing.T) {
		assert.Equal(t, 10*time.Second, config.TargetShareTime)
	})

	t.Run("has appropriate initial difficulty for X100 on Scrypt", func(t *testing.T) {
		// X100 on Scrypt ~15 TH/s, target 10s shares
		// Expected: 15e12 * 10 / 4.295e9 ≈ 35,000
		assert.Equal(t, 35000.0, config.InitialDifficulty)
	})

	t.Run("has stable retarget interval", func(t *testing.T) {
		// 3 minutes for maximum stability
		assert.Equal(t, 3*time.Minute, config.RetargetInterval)
	})

	t.Run("has balanced variance for stability", func(t *testing.T) {
		assert.Equal(t, 25.0, config.VariancePercent)
	})

	t.Run("has larger share window for smoothing", func(t *testing.T) {
		assert.Equal(t, 30, config.ShareWindow)
	})

	t.Run("validates successfully", func(t *testing.T) {
		err := config.Validate()
		assert.NoError(t, err)
	})
}

func TestCalculateOptimalDifficulty(t *testing.T) {
	tests := []struct {
		name            string
		hashrate        float64 // H/s
		targetShareTime float64 // seconds
		expectedDiff    float64
		tolerance       float64 // acceptable variance
	}{
		{
			name:            "X100 on Scrypt (15 TH/s)",
			hashrate:        15e12,
			targetShareTime: 10,
			expectedDiff:    34924.6, // 15e12 * 10 / 4.295e9
			tolerance:       100,
		},
		{
			name:            "X100 on BlockDAG Scrpy (70 TH/s)",
			hashrate:        70e12,
			targetShareTime: 10,
			expectedDiff:    162981.4, // 70e12 * 10 / 4.295e9
			tolerance:       500,
		},
		{
			name:            "Low hashrate GPU (100 MH/s)",
			hashrate:        100e6,
			targetShareTime: 10,
			expectedDiff:    0.233, // 100e6 * 10 / 4.295e9
			tolerance:       0.01,
		},
		{
			name:            "High hashrate ASIC (500 TH/s)",
			hashrate:        500e12,
			targetShareTime: 15,
			expectedDiff:    1746237.4, // 500e12 * 15 / 4.295e9
			tolerance:       5000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := CalculateOptimalDifficulty(tt.hashrate, tt.targetShareTime)
			assert.InDelta(t, tt.expectedDiff, diff, tt.tolerance,
				"Expected difficulty ~%.1f, got %.1f", tt.expectedDiff, diff)
		})
	}
}

func TestCalculateExpectedHashrate(t *testing.T) {
	tests := []struct {
		name             string
		difficulty       float64
		shareTime        float64 // seconds
		expectedHashrate float64 // H/s
		tolerance        float64
	}{
		{
			name:             "X100 difficulty with 10s shares",
			difficulty:       35000,
			shareTime:        10,
			expectedHashrate: 15.03e12, // 35000 * 4.295e9 / 10
			tolerance:        0.1e12,
		},
		{
			name:             "High difficulty with 15s shares",
			difficulty:       163000,
			shareTime:        10,
			expectedHashrate: 70e12,
			tolerance:        1e12,
		},
		{
			name:             "Zero share time returns zero",
			difficulty:       35000,
			shareTime:        0,
			expectedHashrate: 0,
			tolerance:        0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hashrate := CalculateExpectedHashrate(tt.difficulty, tt.shareTime)
			assert.InDelta(t, tt.expectedHashrate, hashrate, tt.tolerance)
		})
	}
}

func TestHighHashrateASICConfig(t *testing.T) {
	config := HighHashrateASICConfig()

	t.Run("validates successfully", func(t *testing.T) {
		err := config.Validate()
		assert.NoError(t, err)
	})

	t.Run("has higher initial difficulty", func(t *testing.T) {
		assert.Equal(t, 100000.0, config.InitialDifficulty)
	})

	t.Run("has longer share window", func(t *testing.T) {
		assert.Equal(t, 30, config.ShareWindow)
	})
}

func TestLowLatencyConfig(t *testing.T) {
	config := LowLatencyConfig()

	t.Run("validates successfully", func(t *testing.T) {
		err := config.Validate()
		assert.NoError(t, err)
	})

	t.Run("has shorter target share time", func(t *testing.T) {
		assert.Equal(t, 5*time.Second, config.TargetShareTime)
	})

	t.Run("has more variance tolerance", func(t *testing.T) {
		assert.Equal(t, 40.0, config.VariancePercent)
	})
}

func TestVardiffStability(t *testing.T) {
	// Test that the vardiff algorithm doesn't oscillate wildly
	config := X100OptimizedConfig()
	manager := NewManager(config)

	minerID := "test-x100-miner"

	// Simulate shares coming in at ~10 second intervals
	// The difficulty should stay relatively stable
	initialDiff := manager.GetDifficulty(minerID)

	// Record 20 shares at roughly target time
	for i := 0; i < 20; i++ {
		shareTime := time.Duration(9+i%3) * time.Second // 9-11 seconds
		manager.RecordShare(minerID, shareTime)
	}

	finalDiff := manager.GetDifficulty(minerID)

	// Difficulty should not have changed dramatically (within 50%)
	ratio := finalDiff / initialDiff
	assert.True(t, ratio >= 0.5 && ratio <= 1.5,
		"Difficulty changed too much: initial=%.0f, final=%.0f, ratio=%.2f",
		initialDiff, finalDiff, ratio)
}
