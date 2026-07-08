package errors

import (
	"context"
	"log"
	"sync"
	"time"
)

// FallbackStrategy defines the interface for fallback strategies
type FallbackStrategy interface {
	// Execute attempts the primary operation, falls back if it fails
	Execute(ctx context.Context, primary func() error, fallback func() error) error
	
	// ExecuteWithResult attempts the primary operation with result, falls back if it fails
	ExecuteWithResult(ctx context.Context, primary func() (interface{}, error), fallback func() (interface{}, error)) (interface{}, error)
}

// RedisFallbackStrategy handles Redis failures by falling back to direct processing
type RedisFallbackStrategy struct {
	circuitBreaker *CircuitBreaker
	metrics        *FallbackMetrics
	mu             sync.RWMutex
}

// DatabaseFallbackStrategy handles database failures by falling back to backup database
type DatabaseFallbackStrategy struct {
	circuitBreaker       *CircuitBreaker
	backupCircuitBreaker *CircuitBreaker
	metrics              *FallbackMetrics
	mu                   sync.RWMutex
}

// WebSocketFallbackStrategy handles WebSocket failures by queueing messages
type WebSocketFallbackStrategy struct {
	circuitBreaker *CircuitBreaker
	metrics        *FallbackMetrics
	mu             sync.RWMutex
}

// FallbackMetrics tracks fallback usage
type FallbackMetrics struct {
	PrimaryAttempts   int64
	PrimarySuccesses  int64
	PrimaryFailures   int64
	FallbackAttempts  int64
	FallbackSuccesses int64
	FallbackFailures  int64
	LastFallbackTime  time.Time
	mu                sync.RWMutex
}

// NewRedisFallbackStrategy creates a new Redis fallback strategy
func NewRedisFallbackStrategy(cb *CircuitBreaker) *RedisFallbackStrategy {
	return &RedisFallbackStrategy{
		circuitBreaker: cb,
		metrics:        &FallbackMetrics{},
	}
}

// Execute attempts Redis operation, falls back to direct processing if Redis fails
func (f *RedisFallbackStrategy) Execute(ctx context.Context, primary func() error, fallback func() error) error {
	f.metrics.incrementPrimaryAttempts()

	// Check if circuit breaker is open
	if f.circuitBreaker != nil && f.circuitBreaker.IsOpen() {
		log.Printf("[FALLBACK] Redis circuit breaker is OPEN, using fallback strategy")
		return f.executeFallback(ctx, fallback)
	}

	// Try primary operation
	err := primary()
	if err != nil {
		f.metrics.incrementPrimaryFailures()
		log.Printf("[FALLBACK] Redis primary operation failed: %v, using fallback", err)
		return f.executeFallback(ctx, fallback)
	}

	f.metrics.incrementPrimarySuccesses()
	return nil
}

// ExecuteWithResult attempts Redis operation with result, falls back if it fails
func (f *RedisFallbackStrategy) ExecuteWithResult(ctx context.Context, primary func() (interface{}, error), fallback func() (interface{}, error)) (interface{}, error) {
	f.metrics.incrementPrimaryAttempts()

	// Check if circuit breaker is open
	if f.circuitBreaker != nil && f.circuitBreaker.IsOpen() {
		log.Printf("[FALLBACK] Redis circuit breaker is OPEN, using fallback strategy")
		return f.executeFallbackWithResult(ctx, fallback)
	}

	// Try primary operation
	result, err := primary()
	if err != nil {
		f.metrics.incrementPrimaryFailures()
		log.Printf("[FALLBACK] Redis primary operation failed: %v, using fallback", err)
		return f.executeFallbackWithResult(ctx, fallback)
	}

	f.metrics.incrementPrimarySuccesses()
	return result, nil
}

// executeFallback executes the fallback operation
func (f *RedisFallbackStrategy) executeFallback(ctx context.Context, fallback func() error) error {
	f.metrics.incrementFallbackAttempts()
	f.metrics.updateLastFallbackTime()

	err := fallback()
	if err != nil {
		f.metrics.incrementFallbackFailures()
		return err
	}

	f.metrics.incrementFallbackSuccesses()
	return nil
}

// executeFallbackWithResult executes the fallback operation with result
func (f *RedisFallbackStrategy) executeFallbackWithResult(ctx context.Context, fallback func() (interface{}, error)) (interface{}, error) {
	f.metrics.incrementFallbackAttempts()
	f.metrics.updateLastFallbackTime()

	result, err := fallback()
	if err != nil {
		f.metrics.incrementFallbackFailures()
		return nil, err
	}

	f.metrics.incrementFallbackSuccesses()
	return result, nil
}

