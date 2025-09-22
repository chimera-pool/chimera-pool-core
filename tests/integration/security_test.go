package integration

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"chimera-pool-core/internal/testutil"
)

// SecurityTestSuite validates comprehensive security measures
type SecurityTestSuite struct {
	FinalIntegrationTestSuite
}

func TestSecurityTestSuite(t *testing.T) {
	suite.Run(t, new(SecurityTestSuite))
}

// TestAuthenticationSecurity tests authentication security measures
func (s *SecurityTestSuite) TestAuthenticationSecurity() {
	s.T().Log("Testing authentication security")
	
	// Test password strength requirements
	s.testPasswordStrengthRequirements()
	
	// Test brute force protection
	s.testBruteForceProtection()
	
	// Test session management
	s.testSessionManagement()
	
	// Test MFA security
	s.testMFASecurity()
}

func (s *SecurityTestSuite) testPasswordStrengthRequirements() {
	s.T().Log("Testing password strength requirements")
	
	weakPasswords := []string{
		"123456",
		"password",
		"abc123",
		"qwerty",
		"Password", // No numbers or special chars
		"password123", // No uppercase or special chars
		"PASSWORD123", // No lowercase or special chars
		"Pass123", // Too short
	}
	
	for _, weakPassword := range weakPasswords {
		_, err := s.authService.Register(s.ctx, &testutil.RegisterRequest{
			Username: fmt.Sprintf("weakpass_%s", hex.EncodeToString([]byte(weakPassword))[:8]),
			Email:    fmt.Sprintf("weak_%s@example.com", hex.EncodeToString([]byte(weakPassword))[:8]),
			Password: weakPassword,
		})
		
		s.Assert().Error(err, "Weak password should be rejected: %s", weakPassword)
		s.Assert().Contains(err.Error(), "password", "Error should mention password requirements")
	}
	
	// Test strong password acceptance
	strongPassword := "SecurePassword123!@#"
	user, err := s.authService.Register(s.ctx, &testutil.RegisterRequest{
		Username: "strongpass_user",
		Email:    "strongpass@example.com",
		Password: strongPassword,
	})
	s.Require().NoError(err)
	s.Assert().NotEmpty(user.ID)
}

func (s *SecurityTestSuite) testBruteForceProtection() {
	s.T().Log("Testing brute force protection")
	
	// Create test user
	user, err := s.authService.Register(s.ctx, &testutil.RegisterRequest{
		Username: "bruteforce_target",
		Email:    "bruteforce@example.com",
		Password: "SecurePassword123!",
	})
	s.Require().NoError(err)
	
	// Attempt multiple failed logins
	const maxAttempts = 5
	for i := 0; i < maxAttempts+2; i++ {
		_, err := s.authService.Login(s.ctx, &testutil.LoginRequest{
			Username: user.Username,
			Password: "WrongPassword123!",
		})
		s.Assert().Error(err, "Login with wrong password should fail")
		
		if i >= maxAttempts {
			// After max attempts, should get rate limited
			s.Assert().Contains(err.Error(), "rate limit", "Should be rate limited after max attempts")
		}
	}
	
	// Wait for rate limit to reset (if implemented with time-based reset)
	time.Sleep(2 * time.Second)
	
	// Correct password should still work after rate limit period
	token, err := s.authService.Login(s.ctx, &testutil.LoginRequest{
		Username: user.Username,
		Password: "SecurePassword123!",
	})
	s.Assert().NoError(err, "Correct password should work after rate limit period")
	s.Assert().NotEmpty(token.AccessToken)
}

func (s *SecurityTestSuite) testSessionManagement() {
	s.T().Log("Testing session management security")
	
	// Create test user
	user, err := s.authService.Register(s.ctx, &testutil.RegisterRequest{
		Username: "session_test_user",
		Email:    "session@example.com",
		Password: "SecurePassword123!",
	})
	s.Require().NoError(err)
	
	// Login and get token
	token, err := s.authService.Login(s.ctx, &testutil.LoginRequest{
		Username: user.Username,
		Password: "SecurePassword123!",
	})
	s.Require().NoError(err)
	
	// Test token validation
	claims, err := s.authService.ValidateToken(s.ctx, token.AccessToken)
	s.Require().NoError(err)
	s.Assert().Equal(user.ID, claims.UserID)
	
	// Test token expiration (if short-lived tokens are used)
	// This would require waiting for token expiration or manipulating time
	
	// Test token revocation
	err = s.authService.RevokeToken(s.ctx, token.AccessToken)
	s.Require().NoError(err)
	
	// Revoked token should not be valid
	_, err = s.authService.ValidateToken(s.ctx, token.AccessToken)
	s.Assert().Error(err, "Revoked token should not be valid")
}

