package api

import (
	"net/http"
	"time"

	"github.com/chimera-pool/chimera-pool-core/internal/payouts"
	"github.com/gin-gonic/gin"
)

// =============================================================================
// ISP-COMPLIANT PAYOUT SERVICE INTERFACES
// Each interface is small and focused on a single responsibility
// =============================================================================

// PayoutSettingsReader reads user payout settings (single responsibility)
type PayoutSettingsReader interface {
	GetUserPayoutSettings(userID int64) (*PayoutSettingsData, error)
}

// PayoutSettingsWriter updates user payout settings (single responsibility)
type PayoutSettingsWriter interface {
	UpdateUserPayoutSettings(userID int64, settings *UpdatePayoutSettingsRequest) (*PayoutSettingsData, error)
}

// PayoutModesReader reads available payout modes (single responsibility)
type PayoutModesReader interface {
	GetAvailablePayoutModes() ([]PayoutModeInfo, error)
	GetPoolFeeConfig() ([]PoolFeeConfigData, error)
}

// PayoutEstimator estimates payouts (single responsibility)
type PayoutEstimator interface {
	EstimatePayout(userID int64, mode payouts.PayoutMode) (*PayoutEstimateData, error)
}

// =============================================================================
// PAYOUT DATA MODELS
// =============================================================================

// PayoutSettingsData represents user payout settings
type PayoutSettingsData struct {
	UserID           int64              `json:"user_id"`
	PayoutMode       payouts.PayoutMode `json:"payout_mode"`
	PayoutModeName   string             `json:"payout_mode_name"`
	MinPayoutAmount  int64              `json:"min_payout_amount"`
	PayoutAddress    string             `json:"payout_address"`
	AutoPayoutEnable bool               `json:"auto_payout_enable"`
	FeePercent       float64            `json:"fee_percent"`
	CreatedAt        time.Time          `json:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at"`
}

// UpdatePayoutSettingsRequest represents payout settings update request
type UpdatePayoutSettingsRequest struct {
	PayoutMode       string `json:"payout_mode" binding:"omitempty"`
	MinPayoutAmount  *int64 `json:"min_payout_amount,omitempty"`
	PayoutAddress    string `json:"payout_address,omitempty"`
	AutoPayoutEnable *bool  `json:"auto_payout_enable,omitempty"`
}

