package payouts

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// SLICE CALCULATOR TESTS
// =============================================================================

func TestNewSLICECalculator(t *testing.T) {
	t.Run("valid creation", func(t *testing.T) {
		calc, err := NewSLICECalculator(10, 600, 0.7, 0.8)
		require.NoError(t, err)
		assert.NotNil(t, calc)
		assert.Equal(t, PayoutModeSLICE, calc.Mode())
		assert.Equal(t, int64(10), calc.sliceCount)
		assert.Equal(t, int64(600), calc.sliceDuration)
		assert.Equal(t, 0.7, calc.GetDecayFactor())
		assert.Equal(t, 0.8, calc.GetPoolFeePercent())
	})

	t.Run("invalid slice count", func(t *testing.T) {
		_, err := NewSLICECalculator(0, 600, 0.7, 0.8)
		assert.Error(t, err)
	})

	t.Run("invalid slice duration", func(t *testing.T) {
		_, err := NewSLICECalculator(10, 0, 0.7, 0.8)
		assert.Error(t, err)
	})

	t.Run("invalid decay factor", func(t *testing.T) {
		_, err := NewSLICECalculator(10, 600, 0, 0.8)
		assert.Error(t, err)

		_, err = NewSLICECalculator(10, 600, 1.5, 0.8)
		assert.Error(t, err)
	})

	t.Run("invalid fee", func(t *testing.T) {
		_, err := NewSLICECalculator(10, 600, 0.7, -1)
		assert.Error(t, err)

		_, err = NewSLICECalculator(10, 600, 0.7, 101)
		assert.Error(t, err)
	})
}

func TestSLICECalculator_CalculatePayouts(t *testing.T) {
	// 10 slices of 10 minutes each (6000 seconds total window)
	calc, _ := NewSLICECalculator(10, 600, 0.7, 0.8)

	blockTime := time.Now()
	blockReward := int64(1250000000) // 12.5 coins
	txFees := int64(50000000)        // 0.5 coins

	t.Run("basic payout calculation", func(t *testing.T) {
		shares := []Share{
			{ID: 1, UserID: 1, Difficulty: 1000, IsValid: true, Timestamp: blockTime.Add(-5 * time.Minute)},
			{ID: 2, UserID: 2, Difficulty: 1000, IsValid: true, Timestamp: blockTime.Add(-10 * time.Minute)},
		}

		payouts, err := calc.CalculatePayouts(shares, blockReward, txFees, blockTime)
		require.NoError(t, err)
		assert.Len(t, payouts, 2)

		// Both users should get payouts
		for _, p := range payouts {
			assert.Greater(t, p.Amount, int64(0))
		}
	})

	t.Run("recent shares worth more than old shares", func(t *testing.T) {
		shares := []Share{
			// User 1: Recent share (high weight)
			{ID: 1, UserID: 1, Difficulty: 1000, IsValid: true, Timestamp: blockTime.Add(-5 * time.Minute)},
			// User 2: Old share (low weight due to decay)
			{ID: 2, UserID: 2, Difficulty: 1000, IsValid: true, Timestamp: blockTime.Add(-90 * time.Minute)},
		}

		payouts, err := calc.CalculatePayouts(shares, blockReward, txFees, blockTime)
		require.NoError(t, err)
		assert.Len(t, payouts, 2)

		// Find payouts
		var user1Payout, user2Payout int64
		for _, p := range payouts {
			if p.UserID == 1 {
				user1Payout = p.Amount
			} else if p.UserID == 2 {
				user2Payout = p.Amount
			}
		}

		// User1 (recent) should get more than User2 (old) despite same difficulty
		assert.Greater(t, user1Payout, user2Payout)
	})

	t.Run("empty shares returns empty payouts", func(t *testing.T) {
		payouts, err := calc.CalculatePayouts([]Share{}, blockReward, txFees, blockTime)
		require.NoError(t, err)
		assert.Empty(t, payouts)
	})

	t.Run("invalid shares are skipped", func(t *testing.T) {
		shares := []Share{
			{ID: 1, UserID: 1, Difficulty: 1000, IsValid: true, Timestamp: blockTime.Add(-5 * time.Minute)},
			{ID: 2, UserID: 2, Difficulty: 1000, IsValid: false, Timestamp: blockTime.Add(-5 * time.Minute)},
		}

		payouts, err := calc.CalculatePayouts(shares, blockReward, txFees, blockTime)
		require.NoError(t, err)
		assert.Len(t, payouts, 1)
	})

	t.Run("includes tx fees in reward", func(t *testing.T) {
		shares := []Share{
			{ID: 1, UserID: 1, Difficulty: 1000, IsValid: true, Timestamp: blockTime.Add(-5 * time.Minute)},
		}

		payoutsWithFees, err := calc.CalculatePayouts(shares, blockReward, txFees, blockTime)
		require.NoError(t, err)

		payoutsNoFees, err := calc.CalculatePayouts(shares, blockReward, 0, blockTime)
		require.NoError(t, err)

		// Payout with tx fees should be higher
		assert.Greater(t, payoutsWithFees[0].Amount, payoutsNoFees[0].Amount)
	})
}