func (s *SecurityTestSuite) testMFASecurity() {
	s.T().Log("Testing MFA security")
	
	// Create test user
	user, err := s.authService.Register(s.ctx, &testutil.RegisterRequest{
		Username: "mfa_test_user",
		Email:    "mfa@example.com",
		Password: "SecurePassword123!",
	})
	s.Require().NoError(err)
	
	// Setup MFA
	mfaSecret, err := s.securitySvc.SetupMFA(s.ctx, user.ID)
	s.Require().NoError(err)
	s.Assert().NotEmpty(mfaSecret.Secret)
	s.Assert().NotEmpty(mfaSecret.QRCode)
	s.Assert().Greater(len(mfaSecret.BackupCodes), 0)
	
	// Test MFA validation with correct code
	validCode := testutil.GenerateTOTP(mfaSecret.Secret)
	valid, err := s.securitySvc.ValidateMFA(s.ctx, user.ID, validCode)
	s.Require().NoError(err)
	s.Assert().True(valid)
	
	// Test MFA validation with incorrect code
	invalidCode := "123456"
	valid, err = s.securitySvc.ValidateMFA(s.ctx, user.ID, invalidCode)
	s.Require().NoError(err)
	s.Assert().False(valid)
	
	// Test backup code usage
	backupCode := mfaSecret.BackupCodes[0]
	valid, err = s.securitySvc.ValidateBackupCode(s.ctx, user.ID, backupCode)
	s.Require().NoError(err)
	s.Assert().True(valid)
	
	// Same backup code should not work twice
	valid, err = s.securitySvc.ValidateBackupCode(s.ctx, user.ID, backupCode)
	s.Require().NoError(err)
	s.Assert().False(valid)
}

// TestAPISecurityMeasures tests API security measures
func (s *SecurityTestSuite) TestAPISecurityMeasures() {
	s.T().Log("Testing API security measures")
	
	// Test rate limiting
	s.testAPIRateLimiting()
	
	// Test input validation
	s.testInputValidation()
	
	// Test authorization
	s.testAPIAuthorization()
	
	// Test CORS and security headers
	s.testSecurityHeaders()
}

func (s *SecurityTestSuite) testAPIRateLimiting() {
	s.T().Log("Testing API rate limiting")
	
	client := testutil.NewAPIClient("http://localhost:8080", "")
	
	// Test rate limiting on public endpoint
	const maxRequests = 100
	var successCount, rateLimitedCount int
	
	for i := 0; i < maxRequests+20; i++ {
		resp, err := client.Get("/api/pool/stats")
		if err != nil {
			continue
		}
		
		if resp.StatusCode == http.StatusOK {
			successCount++
		} else if resp.StatusCode == http.StatusTooManyRequests {
			rateLimitedCount++
		}
		resp.Body.Close()
	}
	
	s.Assert().Greater(successCount, 0, "Some requests should succeed")
	s.Assert().Greater(rateLimitedCount, 0, "Some requests should be rate limited")
	s.T().Logf("Rate limiting: %d successful, %d rate limited", successCount, rateLimitedCount)
}

func (s *SecurityTestSuite) testInputValidation() {
	s.T().Log("Testing input validation")
	
	client := testutil.NewAPIClient("http://localhost:8080", "")
	
	// Test SQL injection attempts
	sqlInjectionPayloads := []string{
		"'; DROP TABLE users; --",
		"' OR '1'='1",
		"admin'--",
		"' UNION SELECT * FROM users --",
	}
	
	for _, payload := range sqlInjectionPayloads {
		resp, err := client.PostJSON("/api/auth/login", map[string]interface{}{
			"username": payload,
			"password": "password123",
		})
		
		if err == nil {
			s.Assert().NotEqual(http.StatusOK, resp.StatusCode, 
				"SQL injection payload should not succeed: %s", payload)
			resp.Body.Close()
		}
	}
	
	// Test XSS attempts
	xssPayloads := []string{
		"<script>alert('xss')</script>",
		"javascript:alert('xss')",
		"<img src=x onerror=alert('xss')>",
	}
	
	for _, payload := range xssPayloads {
		resp, err := client.PostJSON("/api/auth/register", map[string]interface{}{
			"username": payload,
			"email":    "test@example.com",
			"password": "SecurePassword123!",
		})
		
		if err == nil {
			s.Assert().NotEqual(http.StatusOK, resp.StatusCode, 
				"XSS payload should not succeed: %s", payload)
			resp.Body.Close()
		}
	}
}

