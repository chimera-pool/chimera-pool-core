package api

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// RATE LIMITER TESTS - Comprehensive Security Testing
// =============================================================================

func TestNewRateLimiter_DefaultConfig(t *testing.T) {
	config := DefaultRateLimiterConfig()
	rl := NewRateLimiter(config)
	defer rl.Stop()

	assert.NotNil(t, rl)
	assert.Equal(t, 5, rl.config.MaxAttempts)
	assert.Equal(t, 15*time.Minute, rl.config.WindowSize)
	assert.Equal(t, 30*time.Minute, rl.config.BlockDuration)
}

func TestNewRateLimiter_AuthConfig(t *testing.T) {
	config := AuthRateLimiterConfig()
	rl := NewRateLimiter(config)
	defer rl.Stop()

	assert.NotNil(t, rl)
	assert.Equal(t, 5, rl.config.MaxAttempts)
}

func TestRateLimiter_Allow_FirstRequest(t *testing.T) {
	config := DefaultRateLimiterConfig()
	rl := NewRateLimiter(config)
	defer rl.Stop()

	// First request should always be allowed
	allowed := rl.Allow("192.168.1.1")
	assert.True(t, allowed)
}

func TestRateLimiter_Allow_WithinLimit(t *testing.T) {
	config := RateLimiterConfig{
		MaxAttempts:     5,
		WindowSize:      15 * time.Minute,
		BlockDuration:   30 * time.Minute,
		CleanupInterval: 5 * time.Minute,
	}
	rl := NewRateLimiter(config)
	defer rl.Stop()

	ip := "192.168.1.2"

	// Should allow up to MaxAttempts
	for i := 0; i < 5; i++ {
		allowed := rl.Allow(ip)
		assert.True(t, allowed, "Request %d should be allowed", i+1)
	}
}

func TestRateLimiter_Allow_ExceedsLimit(t *testing.T) {
	config := RateLimiterConfig{
		MaxAttempts:     3,
		WindowSize:      15 * time.Minute,
		BlockDuration:   30 * time.Minute,
		CleanupInterval: 5 * time.Minute,
	}
	rl := NewRateLimiter(config)
	defer rl.Stop()

	ip := "192.168.1.3"

	// First 3 requests allowed
	for i := 0; i < 3; i++ {
		allowed := rl.Allow(ip)
		assert.True(t, allowed)
	}

	// 4th request should be blocked
	allowed := rl.Allow(ip)
	assert.False(t, allowed)
}

func TestRateLimiter_Allow_BlockedIP(t *testing.T) {
	config := RateLimiterConfig{
		MaxAttempts:     2,
		WindowSize:      15 * time.Minute,
		BlockDuration:   100 * time.Millisecond, // Short for testing
		CleanupInterval: 5 * time.Minute,
	}
	rl := NewRateLimiter(config)
	defer rl.Stop()

	ip := "192.168.1.4"

	// Exceed limit
	rl.Allow(ip)
	rl.Allow(ip)
	rl.Allow(ip) // Should block

	// Should be blocked
	assert.False(t, rl.Allow(ip))

	// Wait for block to expire
	time.Sleep(150 * time.Millisecond)

	// Should be allowed again
	assert.True(t, rl.Allow(ip))
}

func TestRateLimiter_Allow_WindowExpiry(t *testing.T) {
	config := RateLimiterConfig{
		MaxAttempts:     2,
		WindowSize:      50 * time.Millisecond, // Short for testing
		BlockDuration:   30 * time.Minute,
		CleanupInterval: 5 * time.Minute,
	}
	rl := NewRateLimiter(config)
	defer rl.Stop()

	ip := "192.168.1.5"

	// Use up attempts
	rl.Allow(ip)
	rl.Allow(ip)

	// Wait for window to expire
	time.Sleep(100 * time.Millisecond)

	// Should be allowed (window reset)
	assert.True(t, rl.Allow(ip))
}

func TestRateLimiter_Allow_MultipleIPs(t *testing.T) {
	config := RateLimiterConfig{
		MaxAttempts:     2,
		WindowSize:      15 * time.Minute,
		BlockDuration:   30 * time.Minute,
		CleanupInterval: 5 * time.Minute,
	}
	rl := NewRateLimiter(config)
	defer rl.Stop()

	ip1 := "192.168.1.10"
	ip2 := "192.168.1.11"

	// Exhaust IP1 limit
	rl.Allow(ip1)
	rl.Allow(ip1)
	rl.Allow(ip1) // Blocked

	// IP2 should still be allowed
	assert.True(t, rl.Allow(ip2))
	assert.True(t, rl.Allow(ip2))

	// IP1 should be blocked
	assert.False(t, rl.Allow(ip1))
}

