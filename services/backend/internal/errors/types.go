package errors

import (
	"errors"
	"fmt"
)

// Error Categories
const (
	CategoryValidation = "VALIDATION"
	CategoryDatabase   = "DATABASE"
	CategoryNetwork    = "NETWORK"
	CategoryBusiness   = "BUSINESS"
	CategorySystem     = "SYSTEM"
	CategoryAuth       = "AUTH"
)

// Error Codes
const (
	// Validation Errors (1xxx)
	CodeInvalidRequest      = "ERR_1001"
	CodeInvalidPhoneNumber  = "ERR_1002"
	CodeContentTooLong      = "ERR_1003"
	CodeInvalidTenantID     = "ERR_1004"
	CodeInvalidMessageID    = "ERR_1005"
	CodeInvalidStatus       = "ERR_1006"

	// Business Logic Errors (2xxx)
	CodeInsufficientBalance = "ERR_2001"
	CodeTenantNotFound      = "ERR_2002"
	CodeTenantSuspended     = "ERR_2003"
	CodeMessageNotFound     = "ERR_2004"
	CodeChannelNotFound     = "ERR_2005"
	CodeNoChannelAvailable  = "ERR_2006"
	CodeChannelOffline      = "ERR_2007"
	CodeChannelBusy         = "ERR_2008"
	CodeDuplicateReport     = "ERR_2009"
	CodeRateLimitExceeded   = "ERR_2010"

	// Database Errors (3xxx)
	CodeDatabaseConnection  = "ERR_3001"
	CodeDatabaseQuery       = "ERR_3002"
	CodeDatabaseTransaction = "ERR_3003"
	CodeConcurrentUpdate    = "ERR_3004"
	CodeDeadlock            = "ERR_3005"

	// Network Errors (4xxx)
	CodeConnectionClosed    = "ERR_4001"
	CodeConnectionTimeout   = "ERR_4002"
	CodeSendTimeout         = "ERR_4003"
	CodeBufferFull          = "ERR_4004"

	// System Errors (5xxx)
	CodeInternalError       = "ERR_5001"
	CodeServiceUnavailable  = "ERR_5002"
	CodeCircuitBreakerOpen  = "ERR_5003"
	CodeBackupDatabaseUnavailable = "ERR_5004"
	CodeFallbackFailed      = "ERR_5005"
)

