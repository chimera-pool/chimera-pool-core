package payouts

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPPLNSCalculator_NewCalculator tests PPLNS calculator creation
func TestPPLNSCalculator_NewCalculator(t *testing.T) {
	tests := []struct {
		name           string
		windowSize     int64
		poolFeePercent float64
		expectError    bool
	}{
		{
			name:           "valid configuration",
			windowSize:     1000000,
			poolFeePercent: 1.0,
			expectError:    false,
		},
		{
			name:           "zero window size should fail",
			windowSize:     0,
			poolFeePercent: 1.0,
			expectError:    true,
		},
		{
			name:           "negative window size should fail",
			windowSize:     -1000,
			poolFeePercent: 1.0,
			expectError:    true,
		},
		{
			name:           "negative pool fee should fail",
			windowSize:     1000000,
			poolFeePercent: -1.0,
			expectError:    true,
		},
		{
			name:           "pool fee over 100% should fail",
			windowSize:     1000000,
			poolFeePercent: 101.0,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calculator, err := NewPPLNSCalculator(tt.windowSize, tt.poolFeePercent)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, calculator)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, calculator)
				assert.Equal(t, tt.windowSize, calculator.windowSize)
				assert.Equal(t, tt.poolFeePercent, calculator.poolFeePercent)
			}
		})
	}
}

// TestPPLNSCalculator_CalculatePayouts tests basic payout calculation
func TestPPLNSCalculator_CalculatePayouts(t *testing.T) {
	calculator, err := NewPPLNSCalculator(1000, 1.0) // 1000 difficulty window, 1% pool fee
	require.NoError(t, err)

	// Test case: Simple scenario with 3 miners
	shares := []Share{
		{UserID: 1, Difficulty: 100, Timestamp: time.Now().Add(-10 * time.Minute), IsValid: true},
		{UserID: 2, Difficulty: 200, Timestamp: time.Now().Add(-9 * time.Minute), IsValid: true},
		{UserID: 3, Difficulty: 300, Timestamp: time.Now().Add(-8 * time.Minute), IsValid: true},
		{UserID: 1, Difficulty: 150, Timestamp: time.Now().Add(-7 * time.Minute), IsValid: true},
		{UserID: 2, Difficulty: 250, Timestamp: time.Now().Add(-6 * time.Minute), IsValid: true},
	}

	blockReward := int64(5000000000) // 50 coins in satoshis
	blockTime := time.Now()

	payouts, err := calculator.CalculatePayouts(shares, blockReward, 0, blockTime)
	require.NoError(t, err)
	require.Len(t, payouts, 3) // 3 unique miners

	// Verify total payout equals block reward minus pool fee
	expectedTotal := int64(float64(blockReward) * (100.0 - calculator.GetPoolFeePercent()) / 100.0)
	actualTotal := int64(0)
	for _, payout := range payouts {
		actualTotal += payout.Amount
	}
	assert.Equal(t, expectedTotal, actualTotal)

	// Verify payouts are proportional to difficulty contributions
	// User 1: 250 difficulty (100+150), User 2: 450 difficulty (200+250), User 3: 300 difficulty
	totalDifficulty := float64(250 + 450 + 300) // 1000

	user1Payout := findPayoutForUser(payouts, 1)
	user2Payout := findPayoutForUser(payouts, 2)
	user3Payout := findPayoutForUser(payouts, 3)

	require.NotNil(t, user1Payout)
	require.NotNil(t, user2Payout)
	require.NotNil(t, user3Payout)

	// Check proportional distribution
	expectedUser1 := int64(float64(expectedTotal) * 250.0 / totalDifficulty)
	expectedUser2 := int64(float64(expectedTotal) * 450.0 / totalDifficulty)
	expectedUser3 := int64(float64(expectedTotal) * 300.0 / totalDifficulty)

	assert.Equal(t, expectedUser1, user1Payout.Amount)
	assert.Equal(t, expectedUser2, user2Payout.Amount)
	assert.Equal(t, expectedUser3, user3Payout.Amount)
}

