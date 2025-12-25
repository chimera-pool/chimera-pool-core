package payouts

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// =============================================================================
// ISP-COMPLIANT INTERFACES FOR PAYOUT EXECUTION
// =============================================================================

// BlockNotifier notifies subscribers when blocks are found
type BlockNotifier interface {
	Subscribe(handler func(block *Block))
}

// PayoutQueue manages pending payouts awaiting transaction
type PayoutQueue interface {
	Enqueue(payout PendingPayout) error
	GetPending() []PendingPayout
	MarkProcessed(payoutID int64) error
}

// UserSettingsProvider retrieves user payout preferences
type UserSettingsProvider interface {
	GetUserPayoutSettings(userID int64) (*UserPayoutSettings, error)
}

// ShareProvider retrieves shares for payout calculation
type ShareProvider interface {
	GetSharesInWindow(ctx context.Context, endTime time.Time, windowSize int64) ([]Share, error)
}

// BalanceTracker manages user balance accounting
type BalanceTracker interface {
	GetBalance(userID int64) int64
	AddToBalance(userID int64, amount int64) error
	DeductFromBalance(userID int64, amount int64) error
}

// =============================================================================
// PAYOUT STATUS AND TYPES
// =============================================================================

// PayoutStatus represents the status of a pending payout
type PayoutStatus string

const (
	PayoutStatusPending   PayoutStatus = "pending"
	PayoutStatusProcessed PayoutStatus = "processed"
	PayoutStatusFailed    PayoutStatus = "failed"
	PayoutStatusCancelled PayoutStatus = "cancelled"
)

// PendingPayout represents a payout awaiting transaction
type PendingPayout struct {
	ID           int64        `json:"id"`
	UserID       int64        `json:"user_id"`
	Amount       int64        `json:"amount"`
	Address      string       `json:"address"`
	Status       PayoutStatus `json:"status"`
	PayoutMode   PayoutMode   `json:"payout_mode"`
	BlockID      int64        `json:"block_id"`
	CreatedAt    time.Time    `json:"created_at"`
	ProcessedAt  *time.Time   `json:"processed_at,omitempty"`
	TxHash       string       `json:"tx_hash,omitempty"`
	ErrorMessage string       `json:"error_message,omitempty"`
}

// ExecutorStats holds statistics about payout processing
type ExecutorStats struct {
	BlocksProcessed        int64     `json:"blocks_processed"`
	TotalPayoutsCalculated int64     `json:"total_payouts_calculated"`
	TotalPayoutsQueued     int64     `json:"total_payouts_queued"`
	TotalAmountCredited    int64     `json:"total_amount_credited"`
	LastBlockProcessed     time.Time `json:"last_block_processed"`
	ErrorCount             int64     `json:"error_count"`
}

// Errors
var (
	ErrInsufficientBalance = errors.New("insufficient balance")
	ErrBlockNotConfirmed   = errors.New("block not confirmed")
	ErrNoShares            = errors.New("no shares in window")
)

// =============================================================================
// PAYOUT EXECUTOR IMPLEMENTATION
// =============================================================================

