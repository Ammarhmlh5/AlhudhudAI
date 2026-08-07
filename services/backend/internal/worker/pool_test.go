package worker

import (
	"context"
	"testing"
	"time"

	"github.com/al-hudhud/backend/internal/balance"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/al-hudhud/backend/internal/routing"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// MockBalanceManager is a mock implementation of BalanceManager
type MockBalanceManager struct {
	mock.Mock
}

func (m *MockBalanceManager) DeductBalance(ctx context.Context, op balance.BalanceOperation) (*balance.BalanceResult, error) {
	args := m.Called(ctx, op)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*balance.BalanceResult), args.Error(1)
}

func (m *MockBalanceManager) DeductBalanceTx(ctx context.Context, tx pgx.Tx, op balance.BalanceOperation) (*balance.BalanceResult, error) {
	args := m.Called(ctx, tx, op)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*balance.BalanceResult), args.Error(1)
}

func (m *MockBalanceManager) DeductBalanceWithTimeout(ctx context.Context, op balance.BalanceOperation, timeout time.Duration) (*balance.BalanceResult, error) {
	args := m.Called(ctx, op, timeout)
	return args.Get(0).(*balance.BalanceResult), args.Error(1)
}

func (m *MockBalanceManager) RefundBalance(ctx context.Context, op balance.BalanceOperation) (*balance.BalanceResult, error) {
	args := m.Called(ctx, op)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*balance.BalanceResult), args.Error(1)
}

func (m *MockBalanceManager) RefundBalanceTx(ctx context.Context, tx pgx.Tx, op balance.BalanceOperation) (*balance.BalanceResult, error) {
	args := m.Called(ctx, tx, op)
	return args.Get(0).(*balance.BalanceResult), args.Error(1)
}

func (m *MockBalanceManager) GetBalance(ctx context.Context, tenantID uuid.UUID) (decimal.Decimal, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).(decimal.Decimal), args.Error(1)
}

func (m *MockBalanceManager) CheckSufficient(ctx context.Context, tenantID uuid.UUID, amount decimal.Decimal) (bool, error) {
	args := m.Called(ctx, tenantID, amount)
	return args.Bool(0), args.Error(1)
}

func (m *MockBalanceManager) CreditBalance(ctx context.Context, op balance.BalanceOperation) (*balance.BalanceResult, error) {
	args := m.Called(ctx, op)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*balance.BalanceResult), args.Error(1)
}

func (m *MockBalanceManager) TransferBalance(ctx context.Context, from, to uuid.UUID, amount decimal.Decimal, reason, createdBy, ipAddress string) (*balance.TransferResult, error) {
	args := m.Called(ctx, from, to, amount, reason, createdBy, ipAddress)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*balance.TransferResult), args.Error(1)
}

// MockRoutingEngine is a mock implementation of RoutingEngine
type MockRoutingEngine struct {
	mock.Mock
}

func (m *MockRoutingEngine) SelectChannel(ctx context.Context, tenantID uuid.UUID) (*routing.Channel, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*routing.Channel), args.Error(1)
}

func (m *MockRoutingEngine) IncrementChannelLoad(ctx context.Context, channelID uuid.UUID, tenantID uuid.UUID) error {
	args := m.Called(ctx, channelID, tenantID)
	return args.Error(0)
}

// MockWebSocketHub is a mock implementation of WebSocketHub
type MockWebSocketHub struct {
	mock.Mock
}

func (m *MockWebSocketHub) SendMessage(ctx context.Context, msg WebSocketMessage) error {
	args := m.Called(ctx, msg)
	return args.Error(0)
}

// TestDefaultWorkerPoolConfig tests default configuration
func TestDefaultWorkerPoolConfig(t *testing.T) {
	config := DefaultWorkerPoolConfig()

	assert.NotNil(t, config)
	assert.Equal(t, 50, config.WorkerCount, "Worker count should be 50 as per Requirement 13.1")
	assert.Equal(t, 50, config.MaxWorkers, "Max workers should be 50")
	assert.Equal(t, 100*time.Millisecond, config.PollInterval)
	assert.Equal(t, 5*time.Minute, config.JobTimeout)
	assert.Equal(t, 3, config.RetryPolicy.MaxAttempts, "Max attempts should be 3 as per Requirement 17.4")
	assert.Equal(t, 1*time.Second, config.RetryPolicy.InitialInterval)
	assert.Equal(t, 1*time.Minute, config.RetryPolicy.MaxInterval)
	assert.Equal(t, 2.0, config.RetryPolicy.Multiplier)
}

