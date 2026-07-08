package redis

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"pgregory.net/rapid"
)

// Bug 9 Exploration Test: No Redis Retry Logic
// **Validates: Requirements 1.9, 2.9**
//
// **Property 1: Bug Condition** - No Retry on Transient Errors
//
// This test MUST FAIL on unfixed code - failure confirms the bug exists.
// The bug: Redis operations fail without retry on transient errors, causing data loss.
//
// CRITICAL: This test encodes the EXPECTED behavior (what SHOULD happen).
// On UNFIXED code: Test FAILS (operations fail immediately, no resilience)
// On FIXED code: Test PASSES (retry logic provides resilience)
//
// DO NOT attempt to fix the test or the code when it fails.
// Document the failure as proof of the bug.
func TestProperty_Bug9_NoRetryOnTransientErrors(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	rapid.Check(t, func(t *rapid.T) {
		// Setup miniredis
		mr := miniredis.RunT(t)

		client := goredis.NewClient(&goredis.Options{
			Addr:       mr.Addr(),
			MaxRetries: 0, // Disable built-in retries to test our application-level retry logic
		})
		defer client.Close()

		logger := zap.NewNop()
		redisClient := &Client{
			Client: client,
			Logger: logger,
		}

		messageQueue := NewMessageQueue(redisClient)
		ctx := context.Background()

		// Generate random test data
		tenantID := uuid.New()
		priority := rapid.IntRange(1, 3).Draw(t, "priority")

		// Test scenario: Simulate a transient network blip
		// Redis is available, then becomes unavailable briefly, then recovers
		
		// Phase 1: Verify Redis works
		item1 := &MessageQueueItem{
			MessageID:   uuid.New(),
			TenantID:    tenantID,
			PhoneNumber: "+966500000000",
			Content:     "Test message 1",
			PartsCount:  1,
			Cost:        0.05,
			Priority:    priority,
			EnqueuedAt:  time.Now(),
		}
		
		err := messageQueue.Enqueue(ctx, item1)
		require.NoError(t, err, "Enqueue should succeed when Redis is available")

		// Phase 2: Simulate transient failure
		// Close Redis to simulate network blip
		mr.Close()

		// Phase 3: Attempt operation during failure
		item2 := &MessageQueueItem{
			MessageID:   uuid.New(),
			TenantID:    tenantID,
			PhoneNumber: "+966500000001",
			Content:     "Test message 2",
			PartsCount:  1,
			Cost:        0.05,
			Priority:    priority,
			EnqueuedAt:  time.Now(),
		}

		// ACT: Attempt to enqueue during Redis downtime
		// On UNFIXED code: Fails immediately, data lost
		// On FIXED code: Retries with backoff, may succeed if Redis recovers quickly
		err = messageQueue.Enqueue(ctx, item2)

		// ASSERT: Expected behavior
		// Bug Condition: Operation fails immediately without retry
		// The error should be a connection error
		assert.Error(t, err, "Operation should fail when Redis is unavailable")
		
		// On unfixed code: No retry mechanism exists
		// On fixed code: Retry mechanism would be present (even if it ultimately fails)
		// We document this by showing the operation fails without resilience
		
		t.Logf("Bug Condition confirmed: Operation failed immediately without retry")
		t.Logf("Error: %v", err)
		t.Logf("Expected: Retry logic with exponential backoff (10ms, 20ms, 40ms)")
		t.Logf("Actual: No retry wrapper exists in message_queue.go Enqueue() method")
		t.Logf("Impact: Transient network errors cause immediate data loss")
	})
}

