package routing

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// MetricsTracker tracks routing metrics for dedicated servers
// Requirement 2.8: Maintain routing metrics including request count, success rate, and average latency
type MetricsTracker struct {
	db              *sqlx.DB
	mu              sync.RWMutex
	metricsCache    map[uuid.UUID]*TenantMetrics
	flushInterval   time.Duration
	stopChan        chan struct{}
	wg              sync.WaitGroup
}

// NewMetricsTracker creates a new metrics tracker instance
func NewMetricsTracker(db *sqlx.DB) *MetricsTracker {
	return &MetricsTracker{
		db:            db,
		metricsCache:  make(map[uuid.UUID]*TenantMetrics),
		flushInterval: 10 * time.Second, // Flush metrics every 10 seconds
		stopChan:      make(chan struct{}),
	}
}

// Start begins the periodic metrics flushing process
func (mt *MetricsTracker) Start(ctx context.Context) error {
	mt.wg.Add(1)
	go mt.runPeriodicFlush(ctx)
	return nil
}

// Stop gracefully stops the metrics tracker
func (mt *MetricsTracker) Stop() error {
	close(mt.stopChan)
	mt.wg.Wait()
	
	// Flush remaining metrics
	ctx := context.Background()
	mt.flushAllMetrics(ctx)
	
	return nil
}

// TrackRoutingDecision tracks a routing decision
// Uses atomic operations for concurrency safety
func (mt *MetricsTracker) TrackRoutingDecision(ctx context.Context, tenantID uuid.UUID, success bool, latencyMs int) {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	
	// Get or create metrics for tenant
	metrics, exists := mt.metricsCache[tenantID]
	if !exists {
		metrics = &TenantMetrics{
			TenantID:    tenantID,
			LastUpdated: time.Now(),
		}
		mt.metricsCache[tenantID] = metrics
	}
	
	// Update metrics using atomic operations
	atomic.AddInt64(&metrics.TotalRequests, 1)
	if success {
		atomic.AddInt64(&metrics.SuccessfulRequests, 1)
	} else {
		atomic.AddInt64(&metrics.FailedRequests, 1)
	}
	atomic.AddInt64(&metrics.TotalLatencyMs, int64(latencyMs))
	metrics.LastUpdated = time.Now()
}

