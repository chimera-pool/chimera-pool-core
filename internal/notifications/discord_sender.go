package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// =============================================================================
// DISCORD WEBHOOK SENDER IMPLEMENTATION
// =============================================================================

// DiscordConfig holds Discord sender configuration
type DiscordConfig struct {
	DefaultWebhookURL string
	Username          string
	AvatarURL         string
	Timeout           time.Duration
}

// DiscordWebhookSender implements DiscordSender using Discord webhooks
type DiscordWebhookSender struct {
	config     DiscordConfig
	httpClient *http.Client
	available  bool
}

// NewDiscordWebhookSender creates a new Discord webhook sender
func NewDiscordWebhookSender(config DiscordConfig) *DiscordWebhookSender {
	if config.Timeout == 0 {
		config.Timeout = 10 * time.Second
	}
	if config.Username == "" {
		config.Username = "Chimera Pool"
	}

	return &DiscordWebhookSender{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		available: true,
	}
}

// Channel returns the notification channel
func (s *DiscordWebhookSender) Channel() NotificationChannel {
	return ChannelDiscord
}

// IsAvailable returns whether the sender is properly configured
func (s *DiscordWebhookSender) IsAvailable() bool {
	return s.available
}

// Send sends a Discord notification
func (s *DiscordWebhookSender) Send(ctx context.Context, alert *Alert, webhookURL string) error {
	return s.SendEmbed(ctx, alert, webhookURL)
}

// SendBatch sends multiple alerts in a single Discord message
func (s *DiscordWebhookSender) SendBatch(ctx context.Context, alerts []*Alert, webhookURL string) error {
	if len(alerts) == 0 {
		return nil
	}

	if len(alerts) == 1 {
		return s.Send(ctx, alerts[0], webhookURL)
	}

	// Create a summary embed with multiple fields
	embeds := make([]discordEmbed, 0, len(alerts))
	for _, alert := range alerts {
		embeds = append(embeds, s.createEmbed(alert))
	}

	// Discord has a limit of 10 embeds per message
	if len(embeds) > 10 {
		embeds = embeds[:10]
	}

	payload := discordWebhookPayload{
		Username:  s.config.Username,
		AvatarURL: s.config.AvatarURL,
		Embeds:    embeds,
	}

	return s.sendPayload(ctx, webhookURL, payload)
}

// SendEmbed sends a rich embed message to Discord
func (s *DiscordWebhookSender) SendEmbed(ctx context.Context, alert *Alert, webhookURL string) error {
	if webhookURL == "" {
		webhookURL = s.config.DefaultWebhookURL
	}

	if webhookURL == "" {
		return fmt.Errorf("no webhook URL provided")
	}

	embed := s.createEmbed(alert)

	payload := discordWebhookPayload{
		Username:  s.config.Username,
		AvatarURL: s.config.AvatarURL,
		Embeds:    []discordEmbed{embed},
	}

	return s.sendPayload(ctx, webhookURL, payload)
}

// =============================================================================
// DISCORD PAYLOAD TYPES
// =============================================================================

type discordWebhookPayload struct {
	Username  string         `json:"username,omitempty"`
	AvatarURL string         `json:"avatar_url,omitempty"`
	Content   string         `json:"content,omitempty"`
	Embeds    []discordEmbed `json:"embeds,omitempty"`
}

type discordEmbed struct {
	Title       string              `json:"title,omitempty"`
	Description string              `json:"description,omitempty"`
	URL         string              `json:"url,omitempty"`
	Color       int                 `json:"color,omitempty"`
	Fields      []discordEmbedField `json:"fields,omitempty"`
	Footer      *discordEmbedFooter `json:"footer,omitempty"`
	Timestamp   string              `json:"timestamp,omitempty"`
	Thumbnail   *discordEmbedImage  `json:"thumbnail,omitempty"`
}

type discordEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

type discordEmbedFooter struct {
	Text    string `json:"text"`
	IconURL string `json:"icon_url,omitempty"`
}

type discordEmbedImage struct {
	URL string `json:"url"`
}

// =============================================================================
// HELPER METHODS
// =============================================================================

func (s *DiscordWebhookSender) createEmbed(alert *Alert) discordEmbed {
	embed := discordEmbed{
		Title:       s.getTitle(alert),
		Description: alert.Message,
		Color:       s.getColor(alert.Severity),
		Timestamp:   alert.CreatedAt.Format(time.RFC3339),
		Footer: &discordEmbedFooter{
			Text: "Chimera Pool Alerts",
		},
	}

	// Add fields for metadata
	if alert.WorkerName != "" {
		embed.Fields = append(embed.Fields, discordEmbedField{
			Name:   "Worker",
			Value:  alert.WorkerName,
			Inline: true,
		})
	}

	// Add metadata fields
	for k, v := range alert.Metadata {
		embed.Fields = append(embed.Fields, discordEmbedField{
			Name:   formatFieldName(k),
			Value:  v,
			Inline: true,
		})
	}

	return embed
}

func (s *DiscordWebhookSender) getTitle(alert *Alert) string {
	emoji := ""
	switch alert.Type {
	case AlertTypeWorkerOffline:
		emoji = "🔴"
	case AlertTypeWorkerOnline:
		emoji = "🟢"
	case AlertTypeBlockFound:
		emoji = "🎉"
	case AlertTypePayoutSent:
		emoji = "💰"
	case AlertTypePayoutFailed:
		emoji = "❌"
	case AlertTypeHashrateDrop:
		emoji = "📉"
	case AlertTypeLowBalance:
		emoji = "⚠️"
	case AlertTypePoolDown:
		emoji = "🚨"
	default:
		emoji = "ℹ️"
	}
	return fmt.Sprintf("%s %s", emoji, alert.Title)
}

func (s *DiscordWebhookSender) getColor(severity AlertSeverity) int {
	switch severity {
	case SeverityCritical:
		return 0xDC3545 // Red
	case SeverityWarning:
		return 0xFFC107 // Yellow/Orange
	case SeverityInfo:
		return 0x17A2B8 // Blue
	default:
		return 0x6C757D // Gray
	}
}

func (s *DiscordWebhookSender) sendPayload(ctx context.Context, webhookURL string, payload discordWebhookPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("discord webhook returned status %d", resp.StatusCode)
	}

	return nil
}

func formatFieldName(name string) string {
	// Convert snake_case to Title Case
	words := make([]byte, 0, len(name))
	capitalize := true
	for i := 0; i < len(name); i++ {
		c := name[i]
		if c == '_' {
			words = append(words, ' ')
			capitalize = true
		} else if capitalize {
			if c >= 'a' && c <= 'z' {
				c = c - 'a' + 'A'
			}
			words = append(words, c)
			capitalize = false
		} else {
			words = append(words, c)
		}
	}
	return string(words)
}
