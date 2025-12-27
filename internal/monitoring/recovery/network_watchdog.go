package recovery

import (
	"context"
	"log"
	"net"
	"sync"
	"time"
)

// =============================================================================
// NETWORK WATCHDOG IMPLEMENTATION
// Monitors internet connectivity and notifies observers on state changes
// =============================================================================

// DefaultNetworkWatchdog implements NetworkWatchdog interface
type DefaultNetworkWatchdog struct {
	config    *NetworkWatchdogConfig
	observers []NetworkStateObserver
	stats     *NetworkWatchdogStats
	mu        sync.RWMutex
	running   bool
	stopCh    chan struct{}

	// State tracking
	consecutiveFailures  int
	consecutiveSuccesses int
	currentState         NetworkState
	outageStart          time.Time
}

// NewNetworkWatchdog creates a new network watchdog
func NewNetworkWatchdog(config *NetworkWatchdogConfig) *DefaultNetworkWatchdog {
	if config == nil {
		config = DefaultNetworkWatchdogConfig()
	}

	return &DefaultNetworkWatchdog{
		config:       config,
		observers:    make([]NetworkStateObserver, 0),
		currentState: NetworkStateUnknown,
		stats: &NetworkWatchdogStats{
			CurrentState: NetworkStateUnknown,
		},
	}
}

// Start begins network monitoring
func (w *DefaultNetworkWatchdog) Start(ctx context.Context) error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return nil
	}
	w.running = true
	w.stopCh = make(chan struct{})
	w.stats.StartTime = time.Now()
	w.mu.Unlock()

	log.Printf("[NetworkWatchdog] Starting network monitoring (interval: %v)", w.config.CheckInterval)

	go w.monitorLoop(ctx)
	return nil
}

// Stop gracefully stops monitoring
func (w *DefaultNetworkWatchdog) Stop(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.running {
		return nil
	}

	w.running = false
	close(w.stopCh)
	log.Printf("[NetworkWatchdog] Stopped network monitoring")
	return nil
}

// RegisterObserver adds an observer to receive network state changes
func (w *DefaultNetworkWatchdog) RegisterObserver(observer NetworkStateObserver) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.observers = append(w.observers, observer)
	log.Printf("[NetworkWatchdog] Registered observer (total: %d)", len(w.observers))
}

// IsOnline returns current network status
func (w *DefaultNetworkWatchdog) IsOnline() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.currentState == NetworkStateOnline || w.currentState == NetworkStateRestored
}

// GetStats returns watchdog statistics
func (w *DefaultNetworkWatchdog) GetStats() *NetworkWatchdogStats {
	w.mu.RLock()
	defer w.mu.RUnlock()

	// Return a copy to avoid race conditions
	statsCopy := *w.stats
	return &statsCopy
}

// monitorLoop is the main monitoring goroutine
func (w *DefaultNetworkWatchdog) monitorLoop(ctx context.Context) {
	ticker := time.NewTicker(w.config.CheckInterval)
	defer ticker.Stop()

	// Initial check
	w.checkConnectivity(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.checkConnectivity(ctx)
		}
	}
}

// checkConnectivity performs a connectivity check
func (w *DefaultNetworkWatchdog) checkConnectivity(ctx context.Context) {
	w.mu.Lock()
	w.stats.CheckCount++
	w.mu.Unlock()

	online, latency := w.performCheck(ctx)

	w.mu.Lock()
	defer w.mu.Unlock()

	if online {
		w.consecutiveSuccesses++
		w.consecutiveFailures = 0
		w.stats.LastOnline = time.Now()

		if latency > 0 {
			// Simple moving average for latency
			if w.stats.AvgLatency == 0 {
				w.stats.AvgLatency = latency
			} else {
				w.stats.AvgLatency = (w.stats.AvgLatency + latency) / 2
			}
		}

		// Check if we should transition to online
		if w.currentState != NetworkStateOnline && w.currentState != NetworkStateRestored {
			if w.consecutiveSuccesses >= w.config.ConsecutiveSuccessesBeforeOnline {
				w.transitionToOnline()
			}
		}
	} else {
		w.consecutiveFailures++
		w.consecutiveSuccesses = 0

		// Check if we should transition to offline
		if w.currentState == NetworkStateOnline || w.currentState == NetworkStateRestored {
			if w.consecutiveFailures >= w.config.ConsecutiveFailuresBeforeOffline {
				w.transitionToOffline()
			}
		}
	}
}

