// Package keepalive implements connection keepalive for stratum miners
package keepalive

import (
	"errors"
	"sync"
	"time"
)

// Config holds keepalive configuration
type Config struct {
	Interval        time.Duration // How often to check for activity
	Timeout         time.Duration // How long to wait for response
	MaxMissed       int           // Max missed checks before timeout
	SendWorkAsAlive bool          // Use work updates as keepalive
}

// DefaultConfig returns sensible default configuration
func DefaultConfig() Config {
	return Config{
		Interval:        30 * time.Second, // Check every 30 seconds
		Timeout:         10 * time.Second, // 10 second timeout for responses
		MaxMissed:       3,                // 3 missed = disconnect
		SendWorkAsAlive: true,             // Work updates count as keepalive
	}
}

// Validate checks if the configuration is valid
func (c Config) Validate() error {
	if c.Interval <= 0 {
		return errors.New("interval must be positive")
	}
	if c.MaxMissed < 0 {
		return errors.New("max missed cannot be negative")
	}
	return nil
}

// TimeoutCallback is called when a miner times out
type TimeoutCallback func(minerID string)

// minerState holds per-miner keepalive state
type minerState struct {
	lastActivity time.Time
	missedCount  int
	stopChan     chan struct{}
	stopped      bool
}

// Manager implements keepalive management for miners
type Manager struct {
	config    Config
	miners    map[string]*minerState
	mu        sync.RWMutex
	onTimeout TimeoutCallback
}

// NewManager creates a new keepalive manager
func NewManager(config Config, onTimeout TimeoutCallback) *Manager {
	return &Manager{
		config:    config,
		miners:    make(map[string]*minerState),
		onTimeout: onTimeout,
	}
}

// Start begins keepalive monitoring for a miner
func (m *Manager) Start(minerID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Stop existing if any
	if existing, exists := m.miners[minerID]; exists {
		if !existing.stopped {
			close(existing.stopChan)
		}
	}

	state := &minerState{
		lastActivity: time.Now(),
		missedCount:  0,
		stopChan:     make(chan struct{}),
		stopped:      false,
	}
	m.miners[minerID] = state

	// Start monitoring goroutine
	go m.monitor(minerID, state)
}

// Stop stops keepalive monitoring for a miner
func (m *Manager) Stop(minerID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if state, exists := m.miners[minerID]; exists {
		if !state.stopped {
			state.stopped = true
			close(state.stopChan)
		}
		delete(m.miners, minerID)
	}
}

// RecordActivity records activity from a miner (resets timeout)
func (m *Manager) RecordActivity(minerID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if state, exists := m.miners[minerID]; exists {
		state.lastActivity = time.Now()
		state.missedCount = 0
	}
}

// IsAlive checks if a miner is currently being monitored and alive
func (m *Manager) IsAlive(minerID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, exists := m.miners[minerID]
	if !exists {
		return false
	}
	return !state.stopped
}

// GetConfig returns the current configuration
func (m *Manager) GetConfig() Config {
	return m.config
}

// monitor runs the keepalive check loop for a miner
func (m *Manager) monitor(minerID string, state *minerState) {
	ticker := time.NewTicker(m.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-state.stopChan:
			return
		case <-ticker.C:
			m.checkMiner(minerID, state)
		}
	}
}

// checkMiner checks if a miner is still active
func (m *Manager) checkMiner(minerID string, state *minerState) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if miner was stopped
	if state.stopped {
		return
	}

	// Check last activity
	timeSinceActivity := time.Since(state.lastActivity)
	if timeSinceActivity > m.config.Interval {
		state.missedCount++

		if state.missedCount >= m.config.MaxMissed {
			// Miner timed out
			state.stopped = true
			delete(m.miners, minerID)

			// Call timeout callback outside of lock
			if m.onTimeout != nil {
				go m.onTimeout(minerID)
			}
		}
	} else {
		// Activity within interval, reset counter
		state.missedCount = 0
	}
}

// GetActiveCount returns the number of active miners being monitored
func (m *Manager) GetActiveCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.miners)
}
