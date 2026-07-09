package errors

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"
)

// Example 1: Database Operation with Circuit Breaker
func ExampleDatabaseWithCircuitBreaker() {
	// Create circuit breaker for database
	// Max failures: 5, Timeout: 30 seconds
	dbCircuitBreaker := NewCircuitBreaker(5, 30*time.Second)

	ctx := context.Background()

	// Execute database operation
	err := dbCircuitBreaker.Call(ctx, func() error {
		// Simulate database operation
		// In real code, this would be: db.Pool.Exec(ctx, query, args...)
		return nil
	})

	if err != nil {
		if err == ErrCircuitBreakerOpen {
			fmt.Println("Database circuit breaker is OPEN - service unavailable")
			// Implement fallback strategy:
			// 1. Use backup database
			// 2. Queue operation for later
			// 3. Return cached data
		} else {
			fmt.Printf("Database operation failed: %v\n", err)
		}
	}
}

// Example 2: Redis Operation with Circuit Breaker
func ExampleRedisWithCircuitBreaker() {
	// Create circuit breaker for Redis
	// Max failures: 3, Timeout: 10 seconds
	redisCircuitBreaker := NewCircuitBreaker(3, 10*time.Second)

	ctx := context.Background()

	// Execute Redis operation
	result, err := redisCircuitBreaker.CallWithResult(ctx, func() (interface{}, error) {
		// Simulate Redis operation
		// In real code, this would be: redis.Client.Get(ctx, key).Result()
		return "cached_value", nil
	})

	if err != nil {
		if err == ErrCircuitBreakerOpen {
			fmt.Println("Redis circuit breaker is OPEN - falling back to direct processing")
			// Implement fallback strategy:
			// 1. Query database directly
			// 2. Use in-memory cache
			// 3. Skip caching
		} else {
			fmt.Printf("Redis operation failed: %v\n", err)
		}
	} else {
		fmt.Printf("Got cached value: %v\n", result)
	}
}

// Example 3: WebSocket Operation with Circuit Breaker
func ExampleWebSocketWithCircuitBreaker() {
	// Create circuit breaker for WebSocket
	// Max failures: 5, Timeout: 30 seconds
	wsCircuitBreaker := NewCircuitBreaker(5, 30*time.Second)

	ctx := context.Background()

	// Execute WebSocket send operation
	err := wsCircuitBreaker.Call(ctx, func() error {
		// Simulate WebSocket send
		// In real code, this would be: hub.SendMessage(ctx, message)
		return nil
	})

	if err != nil {
		if err == ErrCircuitBreakerOpen {
			fmt.Println("WebSocket circuit breaker is OPEN - device unavailable")
			// Implement fallback strategy:
			// 1. Queue message for retry
			// 2. Mark message as WAITING
			// 3. Notify user of delay
		} else {
			fmt.Printf("WebSocket send failed: %v\n", err)
		}
	}
}

// Example 4: Monitoring Circuit Breaker State
func ExampleMonitorCircuitBreaker() {
	cb := NewCircuitBreaker(3, 10*time.Second)

	// Check current state
	state := cb.GetState()
	fmt.Printf("Circuit breaker state: %s\n", state)

	// Check if open
	if cb.IsOpen() {
		fmt.Println("Circuit breaker is OPEN - service unavailable")
		failures := cb.GetFailures()
		fmt.Printf("Current failures: %d\n", failures)
	}

	// Check if closed
	if cb.IsClosed() {
		fmt.Println("Circuit breaker is CLOSED - service available")
	}

	// Check if half-open
	if cb.IsHalfOpen() {
		fmt.Println("Circuit breaker is HALF_OPEN - testing service recovery")
	}

	// Reset circuit breaker manually (for admin operations)
	cb.Reset()
	fmt.Println("Circuit breaker has been reset to CLOSED state")
}

// Example 5: Circuit Breaker with Retry Logic
func ExampleCircuitBreakerWithRetry() {
	cb := NewCircuitBreaker(3, 10*time.Second)
	ctx := context.Background()

	maxRetries := 3
	retryDelay := 1 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := cb.Call(ctx, func() error {
			// Simulate operation that might fail
			return fmt.Errorf("temporary error")
		})

		if err == nil {
			fmt.Println("Operation succeeded")
			break
		}

		if err == ErrCircuitBreakerOpen {
			fmt.Println("Circuit breaker is OPEN - stopping retries")
			break
		}

		if attempt < maxRetries {
			fmt.Printf("Attempt %d failed, retrying in %v...\n", attempt, retryDelay)
			time.Sleep(retryDelay)
			retryDelay *= 2 // Exponential backoff
		} else {
			fmt.Printf("All %d attempts failed\n", maxRetries)
		}
	}
}

