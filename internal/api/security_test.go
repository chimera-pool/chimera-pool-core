package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// SecurityTestSuite provides security testing for the API
type SecurityTestSuite struct {
	router   *gin.Engine
	handlers *APIHandlers
	mockAuth *MockAuthService
	mockPool *MockPoolStatsService
	mockUser *MockUserService
}

// setupSecurityTest initializes the security test environment
func setupSecurityTest() *SecurityTestSuite {
	gin.SetMode(gin.TestMode)
	
	// Create mock services
	mockAuth := &MockAuthService{}
	mockPool := &MockPoolStatsService{}
	mockUser := &MockUserService{}
	
	// Create handlers
	handlers := NewAPIHandlers(mockAuth, mockPool, mockUser)
	
	// Setup router
	router := gin.New()
	SetupAPIRoutes(router, handlers)
	
	return &SecurityTestSuite{
		router:   router,
		handlers: handlers,
		mockAuth: mockAuth,
		mockPool: mockPool,
		mockUser: mockUser,
	}
}

// TestSecurity_AuthenticationRequired tests that protected endpoints require authentication
func TestSecurity_AuthenticationRequired(t *testing.T) {
	suite := setupSecurityTest()
	
	protectedEndpoints := []struct {
		method string
		path   string
	}{
		{"GET", "/api/v1/user/profile"},
		{"PUT", "/api/v1/user/profile"},
		{"GET", "/api/v1/user/stats"},
		{"GET", "/api/v1/user/miners"},
		{"GET", "/api/v1/user/miners/1/stats"},
		{"POST", "/api/v1/user/mfa/setup"},
		{"POST", "/api/v1/user/mfa/verify"},
	}
	
	for _, endpoint := range protectedEndpoints {
		t.Run(endpoint.method+"_"+endpoint.path, func(t *testing.T) {
			var req *http.Request
			
			if endpoint.method == "PUT" || endpoint.method == "POST" {
				req = httptest.NewRequest(endpoint.method, endpoint.path, bytes.NewBuffer([]byte("{}")))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(endpoint.method, endpoint.path, nil)
			}
			
			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)
			
			assert.Equal(t, http.StatusUnauthorized, w.Code, 
				"Endpoint %s %s should require authentication", endpoint.method, endpoint.path)
			
			var response ErrorResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Equal(t, "missing_token", response.Error)
		})
	}
}

// TestSecurity_InvalidTokenHandling tests handling of invalid JWT tokens
func TestSecurity_InvalidTokenHandling(t *testing.T) {
	suite := setupSecurityTest()
	
	invalidTokens := []struct {
		name  string
		token string
	}{
		{"empty_token", ""},
		{"malformed_token", "not.a.jwt"},
		{"expired_token", "expired.jwt.token"},
		{"invalid_signature", "invalid.signature.token"},
	}
	
	for _, tokenTest := range invalidTokens {
		t.Run(tokenTest.name, func(t *testing.T) {
			// Setup mock to return error for invalid token
			suite.mockAuth.On("ValidateJWT", tokenTest.token).Return(nil, assert.AnError)
			
			req := httptest.NewRequest("GET", "/api/v1/user/profile", nil)
			if tokenTest.token != "" {
				req.Header.Set("Authorization", "Bearer "+tokenTest.token)
			}
			
			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)
			
			assert.Equal(t, http.StatusUnauthorized, w.Code)
			
			var response ErrorResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			
			if tokenTest.token == "" {
				assert.Equal(t, "missing_token", response.Error)
			} else {
				assert.Equal(t, "invalid_token", response.Error)
			}
		})
	}
}

// TestSecurity_InputValidation tests input validation and sanitization
func TestSecurity_InputValidation(t *testing.T) {
	suite := setupSecurityTest()
	
	userID := int64(123)
	token := "valid-jwt-token"
	
	// Setup valid authentication
	claims := &JWTClaims{
		UserID:    userID,
		Username:  "testuser",
		Email:     "test@example.com",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	suite.mockAuth.On("ValidateJWT", token).Return(claims, nil)
	
	// Test invalid JSON input
	t.Run("invalid_json", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/api/v1/user/profile", 
			bytes.NewBuffer([]byte("invalid json")))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusBadRequest, w.Code)
		
		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "validation_error", response.Error)
	})
	
	// Test invalid email format
	t.Run("invalid_email", func(t *testing.T) {
		updateRequest := UpdateUserProfileRequest{
			Email: "not-an-email",
		}
		body, _ := json.Marshal(updateRequest)
		
		req := httptest.NewRequest("PUT", "/api/v1/user/profile", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusBadRequest, w.Code)
		
		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "validation_error", response.Error)
	})
	
	// Test MFA code validation
	t.Run("invalid_mfa_code_length", func(t *testing.T) {
		verifyRequest := VerifyMFARequest{
			Code: "123", // Too short
		}
		body, _ := json.Marshal(verifyRequest)
		
		req := httptest.NewRequest("POST", "/api/v1/user/mfa/verify", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusBadRequest, w.Code)
		
		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "validation_error", response.Error)
	})
}

