package notifications

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// NOTIFICATION SERVICE TESTS (TDD)
// =============================================================================

func TestNotificationService_Creation(t *testing.T) {
	t.Run("creates service with default config", func(t *testing.T) {
		config := DefaultNotificationConfig()
		service := NewNotificationService(config)

		require.NotNil(t, service)
		assert.NotNil(t, service.config)
	})

	t.Run("creates service with custom config", func(t *testing.T) {
		config := &NotificationConfig{
			MaxAlertsPerHour: 20,
			CooldownPeriod:   10 * time.Minute,
		}
		service := NewNotificationService(config)

		require.NotNil(t, service)
		assert.Equal(t, 20, service.config.MaxAlertsPerHour)
	})
}

func TestNotificationService_RegisterSender(t *testing.T) {
	t.Run("registers email sender", func(t *testing.T) {
		service := NewNotificationService(DefaultNotificationConfig())
		sender := &mockEmailSender{channel: ChannelEmail, available: true}

		service.RegisterSender(sender)

		assert.True(t, service.HasSender(ChannelEmail))
	})

	t.Run("registers multiple senders", func(t *testing.T) {
		service := NewNotificationService(DefaultNotificationConfig())

		service.RegisterSender(&mockEmailSender{channel: ChannelEmail, available: true})
		service.RegisterSender(&mockDiscordSender{channel: ChannelDiscord, available: true})

		assert.True(t, service.HasSender(ChannelEmail))
		assert.True(t, service.HasSender(ChannelDiscord))
	})
}

