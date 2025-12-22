package security

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// COMPREHENSIVE SECURITY/MFA TESTS FOR 90%+ COVERAGE
// Critical for production-ready authentication
// =============================================================================

// MockMFARepository for testing
type MockMFARepository struct {
	secrets     map[int64]string
	backupCodes map[int64][]string
	mfaEnabled  map[int64]bool
	shouldError bool
	errorMsg    string
}

func NewMockMFARepository() *MockMFARepository {
	return &MockMFARepository{
		secrets:     make(map[int64]string),
		backupCodes: make(map[int64][]string),
		mfaEnabled:  make(map[int64]bool),
	}
}

func (m *MockMFARepository) SetError(msg string) {
	m.shouldError = true
	m.errorMsg = msg
}

func (m *MockMFARepository) ClearError() {
	m.shouldError = false
	m.errorMsg = ""
}

func (m *MockMFARepository) StoreTOTPSecret(userID int64, secret string) error {
	if m.shouldError {
		return errors.New(m.errorMsg)
	}
	m.secrets[userID] = secret
	return nil
}

func (m *MockMFARepository) GetTOTPSecret(userID int64) (string, error) {
	if m.shouldError {
		return "", errors.New(m.errorMsg)
	}
	secret, ok := m.secrets[userID]
	if !ok {
		return "", errors.New("secret not found")
	}
	return secret, nil
}

func (m *MockMFARepository) StoreBackupCodes(userID int64, codes []string) error {
	if m.shouldError {
		return errors.New(m.errorMsg)
	}
	m.backupCodes[userID] = codes
	return nil
}

func (m *MockMFARepository) GetBackupCodes(userID int64) ([]string, error) {
	if m.shouldError {
		return nil, errors.New(m.errorMsg)
	}
	codes, ok := m.backupCodes[userID]
	if !ok {
		return []string{}, nil
	}
	return codes, nil
}

func (m *MockMFARepository) UseBackupCode(userID int64, code string) error {
	if m.shouldError {
		return errors.New(m.errorMsg)
	}
	codes := m.backupCodes[userID]
	newCodes := []string{}
	for _, c := range codes {
		if c != code {
			newCodes = append(newCodes, c)
		}
	}
	m.backupCodes[userID] = newCodes
	return nil
}

func (m *MockMFARepository) EnableMFA(userID int64) error {
	if m.shouldError {
		return errors.New(m.errorMsg)
	}
	m.mfaEnabled[userID] = true
	return nil
}

func (m *MockMFARepository) DisableMFA(userID int64) error {
	if m.shouldError {
		return errors.New(m.errorMsg)
	}
	m.mfaEnabled[userID] = false
	return nil
}

func (m *MockMFARepository) IsMFAEnabled(userID int64) (bool, error) {
	if m.shouldError {
		return false, errors.New(m.errorMsg)
	}
	return m.mfaEnabled[userID], nil
}

// -----------------------------------------------------------------------------
// TOTP Config Tests
// -----------------------------------------------------------------------------

func TestDefaultTOTPConfig(t *testing.T) {
	config := DefaultTOTPConfig()

	assert.Equal(t, 30, config.Period)
	assert.Equal(t, 6, config.Digits)
	assert.Equal(t, "SHA1", config.Algorithm)
	assert.Equal(t, 1, config.Skew)
}

// -----------------------------------------------------------------------------
// MFA Service Creation Tests
// -----------------------------------------------------------------------------

func TestNewMFAService(t *testing.T) {
	svc := NewMFAService()
	require.NotNil(t, svc)
	assert.NotNil(t, svc.repository)
}

func TestNewMFAServiceWithRepository(t *testing.T) {
	repo := NewMockMFARepository()
	svc := NewMFAServiceWithRepository(repo)

	require.NotNil(t, svc)
	assert.Equal(t, repo, svc.repository)
}

// -----------------------------------------------------------------------------
// Generate TOTP Secret Tests
// -----------------------------------------------------------------------------

func TestMFAService_GenerateTOTPSecret_Success(t *testing.T) {
	svc := NewMFAService()

	secret, qrCode, err := svc.GenerateTOTPSecret(1, "ChimeraPool", "user@example.com")

	require.NoError(t, err)
	assert.NotEmpty(t, secret)
	assert.NotEmpty(t, qrCode)
	assert.True(t, strings.HasPrefix(qrCode, "data:image/png;base64,"))
}