// TestNewRiverWorkerPool tests worker pool creation
func TestNewRiverWorkerPool(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultWorkerPoolConfig()

	dependencies := &MessageWorkerDependencies{
		BalanceManager: &MockBalanceManager{},
		RoutingEngine:  &MockRoutingEngine{},
		WebSocketHub:   &MockWebSocketHub{},
		Logger:         logger,
	}

	pool, err := NewRiverWorkerPool(config, nil, logger, dependencies)

	assert.NoError(t, err)
	assert.NotNil(t, pool)
	assert.Equal(t, config, pool.config)
	assert.Equal(t, dependencies, pool.dependencies)
	assert.Equal(t, 50, len(pool.workerMetrics), "Should initialize 50 worker metrics")
}

// TestNewRiverWorkerPool_NilDependencies tests error handling for nil dependencies
func TestNewRiverWorkerPool_NilDependencies(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultWorkerPoolConfig()

	pool, err := NewRiverWorkerPool(config, nil, logger, nil)

	assert.Error(t, err)
	assert.Nil(t, pool)
	assert.Contains(t, err.Error(), "dependencies cannot be nil")
}

// TestNewRiverWorkerPool_NilConfig tests default config when nil
func TestNewRiverWorkerPool_NilConfig(t *testing.T) {
	logger := zap.NewNop()

	dependencies := &MessageWorkerDependencies{
		BalanceManager: &MockBalanceManager{},
		RoutingEngine:  &MockRoutingEngine{},
		WebSocketHub:   &MockWebSocketHub{},
		Logger:         logger,
	}

	pool, err := NewRiverWorkerPool(nil, nil, logger, dependencies)

	assert.NoError(t, err)
	assert.NotNil(t, pool)
	assert.NotNil(t, pool.config)
	assert.Equal(t, 50, pool.config.WorkerCount, "Should use default config with 50 workers")
}

// TestRecordJobProcessed tests job processing metrics
func TestRecordJobProcessed(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultWorkerPoolConfig()

	dependencies := &MessageWorkerDependencies{
		BalanceManager: &MockBalanceManager{},
		RoutingEngine:  &MockRoutingEngine{},
		WebSocketHub:   &MockWebSocketHub{},
		Logger:         logger,
	}

	pool, err := NewRiverWorkerPool(config, nil, logger, dependencies)
	assert.NoError(t, err)

	// Record job processed
	workerID := 0
	processingTimeMs := int64(100)

	pool.RecordJobProcessed(workerID, processingTimeMs)

	// Check metrics
	assert.Equal(t, int64(1), pool.processedCount)
	assert.Equal(t, int64(0), pool.failedCount)
	assert.Equal(t, processingTimeMs, pool.totalProcessingTimeMs)
	assert.False(t, pool.lastProcessedAt.IsZero())

	// Check worker metrics
	metrics := pool.workerMetrics[workerID]
	assert.Equal(t, int64(1), metrics.ProcessedCount)
	assert.Equal(t, int64(0), metrics.FailedCount)
	assert.Equal(t, 1.0, metrics.SuccessRate)
	assert.False(t, metrics.LastProcessedAt.IsZero())
}

// TestRecordJobFailed tests job failure metrics
func TestRecordJobFailed(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultWorkerPoolConfig()

	dependencies := &MessageWorkerDependencies{
		BalanceManager: &MockBalanceManager{},
		RoutingEngine:  &MockRoutingEngine{},
		WebSocketHub:   &MockWebSocketHub{},
		Logger:         logger,
	}

	pool, err := NewRiverWorkerPool(config, nil, logger, dependencies)
	assert.NoError(t, err)

	// Record job failed
	workerID := 0

	pool.RecordJobFailed(workerID)

	// Check metrics
	assert.Equal(t, int64(0), pool.processedCount)
	assert.Equal(t, int64(1), pool.failedCount)

	// Check worker metrics
	metrics := pool.workerMetrics[workerID]
	assert.Equal(t, int64(0), metrics.ProcessedCount)
	assert.Equal(t, int64(1), metrics.FailedCount)
	assert.Equal(t, 0.0, metrics.SuccessRate)
}

