package payouts

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// E2EDatabase implements a more realistic database for end-to-end testing
type E2EDatabase struct {
	shares  []Share
	payouts []Payout
	blocks  []Block
	users   []User
}

type User struct {
	ID       int64  `json:"id" db:"id"`
	Username string `json:"username" db:"username"`
	Email    string `json:"email" db:"email"`
}

func (db *E2EDatabase) GetSharesForPayout(ctx context.Context, blockTime time.Time, windowSize int64) ([]Share, error) {
	// Sort shares by timestamp descending
	sortedShares := make([]Share, len(db.shares))
	copy(sortedShares, db.shares)
	
	// Simple bubble sort by timestamp descending
	for i := 0; i < len(sortedShares); i++ {
		for j := i + 1; j < len(sortedShares); j++ {
			if sortedShares[i].Timestamp.Before(sortedShares[j].Timestamp) {
				sortedShares[i], sortedShares[j] = sortedShares[j], sortedShares[i]
			}
		}
	}
	
	return sortedShares, nil
}

func (db *E2EDatabase) CreatePayouts(ctx context.Context, payouts []Payout) error {
	db.payouts = append(db.payouts, payouts...)
	return nil
}

func (db *E2EDatabase) GetBlock(ctx context.Context, blockID int64) (*Block, error) {
	for _, block := range db.blocks {
		if block.ID == blockID {
			return &block, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (db *E2EDatabase) GetPayoutHistory(ctx context.Context, userID int64, limit, offset int) ([]Payout, error) {
	result := make([]Payout, 0)
	
	if userID == 0 {
		result = append(result, db.payouts...)
	} else {
		for _, payout := range db.payouts {
			if payout.UserID == userID {
				result = append(result, payout)
			}
		}
	}
	
	// Sort by timestamp descending
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i].Timestamp.Before(result[j].Timestamp) {
				result[i], result[j] = result[j], result[i]
			}
		}
	}
	
	// Apply pagination
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

// TestE2E_CompleteMiningToPayoutWorkflow tests the complete end-to-end workflow
func TestE2E_CompleteMiningToPayoutWorkflow(t *testing.T) {
	// Setup: Create a realistic mining scenario
	now := time.Now()
	
	// Create users
	users := []User{
		{ID: 1, Username: "alice", Email: "alice@example.com"},
		{ID: 2, Username: "bob", Email: "bob@example.com"},
		{ID: 3, Username: "charlie", Email: "charlie@example.com"},
	}
	
	// Create mining shares over time (simulating 30 minutes of mining)
	shares := []Share{
		// Alice mines consistently with medium difficulty
		{ID: 1, UserID: 1, MinerID: 101, Difficulty: 100, IsValid: true, Timestamp: now.Add(-30 * time.Minute)},
		{ID: 2, UserID: 1, MinerID: 101, Difficulty: 120, IsValid: true, Timestamp: now.Add(-25 * time.Minute)},
		{ID: 3, UserID: 1, MinerID: 101, Difficulty: 110, IsValid: true, Timestamp: now.Add(-20 * time.Minute)},
		{ID: 4, UserID: 1, MinerID: 101, Difficulty: 105, IsValid: true, Timestamp: now.Add(-15 * time.Minute)},
		{ID: 5, UserID: 1, MinerID: 101, Difficulty: 115, IsValid: true, Timestamp: now.Add(-10 * time.Minute)},
		
		// Bob mines with higher difficulty but less frequently
		{ID: 6, UserID: 2, MinerID: 102, Difficulty: 200, IsValid: true, Timestamp: now.Add(-28 * time.Minute)},
		{ID: 7, UserID: 2, MinerID: 102, Difficulty: 180, IsValid: true, Timestamp: now.Add(-18 * time.Minute)},
		{ID: 8, UserID: 2, MinerID: 102, Difficulty: 220, IsValid: true, Timestamp: now.Add(-8 * time.Minute)},
		
		// Charlie mines with lower difficulty but very frequently
		{ID: 9, UserID: 3, MinerID: 103, Difficulty: 50, IsValid: true, Timestamp: now.Add(-29 * time.Minute)},
		{ID: 10, UserID: 3, MinerID: 103, Difficulty: 60, IsValid: true, Timestamp: now.Add(-27 * time.Minute)},
		{ID: 11, UserID: 3, MinerID: 103, Difficulty: 55, IsValid: true, Timestamp: now.Add(-24 * time.Minute)},
		{ID: 12, UserID: 3, MinerID: 103, Difficulty: 65, IsValid: true, Timestamp: now.Add(-22 * time.Minute)},
		{ID: 13, UserID: 3, MinerID: 103, Difficulty: 58, IsValid: true, Timestamp: now.Add(-19 * time.Minute)},
		{ID: 14, UserID: 3, MinerID: 103, Difficulty: 62, IsValid: true, Timestamp: now.Add(-16 * time.Minute)},
		{ID: 15, UserID: 3, MinerID: 103, Difficulty: 57, IsValid: true, Timestamp: now.Add(-13 * time.Minute)},
		{ID: 16, UserID: 3, MinerID: 103, Difficulty: 63, IsValid: true, Timestamp: now.Add(-11 * time.Minute)},
		{ID: 17, UserID: 3, MinerID: 103, Difficulty: 59, IsValid: true, Timestamp: now.Add(-7 * time.Minute)},
		{ID: 18, UserID: 3, MinerID: 103, Difficulty: 61, IsValid: true, Timestamp: now.Add(-5 * time.Minute)},
		
		// Some invalid shares (should be ignored)
		{ID: 19, UserID: 1, MinerID: 101, Difficulty: 90, IsValid: false, Timestamp: now.Add(-12 * time.Minute)},
		{ID: 20, UserID: 2, MinerID: 102, Difficulty: 150, IsValid: false, Timestamp: now.Add(-6 * time.Minute)},
	}
	
	// Block found by Alice
	block := Block{
		ID:       1,
		Height:   100001,
		Hash:     "0x1234567890abcdef",
		FinderID: 1, // Alice found the block
		Reward:   5000000000, // 50 coins in satoshis
		Status:   "confirmed",
		Created:  now,
	}
	
	// Setup database
	db := &E2EDatabase{
		shares:  shares,
		payouts: []Payout{},
		blocks:  []Block{block},
		users:   users,
	}
	
	// Create PPLNS calculator with 1000 difficulty window and 2% pool fee
	calculator, err := NewPPLNSCalculator(1000, 2.0)
	require.NoError(t, err)
	
	// Create payout service
	service := NewPayoutService(db, calculator)
	
	// Step 1: Calculate expected contributions (applying sliding window manually for verification)
	
	// Calculate what the sliding window should contain
	sortedShares := make([]Share, len(shares))
	copy(sortedShares, shares)
	
	// Sort by timestamp descending (newest first)
	for i := 0; i < len(sortedShares); i++ {
		for j := i + 1; j < len(sortedShares); j++ {
			if sortedShares[i].Timestamp.Before(sortedShares[j].Timestamp) {
				sortedShares[i], sortedShares[j] = sortedShares[j], sortedShares[i]
			}
		}
	}
	
	// Apply sliding window manually to understand expected behavior
	windowShares := make([]Share, 0)
	accumulatedDifficulty := float64(0)
	windowSize := float64(1000)
	
	for _, share := range sortedShares {
		if !share.IsValid {
			continue
		}
		
		remainingWindow := windowSize - accumulatedDifficulty
		if remainingWindow <= 0 {
			break
		}
		
		if share.Difficulty <= remainingWindow {
			windowShares = append(windowShares, share)
			accumulatedDifficulty += share.Difficulty
		} else {
			// Partial share
			partialShare := share
			partialShare.Difficulty = remainingWindow
			windowShares = append(windowShares, partialShare)
			accumulatedDifficulty += remainingWindow
			break
		}
	}
	
	// Calculate contributions from window shares
	totalValidDifficulty := float64(0)
	userContributions := make(map[int64]float64)
	
	for _, share := range windowShares {
		totalValidDifficulty += share.Difficulty
		userContributions[share.UserID] += share.Difficulty
	}
	
	t.Logf("Total valid difficulty: %.2f", totalValidDifficulty)
	t.Logf("Alice contribution: %.2f (%.2f%%)", userContributions[1], userContributions[1]/totalValidDifficulty*100)
	t.Logf("Bob contribution: %.2f (%.2f%%)", userContributions[2], userContributions[2]/totalValidDifficulty*100)
	t.Logf("Charlie contribution: %.2f (%.2f%%)", userContributions[3], userContributions[3]/totalValidDifficulty*100)
	
	// Step 2: Process block payout
	err = service.ProcessBlockPayout(context.Background(), 1)
	require.NoError(t, err)
	
	// Step 3: Verify payouts were created
	require.Len(t, db.payouts, 3, "Should create payouts for all 3 users")
	
	// Step 4: Verify mathematical accuracy
	netReward := int64(float64(block.Reward) * 0.98) // 98% after 2% pool fee
	totalPayout := int64(0)
	
	for _, payout := range db.payouts {
		totalPayout += payout.Amount
		
		// Verify payout is proportional to contribution
		expectedAmount := int64(float64(netReward) * userContributions[payout.UserID] / totalValidDifficulty)
		assert.Equal(t, expectedAmount, payout.Amount, "Payout for user %d should be proportional to contribution", payout.UserID)
		
		// Verify payout metadata
		assert.Equal(t, block.ID, payout.BlockID, "Payout should reference correct block")
		assert.WithinDuration(t, block.Created, payout.Timestamp, time.Second, "Payout timestamp should match block time")
		
		t.Logf("User %d payout: %d satoshis (%.8f coins)", payout.UserID, payout.Amount, float64(payout.Amount)/100000000)
	}
	
	// Verify total payout equals net reward
	assert.Equal(t, netReward, totalPayout, "Total payouts should equal net reward after pool fee")
	
	// Step 5: Test payout history retrieval
	aliceHistory, err := service.GetPayoutHistory(context.Background(), 1, 10, 0)
	require.NoError(t, err)
	require.Len(t, aliceHistory, 1, "Alice should have 1 payout")
	
	bobHistory, err := service.GetPayoutHistory(context.Background(), 2, 10, 0)
	require.NoError(t, err)
	require.Len(t, bobHistory, 1, "Bob should have 1 payout")
	
	charlieHistory, err := service.GetPayoutHistory(context.Background(), 3, 10, 0)
	require.NoError(t, err)
	require.Len(t, charlieHistory, 1, "Charlie should have 1 payout")
	
	// Step 6: Test estimated payout calculation
	estimatedReward := int64(6000000000) // 60 coins
	
	aliceEstimate, err := service.CalculateEstimatedPayout(context.Background(), 1, estimatedReward)
	require.NoError(t, err)
	
	expectedAliceEstimate := int64(float64(estimatedReward) * 0.98 * userContributions[1] / totalValidDifficulty)
	assert.Equal(t, expectedAliceEstimate, aliceEstimate, "Alice's estimated payout should be proportional")
	
	// Step 7: Test payout fairness validation
	validation, err := service.ValidatePayoutFairness(context.Background(), 1)
	require.NoError(t, err)
	assert.True(t, validation.IsValid, "Payouts should be mathematically fair")
	assert.Empty(t, validation.Discrepancies, "Should have no discrepancies")
	
	// Step 8: Test payout statistics
	stats, err := service.GetPayoutStatistics(context.Background(), 1, now.Add(-1*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, int64(1), stats.UserID)
	assert.Equal(t, 1, stats.PayoutCount)
	assert.Equal(t, aliceHistory[0].Amount, stats.TotalPayout)
	assert.Equal(t, aliceHistory[0].Amount, stats.AveragePayout)
	
	t.Logf("✅ E2E test completed successfully!")
	t.Logf("   Block reward: %.8f coins", float64(block.Reward)/100000000)
	t.Logf("   Pool fee: %.8f coins (2%%)", float64(block.Reward-netReward)/100000000)
	t.Logf("   Total distributed: %.8f coins", float64(totalPayout)/100000000)
	t.Logf("   Alice earned: %.8f coins", float64(aliceHistory[0].Amount)/100000000)
	t.Logf("   Bob earned: %.8f coins", float64(bobHistory[0].Amount)/100000000)
	t.Logf("   Charlie earned: %.8f coins", float64(charlieHistory[0].Amount)/100000000)
}

// TestE2E_SlidingWindowBehavior tests the sliding window behavior with realistic data
func TestE2E_SlidingWindowBehavior(t *testing.T) {
	now := time.Now()
	
	// Create shares that exceed the window size
	shares := []Share{
		// Old shares (should be excluded from window)
		{ID: 1, UserID: 1, Difficulty: 300, IsValid: true, Timestamp: now.Add(-60 * time.Minute)},
		{ID: 2, UserID: 2, Difficulty: 400, IsValid: true, Timestamp: now.Add(-50 * time.Minute)},
		
		// Recent shares (should be included in window)
		{ID: 3, UserID: 1, Difficulty: 200, IsValid: true, Timestamp: now.Add(-10 * time.Minute)},
		{ID: 4, UserID: 2, Difficulty: 150, IsValid: true, Timestamp: now.Add(-8 * time.Minute)},
		{ID: 5, UserID: 3, Difficulty: 100, IsValid: true, Timestamp: now.Add(-5 * time.Minute)},
		{ID: 6, UserID: 1, Difficulty: 50, IsValid: true, Timestamp: now.Add(-2 * time.Minute)},
	}
	
	block := Block{
		ID:       1,
		Height:   100001,
		Hash:     "0xabcdef",
		FinderID: 1,
		Reward:   1000000000, // 10 coins
		Status:   "confirmed",
		Created:  now,
	}
	
	db := &E2EDatabase{
		shares:  shares,
		payouts: []Payout{},
		blocks:  []Block{block},
	}
	
	// Use small window size to test sliding window behavior
	calculator, err := NewPPLNSCalculator(400, 1.0) // 400 difficulty window
	require.NoError(t, err)
	
	service := NewPayoutService(db, calculator)
	
	// Process payout
	err = service.ProcessBlockPayout(context.Background(), 1)
	require.NoError(t, err)
	
	// Verify only recent shares within window are considered
	// Window should include: 50 + 100 + 150 + 100 (partial from 200) = 400
	// So user1: 50 + 100 = 150, user2: 150, user3: 100
	
	expectedContributions := map[int64]float64{
		1: 150, // 50 + 100 (partial from 200 difficulty share)
		2: 150, // 150
		3: 100, // 100
	}
	
	totalExpected := float64(400) // Window size
	netReward := int64(float64(block.Reward) * 0.99) // 99% after 1% fee
	
	for _, payout := range db.payouts {
		expectedAmount := int64(float64(netReward) * expectedContributions[payout.UserID] / totalExpected)
		assert.Equal(t, expectedAmount, payout.Amount, "User %d payout should reflect sliding window", payout.UserID)
	}
	
	t.Logf("✅ Sliding window test completed successfully!")
	t.Logf("   Window size: %.0f difficulty", totalExpected)
	for userID, contribution := range expectedContributions {
		payout := findPayoutForUser(db.payouts, userID)
		if payout != nil {
			t.Logf("   User %d: %.0f difficulty (%.1f%%) = %d satoshis", 
				userID, contribution, contribution/totalExpected*100, payout.Amount)
		}
	}
}

// TestE2E_MultipleBlocksScenario tests handling multiple blocks and cumulative payouts
func TestE2E_MultipleBlocksScenario(t *testing.T) {
	now := time.Now()
	
	// Shares for first block period
	shares := []Share{
		{ID: 1, UserID: 1, Difficulty: 200, IsValid: true, Timestamp: now.Add(-30 * time.Minute)},
		{ID: 2, UserID: 2, Difficulty: 300, IsValid: true, Timestamp: now.Add(-25 * time.Minute)},
	}
	
	// First block
	block1 := Block{
		ID: 1, Height: 100001, Hash: "0xblock1", FinderID: 1,
		Reward: 2000000000, Status: "confirmed", Created: now.Add(-20 * time.Minute),
	}
	
	// Additional shares for second block period
	additionalShares := []Share{
		{ID: 3, UserID: 1, Difficulty: 150, IsValid: true, Timestamp: now.Add(-15 * time.Minute)},
		{ID: 4, UserID: 3, Difficulty: 250, IsValid: true, Timestamp: now.Add(-10 * time.Minute)},
	}
	
	// Second block
	block2 := Block{
		ID: 2, Height: 100002, Hash: "0xblock2", FinderID: 2,
		Reward: 2500000000, Status: "confirmed", Created: now.Add(-5 * time.Minute),
	}
	
	db := &E2EDatabase{
		shares:  append(shares, additionalShares...),
		payouts: []Payout{},
		blocks:  []Block{block1, block2},
	}
	
	calculator, err := NewPPLNSCalculator(1000, 1.5) // 1.5% pool fee
	require.NoError(t, err)
	
	service := NewPayoutService(db, calculator)
	
	// Process first block
	err = service.ProcessBlockPayout(context.Background(), 1)
	require.NoError(t, err)
	
	// Process second block
	err = service.ProcessBlockPayout(context.Background(), 2)
	require.NoError(t, err)
	
	// Verify payouts for both blocks
	allPayouts := db.payouts
	
	block1Payouts := make([]Payout, 0)
	block2Payouts := make([]Payout, 0)
	
	for _, payout := range allPayouts {
		if payout.BlockID == 1 {
			block1Payouts = append(block1Payouts, payout)
		} else if payout.BlockID == 2 {
			block2Payouts = append(block2Payouts, payout)
		}
	}
	
	// Verify we have payouts for both blocks
	assert.NotEmpty(t, block1Payouts, "Should have payouts for block 1")
	assert.NotEmpty(t, block2Payouts, "Should have payouts for block 2")
	
	// Test cumulative payout history
	user1History, err := service.GetPayoutHistory(context.Background(), 1, 10, 0)
	require.NoError(t, err)
	
	// User 1 should have payouts from both blocks
	assert.Len(t, user1History, 2, "User 1 should have payouts from both blocks")
	
	// Calculate total earnings for user 1
	totalEarnings := int64(0)
	for _, payout := range user1History {
		totalEarnings += payout.Amount
	}
	
	t.Logf("✅ Multiple blocks test completed successfully!")
	t.Logf("   User 1 total earnings: %.8f coins from %d blocks", 
		float64(totalEarnings)/100000000, len(user1History))
	
	// Test statistics over time period
	stats, err := service.GetPayoutStatistics(context.Background(), 1, now.Add(-1*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 2, stats.PayoutCount, "User 1 should have 2 payouts")
	assert.Equal(t, totalEarnings, stats.TotalPayout, "Total payout should match sum")
}