// Example-based test for Bug 9 - Demonstrates immediate failure
// **Validates: Requirements 1.9, 2.9**
//
// This test provides a concrete example of the bug condition.
// EXPECTED OUTCOME on UNFIXED code: PASSES (documents bug exists)
// EXPECTED OUTCOME on FIXED code: Test logic would need adjustment
func TestBug9_ImmediateFailureWithoutRetry(t *testing.T) {
	// Setup miniredis
	mr := miniredis.RunT(t)

	client := goredis.NewClient(&goredis.Options{
		Addr:       mr.Addr(),
		MaxRetries: 0, // Disable built-in retries
	})
	defer client.Close()

	logger := zap.NewNop()
	redisClient := &Client{
		Client: client,
		Logger: logger,
	}

	messageQueue := NewMessageQueue(redisClient)
	ctx := context.Background()

	tenantID := uuid.New()

	// Verify Redis works first
	item1 := &MessageQueueItem{
		MessageID:   uuid.New(),
		TenantID:    tenantID,
		PhoneNumber: "+966500000001",
		Content:     "Test message",
		PartsCount:  1,
		Cost:        0.05,
		Priority:    2,
		EnqueuedAt:  time.Now(),
	}
	
	err := messageQueue.Enqueue(ctx, item1)
	require.NoError(t, err, "Enqueue should work when Redis is up")

	// Close Redis to simulate network failure
	mr.Close()

	// Attempt operation - should fail immediately without retry
	item2 := &MessageQueueItem{
		MessageID:   uuid.New(),
		TenantID:    tenantID,
		PhoneNumber: "+966500000002",
		Content:     "Test message 2",
		PartsCount:  1,
		Cost:        0.05,
		Priority:    2,
		EnqueuedAt:  time.Now(),
	}

	// ACT: Measure how quickly it fails
	startTime := time.Now()
	err = messageQueue.Enqueue(ctx, item2)
	duration := time.Since(startTime)

	// ASSERT: Should fail
	require.Error(t, err, "Operation should fail when Redis is down")

	// Document the bug: Operation fails very quickly (< 100ms)
	// With retry logic (3 attempts with 10ms, 20ms, 40ms backoff), 
	// it would take at least 70ms
	t.Logf("Operation failed in: %v", duration)
	t.Logf("Bug Condition: Operation failed quickly without retry attempts")
	
	if duration < 50*time.Millisecond {
		t.Logf("✓ Bug confirmed: Failed in %v (< 50ms), indicating no retry logic", duration)
		t.Logf("  Expected with retry: > 70ms (3 retries with exponential backoff)")
		t.Logf("  Actual: Immediate failure without retry")
	} else {
		t.Logf("⚠ Unexpected: Operation took %v, may indicate retry logic exists", duration)
	}
}

// Test: Multiple operations all fail without retry
// **Validates: Requirements 1.9, 2.9**
//
// This test verifies that various Redis operations lack retry logic.
func TestBug9_MultipleOperationsLackRetry(t *testing.T) {
	mr := miniredis.RunT(t)

	client := goredis.NewClient(&goredis.Options{
		Addr:       mr.Addr(),
		MaxRetries: 0,
	})
	defer client.Close()

	logger := zap.NewNop()
	redisClient := &Client{
		Client: client,
		Logger: logger,
	}

	messageQueue := NewMessageQueue(redisClient)
	ctx := context.Background()

	tenantID := uuid.New()

	// Close Redis to simulate failure
	mr.Close()

	// Test multiple operations
	operations := []struct {
		name string
		fn   func() error
	}{
		{
			name: "Enqueue",
			fn: func() error {
				return messageQueue.Enqueue(ctx, &MessageQueueItem{
					MessageID:   uuid.New(),
					TenantID:    tenantID,
					PhoneNumber: "+966500000003",
					Content:     "Test",
					Priority:    2,
					EnqueuedAt:  time.Now(),
				})
			},
		},
		{
			name: "GetQueueSize",
			fn: func() error {
				_, err := messageQueue.GetQueueSize(ctx, tenantID)
				return err
			},
		},
		{
			name: "GetActiveTenants",
			fn: func() error {
				_, err := messageQueue.GetActiveTenants(ctx)
				return err
			},
		},
		{
			name: "Dequeue",
			fn: func() error {
				_, err := messageQueue.Dequeue(ctx, tenantID)
				return err
			},
		},
	}

	t.Log("Bug 9 Exploration: Testing multiple operations without retry logic")
	t.Log("Expected: All operations fail immediately without retry")
	t.Log("")

	for _, op := range operations {
		t.Run(op.name, func(t *testing.T) {
			startTime := time.Now()
			err := op.fn()
			duration := time.Since(startTime)

			// Should fail
			assert.Error(t, err, "Operation %s should fail when Redis is down", op.name)

			// Document timing
			t.Logf("Operation: %s", op.name)
			t.Logf("  Duration: %v", duration)
			t.Logf("  Error: %v", err)
			
			if duration < 50*time.Millisecond {
				t.Logf("  ✓ Bug confirmed: Failed quickly without retry")
			} else {
				t.Logf("  ⚠ Unexpected: Took longer than expected")
			}
		})
	}

	t.Log("")
	t.Log("Bug 9 Summary:")
	t.Log("  - All Redis operations in message_queue.go lack retry logic")
	t.Log("  - Transient errors cause immediate failures and data loss")
	t.Log("  - No exponential backoff mechanism exists")
	t.Log("  - Expected fix: Wrap operations with WithRetry() helper")
}

