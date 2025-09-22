package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Repository defines the interface for monitoring data operations
type Repository interface {
	// Metric operations
	StoreMetric(ctx context.Context, metric *Metric) error
	GetMetrics(ctx context.Context, name string, start, end time.Time) ([]*Metric, error)
	
	// Alert operations
	CreateAlert(ctx context.Context, alert *Alert) error
	UpdateAlert(ctx context.Context, alert *Alert) error
	GetActiveAlerts(ctx context.Context) ([]*Alert, error)
	
	// Alert rule operations
	CreateAlertRule(ctx context.Context, rule *AlertRule) error
	GetAlertRules(ctx context.Context) ([]*AlertRule, error)
	
	// Dashboard operations
	CreateDashboard(ctx context.Context, dashboard *Dashboard) error
	GetDashboard(ctx context.Context, id uuid.UUID) (*Dashboard, error)
	
	// Performance metrics
	StorePerformanceMetrics(ctx context.Context, metrics *PerformanceMetrics) error
	GetPerformanceMetrics(ctx context.Context, start, end time.Time) ([]*PerformanceMetrics, error)
	
	// Miner metrics
	StoreMinerMetrics(ctx context.Context, metrics *MinerMetrics) error
	GetMinerMetrics(ctx context.Context, minerID uuid.UUID, start, end time.Time) ([]*MinerMetrics, error)
	
	// Pool metrics
	StorePoolMetrics(ctx context.Context, metrics *PoolMetrics) error
	GetPoolMetrics(ctx context.Context, start, end time.Time) ([]*PoolMetrics, error)
	
	// Alert channels
	CreateAlertChannel(ctx context.Context, channel *AlertChannel) error
	GetAlertChannels(ctx context.Context) ([]*AlertChannel, error)
	
	// Notifications
	RecordNotification(ctx context.Context, notification *Notification) error
}

// PrometheusClient defines the interface for Prometheus operations
type PrometheusClient interface {
	RecordCounter(name string, labels map[string]string, value float64) error
	RecordGauge(name string, labels map[string]string, value float64) error
	RecordHistogram(name string, labels map[string]string, value float64) error
	Query(query string) (float64, error)
}

// Service provides monitoring functionality
type Service struct {
	repo       Repository
	prometheus PrometheusClient
}

// NewService creates a new monitoring service
func NewService(repo Repository, prometheus PrometheusClient) *Service {
	return &Service{
		repo:       repo,
		prometheus: prometheus,
	}
}

// RecordMetric records a metric to both local storage and Prometheus
func (s *Service) RecordMetric(ctx context.Context, metric *Metric) error {
	if err := s.validateMetric(metric); err != nil {
		return err
	}
	
	// Store in local database
	if err := s.repo.StoreMetric(ctx, metric); err != nil {
		return fmt.Errorf("failed to store metric: %w", err)
	}
	
	// Send to Prometheus
	switch metric.Type {
	case "counter":
		if err := s.prometheus.RecordCounter(metric.Name, metric.Labels, metric.Value); err != nil {
			return fmt.Errorf("failed to record counter metric: %w", err)
		}
	case "gauge":
		if err := s.prometheus.RecordGauge(metric.Name, metric.Labels, metric.Value); err != nil {
			return fmt.Errorf("failed to record gauge metric: %w", err)
		}
	case "histogram":
		if err := s.prometheus.RecordHistogram(metric.Name, metric.Labels, metric.Value); err != nil {
			return fmt.Errorf("failed to record histogram metric: %w", err)
		}
	default:
		return fmt.Errorf("unsupported metric type: %s", metric.Type)
	}
	
	return nil
}

// CreateAlert creates a new alert
func (s *Service) CreateAlert(ctx context.Context, name, description, severity string, labels map[string]string) (*Alert, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("alert name cannot be empty")
	}
	
	if !s.isValidSeverity(severity) {
		return nil, fmt.Errorf("invalid severity level: %s", severity)
	}
	
	alert := &Alert{
		ID:          uuid.New(),
		Name:        strings.TrimSpace(name),
		Description: strings.TrimSpace(description),
		Severity:    severity,
		Status:      "active",
		Labels:      labels,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	if err := s.repo.CreateAlert(ctx, alert); err != nil {
		return nil, fmt.Errorf("failed to create alert: %w", err)
	}
	
	return alert, nil
}

// ResolveAlert resolves an active alert
func (s *Service) ResolveAlert(ctx context.Context, alertID uuid.UUID) error {
	// This would typically fetch the alert first, then update it
	// For now, we'll create a minimal implementation
	now := time.Now()
	alert := &Alert{
		ID:         alertID,
		Status:     "resolved",
		UpdatedAt:  now,
		ResolvedAt: &now,
	}
	
	if err := s.repo.UpdateAlert(ctx, alert); err != nil {
		return fmt.Errorf("failed to resolve alert: %w", err)
	}
	
	return nil
}

