# Comprehensive Security Framework

This package provides a complete, enterprise-grade security framework for the Chimera Mining Pool. It implements multiple layers of security protection with easy user onboarding and compliance features.

## Features

### 🔐 Multi-Factor Authentication (MFA)
- **TOTP Support**: Google Authenticator, Microsoft Authenticator, Authy compatible
- **QR Code Generation**: Easy setup with visual QR codes
- **Backup Codes**: 10 single-use backup codes for account recovery
- **Easy Onboarding**: Step-by-step setup wizard with clear instructions

### 🛡️ Advanced Security Measures
- **Progressive Rate Limiting**: Adaptive rate limiting with penalty multipliers
- **Brute Force Protection**: Account lockouts with configurable thresholds
- **DDoS Protection**: Request throttling and suspicious activity detection
- **Intrusion Detection**: Pattern-based malicious input detection

### 🔒 Data Protection
- **End-to-End Encryption**: AES-256-GCM encryption for sensitive data
- **Secure Password Hashing**: bcrypt with configurable cost
- **Secure Wallet Integration**: Encrypted private key storage
- **Data Encryption at Rest**: Automatic encryption of sensitive fields

### 📋 Regulatory Compliance
- **KYC (Know Your Customer)**: Identity verification workflows
- **AML (Anti-Money Laundering)**: Risk scoring and screening
- **Audit Logging**: Comprehensive security event logging
- **Compliance Requirements**: Country-based compliance rules

## Quick Start

```go
package main

import (
    "context"
    "github.com/chimera-pool/chimera-pool-core/internal/security"
)

func main() {
    // Initialize security service with default configuration
    securityService := security.NewSecurityService(nil)
    ctx := context.Background()
    
    // Validate a request
    req := security.SecurityCheckRequest{
        UserID:    123,
        ClientID:  "client-ip-address",
        IPAddress: "192.168.1.1",
        UserAgent: "Mozilla/5.0",
        Action:    "api_call",
        Resource:  "mining",
        Input:     "user input data",
    }
    
    result, err := securityService.ValidateRequest(ctx, req)
    if err != nil {
        // Handle error
        return
    }
    
    if !result.Allowed {
        // Request blocked: result.Reason contains the reason
        // result.Violations contains specific violation types
        return
    }
    
    // Request allowed, proceed with normal processing
}
```

## Components

### MFA Service

```go
// Setup MFA for a user
mfaInfo, err := securityService.SetupMFA(userID, "ChimeraPool", "user@example.com")
if err != nil {
    return err
}

// mfaInfo contains:
// - Secret: Base32 encoded TOTP secret
// - QRCode: Base64 encoded PNG QR code
// - BackupCodes: Array of 10 backup codes

// Enable MFA
err = securityService.mfa.EnableMFA(userID, mfaInfo.Secret, mfaInfo.BackupCodes)

// Verify TOTP or backup code
valid, err := securityService.VerifyMFA(userID, "123456")
```

### Secure Wallet

```go
// Create secure wallet
walletID, err := securityService.CreateSecureWallet(
    "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa", // address
    "5HueCGU8rMjxEXxiPuD5BDku4MkFqeZyd4dZ1jvhTVqvbTLvyTJ", // private key
)

// Sign transaction
signature, err := securityService.SignTransaction(
    walletID,
    "5HueCGU8rMjxEXxiPuD5BDku4MkFqeZyd4dZ1jvhTVqvbTLvyTJ",
    []byte("transaction data"),
)
```

### Compliance

```go
// Check compliance requirements
requirements, err := securityService.CheckCompliance(userID, "US")
if requirements.RequiresKYC {
    // Submit KYC data
    kycData := security.KYCData{
        UserID:    userID,
        FirstName: "John",
        LastName:  "Doe",
        Email:     "john@example.com",
        Country:   "US",
    }
    err = securityService.SubmitKYC(kycData)
}

if requirements.RequiresAML {
    // Perform AML screening
    amlResult, err := securityService.PerformAMLScreening(userID, "John Doe")
}
```

## Configuration

```go
config := &security.SecurityConfig{
    RateLimiting: security.ProgressiveRateLimiterConfig{
        BaseRequestsPerMinute: 60,    // Base rate limit
        BaseBurstSize:         10,    // Burst allowance
        MaxPenaltyMultiplier:  10,    // Maximum penalty
        PenaltyDuration:       time.Hour,
        CleanupInterval:       time.Minute,
    },
    BruteForce: security.BruteForceConfig{
        MaxAttempts:     5,           // Max failed attempts
        WindowDuration:  time.Hour,   // Time window
        LockoutDuration: time.Hour,   // Lockout duration
        CleanupInterval: time.Minute,
    },
    DDoS: security.DDoSConfig{
        RequestsPerSecond:   20,      // Rate limit
        BurstSize:          50,       // Burst size
        SuspiciousThreshold: 200,     // Suspicious activity threshold
        BlockDuration:      time.Hour,
        CleanupInterval:    time.Minute,
    },
    IntrusionDetection: security.IntrusionDetectionConfig{
        SuspiciousPatterns: []string{
            `(?i)(union\s+select|select\s+.*\s+from)`, // SQL injection
            `(?i)(<script[^>]*>|javascript:)`,          // XSS
        },
        MaxViolationsPerHour: 10,
        BlockDuration:        24 * time.Hour,
        CleanupInterval:      time.Hour,
    },
}

securityService := security.NewSecurityService(config)
```

