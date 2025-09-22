package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// APIHandlers provides HTTP handlers for the REST API
type APIHandlers struct {
	authService      AuthService
	poolStatsService PoolStatsService
	userService      UserService
}

// NewAPIHandlers creates new API handlers
func NewAPIHandlers(authService AuthService, poolStatsService PoolStatsService, userService UserService) *APIHandlers {
	return &APIHandlers{
		authService:      authService,
		poolStatsService: poolStatsService,
		userService:      userService,
	}
}

// GetPoolStats returns pool statistics
func (h *APIHandlers) GetPoolStats(c *gin.Context) {
	stats, err := h.poolStatsService.GetPoolStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get pool statistics: " + err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	// Calculate efficiency and invalid shares
	efficiency := float64(0)
	invalidShares := int64(0)
	if stats.TotalShares > 0 {
		efficiency = float64(stats.ValidShares) / float64(stats.TotalShares) * 100
		invalidShares = stats.TotalShares - stats.ValidShares
	}

	response := PoolStatsResponse{
		TotalHashrate:     stats.TotalHashrate,
		ConnectedMiners:   stats.ConnectedMiners,
		TotalShares:       stats.TotalShares,
		ValidShares:       stats.ValidShares,
		InvalidShares:     invalidShares,
		BlocksFound:       stats.BlocksFound,
		LastBlockTime:     stats.LastBlockTime,
		NetworkHashrate:   stats.NetworkHashrate,
		NetworkDifficulty: stats.NetworkDifficulty,
		PoolFee:           stats.PoolFee,
		Efficiency:        efficiency,
	}

	c.JSON(http.StatusOK, response)
}

// GetUserProfile returns the current user's profile
func (h *APIHandlers) GetUserProfile(c *gin.Context) {
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

	profile, err := h.userService.GetUserProfile(userIDInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get user profile: " + err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	response := UserProfileResponse{
		ID:       profile.ID,
		Username: profile.Username,
		Email:    profile.Email,
		JoinedAt: profile.JoinedAt,
		IsActive: profile.IsActive,
	}

	c.JSON(http.StatusOK, response)
}

// UpdateUserProfile updates the current user's profile
func (h *APIHandlers) UpdateUserProfile(c *gin.Context) {
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

	var req UpdateUserProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid request data: " + err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	profile, err := h.userService.UpdateUserProfile(userIDInt, &req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		} else if strings.Contains(err.Error(), "validation") {
			statusCode = http.StatusBadRequest
		}

		c.JSON(statusCode, ErrorResponse{
			Error:   "update_failed",
			Message: "Failed to update user profile: " + err.Error(),
			Code:    statusCode,
		})
		return
	}

	response := UserProfileResponse{
		ID:       profile.ID,
		Username: profile.Username,
		Email:    profile.Email,
		JoinedAt: profile.JoinedAt,
		IsActive: profile.IsActive,
	}

	c.JSON(http.StatusOK, response)
}

// GetUserStats returns the current user's mining statistics
func (h *APIHandlers) GetUserStats(c *gin.Context) {
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

	stats, err := h.poolStatsService.GetUserStats(userIDInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get user statistics: " + err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	// Calculate efficiency
	efficiency := float64(0)
	if stats.TotalShares > 0 {
		efficiency = float64(stats.ValidShares) / float64(stats.TotalShares) * 100
	}

	response := UserStatsResponse{
		UserID:        stats.UserID,
		TotalShares:   stats.TotalShares,
		ValidShares:   stats.ValidShares,
		InvalidShares: stats.InvalidShares,
		TotalHashrate: stats.TotalHashrate,
		LastShare:     stats.LastShare,
		Earnings:      stats.Earnings,
		Efficiency:    efficiency,
	}

	c.JSON(http.StatusOK, response)
}

// GetUserMiners returns the current user's miners
func (h *APIHandlers) GetUserMiners(c *gin.Context) {
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

	miners, err := h.userService.GetUserMiners(userIDInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get user miners: " + err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	response := UserMinersResponse{
		Miners: miners,
		Total:  len(miners),
	}

	c.JSON(http.StatusOK, response)
}

// GetMinerStats returns statistics for a specific miner
func (h *APIHandlers) GetMinerStats(c *gin.Context) {
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

	minerIDStr := c.Param("miner_id")
	minerID, err := strconv.ParseInt(minerIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid miner ID: " + err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	stats, err := h.poolStatsService.GetMinerStats(minerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get miner statistics: " + err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	// Verify the miner belongs to the authenticated user
	if stats.UserID != userIDInt {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "Access denied to miner statistics",
			Code:    http.StatusForbidden,
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// AuthMiddleware provides JWT authentication middleware
func (h *APIHandlers) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "missing_token",
				Message: "Authorization header is required",
				Code:    http.StatusUnauthorized,
			})
			c.Abort()
			return
		}

		// Remove "Bearer " prefix if present
		if strings.HasPrefix(tokenString, "Bearer ") {
			tokenString = strings.TrimPrefix(tokenString, "Bearer ")
		}

		claims, err := h.authService.ValidateJWT(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "invalid_token",
				Message: err.Error(),
				Code:    http.StatusUnauthorized,
			})
			c.Abort()
			return
		}

		// Set user information in context
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("user_email", claims.Email)

		c.Next()
	}
}

