package subscription

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ErrorType represents the type of subscription error
type ErrorType string

const (
	ErrorInsufficientBalance ErrorType = "INSUFFICIENT_BALANCE"
	ErrorPackageNotFound     ErrorType = "PACKAGE_NOT_FOUND"
	ErrorPackageInactive     ErrorType = "PACKAGE_INACTIVE"
	ErrorSubscriptionNotFound ErrorType = "SUBSCRIPTION_NOT_FOUND"
	ErrorInvalidRequest      ErrorType = "INVALID_REQUEST"
	ErrorDatabase            ErrorType = "DATABASE_ERROR"
)

// SubscriptionError represents a subscription-related error
type SubscriptionError struct {
	Type    ErrorType
	Message string
	Details map[string]interface{}
}

func (e *SubscriptionError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Type, e.Message)
}

// NewInsufficientBalanceError creates an insufficient balance error
func NewInsufficientBalanceError(tenantID uuid.UUID, required, available decimal.Decimal) *SubscriptionError {
	return &SubscriptionError{
		Type:    ErrorInsufficientBalance,
		Message: fmt.Sprintf("Insufficient balance: required %s, available %s", required.String(), available.String()),
		Details: map[string]interface{}{
			"tenant_id": tenantID.String(),
			"required":  required.String(),
			"available": available.String(),
		},
	}
}

// NewPackageNotFoundError creates a package not found error
func NewPackageNotFoundError(packageID int64) *SubscriptionError {
	return &SubscriptionError{
		Type:    ErrorPackageNotFound,
		Message: fmt.Sprintf("Package not found: %d", packageID),
		Details: map[string]interface{}{
			"package_id": packageID,
		},
	}
}

// NewPackageInactiveError creates a package inactive error
func NewPackageInactiveError(packageID int64) *SubscriptionError {
	return &SubscriptionError{
		Type:    ErrorPackageInactive,
		Message: fmt.Sprintf("Package is not active: %d", packageID),
		Details: map[string]interface{}{
			"package_id": packageID,
		},
	}
}

// NewSubscriptionNotFoundError creates a subscription not found error
func NewSubscriptionNotFoundError(tenantID uuid.UUID) *SubscriptionError {
	return &SubscriptionError{
		Type:    ErrorSubscriptionNotFound,
		Message: fmt.Sprintf("No active subscription found for tenant: %s", tenantID.String()),
		Details: map[string]interface{}{
			"tenant_id": tenantID.String(),
		},
	}
}

// NewInvalidRequestError creates an invalid request error
func NewInvalidRequestError(message string) *SubscriptionError {
	return &SubscriptionError{
		Type:    ErrorInvalidRequest,
		Message: message,
		Details: map[string]interface{}{},
	}
}

// NewDatabaseError creates a database error
func NewDatabaseError(err error) *SubscriptionError {
	return &SubscriptionError{
		Type:    ErrorDatabase,
		Message: fmt.Sprintf("Database error: %v", err),
		Details: map[string]interface{}{
			"error": err.Error(),
		},
	}
}
