package health

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Mock Implementations for Testing
// =============================================================================

// MockRPCChecker implements RPCChecker for testing.
type MockRPCChecker struct {
	ConnectivityError error
	CallCount         int
}

func (m *MockRPCChecker) CheckRPCConnectivity(ctx context.Context) error {
	m.CallCount++
	return m.ConnectivityError
}

// MockSyncChecker implements SyncChecker for testing.
type MockSyncChecker struct {
	Progress    float64
	ProgressErr error
	IBD         bool
	IBDErr      error
}

func (m *MockSyncChecker) GetSyncProgress(ctx context.Context) (float64, error) {
	return m.Progress, m.ProgressErr
}

func (m *MockSyncChecker) IsInitialBlockDownload(ctx context.Context) (bool, error) {
	return m.IBD, m.IBDErr
}

// MockBlockTemplateChecker implements BlockTemplateChecker for testing.
type MockBlockTemplateChecker struct {
	TemplateError error
	CallCount     int
}

func (m *MockBlockTemplateChecker) CheckBlockTemplateGeneration(ctx context.Context) error {
	m.CallCount++
	return m.TemplateError
}

// MockContainerRestarter implements ContainerRestarter for testing.
type MockContainerRestarter struct {
	RestartError   error
	RestartedNames []string
	Status         ContainerStatus
	StatusError    error
}

func (m *MockContainerRestarter) RestartContainer(ctx context.Context, containerName string) error {
	m.RestartedNames = append(m.RestartedNames, containerName)
	return m.RestartError
}

func (m *MockContainerRestarter) GetContainerStatus(ctx context.Context, containerName string) (ContainerStatus, error) {
	return m.Status, m.StatusError
}

// MockAlertNotifier implements AlertNotifier for testing.
type MockAlertNotifier struct {
	Alerts    []Alert
	SendError error
}

func (m *MockAlertNotifier) SendAlert(ctx context.Context, alert Alert) error {
	m.Alerts = append(m.Alerts, alert)
	return m.SendError
}

// MockNodeHealthChecker implements NodeHealthChecker for testing.
type MockNodeHealthChecker struct {
	RPCError      error
	SyncProgress  float64
	SyncError     error
	IBD           bool
	IBDError      error
	TemplateError error
	Diagnostics   *NodeDiagnostics
	DiagError     error
	ChainName     string
	RPCCallCount  int
	TemplateCount int
}

func (m *MockNodeHealthChecker) CheckRPCConnectivity(ctx context.Context) error {
	m.RPCCallCount++
	return m.RPCError
}

func (m *MockNodeHealthChecker) GetSyncProgress(ctx context.Context) (float64, error) {
	return m.SyncProgress, m.SyncError
}

func (m *MockNodeHealthChecker) IsInitialBlockDownload(ctx context.Context) (bool, error) {
	return m.IBD, m.IBDError
}

func (m *MockNodeHealthChecker) CheckBlockTemplateGeneration(ctx context.Context) error {
	m.TemplateCount++
	return m.TemplateError
}

func (m *MockNodeHealthChecker) RunDiagnostics(ctx context.Context) (*NodeDiagnostics, error) {
	if m.Diagnostics != nil {
		return m.Diagnostics, m.DiagError
	}
	return &NodeDiagnostics{
		ChainName:       m.ChainName,
		Timestamp:       time.Now(),
		RPCConnected:    m.RPCError == nil,
		SyncProgress:    m.SyncProgress,
		IsIBD:           m.IBD,
		BlockTemplateOK: m.TemplateError == nil,
	}, m.DiagError
}

func (m *MockNodeHealthChecker) GetChainName() string {
	if m.ChainName == "" {
		return "mock"
	}
	return m.ChainName
}

// =============================================================================
// Interface Tests
// =============================================================================

func TestHealthStatus_Constants(t *testing.T) {
	assert.Equal(t, HealthStatus("healthy"), HealthStatusHealthy)
	assert.Equal(t, HealthStatus("degraded"), HealthStatusDegraded)
	assert.Equal(t, HealthStatus("unhealthy"), HealthStatusUnhealthy)
	assert.Equal(t, HealthStatus("unknown"), HealthStatusUnknown)
}

