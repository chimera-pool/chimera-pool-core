package poolmanager

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestPoolManagerCompleteIntegration tests the complete integration of all pool manager components
func TestPoolManagerCompleteIntegration(t *testing.T) {
	t.Log("Starting comprehensive pool manager integration test...")

	config := &PoolManagerConfig{
		StratumAddress: ":3333",
		MaxMiners:      1000,
		BlockReward:    5000000000,
	}

	// Create mock components
	mockStratum := &MockStratumServer{}
	mockShares := &MockShareProcessor{}
	mockAuth := &MockAuthService{}
	mockPayouts := &MockPayoutService{}

	// Set up comprehensive mock expectations
	mockStratum.On("Start").Return(nil)
	mockStratum.On("Stop").Return(nil)
	mockStratum.On("GetConnectionCount").Return(10)

	mockShares.On("GetStatistics").Return(ShareStatistics{
		TotalShares:     1000,
		ValidShares:     950,
		InvalidShares:   50,
		TotalDifficulty: 15000.0,
		LastUpdated:     time.Now(),
	})

	mockShares.On("ProcessShare", mock.AnythingOfType("*poolmanager.Share")).Return(ShareProcessingResult{
		Success: true,
		ProcessedShare: &Share{
			ID:         1,
			MinerID:    123,
			UserID:     456,
			JobID:      "integration_test_job",
			Nonce:      "deadbeef",
			Hash:       "abcd1234",
			Difficulty: 1000.0,
			IsValid:    true,
			Timestamp:  time.Now(),
		},
	})

	mockAuth.On("ValidateJWT", "integration_test_token").Return(&JWTClaims{
		UserID:   456,
		Username: "integration_test_user",
		Email:    "test@integration.com",
	}, nil)

	mockPayouts.On("ProcessBlockPayout", mock.Anything, mock.AnythingOfType("int64")).Return(nil)

	// Create pool manager
	manager := NewPoolManager(config, mockStratum, mockShares, mockAuth, mockPayouts)

	// Test 1: Start pool manager
	t.Log("Testing pool manager startup...")
	err := manager.Start()
	assert.NoError(t, err, "Pool manager should start successfully")

	status := manager.GetStatus()
	assert.Equal(t, PoolStatusRunning, status.Status, "Pool should be in running state")
	assert.Equal(t, 10, status.ConnectedMiners, "Should report correct number of connected miners")

	// Test 2: Test all coordination methods
	ctx := context.Background()

	t.Log("Testing Stratum protocol coordination...")
	err = manager.CoordinateStratumProtocol(ctx)
	assert.NoError(t, err, "Stratum protocol coordination should succeed")

	t.Log("Testing share recording coordination...")
	testShare := &Share{
		ID:         1,
		MinerID:    123,
		UserID:     456,
		JobID:      "integration_test_job",
		Nonce:      "deadbeef",
		Difficulty: 1000.0,
		Timestamp:  time.Now(),
	}
	err = manager.CoordinateShareRecording(ctx, testShare)
	assert.NoError(t, err, "Share recording coordination should succeed")

	t.Log("Testing payout distribution coordination...")
	err = manager.CoordinatePayoutDistribution(ctx, 12345)
	assert.NoError(t, err, "Payout distribution coordination should succeed")

	t.Log("Testing concurrent miners coordination...")
	miners := []*MinerConnection{
		{ID: "integration_miner_1", Address: "192.168.1.1:12345", Username: "miner1"},
		{ID: "integration_miner_2", Address: "192.168.1.2:12345", Username: "miner2"},
		{ID: "integration_miner_3", Address: "192.168.1.3:12345", Username: "miner3"},
	}
	err = manager.CoordinateConcurrentMiners(ctx, miners)
	assert.NoError(t, err, "Concurrent miners coordination should succeed")

	t.Log("Testing block discovery coordination...")
	blockData := &BlockDiscovery{
		BlockHash:   "00000integration123test456",
		BlockHeight: 12345,
		Difficulty:  1000000.0,
		Reward:      5000000000,
		FoundBy:     "integration_miner_1",
		Timestamp:   time.Now(),
	}
	err = manager.CoordinateBlockDiscovery(ctx, blockData)
	assert.NoError(t, err, "Block discovery coordination should succeed")

	// Test 3: Test complete mining workflow
	t.Log("Testing complete mining workflow...")
	workflow := &MiningWorkflow{
		MinerConnection: &MinerConnection{
			ID:       "integration_workflow_miner",
			Address:  "192.168.1.100:12345",
			Username: "workflow_test_miner",
		},
		AuthToken: "integration_test_token",
		JobTemplate: &JobTemplate{
			ID:         "integration_workflow_job",
			PrevHash:   "0000integration",
			Difficulty: 1000.0,
		},
	}

	result, err := manager.ExecuteCompleteMiningWorkflow(ctx, workflow)
	assert.NoError(t, err, "Complete mining workflow should succeed")
	assert.NotNil(t, result, "Workflow result should not be nil")
	assert.True(t, result.Success, "Workflow should be successful")
	assert.Equal(t, 1, result.SharesProcessed, "Should process exactly one share")
	assert.Equal(t, 1, result.BlocksFound, "Should find one block (high difficulty)")
	assert.Equal(t, 1, result.PayoutsIssued, "Should issue one payout")
	assert.Empty(t, result.Errors, "Should have no errors")

	// Test 4: Test component health coordination
	t.Log("Testing component health coordination...")
	healthReport, err := manager.CoordinateComponentHealthCheck(ctx)
	assert.NoError(t, err, "Component health coordination should succeed")
	assert.NotNil(t, healthReport, "Health report should not be nil")
	assert.Equal(t, "healthy", healthReport.OverallHealth, "Overall health should be healthy")
	assert.Equal(t, 4, healthReport.Metrics["total_components"], "Should have 4 total components")
	assert.Equal(t, 4, healthReport.Metrics["healthy_components"], "All components should be healthy")
	assert.Equal(t, 100.0, healthReport.Metrics["health_percentage"], "Health percentage should be 100%")

	// Test 5: Test pool statistics
	t.Log("Testing pool statistics...")
	stats := manager.GetPoolStatistics()
	assert.NotNil(t, stats, "Pool statistics should not be nil")
	assert.Equal(t, 10, stats.TotalMiners, "Should report correct total miners")
	assert.Equal(t, 10, stats.ActiveMiners, "Should report correct active miners")
	assert.Equal(t, 15000.0, stats.TotalHashrate, "Should report correct total hashrate")
	assert.Equal(t, int64(1000), stats.ShareStatistics.TotalShares, "Should report correct total shares")
	assert.Equal(t, int64(950), stats.ShareStatistics.ValidShares, "Should report correct valid shares")

	// Test 6: Verify all requirements are met
	t.Log("Verifying requirements compliance...")

	// Requirement 2.1: Stratum v1 protocol support
	assert.NoError(t, manager.CoordinateStratumProtocol(ctx), "Should support Stratum v1 protocol coordination")

	// Requirement 6.1: Share recording and crediting
	assert.NoError(t, manager.CoordinateShareRecording(ctx, testShare), "Should record and credit shares")

	// Requirement 6.2: PPLNS payout distribution
	assert.NoError(t, manager.CoordinatePayoutDistribution(ctx, 12345), "Should handle PPLNS payout distribution")

	// Test 7: Stop pool manager
	t.Log("Testing pool manager shutdown...")
	err = manager.Stop()
	assert.NoError(t, err, "Pool manager should stop successfully")

	finalStatus := manager.GetStatus()
	assert.Equal(t, PoolStatusStopped, finalStatus.Status, "Pool should be in stopped state")

	t.Log("✅ Comprehensive pool manager integration test completed successfully!")

	// Verify all mock expectations were met
	mockStratum.AssertExpectations(t)
	mockShares.AssertExpectations(t)
	mockAuth.AssertExpectations(t)
	mockPayouts.AssertExpectations(t)
}

