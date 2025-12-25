package payouts

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// MOCK WALLET CLIENT FOR TESTING
// =============================================================================

type MockWalletClient struct {
	transactions []MockTransaction
	failNextSend bool
	sendDelay    time.Duration
	mu           sync.RWMutex
}

type MockTransaction struct {
	TxHash  string
	Address string
	Amount  int64
	SentAt  time.Time
}

func NewMockWalletClient() *MockWalletClient {
	return &MockWalletClient{
		transactions: make([]MockTransaction, 0),
	}
}

func (m *MockWalletClient) SendTransaction(ctx context.Context, address string, amount int64) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failNextSend {
		m.failNextSend = false
		return "", errors.New("transaction failed")
	}

	if m.sendDelay > 0 {
		time.Sleep(m.sendDelay)
	}

	txHash := "tx_" + time.Now().Format("20060102150405.000")
	m.transactions = append(m.transactions, MockTransaction{
		TxHash:  txHash,
		Address: address,
		Amount:  amount,
		SentAt:  time.Now(),
	})

	return txHash, nil
}

func (m *MockWalletClient) GetBalance(ctx context.Context) (int64, error) {
	return 100000000000, nil // 1000 LTC
}

func (m *MockWalletClient) ValidateAddress(address string) bool {
	return len(address) > 10 && (address[:3] == "ltc" || address[:1] == "L" || address[:1] == "M")
}

func (m *MockWalletClient) SetFailNextSend() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failNextSend = true
}

func (m *MockWalletClient) GetTransactions() []MockTransaction {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]MockTransaction, len(m.transactions))
	copy(result, m.transactions)
	return result
}

// =============================================================================
// MOCK PAYOUT REPOSITORY FOR TESTING
// =============================================================================

type MockPayoutRepository struct {
	pendingPayouts   []PendingPayout
	completedPayouts []PendingPayout
	failedPayouts    []PendingPayout
	mu               sync.RWMutex
	nextID           int64
}

func NewMockPayoutRepository() *MockPayoutRepository {
	return &MockPayoutRepository{
		pendingPayouts:   make([]PendingPayout, 0),
		completedPayouts: make([]PendingPayout, 0),
		failedPayouts:    make([]PendingPayout, 0),
		nextID:           1,
	}
}

func (m *MockPayoutRepository) GetPendingPayouts(ctx context.Context, limit int) ([]PendingPayout, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]PendingPayout, 0)
	for i, p := range m.pendingPayouts {
		if i >= limit {
			break
		}
		result = append(result, p)
	}
	return result, nil
}

func (m *MockPayoutRepository) MarkPayoutComplete(ctx context.Context, payoutID int64, txHash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, p := range m.pendingPayouts {
		if p.ID == payoutID {
			p.Status = PayoutStatusProcessed
			p.TxHash = txHash
			now := time.Now()
			p.ProcessedAt = &now
			m.completedPayouts = append(m.completedPayouts, p)
			m.pendingPayouts = append(m.pendingPayouts[:i], m.pendingPayouts[i+1:]...)
			return nil
		}
	}
	return errors.New("payout not found")
}

func (m *MockPayoutRepository) MarkPayoutFailed(ctx context.Context, payoutID int64, errorMsg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, p := range m.pendingPayouts {
		if p.ID == payoutID {
			p.Status = PayoutStatusFailed
			p.ErrorMessage = errorMsg
			m.failedPayouts = append(m.failedPayouts, p)
			m.pendingPayouts = append(m.pendingPayouts[:i], m.pendingPayouts[i+1:]...)
			return nil
		}
	}
	return errors.New("payout not found")
}

func (m *MockPayoutRepository) ReturnToBalance(ctx context.Context, userID int64, amount int64) error {
	return nil
}

func (m *MockPayoutRepository) AddPendingPayout(payout PendingPayout) {
	m.mu.Lock()
	defer m.mu.Unlock()
	payout.ID = m.nextID
	m.nextID++
	payout.Status = PayoutStatusPending
	m.pendingPayouts = append(m.pendingPayouts, payout)
}

func (m *MockPayoutRepository) GetCompletedCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.completedPayouts)
}

func (m *MockPayoutRepository) GetFailedCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.failedPayouts)
}

