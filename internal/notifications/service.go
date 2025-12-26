package notifications

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// NOTIFICATION SERVICE IMPLEMENTATION
// =============================================================================

// NotificationConfig holds service configuration
type NotificationConfig struct {
	MaxAlertsPerHour int
	CooldownPeriod   time.Duration
	BatchSize        int
	RetryAttempts    int
	RetryDelay       time.Duration
}

// DefaultNotificationConfig returns sensible defaults
func DefaultNotificationConfig() *NotificationConfig {
	return &NotificationConfig{
		MaxAlertsPerHour: 10,
		CooldownPeriod:   5 * time.Minute,
		BatchSize:        10,
		RetryAttempts:    3,
		RetryDelay:       time.Second,
	}
}

// NotificationService orchestrates sending notifications
type NotificationService struct {
	config      *NotificationConfig
	senders     map[NotificationChannel]NotificationSender
	preferences UserAlertPreferences
	repository  AlertRepository
	rateLimiter *inMemoryRateLimiter
	stats       *AlertStats
	mu          sync.RWMutex
}

// NewNotificationService creates a new notification service
func NewNotificationService(config *NotificationConfig) *NotificationService {
	if config == nil {
		config = DefaultNotificationConfig()
	}
	return &NotificationService{
		config:      config,
		senders:     make(map[NotificationChannel]NotificationSender),
		rateLimiter: newInMemoryRateLimiter(config.MaxAlertsPerHour),
		stats: &AlertStats{
			ByType:    make(map[AlertType]int64),
			ByChannel: make(map[NotificationChannel]int64),
		},
	}
}

// RegisterSender registers a notification sender for a channel
func (s *NotificationService) RegisterSender(sender NotificationSender) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.senders[sender.Channel()] = sender
}

// HasSender checks if a sender is registered for a channel
func (s *NotificationService) HasSender(channel NotificationChannel) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.senders[channel]
	return exists
}

// SetPreferencesProvider sets the user preferences provider
func (s *NotificationService) SetPreferencesProvider(provider UserAlertPreferences) {
	s.preferences = provider
}

// SetRepository sets the alert repository
func (s *NotificationService) SetRepository(repo AlertRepository) {
	s.repository = repo
}

// SendAlert sends an alert to a user based on their preferences
func (s *NotificationService) SendAlert(ctx context.Context, alert *Alert) ([]NotificationResult, error) {
	if alert.ID == "" {
		alert.ID = uuid.New().String()
	}
	if alert.CreatedAt.IsZero() {
		alert.CreatedAt = time.Now()
	}

	// Get user preferences
	prefs, err := s.getUserPreferences(ctx, alert.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user preferences: %w", err)
	}

	// Check if alert type is enabled for user
	if !s.isAlertTypeEnabled(prefs, alert.Type) {
		return []NotificationResult{}, nil
	}

	// Check rate limits
	if !s.rateLimiter.allow(alert.UserID, alert.Type, prefs.MaxAlertsPerHour) {
		return []NotificationResult{}, nil
	}

	// Determine which channels to use
	channels := s.getEnabledChannels(prefs)
	if len(channels) == 0 {
		return []NotificationResult{}, nil
	}

	// Send to each channel
	results := make([]NotificationResult, 0, len(channels))
	for _, ch := range channels {
		result := s.sendToChannel(ctx, alert, prefs, ch)
		results = append(results, result)
		s.updateStats(result)
	}

	// Save alert to repository if configured
	if s.repository != nil {
		s.repository.SaveAlert(ctx, alert)
	}

	return results, nil
}

// SendAlertToAll sends an alert to all users with the alert type enabled
func (s *NotificationService) SendAlertToAll(ctx context.Context, alert *Alert) ([]NotificationResult, error) {
	if s.preferences == nil {
		return nil, fmt.Errorf("preferences provider not configured")
	}

	users, err := s.preferences.GetUsersForAlert(ctx, alert.Type)
	if err != nil {
		return nil, fmt.Errorf("failed to get users for alert: %w", err)
	}

	var allResults []NotificationResult
	for _, user := range users {
		alert.UserID = user.UserID
		results, err := s.SendAlert(ctx, alert)
		if err != nil {
			continue
		}
		allResults = append(allResults, results...)
	}

	return allResults, nil
}

// GetStats returns notification statistics
func (s *NotificationService) GetStats() AlertStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy
	stats := AlertStats{
		TotalSent:   s.stats.TotalSent,
		TotalFailed: s.stats.TotalFailed,
		ByType:      make(map[AlertType]int64),
		ByChannel:   make(map[NotificationChannel]int64),
	}
	for k, v := range s.stats.ByType {
		stats.ByType[k] = v
	}
	for k, v := range s.stats.ByChannel {
		stats.ByChannel[k] = v
	}
	if s.stats.LastAlertAt != nil {
		t := *s.stats.LastAlertAt
		stats.LastAlertAt = &t
	}
	return stats
}

