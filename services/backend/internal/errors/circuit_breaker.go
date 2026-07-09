package errors

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// CircuitState represents the state of a circuit breaker
type CircuitState string

const (
	StateClosed   CircuitState = "CLOSED"
	StateOpen     CircuitState = "OPEN"
	StateHalfOpen CircuitState = "HALF_OPEN"
)

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	maxFailures  int
	timeout      time.Duration
	state        CircuitState
	failures     int
	lastFailTime time.Time
	mu           sync.RWMutex
	logger       *slog.Logger
	name         string
}

// CircuitBreakerConfig holds configuration for circuit breaker
type CircuitBreakerConfig struct {
	MaxFailures int
	Timeout     time.Duration
	Logger      *slog.Logger
	Name        string
}

// NewCircuitBreaker creates a new circuit breaker (backward compatible)
func NewCircuitBreaker(maxFailures int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures: maxFailures,
		timeout:     timeout,
		state:       StateClosed,
	}
}

// NewCircuitBreakerWithConfig creates a new circuit breaker with full config
func NewCircuitBreakerWithConfig(config CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures: config.MaxFailures,
		timeout:     config.Timeout,
		state:       StateClosed,
		logger:      config.Logger,
		name:        config.Name,
	}
}

// Call executes a function with circuit breaker protection
func (cb *CircuitBreaker) Call(ctx context.Context, fn func() error) error {
	// Check if circuit is open
	if !cb.canExecute() {
		return ErrCircuitBreakerOpen
	}

	// Execute function
	err := fn()

	// Update circuit state based on result
	cb.recordResult(err)

	return err
}

// CallWithResult executes a function with circuit breaker and returns result
func (cb *CircuitBreaker) CallWithResult(ctx context.Context, fn func() (interface{}, error)) (interface{}, error) {
	// Check if circuit is open
	if !cb.canExecute() {
		return nil, ErrCircuitBreakerOpen
	}

	// Execute function
	result, err := fn()

	// Update circuit state based on result
	cb.recordResult(err)

	return result, err
}

// canExecute checks if the circuit breaker allows execution
func (cb *CircuitBreaker) canExecute() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		// Check if timeout has elapsed
		if time.Since(cb.lastFailTime) > cb.timeout {
			previousState := cb.state
			cb.state = StateHalfOpen
			
			// Log state change to HALF_OPEN
			if cb.logger != nil {
				cb.logger.Info("circuit_breaker_half_open",
					slog.String("name", cb.name),
					slog.Duration("timeout_elapsed", time.Since(cb.lastFailTime)),
					slog.Duration("configured_timeout", cb.timeout),
				)
			}
			
			// Log state change
			if cb.logger != nil {
				cb.logger.Debug("circuit_breaker_state_changed",
					slog.String("name", cb.name),
					slog.String("from", string(previousState)),
					slog.String("to", string(cb.state)),
				)
			}
			return true
		}
		return false
	case StateHalfOpen:
		return true
	default:
		return false
	}
}

// recordResult updates the circuit breaker state based on execution result
func (cb *CircuitBreaker) recordResult(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	previousState := cb.state

	if err != nil {
		cb.failures++
		cb.lastFailTime = time.Now()

		// Log the failure
		if cb.logger != nil {
			cb.logger.Error("circuit_breaker_failure",
				slog.String("name", cb.name),
				slog.Int("failure_count", cb.failures),
				slog.Int("max_failures", cb.maxFailures),
				slog.String("error", err.Error()),
			)
		}

		// Check if we should open the circuit
		if cb.failures >= cb.maxFailures {
			previousState = cb.state
			cb.state = StateOpen
			
			// Log state change to OPEN
			if cb.logger != nil {
				cb.logger.Warn("circuit_breaker_opened",
					slog.String("name", cb.name),
					slog.Int("failure_count", cb.failures),
					slog.Int("max_failures", cb.maxFailures),
					slog.Duration("timeout", cb.timeout),
				)
			}
		}
	} else {
		// Success - reset failures and close circuit
		wasOpen := cb.state == StateOpen
		cb.failures = 0
		cb.state = StateClosed
		
		// Log the success and state change
		if cb.logger != nil {
			if wasOpen {
				cb.logger.Info("circuit_breaker_closed",
					slog.String("name", cb.name),
					slog.Int("previous_failures", cb.failures),
				)
			}
		}
	}

	// Log state change if it happened
	if previousState != cb.state && cb.logger != nil {
		cb.logger.Debug("circuit_breaker_state_changed",
			slog.String("name", cb.name),
			slog.String("from", string(previousState)),
			slog.String("to", string(cb.state)),
		)
	}
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetFailures returns the current failure count
func (cb *CircuitBreaker) GetFailures() int {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.failures
}

// Reset resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = StateClosed
	cb.failures = 0
}

// IsOpen checks if the circuit breaker is open
func (cb *CircuitBreaker) IsOpen() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state == StateOpen
}

// IsClosed checks if the circuit breaker is closed
func (cb *CircuitBreaker) IsClosed() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state == StateClosed
}

// IsHalfOpen checks if the circuit breaker is half-open
func (cb *CircuitBreaker) IsHalfOpen() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state == StateHalfOpen
}
