package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// E2ETestSuite provides end-to-end testing for the API
type E2ETestSuite struct {
	suite.Suite
	router   *gin.Engine
	handlers *APIHandlers
	mockAuth *MockAuthService
	mockPool *MockPoolStatsService
	mockUser *MockUserService
}

// SetupSuite initializes the test suite
func (suite *E2ETestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)
	
	// Create mock services
	suite.mockAuth = &MockAuthService{}
	suite.mockPool = &MockPoolStatsService{}
	suite.mockUser = &MockUserService{}
	
	// Create handlers
	suite.handlers = NewAPIHandlers(suite.mockAuth, suite.mockPool, suite.mockUser)
	
	// Setup router
	suite.router = gin.New()
	SetupAPIRoutes(suite.router, suite.handlers)
}

// TestE2EAPIWorkflows runs the E2E test suite
func TestE2EAPIWorkflows(t *testing.T) {
	suite.Run(t, new(E2ETestSuite))
}

// TestCompleteUserWorkflow tests the complete user workflow from authentication to data access
func (suite *E2ETestSuite) TestCompleteUserWorkflow() {
	userID := int64(123)
	token := "valid-jwt-token"
	
	// Setup mock expectations for authentication
	claims := &JWTClaims{
		UserID:    userID,
		Username:  "testuser",
		Email:     "test@example.com",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	suite.mockAuth.On("ValidateJWT", token).Return(claims, nil)
	
	// Setup mock expectations for user profile
	userProfile := &UserProfile{
		ID:       userID,
		Username: "testuser",
		Email:    "test@example.com",
		JoinedAt: time.Now().Add(-30 * 24 * time.Hour),
		IsActive: true,
	}
	suite.mockUser.On("GetUserProfile", userID).Return(userProfile, nil)
	
	// Setup mock expectations for user stats
	userStats := &UserStats{
		UserID:        userID,
		TotalShares:   1000,
		ValidShares:   990,
		InvalidShares: 10,
		TotalHashrate: 50000.0,
		LastShare:     time.Now().Add(-5 * time.Minute),
		Earnings:      0.05,
	}
	suite.mockPool.On("GetUserStats", userID).Return(userStats, nil)
	
	// Setup mock expectations for user miners
	miners := []*MinerInfo{
		{
			ID:       1,
			Name:     "miner-1",
			Hashrate: 25000.0,
			LastSeen: time.Now().Add(-2 * time.Minute),
			IsActive: true,
		},
		{
			ID:       2,
			Name:     "miner-2",
			Hashrate: 25000.0,
			LastSeen: time.Now().Add(-1 * time.Minute),
			IsActive: true,
		},
	}
	suite.mockUser.On("GetUserMiners", userID).Return(miners, nil)
	
	// Test 1: Get user profile
	req := httptest.NewRequest("GET", "/api/v1/user/profile", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	
	assert.Equal(suite.T(), http.StatusOK, w.Code)
	
	var profileResponse UserProfileResponse
	err := json.Unmarshal(w.Body.Bytes(), &profileResponse)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), userProfile.Username, profileResponse.Username)
	assert.Equal(suite.T(), userProfile.Email, profileResponse.Email)
	
	// Test 2: Get user stats
	req = httptest.NewRequest("GET", "/api/v1/user/stats", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	
	assert.Equal(suite.T(), http.StatusOK, w.Code)
	
	var statsResponse UserStatsResponse
	err = json.Unmarshal(w.Body.Bytes(), &statsResponse)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), userStats.TotalShares, statsResponse.TotalShares)
	assert.Equal(suite.T(), userStats.ValidShares, statsResponse.ValidShares)
	
	// Test 3: Get user miners
	req = httptest.NewRequest("GET", "/api/v1/user/miners", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	
	assert.Equal(suite.T(), http.StatusOK, w.Code)
	
	var minersResponse UserMinersResponse
	err = json.Unmarshal(w.Body.Bytes(), &minersResponse)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), minersResponse.Miners, 2)
	assert.Equal(suite.T(), miners[0].Name, minersResponse.Miners[0].Name)
	
	// Verify all mock expectations were met
	suite.mockAuth.AssertExpectations(suite.T())
	suite.mockPool.AssertExpectations(suite.T())
	suite.mockUser.AssertExpectations(suite.T())
}

