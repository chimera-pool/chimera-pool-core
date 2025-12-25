package payouts

import (
	"fmt"
	"math"
	"sort"
	"time"
)

// =============================================================================
// SCORE (Time-Weighted PPLNS) CALCULATOR
// Shares are weighted by time - older shares worth less
// Discourages pool hopping by penalizing late arrivals
// =============================================================================

// SCORECalculator implements time-weighted scoring payout calculation
type SCORECalculator struct {
	windowSize     int64   // Total difficulty window
	poolFeePercent float64 // Pool fee percentage (0-100)
	decayFactor    float64 // Decay factor per hour (0-1)
}

// NewSCORECalculator creates a new SCORE calculator
func NewSCORECalculator(windowSize int64, poolFeePercent float64, decayFactor float64) (*SCORECalculator, error) {
	if windowSize <= 0 {
		return nil, fmt.Errorf("window size must be positive, got %d", windowSize)
	}

	if poolFeePercent < 0 || poolFeePercent > 100 {
		return nil, fmt.Errorf("pool fee percent must be between 0 and 100, got %.2f", poolFeePercent)
	}

	if decayFactor <= 0 || decayFactor > 1 {
		return nil, fmt.Errorf("decay factor must be between 0 and 1, got %.2f", decayFactor)
	}

	return &SCORECalculator{
		windowSize:     windowSize,
		poolFeePercent: poolFeePercent,
		decayFactor:    decayFactor,
	}, nil
}

// Mode returns the payout mode
func (c *SCORECalculator) Mode() PayoutMode {
	return PayoutModeSCORE
}

