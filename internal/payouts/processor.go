package payouts

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// =============================================================================
// ISP-COMPLIANT INTERFACES FOR PAYOUT PROCESSING
// =============================================================================

// WalletClient handles cryptocurrency transactions
type WalletClient interface {
	SendTransaction(ctx context.Context, address string, amount int64) (txHash string, err error)
	GetBalance(ctx context.Context) (int64, error)
	ValidateAddress(address string) bool
}

// PayoutRepository manages payout persistence
type PayoutRepository interface {
	GetPendingPayouts(ctx context.Context, limit int) ([]PendingPayout, error)
	MarkPayoutComplete(ctx context.Context, payoutID int64, txHash string) error
	MarkPayoutFailed(ctx context.Context, payoutID int64, errorMsg string) error
	ReturnToBalance(ctx context.Context, userID int64, amount int64) error
}

// =============================================================================
// PROCESSOR CONFIGURATION
// =============================================================================

// ProcessorConfig holds configuration for the payout processor
type ProcessorConfig struct {
	BatchSize       int           `json:"batch_size" yaml:"batch_size"`
	ProcessInterval time.Duration `json:"process_interval" yaml:"process_interval"`
	MaxRetries      int           `json:"max_retries" yaml:"max_retries"`
	MinPayoutAmount int64         `json:"min_payout_amount" yaml:"min_payout_amount"`
}

// DefaultProcessorConfig returns sensible defaults
func DefaultProcessorConfig() ProcessorConfig {
	return ProcessorConfig{
		BatchSize:       10,
		ProcessInterval: time.Minute,
		MaxRetries:      3,
		MinPayoutAmount: 100000, // 0.001 LTC
	}
}

// ProcessorStats holds statistics about payout processing
type ProcessorStats struct {
	PayoutsProcessed int64     `json:"payouts_processed"`
	PayoutsFailed    int64     `json:"payouts_failed"`
	TotalAmountSent  int64     `json:"total_amount_sent"`
	LastProcessedAt  time.Time `json:"last_processed_at"`
	IsRunning        bool      `json:"is_running"`
}

// =============================================================================
// PAYOUT PROCESSOR IMPLEMENTATION
// =============================================================================

// PayoutProcessor handles automatic payout processing
type PayoutProcessor struct {
	wallet WalletClient
	repo   PayoutRepository
	config ProcessorConfig

	// Stats
	stats ProcessorStats
	mu    sync.RWMutex

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewPayoutProcessor creates a new payout processor
func NewPayoutProcessor(wallet WalletClient, repo PayoutRepository, config ProcessorConfig) *PayoutProcessor {
	if wallet == nil || repo == nil {
		return nil
	}

	if config.BatchSize <= 0 {
		config.BatchSize = 10
	}
	if config.ProcessInterval <= 0 {
		config.ProcessInterval = time.Minute
	}
	if config.MaxRetries <= 0 {
		config.MaxRetries = 3
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &PayoutProcessor{
		wallet: wallet,
		repo:   repo,
		config: config,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start begins automatic payout processing
func (p *PayoutProcessor) Start() {
	p.mu.Lock()
	p.stats.IsRunning = true
	p.mu.Unlock()

	p.wg.Add(1)
	go p.processLoop()
}

// Stop gracefully stops the processor
func (p *PayoutProcessor) Stop() {
	p.cancel()
	p.wg.Wait()

	p.mu.Lock()
	p.stats.IsRunning = false
	p.mu.Unlock()
}

// processLoop runs the main processing loop
func (p *PayoutProcessor) processLoop() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.config.ProcessInterval)
	defer ticker.Stop()

	// Process immediately on start
	_ = p.ProcessPendingPayouts(p.ctx)

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			_ = p.ProcessPendingPayouts(p.ctx)
		}
	}
}

// ProcessPendingPayouts processes a batch of pending payouts
func (p *PayoutProcessor) ProcessPendingPayouts(ctx context.Context) error {
	// Get pending payouts
	payouts, err := p.repo.GetPendingPayouts(ctx, p.config.BatchSize)
	if err != nil {
		return fmt.Errorf("failed to get pending payouts: %w", err)
	}

	if len(payouts) == 0 {
		return nil
	}

	for _, payout := range payouts {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			p.processSinglePayout(ctx, payout)
		}
	}

	return nil
}

// processSinglePayout handles a single payout
func (p *PayoutProcessor) processSinglePayout(ctx context.Context, payout PendingPayout) {
	// Check minimum payout amount
	if p.config.MinPayoutAmount > 0 && payout.Amount < p.config.MinPayoutAmount {
		// Skip - amount too low (don't mark as failed, leave for later accumulation)
		return
	}

	// Validate address
	if !p.wallet.ValidateAddress(payout.Address) {
		p.handleFailedPayout(ctx, payout, "invalid address")
		return
	}

	// Attempt to send transaction
	txHash, err := p.wallet.SendTransaction(ctx, payout.Address, payout.Amount)
	if err != nil {
		p.handleFailedPayout(ctx, payout, err.Error())
		return
	}

	// Mark as complete
	if err := p.repo.MarkPayoutComplete(ctx, payout.ID, txHash); err != nil {
		// Log error but transaction was sent
		return
	}

	// Update stats
	p.mu.Lock()
	p.stats.PayoutsProcessed++
	p.stats.TotalAmountSent += payout.Amount
	p.stats.LastProcessedAt = time.Now()
	p.mu.Unlock()
}

// handleFailedPayout handles a failed payout
func (p *PayoutProcessor) handleFailedPayout(ctx context.Context, payout PendingPayout, errorMsg string) {
	// Mark payout as failed
	_ = p.repo.MarkPayoutFailed(ctx, payout.ID, errorMsg)

	// Return funds to user balance
	_ = p.repo.ReturnToBalance(ctx, payout.UserID, payout.Amount)

	// Update stats
	atomic.AddInt64(&p.stats.PayoutsFailed, 1)
}

// GetStats returns current processor statistics
func (p *PayoutProcessor) GetStats() ProcessorStats {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.stats
}

// GetWalletBalance returns the current wallet balance
func (p *PayoutProcessor) GetWalletBalance(ctx context.Context) (int64, error) {
	return p.wallet.GetBalance(ctx)
}

// IsRunning returns whether the processor is running
func (p *PayoutProcessor) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.stats.IsRunning
}