// Example 6: Circuit Breaker State Transitions
func ExampleCircuitBreakerStateTransitions() {
	cb := NewCircuitBreakerWithConfig(CircuitBreakerConfig{
		MaxFailures: 3,
		Timeout:     5 * time.Second,
		Name:        "transitions",
	})
	ctx := context.Background()

	fmt.Println("Initial state:", cb.GetState()) // CLOSED

	// Simulate 3 failures to open the circuit
	for i := 1; i <= 3; i++ {
		err := cb.Call(ctx, func() error {
			return fmt.Errorf("failure %d", i)
		})
		fmt.Printf("Failure %d - State: %s, Failures: %d\n", i, cb.GetState(), cb.GetFailures())
		if err == ErrCircuitBreakerOpen {
			fmt.Println("Circuit breaker opened!")
			break
		}
	}

	// Try to call while open
	err := cb.Call(ctx, func() error {
		return nil
	})
	if err == ErrCircuitBreakerOpen {
		fmt.Println("Call rejected - circuit breaker is OPEN")
	}

	// Wait for timeout to transition to HALF_OPEN
	fmt.Println("Waiting for timeout...")
	time.Sleep(6 * time.Second)

	// Next call will transition to HALF_OPEN
	err = cb.Call(ctx, func() error {
		return nil // Success
	})
	if err == nil {
		fmt.Println("Call succeeded - circuit breaker transitioned to CLOSED")
		fmt.Println("Final state:", cb.GetState())
	}
}

// Example 7: Multiple Circuit Breakers for Different Services
func ExampleMultipleCircuitBreakers() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	
	// Different thresholds for different services with config
	dbCircuitBreaker := NewCircuitBreakerWithConfig(CircuitBreakerConfig{
		MaxFailures: 5,
		Timeout:     30 * time.Second,
		Logger:      logger,
		Name:        "database",
	}) // More tolerant
	redisCircuitBreaker := NewCircuitBreakerWithConfig(CircuitBreakerConfig{
		MaxFailures: 3,
		Timeout:     10 * time.Second,
		Logger:      logger,
		Name:        "redis",
	}) // Less tolerant
	wsCircuitBreaker := NewCircuitBreakerWithConfig(CircuitBreakerConfig{
		MaxFailures: 5,
		Timeout:     30 * time.Second,
		Logger:      logger,
		Name:        "websocket",
	}) // More tolerant

	ctx := context.Background()

	// Database operation
	dbCircuitBreaker.Call(ctx, func() error {
		// Database operation
		return nil
	})

	// Redis operation
	redisCircuitBreaker.Call(ctx, func() error {
		// Redis operation
		return nil
	})

	// WebSocket operation
	wsCircuitBreaker.Call(ctx, func() error {
		// WebSocket operation
		return nil
	})

	// Monitor all circuit breakers
	fmt.Printf("Database CB: %s (failures: %d)\n", dbCircuitBreaker.GetState(), dbCircuitBreaker.GetFailures())
	fmt.Printf("Redis CB: %s (failures: %d)\n", redisCircuitBreaker.GetState(), redisCircuitBreaker.GetFailures())
	fmt.Printf("WebSocket CB: %s (failures: %d)\n", wsCircuitBreaker.GetState(), wsCircuitBreaker.GetFailures())
}

// Example 8: Circuit Breaker with Metrics
func ExampleCircuitBreakerWithMetrics() {
	cb := NewCircuitBreaker(3, 10*time.Second)
	ctx := context.Background()

	// Metrics counters
	var (
		totalCalls    int
		successCalls  int
		failedCalls   int
		rejectedCalls int
	)

	// Execute multiple operations
	for i := 0; i < 10; i++ {
		totalCalls++

		err := cb.Call(ctx, func() error {
			// Simulate operation with 50% failure rate
			if i%2 == 0 {
				return fmt.Errorf("simulated error")
			}
			return nil
		})

		if err == nil {
			successCalls++
		} else if err == ErrCircuitBreakerOpen {
			rejectedCalls++
		} else {
			failedCalls++
		}
	}

	// Print metrics
	fmt.Printf("Total calls: %d\n", totalCalls)
	fmt.Printf("Successful calls: %d\n", successCalls)
	fmt.Printf("Failed calls: %d\n", failedCalls)
	fmt.Printf("Rejected calls (CB open): %d\n", rejectedCalls)
	fmt.Printf("Success rate: %.2f%%\n", float64(successCalls)/float64(totalCalls)*100)
	fmt.Printf("Final CB state: %s\n", cb.GetState())
}
