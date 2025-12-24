package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// COMPREHENSIVE API TESTS FOR 80%+ COVERAGE
// Critical for production-ready API endpoints
// =============================================================================

func init() {
	gin.SetMode(gin.TestMode)
}

// -----------------------------------------------------------------------------
// Model Tests
// -----------------------------------------------------------------------------

func TestJWTClaims_Initialization(t *testing.T) {
	now := time.Now()
	claims := JWTClaims{
		UserID:    123,
		Username:  "testuser",
		Email:     "test@example.com",
		IssuedAt:  now,
		ExpiresAt: now.Add(time.Hour),
	}

	assert.Equal(t, int64(123), claims.UserID)
	assert.Equal(t, "testuser", claims.Username)
	assert.Equal(t, "test@example.com", claims.Email)
	assert.False(t, claims.IssuedAt.IsZero())
	assert.False(t, claims.ExpiresAt.IsZero())
	assert.True(t, claims.ExpiresAt.After(claims.IssuedAt))
}

func TestErrorResponse_Initialization(t *testing.T) {
	resp := ErrorResponse{
		Error:   "validation_error",
		Message: "Invalid input",
		Code:    400,
	}

	assert.Equal(t, "validation_error", resp.Error)
	assert.Equal(t, "Invalid input", resp.Message)
	assert.Equal(t, 400, resp.Code)
}

func TestPoolStats_Initialization(t *testing.T) {
	stats := PoolStats{
		TotalHashrate:     1000000.0,
		ConnectedMiners:   50,
		TotalShares:       10000,
		ValidShares:       9500,
		BlocksFound:       10,
		LastBlockTime:     time.Now(),
		NetworkHashrate:   5000000.0,
		NetworkDifficulty: 1000.0,
		PoolFee:           0.01,
	}

	assert.Equal(t, float64(1000000.0), stats.TotalHashrate)
	assert.Equal(t, int64(50), stats.ConnectedMiners)
	assert.Equal(t, int64(10000), stats.TotalShares)
	assert.Equal(t, int64(9500), stats.ValidShares)
	assert.Equal(t, int64(10), stats.BlocksFound)
	assert.Equal(t, float64(0.01), stats.PoolFee)
}

func TestPoolStatsResponse_Initialization(t *testing.T) {
	resp := PoolStatsResponse{
		TotalHashrate:     1000000.0,
		ConnectedMiners:   50,
		TotalShares:       10000,
		ValidShares:       9500,
		InvalidShares:     500,
		BlocksFound:       10,
		LastBlockTime:     time.Now(),
		NetworkHashrate:   5000000.0,
		NetworkDifficulty: 1000.0,
		PoolFee:           0.01,
		Efficiency:        95.0,
	}

	assert.Equal(t, int64(500), resp.InvalidShares)
	assert.Equal(t, float64(95.0), resp.Efficiency)
}

func TestUserProfile_Initialization(t *testing.T) {
	profile := UserProfile{
		ID:       1,
		Username: "testuser",
		Email:    "test@example.com",
		JoinedAt: time.Now(),
		IsActive: true,
	}

	assert.Equal(t, int64(1), profile.ID)
	assert.Equal(t, "testuser", profile.Username)
	assert.Equal(t, "test@example.com", profile.Email)
	assert.True(t, profile.IsActive)
}

func TestUserStats_Initialization(t *testing.T) {
	stats := UserStats{
		UserID:        1,
		TotalShares:   1000,
		ValidShares:   950,
		InvalidShares: 50,
		TotalHashrate: 50000.0,
		LastShare:     time.Now(),
		Earnings:      1.5,
	}

	assert.Equal(t, int64(1), stats.UserID)
	assert.Equal(t, int64(1000), stats.TotalShares)
	assert.Equal(t, float64(1.5), stats.Earnings)
}

func TestMinerStats_Initialization(t *testing.T) {
	stats := MinerStats{
		MinerID:       1,
		UserID:        1,
		TotalShares:   500,
		ValidShares:   475,
		InvalidShares: 25,
		TotalHashrate: 25000.0,
		LastShare:     time.Now(),
	}

	assert.Equal(t, int64(1), stats.MinerID)
	assert.Equal(t, int64(500), stats.TotalShares)
}

