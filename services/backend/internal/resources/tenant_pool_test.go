package resources

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// MockJob implements the Job interface for testing
type MockJob struct {
	id          string
	tenantID    uuid.UUID
	gatewayID   int
	executeFunc func(ctx context.Context) error
	executed    atomic.Bool
}

func NewMockJob(id string, tenantID uuid.UUID, gatewayID int) *MockJob {
	return &MockJob{
		id:        id,
		tenantID:  tenantID,
		gatewayID: gatewayID,
		executeFunc: func(ctx context.Context) error {
			time.Sleep(10 * time.Millisecond) // Simulate work
			return nil
		},
	}
}

func (m *MockJob) Execute(ctx context.Context) error {
	m.executed.Store(true)
	return m.executeFunc(ctx)
}

func (m *MockJob) GetID() string {
	return m.id
}

func (m *MockJob) GetTenantID() uuid.UUID {
	return m.tenantID
}

func (m *MockJob) GetGatewayID() int {
	return m.gatewayID
}

func (m *MockJob) WasExecuted() bool {
	return m.executed.Load()
}

// TestNewTenantWorkerPool tests creating a new worker pool
func TestNewTenantWorkerPool(t *testing.T) {
	logger := zap.NewNop()
	tenantID := uuid.New()
	gatewayID := 1

	config := &TenantWorkerPoolConfig{
		TenantID:   tenantID,
		GatewayID:  gatewayID,
		MaxWorkers: 5,
		QueueSize:  50,
		RateLimit:  100,
	}

	pool := NewTenantWorkerPool(config, logger)

	assert.NotNil(t, pool)
	assert.Equal(t, tenantID, pool.tenantID)
	assert.Equal(t, gatewayID, pool.gatewayID)
	assert.Equal(t, 5, pool.maxWorkers)
	assert.Equal(t, 50, pool.queueSize)
	assert.False(t, pool.isRunning)
}

// TestTenantWorkerPool_StartStop tests starting and stopping the pool
func TestTenantWorkerPool_StartStop(t *testing.T) {
	logger := zap.NewNop()
	tenantID := uuid.New()
	gatewayID := 1

	config := &TenantWorkerPoolConfig{
		TenantID:   tenantID,
		GatewayID:  gatewayID,
		MaxWorkers: 3,
		QueueSize:  10,
		RateLimit:  100,
	}

	pool := NewTenantWorkerPool(config, logger)

	// Start the pool
	ctx := context.Background()
	err := pool.Start(ctx)
	require.NoError(t, err)
	assert.True(t, pool.isRunning)
	assert.Equal(t, 3, len(pool.workers))

	// Try to start again - should fail
	err = pool.Start(ctx)
	assert.Error(t, err)

	// Stop the pool
	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = pool.Stop(stopCtx)
	require.NoError(t, err)
	assert.False(t, pool.isRunning)
}

// TestTenantWorkerPool_SubmitJob tests submitting jobs to the pool
func TestTenantWorkerPool_SubmitJob(t *testing.T) {
	logger := zap.NewNop()
	tenantID := uuid.New()
	gatewayID := 1

	config := &TenantWorkerPoolConfig{
		TenantID:   tenantID,
		GatewayID:  gatewayID,
		MaxWorkers: 2,
		QueueSize:  10,
		RateLimit:  100,
	}

	pool := NewTenantWorkerPool(config, logger)

	ctx := context.Background()
	err := pool.Start(ctx)
	require.NoError(t, err)
	defer pool.Stop(context.Background())

	// Submit a job
	job := NewMockJob("job1", tenantID, gatewayID)
	err = pool.Submit(ctx, job)
	require.NoError(t, err)

	// Wait for job to be processed
	time.Sleep(100 * time.Millisecond)

	// Check job was executed
	assert.True(t, job.WasExecuted())

	// Check metrics
	status := pool.GetStatus()
	assert.Equal(t, int64(1), status.SubmittedCount)
	assert.True(t, status.ProcessedCount > 0)
}

