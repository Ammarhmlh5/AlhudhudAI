package metrics

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPMetricsMiddleware(t *testing.T) {
	// Reset metrics
	APIResponseTime.Reset()

	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond) // Simulate processing time
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap with metrics middleware
	wrappedHandler := HTTPMetricsMiddleware(handler)

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	// Execute request
	wrappedHandler.ServeHTTP(rec, req)

	// Verify response
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	// Note: We can't easily verify the histogram was updated without accessing internal state
	// This test ensures the middleware doesn't panic and the handler executes correctly
}

func TestHTTPMetricsMiddleware_ErrorStatus(t *testing.T) {
	// Create a test handler that returns an error
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error"))
	})

	// Wrap with metrics middleware
	wrappedHandler := HTTPMetricsMiddleware(handler)

	// Create test request
	req := httptest.NewRequest("POST", "/api/test", nil)
	rec := httptest.NewRecorder()

	// Execute request
	wrappedHandler.ServeHTTP(rec, req)

	// Verify response
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", rec.Code)
	}
}

func TestResponseWriter_WriteHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: rec,
		statusCode:     http.StatusOK,
		written:        false,
	}

	// Write header
	rw.WriteHeader(http.StatusCreated)

	// Verify status code captured
	if rw.statusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", rw.statusCode)
	}

	// Verify written flag set
	if !rw.written {
		t.Error("Expected written flag to be true")
	}

	// Try to write header again (should be ignored)
	rw.WriteHeader(http.StatusBadRequest)

	// Status should remain unchanged
	if rw.statusCode != http.StatusCreated {
		t.Errorf("Expected status to remain 201, got %d", rw.statusCode)
	}
}

func TestResponseWriter_Write(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: rec,
		statusCode:     http.StatusOK,
		written:        false,
	}

	// Write without calling WriteHeader first
	data := []byte("test data")
	n, err := rw.Write(data)

	// Verify write succeeded
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if n != len(data) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(data), n)
	}

	// Verify status code defaulted to 200
	if rw.statusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rw.statusCode)
	}

	// Verify written flag set
	if !rw.written {
		t.Error("Expected written flag to be true")
	}
}

func TestMetricsRecorder_RecordMessageReceived(t *testing.T) {
	// Reset metrics
	MessagesReceived.Reset()

	recorder := NewMetricsRecorder()

	// Record message
	recorder.RecordMessageReceived("1", "HIGH")

	// Verify metric recorded (basic check)
	// Detailed verification is done in metrics_test.go
}

func TestMetricsRecorder_RecordMessageProcessing(t *testing.T) {
	// Reset metrics
	MessageLatency.Reset()

	recorder := NewMetricsRecorder()

	// Record processing with success
	err := recorder.RecordMessageProcessing("worker", func() error {
		time.Sleep(5 * time.Millisecond)
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Record processing with error
	expectedErr := errors.New("processing failed")
	err = recorder.RecordMessageProcessing("worker", func() error {
		return expectedErr
	})

	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
}

func TestMetricsRecorder_RecordBalanceOperationWithTiming(t *testing.T) {
	// Reset metrics
	BalanceOperations.Reset()
	BalanceOperationDuration.Reset()

	recorder := NewMetricsRecorder()

	// Record successful operation
	err := recorder.RecordBalanceOperationWithTiming("1", "DEDUCT", func() error {
		time.Sleep(5 * time.Millisecond)
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Record failed operation
	expectedErr := errors.New("insufficient balance")
	err = recorder.RecordBalanceOperationWithTiming("1", "DEDUCT", func() error {
		return expectedErr
	})

	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
}
