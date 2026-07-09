package resources

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// Job represents a job to be processed by the worker pool
type Job interface {
	Execute(ctx context.Context) error
	GetID() string
	GetTenantID() uuid.UUID
	GetGatewayID() int
}

// Worker represents a single worker in the pool
type Worker struct {
	id          int
	tenantID    uuid.UUID
	gatewayID   int
	jobQueue    <-chan Job
	logger      *zap.Logger
	stopChan    chan struct{}
	wg          *sync.WaitGroup
	
	// Metrics
	processedCount atomic.Int64
	failedCount    atomic.Int64
	lastProcessed  atomic.Value // time.Time
}

// NewWorker creates a new worker
func NewWorker(id int, tenantID uuid.UUID, gatewayID int, jobQueue <-chan Job, logger *zap.Logger, wg *sync.WaitGroup) *Worker {
	w := &Worker{
		id:        id,
		tenantID:  tenantID,
		gatewayID: gatewayID,
		jobQueue:  jobQueue,
		logger:    logger,
		stopChan:  make(chan struct{}),
		wg:        wg,
	}
	w.lastProcessed.Store(time.Time{})
	return w
}

// Start starts the worker
func (w *Worker) Start(ctx context.Context) {
	w.wg.Add(1)
	defer w.wg.Done()

	w.logger.Debug("Worker started",
		zap.Int("worker_id", w.id),
		zap.String("tenant_id", w.tenantID.String()),
		zap.Int("gateway_id", w.gatewayID),
	)

	for {
		select {
		case <-ctx.Done():
			w.logger.Debug("Worker stopped due to context cancellation",
				zap.Int("worker_id", w.id),
			)
			return
		case <-w.stopChan:
			w.logger.Debug("Worker stopped",
				zap.Int("worker_id", w.id),
			)
			return
		case job, ok := <-w.jobQueue:
			if !ok {
				w.logger.Debug("Worker stopped due to closed job queue",
					zap.Int("worker_id", w.id),
				)
				return
			}

			// Process the job
			w.processJob(ctx, job)
		}
	}
}

// Stop stops the worker
func (w *Worker) Stop() {
	close(w.stopChan)
}

// processJob processes a single job
func (w *Worker) processJob(ctx context.Context, job Job) {
	startTime := time.Now()

	w.logger.Debug("Processing job",
		zap.Int("worker_id", w.id),
		zap.String("job_id", job.GetID()),
		zap.String("tenant_id", job.GetTenantID().String()),
		zap.Int("gateway_id", job.GetGatewayID()),
	)

	// Execute the job
	err := job.Execute(ctx)
	
	processingTime := time.Since(startTime)

	if err != nil {
		w.failedCount.Add(1)
		w.logger.Error("Job failed",
			zap.Int("worker_id", w.id),
			zap.String("job_id", job.GetID()),
			zap.Error(err),
			zap.Duration("processing_time", processingTime),
		)
	} else {
		w.processedCount.Add(1)
		w.lastProcessed.Store(time.Now())
		w.logger.Debug("Job completed successfully",
			zap.Int("worker_id", w.id),
			zap.String("job_id", job.GetID()),
			zap.Duration("processing_time", processingTime),
		)
	}
}

// GetMetrics returns worker metrics
func (w *Worker) GetMetrics() WorkerMetrics {
	lastProcessed := w.lastProcessed.Load().(time.Time)
	
	return WorkerMetrics{
		WorkerID:       w.id,
		ProcessedCount: w.processedCount.Load(),
		FailedCount:    w.failedCount.Load(),
		LastProcessed:  lastProcessed,
	}
}

// WorkerMetrics represents metrics for a worker
type WorkerMetrics struct {
	WorkerID       int
	ProcessedCount int64
	FailedCount    int64
	LastProcessed  time.Time
}

// TenantWorkerPool represents a worker pool for a specific tenant/gateway
// Implements Requirement 9.9: THE النظام SHALL يدعم تحديد عدد Worker Threads المسموحة
type TenantWorkerPool struct {
	tenantID    uuid.UUID
	gatewayID   int
	workers     []*Worker
	jobQueue    chan Job
	maxWorkers  int
	queueSize   int
	rateLimiter *rate.Limiter
	logger      *zap.Logger
	
	// State management
	mu        sync.RWMutex
	isRunning bool
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	
	// Metrics
	submittedCount atomic.Int64
	rejectedCount  atomic.Int64
	queuedCount    atomic.Int64
}

// TenantWorkerPoolConfig holds configuration for TenantWorkerPool
type TenantWorkerPoolConfig struct {
	TenantID   uuid.UUID
	GatewayID  int
	MaxWorkers int // max_worker_threads from resource_quotas
	QueueSize  int // Maximum number of jobs in queue (bounded queue)
	RateLimit  int // Jobs per second rate limit
}

