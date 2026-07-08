package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/al-hudhud/backend/internal/balance"
	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/al-hudhud/backend/internal/provider"
	"github.com/al-hudhud/backend/internal/redis"
	"github.com/al-hudhud/backend/internal/router"
	"github.com/al-hudhud/backend/internal/ws"
	"github.com/al-hudhud/backend/pkg/logger"
	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// SendSMSArgs defines the arguments for the SendSMS job
type SendSMSArgs struct {
	TenantID  uuid.UUID `json:"tenant_id"`
	MessageID uuid.UUID `json:"message_id"`
}

func (SendSMSArgs) Kind() string { return "send_sms" }

// AnalysisArgs defines arguments for Phase 2: parse a single incoming message (Legacy)
type AnalysisArgs struct {
	TenantID          uuid.UUID `json:"tenant_id"`
	IncomingMessageID uuid.UUID `json:"incoming_message_id"`
}

func (AnalysisArgs) Kind() string { return "message_analysis" }

// StatsArgs defines arguments for periodic statistics aggregation
type StatsArgs struct {
	BatchSize int `json:"batch_size"` // Optional
}

func (StatsArgs) Kind() string { return "stats_aggregation" }

// SendSMSWorker handles processing of SendSMS messages
type SendSMSWorker struct {
	river.WorkerDefaults[SendSMSArgs]
}

func (w *SendSMSWorker) Work(ctx context.Context, job *river.Job[SendSMSArgs]) error {
	tenantID := job.Args.TenantID
	msgID := job.Args.MessageID

	// Initialize Engine
	routingEngine := router.NewRoutingEngine()
	// Use the global provider registry
	registry := provider.GetGlobalRegistry()

	return db.RunInTenantScope(tenantID, false, func(tx *gorm.DB) error {
		var msg models.Message
		if err := tx.First(&msg, "id = ?", msgID).Error; err != nil {
			return fmt.Errorf("failed to find message %s: %w", msgID, err)
		}

		// 1. ROUTING: Find the best channel
		channel, err := routingEngine.GetBestChannel(ctx, tenantID, &msg)
		if err != nil {
			if err.Error() == "WAIT_MODE_REQUIRED" {
				// Enter Wait Mode: Update message status and stop retrying for now
				tx.Model(&msg).
					Where("id = ? AND status = ?", msg.ID, models.MessageQueued).
					Updates(map[string]interface{}{
						"status":      models.MessageWait,
						"wait_reason": "All dispatch channels are OFFLINE",
						"updated_at":  time.Now(),
					})
				// Also update tenant status
				tx.Model(&models.Tenant{}).Where("id = ?", tenantID).Update("status", models.TenantWaitMode)
				return nil // Job is "done" for now, will be re-queued when device wakes up
			}
			return err
		}

		// 2. ACCOUNTING PHASE 1: Deduct/Reserve balance
		cost := 1.0
		var tenant models.Tenant
		if err := tx.First(&tenant, "id = ?", tenantID).Error; err != nil {
			return err
		}

		if tenant.Balance < cost {
			tx.Model(&msg).
				Where("id = ? AND status = ?", msg.ID, models.MessageQueued).
				Update("status", models.MessageFailed)
			return fmt.Errorf("insufficient balance for tenant %s", tenantID)
		}

		// Record the transaction (PENDING)
		transaction := models.AccountingTransaction{
			TenantID:  tenantID,
			MessageID: msg.ID,
			Amount:    cost,
			Status:    models.TxPending,
		}
		if err := tx.Create(&transaction).Error; err != nil {
			return err
		}

		// Temporarily deduct from balance
		tx.Model(&tenant).Update("balance", tenant.Balance-cost)

		// 3. DISPATCH: Send to selected provider
		p := registry.Get(models.ChannelType(channel.Type))
		if p == nil {
			return fmt.Errorf("provider not found for type %s", channel.Type)
		}

		jobID, err := p.Send(ctx, tenantID, &msg, channel.Config)
		if err != nil {
			// Refund Phase 1 if dispatch fails immediately
			tx.Model(&tenant).Update("balance", tenant.Balance+cost)
			tx.Model(&transaction).Update("status", models.TxRefunded)
			return err
		}

		// 4. Update message status to PROCESSING and wait for ACK
		err = tx.Model(&msg).
			Where("id = ? AND status = ?", msg.ID, models.MessageQueued).
			Updates(map[string]interface{}{
				"channel_id":      channel.ID,
				"status":          models.MessageProcessing,
				"provider_job_id": jobID,
				"cost":            cost,
				"updated_at":      time.Now(),
			}).Error
		if err != nil {
			return err
		}

		// Broadcast 3: DISPATCHED & METRICS_UPDATED
		ws.GlobalHub.Broadcast(ws.Event{
			TenantID: tenantID,
			Type:     "MESSAGE_DISPATCHED",
			Payload: map[string]interface{}{
				"message_id": msg.ID,
				"status":     "PROCESSING",
			},
		})

		ws.GlobalHub.Broadcast(ws.Event{
			TenantID: tenantID,
			Type:     "METRICS_UPDATED",
			Payload:  map[string]interface{}{"reason": "message_dispatched"},
		})

		return nil
	})
}

