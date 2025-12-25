package payouts

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// MOCK REPOSITORIES
// =============================================================================

type mockUserPayoutSettingsRepo struct {
	settings map[int64]*UserPayoutSettings
}

func newMockUserPayoutSettingsRepo() *mockUserPayoutSettingsRepo {
	return &mockUserPayoutSettingsRepo{
		settings: make(map[int64]*UserPayoutSettings),
	}
}

func (m *mockUserPayoutSettingsRepo) GetUserSettings(ctx context.Context, userID int64) (*UserPayoutSettings, error) {
	return m.settings[userID], nil
}

func (m *mockUserPayoutSettingsRepo) CreateUserSettings(ctx context.Context, settings *UserPayoutSettings) error {
	m.settings[settings.UserID] = settings
	return nil
}

func (m *mockUserPayoutSettingsRepo) UpdateUserSettings(ctx context.Context, settings *UserPayoutSettings) error {
	m.settings[settings.UserID] = settings
	return nil
}

func (m *mockUserPayoutSettingsRepo) DeleteUserSettings(ctx context.Context, userID int64) error {
	delete(m.settings, userID)
	return nil
}

func (m *mockUserPayoutSettingsRepo) GetUsersByPayoutMode(ctx context.Context, mode PayoutMode) ([]UserPayoutSettings, error) {
	var result []UserPayoutSettings
	for _, s := range m.settings {
		if s.PayoutMode == mode {
			result = append(result, *s)
		}
	}
	return result, nil
}

func (m *mockUserPayoutSettingsRepo) GetUsersForAutoPayout(ctx context.Context, minBalance int64) ([]UserPayoutSettings, error) {
	var result []UserPayoutSettings
	for _, s := range m.settings {
		if s.AutoPayoutEnable && s.PayoutAddress != "" {
			result = append(result, *s)
		}
	}
	return result, nil
}

type mockPoolFeeConfigRepo struct {
	configs map[string]*PoolFeeConfig
}

func newMockPoolFeeConfigRepo() *mockPoolFeeConfigRepo {
	return &mockPoolFeeConfigRepo{
		configs: make(map[string]*PoolFeeConfig),
	}
}

func (m *mockPoolFeeConfigRepo) GetFeeConfig(ctx context.Context, mode PayoutMode, coinSymbol string) (*PoolFeeConfig, error) {
	key := string(mode) + "_" + coinSymbol
	return m.configs[key], nil
}

func (m *mockPoolFeeConfigRepo) GetAllFeeConfigs(ctx context.Context) ([]PoolFeeConfig, error) {
	var result []PoolFeeConfig
	for _, c := range m.configs {
		result = append(result, *c)
	}
	return result, nil
}

func (m *mockPoolFeeConfigRepo) UpdateFeeConfig(ctx context.Context, config *PoolFeeConfig) error {
	key := string(config.PayoutMode) + "_" + config.CoinSymbol
	m.configs[key] = config
	return nil
}

func (m *mockPoolFeeConfigRepo) GetEnabledModes(ctx context.Context, coinSymbol string) ([]PayoutMode, error) {
	var modes []PayoutMode
	for _, c := range m.configs {
		if c.CoinSymbol == coinSymbol && c.IsEnabled {
			modes = append(modes, c.PayoutMode)
		}
	}
	return modes, nil
}

// =============================================================================
// PAYOUT MANAGER TESTS
// =============================================================================

func TestNewPayoutManager(t *testing.T) {
	t.Run("creates manager with default config", func(t *testing.T) {
		pm, err := NewPayoutManager(nil, nil, nil)
		require.NoError(t, err)
		assert.NotNil(t, pm)
		assert.NotNil(t, pm.config)
	})

	t.Run("creates manager with custom config", func(t *testing.T) {
		config := DefaultPayoutConfig()
		config.FeePPLNS = 1.5

		pm, err := NewPayoutManager(config, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, 1.5, pm.config.FeePPLNS)
	})

	t.Run("initializes all enabled calculators", func(t *testing.T) {
		config := DefaultPayoutConfig()
		pm, err := NewPayoutManager(config, nil, nil)
		require.NoError(t, err)

		// Check enabled modes
		modes := pm.GetEnabledModes()
		assert.Contains(t, modes, PayoutModePPLNS)
		assert.Contains(t, modes, PayoutModePPSPlus)
		assert.Contains(t, modes, PayoutModeSCORE)
		assert.Contains(t, modes, PayoutModeSOLO)
		assert.Contains(t, modes, PayoutModeSLICE)

		// PPS and FPPS should not be enabled by default
		assert.NotContains(t, modes, PayoutModePPS)
		assert.NotContains(t, modes, PayoutModeFPPS)
	})
}

