package balance

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// OperationType represents the type of balance operation
type OperationType string

const (
	OperationTypeDeduct     OperationType = "DEDUCT"
	OperationTypeRefund     OperationType = "REFUND"
	OperationTypeCharge     OperationType = "CHARGE"
	OperationTypeAdjustment OperationType = "ADJUSTMENT"
)

// BalanceOperation represents a balance operation request
type BalanceOperation struct {
	TenantID  uuid.UUID       `json:"tenant_id"`
	MessageID *uuid.UUID      `json:"message_id,omitempty"`
	GatewayID *uuid.UUID      `json:"gateway_id,omitempty"` // Requirement: Link to dedicated gateway
	Amount    decimal.Decimal `json:"amount"`
	Type      OperationType   `json:"type"`
	Reason    string          `json:"reason"`
	CreatedBy string          `json:"created_by"`
	IPAddress string          `json:"ip_address,omitempty"`
}

// BalanceResult represents the result of a balance operation
type BalanceResult struct {
	Success    bool            `json:"success"`
	OldBalance decimal.Decimal `json:"old_balance"`
	NewBalance decimal.Decimal `json:"new_balance"`
	Error      error           `json:"error,omitempty"`
}

// BalanceHistory represents a balance history record
type BalanceHistory struct {
	ID         uuid.UUID       `json:"id"`
	TenantID   uuid.UUID       `json:"tenant_id"`
	MessageID  *uuid.UUID      `json:"message_id,omitempty"`
	GatewayID  *uuid.UUID      `json:"gateway_id,omitempty"` // Link to dedicated gateway
	OldBalance decimal.Decimal `json:"old_balance"`
	NewBalance decimal.Decimal `json:"new_balance"`
	Amount     decimal.Decimal `json:"amount"`
	Type       OperationType   `json:"type"`
	Reason     string          `json:"reason"`
	CreatedBy  string          `json:"created_by"`
	IPAddress  *string         `json:"ip_address,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
}

// Error types
type ErrorType string

const (
	ErrorInsufficientBalance ErrorType = "INSUFFICIENT_BALANCE"
	ErrorTenantNotFound      ErrorType = "TENANT_NOT_FOUND"
	ErrorConcurrentUpdate    ErrorType = "CONCURRENT_UPDATE"
	ErrorDatabaseError       ErrorType = "DATABASE_ERROR"
)

// BalanceError represents a balance operation error
type BalanceError struct {
	Type    ErrorType
	Message string
	Err     error
}

func (e *BalanceError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *BalanceError) Unwrap() error {
	return e.Err
}

// NewInsufficientBalanceError creates an insufficient balance error
func NewInsufficientBalanceError(tenantID uuid.UUID, required, available decimal.Decimal) *BalanceError {
	return &BalanceError{
		Type:    ErrorInsufficientBalance,
		Message: "insufficient balance",
		Err:     nil,
	}
}

// NewTenantNotFoundError creates a tenant not found error
func NewTenantNotFoundError(tenantID uuid.UUID) *BalanceError {
	return &BalanceError{
		Type:    ErrorTenantNotFound,
		Message: "tenant not found",
		Err:     nil,
	}
}

// NewConcurrentUpdateError creates a concurrent update error
func NewConcurrentUpdateError(err error) *BalanceError {
	return &BalanceError{
		Type:    ErrorConcurrentUpdate,
		Message: "concurrent update detected",
		Err:     err,
	}
}

// NewDatabaseError creates a database error
func NewDatabaseError(err error) *BalanceError {
	return &BalanceError{
		Type:    ErrorDatabaseError,
		Message: "database error",
		Err:     err,
	}
}
