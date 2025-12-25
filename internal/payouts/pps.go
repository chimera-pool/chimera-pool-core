package payouts

import (
	"fmt"
	"time"
)

// =============================================================================
// PPS (Pay Per Share) CALCULATOR
// Pool pays fixed amount per share based on expected value
// Pool absorbs all block finding variance
// =============================================================================

// PPSCalculator implements Pay Per Share payout calculation
type PPSCalculator struct {
	poolFeePercent    float64 // Pool fee percentage (0-100)
	networkDifficulty float64 // Current network difficulty
	expectedTxFees    int64   // Expected transaction fees per block
}

// NewPPSCalculator creates a new PPS calculator
func NewPPSCalculator(poolFeePercent float64) (*PPSCalculator, error) {
	if poolFeePercent < 0 || poolFeePercent > 100 {
		return nil, fmt.Errorf("pool fee percent must be between 0 and 100, got %.2f", poolFeePercent)
	}

	return &PPSCalculator{
		poolFeePercent:    poolFeePercent,
		networkDifficulty: 1, // Will be set dynamically
		expectedTxFees:    0,
	}, nil
}

// Mode returns the payout mode
func (c *PPSCalculator) Mode() PayoutMode {
	return PayoutModePPS
}

// CalculatePayouts calculates PPS payouts
// In PPS, payouts are calculated per share submission, not per block
// This method is for batch processing - calculates what each user earned from their shares
func (c *PPSCalculator) CalculatePayouts(shares []Share, blockReward int64, txFees int64, blockTime time.Time) ([]Payout, error) {
	if len(shares) == 0 {
		return []Payout{}, nil
	}

	if c.networkDifficulty <= 0 {
		return nil, fmt.Errorf("network difficulty must be positive, got %.2f", c.networkDifficulty)
	}

	// PPS value per unit of difficulty
	// Expected value = BlockReward / NetworkDifficulty
	// This is what a share of difficulty 1 is worth on average
	expectedValuePerDiff := float64(blockReward) / c.networkDifficulty

	// Apply pool fee
	feeMultiplier := 1.0 - (c.poolFeePercent / 100.0)
	ppsValuePerDiff := expectedValuePerDiff * feeMultiplier

	// Aggregate payouts by user
	userPayouts := make(map[int64]int64)
	for _, share := range shares {
		if !share.IsValid {
			continue
		}
		// Each share earns: shareDifficulty * ppsValuePerDiff
		earned := int64(share.Difficulty * ppsValuePerDiff)
		userPayouts[share.UserID] += earned
	}

	// Convert to payout slice
	payouts := make([]Payout, 0, len(userPayouts))
	for userID, amount := range userPayouts {
		if amount > 0 {
			payouts = append(payouts, Payout{
				UserID:    userID,
				Amount:    amount,
				Timestamp: blockTime,
			})
		}
	}

	return payouts, nil
}

// CalculateShareValue calculates the PPS value of a single share
func (c *PPSCalculator) CalculateShareValue(shareDifficulty float64, blockReward int64) int64 {
	if c.networkDifficulty <= 0 {
		return 0
	}

	expectedValuePerDiff := float64(blockReward) / c.networkDifficulty
	feeMultiplier := 1.0 - (c.poolFeePercent / 100.0)
	return int64(shareDifficulty * expectedValuePerDiff * feeMultiplier)
}

// GetPoolFeePercent returns the pool fee percentage
func (c *PPSCalculator) GetPoolFeePercent() float64 {
	return c.poolFeePercent
}

// SetPoolFeePercent sets the pool fee percentage
func (c *PPSCalculator) SetPoolFeePercent(fee float64) error {
	if fee < 0 || fee > 100 {
		return fmt.Errorf("pool fee percent must be between 0 and 100, got %.2f", fee)
	}
	c.poolFeePercent = fee
	return nil
}

// SetNetworkDifficulty sets the current network difficulty
func (c *PPSCalculator) SetNetworkDifficulty(difficulty float64) {
	c.networkDifficulty = difficulty
}

// SetExpectedTxFees sets the expected transaction fees (not used in basic PPS)
func (c *PPSCalculator) SetExpectedTxFees(fees int64) {
	c.expectedTxFees = fees
}

// ValidateConfiguration validates the calculator configuration
func (c *PPSCalculator) ValidateConfiguration() error {
	if c.poolFeePercent < 0 || c.poolFeePercent > 100 {
		return fmt.Errorf("invalid pool fee percent: %.2f", c.poolFeePercent)
	}
	if c.networkDifficulty <= 0 {
		return fmt.Errorf("network difficulty must be set before calculating payouts")
	}
	return nil
}

// =============================================================================
// PPS+ (PPS Plus) CALCULATOR
// PPS for block reward + PPLNS for transaction fees
// =============================================================================

