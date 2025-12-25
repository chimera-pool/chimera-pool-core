package payouts

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// MOCK IMPLEMENTATIONS FOR TESTING
// =============================================================================

// MockBlockNotifier simulates block found events
type MockBlockNotifier struct {
	subscribers []func(block *Block)
	mu          sync.RWMutex
}

func NewMockBlockNotifier() *MockBlockNotifier {
	return &MockBlockNotifier{
		subscribers: make([]func(block *Block), 0),
	}
}

func (m *MockBlockNotifier) Subscribe(handler func(block *Block)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.subscribers = append(m.subscribers, handler)
}

func (m *MockBlockNotifier) NotifyBlockFound(block *Block) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, handler := range m.subscribers {
		handler(block)
	}
}

// MockPayoutQueue stores pending payouts
type MockPayoutQueue struct {
	payouts []PendingPayout
	mu      sync.RWMutex
}

func NewMockPayoutQueue() *MockPayoutQueue {
	return &MockPayoutQueue{
		payouts: make([]PendingPayout, 0),
	}
}

func (m *MockPayoutQueue) Enqueue(payout PendingPayout) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.payouts = append(m.payouts, payout)
	return nil
}

func (m *MockPayoutQueue) GetPending() []PendingPayout {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]PendingPayout, len(m.payouts))
	copy(result, m.payouts)
	return result
}

func (m *MockPayoutQueue) MarkProcessed(payoutID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.payouts {
		if m.payouts[i].ID == payoutID {
			m.payouts[i].Status = PayoutStatusProcessed
		}
	}
	return nil
}

// MockUserSettingsProvider returns user payout settings
type MockUserSettingsProvider struct {
	settings map[int64]*UserPayoutSettings
}

func NewMockUserSettingsProvider() *MockUserSettingsProvider {
	return &MockUserSettingsProvider{
		settings: make(map[int64]*UserPayoutSettings),
	}
}

func (m *MockUserSettingsProvider) GetUserPayoutSettings(userID int64) (*UserPayoutSettings, error) {
	if settings, ok := m.settings[userID]; ok {
		return settings, nil
	}
	// Return default settings
	return &UserPayoutSettings{
		UserID:           userID,
		PayoutMode:       PayoutModePPLNS,
		MinPayoutAmount:  1000000,
		AutoPayoutEnable: true,
	}, nil
}

func (m *MockUserSettingsProvider) SetUserSettings(userID int64, settings *UserPayoutSettings) {
	m.settings[userID] = settings
}

// MockShareProvider returns shares for payout calculation
type MockShareProvider struct {
	shares []Share
}

func NewMockShareProvider() *MockShareProvider {
	return &MockShareProvider{
		shares: make([]Share, 0),
	}
}

func (m *MockShareProvider) GetSharesInWindow(ctx context.Context, endTime time.Time, windowSize int64) ([]Share, error) {
	return m.shares, nil
}

func (m *MockShareProvider) AddShare(share Share) {
	// Ensure shares are valid by default
	share.IsValid = true
	m.shares = append(m.shares, share)
}

// MockBalanceTracker tracks user balances
type MockBalanceTracker struct {
	balances map[int64]int64
	mu       sync.RWMutex
}

func NewMockBalanceTracker() *MockBalanceTracker {
	return &MockBalanceTracker{
		balances: make(map[int64]int64),
	}
}

func (m *MockBalanceTracker) GetBalance(userID int64) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.balances[userID]
}

func (m *MockBalanceTracker) AddToBalance(userID int64, amount int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.balances[userID] += amount
	return nil
}

func (m *MockBalanceTracker) DeductFromBalance(userID int64, amount int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.balances[userID] < amount {
		return ErrInsufficientBalance
	}
	m.balances[userID] -= amount
	return nil
}

// =============================================================================
// PAYOUT EXECUTOR TESTS (TDD - Write tests first)
// =============================================================================

func TestPayoutExecutor_Creation(t *testing.T) {
	t.Run("creates executor with dependencies", func(t *testing.T) {
		config := DefaultPayoutConfig()
		notifier := NewMockBlockNotifier()
		queue := NewMockPayoutQueue()
		settings := NewMockUserSettingsProvider()
		shares := NewMockShareProvider()
		balances := NewMockBalanceTracker()

		executor := NewPayoutExecutor(config, notifier, queue, settings, shares, balances)

		assert.NotNil(t, executor)
		assert.NotEmpty(t, executor.calculators)
	})

	t.Run("creates executor with nil dependencies returns error", func(t *testing.T) {
		executor := NewPayoutExecutor(nil, nil, nil, nil, nil, nil)
		assert.Nil(t, executor)
	})
}

