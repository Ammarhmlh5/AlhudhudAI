package health

import (
	"testing"
)

func TestStatusClassifier_ClassifyHealthStatus(t *testing.T) {
	classifier := NewStatusClassifier()
	
	tests := []struct {
		name           string
		responseTimeMs int
		errorRate      float64
		expectedStatus HealthStatus
	}{
		{
			name:           "Healthy - low response time and error rate",
			responseTimeMs: 100,
			errorRate:      0.5,
			expectedStatus: Healthy,
		},
		{
			name:           "Healthy - at threshold boundary",
			responseTimeMs: 199,
			errorRate:      0.9,
			expectedStatus: Healthy,
		},
		{
			name:           "Degraded - response time at lower boundary",
			responseTimeMs: 200,
			errorRate:      0.5,
			expectedStatus: Degraded,
		},
		{
			name:           "Degraded - response time in middle range",
			responseTimeMs: 350,
			errorRate:      0.5,
			expectedStatus: Degraded,
		},
		{
			name:           "Degraded - error rate at lower boundary",
			responseTimeMs: 100,
			errorRate:      1.0,
			expectedStatus: Degraded,
		},
		{
			name:           "Degraded - error rate in middle range",
			responseTimeMs: 100,
			errorRate:      3.0,
			expectedStatus: Degraded,
		},
		{
			name:           "Unhealthy - high response time",
			responseTimeMs: 501,
			errorRate:      0.5,
			expectedStatus: Unhealthy,
		},
		{
			name:           "Unhealthy - high error rate",
			responseTimeMs: 100,
			errorRate:      5.1,
			expectedStatus: Unhealthy,
		},
		{
			name:           "Unhealthy - both high",
			responseTimeMs: 600,
			errorRate:      10.0,
			expectedStatus: Unhealthy,
		},
		{
			name:           "Unhealthy - exactly at threshold",
			responseTimeMs: 501,
			errorRate:      5.1,
			expectedStatus: Unhealthy,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := classifier.ClassifyHealthStatus(tt.responseTimeMs, tt.errorRate)
			if status != tt.expectedStatus {
				t.Errorf("ClassifyHealthStatus(%d, %.2f) = %s, want %s",
					tt.responseTimeMs, tt.errorRate, status, tt.expectedStatus)
			}
		})
	}
}

func TestStatusClassifier_IsHealthy(t *testing.T) {
	classifier := NewStatusClassifier()
	
	tests := []struct {
		name           string
		responseTimeMs int
		errorRate      float64
		expected       bool
	}{
		{"Healthy case", 100, 0.5, true},
		{"Degraded case", 300, 0.5, false},
		{"Unhealthy case", 600, 10.0, false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.IsHealthy(tt.responseTimeMs, tt.errorRate)
			if result != tt.expected {
				t.Errorf("IsHealthy(%d, %.2f) = %v, want %v",
					tt.responseTimeMs, tt.errorRate, result, tt.expected)
			}
		})
	}
}

func TestStatusClassifier_IsDegraded(t *testing.T) {
	classifier := NewStatusClassifier()
	
	tests := []struct {
		name           string
		responseTimeMs int
		errorRate      float64
		expected       bool
	}{
		{"Healthy case", 100, 0.5, false},
		{"Degraded case", 300, 0.5, true},
		{"Unhealthy case", 600, 10.0, false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.IsDegraded(tt.responseTimeMs, tt.errorRate)
			if result != tt.expected {
				t.Errorf("IsDegraded(%d, %.2f) = %v, want %v",
					tt.responseTimeMs, tt.errorRate, result, tt.expected)
			}
		})
	}
}

