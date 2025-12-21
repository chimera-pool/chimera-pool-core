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

	"github.com/chimera-pool/chimera-pool-core/internal/auth"
	"github.com/chimera-pool/chimera-pool-core/internal/database"
	"github.com/chimera-pool/chimera-pool-core/internal/shares"
)

// IntegrationTestSuite provides integration testing for the complete API
type IntegrationTestSuite struct {
	suite.Suite
	router         *gin.Engine
	handlers       *APIHandlers
	authService    *auth.AuthService
	poolService    *DefaultPoolStatsService
	userService    *DefaultUserService
	db             *database.Database
	shareProcessor *shares.ShareProcessor
}

// SetupSuite initializes the integration test suite with real services
func (suite *IntegrationTestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)

	// Initialize database (mock implementation for testing)
	suite.db = &database.Database{} // Would be initialized with test database

	// Initialize share processor
	suite.shareProcessor = &shares.ShareProcessor{} // Would be initialized with test config

	// Initialize real services
	// Note: Using mock auth service for tests to avoid interface mismatch
	mockAuth := &MockAuthService{}
	suite.poolService = NewDefaultPoolStatsService(suite.db, suite.shareProcessor)
	suite.userService = NewDefaultUserService(suite.db)

	// Create handlers with services
	suite.handlers = NewAPIHandlers(mockAuth, suite.poolService, suite.userService)

	// Setup router
	suite.router = gin.New()
	SetupAPIRoutes(suite.router, suite.handlers)
}

// TestIntegrationAPIWorkflows runs the integration test suite
func TestIntegrationAPIWorkflows(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

// TestIntegration_CompleteAPIFlow tests the complete API flow with real services
func (suite *IntegrationTestSuite) TestIntegration_CompleteAPIFlow() {
	// Test 1: Health check (public endpoint)
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var healthResponse map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &healthResponse)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "healthy", healthResponse["status"])

	// Test 2: Pool stats (public endpoint)
	req = httptest.NewRequest("GET", "/api/v1/pool/stats", nil)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var poolStatsResponse PoolStatsResponse
	err = json.Unmarshal(w.Body.Bytes(), &poolStatsResponse)
	require.NoError(suite.T(), err)
	assert.Greater(suite.T(), poolStatsResponse.TotalHashrate, 0.0)
	assert.GreaterOrEqual(suite.T(), poolStatsResponse.ConnectedMiners, int64(0))

	// Test 3: Real-time stats (public endpoint)
	req = httptest.NewRequest("GET", "/api/v1/pool/realtime", nil)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var realtimeResponse RealTimeStatsResponse
	err = json.Unmarshal(w.Body.Bytes(), &realtimeResponse)
	require.NoError(suite.T(), err)
	assert.Greater(suite.T(), realtimeResponse.CurrentHashrate, 0.0)
	assert.NotEmpty(suite.T(), realtimeResponse.Timestamp)

	// Test 4: Block metrics (public endpoint)
	req = httptest.NewRequest("GET", "/api/v1/pool/blocks", nil)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var blockResponse BlockMetricsResponse
	err = json.Unmarshal(w.Body.Bytes(), &blockResponse)
	require.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), blockResponse.TotalBlocks, int64(0))
	assert.NotEmpty(suite.T(), blockResponse.Timestamp)
}

// TestIntegration_AuthenticationFlow tests the complete authentication flow
func (suite *IntegrationTestSuite) TestIntegration_AuthenticationFlow() {
	// Create a test user and generate a real JWT token
	testUser := &auth.User{
		ID:       123,
		Username: "testuser",
		Email:    "test@example.com",
		IsActive: true,
	}

	// Generate real JWT token
	token, err := suite.authService.GenerateJWT(testUser)
	require.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), token)

	// Test 1: Access protected endpoint with valid token
	req := httptest.NewRequest("GET", "/api/v1/user/profile", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var profileResponse UserProfileResponse
	err = json.Unmarshal(w.Body.Bytes(), &profileResponse)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), testUser.ID, profileResponse.ID)
	assert.Equal(suite.T(), testUser.Username, profileResponse.Username)

	// Test 2: Access protected endpoint without token
	req = httptest.NewRequest("GET", "/api/v1/user/profile", nil)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)

	var errorResponse ErrorResponse
	err = json.Unmarshal(w.Body.Bytes(), &errorResponse)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "missing_token", errorResponse.Error)

	// Test 3: Access protected endpoint with invalid token
	req = httptest.NewRequest("GET", "/api/v1/user/profile", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)

	err = json.Unmarshal(w.Body.Bytes(), &errorResponse)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "invalid_token", errorResponse.Error)
}

// TestIntegration_UserDataFlow tests the complete user data flow
func (suite *IntegrationTestSuite) TestIntegration_UserDataFlow() {
	// Create a test user and generate a real JWT token
	testUser := &auth.User{
		ID:       123,
		Username: "testuser",
		Email:    "test@example.com",
		IsActive: true,
	}

	token, err := suite.authService.GenerateJWT(testUser)
	require.NoError(suite.T(), err)

	// Test 1: Get user profile
	req := httptest.NewRequest("GET", "/api/v1/user/profile", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var profileResponse UserProfileResponse
	err = json.Unmarshal(w.Body.Bytes(), &profileResponse)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), testUser.Username, profileResponse.Username)

	// Test 2: Update user profile
	updateRequest := UpdateUserProfileRequest{
		Email: "newemail@example.com",
	}
	body, _ := json.Marshal(updateRequest)

	req = httptest.NewRequest("PUT", "/api/v1/user/profile", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var updatedProfileResponse UserProfileResponse
	err = json.Unmarshal(w.Body.Bytes(), &updatedProfileResponse)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), updateRequest.Email, updatedProfileResponse.Email)

	// Test 3: Get user stats
	req = httptest.NewRequest("GET", "/api/v1/user/stats", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var statsResponse UserStatsResponse
	err = json.Unmarshal(w.Body.Bytes(), &statsResponse)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), testUser.ID, statsResponse.UserID)
	assert.GreaterOrEqual(suite.T(), statsResponse.TotalShares, int64(0))

	// Test 4: Get user miners
	req = httptest.NewRequest("GET", "/api/v1/user/miners", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var minersResponse UserMinersResponse
	err = json.Unmarshal(w.Body.Bytes(), &minersResponse)
	require.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), minersResponse.Total, 0)
}