// PPSPlusCalculator implements PPS+ payout calculation
type PPSPlusCalculator struct {
	ppsCalculator  *PPSCalculator
	pplnsWindow    int64   // Window size for tx fee PPLNS calculation
	poolFeePercent float64 // Pool fee percentage (0-100)
}

// NewPPSPlusCalculator creates a new PPS+ calculator
func NewPPSPlusCalculator(poolFeePercent float64, pplnsWindow int64) (*PPSPlusCalculator, error) {
	if poolFeePercent < 0 || poolFeePercent > 100 {
		return nil, fmt.Errorf("pool fee percent must be between 0 and 100, got %.2f", poolFeePercent)
	}

	if pplnsWindow <= 0 {
		return nil, fmt.Errorf("PPLNS window must be positive, got %d", pplnsWindow)
	}

	pps, err := NewPPSCalculator(poolFeePercent)
	if err != nil {
		return nil, err
	}

	return &PPSPlusCalculator{
		ppsCalculator:  pps,
		pplnsWindow:    pplnsWindow,
		poolFeePercent: poolFeePercent,
	}, nil
}

// Mode returns the payout mode
func (c *PPSPlusCalculator) Mode() PayoutMode {
	return PayoutModePPSPlus
}

// CalculatePayouts calculates PPS+ payouts
// Block reward is paid via PPS, transaction fees via PPLNS
func (c *PPSPlusCalculator) CalculatePayouts(shares []Share, blockReward int64, txFees int64, blockTime time.Time) ([]Payout, error) {
	if len(shares) == 0 {
		return []Payout{}, nil
	}

	// Calculate PPS portion (block reward only)
	ppsPayouts, err := c.ppsCalculator.CalculatePayouts(shares, blockReward, 0, blockTime)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate PPS portion: %w", err)
	}

	// Calculate PPLNS portion for transaction fees
	pplnsCalc, err := NewPPLNSCalculator(c.pplnsWindow, c.poolFeePercent)
	if err != nil {
		return nil, fmt.Errorf("failed to create PPLNS calculator: %w", err)
	}

	pplnsPayouts, err := pplnsCalc.CalculatePayouts(shares, txFees, 0, blockTime)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate PPLNS portion: %w", err)
	}

	// Merge payouts
	mergedPayouts := make(map[int64]int64)
	for _, p := range ppsPayouts {
		mergedPayouts[p.UserID] += p.Amount
	}
	for _, p := range pplnsPayouts {
		mergedPayouts[p.UserID] += p.Amount
	}

	// Convert to slice
	payouts := make([]Payout, 0, len(mergedPayouts))
	for userID, amount := range mergedPayouts {
		if amount > 0 {
			payouts = append(payouts, Payout{
				UserID:    userID,
				Amount:    amount,
				Timestamp: blockTime,
			})
		}
	}

	return payouts, nil
}

// GetPoolFeePercent returns the pool fee percentage
func (c *PPSPlusCalculator) GetPoolFeePercent() float64 {
	return c.poolFeePercent
}

// SetPoolFeePercent sets the pool fee percentage
func (c *PPSPlusCalculator) SetPoolFeePercent(fee float64) error {
	if fee < 0 || fee > 100 {
		return fmt.Errorf("pool fee percent must be between 0 and 100, got %.2f", fee)
	}
	c.poolFeePercent = fee
	c.ppsCalculator.poolFeePercent = fee
	return nil
}

// SetNetworkDifficulty sets the current network difficulty for PPS calculation
func (c *PPSPlusCalculator) SetNetworkDifficulty(difficulty float64) {
	c.ppsCalculator.SetNetworkDifficulty(difficulty)
}

// SetExpectedTxFees sets expected tx fees (informational)
func (c *PPSPlusCalculator) SetExpectedTxFees(fees int64) {
	c.ppsCalculator.SetExpectedTxFees(fees)
}

// GetWindowSize returns the PPLNS window size for tx fee calculation
func (c *PPSPlusCalculator) GetWindowSize() int64 {
	return c.pplnsWindow
}

// SetWindowSize sets the PPLNS window size
func (c *PPSPlusCalculator) SetWindowSize(size int64) error {
	if size <= 0 {
		return fmt.Errorf("window size must be positive, got %d", size)
	}
	c.pplnsWindow = size
	return nil
}

// ValidateConfiguration validates the calculator configuration
func (c *PPSPlusCalculator) ValidateConfiguration() error {
	if c.poolFeePercent < 0 || c.poolFeePercent > 100 {
		return fmt.Errorf("invalid pool fee percent: %.2f", c.poolFeePercent)
	}
	if c.pplnsWindow <= 0 {
		return fmt.Errorf("invalid PPLNS window: %d", c.pplnsWindow)
	}
	return c.ppsCalculator.ValidateConfiguration()
}

