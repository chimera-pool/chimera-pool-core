package api

import (
	"time"
)

// JWTClaims represents JWT token claims
type JWTClaims struct {
	UserID    int64     `json:"user_id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	IssuedAt  time.Time `json:"iat"`
	ExpiresAt time.Time `json:"exp"`
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// PoolStats represents pool statistics
type PoolStats struct {
	TotalHashrate     float64   `json:"total_hashrate"`
	ConnectedMiners   int64     `json:"connected_miners"`
	TotalShares       int64     `json:"total_shares"`
	ValidShares       int64     `json:"valid_shares"`
	BlocksFound       int64     `json:"blocks_found"`
	LastBlockTime     time.Time `json:"last_block_time"`
	NetworkHashrate   float64   `json:"network_hashrate"`
	NetworkDifficulty float64   `json:"network_difficulty"`
	PoolFee           float64   `json:"pool_fee"`
}

// PoolStatsResponse represents the pool stats API response
type PoolStatsResponse struct {
	TotalHashrate     float64   `json:"total_hashrate"`
	ConnectedMiners   int64     `json:"connected_miners"`
	TotalShares       int64     `json:"total_shares"`
	ValidShares       int64     `json:"valid_shares"`
	InvalidShares     int64     `json:"invalid_shares"`
	BlocksFound       int64     `json:"blocks_found"`
	LastBlockTime     time.Time `json:"last_block_time"`
	NetworkHashrate   float64   `json:"network_hashrate"`
	NetworkDifficulty float64   `json:"network_difficulty"`
	PoolFee           float64   `json:"pool_fee"`
	Efficiency        float64   `json:"efficiency"`
}

// UserProfile represents a user's profile information
type UserProfile struct {
	ID       int64     `json:"id"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
	JoinedAt time.Time `json:"joined_at"`
	IsActive bool      `json:"is_active"`
}

