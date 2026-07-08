package worker

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/al-hudhud/backend/internal/alerting"
	"github.com/al-hudhud/backend/internal/balance"
	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/redis"
	"github.com/al-hudhud/backend/internal/ws"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// FastMessageProcessor processes messages from Redis queue with ultra-high performance
type FastMessageProcessor struct {
	messageQueue   *redis.MessageQueue
	channelCache   *redis.ChannelCache
	balanceManager balance.Manager
	wsHub          *ws.Hub
	alertManager   *alerting.Manager
	logger         *zap.Logger
	batchSize      int
	workerCount    int
	stopChan       chan struct{}
	
	// Performance metrics
	messagesProcessed uint64
	totalLatencyMs    uint64
	metricsStartTime  time.Time
}

// NewFastMessageProcessor creates a new fast processor
func NewFastMessageProcessor(
	messageQueue *redis.MessageQueue,
	channelCache *redis.ChannelCache,
	balanceManager balance.Manager,
	wsHub *ws.Hub,
	alertManager *alerting.Manager,
	logger *zap.Logger,
	batchSize int,
	workerCount int,
) *FastMessageProcessor {
	if logger == nil {
		panic("logger is required")
	}
	if messageQueue == nil {
		panic("messageQueue is required")
	}
	if balanceManager == nil {
		panic("balanceManager is required")
	}
	// wsHub is optional - log warning if nil
	if wsHub == nil {
		logger.Warn("WebSocket hub not provided - messages will not be sent to devices")
	}
	// alertManager is optional - log warning if nil
	if alertManager == nil {
		logger.Warn("Alert manager not provided - system alerts will not be sent")
	}

	// Set defaults if invalid values provided
	if batchSize <= 0 {
		batchSize = 50 // New default based on benchmarks
	}
	if workerCount <= 0 {
		workerCount = 20 // New default based on benchmarks
	}

	return &FastMessageProcessor{
		messageQueue:      messageQueue,
		channelCache:      channelCache,
		balanceManager:    balanceManager,
		wsHub:             wsHub,
		alertManager:      alertManager,
		logger:            logger,
		batchSize:         batchSize,
		workerCount:       workerCount,
		stopChan:          make(chan struct{}),
		messagesProcessed: 0,
		totalLatencyMs:    0,
		metricsStartTime:  time.Now(),
	}
}

// Start begins processing messages from Redis queue
func (fp *FastMessageProcessor) Start(ctx context.Context) {
	fp.logger.Info("Starting Fast Message Processor",
		zap.Int("batch_size", fp.batchSize),
		zap.Int("worker_count", fp.workerCount),
	)

	// Start multiple workers for parallel processing
	for i := 0; i < fp.workerCount; i++ {
		go fp.worker(ctx, i)
	}
	
	// Start metrics logger (logs every 30 seconds)
	go fp.logMetricsPeriodically(ctx, 30*time.Second)
}

// Stop gracefully stops the processor
func (fp *FastMessageProcessor) Stop() {
	close(fp.stopChan)
	fp.logger.Info("Fast Message Processor stopped")
}

// worker processes messages continuously
func (fp *FastMessageProcessor) worker(ctx context.Context, workerID int) {
	fp.logger.Info("Fast processor worker started", zap.Int("worker_id", workerID))

	ticker := time.NewTicker(100 * time.Millisecond) // Check every 100ms
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-fp.stopChan:
			return
		case <-ticker.C:
			fp.processBatch(ctx, workerID)
		}
	}
}

// processBatch processes a batch of messages from all tenants
func (fp *FastMessageProcessor) processBatch(ctx context.Context, workerID int) {
	// Get list of active tenants from Redis
	activeTenants, err := fp.getActiveTenants(ctx)
	if err != nil {
		return
	}

	for _, tenantID := range activeTenants {
		// Dequeue batch of messages for this tenant
		messages, err := fp.messageQueue.DequeueBatch(ctx, tenantID, fp.batchSize)
		if err != nil || len(messages) == 0 {
			continue
		}

		fp.logger.Debug("Processing message batch",
			zap.Int("worker_id", workerID),
			zap.String("tenant_id", tenantID.String()),
			zap.Int("count", len(messages)),
		)

		// Process batch with single transaction
		if err := fp.processBatchWithTransaction(ctx, tenantID, messages); err != nil {
			fp.logger.Error("Failed to process batch",
				zap.Int("worker_id", workerID),
				zap.String("tenant_id", tenantID.String()),
				zap.Error(err),
			)
			// Re-queue failed messages
			for _, msg := range messages {
				_ = fp.messageQueue.Enqueue(ctx, msg)
			}
		}

		// Check if queue is empty and remove tenant from active set if so
		queueSize, err := fp.messageQueue.GetQueueSize(ctx, tenantID)
		if err == nil && queueSize == 0 {
			if err := fp.messageQueue.RemoveFromActiveSet(ctx, tenantID); err != nil {
				fp.logger.Warn("Failed to remove tenant from active set",
					zap.String("tenant_id", tenantID.String()),
					zap.Error(err),
				)
			}
		}
	}
}

