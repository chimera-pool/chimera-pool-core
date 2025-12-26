package notifications

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
	"time"
)

// =============================================================================
// EMAIL SENDER IMPLEMENTATION
// =============================================================================

// EmailConfig holds email sender configuration
type EmailConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	UseTLS   bool
	Timeout  time.Duration
}

// SMTPEmailSender implements EmailSender using SMTP
type SMTPEmailSender struct {
	config    EmailConfig
	available bool
}

// NewSMTPEmailSender creates a new SMTP email sender
func NewSMTPEmailSender(config EmailConfig) *SMTPEmailSender {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	return &SMTPEmailSender{
		config:    config,
		available: config.Host != "" && config.From != "",
	}
}

// Channel returns the notification channel
func (s *SMTPEmailSender) Channel() NotificationChannel {
	return ChannelEmail
}

// IsAvailable returns whether the sender is properly configured
func (s *SMTPEmailSender) IsAvailable() bool {
	return s.available
}

// Send sends an email notification
func (s *SMTPEmailSender) Send(ctx context.Context, alert *Alert, destination string) error {
	if !s.available {
		return fmt.Errorf("email sender not configured")
	}

	subject := s.formatSubject(alert)
	body := s.formatBody(alert)

	return s.sendEmail(destination, subject, body)
}

// SendBatch sends multiple alerts in a single email
func (s *SMTPEmailSender) SendBatch(ctx context.Context, alerts []*Alert, destination string) error {
	if len(alerts) == 0 {
		return nil
	}

	if len(alerts) == 1 {
		return s.Send(ctx, alerts[0], destination)
	}

	subject := fmt.Sprintf("Chimera Pool: %d Alerts", len(alerts))
	var body strings.Builder
	body.WriteString("<html><body>")
	body.WriteString("<h2>Chimera Pool Alert Summary</h2>")
	body.WriteString("<ul>")

	for _, alert := range alerts {
		body.WriteString(fmt.Sprintf("<li><strong>%s</strong>: %s</li>", alert.Title, alert.Message))
	}

	body.WriteString("</ul>")
	body.WriteString("</body></html>")

	return s.sendEmail(destination, subject, body.String())
}

// SendWithAttachment sends an email with an attachment
func (s *SMTPEmailSender) SendWithAttachment(ctx context.Context, alert *Alert, destination string, attachment []byte, filename string) error {
	// For simplicity, send without attachment for now
	// Full implementation would use multipart MIME
	return s.Send(ctx, alert, destination)
}

// =============================================================================
// HELPER METHODS
// =============================================================================

func (s *SMTPEmailSender) formatSubject(alert *Alert) string {
	prefix := ""
	switch alert.Severity {
	case SeverityCritical:
		prefix = "🚨 CRITICAL: "
	case SeverityWarning:
		prefix = "⚠️ "
	case SeverityInfo:
		prefix = "ℹ️ "
	}
	return fmt.Sprintf("%sChimera Pool - %s", prefix, alert.Title)
}

func (s *SMTPEmailSender) formatBody(alert *Alert) string {
	var body strings.Builder

	body.WriteString("<html><body style='font-family: Arial, sans-serif;'>")

	// Header with severity color
	headerColor := "#17a2b8" // info blue
	switch alert.Severity {
	case SeverityCritical:
		headerColor = "#dc3545" // red
	case SeverityWarning:
		headerColor = "#ffc107" // yellow
	}

	body.WriteString(fmt.Sprintf("<div style='background-color: %s; color: white; padding: 15px; border-radius: 5px;'>", headerColor))
	body.WriteString(fmt.Sprintf("<h2 style='margin: 0;'>%s</h2>", alert.Title))
	body.WriteString("</div>")

	// Body content
	body.WriteString("<div style='padding: 20px;'>")
	body.WriteString(fmt.Sprintf("<p style='font-size: 16px;'>%s</p>", alert.Message))

	// Metadata table if present
	if len(alert.Metadata) > 0 {
		body.WriteString("<table style='border-collapse: collapse; margin-top: 15px;'>")
		for k, v := range alert.Metadata {
			body.WriteString(fmt.Sprintf("<tr><td style='padding: 5px; border: 1px solid #ddd; font-weight: bold;'>%s</td><td style='padding: 5px; border: 1px solid #ddd;'>%s</td></tr>", k, v))
		}
		body.WriteString("</table>")
	}

	// Worker info if present
	if alert.WorkerName != "" {
		body.WriteString(fmt.Sprintf("<p><strong>Worker:</strong> %s</p>", alert.WorkerName))
	}

	// Timestamp
	body.WriteString(fmt.Sprintf("<p style='color: #666; font-size: 12px;'>Alert generated at: %s</p>", alert.CreatedAt.Format(time.RFC1123)))

	body.WriteString("</div>")

	// Footer
	body.WriteString("<div style='background-color: #f8f9fa; padding: 15px; border-top: 1px solid #ddd; text-align: center;'>")
	body.WriteString("<p style='margin: 0; color: #666;'>Chimera Pool - Mining Made Simple</p>")
	body.WriteString("<p style='margin: 5px 0 0 0; font-size: 12px; color: #999;'>Manage your notification preferences in your dashboard.</p>")
	body.WriteString("</div>")

	body.WriteString("</body></html>")

	return body.String()
}

func (s *SMTPEmailSender) sendEmail(to, subject, htmlBody string) error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	// Build email headers
	headers := make(map[string]string)
	headers["From"] = s.config.From
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"

	var message strings.Builder
	for k, v := range headers {
		message.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	message.WriteString("\r\n")
	message.WriteString(htmlBody)

	// Authentication
	auth := smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)

	// Send with TLS if configured (port 465)
	if s.config.UseTLS || s.config.Port == 465 {
		return s.sendWithTLS(addr, auth, to, message.String())
	}

	// Standard SMTP
	return smtp.SendMail(addr, auth, s.config.From, []string{to}, []byte(message.String()))
}

func (s *SMTPEmailSender) sendWithTLS(addr string, auth smtp.Auth, to string, message string) error {
	tlsConfig := &tls.Config{
		ServerName: s.config.Host,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.config.Host)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("auth failed: %w", err)
	}

	if err = client.Mail(s.config.From); err != nil {
		return fmt.Errorf("mail from failed: %w", err)
	}

	if err = client.Rcpt(to); err != nil {
		return fmt.Errorf("rcpt to failed: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("data failed: %w", err)
	}

	_, err = w.Write([]byte(message))
	if err != nil {
		return fmt.Errorf("write failed: %w", err)
	}

	err = w.Close()
	if err != nil {
		return fmt.Errorf("close failed: %w", err)
	}

	return client.Quit()
}
