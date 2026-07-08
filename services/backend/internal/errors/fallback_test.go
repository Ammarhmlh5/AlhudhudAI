package errors

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRedisFallbackStrategy_Execute_Success(t *testing.T) {
	cb := NewCircuitBreaker(3, 10*time.Second)
	strategy := NewRedisFallbackStrategy(cb)

	primaryCalled := false
	fallbackCalled := false

	primary := func() error {
		primaryCalled = true
		return nil
	}

	fallback := func() error {
		fallbackCalled = true
		return nil
	}

	err := strategy.Execute(context.Background(), primary, fallback)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !primaryCalled {
		t.Error("Expected primary to be called")
	}

	if fallbackCalled {
		t.Error("Expected fallback not to be called")
	}

	metrics := strategy.GetMetrics()
	if metrics.PrimaryAttempts != 1 {
		t.Errorf("Expected 1 primary attempt, got %d", metrics.PrimaryAttempts)
	}
	if metrics.PrimarySuccesses != 1 {
		t.Errorf("Expected 1 primary success, got %d", metrics.PrimarySuccesses)
	}
	if metrics.FallbackAttempts != 0 {
		t.Errorf("Expected 0 fallback attempts, got %d", metrics.FallbackAttempts)
	}
}

func TestRedisFallbackStrategy_Execute_PrimaryFails(t *testing.T) {
	cb := NewCircuitBreaker(3, 10*time.Second)
	strategy := NewRedisFallbackStrategy(cb)

	primaryCalled := false
	fallbackCalled := false

	primary := func() error {
		primaryCalled = true
		return errors.New("redis connection failed")
	}

	fallback := func() error {
		fallbackCalled = true
		return nil
	}

	err := strategy.Execute(context.Background(), primary, fallback)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !primaryCalled {
		t.Error("Expected primary to be called")
	}

	if !fallbackCalled {
		t.Error("Expected fallback to be called")
	}

	metrics := strategy.GetMetrics()
	if metrics.PrimaryAttempts != 1 {
		t.Errorf("Expected 1 primary attempt, got %d", metrics.PrimaryAttempts)
	}
	if metrics.PrimaryFailures != 1 {
		t.Errorf("Expected 1 primary failure, got %d", metrics.PrimaryFailures)
	}
	if metrics.FallbackAttempts != 1 {
		t.Errorf("Expected 1 fallback attempt, got %d", metrics.FallbackAttempts)
	}
	if metrics.FallbackSuccesses != 1 {
		t.Errorf("Expected 1 fallback success, got %d", metrics.FallbackSuccesses)
	}
}

func TestRedisFallbackStrategy_Execute_CircuitBreakerOpen(t *testing.T) {
	cb := NewCircuitBreaker(3, 10*time.Second)
	strategy := NewRedisFallbackStrategy(cb)

	// Open the circuit breaker by failing 3 times
	for i := 0; i < 3; i++ {
		cb.Call(context.Background(), func() error {
			return errors.New("failure")
		})
	}

	if !cb.IsOpen() {
		t.Fatal("Expected circuit breaker to be open")
	}

	primaryCalled := false
	fallbackCalled := false

	primary := func() error {
		primaryCalled = true
		return nil
	}

	fallback := func() error {
		fallbackCalled = true
		return nil
	}

	err := strategy.Execute(context.Background(), primary, fallback)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if primaryCalled {
		t.Error("Expected primary not to be called when circuit breaker is open")
	}

	if !fallbackCalled {
		t.Error("Expected fallback to be called when circuit breaker is open")
	}

	metrics := strategy.GetMetrics()
	if metrics.FallbackAttempts != 1 {
		t.Errorf("Expected 1 fallback attempt, got %d", metrics.FallbackAttempts)
	}
}

func TestRedisFallbackStrategy_ExecuteWithResult_Success(t *testing.T) {
	cb := NewCircuitBreaker(3, 10*time.Second)
	strategy := NewRedisFallbackStrategy(cb)

	expectedResult := "cached_value"

	primary := func() (interface{}, error) {
		return expectedResult, nil
	}

	fallback := func() (interface{}, error) {
		return "db_value", nil
	}

	result, err := strategy.ExecuteWithResult(context.Background(), primary, fallback)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result != expectedResult {
		t.Errorf("Expected result %v, got %v", expectedResult, result)
	}

	metrics := strategy.GetMetrics()
	if metrics.PrimarySuccesses != 1 {
		t.Errorf("Expected 1 primary success, got %d", metrics.PrimarySuccesses)
	}
}

func TestDatabaseFallbackStrategy_Execute_Success(t *testing.T) {
	primaryCB := NewCircuitBreaker(5, 30*time.Second)
	backupCB := NewCircuitBreaker(5, 30*time.Second)
	strategy := NewDatabaseFallbackStrategy(primaryCB, backupCB)

	primaryCalled := false
	fallbackCalled := false

	primary := func() error {
		primaryCalled = true
		return nil
	}

	fallback := func() error {
		fallbackCalled = true
		return nil
	}

	err := strategy.Execute(context.Background(), primary, fallback)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !primaryCalled {
		t.Error("Expected primary to be called")
	}

	if fallbackCalled {
		t.Error("Expected fallback not to be called")
	}
}

