package payouts

import (
	"database/sql"
	"errors"
	"fmt"
)

var (
	ErrWalletNotFound       = errors.New("wallet not found")
	ErrWalletPercentageOver = errors.New("total wallet percentages would exceed 100%")
	ErrInvalidPercentage    = errors.New("percentage must be between 0 and 100")
	ErrDuplicateAddress     = errors.New("wallet address already exists for this user")
)

// WalletRepository defines the interface for wallet data access
type WalletRepository interface {
	CreateWallet(wallet *UserWallet) error
	GetWalletByID(id int64) (*UserWallet, error)
	GetWalletsByUserID(userID int64) ([]UserWallet, error)
	GetActiveWalletsByUserID(userID int64) ([]UserWallet, error)
	UpdateWallet(wallet *UserWallet) error
	DeleteWallet(id int64) error
	GetTotalPercentage(userID int64, excludeWalletID int64) (float64, error)
}

// WalletService handles wallet management logic
type WalletService struct {
	repo WalletRepository
}

// NewWalletService creates a new wallet service
func NewWalletService(repo WalletRepository) *WalletService {
	return &WalletService{repo: repo}
}

// CreateWallet creates a new wallet for a user
func (s *WalletService) CreateWallet(userID int64, req CreateWalletRequest) (*UserWallet, error) {
	// Validate percentage
	if req.Percentage <= 0 || req.Percentage > 100 {
		return nil, ErrInvalidPercentage
	}

	// Check total percentage wouldn't exceed 100%
	currentTotal, err := s.repo.GetTotalPercentage(userID, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get current percentage: %w", err)
	}

	if currentTotal+req.Percentage > 100 {
		return nil, fmt.Errorf("%w: current total is %.2f%%, adding %.2f%% would exceed 100%%",
			ErrWalletPercentageOver, currentTotal, req.Percentage)
	}

	wallet := &UserWallet{
		UserID:     userID,
		Address:    req.Address,
		Label:      req.Label,
		Percentage: req.Percentage,
		IsPrimary:  req.IsPrimary,
		IsActive:   true,
	}

	if err := s.repo.CreateWallet(wallet); err != nil {
		return nil, err
	}

	return wallet, nil
}

// GetUserWallets returns all wallets for a user
func (s *WalletService) GetUserWallets(userID int64) ([]UserWallet, error) {
	return s.repo.GetWalletsByUserID(userID)
}

// GetActiveUserWallets returns only active wallets for a user
func (s *WalletService) GetActiveUserWallets(userID int64) ([]UserWallet, error) {
	return s.repo.GetActiveWalletsByUserID(userID)
}

// GetWalletSummary returns a summary of wallet allocations
func (s *WalletService) GetWalletSummary(userID int64) (*WalletSummary, error) {
	wallets, err := s.repo.GetWalletsByUserID(userID)
	if err != nil {
		return nil, err
	}

	summary := &WalletSummary{}
	summary.TotalWallets = len(wallets)

	for _, w := range wallets {
		if w.IsActive {
			summary.ActiveWallets++
			summary.TotalPercentage += w.Percentage
		}
		if w.IsPrimary {
			summary.HasPrimaryWallet = true
		}
	}

	summary.RemainingPercent = 100 - summary.TotalPercentage
	if summary.RemainingPercent < 0 {
		summary.RemainingPercent = 0
	}

	return summary, nil
}

// UpdateWallet updates an existing wallet
func (s *WalletService) UpdateWallet(userID int64, walletID int64, req UpdateWalletRequest) (*UserWallet, error) {
	wallet, err := s.repo.GetWalletByID(walletID)
	if err != nil {
		return nil, err
	}

	// Verify ownership
	if wallet.UserID != userID {
		return nil, ErrWalletNotFound
	}

	// If percentage is changing, validate it
	if req.Percentage > 0 && req.Percentage != wallet.Percentage {
		if req.Percentage > 100 {
			return nil, ErrInvalidPercentage
		}

		currentTotal, err := s.repo.GetTotalPercentage(userID, walletID)
		if err != nil {
			return nil, err
		}

		if currentTotal+req.Percentage > 100 {
			return nil, fmt.Errorf("%w: current total (excluding this wallet) is %.2f%%, setting to %.2f%% would exceed 100%%",
				ErrWalletPercentageOver, currentTotal, req.Percentage)
		}

		wallet.Percentage = req.Percentage
	}

	// Update other fields
	if req.Address != "" {
		wallet.Address = req.Address
	}
	if req.Label != "" {
		wallet.Label = req.Label
	}
	wallet.IsPrimary = req.IsPrimary
	wallet.IsActive = req.IsActive

	if err := s.repo.UpdateWallet(wallet); err != nil {
		return nil, err
	}

	return wallet, nil
}

