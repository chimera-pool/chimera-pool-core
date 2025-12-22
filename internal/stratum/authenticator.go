package stratum

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// =============================================================================
// MINER AUTHENTICATOR
// ISP-compliant interface for miner authentication and authorization
// Designed for high-performance with caching and minimal database lookups
// =============================================================================

var (
	ErrInvalidWorkerName    = errors.New("invalid worker name format")
	ErrUserNotFound         = errors.New("user not found")
	ErrMinerNotFound        = errors.New("miner not found")
	ErrAuthenticationFailed = errors.New("authentication failed")
	ErrUserDisabled         = errors.New("user account is disabled")
)

// AuthResult contains the result of a successful authentication
type AuthResult struct {
	UserID      int64
	MinerID     int64
	Username    string
	WorkerName  string
	IsNewMiner  bool
	Permissions MinerPermissions
}

// MinerPermissions defines what actions a miner is allowed to perform
type MinerPermissions struct {
	CanSubmitShares bool
	CanReceiveJobs  bool
	MaxDifficulty   float64
	MinDifficulty   float64
}

// DefaultMinerPermissions returns standard permissions for regular miners
func DefaultMinerPermissions() MinerPermissions {
	return MinerPermissions{
		CanSubmitShares: true,
		CanReceiveJobs:  true,
		MaxDifficulty:   1000000,
		MinDifficulty:   0.001,
	}
}

// =============================================================================
// ISP-COMPLIANT INTERFACES
// =============================================================================

// MinerAuthenticator handles miner authentication (ISP: single responsibility)
type MinerAuthenticator interface {
	// Authenticate validates a worker name and returns authentication result
	// Worker name format: "username.workername" or just "username"
	Authenticate(ctx context.Context, workerName string, password string) (*AuthResult, error)
}

// MinerLookup provides read-only access to miner information (ISP: query only)
type MinerLookup interface {
	// GetMinerByWorkerName retrieves miner info by worker name
	GetMinerByWorkerName(ctx context.Context, userID int64, workerName string) (*MinerInfo, error)

	// GetUserByUsername retrieves user info by username
	GetUserByUsername(ctx context.Context, username string) (*UserInfo, error)
}

// MinerRegistrar handles miner registration (ISP: command only)
type MinerRegistrar interface {
	// RegisterMiner creates a new miner for a user
	RegisterMiner(ctx context.Context, userID int64, workerName string, ipAddress string) (*MinerInfo, error)

	// UpdateMinerLastSeen updates the last seen timestamp
	UpdateMinerLastSeen(ctx context.Context, minerID int64) error
}

// UserInfo contains minimal user information for authentication
type UserInfo struct {
	ID           int64
	Username     string
	PasswordHash string
	IsActive     bool
	Role         string
}

// MinerInfo contains miner information
type MinerInfo struct {
	ID         int64
	UserID     int64
	WorkerName string
	IPAddress  string
	LastSeen   time.Time
	IsActive   bool
}

// =============================================================================
// CACHED AUTHENTICATOR IMPLEMENTATION
// High-performance authenticator with in-memory caching
// =============================================================================

// CachedAuthenticator wraps a database-backed authenticator with caching
type CachedAuthenticator struct {
	lookup    MinerLookup
	registrar MinerRegistrar

	// Cache for user lookups (username -> UserInfo)
	userCache    sync.Map
	userCacheTTL time.Duration

	// Cache for miner lookups (userID:workerName -> MinerInfo)
	minerCache    sync.Map
	minerCacheTTL time.Duration

	// Cache entry with expiration
	cacheEntries sync.Map
}

type cacheEntry struct {
	data      interface{}
	expiresAt time.Time
}

// CachedAuthenticatorConfig configures the cached authenticator
type CachedAuthenticatorConfig struct {
	UserCacheTTL  time.Duration
	MinerCacheTTL time.Duration
}