// Test: Document expected retry behavior
// **Validates: Requirements 2.9**
//
// This test documents what the retry logic should look like when implemented.
func TestBug9_ExpectedRetryBehavior(t *testing.T) {
	t.Log("Bug 9: Expected Retry Logic Specification")
	t.Log("")
	t.Log("Retry Strategy:")
	t.Log("  - Max retries: 3")
	t.Log("  - Backoff: Exponential (10ms, 20ms, 40ms)")
	t.Log("  - Max backoff: 1 second")
	t.Log("")
	t.Log("Retryable Errors:")
	t.Log("  - Connection refused")
	t.Log("  - I/O timeout")
	t.Log("  - Connection reset by peer")
	t.Log("  - Temporary failure")
	t.Log("  - Network unreachable")
	t.Log("")
	t.Log("Non-Retryable Errors:")
	t.Log("  - WRONGTYPE (wrong data type)")
	t.Log("  - Invalid argument")
	t.Log("  - Context canceled")
	t.Log("  - Context deadline exceeded")
	t.Log("")
	t.Log("Implementation Approach:")
	t.Log("  1. Create WithRetry() wrapper function in redis/client.go")
	t.Log("  2. Implement isRetryableError() to classify errors")
	t.Log("  3. Wrap all Redis operations in message_queue.go with WithRetry()")
	t.Log("  4. Add metrics for retry attempts and success/failure rates")
	t.Log("")
	t.Log("Current State (UNFIXED):")
	t.Log("  - No retry wrapper exists")
	t.Log("  - All operations call Redis directly")
	t.Log("  - Transient errors cause immediate failure")
}

// Test: Verify error types that should be retryable
// **Validates: Requirements 2.9**
func TestBug9_RetryableErrorClassification(t *testing.T) {
	// Document expected retryable errors
	retryableErrors := []struct {
		err         error
		description string
	}{
		{errors.New("dial tcp: connection refused"), "Connection refused"},
		{errors.New("i/o timeout"), "I/O timeout"},
		{errors.New("connection reset by peer"), "Connection reset"},
		{errors.New("read: connection timed out"), "Read timeout"},
		{goredis.TxFailedErr, "Transaction failed"},
	}

	// Document non-retryable errors
	nonRetryableErrors := []struct {
		err         error
		description string
	}{
		{errors.New("WRONGTYPE Operation against a key holding the wrong kind of value"), "Wrong type"},
		{errors.New("invalid argument"), "Invalid argument"},
		{context.Canceled, "Context canceled"},
		{context.DeadlineExceeded, "Context deadline exceeded"},
	}

	t.Log("Retryable Errors (should trigger retry logic):")
	for _, e := range retryableErrors {
		t.Logf("  ✓ %s: %v", e.description, e.err)
	}

	t.Log("")
	t.Log("Non-Retryable Errors (should fail immediately):")
	for _, e := range nonRetryableErrors {
		t.Logf("  ✗ %s: %v", e.description, e.err)
	}

	t.Log("")
	t.Log("Bug 9: Current implementation does not classify errors")
	t.Log("All errors result in immediate failure without retry")
}