// performCheck attempts to connect to external hosts
func (w *DefaultNetworkWatchdog) performCheck(ctx context.Context) (bool, time.Duration) {
	checkCtx, cancel := context.WithTimeout(ctx, w.config.Timeout)
	defer cancel()

	for _, host := range w.config.CheckHosts {
		start := time.Now()
		conn, err := net.DialTimeout("tcp", host, w.config.Timeout)
		if err == nil {
			latency := time.Since(start)
			conn.Close()
			return true, latency
		}

		// Check if context was cancelled
		select {
		case <-checkCtx.Done():
			return false, 0
		default:
		}
	}

	return false, 0
}

// transitionToOnline handles the transition to online state
func (w *DefaultNetworkWatchdog) transitionToOnline() {
	wasOffline := w.currentState == NetworkStateOffline

	if wasOffline {
		w.currentState = NetworkStateRestored
		outageDuration := time.Since(w.outageStart)
		w.stats.TotalOutageDuration += outageDuration
		if outageDuration > w.stats.LongestOutage {
			w.stats.LongestOutage = outageDuration
		}
		log.Printf("[NetworkWatchdog] 🌐 Network RESTORED after %v outage", outageDuration)
	} else {
		w.currentState = NetworkStateOnline
		log.Printf("[NetworkWatchdog] 🌐 Network is ONLINE")
	}

	w.stats.CurrentState = w.currentState
	w.stats.LastStateChange = time.Now()

	// Notify observers (unlock first to avoid deadlock)
	observers := make([]NetworkStateObserver, len(w.observers))
	copy(observers, w.observers)

	// Unlock before notifying to avoid deadlock
	w.mu.Unlock()
	for _, observer := range observers {
		go observer.OnNetworkRestored(context.Background())
	}
	w.mu.Lock()

	// Transition restored -> online after notifications
	if w.currentState == NetworkStateRestored {
		w.currentState = NetworkStateOnline
		w.stats.CurrentState = NetworkStateOnline
	}
}

// transitionToOffline handles the transition to offline state
func (w *DefaultNetworkWatchdog) transitionToOffline() {
	w.currentState = NetworkStateOffline
	w.stats.CurrentState = NetworkStateOffline
	w.stats.LastStateChange = time.Now()
	w.stats.TotalOutages++
	w.stats.LastOffline = time.Now()
	w.outageStart = time.Now()

	log.Printf("[NetworkWatchdog] ⚠️ Network is OFFLINE (outage #%d)", w.stats.TotalOutages)

	// Notify observers
	observers := make([]NetworkStateObserver, len(w.observers))
	copy(observers, w.observers)

	w.mu.Unlock()
	for _, observer := range observers {
		go observer.OnNetworkLost(context.Background())
	}
	w.mu.Lock()
}

// =============================================================================
// SIMPLE NETWORK CHECKER IMPLEMENTATION
// =============================================================================

// SimpleNetworkChecker implements NetworkChecker with basic TCP checks
type SimpleNetworkChecker struct {
	hosts   []string
	timeout time.Duration
}

// NewSimpleNetworkChecker creates a new simple network checker
func NewSimpleNetworkChecker(hosts []string, timeout time.Duration) *SimpleNetworkChecker {
	if len(hosts) == 0 {
		hosts = []string{"8.8.8.8:53", "1.1.1.1:53"}
	}
	if timeout == 0 {
		timeout = 3 * time.Second
	}
	return &SimpleNetworkChecker{hosts: hosts, timeout: timeout}
}

// IsOnline checks if internet is available
func (c *SimpleNetworkChecker) IsOnline(ctx context.Context) bool {
	for _, host := range c.hosts {
		conn, err := net.DialTimeout("tcp", host, c.timeout)
		if err == nil {
			conn.Close()
			return true
		}
	}
	return false
}

// GetLatency returns network latency
func (c *SimpleNetworkChecker) GetLatency(ctx context.Context) (time.Duration, error) {
	for _, host := range c.hosts {
		start := time.Now()
		conn, err := net.DialTimeout("tcp", host, c.timeout)
		if err == nil {
			latency := time.Since(start)
			conn.Close()
			return latency, nil
		}
	}
	return 0, context.DeadlineExceeded
}