// TestTenantWorkerPool_MultipleJobs tests processing multiple jobs
func TestTenantWorkerPool_MultipleJobs(t *testing.T) {
	logger := zap.NewNop()
	tenantID := uuid.New()
	gatewayID := 1

	config := &TenantWorkerPoolConfig{
		TenantID:   tenantID,
		GatewayID:  gatewayID,
		MaxWorkers: 3,
		QueueSize:  20,
		RateLimit:  1000,
	}

	pool := NewTenantWorkerPool(config, logger)

	ctx := context.Background()
	err := pool.Start(ctx)
	require.NoError(t, err)
	defer pool.Stop(context.Background())

	// Submit multiple jobs
	jobCount := 10
	jobs := make([]*MockJob, jobCount)
	for i := 0; i < jobCount; i++ {
		job := NewMockJob(uuid.New().String(), tenantID, gatewayID)
		jobs[i] = job
		err = pool.Submit(ctx, job)
		require.NoError(t, err)
	}

	// Wait for all jobs to be processed
	time.Sleep(500 * time.Millisecond)

	// Check all jobs were executed
	executedCount := 0
	for _, job := range jobs {
		if job.WasExecuted() {
			executedCount++
		}
	}
	assert.Equal(t, jobCount, executedCount)

	// Check metrics
	status := pool.GetStatus()
	assert.Equal(t, int64(jobCount), status.SubmittedCount)
	assert.Equal(t, int64(0), status.RejectedCount)
}

// TestTenantWorkerPool_BoundedQueue tests that the queue is bounded
func TestTenantWorkerPool_BoundedQueue(t *testing.T) {
	logger := zap.NewNop()
	tenantID := uuid.New()
	gatewayID := 1

	config := &TenantWorkerPoolConfig{
		TenantID:   tenantID,
		GatewayID:  gatewayID,
		MaxWorkers: 1,
		QueueSize:  5, // Small queue
		RateLimit:  1000,
	}

	pool := NewTenantWorkerPool(config, logger)

	ctx := context.Background()
	err := pool.Start(ctx)
	require.NoError(t, err)
	defer pool.Stop(context.Background())

	// Create slow jobs to fill the queue
	slowJob := func(ctx context.Context) error {
		time.Sleep(100 * time.Millisecond)
		return nil
	}

	// Submit jobs until queue is full
	var submittedCount, rejectedCount int
	for i := 0; i < 20; i++ {
		job := NewMockJob(uuid.New().String(), tenantID, gatewayID)
		job.executeFunc = slowJob

		err = pool.TrySubmit(job)
		if err != nil {
			rejectedCount++
		} else {
			submittedCount++
		}
	}

	// Should have some rejected jobs due to bounded queue
	assert.True(t, rejectedCount > 0, "Expected some jobs to be rejected due to full queue")
	assert.True(t, submittedCount <= config.QueueSize+config.MaxWorkers, "Submitted count should not exceed queue size + workers")

	// Check metrics
	status := pool.GetStatus()
	assert.True(t, status.RejectedCount > 0)
}

