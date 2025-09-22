package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// AESEncryptor handles AES encryption/decryption
type AESEncryptor struct{}

// NewAESEncryptor creates a new AES encryptor
func NewAESEncryptor() *AESEncryptor {
	return &AESEncryptor{}
}

// GenerateKey generates a new 256-bit AES key
func (e *AESEncryptor) GenerateKey() ([]byte, error) {
	key := make([]byte, 32) // AES-256
	_, err := rand.Read(key)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}
	return key, nil
}

// Encrypt encrypts data using AES-GCM
func (e *AESEncryptor) Encrypt(plaintext, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, errors.New("key must be 32 bytes for AES-256")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts data using AES-GCM
func (e *AESEncryptor) Decrypt(ciphertext, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, errors.New("key must be 32 bytes for AES-256")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// PasswordHasher handles password hashing and verification
type PasswordHasher struct {
	cost int
}

// NewPasswordHasher creates a new password hasher
func NewPasswordHasher() *PasswordHasher {
	return &PasswordHasher{
		cost: bcrypt.DefaultCost,
	}
}

// HashPassword hashes a password using bcrypt
func (h *PasswordHasher) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

// VerifyPassword verifies a password against its hash
func (h *PasswordHasher) VerifyPassword(password, hash string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return false, nil
		}
		return false, fmt.Errorf("failed to verify password: %w", err)
	}
	return true, nil
}

// SecureWallet handles secure wallet operations
type SecureWallet struct {
	encryptor *AESEncryptor
	wallets   map[string]*WalletInfo
	keys      map[string][]byte // In production, this would be in secure storage
}

// WalletInfo contains wallet information
type WalletInfo struct {
	ID                   string
	Address              string
	EncryptedPrivateKey  string
	CreatedAt            time.Time
}

// NewSecureWallet creates a new secure wallet manager
func NewSecureWallet() *SecureWallet {
	return &SecureWallet{
		encryptor: NewAESEncryptor(),
		wallets:   make(map[string]*WalletInfo),
		keys:      make(map[string][]byte),
	}
}

// CreateWallet creates a new secure wallet
func (w *SecureWallet) CreateWallet(address, privateKey string) (string, error) {
	if address == "" {
		return "", errors.New("wallet address is required")
	}
	if privateKey == "" {
		return "", errors.New("private key is required")
	}

	// Generate wallet ID
	walletID := w.generateWalletID()

	// Generate encryption key
	key, err := w.encryptor.GenerateKey()
	if err != nil {
		return "", fmt.Errorf("failed to generate encryption key: %w", err)
	}

	// Encrypt private key
	encryptedKey, err := w.encryptor.Encrypt([]byte(privateKey), key)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt private key: %w", err)
	}

	// Store wallet info
	wallet := &WalletInfo{
		ID:                  walletID,
		Address:             address,
		EncryptedPrivateKey: base64.StdEncoding.EncodeToString(encryptedKey),
		CreatedAt:           time.Now(),
	}

	w.wallets[walletID] = wallet
	w.keys[walletID] = key

	return walletID, nil
}

// GetWallet retrieves wallet information
func (w *SecureWallet) GetWallet(walletID string) (*WalletInfo, error) {
	wallet, exists := w.wallets[walletID]
	if !exists {
		return nil, errors.New("wallet not found")
	}

	// Return a copy to prevent external modification
	return &WalletInfo{
		ID:                  wallet.ID,
		Address:             wallet.Address,
		EncryptedPrivateKey: wallet.EncryptedPrivateKey,
		CreatedAt:           wallet.CreatedAt,
	}, nil
}

// VerifyWallet verifies a wallet with its private key
func (w *SecureWallet) VerifyWallet(walletID, privateKey string) (bool, error) {
	wallet, exists := w.wallets[walletID]
	if !exists {
		return false, errors.New("wallet not found")
	}

	key, exists := w.keys[walletID]
	if !exists {
		return false, errors.New("encryption key not found")
	}

	// Decrypt stored private key
	encryptedKey, err := base64.StdEncoding.DecodeString(wallet.EncryptedPrivateKey)
	if err != nil {
		return false, fmt.Errorf("failed to decode encrypted key: %w", err)
	}

	decryptedKey, err := w.encryptor.Decrypt(encryptedKey, key)
	if err != nil {
		return false, fmt.Errorf("failed to decrypt private key: %w", err)
	}

	return string(decryptedKey) == privateKey, nil
}

