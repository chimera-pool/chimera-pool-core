package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUserRegistration tests user registration functionality
func TestUserRegistration(t *testing.T) {
	tests := []struct {
		name        string
		username    string
		email       string
		password    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid registration",
			username:    "testuser",
			email:       "test@example.com",
			password:    "SecurePass123!",
			expectError: false,
		},
		{
			name:        "empty username",
			username:    "",
			email:       "test@example.com",
			password:    "SecurePass123!",
			expectError: true,
			errorMsg:    "username is required",
		},
		{
			name:        "empty email",
			username:    "testuser",
			email:       "",
			password:    "SecurePass123!",
			expectError: true,
			errorMsg:    "email is required",
		},
		{
			name:        "invalid email format",
			username:    "testuser",
			email:       "invalid-email",
			password:    "SecurePass123!",
			expectError: true,
			errorMsg:    "invalid email format",
		},
		{
			name:        "weak password",
			username:    "testuser",
			email:       "test@example.com",
			password:    "123",
			expectError: true,
			errorMsg:    "password must be at least 8 characters long",
		},
		{
			name:        "username too short",
			username:    "ab",
			email:       "test@example.com",
			password:    "SecurePass123!",
			expectError: true,
			errorMsg:    "username must be at least 3 characters long",
		},
		{
			name:        "username too long",
			username:    "this_is_a_very_long_username_that_exceeds_fifty_chars",
			email:       "test@example.com",
			password:    "SecurePass123!",
			expectError: true,
			errorMsg:    "username must be at most 50 characters long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use mock repository for testing
			mockRepo := NewMockUserRepository()
			service := NewAuthService(mockRepo, "test-secret")

			user, err := service.RegisterUser(tt.username, tt.email, tt.password)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, tt.username, user.Username)
				assert.Equal(t, tt.email, user.Email)
				assert.NotEmpty(t, user.PasswordHash)
				assert.NotEqual(t, tt.password, user.PasswordHash) // Password should be hashed
				assert.True(t, user.IsActive)
				assert.NotZero(t, user.ID)
			}
		})
	}
}

// TestUserLogin tests user login functionality
func TestUserLogin(t *testing.T) {
	tests := []struct {
		name        string
		username    string
		password    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid login",
			username:    "testuser",
			password:    "SecurePass123!",
			expectError: false,
		},
		{
			name:        "invalid username",
			username:    "nonexistent",
			password:    "SecurePass123!",
			expectError: true,
			errorMsg:    "invalid credentials",
		},
		{
			name:        "invalid password",
			username:    "testuser",
			password:    "wrongpassword",
			expectError: true,
			errorMsg:    "invalid credentials",
		},
		{
			name:        "empty username",
			username:    "",
			password:    "SecurePass123!",
			expectError: true,
			errorMsg:    "username is required",
		},
		{
			name:        "empty password",
			username:    "testuser",
			password:    "",
			expectError: true,
			errorMsg:    "password is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use mock repository for testing
			mockRepo := NewMockUserRepository()
			service := NewAuthService(mockRepo, "test-secret")

			// First register a user for valid login test
			if tt.name == "valid login" {
				_, err := service.RegisterUser("testuser", "test@example.com", "SecurePass123!")
				require.NoError(t, err)
			}

			user, token, err := service.LoginUser(tt.username, tt.password)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, user)
				assert.Empty(t, token)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.NotEmpty(t, token)
				assert.Equal(t, tt.username, user.Username)
			}
		})
	}
}

// TestJWTTokenGeneration tests JWT token generation and validation
func TestJWTTokenGeneration(t *testing.T) {
	service := NewAuthService(nil, "test-secret-key")

	user := &User{
		ID:       1,
		Username: "testuser",
		Email:    "test@example.com",
		IsActive: true,
	}

	// Test token generation
	token, err := service.GenerateJWT(user)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// Test token validation
	claims, err := service.ValidateJWT(token)
	assert.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, user.ID, claims.UserID)
	assert.Equal(t, user.Username, claims.Username)
	assert.True(t, claims.ExpiresAt.After(time.Now()))
}

// TestJWTTokenValidation tests JWT token validation with various scenarios
func TestJWTTokenValidation(t *testing.T) {
	service := NewAuthService(nil, "test-secret-key")

	tests := []struct {
		name        string
		token       string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "empty token",
			token:       "",
			expectError: true,
			errorMsg:    "token is required",
		},
		{
			name:        "invalid token format",
			token:       "invalid.token.format",
			expectError: true,
			errorMsg:    "invalid token",
		},
		{
			name:        "token with wrong secret",
			token:       "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
			expectError: true,
			errorMsg:    "invalid token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := service.ValidateJWT(tt.token)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, claims)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, claims)
			}
		})
	}
}

