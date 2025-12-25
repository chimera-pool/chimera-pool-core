package payouts

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// =============================================================================
// SQL PAYOUT REPOSITORY IMPLEMENTATION
// =============================================================================

// SQLPayoutRepository implements PayoutRepository using PostgreSQL
type SQLPayoutRepository struct {
	db *sql.DB
}

// NewSQLPayoutRepository creates a new SQL-backed payout repository
func NewSQLPayoutRepository(db *sql.DB) *SQLPayoutRepository {
	return &SQLPayoutRepository{db: db}
}

// GetPendingPayouts retrieves pending payouts up to the specified limit
func (r *SQLPayoutRepository) GetPendingPayouts(ctx context.Context, limit int) ([]PendingPayout, error) {
	query := `
		SELECT id, user_id, amount, address, status, payout_mode, block_id, 
		       created_at, processed_at, tx_hash, error_message
		FROM pending_payouts
		WHERE status = $1
		ORDER BY created_at ASC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, PayoutStatusPending, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending payouts: %w", err)
	}
	defer rows.Close()

	payouts := make([]PendingPayout, 0)
	for rows.Next() {
		var p PendingPayout
		var processedAt sql.NullTime
		var txHash, errorMsg sql.NullString
		var payoutMode string

		err := rows.Scan(
			&p.ID, &p.UserID, &p.Amount, &p.Address, &p.Status,
			&payoutMode, &p.BlockID, &p.CreatedAt, &processedAt,
			&txHash, &errorMsg,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan payout row: %w", err)
		}

		p.PayoutMode = PayoutMode(payoutMode)
		if processedAt.Valid {
			p.ProcessedAt = &processedAt.Time
		}
		if txHash.Valid {
			p.TxHash = txHash.String
		}
		if errorMsg.Valid {
			p.ErrorMessage = errorMsg.String
		}

		payouts = append(payouts, p)
	}

	return payouts, rows.Err()
}

// MarkPayoutComplete marks a payout as processed with the transaction hash
func (r *SQLPayoutRepository) MarkPayoutComplete(ctx context.Context, payoutID int64, txHash string) error {
	query := `
		UPDATE pending_payouts
		SET status = $1, tx_hash = $2, processed_at = $3
		WHERE id = $4
	`

	result, err := r.db.ExecContext(ctx, query, PayoutStatusProcessed, txHash, time.Now(), payoutID)
	if err != nil {
		return fmt.Errorf("failed to update payout: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// MarkPayoutFailed marks a payout as failed with an error message
func (r *SQLPayoutRepository) MarkPayoutFailed(ctx context.Context, payoutID int64, errorMsg string) error {
	query := `
		UPDATE pending_payouts
		SET status = $1, error_message = $2, processed_at = $3
		WHERE id = $4
	`

	result, err := r.db.ExecContext(ctx, query, PayoutStatusFailed, errorMsg, time.Now(), payoutID)
	if err != nil {
		return fmt.Errorf("failed to update payout: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// ReturnToBalance returns amount to user's balance after a failed payout
func (r *SQLPayoutRepository) ReturnToBalance(ctx context.Context, userID int64, amount int64) error {
	query := `
		UPDATE user_balances
		SET balance = balance + $1, updated_at = $2
		WHERE user_id = $3
	`

	_, err := r.db.ExecContext(ctx, query, amount, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to return balance: %w", err)
	}

	return nil
}

// CreatePendingPayout creates a new pending payout record
func (r *SQLPayoutRepository) CreatePendingPayout(ctx context.Context, payout PendingPayout) (int64, error) {
	query := `
		INSERT INTO pending_payouts (user_id, amount, address, status, payout_mode, block_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	var id int64
	err := r.db.QueryRowContext(
		ctx, query,
		payout.UserID, payout.Amount, payout.Address,
		PayoutStatusPending, payout.PayoutMode, payout.BlockID, time.Now(),
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("failed to create payout: %w", err)
	}

	return id, nil
}

// GetUserBalance retrieves a user's current balance
func (r *SQLPayoutRepository) GetUserBalance(ctx context.Context, userID int64) (int64, error) {
	query := `SELECT COALESCE(balance, 0) FROM user_balances WHERE user_id = $1`

	var balance int64
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&balance)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get balance: %w", err)
	}

	return balance, nil
}

