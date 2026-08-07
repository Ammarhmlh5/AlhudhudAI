package device

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/al-hudhud/backend/internal/auth"
	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/al-hudhud/backend/internal/ws"
	"github.com/al-hudhud/backend/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

// PairDeviceRequest defines input for device pairing
type PairDeviceRequest struct {
	DeviceName      string `json:"device_name" binding:"required"`
	HardwareID      string `json:"hardware_id" binding:"required"`
	PhoneNumber     string `json:"phone_number" binding:"required"`
	IsPrimary       bool   `json:"is_primary"`
	ReplaceDeviceID string `json:"replace_device_id"`
}

// PairDevice registers a new mobile device for a tenant and configures its dispatch channel
func (s *Service) PairDevice(ctx context.Context, tenantID uuid.UUID, req PairDeviceRequest) (string, error) {
	var pairKey string
	var hashedKey string

	err := db.Get(ctx).Transaction(func(tx *gorm.DB) error {
		// Handle replacement if requested
		if req.ReplaceDeviceID != "" {
			var oldDevices []models.PairedDevice
			if err := tx.Where("hardware_id = ? AND tenant_id = ?", req.ReplaceDeviceID, tenantID).Find(&oldDevices).Error; err == nil && len(oldDevices) > 0 {
				oldDevice := oldDevices[0]
				// Delete the old device
				tx.Delete(&oldDevice)

				// Optional: also deactivate or delete the associated channel
				if oldDevice.ChannelID != nil {
					tx.Model(&models.DispatchChannel{}).
						Where("id = ? AND status != 'OFFLINE'", *oldDevice.ChannelID).
						Update("status", "OFFLINE")
				}
			}
		}

		// 1. Check if HardwareID is already used
		var existingDevices []models.PairedDevice
		var existing models.PairedDevice
		if err := tx.Where("hardware_id = ?", req.HardwareID).Find(&existingDevices).Error; err == nil && len(existingDevices) > 0 {
			existing = existingDevices[0]
			if existing.TenantID != tenantID {
				return gorm.ErrInvalidData // Device already belongs to another tenant
			}
		}

		pairKey = generatePairKey()
		hashedKey = auth.HashKey(pairKey)

		// 2. Create or Update DispatchChannel
		var channel models.DispatchChannel
		var channels []models.DispatchChannel
		channelKey := req.HardwareID // Use hardware ID as unique channel key

		// Find existing channel for this device hardware ID
		if err := tx.Where("tenant_id = ? AND channel_key = ?", tenantID, channelKey).Find(&channels).Error; err == nil && len(channels) > 0 {
			channel = channels[0]
			channel.Status = "ONLINE"
			if req.IsPrimary {
				channel.IsDefault = true
			}
			if err := tx.Save(&channel).Error; err != nil {
				return err
			}
		} else {
			channel = models.DispatchChannel{
				TenantID:   tenantID,
				Type:       models.ChannelPhone,
				ChannelKey: channelKey,
				Status:     "ONLINE",
				Priority:   1,
				IsDefault:  req.IsPrimary,
				Config:     "{}",
			}
			if err := tx.Create(&channel).Error; err != nil {
				return err
			}
		}

		// 3. Handle IsPrimary (if true, demote other default channels)
		if req.IsPrimary {
			if err := tx.Model(&models.DispatchChannel{}).
				Where("tenant_id = ? AND id != ?", tenantID, channel.ID).
				Update("is_default", false).Error; err != nil {
				return err
			}
		}

		// 4. Create or Update PairedDevice
		device := &models.PairedDevice{
			TenantID:      tenantID,
			ChannelID:     &channel.ID,
			DeviceName:    req.DeviceName,
			HardwareID:    req.HardwareID,
			DevicePairKey: hashedKey,
			PhoneNumber:   req.PhoneNumber,
			Status:        "ONLINE",
			CreatedAt:     time.Now(),
		}

		if existing.ID != uuid.Nil {
			device.ID = existing.ID
			if err := tx.Save(device).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Create(device).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	// Ensure the tenant has a routing config so messages use the PHONES fast path
	if err := s.EnsureCompanyStatus(ctx, tenantID); err != nil {
		logger.GetLogger().Warn("Failed to ensure company status for tenant",
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err))
	}

	return pairKey, nil
}

// ListDevices returns all devices for a tenant
func (s *Service) ListDevices(ctx context.Context, tenantID uuid.UUID) ([]models.PairedDevice, error) {
	var devices []models.PairedDevice
	if err := db.Get(ctx).Where("tenant_id = ?", tenantID).Find(&devices).Error; err != nil {
		return nil, err
	}
	for i := range devices {
		if devices[i].ChannelID != nil && ws.GlobalHub.IsChannelConnected(tenantID, *devices[i].ChannelID) {
			devices[i].Status = "ONLINE"
		} else {
			devices[i].Status = "OFFLINE"
		}
	}
	return devices, nil
}

// FetchMessages implements the Phase 3 Offline Fallback Pull.
// When a device reconnects after being offline, it calls this endpoint to
// retrieve any QUEUED messages that accumulated while it was disconnected.
//
// Critical design choices:
//  - SELECT FOR UPDATE SKIP LOCKED: prevents two devices from grabbing the same message
//  - Batch size limit: avoids overwhelming a freshly-reconnected device
//  - QUEUED → PROCESSING transition is atomic with the fetch
func (s *Service) FetchMessages(ctx context.Context, tenantID uuid.UUID, deviceID uuid.UUID) ([]models.Message, error) {
	const batchSize = 50 // Max messages per reconnect pull to avoid overload

	var messages []models.Message

	// Resolve this device's dispatch channel(s) so messages bound by channel id
	// (the mobile routing convention) are found too, not only by hardware id.
	deviceMatchIDs := []string{deviceID.String()}
	var device models.PairedDevice
	if err := db.Get(ctx).
		Where("hardware_id = ? OR id = ?", deviceID, deviceID).
		First(&device).Error; err == nil {
		if device.ChannelID != nil {
			deviceMatchIDs = append(deviceMatchIDs, device.ChannelID.String())
		} else {
			deviceMatchIDs = append(deviceMatchIDs, device.ID.String())
		}
	}

	// Use a raw SQL transaction for SELECT FOR UPDATE SKIP LOCKED (GORM doesn't support it natively)
	err := db.Get(ctx).Transaction(func(tx *gorm.DB) error {
	// 1. Lock eligible unsent messages for this tenant: QUEUED (ready for a
	//    device pull), WAITING and FAILED (unsent messages that a mobile gateway
	//    device should retry), plus messages already claimed by THIS device but
	//    stuck in PROCESSING (e.g. fetched while offline and never acknowledged).
	//    Skip any already locked.
	if err := tx.Raw(`
			SELECT id, tenant_id, phone_number, content, message_type,
			       char_count, parts_count, encoding, cost, status,
			       sender_name, scheduled_at, created_at
			FROM   messages
			WHERE  tenant_id = ?
			  AND  ( status IN ('QUEUED', 'WAITING', 'FAILED')
			     OR ( status = 'PROCESSING' AND device_id IN ? ) )
			ORDER  BY scheduled_at ASC
			LIMIT  ?
			FOR UPDATE SKIP LOCKED
		`, tenantID, deviceMatchIDs, batchSize).Scan(&messages).Error; err != nil {
			return err
		}

		if len(messages) == 0 {
			return nil
		}

		// 2. Atomically transition grabbed messages → PROCESSING and assign this device
		messageIDs := make([]uuid.UUID, len(messages))
		for i := range messages {
			messageIDs[i] = messages[i].ID
		}

		now := time.Now()
		if err := tx.Model(&models.Message{}).
			Where("id IN ? AND status IN ?", messageIDs, []string{"QUEUED", "WAITING", "PROCESSING", "FAILED"}).
			Updates(map[string]interface{}{
				"status":     models.MessageProcessing,
				"device_id":  deviceID,
				"updated_at": now,
			}).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	if len(messages) == 0 {
		return []models.Message{}, nil
	}

	// 3. Broadcast to admin UI: device pulled messages from fallback queue
	ws.GlobalHub.Broadcast(ws.Event{
		TenantID: tenantID,
		Type:     "DEVICE_FALLBACK_PULL",
		Payload: map[string]interface{}{
			"device_id":     deviceID,
			"message_count": len(messages),
			"timestamp":     time.Now().Format("15:04:05.000"),
		},
	})

	return messages, nil
}


func (s *Service) ReportStatus(ctx context.Context, tenantID uuid.UUID, messageID uuid.UUID, status string, deliveredAt *time.Time) error {
	return db.RunInTenantScope(tenantID, false, func(tx *gorm.DB) error {
		var msg models.Message
		if err := tx.First(&msg, "id = ? AND tenant_id = ?", messageID, tenantID).Error; err != nil {
			return err
		}

		var msgStatus models.MessageStatus
		switch status {
		case "SENT":
			msgStatus = models.MessageSent
		case "DELIVERED":
			msgStatus = models.MessageDelivered
		case "FAILED":
			msgStatus = models.MessageFailed
		default:
			return nil
		}

		// ACCOUNTING PHASE 2: Settlement or Refund
		var transaction models.AccountingTransaction
		if err := tx.Where("message_id = ? AND status = ?", messageID, models.TxPending).First(&transaction).Error; err == nil {
			if status == "SENT" || status == "DELIVERED" {
				// Settlement: Just commit the transaction
				tx.Model(&transaction).Update("status", models.TxCommitted)
			} else if status == "FAILED" {
				// Refund: Give money back to tenant
				var tenant models.Tenant
				if err := tx.First(&tenant, "id = ?", tenantID).Error; err == nil {
					tx.Model(&tenant).Update("balance", tenant.Balance+transaction.Amount)
					tx.Model(&transaction).Update("status", models.TxRefunded)
				}
			}
		}

		updates := map[string]interface{}{
			"status": msgStatus,
		}
		now := time.Now()
		if status == "SENT" || status == "DELIVERED" {
			updates["sent_at"] = &now
		}
		if deliveredAt != nil {
			updates["delivered_at"] = deliveredAt
		} else if status == "DELIVERED" {
			updates["delivered_at"] = &now
		}

		if err := tx.Model(&msg).
			Where("id = ? AND status <> ?", messageID, models.MessageDelivered).
			Updates(updates).Error; err != nil {
			return err
		}

		// 4. Broadcast Status Update
		ws.GlobalHub.Broadcast(ws.Event{
			TenantID: tenantID,
			Type:     "MESSAGE_STATUS_UPDATE",
			Payload: map[string]interface{}{
				"message_id":   messageID,
				"status":       status,
				"delivered_at": updates["delivered_at"],
				"timestamp":    time.Now().Format("15:04:05.000"),
			},
		})

		return nil
	})
}

// WakeUpTenant transitions a tenant from WAIT_MODE to ACTIVE and re-queues messages
func (s *Service) WakeUpTenant(ctx context.Context, tenantID uuid.UUID) error {
	return db.RunInTenantScope(tenantID, false, func(tx *gorm.DB) error {
		// 1. Set Tenant back to ACTIVE
		if err := tx.Model(&models.Tenant{}).Where("id = ?", tenantID).Update("status", models.TenantActive).Error; err != nil {
			return err
		}

		// 2. Move WAITING messages back to PENDING so they get picked up by the dispatcher
		if err := tx.Model(&models.Message{}).
			Where("status = ?", models.MessageWait).
			Updates(map[string]interface{}{
				"status":      models.MessagePending,
				"wait_reason": "",
				"updated_at":  time.Now(),
			}).Error; err != nil {
			return err
		}

		return nil
	})
}

// RedeliverQueuedMessages re-broadcasts QUEUED messages to connected mobile devices.
// Called when a mobile device (re)connects via /ws/pair so messages that were
// activated while the device was offline are still delivered. When staleThreshold
// is greater than zero, only messages that have been QUEUED for longer than the
// threshold are re-dispatched (used by the periodic flusher).
func (s *Service) RedeliverQueuedMessages(ctx context.Context, tenantID uuid.UUID, staleThreshold time.Duration) error {
	query := db.Get(ctx).
		Where("tenant_id = ? AND status = ?", tenantID, models.MessageQueued)
	if staleThreshold > 0 {
		query = query.Where("updated_at < ?", time.Now().Add(-staleThreshold))
	}

	var messages []models.Message
	if err := query.Order("created_at ASC").Limit(1000).Find(&messages).Error; err != nil {
		return err
	}

	if len(messages) == 0 || ws.GlobalHub == nil {
		return nil
	}

	log := logger.GetLogger()
	log.Info("Redelivering queued messages to mobile devices",
		zap.String("tenant_id", tenantID.String()),
		zap.Int("message_count", len(messages)))

	for i := range messages {
		msg := &messages[i]
		wsMsg := ws.WebSocketMessage{
			Type:      "MESSAGE_DELIVERY",
			MessageID: msg.ID.String(),
			TenantID:  tenantID,
			Payload: map[string]interface{}{
				"message_id":     msg.ID.String(),
				"content":        msg.Content,
				"sender_name":    msg.SenderName,
				"phone_number":   msg.PhoneNumber,
				"message_type":   string(msg.MessageType),
				"sender_user_id": msg.SenderUserID,
				"timestamp":      time.Now(),
			},
			Timestamp: time.Now(),
		}

		if msg.ChannelID != nil {
			// Target a specific device channel
			wsMsg.ChannelID = *msg.ChannelID
			if err := ws.GlobalHub.SendMessage(ctx, wsMsg); err != nil {
				log.Warn("Failed to redeliver queued message to device",
					zap.String("msg_id", msg.ID.String()),
					zap.String("channel_id", msg.ChannelID.String()),
					zap.Error(err))
			}
			continue
		}

		if err := ws.GlobalHub.SendMessageToTenant(ctx, tenantID, wsMsg); err != nil {
			log.Warn("Failed to redeliver queued message to mobile devices",
				zap.String("msg_id", msg.ID.String()),
				zap.Error(err))
		}
	}

	return nil
}

// FlushStaleQueuedMessages re-dispatches stale QUEUED messages for every tenant
// that currently has connected mobile devices. This keeps delivery working as
// long as a phone is online, without depending on the exact connect moment.
func (s *Service) FlushStaleQueuedMessages(ctx context.Context, staleThreshold time.Duration) error {
	if ws.GlobalHub == nil {
		return nil
	}

	var tenantIDs []uuid.UUID
	if err := db.Get(ctx).
		Model(&models.Message{}).
		Where("status = ? AND updated_at < ?", models.MessageQueued, time.Now().Add(-staleThreshold)).
		Distinct().
		Pluck("tenant_id", &tenantIDs).Error; err != nil {
		return err
	}

	if len(tenantIDs) == 0 {
		return nil
	}

	for _, tenantID := range tenantIDs {
		if len(ws.GlobalHub.GetMobileConnectionsByTenant(tenantID)) == 0 {
			continue
		}
		if err := s.RedeliverQueuedMessages(ctx, tenantID, staleThreshold); err != nil {
			logger.GetLogger().Warn("Failed to flush stale queued messages",
				zap.String("tenant_id", tenantID.String()),
				zap.Error(err))
		}
	}

	return nil
}

// StartQueuedMessageFlusher runs a background loop that periodically re-dispatches
// stale QUEUED messages to connected mobile devices.
func StartQueuedMessageFlusher(ctx context.Context, interval, staleThreshold time.Duration) {
	log := logger.GetLogger()
	go func() {
		svc := NewService()
		log.Info("Queued message flusher started",
			zap.Duration("interval", interval),
			zap.Duration("stale_threshold", staleThreshold))
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Info("Queued message flusher stopped")
				return
			case <-ticker.C:
				if err := svc.FlushStaleQueuedMessages(ctx, staleThreshold); err != nil {
					log.Warn("Queued message flusher iteration failed", zap.Error(err))
				}
			}
		}
	}()
}

// GetTenantIDByAPIKey validates API key and returns tenant ID and device ID
func (s *Service) GetTenantIDByAPIKey(ctx context.Context, apiKey string, deviceUUID string) (uuid.UUID, uuid.UUID, error) {
	var device models.PairedDevice

	// Find device by hardware_id (device_uuid from mobile app)
	if err := db.Get(ctx).Where("hardware_id = ?", deviceUUID).First(&device).Error; err != nil {
		return uuid.Nil, uuid.Nil, gorm.ErrRecordNotFound
	}

	// Verify API key matches (hash and compare or direct match if already hashed)
	hashedKey := auth.HashKey(apiKey)
	if hashedKey != device.DevicePairKey && apiKey != device.DevicePairKey {
		return uuid.Nil, uuid.Nil, gorm.ErrInvalidData
	}

	channelID := device.ID
	if device.ChannelID != nil {
		channelID = *device.ChannelID
	}

	// Ensure the tenant has a PHONES routing config so dispatched messages
	// actually flow through the mobile fast path
	if err := s.EnsureCompanyStatus(ctx, device.TenantID); err != nil {
		logger.GetLogger().Warn("Failed to ensure company status during WS pairing",
			zap.String("tenant_id", device.TenantID.String()),
			zap.Error(err))
	}

	return device.TenantID, channelID, nil
}

// GetTenantByDeviceID identifies a previously paired device by its id OR
// hardware_id (the mobile app's reconnection path sends deviceId without an
// api_key). Returns the tenant and the device's dispatch channel id.
func (s *Service) GetTenantByDeviceID(ctx context.Context, deviceID string) (uuid.UUID, uuid.UUID, error) {
	var device models.PairedDevice
	if err := db.Get(ctx).
		Where("hardware_id = ? OR id = ?", deviceID, deviceID).
		First(&device).Error; err != nil {
		return uuid.Nil, uuid.Nil, gorm.ErrRecordNotFound
	}

	channelID := device.ID
	if device.ChannelID != nil {
		channelID = *device.ChannelID
	}

	if err := s.EnsureCompanyStatus(ctx, device.TenantID); err != nil {
		logger.GetLogger().Warn("Failed to ensure company status during WS pairing",
			zap.String("tenant_id", device.TenantID.String()),
			zap.Error(err))
	}

	return device.TenantID, channelID, nil
}

// generatePairKey creates a unique key for device pairing
func generatePairKey() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// EnsureCompanyStatus creates a default PHONES routing config for a tenant if none exists.
// Without a company_statuses row, messages bypass the mobile dispatch fast path entirely.
func (s *Service) EnsureCompanyStatus(ctx context.Context, tenantID uuid.UUID) error {
	var count int64
	if err := db.Get(ctx).Model(&models.CompanyStatus{}).Where("tenant_id = ?", tenantID).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return db.Get(ctx).Create(&models.CompanyStatus{
		TenantID:    tenantID,
		IsActive:    true,
		ChannelType: "PHONES",
	}).Error
}

// DeleteDevice removes a device by its UUID
func (s *Service) DeleteDevice(ctx context.Context, tenantID uuid.UUID, deviceID uuid.UUID) error {
	return db.Get(ctx).Transaction(func(tx *gorm.DB) error {
		var device models.PairedDevice
		if err := tx.Where("id = ? AND tenant_id = ?", deviceID, tenantID).First(&device).Error; err != nil {
			return err
		}

		// Optional: mark channel as offline if it was linked
		if device.ChannelID != nil {
			tx.Model(&models.DispatchChannel{}).Where("id = ?", *device.ChannelID).Update("status", "OFFLINE")
		}

		return tx.Delete(&device).Error
	})
}

// DeleteDeviceByHardwareID removes a device by its hardware identifier
func (s *Service) DeleteDeviceByHardwareID(ctx context.Context, tenantID uuid.UUID, hardwareID string) error {
	return db.Get(ctx).Transaction(func(tx *gorm.DB) error {
		var device models.PairedDevice
		if err := tx.Where("hardware_id = ? AND tenant_id = ?", hardwareID, tenantID).First(&device).Error; err != nil {
			return err
		}

		if device.ChannelID != nil {
			tx.Model(&models.DispatchChannel{}).Where("id = ?", *device.ChannelID).Update("status", "OFFLINE")
		}

		return tx.Delete(&device).Error
	})
}

// PingDevice checks if a device is currently connected via WebSocket
func (s *Service) PingDevice(ctx context.Context, tenantID uuid.UUID, idStr string) (string, error) {
	var device models.PairedDevice
	
	query := db.Get(ctx).Where("tenant_id = ?", tenantID)
	
	if deviceID, err := uuid.Parse(idStr); err == nil {
		query = query.Where("id = ?", deviceID)
	} else {
		query = query.Where("hardware_id = ?", idStr)
	}

	if err := query.First(&device).Error; err != nil {
		return "OFFLINE", err
	}

	if device.ChannelID != nil && ws.GlobalHub.IsChannelConnected(tenantID, *device.ChannelID) {
		return "ONLINE", nil
	}

	return "OFFLINE", nil
}
