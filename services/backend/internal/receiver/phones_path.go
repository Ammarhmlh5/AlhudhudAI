package receiver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/al-hudhud/backend/internal/redis"
	"github.com/al-hudhud/backend/internal/ws"
	"github.com/al-hudhud/backend/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"time"
)

type PhonesDispatcher struct {
	redisClient *redis.Client
}

func NewPhonesDispatcher(redisClient *redis.Client) *PhonesDispatcher {
	return &PhonesDispatcher{redisClient: redisClient}
}

// DispatchToPhone direct publishing to paired mobile device channel
func (pd *PhonesDispatcher) DispatchToPhone(ctx context.Context, tenantID uuid.UUID, deviceID *uuid.UUID, msg *models.Message) (bool, error) {
	if pd.redisClient == nil {
		return false, fmt.Errorf("redis client is unavailable for PHONES channel routing")
	}

	targets := make([]uuid.UUID, 0)

	if deviceID != nil {
		// Direct device targeting: resolve to its channel if set
		targetDeviceID := *deviceID
		var device models.PairedDevice
		if err := db.Get(ctx).First(&device, "id = ?", *deviceID).Error; err == nil && device.ChannelID != nil {
			targetDeviceID = *device.ChannelID
		}
		targets = append(targets, targetDeviceID)
	} else {
		// Fallback: publish to every ONLINE device that is actually connected
		var devices []models.PairedDevice
		if err := db.Get(ctx).Where("tenant_id = ? AND status = ?", tenantID, "ONLINE").Find(&devices).Error; err == nil {
			for _, d := range devices {
				ch := d.ID
				if d.ChannelID != nil {
					ch = *d.ChannelID
				}
				if ws.GlobalHub != nil && ws.GlobalHub.IsChannelConnected(tenantID, ch) {
					targets = append(targets, ch)
				}
			}
		}
		// No actually-connected device found: keep legacy behavior (first ONLINE row)
		if len(targets) == 0 {
			var device models.PairedDevice
			if err := db.Get(ctx).Where("tenant_id = ? AND status = ?", tenantID, "ONLINE").Order("last_seen DESC").First(&device).Error; err == nil {
				targetDeviceID := device.ID
				if device.ChannelID != nil {
					targetDeviceID = *device.ChannelID
				}
				targets = append(targets, targetDeviceID)
			}
		}
	}

	if len(targets) == 0 {
		return false, fmt.Errorf("no active phone device connected for tenant: %s", tenantID)
	}

	msg.DeviceID = &targets[0]

	payload, err := json.Marshal(map[string]interface{}{
		"type":         "MESSAGE_DELIVERY",
		"message_id":   msg.ID,
		"tenant_id":    tenantID,
		"content":      msg.Content,
		"phone_number": msg.PhoneNumber,
		"payload": map[string]interface{}{
			"message_id":   msg.ID,
			"content":      msg.Content,
			"sender_name":  msg.SenderName,
			"phone_number": msg.PhoneNumber,
			"message_type": string(msg.MessageType),
			"timestamp":    time.Now().Format(time.RFC3339),
		},
		"custom_id": msg.CustomID,
	})
	if err != nil {
		return false, fmt.Errorf("failed to marshal phone dispatch payload: %w", err)
	}

	for _, target := range targets {
		channelName := fmt.Sprintf("alhudhud:dispatch:%s", target.String())
		if err := pd.redisClient.Client.Publish(ctx, channelName, string(payload)).Err(); err != nil {
			logger.GetLogger().Error("Failed to publish to Redis PHONES channel",
				zap.String("message_id", msg.ID.String()),
				zap.String("device_id", target.String()),
				zap.Error(err))
			continue
		}
		logger.GetLogger().Info("Strict PHONES path: Message dispatched to mobile channel",
			zap.String("message_id", msg.ID.String()),
			zap.String("device_id", target.String()),
		)
	}

	return true, nil
}
