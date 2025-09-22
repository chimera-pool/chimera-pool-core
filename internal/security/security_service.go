package security

import (
	"context"
	"fmt"
	"time"
)

// SecurityService provides a unified interface for all security components
type SecurityService struct {
	mfa               *MFAService
	rateLimiter       *ProgressiveRateLimiter
	bruteForceProtector *BruteForceProtector
	ddosProtector     *DDoSProtector
	intrusionDetector *IntrusionDetector
	secureWallet      *SecureWallet
	compliance        *ComplianceManager
	auditor           *AuditLogger
	encryptor         *DataEncryptor
}

// SecurityConfig holds configuration for all security components
type SecurityConfig struct {
	RateLimiting      ProgressiveRateLimiterConfig
	BruteForce        BruteForceConfig
	DDoS              DDoSConfig
	IntrusionDetection IntrusionDetectionConfig
}

// DefaultSecurityConfig returns a default security configuration
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		RateLimiting: ProgressiveRateLimiterConfig{
			BaseRequestsPerMinute: 60,
			BaseBurstSize:         10,
			MaxPenaltyMultiplier:  10,
			PenaltyDuration:       time.Hour,
			CleanupInterval:       time.Minute,
		},
		BruteForce: BruteForceConfig{
			MaxAttempts:     5,
			WindowDuration:  time.Hour,
			LockoutDuration: time.Hour,
			CleanupInterval: time.Minute,
		},
		DDoS: DDoSConfig{
			RequestsPerSecond:   20,
			BurstSize:          50,
			SuspiciousThreshold: 200,
			BlockDuration:      time.Hour,
			CleanupInterval:    time.Minute,
		},
		IntrusionDetection: IntrusionDetectionConfig{
			SuspiciousPatterns: []string{
				// SQL Injection patterns
				`(?i)(union\s+select|select\s+.*\s+from|insert\s+into|delete\s+from|drop\s+table|create\s+table|alter\s+table)`,
				`(?i)(\'\s*or\s+\'\d+\'\s*=\s*\'\d+|\'\s*or\s+\d+\s*=\s*\d+|admin\'\s*--|\'\s*;\s*drop)`,
				
				// XSS patterns
				`(?i)(<script[^>]*>|</script>|javascript:|vbscript:|onload\s*=|onerror\s*=|onclick\s*=)`,
				`(?i)(<iframe|<object|<embed|<applet|<meta\s+http-equiv)`,
				
				// Command injection patterns
				`(?i)(;\s*(rm|del|format|shutdown|reboot)|&&\s*(rm|del)|`+"`"+`.*`+"`"+`)`,
				`(?i)(\|\s*(nc|netcat|wget|curl|chmod|chown))`,
				
				// Path traversal patterns
				`(?i)(\.\.\/|\.\.\\|%2e%2e%2f|%2e%2e%5c)`,
				
				// LDAP injection patterns
				`(?i)(\*\)\(|\)\(.*\*|\(\||\)\(&)`,
			},
			MaxViolationsPerHour: 10,
			BlockDuration:        24 * time.Hour,
			CleanupInterval:      time.Hour,
		},
	}
}

// NewSecurityService creates a new security service with all components
func NewSecurityService(config *SecurityConfig) *SecurityService {
	if config == nil {
		config = DefaultSecurityConfig()
	}

	return &SecurityService{
		mfa:               NewMFAService(),
		rateLimiter:       NewProgressiveRateLimiter(config.RateLimiting),
		bruteForceProtector: NewBruteForceProtector(config.BruteForce),
		ddosProtector:     NewDDoSProtector(config.DDoS),
		intrusionDetector: NewIntrusionDetector(config.IntrusionDetection),
		secureWallet:      NewSecureWallet(),
		compliance:        NewComplianceManager(),
		auditor:           NewAuditLogger(),
		encryptor:         NewDataEncryptor(),
	}
}

// SecurityCheckRequest represents a request for security validation
type SecurityCheckRequest struct {
	UserID    int64
	ClientID  string
	IPAddress string
	UserAgent string
	Action    string
	Resource  string
	Input     string
}

// SecurityCheckResult represents the result of security validation
type SecurityCheckResult struct {
	Allowed           bool
	Reason            string
	RateLimited       bool
	BruteForceBlocked bool
	DDoSBlocked       bool
	IntrusionDetected bool
	ThreatInfo        *ThreatInfo
	Violations        []string
}