// processBatchWithTransaction processes multiple messages in a single transaction
func (fp *FastMessageProcessor) processBatchWithTransaction(ctx context.Context, tenantID uuid.UUID, messages []*redis.MessageQueueItem) error {
	startTime := time.Now()

	// Get or select channel (cached)
	channel, err := fp.getOrSelectChannel(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("select channel: %w", err)
	}

	// Start unified transaction
	tx, err := db.PgxPool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	// Proper defer pattern: rollback only if commit not called
	committed := false
	defer func() {
		if !committed {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				fp.logger.Error("Failed to rollback transaction", zap.Error(rbErr))
			}
		}
	}()

	// Process all messages in batch
	var totalCost decimal.Decimal
	messageIDs := make([]uuid.UUID, 0, len(messages))

	for _, msg := range messages {
		totalCost = totalCost.Add(decimal.NewFromFloat(msg.Cost))
		messageIDs = append(messageIDs, msg.MessageID)
	}

	// Single balance deduction for entire batch
	balanceOp := balance.BalanceOperation{
		TenantID:  tenantID,
		GatewayID: channel.GatewayID,
		Amount:    totalCost,
		Reason:    fmt.Sprintf("BATCH_SEND_%d_MESSAGES", len(messages)),
		CreatedBy: "FAST_PROCESSOR",
	}

	_, err = fp.balanceManager.DeductBalanceTx(ctx, tx, balanceOp)
	if err != nil {
		// Update all messages to FAILED
		fp.updateBatchStatus(ctx, tx, messageIDs, "FAILED")
		tx.Commit(ctx)
		return fmt.Errorf("deduct balance: %w", err)
	}

	// Update all messages to PROCESSING in single query
	if err := fp.updateBatchStatus(ctx, tx, messageIDs, "PROCESSING"); err != nil {
		return fmt.Errorf("update batch status: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	committed = true

	// Send all messages via WebSocket (outside transaction)
	successCount := 0
	failedMessages := make([]*redis.MessageQueueItem, 0)

	for _, msg := range messages {
		wsMessage := ws.WebSocketMessage{
			Type:        "MESSAGE",
			MessageID:   msg.MessageID.String(),
			TenantID:    tenantID,
			ChannelID:   channel.ChannelID,
			PhoneNumber: msg.PhoneNumber,
			Content:     msg.Content,
			Timestamp:   time.Now(),
		}

		// Add nil check before calling wsHub methods
		if fp.wsHub == nil {
			fp.logger.Warn("WebSocket hub not initialized, message cannot be sent",
				zap.String("message_id", msg.MessageID.String()),
			)
			failedMessages = append(failedMessages, msg)
			continue
		}

		if err := fp.wsHub.SendMessage(ctx, wsMessage); err != nil {
			fp.logger.Error("Failed to send message via WebSocket",
				zap.String("message_id", msg.MessageID.String()),
				zap.Error(err),
			)
			failedMessages = append(failedMessages, msg)
		} else {
			successCount++
		}
	}

	// Handle failed messages (refund and re-queue)
	if len(failedMessages) > 0 {
		go fp.handleFailedMessages(context.Background(), tenantID, failedMessages)
	}

	processingTime := time.Since(startTime)
	
	// Update metrics atomically
	atomic.AddUint64(&fp.messagesProcessed, uint64(len(messages)))
	atomic.AddUint64(&fp.totalLatencyMs, uint64(processingTime.Milliseconds()))
	
	fp.logger.Info("Batch processed successfully",
		zap.String("tenant_id", tenantID.String()),
		zap.Int("total", len(messages)),
		zap.Int("success", successCount),
		zap.Int("failed", len(failedMessages)),
		zap.Duration("processing_time", processingTime),
		zap.Float64("messages_per_second", float64(len(messages))/processingTime.Seconds()),
	)

	return nil
}

// updateBatchStatus updates status for multiple messages in single query
func (fp *FastMessageProcessor) updateBatchStatus(ctx context.Context, tx pgx.Tx, messageIDs []uuid.UUID, status string) error {
	if len(messageIDs) == 0 {
		return nil
	}

	query := `
		UPDATE messages 
		SET status = $1, updated_at = NOW()
		WHERE id = ANY($2)
	`

	_, err := tx.Exec(ctx, query, status, messageIDs)
	return err
}

// getOrSelectChannel gets cached channel or selects new one
func (fp *FastMessageProcessor) getOrSelectChannel(ctx context.Context, tenantID uuid.UUID) (*redis.CachedChannel, error) {
	// Try cache first
	if fp.channelCache != nil {
		cached, err := fp.channelCache.Get(ctx, tenantID)
		if err == nil && cached.IsOnline {
			return cached, nil
		}
	}

	// Query database for available channels
	query := `
		SELECT c.id, c.type, c.priority, c.current_load, c.gateway_id
		FROM dispatch_channels c
		WHERE c.tenant_id = $1 
		  AND c.status = 'ONLINE'
		  AND c.deleted_at IS NULL
		ORDER BY c.priority ASC, c.current_load ASC
		LIMIT 1
	`
	
	var channelData struct {
		ID          uuid.UUID
		Type        string
		Priority    int
		CurrentLoad int
		GatewayID   *uuid.UUID
	}
	
	err := db.PgxPool.QueryRow(ctx, query, tenantID).Scan(
		&channelData.ID,
		&channelData.Type,
		&channelData.Priority,
		&channelData.CurrentLoad,
		&channelData.GatewayID,
	)
	
	if err != nil {
		return nil, fmt.Errorf("no available channels for tenant %s: %w", tenantID, err)
	}
	
	cachedChannel := &redis.CachedChannel{
		ChannelID: channelData.ID,
		GatewayID: channelData.GatewayID,
		Type:      channelData.Type,
		Priority:  channelData.Priority,
		IsOnline:  true,
		CachedAt:  time.Now(),
	}
	
	// Cache it
	if fp.channelCache != nil {
		if err := fp.channelCache.Set(ctx, tenantID, cachedChannel); err != nil {
			fp.logger.Warn("Failed to cache channel",
				zap.String("tenant_id", tenantID.String()),
				zap.Error(err),
			)
			// Continue - cache failure is not critical
		}
	}
	
	return cachedChannel, nil
}

// handleFailedMessages refunds balance and re-queues failed messages
func (fp *FastMessageProcessor) handleFailedMessages(ctx context.Context, tenantID uuid.UUID, messages []*redis.MessageQueueItem) {
	messageIDs := make([]uuid.UUID, 0, len(messages))

	for _, msg := range messages {
		messageIDs = append(messageIDs, msg.MessageID)

		// Refund each failed message individually to respect database constraints and track accounting properly
		refundOp := balance.BalanceOperation{
			TenantID:  tenantID,
			MessageID: &msg.MessageID,
			Amount:    decimal.NewFromFloat(msg.Cost),
			Reason:    "WEBSOCKET_FAILURE_REFUND",
			CreatedBy: "SYSTEM",
		}

		if _, err := fp.balanceManager.RefundBalance(ctx, refundOp); err != nil {
			fp.logger.Error("Failed to refund balance for failed message",
				zap.String("tenant_id", tenantID.String()),
				zap.String("message_id", msg.MessageID.String()),
				zap.Error(err),
			)
		}
	}

	// Update messages to FAILED
	if len(messageIDs) > 0 {
		query := `
			UPDATE messages 
			SET status = 'FAILED', updated_at = NOW()
			WHERE id = ANY($1)
		`
		if _, err := db.PgxPool.Exec(ctx, query, messageIDs); err != nil {
			fp.logger.Error("Failed to update failed messages status",
				zap.Int("message_count", len(messageIDs)),
				zap.Error(err),
			)
		}
	}
}

// getActiveTenants retrieves list of active tenants from Redis
func (fp *FastMessageProcessor) getActiveTenants(ctx context.Context) ([]uuid.UUID, error) {
	members, err := fp.messageQueue.GetActiveTenants(ctx)
	if err != nil {
		return nil, fmt.Errorf("get active tenants: %w", err)
	}

	tenants := make([]uuid.UUID, 0, len(members))
	for _, member := range members {
		tenantID, err := uuid.Parse(member)
		if err != nil {
			fp.logger.Warn("Invalid tenant UUID in active set", zap.String("value", member))
			continue
		}
		tenants = append(tenants, tenantID)
	}

	return tenants, nil
}

// GetStats returns processor statistics
func (fp *FastMessageProcessor) GetStats(ctx context.Context, tenantID uuid.UUID) (map[string]interface{}, error) {
	if fp.messageQueue == nil {
		return nil, fmt.Errorf("message queue not initialized")
	}

	return fp.messageQueue.GetQueueStats(ctx, tenantID)
}

// logMetricsPeriodically logs performance metrics at regular intervals
func (fp *FastMessageProcessor) logMetricsPeriodically(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-fp.stopChan:
			return
		case <-ticker.C:
			fp.logMetrics()
		}
	}
}

