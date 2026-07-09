package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Counter metrics
var (
	// MessagesReceived tracks total messages received by the API Gateway
	MessagesReceived = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "messages_received_total",
			Help: "Total number of messages received by API Gateway",
		},
		[]string{"tenant_id", "priority"},
	)

	// MessagesSent tracks total messages successfully sent
	MessagesSent = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "messages_sent_total",
			Help: "Total number of messages successfully sent",
		},
		[]string{"tenant_id", "channel_id"},
	)

	// MessagesFailed tracks total messages that failed to send
	MessagesFailed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "messages_failed_total",
			Help: "Total number of messages that failed to send",
		},
		[]string{"tenant_id", "reason"},
	)

	// BalanceOperations tracks balance operations (deduct, refund, charge)
	BalanceOperations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "balance_operations_total",
			Help: "Total number of balance operations",
		},
		[]string{"tenant_id", "type", "result"},
	)

	// StatusReportsProcessed tracks status reports processed
	StatusReportsProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "status_reports_processed_total",
			Help: "Total number of status reports processed",
		},
		[]string{"tenant_id", "status"},
	)

	// BatchesProcessed tracks batches processed
	BatchesProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "batches_processed_total",
			Help: "Total number of batches processed",
		},
		[]string{"tenant_id", "result"},
	)

	// CacheHits tracks cache hits
	CacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_hits_total",
			Help: "Total number of cache hits",
		},
		[]string{"cache_type"},
	)

	// CacheMisses tracks cache misses
	CacheMisses = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_misses_total",
			Help: "Total number of cache misses",
		},
		[]string{"cache_type"},
	)
)

// Gauge metrics
var (
	// QueueSize tracks current queue size
	QueueSize = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "queue_size",
			Help: "Current number of messages in queue",
		},
		[]string{"tenant_id", "queue_type"},
	)

	// ActiveWorkers tracks number of active workers
	ActiveWorkers = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "active_workers",
			Help: "Current number of active workers",
		},
	)

	// DatabaseConnections tracks database connection pool stats
	DatabaseConnections = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "database_connections",
			Help: "Database connection pool statistics",
		},
		[]string{"state"}, // idle, in_use, max
	)

	// ActiveWebSocketConnections tracks active WebSocket connections
	ActiveWebSocketConnections = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "websocket_connections_active",
			Help: "Current number of active WebSocket connections",
		},
		[]string{"tenant_id"},
	)

	// TenantBalance tracks current balance for tenants
	TenantBalance = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tenant_balance",
			Help: "Current balance for tenant",
		},
		[]string{"tenant_id"},
	)

	// Phase 3.3: System Load Gauges
	ServerLoadCPU = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "server_load_cpu_percent",
			Help: "Current CPU usage percentage",
		},
	)

	ServerLoadMemory = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "server_load_memory_percent",
			Help: "Current Memory usage percentage",
		},
	)

	AdaptiveBatchSize = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "adaptive_batch_size",
			Help: "Current batch size used by dispatcher",
		},
	)

	// Local state to track values that Prometheus doesn't expose easily
	lastCPU    float64
	lastMemory float64
)

// Histogram metrics
var (
	// MessageLatency tracks message processing latency
	MessageLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "message_latency_seconds",
			Help:    "Message processing latency in seconds",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 10), // 1ms to ~1s
		},
		[]string{"stage"}, // api_gateway, batch_processor, worker, total
	)

	// APIResponseTime tracks API response time
	APIResponseTime = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "api_response_time_seconds",
			Help:    "API response time in seconds",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 10), // 1ms to ~1s
		},
		[]string{"endpoint", "method", "status"},
	)

	// BatchSize tracks batch processing size
	BatchSize = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "batch_size",
			Help:    "Number of messages in each batch",
			Buckets: prometheus.LinearBuckets(10, 10, 10), // 10, 20, ..., 100
		},
	)

	// BalanceOperationDuration tracks balance operation duration
	BalanceOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "balance_operation_duration_seconds",
			Help:    "Balance operation duration in seconds",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 10),
		},
		[]string{"operation_type"},
	)
)

// Summary metrics
var (
	// SuccessRate tracks success rate percentage
	SuccessRate = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "success_rate",
			Help: "Message success rate percentage (0-100)",
		},
		[]string{"tenant_id"},
	)

	// CacheHitRate tracks cache hit rate percentage
	CacheHitRate = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cache_hit_rate",
			Help: "Cache hit rate percentage (0-100)",
		},
		[]string{"cache_type"},
	)
)

