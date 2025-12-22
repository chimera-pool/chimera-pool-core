package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// COMPREHENSIVE AUTH MODELS TESTS
// World-class test coverage for authentication models
// =============================================================================

// Role tests are in role_test.go - these are User/JWT/Email validation tests

// =============================================================================
// USER VALIDATION TESTS
// =============================================================================

func TestUser_Validate_Success(t *testing.T) {
	user := &User{
		Username: "validuser",
		Email:    "valid@example.com",
	}

	err := user.Validate()
	assert.NoError(t, err)
}

func TestUser_Validate_UsernameRequired(t *testing.T) {
	tests := []struct {
		name     string
		username string
	}{
		{"empty username", ""},
		{"whitespace only", "   "},
		{"tabs only", "\t\t"},
		{"newlines only", "\n\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{
				Username: tt.username,
				Email:    "valid@example.com",
			}

			err := user.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), "username is required")
		})
	}
}

func TestUser_Validate_UsernameTooShort(t *testing.T) {
	tests := []struct {
		name     string
		username string
	}{
		{"1 character", "a"},
		{"2 characters", "ab"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{
				Username: tt.username,
				Email:    "valid@example.com",
			}

			err := user.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), "at least 3 characters")
		})
	}
}

func TestUser_Validate_UsernameTooLong(t *testing.T) {
	user := &User{
		Username: string(make([]byte, 51)), // 51 characters
		Email:    "valid@example.com",
	}

	err := user.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at most 50 characters")
}

func TestUser_Validate_UsernameExactBoundaries(t *testing.T) {
	// Exactly 3 characters (minimum)
	user3 := &User{
		Username: "abc",
		Email:    "valid@example.com",
	}
	assert.NoError(t, user3.Validate())

	// Exactly 50 characters (maximum)
	user50 := &User{
		Username: "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwx", // 50 chars
		Email:    "valid@example.com",
	}
	assert.NoError(t, user50.Validate())
}

func TestUser_Validate_EmailRequired(t *testing.T) {
	tests := []struct {
		name  string
		email string
	}{
		{"empty email", ""},
		{"whitespace only", "   "},
		{"tabs only", "\t\t"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{
				Username: "validuser",
				Email:    tt.email,
			}

			err := user.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), "email is required")
		})
	}
}

func TestUser_Validate_EmailFormat(t *testing.T) {
	validEmails := []string{
		"user@example.com",
		"user.name@example.com",
		"user+tag@example.com",
		"user@subdomain.example.com",
		"user123@example.org",
		"user_name@example.co.uk",
		"user-name@example.io",
	}

	for _, email := range validEmails {
		t.Run("valid: "+email, func(t *testing.T) {
			user := &User{
				Username: "validuser",
				Email:    email,
			}
			err := user.Validate()
			assert.NoError(t, err, "Expected valid email: %s", email)
		})
	}

	invalidEmails := []string{
		"plainaddress",
		"@missinguser.com",
		"missingdomain@",
		"missing@domain",
		"user@.com",
		"user..name@example.com",
		".user@example.com",
		"user@example..com",
	}

	for _, email := range invalidEmails {
		t.Run("invalid: "+email, func(t *testing.T) {
			user := &User{
				Username: "validuser",
				Email:    email,
			}
			err := user.Validate()
			assert.Error(t, err, "Expected invalid email: %s", email)
		})
	}
}

// =============================================================================
// USER STRUCT TESTS
// =============================================================================

func TestUser_StructFields(t *testing.T) {
	now := time.Now()
	user := &User{
		ID:           123,
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hashedpassword",
		Role:         RoleAdmin,
		CreatedAt:    now,
		UpdatedAt:    now,
		IsActive:     true,
	}

	assert.Equal(t, int64(123), user.ID)
	assert.Equal(t, "testuser", user.Username)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "hashedpassword", user.PasswordHash)
	assert.Equal(t, RoleAdmin, user.Role)
	assert.Equal(t, now, user.CreatedAt)
	assert.Equal(t, now, user.UpdatedAt)
	assert.True(t, user.IsActive)
}

func TestUser_DefaultRole(t *testing.T) {
	user := &User{
		Username: "newuser",
		Email:    "new@example.com",
	}

	// Default role should be empty (zero value)
	assert.Equal(t, Role(""), user.Role)
}

// =============================================================================
// JWT CLAIMS TESTS
// =============================================================================