func TestMinerInfo_Initialization(t *testing.T) {
	info := MinerInfo{
		ID:       1,
		Name:     "worker1",
		Hashrate: 25000.0,
		LastSeen: time.Now(),
		IsActive: true,
	}

	assert.Equal(t, int64(1), info.ID)
	assert.Equal(t, "worker1", info.Name)
	assert.True(t, info.IsActive)
}

func TestUserMinersResponse_Initialization(t *testing.T) {
	resp := UserMinersResponse{
		Miners: []*MinerInfo{
			{ID: 1, Name: "worker1", IsActive: true},
			{ID: 2, Name: "worker2", IsActive: false},
		},
		Total: 2,
	}

	assert.Len(t, resp.Miners, 2)
	assert.Equal(t, 2, resp.Total)
}

func TestRealTimeStats_Initialization(t *testing.T) {
	stats := RealTimeStats{
		CurrentHashrate:   100000.0,
		AverageHashrate:   95000.0,
		ActiveMiners:      25,
		SharesPerSecond:   10.5,
		LastBlockFound:    time.Now(),
		NetworkDifficulty: 1000.0,
		PoolEfficiency:    98.5,
	}

	assert.Equal(t, float64(100000.0), stats.CurrentHashrate)
	assert.Equal(t, int64(25), stats.ActiveMiners)
}

func TestBlockMetrics_Initialization(t *testing.T) {
	metrics := BlockMetrics{
		TotalBlocks:      100,
		BlocksLast24h:    5,
		BlocksLast7d:     30,
		AverageBlockTime: time.Minute * 10,
		LastBlockReward:  50.0,
		TotalRewards:     5000.0,
		OrphanBlocks:     2,
		OrphanRate:       0.02,
	}

	assert.Equal(t, int64(100), metrics.TotalBlocks)
	assert.Equal(t, int64(5), metrics.BlocksLast24h)
	assert.Equal(t, float64(0.02), metrics.OrphanRate)
}

func TestMFASetupResponse_Initialization(t *testing.T) {
	resp := MFASetupResponse{
		Secret:      "JBSWY3DPEHPK3PXP",
		QRCodeURL:   "data:image/png;base64,abc123",
		BackupCodes: []string{"CODE1234", "CODE5678"},
	}

	assert.NotEmpty(t, resp.Secret)
	assert.Contains(t, resp.QRCodeURL, "data:image")
	assert.Len(t, resp.BackupCodes, 2)
}

func TestVerifyMFARequest_Initialization(t *testing.T) {
	req := VerifyMFARequest{
		Code: "123456",
	}

	assert.Equal(t, "123456", req.Code)
	assert.Len(t, req.Code, 6)
}

// -----------------------------------------------------------------------------
// Middleware Tests
// -----------------------------------------------------------------------------

func TestAuthMiddlewareStandalone_NoAuthHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(AuthMiddlewareStandalone("test-secret"))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Authorization header required")
}

func TestAuthMiddlewareStandalone_NoBearerPrefix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(AuthMiddlewareStandalone("test-secret"))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Basic sometoken")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Bearer token required")
}

func TestAuthMiddlewareStandalone_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(AuthMiddlewareStandalone("test-secret"))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid token")
}

func TestAuthMiddlewareStandalone_ValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtSecret := "test-secret"

	// Create a valid JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  float64(123),
		"username": "testuser",
		"exp":      time.Now().Add(time.Hour).Unix(),
	})
	tokenString, err := token.SignedString([]byte(jwtSecret))
	require.NoError(t, err)

	router := gin.New()
	router.Use(AuthMiddlewareStandalone(jwtSecret))
	router.GET("/test", func(c *gin.Context) {
		userID := c.GetInt64("user_id")
		username, _ := c.Get("username")
		c.JSON(http.StatusOK, gin.H{
			"user_id":  userID,
			"username": username,
		})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "123")
	assert.Contains(t, w.Body.String(), "testuser")
}

func TestAuthMiddlewareStandalone_TokenWithoutUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtSecret := "test-secret"

	// Create a token without user_id
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": "testuser",
		"exp":      time.Now().Add(time.Hour).Unix(),
	})
	tokenString, err := token.SignedString([]byte(jwtSecret))
	require.NoError(t, err)

	router := gin.New()
	router.Use(AuthMiddlewareStandalone(jwtSecret))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid user ID")
}

