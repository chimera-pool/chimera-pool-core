package health

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Full Integration Test - Health Monitor with Litecoin Node
// =============================================================================

func TestIntegration_FullHealthMonitorLifecycle(t *testing.T) {
	// Create a mock Litecoin RPC server
	ltcServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string `json:"method"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		var resp interface{}
		switch req.Method {
		case "getblockchaininfo":
			resp = map[string]interface{}{
				"result": map[string]interface{}{
					"chain":                "main",
					"blocks":               3026575,
					"headers":              3026575,
					"verificationprogress": 0.9999,
					"initialblockdownload": false,
				},
				"error": nil,
				"id":    "health-check",
			}
		case "getblockcount":
			resp = map[string]interface{}{
				"result": 3026575,
				"error":  nil,
				"id":     "health-check",
			}
		case "getblocktemplate":
			resp = map[string]interface{}{
				"result": map[string]interface{}{
					"version":           536870916,
					"previousblockhash": "abc123",
					"transactions":      []interface{}{},
					"coinbasevalue":     625000000,
					"target":            "00000000ffffffff",
					"curtime":           time.Now().Unix(),
					"bits":              "1d00ffff",
					"height":            3026576,
				},
				"error": nil,
				"id":    "health-check",
			}
		case "getmempoolinfo":
			resp = map[string]interface{}{
				"result": map[string]interface{}{
					"size":  150,
					"bytes": 45000,
				},
				"error": nil,
				"id":    "health-check",
			}
		default:
			resp = map[string]interface{}{
				"result": nil,
				"error":  nil,
				"id":     "health-check",
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ltcServer.Close()

	// Create health service with mock server
	config := &ServiceConfig{
		MonitorConfig: &HealthMonitorConfig{
			CheckInterval:                    100 * time.Millisecond,
			MaxRestartsPerHour:               3,
			RestartCooldown:                  50 * time.Millisecond,
			ConsecutiveFailuresBeforeRestart: 2,
			RPCTimeout:                       5 * time.Second,
			EnableAutoRestart:                false, // Disable for test
			EnableAlerts:                     false,
		},
		LitecoinRPCURL:      ltcServer.URL,
		LitecoinRPCUser:     "test",
		LitecoinRPCPassword: "test",
		LitecoinContainer:   "test-litecoind",
		PrometheusEnabled:   false,
	}

	service := NewHealthService(config)
	require.NotNil(t, service)

	ctx := context.Background()

	// Start service
	err := service.Start(ctx)
	require.NoError(t, err)
	assert.True(t, service.IsRunning())

	// Wait for health checks to run
	time.Sleep(300 * time.Millisecond)

	// Check health status
	status, err := service.GetHealthStatus(ctx)
	require.NoError(t, err)
	require.Contains(t, status, "litecoin")

	// Node should be healthy since mock server returns good data
	assert.Equal(t, HealthStatusHealthy, status["litecoin"].Status)
	assert.Equal(t, 0, status["litecoin"].ConsecutiveFails)

	// Check diagnostics
	diag := status["litecoin"].LastDiagnostics
	require.NotNil(t, diag)
	assert.True(t, diag.RPCConnected)
	assert.InDelta(t, 0.9999, diag.SyncProgress, 0.001)
	assert.False(t, diag.IsIBD)
	assert.True(t, diag.BlockTemplateOK)
	assert.Equal(t, int64(3026575), diag.BlockHeight)

	// Check stats
	stats := service.GetStats()
	assert.Greater(t, stats.TotalChecks, int64(0))
	assert.Equal(t, 1, stats.NodesMonitored)

	// Stop service
	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = service.Stop(stopCtx)
	assert.NoError(t, err)
	assert.False(t, service.IsRunning())
}

func TestIntegration_HealthMonitor_DetectsMWEBError(t *testing.T) {
	callCount := 0

	// Create mock server that returns MWEB error on getblocktemplate
	ltcServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string `json:"method"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		var resp interface{}
		switch req.Method {
		case "getblockchaininfo":
			resp = map[string]interface{}{
				"result": map[string]interface{}{
					"chain":                "main",
					"blocks":               3026575,
					"headers":              3026575,
					"verificationprogress": 0.9999,
					"initialblockdownload": false,
				},
				"error": nil,
				"id":    "health-check",
			}
		case "getblockcount":
			resp = map[string]interface{}{
				"result": 3026575,
				"error":  nil,
				"id":     "health-check",
			}
		case "getblocktemplate":
			callCount++
			// Return MWEB error
			resp = map[string]interface{}{
				"result": nil,
				"error": map[string]interface{}{
					"code":    -1,
					"message": "Block validation failed: bad-blk-mweb-connect-failed",
				},
				"id": "health-check",
			}
		case "getmempoolinfo":
			resp = map[string]interface{}{
				"result": map[string]interface{}{
					"size":  150,
					"bytes": 45000,
				},
				"error": nil,
				"id":    "health-check",
			}
		default:
			resp = map[string]interface{}{
				"result": nil,
				"error":  nil,
				"id":     "health-check",
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ltcServer.Close()

	config := &ServiceConfig{
		MonitorConfig: &HealthMonitorConfig{
			CheckInterval:                    50 * time.Millisecond,
			MaxRestartsPerHour:               3,
			RestartCooldown:                  10 * time.Millisecond,
			ConsecutiveFailuresBeforeRestart: 2,
			RPCTimeout:                       5 * time.Second,
			EnableAutoRestart:                false,
			EnableAlerts:                     false,
		},
		LitecoinRPCURL:      ltcServer.URL,
		LitecoinRPCUser:     "test",
		LitecoinRPCPassword: "test",
		LitecoinContainer:   "test-litecoind",
		PrometheusEnabled:   false,
	}

	service := NewHealthService(config)
	ctx := context.Background()

	err := service.Start(ctx)
	require.NoError(t, err)

	// Wait for checks to detect MWEB error
	time.Sleep(200 * time.Millisecond)

	status, _ := service.GetHealthStatus(ctx)
	require.Contains(t, status, "litecoin")

	// Should detect MWEB failure and mark node as degraded (RPC works but template fails)
	// Note: Status is "degraded" when block template fails but RPC is connected
	assert.True(t, status["litecoin"].Status == HealthStatusDegraded || status["litecoin"].Status == HealthStatusUnhealthy,
		"Expected degraded or unhealthy, got %s", status["litecoin"].Status)
	assert.Greater(t, status["litecoin"].ConsecutiveFails, 0)

	// Check that MWEB error was detected in diagnostics
	diag := status["litecoin"].LastDiagnostics
	require.NotNil(t, diag)
	assert.False(t, diag.BlockTemplateOK)
	assert.Contains(t, diag.BlockTemplateError, "mweb")

	// Should have MWEB_FAILURE in chain-specific errors
	foundMWEB := false
	for _, err := range diag.ChainSpecificErrors {
		if strings.Contains(err, "MWEB") {
			foundMWEB = true
			break
		}
	}
	assert.True(t, foundMWEB, "Should detect MWEB_FAILURE")

	service.Stop(ctx)
}

func TestIntegration_HealthMonitor_WithPrometheusMetrics(t *testing.T) {
	// Create mock Litecoin server
	ltcServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"result": map[string]interface{}{
				"chain":                "main",
				"blocks":               3026575,
				"headers":              3026575,
				"verificationprogress": 0.9999,
				"initialblockdownload": false,
			},
			"error": nil,
			"id":    1,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ltcServer.Close()

	config := &ServiceConfig{
		MonitorConfig: &HealthMonitorConfig{
			CheckInterval:     100 * time.Millisecond,
			RPCTimeout:        5 * time.Second,
			EnableAutoRestart: false,
		},
		LitecoinRPCURL:      ltcServer.URL,
		LitecoinRPCUser:     "test",
		LitecoinRPCPassword: "test",
		LitecoinContainer:   "test-litecoind",
		PrometheusEnabled:   true,
		PrometheusAddr:      "127.0.0.1:0", // Random port
	}

	service := NewHealthService(config)
	ctx := context.Background()

	err := service.Start(ctx)
	require.NoError(t, err)
	defer service.Stop(ctx)

	// Wait for a health check
	time.Sleep(150 * time.Millisecond)

	// Prometheus exporter should be running
	// We can't easily test the HTTP endpoint with random port,
	// but we can verify the service started successfully
	assert.True(t, service.IsRunning())
}

func TestIntegration_ForceCheck(t *testing.T) {
	// Create mock Litecoin server
	ltcServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string `json:"method"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		var resp interface{}
		switch req.Method {
		case "getblockchaininfo":
			resp = map[string]interface{}{
				"result": map[string]interface{}{
					"chain":                "main",
					"blocks":               3026575,
					"headers":              3026575,
					"verificationprogress": 0.9999,
					"initialblockdownload": false,
				},
				"error": nil,
				"id":    "health-check",
			}
		case "getblockcount":
			resp = map[string]interface{}{
				"result": 3026575,
				"error":  nil,
				"id":     "health-check",
			}
		case "getblocktemplate":
			resp = map[string]interface{}{
				"result": map[string]interface{}{
					"version":           536870916,
					"previousblockhash": "abc123",
					"coinbasevalue":     625000000,
					"height":            3026576,
				},
				"error": nil,
				"id":    "health-check",
			}
		case "getmempoolinfo":
			resp = map[string]interface{}{
				"result": map[string]interface{}{
					"size":  100,
					"bytes": 30000,
				},
				"error": nil,
				"id":    "health-check",
			}
		default:
			resp = map[string]interface{}{"result": nil, "error": nil, "id": "health-check"}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ltcServer.Close()

	config := &ServiceConfig{
		MonitorConfig: &HealthMonitorConfig{
			CheckInterval:     1 * time.Hour, // Long interval - we'll use ForceCheck
			RPCTimeout:        5 * time.Second,
			EnableAutoRestart: false,
		},
		LitecoinRPCURL:      ltcServer.URL,
		LitecoinRPCUser:     "test",
		LitecoinRPCPassword: "test",
		LitecoinContainer:   "test-litecoind",
		PrometheusEnabled:   false,
	}

	service := NewHealthService(config)
	ctx := context.Background()

	err := service.Start(ctx)
	require.NoError(t, err)
	defer service.Stop(ctx)

	// Force an immediate check
	diag, err := service.ForceCheck(ctx, "litecoin")
	require.NoError(t, err)
	require.NotNil(t, diag)

	assert.Equal(t, "litecoin", diag.ChainName)
	assert.True(t, diag.RPCConnected)
	assert.True(t, diag.BlockTemplateOK)
	assert.Equal(t, int64(3026575), diag.BlockHeight)
}

func TestIntegration_MultipleNodesMonitoring(t *testing.T) {
	// Create mock Litecoin server
	ltcServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string `json:"method"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		var resp interface{}
		switch req.Method {
		case "getblockchaininfo":
			resp = map[string]interface{}{
				"result": map[string]interface{}{
					"chain":                "main",
					"blocks":               3026575,
					"headers":              3026575,
					"verificationprogress": 0.9999,
					"initialblockdownload": false,
				},
				"error": nil,
				"id":    "health-check",
			}
		case "getblockcount":
			resp = map[string]interface{}{
				"result": 3026575,
				"error":  nil,
				"id":     "health-check",
			}
		case "getblocktemplate":
			resp = map[string]interface{}{
				"result": map[string]interface{}{
					"version": 536870916,
					"height":  3026576,
				},
				"error": nil,
				"id":    "health-check",
			}
		case "getmempoolinfo":
			resp = map[string]interface{}{
				"result": map[string]interface{}{"size": 100, "bytes": 30000},
				"error":  nil,
				"id":     "health-check",
			}
		default:
			resp = map[string]interface{}{"result": nil, "error": nil, "id": "health-check"}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ltcServer.Close()

	// Create mock BlockDAG server
	bdagServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string `json:"method"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		var resp interface{}
		if req.Method == "eth_syncing" {
			resp = map[string]interface{}{
				"result": false, // Not syncing
				"error":  nil,
				"id":     1,
			}
		} else {
			resp = map[string]interface{}{
				"result": "0x1234",
				"error":  nil,
				"id":     1,
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer bdagServer.Close()

	config := &ServiceConfig{
		MonitorConfig: &HealthMonitorConfig{
			CheckInterval:     100 * time.Millisecond,
			RPCTimeout:        5 * time.Second,
			EnableAutoRestart: false,
		},
		LitecoinRPCURL:      ltcServer.URL,
		LitecoinRPCUser:     "test",
		LitecoinRPCPassword: "test",
		LitecoinContainer:   "test-litecoind",
		BlockDAGRPCURL:      bdagServer.URL,
		BlockDAGContainer:   "test-blockdag",
		PrometheusEnabled:   false,
	}

	service := NewHealthService(config)
	ctx := context.Background()

	err := service.Start(ctx)
	require.NoError(t, err)
	defer service.Stop(ctx)

	time.Sleep(200 * time.Millisecond)

	// Should have both nodes registered
	stats := service.GetStats()
	assert.Equal(t, 2, stats.NodesMonitored)

	status, _ := service.GetHealthStatus(ctx)
	assert.Contains(t, status, "litecoin")
	assert.Contains(t, status, "blockdag")
}
