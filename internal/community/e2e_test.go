package community

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestCommunityE2E tests the complete community features workflow
func TestCommunityE2E(t *testing.T) {
	mockRepo := &MockRepository{}
	service := NewService(mockRepo)
	
	ctx := context.Background()
	
	t.Run("Complete Community Workflow", func(t *testing.T) {
		// Step 1: Create a team
		ownerID := uuid.New()
		mockRepo.On("CreateTeam", ctx, mock.AnythingOfType("*community.Team")).Return(nil)
		mockRepo.On("AddTeamMember", ctx, mock.AnythingOfType("*community.TeamMember")).Return(nil)
		
		team, err := service.CreateTeam(ctx, "Elite Miners", "Top performing mining team", ownerID)
		require.NoError(t, err)
		assert.Equal(t, "Elite Miners", team.Name)
		assert.Equal(t, "Top performing mining team", team.Description)
		assert.Equal(t, ownerID, team.OwnerID)
		assert.True(t, team.IsActive)
		
		// Step 2: Another user joins the team
		memberID := uuid.New()
		mockRepo.On("GetTeam", ctx, team.ID).Return(team, nil)
		mockRepo.On("AddTeamMember", ctx, mock.AnythingOfType("*community.TeamMember")).Return(nil)
		
		err = service.JoinTeam(ctx, team.ID, memberID)
		require.NoError(t, err)
		
		// Step 3: Create referral code
		mockRepo.On("CreateReferral", ctx, mock.AnythingOfType("*community.Referral")).Return(nil)
		
		referral, err := service.CreateReferral(ctx, ownerID)
		require.NoError(t, err)
		assert.Equal(t, ownerID, referral.ReferrerID)
		assert.NotEmpty(t, referral.Code)
		assert.Equal(t, "pending", referral.Status)
		
		// Step 4: Process referral when new user joins
		newUserID := uuid.New()
		mockRepo.On("GetReferral", ctx, referral.Code).Return(referral, nil)
		mockRepo.On("UpdateReferral", ctx, mock.AnythingOfType("*community.Referral")).Return(nil)
		
		err = service.ProcessReferral(ctx, referral.Code, newUserID)
		require.NoError(t, err)
		
		// Step 5: Create mining competition
		startTime := time.Now().Add(24 * time.Hour)
		endTime := startTime.Add(7 * 24 * time.Hour)
		mockRepo.On("CreateCompetition", ctx, mock.AnythingOfType("*community.Competition")).Return(nil)
		
		competition, err := service.CreateCompetition(ctx, "Weekly Hashrate Challenge", "Compete for highest hashrate", startTime, endTime, 1000.0)
		require.NoError(t, err)
		assert.Equal(t, "Weekly Hashrate Challenge", competition.Name)
		assert.Equal(t, 1000.0, competition.PrizePool)
		assert.Equal(t, "upcoming", competition.Status)
		
		// Step 6: User joins competition
		mockRepo.On("JoinCompetition", ctx, mock.AnythingOfType("*community.CompetitionParticipant")).Return(nil)
		
		err = service.JoinCompetition(ctx, competition.ID, memberID, &team.ID)
		require.NoError(t, err)
		
		// Step 7: Record social share
		mockRepo.On("RecordSocialShare", ctx, mock.AnythingOfType("*community.SocialShare")).Return(nil)
		
		share, err := service.RecordSocialShare(ctx, memberID, "twitter", "Just joined Elite Miners team on ChimeraPool! 🚀", "team_creation")
		require.NoError(t, err)
		assert.Equal(t, memberID, share.UserID)
		assert.Equal(t, "twitter", share.Platform)
		assert.Equal(t, "team_creation", share.Milestone)
		assert.Equal(t, 15.0, share.BonusAmount) // team_creation bonus
		
		// Step 8: Get team statistics
		mockRepo.On("GetTeamStatistics", ctx, team.ID, "daily", 30).Return([]*TeamStatistics{
			{
				TeamID:        team.ID,
				Period:        "daily",
				Date:          time.Now().Truncate(24 * time.Hour),
				TotalHashrate: 50000.0,
				TotalShares:   1000,
				BlocksFound:   2,
				Earnings:      0.5,
				MemberCount:   2,
			},
		}, nil)
		
		stats, err := service.GetTeamStatistics(ctx, team.ID, "daily", 30)
		require.NoError(t, err)
		assert.Len(t, stats, 1)
		assert.Equal(t, team.ID, stats[0].TeamID)
		assert.Equal(t, "daily", stats[0].Period)
		assert.Equal(t, 50000.0, stats[0].TotalHashrate)
		assert.Equal(t, int64(1000), stats[0].TotalShares)
		assert.Equal(t, 2, stats[0].BlocksFound)
		assert.Equal(t, 0.5, stats[0].Earnings)
		assert.Equal(t, 2, stats[0].MemberCount)
		
		mockRepo.AssertExpectations(t)
	})
}

