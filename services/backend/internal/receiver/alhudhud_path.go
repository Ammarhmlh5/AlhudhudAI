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

type AlhudhudDispatcher struct {
	redisClient *redis.Client
}

func NewAlhudhudDispatcher(redisClient *redis.Client) *AlhudhudDispatcher {
	return &AlhudhudDispatcher{redisClient: redisClient}
}

// DispatchToAlhudhud forwards the message via official Alhudhud dispatch pipeline
func (ad *AlhudhudDispatcher) DispatchToAlhudhud(ctx context.Context, tenantID uuid.UUID, msg *models.Message) (bool, error) {
	if ad.redisClient == nil {
		return false, fmt.Errorf("redis client is unavailable for ALHUDHUD channel routing")
	}

	payload, err := json.Marshal(map[string]interface{}{
		"id":           msg.ID,
		"tenant_id":    tenantID,
		"phone_number": msg.PhoneNumber,
		"content":      msg.Content,
		"custom_id":    msg.CustomID,
		"provider":     "ALHUDHUD_OFFICIAL",
	})
	if err != nil {
		return false, fmt.Errorf("failed to marshal Alhudhud dispatch payload: %w", err)
	}

	// Route to specific Alhudhud tenant channel for processing by Alhudhud shared engines
	channelName := fmt.Sprintf("alhudhud:dispatch:official:%s", tenantID.String())
	err = ad.redisClient.Client.Publish(ctx, channelName, string(payload)).Err()
	if err != nil {
		return false, fmt.Errorf("failed to publish to Redis ALHUDHUD channel: %w", err)
	}

	logger.GetLogger().Info("Strict ALHUDHUD path: Message dispatched to official service queue",
		zap.String("message_id", msg.ID.String()),
		zap.String("tenant_id", tenantID.String()),
	)
	return true, nil
}
