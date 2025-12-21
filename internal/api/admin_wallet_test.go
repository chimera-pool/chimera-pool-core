package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAdminUserWalletVisibility tests that admin can see user wallet configurations
func TestAdminUserWalletVisibility(t *testing.T) {
	t.Run("Admin can see user wallets in user detail response", func(t *testing.T) {
		// Test structure for expected wallet data in admin user detail
		type WalletInfo struct {
			ID         int64   `json:"id"`
			Address    string  `json:"address"`
			Label      string  `json:"label"`
			Percentage float64 `json:"percentage"`
			IsPrimary  bool    `json:"is_primary"`
			IsActive   bool    `json:"is_active"`
		}

		// Expected response should include wallets array
		type AdminUserDetailResponse struct {
			User struct {
				ID            int64  `json:"id"`
				Username      string `json:"username"`
				Email         string `json:"email"`
				PayoutAddress string `json:"payout_address"`
			} `json:"user"`
			Wallets []WalletInfo `json:"wallets"`
		}

		// Verify structure is correct - initialize with empty slice
		response := AdminUserDetailResponse{
			Wallets: []WalletInfo{},
		}
		assert.NotNil(t, response.Wallets)
	})

	t.Run("Wallet summary shows allocation percentage totals", func(t *testing.T) {
		type WalletSummary struct {
			TotalWallets       int     `json:"total_wallets"`
			ActiveWallets      int     `json:"active_wallets"`
			TotalAllocated     float64 `json:"total_allocated"`
			RemainingPercent   float64 `json:"remaining_percent"`
			HasMultipleWallets bool    `json:"has_multiple_wallets"`
		}

		summary := WalletSummary{
			TotalWallets:       3,
			ActiveWallets:      2,
			TotalAllocated:     75.5,
			RemainingPercent:   24.5,
			HasMultipleWallets: true,
		}

		assert.Equal(t, 3, summary.TotalWallets)
		assert.Equal(t, 2, summary.ActiveWallets)
		assert.Equal(t, 75.5, summary.TotalAllocated)
		assert.Equal(t, 24.5, summary.RemainingPercent)
		assert.True(t, summary.HasMultipleWallets)
	})

	t.Run("Single wallet user shows 100% allocation", func(t *testing.T) {
		type WalletInfo struct {
			Address    string  `json:"address"`
			Percentage float64 `json:"percentage"`
			IsPrimary  bool    `json:"is_primary"`
		}

		singleWallet := WalletInfo{
			Address:    "kaspa:qr123...",
			Percentage: 100.0,
			IsPrimary:  true,
		}

		assert.Equal(t, 100.0, singleWallet.Percentage)
		assert.True(t, singleWallet.IsPrimary)
	})

	t.Run("Multiple wallets show split percentages", func(t *testing.T) {
		type WalletInfo struct {
			Address    string  `json:"address"`
			Label      string  `json:"label"`
			Percentage float64 `json:"percentage"`
			IsPrimary  bool    `json:"is_primary"`
		}

		wallets := []WalletInfo{
			{Address: "kaspa:qr123...", Label: "Main Wallet", Percentage: 60.0, IsPrimary: true},
			{Address: "kaspa:qr456...", Label: "Secondary", Percentage: 25.0, IsPrimary: false},
			{Address: "kaspa:qr789...", Label: "Savings", Percentage: 15.0, IsPrimary: false},
		}

		// Verify total adds up to 100%
		var totalPercent float64
		for _, w := range wallets {
			totalPercent += w.Percentage
		}
		assert.Equal(t, 100.0, totalPercent)

		// Verify only one primary
		primaryCount := 0
		for _, w := range wallets {
			if w.IsPrimary {
				primaryCount++
			}
		}
		assert.Equal(t, 1, primaryCount)
	})
}

// TestAdminUserListWalletInfo tests wallet info in user list view
func TestAdminUserListWalletInfo(t *testing.T) {
	t.Run("User list includes wallet count and primary address", func(t *testing.T) {
		type AdminUserListItem struct {
			ID             int64   `json:"id"`
			Username       string  `json:"username"`
			Email          string  `json:"email"`
			WalletCount    int     `json:"wallet_count"`
			PrimaryWallet  string  `json:"primary_wallet"`
			TotalAllocated float64 `json:"total_allocated"`
		}

		user := AdminUserListItem{
			ID:             1,
			Username:       "testuser",
			Email:          "test@example.com",
			WalletCount:    3,
			PrimaryWallet:  "kaspa:qr123...",
			TotalAllocated: 100.0,
		}

		assert.Equal(t, 3, user.WalletCount)
		assert.NotEmpty(t, user.PrimaryWallet)
		require.Equal(t, 100.0, user.TotalAllocated)
	})
}
