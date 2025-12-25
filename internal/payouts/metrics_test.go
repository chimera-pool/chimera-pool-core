package payouts

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// PAYOUT METRICS TESTS (TDD)
// =============================================================================

func TestPayoutMetrics_Creation(t *testing.T) {
	t.Run("creates metrics with namespace", func(t *testing.T) {
		reg := prometheus.NewRegistry()
		metrics := NewPayoutMetrics("chimera_pool", reg)

		require.NotNil(t, metrics)
		assert.NotNil(t, metrics.PayoutsProcessed)
		assert.NotNil(t, metrics.PayoutsFailed)
		assert.NotNil(t, metrics.PayoutAmount)
		assert.NotNil(t, metrics.PayoutDuration)
		assert.NotNil(t, metrics.PendingPayouts)
		assert.NotNil(t, metrics.WalletBalance)
	})

	t.Run("registers metrics with registry", func(t *testing.T) {
		reg := prometheus.NewRegistry()
		_ = NewPayoutMetrics("chimera_pool", reg)

		// Verify metrics are registered
		families, err := reg.Gather()
		require.NoError(t, err)
		assert.Greater(t, len(families), 0)
	})
}

func TestPayoutMetrics_RecordPayout(t *testing.T) {
	t.Run("increments processed counter", func(t *testing.T) {
		reg := prometheus.NewRegistry()
		metrics := NewPayoutMetrics("test", reg)

		metrics.RecordPayoutProcessed("pplns", 1000000)

		count := testutil.ToFloat64(metrics.PayoutsProcessed.WithLabelValues("pplns"))
		assert.Equal(t, float64(1), count)
	})

	t.Run("records payout amount", func(t *testing.T) {
		reg := prometheus.NewRegistry()
		metrics := NewPayoutMetrics("test", reg)

		metrics.RecordPayoutProcessed("pplns", 5000000) // 0.05 LTC

		// Verify counter incremented (histogram tested via registry gather)
		count := testutil.ToFloat64(metrics.PayoutsProcessed.WithLabelValues("pplns"))
		assert.Equal(t, float64(1), count)
	})

	t.Run("records multiple payout modes", func(t *testing.T) {
		reg := prometheus.NewRegistry()
		metrics := NewPayoutMetrics("test", reg)

		metrics.RecordPayoutProcessed("pplns", 1000000)
		metrics.RecordPayoutProcessed("slice", 2000000)
		metrics.RecordPayoutProcessed("pplns", 1500000)

		pplnsCount := testutil.ToFloat64(metrics.PayoutsProcessed.WithLabelValues("pplns"))
		sliceCount := testutil.ToFloat64(metrics.PayoutsProcessed.WithLabelValues("slice"))

		assert.Equal(t, float64(2), pplnsCount)
		assert.Equal(t, float64(1), sliceCount)
	})
}

func TestPayoutMetrics_RecordFailure(t *testing.T) {
	t.Run("increments failed counter by reason", func(t *testing.T) {
		reg := prometheus.NewRegistry()
		metrics := NewPayoutMetrics("test", reg)

		metrics.RecordPayoutFailed("pplns", "insufficient_funds")
		metrics.RecordPayoutFailed("pplns", "invalid_address")
		metrics.RecordPayoutFailed("pplns", "insufficient_funds")

		insufficientCount := testutil.ToFloat64(metrics.PayoutsFailed.WithLabelValues("pplns", "insufficient_funds"))
		invalidCount := testutil.ToFloat64(metrics.PayoutsFailed.WithLabelValues("pplns", "invalid_address"))

		assert.Equal(t, float64(2), insufficientCount)
		assert.Equal(t, float64(1), invalidCount)
	})
}

func TestPayoutMetrics_RecordDuration(t *testing.T) {
	t.Run("records payout processing duration", func(t *testing.T) {
		reg := prometheus.NewRegistry()
		metrics := NewPayoutMetrics("test", reg)

		duration := 150 * time.Millisecond
		metrics.RecordPayoutDuration("pplns", duration)

		// Verify by gathering metrics from registry
		families, err := reg.Gather()
		require.NoError(t, err)

		found := false
		for _, f := range families {
			if f.GetName() == "test_payouts_duration_seconds" {
				found = true
				break
			}
		}
		assert.True(t, found, "duration histogram should be registered")
	})
}

func TestPayoutMetrics_SetPendingPayouts(t *testing.T) {
	t.Run("sets pending payout gauge", func(t *testing.T) {
		reg := prometheus.NewRegistry()
		metrics := NewPayoutMetrics("test", reg)

		metrics.SetPendingPayouts(15)

		count := testutil.ToFloat64(metrics.PendingPayouts)
		assert.Equal(t, float64(15), count)
	})

	t.Run("updates pending payout gauge", func(t *testing.T) {
		reg := prometheus.NewRegistry()
		metrics := NewPayoutMetrics("test", reg)

		metrics.SetPendingPayouts(10)
		metrics.SetPendingPayouts(5)

		count := testutil.ToFloat64(metrics.PendingPayouts)
		assert.Equal(t, float64(5), count)
	})
}

func TestPayoutMetrics_SetWalletBalance(t *testing.T) {
	t.Run("sets wallet balance gauge", func(t *testing.T) {
		reg := prometheus.NewRegistry()
		metrics := NewPayoutMetrics("test", reg)

		metrics.SetWalletBalance(10000000000) // 100 LTC in litoshis

		balance := testutil.ToFloat64(metrics.WalletBalance)
		assert.Equal(t, float64(100), balance) // Displayed in LTC
	})
}

func TestPayoutMetrics_BlockMetrics(t *testing.T) {
	t.Run("records block processing", func(t *testing.T) {
		reg := prometheus.NewRegistry()
		metrics := NewPayoutMetrics("test", reg)

		metrics.RecordBlockProcessed(1000000000, 5) // 10 LTC reward, 5 payouts

		blocksProcessed := testutil.ToFloat64(metrics.BlocksProcessed)
		assert.Equal(t, float64(1), blocksProcessed)
	})
}

func TestPayoutMetrics_CalculatorMetrics(t *testing.T) {
	t.Run("records calculator usage", func(t *testing.T) {
		reg := prometheus.NewRegistry()
		metrics := NewPayoutMetrics("test", reg)

		metrics.RecordCalculatorUsage("pplns", 100*time.Millisecond)
		metrics.RecordCalculatorUsage("slice", 150*time.Millisecond)

		pplnsCount := testutil.ToFloat64(metrics.CalculatorCalls.WithLabelValues("pplns"))
		sliceCount := testutil.ToFloat64(metrics.CalculatorCalls.WithLabelValues("slice"))

		assert.Equal(t, float64(1), pplnsCount)
		assert.Equal(t, float64(1), sliceCount)
	})
}
