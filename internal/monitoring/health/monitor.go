package health

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

var (
	ErrMonitorAlreadyRunning = errors.New("health monitor is already running")
	ErrMonitorNotRunning     = errors.New("health monitor is not running")
	ErrNodeAlreadyExists     = errors.New("node already registered")
	ErrNodeNotFound          = errors.New("node not found")
)

// registeredNode holds information about a monitored node.
type registeredNode struct {
	name          string
	containerName string
	checker       NodeHealthChecker
	health        *NodeHealth
}

// HealthMonitor orchestrates health checking for multiple blockchain nodes.
type HealthMonitor struct {
	config   *HealthMonitorConfig
	recovery RecoveryAction
	logger   *log.Logger

	nodes map[string]*registeredNode
	rules []HealthRule
	stats *MonitorStats

	running      bool
	stopCh       chan struct{}
	wg           sync.WaitGroup
	mu           sync.RWMutex
	hourlyResets map[string]time.Time
}

// NewHealthMonitor creates a new health monitor instance.
func NewHealthMonitor(config *HealthMonitorConfig, recovery RecoveryAction, logger *log.Logger) *HealthMonitor {
	if config == nil {
		config = DefaultHealthMonitorConfig()
	}
	if logger == nil {
		logger = log.Default()
	}

	return &HealthMonitor{
		config:       config,
		recovery:     recovery,
		logger:       logger,
		nodes:        make(map[string]*registeredNode),
		rules:        make([]HealthRule, 0),
		hourlyResets: make(map[string]time.Time),
		stats: &MonitorStats{
			CheckInterval: config.CheckInterval,
			NodeStats:     make(map[string]*NodeHealth),
		},
	}
}

// RegisterNode adds a node to be monitored.
func (m *HealthMonitor) RegisterNode(name string, checker NodeHealthChecker, containerName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.nodes[name]; exists {
		return fmt.Errorf("%w: %s", ErrNodeAlreadyExists, name)
	}

	now := time.Now()
	node := &registeredNode{
		name:          name,
		containerName: containerName,
		checker:       checker,
		health: &NodeHealth{
			Name:          name,
			ContainerName: containerName,
			Status:        HealthStatusUnknown,
			LastCheck:     time.Time{},
			LastHealthy:   time.Time{},
		},
	}

	m.nodes[name] = node
	m.stats.NodesMonitored = len(m.nodes)
	m.stats.NodeStats[name] = node.health
	m.hourlyResets[name] = now

	m.logger.Printf("[HealthMonitor] Registered node: %s (container: %s)", name, containerName)
	return nil
}

// UnregisterNode removes a node from monitoring.
func (m *HealthMonitor) UnregisterNode(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.nodes[name]; !exists {
		return fmt.Errorf("%w: %s", ErrNodeNotFound, name)
	}

	delete(m.nodes, name)
	delete(m.stats.NodeStats, name)
	delete(m.hourlyResets, name)
	m.stats.NodesMonitored = len(m.nodes)

	m.logger.Printf("[HealthMonitor] Unregistered node: %s", name)
	return nil
}

// Start begins health monitoring.
func (m *HealthMonitor) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return ErrMonitorAlreadyRunning
	}
	m.running = true
	m.stopCh = make(chan struct{})
	m.stats.StartTime = time.Now()
	m.mu.Unlock()

	m.logger.Printf("[HealthMonitor] Starting with interval: %v", m.config.CheckInterval)

	m.wg.Add(1)
	go m.monitorLoop(ctx)

	return nil
}

// Stop gracefully stops health monitoring.
func (m *HealthMonitor) Stop(ctx context.Context) error {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return ErrMonitorNotRunning
	}
	m.running = false
	close(m.stopCh)
	m.mu.Unlock()

	// Wait for monitoring goroutine to finish
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		m.logger.Printf("[HealthMonitor] Stopped gracefully")
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// GetHealthStatus returns the current health status of all nodes.
func (m *HealthMonitor) GetHealthStatus(ctx context.Context) (map[string]*NodeHealth, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*NodeHealth, len(m.nodes))
	for name, node := range m.nodes {
		// Return a copy to prevent race conditions
		health := *node.health
		result[name] = &health
	}
	return result, nil
}

// AddRule adds a health rule to the monitor.
func (m *HealthMonitor) AddRule(rule HealthRule) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rules = append(m.rules, rule)
	m.logger.Printf("[HealthMonitor] Added rule: %s", rule.GetDescription())
}

// GetStats returns monitoring statistics.
func (m *HealthMonitor) GetStats() *MonitorStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy
	stats := *m.stats
	stats.NodeStats = make(map[string]*NodeHealth, len(m.stats.NodeStats))
	for k, v := range m.stats.NodeStats {
		health := *v
		stats.NodeStats[k] = &health
	}
	return &stats
}

