package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/al-hudhud/backend/internal/redis"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockMessageBuffer is a mock implementation of MessageBuffer for testing
type mockMessageBuffer struct {
	messages map[uuid.UUID][]BufferedMessage
	pushErr  error
	pushFunc func(ctx context.Context, tenantID uuid.UUID, msg BufferedMessage) error
}

func newMockMessageBuffer() *mockMessageBuffer {
	return &mockMessageBuffer{
		messages: make(map[uuid.UUID][]BufferedMessage),
	}
}

func (m *mockMessageBuffer) Push(ctx context.Context, tenantID uuid.UUID, msg BufferedMessage) error {
	if m.pushFunc != nil {
		return m.pushFunc(ctx, tenantID, msg)
	}
	if m.pushErr != nil {
		return m.pushErr
	}
	m.messages[tenantID] = append(m.messages[tenantID], msg)
	return nil
}

func (m *mockMessageBuffer) GetSize(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	return int64(len(m.messages[tenantID])), nil
}

func setupTestHandler(t *testing.T) (*Handler, *miniredis.Miniredis, *mockMessageBuffer) {
	// Create mini redis server
	mr, err := miniredis.Run()
	require.NoError(t, err)

	// Create redis client
	client := goredis.NewClient(&goredis.Options{
		Addr: mr.Addr(),
	})

	// Wrap in internal redis client
	internalClient := &redis.Client{
		Client: client,
	}

	// Create rate limiter
	rateLimiter := NewRateLimiter(internalClient)

	// Create mock buffer
	buffer := newMockMessageBuffer()

	// Create handler
	handler := NewHandler(rateLimiter, buffer)

	return handler, mr, buffer
}

func TestReceiveMessage_Success(t *testing.T) {
	handler, mr, buffer := setupTestHandler(t)
	defer mr.Close()

	tenantID := uuid.New()
	req := MessageRequest{
		TenantID:    tenantID,
		PhoneNumber: "+1234567890",
		Content:     "Hello, World!",
		Priority:    "NORMAL",
	}

	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest(http.MethodPost, "/api/messages", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.ReceiveMessage(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)

	// Check rate limit headers
	assert.NotEmpty(t, w.Header().Get("X-RateLimit-Limit"))
	assert.NotEmpty(t, w.Header().Get("X-RateLimit-Remaining"))
	assert.NotEmpty(t, w.Header().Get("X-RateLimit-Reset"))

	var response MessageResponse
	json.NewDecoder(w.Body).Decode(&response)

	assert.NotEmpty(t, response.MessageID)
	assert.Equal(t, "QUEUED", response.Status)

	// Verify message was buffered
	assert.Len(t, buffer.messages[tenantID], 1)
	assert.Equal(t, response.MessageID, buffer.messages[tenantID][0].MessageID)
}

func TestReceiveMessage_InvalidJSON(t *testing.T) {
	handler, mr, _ := setupTestHandler(t)
	defer mr.Close()

	httpReq := httptest.NewRequest(http.MethodPost, "/api/messages", strings.NewReader("invalid json"))
	w := httptest.NewRecorder()

	handler.ReceiveMessage(w, httpReq)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	json.NewDecoder(w.Body).Decode(&response)

	assert.Equal(t, "INVALID_JSON", response.Error)
}

func TestReceiveMessage_ValidationError(t *testing.T) {
	handler, mr, _ := setupTestHandler(t)
	defer mr.Close()

	tenantID := uuid.New()
	tests := []struct {
		name          string
		request       MessageRequest
		expectedError string
	}{
		{
			name: "invalid phone number",
			request: MessageRequest{
				TenantID:    tenantID,
				PhoneNumber: "invalid",
				Content:     "Hello",
			},
			expectedError: "phone_number",
		},
		{
			name: "empty content",
			request: MessageRequest{
				TenantID:    tenantID,
				PhoneNumber: "+1234567890",
				Content:     "",
			},
			expectedError: "content",
		},
		{
			name: "content too long",
			request: MessageRequest{
				TenantID:    tenantID,
				PhoneNumber: "+1234567890",
				Content:     strings.Repeat("a", 1601),
			},
			expectedError: "content",
		},
		{
			name: "invalid tenant_id",
			request: MessageRequest{
				TenantID:    uuid.Nil,
				PhoneNumber: "+1234567890",
				Content:     "Hello",
			},
			expectedError: "tenant_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.request)
			httpReq := httptest.NewRequest(http.MethodPost, "/api/messages", bytes.NewReader(body))
			w := httptest.NewRecorder()

			handler.ReceiveMessage(w, httpReq)

			assert.Equal(t, http.StatusBadRequest, w.Code)

			var response ErrorResponse
			json.NewDecoder(w.Body).Decode(&response)

			assert.Equal(t, "VALIDATION_ERROR", response.Error)
			assert.NotEmpty(t, response.Errors)

			// Check if expected field is in errors
			found := false
			for _, err := range response.Errors {
				if err.Field == tt.expectedError {
					found = true
					break
				}
			}

			assert.True(t, found, "expected error for field %s", tt.expectedError)
		})
	}
}

