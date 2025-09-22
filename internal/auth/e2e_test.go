package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2EAuthenticationFlow tests the complete end-to-end authentication workflow
func TestE2EAuthenticationFlow(t *testing.T) {
	// Setup
	mockRepo := NewMockUserRepository()
	authService := NewAuthService(mockRepo, "super-secret-jwt-key")
	
	// Test scenario: Complete user lifecycle
	t.Run("complete_user_lifecycle", func(t *testing.T) {
		// Step 1: Register a new user
		username := "e2euser"
		email := "e2e@example.com"
		password := "SecureE2EPass123!"
		
		user, err := authService.RegisterUser(username, email, password)
		require.NoError(t, err, "User registration should succeed")
		require.NotNil(t, user, "Registered user should not be nil")
		
		// Verify user properties
		assert.Equal(t, username, user.Username)
		assert.Equal(t, email, user.Email)
		assert.NotEmpty(t, user.PasswordHash)
		assert.NotEqual(t, password, user.PasswordHash, "Password should be hashed")
		assert.True(t, user.IsActive)
		assert.NotZero(t, user.ID)
		assert.False(t, user.CreatedAt.IsZero())
		assert.False(t, user.UpdatedAt.IsZero())
		
		// Step 2: Login with the registered user
		loginUser, token, err := authService.LoginUser(username, password)
		require.NoError(t, err, "User login should succeed")
		require.NotNil(t, loginUser, "Login user should not be nil")
		require.NotEmpty(t, token, "JWT token should be generated")
		
		// Verify login user matches registered user
		assert.Equal(t, user.ID, loginUser.ID)
		assert.Equal(t, user.Username, loginUser.Username)
		assert.Equal(t, user.Email, loginUser.Email)
		assert.Equal(t, user.IsActive, loginUser.IsActive)
		
		// Step 3: Validate the JWT token
		claims, err := authService.ValidateJWT(token)
		require.NoError(t, err, "JWT token validation should succeed")
		require.NotNil(t, claims, "JWT claims should not be nil")
		
		// Verify JWT claims
		assert.Equal(t, user.ID, claims.UserID)
		assert.Equal(t, user.Username, claims.Username)
		assert.Equal(t, user.Email, claims.Email)
		assert.True(t, claims.ExpiresAt.After(time.Now()), "Token should not be expired")
		assert.True(t, claims.IssuedAt.Before(time.Now().Add(time.Minute)), "Token should be recently issued")
		
		// Step 4: Use the token for subsequent requests (simulate middleware)
		isAuthenticated := func(tokenString string) (*User, error) {
			claims, err := authService.ValidateJWT(tokenString)
			if err != nil {
				return nil, err
			}
			
			// In a real application, you might fetch fresh user data from the database
			return &User{
				ID:       claims.UserID,
				Username: claims.Username,
				Email:    claims.Email,
				IsActive: true,
			}, nil
		}
		
		authenticatedUser, err := isAuthenticated(token)
		require.NoError(t, err, "Token-based authentication should succeed")
		assert.Equal(t, user.ID, authenticatedUser.ID)
		assert.Equal(t, user.Username, authenticatedUser.Username)
		
		// Step 5: Test security boundaries
		
		// 5a: Wrong password should fail
		_, _, err = authService.LoginUser(username, "wrongpassword")
		assert.Error(t, err, "Login with wrong password should fail")
		assert.Contains(t, err.Error(), "invalid credentials")
		
		// 5b: Non-existent user should fail
		_, _, err = authService.LoginUser("nonexistent", password)
		assert.Error(t, err, "Login with non-existent user should fail")
		assert.Contains(t, err.Error(), "invalid credentials")
		
		// 5c: Invalid token should fail
		_, err = authService.ValidateJWT("invalid.token.here")
		assert.Error(t, err, "Invalid token should fail validation")
		
		// 5d: Empty token should fail
		_, err = authService.ValidateJWT("")
		assert.Error(t, err, "Empty token should fail validation")
		assert.Contains(t, err.Error(), "token is required")
		
		// Step 6: Test duplicate registration prevention
		_, err = authService.RegisterUser(username, "different@example.com", password)
		assert.Error(t, err, "Duplicate username registration should fail")
		assert.Contains(t, err.Error(), "username already exists")
		
		_, err = authService.RegisterUser("differentuser", email, password)
		assert.Error(t, err, "Duplicate email registration should fail")
		assert.Contains(t, err.Error(), "email already exists")
	})
}