// TestMFAWorkflow tests the complete MFA setup and verification workflow
func (suite *E2ETestSuite) TestMFAWorkflow() {
	userID := int64(123)
	token := "valid-jwt-token"
	
	// Setup mock expectations for authentication
	claims := &JWTClaims{
		UserID:    userID,
		Username:  "testuser",
		Email:     "test@example.com",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	suite.mockAuth.On("ValidateJWT", token).Return(claims, nil)
	
	// Setup mock expectations for MFA setup
	mfaSetup := &MFASetupResponse{
		Secret:      "JBSWY3DPEHPK3PXP",
		QRCodeURL:   "otpauth://totp/ChimeraPool:testuser?secret=JBSWY3DPEHPK3PXP&issuer=ChimeraPool",
		BackupCodes: []string{"12345678", "87654321", "11223344", "44332211", "55667788"},
	}
	suite.mockUser.On("SetupMFA", userID).Return(mfaSetup, nil)
	
	// Setup mock expectations for MFA verification
	suite.mockUser.On("VerifyMFA", userID, "123456").Return(true, nil)
	
	// Test 1: Setup MFA
	req := httptest.NewRequest("POST", "/api/v1/user/mfa/setup", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	
	assert.Equal(suite.T(), http.StatusOK, w.Code)
	
	var setupResponse MFASetupResponse
	err := json.Unmarshal(w.Body.Bytes(), &setupResponse)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), mfaSetup.Secret, setupResponse.Secret)
	assert.Equal(suite.T(), mfaSetup.QRCodeURL, setupResponse.QRCodeURL)
	assert.Len(suite.T(), setupResponse.BackupCodes, 5)
	
	// Test 2: Verify MFA
	verifyRequest := VerifyMFARequest{Code: "123456"}
	body, _ := json.Marshal(verifyRequest)
	
	req = httptest.NewRequest("POST", "/api/v1/user/mfa/verify", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	
	assert.Equal(suite.T(), http.StatusOK, w.Code)
	
	var verifyResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &verifyResponse)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), true, verifyResponse["verified"])
	assert.Equal(suite.T(), "MFA enabled successfully", verifyResponse["message"])
	
	// Verify all mock expectations were met
	suite.mockAuth.AssertExpectations(suite.T())
	suite.mockUser.AssertExpectations(suite.T())
}

