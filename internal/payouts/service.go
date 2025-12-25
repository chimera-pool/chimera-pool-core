package payouts

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// PayoutDatabase defines the database interface for payout operations
type PayoutDatabase interface {
	GetSharesForPayout(ctx context.Context, blockTime time.Time, windowSize int64) ([]Share, error)
	CreatePayouts(ctx context.Context, payouts []Payout) error
	GetBlock(ctx context.Context, blockID int64) (*Block, error)
	GetPayoutHistory(ctx context.Context, userID int64, limit, offset int) ([]Payout, error)
}

// PayoutService handles the complete payout processing workflow
type PayoutService struct {
	db         PayoutDatabase
	calculator *PPLNSCalculator
}

// NewPayoutService creates a new payout service
func NewPayoutService(db PayoutDatabase, calculator *PPLNSCalculator) *PayoutService {
	return &PayoutService{
		db:         db,
		calculator: calculator,
	}
}

// ProcessBlockPayout processes payouts for a confirmed block
func (s *PayoutService) ProcessBlockPayout(ctx context.Context, blockID int64) error {
	// Get block information
	block, err := s.db.GetBlock(ctx, blockID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("block not found: %d", blockID)
		}
		return fmt.Errorf("failed to get block: %w", err)
	}

	// Only process confirmed blocks
	if block.Status != "confirmed" {
		return fmt.Errorf("block not confirmed: %s (status: %s)", block.Hash, block.Status)
	}

	// Get shares for payout calculation
	shares, err := s.db.GetSharesForPayout(ctx, block.Timestamp, s.calculator.GetWindowSize())
	if err != nil {
		return fmt.Errorf("failed to get shares: %w", err)
	}

	// Calculate payouts using PPLNS (tx fees passed as 0 for now - can be added to Block struct)
	payouts, err := s.calculator.CalculatePayouts(shares, block.Reward, 0, block.Timestamp)
	if err != nil {
		return fmt.Errorf("failed to calculate payouts: %w", err)
	}

	// Set block ID for all payouts
	for i := range payouts {
		payouts[i].BlockID = blockID
	}

	// Store payouts in database
	if len(payouts) > 0 {
		err = s.db.CreatePayouts(ctx, payouts)
		if err != nil {
			return fmt.Errorf("failed to create payouts: %w", err)
		}
	}

	return nil
}

// CalculateEstimatedPayout calculates estimated payout for a user based on current shares
func (s *PayoutService) CalculateEstimatedPayout(ctx context.Context, userID int64, estimatedBlockReward int64) (int64, error) {
	// Get current shares for estimation
	shares, err := s.db.GetSharesForPayout(ctx, time.Now(), s.calculator.GetWindowSize())
	if err != nil {
		return 0, fmt.Errorf("failed to get shares for estimation: %w", err)
	}

	// Calculate hypothetical payouts
	payouts, err := s.calculator.CalculatePayouts(shares, estimatedBlockReward, 0, time.Now())
	if err != nil {
		return 0, fmt.Errorf("failed to calculate estimated payouts: %w", err)
	}

	// Find payout for the specified user
	for _, payout := range payouts {
		if payout.UserID == userID {
			return payout.Amount, nil
		}
	}

	// User has no shares in current window
	return 0, nil
}

// GetPayoutHistory retrieves payout history for a user
func (s *PayoutService) GetPayoutHistory(ctx context.Context, userID int64, limit, offset int) ([]Payout, error) {
	return s.db.GetPayoutHistory(ctx, userID, limit, offset)
}

// GetPayoutStatistics calculates payout statistics for a user
func (s *PayoutService) GetPayoutStatistics(ctx context.Context, userID int64, since time.Time) (*PayoutStatistics, error) {
	// Get all payouts since the specified time
	payouts, err := s.db.GetPayoutHistory(ctx, userID, 1000, 0) // Get up to 1000 recent payouts
	if err != nil {
		return nil, fmt.Errorf("failed to get payout history: %w", err)
	}

	stats := &PayoutStatistics{
		UserID:      userID,
		TotalPayout: 0,
		PayoutCount: 0,
		Since:       since,
	}

	for _, payout := range payouts {
		if payout.Timestamp.After(since) {
			stats.TotalPayout += payout.Amount
			stats.PayoutCount++

			if stats.LastPayout.IsZero() || payout.Timestamp.After(stats.LastPayout) {
				stats.LastPayout = payout.Timestamp
			}
		}
	}

	if stats.PayoutCount > 0 {
		stats.AveragePayout = stats.TotalPayout / int64(stats.PayoutCount)
	}

	return stats, nil
}

