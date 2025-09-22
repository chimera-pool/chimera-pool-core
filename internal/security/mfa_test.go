package security

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTOTPGeneration(t *testing.T) {
	tests := []struct {
		name        string
		userID      int64
		issuer      string
		accountName string
		wantError   bool
	}{
		{
			name:        "valid TOTP generation",
			userID:      123,
			issuer:      "ChimeraPool",
			accountName: "user@example.com",
			wantError:   false,
		},
		{
			name:        "empty issuer should fail",
			userID:      123,
			issuer:      "",
			accountName: "user@example.com",
			wantError:   true,
		},
		{
			name:        "empty account name should fail",
			userID:      123,
			issuer:      "ChimeraPool",
			accountName: "",
			wantError:   true,
		},
		{
			name:        "invalid user ID should fail",
			userID:      0,
			issuer:      "ChimeraPool",
			accountName: "user@example.com",
			wantError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mfaService := NewMFAService()
			
			secret, qrCode, err := mfaService.GenerateTOTPSecret(tt.userID, tt.issuer, tt.accountName)
			
			if tt.wantError {
				assert.Error(t, err)
				assert.Empty(t, secret)
				assert.Empty(t, qrCode)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, secret)
				assert.NotEmpty(t, qrCode)
				assert.Contains(t, qrCode, "data:image/png;base64,")
				
				// Secret should be base32 encoded and 32 characters
				assert.Len(t, secret, 32)
			}
		})
	}
}

func TestTOTPValidation(t *testing.T) {
	mfaService := NewMFAService()
	
	// Generate a secret for testing
	secret, _, err := mfaService.GenerateTOTPSecret(123, "ChimeraPool", "test@example.com")
	require.NoError(t, err)
	
	tests := []struct {
		name      string
		secret    string
		code      string
		wantValid bool
	}{
		{
			name:      "empty secret should be invalid",
			secret:    "",
			code:      "123456",
			wantValid: false,
		},
		{
			name:      "empty code should be invalid",
			secret:    secret,
			code:      "",
			wantValid: false,
		},
		{
			name:      "invalid code format should be invalid",
			secret:    secret,
			code:      "12345",
			wantValid: false,
		},
		{
			name:      "non-numeric code should be invalid",
			secret:    secret,
			code:      "abcdef",
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := mfaService.ValidateTOTP(tt.secret, tt.code)
			assert.Equal(t, tt.wantValid, valid)
		})
	}
}

func TestTOTPValidationWithCurrentTime(t *testing.T) {
	mfaService := NewMFAService()
	
	// Generate a secret for testing
	secret, _, err := mfaService.GenerateTOTPSecret(123, "ChimeraPool", "test@example.com")
	require.NoError(t, err)
	
	// Generate current TOTP code
	currentCode := mfaService.GenerateTOTPCode(secret, time.Now())
	
	// Current code should be valid
	assert.True(t, mfaService.ValidateTOTP(secret, currentCode))
	
	// Code from 30 seconds ago should still be valid (time window)
	pastCode := mfaService.GenerateTOTPCode(secret, time.Now().Add(-30*time.Second))
	assert.True(t, mfaService.ValidateTOTP(secret, pastCode))
	
	// Code from 2 minutes ago should be invalid
	oldCode := mfaService.GenerateTOTPCode(secret, time.Now().Add(-2*time.Minute))
	assert.False(t, mfaService.ValidateTOTP(secret, oldCode))
}

func TestBackupCodesGeneration(t *testing.T) {
	tests := []struct {
		name      string
		userID    int64
		count     int
		wantError bool
	}{
		{
			name:      "valid backup codes generation",
			userID:    123,
			count:     10,
			wantError: false,
		},
		{
			name:      "invalid user ID should fail",
			userID:    0,
			count:     10,
			wantError: true,
		},
		{
			name:      "invalid count should fail",
			userID:    123,
			count:     0,
			wantError: true,
		},
		{
			name:      "too many codes should fail",
			userID:    123,
			count:     100,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mfaService := NewMFAService()
			
			codes, err := mfaService.GenerateBackupCodes(tt.userID, tt.count)
			
			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, codes)
			} else {
				assert.NoError(t, err)
				assert.Len(t, codes, tt.count)
				
				// Each code should be unique and properly formatted
				codeMap := make(map[string]bool)
				for _, code := range codes {
					assert.Len(t, code, 8) // 8-character backup codes
					assert.Regexp(t, `^[A-Z0-9]{8}$`, code)
					assert.False(t, codeMap[code], "duplicate backup code found")
					codeMap[code] = true
				}
			}
		})
	}
}

func TestBackupCodeValidation(t *testing.T) {
	mfaService := NewMFAService()
	userID := int64(123)
	
	// Generate backup codes
	codes, err := mfaService.GenerateBackupCodes(userID, 5)
	require.NoError(t, err)
	require.Len(t, codes, 5)
	
	// Store backup codes (simulate database storage)
	err = mfaService.StoreBackupCodes(userID, codes)
	require.NoError(t, err)
	
	tests := []struct {
		name      string
		userID    int64
		code      string
		wantValid bool
	}{
		{
			name:      "valid backup code should work",
			userID:    userID,
			code:      codes[0],
			wantValid: true,
		},
		{
			name:      "invalid backup code should fail",
			userID:    userID,
			code:      "INVALID1",
			wantValid: false,
		},
		{
			name:      "empty code should fail",
			userID:    userID,
			code:      "",
			wantValid: false,
		},
		{
			name:      "wrong user ID should fail",
			userID:    999,
			code:      codes[1],
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := mfaService.ValidateBackupCode(tt.userID, tt.code)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantValid, valid)
		})
	}
	
	// Used backup code should not work again
	valid, err := mfaService.ValidateBackupCode(userID, codes[0])
	assert.NoError(t, err)
	assert.False(t, valid, "used backup code should not be valid again")
}

func TestMFASetupWorkflow(t *testing.T) {
	mfaService := NewMFAService()
	userID := int64(123)
	
	// Step 1: Generate TOTP secret
	secret, qrCode, err := mfaService.GenerateTOTPSecret(userID, "ChimeraPool", "test@example.com")
	require.NoError(t, err)
	require.NotEmpty(t, secret)
	require.NotEmpty(t, qrCode)
	
	// Step 2: Generate backup codes
	backupCodes, err := mfaService.GenerateBackupCodes(userID, 10)
	require.NoError(t, err)
	require.Len(t, backupCodes, 10)
	
	// Step 3: Verify TOTP setup
	currentCode := mfaService.GenerateTOTPCode(secret, time.Now())
	setupValid, err := mfaService.VerifyMFASetup(userID, secret, currentCode, backupCodes)
	require.NoError(t, err)
	assert.True(t, setupValid)
	
	// Step 4: Enable MFA for user
	err = mfaService.EnableMFA(userID, secret, backupCodes)
	require.NoError(t, err)
	
	// Step 5: Verify MFA is enabled
	enabled, err := mfaService.IsMFAEnabled(userID)
	require.NoError(t, err)
	assert.True(t, enabled)
}