// TestPoolManagerErrorHandling tests error handling scenarios
func TestPoolManagerErrorHandling(t *testing.T) {
	t.Log("Testing pool manager error handling scenarios...")

	config := &PoolManagerConfig{
		StratumAddress: ":3333",
		MaxMiners:      1000,
		BlockReward:    5000000000,
	}

	mockStratum := &MockStratumServer{}
	mockShares := &MockShareProcessor{}
	mockAuth := &MockAuthService{}
	mockPayouts := &MockPayoutService{}

	manager := NewPoolManager(config, mockStratum, mockShares, mockAuth, mockPayouts)
	ctx := context.Background()

	// Test error handling when pool is not running
	t.Log("Testing operations when pool is not running...")

	err := manager.CoordinateStratumProtocol(ctx)
	assert.Error(t, err, "Should fail when pool is not running")
	assert.Contains(t, err.Error(), "must be running", "Error should indicate pool is not running")

	testShare := &Share{ID: 1, MinerID: 123, UserID: 456}
	err = manager.CoordinateShareRecording(ctx, testShare)
	assert.Error(t, err, "Should fail when pool is not running")

	err = manager.CoordinatePayoutDistribution(ctx, 12345)
	assert.Error(t, err, "Should fail when pool is not running")

	miners := []*MinerConnection{{ID: "test", Username: "test"}}
	err = manager.CoordinateConcurrentMiners(ctx, miners)
	assert.Error(t, err, "Should fail when pool is not running")

	blockData := &BlockDiscovery{BlockHash: "test", BlockHeight: 1, Reward: 1000}
	err = manager.CoordinateBlockDiscovery(ctx, blockData)
	assert.Error(t, err, "Should fail when pool is not running")

	t.Log("✅ Error handling test completed successfully!")
}

