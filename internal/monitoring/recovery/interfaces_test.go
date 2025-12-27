package recovery

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// TDD TESTS FOR RECOVERY INTERFACES
// =============================================================================

func TestDefaultNetworkWatchdogConfig(t *testing.T) {
	config := DefaultNetworkWatchdogConfig()

	require.NotNil(t, config)
	assert.Equal(t, 5*time.Second, config.CheckInterval)
	assert.Len(t, config.CheckHosts, 3)
	assert.Contains(t, config.CheckHosts, "8.8.8.8:53")
	assert.Equal(t, 3*time.Second, config.Timeout)
	assert.Equal(t, 3, config.ConsecutiveFailuresBeforeOffline)
	assert.Equal(t, 2, config.ConsecutiveSuccessesBeforeOnline)
}

func TestDefaultRecoveryOrchestratorConfig(t *testing.T) {
	config := DefaultRecoveryOrchestratorConfig()

	require.NotNil(t, config)
	assert.Equal(t, 10, config.MaxRestartsPerHour)
	assert.Equal(t, 5*time.Second, config.BaseRetryDelay)
	assert.Equal(t, 5*time.Minute, config.MaxRetryDelay)
	assert.Equal(t, 2*time.Minute, config.ServiceStartupTimeout)
	assert.True(t, config.ResetCountersOnNetworkRestore, "CRITICAL: Must reset counters on network restore")
	assert.True(t, config.EnableAutoRecovery)
	assert.Contains(t, config.RecoveryOrder, "litecoind")
	assert.Contains(t, config.RecoveryOrder, "chimera-pool-stratum")
}

func TestRecoveryOrder(t *testing.T) {
	config := DefaultRecoveryOrchestratorConfig()

	// Verify correct startup order: infrastructure -> blockchain -> application
	expectedOrder := []string{
		"redis",
		"postgres",
		"litecoind",
		"chimera-pool-api",
		"chimera-pool-stratum",
		"chimera-pool-web",
		"nginx",
	}

	assert.Equal(t, expectedOrder, config.RecoveryOrder)
}

// =============================================================================
// INTERFACE SEGREGATION TESTS
// =============================================================================

func TestInterfaceSegregation_NetworkChecker(t *testing.T) {
	// Verify NetworkChecker is independently usable
	var _ NetworkChecker = (*mockNetworkChecker)(nil)
}

func TestInterfaceSegregation_NetworkStateObserver(t *testing.T) {
	var _ NetworkStateObserver = (*mockNetworkStateObserver)(nil)
}

func TestInterfaceSegregation_NetworkWatchdog(t *testing.T) {
	var _ NetworkWatchdog = (*mockNetworkWatchdog)(nil)
}

func TestInterfaceSegregation_ServiceChecker(t *testing.T) {
	var _ ServiceChecker = (*mockServiceChecker)(nil)
}

func TestInterfaceSegregation_ServiceRestarter(t *testing.T) {
	var _ ServiceRestarter = (*mockServiceRestarter)(nil)
}

func TestInterfaceSegregation_ServiceRecoverable(t *testing.T) {
	var _ ServiceRecoverable = (*mockServiceRecoverable)(nil)
}

func TestInterfaceSegregation_RecoveryOrchestrator(t *testing.T) {
	var _ RecoveryOrchestrator = (*mockRecoveryOrchestrator)(nil)
}

func TestInterfaceSegregation_RecoveryAlertSender(t *testing.T) {
	var _ RecoveryAlertSender = (*mockRecoveryAlertSender)(nil)
}

// =============================================================================
// DATA TYPE TESTS
// =============================================================================

func TestNetworkState_Constants(t *testing.T) {
	assert.Equal(t, NetworkState("online"), NetworkStateOnline)
	assert.Equal(t, NetworkState("offline"), NetworkStateOffline)
	assert.Equal(t, NetworkState("unknown"), NetworkStateUnknown)
	assert.Equal(t, NetworkState("restored"), NetworkStateRestored)
}

func TestRecoveryState_Constants(t *testing.T) {
	assert.Equal(t, RecoveryState("idle"), RecoveryStateIdle)
	assert.Equal(t, RecoveryState("in_progress"), RecoveryStateInProgress)
	assert.Equal(t, RecoveryState("complete"), RecoveryStateComplete)
	assert.Equal(t, RecoveryState("failed"), RecoveryStateFailed)
}

func TestNetworkWatchdogStats_Struct(t *testing.T) {
	now := time.Now()
	stats := &NetworkWatchdogStats{
		StartTime:           now.Add(-1 * time.Hour),
		CurrentState:        NetworkStateOnline,
		LastStateChange:     now.Add(-30 * time.Minute),
		TotalOutages:        2,
		TotalOutageDuration: 15 * time.Minute,
		LongestOutage:       10 * time.Minute,
		LastOnline:          now,
		CheckCount:          720,
		AvgLatency:          50 * time.Millisecond,
	}

	assert.Equal(t, NetworkStateOnline, stats.CurrentState)
	assert.Equal(t, int64(2), stats.TotalOutages)
	assert.Equal(t, int64(720), stats.CheckCount)
}

