package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockAuthService for testing
type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) ValidateJWT(token string) (*JWTClaims, error) {
	args := m.Called(token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*JWTClaims), args.Error(1)
}

// MockPoolStatsService for testing
type MockPoolStatsService struct {
	mock.Mock
}

func (m *MockPoolStatsService) GetPoolStats() (*PoolStats, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*PoolStats), args.Error(1)
}

func (m *MockPoolStatsService) GetMinerStats(userID int64) (*MinerStats, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*MinerStats), args.Error(1)
}

func (m *MockPoolStatsService) GetUserStats(userID int64) (*UserStats, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*UserStats), args.Error(1)
}

func (m *MockPoolStatsService) GetRealTimeStats() (*RealTimeStats, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RealTimeStats), args.Error(1)
}

func (m *MockPoolStatsService) GetBlockMetrics() (*BlockMetrics, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*BlockMetrics), args.Error(1)
}

// MockUserService for testing
type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) GetUserProfile(userID int64) (*UserProfile, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*UserProfile), args.Error(1)
}

func (m *MockUserService) UpdateUserProfile(userID int64, profile *UpdateUserProfileRequest) (*UserProfile, error) {
	args := m.Called(userID, profile)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*UserProfile), args.Error(1)
}

func (m *MockUserService) GetUserMiners(userID int64) ([]*MinerInfo, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*MinerInfo), args.Error(1)
}

func (m *MockUserService) SetupMFA(userID int64) (*MFASetupResponse, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*MFASetupResponse), args.Error(1)
}

func (m *MockUserService) VerifyMFA(userID int64, code string) (bool, error) {
	args := m.Called(userID, code)
	return args.Bool(0), args.Error(1)
}

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestPoolStatsHandler_GetPoolStats_Success(t *testing.T) {
	// Arrange
	mockPoolStats := &MockPoolStatsService{}
	handlers := NewAPIHandlers(nil, mockPoolStats, nil)
	
	expectedStats := &PoolStats{
		TotalHashrate:    1000000.0,
		ConnectedMiners:  150,
		TotalShares:      50000,
		ValidShares:      49500,
		BlocksFound:      25,
		LastBlockTime:    time.Now().Add(-10 * time.Minute),
		NetworkHashrate:  50000000.0,
		NetworkDifficulty: 1000000.0,
		PoolFee:          1.0,
	}
	
	mockPoolStats.On("GetPoolStats").Return(expectedStats, nil)
	
	router := setupTestRouter()
	router.GET("/api/pool/stats", handlers.GetPoolStats)
	
	// Act
	req := httptest.NewRequest("GET", "/api/pool/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response PoolStatsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, expectedStats.TotalHashrate, response.TotalHashrate)
	assert.Equal(t, expectedStats.ConnectedMiners, response.ConnectedMiners)
	assert.Equal(t, expectedStats.TotalShares, response.TotalShares)
	assert.Equal(t, expectedStats.ValidShares, response.ValidShares)
	assert.Equal(t, expectedStats.BlocksFound, response.BlocksFound)
	
	mockPoolStats.AssertExpectations(t)
}

func TestPoolStatsHandler_GetPoolStats_ServiceError(t *testing.T) {
	// Arrange
	mockPoolStats := &MockPoolStatsService{}
	handlers := NewAPIHandlers(nil, mockPoolStats, nil)
	
	mockPoolStats.On("GetPoolStats").Return(nil, assert.AnError)
	
	router := setupTestRouter()
	router.GET("/api/pool/stats", handlers.GetPoolStats)
	
	// Act
	req := httptest.NewRequest("GET", "/api/pool/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	
	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "internal_error", response.Error)
	assert.Contains(t, response.Message, "Failed to get pool statistics")
	
	mockPoolStats.AssertExpectations(t)
}

func TestUserHandler_GetUserProfile_Success(t *testing.T) {
	// Arrange
	mockAuth := &MockAuthService{}
	mockUser := &MockUserService{}
	handlers := NewAPIHandlers(mockAuth, nil, mockUser)
	
	userID := int64(123)
	expectedProfile := &UserProfile{
		ID:       userID,
		Username: "testuser",
		Email:    "test@example.com",
		JoinedAt: time.Now().Add(-30 * 24 * time.Hour),
		IsActive: true,
	}
	
	mockUser.On("GetUserProfile", userID).Return(expectedProfile, nil)
	
	router := setupTestRouter()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	})
	router.GET("/api/user/profile", handlers.GetUserProfile)
	
	// Act
	req := httptest.NewRequest("GET", "/api/user/profile", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response UserProfileResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, expectedProfile.ID, response.ID)
	assert.Equal(t, expectedProfile.Username, response.Username)
	assert.Equal(t, expectedProfile.Email, response.Email)
	assert.Equal(t, expectedProfile.IsActive, response.IsActive)
	
	mockUser.AssertExpectations(t)
}

