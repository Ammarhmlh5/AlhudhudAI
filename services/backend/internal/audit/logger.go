package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/al-hudhud/backend/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Legal compliance event types
const (
	EventConsentGranted    = "consent_granted"
	EventConsentRevoked    = "consent_revoked"
	EventDataExported      = "data_exported"
	EventDataDeleted       = "data_deleted"
	EventFraudDetected     = "fraud_detected"
	EventAccountSuspended  = "account_suspended"
	EventAccountDeleted    = "account_deleted"
	EventPaymentProcessed  = "payment_processed"
	EventPaymentFailed     = "payment_failed"
	EventSubscriptionChanged = "subscription_changed"
)

// Priority levels for audit events
const (
	PriorityLow      = "low"
	PriorityMedium   = "medium"
	PriorityHigh     = "high"
	PriorityCritical = "critical"
)

// Logger implements AuditLogger interface
type Logger struct {
	db *sql.DB
}

// NewLogger creates a new audit logger
func NewLogger(db *sql.DB) *Logger {
	return &Logger{
		db: db,
	}
}

// Log logs an audit entry
func (l *Logger) Log(entry AuditEntry) error {
	log := logger.GetLogger()

	// Marshal details to JSON
	var detailsJSON []byte
	var err error
	if entry.Details != nil {
		detailsJSON, err = json.Marshal(entry.Details)
		if err != nil {
			log.Error("Failed to marshal audit details", zap.Error(err))
			detailsJSON = []byte("{}")
		}
	}

	// Set created_at if not set
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}

	// Insert audit entry
	query := `
		INSERT INTO audit_trail (
			user_id, tenant_id, action, resource, resource_id,
			ip_address, user_agent, result, error_message, details, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id
	`

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = l.db.QueryRowContext(
		ctx,
		query,
		entry.UserID,
		entry.TenantID,
		entry.Action,
		entry.Resource,
		entry.ResourceID,
		entry.IPAddress,
		entry.UserAgent,
		entry.Result,
		entry.ErrorMessage,
		detailsJSON,
		entry.CreatedAt,
	).Scan(&entry.ID)

	if err != nil {
		log.Error("Failed to insert audit entry",
			zap.String("action", entry.Action),
			zap.String("resource", entry.Resource),
			zap.Error(err))
		return fmt.Errorf("failed to insert audit entry: %w", err)
	}

	log.Debug("Audit entry logged",
		zap.Int64("id", entry.ID),
		zap.String("action", entry.Action),
		zap.String("resource", entry.Resource),
		zap.String("result", entry.Result))

	return nil
}

// LogSuccess logs a successful operation
func (l *Logger) LogSuccess(action, resource, resourceID string, userID, tenantID *uuid.UUID, ipAddress string, details map[string]interface{}) error {
	entry := AuditEntry{
		UserID:     userID,
		TenantID:   tenantID,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		IPAddress:  ipAddress,
		Result:     ResultSuccess,
		Details:    details,
	}
	return l.Log(entry)
}

// LogFailure logs a failed operation
func (l *Logger) LogFailure(action, resource, resourceID string, userID, tenantID *uuid.UUID, ipAddress string, errorMsg string, details map[string]interface{}) error {
	entry := AuditEntry{
		UserID:       userID,
		TenantID:     tenantID,
		Action:       action,
		Resource:     resource,
		ResourceID:   resourceID,
		IPAddress:    ipAddress,
		Result:       ResultFailure,
		ErrorMessage: errorMsg,
		Details:      details,
	}
	return l.Log(entry)
}

