package security

import (
	"context"
	"time"
)

// =============================================================================
// ISP-COMPLIANT SECURITY INTERFACES
// Each interface is small and focused on a single responsibility
// Enables easy mocking, testing, and swapping implementations
// =============================================================================

// -----------------------------------------------------------------------------
// Rate Limiting Interfaces
// -----------------------------------------------------------------------------

// RateLimitChecker checks if requests should be allowed
type RateLimitChecker interface {
	Allow(ctx context.Context, clientID string) (bool, error)
}

// RateLimitRecorder records rate limit violations
type RateLimitRecorder interface {
	RecordViolation(ctx context.Context, clientID string, violationType ViolationType) error
}

// RateLimitInfoProvider provides rate limit status information
type RateLimitInfoProvider interface {
	GetClientInfo(ctx context.Context, clientID string) (*ClientInfo, error)
}

// ProgressiveRateLimiting combines all rate limiting capabilities
type ProgressiveRateLimiting interface {
	RateLimitChecker
	RateLimitRecorder
	RateLimitInfoProvider
}

// -----------------------------------------------------------------------------
// Authentication Protection Interfaces
// -----------------------------------------------------------------------------

// BruteForceChecker checks if authentication attempts are allowed
type BruteForceChecker interface {
	CheckAttempt(ctx context.Context, clientID string) (bool, error)
}

// BruteForceRecorder records authentication attempts
type BruteForceRecorder interface {
	RecordFailedAttempt(ctx context.Context, clientID string) error
	RecordSuccessfulAttempt(ctx context.Context, clientID string) error
}

// BruteForceProtection combines checking and recording
type BruteForceProtection interface {
	BruteForceChecker
	BruteForceRecorder
}

// -----------------------------------------------------------------------------
// DDoS Protection Interfaces
// -----------------------------------------------------------------------------

// DDoSChecker checks if requests are allowed (DDoS protection)
type DDoSChecker interface {
	CheckRequest(ctx context.Context, clientID string) (bool, error)
}

// DDoSInfoProvider provides DDoS status information
type DDoSInfoProvider interface {
	GetClientInfo(ctx context.Context, clientID string) (*DDoSClientInfo, error)
}

// DDoSProtection combines DDoS capabilities
type DDoSProtection interface {
	DDoSChecker
	DDoSInfoProvider
}

// -----------------------------------------------------------------------------
// Intrusion Detection Interfaces
// -----------------------------------------------------------------------------

// ThreatAnalyzer analyzes requests for threats
type ThreatAnalyzer interface {
	AnalyzeRequest(ctx context.Context, clientID, input string) (*ThreatInfo, error)
}

// BlockChecker checks if a client is blocked
type BlockChecker interface {
	IsBlocked(ctx context.Context, clientID string) (bool, error)
}

// IntrusionDetection combines threat analysis and blocking
type IntrusionDetection interface {
	ThreatAnalyzer
	BlockChecker
}

// -----------------------------------------------------------------------------
// Encryption Interfaces
// -----------------------------------------------------------------------------

// Encryptor encrypts data
type Encryptor interface {
	Encrypt(plaintext, key []byte) ([]byte, error)
}

// Decryptor decrypts data
type Decryptor interface {
	Decrypt(ciphertext, key []byte) ([]byte, error)
}

// KeyGenerator generates encryption keys
type KeyGenerator interface {
	GenerateKey() ([]byte, error)
}

// SymmetricCrypto combines encryption capabilities
type SymmetricCrypto interface {
	Encryptor
	Decryptor
	KeyGenerator
}

// -----------------------------------------------------------------------------
// Password Interfaces
// -----------------------------------------------------------------------------

// PasswordHasherService hashes passwords
type PasswordHasherService interface {
	HashPassword(password string) (string, error)
}

// PasswordVerifier verifies passwords
type PasswordVerifier interface {
	VerifyPassword(password, hash string) (bool, error)
}

// PasswordManager combines hashing and verification
type PasswordManager interface {
	PasswordHasherService
	PasswordVerifier
}

// -----------------------------------------------------------------------------
// MFA Interfaces
// -----------------------------------------------------------------------------

// TOTPGenerator generates TOTP secrets
type TOTPGenerator interface {
	GenerateTOTPSecret(userID int64, issuer, accountName string) (secret, qrCode string, err error)
}

// TOTPValidator validates TOTP codes
type TOTPValidator interface {
	ValidateTOTP(secret, code string) bool
}

// BackupCodeGenerator generates backup codes
type BackupCodeGenerator interface {
	GenerateBackupCodes(userID int64, count int) ([]string, error)
}

// BackupCodeValidator validates backup codes
type BackupCodeValidator interface {
	ValidateBackupCode(userID int64, code string) (bool, error)
}

// MFAManager combines all MFA capabilities
type MFAManager interface {
	TOTPGenerator
	TOTPValidator
	BackupCodeGenerator
	BackupCodeValidator
}

// -----------------------------------------------------------------------------
// Wallet Security Interfaces
// -----------------------------------------------------------------------------

// WalletCreator creates secure wallets
type WalletCreator interface {
	CreateWallet(address, privateKey string) (walletID string, err error)
}

// WalletVerifier verifies wallet ownership
type WalletVerifier interface {
	VerifyWallet(walletID, privateKey string) (bool, error)
}

// TransactionSigner signs transactions
type TransactionSigner interface {
	SignTransaction(walletID, privateKey string, transactionData []byte) (signature string, err error)
}

