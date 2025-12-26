package health

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// PoolMetrics contains pool-level mining metrics for Prometheus export.
type PoolMetrics struct {
	TotalHashrate    float64 // Pool total hashrate in H/s
	WorkersOnline    int     // Number of online workers
	WorkersOffline   int     // Number of offline workers
	SharesSubmitted  int64   // Total shares submitted
	SharesAccepted   int64   // Total shares accepted
	SharesRejected   int64   // Total shares rejected
	BlocksFound      int64   // Total blocks found
	WalletBalanceLTC float64 // Wallet balance in LTC
	Difficulty       float64 // Current network difficulty
	BlockHeight      int64   // Current block height
}

// PoolMetricsProvider interface for getting pool metrics
type PoolMetricsProvider interface {
	GetPoolMetrics() *PoolMetrics
}

// PrometheusExporter exports health metrics in Prometheus format.
type PrometheusExporter struct {
	monitor      *HealthMonitor
	listenAddr   string
	server       *http.Server
	poolProvider PoolMetricsProvider
	mu           sync.RWMutex
}

// NewPrometheusExporter creates a new Prometheus metrics exporter.
func NewPrometheusExporter(monitor *HealthMonitor, listenAddr string) *PrometheusExporter {
	return &PrometheusExporter{
		monitor:    monitor,
		listenAddr: listenAddr,
	}
}

// SetPoolMetricsProvider sets the pool metrics provider for pool-level metrics.
func (p *PrometheusExporter) SetPoolMetricsProvider(provider PoolMetricsProvider) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.poolProvider = provider
}

// Start starts the Prometheus HTTP endpoint.
func (p *PrometheusExporter) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", p.metricsHandler)
	mux.HandleFunc("/health", p.healthHandler)

	p.mu.Lock()
	p.server = &http.Server{
		Addr:         p.listenAddr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	p.mu.Unlock()

	go func() {
		if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// Log error but don't crash - metrics are non-critical
		}
	}()

	return nil
}

// Stop stops the Prometheus HTTP endpoint.
func (p *PrometheusExporter) Stop() error {
	p.mu.RLock()
	server := p.server
	p.mu.RUnlock()

	if server != nil {
		return server.Close()
	}
	return nil
}

