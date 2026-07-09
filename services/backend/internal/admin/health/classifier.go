package health

// StatusClassifier classifies server health status based on response time and error rate
type StatusClassifier struct {
	// Thresholds for classification
	healthyResponseTimeMs   int
	degradedResponseTimeMs  int
	unhealthyResponseTimeMs int
	healthyErrorRate        float64
	degradedErrorRate       float64
	unhealthyErrorRate      float64
}

// NewStatusClassifier creates a new status classifier with default thresholds
func NewStatusClassifier() *StatusClassifier {
	return &StatusClassifier{
		healthyResponseTimeMs:   200,  // < 200ms
		degradedResponseTimeMs:  200,  // 200-500ms
		unhealthyResponseTimeMs: 500,  // > 500ms
		healthyErrorRate:        1.0,  // < 1%
		degradedErrorRate:       1.0,  // 1-5%
		unhealthyErrorRate:      5.0,  // > 5%
	}
}

// NewStatusClassifierWithThresholds creates a classifier with custom thresholds
func NewStatusClassifierWithThresholds(
	healthyResponseMs, degradedResponseMs, unhealthyResponseMs int,
	healthyErrorRate, degradedErrorRate, unhealthyErrorRate float64,
) *StatusClassifier {
	return &StatusClassifier{
		healthyResponseTimeMs:   healthyResponseMs,
		degradedResponseTimeMs:  degradedResponseMs,
		unhealthyResponseTimeMs: unhealthyResponseMs,
		healthyErrorRate:        healthyErrorRate,
		degradedErrorRate:       degradedErrorRate,
		unhealthyErrorRate:      unhealthyErrorRate,
	}
}

// ClassifyHealthStatus classifies the health status based on response time and error rate
// Classification criteria:
// - Healthy: response_time < 200ms AND error_rate < 1%
// - Degraded: response_time 200-500ms OR error_rate 1-5%
// - Unhealthy: response_time > 500ms OR error_rate > 5%
func (sc *StatusClassifier) ClassifyHealthStatus(responseTimeMs int, errorRate float64) HealthStatus {
	// Check for unhealthy conditions first (most severe)
	if responseTimeMs > sc.unhealthyResponseTimeMs || errorRate > sc.unhealthyErrorRate {
		return Unhealthy
	}
	
	// Check for degraded conditions
	if (responseTimeMs >= sc.degradedResponseTimeMs && responseTimeMs <= sc.unhealthyResponseTimeMs) ||
		(errorRate >= sc.degradedErrorRate && errorRate <= sc.unhealthyErrorRate) {
		return Degraded
	}
	
	// Otherwise, it's healthy
	return Healthy
}

// ClassifyHealthStatusFromResult classifies health status from a complete health check result
func (sc *StatusClassifier) ClassifyHealthStatusFromResult(result *HealthCheckResult) HealthStatus {
	// If server is not available, it's unhealthy
	if !result.IsAvailable {
		return Unhealthy
	}
	
	return sc.ClassifyHealthStatus(result.ResponseTimeMs, result.ErrorRate)
}

// IsHealthy checks if the given metrics indicate a healthy status
func (sc *StatusClassifier) IsHealthy(responseTimeMs int, errorRate float64) bool {
	return sc.ClassifyHealthStatus(responseTimeMs, errorRate) == Healthy
}

// IsDegraded checks if the given metrics indicate a degraded status
func (sc *StatusClassifier) IsDegraded(responseTimeMs int, errorRate float64) bool {
	return sc.ClassifyHealthStatus(responseTimeMs, errorRate) == Degraded
}

// IsUnhealthy checks if the given metrics indicate an unhealthy status
func (sc *StatusClassifier) IsUnhealthy(responseTimeMs int, errorRate float64) bool {
	return sc.ClassifyHealthStatus(responseTimeMs, errorRate) == Unhealthy
}

// GetThresholds returns the current thresholds
func (sc *StatusClassifier) GetThresholds() map[string]interface{} {
	return map[string]interface{}{
		"healthy_response_time_ms":   sc.healthyResponseTimeMs,
		"degraded_response_time_ms":  sc.degradedResponseTimeMs,
		"unhealthy_response_time_ms": sc.unhealthyResponseTimeMs,
		"healthy_error_rate":         sc.healthyErrorRate,
		"degraded_error_rate":        sc.degradedErrorRate,
		"unhealthy_error_rate":       sc.unhealthyErrorRate,
	}
}

// UpdateThresholds updates the classification thresholds
func (sc *StatusClassifier) UpdateThresholds(
	healthyResponseMs, degradedResponseMs, unhealthyResponseMs int,
	healthyErrorRate, degradedErrorRate, unhealthyErrorRate float64,
) {
	sc.healthyResponseTimeMs = healthyResponseMs
	sc.degradedResponseTimeMs = degradedResponseMs
	sc.unhealthyResponseTimeMs = unhealthyResponseMs
	sc.healthyErrorRate = healthyErrorRate
	sc.degradedErrorRate = degradedErrorRate
	sc.unhealthyErrorRate = unhealthyErrorRate
}
