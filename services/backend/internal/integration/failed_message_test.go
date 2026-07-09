package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/al-hudhud/backend/internal/balance"
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

func TestFailedMessageRefundFlow(t *testing.T) {
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
	balanceManager := balance.NewManager(db.Pool, logger)
	statusHandler := status.NewHandler(db.SqlDB, redisClient, logger)

	// 3. Setup test data
	cleanupTestData(t, ctx, db, redisClient)
	initialBalance := decimal.NewFromFloat(100.0)
	tenantID := createTestTenant(t, ctx, db, initialBalance)
	channelID := createTestChannel(t, ctx, db, tenantID)

	// 4. Send message via API
	messageReq := gateway.MessageRequest{
		TenantID:    tenantID,
		PhoneNumber: "+1234567890",
		Content:     "Refund test",
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

	// 5. Process batch to move to DB
	result, err := batchProcessor.ProcessBatch(ctx, tenantID)
	require.NoError(t, err)
	assert.Equal(t, 1, result.ProcessedCount)

	// 6. Simulate Status Report (FAILED)
	report := status.StatusReport{
		MessageID:  messageID,
		TenantID:   tenantID, // Fixed: use uuid directly
		ChannelID:  channelID, // Fixed: use uuid directly
		Status:     status.StatusFailed,
		FailReason: "GATEWAY_TIMEOUT",
		Timestamp:  time.Now(),
	}

	err = statusHandler.ProcessReport(ctx, report)
	require.NoError(t, err)

	// 7. Verify Refund
	currentBalance, err := balanceManager.GetBalance(ctx, tenantID)
	require.NoError(t, err)
	assert.True(t, currentBalance.Equal(initialBalance), "Balance should be refunded to initial state")
}
