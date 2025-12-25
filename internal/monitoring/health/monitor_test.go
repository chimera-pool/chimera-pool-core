package health

import (
	"context"
	"errors"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Constructor Tests
// =============================================================================

func TestNewHealthMonitor(t *testing.T) {
	config := DefaultHealthMonitorConfig()
	recovery := NewLogRecoveryAction(nil)
	logger := log.New(os.Stdout, "[test] ", log.LstdFlags)

	monitor := NewHealthMonitor(config, recovery, logger)

	require.NotNil(t, monitor)
	assert.Equal(t, config, monitor.config)
	assert.NotNil(t, monitor.nodes)
	assert.NotNil(t, monitor.rules)
	assert.NotNil(t, monitor.stats)
	assert.False(t, monitor.running)
}

func TestNewHealthMonitor_NilConfig(t *testing.T) {
	monitor := NewHealthMonitor(nil, nil, nil)

	require.NotNil(t, monitor)
	assert.NotNil(t, monitor.config)
	assert.Equal(t, 30*time.Second, monitor.config.CheckInterval)
}

func TestNewHealthMonitor_NilLogger(t *testing.T) {
	monitor := NewHealthMonitor(nil, nil, nil)

	require.NotNil(t, monitor)
	assert.NotNil(t, monitor.logger)
}

// =============================================================================
// Node Registration Tests
// =============================================================================

func TestHealthMonitor_RegisterNode(t *testing.T) {
	monitor := NewHealthMonitor(nil, nil, nil)
	checker := &MockNodeHealthChecker{ChainName: "litecoin"}

	err := monitor.RegisterNode("litecoin", checker, "docker-litecoind-1")

	assert.NoError(t, err)
	assert.Len(t, monitor.nodes, 1)
	assert.Equal(t, 1, monitor.stats.NodesMonitored)
}

func TestHealthMonitor_RegisterNode_Duplicate(t *testing.T) {
	monitor := NewHealthMonitor(nil, nil, nil)
	checker := &MockNodeHealthChecker{ChainName: "litecoin"}

	err := monitor.RegisterNode("litecoin", checker, "docker-litecoind-1")
	require.NoError(t, err)

	err = monitor.RegisterNode("litecoin", checker, "docker-litecoind-2")

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrNodeAlreadyExists))
}

func TestHealthMonitor_UnregisterNode(t *testing.T) {
	monitor := NewHealthMonitor(nil, nil, nil)
	checker := &MockNodeHealthChecker{ChainName: "litecoin"}
	monitor.RegisterNode("litecoin", checker, "docker-litecoind-1")

	err := monitor.UnregisterNode("litecoin")

	assert.NoError(t, err)
	assert.Len(t, monitor.nodes, 0)
	assert.Equal(t, 0, monitor.stats.NodesMonitored)
}

func TestHealthMonitor_UnregisterNode_NotFound(t *testing.T) {
	monitor := NewHealthMonitor(nil, nil, nil)

	err := monitor.UnregisterNode("nonexistent")

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrNodeNotFound))
}

// =============================================================================
// Start/Stop Tests
// =============================================================================

func TestHealthMonitor_StartStop(t *testing.T) {
	config := DefaultHealthMonitorConfig()
	config.CheckInterval = 100 * time.Millisecond
	monitor := NewHealthMonitor(config, nil, nil)
	checker := &MockNodeHealthChecker{
		ChainName:    "litecoin",
		SyncProgress: 0.9999,
	}
	monitor.RegisterNode("litecoin", checker, "docker-litecoind-1")

	ctx := context.Background()
	err := monitor.Start(ctx)
	require.NoError(t, err)
	assert.True(t, monitor.IsRunning())

	// Let it run a few checks
	time.Sleep(250 * time.Millisecond)

	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = monitor.Stop(stopCtx)

	assert.NoError(t, err)
	assert.False(t, monitor.IsRunning())
	assert.Greater(t, monitor.stats.TotalChecks, int64(0))
}

func TestHealthMonitor_Start_AlreadyRunning(t *testing.T) {
	config := DefaultHealthMonitorConfig()
	config.CheckInterval = 1 * time.Second
	monitor := NewHealthMonitor(config, nil, nil)

	ctx := context.Background()
	monitor.Start(ctx)
	defer monitor.Stop(ctx)

	err := monitor.Start(ctx)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrMonitorAlreadyRunning))
}

