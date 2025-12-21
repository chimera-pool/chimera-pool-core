package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// =============================================================================
// ISP-COMPLIANT USER SERVICE INTERFACES
// Each interface is small and focused on a single responsibility
// =============================================================================

// UserProfileReader reads user profile data (single responsibility)
type UserProfileReader interface {
	GetProfile(userID int64) (*UserProfileData, error)
}

// UserProfileWriter updates user profile data (single responsibility)
type UserProfileWriter interface {
	UpdateProfile(userID int64, data *UpdateProfileData) (*UserProfileData, error)
}

// UserPasswordChanger handles password changes (single responsibility)
type UserPasswordChanger interface {
	ChangePassword(userID int64, currentPassword, newPassword string) error
}

// UserMinerReader reads user's miner data (single responsibility)
type UserMinerReader interface {
	GetMiners(userID int64) ([]*MinerData, error)
}

// UserPayoutReader reads user's payout data (single responsibility)
type UserPayoutReader interface {
	GetPayouts(userID int64, limit, offset int) ([]*PayoutData, error)
}

// UserStatsReader reads user's mining statistics (single responsibility)
type UserStatsReader interface {
	GetHashrateHistory(userID int64, period string) ([]*HashratePoint, error)
	GetSharesHistory(userID int64, period string) ([]*SharesPoint, error)
	GetEarningsHistory(userID int64, period string) ([]*EarningsPoint, error)
}

// =============================================================================
// USER DATA MODELS
// =============================================================================

// UserProfileData represents user profile information
type UserProfileData struct {
	ID            int64     `json:"id"`
	Username      string    `json:"username"`
	Email         string    `json:"email"`
	PayoutAddress string    `json:"payout_address"`
	IsAdmin       bool      `json:"is_admin"`
	CreatedAt     time.Time `json:"created_at"`
}

// UpdateProfileData represents profile update request
type UpdateProfileData struct {
	Username      string `json:"username,omitempty"`
	PayoutAddress string `json:"payout_address,omitempty"`
}

// MinerData represents a user's miner
type MinerData struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	Hashrate   float64   `json:"hashrate"`
	LastSeen   time.Time `json:"last_seen"`
	IsActive   bool      `json:"is_active"`
	ShareCount int64     `json:"share_count"`
}

