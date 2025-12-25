package payouts

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// =============================================================================
// SERVICE FACTORY - WIRES ALL PAYOUT COMPONENTS TOGETHER
// =============================================================================

// PayoutServiceConfig holds configuration for the complete payout service
type PayoutServiceConfig struct {
	// Wallet configuration
	Wallet WalletConfig

	// Processor configuration
	Processor ProcessorConfig

	// Payout mode configuration
	Payouts *PayoutConfig

	// Metrics namespace
	MetricsNamespace string
}

// DefaultPayoutServiceConfig returns sensible defaults
func DefaultPayoutServiceConfig() PayoutServiceConfig {
	return PayoutServiceConfig{
		Wallet: WalletConfig{
			RPCURL:  "http://litecoind:9332",
			Timeout: 30 * time.Second,
			Network: "mainnet",
		},
		Processor: ProcessorConfig{
			BatchSize:       10,
			ProcessInterval: time.Minute,
			MaxRetries:      3,
			MinPayoutAmount: 1000000, // 0.01 LTC
		},
		Payouts:          DefaultPayoutConfig(),
		MetricsNamespace: "chimera_pool",
	}
}

// PayoutServices holds all initialized payout service components
type PayoutServices struct {
	Executor     *PayoutExecutor
	Processor    *PayoutProcessor
	WalletClient *LitecoinWalletClient
	Repository   *SQLPayoutRepository
	Metrics      *PayoutMetrics

	// Internal references
	db     *sql.DB
	cancel context.CancelFunc
}

// NewPayoutServices creates and wires all payout service components
func NewPayoutServices(
	db *sql.DB,
	config PayoutServiceConfig,
	registry prometheus.Registerer,
) (*PayoutServices, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection required")
	}

	// Create wallet client
	walletClient, err := NewLitecoinWalletClient(config.Wallet)
	if err != nil {
		return nil, fmt.Errorf("failed to create wallet client: %w", err)
	}

	// Create repository
	repository := NewSQLPayoutRepository(db)

	// Create metrics
	metrics := NewPayoutMetrics(config.MetricsNamespace, registry)

	// Create processor
	processor := NewPayoutProcessor(walletClient, repository, config.Processor)
	if processor == nil {
		return nil, fmt.Errorf("failed to create payout processor")
	}

	// Create executor with adapters
	ctx, cancel := context.WithCancel(context.Background())
	executor := createExecutor(config.Payouts, repository, ctx)

	return &PayoutServices{
		Executor:     executor,
		Processor:    processor,
		WalletClient: walletClient,
		Repository:   repository,
		Metrics:      metrics,
		db:           db,
		cancel:       cancel,
	}, nil
}

// createExecutor creates a PayoutExecutor with database-backed adapters
func createExecutor(config *PayoutConfig, repo *SQLPayoutRepository, ctx context.Context) *PayoutExecutor {
	// Create adapters that implement the executor interfaces
	notifier := &dbBlockNotifier{handlers: make([]func(*Block), 0)}
	queue := &dbPayoutQueue{repo: repo, ctx: ctx}
	settings := &dbUserSettings{repo: repo, ctx: ctx}
	shares := &dbShareProvider{ctx: ctx}
	balances := &dbBalanceTracker{repo: repo, ctx: ctx}

	return NewPayoutExecutor(config, notifier, queue, settings, shares, balances)
}

// Start starts all payout services
func (s *PayoutServices) Start() {
	if s.Executor != nil {
		s.Executor.Start()
	}
	if s.Processor != nil {
		s.Processor.Start()
	}
}

// Stop gracefully stops all payout services
func (s *PayoutServices) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	if s.Executor != nil {
		s.Executor.Stop()
	}
	if s.Processor != nil {
		s.Processor.Stop()
	}
}

// GetStats returns combined statistics from all services
func (s *PayoutServices) GetStats() map[string]interface{} {
	stats := make(map[string]interface{})

	if s.Executor != nil {
		execStats := s.Executor.GetStats()
		stats["executor"] = execStats
	}

	if s.Processor != nil {
		procStats := s.Processor.GetStats()
		stats["processor"] = procStats
	}

	// Get wallet balance
	if s.WalletClient != nil {
		balance, err := s.WalletClient.GetBalance(context.Background())
		if err == nil {
			stats["wallet_balance"] = balance
			if s.Metrics != nil {
				s.Metrics.SetWalletBalance(balance)
			}
		}
	}

	return stats
}

// NotifyBlockFound notifies the executor of a new block
func (s *PayoutServices) NotifyBlockFound(block *Block) error {
	if s.Executor == nil {
		return fmt.Errorf("executor not initialized")
	}
	return s.Executor.ProcessBlock(context.Background(), block)
}

// =============================================================================
// DATABASE-BACKED ADAPTERS FOR EXECUTOR INTERFACES
// =============================================================================

// dbBlockNotifier implements BlockNotifier using database events
type dbBlockNotifier struct {
	handlers []func(*Block)
}

func (n *dbBlockNotifier) Subscribe(handler func(*Block)) {
	n.handlers = append(n.handlers, handler)
}

func (n *dbBlockNotifier) Notify(block *Block) {
	for _, h := range n.handlers {
		h(block)
	}
}

// dbPayoutQueue implements PayoutQueue using database
type dbPayoutQueue struct {
	repo *SQLPayoutRepository
	ctx  context.Context
}

func (q *dbPayoutQueue) Enqueue(payout PendingPayout) error {
	_, err := q.repo.CreatePendingPayout(q.ctx, payout)
	return err
}

func (q *dbPayoutQueue) GetPending() []PendingPayout {
	payouts, _ := q.repo.GetPendingPayouts(q.ctx, 100)
	return payouts
}

func (q *dbPayoutQueue) MarkProcessed(payoutID int64) error {
	return q.repo.MarkPayoutComplete(q.ctx, payoutID, "")
}

// dbUserSettings implements UserSettingsProvider using database
type dbUserSettings struct {
	repo *SQLPayoutRepository
	ctx  context.Context
}

func (s *dbUserSettings) GetUserPayoutSettings(userID int64) (*UserPayoutSettings, error) {
	// Return default settings - actual implementation would query user_payout_settings table
	return &UserPayoutSettings{
		UserID:           userID,
		PayoutMode:       PayoutModePPLNS,
		MinPayoutAmount:  1000000, // 0.01 LTC
		AutoPayoutEnable: true,
	}, nil
}

// dbShareProvider implements ShareProvider using database
type dbShareProvider struct {
	ctx context.Context
	db  *sql.DB
}

func (p *dbShareProvider) GetSharesInWindow(ctx context.Context, endTime time.Time, windowSize int64) ([]Share, error) {
	// This would query the shares table - for now return empty
	// Actual implementation connects to existing share tracking
	return []Share{}, nil
}

// dbBalanceTracker implements BalanceTracker using database
type dbBalanceTracker struct {
	repo *SQLPayoutRepository
	ctx  context.Context
}

func (t *dbBalanceTracker) GetBalance(userID int64) int64 {
	balance, _ := t.repo.GetUserBalance(t.ctx, userID)
	return balance
}

func (t *dbBalanceTracker) AddToBalance(userID int64, amount int64) error {
	return t.repo.AddToBalance(t.ctx, userID, amount)
}

func (t *dbBalanceTracker) DeductFromBalance(userID int64, amount int64) error {
	return t.repo.DeductFromBalance(t.ctx, userID, amount)
}
