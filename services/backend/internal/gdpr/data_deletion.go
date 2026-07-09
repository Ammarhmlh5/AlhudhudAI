package gdpr

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// DataDeletionManager handles GDPR data deletion requests
type DataDeletionManager struct {
	db *sql.DB
}

// NewDataDeletionManager creates a new data deletion manager
func NewDataDeletionManager(db *sql.DB) *DataDeletionManager {
	return &DataDeletionManager{db: db}
}

// RequestDeletion creates a data deletion request
// Requirements: 36.5
func (m *DataDeletionManager) RequestDeletion(ctx context.Context, tenantID, userID int64, reason string) (int64, error) {
	query := `
		INSERT INTO data_deletion_requests (
			tenant_id, user_id, requested_at, status, reason
		) VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	var requestID int64
	err := m.db.QueryRowContext(ctx, query,
		tenantID,
		userID,
		time.Now(),
		DeletionStatusPending,
		reason,
	).Scan(&requestID)

	if err != nil {
		return 0, fmt.Errorf("failed to create deletion request: %w", err)
	}

	// TODO: Send confirmation email
	// TODO: Schedule deletion job (after grace period)

	return requestID, nil
}

// ProcessDeletion processes a deletion request
// Requirements: 36.5, 36.6, 36.7
func (m *DataDeletionManager) ProcessDeletion(ctx context.Context, requestID int64) error {
	// Get deletion request
	var req DeletionRequest
	query := `
		SELECT id, tenant_id, user_id, status
		FROM data_deletion_requests
		WHERE id = $1
	`
	err := m.db.QueryRowContext(ctx, query, requestID).Scan(
		&req.ID, &req.TenantID, &req.UserID, &req.Status,
	)
	if err != nil {
		return fmt.Errorf("failed to get deletion request: %w", err)
	}

	// Check if already processed
	if req.Status != DeletionStatusPending {
		return fmt.Errorf("deletion request already processed: %s", req.Status)
	}

	// Update status to processing
	_, err = m.db.ExecContext(ctx, `
		UPDATE data_deletion_requests
		SET status = $1, updated_at = $2
		WHERE id = $3
	`, DeletionStatusProcessing, time.Now(), requestID)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	// Start transaction for deletion
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete personal data
	err = m.deletePersonalDataTx(ctx, tx, req.TenantID, req.UserID)
	if err != nil {
		m.updateDeletionStatus(ctx, requestID, DeletionStatusFailed, err.Error())
		return fmt.Errorf("failed to delete personal data: %w", err)
	}

	// Anonymize financial records
	err = m.anonymizeFinancialRecordsTx(ctx, tx, req.TenantID, req.UserID)
	if err != nil {
		m.updateDeletionStatus(ctx, requestID, DeletionStatusFailed, err.Error())
		return fmt.Errorf("failed to anonymize financial records: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		m.updateDeletionStatus(ctx, requestID, DeletionStatusFailed, err.Error())
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Update status to completed
	now := time.Now()
	_, err = m.db.ExecContext(ctx, `
		UPDATE data_deletion_requests
		SET status = $1, processed_at = $2, updated_at = $3
		WHERE id = $4
	`, DeletionStatusCompleted, now, now, requestID)

	return err
}

// deletePersonalDataTx deletes all personal data for a user within a transaction
// Requirements: 36.6
func (m *DataDeletionManager) deletePersonalDataTx(ctx context.Context, tx *sql.Tx, tenantID, userID int64) error {
	// Whitelist of allowed tables for deletion (SQL injection prevention)
	allowedTables := map[string]bool{
		"user_consents":  true,
		"notifications":  true,
		// Add more tables as needed
	}

	// Tables to delete from (personal data)
	tables := []string{
		"user_consents",
		"notifications",
		// Add more tables as needed
	}

	for _, table := range tables {
		// Validate table name against whitelist (SQL injection prevention)
		if !allowedTables[table] {
			return fmt.Errorf("invalid table name: %s", table)
		}
		
		// Safe to use table name after whitelist validation
		query := fmt.Sprintf("DELETE FROM %s WHERE tenant_id = $1 AND user_id = $2", table)
		_, err := tx.ExecContext(ctx, query, tenantID, userID)
		if err != nil {
			return fmt.Errorf("failed to delete from %s: %w", table, err)
		}
	}

	// Delete user record (or anonymize)
	_, err := tx.ExecContext(ctx, `
		UPDATE users
		SET 
			email = 'deleted_' || id || '@deleted.local',
			phone = NULL,
			first_name = 'Deleted',
			last_name = 'User',
			deleted_at = $1
		WHERE tenant_id = $2 AND id = $3
	`, time.Now(), tenantID, userID)

	return err
}

// anonymizeFinancialRecordsTx anonymizes financial records within a transaction
// Requirements: 36.7
func (m *DataDeletionManager) anonymizeFinancialRecordsTx(ctx context.Context, tx *sql.Tx, tenantID, userID int64) error {
	// Anonymize accounting transactions (keep for audit)
	_, err := tx.ExecContext(ctx, `
		UPDATE accounting_transactions
		SET 
			description = 'Anonymized transaction',
			metadata = NULL
		WHERE tenant_id = $1 AND user_id = $2
	`, tenantID, userID)
	if err != nil {
		return fmt.Errorf("failed to anonymize accounting_transactions: %w", err)
	}

	// Anonymize balance history (keep for audit)
	_, err = tx.ExecContext(ctx, `
		UPDATE balance_history
		SET 
			reason = 'Anonymized',
			metadata = NULL
		WHERE tenant_id = $1 AND user_id = $2
	`, tenantID, userID)
	if err != nil {
		return fmt.Errorf("failed to anonymize balance_history: %w", err)
	}

	// Anonymize subscriptions
	_, err = tx.ExecContext(ctx, `
		UPDATE subscriptions
		SET 
			user_id = NULL,
			metadata = NULL
		WHERE tenant_id = $1 AND user_id = $2
	`, tenantID, userID)
	if err != nil {
		return fmt.Errorf("failed to anonymize subscriptions: %w", err)
	}

	return nil
}

// updateDeletionStatus updates the status of a deletion request
func (m *DataDeletionManager) updateDeletionStatus(ctx context.Context, requestID int64, status DeletionStatus, errorMsg string) error {
	_, err := m.db.ExecContext(ctx, `
		UPDATE data_deletion_requests
		SET status = $1, error_msg = $2, updated_at = $3
		WHERE id = $4
	`, status, errorMsg, time.Now(), requestID)
	return err
}

// GetDeletionRequest retrieves a deletion request by ID
func (m *DataDeletionManager) GetDeletionRequest(ctx context.Context, requestID int64) (*DeletionRequest, error) {
	query := `
		SELECT 
			id, tenant_id, user_id, requested_at, processed_at,
			status, reason, error_msg, created_at, updated_at
		FROM data_deletion_requests
		WHERE id = $1
	`

	var req DeletionRequest
	err := m.db.QueryRowContext(ctx, query, requestID).Scan(
		&req.ID, &req.TenantID, &req.UserID, &req.RequestedAt, &req.ProcessedAt,
		&req.Status, &req.Reason, &req.ErrorMsg, &req.CreatedAt, &req.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get deletion request: %w", err)
	}

	return &req, nil
}

// ListDeletionRequests lists all deletion requests for a tenant
func (m *DataDeletionManager) ListDeletionRequests(ctx context.Context, tenantID int64, status DeletionStatus) ([]DeletionRequest, error) {
	query := `
		SELECT 
			id, tenant_id, user_id, requested_at, processed_at,
			status, reason, error_msg, created_at, updated_at
		FROM data_deletion_requests
		WHERE tenant_id = $1
	`

	args := []interface{}{tenantID}
	if status != "" {
		query += " AND status = $2"
		args = append(args, status)
	}

	query += " ORDER BY requested_at DESC"

	rows, err := m.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list deletion requests: %w", err)
	}
	defer rows.Close()

	var requests []DeletionRequest
	for rows.Next() {
		var req DeletionRequest
		err := rows.Scan(
			&req.ID, &req.TenantID, &req.UserID, &req.RequestedAt, &req.ProcessedAt,
			&req.Status, &req.Reason, &req.ErrorMsg, &req.CreatedAt, &req.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan deletion request: %w", err)
		}
		requests = append(requests, req)
	}

	return requests, nil
}
