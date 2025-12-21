package api

import (
	"net/http"
	"time"

	"github.com/chimera-pool/chimera-pool-core/internal/monitoring"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// MonitoringHandlers handles monitoring-related API endpoints
type MonitoringHandlers struct {
	service *monitoring.Service
}

// NewMonitoringHandlers creates new monitoring handlers
func NewMonitoringHandlers(service *monitoring.Service) *MonitoringHandlers {
	return &MonitoringHandlers{
		service: service,
	}
}

// RecordMetricRequest represents the request to record a metric
type RecordMetricRequest struct {
	Name   string            `json:"name" binding:"required"`
	Value  float64           `json:"value" binding:"required"`
	Labels map[string]string `json:"labels"`
	Type   string            `json:"type" binding:"required"`
}

// RecordMetric records a new metric
func (h *MonitoringHandlers) RecordMetric(c *gin.Context) {
	var req RecordMetricRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	metric := &monitoring.Metric{
		Name:      req.Name,
		Value:     req.Value,
		Labels:    req.Labels,
		Timestamp: time.Now(),
		Type:      req.Type,
	}

	err := h.service.RecordMetric(c.Request.Context(), metric)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to record metric",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Metric recorded successfully",
		"metric":  metric,
	})
}

// GetMetrics retrieves metrics for a given time range
func (h *MonitoringHandlers) GetMetrics(c *gin.Context) {
	name := c.Query("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Metric name is required",
		})
		return
	}

	startStr := c.DefaultQuery("start", time.Now().Add(-24*time.Hour).Format(time.RFC3339))
	endStr := c.DefaultQuery("end", time.Now().Format(time.RFC3339))

	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid start time format",
			"details": err.Error(),
		})
		return
	}

	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid end time format",
			"details": err.Error(),
		})
		return
	}

	metrics, err := h.service.GetMetrics(c.Request.Context(), name, start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get metrics",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"metrics": metrics,
		"name":    name,
		"start":   start,
		"end":     end,
		"count":   len(metrics),
	})
}

// CreateAlertRequest represents the request to create an alert
type CreateAlertRequest struct {
	Name        string            `json:"name" binding:"required"`
	Description string            `json:"description"`
	Severity    string            `json:"severity" binding:"required"`
	Labels      map[string]string `json:"labels"`
}

// CreateAlert creates a new alert
func (h *MonitoringHandlers) CreateAlert(c *gin.Context) {
	var req CreateAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	alert, err := h.service.CreateAlert(c.Request.Context(), req.Name, req.Description, req.Severity, req.Labels)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create alert",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"alert":   alert,
		"message": "Alert created successfully",
	})
}

// ResolveAlert resolves an active alert
func (h *MonitoringHandlers) ResolveAlert(c *gin.Context) {
	alertIDStr := c.Param("alertId")
	alertID, err := uuid.Parse(alertIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid alert ID format",
		})
		return
	}

	err = h.service.ResolveAlert(c.Request.Context(), alertID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to resolve alert",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Alert resolved successfully",
		"alert_id": alertID,
	})
}

// CreateAlertRuleRequest represents the request to create an alert rule
type CreateAlertRuleRequest struct {
	Name      string  `json:"name" binding:"required"`
	Query     string  `json:"query" binding:"required"`
	Condition string  `json:"condition" binding:"required"`
	Threshold float64 `json:"threshold" binding:"required"`
	Duration  string  `json:"duration" binding:"required"`
	Severity  string  `json:"severity" binding:"required"`
}

// CreateAlertRule creates a new alert rule
func (h *MonitoringHandlers) CreateAlertRule(c *gin.Context) {
	var req CreateAlertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	rule, err := h.service.CreateAlertRule(c.Request.Context(), req.Name, req.Query, req.Condition, req.Threshold, req.Duration, req.Severity)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create alert rule",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"rule":    rule,
		"message": "Alert rule created successfully",
	})
}

// EvaluateAlertRules evaluates all active alert rules
func (h *MonitoringHandlers) EvaluateAlertRules(c *gin.Context) {
	alerts, err := h.service.EvaluateAlertRules(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to evaluate alert rules",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"alerts":  alerts,
		"count":   len(alerts),
		"message": "Alert rules evaluated successfully",
	})
}

// CreateDashboardRequest represents the request to create a dashboard
type CreateDashboardRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Config      string `json:"config" binding:"required"`
	IsPublic    bool   `json:"is_public"`
}

// CreateDashboard creates a new monitoring dashboard
func (h *MonitoringHandlers) CreateDashboard(c *gin.Context) {
	var req CreateDashboardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	createdBy, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Invalid user ID format",
		})
		return
	}

	dashboard, err := h.service.CreateDashboard(c.Request.Context(), req.Name, req.Description, req.Config, req.IsPublic, createdBy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create dashboard",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"dashboard": dashboard,
		"message":   "Dashboard created successfully",
	})
}

