package health

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPrometheusExporter(t *testing.T) {
	monitor := NewHealthMonitor(nil, nil, nil)
	exporter := NewPrometheusExporter(monitor, ":9090")

	require.NotNil(t, exporter)
	assert.Equal(t, monitor, exporter.monitor)
	assert.Equal(t, ":9090", exporter.listenAddr)
}

func TestPrometheusExporter_MetricsHandler(t *testing.T) {
	config := DefaultHealthMonitorConfig()
	monitor := NewHealthMonitor(config, nil, nil)

	// Register a node with mock checker
	checker := &MockNodeHealthChecker{
		ChainName:    "litecoin",
		SyncProgress: 0.9999,
		Diagnostics: &NodeDiagnostics{
			ChainName:            "litecoin",
			Timestamp:            time.Now(),
			RPCConnected:         true,
			RPCLatency:           50 * time.Millisecond,
			SyncProgress:         0.9999,
			IsIBD:                false,
			BlockHeight:          3026575,
			BlockTemplateOK:      true,
			BlockTemplateLatency: 100 * time.Millisecond,
			Mempool: &MempoolInfo{
				Size:  500,
				Bytes: 125000,
			},
		},
	}
	monitor.RegisterNode("litecoin", checker, "docker-litecoind-1")

	exporter := NewPrometheusExporter(monitor, ":0")

	// Create test request
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	// Call handler
	exporter.metricsHandler(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()

	// Verify Prometheus metrics are present
	assert.Contains(t, body, "chimera_health_monitor_checks_total")
	assert.Contains(t, body, "chimera_health_monitor_restarts_total")
	assert.Contains(t, body, "chimera_health_monitor_alerts_total")
	assert.Contains(t, body, "chimera_health_monitor_nodes_monitored")
	assert.Contains(t, body, "chimera_node_healthy")
	assert.Contains(t, body, `node="litecoin"`)
}

func TestPrometheusExporter_MetricsHandler_WithDiagnostics(t *testing.T) {
	config := DefaultHealthMonitorConfig()
	config.CheckInterval = 50 * time.Millisecond
	monitor := NewHealthMonitor(config, nil, nil)

	checker := &MockNodeHealthChecker{
		ChainName:    "litecoin",
		SyncProgress: 0.9999,
	}
	monitor.RegisterNode("litecoin", checker, "docker-litecoind-1")

	// Start monitor briefly to populate diagnostics
	ctx := context.Background()
	monitor.Start(ctx)
	time.Sleep(100 * time.Millisecond)
	monitor.Stop(ctx)

	exporter := NewPrometheusExporter(monitor, ":0")

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	exporter.metricsHandler(w, req)

	body := w.Body.String()

	// Check for diagnostic metrics
	assert.Contains(t, body, "chimera_node_rpc_connected")
	assert.Contains(t, body, "chimera_node_sync_progress")
	assert.Contains(t, body, "chimera_node_block_template_ok")
}

func TestPrometheusExporter_HealthHandler_Healthy(t *testing.T) {
	monitor := NewHealthMonitor(nil, nil, nil)
	checker := &MockNodeHealthChecker{
		ChainName:    "litecoin",
		SyncProgress: 0.9999,
	}
	monitor.RegisterNode("litecoin", checker, "docker-litecoind-1")

	// Set node to healthy status
	monitor.mu.Lock()
	monitor.nodes["litecoin"].health.Status = HealthStatusHealthy
	monitor.mu.Unlock()

	exporter := NewPrometheusExporter(monitor, ":0")

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	exporter.healthHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"status":"healthy"`)
}

func TestPrometheusExporter_HealthHandler_Unhealthy(t *testing.T) {
	monitor := NewHealthMonitor(nil, nil, nil)
	checker := &MockNodeHealthChecker{
		ChainName: "litecoin",
		RPCError:  ErrRPCUnreachable,
	}
	monitor.RegisterNode("litecoin", checker, "docker-litecoind-1")

	// Set node to unhealthy status
	monitor.mu.Lock()
	monitor.nodes["litecoin"].health.Status = HealthStatusUnhealthy
	monitor.mu.Unlock()

	exporter := NewPrometheusExporter(monitor, ":0")

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	exporter.healthHandler(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), `"status":"unhealthy"`)
}

func TestPrometheusExporter_StartStop(t *testing.T) {
	monitor := NewHealthMonitor(nil, nil, nil)
	exporter := NewPrometheusExporter(monitor, "127.0.0.1:0")

	err := exporter.Start()
	assert.NoError(t, err)

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	err = exporter.Stop()
	assert.NoError(t, err)
}

func TestPrometheusExporter_MetricsFormat(t *testing.T) {
	monitor := NewHealthMonitor(nil, nil, nil)
	exporter := NewPrometheusExporter(monitor, ":0")

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	exporter.metricsHandler(w, req)

	// Check content type
	assert.Equal(t, "text/plain; version=0.0.4; charset=utf-8", w.Header().Get("Content-Type"))

	// Check format - each metric should have HELP and TYPE
	body := w.Body.String()
	lines := strings.Split(body, "\n")

	helpCount := 0
	typeCount := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "# HELP") {
			helpCount++
		}
		if strings.HasPrefix(line, "# TYPE") {
			typeCount++
		}
	}

	// Should have at least base metrics
	assert.Greater(t, helpCount, 0)
	assert.Equal(t, helpCount, typeCount, "Each metric should have both HELP and TYPE")
}

func TestPrometheusExporter_MultipleNodes(t *testing.T) {
	monitor := NewHealthMonitor(nil, nil, nil)

	ltcChecker := &MockNodeHealthChecker{ChainName: "litecoin"}
	bdagChecker := &MockNodeHealthChecker{ChainName: "blockdag"}

	monitor.RegisterNode("litecoin", ltcChecker, "docker-litecoind-1")
	monitor.RegisterNode("blockdag", bdagChecker, "docker-blockdag-1")

	exporter := NewPrometheusExporter(monitor, ":0")

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	exporter.metricsHandler(w, req)

	body := w.Body.String()

	// Should have metrics for both nodes
	assert.Contains(t, body, `node="litecoin"`)
	assert.Contains(t, body, `node="blockdag"`)
	assert.Contains(t, body, "chimera_health_monitor_nodes_monitored 2")
}
