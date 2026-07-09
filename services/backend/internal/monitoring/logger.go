package monitoring

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/google/uuid"
)

// Logger logs resource usage metrics to the database
type Logger struct {
	db        *sql.DB
	collector *Collector
}

// NewLogger creates a new usage logger
func NewLogger(db *sql.DB, collector *Collector) *Logger {
	return &Logger{
		db:        db,
		collector: collector,
	}
}

// LogResourceUsage logs resource usage for all active gateways
// This should be called every minute
func (l *Logger) LogResourceUsage(ctx context.Context) error {
	// Collect metrics for all active gateways
	allMetrics, err := l.collector.CollectAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to collect metrics: %w", err)
	}

	// Log each gateway's metrics
	for _, metrics := range allMetrics {
		if err := l.logGatewayMetrics(ctx, &metrics); err != nil {
			// Log error but continue with other gateways
			fmt.Printf("Error logging metrics for gateway %d: %v\n", metrics.GatewayID, err)
			continue
		}
	}

	return nil
}

// logGatewayMetrics logs metrics for a single gateway
func (l *Logger) logGatewayMetrics(ctx context.Context, metrics *GatewayMetrics) error {
	// Get additional metrics (messages, API requests, workers) - these don't need tenant context
	messageMetrics, err := l.getMessageMetrics(ctx, metrics.GatewayID)
	if err != nil {
		return fmt.Errorf("failed to get message metrics: %w", err)
	}

	apiMetrics, err := l.getAPIMetrics(ctx, metrics.GatewayID)
	if err != nil {
		return fmt.Errorf("failed to get API metrics: %w", err)
	}

	workerMetrics, err := l.getWorkerMetrics(ctx, metrics.GatewayID)
	if err != nil {
		return fmt.Errorf("failed to get worker metrics: %w", err)
	}

	// Insert into resource_usage table using tenant-scoped transaction
	query := `
		INSERT INTO resource_usage (
			gateway_id,
			tenant_id,
			timestamp,
			cpu_usage_percent,
			memory_usage_mb,
			bandwidth_usage_mbps,
			active_connections,
			db_connections_active,
			queries_per_second,
			redis_memory_usage_mb,
			redis_keys_count,
			messages_processed_last_hour,
			messages_processed_today,
			current_queue_depth,
			api_requests_last_minute,
			api_requests_last_hour,
			active_worker_threads
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
		)
	`

	return db.RunInTenantTx(ctx, metrics.TenantID, false, l.db, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx,
			query,
			metrics.GatewayID,
			metrics.TenantID,
			metrics.Timestamp,
			metrics.SystemMetrics.CPUUsagePercent,
			int(metrics.SystemMetrics.MemoryUsageMB),
			metrics.SystemMetrics.BandwidthUsageKBps/1024, // Convert KB/s to MB/s
			0, // active_connections - placeholder
			metrics.DatabaseMetrics.ActiveConnections,
			int(metrics.DatabaseMetrics.QueriesPerSecond),
			int(metrics.RedisMetrics.MemoryUsed/1024/1024),
			int(metrics.RedisMetrics.KeysCount),
			messageMetrics.LastHour,
			messageMetrics.Today,
			messageMetrics.QueueDepth,
			apiMetrics.LastMinute,
			apiMetrics.LastHour,
			workerMetrics.ActiveThreads,
		)
		return err
	})
}

// MessageMetrics represents message processing metrics
type MessageMetrics struct {
	LastHour   int
	Today      int
	QueueDepth int
}

// getMessageMetrics retrieves message processing metrics
func (l *Logger) getMessageMetrics(ctx context.Context, gatewayID int) (*MessageMetrics, error) {
	metrics := &MessageMetrics{}

	// Get messages processed in the last hour
	query := `
		SELECT COUNT(*)
		FROM messages
		WHERE gateway_id = $1
		  AND created_at >= NOW() - INTERVAL '1 hour'
		  AND deleted_at IS NULL
	`
	err := l.db.QueryRowContext(ctx, query, gatewayID).Scan(&metrics.LastHour)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get messages last hour: %w", err)
	}

	// Get messages processed today
	query = `
		SELECT COUNT(*)
		FROM messages
		WHERE gateway_id = $1
		  AND DATE(created_at) = CURRENT_DATE
		  AND deleted_at IS NULL
	`
	err = l.db.QueryRowContext(ctx, query, gatewayID).Scan(&metrics.Today)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get messages today: %w", err)
	}

	// Get current queue depth
	// This would depend on your queue implementation (River, Redis, etc.)
	// For now, we'll use a placeholder
	metrics.QueueDepth = 0

	return metrics, nil
}

// APIMetrics represents API request metrics
type APIMetrics struct {
	LastMinute int
	LastHour   int
}

// getAPIMetrics retrieves API request metrics
func (l *Logger) getAPIMetrics(ctx context.Context, gatewayID int) (*APIMetrics, error) {
	metrics := &APIMetrics{}

	// Get API requests in the last minute
	// This would typically come from an API request log table
	// For now, we'll use placeholders
	metrics.LastMinute = 0
	metrics.LastHour = 0

	return metrics, nil
}

// WorkerMetrics represents worker thread metrics
type WorkerMetrics struct {
	ActiveThreads int
}

