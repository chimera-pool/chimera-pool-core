package payouts

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// =============================================================================
// MULTI-MODE PAYOUT MANAGER
// =============================================================================

// PayoutManager manages multiple payout calculators and routes payouts based on user settings
type PayoutManager struct {
	calculators    map[PayoutMode]PayoutCalculator
	config         *PayoutConfig
	settingsRepo   UserPayoutSettingsRepository
	feeConfigRepo  PoolFeeConfigRepository
	networkDiff    float64
	expectedTxFees int64
	mu             sync.RWMutex
}

// NewPayoutManager creates a new payout manager with all supported calculators
func NewPayoutManager(
	config *PayoutConfig,
	settingsRepo UserPayoutSettingsRepository,
	feeConfigRepo PoolFeeConfigRepository,
) (*PayoutManager, error) {
	if config == nil {
		config = DefaultPayoutConfig()
	}

	pm := &PayoutManager{
		calculators:   make(map[PayoutMode]PayoutCalculator),
		config:        config,
		settingsRepo:  settingsRepo,
		feeConfigRepo: feeConfigRepo,
	}

	// Initialize all calculators
	if err := pm.initializeCalculators(); err != nil {
		return nil, fmt.Errorf("failed to initialize calculators: %w", err)
	}

	return pm, nil
}

// initializeCalculators creates all payout calculators based on config
func (pm *PayoutManager) initializeCalculators() error {
	// PPLNS
	if pm.config.EnablePPLNS {
		calc, err := NewPPLNSCalculator(pm.config.PPLNSWindowSize, pm.config.FeePPLNS)
		if err != nil {
			return fmt.Errorf("failed to create PPLNS calculator: %w", err)
		}
		pm.calculators[PayoutModePPLNS] = calc
	}

	// PPS
	if pm.config.EnablePPS {
		calc, err := NewPPSCalculator(pm.config.FeePPS)
		if err != nil {
			return fmt.Errorf("failed to create PPS calculator: %w", err)
		}
		pm.calculators[PayoutModePPS] = calc
	}

	// PPS+
	if pm.config.EnablePPSPlus {
		calc, err := NewPPSPlusCalculator(pm.config.FeePPSPlus, pm.config.PPLNSWindowSize)
		if err != nil {
			return fmt.Errorf("failed to create PPS+ calculator: %w", err)
		}
		pm.calculators[PayoutModePPSPlus] = calc
	}

	// FPPS
	if pm.config.EnableFPPS {
		calc, err := NewFPPSCalculator(pm.config.FeeFPPS)
		if err != nil {
			return fmt.Errorf("failed to create FPPS calculator: %w", err)
		}
		pm.calculators[PayoutModeFPPS] = calc
	}

	// SCORE
	if pm.config.EnableSCORE {
		calc, err := NewSCORECalculator(
			pm.config.PPLNSWindowSize,
			pm.config.FeeSCORE,
			pm.config.SCOREDecayFactor,
		)
		if err != nil {
			return fmt.Errorf("failed to create SCORE calculator: %w", err)
		}
		pm.calculators[PayoutModeSCORE] = calc
	}

	// SOLO
	if pm.config.EnableSOLO {
		calc, err := NewSOLOCalculator(pm.config.FeeSOLO)
		if err != nil {
			return fmt.Errorf("failed to create SOLO calculator: %w", err)
		}
		pm.calculators[PayoutModeSOLO] = calc
	}

	// SLICE (placeholder - will be fully implemented in Phase 6)
	if pm.config.EnableSLICE {
		// For now, SLICE uses enhanced PPLNS with slice-specific config
		calc, err := NewPPLNSCalculator(
			pm.config.SLICEWindowSize*pm.config.SLICESliceDuration,
			pm.config.FeeSLICE,
		)
		if err != nil {
			return fmt.Errorf("failed to create SLICE calculator: %w", err)
		}
		pm.calculators[PayoutModeSLICE] = calc
	}

	return nil
}