func TestHealthMonitor_Stop_NotRunning(t *testing.T) {
	monitor := NewHealthMonitor(nil, nil, nil)
	ctx := context.Background()

	err := monitor.Stop(ctx)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrMonitorNotRunning))
}

// =============================================================================
// Health Status Tests
// =============================================================================

func TestHealthMonitor_GetHealthStatus(t *testing.T) {
	monitor := NewHealthMonitor(nil, nil, nil)
	checker := &MockNodeHealthChecker{ChainName: "litecoin"}
	monitor.RegisterNode("litecoin", checker, "docker-litecoind-1")

	ctx := context.Background()
	status, err := monitor.GetHealthStatus(ctx)

	require.NoError(t, err)
	assert.Len(t, status, 1)
	assert.Contains(t, status, "litecoin")
	assert.Equal(t, HealthStatusUnknown, status["litecoin"].Status)
}

func TestHealthMonitor_GetStats(t *testing.T) {
	monitor := NewHealthMonitor(nil, nil, nil)
	checker := &MockNodeHealthChecker{ChainName: "litecoin"}
	monitor.RegisterNode("litecoin", checker, "docker-litecoind-1")

	stats := monitor.GetStats()

	require.NotNil(t, stats)
	assert.Equal(t, 1, stats.NodesMonitored)
}

// =============================================================================
// Health Check Tests
// =============================================================================

func TestHealthMonitor_CheckNode_Healthy(t *testing.T) {
	config := DefaultHealthMonitorConfig()
	config.CheckInterval = 50 * time.Millisecond
	monitor := NewHealthMonitor(config, nil, nil)
	checker := &MockNodeHealthChecker{
		ChainName:    "litecoin",
		SyncProgress: 0.9999,
		IBD:          false,
	}
	monitor.RegisterNode("litecoin", checker, "docker-litecoind-1")

	ctx := context.Background()
	monitor.Start(ctx)
	time.Sleep(100 * time.Millisecond)
	monitor.Stop(ctx)

	status, _ := monitor.GetHealthStatus(ctx)
	assert.Equal(t, HealthStatusHealthy, status["litecoin"].Status)
	assert.Equal(t, 0, status["litecoin"].ConsecutiveFails)
}

func TestHealthMonitor_CheckNode_Unhealthy_RPCDown(t *testing.T) {
	config := DefaultHealthMonitorConfig()
	config.CheckInterval = 50 * time.Millisecond
	config.EnableAutoRestart = false // Disable restart for this test
	monitor := NewHealthMonitor(config, nil, nil)
	checker := &MockNodeHealthChecker{
		ChainName: "litecoin",
		RPCError:  ErrRPCUnreachable,
	}
	monitor.RegisterNode("litecoin", checker, "docker-litecoind-1")

	ctx := context.Background()
	monitor.Start(ctx)
	time.Sleep(100 * time.Millisecond)
	monitor.Stop(ctx)

	status, _ := monitor.GetHealthStatus(ctx)
	assert.Equal(t, HealthStatusUnhealthy, status["litecoin"].Status)
	assert.Greater(t, status["litecoin"].ConsecutiveFails, 0)
}

func TestHealthMonitor_CheckNode_Degraded_IBD(t *testing.T) {
	config := DefaultHealthMonitorConfig()
	config.CheckInterval = 50 * time.Millisecond
	monitor := NewHealthMonitor(config, nil, nil)
	checker := &MockNodeHealthChecker{
		ChainName:    "litecoin",
		SyncProgress: 0.75,
		IBD:          true,
	}
	monitor.RegisterNode("litecoin", checker, "docker-litecoind-1")

	ctx := context.Background()
	monitor.Start(ctx)
	time.Sleep(100 * time.Millisecond)
	monitor.Stop(ctx)

	status, _ := monitor.GetHealthStatus(ctx)
	assert.Equal(t, HealthStatusDegraded, status["litecoin"].Status)
	// IBD should not count as failure
	assert.Equal(t, 0, status["litecoin"].ConsecutiveFails)
}

// =============================================================================
// Auto-Restart Tests
// =============================================================================