// RecordPerformanceMetricsRequest represents the request to record performance metrics
type RecordPerformanceMetricsRequest struct {
	CPUUsage        float64 `json:"cpu_usage" binding:"required"`
	MemoryUsage     float64 `json:"memory_usage" binding:"required"`
	DiskUsage       float64 `json:"disk_usage" binding:"required"`
	NetworkIn       float64 `json:"network_in"`
	NetworkOut      float64 `json:"network_out"`
	ActiveMiners    int     `json:"active_miners" binding:"required"`
	TotalHashrate   float64 `json:"total_hashrate" binding:"required"`
	SharesPerSecond float64 `json:"shares_per_second"`
	BlocksFound     int     `json:"blocks_found"`
	Uptime          float64 `json:"uptime"`
}

// RecordPerformanceMetrics records system performance metrics
func (h *MonitoringHandlers) RecordPerformanceMetrics(c *gin.Context) {
	var req RecordPerformanceMetricsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	metrics := &monitoring.PerformanceMetrics{
		Timestamp:       time.Now(),
		CPUUsage:        req.CPUUsage,
		MemoryUsage:     req.MemoryUsage,
		DiskUsage:       req.DiskUsage,
		NetworkIn:       req.NetworkIn,
		NetworkOut:      req.NetworkOut,
		ActiveMiners:    req.ActiveMiners,
		TotalHashrate:   req.TotalHashrate,
		SharesPerSecond: req.SharesPerSecond,
		BlocksFound:     req.BlocksFound,
		Uptime:          req.Uptime,
	}

	err := h.service.RecordPerformanceMetrics(c.Request.Context(), metrics)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to record performance metrics",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Performance metrics recorded successfully",
		"metrics": metrics,
	})
}

// RecordMinerMetricsRequest represents the request to record miner metrics
type RecordMinerMetricsRequest struct {
	MinerID         uuid.UUID `json:"miner_id" binding:"required"`
	Hashrate        float64   `json:"hashrate" binding:"required"`
	SharesSubmitted int64     `json:"shares_submitted"`
	SharesAccepted  int64     `json:"shares_accepted"`
	SharesRejected  int64     `json:"shares_rejected"`
	IsOnline        bool      `json:"is_online"`
	Difficulty      float64   `json:"difficulty"`
	Earnings        float64   `json:"earnings"`
}

// RecordMinerMetrics records individual miner metrics
func (h *MonitoringHandlers) RecordMinerMetrics(c *gin.Context) {
	var req RecordMinerMetricsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	metrics := &monitoring.MinerMetrics{
		MinerID:         req.MinerID,
		Timestamp:       time.Now(),
		Hashrate:        req.Hashrate,
		SharesSubmitted: req.SharesSubmitted,
		SharesAccepted:  req.SharesAccepted,
		SharesRejected:  req.SharesRejected,
		LastSeen:        time.Now(),
		IsOnline:        req.IsOnline,
		Difficulty:      req.Difficulty,
		Earnings:        req.Earnings,
	}

	err := h.service.RecordMinerMetrics(c.Request.Context(), metrics)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to record miner metrics",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Miner metrics recorded successfully",
		"metrics": metrics,
	})
}

// GetPerformanceMetrics retrieves performance metrics for a time range
func (h *MonitoringHandlers) GetPerformanceMetrics(c *gin.Context) {
	startStr := c.DefaultQuery("start", time.Now().Add(-24*time.Hour).Format(time.RFC3339))
	endStr := c.DefaultQuery("end", time.Now().Format(time.RFC3339))

	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid start time format",
			"details": err.Error(),
		})
		return
	}

	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid end time format",
			"details": err.Error(),
		})
		return
	}

	// This would typically call a repository method to get performance metrics
	// For now, we'll return a placeholder response
	c.JSON(http.StatusOK, gin.H{
		"message": "Performance metrics retrieved successfully",
		"start":   start,
		"end":     end,
		"metrics": []interface{}{}, // Placeholder
	})
}

// GetMinerMetrics retrieves miner metrics for a specific miner and time range
func (h *MonitoringHandlers) GetMinerMetrics(c *gin.Context) {
	minerIDStr := c.Param("minerId")
	minerID, err := uuid.Parse(minerIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid miner ID format",
		})
		return
	}

	startStr := c.DefaultQuery("start", time.Now().Add(-24*time.Hour).Format(time.RFC3339))
	endStr := c.DefaultQuery("end", time.Now().Format(time.RFC3339))

	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid start time format",
			"details": err.Error(),
		})
		return
	}

	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid end time format",
			"details": err.Error(),
		})
		return
	}

	// This would typically call a repository method to get miner metrics
	// For now, we'll return a placeholder response
	c.JSON(http.StatusOK, gin.H{
		"message":  "Miner metrics retrieved successfully",
		"miner_id": minerID,
		"start":    start,
		"end":      end,
		"metrics":  []interface{}{}, // Placeholder
	})
}
