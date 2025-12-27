package recovery

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// NETWORK WATCHDOG TDD TESTS
// =============================================================================

func TestNewNetworkWatchdog(t *testing.T) {
	t.Run("with nil config uses defaults", func(t *testing.T) {
		watchdog := NewNetworkWatchdog(nil)
		require.NotNil(t, watchdog)
		assert.Equal(t, 5*time.Second, watchdog.config.CheckInterval)
		assert.Equal(t, NetworkStateUnknown, watchdog.currentState)
	})

	t.Run("with custom config", func(t *testing.T) {
		config := &NetworkWatchdogConfig{
			CheckInterval: 10 * time.Second,
			CheckHosts:    []string{"1.1.1.1:53"},
		}
		watchdog := NewNetworkWatchdog(config)
		assert.Equal(t, 10*time.Second, watchdog.config.CheckInterval)
	})
}

func TestNetworkWatchdog_StartStop(t *testing.T) {
	watchdog := NewNetworkWatchdog(&NetworkWatchdogConfig{
		CheckInterval: 100 * time.Millisecond,
		CheckHosts:    []string{"8.8.8.8:53"},
		Timeout:       50 * time.Millisecond,
	})

	ctx := context.Background()

	// Start
	err := watchdog.Start(ctx)
	require.NoError(t, err)
	assert.True(t, watchdog.running)

	// Double start should be safe
	err = watchdog.Start(ctx)
	require.NoError(t, err)

	// Stop
	err = watchdog.Stop(ctx)
	require.NoError(t, err)
	assert.False(t, watchdog.running)

	// Double stop should be safe
	err = watchdog.Stop(ctx)
	require.NoError(t, err)
}

func TestNetworkWatchdog_RegisterObserver(t *testing.T) {
	watchdog := NewNetworkWatchdog(nil)
	observer := &testObserver{}

	watchdog.RegisterObserver(observer)
	assert.Len(t, watchdog.observers, 1)

	watchdog.RegisterObserver(observer)
	assert.Len(t, watchdog.observers, 2)
}

func TestNetworkWatchdog_GetStats(t *testing.T) {
	watchdog := NewNetworkWatchdog(nil)

	stats := watchdog.GetStats()
	require.NotNil(t, stats)
	assert.Equal(t, NetworkStateUnknown, stats.CurrentState)
	assert.Equal(t, int64(0), stats.CheckCount)
}

func TestNetworkWatchdog_ImplementsInterface(t *testing.T) {
	var _ NetworkWatchdog = (*DefaultNetworkWatchdog)(nil)
}

func TestSimpleNetworkChecker(t *testing.T) {
	t.Run("with default hosts", func(t *testing.T) {
		checker := NewSimpleNetworkChecker(nil, 0)
		assert.Len(t, checker.hosts, 2)
		assert.Equal(t, 3*time.Second, checker.timeout)
	})

	t.Run("with custom hosts", func(t *testing.T) {
		checker := NewSimpleNetworkChecker([]string{"1.1.1.1:53"}, 5*time.Second)
		assert.Len(t, checker.hosts, 1)
		assert.Equal(t, 5*time.Second, checker.timeout)
	})

	t.Run("implements NetworkChecker", func(t *testing.T) {
		var _ NetworkChecker = (*SimpleNetworkChecker)(nil)
	})
}

// =============================================================================
// ORCHESTRATOR TDD TESTS
// =============================================================================

func TestNewRecoveryOrchestrator(t *testing.T) {
	t.Run("with nil config uses defaults", func(t *testing.T) {
		orchestrator := NewRecoveryOrchestrator(nil, nil)
		require.NotNil(t, orchestrator)
		assert.Equal(t, 10, orchestrator.config.MaxRestartsPerHour)
		assert.True(t, orchestrator.config.ResetCountersOnNetworkRestore)
	})

	t.Run("with custom config", func(t *testing.T) {
		config := &RecoveryOrchestratorConfig{
			MaxRestartsPerHour: 5,
		}
		orchestrator := NewRecoveryOrchestrator(config, nil)
		assert.Equal(t, 5, orchestrator.config.MaxRestartsPerHour)
	})
}

func TestRecoveryOrchestrator_RegisterService(t *testing.T) {
	orchestrator := NewRecoveryOrchestrator(nil, nil)

	service1 := &testService{name: "service1", healthy: true}
	service2 := &testService{name: "service2", healthy: true}

	err := orchestrator.RegisterService(service1, 2)
	require.NoError(t, err)

	err = orchestrator.RegisterService(service2, 1)
	require.NoError(t, err)

	assert.Len(t, orchestrator.services, 2)

	// Verify order (lower priority first)
	assert.Equal(t, "service2", orchestrator.serviceOrder[0])
	assert.Equal(t, "service1", orchestrator.serviceOrder[1])
}