func TestPayoutExecutor_ProcessBlock(t *testing.T) {
	t.Run("processes block and calculates payouts", func(t *testing.T) {
		config := DefaultPayoutConfig()
		notifier := NewMockBlockNotifier()
		queue := NewMockPayoutQueue()
		settings := NewMockUserSettingsProvider()
		shares := NewMockShareProvider()
		balances := NewMockBalanceTracker()

		// Add shares for two users
		shares.AddShare(Share{UserID: 1, Difficulty: 100, Timestamp: time.Now().Add(-5 * time.Minute)})
		shares.AddShare(Share{UserID: 1, Difficulty: 100, Timestamp: time.Now().Add(-4 * time.Minute)})
		shares.AddShare(Share{UserID: 2, Difficulty: 100, Timestamp: time.Now().Add(-3 * time.Minute)})
		shares.AddShare(Share{UserID: 2, Difficulty: 100, Timestamp: time.Now().Add(-2 * time.Minute)})

		executor := NewPayoutExecutor(config, notifier, queue, settings, shares, balances)
		require.NotNil(t, executor)

		// Process a confirmed block
		block := &Block{
			ID:        1,
			Hash:      "test_block_hash",
			Reward:    1000000000, // 10 LTC in satoshis
			Status:    "confirmed",
			Timestamp: time.Now(),
		}

		err := executor.ProcessBlock(context.Background(), block)
		require.NoError(t, err)

		// Check that balances were credited
		assert.Greater(t, balances.GetBalance(1), int64(0))
		assert.Greater(t, balances.GetBalance(2), int64(0))

		// Total credited should be block reward minus pool fee
		totalCredited := balances.GetBalance(1) + balances.GetBalance(2)
		expectedTotal := int64(float64(block.Reward) * (1 - config.FeePPLNS/100))
		assert.InDelta(t, expectedTotal, totalCredited, 1000) // Allow small rounding
	})

	t.Run("skips unconfirmed blocks", func(t *testing.T) {
		config := DefaultPayoutConfig()
		notifier := NewMockBlockNotifier()
		queue := NewMockPayoutQueue()
		settings := NewMockUserSettingsProvider()
		shares := NewMockShareProvider()
		balances := NewMockBalanceTracker()

		executor := NewPayoutExecutor(config, notifier, queue, settings, shares, balances)

		block := &Block{
			ID:     1,
			Status: "pending",
			Reward: 1000000000,
		}

		err := executor.ProcessBlock(context.Background(), block)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not confirmed")
	})

	t.Run("handles empty shares gracefully", func(t *testing.T) {
		config := DefaultPayoutConfig()
		notifier := NewMockBlockNotifier()
		queue := NewMockPayoutQueue()
		settings := NewMockUserSettingsProvider()
		shares := NewMockShareProvider() // Empty shares
		balances := NewMockBalanceTracker()

		executor := NewPayoutExecutor(config, notifier, queue, settings, shares, balances)

		block := &Block{
			ID:        1,
			Status:    "confirmed",
			Reward:    1000000000,
			Timestamp: time.Now(),
		}

		err := executor.ProcessBlock(context.Background(), block)
		assert.NoError(t, err) // Should not error, just no payouts
	})
}

func TestPayoutExecutor_MultiModePayouts(t *testing.T) {
	t.Run("uses different calculator based on user settings", func(t *testing.T) {
		config := DefaultPayoutConfig()
		notifier := NewMockBlockNotifier()
		queue := NewMockPayoutQueue()
		settings := NewMockUserSettingsProvider()
		shares := NewMockShareProvider()
		balances := NewMockBalanceTracker()

		// User 1 uses PPLNS, User 2 uses SLICE
		settings.SetUserSettings(1, &UserPayoutSettings{
			UserID:           1,
			PayoutMode:       PayoutModePPLNS,
			MinPayoutAmount:  1000000,
			AutoPayoutEnable: true,
		})
		settings.SetUserSettings(2, &UserPayoutSettings{
			UserID:           2,
			PayoutMode:       PayoutModeSLICE,
			MinPayoutAmount:  1000000,
			AutoPayoutEnable: true,
		})

		// Add equal shares for both users
		shares.AddShare(Share{UserID: 1, Difficulty: 100, Timestamp: time.Now().Add(-5 * time.Minute)})
		shares.AddShare(Share{UserID: 2, Difficulty: 100, Timestamp: time.Now().Add(-4 * time.Minute)})

		executor := NewPayoutExecutor(config, notifier, queue, settings, shares, balances)

		block := &Block{
			ID:        1,
			Status:    "confirmed",
			Reward:    1000000000,
			Timestamp: time.Now(),
		}

		err := executor.ProcessBlock(context.Background(), block)
		require.NoError(t, err)

		// Both users should have received payouts
		assert.Greater(t, balances.GetBalance(1), int64(0))
		assert.Greater(t, balances.GetBalance(2), int64(0))
	})
}