// GlobalEventArgs defines arguments for Phase 2: parse a global raw event
type GlobalEventArgs struct {
	GlobalEventID int64     `json:"global_event_id"`
	TenantID      uuid.UUID `json:"tenant_id"`
}

func (GlobalEventArgs) Kind() string { return "global_event_parse" }

type BulkMessagePayload struct {
	MessageType    string              `json:"message_type"`
	SenderName     string              `json:"sender_name"`
	SenderUserID   *uuid.UUID          `json:"sender_user_id"`
	SenderDeviceID *uuid.UUID          `json:"sender_device_id"`
	Messages       []SingleMessageItem `json:"messages"`
}

type SingleMessageItem struct {
	PhoneNumber string     `json:"phone_number"`
	Content     string     `json:"content"`
	ScheduledAt *time.Time `json:"scheduled_at"`
}

// GlobalEventWorker handles Phase 2: parse raw JSON events into message rows
type GlobalEventWorker struct {
	river.WorkerDefaults[GlobalEventArgs]
}

func (w *GlobalEventWorker) Work(ctx context.Context, job *river.Job[GlobalEventArgs]) error {
	tenantID := job.Args.TenantID
	eventID := job.Args.GlobalEventID

	return db.RunInTenantScope(tenantID, false, func(tx *gorm.DB) error {
		// 1. Load the raw event
		var event models.GlobalIncomingEvent
		if err := db.GormDB.First(&event, eventID).Error; err != nil {
			return fmt.Errorf("event not found: %w", err)
		}

		// 2. Parse the raw JSON
		var payload BulkMessagePayload
		if err := json.Unmarshal([]byte(event.RawPayload), &payload); err != nil {
			db.GormDB.Model(&event).Update("status", "FAILED")
			return fmt.Errorf("parse payload: %w", err)
		}

		// 3. Determine message category
		msgType := models.MsgCatOther
		switch payload.MessageType {
		case "INVOICE": msgType = models.MsgCatInvoice
		case "PAYMENT": msgType = models.MsgCatPayment
		case "ALERT":   msgType = models.MsgCatAlert
		case "REVERSAL": msgType = models.MsgCatReversal
		case "PROMO":    msgType = models.MsgCatPromo
		case "OTP":      msgType = models.MsgCatOTP
		case "REMINDER": msgType = models.MsgCatReminder
		}

		// 4. Create MessageEvent for structured tracking (Requirement 3007)
		msgEvent := models.MessageEvent{
			ID:          uuid.New(),
			TenantID:    tenantID,
			Name:        fmt.Sprintf("Bulk Ingestion %s", time.Now().Format("2006-01-02 15:04")),
			Source:      "BULK_API",
			Status:      models.EventProcessing,
			MessageType: payload.MessageType,
			TotalItems:  len(payload.Messages),
			StartedAt:   nil, // Will be set when first msg starts
			CreatedAt:   time.Now(),
		}
		if err := tx.Create(&msgEvent).Error; err != nil {
			return fmt.Errorf("create message event: %w", err)
		}

		// 5. Initialize Redis ID Generator for message IDs
		redisClient := redis.GetGlobalClient() // Assuming global redis client exists
		if redisClient == nil {
			return fmt.Errorf("redis client not available for ID generation")
		}
		idGen := redis.NewIDGenerator(redisClient)

		// 6. Create message rows with custom IDs
		var messages []models.Message
		for _, item := range payload.Messages {
			charCount, partsCount, encoding, cost := calcMessagePricing(item.Content)
			
			scheduledAt := time.Now()
			if item.ScheduledAt != nil {
				scheduledAt = *item.ScheduledAt
			}

			// Generate custom message ID linked to event
			customID, err := idGen.GenerateMessageID(ctx, event.CustomID)
			if err != nil {
				return fmt.Errorf("failed to generate message ID: %w", err)
			}

			msg := models.Message{
				ID:             uuid.New(),
				CustomID:       customID,
				EventCustomID:  event.CustomID,
				TenantID:       tenantID,
				GlobalEventID:  &eventID,
				EventID:        &msgEvent.ID,
				SenderName:     payload.SenderName,
				SenderUserID:   payload.SenderUserID,
				SenderDeviceID: payload.SenderDeviceID,
				MessageType:    msgType,
				PhoneNumber:    item.PhoneNumber,
				Content:        item.Content,
				CharCount:      charCount,
				PartsCount:     partsCount,
				Encoding:       encoding,
				Cost:           cost,
				Status:         models.MessagePending,
				ScheduledAt:    scheduledAt,
				CreatedAt:      time.Now(),
			}
			messages = append(messages, msg)
		}

		// 7. Bulk insert messages
		if len(messages) > 0 {
			if err := tx.CreateInBatches(messages, 100).Error; err != nil {
				return fmt.Errorf("bulk insert messages: %w", err)
			}
		}

		// 8. Mark Legacy Event as PROCESSING 
		tx.Model(&models.GlobalIncomingEvent{}).Where("id = ?", eventID).Updates(map[string]interface{}{
			"status":                "PROCESSING",
			"total_items":           len(messages),
			"processing_started_at": time.Now(),
			"updated_at":           time.Now(),
		})

		// Phase 14: Suspicious Activity & Bulk Alerting
		if len(messages) > 50000 {
			alert := models.SystemAlert{
				ID:        uuid.New(),
				TenantID:  tenantID,
				Severity:  models.SeverityCritical,
				Title:     "Huge Bulk Ingestion Detected",
				Message:   fmt.Sprintf("Tenant %s is ingesting a massive batch of %d messages. This may impact system performance.", tenantID, len(messages)),
				CreatedAt: time.Now(),
			}
			tx.Create(&alert)
		}

		// 9. Phase 3: Trigger Activation for each created message
		// We use InitTenantClient to get a worker-safe client for insertion.
		client, err := InitTenantClient(ctx, tenantID, nil)
		if err == nil {
			for _, msg := range messages {
				_, _ = client.Insert(ctx, MessageActivationArgs{
					MessageID: msg.ID,
					TenantID:  tenantID,
				}, nil)
			}
		}

		return nil
	})
}