// NewTenantWorkerPool creates a new TenantWorkerPool
// Implements Requirement 9.9: تحديد عدد Worker Threads المسموحة
func NewTenantWorkerPool(config *TenantWorkerPoolConfig, logger *zap.Logger) *TenantWorkerPool {
	// Default queue size: 10x the number of workers
	queueSize := config.QueueSize
	if queueSize <= 0 {
		queueSize = config.MaxWorkers * 10
	}

	// Default rate limit: 1000 jobs per second
	rateLimit := config.RateLimit
	if rateLimit <= 0 {
		rateLimit = 1000
	}

	return &TenantWorkerPool{
		tenantID:    config.TenantID,
		gatewayID:   config.GatewayID,
		workers:     make([]*Worker, 0, config.MaxWorkers),
		jobQueue:    make(chan Job, queueSize), // Bounded queue
		maxWorkers:  config.MaxWorkers,
		queueSize:   queueSize,
		rateLimiter: rate.NewLimiter(rate.Limit(rateLimit), rateLimit),
		logger:      logger,
		isRunning:   false,
	}
}

// Start starts the worker pool
// Implements Requirement 9.9: تشغيل Worker Pool محدود
func (twp *TenantWorkerPool) Start(ctx context.Context) error {
	twp.mu.Lock()
	defer twp.mu.Unlock()

	if twp.isRunning {
		return fmt.Errorf("worker pool already running for tenant %s gateway %d", twp.tenantID, twp.gatewayID)
	}

	twp.ctx, twp.cancel = context.WithCancel(ctx)

	twp.logger.Info("Starting tenant worker pool",
		zap.String("tenant_id", twp.tenantID.String()),
		zap.Int("gateway_id", twp.gatewayID),
		zap.Int("max_workers", twp.maxWorkers),
		zap.Int("queue_size", twp.queueSize),
	)

	// Start workers
	for i := 0; i < twp.maxWorkers; i++ {
		worker := NewWorker(i, twp.tenantID, twp.gatewayID, twp.jobQueue, twp.logger, &twp.wg)
		twp.workers = append(twp.workers, worker)
		go worker.Start(twp.ctx)
	}

	twp.isRunning = true

	twp.logger.Info("Tenant worker pool started successfully",
		zap.String("tenant_id", twp.tenantID.String()),
		zap.Int("gateway_id", twp.gatewayID),
		zap.Int("worker_count", len(twp.workers)),
	)

	return nil
}

// Stop stops the worker pool gracefully
func (twp *TenantWorkerPool) Stop(ctx context.Context) error {
	twp.mu.Lock()
	if !twp.isRunning {
		twp.mu.Unlock()
		return fmt.Errorf("worker pool not running for tenant %s gateway %d", twp.tenantID, twp.gatewayID)
	}
	twp.mu.Unlock()

	twp.logger.Info("Stopping tenant worker pool",
		zap.String("tenant_id", twp.tenantID.String()),
		zap.Int("gateway_id", twp.gatewayID),
	)

	// Cancel context to signal workers to stop
	twp.cancel()

	// Close job queue to prevent new submissions
	close(twp.jobQueue)

	// Wait for all workers to finish with timeout
	done := make(chan struct{})
	go func() {
		twp.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		twp.logger.Info("All workers stopped gracefully",
			zap.String("tenant_id", twp.tenantID.String()),
			zap.Int("gateway_id", twp.gatewayID),
		)
	case <-ctx.Done():
		twp.logger.Warn("Worker pool stop timed out",
			zap.String("tenant_id", twp.tenantID.String()),
			zap.Int("gateway_id", twp.gatewayID),
		)
		return ctx.Err()
	}

	twp.mu.Lock()
	twp.isRunning = false
	twp.mu.Unlock()

	return nil
}