// =============================================================================
// PAYOUT PROCESSOR TESTS (TDD)
// =============================================================================

func TestPayoutProcessor_Creation(t *testing.T) {
	t.Run("creates processor with dependencies", func(t *testing.T) {
		wallet := NewMockWalletClient()
		repo := NewMockPayoutRepository()

		processor := NewPayoutProcessor(wallet, repo, ProcessorConfig{
			BatchSize:       10,
			ProcessInterval: time.Second,
			MaxRetries:      3,
		})

		assert.NotNil(t, processor)
	})

	t.Run("returns nil with nil dependencies", func(t *testing.T) {
		processor := NewPayoutProcessor(nil, nil, ProcessorConfig{})
		assert.Nil(t, processor)
	})
}

func TestPayoutProcessor_ProcessSinglePayout(t *testing.T) {
	t.Run("successfully processes payout", func(t *testing.T) {
		wallet := NewMockWalletClient()
		repo := NewMockPayoutRepository()

		processor := NewPayoutProcessor(wallet, repo, ProcessorConfig{
			BatchSize:       10,
			ProcessInterval: time.Second,
			MaxRetries:      3,
		})

		// Add a pending payout
		repo.AddPendingPayout(PendingPayout{
			UserID:  1,
			Amount:  1000000, // 0.01 LTC
			Address: "ltc1qtest123456789",
		})

		// Process the payout
		err := processor.ProcessPendingPayouts(context.Background())
		require.NoError(t, err)

		// Verify transaction was sent
		txs := wallet.GetTransactions()
		assert.Len(t, txs, 1)
		assert.Equal(t, int64(1000000), txs[0].Amount)

		// Verify payout marked complete
		assert.Equal(t, 1, repo.GetCompletedCount())
	})

	t.Run("handles failed transaction", func(t *testing.T) {
		wallet := NewMockWalletClient()
		repo := NewMockPayoutRepository()

		processor := NewPayoutProcessor(wallet, repo, ProcessorConfig{
			BatchSize:       10,
			ProcessInterval: time.Second,
			MaxRetries:      1, // Only 1 retry for quick test
		})

		// Add a pending payout
		repo.AddPendingPayout(PendingPayout{
			UserID:  1,
			Amount:  1000000,
			Address: "ltc1qtest123456789",
		})

		// Make wallet fail
		wallet.SetFailNextSend()

		// Process the payout
		err := processor.ProcessPendingPayouts(context.Background())
		assert.NoError(t, err) // Processor handles errors gracefully

		// Verify payout marked failed
		assert.Equal(t, 1, repo.GetFailedCount())
		assert.Equal(t, 0, repo.GetCompletedCount())
	})

	t.Run("validates address before sending", func(t *testing.T) {
		wallet := NewMockWalletClient()
		repo := NewMockPayoutRepository()

		processor := NewPayoutProcessor(wallet, repo, ProcessorConfig{
			BatchSize:       10,
			ProcessInterval: time.Second,
			MaxRetries:      3,
		})

		// Add payout with invalid address
		repo.AddPendingPayout(PendingPayout{
			UserID:  1,
			Amount:  1000000,
			Address: "invalid",
		})

		err := processor.ProcessPendingPayouts(context.Background())
		assert.NoError(t, err)

		// Verify no transaction sent
		txs := wallet.GetTransactions()
		assert.Len(t, txs, 0)

		// Verify payout marked failed
		assert.Equal(t, 1, repo.GetFailedCount())
	})
}

