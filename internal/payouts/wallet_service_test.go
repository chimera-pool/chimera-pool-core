package payouts

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockWalletRepository implements WalletRepository for testing
type MockWalletRepository struct {
	wallets map[int64]*UserWallet
	nextID  int64
}

func NewMockWalletRepository() *MockWalletRepository {
	return &MockWalletRepository{
		wallets: make(map[int64]*UserWallet),
		nextID:  1,
	}
}

func (m *MockWalletRepository) CreateWallet(wallet *UserWallet) error {
	wallet.ID = m.nextID
	m.nextID++
	m.wallets[wallet.ID] = wallet
	return nil
}

func (m *MockWalletRepository) GetWalletByID(id int64) (*UserWallet, error) {
	wallet, exists := m.wallets[id]
	if !exists {
		return nil, ErrWalletNotFound
	}
	return wallet, nil
}

func (m *MockWalletRepository) GetWalletsByUserID(userID int64) ([]UserWallet, error) {
	var result []UserWallet
	for _, w := range m.wallets {
		if w.UserID == userID {
			result = append(result, *w)
		}
	}
	return result, nil
}

func (m *MockWalletRepository) GetActiveWalletsByUserID(userID int64) ([]UserWallet, error) {
	var result []UserWallet
	for _, w := range m.wallets {
		if w.UserID == userID && w.IsActive {
			result = append(result, *w)
		}
	}
	return result, nil
}

func (m *MockWalletRepository) UpdateWallet(wallet *UserWallet) error {
	if _, exists := m.wallets[wallet.ID]; !exists {
		return ErrWalletNotFound
	}
	m.wallets[wallet.ID] = wallet
	return nil
}

func (m *MockWalletRepository) DeleteWallet(id int64) error {
	if _, exists := m.wallets[id]; !exists {
		return ErrWalletNotFound
	}
	delete(m.wallets, id)
	return nil
}

func (m *MockWalletRepository) GetTotalPercentage(userID int64, excludeWalletID int64) (float64, error) {
	var total float64
	for _, w := range m.wallets {
		if w.UserID == userID && w.IsActive && w.ID != excludeWalletID {
			total += w.Percentage
		}
	}
	return total, nil
}

// Test: Creating a single wallet with 100% allocation
func TestCreateWallet_SingleWallet_100Percent(t *testing.T) {
	repo := NewMockWalletRepository()
	service := NewWalletService(repo)

	wallet, err := service.CreateWallet(1, CreateWalletRequest{
		Address:    "0x1234567890abcdef",
		Label:      "Main Wallet",
		Percentage: 100,
		IsPrimary:  true,
	})

	require.NoError(t, err)
	assert.Equal(t, int64(1), wallet.ID)
	assert.Equal(t, "0x1234567890abcdef", wallet.Address)
	assert.Equal(t, "Main Wallet", wallet.Label)
	assert.Equal(t, 100.0, wallet.Percentage)
	assert.True(t, wallet.IsPrimary)
	assert.True(t, wallet.IsActive)
}

// Test: Creating multiple wallets that sum to 100%
func TestCreateWallet_MultipleWallets_SumTo100(t *testing.T) {
	repo := NewMockWalletRepository()
	service := NewWalletService(repo)

	// Create first wallet with 70%
	wallet1, err := service.CreateWallet(1, CreateWalletRequest{
		Address:    "0xwallet1",
		Label:      "Primary",
		Percentage: 70,
		IsPrimary:  true,
	})
	require.NoError(t, err)
	assert.Equal(t, 70.0, wallet1.Percentage)

	// Create second wallet with 30%
	wallet2, err := service.CreateWallet(1, CreateWalletRequest{
		Address:    "0xwallet2",
		Label:      "Secondary",
		Percentage: 30,
		IsPrimary:  false,
	})
	require.NoError(t, err)
	assert.Equal(t, 30.0, wallet2.Percentage)

	// Verify summary
	summary, err := service.GetWalletSummary(1)
	require.NoError(t, err)
	assert.Equal(t, 2, summary.TotalWallets)
	assert.Equal(t, 2, summary.ActiveWallets)
	assert.Equal(t, 100.0, summary.TotalPercentage)
	assert.Equal(t, 0.0, summary.RemainingPercent)
}