// logMetrics logs current performance metrics
func (fp *FastMessageProcessor) logMetrics() {
	messagesProcessed := atomic.LoadUint64(&fp.messagesProcessed)
	totalLatencyMs := atomic.LoadUint64(&fp.totalLatencyMs)
	
	if messagesProcessed == 0 {
		fp.logger.Info("Fast Message Processor Metrics",
			zap.Uint64("messages_processed_total", 0),
			zap.Float64("throughput_msg_per_sec", 0.0),
			zap.Float64("avg_latency_ms", 0.0),
		)
		return
	}
	
	elapsedTime := time.Since(fp.metricsStartTime).Seconds()
	throughput := float64(messagesProcessed) / elapsedTime
	avgLatency := float64(totalLatencyMs) / float64(messagesProcessed)
	
	fp.logger.Info("Fast Message Processor Metrics",
		zap.Uint64("messages_processed_total", messagesProcessed),
		zap.Float64("throughput_msg_per_sec", throughput),
		zap.Float64("avg_latency_ms", avgLatency),
		zap.Int("batch_size", fp.batchSize),
		zap.Int("worker_count", fp.workerCount),
		zap.Duration("uptime", time.Since(fp.metricsStartTime)),
	)
}

// MonitorQueueHealth monitors queue sizes and sends alerts when thresholds are exceeded
// This implements Bug 24 requirement: Monitor queue backup
func (fp *FastMessageProcessor) MonitorQueueHealth(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute) // Check every minute
	defer ticker.Stop()
	
	fp.logger.Info("Queue health monitoring started")
	
	for {
		select {
		case <-ctx.Done():
			fp.logger.Info("Queue health monitoring stopped")
			return
		case <-fp.stopChan:
			fp.logger.Info("Queue health monitoring stopped")
			return
		case <-ticker.C:
			fp.checkQueueBackup(ctx)
		}
	}
}

