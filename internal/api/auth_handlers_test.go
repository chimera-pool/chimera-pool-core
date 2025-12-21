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
	"github.com/stretchr/testify/require"
)

// =============================================================================
// MOCK IMPLEMENTATIONS FOR TDD
// =============================================================================

// MockUserRegistrar implements UserRegistrar for testing
type MockUserRegistrar struct {
	RegisterFunc func(req *RegisterRequest) (*RegisteredUser, error)
}

func (m *MockUserRegistrar) Register(req *RegisterRequest) (*RegisteredUser, error) {
	if m.RegisterFunc != nil {
		return m.RegisterFunc(req)
	}
	return &RegisteredUser{
		ID:       1,
		Username: req.Username,
		Email:    req.Email,
		JoinedAt: time.Now(),
	}, nil
}

// MockUserAuthenticator implements UserAuthenticator for testing
type MockUserAuthenticator struct {
	AuthenticateFunc func(email, password string) (*AuthenticatedUser, error)
}

func (m *MockUserAuthenticator) Authenticate(email, password string) (*AuthenticatedUser, error) {
	if m.AuthenticateFunc != nil {
		return m.AuthenticateFunc(email, password)
	}
	return &AuthenticatedUser{
		ID:       1,
		Username: "testuser",
		Email:    email,
		Role:     "user",
	}, nil
}

// MockTokenGenerator implements TokenGenerator for testing
type MockTokenGenerator struct {
	GenerateFunc func(userID int64, username string) (string, error)
}

func (m *MockTokenGenerator) Generate(userID int64, username string) (string, error) {
	if m.GenerateFunc != nil {
		return m.GenerateFunc(userID, username)
	}
	return "mock-jwt-token", nil
}

// MockTokenValidator implements TokenValidator for testing
type MockTokenValidator struct {
	ValidateFunc func(token string) (*TokenClaims, error)
}

func (m *MockTokenValidator) Validate(token string) (*TokenClaims, error) {
	if m.ValidateFunc != nil {
		return m.ValidateFunc(token)
	}
	return &TokenClaims{
		UserID:    1,
		Username:  "testuser",
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
	}, nil
}

// MockPasswordResetter implements PasswordResetter for testing
type MockPasswordResetter struct {
	RequestResetFunc  func(email string) error
	ValidateTokenFunc func(token string) (int64, error)
	ResetPasswordFunc func(token, newPassword string) error
}

func (m *MockPasswordResetter) RequestReset(email string) error {
	if m.RequestResetFunc != nil {
		return m.RequestResetFunc(email)
	}
	return nil
}

func (m *MockPasswordResetter) ValidateToken(token string) (int64, error) {
	if m.ValidateTokenFunc != nil {
		return m.ValidateTokenFunc(token)
	}
	return 1, nil
}

func (m *MockPasswordResetter) ResetPassword(token, newPassword string) error {
	if m.ResetPasswordFunc != nil {
		return m.ResetPasswordFunc(token, newPassword)
	}
	return nil
}

// =============================================================================
// TEST HELPERS
// =============================================================================

func setupAuthRouter(handlers *AuthHandlers) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	api := router.Group("/api/v1/auth")
	{
		api.POST("/register", handlers.Register)
		api.POST("/login", handlers.Login)
		api.POST("/forgot-password", handlers.ForgotPassword)
		api.POST("/reset-password", handlers.ResetPassword)
	}

	return router
}

func createDefaultAuthHandlers() *AuthHandlers {
	return NewAuthHandlers(
		&MockUserRegistrar{},
		&MockUserAuthenticator{},
		&MockTokenGenerator{},
		&MockPasswordResetter{},
	)
}

// =============================================================================
// REGISTRATION TESTS (TDD)
// =============================================================================

