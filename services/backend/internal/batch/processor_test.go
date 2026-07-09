package batch

import (
	"context"
	"testing"
	"time"

	"github.com/al-hudhud/backend/internal/gateway"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestBatchProcessor_AdaptBatchSize tests adaptive batch sizing
func TestBatchProcessor_AdaptBatchSize(t *testing.T) {
	tests := []struct {
		name           string
		bufferSize     int64
		expectedSize   int
		adaptiveMode   bool
		defaultSize    int
		minSize        int
	}{
		{
			name:         "empty buffer",
			bufferSize:   0,
			expectedSize: 0,
			adaptiveMode: true,
			defaultSize:  100,
			minSize:      10,
		},
		{
			name:         "low pressure - less than min",
			bufferSize:   5,
			expectedSize: 5,
			adaptiveMode: true,
			defaultSize:  100,
			minSize:      10,
		},
		{
			name:         "medium pressure",
			bufferSize:   50,
			expectedSize: 50,
			adaptiveMode: true,
			defaultSize:  100,
			minSize:      10,
		},
		{
			name:         "high pressure",
			bufferSize:   150,
			expectedSize: 100,
			adaptiveMode: true,
			defaultSize:  100,
			minSize:      10,
		},
		{
			name:         "adaptive mode disabled",
			bufferSize:   150,
			expectedSize: 100,
			adaptiveMode: false,
			defaultSize:  100,
			minSize:      10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			logger := zap.NewNop()
			config := &BatchConfig{
				DefaultBatchSize: tt.defaultSize,
				MinBatchSize:     tt.minSize,
				MaxBatchSize:     1000,
				AdaptiveMode:     tt.adaptiveMode,
			}

			// Create mock buffer that returns specific size
			mockBuffer := &mockMessageBuffer{
				size: tt.bufferSize,
			}

			bp := &BatchProcessor{
				buffer: mockBuffer,
				config: config,
				logger: logger,
			}

			// Execute
			tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
			size := bp.AdaptBatchSize(context.Background(), tenantID)

			// Assert
			assert.Equal(t, tt.expectedSize, size)
		})
	}
}

// TestBatchProcessor_ConvertToDBFormat tests message conversion
func TestBatchProcessor_ConvertToDBFormat(t *testing.T) {
	bp := &BatchProcessor{
		logger: zap.NewNop(),
	}

	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000123")
	bufferedMessages := []gateway.BufferedMessage{
		{
			MessageID:   uuid.New().String(),
			PhoneNumber: "+1234567890",
			Content:     "Test message",
			Priority:    "NORMAL",
			QueuedAt:    time.Now(),
		},
		{
			MessageID:   uuid.New().String(),
			PhoneNumber: "+0987654321",
			Content:     "Another test",
			Priority:    "HIGH",
			QueuedAt:    time.Now(),
		},
	}

	messages := bp.convertToDBFormat(tenantID, bufferedMessages)

	require.Len(t, messages, 2)

	// Check first message
	assert.Equal(t, tenantID.String(), messages[0].TenantID)
	assert.Equal(t, bufferedMessages[0].MessageID, messages[0].MessageID)
	assert.Equal(t, bufferedMessages[0].PhoneNumber, messages[0].PhoneNumber)
	assert.Equal(t, bufferedMessages[0].Content, messages[0].Content)
	assert.Equal(t, "PENDING", messages[0].Status)
	assert.Equal(t, "NORMAL", messages[0].Priority)
	assert.Equal(t, 1, messages[0].PartsCount)
	assert.Equal(t, "0.05", messages[0].Cost.String())

	// Check second message
	assert.Equal(t, tenantID.String(), messages[1].TenantID)
	assert.Equal(t, "HIGH", messages[1].Priority)
}

// TestBatchProcessor_ProcessBatch_EmptyBuffer tests processing empty buffer
func TestBatchProcessor_ProcessBatch_EmptyBuffer(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultBatchConfig()

	mockBuffer := &mockMessageBuffer{
		size:     0,
		messages: []gateway.BufferedMessage{},
	}

	bp := &BatchProcessor{
		buffer: mockBuffer,
		config: config,
		logger: logger,
	}

	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	result, err := bp.ProcessBatch(context.Background(), tenantID)

	require.NoError(t, err)
	assert.Equal(t, tenantID, result.TenantID)
	assert.Equal(t, 0, result.ProcessedCount)
	assert.Equal(t, 0, result.SuccessCount)
	assert.Equal(t, 0, result.FailedCount)
}

// TestBatchProcessor_ProcessAllTenants tests processing multiple tenants
func TestBatchProcessor_ProcessAllTenants(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultBatchConfig()

	t1 := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	t2 := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	t3 := uuid.MustParse("00000000-0000-0000-0000-000000000003")

	mockBuffer := &mockMessageBuffer{
		tenants: []uuid.UUID{t1, t2, t3},
		size:    0,
	}

	bp := &BatchProcessor{
		buffer: mockBuffer,
		config: config,
		logger: logger,
	}

	err := bp.ProcessAllTenants(context.Background())
	require.NoError(t, err)

	// Verify all tenants were processed
	assert.True(t, mockBuffer.getAllTenantsCalled)
}

// Mock implementations for testing

type mockMessageBuffer struct {
	size                  int64
	messages              []gateway.BufferedMessage
	tenants               []uuid.UUID
	getAllTenantsCalled   bool
	removeActiveTenantCalled bool
}

func (m *mockMessageBuffer) Push(ctx context.Context, tenantID uuid.UUID, msg gateway.BufferedMessage) error {
	m.messages = append(m.messages, msg)
	m.size++
	return nil
}

func (m *mockMessageBuffer) PopBatch(ctx context.Context, tenantID uuid.UUID, batchSize int) ([]gateway.BufferedMessage, error) {
	if len(m.messages) == 0 {
		return []gateway.BufferedMessage{}, nil
	}

	count := 0
	if batchSize < len(m.messages) {
		count = batchSize
	} else {
		count = len(m.messages)
	}

	result := m.messages[:count]
	m.messages = m.messages[count:]
	m.size -= int64(count)

	return result, nil
}

func (m *mockMessageBuffer) GetSize(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	return m.size, nil
}

func (m *mockMessageBuffer) GetAllActiveTenants(ctx context.Context) ([]uuid.UUID, error) {
	m.getAllTenantsCalled = true
	return m.tenants, nil
}

func (m *mockMessageBuffer) RemoveActiveTenant(ctx context.Context, tenantID uuid.UUID) error {
	m.removeActiveTenantCalled = true
	return nil
}

func (m *mockMessageBuffer) Clear(ctx context.Context, tenantID uuid.UUID) error {
	m.messages = []gateway.BufferedMessage{}
	m.size = 0
	return nil
}

func (m *mockMessageBuffer) SetTTL(ctx context.Context, tenantID uuid.UUID, ttl time.Duration) error {
	return nil
}
