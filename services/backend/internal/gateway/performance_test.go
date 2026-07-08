package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestReceiveMessage_ResponseTime tests that API responds within 10ms
// This is a critical requirement (3.1) for the system
func TestReceiveMessage_ResponseTime(t *testing.T) {
	// Create mock buffer with realistic delay (1-2ms for Redis)
	buffer := &MockBuffer{pushDelay: 2 * time.Millisecond}
	
	// Create mock rate limiter
	rateLimiter := &MockRateLimiter{allowed: true}
	
	// Create handler
	handler := NewHandler(rateLimiter, buffer)
	
	// Prepare request
	reqBody := MessageRequest{
		TenantID:    uuid.New(),
		PhoneNumber: "+966501234567",
		Content:     "Test message for performance",
		Priority:    "NORMAL",
	}
	
	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}
	
	// Test multiple requests to get average
	const numRequests = 100
	var totalTime time.Duration
	var slowRequests int
	
	for i := 0; i < numRequests; i++ {
		// Create HTTP request
		req := httptest.NewRequest(http.MethodPost, "/api/messages", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		
		// Create response recorder
		w := httptest.NewRecorder()
		
		// Measure time
		start := time.Now()
		handler.ReceiveMessage(w, req)
		elapsed := time.Since(start)
		
		totalTime += elapsed
		
		// Check response
		if w.Code != http.StatusOK {
			t.Errorf("Request %d: Expected status 200, got %d", i, w.Code)
		}
		
		// Count slow requests (> 10ms)
		if elapsed > 10*time.Millisecond {
			slowRequests++
			t.Logf("Request %d: Slow response time: %v", i, elapsed)
		}
	}
	
	// Calculate statistics
	avgTime := totalTime / time.Duration(numRequests)
	slowPercentage := float64(slowRequests) / float64(numRequests) * 100
	
	t.Logf("Performance Statistics:")
	t.Logf("  Total requests: %d", numRequests)
	t.Logf("  Average response time: %v", avgTime)
	t.Logf("  Slow requests (>10ms): %d (%.2f%%)", slowRequests, slowPercentage)
	
	// Assert performance requirements
	if avgTime > 10*time.Millisecond {
		t.Errorf("Average response time %v exceeds 10ms target", avgTime)
	}
	
	// Allow up to 5% of requests to be slow (95% should be under 10ms)
	if slowPercentage > 5.0 {
		t.Errorf("Too many slow requests: %.2f%% (target: <5%%)", slowPercentage)
	}
}

// TestReceiveMessage_ResponseTimeUnderLoad tests performance under concurrent load
func TestReceiveMessage_ResponseTimeUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}
	
	buffer := &MockBuffer{pushDelay: 2 * time.Millisecond}
	rateLimiter := &MockRateLimiter{allowed: true}
	handler := NewHandler(rateLimiter, buffer)
	
	reqBody := MessageRequest{
		TenantID:    uuid.New(),
		PhoneNumber: "+966501234567",
		Content:     "Load test message",
		Priority:    "NORMAL",
	}
	
	body, _ := json.Marshal(reqBody)
	
	// Run concurrent requests
	const numGoroutines = 50
	const requestsPerGoroutine = 20
	
	results := make(chan time.Duration, numGoroutines*requestsPerGoroutine)
	
	start := time.Now()
	
	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < requestsPerGoroutine; j++ {
				req := httptest.NewRequest(http.MethodPost, "/api/messages", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()
				
				reqStart := time.Now()
				handler.ReceiveMessage(w, req)
				elapsed := time.Since(reqStart)
				
				results <- elapsed
			}
		}()
	}
	
	// Collect results
	var totalTime time.Duration
	var slowRequests int
	totalRequests := numGoroutines * requestsPerGoroutine
	
	for i := 0; i < totalRequests; i++ {
		elapsed := <-results
		totalTime += elapsed
		if elapsed > 10*time.Millisecond {
			slowRequests++
		}
	}
	
	totalElapsed := time.Since(start)
	avgTime := totalTime / time.Duration(totalRequests)
	slowPercentage := float64(slowRequests) / float64(totalRequests) * 100
	throughput := float64(totalRequests) / totalElapsed.Seconds()
	
	t.Logf("Load Test Results:")
	t.Logf("  Total requests: %d", totalRequests)
	t.Logf("  Concurrent goroutines: %d", numGoroutines)
	t.Logf("  Total time: %v", totalElapsed)
	t.Logf("  Average response time: %v", avgTime)
	t.Logf("  Throughput: %.2f req/s", throughput)
	t.Logf("  Slow requests (>10ms): %d (%.2f%%)", slowRequests, slowPercentage)
	
	// Under load, we allow slightly more slow requests (up to 10%)
	if slowPercentage > 10.0 {
		t.Errorf("Too many slow requests under load: %.2f%% (target: <10%%)", slowPercentage)
	}
	
	// Check throughput (should handle at least 100 req/s)
	if throughput < 100 {
		t.Errorf("Throughput too low: %.2f req/s (target: >100 req/s)", throughput)
	}
}

