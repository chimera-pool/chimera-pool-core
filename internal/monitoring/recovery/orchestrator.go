package recovery

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"sort"
	"sync"
	"time"
)

// =============================================================================
// RECOVERY ORCHESTRATOR IMPLEMENTATION
// Coordinates automatic recovery of all services after network restoration
// =============================================================================

// DefaultRecoveryOrchestrator implements RecoveryOrchestrator interface
type DefaultRecoveryOrchestrator struct {
	config       *RecoveryOrchestratorConfig
	services     map[string]*managedService
	serviceOrder []string
	status       *RecoveryStatus
	mu           sync.RWMutex
	running      bool
	stopCh       chan struct{}
	alertSender  RecoveryAlertSender
}

// managedService wraps a service with recovery metadata
type managedService struct {
	service  ServiceRecoverable
	priority int
	stats    *ServiceRecoveryStats
	restarts []time.Time // Timestamps of restarts for hourly tracking
}

// NewRecoveryOrchestrator creates a new recovery orchestrator
func NewRecoveryOrchestrator(config *RecoveryOrchestratorConfig, alertSender RecoveryAlertSender) *DefaultRecoveryOrchestrator {
	if config == nil {
		config = DefaultRecoveryOrchestratorConfig()
	}

	return &DefaultRecoveryOrchestrator{
		config:      config,
		services:    make(map[string]*managedService),
		alertSender: alertSender,
		status: &RecoveryStatus{
			State:        RecoveryStateIdle,
			ServiceStats: make(map[string]*ServiceRecoveryStats),
		},
	}
}

// Start begins the orchestrator
func (o *DefaultRecoveryOrchestrator) Start(ctx context.Context) error {
	o.mu.Lock()
	if o.running {
		o.mu.Unlock()
		return nil
	}
	o.running = true
	o.stopCh = make(chan struct{})
	o.mu.Unlock()

	log.Printf("[RecoveryOrchestrator] Started with %d registered services", len(o.services))
	return nil
}

// Stop gracefully stops the orchestrator
func (o *DefaultRecoveryOrchestrator) Stop(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if !o.running {
		return nil
	}

	o.running = false
	close(o.stopCh)
	log.Printf("[RecoveryOrchestrator] Stopped")
	return nil
}

// RegisterService adds a service to be managed
func (o *DefaultRecoveryOrchestrator) RegisterService(service ServiceRecoverable, priority int) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	name := service.GetServiceName()
	o.services[name] = &managedService{
		service:  service,
		priority: priority,
		stats: &ServiceRecoveryStats{
			ServiceName: name,
		},
		restarts: make([]time.Time, 0),
	}

	// Rebuild service order
	o.rebuildServiceOrder()

	log.Printf("[RecoveryOrchestrator] Registered service: %s (priority: %d)", name, priority)
	return nil
}

// rebuildServiceOrder sorts services by priority
func (o *DefaultRecoveryOrchestrator) rebuildServiceOrder() {
	type serviceWithPriority struct {
		name     string
		priority int
	}

	services := make([]serviceWithPriority, 0, len(o.services))
	for name, svc := range o.services {
		services = append(services, serviceWithPriority{name, svc.priority})
	}

	sort.Slice(services, func(i, j int) bool {
		return services[i].priority < services[j].priority
	})

	o.serviceOrder = make([]string, len(services))
	for i, svc := range services {
		o.serviceOrder[i] = svc.name
	}
}

// TriggerRecovery manually triggers recovery sequence
func (o *DefaultRecoveryOrchestrator) TriggerRecovery(ctx context.Context) error {
	o.mu.Lock()
	if o.status.State == RecoveryStateInProgress {
		o.mu.Unlock()
		return fmt.Errorf("recovery already in progress")
	}

	o.status.State = RecoveryStateInProgress
	o.status.StartedAt = time.Now()
	o.status.Reason = "manual_trigger"
	o.status.ServicesRecovered = nil
	o.status.ServicesFailed = nil
	o.mu.Unlock()

	log.Printf("[RecoveryOrchestrator] 🔄 Starting recovery sequence...")

	if o.alertSender != nil {
		o.alertSender.SendRecoveryStarted(ctx, "manual_trigger")
	}

	return o.executeRecovery(ctx)
}

