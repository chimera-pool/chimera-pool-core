package payouts

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// PAYOUT MODE TESTS
// =============================================================================

func TestPayoutMode_IsValid(t *testing.T) {
	validModes := []PayoutMode{
		PayoutModePPLNS,
		PayoutModePPS,
		PayoutModePPSPlus,
		PayoutModeFPPS,
		PayoutModeSCORE,
		PayoutModeSOLO,
		PayoutModeSLICE,
	}

	for _, mode := range validModes {
		t.Run(string(mode), func(t *testing.T) {
			assert.True(t, mode.IsValid())
		})
	}

	t.Run("invalid mode", func(t *testing.T) {
		assert.False(t, PayoutMode("invalid").IsValid())
	})
}

func TestPayoutMode_String(t *testing.T) {
	assert.Equal(t, "pplns", PayoutModePPLNS.String())
	assert.Equal(t, "pps", PayoutModePPS.String())
	assert.Equal(t, "pps_plus", PayoutModePPSPlus.String())
	assert.Equal(t, "fpps", PayoutModeFPPS.String())
	assert.Equal(t, "score", PayoutModeSCORE.String())
	assert.Equal(t, "solo", PayoutModeSOLO.String())
	assert.Equal(t, "slice", PayoutModeSLICE.String())
}

func TestPayoutMode_Description(t *testing.T) {
	tests := []struct {
		mode     PayoutMode
		contains string
	}{
		{PayoutModePPLNS, "Last N Shares"},
		{PayoutModePPS, "Pay Per Share"},
		{PayoutModePPSPlus, "PPS+"},
		{PayoutModeFPPS, "Full PPS"},
		{PayoutModeSCORE, "Score-based"},
		{PayoutModeSOLO, "Solo Mining"},
		{PayoutModeSLICE, "SLICE"},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			desc := tt.mode.Description()
			assert.Contains(t, desc, tt.contains)
		})
	}

	t.Run("unknown mode", func(t *testing.T) {
		desc := PayoutMode("unknown").Description()
		assert.Equal(t, "Unknown payout mode", desc)
	})
}

func TestPayoutMode_DefaultFeePercent(t *testing.T) {
	tests := []struct {
		mode        PayoutMode
		expectedFee float64
	}{
		{PayoutModePPLNS, 1.0},
		{PayoutModePPS, 2.0},
		{PayoutModePPSPlus, 1.5},
		{PayoutModeFPPS, 2.0},
		{PayoutModeSCORE, 1.0},
		{PayoutModeSOLO, 0.5},
		{PayoutModeSLICE, 0.8},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			assert.Equal(t, tt.expectedFee, tt.mode.DefaultFeePercent())
		})
	}
}

func TestAllPayoutModes(t *testing.T) {
	modes := AllPayoutModes()
	assert.Len(t, modes, 7)
	assert.Contains(t, modes, PayoutModePPLNS)
	assert.Contains(t, modes, PayoutModePPS)
	assert.Contains(t, modes, PayoutModePPSPlus)
	assert.Contains(t, modes, PayoutModeFPPS)
	assert.Contains(t, modes, PayoutModeSCORE)
	assert.Contains(t, modes, PayoutModeSOLO)
	assert.Contains(t, modes, PayoutModeSLICE)
}

// =============================================================================
// PAYOUT CONFIG TESTS
// =============================================================================

func TestDefaultPayoutConfig(t *testing.T) {
	config := DefaultPayoutConfig()

	t.Run("default mode is PPLNS", func(t *testing.T) {
		assert.Equal(t, PayoutModePPLNS, config.DefaultMode)
	})

	t.Run("has correct default fees", func(t *testing.T) {
		assert.Equal(t, 1.0, config.FeePPLNS)
		assert.Equal(t, 2.0, config.FeePPS)
		assert.Equal(t, 1.5, config.FeePPSPlus)
		assert.Equal(t, 2.0, config.FeeFPPS)
		assert.Equal(t, 1.0, config.FeeSCORE)
		assert.Equal(t, 0.5, config.FeeSOLO)
		assert.Equal(t, 0.8, config.FeeSLICE)
	})

	t.Run("has correct PPLNS window size", func(t *testing.T) {
		assert.Equal(t, int64(200000), config.PPLNSWindowSize)
	})

	t.Run("has correct SCORE decay factor", func(t *testing.T) {
		assert.Equal(t, 0.5, config.SCOREDecayFactor)
	})

	t.Run("has correct SLICE config", func(t *testing.T) {
		assert.Equal(t, int64(10), config.SLICEWindowSize)
		assert.Equal(t, int64(600), config.SLICESliceDuration)
		assert.Equal(t, 0.7, config.SLICEDecayFactor)
	})

	t.Run("has correct minimum payouts", func(t *testing.T) {
		assert.Equal(t, int64(1000000), config.MinPayoutLTC)
		assert.Equal(t, int64(1000000000), config.MinPayoutBDAG)
	})

	t.Run("correct modes enabled by default", func(t *testing.T) {
		assert.True(t, config.EnablePPLNS)
		assert.False(t, config.EnablePPS) // High risk
		assert.True(t, config.EnablePPSPlus)
		assert.False(t, config.EnableFPPS) // Highest risk
		assert.True(t, config.EnableSCORE)
		assert.True(t, config.EnableSOLO)
		assert.True(t, config.EnableSLICE)
	})
}