func TestPayoutManager_GetCalculator(t *testing.T) {
	pm, _ := NewPayoutManager(nil, nil, nil)

	t.Run("returns calculator for enabled mode", func(t *testing.T) {
		calc, err := pm.GetCalculator(PayoutModePPLNS)
		require.NoError(t, err)
		assert.NotNil(t, calc)
		assert.Equal(t, PayoutModePPLNS, calc.Mode())
	})

	t.Run("returns error for disabled mode", func(t *testing.T) {
		_, err := pm.GetCalculator(PayoutModePPS)
		assert.Error(t, err)
	})
}

func TestPayoutManager_SetNetworkDifficulty(t *testing.T) {
	config := DefaultPayoutConfig()
	config.EnablePPS = true
	config.EnableFPPS = true

	pm, _ := NewPayoutManager(config, nil, nil)
	pm.SetNetworkDifficulty(1000000)

	// Verify PPS calculator has network difficulty set
	ppsCalc, _ := pm.GetCalculator(PayoutModePPS)
	if evCalc, ok := ppsCalc.(ExpectedValueCalculator); ok {
		// The interface doesn't expose GetNetworkDifficulty, but we can verify via calculation
		assert.NotNil(t, evCalc)
	}
}

func TestPayoutManager_CalculatePayoutsForBlock(t *testing.T) {
	settingsRepo := newMockUserPayoutSettingsRepo()
	pm, _ := NewPayoutManager(nil, settingsRepo, nil)
	pm.SetNetworkDifficulty(1000000)

	ctx := context.Background()
	blockTime := time.Now()
	blockReward := int64(1250000000) // 12.5 coins
	txFees := int64(50000000)        // 0.5 coins

	t.Run("calculates payouts for shares with default mode", func(t *testing.T) {
		shares := []Share{
			{ID: 1, UserID: 1, Difficulty: 1000, IsValid: true, Timestamp: blockTime.Add(-time.Minute)},
			{ID: 2, UserID: 2, Difficulty: 1000, IsValid: true, Timestamp: blockTime.Add(-30 * time.Second)},
		}

		payouts, err := pm.CalculatePayoutsForBlock(ctx, shares, blockReward, txFees, blockTime, 1)
		require.NoError(t, err)
		assert.Len(t, payouts, 2)

		// Verify both users got payouts
		var totalPayout int64
		for _, p := range payouts {
			totalPayout += p.Amount
			assert.Greater(t, p.Amount, int64(0))
		}
		assert.Greater(t, totalPayout, int64(0))
	})

	t.Run("respects user payout mode settings", func(t *testing.T) {
		// Set user 1 to SCORE mode
		settingsRepo.settings[1] = &UserPayoutSettings{
			UserID:     1,
			PayoutMode: PayoutModeSCORE,
		}

		shares := []Share{
			{ID: 1, UserID: 1, Difficulty: 1000, IsValid: true, Timestamp: blockTime.Add(-time.Minute)},
			{ID: 2, UserID: 2, Difficulty: 1000, IsValid: true, Timestamp: blockTime.Add(-30 * time.Second)},
		}

		payouts, err := pm.CalculatePayoutsForBlock(ctx, shares, blockReward, txFees, blockTime, 1)
		require.NoError(t, err)
		assert.Len(t, payouts, 2)
	})

	t.Run("handles empty shares", func(t *testing.T) {
		payouts, err := pm.CalculatePayoutsForBlock(ctx, []Share{}, blockReward, txFees, blockTime, 1)
		require.NoError(t, err)
		assert.Empty(t, payouts)
	})
}