func TestJWTClaims_StructFields(t *testing.T) {
	issuedAt := time.Now()
	expiresAt := issuedAt.Add(24 * time.Hour)

	claims := &JWTClaims{
		UserID:    123,
		Username:  "testuser",
		Email:     "test@example.com",
		IssuedAt:  issuedAt,
		ExpiresAt: expiresAt,
	}

	assert.Equal(t, int64(123), claims.UserID)
	assert.Equal(t, "testuser", claims.Username)
	assert.Equal(t, "test@example.com", claims.Email)
	assert.Equal(t, issuedAt, claims.IssuedAt)
	assert.Equal(t, expiresAt, claims.ExpiresAt)
}

func TestJWTClaims_Expiration(t *testing.T) {
	now := time.Now()

	// Not expired
	validClaims := &JWTClaims{
		UserID:    123,
		ExpiresAt: now.Add(1 * time.Hour),
	}
	assert.True(t, validClaims.ExpiresAt.After(now))

	// Expired
	expiredClaims := &JWTClaims{
		UserID:    123,
		ExpiresAt: now.Add(-1 * time.Hour),
	}
	assert.True(t, expiredClaims.ExpiresAt.Before(now))
}

// =============================================================================
// EMAIL VALIDATION HELPER TESTS
// =============================================================================

func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		email    string
		expected bool
	}{
		// Valid emails
		{"user@example.com", true},
		{"user.name@example.com", true},
		{"user+tag@example.com", true},
		{"user123@example.org", true},
		{"a@b.co", true},

		// Invalid emails
		{"", false},
		{"plainaddress", false},
		{"@missinguser.com", false},
		{"missingdomain@", false},
		{"missing@domain", false},
		{"user@.com", false},
		{"user..name@example.com", false},
		{".user@example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			result := isValidEmail(tt.email)
			assert.Equal(t, tt.expected, result, "Email: %s", tt.email)
		})
	}
}

// =============================================================================
// BENCHMARK TESTS
// =============================================================================

func BenchmarkRole_IsValid(b *testing.B) {
	role := RoleAdmin
	for i := 0; i < b.N; i++ {
		role.IsValid()
	}
}

func BenchmarkRole_Level(b *testing.B) {
	role := RoleAdmin
	for i := 0; i < b.N; i++ {
		role.Level()
	}
}

func BenchmarkRole_CanManageRole(b *testing.B) {
	role := RoleAdmin
	target := RoleModerator
	for i := 0; i < b.N; i++ {
		role.CanManageRole(target)
	}
}

func BenchmarkUser_Validate(b *testing.B) {
	user := &User{
		Username: "validuser",
		Email:    "valid@example.com",
	}
	for i := 0; i < b.N; i++ {
		user.Validate()
	}
}

func BenchmarkIsValidEmail(b *testing.B) {
	email := "user@example.com"
	for i := 0; i < b.N; i++ {
		isValidEmail(email)
	}
}

// =============================================================================
// EDGE CASES AND SECURITY TESTS
// =============================================================================

func TestUser_Validate_SQLInjectionAttempts(t *testing.T) {
	// These should all fail validation due to email format
	injectionAttempts := []string{
		"user@example.com'; DROP TABLE users;--",
		"user@example.com' OR '1'='1",
		"user@example.com; DELETE FROM users",
	}

	for _, email := range injectionAttempts {
		t.Run("injection attempt", func(t *testing.T) {
			user := &User{
				Username: "validuser",
				Email:    email,
			}
			err := user.Validate()
			assert.Error(t, err, "Should reject SQL injection attempt: %s", email)
		})
	}
}

func TestUser_Validate_XSSAttempts(t *testing.T) {
	// XSS in username should still pass validation (escaping happens elsewhere)
	// But we test that validation doesn't crash
	xssAttempts := []string{
		"<script>alert('xss')</script>",
		"user<img src=x onerror=alert(1)>",
		"javascript:alert(1)",
	}

	for _, username := range xssAttempts {
		t.Run("xss in username", func(t *testing.T) {
			user := &User{
				Username: username,
				Email:    "valid@example.com",
			}
			// Should not panic
			_ = user.Validate()
		})
	}
}

func TestRole_String(t *testing.T) {
	// Role is a string type, so string conversion should work
	assert.Equal(t, "user", string(RoleUser))
	assert.Equal(t, "moderator", string(RoleModerator))
	assert.Equal(t, "admin", string(RoleAdmin))
	assert.Equal(t, "super_admin", string(RoleSuperAdmin))
}