// DeductFromBalance deducts amount from user's balance
func (r *SQLPayoutRepository) DeductFromBalance(ctx context.Context, userID int64, amount int64) error {
	// First check balance
	balance, err := r.GetUserBalance(ctx, userID)
	if err != nil {
		return err
	}

	if balance < amount {
		return ErrInsufficientBalance
	}

	query := `
		UPDATE user_balances
		SET balance = balance - $1, updated_at = $2
		WHERE user_id = $3
	`

	_, err = r.db.ExecContext(ctx, query, amount, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to deduct balance: %w", err)
	}

	return nil
}

// AddToBalance adds amount to user's balance
func (r *SQLPayoutRepository) AddToBalance(ctx context.Context, userID int64, amount int64) error {
	query := `
		INSERT INTO user_balances (user_id, balance, created_at, updated_at)
		VALUES ($1, $2, $3, $3)
		ON CONFLICT (user_id) DO UPDATE
		SET balance = user_balances.balance + $2, updated_at = $3
	`

	_, err := r.db.ExecContext(ctx, query, userID, amount, time.Now())
	if err != nil {
		return fmt.Errorf("failed to add to balance: %w", err)
	}

	return nil
}

// GetPayoutHistory retrieves payout history for a user
func (r *SQLPayoutRepository) GetPayoutHistory(ctx context.Context, userID int64, limit, offset int) ([]PendingPayout, error) {
	query := `
		SELECT id, user_id, amount, address, status, payout_mode, block_id,
		       created_at, processed_at, tx_hash, error_message
		FROM pending_payouts
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query payout history: %w", err)
	}
	defer rows.Close()

	payouts := make([]PendingPayout, 0)
	for rows.Next() {
		var p PendingPayout
		var processedAt sql.NullTime
		var txHash, errorMsg sql.NullString
		var payoutMode string

		err := rows.Scan(
			&p.ID, &p.UserID, &p.Amount, &p.Address, &p.Status,
			&payoutMode, &p.BlockID, &p.CreatedAt, &processedAt,
			&txHash, &errorMsg,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan payout row: %w", err)
		}

		p.PayoutMode = PayoutMode(payoutMode)
		if processedAt.Valid {
			p.ProcessedAt = &processedAt.Time
		}
		if txHash.Valid {
			p.TxHash = txHash.String
		}
		if errorMsg.Valid {
			p.ErrorMessage = errorMsg.String
		}

		payouts = append(payouts, p)
	}

	return payouts, rows.Err()
}

// GetPayoutStats retrieves payout statistics
func (r *SQLPayoutRepository) GetPayoutStats(ctx context.Context) (*PayoutRepoStats, error) {
	query := `
		SELECT 
			COUNT(*) FILTER (WHERE status = 'pending') as pending_count,
			COUNT(*) FILTER (WHERE status = 'processed') as processed_count,
			COUNT(*) FILTER (WHERE status = 'failed') as failed_count,
			COALESCE(SUM(amount) FILTER (WHERE status = 'processed'), 0) as total_paid
		FROM pending_payouts
	`

	var stats PayoutRepoStats
	err := r.db.QueryRowContext(ctx, query).Scan(
		&stats.PendingCount, &stats.ProcessedCount,
		&stats.FailedCount, &stats.TotalPaid,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get payout stats: %w", err)
	}

	return &stats, nil
}

// PayoutRepoStats holds repository statistics
type PayoutRepoStats struct {
	PendingCount   int64 `json:"pending_count"`
	ProcessedCount int64 `json:"processed_count"`
	FailedCount    int64 `json:"failed_count"`
	TotalPaid      int64 `json:"total_paid"`
}