// GetRealTimeStats returns real-time pool statistics (Requirement 7.1)
func (h *APIHandlers) GetRealTimeStats(c *gin.Context) {
	stats, err := h.poolStatsService.GetRealTimeStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get real-time statistics: " + err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	response := RealTimeStatsResponse{
		CurrentHashrate:   stats.CurrentHashrate,
		AverageHashrate:   stats.AverageHashrate,
		ActiveMiners:      stats.ActiveMiners,
		SharesPerSecond:   stats.SharesPerSecond,
		LastBlockFound:    stats.LastBlockFound,
		NetworkDifficulty: stats.NetworkDifficulty,
		PoolEfficiency:    stats.PoolEfficiency,
		Timestamp:         time.Now().UTC(),
	}

	c.JSON(http.StatusOK, response)
}

// GetBlockMetrics returns block discovery metrics (Requirement 7.2)
func (h *APIHandlers) GetBlockMetrics(c *gin.Context) {
	metrics, err := h.poolStatsService.GetBlockMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get block metrics: " + err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	response := BlockMetricsResponse{
		TotalBlocks:       metrics.TotalBlocks,
		BlocksLast24h:     metrics.BlocksLast24h,
		BlocksLast7d:      metrics.BlocksLast7d,
		AverageBlockTime:  int64(metrics.AverageBlockTime.Seconds()),
		LastBlockReward:   metrics.LastBlockReward,
		TotalRewards:      metrics.TotalRewards,
		OrphanBlocks:      metrics.OrphanBlocks,
		OrphanRate:        metrics.OrphanRate,
		Timestamp:         time.Now().UTC(),
	}

	c.JSON(http.StatusOK, response)
}

// SetupMFA initiates MFA setup for a user (Requirement 21.1)
func (h *APIHandlers) SetupMFA(c *gin.Context) {
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

	mfaSetup, err := h.userService.SetupMFA(userIDInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to setup MFA: " + err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, mfaSetup)
}

// VerifyMFA verifies MFA code and enables MFA for user (Requirement 21.1)
func (h *APIHandlers) VerifyMFA(c *gin.Context) {
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

	var req VerifyMFARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid request data: " + err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	verified, err := h.userService.VerifyMFA(userIDInt, req.Code)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_mfa_code",
			Message: "Invalid MFA code: " + err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	if !verified {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_mfa_code",
			Message: "Invalid MFA code provided",
			Code:    http.StatusBadRequest,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"verified": true,
		"message":  "MFA enabled successfully",
	})
}

// SetupAPIRoutes sets up all API routes
func SetupAPIRoutes(router *gin.Engine, handlers *APIHandlers) {
	// Health check endpoint (public)
	router.GET("/health", HealthCheck)

	// API v1 routes
	api := router.Group("/api/v1")
	{
		// Public pool statistics
		api.GET("/pool/stats", handlers.GetPoolStats)
		api.GET("/pool/realtime", handlers.GetRealTimeStats)
		api.GET("/pool/blocks", handlers.GetBlockMetrics)

		// Protected user routes
		user := api.Group("/user")
		user.Use(handlers.AuthMiddleware())
		{
			user.GET("/profile", handlers.GetUserProfile)
			user.PUT("/profile", handlers.UpdateUserProfile)
			user.GET("/stats", handlers.GetUserStats)
			user.GET("/miners", handlers.GetUserMiners)
			user.GET("/miners/:miner_id/stats", handlers.GetMinerStats)
			
			// MFA endpoints
			mfa := user.Group("/mfa")
			{
				mfa.POST("/setup", handlers.SetupMFA)
				mfa.POST("/verify", handlers.VerifyMFA)
			}
		}
	}
}

// HealthCheck provides a health check endpoint
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "chimera-pool-api",
		"timestamp": time.Now().UTC(),
		"version":   "1.0.0",
	})
}