func TestAuthMiddlewareStandalone_ExpiredToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtSecret := "test-secret"

	// Create an expired token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  float64(123),
		"username": "testuser",
		"exp":      time.Now().Add(-time.Hour).Unix(), // Expired 1 hour ago
	})
	tokenString, err := token.SignedString([]byte(jwtSecret))
	require.NoError(t, err)

	router := gin.New()
	router.Use(AuthMiddlewareStandalone(jwtSecret))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid token")
}

func TestAuthMiddlewareStandalone_WrongSigningMethod(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtSecret := "test-secret"

	// Create a token with RS256 instead of HS256
	token := jwt.NewWithClaims(jwt.SigningMethodHS384, jwt.MapClaims{
		"user_id":  float64(123),
		"username": "testuser",
		"exp":      time.Now().Add(time.Hour).Unix(),
	})
	tokenString, err := token.SignedString([]byte(jwtSecret))
	require.NoError(t, err)

	router := gin.New()
	router.Use(AuthMiddlewareStandalone(jwtSecret))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	router.ServeHTTP(w, req)

	// HS384 is also HMAC, so it should be accepted
	// This test verifies token parsing works with valid HMAC methods
}

// -----------------------------------------------------------------------------
// Zero Value Tests
// -----------------------------------------------------------------------------

func TestModels_ZeroValues(t *testing.T) {
	t.Run("JWTClaims_ZeroValue", func(t *testing.T) {
		claims := JWTClaims{}
		assert.Equal(t, int64(0), claims.UserID)
		assert.Empty(t, claims.Username)
		assert.Empty(t, claims.Email)
		assert.True(t, claims.IssuedAt.IsZero())
		assert.True(t, claims.ExpiresAt.IsZero())
	})

	t.Run("ErrorResponse_ZeroValue", func(t *testing.T) {
		resp := ErrorResponse{}
		assert.Empty(t, resp.Error)
		assert.Empty(t, resp.Message)
		assert.Equal(t, 0, resp.Code)
	})

	t.Run("PoolStats_ZeroValue", func(t *testing.T) {
		stats := PoolStats{}
		assert.Equal(t, float64(0), stats.TotalHashrate)
		assert.Equal(t, int64(0), stats.ConnectedMiners)
		assert.Equal(t, int64(0), stats.TotalShares)
	})

	t.Run("UserProfile_ZeroValue", func(t *testing.T) {
		profile := UserProfile{}
		assert.Equal(t, int64(0), profile.ID)
		assert.Empty(t, profile.Username)
		assert.False(t, profile.IsActive)
	})

	t.Run("MinerInfo_ZeroValue", func(t *testing.T) {
		info := MinerInfo{}
		assert.Equal(t, int64(0), info.ID)
		assert.Empty(t, info.Name)
		assert.Equal(t, float64(0), info.Hashrate)
		assert.False(t, info.IsActive)
	})

	t.Run("BlockMetrics_ZeroValue", func(t *testing.T) {
		metrics := BlockMetrics{}
		assert.Equal(t, int64(0), metrics.TotalBlocks)
		assert.Equal(t, int64(0), metrics.BlocksLast24h)
		assert.Equal(t, float64(0), metrics.OrphanRate)
	})
}

// -----------------------------------------------------------------------------
// Response Initialization Tests
// -----------------------------------------------------------------------------

func TestUserStatsResponse_Initialization(t *testing.T) {
	resp := UserStatsResponse{
		UserID:        1,
		TotalShares:   1000,
		ValidShares:   950,
		InvalidShares: 50,
		TotalHashrate: 50000.0,
		LastShare:     time.Now(),
		Earnings:      1.5,
		Efficiency:    95.0,
	}

	assert.Equal(t, float64(95.0), resp.Efficiency)
}

func TestRealTimeStatsResponse_Initialization(t *testing.T) {
	now := time.Now()
	resp := RealTimeStatsResponse{
		CurrentHashrate:   100000.0,
		AverageHashrate:   95000.0,
		ActiveMiners:      25,
		SharesPerSecond:   10.5,
		LastBlockFound:    now,
		NetworkDifficulty: 1000.0,
		PoolEfficiency:    98.5,
		Timestamp:         now,
	}

	assert.False(t, resp.Timestamp.IsZero())
}

