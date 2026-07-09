package notifications

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/al-hudhud/backend/internal/models"
	"github.com/al-hudhud/backend/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// SettingsService handles notification settings operations
type SettingsService struct {
	db *sql.DB
}

// NewSettingsService creates a new notification settings service
func NewSettingsService(db *sql.DB) *SettingsService {
	return &SettingsService{db: db}
}

// NotificationSetting represents a notification setting
type NotificationSetting struct {
	ID               uuid.UUID       `json:"id"`
	TenantID         uuid.UUID       `json:"tenant_id"`
	NotificationType string          `json:"notification_type"`
	IsEnabled        bool            `json:"is_enabled"`
	Config           json.RawMessage `json:"config"`
	CreatedAt        string          `json:"created_at"`
	UpdatedAt        string          `json:"updated_at"`
}

// NotificationTemplate represents a notification template
type NotificationTemplate struct {
	ID               uuid.UUID       `json:"id"`
	TenantID         *uuid.UUID      `json:"tenant_id,omitempty"`
	NotificationType string          `json:"notification_type"`
	TitleTemplate    string          `json:"title_template"`
	BodyTemplate     string          `json:"body_template"`
	IsDefault        bool            `json:"is_default"`
	CreatedAt        string          `json:"created_at"`
	UpdatedAt        string          `json:"updated_at"`
}

// NotificationLog represents a notification log entry
type NotificationLog struct {
	ID               uuid.UUID  `json:"id"`
	TenantID         uuid.UUID  `json:"tenant_id"`
	UserID           *uuid.UUID `json:"user_id,omitempty"`
	NotificationType string     `json:"notification_type"`
	Title            string     `json:"title"`
	Body             string     `json:"body"`
	SentAt           string     `json:"sent_at"`
	Status           string     `json:"status"`
	ErrorMessage     *string    `json:"error_message,omitempty"`
}

// GetSettings retrieves all notification settings for a tenant
func (s *SettingsService) GetSettings(ctx context.Context, tenantID uuid.UUID) ([]NotificationSetting, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, tenant_id, notification_type, is_enabled, config, 
		       created_at::text, updated_at::text
		FROM notification_settings
		WHERE tenant_id = $1
		ORDER BY notification_type
	`, tenantID)

	if err != nil {
		logger.GetLogger().Error("Failed to get notification settings", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var settings []NotificationSetting
	for rows.Next() {
		var setting NotificationSetting
		err := rows.Scan(
			&setting.ID, &setting.TenantID, &setting.NotificationType,
			&setting.IsEnabled, &setting.Config, &setting.CreatedAt, &setting.UpdatedAt,
		)
		if err != nil {
			logger.GetLogger().Error("Failed to scan notification setting", zap.Error(err))
			continue
		}
		settings = append(settings, setting)
	}

	return settings, nil
}

// GetSetting retrieves a specific notification setting
func (s *SettingsService) GetSetting(ctx context.Context, tenantID uuid.UUID, notificationType string) (*NotificationSetting, error) {
	var setting NotificationSetting
	err := s.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, notification_type, is_enabled, config, 
		       created_at::text, updated_at::text
		FROM notification_settings
		WHERE tenant_id = $1 AND notification_type = $2
	`, tenantID, notificationType).Scan(
		&setting.ID, &setting.TenantID, &setting.NotificationType,
		&setting.IsEnabled, &setting.Config, &setting.CreatedAt, &setting.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		logger.GetLogger().Error("Failed to get notification setting", zap.Error(err))
		return nil, err
	}

	return &setting, nil
}

