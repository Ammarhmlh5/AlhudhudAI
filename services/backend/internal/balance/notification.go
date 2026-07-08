package balance

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// NotificationLevel represents the severity level of a notification
type NotificationLevel string

const (
	NotificationLevelWarning  NotificationLevel = "WARNING"
	NotificationLevelCritical NotificationLevel = "CRITICAL"
)

// NotificationType represents the type of notification
type NotificationType string

const (
	NotificationTypeLowBalance         NotificationType = "LOW_BALANCE"
	NotificationTypeCriticalBalance    NotificationType = "CRITICAL_BALANCE"
	NotificationTypeInsufficientBalance NotificationType = "INSUFFICIENT_BALANCE"
)

// Notification represents a balance notification
type Notification struct {
	TenantID  uuid.UUID         `json:"tenant_id"`
	Type      NotificationType  `json:"type"`
	Level     NotificationLevel `json:"level"`
	Balance   decimal.Decimal   `json:"balance"`
	Threshold decimal.Decimal   `json:"threshold"`
	Message   string            `json:"message"`
	MessageID *uuid.UUID        `json:"message_id,omitempty"`
}

// NotificationService interface defines notification operations
type NotificationService interface {
	// CheckAndNotifyLowBalance checks balance and sends notification if below thresholds
	// Implements Requirements 11.4, 11.5
	CheckAndNotifyLowBalance(ctx context.Context, tenantID uuid.UUID, balance decimal.Decimal) error

	// LogInsufficientBalanceAttempt logs a failed deduction attempt
	// Implements Requirement 11.6
	LogInsufficientBalanceAttempt(ctx context.Context, tenantID uuid.UUID, messageID *uuid.UUID, required, available decimal.Decimal) error

	// SendNotification sends a notification (can be extended to email, SMS, webhook, etc.)
	SendNotification(ctx context.Context, notification Notification) error
}

// notificationService implements the NotificationService interface
type notificationService struct {
	logger *zap.Logger
}

// NewNotificationService creates a new notification service
func NewNotificationService(logger *zap.Logger) NotificationService {
	return &notificationService{
		logger: logger,
	}
}

// CheckAndNotifyLowBalance checks balance and sends notification if below thresholds
// Implements Requirements 11.4, 11.5
func (ns *notificationService) CheckAndNotifyLowBalance(ctx context.Context, tenantID uuid.UUID, balance decimal.Decimal) error {
	// Threshold values
	criticalThreshold := decimal.NewFromInt(5)
	warningThreshold := decimal.NewFromInt(10)

	// Check critical threshold (< 5 units) - Requirement 11.5
	if balance.LessThan(criticalThreshold) {
		notification := Notification{
			TenantID:  tenantID,
			Type:      NotificationTypeCriticalBalance,
			Level:     NotificationLevelCritical,
			Balance:   balance,
			Threshold: criticalThreshold,
			Message:   fmt.Sprintf("CRITICAL: Balance is critically low (%.4f units). Please recharge immediately.", balance.InexactFloat64()),
		}

		if err := ns.SendNotification(ctx, notification); err != nil {
			ns.logger.Error("Failed to send critical balance notification",
				zap.String("tenant_id", tenantID.String()),
				zap.String("balance", balance.String()),
				zap.Error(err),
			)
			return fmt.Errorf("failed to send critical balance notification: %w", err)
		}

		return nil
	}

	// Check warning threshold (< 10 units) - Requirement 11.4
	if balance.LessThan(warningThreshold) {
		notification := Notification{
			TenantID:  tenantID,
			Type:      NotificationTypeLowBalance,
			Level:     NotificationLevelWarning,
			Balance:   balance,
			Threshold: warningThreshold,
			Message:   fmt.Sprintf("WARNING: Balance is low (%.4f units). Consider recharging soon.", balance.InexactFloat64()),
		}

		if err := ns.SendNotification(ctx, notification); err != nil {
			ns.logger.Error("Failed to send low balance notification",
				zap.String("tenant_id", tenantID.String()),
				zap.String("balance", balance.String()),
				zap.Error(err),
			)
			return fmt.Errorf("failed to send low balance notification: %w", err)
		}
	}

	return nil
}

// LogInsufficientBalanceAttempt logs a failed deduction attempt
// Implements Requirement 11.6
func (ns *notificationService) LogInsufficientBalanceAttempt(ctx context.Context, tenantID uuid.UUID, messageID *uuid.UUID, required, available decimal.Decimal) error {
	notification := Notification{
		TenantID:  tenantID,
		Type:      NotificationTypeInsufficientBalance,
		Level:     NotificationLevelWarning,
		Balance:   available,
		Threshold: required,
		Message:   fmt.Sprintf("Failed to deduct balance: required %.4f units, available %.4f units", required.InexactFloat64(), available.InexactFloat64()),
		MessageID: messageID,
	}

	if err := ns.SendNotification(ctx, notification); err != nil {
		ns.logger.Error("Failed to log insufficient balance attempt",
			zap.String("tenant_id", tenantID.String()),
			zap.Any("message_id", messageID),
			zap.String("required", required.String()),
			zap.String("available", available.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to log insufficient balance attempt: %w", err)
	}

	return nil
}

// SendNotification sends a notification
// Currently logs to structured logger, can be extended to send emails, SMS, webhooks, etc.
func (ns *notificationService) SendNotification(ctx context.Context, notification Notification) error {
	// Log the notification with structured fields
	logFields := []zap.Field{
		zap.String("tenant_id", notification.TenantID.String()),
		zap.String("type", string(notification.Type)),
		zap.String("level", string(notification.Level)),
		zap.String("balance", notification.Balance.String()),
		zap.String("threshold", notification.Threshold.String()),
		zap.String("message", notification.Message),
	}

	if notification.MessageID != nil {
		logFields = append(logFields, zap.String("message_id", notification.MessageID.String()))
	}

	// Log based on level
	switch notification.Level {
	case NotificationLevelCritical:
		ns.logger.Error("CRITICAL BALANCE NOTIFICATION", logFields...)
	case NotificationLevelWarning:
		ns.logger.Warn("BALANCE NOTIFICATION", logFields...)
	default:
		ns.logger.Info("BALANCE NOTIFICATION", logFields...)
	}

	// TODO: Future enhancements:
	// - Send email notification
	// - Send SMS notification
	// - Send webhook notification
	// - Store notification in database for history
	// - Implement notification preferences per tenant
	// - Implement rate limiting to avoid notification spam

	return nil
}