func TestNotificationService_SendAlert(t *testing.T) {
	t.Run("sends alert to user via email", func(t *testing.T) {
		service := NewNotificationService(DefaultNotificationConfig())
		emailSender := &mockEmailSender{channel: ChannelEmail, available: true}
		service.RegisterSender(emailSender)

		prefs := &mockPreferences{
			settings: &UserNotificationSettings{
				UserID:               1,
				Email:                "test@example.com",
				EmailEnabled:         true,
				WorkerOfflineEnabled: true,
			},
		}
		service.SetPreferencesProvider(prefs)

		alert := &Alert{
			ID:       "alert-1",
			Type:     AlertTypeWorkerOffline,
			Severity: SeverityWarning,
			Title:    "Worker Offline",
			Message:  "Your worker 'rig1' is offline",
			UserID:   1,
		}

		results, err := service.SendAlert(context.Background(), alert)

		require.NoError(t, err)
		assert.Len(t, results, 1)
		assert.True(t, results[0].Success)
		assert.Equal(t, ChannelEmail, results[0].Channel)
	})

	t.Run("sends to multiple channels", func(t *testing.T) {
		service := NewNotificationService(DefaultNotificationConfig())
		service.RegisterSender(&mockEmailSender{channel: ChannelEmail, available: true})
		service.RegisterSender(&mockDiscordSender{channel: ChannelDiscord, available: true})

		prefs := &mockPreferences{
			settings: &UserNotificationSettings{
				UserID:               1,
				Email:                "test@example.com",
				DiscordWebhook:       "https://discord.com/api/webhooks/123",
				EmailEnabled:         true,
				DiscordEnabled:       true,
				WorkerOfflineEnabled: true,
			},
		}
		service.SetPreferencesProvider(prefs)

		alert := &Alert{
			ID:       "alert-2",
			Type:     AlertTypeWorkerOffline,
			Severity: SeverityWarning,
			UserID:   1,
		}

		results, err := service.SendAlert(context.Background(), alert)

		require.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("skips disabled alert types", func(t *testing.T) {
		service := NewNotificationService(DefaultNotificationConfig())
		service.RegisterSender(&mockEmailSender{channel: ChannelEmail, available: true})

		prefs := &mockPreferences{
			settings: &UserNotificationSettings{
				UserID:               1,
				Email:                "test@example.com",
				EmailEnabled:         true,
				WorkerOfflineEnabled: false, // Disabled
			},
		}
		service.SetPreferencesProvider(prefs)

		alert := &Alert{
			ID:     "alert-3",
			Type:   AlertTypeWorkerOffline,
			UserID: 1,
		}

		results, err := service.SendAlert(context.Background(), alert)

		require.NoError(t, err)
		assert.Len(t, results, 0) // No notifications sent
	})
}

func TestNotificationService_RateLimiting(t *testing.T) {
	t.Run("respects rate limits", func(t *testing.T) {
		config := &NotificationConfig{
			MaxAlertsPerHour: 2,
			CooldownPeriod:   time.Minute,
			RetryAttempts:    1,
			RetryDelay:       time.Millisecond,
		}
		service := NewNotificationService(config)
		service.RegisterSender(&mockEmailSender{channel: ChannelEmail, available: true})

		prefs := &mockPreferences{
			settings: &UserNotificationSettings{
				UserID:               1,
				Email:                "test@example.com",
				EmailEnabled:         true,
				WorkerOfflineEnabled: true,
				MaxAlertsPerHour:     2,
			},
		}
		service.SetPreferencesProvider(prefs)

		// First should succeed
		alert1 := &Alert{
			ID:     "alert-rate-1",
			Type:   AlertTypeWorkerOffline,
			UserID: 1,
		}
		results1, err := service.SendAlert(context.Background(), alert1)
		require.NoError(t, err)
		assert.Len(t, results1, 1)

		// Second should succeed
		alert2 := &Alert{
			ID:     "alert-rate-2",
			Type:   AlertTypeWorkerOffline,
			UserID: 1,
		}
		results2, err := service.SendAlert(context.Background(), alert2)
		require.NoError(t, err)
		assert.Len(t, results2, 1)

		// Third should be rate limited
		alert3 := &Alert{
			ID:     "alert-rate-3",
			Type:   AlertTypeWorkerOffline,
			UserID: 1,
		}
		results3, err := service.SendAlert(context.Background(), alert3)
		assert.NoError(t, err)
		assert.Len(t, results3, 0) // Rate limited
	})
}

func TestNotificationService_AlertCreation(t *testing.T) {
	t.Run("creates worker offline alert", func(t *testing.T) {
		alert := NewWorkerOfflineAlert(1, 100, "rig1")

		assert.Equal(t, AlertTypeWorkerOffline, alert.Type)
		assert.Equal(t, SeverityWarning, alert.Severity)
		assert.Equal(t, int64(1), alert.UserID)
		assert.Equal(t, int64(100), alert.WorkerID)
		assert.Equal(t, "rig1", alert.WorkerName)
		assert.Contains(t, alert.Title, "Offline")
	})

	t.Run("creates block found alert", func(t *testing.T) {
		alert := NewBlockFoundAlert(12345, 1250000000, "LTC")

		assert.Equal(t, AlertTypeBlockFound, alert.Type)
		assert.Equal(t, SeverityInfo, alert.Severity)
		assert.Contains(t, alert.Message, "12345")
	})

	t.Run("creates payout sent alert", func(t *testing.T) {
		alert := NewPayoutSentAlert(1, 5000000, "ltc1qtest...", "txhash123")

		assert.Equal(t, AlertTypePayoutSent, alert.Type)
		assert.Equal(t, SeverityInfo, alert.Severity)
		assert.Equal(t, int64(1), alert.UserID)
	})
}

func TestNotificationService_GetStats(t *testing.T) {
	t.Run("returns statistics", func(t *testing.T) {
		service := NewNotificationService(DefaultNotificationConfig())
		service.RegisterSender(&mockEmailSender{channel: ChannelEmail, available: true})

		prefs := &mockPreferences{
			settings: &UserNotificationSettings{
				UserID:               1,
				Email:                "test@example.com",
				EmailEnabled:         true,
				WorkerOfflineEnabled: true,
				MaxAlertsPerHour:     100,
			},
		}
		service.SetPreferencesProvider(prefs)

		// Send some alerts
		alert := &Alert{ID: "stat-1", Type: AlertTypeWorkerOffline, UserID: 1}
		service.SendAlert(context.Background(), alert)

		alert.ID = "stat-2"
		alert.Type = AlertTypeBlockFound
		service.SendAlert(context.Background(), alert)

		stats := service.GetStats()

		assert.GreaterOrEqual(t, stats.TotalSent, int64(1))
	})
}

// =============================================================================
// MOCK IMPLEMENTATIONS FOR TESTING
// =============================================================================

type mockEmailSender struct {
	channel   NotificationChannel
	available bool
	sentCount int
}

func (m *mockEmailSender) Channel() NotificationChannel { return m.channel }
func (m *mockEmailSender) IsAvailable() bool            { return m.available }
func (m *mockEmailSender) Send(ctx context.Context, alert *Alert, dest string) error {
	m.sentCount++
	return nil
}
func (m *mockEmailSender) SendBatch(ctx context.Context, alerts []*Alert, dest string) error {
	m.sentCount += len(alerts)
	return nil
}
func (m *mockEmailSender) SendWithAttachment(ctx context.Context, alert *Alert, dest string, attachment []byte, filename string) error {
	return m.Send(ctx, alert, dest)
}

type mockDiscordSender struct {
	channel   NotificationChannel
	available bool
	sentCount int
}

func (m *mockDiscordSender) Channel() NotificationChannel { return m.channel }
func (m *mockDiscordSender) IsAvailable() bool            { return m.available }
func (m *mockDiscordSender) Send(ctx context.Context, alert *Alert, dest string) error {
	m.sentCount++
	return nil
}
func (m *mockDiscordSender) SendBatch(ctx context.Context, alerts []*Alert, dest string) error {
	m.sentCount += len(alerts)
	return nil
}
func (m *mockDiscordSender) SendEmbed(ctx context.Context, alert *Alert, webhookURL string) error {
	return m.Send(ctx, alert, webhookURL)
}

type mockPreferences struct {
	settings *UserNotificationSettings
}

func (m *mockPreferences) GetUserPreferences(ctx context.Context, userID int64) (*UserNotificationSettings, error) {
	if m.settings != nil && m.settings.UserID == userID {
		return m.settings, nil
	}
	return DefaultUserNotificationSettings(userID, ""), nil
}

func (m *mockPreferences) GetUsersForAlert(ctx context.Context, alertType AlertType) ([]UserNotificationSettings, error) {
	if m.settings != nil {
		return []UserNotificationSettings{*m.settings}, nil
	}
	return []UserNotificationSettings{}, nil
}

func (m *mockPreferences) UpdatePreferences(ctx context.Context, userID int64, settings *UserNotificationSettings) error {
	m.settings = settings
	return nil
}
