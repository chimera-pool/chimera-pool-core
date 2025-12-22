package api

import (
	"context"
	"time"
)

// =============================================================================
// ISP-COMPLIANT API SERVICE INTERFACES
// Granular interfaces for maximum flexibility and testability
// =============================================================================

// -----------------------------------------------------------------------------
// Authentication Interfaces
// -----------------------------------------------------------------------------

// JWTValidator validates JWT tokens
type JWTValidator interface {
	ValidateJWT(token string) (*JWTClaims, error)
}

// JWTGenerator generates JWT tokens
type JWTGenerator interface {
	GenerateJWT(userID int64, username, email string) (string, error)
}

// JWTRefresher refreshes JWT tokens
type JWTRefresher interface {
	RefreshJWT(token string) (string, error)
}

// TokenRevoker revokes tokens
type TokenRevoker interface {
	RevokeToken(token string) error
	IsTokenRevoked(token string) bool
}

// FullAuthService combines all authentication operations
type FullAuthService interface {
	JWTValidator
	JWTGenerator
	JWTRefresher
	TokenRevoker
}

// -----------------------------------------------------------------------------
// Pool Statistics Interfaces (Granular)
// -----------------------------------------------------------------------------

// PoolHashrateProvider provides hashrate data
type PoolHashrateProvider interface {
	GetTotalHashrate(ctx context.Context) (float64, error)
	GetCurrentHashrate(ctx context.Context) (float64, error)
	GetAverageHashrate(ctx context.Context, period time.Duration) (float64, error)
}

// PoolMinerCounter counts pool miners
type PoolMinerCounter interface {
	GetConnectedMinersCount(ctx context.Context) (int64, error)
	GetActiveMinersCount(ctx context.Context) (int64, error)
}

// PoolSharesProvider provides share statistics
type PoolSharesProvider interface {
	GetTotalShares(ctx context.Context) (int64, error)
	GetValidShares(ctx context.Context) (int64, error)
	GetSharesPerSecond(ctx context.Context) (float64, error)
}

// PoolBlocksProvider provides block statistics
type PoolBlocksProvider interface {
	GetBlocksFound(ctx context.Context) (int64, error)
	GetLastBlockTime(ctx context.Context) (time.Time, error)
	GetBlocksInPeriod(ctx context.Context, period time.Duration) (int64, error)
}

// NetworkInfoProvider provides network information
type NetworkInfoProvider interface {
	GetNetworkHashrate(ctx context.Context) (float64, error)
	GetNetworkDifficulty(ctx context.Context) (float64, error)
}

// PoolStatsProvider combines all pool statistics (for backward compatibility)
type PoolStatsProvider interface {
	GetPoolStats() (*PoolStats, error)
	GetRealTimeStats() (*RealTimeStats, error)
	GetBlockMetrics() (*BlockMetrics, error)
}

// -----------------------------------------------------------------------------
// User Statistics Interfaces (Granular)
// -----------------------------------------------------------------------------

// UserStatsProvider provides user-specific statistics
type UserStatsProvider interface {
	GetUserStats(userID int64) (*UserStats, error)
}

// MinerStatsProvider provides miner-specific statistics
type MinerStatsProvider interface {
	GetMinerStats(minerID int64) (*MinerStats, error)
}

// UserMinerLister lists user's miners
type UserMinerLister interface {
	GetUserMiners(userID int64) ([]*MinerInfo, error)
}

// -----------------------------------------------------------------------------
// User Profile Interfaces (Granular)
// -----------------------------------------------------------------------------

// ProfileReader reads user profiles
type ProfileReader interface {
	GetUserProfile(userID int64) (*UserProfile, error)
}

// ProfileWriter writes user profiles
type ProfileWriter interface {
	UpdateUserProfile(userID int64, profile *UpdateUserProfileRequest) (*UserProfile, error)
}

// ProfileManager combines read and write operations
type ProfileManager interface {
	ProfileReader
	ProfileWriter
}

// -----------------------------------------------------------------------------
// MFA Interfaces (Granular)
// -----------------------------------------------------------------------------

// MFASetupProvider provides MFA setup functionality
type MFASetupProvider interface {
	SetupMFA(userID int64) (*MFASetupResponse, error)
}

// MFAVerifier verifies MFA codes
type MFAVerifier interface {
	VerifyMFA(userID int64, code string) (bool, error)
}

// MFADisabler disables MFA
type MFADisabler interface {
	DisableMFA(userID int64) error
}

// MFAManager combines all MFA operations
type MFAManager interface {
	MFASetupProvider
	MFAVerifier
	MFADisabler
}