// DefaultCachedAuthenticatorConfig returns production defaults
func DefaultCachedAuthenticatorConfig() CachedAuthenticatorConfig {
	return CachedAuthenticatorConfig{
		UserCacheTTL:  5 * time.Minute,
		MinerCacheTTL: 1 * time.Minute,
	}
}

// NewCachedAuthenticator creates a new cached authenticator
func NewCachedAuthenticator(lookup MinerLookup, registrar MinerRegistrar, config CachedAuthenticatorConfig) *CachedAuthenticator {
	return &CachedAuthenticator{
		lookup:        lookup,
		registrar:     registrar,
		userCacheTTL:  config.UserCacheTTL,
		minerCacheTTL: config.MinerCacheTTL,
	}
}

// Authenticate implements MinerAuthenticator
func (ca *CachedAuthenticator) Authenticate(ctx context.Context, workerName string, password string) (*AuthResult, error) {
	// Parse worker name: "username.workername" or just "username"
	username, minerName, err := ParseWorkerName(workerName)
	if err != nil {
		return nil, err
	}

	// Look up user (with cache)
	user, err := ca.getUserCached(ctx, username)
	if err != nil {
		return nil, err
	}

	// Verify user is active
	if !user.IsActive {
		return nil, ErrUserDisabled
	}

	// Note: Password verification is typically not done in stratum
	// Workers authenticate by username only (password field is often ignored)
	// Real security comes from wallet address verification in payouts
	_ = password

	// Look up or create miner
	miner, isNew, err := ca.getOrCreateMiner(ctx, user.ID, minerName)
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		UserID:      user.ID,
		MinerID:     miner.ID,
		Username:    user.Username,
		WorkerName:  minerName,
		IsNewMiner:  isNew,
		Permissions: DefaultMinerPermissions(),
	}, nil
}

// getUserCached retrieves user from cache or database
func (ca *CachedAuthenticator) getUserCached(ctx context.Context, username string) (*UserInfo, error) {
	cacheKey := "user:" + username

	// Check cache
	if entry, ok := ca.cacheEntries.Load(cacheKey); ok {
		ce := entry.(*cacheEntry)
		if time.Now().Before(ce.expiresAt) {
			return ce.data.(*UserInfo), nil
		}
		ca.cacheEntries.Delete(cacheKey)
	}

	// Fetch from database
	user, err := ca.lookup.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	// Cache result
	ca.cacheEntries.Store(cacheKey, &cacheEntry{
		data:      user,
		expiresAt: time.Now().Add(ca.userCacheTTL),
	})

	return user, nil
}

// getOrCreateMiner retrieves or creates a miner
func (ca *CachedAuthenticator) getOrCreateMiner(ctx context.Context, userID int64, workerName string) (*MinerInfo, bool, error) {
	cacheKey := fmt.Sprintf("%d:%s", userID, workerName)

	// Check cache
	if entry, ok := ca.cacheEntries.Load(cacheKey); ok {
		ce := entry.(*cacheEntry)
		if time.Now().Before(ce.expiresAt) {
			return ce.data.(*MinerInfo), false, nil
		}
		ca.cacheEntries.Delete(cacheKey)
	}

	// Try to find existing miner
	miner, err := ca.lookup.GetMinerByWorkerName(ctx, userID, workerName)
	if err == nil {
		// Cache and return existing miner
		ca.cacheEntries.Store(cacheKey, &cacheEntry{
			data:      miner,
			expiresAt: time.Now().Add(ca.minerCacheTTL),
		})
		return miner, false, nil
	}

	// Miner not found - create new one
	if errors.Is(err, ErrMinerNotFound) {
		miner, err = ca.registrar.RegisterMiner(ctx, userID, workerName, "")
		if err != nil {
			return nil, false, err
		}

		// Cache new miner
		ca.cacheEntries.Store(cacheKey, &cacheEntry{
			data:      miner,
			expiresAt: time.Now().Add(ca.minerCacheTTL),
		})
		return miner, true, nil
	}

	return nil, false, err
}

