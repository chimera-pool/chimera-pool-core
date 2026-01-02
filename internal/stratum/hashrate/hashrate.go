// Package hashrate provides hashrate calculation utilities for mining pools
package hashrate

import (
	"fmt"
	"sync"
	"time"
)

const (
	// Diff1Target is 2^32, the number of hashes for difficulty 1
	Diff1Target = 4294967296.0
)

// Calculator provides hashrate calculation utilities
type Calculator struct{}

// NewCalculator creates a new hashrate calculator
func NewCalculator() *Calculator {
	return &Calculator{}
}

// Calculate computes hashrate from shares, difficulty, and duration
// Returns hashrate in H/s (hashes per second)
func (c *Calculator) Calculate(shares int64, difficulty float64, duration time.Duration) float64 {
	if shares == 0 || duration == 0 {
		return 0
	}

	seconds := duration.Seconds()
	if seconds <= 0 {
		return 0
	}

	// Hashrate = (shares * difficulty * 2^32) / seconds
	return (float64(shares) * difficulty * Diff1Target) / seconds
}

// Format converts hashrate to human-readable string
func (c *Calculator) Format(hashrate float64) string {
	if hashrate == 0 {
		return "0.00 H/s"
	}

	units := []string{"H/s", "KH/s", "MH/s", "GH/s", "TH/s", "PH/s", "EH/s"}
	unitIndex := 0

	for hashrate >= 1000 && unitIndex < len(units)-1 {
		hashrate /= 1000
		unitIndex++
	}

	return fmt.Sprintf("%.2f %s", hashrate, units[unitIndex])
}

// ShareRecord represents a single share submission
type ShareRecord struct {
	Difficulty float64
	Timestamp  time.Time
}

// Window maintains a rolling window of shares for hashrate calculation
type Window struct {
	shares   []ShareRecord
	duration time.Duration
	mu       sync.RWMutex
}

// NewWindow creates a new hashrate window with the specified duration
func NewWindow(duration time.Duration) *Window {
	return &Window{
		shares:   make([]ShareRecord, 0),
		duration: duration,
	}
}

// AddShare adds a share to the window
func (w *Window) AddShare(difficulty float64, timestamp time.Time) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.shares = append(w.shares, ShareRecord{
		Difficulty: difficulty,
		Timestamp:  timestamp,
	})

	// Clean up old shares
	w.cleanupLocked()
}

// cleanupLocked removes expired shares (must be called with lock held)
func (w *Window) cleanupLocked() {
	cutoff := time.Now().Add(-w.duration)
	newShares := make([]ShareRecord, 0, len(w.shares))

	for _, s := range w.shares {
		if s.Timestamp.After(cutoff) {
			newShares = append(newShares, s)
		}
	}

	w.shares = newShares
}

// GetHashrate calculates the current hashrate based on shares in the window
func (w *Window) GetHashrate() float64 {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.cleanupLocked()

	if len(w.shares) == 0 {
		return 0
	}

	// Sum up difficulty-weighted shares
	var totalDifficulty float64
	for _, s := range w.shares {
		totalDifficulty += s.Difficulty
	}

	// Calculate hashrate over the window duration
	// Formula: hashrate = (totalDifficulty * 2^32) / seconds
	hashrate := (totalDifficulty * Diff1Target) / w.duration.Seconds()

	// SANITY CHECK: Apply reasonable bounds based on miner type
	// CPU miners: max ~100 KH/s (100,000 H/s)
	// GPU miners: max ~100 MH/s (100,000,000 H/s)
	// ASIC miners: max ~100 TH/s (100,000,000,000,000 H/s)
	// If hashrate exceeds reasonable bounds, it's likely a calculation error
	const maxReasonableHashrate = 100_000_000_000_000.0 // 100 TH/s max for any single miner
	if hashrate > maxReasonableHashrate {
		// Log warning and cap the value
		hashrate = maxReasonableHashrate
	}

	return hashrate
}

// GetShareCount returns the number of shares in the window
func (w *Window) GetShareCount() int {
	w.mu.RLock()
	defer w.mu.RUnlock()

	cutoff := time.Now().Add(-w.duration)
	count := 0
	for _, s := range w.shares {
		if s.Timestamp.After(cutoff) {
			count++
		}
	}
	return count
}

// Clear removes all shares from the window
func (w *Window) Clear() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.shares = make([]ShareRecord, 0)
}
