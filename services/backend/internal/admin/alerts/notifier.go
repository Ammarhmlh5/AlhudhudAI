package alerts

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// NotificationChannel represents a notification delivery channel
type NotificationChannel string

const (
	ChannelEmail NotificationChannel = "email"
	ChannelSMS   NotificationChannel = "sms"
	ChannelInApp NotificationChannel = "in_app"
	ChannelAll   NotificationChannel = "all"
)

// NotificationService interface for sending notifications
// This allows integration with the existing notification service
type NotificationService interface {
	SendEmail(ctx context.Context, tenantID uuid.UUID, subject string, message string) error
	SendSMS(ctx context.Context, tenantID uuid.UUID, message string) error
	SendInApp(ctx context.Context, tenantID uuid.UUID, title string, message string, data map[string]interface{}) error
}

// Notifier handles multi-channel alert notifications
// Implements Requirement 6.5
type Notifier struct {
	notificationService NotificationService
	logger              *zap.Logger
	enabledChannels     map[NotificationChannel]bool
}

// NotifierConfig holds configuration for the notifier
type NotifierConfig struct {
	EnableEmail bool
	EnableSMS   bool
	EnableInApp bool
}

// NewNotifier creates a new alert notifier
func NewNotifier(notificationService NotificationService, logger *zap.Logger, config *NotifierConfig) *Notifier {
	enabledChannels := make(map[NotificationChannel]bool)
	
	if config != nil {
		enabledChannels[ChannelEmail] = config.EnableEmail
		enabledChannels[ChannelSMS] = config.EnableSMS
		enabledChannels[ChannelInApp] = config.EnableInApp
	} else {
		// Default: enable all channels
		enabledChannels[ChannelEmail] = true
		enabledChannels[ChannelSMS] = true
		enabledChannels[ChannelInApp] = true
	}

	return &Notifier{
		notificationService: notificationService,
		logger:              logger,
		enabledChannels:     enabledChannels,
	}
}

// SendAlert sends an alert through multiple notification channels
// Implements Requirement 6.5
func (n *Notifier) SendAlert(ctx context.Context, alert *Alert, channels []NotificationChannel) error {
	if alert == nil {
		return fmt.Errorf("alert is nil")
	}

	// If no channels specified, use all enabled channels
	if len(channels) == 0 {
		channels = []NotificationChannel{ChannelAll}
	}

	// Track errors from each channel
	errors := make(map[NotificationChannel]error)
	successCount := 0

	for _, channel := range channels {
		// Handle "all" channel
		if channel == ChannelAll {
			if err := n.sendToAllChannels(ctx, alert); err != nil {
				errors[ChannelAll] = err
			} else {
				successCount++
			}
			continue
		}

		// Check if channel is enabled
		if !n.enabledChannels[channel] {
			n.logger.Debug("Channel not enabled, skipping",
				zap.String("channel", string(channel)),
				zap.Int64("alert_id", alert.ID),
			)
			continue
		}

		// Send to specific channel
		var err error
		switch channel {
		case ChannelEmail:
			err = n.sendEmail(ctx, alert)
		case ChannelSMS:
			err = n.sendSMS(ctx, alert)
		case ChannelInApp:
			err = n.sendInApp(ctx, alert)
		default:
			err = fmt.Errorf("unknown channel: %s", channel)
		}

		if err != nil {
			errors[channel] = err
			n.logger.Error("Failed to send alert via channel",
				zap.String("channel", string(channel)),
				zap.Int64("alert_id", alert.ID),
				zap.Error(err),
			)
		} else {
			successCount++
			n.logger.Info("Alert sent via channel",
				zap.String("channel", string(channel)),
				zap.Int64("alert_id", alert.ID),
			)
		}
	}

	// If all channels failed, return error
	if successCount == 0 && len(errors) > 0 {
		return fmt.Errorf("failed to send alert via any channel: %v", errors)
	}

	// If some channels failed, log warning but don't return error
	if len(errors) > 0 {
		n.logger.Warn("Some channels failed to send alert",
			zap.Int64("alert_id", alert.ID),
			zap.Int("success_count", successCount),
			zap.Int("error_count", len(errors)),
			zap.Any("errors", errors),
		)
	}

	return nil
}

// sendToAllChannels sends the alert to all enabled channels
func (n *Notifier) sendToAllChannels(ctx context.Context, alert *Alert) error {
	channels := []NotificationChannel{}
	
	if n.enabledChannels[ChannelEmail] {
		channels = append(channels, ChannelEmail)
	}
	if n.enabledChannels[ChannelSMS] {
		channels = append(channels, ChannelSMS)
	}
	if n.enabledChannels[ChannelInApp] {
		channels = append(channels, ChannelInApp)
	}

	return n.SendAlert(ctx, alert, channels)
}