// InvalidateUserCache removes a user from the cache
func (ca *CachedAuthenticator) InvalidateUserCache(username string) {
	ca.cacheEntries.Delete("user:" + username)
}

// InvalidateMinerCache removes a miner from the cache
func (ca *CachedAuthenticator) InvalidateMinerCache(userID int64, workerName string) {
	ca.cacheEntries.Delete(fmt.Sprintf("%d:%s", userID, workerName))
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// ParseWorkerName parses a worker name into username and miner name
// Formats supported:
//   - "username.workername" -> ("username", "workername", nil)
//   - "username" -> ("username", "default", nil)
//   - "" -> ("", "", ErrInvalidWorkerName)
func ParseWorkerName(workerName string) (username string, minerName string, err error) {
	workerName = strings.TrimSpace(workerName)
	if workerName == "" {
		return "", "", ErrInvalidWorkerName
	}

	// Split on first dot
	parts := strings.SplitN(workerName, ".", 2)
	username = parts[0]

	if len(parts) > 1 && parts[1] != "" {
		minerName = parts[1]
	} else {
		minerName = "default"
	}

	// Validate username
	if len(username) < 2 || len(username) > 50 {
		return "", "", ErrInvalidWorkerName
	}

	return username, minerName, nil
}

// =============================================================================
// MOCK AUTHENTICATOR (for testing)
// =============================================================================

// MockAuthenticator is a test implementation that accepts all workers
type MockAuthenticator struct {
	// AllowAll when true accepts any worker name
	AllowAll bool

	// Users maps usernames to UserInfo for controlled testing
	Users map[string]*UserInfo

	// Miners maps userID:workerName to MinerInfo
	Miners map[string]*MinerInfo

	mu sync.RWMutex
}

// NewMockAuthenticator creates a new mock authenticator
func NewMockAuthenticator() *MockAuthenticator {
	return &MockAuthenticator{
		AllowAll: true,
		Users:    make(map[string]*UserInfo),
		Miners:   make(map[string]*MinerInfo),
	}
}

// Authenticate implements MinerAuthenticator for testing
func (ma *MockAuthenticator) Authenticate(ctx context.Context, workerName string, password string) (*AuthResult, error) {
	username, minerName, err := ParseWorkerName(workerName)
	if err != nil {
		return nil, err
	}

	ma.mu.RLock()
	defer ma.mu.RUnlock()

	// Check if user exists in mock data
	user, exists := ma.Users[username]
	if !exists {
		if !ma.AllowAll {
			return nil, ErrUserNotFound
		}
		// Auto-create user for AllowAll mode
		user = &UserInfo{
			ID:       time.Now().UnixNano(), // Generate unique ID
			Username: username,
			IsActive: true,
		}
	}

	// Check for existing miner
	minerKey := fmt.Sprintf("%d:%s", user.ID, minerName)
	miner, minerExists := ma.Miners[minerKey]
	if !minerExists {
		miner = &MinerInfo{
			ID:         time.Now().UnixNano(), // Generate unique ID
			UserID:     user.ID,
			WorkerName: minerName,
			LastSeen:   time.Now(),
			IsActive:   true,
		}
	}

	return &AuthResult{
		UserID:      user.ID,
		MinerID:     miner.ID,
		Username:    username,
		WorkerName:  minerName,
		IsNewMiner:  !minerExists,
		Permissions: DefaultMinerPermissions(),
	}, nil
}

// AddUser adds a user to the mock
func (ma *MockAuthenticator) AddUser(user *UserInfo) {
	ma.mu.Lock()
	defer ma.mu.Unlock()
	ma.Users[user.Username] = user
}

// AddMiner adds a miner to the mock
func (ma *MockAuthenticator) AddMiner(miner *MinerInfo) {
	ma.mu.Lock()
	defer ma.mu.Unlock()
	ma.Miners[fmt.Sprintf("%d:%s", miner.UserID, miner.WorkerName)] = miner
}
