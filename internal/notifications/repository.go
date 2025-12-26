package notifications

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// =============================================================================
// SQL NOTIFICATION REPOSITORY IMPLEMENTATION
// =============================================================================

// SQLNotificationRepository implements notification persistence
type SQLNotificationRepository struct {
	db *sql.DB
}

// NewSQLNotificationRepository creates a new SQL notification repository
func NewSQLNotificationRepository(db *sql.DB) *SQLNotificationRepository {
	return &SQLNotificationRepository{db: db}
}

// =============================================================================
// USER ALERT PREFERENCES IMPLEMENTATION
// =============================================================================

// GetUserPreferences retrieves user notification settings
func (r *SQLNotificationRepository) GetUserPreferences(ctx context.Context, userID int64) (*UserNotificationSettings, error) {
	query := `
		SELECT user_id, email, discord_webhook, phone_number,
		       worker_offline_enabled, worker_offline_delay,
		       hashrate_drop_enabled, hashrate_drop_percent,
		       block_found_enabled, payout_enabled,
		       email_enabled, discord_enabled, sms_enabled,
		       max_alerts_per_hour, quiet_hours_start, quiet_hours_end,
		       created_at, updated_at
		FROM user_notification_settings
		WHERE user_id = $1
	`

	var s UserNotificationSettings
	var discordWebhook, phoneNumber sql.NullString
	var quietStart, quietEnd sql.NullInt32

	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&s.UserID, &s.Email, &discordWebhook, &phoneNumber,
		&s.WorkerOfflineEnabled, &s.WorkerOfflineDelay,
		&s.HashrateDropEnabled, &s.HashrateDropPercent,
		&s.BlockFoundEnabled, &s.PayoutEnabled,
		&s.EmailEnabled, &s.DiscordEnabled, &s.SMSEnabled,
		&s.MaxAlertsPerHour, &quietStart, &quietEnd,
		&s.CreatedAt, &s.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		// Return defaults for user without settings
		return DefaultUserNotificationSettings(userID, ""), nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user preferences: %w", err)
	}

	if discordWebhook.Valid {
		s.DiscordWebhook = discordWebhook.String
	}
	if phoneNumber.Valid {
		s.PhoneNumber = phoneNumber.String
	}
	if quietStart.Valid {
		val := int(quietStart.Int32)
		s.QuietHoursStart = &val
	}
	if quietEnd.Valid {
		val := int(quietEnd.Int32)
		s.QuietHoursEnd = &val
	}

	return &s, nil
}

// GetUsersForAlert retrieves all users who have enabled a specific alert type
func (r *SQLNotificationRepository) GetUsersForAlert(ctx context.Context, alertType AlertType) ([]UserNotificationSettings, error) {
	// Build query based on alert type
	var condition string
	switch alertType {
	case AlertTypeWorkerOffline, AlertTypeWorkerOnline:
		condition = "worker_offline_enabled = true"
	case AlertTypeHashrateDrop:
		condition = "hashrate_drop_enabled = true"
	case AlertTypeBlockFound:
		condition = "block_found_enabled = true"
	case AlertTypePayoutSent, AlertTypePayoutFailed:
		condition = "payout_enabled = true"
	default:
		condition = "true" // All users for unknown types
	}

	query := fmt.Sprintf(`
		SELECT user_id, email, discord_webhook, phone_number,
		       worker_offline_enabled, worker_offline_delay,
		       hashrate_drop_enabled, hashrate_drop_percent,
		       block_found_enabled, payout_enabled,
		       email_enabled, discord_enabled, sms_enabled,
		       max_alerts_per_hour, quiet_hours_start, quiet_hours_end,
		       created_at, updated_at
		FROM user_notification_settings
		WHERE %s AND (email_enabled = true OR discord_enabled = true OR sms_enabled = true)
	`, condition)

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query users for alert: %w", err)
	}
	defer rows.Close()

	var users []UserNotificationSettings
	for rows.Next() {
		var s UserNotificationSettings
		var discordWebhook, phoneNumber sql.NullString
		var quietStart, quietEnd sql.NullInt32

		err := rows.Scan(
			&s.UserID, &s.Email, &discordWebhook, &phoneNumber,
			&s.WorkerOfflineEnabled, &s.WorkerOfflineDelay,
			&s.HashrateDropEnabled, &s.HashrateDropPercent,
			&s.BlockFoundEnabled, &s.PayoutEnabled,
			&s.EmailEnabled, &s.DiscordEnabled, &s.SMSEnabled,
			&s.MaxAlertsPerHour, &quietStart, &quietEnd,
			&s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}

		if discordWebhook.Valid {
			s.DiscordWebhook = discordWebhook.String
		}
		if phoneNumber.Valid {
			s.PhoneNumber = phoneNumber.String
		}
		if quietStart.Valid {
			val := int(quietStart.Int32)
			s.QuietHoursStart = &val
		}
		if quietEnd.Valid {
			val := int(quietEnd.Int32)
			s.QuietHoursEnd = &val
		}

		users = append(users, s)
	}

	return users, rows.Err()
}