// GetCalculator returns the calculator for a specific payout mode
func (pm *PayoutManager) GetCalculator(mode PayoutMode) (PayoutCalculator, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	calc, exists := pm.calculators[mode]
	if !exists {
		return nil, fmt.Errorf("payout mode %s is not enabled", mode)
	}
	return calc, nil
}

// GetEnabledModes returns all enabled payout modes
func (pm *PayoutManager) GetEnabledModes() []PayoutMode {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	modes := make([]PayoutMode, 0, len(pm.calculators))
	for mode := range pm.calculators {
		modes = append(modes, mode)
	}
	return modes
}

// SetNetworkDifficulty updates network difficulty for all ExpectedValue calculators
func (pm *PayoutManager) SetNetworkDifficulty(difficulty float64) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.networkDiff = difficulty

	for _, calc := range pm.calculators {
		if evCalc, ok := calc.(ExpectedValueCalculator); ok {
			evCalc.SetNetworkDifficulty(difficulty)
		}
	}
}

// SetExpectedTxFees updates expected tx fees for FPPS calculator
func (pm *PayoutManager) SetExpectedTxFees(fees int64) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.expectedTxFees = fees

	for _, calc := range pm.calculators {
		if evCalc, ok := calc.(ExpectedValueCalculator); ok {
			evCalc.SetExpectedTxFees(fees)
		}
	}
}

// CalculatePayoutsForBlock calculates payouts for all users based on their payout mode preferences
func (pm *PayoutManager) CalculatePayoutsForBlock(
	ctx context.Context,
	shares []Share,
	blockReward int64,
	txFees int64,
	blockTime time.Time,
	blockFinderID int64,
) ([]Payout, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// Group shares by user's payout mode
	sharesByMode := make(map[PayoutMode][]Share)

	for _, share := range shares {
		mode := pm.getUserPayoutMode(ctx, share.UserID)
		sharesByMode[mode] = append(sharesByMode[mode], share)
	}

	// Calculate payouts for each mode
	allPayouts := make([]Payout, 0)

	for mode, modeShares := range sharesByMode {
		calc, exists := pm.calculators[mode]
		if !exists {
			// Fall back to PPLNS if mode not available
			calc = pm.calculators[PayoutModePPLNS]
			if calc == nil {
				continue
			}
		}

		// Calculate mode-specific reward portion
		modeReward, modeTxFees := pm.calculateModeRewardPortion(
			mode, modeShares, shares, blockReward, txFees,
		)

		payouts, err := calc.CalculatePayouts(modeShares, modeReward, modeTxFees, blockTime)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate %s payouts: %w", mode, err)
		}

		allPayouts = append(allPayouts, payouts...)
	}

	return allPayouts, nil
}

// getUserPayoutMode retrieves the payout mode for a user
func (pm *PayoutManager) getUserPayoutMode(ctx context.Context, userID int64) PayoutMode {
	if pm.settingsRepo == nil {
		return pm.config.DefaultMode
	}

	settings, err := pm.settingsRepo.GetUserSettings(ctx, userID)
	if err != nil || settings == nil {
		return pm.config.DefaultMode
	}

	// Verify the mode is enabled
	if _, exists := pm.calculators[settings.PayoutMode]; !exists {
		return pm.config.DefaultMode
	}

	return settings.PayoutMode
}

// calculateModeRewardPortion calculates the reward portion for a specific mode
func (pm *PayoutManager) calculateModeRewardPortion(
	mode PayoutMode,
	modeShares []Share,
	allShares []Share,
	totalReward int64,
	totalTxFees int64,
) (int64, int64) {
	// Calculate total difficulty
	var totalDiff, modeDiff float64
	for _, s := range allShares {
		if s.IsValid {
			totalDiff += s.Difficulty
		}
	}
	for _, s := range modeShares {
		if s.IsValid {
			modeDiff += s.Difficulty
		}
	}

	if totalDiff == 0 {
		return 0, 0
	}

	// Proportion of reward based on difficulty contribution
	proportion := modeDiff / totalDiff
	modeReward := int64(float64(totalReward) * proportion)
	modeTxFees := int64(float64(totalTxFees) * proportion)

	return modeReward, modeTxFees
}

