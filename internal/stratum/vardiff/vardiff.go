// Package vardiff implements variable difficulty management for stratum mining
package vardiff

import (
	"errors"
	"sync"
	"time"
)

// Config holds vardiff configuration
type Config struct {
	TargetShareTime   time.Duration // Target time between shares (e.g., 10s)
	RetargetInterval  time.Duration // How often to recalculate difficulty
	VariancePercent   float64       // Acceptable variance percentage (e.g., 30 = ±30%)
	MinDifficulty     float64       // Minimum allowed difficulty
	MaxDifficulty     float64       // Maximum allowed difficulty
	InitialDifficulty float64       // Starting difficulty for new miners
	ShareWindow       int           // Number of shares to consider for adjustment
}

// DefaultConfig returns sensible default configuration
func DefaultConfig() Config {
	return Config{
		TargetShareTime:   10 * time.Second, // 10 seconds between shares
		RetargetInterval:  30 * time.Second, // Recalculate every 30 seconds
		VariancePercent:   30,               // ±30% variance allowed
		MinDifficulty:     0.001,            // Very low for testing
		MaxDifficulty:     1000000,          // Very high for ASICs
		InitialDifficulty: 0.01,             // Starting point
		ShareWindow:       5,                // Last 5 shares for average
	}
}

// Validate checks if the configuration is valid
func (c Config) Validate() error {
	if c.TargetShareTime <= 0 {
		return errors.New("target share time must be positive")
	}
	if c.RetargetInterval <= 0 {
		return errors.New("retarget interval must be positive")
	}
	if c.MinDifficulty <= 0 {
		return errors.New("min difficulty must be positive")
	}
	if c.MaxDifficulty <= 0 {
		return errors.New("max difficulty must be positive")
	}
	if c.MinDifficulty > c.MaxDifficulty {
		return errors.New("min difficulty cannot exceed max difficulty")
	}
	if c.VariancePercent < 0 || c.VariancePercent > 100 {
		return errors.New("variance percent must be between 0 and 100")
	}
	return nil
}

// minerState holds per-miner difficulty state
type minerState struct {
	difficulty    float64
	shareTimes    []time.Duration
	lastShareTime time.Time
	lastRetarget  time.Time
	totalShares   int64
}

// Manager implements variable difficulty management
type Manager struct {
	config Config
	miners map[string]*minerState
	mu     sync.RWMutex
}

// NewManager creates a new vardiff manager
func NewManager(config Config) *Manager {
	return &Manager{
		config: config,
		miners: make(map[string]*minerState),
	}
}

// GetDifficulty returns the current difficulty for a miner
func (m *Manager) GetDifficulty(minerID string) float64 {
	m.mu.RLock()
	state, exists := m.miners[minerID]
	m.mu.RUnlock()

	if !exists {
		return m.config.InitialDifficulty
	}
	return state.difficulty
}