// checkQueueBackup checks queue sizes for all active tenants and sends alerts
func (fp *FastMessageProcessor) checkQueueBackup(ctx context.Context) {
	if fp.alertManager == nil {
		return
	}
	
	activeTenants, err := fp.getActiveTenants(ctx)
	if err != nil {
		fp.logger.Error("Failed to get active tenants for queue monitoring", zap.Error(err))
		return
	}
	
	for _, tenantID := range activeTenants {
		queueSize, err := fp.messageQueue.GetQueueSize(ctx, tenantID)
		if err != nil {
			fp.logger.Error("Failed to get queue size",
				zap.String("tenant_id", tenantID.String()),
				zap.Error(err),
			)
			continue
		}
		
		// Critical alert if queue is backing up significantly
		if queueSize > 10000 {
			alert := alerting.NewAlert(
				alerting.AlertTypeQueueBackup,
				alerting.AlertLevelCritical,
				"Critical: Queue Backup Detected",
				fmt.Sprintf("Tenant %s has %d messages in queue (threshold: 10000)", tenantID.String(), queueSize),
				"fast_processor",
			).WithTenantID(tenantID).
				WithMetadata("queue_size", queueSize).
				WithMetadata("threshold", 10000)
			
			fp.alertManager.SendAlertAsync(alert)
			
			fp.logger.Warn("Critical queue backup detected",
				zap.String("tenant_id", tenantID.String()),
				zap.Int64("queue_size", queueSize),
			)
		} else if queueSize > 5000 {
			// Warning alert for high queue size
			alert := alerting.NewAlert(
				alerting.AlertTypeQueueBackup,
				alerting.AlertLevelWarning,
				"Warning: High Queue Size",
				fmt.Sprintf("Tenant %s has %d messages in queue (threshold: 5000)", tenantID.String(), queueSize),
				"fast_processor",
			).WithTenantID(tenantID).
				WithMetadata("queue_size", queueSize).
				WithMetadata("threshold", 5000)
			
			fp.alertManager.SendAlertAsync(alert)
			
			fp.logger.Info("High queue size detected",
				zap.String("tenant_id", tenantID.String()),
				zap.Int64("queue_size", queueSize),
			)
		}
	}
}