// ValidateRequest performs comprehensive security validation on a request
func (s *SecurityService) ValidateRequest(ctx context.Context, req SecurityCheckRequest) (*SecurityCheckResult, error) {
	result := &SecurityCheckResult{
		Allowed:    true,
		Violations: make([]string, 0),
	}

	// 1. Check rate limiting
	allowed, err := s.rateLimiter.Allow(ctx, req.ClientID)
	if err != nil {
		return nil, fmt.Errorf("rate limiting check failed: %w", err)
	}
	if !allowed {
		result.Allowed = false
		result.RateLimited = true
		result.Reason = "rate limit exceeded"
		result.Violations = append(result.Violations, "rate_limit_exceeded")
		
		// Record violation
		s.rateLimiter.RecordViolation(ctx, req.ClientID, ViolationTypeRateLimit)
	}

	// 2. Check brute force protection (for authentication actions)
	if req.Action == "login" || req.Action == "auth" {
		allowed, err := s.bruteForceProtector.CheckAttempt(ctx, req.ClientID)
		if err != nil {
			return nil, fmt.Errorf("brute force check failed: %w", err)
		}
		if !allowed {
			result.Allowed = false
			result.BruteForceBlocked = true
			result.Reason = "brute force protection triggered"
			result.Violations = append(result.Violations, "brute_force_blocked")
		}
	}

	// 3. Check DDoS protection
	allowed, err = s.ddosProtector.CheckRequest(ctx, req.ClientID)
	if err != nil {
		return nil, fmt.Errorf("DDoS check failed: %w", err)
	}
	if !allowed {
		result.Allowed = false
		result.DDoSBlocked = true
		result.Reason = "DDoS protection triggered"
		result.Violations = append(result.Violations, "ddos_blocked")
	}

	// 4. Check for intrusion attempts
	if req.Input != "" {
		threat, err := s.intrusionDetector.AnalyzeRequest(ctx, req.ClientID, req.Input)
		if err != nil {
			return nil, fmt.Errorf("intrusion detection failed: %w", err)
		}
		
		result.ThreatInfo = threat
		
		if threat.IsMalicious {
			result.Allowed = false
			result.IntrusionDetected = true
			result.Reason = "malicious input detected"
			result.Violations = append(result.Violations, "malicious_input")
		}
		
		if threat.IsBlocked {
			result.Allowed = false
			result.IntrusionDetected = true
			result.Reason = "client blocked due to previous violations"
			result.Violations = append(result.Violations, "client_blocked")
		}
	}

	// 5. Log the security check
	s.auditor.LogEvent(AuditEvent{
		UserID:    req.UserID,
		Action:    fmt.Sprintf("security_check_%s", req.Action),
		Resource:  "security",
		IPAddress: req.IPAddress,
		UserAgent: req.UserAgent,
		Success:   result.Allowed,
		Error:     result.Reason,
		Metadata: map[string]interface{}{
			"violations":         result.Violations,
			"rate_limited":       result.RateLimited,
			"brute_force_blocked": result.BruteForceBlocked,
			"ddos_blocked":       result.DDoSBlocked,
			"intrusion_detected": result.IntrusionDetected,
		},
	})

	return result, nil
}

// RecordAuthenticationAttempt records an authentication attempt result
func (s *SecurityService) RecordAuthenticationAttempt(ctx context.Context, clientID string, success bool) error {
	if success {
		return s.bruteForceProtector.RecordSuccessfulAttempt(ctx, clientID)
	} else {
		return s.bruteForceProtector.RecordFailedAttempt(ctx, clientID)
	}
}

// SetupMFA sets up multi-factor authentication for a user
func (s *SecurityService) SetupMFA(userID int64, issuer, accountName string) (*MFASetupInfo, error) {
	secret, qrCode, err := s.mfa.GenerateTOTPSecret(userID, issuer, accountName)
	if err != nil {
		return nil, fmt.Errorf("failed to generate TOTP secret: %w", err)
	}

	backupCodes, err := s.mfa.GenerateBackupCodes(userID, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to generate backup codes: %w", err)
	}

	return &MFASetupInfo{
		Secret:      secret,
		QRCode:      qrCode,
		BackupCodes: backupCodes,
	}, nil
}

// MFASetupInfo contains MFA setup information
type MFASetupInfo struct {
	Secret      string
	QRCode      string
	BackupCodes []string
}

// VerifyMFA verifies a TOTP code or backup code
func (s *SecurityService) VerifyMFA(userID int64, code string) (bool, error) {
	// First try TOTP
	secret, err := s.mfa.repository.GetTOTPSecret(userID)
	if err == nil && s.mfa.ValidateTOTP(secret, code) {
		return true, nil
	}

	// Try backup code
	valid, err := s.mfa.ValidateBackupCode(userID, code)
	if err != nil {
		return false, fmt.Errorf("failed to validate backup code: %w", err)
	}

	return valid, nil
}

