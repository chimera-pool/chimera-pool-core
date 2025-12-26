package payouts

import (
	"context"

	"github.com/chimera-pool/chimera-pool-core/internal/notifications"
)

// =============================================================================
// NOTIFICATION ADAPTER - BRIDGES PAYOUT PROCESSOR WITH NOTIFICATION SERVICE
// =============================================================================

// NotificationAdapter implements PayoutNotifier using the notification service
type NotificationAdapter struct {
	service *notifications.NotificationService
}

// NewNotificationAdapter creates a new notification adapter
func NewNotificationAdapter(service *notifications.NotificationService) *NotificationAdapter {
	return &NotificationAdapter{service: service}
}

// NotifyPayoutSent sends a notification when a payout is successfully sent
func (a *NotificationAdapter) NotifyPayoutSent(ctx context.Context, userID int64, amount int64, address, txHash string) error {
	if a.service == nil {
		return nil
	}

	alert := notifications.NewPayoutSentAlert(userID, amount, address, txHash)
	_, err := a.service.SendAlert(ctx, alert)
	return err
}

// NotifyPayoutFailed sends a notification when a payout fails
func (a *NotificationAdapter) NotifyPayoutFailed(ctx context.Context, userID int64, amount int64, reason string) error {
	if a.service == nil {
		return nil
	}

	alert := notifications.NewPayoutFailedAlert(userID, amount, reason)
	_, err := a.service.SendAlert(ctx, alert)
	return err
}

// Ensure NotificationAdapter implements PayoutNotifier
var _ PayoutNotifier = (*NotificationAdapter)(nil)