// OnNetworkRestored implements NetworkStateObserver - called when network comes back
func (o *DefaultRecoveryOrchestrator) OnNetworkRestored(ctx context.Context) {
	log.Printf("[RecoveryOrchestrator] 🌐 Network restored - initiating recovery")

	// Reset restart counters if configured
	if o.config.ResetCountersOnNetworkRestore {
		o.ResetRestartCounters()
	}

	// Trigger auto-recovery if enabled
	if o.config.EnableAutoRecovery {
		o.mu.Lock()
		o.status.Reason = "network_restored"
		o.mu.Unlock()

		go func() {
			if err := o.TriggerRecovery(context.Background()); err != nil {
				log.Printf("[RecoveryOrchestrator] Recovery failed: %v", err)
			}
		}()
	}
}

// OnNetworkLost implements NetworkStateObserver - called when network goes down
func (o *DefaultRecoveryOrchestrator) OnNetworkLost(ctx context.Context) {
	log.Printf("[RecoveryOrchestrator] ⚠️ Network lost - services may become unhealthy")

	if o.alertSender != nil {
		o.alertSender.SendNetworkStateChange(ctx, false)
	}
}

// GetRecoveryStatus returns current recovery status
func (o *DefaultRecoveryOrchestrator) GetRecoveryStatus() *RecoveryStatus {
	o.mu.RLock()
	defer o.mu.RUnlock()

	// Return a copy
	statusCopy := *o.status
	return &statusCopy
}

// ResetRestartCounters resets all restart counters
func (o *DefaultRecoveryOrchestrator) ResetRestartCounters() {
	o.mu.Lock()
	defer o.mu.Unlock()

	for name, svc := range o.services {
		svc.stats.RestartsThisHour = 0
		svc.stats.ConsecutiveFails = 0
		svc.restarts = make([]time.Time, 0)
		log.Printf("[RecoveryOrchestrator] Reset restart counter for %s", name)
	}

	log.Printf("[RecoveryOrchestrator] ✅ All restart counters reset - ready for recovery")
}

// executeRecovery performs the actual recovery sequence
func (o *DefaultRecoveryOrchestrator) executeRecovery(ctx context.Context) error {
	startTime := time.Now()
	recovered := make([]string, 0)
	failed := make([]string, 0)

	for _, serviceName := range o.serviceOrder {
		o.mu.RLock()
		managed, exists := o.services[serviceName]
		o.mu.RUnlock()

		if !exists {
			continue
		}

		// Check if service is already healthy
		if managed.service.IsHealthy(ctx) {
			log.Printf("[RecoveryOrchestrator] ✅ %s is already healthy", serviceName)
			managed.stats.CurrentlyHealthy = true
			managed.stats.LastHealthy = time.Now()
			recovered = append(recovered, serviceName)
			continue
		}

		// Check restart limits
		if !o.canRestart(managed) {
			log.Printf("[RecoveryOrchestrator] ⚠️ %s exceeded restart limit, skipping", serviceName)
			failed = append(failed, serviceName)
			continue
		}

		// Attempt restart with retries
		if err := o.restartServiceWithRetry(ctx, managed); err != nil {
			log.Printf("[RecoveryOrchestrator] ❌ Failed to recover %s: %v", serviceName, err)
			failed = append(failed, serviceName)
			if o.alertSender != nil {
				o.alertSender.SendRecoveryFailed(ctx, serviceName, err)
			}
		} else {
			log.Printf("[RecoveryOrchestrator] ✅ Successfully recovered %s", serviceName)
			recovered = append(recovered, serviceName)
		}
	}

	duration := time.Since(startTime)

	o.mu.Lock()
	o.status.ServicesRecovered = recovered
	o.status.ServicesFailed = failed
	o.status.CompletedAt = time.Now()
	o.status.TotalRecoveries++

	if len(failed) == 0 {
		o.status.State = RecoveryStateComplete
		log.Printf("[RecoveryOrchestrator] 🎉 Recovery complete! %d services recovered in %v", len(recovered), duration)
	} else {
		o.status.State = RecoveryStateFailed
		log.Printf("[RecoveryOrchestrator] ⚠️ Recovery partially failed: %d recovered, %d failed", len(recovered), len(failed))
	}
	o.mu.Unlock()

	if o.alertSender != nil && len(failed) == 0 {
		o.alertSender.SendRecoveryComplete(ctx, recovered, duration)
	}

	return nil
}

