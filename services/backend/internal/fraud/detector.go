package fraud

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// FraudDetector handles fraud detection operations
type FraudDetector struct {
	db *sql.DB
}

// NewFraudDetector creates a new fraud detector
func NewFraudDetector(db *sql.DB) *FraudDetector {
	return &FraudDetector{db: db}
}

// CheckTransaction checks a transaction for fraud
// Requirements: 36.9
func (d *FraudDetector) CheckTransaction(ctx context.Context, transaction *Transaction) (*FraudCheckResult, error) {
	result := &FraudCheckResult{
		IsFraud:        false,
		Severity:       SeverityLow,
		TriggeredRules: []string{},
		Score:          0,
		Alerts:         []FraudAlert{},
	}

	// Get enabled fraud rules
	rules, err := d.getEnabledRules(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get fraud rules: %w", err)
	}

	// Check each rule
	for _, rule := range rules {
		triggered, err := d.checkRule(ctx, transaction, &rule)
		if err != nil {
			// Log error but continue checking other rules
			fmt.Printf("Error checking rule %s: %v\n", rule.Name, err)
			continue
		}

		if triggered {
			result.IsFraud = true
			result.TriggeredRules = append(result.TriggeredRules, rule.Name)
			
			// Update severity to highest triggered rule
			if rule.Severity == SeverityCritical {
				result.Severity = SeverityCritical
			} else if rule.Severity == SeverityHigh && result.Severity != SeverityCritical {
				result.Severity = SeverityHigh
			} else if rule.Severity == SeverityMedium && result.Severity == SeverityLow {
				result.Severity = SeverityMedium
			}

			// Create alert
			alert, err := d.createAlert(ctx, transaction, &rule)
			if err != nil {
				fmt.Printf("Error creating alert: %v\n", err)
			} else {
				result.Alerts = append(result.Alerts, *alert)
			}

			// Calculate fraud score
			result.Score += d.calculateRuleScore(rule.Severity)
		}
	}

	// Cap score at 100
	if result.Score > 100 {
		result.Score = 100
	}

	return result, nil
}

// checkRule checks if a specific rule is triggered
func (d *FraudDetector) checkRule(ctx context.Context, transaction *Transaction, rule *FraudRule) (bool, error) {
	switch rule.RuleType {
	case "multiple_failed_payments":
		return d.checkMultipleFailedPayments(ctx, transaction, rule)
	case "unusual_amount":
		return d.checkUnusualAmount(ctx, transaction, rule)
	case "rapid_subscription_changes":
		return d.checkRapidSubscriptionChanges(ctx, transaction, rule)
	case "geographic_anomaly":
		return d.checkGeographicAnomaly(ctx, transaction, rule)
	case "velocity_check":
		return d.checkVelocity(ctx, transaction, rule)
	default:
		return false, fmt.Errorf("unknown rule type: %s", rule.RuleType)
	}
}

// checkMultipleFailedPayments checks for multiple failed payment attempts
func (d *FraudDetector) checkMultipleFailedPayments(ctx context.Context, transaction *Transaction, rule *FraudRule) (bool, error) {
	query := `
		SELECT COUNT(*)
		FROM accounting_transactions
		WHERE tenant_id = $1 
		AND user_id = $2
		AND transaction_type = 'payment_failed'
		AND created_at > NOW() - INTERVAL '1 hour' * $3
	`

	var count int
	err := d.db.QueryRowContext(ctx, query, transaction.TenantID, transaction.UserID, rule.TimeWindow/60).Scan(&count)
	if err != nil {
		return false, err
	}

	return float64(count) >= rule.Threshold, nil
}

// checkUnusualAmount checks for unusual transaction amounts
func (d *FraudDetector) checkUnusualAmount(ctx context.Context, transaction *Transaction, rule *FraudRule) (bool, error) {
	// Get average transaction amount for user
	query := `
		SELECT AVG(amount)
		FROM accounting_transactions
		WHERE tenant_id = $1 
		AND user_id = $2
		AND created_at > NOW() - INTERVAL '30 days'
	`

	var avgAmount sql.NullFloat64
	err := d.db.QueryRowContext(ctx, query, transaction.TenantID, transaction.UserID).Scan(&avgAmount)
	if err != nil {
		return false, err
	}

	if !avgAmount.Valid || avgAmount.Float64 == 0 {
		return false, nil
	}

	// Check if current amount is more than threshold times the average
	return transaction.Amount > (avgAmount.Float64 * rule.Threshold), nil
}

// checkRapidSubscriptionChanges checks for rapid subscription changes
func (d *FraudDetector) checkRapidSubscriptionChanges(ctx context.Context, transaction *Transaction, rule *FraudRule) (bool, error) {
	query := `
		SELECT COUNT(*)
		FROM subscriptions
		WHERE tenant_id = $1 
		AND user_id = $2
		AND created_at > NOW() - INTERVAL '1 day'
	`

	var count int
	err := d.db.QueryRowContext(ctx, query, transaction.TenantID, transaction.UserID).Scan(&count)
	if err != nil {
		return false, err
	}

	return float64(count) >= rule.Threshold, nil
}

