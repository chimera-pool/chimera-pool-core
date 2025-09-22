package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestRouter creates a test router with auth routes
func setupTestRouter() (*gin.Engine, *AuthService) {
	gin.SetMode(gin.TestMode)
	
	mockRepo := NewMockUserRepository()
	authService := NewAuthService(mockRepo, "test-secret-key")
	
	router := gin.New()
	SetupAuthRoutes(router, authService)
	
	return router, authService
}

// TestRegisterHandler tests the registration endpoint
func TestRegisterHandler(t *testing.T) {
	router, _ := setupTestRouter()
	
	tests := []struct {
		name           string
		requestBody    RegisterRequest
		expectedStatus int
		expectToken    bool
	}{
		{
			name: "valid registration",
			requestBody: RegisterRequest{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "SecurePass123!",
			},
			expectedStatus: http.StatusCreated,
			expectToken:    true,
		},
		{
			name: "missing username",
			requestBody: RegisterRequest{
				Email:    "test@example.com",
				Password: "SecurePass123!",
			},
			expectedStatus: http.StatusBadRequest,
			expectToken:    false,
		},
		{
			name: "invalid email",
			requestBody: RegisterRequest{
				Username: "testuser",
				Email:    "invalid-email",
				Password: "SecurePass123!",
			},
			expectedStatus: http.StatusBadRequest,
			expectToken:    false,
		},
		{
			name: "weak password",
			requestBody: RegisterRequest{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "123",
			},
			expectedStatus: http.StatusBadRequest,
			expectToken:    false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonBody, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			assert.Equal(t, tt.expectedStatus, w.Code)
			
			if tt.expectToken {
				var response AuthResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				
				assert.NotNil(t, response.User)
				assert.NotEmpty(t, response.Token)
				assert.Equal(t, tt.requestBody.Username, response.User.Username)
				assert.Equal(t, tt.requestBody.Email, response.User.Email)
			} else {
				var response ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				
				assert.NotEmpty(t, response.Error)
				assert.NotEmpty(t, response.Message)
			}
		})
	}
}

// TestLoginHandler tests the login endpoint
func TestLoginHandler(t *testing.T) {
	router, authService := setupTestRouter()
	
	// First register a user
	_, err := authService.RegisterUser("testuser", "test@example.com", "SecurePass123!")
	require.NoError(t, err)
	
	tests := []struct {
		name           string
		requestBody    LoginRequest
		expectedStatus int
		expectToken    bool
	}{
		{
			name: "valid login",
			requestBody: LoginRequest{
				Username: "testuser",
				Password: "SecurePass123!",
			},
			expectedStatus: http.StatusOK,
			expectToken:    true,
		},
		{
			name: "invalid username",
			requestBody: LoginRequest{
				Username: "nonexistent",
				Password: "SecurePass123!",
			},
			expectedStatus: http.StatusUnauthorized,
			expectToken:    false,
		},
		{
			name: "invalid password",
			requestBody: LoginRequest{
				Username: "testuser",
				Password: "wrongpassword",
			},
			expectedStatus: http.StatusUnauthorized,
			expectToken:    false,
		},
		{
			name: "missing username",
			requestBody: LoginRequest{
				Password: "SecurePass123!",
			},
			expectedStatus: http.StatusBadRequest,
			expectToken:    false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonBody, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			assert.Equal(t, tt.expectedStatus, w.Code)
			
			if tt.expectToken {
				var response AuthResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				
				assert.NotNil(t, response.User)
				assert.NotEmpty(t, response.Token)
				assert.Equal(t, tt.requestBody.Username, response.User.Username)
			} else {
				var response ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				
				assert.NotEmpty(t, response.Error)
				assert.NotEmpty(t, response.Message)
			}
		})
	}
}

