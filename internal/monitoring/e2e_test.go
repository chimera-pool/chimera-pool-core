package monitoring

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestMonitoringE2E tests the complete monitoring workflow
func TestMonitoringE2E(t *testing.T) {
	// This would typically use a real database connection
	// For now, we'll use mocks to demonstrate the E2E flow
	
	mockRepo := &MockRepository{}
	mockPrometheus := &MockPrometheusClient{}
	service := NewService(mockRepo, mockPrometheus)
	
	ctx := context.Background()
	
	t.Run("Complete Monitoring Workflow", func(t *testing.T) {
		// Step 1: Record performance metrics
		perfMetrics := &PerformanceMetrics{
			Timestamp:       time.Now(),
			CPUUsage:        85.5,
			MemoryUsage:     70.2,
			DiskUsage:       45.0,
			NetworkIn:       1000.0,
			NetworkOut:      800.0,
			ActiveMiners:    150,
			TotalHashrate:   1500000.0,
			SharesPerSecond: 25.5,
			BlocksFound:     5,
			Uptime:          86400.0,
		}
		
		mockRepo.On("StorePerformanceMetrics", ctx, perfMetrics).Return(nil)
		mockPrometheus.On("RecordGauge", "pool_cpu_usage", map[string]string{"component": "pool"}, 85.5).Return(nil)
		mockPrometheus.On("RecordGauge", "pool_memory_usage", map[string]string{"component": "pool"}, 70.2).Return(nil)
		mockPrometheus.On("RecordGauge", "pool_disk_usage", map[string]string{"component": "pool"}, 45.0).Return(nil)
		mockPrometheus.On("RecordGauge", "pool_active_miners", map[string]string{"component": "pool"}, 150.0).Return(nil)
		mockPrometheus.On("RecordGauge", "pool_total_hashrate", map[string]string{"component": "pool"}, 1500000.0).Return(nil)
		mockPrometheus.On("RecordGauge", "pool_shares_per_second", map[string]string{"component": "pool"}, 25.5).Return(nil)
		mockPrometheus.On("RecordCounter", "pool_blocks_found_total", map[string]string{"component": "pool"}, 5.0).Return(nil)
		mockPrometheus.On("RecordGauge", "pool_uptime_seconds", map[string]string{"component": "pool"}, 86400.0).Return(nil)
		
		err := service.RecordPerformanceMetrics(ctx, perfMetrics)
		require.NoError(t, err)
		
		// Step 2: Record miner metrics
		minerID := uuid.New()
		minerMetrics := &MinerMetrics{
			MinerID:         minerID,
			Timestamp:       time.Now(),
			Hashrate:        10000.0,
			SharesSubmitted: 100,
			SharesAccepted:  95,
			SharesRejected:  5,
			LastSeen:        time.Now(),
			IsOnline:        true,
			Difficulty:      1000.0,
			Earnings:        0.05,
		}
		
		mockRepo.On("StoreMinerMetrics", ctx, minerMetrics).Return(nil)
		minerLabels := map[string]string{"miner_id": minerID.String()}
		mockPrometheus.On("RecordGauge", "miner_hashrate", minerLabels, 10000.0).Return(nil)
		mockPrometheus.On("RecordCounter", "miner_shares_submitted_total", minerLabels, 100.0).Return(nil)
		mockPrometheus.On("RecordCounter", "miner_shares_accepted_total", minerLabels, 95.0).Return(nil)
		mockPrometheus.On("RecordCounter", "miner_shares_rejected_total", minerLabels, 5.0).Return(nil)
		mockPrometheus.On("RecordGauge", "miner_difficulty", minerLabels, 1000.0).Return(nil)
		mockPrometheus.On("RecordGauge", "miner_earnings", minerLabels, 0.05).Return(nil)
		mockPrometheus.On("RecordGauge", "miner_online", minerLabels, 1.0).Return(nil)
		
		err = service.RecordMinerMetrics(ctx, minerMetrics)
		require.NoError(t, err)
		
		// Step 3: Create alert rule
		mockRepo.On("CreateAlertRule", ctx, mock.AnythingOfType("*monitoring.AlertRule")).Return(nil)
		
		alertRule, err := service.CreateAlertRule(ctx, "High CPU Usage", "cpu_usage", ">", 90.0, "5m", "warning")
		require.NoError(t, err)
		assert.Equal(t, "High CPU Usage", alertRule.Name)
		assert.Equal(t, "cpu_usage", alertRule.Query)
		assert.Equal(t, ">", alertRule.Condition)
		assert.Equal(t, 90.0, alertRule.Threshold)
		
		// Step 4: Evaluate alert rules (should not trigger)
		mockRepo.On("GetAlertRules", ctx).Return([]*AlertRule{alertRule}, nil)
		mockPrometheus.On("Query", "cpu_usage").Return(85.5, nil) // Below threshold
		
		alerts, err := service.EvaluateAlertRules(ctx)
		require.NoError(t, err)
		assert.Len(t, alerts, 0) // No alerts should be created
		
		// Step 5: Evaluate alert rules (should trigger)
		mockPrometheus.On("Query", "cpu_usage").Return(95.0, nil) // Above threshold
		mockRepo.On("CreateAlert", ctx, mock.AnythingOfType("*monitoring.Alert")).Return(nil)
		
		alerts, err = service.EvaluateAlertRules(ctx)
		require.NoError(t, err)
		assert.Len(t, alerts, 1) // One alert should be created
		assert.Contains(t, alerts[0].Name, "High CPU Usage")
		assert.Equal(t, "warning", alerts[0].Severity)
		
		// Step 6: Create dashboard
		userID := uuid.New()
		dashboardConfig := `{
			"panels": [
				{
					"title": "CPU Usage",
					"type": "graph",
					"query": "cpu_usage"
				},
				{
					"title": "Active Miners",
					"type": "stat",
					"query": "pool_active_miners"
				}
			]
		}`
		
		mockRepo.On("CreateDashboard", ctx, mock.AnythingOfType("*monitoring.Dashboard")).Return(nil)
		
		dashboard, err := service.CreateDashboard(ctx, "Pool Overview", "Main monitoring dashboard", dashboardConfig, true, userID)
		require.NoError(t, err)
		assert.Equal(t, "Pool Overview", dashboard.Name)
		assert.Equal(t, "Main monitoring dashboard", dashboard.Description)
		assert.True(t, dashboard.IsPublic)
		assert.Equal(t, userID, dashboard.CreatedBy)
		
		mockRepo.AssertExpectations(t)
		mockPrometheus.AssertExpectations(t)
	})
}

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
		
		mockRepo.AssertExpectations(t)
	})
}