// TestRecordJobMetrics_SuccessRate tests success rate calculation
func TestRecordJobMetrics_SuccessRate(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultWorkerPoolConfig()

	dependencies := &MessageWorkerDependencies{
		BalanceManager: &MockBalanceManager{},
		RoutingEngine:  &MockRoutingEngine{},
		WebSocketHub:   &MockWebSocketHub{},
		Logger:         logger,
	}

	pool, err := NewRiverWorkerPool(config, nil, logger, dependencies)
	assert.NoError(t, err)

	workerID := 0

	// Process 8 successful jobs
	for i := 0; i < 8; i++ {
		pool.RecordJobProcessed(workerID, 100)
	}

	// Process 2 failed jobs
	for i := 0; i < 2; i++ {
		pool.RecordJobFailed(workerID)
	}

	// Check metrics
	metrics := pool.workerMetrics[workerID]
	assert.Equal(t, int64(8), metrics.ProcessedCount)
	assert.Equal(t, int64(2), metrics.FailedCount)
	assert.Equal(t, 0.8, metrics.SuccessRate, "Success rate should be 80%")
}

// TestGetWorkerMetrics tests worker metrics retrieval
func TestGetWorkerMetrics(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultWorkerPoolConfig()

	dependencies := &MessageWorkerDependencies{
		BalanceManager: &MockBalanceManager{},
		RoutingEngine:  &MockRoutingEngine{},
		WebSocketHub:   &MockWebSocketHub{},
		Logger:         logger,
	}

	pool, err := NewRiverWorkerPool(config, nil, logger, dependencies)
	assert.NoError(t, err)

	// Record some jobs
	pool.RecordJobProcessed(0, 100)
	pool.RecordJobProcessed(1, 200)
	pool.RecordJobFailed(2)

	// Get metrics
	metrics := pool.GetWorkerMetrics()

	assert.Equal(t, 50, len(metrics), "Should have metrics for all 50 workers")
	assert.Equal(t, int64(1), metrics[0].ProcessedCount)
	assert.Equal(t, int64(1), metrics[1].ProcessedCount)
	assert.Equal(t, int64(1), metrics[2].FailedCount)
}

// TestWorkerPoolConfig_CustomValues tests custom configuration
func TestWorkerPoolConfig_CustomValues(t *testing.T) {
	config := &WorkerPoolConfig{
		WorkerCount:  100,
		QueueName:    "custom_queue",
		MaxWorkers:   100,
		PollInterval: 50 * time.Millisecond,
		JobTimeout:   10 * time.Minute,
		RetryPolicy: RetryPolicy{
			MaxAttempts:     5,
			InitialInterval: 2 * time.Second,
			MaxInterval:     2 * time.Minute,
			Multiplier:      3.0,
		},
	}

	assert.Equal(t, 100, config.WorkerCount)
	assert.Equal(t, "custom_queue", config.QueueName)
	assert.Equal(t, 100, config.MaxWorkers)
	assert.Equal(t, 50*time.Millisecond, config.PollInterval)
	assert.Equal(t, 10*time.Minute, config.JobTimeout)
	assert.Equal(t, 5, config.RetryPolicy.MaxAttempts)
	assert.Equal(t, 2*time.Second, config.RetryPolicy.InitialInterval)
	assert.Equal(t, 2*time.Minute, config.RetryPolicy.MaxInterval)
	assert.Equal(t, 3.0, config.RetryPolicy.Multiplier)
}

// TestWorkerPoolStatus tests status structure
func TestWorkerPoolStatus(t *testing.T) {
	status := &WorkerPoolStatus{
		WorkerCount:           50,
		QueueSize:             100,
		ProcessedCount:        1000,
		FailedCount:           10,
		AverageProcessingTime: 150 * time.Millisecond,
		LastProcessedAt:       time.Now(),
		IsRunning:             true,
	}

	assert.Equal(t, 50, status.WorkerCount)
	assert.Equal(t, int64(100), status.QueueSize)
	assert.Equal(t, int64(1000), status.ProcessedCount)
	assert.Equal(t, int64(10), status.FailedCount)
	assert.Equal(t, 150*time.Millisecond, status.AverageProcessingTime)
	assert.True(t, status.IsRunning)
}

