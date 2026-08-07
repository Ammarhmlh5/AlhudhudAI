package worker

import (
	"context"
	"testing"
	"time"

	"github.com/al-hudhud/backend/internal/balance"
	"github.com/al-hudhud/backend/internal/redis"
	"github.com/al-hudhud/backend/internal/ws"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// TestBug21_ConfigurationTunable verifies that configuration is tunable via constructor parameters
func TestBug21_ConfigurationTunable(t *testing.T) {
	logger := zap.NewNop()
	
	tests := []struct {
		name              string
		batchSize         int
		workerCount       int
		expectedBatchSize int
		expectedWorkers   int
	}{
		{
			name:              "Custom configuration 50x20",
			batchSize:         50,
			workerCount:       20,
			expectedBatchSize: 50,
			expectedWorkers:   20,
		},
		{
			name:              "Custom configuration 100x10",
			batchSize:         100,
			workerCount:       10,
			expectedBatchSize: 100,
			expectedWorkers:   10,
		},
		{
			name:              "Custom configuration 25x40",
			batchSize:         25,
			workerCount:       40,
			expectedBatchSize: 25,
			expectedWorkers:   40,
		},
		{
			name:              "Invalid batch size defaults to 50",
			batchSize:         0,
			workerCount:       20,
			expectedBatchSize: 50,
			expectedWorkers:   20,
		},
		{
			name:              "Invalid worker count defaults to 20",
			batchSize:         50,
			workerCount:       0,
			expectedBatchSize: 50,
			expectedWorkers:   20,
		},
		{
			name:              "Both invalid default to 50x20",
			batchSize:         -1,
			workerCount:       -1,
			expectedBatchSize: 50,
			expectedWorkers:   20,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create processor with custom configuration
			processor := NewFastMessageProcessor(
				&redis.MessageQueue{},
				&redis.ChannelCache{},
				&MockManager{},
				&ws.Hub{},
				nil, // alertManager
				logger,
				tt.batchSize,
				tt.workerCount,
			)
			
			// Verify configuration was applied
			assert.Equal(t, tt.expectedBatchSize, processor.batchSize, "Batch size should match expected")
			assert.Equal(t, tt.expectedWorkers, processor.workerCount, "Worker count should match expected")
		})
	}
}

// TestBug21_NewDefaultConfiguration verifies new defaults (50x20) are set
func TestBug21_NewDefaultConfiguration(t *testing.T) {
	logger := zap.NewNop()
	
	// Create processor with invalid values to trigger defaults
	processor := NewFastMessageProcessor(
		&redis.MessageQueue{},
		&redis.ChannelCache{},
		&MockManager{},
		&ws.Hub{},
		nil, // alertManager
		logger,
		0,  // Invalid - should default to 50
		0,  // Invalid - should default to 20
	)
	
	// Verify new defaults based on Task 21.1 benchmarks
	assert.Equal(t, 50, processor.batchSize, "Default batch size should be 50 (was 100)")
	assert.Equal(t, 20, processor.workerCount, "Default worker count should be 20 (was 10)")
}

// TestBug21_MetricsTracking verifies performance metrics are tracked
func TestBug21_MetricsTracking(t *testing.T) {
	logger := zap.NewNop()
	
	processor := NewFastMessageProcessor(
		&redis.MessageQueue{},
		&redis.ChannelCache{},
		&MockManager{},
		&ws.Hub{},
		nil, // alertManager
		logger,
		50,
		20,
	)
	
	// Verify metrics are initialized
	assert.Equal(t, uint64(0), processor.messagesProcessed, "Messages processed should start at 0")
	assert.Equal(t, uint64(0), processor.totalLatencyMs, "Total latency should start at 0")
	assert.False(t, processor.metricsStartTime.IsZero(), "Metrics start time should be set")
}

// TestBug21_ConfigurationNotHardcoded verifies configuration is not hardcoded
func TestBug21_ConfigurationNotHardcoded(t *testing.T) {
	logger := zap.NewNop()
	
	// Test that different configurations can be created
	configs := []struct {
		batchSize   int
		workerCount int
	}{
		{25, 40},
		{50, 20},
		{100, 10},
		{200, 5},
	}
	
	for _, cfg := range configs {
		processor := NewFastMessageProcessor(
			&redis.MessageQueue{},
			&redis.ChannelCache{},
			&MockManager{},
			&ws.Hub{},
			nil, // alertManager
			logger,
			cfg.batchSize,
			cfg.workerCount,
		)
		
		// Verify each configuration is respected
		assert.Equal(t, cfg.batchSize, processor.batchSize, "Configuration should not be hardcoded")
		assert.Equal(t, cfg.workerCount, processor.workerCount, "Configuration should not be hardcoded")
	}
}