// =============================================================================
// HELPER METHODS
// =============================================================================

func (s *NotificationService) getUserPreferences(ctx context.Context, userID int64) (*UserNotificationSettings, error) {
	if s.preferences == nil {
		return DefaultUserNotificationSettings(userID, ""), nil
	}
	return s.preferences.GetUserPreferences(ctx, userID)
}

func (s *NotificationService) isAlertTypeEnabled(prefs *UserNotificationSettings, alertType AlertType) bool {
	switch alertType {
	case AlertTypeWorkerOffline, AlertTypeWorkerOnline:
		return prefs.WorkerOfflineEnabled
	case AlertTypeHashrateDrop:
		return prefs.HashrateDropEnabled
	case AlertTypeBlockFound:
		return prefs.BlockFoundEnabled
	case AlertTypePayoutSent, AlertTypePayoutFailed:
		return prefs.PayoutEnabled
	default:
		return true // Unknown types default to enabled
	}
}

func (s *NotificationService) getEnabledChannels(prefs *UserNotificationSettings) []NotificationChannel {
	var channels []NotificationChannel

	if prefs.EmailEnabled && prefs.Email != "" {
		channels = append(channels, ChannelEmail)
	}
	if prefs.DiscordEnabled && prefs.DiscordWebhook != "" {
		channels = append(channels, ChannelDiscord)
	}
	if prefs.SMSEnabled && prefs.PhoneNumber != "" {
		channels = append(channels, ChannelSMS)
	}

	return channels
}

func (s *NotificationService) sendToChannel(ctx context.Context, alert *Alert, prefs *UserNotificationSettings, channel NotificationChannel) NotificationResult {
	s.mu.RLock()
	sender, exists := s.senders[channel]
	s.mu.RUnlock()

	result := NotificationResult{
		AlertID: alert.ID,
		Channel: channel,
		SentAt:  time.Now(),
	}

	if !exists {
		result.Success = false
		result.Error = "sender not registered"
		return result
	}

	if !sender.IsAvailable() {
		result.Success = false
		result.Error = "sender not available"
		return result
	}

	// Get destination based on channel
	var destination string
	switch channel {
	case ChannelEmail:
		destination = prefs.Email
	case ChannelDiscord:
		destination = prefs.DiscordWebhook
	case ChannelSMS:
		destination = prefs.PhoneNumber
	}
	result.Destination = destination

	// Send with retry
	var lastErr error
	for i := 0; i < s.config.RetryAttempts; i++ {
		err := sender.Send(ctx, alert, destination)
		if err == nil {
			result.Success = true
			return result
		}
		lastErr = err
		time.Sleep(s.config.RetryDelay)
	}

	result.Success = false
	result.Error = lastErr.Error()
	return result
}

func (s *NotificationService) updateStats(result NotificationResult) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if result.Success {
		s.stats.TotalSent++
		s.stats.ByChannel[result.Channel]++
	} else {
		s.stats.TotalFailed++
	}
	now := time.Now()
	s.stats.LastAlertAt = &now
}

// =============================================================================
// RATE LIMITER
// =============================================================================

type inMemoryRateLimiter struct {
	maxPerHour int
	counts     map[string][]time.Time
	mu         sync.Mutex
}

func newInMemoryRateLimiter(maxPerHour int) *inMemoryRateLimiter {
	return &inMemoryRateLimiter{
		maxPerHour: maxPerHour,
		counts:     make(map[string][]time.Time),
	}
}

func (r *inMemoryRateLimiter) allow(userID int64, alertType AlertType, userMax int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := fmt.Sprintf("%d:%s", userID, alertType)
	now := time.Now()
	hourAgo := now.Add(-time.Hour)

	// Clean old entries
	times := r.counts[key]
	var recent []time.Time
	for _, t := range times {
		if t.After(hourAgo) {
			recent = append(recent, t)
		}
	}

	// Check limit
	limit := r.maxPerHour
	if userMax > 0 && userMax < limit {
		limit = userMax
	}

	if len(recent) >= limit {
		return false
	}

	// Record this attempt
	r.counts[key] = append(recent, now)
	return true
}

// =============================================================================
// ALERT FACTORY FUNCTIONS
// =============================================================================