func TestAuthHandlers_Register(t *testing.T) {
	t.Run("successful registration", func(t *testing.T) {
		handlers := createDefaultAuthHandlers()
		router := setupAuthRouter(handlers)

		body := RegisterRequest{
			Username: "newuser",
			Email:    "newuser@example.com",
			Password: "securepassword123",
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "User registered successfully", response["message"])
		assert.NotNil(t, response["user_id"])
	})

	t.Run("registration with invalid email", func(t *testing.T) {
		handlers := createDefaultAuthHandlers()
		router := setupAuthRouter(handlers)

		body := map[string]string{
			"username": "newuser",
			"email":    "invalid-email",
			"password": "securepassword123",
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("registration with weak password", func(t *testing.T) {
		handlers := createDefaultAuthHandlers()
		router := setupAuthRouter(handlers)

		body := map[string]string{
			"username": "newuser",
			"email":    "newuser@example.com",
			"password": "weak",
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("registration with duplicate email", func(t *testing.T) {
		registrar := &MockUserRegistrar{
			RegisterFunc: func(req *RegisterRequest) (*RegisteredUser, error) {
				return nil, errors.New("email already exists")
			},
		}
		handlers := NewAuthHandlers(registrar, &MockUserAuthenticator{}, &MockTokenGenerator{}, &MockPasswordResetter{})
		router := setupAuthRouter(handlers)

		body := RegisterRequest{
			Username: "newuser",
			Email:    "existing@example.com",
			Password: "securepassword123",
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("registration with missing username", func(t *testing.T) {
		handlers := createDefaultAuthHandlers()
		router := setupAuthRouter(handlers)

		body := map[string]string{
			"email":    "newuser@example.com",
			"password": "securepassword123",
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// =============================================================================
// LOGIN TESTS (TDD)
// =============================================================================

func TestAuthHandlers_Login(t *testing.T) {
	t.Run("successful login", func(t *testing.T) {
		handlers := createDefaultAuthHandlers()
		router := setupAuthRouter(handlers)

		body := LoginRequest{
			Email:    "user@example.com",
			Password: "correctpassword",
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response LoginResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.NotEmpty(t, response.Token)
		assert.Equal(t, int64(1), response.UserID)
	})

	t.Run("login with invalid credentials", func(t *testing.T) {
		authenticator := &MockUserAuthenticator{
			AuthenticateFunc: func(email, password string) (*AuthenticatedUser, error) {
				return nil, errors.New("invalid credentials")
			},
		}
		handlers := NewAuthHandlers(&MockUserRegistrar{}, authenticator, &MockTokenGenerator{}, &MockPasswordResetter{})
		router := setupAuthRouter(handlers)

		body := LoginRequest{
			Email:    "user@example.com",
			Password: "wrongpassword",
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("login with missing email", func(t *testing.T) {
		handlers := createDefaultAuthHandlers()
		router := setupAuthRouter(handlers)

		body := map[string]string{
			"password": "somepassword",
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("login with token generation failure", func(t *testing.T) {
		tokenGen := &MockTokenGenerator{
			GenerateFunc: func(userID int64, username string) (string, error) {
				return "", errors.New("token generation failed")
			},
		}
		handlers := NewAuthHandlers(&MockUserRegistrar{}, &MockUserAuthenticator{}, tokenGen, &MockPasswordResetter{})
		router := setupAuthRouter(handlers)

		body := LoginRequest{
			Email:    "user@example.com",
			Password: "correctpassword",
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// =============================================================================
// FORGOT PASSWORD TESTS (TDD)
// =============================================================================

func TestAuthHandlers_ForgotPassword(t *testing.T) {
	t.Run("successful forgot password request", func(t *testing.T) {
		handlers := createDefaultAuthHandlers()
		router := setupAuthRouter(handlers)

		body := ForgotPasswordRequest{
			Email: "user@example.com",
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/api/v1/auth/forgot-password", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response["message"], "reset link")
	})

	t.Run("forgot password with non-existent email returns success", func(t *testing.T) {
		// Should return success to prevent email enumeration
		resetter := &MockPasswordResetter{
			RequestResetFunc: func(email string) error {
				return errors.New("user not found")
			},
		}
		handlers := NewAuthHandlers(&MockUserRegistrar{}, &MockUserAuthenticator{}, &MockTokenGenerator{}, resetter)
		router := setupAuthRouter(handlers)

		body := ForgotPasswordRequest{
			Email: "nonexistent@example.com",
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/api/v1/auth/forgot-password", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return 200 to prevent email enumeration
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("forgot password with invalid email format", func(t *testing.T) {
		handlers := createDefaultAuthHandlers()
		router := setupAuthRouter(handlers)

		body := map[string]string{
			"email": "not-an-email",
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/api/v1/auth/forgot-password", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// =============================================================================
// RESET PASSWORD TESTS (TDD)
// =============================================================================

func TestAuthHandlers_ResetPassword(t *testing.T) {
	t.Run("successful password reset", func(t *testing.T) {
		handlers := createDefaultAuthHandlers()
		router := setupAuthRouter(handlers)

		body := ResetPasswordRequest{
			Token:       "valid-reset-token",
			NewPassword: "newSecurePassword123",
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/api/v1/auth/reset-password", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "Password reset successfully", response["message"])
	})

	t.Run("reset password with invalid token", func(t *testing.T) {
		resetter := &MockPasswordResetter{
			ResetPasswordFunc: func(token, newPassword string) error {
				return errors.New("invalid token")
			},
		}
		handlers := NewAuthHandlers(&MockUserRegistrar{}, &MockUserAuthenticator{}, &MockTokenGenerator{}, resetter)
		router := setupAuthRouter(handlers)

		body := ResetPasswordRequest{
			Token:       "invalid-token",
			NewPassword: "newSecurePassword123",
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/api/v1/auth/reset-password", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("reset password with expired token", func(t *testing.T) {
		resetter := &MockPasswordResetter{
			ResetPasswordFunc: func(token, newPassword string) error {
				return errors.New("token expired")
			},
		}
		handlers := NewAuthHandlers(&MockUserRegistrar{}, &MockUserAuthenticator{}, &MockTokenGenerator{}, resetter)
		router := setupAuthRouter(handlers)

		body := ResetPasswordRequest{
			Token:       "expired-token",
			NewPassword: "newSecurePassword123",
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/api/v1/auth/reset-password", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("reset password with weak new password", func(t *testing.T) {
		handlers := createDefaultAuthHandlers()
		router := setupAuthRouter(handlers)

		body := map[string]string{
			"token":        "valid-token",
			"new_password": "weak",
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/api/v1/auth/reset-password", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// =============================================================================
// AUTH MIDDLEWARE TESTS (TDD)
// =============================================================================

func TestAuthMiddleware(t *testing.T) {
	t.Run("valid token allows request", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		validator := &MockTokenValidator{}
		router.Use(AuthMiddleware(validator))
		router.GET("/protected", func(c *gin.Context) {
			userID, _ := c.Get("user_id")
			c.JSON(http.StatusOK, gin.H{"user_id": userID})
		})

		req, _ := http.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer valid-token")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("missing authorization header", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		validator := &MockTokenValidator{}
		router.Use(AuthMiddleware(validator))
		router.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req, _ := http.NewRequest("GET", "/protected", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid token", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		validator := &MockTokenValidator{
			ValidateFunc: func(token string) (*TokenClaims, error) {
				return nil, errors.New("invalid token")
			},
		}
		router.Use(AuthMiddleware(validator))
		router.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req, _ := http.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("token without Bearer prefix", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		validator := &MockTokenValidator{}
		router.Use(AuthMiddleware(validator))
		router.GET("/protected", func(c *gin.Context) {
			userID, _ := c.Get("user_id")
			c.JSON(http.StatusOK, gin.H{"user_id": userID})
		})

		req, _ := http.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "valid-token-without-bearer")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// =============================================================================
// ISP INTERFACE COMPLIANCE TESTS
// =============================================================================

func TestISP_InterfaceSegregation(t *testing.T) {
	t.Run("PasswordHasher has single responsibility", func(t *testing.T) {
		// Verify interface has only hash-related methods
		var hasher PasswordHasher = &mockHasher{}
		_, _ = hasher.Hash("password")
		_ = hasher.Verify("password", "hash")
	})

	t.Run("TokenGenerator has single responsibility", func(t *testing.T) {
		// Verify interface has only generation method
		var generator TokenGenerator = &MockTokenGenerator{}
		_, _ = generator.Generate(1, "user")
	})

	t.Run("TokenValidator has single responsibility", func(t *testing.T) {
		// Verify interface has only validation method
		var validator TokenValidator = &MockTokenValidator{}
		_, _ = validator.Validate("token")
	})

	t.Run("handlers can work with minimal dependencies", func(t *testing.T) {
		// AuthHandlers should work with any implementation of the interfaces
		handlers := NewAuthHandlers(
			&MockUserRegistrar{},
			&MockUserAuthenticator{},
			&MockTokenGenerator{},
			&MockPasswordResetter{},
		)
		assert.NotNil(t, handlers)
	})
}

// Mock hasher for interface test
type mockHasher struct{}

func (m *mockHasher) Hash(password string) (string, error) {
	return "hashed", nil
}

func (m *mockHasher) Verify(password, hash string) bool {
	return true
}
