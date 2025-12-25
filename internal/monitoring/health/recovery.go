package health

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

var (
	ErrContainerNotFound = errors.New("container not found")
	ErrRestartFailed     = errors.New("container restart failed")
	ErrAlertFailed       = errors.New("alert notification failed")
)

// DockerRecoveryAction implements RecoveryAction using Docker CLI.
type DockerRecoveryAction struct {
	dockerPath     string
	alertWebhook   string
	httpClient     *http.Client
	commandTimeout time.Duration
}

// DockerRecoveryConfig contains configuration for Docker recovery.
type DockerRecoveryConfig struct {
	DockerPath     string        `json:"docker_path" yaml:"docker_path"`
	AlertWebhook   string        `json:"alert_webhook" yaml:"alert_webhook"`
	CommandTimeout time.Duration `json:"command_timeout" yaml:"command_timeout"`
}

// DefaultDockerRecoveryConfig returns sensible defaults.
func DefaultDockerRecoveryConfig() *DockerRecoveryConfig {
	dockerPath := "/usr/bin/docker"
	if runtime.GOOS == "windows" {
		dockerPath = "docker.exe"
	}
	return &DockerRecoveryConfig{
		DockerPath:     dockerPath,
		CommandTimeout: 60 * time.Second,
	}
}

// NewDockerRecoveryAction creates a new Docker recovery action handler.
func NewDockerRecoveryAction(config *DockerRecoveryConfig) *DockerRecoveryAction {
	if config == nil {
		config = DefaultDockerRecoveryConfig()
	}
	if config.CommandTimeout == 0 {
		config.CommandTimeout = 60 * time.Second
	}
	return &DockerRecoveryAction{
		dockerPath:     config.DockerPath,
		alertWebhook:   config.AlertWebhook,
		commandTimeout: config.CommandTimeout,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// RestartContainer restarts a Docker container by name.
func (d *DockerRecoveryAction) RestartContainer(ctx context.Context, containerName string) error {
	if containerName == "" {
		return fmt.Errorf("%w: empty container name", ErrContainerNotFound)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, d.commandTimeout)
	defer cancel()

	// Execute docker restart command
	cmd := exec.CommandContext(ctx, d.dockerPath, "restart", containerName)
	output, err := cmd.CombinedOutput()
	outputStr := strings.TrimSpace(string(output))

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("%w: timeout restarting %s", ErrRestartFailed, containerName)
		}
		// Log the actual error for debugging
		return fmt.Errorf("%w: %s (exit error: %v, output: %q)", ErrRestartFailed, containerName, err, outputStr)
	}

	// Verify the output contains the container name (success indicator)
	if !strings.Contains(outputStr, containerName) {
		return fmt.Errorf("%w: unexpected output for %s: %q", ErrRestartFailed, containerName, outputStr)
	}

	return nil
}

// GetContainerStatus returns the current status of a Docker container.
func (d *DockerRecoveryAction) GetContainerStatus(ctx context.Context, containerName string) (ContainerStatus, error) {
	if containerName == "" {
		return ContainerStatusUnknown, fmt.Errorf("%w: empty container name", ErrContainerNotFound)
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Use docker inspect to get container state
	cmd := exec.CommandContext(ctx, d.dockerPath, "inspect", "--format", "{{.State.Status}}", containerName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "No such") {
			return ContainerStatusUnknown, fmt.Errorf("%w: %s", ErrContainerNotFound, containerName)
		}
		return ContainerStatusUnknown, err
	}

	status := strings.TrimSpace(string(output))
	switch status {
	case "running":
		return ContainerStatusRunning, nil
	case "exited", "dead":
		return ContainerStatusStopped, nil
	case "restarting":
		return ContainerStatusRestarting, nil
	default:
		return ContainerStatusUnknown, nil
	}
}

// SendAlert sends an alert notification via webhook.
func (d *DockerRecoveryAction) SendAlert(ctx context.Context, alert Alert) error {
	if d.alertWebhook == "" {
		// No webhook configured, just log
		return nil
	}

	// Determine if this is Discord or Slack based on URL
	if strings.Contains(d.alertWebhook, "discord.com") {
		return d.sendDiscordAlert(ctx, alert)
	}
	return d.sendSlackAlert(ctx, alert)
}