// SignTransaction signs transaction data with the wallet's private key
func (w *SecureWallet) SignTransaction(walletID, privateKey string, transactionData []byte) (string, error) {
	// Verify wallet first
	valid, err := w.VerifyWallet(walletID, privateKey)
	if err != nil {
		return "", err
	}
	if !valid {
		return "", errors.New("invalid private key")
	}

	// For this implementation, we'll use a simple hash-based signature
	// In production, this would use proper cryptographic signing
	hash := sha256.Sum256(append([]byte(privateKey), transactionData...))
	signature := base64.StdEncoding.EncodeToString(hash[:])

	return signature, nil
}

// VerifySignature verifies a transaction signature
func (w *SecureWallet) VerifySignature(walletID string, transactionData []byte, signature string) (bool, error) {
	wallet, exists := w.wallets[walletID]
	if !exists {
		return false, errors.New("wallet not found")
	}

	key, exists := w.keys[walletID]
	if !exists {
		return false, errors.New("encryption key not found")
	}

	// Decrypt private key
	encryptedKey, err := base64.StdEncoding.DecodeString(wallet.EncryptedPrivateKey)
	if err != nil {
		return false, fmt.Errorf("failed to decode encrypted key: %w", err)
	}

	decryptedKey, err := w.encryptor.Decrypt(encryptedKey, key)
	if err != nil {
		return false, fmt.Errorf("failed to decrypt private key: %w", err)
	}

	// Recreate signature
	hash := sha256.Sum256(append(decryptedKey, transactionData...))
	expectedSignature := base64.StdEncoding.EncodeToString(hash[:])

	return signature == expectedSignature, nil
}

func (w *SecureWallet) generateWalletID() string {
	// Generate a random wallet ID
	id := make([]byte, 16)
	rand.Read(id)
	return base64.URLEncoding.EncodeToString(id)
}

// ComplianceManager handles regulatory compliance
type ComplianceManager struct {
	kycData map[int64]*KYCData
	amlData map[int64]*AMLResult
}

// ComplianceRequirements defines what compliance is required
type ComplianceRequirements struct {
	RequiresKYC bool
	RequiresAML bool
	Country     string
}

// KYCStatus represents KYC verification status
type KYCStatus string

const (
	KYCStatusPending  KYCStatus = "pending"
	KYCStatusApproved KYCStatus = "approved"
	KYCStatusRejected KYCStatus = "rejected"
)

// KYCData contains KYC information
type KYCData struct {
	UserID    int64
	FirstName string
	LastName  string
	Email     string
	Country   string
	Status    KYCStatus
	CreatedAt time.Time
}

// AMLResult contains AML screening results
type AMLResult struct {
	UserID      int64
	FullName    string
	RiskScore   int
	IsHighRisk  bool
	ScreenedAt  time.Time
}

// NewComplianceManager creates a new compliance manager
func NewComplianceManager() *ComplianceManager {
	return &ComplianceManager{
		kycData: make(map[int64]*KYCData),
		amlData: make(map[int64]*AMLResult),
	}
}

// GetComplianceRequirements returns compliance requirements for a user
func (c *ComplianceManager) GetComplianceRequirements(userID int64, country string) (*ComplianceRequirements, error) {
	// Define countries that require compliance
	regulatedCountries := map[string]bool{
		"US": true,
		"GB": true,
		"DE": true,
		"FR": true,
		"JP": true,
		"CA": true,
		"AU": true,
	}

	requiresCompliance := regulatedCountries[country]

	return &ComplianceRequirements{
		RequiresKYC: requiresCompliance,
		RequiresAML: requiresCompliance,
		Country:     country,
	}, nil
}

// SubmitKYC submits KYC data for verification
func (c *ComplianceManager) SubmitKYC(data KYCData) error {
	if data.UserID <= 0 {
		return errors.New("invalid user ID")
	}
	if data.FirstName == "" || data.LastName == "" {
		return errors.New("first name and last name are required")
	}
	if data.Email == "" {
		return errors.New("email is required")
	}

	data.Status = KYCStatusPending
	data.CreatedAt = time.Now()

	c.kycData[data.UserID] = &data
	return nil
}

