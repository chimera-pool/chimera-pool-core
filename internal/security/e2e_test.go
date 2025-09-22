package security

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompleteSecurityWorkflow tests the entire security framework integration
func TestCompleteSecurityWorkflow(t *testing.T) {
	// Initialize all security components
	mfaService := NewMFAService()
	rateLimiter := NewProgressiveRateLimiter(ProgressiveRateLimiterConfig{
		BaseRequestsPerMinute: 60,
		BaseBurstSize:         10,
		MaxPenaltyMultiplier:  5,
		PenaltyDuration:       time.Minute,
		CleanupInterval:       time.Minute,
	})
	bruteForceProtector := NewBruteForceProtector(BruteForceConfig{
		MaxAttempts:     3,
		WindowDuration:  time.Minute,
		LockoutDuration: 5 * time.Minute,
		CleanupInterval: time.Minute,
	})
	ddosProtector := NewDDoSProtector(DDoSConfig{
		RequestsPerSecond:   10,
		BurstSize:          20,
		SuspiciousThreshold: 100,
		BlockDuration:      time.Minute,
		CleanupInterval:    time.Minute,
	})
	intrusionDetector := NewIntrusionDetector(IntrusionDetectionConfig{
		SuspiciousPatterns: []string{
			`(?i)(union|select|insert|delete)`,
			`(?i)(<script|javascript)`,
		},
		MaxViolationsPerHour: 5,
		BlockDuration:        time.Hour,
		CleanupInterval:      time.Minute,
	})
	secureWallet := NewSecureWallet()
	compliance := NewComplianceManager()
	auditor := NewAuditLogger()

	ctx := context.Background()
	userID := int64(12345)
	clientID := "test-client-ip"

	t.Run("user_registration_and_mfa_setup", func(t *testing.T) {
		// Step 1: User registration with MFA setup
		secret, qrCode, err := mfaService.GenerateTOTPSecret(userID, "ChimeraPool", "user@example.com")
		require.NoError(t, err)
		assert.NotEmpty(t, secret)
		assert.NotEmpty(t, qrCode)

		// Generate backup codes
		backupCodes, err := mfaService.GenerateBackupCodes(userID, 10)
		require.NoError(t, err)
		assert.Len(t, backupCodes, 10)

		// Verify MFA setup
		currentCode := mfaService.GenerateTOTPCode(secret, time.Now())
		setupValid, err := mfaService.VerifyMFASetup(userID, secret, currentCode, backupCodes)
		require.NoError(t, err)
		assert.True(t, setupValid)

		// Enable MFA
		err = mfaService.EnableMFA(userID, secret, backupCodes)
		require.NoError(t, err)

		// Verify MFA is enabled
		enabled, err := mfaService.IsMFAEnabled(userID)
		require.NoError(t, err)
		assert.True(t, enabled)

		// Log audit event
		err = auditor.LogEvent(AuditEvent{
			UserID:    userID,
			Action:    "mfa_enabled",
			Resource:  "auth",
			IPAddress: "192.168.1.1",
			Success:   true,
		})
		require.NoError(t, err)
	})

	t.Run("secure_wallet_creation", func(t *testing.T) {
		// Create secure wallet
		walletAddress := "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa"
		privateKey := "5HueCGU8rMjxEXxiPuD5BDku4MkFqeZyd4dZ1jvhTVqvbTLvyTJ"

		walletID, err := secureWallet.CreateWallet(walletAddress, privateKey)
		require.NoError(t, err)
		assert.NotEmpty(t, walletID)

		// Verify wallet
		valid, err := secureWallet.VerifyWallet(walletID, privateKey)
		require.NoError(t, err)
		assert.True(t, valid)

		// Test transaction signing
		transactionData := []byte("send 0.001 BTC to 1BvBMSEYstWetqTFn5Au4m4GFg7xJaNVN2")
		signature, err := secureWallet.SignTransaction(walletID, privateKey, transactionData)
		require.NoError(t, err)
		assert.NotEmpty(t, signature)

		// Verify signature
		valid, err = secureWallet.VerifySignature(walletID, transactionData, signature)
		require.NoError(t, err)
		assert.True(t, valid)

		// Log audit event
		err = auditor.LogEvent(AuditEvent{
			UserID:    userID,
			Action:    "wallet_created",
			Resource:  "wallet",
			IPAddress: "192.168.1.1",
			Success:   true,
			Metadata: map[string]interface{}{
				"wallet_id": walletID,
				"address":   walletAddress,
			},
		})
		require.NoError(t, err)
	})

	t.Run("compliance_workflow", func(t *testing.T) {
		// Check compliance requirements
		requirements, err := compliance.GetComplianceRequirements(userID, "US")
		require.NoError(t, err)
		assert.True(t, requirements.RequiresKYC)
		assert.True(t, requirements.RequiresAML)

		// Submit KYC
		kycData := KYCData{
			UserID:    userID,
			FirstName: "John",
			LastName:  "Doe",
			Email:     "john.doe@example.com",
			Country:   "US",
		}
		err = compliance.SubmitKYC(kycData)
		require.NoError(t, err)

		// Check KYC status
		status, err := compliance.GetKYCStatus(userID)
		require.NoError(t, err)
		assert.Equal(t, KYCStatusPending, status)

		// Perform AML screening
		amlResult, err := compliance.PerformAMLScreening(userID, "John Doe")
		require.NoError(t, err)
		assert.NotNil(t, amlResult)
		assert.False(t, amlResult.IsHighRisk) // "John Doe" should not be high risk

		// Log audit events
		err = auditor.LogEvent(AuditEvent{
			UserID:    userID,
			Action:    "kyc_submitted",
			Resource:  "compliance",
			IPAddress: "192.168.1.1",
			Success:   true,
		})
		require.NoError(t, err)

		err = auditor.LogEvent(AuditEvent{
			UserID:    userID,
			Action:    "aml_screening",
			Resource:  "compliance",
			IPAddress: "192.168.1.1",
			Success:   true,
			Metadata: map[string]interface{}{
				"risk_score":   amlResult.RiskScore,
				"is_high_risk": amlResult.IsHighRisk,
			},
		})
		require.NoError(t, err)
	})

	t.Run("normal_user_activity", func(t *testing.T) {
		// Normal API requests should be allowed
		for i := 0; i < 5; i++ {
			// Check rate limiting
			allowed, err := rateLimiter.Allow(ctx, clientID)
			require.NoError(t, err)
			assert.True(t, allowed, "normal request %d should be allowed", i+1)

			// Check DDoS protection
			allowed, err = ddosProtector.CheckRequest(ctx, clientID)
			require.NoError(t, err)
			assert.True(t, allowed, "normal request %d should pass DDoS protection", i+1)

			// Check intrusion detection with normal input
			threat, err := intrusionDetector.AnalyzeRequest(ctx, clientID, "normal user input")
			require.NoError(t, err)
			assert.False(t, threat.IsMalicious, "normal input should not be flagged")
		}

		// Log successful activity
		err := auditor.LogEvent(AuditEvent{
			UserID:    userID,
			Action:    "api_request",
			Resource:  "api",
			IPAddress: "192.168.1.1",
			Success:   true,
		})
		require.NoError(t, err)
	})

	t.Run("attack_scenarios_and_protection", func(t *testing.T) {
		attackerClientID := "attacker-ip"

		// Scenario 1: Brute force attack
		t.Run("brute_force_attack", func(t *testing.T) {
			// Simulate failed login attempts
			for i := 0; i < 3; i++ {
				allowed, err := bruteForceProtector.CheckAttempt(ctx, attackerClientID)
				require.NoError(t, err)
				assert.True(t, allowed, "attempt %d should be allowed", i+1)

				err = bruteForceProtector.RecordFailedAttempt(ctx, attackerClientID)
				require.NoError(t, err)

				// Log failed attempt
				err = auditor.LogEvent(AuditEvent{
					UserID:    userID,
					Action:    "login_failed",
					Resource:  "auth",
					IPAddress: attackerClientID,
					Success:   false,
					Error:     "invalid credentials",
				})
				require.NoError(t, err)
			}

			// Should be locked out now
			allowed, err := bruteForceProtector.CheckAttempt(ctx, attackerClientID)
			require.NoError(t, err)
			assert.False(t, allowed, "should be locked out after max attempts")

			// Record violation in rate limiter
			err = rateLimiter.RecordViolation(ctx, attackerClientID, ViolationTypeBruteForce)
			require.NoError(t, err)
		})

		// Scenario 2: SQL injection attempt
		t.Run("sql_injection_attack", func(t *testing.T) {
			maliciousInputs := []string{
				"'; DROP TABLE users; --",
				"' UNION SELECT * FROM passwords --",
				"admin' OR '1'='1",
			}

			for _, input := range maliciousInputs {
				threat, err := intrusionDetector.AnalyzeRequest(ctx, attackerClientID, input)
				require.NoError(t, err)
				assert.True(t, threat.IsMalicious, "SQL injection should be detected")
				assert.NotEmpty(t, threat.MatchedPatterns, "should have matched patterns")

				// Log security violation
				err = auditor.LogEvent(AuditEvent{
					UserID:    0, // Unknown user
					Action:    "sql_injection_attempt",
					Resource:  "security",
					IPAddress: attackerClientID,
					Success:   false,
					Error:     "malicious input detected",
					Metadata: map[string]interface{}{
						"input":           input,
						"matched_patterns": threat.MatchedPatterns,
						"risk_score":      threat.RiskScore,
					},
				})
				require.NoError(t, err)
			}

			// Check if attacker is blocked
			blocked, err := intrusionDetector.IsBlocked(ctx, attackerClientID)
			require.NoError(t, err)
			assert.True(t, blocked, "attacker should be blocked after multiple violations")
		})

		// Scenario 3: DDoS attack
		t.Run("ddos_attack", func(t *testing.T) {
			ddosClientID := "ddos-attacker"

			// Generate excessive requests
			allowedCount := 0
			for i := 0; i < 50; i++ {
				allowed, err := ddosProtector.CheckRequest(ctx, ddosClientID)
				require.NoError(t, err)
				if allowed {
					allowedCount++
				}
			}

			// Should have blocked some requests
			assert.Less(t, allowedCount, 50, "DDoS protection should have blocked some requests")

			// Check if client is flagged as suspicious
			info, err := ddosProtector.GetClientInfo(ctx, ddosClientID)
			require.NoError(t, err)
			assert.True(t, info.IsSuspicious, "client should be flagged as suspicious")

			// Log DDoS attempt
			err = auditor.LogEvent(AuditEvent{
				UserID:    0,
				Action:    "ddos_attempt",
				Resource:  "security",
				IPAddress: ddosClientID,
				Success:   false,
				Error:     "excessive requests detected",
				Metadata: map[string]interface{}{
					"request_count":  info.RequestCount,
					"is_suspicious":  info.IsSuspicious,
					"allowed_count":  allowedCount,
				},
			})
			require.NoError(t, err)
		})
	})

	t.Run("security_recovery_procedures", func(t *testing.T) {
		// Test MFA recovery with backup codes
		backupCodes, err := mfaService.repository.GetBackupCodes(userID)
		require.NoError(t, err)
		require.NotEmpty(t, backupCodes)

		// Use a backup code
		valid, err := mfaService.ValidateBackupCode(userID, backupCodes[0])
		require.NoError(t, err)
		assert.True(t, valid, "backup code should be valid")

		// Same backup code should not work again
		valid, err = mfaService.ValidateBackupCode(userID, backupCodes[0])
		require.NoError(t, err)
		assert.False(t, valid, "used backup code should not work again")

		// Log recovery event
		err = auditor.LogEvent(AuditEvent{
			UserID:    userID,
			Action:    "mfa_recovery",
			Resource:  "auth",
			IPAddress: "192.168.1.1",
			Success:   true,
			Metadata: map[string]interface{}{
				"recovery_method": "backup_code",
			},
		})
		require.NoError(t, err)
	})

	t.Run("audit_log_verification", func(t *testing.T) {
		// Retrieve audit logs for the user
		logs, err := auditor.GetUserAuditLogs(userID, 20)
		require.NoError(t, err)
		assert.NotEmpty(t, logs, "should have audit logs")

		// Verify key events are logged
		eventTypes := make(map[string]bool)
		for _, log := range logs {
			eventTypes[log.Action] = true
		}

		expectedEvents := []string{
			"mfa_enabled",
			"wallet_created",
			"kyc_submitted",
			"aml_screening",
			"api_request",
			"mfa_recovery",
		}

		for _, expectedEvent := range expectedEvents {
			assert.True(t, eventTypes[expectedEvent], "should have logged %s event", expectedEvent)
		}
	})

	t.Run("progressive_rate_limiting_verification", func(t *testing.T) {
		// Check that the attacker has increased penalties
		info, err := rateLimiter.GetClientInfo(ctx, "attacker-ip")
		require.NoError(t, err)
		assert.Greater(t, info.PenaltyMultiplier, 1.0, "attacker should have penalty multiplier")
		assert.True(t, info.UnderPenalty, "attacker should be under penalty")
		assert.Greater(t, info.ViolationCount, 0, "attacker should have violations")

		// Normal client should have no penalties
		normalInfo, err := rateLimiter.GetClientInfo(ctx, clientID)
		require.NoError(t, err)
		assert.Equal(t, 1.0, normalInfo.PenaltyMultiplier, "normal client should have no penalty")
		assert.False(t, normalInfo.UnderPenalty, "normal client should not be under penalty")
	})
}