// TestE2EMultipleUsers tests authentication with multiple users
func TestE2EMultipleUsers(t *testing.T) {
	// Setup
	mockRepo := NewMockUserRepository()
	authService := NewAuthService(mockRepo, "multi-user-secret-key")
	
	// Create multiple users
	users := []struct {
		username string
		email    string
		password string
	}{
		{"user1", "user1@example.com", "Password1!"},
		{"user2", "user2@example.com", "Password2!"},
		{"user3", "user3@example.com", "Password3!"},
	}
	
	var registeredUsers []*User
	var tokens []string
	
	// Register all users
	for _, userData := range users {
		user, err := authService.RegisterUser(userData.username, userData.email, userData.password)
		require.NoError(t, err, "User registration should succeed for %s", userData.username)
		registeredUsers = append(registeredUsers, user)
		
		// Login each user
		_, token, err := authService.LoginUser(userData.username, userData.password)
		require.NoError(t, err, "User login should succeed for %s", userData.username)
		tokens = append(tokens, token)
	}
	
	// Verify all users have unique IDs
	for i, user1 := range registeredUsers {
		for j, user2 := range registeredUsers {
			if i != j {
				assert.NotEqual(t, user1.ID, user2.ID, "Users should have unique IDs")
				assert.NotEqual(t, user1.Username, user2.Username, "Users should have unique usernames")
				assert.NotEqual(t, user1.Email, user2.Email, "Users should have unique emails")
			}
		}
	}
	
	// Verify all tokens are valid and unique
	for i, token1 := range tokens {
		claims1, err := authService.ValidateJWT(token1)
		require.NoError(t, err, "Token should be valid")
		
		for j, token2 := range tokens {
			if i != j {
				assert.NotEqual(t, token1, token2, "Tokens should be unique")
				
				claims2, err := authService.ValidateJWT(token2)
				require.NoError(t, err, "Token should be valid")
				
				assert.NotEqual(t, claims1.UserID, claims2.UserID, "Token claims should have different user IDs")
			}
		}
	}
	
	// Test cross-user authentication (user1's token shouldn't work for user2's data)
	for i, token := range tokens {
		claims, err := authService.ValidateJWT(token)
		require.NoError(t, err)
		
		// Verify the token belongs to the correct user
		expectedUser := registeredUsers[i]
		assert.Equal(t, expectedUser.ID, claims.UserID)
		assert.Equal(t, expectedUser.Username, claims.Username)
		assert.Equal(t, expectedUser.Email, claims.Email)
	}
}

// TestE2EPasswordSecurity tests password security features
func TestE2EPasswordSecurity(t *testing.T) {
	mockRepo := NewMockUserRepository()
	authService := NewAuthService(mockRepo, "password-security-key")
	
	// Test various password scenarios
	testCases := []struct {
		name        string
		password    string
		shouldWork  bool
		errorMsg    string
	}{
		{"strong password", "StrongPass123!", true, ""},
		{"weak password", "123", false, "password must be at least 8 characters"},
		{"empty password", "", false, "password is required"},
		{"whitespace password", "   ", false, "password is required"},
		{"long password", "ThisIsAVeryLongPasswordThatShouldStillWorkFine123!", true, ""},
		{"special chars", "P@ssw0rd!#$%", true, ""},
		{"unicode password", "Pässwörd123!", true, ""},
	}
	
	for i, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			username := "testuser" + string(rune(i+'0'))
			email := "test" + string(rune(i+'0')) + "@example.com"
			
			user, err := authService.RegisterUser(username, email, tc.password)
			
			if tc.shouldWork {
				require.NoError(t, err, "Registration should succeed for valid password")
				require.NotNil(t, user)
				
				// Verify password is properly hashed
				assert.NotEqual(t, tc.password, user.PasswordHash, "Password should be hashed")
				assert.True(t, len(user.PasswordHash) > 50, "Hash should be substantial length")
				
				// Verify login works
				_, token, err := authService.LoginUser(username, tc.password)
				require.NoError(t, err, "Login should work with correct password")
				assert.NotEmpty(t, token)
				
				// Verify wrong password fails
				_, _, err = authService.LoginUser(username, "wrongpassword")
				assert.Error(t, err, "Login should fail with wrong password")
				
				// Test password verification directly
				isValid := authService.VerifyPassword(tc.password, user.PasswordHash)
				assert.True(t, isValid, "Password verification should succeed")
				
				isValid = authService.VerifyPassword("wrongpassword", user.PasswordHash)
				assert.False(t, isValid, "Wrong password verification should fail")
				
			} else {
				assert.Error(t, err, "Registration should fail for invalid password")
				assert.Contains(t, err.Error(), tc.errorMsg)
				assert.Nil(t, user)
			}
		})
	}
}