// TestTenantWorkerPool_TenantIsolation tests that jobs from different tenants are rejected
func TestTenantWorkerPool_TenantIsolation(t *testing.T) {
	logger := zap.NewNop()
	tenantID1 := uuid.New()
	tenantID2 := uuid.New()
	gatewayID := 1

	config := &TenantWorkerPoolConfig{
		TenantID:   tenantID1,
		GatewayID:  gatewayID,
		MaxWorkers: 2,
		QueueSize:  10,
		RateLimit:  100,
	}

	pool := NewTenantWorkerPool(config, logger)

	ctx := context.Background()
	err := pool.Start(ctx)
	require.NoError(t, err)
	defer pool.Stop(context.Background())

	// Submit job from correct tenant - should succeed
	job1 := NewMockJob("job1", tenantID1, gatewayID)
	err = pool.Submit(ctx, job1)
	assert.NoError(t, err)

	// Submit job from different tenant - should fail
	job2 := NewMockJob("job2", tenantID2, gatewayID)
	err = pool.Submit(ctx, job2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not match pool tenant_id")

	// Check metrics
	status := pool.GetStatus()
	assert.Equal(t, int64(1), status.SubmittedCount)
	assert.Equal(t, int64(1), status.RejectedCount)
}

// TestTenantWorkerPool_GatewayIsolation tests that jobs from different gateways are rejected
func TestTenantWorkerPool_GatewayIsolation(t *testing.T) {
	logger := zap.NewNop()
	tenantID := uuid.New()
	gatewayID1 := 1
	gatewayID2 := 2

	config := &TenantWorkerPoolConfig{
		TenantID:   tenantID,
		GatewayID:  gatewayID1,
		MaxWorkers: 2,
		QueueSize:  10,
		RateLimit:  100,
	}

	pool := NewTenantWorkerPool(config, logger)

	ctx := context.Background()
	err := pool.Start(ctx)
	require.NoError(t, err)
	defer pool.Stop(context.Background())

	// Submit job from correct gateway - should succeed
	job1 := NewMockJob("job1", tenantID, gatewayID1)
	err = pool.Submit(ctx, job1)
	assert.NoError(t, err)

	// Submit job from different gateway - should fail
	job2 := NewMockJob("job2", tenantID, gatewayID2)
	err = pool.Submit(ctx, job2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not match pool gateway_id")

	// Check metrics
	status := pool.GetStatus()
	assert.Equal(t, int64(1), status.SubmittedCount)
	assert.Equal(t, int64(1), status.RejectedCount)
}

// TestTenantWorkerPool_RateLimiting tests rate limiting
func TestTenantWorkerPool_RateLimiting(t *testing.T) {
	logger := zap.NewNop()
	tenantID := uuid.New()
	gatewayID := 1

	config := &TenantWorkerPoolConfig{
		TenantID:   tenantID,
		GatewayID:  gatewayID,
		MaxWorkers: 5,
		QueueSize:  100,
		RateLimit:  10, // Very low rate limit
	}

	pool := NewTenantWorkerPool(config, logger)

	ctx := context.Background()
	err := pool.Start(ctx)
	require.NoError(t, err)
	defer pool.Stop(context.Background())

	// Try to submit many jobs quickly
	start := time.Now()
	jobCount := 20
	for i := 0; i < jobCount; i++ {
		job := NewMockJob(uuid.New().String(), tenantID, gatewayID)
		err = pool.Submit(ctx, job)
		require.NoError(t, err)
	}
	duration := time.Since(start)

	// With rate limit of 10/sec, 20 jobs should take at least 1 second
	assert.True(t, duration >= 1*time.Second, "Rate limiting should slow down submissions")
}

// TestTenantWorkerPool_FailedJobs tests handling of failed jobs
func TestTenantWorkerPool_FailedJobs(t *testing.T) {
	logger := zap.NewNop()
	tenantID := uuid.New()
	gatewayID := 1

	config := &TenantWorkerPoolConfig{
		TenantID:   tenantID,
		GatewayID:  gatewayID,
		MaxWorkers: 2,
		QueueSize:  10,
		RateLimit:  100,
	}

	pool := NewTenantWorkerPool(config, logger)

	ctx := context.Background()
	err := pool.Start(ctx)
	require.NoError(t, err)
	defer pool.Stop(context.Background())

	// Submit successful job
	successJob := NewMockJob("success", tenantID, gatewayID)
	err = pool.Submit(ctx, successJob)
	require.NoError(t, err)

	// Submit failing job
	failJob := NewMockJob("fail", tenantID, gatewayID)
	failJob.executeFunc = func(ctx context.Context) error {
		return errors.New("job failed")
	}
	err = pool.Submit(ctx, failJob)
	require.NoError(t, err)

	// Wait for jobs to be processed
	time.Sleep(100 * time.Millisecond)

	// Check metrics
	status := pool.GetStatus()
	assert.Equal(t, int64(2), status.SubmittedCount)
	assert.True(t, status.ProcessedCount > 0)
	assert.True(t, status.FailedCount > 0)
}

// TestTenantWorkerPool_QueueUtilization tests queue utilization metrics
func TestTenantWorkerPool_QueueUtilization(t *testing.T) {
	logger := zap.NewNop()
	tenantID := uuid.New()
	gatewayID := 1

	config := &TenantWorkerPoolConfig{
		TenantID:   tenantID,
		GatewayID:  gatewayID,
		MaxWorkers: 1,
		QueueSize:  10,
		RateLimit:  1000,
	}

	pool := NewTenantWorkerPool(config, logger)

	ctx := context.Background()
	err := pool.Start(ctx)
	require.NoError(t, err)
	defer pool.Stop(context.Background())

	// Initially queue should be empty
	assert.Equal(t, 0, pool.GetQueueLength())
	assert.Equal(t, 10, pool.GetQueueCapacity())
	assert.False(t, pool.IsQueueFull())
	assert.Equal(t, 0.0, pool.GetQueueUtilization())

	// Submit slow jobs to fill queue
	slowJob := func(ctx context.Context) error {
		time.Sleep(200 * time.Millisecond)
		return nil
	}

	for i := 0; i < 5; i++ {
		job := NewMockJob(uuid.New().String(), tenantID, gatewayID)
		job.executeFunc = slowJob
		err = pool.TrySubmit(job)
		require.NoError(t, err)
	}

	// Check queue utilization
	utilization := pool.GetQueueUtilization()
	assert.True(t, utilization > 0, "Queue utilization should be greater than 0")
	assert.True(t, utilization <= 100, "Queue utilization should not exceed 100%")
}

// TestTenantWorkerPoolManager tests the pool manager
func TestTenantWorkerPoolManager(t *testing.T) {
	logger := zap.NewNop()
	manager := NewTenantWorkerPoolManager(logger)

	tenantID1 := uuid.New()
	tenantID2 := uuid.New()
	gatewayID1 := 1
	gatewayID2 := 2

	ctx := context.Background()

	// Create first pool
	config1 := &TenantWorkerPoolConfig{
		TenantID:   tenantID1,
		GatewayID:  gatewayID1,
		MaxWorkers: 2,
		QueueSize:  10,
		RateLimit:  100,
	}
	pool1, err := manager.GetOrCreatePool(ctx, config1)
	require.NoError(t, err)
	assert.NotNil(t, pool1)

	// Get same pool again - should return existing
	pool1Again, err := manager.GetOrCreatePool(ctx, config1)
	require.NoError(t, err)
	assert.Equal(t, pool1, pool1Again)

	// Create second pool
	config2 := &TenantWorkerPoolConfig{
		TenantID:   tenantID2,
		GatewayID:  gatewayID2,
		MaxWorkers: 3,
		QueueSize:  20,
		RateLimit:  200,
	}
	pool2, err := manager.GetOrCreatePool(ctx, config2)
	require.NoError(t, err)
	assert.NotNil(t, pool2)
	assert.NotEqual(t, pool1, pool2)

	// Get all statuses
	statuses := manager.GetAllStatuses()
	assert.Equal(t, 2, len(statuses))

	// Stop specific pool
	err = manager.StopPool(context.Background(), tenantID1, gatewayID1)
	require.NoError(t, err)

	// Try to get stopped pool - should fail
	_, err = manager.GetPool(tenantID1, gatewayID1)
	assert.Error(t, err)

	// Stop all pools
	err = manager.StopAll(context.Background())
	require.NoError(t, err)

	// All pools should be gone
	statuses = manager.GetAllStatuses()
	assert.Equal(t, 0, len(statuses))
}

// TestTenantWorkerPool_ConcurrentSubmissions tests concurrent job submissions
func TestTenantWorkerPool_ConcurrentSubmissions(t *testing.T) {
	logger := zap.NewNop()
	tenantID := uuid.New()
	gatewayID := 1

	config := &TenantWorkerPoolConfig{
		TenantID:   tenantID,
		GatewayID:  gatewayID,
		MaxWorkers: 5,
		QueueSize:  100,
		RateLimit:  1000,
	}

	pool := NewTenantWorkerPool(config, logger)

	ctx := context.Background()
	err := pool.Start(ctx)
	require.NoError(t, err)
	defer pool.Stop(context.Background())

	// Submit jobs concurrently from multiple goroutines
	jobCount := 100
	var wg sync.WaitGroup
	var successCount, failCount atomic.Int64

	for i := 0; i < jobCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			job := NewMockJob(uuid.New().String(), tenantID, gatewayID)
			err := pool.Submit(ctx, job)
			if err != nil {
				failCount.Add(1)
			} else {
				successCount.Add(1)
			}
		}(i)
	}

	wg.Wait()

	// Most jobs should succeed
	assert.True(t, successCount.Load() > 0, "Some jobs should succeed")
	
	// Wait for processing
	time.Sleep(500 * time.Millisecond)

	// Check final status
	status := pool.GetStatus()
	assert.Equal(t, successCount.Load(), status.SubmittedCount)
	assert.Equal(t, failCount.Load(), status.RejectedCount)
}

// TestTenantWorkerPool_GracefulShutdown tests graceful shutdown
func TestTenantWorkerPool_GracefulShutdown(t *testing.T) {
	logger := zap.NewNop()
	tenantID := uuid.New()
	gatewayID := 1

	config := &TenantWorkerPoolConfig{
		TenantID:   tenantID,
		GatewayID:  gatewayID,
		MaxWorkers: 2,
		QueueSize:  10,
		RateLimit:  100,
	}

	pool := NewTenantWorkerPool(config, logger)

	ctx := context.Background()
	err := pool.Start(ctx)
	require.NoError(t, err)

	// Submit some jobs
	for i := 0; i < 5; i++ {
		job := NewMockJob(uuid.New().String(), tenantID, gatewayID)
		err = pool.Submit(ctx, job)
		require.NoError(t, err)
	}

	// Stop gracefully
	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = pool.Stop(stopCtx)
	require.NoError(t, err)

	// Pool should be stopped
	assert.False(t, pool.isRunning)

	// Try to submit after stop - should fail
	job := NewMockJob("after-stop", tenantID, gatewayID)
	err = pool.Submit(ctx, job)
	assert.Error(t, err)
}
