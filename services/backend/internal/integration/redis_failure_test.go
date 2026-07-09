package integration

import (
	"context"
	"testing"
	"time"

	"github.com/al-hudhud/backend/internal/buffer"
	"github.com/al-hudhud/backend/internal/config"
	"github.com/al-hudhud/backend/internal/database"
	"github.com/al-hudhud/backend/internal/gateway"
	goredis "github.com/redis/go-redis/v9"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestRedisFailureRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping redis failure recovery test")
	}

	ctx := context.Background()
	logger := zap.NewNop()

	cfg := &config.Config{}
	cfg.Database.Host = "localhost"
	cfg.Database.Port = "5432"
	cfg.Database.User = "postgres"
	cfg.Database.Password = "postgres"
	cfg.Database.Name = "sms_gateway_test"

	db, err := database.NewDatabase(cfg, logger)
	require.NoError(t, err)
	defer db.Close()

	redisClient := goredis.NewClient(&goredis.Options{
		Addr: "localhost:6379",
		DB:   2,
	})
	require.NoError(t, redisClient.Ping(ctx).Err())
	defer redisClient.Close()

	messageBuffer := buffer.NewMessageBuffer(redisClient, logger)
	tenantID := uuid.New()

	t.Run("RedisPushWithFailure", func(t *testing.T) {
		msg := gateway.BufferedMessage{
			MessageID:   uuid.New().String(),
			TenantID:    tenantID,
			PhoneNumber: "+1234567890",
			Content:     "Test message",
			QueuedAt:    time.Now(),
		}

		// Success case
		err := messageBuffer.Push(ctx, tenantID, msg)
		require.NoError(t, err)

		// Simulate failure by closing client
		redisClient.Close()
		
		// Attempt push (should fail or handle gracefully)
		err = messageBuffer.Push(ctx, tenantID, msg)
		assert.Error(t, err)
		
		// Reconnect for cleanup
		redisClient = goredis.NewClient(&goredis.Options{
			Addr: "localhost:6379",
			DB:   2,
		})
		redisClient.FlushDB(ctx)
	})
}
