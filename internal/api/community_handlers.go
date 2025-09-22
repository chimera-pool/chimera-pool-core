package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"chimera-pool-core/internal/community"
)

// CommunityHandlers handles community-related API endpoints
type CommunityHandlers struct {
	service *community.Service
}

// NewCommunityHandlers creates new community handlers
func NewCommunityHandlers(service *community.Service) *CommunityHandlers {
	return &CommunityHandlers{
		service: service,
	}
}

// CreateTeamRequest represents the request to create a team
type CreateTeamRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// CreateTeam creates a new mining team
func (h *CommunityHandlers) CreateTeam(c *gin.Context) {
	var req CreateTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
			"details": err.Error(),
		})
		return
	}
	
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}
	
	ownerID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Invalid user ID format",
		})
		return
	}
	
	team, err := h.service.CreateTeam(c.Request.Context(), req.Name, req.Description, ownerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create team",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{
		"team": team,
		"message": "Team created successfully",
	})
}

// JoinTeam allows a user to join a team
func (h *CommunityHandlers) JoinTeam(c *gin.Context) {
	teamIDStr := c.Param("teamId")
	teamID, err := uuid.Parse(teamIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid team ID format",
		})
		return
	}
	
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}
	
	memberID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Invalid user ID format",
		})
		return
	}
	
	err = h.service.JoinTeam(c.Request.Context(), teamID, memberID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to join team",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Successfully joined team",
	})
}

// LeaveTeam allows a user to leave a team
func (h *CommunityHandlers) LeaveTeam(c *gin.Context) {
	teamIDStr := c.Param("teamId")
	teamID, err := uuid.Parse(teamIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid team ID format",
		})
		return
	}
	
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}
	
	memberID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Invalid user ID format",
		})
		return
	}
	
	err = h.service.LeaveTeam(c.Request.Context(), teamID, memberID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to leave team",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Successfully left team",
	})
}

// CreateReferral creates a new referral code
func (h *CommunityHandlers) CreateReferral(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}
	
	referrerID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Invalid user ID format",
		})
		return
	}
	
	referral, err := h.service.CreateReferral(c.Request.Context(), referrerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create referral",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{
		"referral": referral,
		"message": "Referral code created successfully",
	})
}

// ProcessReferralRequest represents the request to process a referral
type ProcessReferralRequest struct {
	ReferralCode string `json:"referral_code" binding:"required"`
}

// ProcessReferral processes a referral when a new user joins
func (h *CommunityHandlers) ProcessReferral(c *gin.Context) {
	var req ProcessReferralRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
			"details": err.Error(),
		})
		return
	}
	
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}
	
	referredID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Invalid user ID format",
		})
		return
	}
	
	err := h.service.ProcessReferral(c.Request.Context(), req.ReferralCode, referredID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to process referral",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Referral processed successfully",
	})
}

// CreateCompetitionRequest represents the request to create a competition
type CreateCompetitionRequest struct {
	Name        string    `json:"name" binding:"required"`
	Description string    `json:"description"`
	StartTime   time.Time `json:"start_time" binding:"required"`
	EndTime     time.Time `json:"end_time" binding:"required"`
	PrizePool   float64   `json:"prize_pool" binding:"required"`
}

// CreateCompetition creates a new mining competition
func (h *CommunityHandlers) CreateCompetition(c *gin.Context) {
	var req CreateCompetitionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
			"details": err.Error(),
		})
		return
	}
	
	competition, err := h.service.CreateCompetition(c.Request.Context(), req.Name, req.Description, req.StartTime, req.EndTime, req.PrizePool)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create competition",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{
		"competition": competition,
		"message": "Competition created successfully",
	})
}

// JoinCompetition allows a user to join a competition
func (h *CommunityHandlers) JoinCompetition(c *gin.Context) {
	competitionIDStr := c.Param("competitionId")
	competitionID, err := uuid.Parse(competitionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid competition ID format",
		})
		return
	}
	
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}
	
	participantID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Invalid user ID format",
		})
		return
	}
	
	// Optional team ID from query parameter
	var teamID *uuid.UUID
	if teamIDStr := c.Query("team_id"); teamIDStr != "" {
		if parsedTeamID, err := uuid.Parse(teamIDStr); err == nil {
			teamID = &parsedTeamID
		}
	}
	
	err = h.service.JoinCompetition(c.Request.Context(), competitionID, participantID, teamID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to join competition",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Successfully joined competition",
	})
}

// RecordSocialShareRequest represents the request to record a social share
type RecordSocialShareRequest struct {
	Platform  string `json:"platform" binding:"required"`
	Content   string `json:"content" binding:"required"`
	Milestone string `json:"milestone" binding:"required"`
}

// RecordSocialShare records a social media share
func (h *CommunityHandlers) RecordSocialShare(c *gin.Context) {
	var req RecordSocialShareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
			"details": err.Error(),
		})
		return
	}
	
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}
	
	sharerID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Invalid user ID format",
		})
		return
	}
	
	share, err := h.service.RecordSocialShare(c.Request.Context(), sharerID, req.Platform, req.Content, req.Milestone)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to record social share",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{
		"share": share,
		"message": "Social share recorded successfully",
	})
}

// GetTeamStatistics retrieves team statistics
func (h *CommunityHandlers) GetTeamStatistics(c *gin.Context) {
	teamIDStr := c.Param("teamId")
	teamID, err := uuid.Parse(teamIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid team ID format",
		})
		return
	}
	
	period := c.DefaultQuery("period", "daily")
	daysStr := c.DefaultQuery("days", "30")
	days, err := strconv.Atoi(daysStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid days parameter",
		})
		return
	}
	
	stats, err := h.service.GetTeamStatistics(c.Request.Context(), teamID, period, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get team statistics",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"statistics": stats,
		"team_id": teamID,
		"period": period,
		"days": days,
	})
}