// TestPPLNSCalculator_SlidingWindow tests the sliding window functionality
func TestPPLNSCalculator_SlidingWindow(t *testing.T) {
	calculator, err := NewPPLNSCalculator(500, 2.0) // 500 difficulty window, 2% pool fee
	require.NoError(t, err)

	// Create shares that exceed the window size
	shares := []Share{
		{UserID: 1, Difficulty: 200, Timestamp: time.Now().Add(-20 * time.Minute), IsValid: true},
		{UserID: 2, Difficulty: 200, Timestamp: time.Now().Add(-15 * time.Minute), IsValid: true},
		{UserID: 3, Difficulty: 200, Timestamp: time.Now().Add(-10 * time.Minute), IsValid: true}, // Total: 600, exceeds window
		{UserID: 1, Difficulty: 100, Timestamp: time.Now().Add(-5 * time.Minute), IsValid: true},
		{UserID: 2, Difficulty: 100, Timestamp: time.Now().Add(-2 * time.Minute), IsValid: true},
	}

	blockReward := int64(1000000000) // 10 coins
	blockTime := time.Now()

	payouts, err := calculator.CalculatePayouts(shares, blockReward, 0, blockTime)
	require.NoError(t, err)

	// Should only include the last 500 difficulty worth of shares
	// Starting from the most recent: 100 + 100 + 200 + 100 (from the 200 share) = 500
	// So user1: 100+100=200, user2: 100, user3: 100 (partial from 200 difficulty share)

	expectedTotal := int64(float64(blockReward) * (100.0 - calculator.GetPoolFeePercent()) / 100.0)

	actualTotal := int64(0)
	for _, payout := range payouts {
		actualTotal += payout.Amount
	}
	assert.Equal(t, expectedTotal, actualTotal)
}

// TestPPLNSCalculator_InvalidShares tests handling of invalid shares
func TestPPLNSCalculator_InvalidShares(t *testing.T) {
	calculator, err := NewPPLNSCalculator(1000, 1.5)
	require.NoError(t, err)

	shares := []Share{
		{UserID: 1, Difficulty: 100, Timestamp: time.Now().Add(-10 * time.Minute), IsValid: true},
		{UserID: 1, Difficulty: 200, Timestamp: time.Now().Add(-9 * time.Minute), IsValid: false}, // Invalid
		{UserID: 2, Difficulty: 300, Timestamp: time.Now().Add(-8 * time.Minute), IsValid: true},
		{UserID: 2, Difficulty: 150, Timestamp: time.Now().Add(-7 * time.Minute), IsValid: false}, // Invalid
	}

	blockReward := int64(2000000000)
	blockTime := time.Now()

	payouts, err := calculator.CalculatePayouts(shares, blockReward, 0, blockTime)
	require.NoError(t, err)

	// Should only count valid shares: User1: 100, User2: 300
	totalValidDifficulty := float64(400)
	expectedTotal := int64(float64(blockReward) * (100.0 - calculator.GetPoolFeePercent()) / 100.0)

	user1Payout := findPayoutForUser(payouts, 1)
	user2Payout := findPayoutForUser(payouts, 2)

	require.NotNil(t, user1Payout)
	require.NotNil(t, user2Payout)

	expectedUser1 := int64(float64(expectedTotal) * 100.0 / totalValidDifficulty)
	expectedUser2 := int64(float64(expectedTotal) * 300.0 / totalValidDifficulty)

	assert.Equal(t, expectedUser1, user1Payout.Amount)
	assert.Equal(t, expectedUser2, user2Payout.Amount)
}

// TestPPLNSCalculator_EmptyShares tests handling of empty share list
func TestPPLNSCalculator_EmptyShares(t *testing.T) {
	calculator, err := NewPPLNSCalculator(1000, 1.0)
	require.NoError(t, err)

	shares := []Share{}
	blockReward := int64(1000000000)
	blockTime := time.Now()

	payouts, err := calculator.CalculatePayouts(shares, blockReward, 0, blockTime)
	require.NoError(t, err)
	assert.Empty(t, payouts)
}

// TestPPLNSCalculator_ZeroBlockReward tests handling of zero block reward
func TestPPLNSCalculator_ZeroBlockReward(t *testing.T) {
	calculator, err := NewPPLNSCalculator(1000, 1.0)
	require.NoError(t, err)

	shares := []Share{
		{UserID: 1, Difficulty: 100, Timestamp: time.Now(), IsValid: true},
	}
	blockReward := int64(0)
	blockTime := time.Now()

	payouts, err := calculator.CalculatePayouts(shares, blockReward, 0, blockTime)
	require.NoError(t, err)
	assert.Empty(t, payouts)
}

