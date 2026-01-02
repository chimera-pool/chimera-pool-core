package validation

import (
	"errors"
	"regexp"
	"strings"
)

var (
	ErrInvalidWalletAddress  = errors.New("invalid wallet address format")
	ErrWalletAddressTooShort = errors.New("wallet address too short")
	ErrWalletAddressTooLong  = errors.New("wallet address too long")
	ErrInvalidCharacters     = errors.New("wallet address contains invalid characters")
	ErrSQLInjectionDetected  = errors.New("invalid characters detected in input")
	ErrXSSDetected           = errors.New("potentially malicious content detected")
)

// ValidateLitecoinAddress validates a Litecoin wallet address
// Litecoin addresses:
// - Legacy (P2PKH): Start with 'L', 26-35 characters
// - Legacy (P2SH): Start with 'M' or '3', 26-35 characters
// - Bech32 (SegWit): Start with 'ltc1', 42-62 characters
func ValidateLitecoinAddress(address string) error {
	address = strings.TrimSpace(address)

	if len(address) == 0 {
		return ErrInvalidWalletAddress
	}

	// Check for SQL injection patterns
	if containsSQLInjection(address) {
		return ErrSQLInjectionDetected
	}

	// Check for XSS patterns
	if containsXSS(address) {
		return ErrXSSDetected
	}

	// Bech32 address (SegWit)
	if strings.HasPrefix(strings.ToLower(address), "ltc1") {
		return validateBech32Address(address)
	}

	// Legacy address
	return validateLegacyAddress(address)
}

// validateBech32Address validates Litecoin Bech32 (SegWit) addresses
func validateBech32Address(address string) error {
	// Bech32 addresses are 42-62 characters
	if len(address) < 42 {
		return ErrWalletAddressTooShort
	}
	if len(address) > 62 {
		return ErrWalletAddressTooLong
	}

	// Must start with ltc1
	if !strings.HasPrefix(strings.ToLower(address), "ltc1") {
		return ErrInvalidWalletAddress
	}

	// Bech32 uses lowercase alphanumeric except 1, b, i, o
	// Valid characters: 023456789acdefghjklmnpqrstuvwxyz
	bech32Regex := regexp.MustCompile(`^ltc1[02-9ac-hj-np-z]{39,59}$`)
	if !bech32Regex.MatchString(strings.ToLower(address)) {
		return ErrInvalidCharacters
	}

	return nil
}

// validateLegacyAddress validates Litecoin legacy (P2PKH/P2SH) addresses
func validateLegacyAddress(address string) error {
	// Legacy addresses are 26-35 characters
	if len(address) < 26 {
		return ErrWalletAddressTooShort
	}
	if len(address) > 35 {
		return ErrWalletAddressTooLong
	}

	// Must start with L, M, or 3
	firstChar := address[0]
	if firstChar != 'L' && firstChar != 'M' && firstChar != '3' {
		return ErrInvalidWalletAddress
	}

	// Base58 characters only (no 0, O, I, l)
	base58Regex := regexp.MustCompile(`^[LM3][1-9A-HJ-NP-Za-km-z]{25,34}$`)
	if !base58Regex.MatchString(address) {
		return ErrInvalidCharacters
	}

	return nil
}

// containsSQLInjection checks for common SQL injection patterns
func containsSQLInjection(input string) bool {
	lowered := strings.ToLower(input)

	// Common SQL injection patterns
	patterns := []string{
		"'",
		"\"",
		";",
		"--",
		"/*",
		"*/",
		"union",
		"select",
		"insert",
		"update",
		"delete",
		"drop",
		"exec",
		"execute",
		"xp_",
		"sp_",
		"0x",
		"\\x",
		"char(",
		"nchar(",
		"varchar(",
		"nvarchar(",
		"cast(",
		"convert(",
		"table",
		"from",
		"where",
		"or 1=1",
		"or '1'='1",
		"or \"1\"=\"1",
		"1=1",
		"1 = 1",
	}

	for _, pattern := range patterns {
		if strings.Contains(lowered, pattern) {
			return true
		}
	}

	return false
}

// containsXSS checks for common XSS patterns
func containsXSS(input string) bool {
	lowered := strings.ToLower(input)

	// Common XSS patterns
	patterns := []string{
		"<script",
		"</script",
		"javascript:",
		"onerror",
		"onload",
		"onclick",
		"onmouseover",
		"onfocus",
		"onblur",
		"<img",
		"<iframe",
		"<object",
		"<embed",
		"<svg",
		"<math",
		"<video",
		"<audio",
		"<body",
		"<input",
		"<form",
		"<link",
		"<meta",
		"<style",
		"expression(",
		"url(",
		"eval(",
		"alert(",
		"confirm(",
		"prompt(",
		"document.",
		"window.",
		"<",
		">",
		"&lt;",
		"&gt;",
	}

	for _, pattern := range patterns {
		if strings.Contains(lowered, pattern) {
			return true
		}
	}

	return false
}

// SanitizeInput removes potentially dangerous characters from input
// Use this for fields that don't have strict format requirements
func SanitizeInput(input string) string {
	// Remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")

	// Trim whitespace
	input = strings.TrimSpace(input)

	// Remove control characters
	var result strings.Builder
	for _, r := range input {
		if r >= 32 && r != 127 {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// ValidateUsername validates username format
func ValidateUsername(username string) error {
	username = strings.TrimSpace(username)

	if len(username) < 3 {
		return errors.New("username must be at least 3 characters")
	}
	if len(username) > 30 {
		return errors.New("username must be at most 30 characters")
	}

	// Check for SQL injection
	if containsSQLInjection(username) {
		return ErrSQLInjectionDetected
	}

	// Check for XSS
	if containsXSS(username) {
		return ErrXSSDetected
	}

	// Only allow alphanumeric, underscore, hyphen
	usernameRegex := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !usernameRegex.MatchString(username) {
		return errors.New("username can only contain letters, numbers, underscores, and hyphens")
	}

	return nil
}

// ValidatePassword validates password strength
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	if len(password) > 128 {
		return errors.New("password must be at most 128 characters")
	}

	var hasUpper, hasLower, hasNumber, hasSpecial bool
	for _, c := range password {
		switch {
		case c >= 'A' && c <= 'Z':
			hasUpper = true
		case c >= 'a' && c <= 'z':
			hasLower = true
		case c >= '0' && c <= '9':
			hasNumber = true
		case strings.ContainsRune("!@#$%^&*()_+-=[]{}|;':\",./<>?`~", c):
			hasSpecial = true
		}
	}

	if !hasUpper {
		return errors.New("password must contain at least one uppercase letter")
	}
	if !hasLower {
		return errors.New("password must contain at least one lowercase letter")
	}
	if !hasNumber {
		return errors.New("password must contain at least one number")
	}
	if !hasSpecial {
		return errors.New("password must contain at least one special character (!@#$%^&*)")
	}

	return nil
}