// TestPoolStatisticsWorkflow tests the pool statistics endpoints
func (suite *E2ETestSuite) TestPoolStatisticsWorkflow() {
	// Setup mock expectations for pool stats
	poolStats := &PoolStats{
		TotalHashrate:     1000000.0,
		ConnectedMiners:   150,
		TotalShares:       50000,
		ValidShares:       49500,
		BlocksFound:       25,
		LastBlockTime:     time.Now().Add(-10 * time.Minute),
		NetworkHashrate:   50000000.0,
		NetworkDifficulty: 1000000.0,
		PoolFee:           1.0,
	}
	suite.mockPool.On("GetPoolStats").Return(poolStats, nil)
	
	// Setup mock expectations for real-time stats
	realTimeStats := &RealTimeStats{
		CurrentHashrate:   1500000.0,
		AverageHashrate:   1200000.0,
		ActiveMiners:      175,
		SharesPerSecond:   25.5,
		LastBlockFound:    time.Now().Add(-5 * time.Minute),
		NetworkDifficulty: 1500000.0,
		PoolEfficiency:    99.2,
	}
	suite.mockPool.On("GetRealTimeStats").Return(realTimeStats, nil)
	
	// Setup mock expectations for block metrics
	blockMetrics := &BlockMetrics{
		TotalBlocks:       50,
		BlocksLast24h:     12,
		BlocksLast7d:      85,
		AverageBlockTime:  time.Duration(30 * time.Minute),
		LastBlockReward:   6.25,
		TotalRewards:      312.5,
		OrphanBlocks:      2,
		OrphanRate:        4.0,
	}
	suite.mockPool.On("GetBlockMetrics").Return(blockMetrics, nil)
	
	// Test 1: Get pool stats (public endpoint)
	req := httptest.NewRequest("GET", "/api/v1/pool/stats", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	
	assert.Equal(suite.T(), http.StatusOK, w.Code)
	
	var statsResponse PoolStatsResponse
	err := json.Unmarshal(w.Body.Bytes(), &statsResponse)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), poolStats.TotalHashrate, statsResponse.TotalHashrate)
	assert.Equal(suite.T(), poolStats.ConnectedMiners, statsResponse.ConnectedMiners)
	
	// Test 2: Get real-time stats (public endpoint)
	req = httptest.NewRequest("GET", "/api/v1/pool/realtime", nil)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	
	assert.Equal(suite.T(), http.StatusOK, w.Code)
	
	var realtimeResponse RealTimeStatsResponse
	err = json.Unmarshal(w.Body.Bytes(), &realtimeResponse)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), realTimeStats.CurrentHashrate, realtimeResponse.CurrentHashrate)
	assert.Equal(suite.T(), realTimeStats.ActiveMiners, realtimeResponse.ActiveMiners)
	
	// Test 3: Get block metrics (public endpoint)
	req = httptest.NewRequest("GET", "/api/v1/pool/blocks", nil)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	
	assert.Equal(suite.T(), http.StatusOK, w.Code)
	
	var blockResponse BlockMetricsResponse
	err = json.Unmarshal(w.Body.Bytes(), &blockResponse)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), blockMetrics.TotalBlocks, blockResponse.TotalBlocks)
	assert.Equal(suite.T(), blockMetrics.BlocksLast24h, blockResponse.BlocksLast24h)
	
	// Verify all mock expectations were met
	suite.mockPool.AssertExpectations(suite.T())
}

// TestSecurityAndErrorHandling tests security features and error handling
func (suite *E2ETestSuite) TestSecurityAndErrorHandling() {
	// Test 1: Access protected endpoint without token
	req := httptest.NewRequest("GET", "/api/v1/user/profile", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	
	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
	
	var errorResponse ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "missing_token", errorResponse.Error)
	
	// Test 2: Access protected endpoint with invalid token
	suite.mockAuth.On("ValidateJWT", "invalid-token").Return(nil, assert.AnError)
	
	req = httptest.NewRequest("GET", "/api/v1/user/profile", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	
	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
	
	err = json.Unmarshal(w.Body.Bytes(), &errorResponse)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "invalid_token", errorResponse.Error)
	
	// Test 3: Service error handling
	suite.mockPool.On("GetPoolStats").Return(nil, assert.AnError)
	
	req = httptest.NewRequest("GET", "/api/v1/pool/stats", nil)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	
	assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)
	
	err = json.Unmarshal(w.Body.Bytes(), &errorResponse)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "internal_error", errorResponse.Error)
	assert.Contains(suite.T(), errorResponse.Message, "Failed to get pool statistics")
	
	// Verify all mock expectations were met
	suite.mockAuth.AssertExpectations(suite.T())
	suite.mockPool.AssertExpectations(suite.T())
}

// TestHealthCheck tests the health check endpoint
func (suite *E2ETestSuite) TestHealthCheck() {
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	
	assert.Equal(suite.T(), http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	
	assert.Equal(suite.T(), "healthy", response["status"])
	assert.Equal(suite.T(), "chimera-pool-api", response["service"])
	assert.NotEmpty(suite.T(), response["timestamp"])
}