// TestPPLNSCalculator_PoolFeeCalculation tests pool fee calculation accuracy
func TestPPLNSCalculator_PoolFeeCalculation(t *testing.T) {
	tests := []struct {
		name           string
		poolFeePercent float64
		blockReward    int64
		expectedFee    int64
	}{
		{
			name:           "1% pool fee",
			poolFeePercent: 1.0,
			blockReward:    10000000000, // 100 coins
			expectedFee:    100000000,   // 1 coin
		},
		{
			name:           "2.5% pool fee",
			poolFeePercent: 2.5,
			blockReward:    10000000000, // 100 coins
			expectedFee:    250000000,   // 2.5 coins
		},
		{
			name:           "0% pool fee",
			poolFeePercent: 0.0,
			blockReward:    10000000000, // 100 coins
			expectedFee:    0,           // 0 coins
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calculator, err := NewPPLNSCalculator(1000, tt.poolFeePercent)
			require.NoError(t, err)

			shares := []Share{
				{UserID: 1, Difficulty: 500, Timestamp: time.Now(), IsValid: true},
			}

			payouts, err := calculator.CalculatePayouts(shares, tt.blockReward, 0, time.Now())
			require.NoError(t, err)
			require.Len(t, payouts, 1)

			expectedPayout := tt.blockReward - tt.expectedFee
			assert.Equal(t, expectedPayout, payouts[0].Amount)
		})
	}
}

// TestPPLNSCalculator_WithTxFees tests PPLNS calculation including transaction fees
func TestPPLNSCalculator_WithTxFees(t *testing.T) {
	calculator, err := NewPPLNSCalculator(1000, 1.0) // 1% fee
	require.NoError(t, err)

	shares := []Share{
		{UserID: 1, Difficulty: 500, Timestamp: time.Now(), IsValid: true},
		{UserID: 2, Difficulty: 500, Timestamp: time.Now(), IsValid: true},
	}

	blockReward := int64(1000000000) // 10 coins
	txFees := int64(100000000)       // 1 coin

	payouts, err := calculator.CalculatePayouts(shares, blockReward, txFees, time.Now())
	require.NoError(t, err)
	require.Len(t, payouts, 2)

	// Total reward = 11 coins, minus 1% fee = 10.89 coins
	totalReward := blockReward + txFees
	expectedTotal := int64(float64(totalReward) * 0.99)

	actualTotal := int64(0)
	for _, p := range payouts {
		actualTotal += p.Amount
	}
	assert.Equal(t, expectedTotal, actualTotal)

	// Each user should get half
	for _, p := range payouts {
		assert.Equal(t, expectedTotal/2, p.Amount)
	}
}

// TestPPLNSCalculator_Mode tests that Mode returns correct payout mode
func TestPPLNSCalculator_Mode(t *testing.T) {
	calculator, err := NewPPLNSCalculator(1000, 1.0)
	require.NoError(t, err)
	assert.Equal(t, PayoutModePPLNS, calculator.Mode())
}

// TestPPLNSCalculator_SetPoolFeePercent tests setting pool fee
func TestPPLNSCalculator_SetPoolFeePercent(t *testing.T) {
	calculator, err := NewPPLNSCalculator(1000, 1.0)
	require.NoError(t, err)

	err = calculator.SetPoolFeePercent(2.5)
	require.NoError(t, err)
	assert.Equal(t, 2.5, calculator.GetPoolFeePercent())

	err = calculator.SetPoolFeePercent(-1.0)
	assert.Error(t, err)

	err = calculator.SetPoolFeePercent(101.0)
	assert.Error(t, err)
}

// TestPPLNSCalculator_SetWindowSize tests setting window size
func TestPPLNSCalculator_SetWindowSize(t *testing.T) {
	calculator, err := NewPPLNSCalculator(1000, 1.0)
	require.NoError(t, err)

	err = calculator.SetWindowSize(2000)
	require.NoError(t, err)
	assert.Equal(t, int64(2000), calculator.GetWindowSize())

	err = calculator.SetWindowSize(0)
	assert.Error(t, err)

	err = calculator.SetWindowSize(-100)
	assert.Error(t, err)
}

// Helper function to find payout for a specific user
func findPayoutForUser(payouts []Payout, userID int64) *Payout {
	for _, payout := range payouts {
		if payout.UserID == userID {
			return &payout
		}
	}
	return nil
}
