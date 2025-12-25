package payouts

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// =============================================================================
// PAYOUT METRICS FOR PROMETHEUS MONITORING
// =============================================================================

// PayoutMetrics holds all payout-related Prometheus metrics
type PayoutMetrics struct {
	// Counters
	PayoutsProcessed *prometheus.CounterVec
	PayoutsFailed    *prometheus.CounterVec
	BlocksProcessed  prometheus.Counter
	CalculatorCalls  *prometheus.CounterVec

	// Histograms
	PayoutAmount       *prometheus.HistogramVec
	PayoutDuration     *prometheus.HistogramVec
	CalculatorDuration *prometheus.HistogramVec

	// Gauges
	PendingPayouts prometheus.Gauge
	WalletBalance  prometheus.Gauge
	UserBalances   prometheus.Gauge
}

// NewPayoutMetrics creates and registers all payout metrics
func NewPayoutMetrics(namespace string, reg prometheus.Registerer) *PayoutMetrics {
	m := &PayoutMetrics{
		PayoutsProcessed: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "payouts",
				Name:      "processed_total",
				Help:      "Total number of payouts processed successfully",
			},
			[]string{"mode"},
		),

		PayoutsFailed: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "payouts",
				Name:      "failed_total",
				Help:      "Total number of payouts that failed",
			},
			[]string{"mode", "reason"},
		),

		BlocksProcessed: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "payouts",
				Name:      "blocks_processed_total",
				Help:      "Total number of blocks processed for payouts",
			},
		),

		CalculatorCalls: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "payouts",
				Name:      "calculator_calls_total",
				Help:      "Total number of calculator invocations by mode",
			},
			[]string{"mode"},
		),

		PayoutAmount: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "payouts",
				Name:      "amount_litoshis",
				Help:      "Distribution of payout amounts in litoshis",
				Buckets:   []float64{100000, 500000, 1000000, 5000000, 10000000, 50000000, 100000000, 500000000, 1000000000},
			},
			[]string{"mode"},
		),

		PayoutDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "payouts",
				Name:      "duration_seconds",
				Help:      "Time taken to process payouts",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"mode"},
		),

		CalculatorDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "payouts",
				Name:      "calculator_duration_seconds",
				Help:      "Time taken by payout calculators",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"mode"},
		),

		PendingPayouts: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "payouts",
				Name:      "pending_count",
				Help:      "Current number of pending payouts",
			},
		),

		WalletBalance: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "payouts",
				Name:      "wallet_balance_ltc",
				Help:      "Current wallet balance in LTC",
			},
		),

		UserBalances: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "payouts",
				Name:      "user_balances_total_ltc",
				Help:      "Total of all user balances in LTC",
			},
		),
	}

	// Register all metrics
	reg.MustRegister(
		m.PayoutsProcessed,
		m.PayoutsFailed,
		m.BlocksProcessed,
		m.CalculatorCalls,
		m.PayoutAmount,
		m.PayoutDuration,
		m.CalculatorDuration,
		m.PendingPayouts,
		m.WalletBalance,
		m.UserBalances,
	)

	return m
}

// RecordPayoutProcessed records a successful payout
func (m *PayoutMetrics) RecordPayoutProcessed(mode string, amount int64) {
	m.PayoutsProcessed.WithLabelValues(mode).Inc()
	m.PayoutAmount.WithLabelValues(mode).Observe(float64(amount))
}

// RecordPayoutFailed records a failed payout
func (m *PayoutMetrics) RecordPayoutFailed(mode string, reason string) {
	m.PayoutsFailed.WithLabelValues(mode, reason).Inc()
}

// RecordPayoutDuration records payout processing duration
func (m *PayoutMetrics) RecordPayoutDuration(mode string, duration time.Duration) {
	m.PayoutDuration.WithLabelValues(mode).Observe(duration.Seconds())
}

// SetPendingPayouts sets the current pending payout count
func (m *PayoutMetrics) SetPendingPayouts(count int) {
	m.PendingPayouts.Set(float64(count))
}

// SetWalletBalance sets the current wallet balance (converts litoshis to LTC)
func (m *PayoutMetrics) SetWalletBalance(litoshis int64) {
	ltc := float64(litoshis) / 100000000
	m.WalletBalance.Set(ltc)
}

// SetUserBalancesTotal sets the total of all user balances
func (m *PayoutMetrics) SetUserBalancesTotal(litoshis int64) {
	ltc := float64(litoshis) / 100000000
	m.UserBalances.Set(ltc)
}

// RecordBlockProcessed records a block being processed
func (m *PayoutMetrics) RecordBlockProcessed(reward int64, payoutCount int) {
	m.BlocksProcessed.Inc()
}

// RecordCalculatorUsage records calculator invocation
func (m *PayoutMetrics) RecordCalculatorUsage(mode string, duration time.Duration) {
	m.CalculatorCalls.WithLabelValues(mode).Inc()
	m.CalculatorDuration.WithLabelValues(mode).Observe(duration.Seconds())
}

// =============================================================================
// INSTRUMENTED PROCESSOR WITH METRICS
// =============================================================================

// InstrumentedProcessor wraps PayoutProcessor with metrics
type InstrumentedProcessor struct {
	processor *PayoutProcessor
	metrics   *PayoutMetrics
}

// NewInstrumentedProcessor creates a processor with metrics instrumentation
func NewInstrumentedProcessor(processor *PayoutProcessor, metrics *PayoutMetrics) *InstrumentedProcessor {
	return &InstrumentedProcessor{
		processor: processor,
		metrics:   metrics,
	}
}

// Start starts the instrumented processor
func (p *InstrumentedProcessor) Start() {
	p.processor.Start()
}

// Stop stops the instrumented processor
func (p *InstrumentedProcessor) Stop() {
	p.processor.Stop()
}

// UpdateMetrics updates gauge metrics from current state
func (p *InstrumentedProcessor) UpdateMetrics() {
	stats := p.processor.GetStats()
	p.metrics.SetPendingPayouts(int(stats.PayoutsProcessed - stats.PayoutsFailed))
}