// CreateSecureWallet creates a new secure wallet
func (s *SecurityService) CreateSecureWallet(address, privateKey string) (string, error) {
	return s.secureWallet.CreateWallet(address, privateKey)
}

// SignTransaction signs a transaction with a secure wallet
func (s *SecurityService) SignTransaction(walletID, privateKey string, transactionData []byte) (string, error) {
	return s.secureWallet.SignTransaction(walletID, privateKey, transactionData)
}

// CheckCompliance checks compliance requirements for a user
func (s *SecurityService) CheckCompliance(userID int64, country string) (*ComplianceRequirements, error) {
	return s.compliance.GetComplianceRequirements(userID, country)
}

// SubmitKYC submits KYC data for verification
func (s *SecurityService) SubmitKYC(data KYCData) error {
	return s.compliance.SubmitKYC(data)
}

// PerformAMLScreening performs AML screening
func (s *SecurityService) PerformAMLScreening(userID int64, fullName string) (*AMLResult, error) {
	return s.compliance.PerformAMLScreening(userID, fullName)
}

// EncryptSensitiveData encrypts sensitive data for storage
func (s *SecurityService) EncryptSensitiveData(data string) (string, error) {
	return s.encryptor.EncryptSensitiveData(data)
}

// DecryptSensitiveData decrypts sensitive data from storage
func (s *SecurityService) DecryptSensitiveData(encryptedData string) (string, error) {
	return s.encryptor.DecryptSensitiveData(encryptedData)
}

// GetSecurityMetrics returns security metrics and statistics
func (s *SecurityService) GetSecurityMetrics(ctx context.Context) (*SecurityMetrics, error) {
	// This would typically aggregate metrics from all components
	return &SecurityMetrics{
		TotalRequests:        1000, // Example values
		BlockedRequests:      50,
		MaliciousRequests:    10,
		ActiveMFAUsers:       100,
		ComplianceChecks:     200,
		AuditLogEntries:      500,
	}, nil
}

// SecurityMetrics contains security statistics
type SecurityMetrics struct {
	TotalRequests        int64
	BlockedRequests      int64
	MaliciousRequests    int64
	ActiveMFAUsers       int64
	ComplianceChecks     int64
	AuditLogEntries      int64
}

// GetUserAuditLogs retrieves audit logs for a user
func (s *SecurityService) GetUserAuditLogs(userID int64, limit int) ([]AuditEvent, error) {
	return s.auditor.GetUserAuditLogs(userID, limit)
}

// LogSecurityEvent logs a security-related event
func (s *SecurityService) LogSecurityEvent(event AuditEvent) error {
	return s.auditor.LogEvent(event)
}

// IsClientBlocked checks if a client is currently blocked by any security component
func (s *SecurityService) IsClientBlocked(ctx context.Context, clientID string) (*ClientBlockStatus, error) {
	status := &ClientBlockStatus{
		ClientID: clientID,
	}

	// Check intrusion detection
	blocked, err := s.intrusionDetector.IsBlocked(ctx, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to check intrusion detection: %w", err)
	}
	status.IntrusionBlocked = blocked

	// Check DDoS protection
	ddosInfo, err := s.ddosProtector.GetClientInfo(ctx, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to check DDoS protection: %w", err)
	}
	status.DDoSBlocked = ddosInfo.IsBlocked
	status.Suspicious = ddosInfo.IsSuspicious

	// Check rate limiting penalties
	rateLimitInfo, err := s.rateLimiter.GetClientInfo(ctx, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to check rate limiting: %w", err)
	}
	status.UnderPenalty = rateLimitInfo.UnderPenalty
	status.PenaltyMultiplier = rateLimitInfo.PenaltyMultiplier

	status.IsBlocked = status.IntrusionBlocked || status.DDoSBlocked

	return status, nil
}

// ClientBlockStatus contains information about a client's block status
type ClientBlockStatus struct {
	ClientID          string
	IsBlocked         bool
	IntrusionBlocked  bool
	DDoSBlocked       bool
	Suspicious        bool
	UnderPenalty      bool
	PenaltyMultiplier float64
}

// UnblockClient removes blocks for a client (admin function)
func (s *SecurityService) UnblockClient(ctx context.Context, clientID string) error {
	// This would typically require admin privileges
	// For now, we'll just log the action
	return s.auditor.LogEvent(AuditEvent{
		UserID:    0, // System action
		Action:    "client_unblocked",
		Resource:  "security",
		Success:   true,
		Metadata: map[string]interface{}{
			"client_id": clientID,
		},
	})
}