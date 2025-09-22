package community

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Repository defines the interface for community data operations
type Repository interface {
	// Team operations
	CreateTeam(ctx context.Context, team *Team) error
	GetTeam(ctx context.Context, id uuid.UUID) (*Team, error)
	UpdateTeam(ctx context.Context, team *Team) error
	DeleteTeam(ctx context.Context, id uuid.UUID) error
	
	// Team member operations
	AddTeamMember(ctx context.Context, member *TeamMember) error
	RemoveTeamMember(ctx context.Context, teamID, userID uuid.UUID) error
	GetTeamMembers(ctx context.Context, teamID uuid.UUID) ([]*TeamMember, error)
	
	// Referral operations
	CreateReferral(ctx context.Context, referral *Referral) error
	GetReferral(ctx context.Context, code string) (*Referral, error)
	UpdateReferral(ctx context.Context, referral *Referral) error
	
	// Competition operations
	CreateCompetition(ctx context.Context, competition *Competition) error
	GetActiveCompetitions(ctx context.Context) ([]*Competition, error)
	JoinCompetition(ctx context.Context, participant *CompetitionParticipant) error
	
	// Social sharing operations
	RecordSocialShare(ctx context.Context, share *SocialShare) error
	
	// Statistics operations
	GetTeamStatistics(ctx context.Context, teamID uuid.UUID, period string, days int) ([]*TeamStatistics, error)
}

// Service provides community features functionality
type Service struct {
	repo Repository
}

// NewService creates a new community service
func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
	}
}

// CreateTeam creates a new mining team
func (s *Service) CreateTeam(ctx context.Context, name, description string, ownerID uuid.UUID) (*Team, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("team name cannot be empty")
	}
	
	if ownerID == uuid.Nil {
		return nil, fmt.Errorf("owner ID cannot be empty")
	}
	
	team := &Team{
		ID:          uuid.New(),
		Name:        strings.TrimSpace(name),
		Description: strings.TrimSpace(description),
		OwnerID:     ownerID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		IsActive:    true,
	}
	
	if err := s.repo.CreateTeam(ctx, team); err != nil {
		return nil, fmt.Errorf("failed to create team: %w", err)
	}
	
	// Add owner as team member
	member := &TeamMember{
		ID:       uuid.New(),
		TeamID:   team.ID,
		UserID:   ownerID,
		Role:     "owner",
		JoinedAt: time.Now(),
		IsActive: true,
	}
	
	if err := s.repo.AddTeamMember(ctx, member); err != nil {
		return nil, fmt.Errorf("failed to add owner as team member: %w", err)
	}
	
	return team, nil
}

// JoinTeam adds a user to a team
func (s *Service) JoinTeam(ctx context.Context, teamID, userID uuid.UUID) error {
	if teamID == uuid.Nil {
		return fmt.Errorf("team ID cannot be empty")
	}
	
	if userID == uuid.Nil {
		return fmt.Errorf("user ID cannot be empty")
	}
	
	// Check if team exists and is active
	team, err := s.repo.GetTeam(ctx, teamID)
	if err != nil {
		return fmt.Errorf("failed to get team: %w", err)
	}
	
	if !team.IsActive {
		return fmt.Errorf("team is not active")
	}
	
	member := &TeamMember{
		ID:       uuid.New(),
		TeamID:   teamID,
		UserID:   userID,
		Role:     "member",
		JoinedAt: time.Now(),
		IsActive: true,
	}
	
	if err := s.repo.AddTeamMember(ctx, member); err != nil {
		return fmt.Errorf("failed to add team member: %w", err)
	}
	
	return nil
}

// LeaveTeam removes a user from a team
func (s *Service) LeaveTeam(ctx context.Context, teamID, userID uuid.UUID) error {
	if teamID == uuid.Nil {
		return fmt.Errorf("team ID cannot be empty")
	}
	
	if userID == uuid.Nil {
		return fmt.Errorf("user ID cannot be empty")
	}
	
	if err := s.repo.RemoveTeamMember(ctx, teamID, userID); err != nil {
		return fmt.Errorf("failed to remove team member: %w", err)
	}
	
	return nil
}

// CreateReferral creates a new referral code for a user
func (s *Service) CreateReferral(ctx context.Context, referrerID uuid.UUID) (*Referral, error) {
	if referrerID == uuid.Nil {
		return nil, fmt.Errorf("referrer ID cannot be empty")
	}
	
	// Generate unique referral code
	code, err := s.generateReferralCode()
	if err != nil {
		return nil, fmt.Errorf("failed to generate referral code: %w", err)
	}
	
	referral := &Referral{
		ID:         uuid.New(),
		ReferrerID: referrerID,
		Code:       code,
		CreatedAt:  time.Now(),
		Status:     "pending",
	}
	
	if err := s.repo.CreateReferral(ctx, referral); err != nil {
		return nil, fmt.Errorf("failed to create referral: %w", err)
	}
	
	return referral, nil
}

