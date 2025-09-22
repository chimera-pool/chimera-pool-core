package monitoring

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// PostgreSQLRepository implements the Repository interface using PostgreSQL
type PostgreSQLRepository struct {
	db *sqlx.DB
}

// NewPostgreSQLRepository creates a new PostgreSQL repository
func NewPostgreSQLRepository(db *sqlx.DB) *PostgreSQLRepository {
	return &PostgreSQLRepository{
		db: db,
	}
}

// StoreMetric stores a metric in the database
func (r *PostgreSQLRepository) StoreMetric(ctx context.Context, metric *Metric) error {
	query := `
		INSERT INTO metrics (name, value, labels, timestamp, type)
		VALUES ($1, $2, $3, $4, $5)
	`
	
	labelsJSON, err := json.Marshal(metric.Labels)
	if err != nil {
		return fmt.Errorf("failed to marshal labels: %w", err)
	}
	
	_, err = r.db.ExecContext(ctx, query, metric.Name, metric.Value, labelsJSON, metric.Timestamp, metric.Type)
	if err != nil {
		return fmt.Errorf("failed to store metric: %w", err)
	}
	
	return nil
}

// GetMetrics retrieves metrics from the database
func (r *PostgreSQLRepository) GetMetrics(ctx context.Context, name string, start, end time.Time) ([]*Metric, error) {
	query := `
		SELECT name, value, labels, timestamp, type
		FROM metrics
		WHERE name = $1 AND timestamp BETWEEN $2 AND $3
		ORDER BY timestamp ASC
	`
	
	rows, err := r.db.QueryContext(ctx, query, name, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query metrics: %w", err)
	}
	defer rows.Close()
	
	var metrics []*Metric
	for rows.Next() {
		var metric Metric
		var labelsJSON []byte
		
		err := rows.Scan(&metric.Name, &metric.Value, &labelsJSON, &metric.Timestamp, &metric.Type)
		if err != nil {
			return nil, fmt.Errorf("failed to scan metric: %w", err)
		}
		
		if err := json.Unmarshal(labelsJSON, &metric.Labels); err != nil {
			return nil, fmt.Errorf("failed to unmarshal labels: %w", err)
		}
		
		metrics = append(metrics, &metric)
	}
	
	return metrics, nil
}

// CreateAlert creates a new alert
func (r *PostgreSQLRepository) CreateAlert(ctx context.Context, alert *Alert) error {
	query := `
		INSERT INTO alerts (id, name, description, severity, status, labels, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	
	labelsJSON, err := json.Marshal(alert.Labels)
	if err != nil {
		return fmt.Errorf("failed to marshal labels: %w", err)
	}
	
	_, err = r.db.ExecContext(ctx, query, alert.ID, alert.Name, alert.Description, alert.Severity, alert.Status, labelsJSON, alert.CreatedAt, alert.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create alert: %w", err)
	}
	
	return nil
}

// UpdateAlert updates an existing alert
func (r *PostgreSQLRepository) UpdateAlert(ctx context.Context, alert *Alert) error {
	query := `
		UPDATE alerts
		SET status = $2, updated_at = $3, resolved_at = $4
		WHERE id = $1
	`
	
	_, err := r.db.ExecContext(ctx, query, alert.ID, alert.Status, alert.UpdatedAt, alert.ResolvedAt)
	if err != nil {
		return fmt.Errorf("failed to update alert: %w", err)
	}
	
	return nil
}

// GetActiveAlerts retrieves all active alerts
func (r *PostgreSQLRepository) GetActiveAlerts(ctx context.Context) ([]*Alert, error) {
	query := `
		SELECT id, name, description, severity, status, labels, created_at, updated_at, resolved_at
		FROM alerts
		WHERE status = 'active'
		ORDER BY created_at DESC
	`
	
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query active alerts: %w", err)
	}
	defer rows.Close()
	
	var alerts []*Alert
	for rows.Next() {
		var alert Alert
		var labelsJSON []byte
		
		err := rows.Scan(&alert.ID, &alert.Name, &alert.Description, &alert.Severity, &alert.Status, &labelsJSON, &alert.CreatedAt, &alert.UpdatedAt, &alert.ResolvedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan alert: %w", err)
		}
		
		if err := json.Unmarshal(labelsJSON, &alert.Labels); err != nil {
			return nil, fmt.Errorf("failed to unmarshal labels: %w", err)
		}
		
		alerts = append(alerts, &alert)
	}
	
	return alerts, nil
}

// CreateAlertRule creates a new alert rule
func (r *PostgreSQLRepository) CreateAlertRule(ctx context.Context, rule *AlertRule) error {
	query := `
		INSERT INTO alert_rules (id, name, query, condition, threshold, duration, severity, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	
	_, err := r.db.ExecContext(ctx, query, rule.ID, rule.Name, rule.Query, rule.Condition, rule.Threshold, rule.Duration, rule.Severity, rule.IsActive, rule.CreatedAt, rule.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create alert rule: %w", err)
	}
	
	return nil
}

// GetAlertRules retrieves all alert rules
func (r *PostgreSQLRepository) GetAlertRules(ctx context.Context) ([]*AlertRule, error) {
	query := `
		SELECT id, name, query, condition, threshold, duration, severity, is_active, created_at, updated_at
		FROM alert_rules
		ORDER BY created_at DESC
	`
	
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query alert rules: %w", err)
	}
	defer rows.Close()
	
	var rules []*AlertRule
	for rows.Next() {
		var rule AlertRule
		
		err := rows.Scan(&rule.ID, &rule.Name, &rule.Query, &rule.Condition, &rule.Threshold, &rule.Duration, &rule.Severity, &rule.IsActive, &rule.CreatedAt, &rule.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan alert rule: %w", err)
		}
		
		rules = append(rules, &rule)
	}
	
	return rules, nil
}

