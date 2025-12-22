package database

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBatchInserterConfig_Defaults(t *testing.T) {
	config := DefaultBatchInserterConfig()

	assert.Equal(t, 1000, config.BatchSize)
	assert.Equal(t, 100*time.Millisecond, config.FlushInterval)
	assert.Equal(t, 4, config.WorkerCount)
	assert.Equal(t, 100, config.QueueSize)
	assert.Equal(t, 30*time.Second, config.InsertTimeout)
}

func TestShareBatchInserter_BuildBatchInsert(t *testing.T) {
	// Create inserter with mock pool (we only test query building)
	config := DefaultBatchInserterConfig()
	bi := &ShareBatchInserter{
		config: config,
	}

	shares := []*Share{
		{
			MinerID:    1,
			UserID:     10,
			Difficulty: 1000.5,
			IsValid:    true,
			Nonce:      "abc123",
			Hash:       "def456",
			Timestamp:  time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
		},
		{
			MinerID:    2,
			UserID:     20,
			Difficulty: 2000.5,
			IsValid:    false,
			Nonce:      "xyz789",
			Hash:       "ghi012",
			Timestamp:  time.Date(2025, 1, 1, 13, 0, 0, 0, time.UTC),
		},
	}

	query, args := bi.buildBatchInsert(shares)

	// Verify query structure
	assert.Contains(t, query, "INSERT INTO shares")
	assert.Contains(t, query, "miner_id, user_id, difficulty, is_valid, nonce, hash, timestamp")
	assert.Contains(t, query, "VALUES")
	assert.Contains(t, query, "$1")
	assert.Contains(t, query, "$14") // 2 rows * 7 columns = 14 params

	// Verify args count
	assert.Len(t, args, 14)

	// Verify first row values
	assert.Equal(t, int64(1), args[0])
	assert.Equal(t, int64(10), args[1])
	assert.Equal(t, 1000.5, args[2])
	assert.Equal(t, true, args[3])
	assert.Equal(t, "abc123", args[4])
	assert.Equal(t, "def456", args[5])

	// Verify second row values
	assert.Equal(t, int64(2), args[7])
	assert.Equal(t, int64(20), args[8])
}

func TestShareBatchInserter_BuildBatchInsert_EmptyTimestamp(t *testing.T) {
	config := DefaultBatchInserterConfig()
	bi := &ShareBatchInserter{
		config: config,
	}

	shares := []*Share{
		{
			MinerID:    1,
			UserID:     10,
			Difficulty: 1000.0,
			IsValid:    true,
			Nonce:      "abc",
			Hash:       "def",
			// Timestamp is zero
		},
	}

	_, args := bi.buildBatchInsert(shares)

	// Timestamp should be set to current time (not zero)
	timestamp, ok := args[6].(time.Time)
	require.True(t, ok)
	assert.False(t, timestamp.IsZero())
}

func TestShareBatchInserter_Stats(t *testing.T) {
	config := DefaultBatchInserterConfig()
	bi := &ShareBatchInserter{
		config: config,
	}

	// Initial stats should be zero
	stats := bi.GetStats()
	assert.Equal(t, int64(0), stats.TotalInserted)
	assert.Equal(t, int64(0), stats.TotalBatches)
	assert.Equal(t, int64(0), stats.TotalErrors)
}

func TestGenericBatchInserter_InsertBatch_ValidationError(t *testing.T) {
	gbi := &GenericBatchInserter{}

	// Rows with mismatched column count should error
	columns := []string{"a", "b", "c"}
	values := [][]interface{}{
		{1, 2, 3},
		{4, 5}, // Missing value
	}

	_, err := gbi.InsertBatch(context.Background(), "test", columns, values)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "row 1 has 2 values, expected 3")
}

func TestGenericBatchInserter_InsertBatch_EmptyValues(t *testing.T) {
	gbi := &GenericBatchInserter{}

	// Empty values should return 0, nil
	count, err := gbi.InsertBatch(context.Background(), "test", []string{"a"}, nil)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestBatchInsertStats(t *testing.T) {
	stats := BatchInsertStats{
		TotalInserted:  1000,
		TotalBatches:   10,
		TotalErrors:    2,
		AvgBatchTimeNs: 1000000,
		MaxBatchTimeNs: 5000000,
		PendingShares:  50,
		InsertRate:     10000,
	}

	assert.Equal(t, int64(1000), stats.TotalInserted)
	assert.Equal(t, int64(10), stats.TotalBatches)
	assert.Equal(t, int64(2), stats.TotalErrors)
	assert.Equal(t, int64(1000000), stats.AvgBatchTimeNs)
	assert.Equal(t, int64(5000000), stats.MaxBatchTimeNs)
	assert.Equal(t, int64(50), stats.PendingShares)
	assert.Equal(t, int64(10000), stats.InsertRate)
}

// Benchmark tests for batch insert query building
func BenchmarkBuildBatchInsert_10(b *testing.B) {
	benchmarkBuildBatchInsert(b, 10)
}

func BenchmarkBuildBatchInsert_100(b *testing.B) {
	benchmarkBuildBatchInsert(b, 100)
}

func BenchmarkBuildBatchInsert_1000(b *testing.B) {
	benchmarkBuildBatchInsert(b, 1000)
}

func benchmarkBuildBatchInsert(b *testing.B, count int) {
	config := DefaultBatchInserterConfig()
	bi := &ShareBatchInserter{
		config: config,
	}

	shares := make([]*Share, count)
	for i := 0; i < count; i++ {
		shares[i] = &Share{
			MinerID:    int64(i),
			UserID:     int64(i % 100),
			Difficulty: 1000.0,
			IsValid:    true,
			Nonce:      "abcd1234",
			Hash:       "deadbeef",
			Timestamp:  time.Now(),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bi.buildBatchInsert(shares)
	}
}