// UpdateSetting updates a notification setting
func (s *SettingsService) UpdateSetting(ctx context.Context, tenantID uuid.UUID, notificationType string, isEnabled bool, config json.RawMessage) (*NotificationSetting, error) {
	var setting NotificationSetting
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO notification_settings (tenant_id, notification_type, is_enabled, config, updated_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (tenant_id, notification_type) 
		DO UPDATE SET is_enabled = $3, config = $4, updated_at = NOW()
		RETURNING id, tenant_id, notification_type, is_enabled, config, created_at::text, updated_at::text
	`, tenantID, notificationType, isEnabled, config).Scan(
		&setting.ID, &setting.TenantID, &setting.NotificationType,
		&setting.IsEnabled, &setting.Config, &setting.CreatedAt, &setting.UpdatedAt,
	)

	if err != nil {
		logger.GetLogger().Error("Failed to update notification setting", zap.Error(err))
		return nil, err
	}

	logger.GetLogger().Info("Notification setting updated",
		zap.String("tenant_id", tenantID.String()),
		zap.String("notification_type", notificationType),
		zap.Bool("is_enabled", isEnabled),
	)

	return &setting, nil
}

// IsNotificationEnabled checks if a notification type is enabled for a tenant
func (s *SettingsService) IsNotificationEnabled(ctx context.Context, tenantID uuid.UUID, notificationType string) (bool, error) {
	var isEnabled bool
	err := s.db.QueryRowContext(ctx, `
		SELECT is_enabled FROM notification_settings
		WHERE tenant_id = $1 AND notification_type = $2
	`, tenantID, notificationType).Scan(&isEnabled)

	if err == sql.ErrNoRows {
		// Default to true if no setting exists
		return true, nil
	}
	if err != nil {
		logger.GetLogger().Error("Failed to check notification enabled", zap.Error(err))
		return false, err
	}

	return isEnabled, nil
}

// GetNotificationConfig returns the config for a notification type
func (s *SettingsService) GetNotificationConfig(ctx context.Context, tenantID uuid.UUID, notificationType string) (map[string]interface{}, error) {
	var config json.RawMessage
	err := s.db.QueryRowContext(ctx, `
		SELECT config FROM notification_settings
		WHERE tenant_id = $1 AND notification_type = $2
	`, tenantID, notificationType).Scan(&config)

	if err == sql.ErrNoRows {
		return make(map[string]interface{}), nil
	}
	if err != nil {
		logger.GetLogger().Error("Failed to get notification config", zap.Error(err))
		return nil, err
	}

	var result map[string]interface{}
	if config != nil {
		json.Unmarshal(config, &result)
	}

	return result, nil
}

// GetTemplates retrieves notification templates
func (s *SettingsService) GetTemplates(ctx context.Context, tenantID *uuid.UUID) ([]NotificationTemplate, error) {
	var query string
	var args []interface{}

	if tenantID == nil {
		query = `
			SELECT id, tenant_id, notification_type, title_template, body_template, 
			       is_default, created_at::text, updated_at::text
			FROM notification_templates
			WHERE tenant_id IS NULL
			ORDER BY notification_type
		`
	} else {
		query = `
			SELECT id, tenant_id, notification_type, title_template, body_template, 
			       is_default, created_at::text, updated_at::text
			FROM notification_templates
			WHERE tenant_id IS NULL OR tenant_id = $1
			ORDER BY notification_type, is_default DESC
		`
		args = append(args, *tenantID)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		logger.GetLogger().Error("Failed to get notification templates", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var templates []NotificationTemplate
	for rows.Next() {
		var template NotificationTemplate
		err := rows.Scan(
			&template.ID, &template.TenantID, &template.NotificationType,
			&template.TitleTemplate, &template.BodyTemplate, &template.IsDefault,
			&template.CreatedAt, &template.UpdatedAt,
		)
		if err != nil {
			logger.GetLogger().Error("Failed to scan notification template", zap.Error(err))
			continue
		}
		templates = append(templates, template)
	}

	return templates, nil
}

// UpdateTemplate updates or creates a notification template
func (s *SettingsService) UpdateTemplate(ctx context.Context, tenantID *uuid.UUID, notificationType, titleTemplate, bodyTemplate string) (*NotificationTemplate, error) {
	var template NotificationTemplate
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO notification_templates (tenant_id, notification_type, title_template, body_template, is_default)
		VALUES ($1, $2, $3, $4, false)
		ON CONFLICT (tenant_id, notification_type) 
		DO UPDATE SET title_template = $3, body_template = $4, updated_at = NOW()
		RETURNING id, tenant_id, notification_type, title_template, body_template, is_default, created_at::text, updated_at::text
	`, tenantID, notificationType, titleTemplate, bodyTemplate).Scan(
		&template.ID, &template.TenantID, &template.NotificationType,
		&template.TitleTemplate, &template.BodyTemplate, &template.IsDefault,
		&template.CreatedAt, &template.UpdatedAt,
	)

	if err != nil {
		logger.GetLogger().Error("Failed to update notification template", zap.Error(err))
		return nil, err
	}

	return &template, nil
}