// NewWorkerOfflineAlert creates a worker offline alert
func NewWorkerOfflineAlert(userID, workerID int64, workerName string) *Alert {
	return &Alert{
		ID:         uuid.New().String(),
		Type:       AlertTypeWorkerOffline,
		Severity:   SeverityWarning,
		Title:      fmt.Sprintf("Worker Offline: %s", workerName),
		Message:    fmt.Sprintf("Your worker '%s' has stopped submitting shares and appears to be offline.", workerName),
		UserID:     userID,
		WorkerID:   workerID,
		WorkerName: workerName,
		CreatedAt:  time.Now(),
	}
}

// NewWorkerOnlineAlert creates a worker back online alert
func NewWorkerOnlineAlert(userID, workerID int64, workerName string) *Alert {
	return &Alert{
		ID:         uuid.New().String(),
		Type:       AlertTypeWorkerOnline,
		Severity:   SeverityInfo,
		Title:      fmt.Sprintf("Worker Online: %s", workerName),
		Message:    fmt.Sprintf("Your worker '%s' is back online and submitting shares.", workerName),
		UserID:     userID,
		WorkerID:   workerID,
		WorkerName: workerName,
		CreatedAt:  time.Now(),
	}
}

// NewBlockFoundAlert creates a block found alert
func NewBlockFoundAlert(blockHeight int64, reward int64, coin string) *Alert {
	rewardFloat := float64(reward) / 100000000
	return &Alert{
		ID:       uuid.New().String(),
		Type:     AlertTypeBlockFound,
		Severity: SeverityInfo,
		Title:    fmt.Sprintf("🎉 Block Found! #%d", blockHeight),
		Message:  fmt.Sprintf("Pool found block #%d! Reward: %.8f %s", blockHeight, rewardFloat, coin),
		Metadata: map[string]string{
			"block_height": fmt.Sprintf("%d", blockHeight),
			"reward":       fmt.Sprintf("%d", reward),
			"coin":         coin,
		},
		CreatedAt: time.Now(),
	}
}

// NewPayoutSentAlert creates a payout sent alert
func NewPayoutSentAlert(userID int64, amount int64, address, txHash string) *Alert {
	amountFloat := float64(amount) / 100000000
	return &Alert{
		ID:       uuid.New().String(),
		Type:     AlertTypePayoutSent,
		Severity: SeverityInfo,
		Title:    "Payout Sent",
		Message:  fmt.Sprintf("Payout of %.8f LTC sent to %s", amountFloat, address),
		UserID:   userID,
		Metadata: map[string]string{
			"amount":  fmt.Sprintf("%d", amount),
			"address": address,
			"tx_hash": txHash,
		},
		CreatedAt: time.Now(),
	}
}

// NewPayoutFailedAlert creates a payout failed alert
func NewPayoutFailedAlert(userID int64, amount int64, reason string) *Alert {
	amountFloat := float64(amount) / 100000000
	return &Alert{
		ID:       uuid.New().String(),
		Type:     AlertTypePayoutFailed,
		Severity: SeverityWarning,
		Title:    "Payout Failed",
		Message:  fmt.Sprintf("Payout of %.8f LTC failed: %s", amountFloat, reason),
		UserID:   userID,
		Metadata: map[string]string{
			"amount": fmt.Sprintf("%d", amount),
			"reason": reason,
		},
		CreatedAt: time.Now(),
	}
}

// NewHashrateDropAlert creates a hashrate drop alert
func NewHashrateDropAlert(userID, workerID int64, workerName string, dropPercent int) *Alert {
	return &Alert{
		ID:         uuid.New().String(),
		Type:       AlertTypeHashrateDrop,
		Severity:   SeverityWarning,
		Title:      fmt.Sprintf("Hashrate Drop: %s", workerName),
		Message:    fmt.Sprintf("Worker '%s' hashrate dropped by %d%%", workerName, dropPercent),
		UserID:     userID,
		WorkerID:   workerID,
		WorkerName: workerName,
		Metadata: map[string]string{
			"drop_percent": fmt.Sprintf("%d", dropPercent),
		},
		CreatedAt: time.Now(),
	}
}

// NewLowBalanceAlert creates a low wallet balance alert
func NewLowBalanceAlert(balance int64, threshold int64) *Alert {
	balanceFloat := float64(balance) / 100000000
	thresholdFloat := float64(threshold) / 100000000
	return &Alert{
		ID:       uuid.New().String(),
		Type:     AlertTypeLowBalance,
		Severity: SeverityCritical,
		Title:    "Low Wallet Balance",
		Message:  fmt.Sprintf("Pool wallet balance (%.8f LTC) is below threshold (%.8f LTC)", balanceFloat, thresholdFloat),
		Metadata: map[string]string{
			"balance":   fmt.Sprintf("%d", balance),
			"threshold": fmt.Sprintf("%d", threshold),
		},
		CreatedAt: time.Now(),
	}
}
