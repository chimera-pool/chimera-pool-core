package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chimera-pool/chimera-pool-core/internal/payouts"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// MOCK IMPLEMENTATIONS
// =============================================================================

type mockPayoutSettingsReader struct {
	settings map[int64]*PayoutSettingsData
}

func (m *mockPayoutSettingsReader) GetUserPayoutSettings(userID int64) (*PayoutSettingsData, error) {
	return m.settings[userID], nil
}

type mockPayoutSettingsWriter struct {
	settings map[int64]*PayoutSettingsData
}

func (m *mockPayoutSettingsWriter) UpdateUserPayoutSettings(userID int64, req *UpdatePayoutSettingsRequest) (*PayoutSettingsData, error) {
	settings := m.settings[userID]
	if settings == nil {
		settings = &PayoutSettingsData{
			UserID:     userID,
			PayoutMode: payouts.PayoutModePPLNS,
		}
	}

	if req.PayoutMode != "" {
		mode := payouts.PayoutMode(req.PayoutMode)
		if !mode.IsValid() {
			return nil, nil
		}
		settings.PayoutMode = mode
	}
	if req.MinPayoutAmount != nil {
		settings.MinPayoutAmount = *req.MinPayoutAmount
	}
	if req.PayoutAddress != "" {
		settings.PayoutAddress = req.PayoutAddress
	}
	if req.AutoPayoutEnable != nil {
		settings.AutoPayoutEnable = *req.AutoPayoutEnable
	}
	settings.UpdatedAt = time.Now()

	m.settings[userID] = settings
	return settings, nil
}

type mockPayoutModesReader struct{}

func (m *mockPayoutModesReader) GetAvailablePayoutModes() ([]PayoutModeInfo, error) {
	return []PayoutModeInfo{
		{Mode: payouts.PayoutModePPLNS, Name: "pplns", FeePercent: 1.0, IsEnabled: true},
		{Mode: payouts.PayoutModePPS, Name: "pps", FeePercent: 2.0, IsEnabled: false},
		{Mode: payouts.PayoutModeSLICE, Name: "slice", FeePercent: 0.8, IsEnabled: true},
	}, nil
}

func (m *mockPayoutModesReader) GetPoolFeeConfig() ([]PoolFeeConfigData, error) {
	return []PoolFeeConfigData{
		{PayoutMode: payouts.PayoutModePPLNS, CoinSymbol: "LTC", FeePercent: 1.0, MinPayout: 1000000, IsEnabled: true},
		{PayoutMode: payouts.PayoutModePPLNS, CoinSymbol: "BDAG", FeePercent: 1.0, MinPayout: 1000000000, IsEnabled: true},
	}, nil
}

type mockPayoutEstimator struct{}

func (m *mockPayoutEstimator) EstimatePayout(userID int64, mode payouts.PayoutMode) (*PayoutEstimateData, error) {
	return &PayoutEstimateData{
		UserID:          userID,
		PayoutMode:      mode,
		EstimatedAmount: 5000000,
		FeePercent:      1.0,
		NetEstimate:     4950000,
		EstimatedAt:     time.Now(),
	}, nil
}

// =============================================================================
// TEST SETUP
// =============================================================================

func setupPayoutTestRouter() (*gin.Engine, *PayoutHandlers) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	settingsReader := &mockPayoutSettingsReader{
		settings: map[int64]*PayoutSettingsData{
			1: {
				UserID:           1,
				PayoutMode:       payouts.PayoutModePPLNS,
				PayoutModeName:   "PPLNS",
				MinPayoutAmount:  1000000,
				AutoPayoutEnable: true,
				FeePercent:       1.0,
			},
		},
	}
	settingsWriter := &mockPayoutSettingsWriter{
		settings: make(map[int64]*PayoutSettingsData),
	}

	handlers := NewPayoutHandlers(
		settingsReader,
		settingsWriter,
		&mockPayoutModesReader{},
		&mockPayoutEstimator{},
	)

	return router, handlers
}

func authMiddleware(userID int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}
}

// =============================================================================
// TESTS
// =============================================================================