// GetMetrics returns the current fallback metrics
func (f *RedisFallbackStrategy) GetMetrics() FallbackMetrics {
	return f.metrics.snapshot()
}

// NewDatabaseFallbackStrategy creates a new database fallback strategy
func NewDatabaseFallbackStrategy(primaryCB *CircuitBreaker, backupCB *CircuitBreaker) *DatabaseFallbackStrategy {
	return &DatabaseFallbackStrategy{
		circuitBreaker:       primaryCB,
		backupCircuitBreaker: backupCB,
		metrics:              &FallbackMetrics{},
	}
}

// Execute attempts primary database, falls back to backup database if primary fails
func (f *DatabaseFallbackStrategy) Execute(ctx context.Context, primary func() error, fallback func() error) error {
	f.metrics.incrementPrimaryAttempts()

	// Check if primary circuit breaker is open
	if f.circuitBreaker != nil && f.circuitBreaker.IsOpen() {
		log.Printf("[FALLBACK] Primary database circuit breaker is OPEN, using backup database")
		return f.executeFallback(ctx, fallback)
	}

	// Try primary database
	err := primary()
	if err != nil {
		f.metrics.incrementPrimaryFailures()
		log.Printf("[FALLBACK] Primary database operation failed: %v, using backup database", err)
		return f.executeFallback(ctx, fallback)
	}

	f.metrics.incrementPrimarySuccesses()
	return nil
}

// ExecuteWithResult attempts primary database with result, falls back if it fails
func (f *DatabaseFallbackStrategy) ExecuteWithResult(ctx context.Context, primary func() (interface{}, error), fallback func() (interface{}, error)) (interface{}, error) {
	f.metrics.incrementPrimaryAttempts()

	// Check if primary circuit breaker is open
	if f.circuitBreaker != nil && f.circuitBreaker.IsOpen() {
		log.Printf("[FALLBACK] Primary database circuit breaker is OPEN, using backup database")
		return f.executeFallbackWithResult(ctx, fallback)
	}

	// Try primary database
	result, err := primary()
	if err != nil {
		f.metrics.incrementPrimaryFailures()
		log.Printf("[FALLBACK] Primary database operation failed: %v, using backup database", err)
		return f.executeFallbackWithResult(ctx, fallback)
	}

	f.metrics.incrementPrimarySuccesses()
	return result, nil
}

// executeFallback executes the fallback operation (backup database)
func (f *DatabaseFallbackStrategy) executeFallback(ctx context.Context, fallback func() error) error {
	f.metrics.incrementFallbackAttempts()
	f.metrics.updateLastFallbackTime()

	// Check if backup circuit breaker is open
	if f.backupCircuitBreaker != nil && f.backupCircuitBreaker.IsOpen() {
		f.metrics.incrementFallbackFailures()
		return ErrBackupDatabaseUnavailable
	}

	err := fallback()
	if err != nil {
		f.metrics.incrementFallbackFailures()
		return err
	}

	f.metrics.incrementFallbackSuccesses()
	return nil
}

// executeFallbackWithResult executes the fallback operation with result
func (f *DatabaseFallbackStrategy) executeFallbackWithResult(ctx context.Context, fallback func() (interface{}, error)) (interface{}, error) {
	f.metrics.incrementFallbackAttempts()
	f.metrics.updateLastFallbackTime()

	// Check if backup circuit breaker is open
	if f.backupCircuitBreaker != nil && f.backupCircuitBreaker.IsOpen() {
		f.metrics.incrementFallbackFailures()
		return nil, ErrBackupDatabaseUnavailable
	}

	result, err := fallback()
	if err != nil {
		f.metrics.incrementFallbackFailures()
		return nil, err
	}

	f.metrics.incrementFallbackSuccesses()
	return result, nil
}

// GetMetrics returns the current fallback metrics
func (f *DatabaseFallbackStrategy) GetMetrics() FallbackMetrics {
	return f.metrics.snapshot()
}

// NewWebSocketFallbackStrategy creates a new WebSocket fallback strategy
func NewWebSocketFallbackStrategy(cb *CircuitBreaker) *WebSocketFallbackStrategy {
	return &WebSocketFallbackStrategy{
		circuitBreaker: cb,
		metrics:        &FallbackMetrics{},
	}
}

// Execute attempts WebSocket send, falls back to queueing if WebSocket fails
func (f *WebSocketFallbackStrategy) Execute(ctx context.Context, primary func() error, fallback func() error) error {
	f.metrics.incrementPrimaryAttempts()

	// Check if circuit breaker is open
	if f.circuitBreaker != nil && f.circuitBreaker.IsOpen() {
		log.Printf("[FALLBACK] WebSocket circuit breaker is OPEN, queueing message for later")
		return f.executeFallback(ctx, fallback)
	}

	// Try primary operation (WebSocket send)
	err := primary()
	if err != nil {
		f.metrics.incrementPrimaryFailures()
		log.Printf("[FALLBACK] WebSocket send failed: %v, queueing message", err)
		return f.executeFallback(ctx, fallback)
	}

	f.metrics.incrementPrimarySuccesses()
	return nil
}