// IsRunning returns whether the monitor is currently running.
func (m *HealthMonitor) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// monitorLoop is the main monitoring loop.
func (m *HealthMonitor) monitorLoop(ctx context.Context) {
	defer m.wg.Done()

	ticker := time.NewTicker(m.config.CheckInterval)
	defer ticker.Stop()

	// Run initial check immediately
	m.checkAllNodes(ctx)

	for {
		select {
		case <-m.stopCh:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.checkAllNodes(ctx)
		}
	}
}

// checkAllNodes performs health checks on all registered nodes.
func (m *HealthMonitor) checkAllNodes(ctx context.Context) {
	m.mu.RLock()
	nodes := make([]*registeredNode, 0, len(m.nodes))
	for _, node := range m.nodes {
		nodes = append(nodes, node)
	}
	m.mu.RUnlock()

	for _, node := range nodes {
		m.checkNode(ctx, node)
	}
}

// checkNode performs a health check on a single node.
func (m *HealthMonitor) checkNode(ctx context.Context, node *registeredNode) {
	checkCtx, cancel := context.WithTimeout(ctx, m.config.RPCTimeout*2)
	defer cancel()

	// Run diagnostics
	diag, err := node.checker.RunDiagnostics(checkCtx)
	if err != nil {
		m.logger.Printf("[HealthMonitor] Error running diagnostics for %s: %v", node.name, err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	node.health.LastCheck = now
	node.health.TotalChecks++
	node.health.LastDiagnostics = diag
	m.stats.TotalChecks++

	// Reset hourly restart counter if needed
	if resetTime, ok := m.hourlyResets[node.name]; ok {
		if now.Sub(resetTime) >= time.Hour {
			node.health.RestartsThisHour = 0
			m.hourlyResets[node.name] = now
		}
	}

	// Determine health status
	if diag == nil {
		node.health.Status = HealthStatusUnhealthy
		node.health.ConsecutiveFails++
		node.health.TotalFailures++
	} else if !diag.RPCConnected {
		node.health.Status = HealthStatusUnhealthy
		node.health.ConsecutiveFails++
		node.health.TotalFailures++
	} else if !diag.BlockTemplateOK {
		node.health.Status = HealthStatusDegraded
		node.health.ConsecutiveFails++
		node.health.TotalFailures++
	} else if diag.IsIBD {
		node.health.Status = HealthStatusDegraded
		// Don't count IBD as failure
		node.health.ConsecutiveFails = 0
		node.health.LastHealthy = now
	} else {
		node.health.Status = HealthStatusHealthy
		node.health.ConsecutiveFails = 0
		node.health.LastHealthy = now
	}

	// Check if recovery action is needed
	m.evaluateRecovery(ctx, node, diag)
}

// evaluateRecovery checks if recovery action should be taken.
func (m *HealthMonitor) evaluateRecovery(ctx context.Context, node *registeredNode, diag *NodeDiagnostics) {
	// Skip if auto-restart is disabled
	if !m.config.EnableAutoRestart {
		return
	}

	// Skip if in cooldown
	if !node.health.CooldownUntil.IsZero() && time.Now().Before(node.health.CooldownUntil) {
		return
	}

	// Skip if max restarts reached
	if node.health.RestartsThisHour >= m.config.MaxRestartsPerHour {
		m.logger.Printf("[HealthMonitor] Max restarts reached for %s (%d/%d)",
			node.name, node.health.RestartsThisHour, m.config.MaxRestartsPerHour)
		return
	}

	// Check if we should restart
	shouldRestart := false
	reason := ""

	// Check consecutive failures
	if node.health.ConsecutiveFails >= m.config.ConsecutiveFailuresBeforeRestart {
		shouldRestart = true
		reason = fmt.Sprintf("%d consecutive failures", node.health.ConsecutiveFails)
	}

	// Check for MWEB errors (immediate restart)
	if diag != nil && len(diag.ChainSpecificErrors) > 0 {
		for _, err := range diag.ChainSpecificErrors {
			if err == "MWEB_FAILURE" {
				shouldRestart = true
				reason = "MWEB block validation failure"
				break
			}
		}
	}

	// Evaluate custom rules
	for _, rule := range m.rules {
		if rule.Evaluate(ctx, diag) && rule.GetAction() == RecoveryActionRestart {
			shouldRestart = true
			reason = rule.GetDescription()
			break
		}
	}

	if shouldRestart {
		m.performRestart(ctx, node, reason)
	}
}

// performRestart executes a container restart.
func (m *HealthMonitor) performRestart(ctx context.Context, node *registeredNode, reason string) {
	m.logger.Printf("[HealthMonitor] Restarting %s: %s", node.name, reason)

	// Send alert
	if m.config.EnableAlerts && m.recovery != nil {
		alert := Alert{
			Severity:    AlertSeverityCritical,
			Title:       fmt.Sprintf("Restarting %s", node.name),
			Message:     reason,
			NodeName:    node.name,
			Timestamp:   time.Now(),
			ActionTaken: "Container restart initiated",
		}
		if err := m.recovery.SendAlert(ctx, alert); err != nil {
			m.logger.Printf("[HealthMonitor] Failed to send alert: %v", err)
		}
		m.stats.TotalAlerts++
	}

	// Perform restart
	if m.recovery != nil {
		if err := m.recovery.RestartContainer(ctx, node.containerName); err != nil {
			m.logger.Printf("[HealthMonitor] Failed to restart %s: %v", node.containerName, err)
			return
		}
	}

	// Update stats
	node.health.TotalRestarts++
	node.health.RestartsThisHour++
	node.health.LastRestart = time.Now()
	node.health.CooldownUntil = time.Now().Add(m.config.RestartCooldown)
	node.health.ConsecutiveFails = 0
	m.stats.TotalRestarts++

	m.logger.Printf("[HealthMonitor] Successfully restarted %s (total restarts: %d)",
		node.name, node.health.TotalRestarts)
}

// ForceCheck forces an immediate health check on a specific node.
func (m *HealthMonitor) ForceCheck(ctx context.Context, nodeName string) (*NodeDiagnostics, error) {
	m.mu.RLock()
	node, exists := m.nodes[nodeName]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrNodeNotFound, nodeName)
	}

	checkCtx, cancel := context.WithTimeout(ctx, m.config.RPCTimeout*2)
	defer cancel()

	diag, err := node.checker.RunDiagnostics(checkCtx)
	if err != nil {
		return nil, err
	}

	// Update health status
	m.checkNode(ctx, node)

	return diag, nil
}