// canRestart checks if a service can be restarted based on limits
func (o *DefaultRecoveryOrchestrator) canRestart(managed *managedService) bool {
	// Clean up old restarts (older than 1 hour)
	hourAgo := time.Now().Add(-1 * time.Hour)
	validRestarts := make([]time.Time, 0)
	for _, t := range managed.restarts {
		if t.After(hourAgo) {
			validRestarts = append(validRestarts, t)
		}
	}
	managed.restarts = validRestarts
	managed.stats.RestartsThisHour = len(validRestarts)

	return managed.stats.RestartsThisHour < o.config.MaxRestartsPerHour
}

// restartServiceWithRetry attempts to restart a service with exponential backoff
func (o *DefaultRecoveryOrchestrator) restartServiceWithRetry(ctx context.Context, managed *managedService) error {
	maxRetries := 3
	delay := o.config.BaseRetryDelay

	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("[RecoveryOrchestrator] Restarting %s (attempt %d/%d)", managed.service.GetServiceName(), attempt, maxRetries)

		// Record restart attempt
		managed.restarts = append(managed.restarts, time.Now())
		managed.stats.TotalRestarts++
		managed.stats.RestartsThisHour++
		managed.stats.LastRestart = time.Now()

		// Perform restart
		if err := managed.service.Restart(ctx); err != nil {
			log.Printf("[RecoveryOrchestrator] Restart command failed: %v", err)
			managed.stats.ConsecutiveFails++

			if attempt < maxRetries {
				log.Printf("[RecoveryOrchestrator] Waiting %v before retry...", delay)
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(delay):
				}
				delay = min(delay*2, o.config.MaxRetryDelay)
			}
			continue
		}

		// Wait for service to become healthy
		if err := o.waitForHealthy(ctx, managed); err != nil {
			log.Printf("[RecoveryOrchestrator] Service didn't become healthy: %v", err)
			managed.stats.ConsecutiveFails++

			if attempt < maxRetries {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(delay):
				}
				delay = min(delay*2, o.config.MaxRetryDelay)
			}
			continue
		}

		// Success!
		managed.stats.ConsecutiveFails = 0
		managed.stats.CurrentlyHealthy = true
		managed.stats.LastHealthy = time.Now()
		return nil
	}

	return fmt.Errorf("failed to recover after %d attempts", maxRetries)
}

// waitForHealthy waits for a service to become healthy
func (o *DefaultRecoveryOrchestrator) waitForHealthy(ctx context.Context, managed *managedService) error {
	deadline := time.Now().Add(o.config.ServiceStartupTimeout)
	checkInterval := 5 * time.Second

	for time.Now().Before(deadline) {
		if managed.service.IsHealthy(ctx) {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(checkInterval):
		}
	}

	return fmt.Errorf("timeout waiting for %s to become healthy", managed.service.GetServiceName())
}

// =============================================================================
// DOCKER SERVICE IMPLEMENTATION
// =============================================================================

// DockerService implements ServiceRecoverable for Docker containers
type DockerService struct {
	name          string
	containerName string
	healthURL     string
	healthChecker func(ctx context.Context) bool
}

// NewDockerService creates a new Docker service
func NewDockerService(name, containerName string, healthChecker func(ctx context.Context) bool) *DockerService {
	return &DockerService{
		name:          name,
		containerName: containerName,
		healthChecker: healthChecker,
	}
}

// GetServiceName returns the service name
func (s *DockerService) GetServiceName() string {
	return s.name
}

// IsHealthy checks if the container is healthy
func (s *DockerService) IsHealthy(ctx context.Context) bool {
	if s.healthChecker != nil {
		return s.healthChecker(ctx)
	}

	// Default: check if container is running
	cmd := exec.CommandContext(ctx, "docker", "inspect", "--format", "{{.State.Health.Status}}", s.containerName)
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	status := string(output)
	return status == "healthy\n" || status == "healthy"
}

// Restart restarts the Docker container
func (s *DockerService) Restart(ctx context.Context) error {
	log.Printf("[DockerService] Restarting container: %s", s.containerName)

	cmd := exec.CommandContext(ctx, "docker", "restart", s.containerName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker restart failed: %v, output: %s", err, string(output))
	}

	log.Printf("[DockerService] Container %s restarted successfully", s.containerName)
	return nil
}

// min returns the smaller of two durations
func min(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
