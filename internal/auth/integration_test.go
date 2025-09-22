package auth

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// AuthIntegrationTestSuite provides integration tests for the auth service
type AuthIntegrationTestSuite struct {
	suite.Suite
	db          *sql.DB
	authService *AuthService
	userRepo    *PostgreSQLUserRepository
}

// SetupSuite sets up the test suite
func (suite *AuthIntegrationTestSuite) SetupSuite() {
	// This would normally connect to a test database
	// For now, we'll skip database-dependent tests if no DB is available
	suite.db = nil // Will be set up when database is available
	
	if suite.db != nil {
		suite.userRepo = NewPostgreSQLUserRepository(suite.db)
		suite.authService = NewAuthService(suite.userRepo, "test-secret-key")
	} else {
		// Create service without database for unit testing
		suite.authService = NewAuthService(nil, "test-secret-key")
	}
}

// TearDownSuite cleans up after tests
func (suite *AuthIntegrationTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Close()
	}
}

// SetupTest sets up each test
func (suite *AuthIntegrationTestSuite) SetupTest() {
	if suite.db != nil {
		// Clean up users table before each test
		_, err := suite.db.Exec("DELETE FROM users WHERE username LIKE 'test%'")
		suite.Require().NoError(err)
	}
}

// TestCompleteAuthenticationFlow tests the complete authentication workflow
func (suite *AuthIntegrationTestSuite) TestCompleteAuthenticationFlow() {
	t := suite.T()
	
	// Test data
	username := "testuser"
	email := "test@example.com"
	password := "SecurePass123!"
	
	// Step 1: Register a new user
	user, err := suite.authService.RegisterUser(username, email, password)
	
	if suite.db == nil {
		// Without database, we can still test validation logic
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, username, user.Username)
		assert.Equal(t, email, user.Email)
		assert.NotEmpty(t, user.PasswordHash)
		assert.NotEqual(t, password, user.PasswordHash)
		assert.True(t, user.IsActive)
		
		// Skip database-dependent tests
		t.Skip("Database not available, skipping database-dependent tests")
		return
	}
	
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, username, user.Username)
	assert.Equal(t, email, user.Email)
	assert.NotEmpty(t, user.PasswordHash)
	assert.NotEqual(t, password, user.PasswordHash)
	assert.True(t, user.IsActive)
	assert.NotZero(t, user.ID)
	assert.False(t, user.CreatedAt.IsZero())
	assert.False(t, user.UpdatedAt.IsZero())
	
	// Step 2: Attempt to register with same username (should fail)
	_, err = suite.authService.RegisterUser(username, "different@example.com", password)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "username already exists")
	
	// Step 3: Attempt to register with same email (should fail)
	_, err = suite.authService.RegisterUser("differentuser", email, password)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "email already exists")
	
	// Step 4: Login with correct credentials
	loginUser, token, err := suite.authService.LoginUser(username, password)
	require.NoError(t, err)
	require.NotNil(t, loginUser)
	assert.NotEmpty(t, token)
	assert.Equal(t, user.ID, loginUser.ID)
	assert.Equal(t, user.Username, loginUser.Username)
	assert.Equal(t, user.Email, loginUser.Email)
	
	// Step 5: Validate the JWT token
	claims, err := suite.authService.ValidateJWT(token)
	require.NoError(t, err)
	require.NotNil(t, claims)
	assert.Equal(t, user.ID, claims.UserID)
	assert.Equal(t, user.Username, claims.Username)
	assert.Equal(t, user.Email, claims.Email)
	assert.True(t, claims.ExpiresAt.After(time.Now()))
	
	// Step 6: Login with wrong password (should fail)
	_, _, err = suite.authService.LoginUser(username, "wrongpassword")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid credentials")
	
	// Step 7: Login with non-existent user (should fail)
	_, _, err = suite.authService.LoginUser("nonexistent", password)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid credentials")
}

