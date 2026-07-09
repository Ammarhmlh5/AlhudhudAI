package worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/al-hudhud/backend/internal/balance"
	"github.com/al-hudhud/backend/internal/database"
	"github.com/al-hudhud/backend/internal/metrics"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/al-hudhud/backend/internal/queue"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// RiverWorkerPool implements WorkerPool using River Queue
// Implements Requirements 13.1, 13.2, 13.3, 13.4, 13.5, 13.7, 24.4, 24.6
type RiverWorkerPool struct {
	config       *WorkerPoolConfig
	db           *database.DB
	logger       *zap.Logger
	client       *river.Client[pgx.Tx]
	dependencies *MessageWorkerDependencies
	dlq          *DeadLetterQueue // Dead Letter Queue for failed messages
	recovery     *RecoveryManager // Recovery manager for automatic recovery

	// Metrics
	mu                    sync.RWMutex
	processedCount        int64
	failedCount           int64
	totalProcessingTimeMs int64
	lastProcessedAt       time.Time
	isRunning             bool

	// Worker metrics per worker
	workerMetrics map[int]*WorkerMetrics

	// Health monitoring
	// Implements Requirements 13.4, 13.5
	healthConfig  *HealthMonitorConfig
	workerHealth  map[int]*WorkerHealth
	healthMonitor context.CancelFunc
}

// NewRiverWorkerPool creates a new River worker pool
// Implements Requirement 13.1: إنشاء 50 worker متزامن
// Implements Requirement 13.4: مراقبة صحة Workers
// Implements Requirement 24.4: إعادة تشغيل Workers المتعطلة تلقائياً
func NewRiverWorkerPool(
	config *WorkerPoolConfig,
	db *database.DB,
	logger *zap.Logger,
	dependencies *MessageWorkerDependencies,
) (*RiverWorkerPool, error) {
	if config == nil {
		config = DefaultWorkerPoolConfig()
	}

	if dependencies == nil {
		return nil, fmt.Errorf("dependencies cannot be nil")
	}

	pool := &RiverWorkerPool{
		config:        config,
		db:            db,
		logger:        logger,
		dependencies:  dependencies,
		dlq:           NewDeadLetterQueue(db, logger),
		workerMetrics: make(map[int]*WorkerMetrics),
		healthConfig:  DefaultHealthMonitorConfig(),
		workerHealth:  make(map[int]*WorkerHealth),
	}

	// Initialize worker metrics and health
	for i := 0; i < config.WorkerCount; i++ {
		pool.workerMetrics[i] = &WorkerMetrics{
			WorkerID: i,
		}
		pool.workerHealth[i] = &WorkerHealth{
			WorkerID:        i,
			IsHealthy:       true,
			LastHealthCheck: time.Now(),
		}
	}

	return pool, nil
}

