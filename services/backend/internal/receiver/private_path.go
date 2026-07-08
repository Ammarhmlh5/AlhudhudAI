package receiver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/al-hudhud/backend/internal/models"
	"github.com/al-hudhud/backend/internal/redis"
	"github.com/al-hudhud/backend/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type PrivateDispatcher struct {
	redisClient *redis.Client
}

func NewPrivateDispatcher(redisClient *redis.Client) *PrivateDispatcher {
	return &PrivateDispatcher{redisClient: redisClient}
}

// DispatchToPrivateServer queues the message into the private gateway path via Redis
func (pd *PrivateDispatcher) DispatchToPrivateServer(ctx context.Context, tenantID uuid.UUID, serverURL string, msg *models.Message) (bool, error) {
	if serverURL == "" {
		return false, fmt.Errorf("private server URL is empty for tenant: %s", tenantID)
	}

	if pd.redisClient == nil {
		return false, fmt.Errorf("redis client is unavailable for PRIVATE channel routing")
	}

	payload, err := json.Marshal(map[string]interface{}{
		"id":           msg.ID,
		"custom_id":    msg.CustomID,
		"tenant_id":    tenantID,
		"phone_number": msg.PhoneNumber,
		"content":      msg.Content,
		"server_url":   serverURL,
	})
	if err != nil {
		return false, fmt.Errorf("failed to marshal private dispatch payload: %w", err)
	}

	// Route to specific Private tenant channel for processing by isolated worker pool
	channelName := fmt.Sprintf("alhudhud:dispatch:private:%s", tenantID.String())
	err = pd.redisClient.Client.Publish(ctx, channelName, string(payload)).Err()
	if err != nil {
		return false, fmt.Errorf("failed to publish to Redis PRIVATE channel: %w", err)
	}

	logger.GetLogger().Info("Strict PRIVATE path: Message dispatched to private service queue",
		zap.String("message_id", msg.ID.String()),
		zap.String("tenant_id", tenantID.String()),
		zap.String("server_url", serverURL),
	)
	return true, nil
}
