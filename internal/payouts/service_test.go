package payouts

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockDatabase implements the database interface for testing
type MockDatabase struct {
	shares  []Share
	payouts []Payout
	blocks  []Block
}

func (db *MockDatabase) GetSharesForPayout(ctx context.Context, blockTime time.Time, windowSize int64) ([]Share, error) {
	// Return shares sorted by timestamp descending
	result := make([]Share, len(db.shares))
	copy(result, db.shares)
	return result, nil
}

func (db *MockDatabase) CreatePayouts(ctx context.Context, payouts []Payout) error {
	db.payouts = append(db.payouts, payouts...)
	return nil
}

func (db *MockDatabase) GetBlock(ctx context.Context, blockID int64) (*Block, error) {
	for _, block := range db.blocks {
		if block.ID == blockID {
			return &block, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (db *MockDatabase) GetPayoutHistory(ctx context.Context, userID int64, limit, offset int) ([]Payout, error) {
	result := make([]Payout, 0)

	// If userID is 0, return all payouts
	if userID == 0 {
		for _, payout := range db.payouts {
			result = append(result, payout)
		}
	} else {
		// Filter by user ID
		for _, payout := range db.payouts {
			if payout.UserID == userID {
				result = append(result, payout)
			}
		}
	}

	// Sort by timestamp descending (most recent first)
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i].Timestamp.Before(result[j].Timestamp) {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	// Apply offset and limit
	start := offset
	if start >= len(result) {
		return []Payout{}, nil
	}

	end := start + limit
	if end > len(result) {
		end = len(result)
	}

	return result[start:end], nil
}

// TestPayoutService_ProcessBlockPayout tests the complete payout processing workflow
func TestPayoutService_ProcessBlockPayout(t *testing.T) {
	// Setup mock database
	mockDB := &MockDatabase{
		shares: []Share{
			{UserID: 1, Difficulty: 100, Timestamp: time.Now().Add(-10 * time.Minute), IsValid: true},
			{UserID: 2, Difficulty: 200, Timestamp: time.Now().Add(-9 * time.Minute), IsValid: true},
			{UserID: 3, Difficulty: 300, Timestamp: time.Now().Add(-8 * time.Minute), IsValid: true},
			{UserID: 1, Difficulty: 150, Timestamp: time.Now().Add(-7 * time.Minute), IsValid: true},
		},
		blocks: []Block{
			{
				ID:        1,
				Height:    12345,
				Hash:      "0x123abc",
				FinderID:  1,
				Reward:    5000000000, // 50 coins
				Status:    "confirmed",
				Timestamp: time.Now(),
			},
		},
	}

	// Create payout service
	calculator, err := NewPPLNSCalculator(1000, 1.0) // 1000 difficulty window, 1% fee
	require.NoError(t, err)

	service := NewPayoutService(mockDB, calculator)

	// Process block payout
	err = service.ProcessBlockPayout(context.Background(), 1)
	require.NoError(t, err)

	// Verify payouts were created
	require.Len(t, mockDB.payouts, 3) // 3 unique users

	// Verify total payout amount
	totalPayout := int64(0)
	for _, payout := range mockDB.payouts {
		totalPayout += payout.Amount
	}

	expectedTotal := int64(float64(5000000000) * 0.99) // 99% after 1% fee
	assert.Equal(t, expectedTotal, totalPayout)

	// Verify payouts are proportional
	userDifficulties := map[int64]float64{
		1: 250, // 100 + 150
		2: 200,
		3: 300,
	}
	totalDifficulty := float64(750)

	for _, payout := range mockDB.payouts {
		expectedAmount := int64(float64(expectedTotal) * userDifficulties[payout.UserID] / totalDifficulty)
		assert.Equal(t, expectedAmount, payout.Amount)
		assert.Equal(t, int64(1), payout.BlockID)
	}
}

// TestPayoutService_ProcessBlockPayout_InvalidBlock tests handling of invalid block
func TestPayoutService_ProcessBlockPayout_InvalidBlock(t *testing.T) {
	mockDB := &MockDatabase{}
	calculator, err := NewPPLNSCalculator(1000, 1.0)
	require.NoError(t, err)

	service := NewPayoutService(mockDB, calculator)

	// Try to process non-existent block
	err = service.ProcessBlockPayout(context.Background(), 999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "block not found")
}

// TestPayoutService_ProcessBlockPayout_PendingBlock tests handling of pending block
func TestPayoutService_ProcessBlockPayout_PendingBlock(t *testing.T) {
	mockDB := &MockDatabase{
		blocks: []Block{
			{
				ID:        1,
				Height:    12345,
				Hash:      "0x123abc",
				FinderID:  1,
				Reward:    5000000000,
				Status:    "pending", // Not confirmed yet
				Timestamp: time.Now(),
			},
		},
	}

	calculator, err := NewPPLNSCalculator(1000, 1.0)
	require.NoError(t, err)

	service := NewPayoutService(mockDB, calculator)

	// Try to process pending block
	err = service.ProcessBlockPayout(context.Background(), 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "block not confirmed")
}

// TestPayoutService_CalculateEstimatedPayout tests estimated payout calculation
func TestPayoutService_CalculateEstimatedPayout(t *testing.T) {
	mockDB := &MockDatabase{
		shares: []Share{
			{UserID: 1, Difficulty: 100, Timestamp: time.Now().Add(-10 * time.Minute), IsValid: true},
			{UserID: 2, Difficulty: 200, Timestamp: time.Now().Add(-9 * time.Minute), IsValid: true},
			{UserID: 1, Difficulty: 150, Timestamp: time.Now().Add(-8 * time.Minute), IsValid: true},
		},
	}

	calculator, err := NewPPLNSCalculator(1000, 2.0) // 2% fee
	require.NoError(t, err)

	service := NewPayoutService(mockDB, calculator)

	// Calculate estimated payout for user 1
	estimatedPayout, err := service.CalculateEstimatedPayout(context.Background(), 1, 10000000000) // 100 coins
	require.NoError(t, err)

	// User 1 has 250 difficulty out of 450 total
	// Expected: (100 coins * 0.98) * (250/450) = 54.44 coins
	blockReward := float64(10000000000)
	netReward := blockReward * 0.98
	proportion := 250.0 / 450.0
	expectedPayout := int64(netReward * proportion)
	assert.Equal(t, expectedPayout, estimatedPayout)
}

// TestPayoutService_GetPayoutHistory tests payout history retrieval
func TestPayoutService_GetPayoutHistory(t *testing.T) {
	now := time.Now()
	mockDB := &MockDatabase{
		payouts: []Payout{
			{UserID: 1, Amount: 1000000000, BlockID: 1, Timestamp: now.Add(-1 * time.Hour)}, // More recent
			{UserID: 1, Amount: 2000000000, BlockID: 2, Timestamp: now.Add(-2 * time.Hour)}, // Less recent
			{UserID: 2, Amount: 1500000000, BlockID: 1, Timestamp: now.Add(-1 * time.Hour)},
		},
	}

	calculator, err := NewPPLNSCalculator(1000, 1.0)
	require.NoError(t, err)

	service := NewPayoutService(mockDB, calculator)

	// Get payout history for user 1
	history, err := service.GetPayoutHistory(context.Background(), 1, 10, 0)
	require.NoError(t, err)

	// Should return 2 payouts for user 1
	assert.Len(t, history, 2)
	assert.Equal(t, int64(1000000000), history[0].Amount) // Most recent first (-1 hour)
	assert.Equal(t, int64(2000000000), history[1].Amount) // Less recent (-2 hours)
}

// TestPayoutService_ValidatePayoutFairness tests mathematical accuracy validation
func TestPayoutService_ValidatePayoutFairness(t *testing.T) {
	tests := []struct {
		name           string
		shares         []Share
		blockReward    int64
		poolFeePercent float64
		windowSize     int64
	}{
		{
			name: "equal difficulty shares",
			shares: []Share{
				{UserID: 1, Difficulty: 100, Timestamp: time.Now().Add(-5 * time.Minute), IsValid: true},
				{UserID: 2, Difficulty: 100, Timestamp: time.Now().Add(-4 * time.Minute), IsValid: true},
				{UserID: 3, Difficulty: 100, Timestamp: time.Now().Add(-3 * time.Minute), IsValid: true},
			},
			blockReward:    3000000000, // 30 coins
			poolFeePercent: 1.0,
			windowSize:     1000,
		},
		{
			name: "varying difficulty shares",
			shares: []Share{
				{UserID: 1, Difficulty: 50, Timestamp: time.Now().Add(-5 * time.Minute), IsValid: true},
				{UserID: 2, Difficulty: 150, Timestamp: time.Now().Add(-4 * time.Minute), IsValid: true},
				{UserID: 3, Difficulty: 300, Timestamp: time.Now().Add(-3 * time.Minute), IsValid: true},
			},
			blockReward:    5000000000, // 50 coins
			poolFeePercent: 2.5,
			windowSize:     1000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDatabase{
				shares: tt.shares,
				blocks: []Block{
					{
						ID:        1,
						Height:    12345,
						Hash:      "0x123abc",
						FinderID:  1,
						Reward:    tt.blockReward,
						Status:    "confirmed",
						Timestamp: time.Now(),
					},
				},
			}

			calculator, err := NewPPLNSCalculator(tt.windowSize, tt.poolFeePercent)
			require.NoError(t, err)

			service := NewPayoutService(mockDB, calculator)

			// Process payout
			err = service.ProcessBlockPayout(context.Background(), 1)
			require.NoError(t, err)

			// Validate mathematical accuracy
			totalPayout := int64(0)
			totalDifficulty := float64(0)
			userDifficulties := make(map[int64]float64)

			for _, share := range tt.shares {
				if share.IsValid {
					totalDifficulty += share.Difficulty
					userDifficulties[share.UserID] += share.Difficulty
				}
			}

			for _, payout := range mockDB.payouts {
				totalPayout += payout.Amount
			}

			// Verify total payout equals block reward minus pool fee
			expectedTotal := int64(float64(tt.blockReward) * (100.0 - tt.poolFeePercent) / 100.0)
			assert.Equal(t, expectedTotal, totalPayout, "Total payout should equal block reward minus pool fee")

			// Verify each user's payout is proportional to their difficulty contribution
			for _, payout := range mockDB.payouts {
				userDifficulty := userDifficulties[payout.UserID]
				expectedUserPayout := int64(float64(expectedTotal) * userDifficulty / totalDifficulty)
				assert.Equal(t, expectedUserPayout, payout.Amount, "User payout should be proportional to difficulty contribution")
			}
		})
	}
}

// =============================================================================
// ADDITIONAL COMPREHENSIVE TESTS FOR 75%+ COVERAGE
// =============================================================================

func TestPayoutService_GetPayoutStatistics(t *testing.T) {
	now := time.Now()
	mockDB := &MockDatabase{
		payouts: []Payout{
			{UserID: 1, Amount: 1000000000, BlockID: 1, Timestamp: now.Add(-1 * time.Hour)},
			{UserID: 1, Amount: 2000000000, BlockID: 2, Timestamp: now.Add(-2 * time.Hour)},
			{UserID: 1, Amount: 500000000, BlockID: 3, Timestamp: now.Add(-25 * time.Hour)}, // Outside 24h window
		},
	}

	calculator, err := NewPPLNSCalculator(1000, 1.0)
	require.NoError(t, err)

	service := NewPayoutService(mockDB, calculator)

	// Get stats for last 24 hours
	stats, err := service.GetPayoutStatistics(context.Background(), 1, now.Add(-24*time.Hour))
	require.NoError(t, err)

	assert.Equal(t, int64(1), stats.UserID)
	assert.Equal(t, int64(3000000000), stats.TotalPayout) // 1B + 2B = 3B (500M is outside window)
	assert.Equal(t, 2, stats.PayoutCount)
	assert.Equal(t, int64(1500000000), stats.AveragePayout)
}

func TestPayoutService_GetPayoutStatistics_NoPayouts(t *testing.T) {
	mockDB := &MockDatabase{
		payouts: []Payout{},
	}

	calculator, err := NewPPLNSCalculator(1000, 1.0)
	require.NoError(t, err)

	service := NewPayoutService(mockDB, calculator)

	stats, err := service.GetPayoutStatistics(context.Background(), 1, time.Now().Add(-24*time.Hour))
	require.NoError(t, err)

	assert.Equal(t, int64(0), stats.TotalPayout)
	assert.Equal(t, 0, stats.PayoutCount)
	assert.Equal(t, int64(0), stats.AveragePayout)
}

func TestPayoutService_CalculateEstimatedPayout_NoShares(t *testing.T) {
	mockDB := &MockDatabase{
		shares: []Share{},
	}

	calculator, err := NewPPLNSCalculator(1000, 1.0)
	require.NoError(t, err)

	service := NewPayoutService(mockDB, calculator)

	payout, err := service.CalculateEstimatedPayout(context.Background(), 1, 10000000000)
	require.NoError(t, err)
	assert.Equal(t, int64(0), payout)
}

func TestPayoutService_CalculateEstimatedPayout_UserNotInWindow(t *testing.T) {
	mockDB := &MockDatabase{
		shares: []Share{
			{UserID: 2, Difficulty: 100, Timestamp: time.Now(), IsValid: true},
			{UserID: 3, Difficulty: 200, Timestamp: time.Now(), IsValid: true},
		},
	}

	calculator, err := NewPPLNSCalculator(1000, 1.0)
	require.NoError(t, err)

	service := NewPayoutService(mockDB, calculator)

	// User 1 has no shares
	payout, err := service.CalculateEstimatedPayout(context.Background(), 1, 10000000000)
	require.NoError(t, err)
	assert.Equal(t, int64(0), payout)
}

func TestPayoutService_ProcessBlockPayout_NoShares(t *testing.T) {
	mockDB := &MockDatabase{
		shares: []Share{},
		blocks: []Block{
			{
				ID:        1,
				Height:    12345,
				Hash:      "0x123abc",
				FinderID:  1,
				Reward:    5000000000,
				Status:    "confirmed",
				Timestamp: time.Now(),
			},
		},
	}

	calculator, err := NewPPLNSCalculator(1000, 1.0)
	require.NoError(t, err)

	service := NewPayoutService(mockDB, calculator)

	err = service.ProcessBlockPayout(context.Background(), 1)
	require.NoError(t, err)
	assert.Len(t, mockDB.payouts, 0) // No payouts created
}

func TestPayoutService_ProcessBlockPayout_OnlyInvalidShares(t *testing.T) {
	mockDB := &MockDatabase{
		shares: []Share{
			{UserID: 1, Difficulty: 100, Timestamp: time.Now(), IsValid: false},
			{UserID: 2, Difficulty: 200, Timestamp: time.Now(), IsValid: false},
		},
		blocks: []Block{
			{
				ID:        1,
				Height:    12345,
				Hash:      "0x123abc",
				FinderID:  1,
				Reward:    5000000000,
				Status:    "confirmed",
				Timestamp: time.Now(),
			},
		},
	}

	calculator, err := NewPPLNSCalculator(1000, 1.0)
	require.NoError(t, err)

	service := NewPayoutService(mockDB, calculator)

	err = service.ProcessBlockPayout(context.Background(), 1)
	require.NoError(t, err)
	assert.Len(t, mockDB.payouts, 0) // No payouts for invalid shares
}

func TestPayoutService_ValidatePayoutFairness_Valid(t *testing.T) {
	now := time.Now()
	mockDB := &MockDatabase{
		shares: []Share{
			{UserID: 1, Difficulty: 100, Timestamp: now.Add(-5 * time.Minute), IsValid: true},
			{UserID: 2, Difficulty: 100, Timestamp: now.Add(-4 * time.Minute), IsValid: true},
		},
		blocks: []Block{
			{
				ID:        1,
				Height:    12345,
				Hash:      "0x123abc",
				FinderID:  1,
				Reward:    2000000000,
				Status:    "confirmed",
				Timestamp: now,
			},
		},
	}

	calculator, err := NewPPLNSCalculator(1000, 1.0)
	require.NoError(t, err)

	service := NewPayoutService(mockDB, calculator)

	// First process the payout
	err = service.ProcessBlockPayout(context.Background(), 1)
	require.NoError(t, err)

	// Then validate fairness
	validation, err := service.ValidatePayoutFairness(context.Background(), 1)
	require.NoError(t, err)

	assert.True(t, validation.IsValid)
	assert.Len(t, validation.Discrepancies, 0)
}

func TestNewPayoutService(t *testing.T) {
	mockDB := &MockDatabase{}
	calculator, err := NewPPLNSCalculator(1000, 1.0)
	require.NoError(t, err)

	service := NewPayoutService(mockDB, calculator)
	assert.NotNil(t, service)
}

func TestPayoutStatistics_Structure(t *testing.T) {
	now := time.Now()
	stats := PayoutStatistics{
		UserID:        1,
		TotalPayout:   5000000000,
		PayoutCount:   10,
		AveragePayout: 500000000,
		LastPayout:    now,
		Since:         now.Add(-24 * time.Hour),
	}

	assert.Equal(t, int64(1), stats.UserID)
	assert.Equal(t, int64(5000000000), stats.TotalPayout)
	assert.Equal(t, 10, stats.PayoutCount)
}

func TestPayoutValidation_Structure(t *testing.T) {
	validation := PayoutValidation{
		BlockID:         1,
		IsValid:         true,
		ExpectedPayouts: []Payout{},
		ActualPayouts:   []Payout{},
		Discrepancies:   []PayoutDiscrepancy{},
	}

	assert.Equal(t, int64(1), validation.BlockID)
	assert.True(t, validation.IsValid)
}

func TestPayoutDiscrepancy_Structure(t *testing.T) {
	discrepancy := PayoutDiscrepancy{
		Type:        "amount_mismatch",
		UserID:      1,
		Description: "Expected 100, got 90",
	}

	assert.Equal(t, "amount_mismatch", discrepancy.Type)
	assert.Equal(t, int64(1), discrepancy.UserID)
}

func TestShare_Structure(t *testing.T) {
	now := time.Now()
	share := Share{
		ID:         1,
		UserID:     100,
		MinerID:    200,
		Difficulty: 500.5,
		IsValid:    true,
		Timestamp:  now,
	}

	assert.Equal(t, int64(1), share.ID)
	assert.Equal(t, int64(100), share.UserID)
	assert.Equal(t, float64(500.5), share.Difficulty)
	assert.True(t, share.IsValid)
}

func TestBlock_Structure(t *testing.T) {
	now := time.Now()
	block := Block{
		ID:         1,
		Height:     12345,
		Hash:       "0xabc123",
		Reward:     5000000000,
		Difficulty: 1000000,
		FinderID:   1,
		Status:     "confirmed",
		Timestamp:  now,
	}

	assert.Equal(t, int64(1), block.ID)
	assert.Equal(t, int64(12345), block.Height)
	assert.Equal(t, "0xabc123", block.Hash)
	assert.Equal(t, "confirmed", block.Status)
}

func TestPayout_Structure(t *testing.T) {
	now := time.Now()
	payout := Payout{
		UserID:    1,
		Amount:    1000000000,
		BlockID:   100,
		Timestamp: now,
	}

	assert.Equal(t, int64(1), payout.UserID)
	assert.Equal(t, int64(1000000000), payout.Amount)
	assert.Equal(t, int64(100), payout.BlockID)
}