// TestSecurityFrameworkPerformance tests the performance of security components
func TestSecurityFrameworkPerformance(t *testing.T) {
	// Initialize components
	rateLimiter := NewRateLimiter(RateLimiterConfig{
		RequestsPerMinute: 1000,
		BurstSize:         100,
		CleanupInterval:   time.Minute,
	})
	intrusionDetector := NewIntrusionDetector(IntrusionDetectionConfig{
		SuspiciousPatterns: []string{
			`(?i)(union|select)`,
			`(?i)(<script)`,
		},
		MaxViolationsPerHour: 100,
		BlockDuration:        time.Hour,
		CleanupInterval:      time.Minute,
	})

	ctx := context.Background()

	t.Run("rate_limiter_performance", func(t *testing.T) {
		start := time.Now()
		
		// Test 1000 requests
		for i := 0; i < 1000; i++ {
			clientID := fmt.Sprintf("client-%d", i%100) // 100 different clients
			_, err := rateLimiter.Allow(ctx, clientID)
			require.NoError(t, err)
		}

		duration := time.Since(start)
		t.Logf("Rate limiter processed 1000 requests in %v", duration)
		
		// Should process requests quickly
		assert.Less(t, duration, time.Second, "rate limiter should be fast")
	})

	t.Run("intrusion_detection_performance", func(t *testing.T) {
		inputs := []string{
			"normal user input",
			"SELECT * FROM users",
			"<script>alert('xss')</script>",
			"regular text without issues",
		}

		start := time.Now()

		// Test 1000 requests
		for i := 0; i < 1000; i++ {
			clientID := fmt.Sprintf("client-%d", i%50)
			input := inputs[i%len(inputs)]
			_, err := intrusionDetector.AnalyzeRequest(ctx, clientID, input)
			require.NoError(t, err)
		}

		duration := time.Since(start)
		t.Logf("Intrusion detector processed 1000 requests in %v", duration)
		
		// Should process requests quickly
		assert.Less(t, duration, time.Second, "intrusion detector should be fast")
	})
}

// TestSecurityFrameworkConcurrency tests concurrent access to security components
func TestSecurityFrameworkConcurrency(t *testing.T) {
	rateLimiter := NewRateLimiter(RateLimiterConfig{
		RequestsPerMinute: 100,
		BurstSize:         10,
		CleanupInterval:   time.Minute,
	})

	ctx := context.Background()
	numGoroutines := 10
	requestsPerGoroutine := 100

	// Test concurrent access
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer func() { done <- true }()

			clientID := fmt.Sprintf("client-%d", goroutineID)
			
			for j := 0; j < requestsPerGoroutine; j++ {
				_, err := rateLimiter.Allow(ctx, clientID)
				assert.NoError(t, err)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		select {
		case <-done:
			// Goroutine completed
		case <-time.After(10 * time.Second):
			t.Fatal("Test timed out - possible deadlock")
		}
	}
}