// SignatureVerifier verifies transaction signatures
type SignatureVerifier interface {
	VerifySignature(walletID string, transactionData []byte, signature string) (bool, error)
}

// SecureWalletManager combines wallet security operations
type SecureWalletManager interface {
	WalletCreator
	WalletVerifier
	TransactionSigner
	SignatureVerifier
}

// -----------------------------------------------------------------------------
// Compliance Interfaces
// -----------------------------------------------------------------------------

// ComplianceChecker checks compliance requirements
type ComplianceChecker interface {
	GetComplianceRequirements(userID int64, country string) (*ComplianceRequirements, error)
}

// KYCSubmitter submits KYC data
type KYCSubmitter interface {
	SubmitKYC(data KYCData) error
}

// KYCStatusChecker checks KYC status
type KYCStatusChecker interface {
	GetKYCStatus(userID int64) (KYCStatus, error)
}

// AMLScreener performs AML screening
type AMLScreener interface {
	PerformAMLScreening(userID int64, fullName string) (*AMLResult, error)
}

// ComplianceService combines all compliance operations
type ComplianceService interface {
	ComplianceChecker
	KYCSubmitter
	KYCStatusChecker
	AMLScreener
}

// -----------------------------------------------------------------------------
// Audit Interfaces
// -----------------------------------------------------------------------------

// AuditEventLogger logs audit events
type AuditEventLogger interface {
	LogEvent(event AuditEvent) error
}

// AuditEventReader reads audit events
type AuditEventReader interface {
	GetUserAuditLogs(userID int64, limit int) ([]AuditEvent, error)
}

// AuditService combines audit operations
type AuditService interface {
	AuditEventLogger
	AuditEventReader
}

// -----------------------------------------------------------------------------
// Data Protection Interfaces
// -----------------------------------------------------------------------------

// SensitiveDataEncryptor encrypts sensitive data for storage
type SensitiveDataEncryptor interface {
	EncryptSensitiveData(data string) (string, error)
}

// SensitiveDataDecryptor decrypts sensitive data from storage
type SensitiveDataDecryptor interface {
	DecryptSensitiveData(encryptedData string) (string, error)
}

// DataProtection combines data encryption operations
type DataProtection interface {
	SensitiveDataEncryptor
	SensitiveDataDecryptor
}

// -----------------------------------------------------------------------------
// Composite Security Interface
// -----------------------------------------------------------------------------

// SecurityValidator performs comprehensive security validation
type SecurityValidator interface {
	ValidateRequest(ctx context.Context, req SecurityCheckRequest) (*SecurityCheckResult, error)
}

// ClientBlockManager manages client blocking
type ClientBlockManager interface {
	IsClientBlocked(ctx context.Context, clientID string) (*ClientBlockStatus, error)
	UnblockClient(ctx context.Context, clientID string) error
}

// AuthenticationRecorder records authentication attempts
type AuthenticationRecorder interface {
	RecordAuthenticationAttempt(ctx context.Context, clientID string, success bool) error
}

// -----------------------------------------------------------------------------
// Key Management Interface (for production)
// -----------------------------------------------------------------------------

// KeyProvider provides encryption keys from secure storage
type KeyProvider interface {
	GetMasterKey(ctx context.Context, keyID string) ([]byte, error)
	RotateMasterKey(ctx context.Context, keyID string) error
}

// KeyStore stores and retrieves keys
type KeyStore interface {
	StoreKey(ctx context.Context, keyID string, key []byte) error
	RetrieveKey(ctx context.Context, keyID string) ([]byte, error)
	DeleteKey(ctx context.Context, keyID string) error
}

// SecureKeyManager combines key management operations
type SecureKeyManager interface {
	KeyProvider
	KeyStore
}

// -----------------------------------------------------------------------------
// Distributed Security Interfaces (for horizontal scaling)
// -----------------------------------------------------------------------------

// DistributedRateLimiter rate limits across multiple instances
type DistributedRateLimiter interface {
	RateLimitChecker
	// Sync synchronizes state with distributed store
	Sync(ctx context.Context) error
}

// DistributedBlockList manages blocks across instances
type DistributedBlockList interface {
	AddToBlockList(ctx context.Context, clientID string, duration time.Duration, reason string) error
	RemoveFromBlockList(ctx context.Context, clientID string) error
	IsOnBlockList(ctx context.Context, clientID string) (bool, error)
}

// -----------------------------------------------------------------------------
// Security Metrics Interface
// -----------------------------------------------------------------------------

// SecurityMetricsCollector collects security metrics
type SecurityMetricsCollector interface {
	RecordRateLimitHit(clientID string)
	RecordBruteForceAttempt(clientID string, success bool)
	RecordDDoSBlock(clientID string)
	RecordIntrusionAttempt(clientID string, threatType string)
}

// SecurityMetricsReader reads security metrics
type SecurityMetricsReader interface {
	GetMetrics(ctx context.Context) (*SecurityMetrics, error)
	GetClientMetrics(ctx context.Context, clientID string) (*ClientSecurityMetrics, error)
}

// ClientSecurityMetrics contains security metrics for a specific client
type ClientSecurityMetrics struct {
	ClientID           string
	RateLimitHits      int64
	BruteForceAttempts int64
	DDoSBlocks         int64
	IntrusionAttempts  int64
	LastActivity       time.Time
}