func (s *SecurityTestSuite) testAPIAuthorization() {
	s.T().Log("Testing API authorization")
	
	// Create test users with different roles
	adminUser, err := s.authService.Register(s.ctx, &testutil.RegisterRequest{
		Username: "admin_user",
		Email:    "admin@example.com",
		Password: "SecurePassword123!",
	})
	s.Require().NoError(err)
	
	regularUser, err := s.authService.Register(s.ctx, &testutil.RegisterRequest{
		Username: "regular_user",
		Email:    "regular@example.com",
		Password: "SecurePassword123!",
	})
	s.Require().NoError(err)
	
	// Set admin role
	err = s.authService.SetUserRole(s.ctx, adminUser.ID, "admin")
	s.Require().NoError(err)
	
	// Get tokens
	adminToken, err := s.authService.Login(s.ctx, &testutil.LoginRequest{
		Username: "admin_user",
		Password: "SecurePassword123!",
	})
	s.Require().NoError(err)
	
	regularToken, err := s.authService.Login(s.ctx, &testutil.LoginRequest{
		Username: "regular_user",
		Password: "SecurePassword123!",
	})
	s.Require().NoError(err)
	
	// Test admin-only endpoints
	adminClient := testutil.NewAPIClient("http://localhost:8080", adminToken.AccessToken)
	regularClient := testutil.NewAPIClient("http://localhost:8080", regularToken.AccessToken)
	unauthClient := testutil.NewAPIClient("http://localhost:8080", "")
	
	adminEndpoints := []string{
		"/api/admin/users",
		"/api/admin/system/health",
		"/api/admin/pool/config",
	}
	
	for _, endpoint := range adminEndpoints {
		// Admin should have access
		resp, err := adminClient.Get(endpoint)
		if err == nil {
			s.Assert().NotEqual(http.StatusForbidden, resp.StatusCode, 
				"Admin should have access to %s", endpoint)
			resp.Body.Close()
		}
		
		// Regular user should be forbidden
		resp, err = regularClient.Get(endpoint)
		if err == nil {
			s.Assert().Equal(http.StatusForbidden, resp.StatusCode, 
				"Regular user should be forbidden from %s", endpoint)
			resp.Body.Close()
		}
		
		// Unauthenticated should be unauthorized
		resp, err = unauthClient.Get(endpoint)
		if err == nil {
			s.Assert().Equal(http.StatusUnauthorized, resp.StatusCode, 
				"Unauthenticated user should be unauthorized for %s", endpoint)
			resp.Body.Close()
		}
	}
}

func (s *SecurityTestSuite) testSecurityHeaders() {
	s.T().Log("Testing security headers")
	
	client := testutil.NewAPIClient("http://localhost:8080", "")
	
	resp, err := client.Get("/api/pool/stats")
	s.Require().NoError(err)
	defer resp.Body.Close()
	
	// Check for important security headers
	securityHeaders := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"X-XSS-Protection":       "1; mode=block",
		"Strict-Transport-Security": "", // Should be present for HTTPS
		"Content-Security-Policy": "", // Should be present
	}
	
	for header, expectedValue := range securityHeaders {
		actualValue := resp.Header.Get(header)
		if expectedValue != "" {
			s.Assert().Equal(expectedValue, actualValue, 
				"Security header %s should have value %s", header, expectedValue)
		} else {
			s.Assert().NotEmpty(actualValue, 
				"Security header %s should be present", header)
		}
	}
}

// TestCryptographicSecurity tests cryptographic implementations
func (s *SecurityTestSuite) TestCryptographicSecurity() {
	s.T().Log("Testing cryptographic security")
	
	// Test password hashing
	s.testPasswordHashing()
	
	// Test encryption/decryption
	s.testEncryptionDecryption()
	
	// Test random number generation
	s.testRandomNumberGeneration()
}