func TestHealthMonitor_AutoRestart_OnConsecutiveFailures(t *testing.T) {
	config := DefaultHealthMonitorConfig()
	config.CheckInterval = 30 * time.Millisecond
	config.ConsecutiveFailuresBeforeRestart = 2
	config.EnableAutoRestart = true
	config.EnableAlerts = false

	recovery := NewLogRecoveryAction(nil)
	monitor := NewHealthMonitor(config, recovery, nil)
	checker := &MockNodeHealthChecker{
		ChainName:     "litecoin",
		TemplateError: ErrTemplateGenFailed,
	}
	monitor.RegisterNode("litecoin", checker, "docker-litecoind-1")

	ctx := context.Background()
	monitor.Start(ctx)
	time.Sleep(150 * time.Millisecond)
	monitor.Stop(ctx)

	// Should have attempted restart after consecutive failures
	assert.Contains(t, recovery.RestartCalls, "docker-litecoind-1")
}

func TestHealthMonitor_AutoRestart_MaxRestartsPerHour(t *testing.T) {
	config := DefaultHealthMonitorConfig()
	config.CheckInterval = 20 * time.Millisecond
	config.ConsecutiveFailuresBeforeRestart = 1
	config.MaxRestartsPerHour = 2
	config.RestartCooldown = 10 * time.Millisecond
	config.EnableAutoRestart = true
	config.EnableAlerts = false

	recovery := NewLogRecoveryAction(nil)
	monitor := NewHealthMonitor(config, recovery, nil)
	checker := &MockNodeHealthChecker{
		ChainName:     "litecoin",
		TemplateError: ErrTemplateGenFailed,
	}
	monitor.RegisterNode("litecoin", checker, "docker-litecoind-1")

	ctx := context.Background()
	monitor.Start(ctx)
	time.Sleep(200 * time.Millisecond)
	monitor.Stop(ctx)

	// Should not exceed max restarts per hour
	assert.LessOrEqual(t, len(recovery.RestartCalls), config.MaxRestartsPerHour)
}

func TestHealthMonitor_AutoRestart_Disabled(t *testing.T) {
	config := DefaultHealthMonitorConfig()
	config.CheckInterval = 30 * time.Millisecond
	config.ConsecutiveFailuresBeforeRestart = 1
	config.EnableAutoRestart = false

	recovery := NewLogRecoveryAction(nil)
	monitor := NewHealthMonitor(config, recovery, nil)
	checker := &MockNodeHealthChecker{
		ChainName:     "litecoin",
		TemplateError: ErrTemplateGenFailed,
	}
	monitor.RegisterNode("litecoin", checker, "docker-litecoind-1")

	ctx := context.Background()
	monitor.Start(ctx)
	time.Sleep(100 * time.Millisecond)
	monitor.Stop(ctx)

	// Should not restart when disabled
	assert.Empty(t, recovery.RestartCalls)
}

// =============================================================================
// Rule Tests
// =============================================================================

func TestHealthMonitor_AddRule(t *testing.T) {
	monitor := NewHealthMonitor(nil, nil, nil)
	rule := &MWEBFailureRule{}

	monitor.AddRule(rule)

	assert.Len(t, monitor.rules, 1)
}

func TestMWEBFailureRule_Evaluate_True(t *testing.T) {
	rule := &MWEBFailureRule{}
	diag := &NodeDiagnostics{
		ChainSpecificErrors: []string{"MWEB_FAILURE"},
	}

	result := rule.Evaluate(context.Background(), diag)

	assert.True(t, result)
}

func TestMWEBFailureRule_Evaluate_False(t *testing.T) {
	rule := &MWEBFailureRule{}
	diag := &NodeDiagnostics{
		ChainSpecificErrors: []string{},
	}

	result := rule.Evaluate(context.Background(), diag)

	assert.False(t, result)
}

func TestMWEBFailureRule_Evaluate_NilDiag(t *testing.T) {
	rule := &MWEBFailureRule{}

	result := rule.Evaluate(context.Background(), nil)

	assert.False(t, result)
}

func TestMWEBFailureRule_GetAction(t *testing.T) {
	rule := &MWEBFailureRule{}

	assert.Equal(t, RecoveryActionRestart, rule.GetAction())
}

func TestMWEBFailureRule_GetDescription(t *testing.T) {
	rule := &MWEBFailureRule{}

	assert.Contains(t, rule.GetDescription(), "MWEB")
}

