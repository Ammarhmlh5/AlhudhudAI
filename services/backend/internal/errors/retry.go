package errors

import (
	"context"
	"fmt"
	"time"
)

// RetryConfig defines retry behavior
type RetryConfig struct {
	MaxRetries      int
	InitialBackoff  time.Duration
	MaxBackoff      time.Duration
	BackoffMultiplier float64
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:        3,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        5 * time.Second,
		BackoffMultiplier: 2.0,
	}
}

// RetryableFunc is a function that can be retried
type RetryableFunc func() error

// Retry executes a function with exponential backoff retry
func Retry(ctx context.Context, config RetryConfig, fn RetryableFunc) error {
	var lastErr error
	backoff := config.InitialBackoff

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		// Execute function
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !IsRetryable(err) {
			return err
		}

		// Check if we've exhausted retries
		if attempt == config.MaxRetries {
			break
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			return fmt.Errorf("retry cancelled: %w", ctx.Err())
		case <-time.After(backoff):
			// Calculate next backoff
			backoff = time.Duration(float64(backoff) * config.BackoffMultiplier)
			if backoff > config.MaxBackoff {
				backoff = config.MaxBackoff
			}
		}
	}

	return Wrap(lastErr, CodeInternalError, CategorySystem, 
		fmt.Sprintf("max retries (%d) exceeded", config.MaxRetries))
}

// RetryWithResult executes a function with retry and returns a result
func RetryWithResult[T any](ctx context.Context, config RetryConfig, fn func() (T, error)) (T, error) {
	var result T
	var lastErr error
	backoff := config.InitialBackoff

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		// Execute function
		res, err := fn()
		if err == nil {
			return res, nil
		}

		lastErr = err

		// Check if error is retryable
		if !IsRetryable(err) {
			return result, err
		}

		// Check if we've exhausted retries
		if attempt == config.MaxRetries {
			break
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			return result, fmt.Errorf("retry cancelled: %w", ctx.Err())
		case <-time.After(backoff):
			// Calculate next backoff
			backoff = time.Duration(float64(backoff) * config.BackoffMultiplier)
			if backoff > config.MaxBackoff {
				backoff = config.MaxBackoff
			}
		}
	}

	return result, Wrap(lastErr, CodeInternalError, CategorySystem,
		fmt.Sprintf("max retries (%d) exceeded", config.MaxRetries))
}

// ShouldRetry checks if an operation should be retried based on the error
func ShouldRetry(err error, attempt int, maxRetries int) bool {
	if attempt >= maxRetries {
		return false
	}
	return IsRetryable(err)
}

// CalculateBackoff calculates exponential backoff duration
func CalculateBackoff(attempt int, config RetryConfig) time.Duration {
	backoff := config.InitialBackoff
	for i := 0; i < attempt; i++ {
		backoff = time.Duration(float64(backoff) * config.BackoffMultiplier)
		if backoff > config.MaxBackoff {
			return config.MaxBackoff
		}
	}
	return backoff
}