// TestMessageWorkerDependencies tests dependencies structure
func TestMessageWorkerDependencies(t *testing.T) {
	logger := zap.NewNop()

	dependencies := &MessageWorkerDependencies{
		BalanceManager: &MockBalanceManager{},
		RoutingEngine:  &MockRoutingEngine{},
		WebSocketHub:   &MockWebSocketHub{},
		Logger:         logger,
	}

	assert.NotNil(t, dependencies.BalanceManager)
	assert.NotNil(t, dependencies.RoutingEngine)
	assert.NotNil(t, dependencies.WebSocketHub)
	assert.NotNil(t, dependencies.Logger)
}

// TestChannel tests channel structure
func TestChannel(t *testing.T) {
	tenantID := uuid.New()
	channelID := uuid.New()

	channel := &routing.Channel{
		ID:         channelID,
		TenantID:   tenantID,
		ChannelKey: "device-123",
		Status:     "ONLINE",
		Priority:   1,
	}

	assert.Equal(t, channelID, channel.ID)
	assert.Equal(t, tenantID, channel.TenantID)
	assert.Equal(t, "device-123", channel.ChannelKey)
	assert.Equal(t, "ONLINE", channel.Status)
	assert.Equal(t, 1, channel.Priority)
}

// TestMessage tests message structure
func TestMessage(t *testing.T) {
	messageID := uuid.New()
	tenantID := uuid.New()

	message := &models.Message{
		ID:          messageID,
		TenantID:    tenantID,
		PhoneNumber: "+1234567890",
		Content:     "Test message",
		Status:      "PENDING",
		Cost:        0.05,
	}

	assert.Equal(t, messageID, message.ID)
	assert.Equal(t, tenantID, message.TenantID)
	assert.Equal(t, "+1234567890", message.PhoneNumber)
	assert.Equal(t, "Test message", message.Content)
	assert.Equal(t, "PENDING", string(message.Status))
	assert.Equal(t, 0.05, message.Cost)
}

// TestWorkerMetrics tests worker metrics structure
func TestWorkerMetrics(t *testing.T) {
	metrics := &WorkerMetrics{
		WorkerID:        0,
		ProcessedCount:  100,
		FailedCount:     5,
		SuccessRate:     0.95,
		AverageTime:     150 * time.Millisecond,
		LastProcessedAt: time.Now(),
	}

	assert.Equal(t, 0, metrics.WorkerID)
	assert.Equal(t, int64(100), metrics.ProcessedCount)
	assert.Equal(t, int64(5), metrics.FailedCount)
	assert.Equal(t, 0.95, metrics.SuccessRate)
	assert.Equal(t, 150*time.Millisecond, metrics.AverageTime)
	assert.False(t, metrics.LastProcessedAt.IsZero())
}

// TestConcurrentMetricsUpdate tests concurrent metrics updates
func TestConcurrentMetricsUpdate(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultWorkerPoolConfig()

	dependencies := &MessageWorkerDependencies{
		BalanceManager: &MockBalanceManager{},
		RoutingEngine:  &MockRoutingEngine{},
		WebSocketHub:   &MockWebSocketHub{},
		Logger:         logger,
	}

	pool, err := NewRiverWorkerPool(config, nil, logger, dependencies)
	assert.NoError(t, err)

	// Simulate concurrent updates from multiple workers
	done := make(chan bool)
	workers := 10
	jobsPerWorker := 100

	for w := 0; w < workers; w++ {
		go func(workerID int) {
			for j := 0; j < jobsPerWorker; j++ {
				if j%10 == 0 {
					pool.RecordJobFailed(workerID)
				} else {
					pool.RecordJobProcessed(workerID, 100)
				}
			}
			done <- true
		}(w)
	}

	// Wait for all workers to finish
	for w := 0; w < workers; w++ {
		<-done
	}

	// Check total counts
	expectedProcessed := int64(workers * jobsPerWorker * 9 / 10) // 90% success
	expectedFailed := int64(workers * jobsPerWorker / 10)        // 10% failure

	assert.Equal(t, expectedProcessed, pool.processedCount)
	assert.Equal(t, expectedFailed, pool.failedCount)
}