// PayoutData represents a payout record
type PayoutData struct {
	ID        int64     `json:"id"`
	Amount    float64   `json:"amount"`
	TxHash    string    `json:"tx_hash"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// HashratePoint represents a hashrate data point
type HashratePoint struct {
	Timestamp time.Time `json:"timestamp"`
	Hashrate  float64   `json:"hashrate"`
}

// SharesPoint represents a shares data point
type SharesPoint struct {
	Timestamp     time.Time `json:"timestamp"`
	ValidShares   int64     `json:"valid_shares"`
	InvalidShares int64     `json:"invalid_shares"`
}

// EarningsPoint represents an earnings data point
type EarningsPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Earnings  float64   `json:"earnings"`
}

// ChangePasswordRequest represents password change request
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

// =============================================================================
// USER HANDLERS - ISP-COMPLIANT IMPLEMENTATION
// =============================================================================

// UserHandlers provides HTTP handlers for user-related endpoints
type UserHandlers struct {
	profileReader   UserProfileReader
	profileWriter   UserProfileWriter
	passwordChanger UserPasswordChanger
	minerReader     UserMinerReader
	payoutReader    UserPayoutReader
	statsReader     UserStatsReader
}

// NewUserHandlers creates new user handlers with injected dependencies
func NewUserHandlers(
	profileReader UserProfileReader,
	profileWriter UserProfileWriter,
	passwordChanger UserPasswordChanger,
	minerReader UserMinerReader,
	payoutReader UserPayoutReader,
	statsReader UserStatsReader,
) *UserHandlers {
	return &UserHandlers{
		profileReader:   profileReader,
		profileWriter:   profileWriter,
		passwordChanger: passwordChanger,
		minerReader:     minerReader,
		payoutReader:    payoutReader,
		statsReader:     statsReader,
	}
}

// GetProfile returns user profile
// GET /api/v1/user/profile
func (h *UserHandlers) GetProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
			Code:    http.StatusUnauthorized,
		})
		return
	}

	userIDInt, ok := userID.(int64)
	if !ok {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Invalid user ID format",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	profile, err := h.profileReader.GetProfile(userIDInt)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "User not found",
			Code:    http.StatusNotFound,
		})
		return
	}

	c.JSON(http.StatusOK, profile)
}

// UpdateProfile updates user profile
// PUT /api/v1/user/profile
func (h *UserHandlers) UpdateProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
			Code:    http.StatusUnauthorized,
		})
		return
	}

	userIDInt, ok := userID.(int64)
	if !ok {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Invalid user ID format",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	var req UpdateProfileData
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid request data: " + err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	profile, err := h.profileWriter.UpdateProfile(userIDInt, &req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "username already taken" {
			statusCode = http.StatusConflict
		} else if err.Error() == "invalid username length" {
			statusCode = http.StatusBadRequest
		}
		c.JSON(statusCode, ErrorResponse{
			Error:   "update_failed",
			Message: err.Error(),
			Code:    statusCode,
		})
		return
	}

	c.JSON(http.StatusOK, profile)
}

// ChangePassword changes user password
// PUT /api/v1/user/password
func (h *UserHandlers) ChangePassword(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
			Code:    http.StatusUnauthorized,
		})
		return
	}

	userIDInt, ok := userID.(int64)
	if !ok {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Invalid user ID format",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid request data: " + err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	err := h.passwordChanger.ChangePassword(userIDInt, req.CurrentPassword, req.NewPassword)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "incorrect current password" {
			statusCode = http.StatusUnauthorized
		}
		c.JSON(statusCode, ErrorResponse{
			Error:   "password_change_failed",
			Message: err.Error(),
			Code:    statusCode,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}

// GetMiners returns user's miners
// GET /api/v1/user/miners
func (h *UserHandlers) GetMiners(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
			Code:    http.StatusUnauthorized,
		})
		return
	}

	userIDInt, ok := userID.(int64)
	if !ok {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Invalid user ID format",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	miners, err := h.minerReader.GetMiners(userIDInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get miners: " + err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"miners": miners,
		"total":  len(miners),
	})
}

// GetPayouts returns user's payouts
// GET /api/v1/user/payouts
func (h *UserHandlers) GetPayouts(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
			Code:    http.StatusUnauthorized,
		})
		return
	}

	userIDInt, ok := userID.(int64)
	if !ok {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Invalid user ID format",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	// Parse pagination parameters
	limit := 50
	offset := 0
	if l := c.Query("limit"); l != "" {
		if parsed, err := parseInt(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	if o := c.Query("offset"); o != "" {
		if parsed, err := parseInt(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	payouts, err := h.payoutReader.GetPayouts(userIDInt, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get payouts: " + err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"payouts": payouts,
		"limit":   limit,
		"offset":  offset,
	})
}

// GetHashrateHistory returns user's hashrate history
// GET /api/v1/user/stats/hashrate
func (h *UserHandlers) GetHashrateHistory(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
			Code:    http.StatusUnauthorized,
		})
		return
	}

	userIDInt, ok := userID.(int64)
	if !ok {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Invalid user ID format",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	period := c.DefaultQuery("period", "24h")
	validPeriods := map[string]bool{"1h": true, "6h": true, "24h": true, "7d": true, "30d": true}
	if !validPeriods[period] {
		period = "24h"
	}

	history, err := h.statsReader.GetHashrateHistory(userIDInt, period)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get hashrate history: " + err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"history": history,
		"period":  period,
	})
}

// GetSharesHistory returns user's shares history
// GET /api/v1/user/stats/shares
func (h *UserHandlers) GetSharesHistory(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
			Code:    http.StatusUnauthorized,
		})
		return
	}

	userIDInt, ok := userID.(int64)
	if !ok {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Invalid user ID format",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	period := c.DefaultQuery("period", "24h")
	validPeriods := map[string]bool{"1h": true, "6h": true, "24h": true, "7d": true, "30d": true}
	if !validPeriods[period] {
		period = "24h"
	}

	history, err := h.statsReader.GetSharesHistory(userIDInt, period)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get shares history: " + err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"history": history,
		"period":  period,
	})
}

// GetEarningsHistory returns user's earnings history
// GET /api/v1/user/stats/earnings
func (h *UserHandlers) GetEarningsHistory(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
			Code:    http.StatusUnauthorized,
		})
		return
	}

	userIDInt, ok := userID.(int64)
	if !ok {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Invalid user ID format",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	period := c.DefaultQuery("period", "24h")
	validPeriods := map[string]bool{"1h": true, "6h": true, "24h": true, "7d": true, "30d": true}
	if !validPeriods[period] {
		period = "24h"
	}

	history, err := h.statsReader.GetEarningsHistory(userIDInt, period)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get earnings history: " + err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"history": history,
		"period":  period,
	})
}

// Helper function to parse int
func parseInt(s string) (int, error) {
	var result int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, nil
		}
		result = result*10 + int(c-'0')
	}
	return result, nil
}
