package alerts

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// Manager handles alert operations
// Implements Requirements 6.1-6.4, 6.9
type Manager struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewManager creates a new alert manager
func NewManager(db *sqlx.DB, logger *zap.Logger) *Manager {
	return &Manager{
		db:     db,
		logger: logger,
	}
}

// CreateAlert creates a new alert
// Implements Requirements 6.1-6.4
func (m *Manager) CreateAlert(ctx context.Context, req *CreateAlertRequest) (*Alert, error) {
	// Validate request
	if err := m.validateCreateRequest(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Serialize context to JSON
	var contextJSON []byte
	var err error
	if req.Context != nil {
		contextJSON, err = json.Marshal(req.Context)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal context: %w", err)
		}
	}

	// Insert alert
	query := `
		INSERT INTO alerts (
			tenant_id, alert_type, severity, title, message, 
			context, status, dedup_key, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		) RETURNING id, created_at
	`

	var alert Alert
	err = m.db.QueryRowContext(
		ctx,
		query,
		req.TenantID,
		req.AlertType,
		req.Severity,
		req.Title,
		req.Message,
		contextJSON,
		AlertStatusActive,
		req.DedupKey,
		time.Now(),
	).Scan(&alert.ID, &alert.CreatedAt)

	if err != nil {
		m.logger.Error("Failed to create alert",
			zap.Error(err),
			zap.String("alert_type", string(req.AlertType)),
			zap.String("severity", string(req.Severity)),
		)
		return nil, fmt.Errorf("failed to create alert: %w", err)
	}

	// Populate alert fields
	alert.TenantID = req.TenantID
	alert.AlertType = req.AlertType
	alert.Severity = req.Severity
	alert.Title = req.Title
	alert.Message = req.Message
	alert.Context = req.Context
	alert.Status = AlertStatusActive
	alert.DedupKey = req.DedupKey

	m.logger.Info("Alert created",
		zap.Int64("alert_id", alert.ID),
		zap.String("alert_type", string(alert.AlertType)),
		zap.String("severity", string(alert.Severity)),
	)

	return &alert, nil
}

// GetAlerts retrieves alerts with filtering
// Implements Requirements 6.9
func (m *Manager) GetAlerts(ctx context.Context, filter *AlertFilter) (*AlertListResponse, error) {
	// Build query
	query := `
		SELECT 
			id, tenant_id, alert_type, severity, title, message,
			context, status, acknowledged_at, acknowledged_by,
			resolved_at, dedup_key, created_at
		FROM alerts
		WHERE 1=1
	`
	countQuery := `SELECT COUNT(*) FROM alerts WHERE 1=1`

	args := []interface{}{}
	argIndex := 1

	// Apply filters
	if filter.TenantID != nil {
		query += fmt.Sprintf(" AND tenant_id = $%d", argIndex)
		countQuery += fmt.Sprintf(" AND tenant_id = $%d", argIndex)
		args = append(args, filter.TenantID)
		argIndex++
	}

	if filter.AlertType != nil {
		query += fmt.Sprintf(" AND alert_type = $%d", argIndex)
		countQuery += fmt.Sprintf(" AND alert_type = $%d", argIndex)
		args = append(args, *filter.AlertType)
		argIndex++
	}

	if filter.Severity != nil {
		query += fmt.Sprintf(" AND severity = $%d", argIndex)
		countQuery += fmt.Sprintf(" AND severity = $%d", argIndex)
		args = append(args, *filter.Severity)
		argIndex++
	}

	if filter.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		countQuery += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, *filter.Status)
		argIndex++
	}

	if filter.StartTime != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argIndex)
		countQuery += fmt.Sprintf(" AND created_at >= $%d", argIndex)
		args = append(args, *filter.StartTime)
		argIndex++
	}

	if filter.EndTime != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argIndex)
		countQuery += fmt.Sprintf(" AND created_at <= $%d", argIndex)
		args = append(args, *filter.EndTime)
		argIndex++
	}

	// Get total count
	var totalCount int64
	err := m.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count alerts: %w", err)
	}

	// Add ordering and pagination
	query += " ORDER BY created_at DESC"

	limit := filter.Limit
	if limit <= 0 {
		limit = 50 // Default limit
	}
	query += fmt.Sprintf(" LIMIT $%d", argIndex)
	args = append(args, limit)
	argIndex++

	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}
	query += fmt.Sprintf(" OFFSET $%d", argIndex)
	args = append(args, offset)

	// Execute query
	rows, err := m.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query alerts: %w", err)
	}
	defer rows.Close()

	// Parse results
	alerts := []*Alert{}
	for rows.Next() {
		var alert Alert
		var contextJSON []byte

		err := rows.Scan(
			&alert.ID,
			&alert.TenantID,
			&alert.AlertType,
			&alert.Severity,
			&alert.Title,
			&alert.Message,
			&contextJSON,
			&alert.Status,
			&alert.AcknowledgedAt,
			&alert.AcknowledgedBy,
			&alert.ResolvedAt,
			&alert.DedupKey,
			&alert.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan alert: %w", err)
		}

		// Deserialize context
		if len(contextJSON) > 0 {
			if err := json.Unmarshal(contextJSON, &alert.Context); err != nil {
				m.logger.Warn("Failed to unmarshal alert context",
					zap.Int64("alert_id", alert.ID),
					zap.Error(err),
				)
			}
		}

		alerts = append(alerts, &alert)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating alerts: %w", err)
	}

	return &AlertListResponse{
		Alerts:     alerts,
		TotalCount: totalCount,
		Limit:      limit,
		Offset:     offset,
	}, nil
}