func TestRecoveryActionType_Constants(t *testing.T) {
	assert.Equal(t, RecoveryActionType("none"), RecoveryActionNone)
	assert.Equal(t, RecoveryActionType("restart"), RecoveryActionRestart)
	assert.Equal(t, RecoveryActionType("alert"), RecoveryActionAlert)
	assert.Equal(t, RecoveryActionType("failover"), RecoveryActionFailover)
}

func TestContainerStatus_Constants(t *testing.T) {
	assert.Equal(t, ContainerStatus("running"), ContainerStatusRunning)
	assert.Equal(t, ContainerStatus("stopped"), ContainerStatusStopped)
	assert.Equal(t, ContainerStatus("restarting"), ContainerStatusRestarting)
	assert.Equal(t, ContainerStatus("unknown"), ContainerStatusUnknown)
}

func TestAlertSeverity_Constants(t *testing.T) {
	assert.Equal(t, AlertSeverity("info"), AlertSeverityInfo)
	assert.Equal(t, AlertSeverity("warning"), AlertSeverityWarning)
	assert.Equal(t, AlertSeverity("critical"), AlertSeverityCritical)
}

func TestDefaultHealthMonitorConfig(t *testing.T) {
	config := DefaultHealthMonitorConfig()

	require.NotNil(t, config)
	assert.Equal(t, 30*time.Second, config.CheckInterval)
	assert.Equal(t, 3, config.MaxRestartsPerHour)
	assert.Equal(t, 60*time.Second, config.RestartCooldown)
	assert.Equal(t, 3, config.ConsecutiveFailuresBeforeRestart)
	assert.Equal(t, 10*time.Second, config.RPCTimeout)
	assert.True(t, config.EnableAutoRestart)
	assert.True(t, config.EnableAlerts)
}

// =============================================================================
// Mock Implementation Tests
// =============================================================================

func TestMockRPCChecker_Success(t *testing.T) {
	mock := &MockRPCChecker{ConnectivityError: nil}
	ctx := context.Background()

	err := mock.CheckRPCConnectivity(ctx)

	assert.NoError(t, err)
	assert.Equal(t, 1, mock.CallCount)
}

func TestMockRPCChecker_Error(t *testing.T) {
	expectedErr := assert.AnError
	mock := &MockRPCChecker{ConnectivityError: expectedErr}
	ctx := context.Background()

	err := mock.CheckRPCConnectivity(ctx)

	assert.Equal(t, expectedErr, err)
	assert.Equal(t, 1, mock.CallCount)
}

func TestMockSyncChecker_GetSyncProgress(t *testing.T) {
	mock := &MockSyncChecker{Progress: 0.9999, ProgressErr: nil}
	ctx := context.Background()

	progress, err := mock.GetSyncProgress(ctx)

	assert.NoError(t, err)
	assert.Equal(t, 0.9999, progress)
}

func TestMockSyncChecker_IsInitialBlockDownload(t *testing.T) {
	mock := &MockSyncChecker{IBD: true, IBDErr: nil}
	ctx := context.Background()

	ibd, err := mock.IsInitialBlockDownload(ctx)

	assert.NoError(t, err)
	assert.True(t, ibd)
}

func TestMockBlockTemplateChecker_Success(t *testing.T) {
	mock := &MockBlockTemplateChecker{TemplateError: nil}
	ctx := context.Background()

	err := mock.CheckBlockTemplateGeneration(ctx)

	assert.NoError(t, err)
	assert.Equal(t, 1, mock.CallCount)
}

func TestMockBlockTemplateChecker_Error(t *testing.T) {
	expectedErr := assert.AnError
	mock := &MockBlockTemplateChecker{TemplateError: expectedErr}
	ctx := context.Background()

	err := mock.CheckBlockTemplateGeneration(ctx)

	assert.Equal(t, expectedErr, err)
}

func TestMockContainerRestarter_Restart(t *testing.T) {
	mock := &MockContainerRestarter{RestartError: nil}
	ctx := context.Background()

	err := mock.RestartContainer(ctx, "test-container")

	assert.NoError(t, err)
	assert.Contains(t, mock.RestartedNames, "test-container")
}

