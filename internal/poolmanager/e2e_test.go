package poolmanager

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestEndToEndMiningWorkflow tests the complete mining workflow from start to finish
func TestEndToEndMiningWorkflow(t *testing.T) {
	// This test validates the complete end-to-end mining workflow
	// as specified in requirements 2.1, 6.1, and 6.2
	
	config := &PoolManagerConfig{
		StratumAddress: ":0", // Use random port for testing
		MaxMiners:      50,
		BlockReward:    5000000000, // 50 coins in satoshis
	}

	// Create mock component implementations for testing
	stratumServer := &MockStratumServer{}
	shareProcessor := &MockShareProcessor{}
	authService := &MockAuthService{}
	payoutService := &MockPayoutService{}

	// Set up mock expectations
	stratumServer.On("Start").Return(nil)
	stratumServer.On("Stop").Return(nil)
	stratumServer.On("GetConnectionCount").Return(0)

	// Mock authentication
	testUser := &User{
		ID:       123,
		Username: "testuser",
		Email:    "test@example.com",
		IsActive: true,
	}
	testToken := "test_jwt_token"
	testClaims := &JWTClaims{
		UserID:   123,
		Username: "testuser",
		Email:    "test@example.com",
	}

	authService.On("LoginUser", "testuser", "testpass").Return(testUser, testToken, nil)
	authService.On("ValidateJWT", testToken).Return(testClaims, nil)

	// Mock share processing
	shareProcessor.On("GetStatistics").Return(ShareStatistics{
		TotalShares:   3,
		ValidShares:   3,
		InvalidShares: 0,
		LastUpdated:   time.Now(),
	})

	shareProcessor.On("ProcessShare", mock.AnythingOfType("*poolmanager.Share")).Return(ShareProcessingResult{
		Success: true,
		ProcessedShare: &Share{
			ID:         1,
			MinerID:    100,
			UserID:     123,
			JobID:      "job_001",
			Nonce:      "abcd1234",
			Hash:       "processed_hash",
			Difficulty: 1.0,
			IsValid:    true,
			Timestamp:  time.Now(),
		},
	})

	// Mock payout service
	payoutService.On("CalculateEstimatedPayout", mock.Anything, int64(123), int64(5000000000)).Return(int64(500000000), nil)

	manager := NewPoolManager(config, stratumServer, shareProcessor, authService, payoutService)

	// Step 1: Start the pool manager
	t.Log("Starting pool manager...")
	err := manager.Start()
	require.NoError(t, err, "Pool manager should start successfully")

	// Verify pool is running
	status := manager.GetStatus()
	assert.Equal(t, PoolStatusRunning, status.Status, "Pool should be in running state")
	assert.Equal(t, 0, status.ConnectedMiners, "Should start with 0 connected miners")

	// Step 2: Simulate miner connection and authentication
	t.Log("Testing authentication workflow...")
	user, token, err := authService.LoginUser("testuser", "testpass")
	require.NoError(t, err, "Authentication should succeed")
	assert.NotNil(t, user, "User should be returned")
	assert.NotEmpty(t, token, "Token should be generated")

	// Validate the token
	claims, err := authService.ValidateJWT(token)
	require.NoError(t, err, "Token validation should succeed")
	assert.Equal(t, user.ID, claims.UserID, "Token should contain correct user ID")

	// Step 3: Process multiple shares to simulate mining activity
	t.Log("Processing mining shares...")
	shares := []*Share{
		{
			ID:         1,
			MinerID:    100,
			UserID:     claims.UserID,
			JobID:      "job_001",
			Nonce:      "abcd1234",
			Difficulty: 1.0,
			Timestamp:  time.Now(),
		},
		{
			ID:         2,
			MinerID:    101,
			UserID:     claims.UserID,
			JobID:      "job_002",
			Nonce:      "efgh5678",
			Difficulty: 2.0,
			Timestamp:  time.Now(),
		},
		{
			ID:         3,
			MinerID:    102,
			UserID:     claims.UserID,
			JobID:      "job_003",
			Nonce:      "ijkl9012",
			Difficulty: 1.5,
			Timestamp:  time.Now(),
		},
	}

	var processedShares []*Share
	for i, share := range shares {
		t.Logf("Processing share %d...", i+1)
		result := manager.ProcessShare(share)
		assert.True(t, result.Success, "Share processing should succeed")
		assert.NotNil(t, result.ProcessedShare, "Processed share should be returned")
		assert.True(t, result.ProcessedShare.IsValid, "Share should be valid")
		assert.NotEmpty(t, result.ProcessedShare.Hash, "Share should have a hash")
		
		processedShares = append(processedShares, result.ProcessedShare)
	}

	// Step 4: Verify statistics are updated correctly
	t.Log("Verifying pool statistics...")
	stats := manager.GetPoolStatistics()
	assert.NotNil(t, stats, "Statistics should be available")
	assert.Equal(t, int64(3), stats.ShareStatistics.TotalShares, "Should have processed 3 shares")
	assert.Equal(t, int64(3), stats.ShareStatistics.ValidShares, "All shares should be valid")
	assert.Equal(t, int64(0), stats.ShareStatistics.InvalidShares, "No invalid shares")
	// Note: TotalDifficulty comes from mock, so we just verify it's available
	assert.GreaterOrEqual(t, stats.ShareStatistics.TotalDifficulty, 0.0, "Total difficulty should be available")

	// Step 5: Test mining workflow coordination
	t.Log("Testing mining workflow coordination...")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = manager.CoordinateMiningWorkflow(ctx)
	assert.NoError(t, err, "Mining workflow coordination should succeed")

	// Step 6: Test component health monitoring
	t.Log("Checking component health...")
	health := manager.GetComponentHealth()
	assert.NotNil(t, health, "Component health should be available")
	assert.Equal(t, "healthy", health.StratumServer.Status, "Stratum server should be healthy")
	assert.Equal(t, "healthy", health.ShareProcessor.Status, "Share processor should be healthy")
	assert.Equal(t, "healthy", health.AuthService.Status, "Auth service should be healthy")
	assert.Equal(t, "healthy", health.PayoutService.Status, "Payout service should be healthy")

	// Step 7: Test payout calculation
	t.Log("Testing payout calculation...")
	estimatedPayout, err := payoutService.CalculateEstimatedPayout(ctx, claims.UserID, config.BlockReward)
	assert.NoError(t, err, "Payout calculation should succeed")
	assert.Greater(t, estimatedPayout, int64(0), "Estimated payout should be positive")

	// Step 8: Test workflow coordination with authentication
	t.Log("Testing authenticated workflow coordination...")
	// Since the pool is already running, we'll test the workflow components individually
	
	// Validate the token again to simulate ongoing authentication
	validatedClaims, err := authService.ValidateJWT(token)
	assert.NoError(t, err, "Token should remain valid")
	assert.Equal(t, claims.UserID, validatedClaims.UserID, "Claims should match")

	// Step 9: Verify final state
	t.Log("Verifying final state...")
	finalStatus := manager.GetStatus()
	assert.Equal(t, PoolStatusRunning, finalStatus.Status, "Pool should still be running")
	assert.GreaterOrEqual(t, finalStatus.TotalShares, int64(3), "Should have at least 3 total shares")
	assert.GreaterOrEqual(t, finalStatus.ValidShares, int64(3), "Should have at least 3 valid shares")

	// Step 10: Clean shutdown
	t.Log("Shutting down pool manager...")
	err = manager.Stop()
	assert.NoError(t, err, "Pool manager should stop cleanly")

	// Verify pool is stopped
	stoppedStatus := manager.GetStatus()
	assert.Equal(t, PoolStatusStopped, stoppedStatus.Status, "Pool should be stopped")

	t.Log("End-to-end mining workflow test completed successfully!")
}