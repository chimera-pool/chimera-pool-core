package health

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultServiceConfig(t *testing.T) {
	config := DefaultServiceConfig()

	require.NotNil(t, config)
	assert.NotNil(t, config.MonitorConfig)
	assert.Equal(t, "http://litecoind:9332", config.LitecoinRPCURL)
	assert.Equal(t, "chimera", config.LitecoinRPCUser)
	assert.Equal(t, "docker-litecoind-1", config.LitecoinContainer)
	assert.True(t, config.PrometheusEnabled)
	assert.Equal(t, ":9091", config.PrometheusAddr)
}

func TestLoadServiceConfigFromEnv(t *testing.T) {
	// Set environment variables
	os.Setenv("LITECOIN_RPC_URL", "http://custom:9332")
	os.Setenv("LITECOIN_RPC_USER", "testuser")
	os.Setenv("LITECOIN_RPC_PASSWORD", "testpass")
	os.Setenv("LITECOIN_CONTAINER", "custom-container")
	os.Setenv("HEALTH_ALERT_WEBHOOK", "https://hooks.slack.com/test")
	os.Setenv("HEALTH_PROMETHEUS_ADDR", ":9999")
	defer func() {
		os.Unsetenv("LITECOIN_RPC_URL")
		os.Unsetenv("LITECOIN_RPC_USER")
		os.Unsetenv("LITECOIN_RPC_PASSWORD")
		os.Unsetenv("LITECOIN_CONTAINER")
		os.Unsetenv("HEALTH_ALERT_WEBHOOK")
		os.Unsetenv("HEALTH_PROMETHEUS_ADDR")
	}()

	config := LoadServiceConfigFromEnv()

	assert.Equal(t, "http://custom:9332", config.LitecoinRPCURL)
	assert.Equal(t, "testuser", config.LitecoinRPCUser)
	assert.Equal(t, "testpass", config.LitecoinRPCPassword)
	assert.Equal(t, "custom-container", config.LitecoinContainer)
	assert.Equal(t, "https://hooks.slack.com/test", config.AlertWebhook)
	assert.Equal(t, ":9999", config.PrometheusAddr)
}

func TestLoadServiceConfigFromEnv_DisablePrometheus(t *testing.T) {
	os.Setenv("HEALTH_PROMETHEUS_ENABLED", "false")
	defer os.Unsetenv("HEALTH_PROMETHEUS_ENABLED")

	config := LoadServiceConfigFromEnv()

	assert.False(t, config.PrometheusEnabled)
}

func TestLoadServiceConfigFromEnv_DisableAutoRestart(t *testing.T) {
	os.Setenv("HEALTH_AUTO_RESTART", "false")
	defer os.Unsetenv("HEALTH_AUTO_RESTART")

	config := LoadServiceConfigFromEnv()

	assert.False(t, config.MonitorConfig.EnableAutoRestart)
}

func TestNewHealthService(t *testing.T) {
	config := DefaultServiceConfig()
	config.PrometheusEnabled = false // Disable for test

	service := NewHealthService(config)

	require.NotNil(t, service)
	assert.NotNil(t, service.monitor)
	assert.NotNil(t, service.recovery)
	assert.NotNil(t, service.logger)
	assert.False(t, service.running)
}

func TestNewHealthService_NilConfig(t *testing.T) {
	// Clear env vars that might affect test
	os.Unsetenv("LITECOIN_RPC_URL")

	service := NewHealthService(nil)

	require.NotNil(t, service)
	assert.NotNil(t, service.config)
}

func TestNewHealthService_WithAutoRestartDisabled(t *testing.T) {
	config := DefaultServiceConfig()
	config.MonitorConfig.EnableAutoRestart = false
	config.PrometheusEnabled = false

	service := NewHealthService(config)

	// Should use LogRecoveryAction when auto-restart is disabled
	_, isLogRecovery := service.recovery.(*LogRecoveryAction)
	assert.True(t, isLogRecovery)
}

func TestNewHealthService_WithAutoRestartEnabled(t *testing.T) {
	config := DefaultServiceConfig()
	config.MonitorConfig.EnableAutoRestart = true
	config.PrometheusEnabled = false

	service := NewHealthService(config)

	// Should use DockerRecoveryAction when auto-restart is enabled
	_, isDockerRecovery := service.recovery.(*DockerRecoveryAction)
	assert.True(t, isDockerRecovery)
}

func TestHealthService_StartStop(t *testing.T) {
	config := DefaultServiceConfig()
	config.PrometheusEnabled = false
	config.MonitorConfig.CheckInterval = 100 * time.Millisecond
	config.MonitorConfig.EnableAutoRestart = false

	service := NewHealthService(config)

	ctx := context.Background()

	// Start service
	err := service.Start(ctx)
	assert.NoError(t, err)
	assert.True(t, service.IsRunning())

	// Give it time to run
	time.Sleep(150 * time.Millisecond)

	// Stop service
	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = service.Stop(stopCtx)

	assert.NoError(t, err)
	assert.False(t, service.IsRunning())
}