// -----------------------------------------------------------------------------
// Payout Interfaces
// -----------------------------------------------------------------------------

// PayoutHistoryProvider provides payout history
type PayoutHistoryProvider interface {
	GetUserPayouts(ctx context.Context, userID int64, limit, offset int) ([]*PayoutInfo, error)
	GetPendingPayout(ctx context.Context, userID int64) (*PayoutInfo, error)
}

// PayoutThresholdManager manages payout thresholds
type PayoutThresholdManager interface {
	GetPayoutThreshold(ctx context.Context, userID int64) (float64, error)
	SetPayoutThreshold(ctx context.Context, userID int64, threshold float64) error
}

// PayoutInfo represents payout information
type PayoutInfo struct {
	ID          int64      `json:"id"`
	UserID      int64      `json:"user_id"`
	Amount      float64    `json:"amount"`
	Address     string     `json:"address"`
	TxHash      string     `json:"tx_hash,omitempty"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	ProcessedAt *time.Time `json:"processed_at,omitempty"`
}

// -----------------------------------------------------------------------------
// Worker Management Interfaces
// -----------------------------------------------------------------------------

// WorkerCreator creates new workers
type WorkerCreator interface {
	CreateWorker(ctx context.Context, userID int64, name string) (*MinerInfo, error)
}

// WorkerUpdater updates workers
type WorkerUpdater interface {
	UpdateWorker(ctx context.Context, workerID int64, name string) (*MinerInfo, error)
	SetWorkerActive(ctx context.Context, workerID int64, active bool) error
}

// WorkerDeleter deletes workers
type WorkerDeleter interface {
	DeleteWorker(ctx context.Context, workerID int64) error
}

// WorkerManager combines all worker operations
type WorkerManager interface {
	UserMinerLister
	WorkerCreator
	WorkerUpdater
	WorkerDeleter
}

// -----------------------------------------------------------------------------
// Notification Interfaces
// -----------------------------------------------------------------------------

// NotificationSender sends notifications
type NotificationSender interface {
	SendNotification(ctx context.Context, userID int64, title, message string) error
}

// NotificationReader reads notifications
type NotificationReader interface {
	GetNotifications(ctx context.Context, userID int64, limit int) ([]*Notification, error)
	MarkAsRead(ctx context.Context, notificationID int64) error
}

// Notification represents a user notification
type Notification struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Read      bool      `json:"read"`
	CreatedAt time.Time `json:"created_at"`
}

// -----------------------------------------------------------------------------
// API Response Helpers
// -----------------------------------------------------------------------------

// ResponseBuilder builds API responses
type ResponseBuilder interface {
	Success(data interface{}) interface{}
	Error(code int, message string) interface{}
	Paginated(data interface{}, total, page, pageSize int) interface{}
}

// -----------------------------------------------------------------------------
// Request Validation Interfaces
// -----------------------------------------------------------------------------

// RequestValidator validates API requests
type RequestValidator interface {
	ValidateRequest(req interface{}) error
}

// InputSanitizer sanitizes user input
type InputSanitizer interface {
	Sanitize(input string) string
	SanitizeHTML(html string) string
}

// -----------------------------------------------------------------------------
// Rate Limiting for API
// -----------------------------------------------------------------------------

// APIRateLimiter rate limits API requests
type APIRateLimiter interface {
	AllowRequest(ctx context.Context, clientID, endpoint string) (bool, error)
	GetRemainingRequests(ctx context.Context, clientID, endpoint string) (int, error)
}

// -----------------------------------------------------------------------------
// Caching Interfaces
// -----------------------------------------------------------------------------

// ResponseCache caches API responses
type ResponseCache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}, ttl time.Duration)
	Delete(key string)
	Clear()
}

// CacheKeyGenerator generates cache keys
type CacheKeyGenerator interface {
	GenerateKey(endpoint string, params map[string]string) string
}

// -----------------------------------------------------------------------------
// Metrics Interfaces
// -----------------------------------------------------------------------------

// APIMetricsRecorder records API metrics
type APIMetricsRecorder interface {
	RecordRequest(endpoint, method string, statusCode int, duration time.Duration)
	RecordError(endpoint, method string, errorType string)
}

// APIMetricsReader reads API metrics
type APIMetricsReader interface {
	GetRequestCount(endpoint string) int64
	GetAverageLatency(endpoint string) time.Duration
	GetErrorRate(endpoint string) float64
}

// APIMetrics combines recording and reading
type APIMetrics interface {
	APIMetricsRecorder
	APIMetricsReader
}
