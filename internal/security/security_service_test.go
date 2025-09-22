package security

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecurityServiceIntegration(t *testing.T) {
	securityService := NewSecurityService(nil) // Use default config
	ctx := context.Background()

	t.Run("normal_request_validation", func(t *testing.T) {
		req := SecurityCheckRequest{
			UserID:    123,
			ClientID:  "normal-client",
			IPAddress: "192.168.1.1",
			UserAgent: "Mozilla/5.0",
			Action:    "api_call",
			Resource:  "mining",
			Input:     "normal user input",
		}

		result, err := securityService.ValidateRequest(ctx, req)
		require.NoError(t, err)
		assert.True(t, result.Allowed)
		assert.False(t, result.RateLimited)
		assert.False(t, result.BruteForceBlocked)
		assert.False(t, result.DDoSBlocked)
		assert.False(t, result.IntrusionDetected)
		assert.Empty(t, result.Violations)
	})

	t.Run("malicious_request_validation", func(t *testing.T) {
		req := SecurityCheckRequest{
			UserID:    123,
			ClientID:  "malicious-client",
			IPAddress: "192.168.1.100",
			UserAgent: "AttackBot/1.0",
			Action:    "api_call",
			Resource:  "mining",
			Input:     "'; DROP TABLE users; --",
		}

		result, err := securityService.ValidateRequest(ctx, req)
		require.NoError(t, err)
		assert.False(t, result.Allowed)
		assert.True(t, result.IntrusionDetected)
		assert.Contains(t, result.Violations, "malicious_input")
		assert.NotNil(t, result.ThreatInfo)
		assert.True(t, result.ThreatInfo.IsMalicious)
	})

	t.Run("mfa_setup_and_verification", func(t *testing.T) {
		userID := int64(456)

		// Setup MFA
		mfaInfo, err := securityService.SetupMFA(userID, "ChimeraPool", "test@example.com")
		require.NoError(t, err)
		assert.NotEmpty(t, mfaInfo.Secret)
		assert.NotEmpty(t, mfaInfo.QRCode)
		assert.Len(t, mfaInfo.BackupCodes, 10)

		// Enable MFA
		err = securityService.mfa.EnableMFA(userID, mfaInfo.Secret, mfaInfo.BackupCodes)
		require.NoError(t, err)

		// Generate and verify TOTP code
		currentCode := securityService.mfa.GenerateTOTPCode(mfaInfo.Secret, time.Now())
		valid, err := securityService.VerifyMFA(userID, currentCode)
		require.NoError(t, err)
		assert.True(t, valid)

		// Verify backup code
		valid, err = securityService.VerifyMFA(userID, mfaInfo.BackupCodes[0])
		require.NoError(t, err)
		assert.True(t, valid)

		// Same backup code should not work again
		valid, err = securityService.VerifyMFA(userID, mfaInfo.BackupCodes[0])
		require.NoError(t, err)
		assert.False(t, valid)
	})

	t.Run("secure_wallet_operations", func(t *testing.T) {
		// Create wallet
		walletID, err := securityService.CreateSecureWallet(
			"1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa",
			"5HueCGU8rMjxEXxiPuD5BDku4MkFqeZyd4dZ1jvhTVqvbTLvyTJ",
		)
		require.NoError(t, err)
		assert.NotEmpty(t, walletID)

		// Sign transaction
		transactionData := []byte("send 0.001 BTC to 1BvBMSEYstWetqTFn5Au4m4GFg7xJaNVN2")
		signature, err := securityService.SignTransaction(
			walletID,
			"5HueCGU8rMjxEXxiPuD5BDku4MkFqeZyd4dZ1jvhTVqvbTLvyTJ",
			transactionData,
		)
		require.NoError(t, err)
		assert.NotEmpty(t, signature)

		// Verify signature
		valid, err := securityService.secureWallet.VerifySignature(walletID, transactionData, signature)
		require.NoError(t, err)
		assert.True(t, valid)
	})

	t.Run("compliance_workflow", func(t *testing.T) {
		userID := int64(789)

		// Check compliance requirements
		requirements, err := securityService.CheckCompliance(userID, "US")
		require.NoError(t, err)
		assert.True(t, requirements.RequiresKYC)
		assert.True(t, requirements.RequiresAML)

		// Submit KYC
		kycData := KYCData{
			UserID:    userID,
			FirstName: "Jane",
			LastName:  "Doe",
			Email:     "jane.doe@example.com",
			Country:   "US",
		}
		err = securityService.SubmitKYC(kycData)
		require.NoError(t, err)

		// Perform AML screening
		amlResult, err := securityService.PerformAMLScreening(userID, "Jane Doe")
		require.NoError(t, err)
		assert.NotNil(t, amlResult)
		assert.False(t, amlResult.IsHighRisk)
	})

	t.Run("data_encryption", func(t *testing.T) {
		sensitiveData := "user private information"

		// Encrypt data
		encrypted, err := securityService.EncryptSensitiveData(sensitiveData)
		require.NoError(t, err)
		assert.NotEmpty(t, encrypted)
		assert.NotEqual(t, sensitiveData, encrypted)

		// Decrypt data
		decrypted, err := securityService.DecryptSensitiveData(encrypted)
		require.NoError(t, err)
		assert.Equal(t, sensitiveData, decrypted)
	})

	t.Run("client_blocking_status", func(t *testing.T) {
		clientID := "test-client"

		// Initially should not be blocked
		status, err := securityService.IsClientBlocked(ctx, clientID)
		require.NoError(t, err)
		assert.False(t, status.IsBlocked)
		assert.False(t, status.IntrusionBlocked)
		assert.False(t, status.DDoSBlocked)
		assert.False(t, status.Suspicious)

		// Generate malicious requests to trigger blocking
		for i := 0; i < 10; i++ {
			req := SecurityCheckRequest{
				UserID:    0,
				ClientID:  clientID,
				IPAddress: "192.168.1.200",
				Action:    "api_call",
				Resource:  "test",
				Input:     "SELECT * FROM users WHERE id = 1",
			}
			securityService.ValidateRequest(ctx, req)
		}

		// Should be blocked now
		status, err = securityService.IsClientBlocked(ctx, clientID)
		require.NoError(t, err)
		assert.True(t, status.IsBlocked)
		assert.True(t, status.IntrusionBlocked)
	})

	t.Run("authentication_attempt_tracking", func(t *testing.T) {
		clientID := "auth-client"

		// Record failed attempts
		for i := 0; i < 3; i++ {
			err := securityService.RecordAuthenticationAttempt(ctx, clientID, false)
			require.NoError(t, err)
		}

		// Should be blocked for authentication
		req := SecurityCheckRequest{
			UserID:    123,
			ClientID:  clientID,
			Action:    "login",
			Resource:  "auth",
		}

		result, err := securityService.ValidateRequest(ctx, req)
		require.NoError(t, err)
		assert.False(t, result.Allowed)
		assert.True(t, result.BruteForceBlocked)

		// Record successful attempt (should reset)
		err = securityService.RecordAuthenticationAttempt(ctx, clientID, true)
		require.NoError(t, err)

		// Should be allowed again
		result, err = securityService.ValidateRequest(ctx, req)
		require.NoError(t, err)
		assert.True(t, result.Allowed)
		assert.False(t, result.BruteForceBlocked)
	})

	t.Run("audit_logging", func(t *testing.T) {
		userID := int64(999)

		// Log some events
		events := []AuditEvent{
			{
				UserID:    userID,
				Action:    "login",
				Resource:  "auth",
				IPAddress: "192.168.1.1",
				Success:   true,
			},
			{
				UserID:    userID,
				Action:    "wallet_created",
				Resource:  "wallet",
				IPAddress: "192.168.1.1",
				Success:   true,
			},
			{
				UserID:    userID,
				Action:    "api_call",
				Resource:  "mining",
				IPAddress: "192.168.1.1",
				Success:   true,
			},
		}

		for _, event := range events {
			err := securityService.LogSecurityEvent(event)
			require.NoError(t, err)
		}

		// Retrieve audit logs
		logs, err := securityService.GetUserAuditLogs(userID, 10)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(logs), 3)

		// Verify events are logged
		actionMap := make(map[string]bool)
		for _, log := range logs {
			if log.UserID == userID {
				actionMap[log.Action] = true
			}
		}

		assert.True(t, actionMap["login"])
		assert.True(t, actionMap["wallet_created"])
		assert.True(t, actionMap["api_call"])
	})

	t.Run("security_metrics", func(t *testing.T) {
		metrics, err := securityService.GetSecurityMetrics(ctx)
		require.NoError(t, err)
		assert.NotNil(t, metrics)
		assert.Greater(t, metrics.TotalRequests, int64(0))
	})
}