// UpdatePreferences updates user notification settings
func (r *SQLNotificationRepository) UpdatePreferences(ctx context.Context, userID int64, settings *UserNotificationSettings) error {
	query := `
		INSERT INTO user_notification_settings (
			user_id, email, discord_webhook, phone_number,
			worker_offline_enabled, worker_offline_delay,
			hashrate_drop_enabled, hashrate_drop_percent,
			block_found_enabled, payout_enabled,
			email_enabled, discord_enabled, sms_enabled,
			max_alerts_per_hour, quiet_hours_start, quiet_hours_end
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (user_id) DO UPDATE SET
			email = EXCLUDED.email,
			discord_webhook = EXCLUDED.discord_webhook,
			phone_number = EXCLUDED.phone_number,
			worker_offline_enabled = EXCLUDED.worker_offline_enabled,
			worker_offline_delay = EXCLUDED.worker_offline_delay,
			hashrate_drop_enabled = EXCLUDED.hashrate_drop_enabled,
			hashrate_drop_percent = EXCLUDED.hashrate_drop_percent,
			block_found_enabled = EXCLUDED.block_found_enabled,
			payout_enabled = EXCLUDED.payout_enabled,
			email_enabled = EXCLUDED.email_enabled,
			discord_enabled = EXCLUDED.discord_enabled,
			sms_enabled = EXCLUDED.sms_enabled,
			max_alerts_per_hour = EXCLUDED.max_alerts_per_hour,
			quiet_hours_start = EXCLUDED.quiet_hours_start,
			quiet_hours_end = EXCLUDED.quiet_hours_end,
			updated_at = NOW()
	`

	var quietStart, quietEnd *int32
	if settings.QuietHoursStart != nil {
		val := int32(*settings.QuietHoursStart)
		quietStart = &val
	}
	if settings.QuietHoursEnd != nil {
		val := int32(*settings.QuietHoursEnd)
		quietEnd = &val
	}

	_, err := r.db.ExecContext(ctx, query,
		userID, settings.Email, nullString(settings.DiscordWebhook), nullString(settings.PhoneNumber),
		settings.WorkerOfflineEnabled, settings.WorkerOfflineDelay,
		settings.HashrateDropEnabled, settings.HashrateDropPercent,
		settings.BlockFoundEnabled, settings.PayoutEnabled,
		settings.EmailEnabled, settings.DiscordEnabled, settings.SMSEnabled,
		settings.MaxAlertsPerHour, quietStart, quietEnd,
	)

	if err != nil {
		return fmt.Errorf("failed to update preferences: %w", err)
	}

	return nil
}

// =============================================================================
// ALERT REPOSITORY IMPLEMENTATION
// =============================================================================