// TestPasswordHashing tests password hashing functionality
func TestPasswordHashing(t *testing.T) {
	service := NewAuthService(nil, "test-secret")

	password := "SecurePass123!"

	// Test password hashing
	hash, err := service.HashPassword(password)
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash)

	// Test password verification
	isValid := service.VerifyPassword(password, hash)
	assert.True(t, isValid)

	// Test wrong password
	isValid = service.VerifyPassword("wrongpassword", hash)
	assert.False(t, isValid)
}

// TestUserModel tests the User model validation
func TestUserModel(t *testing.T) {
	tests := []struct {
		name        string
		user        User
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid user",
			user: User{
				Username: "testuser",
				Email:    "test@example.com",
				IsActive: true,
			},
			expectError: false,
		},
		{
			name: "empty username",
			user: User{
				Username: "",
				Email:    "test@example.com",
				IsActive: true,
			},
			expectError: true,
			errorMsg:    "username is required",
		},
		{
			name: "empty email",
			user: User{
				Username: "testuser",
				Email:    "",
				IsActive: true,
			},
			expectError: true,
			errorMsg:    "email is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.user.Validate()

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// =============================================================================
// ADDITIONAL COMPREHENSIVE TESTS FOR 60%+ COVERAGE
// =============================================================================

func TestAuthService_HashPassword_EmptyPassword(t *testing.T) {
	service := NewAuthService(nil, "test-secret")

	hash, err := service.HashPassword("")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "password is required")
	assert.Empty(t, hash)
}

func TestAuthService_HashPassword_WhitespacePassword(t *testing.T) {
	service := NewAuthService(nil, "test-secret")

	hash, err := service.HashPassword("   ")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "password is required")
	assert.Empty(t, hash)
}

func TestAuthService_VerifyPassword_EmptyPassword(t *testing.T) {
	service := NewAuthService(nil, "test-secret")

	isValid := service.VerifyPassword("", "somehash")

	assert.False(t, isValid)
}

func TestAuthService_VerifyPassword_EmptyHash(t *testing.T) {
	service := NewAuthService(nil, "test-secret")

	isValid := service.VerifyPassword("password", "")

	assert.False(t, isValid)
}

func TestAuthService_VerifyPassword_BothEmpty(t *testing.T) {
	service := NewAuthService(nil, "test-secret")

	isValid := service.VerifyPassword("", "")

	assert.False(t, isValid)
}

func TestAuthService_GenerateJWT_NilUser(t *testing.T) {
	service := NewAuthService(nil, "test-secret")

	token, err := service.GenerateJWT(nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user is required")
	assert.Empty(t, token)
}

func TestAuthService_ValidateJWT_WhitespaceToken(t *testing.T) {
	service := NewAuthService(nil, "test-secret")

	claims, err := service.ValidateJWT("   ")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token is required")
	assert.Nil(t, claims)
}

func TestAuthService_LoginUser_NoRepository(t *testing.T) {
	service := NewAuthService(nil, "test-secret")

	user, token, err := service.LoginUser("testuser", "password123")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user repository not configured")
	assert.Nil(t, user)
	assert.Empty(t, token)
}

func TestAuthService_LoginUser_InactiveUser(t *testing.T) {
	mockRepo := NewMockUserRepository()
	service := NewAuthService(mockRepo, "test-secret")

	// Register a user first
	_, err := service.RegisterUser("testuser", "test@example.com", "SecurePass123!")
	require.NoError(t, err)

	// Deactivate the user using DeleteUser (soft delete sets IsActive = false)
	user, _ := mockRepo.GetUserByUsername("testuser")
	require.NotNil(t, user)
	mockRepo.DeleteUser(user.ID)

	// Try to login - mock returns nil for inactive users, so this returns "invalid credentials"
	_, _, err = service.LoginUser("testuser", "SecurePass123!")

	assert.Error(t, err)
	// Mock implementation filters out inactive users, so we get "invalid credentials"
	assert.Contains(t, err.Error(), "invalid credentials")
}

func TestAuthService_RegisterUser_DuplicateUsername(t *testing.T) {
	mockRepo := NewMockUserRepository()
	service := NewAuthService(mockRepo, "test-secret")

	// Register first user
	_, err := service.RegisterUser("testuser", "test1@example.com", "SecurePass123!")
	require.NoError(t, err)

	// Try to register with same username
	_, err = service.RegisterUser("testuser", "test2@example.com", "SecurePass123!")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "username already exists")
}