// AppError represents a structured application error
type AppError struct {
	Code       string                 `json:"code"`
	Category   string                 `json:"category"`
	Message    string                 `json:"message"`
	Details    map[string]interface{} `json:"details,omitempty"`
	Err        error                  `json:"-"`
	StackTrace string                 `json:"stack_trace,omitempty"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %s (caused by: %v)", e.Code, e.Category, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s: %s", e.Code, e.Category, e.Message)
}

// Unwrap returns the underlying error
func (e *AppError) Unwrap() error {
	return e.Err
}

// WithDetails adds details to the error
func (e *AppError) WithDetails(key string, value interface{}) *AppError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// WithStackTrace adds stack trace to the error
func (e *AppError) WithStackTrace(trace string) *AppError {
	e.StackTrace = trace
	return e
}

// New creates a new AppError
func New(code, category, message string) *AppError {
	return &AppError{
		Code:     code,
		Category: category,
		Message:  message,
	}
}

// Wrap wraps an existing error with AppError
func Wrap(err error, code, category, message string) *AppError {
	return &AppError{
		Code:     code,
		Category: category,
		Message:  message,
		Err:      err,
	}
}

// Predefined Errors

// Validation Errors
var (
	ErrInvalidRequest = New(
		CodeInvalidRequest,
		CategoryValidation,
		"Invalid request data",
	)

	ErrInvalidPhoneNumber = New(
		CodeInvalidPhoneNumber,
		CategoryValidation,
		"Invalid phone number format",
	)

	ErrContentTooLong = New(
		CodeContentTooLong,
		CategoryValidation,
		"Content exceeds maximum length of 1600 characters",
	)

	ErrInvalidTenantID = New(
		CodeInvalidTenantID,
		CategoryValidation,
		"Invalid tenant ID",
	)

	ErrInvalidMessageID = New(
		CodeInvalidMessageID,
		CategoryValidation,
		"Invalid message ID",
	)

	ErrInvalidStatus = New(
		CodeInvalidStatus,
		CategoryValidation,
		"Invalid status value",
	)
)

// Business Logic Errors
var (
	ErrInsufficientBalance = New(
		CodeInsufficientBalance,
		CategoryBusiness,
		"Insufficient balance",
	)

	ErrTenantNotFound = New(
		CodeTenantNotFound,
		CategoryBusiness,
		"Tenant not found",
	)

	ErrTenantSuspended = New(
		CodeTenantSuspended,
		CategoryBusiness,
		"Tenant account is suspended",
	)

	ErrMessageNotFound = New(
		CodeMessageNotFound,
		CategoryBusiness,
		"Message not found",
	)

	ErrChannelNotFound = New(
		CodeChannelNotFound,
		CategoryBusiness,
		"Channel not found",
	)

	ErrNoChannelAvailable = New(
		CodeNoChannelAvailable,
		CategoryBusiness,
		"No channel available",
	)

	ErrChannelOffline = New(
		CodeChannelOffline,
		CategoryBusiness,
		"Channel is offline",
	)

	ErrChannelBusy = New(
		CodeChannelBusy,
		CategoryBusiness,
		"Channel is busy",
	)

	ErrDuplicateReport = New(
		CodeDuplicateReport,
		CategoryBusiness,
		"Duplicate status report",
	)

	ErrRateLimitExceeded = New(
		CodeRateLimitExceeded,
		CategoryBusiness,
		"Rate limit exceeded",
	)
)

// Database Errors
var (
	ErrDatabaseConnection = New(
		CodeDatabaseConnection,
		CategoryDatabase,
		"Database connection failed",
	)

	ErrDatabaseQuery = New(
		CodeDatabaseQuery,
		CategoryDatabase,
		"Database query failed",
	)

	ErrDatabaseTransaction = New(
		CodeDatabaseTransaction,
		CategoryDatabase,
		"Database transaction failed",
	)

	ErrConcurrentUpdate = New(
		CodeConcurrentUpdate,
		CategoryDatabase,
		"Concurrent update detected",
	)

	ErrDeadlock = New(
		CodeDeadlock,
		CategoryDatabase,
		"Database deadlock detected",
	)
)

// Network Errors
var (
	ErrConnectionClosed = New(
		CodeConnectionClosed,
		CategoryNetwork,
		"Connection closed",
	)

	ErrConnectionTimeout = New(
		CodeConnectionTimeout,
		CategoryNetwork,
		"Connection timeout",
	)

	ErrSendTimeout = New(
		CodeSendTimeout,
		CategoryNetwork,
		"Send operation timeout",
	)

	ErrBufferFull = New(
		CodeBufferFull,
		CategoryNetwork,
		"Buffer is full",
	)
)

// System Errors
var (
	ErrInternalError = New(
		CodeInternalError,
		CategorySystem,
		"Internal server error",
	)

	ErrServiceUnavailable = New(
		CodeServiceUnavailable,
		CategorySystem,
		"Service temporarily unavailable",
	)

	ErrCircuitBreakerOpen = New(
		CodeCircuitBreakerOpen,
		CategorySystem,
		"Circuit breaker is open",
	)

	ErrBackupDatabaseUnavailable = New(
		CodeBackupDatabaseUnavailable,
		CategorySystem,
		"Backup database is unavailable",
	)

	ErrFallbackFailed = New(
		CodeFallbackFailed,
		CategorySystem,
		"Fallback strategy failed",
	)
)

// IsRetryable checks if an error is retryable
func IsRetryable(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		switch appErr.Code {
		case CodeConcurrentUpdate, CodeDeadlock, CodeConnectionTimeout,
			CodeSendTimeout, CodeServiceUnavailable:
			return true
		}
	}
	return false
}

// IsCritical checks if an error is critical
func IsCritical(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		switch appErr.Category {
		case CategoryDatabase, CategorySystem:
			return true
		}
	}
	return false
}
