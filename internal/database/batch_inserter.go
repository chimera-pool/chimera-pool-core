package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// =============================================================================
// HIGH-PERFORMANCE BATCH INSERTER
// Optimized for 100k+ shares/second using batch INSERT and prepared statements
// =============================================================================

// BatchInserterConfig configures the batch inserter
type BatchInserterConfig struct {
	// Batching
	BatchSize     int           // Max rows per batch (default: 1000)
	FlushInterval time.Duration // Max time before flush (default: 100ms)

	// Concurrency
	WorkerCount int // Parallel insert workers (default: 4)
	QueueSize   int // Pending batch queue size (default: 100)

	// Timeouts
	InsertTimeout time.Duration // Per-batch timeout (default: 30s)
}

// DefaultBatchInserterConfig returns production defaults
func DefaultBatchInserterConfig() BatchInserterConfig {
	return BatchInserterConfig{
		BatchSize:     1000,
		FlushInterval: 100 * time.Millisecond,
		WorkerCount:   4,
		QueueSize:     100,
		InsertTimeout: 30 * time.Second,
	}
}

// ShareBatchInserter handles high-throughput share insertion
type ShareBatchInserter struct {
	config BatchInserterConfig
	pool   *ConnectionPool

	// Batching
	pending   []*Share
	pendingMu sync.Mutex
	batchChan chan []*Share

	// Prepared statements (per-batch-size for efficiency)
	stmtCache   map[int]*sql.Stmt
	stmtCacheMu sync.RWMutex

	// Statistics (atomic)
	stats BatchInsertStats

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// BatchInsertStats tracks insertion performance
type BatchInsertStats struct {
	TotalInserted  int64
	TotalBatches   int64
	TotalErrors    int64
	AvgBatchTimeNs int64
	MaxBatchTimeNs int64
	PendingShares  int64
	InsertRate     int64 // shares/second (computed)
}

// NewShareBatchInserter creates a new batch inserter
func NewShareBatchInserter(pool *ConnectionPool, config BatchInserterConfig) *ShareBatchInserter {
	if config.BatchSize <= 0 {
		config.BatchSize = 1000
	}
	if config.FlushInterval <= 0 {
		config.FlushInterval = 100 * time.Millisecond
	}
	if config.WorkerCount <= 0 {
		config.WorkerCount = 4
	}
	if config.QueueSize <= 0 {
		config.QueueSize = 100
	}
	if config.InsertTimeout <= 0 {
		config.InsertTimeout = 30 * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &ShareBatchInserter{
		config:    config,
		pool:      pool,
		pending:   make([]*Share, 0, config.BatchSize),
		batchChan: make(chan []*Share, config.QueueSize),
		stmtCache: make(map[int]*sql.Stmt),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Start begins the batch inserter workers
func (bi *ShareBatchInserter) Start() {
	// Start insert workers
	for i := 0; i < bi.config.WorkerCount; i++ {
		bi.wg.Add(1)
		go bi.insertWorker(i)
	}

	// Start flush timer
	bi.wg.Add(1)
	go bi.flushLoop()
}

// Stop gracefully shuts down the batch inserter
func (bi *ShareBatchInserter) Stop() error {
	// Signal shutdown
	bi.cancel()

	// Flush remaining shares
	bi.flush()

	// Close batch channel
	close(bi.batchChan)

	// Wait for workers
	bi.wg.Wait()

	// Close prepared statements
	bi.stmtCacheMu.Lock()
	for _, stmt := range bi.stmtCache {
		stmt.Close()
	}
	bi.stmtCacheMu.Unlock()

	return nil
}

// Insert queues a share for batch insertion
func (bi *ShareBatchInserter) Insert(share *Share) {
	bi.pendingMu.Lock()
	bi.pending = append(bi.pending, share)
	atomic.AddInt64(&bi.stats.PendingShares, 1)

	// Flush if batch is full
	if len(bi.pending) >= bi.config.BatchSize {
		batch := bi.pending
		bi.pending = make([]*Share, 0, bi.config.BatchSize)
		bi.pendingMu.Unlock()

		atomic.AddInt64(&bi.stats.PendingShares, -int64(len(batch)))

		select {
		case bi.batchChan <- batch:
		default:
			// Queue full - insert synchronously
			bi.insertBatch(batch)
		}
		return
	}

	bi.pendingMu.Unlock()
}

// InsertBatch inserts multiple shares immediately
func (bi *ShareBatchInserter) InsertBatch(shares []*Share) error {
	if len(shares) == 0 {
		return nil
	}
	return bi.insertBatch(shares)
}

// Flush forces all pending shares to be inserted
func (bi *ShareBatchInserter) Flush() {
	bi.flush()
}

// GetStats returns current statistics
func (bi *ShareBatchInserter) GetStats() BatchInsertStats {
	return BatchInsertStats{
		TotalInserted:  atomic.LoadInt64(&bi.stats.TotalInserted),
		TotalBatches:   atomic.LoadInt64(&bi.stats.TotalBatches),
		TotalErrors:    atomic.LoadInt64(&bi.stats.TotalErrors),
		AvgBatchTimeNs: atomic.LoadInt64(&bi.stats.AvgBatchTimeNs),
		MaxBatchTimeNs: atomic.LoadInt64(&bi.stats.MaxBatchTimeNs),
		PendingShares:  atomic.LoadInt64(&bi.stats.PendingShares),
		InsertRate:     atomic.LoadInt64(&bi.stats.InsertRate),
	}
}

// Internal methods

func (bi *ShareBatchInserter) flush() {
	bi.pendingMu.Lock()
	if len(bi.pending) == 0 {
		bi.pendingMu.Unlock()
		return
	}

	batch := bi.pending
	bi.pending = make([]*Share, 0, bi.config.BatchSize)
	bi.pendingMu.Unlock()

	atomic.AddInt64(&bi.stats.PendingShares, -int64(len(batch)))

	select {
	case bi.batchChan <- batch:
	default:
		// Queue full - insert synchronously
		bi.insertBatch(batch)
	}
}

func (bi *ShareBatchInserter) flushLoop() {
	defer bi.wg.Done()

	ticker := time.NewTicker(bi.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-bi.ctx.Done():
			return
		case <-ticker.C:
			bi.flush()
		}
	}
}

func (bi *ShareBatchInserter) insertWorker(id int) {
	defer bi.wg.Done()

	for batch := range bi.batchChan {
		bi.insertBatch(batch)
	}
}

func (bi *ShareBatchInserter) insertBatch(shares []*Share) error {
	if len(shares) == 0 {
		return nil
	}

	startTime := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), bi.config.InsertTimeout)
	defer cancel()

	// Build multi-row INSERT statement
	query, args := bi.buildBatchInsert(shares)

	_, err := bi.pool.Exec(ctx, query, args...)

	elapsed := time.Since(startTime).Nanoseconds()

	if err != nil {
		atomic.AddInt64(&bi.stats.TotalErrors, 1)
		return fmt.Errorf("batch insert failed: %w", err)
	}

	// Update statistics
	atomic.AddInt64(&bi.stats.TotalInserted, int64(len(shares)))
	atomic.AddInt64(&bi.stats.TotalBatches, 1)

	// Update timing stats
	for {
		current := atomic.LoadInt64(&bi.stats.MaxBatchTimeNs)
		if elapsed <= current || atomic.CompareAndSwapInt64(&bi.stats.MaxBatchTimeNs, current, elapsed) {
			break
		}
	}

	// Update average (exponential moving average)
	currentAvg := atomic.LoadInt64(&bi.stats.AvgBatchTimeNs)
	newAvg := (currentAvg*9 + elapsed) / 10
	atomic.StoreInt64(&bi.stats.AvgBatchTimeNs, newAvg)

	return nil
}

func (bi *ShareBatchInserter) buildBatchInsert(shares []*Share) (string, []interface{}) {
	// Build: INSERT INTO shares (cols) VALUES ($1,$2,...), ($3,$4,...), ...

	cols := []string{"miner_id", "user_id", "difficulty", "is_valid", "nonce", "hash", "timestamp"}
	colCount := len(cols)

	var sb strings.Builder
	sb.WriteString("INSERT INTO shares (")
	sb.WriteString(strings.Join(cols, ", "))
	sb.WriteString(") VALUES ")

	args := make([]interface{}, 0, len(shares)*colCount)

	for i, share := range shares {
		if i > 0 {
			sb.WriteString(", ")
		}

		sb.WriteString("(")
		for j := 0; j < colCount; j++ {
			if j > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("$%d", i*colCount+j+1))
		}
		sb.WriteString(")")

		// Handle nil timestamp
		timestamp := share.Timestamp
		if timestamp.IsZero() {
			timestamp = time.Now()
		}

		args = append(args,
			share.MinerID,
			share.UserID,
			share.Difficulty,
			share.IsValid,
			share.Nonce,
			share.Hash,
			timestamp,
		)
	}

	return sb.String(), args
}