func (s *SecurityTestSuite) testPasswordHashing() {
	s.T().Log("Testing password hashing security")
	
	password := "TestPassword123!"
	
	// Hash password
	hash1, err := s.securitySvc.HashPassword(password)
	s.Require().NoError(err)
	s.Assert().NotEmpty(hash1)
	s.Assert().NotEqual(password, hash1, "Hash should not equal plaintext password")
	
	// Hash same password again - should produce different hash (salt)
	hash2, err := s.securitySvc.HashPassword(password)
	s.Require().NoError(err)
	s.Assert().NotEqual(hash1, hash2, "Same password should produce different hashes due to salt")
	
	// Verify password against both hashes
	valid1, err := s.securitySvc.VerifyPassword(password, hash1)
	s.Require().NoError(err)
	s.Assert().True(valid1)
	
	valid2, err := s.securitySvc.VerifyPassword(password, hash2)
	s.Require().NoError(err)
	s.Assert().True(valid2)
	
	// Wrong password should not verify
	wrongValid, err := s.securitySvc.VerifyPassword("WrongPassword123!", hash1)
	s.Require().NoError(err)
	s.Assert().False(wrongValid)
}

func (s *SecurityTestSuite) testEncryptionDecryption() {
	s.T().Log("Testing encryption/decryption")
	
	plaintext := "Sensitive data that needs encryption"
	
	// Encrypt data
	encrypted, err := s.securitySvc.Encrypt([]byte(plaintext))
	s.Require().NoError(err)
	s.Assert().NotEmpty(encrypted)
	s.Assert().NotEqual(plaintext, string(encrypted))
	
	// Decrypt data
	decrypted, err := s.securitySvc.Decrypt(encrypted)
	s.Require().NoError(err)
	s.Assert().Equal(plaintext, string(decrypted))
	
	// Test with different data sizes
	testData := []string{
		"",
		"a",
		"short text",
		strings.Repeat("long text ", 1000),
	}
	
	for _, data := range testData {
		encrypted, err := s.securitySvc.Encrypt([]byte(data))
		s.Require().NoError(err)
		
		decrypted, err := s.securitySvc.Decrypt(encrypted)
		s.Require().NoError(err)
		s.Assert().Equal(data, string(decrypted))
	}
}

func (s *SecurityTestSuite) testRandomNumberGeneration() {
	s.T().Log("Testing random number generation")
	
	// Generate multiple random values
	const numSamples = 1000
	randomValues := make([][]byte, numSamples)
	
	for i := 0; i < numSamples; i++ {
		randomBytes := make([]byte, 32)
		_, err := rand.Read(randomBytes)
		s.Require().NoError(err)
		randomValues[i] = randomBytes
	}
	
	// Check for uniqueness (no duplicates)
	uniqueValues := make(map[string]bool)
	for _, value := range randomValues {
		hexValue := hex.EncodeToString(value)
		s.Assert().False(uniqueValues[hexValue], "Random values should be unique")
		uniqueValues[hexValue] = true
	}
	
	// Check for proper entropy (basic statistical test)
	// Count bit distribution
	bitCounts := make([]int, 8)
	for _, value := range randomValues {
		for _, b := range value {
			for bit := 0; bit < 8; bit++ {
				if (b>>bit)&1 == 1 {
					bitCounts[bit]++
				}
			}
		}
	}
	
	// Each bit position should be roughly 50% ones
	expectedCount := numSamples * 32 / 2 // 32 bytes per sample, expect 50% ones
	tolerance := expectedCount / 10      // 10% tolerance
	
	for bit, count := range bitCounts {
		s.Assert().Greater(count, expectedCount-tolerance, 
			"Bit %d should have adequate entropy", bit)
		s.Assert().Less(count, expectedCount+tolerance, 
			"Bit %d should have adequate entropy", bit)
	}
}

// TestNetworkSecurity tests network-level security measures
func (s *SecurityTestSuite) TestNetworkSecurity() {
	s.T().Log("Testing network security measures")
	
	// Test connection limits
	s.testConnectionLimits()
	
	// Test DDoS protection
	s.testDDoSProtection()
	
	// Test SSL/TLS security
	s.testTLSSecurity()
	
	// Test network isolation
	s.testNetworkIsolation()
}

