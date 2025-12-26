package api

import (
	"net/http"
	"strconv"

	"github.com/chimera-pool/chimera-pool-core/internal/notifications"
	"github.com/gin-gonic/gin"
)

// =============================================================================
// NOTIFICATION API HANDLERS (Gin)
// =============================================================================

// NotificationHandlers handles notification-related API requests
type NotificationHandlers struct {
	notificationService *notifications.NotificationService
	preferencesRepo     notifications.UserAlertPreferences
}

// NewNotificationHandlers creates new notification handlers
func NewNotificationHandlers(
	service *notifications.NotificationService,
	prefsRepo notifications.UserAlertPreferences,
) *NotificationHandlers {
	return &NotificationHandlers{
		notificationService: service,
		preferencesRepo:     prefsRepo,
	}
}

// GetUserNotificationSettings returns the user's notification preferences
func (h *NotificationHandlers) GetUserNotificationSettings(c *gin.Context) {
	userID := getUserIDFromGinContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	settings, err := h.preferencesRepo.GetUserPreferences(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get settings"})
		return
	}

	c.JSON(http.StatusOK, settings)
}

// UpdateUserNotificationSettings updates the user's notification preferences
func (h *NotificationHandlers) UpdateUserNotificationSettings(c *gin.Context) {
	userID := getUserIDFromGinContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var settings notifications.UserNotificationSettings
	if err := c.ShouldBindJSON(&settings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	settings.UserID = userID

	if err := h.preferencesRepo.UpdatePreferences(c.Request.Context(), userID, &settings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update settings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}

// TestNotification sends a test notification to the user
func (h *NotificationHandlers) TestNotification(c *gin.Context) {
	userID := getUserIDFromGinContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	channel := c.Query("channel")
	if channel == "" {
		channel = "email"
	}

	alert := &notifications.Alert{
		Type:     notifications.AlertTypeWorkerOffline,
		Severity: notifications.SeverityInfo,
		Title:    "Test Notification",
		Message:  "This is a test notification from Chimera Pool. If you received this, your notifications are working correctly!",
		UserID:   userID,
	}

	results, err := h.notificationService.SendAlert(c.Request.Context(), alert)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send test notification"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "sent",
		"results": results,
	})
}

// GetNotificationStats returns notification statistics
func (h *NotificationHandlers) GetNotificationStats(c *gin.Context) {
	stats := h.notificationService.GetStats()
	c.JSON(http.StatusOK, stats)
}

// =============================================================================
// ALERTMANAGER WEBHOOK HANDLERS
// =============================================================================

// AlertManagerWebhook handles incoming alerts from Prometheus AlertManager
type AlertManagerWebhook struct {
	notificationService *notifications.NotificationService
	workerMonitor       *notifications.WorkerMonitor
}

// NewAlertManagerWebhook creates a new AlertManager webhook handler
func NewAlertManagerWebhook(
	service *notifications.NotificationService,
	monitor *notifications.WorkerMonitor,
) *AlertManagerWebhook {
	return &AlertManagerWebhook{
		notificationService: service,
		workerMonitor:       monitor,
	}
}

// AlertManagerPayload represents the AlertManager webhook payload
type AlertManagerPayload struct {
	Version           string              `json:"version"`
	GroupKey          string              `json:"groupKey"`
	TruncatedAlerts   int                 `json:"truncatedAlerts"`
	Status            string              `json:"status"` // "firing" or "resolved"
	Receiver          string              `json:"receiver"`
	GroupLabels       map[string]string   `json:"groupLabels"`
	CommonLabels      map[string]string   `json:"commonLabels"`
	CommonAnnotations map[string]string   `json:"commonAnnotations"`
	ExternalURL       string              `json:"externalURL"`
	Alerts            []AlertManagerAlert `json:"alerts"`
}