// TestIntegratedMonitoringAndCommunity tests integration between monitoring and community features
func TestIntegratedMonitoringAndCommunity(t *testing.T) {
	mockMonitoringRepo := &MockRepository{}
	mockCommunityRepo := &MockRepository{}
	mockPrometheus := &MockPrometheusClient{}
	
	monitoringService := NewService(mockMonitoringRepo, mockPrometheus)
	communityService := NewService(mockCommunityRepo)
	
	ctx := context.Background()
	
	t.Run("Team Performance Monitoring", func(t *testing.T) {
		// Create a team
		ownerID := uuid.New()
		mockCommunityRepo.On("CreateTeam", ctx, mock.AnythingOfType("*community.Team")).Return(nil)
		mockCommunityRepo.On("AddTeamMember", ctx, mock.AnythingOfType("*community.TeamMember")).Return(nil)
		
		team, err := communityService.CreateTeam(ctx, "Performance Team", "High performance mining team", ownerID)
		require.NoError(t, err)
		
		// Record team member mining metrics
		memberID := uuid.New()
		minerMetrics := &MinerMetrics{
			MinerID:         memberID,
			Timestamp:       time.Now(),
			Hashrate:        50000.0,
			SharesSubmitted: 500,
			SharesAccepted:  475,
			SharesRejected:  25,
			LastSeen:        time.Now(),
			IsOnline:        true,
			Difficulty:      2000.0,
			Earnings:        0.25,
		}
		
		mockMonitoringRepo.On("StoreMinerMetrics", ctx, minerMetrics).Return(nil)
		memberLabels := map[string]string{"miner_id": memberID.String()}
		mockPrometheus.On("RecordGauge", "miner_hashrate", memberLabels, 50000.0).Return(nil)
		mockPrometheus.On("RecordCounter", "miner_shares_submitted_total", memberLabels, 500.0).Return(nil)
		mockPrometheus.On("RecordCounter", "miner_shares_accepted_total", memberLabels, 475.0).Return(nil)
		mockPrometheus.On("RecordCounter", "miner_shares_rejected_total", memberLabels, 25.0).Return(nil)
		mockPrometheus.On("RecordGauge", "miner_difficulty", memberLabels, 2000.0).Return(nil)
		mockPrometheus.On("RecordGauge", "miner_earnings", memberLabels, 0.25).Return(nil)
		mockPrometheus.On("RecordGauge", "miner_online", memberLabels, 1.0).Return(nil)
		
		err = monitoringService.RecordMinerMetrics(ctx, minerMetrics)
		require.NoError(t, err)
		
		// Create alert for team performance
		mockMonitoringRepo.On("CreateAlertRule", ctx, mock.AnythingOfType("*monitoring.AlertRule")).Return(nil)
		
		alertRule, err := monitoringService.CreateAlertRule(ctx, 
			"Team Low Performance", 
			"avg(miner_hashrate{team_id=\"" + team.ID.String() + "\"})", 
			"<", 
			10000.0, 
			"10m", 
			"warning")
		require.NoError(t, err)
		assert.Contains(t, alertRule.Query, team.ID.String())
		
		// Record social share for achievement
		mockCommunityRepo.On("RecordSocialShare", ctx, mock.AnythingOfType("*community.SocialShare")).Return(nil)
		
		share, err := communityService.RecordSocialShare(ctx, memberID, "discord", "Our team just hit 50k hashrate! 💪", "team_milestone")
		require.NoError(t, err)
		assert.Equal(t, "discord", share.Platform)
		
		mockMonitoringRepo.AssertExpectations(t)
		mockCommunityRepo.AssertExpectations(t)
		mockPrometheus.AssertExpectations(t)
	})
}