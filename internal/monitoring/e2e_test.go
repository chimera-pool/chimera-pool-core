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

// Note: Community E2E tests have been moved to internal/community/e2e_test.go
// This file focuses on monitoring-specific E2E tests
