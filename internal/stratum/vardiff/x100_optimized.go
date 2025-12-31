// Package vardiff implements variable difficulty management for stratum mining
// This file contains X100-optimized vardiff configuration and algorithms
package vardiff

import (
	"time"
)

// X100OptimizedConfig returns vardiff configuration optimized for BlockDAG X100 ASICs
// on Scrypt (Litecoin) mining
//
// X100 Specs:
// - BlockDAG Scrpy-variant: ~70 TH/s
// - Litecoin Scrypt: ~13-15 TH/s (different algorithm)
//
// Optimal difficulty calculation:
// - Target hashrate: 15 TH/s = 15e12 H/s
// - Target share time: 10 seconds
// - Difficulty = Hashrate * ShareTime / 2^32
// - Difficulty = 15e12 * 10 / 4.295e9 = ~35,000
func X100OptimizedConfig() Config {
	return Config{
		// Target 10 seconds between shares for good granularity
		TargetShareTime: 10 * time.Second,

		// Retarget every 3 minutes for maximum stability
		// Longer interval = more data points = smoother adjustments
		RetargetInterval: 3 * time.Minute,

		// ±25% variance allowed - balanced between responsiveness and stability
		VariancePercent: 25,

		// Minimum difficulty - lowered to support CPU miners (~10 KH/s)
		// CPU at 10 KH/s needs difficulty ~0.002 for 10s shares
		MinDifficulty: 0.001,

		// Maximum difficulty - supports up to 500 TH/s ASICs
		MaxDifficulty: 10000000,

		// Starting difficulty optimized for X100 on Scrypt (~15 TH/s)
		// D = 15e12 * 10 / 4.295e9 ≈ 35,000
		// Note: handleSubscribe now overrides this based on detected miner type
		InitialDifficulty: 35000,

		// Consider last 30 shares for smoother averaging
		// At 10s/share, this is 5 minutes of data
		ShareWindow: 30,
	}
}

// HighHashrateASICConfig returns vardiff configuration for high-hashrate ASICs
// This is more aggressive than X100OptimizedConfig for miners >50 TH/s
func HighHashrateASICConfig() Config {
	return Config{
		TargetShareTime:   15 * time.Second, // Longer share time for high hashrate
		RetargetInterval:  3 * time.Minute,  // Less frequent adjustments
		VariancePercent:   25,               // Moderate variance
		MinDifficulty:     10000,            // Higher minimum
		MaxDifficulty:     100000000,        // Supports up to 5 PH/s
		InitialDifficulty: 100000,           // Start high
		ShareWindow:       30,               // Large window for stability
	}
}

// LowLatencyConfig returns vardiff configuration optimized for low latency
// Use when network latency is a concern
func LowLatencyConfig() Config {
	return Config{
		TargetShareTime:   5 * time.Second, // Faster share submission
		RetargetInterval:  1 * time.Minute, // Responsive adjustments
		VariancePercent:   40,              // More tolerance
		MinDifficulty:     100,             // Allow lower diff
		MaxDifficulty:     1000000,         // Cap for latency
		InitialDifficulty: 5000,            // Start moderate
		ShareWindow:       10,              // Smaller window
	}
}

// CalculateOptimalDifficulty calculates the optimal difficulty for a given hashrate
// hashrate: in H/s (e.g., 15e12 for 15 TH/s)
// targetShareTime: desired time between shares in seconds
func CalculateOptimalDifficulty(hashrate float64, targetShareTime float64) float64 {
	// Difficulty = Hashrate * ShareTime / 2^32
	return hashrate * targetShareTime / Diff1Target
}

// CalculateExpectedHashrate calculates expected hashrate from difficulty and share time
// difficulty: stratum difficulty
// shareTime: average time between shares in seconds
func CalculateExpectedHashrate(difficulty float64, shareTime float64) float64 {
	if shareTime <= 0 {
		return 0
	}
	// Hashrate = Difficulty * 2^32 / ShareTime
	return difficulty * Diff1Target / shareTime
}

// Diff1Target is 2^32, the number of hashes for difficulty 1
const Diff1Target = 4294967296.0