## Security Patterns Detected

The intrusion detection system recognizes these malicious patterns:

### SQL Injection
- `UNION SELECT` statements
- `INSERT INTO`, `DELETE FROM`, `DROP TABLE`
- `' OR '1'='1'` patterns
- Comment-based injections (`--`, `/*`)

### Cross-Site Scripting (XSS)
- `<script>` tags and variations
- `javascript:` URLs
- Event handlers (`onload`, `onerror`, `onclick`)
- `<iframe>`, `<object>`, `<embed>` tags

### Command Injection
- Shell command separators (`;`, `&&`, `||`)
- Common dangerous commands (`rm`, `del`, `format`)
- Backtick command execution

### Path Traversal
- Directory traversal patterns (`../`, `..\\`)
- URL-encoded variations (`%2e%2e%2f`)

### LDAP Injection
- LDAP filter manipulation patterns

## Audit Logging

All security events are automatically logged:

```go
// Retrieve user audit logs
logs, err := securityService.GetUserAuditLogs(userID, 50)

// Log custom security event
err = securityService.LogSecurityEvent(security.AuditEvent{
    UserID:    userID,
    Action:    "custom_action",
    Resource:  "resource_name",
    IPAddress: "192.168.1.1",
    Success:   true,
    Metadata: map[string]interface{}{
        "custom_field": "value",
    },
})
```

## Client Blocking

Check if a client is blocked by any security component:

```go
status, err := securityService.IsClientBlocked(ctx, "client-ip")
if status.IsBlocked {
    // Client is blocked
    if status.IntrusionBlocked {
        // Blocked due to malicious activity
    }
    if status.DDoSBlocked {
        // Blocked due to DDoS protection
    }
    if status.UnderPenalty {
        // Under rate limiting penalty
    }
}
```

## Data Encryption

```go
// Encrypt sensitive data for storage
encrypted, err := securityService.EncryptSensitiveData("sensitive information")

// Decrypt data when needed
decrypted, err := securityService.DecryptSensitiveData(encrypted)
```

## Testing

The framework includes comprehensive tests:

```bash
# Run all security tests
go test ./internal/security/...

# Run with coverage
go test -cover ./internal/security/...

# Run specific test suites
go test ./internal/security/ -run TestMFA
go test ./internal/security/ -run TestRateLimit
go test ./internal/security/ -run TestEncryption
go test ./internal/security/ -run TestCompleteSecurityWorkflow
```

## Performance

The security framework is designed for high performance:

- **Rate Limiting**: Handles 1000+ requests/second
- **Intrusion Detection**: Pattern matching optimized with compiled regex
- **Encryption**: Hardware-accelerated AES when available
- **Concurrent Safe**: All components are thread-safe
- **Memory Efficient**: Automatic cleanup of old data

## Production Deployment

### Environment Variables

```bash
# Master encryption key (base64 encoded)
SECURITY_MASTER_KEY="base64-encoded-key"

# Rate limiting configuration
SECURITY_RATE_LIMIT_RPM=60
SECURITY_RATE_LIMIT_BURST=10

# Brute force protection
SECURITY_BRUTE_FORCE_MAX_ATTEMPTS=5
SECURITY_BRUTE_FORCE_LOCKOUT_DURATION=3600

# DDoS protection
SECURITY_DDOS_RPS=20
SECURITY_DDOS_BURST=50
```

### Database Schema

The security framework requires these database tables:

```sql
-- MFA secrets and backup codes
CREATE TABLE mfa_secrets (
    user_id BIGINT PRIMARY KEY,
    secret_encrypted TEXT NOT NULL,
    enabled BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE mfa_backup_codes (
    user_id BIGINT,
    code_hash TEXT,
    used BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (user_id, code_hash)
);

-- Secure wallets
CREATE TABLE secure_wallets (
    id TEXT PRIMARY KEY,
    user_id BIGINT,
    address TEXT NOT NULL,
    private_key_encrypted TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Compliance data
CREATE TABLE kyc_data (
    user_id BIGINT PRIMARY KEY,
    first_name_encrypted TEXT,
    last_name_encrypted TEXT,
    email_encrypted TEXT,
    country TEXT,
    status TEXT DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT NOW()
);

-- Audit logs
CREATE TABLE audit_logs (
    id TEXT PRIMARY KEY,
    user_id BIGINT,
    action TEXT NOT NULL,
    resource TEXT NOT NULL,
    ip_address TEXT,
    user_agent TEXT,
    success BOOLEAN,
    error_message TEXT,
    metadata JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);
```

## Security Best Practices

1. **Key Management**: Use proper key management systems in production
2. **Rate Limiting**: Adjust limits based on your traffic patterns
3. **Monitoring**: Set up alerts for security violations
4. **Regular Updates**: Keep security patterns updated
5. **Backup**: Ensure audit logs are backed up regularly
6. **Testing**: Regularly test security measures with penetration testing

## Contributing

When contributing to the security framework:

1. **Write Tests**: All security code must have comprehensive tests
2. **Security Review**: Security changes require additional review
3. **Documentation**: Update documentation for any API changes
4. **Performance**: Ensure changes don't impact performance
5. **Backward Compatibility**: Maintain API compatibility when possible

## License

This security framework is part of the Chimera Mining Pool project and follows the same license terms.