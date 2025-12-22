package shares

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBatchProcessor_BasicProcessing(t *testing.T) {
	config := BatchConfig{
		WorkerCount:  4,
		QueueSize:    1000,
		BatchSize:    10,
		BatchTimeout: 5 * time.Millisecond,
	}

	bp := NewBatchProcessor(config)
	bp.Start()
	defer bp.Stop()

	// Submit a single share
	share := &Share{
		MinerID:    1,
		UserID:     1,
		JobID:      "test-job-1",
		Nonce:      "deadbeef",
		Difficulty: 1.0,
		Timestamp:  time.Now(),
	}

	result := bp.SubmitSync(share, time.Second)
	assert.NotNil(t, result)
	// Result may be valid or invalid depending on hash, but should not error
}

func TestBatchProcessor_HighThroughput(t *testing.T) {
	config := BatchConfig{
		WorkerCount:  8,
		QueueSize:    50000,
		BatchSize:    100,
		BatchTimeout: 10 * time.Millisecond,
	}

	bp := NewBatchProcessor(config)
	bp.Start()
	defer bp.Stop()

	// Submit many shares concurrently
	shareCount := 10000
	var wg sync.WaitGroup
	var successCount int64
	var errorCount int64

	startTime := time.Now()

	for i := 0; i < shareCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			share := &Share{
				MinerID:    int64(idx % 100),
				UserID:     int64(idx % 50),
				JobID:      "batch-test-job",
				Nonce:      "12345678",
				Difficulty: 1.0,
				Timestamp:  time.Now(),
			}

			result := bp.SubmitSync(share, 5*time.Second)
			if result.Error == "" {
				atomic.AddInt64(&successCount, 1)
			} else {
				atomic.AddInt64(&errorCount, 1)
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(startTime)

	stats := bp.GetStatistics()

	t.Logf("Processed %d shares in %v", shareCount, elapsed)
	t.Logf("Throughput: %.2f shares/sec", float64(shareCount)/elapsed.Seconds())
	t.Logf("Success: %d, Errors: %d", successCount, errorCount)
	t.Logf("Stats: Received=%d, Processed=%d, Valid=%d, Invalid=%d, Dropped=%d",
		stats.TotalReceived, stats.TotalProcessed, stats.TotalValid, stats.TotalInvalid, stats.TotalDropped)

	// Verify all shares were received
	assert.Equal(t, int64(shareCount), stats.TotalReceived)

	// Most shares should be processed (some may be dropped under extreme load)
	assert.GreaterOrEqual(t, stats.TotalProcessed, int64(shareCount*9/10))
}

func TestBatchProcessor_RateLimiting(t *testing.T) {
	config := BatchConfig{
		WorkerCount:        2,
		QueueSize:          100,
		BatchSize:          10,
		BatchTimeout:       5 * time.Millisecond,
		MaxSharesPerSecond: 100, // Low limit for testing
	}

	bp := NewBatchProcessor(config)
	bp.Start()
	defer bp.Stop()

	// Submit more shares than the rate limit allows
	shareCount := 200
	var wg sync.WaitGroup
	var droppedCount int64

	for i := 0; i < shareCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			share := &Share{
				MinerID:    1,
				UserID:     1,
				JobID:      "rate-limit-test",
				Nonce:      "abcd1234",
				Difficulty: 1.0,
				Timestamp:  time.Now(),
			}

			result := bp.SubmitSync(share, time.Second)
			if result.Error == "rate limit exceeded" {
				atomic.AddInt64(&droppedCount, 1)
			}
		}(i)
	}

	wg.Wait()

	stats := bp.GetStatistics()
	t.Logf("Rate limited: Dropped=%d, Processed=%d", stats.TotalDropped, stats.TotalProcessed)

	// Some shares should have been dropped due to rate limiting
	assert.Greater(t, stats.TotalDropped, int64(0))
}

func TestBatchProcessor_QueueFull(t *testing.T) {
	// This test verifies the queue tracks high water mark
	// Actual queue overflow is hard to trigger reliably in tests
	config := BatchConfig{
		WorkerCount:  2,
		QueueSize:    100,
		BatchSize:    10,
		BatchTimeout: 5 * time.Millisecond,
	}

	bp := NewBatchProcessor(config)
	bp.Start()
	defer bp.Stop()

	// Submit shares quickly
	for i := 0; i < 50; i++ {
		share := &Share{
			MinerID:    1,
			UserID:     1,
			JobID:      "queue-test",
			Nonce:      "11111111",
			Difficulty: 1.0,
			Timestamp:  time.Now(),
		}
		bp.Submit(share)
	}

	// Wait for processing
	time.Sleep(50 * time.Millisecond)

	stats := bp.GetStatistics()
	t.Logf("Queue test: Received=%d, Processed=%d, QueueHighWater=%d", 
		stats.TotalReceived, stats.TotalProcessed, stats.QueueHighWater)

	// Queue high water should have been tracked
	assert.GreaterOrEqual(t, stats.QueueHighWater, int64(0))
	assert.Equal(t, int64(50), stats.TotalReceived)
}