// CreateAlertRule creates a new alert rule
func (s *Service) CreateAlertRule(ctx context.Context, name, query, condition string, threshold float64, duration, severity string) (*AlertRule, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("alert rule name cannot be empty")
	}
	
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}
	
	if !s.isValidCondition(condition) {
		return nil, fmt.Errorf("invalid condition: %s", condition)
	}
	
	if !s.isValidDuration(duration) {
		return nil, fmt.Errorf("invalid duration format: %s", duration)
	}
	
	if !s.isValidSeverity(severity) {
		return nil, fmt.Errorf("invalid severity level: %s", severity)
	}
	
	rule := &AlertRule{
		ID:        uuid.New(),
		Name:      strings.TrimSpace(name),
		Query:     strings.TrimSpace(query),
		Condition: condition,
		Threshold: threshold,
		Duration:  duration,
		Severity:  severity,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	if err := s.repo.CreateAlertRule(ctx, rule); err != nil {
		return nil, fmt.Errorf("failed to create alert rule: %w", err)
	}
	
	return rule, nil
}

// EvaluateAlertRules evaluates all active alert rules and creates alerts if conditions are met
func (s *Service) EvaluateAlertRules(ctx context.Context) ([]*Alert, error) {
	rules, err := s.repo.GetAlertRules(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get alert rules: %w", err)
	}
	
	var alerts []*Alert
	
	for _, rule := range rules {
		if !rule.IsActive {
			continue
		}
		
		// Query Prometheus for the metric value
		value, err := s.prometheus.Query(rule.Query)
		if err != nil {
			// Log error but continue with other rules
			continue
		}
		
		// Evaluate condition
		if s.evaluateCondition(value, rule.Condition, rule.Threshold) {
			alert, err := s.CreateAlert(ctx, 
				fmt.Sprintf("Alert: %s", rule.Name),
				fmt.Sprintf("Rule '%s' triggered: %s %s %.2f (current: %.2f)", 
					rule.Name, rule.Query, rule.Condition, rule.Threshold, value),
				rule.Severity,
				map[string]string{
					"rule_id": rule.ID.String(),
					"query":   rule.Query,
				},
			)
			if err != nil {
				// Log error but continue
				continue
			}
			
			alerts = append(alerts, alert)
		}
	}
	
	return alerts, nil
}

// CreateDashboard creates a new monitoring dashboard
func (s *Service) CreateDashboard(ctx context.Context, name, description, config string, isPublic bool, createdBy uuid.UUID) (*Dashboard, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("dashboard name cannot be empty")
	}
	
	if createdBy == uuid.Nil {
		return nil, fmt.Errorf("created by user ID cannot be empty")
	}
	
	// Validate JSON configuration
	if strings.TrimSpace(config) != "" {
		var configMap map[string]interface{}
		if err := json.Unmarshal([]byte(config), &configMap); err != nil {
			return nil, fmt.Errorf("invalid JSON configuration: %w", err)
		}
	}
	
	dashboard := &Dashboard{
		ID:          uuid.New(),
		Name:        strings.TrimSpace(name),
		Description: strings.TrimSpace(description),
		Config:      config,
		IsPublic:    isPublic,
		CreatedBy:   createdBy,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	if err := s.repo.CreateDashboard(ctx, dashboard); err != nil {
		return nil, fmt.Errorf("failed to create dashboard: %w", err)
	}
	
	return dashboard, nil
}

// RecordPerformanceMetrics records system performance metrics
func (s *Service) RecordPerformanceMetrics(ctx context.Context, metrics *PerformanceMetrics) error {
	if metrics == nil {
		return fmt.Errorf("metrics cannot be nil")
	}
	
	// Store in database
	if err := s.repo.StorePerformanceMetrics(ctx, metrics); err != nil {
		return fmt.Errorf("failed to store performance metrics: %w", err)
	}
	
	// Send key metrics to Prometheus
	labels := map[string]string{"component": "pool"}
	
	_ = s.prometheus.RecordGauge("pool_cpu_usage", labels, metrics.CPUUsage)
	_ = s.prometheus.RecordGauge("pool_memory_usage", labels, metrics.MemoryUsage)
	_ = s.prometheus.RecordGauge("pool_disk_usage", labels, metrics.DiskUsage)
	_ = s.prometheus.RecordGauge("pool_active_miners", labels, float64(metrics.ActiveMiners))
	_ = s.prometheus.RecordGauge("pool_total_hashrate", labels, metrics.TotalHashrate)
	_ = s.prometheus.RecordGauge("pool_shares_per_second", labels, metrics.SharesPerSecond)
	_ = s.prometheus.RecordCounter("pool_blocks_found_total", labels, float64(metrics.BlocksFound))
	_ = s.prometheus.RecordGauge("pool_uptime_seconds", labels, metrics.Uptime)
	
	return nil
}