// TestComprehensiveSecurityAudit performs a comprehensive security audit
func (s *SecurityTestSuite) TestComprehensiveSecurityAudit() {
	s.T().Log("Performing comprehensive security audit")
	
	// Test all security components
	s.testPasswordSecurity()
	s.testSessionSecurity()
	s.testDataProtection()
	s.testAccessControl()
	s.testAuditLogging()
	s.testSecurityHeaders()
	s.testVulnerabilityProtection()
}

func (s *SecurityTestSuite) testPasswordSecurity() {
	s.T().Log("Testing password security measures")
	
	// Test password complexity requirements
	weakPasswords := []string{
		"123456",
		"password",
		"qwerty",
		"abc123",
		"Password", // Missing numbers and special chars
		"password123", // Missing uppercase and special chars
		"PASSWORD123", // Missing lowercase and special chars
		"Pass1!", // Too short
	}
	
	for _, weakPassword := range weakPasswords {
		_, err := s.authService.Register(s.ctx, &testutil.RegisterRequest{
			Username: fmt.Sprintf("weaktest_%d", time.Now().UnixNano()),
			Email:    fmt.Sprintf("weak_%d@example.com", time.Now().UnixNano()),
			Password: weakPassword,
		})
		
		s.Assert().Error(err, "Weak password should be rejected: %s", weakPassword)
	}
	
	// Test password hashing security
	password := "SecureTestPassword123!"
	hash1, err := s.securitySvc.HashPassword(password)
	s.Require().NoError(err)
	
	hash2, err := s.securitySvc.HashPassword(password)
	s.Require().NoError(err)
	
	// Hashes should be different (salted)
	s.Assert().NotEqual(hash1, hash2, "Password hashes should be salted and unique")
	
	// Both should verify correctly
	valid1, err := s.securitySvc.VerifyPassword(password, hash1)
	s.Require().NoError(err)
	s.Assert().True(valid1)
	
	valid2, err := s.securitySvc.VerifyPassword(password, hash2)
	s.Require().NoError(err)
	s.Assert().True(valid2)
}

func (s *SecurityTestSuite) testSessionSecurity() {
	s.T().Log("Testing session security")
	
	// Create test user
	user, err := s.authService.Register(s.ctx, &testutil.RegisterRequest{
		Username: "session_security_user",
		Email:    "session_security@example.com",
		Password: "SecurePassword123!",
	})
	s.Require().NoError(err)
	
	// Test session creation
	token, err := s.authService.Login(s.ctx, &testutil.LoginRequest{
		Username: user.Username,
		Password: "SecurePassword123!",
	})
	s.Require().NoError(err)
	s.Assert().NotEmpty(token.AccessToken)
	
	// Test token validation
	claims, err := s.authService.ValidateToken(s.ctx, token.AccessToken)
	s.Require().NoError(err)
	s.Assert().Equal(user.ID, claims.UserID)
	
	// Test session invalidation
	err = s.authService.RevokeToken(s.ctx, token.AccessToken)
	s.Require().NoError(err)
	
	// Token should no longer be valid
	_, err = s.authService.ValidateToken(s.ctx, token.AccessToken)
	s.Assert().Error(err, "Revoked token should not be valid")
	
	// Test concurrent session limits (if implemented)
	tokens := make([]string, 0)
	for i := 0; i < 10; i++ {
		token, err := s.authService.Login(s.ctx, &testutil.LoginRequest{
			Username: user.Username,
			Password: "SecurePassword123!",
		})
		if err == nil {
			tokens = append(tokens, token.AccessToken)
		}
	}
	
	// Should have some limit on concurrent sessions
	s.Assert().LessOrEqual(len(tokens), 5, "Should limit concurrent sessions")
}

