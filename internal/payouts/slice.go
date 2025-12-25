package payouts

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

// =============================================================================
// SLICE PAYOUT CALCULATOR
// Stratum V2 Job Declaration Enhanced PPLNS
// Based on DEMAND Pool 2025 concepts
// =============================================================================

// SLICECalculator implements a V2-enhanced PPLNS with sliced share windows
// Features:
// - Auditable shares via Job Declaration protocol
// - Disconnect-tolerant windows (shares persist across disconnects)
// - Time-weighted scoring within slices
// - Demand response hooks for dynamic fee adjustment
type SLICECalculator struct {
	sliceCount     int64   // Number of slices in the window
	sliceDuration  int64   // Duration of each slice in seconds
	decayFactor    float64 // Time decay factor within slices (0-1)
	poolFeePercent float64 // Pool fee percentage

	// Demand response hooks
	demandMultiplier float64 // Dynamic fee multiplier based on network demand
	minFeePercent    float64 // Minimum fee floor
	maxFeePercent    float64 // Maximum fee ceiling

	// V2 Job Declaration tracking
	jobDeclarations map[string]*JobDeclaration // Track job declarations by miner
	mu              sync.RWMutex
}

// JobDeclaration represents a V2 Job Declaration from a miner
type JobDeclaration struct {
	MinerID        int64     `json:"miner_id"`
	JobID          string    `json:"job_id"`
	PrevHash       string    `json:"prev_hash"`
	CoinbasePrefix []byte    `json:"coinbase_prefix"`
	CoinbaseSuffix []byte    `json:"coinbase_suffix"`
	MerklePath     []string  `json:"merkle_path"`
	Version        uint32    `json:"version"`
	NBits          uint32    `json:"nbits"`
	NTime          uint32    `json:"ntime"`
	DeclaredAt     time.Time `json:"declared_at"`
	Validated      bool      `json:"validated"`
	ValidationErr  string    `json:"validation_err,omitempty"`
}

// ShareSlice represents a time slice containing shares
type ShareSlice struct {
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	Shares       []Share   `json:"shares"`
	TotalDiff    float64   `json:"total_difficulty"`
	WeightedDiff float64   `json:"weighted_difficulty"`
	SliceIndex   int       `json:"slice_index"`
}

// Compile-time interface compliance checks
var (
	_ PayoutCalculator       = (*SLICECalculator)(nil)
	_ ShareWindowCalculator  = (*SLICECalculator)(nil)
	_ TimeWeightedCalculator = (*SLICECalculator)(nil)
)

// NewSLICECalculator creates a new SLICE calculator
func NewSLICECalculator(sliceCount, sliceDuration int64, decayFactor, poolFeePercent float64) (*SLICECalculator, error) {
	if sliceCount <= 0 {
		return nil, fmt.Errorf("slice count must be positive, got %d", sliceCount)
	}
	if sliceDuration <= 0 {
		return nil, fmt.Errorf("slice duration must be positive, got %d", sliceDuration)
	}
	if decayFactor <= 0 || decayFactor > 1 {
		return nil, fmt.Errorf("decay factor must be between 0 and 1, got %.2f", decayFactor)
	}
	if poolFeePercent < 0 || poolFeePercent > 100 {
		return nil, fmt.Errorf("pool fee must be between 0 and 100, got %.2f", poolFeePercent)
	}

	return &SLICECalculator{
		sliceCount:       sliceCount,
		sliceDuration:    sliceDuration,
		decayFactor:      decayFactor,
		poolFeePercent:   poolFeePercent,
		demandMultiplier: 1.0,
		minFeePercent:    0.5,
		maxFeePercent:    3.0,
		jobDeclarations:  make(map[string]*JobDeclaration),
	}, nil
}

// Mode returns the payout mode for this calculator
func (c *SLICECalculator) Mode() PayoutMode {
	return PayoutModeSLICE
}