// LogNotification logs a sent notification
func (s *SettingsService) LogNotification(ctx context.Context, tenantID, userID uuid.UUID, notificationType, title, body, status string, errorMessage *string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO notification_logs (tenant_id, user_id, notification_type, title, body, status, error_message)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, tenantID, userID, notificationType, title, body, status, errorMessage)

	if err != nil {
		logger.GetLogger().Error("Failed to log notification", zap.Error(err))
		return err
	}

	return nil
}

// GetNotificationLogs retrieves notification logs for a tenant
func (s *SettingsService) GetNotificationLogs(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]NotificationLog, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, tenant_id, user_id, notification_type, title, body, 
		       sent_at::text, status, error_message
		FROM notification_logs
		WHERE tenant_id = $1
		ORDER BY sent_at DESC
		LIMIT $2 OFFSET $3
	`, tenantID, limit, offset)

	if err != nil {
		logger.GetLogger().Error("Failed to get notification logs", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var logs []NotificationLog
	for rows.Next() {
		var log NotificationLog
		err := rows.Scan(
			&log.ID, &log.TenantID, &log.UserID, &log.NotificationType,
			&log.Title, &log.Body, &log.SentAt, &log.Status, &log.ErrorMessage,
		)
		if err != nil {
			logger.GetLogger().Error("Failed to scan notification log", zap.Error(err))
			continue
		}
		logs = append(logs, log)
	}

	return logs, nil
}

// GetNotificationStats returns notification statistics for a tenant
func (s *SettingsService) GetNotificationStats(ctx context.Context, tenantID uuid.UUID) ([]models.NotificationStats, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT 
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'sent') as sent,
			COUNT(*) FILTER (WHERE status = 'failed') as failed,
			COUNT(*) FILTER (WHERE notification_type = 'delivery') as deliveries,
			COUNT(*) FILTER (WHERE notification_type = 'status') as statuses,
			COUNT(*) FILTER (WHERE notification_type = 'system') as systems,
			COUNT(*) FILTER (WHERE notification_type = 'alert') as alerts
		FROM notification_logs
		WHERE tenant_id = $1
	`, tenantID)

	if err != nil {
		logger.GetLogger().Error("Failed to get notification stats", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var stats []models.NotificationStats
	for rows.Next() {
		var stat models.NotificationStats
		err := rows.Scan(
			&stat.Total, &stat.Unread, &stat.Messages, &stat.Deliveries,
			&stat.Statuses, &stat.Systems, &stat.Alerts,
		)
		if err != nil {
			logger.GetLogger().Error("Failed to scan notification stat", zap.Error(err))
			continue
		}
		stats = append(stats, stat)
	}

	return stats, nil
}

// SetDefaultSettings sets default notification settings for a new tenant
func (s *SettingsService) SetDefaultSettings(ctx context.Context, tenantID uuid.UUID) error {
	notificationTypes := []string{
		"reconnection_alert",
		"package_expiry_warning",
		"package_expiry_alert",
		"low_balance_warning",
		"message_pending_alert",
		"delivery_failed_alert",
		"system_update",
		"new_message",
	}

	for _, notificationType := range notificationTypes {
		isEnabled := true
		if notificationType == "new_message" {
			isEnabled = false // Disabled by default
		}

		_, err := s.db.ExecContext(ctx, `
			INSERT INTO notification_settings (tenant_id, notification_type, is_enabled, config)
			VALUES ($1, $2, $3, '{}')
			ON CONFLICT (tenant_id, notification_type) DO NOTHING
		`, tenantID, notificationType, isEnabled)

		if err != nil {
			logger.GetLogger().Error("Failed to set default notification setting",
				zap.String("tenant_id", tenantID.String()),
				zap.String("notification_type", notificationType),
				zap.Error(err),
			)
			return err
		}
	}

	logger.GetLogger().Info("Default notification settings set",
		zap.String("tenant_id", tenantID.String()),
		zap.Int("settings_count", len(notificationTypes)),
	)

	return nil
}