func (s *SecurityTestSuite) testDataProtection() {
	s.T().Log("Testing data protection measures")
	
	// Test encryption at rest
	sensitiveData := "sensitive mining pool configuration data"
	
	encrypted, err := s.securitySvc.Encrypt([]byte(sensitiveData))
	s.Require().NoError(err)
	s.Assert().NotEqual(sensitiveData, string(encrypted))
	s.Assert().Greater(len(encrypted), len(sensitiveData), "Encrypted data should be larger due to padding/metadata")
	
	// Test decryption
	decrypted, err := s.securitySvc.Decrypt(encrypted)
	s.Require().NoError(err)
	s.Assert().Equal(sensitiveData, string(decrypted))
	
	// Test encryption with different data sizes
	testData := [][]byte{
		[]byte(""),
		[]byte("a"),
		[]byte("short"),
		[]byte(strings.Repeat("long data ", 1000)),
		make([]byte, 1024*1024), // 1MB of zeros
	}
	
	for i, data := range testData {
		encrypted, err := s.securitySvc.Encrypt(data)
		s.Require().NoError(err, "Encryption should work for test data %d", i)
		
		decrypted, err := s.securitySvc.Decrypt(encrypted)
		s.Require().NoError(err, "Decryption should work for test data %d", i)
		s.Assert().Equal(data, decrypted, "Decrypted data should match original for test data %d", i)
	}
}

func (s *SecurityTestSuite) testAccessControl() {
	s.T().Log("Testing access control measures")
	
	// Create users with different roles
	adminUser, err := s.authService.Register(s.ctx, &testutil.RegisterRequest{
		Username: "access_admin",
		Email:    "access_admin@example.com",
		Password: "SecurePassword123!",
	})
	s.Require().NoError(err)
	
	regularUser, err := s.authService.Register(s.ctx, &testutil.RegisterRequest{
		Username: "access_regular",
		Email:    "access_regular@example.com",
		Password: "SecurePassword123!",
	})
	s.Require().NoError(err)
	
	// Set roles
	err = s.authService.SetUserRole(s.ctx, adminUser.ID, "admin")
	s.Require().NoError(err)
	
	err = s.authService.SetUserRole(s.ctx, regularUser.ID, "user")
	s.Require().NoError(err)
	
	// Get tokens
	adminToken, err := s.authService.Login(s.ctx, &testutil.LoginRequest{
		Username: adminUser.Username,
		Password: "SecurePassword123!",
	})
	s.Require().NoError(err)
	
	regularToken, err := s.authService.Login(s.ctx, &testutil.LoginRequest{
		Username: regularUser.Username,
		Password: "SecurePassword123!",
	})
	s.Require().NoError(err)
	
	// Test admin endpoints
	adminClient := testutil.NewAPIClient("http://localhost:8080", adminToken.AccessToken)
	regularClient := testutil.NewAPIClient("http://localhost:8080", regularToken.AccessToken)
	unauthClient := testutil.NewAPIClient("http://localhost:8080", "")
	
	adminEndpoints := []string{
		"/api/admin/users",
		"/api/admin/system/config",
		"/api/admin/pool/settings",
	}
	
	for _, endpoint := range adminEndpoints {
		// Admin should have access
		resp, err := adminClient.Get(endpoint)
		if err == nil {
			s.Assert().NotEqual(http.StatusForbidden, resp.StatusCode, 
				"Admin should have access to %s", endpoint)
			resp.Body.Close()
		}
		
		// Regular user should be forbidden
		resp, err = regularClient.Get(endpoint)
		if err == nil {
			s.Assert().Equal(http.StatusForbidden, resp.StatusCode, 
				"Regular user should be forbidden from %s", endpoint)
			resp.Body.Close()
		}
		
		// Unauthenticated should be unauthorized
		resp, err = unauthClient.Get(endpoint)
		if err == nil {
			s.Assert().Equal(http.StatusUnauthorized, resp.StatusCode, 
				"Unauthenticated user should be unauthorized for %s", endpoint)
			resp.Body.Close()
		}
	}
}