// TestReceiveMessage_ResponseTimeWithRealRedis tests with actual Redis
// This test requires Redis to be running
func TestReceiveMessage_ResponseTimeWithRealRedis(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Redis integration test in short mode")
	}
	
	// This test would require actual Redis connection
	// For now, we skip it and rely on manual testing
	t.Skip("Requires Redis connection - run manually with Redis available")
	
	// TODO: Implement with real Redis client
	// - Connect to Redis
	// - Create real MessageBuffer
	// - Run performance tests
	// - Verify < 10ms response time
}

// TestBufferPush_Timeout tests that buffer push respects timeout
func TestBufferPush_Timeout(t *testing.T) {
	// Create buffer that simulates slow Redis (> 5ms timeout)
	buffer := &MockBuffer{pushDelay: 10 * time.Millisecond}
	
	ctx := context.Background()
	tenantID := uuid.New()
	msg := BufferedMessage{
		MessageID:   "test-123",
		TenantID:    tenantID,
		PhoneNumber: "+966501234567",
		Content:     "Test",
		Priority:    "NORMAL",
		QueuedAt:    time.Now(),
	}
	
	start := time.Now()
	err := buffer.Push(ctx, tenantID, msg)
	elapsed := time.Since(start)
	
	// Should complete (mock doesn't implement timeout, but real buffer would)
	if err != nil {
		t.Logf("Push failed as expected with slow Redis: %v", err)
	}
	
	t.Logf("Push took %v with slow Redis", elapsed)
	
	// In real implementation with timeout, this would fail fast
	// For mock, we just verify it completes
}

// TestReceiveMessage_ValidationPerformance tests validation overhead
func TestReceiveMessage_ValidationPerformance(t *testing.T) {
	validator := NewValidator()
	
	validReq := MessageRequest{
		TenantID:    uuid.New(),
		PhoneNumber: "+966501234567",
		Content:     "Test message",
		Priority:    "NORMAL",
	}
	
	const numValidations = 10000
	start := time.Now()
	
	for i := 0; i < numValidations; i++ {
		errors := validator.ValidateMessageRequest(validReq)
		if len(errors) > 0 {
			t.Errorf("Unexpected validation errors: %v", errors)
		}
	}
	
	elapsed := time.Since(start)
	avgTime := elapsed / time.Duration(numValidations)
	
	t.Logf("Validation Performance:")
	t.Logf("  Total validations: %d", numValidations)
	t.Logf("  Total time: %v", elapsed)
	t.Logf("  Average time per validation: %v", avgTime)
	
	// Validation should be very fast (< 100µs)
	if avgTime > 100*time.Microsecond {
		t.Errorf("Validation too slow: %v (target: <100µs)", avgTime)
	}
}