// RecordMinerMetrics records individual miner metrics
func (s *Service) RecordMinerMetrics(ctx context.Context, metrics *MinerMetrics) error {
	if metrics == nil {
		return fmt.Errorf("metrics cannot be nil")
	}
	
	if metrics.MinerID == uuid.Nil {
		return fmt.Errorf("miner ID cannot be empty")
	}
	
	// Store in database
	if err := s.repo.StoreMinerMetrics(ctx, metrics); err != nil {
		return fmt.Errorf("failed to store miner metrics: %w", err)
	}
	
	// Send to Prometheus
	labels := map[string]string{"miner_id": metrics.MinerID.String()}
	
	_ = s.prometheus.RecordGauge("miner_hashrate", labels, metrics.Hashrate)
	_ = s.prometheus.RecordCounter("miner_shares_submitted_total", labels, float64(metrics.SharesSubmitted))
	_ = s.prometheus.RecordCounter("miner_shares_accepted_total", labels, float64(metrics.SharesAccepted))
	_ = s.prometheus.RecordCounter("miner_shares_rejected_total", labels, float64(metrics.SharesRejected))
	_ = s.prometheus.RecordGauge("miner_difficulty", labels, metrics.Difficulty)
	_ = s.prometheus.RecordGauge("miner_earnings", labels, metrics.Earnings)
	
	if metrics.IsOnline {
		_ = s.prometheus.RecordGauge("miner_online", labels, 1)
	} else {
		_ = s.prometheus.RecordGauge("miner_online", labels, 0)
	}
	
	return nil
}

// RecordPoolMetrics records pool-wide metrics
func (s *Service) RecordPoolMetrics(ctx context.Context, metrics *PoolMetrics) error {
	if metrics == nil {
		return fmt.Errorf("metrics cannot be nil")
	}
	
	// Store in database
	if err := s.repo.StorePoolMetrics(ctx, metrics); err != nil {
		return fmt.Errorf("failed to store pool metrics: %w", err)
	}
	
	// Send to Prometheus
	labels := map[string]string{"pool": "main"}
	
	_ = s.prometheus.RecordGauge("pool_total_hashrate", labels, metrics.TotalHashrate)
	_ = s.prometheus.RecordGauge("pool_active_miners", labels, float64(metrics.ActiveMiners))
	_ = s.prometheus.RecordCounter("pool_total_shares", labels, float64(metrics.TotalShares))
	_ = s.prometheus.RecordCounter("pool_valid_shares", labels, float64(metrics.ValidShares))
	_ = s.prometheus.RecordCounter("pool_invalid_shares", labels, float64(metrics.InvalidShares))
	_ = s.prometheus.RecordCounter("pool_blocks_found_total", labels, float64(metrics.BlocksFound))
	_ = s.prometheus.RecordGauge("pool_network_difficulty", labels, metrics.NetworkDifficulty)
	_ = s.prometheus.RecordGauge("pool_difficulty", labels, metrics.PoolDifficulty)
	_ = s.prometheus.RecordGauge("pool_network_hashrate", labels, metrics.NetworkHashrate)
	_ = s.prometheus.RecordGauge("pool_efficiency", labels, metrics.PoolEfficiency)
	_ = s.prometheus.RecordGauge("pool_luck", labels, metrics.Luck)
	
	return nil
}

// GetMetrics retrieves metrics for a given time range
func (s *Service) GetMetrics(ctx context.Context, name string, start, end time.Time) ([]*Metric, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("metric name cannot be empty")
	}
	
	if end.Before(start) {
		return nil, fmt.Errorf("end time must be after start time")
	}
	
	metrics, err := s.repo.GetMetrics(ctx, name, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}
	
	return metrics, nil
}

// validateMetric validates a metric before recording
func (s *Service) validateMetric(metric *Metric) error {
	if strings.TrimSpace(metric.Name) == "" {
		return fmt.Errorf("metric name cannot be empty")
	}
	
	validTypes := map[string]bool{
		"counter":   true,
		"gauge":     true,
		"histogram": true,
		"summary":   true,
	}
	
	if !validTypes[metric.Type] {
		return fmt.Errorf("unsupported metric type: %s", metric.Type)
	}
	
	return nil
}

// isValidSeverity checks if severity level is valid
func (s *Service) isValidSeverity(severity string) bool {
	validSeverities := map[string]bool{
		"info":     true,
		"warning":  true,
		"error":    true,
		"critical": true,
	}
	
	return validSeverities[severity]
}

// isValidCondition checks if condition is valid
func (s *Service) isValidCondition(condition string) bool {
	validConditions := map[string]bool{
		">":  true,
		"<":  true,
		">=": true,
		"<=": true,
		"==": true,
		"!=": true,
	}
	
	return validConditions[condition]
}

// isValidDuration checks if duration format is valid
func (s *Service) isValidDuration(duration string) bool {
	// Simple regex for duration format like "5m", "1h", "30s"
	matched, _ := regexp.MatchString(`^\d+[smh]$`, duration)
	return matched
}

// evaluateCondition evaluates if a condition is met
func (s *Service) evaluateCondition(value float64, condition string, threshold float64) bool {
	switch condition {
	case ">":
		return value > threshold
	case "<":
		return value < threshold
	case ">=":
		return value >= threshold
	case "<=":
		return value <= threshold
	case "==":
		return value == threshold
	case "!=":
		return value != threshold
	default:
		return false
	}
}