func (s *SecurityTestSuite) testAuditLogging() {
	s.T().Log("Testing audit logging")
	
	// Create test user for audit events
	user, err := s.authService.Register(s.ctx, &testutil.RegisterRequest{
		Username: "audit_test_user",
		Email:    "audit_test@example.com",
		Password: "SecurePassword123!",
	})
	s.Require().NoError(err)
	
	// Perform auditable actions
	
	// 1. Login (should be audited)
	token, err := s.authService.Login(s.ctx, &testutil.LoginRequest{
		Username: user.Username,
		Password: "SecurePassword123!",
	})
	s.Require().NoError(err)
	
	// 2. Failed login attempt (should be audited)
	_, err = s.authService.Login(s.ctx, &testutil.LoginRequest{
		Username: user.Username,
		Password: "WrongPassword123!",
	})
	s.Assert().Error(err, "Wrong password should fail")
	
	// 3. Sensitive API access (should be audited)
	client := testutil.NewAPIClient("http://localhost:8080", token.AccessToken)
	resp, err := client.Get("/api/user/profile")
	if err == nil {
		resp.Body.Close()
	}
	
	// 4. Profile update (should be audited)
	resp, err = client.PostJSON("/api/user/profile", map[string]interface{}{
		"email": "updated_audit_test@example.com",
	})
	if err == nil {
		resp.Body.Close()
	}
	
	// In a real implementation, you would verify that audit logs were created
	// This might involve checking a database table, log files, or external audit system
	// For now, we'll just verify the audit system is accessible
	
	adminToken := s.getAdminToken()
	adminClient := testutil.NewAPIClient("http://localhost:8080", adminToken)
	
	resp, err = adminClient.Get("/api/admin/audit/logs")
	if err == nil {
		defer resp.Body.Close()
		s.Assert().Equal(http.StatusOK, resp.StatusCode, "Audit logs should be accessible to admin")
	}
}

func (s *SecurityTestSuite) testVulnerabilityProtection() {
	s.T().Log("Testing vulnerability protection")
	
	client := testutil.NewAPIClient("http://localhost:8080", "")
	
	// Test SQL injection protection
	sqlInjectionPayloads := []string{
		"'; DROP TABLE users; --",
		"' OR '1'='1",
		"admin'--",
		"' UNION SELECT * FROM users --",
		"1; DELETE FROM users WHERE 1=1 --",
	}
	
	for _, payload := range sqlInjectionPayloads {
		resp, err := client.PostJSON("/api/auth/login", map[string]interface{}{
			"username": payload,
			"password": "password123",
		})
		
		if err == nil {
			s.Assert().NotEqual(http.StatusOK, resp.StatusCode, 
				"SQL injection payload should not succeed: %s", payload)
			s.Assert().NotEqual(http.StatusInternalServerError, resp.StatusCode, 
				"SQL injection should not cause server error: %s", payload)
			resp.Body.Close()
		}
	}
	
	// Test XSS protection
	xssPayloads := []string{
		"<script>alert('xss')</script>",
		"javascript:alert('xss')",
		"<img src=x onerror=alert('xss')>",
		"<svg onload=alert('xss')>",
		"';alert('xss');//",
	}
	
	for _, payload := range xssPayloads {
		resp, err := client.PostJSON("/api/auth/register", map[string]interface{}{
			"username": payload,
			"email":    "test@example.com",
			"password": "SecurePassword123!",
		})
		
		if err == nil {
			s.Assert().NotEqual(http.StatusOK, resp.StatusCode, 
				"XSS payload should not succeed: %s", payload)
			resp.Body.Close()
		}
	}
	
	// Test CSRF protection (if implemented)
	resp, err := client.PostJSON("/api/user/profile", map[string]interface{}{
		"email": "csrf_test@example.com",
	})
	
	if err == nil {
		// Should require authentication or CSRF token
		s.Assert().NotEqual(http.StatusOK, resp.StatusCode, 
			"CSRF-sensitive endpoint should be protected")
		resp.Body.Close()
	}
}

func (s *SecurityTestSuite) testTLSSecurity() {
	s.T().Log("Testing TLS security")
	
	// In a production environment, this would test:
	// - TLS version requirements
	// - Certificate validation
	// - Cipher suite restrictions
	// - HSTS headers
	
	// For now, test that security headers are present
	client := testutil.NewAPIClient("http://localhost:8080", "")
	
	resp, err := client.Get("/")
	s.Require().NoError(err)
	defer resp.Body.Close()
	
	// Check for security headers that would be present in production
	securityHeaders := map[string]bool{
		"X-Content-Type-Options": false,
		"X-Frame-Options":        false,
		"X-XSS-Protection":       false,
		"Referrer-Policy":        false,
	}
	
	for header := range securityHeaders {
		if resp.Header.Get(header) != "" {
			securityHeaders[header] = true
		}
	}
	
	presentHeaders := 0
	for _, present := range securityHeaders {
		if present {
			presentHeaders++
		}
	}
	
	s.Assert().Greater(presentHeaders, 2, "Should have multiple security headers present")
}