// Submit submits a job to the worker pool
// Implements bounded queue to prevent memory exhaustion
func (twp *TenantWorkerPool) Submit(ctx context.Context, job Job) error {
	twp.mu.RLock()
	if !twp.isRunning {
		twp.mu.RUnlock()
		return fmt.Errorf("worker pool not running for tenant %s gateway %d", twp.tenantID, twp.gatewayID)
	}
	twp.mu.RUnlock()

	// Validate job belongs to this tenant/gateway
	if job.GetTenantID() != twp.tenantID {
		twp.rejectedCount.Add(1)
		return fmt.Errorf("job tenant_id %s does not match pool tenant_id %s", job.GetTenantID(), twp.tenantID)
	}

	if job.GetGatewayID() != twp.gatewayID {
		twp.rejectedCount.Add(1)
		return fmt.Errorf("job gateway_id %d does not match pool gateway_id %d", job.GetGatewayID(), twp.gatewayID)
	}

	// Apply rate limiting
	if err := twp.rateLimiter.Wait(ctx); err != nil {
		twp.rejectedCount.Add(1)
		return fmt.Errorf("rate limit exceeded for tenant %s gateway %d: %w", twp.tenantID, twp.gatewayID, err)
	}

	// Try to submit to bounded queue
	// Non-blocking submission with timeout
	select {
	case twp.jobQueue <- job:
		twp.submittedCount.Add(1)
		twp.queuedCount.Add(1)
		twp.logger.Debug("Job submitted to queue",
			zap.String("job_id", job.GetID()),
			zap.String("tenant_id", twp.tenantID.String()),
			zap.Int("gateway_id", twp.gatewayID),
			zap.Int("queue_length", len(twp.jobQueue)),
		)
		return nil
	case <-ctx.Done():
		twp.rejectedCount.Add(1)
		return ctx.Err()
	default:
		// Queue is full - reject immediately to prevent blocking
		twp.rejectedCount.Add(1)
		return fmt.Errorf("job queue full for tenant %s gateway %d (size: %d)", twp.tenantID, twp.gatewayID, twp.queueSize)
	}
}

// TrySubmit attempts to submit a job without blocking
// Returns error if queue is full
func (twp *TenantWorkerPool) TrySubmit(job Job) error {
	twp.mu.RLock()
	if !twp.isRunning {
		twp.mu.RUnlock()
		return fmt.Errorf("worker pool not running for tenant %s gateway %d", twp.tenantID, twp.gatewayID)
	}
	twp.mu.RUnlock()

	// Validate job belongs to this tenant/gateway
	if job.GetTenantID() != twp.tenantID {
		twp.rejectedCount.Add(1)
		return fmt.Errorf("job tenant_id %s does not match pool tenant_id %s", job.GetTenantID(), twp.tenantID)
	}

	if job.GetGatewayID() != twp.gatewayID {
		twp.rejectedCount.Add(1)
		return fmt.Errorf("job gateway_id %d does not match pool gateway_id %d", job.GetGatewayID(), twp.gatewayID)
	}

	// Check rate limit without waiting
	if !twp.rateLimiter.Allow() {
		twp.rejectedCount.Add(1)
		return fmt.Errorf("rate limit exceeded for tenant %s gateway %d", twp.tenantID, twp.gatewayID)
	}

	// Try to submit without blocking
	select {
	case twp.jobQueue <- job:
		twp.submittedCount.Add(1)
		twp.queuedCount.Add(1)
		return nil
	default:
		twp.rejectedCount.Add(1)
		return fmt.Errorf("job queue full for tenant %s gateway %d (size: %d)", twp.tenantID, twp.gatewayID, twp.queueSize)
	}
}

// GetStatus returns the current status of the worker pool
func (twp *TenantWorkerPool) GetStatus() *TenantWorkerPoolStatus {
	twp.mu.RLock()
	defer twp.mu.RUnlock()

	// Collect worker metrics
	workerMetrics := make([]WorkerMetrics, 0, len(twp.workers))
	var totalProcessed, totalFailed int64
	for _, worker := range twp.workers {
		metrics := worker.GetMetrics()
		workerMetrics = append(workerMetrics, metrics)
		totalProcessed += metrics.ProcessedCount
		totalFailed += metrics.FailedCount
	}

	return &TenantWorkerPoolStatus{
		TenantID:       twp.tenantID,
		GatewayID:      twp.gatewayID,
		IsRunning:      twp.isRunning,
		MaxWorkers:     twp.maxWorkers,
		ActiveWorkers:  len(twp.workers),
		QueueSize:      twp.queueSize,
		QueueLength:    len(twp.jobQueue),
		SubmittedCount: twp.submittedCount.Load(),
		RejectedCount:  twp.rejectedCount.Load(),
		ProcessedCount: totalProcessed,
		FailedCount:    totalFailed,
		WorkerMetrics:  workerMetrics,
	}
}

// GetQueueLength returns the current length of the job queue
func (twp *TenantWorkerPool) GetQueueLength() int {
	return len(twp.jobQueue)
}

// GetQueueCapacity returns the maximum capacity of the job queue
func (twp *TenantWorkerPool) GetQueueCapacity() int {
	return twp.queueSize
}

// IsQueueFull returns true if the job queue is full
func (twp *TenantWorkerPool) IsQueueFull() bool {
	return len(twp.jobQueue) >= twp.queueSize
}

// GetQueueUtilization returns the queue utilization percentage (0-100)
func (twp *TenantWorkerPool) GetQueueUtilization() float64 {
	if twp.queueSize == 0 {
		return 0
	}
	return (float64(len(twp.jobQueue)) / float64(twp.queueSize)) * 100.0
}