func TestBlockTemplateFailureRule_Evaluate(t *testing.T) {
	rule := &BlockTemplateFailureRule{}

	testCases := []struct {
		name     string
		diag     *NodeDiagnostics
		expected bool
	}{
		{
			name:     "template_ok",
			diag:     &NodeDiagnostics{BlockTemplateOK: true},
			expected: false,
		},
		{
			name:     "template_fail",
			diag:     &NodeDiagnostics{BlockTemplateOK: false, IsIBD: false},
			expected: true,
		},
		{
			name:     "template_fail_but_ibd",
			diag:     &NodeDiagnostics{BlockTemplateOK: false, IsIBD: true},
			expected: false,
		},
		{
			name:     "nil_diag",
			diag:     nil,
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := rule.Evaluate(context.Background(), tc.diag)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestRPCDownRule_Evaluate(t *testing.T) {
	rule := &RPCDownRule{}

	testCases := []struct {
		name     string
		diag     *NodeDiagnostics
		expected bool
	}{
		{
			name:     "rpc_connected",
			diag:     &NodeDiagnostics{RPCConnected: true},
			expected: false,
		},
		{
			name:     "rpc_down",
			diag:     &NodeDiagnostics{RPCConnected: false},
			expected: true,
		},
		{
			name:     "nil_diag",
			diag:     nil,
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := rule.Evaluate(context.Background(), tc.diag)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestHighLatencyRule_Evaluate(t *testing.T) {
	rule := &HighLatencyRule{Threshold: 100 * time.Millisecond}

	testCases := []struct {
		name     string
		diag     *NodeDiagnostics
		expected bool
	}{
		{
			name:     "low_latency",
			diag:     &NodeDiagnostics{RPCLatency: 50 * time.Millisecond},
			expected: false,
		},
		{
			name:     "high_latency",
			diag:     &NodeDiagnostics{RPCLatency: 200 * time.Millisecond},
			expected: true,
		},
		{
			name:     "nil_diag",
			diag:     nil,
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := rule.Evaluate(context.Background(), tc.diag)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestHighLatencyRule_GetAction(t *testing.T) {
	rule := &HighLatencyRule{}

	assert.Equal(t, RecoveryActionAlert, rule.GetAction())
}

// =============================================================================
// ForceCheck Tests
// =============================================================================

func TestHealthMonitor_ForceCheck(t *testing.T) {
	monitor := NewHealthMonitor(nil, nil, nil)
	checker := &MockNodeHealthChecker{
		ChainName:    "litecoin",
		SyncProgress: 0.9999,
	}
	monitor.RegisterNode("litecoin", checker, "docker-litecoind-1")

	ctx := context.Background()
	diag, err := monitor.ForceCheck(ctx, "litecoin")

	require.NoError(t, err)
	require.NotNil(t, diag)
	assert.Equal(t, "litecoin", diag.ChainName)
}

func TestHealthMonitor_ForceCheck_NotFound(t *testing.T) {
	monitor := NewHealthMonitor(nil, nil, nil)
	ctx := context.Background()

	_, err := monitor.ForceCheck(ctx, "nonexistent")

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrNodeNotFound))
}

// =============================================================================
// Interface Compliance Tests
// =============================================================================

func TestHealthMonitor_ImplementsHealthMonitorService(t *testing.T) {
	var _ HealthMonitorService = (*HealthMonitor)(nil)
}

func TestMWEBFailureRule_ImplementsHealthRule(t *testing.T) {
	var _ HealthRule = (*MWEBFailureRule)(nil)
}

func TestBlockTemplateFailureRule_ImplementsHealthRule(t *testing.T) {
	var _ HealthRule = (*BlockTemplateFailureRule)(nil)
}

func TestRPCDownRule_ImplementsHealthRule(t *testing.T) {
	var _ HealthRule = (*RPCDownRule)(nil)
}

func TestHighLatencyRule_ImplementsHealthRule(t *testing.T) {
	var _ HealthRule = (*HighLatencyRule)(nil)
}

// =============================================================================
// Error Variable Tests
// =============================================================================

func TestMonitorErrorVariables(t *testing.T) {
	assert.NotNil(t, ErrMonitorAlreadyRunning)
	assert.NotNil(t, ErrMonitorNotRunning)
	assert.NotNil(t, ErrNodeAlreadyExists)
	assert.NotNil(t, ErrNodeNotFound)
}