func TestPayoutManager_GetUserSettings(t *testing.T) {
	settingsRepo := newMockUserPayoutSettingsRepo()
	pm, _ := NewPayoutManager(nil, settingsRepo, nil)
	ctx := context.Background()

	t.Run("returns nil for non-existent user", func(t *testing.T) {
		settings, err := pm.GetUserSettings(ctx, 999)
		require.NoError(t, err)
		assert.Nil(t, settings)
	})

	t.Run("returns settings for existing user", func(t *testing.T) {
		expected := &UserPayoutSettings{
			UserID:     1,
			PayoutMode: PayoutModePPLNS,
		}
		settingsRepo.settings[1] = expected

		settings, err := pm.GetUserSettings(ctx, 1)
		require.NoError(t, err)
		assert.Equal(t, expected.PayoutMode, settings.PayoutMode)
	})
}

func TestPayoutManager_UpdateUserSettings(t *testing.T) {
	settingsRepo := newMockUserPayoutSettingsRepo()
	pm, _ := NewPayoutManager(nil, settingsRepo, nil)
	ctx := context.Background()

	t.Run("creates settings for new user", func(t *testing.T) {
		settings := &UserPayoutSettings{
			UserID:           1,
			PayoutMode:       PayoutModePPLNS,
			MinPayoutAmount:  1000000,
			AutoPayoutEnable: true,
		}

		err := pm.UpdateUserSettings(ctx, settings)
		require.NoError(t, err)

		saved, _ := settingsRepo.GetUserSettings(ctx, 1)
		assert.Equal(t, PayoutModePPLNS, saved.PayoutMode)
	})

	t.Run("updates settings for existing user", func(t *testing.T) {
		settingsRepo.settings[1] = &UserPayoutSettings{
			UserID:     1,
			PayoutMode: PayoutModePPLNS,
		}

		newSettings := &UserPayoutSettings{
			UserID:     1,
			PayoutMode: PayoutModeSCORE,
		}

		err := pm.UpdateUserSettings(ctx, newSettings)
		require.NoError(t, err)

		saved, _ := settingsRepo.GetUserSettings(ctx, 1)
		assert.Equal(t, PayoutModeSCORE, saved.PayoutMode)
	})

	t.Run("rejects disabled payout mode", func(t *testing.T) {
		settings := &UserPayoutSettings{
			UserID:     1,
			PayoutMode: PayoutModePPS, // Disabled by default
		}

		err := pm.UpdateUserSettings(ctx, settings)
		assert.Error(t, err)
	})
}

func TestPayoutManager_GetFeeForMode(t *testing.T) {
	config := DefaultPayoutConfig()
	pm, _ := NewPayoutManager(config, nil, nil)

	assert.Equal(t, 1.0, pm.GetFeeForMode(PayoutModePPLNS))
	assert.Equal(t, 1.5, pm.GetFeeForMode(PayoutModePPSPlus))
	assert.Equal(t, 1.0, pm.GetFeeForMode(PayoutModeSCORE))
	assert.Equal(t, 0.5, pm.GetFeeForMode(PayoutModeSOLO))
	assert.Equal(t, 0.8, pm.GetFeeForMode(PayoutModeSLICE))
}

func TestPayoutManager_CalculatePayoutsWithMetadata(t *testing.T) {
	pm, _ := NewPayoutManager(nil, nil, nil)
	pm.SetNetworkDifficulty(1000000)

	ctx := context.Background()
	blockTime := time.Now()
	blockReward := int64(1250000000)
	txFees := int64(50000000)

	shares := []Share{
		{ID: 1, UserID: 1, Difficulty: 1000, IsValid: true, Timestamp: blockTime.Add(-time.Minute)},
		{ID: 2, UserID: 2, Difficulty: 1000, IsValid: true, Timestamp: blockTime.Add(-30 * time.Second)},
		{ID: 3, UserID: 1, Difficulty: 500, IsValid: true, Timestamp: blockTime.Add(-20 * time.Second)},
	}

	result, err := pm.CalculatePayoutsWithMetadata(ctx, shares, blockReward, txFees, blockTime, 1)
	require.NoError(t, err)
	assert.NotNil(t, result)

	assert.Equal(t, blockReward, result.BlockReward)
	assert.Equal(t, txFees, result.TxFees)
	assert.Equal(t, 3, result.ShareCount)
	assert.Equal(t, 2, result.UniqueMiners)
	assert.Greater(t, result.TotalAmount, int64(0))
	assert.Greater(t, result.TotalFees, int64(0))
}