// calcMessagePricing detects encoding and calculates parts count for a message
func calcMessagePricing(content string) (charCount int, partsCount int, encoding string, cost float64) {
	if content == "" { return 0, 1, "GSM7", 0.05 }
	charCount = len([]rune(content))
	isUnicode := false
	for _, r := range content {
		if r > 127 { isUnicode = true; break }
	}
	if isUnicode {
		encoding = "UNICODE"
		if charCount <= 70 { partsCount = 1 } else { partsCount = (charCount + 66) / 67 }
	} else {
		encoding = "GSM7"
		if charCount <= 160 { partsCount = 1 } else { partsCount = (charCount + 152) / 153 }
	}
	cost = float64(partsCount) * 0.05
	return charCount, partsCount, encoding, cost
}

// MessageActivationArgs defines arguments for Phase 3 activation
type MessageActivationArgs struct {
	MessageID uuid.UUID `json:"message_id"`
	TenantID  uuid.UUID `json:"tenant_id"`
}

func (MessageActivationArgs) Kind() string { return "message_activation" }

// MessageActivationWorker handles atomic balance deduction and Fast Path dispatch
type MessageActivationWorker struct {
	river.WorkerDefaults[MessageActivationArgs]
	Redis *redis.Client
}

func (w *MessageActivationWorker) Work(ctx context.Context, job *river.Job[MessageActivationArgs]) error {
	msgID := job.Args.MessageID
	tenantID := job.Args.TenantID
	log := logger.GetLogger()

	// Fetch the message from database
	var msg models.Message
	if err := db.GormDB.First(&msg, "id = ?", msgID).Error; err != nil {
		log.Error("Message not found for activation", zap.String("msg_id", msgID.String()), zap.Error(err))
		return nil // Don't retry if message not found
	}

	// Use pre-calculated cost or calculate fallback
	totalCost := decimal.NewFromFloat(msg.Cost)
	if totalCost.IsZero() {
		parts := msg.PartsCount
		if parts == 0 {
			parts = 1
		}
		totalCost = decimal.NewFromFloat(0.05).Mul(decimal.NewFromInt(int64(parts)))
	}

	// EXECUTE PHASE 3 CORE ATOMIC TRANSACTION INSIDE RAW DB TRANSACTION
	err := db.RunInRawTx(ctx, func(tx db.RawTx) error {
		return balance.DeductAndActivateMessage(ctx, tx, tenantID, msgID, totalCost)
	})

	// PHASE 7: Update Parent Event Progress (Requirement 3046)
	if msg.EventID != nil {
		incrementField := "processed_items"
		if err != nil {
			incrementField = "failed_items"
		}
		
		db.GormDB.Model(&models.MessageEvent{}).
			Where("id = ?", *msg.EventID).
			Updates(map[string]interface{}{
				incrementField: gorm.Expr(incrementField + " + 1"),
				"updated_at":   time.Now(),
			})

		// Check if completed
		var evt models.MessageEvent
		if err := db.GormDB.First(&evt, *msg.EventID).Error; err == nil {
			if evt.ProcessedItems+evt.FailedItems >= evt.TotalItems && evt.TotalItems > 0 {
				db.GormDB.Model(&evt).Updates(map[string]interface{}{
					"status":      models.EventCompleted,
					"finished_at": time.Now(),
				})
			}
		}
	}

	if err != nil {
		log.Error("Message activation failed (insufficient balance or DB error)", zap.String("msg_id", msgID.String()), zap.Error(err))
		return err
	}

	// Fast Path Dispatch via Redis (existing functionality)
	if w.Redis != nil {
		// Use deviceID if set, otherwise broadcast to all online connections
		if msg.DeviceID != nil {
			_ = w.Redis.PublishMessage(ctx, *msg.DeviceID, map[string]interface{}{
				"type":       "NEW_MESSAGE",
				"message_id": msg.ID,
				"content":    msg.Content,
				"recipient":  msg.PhoneNumber,
				"parts":      msg.PartsCount,
				"timestamp":  time.Now(),
			})
		}
	}

	// Mobile WebSocket Dispatch
	// Dispatch message to ALL connected mobile devices for this tenant
	if ws.GlobalHub != nil {
		connections := ws.GlobalHub.GetMobileConnectionsByTenant(tenantID)

		if len(connections) > 0 {
			wsMsg := ws.WebSocketMessage{
				Type:      "MESSAGE_DELIVERY",
				MessageID: msg.ID.String(),
				TenantID:  tenantID,
				Payload: map[string]interface{}{
					"message_id":   msg.ID.String(),
					"content":      msg.Content,
					"sender_name":  msg.SenderName,
					"phone_number": msg.PhoneNumber,
					"message_type": string(msg.MessageType),
					"sender_user_id": msg.SenderUserID,
					"timestamp":    time.Now(),
				},
				Timestamp: time.Now(),
			}

			for _, conn := range connections {
				wsMsgCopy := wsMsg
				wsMsgCopy.ChannelID = conn.ChannelID

				if err := ws.GlobalHub.SendMessage(ctx, wsMsgCopy); err != nil {
					log.Error("Failed to send message to mobile device",
						zap.String("msg_id", msgID.String()),
						zap.String("device_id", conn.DeviceID),
						zap.Error(err))
				} else {
					log.Info("Message dispatched to mobile device",
						zap.String("msg_id", msgID.String()),
						zap.String("device_id", conn.DeviceID),
						zap.String("channel_id", conn.ChannelID.String()))
				}

				deliveryLog := models.MessageDeliveryLog{
					ID:              uuid.New(),
					MessageID:       msg.ID,
					TenantID:        tenantID,
					DeviceID:        uuid.MustParse(conn.DeviceID),
					ChannelID:       conn.ChannelID,
					DeliveryChannel: "MOBILE",
					Delivered:       err == nil,
					DeliveredAt:     time.Now(),
				}
				if dbErr := db.GormDB.Create(&deliveryLog).Error; dbErr != nil {
					log.Error("Failed to create delivery log",
						zap.String("msg_id", msgID.String()),
						zap.Error(dbErr))
				}
			}
		} else {
			log.Debug("No connected mobile devices for tenant, message marked for pending delivery",
				zap.String("msg_id", msgID.String()),
				zap.String("tenant_id", tenantID.String()))
		}
	}

	return nil
}