// TestValidateTokenHandler tests the token validation endpoint
func TestValidateTokenHandler(t *testing.T) {
	router, authService := setupTestRouter()
	
	// Register and login to get a valid token
	user, err := authService.RegisterUser("testuser", "test@example.com", "SecurePass123!")
	require.NoError(t, err)
	
	token, err := authService.GenerateJWT(user)
	require.NoError(t, err)
	
	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
		expectValid    bool
	}{
		{
			name:           "valid token",
			authHeader:     "Bearer " + token,
			expectedStatus: http.StatusOK,
			expectValid:    true,
		},
		{
			name:           "valid token without Bearer prefix",
			authHeader:     token,
			expectedStatus: http.StatusOK,
			expectValid:    true,
		},
		{
			name:           "invalid token",
			authHeader:     "Bearer invalid.token.here",
			expectedStatus: http.StatusUnauthorized,
			expectValid:    false,
		},
		{
			name:           "missing token",
			authHeader:     "",
			expectedStatus: http.StatusBadRequest,
			expectValid:    false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("POST", "/api/auth/validate", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			assert.Equal(t, tt.expectedStatus, w.Code)
			
			if tt.expectValid {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				
				assert.True(t, response["valid"].(bool))
				assert.NotNil(t, response["claims"])
				assert.Equal(t, float64(user.ID), response["user_id"].(float64))
				assert.Equal(t, user.Username, response["username"].(string))
			} else {
				var response ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				
				assert.NotEmpty(t, response.Error)
				assert.NotEmpty(t, response.Message)
			}
		})
	}
}

// TestAuthMiddleware tests the authentication middleware
func TestAuthMiddleware(t *testing.T) {
	router, authService := setupTestRouter()
	
	// Register and login to get a valid token
	user, err := authService.RegisterUser("testuser", "test@example.com", "SecurePass123!")
	require.NoError(t, err)
	
	token, err := authService.GenerateJWT(user)
	require.NoError(t, err)
	
	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
	}{
		{
			name:           "valid token",
			authHeader:     "Bearer " + token,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid token",
			authHeader:     "Bearer invalid.token.here",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "missing token",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/api/user/profile", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			assert.Equal(t, tt.expectedStatus, w.Code)
			
			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				
				assert.NotNil(t, response["user"])
				userMap := response["user"].(map[string]interface{})
				assert.Equal(t, float64(user.ID), userMap["id"].(float64))
				assert.Equal(t, user.Username, userMap["username"].(string))
			}
		})
	}
}

// TestE2EAuthFlow tests the complete authentication flow via HTTP
func TestE2EAuthFlow(t *testing.T) {
	router, _ := setupTestRouter()
	
	// Step 1: Register a new user
	registerReq := RegisterRequest{
		Username: "e2euser",
		Email:    "e2e@example.com",
		Password: "E2ESecure123!",
	}
	
	jsonBody, _ := json.Marshal(registerReq)
	req, _ := http.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	require.Equal(t, http.StatusCreated, w.Code)
	
	var registerResponse AuthResponse
	err := json.Unmarshal(w.Body.Bytes(), &registerResponse)
	require.NoError(t, err)
	
	assert.NotNil(t, registerResponse.User)
	assert.NotEmpty(t, registerResponse.Token)
	
	// Step 2: Login with the registered user
	loginReq := LoginRequest{
		Username: registerReq.Username,
		Password: registerReq.Password,
	}
	
	jsonBody, _ = json.Marshal(loginReq)
	req, _ = http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	require.Equal(t, http.StatusOK, w.Code)
	
	var loginResponse AuthResponse
	err = json.Unmarshal(w.Body.Bytes(), &loginResponse)
	require.NoError(t, err)
	
	assert.Equal(t, registerResponse.User.ID, loginResponse.User.ID)
	assert.NotEmpty(t, loginResponse.Token)
	
	// Step 3: Access protected endpoint with token
	req, _ = http.NewRequest("GET", "/api/user/profile", nil)
	req.Header.Set("Authorization", "Bearer "+loginResponse.Token)
	
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	require.Equal(t, http.StatusOK, w.Code)
	
	var profileResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &profileResponse)
	require.NoError(t, err)
	
	assert.NotNil(t, profileResponse["user"])
	userMap := profileResponse["user"].(map[string]interface{})
	assert.Equal(t, registerReq.Username, userMap["username"].(string))
	assert.Equal(t, registerReq.Email, userMap["email"].(string))
	
	// Step 4: Validate token
	req, _ = http.NewRequest("POST", "/api/auth/validate", nil)
	req.Header.Set("Authorization", "Bearer "+loginResponse.Token)
	
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	require.Equal(t, http.StatusOK, w.Code)
	
	var validateResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &validateResponse)
	require.NoError(t, err)
	
	assert.True(t, validateResponse["valid"].(bool))
	assert.Equal(t, registerReq.Username, validateResponse["username"].(string))
}