// GetAlertByID retrieves a single alert by ID.
// tenantID is REQUIRED to prevent cross-tenant data leakage.
func (m *Manager) GetAlertByID(ctx context.Context, alertID int64, tenantID uuid.UUID) (*Alert, error) {
	// SECURITY: always filter by both id AND tenant_id so a tenant
	// cannot read alerts that belong to another tenant.
	query := `
		SELECT
			id, tenant_id, alert_type, severity, title, message,
			context, status, acknowledged_at, acknowledged_by,
			resolved_at, dedup_key, created_at
		FROM alerts
		WHERE id = $1 AND tenant_id = $2
	`

	var alert Alert
	var contextJSON []byte

	err := m.db.QueryRowContext(ctx, query, alertID, tenantID).Scan(
		&alert.ID,
		&alert.TenantID,
		&alert.AlertType,
		&alert.Severity,
		&alert.Title,
		&alert.Message,
		&contextJSON,
		&alert.Status,
		&alert.AcknowledgedAt,
		&alert.AcknowledgedBy,
		&alert.ResolvedAt,
		&alert.DedupKey,
		&alert.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("alert not found: %d", alertID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get alert: %w", err)
	}

	// Deserialize context
	if len(contextJSON) > 0 {
		if err := json.Unmarshal(contextJSON, &alert.Context); err != nil {
			m.logger.Warn("Failed to unmarshal alert context",
				zap.Int64("alert_id", alert.ID),
				zap.Error(err),
			)
		}
	}

	return &alert, nil
}

// AcknowledgeAlert marks an alert as acknowledged
// Implements Requirements 6.9
func (m *Manager) AcknowledgeAlert(ctx context.Context, alertID int64, userID uuid.UUID) error {
	query := `
		UPDATE alerts
		SET 
			status = $1,
			acknowledged_at = $2,
			acknowledged_by = $3
		WHERE id = $4 AND status = $5
	`

	result, err := m.db.ExecContext(
		ctx,
		query,
		AlertStatusAcknowledged,
		time.Now(),
		userID,
		alertID,
		AlertStatusActive,
	)

	if err != nil {
		m.logger.Error("Failed to acknowledge alert",
			zap.Int64("alert_id", alertID),
			zap.Error(err),
		)
		return fmt.Errorf("failed to acknowledge alert: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("alert not found or already acknowledged: %d", alertID)
	}

	m.logger.Info("Alert acknowledged",
		zap.Int64("alert_id", alertID),
		zap.String("user_id", userID.String()),
	)

	return nil
}

// ResolveAlert marks an alert as resolved
// Implements Requirements 6.9
func (m *Manager) ResolveAlert(ctx context.Context, alertID int64) error {
	query := `
		UPDATE alerts
		SET 
			status = $1,
			resolved_at = $2
		WHERE id = $3 AND status != $4
	`

	result, err := m.db.ExecContext(
		ctx,
		query,
		AlertStatusResolved,
		time.Now(),
		alertID,
		AlertStatusResolved,
	)

	if err != nil {
		m.logger.Error("Failed to resolve alert",
			zap.Int64("alert_id", alertID),
			zap.Error(err),
		)
		return fmt.Errorf("failed to resolve alert: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("alert not found or already resolved: %d", alertID)
	}

	m.logger.Info("Alert resolved",
		zap.Int64("alert_id", alertID),
	)

	return nil
}

// validateCreateRequest validates the create alert request
func (m *Manager) validateCreateRequest(req *CreateAlertRequest) error {
	if req.AlertType == "" {
		return fmt.Errorf("alert_type is required")
	}

	if req.Severity == "" {
		return fmt.Errorf("severity is required")
	}

	if req.Title == "" {
		return fmt.Errorf("title is required")
	}

	if req.Message == "" {
		return fmt.Errorf("message is required")
	}

	// Validate alert type
	validTypes := map[AlertType]bool{
		AlertTypeRatingChange:       true,
		AlertTypeServerFailure:      true,
		AlertTypeThresholdViolation: true,
		AlertTypeDeviceFailure:      true,
	}
	if !validTypes[req.AlertType] {
		return fmt.Errorf("invalid alert_type: %s", req.AlertType)
	}

	// Validate severity
	validSeverities := map[AlertSeverity]bool{
		AlertSeverityCritical: true,
		AlertSeverityWarning:  true,
		AlertSeverityInfo:     true,
	}
	if !validSeverities[req.Severity] {
		return fmt.Errorf("invalid severity: %s", req.Severity)
	}

	return nil
}
