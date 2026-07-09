package consent

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Manager handles user consent operations
type Manager struct {
	db *sql.DB
}

// NewManager creates a new consent manager
func NewManager(db *sql.DB) *Manager {
	return &Manager{db: db}
}

// RecordConsent records user consent
// Requirements: 36.3, 36.4
func (m *Manager) RecordConsent(ctx context.Context, tenantID int64, req *ConsentRequest) error {
	query := `
		INSERT INTO user_consents (
			tenant_id, user_id, consent_type, version, granted,
			granted_at, ip_address, user_agent
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := m.db.ExecContext(ctx, query,
		tenantID,
		req.UserID,
		req.Type,
		req.Version,
		req.Granted,
		time.Now(),
		req.IPAddress,
		req.UserAgent,
	)

	if err != nil {
		return fmt.Errorf("failed to record consent: %w", err)
	}

	return nil
}

// GetUserConsents retrieves all consents for a user
// Requirements: 36.3
func (m *Manager) GetUserConsents(ctx context.Context, tenantID, userID int64) ([]Consent, error) {
	query := `
		SELECT 
			id, tenant_id, user_id, consent_type, version, granted,
			granted_at, revoked_at, ip_address, user_agent, created_at, updated_at
		FROM user_consents
		WHERE tenant_id = $1 AND user_id = $2
		ORDER BY granted_at DESC
	`

	rows, err := m.db.QueryContext(ctx, query, tenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user consents: %w", err)
	}
	defer rows.Close()

	var consents []Consent
	for rows.Next() {
		var c Consent
		err := rows.Scan(
			&c.ID, &c.TenantID, &c.UserID, &c.Type, &c.Version, &c.Granted,
			&c.GrantedAt, &c.RevokedAt, &c.IPAddress, &c.UserAgent, &c.CreatedAt, &c.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan consent: %w", err)
		}
		consents = append(consents, c)
	}

	return consents, nil
}

// RevokeConsent revokes a specific consent
// Requirements: 36.4
func (m *Manager) RevokeConsent(ctx context.Context, tenantID, userID int64, consentType ConsentType) error {
	query := `
		UPDATE user_consents
		SET revoked_at = $1, granted = false
		WHERE tenant_id = $2 AND user_id = $3 AND consent_type = $4 AND revoked_at IS NULL
	`

	result, err := m.db.ExecContext(ctx, query, time.Now(), tenantID, userID, consentType)
	if err != nil {
		return fmt.Errorf("failed to revoke consent: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no active consent found to revoke")
	}

	return nil
}

// CheckConsent checks if user has granted specific consent
// Requirements: 36.3
func (m *Manager) CheckConsent(ctx context.Context, tenantID, userID int64, consentType ConsentType) (bool, error) {
	query := `
		SELECT granted
		FROM user_consents
		WHERE tenant_id = $1 AND user_id = $2 AND consent_type = $3
		AND revoked_at IS NULL
		ORDER BY granted_at DESC
		LIMIT 1
	`

	var granted bool
	err := m.db.QueryRowContext(ctx, query, tenantID, userID, consentType).Scan(&granted)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check consent: %w", err)
	}

	return granted, nil
}

// GetLatestConsent retrieves the latest consent for a specific type
// Requirements: 36.3
func (m *Manager) GetLatestConsent(ctx context.Context, tenantID, userID int64, consentType ConsentType) (*Consent, error) {
	query := `
		SELECT 
			id, tenant_id, user_id, consent_type, version, granted,
			granted_at, revoked_at, ip_address, user_agent, created_at, updated_at
		FROM user_consents
		WHERE tenant_id = $1 AND user_id = $2 AND consent_type = $3
		ORDER BY granted_at DESC
		LIMIT 1
	`

	var c Consent
	err := m.db.QueryRowContext(ctx, query, tenantID, userID, consentType).Scan(
		&c.ID, &c.TenantID, &c.UserID, &c.Type, &c.Version, &c.Granted,
		&c.GrantedAt, &c.RevokedAt, &c.IPAddress, &c.UserAgent, &c.CreatedAt, &c.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get latest consent: %w", err)
	}

	return &c, nil
}
