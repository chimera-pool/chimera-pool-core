package notifications

import (
	"context"
	"sync"
	"time"
)

// =============================================================================
// WORKER MONITOR - DETECTS OFFLINE WORKERS
// =============================================================================

// WorkerMonitorConfig holds configuration for worker monitoring
type WorkerMonitorConfig struct {
	OfflineThreshold time.Duration // Time without activity before considered offline
	CheckInterval    time.Duration // How often to check for offline workers
	AlertCooldown    time.Duration // Minimum time between alerts for same worker
}

// DefaultWorkerMonitorConfig returns sensible defaults
func DefaultWorkerMonitorConfig() *WorkerMonitorConfig {
	return &WorkerMonitorConfig{
		OfflineThreshold: 5 * time.Minute,
		CheckInterval:    30 * time.Second,
		AlertCooldown:    30 * time.Minute,
	}
}

// WorkerState tracks the state of a single worker
type WorkerState struct {
	WorkerID    int64      `json:"worker_id"`
	UserID      int64      `json:"user_id"`
	WorkerName  string     `json:"worker_name"`
	LastSeen    time.Time  `json:"last_seen"`
	IsOnline    bool       `json:"is_online"`
	AlertSentAt *time.Time `json:"alert_sent_at,omitempty"`
}

// WorkerMonitorStats holds monitoring statistics
type WorkerMonitorStats struct {
	TotalWorkers   int `json:"total_workers"`
	OnlineWorkers  int `json:"online_workers"`
	OfflineWorkers int `json:"offline_workers"`
}

// AlertSender interface for sending alerts (ISP)
type AlertSender interface {
	SendAlert(ctx context.Context, alert *Alert) ([]NotificationResult, error)
}

// WorkerActivityProvider provides worker activity data (ISP)
type WorkerActivityProvider interface {
	GetAllWorkers(ctx context.Context) ([]WorkerState, error)
	GetWorkerLastActivity(ctx context.Context, workerID int64) (time.Time, error)
}

// WorkerMonitor monitors worker activity and sends offline alerts
type WorkerMonitor struct {
	config           *WorkerMonitorConfig
	notifier         AlertSender
	activityProvider WorkerActivityProvider
	workers          map[int64]*WorkerState
	mu               sync.RWMutex
	stopCh           chan struct{}
	running          bool
}

// NewWorkerMonitor creates a new worker monitor
func NewWorkerMonitor(config *WorkerMonitorConfig, notifier AlertSender, provider WorkerActivityProvider) *WorkerMonitor {
	if config == nil {
		config = DefaultWorkerMonitorConfig()
	}
	return &WorkerMonitor{
		config:           config,
		notifier:         notifier,
		activityProvider: provider,
		workers:          make(map[int64]*WorkerState),
		stopCh:           make(chan struct{}),
	}
}

// Start begins monitoring workers in the background
func (m *WorkerMonitor) Start() {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return
	}
	m.running = true
	m.mu.Unlock()

	go m.monitorLoop()
}

// Stop stops the monitor
func (m *WorkerMonitor) Stop() {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return
	}
	m.running = false
	m.mu.Unlock()

	close(m.stopCh)
}

// RecordActivity records worker activity (call this when share is submitted)
func (m *WorkerMonitor) RecordActivity(userID, workerID int64, workerName string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, exists := m.workers[workerID]
	now := time.Now()

	if !exists {
		m.workers[workerID] = &WorkerState{
			WorkerID:   workerID,
			UserID:     userID,
			WorkerName: workerName,
			LastSeen:   now,
			IsOnline:   true,
		}
		return
	}

	// Worker coming back online?
	if !state.IsOnline {
		state.IsOnline = true
		state.AlertSentAt = nil

		// Send back online alert
		if m.notifier != nil {
			alert := NewWorkerOnlineAlert(userID, workerID, workerName)
			m.notifier.SendAlert(context.Background(), alert)
		}
	}

	state.LastSeen = now
	state.WorkerName = workerName
	state.UserID = userID
}

// GetWorkerState returns the state of a specific worker
func (m *WorkerMonitor) GetWorkerState(workerID int64) *WorkerState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if state, exists := m.workers[workerID]; exists {
		// Return a copy
		copy := *state
		return &copy
	}
	return nil
}

// CheckOfflineWorkers checks for offline workers and sends alerts
func (m *WorkerMonitor) CheckOfflineWorkers(ctx context.Context) []*WorkerState {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	threshold := now.Add(-m.config.OfflineThreshold)
	offline := make([]*WorkerState, 0)

	for _, state := range m.workers {
		if state.LastSeen.Before(threshold) && state.IsOnline {
			// Worker is offline
			state.IsOnline = false
			offline = append(offline, state)

			// Send alert if not recently sent
			if m.shouldSendAlert(state) {
				if m.notifier != nil {
					alert := NewWorkerOfflineAlert(state.UserID, state.WorkerID, state.WorkerName)
					m.notifier.SendAlert(ctx, alert)
				}
				state.AlertSentAt = &now
			}
		}
	}

	return offline
}

// GetStats returns current monitoring statistics
func (m *WorkerMonitor) GetStats() WorkerMonitorStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := WorkerMonitorStats{
		TotalWorkers: len(m.workers),
	}

	for _, state := range m.workers {
		if state.IsOnline {
			stats.OnlineWorkers++
		} else {
			stats.OfflineWorkers++
		}
	}

	return stats
}

// GetAllWorkerStates returns all tracked workers
func (m *WorkerMonitor) GetAllWorkerStates() []*WorkerState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	states := make([]*WorkerState, 0, len(m.workers))
	for _, state := range m.workers {
		copy := *state
		states = append(states, &copy)
	}
	return states
}

// GetOfflineWorkers returns all currently offline workers
func (m *WorkerMonitor) GetOfflineWorkers() []*WorkerState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	offline := make([]*WorkerState, 0)
	for _, state := range m.workers {
		if !state.IsOnline {
			copy := *state
			offline = append(offline, &copy)
		}
	}
	return offline
}

// =============================================================================
// INTERNAL METHODS
// =============================================================================

func (m *WorkerMonitor) monitorLoop() {
	ticker := time.NewTicker(m.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.CheckOfflineWorkers(context.Background())
		}
	}
}

func (m *WorkerMonitor) shouldSendAlert(state *WorkerState) bool {
	if state.AlertSentAt == nil {
		return true
	}
	return time.Since(*state.AlertSentAt) > m.config.AlertCooldown
}

// LoadWorkersFromProvider loads initial worker state from provider
func (m *WorkerMonitor) LoadWorkersFromProvider(ctx context.Context) error {
	if m.activityProvider == nil {
		return nil
	}

	workers, err := m.activityProvider.GetAllWorkers(ctx)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, w := range workers {
		m.workers[w.WorkerID] = &WorkerState{
			WorkerID:   w.WorkerID,
			UserID:     w.UserID,
			WorkerName: w.WorkerName,
			LastSeen:   w.LastSeen,
			IsOnline:   w.IsOnline,
		}
	}

	return nil
}
