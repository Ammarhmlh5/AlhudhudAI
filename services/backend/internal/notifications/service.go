package notifications

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/al-hudhud/backend/internal/models"
	"github.com/al-hudhud/backend/internal/ws"
	"github.com/al-hudhud/backend/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// DBQueryer interface for database operations
type DBQueryer interface {
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

// HubBroadcaster interface for WebSocket hub operations
type HubBroadcaster interface {
	BroadcastToTenant(ctx context.Context, tenantID uuid.UUID, msg ws.WebSocketMessage) error
}

// Service handles notification operations with complete tenant isolation
type Service struct {
	db        DBQueryer
	wsHub     HubBroadcaster
}

// NewService creates a new notification service
func NewService(db DBQueryer, wsHub HubBroadcaster) *Service {
	return &Service{
		db:    db,
		wsHub: wsHub,
	}
}

// Notification represents a notification record
type Notification struct {
	ID        uuid.UUID
	TenantID  uuid.UUID
	UserID    uuid.UUID
	Type      string
	Title     string
	Body      string
	Data      json.RawMessage
	ReadAt    *time.Time
	CreatedAt time.Time
}

// CreateNotificationInput represents input for creating a notification
type CreateNotificationInput struct {
	TenantID uuid.UUID
	UserID   uuid.UUID
	Type     string // 'message', 'status', 'system', 'delivery', 'alert'
	Title    string
	Body     string
	Data     map[string]interface{}
}

// Create creates a new notification with tenant isolation
func (s *Service) Create(ctx context.Context, input CreateNotificationInput) (*Notification, error) {
	// Verify tenant isolation - only create for valid tenant
	var tenantExists bool
	err := s.db.QueryRowContext(ctx, 
		"SELECT EXISTS(SELECT 1 FROM tenants WHERE id = $1)", 
		input.TenantID,
	).Scan(&tenantExists)
	
	if err != nil {
		logger.GetLogger().Error("Failed to verify tenant", zap.Error(err))
		return nil, err
	}
	
	if !tenantExists {
		logger.GetLogger().Warn("Attempted to create notification for non-existent tenant", 
			zap.String("tenant_id", input.TenantID.String()))
		return nil, fmt.Errorf("tenant does not exist")
	}

	// Serialize data to JSON
	dataJSON, _ := json.Marshal(input.Data)

	// Insert notification
	var notification Notification
	err = s.db.QueryRowContext(ctx, `
		INSERT INTO notifications (id, tenant_id, user_id, type, title, body, data)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6)
		RETURNING id, tenant_id, user_id, type, title, body, data, created_at
	`, input.TenantID, input.UserID, input.Type, input.Title, input.Body, dataJSON,
	).Scan(
		&notification.ID,
		&notification.TenantID,
		&notification.UserID,
		&notification.Type,
		&notification.Title,
		&notification.Body,
		&notification.Data,
		&notification.CreatedAt,
	)

	if err != nil {
		logger.GetLogger().Error("Failed to create notification", zap.Error(err))
		return nil, err
	}

	logger.GetLogger().Info("Notification created",
		zap.String("notification_id", notification.ID.String()),
		zap.String("tenant_id", notification.TenantID.String()),
		zap.String("user_id", notification.UserID.String()),
		zap.String("type", notification.Type),
	)

	// Send real-time notification via WebSocket
	s.sendRealTimeNotification(&notification)

	return &notification, nil
}

// GetByTenantAndUser retrieves notifications for a specific tenant and user
func (s *Service) GetByTenantAndUser(ctx context.Context, tenantID, userID uuid.UUID, limit int) ([]Notification, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, tenant_id, user_id, type, title, body, data, read_at, created_at
		FROM notifications
		WHERE tenant_id = $1 AND user_id = $2
		ORDER BY created_at DESC
		LIMIT $3
	`, tenantID, userID, limit)

	if err != nil {
		logger.GetLogger().Error("Failed to get notifications", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var notifications []Notification
	for rows.Next() {
		var n Notification
		err := rows.Scan(
			&n.ID, &n.TenantID, &n.UserID, &n.Type, &n.Title, &n.Body,
			&n.Data, &n.ReadAt, &n.CreatedAt,
		)
		if err != nil {
			logger.GetLogger().Error("Failed to scan notification", zap.Error(err))
			continue
		}
		notifications = append(notifications, n)
	}

	return notifications, nil
}

// GetUnreadCount returns unread notification count for a tenant and user
func (s *Service) GetUnreadCount(ctx context.Context, tenantID, userID uuid.UUID) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM notifications
		WHERE tenant_id = $1 AND user_id = $2 AND read_at IS NULL
	`, tenantID, userID).Scan(&count)

	if err != nil {
		logger.GetLogger().Error("Failed to get unread count", zap.Error(err))
		return 0, err
	}

	return count, nil
}