func TestPayoutConfig_GetFeeForMode(t *testing.T) {
	config := DefaultPayoutConfig()

	tests := []struct {
		mode        PayoutMode
		expectedFee float64
	}{
		{PayoutModePPLNS, 1.0},
		{PayoutModePPS, 2.0},
		{PayoutModePPSPlus, 1.5},
		{PayoutModeFPPS, 2.0},
		{PayoutModeSCORE, 1.0},
		{PayoutModeSOLO, 0.5},
		{PayoutModeSLICE, 0.8},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			assert.Equal(t, tt.expectedFee, config.GetFeeForMode(tt.mode))
		})
	}

	t.Run("unknown mode returns PPLNS fee", func(t *testing.T) {
		assert.Equal(t, config.FeePPLNS, config.GetFeeForMode(PayoutMode("unknown")))
	})
}

func TestPayoutConfig_IsModeEnabled(t *testing.T) {
	config := DefaultPayoutConfig()

	t.Run("enabled modes", func(t *testing.T) {
		assert.True(t, config.IsModeEnabled(PayoutModePPLNS))
		assert.True(t, config.IsModeEnabled(PayoutModePPSPlus))
		assert.True(t, config.IsModeEnabled(PayoutModeSCORE))
		assert.True(t, config.IsModeEnabled(PayoutModeSOLO))
		assert.True(t, config.IsModeEnabled(PayoutModeSLICE))
	})

	t.Run("disabled modes", func(t *testing.T) {
		assert.False(t, config.IsModeEnabled(PayoutModePPS))
		assert.False(t, config.IsModeEnabled(PayoutModeFPPS))
	})

	t.Run("unknown mode", func(t *testing.T) {
		assert.False(t, config.IsModeEnabled(PayoutMode("unknown")))
	})
}

func TestPayoutConfig_GetEnabledModes(t *testing.T) {
	config := DefaultPayoutConfig()

	modes := config.GetEnabledModes()
	assert.Len(t, modes, 5) // PPLNS, PPS+, SCORE, SOLO, SLICE

	assert.Contains(t, modes, PayoutModePPLNS)
	assert.Contains(t, modes, PayoutModePPSPlus)
	assert.Contains(t, modes, PayoutModeSCORE)
	assert.Contains(t, modes, PayoutModeSOLO)
	assert.Contains(t, modes, PayoutModeSLICE)

	// PPS and FPPS should NOT be in the list
	assert.NotContains(t, modes, PayoutModePPS)
	assert.NotContains(t, modes, PayoutModeFPPS)
}

// =============================================================================
// NULL MERGED MINING PROVIDER TESTS
// =============================================================================

func TestNullMergedMiningProvider(t *testing.T) {
	provider := &NullMergedMiningProvider{}

	t.Run("GetAuxChains returns nil", func(t *testing.T) {
		assert.Nil(t, provider.GetAuxChains())
	})

	t.Run("GetAuxBlockTemplate returns nil", func(t *testing.T) {
		data, err := provider.GetAuxBlockTemplate(nil, "test")
		assert.NoError(t, err)
		assert.Nil(t, data)
	})

	t.Run("SubmitAuxBlock returns nil", func(t *testing.T) {
		err := provider.SubmitAuxBlock(nil, "test", nil)
		assert.NoError(t, err)
	})

	t.Run("CalculateAuxReward returns zero", func(t *testing.T) {
		reward, err := provider.CalculateAuxReward(nil, "test", nil)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), reward)
	})
}