// DeleteWallet removes a wallet
func (s *WalletService) DeleteWallet(userID int64, walletID int64) error {
	wallet, err := s.repo.GetWalletByID(walletID)
	if err != nil {
		return err
	}

	// Verify ownership
	if wallet.UserID != userID {
		return ErrWalletNotFound
	}

	return s.repo.DeleteWallet(walletID)
}

// CalculateSplitPayouts calculates how to split a payout for a user
func (s *WalletService) CalculateSplitPayouts(userID int64, totalAmount int64) ([]PayoutSplit, error) {
	wallets, err := s.repo.GetActiveWalletsByUserID(userID)
	if err != nil {
		return nil, err
	}

	return CalculatePayoutSplits(wallets, totalAmount), nil
}

// PostgresWalletRepository implements WalletRepository using PostgreSQL
type PostgresWalletRepository struct {
	db *sql.DB
}

// NewPostgresWalletRepository creates a new PostgreSQL wallet repository
func NewPostgresWalletRepository(db *sql.DB) *PostgresWalletRepository {
	return &PostgresWalletRepository{db: db}
}

func (r *PostgresWalletRepository) CreateWallet(wallet *UserWallet) error {
	query := `
		INSERT INTO user_wallets (user_id, address, label, percentage, is_primary, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`

	return r.db.QueryRow(query,
		wallet.UserID, wallet.Address, wallet.Label,
		wallet.Percentage, wallet.IsPrimary, wallet.IsActive,
	).Scan(&wallet.ID, &wallet.CreatedAt, &wallet.UpdatedAt)
}

func (r *PostgresWalletRepository) GetWalletByID(id int64) (*UserWallet, error) {
	query := `
		SELECT id, user_id, address, label, percentage, is_primary, is_active, created_at, updated_at
		FROM user_wallets WHERE id = $1`

	wallet := &UserWallet{}
	err := r.db.QueryRow(query, id).Scan(
		&wallet.ID, &wallet.UserID, &wallet.Address, &wallet.Label,
		&wallet.Percentage, &wallet.IsPrimary, &wallet.IsActive,
		&wallet.CreatedAt, &wallet.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrWalletNotFound
	}
	return wallet, err
}

func (r *PostgresWalletRepository) GetWalletsByUserID(userID int64) ([]UserWallet, error) {
	query := `
		SELECT id, user_id, address, label, percentage, is_primary, is_active, created_at, updated_at
		FROM user_wallets WHERE user_id = $1
		ORDER BY is_primary DESC, created_at ASC`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var wallets []UserWallet
	for rows.Next() {
		var w UserWallet
		if err := rows.Scan(
			&w.ID, &w.UserID, &w.Address, &w.Label,
			&w.Percentage, &w.IsPrimary, &w.IsActive,
			&w.CreatedAt, &w.UpdatedAt,
		); err != nil {
			return nil, err
		}
		wallets = append(wallets, w)
	}

	return wallets, rows.Err()
}

func (r *PostgresWalletRepository) GetActiveWalletsByUserID(userID int64) ([]UserWallet, error) {
	query := `
		SELECT id, user_id, address, label, percentage, is_primary, is_active, created_at, updated_at
		FROM user_wallets WHERE user_id = $1 AND is_active = true
		ORDER BY is_primary DESC, created_at ASC`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var wallets []UserWallet
	for rows.Next() {
		var w UserWallet
		if err := rows.Scan(
			&w.ID, &w.UserID, &w.Address, &w.Label,
			&w.Percentage, &w.IsPrimary, &w.IsActive,
			&w.CreatedAt, &w.UpdatedAt,
		); err != nil {
			return nil, err
		}
		wallets = append(wallets, w)
	}

	return wallets, rows.Err()
}

func (r *PostgresWalletRepository) UpdateWallet(wallet *UserWallet) error {
	query := `
		UPDATE user_wallets 
		SET address = $1, label = $2, percentage = $3, is_primary = $4, is_active = $5
		WHERE id = $6`

	result, err := r.db.Exec(query,
		wallet.Address, wallet.Label, wallet.Percentage,
		wallet.IsPrimary, wallet.IsActive, wallet.ID,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrWalletNotFound
	}

	return nil
}

func (r *PostgresWalletRepository) DeleteWallet(id int64) error {
	result, err := r.db.Exec("DELETE FROM user_wallets WHERE id = $1", id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrWalletNotFound
	}

	return nil
}

func (r *PostgresWalletRepository) GetTotalPercentage(userID int64, excludeWalletID int64) (float64, error) {
	query := `
		SELECT COALESCE(SUM(percentage), 0) 
		FROM user_wallets 
		WHERE user_id = $1 AND is_active = true AND id != $2`

	var total float64
	err := r.db.QueryRow(query, userID, excludeWalletID).Scan(&total)
	return total, err
}