func TestSecurityServiceConfiguration(t *testing.T) {
	// Test with custom configuration
	config := &SecurityConfig{
		RateLimiting: ProgressiveRateLimiterConfig{
			BaseRequestsPerMinute: 30,
			BaseBurstSize:         5,
			MaxPenaltyMultiplier:  5,
			PenaltyDuration:       30 * time.Minute,
			CleanupInterval:       time.Minute,
		},
		BruteForce: BruteForceConfig{
			MaxAttempts:     2,
			WindowDuration:  30 * time.Minute,
			LockoutDuration: 30 * time.Minute,
			CleanupInterval: time.Minute,
		},
		DDoS: DDoSConfig{
			RequestsPerSecond:   5,
			BurstSize:          10,
			SuspiciousThreshold: 50,
			BlockDuration:      30 * time.Minute,
			CleanupInterval:    time.Minute,
		},
		IntrusionDetection: IntrusionDetectionConfig{
			SuspiciousPatterns: []string{
				`(?i)(select|union)`,
			},
			MaxViolationsPerHour: 3,
			BlockDuration:        time.Hour,
			CleanupInterval:      time.Minute,
		},
	}

	securityService := NewSecurityService(config)
	ctx := context.Background()

	// Test that custom configuration is applied
	req := SecurityCheckRequest{
		UserID:    123,
		ClientID:  "config-test-client",
		IPAddress: "192.168.1.1",
		Action:    "api_call",
		Resource:  "test",
		Input:     "normal input",
	}

	// Make requests up to the custom burst limit
	for i := 0; i < 5; i++ {
		result, err := securityService.ValidateRequest(ctx, req)
		require.NoError(t, err)
		assert.True(t, result.Allowed, "request %d should be allowed", i+1)
	}

	// Next request should be rate limited (custom burst size is 5)
	result, err := securityService.ValidateRequest(ctx, req)
	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.True(t, result.RateLimited)
}

