package metrics

import (
	"net/http"
	"strconv"
	"time"
)

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

// HTTPMetricsMiddleware records HTTP metrics for all requests
func HTTPMetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
			written:        false,
		}

		// Process request
		next.ServeHTTP(wrapped, r)

		// Record metrics
		duration := time.Since(start).Seconds()
		endpoint := r.URL.Path
		method := r.Method
		status := strconv.Itoa(wrapped.statusCode)

		ObserveAPIResponseTime(endpoint, method, status, duration)
	})
}

// MetricsRecorder provides methods to record various metrics
type MetricsRecorder struct{}

// NewMetricsRecorder creates a new metrics recorder
func NewMetricsRecorder() *MetricsRecorder {
	return &MetricsRecorder{}
}

// RecordMessageReceived records a message received event with timing
func (m *MetricsRecorder) RecordMessageReceived(tenantID, priority string) {
	RecordMessageReceived(tenantID, priority)
}

// RecordMessageProcessing records message processing with latency
func (m *MetricsRecorder) RecordMessageProcessing(stage string, fn func() error) error {
	start := time.Now()
	err := fn()
	duration := time.Since(start).Seconds()
	
	ObserveMessageLatency(stage, duration)
	
	return err
}

// RecordBalanceOperationWithTiming records balance operation with timing
func (m *MetricsRecorder) RecordBalanceOperationWithTiming(tenantID, opType string, fn func() error) error {
	start := time.Now()
	err := fn()
	duration := time.Since(start).Seconds()
	
	result := "success"
	if err != nil {
		result = "failure"
	}
	
	RecordBalanceOperation(tenantID, opType, result)
	ObserveBalanceOperationDuration(opType, duration)
	
	return err
}