// sendDiscordAlert sends an alert to Discord webhook.
func (d *DockerRecoveryAction) sendDiscordAlert(ctx context.Context, alert Alert) error {
	// Discord webhook format
	color := 0x00FF00 // Green for info
	switch alert.Severity {
	case AlertSeverityWarning:
		color = 0xFFFF00 // Yellow
	case AlertSeverityCritical:
		color = 0xFF0000 // Red
	}

	payload := map[string]interface{}{
		"embeds": []map[string]interface{}{
			{
				"title":       alert.Title,
				"description": alert.Message,
				"color":       color,
				"fields": []map[string]interface{}{
					{"name": "Node", "value": alert.NodeName, "inline": true},
					{"name": "Severity", "value": string(alert.Severity), "inline": true},
					{"name": "Time", "value": alert.Timestamp.Format(time.RFC3339), "inline": true},
				},
			},
		},
	}

	if alert.ActionTaken != "" {
		payload["embeds"].([]map[string]interface{})[0]["fields"] = append(
			payload["embeds"].([]map[string]interface{})[0]["fields"].([]map[string]interface{}),
			map[string]interface{}{"name": "Action Taken", "value": alert.ActionTaken, "inline": false},
		)
	}

	return d.sendWebhook(ctx, payload)
}

// sendSlackAlert sends an alert to Slack webhook.
func (d *DockerRecoveryAction) sendSlackAlert(ctx context.Context, alert Alert) error {
	// Slack webhook format
	color := "good"
	switch alert.Severity {
	case AlertSeverityWarning:
		color = "warning"
	case AlertSeverityCritical:
		color = "danger"
	}

	payload := map[string]interface{}{
		"attachments": []map[string]interface{}{
			{
				"color": color,
				"title": alert.Title,
				"text":  alert.Message,
				"fields": []map[string]interface{}{
					{"title": "Node", "value": alert.NodeName, "short": true},
					{"title": "Severity", "value": string(alert.Severity), "short": true},
				},
				"ts": alert.Timestamp.Unix(),
			},
		},
	}

	if alert.ActionTaken != "" {
		payload["attachments"].([]map[string]interface{})[0]["fields"] = append(
			payload["attachments"].([]map[string]interface{})[0]["fields"].([]map[string]interface{}),
			map[string]interface{}{"title": "Action Taken", "value": alert.ActionTaken, "short": false},
		)
	}

	return d.sendWebhook(ctx, payload)
}

// sendWebhook sends a JSON payload to the configured webhook URL.
func (d *DockerRecoveryAction) sendWebhook(ctx context.Context, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal alert: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", d.alertWebhook, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrAlertFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("%w: HTTP %d", ErrAlertFailed, resp.StatusCode)
	}

	return nil
}

// Ensure DockerRecoveryAction implements RecoveryAction.
var _ RecoveryAction = (*DockerRecoveryAction)(nil)

// =============================================================================
// Log-based Recovery Action (for testing or non-Docker environments)
// =============================================================================

// LogRecoveryAction implements RecoveryAction by logging actions.
type LogRecoveryAction struct {
	Logger        func(format string, args ...interface{})
	RestartCalls  []string
	AlertsSent    []Alert
	SimulateError bool
}

// NewLogRecoveryAction creates a recovery action that just logs.
func NewLogRecoveryAction(logger func(format string, args ...interface{})) *LogRecoveryAction {
	if logger == nil {
		logger = func(format string, args ...interface{}) {}
	}
	return &LogRecoveryAction{
		Logger:       logger,
		RestartCalls: make([]string, 0),
		AlertsSent:   make([]Alert, 0),
	}
}

func (l *LogRecoveryAction) RestartContainer(ctx context.Context, containerName string) error {
	l.RestartCalls = append(l.RestartCalls, containerName)
	l.Logger("RESTART: Container %s", containerName)
	if l.SimulateError {
		return ErrRestartFailed
	}
	return nil
}

func (l *LogRecoveryAction) GetContainerStatus(ctx context.Context, containerName string) (ContainerStatus, error) {
	l.Logger("STATUS CHECK: Container %s", containerName)
	if l.SimulateError {
		return ContainerStatusUnknown, ErrContainerNotFound
	}
	return ContainerStatusRunning, nil
}

func (l *LogRecoveryAction) SendAlert(ctx context.Context, alert Alert) error {
	l.AlertsSent = append(l.AlertsSent, alert)
	l.Logger("ALERT [%s]: %s - %s", alert.Severity, alert.Title, alert.Message)
	if l.SimulateError {
		return ErrAlertFailed
	}
	return nil
}

var _ RecoveryAction = (*LogRecoveryAction)(nil)

// =============================================================================
// TCP Connection Check (for stratum/miner monitoring)
// =============================================================================

// CheckTCPPort checks if a TCP port is accepting connections.
func CheckTCPPort(address string, timeout time.Duration) error {
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return err
	}
	conn.Close()
	return nil
}

// CheckStratumPort checks if the stratum server is accepting connections.
func CheckStratumPort(host string, port int, timeout time.Duration) error {
	address := fmt.Sprintf("%s:%d", host, port)
	return CheckTCPPort(address, timeout)
}