func TestMockContainerRestarter_GetStatus(t *testing.T) {
	mock := &MockContainerRestarter{Status: ContainerStatusRunning}
	ctx := context.Background()

	status, err := mock.GetContainerStatus(ctx, "test-container")

	assert.NoError(t, err)
	assert.Equal(t, ContainerStatusRunning, status)
}

func TestMockAlertNotifier_SendAlert(t *testing.T) {
	mock := &MockAlertNotifier{SendError: nil}
	ctx := context.Background()
	alert := Alert{
		Severity:  AlertSeverityCritical,
		Title:     "Test Alert",
		Message:   "This is a test",
		NodeName:  "litecoin",
		Timestamp: time.Now(),
	}

	err := mock.SendAlert(ctx, alert)

	assert.NoError(t, err)
	assert.Len(t, mock.Alerts, 1)
	assert.Equal(t, "Test Alert", mock.Alerts[0].Title)
}

func TestMockNodeHealthChecker_RunDiagnostics(t *testing.T) {
	mock := &MockNodeHealthChecker{
		ChainName:    "litecoin",
		SyncProgress: 0.9999,
		IBD:          false,
	}
	ctx := context.Background()

	diag, err := mock.RunDiagnostics(ctx)

	require.NoError(t, err)
	require.NotNil(t, diag)
	assert.Equal(t, "litecoin", diag.ChainName)
	assert.Equal(t, 0.9999, diag.SyncProgress)
	assert.False(t, diag.IsIBD)
	assert.True(t, diag.RPCConnected)
	assert.True(t, diag.BlockTemplateOK)
}

func TestMockNodeHealthChecker_GetChainName(t *testing.T) {
	mock := &MockNodeHealthChecker{ChainName: "blockdag"}

	name := mock.GetChainName()

	assert.Equal(t, "blockdag", name)
}

func TestMockNodeHealthChecker_DefaultChainName(t *testing.T) {
	mock := &MockNodeHealthChecker{}

	name := mock.GetChainName()

	assert.Equal(t, "mock", name)
}

// =============================================================================
// Data Type Tests
// =============================================================================

func TestNodeDiagnostics_Fields(t *testing.T) {
	now := time.Now()
	diag := &NodeDiagnostics{
		ChainName:            "litecoin",
		Timestamp:            now,
		RPCConnected:         true,
		RPCLatency:           50 * time.Millisecond,
		SyncProgress:         0.9999,
		IsIBD:                false,
		BlockHeight:          3026575,
		BlockTemplateOK:      true,
		BlockTemplateLatency: 100 * time.Millisecond,
		Mempool: &MempoolInfo{
			Size:  100,
			Bytes: 50000,
		},
	}

	assert.Equal(t, "litecoin", diag.ChainName)
	assert.Equal(t, now, diag.Timestamp)
	assert.True(t, diag.RPCConnected)
	assert.Equal(t, 50*time.Millisecond, diag.RPCLatency)
	assert.Equal(t, 0.9999, diag.SyncProgress)
	assert.False(t, diag.IsIBD)
	assert.Equal(t, int64(3026575), diag.BlockHeight)
	assert.True(t, diag.BlockTemplateOK)
	assert.NotNil(t, diag.Mempool)
	assert.Equal(t, 100, diag.Mempool.Size)
}

func TestNodeHealth_Fields(t *testing.T) {
	now := time.Now()
	health := &NodeHealth{
		Name:             "litecoin-main",
		ContainerName:    "docker-litecoind-1",
		Status:           HealthStatusHealthy,
		LastCheck:        now,
		LastHealthy:      now,
		ConsecutiveFails: 0,
		TotalChecks:      100,
		TotalFailures:    5,
		TotalRestarts:    1,
		RestartsThisHour: 1,
	}

	assert.Equal(t, "litecoin-main", health.Name)
	assert.Equal(t, "docker-litecoind-1", health.ContainerName)
	assert.Equal(t, HealthStatusHealthy, health.Status)
	assert.Equal(t, int64(100), health.TotalChecks)
	assert.Equal(t, int64(5), health.TotalFailures)
}

