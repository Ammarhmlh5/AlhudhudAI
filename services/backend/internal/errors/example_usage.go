package errors

// This file contains example usage patterns for the error handling package.
// These examples are for documentation purposes only.

import (
	"context"
	"log/slog"
	"time"
)

// Example 1: Creating and using custom errors
func exampleCreateError() error {
	// Create a new error
	err := New(
		CodeInvalidRequest,
		CategoryValidation,
		"Invalid phone number format",
	)

	// Add details
	err.WithDetails("field", "phone_number").
		WithDetails("value", "123")

	return err
}

// Example 2: Wrapping existing errors
func exampleWrapError(dbErr error) error {
	return Wrap(
		dbErr,
		CodeDatabaseQuery,
		CategoryDatabase,
		"Failed to query messages",
	).WithDetails("tenant_id", 123)
}

// Example 3: Using predefined errors
func examplePredefinedErrors(balance, cost float64) error {
	if balance < cost {
		return ErrInsufficientBalance.
			WithDetails("balance", balance).
			WithDetails("cost", cost)
	}
	return nil
}

// Example 4: Logging errors with stack trace
func exampleLogError(ctx context.Context, err error) {
	logger := slog.Default()
	errLogger := NewLogger(logger)

	errLogger.LogError(ctx, err, "Failed to process message",
		"message_id", "123",
		"tenant_id", 456,
	)
}

// Example 5: Using retry mechanism
func exampleRetry(ctx context.Context) error {
	config := DefaultRetryConfig()

	return Retry(ctx, config, func() error {
		// Your operation here
		return nil
	})
}

// Example 6: Using circuit breaker
func exampleCircuitBreaker(ctx context.Context) error {
	cb := NewCircuitBreaker(3, 5*time.Second)

	return cb.Call(ctx, func() error {
		// Your operation here
		return nil
	})
}

// Example 7: Checking error type
func exampleCheckErrorType(err error) {
	if IsRetryable(err) {
		// Retry the operation
	}

	if IsCritical(err) {
		// Send alert
	}
}
