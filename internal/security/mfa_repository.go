package security

import (
	"errors"
	"sync"
)

// InMemoryMFARepository is an in-memory implementation of MFARepository for testing
type InMemoryMFARepository struct {
	mu          sync.RWMutex
	secrets     map[int64]string
	backupCodes map[int64][]string
	mfaEnabled  map[int64]bool
}

// NewInMemoryMFARepository creates a new in-memory MFA repository
func NewInMemoryMFARepository() *InMemoryMFARepository {
	return &InMemoryMFARepository{
		secrets:     make(map[int64]string),
		backupCodes: make(map[int64][]string),
		mfaEnabled:  make(map[int64]bool),
	}
}

// StoreTOTPSecret stores a TOTP secret for a user
func (r *InMemoryMFARepository) StoreTOTPSecret(userID int64, secret string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.secrets[userID] = secret
	return nil
}

// GetTOTPSecret retrieves a TOTP secret for a user
func (r *InMemoryMFARepository) GetTOTPSecret(userID int64) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	secret, exists := r.secrets[userID]
	if !exists {
		return "", errors.New("TOTP secret not found")
	}
	
	return secret, nil
}

// StoreBackupCodes stores backup codes for a user
func (r *InMemoryMFARepository) StoreBackupCodes(userID int64, codes []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Make a copy to avoid external modifications
	codesCopy := make([]string, len(codes))
	copy(codesCopy, codes)
	
	r.backupCodes[userID] = codesCopy
	return nil
}

// GetBackupCodes retrieves backup codes for a user
func (r *InMemoryMFARepository) GetBackupCodes(userID int64) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	codes, exists := r.backupCodes[userID]
	if !exists {
		return nil, errors.New("backup codes not found")
	}
	
	// Return a copy to avoid external modifications
	codesCopy := make([]string, len(codes))
	copy(codesCopy, codes)
	
	return codesCopy, nil
}

// UseBackupCode marks a backup code as used (removes it)
func (r *InMemoryMFARepository) UseBackupCode(userID int64, code string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	codes, exists := r.backupCodes[userID]
	if !exists {
		return errors.New("backup codes not found")
	}
	
	// Find and remove the code
	for i, storedCode := range codes {
		if storedCode == code {
			// Remove the code by creating a new slice without it
			newCodes := make([]string, 0, len(codes)-1)
			newCodes = append(newCodes, codes[:i]...)
			newCodes = append(newCodes, codes[i+1:]...)
			r.backupCodes[userID] = newCodes
			return nil
		}
	}
	
	return errors.New("backup code not found")
}

// EnableMFA enables MFA for a user
func (r *InMemoryMFARepository) EnableMFA(userID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.mfaEnabled[userID] = true
	return nil
}

// DisableMFA disables MFA for a user
func (r *InMemoryMFARepository) DisableMFA(userID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Remove all MFA data
	delete(r.secrets, userID)
	delete(r.backupCodes, userID)
	delete(r.mfaEnabled, userID)
	
	return nil
}

// IsMFAEnabled checks if MFA is enabled for a user
func (r *InMemoryMFARepository) IsMFAEnabled(userID int64) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	enabled, exists := r.mfaEnabled[userID]
	return exists && enabled, nil
}