// ExecuteWithResult attempts WebSocket operation with result, falls back if it fails
func (f *WebSocketFallbackStrategy) ExecuteWithResult(ctx context.Context, primary func() (interface{}, error), fallback func() (interface{}, error)) (interface{}, error) {
	f.metrics.incrementPrimaryAttempts()

	// Check if circuit breaker is open
	if f.circuitBreaker != nil && f.circuitBreaker.IsOpen() {
		log.Printf("[FALLBACK] WebSocket circuit breaker is OPEN, using fallback strategy")
		return f.executeFallbackWithResult(ctx, fallback)
	}

	// Try primary operation
	result, err := primary()
	if err != nil {
		f.metrics.incrementPrimaryFailures()
		log.Printf("[FALLBACK] WebSocket primary operation failed: %v, using fallback", err)
		return f.executeFallbackWithResult(ctx, fallback)
	}

	f.metrics.incrementPrimarySuccesses()
	return result, nil
}

// executeFallback executes the fallback operation (queue message)
func (f *WebSocketFallbackStrategy) executeFallback(ctx context.Context, fallback func() error) error {
	f.metrics.incrementFallbackAttempts()
	f.metrics.updateLastFallbackTime()

	err := fallback()
	if err != nil {
		f.metrics.incrementFallbackFailures()
		return err
	}

	f.metrics.incrementFallbackSuccesses()
	return nil
}

// executeFallbackWithResult executes the fallback operation with result
func (f *WebSocketFallbackStrategy) executeFallbackWithResult(ctx context.Context, fallback func() (interface{}, error)) (interface{}, error) {
	f.metrics.incrementFallbackAttempts()
	f.metrics.updateLastFallbackTime()

	result, err := fallback()
	if err != nil {
		f.metrics.incrementFallbackFailures()
		return nil, err
	}

	f.metrics.incrementFallbackSuccesses()
	return result, nil
}

// GetMetrics returns the current fallback metrics
func (f *WebSocketFallbackStrategy) GetMetrics() FallbackMetrics {
	return f.metrics.snapshot()
}

// FallbackMetrics methods

func (m *FallbackMetrics) incrementPrimaryAttempts() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.PrimaryAttempts++
}

func (m *FallbackMetrics) incrementPrimarySuccesses() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.PrimarySuccesses++
}

func (m *FallbackMetrics) incrementPrimaryFailures() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.PrimaryFailures++
}

func (m *FallbackMetrics) incrementFallbackAttempts() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.FallbackAttempts++
}

func (m *FallbackMetrics) incrementFallbackSuccesses() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.FallbackSuccesses++
}

func (m *FallbackMetrics) incrementFallbackFailures() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.FallbackFailures++
}

func (m *FallbackMetrics) updateLastFallbackTime() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.LastFallbackTime = time.Now()
}

func (m *FallbackMetrics) snapshot() FallbackMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return FallbackMetrics{
		PrimaryAttempts:   m.PrimaryAttempts,
		PrimarySuccesses:  m.PrimarySuccesses,
		PrimaryFailures:   m.PrimaryFailures,
		FallbackAttempts:  m.FallbackAttempts,
		FallbackSuccesses: m.FallbackSuccesses,
		FallbackFailures:  m.FallbackFailures,
		LastFallbackTime:  m.LastFallbackTime,
	}
}

// GetPrimarySuccessRate returns the success rate of primary operations (0-100)
func (m *FallbackMetrics) GetPrimarySuccessRate() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.PrimaryAttempts == 0 {
		return 100.0
	}
	return (float64(m.PrimarySuccesses) / float64(m.PrimaryAttempts)) * 100.0
}

// GetFallbackSuccessRate returns the success rate of fallback operations (0-100)
func (m *FallbackMetrics) GetFallbackSuccessRate() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.FallbackAttempts == 0 {
		return 100.0
	}
	return (float64(m.FallbackSuccesses) / float64(m.FallbackAttempts)) * 100.0
}

// GetFallbackUsageRate returns the percentage of operations that used fallback (0-100)
func (m *FallbackMetrics) GetFallbackUsageRate() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	totalAttempts := m.PrimaryAttempts
	if totalAttempts == 0 {
		return 0.0
	}
	return (float64(m.FallbackAttempts) / float64(totalAttempts)) * 100.0
}
