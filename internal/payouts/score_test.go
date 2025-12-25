package payouts

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// SCORE CALCULATOR TESTS
// =============================================================================

func TestNewSCORECalculator(t *testing.T) {
	t.Run("valid creation", func(t *testing.T) {
		calc, err := NewSCORECalculator(100000, 1.0, 0.5)
		require.NoError(t, err)
		assert.NotNil(t, calc)
		assert.Equal(t, PayoutModeSCORE, calc.Mode())
		assert.Equal(t, int64(100000), calc.GetWindowSize())
		assert.Equal(t, 1.0, calc.GetPoolFeePercent())
		assert.Equal(t, 0.5, calc.GetDecayFactor())
	})

	t.Run("invalid window size", func(t *testing.T) {
		_, err := NewSCORECalculator(0, 1.0, 0.5)
		assert.Error(t, err)
	})

	t.Run("invalid fee", func(t *testing.T) {
		_, err := NewSCORECalculator(100000, 101.0, 0.5)
		assert.Error(t, err)
	})

	t.Run("invalid decay factor zero", func(t *testing.T) {
		_, err := NewSCORECalculator(100000, 1.0, 0.0)
		assert.Error(t, err)
	})

	t.Run("invalid decay factor over 1", func(t *testing.T) {
		_, err := NewSCORECalculator(100000, 1.0, 1.5)
		assert.Error(t, err)
	})
}

func TestSCORECalculator_CalculatePayouts(t *testing.T) {
	calc, _ := NewSCORECalculator(100000, 1.0, 0.5) // 50% decay per hour

	blockTime := time.Now()
	blockReward := int64(1250000000) // 12.5 LTC
	txFees := int64(50000000)        // 0.5 LTC

	t.Run("recent shares worth more than old shares", func(t *testing.T) {
		shares := []Share{
			// User 1: Recent share (high weight)
			{ID: 1, UserID: 1, Difficulty: 1000, IsValid: true, Timestamp: blockTime.Add(-10 * time.Minute)},
			// User 2: Old share (low weight due to decay)
			{ID: 2, UserID: 2, Difficulty: 1000, IsValid: true, Timestamp: blockTime.Add(-2 * time.Hour)},
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
			{ID: 1, UserID: 1, Difficulty: 1000, IsValid: true, Timestamp: blockTime},
			{ID: 2, UserID: 2, Difficulty: 1000, IsValid: false, Timestamp: blockTime},
		}

		payouts, err := calc.CalculatePayouts(shares, blockReward, txFees, blockTime)
		require.NoError(t, err)
		assert.Len(t, payouts, 1)
	})

	t.Run("includes tx fees in reward", func(t *testing.T) {
		shares := []Share{
			{ID: 1, UserID: 1, Difficulty: 1000, IsValid: true, Timestamp: blockTime},
		}

		payoutsWithFees, err := calc.CalculatePayouts(shares, blockReward, txFees, blockTime)
		require.NoError(t, err)

		payoutsNoFees, err := calc.CalculatePayouts(shares, blockReward, 0, blockTime)
		require.NoError(t, err)

		// Payout with tx fees should be higher
		assert.Greater(t, payoutsWithFees[0].Amount, payoutsNoFees[0].Amount)
	})
}

func TestSCORECalculator_SetDecayFactor(t *testing.T) {
	calc, _ := NewSCORECalculator(100000, 1.0, 0.5)

	t.Run("valid decay factor", func(t *testing.T) {
		err := calc.SetDecayFactor(0.7)
		require.NoError(t, err)
		assert.Equal(t, 0.7, calc.GetDecayFactor())
	})

	t.Run("invalid decay factor", func(t *testing.T) {
		err := calc.SetDecayFactor(0.0)
		assert.Error(t, err)

		err = calc.SetDecayFactor(1.5)
		assert.Error(t, err)
	})
}

func TestSCORECalculator_SetWindowSize(t *testing.T) {
	calc, _ := NewSCORECalculator(100000, 1.0, 0.5)

	t.Run("valid window size", func(t *testing.T) {
		err := calc.SetWindowSize(200000)
		require.NoError(t, err)
		assert.Equal(t, int64(200000), calc.GetWindowSize())
	})

	t.Run("invalid window size", func(t *testing.T) {
		err := calc.SetWindowSize(0)
		assert.Error(t, err)
	})
}

// =============================================================================
// SOLO CALCULATOR TESTS
// =============================================================================

func TestNewSOLOCalculator(t *testing.T) {
	t.Run("valid creation", func(t *testing.T) {
		calc, err := NewSOLOCalculator(0.5)
		require.NoError(t, err)
		assert.NotNil(t, calc)
		assert.Equal(t, PayoutModeSOLO, calc.Mode())
		assert.Equal(t, 0.5, calc.GetPoolFeePercent())
	})

	t.Run("invalid fee", func(t *testing.T) {
		_, err := NewSOLOCalculator(101.0)
		assert.Error(t, err)
	})
}

func TestSOLOCalculator_CalculatePayouts(t *testing.T) {
	calc, _ := NewSOLOCalculator(0.5) // 0.5% fee

	blockTime := time.Now()
	blockReward := int64(1250000000) // 12.5 LTC
	txFees := int64(50000000)        // 0.5 LTC

	t.Run("full reward goes to finder", func(t *testing.T) {
		shares := []Share{
			{ID: 1, UserID: 1, Difficulty: 1000, IsValid: true, Timestamp: blockTime.Add(-time.Minute)},
			{ID: 2, UserID: 1, Difficulty: 500, IsValid: true, Timestamp: blockTime.Add(-30 * time.Second)},
			{ID: 3, UserID: 2, Difficulty: 100, IsValid: true, Timestamp: blockTime.Add(-10 * time.Second)}, // Block finder
		}

		payouts, err := calc.CalculatePayouts(shares, blockReward, txFees, blockTime)
		require.NoError(t, err)
		assert.Len(t, payouts, 1)

		// Total reward minus 0.5% fee
		expectedReward := int64(float64(blockReward+txFees) * 0.995)
		assert.Equal(t, expectedReward, payouts[0].Amount)
	})

	t.Run("empty shares returns empty payouts", func(t *testing.T) {
		payouts, err := calc.CalculatePayouts([]Share{}, blockReward, txFees, blockTime)
		require.NoError(t, err)
		assert.Empty(t, payouts)
	})

	t.Run("all invalid shares returns empty", func(t *testing.T) {
		shares := []Share{
			{ID: 1, UserID: 1, Difficulty: 1000, IsValid: false, Timestamp: blockTime},
		}

		payouts, err := calc.CalculatePayouts(shares, blockReward, txFees, blockTime)
		require.NoError(t, err)
		assert.Empty(t, payouts)
	})
}

func TestSOLOCalculator_ValidateConfiguration(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		calc, _ := NewSOLOCalculator(0.5)
		assert.NoError(t, calc.ValidateConfiguration())
	})
}
