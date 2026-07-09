package errors

import (
	"context"
	"fmt"
	"log"
	"time"
)

// Example 1: Redis Fallback - Cache Miss Fallback to Database
func ExampleRedisFallback_CacheMissToDatabase() {
	// Setup
	redisCircuitBreaker := NewCircuitBreaker(3, 10*time.Second)
	fallbackStrategy := NewRedisFallbackStrategy(redisCircuitBreaker)

	// Simulated Redis and Database clients
	var redisClient interface{} // Your Redis client
	var dbClient interface{}    // Your Database client

	tenantID := int64(123)
	cacheKey := fmt.Sprintf("channels:cache:%d", tenantID)

	// Execute with fallback
	result, err := fallbackStrategy.ExecuteWithResult(
		context.Background(),
		// Primary: Try Redis cache
		func() (interface{}, error) {
			log.Printf("[PRIMARY] Attempting to get channels from Redis cache: %s", cacheKey)
			// channels, err := redisClient.Get(ctx, cacheKey)
			// return channels, err
			_ = redisClient
			return nil, fmt.Errorf("redis connection failed") // Simulated failure
		},
		// Fallback: Query database directly
		func() (interface{}, error) {
			log.Printf("[FALLBACK] Redis failed, querying database directly for tenant %d", tenantID)
			// channels, err := dbClient.Query(ctx, "SELECT * FROM channels WHERE tenant_id = $1", tenantID)
			// return channels, err
			_ = dbClient
			return []string{"channel1", "channel2"}, nil // Simulated success
		},
	)

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	log.Printf("Success: Got channels %v", result)

	// Check metrics
	metrics := fallbackStrategy.GetMetrics()
	log.Printf("Metrics: Primary attempts=%d, Fallback attempts=%d, Fallback usage=%.2f%%",
		metrics.PrimaryAttempts, metrics.FallbackAttempts, metrics.GetFallbackUsageRate())
}

// Example 2: Redis Fallback - Buffer Push Fallback to Direct Database Insert
func ExampleRedisFallback_BufferToDirectInsert() {
	// Setup
	redisCircuitBreaker := NewCircuitBreaker(3, 10*time.Second)
	fallbackStrategy := NewRedisFallbackStrategy(redisCircuitBreaker)

	// Simulated clients
	var redisClient interface{}
	var dbClient interface{}

	message := map[string]interface{}{
		"tenant_id":    123,
		"phone_number": "+1234567890",
		"content":      "Hello World",
	}

	// Execute with fallback
	err := fallbackStrategy.Execute(
		context.Background(),
		// Primary: Push to Redis buffer
		func() error {
			log.Printf("[PRIMARY] Attempting to push message to Redis buffer")
			// return redisClient.LPush(ctx, "messages:pending:123", messageJSON)
			_ = redisClient
			return fmt.Errorf("redis unavailable") // Simulated failure
		},
		// Fallback: Insert directly to database
		func() error {
			log.Printf("[FALLBACK] Redis unavailable, inserting message directly to database")
			// return dbClient.Insert(ctx, "messages", message)
			_ = dbClient
			_ = message
			log.Printf("Message inserted directly to database")
			return nil
		},
	)

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	log.Printf("Message queued successfully (via fallback)")
}

// Example 3: Database Fallback - Primary to Backup Database
func ExampleDatabaseFallback_PrimaryToBackup() {
	// Setup
	primaryCircuitBreaker := NewCircuitBreaker(5, 30*time.Second)
	backupCircuitBreaker := NewCircuitBreaker(5, 30*time.Second)
	fallbackStrategy := NewDatabaseFallbackStrategy(primaryCircuitBreaker, backupCircuitBreaker)

	// Simulated database clients
	var primaryDB interface{}
	var backupDB interface{}

	tenantID := int64(123)
	amount := 10.0

	// Execute with fallback
	result, err := fallbackStrategy.ExecuteWithResult(
		context.Background(),
		// Primary: Try primary database
		func() (interface{}, error) {
			log.Printf("[PRIMARY] Attempting to deduct balance from primary database")
			// return primaryDB.Exec(ctx, "UPDATE tenants SET balance = balance - $1 WHERE id = $2", amount, tenantID)
			_ = primaryDB
			return nil, fmt.Errorf("primary database connection timeout") // Simulated failure
		},
		// Fallback: Use backup database
		func() (interface{}, error) {
			log.Printf("[FALLBACK] Primary database failed, using backup database")
			// return backupDB.Exec(ctx, "UPDATE tenants SET balance = balance - $1 WHERE id = $2", amount, tenantID)
			_ = backupDB
			_ = amount
			_ = tenantID
			log.Printf("Balance deducted from backup database")
			return map[string]interface{}{"new_balance": 90.0}, nil
		},
	)

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	log.Printf("Success: %v", result)
}

