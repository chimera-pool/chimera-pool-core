package security

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/skip2/go-qrcode"
)

// MFAService handles multi-factor authentication operations
type MFAService struct {
	repository MFARepository
}

// MFARepository defines the interface for MFA data operations
type MFARepository interface {
	StoreTOTPSecret(userID int64, secret string) error
	GetTOTPSecret(userID int64) (string, error)
	StoreBackupCodes(userID int64, codes []string) error
	GetBackupCodes(userID int64) ([]string, error)
	UseBackupCode(userID int64, code string) error
	EnableMFA(userID int64) error
	DisableMFA(userID int64) error
	IsMFAEnabled(userID int64) (bool, error)
}

// TOTPConfig holds TOTP configuration
type TOTPConfig struct {
	Period    int    // Time period in seconds (default: 30)
	Digits    int    // Number of digits (default: 6)
	Algorithm string // Hash algorithm (default: SHA1)
	Skew      int    // Time skew tolerance in periods (default: 1)
}

// DefaultTOTPConfig returns default TOTP configuration
func DefaultTOTPConfig() *TOTPConfig {
	return &TOTPConfig{
		Period:    30,
		Digits:    6,
		Algorithm: "SHA1",
		Skew:      1,
	}
}

// NewMFAService creates a new MFA service
func NewMFAService() *MFAService {
	return &MFAService{
		repository: NewInMemoryMFARepository(), // Default to in-memory for testing
	}
}

// NewMFAServiceWithRepository creates a new MFA service with custom repository
func NewMFAServiceWithRepository(repo MFARepository) *MFAService {
	return &MFAService{
		repository: repo,
	}
}

// GenerateTOTPSecret generates a new TOTP secret and QR code for a user
func (s *MFAService) GenerateTOTPSecret(userID int64, issuer, accountName string) (string, string, error) {
	if userID <= 0 {
		return "", "", errors.New("invalid user ID")
	}
	if strings.TrimSpace(issuer) == "" {
		return "", "", errors.New("issuer is required")
	}
	if strings.TrimSpace(accountName) == "" {
		return "", "", errors.New("account name is required")
	}

	// Generate 32-byte random secret
	secretBytes := make([]byte, 20)
	_, err := rand.Read(secretBytes)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate secret: %w", err)
	}

	// Encode secret as base32
	secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secretBytes)

	// Generate QR code URL
	qrURL := fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s",
		issuer, accountName, secret, issuer)

	// Generate QR code image
	qrCode, err := s.generateQRCode(qrURL)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate QR code: %w", err)
	}

	return secret, qrCode, nil
}

// generateQRCode generates a base64-encoded PNG QR code
func (s *MFAService) generateQRCode(content string) (string, error) {
	qr, err := qrcode.Encode(content, qrcode.Medium, 256)
	if err != nil {
		return "", err
	}

	encoded := base64.StdEncoding.EncodeToString(qr)
	return "data:image/png;base64," + encoded, nil
}

// ValidateTOTP validates a TOTP code against a secret
func (s *MFAService) ValidateTOTP(secret, code string) bool {
	if strings.TrimSpace(secret) == "" || strings.TrimSpace(code) == "" {
		return false
	}

	// Validate code format (6 digits)
	if len(code) != 6 {
		return false
	}
	if matched, _ := regexp.MatchString(`^\d{6}$`, code); !matched {
		return false
	}

	config := DefaultTOTPConfig()
	now := time.Now()

	// Check current time and skew periods
	for i := -config.Skew; i <= config.Skew; i++ {
		testTime := now.Add(time.Duration(i*config.Period) * time.Second)
		expectedCode := s.GenerateTOTPCode(secret, testTime)
		if code == expectedCode {
			return true
		}
	}

	return false
}

// GenerateTOTPCode generates a TOTP code for a given secret and time
func (s *MFAService) GenerateTOTPCode(secret string, t time.Time) string {
	config := DefaultTOTPConfig()
	
	// Decode base32 secret
	secretBytes, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		return ""
	}

	// Calculate time counter
	counter := uint64(t.Unix()) / uint64(config.Period)

	// Generate HMAC-SHA1
	hash := sha1.New()
	counterBytes := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		counterBytes[i] = byte(counter & 0xff)
		counter >>= 8
	}

	// HMAC-SHA1(secret, counter)
	hash.Write(counterBytes)
	hmac := hash.Sum(secretBytes)

	// Dynamic truncation
	offset := hmac[len(hmac)-1] & 0x0f
	code := ((int(hmac[offset]) & 0x7f) << 24) |
		((int(hmac[offset+1]) & 0xff) << 16) |
		((int(hmac[offset+2]) & 0xff) << 8) |
		(int(hmac[offset+3]) & 0xff)

	// Generate 6-digit code
	code = code % 1000000

	return fmt.Sprintf("%06d", code)
}