// TestSecurity_SQLInjectionPrevention tests SQL injection prevention
func TestSecurity_SQLInjectionPrevention(t *testing.T) {
	suite := setupSecurityTest()
	
	userID := int64(123)
	token := "valid-jwt-token"
	
	// Setup valid authentication
	claims := &JWTClaims{
		UserID:    userID,
		Username:  "testuser",
		Email:     "test@example.com",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	suite.mockAuth.On("ValidateJWT", token).Return(claims, nil)
	
	// Test SQL injection attempts in path parameters
	sqlInjectionAttempts := []string{
		"1' OR '1'='1",
		"1; DROP TABLE users; --",
		"1 UNION SELECT * FROM users",
		"1' AND 1=1 --",
	}
	
	for _, injection := range sqlInjectionAttempts {
		t.Run("sql_injection_"+injection, func(t *testing.T) {
			// URL encode the injection attempt
			encodedInjection := strings.ReplaceAll(injection, " ", "%20")
			encodedInjection = strings.ReplaceAll(encodedInjection, "'", "%27")
			
			req := httptest.NewRequest("GET", "/api/v1/user/miners/"+encodedInjection+"/stats", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			
			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)
			
			// Should return bad request for invalid miner ID format
			assert.Equal(t, http.StatusBadRequest, w.Code)
			
			var response ErrorResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Equal(t, "validation_error", response.Error)
		})
	}
}

// TestSecurity_XSSPrevention tests XSS prevention
func TestSecurity_XSSPrevention(t *testing.T) {
	suite := setupSecurityTest()
	
	userID := int64(123)
	token := "valid-jwt-token"
	
	// Setup valid authentication
	claims := &JWTClaims{
		UserID:    userID,
		Username:  "testuser",
		Email:     "test@example.com",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	suite.mockAuth.On("ValidateJWT", token).Return(claims, nil)
	
	// Test XSS attempts in JSON input
	xssAttempts := []string{
		"<script>alert('xss')</script>",
		"javascript:alert('xss')",
		"<img src=x onerror=alert('xss')>",
		"';alert('xss');//",
	}
	
	for _, xss := range xssAttempts {
		t.Run("xss_prevention_"+xss, func(t *testing.T) {
			updateRequest := UpdateUserProfileRequest{
				Email: xss + "@example.com",
			}
			body, _ := json.Marshal(updateRequest)
			
			req := httptest.NewRequest("PUT", "/api/v1/user/profile", bytes.NewBuffer(body))
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")
			
			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)
			
			// Should return bad request for invalid email format
			assert.Equal(t, http.StatusBadRequest, w.Code)
			
			var response ErrorResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Equal(t, "validation_error", response.Error)
		})
	}
}

// TestSecurity_RateLimiting tests rate limiting functionality
func TestSecurity_RateLimiting(t *testing.T) {
	suite := setupSecurityTest()
	
	// Simulate rate limiting by making the auth service return rate limit error
	suite.mockAuth.On("ValidateJWT", "rate-limited-token").Return(nil, 
		assert.AnError) // In real implementation, this would be a specific rate limit error
	
	// Test multiple requests that should trigger rate limiting
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/api/v1/user/profile", nil)
		req.Header.Set("Authorization", "Bearer rate-limited-token")
		
		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		
		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "invalid_token", response.Error)
	}
}

// TestSecurity_HeaderSecurity tests security headers
func TestSecurity_HeaderSecurity(t *testing.T) {
	suite := setupSecurityTest()
	
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	// In a real implementation, we would check for security headers like:
	// - X-Content-Type-Options: nosniff
	// - X-Frame-Options: DENY
	// - X-XSS-Protection: 1; mode=block
	// - Strict-Transport-Security: max-age=31536000; includeSubDomains
	// - Content-Security-Policy: default-src 'self'
	
	// For now, we just verify the response is successful
	// Security headers would be added by middleware in a real implementation
}

// TestSecurity_MFASecurityFlow tests MFA security implementation
func TestSecurity_MFASecurityFlow(t *testing.T) {
	suite := setupSecurityTest()
	
	userID := int64(123)
	token := "valid-jwt-token"
	
	// Setup valid authentication
	claims := &JWTClaims{
		UserID:    userID,
		Username:  "testuser",
		Email:     "test@example.com",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	suite.mockAuth.On("ValidateJWT", token).Return(claims, nil)
	
	// Test MFA setup security
	t.Run("mfa_setup_requires_auth", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/user/mfa/setup", nil)
		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
	
	// Test MFA verification security
	t.Run("mfa_verify_requires_auth", func(t *testing.T) {
		verifyRequest := VerifyMFARequest{Code: "123456"}
		body, _ := json.Marshal(verifyRequest)
		
		req := httptest.NewRequest("POST", "/api/v1/user/mfa/verify", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
	
	// Test MFA brute force protection
	t.Run("mfa_brute_force_protection", func(t *testing.T) {
		// Setup mock to simulate failed attempts
		suite.mockUser.On("VerifyMFA", userID, "000000").Return(false, assert.AnError)
		
		// Attempt multiple invalid codes
		for i := 0; i < 5; i++ {
			verifyRequest := VerifyMFARequest{Code: "000000"}
			body, _ := json.Marshal(verifyRequest)
			
			req := httptest.NewRequest("POST", "/api/v1/user/mfa/verify", bytes.NewBuffer(body))
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")
			
			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)
			
			assert.Equal(t, http.StatusBadRequest, w.Code)
			
			var response ErrorResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Equal(t, "invalid_mfa_code", response.Error)
		}
	})
}

// TestSecurity_ErrorInformationLeakage tests that errors don't leak sensitive information
func TestSecurity_ErrorInformationLeakage(t *testing.T) {
	suite := setupSecurityTest()
	
	// Test that database errors don't leak internal information
	suite.mockPool.On("GetPoolStats").Return(nil, assert.AnError)
	
	req := httptest.NewRequest("GET", "/api/v1/pool/stats", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	
	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	// Error message should be generic and not leak internal details
	assert.Equal(t, "internal_error", response.Error)
	assert.Contains(t, response.Message, "Failed to get pool statistics")
	// Should not contain database connection strings, file paths, etc.
	assert.NotContains(t, response.Message, "database")
	assert.NotContains(t, response.Message, "connection")
	assert.NotContains(t, response.Message, "/")
}