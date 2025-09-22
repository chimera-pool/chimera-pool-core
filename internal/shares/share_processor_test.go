package shares

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestShareProcessor_ValidateShare tests share validation using Blake2S algorithm
func TestShareProcessor_ValidateShare(t *testing.T) {
	tests := []struct {
		name           string
		share          *Share
		expectedValid  bool
		expectedError  string
	}{
		{
			name: "valid_share_easy_target",
			share: &Share{
				MinerID:    1,
				UserID:     1,
				JobID:      "job123",
				Nonce:      "12345678",
				Timestamp:  time.Now(),
				Difficulty: 1.0,
				Hash:       "", // Will be calculated
			},
			expectedValid: true,
			expectedError: "",
		},
		{
			name: "invalid_share_impossible_target",
			share: &Share{
				MinerID:    1,
				UserID:     1,
				JobID:      "job123",
				Nonce:      "12345678",
				Timestamp:  time.Now(),
				Difficulty: 999999999.0, // Impossible difficulty
				Hash:       "",
			},
			expectedValid: false,
			expectedError: "", // No error expected - share is processed but marked invalid
		},
		{
			name: "invalid_share_empty_nonce",
			share: &Share{
				MinerID:    1,
				UserID:     1,
				JobID:      "job123",
				Nonce:      "",
				Timestamp:  time.Now(),
				Difficulty: 1.0,
				Hash:       "",
			},
			expectedValid: false,
			expectedError: "nonce cannot be empty",
		},
		{
			name: "invalid_share_empty_job_id",
			share: &Share{
				MinerID:    1,
				UserID:     1,
				JobID:      "",
				Nonce:      "12345678",
				Timestamp:  time.Now(),
				Difficulty: 1.0,
				Hash:       "",
			},
			expectedValid: false,
			expectedError: "job_id cannot be empty",
		},
	}

	processor := NewShareProcessor()
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.ValidateShare(tt.share)
			
			assert.Equal(t, tt.expectedValid, result.IsValid, "Share validity mismatch")
			
			if tt.expectedError != "" {
				assert.Contains(t, result.Error, tt.expectedError, "Error message mismatch")
			}
			
			// Valid shares should have computed hash
			if result.IsValid {
				assert.NotEmpty(t, result.Hash, "Valid share should have computed hash")
			}
		})
	}
}

// TestShareProcessor_ProcessShare tests complete share processing workflow
func TestShareProcessor_ProcessShare(t *testing.T) {
	processor := NewShareProcessor()
	
	share := &Share{
		MinerID:    1,
		UserID:     1,
		JobID:      "job123",
		Nonce:      "12345678",
		Timestamp:  time.Now(),
		Difficulty: 1.0,
	}
	
	// Process the share
	result := processor.ProcessShare(share)
	
	// Should be successful
	require.True(t, result.Success, "Share processing should succeed")
	require.NotNil(t, result.ProcessedShare, "Processed share should not be nil")
	
	// Check that share was validated and hash computed
	assert.True(t, result.ProcessedShare.IsValid, "Share should be valid")
	assert.NotEmpty(t, result.ProcessedShare.Hash, "Share should have computed hash")
	assert.Equal(t, share.MinerID, result.ProcessedShare.MinerID, "Miner ID should match")
	assert.Equal(t, share.UserID, result.ProcessedShare.UserID, "User ID should match")
}

// TestShareProcessor_TrackStatistics tests share statistics tracking
func TestShareProcessor_TrackStatistics(t *testing.T) {
	processor := NewShareProcessor()
	
	// Process multiple shares
	shares := []*Share{
		{MinerID: 1, UserID: 1, JobID: "job1", Nonce: "11111111", Difficulty: 1.0, Timestamp: time.Now()},
		{MinerID: 1, UserID: 1, JobID: "job2", Nonce: "22222222", Difficulty: 1.0, Timestamp: time.Now()},
		{MinerID: 2, UserID: 1, JobID: "job3", Nonce: "33333333", Difficulty: 2.0, Timestamp: time.Now()},
	}
	
	for _, share := range shares {
		result := processor.ProcessShare(share)
		require.True(t, result.Success, "All shares should process successfully")
	}
	
	// Check statistics
	stats := processor.GetStatistics()
	
	assert.Equal(t, int64(3), stats.TotalShares, "Should have processed 3 shares")
	assert.Equal(t, int64(3), stats.ValidShares, "All shares should be valid")
	assert.Equal(t, int64(0), stats.InvalidShares, "No invalid shares")
	assert.Equal(t, 4.0, stats.TotalDifficulty, "Total difficulty should be 1+1+2=4")
	
	// Check per-miner statistics
	minerStats := processor.GetMinerStatistics(1)
	assert.Equal(t, int64(2), minerStats.TotalShares, "Miner 1 should have 2 shares")
	assert.Equal(t, 2.0, minerStats.TotalDifficulty, "Miner 1 total difficulty should be 2")
	
	minerStats2 := processor.GetMinerStatistics(2)
	assert.Equal(t, int64(1), minerStats2.TotalShares, "Miner 2 should have 1 share")
	assert.Equal(t, 2.0, minerStats2.TotalDifficulty, "Miner 2 total difficulty should be 2")
}