// LogLegalEvent logs legal compliance events with high priority
// Requirements: 36.10, 36.11
func (l *Logger) LogLegalEvent(ctx context.Context, event *LegalEvent) error {
	log := logger.GetLogger()

	// Marshal details to JSON
	detailsJSON, err := json.Marshal(event.Details)
	if err != nil {
		log.Error("Failed to marshal legal event details", zap.Error(err))
		detailsJSON = []byte("{}")
	}

	// Set created_at if not set
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}

	// Insert legal event with high priority
	query := `
		INSERT INTO audit_trail (
			user_id, tenant_id, action, resource, resource_id,
			ip_address, user_agent, result, details, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`

	err = l.db.QueryRowContext(
		ctx,
		query,
		event.UserID,
		event.TenantID,
		event.EventType,
		"legal_compliance",
		event.ResourceID,
		event.IPAddress,
		event.UserAgent,
		ResultSuccess,
		detailsJSON,
		event.CreatedAt,
	).Scan(&event.ID)

	if err != nil {
		log.Error("Failed to insert legal event",
			zap.String("event_type", event.EventType),
			zap.Error(err))
		return fmt.Errorf("failed to insert legal event: %w", err)
	}

	log.Info("Legal event logged",
		zap.Int64("id", event.ID),
		zap.String("event_type", event.EventType),
		zap.String("priority", PriorityHigh))

	return nil
}

// LogConsentGranted logs when user grants consent
func (l *Logger) LogConsentGranted(ctx context.Context, userID, tenantID *uuid.UUID, consentType, version, ipAddress string) error {
	event := &LegalEvent{
		UserID:     userID,
		TenantID:   tenantID,
		EventType:  EventConsentGranted,
		IPAddress:  ipAddress,
		Details: map[string]interface{}{
			"consent_type": consentType,
			"version":      version,
		},
	}
	return l.LogLegalEvent(ctx, event)
}

// LogConsentRevoked logs when user revokes consent
func (l *Logger) LogConsentRevoked(ctx context.Context, userID, tenantID *uuid.UUID, consentType, ipAddress string) error {
	event := &LegalEvent{
		UserID:     userID,
		TenantID:   tenantID,
		EventType:  EventConsentRevoked,
		IPAddress:  ipAddress,
		Details: map[string]interface{}{
			"consent_type": consentType,
		},
	}
	return l.LogLegalEvent(ctx, event)
}

// LogDataExported logs when user exports their data
func (l *Logger) LogDataExported(ctx context.Context, userID, tenantID *uuid.UUID, exportID, ipAddress string) error {
	event := &LegalEvent{
		UserID:     userID,
		TenantID:   tenantID,
		EventType:  EventDataExported,
		ResourceID: exportID,
		IPAddress:  ipAddress,
		Details: map[string]interface{}{
			"export_id": exportID,
		},
	}
	return l.LogLegalEvent(ctx, event)
}

// LogDataDeleted logs when user data is deleted
func (l *Logger) LogDataDeleted(ctx context.Context, userID, tenantID *uuid.UUID, deletionRequestID, ipAddress string) error {
	event := &LegalEvent{
		UserID:     userID,
		TenantID:   tenantID,
		EventType:  EventDataDeleted,
		ResourceID: deletionRequestID,
		IPAddress:  ipAddress,
		Details: map[string]interface{}{
			"deletion_request_id": deletionRequestID,
		},
	}
	return l.LogLegalEvent(ctx, event)
}

// LogFraudDetected logs when fraud is detected
func (l *Logger) LogFraudDetected(ctx context.Context, userID, tenantID *uuid.UUID, alertID int64, severity, ipAddress string) error {
	event := &LegalEvent{
		UserID:     userID,
		TenantID:   tenantID,
		EventType:  EventFraudDetected,
		ResourceID: fmt.Sprintf("%d", alertID),
		IPAddress:  ipAddress,
		Details: map[string]interface{}{
			"alert_id": alertID,
			"severity": severity,
		},
	}
	return l.LogLegalEvent(ctx, event)
}

// LogAccountSuspended logs when account is suspended
func (l *Logger) LogAccountSuspended(ctx context.Context, userID, tenantID *uuid.UUID, reason, ipAddress string) error {
	event := &LegalEvent{
		UserID:     userID,
		TenantID:   tenantID,
		EventType:  EventAccountSuspended,
		IPAddress:  ipAddress,
		Details: map[string]interface{}{
			"reason": reason,
		},
	}
	return l.LogLegalEvent(ctx, event)
}

