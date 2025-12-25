package payouts

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// PPS CALCULATOR TESTS
// =============================================================================

func TestNewPPSCalculator(t *testing.T) {
	t.Run("valid creation", func(t *testing.T) {
		calc, err := NewPPSCalculator(2.0)
		require.NoError(t, err)
		assert.NotNil(t, calc)
		assert.Equal(t, 2.0, calc.GetPoolFeePercent())
		assert.Equal(t, PayoutModePPS, calc.Mode())
	})

	t.Run("invalid fee too high", func(t *testing.T) {
		_, err := NewPPSCalculator(101.0)
		assert.Error(t, err)
	})

	t.Run("invalid fee negative", func(t *testing.T) {
		_, err := NewPPSCalculator(-1.0)
		assert.Error(t, err)
	})

	t.Run("zero fee valid", func(t *testing.T) {
		calc, err := NewPPSCalculator(0.0)
		require.NoError(t, err)
		assert.Equal(t, 0.0, calc.GetPoolFeePercent())
	})
}

func TestPPSCalculator_CalculatePayouts(t *testing.T) {
	calc, _ := NewPPSCalculator(2.0)
	calc.SetNetworkDifficulty(1000000) // 1M network difficulty

	blockTime := time.Now()
	blockReward := int64(1250000000) // 12.5 LTC in litoshis

	t.Run("basic payout calculation", func(t *testing.T) {
		shares := []Share{
			{ID: 1, UserID: 1, Difficulty: 1000, IsValid: true, Timestamp: blockTime.Add(-time.Minute)},
			{ID: 2, UserID: 1, Difficulty: 1000, IsValid: true, Timestamp: blockTime.Add(-30 * time.Second)},
			{ID: 3, UserID: 2, Difficulty: 500, IsValid: true, Timestamp: blockTime.Add(-20 * time.Second)},
		}

		payouts, err := calc.CalculatePayouts(shares, blockReward, 0, blockTime)
		require.NoError(t, err)
		assert.Len(t, payouts, 2)

		// Verify payouts exist for both users
		var user1Payout, user2Payout int64
		for _, p := range payouts {
			if p.UserID == 1 {
				user1Payout = p.Amount
			} else if p.UserID == 2 {
				user2Payout = p.Amount
			}
		}

		// User1 has 2000 difficulty (1000+1000), User2 has 500, so 4x ratio
		assert.Greater(t, user1Payout, user2Payout)
		assert.InDelta(t, 4.0, float64(user1Payout)/float64(user2Payout), 0.01)
	})

	t.Run("empty shares returns empty payouts", func(t *testing.T) {
		payouts, err := calc.CalculatePayouts([]Share{}, blockReward, 0, blockTime)
		require.NoError(t, err)
		assert.Empty(t, payouts)
	})

	t.Run("invalid shares are skipped", func(t *testing.T) {
		shares := []Share{
			{ID: 1, UserID: 1, Difficulty: 1000, IsValid: true, Timestamp: blockTime},
			{ID: 2, UserID: 2, Difficulty: 1000, IsValid: false, Timestamp: blockTime}, // Invalid
		}

		payouts, err := calc.CalculatePayouts(shares, blockReward, 0, blockTime)
		require.NoError(t, err)
		assert.Len(t, payouts, 1)
		assert.Equal(t, int64(1), payouts[0].UserID)
	})

	t.Run("error on zero network difficulty", func(t *testing.T) {
		badCalc, _ := NewPPSCalculator(1.0)
		badCalc.SetNetworkDifficulty(0)

		_, err := badCalc.CalculatePayouts([]Share{{IsValid: true, Difficulty: 100}}, blockReward, 0, blockTime)
		assert.Error(t, err)
	})
}

func TestPPSCalculator_CalculateShareValue(t *testing.T) {
	calc, _ := NewPPSCalculator(2.0)
	calc.SetNetworkDifficulty(1000000)

	blockReward := int64(1250000000) // 12.5 LTC

	t.Run("calculates correct share value", func(t *testing.T) {
		// Expected: (1250000000 / 1000000) * 0.98 * 1000 = 1225000
		value := calc.CalculateShareValue(1000, blockReward)
		assert.Equal(t, int64(1225000), value)
	})

	t.Run("zero difficulty returns zero", func(t *testing.T) {
		value := calc.CalculateShareValue(0, blockReward)
		assert.Equal(t, int64(0), value)
	})
}

func TestPPSCalculator_SetPoolFeePercent(t *testing.T) {
	calc, _ := NewPPSCalculator(1.0)

	t.Run("valid fee update", func(t *testing.T) {
		err := calc.SetPoolFeePercent(1.5)
		require.NoError(t, err)
		assert.Equal(t, 1.5, calc.GetPoolFeePercent())
	})

	t.Run("invalid fee rejected", func(t *testing.T) {
		err := calc.SetPoolFeePercent(101.0)
		assert.Error(t, err)
	})
}

// =============================================================================
// PPS+ CALCULATOR TESTS
// =============================================================================