func TestStatusClassifier_IsUnhealthy(t *testing.T) {
	classifier := NewStatusClassifier()
	
	tests := []struct {
		name           string
		responseTimeMs int
		errorRate      float64
		expected       bool
	}{
		{"Healthy case", 100, 0.5, false},
		{"Degraded case", 300, 0.5, false},
		{"Unhealthy case", 600, 10.0, true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.IsUnhealthy(tt.responseTimeMs, tt.errorRate)
			if result != tt.expected {
				t.Errorf("IsUnhealthy(%d, %.2f) = %v, want %v",
					tt.responseTimeMs, tt.errorRate, result, tt.expected)
			}
		})
	}
}

func TestStatusClassifier_CustomThresholds(t *testing.T) {
	// Create classifier with custom thresholds
	classifier := NewStatusClassifierWithThresholds(
		100, 100, 300,  // Response time thresholds
		0.5, 0.5, 3.0,  // Error rate thresholds
	)
	
	tests := []struct {
		name           string
		responseTimeMs int
		errorRate      float64
		expectedStatus HealthStatus
	}{
		{
			name:           "Healthy with custom thresholds",
			responseTimeMs: 50,
			errorRate:      0.3,
			expectedStatus: Healthy,
		},
		{
			name:           "Degraded with custom thresholds",
			responseTimeMs: 150,
			errorRate:      0.3,
			expectedStatus: Degraded,
		},
		{
			name:           "Unhealthy with custom thresholds",
			responseTimeMs: 350,
			errorRate:      0.3,
			expectedStatus: Unhealthy,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := classifier.ClassifyHealthStatus(tt.responseTimeMs, tt.errorRate)
			if status != tt.expectedStatus {
				t.Errorf("ClassifyHealthStatus(%d, %.2f) = %s, want %s",
					tt.responseTimeMs, tt.errorRate, status, tt.expectedStatus)
			}
		})
	}
}

func TestStatusClassifier_UpdateThresholds(t *testing.T) {
	classifier := NewStatusClassifier()
	
	// Update thresholds
	classifier.UpdateThresholds(
		150, 150, 400,
		0.8, 0.8, 4.0,
	)
	
	// Test with new thresholds
	status := classifier.ClassifyHealthStatus(200, 1.0)
	if status != Degraded {
		t.Errorf("After updating thresholds, ClassifyHealthStatus(200, 1.0) = %s, want %s",
			status, Degraded)
	}
}

func TestStatusClassifier_GetThresholds(t *testing.T) {
	classifier := NewStatusClassifier()
	
	thresholds := classifier.GetThresholds()
	
	if thresholds["healthy_response_time_ms"] != 200 {
		t.Errorf("Expected healthy_response_time_ms = 200, got %v",
			thresholds["healthy_response_time_ms"])
	}
	
	if thresholds["unhealthy_error_rate"] != 5.0 {
		t.Errorf("Expected unhealthy_error_rate = 5.0, got %v",
			thresholds["unhealthy_error_rate"])
	}
}

func TestStatusClassifier_ClassifyHealthStatusFromResult(t *testing.T) {
	classifier := NewStatusClassifier()
	
	tests := []struct {
		name           string
		result         *HealthCheckResult
		expectedStatus HealthStatus
	}{
		{
			name: "Unavailable server is unhealthy",
			result: &HealthCheckResult{
				ResponseTimeMs: 100,
				ErrorRate:      0.5,
				IsAvailable:    false,
			},
			expectedStatus: Unhealthy,
		},
		{
			name: "Available server with good metrics is healthy",
			result: &HealthCheckResult{
				ResponseTimeMs: 100,
				ErrorRate:      0.5,
				IsAvailable:    true,
			},
			expectedStatus: Healthy,
		},
		{
			name: "Available server with degraded metrics",
			result: &HealthCheckResult{
				ResponseTimeMs: 300,
				ErrorRate:      2.0,
				IsAvailable:    true,
			},
			expectedStatus: Degraded,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := classifier.ClassifyHealthStatusFromResult(tt.result)
			if status != tt.expectedStatus {
				t.Errorf("ClassifyHealthStatusFromResult() = %s, want %s",
					status, tt.expectedStatus)
			}
		})
	}
}