func TestUserHandler_GetUserProfile_Unauthorized(t *testing.T) {
	// Arrange
	handlers := NewAPIHandlers(nil, nil, nil)
	
	router := setupTestRouter()
	router.GET("/api/user/profile", handlers.GetUserProfile)
	
	// Act
	req := httptest.NewRequest("GET", "/api/user/profile", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Assert
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	
	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "unauthorized", response.Error)
}

func TestUserHandler_UpdateUserProfile_Success(t *testing.T) {
	// Arrange
	mockUser := &MockUserService{}
	handlers := NewAPIHandlers(nil, nil, mockUser)
	
	userID := int64(123)
	updateRequest := &UpdateUserProfileRequest{
		Email: "newemail@example.com",
	}
	
	expectedProfile := &UserProfile{
		ID:       userID,
		Username: "testuser",
		Email:    updateRequest.Email,
		JoinedAt: time.Now().Add(-30 * 24 * time.Hour),
		IsActive: true,
	}
	
	mockUser.On("UpdateUserProfile", userID, updateRequest).Return(expectedProfile, nil)
	
	router := setupTestRouter()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	})
	router.PUT("/api/user/profile", handlers.UpdateUserProfile)
	
	// Act
	body, _ := json.Marshal(updateRequest)
	req := httptest.NewRequest("PUT", "/api/user/profile", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response UserProfileResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, expectedProfile.Email, response.Email)
	
	mockUser.AssertExpectations(t)
}

func TestUserHandler_GetUserStats_Success(t *testing.T) {
	// Arrange
	mockPoolStats := &MockPoolStatsService{}
	handlers := NewAPIHandlers(nil, mockPoolStats, nil)
	
	userID := int64(123)
	expectedStats := &UserStats{
		UserID:        userID,
		TotalShares:   1000,
		ValidShares:   990,
		InvalidShares: 10,
		TotalHashrate: 50000.0,
		LastShare:     time.Now().Add(-5 * time.Minute),
		Earnings:      0.05,
	}
	
	mockPoolStats.On("GetUserStats", userID).Return(expectedStats, nil)
	
	router := setupTestRouter()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	})
	router.GET("/api/user/stats", handlers.GetUserStats)
	
	// Act
	req := httptest.NewRequest("GET", "/api/user/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response UserStatsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, expectedStats.UserID, response.UserID)
	assert.Equal(t, expectedStats.TotalShares, response.TotalShares)
	assert.Equal(t, expectedStats.ValidShares, response.ValidShares)
	assert.Equal(t, expectedStats.TotalHashrate, response.TotalHashrate)
	
	mockPoolStats.AssertExpectations(t)
}

func TestUserHandler_GetUserMiners_Success(t *testing.T) {
	// Arrange
	mockUser := &MockUserService{}
	handlers := NewAPIHandlers(nil, nil, mockUser)
	
	userID := int64(123)
	expectedMiners := []*MinerInfo{
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
	
	mockUser.On("GetUserMiners", userID).Return(expectedMiners, nil)
	
	router := setupTestRouter()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	})
	router.GET("/api/user/miners", handlers.GetUserMiners)
	
	// Act
	req := httptest.NewRequest("GET", "/api/user/miners", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response UserMinersResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Len(t, response.Miners, 2)
	assert.Equal(t, expectedMiners[0].Name, response.Miners[0].Name)
	assert.Equal(t, expectedMiners[1].Name, response.Miners[1].Name)
	
	mockUser.AssertExpectations(t)
}

func TestAuthMiddleware_ValidToken_Success(t *testing.T) {
	// Arrange
	mockAuth := &MockAuthService{}
	handlers := NewAPIHandlers(mockAuth, nil, nil)
	
	token := "valid-jwt-token"
	claims := &JWTClaims{
		UserID:   123,
		Username: "testuser",
		Email:    "test@example.com",
	}
	
	mockAuth.On("ValidateJWT", token).Return(claims, nil)
	
	router := setupTestRouter()
	router.Use(handlers.AuthMiddleware())
	router.GET("/protected", func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})
	
	// Act
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, float64(123), response["user_id"])
	
	mockAuth.AssertExpectations(t)
}