func TestPayoutExecutor_AutoPayoutTriggering(t *testing.T) {
	t.Run("queues payout when balance exceeds threshold", func(t *testing.T) {
		config := DefaultPayoutConfig()
		notifier := NewMockBlockNotifier()
		queue := NewMockPayoutQueue()
		settings := NewMockUserSettingsProvider()
		shares := NewMockShareProvider()
		balances := NewMockBalanceTracker()

		// Set low threshold for testing
		settings.SetUserSettings(1, &UserPayoutSettings{
			UserID:           1,
			PayoutMode:       PayoutModePPLNS,
			MinPayoutAmount:  100000, // 0.001 LTC
			AutoPayoutEnable: true,
			PayoutAddress:    "ltc1qtest123",
		})

		// Add large shares so payout exceeds threshold
		for i := 0; i < 100; i++ {
			shares.AddShare(Share{UserID: 1, Difficulty: 1000, Timestamp: time.Now().Add(-time.Duration(i) * time.Minute)})
		}

		executor := NewPayoutExecutor(config, notifier, queue, settings, shares, balances)

		block := &Block{
			ID:        1,
			Status:    "confirmed",
			Reward:    1000000000, // 10 LTC
			Timestamp: time.Now(),
		}

		err := executor.ProcessBlock(context.Background(), block)
		require.NoError(t, err)

		// Check that payout was queued
		pending := queue.GetPending()
		assert.GreaterOrEqual(t, len(pending), 1)
	})

	t.Run("does not queue payout when auto-payout disabled", func(t *testing.T) {
		config := DefaultPayoutConfig()
		notifier := NewMockBlockNotifier()
		queue := NewMockPayoutQueue()
		settings := NewMockUserSettingsProvider()
		shares := NewMockShareProvider()
		balances := NewMockBalanceTracker()

		// Disable auto-payout
		settings.SetUserSettings(1, &UserPayoutSettings{
			UserID:           1,
			PayoutMode:       PayoutModePPLNS,
			MinPayoutAmount:  100000,
			AutoPayoutEnable: false,
			PayoutAddress:    "ltc1qtest123",
		})

		shares.AddShare(Share{UserID: 1, Difficulty: 10000, Timestamp: time.Now()})

		executor := NewPayoutExecutor(config, notifier, queue, settings, shares, balances)

		block := &Block{
			ID:        1,
			Status:    "confirmed",
			Reward:    1000000000,
			Timestamp: time.Now(),
		}

		err := executor.ProcessBlock(context.Background(), block)
		require.NoError(t, err)

		// Balance should be credited but no payout queued
		assert.Greater(t, balances.GetBalance(1), int64(0))
		pending := queue.GetPending()
		assert.Len(t, pending, 0)
	})
}

func TestPayoutExecutor_BlockEventSubscription(t *testing.T) {
	t.Run("automatically processes blocks from notifier", func(t *testing.T) {
		config := DefaultPayoutConfig()
		notifier := NewMockBlockNotifier()
		queue := NewMockPayoutQueue()
		settings := NewMockUserSettingsProvider()
		shares := NewMockShareProvider()
		balances := NewMockBalanceTracker()

		shares.AddShare(Share{UserID: 1, Difficulty: 100, Timestamp: time.Now()})

		executor := NewPayoutExecutor(config, notifier, queue, settings, shares, balances)
		executor.Start()
		defer executor.Stop()

		// Simulate block found event
		block := &Block{
			ID:        1,
			Status:    "confirmed",
			Reward:    1000000000,
			Timestamp: time.Now(),
		}
		notifier.NotifyBlockFound(block)

		// Wait for processing
		time.Sleep(100 * time.Millisecond)

		// Check balance was credited
		assert.Greater(t, balances.GetBalance(1), int64(0))
	})
}

func TestPayoutExecutor_ErrorHandling(t *testing.T) {
	t.Run("handles calculator errors gracefully", func(t *testing.T) {
		config := DefaultPayoutConfig()
		notifier := NewMockBlockNotifier()
		queue := NewMockPayoutQueue()
		settings := NewMockUserSettingsProvider()
		shares := NewMockShareProvider()
		balances := NewMockBalanceTracker()

		executor := NewPayoutExecutor(config, notifier, queue, settings, shares, balances)

		// Block with zero reward
		block := &Block{
			ID:        1,
			Status:    "confirmed",
			Reward:    0,
			Timestamp: time.Now(),
		}

		err := executor.ProcessBlock(context.Background(), block)
		assert.NoError(t, err) // Should handle gracefully
	})
}

func TestPayoutExecutor_GetStats(t *testing.T) {
	t.Run("returns processing statistics", func(t *testing.T) {
		config := DefaultPayoutConfig()
		notifier := NewMockBlockNotifier()
		queue := NewMockPayoutQueue()
		settings := NewMockUserSettingsProvider()
		shares := NewMockShareProvider()
		balances := NewMockBalanceTracker()

		shares.AddShare(Share{UserID: 1, Difficulty: 100, Timestamp: time.Now()})

		executor := NewPayoutExecutor(config, notifier, queue, settings, shares, balances)

		block := &Block{
			ID:        1,
			Status:    "confirmed",
			Reward:    1000000000,
			Timestamp: time.Now(),
		}

		_ = executor.ProcessBlock(context.Background(), block)

		stats := executor.GetStats()
		assert.Equal(t, int64(1), stats.BlocksProcessed)
		assert.Greater(t, stats.TotalPayoutsCalculated, int64(0))
	})
}
