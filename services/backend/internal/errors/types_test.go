package errors

import (
	"errors"
	"testing"
)

func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *AppError
		expected string
	}{
		{
			name: "error without underlying error",
			err: &AppError{
				Code:     CodeInvalidRequest,
				Category: CategoryValidation,
				Message:  "Invalid data",
			},
			expected: "[ERR_1001] VALIDATION: Invalid data",
		},
		{
			name: "error with underlying error",
			err: &AppError{
				Code:     CodeDatabaseQuery,
				Category: CategoryDatabase,
				Message:  "Query failed",
				Err:      errors.New("connection timeout"),
			},
			expected: "[ERR_3002] DATABASE: Query failed (caused by: connection timeout)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("Error() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAppError_WithDetails(t *testing.T) {
	err := New(CodeInvalidRequest, CategoryValidation, "Invalid data")
	err.WithDetails("field", "phone_number")
	err.WithDetails("value", "123")

	if len(err.Details) != 2 {
		t.Errorf("Expected 2 details, got %d", len(err.Details))
	}

	if err.Details["field"] != "phone_number" {
		t.Errorf("Expected field=phone_number, got %v", err.Details["field"])
	}

	if err.Details["value"] != "123" {
		t.Errorf("Expected value=123, got %v", err.Details["value"])
	}
}

func TestAppError_Unwrap(t *testing.T) {
	underlying := errors.New("underlying error")
	err := Wrap(underlying, CodeDatabaseQuery, CategoryDatabase, "Query failed")

	unwrapped := err.Unwrap()
	if unwrapped != underlying {
		t.Errorf("Expected unwrapped error to be %v, got %v", underlying, unwrapped)
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "concurrent update is retryable",
			err:      ErrConcurrentUpdate,
			expected: true,
		},
		{
			name:     "deadlock is retryable",
			err:      ErrDeadlock,
			expected: true,
		},
		{
			name:     "connection timeout is retryable",
			err:      ErrConnectionTimeout,
			expected: true,
		},
		{
			name:     "insufficient balance is not retryable",
			err:      ErrInsufficientBalance,
			expected: false,
		},
		{
			name:     "invalid request is not retryable",
			err:      ErrInvalidRequest,
			expected: false,
		},
		{
			name:     "standard error is not retryable",
			err:      errors.New("standard error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRetryable(tt.err); got != tt.expected {
				t.Errorf("IsRetryable() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsCritical(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "database error is critical",
			err:      ErrDatabaseConnection,
			expected: true,
		},
		{
			name:     "system error is critical",
			err:      ErrInternalError,
			expected: true,
		},
		{
			name:     "validation error is not critical",
			err:      ErrInvalidRequest,
			expected: false,
		},
		{
			name:     "business error is not critical",
			err:      ErrInsufficientBalance,
			expected: false,
		},
		{
			name:     "standard error is not critical",
			err:      errors.New("standard error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsCritical(tt.err); got != tt.expected {
				t.Errorf("IsCritical() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNew(t *testing.T) {
	err := New(CodeInvalidRequest, CategoryValidation, "Invalid data")

	if err.Code != CodeInvalidRequest {
		t.Errorf("Expected code %s, got %s", CodeInvalidRequest, err.Code)
	}

	if err.Category != CategoryValidation {
		t.Errorf("Expected category %s, got %s", CategoryValidation, err.Category)
	}

	if err.Message != "Invalid data" {
		t.Errorf("Expected message 'Invalid data', got %s", err.Message)
	}
}

func TestWrap(t *testing.T) {
	underlying := errors.New("underlying error")
	err := Wrap(underlying, CodeDatabaseQuery, CategoryDatabase, "Query failed")

	if err.Code != CodeDatabaseQuery {
		t.Errorf("Expected code %s, got %s", CodeDatabaseQuery, err.Code)
	}

	if err.Err != underlying {
		t.Errorf("Expected underlying error to be %v, got %v", underlying, err.Err)
	}
}

func TestPredefinedErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      *AppError
		code     string
		category string
	}{
		{"ErrInvalidRequest", ErrInvalidRequest, CodeInvalidRequest, CategoryValidation},
		{"ErrInsufficientBalance", ErrInsufficientBalance, CodeInsufficientBalance, CategoryBusiness},
		{"ErrDatabaseConnection", ErrDatabaseConnection, CodeDatabaseConnection, CategoryDatabase},
		{"ErrConnectionClosed", ErrConnectionClosed, CodeConnectionClosed, CategoryNetwork},
		{"ErrInternalError", ErrInternalError, CodeInternalError, CategorySystem},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Code != tt.code {
				t.Errorf("Expected code %s, got %s", tt.code, tt.err.Code)
			}
			if tt.err.Category != tt.category {
				t.Errorf("Expected category %s, got %s", tt.category, tt.err.Category)
			}
		})
	}
}