// ProcessReferral processes a referral when a new user joins
func (s *Service) ProcessReferral(ctx context.Context, referralCode string, referredID uuid.UUID) error {
	if strings.TrimSpace(referralCode) == "" {
		return fmt.Errorf("referral code cannot be empty")
	}
	
	if referredID == uuid.Nil {
		return fmt.Errorf("referred user ID cannot be empty")
	}
	
	referral, err := s.repo.GetReferral(ctx, referralCode)
	if err != nil {
		return fmt.Errorf("failed to get referral: %w", err)
	}
	
	if referral.Status != "pending" {
		return fmt.Errorf("referral is not pending")
	}
	
	// Check if referral is not expired (30 days)
	if time.Since(referral.CreatedAt) > 30*24*time.Hour {
		return fmt.Errorf("referral has expired")
	}
	
	// Update referral
	now := time.Now()
	referral.ReferredID = referredID
	referral.CompletedAt = &now
	referral.Status = "completed"
	referral.BonusAmount = 10.0 // Default bonus amount
	
	if err := s.repo.UpdateReferral(ctx, referral); err != nil {
		return fmt.Errorf("failed to update referral: %w", err)
	}
	
	return nil
}

// CreateCompetition creates a new mining competition
func (s *Service) CreateCompetition(ctx context.Context, name, description string, startTime, endTime time.Time, prizePool float64) (*Competition, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("competition name cannot be empty")
	}
	
	if endTime.Before(startTime) {
		return nil, fmt.Errorf("end time must be after start time")
	}
	
	if prizePool < 0 {
		return nil, fmt.Errorf("prize pool cannot be negative")
	}
	
	status := "upcoming"
	if startTime.Before(time.Now()) {
		status = "active"
	}
	
	competition := &Competition{
		ID:          uuid.New(),
		Name:        strings.TrimSpace(name),
		Description: strings.TrimSpace(description),
		StartTime:   startTime,
		EndTime:     endTime,
		PrizePool:   prizePool,
		Status:      status,
		CreatedAt:   time.Now(),
	}
	
	if err := s.repo.CreateCompetition(ctx, competition); err != nil {
		return nil, fmt.Errorf("failed to create competition: %w", err)
	}
	
	return competition, nil
}

// JoinCompetition allows a user to join a competition
func (s *Service) JoinCompetition(ctx context.Context, competitionID, userID uuid.UUID, teamID *uuid.UUID) error {
	if competitionID == uuid.Nil {
		return fmt.Errorf("competition ID cannot be empty")
	}
	
	if userID == uuid.Nil {
		return fmt.Errorf("user ID cannot be empty")
	}
	
	participant := &CompetitionParticipant{
		ID:            uuid.New(),
		CompetitionID: competitionID,
		UserID:        userID,
		TeamID:        teamID,
		JoinedAt:      time.Now(),
	}
	
	if err := s.repo.JoinCompetition(ctx, participant); err != nil {
		return fmt.Errorf("failed to join competition: %w", err)
	}
	
	return nil
}

// RecordSocialShare records a social media share for bonus rewards
func (s *Service) RecordSocialShare(ctx context.Context, userID uuid.UUID, platform, content, milestone string) (*SocialShare, error) {
	if userID == uuid.Nil {
		return nil, fmt.Errorf("user ID cannot be empty")
	}
	
	// Validate platform
	validPlatforms := map[string]bool{
		"twitter":  true,
		"discord":  true,
		"telegram": true,
		"facebook": true,
	}
	
	if !validPlatforms[strings.ToLower(platform)] {
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}
	
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("content cannot be empty")
	}
	
	// Calculate bonus amount based on milestone
	bonusAmount := s.calculateSocialShareBonus(milestone)
	
	share := &SocialShare{
		ID:          uuid.New(),
		UserID:      userID,
		Platform:    strings.ToLower(platform),
		Content:     strings.TrimSpace(content),
		Milestone:   milestone,
		SharedAt:    time.Now(),
		BonusAmount: bonusAmount,
	}
	
	if err := s.repo.RecordSocialShare(ctx, share); err != nil {
		return nil, fmt.Errorf("failed to record social share: %w", err)
	}
	
	return share, nil
}

// GetTeamStatistics retrieves team performance statistics
func (s *Service) GetTeamStatistics(ctx context.Context, teamID uuid.UUID, period string, days int) ([]*TeamStatistics, error) {
	if teamID == uuid.Nil {
		return nil, fmt.Errorf("team ID cannot be empty")
	}
	
	validPeriods := map[string]bool{
		"daily":   true,
		"weekly":  true,
		"monthly": true,
	}
	
	if !validPeriods[period] {
		return nil, fmt.Errorf("invalid period: %s", period)
	}
	
	if days <= 0 {
		return nil, fmt.Errorf("days must be positive")
	}
	
	stats, err := s.repo.GetTeamStatistics(ctx, teamID, period, days)
	if err != nil {
		return nil, fmt.Errorf("failed to get team statistics: %w", err)
	}
	
	return stats, nil
}

// generateReferralCode generates a unique referral code
func (s *Service) generateReferralCode() (string, error) {
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	
	return "REF" + strings.ToUpper(hex.EncodeToString(bytes)), nil
}

// calculateSocialShareBonus calculates bonus amount based on milestone
func (s *Service) calculateSocialShareBonus(milestone string) float64 {
	bonuses := map[string]float64{
		"first_share":    5.0,
		"100_shares":     10.0,
		"1000_shares":    25.0,
		"10000_shares":   50.0,
		"first_block":    100.0,
		"team_creation":  15.0,
		"competition_win": 200.0,
	}
	
	if bonus, exists := bonuses[milestone]; exists {
		return bonus
	}
	
	return 1.0 // Default bonus
}