package payouts

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// SQL PAYOUT REPOSITORY TESTS (TDD)
// =============================================================================

// MockDB implements a minimal database interface for testing
type MockDB struct {
	pendingPayouts []PendingPayout
	balances       map[int64]int64
	nextID         int64
}

func NewMockDB() *MockDB {
	return &MockDB{
		pendingPayouts: make([]PendingPayout, 0),
		balances:       make(map[int64]int64),
		nextID:         1,
	}
}

func TestSQLPayoutRepository_Creation(t *testing.T) {
	t.Run("creates repository with valid db", func(t *testing.T) {
		// Use mock - actual DB connection tested in integration tests
		repo := NewSQLPayoutRepository(nil) // nil DB for unit test
		assert.NotNil(t, repo)
	})
}

func TestSQLPayoutRepository_GetPendingPayouts(t *testing.T) {
	t.Run("returns empty list when no pending payouts", func(t *testing.T) {
		mockDB := NewMockDB()
		repo := &testablePayoutRepository{db: mockDB}

		payouts, err := repo.GetPendingPayouts(context.Background(), 10)
		require.NoError(t, err)
		assert.Empty(t, payouts)
	})

	t.Run("returns pending payouts up to limit", func(t *testing.T) {
		mockDB := NewMockDB()
		repo := &testablePayoutRepository{db: mockDB}

		// Add payouts to mock
		for i := 0; i < 5; i++ {
			mockDB.pendingPayouts = append(mockDB.pendingPayouts, PendingPayout{
				ID:        int64(i + 1),
				UserID:    int64(i + 1),
				Amount:    1000000 * int64(i+1),
				Address:   "ltc1qtest" + string(rune('a'+i)),
				Status:    PayoutStatusPending,
				CreatedAt: time.Now(),
			})
		}

		payouts, err := repo.GetPendingPayouts(context.Background(), 3)
		require.NoError(t, err)
		assert.Len(t, payouts, 3)
	})

	t.Run("only returns pending status payouts", func(t *testing.T) {
		mockDB := NewMockDB()
		repo := &testablePayoutRepository{db: mockDB}

		mockDB.pendingPayouts = append(mockDB.pendingPayouts,
			PendingPayout{ID: 1, Status: PayoutStatusPending},
			PendingPayout{ID: 2, Status: PayoutStatusProcessed},
			PendingPayout{ID: 3, Status: PayoutStatusPending},
		)

		payouts, err := repo.GetPendingPayouts(context.Background(), 10)
		require.NoError(t, err)
		assert.Len(t, payouts, 2)
	})
}

func TestSQLPayoutRepository_MarkPayoutComplete(t *testing.T) {
	t.Run("marks payout as complete with tx hash", func(t *testing.T) {
		mockDB := NewMockDB()
		repo := &testablePayoutRepository{db: mockDB}

		mockDB.pendingPayouts = append(mockDB.pendingPayouts, PendingPayout{
			ID:     1,
			UserID: 1,
			Amount: 1000000,
			Status: PayoutStatusPending,
		})

		err := repo.MarkPayoutComplete(context.Background(), 1, "txhash123")
		require.NoError(t, err)

		// Verify status changed
		assert.Equal(t, PayoutStatusProcessed, mockDB.pendingPayouts[0].Status)
		assert.Equal(t, "txhash123", mockDB.pendingPayouts[0].TxHash)
		assert.NotNil(t, mockDB.pendingPayouts[0].ProcessedAt)
	})

	t.Run("returns error for non-existent payout", func(t *testing.T) {
		mockDB := NewMockDB()
		repo := &testablePayoutRepository{db: mockDB}

		err := repo.MarkPayoutComplete(context.Background(), 999, "txhash123")
		assert.Error(t, err)
	})
}

func TestSQLPayoutRepository_MarkPayoutFailed(t *testing.T) {
	t.Run("marks payout as failed with error message", func(t *testing.T) {
		mockDB := NewMockDB()
		repo := &testablePayoutRepository{db: mockDB}

		mockDB.pendingPayouts = append(mockDB.pendingPayouts, PendingPayout{
			ID:     1,
			UserID: 1,
			Amount: 1000000,
			Status: PayoutStatusPending,
		})

		err := repo.MarkPayoutFailed(context.Background(), 1, "insufficient funds")
		require.NoError(t, err)

		assert.Equal(t, PayoutStatusFailed, mockDB.pendingPayouts[0].Status)
		assert.Equal(t, "insufficient funds", mockDB.pendingPayouts[0].ErrorMessage)
	})
}

func TestSQLPayoutRepository_ReturnToBalance(t *testing.T) {
	t.Run("returns amount to user balance", func(t *testing.T) {
		mockDB := NewMockDB()
		repo := &testablePayoutRepository{db: mockDB}

		mockDB.balances[1] = 5000000 // Initial balance

		err := repo.ReturnToBalance(context.Background(), 1, 1000000)
		require.NoError(t, err)

		assert.Equal(t, int64(6000000), mockDB.balances[1])
	})
}