func TestDatabaseFallbackStrategy_Execute_PrimaryFails(t *testing.T) {
	primaryCB := NewCircuitBreaker(5, 30*time.Second)
	backupCB := NewCircuitBreaker(5, 30*time.Second)
	strategy := NewDatabaseFallbackStrategy(primaryCB, backupCB)

	primaryCalled := false
	fallbackCalled := false

	primary := func() error {
		primaryCalled = true
		return errors.New("primary database connection failed")
	}

	fallback := func() error {
		fallbackCalled = true
		return nil
	}

	err := strategy.Execute(context.Background(), primary, fallback)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !primaryCalled {
		t.Error("Expected primary to be called")
	}

	if !fallbackCalled {
		t.Error("Expected fallback to be called")
	}

	metrics := strategy.GetMetrics()
	if metrics.FallbackSuccesses != 1 {
		t.Errorf("Expected 1 fallback success, got %d", metrics.FallbackSuccesses)
	}
}

func TestDatabaseFallbackStrategy_Execute_BothFail(t *testing.T) {
	primaryCB := NewCircuitBreaker(5, 30*time.Second)
	backupCB := NewCircuitBreaker(5, 30*time.Second)
	strategy := NewDatabaseFallbackStrategy(primaryCB, backupCB)

	// Open backup circuit breaker
	for i := 0; i < 5; i++ {
		backupCB.Call(context.Background(), func() error {
			return errors.New("failure")
		})
	}

	primary := func() error {
		return errors.New("primary database failed")
	}

	fallback := func() error {
		return errors.New("backup database failed")
	}

	err := strategy.Execute(context.Background(), primary, fallback)

	if err == nil {
		t.Error("Expected error when both primary and backup fail")
	}

	if err != ErrBackupDatabaseUnavailable {
		t.Errorf("Expected ErrBackupDatabaseUnavailable, got %v", err)
	}
}

func TestWebSocketFallbackStrategy_Execute_Success(t *testing.T) {
	cb := NewCircuitBreaker(5, 30*time.Second)
	strategy := NewWebSocketFallbackStrategy(cb)

	primaryCalled := false
	fallbackCalled := false

	primary := func() error {
		primaryCalled = true
		return nil
	}

	fallback := func() error {
		fallbackCalled = true
		return nil
	}

	err := strategy.Execute(context.Background(), primary, fallback)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !primaryCalled {
		t.Error("Expected primary to be called")
	}

	if fallbackCalled {
		t.Error("Expected fallback not to be called")
	}
}

func TestWebSocketFallbackStrategy_Execute_PrimaryFails(t *testing.T) {
	cb := NewCircuitBreaker(5, 30*time.Second)
	strategy := NewWebSocketFallbackStrategy(cb)

	primaryCalled := false
	fallbackCalled := false

	primary := func() error {
		primaryCalled = true
		return errors.New("websocket connection closed")
	}

	fallback := func() error {
		fallbackCalled = true
		return nil
	}

	err := strategy.Execute(context.Background(), primary, fallback)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !primaryCalled {
		t.Error("Expected primary to be called")
	}

	if !fallbackCalled {
		t.Error("Expected fallback to be called (message should be queued)")
	}

	metrics := strategy.GetMetrics()
	if metrics.FallbackAttempts != 1 {
		t.Errorf("Expected 1 fallback attempt, got %d", metrics.FallbackAttempts)
	}
}

func TestFallbackMetrics_SuccessRates(t *testing.T) {
	cb := NewCircuitBreaker(3, 10*time.Second)
	strategy := NewRedisFallbackStrategy(cb)

	// Execute 10 operations: 7 primary success, 3 primary fail -> fallback success
	for i := 0; i < 7; i++ {
		strategy.Execute(context.Background(), func() error {
			return nil
		}, func() error {
			return nil
		})
	}

	for i := 0; i < 3; i++ {
		strategy.Execute(context.Background(), func() error {
			return errors.New("failure")
		}, func() error {
			return nil
		})
	}

	metrics := strategy.GetMetrics()

	// Primary success rate should be 70%
	primaryRate := metrics.GetPrimarySuccessRate()
	if primaryRate < 69.9 || primaryRate > 70.1 {
		t.Errorf("Expected primary success rate ~70%%, got %.2f%%", primaryRate)
	}

	// Fallback success rate should be 100%
	fallbackRate := metrics.GetFallbackSuccessRate()
	if fallbackRate != 100.0 {
		t.Errorf("Expected fallback success rate 100%%, got %.2f%%", fallbackRate)
	}

	// Fallback usage rate should be 30%
	usageRate := metrics.GetFallbackUsageRate()
	if usageRate < 29.9 || usageRate > 30.1 {
		t.Errorf("Expected fallback usage rate ~30%%, got %.2f%%", usageRate)
	}
}

func TestFallbackMetrics_LastFallbackTime(t *testing.T) {
	cb := NewCircuitBreaker(3, 10*time.Second)
	strategy := NewRedisFallbackStrategy(cb)

	before := time.Now()

	strategy.Execute(context.Background(), func() error {
		return errors.New("failure")
	}, func() error {
		return nil
	})

	after := time.Now()

	metrics := strategy.GetMetrics()

	if metrics.LastFallbackTime.Before(before) || metrics.LastFallbackTime.After(after) {
		t.Error("LastFallbackTime should be between before and after timestamps")
	}
}