// GetUserSettings retrieves payout settings for a user
func (pm *PayoutManager) GetUserSettings(ctx context.Context, userID int64) (*UserPayoutSettings, error) {
	if pm.settingsRepo == nil {
		return nil, fmt.Errorf("settings repository not configured")
	}
	return pm.settingsRepo.GetUserSettings(ctx, userID)
}

// UpdateUserSettings updates payout settings for a user
func (pm *PayoutManager) UpdateUserSettings(ctx context.Context, settings *UserPayoutSettings) error {
	if pm.settingsRepo == nil {
		return fmt.Errorf("settings repository not configured")
	}

	// Validate payout mode is enabled
	if _, exists := pm.calculators[settings.PayoutMode]; !exists {
		return fmt.Errorf("payout mode %s is not enabled", settings.PayoutMode)
	}

	// Get existing settings
	existing, err := pm.settingsRepo.GetUserSettings(ctx, settings.UserID)
	if err != nil {
		return err
	}

	if existing == nil {
		return pm.settingsRepo.CreateUserSettings(ctx, settings)
	}
	return pm.settingsRepo.UpdateUserSettings(ctx, settings)
}

// GetFeeForMode returns the configured fee for a payout mode
func (pm *PayoutManager) GetFeeForMode(mode PayoutMode) float64 {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if calc, exists := pm.calculators[mode]; exists {
		return calc.GetPoolFeePercent()
	}
	return pm.config.GetFeeForMode(mode)
}

// GetConfig returns the current payout configuration
func (pm *PayoutManager) GetConfig() *PayoutConfig {
	return pm.config
}

// =============================================================================
// PAYOUT RESULT AGGREGATION
// =============================================================================

// AggregatedPayoutResult contains the result of payout calculation with metadata
type AggregatedPayoutResult struct {
	Payouts      []Payout                `json:"payouts"`
	TotalAmount  int64                   `json:"total_amount"`
	TotalFees    int64                   `json:"total_fees"`
	ByMode       map[PayoutMode][]Payout `json:"by_mode"`
	BlockReward  int64                   `json:"block_reward"`
	TxFees       int64                   `json:"tx_fees"`
	BlockTime    time.Time               `json:"block_time"`
	ShareCount   int                     `json:"share_count"`
	UniqueMiners int                     `json:"unique_miners"`
}

// CalculatePayoutsWithMetadata calculates payouts and returns detailed result
func (pm *PayoutManager) CalculatePayoutsWithMetadata(
	ctx context.Context,
	shares []Share,
	blockReward int64,
	txFees int64,
	blockTime time.Time,
	blockFinderID int64,
) (*AggregatedPayoutResult, error) {
	payouts, err := pm.CalculatePayoutsForBlock(ctx, shares, blockReward, txFees, blockTime, blockFinderID)
	if err != nil {
		return nil, err
	}

	result := &AggregatedPayoutResult{
		Payouts:     payouts,
		BlockReward: blockReward,
		TxFees:      txFees,
		BlockTime:   blockTime,
		ShareCount:  len(shares),
		ByMode:      make(map[PayoutMode][]Payout),
	}

	// Aggregate results
	uniqueMiners := make(map[int64]bool)
	for _, p := range payouts {
		result.TotalAmount += p.Amount
		uniqueMiners[p.UserID] = true
	}
	result.UniqueMiners = len(uniqueMiners)

	// Calculate total fees
	result.TotalFees = (blockReward + txFees) - result.TotalAmount

	// Group by mode (would need payout mode in Payout struct for full implementation)
	result.ByMode[PayoutModePPLNS] = payouts // Simplified for now

	return result, nil
}