// sendEmail sends the alert via email
func (n *Notifier) sendEmail(ctx context.Context, alert *Alert) error {
	if alert.TenantID == nil {
		// System-wide alert, send to admins
		// TODO: Implement admin notification logic
		n.logger.Info("System-wide alert, would send to admins",
			zap.Int64("alert_id", alert.ID),
		)
		return nil
	}

	subject := n.formatEmailSubject(alert)
	message := n.formatEmailMessage(alert)

	return n.notificationService.SendEmail(ctx, *alert.TenantID, subject, message)
}

// sendSMS sends the alert via SMS
func (n *Notifier) sendSMS(ctx context.Context, alert *Alert) error {
	if alert.TenantID == nil {
		// System-wide alert, send to admins
		n.logger.Info("System-wide alert, would send SMS to admins",
			zap.Int64("alert_id", alert.ID),
		)
		return nil
	}

	message := n.formatSMSMessage(alert)

	return n.notificationService.SendSMS(ctx, *alert.TenantID, message)
}

// sendInApp sends the alert as an in-app notification
func (n *Notifier) sendInApp(ctx context.Context, alert *Alert) error {
	if alert.TenantID == nil {
		// System-wide alert, send to admins
		n.logger.Info("System-wide alert, would send in-app to admins",
			zap.Int64("alert_id", alert.ID),
		)
		return nil
	}

	data := map[string]interface{}{
		"alert_id":   alert.ID,
		"alert_type": alert.AlertType,
		"severity":   alert.Severity,
		"context":    alert.Context,
	}

	return n.notificationService.SendInApp(ctx, *alert.TenantID, alert.Title, alert.Message, data)
}

// formatEmailSubject formats the alert subject for email
func (n *Notifier) formatEmailSubject(alert *Alert) string {
	severityPrefix := ""
	switch alert.Severity {
	case AlertSeverityCritical:
		severityPrefix = "[حرج] "
	case AlertSeverityWarning:
		severityPrefix = "[تحذير] "
	case AlertSeverityInfo:
		severityPrefix = "[معلومة] "
	}

	return severityPrefix + alert.Title
}

// formatEmailMessage formats the alert message for email
func (n *Notifier) formatEmailMessage(alert *Alert) string {
	message := fmt.Sprintf(`
تنبيه من نظام الإدارة

النوع: %s
الخطورة: %s
العنوان: %s

الرسالة:
%s

الوقت: %s
معرف التنبيه: %d
`,
		n.getAlertTypeArabic(alert.AlertType),
		n.getSeverityArabic(alert.Severity),
		alert.Title,
		alert.Message,
		alert.CreatedAt.Format("2006-01-02 15:04:05"),
		alert.ID,
	)

	// Add context if available
	if alert.Context != nil && len(alert.Context) > 0 {
		message += "\nمعلومات إضافية:\n"
		for key, value := range alert.Context {
			message += fmt.Sprintf("- %s: %v\n", key, value)
		}
	}

	return message
}

// formatSMSMessage formats the alert message for SMS (shorter version)
func (n *Notifier) formatSMSMessage(alert *Alert) string {
	severityEmoji := ""
	switch alert.Severity {
	case AlertSeverityCritical:
		severityEmoji = "🔴"
	case AlertSeverityWarning:
		severityEmoji = "⚠️"
	case AlertSeverityInfo:
		severityEmoji = "ℹ️"
	}

	return fmt.Sprintf("%s %s: %s", severityEmoji, alert.Title, alert.Message)
}

// getAlertTypeArabic returns the Arabic name for alert type
func (n *Notifier) getAlertTypeArabic(alertType AlertType) string {
	switch alertType {
	case AlertTypeRatingChange:
		return "تغيير التقييم"
	case AlertTypeServerFailure:
		return "فشل السيرفر"
	case AlertTypeThresholdViolation:
		return "تجاوز العتبة"
	case AlertTypeDeviceFailure:
		return "فشل الجهاز"
	default:
		return string(alertType)
	}
}

// getSeverityArabic returns the Arabic name for severity
func (n *Notifier) getSeverityArabic(severity AlertSeverity) string {
	switch severity {
	case AlertSeverityCritical:
		return "حرج"
	case AlertSeverityWarning:
		return "تحذير"
	case AlertSeverityInfo:
		return "معلومة"
	default:
		return string(severity)
	}
}

// SetChannelEnabled enables or disables a notification channel
func (n *Notifier) SetChannelEnabled(channel NotificationChannel, enabled bool) {
	n.enabledChannels[channel] = enabled
	n.logger.Info("Channel status changed",
		zap.String("channel", string(channel)),
		zap.Bool("enabled", enabled),
	)
}

// IsChannelEnabled checks if a notification channel is enabled
func (n *Notifier) IsChannelEnabled(channel NotificationChannel) bool {
	return n.enabledChannels[channel]
}
