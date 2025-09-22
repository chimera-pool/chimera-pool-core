package payouts

import (
	"fmt"
	"sort"
	"time"
)

// Share represents a mining share for payout calculation
type Share struct {
	ID         int64     `json:"id" db:"id"`
	UserID     int64     `json:"user_id" db:"user_id"`
	MinerID    int64     `json:"miner_id" db:"miner_id"`
	Difficulty float64   `json:"difficulty" db:"difficulty"`
	IsValid    bool      `json:"is_valid" db:"is_valid"`
	Timestamp  time.Time `json:"timestamp" db:"timestamp"`
}

// Payout represents a calculated payout for a user
type Payout struct {
	UserID    int64     `json:"user_id" db:"user_id"`
	Amount    int64     `json:"amount" db:"amount"`
	BlockID   int64     `json:"block_id" db:"block_id"`
	Timestamp time.Time `json:"timestamp" db:"timestamp"`
}

// PPLNSCalculator implements Pay Per Last N Shares payout calculation
type PPLNSCalculator struct {
	windowSize     int64   // Total difficulty window for PPLNS calculation
	poolFeePercent float64 // Pool fee percentage (0-100)
}

// NewPPLNSCalculator creates a new PPLNS calculator with validation
func NewPPLNSCalculator(windowSize int64, poolFeePercent float64) (*PPLNSCalculator, error) {
	if windowSize <= 0 {
		return nil, fmt.Errorf("window size must be positive, got %d", windowSize)
	}
	
	if poolFeePercent < 0 || poolFeePercent > 100 {
		return nil, fmt.Errorf("pool fee percent must be between 0 and 100, got %.2f", poolFeePercent)
	}
	
	return &PPLNSCalculator{
		windowSize:     windowSize,
		poolFeePercent: poolFeePercent,
	}, nil
}

// CalculatePayouts calculates PPLNS payouts for a found block
func (calc *PPLNSCalculator) CalculatePayouts(shares []Share, blockReward int64, blockTime time.Time) ([]Payout, error) {
	if blockReward <= 0 {
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
	
	// Apply sliding window - collect shares until we reach windowSize difficulty
	windowShares := calc.applySlidingWindow(validShares)
	
	if len(windowShares) == 0 {
		return []Payout{}, nil
	}
	
	// Calculate total difficulty in window
	totalDifficulty := float64(0)
	for _, share := range windowShares {
		totalDifficulty += share.Difficulty
	}
	
	if totalDifficulty == 0 {
		return []Payout{}, nil
	}
	
	// Calculate net reward after pool fee
	poolFee := int64(float64(blockReward) * calc.poolFeePercent / 100.0)
	netReward := blockReward - poolFee
	
	// Aggregate difficulty by user
	userDifficulties := make(map[int64]float64)
	for _, share := range windowShares {
		userDifficulties[share.UserID] += share.Difficulty
	}
	
	// Calculate payouts proportionally
	payouts := make([]Payout, 0, len(userDifficulties))
	for userID, userDifficulty := range userDifficulties {
		proportion := userDifficulty / totalDifficulty
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

// applySlidingWindow applies the PPLNS sliding window to shares
func (calc *PPLNSCalculator) applySlidingWindow(sortedShares []Share) []Share {
	windowShares := make([]Share, 0, len(sortedShares))
	accumulatedDifficulty := float64(0)
	
	for _, share := range sortedShares {
		remainingWindow := float64(calc.windowSize) - accumulatedDifficulty
		
		if remainingWindow <= 0 {
			break // Window is full
		}
		
		if share.Difficulty <= remainingWindow {
			// Share fits completely in window
			windowShares = append(windowShares, share)
			accumulatedDifficulty += share.Difficulty
		} else {
			// Share partially fits in window - create partial share
			partialShare := share
			partialShare.Difficulty = remainingWindow
			windowShares = append(windowShares, partialShare)
			accumulatedDifficulty += remainingWindow
			break // Window is now full
		}
	}
	
	return windowShares
}

// GetWindowSize returns the configured window size
func (calc *PPLNSCalculator) GetWindowSize() int64 {
	return calc.windowSize
}

// GetPoolFeePercent returns the configured pool fee percentage
func (calc *PPLNSCalculator) GetPoolFeePercent() float64 {
	return calc.poolFeePercent
}

// ValidateConfiguration validates the calculator configuration
func (calc *PPLNSCalculator) ValidateConfiguration() error {
	if calc.windowSize <= 0 {
		return fmt.Errorf("invalid window size: %d", calc.windowSize)
	}
	
	if calc.poolFeePercent < 0 || calc.poolFeePercent > 100 {
		return fmt.Errorf("invalid pool fee percent: %.2f", calc.poolFeePercent)
	}
	
	return nil
}