func TestAuthMiddleware_InvalidToken_Unauthorized(t *testing.T) {
	// Arrange
	mockAuth := &MockAuthService{}
	handlers := NewAPIHandlers(mockAuth, nil, nil)
	
	token := "invalid-jwt-token"
	
	mockAuth.On("ValidateJWT", token).Return(nil, assert.AnError)
	
	router := setupTestRouter()
	router.Use(handlers.AuthMiddleware())
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})
	
	// Act
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Assert
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	
	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "invalid_token", response.Error)
	
	mockAuth.AssertExpectations(t)
}

func TestAuthMiddleware_MissingToken_Unauthorized(t *testing.T) {
	// Arrange
	handlers := NewAPIHandlers(nil, nil, nil)
	
	router := setupTestRouter()
	router.Use(handlers.AuthMiddleware())
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})
	
	// Act
	req := httptest.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Assert
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	
	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "missing_token", response.Error)
}

func TestHealthCheck_Success(t *testing.T) {
	// Arrange
	router := setupTestRouter()
	router.GET("/health", HealthCheck)
	
	// Act
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "healthy", response["status"])
	assert.Equal(t, "chimera-pool-api", response["service"])
	assert.NotEmpty(t, response["timestamp"])
}

// Test for real-time statistics endpoint (Requirement 7.1)
func TestPoolStatsHandler_GetRealTimeStats_Success(t *testing.T) {
	// Arrange
	mockPoolStats := &MockPoolStatsService{}
	handlers := NewAPIHandlers(nil, mockPoolStats, nil)
	
	expectedStats := &RealTimeStats{
		CurrentHashrate:   1500000.0,
		AverageHashrate:   1200000.0,
		ActiveMiners:      175,
		SharesPerSecond:   25.5,
		LastBlockFound:    time.Now().Add(-5 * time.Minute),
		NetworkDifficulty: 1500000.0,
		PoolEfficiency:    99.2,
	}
	
	mockPoolStats.On("GetRealTimeStats").Return(expectedStats, nil)
	
	router := setupTestRouter()
	router.GET("/api/pool/realtime", handlers.GetRealTimeStats)
	
	// Act
	req := httptest.NewRequest("GET", "/api/pool/realtime", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response RealTimeStatsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, expectedStats.CurrentHashrate, response.CurrentHashrate)
	assert.Equal(t, expectedStats.ActiveMiners, response.ActiveMiners)
	assert.Equal(t, expectedStats.SharesPerSecond, response.SharesPerSecond)
	assert.Equal(t, expectedStats.PoolEfficiency, response.PoolEfficiency)
	
	mockPoolStats.AssertExpectations(t)
}

func TestPoolStatsHandler_GetRealTimeStats_ServiceError(t *testing.T) {
	// Arrange
	mockPoolStats := &MockPoolStatsService{}
	handlers := NewAPIHandlers(nil, mockPoolStats, nil)
	
	mockPoolStats.On("GetRealTimeStats").Return(nil, assert.AnError)
	
	router := setupTestRouter()
	router.GET("/api/pool/realtime", handlers.GetRealTimeStats)
	
	// Act
	req := httptest.NewRequest("GET", "/api/pool/realtime", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	
	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "internal_error", response.Error)
	assert.Contains(t, response.Message, "Failed to get real-time statistics")
	
	mockPoolStats.AssertExpectations(t)
}

// Test for block discovery metrics (Requirement 7.2)
func TestPoolStatsHandler_GetBlockMetrics_Success(t *testing.T) {
	// Arrange
	mockPoolStats := &MockPoolStatsService{}
	handlers := NewAPIHandlers(nil, mockPoolStats, nil)
	
	expectedMetrics := &BlockMetrics{
		TotalBlocks:       50,
		BlocksLast24h:     12,
		BlocksLast7d:      85,
		AverageBlockTime:  time.Duration(30 * time.Minute),
		LastBlockReward:   6.25,
		TotalRewards:      312.5,
		OrphanBlocks:      2,
		OrphanRate:        4.0,
	}
	
	mockPoolStats.On("GetBlockMetrics").Return(expectedMetrics, nil)
	
	router := setupTestRouter()
	router.GET("/api/pool/blocks", handlers.GetBlockMetrics)
	
	// Act
	req := httptest.NewRequest("GET", "/api/pool/blocks", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response BlockMetricsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, expectedMetrics.TotalBlocks, response.TotalBlocks)
	assert.Equal(t, expectedMetrics.BlocksLast24h, response.BlocksLast24h)
	assert.Equal(t, expectedMetrics.LastBlockReward, response.LastBlockReward)
	assert.Equal(t, expectedMetrics.OrphanRate, response.OrphanRate)
	
	mockPoolStats.AssertExpectations(t)
}