// AnalysisWorker handles parsing of raw single messages (Legacy Phase 2)
type AnalysisWorker struct {
	river.WorkerDefaults[AnalysisArgs]
}

func (w *AnalysisWorker) Work(ctx context.Context, job *river.Job[AnalysisArgs]) error {
	tenantID := job.Args.TenantID
	msgID := job.Args.IncomingMessageID

	return db.RunInTenantScope(tenantID, false, func(tx *gorm.DB) error {
		var incoming models.IncomingMessage
		if err := tx.First(&incoming, "id = ?", msgID).Error; err != nil {
			return err
		}

		// Compute pricing & create message (simpler version for legacy)
		charCount, partsCount, encoding, cost := calcMessagePricing(incoming.MessageContent)

		// Phase 4: Create MessageEvent for reporting
		msgEvent := models.MessageEvent{
			ID:          uuid.New(),
			TenantID:    tenantID,
			Name:        fmt.Sprintf("Legacy Inbound: %s", incoming.SenderName),
			Source:      "LEGACY_INGESTION",
			Status:      models.EventCompleted,
			MessageType: "INBOUND_TEXT",
			TotalItems:  1,
			ProcessedItems: 1,
			CreatedAt:   time.Now(),
		}
		if err := tx.Create(&msgEvent).Error; err != nil {
			return fmt.Errorf("create message event: %w", err)
		}

		// Phase 4: Generate Custom IDs
		idGen := redis.NewIDGenerator(redis.GetGlobalClient())
		eventCustomID, _ := idGen.GenerateEventID(ctx, tenantID, "ANALYSIS_SERVICE")
		customID, _ := idGen.GenerateMessageID(ctx, eventCustomID)

		message := &models.Message{
			ID:                uuid.New(),
			CustomID:          customID,
			EventCustomID:     eventCustomID,
			EventID:           &msgEvent.ID, // Link to the UUID event
			TenantID:          tenantID,
			IncomingMessageID: &incoming.ID,
			SenderName:        incoming.SenderName,
			PhoneNumber:       incoming.PhoneNumber,
			Content:           incoming.MessageContent,
			CharCount:         charCount,
			PartsCount:        partsCount,
			Encoding:          encoding,
			Cost:              cost,
			Status:            models.MessagePending,
			CreatedAt:         time.Now(),
		}

		if err := tx.Create(message).Error; err != nil {
			return err
		}
		
		tx.Model(&incoming).Update("is_processed", true)

		// Trigger activation job
		client, err := InitTenantClient(ctx, tenantID, nil)
		if err == nil {
			client.Insert(ctx, &MessageActivationArgs{
				MessageID: message.ID,
				TenantID:  tenantID,
			}, nil)
		}

		return nil
	})
}