func TestAlert_Fields(t *testing.T) {
	now := time.Now()
	alert := Alert{
		Severity:    AlertSeverityCritical,
		Title:       "Node Down",
		Message:     "Litecoin node is not responding",
		NodeName:    "litecoin",
		Timestamp:   now,
		ActionTaken: "Restarted container",
	}

	assert.Equal(t, AlertSeverityCritical, alert.Severity)
	assert.Equal(t, "Node Down", alert.Title)
	assert.Equal(t, "litecoin", alert.NodeName)
	assert.Equal(t, "Restarted container", alert.ActionTaken)
}

func TestMinerStats_Fields(t *testing.T) {
	now := time.Now()
	stats := &MinerStats{
		ActiveMiners:   5,
		TotalConnected: 10,
		TotalHashrate:  50.5,
		AvgHashrate:    10.1,
		LastConnection: now,
	}

	assert.Equal(t, 5, stats.ActiveMiners)
	assert.Equal(t, 10, stats.TotalConnected)
	assert.Equal(t, 50.5, stats.TotalHashrate)
}

func TestShareStats_Fields(t *testing.T) {
	now := time.Now()
	stats := &ShareStats{
		SharesPerMinute: 120.5,
		ValidShares:     1000,
		InvalidShares:   10,
		StaleShares:     5,
		LastShare:       now,
		AcceptanceRate:  0.985,
	}

	assert.Equal(t, 120.5, stats.SharesPerMinute)
	assert.Equal(t, int64(1000), stats.ValidShares)
	assert.Equal(t, 0.985, stats.AcceptanceRate)
}

func TestMonitorStats_Fields(t *testing.T) {
	now := time.Now()
	stats := &MonitorStats{
		StartTime:      now,
		TotalChecks:    500,
		TotalRestarts:  3,
		TotalAlerts:    10,
		NodesMonitored: 2,
		CheckInterval:  30 * time.Second,
	}

	assert.Equal(t, int64(500), stats.TotalChecks)
	assert.Equal(t, int64(3), stats.TotalRestarts)
	assert.Equal(t, 2, stats.NodesMonitored)
}

// =============================================================================
// Configuration Tests
// =============================================================================

func TestLitecoinNodeConfig_Fields(t *testing.T) {
	config := LitecoinNodeConfig{
		RPCURL:      "http://localhost:9332",
		RPCUser:     "user",
		RPCPassword: "pass",
		Container:   "docker-litecoind-1",
	}

	assert.Equal(t, "http://localhost:9332", config.RPCURL)
	assert.Equal(t, "user", config.RPCUser)
	assert.Equal(t, "docker-litecoind-1", config.Container)
}

func TestBlockDAGNodeConfig_Fields(t *testing.T) {
	config := BlockDAGNodeConfig{
		RPCURL:        "https://rpc.blockdag.com",
		WalletAddress: "0x1234567890",
		Container:     "docker-blockdag-1",
	}

	assert.Equal(t, "https://rpc.blockdag.com", config.RPCURL)
	assert.Equal(t, "0x1234567890", config.WalletAddress)
	assert.Equal(t, "docker-blockdag-1", config.Container)
}

// =============================================================================
// Interface Compliance Tests
// =============================================================================

func TestMockRPCChecker_ImplementsRPCChecker(t *testing.T) {
	var _ RPCChecker = (*MockRPCChecker)(nil)
}

func TestMockSyncChecker_ImplementsSyncChecker(t *testing.T) {
	var _ SyncChecker = (*MockSyncChecker)(nil)
}

func TestMockBlockTemplateChecker_ImplementsBlockTemplateChecker(t *testing.T) {
	var _ BlockTemplateChecker = (*MockBlockTemplateChecker)(nil)
}

func TestMockContainerRestarter_ImplementsContainerRestarter(t *testing.T) {
	var _ ContainerRestarter = (*MockContainerRestarter)(nil)
}

func TestMockAlertNotifier_ImplementsAlertNotifier(t *testing.T) {
	var _ AlertNotifier = (*MockAlertNotifier)(nil)
}

func TestMockNodeHealthChecker_ImplementsNodeHealthChecker(t *testing.T) {
	var _ NodeHealthChecker = (*MockNodeHealthChecker)(nil)
}
