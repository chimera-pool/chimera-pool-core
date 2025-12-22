package api

import (
	"testing"
	"time"
)

// =============================================================================
// AUTH SERVICE TESTS
// Unit tests for ISP-compliant auth service implementations
// =============================================================================

func TestBcryptHasher_Hash(t *testing.T) {
	hasher := NewBcryptHasher()

	password := "testPassword123"
	hash, err := hasher.Hash(password)

	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	if hash == "" {
		t.Error("Hash() returned empty string")
	}

	if hash == password {
		t.Error("Hash() returned unhashed password")
	}
}

func TestBcryptHasher_Verify(t *testing.T) {
	hasher := NewBcryptHasher()

	password := "testPassword123"
	hash, _ := hasher.Hash(password)

	tests := []struct {
		name     string
		password string
		hash     string
		want     bool
	}{
		{"correct password", password, hash, true},
		{"wrong password", "wrongPassword", hash, false},
		{"empty password", "", hash, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasher.Verify(tt.password, tt.hash); got != tt.want {
				t.Errorf("Verify() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJWTTokenGenerator_Generate(t *testing.T) {
	secret := "test-secret-key"
	expiration := 24 * time.Hour
	generator := NewJWTTokenGenerator(secret, expiration)

	userID := int64(123)
	username := "testuser"

	token, err := generator.Generate(userID, username)

	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if token == "" {
		t.Error("Generate() returned empty token")
	}

	// Token should be a valid JWT (3 parts separated by dots)
	parts := 0
	for _, c := range token {
		if c == '.' {
			parts++
		}
	}
	if parts != 2 {
		t.Errorf("Generate() token has %d dots, expected 2 (JWT format)", parts)
	}
}

func TestJWTTokenValidator_Validate(t *testing.T) {
	secret := "test-secret-key"
	expiration := 24 * time.Hour

	generator := NewJWTTokenGenerator(secret, expiration)
	validator := NewJWTTokenValidator(secret)

	userID := int64(456)
	username := "testuser2"

	token, _ := generator.Generate(userID, username)

	claims, err := validator.Validate(token)

	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("Validate() UserID = %d, want %d", claims.UserID, userID)
	}

	if claims.Username != username {
		t.Errorf("Validate() Username = %s, want %s", claims.Username, username)
	}
}

func TestJWTTokenValidator_Validate_InvalidToken(t *testing.T) {
	validator := NewJWTTokenValidator("test-secret")

	tests := []struct {
		name  string
		token string
	}{
		{"empty token", ""},
		{"random string", "not-a-valid-token"},
		{"malformed jwt", "header.payload"},
		{"wrong signature", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxMjN9.wrong-signature"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validator.Validate(tt.token)
			if err == nil {
				t.Error("Validate() expected error for invalid token")
			}
		})
	}
}

func TestJWTTokenValidator_Validate_WrongSecret(t *testing.T) {
	generator := NewJWTTokenGenerator("secret1", 24*time.Hour)
	validator := NewJWTTokenValidator("secret2")

	token, _ := generator.Generate(123, "user")

	_, err := validator.Validate(token)
	if err == nil {
		t.Error("Validate() should fail with wrong secret")
	}
}

func TestAuthServices_Factory(t *testing.T) {
	// Test that factory creates all services without nil pointers
	// Note: db is nil here, which is fine for factory creation test
	services := NewAuthServices(nil, "test-secret")

	if services.Hasher == nil {
		t.Error("Hasher is nil")
	}
	if services.TokenGenerator == nil {
		t.Error("TokenGenerator is nil")
	}
	if services.TokenValidator == nil {
		t.Error("TokenValidator is nil")
	}
	if services.Registrar == nil {
		t.Error("Registrar is nil")
	}
	if services.Authenticator == nil {
		t.Error("Authenticator is nil")
	}
	if services.PasswordResetter == nil {
		t.Error("PasswordResetter is nil")
	}
}