// CalculatePayouts calculates SCORE payouts for a found block
// Each share is weighted by: difficulty * decayFactor^(hoursOld)
func (c *SCORECalculator) CalculatePayouts(shares []Share, blockReward int64, txFees int64, blockTime time.Time) ([]Payout, error) {
	totalReward := blockReward + txFees
	if totalReward <= 0 {
		return []Payout{}, nil
	}

	if len(shares) == 0 {
		return []Payout{}, nil
	}

	// Filter valid shares and sort by timestamp (newest first)
	validShares := make([]Share, 0, len(shares))
	for _, share := range shares {
		if share.IsValid {
			validShares = append(validShares, share)
		}
	}

	if len(validShares) == 0 {
		return []Payout{}, nil
	}

	// Sort shares by timestamp descending (newest first)
	sort.Slice(validShares, func(i, j int) bool {
		return validShares[i].Timestamp.After(validShares[j].Timestamp)
	})

	// Apply sliding window and calculate time-weighted scores
	windowShares, scores := c.applyTimeWeightedWindow(validShares, blockTime)

	if len(windowShares) == 0 {
		return []Payout{}, nil
	}

	// Calculate total score
	totalScore := float64(0)
	for _, score := range scores {
		totalScore += score
	}

	if totalScore == 0 {
		return []Payout{}, nil
	}

	// Calculate net reward after pool fee
	poolFee := int64(float64(totalReward) * c.poolFeePercent / 100.0)
	netReward := totalReward - poolFee

	// Aggregate scores by user
	userScores := make(map[int64]float64)
	for i, share := range windowShares {
		userScores[share.UserID] += scores[i]
	}

	// Calculate payouts proportionally based on score
	payouts := make([]Payout, 0, len(userScores))
	for userID, userScore := range userScores {
		proportion := userScore / totalScore
		amount := int64(float64(netReward) * proportion)

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

// applyTimeWeightedWindow applies the sliding window with time weighting
func (c *SCORECalculator) applyTimeWeightedWindow(sortedShares []Share, blockTime time.Time) ([]Share, []float64) {
	windowShares := make([]Share, 0, len(sortedShares))
	scores := make([]float64, 0, len(sortedShares))
	accumulatedDifficulty := float64(0)

	for _, share := range sortedShares {
		remainingWindow := float64(c.windowSize) - accumulatedDifficulty

		if remainingWindow <= 0 {
			break // Window is full
		}

		// Calculate time-based weight
		// Weight = decayFactor^(hours since share)
		hoursOld := blockTime.Sub(share.Timestamp).Hours()
		if hoursOld < 0 {
			hoursOld = 0 // Share submitted after block (edge case)
		}
		timeWeight := math.Pow(c.decayFactor, hoursOld)

		// Clamp minimum weight to prevent shares becoming worthless
		if timeWeight < 0.01 {
			timeWeight = 0.01
		}

		effectiveDifficulty := share.Difficulty
		if effectiveDifficulty > remainingWindow {
			effectiveDifficulty = remainingWindow // Partial share at window edge
		}

		// Score = difficulty * timeWeight
		score := effectiveDifficulty * timeWeight

		windowShares = append(windowShares, share)
		scores = append(scores, score)
		accumulatedDifficulty += effectiveDifficulty

		if accumulatedDifficulty >= float64(c.windowSize) {
			break
		}
	}

	return windowShares, scores
}

// GetWindowSize returns the configured window size
func (c *SCORECalculator) GetWindowSize() int64 {
	return c.windowSize
}

// SetWindowSize sets the window size
func (c *SCORECalculator) SetWindowSize(size int64) error {
	if size <= 0 {
		return fmt.Errorf("window size must be positive, got %d", size)
	}
	c.windowSize = size
	return nil
}

// GetPoolFeePercent returns the configured pool fee percentage
func (c *SCORECalculator) GetPoolFeePercent() float64 {
	return c.poolFeePercent
}

// SetPoolFeePercent sets the pool fee percentage
func (c *SCORECalculator) SetPoolFeePercent(fee float64) error {
	if fee < 0 || fee > 100 {
		return fmt.Errorf("pool fee percent must be between 0 and 100, got %.2f", fee)
	}
	c.poolFeePercent = fee
	return nil
}

// GetDecayFactor returns the time decay factor
func (c *SCORECalculator) GetDecayFactor() float64 {
	return c.decayFactor
}

// SetDecayFactor sets the time decay factor
func (c *SCORECalculator) SetDecayFactor(factor float64) error {
	if factor <= 0 || factor > 1 {
		return fmt.Errorf("decay factor must be between 0 and 1, got %.2f", factor)
	}
	c.decayFactor = factor
	return nil
}

// ValidateConfiguration validates the calculator configuration
func (c *SCORECalculator) ValidateConfiguration() error {
	if c.windowSize <= 0 {
		return fmt.Errorf("invalid window size: %d", c.windowSize)
	}

	if c.poolFeePercent < 0 || c.poolFeePercent > 100 {
		return fmt.Errorf("invalid pool fee percent: %.2f", c.poolFeePercent)
	}

	if c.decayFactor <= 0 || c.decayFactor > 1 {
		return fmt.Errorf("invalid decay factor: %.2f", c.decayFactor)
	}

	return nil
}

// =============================================================================
// SOLO MINING CALCULATOR
// Miner keeps entire block reward minus pool fee
// =============================================================================

// SOLOCalculator implements solo mining payout calculation
type SOLOCalculator struct {
	poolFeePercent float64 // Pool fee percentage (0-100)
}

// NewSOLOCalculator creates a new SOLO calculator
func NewSOLOCalculator(poolFeePercent float64) (*SOLOCalculator, error) {
	if poolFeePercent < 0 || poolFeePercent > 100 {
		return nil, fmt.Errorf("pool fee percent must be between 0 and 100, got %.2f", poolFeePercent)
	}

	return &SOLOCalculator{
		poolFeePercent: poolFeePercent,
	}, nil
}

// Mode returns the payout mode
func (c *SOLOCalculator) Mode() PayoutMode {
	return PayoutModeSOLO
}

// CalculatePayouts calculates SOLO payout - entire reward goes to block finder
func (c *SOLOCalculator) CalculatePayouts(shares []Share, blockReward int64, txFees int64, blockTime time.Time) ([]Payout, error) {
	totalReward := blockReward + txFees
	if totalReward <= 0 {
		return []Payout{}, nil
	}

	if len(shares) == 0 {
		return []Payout{}, nil
	}

	// Find the share that found the block (last valid share before block time)
	var finderShare *Share
	for i := len(shares) - 1; i >= 0; i-- {
		if shares[i].IsValid && !shares[i].Timestamp.After(blockTime) {
			finderShare = &shares[i]
			break
		}
	}

	if finderShare == nil {
		// Fallback: use first valid share's user
		for _, share := range shares {
			if share.IsValid {
				finderShare = &share
				break
			}
		}
	}

	if finderShare == nil {
		return []Payout{}, nil
	}

	// Calculate net reward after pool fee
	poolFee := int64(float64(totalReward) * c.poolFeePercent / 100.0)
	netReward := totalReward - poolFee

	return []Payout{{
		UserID:    finderShare.UserID,
		Amount:    netReward,
		Timestamp: blockTime,
	}}, nil
}

// GetPoolFeePercent returns the pool fee percentage
func (c *SOLOCalculator) GetPoolFeePercent() float64 {
	return c.poolFeePercent
}

// SetPoolFeePercent sets the pool fee percentage
func (c *SOLOCalculator) SetPoolFeePercent(fee float64) error {
	if fee < 0 || fee > 100 {
		return fmt.Errorf("pool fee percent must be between 0 and 100, got %.2f", fee)
	}
	c.poolFeePercent = fee
	return nil
}

// ValidateConfiguration validates the calculator configuration
func (c *SOLOCalculator) ValidateConfiguration() error {
	if c.poolFeePercent < 0 || c.poolFeePercent > 100 {
		return fmt.Errorf("invalid pool fee percent: %.2f", c.poolFeePercent)
	}
	return nil
}