// AlertManagerAlert represents a single alert in the payload
type AlertManagerAlert struct {
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     string            `json:"startsAt"`
	EndsAt       string            `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
	Fingerprint  string            `json:"fingerprint"`
}

// HandleAlerts handles general alerts from AlertManager
func (h *AlertManagerWebhook) HandleAlerts(c *gin.Context) {
	var payload AlertManagerPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	for _, alert := range payload.Alerts {
		h.processAlert(c.Request.Context(), alert, payload.Status)
	}

	c.JSON(http.StatusOK, gin.H{"status": "received"})
}

// HandleCriticalAlerts handles critical alerts
func (h *AlertManagerWebhook) HandleCriticalAlerts(c *gin.Context) {
	h.HandleAlerts(c)
}

// HandleWorkerAlerts handles worker-specific alerts
func (h *AlertManagerWebhook) HandleWorkerAlerts(c *gin.Context) {
	h.HandleAlerts(c)
}

// HandleBlockAlerts handles block found alerts
func (h *AlertManagerWebhook) HandleBlockAlerts(c *gin.Context) {
	var payload AlertManagerPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	for _, alert := range payload.Alerts {
		if alert.Labels["alertname"] == "BlockFound" {
			blockHeight, _ := strconv.ParseInt(alert.Labels["block_height"], 10, 64)
			reward, _ := strconv.ParseInt(alert.Labels["reward"], 10, 64)
			coin := alert.Labels["coin"]
			if coin == "" {
				coin = "LTC"
			}

			blockAlert := notifications.NewBlockFoundAlert(blockHeight, reward, coin)
			h.notificationService.SendAlertToAll(c.Request.Context(), blockAlert)
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "received"})
}

// HandlePayoutAlerts handles payout-related alerts
func (h *AlertManagerWebhook) HandlePayoutAlerts(c *gin.Context) {
	h.HandleAlerts(c)
}

func (h *AlertManagerWebhook) processAlert(ctx interface{}, alert AlertManagerAlert, status string) {
	// Map AlertManager alert to our notification system
	alertName := alert.Labels["alertname"]
	severity := mapSeverity(alert.Labels["severity"])

	switch alertName {
	case "WorkerOffline":
		// Worker offline alerts are handled by WorkerMonitor
		return
	case "PoolHashrateDrop":
		// Broadcast to all admins
	case "LowWalletBalance":
		// Alert pool operators
	}

	// For now, log the alert
	_ = alertName
	_ = severity
}

func mapSeverity(s string) notifications.AlertSeverity {
	switch s {
	case "critical":
		return notifications.SeverityCritical
	case "warning":
		return notifications.SeverityWarning
	default:
		return notifications.SeverityInfo
	}
}

// =============================================================================
// DISCORD WEBHOOK HANDLER
// =============================================================================

// DiscordInteractionHandler handles Discord interaction webhooks (for slash commands)
type DiscordInteractionHandler struct {
	// Future: Handle Discord slash commands
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// RegisterNotificationRoutes registers all notification routes with Gin
func RegisterNotificationRoutes(r *gin.RouterGroup, handlers *NotificationHandlers, webhook *AlertManagerWebhook) {
	// User notification settings (requires auth)
	notifications := r.Group("/notifications")
	{
		notifications.GET("/settings", handlers.GetUserNotificationSettings)
		notifications.PUT("/settings", handlers.UpdateUserNotificationSettings)
		notifications.POST("/test", handlers.TestNotification)
		notifications.GET("/stats", handlers.GetNotificationStats)
	}

	// AlertManager webhooks (internal, no auth required but should be IP restricted)
	webhooks := r.Group("/webhooks/alerts")
	{
		webhooks.POST("/", webhook.HandleAlerts)
		webhooks.POST("/critical", webhook.HandleCriticalAlerts)
		webhooks.POST("/workers", webhook.HandleWorkerAlerts)
		webhooks.POST("/blocks", webhook.HandleBlockAlerts)
		webhooks.POST("/payouts", webhook.HandlePayoutAlerts)
	}
}

// getUserIDFromGinContext extracts user ID from Gin context
func getUserIDFromGinContext(c *gin.Context) int64 {
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(int64); ok {
			return id
		}
	}
	return 0
}
