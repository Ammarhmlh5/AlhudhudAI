package monitoring

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/google/uuid"
)

// BalanceMonitor monitors telecom balance for dedicated gateways
// Implements Requirements 8.13, 8.14, 8.15, 17.8
//
// Important Note - Business Model:
// - Enterprise companies pay telecom providers directly for message costs
// - The platform only monitors the balance and sends alerts
// - The platform does NOT process any payments related to message costs
// - The platform only charges: (1) subscription fees, (2) server rental fees
type BalanceMonitor struct {
	db              *sql.DB
	alertManager    *AlertManager
	telecomClients  map[string]TelecomBalanceChecker // Map of telecom provider -> client
	checkInterval   time.Duration
}

// TelecomBalanceChecker interface for checking balance with telecom providers
type TelecomBalanceChecker interface {
	CheckBalance(ctx context.Context, credentials TelecomCredentials) (float64, error)
}

// TelecomCredentials holds credentials for telecom API
type TelecomCredentials struct {
	Provider    string
	APIEndpoint string
	Username    string
	Password    string // Decrypted
	APIKey      string // Decrypted
}

// GatewayBalanceInfo holds balance information for a gateway
type GatewayBalanceInfo struct {
	GatewayID              int
	TenantID               uuid.UUID
	GatewayName            string
	TelecomProvider        string
	CurrentBalance         float64
	BalanceAlertThreshold  float64
	LastBalanceCheck       time.Time
	BalanceCheckEnabled    bool
}

// NewBalanceMonitor creates a new balance monitor
func NewBalanceMonitor(db *sql.DB, alertManager *AlertManager) *BalanceMonitor {
	return &BalanceMonitor{
		db:             db,
		alertManager:   alertManager,
		telecomClients: make(map[string]TelecomBalanceChecker),
		checkInterval:  1 * time.Hour, // Check every hour by default
	}
}

// RegisterTelecomClient registers a telecom balance checker for a provider
func (bm *BalanceMonitor) RegisterTelecomClient(provider string, client TelecomBalanceChecker) {
	bm.telecomClients[provider] = client
}

// SetCheckInterval sets the interval for balance checking
func (bm *BalanceMonitor) SetCheckInterval(interval time.Duration) {
	bm.checkInterval = interval
}

// CheckTelecomBalance checks the telecom balance for a specific gateway
// Implements Requirement 8.13: Monitor company balance with telecom provider (read-only)
func (bm *BalanceMonitor) CheckTelecomBalance(ctx context.Context, gatewayID int, tenantID uuid.UUID) error {
	// Get gateway information
	gateway, err := bm.getGatewayInfo(ctx, gatewayID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to get gateway info: %w", err)
	}

	// Check if balance monitoring is enabled
	if !gateway.BalanceCheckEnabled {
		return nil // Balance monitoring is disabled for this gateway
	}

	// Get telecom client for this provider
	client, ok := bm.telecomClients[gateway.TelecomProvider]
	if !ok {
		return fmt.Errorf("no telecom client registered for provider: %s", gateway.TelecomProvider)
	}

	// Get decrypted credentials
	credentials, err := bm.getDecryptedCredentials(ctx, gatewayID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to get credentials: %w", err)
	}

	// Check balance with telecom provider
	balance, err := client.CheckBalance(ctx, credentials)
	if err != nil {
		return fmt.Errorf("failed to check balance with telecom provider: %w", err)
	}

	// Update balance in database
	if err := bm.updateBalance(ctx, gatewayID, tenantID, balance); err != nil {
		return fmt.Errorf("failed to update balance: %w", err)
	}

	// Check if balance is below threshold and send alert
	// Implements Requirement 8.14: Send alert when balance drops below threshold
	// Implements Requirement 17.8: Send alert to enterprise company
	if gateway.BalanceAlertThreshold > 0 && balance < gateway.BalanceAlertThreshold {
		if err := bm.sendLowBalanceAlert(ctx, gatewayID, tenantID, gateway.GatewayName, balance, gateway.BalanceAlertThreshold); err != nil {
			return fmt.Errorf("failed to send low balance alert: %w", err)
		}
	}

	return nil
}

// CheckAllGateways checks balance for all active gateways with balance monitoring enabled
func (bm *BalanceMonitor) CheckAllGateways(ctx context.Context) error {
	// Get all active gateways with balance monitoring enabled
	gateways, err := bm.getActiveGatewaysForBalanceCheck(ctx)
	if err != nil {
		return fmt.Errorf("failed to get active gateways: %w", err)
	}

	fmt.Printf("Checking balance for %d gateways\n", len(gateways))

	for _, gateway := range gateways {
		if err := bm.CheckTelecomBalance(ctx, gateway.GatewayID, gateway.TenantID); err != nil {
			// Log error but continue with other gateways
			fmt.Printf("Error checking balance for gateway %d: %v\n", gateway.GatewayID, err)
			continue
		}
	}

	return nil
}