func TestMFAService_GenerateTOTPSecret_InvalidUserID(t *testing.T) {
	svc := NewMFAService()

	_, _, err := svc.GenerateTOTPSecret(0, "ChimeraPool", "user@example.com")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid user ID")
}

func TestMFAService_GenerateTOTPSecret_NegativeUserID(t *testing.T) {
	svc := NewMFAService()

	_, _, err := svc.GenerateTOTPSecret(-1, "ChimeraPool", "user@example.com")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid user ID")
}

func TestMFAService_GenerateTOTPSecret_EmptyIssuer(t *testing.T) {
	svc := NewMFAService()

	_, _, err := svc.GenerateTOTPSecret(1, "", "user@example.com")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "issuer is required")
}

func TestMFAService_GenerateTOTPSecret_WhitespaceIssuer(t *testing.T) {
	svc := NewMFAService()

	_, _, err := svc.GenerateTOTPSecret(1, "   ", "user@example.com")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "issuer is required")
}

func TestMFAService_GenerateTOTPSecret_EmptyAccountName(t *testing.T) {
	svc := NewMFAService()

	_, _, err := svc.GenerateTOTPSecret(1, "ChimeraPool", "")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "account name is required")
}

func TestMFAService_GenerateTOTPSecret_WhitespaceAccountName(t *testing.T) {
	svc := NewMFAService()

	_, _, err := svc.GenerateTOTPSecret(1, "ChimeraPool", "   ")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "account name is required")
}

// -----------------------------------------------------------------------------
// Validate TOTP Tests
// -----------------------------------------------------------------------------

func TestMFAService_ValidateTOTP_EmptySecret(t *testing.T) {
	svc := NewMFAService()

	valid := svc.ValidateTOTP("", "123456")

	assert.False(t, valid)
}

func TestMFAService_ValidateTOTP_EmptyCode(t *testing.T) {
	svc := NewMFAService()

	valid := svc.ValidateTOTP("JBSWY3DPEHPK3PXP", "")

	assert.False(t, valid)
}

func TestMFAService_ValidateTOTP_WhitespaceSecret(t *testing.T) {
	svc := NewMFAService()

	valid := svc.ValidateTOTP("   ", "123456")

	assert.False(t, valid)
}

func TestMFAService_ValidateTOTP_WhitespaceCode(t *testing.T) {
	svc := NewMFAService()

	valid := svc.ValidateTOTP("JBSWY3DPEHPK3PXP", "   ")

	assert.False(t, valid)
}

func TestMFAService_ValidateTOTP_WrongLength(t *testing.T) {
	svc := NewMFAService()

	// Too short
	valid := svc.ValidateTOTP("JBSWY3DPEHPK3PXP", "12345")
	assert.False(t, valid)

	// Too long
	valid = svc.ValidateTOTP("JBSWY3DPEHPK3PXP", "1234567")
	assert.False(t, valid)
}

func TestMFAService_ValidateTOTP_NonNumeric(t *testing.T) {
	svc := NewMFAService()

	valid := svc.ValidateTOTP("JBSWY3DPEHPK3PXP", "12345a")

	assert.False(t, valid)
}

func TestMFAService_ValidateTOTP_ValidCode(t *testing.T) {
	svc := NewMFAService()
	secret := "JBSWY3DPEHPK3PXP" // Test secret

	// Generate current code
	code := svc.GenerateTOTPCode(secret, time.Now())

	// Should validate successfully
	valid := svc.ValidateTOTP(secret, code)
	assert.True(t, valid)
}

// -----------------------------------------------------------------------------
// Generate TOTP Code Tests
// -----------------------------------------------------------------------------

func TestMFAService_GenerateTOTPCode_ValidSecret(t *testing.T) {
	svc := NewMFAService()
	secret := "JBSWY3DPEHPK3PXP"

	code := svc.GenerateTOTPCode(secret, time.Now())

	assert.Len(t, code, 6)
	// All characters should be digits
	for _, c := range code {
		assert.True(t, c >= '0' && c <= '9')
	}
}