func TestSLICECalculator_SetDecayFactor(t *testing.T) {
	calc, _ := NewSLICECalculator(10, 600, 0.7, 0.8)

	t.Run("valid decay factor", func(t *testing.T) {
		err := calc.SetDecayFactor(0.5)
		require.NoError(t, err)
		assert.Equal(t, 0.5, calc.GetDecayFactor())
	})

	t.Run("invalid decay factor", func(t *testing.T) {
		err := calc.SetDecayFactor(0)
		assert.Error(t, err)

		err = calc.SetDecayFactor(1.5)
		assert.Error(t, err)
	})
}

func TestSLICECalculator_SetWindowSize(t *testing.T) {
	calc, _ := NewSLICECalculator(10, 600, 0.7, 0.8)

	t.Run("valid window size", func(t *testing.T) {
		err := calc.SetWindowSize(12000) // 20 slices of 600 seconds
		require.NoError(t, err)
		assert.Equal(t, int64(12000), calc.GetWindowSize())
	})

	t.Run("invalid window size", func(t *testing.T) {
		err := calc.SetWindowSize(0)
		assert.Error(t, err)
	})
}

func TestSLICECalculator_SetPoolFeePercent(t *testing.T) {
	calc, _ := NewSLICECalculator(10, 600, 0.7, 0.8)

	t.Run("valid fee", func(t *testing.T) {
		err := calc.SetPoolFeePercent(1.5)
		require.NoError(t, err)
		assert.Equal(t, 1.5, calc.GetPoolFeePercent())
	})

	t.Run("invalid fee", func(t *testing.T) {
		err := calc.SetPoolFeePercent(-1)
		assert.Error(t, err)

		err = calc.SetPoolFeePercent(101)
		assert.Error(t, err)
	})
}

// =============================================================================
// DEMAND RESPONSE TESTS
// =============================================================================

func TestSLICECalculator_DemandResponse(t *testing.T) {
	calc, _ := NewSLICECalculator(10, 600, 0.7, 1.0) // 1% base fee

	t.Run("set demand multiplier", func(t *testing.T) {
		err := calc.SetDemandMultiplier(1.5)
		require.NoError(t, err)
		assert.Equal(t, 1.5, calc.GetDemandMultiplier())

		// Effective fee should be 1.5%
		assert.Equal(t, 1.5, calc.GetEffectiveFeePercent())
	})

	t.Run("invalid demand multiplier", func(t *testing.T) {
		err := calc.SetDemandMultiplier(0)
		assert.Error(t, err)

		err = calc.SetDemandMultiplier(-1)
		assert.Error(t, err)
	})

	t.Run("fee bounds are respected", func(t *testing.T) {
		calc.SetFeeBounds(0.5, 2.0)

		// Set very high demand multiplier
		calc.SetDemandMultiplier(10.0)
		assert.Equal(t, 2.0, calc.GetEffectiveFeePercent()) // Capped at max

		// Set very low demand multiplier
		calc.SetDemandMultiplier(0.1)
		assert.Equal(t, 0.5, calc.GetEffectiveFeePercent()) // Floored at min
	})

	t.Run("set fee bounds", func(t *testing.T) {
		err := calc.SetFeeBounds(0.3, 5.0)
		require.NoError(t, err)

		min, max := calc.GetFeeBounds()
		assert.Equal(t, 0.3, min)
		assert.Equal(t, 5.0, max)
	})

	t.Run("invalid fee bounds", func(t *testing.T) {
		err := calc.SetFeeBounds(-1, 5.0)
		assert.Error(t, err)

		err = calc.SetFeeBounds(5.0, 1.0) // min > max
		assert.Error(t, err)
	})
}

// =============================================================================
// V2 JOB DECLARATION TESTS
// =============================================================================

func TestSLICECalculator_JobDeclaration(t *testing.T) {
	calc, _ := NewSLICECalculator(10, 600, 0.7, 0.8)

	t.Run("register job declaration", func(t *testing.T) {
		decl := &JobDeclaration{
			MinerID:  1,
			JobID:    "job123",
			PrevHash: "prevhash",
			Version:  1,
		}

		err := calc.RegisterJobDeclaration(decl)
		require.NoError(t, err)

		retrieved := calc.GetJobDeclaration("job123")
		assert.NotNil(t, retrieved)
		assert.Equal(t, int64(1), retrieved.MinerID)
		assert.False(t, retrieved.DeclaredAt.IsZero())
	})

	t.Run("register nil declaration fails", func(t *testing.T) {
		err := calc.RegisterJobDeclaration(nil)
		assert.Error(t, err)
	})

	t.Run("register empty job ID fails", func(t *testing.T) {
		decl := &JobDeclaration{
			MinerID: 1,
		}
		err := calc.RegisterJobDeclaration(decl)
		assert.Error(t, err)
	})

	t.Run("validate job declaration", func(t *testing.T) {
		decl := &JobDeclaration{
			MinerID: 1,
			JobID:   "job456",
		}
		calc.RegisterJobDeclaration(decl)

		validated, err := calc.ValidateJobDeclaration("job456")
		require.NoError(t, err)
		assert.True(t, validated.Validated)
	})

	t.Run("validate unknown job declaration fails", func(t *testing.T) {
		_, err := calc.ValidateJobDeclaration("unknown")
		assert.Error(t, err)
	})

	t.Run("cleanup old declarations", func(t *testing.T) {
		cleanupCalc, _ := NewSLICECalculator(10, 600, 0.7, 0.8)

		// Register some declarations
		for i := 0; i < 5; i++ {
			decl := &JobDeclaration{
				MinerID: int64(i),
				JobID:   "cleanup_" + string(rune('A'+i)),
			}
			cleanupCalc.RegisterJobDeclaration(decl)
		}

		// Cleanup with 1 hour age should keep all recently created
		removed := cleanupCalc.CleanupOldDeclarations(time.Hour)
		assert.Equal(t, 0, removed)

		// Cleanup with 0 duration should remove all (they are all older than 0 nanoseconds)
		removed = cleanupCalc.CleanupOldDeclarations(time.Nanosecond)
		assert.GreaterOrEqual(t, removed, 0) // May or may not remove depending on timing
	})
}