// Example 4: WebSocket Fallback - Send to Queue on Failure
func ExampleWebSocketFallback_SendToQueue() {
	// Setup
	wsCircuitBreaker := NewCircuitBreaker(5, 30*time.Second)
	fallbackStrategy := NewWebSocketFallbackStrategy(wsCircuitBreaker)

	// Simulated clients
	var wsHub interface{}
	var messageQueue interface{}

	message := map[string]interface{}{
		"message_id": "msg-123",
		"tenant_id":  123,
		"channel_id": 456,
		"content":    "Hello World",
	}

	// Execute with fallback
	err := fallbackStrategy.Execute(
		context.Background(),
		// Primary: Try WebSocket send
		func() error {
			log.Printf("[PRIMARY] Attempting to send message via WebSocket")
			// return wsHub.SendMessage(ctx, message)
			_ = wsHub
			return fmt.Errorf("websocket connection closed") // Simulated failure
		},
		// Fallback: Queue message for later delivery
		func() error {
			log.Printf("[FALLBACK] WebSocket failed, queueing message for later delivery")
			// return messageQueue.Enqueue(ctx, message)
			_ = messageQueue
			_ = message
			log.Printf("Message queued for later delivery")
			
			// Also update message status to WAITING
			log.Printf("Updated message status to WAITING")
			
			return nil
		},
	)

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	log.Printf("Message handling completed (via fallback)")
}

// Example 5: Monitoring Fallback Metrics
func ExampleMonitoringFallbackMetrics() {
	// Setup
	redisCircuitBreaker := NewCircuitBreaker(3, 10*time.Second)
	fallbackStrategy := NewRedisFallbackStrategy(redisCircuitBreaker)

	// Simulate some operations
	for i := 0; i < 100; i++ {
		fallbackStrategy.Execute(
			context.Background(),
			func() error {
				// Simulate 80% success rate
				if i%5 == 0 {
					return fmt.Errorf("failure")
				}
				return nil
			},
			func() error {
				return nil
			},
		)
	}

	// Get metrics
	metrics := fallbackStrategy.GetMetrics()

	log.Printf("=== Fallback Metrics ===")
	log.Printf("Primary Attempts: %d", metrics.PrimaryAttempts)
	log.Printf("Primary Successes: %d", metrics.PrimarySuccesses)
	log.Printf("Primary Failures: %d", metrics.PrimaryFailures)
	log.Printf("Primary Success Rate: %.2f%%", metrics.GetPrimarySuccessRate())
	log.Printf("")
	log.Printf("Fallback Attempts: %d", metrics.FallbackAttempts)
	log.Printf("Fallback Successes: %d", metrics.FallbackSuccesses)
	log.Printf("Fallback Failures: %d", metrics.FallbackFailures)
	log.Printf("Fallback Success Rate: %.2f%%", metrics.GetFallbackSuccessRate())
	log.Printf("")
	log.Printf("Fallback Usage Rate: %.2f%%", metrics.GetFallbackUsageRate())
	log.Printf("Last Fallback Time: %v", metrics.LastFallbackTime)

	// Alert if fallback usage is high
	if metrics.GetFallbackUsageRate() > 50.0 {
		log.Printf("⚠️  WARNING: High fallback usage rate (%.2f%%) - Primary service may be degraded",
			metrics.GetFallbackUsageRate())
	}
}

// Example 6: Integration with Existing Circuit Breaker
func ExampleIntegrationWithCircuitBreaker() {
	// Assume you already have circuit breakers from Task 13.3
	// var dbCircuitBreaker *CircuitBreaker // from database.DB
	// var redisCircuitBreaker *CircuitBreaker // from redis.Client

	// Create fallback strategies using existing circuit breakers
	dbCircuitBreaker := NewCircuitBreaker(5, 30*time.Second)
	backupDBCircuitBreaker := NewCircuitBreaker(5, 30*time.Second)
	
	databaseFallback := NewDatabaseFallbackStrategy(dbCircuitBreaker, backupDBCircuitBreaker)

	redisCircuitBreaker := NewCircuitBreaker(3, 10*time.Second)
	redisFallback := NewRedisFallbackStrategy(redisCircuitBreaker)

	wsCircuitBreaker := NewCircuitBreaker(5, 30*time.Second)
	wsFallback := NewWebSocketFallbackStrategy(wsCircuitBreaker)

	log.Printf("Fallback strategies initialized:")
	log.Printf("- Database Fallback: Ready")
	log.Printf("- Redis Fallback: Ready")
	log.Printf("- WebSocket Fallback: Ready")

	_ = databaseFallback
	_ = redisFallback
	_ = wsFallback
}

// Example 7: Handling Both Circuit Breaker Open and Operation Failure
func ExampleHandlingCircuitBreakerAndFailure() {
	redisCircuitBreaker := NewCircuitBreaker(3, 10*time.Second)
	fallbackStrategy := NewRedisFallbackStrategy(redisCircuitBreaker)

	// Simulate 3 failures to open circuit breaker
	for i := 0; i < 3; i++ {
		redisCircuitBreaker.Call(context.Background(), func() error {
			return fmt.Errorf("redis failure")
		})
	}

	log.Printf("Circuit Breaker State: %s", redisCircuitBreaker.GetState())

	// Now try to execute - should use fallback immediately
	err := fallbackStrategy.Execute(
		context.Background(),
		func() error {
			log.Printf("[PRIMARY] This should not be called when circuit breaker is open")
			return nil
		},
		func() error {
			log.Printf("[FALLBACK] Circuit breaker is open, using fallback immediately")
			return nil
		},
	)

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	metrics := fallbackStrategy.GetMetrics()
	log.Printf("Fallback was used immediately due to open circuit breaker")
	log.Printf("Fallback attempts: %d", metrics.FallbackAttempts)
}