// TestPoolManagerConcurrency tests concurrent operations
func TestPoolManagerConcurrency(t *testing.T) {
	t.Log("Testing pool manager concurrent operations...")

	config := &PoolManagerConfig{
		StratumAddress: ":3333",
		MaxMiners:      1000,
		BlockReward:    5000000000,
	}

	mockStratum := &MockStratumServer{}
	mockShares := &MockShareProcessor{}
	mockAuth := &MockAuthService{}
	mockPayouts := &MockPayoutService{}

	// Set up mock expectations for concurrent operations
	mockStratum.On("Start").Return(nil)
	mockStratum.On("Stop").Return(nil)
	mockStratum.On("GetConnectionCount").Return(100)

	mockShares.On("GetStatistics").Return(ShareStatistics{
		TotalShares:   10000,
		ValidShares:   9500,
		InvalidShares: 500,
		LastUpdated:   time.Now(),
	})

	manager := NewPoolManager(config, mockStratum, mockShares, mockAuth, mockPayouts)

	// Start the manager
	err := manager.Start()
	assert.NoError(t, err, "Pool manager should start successfully")

	ctx := context.Background()

	// Test concurrent health checks
	t.Log("Testing concurrent health checks...")
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			healthReport, err := manager.CoordinateComponentHealthCheck(ctx)
			assert.NoError(t, err, "Concurrent health check should succeed")
			assert.NotNil(t, healthReport, "Health report should not be nil")
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Test concurrent status requests
	t.Log("Testing concurrent status requests...")
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			status := manager.GetStatus()
			assert.NotNil(t, status, "Status should not be nil")
			assert.Equal(t, PoolStatusRunning, status.Status, "Pool should be running")
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Stop the manager
	err = manager.Stop()
	assert.NoError(t, err, "Pool manager should stop successfully")

	t.Log("✅ Concurrency test completed successfully!")

	// Verify mock expectations
	mockStratum.AssertExpectations(t)
	mockShares.AssertExpectations(t)
}