// TenantWorkerPoolStatus represents the status of a tenant worker pool
type TenantWorkerPoolStatus struct {
	TenantID       uuid.UUID
	GatewayID      int
	IsRunning      bool
	MaxWorkers     int
	ActiveWorkers  int
	QueueSize      int
	QueueLength    int
	SubmittedCount int64
	RejectedCount  int64
	ProcessedCount int64
	FailedCount    int64
	WorkerMetrics  []WorkerMetrics
}

// TenantWorkerPoolManager manages multiple tenant worker pools
type TenantWorkerPoolManager struct {
	pools  map[string]*TenantWorkerPool // key: "tenantID:gatewayID"
	mu     sync.RWMutex
	logger *zap.Logger
}

// NewTenantWorkerPoolManager creates a new TenantWorkerPoolManager
func NewTenantWorkerPoolManager(logger *zap.Logger) *TenantWorkerPoolManager {
	return &TenantWorkerPoolManager{
		pools:  make(map[string]*TenantWorkerPool),
		logger: logger,
	}
}

// GetOrCreatePool gets an existing pool or creates a new one
func (tpm *TenantWorkerPoolManager) GetOrCreatePool(ctx context.Context, config *TenantWorkerPoolConfig) (*TenantWorkerPool, error) {
	key := fmt.Sprintf("%s:%d", config.TenantID.String(), config.GatewayID)

	tpm.mu.RLock()
	pool, exists := tpm.pools[key]
	tpm.mu.RUnlock()

	if exists {
		return pool, nil
	}

	// Create new pool
	tpm.mu.Lock()
	defer tpm.mu.Unlock()

	// Double-check after acquiring write lock
	pool, exists = tpm.pools[key]
	if exists {
		return pool, nil
	}

	pool = NewTenantWorkerPool(config, tpm.logger)
	if err := pool.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start worker pool: %w", err)
	}

	tpm.pools[key] = pool

	tpm.logger.Info("Created new tenant worker pool",
		zap.String("tenant_id", config.TenantID.String()),
		zap.Int("gateway_id", config.GatewayID),
		zap.Int("max_workers", config.MaxWorkers),
	)

	return pool, nil
}

// GetPool gets an existing pool
func (tpm *TenantWorkerPoolManager) GetPool(tenantID uuid.UUID, gatewayID int) (*TenantWorkerPool, error) {
	key := fmt.Sprintf("%s:%d", tenantID.String(), gatewayID)

	tpm.mu.RLock()
	defer tpm.mu.RUnlock()

	pool, exists := tpm.pools[key]
	if !exists {
		return nil, fmt.Errorf("worker pool not found for tenant %s gateway %d", tenantID, gatewayID)
	}

	return pool, nil
}

// StopPool stops a specific pool
func (tpm *TenantWorkerPoolManager) StopPool(ctx context.Context, tenantID uuid.UUID, gatewayID int) error {
	key := fmt.Sprintf("%s:%d", tenantID.String(), gatewayID)

	tpm.mu.Lock()
	defer tpm.mu.Unlock()

	pool, exists := tpm.pools[key]
	if !exists {
		return fmt.Errorf("worker pool not found for tenant %s gateway %d", tenantID, gatewayID)
	}

	if err := pool.Stop(ctx); err != nil {
		return err
	}

	delete(tpm.pools, key)

	tpm.logger.Info("Stopped and removed tenant worker pool",
		zap.String("tenant_id", tenantID.String()),
		zap.Int("gateway_id", gatewayID),
	)

	return nil
}

// StopAll stops all pools
func (tpm *TenantWorkerPoolManager) StopAll(ctx context.Context) error {
	tpm.mu.Lock()
	defer tpm.mu.Unlock()

	var errors []error

	for key, pool := range tpm.pools {
		if err := pool.Stop(ctx); err != nil {
			errors = append(errors, fmt.Errorf("failed to stop pool %s: %w", key, err))
		}
	}

	tpm.pools = make(map[string]*TenantWorkerPool)

	if len(errors) > 0 {
		return fmt.Errorf("errors stopping pools: %v", errors)
	}

	tpm.logger.Info("Stopped all tenant worker pools")

	return nil
}

// GetAllStatuses returns status for all pools
func (tpm *TenantWorkerPoolManager) GetAllStatuses() map[string]*TenantWorkerPoolStatus {
	tpm.mu.RLock()
	defer tpm.mu.RUnlock()

	statuses := make(map[string]*TenantWorkerPoolStatus)
	for key, pool := range tpm.pools {
		statuses[key] = pool.GetStatus()
	}

	return statuses
}