// StartPeriodicBalanceCheck starts checking balances periodically (hourly by default)
// This is the Cron Job implementation for periodic balance checking
func (bm *BalanceMonitor) StartPeriodicBalanceCheck(ctx context.Context) {
	ticker := time.NewTicker(bm.checkInterval)
	defer ticker.Stop()

	fmt.Printf("Starting periodic balance check with interval: %v\n", bm.checkInterval)

	// Check immediately on start
	if err := bm.CheckAllGateways(ctx); err != nil {
		fmt.Printf("Error checking balances: %v\n", err)
	}

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Stopping periodic balance checking")
			return
		case <-ticker.C:
			if err := bm.CheckAllGateways(ctx); err != nil {
				fmt.Printf("Error checking balances: %v\n", err)
			}
		}
	}
}

// updateBalance updates the telecom balance and last check time in the database
// Implements Requirement 8.13: Update telecom_balance and last_balance_check
func (bm *BalanceMonitor) updateBalance(ctx context.Context, gatewayID int, tenantID uuid.UUID, balance float64) error {
	query := `
		UPDATE dedicated_gateways
		SET telecom_balance = $1,
		    last_balance_check = $2,
		    updated_at = $2
		WHERE id = $3
		  AND tenant_id = $4
	`

	err := db.RunInTenantTx(ctx, tenantID, false, bm.db, func(tx *sql.Tx) error {
		now := time.Now()
		result, err := tx.ExecContext(ctx, query, balance, now, gatewayID, tenantID)
		if err != nil {
			return err
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return err
		}

		if rowsAffected == 0 {
			return fmt.Errorf("no gateway found with id %d for tenant %s", gatewayID, tenantID)
		}
		
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to update balance: %w", err)
	}

	return nil
}

// sendLowBalanceAlert sends an alert when balance drops below threshold
// Implements Requirement 8.14: Send alert when balance drops below balance_alert_threshold
// Implements Requirement 17.8: Send alert to enterprise company when gateway approaches resource limit
func (bm *BalanceMonitor) sendLowBalanceAlert(ctx context.Context, gatewayID int, tenantID uuid.UUID, gatewayName string, currentBalance, threshold float64) error {
	alert := &Alert{
		ID:          uuid.New(),
		GatewayID:   gatewayID,
		TenantID:    tenantID,
		AlertType:   "TELECOM_BALANCE",
		Severity:    string(SeverityWarning),
		Message:     fmt.Sprintf("Telecom balance low for gateway '%s': %.2f (threshold: %.2f). Please top up your balance with the telecom provider.", gatewayName, currentBalance, threshold),
		MetricValue: currentBalance,
		Threshold:   threshold,
		Timestamp:   time.Now(),
		Resolved:    false,
	}

	// Send alert through alert manager
	if err := bm.alertManager.SendAlert(ctx, alert); err != nil {
		return fmt.Errorf("failed to send alert: %w", err)
	}

	return nil
}

// getGatewayInfo retrieves gateway information for balance checking
func (bm *BalanceMonitor) getGatewayInfo(ctx context.Context, gatewayID int, tenantID uuid.UUID) (*GatewayBalanceInfo, error) {
	query := `
		SELECT 
			id,
			tenant_id,
			gateway_name,
			telecom_provider,
			COALESCE(telecom_balance, 0) as telecom_balance,
			COALESCE(balance_alert_threshold, 0) as balance_alert_threshold,
			COALESCE(last_balance_check, '1970-01-01'::timestamp) as last_balance_check,
			COALESCE(balance_check_enabled, true) as balance_check_enabled
		FROM dedicated_gateways
		WHERE id = $1
		  AND tenant_id = $2
		  AND status = 'active'
	`

	var info GatewayBalanceInfo
	err := db.RunInTenantTx(ctx, tenantID, false, bm.db, func(tx *sql.Tx) error {
		return tx.QueryRowContext(ctx, query, gatewayID, tenantID).Scan(
			&info.GatewayID,
			&info.TenantID,
			&info.GatewayName,
			&info.TelecomProvider,
			&info.CurrentBalance,
			&info.BalanceAlertThreshold,
			&info.LastBalanceCheck,
			&info.BalanceCheckEnabled,
		)
	})

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("gateway not found or not active")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query gateway info: %w", err)
	}

	return &info, nil
}

