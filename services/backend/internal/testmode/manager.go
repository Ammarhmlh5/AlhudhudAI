package testmode

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/al-hudhud/backend/pkg/logger"
	"go.uber.org/zap"
)

var (
	ErrNotInTestMode = errors.New("not in test mode")
	ErrInvalidTestNumber = errors.New("invalid test number")
)

// Manager manages test mode operations
type Manager struct {
	db      *sql.DB
	enabled bool
}

// NewManager creates a new test mode manager
func NewManager(db *sql.DB, enabled bool) *Manager {
	return &Manager{
		db:      db,
		enabled: enabled,
	}
}

// IsTestMode checks if system is in test mode
func (m *Manager) IsTestMode(ctx context.Context) bool {
	// Check if globally enabled
	if m.enabled {
		return true
	}
	
	// Check from context (API key)
	apiKey, ok := ctx.Value("api_key").(string)
	if !ok {
		return false
	}
	
	return IsTestAPIKey(apiKey)
}

// MockSendMessage mocks message sending in test mode
func (m *Manager) MockSendMessage(ctx context.Context, message *Message) (*MessageResult, error) {
	if !m.IsTestMode(ctx) {
		return nil, ErrNotInTestMode
	}
	
	// Validate test phone number
	if !IsTestPhoneNumber(message.To) {
		return nil, ErrInvalidTestNumber
	}
	
	// Get expected result for test number
	status := GetTestResult(message.To)
	
	// Generate mock message ID
	messageID := "test_" + generateID()
	
	result := &MessageResult{
		MessageID: messageID,
		Status:    status,
		Timestamp: time.Now(),
	}
	
	// Add error message for failed statuses
	if status == string(TestStatusFailed) {
		result.Error = "Test failure: simulated error"
	} else if status == string(TestStatusInvalidNumber) {
		result.Error = "Invalid phone number"
	} else if status == string(TestStatusInsufficientBalance) {
		result.Error = "Insufficient balance"
	} else if status == string(TestStatusRateLimited) {
		result.Error = "Rate limit exceeded"
	}
	
	// Log the test operation
	err := m.LogTestOperation(ctx, &TestOperation{
		TenantID:    message.TenantID,
		APIKey:      getAPIKeyFromContext(ctx),
		Operation:   "send_message",
		TestNumber:  message.To,
		Result:      status,
		RequestData: fmt.Sprintf("to=%s, body=%s", message.To, message.Body),
	})
	if err != nil {
		// Log error but don't fail the operation
		logger.GetLogger().Error("Failed to log test operation", zap.Error(err))

	}
	
	return result, nil
}

// LogTestOperation logs a test mode operation
func (m *Manager) LogTestOperation(ctx context.Context, operation *TestOperation) error {
	query := `
		INSERT INTO test_operations (
			tenant_id, api_key, operation, test_number, result, request_data
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`
	
	err := m.db.QueryRowContext(
		ctx, query,
		operation.TenantID, operation.APIKey, operation.Operation,
		operation.TestNumber, operation.Result, operation.RequestData,
	).Scan(&operation.ID, &operation.CreatedAt)
	
	if err != nil {
		return fmt.Errorf("failed to log test operation: %w", err)
	}
	
	return nil
}

// GetTestOperations retrieves test operations for a tenant
func (m *Manager) GetTestOperations(ctx context.Context, tenantID int64, limit, offset int) ([]TestOperation, error) {
	query := `
		SELECT id, tenant_id, api_key, operation, test_number, result, request_data, created_at
		FROM test_operations
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	
	rows, err := m.db.QueryContext(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get test operations: %w", err)
	}
	defer rows.Close()
	
	var operations []TestOperation
	for rows.Next() {
		var op TestOperation
		err := rows.Scan(
			&op.ID, &op.TenantID, &op.APIKey, &op.Operation,
			&op.TestNumber, &op.Result, &op.RequestData, &op.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan test operation: %w", err)
		}
		operations = append(operations, op)
	}
	
	return operations, nil
}

// ClearTestOperations clears test operations for a tenant
func (m *Manager) ClearTestOperations(ctx context.Context, tenantID int64) error {
	query := `DELETE FROM test_operations WHERE tenant_id = $1`
	
	_, err := m.db.ExecContext(ctx, query, tenantID)
	if err != nil {
		return fmt.Errorf("failed to clear test operations: %w", err)
	}
	
	return nil
}

// GetTestStatistics gets statistics for test operations
func (m *Manager) GetTestStatistics(ctx context.Context, tenantID int64) (*TestStatistics, error) {
	query := `
		SELECT 
			COUNT(*) as total_operations,
			COUNT(CASE WHEN result = 'delivered' THEN 1 END) as successful,
			COUNT(CASE WHEN result = 'failed' THEN 1 END) as failed,
			COUNT(CASE WHEN result = 'pending' THEN 1 END) as pending
		FROM test_operations
		WHERE tenant_id = $1
	`
	
	var stats TestStatistics
	err := m.db.QueryRowContext(ctx, query, tenantID).Scan(
		&stats.TotalOperations,
		&stats.Successful,
		&stats.Failed,
		&stats.Pending,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get test statistics: %w", err)
	}
	
	if stats.TotalOperations > 0 {
		stats.SuccessRate = float64(stats.Successful) / float64(stats.TotalOperations) * 100
	}
	
	return &stats, nil
}

// Helper functions

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func getAPIKeyFromContext(ctx context.Context) string {
	apiKey, ok := ctx.Value("api_key").(string)
	if !ok {
		return ""
	}
	return apiKey
}

// TestStatistics represents test mode statistics
type TestStatistics struct {
	TotalOperations int     `json:"total_operations"`
	Successful      int     `json:"successful"`
	Failed          int     `json:"failed"`
	Pending         int     `json:"pending"`
	SuccessRate     float64 `json:"success_rate"`
}