func TestRateLimiter_RecordFailure(t *testing.T) {
	config := RateLimiterConfig{
		MaxAttempts:     3,
		WindowSize:      15 * time.Minute,
		BlockDuration:   30 * time.Minute,
		CleanupInterval: 5 * time.Minute,
	}
	rl := NewRateLimiter(config)
	defer rl.Stop()

	ip := "192.168.1.20"

	// Record failures without calling Allow
	rl.RecordFailure(ip)
	rl.RecordFailure(ip)
	rl.RecordFailure(ip)
	rl.RecordFailure(ip) // Should trigger block

	// Should be blocked
	assert.False(t, rl.Allow(ip))
}

func TestRateLimiter_Reset(t *testing.T) {
	config := RateLimiterConfig{
		MaxAttempts:     2,
		WindowSize:      15 * time.Minute,
		BlockDuration:   30 * time.Minute,
		CleanupInterval: 5 * time.Minute,
	}
	rl := NewRateLimiter(config)
	defer rl.Stop()

	ip := "192.168.1.30"

	// Exhaust and block
	rl.Allow(ip)
	rl.Allow(ip)
	rl.Allow(ip)
	assert.False(t, rl.Allow(ip))

	// Reset
	rl.Reset(ip)

	// Should be allowed again
	assert.True(t, rl.Allow(ip))
}

func TestRateLimiter_GetRemainingAttempts(t *testing.T) {
	config := RateLimiterConfig{
		MaxAttempts:     5,
		WindowSize:      15 * time.Minute,
		BlockDuration:   30 * time.Minute,
		CleanupInterval: 5 * time.Minute,
	}
	rl := NewRateLimiter(config)
	defer rl.Stop()

	ip := "192.168.1.40"

	// Initially should have max attempts
	assert.Equal(t, 5, rl.GetRemainingAttempts(ip))

	// After one request
	rl.Allow(ip)
	assert.Equal(t, 4, rl.GetRemainingAttempts(ip))

	// After more requests
	rl.Allow(ip)
	rl.Allow(ip)
	assert.Equal(t, 2, rl.GetRemainingAttempts(ip))
}

func TestRateLimiter_GetRemainingAttempts_Blocked(t *testing.T) {
	config := RateLimiterConfig{
		MaxAttempts:     2,
		WindowSize:      15 * time.Minute,
		BlockDuration:   30 * time.Minute,
		CleanupInterval: 5 * time.Minute,
	}
	rl := NewRateLimiter(config)
	defer rl.Stop()

	ip := "192.168.1.41"

	// Exhaust and block
	rl.Allow(ip)
	rl.Allow(ip)
	rl.Allow(ip)

	// Should return 0 when blocked
	assert.Equal(t, 0, rl.GetRemainingAttempts(ip))
}

func TestRateLimiter_GetBlockedUntil(t *testing.T) {
	config := RateLimiterConfig{
		MaxAttempts:     2,
		WindowSize:      15 * time.Minute,
		BlockDuration:   30 * time.Minute,
		CleanupInterval: 5 * time.Minute,
	}
	rl := NewRateLimiter(config)
	defer rl.Stop()

	ip := "192.168.1.50"

	// Not blocked yet
	blockedUntil := rl.GetBlockedUntil(ip)
	assert.True(t, blockedUntil.IsZero())

	// Block the IP
	rl.Allow(ip)
	rl.Allow(ip)
	rl.Allow(ip)

	// Should have a blocked until time
	blockedUntil = rl.GetBlockedUntil(ip)
	assert.False(t, blockedUntil.IsZero())
	assert.True(t, blockedUntil.After(time.Now()))
}

func TestRateLimiter_Cleanup(t *testing.T) {
	config := RateLimiterConfig{
		MaxAttempts:     5,
		WindowSize:      50 * time.Millisecond,
		BlockDuration:   50 * time.Millisecond,
		CleanupInterval: 10 * time.Millisecond,
	}
	rl := NewRateLimiter(config)
	defer rl.Stop()

	ip := "192.168.1.60"
	rl.Allow(ip)

	// Wait for cleanup to run
	time.Sleep(200 * time.Millisecond)

	// Should have max attempts again (record cleaned up)
	assert.Equal(t, 5, rl.GetRemainingAttempts(ip))
}

func TestRateLimiter_Concurrency(t *testing.T) {
	config := RateLimiterConfig{
		MaxAttempts:     100,
		WindowSize:      15 * time.Minute,
		BlockDuration:   30 * time.Minute,
		CleanupInterval: 5 * time.Minute,
	}
	rl := NewRateLimiter(config)
	defer rl.Stop()

	var wg sync.WaitGroup
	ip := "192.168.1.70"

	// Concurrent requests
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rl.Allow(ip)
		}()
	}

	wg.Wait()

	// Should have counted all attempts
	remaining := rl.GetRemainingAttempts(ip)
	assert.Equal(t, 50, remaining)
}

