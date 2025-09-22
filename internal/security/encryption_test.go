package security

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAESEncryption(t *testing.T) {
	encryptor := NewAESEncryptor()
	
	tests := []struct {
		name      string
		plaintext string
		wantError bool
	}{
		{
			name:      "normal text encryption",
			plaintext: "Hello, World!",
			wantError: false,
		},
		{
			name:      "empty text encryption",
			plaintext: "",
			wantError: false,
		},
		{
			name:      "long text encryption",
			plaintext: "This is a very long text that should be encrypted properly without any issues even though it's quite lengthy and contains various characters including numbers 123456789 and symbols !@#$%^&*()",
			wantError: false,
		},
		{
			name:      "unicode text encryption",
			plaintext: "Hello 世界 🌍 Здравствуй мир",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate key
			key, err := encryptor.GenerateKey()
			require.NoError(t, err)
			require.Len(t, key, 32) // AES-256 key

			// Encrypt
			ciphertext, err := encryptor.Encrypt([]byte(tt.plaintext), key)
			if tt.wantError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotEmpty(t, ciphertext)
			assert.NotEqual(t, tt.plaintext, string(ciphertext))

			// Decrypt
			decrypted, err := encryptor.Decrypt(ciphertext, key)
			require.NoError(t, err)
			assert.Equal(t, tt.plaintext, string(decrypted))
		})
	}
}

func TestAESEncryptionWithWrongKey(t *testing.T) {
	encryptor := NewAESEncryptor()
	plaintext := "secret message"

	// Generate two different keys
	key1, err := encryptor.GenerateKey()
	require.NoError(t, err)

	key2, err := encryptor.GenerateKey()
	require.NoError(t, err)

	// Encrypt with key1
	ciphertext, err := encryptor.Encrypt([]byte(plaintext), key1)
	require.NoError(t, err)

	// Try to decrypt with key2 (should fail)
	_, err = encryptor.Decrypt(ciphertext, key2)
	assert.Error(t, err)
}