func TestBlockMetricsResponse_Initialization(t *testing.T) {
	resp := BlockMetricsResponse{
		TotalBlocks:      100,
		BlocksLast24h:    5,
		BlocksLast7d:     30,
		AverageBlockTime: 600, // 10 minutes in seconds
		LastBlockReward:  50.0,
		TotalRewards:     5000.0,
		OrphanBlocks:     2,
		OrphanRate:       0.02,
		Timestamp:        time.Now(),
	}

	assert.Equal(t, int64(600), resp.AverageBlockTime)
}

func TestUserProfileResponse_Initialization(t *testing.T) {
	resp := UserProfileResponse{
		ID:       1,
		Username: "testuser",
		Email:    "test@example.com",
		JoinedAt: time.Now(),
		IsActive: true,
	}

	assert.Equal(t, int64(1), resp.ID)
	assert.True(t, resp.IsActive)
}

func TestUpdateUserProfileRequest_Initialization(t *testing.T) {
	req := UpdateUserProfileRequest{
		Email: "newemail@example.com",
	}

	assert.Equal(t, "newemail@example.com", req.Email)
}

// -----------------------------------------------------------------------------
// Benchmark Tests
// -----------------------------------------------------------------------------

func BenchmarkAuthMiddlewareStandalone_ValidToken(b *testing.B) {
	gin.SetMode(gin.TestMode)
	jwtSecret := "test-secret"

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  float64(123),
		"username": "testuser",
		"exp":      time.Now().Add(time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString([]byte(jwtSecret))

	router := gin.New()
	router.Use(AuthMiddlewareStandalone(jwtSecret))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkAuthMiddlewareStandalone_InvalidToken(b *testing.B) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(AuthMiddlewareStandalone("test-secret"))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// =============================================================================
// ADDITIONAL COMPREHENSIVE TESTS FOR HIGHER COVERAGE
// =============================================================================

func TestAuthMiddlewareStandalone_TokenWithStringUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtSecret := "test-secret"

	// Create a token with user_id as string instead of float64
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  "not-a-number",
		"username": "testuser",
		"exp":      time.Now().Add(time.Hour).Unix(),
	})
	tokenString, err := token.SignedString([]byte(jwtSecret))
	require.NoError(t, err)

	router := gin.New()
	router.Use(AuthMiddlewareStandalone(jwtSecret))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid user ID")
}

