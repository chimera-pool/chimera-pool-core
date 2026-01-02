package api

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// =============================================================================
// RATE LIMITER - Security Enhancement for Auth Endpoints
// Prevents brute force attacks on login/register endpoints
// =============================================================================

// RateLimiterConfig configures the rate limiter
type RateLimiterConfig struct {
	// Rate limiting
	MaxAttempts   int           // Max attempts per window (default: 5)
	WindowSize    time.Duration // Time window (default: 15 minutes)
	BlockDuration time.Duration // Block duration after max attempts (default: 30 minutes)

	// Cleanup
	CleanupInterval time.Duration // Interval to clean expired entries (default: 5 minutes)
}

// DefaultRateLimiterConfig returns secure defaults
func DefaultRateLimiterConfig() RateLimiterConfig {
	return RateLimiterConfig{
		MaxAttempts:     30,              // Increased from 5 - allows burst during launches
		WindowSize:      5 * time.Minute, // Reduced from 15 - faster reset
		BlockDuration:   5 * time.Minute, // Reduced from 30 - shorter penalty
		CleanupInterval: 1 * time.Minute,
	}
}

// AuthRateLimiterConfig returns config for auth endpoints
// Balanced for security while allowing community onboarding bursts
func AuthRateLimiterConfig() RateLimiterConfig {
	return RateLimiterConfig{
		MaxAttempts:     20,              // Increased from 5 - allows multiple registrations
		WindowSize:      5 * time.Minute, // Reduced from 15 - faster reset
		BlockDuration:   5 * time.Minute, // Reduced from 30 - shorter penalty
		CleanupInterval: 1 * time.Minute,
	}
}

// ipRecord tracks attempts for an IP address
type ipRecord struct {
	Attempts  int
	FirstSeen time.Time
	BlockedAt time.Time
	IsBlocked bool
}

// RateLimiter implements IP-based rate limiting
type RateLimiter struct {
	config  RateLimiterConfig
	records map[string]*ipRecord
	mu      sync.RWMutex
	stopCh  chan struct{}
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config RateLimiterConfig) *RateLimiter {
	rl := &RateLimiter{
		config:  config,
		records: make(map[string]*ipRecord),
		stopCh:  make(chan struct{}),
	}

	// Start cleanup goroutine
	go rl.cleanupLoop()

	return rl
}

// Allow checks if the IP is allowed to make a request
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	record, exists := rl.records[ip]

	if !exists {
		// First request from this IP
		rl.records[ip] = &ipRecord{
			Attempts:  1,
			FirstSeen: now,
		}
		return true
	}

	// Check if blocked
	if record.IsBlocked {
		if now.Sub(record.BlockedAt) > rl.config.BlockDuration {
			// Unblock
			record.IsBlocked = false
			record.Attempts = 1
			record.FirstSeen = now
			return true
		}
		return false
	}

	// Check if window has expired
	if now.Sub(record.FirstSeen) > rl.config.WindowSize {
		// Reset window
		record.Attempts = 1
		record.FirstSeen = now
		return true
	}

	// Increment attempts
	record.Attempts++

	// Check if max attempts exceeded
	if record.Attempts > rl.config.MaxAttempts {
		record.IsBlocked = true
		record.BlockedAt = now
		return false
	}

	return true
}

// RecordFailure records a failed attempt (e.g., wrong password)
func (rl *RateLimiter) RecordFailure(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	record, exists := rl.records[ip]
	if !exists {
		rl.records[ip] = &ipRecord{
			Attempts:  1,
			FirstSeen: time.Now(),
		}
		return
	}

	record.Attempts++
	if record.Attempts > rl.config.MaxAttempts {
		record.IsBlocked = true
		record.BlockedAt = time.Now()
	}
}

// Reset resets the rate limit for an IP (e.g., after successful login)
func (rl *RateLimiter) Reset(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.records, ip)
}

// GetRemainingAttempts returns remaining attempts for an IP
func (rl *RateLimiter) GetRemainingAttempts(ip string) int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	record, exists := rl.records[ip]
	if !exists {
		return rl.config.MaxAttempts
	}

	if record.IsBlocked {
		return 0
	}

	remaining := rl.config.MaxAttempts - record.Attempts
	if remaining < 0 {
		return 0
	}
	return remaining
}

// GetBlockedUntil returns when the IP will be unblocked
func (rl *RateLimiter) GetBlockedUntil(ip string) time.Time {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	record, exists := rl.records[ip]
	if !exists || !record.IsBlocked {
		return time.Time{}
	}

	return record.BlockedAt.Add(rl.config.BlockDuration)
}

// cleanupLoop periodically removes expired records
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-rl.stopCh:
			return
		case <-ticker.C:
			rl.cleanup()
		}
	}
}

// cleanup removes expired records
func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	expiry := rl.config.WindowSize + rl.config.BlockDuration

	for ip, record := range rl.records {
		if now.Sub(record.FirstSeen) > expiry {
			delete(rl.records, ip)
		}
	}
}

// Stop stops the rate limiter cleanup goroutine
func (rl *RateLimiter) Stop() {
	close(rl.stopCh)
}

// =============================================================================
// GIN MIDDLEWARE
// =============================================================================

// RateLimitMiddleware returns a Gin middleware for rate limiting
func RateLimitMiddleware(rl *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		if !rl.Allow(ip) {
			blockedUntil := rl.GetBlockedUntil(ip)
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":         "Too many requests",
				"message":       "You have exceeded the rate limit. Please try again later.",
				"blocked_until": blockedUntil.UTC(),
				"retry_after":   int(time.Until(blockedUntil).Seconds()),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// AuthRateLimitMiddleware returns a stricter rate limiter for auth endpoints
func AuthRateLimitMiddleware() gin.HandlerFunc {
	rl := NewRateLimiter(AuthRateLimiterConfig())
	return RateLimitMiddleware(rl)
}

// SensitiveOperationRateLimiterConfig returns config for sensitive operations
// (password changes, wallet updates, payout settings)
func SensitiveOperationRateLimiterConfig() RateLimiterConfig {
	return RateLimiterConfig{
		MaxAttempts:     10,               // Stricter limit for sensitive ops
		WindowSize:      10 * time.Minute, // 10-minute window
		BlockDuration:   15 * time.Minute, // 15-minute block
		CleanupInterval: 2 * time.Minute,
	}
}

// APIRateLimiterConfig returns config for general API endpoints
func APIRateLimiterConfig() RateLimiterConfig {
	return RateLimiterConfig{
		MaxAttempts:     100,             // Higher limit for general API
		WindowSize:      1 * time.Minute, // Per-minute rate limiting
		BlockDuration:   1 * time.Minute, // Short block
		CleanupInterval: 30 * time.Second,
	}
}

// SensitiveRateLimitMiddleware returns a rate limiter for sensitive operations
func SensitiveRateLimitMiddleware() gin.HandlerFunc {
	rl := NewRateLimiter(SensitiveOperationRateLimiterConfig())
	return RateLimitMiddleware(rl)
}

// APIRateLimitMiddleware returns a rate limiter for general API endpoints
func APIRateLimitMiddleware() gin.HandlerFunc {
	rl := NewRateLimiter(APIRateLimiterConfig())
	return RateLimitMiddleware(rl)
}