// GenerateBackupCodes generates backup codes for a user
func (s *MFAService) GenerateBackupCodes(userID int64, count int) ([]string, error) {
	if userID <= 0 {
		return nil, errors.New("invalid user ID")
	}
	if count <= 0 || count > 50 {
		return nil, errors.New("invalid backup code count (must be 1-50)")
	}

	codes := make([]string, count)
	for i := 0; i < count; i++ {
		code, err := s.generateBackupCode()
		if err != nil {
			return nil, fmt.Errorf("failed to generate backup code: %w", err)
		}
		codes[i] = code
	}

	return codes, nil
}

// generateBackupCode generates a single 8-character backup code
func (s *MFAService) generateBackupCode() (string, error) {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const codeLength = 8

	code := make([]byte, codeLength)
	for i := range code {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		code[i] = charset[num.Int64()]
	}

	return string(code), nil
}

// StoreBackupCodes stores backup codes for a user
func (s *MFAService) StoreBackupCodes(userID int64, codes []string) error {
	return s.repository.StoreBackupCodes(userID, codes)
}

// ValidateBackupCode validates and uses a backup code
func (s *MFAService) ValidateBackupCode(userID int64, code string) (bool, error) {
	if userID <= 0 {
		return false, errors.New("invalid user ID")
	}
	if strings.TrimSpace(code) == "" {
		return false, nil
	}

	// Get stored backup codes
	storedCodes, err := s.repository.GetBackupCodes(userID)
	if err != nil {
		return false, fmt.Errorf("failed to get backup codes: %w", err)
	}

	// Check if code exists
	codeUpper := strings.ToUpper(strings.TrimSpace(code))
	for _, storedCode := range storedCodes {
		if storedCode == codeUpper {
			// Use the backup code (mark as used)
			err := s.repository.UseBackupCode(userID, codeUpper)
			if err != nil {
				return false, fmt.Errorf("failed to use backup code: %w", err)
			}
			return true, nil
		}
	}

	return false, nil
}

// VerifyMFASetup verifies the complete MFA setup process
func (s *MFAService) VerifyMFASetup(userID int64, secret, totpCode string, backupCodes []string) (bool, error) {
	if userID <= 0 {
		return false, errors.New("invalid user ID")
	}

	// Verify TOTP code
	if !s.ValidateTOTP(secret, totpCode) {
		return false, nil
	}

	// Verify backup codes format
	if len(backupCodes) == 0 {
		return false, errors.New("backup codes are required")
	}

	for _, code := range backupCodes {
		if len(code) != 8 {
			return false, errors.New("invalid backup code format")
		}
		if matched, _ := regexp.MatchString(`^[A-Z0-9]{8}$`, code); !matched {
			return false, errors.New("invalid backup code format")
		}
	}

	return true, nil
}

// EnableMFA enables MFA for a user
func (s *MFAService) EnableMFA(userID int64, secret string, backupCodes []string) error {
	if userID <= 0 {
		return errors.New("invalid user ID")
	}

	// Store TOTP secret
	err := s.repository.StoreTOTPSecret(userID, secret)
	if err != nil {
		return fmt.Errorf("failed to store TOTP secret: %w", err)
	}

	// Store backup codes
	err = s.repository.StoreBackupCodes(userID, backupCodes)
	if err != nil {
		return fmt.Errorf("failed to store backup codes: %w", err)
	}

	// Enable MFA
	err = s.repository.EnableMFA(userID)
	if err != nil {
		return fmt.Errorf("failed to enable MFA: %w", err)
	}

	return nil
}

// IsMFAEnabled checks if MFA is enabled for a user
func (s *MFAService) IsMFAEnabled(userID int64) (bool, error) {
	if userID <= 0 {
		return false, errors.New("invalid user ID")
	}

	return s.repository.IsMFAEnabled(userID)
}

// DisableMFA disables MFA for a user
func (s *MFAService) DisableMFA(userID int64) error {
	if userID <= 0 {
		return errors.New("invalid user ID")
	}

	return s.repository.DisableMFA(userID)
}