// runPeriodicFlush periodically flushes metrics to the database
func (mt *MetricsTracker) runPeriodicFlush(ctx context.Context) {
	defer mt.wg.Done()
	
	ticker := time.NewTicker(mt.flushInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			mt.flushAllMetrics(ctx)
		case <-mt.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// flushAllMetrics flushes all cached metrics to the database
func (mt *MetricsTracker) flushAllMetrics(ctx context.Context) {
	mt.mu.Lock()
	
	// Copy metrics to avoid holding lock during DB operations
	metricsToFlush := make(map[uuid.UUID]*TenantMetrics)
	for tenantID, metrics := range mt.metricsCache {
		metricsToFlush[tenantID] = &TenantMetrics{
			TenantID:           metrics.TenantID,
			TotalRequests:      atomic.LoadInt64(&metrics.TotalRequests),
			SuccessfulRequests: atomic.LoadInt64(&metrics.SuccessfulRequests),
			FailedRequests:     atomic.LoadInt64(&metrics.FailedRequests),
			TotalLatencyMs:     atomic.LoadInt64(&metrics.TotalLatencyMs),
			LastUpdated:        metrics.LastUpdated,
		}
	}
	
	// Clear cache after copying
	mt.metricsCache = make(map[uuid.UUID]*TenantMetrics)
	mt.mu.Unlock()
	
	// Flush each tenant's metrics
	for _, metrics := range metricsToFlush {
		if err := mt.flushTenantMetrics(ctx, metrics); err != nil {
			fmt.Printf("Error flushing metrics for tenant %s: %v\n", metrics.TenantID, err)
		}
	}
}

// flushTenantMetrics flushes metrics for a single tenant to the database
func (mt *MetricsTracker) flushTenantMetrics(ctx context.Context, metrics *TenantMetrics) error {
	if metrics.TotalRequests == 0 {
		// No requests to flush
		return nil
	}
	
	// Calculate average latency
	var averageLatencyMs int
	if metrics.TotalRequests > 0 {
		averageLatencyMs = int(metrics.TotalLatencyMs / metrics.TotalRequests)
	}
	
	// Update routing configuration metrics
	query := `
		UPDATE routing_configurations
		SET 
			total_requests = total_requests + $1,
			successful_requests = successful_requests + $2,
			failed_requests = failed_requests + $3,
			average_latency_ms = CASE 
				WHEN total_requests = 0 THEN $4
				ELSE ((average_latency_ms * total_requests) + ($4 * $1)) / (total_requests + $1)
			END,
			updated_at = NOW()
		WHERE tenant_id = $5
	`
	
	result, err := mt.db.ExecContext(ctx, query,
		metrics.TotalRequests,
		metrics.SuccessfulRequests,
		metrics.FailedRequests,
		averageLatencyMs,
		metrics.TenantID,
	)
	if err != nil {
		return fmt.Errorf("failed to update routing metrics: %w", err)
	}
	
	// Check if any rows were affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		// No routing configuration exists, create one
		return mt.createRoutingConfigWithMetrics(ctx, metrics, averageLatencyMs)
	}
	
	return nil
}

// createRoutingConfigWithMetrics creates a new routing configuration with initial metrics
func (mt *MetricsTracker) createRoutingConfigWithMetrics(ctx context.Context, metrics *TenantMetrics, averageLatencyMs int) error {
	query := `
		INSERT INTO routing_configurations (
			tenant_id, routing_type, total_requests, successful_requests,
			failed_requests, average_latency_ms, is_active
		) VALUES ($1, 'shared', $2, $3, $4, $5, true)
		ON CONFLICT (tenant_id) DO UPDATE SET
			total_requests = routing_configurations.total_requests + EXCLUDED.total_requests,
			successful_requests = routing_configurations.successful_requests + EXCLUDED.successful_requests,
			failed_requests = routing_configurations.failed_requests + EXCLUDED.failed_requests,
			average_latency_ms = CASE 
				WHEN routing_configurations.total_requests = 0 THEN EXCLUDED.average_latency_ms
				ELSE ((routing_configurations.average_latency_ms * routing_configurations.total_requests) + 
					  (EXCLUDED.average_latency_ms * EXCLUDED.total_requests)) / 
					 (routing_configurations.total_requests + EXCLUDED.total_requests)
			END,
			updated_at = NOW()
	`
	
	_, err := mt.db.ExecContext(ctx, query,
		metrics.TenantID,
		metrics.TotalRequests,
		metrics.SuccessfulRequests,
		metrics.FailedRequests,
		averageLatencyMs,
	)
	if err != nil {
		return fmt.Errorf("failed to create routing config with metrics: %w", err)
	}
	
	return nil
}

// GetMetrics returns current metrics for a tenant (from cache + database)
func (mt *MetricsTracker) GetMetrics(ctx context.Context, tenantID uuid.UUID) (*RoutingMetrics, error) {
	// Get cached metrics
	mt.mu.RLock()
	cachedMetrics, hasCached := mt.metricsCache[tenantID]
	mt.mu.RUnlock()
	
	// Get database metrics
	query := `
		SELECT 
			total_requests, successful_requests, failed_requests, average_latency_ms
		FROM routing_configurations
		WHERE tenant_id = $1
	`
	
	var dbMetrics struct {
		TotalRequests      int64 `db:"total_requests"`
		SuccessfulRequests int64 `db:"successful_requests"`
		FailedRequests     int64 `db:"failed_requests"`
		AverageLatencyMs   *int  `db:"average_latency_ms"`
	}
	
	err := mt.db.GetContext(ctx, &dbMetrics, query, tenantID)
	if err != nil {
		// If no config exists, return cached metrics only
		if hasCached {
			totalRequests := atomic.LoadInt64(&cachedMetrics.TotalRequests)
			successfulRequests := atomic.LoadInt64(&cachedMetrics.SuccessfulRequests)
			failedRequests := atomic.LoadInt64(&cachedMetrics.FailedRequests)
			totalLatencyMs := atomic.LoadInt64(&cachedMetrics.TotalLatencyMs)
			
			var avgLatency *int
			if totalRequests > 0 {
				avg := int(totalLatencyMs / totalRequests)
				avgLatency = &avg
			}
			
			return &RoutingMetrics{
				TenantID:           tenantID,
				TotalRequests:      totalRequests,
				SuccessfulRequests: successfulRequests,
				FailedRequests:     failedRequests,
				AverageLatencyMs:   avgLatency,
			}, nil
		}
		return nil, fmt.Errorf("failed to get routing metrics: %w", err)
	}
	
	// Combine database and cached metrics
	totalRequests := dbMetrics.TotalRequests
	successfulRequests := dbMetrics.SuccessfulRequests
	failedRequests := dbMetrics.FailedRequests
	
	if hasCached {
		totalRequests += atomic.LoadInt64(&cachedMetrics.TotalRequests)
		successfulRequests += atomic.LoadInt64(&cachedMetrics.SuccessfulRequests)
		failedRequests += atomic.LoadInt64(&cachedMetrics.FailedRequests)
	}
	
	metrics := &RoutingMetrics{
		TenantID:           tenantID,
		TotalRequests:      totalRequests,
		SuccessfulRequests: successfulRequests,
		FailedRequests:     failedRequests,
		AverageLatencyMs:   dbMetrics.AverageLatencyMs,
	}
	
	// Calculate success rate
	if totalRequests > 0 {
		successRate := (float64(successfulRequests) / float64(totalRequests)) * 100.0
		metrics.SuccessRate = &successRate
	}
	
	return metrics, nil
}

// ResetMetrics resets metrics for a tenant (useful for testing)
func (mt *MetricsTracker) ResetMetrics(ctx context.Context, tenantID uuid.UUID) error {
	// Clear cache
	mt.mu.Lock()
	delete(mt.metricsCache, tenantID)
	mt.mu.Unlock()
	
	// Reset database metrics
	query := `
		UPDATE routing_configurations
		SET 
			total_requests = 0,
			successful_requests = 0,
			failed_requests = 0,
			average_latency_ms = NULL,
			updated_at = NOW()
		WHERE tenant_id = $1
	`
	
	_, err := mt.db.ExecContext(ctx, query, tenantID)
	if err != nil {
		return fmt.Errorf("failed to reset routing metrics: %w", err)
	}
	
	return nil
}
