package monitoring

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Rating System Metrics
	RatingCalculationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "admin_rating_calculation_duration_seconds",
			Help:    "Duration of rating calculations in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"tenant_id"},
	)

	RatingCalculationTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "admin_rating_calculation_total",
			Help: "Total number of rating calculations",
		},
		[]string{"tenant_id", "category"},
	)

	CompanyRatingScore = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "admin_company_rating_score",
			Help: "Current rating score for companies",
		},
		[]string{"tenant_id", "category"},
	)

	// Server Health Metrics
	ServerHealthCheckDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "admin_server_health_check_duration_seconds",
			Help:    "Duration of server health checks in seconds",
			Buckets: []float64{0.05, 0.1, 0.2, 0.5, 1.0, 2.0, 5.0},
		},
		[]string{"server_id", "tenant_id"},
	)

	ServerHealthCheckTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "admin_server_health_check_total",
			Help: "Total number of server health checks",
		},
		[]string{"server_id", "tenant_id", "status"},
	)

	ServerHealthStatus = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "admin_server_health_status",
			Help: "Current health status of servers (0=unhealthy, 1=degraded, 2=healthy)",
		},
		[]string{"server_id", "tenant_id"},
	)

	ServerResponseTime = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "admin_server_response_time_ms",
			Help: "Current response time of servers in milliseconds",
		},
		[]string{"server_id", "tenant_id"},
	)

	// Distribution Metrics
	MessageDistributionTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "admin_message_distribution_total",
			Help: "Total number of messages distributed",
		},
		[]string{"tenant_id", "device_id", "strategy"},
	)

	DeviceLoadCurrent = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "admin_device_load_current",
			Help: "Current load on devices",
		},
		[]string{"tenant_id", "device_id"},
	)

	DeviceStatusActive = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "admin_device_status_active",
			Help: "Device active status (0=inactive, 1=active)",
		},
		[]string{"tenant_id", "device_id"},
	)

	DistributionStrategyUsage = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "admin_distribution_strategy_usage_total",
			Help: "Total usage count of distribution strategies",
		},
		[]string{"tenant_id", "strategy_type"},
	)

	// Signal/Push Timing Metrics
	TimingViolationTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "admin_timing_violation_total",
			Help: "Total number of timing violations",
		},
		[]string{"tenant_id", "message_type", "severity"},
	)

	MessageGapDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "admin_message_gap_duration_ms",
			Help:    "Duration between messages in milliseconds",
			Buckets: []float64{100, 200, 500, 1000, 2000, 5000, 10000},
		},
		[]string{"tenant_id", "message_type"},
	)

	CurrentMessagesPerMinute = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "admin_current_messages_per_minute",
			Help: "Current messages per minute rate",
		},
		[]string{"tenant_id", "message_type"},
	)

	AnomalyDetectionScore = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "admin_anomaly_detection_score",
			Help: "Current anomaly detection score",
		},
		[]string{"tenant_id", "message_type"},
	)

	EnforcementActionTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "admin_enforcement_action_total",
			Help: "Total number of enforcement actions taken",
		},
		[]string{"tenant_id", "action_type", "anomaly_level"},
	)

	// Alert Metrics
	AlertCreatedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "admin_alert_created_total",
			Help: "Total number of alerts created",
		},
		[]string{"tenant_id", "alert_type", "severity"},
	)

	AlertAcknowledgedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "admin_alert_acknowledged_total",
			Help: "Total number of alerts acknowledged",
		},
		[]string{"tenant_id", "alert_type"},
	)

	AlertResolvedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "admin_alert_resolved_total",
			Help: "Total number of alerts resolved",
		},
		[]string{"tenant_id", "alert_type"},
	)

	ActiveAlertsCount = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "admin_active_alerts_count",
			Help: "Current number of active alerts",
		},
		[]string{"tenant_id", "severity"},
	)

	// API Metrics
	APIRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "admin_api_request_duration_seconds",
			Help:    "Duration of API requests in seconds",
			Buckets: []float64{0.01, 0.05, 0.1, 0.2, 0.5, 1.0, 2.0},
		},
		[]string{"method", "endpoint", "status_code"},
	)

	APIRequestTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "admin_api_request_total",
			Help: "Total number of API requests",
		},
		[]string{"method", "endpoint", "status_code"},
	)

	// Routing Metrics
	RoutingDecisionDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "admin_routing_decision_duration_seconds",
			Help:    "Duration of routing decisions in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.2},
		},
		[]string{"tenant_id", "routing_type"},
	)

	RoutingDecisionTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "admin_routing_decision_total",
			Help: "Total number of routing decisions",
		},
		[]string{"tenant_id", "routing_type", "result"},
	)

	DedicatedServerAvailability = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "admin_dedicated_server_availability",
			Help: "Availability of dedicated servers (0=unavailable, 1=available)",
		},
		[]string{"tenant_id", "server_id"},
	)

	// Performance Metrics
	PerformanceMetricsCollectionDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "admin_performance_metrics_collection_duration_seconds",
			Help:    "Duration of performance metrics collection in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"tenant_id", "period_type"},
	)

	MessageSuccessRate = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "admin_message_success_rate",
			Help: "Message success rate percentage",
		},
		[]string{"tenant_id", "period_type"},
	)

	MessageVolumeTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "admin_message_volume_total",
			Help: "Total message volume",
		},
		[]string{"tenant_id", "status"},
	)
)

// Helper functions to convert status to numeric values
func HealthStatusToValue(status string) float64 {
	switch status {
	case "healthy":
		return 2
	case "degraded":
		return 1
	case "unhealthy":
		return 0
	default:
		return -1
	}
}

func DeviceStatusToValue(status string) float64 {
	if status == "active" {
		return 1
	}
	return 0
}

func ServerAvailabilityToValue(available bool) float64 {
	if available {
		return 1
	}
	return 0
}
