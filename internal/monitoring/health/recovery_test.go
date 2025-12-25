package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// DockerRecoveryConfig Tests
// =============================================================================

func TestDefaultDockerRecoveryConfig(t *testing.T) {
	config := DefaultDockerRecoveryConfig()

	require.NotNil(t, config)
	assert.NotEmpty(t, config.DockerPath)
	assert.Equal(t, 60*time.Second, config.CommandTimeout)
}

func TestNewDockerRecoveryAction(t *testing.T) {
	config := &DockerRecoveryConfig{
		DockerPath:     "/usr/bin/docker",
		AlertWebhook:   "https://hooks.slack.com/test",
		CommandTimeout: 30 * time.Second,
	}

	action := NewDockerRecoveryAction(config)

	require.NotNil(t, action)
	assert.Equal(t, "/usr/bin/docker", action.dockerPath)
	assert.Equal(t, "https://hooks.slack.com/test", action.alertWebhook)
	assert.Equal(t, 30*time.Second, action.commandTimeout)
}

func TestNewDockerRecoveryAction_NilConfig(t *testing.T) {
	action := NewDockerRecoveryAction(nil)

	require.NotNil(t, action)
	assert.NotEmpty(t, action.dockerPath)
	assert.Equal(t, 60*time.Second, action.commandTimeout)
}

func TestNewDockerRecoveryAction_ZeroTimeout(t *testing.T) {
	config := &DockerRecoveryConfig{
		CommandTimeout: 0,
	}

	action := NewDockerRecoveryAction(config)

	assert.Equal(t, 60*time.Second, action.commandTimeout)
}

// =============================================================================
// RestartContainer Tests
// =============================================================================

func TestDockerRecoveryAction_RestartContainer_EmptyName(t *testing.T) {
	action := NewDockerRecoveryAction(nil)
	ctx := context.Background()

	err := action.RestartContainer(ctx, "")

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrContainerNotFound))
}

// Note: Full RestartContainer tests require Docker daemon access
// These are tested in integration tests

// =============================================================================
// GetContainerStatus Tests
// =============================================================================

func TestDockerRecoveryAction_GetContainerStatus_EmptyName(t *testing.T) {
	action := NewDockerRecoveryAction(nil)
	ctx := context.Background()

	status, err := action.GetContainerStatus(ctx, "")

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrContainerNotFound))
	assert.Equal(t, ContainerStatusUnknown, status)
}

// =============================================================================
// SendAlert Tests
// =============================================================================

func TestDockerRecoveryAction_SendAlert_NoWebhook(t *testing.T) {
	action := NewDockerRecoveryAction(&DockerRecoveryConfig{})
	ctx := context.Background()
	alert := Alert{
		Severity: AlertSeverityCritical,
		Title:    "Test Alert",
		Message:  "This is a test",
	}

	err := action.SendAlert(ctx, alert)

	assert.NoError(t, err) // Should succeed silently when no webhook
}

func TestDockerRecoveryAction_SendAlert_Discord(t *testing.T) {
	var receivedPayload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedPayload)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Pretend it's a Discord webhook
	webhookURL := server.URL + "/discord.com/api/webhooks/123"
	action := NewDockerRecoveryAction(&DockerRecoveryConfig{
		AlertWebhook: webhookURL,
	})
	// Override for test
	action.alertWebhook = server.URL

	ctx := context.Background()
	alert := Alert{
		Severity:    AlertSeverityCritical,
		Title:       "Node Down",
		Message:     "Litecoin node is not responding",
		NodeName:    "litecoin",
		Timestamp:   time.Now(),
		ActionTaken: "Restarted container",
	}

	// Force Discord path by modifying webhook URL
	action.alertWebhook = server.URL + "?discord.com"

	err := action.SendAlert(ctx, alert)

	assert.NoError(t, err)
	assert.NotNil(t, receivedPayload)
	assert.Contains(t, receivedPayload, "embeds")
}

func TestDockerRecoveryAction_SendAlert_Slack(t *testing.T) {
	var receivedPayload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedPayload)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	action := NewDockerRecoveryAction(&DockerRecoveryConfig{
		AlertWebhook: server.URL,
	})

	ctx := context.Background()
	alert := Alert{
		Severity:  AlertSeverityWarning,
		Title:     "High Latency",
		Message:   "RPC latency is elevated",
		NodeName:  "litecoin",
		Timestamp: time.Now(),
	}

	err := action.SendAlert(ctx, alert)

	assert.NoError(t, err)
	assert.NotNil(t, receivedPayload)
	assert.Contains(t, receivedPayload, "attachments")
}

func TestDockerRecoveryAction_SendAlert_WebhookError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	action := NewDockerRecoveryAction(&DockerRecoveryConfig{
		AlertWebhook: server.URL,
	})

	ctx := context.Background()
	alert := Alert{
		Severity: AlertSeverityInfo,
		Title:    "Test",
		Message:  "Test message",
	}

	err := action.SendAlert(ctx, alert)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrAlertFailed))
}