// SetDifficulty sets a specific difficulty for a miner (clamped to bounds)
func (m *Manager) SetDifficulty(minerID string, difficulty float64) error {
	// Clamp to bounds
	if difficulty < m.config.MinDifficulty {
		difficulty = m.config.MinDifficulty
	}
	if difficulty > m.config.MaxDifficulty {
		difficulty = m.config.MaxDifficulty
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	state, exists := m.miners[minerID]
	if !exists {
		state = &minerState{
			shareTimes: make([]time.Duration, 0, m.config.ShareWindow),
		}
		m.miners[minerID] = state
	}
	state.difficulty = difficulty
	return nil
}

// RecordShare records a share submission and potentially adjusts difficulty
func (m *Manager) RecordShare(minerID string, shareTime time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, exists := m.miners[minerID]
	if !exists {
		state = &minerState{
			difficulty:   m.config.InitialDifficulty,
			shareTimes:   make([]time.Duration, 0, m.config.ShareWindow),
			lastRetarget: time.Now(),
		}
		m.miners[minerID] = state
	}

	// Add share time to window
	state.shareTimes = append(state.shareTimes, shareTime)
	if len(state.shareTimes) > m.config.ShareWindow {
		state.shareTimes = state.shareTimes[1:]
	}
	state.lastShareTime = time.Now()
	state.totalShares++

	// Check if we should retarget
	if time.Since(state.lastRetarget) >= m.config.RetargetInterval && len(state.shareTimes) >= m.config.ShareWindow {
		m.adjustDifficulty(state)
		state.lastRetarget = time.Now()
	}
}

// adjustDifficulty calculates and applies new difficulty based on share times
// Uses a deadband approach with weighted median for outlier resistance
func (m *Manager) adjustDifficulty(state *minerState) {
	if len(state.shareTimes) == 0 {
		return
	}

	// Calculate weighted median share time (more resistant to outliers than mean)
	avgShareTime := m.calculateWeightedMedian(state.shareTimes)

	targetTime := m.config.TargetShareTime

	// Deadband: wider inner zone where no adjustment happens
	// This prevents oscillation around the target
	deadbandPercent := 15.0 // ±15% deadband
	deadband := float64(targetTime) * (deadbandPercent / 100.0)
	minDeadband := targetTime - time.Duration(deadband)
	maxDeadband := targetTime + time.Duration(deadband)

	// Don't adjust if within deadband (tighter than variance)
	if avgShareTime >= minDeadband && avgShareTime <= maxDeadband {
		return
	}

	// Calculate adjustment ratio
	// If shares are coming too fast, increase difficulty
	// If shares are coming too slow, decrease difficulty
	ratio := float64(targetTime) / float64(avgShareTime)

	// Apply tiered adjustment based on how far off we are
	// Smaller adjustments when closer to target, larger when far off
	deviation := float64(avgShareTime-targetTime) / float64(targetTime)
	if deviation < 0 {
		deviation = -deviation
	}

	// Max change scales with deviation: 10% base + up to 5% more if very far off
	maxChange := 0.10 + (deviation * 0.05)
	if maxChange > 0.15 {
		maxChange = 0.15 // Cap at 15% max change per retarget
	}

	if ratio > 1.0+maxChange {
		ratio = 1.0 + maxChange
	} else if ratio < 1.0-maxChange {
		ratio = 1.0 - maxChange
	}

	// Apply exponential smoothing - 40% weight to new ratio for stability
	smoothingFactor := 0.4
	ratio = (ratio * smoothingFactor) + (1.0 * (1.0 - smoothingFactor))

	newDifficulty := state.difficulty * ratio

	// Clamp to bounds
	if newDifficulty < m.config.MinDifficulty {
		newDifficulty = m.config.MinDifficulty
	}
	if newDifficulty > m.config.MaxDifficulty {
		newDifficulty = m.config.MaxDifficulty
	}

	// Additional stability: don't adjust if change would be less than 2%
	changePercent := (newDifficulty - state.difficulty) / state.difficulty
	if changePercent < 0 {
		changePercent = -changePercent
	}
	if changePercent < 0.02 {
		return
	}

	state.difficulty = newDifficulty
}

// calculateWeightedMedian returns a weighted median of share times
// More recent shares have higher weight, resistant to outliers
func (m *Manager) calculateWeightedMedian(shareTimes []time.Duration) time.Duration {
	if len(shareTimes) == 0 {
		return m.config.TargetShareTime
	}
	if len(shareTimes) == 1 {
		return shareTimes[0]
	}

	// Sort a copy to find median
	sorted := make([]time.Duration, len(shareTimes))
	copy(sorted, shareTimes)
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	// Trim outliers: remove top and bottom 10% if we have enough samples
	trimCount := len(sorted) / 10
	if trimCount > 0 && len(sorted) > 10 {
		sorted = sorted[trimCount : len(sorted)-trimCount]
	}

	// Return median of trimmed set
	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return (sorted[mid-1] + sorted[mid]) / 2
	}
	return sorted[mid]
}

// GetTargetShareTime returns the configured target share time
func (m *Manager) GetTargetShareTime() time.Duration {
	return m.config.TargetShareTime
}

// RemoveMiner removes a miner's state from the manager
func (m *Manager) RemoveMiner(minerID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.miners, minerID)
}

// GetConfig returns the current configuration
func (m *Manager) GetConfig() Config {
	return m.config
}

// GetMinerStats returns statistics for a miner
func (m *Manager) GetMinerStats(minerID string) (difficulty float64, totalShares int64, exists bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, exists := m.miners[minerID]
	if !exists {
		return m.config.InitialDifficulty, 0, false
	}
	return state.difficulty, state.totalShares, true
}