func (MessageActivationWorker) Kind() string { return "message_activation" }

// DLRProcessArgs defines arguments for Phase 8: Durable DLR processing
type DLRProcessArgs struct {
	MessageID  uuid.UUID            `json:"message_id"`
	NewStatus  models.MessageStatus `json:"new_status"`
	ExternalID string               `json:"external_id"`
	TenantID   uuid.UUID            `json:"tenant_id"`
}

func (DLRProcessArgs) Kind() string { return "dlr_process" }

// DLRWorker handles background processing of Delivery Reports (Phase 8)
type DLRWorker struct {
	river.WorkerDefaults[DLRProcessArgs]
}

func (w *DLRWorker) Work(ctx context.Context, job *river.Job[DLRProcessArgs]) error {
	msgID := job.Args.MessageID
	newStatus := job.Args.NewStatus
	externalID := job.Args.ExternalID
	tenantID := job.Args.TenantID
	now := time.Now()

	log := logger.GetLogger()

	return db.RunInTenantScope(tenantID, false, func(tx *gorm.DB) error {
		// Single indexed UPDATE on message_id primary key – O(1)
		updates := map[string]interface{}{
			"status":     newStatus,
			"updated_at": now,
		}
		if newStatus == models.MessageSent {
			updates["sent_at"] = now
		}
		if newStatus == models.MessageDelivered {
			updates["delivered_at"] = now
		}
		if externalID != "" {
			updates["provider_job_id"] = externalID
		}

		result := tx.Model(&models.Message{}).
			Where("id = ? AND status NOT IN (?, ?)", msgID, models.MessageDelivered, models.MessageFailed).
			Updates(updates)

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected > 0 {
			// Broadcast real-time update to the tenant's UI
			ws.GlobalHub.Broadcast(ws.Event{
				TenantID: tenantID,
				Type:     "MESSAGE_STATUS_UPDATE",
				Payload: map[string]interface{}{
					"message_id": msgID,
					"status":     newStatus,
					"updated_at": now.Format("15:04:05.000"),
				},
			})
			log.Info("DLR applied via Worker", zap.String("msg_id", msgID.String()), zap.String("status", string(newStatus)))
		}

		return nil
	})
}
