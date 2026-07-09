package gdpr

import (
	"archive/zip"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// UserDataExport represents a complete export of user data
type UserDataExport struct {
	PersonalInfo  map[string]interface{}   `json:"personal_info"`
	Subscriptions []map[string]interface{} `json:"subscriptions"`
	Transactions  []map[string]interface{} `json:"transactions"`
	Messages      []map[string]interface{} `json:"messages"`
	Consents      []map[string]interface{} `json:"consents"`
	ExportedAt    time.Time                `json:"exported_at"`
	ExportVersion string                   `json:"export_version"`
}

// DataExportManager handles GDPR data export requests
type DataExportManager struct {
	db        *sql.DB
	exportDir string // Directory to store temporary export files
}

// NewDataExportManager creates a new data export manager
func NewDataExportManager(db *sql.DB, exportDir string) *DataExportManager {
	return &DataExportManager{
		db:        db,
		exportDir: exportDir,
	}
}

// ExportUserData exports all user data in JSON format
// Requirements: 36.8
func (m *DataExportManager) ExportUserData(ctx context.Context, tenantID, userID int64) (*UserDataExport, error) {
	export := &UserDataExport{
		ExportedAt:    time.Now(),
		ExportVersion: "1.0",
	}

	// Export personal info
	personalInfo, err := m.exportPersonalInfo(ctx, tenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to export personal info: %w", err)
	}
	export.PersonalInfo = personalInfo

	// Export subscriptions
	subscriptions, err := m.exportSubscriptions(ctx, tenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to export subscriptions: %w", err)
	}
	export.Subscriptions = subscriptions

	// Export transactions
	transactions, err := m.exportTransactions(ctx, tenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to export transactions: %w", err)
	}
	export.Transactions = transactions

	// Export messages
	messages, err := m.exportMessages(ctx, tenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to export messages: %w", err)
	}
	export.Messages = messages

	// Export consents
	consents, err := m.exportConsents(ctx, tenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to export consents: %w", err)
	}
	export.Consents = consents

	return export, nil
}

// exportPersonalInfo exports user personal information
func (m *DataExportManager) exportPersonalInfo(ctx context.Context, tenantID, userID int64) (map[string]interface{}, error) {
	query := `
		SELECT 
			id, email, phone, first_name, last_name,
			created_at, updated_at
		FROM users
		WHERE tenant_id = $1 AND id = $2
	`

	var (
		id                                 int64
		email, phone, firstName, lastName  string
		createdAt, updatedAt               time.Time
	)

	err := m.db.QueryRowContext(ctx, query, tenantID, userID).Scan(
		&id, &email, &phone, &firstName, &lastName,
		&createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":         id,
		"email":      email,
		"phone":      phone,
		"first_name": firstName,
		"last_name":  lastName,
		"created_at": createdAt,
		"updated_at": updatedAt,
	}, nil
}

// exportSubscriptions exports user subscriptions
func (m *DataExportManager) exportSubscriptions(ctx context.Context, tenantID, userID int64) ([]map[string]interface{}, error) {
	query := `
		SELECT 
			id, package_id, status, start_date, end_date,
			messages_included, messages_used, messages_remaining,
			created_at, updated_at
		FROM subscriptions
		WHERE tenant_id = $1 AND user_id = $2
		ORDER BY created_at DESC
	`

	rows, err := m.db.QueryContext(ctx, query, tenantID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subscriptions []map[string]interface{}
	for rows.Next() {
		var (
			id, packageID                                      int64
			status                                             string
			startDate, endDate                                 time.Time
			messagesIncluded, messagesUsed, messagesRemaining  int
			createdAt, updatedAt                               time.Time
		)

		err := rows.Scan(
			&id, &packageID, &status, &startDate, &endDate,
			&messagesIncluded, &messagesUsed, &messagesRemaining,
			&createdAt, &updatedAt,
		)
		if err != nil {
			return nil, err
		}

		subscriptions = append(subscriptions, map[string]interface{}{
			"id":                 id,
			"package_id":         packageID,
			"status":             status,
			"start_date":         startDate,
			"end_date":           endDate,
			"messages_included":  messagesIncluded,
			"messages_used":      messagesUsed,
			"messages_remaining": messagesRemaining,
			"created_at":         createdAt,
			"updated_at":         updatedAt,
		})
	}

	return subscriptions, nil
}

// exportTransactions exports user financial transactions
func (m *DataExportManager) exportTransactions(ctx context.Context, tenantID, userID int64) ([]map[string]interface{}, error) {
	query := `
		SELECT 
			id, transaction_type, amount, currency,
			balance_before, balance_after, description,
			created_at
		FROM accounting_transactions
		WHERE tenant_id = $1 AND user_id = $2
		ORDER BY created_at DESC
	`

	rows, err := m.db.QueryContext(ctx, query, tenantID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []map[string]interface{}
	for rows.Next() {
		var (
			id                                      int64
			transactionType, currency, description  string
			amount, balanceBefore, balanceAfter     float64
			createdAt                               time.Time
		)

		err := rows.Scan(
			&id, &transactionType, &amount, &currency,
			&balanceBefore, &balanceAfter, &description,
			&createdAt,
		)
		if err != nil {
			return nil, err
		}

		transactions = append(transactions, map[string]interface{}{
			"id":              id,
			"transaction_type": transactionType,
			"amount":          amount,
			"currency":        currency,
			"balance_before":  balanceBefore,
			"balance_after":   balanceAfter,
			"description":     description,
			"created_at":      createdAt,
		})
	}

	return transactions, nil
}

// exportMessages exports user messages
func (m *DataExportManager) exportMessages(ctx context.Context, tenantID, userID int64) ([]map[string]interface{}, error) {
	query := `
		SELECT 
			id, recipient, message_text, status,
			sent_at, delivered_at, created_at
		FROM messages
		WHERE tenant_id = $1 AND user_id = $2
		ORDER BY created_at DESC
		LIMIT 1000
	`

	rows, err := m.db.QueryContext(ctx, query, tenantID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []map[string]interface{}
	for rows.Next() {
		var (
			id                                int64
			recipient, messageText, status    string
			sentAt, deliveredAt, createdAt    *time.Time
		)

		err := rows.Scan(
			&id, &recipient, &messageText, &status,
			&sentAt, &deliveredAt, &createdAt,
		)
		if err != nil {
			return nil, err
		}

		msg := map[string]interface{}{
			"id":           id,
			"recipient":    recipient,
			"message_text": messageText,
			"status":       status,
			"created_at":   createdAt,
		}

		if sentAt != nil {
			msg["sent_at"] = sentAt
		}
		if deliveredAt != nil {
			msg["delivered_at"] = deliveredAt
		}

		messages = append(messages, msg)
	}

	return messages, nil
}

// exportConsents exports user consent records
func (m *DataExportManager) exportConsents(ctx context.Context, tenantID, userID int64) ([]map[string]interface{}, error) {
	query := `
		SELECT 
			id, consent_type, version, granted,
			granted_at, revoked_at, created_at
		FROM user_consents
		WHERE tenant_id = $1 AND user_id = $2
		ORDER BY granted_at DESC
	`

	rows, err := m.db.QueryContext(ctx, query, tenantID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var consents []map[string]interface{}
	for rows.Next() {
		var (
			id                          int64
			consentType, version        string
			granted                     bool
			grantedAt, createdAt        time.Time
			revokedAt                   *time.Time
		)

		err := rows.Scan(
			&id, &consentType, &version, &granted,
			&grantedAt, &revokedAt, &createdAt,
		)
		if err != nil {
			return nil, err
		}

		consent := map[string]interface{}{
			"id":           id,
			"consent_type": consentType,
			"version":      version,
			"granted":      granted,
			"granted_at":   grantedAt,
			"created_at":   createdAt,
		}

		if revokedAt != nil {
			consent["revoked_at"] = revokedAt
		}

		consents = append(consents, consent)
	}

	return consents, nil
}

// GenerateExportFile generates a downloadable export file (ZIP with JSON)
// Requirements: 36.8
func (m *DataExportManager) GenerateExportFile(ctx context.Context, tenantID, userID int64) (string, error) {
	// Export data
	export, err := m.ExportUserData(ctx, tenantID, userID)
	if err != nil {
		return "", fmt.Errorf("failed to export data: %w", err)
	}

	// Create export directory if not exists
	if err := os.MkdirAll(m.exportDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create export directory: %w", err)
	}

	// Generate filename
	filename := fmt.Sprintf("user_data_export_%d_%d_%d.zip", tenantID, userID, time.Now().Unix())
	filepath := filepath.Join(m.exportDir, filename)

	// Create ZIP file
	zipFile, err := os.Create(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to create zip file: %w", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Add JSON file to ZIP
	jsonWriter, err := zipWriter.Create("user_data.json")
	if err != nil {
		return "", fmt.Errorf("failed to create json file in zip: %w", err)
	}

	// Write JSON data
	encoder := json.NewEncoder(jsonWriter)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(export); err != nil {
		return "", fmt.Errorf("failed to encode json: %w", err)
	}

	// Add README file
	readmeWriter, err := zipWriter.Create("README.txt")
	if err != nil {
		return "", fmt.Errorf("failed to create readme in zip: %w", err)
	}

	readme := fmt.Sprintf(`User Data Export
================

Tenant ID: %d
User ID: %d
Exported At: %s
Export Version: %s

This archive contains all your personal data in JSON format.

Files:
- user_data.json: Complete data export

Data Included:
- Personal Information
- Subscriptions
- Financial Transactions
- Messages (last 1000)
- Consent Records

This export was generated in compliance with GDPR Article 20 (Right to Data Portability).

For questions, please contact support.
`, tenantID, userID, export.ExportedAt.Format(time.RFC3339), export.ExportVersion)

	if _, err := io.WriteString(readmeWriter, readme); err != nil {
		return "", fmt.Errorf("failed to write readme: %w", err)
	}

	return filepath, nil
}

// CleanupOldExports removes export files older than 7 days
func (m *DataExportManager) CleanupOldExports(ctx context.Context) error {
	files, err := os.ReadDir(m.exportDir)
	if err != nil {
		return fmt.Errorf("failed to read export directory: %w", err)
	}

	cutoff := time.Now().Add(-7 * 24 * time.Hour)

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		info, err := file.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			filepath := filepath.Join(m.exportDir, file.Name())
			if err := os.Remove(filepath); err != nil {
				// Log error but continue
				fmt.Printf("Failed to remove old export file %s: %v\n", filepath, err)
			}
		}
	}

	return nil
}