// ValidatePayoutFairness validates that payouts are mathematically fair
func (s *PayoutService) ValidatePayoutFairness(ctx context.Context, blockID int64) (*PayoutValidation, error) {
	// Get block information
	block, err := s.db.GetBlock(ctx, blockID)
	if err != nil {
		return nil, fmt.Errorf("failed to get block: %w", err)
	}

	// Get shares used for this block's payout
	shares, err := s.db.GetSharesForPayout(ctx, block.Timestamp, s.calculator.GetWindowSize())
	if err != nil {
		return nil, fmt.Errorf("failed to get shares: %w", err)
	}

	// Recalculate payouts
	expectedPayouts, err := s.calculator.CalculatePayouts(shares, block.Reward, 0, block.Timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to recalculate payouts: %w", err)
	}

	// Get actual payouts from database
	actualPayouts, err := s.db.GetPayoutHistory(ctx, 0, 1000, 0) // Get all recent payouts
	if err != nil {
		return nil, fmt.Errorf("failed to get actual payouts: %w", err)
	}

	// Filter payouts for this block
	blockPayouts := make([]Payout, 0)
	for _, payout := range actualPayouts {
		if payout.BlockID == blockID {
			blockPayouts = append(blockPayouts, payout)
		}
	}

	// Validate fairness
	validation := &PayoutValidation{
		BlockID:         blockID,
		IsValid:         true,
		ExpectedPayouts: expectedPayouts,
		ActualPayouts:   blockPayouts,
		Discrepancies:   make([]PayoutDiscrepancy, 0),
	}

	// Check if payout counts match
	if len(expectedPayouts) != len(blockPayouts) {
		validation.IsValid = false
		validation.Discrepancies = append(validation.Discrepancies, PayoutDiscrepancy{
			Type:        "count_mismatch",
			Description: fmt.Sprintf("Expected %d payouts, got %d", len(expectedPayouts), len(blockPayouts)),
		})
	}

	// Check individual payout amounts
	expectedMap := make(map[int64]int64)
	for _, payout := range expectedPayouts {
		expectedMap[payout.UserID] = payout.Amount
	}

	actualMap := make(map[int64]int64)
	for _, payout := range blockPayouts {
		actualMap[payout.UserID] = payout.Amount
	}

	for userID, expectedAmount := range expectedMap {
		actualAmount, exists := actualMap[userID]
		if !exists {
			validation.IsValid = false
			validation.Discrepancies = append(validation.Discrepancies, PayoutDiscrepancy{
				Type:        "missing_payout",
				UserID:      userID,
				Description: fmt.Sprintf("Expected payout of %d for user %d, but no payout found", expectedAmount, userID),
			})
		} else if actualAmount != expectedAmount {
			validation.IsValid = false
			validation.Discrepancies = append(validation.Discrepancies, PayoutDiscrepancy{
				Type:        "amount_mismatch",
				UserID:      userID,
				Description: fmt.Sprintf("Expected %d, got %d for user %d", expectedAmount, actualAmount, userID),
			})
		}
	}

	// Check for unexpected payouts
	for userID, actualAmount := range actualMap {
		if _, exists := expectedMap[userID]; !exists {
			validation.IsValid = false
			validation.Discrepancies = append(validation.Discrepancies, PayoutDiscrepancy{
				Type:        "unexpected_payout",
				UserID:      userID,
				Description: fmt.Sprintf("Unexpected payout of %d for user %d", actualAmount, userID),
			})
		}
	}

	return validation, nil
}

// PayoutStatistics represents payout statistics for a user
type PayoutStatistics struct {
	UserID        int64     `json:"user_id"`
	TotalPayout   int64     `json:"total_payout"`
	PayoutCount   int       `json:"payout_count"`
	AveragePayout int64     `json:"average_payout"`
	LastPayout    time.Time `json:"last_payout"`
	Since         time.Time `json:"since"`
}

// PayoutValidation represents the result of payout fairness validation
type PayoutValidation struct {
	BlockID         int64               `json:"block_id"`
	IsValid         bool                `json:"is_valid"`
	ExpectedPayouts []Payout            `json:"expected_payouts"`
	ActualPayouts   []Payout            `json:"actual_payouts"`
	Discrepancies   []PayoutDiscrepancy `json:"discrepancies"`
}

// PayoutDiscrepancy represents a discrepancy found during validation
type PayoutDiscrepancy struct {
	Type        string `json:"type"`
	UserID      int64  `json:"user_id,omitempty"`
	Description string `json:"description"`
}