func TestAESEncryptionKeyValidation(t *testing.T) {
	encryptor := NewAESEncryptor()
	plaintext := "test message"

	tests := []struct {
		name    string
		keySize int
		wantErr bool
	}{
		{
			name:    "valid 32-byte key",
			keySize: 32,
			wantErr: false,
		},
		{
			name:    "invalid 16-byte key",
			keySize: 16,
			wantErr: true,
		},
		{
			name:    "invalid 24-byte key",
			keySize: 24,
			wantErr: true,
		},
		{
			name:    "invalid empty key",
			keySize: 0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := make([]byte, tt.keySize)
			
			_, err := encryptor.Encrypt([]byte(plaintext), key)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPasswordHashing(t *testing.T) {
	hasher := NewPasswordHasher()

	tests := []struct {
		name     string
		password string
	}{
		{
			name:     "normal password",
			password: "mySecurePassword123!",
		},
		{
			name:     "short password",
			password: "abc",
		},
		{
			name:     "long password",
			password: "thisIsAVeryLongPasswordThatShouldStillWorkProperly123456789!@#$%^&*()",
		},
		{
			name:     "unicode password",
			password: "пароль123世界🔒",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Hash password
			hash, err := hasher.HashPassword(tt.password)
			require.NoError(t, err)
			assert.NotEmpty(t, hash)
			assert.NotEqual(t, tt.password, hash)

			// Verify correct password
			valid, err := hasher.VerifyPassword(tt.password, hash)
			require.NoError(t, err)
			assert.True(t, valid)

			// Verify wrong password
			valid, err = hasher.VerifyPassword(tt.password+"wrong", hash)
			require.NoError(t, err)
			assert.False(t, valid)
		})
	}
}

func TestPasswordHashingConsistency(t *testing.T) {
	hasher := NewPasswordHasher()
	password := "testPassword123"

	// Hash the same password multiple times
	hash1, err := hasher.HashPassword(password)
	require.NoError(t, err)

	hash2, err := hasher.HashPassword(password)
	require.NoError(t, err)

	// Hashes should be different (due to salt)
	assert.NotEqual(t, hash1, hash2)

	// But both should verify correctly
	valid1, err := hasher.VerifyPassword(password, hash1)
	require.NoError(t, err)
	assert.True(t, valid1)

	valid2, err := hasher.VerifyPassword(password, hash2)
	require.NoError(t, err)
	assert.True(t, valid2)
}

func TestSecureWalletIntegration(t *testing.T) {
	wallet := NewSecureWallet()

	tests := []struct {
		name           string
		walletAddress  string
		privateKey     string
		expectedValid  bool
	}{
		{
			name:          "valid wallet creation",
			walletAddress: "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa", // Example Bitcoin address
			privateKey:    "5HueCGU8rMjxEXxiPuD5BDku4MkFqeZyd4dZ1jvhTVqvbTLvyTJ", // Example private key
			expectedValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create wallet
			walletID, err := wallet.CreateWallet(tt.walletAddress, tt.privateKey)
			if tt.expectedValid {
				require.NoError(t, err)
				assert.NotEmpty(t, walletID)
			} else {
				assert.Error(t, err)
				return
			}

			// Retrieve wallet
			retrievedWallet, err := wallet.GetWallet(walletID)
			require.NoError(t, err)
			assert.Equal(t, tt.walletAddress, retrievedWallet.Address)
			
			// Private key should be encrypted
			assert.NotEqual(t, tt.privateKey, retrievedWallet.EncryptedPrivateKey)
			assert.NotEmpty(t, retrievedWallet.EncryptedPrivateKey)

			// Verify wallet
			valid, err := wallet.VerifyWallet(walletID, tt.privateKey)
			require.NoError(t, err)
			assert.True(t, valid)

			// Test wrong private key
			valid, err = wallet.VerifyWallet(walletID, "wrongkey")
			require.NoError(t, err)
			assert.False(t, valid)
		})
	}
}

func TestSecureWalletTransactionSigning(t *testing.T) {
	wallet := NewSecureWallet()
	
	// Create a test wallet
	walletAddress := "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa"
	privateKey := "5HueCGU8rMjxEXxiPuD5BDku4MkFqeZyd4dZ1jvhTVqvbTLvyTJ"
	
	walletID, err := wallet.CreateWallet(walletAddress, privateKey)
	require.NoError(t, err)

	// Test transaction signing
	transactionData := "test transaction data"
	signature, err := wallet.SignTransaction(walletID, privateKey, []byte(transactionData))
	require.NoError(t, err)
	assert.NotEmpty(t, signature)

	// Verify signature
	valid, err := wallet.VerifySignature(walletID, []byte(transactionData), signature)
	require.NoError(t, err)
	assert.True(t, valid)

	// Test with wrong data
	valid, err = wallet.VerifySignature(walletID, []byte("wrong data"), signature)
	require.NoError(t, err)
	assert.False(t, valid)
}

func TestComplianceFeatures(t *testing.T) {
	compliance := NewComplianceManager()

	tests := []struct {
		name           string
		userID         int64
		country        string
		expectedKYC    bool
		expectedAML    bool
	}{
		{
			name:        "US user requires full compliance",
			userID:      123,
			country:     "US",
			expectedKYC: true,
			expectedAML: true,
		},
		{
			name:        "EU user requires full compliance",
			userID:      124,
			country:     "DE",
			expectedKYC: true,
			expectedAML: true,
		},
		{
			name:        "non-regulated country",
			userID:      125,
			country:     "XX",
			expectedKYC: false,
			expectedAML: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Check compliance requirements
			requirements, err := compliance.GetComplianceRequirements(tt.userID, tt.country)
			require.NoError(t, err)
			
			assert.Equal(t, tt.expectedKYC, requirements.RequiresKYC)
			assert.Equal(t, tt.expectedAML, requirements.RequiresAML)

			if requirements.RequiresKYC {
				// Test KYC verification
				kycData := KYCData{
					UserID:    tt.userID,
					FirstName: "John",
					LastName:  "Doe",
					Email:     "john.doe@example.com",
					Country:   tt.country,
				}

				err := compliance.SubmitKYC(kycData)
				require.NoError(t, err)

				// Check KYC status
				status, err := compliance.GetKYCStatus(tt.userID)
				require.NoError(t, err)
				assert.Equal(t, KYCStatusPending, status)
			}

			if requirements.RequiresAML {
				// Test AML screening
				amlResult, err := compliance.PerformAMLScreening(tt.userID, "John Doe")
				require.NoError(t, err)
				assert.NotNil(t, amlResult)
			}
		})
	}
}