// PayoutModeInfo represents information about a payout mode
type PayoutModeInfo struct {
	Mode        payouts.PayoutMode `json:"mode"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	FeePercent  float64            `json:"fee_percent"`
	IsEnabled   bool               `json:"is_enabled"`
	RiskLevel   string             `json:"risk_level"` // low, medium, high (for pool)
	BestFor     string             `json:"best_for"`
}

// PoolFeeConfigData represents pool fee configuration
type PoolFeeConfigData struct {
	PayoutMode payouts.PayoutMode `json:"payout_mode"`
	CoinSymbol string             `json:"coin_symbol"`
	FeePercent float64            `json:"fee_percent"`
	MinPayout  int64              `json:"min_payout"`
	IsEnabled  bool               `json:"is_enabled"`
}

// PayoutEstimateData represents payout estimate
type PayoutEstimateData struct {
	UserID          int64              `json:"user_id"`
	PayoutMode      payouts.PayoutMode `json:"payout_mode"`
	EstimatedAmount int64              `json:"estimated_amount"`
	CurrentDiff     float64            `json:"current_difficulty"`
	WindowDiff      float64            `json:"window_difficulty"`
	SharePercentage float64            `json:"share_percentage"`
	FeePercent      float64            `json:"fee_percent"`
	NetEstimate     int64              `json:"net_estimate"`
	EstimatedAt     time.Time          `json:"estimated_at"`
}

// =============================================================================
// PAYOUT HANDLERS - ISP-COMPLIANT IMPLEMENTATION
// =============================================================================

// PayoutHandlers provides HTTP handlers for payout-related endpoints
type PayoutHandlers struct {
	settingsReader PayoutSettingsReader
	settingsWriter PayoutSettingsWriter
	modesReader    PayoutModesReader
	estimator      PayoutEstimator
}

// NewPayoutHandlers creates new payout handlers with injected dependencies
func NewPayoutHandlers(
	settingsReader PayoutSettingsReader,
	settingsWriter PayoutSettingsWriter,
	modesReader PayoutModesReader,
	estimator PayoutEstimator,
) *PayoutHandlers {
	return &PayoutHandlers{
		settingsReader: settingsReader,
		settingsWriter: settingsWriter,
		modesReader:    modesReader,
		estimator:      estimator,
	}
}

// GetPayoutSettings returns user's payout settings
// GET /api/v1/user/payout-settings
func (h *PayoutHandlers) GetPayoutSettings(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
			Code:    http.StatusUnauthorized,
		})
		return
	}

	userIDInt, ok := userID.(int64)
	if !ok {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Invalid user ID format",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	settings, err := h.settingsReader.GetUserPayoutSettings(userIDInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get payout settings: " + err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	// Return default settings if none exist
	if settings == nil {
		settings = &PayoutSettingsData{
			UserID:           userIDInt,
			PayoutMode:       payouts.PayoutModePPLNS,
			PayoutModeName:   "PPLNS",
			MinPayoutAmount:  1000000, // 0.01 LTC default
			AutoPayoutEnable: true,
			FeePercent:       1.0,
		}
	}

	c.JSON(http.StatusOK, settings)
}

// UpdatePayoutSettings updates user's payout settings
// PUT /api/v1/user/payout-settings
func (h *PayoutHandlers) UpdatePayoutSettings(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
			Code:    http.StatusUnauthorized,
		})
		return
	}

	userIDInt, ok := userID.(int64)
	if !ok {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Invalid user ID format",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	var req UpdatePayoutSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid request data: " + err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	// Validate payout mode if provided
	if req.PayoutMode != "" {
		mode := payouts.PayoutMode(req.PayoutMode)
		if !mode.IsValid() {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "validation_error",
				Message: "Invalid payout mode: " + req.PayoutMode,
				Code:    http.StatusBadRequest,
			})
			return
		}
	}

	settings, err := h.settingsWriter.UpdateUserPayoutSettings(userIDInt, &req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errMsg := err.Error()
		if errMsg == "payout mode not enabled" {
			statusCode = http.StatusBadRequest
		}
		c.JSON(statusCode, ErrorResponse{
			Error:   "update_failed",
			Message: errMsg,
			Code:    statusCode,
		})
		return
	}

	c.JSON(http.StatusOK, settings)
}

// GetAvailablePayoutModes returns all available payout modes
// GET /api/v1/payout-modes
func (h *PayoutHandlers) GetAvailablePayoutModes(c *gin.Context) {
	modes, err := h.modesReader.GetAvailablePayoutModes()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get payout modes: " + err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"modes": modes,
		"total": len(modes),
	})
}

// GetPoolFeeConfig returns pool fee configuration
// GET /api/v1/pool/fees
func (h *PayoutHandlers) GetPoolFeeConfig(c *gin.Context) {
	config, err := h.modesReader.GetPoolFeeConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get fee config: " + err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"fees": config,
	})
}

// GetPayoutEstimate returns payout estimate for user
// GET /api/v1/user/payout-estimate
func (h *PayoutHandlers) GetPayoutEstimate(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
			Code:    http.StatusUnauthorized,
		})
		return
	}

	userIDInt, ok := userID.(int64)
	if !ok {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Invalid user ID format",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	// Get mode from query param, default to user's current mode
	modeStr := c.Query("mode")
	var mode payouts.PayoutMode
	if modeStr != "" {
		mode = payouts.PayoutMode(modeStr)
		if !mode.IsValid() {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "validation_error",
				Message: "Invalid payout mode: " + modeStr,
				Code:    http.StatusBadRequest,
			})
			return
		}
	} else {
		mode = payouts.PayoutModePPLNS // Default
	}

	estimate, err := h.estimator.EstimatePayout(userIDInt, mode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to estimate payout: " + err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, estimate)
}

// =============================================================================
// DEFAULT SERVICE IMPLEMENTATION
// =============================================================================

// DefaultPayoutService implements the payout service interfaces
type DefaultPayoutService struct {
	manager *payouts.PayoutManager
	config  *payouts.PayoutConfig
}

// NewDefaultPayoutService creates a default payout service
func NewDefaultPayoutService(manager *payouts.PayoutManager) *DefaultPayoutService {
	config := payouts.DefaultPayoutConfig()
	if manager != nil {
		config = manager.GetConfig()
	}
	return &DefaultPayoutService{
		manager: manager,
		config:  config,
	}
}

// GetUserPayoutSettings implements PayoutSettingsReader
func (s *DefaultPayoutService) GetUserPayoutSettings(userID int64) (*PayoutSettingsData, error) {
	if s.manager == nil {
		return nil, nil // Return nil to trigger default settings
	}

	// ctx would normally come from request
	settings, err := s.manager.GetUserSettings(nil, userID)
	if err != nil {
		return nil, err
	}

	if settings == nil {
		return nil, nil
	}

	return &PayoutSettingsData{
		UserID:           settings.UserID,
		PayoutMode:       settings.PayoutMode,
		PayoutModeName:   string(settings.PayoutMode),
		MinPayoutAmount:  settings.MinPayoutAmount,
		PayoutAddress:    settings.PayoutAddress,
		AutoPayoutEnable: settings.AutoPayoutEnable,
		FeePercent:       s.manager.GetFeeForMode(settings.PayoutMode),
		CreatedAt:        settings.CreatedAt,
		UpdatedAt:        settings.UpdatedAt,
	}, nil
}

// UpdateUserPayoutSettings implements PayoutSettingsWriter
func (s *DefaultPayoutService) UpdateUserPayoutSettings(userID int64, req *UpdatePayoutSettingsRequest) (*PayoutSettingsData, error) {
	if s.manager == nil {
		return nil, nil
	}

	// Build settings from request
	settings := &payouts.UserPayoutSettings{
		UserID: userID,
	}

	// Get existing settings first
	existing, _ := s.manager.GetUserSettings(nil, userID)
	if existing != nil {
		settings = existing
	}

	// Apply updates
	if req.PayoutMode != "" {
		settings.PayoutMode = payouts.PayoutMode(req.PayoutMode)
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

	// Update settings
	if err := s.manager.UpdateUserSettings(nil, settings); err != nil {
		return nil, err
	}

	return &PayoutSettingsData{
		UserID:           settings.UserID,
		PayoutMode:       settings.PayoutMode,
		PayoutModeName:   string(settings.PayoutMode),
		MinPayoutAmount:  settings.MinPayoutAmount,
		PayoutAddress:    settings.PayoutAddress,
		AutoPayoutEnable: settings.AutoPayoutEnable,
		FeePercent:       s.manager.GetFeeForMode(settings.PayoutMode),
	}, nil
}

// GetAvailablePayoutModes implements PayoutModesReader
func (s *DefaultPayoutService) GetAvailablePayoutModes() ([]PayoutModeInfo, error) {
	modes := make([]PayoutModeInfo, 0)

	for _, mode := range payouts.AllPayoutModes() {
		isEnabled := s.config.IsModeEnabled(mode)

		riskLevel := "low"
		bestFor := ""
		switch mode {
		case payouts.PayoutModePPLNS:
			riskLevel = "low"
			bestFor = "Loyal miners who mine consistently"
		case payouts.PayoutModePPS:
			riskLevel = "high"
			bestFor = "Risk-averse miners wanting predictable income"
		case payouts.PayoutModePPSPlus:
			riskLevel = "medium"
			bestFor = "Balance of stability and upside from tx fees"
		case payouts.PayoutModeFPPS:
			riskLevel = "high"
			bestFor = "Maximum predictability"
		case payouts.PayoutModeSCORE:
			riskLevel = "low"
			bestFor = "Discouraging pool hopping"
		case payouts.PayoutModeSOLO:
			riskLevel = "low"
			bestFor = "Large miners wanting full block rewards"
		case payouts.PayoutModeSLICE:
			riskLevel = "low"
			bestFor = "V2 miners with job negotiation, advanced miners"
		}

		modes = append(modes, PayoutModeInfo{
			Mode:        mode,
			Name:        string(mode),
			Description: mode.Description(),
			FeePercent:  s.config.GetFeeForMode(mode),
			IsEnabled:   isEnabled,
			RiskLevel:   riskLevel,
			BestFor:     bestFor,
		})
	}

	return modes, nil
}

// GetPoolFeeConfig implements PayoutModesReader
func (s *DefaultPayoutService) GetPoolFeeConfig() ([]PoolFeeConfigData, error) {
	configs := make([]PoolFeeConfigData, 0)

	for _, mode := range payouts.AllPayoutModes() {
		// LTC config
		configs = append(configs, PoolFeeConfigData{
			PayoutMode: mode,
			CoinSymbol: "LTC",
			FeePercent: s.config.GetFeeForMode(mode),
			MinPayout:  s.config.MinPayoutLTC,
			IsEnabled:  s.config.IsModeEnabled(mode),
		})

		// BDAG config
		configs = append(configs, PoolFeeConfigData{
			PayoutMode: mode,
			CoinSymbol: "BDAG",
			FeePercent: s.config.GetFeeForMode(mode),
			MinPayout:  s.config.MinPayoutBDAG,
			IsEnabled:  s.config.IsModeEnabled(mode),
		})
	}

	return configs, nil
}

// EstimatePayout implements PayoutEstimator
func (s *DefaultPayoutService) EstimatePayout(userID int64, mode payouts.PayoutMode) (*PayoutEstimateData, error) {
	feePercent := s.config.GetFeeForMode(mode)

	// This is a simplified estimate - real implementation would query share data
	return &PayoutEstimateData{
		UserID:          userID,
		PayoutMode:      mode,
		EstimatedAmount: 0, // Would be calculated from actual shares
		FeePercent:      feePercent,
		NetEstimate:     0,
		EstimatedAt:     time.Now(),
	}, nil
}