// SaveAlert saves an alert to the database
func (r *SQLNotificationRepository) SaveAlert(ctx context.Context, alert *Alert) error {
	query := `
		INSERT INTO alert_history (
			alert_id, user_id, alert_type, severity, title, message,
			worker_id, worker_name, metadata, is_resolved, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (alert_id) DO UPDATE SET
			is_resolved = EXCLUDED.is_resolved,
			resolved_at = CASE WHEN EXCLUDED.is_resolved THEN NOW() ELSE NULL END
	`

	var metadataJSON []byte
	if alert.Metadata != nil {
		var err error
		metadataJSON, err = json.Marshal(alert.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	var workerID *int64
	if alert.WorkerID != 0 {
		workerID = &alert.WorkerID
	}

	_, err := r.db.ExecContext(ctx, query,
		alert.ID, nullInt64(alert.UserID), alert.Type, alert.Severity,
		alert.Title, alert.Message, workerID, nullString(alert.WorkerName),
		metadataJSON, alert.IsResolved, alert.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save alert: %w", err)
	}

	return nil
}

// GetAlert retrieves an alert by ID
func (r *SQLNotificationRepository) GetAlert(ctx context.Context, alertID string) (*Alert, error) {
	query := `
		SELECT alert_id, user_id, alert_type, severity, title, message,
		       worker_id, worker_name, metadata, is_resolved, created_at, resolved_at
		FROM alert_history
		WHERE alert_id = $1
	`

	var alert Alert
	var userID, workerID sql.NullInt64
	var workerName sql.NullString
	var metadataJSON []byte
	var resolvedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, alertID).Scan(
		&alert.ID, &userID, &alert.Type, &alert.Severity,
		&alert.Title, &alert.Message, &workerID, &workerName,
		&metadataJSON, &alert.IsResolved, &alert.CreatedAt, &resolvedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get alert: %w", err)
	}

	if userID.Valid {
		alert.UserID = userID.Int64
	}
	if workerID.Valid {
		alert.WorkerID = workerID.Int64
	}
	if workerName.Valid {
		alert.WorkerName = workerName.String
	}
	if len(metadataJSON) > 0 {
		json.Unmarshal(metadataJSON, &alert.Metadata)
	}
	if resolvedAt.Valid {
		alert.ResolvedAt = &resolvedAt.Time
	}

	return &alert, nil
}

// GetUserAlerts retrieves recent alerts for a user
func (r *SQLNotificationRepository) GetUserAlerts(ctx context.Context, userID int64, limit int) ([]*Alert, error) {
	query := `
		SELECT alert_id, user_id, alert_type, severity, title, message,
		       worker_id, worker_name, metadata, is_resolved, created_at, resolved_at
		FROM alert_history
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query user alerts: %w", err)
	}
	defer rows.Close()

	return r.scanAlerts(rows)
}

// GetRecentAlerts retrieves recent alerts of a specific type
func (r *SQLNotificationRepository) GetRecentAlerts(ctx context.Context, alertType AlertType, duration time.Duration) ([]*Alert, error) {
	query := `
		SELECT alert_id, user_id, alert_type, severity, title, message,
		       worker_id, worker_name, metadata, is_resolved, created_at, resolved_at
		FROM alert_history
		WHERE alert_type = $1 AND created_at > $2
		ORDER BY created_at DESC
	`

	since := time.Now().Add(-duration)
	rows, err := r.db.QueryContext(ctx, query, alertType, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent alerts: %w", err)
	}
	defer rows.Close()

	return r.scanAlerts(rows)
}

// MarkResolved marks an alert as resolved
func (r *SQLNotificationRepository) MarkResolved(ctx context.Context, alertID string) error {
	query := `
		UPDATE alert_history
		SET is_resolved = true, resolved_at = NOW()
		WHERE alert_id = $1
	`

	_, err := r.db.ExecContext(ctx, query, alertID)
	if err != nil {
		return fmt.Errorf("failed to mark alert resolved: %w", err)
	}

	return nil
}

// =============================================================================
// DELIVERY LOG
// =============================================================================

// LogDelivery logs a notification delivery attempt
func (r *SQLNotificationRepository) LogDelivery(ctx context.Context, result NotificationResult, userID int64) error {
	query := `
		INSERT INTO notification_delivery_log (alert_id, user_id, channel, destination, success, error_message, sent_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.ExecContext(ctx, query,
		result.AlertID, userID, result.Channel, result.Destination,
		result.Success, nullString(result.Error), result.SentAt,
	)

	if err != nil {
		return fmt.Errorf("failed to log delivery: %w", err)
	}

	return nil
}

// =============================================================================
// HELPERS
// =============================================================================

func (r *SQLNotificationRepository) scanAlerts(rows *sql.Rows) ([]*Alert, error) {
	var alerts []*Alert
	for rows.Next() {
		var alert Alert
		var userID, workerID sql.NullInt64
		var workerName sql.NullString
		var metadataJSON []byte
		var resolvedAt sql.NullTime

		err := rows.Scan(
			&alert.ID, &userID, &alert.Type, &alert.Severity,
			&alert.Title, &alert.Message, &workerID, &workerName,
			&metadataJSON, &alert.IsResolved, &alert.CreatedAt, &resolvedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan alert: %w", err)
		}

		if userID.Valid {
			alert.UserID = userID.Int64
		}
		if workerID.Valid {
			alert.WorkerID = workerID.Int64
		}
		if workerName.Valid {
			alert.WorkerName = workerName.String
		}
		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &alert.Metadata)
		}
		if resolvedAt.Valid {
			alert.ResolvedAt = &resolvedAt.Time
		}

		alerts = append(alerts, &alert)
	}

	return alerts, rows.Err()
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func nullInt64(i int64) sql.NullInt64 {
	if i == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: i, Valid: true}
}