func TestNewPPSPlusCalculator(t *testing.T) {
	t.Run("valid creation", func(t *testing.T) {
		calc, err := NewPPSPlusCalculator(1.5, 100000)
		require.NoError(t, err)
		assert.NotNil(t, calc)
		assert.Equal(t, 1.5, calc.GetPoolFeePercent())
		assert.Equal(t, PayoutModePPSPlus, calc.Mode())
		assert.Equal(t, int64(100000), calc.GetWindowSize())
	})

	t.Run("invalid fee", func(t *testing.T) {
		_, err := NewPPSPlusCalculator(101.0, 100000)
		assert.Error(t, err)
	})

	t.Run("invalid window size", func(t *testing.T) {
		_, err := NewPPSPlusCalculator(1.5, 0)
		assert.Error(t, err)
	})
}

func TestPPSPlusCalculator_CalculatePayouts(t *testing.T) {
	calc, _ := NewPPSPlusCalculator(1.5, 100000)
	calc.SetNetworkDifficulty(1000000)

	blockTime := time.Now()
	blockReward := int64(1250000000) // 12.5 LTC
	txFees := int64(50000000)        // 0.5 LTC in tx fees

	t.Run("splits block reward and tx fees", func(t *testing.T) {
		shares := []Share{
			{ID: 1, UserID: 1, Difficulty: 1000, IsValid: true, Timestamp: blockTime.Add(-time.Minute)},
			{ID: 2, UserID: 2, Difficulty: 1000, IsValid: true, Timestamp: blockTime.Add(-30 * time.Second)},
		}

		payouts, err := calc.CalculatePayouts(shares, blockReward, txFees, blockTime)
		require.NoError(t, err)
		assert.Len(t, payouts, 2)

		// Both users should get payouts
		for _, p := range payouts {
			assert.Greater(t, p.Amount, int64(0))
		}
	})

	t.Run("handles zero tx fees", func(t *testing.T) {
		shares := []Share{
			{ID: 1, UserID: 1, Difficulty: 1000, IsValid: true, Timestamp: blockTime},
		}

		payouts, err := calc.CalculatePayouts(shares, blockReward, 0, blockTime)
		require.NoError(t, err)
		assert.Len(t, payouts, 1)
	})
}

// =============================================================================
// FPPS CALCULATOR TESTS
// =============================================================================

func TestNewFPPSCalculator(t *testing.T) {
	t.Run("valid creation", func(t *testing.T) {
		calc, err := NewFPPSCalculator(2.0)
		require.NoError(t, err)
		assert.NotNil(t, calc)
		assert.Equal(t, 2.0, calc.GetPoolFeePercent())
		assert.Equal(t, PayoutModeFPPS, calc.Mode())
	})

	t.Run("invalid fee", func(t *testing.T) {
		_, err := NewFPPSCalculator(-1.0)
		assert.Error(t, err)
	})
}

func TestFPPSCalculator_CalculatePayouts(t *testing.T) {
	calc, _ := NewFPPSCalculator(2.0)
	calc.SetNetworkDifficulty(1000000)
	calc.SetExpectedTxFees(50000000) // 0.5 LTC expected

	blockTime := time.Now()
	blockReward := int64(1250000000) // 12.5 LTC

	t.Run("includes expected tx fees", func(t *testing.T) {
		shares := []Share{
			{ID: 1, UserID: 1, Difficulty: 1000, IsValid: true, Timestamp: blockTime},
		}

		payouts, err := calc.CalculatePayouts(shares, blockReward, 0, blockTime)
		require.NoError(t, err)
		assert.Len(t, payouts, 1)

		// Should include tx fees in calculation
		// (blockReward + txFees) / networkDiff * shareDiff * (1 - fee)
		// (1250000000 + 50000000) / 1000000 * 1000 * 0.98 = 1274000
		assert.Equal(t, int64(1274000), payouts[0].Amount)
	})

	t.Run("uses provided tx fees over expected", func(t *testing.T) {
		shares := []Share{
			{ID: 1, UserID: 1, Difficulty: 1000, IsValid: true, Timestamp: blockTime},
		}

		// Provide actual tx fees instead of using expected
		payouts, err := calc.CalculatePayouts(shares, blockReward, 100000000, blockTime)
		require.NoError(t, err)
		assert.Len(t, payouts, 1)

		// Should use provided 1 LTC tx fees
		// (1250000000 + 100000000) / 1000000 * 1000 * 0.98 = 1323000
		assert.Equal(t, int64(1323000), payouts[0].Amount)
	})
}

func TestFPPSCalculator_CalculateShareValue(t *testing.T) {
	calc, _ := NewFPPSCalculator(2.0)
	calc.SetNetworkDifficulty(1000000)
	calc.SetExpectedTxFees(50000000)

	blockReward := int64(1250000000)

	t.Run("includes tx fees in share value", func(t *testing.T) {
		value := calc.CalculateShareValue(1000, blockReward)
		// (1250000000 + 50000000) / 1000000 * 1000 * 0.98 = 1274000
		assert.Equal(t, int64(1274000), value)
	})
}
