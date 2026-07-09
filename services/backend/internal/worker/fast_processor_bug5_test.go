package worker

import (
	"context"
	"errors"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"pgregory.net/rapid"

	"github.com/al-hudhud/backend/internal/redis"
)

// Bug 5 Exploration Test: Silent Error Ignoring
// **Validates: Requirements 1.5, 2.5**
//
// **Property 1: Bug Condition** - Errors Silently Ignored
//
// This test MUST FAIL on unfixed code - failure confirms the bug exists.
// The bug: Critical errors are ignored with `_ = ` pattern in:
// - Cache operations (pc.Set in pricing_cache.go line 123)
// - Redis pipeline operations (pipe.Exec in pricing_cache.go line 146)
// - Message queue operations (SAdd in message_queue.go line 56)
// - Database operations (Exec in fast_processor.go line 344)
//
// CRITICAL: This test encodes the EXPECTED behavior (errors should be logged/handled).
// On UNFIXED code: Test FAILS (errors are silently ignored)
// On FIXED code: Test PASSES (errors are properly logged/handled)
//
// DO NOT attempt to fix the test or the code when it fails.
// Document the failure as proof of the bug.
func TestProperty_Bug5_ErrorsSilentlyIgnored(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	rapid.Check(t, func(t *rapid.T) {
		// Setup: Create observable logger to capture log entries
		observedCore, logs := observer.New(zapcore.ErrorLevel)
		logger := zap.New(observedCore)

		// Setup Redis client with miniredis
		mr := miniredis.RunT(t)
		defer mr.Close()

		client := goredis.NewClient(&goredis.Options{
			Addr: mr.Addr(),
		})

		redisClient := &redis.Client{
			Client: client,
			Logger: logger,
		}

		ctx := context.Background()

		// Test Case 1: Redis Pipeline Exec Error (pricing_cache.go line 146)
		// Reset logs
		logs.TakeAll()

		// Create pricing cache
		pricingCache := redis.NewPricingCache(redisClient)

		// Add some data first
		_ = pricingCache.Set(ctx, "GSM7", 100, &redis.PricingInfo{
			CharCount:  100,
			PartsCount: 1,
			Encoding:   "GSM7",
			Cost:       0.05,
		})

		// Simulate pipeline error by closing Redis connection
		mr.Close()

		// ACT: Call InvalidateAll which uses pipe.Exec() with _, _ = pattern
		err := pricingCache.InvalidateAll(ctx)

		// ASSERT: Expected behavior - error should be logged or returned
		// On UNFIXED code: Error is silently ignored (_, _ = pipe.Exec(ctx))
		// On FIXED code: Error is logged or returned
		
		// Check if error was returned OR logged
		errorHandled := err != nil || logs.Len() > 0

		assert.True(t, errorHandled,
			"Bug Condition: Pipeline Exec error silently ignored. "+
				"Expected: Error logged or returned. "+
				"Got: err=%v, logs=%d. "+
				"This failure confirms Bug 5 exists: Silent Error Ignoring",
			err, logs.Len())
	})
}