// Query queries audit logs with filters
func (l *Logger) Query(filters AuditQueryFilters) ([]AuditEntry, error) {
	log := logger.GetLogger()

	// Build query
	query := `
		SELECT 
			id, user_id, tenant_id, action, resource, resource_id,
			ip_address, user_agent, result, error_message, details, created_at
		FROM audit_trail
		WHERE 1=1
	`
	args := []interface{}{}
	argCount := 1

	// Add filters
	if filters.UserID != nil {
		query += fmt.Sprintf(" AND user_id = $%d", argCount)
		args = append(args, filters.UserID)
		argCount++
	}

	if filters.TenantID != nil {
		query += fmt.Sprintf(" AND tenant_id = $%d", argCount)
		args = append(args, filters.TenantID)
		argCount++
	}

	if filters.Action != "" {
		query += fmt.Sprintf(" AND action = $%d", argCount)
		args = append(args, filters.Action)
		argCount++
	}

	if filters.Resource != "" {
		query += fmt.Sprintf(" AND resource = $%d", argCount)
		args = append(args, filters.Resource)
		argCount++
	}

	if filters.ResourceID != "" {
		query += fmt.Sprintf(" AND resource_id = $%d", argCount)
		args = append(args, filters.ResourceID)
		argCount++
	}

	if filters.Result != "" {
		query += fmt.Sprintf(" AND result = $%d", argCount)
		args = append(args, filters.Result)
		argCount++
	}

	if filters.StartDate != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argCount)
		args = append(args, filters.StartDate)
		argCount++
	}

	if filters.EndDate != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argCount)
		args = append(args, filters.EndDate)
		argCount++
	}

	// Order by created_at DESC
	query += " ORDER BY created_at DESC"

	// Add limit and offset
	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, filters.Limit)
		argCount++
	} else {
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, 100) // Default limit
		argCount++
	}

	if filters.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, filters.Offset)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := l.db.QueryContext(ctx, query, args...)
	if err != nil {
		log.Error("Failed to query audit logs", zap.Error(err))
		return nil, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer rows.Close()

	entries := []AuditEntry{}
	for rows.Next() {
		var entry AuditEntry
		var detailsJSON []byte
		var userID, tenantID sql.NullString
		var resourceID, userAgent, errorMessage sql.NullString

		err := rows.Scan(
			&entry.ID,
			&userID,
			&tenantID,
			&entry.Action,
			&entry.Resource,
			&resourceID,
			&entry.IPAddress,
			&userAgent,
			&entry.Result,
			&errorMessage,
			&detailsJSON,
			&entry.CreatedAt,
		)
		if err != nil {
			log.Error("Failed to scan audit entry", zap.Error(err))
			continue
		}

		// Parse UUIDs
		if userID.Valid {
			uid, err := uuid.Parse(userID.String)
			if err == nil {
				entry.UserID = &uid
			}
		}

		if tenantID.Valid {
			tid, err := uuid.Parse(tenantID.String)
			if err == nil {
				entry.TenantID = &tid
			}
		}

		// Parse optional fields
		if resourceID.Valid {
			entry.ResourceID = resourceID.String
		}
		if userAgent.Valid {
			entry.UserAgent = userAgent.String
		}
		if errorMessage.Valid {
			entry.ErrorMessage = errorMessage.String
		}

		// Unmarshal details
		if len(detailsJSON) > 0 {
			var details map[string]interface{}
			if err := json.Unmarshal(detailsJSON, &details); err == nil {
				entry.Details = details
			}
		}

		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		log.Error("Error iterating audit entries", zap.Error(err))
		return nil, fmt.Errorf("error iterating audit entries: %w", err)
	}

	return entries, nil
}