// TestCommunityFeatureIntegration tests integration between different community features
func TestCommunityFeatureIntegration(t *testing.T) {
	mockRepo := &MockRepository{}
	service := NewService(mockRepo)
	
	ctx := context.Background()
	
	t.Run("Team Competition Integration", func(t *testing.T) {
		// Create team
		ownerID := uuid.New()
		mockRepo.On("CreateTeam", ctx, mock.AnythingOfType("*community.Team")).Return(nil)
		mockRepo.On("AddTeamMember", ctx, mock.AnythingOfType("*community.TeamMember")).Return(nil)
		
		team, err := service.CreateTeam(ctx, "Competition Team", "Team for competitions", ownerID)
		require.NoError(t, err)
		
		// Create competition
		startTime := time.Now().Add(1 * time.Hour)
		endTime := startTime.Add(24 * time.Hour)
		mockRepo.On("CreateCompetition", ctx, mock.AnythingOfType("*community.Competition")).Return(nil)
		
		competition, err := service.CreateCompetition(ctx, "Team Challenge", "Team-based mining competition", startTime, endTime, 500.0)
		require.NoError(t, err)
		
		// Team joins competition
		mockRepo.On("JoinCompetition", ctx, mock.AnythingOfType("*community.CompetitionParticipant")).Return(nil)
		
		err = service.JoinCompetition(ctx, competition.ID, ownerID, &team.ID)
		require.NoError(t, err)
		
		// Record social share about competition
		mockRepo.On("RecordSocialShare", ctx, mock.AnythingOfType("*community.SocialShare")).Return(nil)
		
		share, err := service.RecordSocialShare(ctx, ownerID, "discord", "Our team just joined the Team Challenge competition! 💪", "competition_join")
		require.NoError(t, err)
		assert.Equal(t, "discord", share.Platform)
		assert.Equal(t, "competition_join", share.Milestone)
		
		mockRepo.AssertExpectations(t)
	})
	
	t.Run("Referral Team Integration", func(t *testing.T) {
		// Create referral
		referrerID := uuid.New()
		mockRepo.On("CreateReferral", ctx, mock.AnythingOfType("*community.Referral")).Return(nil)
		
		referral, err := service.CreateReferral(ctx, referrerID)
		require.NoError(t, err)
		
		// Process referral
		referredID := uuid.New()
		mockRepo.On("GetReferral", ctx, referral.Code).Return(referral, nil)
		mockRepo.On("UpdateReferral", ctx, mock.AnythingOfType("*community.Referral")).Return(nil)
		
		err = service.ProcessReferral(ctx, referral.Code, referredID)
		require.NoError(t, err)
		
		// Referred user creates team
		mockRepo.On("CreateTeam", ctx, mock.AnythingOfType("*community.Team")).Return(nil)
		mockRepo.On("AddTeamMember", ctx, mock.AnythingOfType("*community.TeamMember")).Return(nil)
		
		team, err := service.CreateTeam(ctx, "Referred Team", "Team created by referred user", referredID)
		require.NoError(t, err)
		
		// Referrer joins the team
		mockRepo.On("GetTeam", ctx, team.ID).Return(team, nil)
		mockRepo.On("AddTeamMember", ctx, mock.AnythingOfType("*community.TeamMember")).Return(nil)
		
		err = service.JoinTeam(ctx, team.ID, referrerID)
		require.NoError(t, err)
		
		// Record social share about successful referral
		mockRepo.On("RecordSocialShare", ctx, mock.AnythingOfType("*community.SocialShare")).Return(nil)
		
		share, err := service.RecordSocialShare(ctx, referrerID, "twitter", "Successfully referred a new miner who created an awesome team! 🎉", "successful_referral")
		require.NoError(t, err)
		assert.Equal(t, "successful_referral", share.Milestone)
		
		mockRepo.AssertExpectations(t)
	})
}