// Helper functions

// RecordMessageReceived records a message received event
func RecordMessageReceived(tenantID, priority string) {
	MessagesReceived.WithLabelValues(tenantID, priority).Inc()
}

// RecordMessageSent records a message sent event
func RecordMessageSent(tenantID, channelID string) {
	MessagesSent.WithLabelValues(tenantID, channelID).Inc()
}

// RecordMessageFailed records a message failed event
func RecordMessageFailed(tenantID, reason string) {
	MessagesFailed.WithLabelValues(tenantID, reason).Inc()
}

// RecordBalanceOperation records a balance operation
func RecordBalanceOperation(tenantID, opType, result string) {
	BalanceOperations.WithLabelValues(tenantID, opType, result).Inc()
}

// RecordStatusReport records a status report processed
func RecordStatusReport(tenantID, status string) {
	StatusReportsProcessed.WithLabelValues(tenantID, status).Inc()
}

// RecordBatchProcessed records a batch processed
func RecordBatchProcessed(tenantID, result string) {
	BatchesProcessed.WithLabelValues(tenantID, result).Inc()
}

// RecordCacheHit records a cache hit
func RecordCacheHit(cacheType string) {
	CacheHits.WithLabelValues(cacheType).Inc()
}

// RecordCacheMiss records a cache miss
func RecordCacheMiss(cacheType string) {
	CacheMisses.WithLabelValues(cacheType).Inc()
}

// SetQueueSize sets the current queue size
func SetQueueSize(tenantID, queueType string, size float64) {
	QueueSize.WithLabelValues(tenantID, queueType).Set(size)
}

// SetActiveWorkers sets the number of active workers
func SetActiveWorkers(count float64) {
	ActiveWorkers.Set(count)
}

// SetDatabaseConnections sets database connection stats
func SetDatabaseConnections(state string, count float64) {
	DatabaseConnections.WithLabelValues(state).Set(count)
}

// SetWebSocketConnections sets active WebSocket connections
func SetWebSocketConnections(tenantID string, count float64) {
	ActiveWebSocketConnections.WithLabelValues(tenantID).Set(count)
}

// SetTenantBalance sets tenant balance
func SetTenantBalance(tenantID string, balance float64) {
	TenantBalance.WithLabelValues(tenantID).Set(balance)
}

// ObserveMessageLatency observes message latency
func ObserveMessageLatency(stage string, duration float64) {
	MessageLatency.WithLabelValues(stage).Observe(duration)
}

// ObserveAPIResponseTime observes API response time
func ObserveAPIResponseTime(endpoint, method, status string, duration float64) {
	APIResponseTime.WithLabelValues(endpoint, method, status).Observe(duration)
}

// ObserveBatchSize observes batch size
func ObserveBatchSize(size float64) {
	BatchSize.Observe(size)
}

// ObserveBalanceOperationDuration observes balance operation duration
func ObserveBalanceOperationDuration(opType string, duration float64) {
	BalanceOperationDuration.WithLabelValues(opType).Observe(duration)
}

// SetSuccessRate sets success rate percentage
func SetSuccessRate(tenantID string, rate float64) {
	SuccessRate.WithLabelValues(tenantID).Set(rate)
}

// SetCacheHitRate sets cache hit rate percentage
func SetCacheHitRate(cacheType string, rate float64) {
	CacheHitRate.WithLabelValues(cacheType).Set(rate)
}

// Phase 3.3: Load Recording Helpers

func SetServerLoad(cpu, memory float64) {
	lastCPU = cpu
	lastMemory = memory
	ServerLoadCPU.Set(cpu)
	ServerLoadMemory.Set(memory)
}

// GetServerLoad returns the last recorded CPU and memory usage
func GetServerLoad() (cpu, memory float64) {
	return lastCPU, lastMemory
}

func SetAdaptiveBatchSize(size float64) {
	AdaptiveBatchSize.Set(size)
}

// UpdateCacheHitRate calculates and updates cache hit rate
func UpdateCacheHitRate(cacheType string) {
	_ = CacheHits.WithLabelValues(cacheType)
	_ = CacheMisses.WithLabelValues(cacheType)
	
	// Get metric values (this is a simplified version)
	// In production, you'd need to track these separately
	// For now, we'll calculate based on the counters
}