func TestAuditLogging(t *testing.T) {
	auditor := NewAuditLogger()

	tests := []struct {
		name   string
		event  AuditEvent
	}{
		{
			name: "user login event",
			event: AuditEvent{
				UserID:    123,
				Action:    "user_login",
				Resource:  "auth",
				IPAddress: "192.168.1.1",
				UserAgent: "Mozilla/5.0",
				Success:   true,
			},
		},
		{
			name: "failed login event",
			event: AuditEvent{
				UserID:    123,
				Action:    "user_login",
				Resource:  "auth",
				IPAddress: "192.168.1.1",
				UserAgent: "Mozilla/5.0",
				Success:   false,
				Error:     "invalid credentials",
			},
		},
		{
			name: "wallet transaction event",
			event: AuditEvent{
				UserID:    123,
				Action:    "wallet_transaction",
				Resource:  "wallet",
				IPAddress: "192.168.1.1",
				Success:   true,
				Metadata: map[string]interface{}{
					"amount":      "0.001",
					"destination": "1BvBMSEYstWetqTFn5Au4m4GFg7xJaNVN2",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Log audit event
			err := auditor.LogEvent(tt.event)
			require.NoError(t, err)

			// Retrieve audit logs
			logs, err := auditor.GetUserAuditLogs(tt.event.UserID, 10)
			require.NoError(t, err)
			assert.NotEmpty(t, logs)

			// Find our event
			found := false
			for _, log := range logs {
				if log.Action == tt.event.Action && log.Resource == tt.event.Resource {
					found = true
					assert.Equal(t, tt.event.UserID, log.UserID)
					assert.Equal(t, tt.event.IPAddress, log.IPAddress)
					assert.Equal(t, tt.event.Success, log.Success)
					break
				}
			}
			assert.True(t, found, "audit event should be found in logs")
		})
	}
}

func TestDataEncryptionAtRest(t *testing.T) {
	encryptor := NewDataEncryptor()

	// Test sensitive data encryption
	sensitiveData := map[string]interface{}{
		"ssn":           "123-45-6789",
		"credit_card":   "4111-1111-1111-1111",
		"private_key":   "5HueCGU8rMjxEXxiPuD5BDku4MkFqeZyd4dZ1jvhTVqvbTLvyTJ",
		"personal_info": "John Doe, 123 Main St, Anytown USA",
	}

	for field, value := range sensitiveData {
		t.Run("encrypt_"+field, func(t *testing.T) {
			// Encrypt data
			encrypted, err := encryptor.EncryptSensitiveData(value.(string))
			require.NoError(t, err)
			assert.NotEmpty(t, encrypted)
			assert.NotEqual(t, value, encrypted)

			// Decrypt data
			decrypted, err := encryptor.DecryptSensitiveData(encrypted)
			require.NoError(t, err)
			assert.Equal(t, value, decrypted)
		})
	}
}

func TestEndToEndEncryption(t *testing.T) {
	// Test complete encryption workflow
	encryptor := NewAESEncryptor()
	hasher := NewPasswordHasher()

	// User registration with encrypted data
	userPassword := "userSecurePassword123!"
	sensitiveData := "user private information"

	// Hash password
	passwordHash, err := hasher.HashPassword(userPassword)
	require.NoError(t, err)

	// Generate encryption key from password
	key, err := encryptor.GenerateKey()
	require.NoError(t, err)

	// Encrypt sensitive data
	encryptedData, err := encryptor.Encrypt([]byte(sensitiveData), key)
	require.NoError(t, err)

	// Simulate storage and retrieval
	// In real implementation, key would be derived from user password
	
	// Verify password
	validPassword, err := hasher.VerifyPassword(userPassword, passwordHash)
	require.NoError(t, err)
	assert.True(t, validPassword)

	// Decrypt data
	decryptedData, err := encryptor.Decrypt(encryptedData, key)
	require.NoError(t, err)
	assert.Equal(t, sensitiveData, string(decryptedData))
}