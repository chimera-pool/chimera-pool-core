package notifications

import (
	"context"
	"time"
)

// =============================================================================
// NOTIFICATION INTERFACES (ISP - Interface Segregation Principle)
// =============================================================================

// AlertType represents different types of alerts
type AlertType string

const (
	AlertTypeWorkerOffline  AlertType = "worker_offline"
	AlertTypeWorkerOnline   AlertType = "worker_online"
	AlertTypeHashrateDrop   AlertType = "hashrate_drop"
	AlertTypeBlockFound     AlertType = "block_found"
	AlertTypePayoutSent     AlertType = "payout_sent"
	AlertTypePayoutFailed   AlertType = "payout_failed"
	AlertTypeLowBalance     AlertType = "low_balance"
	AlertTypePoolDown       AlertType = "pool_down"
	AlertTypeHighRejectRate AlertType = "high_reject_rate"
)

// AlertSeverity represents alert severity levels
type AlertSeverity string

const (
	SeverityInfo     AlertSeverity = "info"
	SeverityWarning  AlertSeverity = "warning"
	SeverityCritical AlertSeverity = "critical"
)

// NotificationChannel represents delivery channels
type NotificationChannel string

const (
	ChannelEmail   NotificationChannel = "email"
	ChannelDiscord NotificationChannel = "discord"
	ChannelSMS     NotificationChannel = "sms"
	ChannelWebhook NotificationChannel = "webhook"
)

// =============================================================================
// CORE INTERFACES
// =============================================================================

// Alert represents an alert to be sent
type Alert struct {
	ID         string            `json:"id"`
	Type       AlertType         `json:"type"`
	Severity   AlertSeverity     `json:"severity"`
	Title      string            `json:"title"`
	Message    string            `json:"message"`
	UserID     int64             `json:"user_id,omitempty"`
	WorkerID   int64             `json:"worker_id,omitempty"`
	WorkerName string            `json:"worker_name,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
	ResolvedAt *time.Time        `json:"resolved_at,omitempty"`
	IsResolved bool              `json:"is_resolved"`
}

// NotificationSender sends notifications through a specific channel (ISP)
type NotificationSender interface {
	Channel() NotificationChannel
	Send(ctx context.Context, alert *Alert, destination string) error
	SendBatch(ctx context.Context, alerts []*Alert, destination string) error
	IsAvailable() bool
}

// EmailSender sends email notifications (ISP)
type EmailSender interface {
	NotificationSender
	SendWithAttachment(ctx context.Context, alert *Alert, destination string, attachment []byte, filename string) error
}

// DiscordSender sends Discord webhook notifications (ISP)
type DiscordSender interface {
	NotificationSender
	SendEmbed(ctx context.Context, alert *Alert, webhookURL string) error
}

// UserAlertPreferences retrieves user notification preferences (ISP)
type UserAlertPreferences interface {
	GetUserPreferences(ctx context.Context, userID int64) (*UserNotificationSettings, error)
	GetUsersForAlert(ctx context.Context, alertType AlertType) ([]UserNotificationSettings, error)
	UpdatePreferences(ctx context.Context, userID int64, settings *UserNotificationSettings) error
}

// AlertRepository stores and retrieves alerts (ISP)
type AlertRepository interface {
	SaveAlert(ctx context.Context, alert *Alert) error
	GetAlert(ctx context.Context, alertID string) (*Alert, error)
	GetUserAlerts(ctx context.Context, userID int64, limit int) ([]*Alert, error)
	GetRecentAlerts(ctx context.Context, alertType AlertType, duration time.Duration) ([]*Alert, error)
	MarkResolved(ctx context.Context, alertID string) error
}

// RateLimiter controls alert frequency (ISP)
type RateLimiter interface {
	Allow(ctx context.Context, userID int64, alertType AlertType) bool
	GetCooldown(ctx context.Context, userID int64, alertType AlertType) time.Duration
	Reset(ctx context.Context, userID int64, alertType AlertType)
}

// =============================================================================
// DATA STRUCTURES
// =============================================================================

// UserNotificationSettings holds user's notification preferences
type UserNotificationSettings struct {
	UserID         int64  `json:"user_id" db:"user_id"`
	Email          string `json:"email" db:"email"`
	DiscordWebhook string `json:"discord_webhook,omitempty" db:"discord_webhook"`
	PhoneNumber    string `json:"phone_number,omitempty" db:"phone_number"`

	// Per-alert-type settings
	WorkerOfflineEnabled bool `json:"worker_offline_enabled" db:"worker_offline_enabled"`
	WorkerOfflineDelay   int  `json:"worker_offline_delay" db:"worker_offline_delay"` // minutes
	HashrateDropEnabled  bool `json:"hashrate_drop_enabled" db:"hashrate_drop_enabled"`
	HashrateDropPercent  int  `json:"hashrate_drop_percent" db:"hashrate_drop_percent"`
	BlockFoundEnabled    bool `json:"block_found_enabled" db:"block_found_enabled"`
	PayoutEnabled        bool `json:"payout_enabled" db:"payout_enabled"`

	// Channel preferences
	EmailEnabled   bool `json:"email_enabled" db:"email_enabled"`
	DiscordEnabled bool `json:"discord_enabled" db:"discord_enabled"`
	SMSEnabled     bool `json:"sms_enabled" db:"sms_enabled"`

	// Rate limiting
	MaxAlertsPerHour int  `json:"max_alerts_per_hour" db:"max_alerts_per_hour"`
	QuietHoursStart  *int `json:"quiet_hours_start,omitempty" db:"quiet_hours_start"`
	QuietHoursEnd    *int `json:"quiet_hours_end,omitempty" db:"quiet_hours_end"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// DefaultUserNotificationSettings returns sensible defaults
func DefaultUserNotificationSettings(userID int64, email string) *UserNotificationSettings {
	return &UserNotificationSettings{
		UserID:               userID,
		Email:                email,
		WorkerOfflineEnabled: true,
		WorkerOfflineDelay:   5, // 5 minutes
		HashrateDropEnabled:  true,
		HashrateDropPercent:  50, // 50% drop
		BlockFoundEnabled:    true,
		PayoutEnabled:        true,
		EmailEnabled:         true,
		DiscordEnabled:       false,
		SMSEnabled:           false,
		MaxAlertsPerHour:     10,
	}
}

// NotificationResult represents the result of sending a notification
type NotificationResult struct {
	AlertID     string              `json:"alert_id"`
	Channel     NotificationChannel `json:"channel"`
	Destination string              `json:"destination"`
	Success     bool                `json:"success"`
	Error       string              `json:"error,omitempty"`
	SentAt      time.Time           `json:"sent_at"`
}

// AlertStats holds notification statistics
type AlertStats struct {
	TotalSent   int64                         `json:"total_sent"`
	TotalFailed int64                         `json:"total_failed"`
	ByType      map[AlertType]int64           `json:"by_type"`
	ByChannel   map[NotificationChannel]int64 `json:"by_channel"`
	LastAlertAt *time.Time                    `json:"last_alert_at,omitempty"`
}