// Test: Cannot exceed 100% allocation
func TestCreateWallet_ExceedsPercentage_Error(t *testing.T) {
	repo := NewMockWalletRepository()
	service := NewWalletService(repo)

	// Create first wallet with 80%
	_, err := service.CreateWallet(1, CreateWalletRequest{
		Address:    "0xwallet1",
		Label:      "Primary",
		Percentage: 80,
	})
	require.NoError(t, err)

	// Try to create second wallet with 30% (would exceed 100%)
	_, err = service.CreateWallet(1, CreateWalletRequest{
		Address:    "0xwallet2",
		Label:      "Secondary",
		Percentage: 30,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceed 100%")
}

// Test: Invalid percentage (0 or negative)
func TestCreateWallet_InvalidPercentage_Error(t *testing.T) {
	repo := NewMockWalletRepository()
	service := NewWalletService(repo)

	_, err := service.CreateWallet(1, CreateWalletRequest{
		Address:    "0xwallet1",
		Percentage: 0,
	})
	assert.ErrorIs(t, err, ErrInvalidPercentage)

	_, err = service.CreateWallet(1, CreateWalletRequest{
		Address:    "0xwallet1",
		Percentage: -10,
	})
	assert.ErrorIs(t, err, ErrInvalidPercentage)

	_, err = service.CreateWallet(1, CreateWalletRequest{
		Address:    "0xwallet1",
		Percentage: 150,
	})
	assert.ErrorIs(t, err, ErrInvalidPercentage)
}

// Test: Updating wallet percentage
func TestUpdateWallet_ChangePercentage(t *testing.T) {
	repo := NewMockWalletRepository()
	service := NewWalletService(repo)

	// Create two wallets
	wallet1, _ := service.CreateWallet(1, CreateWalletRequest{
		Address:    "0xwallet1",
		Percentage: 60,
	})
	_, _ = service.CreateWallet(1, CreateWalletRequest{
		Address:    "0xwallet2",
		Percentage: 40,
	})

	// Update first wallet to 50%
	updated, err := service.UpdateWallet(1, wallet1.ID, UpdateWalletRequest{
		Percentage: 50,
		IsActive:   true,
	})
	require.NoError(t, err)
	assert.Equal(t, 50.0, updated.Percentage)
}

// Test: Cannot update to exceed 100%
func TestUpdateWallet_ExceedsPercentage_Error(t *testing.T) {
	repo := NewMockWalletRepository()
	service := NewWalletService(repo)

	wallet1, _ := service.CreateWallet(1, CreateWalletRequest{
		Address:    "0xwallet1",
		Percentage: 50,
	})
	_, _ = service.CreateWallet(1, CreateWalletRequest{
		Address:    "0xwallet2",
		Percentage: 50,
	})

	// Try to update first wallet to 60% (would make total 110%)
	_, err := service.UpdateWallet(1, wallet1.ID, UpdateWalletRequest{
		Percentage: 60,
		IsActive:   true,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceed 100%")
}

// Test: Deleting a wallet
func TestDeleteWallet_Success(t *testing.T) {
	repo := NewMockWalletRepository()
	service := NewWalletService(repo)

	wallet, _ := service.CreateWallet(1, CreateWalletRequest{
		Address:    "0xwallet1",
		Percentage: 100,
	})

	err := service.DeleteWallet(1, wallet.ID)
	require.NoError(t, err)

	wallets, _ := service.GetUserWallets(1)
	assert.Len(t, wallets, 0)
}

// Test: Cannot delete wallet belonging to another user
func TestDeleteWallet_WrongUser_Error(t *testing.T) {
	repo := NewMockWalletRepository()
	service := NewWalletService(repo)

	wallet, _ := service.CreateWallet(1, CreateWalletRequest{
		Address:    "0xwallet1",
		Percentage: 100,
	})

	// Try to delete with different user ID
	err := service.DeleteWallet(2, wallet.ID)
	assert.ErrorIs(t, err, ErrWalletNotFound)
}

// Test: Payout split calculation - single wallet
func TestCalculatePayoutSplits_SingleWallet(t *testing.T) {
	wallets := []UserWallet{
		{ID: 1, Address: "0xwallet1", Percentage: 100, IsActive: true},
	}

	splits := CalculatePayoutSplits(wallets, 1000000000) // 10 BDAG

	require.Len(t, splits, 1)
	assert.Equal(t, int64(1000000000), splits[0].Amount)
	assert.Equal(t, "0xwallet1", splits[0].Address)
}

// Test: Payout split calculation - multiple wallets
func TestCalculatePayoutSplits_MultipleWallets(t *testing.T) {
	wallets := []UserWallet{
		{ID: 1, Address: "0xwallet1", Percentage: 70, IsActive: true},
		{ID: 2, Address: "0xwallet2", Percentage: 30, IsActive: true},
	}

	splits := CalculatePayoutSplits(wallets, 1000000000) // 10 BDAG

	require.Len(t, splits, 2)

	// First wallet gets 70%
	assert.Equal(t, int64(700000000), splits[0].Amount)
	assert.Equal(t, "0xwallet1", splits[0].Address)

	// Second wallet gets remaining 30%
	assert.Equal(t, int64(300000000), splits[1].Amount)
	assert.Equal(t, "0xwallet2", splits[1].Address)

	// Verify total equals original amount (no rounding loss)
	totalSplit := splits[0].Amount + splits[1].Amount
	assert.Equal(t, int64(1000000000), totalSplit)
}

// Test: Payout split - handles odd amounts correctly
func TestCalculatePayoutSplits_OddAmount_NoLoss(t *testing.T) {
	wallets := []UserWallet{
		{ID: 1, Address: "0xwallet1", Percentage: 33.33, IsActive: true},
		{ID: 2, Address: "0xwallet2", Percentage: 33.33, IsActive: true},
		{ID: 3, Address: "0xwallet3", Percentage: 33.34, IsActive: true},
	}

	splits := CalculatePayoutSplits(wallets, 1000000000) // 10 BDAG

	require.Len(t, splits, 3)

	// Verify total equals original amount (last wallet gets remainder)
	totalSplit := splits[0].Amount + splits[1].Amount + splits[2].Amount
	assert.Equal(t, int64(1000000000), totalSplit)
}

// Test: Inactive wallets excluded from splits
func TestCalculatePayoutSplits_InactiveWalletExcluded(t *testing.T) {
	wallets := []UserWallet{
		{ID: 1, Address: "0xwallet1", Percentage: 70, IsActive: true},
		{ID: 2, Address: "0xwallet2", Percentage: 30, IsActive: false}, // Inactive
	}

	splits := CalculatePayoutSplits(wallets, 1000000000)

	require.Len(t, splits, 1)
	assert.Equal(t, int64(1000000000), splits[0].Amount) // All goes to active wallet
}

// Test: No active wallets returns empty splits
func TestCalculatePayoutSplits_NoActiveWallets(t *testing.T) {
	wallets := []UserWallet{
		{ID: 1, Address: "0xwallet1", Percentage: 100, IsActive: false},
	}

	splits := CalculatePayoutSplits(wallets, 1000000000)
	assert.Len(t, splits, 0)
}

// Test: Wallet summary calculation
func TestGetWalletSummary(t *testing.T) {
	repo := NewMockWalletRepository()
	service := NewWalletService(repo)

	// Create wallets totaling 80%
	service.CreateWallet(1, CreateWalletRequest{
		Address:    "0xwallet1",
		Percentage: 50,
		IsPrimary:  true,
	})
	service.CreateWallet(1, CreateWalletRequest{
		Address:    "0xwallet2",
		Percentage: 30,
	})

	summary, err := service.GetWalletSummary(1)
	require.NoError(t, err)

	assert.Equal(t, 2, summary.TotalWallets)
	assert.Equal(t, 2, summary.ActiveWallets)
	assert.Equal(t, 80.0, summary.TotalPercentage)
	assert.Equal(t, 20.0, summary.RemainingPercent)
	assert.True(t, summary.HasPrimaryWallet)
}