// checkGeographicAnomaly checks for geographic anomalies
func (d *FraudDetector) checkGeographicAnomaly(ctx context.Context, transaction *Transaction, rule *FraudRule) (bool, error) {
	// Get last transaction country
	query := `
		SELECT country
		FROM accounting_transactions
		WHERE tenant_id = $1 
		AND user_id = $2
		AND id < $3
		ORDER BY created_at DESC
		LIMIT 1
	`

	var lastCountry sql.NullString
	err := d.db.QueryRowContext(ctx, query, transaction.TenantID, transaction.UserID, transaction.ID).Scan(&lastCountry)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	// Check if country changed
	if lastCountry.Valid && lastCountry.String != transaction.Country {
		return true, nil
	}

	return false, nil
}

// checkVelocity checks for too many API calls
func (d *FraudDetector) checkVelocity(ctx context.Context, transaction *Transaction, rule *FraudRule) (bool, error) {
	query := `
		SELECT COUNT(*)
		FROM accounting_transactions
		WHERE tenant_id = $1 
		AND user_id = $2
		AND created_at > NOW() - INTERVAL '1 minute' * $3
	`

	var count int
	err := d.db.QueryRowContext(ctx, query, transaction.TenantID, transaction.UserID, rule.TimeWindow).Scan(&count)
	if err != nil {
		return false, err
	}

	return float64(count) >= rule.Threshold, nil
}

// calculateRuleScore calculates the fraud score contribution for a rule
func (d *FraudDetector) calculateRuleScore(severity Severity) float64 {
	switch severity {
	case SeverityCritical:
		return 50
	case SeverityHigh:
		return 30
	case SeverityMedium:
		return 15
	case SeverityLow:
		return 5
	default:
		return 0
	}
}

// createAlert creates a fraud alert
func (d *FraudDetector) createAlert(ctx context.Context, transaction *Transaction, rule *FraudRule) (*FraudAlert, error) {
	metadata, _ := json.Marshal(map[string]interface{}{
		"transaction_id": transaction.ID,
		"amount":         transaction.Amount,
		"currency":       transaction.Currency,
		"ip_address":     transaction.IPAddress,
		"country":        transaction.Country,
	})

	query := `
		INSERT INTO fraud_alerts (
			tenant_id, user_id, rule_id, severity, description,
			metadata, detected_at, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`

	alert := &FraudAlert{
		TenantID:    transaction.TenantID,
		UserID:      transaction.UserID,
		RuleID:      rule.ID,
		Severity:    rule.Severity,
		Description: fmt.Sprintf("Fraud rule triggered: %s", rule.Name),
		Metadata:    string(metadata),
		DetectedAt:  time.Now(),
		Status:      AlertStatusNew,
	}

	err := d.db.QueryRowContext(ctx, query,
		alert.TenantID,
		alert.UserID,
		alert.RuleID,
		alert.Severity,
		alert.Description,
		alert.Metadata,
		alert.DetectedAt,
		alert.Status,
	).Scan(&alert.ID, &alert.CreatedAt, &alert.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create alert: %w", err)
	}

	return alert, nil
}

// getEnabledRules retrieves all enabled fraud rules
func (d *FraudDetector) getEnabledRules(ctx context.Context) ([]FraudRule, error) {
	query := `
		SELECT 
			id, name, description, rule_type, enabled, severity,
			threshold, time_window, created_at, updated_at
		FROM fraud_rules
		WHERE enabled = true
		ORDER BY severity DESC
	`

	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []FraudRule
	for rows.Next() {
		var rule FraudRule
		err := rows.Scan(
			&rule.ID, &rule.Name, &rule.Description, &rule.RuleType,
			&rule.Enabled, &rule.Severity, &rule.Threshold, &rule.TimeWindow,
			&rule.CreatedAt, &rule.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}

	return rules, nil
}

// ResolveAlert resolves a fraud alert
func (d *FraudDetector) ResolveAlert(ctx context.Context, alertID, resolvedBy int64, notes string, falsePositive bool) error {
	status := AlertStatusResolved
	if falsePositive {
		status = AlertStatusFalsePositive
	}

	query := `
		UPDATE fraud_alerts
		SET 
			status = $1,
			resolved_at = $2,
			resolved_by = $3,
			notes = $4,
			updated_at = $5
		WHERE id = $6
	`

	now := time.Now()
	_, err := d.db.ExecContext(ctx, query, status, now, resolvedBy, notes, now, alertID)
	return err
}

// GetAlerts retrieves fraud alerts
func (d *FraudDetector) GetAlerts(ctx context.Context, tenantID int64, status AlertStatus, limit int) ([]FraudAlert, error) {
	query := `
		SELECT 
			id, tenant_id, user_id, rule_id, severity, description,
			metadata, detected_at, status, resolved_at, resolved_by,
			notes, created_at, updated_at
		FROM fraud_alerts
		WHERE tenant_id = $1
	`

	args := []interface{}{tenantID}
	argIndex := 2
	
	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, status)
		argIndex++
	}

	// Use safe parameterized query for LIMIT
	query += fmt.Sprintf(" ORDER BY detected_at DESC LIMIT $%d", argIndex)
	args = append(args, limit)

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []FraudAlert
	for rows.Next() {
		var alert FraudAlert
		err := rows.Scan(
			&alert.ID, &alert.TenantID, &alert.UserID, &alert.RuleID,
			&alert.Severity, &alert.Description, &alert.Metadata,
			&alert.DetectedAt, &alert.Status, &alert.ResolvedAt,
			&alert.ResolvedBy, &alert.Notes, &alert.CreatedAt, &alert.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		alerts = append(alerts, alert)
	}

	return alerts, nil
}
