package errors

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker(3, 5*time.Second)

	if cb.maxFailures != 3 {
		t.Errorf("Expected maxFailures=3, got %d", cb.maxFailures)
	}

	if cb.timeout != 5*time.Second {
		t.Errorf("Expected timeout=5s, got %v", cb.timeout)
	}

	if cb.state != StateClosed {
		t.Errorf("Expected initial state=CLOSED, got %s", cb.state)
	}
}

func TestCircuitBreaker_Call_Success(t *testing.T) {
	cb := NewCircuitBreaker(3, 5*time.Second)
	ctx := context.Background()

	// Successful call should keep circuit closed
	err := cb.Call(ctx, func() error {
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if cb.GetState() != StateClosed {
		t.Errorf("Expected state=CLOSED, got %s", cb.GetState())
	}

	if cb.GetFailures() != 0 {
		t.Errorf("Expected failures=0, got %d", cb.GetFailures())
	}
}

func TestCircuitBreaker_Call_Failure(t *testing.T) {
	cb := NewCircuitBreaker(3, 5*time.Second)
	ctx := context.Background()

	testErr := errors.New("test error")

	// First failure
	err := cb.Call(ctx, func() error {
		return testErr
	})

	if err != testErr {
		t.Errorf("Expected error=%v, got %v", testErr, err)
	}

	if cb.GetFailures() != 1 {
		t.Errorf("Expected failures=1, got %d", cb.GetFailures())
	}

	if cb.GetState() != StateClosed {
		t.Errorf("Expected state=CLOSED after 1 failure, got %s", cb.GetState())
	}
}

func TestCircuitBreaker_OpensAfterMaxFailures(t *testing.T) {
	cb := NewCircuitBreaker(3, 5*time.Second)
	ctx := context.Background()

	testErr := errors.New("test error")

	// Trigger 3 failures to open circuit
	for i := 0; i < 3; i++ {
		cb.Call(ctx, func() error {
			return testErr
		})
	}

	if cb.GetState() != StateOpen {
		t.Errorf("Expected state=OPEN after 3 failures, got %s", cb.GetState())
	}

	if !cb.IsOpen() {
		t.Error("Expected IsOpen()=true")
	}

	// Next call should fail immediately with circuit breaker error
	err := cb.Call(ctx, func() error {
		return nil
	})

	if err != ErrCircuitBreakerOpen {
		t.Errorf("Expected ErrCircuitBreakerOpen, got %v", err)
	}
}

func TestCircuitBreaker_HalfOpenAfterTimeout(t *testing.T) {
	cb := NewCircuitBreaker(2, 100*time.Millisecond)
	ctx := context.Background()

	testErr := errors.New("test error")

	// Open the circuit
	for i := 0; i < 2; i++ {
		cb.Call(ctx, func() error {
			return testErr
		})
	}

	if cb.GetState() != StateOpen {
		t.Errorf("Expected state=OPEN, got %s", cb.GetState())
	}

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Next call should transition to half-open
	cb.Call(ctx, func() error {
		return nil
	})

	if cb.GetState() != StateClosed {
		t.Errorf("Expected state=CLOSED after successful half-open call, got %s", cb.GetState())
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := NewCircuitBreaker(2, 5*time.Second)
	ctx := context.Background()

	testErr := errors.New("test error")

	// Open the circuit
	for i := 0; i < 2; i++ {
		cb.Call(ctx, func() error {
			return testErr
		})
	}

	if cb.GetState() != StateOpen {
		t.Errorf("Expected state=OPEN, got %s", cb.GetState())
	}

	// Reset circuit
	cb.Reset()

	if cb.GetState() != StateClosed {
		t.Errorf("Expected state=CLOSED after reset, got %s", cb.GetState())
	}

	if cb.GetFailures() != 0 {
		t.Errorf("Expected failures=0 after reset, got %d", cb.GetFailures())
	}
}

func TestCircuitBreaker_CallWithResult(t *testing.T) {
	cb := NewCircuitBreaker(3, 5*time.Second)
	ctx := context.Background()

	// Successful call with result
	result, err := cb.CallWithResult(ctx, func() (interface{}, error) {
		return "success", nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result != "success" {
		t.Errorf("Expected result='success', got %v", result)
	}

	// Failed call with result
	testErr := errors.New("test error")
	result, err = cb.CallWithResult(ctx, func() (interface{}, error) {
		return nil, testErr
	})

	if err != testErr {
		t.Errorf("Expected error=%v, got %v", testErr, err)
	}

	if result != nil {
		t.Errorf("Expected nil result, got %v", result)
	}
}

func TestCircuitBreaker_StateChecks(t *testing.T) {
	cb := NewCircuitBreaker(2, 5*time.Second)

	// Initially closed
	if !cb.IsClosed() {
		t.Error("Expected IsClosed()=true initially")
	}
	if cb.IsOpen() {
		t.Error("Expected IsOpen()=false initially")
	}
	if cb.IsHalfOpen() {
		t.Error("Expected IsHalfOpen()=false initially")
	}

	// Open the circuit
	ctx := context.Background()
	testErr := errors.New("test error")
	for i := 0; i < 2; i++ {
		cb.Call(ctx, func() error {
			return testErr
		})
	}

	if cb.IsClosed() {
		t.Error("Expected IsClosed()=false after opening")
	}
	if !cb.IsOpen() {
		t.Error("Expected IsOpen()=true after opening")
	}
}
