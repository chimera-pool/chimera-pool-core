package security

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// skipTimingTest skips timing-sensitive tests in CI environments
func skipTimingTest(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping timing-sensitive test - set INTEGRATION_TEST=true to run")
	}
}

func TestRateLimiterBasicFunctionality(t *testing.T) {
	limiter := NewRateLimiter(RateLimiterConfig{
		RequestsPerMinute: 10,
		BurstSize:         5,
		CleanupInterval:   time.Minute,
	})

	clientID := "test-client"
	ctx := context.Background()

	// Should allow initial requests up to burst size
	for i := 0; i < 5; i++ {
		allowed, err := limiter.Allow(ctx, clientID)
		require.NoError(t, err)
		assert.True(t, allowed, "request %d should be allowed", i+1)
	}

	// Should deny requests beyond burst size
	allowed, err := limiter.Allow(ctx, clientID)
	require.NoError(t, err)
	assert.False(t, allowed, "request beyond burst should be denied")
}

func TestRateLimiterTokenRefill(t *testing.T) {
	skipTimingTest(t)
	limiter := NewRateLimiter(RateLimiterConfig{
		RequestsPerMinute: 60, // 1 request per second
		BurstSize:         1,
		CleanupInterval:   time.Minute,
	})

	clientID := "test-client"
	ctx := context.Background()

	// Use up the burst
	allowed, err := limiter.Allow(ctx, clientID)
	require.NoError(t, err)
	assert.True(t, allowed)

	// Should be denied immediately
	allowed, err = limiter.Allow(ctx, clientID)
	require.NoError(t, err)
	assert.False(t, allowed)

	// Wait for token refill (slightly more than 1 second)
	time.Sleep(1100 * time.Millisecond)

	// Should be allowed again
	allowed, err = limiter.Allow(ctx, clientID)
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestRateLimiterMultipleClients(t *testing.T) {
	limiter := NewRateLimiter(RateLimiterConfig{
		RequestsPerMinute: 10,
		BurstSize:         2,
		CleanupInterval:   time.Minute,
	})

	ctx := context.Background()

	// Client 1 uses up its quota
	for i := 0; i < 2; i++ {
		allowed, err := limiter.Allow(ctx, "client1")
		require.NoError(t, err)
		assert.True(t, allowed)
	}

	// Client 1 should be denied
	allowed, err := limiter.Allow(ctx, "client1")
	require.NoError(t, err)
	assert.False(t, allowed)

	// Client 2 should still be allowed
	allowed, err = limiter.Allow(ctx, "client2")
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestProgressiveRateLimiting(t *testing.T) {
	limiter := NewProgressiveRateLimiter(ProgressiveRateLimiterConfig{
		BaseRequestsPerMinute: 10,
		BaseBurstSize:         5,
		MaxPenaltyMultiplier:  10,
		PenaltyDuration:       time.Minute,
		CleanupInterval:       time.Minute,
	})

	clientID := "test-client"
	ctx := context.Background()

	// Normal usage should work
	for i := 0; i < 5; i++ {
		allowed, err := limiter.Allow(ctx, clientID)
		require.NoError(t, err)
		assert.True(t, allowed)
	}

	// Trigger rate limiting
	allowed, err := limiter.Allow(ctx, clientID)
	require.NoError(t, err)
	assert.False(t, allowed)

	// Record violation
	err = limiter.RecordViolation(ctx, clientID, ViolationTypeBruteForce)
	require.NoError(t, err)

	// Should have reduced limits now
	info, err := limiter.GetClientInfo(ctx, clientID)
	require.NoError(t, err)
	assert.Greater(t, info.PenaltyMultiplier, 1.0)
	assert.True(t, info.UnderPenalty)
}

func TestBruteForceProtection(t *testing.T) {
	protector := NewBruteForceProtector(BruteForceConfig{
		MaxAttempts:     3,
		WindowDuration:  time.Minute,
		LockoutDuration: 5 * time.Minute,
		CleanupInterval: time.Minute,
	})

	clientID := "test-client"
	ctx := context.Background()

	// Should allow initial attempts
	for i := 0; i < 3; i++ {
		allowed, err := protector.CheckAttempt(ctx, clientID)
		require.NoError(t, err)
		assert.True(t, allowed, "attempt %d should be allowed", i+1)

		// Record failed attempt
		err = protector.RecordFailedAttempt(ctx, clientID)
		require.NoError(t, err)
	}

	// Should be locked out after max attempts
	allowed, err := protector.CheckAttempt(ctx, clientID)
	require.NoError(t, err)
	assert.False(t, allowed, "should be locked out after max attempts")

	// Should still be locked out
	allowed, err = protector.CheckAttempt(ctx, clientID)
	require.NoError(t, err)
	assert.False(t, allowed, "should still be locked out")
}

func TestBruteForceProtectionReset(t *testing.T) {
	skipTimingTest(t)
	protector := NewBruteForceProtector(BruteForceConfig{
		MaxAttempts:     2,
		WindowDuration:  time.Minute,
		LockoutDuration: 100 * time.Millisecond, // Short lockout for testing
		CleanupInterval: time.Minute,
	})

	clientID := "test-client"
	ctx := context.Background()

	// Trigger lockout
	for i := 0; i < 2; i++ {
		allowed, err := protector.CheckAttempt(ctx, clientID)
		require.NoError(t, err)
		assert.True(t, allowed)

		err = protector.RecordFailedAttempt(ctx, clientID)
		require.NoError(t, err)
	}

	// Should be locked out
	allowed, err := protector.CheckAttempt(ctx, clientID)
	require.NoError(t, err)
	assert.False(t, allowed)

	// Wait for lockout to expire
	time.Sleep(150 * time.Millisecond)

	// Should be allowed again
	allowed, err = protector.CheckAttempt(ctx, clientID)
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestBruteForceProtectionSuccessfulReset(t *testing.T) {
	protector := NewBruteForceProtector(BruteForceConfig{
		MaxAttempts:     3,
		WindowDuration:  time.Minute,
		LockoutDuration: 5 * time.Minute,
		CleanupInterval: time.Minute,
	})

	clientID := "test-client"
	ctx := context.Background()

	// Record some failed attempts
	for i := 0; i < 2; i++ {
		allowed, err := protector.CheckAttempt(ctx, clientID)
		require.NoError(t, err)
		assert.True(t, allowed)

		err = protector.RecordFailedAttempt(ctx, clientID)
		require.NoError(t, err)
	}

	// Record successful attempt (should reset counter)
	err := protector.RecordSuccessfulAttempt(ctx, clientID)
	require.NoError(t, err)

	// Should allow more attempts now
	for i := 0; i < 3; i++ {
		allowed, err := protector.CheckAttempt(ctx, clientID)
		require.NoError(t, err)
		assert.True(t, allowed, "attempt %d should be allowed after reset", i+1)

		err = protector.RecordFailedAttempt(ctx, clientID)
		require.NoError(t, err)
	}

	// Now should be locked out
	allowed, err := protector.CheckAttempt(ctx, clientID)
	require.NoError(t, err)
	assert.False(t, allowed)
}

func TestDDoSProtection(t *testing.T) {
	protector := NewDDoSProtector(DDoSConfig{
		RequestsPerSecond:   10,
		BurstSize:           20,
		SuspiciousThreshold: 100,
		BlockDuration:       time.Minute,
		CleanupInterval:     time.Minute,
	})

	ctx := context.Background()

	// Normal traffic should be allowed
	for i := 0; i < 10; i++ {
		allowed, err := protector.CheckRequest(ctx, "normal-client")
		require.NoError(t, err)
		assert.True(t, allowed)
	}

	// Burst traffic should be allowed up to burst size
	for i := 0; i < 20; i++ {
		allowed, err := protector.CheckRequest(ctx, "burst-client")
		require.NoError(t, err)
		assert.True(t, allowed)
	}

	// Excessive traffic should be blocked
	allowed, err := protector.CheckRequest(ctx, "burst-client")
	require.NoError(t, err)
	assert.False(t, allowed)
}

func TestDDoSProtectionSuspiciousActivity(t *testing.T) {
	protector := NewDDoSProtector(DDoSConfig{
		RequestsPerSecond:   10,
		BurstSize:           20,
		SuspiciousThreshold: 50,
		BlockDuration:       time.Minute,
		CleanupInterval:     time.Minute,
	})

	ctx := context.Background()
	clientID := "suspicious-client"

	// Generate suspicious traffic pattern
	for i := 0; i < 60; i++ {
		protector.CheckRequest(ctx, clientID)
		time.Sleep(10 * time.Millisecond) // Very fast requests
	}

	// Client should be flagged as suspicious
	info, err := protector.GetClientInfo(ctx, clientID)
	require.NoError(t, err)
	assert.True(t, info.IsSuspicious)
	assert.Greater(t, info.RequestCount, 50)
}

func TestIntrusionDetection(t *testing.T) {
	detector := NewIntrusionDetector(IntrusionDetectionConfig{
		SuspiciousPatterns: []string{
			`(?i)(union|select|insert|delete|drop|create|alter)`,
			`(?i)(script|javascript|vbscript)`,
			`(?i)(<script|<iframe|<object)`,
		},
		MaxViolationsPerHour: 5,
		BlockDuration:        time.Hour,
		CleanupInterval:      time.Minute,
	})

	ctx := context.Background()
	clientID := "test-client"

	tests := []struct {
		name      string
		input     string
		malicious bool
	}{
		{
			name:      "normal input",
			input:     "normal user input",
			malicious: false,
		},
		{
			name:      "SQL injection attempt",
			input:     "'; DROP TABLE users; --",
			malicious: true,
		},
		{
			name:      "XSS attempt",
			input:     "<script>alert('xss')</script>",
			malicious: true,
		},
		{
			name:      "JavaScript injection",
			input:     "javascript:alert('test')",
			malicious: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			threat, err := detector.AnalyzeRequest(ctx, clientID, tt.input)
			require.NoError(t, err)

			if tt.malicious {
				assert.True(t, threat.IsMalicious, "input should be detected as malicious")
				assert.NotEmpty(t, threat.MatchedPatterns, "should have matched patterns")
			} else {
				assert.False(t, threat.IsMalicious, "input should not be detected as malicious")
				assert.Empty(t, threat.MatchedPatterns, "should not have matched patterns")
			}
		})
	}
}

func TestIntrusionDetectionBlocking(t *testing.T) {
	skipTimingTest(t)
	detector := NewIntrusionDetector(IntrusionDetectionConfig{
		SuspiciousPatterns: []string{
			`(?i)(union|select)`,
		},
		MaxViolationsPerHour: 2,
		BlockDuration:        time.Minute,
		CleanupInterval:      time.Minute,
	})

	ctx := context.Background()
	clientID := "malicious-client"

	// Generate violations
	for i := 0; i < 3; i++ {
		threat, err := detector.AnalyzeRequest(ctx, clientID, "SELECT * FROM users")
		require.NoError(t, err)
		assert.True(t, threat.IsMalicious)
	}

	// Client should be blocked
	blocked, err := detector.IsBlocked(ctx, clientID)
	require.NoError(t, err)
	assert.True(t, blocked)

	// Further requests should be blocked
	threat, err := detector.AnalyzeRequest(ctx, clientID, "normal input")
	require.NoError(t, err)
	assert.True(t, threat.IsBlocked)
}