func TestAuthService_RegisterUser_DuplicateEmail(t *testing.T) {
	mockRepo := NewMockUserRepository()
	service := NewAuthService(mockRepo, "test-secret")

	// Register first user
	_, err := service.RegisterUser("testuser1", "test@example.com", "SecurePass123!")
	require.NoError(t, err)

	// Try to register with same email
	_, err = service.RegisterUser("testuser2", "test@example.com", "SecurePass123!")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "email already exists")
}

func TestAuthService_RegisterUser_EmptyPassword(t *testing.T) {
	mockRepo := NewMockUserRepository()
	service := NewAuthService(mockRepo, "test-secret")

	_, err := service.RegisterUser("testuser", "test@example.com", "")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "password is required")
}

func TestAuthService_RegisterUser_WhitespaceUsername(t *testing.T) {
	mockRepo := NewMockUserRepository()
	service := NewAuthService(mockRepo, "test-secret")

	_, err := service.RegisterUser("   ", "test@example.com", "SecurePass123!")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "username is required")
}

func TestAuthService_RegisterUser_WhitespaceEmail(t *testing.T) {
	mockRepo := NewMockUserRepository()
	service := NewAuthService(mockRepo, "test-secret")

	_, err := service.RegisterUser("testuser", "   ", "SecurePass123!")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "email is required")
}

func TestAuthService_RegisterUser_WhitespacePassword(t *testing.T) {
	mockRepo := NewMockUserRepository()
	service := NewAuthService(mockRepo, "test-secret")

	_, err := service.RegisterUser("testuser", "test@example.com", "   ")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "password is required")
}

func TestAuthService_LoginUser_WhitespaceUsername(t *testing.T) {
	mockRepo := NewMockUserRepository()
	service := NewAuthService(mockRepo, "test-secret")

	_, _, err := service.LoginUser("   ", "password123")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "username is required")
}

func TestAuthService_LoginUser_WhitespacePassword(t *testing.T) {
	mockRepo := NewMockUserRepository()
	service := NewAuthService(mockRepo, "test-secret")

	_, _, err := service.LoginUser("testuser", "   ")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "password is required")
}

func TestNewAuthService(t *testing.T) {
	mockRepo := NewMockUserRepository()
	service := NewAuthService(mockRepo, "test-secret")

	assert.NotNil(t, service)
}

func TestAuthService_GenerateAndValidateJWT(t *testing.T) {
	service := NewAuthService(nil, "test-secret-key-for-jwt")

	user := &User{
		ID:       42,
		Username: "jwtuser",
		Email:    "jwt@example.com",
		IsActive: true,
	}

	// Generate token
	token, err := service.GenerateJWT(user)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// Validate token
	claims, err := service.ValidateJWT(token)
	require.NoError(t, err)
	require.NotNil(t, claims)

	assert.Equal(t, int64(42), claims.UserID)
	assert.Equal(t, "jwtuser", claims.Username)
	assert.Equal(t, "jwt@example.com", claims.Email)
	assert.False(t, claims.IssuedAt.IsZero())
	assert.False(t, claims.ExpiresAt.IsZero())
	assert.True(t, claims.ExpiresAt.After(claims.IssuedAt))
}

func TestJWTClaims_Structure(t *testing.T) {
	now := time.Now()
	claims := JWTClaims{
		UserID:    123,
		Username:  "testuser",
		Email:     "test@example.com",
		IssuedAt:  now,
		ExpiresAt: now.Add(24 * time.Hour),
	}

	assert.Equal(t, int64(123), claims.UserID)
	assert.Equal(t, "testuser", claims.Username)
	assert.Equal(t, "test@example.com", claims.Email)
	assert.True(t, claims.ExpiresAt.After(claims.IssuedAt))
}

func TestUser_Structure(t *testing.T) {
	now := time.Now()
	user := User{
		ID:           1,
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hashedpassword",
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	assert.Equal(t, int64(1), user.ID)
	assert.Equal(t, "testuser", user.Username)
	assert.Equal(t, "test@example.com", user.Email)
	assert.NotEmpty(t, user.PasswordHash)
	assert.True(t, user.IsActive)
}