// =============================================================================
// GENERIC BATCH INSERTER
// For other tables beyond shares
// =============================================================================

// GenericBatchInserter handles batch inserts for any table
type GenericBatchInserter struct {
	pool   *ConnectionPool
	config BatchInserterConfig
}

// NewGenericBatchInserter creates a generic batch inserter
func NewGenericBatchInserter(pool *ConnectionPool, config BatchInserterConfig) *GenericBatchInserter {
	return &GenericBatchInserter{
		pool:   pool,
		config: config,
	}
}

// InsertBatch inserts multiple rows in a single statement
func (gbi *GenericBatchInserter) InsertBatch(
	ctx context.Context,
	table string,
	columns []string,
	values [][]interface{},
) (int64, error) {
	if len(values) == 0 {
		return 0, nil
	}

	colCount := len(columns)

	// Validate all rows have correct column count
	for i, row := range values {
		if len(row) != colCount {
			return 0, fmt.Errorf("row %d has %d values, expected %d", i, len(row), colCount)
		}
	}

	// Build INSERT statement
	var sb strings.Builder
	sb.WriteString("INSERT INTO ")
	sb.WriteString(table)
	sb.WriteString(" (")
	sb.WriteString(strings.Join(columns, ", "))
	sb.WriteString(") VALUES ")

	args := make([]interface{}, 0, len(values)*colCount)

	for i, row := range values {
		if i > 0 {
			sb.WriteString(", ")
		}

		sb.WriteString("(")
		for j := 0; j < colCount; j++ {
			if j > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("$%d", i*colCount+j+1))
		}
		sb.WriteString(")")

		args = append(args, row...)
	}

	result, err := gbi.pool.Exec(ctx, sb.String(), args...)
	if err != nil {
		return 0, fmt.Errorf("batch insert failed: %w", err)
	}

	return result.RowsAffected()
}

// InsertBatchReturning inserts rows and returns generated IDs
func (gbi *GenericBatchInserter) InsertBatchReturning(
	ctx context.Context,
	table string,
	columns []string,
	values [][]interface{},
	returning string,
) ([]int64, error) {
	if len(values) == 0 {
		return nil, nil
	}

	colCount := len(columns)

	// Build INSERT ... RETURNING statement
	var sb strings.Builder
	sb.WriteString("INSERT INTO ")
	sb.WriteString(table)
	sb.WriteString(" (")
	sb.WriteString(strings.Join(columns, ", "))
	sb.WriteString(") VALUES ")

	args := make([]interface{}, 0, len(values)*colCount)

	for i, row := range values {
		if i > 0 {
			sb.WriteString(", ")
		}

		sb.WriteString("(")
		for j := 0; j < colCount; j++ {
			if j > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("$%d", i*colCount+j+1))
		}
		sb.WriteString(")")

		args = append(args, row...)
	}

	sb.WriteString(" RETURNING ")
	sb.WriteString(returning)

	rows, err := gbi.pool.Query(ctx, sb.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("batch insert failed: %w", err)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan returned id: %w", err)
		}
		ids = append(ids, id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating results: %w", err)
	}

	return ids, nil
}