// =============================================================================
// FPPS (Full Pay Per Share) CALCULATOR
// Pool pays expected block reward + expected transaction fees per share
// =============================================================================

// FPPSCalculator implements Full Pay Per Share payout calculation
type FPPSCalculator struct {
	poolFeePercent    float64 // Pool fee percentage (0-100)
	networkDifficulty float64 // Current network difficulty
	expectedTxFees    int64   // Expected average transaction fees per block
}

// NewFPPSCalculator creates a new FPPS calculator
func NewFPPSCalculator(poolFeePercent float64) (*FPPSCalculator, error) {
	if poolFeePercent < 0 || poolFeePercent > 100 {
		return nil, fmt.Errorf("pool fee percent must be between 0 and 100, got %.2f", poolFeePercent)
	}

	return &FPPSCalculator{
		poolFeePercent:    poolFeePercent,
		networkDifficulty: 1,
		expectedTxFees:    0,
	}, nil
}

// Mode returns the payout mode
func (c *FPPSCalculator) Mode() PayoutMode {
	return PayoutModeFPPS
}

// CalculatePayouts calculates FPPS payouts
// Pays expected (blockReward + txFees) / networkDifficulty per unit difficulty
func (c *FPPSCalculator) CalculatePayouts(shares []Share, blockReward int64, txFees int64, blockTime time.Time) ([]Payout, error) {
	if len(shares) == 0 {
		return []Payout{}, nil
	}

	if c.networkDifficulty <= 0 {
		return nil, fmt.Errorf("network difficulty must be positive, got %.2f", c.networkDifficulty)
	}

	// Use provided txFees or fall back to expected
	effectiveTxFees := txFees
	if effectiveTxFees == 0 {
		effectiveTxFees = c.expectedTxFees
	}

	// FPPS value per unit of difficulty
	// Expected value = (BlockReward + TxFees) / NetworkDifficulty
	totalReward := blockReward + effectiveTxFees
	expectedValuePerDiff := float64(totalReward) / c.networkDifficulty

	// Apply pool fee
	feeMultiplier := 1.0 - (c.poolFeePercent / 100.0)
	fppsValuePerDiff := expectedValuePerDiff * feeMultiplier

	// Aggregate payouts by user
	userPayouts := make(map[int64]int64)
	for _, share := range shares {
		if !share.IsValid {
			continue
		}
		earned := int64(share.Difficulty * fppsValuePerDiff)
		userPayouts[share.UserID] += earned
	}

	// Convert to payout slice
	payouts := make([]Payout, 0, len(userPayouts))
	for userID, amount := range userPayouts {
		if amount > 0 {
			payouts = append(payouts, Payout{
				UserID:    userID,
				Amount:    amount,
				Timestamp: blockTime,
			})
		}
	}

	return payouts, nil
}

// CalculateShareValue calculates the FPPS value of a single share
func (c *FPPSCalculator) CalculateShareValue(shareDifficulty float64, blockReward int64) int64 {
	if c.networkDifficulty <= 0 {
		return 0
	}

	totalReward := blockReward + c.expectedTxFees
	expectedValuePerDiff := float64(totalReward) / c.networkDifficulty
	feeMultiplier := 1.0 - (c.poolFeePercent / 100.0)
	return int64(shareDifficulty * expectedValuePerDiff * feeMultiplier)
}

// GetPoolFeePercent returns the pool fee percentage
func (c *FPPSCalculator) GetPoolFeePercent() float64 {
	return c.poolFeePercent
}

// SetPoolFeePercent sets the pool fee percentage
func (c *FPPSCalculator) SetPoolFeePercent(fee float64) error {
	if fee < 0 || fee > 100 {
		return fmt.Errorf("pool fee percent must be between 0 and 100, got %.2f", fee)
	}
	c.poolFeePercent = fee
	return nil
}

// SetNetworkDifficulty sets the current network difficulty
func (c *FPPSCalculator) SetNetworkDifficulty(difficulty float64) {
	c.networkDifficulty = difficulty
}

// SetExpectedTxFees sets the expected transaction fees per block
func (c *FPPSCalculator) SetExpectedTxFees(fees int64) {
	c.expectedTxFees = fees
}

// ValidateConfiguration validates the calculator configuration
func (c *FPPSCalculator) ValidateConfiguration() error {
	if c.poolFeePercent < 0 || c.poolFeePercent > 100 {
		return fmt.Errorf("invalid pool fee percent: %.2f", c.poolFeePercent)
	}
	if c.networkDifficulty <= 0 {
		return fmt.Errorf("network difficulty must be set before calculating payouts")
	}
	return nil
}