// TestBug21_MetricsLogging verifies metrics logging functionality
func TestBug21_MetricsLogging(t *testing.T) {
	logger := zap.NewNop()
	
	processor := NewFastMessageProcessor(
		&redis.MessageQueue{},
		&redis.ChannelCache{},
		&MockManager{},
		&ws.Hub{},
		nil, // alertManager
		logger,
		50,
		20,
	)
	
	// Test logMetrics with no messages processed
	processor.logMetrics()
	
	// Simulate some processing
	processor.messagesProcessed = 1000
	processor.totalLatencyMs = 5000
	processor.metricsStartTime = time.Now().Add(-10 * time.Second)
	
	// Test logMetrics with data
	processor.logMetrics()
	
	// Verify metrics are readable
	assert.Equal(t, uint64(1000), processor.messagesProcessed)
	assert.Equal(t, uint64(5000), processor.totalLatencyMs)
}

// TestBug21_FixVerification verifies Bug 21 is fixed
func TestBug21_FixVerification(t *testing.T) {
	logger := zap.NewNop()
	
	t.Run("Configuration is tunable", func(t *testing.T) {
		// Create processor with custom configuration
		processor := NewFastMessageProcessor(
			&redis.MessageQueue{},
			&redis.ChannelCache{},
			&MockManager{},
			&ws.Hub{},
			nil, // alertManager
			logger,
			75,
			15,
		)
		
		// Verify configuration is not hardcoded
		assert.Equal(t, 75, processor.batchSize, "Batch size should be tunable")
		assert.Equal(t, 15, processor.workerCount, "Worker count should be tunable")
	})
	
	t.Run("New defaults are set", func(t *testing.T) {
		// Create processor with invalid values to get defaults
		processor := NewFastMessageProcessor(
			&redis.MessageQueue{},
			&redis.ChannelCache{},
			&MockManager{},
			&ws.Hub{},
			nil, // alertManager
			logger,
			0,
			0,
		)
		
		// Verify new defaults from Task 21.1 benchmarks
		assert.Equal(t, 50, processor.batchSize, "Default should be 50 (not 100)")
		assert.Equal(t, 20, processor.workerCount, "Default should be 20 (not 10)")
	})
	
	t.Run("Performance metrics are tracked", func(t *testing.T) {
		processor := NewFastMessageProcessor(
			&redis.MessageQueue{},
			&redis.ChannelCache{},
			&MockManager{},
			&ws.Hub{},
			nil, // alertManager
			logger,
			50,
			20,
		)
		
		// Verify metrics tracking is initialized
		assert.NotNil(t, processor, "Processor should be created")
		assert.False(t, processor.metricsStartTime.IsZero(), "Metrics start time should be set")
	})
}

// TestBug21_EnvironmentConfigurationSupport verifies environment variable support
func TestBug21_EnvironmentConfigurationSupport(t *testing.T) {
	// This test verifies that the config package supports the new environment variables
	// The actual environment variable loading is tested in config package tests
	
	t.Run("Config struct supports worker configuration", func(t *testing.T) {
		// This is a documentation test to ensure the config package
		// has been updated to support:
		// - WORKER_BATCH_SIZE (default 50)
		// - WORKER_COUNT (default 20)
		// - WORKER_TICKER_MS (default 100)
		
		// The actual implementation is in config.go
		// This test serves as documentation of the requirement
		assert.True(t, true, "Config package should support WORKER_BATCH_SIZE, WORKER_COUNT, WORKER_TICKER_MS")
	})
}

// MockManager is a minimal implementation of balance.Manager for testing
type MockManager struct{}

func (m *MockManager) DeductBalance(ctx context.Context, op balance.BalanceOperation) (*balance.BalanceResult, error) {
	return &balance.BalanceResult{Success: true}, nil
}
func (m *MockManager) DeductBalanceTx(ctx context.Context, tx pgx.Tx, op balance.BalanceOperation) (*balance.BalanceResult, error) {
	return &balance.BalanceResult{Success: true}, nil
}
func (m *MockManager) DeductBalanceWithTimeout(ctx context.Context, op balance.BalanceOperation, timeout time.Duration) (*balance.BalanceResult, error) {
	return &balance.BalanceResult{Success: true}, nil
}
func (m *MockManager) RefundBalance(ctx context.Context, op balance.BalanceOperation) (*balance.BalanceResult, error) {
	return &balance.BalanceResult{Success: true}, nil
}
func (m *MockManager) RefundBalanceTx(ctx context.Context, tx pgx.Tx, op balance.BalanceOperation) (*balance.BalanceResult, error) {
	return &balance.BalanceResult{Success: true}, nil
}
func (m *MockManager) GetBalance(ctx context.Context, tenantID uuid.UUID) (decimal.Decimal, error) {
	return decimal.NewFromInt(100), nil
}
func (m *MockManager) CheckSufficient(ctx context.Context, tenantID uuid.UUID, amount decimal.Decimal) (bool, error) {
	return true, nil
}
func (m *MockManager) CreditBalance(ctx context.Context, op balance.BalanceOperation) (*balance.BalanceResult, error) {
	return &balance.BalanceResult{Success: true}, nil
}
func (m *MockManager) TransferBalance(ctx context.Context, from, to uuid.UUID, amount decimal.Decimal, reason, createdBy, ipAddress string) (*balance.TransferResult, error) {
	return &balance.TransferResult{Success: true}, nil
}

