package notifications

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// WORKER MONITOR TESTS (TDD)
// =============================================================================

func TestWorkerMonitor_Creation(t *testing.T) {
	t.Run("creates monitor with default config", func(t *testing.T) {
		config := DefaultWorkerMonitorConfig()
		monitor := NewWorkerMonitor(config, nil, nil)

		require.NotNil(t, monitor)
		assert.Equal(t, 5*time.Minute, monitor.config.OfflineThreshold)
	})

	t.Run("creates monitor with custom config", func(t *testing.T) {
		config := &WorkerMonitorConfig{
			OfflineThreshold: 10 * time.Minute,
			CheckInterval:    time.Minute,
		}
		monitor := NewWorkerMonitor(config, nil, nil)

		require.NotNil(t, monitor)
		assert.Equal(t, 10*time.Minute, monitor.config.OfflineThreshold)
	})
}

func TestWorkerMonitor_RecordActivity(t *testing.T) {
	t.Run("records worker activity", func(t *testing.T) {
		monitor := NewWorkerMonitor(DefaultWorkerMonitorConfig(), nil, nil)

		monitor.RecordActivity(1, 100, "rig1")

		state := monitor.GetWorkerState(100)
		require.NotNil(t, state)
		assert.Equal(t, int64(100), state.WorkerID)
		assert.Equal(t, "rig1", state.WorkerName)
		assert.True(t, state.IsOnline)
	})

	t.Run("updates last seen time on activity", func(t *testing.T) {
		monitor := NewWorkerMonitor(DefaultWorkerMonitorConfig(), nil, nil)

		monitor.RecordActivity(1, 100, "rig1")
		firstSeen := monitor.GetWorkerState(100).LastSeen

		time.Sleep(10 * time.Millisecond)
		monitor.RecordActivity(1, 100, "rig1")
		secondSeen := monitor.GetWorkerState(100).LastSeen

		assert.True(t, secondSeen.After(firstSeen))
	})
}

func TestWorkerMonitor_DetectOffline(t *testing.T) {
	t.Run("detects offline workers", func(t *testing.T) {
		config := &WorkerMonitorConfig{
			OfflineThreshold: 50 * time.Millisecond,
			CheckInterval:    10 * time.Millisecond,
		}

		alertsSent := make([]*Alert, 0)
		mockNotifier := &mockNotificationService{
			onSend: func(alert *Alert) {
				alertsSent = append(alertsSent, alert)
			},
		}

		monitor := NewWorkerMonitor(config, mockNotifier, nil)

		// Record activity
		monitor.RecordActivity(1, 100, "rig1")

		// Wait for threshold to pass
		time.Sleep(100 * time.Millisecond)

		// Check for offline workers
		offline := monitor.CheckOfflineWorkers(context.Background())

		assert.Len(t, offline, 1)
		assert.Equal(t, int64(100), offline[0].WorkerID)
	})

	t.Run("sends alert when worker goes offline", func(t *testing.T) {
		config := &WorkerMonitorConfig{
			OfflineThreshold: 50 * time.Millisecond,
			CheckInterval:    10 * time.Millisecond,
		}

		alertsSent := make([]*Alert, 0)
		mockNotifier := &mockNotificationService{
			onSend: func(alert *Alert) {
				alertsSent = append(alertsSent, alert)
			},
		}

		monitor := NewWorkerMonitor(config, mockNotifier, nil)
		monitor.RecordActivity(1, 100, "rig1")

		time.Sleep(100 * time.Millisecond)
		monitor.CheckOfflineWorkers(context.Background())

		require.Len(t, alertsSent, 1)
		assert.Equal(t, AlertTypeWorkerOffline, alertsSent[0].Type)
		assert.Equal(t, "rig1", alertsSent[0].WorkerName)
	})

	t.Run("does not alert twice for same offline worker", func(t *testing.T) {
		config := &WorkerMonitorConfig{
			OfflineThreshold: 50 * time.Millisecond,
			CheckInterval:    10 * time.Millisecond,
		}

		alertCount := 0
		mockNotifier := &mockNotificationService{
			onSend: func(alert *Alert) {
				alertCount++
			},
		}

		monitor := NewWorkerMonitor(config, mockNotifier, nil)
		monitor.RecordActivity(1, 100, "rig1")

		time.Sleep(100 * time.Millisecond)

		// Check twice
		monitor.CheckOfflineWorkers(context.Background())
		monitor.CheckOfflineWorkers(context.Background())

		assert.Equal(t, 1, alertCount) // Only one alert
	})
}

func TestWorkerMonitor_DetectBackOnline(t *testing.T) {
	t.Run("sends alert when worker comes back online", func(t *testing.T) {
		config := &WorkerMonitorConfig{
			OfflineThreshold: 50 * time.Millisecond,
			CheckInterval:    10 * time.Millisecond,
		}

		alertsSent := make([]*Alert, 0)
		mockNotifier := &mockNotificationService{
			onSend: func(alert *Alert) {
				alertsSent = append(alertsSent, alert)
			},
		}

		monitor := NewWorkerMonitor(config, mockNotifier, nil)

		// Record activity, then go offline
		monitor.RecordActivity(1, 100, "rig1")
		time.Sleep(100 * time.Millisecond)
		monitor.CheckOfflineWorkers(context.Background())

		// Worker comes back
		monitor.RecordActivity(1, 100, "rig1")

		// Should have 2 alerts: offline and online
		require.Len(t, alertsSent, 2)
		assert.Equal(t, AlertTypeWorkerOffline, alertsSent[0].Type)
		assert.Equal(t, AlertTypeWorkerOnline, alertsSent[1].Type)
	})
}

func TestWorkerMonitor_GetStats(t *testing.T) {
	t.Run("returns worker statistics", func(t *testing.T) {
		monitor := NewWorkerMonitor(DefaultWorkerMonitorConfig(), nil, nil)

		monitor.RecordActivity(1, 100, "rig1")
		monitor.RecordActivity(1, 101, "rig2")
		monitor.RecordActivity(2, 102, "rig3")

		stats := monitor.GetStats()

		assert.Equal(t, 3, stats.TotalWorkers)
		assert.Equal(t, 3, stats.OnlineWorkers)
		assert.Equal(t, 0, stats.OfflineWorkers)
	})
}

// =============================================================================
// MOCK NOTIFICATION SERVICE
// =============================================================================

type mockNotificationService struct {
	onSend func(alert *Alert)
}

func (m *mockNotificationService) SendAlert(ctx context.Context, alert *Alert) ([]NotificationResult, error) {
	if m.onSend != nil {
		m.onSend(alert)
	}
	return []NotificationResult{{Success: true}}, nil
}