func TestHealthService_StartTwice(t *testing.T) {
	config := DefaultServiceConfig()
	config.PrometheusEnabled = false
	config.MonitorConfig.CheckInterval = 1 * time.Second

	service := NewHealthService(config)
	ctx := context.Background()

	service.Start(ctx)
	defer service.Stop(ctx)

	// Try to start again
	err := service.Start(ctx)

	assert.Error(t, err)
}

func TestHealthService_StopNotRunning(t *testing.T) {
	config := DefaultServiceConfig()
	config.PrometheusEnabled = false

	service := NewHealthService(config)
	ctx := context.Background()

	err := service.Stop(ctx)

	assert.Error(t, err)
}

func TestHealthService_GetMonitor(t *testing.T) {
	config := DefaultServiceConfig()
	config.PrometheusEnabled = false

	service := NewHealthService(config)

	monitor := service.GetMonitor()

	assert.NotNil(t, monitor)
	assert.Equal(t, service.monitor, monitor)
}

func TestHealthService_GetHealthStatus(t *testing.T) {
	config := DefaultServiceConfig()
	config.PrometheusEnabled = false
	config.MonitorConfig.EnableAutoRestart = false
	config.MonitorConfig.CheckInterval = 1 * time.Second

	service := NewHealthService(config)
	ctx := context.Background()

	// Start service to register nodes
	err := service.Start(ctx)
	require.NoError(t, err)
	defer service.Stop(ctx)

	status, err := service.GetHealthStatus(ctx)

	require.NoError(t, err)
	assert.NotNil(t, status)
	// Should have litecoin registered after Start()
	assert.Contains(t, status, "litecoin")
}

func TestHealthService_GetStats(t *testing.T) {
	config := DefaultServiceConfig()
	config.PrometheusEnabled = false

	service := NewHealthService(config)

	stats := service.GetStats()

	require.NotNil(t, stats)
}

func TestHealthService_RegisterBlockDAGNode_NoContainer(t *testing.T) {
	config := DefaultServiceConfig()
	config.PrometheusEnabled = false
	config.BlockDAGContainer = "" // No container configured

	service := NewHealthService(config)

	err := service.RegisterBlockDAGNode()

	// Should succeed silently when no container configured
	assert.NoError(t, err)
}

func TestHealthService_RegisterBlockDAGNode_WithContainer(t *testing.T) {
	config := DefaultServiceConfig()
	config.PrometheusEnabled = false
	config.BlockDAGContainer = "docker-blockdag-1"

	service := NewHealthService(config)

	err := service.RegisterBlockDAGNode()

	assert.NoError(t, err)

	status, _ := service.GetHealthStatus(context.Background())
	assert.Contains(t, status, "blockdag")
}

func TestHealthService_WithPrometheus(t *testing.T) {
	config := DefaultServiceConfig()
	config.PrometheusEnabled = true
	config.PrometheusAddr = "127.0.0.1:0" // Random port
	config.MonitorConfig.CheckInterval = 100 * time.Millisecond

	service := NewHealthService(config)
	assert.NotNil(t, service.exporter)

	ctx := context.Background()
	err := service.Start(ctx)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	service.Stop(stopCtx)
}

// =============================================================================
// Global Health Service Tests
// =============================================================================

func TestInitGlobalHealthService(t *testing.T) {
	// Reset global state
	globalHealthService = nil
	globalHealthServiceOnce = false

	config := DefaultServiceConfig()
	config.PrometheusEnabled = false

	service := InitGlobalHealthService(config)

	assert.NotNil(t, service)
	assert.Equal(t, service, GetGlobalHealthService())
}

func TestInitGlobalHealthService_CalledTwice(t *testing.T) {
	// Reset global state
	globalHealthService = nil
	globalHealthServiceOnce = false

	config := DefaultServiceConfig()
	config.PrometheusEnabled = false

	service1 := InitGlobalHealthService(config)
	service2 := InitGlobalHealthService(config)

	// Should return same instance
	assert.Equal(t, service1, service2)
}

func TestStartStopGlobalHealthService(t *testing.T) {
	// Reset global state
	globalHealthService = nil
	globalHealthServiceOnce = false

	ctx := context.Background()

	err := StartGlobalHealthService(ctx)
	assert.NoError(t, err)

	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = StopGlobalHealthService(stopCtx)
	assert.NoError(t, err)
}

func TestStopGlobalHealthService_NilService(t *testing.T) {
	// Reset global state
	globalHealthService = nil
	globalHealthServiceOnce = false

	ctx := context.Background()
	err := StopGlobalHealthService(ctx)

	// Should not error when service is nil
	assert.NoError(t, err)
}
