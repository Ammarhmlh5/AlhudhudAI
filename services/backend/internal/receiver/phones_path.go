package receiver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/al-hudhud/backend/internal/redis"
	"github.com/al-hudhud/backend/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type PhonesDispatcher struct {
	redisClient *redis.Client
}

func NewPhonesDispatcher(redisClient *redis.Client) *PhonesDispatcher {
	return &PhonesDispatcher{redisClient: redisClient}
}

// DispatchToPhone direct publishing to paired mobile device channel
func (pd *PhonesDispatcher) DispatchToPhone(ctx context.Context, tenantID uuid.UUID, deviceID *uuid.UUID, msg *models.Message) (bool, error) {
	var targetDeviceID uuid.UUID
	if deviceID != nil {
		targetDeviceID = *deviceID
	} else {
		// Fallback to find any online device
		var device models.PairedDevice
		if err := db.Get(ctx).Where("tenant_id = ? AND status = ?", tenantID, "ONLINE").Order("last_seen DESC").First(&device).Error; err == nil {
			// Use ChannelID if set (must match WebSocket connection's ChannelID)
			targetDeviceID = device.ID
			if device.ChannelID != nil {
				targetDeviceID = *device.ChannelID
			}
		}
	}

	// If deviceID was passed directly, also check ChannelID mismatch
	if deviceID != nil {
		var device models.PairedDevice
		if err := db.Get(ctx).First(&device, "id = ?", *deviceID).Error; err == nil {
			if device.ChannelID != nil {
				targetDeviceID = *device.ChannelID
			}
		}
	}

	if targetDeviceID == uuid.Nil {
		return false, fmt.Errorf("no active phone device connected for tenant: %s", tenantID)
	}

	if pd.redisClient == nil {
		return false, fmt.Errorf("redis client is unavailable for PHONES channel routing")
	}

	msg.DeviceID = &targetDeviceID

	payload, err := json.Marshal(map[string]interface{}{
		"id":           msg.ID,
		"phone_number": msg.PhoneNumber,
		"content":      msg.Content,
		"custom_id":    msg.CustomID,
	})
	if err != nil {
		return false, fmt.Errorf("failed to marshal phone dispatch payload: %w", err)
	}

	channelName := fmt.Sprintf("alhudhud:dispatch:%s", targetDeviceID.String())
	err = pd.redisClient.Client.Publish(ctx, channelName, string(payload)).Err()
	if err != nil {
		return false, fmt.Errorf("failed to publish to Redis PHONES channel: %w", err)
	}

	logger.GetLogger().Info("Strict PHONES path: Message dispatched to mobile channel",
		zap.String("message_id", msg.ID.String()),
		zap.String("device_id", targetDeviceID.String()),
	)
	return true, nil
}