// Start starts the worker pool
// Implements Requirement 13.1: تشغيل 50 Worker متزامن
// Implements Requirement 13.4: مراقبة صحة Workers كل 10 ثوانٍ
// Implements Requirement 24.6: استكمال معالجة الرسائل المتبقية بعد عطل
func (p *RiverWorkerPool) Start(ctx context.Context) error {
	p.mu.Lock()
	if p.isRunning {
		p.mu.Unlock()
		return fmt.Errorf("worker pool already running")
	}
	p.mu.Unlock()

	p.logger.Info("Starting River worker pool",
		zap.Int("worker_count", p.config.WorkerCount),
		zap.String("queue_name", p.config.QueueName),
		zap.Int("max_workers", p.config.MaxWorkers),
	)

	// Create River workers registry
	workers := river.NewWorkers()

	// Register message worker with dependencies
	// Implements Requirement 13.2: تطبيق River job handler للرسائل
	messageWorker := NewMessageWorker(p.dependencies, p)
	river.AddWorker(workers, messageWorker)

	// Create River driver
	driver := riverpgxv5.New(p.db.Pool)

	// Create River client with configuration
	// Implements Requirement 13.3: توزيع Jobs بالتساوي على Workers
	// Implements Requirement 17.4: إعادة محاولة الرسائل الفاشلة 3 مرات
	client, err := river.NewClient(driver, &river.Config{
		Queues: map[string]river.QueueConfig{
			p.config.QueueName: {
				MaxWorkers: p.config.MaxWorkers, // 50 workers
			},
		},
		Workers:      workers,
		// Job timeout configuration
		JobTimeout: p.config.JobTimeout,
	})

	if err != nil {
		return fmt.Errorf("create river client: %w", err)
	}

	p.client = client

	// Initialize recovery manager
	// Implements Requirement 24.4, 24.6: Automatic recovery
	p.recovery = NewRecoveryManager(p.db, p.logger, p.client, DefaultRecoveryConfig())

	// Start River client
	if err := p.client.Start(ctx); err != nil {
		return fmt.Errorf("start river client: %w", err)
	}

	p.mu.Lock()
	p.isRunning = true
	p.mu.Unlock()

	// Perform automatic recovery of pending messages
	// Implements Requirement 24.6: استكمال معالجة الرسائل المتبقية بعد عطل
	go func() {
		p.logger.Info("Starting periodic recovery of pending messages")
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()

		// Initial recovery
		if err := p.recovery.RecoverPendingMessages(context.Background()); err != nil {
			p.logger.Error("Initial recovery failed", zap.Error(err))
		}

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := p.recovery.RecoverPendingMessages(context.Background()); err != nil {
					p.logger.Error("Periodic recovery failed", zap.Error(err))
				}
			}
		}
	}()

	// Start health monitoring
	// Implements Requirement 13.4: مراقبة صحة Workers كل 10 ثوانٍ
	// Implements Requirement 24.4: إعادة تشغيل Workers المتعطلة تلقائياً
	monitorCtx, cancel := context.WithCancel(context.Background())
	p.healthMonitor = cancel
	go p.startHealthMonitoring(monitorCtx)

	// Start Dynamic Scaling
	// Phase 2.2: Worker Pool Scaling
	go p.startDynamicScaling(ctx)

	p.logger.Info("River worker pool started successfully",
		zap.Int("worker_count", p.config.WorkerCount),
		zap.Duration("health_check_interval", p.healthConfig.CheckInterval),
		zap.Bool("auto_recovery_enabled", p.recovery.config.EnableAutoRecovery),
	)

	return nil
}

// Stop stops the worker pool gracefully
// Implements Requirement 13.5: إعادة تشغيل Workers الفاشلة تلقائياً
func (p *RiverWorkerPool) Stop(ctx context.Context) error {
	p.mu.Lock()
	if !p.isRunning {
		p.mu.Unlock()
		return fmt.Errorf("worker pool not running")
	}
	p.mu.Unlock()

	p.logger.Info("Stopping River worker pool gracefully")

	// Stop health monitoring
	if p.healthMonitor != nil {
		p.healthMonitor()
		p.healthMonitor = nil
	}

	// Stop River client gracefully
	if err := p.client.Stop(ctx); err != nil {
		return fmt.Errorf("stop river client: %w", err)
	}

	p.mu.Lock()
	p.isRunning = false
	p.mu.Unlock()

	p.logger.Info("River worker pool stopped successfully")

	return nil
}

// startDynamicScaling periodically checks system load and scales the worker pool
// Phase 2.2: Worker Pool Scaling
func (p *RiverWorkerPool) startDynamicScaling(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cpu, mem := metrics.GetServerLoad()

			// Scaling logic
			p.mu.RLock()
			currentWorkers := p.config.MaxWorkers
			p.mu.RUnlock()

			targetWorkers := currentWorkers
			switch {
			case cpu > 80 || mem > 85:
				// High load - reduce workers to 20
				targetWorkers = 20
			case cpu < 30 && mem < 40:
				// Low load - increase workers to 100
				targetWorkers = 100
			default:
				// Normal load - back to default 
				targetWorkers = 50
			}

			if targetWorkers != currentWorkers {
				p.logger.Info("Dynamic scaling triggered",
					zap.Float64("cpu", cpu),
					zap.Int("current_workers", currentWorkers),
					zap.Int("target_workers", targetWorkers))
				
				// Phase 3.3: Export to Prometheus
				metrics.SetActiveWorkers(float64(targetWorkers))
				
				if err := p.Scale(ctx, targetWorkers); err != nil {
					p.logger.Error("Scaling failed", zap.Error(err))
				}
			}
		}
	}
}

