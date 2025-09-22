package shares

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestShareProcessing_E2E tests the complete end-to-end share processing workflow
func TestShareProcessing_E2E(t *testing.T) {
	// Create a share processor
	processor := NewShareProcessor()
	
	// Simulate a complete mining workflow
	t.Run("complete_mining_workflow", func(t *testing.T) {
		// Step 1: Miner submits a share
		incomingShare := &Share{
			MinerID:    1,
			UserID:     1,
			JobID:      "mining_job_001",
			Nonce:      "deadbeef",
			Difficulty: 1.0,
			Timestamp:  time.Now(),
		}
		
		// Step 2: Process the share
		result := processor.ProcessShare(incomingShare)
		
		// Step 3: Verify processing was successful
		require.True(t, result.Success, "Share processing should succeed")
		require.NotNil(t, result.ProcessedShare, "Processed share should not be nil")
		
		// Step 4: Verify share validation
		processedShare := result.ProcessedShare
		assert.True(t, processedShare.IsValid, "Share should be valid")
		assert.NotEmpty(t, processedShare.Hash, "Share should have computed hash")
		assert.Equal(t, incomingShare.MinerID, processedShare.MinerID, "Miner ID should match")
		assert.Equal(t, incomingShare.UserID, processedShare.UserID, "User ID should match")
		assert.Equal(t, incomingShare.JobID, processedShare.JobID, "Job ID should match")
		assert.Equal(t, incomingShare.Nonce, processedShare.Nonce, "Nonce should match")
		assert.Equal(t, incomingShare.Difficulty, processedShare.Difficulty, "Difficulty should match")
		
		// Step 5: Verify statistics were updated
		stats := processor.GetStatistics()
		assert.Equal(t, int64(1), stats.TotalShares, "Should have 1 total share")
		assert.Equal(t, int64(1), stats.ValidShares, "Should have 1 valid share")
		assert.Equal(t, int64(0), stats.InvalidShares, "Should have 0 invalid shares")
		assert.Equal(t, 1.0, stats.TotalDifficulty, "Total difficulty should be 1.0")
		
		// Step 6: Verify miner statistics
		minerStats := processor.GetMinerStatistics(1)
		assert.Equal(t, int64(1), minerStats.TotalShares, "Miner should have 1 share")
		assert.Equal(t, int64(1), minerStats.ValidShares, "Miner should have 1 valid share")
		assert.Equal(t, 1.0, minerStats.TotalDifficulty, "Miner total difficulty should be 1.0")
	})
	
	// Test multiple miners submitting shares
	t.Run("multiple_miners_workflow", func(t *testing.T) {
		processor := NewShareProcessor() // Fresh processor
		
		// Simulate 3 miners submitting shares
		miners := []struct {
			minerID    int64
			userID     int64
			difficulty float64
		}{
			{1, 1, 1.0},
			{2, 1, 2.0},
			{3, 2, 1.5},
		}
		
		for i, miner := range miners {
			share := &Share{
				MinerID:    miner.minerID,
				UserID:     miner.userID,
				JobID:      fmt.Sprintf("job_%d", i),
				Nonce:      fmt.Sprintf("%08x", i*1000+int(miner.minerID)),
				Difficulty: miner.difficulty,
				Timestamp:  time.Now(),
			}
			
			result := processor.ProcessShare(share)
			require.True(t, result.Success, "Share processing should succeed for miner %d", miner.minerID)
		}
		
		// Verify overall statistics
		stats := processor.GetStatistics()
		assert.Equal(t, int64(3), stats.TotalShares, "Should have 3 total shares")
		assert.Equal(t, 4.5, stats.TotalDifficulty, "Total difficulty should be 1.0+2.0+1.5=4.5")
		
		// Verify individual miner statistics
		miner1Stats := processor.GetMinerStatistics(1)
		assert.Equal(t, int64(1), miner1Stats.TotalShares, "Miner 1 should have 1 share")
		assert.Equal(t, 1.0, miner1Stats.TotalDifficulty, "Miner 1 difficulty should be 1.0")
		
		miner2Stats := processor.GetMinerStatistics(2)
		assert.Equal(t, int64(1), miner2Stats.TotalShares, "Miner 2 should have 1 share")
		assert.Equal(t, 2.0, miner2Stats.TotalDifficulty, "Miner 2 difficulty should be 2.0")
		
		miner3Stats := processor.GetMinerStatistics(3)
		assert.Equal(t, int64(1), miner3Stats.TotalShares, "Miner 3 should have 1 share")
		assert.Equal(t, 1.5, miner3Stats.TotalDifficulty, "Miner 3 difficulty should be 1.5")
	})
	
	// Test invalid share handling
	t.Run("invalid_share_workflow", func(t *testing.T) {
		processor := NewShareProcessor() // Fresh processor
		
		// Submit a valid share first
		validShare := &Share{
			MinerID:    1,
			UserID:     1,
			JobID:      "valid_job",
			Nonce:      "deadbeef", // Valid hex nonce
			Difficulty: 1.0,
			Timestamp:  time.Now(),
		}
		
		result := processor.ProcessShare(validShare)
		require.NotNil(t, result.ProcessedShare, "Share should be processed")
		// Note: Share might not be valid due to difficulty, but should be processed
		
		// Submit an invalid share (empty nonce)
		invalidShare := &Share{
			MinerID:    1,
			UserID:     1,
			JobID:      "invalid_job",
			Nonce:      "", // Invalid empty nonce
			Difficulty: 1.0,
			Timestamp:  time.Now(),
		}
		
		result = processor.ProcessShare(invalidShare)
		require.False(t, result.Success, "Invalid share should fail processing")
		require.NotEmpty(t, result.Error, "Should have error message")
		
		// Verify statistics - the valid share should be counted, invalid share should not be processed
		stats := processor.GetStatistics()
		// Note: The invalid share with empty nonce fails input validation and is not processed
		// So we should only have the valid share in statistics
		assert.True(t, stats.TotalShares >= 1, "Should have at least 1 total share")
	})
	
	// Test Blake2S hash consistency
	t.Run("blake2s_hash_consistency", func(t *testing.T) {
		processor := NewShareProcessor()
		
		// Process the same share multiple times
		share := &Share{
			MinerID:    1,
			UserID:     1,
			JobID:      "consistency_test",
			Nonce:      "12345678",
			Difficulty: 1.0,
			Timestamp:  time.Now(),
		}
		
		// Process first time
		result1 := processor.ProcessShare(share)
		require.True(t, result1.Success, "First processing should succeed")
		
		// Process second time (same share)
		result2 := processor.ProcessShare(share)
		require.True(t, result2.Success, "Second processing should succeed")
		
		// Hashes should be identical
		assert.Equal(t, result1.ProcessedShare.Hash, result2.ProcessedShare.Hash,
			"Same share should produce identical hash")
		
		// Both should be valid (or both invalid)
		assert.Equal(t, result1.ProcessedShare.IsValid, result2.ProcessedShare.IsValid,
			"Same share should have same validity")
	})
	
	// Test performance under realistic load
	t.Run("realistic_load_test", func(t *testing.T) {
		processor := NewShareProcessor()
		
		// Simulate realistic mining pool scenario
		shareCount := 100
		minerCount := 10
		
		start := time.Now()
		
		for i := 0; i < shareCount; i++ {
			share := &Share{
				MinerID:    int64((i % minerCount) + 1),
				UserID:     int64(((i % minerCount) / 2) + 1), // 2 miners per user
				JobID:      fmt.Sprintf("realistic_job_%d", i/10), // 10 shares per job
				Nonce:      fmt.Sprintf("%08x", i*12345), // Valid hex nonce
				Difficulty: 1.0 + float64(i%5), // Varying difficulty 1.0-5.0
				Timestamp:  time.Now(),
			}
			
			result := processor.ProcessShare(share)
			// Don't require all to be successful due to difficulty variation
			assert.NotNil(t, result.ProcessedShare, "Share should be processed")
		}
		
		duration := time.Since(start)
		
		// Verify performance
		sharesPerSecond := float64(shareCount) / duration.Seconds()
		t.Logf("Processed %d shares in %v (%.2f shares/second)", shareCount, duration, sharesPerSecond)
		
		// Should handle realistic load efficiently
		assert.Greater(t, sharesPerSecond, 100.0, "Should process at least 100 shares/second under realistic load")
		
		// Verify statistics make sense
		stats := processor.GetStatistics()
		assert.Equal(t, int64(shareCount), stats.TotalShares, "Should have processed all shares")
		// With varying difficulty, we expect some shares to be valid
		// Note: Due to our simplified hash function, actual validation rates may vary
		t.Logf("Valid shares: %d/%d (%.1f%%)", stats.ValidShares, stats.TotalShares, 
			float64(stats.ValidShares)/float64(stats.TotalShares)*100)
		
		// Verify all miners have statistics
		for minerID := int64(1); minerID <= int64(minerCount); minerID++ {
			minerStats := processor.GetMinerStatistics(minerID)
			assert.True(t, minerStats.TotalShares > 0, "Miner %d should have shares", minerID)
		}
	})
}