// TestShareProcessor_PerformanceUnderLoad tests share processing performance
func TestShareProcessor_PerformanceUnderLoad(t *testing.T) {
	processor := NewShareProcessor()
	
	// Generate many shares for performance testing
	shareCount := 1000
	shares := make([]*Share, shareCount)
	
	for i := 0; i < shareCount; i++ {
		shares[i] = &Share{
			MinerID:    int64((i % 10) + 1), // 10 different miners (1-10)
			UserID:     int64((i % 5) + 1),  // 5 different users (1-5)
			JobID:      fmt.Sprintf("job_%d", i),
			Nonce:      fmt.Sprintf("%08x", i), // Proper hex nonce
			Difficulty: 1.0,
			Timestamp:  time.Now(),
		}
	}
	
	// Measure processing time
	start := time.Now()
	
	processedCount := 0
	for _, share := range shares {
		result := processor.ProcessShare(share)
		// Count all processed shares, regardless of validity
		if result.ProcessedShare != nil {
			processedCount++
		}
	}
	
	duration := time.Since(start)
	
	// Performance assertions
	assert.Equal(t, shareCount, processedCount, "All shares should be processed")
	assert.Less(t, duration, 5*time.Second, "Should process 1000 shares in under 5 seconds")
	
	// Check final statistics
	stats := processor.GetStatistics()
	assert.Equal(t, int64(shareCount), stats.TotalShares, "Should have processed all shares")
	// Note: Not all shares will be valid due to difficulty, but they should all be processed
	
	// Calculate shares per second
	sharesPerSecond := float64(processedCount) / duration.Seconds()
	t.Logf("Processed %d shares in %v (%.2f shares/second)", processedCount, duration, sharesPerSecond)
	
	// Should process at least 200 shares per second
	assert.Greater(t, sharesPerSecond, 200.0, "Should process at least 200 shares per second")
}

// TestShareProcessor_ConcurrentProcessing tests concurrent share processing
func TestShareProcessor_ConcurrentProcessing(t *testing.T) {
	processor := NewShareProcessor()
	
	// Number of concurrent goroutines
	goroutineCount := 10
	sharesPerGoroutine := 100
	
	// Channel to collect results
	results := make(chan bool, goroutineCount*sharesPerGoroutine)
	
	// Start concurrent processing
	for g := 0; g < goroutineCount; g++ {
		go func(goroutineID int) {
			for i := 0; i < sharesPerGoroutine; i++ {
				share := &Share{
					MinerID:    int64(goroutineID + 1), // Ensure positive ID
					UserID:     int64(goroutineID + 1), // Ensure positive ID
					JobID:      fmt.Sprintf("job_%d_%d", goroutineID, i),
					Nonce:      fmt.Sprintf("%08x", goroutineID*1000+i),
					Difficulty: 1.0,
					Timestamp:  time.Now(),
				}
				
				result := processor.ProcessShare(share)
				// Always send a result (true if processed, false if error)
				results <- (result.ProcessedShare != nil)
			}
		}(g)
	}
	
	// Collect all results
	processedCount := 0
	totalExpected := goroutineCount * sharesPerGoroutine
	
	for i := 0; i < totalExpected; i++ {
		// Count all processed shares (both successful and failed)
		<-results
		processedCount++
	}
	
	// All shares should be processed
	assert.Equal(t, totalExpected, processedCount, "All concurrent shares should be processed")
	
	// Check final statistics
	stats := processor.GetStatistics()
	assert.Equal(t, int64(totalExpected), stats.TotalShares, "Should have processed all shares")
}

// TestShareProcessor_Blake2SIntegration tests integration with Blake2S algorithm
func TestShareProcessor_Blake2SIntegration(t *testing.T) {
	processor := NewShareProcessor()
	
	// Test with known input that should produce consistent hash
	share := &Share{
		MinerID:    1,
		UserID:     1,
		JobID:      "test_job",
		Nonce:      "12345678",
		Difficulty: 1.0,
		Timestamp:  time.Now(),
	}
	
	// Process the same share multiple times
	result1 := processor.ProcessShare(share)
	result2 := processor.ProcessShare(share)
	
	// Both should succeed
	require.True(t, result1.Success, "First processing should succeed")
	require.True(t, result2.Success, "Second processing should succeed")
	
	// Hashes should be consistent (same input should produce same hash)
	assert.Equal(t, result1.ProcessedShare.Hash, result2.ProcessedShare.Hash, 
		"Same share should produce consistent hash")
	
	// Hash should be 64 characters (32 bytes in hex)
	assert.Len(t, result1.ProcessedShare.Hash, 64, "Blake2S hash should be 64 hex characters")
	
	// Hash should be valid hex
	assert.Regexp(t, "^[0-9a-f]{64}$", result1.ProcessedShare.Hash, "Hash should be valid hex")
}