// getWorkerMetrics retrieves worker thread metrics
func (l *Logger) getWorkerMetrics(ctx context.Context, gatewayID int) (*WorkerMetrics, error) {
	metrics := &WorkerMetrics{}

	// Get active worker threads
	// This would depend on your worker pool implementation
	// For now, we'll use a placeholder
	metrics.ActiveThreads = 0

	return metrics, nil
}

// StartPeriodicLogging starts logging resource usage every minute
func (l *Logger) StartPeriodicLogging(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	// Log immediately on start
	if err := l.LogResourceUsage(ctx); err != nil {
		fmt.Printf("Error logging resource usage: %v\n", err)
	}

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Stopping periodic resource usage logging")
			return
		case <-ticker.C:
			if err := l.LogResourceUsage(ctx); err != nil {
				fmt.Printf("Error logging resource usage: %v\n", err)
			}
		}
	}
}

// LogForGateway logs resource usage for a specific gateway
func (l *Logger) LogForGateway(ctx context.Context, gatewayID int, tenantID uuid.UUID) error {
	// Collect metrics for the specific gateway
	metrics, err := l.collector.CollectForGateway(ctx, gatewayID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to collect metrics for gateway %d: %w", gatewayID, err)
	}

	// Log the metrics
	if err := l.logGatewayMetrics(ctx, metrics); err != nil {
		return fmt.Errorf("failed to log metrics for gateway %d: %w", gatewayID, err)
	}

	return nil
}

// GetRecentUsage retrieves recent resource usage for a gateway
func (l *Logger) GetRecentUsage(ctx context.Context, gatewayID int, tenantID uuid.UUID, limit int) ([]ResourceUsageRecord, error) {
	query := `
		SELECT
			id,
			gateway_id,
			tenant_id,
			timestamp,
			cpu_usage_percent,
			memory_usage_mb,
			bandwidth_usage_mbps,
			active_connections,
			db_connections_active,
			queries_per_second,
			redis_memory_usage_mb,
			redis_keys_count,
			messages_processed_last_hour,
			messages_processed_today,
			current_queue_depth,
			api_requests_last_minute,
			api_requests_last_hour,
			active_worker_threads
		FROM resource_usage
		WHERE gateway_id = $1
		ORDER BY timestamp DESC
		LIMIT $2
	`

	var records []ResourceUsageRecord
	err := db.RunInTenantTx(ctx, tenantID, false, l.db, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, query, gatewayID, limit)
		if err != nil {
			return fmt.Errorf("failed to query resource usage: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var record ResourceUsageRecord
			var cpuUsage, bandwidthUsage sql.NullFloat64
			var memoryUsage, activeConns, dbConns, qps, redisMemory, redisKeys sql.NullInt64
			var messagesHour, messagesToday, queueDepth, apiMinute, apiHour, workers sql.NullInt64

			err := rows.Scan(
				&record.ID,
				&record.GatewayID,
				&record.TenantID,
				&record.Timestamp,
				&cpuUsage,
				&memoryUsage,
				&bandwidthUsage,
				&activeConns,
				&dbConns,
				&qps,
				&redisMemory,
				&redisKeys,
				&messagesHour,
				&messagesToday,
				&queueDepth,
				&apiMinute,
				&apiHour,
				&workers,
			)
			if err != nil {
				return fmt.Errorf("failed to scan resource usage: %w", err)
			}

			// Handle nullable fields
			if cpuUsage.Valid {
				record.CPUUsagePercent = cpuUsage.Float64
			}
			if memoryUsage.Valid {
				record.MemoryUsageMB = int(memoryUsage.Int64)
			}
			if bandwidthUsage.Valid {
				record.BandwidthUsageMBps = bandwidthUsage.Float64
			}
			if dbConns.Valid {
				record.DBConnectionsActive = int(dbConns.Int64)
			}
			if qps.Valid {
				record.QueriesPerSecond = int(qps.Int64)
			}
			if redisMemory.Valid {
				record.RedisMemoryUsageMB = int(redisMemory.Int64)
			}
			if redisKeys.Valid {
				record.RedisKeysCount = int(redisKeys.Int64)
			}
			if messagesHour.Valid {
				record.MessagesProcessedLastHour = int(messagesHour.Int64)
			}
			if messagesToday.Valid {
				record.MessagesProcessedToday = int(messagesToday.Int64)
			}
			if queueDepth.Valid {
				record.CurrentQueueDepth = int(queueDepth.Int64)
			}
			if apiMinute.Valid {
				record.APIRequestsLastMinute = int(apiMinute.Int64)
			}
			if apiHour.Valid {
				record.APIRequestsLastHour = int(apiHour.Int64)
			}
			if workers.Valid {
				record.ActiveWorkerThreads = int(workers.Int64)
			}

			records = append(records, record)
		}

		return rows.Err()
	})

	if err != nil {
		return nil, err
	}
	return records, nil
}

// ResourceUsageRecord represents a single resource usage record
type ResourceUsageRecord struct {
	ID                        int64
	GatewayID                 int
	TenantID                  uuid.UUID
	Timestamp                 time.Time
	CPUUsagePercent           float64
	MemoryUsageMB             int
	BandwidthUsageMBps        float64
	ActiveConnections         int
	DBConnectionsActive       int
	QueriesPerSecond          int
	RedisMemoryUsageMB        int
	RedisKeysCount            int
	MessagesProcessedLastHour int
	MessagesProcessedToday    int
	CurrentQueueDepth         int
	APIRequestsLastMinute     int
	APIRequestsLastHour       int
	ActiveWorkerThreads       int
}