// =============================================================================
// MIDDLEWARE TESTS
// =============================================================================

func setupTestRouterForRateLimiter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestRateLimitMiddleware_AllowedRequest(t *testing.T) {
	config := RateLimiterConfig{
		MaxAttempts:     5,
		WindowSize:      15 * time.Minute,
		BlockDuration:   30 * time.Minute,
		CleanupInterval: 5 * time.Minute,
	}
	rl := NewRateLimiter(config)
	defer rl.Stop()

	router := setupTestRouterForRateLimiter()
	router.Use(RateLimitMiddleware(rl))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRateLimitMiddleware_BlockedRequest(t *testing.T) {
	config := RateLimiterConfig{
		MaxAttempts:     2,
		WindowSize:      15 * time.Minute,
		BlockDuration:   30 * time.Minute,
		CleanupInterval: 5 * time.Minute,
	}
	rl := NewRateLimiter(config)
	defer rl.Stop()

	router := setupTestRouterForRateLimiter()
	router.Use(RateLimitMiddleware(rl))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	// Exhaust rate limit
	ip := "192.168.1.101"
	rl.Allow(ip)
	rl.Allow(ip)
	rl.Allow(ip) // Block

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", ip)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestRateLimitMiddleware_ResponseBody(t *testing.T) {
	config := RateLimiterConfig{
		MaxAttempts:     1,
		WindowSize:      15 * time.Minute,
		BlockDuration:   30 * time.Minute,
		CleanupInterval: 5 * time.Minute,
	}
	rl := NewRateLimiter(config)
	defer rl.Stop()

	router := setupTestRouterForRateLimiter()
	router.Use(RateLimitMiddleware(rl))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	ip := "192.168.1.102"
	rl.Allow(ip)
	rl.Allow(ip) // Block

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", ip)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Contains(t, w.Body.String(), "Too many requests")
	assert.Contains(t, w.Body.String(), "retry_after")
	assert.Contains(t, w.Body.String(), "blocked_until")
}

func TestAuthRateLimitMiddleware(t *testing.T) {
	router := setupTestRouterForRateLimiter()
	router.Use(AuthRateLimitMiddleware())
	router.POST("/auth/login", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	req := httptest.NewRequest("POST", "/auth/login", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// =============================================================================
// EDGE CASES
// =============================================================================

func TestRateLimiter_EmptyIP(t *testing.T) {
	config := DefaultRateLimiterConfig()
	rl := NewRateLimiter(config)
	defer rl.Stop()

	// Empty IP should still work
	assert.True(t, rl.Allow(""))
	assert.Equal(t, 4, rl.GetRemainingAttempts(""))
}

func TestRateLimiter_IPv6(t *testing.T) {
	config := DefaultRateLimiterConfig()
	rl := NewRateLimiter(config)
	defer rl.Stop()

	ipv6 := "2001:0db8:85a3:0000:0000:8a2e:0370:7334"
	assert.True(t, rl.Allow(ipv6))
	assert.Equal(t, 4, rl.GetRemainingAttempts(ipv6))
}

func TestRateLimiter_Stop(t *testing.T) {
	config := RateLimiterConfig{
		MaxAttempts:     5,
		WindowSize:      15 * time.Minute,
		BlockDuration:   30 * time.Minute,
		CleanupInterval: 10 * time.Millisecond,
	}
	rl := NewRateLimiter(config)

	// Stop should not panic
	require.NotPanics(t, func() {
		rl.Stop()
	})
}

func TestRateLimiter_RecordFailure_NewIP(t *testing.T) {
	config := DefaultRateLimiterConfig()
	rl := NewRateLimiter(config)
	defer rl.Stop()

	ip := "192.168.1.200"

	// Record failure for new IP
	rl.RecordFailure(ip)

	// Should have 4 remaining (started at 1, not 0)
	assert.Equal(t, 4, rl.GetRemainingAttempts(ip))
}

// =============================================================================
// BENCHMARK TESTS
// =============================================================================

func BenchmarkRateLimiter_Allow(b *testing.B) {
	config := DefaultRateLimiterConfig()
	rl := NewRateLimiter(config)
	defer rl.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rl.Allow("192.168.1.1")
		rl.Reset("192.168.1.1")
	}
}

func BenchmarkRateLimiter_Allow_Concurrent(b *testing.B) {
	config := RateLimiterConfig{
		MaxAttempts:     1000000,
		WindowSize:      15 * time.Minute,
		BlockDuration:   30 * time.Minute,
		CleanupInterval: 5 * time.Minute,
	}
	rl := NewRateLimiter(config)
	defer rl.Stop()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rl.Allow("192.168.1.1")
		}
	})
}
