package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/al-hudhud/backend/internal/batch"
	"github.com/al-hudhud/backend/internal/buffer"
	"github.com/al-hudhud/backend/internal/config"
	"github.com/al-hudhud/backend/internal/database"
	"github.com/al-hudhud/backend/internal/gateway"
	"github.com/al-hudhud/backend/internal/status"
	goredis "github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestEndToEndFlow(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	logger := zap.NewNop()

	// 1. Setup real Database and Redis for integration test
	cfg := &config.Config{}
	cfg.Database.Host = "localhost"
	cfg.Database.Port = "5432"
	cfg.Database.User = "postgres"
	cfg.Database.Password = "postgres"
	cfg.Database.Name = "sms_gateway_test"
	cfg.Database.MaxOpenConns = 10
	cfg.Database.MaxIdleConns = 5
	cfg.Database.ConnMaxLifetime = time.Hour
	cfg.Database.ConnMaxIdleTime = 30 * time.Minute

	db, err := database.NewDatabase(cfg, logger)
	require.NoError(t, err)
	defer db.Close()

	redisClient := goredis.NewClient(&goredis.Options{
		Addr: "localhost:6379",
		DB:   1,
	})
	require.NoError(t, redisClient.Ping(ctx).Err())
	defer redisClient.Close()

	// 2. Setup services
	messageBuffer := buffer.NewMessageBuffer(redisClient, logger)
	rateLimiter := &MockRateLimiter{}
	apiHandler := gateway.NewHandler(rateLimiter, messageBuffer)
	batchProcessor := batch.NewBatchProcessor(messageBuffer, db, &batch.BatchConfig{
		DefaultBatchSize: 100,
		ProcessInterval:  100 * time.Millisecond,
	}, logger, nil)
	statusHandler := status.NewHandler(db.SqlDB, redisClient, logger)

	// 3. Setup test data
	cleanupTestData(t, ctx, db, redisClient)
	tenantID := createTestTenant(t, ctx, db, decimal.NewFromFloat(100.0))
	channelID := createTestChannel(t, ctx, db, tenantID)

	// 4. Send message via API
	messageReq := gateway.MessageRequest{
		TenantID:    tenantID,
		PhoneNumber: "+1234567890",
		Content:     "Hello from integration test",
	}
	body, _ := json.Marshal(messageReq)

	req := httptest.NewRequest(http.MethodPost, "/api/messages", bytes.NewBuffer(body))
	req.Header.Set("X-Tenant-ID", tenantID.String())
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	apiHandler.ReceiveMessage(w, req)

	// Verify API response
	assert.Equal(t, http.StatusAccepted, w.Code)
	var messageResp gateway.MessageResponse
	err = json.Unmarshal(w.Body.Bytes(), &messageResp)
	require.NoError(t, err)
	messageID := messageResp.MessageID
	require.NotEmpty(t, messageID)

	// 5. Process batch to move to DB
	result, err := batchProcessor.ProcessBatch(ctx, tenantID)
	require.NoError(t, err)
	assert.Equal(t, 1, result.ProcessedCount)

	// 6. Simulate Status Report (DELIVERED)
	report := status.StatusReport{
		MessageID: messageID,
		TenantID:  tenantID, // Fixed: use uuid directly
		ChannelID: channelID, // Fixed: use uuid directly
		Status:    status.StatusDelivered,
		Timestamp: time.Now(),
	}

	err = statusHandler.ProcessReport(ctx, report)
	require.NoError(t, err)

	// 7. Verify message status in DB
	var dbStatus string
	err = db.Pool.QueryRow(ctx, "SELECT status FROM messages WHERE message_id = $1", messageID).Scan(&dbStatus)
	require.NoError(t, err)
	assert.Equal(t, status.StatusDelivered, dbStatus)
}