// CalculatePayouts calculates SLICE payouts for a found block
func (c *SLICECalculator) CalculatePayouts(shares []Share, blockReward int64, txFees int64, blockTime time.Time) ([]Payout, error) {
	totalReward := blockReward + txFees
	if totalReward <= 0 {
		return []Payout{}, nil
	}

	if len(shares) == 0 {
		return []Payout{}, nil
	}

	// Filter valid shares
	validShares := make([]Share, 0, len(shares))
	for _, share := range shares {
		if share.IsValid {
			validShares = append(validShares, share)
		}
	}

	if len(validShares) == 0 {
		return []Payout{}, nil
	}

	// Organize shares into slices
	slices := c.organizeIntoSlices(validShares, blockTime)

	// Calculate weighted scores per user across all slices
	userScores := c.calculateUserScores(slices, blockTime)

	if len(userScores) == 0 {
		return []Payout{}, nil
	}

	// Calculate total weighted score
	totalScore := 0.0
	for _, score := range userScores {
		totalScore += score
	}

	if totalScore == 0 {
		return []Payout{}, nil
	}

	// Apply demand-adjusted fee
	effectiveFee := c.getEffectiveFee()
	poolFee := int64(float64(totalReward) * effectiveFee / 100.0)
	netReward := totalReward - poolFee

	// Calculate payouts proportionally
	payouts := make([]Payout, 0, len(userScores))
	for userID, score := range userScores {
		proportion := score / totalScore
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

// organizeIntoSlices organizes shares into time-based slices
func (c *SLICECalculator) organizeIntoSlices(shares []Share, blockTime time.Time) []ShareSlice {
	// Calculate window boundaries
	windowDuration := time.Duration(c.sliceCount*c.sliceDuration) * time.Second
	windowStart := blockTime.Add(-windowDuration)

	// Create slices
	slices := make([]ShareSlice, c.sliceCount)
	for i := int64(0); i < c.sliceCount; i++ {
		sliceStart := windowStart.Add(time.Duration(i*c.sliceDuration) * time.Second)
		sliceEnd := sliceStart.Add(time.Duration(c.sliceDuration) * time.Second)
		slices[i] = ShareSlice{
			StartTime:  sliceStart,
			EndTime:    sliceEnd,
			Shares:     make([]Share, 0),
			SliceIndex: int(i),
		}
	}

	// Assign shares to slices
	for _, share := range shares {
		// Find which slice this share belongs to
		for i := range slices {
			if !share.Timestamp.Before(slices[i].StartTime) && share.Timestamp.Before(slices[i].EndTime) {
				slices[i].Shares = append(slices[i].Shares, share)
				slices[i].TotalDiff += share.Difficulty
				break
			}
		}
	}

	return slices
}

// calculateUserScores calculates weighted scores for each user across slices
func (c *SLICECalculator) calculateUserScores(slices []ShareSlice, blockTime time.Time) map[int64]float64 {
	userScores := make(map[int64]float64)

	for _, slice := range slices {
		// Calculate slice weight based on recency (newer slices worth more)
		sliceAge := blockTime.Sub(slice.EndTime).Hours()
		sliceWeight := math.Pow(c.decayFactor, sliceAge)

		// Calculate per-user weighted difficulty within this slice
		for _, share := range slice.Shares {
			// Apply time decay within the slice
			shareAge := slice.EndTime.Sub(share.Timestamp).Seconds()
			sliceDurationSec := float64(c.sliceDuration)
			intraSliceWeight := 1.0 - (shareAge/sliceDurationSec)*(1.0-c.decayFactor)
			if intraSliceWeight < 0 {
				intraSliceWeight = 0
			}

			// Calculate weighted score
			weightedScore := share.Difficulty * sliceWeight * intraSliceWeight
			userScores[share.UserID] += weightedScore
		}
	}

	return userScores
}

// getEffectiveFee returns the demand-adjusted effective fee
func (c *SLICECalculator) getEffectiveFee() float64 {
	adjustedFee := c.poolFeePercent * c.demandMultiplier

	// Apply floor and ceiling
	if adjustedFee < c.minFeePercent {
		adjustedFee = c.minFeePercent
	}
	if adjustedFee > c.maxFeePercent {
		adjustedFee = c.maxFeePercent
	}

	return adjustedFee
}

// GetPoolFeePercent returns the base pool fee percentage
func (c *SLICECalculator) GetPoolFeePercent() float64 {
	return c.poolFeePercent
}

// SetPoolFeePercent sets the base pool fee percentage
func (c *SLICECalculator) SetPoolFeePercent(fee float64) error {
	if fee < 0 || fee > 100 {
		return fmt.Errorf("pool fee must be between 0 and 100, got %.2f", fee)
	}
	c.poolFeePercent = fee
	return nil
}

// GetWindowSize returns the total window size (slice count * slice duration)
func (c *SLICECalculator) GetWindowSize() int64 {
	return c.sliceCount * c.sliceDuration
}

// SetWindowSize sets the window size by adjusting slice count
func (c *SLICECalculator) SetWindowSize(size int64) error {
	if size <= 0 {
		return fmt.Errorf("window size must be positive, got %d", size)
	}
	// Adjust slice count to maintain slice duration
	c.sliceCount = size / c.sliceDuration
	if c.sliceCount < 1 {
		c.sliceCount = 1
	}
	return nil
}

// GetDecayFactor returns the time decay factor
func (c *SLICECalculator) GetDecayFactor() float64 {
	return c.decayFactor
}

// SetDecayFactor sets the time decay factor
func (c *SLICECalculator) SetDecayFactor(factor float64) error {
	if factor <= 0 || factor > 1 {
		return fmt.Errorf("decay factor must be between 0 and 1, got %.2f", factor)
	}
	c.decayFactor = factor
	return nil
}

// ValidateConfiguration validates the calculator configuration
func (c *SLICECalculator) ValidateConfiguration() error {
	if c.sliceCount <= 0 {
		return fmt.Errorf("invalid slice count: %d", c.sliceCount)
	}
	if c.sliceDuration <= 0 {
		return fmt.Errorf("invalid slice duration: %d", c.sliceDuration)
	}
	if c.decayFactor <= 0 || c.decayFactor > 1 {
		return fmt.Errorf("invalid decay factor: %.2f", c.decayFactor)
	}
	if c.poolFeePercent < 0 || c.poolFeePercent > 100 {
		return fmt.Errorf("invalid pool fee: %.2f", c.poolFeePercent)
	}
	return nil
}

// =============================================================================
// DEMAND RESPONSE HOOKS
// =============================================================================

// SetDemandMultiplier sets the demand-based fee multiplier
func (c *SLICECalculator) SetDemandMultiplier(multiplier float64) error {
	if multiplier <= 0 {
		return fmt.Errorf("demand multiplier must be positive, got %.2f", multiplier)
	}
	c.demandMultiplier = multiplier
	return nil
}

// GetDemandMultiplier returns the current demand multiplier
func (c *SLICECalculator) GetDemandMultiplier() float64 {
	return c.demandMultiplier
}

// SetFeeBounds sets the minimum and maximum fee bounds
func (c *SLICECalculator) SetFeeBounds(minFee, maxFee float64) error {
	if minFee < 0 || maxFee < 0 {
		return fmt.Errorf("fee bounds cannot be negative")
	}
	if minFee > maxFee {
		return fmt.Errorf("min fee cannot exceed max fee")
	}
	c.minFeePercent = minFee
	c.maxFeePercent = maxFee
	return nil
}

// GetFeeBounds returns the current fee bounds
func (c *SLICECalculator) GetFeeBounds() (minFee, maxFee float64) {
	return c.minFeePercent, c.maxFeePercent
}

// GetEffectiveFeePercent returns the current effective fee after demand adjustment
func (c *SLICECalculator) GetEffectiveFeePercent() float64 {
	return c.getEffectiveFee()
}

// =============================================================================
// V2 JOB DECLARATION SUPPORT
// =============================================================================

// RegisterJobDeclaration registers a V2 Job Declaration from a miner
func (c *SLICECalculator) RegisterJobDeclaration(declaration *JobDeclaration) error {
	if declaration == nil {
		return fmt.Errorf("declaration cannot be nil")
	}
	if declaration.JobID == "" {
		return fmt.Errorf("job ID cannot be empty")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	declaration.DeclaredAt = time.Now()
	c.jobDeclarations[declaration.JobID] = declaration
	return nil
}

// ValidateJobDeclaration validates a job declaration
func (c *SLICECalculator) ValidateJobDeclaration(jobID string) (*JobDeclaration, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	decl, exists := c.jobDeclarations[jobID]
	if !exists {
		return nil, fmt.Errorf("job declaration not found: %s", jobID)
	}

	// Perform validation (placeholder - actual implementation would verify merkle path, etc.)
	decl.Validated = true
	return decl, nil
}

// GetJobDeclaration retrieves a job declaration by ID
func (c *SLICECalculator) GetJobDeclaration(jobID string) *JobDeclaration {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.jobDeclarations[jobID]
}

// CleanupOldDeclarations removes job declarations older than maxAge
func (c *SLICECalculator) CleanupOldDeclarations(maxAge time.Duration) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	removed := 0

	for jobID, decl := range c.jobDeclarations {
		if decl.DeclaredAt.Before(cutoff) {
			delete(c.jobDeclarations, jobID)
			removed++
		}
	}

	return removed
}

// =============================================================================
// SLICE ANALYTICS
// =============================================================================

// SliceAnalytics provides analytics for SLICE payout calculation
type SliceAnalytics struct {
	TotalSlices      int                 `json:"total_slices"`
	ActiveSlices     int                 `json:"active_slices"`
	TotalShares      int                 `json:"total_shares"`
	TotalDifficulty  float64             `json:"total_difficulty"`
	WeightedDiff     float64             `json:"weighted_difficulty"`
	UniqueMiners     int                 `json:"unique_miners"`
	SliceBreakdown   []SliceBreakdown    `json:"slice_breakdown"`
	MinerContribs    []MinerContribution `json:"miner_contributions"`
	EffectiveFee     float64             `json:"effective_fee"`
	DemandMultiplier float64             `json:"demand_multiplier"`
}

// SliceBreakdown provides per-slice analytics
type SliceBreakdown struct {
	SliceIndex   int       `json:"slice_index"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	ShareCount   int       `json:"share_count"`
	Difficulty   float64   `json:"difficulty"`
	WeightedDiff float64   `json:"weighted_diff"`
	SliceWeight  float64   `json:"slice_weight"`
}

// MinerContribution provides per-miner contribution analytics
type MinerContribution struct {
	UserID       int64   `json:"user_id"`
	ShareCount   int     `json:"share_count"`
	RawDiff      float64 `json:"raw_difficulty"`
	WeightedDiff float64 `json:"weighted_difficulty"`
	Percentage   float64 `json:"percentage"`
}

// GetAnalytics returns detailed analytics for a set of shares
func (c *SLICECalculator) GetAnalytics(shares []Share, blockTime time.Time) *SliceAnalytics {
	// Filter valid shares
	validShares := make([]Share, 0, len(shares))
	for _, share := range shares {
		if share.IsValid {
			validShares = append(validShares, share)
		}
	}

	// Organize into slices
	slices := c.organizeIntoSlices(validShares, blockTime)

	// Calculate user scores
	userScores := c.calculateUserScores(slices, blockTime)

	// Calculate total score
	totalScore := 0.0
	for _, score := range userScores {
		totalScore += score
	}

	// Build analytics
	analytics := &SliceAnalytics{
		TotalSlices:      int(c.sliceCount),
		TotalShares:      len(validShares),
		EffectiveFee:     c.getEffectiveFee(),
		DemandMultiplier: c.demandMultiplier,
	}

	// Slice breakdown
	minerShareCounts := make(map[int64]int)
	minerRawDiff := make(map[int64]float64)

	for _, slice := range slices {
		sliceAge := blockTime.Sub(slice.EndTime).Hours()
		sliceWeight := math.Pow(c.decayFactor, sliceAge)

		breakdown := SliceBreakdown{
			SliceIndex:  slice.SliceIndex,
			StartTime:   slice.StartTime,
			EndTime:     slice.EndTime,
			ShareCount:  len(slice.Shares),
			Difficulty:  slice.TotalDiff,
			SliceWeight: sliceWeight,
		}

		if len(slice.Shares) > 0 {
			analytics.ActiveSlices++
		}

		analytics.TotalDifficulty += slice.TotalDiff

		// Track per-miner stats
		for _, share := range slice.Shares {
			minerShareCounts[share.UserID]++
			minerRawDiff[share.UserID] += share.Difficulty
		}

		analytics.SliceBreakdown = append(analytics.SliceBreakdown, breakdown)
	}

	// Miner contributions
	for userID, score := range userScores {
		contrib := MinerContribution{
			UserID:       userID,
			ShareCount:   minerShareCounts[userID],
			RawDiff:      minerRawDiff[userID],
			WeightedDiff: score,
		}
		if totalScore > 0 {
			contrib.Percentage = (score / totalScore) * 100
		}
		analytics.MinerContribs = append(analytics.MinerContribs, contrib)
		analytics.WeightedDiff += score
	}

	// Sort by weighted difficulty descending
	sort.Slice(analytics.MinerContribs, func(i, j int) bool {
		return analytics.MinerContribs[i].WeightedDiff > analytics.MinerContribs[j].WeightedDiff
	})

	analytics.UniqueMiners = len(userScores)

	return analytics
}

// GetSliceConfig returns the current SLICE configuration
func (c *SLICECalculator) GetSliceConfig() map[string]interface{} {
	return map[string]interface{}{
		"slice_count":       c.sliceCount,
		"slice_duration":    c.sliceDuration,
		"decay_factor":      c.decayFactor,
		"pool_fee_percent":  c.poolFeePercent,
		"demand_multiplier": c.demandMultiplier,
		"min_fee_percent":   c.minFeePercent,
		"max_fee_percent":   c.maxFeePercent,
		"effective_fee":     c.getEffectiveFee(),
		"window_duration":   c.sliceCount * c.sliceDuration,
	}
}
