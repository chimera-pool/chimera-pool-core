package notifications

import (
	"context"
	"testing"
	"time"
)

// =============================================================================
// DISCORD SENDER TESTS - TDD COMPLIANCE
// =============================================================================

func TestDiscordWebhookSender_Creation(t *testing.T) {
	tests := []struct {
		name      string
		config    DiscordConfig
		available bool
	}{
		{
			name: "creates sender with valid config",
			config: DiscordConfig{
				DefaultWebhookURL: "https://discord.com/api/webhooks/123/abc",
				Username:          "Chimera Pool",
			},
			available: true,
		},
		{
			name: "available even without default webhook (per-user webhooks)",
			config: DiscordConfig{
				Username: "Chimera Pool",
			},
			available: true,
		},
		{
			name: "sets default username when empty",
			config: DiscordConfig{
				DefaultWebhookURL: "https://discord.com/api/webhooks/123/abc",
			},
			available: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := NewDiscordWebhookSender(tt.config)
			if sender == nil {
				t.Fatal("expected sender to be created")
			}
			if sender.IsAvailable() != tt.available {
				t.Errorf("expected available=%v, got %v", tt.available, sender.IsAvailable())
			}
			if sender.Channel() != ChannelDiscord {
				t.Errorf("expected channel=%v, got %v", ChannelDiscord, sender.Channel())
			}
		})
	}
}

func TestDiscordWebhookSender_SendNoWebhook(t *testing.T) {
	sender := NewDiscordWebhookSender(DiscordConfig{})

	alert := &Alert{
		Type:    AlertTypeWorkerOffline,
		Title:   "Test",
		Message: "Test message",
	}

	// Should error when no webhook URL provided
	err := sender.Send(context.Background(), alert, "")
	if err == nil {
		t.Error("expected error when no webhook URL provided")
	}
}

func TestDiscordWebhookSender_SendBatchEmpty(t *testing.T) {
	sender := NewDiscordWebhookSender(DiscordConfig{
		DefaultWebhookURL: "https://discord.com/api/webhooks/123/abc",
	})

	// Empty batch should return nil
	err := sender.SendBatch(context.Background(), []*Alert{}, "")
	if err != nil {
		t.Errorf("expected nil error for empty batch, got %v", err)
	}
}

func TestDiscordWebhookSender_SendBatchSingle(t *testing.T) {
	sender := NewDiscordWebhookSender(DiscordConfig{})

	alerts := []*Alert{
		{Type: AlertTypeWorkerOffline, Title: "Test1", CreatedAt: time.Now()},
	}

	// Should error because no webhook URL
	err := sender.SendBatch(context.Background(), alerts, "")
	if err == nil {
		t.Error("expected error when no webhook URL provided")
	}
}

// =============================================================================
// INTERFACE COMPLIANCE TESTS
// =============================================================================

func TestDiscordWebhookSender_ImplementsNotificationSender(t *testing.T) {
	var _ NotificationSender = (*DiscordWebhookSender)(nil)
}