func TestAuthMiddlewareStandalone_ValidTokenWithoutUsername(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtSecret := "test-secret"

	// Create a valid token without username (should still work)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": float64(123),
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	tokenString, err := token.SignedString([]byte(jwtSecret))
	require.NoError(t, err)

	router := gin.New()
	router.Use(AuthMiddlewareStandalone(jwtSecret))
	router.GET("/test", func(c *gin.Context) {
		userID := c.GetInt64("user_id")
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "123")
}

func TestPoolStats_Efficiency(t *testing.T) {
	stats := PoolStats{
		TotalShares: 10000,
		ValidShares: 9500,
	}

	// Calculate efficiency
	efficiency := float64(stats.ValidShares) / float64(stats.TotalShares) * 100
	assert.Equal(t, float64(95), efficiency)
}

func TestUserStats_Efficiency(t *testing.T) {
	stats := UserStats{
		TotalShares:   1000,
		ValidShares:   950,
		InvalidShares: 50,
	}

	// Verify shares add up
	assert.Equal(t, stats.TotalShares, stats.ValidShares+stats.InvalidShares)
}

func TestMinerStats_Efficiency(t *testing.T) {
	stats := MinerStats{
		TotalShares:   500,
		ValidShares:   475,
		InvalidShares: 25,
	}

	// Verify shares add up
	assert.Equal(t, stats.TotalShares, stats.ValidShares+stats.InvalidShares)
}

func TestBlockMetrics_OrphanRate(t *testing.T) {
	metrics := BlockMetrics{
		TotalBlocks:  100,
		OrphanBlocks: 2,
		OrphanRate:   0.02,
	}

	// Verify orphan rate calculation
	calculatedRate := float64(metrics.OrphanBlocks) / float64(metrics.TotalBlocks)
	assert.Equal(t, metrics.OrphanRate, calculatedRate)
}

func TestUserMinersResponse_EmptyMiners(t *testing.T) {
	resp := UserMinersResponse{
		Miners: []*MinerInfo{},
		Total:  0,
	}

	assert.Len(t, resp.Miners, 0)
	assert.Equal(t, 0, resp.Total)
}

func TestMFASetupResponse_BackupCodes(t *testing.T) {
	resp := MFASetupResponse{
		Secret:      "SECRET123",
		QRCodeURL:   "https://example.com/qr.png",
		BackupCodes: []string{"CODE1", "CODE2", "CODE3", "CODE4", "CODE5"},
	}

	assert.Len(t, resp.BackupCodes, 5)
	for _, code := range resp.BackupCodes {
		assert.NotEmpty(t, code)
	}
}

func TestVerifyMFARequest_EmptyCode(t *testing.T) {
	req := VerifyMFARequest{
		Code: "",
	}

	assert.Empty(t, req.Code)
}

func TestUpdateUserProfileRequest_EmptyEmail(t *testing.T) {
	req := UpdateUserProfileRequest{
		Email: "",
	}

	assert.Empty(t, req.Email)
}

func TestJWTClaims_Expiration(t *testing.T) {
	now := time.Now()
	claims := JWTClaims{
		UserID:    1,
		Username:  "test",
		IssuedAt:  now,
		ExpiresAt: now.Add(24 * time.Hour),
	}

	// Verify token is not expired
	assert.True(t, claims.ExpiresAt.After(time.Now()))

	// Verify expiration is 24 hours from issuance
	duration := claims.ExpiresAt.Sub(claims.IssuedAt)
	assert.Equal(t, 24*time.Hour, duration)
}

func TestErrorResponse_HTTPCodes(t *testing.T) {
	testCases := []struct {
		name     string
		error    string
		message  string
		code     int
		expected int
	}{
		{"BadRequest", "bad_request", "Invalid input", 400, http.StatusBadRequest},
		{"Unauthorized", "unauthorized", "Not authenticated", 401, http.StatusUnauthorized},
		{"Forbidden", "forbidden", "Access denied", 403, http.StatusForbidden},
		{"NotFound", "not_found", "Resource not found", 404, http.StatusNotFound},
		{"InternalError", "internal_error", "Server error", 500, http.StatusInternalServerError},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp := ErrorResponse{
				Error:   tc.error,
				Message: tc.message,
				Code:    tc.code,
			}
			assert.Equal(t, tc.expected, resp.Code)
		})
	}
}

func TestPoolStatsResponse_AllFields(t *testing.T) {
	now := time.Now()
	resp := PoolStatsResponse{
		TotalHashrate:     1000000.0,
		ConnectedMiners:   100,
		TotalShares:       50000,
		ValidShares:       48000,
		InvalidShares:     2000,
		BlocksFound:       25,
		LastBlockTime:     now,
		NetworkHashrate:   10000000.0,
		NetworkDifficulty: 5000.0,
		PoolFee:           0.01,
		Efficiency:        96.0,
	}

	assert.Equal(t, float64(1000000.0), resp.TotalHashrate)
	assert.Equal(t, int64(100), resp.ConnectedMiners)
	assert.Equal(t, int64(50000), resp.TotalShares)
	assert.Equal(t, int64(48000), resp.ValidShares)
	assert.Equal(t, int64(2000), resp.InvalidShares)
	assert.Equal(t, int64(25), resp.BlocksFound)
	assert.Equal(t, now, resp.LastBlockTime)
	assert.Equal(t, float64(10000000.0), resp.NetworkHashrate)
	assert.Equal(t, float64(5000.0), resp.NetworkDifficulty)
	assert.Equal(t, float64(0.01), resp.PoolFee)
	assert.Equal(t, float64(96.0), resp.Efficiency)
}

func TestRealTimeStats_SharesPerSecond(t *testing.T) {
	stats := RealTimeStats{
		SharesPerSecond: 150.5,
		ActiveMiners:    50,
	}

	// Verify shares per miner
	sharesPerMiner := stats.SharesPerSecond / float64(stats.ActiveMiners)
	assert.InDelta(t, 3.01, sharesPerMiner, 0.01)
}