// TestCommunityErrorHandling tests error handling in community features
func TestCommunityErrorHandling(t *testing.T) {
	mockRepo := &MockRepository{}
	service := NewService(mockRepo)
	
	ctx := context.Background()
	
	t.Run("Invalid Team Operations", func(t *testing.T) {
		// Try to join non-existent team
		nonExistentTeamID := uuid.New()
		userID := uuid.New()
		
		mockRepo.On("GetTeam", ctx, nonExistentTeamID).Return((*Team)(nil), assert.AnError)
		
		err := service.JoinTeam(ctx, nonExistentTeamID, userID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get team")
		
		// Try to join inactive team
		inactiveTeam := &Team{
			ID:       uuid.New(),
			Name:     "Inactive Team",
			IsActive: false,
		}
		
		mockRepo.On("GetTeam", ctx, inactiveTeam.ID).Return(inactiveTeam, nil)
		
		err = service.JoinTeam(ctx, inactiveTeam.ID, userID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "team is not active")
		
		mockRepo.AssertExpectations(t)
	})
	
	t.Run("Invalid Referral Operations", func(t *testing.T) {
		// Try to process expired referral
		expiredReferral := &Referral{
			ID:         uuid.New(),
			ReferrerID: uuid.New(),
			Code:       "EXPIRED123",
			Status:     "pending",
			CreatedAt:  time.Now().Add(-31 * 24 * time.Hour), // 31 days ago
		}
		
		mockRepo.On("GetReferral", ctx, "EXPIRED123").Return(expiredReferral, nil)
		
		err := service.ProcessReferral(ctx, "EXPIRED123", uuid.New())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "referral has expired")
		
		// Try to process already completed referral
		completedReferral := &Referral{
			ID:         uuid.New(),
			ReferrerID: uuid.New(),
			Code:       "COMPLETED123",
			Status:     "completed",
			CreatedAt:  time.Now().Add(-1 * time.Hour),
		}
		
		mockRepo.On("GetReferral", ctx, "COMPLETED123").Return(completedReferral, nil)
		
		err = service.ProcessReferral(ctx, "COMPLETED123", uuid.New())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "referral is not pending")
		
		mockRepo.AssertExpectations(t)
	})
	
	t.Run("Invalid Competition Operations", func(t *testing.T) {
		// Try to create competition with end time before start time
		startTime := time.Now().Add(24 * time.Hour)
		endTime := startTime.Add(-1 * time.Hour) // End before start
		
		competition, err := service.CreateCompetition(ctx, "Invalid Competition", "Invalid time range", startTime, endTime, 100.0)
		assert.Error(t, err)
		assert.Nil(t, competition)
		assert.Contains(t, err.Error(), "end time must be after start time")
		
		// Try to create competition with negative prize pool
		competition, err = service.CreateCompetition(ctx, "Invalid Prize", "Negative prize", time.Now().Add(1*time.Hour), time.Now().Add(25*time.Hour), -100.0)
		assert.Error(t, err)
		assert.Nil(t, competition)
		assert.Contains(t, err.Error(), "prize pool cannot be negative")
	})
	
	t.Run("Invalid Social Share Operations", func(t *testing.T) {
		userID := uuid.New()
		
		// Try to record share with invalid platform
		share, err := service.RecordSocialShare(ctx, userID, "invalid_platform", "Test content", "test_milestone")
		assert.Error(t, err)
		assert.Nil(t, share)
		assert.Contains(t, err.Error(), "unsupported platform")
		
		// Try to record share with empty content
		share, err = service.RecordSocialShare(ctx, userID, "twitter", "", "test_milestone")
		assert.Error(t, err)
		assert.Nil(t, share)
		assert.Contains(t, err.Error(), "content cannot be empty")
	})
}