// Test for MFA setup endpoint (Requirement 21.1)
func TestUserHandler_SetupMFA_Success(t *testing.T) {
	// Arrange
	mockAuth := &MockAuthService{}
	mockUser := &MockUserService{}
	handlers := NewAPIHandlers(mockAuth, nil, mockUser)
	
	userID := int64(123)
	expectedMFASetup := &MFASetupResponse{
		Secret:      "JBSWY3DPEHPK3PXP",
		QRCodeURL:   "otpauth://totp/ChimeraPool:testuser?secret=JBSWY3DPEHPK3PXP&issuer=ChimeraPool",
		BackupCodes: []string{"12345678", "87654321", "11223344", "44332211", "55667788"},
	}
	
	mockUser.On("SetupMFA", userID).Return(expectedMFASetup, nil)
	
	router := setupTestRouter()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	})
	router.POST("/api/user/mfa/setup", handlers.SetupMFA)
	
	// Act
	req := httptest.NewRequest("POST", "/api/user/mfa/setup", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response MFASetupResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, expectedMFASetup.Secret, response.Secret)
	assert.Equal(t, expectedMFASetup.QRCodeURL, response.QRCodeURL)
	assert.Len(t, response.BackupCodes, 5)
	
	mockUser.AssertExpectations(t)
}

func TestUserHandler_VerifyMFA_Success(t *testing.T) {
	// Arrange
	mockUser := &MockUserService{}
	handlers := NewAPIHandlers(nil, nil, mockUser)
	
	userID := int64(123)
	verifyRequest := &VerifyMFARequest{
		Code: "123456",
	}
	
	mockUser.On("VerifyMFA", userID, verifyRequest.Code).Return(true, nil)
	
	router := setupTestRouter()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	})
	router.POST("/api/user/mfa/verify", handlers.VerifyMFA)
	
	// Act
	body, _ := json.Marshal(verifyRequest)
	req := httptest.NewRequest("POST", "/api/user/mfa/verify", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, true, response["verified"])
	assert.Equal(t, "MFA enabled successfully", response["message"])
	
	mockUser.AssertExpectations(t)
}

func TestUserHandler_VerifyMFA_InvalidCode(t *testing.T) {
	// Arrange
	mockUser := &MockUserService{}
	handlers := NewAPIHandlers(nil, nil, mockUser)
	
	userID := int64(123)
	verifyRequest := &VerifyMFARequest{
		Code: "000000",
	}
	
	mockUser.On("VerifyMFA", userID, verifyRequest.Code).Return(false, errors.New("invalid code"))
	
	router := setupTestRouter()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	})
	router.POST("/api/user/mfa/verify", handlers.VerifyMFA)
	
	// Act
	body, _ := json.Marshal(verifyRequest)
	req := httptest.NewRequest("POST", "/api/user/mfa/verify", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)
	
	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "invalid_mfa_code", response.Error)
	assert.Contains(t, response.Message, "Invalid MFA code")
	
	mockUser.AssertExpectations(t)
}

// Test for rate limiting (Security requirement)
func TestAuthMiddleware_RateLimit_TooManyRequests(t *testing.T) {
	// Arrange
	mockAuth := &MockAuthService{}
	handlers := NewAPIHandlers(mockAuth, nil, nil)
	
	// Mock rate limiter to return error
	mockAuth.On("ValidateJWT", "valid-token").Return(nil, errors.New("rate limit exceeded"))
	
	router := setupTestRouter()
	router.Use(handlers.AuthMiddleware())
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})
	
	// Act
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Assert
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	
	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "invalid_token", response.Error)
	
	mockAuth.AssertExpectations(t)
}

// Test for comprehensive error handling
func TestAPIHandlers_ErrorHandling_DatabaseConnection(t *testing.T) {
	// Arrange
	mockPoolStats := &MockPoolStatsService{}
	handlers := NewAPIHandlers(nil, mockPoolStats, nil)
	
	mockPoolStats.On("GetPoolStats").Return(nil, errors.New("database connection failed"))
	
	router := setupTestRouter()
	router.GET("/api/pool/stats", handlers.GetPoolStats)
	
	// Act
	req := httptest.NewRequest("GET", "/api/pool/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	
	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "internal_error", response.Error)
	assert.Contains(t, response.Message, "Failed to get pool statistics")
	assert.Contains(t, response.Message, "database connection failed")
	
	mockPoolStats.AssertExpectations(t)
}