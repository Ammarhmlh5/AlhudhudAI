package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/al-hudhud/backend/internal/ws"
	"github.com/al-hudhud/backend/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type PhoneProvider struct{}

func NewPhoneProvider() *PhoneProvider {
	return &PhoneProvider{}
}

func (p *PhoneProvider) GetType() models.ChannelType {
	return models.ChannelPhone
}

func (p *PhoneProvider) Send(ctx context.Context, tenantID uuid.UUID, msg *models.Message, config string) (string, error) {
	// 1. Get the device ID from the message (set by routing logic)
	var targetDeviceID *uuid.UUID
	if msg.DeviceID != nil {
		targetDeviceID = msg.DeviceID
	} else {
		// Fallback: find any online device for this tenant
		var device models.PairedDevice
		if err := db.Get(ctx).Where("tenant_id = ? AND status = ?", tenantID, "ONLINE").Order("last_seen DESC").First(&device).Error; err == nil {
			targetDeviceID = &device.ID
		}
	}

	if targetDeviceID == nil {
		return "", fmt.Errorf("no active phone device connected for tenant: %s", tenantID)
	}

	// 2. Get the ChannelID for this device
	var channelID uuid.UUID
	var device models.PairedDevice
	if err := db.Get(ctx).Where("id = ?", targetDeviceID).First(&device).Error; err != nil {
		return "", fmt.Errorf("device not found: %w", err)
	}
	if device.ChannelID == nil {
		return "", fmt.Errorf("device has no associated channel")
	}
	channelID = *device.ChannelID

	// 3. Build WebSocket message with required fields
	wsMsg := ws.WebSocketMessage{
		Type:      "MESSAGE",
		MessageID: msg.ID.String(),
		TenantID:  tenantID,
		ChannelID: channelID,
		Payload: map[string]interface{}{
			"id":               msg.ID.String(),
			"phone_number":     msg.PhoneNumber,
			"content":          msg.Content,
			"idempotency_key":  msg.IdempotencyKey,
			"custom_id":        msg.CustomID,
			"tenant_id":        tenantID.String(),
			"timestamp":        time.Now().Format(time.RFC3339),
		},
	}

	// 4. Send via WebSocket Hub to specific device
	if err := ws.GlobalHub.SendMessage(ctx, wsMsg); err != nil {
		logger.GetLogger().Warn("WebSocket send failed, message will be queued for retry",
			zap.String("message_id", msg.ID.String()),
			zap.Error(err))
		return "", err
	}

	logger.GetLogger().Info("Message sent to mobile device via WebSocket",
		zap.String("message_id", msg.ID.String()),
		zap.String("device_id", targetDeviceID.String()),
		zap.String("channel_id", channelID.String()),
		zap.String("phone_number", msg.PhoneNumber),
		zap.String("content_preview", truncateString(msg.Content, 50)))

	return "PHONE_DISPATCHED", nil
}

// truncateString truncates a string to maxLength
func truncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength] + "..."
}

type GatewayHudhudProvider struct{}

func NewGatewayHudhudProvider() *GatewayHudhudProvider {
	return &GatewayHudhudProvider{}
}

func (p *GatewayHudhudProvider) GetType() models.ChannelType {
	return models.ChannelGatewayHudhud
}

func (p *GatewayHudhudProvider) Send(ctx context.Context, tenantID uuid.UUID, msg *models.Message, configStr string) (string, error) {
	if msg == nil {
		return "", fmt.Errorf("message cannot be nil")
	}

	// Mock implementation of a gateway API call
	logger.GetLogger().Info("[GATEWAY] Sending message via Hudhud Gateway", zap.String("msg_id", msg.ID.String()))

	// Simulate external job ID
	externalID := fmt.Sprintf("HUD-%s", uuid.New().String()[:8])

	return externalID, nil
}

type GatewayPrivateProvider struct{}

func NewGatewayPrivateProvider() *GatewayPrivateProvider {
	return &GatewayPrivateProvider{}
}

func (p *GatewayPrivateProvider) GetType() models.ChannelType {
	return models.ChannelGatewayPrivate
}

type GatewayConfig struct {
	APIKey string `json:"api_key"`
	URL    string `json:"url"`
}

func (p *GatewayPrivateProvider) Send(ctx context.Context, tenantID uuid.UUID, msg *models.Message, configStr string) (string, error) {
	if msg == nil {
		return "", fmt.Errorf("message cannot be nil")
	}

	var conf GatewayConfig
	if configStr != "" {
		if err := json.Unmarshal([]byte(configStr), &conf); err != nil {
			logger.GetLogger().Error("[PRIVATE-GW] Invalid configuration format", zap.Error(err))
			return "", fmt.Errorf("invalid provider configuration: %w", err)
		}
	}

	// Private gateways typically require specific API credentials from the config
	logger.GetLogger().Info("[PRIVATE-GW] Sending message via Private Gateway", zap.String("msg_id", msg.ID.String()))

	// Simulate external call
	externalID := fmt.Sprintf("PRV-%s", uuid.New().String()[:8])

	return externalID, nil
}