func TestRecoveryOrchestrator_ResetRestartCounters(t *testing.T) {
	orchestrator := NewRecoveryOrchestrator(nil, nil)

	service := &testService{name: "test", healthy: true}
	orchestrator.RegisterService(service, 1)

	// Simulate some restarts
	orchestrator.services["test"].stats.RestartsThisHour = 5
	orchestrator.services["test"].stats.ConsecutiveFails = 3

	// Reset
	orchestrator.ResetRestartCounters()

	assert.Equal(t, 0, orchestrator.services["test"].stats.RestartsThisHour)
	assert.Equal(t, 0, orchestrator.services["test"].stats.ConsecutiveFails)
}

func TestRecoveryOrchestrator_OnNetworkRestored(t *testing.T) {
	orchestrator := NewRecoveryOrchestrator(&RecoveryOrchestratorConfig{
		MaxRestartsPerHour:            10,
		ResetCountersOnNetworkRestore: true,
		EnableAutoRecovery:            false, // Disable for test
	}, nil)

	service := &testService{name: "test", healthy: true}
	orchestrator.RegisterService(service, 1)
	orchestrator.services["test"].stats.RestartsThisHour = 10

	// Trigger network restored
	orchestrator.OnNetworkRestored(context.Background())

	// Counters should be reset
	assert.Equal(t, 0, orchestrator.services["test"].stats.RestartsThisHour)
}

func TestRecoveryOrchestrator_GetRecoveryStatus(t *testing.T) {
	orchestrator := NewRecoveryOrchestrator(nil, nil)

	status := orchestrator.GetRecoveryStatus()
	require.NotNil(t, status)
	assert.Equal(t, RecoveryStateIdle, status.State)
}

func TestRecoveryOrchestrator_ImplementsNetworkStateObserver(t *testing.T) {
	var _ NetworkStateObserver = (*DefaultRecoveryOrchestrator)(nil)
}

func TestDockerService(t *testing.T) {
	t.Run("GetServiceName", func(t *testing.T) {
		service := NewDockerService("litecoind", "docker-litecoind-1", nil)
		assert.Equal(t, "litecoind", service.GetServiceName())
	})

	t.Run("implements ServiceRecoverable", func(t *testing.T) {
		var _ ServiceRecoverable = (*DockerService)(nil)
	})

	t.Run("custom health checker", func(t *testing.T) {
		healthCalled := false
		service := NewDockerService("test", "test-container", func(ctx context.Context) bool {
			healthCalled = true
			return true
		})

		healthy := service.IsHealthy(context.Background())
		assert.True(t, healthCalled)
		assert.True(t, healthy)
	})
}

// =============================================================================
// INTEGRATION TEST: Network Watchdog with Orchestrator
// =============================================================================

func TestNetworkWatchdog_OrchestratorIntegration(t *testing.T) {
	// Create orchestrator
	orchestrator := NewRecoveryOrchestrator(&RecoveryOrchestratorConfig{
		MaxRestartsPerHour:            10,
		ResetCountersOnNetworkRestore: true,
		EnableAutoRecovery:            false,
	}, nil)

	// Register a service with simulated restarts
	service := &testService{name: "test", healthy: true}
	orchestrator.RegisterService(service, 1)
	orchestrator.services["test"].stats.RestartsThisHour = 10

	// Create watchdog and register orchestrator as observer
	watchdog := NewNetworkWatchdog(nil)
	watchdog.RegisterObserver(orchestrator)

	assert.Len(t, watchdog.observers, 1)

	// Simulate network restored notification
	orchestrator.OnNetworkRestored(context.Background())

	// Verify counters were reset
	assert.Equal(t, 0, orchestrator.services["test"].stats.RestartsThisHour)
}

// =============================================================================
// TEST HELPERS
// =============================================================================

type testObserver struct {
	restoredCount int32
	lostCount     int32
}

func (o *testObserver) OnNetworkRestored(ctx context.Context) {
	atomic.AddInt32(&o.restoredCount, 1)
}

func (o *testObserver) OnNetworkLost(ctx context.Context) {
	atomic.AddInt32(&o.lostCount, 1)
}

type testService struct {
	name         string
	healthy      bool
	restartCount int
	restartErr   error
}

func (s *testService) GetServiceName() string {
	return s.name
}

func (s *testService) IsHealthy(ctx context.Context) bool {
	return s.healthy
}

func (s *testService) Restart(ctx context.Context) error {
	s.restartCount++
	return s.restartErr
}