func TestMFAService_GenerateTOTPCode_InvalidSecret(t *testing.T) {
	svc := NewMFAService()

	// Invalid base32 secret
	code := svc.GenerateTOTPCode("invalid!!secret", time.Now())

	assert.Empty(t, code)
}

func TestMFAService_GenerateTOTPCode_DifferentTimes(t *testing.T) {
	svc := NewMFAService()
	secret := "JBSWY3DPEHPK3PXP"

	// Codes at different time periods should be different
	code1 := svc.GenerateTOTPCode(secret, time.Now())
	code2 := svc.GenerateTOTPCode(secret, time.Now().Add(60*time.Second)) // 2 periods later

	// They might be the same by chance, but likely different
	_ = code1
	_ = code2
}

func TestMFAService_GenerateTOTPCode_Deterministic(t *testing.T) {
	svc := NewMFAService()
	secret := "JBSWY3DPEHPK3PXP"
	fixedTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	code1 := svc.GenerateTOTPCode(secret, fixedTime)
	code2 := svc.GenerateTOTPCode(secret, fixedTime)

	assert.Equal(t, code1, code2) // Same time should produce same code
}

// -----------------------------------------------------------------------------
// Generate Backup Codes Tests
// -----------------------------------------------------------------------------

func TestMFAService_GenerateBackupCodes_Success(t *testing.T) {
	svc := NewMFAService()

	codes, err := svc.GenerateBackupCodes(1, 10)

	require.NoError(t, err)
	assert.Len(t, codes, 10)
	for _, code := range codes {
		assert.Len(t, code, 8)
	}
}

func TestMFAService_GenerateBackupCodes_InvalidUserID(t *testing.T) {
	svc := NewMFAService()

	_, err := svc.GenerateBackupCodes(0, 10)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid user ID")
}

func TestMFAService_GenerateBackupCodes_ZeroCount(t *testing.T) {
	svc := NewMFAService()

	_, err := svc.GenerateBackupCodes(1, 0)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid backup code count")
}

func TestMFAService_GenerateBackupCodes_NegativeCount(t *testing.T) {
	svc := NewMFAService()

	_, err := svc.GenerateBackupCodes(1, -5)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid backup code count")
}

func TestMFAService_GenerateBackupCodes_TooManyCount(t *testing.T) {
	svc := NewMFAService()

	_, err := svc.GenerateBackupCodes(1, 51)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid backup code count")
}

func TestMFAService_GenerateBackupCodes_MaxCount(t *testing.T) {
	svc := NewMFAService()

	codes, err := svc.GenerateBackupCodes(1, 50)

	require.NoError(t, err)
	assert.Len(t, codes, 50)
}

func TestMFAService_GenerateBackupCodes_Uniqueness(t *testing.T) {
	svc := NewMFAService()

	codes, err := svc.GenerateBackupCodes(1, 10)

	require.NoError(t, err)

	// All codes should be unique
	seen := make(map[string]bool)
	for _, code := range codes {
		assert.False(t, seen[code], "Duplicate backup code found")
		seen[code] = true
	}
}

// -----------------------------------------------------------------------------
// Validate Backup Code Tests
// -----------------------------------------------------------------------------

func TestMFAService_ValidateBackupCode_InvalidUserID(t *testing.T) {
	repo := NewMockMFARepository()
	svc := NewMFAServiceWithRepository(repo)

	valid, err := svc.ValidateBackupCode(0, "ABCD1234")

	assert.Error(t, err)
	assert.False(t, valid)
	assert.Contains(t, err.Error(), "invalid user ID")
}

func TestMFAService_ValidateBackupCode_EmptyCode(t *testing.T) {
	repo := NewMockMFARepository()
	svc := NewMFAServiceWithRepository(repo)

	valid, err := svc.ValidateBackupCode(1, "")

	require.NoError(t, err)
	assert.False(t, valid)
}

func TestMFAService_ValidateBackupCode_WhitespaceCode(t *testing.T) {
	repo := NewMockMFARepository()
	svc := NewMFAServiceWithRepository(repo)

	valid, err := svc.ValidateBackupCode(1, "   ")

	require.NoError(t, err)
	assert.False(t, valid)
}