// MarkAsRead marks a notification as read with tenant isolation
func (s *Service) MarkAsRead(ctx context.Context, notificationID, tenantID uuid.UUID) (bool, error) {
	result, err := s.db.ExecContext(ctx, `
		UPDATE notifications
		SET read_at = NOW()
		WHERE id = $1 AND tenant_id = $2 AND read_at IS NULL
	`, notificationID, tenantID)

	if err != nil {
		logger.GetLogger().Error("Failed to mark notification as read", zap.Error(err))
		return false, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}

	return rowsAffected > 0, nil
}

// MarkAllAsRead marks all notifications as read for a tenant and user
func (s *Service) MarkAllAsRead(ctx context.Context, tenantID, userID uuid.UUID) (int, error) {
	result, err := s.db.ExecContext(ctx, `
		UPDATE notifications
		SET read_at = NOW()
		WHERE tenant_id = $1 AND user_id = $2 AND read_at IS NULL
	`, tenantID, userID)

	if err != nil {
		logger.GetLogger().Error("Failed to mark all notifications as read", zap.Error(err))
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return int(rowsAffected), nil
}

// DeleteOldNotifications deletes notifications older than specified days for a tenant
func (s *Service) DeleteOldNotifications(ctx context.Context, tenantID uuid.UUID, days int) (int, error) {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM notifications
		WHERE tenant_id = $1 AND created_at < NOW() - INTERVAL '%d days'
	`, tenantID, days)

	if err != nil {
		logger.GetLogger().Error("Failed to delete old notifications", zap.Error(err))
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return int(rowsAffected), nil
}

// GetStats returns notification statistics for a tenant
func (s *Service) GetStats(ctx context.Context, tenantID uuid.UUID) (*models.NotificationStats, error) {
	var stats models.NotificationStats
	err := s.db.QueryRowContext(ctx, `
		SELECT 
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE read_at IS NULL) as unread,
			COUNT(*) FILTER (WHERE type = 'message') as messages,
			COUNT(*) FILTER (WHERE type = 'delivery') as deliveries,
			COUNT(*) FILTER (WHERE type = 'status') as statuses,
			COUNT(*) FILTER (WHERE type = 'system') as systems,
			COUNT(*) FILTER (WHERE type = 'alert') as alerts
		FROM notifications
		WHERE tenant_id = $1
	`, tenantID).Scan(
		&stats.Total, &stats.Unread, &stats.Messages, &stats.Deliveries,
		&stats.Statuses, &stats.Systems, &stats.Alerts,
	)

	if err != nil {
		logger.GetLogger().Error("Failed to get notification stats", zap.Error(err))
		return nil, err
	}

	return &stats, nil
}

// sendRealTimeNotification sends notification to mobile app via WebSocket
func (s *Service) sendRealTimeNotification(notification *Notification) {
	if s.wsHub == nil {
		logger.GetLogger().Warn("WebSocket hub not available for real-time notification")
		return
	}

	// Create WebSocket message payload
	payload := map[string]interface{}{
		"type": "notification",
		"data": map[string]interface{}{
			"id":         notification.ID.String(),
			"tenant_id":  notification.TenantID.String(),
			"type":       notification.Type,
			"title":      notification.Title,
			"body":       notification.Body,
			"data":       notification.Data,
			"created_at": notification.CreatedAt.Format(time.RFC3339),
		},
	}

	msg := ws.WebSocketMessage{
		Type:      "notification",
		TenantID:  notification.TenantID,
		Timestamp: time.Now(),
		Payload:   payload,
	}
	s.wsHub.BroadcastToTenant(context.Background(), notification.TenantID, msg)

	logger.GetLogger().Info("Real-time notification sent",
		zap.String("notification_id", notification.ID.String()),
		zap.String("tenant_id", notification.TenantID.String()),
	)
}

// CreateFromMessage creates a notification from a new message
func (s *Service) CreateFromMessage(ctx context.Context, message *models.Message) (*Notification, error) {
	return s.Create(ctx, CreateNotificationInput{
		TenantID: message.TenantID,
		UserID:   *message.SenderUserID,
		Type:     "message",
		Title:    "رسالة جديدة",
		Body:     message.Content,
		Data: map[string]interface{}{
			"message_id":   message.ID.String(),
			"message_type": message.MessageType,
			"sender_name":  message.SenderName,
		},
	})
}

// CreateFromDeliveryStatus creates a notification from delivery status change
func (s *Service) CreateFromDeliveryStatus(ctx context.Context, tenantID, userID uuid.UUID, messageID string, status string) (*Notification, error) {
	title := "تحديث حالة التسليم"
	body := fmt.Sprintf("تم تسليم الرسالة بنجاح")
	
	if status == "failed" {
		body = "فشل في تسليم الرسالة"
	} else if status == "pending" {
		body = "الرسالة قيد الانتظار"
	}

	return s.Create(ctx, CreateNotificationInput{
		TenantID: tenantID,
		UserID:   userID,
		Type:     "delivery",
		Title:    title,
		Body:     body,
		Data: map[string]interface{}{
			"message_id": messageID,
			"status":     status,
		},
	})
}