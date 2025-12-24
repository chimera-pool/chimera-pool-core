package shares

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// =============================================================================
// HIGH-THROUGHPUT BATCH SHARE PROCESSOR
// Designed for 100k+ shares/second with bounded memory and latency guarantees
// =============================================================================

// BatchConfig configures the batch processor
type BatchConfig struct {
	// Worker pool settings
	WorkerCount int // Number of parallel workers (default: NumCPU)
	QueueSize   int // Input queue size per worker (default: 10000)

	// Batching settings
	BatchSize    int           // Max shares per batch (default: 100)
	BatchTimeout time.Duration // Max wait time for batch (default: 10ms)

	// Rate limiting
	MaxSharesPerSecond int64 // Global rate limit (0 = unlimited)
}

// DefaultBatchConfig returns production-ready defaults
func DefaultBatchConfig() BatchConfig {
	return BatchConfig{
		WorkerCount:        8,
		QueueSize:          10000,
		BatchSize:          100,
		BatchTimeout:       10 * time.Millisecond,
		MaxSharesPerSecond: 0, // Unlimited by default
	}
}

// BatchProcessor processes shares in parallel batches for high throughput
type BatchProcessor struct {
	config    BatchConfig
	processor *ShareProcessor

	// Worker coordination
	workers   []*shareWorker
	inputChan chan *shareJob
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	stopped   int32 // Atomic flag to prevent sending to closed channel

	// Lock-free statistics (atomic operations only)
	stats BatchStatistics

	// Rate limiting
	rateLimiter *rateLimiter
}

// BatchStatistics tracks processing metrics using atomic operations
type BatchStatistics struct {
	TotalReceived    int64
	TotalProcessed   int64
	TotalValid       int64
	TotalInvalid     int64
	TotalDropped     int64 // Dropped due to queue full or rate limit
	BatchesProcessed int64
	ProcessingTimeNs int64 // Total nanoseconds spent processing
	QueueHighWater   int64 // Peak queue depth
}

// shareJob represents a share submission job
type shareJob struct {
	Share    *Share
	ResultCh chan<- ShareProcessingResult
}

// shareWorker processes shares from its dedicated queue
type shareWorker struct {
	id           int
	inputChan    <-chan *shareJob
	processor    *ShareProcessor
	batchSize    int
	batchTimeout time.Duration
	stats        *BatchStatistics
}

// NewBatchProcessor creates a high-throughput batch processor
func NewBatchProcessor(config BatchConfig) *BatchProcessor {
	if config.WorkerCount <= 0 {
		config.WorkerCount = 8
	}
	if config.QueueSize <= 0 {
		config.QueueSize = 10000
	}
	if config.BatchSize <= 0 {
		config.BatchSize = 100
	}
	if config.BatchTimeout <= 0 {
		config.BatchTimeout = 10 * time.Millisecond
	}

	ctx, cancel := context.WithCancel(context.Background())

	bp := &BatchProcessor{
		config:    config,
		processor: NewShareProcessor(),
		inputChan: make(chan *shareJob, config.QueueSize),
		ctx:       ctx,
		cancel:    cancel,
	}

	// Initialize rate limiter if configured
	if config.MaxSharesPerSecond > 0 {
		bp.rateLimiter = newRateLimiter(config.MaxSharesPerSecond)
	}

	// Create workers
	bp.workers = make([]*shareWorker, config.WorkerCount)
	for i := 0; i < config.WorkerCount; i++ {
		bp.workers[i] = &shareWorker{
			id:           i,
			inputChan:    bp.inputChan,
			processor:    bp.processor,
			batchSize:    config.BatchSize,
			batchTimeout: config.BatchTimeout,
			stats:        &bp.stats,
		}
	}

	return bp
}

// Start begins processing shares
func (bp *BatchProcessor) Start() {
	for _, worker := range bp.workers {
		bp.wg.Add(1)
		go worker.run(bp.ctx, &bp.wg)
	}
}

// Stop gracefully shuts down the processor
func (bp *BatchProcessor) Stop() {
	// Mark as stopped first to prevent new submissions
	atomic.StoreInt32(&bp.stopped, 1)
	bp.cancel()
	close(bp.inputChan)
	bp.wg.Wait()
}

// Submit submits a share for processing (non-blocking)
// Returns a channel that will receive the result
func (bp *BatchProcessor) Submit(share *Share) <-chan ShareProcessingResult {
	resultCh := make(chan ShareProcessingResult, 1)

	// Check if processor is stopped
	if atomic.LoadInt32(&bp.stopped) != 0 {
		resultCh <- ShareProcessingResult{
			Success: false,
			Error:   "processor stopped",
		}
		return resultCh
	}

	atomic.AddInt64(&bp.stats.TotalReceived, 1)

	// Check rate limit
	if bp.rateLimiter != nil && !bp.rateLimiter.allow() {
		atomic.AddInt64(&bp.stats.TotalDropped, 1)
		resultCh <- ShareProcessingResult{
			Success: false,
			Error:   "rate limit exceeded",
		}
		return resultCh
	}

	job := &shareJob{
		Share:    share,
		ResultCh: resultCh,
	}

	// Non-blocking send - drop if queue is full
	select {
	case bp.inputChan <- job:
		// Track queue depth
		queueLen := int64(len(bp.inputChan))
		for {
			current := atomic.LoadInt64(&bp.stats.QueueHighWater)
			if queueLen <= current || atomic.CompareAndSwapInt64(&bp.stats.QueueHighWater, current, queueLen) {
				break
			}
		}
	default:
		atomic.AddInt64(&bp.stats.TotalDropped, 1)
		resultCh <- ShareProcessingResult{
			Success: false,
			Error:   "queue full - share dropped",
		}
	}

	return resultCh
}