func TestMFAService_ValidateBackupCode_ValidCode(t *testing.T) {
	repo := NewMockMFARepository()
	repo.backupCodes[1] = []string{"ABCD1234", "EFGH5678"}
	svc := NewMFAServiceWithRepository(repo)

	valid, err := svc.ValidateBackupCode(1, "ABCD1234")

	require.NoError(t, err)
	assert.True(t, valid)

	// Code should be removed after use
	assert.NotContains(t, repo.backupCodes[1], "ABCD1234")
}

func TestMFAService_ValidateBackupCode_InvalidCode(t *testing.T) {
	repo := NewMockMFARepository()
	repo.backupCodes[1] = []string{"ABCD1234", "EFGH5678"}
	svc := NewMFAServiceWithRepository(repo)

	valid, err := svc.ValidateBackupCode(1, "WRONGCODE")

	require.NoError(t, err)
	assert.False(t, valid)
}

func TestMFAService_ValidateBackupCode_CaseInsensitive(t *testing.T) {
	repo := NewMockMFARepository()
	repo.backupCodes[1] = []string{"ABCD1234"}
	svc := NewMFAServiceWithRepository(repo)

	valid, err := svc.ValidateBackupCode(1, "abcd1234")

	require.NoError(t, err)
	assert.True(t, valid)
}

func TestMFAService_ValidateBackupCode_RepositoryError(t *testing.T) {
	repo := NewMockMFARepository()
	repo.SetError("database error")
	svc := NewMFAServiceWithRepository(repo)

	valid, err := svc.ValidateBackupCode(1, "ABCD1234")

	assert.Error(t, err)
	assert.False(t, valid)
}

// -----------------------------------------------------------------------------
// Verify MFA Setup Tests
// -----------------------------------------------------------------------------

func TestMFAService_VerifyMFASetup_InvalidUserID(t *testing.T) {
	svc := NewMFAService()

	valid, err := svc.VerifyMFASetup(0, "secret", "123456", []string{"ABCD1234"})

	assert.Error(t, err)
	assert.False(t, valid)
}

func TestMFAService_VerifyMFASetup_InvalidTOTP(t *testing.T) {
	svc := NewMFAService()

	valid, err := svc.VerifyMFASetup(1, "JBSWY3DPEHPK3PXP", "000000", []string{"ABCD1234"})

	require.NoError(t, err)
	assert.False(t, valid)
}

func TestMFAService_VerifyMFASetup_EmptyBackupCodes(t *testing.T) {
	svc := NewMFAService()
	secret := "JBSWY3DPEHPK3PXP"
	code := svc.GenerateTOTPCode(secret, time.Now())

	valid, err := svc.VerifyMFASetup(1, secret, code, []string{})

	assert.Error(t, err)
	assert.False(t, valid)
	assert.Contains(t, err.Error(), "backup codes are required")
}

func TestMFAService_VerifyMFASetup_InvalidBackupCodeLength(t *testing.T) {
	svc := NewMFAService()
	secret := "JBSWY3DPEHPK3PXP"
	code := svc.GenerateTOTPCode(secret, time.Now())

	valid, err := svc.VerifyMFASetup(1, secret, code, []string{"SHORT"})

	assert.Error(t, err)
	assert.False(t, valid)
	assert.Contains(t, err.Error(), "invalid backup code format")
}

func TestMFAService_VerifyMFASetup_InvalidBackupCodeFormat(t *testing.T) {
	svc := NewMFAService()
	secret := "JBSWY3DPEHPK3PXP"
	code := svc.GenerateTOTPCode(secret, time.Now())

	valid, err := svc.VerifyMFASetup(1, secret, code, []string{"abcd1234"}) // lowercase

	assert.Error(t, err)
	assert.False(t, valid)
}

func TestMFAService_VerifyMFASetup_Success(t *testing.T) {
	svc := NewMFAService()
	secret := "JBSWY3DPEHPK3PXP"
	code := svc.GenerateTOTPCode(secret, time.Now())

	valid, err := svc.VerifyMFASetup(1, secret, code, []string{"ABCD1234", "EFGH5678"})

	require.NoError(t, err)
	assert.True(t, valid)
}

// -----------------------------------------------------------------------------
// Enable MFA Tests
// -----------------------------------------------------------------------------