func TestPayoutHandlers_GetPayoutSettings(t *testing.T) {
	router, handlers := setupPayoutTestRouter()
	router.GET("/api/v1/user/payout-settings", authMiddleware(1), handlers.GetPayoutSettings)

	t.Run("returns user settings", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/user/payout-settings", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response PayoutSettingsData
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, payouts.PayoutModePPLNS, response.PayoutMode)
		assert.Equal(t, int64(1000000), response.MinPayoutAmount)
	})

	t.Run("returns default settings for new user", func(t *testing.T) {
		router2, handlers2 := setupPayoutTestRouter()
		router2.GET("/api/v1/user/payout-settings", authMiddleware(999), handlers2.GetPayoutSettings)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/user/payout-settings", nil)
		w := httptest.NewRecorder()
		router2.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response PayoutSettingsData
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, payouts.PayoutModePPLNS, response.PayoutMode) // Default
	})

	t.Run("returns unauthorized without auth", func(t *testing.T) {
		router3 := gin.New()
		router3.GET("/api/v1/user/payout-settings", handlers.GetPayoutSettings)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/user/payout-settings", nil)
		w := httptest.NewRecorder()
		router3.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestPayoutHandlers_UpdatePayoutSettings(t *testing.T) {
	router, handlers := setupPayoutTestRouter()
	router.PUT("/api/v1/user/payout-settings", authMiddleware(1), handlers.UpdatePayoutSettings)

	t.Run("updates payout mode", func(t *testing.T) {
		body := UpdatePayoutSettingsRequest{
			PayoutMode: "slice",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPut, "/api/v1/user/payout-settings", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response PayoutSettingsData
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, payouts.PayoutModeSLICE, response.PayoutMode)
	})

	t.Run("updates min payout amount", func(t *testing.T) {
		minAmount := int64(5000000)
		body := UpdatePayoutSettingsRequest{
			MinPayoutAmount: &minAmount,
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPut, "/api/v1/user/payout-settings", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response PayoutSettingsData
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, int64(5000000), response.MinPayoutAmount)
	})

	t.Run("rejects invalid payout mode", func(t *testing.T) {
		body := UpdatePayoutSettingsRequest{
			PayoutMode: "invalid_mode",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPut, "/api/v1/user/payout-settings", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestPayoutHandlers_GetAvailablePayoutModes(t *testing.T) {
	router, handlers := setupPayoutTestRouter()
	router.GET("/api/v1/payout-modes", handlers.GetAvailablePayoutModes)

	t.Run("returns all payout modes", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/payout-modes", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response struct {
			Modes []PayoutModeInfo `json:"modes"`
			Total int              `json:"total"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, 3, response.Total)
		assert.Len(t, response.Modes, 3)
	})
}

func TestPayoutHandlers_GetPoolFeeConfig(t *testing.T) {
	router, handlers := setupPayoutTestRouter()
	router.GET("/api/v1/pool/fees", handlers.GetPoolFeeConfig)

	t.Run("returns fee configuration", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/pool/fees", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response struct {
			Fees []PoolFeeConfigData `json:"fees"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.NotEmpty(t, response.Fees)
	})
}

func TestPayoutHandlers_GetPayoutEstimate(t *testing.T) {
	router, handlers := setupPayoutTestRouter()
	router.GET("/api/v1/user/payout-estimate", authMiddleware(1), handlers.GetPayoutEstimate)

	t.Run("returns payout estimate", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/user/payout-estimate", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response PayoutEstimateData
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, int64(1), response.UserID)
		assert.Greater(t, response.EstimatedAmount, int64(0))
	})

	t.Run("accepts mode parameter", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/user/payout-estimate?mode=slice", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response PayoutEstimateData
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, payouts.PayoutModeSLICE, response.PayoutMode)
	})

	t.Run("rejects invalid mode parameter", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/user/payout-estimate?mode=invalid", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// =============================================================================
// DEFAULT SERVICE TESTS
// =============================================================================

func TestDefaultPayoutService_GetAvailablePayoutModes(t *testing.T) {
	service := NewDefaultPayoutService(nil)

	modes, err := service.GetAvailablePayoutModes()
	require.NoError(t, err)
	assert.Len(t, modes, 7) // All 7 payout modes

	// Verify each mode has required fields
	for _, mode := range modes {
		assert.NotEmpty(t, mode.Name)
		assert.NotEmpty(t, mode.Description)
		assert.NotEmpty(t, mode.RiskLevel)
		assert.NotEmpty(t, mode.BestFor)
		assert.GreaterOrEqual(t, mode.FeePercent, 0.0)
	}
}

func TestDefaultPayoutService_GetPoolFeeConfig(t *testing.T) {
	service := NewDefaultPayoutService(nil)

	config, err := service.GetPoolFeeConfig()
	require.NoError(t, err)
	assert.Len(t, config, 14) // 7 modes * 2 coins (LTC, BDAG)

	// Verify LTC and BDAG configs exist
	ltcCount := 0
	bdagCount := 0
	for _, c := range config {
		if c.CoinSymbol == "LTC" {
			ltcCount++
		} else if c.CoinSymbol == "BDAG" {
			bdagCount++
		}
	}
	assert.Equal(t, 7, ltcCount)
	assert.Equal(t, 7, bdagCount)
}

func TestDefaultPayoutService_EstimatePayout(t *testing.T) {
	service := NewDefaultPayoutService(nil)

	estimate, err := service.EstimatePayout(1, payouts.PayoutModePPLNS)
	require.NoError(t, err)
	assert.Equal(t, int64(1), estimate.UserID)
	assert.Equal(t, payouts.PayoutModePPLNS, estimate.PayoutMode)
	assert.Equal(t, 1.0, estimate.FeePercent)
}