func TestSecurityServiceErrorHandling(t *testing.T) {
	securityService := NewSecurityService(nil)
	ctx := context.Background()

	t.Run("invalid_mfa_setup", func(t *testing.T) {
		// Test with invalid user ID
		_, err := securityService.SetupMFA(0, "ChimeraPool", "test@example.com")
		assert.Error(t, err)

		// Test with empty issuer
		_, err = securityService.SetupMFA(123, "", "test@example.com")
		assert.Error(t, err)

		// Test with empty account name
		_, err = securityService.SetupMFA(123, "ChimeraPool", "")
		assert.Error(t, err)
	})

	t.Run("invalid_wallet_creation", func(t *testing.T) {
		// Test with empty address
		_, err := securityService.CreateSecureWallet("", "privatekey")
		assert.Error(t, err)

		// Test with empty private key
		_, err = securityService.CreateSecureWallet("address", "")
		assert.Error(t, err)
	})

	t.Run("invalid_compliance_data", func(t *testing.T) {
		// Test with invalid KYC data
		invalidKYC := KYCData{
			UserID: 0, // Invalid user ID
		}
		err := securityService.SubmitKYC(invalidKYC)
		assert.Error(t, err)

		// Test with missing required fields
		invalidKYC = KYCData{
			UserID:    123,
			FirstName: "", // Missing first name
			LastName:  "Doe",
			Email:     "test@example.com",
		}
		err = securityService.SubmitKYC(invalidKYC)
		assert.Error(t, err)
	})
}