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

// MockBuffer is a mock implementation of MessageBuffer for benchmarking
type MockBuffer struct {
	pushDelay time.Duration
}

func (m *MockBuffer) Push(ctx context.Context, tenantID uuid.UUID, msg BufferedMessage) error {
	if m.pushDelay > 0 {
		time.Sleep(m.pushDelay)
	}
	return nil
}

func (m *MockBuffer) GetSize(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	return 0, nil
}

// BenchmarkReceiveMessage_ResponseTime benchmarks the API response time
// Target: < 10ms per request
func BenchmarkReceiveMessage_ResponseTime(b *testing.B) {
	// Create mock buffer with minimal delay (simulating fast Redis)
	buffer := &MockBuffer{pushDelay: 1 * time.Millisecond}
	
	// Create mock rate limiter that always allows
	rateLimiter := &MockRateLimiter{allowed: true}
	
	// Create handler
	handler := NewHandler(rateLimiter, buffer)
	
	// Prepare request
	reqBody := MessageRequest{
		TenantID:    uuid.New(),
		PhoneNumber: "+966501234567",
		Content:     "Test message",
		Priority:    "NORMAL",
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		// Marshal request
		body, _ := json.Marshal(reqBody)
		
		// Create HTTP request
		req := httptest.NewRequest(http.MethodPost, "/api/messages", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		
		// Create response recorder
		w := httptest.NewRecorder()
		
		// Measure time
		start := time.Now()
		
		// Handle request
		handler.ReceiveMessage(w, req)
		
		// Calculate elapsed time
		elapsed := time.Since(start)
		
		// Check response
		if w.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
		
		// Report if response time exceeds 10ms
		if elapsed > 10*time.Millisecond {
			b.Logf("Warning: Response time %v exceeds 10ms target", elapsed)
		}
	}
}

// BenchmarkReceiveMessage_Concurrent benchmarks concurrent requests
func BenchmarkReceiveMessage_Concurrent(b *testing.B) {
	buffer := &MockBuffer{pushDelay: 1 * time.Millisecond}
	rateLimiter := &MockRateLimiter{allowed: true}
	handler := NewHandler(rateLimiter, buffer)
	
	reqBody := MessageRequest{
		TenantID:    uuid.New(),
		PhoneNumber: "+966501234567",
		Content:     "Test message",
		Priority:    "NORMAL",
	}
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest(http.MethodPost, "/api/messages", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			
			handler.ReceiveMessage(w, req)
			
			if w.Code != http.StatusOK {
				b.Fatalf("Expected status 200, got %d", w.Code)
			}
		}
	})
}

// BenchmarkReceiveMessage_WithRealTiming measures actual response time distribution
func BenchmarkReceiveMessage_WithRealTiming(b *testing.B) {
	buffer := &MockBuffer{pushDelay: 1 * time.Millisecond}
	rateLimiter := &MockRateLimiter{allowed: true}
	handler := NewHandler(rateLimiter, buffer)
	
	reqBody := MessageRequest{
		TenantID:    uuid.New(),
		PhoneNumber: "+966501234567",
		Content:     "Test message",
		Priority:    "NORMAL",
	}
	
	var totalTime time.Duration
	var maxTime time.Duration
	var minTime time.Duration = time.Hour
	var under10ms int
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/messages", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		
		start := time.Now()
		handler.ReceiveMessage(w, req)
		elapsed := time.Since(start)
		
		totalTime += elapsed
		if elapsed > maxTime {
			maxTime = elapsed
		}
		if elapsed < minTime {
			minTime = elapsed
		}
		if elapsed < 10*time.Millisecond {
			under10ms++
		}
	}
	
	avgTime := totalTime / time.Duration(b.N)
	percentUnder10ms := float64(under10ms) / float64(b.N) * 100
	
	b.Logf("Average response time: %v", avgTime)
	b.Logf("Min response time: %v", minTime)
	b.Logf("Max response time: %v", maxTime)
	b.Logf("Requests under 10ms: %.2f%%", percentUnder10ms)
	
	if percentUnder10ms < 95.0 {
		b.Logf("Warning: Only %.2f%% of requests completed under 10ms (target: 95%%+)", percentUnder10ms)
	}
}

// MockRateLimiter for testing
type MockRateLimiter struct {
	allowed bool
}

func (m *MockRateLimiter) Allow(ctx context.Context, tenantID uuid.UUID) (bool, *RateLimitInfo, error) {
	return m.allowed, &RateLimitInfo{
		Limit:     100,
		Remaining: 99,
		Reset:     time.Now().Add(time.Minute).Unix(),
	}, nil
}