// PayoutExecutor orchestrates payout calculation and queuing
type PayoutExecutor struct {
	config           *PayoutConfig
	calculators      map[PayoutMode]PayoutCalculator
	notifier         BlockNotifier
	queue            PayoutQueue
	settingsProvider UserSettingsProvider
	shareProvider    ShareProvider
	balanceTracker   BalanceTracker

	// Stats
	stats ExecutorStats
	mu    sync.RWMutex

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewPayoutExecutor creates a new payout executor with all dependencies
func NewPayoutExecutor(
	config *PayoutConfig,
	notifier BlockNotifier,
	queue PayoutQueue,
	settingsProvider UserSettingsProvider,
	shareProvider ShareProvider,
	balanceTracker BalanceTracker,
) *PayoutExecutor {
	if config == nil || notifier == nil || queue == nil ||
		settingsProvider == nil || shareProvider == nil || balanceTracker == nil {
		return nil
	}

	// Initialize calculators directly
	calculators := make(map[PayoutMode]PayoutCalculator)

	// PPLNS Calculator
	if config.EnablePPLNS {
		if calc, err := NewPPLNSCalculator(config.PPLNSWindowSize, config.FeePPLNS); err == nil {
			calculators[PayoutModePPLNS] = calc
		}
	}

	// SLICE Calculator
	if config.EnableSLICE {
		if calc, err := NewSLICECalculator(
			config.SLICEWindowSize,
			config.SLICESliceDuration,
			config.SLICEDecayFactor,
			config.FeeSLICE,
		); err == nil {
			calculators[PayoutModeSLICE] = calc
		}
	}

	// PPS Calculator
	if config.EnablePPS {
		if calc, err := NewPPSCalculator(config.FeePPS); err == nil {
			calculators[PayoutModePPS] = calc
		}
	}

	// SCORE Calculator (uses PPLNS window size)
	if config.EnableSCORE {
		if calc, err := NewSCORECalculator(config.PPLNSWindowSize, config.FeeSCORE, config.SCOREDecayFactor); err == nil {
			calculators[PayoutModeSCORE] = calc
		}
	}

	// SOLO Calculator
	if config.EnableSOLO {
		if calc, err := NewSOLOCalculator(config.FeeSOLO); err == nil {
			calculators[PayoutModeSOLO] = calc
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &PayoutExecutor{
		config:           config,
		calculators:      calculators,
		notifier:         notifier,
		queue:            queue,
		settingsProvider: settingsProvider,
		shareProvider:    shareProvider,
		balanceTracker:   balanceTracker,
		ctx:              ctx,
		cancel:           cancel,
	}
}

// Start begins listening for block events
func (e *PayoutExecutor) Start() {
	e.notifier.Subscribe(func(block *Block) {
		e.wg.Add(1)
		go func() {
			defer e.wg.Done()
			if err := e.ProcessBlock(e.ctx, block); err != nil {
				atomic.AddInt64(&e.stats.ErrorCount, 1)
			}
		}()
	})
}

// Stop gracefully shuts down the executor
func (e *PayoutExecutor) Stop() {
	e.cancel()
	e.wg.Wait()
}

// ProcessBlock calculates and credits payouts for a confirmed block
func (e *PayoutExecutor) ProcessBlock(ctx context.Context, block *Block) error {
	if block.Status != "confirmed" {
		return fmt.Errorf("%w: status=%s", ErrBlockNotConfirmed, block.Status)
	}

	// Get shares in the payout window
	windowSize := e.config.PPLNSWindowSize
	shares, err := e.shareProvider.GetSharesInWindow(ctx, block.Timestamp, windowSize)
	if err != nil {
		return fmt.Errorf("failed to get shares: %w", err)
	}

	if len(shares) == 0 {
		// No shares, nothing to pay out
		return nil
	}

	// Group shares by user and their payout mode
	userShares := make(map[int64][]Share)
	userModes := make(map[int64]PayoutMode)

	for _, share := range shares {
		userShares[share.UserID] = append(userShares[share.UserID], share)

		// Get user's payout mode if not already fetched
		if _, exists := userModes[share.UserID]; !exists {
			settings, err := e.settingsProvider.GetUserPayoutSettings(share.UserID)
			if err != nil {
				userModes[share.UserID] = PayoutModePPLNS // Default
			} else {
				userModes[share.UserID] = settings.PayoutMode
			}
		}
	}

	// Calculate payouts using available calculator
	// For simplicity, we'll use PPLNS calculator for all users in this block
	// A more sophisticated implementation would group by mode
	calculator, ok := e.calculators[PayoutModePPLNS]
	if !ok {
		calculator, ok = e.calculators[PayoutModeSLICE]
	}

	if !ok || calculator == nil {
		return fmt.Errorf("no calculator available")
	}

	payouts, err := calculator.CalculatePayouts(shares, block.Reward, 0, block.Timestamp)
	if err != nil {
		return fmt.Errorf("failed to calculate payouts: %w", err)
	}

	// Credit balances and check for auto-payout triggering
	var totalCredited int64
	var payoutsQueued int64

	for _, payout := range payouts {
		// Credit user balance
		if err := e.balanceTracker.AddToBalance(payout.UserID, payout.Amount); err != nil {
			continue
		}
		totalCredited += payout.Amount

		// Check if auto-payout should be triggered
		settings, err := e.settingsProvider.GetUserPayoutSettings(payout.UserID)
		if err != nil {
			continue
		}

		if settings.AutoPayoutEnable && settings.PayoutAddress != "" {
			balance := e.balanceTracker.GetBalance(payout.UserID)
			if balance >= settings.MinPayoutAmount {
				// Queue payout
				pending := PendingPayout{
					ID:         time.Now().UnixNano(), // Simple ID generation
					UserID:     payout.UserID,
					Amount:     balance,
					Address:    settings.PayoutAddress,
					Status:     PayoutStatusPending,
					PayoutMode: settings.PayoutMode,
					BlockID:    block.ID,
					CreatedAt:  time.Now(),
				}

				if err := e.queue.Enqueue(pending); err == nil {
					// Deduct from balance (will be returned if tx fails)
					e.balanceTracker.DeductFromBalance(payout.UserID, balance)
					payoutsQueued++
				}
			}
		}
	}

	// Update stats
	e.mu.Lock()
	e.stats.BlocksProcessed++
	e.stats.TotalPayoutsCalculated += int64(len(payouts))
	e.stats.TotalPayoutsQueued += payoutsQueued
	e.stats.TotalAmountCredited += totalCredited
	e.stats.LastBlockProcessed = time.Now()
	e.mu.Unlock()

	return nil
}

// GetStats returns current executor statistics
func (e *PayoutExecutor) GetStats() ExecutorStats {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.stats
}

// GetPendingPayouts returns all pending payouts
func (e *PayoutExecutor) GetPendingPayouts() []PendingPayout {
	return e.queue.GetPending()
}

// GetUserBalance returns a user's current balance
func (e *PayoutExecutor) GetUserBalance(userID int64) int64 {
	return e.balanceTracker.GetBalance(userID)
}