func TestPayoutProcessor_BatchProcessing(t *testing.T) {
	t.Run("processes multiple payouts in batch", func(t *testing.T) {
		wallet := NewMockWalletClient()
		repo := NewMockPayoutRepository()

		processor := NewPayoutProcessor(wallet, repo, ProcessorConfig{
			BatchSize:       5,
			ProcessInterval: time.Second,
			MaxRetries:      3,
		})

		// Add multiple pending payouts
		for i := 0; i < 3; i++ {
			repo.AddPendingPayout(PendingPayout{
				UserID:  int64(i + 1),
				Amount:  int64(1000000 * (i + 1)),
				Address: "ltc1qtest" + string(rune('a'+i)) + "123456",
			})
		}

		err := processor.ProcessPendingPayouts(context.Background())
		require.NoError(t, err)

		// Verify all transactions sent
		txs := wallet.GetTransactions()
		assert.Len(t, txs, 3)

		// Verify all marked complete
		assert.Equal(t, 3, repo.GetCompletedCount())
	})

	t.Run("respects batch size limit", func(t *testing.T) {
		wallet := NewMockWalletClient()
		repo := NewMockPayoutRepository()

		processor := NewPayoutProcessor(wallet, repo, ProcessorConfig{
			BatchSize:       2, // Only process 2 at a time
			ProcessInterval: time.Second,
			MaxRetries:      3,
		})

		// Add 5 pending payouts
		for i := 0; i < 5; i++ {
			repo.AddPendingPayout(PendingPayout{
				UserID:  int64(i + 1),
				Amount:  1000000,
				Address: "ltc1qtest" + string(rune('a'+i)) + "123456",
			})
		}

		// First batch
		err := processor.ProcessPendingPayouts(context.Background())
		require.NoError(t, err)
		assert.Equal(t, 2, repo.GetCompletedCount())

		// Second batch
		err = processor.ProcessPendingPayouts(context.Background())
		require.NoError(t, err)
		assert.Equal(t, 4, repo.GetCompletedCount())

		// Third batch
		err = processor.ProcessPendingPayouts(context.Background())
		require.NoError(t, err)
		assert.Equal(t, 5, repo.GetCompletedCount())
	})
}

func TestPayoutProcessor_AutoProcessing(t *testing.T) {
	t.Run("starts and stops auto-processing", func(t *testing.T) {
		wallet := NewMockWalletClient()
		repo := NewMockPayoutRepository()

		processor := NewPayoutProcessor(wallet, repo, ProcessorConfig{
			BatchSize:       10,
			ProcessInterval: 50 * time.Millisecond,
			MaxRetries:      3,
		})

		// Add a pending payout
		repo.AddPendingPayout(PendingPayout{
			UserID:  1,
			Amount:  1000000,
			Address: "ltc1qtest123456789",
		})

		// Start auto-processing
		processor.Start()

		// Wait for processing
		time.Sleep(150 * time.Millisecond)

		// Stop and verify
		processor.Stop()

		assert.Equal(t, 1, repo.GetCompletedCount())
	})
}

func TestPayoutProcessor_GetStats(t *testing.T) {
	t.Run("returns processing statistics", func(t *testing.T) {
		wallet := NewMockWalletClient()
		repo := NewMockPayoutRepository()

		processor := NewPayoutProcessor(wallet, repo, ProcessorConfig{
			BatchSize:       10,
			ProcessInterval: time.Second,
			MaxRetries:      3,
		})

		// Process some payouts
		repo.AddPendingPayout(PendingPayout{
			UserID:  1,
			Amount:  1000000,
			Address: "ltc1qtest123456789",
		})
		repo.AddPendingPayout(PendingPayout{
			UserID:  2,
			Amount:  2000000,
			Address: "ltc1qtest987654321",
		})

		_ = processor.ProcessPendingPayouts(context.Background())

		stats := processor.GetStats()
		assert.Equal(t, int64(2), stats.PayoutsProcessed)
		assert.Equal(t, int64(3000000), stats.TotalAmountSent)
	})
}

func TestPayoutProcessor_MinimumPayout(t *testing.T) {
	t.Run("skips payouts below minimum", func(t *testing.T) {
		wallet := NewMockWalletClient()
		repo := NewMockPayoutRepository()

		processor := NewPayoutProcessor(wallet, repo, ProcessorConfig{
			BatchSize:       10,
			ProcessInterval: time.Second,
			MaxRetries:      3,
			MinPayoutAmount: 500000, // 0.005 LTC minimum
		})

		// Add payout below minimum
		repo.AddPendingPayout(PendingPayout{
			UserID:  1,
			Amount:  100000, // Below minimum
			Address: "ltc1qtest123456789",
		})

		err := processor.ProcessPendingPayouts(context.Background())
		assert.NoError(t, err)

		// Should not be processed
		txs := wallet.GetTransactions()
		assert.Len(t, txs, 0)
	})
}