// GetKYCStatus returns the KYC status for a user
func (c *ComplianceManager) GetKYCStatus(userID int64) (KYCStatus, error) {
	data, exists := c.kycData[userID]
	if !exists {
		return "", errors.New("KYC data not found")
	}
	return data.Status, nil
}

// PerformAMLScreening performs AML screening for a user
func (c *ComplianceManager) PerformAMLScreening(userID int64, fullName string) (*AMLResult, error) {
	if userID <= 0 {
		return nil, errors.New("invalid user ID")
	}
	if fullName == "" {
		return nil, errors.New("full name is required")
	}

	// Simple risk scoring based on name patterns (in production, this would use real AML services)
	riskScore := 0
	isHighRisk := false

	// Check for high-risk patterns
	highRiskPatterns := []string{"test", "fake", "anonymous"}
	for _, pattern := range highRiskPatterns {
		if contains(fullName, pattern) {
			riskScore += 50
		}
	}

	if riskScore >= 50 {
		isHighRisk = true
	}

	result := &AMLResult{
		UserID:     userID,
		FullName:   fullName,
		RiskScore:  riskScore,
		IsHighRisk: isHighRisk,
		ScreenedAt: time.Now(),
	}

	c.amlData[userID] = result
	return result, nil
}

// AuditLogger handles security audit logging
type AuditLogger struct {
	logs []AuditEvent
}

// AuditEvent represents a security audit event
type AuditEvent struct {
	ID        string
	UserID    int64
	Action    string
	Resource  string
	IPAddress string
	UserAgent string
	Success   bool
	Error     string
	Metadata  map[string]interface{}
	Timestamp time.Time
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger() *AuditLogger {
	return &AuditLogger{
		logs: make([]AuditEvent, 0),
	}
}

// LogEvent logs an audit event
func (a *AuditLogger) LogEvent(event AuditEvent) error {
	// Generate event ID
	event.ID = a.generateEventID()
	event.Timestamp = time.Now()

	a.logs = append(a.logs, event)
	return nil
}

// GetUserAuditLogs retrieves audit logs for a user
func (a *AuditLogger) GetUserAuditLogs(userID int64, limit int) ([]AuditEvent, error) {
	userLogs := make([]AuditEvent, 0)
	
	// Filter logs for the user (in reverse chronological order)
	for i := len(a.logs) - 1; i >= 0 && len(userLogs) < limit; i-- {
		if a.logs[i].UserID == userID {
			userLogs = append(userLogs, a.logs[i])
		}
	}

	return userLogs, nil
}

func (a *AuditLogger) generateEventID() string {
	id := make([]byte, 8)
	rand.Read(id)
	return base64.URLEncoding.EncodeToString(id)
}

// DataEncryptor handles encryption of sensitive data at rest
type DataEncryptor struct {
	encryptor *AESEncryptor
	masterKey []byte
}

// NewDataEncryptor creates a new data encryptor
func NewDataEncryptor() *DataEncryptor {
	encryptor := NewAESEncryptor()
	
	// In production, this would be loaded from secure key management
	masterKey, _ := encryptor.GenerateKey()

	return &DataEncryptor{
		encryptor: encryptor,
		masterKey: masterKey,
	}
}

// EncryptSensitiveData encrypts sensitive data for storage
func (d *DataEncryptor) EncryptSensitiveData(data string) (string, error) {
	encrypted, err := d.encryptor.Encrypt([]byte(data), d.masterKey)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt sensitive data: %w", err)
	}

	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// DecryptSensitiveData decrypts sensitive data from storage
func (d *DataEncryptor) DecryptSensitiveData(encryptedData string) (string, error) {
	encrypted, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return "", fmt.Errorf("failed to decode encrypted data: %w", err)
	}

	decrypted, err := d.encryptor.Decrypt(encrypted, d.masterKey)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt sensitive data: %w", err)
	}

	return string(decrypted), nil
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && 
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		 len(s) > len(substr) && s[1:len(substr)+1] == substr))
}