// TestUserRepository tests the user repository functionality
func (suite *AuthIntegrationTestSuite) TestUserRepository() {
	t := suite.T()
	
	if suite.db == nil {
		t.Skip("Database not available, skipping repository tests")
		return
	}
	
	// Test data
	user := &User{
		Username:     "repotest",
		Email:        "repo@example.com",
		PasswordHash: "hashedpassword",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		IsActive:     true,
	}
	
	// Test CreateUser
	err := suite.userRepo.CreateUser(user)
	require.NoError(t, err)
	assert.NotZero(t, user.ID)
	
	// Test GetUserByUsername
	retrievedUser, err := suite.userRepo.GetUserByUsername(user.Username)
	require.NoError(t, err)
	require.NotNil(t, retrievedUser)
	assert.Equal(t, user.ID, retrievedUser.ID)
	assert.Equal(t, user.Username, retrievedUser.Username)
	assert.Equal(t, user.Email, retrievedUser.Email)
	
	// Test GetUserByEmail
	retrievedUser, err = suite.userRepo.GetUserByEmail(user.Email)
	require.NoError(t, err)
	require.NotNil(t, retrievedUser)
	assert.Equal(t, user.ID, retrievedUser.ID)
	
	// Test GetUserByID
	retrievedUser, err = suite.userRepo.GetUserByID(user.ID)
	require.NoError(t, err)
	require.NotNil(t, retrievedUser)
	assert.Equal(t, user.ID, retrievedUser.ID)
	
	// Test UpdateUser
	user.Email = "updated@example.com"
	err = suite.userRepo.UpdateUser(user)
	require.NoError(t, err)
	
	retrievedUser, err = suite.userRepo.GetUserByID(user.ID)
	require.NoError(t, err)
	assert.Equal(t, "updated@example.com", retrievedUser.Email)
	
	// Test DeleteUser (soft delete)
	err = suite.userRepo.DeleteUser(user.ID)
	require.NoError(t, err)
	
	// User should not be found by username/email (because is_active = false)
	retrievedUser, err = suite.userRepo.GetUserByUsername(user.Username)
	require.NoError(t, err)
	assert.Nil(t, retrievedUser)
	
	// But should still be found by ID
	retrievedUser, err = suite.userRepo.GetUserByID(user.ID)
	require.NoError(t, err)
	require.NotNil(t, retrievedUser)
	assert.False(t, retrievedUser.IsActive)
}

// TestPasswordSecurity tests password hashing and verification
func (suite *AuthIntegrationTestSuite) TestPasswordSecurity() {
	t := suite.T()
	
	passwords := []string{
		"SimplePass123!",
		"ComplexP@ssw0rd!",
		"AnotherSecure123",
		"VeryLongPasswordWithSpecialChars!@#$%^&*()",
	}
	
	for _, password := range passwords {
		t.Run("password_"+password[:8], func(t *testing.T) {
			// Hash password
			hash, err := suite.authService.HashPassword(password)
			require.NoError(t, err)
			assert.NotEmpty(t, hash)
			assert.NotEqual(t, password, hash)
			
			// Verify correct password
			isValid := suite.authService.VerifyPassword(password, hash)
			assert.True(t, isValid)
			
			// Verify wrong password
			isValid = suite.authService.VerifyPassword("wrongpassword", hash)
			assert.False(t, isValid)
			
			// Each hash should be unique (even for same password)
			hash2, err := suite.authService.HashPassword(password)
			require.NoError(t, err)
			assert.NotEqual(t, hash, hash2)
			
			// But both should verify correctly
			isValid = suite.authService.VerifyPassword(password, hash2)
			assert.True(t, isValid)
		})
	}
}

// TestJWTTokenSecurity tests JWT token security features
func (suite *AuthIntegrationTestSuite) TestJWTTokenSecurity() {
	t := suite.T()
	
	user := &User{
		ID:       1,
		Username: "testuser",
		Email:    "test@example.com",
		IsActive: true,
	}
	
	// Generate token
	token, err := suite.authService.GenerateJWT(user)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	
	// Validate token
	claims, err := suite.authService.ValidateJWT(token)
	require.NoError(t, err)
	require.NotNil(t, claims)
	assert.Equal(t, user.ID, claims.UserID)
	assert.Equal(t, user.Username, claims.Username)
	
	// Test token with different secret (should fail)
	differentService := NewAuthService(nil, "different-secret")
	_, err = differentService.ValidateJWT(token)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token")
	
	// Test malformed token
	_, err = suite.authService.ValidateJWT("invalid.token.format")
	assert.Error(t, err)
	
	// Test empty token
	_, err = suite.authService.ValidateJWT("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token is required")
}

// TestInputValidation tests input validation and edge cases
func (suite *AuthIntegrationTestSuite) TestInputValidation() {
	t := suite.T()
	
	// Test registration with invalid inputs
	testCases := []struct {
		name        string
		username    string
		email       string
		password    string
		expectedErr string
	}{
		{"empty username", "", "test@example.com", "password123", "username is required"},
		{"empty email", "testuser", "", "password123", "email is required"},
		{"empty password", "testuser", "test@example.com", "", "password is required"},
		{"short username", "ab", "test@example.com", "password123", "username must be at least 3 characters"},
		{"long username", "this_is_a_very_long_username_that_exceeds_the_fifty_character_limit_for_usernames", "test@example.com", "password123", "username must be at most 50 characters"},
		{"invalid email", "testuser", "invalid-email", "password123", "invalid email format"},
		{"short password", "testuser", "test@example.com", "123", "password must be at least 8 characters"},
		{"whitespace username", "   ", "test@example.com", "password123", "username is required"},
		{"whitespace email", "testuser", "   ", "password123", "email is required"},
		{"whitespace password", "testuser", "test@example.com", "   ", "password is required"},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := suite.authService.RegisterUser(tc.username, tc.email, tc.password)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectedErr)
		})
	}
}

// Run the test suite
func TestAuthIntegrationSuite(t *testing.T) {
	suite.Run(t, new(AuthIntegrationTestSuite))
}