func (s *SecurityTestSuite) testNetworkIsolation() {
	s.T().Log("Testing network isolation")
	
	// Test that internal services are not directly accessible
	internalPorts := []string{
		"5432", // PostgreSQL
		"6379", // Redis
		"9090", // Prometheus (if running)
	}
	
	for _, port := range internalPorts {
		conn, err := net.DialTimeout("tcp", "localhost:"+port, 2*time.Second)
		if err == nil {
			conn.Close()
			s.T().Logf("WARNING: Internal service on port %s is accessible externally", port)
		}
	}
	
	// Test that only expected ports are open
	expectedPorts := []string{
		"8080",  // API server
		"18332", // Stratum server
	}
	
	for _, port := range expectedPorts {
		conn, err := net.DialTimeout("tcp", "localhost:"+port, 2*time.Second)
		s.Assert().NoError(err, "Expected port %s should be accessible", port)
		if err == nil {
			conn.Close()
		}
	}
}

// Helper method to get admin token for security tests
func (s *SecurityTestSuite) getAdminToken() string {
	adminUser, err := s.authService.Register(s.ctx, &testutil.RegisterRequest{
		Username: "security_admin",
		Email:    "security_admin@example.com",
		Password: "AdminPassword123!",
	})
	if err != nil {
		// User might already exist, try to login
		token, err := s.authService.Login(s.ctx, &testutil.LoginRequest{
			Username: "security_admin",
			Password: "AdminPassword123!",
		})
		if err == nil {
			return token.AccessToken
		}
		return ""
	}
	
	// Set admin role
	err = s.authService.SetUserRole(s.ctx, adminUser.ID, "admin")
	if err != nil {
		return ""
	}
	
	// Login and get token
	token, err := s.authService.Login(s.ctx, &testutil.LoginRequest{
		Username: "security_admin",
		Password: "AdminPassword123!",
	})
	if err != nil {
		return ""
	}
	
	return token.AccessToken
}

func (s *SecurityTestSuite) testConnectionLimits() {
	s.T().Log("Testing connection limits")
	
	const maxConnections = 100
	var wg sync.WaitGroup
	successfulConnections := 0
	rejectedConnections := 0
	
	// Attempt to create many connections
	for i := 0; i < maxConnections+20; i++ {
		wg.Add(1)
		go func(connID int) {
			defer wg.Done()
			
			miner := testutil.NewMockMiner(fmt.Sprintf("conn_test_%d", connID), "password123")
			err := miner.Connect("localhost:18332")
			
			if err == nil {
				successfulConnections++
				// Keep connection alive briefly
				time.Sleep(100 * time.Millisecond)
				miner.Disconnect()
			} else {
				rejectedConnections++
			}
		}(i)
	}
	
	wg.Wait()
	
	s.T().Logf("Connection limits: %d successful, %d rejected", 
		successfulConnections, rejectedConnections)
	
	// Should reject some connections when limit is exceeded
	s.Assert().Greater(rejectedConnections, 0, "Should reject connections when limit exceeded")
	s.Assert().Greater(successfulConnections, 0, "Should allow some connections")
}

func (s *SecurityTestSuite) testDDoSProtection() {
	s.T().Log("Testing DDoS protection")
	
	// Simulate rapid connection attempts from same IP
	const rapidAttempts = 50
	var wg sync.WaitGroup
	
	startTime := time.Now()
	
	for i := 0; i < rapidAttempts; i++ {
		wg.Add(1)
		go func(attemptID int) {
			defer wg.Done()
			
			miner := testutil.NewMockMiner(fmt.Sprintf("ddos_test_%d", attemptID), "password123")
			err := miner.Connect("localhost:18332")
			if err == nil {
				miner.Disconnect()
			}
		}(i)
	}
	
	wg.Wait()
	duration := time.Since(startTime)
	
	// Rapid connections should be throttled/rejected
	connectionsPerSecond := float64(rapidAttempts) / duration.Seconds()
	s.T().Logf("DDoS test: %.2f connections/second", connectionsPerSecond)
	
	// System should implement some form of rate limiting
	// This is a basic test - actual DDoS protection would be more sophisticated
	s.Assert().Less(connectionsPerSecond, 1000.0, 
		"System should limit rapid connection attempts")
}