// Ensure HealthMonitor implements HealthMonitorService.
var _ HealthMonitorService = (*HealthMonitor)(nil)

// =============================================================================
// Built-in Health Rules
// =============================================================================

// MWEBFailureRule triggers restart on MWEB errors.
type MWEBFailureRule struct{}

func (r *MWEBFailureRule) Evaluate(ctx context.Context, diag *NodeDiagnostics) bool {
	if diag == nil {
		return false
	}
	for _, err := range diag.ChainSpecificErrors {
		if err == "MWEB_FAILURE" {
			return true
		}
	}
	return false
}

func (r *MWEBFailureRule) GetAction() RecoveryActionType {
	return RecoveryActionRestart
}

func (r *MWEBFailureRule) GetDescription() string {
	return "Restart on MWEB block validation failure"
}

var _ HealthRule = (*MWEBFailureRule)(nil)

// BlockTemplateFailureRule triggers restart on block template generation failures.
type BlockTemplateFailureRule struct {
	MinConsecutiveFails int
}

func (r *BlockTemplateFailureRule) Evaluate(ctx context.Context, diag *NodeDiagnostics) bool {
	if diag == nil {
		return false
	}
	return !diag.BlockTemplateOK && !diag.IsIBD
}

func (r *BlockTemplateFailureRule) GetAction() RecoveryActionType {
	return RecoveryActionRestart
}

func (r *BlockTemplateFailureRule) GetDescription() string {
	return "Restart on block template generation failure"
}

var _ HealthRule = (*BlockTemplateFailureRule)(nil)

// RPCDownRule triggers restart when RPC is unreachable.
type RPCDownRule struct{}

func (r *RPCDownRule) Evaluate(ctx context.Context, diag *NodeDiagnostics) bool {
	if diag == nil {
		return true
	}
	return !diag.RPCConnected
}

func (r *RPCDownRule) GetAction() RecoveryActionType {
	return RecoveryActionRestart
}

func (r *RPCDownRule) GetDescription() string {
	return "Restart when RPC is unreachable"
}

var _ HealthRule = (*RPCDownRule)(nil)

// HighLatencyRule triggers alert on high RPC latency.
type HighLatencyRule struct {
	Threshold time.Duration
}

func (r *HighLatencyRule) Evaluate(ctx context.Context, diag *NodeDiagnostics) bool {
	if diag == nil {
		return false
	}
	return diag.RPCLatency > r.Threshold
}

func (r *HighLatencyRule) GetAction() RecoveryActionType {
	return RecoveryActionAlert
}

func (r *HighLatencyRule) GetDescription() string {
	return fmt.Sprintf("Alert when RPC latency exceeds %v", r.Threshold)
}

var _ HealthRule = (*HighLatencyRule)(nil)