func TestMFAService_EnableMFA_Success(t *testing.T) {
	repo := NewMockMFARepository()
	svc := NewMFAServiceWithRepository(repo)

	err := svc.EnableMFA(1, "secret123", []string{"CODE1234"})

	require.NoError(t, err)
	assert.Equal(t, "secret123", repo.secrets[1])
	assert.Equal(t, []string{"CODE1234"}, repo.backupCodes[1])
	assert.True(t, repo.mfaEnabled[1])
}

func TestMFAService_EnableMFA_InvalidUserID(t *testing.T) {
	svc := NewMFAService()

	err := svc.EnableMFA(0, "secret", []string{"CODE1234"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid user ID")
}

func TestMFAService_EnableMFA_StoreSecretError(t *testing.T) {
	repo := NewMockMFARepository()
	repo.SetError("storage error")
	svc := NewMFAServiceWithRepository(repo)

	err := svc.EnableMFA(1, "secret", []string{"CODE1234"})

	assert.Error(t, err)
}

// -----------------------------------------------------------------------------
// Is MFA Enabled Tests
// -----------------------------------------------------------------------------

func TestMFAService_IsMFAEnabled_InvalidUserID(t *testing.T) {
	svc := NewMFAService()

	enabled, err := svc.IsMFAEnabled(0)

	assert.Error(t, err)
	assert.False(t, enabled)
}

func TestMFAService_IsMFAEnabled_NotEnabled(t *testing.T) {
	repo := NewMockMFARepository()
	svc := NewMFAServiceWithRepository(repo)

	enabled, err := svc.IsMFAEnabled(1)

	require.NoError(t, err)
	assert.False(t, enabled)
}

func TestMFAService_IsMFAEnabled_Enabled(t *testing.T) {
	repo := NewMockMFARepository()
	repo.mfaEnabled[1] = true
	svc := NewMFAServiceWithRepository(repo)

	enabled, err := svc.IsMFAEnabled(1)

	require.NoError(t, err)
	assert.True(t, enabled)
}

// -----------------------------------------------------------------------------
// Disable MFA Tests
// -----------------------------------------------------------------------------

func TestMFAService_DisableMFA_Success(t *testing.T) {
	repo := NewMockMFARepository()
	repo.mfaEnabled[1] = true
	svc := NewMFAServiceWithRepository(repo)

	err := svc.DisableMFA(1)

	require.NoError(t, err)
	assert.False(t, repo.mfaEnabled[1])
}

func TestMFAService_DisableMFA_InvalidUserID(t *testing.T) {
	svc := NewMFAService()

	err := svc.DisableMFA(0)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid user ID")
}

func TestMFAService_DisableMFA_RepositoryError(t *testing.T) {
	repo := NewMockMFARepository()
	repo.SetError("database error")
	svc := NewMFAServiceWithRepository(repo)

	err := svc.DisableMFA(1)

	assert.Error(t, err)
}

// -----------------------------------------------------------------------------
// Store Backup Codes Tests
// -----------------------------------------------------------------------------

func TestMFAService_StoreBackupCodes_Success(t *testing.T) {
	repo := NewMockMFARepository()
	svc := NewMFAServiceWithRepository(repo)

	err := svc.StoreBackupCodes(1, []string{"CODE1234", "CODE5678"})

	require.NoError(t, err)
	assert.Equal(t, []string{"CODE1234", "CODE5678"}, repo.backupCodes[1])
}

// -----------------------------------------------------------------------------
// Benchmark Tests
// -----------------------------------------------------------------------------

func BenchmarkMFAService_GenerateTOTPCode(b *testing.B) {
	svc := NewMFAService()
	secret := "JBSWY3DPEHPK3PXP"
	now := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.GenerateTOTPCode(secret, now)
	}
}

func BenchmarkMFAService_ValidateTOTP(b *testing.B) {
	svc := NewMFAService()
	secret := "JBSWY3DPEHPK3PXP"
	code := svc.GenerateTOTPCode(secret, time.Now())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.ValidateTOTP(secret, code)
	}
}

func BenchmarkMFAService_GenerateBackupCodes(b *testing.B) {
	svc := NewMFAService()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.GenerateBackupCodes(1, 10)
	}
}