// MonitorErrorRates monitors processing error rates and sends alerts when thresholds are exceeded
// This implements Bug 24 requirement: Monitor error rates
func (fp *FastMessageProcessor) MonitorErrorRates(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute) // Check every 5 minutes
	defer ticker.Stop()
	
	fp.logger.Info("Error rate monitoring started")
	
	// Track error counts per tenant
	errorCounts := make(map[uuid.UUID]int64)
	successCounts := make(map[uuid.UUID]int64)
	var mu sync.RWMutex
	
	// Reset counters periodically
	resetTicker := time.NewTicker(15 * time.Minute)
	defer resetTicker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			fp.logger.Info("Error rate monitoring stopped")
			return
		case <-fp.stopChan:
			fp.logger.Info("Error rate monitoring stopped")
			return
		case <-resetTicker.C:
			// Reset counters
			mu.Lock()
			errorCounts = make(map[uuid.UUID]int64)
			successCounts = make(map[uuid.UUID]int64)
			mu.Unlock()
		case <-ticker.C:
			fp.checkErrorRates(ctx, &errorCounts, &successCounts, &mu)
		}
	}
}

// checkErrorRates checks error rates from database and sends alerts
func (fp *FastMessageProcessor) checkErrorRates(ctx context.Context, errorCounts *map[uuid.UUID]int64, successCounts *map[uuid.UUID]int64, mu *sync.RWMutex) {
	if fp.alertManager == nil {
		return
	}
	
	// Query recent message statuses (last 5 minutes)
	query := `
		SELECT 
			m.tenant_id,
			m.status,
			COUNT(*) as count
		FROM messages m
		WHERE m.updated_at > NOW() - INTERVAL '5 minutes'
		GROUP BY m.tenant_id, m.status
	`
	
	rows, err := db.PgxPool.Query(ctx, query)
	if err != nil {
		fp.logger.Error("Failed to query message statuses for error rate monitoring", zap.Error(err))
		return
	}
	defer rows.Close()
	
	// Aggregate counts per tenant
	tenantStats := make(map[uuid.UUID]struct {
		success int64
		failed  int64
	})
	
	for rows.Next() {
		var tenantID uuid.UUID
		var status string
		var count int64
		
		if err := rows.Scan(&tenantID, &status, &count); err != nil {
			fp.logger.Error("Failed to scan message status row", zap.Error(err))
			continue
		}
		
		stats := tenantStats[tenantID]
		if status == "SENT" || status == "DELIVERED" {
			stats.success += count
		} else if status == "FAILED" {
			stats.failed += count
		}
		tenantStats[tenantID] = stats
	}
	
	// Check error rates and send alerts
	for tenantID, stats := range tenantStats {
		total := stats.success + stats.failed
		if total == 0 {
			continue
		}
		
		errorRate := float64(stats.failed) / float64(total) * 100.0
		
		// Critical alert if error rate > 50%
		if errorRate > 50.0 {
			alert := alerting.NewAlert(
				alerting.AlertTypeHighErrorRate,
				alerting.AlertLevelCritical,
				"Critical: High Error Rate",
				fmt.Sprintf("Tenant %s has %.1f%% error rate (%d failed / %d total)", 
					tenantID.String(), errorRate, stats.failed, total),
				"fast_processor",
			).WithTenantID(tenantID).
				WithMetadata("error_rate", fmt.Sprintf("%.1f%%", errorRate)).
				WithMetadata("failed_count", stats.failed).
				WithMetadata("total_count", total)
			
			fp.alertManager.SendAlertAsync(alert)
			
			fp.logger.Warn("Critical error rate detected",
				zap.String("tenant_id", tenantID.String()),
				zap.Float64("error_rate", errorRate),
				zap.Int64("failed", stats.failed),
				zap.Int64("total", total),
			)
		} else if errorRate > 25.0 {
			// Warning alert if error rate > 25%
			alert := alerting.NewAlert(
				alerting.AlertTypeHighErrorRate,
				alerting.AlertLevelWarning,
				"Warning: Elevated Error Rate",
				fmt.Sprintf("Tenant %s has %.1f%% error rate (%d failed / %d total)", 
					tenantID.String(), errorRate, stats.failed, total),
				"fast_processor",
			).WithTenantID(tenantID).
				WithMetadata("error_rate", fmt.Sprintf("%.1f%%", errorRate)).
				WithMetadata("failed_count", stats.failed).
				WithMetadata("total_count", total)
			
			fp.alertManager.SendAlertAsync(alert)
			
			fp.logger.Info("Elevated error rate detected",
				zap.String("tenant_id", tenantID.String()),
				zap.Float64("error_rate", errorRate),
				zap.Int64("failed", stats.failed),
				zap.Int64("total", total),
			)
		}
	}
}