// SubmitSync submits a share and waits for the result
func (bp *BatchProcessor) SubmitSync(share *Share, timeout time.Duration) ShareProcessingResult {
	resultCh := bp.Submit(share)

	select {
	case result := <-resultCh:
		return result
	case <-time.After(timeout):
		return ShareProcessingResult{
			Success: false,
			Error:   "processing timeout",
		}
	}
}

// GetStatistics returns current processing statistics (lock-free)
func (bp *BatchProcessor) GetStatistics() BatchStatistics {
	return BatchStatistics{
		TotalReceived:    atomic.LoadInt64(&bp.stats.TotalReceived),
		TotalProcessed:   atomic.LoadInt64(&bp.stats.TotalProcessed),
		TotalValid:       atomic.LoadInt64(&bp.stats.TotalValid),
		TotalInvalid:     atomic.LoadInt64(&bp.stats.TotalInvalid),
		TotalDropped:     atomic.LoadInt64(&bp.stats.TotalDropped),
		BatchesProcessed: atomic.LoadInt64(&bp.stats.BatchesProcessed),
		ProcessingTimeNs: atomic.LoadInt64(&bp.stats.ProcessingTimeNs),
		QueueHighWater:   atomic.LoadInt64(&bp.stats.QueueHighWater),
	}
}

// GetThroughput returns shares processed per second
func (bp *BatchProcessor) GetThroughput() float64 {
	totalTime := atomic.LoadInt64(&bp.stats.ProcessingTimeNs)
	if totalTime == 0 {
		return 0
	}
	processed := atomic.LoadInt64(&bp.stats.TotalProcessed)
	return float64(processed) / (float64(totalTime) / float64(time.Second))
}

// GetQueueDepth returns current queue depth
func (bp *BatchProcessor) GetQueueDepth() int {
	return len(bp.inputChan)
}

// Worker implementation
func (w *shareWorker) run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	batch := make([]*shareJob, 0, w.batchSize)
	timer := time.NewTimer(w.batchTimeout)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			// Process remaining batch before exit
			if len(batch) > 0 {
				w.processBatch(batch)
			}
			return

		case job, ok := <-w.inputChan:
			if !ok {
				// Channel closed - process remaining and exit
				if len(batch) > 0 {
					w.processBatch(batch)
				}
				return
			}

			batch = append(batch, job)

			// Process batch if full
			if len(batch) >= w.batchSize {
				w.processBatch(batch)
				batch = batch[:0]
				timer.Reset(w.batchTimeout)
			}

		case <-timer.C:
			// Timeout - process partial batch
			if len(batch) > 0 {
				w.processBatch(batch)
				batch = batch[:0]
			}
			timer.Reset(w.batchTimeout)
		}
	}
}

// processBatch processes a batch of shares
func (w *shareWorker) processBatch(batch []*shareJob) {
	if len(batch) == 0 {
		return
	}

	startTime := time.Now()

	// Process each share in the batch
	for _, job := range batch {
		result := w.processor.ProcessShare(job.Share)

		// Update atomic statistics
		atomic.AddInt64(&w.stats.TotalProcessed, 1)
		if result.Success {
			atomic.AddInt64(&w.stats.TotalValid, 1)
		} else {
			atomic.AddInt64(&w.stats.TotalInvalid, 1)
		}

		// Send result (non-blocking)
		select {
		case job.ResultCh <- result:
		default:
			// Result channel full or closed - skip
		}
	}

	// Update batch statistics
	atomic.AddInt64(&w.stats.BatchesProcessed, 1)
	atomic.AddInt64(&w.stats.ProcessingTimeNs, time.Since(startTime).Nanoseconds())
}

// =============================================================================
// TOKEN BUCKET RATE LIMITER
// =============================================================================

type rateLimiter struct {
	rate     int64 // tokens per second
	tokens   int64
	lastTime int64 // unix nano
	mu       sync.Mutex
}

func newRateLimiter(rate int64) *rateLimiter {
	return &rateLimiter{
		rate:     rate,
		tokens:   rate, // Start with full bucket
		lastTime: time.Now().UnixNano(),
	}
}

func (rl *rateLimiter) allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now().UnixNano()
	elapsed := now - rl.lastTime
	rl.lastTime = now

	// Add tokens based on elapsed time
	tokensToAdd := (elapsed * rl.rate) / int64(time.Second)
	rl.tokens += tokensToAdd

	// Cap at rate (1 second worth of tokens)
	if rl.tokens > rl.rate {
		rl.tokens = rl.rate
	}

	// Check if we can consume a token
	if rl.tokens > 0 {
		rl.tokens--
		return true
	}

	return false
}