// metricsHandler handles /metrics endpoint in Prometheus format.
func (p *PrometheusExporter) metricsHandler(w http.ResponseWriter, r *http.Request) {
	stats := p.monitor.GetStats()

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

	// Monitor-level metrics
	fmt.Fprintf(w, "# HELP chimera_health_monitor_checks_total Total number of health checks performed\n")
	fmt.Fprintf(w, "# TYPE chimera_health_monitor_checks_total counter\n")
	fmt.Fprintf(w, "chimera_health_monitor_checks_total %d\n", stats.TotalChecks)

	fmt.Fprintf(w, "# HELP chimera_health_monitor_restarts_total Total number of container restarts\n")
	fmt.Fprintf(w, "# TYPE chimera_health_monitor_restarts_total counter\n")
	fmt.Fprintf(w, "chimera_health_monitor_restarts_total %d\n", stats.TotalRestarts)

	fmt.Fprintf(w, "# HELP chimera_health_monitor_alerts_total Total number of alerts sent\n")
	fmt.Fprintf(w, "# TYPE chimera_health_monitor_alerts_total counter\n")
	fmt.Fprintf(w, "chimera_health_monitor_alerts_total %d\n", stats.TotalAlerts)

	fmt.Fprintf(w, "# HELP chimera_health_monitor_nodes_monitored Number of nodes being monitored\n")
	fmt.Fprintf(w, "# TYPE chimera_health_monitor_nodes_monitored gauge\n")
	fmt.Fprintf(w, "chimera_health_monitor_nodes_monitored %d\n", stats.NodesMonitored)

	fmt.Fprintf(w, "# HELP chimera_health_monitor_uptime_seconds Seconds since monitor started\n")
	fmt.Fprintf(w, "# TYPE chimera_health_monitor_uptime_seconds gauge\n")
	fmt.Fprintf(w, "chimera_health_monitor_uptime_seconds %.0f\n", time.Since(stats.StartTime).Seconds())

	// Per-node metrics
	for name, health := range stats.NodeStats {
		labels := fmt.Sprintf(`node="%s"`, name)

		// Node health status (1 = healthy, 0 = unhealthy)
		fmt.Fprintf(w, "# HELP chimera_node_healthy Node health status (1=healthy, 0=unhealthy)\n")
		fmt.Fprintf(w, "# TYPE chimera_node_healthy gauge\n")
		healthValue := 0
		if health.Status == HealthStatusHealthy {
			healthValue = 1
		}
		fmt.Fprintf(w, "chimera_node_healthy{%s} %d\n", labels, healthValue)

		// Node status enum
		fmt.Fprintf(w, "# HELP chimera_node_status Node status (0=unknown, 1=healthy, 2=degraded, 3=unhealthy)\n")
		fmt.Fprintf(w, "# TYPE chimera_node_status gauge\n")
		statusValue := 0
		switch health.Status {
		case HealthStatusHealthy:
			statusValue = 1
		case HealthStatusDegraded:
			statusValue = 2
		case HealthStatusUnhealthy:
			statusValue = 3
		}
		fmt.Fprintf(w, "chimera_node_status{%s} %d\n", labels, statusValue)

		// Consecutive failures
		fmt.Fprintf(w, "# HELP chimera_node_consecutive_failures Current consecutive failure count\n")
		fmt.Fprintf(w, "# TYPE chimera_node_consecutive_failures gauge\n")
		fmt.Fprintf(w, "chimera_node_consecutive_failures{%s} %d\n", labels, health.ConsecutiveFails)

		// Total checks
		fmt.Fprintf(w, "# HELP chimera_node_checks_total Total health checks for this node\n")
		fmt.Fprintf(w, "# TYPE chimera_node_checks_total counter\n")
		fmt.Fprintf(w, "chimera_node_checks_total{%s} %d\n", labels, health.TotalChecks)

		// Total failures
		fmt.Fprintf(w, "# HELP chimera_node_failures_total Total failures for this node\n")
		fmt.Fprintf(w, "# TYPE chimera_node_failures_total counter\n")
		fmt.Fprintf(w, "chimera_node_failures_total{%s} %d\n", labels, health.TotalFailures)

		// Total restarts
		fmt.Fprintf(w, "# HELP chimera_node_restarts_total Total restarts for this node\n")
		fmt.Fprintf(w, "# TYPE chimera_node_restarts_total counter\n")
		fmt.Fprintf(w, "chimera_node_restarts_total{%s} %d\n", labels, health.TotalRestarts)

		// Restarts this hour
		fmt.Fprintf(w, "# HELP chimera_node_restarts_this_hour Restarts in current hour\n")
		fmt.Fprintf(w, "# TYPE chimera_node_restarts_this_hour gauge\n")
		fmt.Fprintf(w, "chimera_node_restarts_this_hour{%s} %d\n", labels, health.RestartsThisHour)

		// Last check timestamp
		fmt.Fprintf(w, "# HELP chimera_node_last_check_timestamp_seconds Unix timestamp of last health check\n")
		fmt.Fprintf(w, "# TYPE chimera_node_last_check_timestamp_seconds gauge\n")
		fmt.Fprintf(w, "chimera_node_last_check_timestamp_seconds{%s} %d\n", labels, health.LastCheck.Unix())

		// Last healthy timestamp
		if !health.LastHealthy.IsZero() {
			fmt.Fprintf(w, "# HELP chimera_node_last_healthy_timestamp_seconds Unix timestamp when node was last healthy\n")
			fmt.Fprintf(w, "# TYPE chimera_node_last_healthy_timestamp_seconds gauge\n")
			fmt.Fprintf(w, "chimera_node_last_healthy_timestamp_seconds{%s} %d\n", labels, health.LastHealthy.Unix())
		}

		// Diagnostics metrics (if available)
		if health.LastDiagnostics != nil {
			diag := health.LastDiagnostics

			// RPC connected
			fmt.Fprintf(w, "# HELP chimera_node_rpc_connected RPC connectivity status\n")
			fmt.Fprintf(w, "# TYPE chimera_node_rpc_connected gauge\n")
			rpcConnected := 0
			if diag.RPCConnected {
				rpcConnected = 1
			}
			fmt.Fprintf(w, "chimera_node_rpc_connected{%s} %d\n", labels, rpcConnected)

			// RPC latency
			fmt.Fprintf(w, "# HELP chimera_node_rpc_latency_seconds RPC latency in seconds\n")
			fmt.Fprintf(w, "# TYPE chimera_node_rpc_latency_seconds gauge\n")
			fmt.Fprintf(w, "chimera_node_rpc_latency_seconds{%s} %.6f\n", labels, diag.RPCLatency.Seconds())

			// Sync progress
			fmt.Fprintf(w, "# HELP chimera_node_sync_progress Node sync progress (0.0-1.0)\n")
			fmt.Fprintf(w, "# TYPE chimera_node_sync_progress gauge\n")
			fmt.Fprintf(w, "chimera_node_sync_progress{%s} %.6f\n", labels, diag.SyncProgress)

			// Block height
			fmt.Fprintf(w, "# HELP chimera_node_block_height Current block height\n")
			fmt.Fprintf(w, "# TYPE chimera_node_block_height gauge\n")
			fmt.Fprintf(w, "chimera_node_block_height{%s} %d\n", labels, diag.BlockHeight)

			// Block template OK
			fmt.Fprintf(w, "# HELP chimera_node_block_template_ok Block template generation status\n")
			fmt.Fprintf(w, "# TYPE chimera_node_block_template_ok gauge\n")
			templateOK := 0
			if diag.BlockTemplateOK {
				templateOK = 1
			}
			fmt.Fprintf(w, "chimera_node_block_template_ok{%s} %d\n", labels, templateOK)

			// Block template latency
			fmt.Fprintf(w, "# HELP chimera_node_block_template_latency_seconds Block template generation latency\n")
			fmt.Fprintf(w, "# TYPE chimera_node_block_template_latency_seconds gauge\n")
			fmt.Fprintf(w, "chimera_node_block_template_latency_seconds{%s} %.6f\n", labels, diag.BlockTemplateLatency.Seconds())

			// IBD status
			fmt.Fprintf(w, "# HELP chimera_node_in_ibd Node is in initial block download\n")
			fmt.Fprintf(w, "# TYPE chimera_node_in_ibd gauge\n")
			ibdValue := 0
			if diag.IsIBD {
				ibdValue = 1
			}
			fmt.Fprintf(w, "chimera_node_in_ibd{%s} %d\n", labels, ibdValue)

			// Mempool metrics
			if diag.Mempool != nil {
				fmt.Fprintf(w, "# HELP chimera_node_mempool_size Number of transactions in mempool\n")
				fmt.Fprintf(w, "# TYPE chimera_node_mempool_size gauge\n")
				fmt.Fprintf(w, "chimera_node_mempool_size{%s} %d\n", labels, diag.Mempool.Size)

				fmt.Fprintf(w, "# HELP chimera_node_mempool_bytes Mempool size in bytes\n")
				fmt.Fprintf(w, "# TYPE chimera_node_mempool_bytes gauge\n")
				fmt.Fprintf(w, "chimera_node_mempool_bytes{%s} %d\n", labels, diag.Mempool.Bytes)
			}

			// Chain-specific errors count
			fmt.Fprintf(w, "# HELP chimera_node_chain_errors_count Number of chain-specific errors\n")
			fmt.Fprintf(w, "# TYPE chimera_node_chain_errors_count gauge\n")
			fmt.Fprintf(w, "chimera_node_chain_errors_count{%s} %d\n", labels, len(diag.ChainSpecificErrors))
		}
	}

	// Pool-level metrics (if provider is set)
	p.mu.RLock()
	provider := p.poolProvider
	p.mu.RUnlock()

	if provider != nil {
		poolMetrics := provider.GetPoolMetrics()
		if poolMetrics != nil {
			// Pool hashrate
			fmt.Fprintf(w, "# HELP chimera_pool_hashrate_total Total pool hashrate in H/s\n")
			fmt.Fprintf(w, "# TYPE chimera_pool_hashrate_total gauge\n")
			fmt.Fprintf(w, "chimera_pool_hashrate_total %.2f\n", poolMetrics.TotalHashrate)

			// Workers online
			fmt.Fprintf(w, "# HELP chimera_pool_workers_online Number of online workers\n")
			fmt.Fprintf(w, "# TYPE chimera_pool_workers_online gauge\n")
			fmt.Fprintf(w, "chimera_pool_workers_online %d\n", poolMetrics.WorkersOnline)

			// Workers offline
			fmt.Fprintf(w, "# HELP chimera_pool_workers_offline Number of offline workers\n")
			fmt.Fprintf(w, "# TYPE chimera_pool_workers_offline gauge\n")
			fmt.Fprintf(w, "chimera_pool_workers_offline %d\n", poolMetrics.WorkersOffline)

			// Shares submitted
			fmt.Fprintf(w, "# HELP chimera_pool_shares_submitted_total Total shares submitted\n")
			fmt.Fprintf(w, "# TYPE chimera_pool_shares_submitted_total counter\n")
			fmt.Fprintf(w, "chimera_pool_shares_submitted_total %d\n", poolMetrics.SharesSubmitted)

			// Shares accepted
			fmt.Fprintf(w, "# HELP chimera_pool_shares_accepted_total Total shares accepted\n")
			fmt.Fprintf(w, "# TYPE chimera_pool_shares_accepted_total counter\n")
			fmt.Fprintf(w, "chimera_pool_shares_accepted_total %d\n", poolMetrics.SharesAccepted)

			// Shares rejected
			fmt.Fprintf(w, "# HELP chimera_pool_shares_rejected_total Total shares rejected\n")
			fmt.Fprintf(w, "# TYPE chimera_pool_shares_rejected_total counter\n")
			fmt.Fprintf(w, "chimera_pool_shares_rejected_total %d\n", poolMetrics.SharesRejected)

			// Blocks found
			fmt.Fprintf(w, "# HELP chimera_pool_blocks_found_total Total blocks found\n")
			fmt.Fprintf(w, "# TYPE chimera_pool_blocks_found_total counter\n")
			fmt.Fprintf(w, "chimera_pool_blocks_found_total %d\n", poolMetrics.BlocksFound)

			// Wallet balance
			fmt.Fprintf(w, "# HELP chimera_pool_payouts_wallet_balance_ltc Wallet balance in LTC\n")
			fmt.Fprintf(w, "# TYPE chimera_pool_payouts_wallet_balance_ltc gauge\n")
			fmt.Fprintf(w, "chimera_pool_payouts_wallet_balance_ltc %.8f\n", poolMetrics.WalletBalanceLTC)

			// Network difficulty
			fmt.Fprintf(w, "# HELP chimera_pool_network_difficulty Current network difficulty\n")
			fmt.Fprintf(w, "# TYPE chimera_pool_network_difficulty gauge\n")
			fmt.Fprintf(w, "chimera_pool_network_difficulty %.2f\n", poolMetrics.Difficulty)

			// Block height
			fmt.Fprintf(w, "# HELP chimera_pool_block_height Current block height\n")
			fmt.Fprintf(w, "# TYPE chimera_pool_block_height gauge\n")
			fmt.Fprintf(w, "chimera_pool_block_height %d\n", poolMetrics.BlockHeight)
		}
	}
}

// healthHandler handles /health endpoint for simple health checks.
func (p *PrometheusExporter) healthHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	status, err := p.monitor.GetHealthStatus(ctx)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"status":"error","message":"%s"}`, err.Error())
		return
	}

	// Check if any node is unhealthy
	allHealthy := true
	for _, health := range status {
		if health.Status == HealthStatusUnhealthy {
			allHealthy = false
			break
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if allHealthy {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy","nodes":%d}`, len(status))
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, `{"status":"unhealthy","nodes":%d}`, len(status))
	}
}