func TestServiceRecoveryStats_Struct(t *testing.T) {
	now := time.Now()
	stats := &ServiceRecoveryStats{
		ServiceName:      "litecoind",
		TotalRestarts:    5,
		RestartsThisHour: 2,
		LastRestart:      now.Add(-10 * time.Minute),
		LastHealthy:      now,
		ConsecutiveFails: 0,
		CurrentlyHealthy: true,
		AvgRestartTime:   45 * time.Second,
	}

	assert.Equal(t, "litecoind", stats.ServiceName)
	assert.True(t, stats.CurrentlyHealthy)
	assert.Equal(t, 2, stats.RestartsThisHour)
}

func TestRecoveryStatus_Struct(t *testing.T) {
	now := time.Now()
	status := &RecoveryStatus{
		State:             RecoveryStateComplete,
		StartedAt:         now.Add(-2 * time.Minute),
		CompletedAt:       now,
		Reason:            "network_restored",
		ServicesRecovered: []string{"litecoind", "chimera-pool-stratum"},
		ServicesFailed:    []string{},
		TotalRecoveries:   5,
	}

	assert.Equal(t, RecoveryStateComplete, status.State)
	assert.Len(t, status.ServicesRecovered, 2)
	assert.Empty(t, status.ServicesFailed)
}

func TestServiceConfig_Struct(t *testing.T) {
	config := ServiceConfig{
		Name:          "litecoind",
		ContainerName: "docker-litecoind-1",
		HealthURL:     "http://localhost:9332",
		Priority:      3,
		Critical:      true,
		DependsOn:     []string{"postgres"},
		StartupDelay:  10 * time.Second,
	}

	assert.Equal(t, "litecoind", config.Name)
	assert.True(t, config.Critical)
	assert.Equal(t, 3, config.Priority)
}

// =============================================================================
// MOCK IMPLEMENTATIONS FOR INTERFACE VERIFICATION
// =============================================================================

type mockNetworkChecker struct{}

func (m *mockNetworkChecker) IsOnline(ctx context.Context) bool {
	return true
}
func (m *mockNetworkChecker) GetLatency(ctx context.Context) (time.Duration, error) {
	return 50 * time.Millisecond, nil
}

type mockNetworkStateObserver struct {
	onlineCount  int
	offlineCount int
}

func (m *mockNetworkStateObserver) OnNetworkRestored(ctx context.Context) {
	m.onlineCount++
}
func (m *mockNetworkStateObserver) OnNetworkLost(ctx context.Context) {
	m.offlineCount++
}

type mockNetworkWatchdog struct {
	running   bool
	observers []NetworkStateObserver
}

func (m *mockNetworkWatchdog) Start(ctx context.Context) error {
	m.running = true
	return nil
}
func (m *mockNetworkWatchdog) Stop(ctx context.Context) error {
	m.running = false
	return nil
}
func (m *mockNetworkWatchdog) RegisterObserver(observer NetworkStateObserver) {
	m.observers = append(m.observers, observer)
}
func (m *mockNetworkWatchdog) IsOnline() bool {
	return true
}
func (m *mockNetworkWatchdog) GetStats() *NetworkWatchdogStats {
	return &NetworkWatchdogStats{CurrentState: NetworkStateOnline}
}

type mockServiceChecker struct {
	name    string
	healthy bool
}

func (m *mockServiceChecker) IsHealthy(ctx context.Context) bool {
	return m.healthy
}
func (m *mockServiceChecker) GetServiceName() string {
	return m.name
}

type mockServiceRestarter struct {
	name         string
	restartCount int
	restartErr   error
}

func (m *mockServiceRestarter) Restart(ctx context.Context) error {
	m.restartCount++
	return m.restartErr
}
func (m *mockServiceRestarter) GetServiceName() string {
	return m.name
}

type mockServiceRecoverable struct {
	mockServiceChecker
	mockServiceRestarter
}

func (m *mockServiceRecoverable) GetServiceName() string {
	return m.mockServiceChecker.name
}

type mockRecoveryOrchestrator struct {
	services []ServiceRecoverable
	running  bool
}

func (m *mockRecoveryOrchestrator) Start(ctx context.Context) error {
	m.running = true
	return nil
}
func (m *mockRecoveryOrchestrator) Stop(ctx context.Context) error {
	m.running = false
	return nil
}
func (m *mockRecoveryOrchestrator) RegisterService(service ServiceRecoverable, priority int) error {
	m.services = append(m.services, service)
	return nil
}
func (m *mockRecoveryOrchestrator) TriggerRecovery(ctx context.Context) error {
	return nil
}
func (m *mockRecoveryOrchestrator) GetRecoveryStatus() *RecoveryStatus {
	return &RecoveryStatus{State: RecoveryStateIdle}
}
func (m *mockRecoveryOrchestrator) ResetRestartCounters() {}

type mockRecoveryAlertSender struct {
	alerts []string
}

func (m *mockRecoveryAlertSender) SendRecoveryStarted(ctx context.Context, reason string) error {
	m.alerts = append(m.alerts, "started:"+reason)
	return nil
}
func (m *mockRecoveryAlertSender) SendRecoveryComplete(ctx context.Context, services []string, duration time.Duration) error {
	m.alerts = append(m.alerts, "complete")
	return nil
}
func (m *mockRecoveryAlertSender) SendRecoveryFailed(ctx context.Context, service string, err error) error {
	m.alerts = append(m.alerts, "failed:"+service)
	return nil
}
func (m *mockRecoveryAlertSender) SendNetworkStateChange(ctx context.Context, online bool) error {
	if online {
		m.alerts = append(m.alerts, "network:online")
	} else {
		m.alerts = append(m.alerts, "network:offline")
	}
	return nil
}