// CreateDashboard creates a new dashboard
func (r *PostgreSQLRepository) CreateDashboard(ctx context.Context, dashboard *Dashboard) error {
	query := `
		INSERT INTO dashboards (id, name, description, config, is_public, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	
	_, err := r.db.ExecContext(ctx, query, dashboard.ID, dashboard.Name, dashboard.Description, dashboard.Config, dashboard.IsPublic, dashboard.CreatedBy, dashboard.CreatedAt, dashboard.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create dashboard: %w", err)
	}
	
	return nil
}

// GetDashboard retrieves a dashboard by ID
func (r *PostgreSQLRepository) GetDashboard(ctx context.Context, id uuid.UUID) (*Dashboard, error) {
	query := `
		SELECT id, name, description, config, is_public, created_by, created_at, updated_at
		FROM dashboards
		WHERE id = $1
	`
	
	var dashboard Dashboard
	err := r.db.GetContext(ctx, &dashboard, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get dashboard: %w", err)
	}
	
	return &dashboard, nil
}

// StorePerformanceMetrics stores performance metrics
func (r *PostgreSQLRepository) StorePerformanceMetrics(ctx context.Context, metrics *PerformanceMetrics) error {
	query := `
		INSERT INTO performance_metrics (
			timestamp, cpu_usage, memory_usage, disk_usage, network_in, network_out,
			active_miners, total_hashrate, shares_per_second, blocks_found, uptime
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	
	_, err := r.db.ExecContext(ctx, query,
		metrics.Timestamp, metrics.CPUUsage, metrics.MemoryUsage, metrics.DiskUsage,
		metrics.NetworkIn, metrics.NetworkOut, metrics.ActiveMiners, metrics.TotalHashrate,
		metrics.SharesPerSecond, metrics.BlocksFound, metrics.Uptime,
	)
	if err != nil {
		return fmt.Errorf("failed to store performance metrics: %w", err)
	}
	
	return nil
}

// GetPerformanceMetrics retrieves performance metrics for a time range
func (r *PostgreSQLRepository) GetPerformanceMetrics(ctx context.Context, start, end time.Time) ([]*PerformanceMetrics, error) {
	query := `
		SELECT timestamp, cpu_usage, memory_usage, disk_usage, network_in, network_out,
			   active_miners, total_hashrate, shares_per_second, blocks_found, uptime
		FROM performance_metrics
		WHERE timestamp BETWEEN $1 AND $2
		ORDER BY timestamp ASC
	`
	
	var metrics []*PerformanceMetrics
	err := r.db.SelectContext(ctx, &metrics, query, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get performance metrics: %w", err)
	}
	
	return metrics, nil
}

// StoreMinerMetrics stores miner metrics
func (r *PostgreSQLRepository) StoreMinerMetrics(ctx context.Context, metrics *MinerMetrics) error {
	query := `
		INSERT INTO miner_metrics (
			miner_id, timestamp, hashrate, shares_submitted, shares_accepted, shares_rejected,
			last_seen, is_online, difficulty, earnings
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	
	_, err := r.db.ExecContext(ctx, query,
		metrics.MinerID, metrics.Timestamp, metrics.Hashrate, metrics.SharesSubmitted,
		metrics.SharesAccepted, metrics.SharesRejected, metrics.LastSeen, metrics.IsOnline,
		metrics.Difficulty, metrics.Earnings,
	)
	if err != nil {
		return fmt.Errorf("failed to store miner metrics: %w", err)
	}
	
	return nil
}

// GetMinerMetrics retrieves miner metrics for a specific miner and time range
func (r *PostgreSQLRepository) GetMinerMetrics(ctx context.Context, minerID uuid.UUID, start, end time.Time) ([]*MinerMetrics, error) {
	query := `
		SELECT miner_id, timestamp, hashrate, shares_submitted, shares_accepted, shares_rejected,
			   last_seen, is_online, difficulty, earnings
		FROM miner_metrics
		WHERE miner_id = $1 AND timestamp BETWEEN $2 AND $3
		ORDER BY timestamp ASC
	`
	
	var metrics []*MinerMetrics
	err := r.db.SelectContext(ctx, &metrics, query, minerID, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get miner metrics: %w", err)
	}
	
	return metrics, nil
}

// StorePoolMetrics stores pool metrics
func (r *PostgreSQLRepository) StorePoolMetrics(ctx context.Context, metrics *PoolMetrics) error {
	query := `
		INSERT INTO pool_metrics (
			timestamp, total_hashrate, active_miners, total_shares, valid_shares, invalid_shares,
			blocks_found, network_difficulty, pool_difficulty, network_hashrate, pool_efficiency, luck
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	
	_, err := r.db.ExecContext(ctx, query,
		metrics.Timestamp, metrics.TotalHashrate, metrics.ActiveMiners, metrics.TotalShares,
		metrics.ValidShares, metrics.InvalidShares, metrics.BlocksFound, metrics.NetworkDifficulty,
		metrics.PoolDifficulty, metrics.NetworkHashrate, metrics.PoolEfficiency, metrics.Luck,
	)
	if err != nil {
		return fmt.Errorf("failed to store pool metrics: %w", err)
	}
	
	return nil
}

// GetPoolMetrics retrieves pool metrics for a time range
func (r *PostgreSQLRepository) GetPoolMetrics(ctx context.Context, start, end time.Time) ([]*PoolMetrics, error) {
	query := `
		SELECT timestamp, total_hashrate, active_miners, total_shares, valid_shares, invalid_shares,
			   blocks_found, network_difficulty, pool_difficulty, network_hashrate, pool_efficiency, luck
		FROM pool_metrics
		WHERE timestamp BETWEEN $1 AND $2
		ORDER BY timestamp ASC
	`
	
	var metrics []*PoolMetrics
	err := r.db.SelectContext(ctx, &metrics, query, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get pool metrics: %w", err)
	}
	
	return metrics, nil
}

// CreateAlertChannel creates a new alert channel
func (r *PostgreSQLRepository) CreateAlertChannel(ctx context.Context, channel *AlertChannel) error {
	query := `
		INSERT INTO alert_channels (id, name, type, config, is_active)
		VALUES ($1, $2, $3, $4, $5)
	`
	
	configJSON, err := json.Marshal(channel.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	_, err = r.db.ExecContext(ctx, query, channel.ID, channel.Name, channel.Type, configJSON, channel.IsActive)
	if err != nil {
		return fmt.Errorf("failed to create alert channel: %w", err)
	}
	
	return nil
}

// GetAlertChannels retrieves all alert channels
func (r *PostgreSQLRepository) GetAlertChannels(ctx context.Context) ([]*AlertChannel, error) {
	query := `
		SELECT id, name, type, config, is_active
		FROM alert_channels
		WHERE is_active = true
		ORDER BY name ASC
	`
	
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query alert channels: %w", err)
	}
	defer rows.Close()
	
	var channels []*AlertChannel
	for rows.Next() {
		var channel AlertChannel
		var configJSON []byte
		
		err := rows.Scan(&channel.ID, &channel.Name, &channel.Type, &configJSON, &channel.IsActive)
		if err != nil {
			return nil, fmt.Errorf("failed to scan alert channel: %w", err)
		}
		
		if err := json.Unmarshal(configJSON, &channel.Config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config: %w", err)
		}
		
		channels = append(channels, &channel)
	}
	
	return channels, nil
}

// RecordNotification records a notification
func (r *PostgreSQLRepository) RecordNotification(ctx context.Context, notification *Notification) error {
	query := `
		INSERT INTO notifications (id, alert_id, channel_id, status, sent_at, error)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	
	_, err := r.db.ExecContext(ctx, query, notification.ID, notification.AlertID, notification.ChannelID, notification.Status, notification.SentAt, notification.Error)
	if err != nil {
		return fmt.Errorf("failed to record notification: %w", err)
	}
	
	return nil
}

// Custom types for JSON handling
type JSONMap map[string]string

func (j JSONMap) Value() (driver.Value, error) {
	return json.Marshal(j)
}

func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = make(map[string]string)
		return nil
	}
	
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into JSONMap", value)
	}
	
	return json.Unmarshal(bytes, j)
}