// =============================================================================
// LogRecoveryAction Tests
// =============================================================================

func TestNewLogRecoveryAction(t *testing.T) {
	var logged []string
	logger := func(format string, args ...interface{}) {
		logged = append(logged, format)
	}

	action := NewLogRecoveryAction(logger)

	require.NotNil(t, action)
	assert.NotNil(t, action.Logger)
	assert.Empty(t, action.RestartCalls)
	assert.Empty(t, action.AlertsSent)
}

func TestNewLogRecoveryAction_NilLogger(t *testing.T) {
	action := NewLogRecoveryAction(nil)

	require.NotNil(t, action)
	assert.NotNil(t, action.Logger)
}

func TestLogRecoveryAction_RestartContainer(t *testing.T) {
	var logged []string
	logger := func(format string, args ...interface{}) {
		logged = append(logged, format)
	}

	action := NewLogRecoveryAction(logger)
	ctx := context.Background()

	err := action.RestartContainer(ctx, "test-container")

	assert.NoError(t, err)
	assert.Contains(t, action.RestartCalls, "test-container")
	assert.Len(t, logged, 1)
}

func TestLogRecoveryAction_RestartContainer_SimulateError(t *testing.T) {
	action := NewLogRecoveryAction(nil)
	action.SimulateError = true
	ctx := context.Background()

	err := action.RestartContainer(ctx, "test-container")

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrRestartFailed))
}

func TestLogRecoveryAction_GetContainerStatus(t *testing.T) {
	action := NewLogRecoveryAction(nil)
	ctx := context.Background()

	status, err := action.GetContainerStatus(ctx, "test-container")

	assert.NoError(t, err)
	assert.Equal(t, ContainerStatusRunning, status)
}

func TestLogRecoveryAction_GetContainerStatus_SimulateError(t *testing.T) {
	action := NewLogRecoveryAction(nil)
	action.SimulateError = true
	ctx := context.Background()

	status, err := action.GetContainerStatus(ctx, "test-container")

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrContainerNotFound))
	assert.Equal(t, ContainerStatusUnknown, status)
}

func TestLogRecoveryAction_SendAlert(t *testing.T) {
	action := NewLogRecoveryAction(nil)
	ctx := context.Background()
	alert := Alert{
		Severity: AlertSeverityCritical,
		Title:    "Test Alert",
		Message:  "Test message",
		NodeName: "test-node",
	}

	err := action.SendAlert(ctx, alert)

	assert.NoError(t, err)
	assert.Len(t, action.AlertsSent, 1)
	assert.Equal(t, "Test Alert", action.AlertsSent[0].Title)
}

func TestLogRecoveryAction_SendAlert_SimulateError(t *testing.T) {
	action := NewLogRecoveryAction(nil)
	action.SimulateError = true
	ctx := context.Background()
	alert := Alert{
		Severity: AlertSeverityInfo,
		Title:    "Test",
	}

	err := action.SendAlert(ctx, alert)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrAlertFailed))
}

// =============================================================================
// TCP Check Tests
// =============================================================================

func TestCheckTCPPort_Success(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	// Extract host:port from server URL
	addr := server.Listener.Addr().String()

	err := CheckTCPPort(addr, 5*time.Second)

	assert.NoError(t, err)
}

func TestCheckTCPPort_Failure(t *testing.T) {
	// Try to connect to a port that's not listening
	err := CheckTCPPort("localhost:59999", 1*time.Second)

	assert.Error(t, err)
}

func TestCheckStratumPort(t *testing.T) {
	// Create a test server to simulate stratum port
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	// This will fail because httptest uses random port, but tests the function signature
	err := CheckStratumPort("localhost", 59998, 1*time.Second)

	assert.Error(t, err) // Expected to fail - port not open
}

// =============================================================================
// Interface Compliance Tests
// =============================================================================

func TestDockerRecoveryAction_ImplementsRecoveryAction(t *testing.T) {
	var _ RecoveryAction = (*DockerRecoveryAction)(nil)
}

func TestDockerRecoveryAction_ImplementsContainerRestarter(t *testing.T) {
	var _ ContainerRestarter = (*DockerRecoveryAction)(nil)
}

func TestDockerRecoveryAction_ImplementsAlertNotifier(t *testing.T) {
	var _ AlertNotifier = (*DockerRecoveryAction)(nil)
}

func TestLogRecoveryAction_ImplementsRecoveryAction(t *testing.T) {
	var _ RecoveryAction = (*LogRecoveryAction)(nil)
}

// =============================================================================
// Error Variable Tests
// =============================================================================

func TestErrorVariables(t *testing.T) {
	assert.NotNil(t, ErrContainerNotFound)
	assert.NotNil(t, ErrRestartFailed)
	assert.NotNil(t, ErrAlertFailed)

	assert.Contains(t, ErrContainerNotFound.Error(), "not found")
	assert.Contains(t, ErrRestartFailed.Error(), "restart failed")
	assert.Contains(t, ErrAlertFailed.Error(), "alert")
}
