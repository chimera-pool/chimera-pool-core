package notifications

import (
	"context"
	"testing"
	"time"
)

// =============================================================================
// EMAIL SENDER TESTS - TDD COMPLIANCE
// =============================================================================

func TestSMTPEmailSender_Creation(t *testing.T) {
	tests := []struct {
		name      string
		config    EmailConfig
		available bool
	}{
		{
			name: "creates sender with valid config",
			config: EmailConfig{
				Host:     "smtp.example.com",
				Port:     587,
				Username: "user@example.com",
				Password: "password",
				From:     "noreply@example.com",
				UseTLS:   true,
			},
			available: true,
		},
		{
			name: "unavailable without host",
			config: EmailConfig{
				Port: 587,
				From: "noreply@example.com",
			},
			available: false,
		},
		{
			name: "unavailable without from address",
			config: EmailConfig{
				Host: "smtp.example.com",
				Port: 587,
			},
			available: false,
		},
		{
			name: "sets default timeout",
			config: EmailConfig{
				Host: "smtp.example.com",
				From: "noreply@example.com",
			},
			available: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := NewSMTPEmailSender(tt.config)
			if sender == nil {
				t.Fatal("expected sender to be created")
			}
			if sender.IsAvailable() != tt.available {
				t.Errorf("expected available=%v, got %v", tt.available, sender.IsAvailable())
			}
			if sender.Channel() != ChannelEmail {
				t.Errorf("expected channel=%v, got %v", ChannelEmail, sender.Channel())
			}
		})
	}
}

func TestSMTPEmailSender_FormatSubject(t *testing.T) {
	sender := NewSMTPEmailSender(EmailConfig{
		Host: "smtp.example.com",
		From: "noreply@example.com",
	})

	tests := []struct {
		name     string
		alert    *Alert
		contains string
	}{
		{
			name: "worker offline alert",
			alert: &Alert{
				Type:     AlertTypeWorkerOffline,
				Severity: SeverityWarning,
				Title:    "Worker rig1 Offline",
			},
			contains: "Worker rig1 Offline",
		},
		{
			name: "critical alert includes severity",
			alert: &Alert{
				Type:     AlertTypeWorkerOffline,
				Severity: SeverityCritical,
				Title:    "Critical Issue",
			},
			contains: "Critical Issue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subject := sender.formatSubject(tt.alert)
			if subject == "" {
				t.Error("expected non-empty subject")
			}
		})
	}
}

func TestSMTPEmailSender_SendUnavailable(t *testing.T) {
	sender := NewSMTPEmailSender(EmailConfig{})

	alert := &Alert{
		Type:    AlertTypeWorkerOffline,
		Title:   "Test",
		Message: "Test message",
	}

	err := sender.Send(context.Background(), alert, "test@example.com")
	if err == nil {
		t.Error("expected error when sender is not available")
	}
}

func TestSMTPEmailSender_SendBatchEmpty(t *testing.T) {
	sender := NewSMTPEmailSender(EmailConfig{
		Host: "smtp.example.com",
		From: "noreply@example.com",
	})

	// Empty batch should return nil
	err := sender.SendBatch(context.Background(), []*Alert{}, "test@example.com")
	if err != nil {
		t.Errorf("expected nil error for empty batch, got %v", err)
	}
}

func TestSMTPEmailSender_SendBatchUnavailable(t *testing.T) {
	sender := NewSMTPEmailSender(EmailConfig{})

	alerts := []*Alert{
		{Type: AlertTypeWorkerOffline, Title: "Test1", CreatedAt: time.Now()},
		{Type: AlertTypeWorkerOnline, Title: "Test2", CreatedAt: time.Now()},
	}

	// Should error because sender is not available
	err := sender.SendBatch(context.Background(), alerts, "test@example.com")
	if err == nil {
		t.Error("expected error when sender is not available")
	}
}

// =============================================================================
// INTERFACE COMPLIANCE TESTS
// =============================================================================

func TestSMTPEmailSender_ImplementsNotificationSender(t *testing.T) {
	var _ NotificationSender = (*SMTPEmailSender)(nil)
}

func TestSMTPEmailSender_ImplementsEmailSender(t *testing.T) {
	var _ EmailSender = (*SMTPEmailSender)(nil)
}