// TestE2ETokenExpiration tests JWT token expiration (simulated)
func TestE2ETokenExpiration(t *testing.T) {
	mockRepo := NewMockUserRepository()
	authService := NewAuthService(mockRepo, "expiration-test-key")
	
	// Register and login user
	user, err := authService.RegisterUser("exptest", "exp@example.com", "ExpTest123!")
	require.NoError(t, err)
	
	_, token, err := authService.LoginUser("exptest", "ExpTest123!")
	require.NoError(t, err)
	
	// Token should be valid now
	claims, err := authService.ValidateJWT(token)
	require.NoError(t, err)
	assert.Equal(t, user.ID, claims.UserID)
	
	// Verify token expiration time is reasonable (24 hours from now)
	expectedExpiration := time.Now().Add(24 * time.Hour)
	timeDiff := claims.ExpiresAt.Sub(expectedExpiration)
	assert.True(t, timeDiff < time.Minute && timeDiff > -time.Minute, 
		"Token expiration should be approximately 24 hours from now")
	
	// Test that issued time is recent
	issuedDiff := time.Now().Sub(claims.IssuedAt)
	assert.True(t, issuedDiff < time.Minute, 
		"Token should be issued recently")
}

// TestE2EInputSanitization tests input sanitization and validation
func TestE2EInputSanitization(t *testing.T) {
	mockRepo := NewMockUserRepository()
	authService := NewAuthService(mockRepo, "sanitization-key")
	
	// Test input with whitespace
	user, err := authService.RegisterUser("  testuser  ", "  test@example.com  ", "TestPass123!")
	require.NoError(t, err, "Registration should handle whitespace")
	assert.Equal(t, "testuser", user.Username, "Username should be trimmed")
	assert.Equal(t, "test@example.com", user.Email, "Email should be trimmed")
	
	// Login should also handle whitespace
	_, token, err := authService.LoginUser("  testuser  ", "TestPass123!")
	require.NoError(t, err, "Login should handle whitespace in username")
	assert.NotEmpty(t, token)
	
	// Test various email formats
	validEmails := []string{
		"simple@example.com",
		"user.name@example.com",
		"user+tag@example.com",
		"user123@example-domain.com",
		"test@subdomain.example.com",
	}
	
	for i, email := range validEmails {
		username := "emailtest" + string(rune(i+'0'))
		user, err := authService.RegisterUser(username, email, "EmailTest123!")
		assert.NoError(t, err, "Valid email should be accepted: %s", email)
		assert.Equal(t, email, user.Email)
	}
	
	// Test invalid email formats
	invalidEmails := []string{
		"invalid-email",
		"@example.com",
		"user@",
		"user..name@example.com",
		"user@.com",
		"",
		"   ",
	}
	
	for i, email := range invalidEmails {
		username := "invalidemail" + string(rune(i+'0'))
		_, err := authService.RegisterUser(username, email, "InvalidEmail123!")
		assert.Error(t, err, "Invalid email should be rejected: %s", email)
	}
}