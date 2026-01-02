package validation

import (
	"testing"
)

func TestValidateLitecoinAddress(t *testing.T) {
	tests := []struct {
		name    string
		address string
		wantErr error
	}{
		// Valid Bech32 addresses
		{
			name:    "valid bech32 address",
			address: "ltc1qgsm3fv44wprdcsh3trgarm05rr7l8ryggujr5w",
			wantErr: nil,
		},
		{
			name:    "valid bech32 address uppercase",
			address: "LTC1QGSM3FV44WPRDCSH3TRGARM05RR7L8RYGGUJR5W",
			wantErr: nil,
		},
		// Valid Legacy addresses
		{
			name:    "valid legacy L address",
			address: "LVg2kJoFNg45Nbpy53h7Fe1wKyeXVRhMH9",
			wantErr: nil,
		},
		{
			name:    "valid legacy M address",
			address: "MJKoMfDLQmV9qV5R5pq5cXvPLMYpKjLT8H",
			wantErr: nil,
		},
		// Invalid addresses
		{
			name:    "empty address",
			address: "",
			wantErr: ErrInvalidWalletAddress,
		},
		{
			name:    "too short",
			address: "ltc1abc",
			wantErr: ErrWalletAddressTooShort,
		},
		{
			name:    "random string",
			address: "INVALID123FAKE",
			wantErr: ErrWalletAddressTooShort,
		},
		{
			name:    "attacker wallet",
			address: "ATTACKER_WALLET_XYZ123",
			wantErr: ErrWalletAddressTooShort,
		},
		// SQL Injection attempts
		{
			name:    "sql injection single quote",
			address: "' OR '1'='1",
			wantErr: ErrSQLInjectionDetected,
		},
		{
			name:    "sql injection union",
			address: "ltc1 UNION SELECT * FROM users",
			wantErr: ErrSQLInjectionDetected,
		},
		{
			name:    "sql injection drop table",
			address: "ltc1; DROP TABLE users;--",
			wantErr: ErrSQLInjectionDetected,
		},
		{
			name:    "sql injection comment",
			address: "ltc1--comment",
			wantErr: ErrSQLInjectionDetected,
		},
		// XSS attempts (note: some XSS payloads contain SQL injection chars like ' which are caught first)
		{
			name:    "xss script tag with quotes",
			address: "<script>alert('XSS')</script>",
			wantErr: ErrSQLInjectionDetected, // Caught by SQL injection first due to single quote
		},
		{
			name:    "xss script tag no quotes",
			address: "<script>alert(1)</script>",
			wantErr: ErrXSSDetected,
		},
		{
			name:    "xss angle bracket only",
			address: "test<test",
			wantErr: ErrXSSDetected,
		},
		{
			name:    "xss img onerror",
			address: "<img src=x onerror=alert(1)>",
			wantErr: ErrXSSDetected,
		},
		{
			name:    "xss javascript protocol",
			address: "javascript:alert(1)",
			wantErr: ErrXSSDetected,
		},
		{
			name:    "xss event handler",
			address: "ltc1\" onclick=\"alert(1)",
			wantErr: ErrSQLInjectionDetected, // Caught by SQL injection first due to quote
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLitecoinAddress(tt.address)
			if err != tt.wantErr {
				t.Errorf("ValidateLitecoinAddress(%q) = %v, want %v", tt.address, err, tt.wantErr)
			}
		})
	}
}

func TestContainsSQLInjection(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"normal text", false},
		{"ltc1qgsm3fv44wprdcsh3trgarm05rr7l8ryggujr5w", false},
		{"' OR '1'='1", true},
		{"1; DROP TABLE users", true},
		{"UNION SELECT * FROM", true},
		{"admin'--", true},
		{"test; DELETE FROM users", true},
		{"SELECT password FROM users", true},
		{"INSERT INTO users", true},
		{"UPDATE users SET", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := containsSQLInjection(tt.input)
			if result != tt.expected {
				t.Errorf("containsSQLInjection(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestContainsXSS(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"normal text", false},
		{"ltc1qgsm3fv44wprdcsh3trgarm05rr7l8ryggujr5w", false},
		{"<script>alert(1)</script>", true},
		{"<img src=x onerror=alert(1)>", true},
		{"javascript:alert(1)", true},
		{"<svg onload=alert(1)>", true},
		{"<iframe src=evil.com>", true},
		{"onclick=alert(1)", true},
		{"document.cookie", true},
		{"window.location", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := containsXSS(tt.input)
			if result != tt.expected {
				t.Errorf("containsXSS(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"valid password", "SecurePass123!", false},
		{"too short", "Ab1!", true},
		{"no uppercase", "securepass123!", true},
		{"no lowercase", "SECUREPASS123!", true},
		{"no number", "SecurePassword!", true},
		{"no special", "SecurePass1234", true},
		{"all requirements met", "MyP@ssw0rd!", false},
		{"complex password", "C0mpl3x!P@ssw0rd#2024", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePassword(%q) error = %v, wantErr %v", tt.password, err, tt.wantErr)
			}
		})
	}
}

func TestValidateUsername(t *testing.T) {
	tests := []struct {
		name     string
		username string
		wantErr  bool
	}{
		{"valid username", "user123", false},
		{"valid with underscore", "user_name", false},
		{"valid with hyphen", "user-name", false},
		{"too short", "ab", true},
		{"sql injection", "admin'--", true},
		{"xss attempt", "<script>", true},
		{"special chars", "user@name", true},
		{"spaces", "user name", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUsername(tt.username)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUsername(%q) error = %v, wantErr %v", tt.username, err, tt.wantErr)
			}
		})
	}
}

func TestSanitizeInput(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal text", "normal text"},
		{"  trimmed  ", "trimmed"},
		{"with\x00null", "withnull"},
		{"control\x01chars", "controlchars"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := SanitizeInput(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeInput(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
