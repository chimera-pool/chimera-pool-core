package payouts

import (
	"time"
)

// UserWallet represents a wallet address for a user
type UserWallet struct {
	ID         int64     `json:"id" db:"id"`
	UserID     int64     `json:"user_id" db:"user_id"`
	Address    string    `json:"address" db:"address"`
	Label      string    `json:"label" db:"label"`
	Percentage float64   `json:"percentage" db:"percentage"`
	IsPrimary  bool      `json:"is_primary" db:"is_primary"`
	IsActive   bool      `json:"is_active" db:"is_active"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

// CreateWalletRequest is the request body for creating a wallet
type CreateWalletRequest struct {
	Address    string  `json:"address" binding:"required"`
	Label      string  `json:"label"`
	Percentage float64 `json:"percentage" binding:"required,gt=0,lte=100"`
	IsPrimary  bool    `json:"is_primary"`
}

// UpdateWalletRequest is the request body for updating a wallet
type UpdateWalletRequest struct {
	Address    string  `json:"address"`
	Label      string  `json:"label"`
	Percentage float64 `json:"percentage" binding:"omitempty,gt=0,lte=100"`
	IsPrimary  bool    `json:"is_primary"`
	IsActive   bool    `json:"is_active"`
}

// WalletSummary provides a summary of wallet allocations
type WalletSummary struct {
	TotalWallets     int     `json:"total_wallets"`
	ActiveWallets    int     `json:"active_wallets"`
	TotalPercentage  float64 `json:"total_percentage"`
	RemainingPercent float64 `json:"remaining_percentage"`
	HasPrimaryWallet bool    `json:"has_primary_wallet"`
}

// PayoutSplit represents how a payout should be split across wallets
type PayoutSplit struct {
	WalletID   int64   `json:"wallet_id"`
	Address    string  `json:"address"`
	Percentage float64 `json:"percentage"`
	Amount     int64   `json:"amount"` // Amount in smallest unit (satoshi-like)
}

// CalculatePayoutSplits calculates how to split a payout amount across wallets
func CalculatePayoutSplits(wallets []UserWallet, totalAmount int64) []PayoutSplit {
	var splits []PayoutSplit
	var totalAllocated int64 = 0

	activeWallets := make([]UserWallet, 0)
	for _, w := range wallets {
		if w.IsActive {
			activeWallets = append(activeWallets, w)
		}
	}

	if len(activeWallets) == 0 {
		return splits
	}

	// Calculate split for each wallet
	for i, wallet := range activeWallets {
		var amount int64

		// Last wallet gets the remainder to avoid rounding issues
		if i == len(activeWallets)-1 {
			amount = totalAmount - totalAllocated
		} else {
			amount = int64(float64(totalAmount) * (wallet.Percentage / 100.0))
			totalAllocated += amount
		}

		splits = append(splits, PayoutSplit{
			WalletID:   wallet.ID,
			Address:    wallet.Address,
			Percentage: wallet.Percentage,
			Amount:     amount,
		})
	}

	return splits
}