func TestReceiveMessage_MethodNotAllowed(t *testing.T) {
	handler, mr, _ := setupTestHandler(t)
	defer mr.Close()

	httpReq := httptest.NewRequest(http.MethodGet, "/api/messages", nil)
	w := httptest.NewRecorder()

	handler.ReceiveMessage(w, httpReq)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)

	var response ErrorResponse
	json.NewDecoder(w.Body).Decode(&response)

	assert.Equal(t, "METHOD_NOT_ALLOWED", response.Error)
}

func TestReceiveBulk_Success(t *testing.T) {
	handler, mr, buffer := setupTestHandler(t)
	defer mr.Close()

	tenantID := uuid.New()
	req := BulkMessageRequest{
		TenantID: tenantID,
		Messages: []MessageRequest{
			{
				TenantID:    tenantID,
				PhoneNumber: "+1234567890",
				Content:     "Message 1",
			},
			{
				TenantID:    tenantID,
				PhoneNumber: "+1234567891",
				Content:     "Message 2",
			},
		},
	}

	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest(http.MethodPost, "/api/messages/bulk", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.ReceiveBulk(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var response BulkResponse
	json.NewDecoder(w.Body).Decode(&response)

	assert.NotEmpty(t, response.BatchID)
	assert.Equal(t, "PROCESSING", response.Status)
	assert.Equal(t, 2, response.TotalCount)
	assert.Equal(t, 2, response.AcceptedCount)

	// Verify messages were buffered
	assert.Len(t, buffer.messages[tenantID], 2)
}

func TestReceiveBulk_ValidationError(t *testing.T) {
	handler, mr, _ := setupTestHandler(t)
	defer mr.Close()

	tenantID := uuid.New()
	tests := []struct {
		name    string
		request BulkMessageRequest
	}{
		{
			name: "invalid tenant_id",
			request: BulkMessageRequest{
				TenantID: uuid.Nil,
				Messages: []MessageRequest{
					{
						TenantID:    tenantID,
						PhoneNumber: "+1234567890",
						Content:     "Message 1",
					},
				},
			},
		},
		{
			name: "empty messages",
			request: BulkMessageRequest{
				TenantID: tenantID,
				Messages: []MessageRequest{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.request)
			httpReq := httptest.NewRequest(http.MethodPost, "/api/messages/bulk", bytes.NewReader(body))
			w := httptest.NewRecorder()

			handler.ReceiveBulk(w, httpReq)

			assert.Equal(t, http.StatusBadRequest, w.Code)

			var response ErrorResponse
			json.NewDecoder(w.Body).Decode(&response)

			assert.Equal(t, "VALIDATION_ERROR", response.Error)
		})
	}
}

func TestRegisterRoutes(t *testing.T) {
	handler, mr, _ := setupTestHandler(t)
	defer mr.Close()

	mux := http.NewServeMux()

	handler.RegisterRoutes(mux)

	// Test that routes are registered
	req := httptest.NewRequest(http.MethodGet, "/api/messages", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// Should get method not allowed (route exists but wrong method)
	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}


func TestReceiveMessage_RateLimitExceeded(t *testing.T) {
	handler, mr, _ := setupTestHandler(t)
	defer mr.Close()

	tenantID := uuid.New()
	req := MessageRequest{
		TenantID:    tenantID,
		PhoneNumber: "+1234567890",
		Content:     "Hello, World!",
		Priority:    "NORMAL",
	}

	// Send 100 requests (should all succeed)
	for i := 0; i < 100; i++ {
		body, _ := json.Marshal(req)
		httpReq := httptest.NewRequest(http.MethodPost, "/api/messages", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler.ReceiveMessage(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code, "Request %d should succeed", i+1)
	}

	// 101st request should be rate limited
	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest(http.MethodPost, "/api/messages", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.ReceiveMessage(w, httpReq)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	// Check rate limit headers
	assert.Equal(t, "100", w.Header().Get("X-RateLimit-Limit"))
	assert.Equal(t, "-1", w.Header().Get("X-RateLimit-Remaining"))
	assert.NotEmpty(t, w.Header().Get("X-RateLimit-Reset"))

	var response ErrorResponse
	json.NewDecoder(w.Body).Decode(&response)

	assert.Equal(t, "RATE_LIMIT_EXCEEDED", response.Error)
}

func TestReceiveMessage_RateLimitTenantIsolation(t *testing.T) {
	handler, mr, _ := setupTestHandler(t)
	defer mr.Close()

	tenant1ID := uuid.New()
	tenant2ID := uuid.New()

	tenant1Req := MessageRequest{
		TenantID:    tenant1ID,
		PhoneNumber: "+1234567890",
		Content:     "Tenant 1 message",
	}

	tenant2Req := MessageRequest{
		TenantID:    tenant2ID,
		PhoneNumber: "+1234567891",
		Content:     "Tenant 2 message",
	}

	// Exhaust rate limit for tenant 1
	for i := 0; i < 100; i++ {
		body, _ := json.Marshal(tenant1Req)
		httpReq := httptest.NewRequest(http.MethodPost, "/api/messages", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler.ReceiveMessage(w, httpReq)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	// Tenant 1 should be rate limited
	body1, _ := json.Marshal(tenant1Req)
	httpReq1 := httptest.NewRequest(http.MethodPost, "/api/messages", bytes.NewReader(body1))
	w1 := httptest.NewRecorder()

	handler.ReceiveMessage(w1, httpReq1)
	assert.Equal(t, http.StatusTooManyRequests, w1.Code)

	// Tenant 2 should still be allowed
	body2, _ := json.Marshal(tenant2Req)
	httpReq2 := httptest.NewRequest(http.MethodPost, "/api/messages", bytes.NewReader(body2))
	w2 := httptest.NewRecorder()

	handler.ReceiveMessage(w2, httpReq2)
	assert.Equal(t, http.StatusOK, w2.Code)

	// Check tenant 2 has full limit
	assert.Equal(t, "100", w2.Header().Get("X-RateLimit-Limit"))
	assert.Equal(t, "99", w2.Header().Get("X-RateLimit-Remaining"))
}

func TestReceiveMessage_BufferError(t *testing.T) {
	handler, mr, buffer := setupTestHandler(t)
	defer mr.Close()

	// Simulate buffer error
	buffer.pushErr = assert.AnError

	tenantID := uuid.New()
	req := MessageRequest{
		TenantID:    tenantID,
		PhoneNumber: "+1234567890",
		Content:     "Hello, World!",
	}

	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest(http.MethodPost, "/api/messages", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.ReceiveMessage(w, httpReq)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response ErrorResponse
	json.NewDecoder(w.Body).Decode(&response)

	assert.Equal(t, "BUFFER_ERROR", response.Error)
}

func TestReceiveBulk_PartialSuccess(t *testing.T) {
	handler, mr, buffer := setupTestHandler(t)
	defer mr.Close()

	tenantID := uuid.New()
	req := BulkMessageRequest{
		TenantID: tenantID,
		Messages: []MessageRequest{
			{
				TenantID:    tenantID,
				PhoneNumber: "+1234567890",
				Content:     "Message 1",
			},
			{
				TenantID:    tenantID,
				PhoneNumber: "+1234567891",
				Content:     "Message 2",
			},
			{
				TenantID:    tenantID,
				PhoneNumber: "+1234567892",
				Content:     "Message 3",
			},
		},
	}

	// Simulate error after first message
	callCount := 0
	buffer.pushErr = nil
	oldPushErr := buffer.pushErr
	
	// Create a wrapper that fails after first call
	originalMessages := buffer.messages
	buffer.messages = make(map[uuid.UUID][]BufferedMessage)
	
	// Override Push to fail after first call
	buffer.pushFunc = func(ctx context.Context, tenantID uuid.UUID, msg BufferedMessage) error {
		callCount++
		if callCount > 1 {
			return assert.AnError
		}
		buffer.messages[tenantID] = append(buffer.messages[tenantID], msg)
		return nil
	}
	defer func() {
		buffer.pushFunc = nil
		buffer.pushErr = oldPushErr
		buffer.messages = originalMessages
	}()

	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest(http.MethodPost, "/api/messages/bulk", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.ReceiveBulk(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var response BulkResponse
	json.NewDecoder(w.Body).Decode(&response)

	assert.Equal(t, 3, response.TotalCount)
	assert.Equal(t, 1, response.AcceptedCount)
	assert.Equal(t, 2, response.RejectedCount)
}

func TestReceiveMessage_TenantIsolation(t *testing.T) {
	handler, mr, buffer := setupTestHandler(t)
	defer mr.Close()

	tenant1ID := uuid.New()
	tenant2ID := uuid.New()

	// Send messages for different tenants
	tenant1Req := MessageRequest{
		TenantID:    tenant1ID,
		PhoneNumber: "+1234567890",
		Content:     "Tenant 1 message",
	}

	tenant2Req := MessageRequest{
		TenantID:    tenant2ID,
		PhoneNumber: "+1234567891",
		Content:     "Tenant 2 message",
	}

	// Send message for tenant 1
	body1, _ := json.Marshal(tenant1Req)
	httpReq1 := httptest.NewRequest(http.MethodPost, "/api/messages", bytes.NewReader(body1))
	w1 := httptest.NewRecorder()
	handler.ReceiveMessage(w1, httpReq1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// Send message for tenant 2
	body2, _ := json.Marshal(tenant2Req)
	httpReq2 := httptest.NewRequest(http.MethodPost, "/api/messages", bytes.NewReader(body2))
	w2 := httptest.NewRecorder()
	handler.ReceiveMessage(w2, httpReq2)
	assert.Equal(t, http.StatusOK, w2.Code)

	// Verify tenant isolation in buffer
	assert.Len(t, buffer.messages[tenant1ID], 1)
	assert.Len(t, buffer.messages[tenant2ID], 1)
	assert.Equal(t, tenant1ID, buffer.messages[tenant1ID][0].TenantID)
	assert.Equal(t, tenant2ID, buffer.messages[tenant2ID][0].TenantID)
}