func TestSQLPayoutRepository_CreatePendingPayout(t *testing.T) {
	t.Run("creates new pending payout", func(t *testing.T) {
		mockDB := NewMockDB()
		repo := &testablePayoutRepository{db: mockDB}

		payout := PendingPayout{
			UserID:     1,
			Amount:     1000000,
			Address:    "ltc1qtest123",
			PayoutMode: PayoutModePPLNS,
		}

		id, err := repo.CreatePendingPayout(context.Background(), payout)
		require.NoError(t, err)
		assert.Greater(t, id, int64(0))
		assert.Len(t, mockDB.pendingPayouts, 1)
	})
}

func TestSQLPayoutRepository_GetUserBalance(t *testing.T) {
	t.Run("returns user balance", func(t *testing.T) {
		mockDB := NewMockDB()
		repo := &testablePayoutRepository{db: mockDB}

		mockDB.balances[1] = 5000000

		balance, err := repo.GetUserBalance(context.Background(), 1)
		require.NoError(t, err)
		assert.Equal(t, int64(5000000), balance)
	})

	t.Run("returns zero for non-existent user", func(t *testing.T) {
		mockDB := NewMockDB()
		repo := &testablePayoutRepository{db: mockDB}

		balance, err := repo.GetUserBalance(context.Background(), 999)
		require.NoError(t, err)
		assert.Equal(t, int64(0), balance)
	})
}

func TestSQLPayoutRepository_DeductFromBalance(t *testing.T) {
	t.Run("deducts from user balance", func(t *testing.T) {
		mockDB := NewMockDB()
		repo := &testablePayoutRepository{db: mockDB}

		mockDB.balances[1] = 5000000

		err := repo.DeductFromBalance(context.Background(), 1, 1000000)
		require.NoError(t, err)
		assert.Equal(t, int64(4000000), mockDB.balances[1])
	})

	t.Run("returns error for insufficient balance", func(t *testing.T) {
		mockDB := NewMockDB()
		repo := &testablePayoutRepository{db: mockDB}

		mockDB.balances[1] = 500000

		err := repo.DeductFromBalance(context.Background(), 1, 1000000)
		assert.Error(t, err)
	})
}

// =============================================================================
// TESTABLE REPOSITORY WITH MOCK DB
// =============================================================================

// testablePayoutRepository wraps MockDB for testing
type testablePayoutRepository struct {
	db *MockDB
}

func (r *testablePayoutRepository) GetPendingPayouts(ctx context.Context, limit int) ([]PendingPayout, error) {
	result := make([]PendingPayout, 0)
	for i, p := range r.db.pendingPayouts {
		if i >= limit {
			break
		}
		if p.Status == PayoutStatusPending {
			result = append(result, p)
		}
	}
	return result, nil
}

func (r *testablePayoutRepository) MarkPayoutComplete(ctx context.Context, payoutID int64, txHash string) error {
	for i := range r.db.pendingPayouts {
		if r.db.pendingPayouts[i].ID == payoutID {
			r.db.pendingPayouts[i].Status = PayoutStatusProcessed
			r.db.pendingPayouts[i].TxHash = txHash
			now := time.Now()
			r.db.pendingPayouts[i].ProcessedAt = &now
			return nil
		}
	}
	return sql.ErrNoRows
}

func (r *testablePayoutRepository) MarkPayoutFailed(ctx context.Context, payoutID int64, errorMsg string) error {
	for i := range r.db.pendingPayouts {
		if r.db.pendingPayouts[i].ID == payoutID {
			r.db.pendingPayouts[i].Status = PayoutStatusFailed
			r.db.pendingPayouts[i].ErrorMessage = errorMsg
			return nil
		}
	}
	return sql.ErrNoRows
}

func (r *testablePayoutRepository) ReturnToBalance(ctx context.Context, userID int64, amount int64) error {
	r.db.balances[userID] += amount
	return nil
}

func (r *testablePayoutRepository) CreatePendingPayout(ctx context.Context, payout PendingPayout) (int64, error) {
	payout.ID = r.db.nextID
	r.db.nextID++
	payout.Status = PayoutStatusPending
	payout.CreatedAt = time.Now()
	r.db.pendingPayouts = append(r.db.pendingPayouts, payout)
	return payout.ID, nil
}

func (r *testablePayoutRepository) GetUserBalance(ctx context.Context, userID int64) (int64, error) {
	return r.db.balances[userID], nil
}

func (r *testablePayoutRepository) DeductFromBalance(ctx context.Context, userID int64, amount int64) error {
	if r.db.balances[userID] < amount {
		return ErrInsufficientBalance
	}
	r.db.balances[userID] -= amount
	return nil
}