// getActiveGatewaysForBalanceCheck retrieves all active gateways with balance monitoring enabled
func (bm *BalanceMonitor) getActiveGatewaysForBalanceCheck(ctx context.Context) ([]GatewayBalanceInfo, error) {
	query := `
		SELECT 
			id,
			tenant_id,
			gateway_name,
			telecom_provider,
			COALESCE(telecom_balance, 0) as telecom_balance,
			COALESCE(balance_alert_threshold, 0) as balance_alert_threshold,
			COALESCE(last_balance_check, '1970-01-01'::timestamp) as last_balance_check,
			COALESCE(balance_check_enabled, true) as balance_check_enabled
		FROM dedicated_gateways
		WHERE status = 'active'
		  AND balance_check_enabled = true
		ORDER BY last_balance_check ASC
	`

	rows, err := bm.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query active gateways: %w", err)
	}
	defer rows.Close()

	var gateways []GatewayBalanceInfo
	for rows.Next() {
		var info GatewayBalanceInfo
		if err := rows.Scan(
			&info.GatewayID,
			&info.TenantID,
			&info.GatewayName,
			&info.TelecomProvider,
			&info.CurrentBalance,
			&info.BalanceAlertThreshold,
			&info.LastBalanceCheck,
			&info.BalanceCheckEnabled,
		); err != nil {
			return nil, fmt.Errorf("failed to scan gateway info: %w", err)
		}
		gateways = append(gateways, info)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return gateways, nil
}

// getDecryptedCredentials retrieves and decrypts telecom credentials
// Note: This is a placeholder - actual decryption should use AES-256
func (bm *BalanceMonitor) getDecryptedCredentials(ctx context.Context, gatewayID int, tenantID uuid.UUID) (TelecomCredentials, error) {
	query := `
		SELECT 
			telecom_provider,
			telecom_api_endpoint,
			COALESCE(telecom_username, '') as telecom_username,
			COALESCE(telecom_password_encrypted, '') as telecom_password_encrypted,
			COALESCE(telecom_api_key_encrypted, '') as telecom_api_key_encrypted
		FROM dedicated_gateways
		WHERE id = $1
		  AND tenant_id = $2
	`

	var creds TelecomCredentials
	var passwordEncrypted, apiKeyEncrypted string

	err := db.RunInTenantTx(ctx, tenantID, false, bm.db, func(tx *sql.Tx) error {
		return tx.QueryRowContext(ctx, query, gatewayID, tenantID).Scan(
			&creds.Provider,
			&creds.APIEndpoint,
			&creds.Username,
			&passwordEncrypted,
			&apiKeyEncrypted,
		)
	})

	if err != nil {
		return TelecomCredentials{}, fmt.Errorf("failed to query credentials: %w", err)
	}

	// TODO: Implement actual AES-256 decryption
	// For now, we'll use the encrypted values as-is (placeholder)
	creds.Password = passwordEncrypted
	creds.APIKey = apiKeyEncrypted

	return creds, nil
}

// GetBalanceHistory retrieves balance check history for a gateway
func (bm *BalanceMonitor) GetBalanceHistory(ctx context.Context, gatewayID int, tenantID uuid.UUID, limit int) ([]struct {
	Balance   float64
	CheckedAt time.Time
}, error) {
	// This would query a balance_history table if we had one
	// For now, we'll return the current balance only
	query := `
		SELECT 
			COALESCE(telecom_balance, 0) as balance,
			COALESCE(last_balance_check, CURRENT_TIMESTAMP) as checked_at
		FROM dedicated_gateways
		WHERE id = $1
		  AND tenant_id = $2
		LIMIT 1
	`

	var history []struct {
		Balance   float64
		CheckedAt time.Time
	}

	err := db.RunInTenantTx(ctx, tenantID, false, bm.db, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, query, gatewayID, tenantID)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var record struct {
				Balance   float64
				CheckedAt time.Time
			}
			if err := rows.Scan(&record.Balance, &record.CheckedAt); err != nil {
				return err
			}
			history = append(history, record)
		}
		return rows.Err()
	})

	if err != nil {
		return nil, fmt.Errorf("failed to query balance history: %w", err)
	}

	return history, nil
}

// MockTelecomClient is a mock implementation for testing
type MockTelecomClient struct {
	MockBalance float64
	MockError   error
}

// CheckBalance implements TelecomBalanceChecker for testing
func (m *MockTelecomClient) CheckBalance(ctx context.Context, credentials TelecomCredentials) (float64, error) {
	if m.MockError != nil {
		return 0, m.MockError
	}
	return m.MockBalance, nil
}