// UserProfileResponse represents the user profile API response
type UserProfileResponse struct {
	ID       int64     `json:"id"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
	JoinedAt time.Time `json:"joined_at"`
	IsActive bool      `json:"is_active"`
}

// UpdateUserProfileRequest represents a user profile update request
type UpdateUserProfileRequest struct {
	Email string `json:"email,omitempty" binding:"omitempty,email"`
}

// UserStats represents user mining statistics
type UserStats struct {
	UserID        int64     `json:"user_id"`
	TotalShares   int64     `json:"total_shares"`
	ValidShares   int64     `json:"valid_shares"`
	InvalidShares int64     `json:"invalid_shares"`
	TotalHashrate float64   `json:"total_hashrate"`
	LastShare     time.Time `json:"last_share"`
	Earnings      float64   `json:"earnings"`
}

// UserStatsResponse represents the user stats API response
type UserStatsResponse struct {
	UserID        int64     `json:"user_id"`
	TotalShares   int64     `json:"total_shares"`
	ValidShares   int64     `json:"valid_shares"`
	InvalidShares int64     `json:"invalid_shares"`
	TotalHashrate float64   `json:"total_hashrate"`
	LastShare     time.Time `json:"last_share"`
	Earnings      float64   `json:"earnings"`
	Efficiency    float64   `json:"efficiency"`
}

// MinerStats represents miner-specific statistics
type MinerStats struct {
	MinerID       int64     `json:"miner_id"`
	UserID        int64     `json:"user_id"`
	TotalShares   int64     `json:"total_shares"`
	ValidShares   int64     `json:"valid_shares"`
	InvalidShares int64     `json:"invalid_shares"`
	TotalHashrate float64   `json:"total_hashrate"`
	LastShare     time.Time `json:"last_share"`
}

// MinerInfo represents basic miner information
type MinerInfo struct {
	ID       int64     `json:"id"`
	Name     string    `json:"name"`
	Hashrate float64   `json:"hashrate"`
	LastSeen time.Time `json:"last_seen"`
	IsActive bool      `json:"is_active"`
}

// UserMinersResponse represents the user miners API response
type UserMinersResponse struct {
	Miners []*MinerInfo `json:"miners"`
	Total  int          `json:"total"`
}

// RealTimeStats represents real-time pool statistics
type RealTimeStats struct {
	CurrentHashrate   float64   `json:"current_hashrate"`
	AverageHashrate   float64   `json:"average_hashrate"`
	ActiveMiners      int64     `json:"active_miners"`
	SharesPerSecond   float64   `json:"shares_per_second"`
	LastBlockFound    time.Time `json:"last_block_found"`
	NetworkDifficulty float64   `json:"network_difficulty"`
	PoolEfficiency    float64   `json:"pool_efficiency"`
}

// RealTimeStatsResponse represents the real-time stats API response
type RealTimeStatsResponse struct {
	CurrentHashrate   float64   `json:"current_hashrate"`
	AverageHashrate   float64   `json:"average_hashrate"`
	ActiveMiners      int64     `json:"active_miners"`
	SharesPerSecond   float64   `json:"shares_per_second"`
	LastBlockFound    time.Time `json:"last_block_found"`
	NetworkDifficulty float64   `json:"network_difficulty"`
	PoolEfficiency    float64   `json:"pool_efficiency"`
	Timestamp         time.Time `json:"timestamp"`
}

// BlockMetrics represents block discovery metrics
type BlockMetrics struct {
	TotalBlocks      int64         `json:"total_blocks"`
	BlocksLast24h    int64         `json:"blocks_last_24h"`
	BlocksLast7d     int64         `json:"blocks_last_7d"`
	AverageBlockTime time.Duration `json:"average_block_time"`
	LastBlockReward  float64       `json:"last_block_reward"`
	TotalRewards     float64       `json:"total_rewards"`
	OrphanBlocks     int64         `json:"orphan_blocks"`
	OrphanRate       float64       `json:"orphan_rate"`
}

// BlockMetricsResponse represents the block metrics API response
type BlockMetricsResponse struct {
	TotalBlocks      int64     `json:"total_blocks"`
	BlocksLast24h    int64     `json:"blocks_last_24h"`
	BlocksLast7d     int64     `json:"blocks_last_7d"`
	AverageBlockTime int64     `json:"average_block_time_seconds"`
	LastBlockReward  float64   `json:"last_block_reward"`
	TotalRewards     float64   `json:"total_rewards"`
	OrphanBlocks     int64     `json:"orphan_blocks"`
	OrphanRate       float64   `json:"orphan_rate"`
	Timestamp        time.Time `json:"timestamp"`
}

// MFASetupResponse represents MFA setup response
type MFASetupResponse struct {
	Secret      string   `json:"secret"`
	QRCodeURL   string   `json:"qr_code_url"`
	BackupCodes []string `json:"backup_codes"`
}

// VerifyMFARequest represents MFA verification request
type VerifyMFARequest struct {
	Code string `json:"code" binding:"required,len=6"`
}

// AuthService interface for authentication operations
type AuthService interface {
	ValidateJWT(token string) (*JWTClaims, error)
}

// PoolStatsService interface for pool statistics operations
type PoolStatsService interface {
	GetPoolStats() (*PoolStats, error)
	GetMinerStats(userID int64) (*MinerStats, error)
	GetUserStats(userID int64) (*UserStats, error)
	GetRealTimeStats() (*RealTimeStats, error)
	GetBlockMetrics() (*BlockMetrics, error)
}

// UserService interface for user operations
type UserService interface {
	GetUserProfile(userID int64) (*UserProfile, error)
	UpdateUserProfile(userID int64, profile *UpdateUserProfileRequest) (*UserProfile, error)
	GetUserMiners(userID int64) ([]*MinerInfo, error)
	SetupMFA(userID int64) (*MFASetupResponse, error)
	VerifyMFA(userID int64, code string) (bool, error)
}