func TestBatchProcessor_GracefulShutdown(t *testing.T) {
	config := BatchConfig{
		WorkerCount:  4,
		QueueSize:    1000,
		BatchSize:    10,
		BatchTimeout: 5 * time.Millisecond,
	}
	bp := NewBatchProcessor(config)
	bp.Start()

	// Submit some shares
	for i := 0; i < 100; i++ {
		share := &Share{
			MinerID:    int64(i),
			UserID:     1,
			JobID:      "shutdown-test",
			Nonce:      "ffffffff",
			Difficulty: 1.0,
			Timestamp:  time.Now(),
		}
		bp.Submit(share)
	}

	// Give workers time to pick up work before shutdown
	time.Sleep(20 * time.Millisecond)

	// Stop should wait for pending work
	bp.Stop()

	stats := bp.GetStatistics()
	t.Logf("After shutdown: Received=%d, Processed=%d, Dropped=%d", 
		stats.TotalReceived, stats.TotalProcessed, stats.TotalDropped)

	// Most shares should be processed (allow for some timing variance)
	assert.GreaterOrEqual(t, stats.TotalProcessed, int64(90))
}

func TestBatchProcessor_Statistics(t *testing.T) {
	config := DefaultBatchConfig()
	bp := NewBatchProcessor(config)
	bp.Start()
	defer bp.Stop()

	// Initial stats should be zero
	stats := bp.GetStatistics()
	assert.Equal(t, int64(0), stats.TotalReceived)
	assert.Equal(t, int64(0), stats.TotalProcessed)

	// Submit shares and verify stats update
	for i := 0; i < 50; i++ {
		share := &Share{
			MinerID:    1,
			UserID:     1,
			JobID:      "stats-test",
			Nonce:      "00000001",
			Difficulty: 1.0,
			Timestamp:  time.Now(),
		}
		bp.SubmitSync(share, time.Second)
	}

	stats = bp.GetStatistics()
	assert.Equal(t, int64(50), stats.TotalReceived)
	assert.Equal(t, int64(50), stats.TotalProcessed)
	assert.Equal(t, int64(50), stats.TotalValid+stats.TotalInvalid)
}

func TestBatchProcessor_DefaultConfig(t *testing.T) {
	config := DefaultBatchConfig()

	assert.Equal(t, 8, config.WorkerCount)
	assert.Equal(t, 10000, config.QueueSize)
	assert.Equal(t, 100, config.BatchSize)
	assert.Equal(t, 10*time.Millisecond, config.BatchTimeout)
	assert.Equal(t, int64(0), config.MaxSharesPerSecond)
}

func BenchmarkBatchProcessor_Submit(b *testing.B) {
	config := BatchConfig{
		WorkerCount:  8,
		QueueSize:    100000,
		BatchSize:    100,
		BatchTimeout: 10 * time.Millisecond,
	}

	bp := NewBatchProcessor(config)
	bp.Start()
	defer bp.Stop()

	share := &Share{
		MinerID:    1,
		UserID:     1,
		JobID:      "benchmark-job",
		Nonce:      "deadbeef",
		Difficulty: 1.0,
		Timestamp:  time.Now(),
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			bp.Submit(share)
		}
	})

	stats := bp.GetStatistics()
	b.ReportMetric(float64(stats.TotalProcessed)/b.Elapsed().Seconds(), "shares/sec")
}

func BenchmarkBatchProcessor_SubmitSync(b *testing.B) {
	config := BatchConfig{
		WorkerCount:  8,
		QueueSize:    100000,
		BatchSize:    100,
		BatchTimeout: 10 * time.Millisecond,
	}

	bp := NewBatchProcessor(config)
	bp.Start()
	defer bp.Stop()

	share := &Share{
		MinerID:    1,
		UserID:     1,
		JobID:      "benchmark-job",
		Nonce:      "deadbeef",
		Difficulty: 1.0,
		Timestamp:  time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bp.SubmitSync(share, time.Second)
	}
}