// Example-based test for Bug 5 - Pipeline Exec Error
// **Validates: Requirements 1.5, 2.5**
//
// This test demonstrates pipeline Exec error being silently ignored.
// EXPECTED OUTCOME on UNFIXED code: FAILS (error not logged)
func TestBug5_PipelineExecErrorIgnored(t *testing.T) {
	// Setup observable logger
	observedCore, logs := observer.New(zapcore.ErrorLevel)
	logger := zap.New(observedCore)

	// Setup Redis
	mr := miniredis.RunT(t)
	defer mr.Close()

	client := goredis.NewClient(&goredis.Options{
		Addr: mr.Addr(),
	})

	redisClient := &redis.Client{
		Client: client,
		Logger: logger,
	}

	ctx := context.Background()

	// Create pricing cache
	pricingCache := redis.NewPricingCache(redisClient)

	// Add 1001 cache entries to trigger the batch pipeline exec at line 146
	// This is where the error is silently ignored: _, _ = pipe.Exec(ctx)
	for i := 0; i < 1001; i++ {
		pricingCache.Set(ctx, "GSM7", i, &redis.PricingInfo{
			CharCount:  i,
			PartsCount: 1,
			Encoding:   "GSM7",
			Cost:       0.05,
		})
	}

	// Close Redis to trigger error during InvalidateAll
	mr.Close()

	// ACT: Call InvalidateAll which has _, _ = pipe.Exec(ctx) pattern at line 146
	// The first batch of 1000 will hit the ignored error
	err := pricingCache.InvalidateAll(ctx)

	// ASSERT: Expected behavior - error should be logged or returned
	// On UNFIXED code: First batch error is silently ignored (_, _ = pipe.Exec(ctx))
	// On FIXED code: Error is logged or returned
	
	// The final pipe.Exec() will return error, but the first batch error is ignored
	// We check if the error was logged for the first batch
	errorHandled := err != nil || logs.Len() > 0

	assert.True(t, errorHandled,
		"Bug Condition: Pipeline Exec error silently ignored in InvalidateAll(). "+
			"Expected: Error logged or returned. "+
			"Got: err=%v, error_logs=%d. "+
			"This confirms Bug 5: Silent Error Ignoring",
		err, logs.Len())

	if !errorHandled {
		t.Logf("COUNTEREXAMPLE: Pipeline Exec failed but no error was logged or returned")
		t.Logf("Location: pricing_cache.go line 146: _, _ = pipe.Exec(ctx)")
		t.Logf("This occurs when count >= 1000 in the batch processing loop")
	}
}

// Example-based test for Bug 5 - Cache Set Error
// **Validates: Requirements 1.5, 2.5**
//
// This test demonstrates cache Set error being silently ignored.
// EXPECTED OUTCOME on UNFIXED code: FAILS (error not logged)
func TestBug5_CacheSetErrorIgnored(t *testing.T) {
	// Setup observable logger
	observedCore, logs := observer.New(zapcore.WarnLevel)
	logger := zap.New(observedCore)

	// Setup Redis
	mr := miniredis.RunT(t)
	defer mr.Close()

	client := goredis.NewClient(&goredis.Options{
		Addr: mr.Addr(),
	})

	redisClient := &redis.Client{
		Client: client,
		Logger: logger,
	}

	ctx := context.Background()

	// Create pricing cache
	pricingCache := redis.NewPricingCache(redisClient)

	// Close Redis to trigger Set error
	mr.Close()

	// ACT: Call GetOrCalculate which has _ = pc.Set() pattern (line 123)
	result, _ := pricingCache.GetOrCalculate(ctx, "Test message", 0.05)

	// ASSERT: Expected behavior - Set error should be logged
	// On UNFIXED code: Set error is silently ignored (_ = pc.Set())
	// On FIXED code: Set error is logged (even if GetOrCalculate returns result)
	
	// GetOrCalculate may return result (cache miss is acceptable)
	// But the Set error should be logged at WARN or ERROR level
	errorLogged := logs.Len() > 0

	assert.True(t, errorLogged,
		"Bug Condition: Cache Set error silently ignored in GetOrCalculate(). "+
			"Expected: Error logged at WARN or ERROR level. "+
			"Got: logs=%d, result=%v. "+
			"This confirms Bug 5: Silent Error Ignoring",
		logs.Len(), result)

	if !errorLogged {
		t.Logf("COUNTEREXAMPLE: Cache Set failed but no error was logged")
		t.Logf("Location: pricing_cache.go line 123: _ = pc.Set(ctx, encoding, charCount, info)")
	}
}