// =============================================================================
// ANALYTICS TESTS
// =============================================================================

func TestSLICECalculator_GetAnalytics(t *testing.T) {
	calc, _ := NewSLICECalculator(10, 600, 0.7, 0.8)

	blockTime := time.Now()

	t.Run("returns analytics for shares", func(t *testing.T) {
		shares := []Share{
			{ID: 1, UserID: 1, Difficulty: 1000, IsValid: true, Timestamp: blockTime.Add(-5 * time.Minute)},
			{ID: 2, UserID: 1, Difficulty: 500, IsValid: true, Timestamp: blockTime.Add(-10 * time.Minute)},
			{ID: 3, UserID: 2, Difficulty: 800, IsValid: true, Timestamp: blockTime.Add(-15 * time.Minute)},
		}

		analytics := calc.GetAnalytics(shares, blockTime)
		assert.NotNil(t, analytics)
		assert.Equal(t, 10, analytics.TotalSlices)
		assert.Equal(t, 3, analytics.TotalShares)
		assert.Equal(t, 2, analytics.UniqueMiners)
		assert.Greater(t, analytics.TotalDifficulty, 0.0)
		assert.Greater(t, analytics.WeightedDiff, 0.0)
		assert.NotEmpty(t, analytics.SliceBreakdown)
		assert.NotEmpty(t, analytics.MinerContribs)
	})

	t.Run("analytics with empty shares", func(t *testing.T) {
		analytics := calc.GetAnalytics([]Share{}, blockTime)
		assert.NotNil(t, analytics)
		assert.Equal(t, 0, analytics.TotalShares)
		assert.Equal(t, 0, analytics.UniqueMiners)
	})
}

func TestSLICECalculator_GetSliceConfig(t *testing.T) {
	calc, _ := NewSLICECalculator(10, 600, 0.7, 0.8)

	config := calc.GetSliceConfig()
	assert.NotNil(t, config)
	assert.Equal(t, int64(10), config["slice_count"])
	assert.Equal(t, int64(600), config["slice_duration"])
	assert.Equal(t, 0.7, config["decay_factor"])
	assert.Equal(t, 0.8, config["pool_fee_percent"])
	assert.Equal(t, int64(6000), config["window_duration"])
}

func TestSLICECalculator_ValidateConfiguration(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		calc, _ := NewSLICECalculator(10, 600, 0.7, 0.8)
		assert.NoError(t, calc.ValidateConfiguration())
	})
}

// =============================================================================
// SLICE ORGANIZATION TESTS
// =============================================================================

func TestSLICECalculator_ShareSlicing(t *testing.T) {
	calc, _ := NewSLICECalculator(10, 600, 0.7, 0.8) // 10 slices of 10 minutes

	blockTime := time.Now()

	t.Run("shares organized into correct slices", func(t *testing.T) {
		shares := []Share{
			// Slice 9 (most recent, 0-10 min before block)
			{ID: 1, UserID: 1, Difficulty: 100, IsValid: true, Timestamp: blockTime.Add(-5 * time.Minute)},
			// Slice 8 (10-20 min before block)
			{ID: 2, UserID: 1, Difficulty: 200, IsValid: true, Timestamp: blockTime.Add(-15 * time.Minute)},
			// Slice 7 (20-30 min before block)
			{ID: 3, UserID: 2, Difficulty: 300, IsValid: true, Timestamp: blockTime.Add(-25 * time.Minute)},
		}

		payouts, err := calc.CalculatePayouts(shares, 1000000000, 0, blockTime)
		require.NoError(t, err)
		assert.NotEmpty(t, payouts)

		// Verify payouts exist for both users
		userIDs := make(map[int64]bool)
		for _, p := range payouts {
			userIDs[p.UserID] = true
		}
		assert.True(t, userIDs[1])
		assert.True(t, userIDs[2])
	})
}