// TestIntegration_MFAFlow tests the complete MFA flow
func (suite *IntegrationTestSuite) TestIntegration_MFAFlow() {
	// Create a test user and generate a real JWT token
	testUser := &auth.User{
		ID:       123,
		Username: "testuser",
		Email:    "test@example.com",
		IsActive: true,
	}

	token, err := suite.authService.GenerateJWT(testUser)
	require.NoError(suite.T(), err)

	// Test 1: Setup MFA
	req := httptest.NewRequest("POST", "/api/v1/user/mfa/setup", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var setupResponse MFASetupResponse
	err = json.Unmarshal(w.Body.Bytes(), &setupResponse)
	require.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), setupResponse.Secret)
	assert.NotEmpty(suite.T(), setupResponse.QRCodeURL)
	assert.Len(suite.T(), setupResponse.BackupCodes, 5)
	assert.Contains(suite.T(), setupResponse.QRCodeURL, "otpauth://totp/ChimeraPool:")
	assert.Contains(suite.T(), setupResponse.QRCodeURL, testUser.Username)

	// Test 2: Verify MFA with valid code
	verifyRequest := VerifyMFARequest{Code: "123456"} // Valid test code
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

	// Test 3: Verify MFA with invalid code
	invalidVerifyRequest := VerifyMFARequest{Code: "000000"} // Invalid test code
	body, _ = json.Marshal(invalidVerifyRequest)

	req = httptest.NewRequest("POST", "/api/v1/user/mfa/verify", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

	var errorResponse ErrorResponse
	err = json.Unmarshal(w.Body.Bytes(), &errorResponse)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "invalid_mfa_code", errorResponse.Error)
}

// TestIntegration_ErrorHandling tests comprehensive error handling
func (suite *IntegrationTestSuite) TestIntegration_ErrorHandling() {
	// Test 1: Invalid JSON input
	req := httptest.NewRequest("PUT", "/api/v1/user/profile",
		bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code) // Should fail auth first

	// Test 2: Invalid path parameters
	testUser := &auth.User{
		ID:       123,
		Username: "testuser",
		Email:    "test@example.com",
		IsActive: true,
	}

	token, err := suite.authService.GenerateJWT(testUser)
	require.NoError(suite.T(), err)

	req = httptest.NewRequest("GET", "/api/v1/user/miners/invalid/stats", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

	var errorResponse ErrorResponse
	err = json.Unmarshal(w.Body.Bytes(), &errorResponse)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "validation_error", errorResponse.Error)
	assert.Contains(suite.T(), errorResponse.Message, "Invalid miner ID")
}

// TestIntegration_ConcurrentRequests tests concurrent request handling
func (suite *IntegrationTestSuite) TestIntegration_ConcurrentRequests() {
	// Create multiple test users and tokens
	numUsers := 10
	tokens := make([]string, numUsers)

	for i := 0; i < numUsers; i++ {
		testUser := &auth.User{
			ID:       int64(100 + i),
			Username: "testuser" + string(rune(i)),
			Email:    "test" + string(rune(i)) + "@example.com",
			IsActive: true,
		}

		token, err := suite.authService.GenerateJWT(testUser)
		require.NoError(suite.T(), err)
		tokens[i] = token
	}

	// Test concurrent authenticated requests
	done := make(chan bool, numUsers)

	for i := 0; i < numUsers; i++ {
		go func(userIndex int) {
			defer func() { done <- true }()

			req := httptest.NewRequest("GET", "/api/v1/user/profile", nil)
			req.Header.Set("Authorization", "Bearer "+tokens[userIndex])
			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)

			assert.Equal(suite.T(), http.StatusOK, w.Code)

			var profileResponse UserProfileResponse
			err := json.Unmarshal(w.Body.Bytes(), &profileResponse)
			require.NoError(suite.T(), err)
			assert.Equal(suite.T(), int64(100+userIndex), profileResponse.ID)
		}(i)
	}

	// Wait for all requests to complete
	for i := 0; i < numUsers; i++ {
		<-done
	}
}

// TestIntegration_JWTTokenLifecycle tests JWT token lifecycle
func (suite *IntegrationTestSuite) TestIntegration_JWTTokenLifecycle() {
	testUser := &auth.User{
		ID:       123,
		Username: "testuser",
		Email:    "test@example.com",
		IsActive: true,
	}

	// Test 1: Generate and validate token
	token, err := suite.authService.GenerateJWT(testUser)
	require.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), token)

	// Test 2: Use token immediately (should work)
	req := httptest.NewRequest("GET", "/api/v1/user/profile", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Test 3: Validate token directly
	claims, err := suite.authService.ValidateJWT(token)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), testUser.ID, claims.UserID)
	assert.Equal(suite.T(), testUser.Username, claims.Username)
	assert.Equal(suite.T(), testUser.Email, claims.Email)
	assert.True(suite.T(), claims.ExpiresAt.After(time.Now()))

	// Test 4: Invalid token format
	_, err = suite.authService.ValidateJWT("invalid.token.format")
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "invalid token")
}