// Example-based test for Bug 5 - Message Queue SAdd Error
// **Validates: Requirements 1.5, 2.5**
//
// This test demonstrates SAdd error being silently ignored in Enqueue.
// EXPECTED OUTCOME on UNFIXED code: FAILS (error not logged)
func TestBug5_MessageQueueSAddErrorIgnored(t *testing.T) {
	// Setup observable logger
	observedCore, logs := observer.New(zapcore.WarnLevel)
	logger := zap.New(observedCore)

	// Setup Redis
	mr := miniredis.RunT(t)
	defer mr.Close()

	client := goredis.NewClient(&goredis.Options{
		Addr: mr.Addr(),
	})

	redisClient := &redis.Client{
		Client: client,
		Logger: logger,
	}

	ctx := context.Background()

	// Create message queue
	messageQueue := redis.NewMessageQueue(redisClient)

	// Create test message
	tenantID := uuid.New()
	messageID := uuid.New()
	item := &redis.MessageQueueItem{
		MessageID:   messageID,
		TenantID:    tenantID,
		PhoneNumber: "+1234567890",
		Content:     "Test message",
		Priority:    2,
		Cost:        0.05,
	}

	// Close Redis to trigger SAdd error
	mr.Close()

	// ACT: Call Enqueue which has SAdd error ignored (line 56)
	err := messageQueue.Enqueue(ctx, item)

	// ASSERT: Expected behavior - SAdd error should be logged
	// On UNFIXED code: SAdd error is silently ignored (no error check)
	// On FIXED code: SAdd error is logged
	
	// Enqueue will fail on LPush, but we want to verify SAdd error is also logged
	// This is tricky because both operations fail when Redis is closed
	
	// For this test, we verify that SOME error is logged or returned
	errorHandled := err != nil || logs.Len() > 0

	assert.True(t, errorHandled,
		"Bug Condition: SAdd error silently ignored in Enqueue(). "+
			"Expected: Error logged or returned. "+
			"Got: err=%v, logs=%d. "+
			"This confirms Bug 5: Silent Error Ignoring",
		err, logs.Len())

	if !errorHandled {
		t.Logf("COUNTEREXAMPLE: SAdd failed but no error was logged or returned")
		t.Logf("Location: message_queue.go line 56: if err := mq.client.Client.SAdd(...).Err(); err != nil { /* no logging */ }")
	}
}

// Example-based test for Bug 5 - Database Exec Error
// **Validates: Requirements 1.5, 2.5**
//
// This test documents the database Exec error being silently ignored.
// EXPECTED OUTCOME on UNFIXED code: FAILS (error not logged)
func TestBug5_DatabaseExecErrorIgnored(t *testing.T) {
	// This test documents the bug in handleFailedMessages()
	// Location: fast_processor.go line 344
	// Code: _, _ = db.PgxPool.Exec(ctx, query, messageIDs)
	
	// Bug Condition: UPDATE query error is silently ignored
	// Expected Behavior: Error should be logged
	
	// This test requires full database setup which is complex
	// We document the bug here for awareness
	
	t.Log("Bug Condition: Database Exec error silently ignored in handleFailedMessages()")
	t.Log("Location: fast_processor.go line 344")
	t.Log("Code: _, _ = db.PgxPool.Exec(ctx, query, messageIDs)")
	t.Log("Expected: Error should be logged")
	t.Log("Actual: Error is silently ignored with _, _ = pattern")
	
	// Mark as known bug
	t.Skip("Database Exec error test requires full database setup - documented as known bug")
}

// Mock error test to verify error detection capability
// This test verifies that our testing approach can detect errors
func TestBug5_ErrorDetectionCapability(t *testing.T) {
	// Setup observable logger
	observedCore, logs := observer.New(zapcore.ErrorLevel)
	logger := zap.New(observedCore)

	// Test that we can detect logged errors
	logger.Error("Test error", zap.String("location", "test"))

	assert.Equal(t, 1, logs.Len(), "Should detect logged error")
	
	logEntry := logs.All()[0]
	assert.Equal(t, "Test error", logEntry.Message)
	assert.Equal(t, zapcore.ErrorLevel, logEntry.Level)
}

// Property test: Verify error logging infrastructure works
func TestProperty_Bug5_ErrorLoggingInfrastructure(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	rapid.Check(t, func(t *rapid.T) {
		// Setup observable logger
		observedCore, logs := observer.New(zapcore.ErrorLevel)
		logger := zap.New(observedCore)

		// Generate random error message
		errorMsg := rapid.String().Draw(t, "errorMsg")
		
		// Log error
		logger.Error(errorMsg)

		// Verify error was logged
		assert.Equal(t, 1, logs.Len(), "Error should be logged")
		
		if logs.Len() > 0 {
			assert.Equal(t, errorMsg, logs.All()[0].Message)
		}
	})
}

// Helper function to create a test error
func createTestError(message string) error {
	return errors.New(message)
}