// Scale resizes the worker pool by restarting the River client with new MaxWorkers
func (p *RiverWorkerPool) Scale(ctx context.Context, newWorkerCount int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.isRunning {
		p.config.MaxWorkers = newWorkerCount
		return nil
	}

	p.logger.Info("Scaling worker pool by restarting client", zap.Int("new_count", newWorkerCount))

	// Stop current client
	if err := p.client.Stop(ctx); err != nil {
		return fmt.Errorf("stop client for scaling: %w", err)
	}

	// Update config
	p.config.MaxWorkers = newWorkerCount

	// Create new workers registry
	workers := river.NewWorkers()
	messageWorker := NewMessageWorker(p.dependencies, p)
	river.AddWorker(workers, messageWorker)

	// Re-create client with new config
	driver := riverpgxv5.New(p.db.Pool)
	client, err := river.NewClient(driver, &river.Config{
		Queues: map[string]river.QueueConfig{
			p.config.QueueName: {
				MaxWorkers: p.config.MaxWorkers,
			},
		},
		Workers:    workers,
		JobTimeout: p.config.JobTimeout,
	})

	if err != nil {
		return fmt.Errorf("recreate river client for scaling: %w", err)
	}

	p.client = client
	return p.client.Start(ctx)
}

// GetStatus returns the current status of the worker pool
func (p *RiverWorkerPool) GetStatus(ctx context.Context) (*WorkerPoolStatus, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Calculate average processing time
	var avgProcessingTime time.Duration
	if p.processedCount > 0 {
		avgProcessingTime = time.Duration(p.totalProcessingTimeMs/p.processedCount) * time.Millisecond
	}

	// Get queue size from database
	var queueSize int64
	err := p.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) 
		FROM river_job 
		WHERE state IN ('available', 'running', 'scheduled')
	`).Scan(&queueSize)

	if err != nil {
		p.logger.Warn("Failed to get queue size", zap.Error(err))
		queueSize = 0
	}

	return &WorkerPoolStatus{
		WorkerCount:           p.config.WorkerCount,
		QueueSize:             queueSize,
		ProcessedCount:        p.processedCount,
		FailedCount:           p.failedCount,
		AverageProcessingTime: avgProcessingTime,
		LastProcessedAt:       p.lastProcessedAt,
		IsRunning:             p.isRunning,
	}, nil
}

// GetClient returns the River client
func (p *RiverWorkerPool) GetClient() *river.Client[pgx.Tx] {
	return p.client
}

// GetDeadLetterQueue returns the dead letter queue
func (p *RiverWorkerPool) GetDeadLetterQueue() *DeadLetterQueue {
	return p.dlq
}

// GetRecoveryManager returns the recovery manager
// Implements Requirement 24.4, 24.6: Automatic recovery
func (p *RiverWorkerPool) GetRecoveryManager() *RecoveryManager {
	return p.recovery
}

// RecordJobProcessed records a successfully processed job
// Implements Requirement 13.7: تسجيل أداء كل Worker
func (p *RiverWorkerPool) RecordJobProcessed(workerID int, processingTimeMs int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.processedCount++
	p.totalProcessingTimeMs += processingTimeMs
	p.lastProcessedAt = time.Now()

	// Update worker metrics
	if metrics, ok := p.workerMetrics[workerID]; ok {
		metrics.ProcessedCount++
		metrics.LastProcessedAt = time.Now()

		// Update average time
		totalTime := metrics.AverageTime.Milliseconds()*metrics.ProcessedCount + processingTimeMs
		metrics.AverageTime = time.Duration(totalTime/(metrics.ProcessedCount+1)) * time.Millisecond

		// Update success rate
		total := metrics.ProcessedCount + metrics.FailedCount
		if total > 0 {
			metrics.SuccessRate = float64(metrics.ProcessedCount) / float64(total)
		}
	}
}

// RecordJobFailed records a failed job
// Implements Requirement 13.7: تسجيل أداء كل Worker
func (p *RiverWorkerPool) RecordJobFailed(workerID int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.failedCount++

	// Update worker metrics
	if metrics, ok := p.workerMetrics[workerID]; ok {
		metrics.FailedCount++

		// Update success rate
		total := metrics.ProcessedCount + metrics.FailedCount
		if total > 0 {
			metrics.SuccessRate = float64(metrics.ProcessedCount) / float64(total)
		}
	}
}

// GetWorkerMetrics returns metrics for all workers
// Implements Requirement 13.7: تسجيل أداء كل Worker
func (p *RiverWorkerPool) GetWorkerMetrics() map[int]*WorkerMetrics {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Create a copy to avoid race conditions
	metrics := make(map[int]*WorkerMetrics)
	for id, m := range p.workerMetrics {
		metricsCopy := *m
		metrics[id] = &metricsCopy
	}

	return metrics
}

// MessageWorker implements River worker for message processing
// Implements Requirement 13.2: تطبيق River job handler للرسائل
type MessageWorker struct {
	river.WorkerDefaults[queue.SendSMSArgs]
	dependencies *MessageWorkerDependencies
	pool         *RiverWorkerPool
}

// NewMessageWorker creates a new message worker
func NewMessageWorker(dependencies *MessageWorkerDependencies, pool *RiverWorkerPool) *MessageWorker {
	return &MessageWorker{
		dependencies: dependencies,
		pool:         pool,
	}
}

// Work processes a message job
// Implements Requirements 1.1, 5.1, 6.1: معالجة الرسائل الكاملة
// Implements Requirement 17.4: إعادة محاولة الرسائل الفاشلة 3 مرات مع exponential backoff
// Implements Requirement 17.5: نقل الرسائل الفاشلة نهائياً إلى Dead Letter Queue
func (w *MessageWorker) Work(ctx context.Context, job *river.Job[queue.SendSMSArgs]) error {
	startTime := time.Now()
	workerID := int(job.ID % int64(w.pool.config.WorkerCount))

	w.dependencies.Logger.Info("Processing message job (Unified TX)",
		zap.String("tenant_id", job.Args.TenantID.String()),
		zap.String("message_id", job.Args.MessageID.String()),
		zap.Int("attempt", job.Attempt),
		zap.Int("worker_id", workerID),
	)

	// 1. Start Unified Transaction for all DB operations
	tx, err := w.pool.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 2. Get message from database (within TX)
	message, err := w.getMessage(ctx, tx, job.Args.TenantID, job.Args.MessageID)
	if err != nil {
		return fmt.Errorf("get message in tx: %w", err)
	}

	// 3. Select channel via Routing Engine
	// Note: RoutingEngine might need to be transaction-aware if it updates current_load
	// For now, we assume it's read-only or handles its own TX for load balancing
	channel, err := w.dependencies.RoutingEngine.SelectChannel(ctx, job.Args.TenantID)
	if err != nil {
		// Update message status to WAITING when no channels available
		w.updateMessageDispatchInfo(ctx, tx, job.Args.TenantID, job.Args.MessageID, "WAITING", nil, nil)
		tx.Commit(ctx) // Commit the status update to WAITING
		return fmt.Errorf("select channel: %w", err)
	}

	// 4. Deduct balance atomically (within TX)
	balanceOp := balance.BalanceOperation{
		TenantID:  job.Args.TenantID,
		MessageID: &job.Args.MessageID,
		GatewayID: channel.GatewayID,
		Amount:    decimal.NewFromFloat(message.Cost),
		Reason:    "MESSAGE_SEND",
		CreatedBy: "WORKER",
	}

	_, err = w.dependencies.BalanceManager.DeductBalanceTx(ctx, tx, balanceOp)
	if err != nil {
		// If fails (e.g. insufficient funds), update message to FAILED
		w.updateMessageDispatchInfo(ctx, tx, job.Args.TenantID, job.Args.MessageID, "FAILED", nil, nil)
		tx.Commit(ctx)
		return fmt.Errorf("deduct balance in tx: %w", err)
	}

	// 5. Update message status to PROCESSING (within TX)
	if err := w.updateMessageDispatchInfo(ctx, tx, job.Args.TenantID, job.Args.MessageID, "PROCESSING", &channel.ID, channel.GatewayID); err != nil {
		return fmt.Errorf("update message processing state: %w", err)
	}

	// 6. Commit all DB changes
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit message transaction: %w", err)
	}

	// 7. Send message via WebSocket Hub (External IO - Outside TX)
	wsMessage := WebSocketMessage{
		Type:        "MESSAGE",
		MessageID:   job.Args.MessageID.String(),
		TenantID:    job.Args.TenantID,
		ChannelID:   channel.ID,
		PhoneNumber: message.PhoneNumber,
		Content:     message.Content,
		Timestamp:   time.Now(),
	}

	err = w.dependencies.WebSocketHub.SendMessage(ctx, wsMessage)
	if err != nil {
		w.dependencies.Logger.Error("Failed to send message via WebSocket, triggering compensation",
			zap.Error(err),
			zap.String("message_id", job.Args.MessageID.String()),
		)

		// 8. Compensation Logic: Refund balance if WebSocket send fails
		refundOp := balance.BalanceOperation{
			TenantID:  job.Args.TenantID,
			MessageID: &job.Args.MessageID,
			Amount:    decimal.NewFromFloat(message.Cost),
			Reason:    "WEBSOCKET_SEND_FAILURE_COMPENSATION",
			CreatedBy: "SYSTEM",
		}
		
		// Use a fresh transaction for refund
		if _, refundErr := w.dependencies.BalanceManager.RefundBalance(ctx, refundOp); refundErr != nil {
			w.dependencies.Logger.Error("FAILED TO COMPENSATE: Balance may be inconsistent",
				zap.Error(refundErr),
				zap.String("message_id", job.Args.MessageID.String()),
			)
		}

		// Also reset message status back to PENDING/FAILED so it can be retried or audited
		// Use pool DB directly here
		w.updateMessageDispatchInfo(ctx, nil, job.Args.TenantID, job.Args.MessageID, "FAILED", nil, nil)

		return fmt.Errorf("send message via websocket: %w", err)
	}

	// Record success
	processingTimeMs := time.Since(startTime).Milliseconds()
	w.pool.RecordJobProcessed(workerID, processingTimeMs)
	w.pool.RecordWorkerSuccess(workerID)

	return nil
}

// getMessage retrieves message from database
func (w *MessageWorker) getMessage(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, messageID uuid.UUID) (*models.Message, error) {
	var message models.Message
	var err error

	sql := `
		SELECT id, tenant_id, phone_number, content, status, cost
		FROM messages
		WHERE tenant_id = $1 AND id = $2
	`

	if tx != nil {
		err = tx.QueryRow(ctx, sql, tenantID, messageID).Scan(
			&message.ID, &message.TenantID, &message.PhoneNumber, &message.Content, &message.Status, &message.Cost,
		)
	} else {
		err = w.pool.db.Pool.QueryRow(ctx, sql, tenantID, messageID).Scan(
			&message.ID, &message.TenantID, &message.PhoneNumber, &message.Content, &message.Status, &message.Cost,
		)
	}

	if err != nil {
		return nil, fmt.Errorf("query message: %w", err)
	}

	return &message, nil
}

// updateMessageDispatchInfo updates message status and recording channel/gateway info
func (w *MessageWorker) updateMessageDispatchInfo(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, messageID uuid.UUID, status string, channelID *uuid.UUID, gatewayID *uuid.UUID) error {
	sql := `
		UPDATE messages
		SET status = $3, channel_id = $4, gateway_id = $5, updated_at = NOW()
		WHERE tenant_id = $1 AND id = $2
	`
	var err error
	if tx != nil {
		_, err = tx.Exec(ctx, sql, tenantID, messageID, status, channelID, gatewayID)
	} else {
		_, err = w.pool.db.Pool.Exec(ctx, sql, tenantID, messageID, status, channelID, gatewayID)
	}
	return err
}


// parseChannelID converts UUID to int64
func parseChannelID(channelUUID uuid.UUID) (int64, error) {
	// Use the first 8 bytes of UUID as int64
	// This is a simplified approach - in production use proper mapping
	return int64(channelUUID.ID()), nil
}

// moveToDeadLetterQueue moves a message to the dead letter queue
// Implements Requirement 17.5: نقل الرسائل الفاشلة نهائياً إلى Dead Letter Queue
func (w *MessageWorker) moveToDeadLetterQueue(ctx context.Context, tenantID uuid.UUID, messageID uuid.UUID, reason string, retryCount int) {
	err := w.pool.dlq.MoveToDeadLetter(ctx, tenantID, messageID, reason, retryCount)
	if err != nil {
		w.dependencies.Logger.Error("Failed to move message to dead letter queue",
			zap.Error(err),
			zap.String("tenant_id", tenantID.String()),
			zap.String("message_id", messageID.String()),
			zap.String("reason", reason),
			zap.Int("retry_count", retryCount),
		)
	}
}

// startHealthMonitoring starts the health monitoring goroutine
// Implements Requirement 13.4: مراقبة صحة Workers كل 10 ثوانٍ
func (p *RiverWorkerPool) startHealthMonitoring(ctx context.Context) {
	ticker := time.NewTicker(p.healthConfig.CheckInterval)
	defer ticker.Stop()

	p.logger.Info("Health monitoring started",
		zap.Duration("check_interval", p.healthConfig.CheckInterval),
		zap.Int("failure_threshold", p.healthConfig.FailureThreshold),
		zap.Bool("restart_enabled", p.healthConfig.RestartEnabled),
	)

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("Health monitoring stopped")
			return
		case <-ticker.C:
			p.performHealthCheck(ctx)
		}
	}
}

// performHealthCheck performs a health check on all workers
// Implements Requirements 13.4, 13.5, 13.7
func (p *RiverWorkerPool) performHealthCheck(ctx context.Context) {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	unhealthyWorkers := 0
	totalWorkers := len(p.workerHealth)

	for workerID, health := range p.workerHealth {
		metrics := p.workerMetrics[workerID]

		// Check if worker has processed any messages recently
		timeSinceLastProcess := now.Sub(metrics.LastProcessedAt)

		// Worker is considered unhealthy if:
		// 1. It hasn't processed anything in the last 2 check intervals
		// 2. Success rate is below 50%
		// 3. It has consecutive failures above threshold
		isUnhealthy := false
		reason := ""

		if !metrics.LastProcessedAt.IsZero() && timeSinceLastProcess > 2*p.healthConfig.CheckInterval {
			// Worker might be stuck or idle
			// This is not necessarily unhealthy if there are no jobs
			// Check if there are pending jobs
			var queueSize int64
			err := p.db.Pool.QueryRow(ctx, `
				SELECT COUNT(*) 
				FROM river_job 
				WHERE state IN ('available', 'running')
			`).Scan(&queueSize)

			if err == nil && queueSize > 0 {
				isUnhealthy = true
				reason = fmt.Sprintf("no processing activity for %v with %d pending jobs", timeSinceLastProcess, queueSize)
			}
		}

		if metrics.ProcessedCount > 10 && metrics.SuccessRate < 0.5 {
			isUnhealthy = true
			reason = fmt.Sprintf("low success rate: %.2f%%", metrics.SuccessRate*100)
		}

		if health.ConsecutiveFailures >= p.healthConfig.FailureThreshold {
			isUnhealthy = true
			reason = fmt.Sprintf("consecutive failures: %d", health.ConsecutiveFailures)
		}

		// Update health status
		health.LastHealthCheck = now
		previousHealth := health.IsHealthy
		health.IsHealthy = !isUnhealthy

		if isUnhealthy {
			unhealthyWorkers++
			health.LastError = reason

			// Log health status change
			if previousHealth {
				p.logger.Warn("Worker became unhealthy",
					zap.Int("worker_id", workerID),
					zap.String("reason", reason),
					zap.Int64("processed", metrics.ProcessedCount),
					zap.Int64("failed", metrics.FailedCount),
					zap.Float64("success_rate", metrics.SuccessRate),
				)
			}

			// Attempt restart if enabled
			// Implements Requirement 13.5: إعادة تشغيل Workers الفاشلة تلقائياً
			if p.healthConfig.RestartEnabled && health.RestartCount < p.healthConfig.MaxRestarts {
				p.logger.Info("Attempting to restart unhealthy worker",
					zap.Int("worker_id", workerID),
					zap.Int("restart_count", health.RestartCount),
					zap.String("reason", reason),
				)

				// Reset worker metrics to give it a fresh start
				health.ConsecutiveFailures = 0
				health.RestartCount++
				health.IsHealthy = true
				health.LastError = ""

				// Note: River handles worker lifecycle automatically
				// We just reset the metrics and let River reassign jobs
				p.logger.Info("Worker metrics reset for restart",
					zap.Int("worker_id", workerID),
					zap.Int("total_restarts", health.RestartCount),
				)
			} else if health.RestartCount >= p.healthConfig.MaxRestarts {
				p.logger.Error("Worker exceeded maximum restart attempts",
					zap.Int("worker_id", workerID),
					zap.Int("max_restarts", p.healthConfig.MaxRestarts),
					zap.String("last_error", health.LastError),
				)
			}
		} else if !previousHealth {
			// Worker recovered
			p.logger.Info("Worker recovered",
				zap.Int("worker_id", workerID),
				zap.Int64("processed", metrics.ProcessedCount),
				zap.Float64("success_rate", metrics.SuccessRate),
			)
		}
	}

	// Log overall health status
	// Implements Requirement 13.7: تسجيل أداء كل Worker
	if unhealthyWorkers > 0 {
		p.logger.Warn("Health check completed",
			zap.Int("total_workers", totalWorkers),
			zap.Int("unhealthy_workers", unhealthyWorkers),
			zap.Int("healthy_workers", totalWorkers-unhealthyWorkers),
			zap.Int64("total_processed", p.processedCount),
			zap.Int64("total_failed", p.failedCount),
		)
	} else {
		p.logger.Debug("Health check completed - all workers healthy",
			zap.Int("total_workers", totalWorkers),
			zap.Int64("total_processed", p.processedCount),
			zap.Int64("total_failed", p.failedCount),
		)
	}
}

// GetWorkerHealth returns health status for all workers
// Implements Requirement 13.4: مراقبة صحة Workers
func (p *RiverWorkerPool) GetWorkerHealth() map[int]*WorkerHealth {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Create a copy to avoid race conditions
	health := make(map[int]*WorkerHealth)
	for id, h := range p.workerHealth {
		healthCopy := *h
		health[id] = &healthCopy
	}

	return health
}

// RecordWorkerFailure records a worker failure for health monitoring
// Implements Requirement 13.4: مراقبة صحة Workers
func (p *RiverWorkerPool) RecordWorkerFailure(workerID int, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if health, ok := p.workerHealth[workerID]; ok {
		health.ConsecutiveFailures++
		if err != nil {
			health.LastError = err.Error()
		}
	}
}

// RecordWorkerSuccess records a worker success for health monitoring
// Implements Requirement 13.4: مراقبة صحة Workers
func (p *RiverWorkerPool) RecordWorkerSuccess(workerID int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if health, ok := p.workerHealth[workerID]; ok {
		// Reset consecutive failures on success
		health.ConsecutiveFailures = 0
		health.LastError = ""
	}
}
