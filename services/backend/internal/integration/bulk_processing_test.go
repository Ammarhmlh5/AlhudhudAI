package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/al-hudhud/backend/internal/batch"
	"github.com/al-hudhud/backend/internal/buffer"
	"github.com/al-hudhud/backend/internal/config"
	"github.com/al-hudhud/backend/internal/database"
	"github.com/al-hudhud/backend/internal/gateway"
	goredis "github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestBulkProcessingFlow(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	logger := zap.NewNop()

	// 1. Setup real Database and Redis for integration test
	cfg := &config.Config{}
	cfg.Database.Host = "localhost"
	cfg.Database.Port = "5432"
	cfg.Database.User = "postgres"
	cfg.Database.Password = testDBPassword()
	cfg.Database.Name = "sms_gateway_test"
	cfg.Database.SSLMode = "disable"
	cfg.Database.MaxOpenConns = 10
	cfg.Database.MaxIdleConns = 5
	cfg.Database.ConnMaxLifetime = time.Hour
	cfg.Database.ConnMaxIdleTime = 30 * time.Minute

	db, err := database.NewDatabase(cfg, logger)
	require.NoError(t, err)
	defer db.Close()

	redisClient := goredis.NewClient(&goredis.Options{
		Addr: "localhost:6379",
		DB:   13, // Dedicated test DB so FlushDB does not collide with other packages
	})
	requireRedisPing(t, redisClient)
	defer redisClient.Close()

	// 2. Setup services
	messageBuffer := buffer.NewMessageBuffer(redisClient, logger)
	rateLimiter := &MockRateLimiter{}
	apiHandler := gateway.NewHandler(rateLimiter, messageBuffer)
	batchProcessor := batch.NewBatchProcessor(messageBuffer, db, &batch.BatchConfig{
		DefaultBatchSize: 100,
		ProcessInterval:  100 * time.Millisecond,
	}, logger, nil)

	// 3. Setup test data
	cleanupTestData(t, ctx, db, redisClient)
	tenantID := createTestTenant(t, ctx, db, decimal.NewFromFloat(100.0))
	_ = createTestChannel(t, ctx, db, tenantID)

	// 4. Send bulk messages via API
	messages := make([]gateway.MessageRequest, 50)
	for i := 0; i < 50; i++ {
		messages[i] = gateway.MessageRequest{
			TenantID:    tenantID,
			PhoneNumber: fmt.Sprintf("+12345678%02d", i),
			Content:     "Bulk message test content",
		}
	}

	bulkReq := gateway.BulkMessageRequest{
		TenantID: tenantID,
		Messages: messages,
	}
	body, _ := json.Marshal(bulkReq)

	req := httptest.NewRequest(http.MethodPost, "/api/messages/bulk", bytes.NewBuffer(body))
	req.Header.Set("X-Tenant-ID", tenantID.String())
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	apiHandler.ReceiveBulk(w, req)

	// Verify API response
	assert.Equal(t, http.StatusOK, w.Code)
	var bulkResp gateway.BulkResponse
	err = json.Unmarshal(w.Body.Bytes(), &bulkResp)
	require.NoError(t, err)
	assert.Equal(t, 50, bulkResp.AcceptedCount)

	// 5. Verify messages are in Redis buffer
	size, err := messageBuffer.GetSize(ctx, tenantID)
	require.NoError(t, err)
	assert.Equal(t, int64(50), size)

	// 6. Run batch processor to move messages to DB
	result, err := batchProcessor.ProcessBatch(ctx, tenantID)
	require.NoError(t, err)
	assert.Equal(t, 50, result.ProcessedCount)

	// 7. Verify messages are in DB
	var count int
	err = db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM messages WHERE tenant_id = $1", tenantID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 50, count)

	// 8. Verify Redis buffer is empty
	size, err = messageBuffer.GetSize(ctx, tenantID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), size)
}
