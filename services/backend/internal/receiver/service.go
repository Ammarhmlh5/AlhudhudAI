package receiver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/dispatcher"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/al-hudhud/backend/internal/queue"
	"github.com/al-hudhud/backend/internal/redis"
	"github.com/al-hudhud/backend/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Service struct{
	redisClient        *redis.Client
	messageQueue       *redis.MessageQueue
	pricingCache       *redis.PricingCache
	channelCache       *redis.ChannelCache
	phonesDispatcher   *PhonesDispatcher
	alhudhudDispatcher *AlhudhudDispatcher
	privateDispatcher  *PrivateDispatcher
}

func NewService(redisClient *redis.Client) *Service {
	var messageQueue *redis.MessageQueue
	var pricingCache *redis.PricingCache
	var channelCache *redis.ChannelCache
	
	if redisClient != nil {
		messageQueue = redis.NewMessageQueue(redisClient)
		pricingCache = redis.NewPricingCache(redisClient)
		channelCache = redis.NewChannelCache(redisClient)
	}
	
	return &Service{
		redisClient:        redisClient,
		messageQueue:       messageQueue,
		pricingCache:       pricingCache,
		channelCache:       channelCache,
		phonesDispatcher:   NewPhonesDispatcher(redisClient),
		alhudhudDispatcher: NewAlhudhudDispatcher(redisClient),
		privateDispatcher:  NewPrivateDispatcher(redisClient),
	}
}

// PaginatedMessages represents a paginated list of messages
type PaginatedMessages struct {
	Messages   []models.Message `json:"messages"`
	TotalCount int64            `json:"total_count"`
	TotalPages int              `json:"total_pages"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
}

// MessageFilters holds all supported filter dimensions for listing messages
type MessageFilters struct {
	Status      models.MessageStatus
	DateFrom    string // "YYYY-MM-DD"
	DateTo      string // "YYYY-MM-DD"
	PhoneNumber string
	SenderName  string
	Source      string // WEB_UI, WEB_API, ACCOUNTING, MOBILE
	MessageType string // INVOICE, PAYMENT, ALERT, etc.
	ChannelID   string // UUID string
}

// StageMessage validates and saves a message to the database with Redis queue optimization
func (s *Service) StageMessage(ctx context.Context, tenantIDStr string, phoneNumber string, content string, scheduledAt *time.Time, idempotencyKey string, messageType models.MessageCategory, senderUserID *uuid.UUID, senderDeviceID *uuid.UUID) (*models.Message, error) {
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return nil, errors.New("invalid tenant id format")
	}

	// Use default message type if not provided
	if messageType == "" {
		messageType = models.MsgCatOther
	}

	// Phase 3.2: Idempotency Check (Redis first for speed)
	if idempotencyKey != "" && s.redisClient != nil {
		idempKey := fmt.Sprintf("%s:idempotency:%s:%s", "alhudhud", tenantID.String(), idempotencyKey)
		exists, _ := s.redisClient.Client.Exists(ctx, idempKey).Result()
		if exists > 0 {
			// Get existing message ID from Redis
			existingIDStr, _ := s.redisClient.Client.Get(ctx, idempKey).Result()
			if existingIDStr != "" {
				existingID, _ := uuid.Parse(existingIDStr)
				var existing models.Message
				if err := db.Get(ctx).Where("id = ?", existingID).First(&existing).Error; err == nil {
					return &existing, nil
				}
			}
		}
	}

	// Basic validation
	if phoneNumber == "" {
		return nil, errors.New("phone number is required")
	}
	if content == "" {
		return nil, errors.New("message content is required")
	}

	// Phase 3: Check Company Status for Fast Ingestion
	var companyStatus models.CompanyStatus
	isActive := false
	if err := db.Get(ctx).Where("tenant_id = ?", tenantID).First(&companyStatus).Error; err == nil {
		isActive = companyStatus.IsActive
	}

	var charCount, partsCount int
	var encoding string
	var cost float64

	if !isActive {
		// Calculate pricing only if not fast-path
		charCount, partsCount, encoding, cost = s.calculatePricing(ctx, content)
	} else {
		// Default dummy values for fast path to avoid expensive calculations
		charCount = len(content)
		partsCount = 1
		encoding = "GSM7"
		cost = 0.0
	}

	// Phase 4: Generate Custom IDs
	var customID, eventCustomID string
	if s.redisClient != nil {
		idGen := redis.NewIDGenerator(s.redisClient)
		var err error
		eventCustomID, err = idGen.GenerateEventID(ctx, tenantID, "SYSTEM") // Single messages get a system-generated event
		if err != nil {
			return nil, fmt.Errorf("failed to generate event ID: %w", err)
		}
		customID, err = idGen.GenerateMessageID(ctx, eventCustomID)
		if err != nil {
			return nil, fmt.Errorf("failed to generate message ID: %w", err)
		}
	} else {
		// Fallback for tests or when Redis is down
		eventCustomID = "EV_FALLBACK_" + uuid.New().String()[:8]
		customID = "MS_FALLBACK_" + uuid.New().String()[:8]
	}

	source, _ := ctx.Value("request_source").(string)
	if source == "" {
		source = "UNKNOWN"
	}

	msg := &models.Message{
		ID:             uuid.New(),
		CustomID:       customID,
		EventCustomID:  eventCustomID,
		TenantID:       tenantID,
		PhoneNumber:    phoneNumber,
		Content:        content,
		MessageType:    messageType,
		SenderUserID:   senderUserID,
		SenderDeviceID: senderDeviceID,
		CharCount:      charCount,
		PartsCount:     partsCount,
		Encoding:       encoding,
		Cost:           cost,
		Status:         models.MessagePending,
		Source:         source,
		IdempotencyKey: idempotencyKey,
		CreatedAt:      time.Now(),
	}

	if scheduledAt != nil {
		msg.ScheduledAt = *scheduledAt
	} else {
		msg.ScheduledAt = time.Now()
	}

	// Store idempotency key in Redis for fast lookup
	if idempotencyKey != "" && s.redisClient != nil {
		idempKey := fmt.Sprintf("%s:idempotency:%s:%s", "alhudhud", tenantID.String(), idempotencyKey)
		s.redisClient.Client.Set(ctx, idempKey, msg.ID.String(), 24*time.Hour)
	}

	// Phase 3: Strictly Isolated Fast-Path Dispatch
	if isActive {
		msg.Status = models.MessageProcessing
		dispatched := false

		switch companyStatus.ChannelType {
		case "PHONES":
			ok, err := s.phonesDispatcher.DispatchToPhone(ctx, tenantID, companyStatus.DeviceID, msg)
			if err != nil {
				logger.GetLogger().Warn("PHONES dispatch failed, falling back to queue", zap.Error(err))
			} else {
				dispatched = ok
			}

		case "ALHUDHUD":
			ok, err := s.alhudhudDispatcher.DispatchToAlhudhud(ctx, tenantID, msg)
			if err != nil {
				logger.GetLogger().Warn("ALHUDHUD dispatch failed, falling back to queue", zap.Error(err))
			} else {
				dispatched = ok
			}

		case "PRIVATE":
			ok, err := s.privateDispatcher.DispatchToPrivateServer(ctx, tenantID, companyStatus.PrivateServerURL, msg)
			if err != nil {
				logger.GetLogger().Warn("PRIVATE dispatch failed, falling back to queue", zap.Error(err))
			} else {
				dispatched = ok
			}
		}

		if !dispatched {
			msg.Status = models.MessagePending
		}
	}

	// Save/Create in PostgreSQL database after dispatch attempt (fast ingestion)
	if err := db.Get(ctx).Create(msg).Error; err != nil {
		return nil, err
	}

	if cost > 0 {
		transaction := &models.AccountingTransaction{
			ID:        uuid.New(),
			TenantID:  tenantID,
			MessageID: msg.ID,
			Amount:    cost,
			Status:    models.TxPending,
			Type:      "DEDUCTION",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := db.Get(ctx).Create(transaction).Error; err != nil {
			logger.GetLogger().Warn("Failed to create accounting transaction at staging",
				zap.String("message_id", msg.ID.String()),
				zap.Error(err),
			)
		}
	}

	// Enqueue to Redis for standard processing (Fallback or inactive company)
	if msg.Status == models.MessagePending && s.messageQueue != nil {
		queueItem := &redis.MessageQueueItem{
			MessageID:   msg.ID,
			TenantID:    tenantID,
			PhoneNumber: phoneNumber,
			Content:     content,
			PartsCount:  partsCount,
			Cost:        cost,
			Priority:    2, // Normal priority
			EnqueuedAt:  time.Now(),
		}
		
		if err := s.messageQueue.Enqueue(ctx, queueItem); err != nil {
			logger.GetLogger().Error("Failed to enqueue message to Redis",
				zap.String("message_id", msg.ID.String()),
				zap.Error(err),
			)
		}
	}

	return msg, nil
}

type BulkMessageItem struct {
	PhoneNumber string     `json:"phone_number" binding:"required"`
	Content     string     `json:"content" binding:"required"`
	ScheduledAt *time.Time `json:"scheduled_at"`
}

// StageBulkMessages handles bulk message staging
func (s *Service) StageBulkMessages(ctx context.Context, tenantIDStr string, messages []BulkMessageItem, baseIdempotencyKey string) ([]models.Message, error) {
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return nil, errors.New("invalid tenant id format")
	}

	var stagedMessages []models.Message
	for i, m := range messages {
		idempotencyKey := ""
		if baseIdempotencyKey != "" {
			idempotencyKey = fmt.Sprintf("%s:%d", baseIdempotencyKey, i)
			
			// Simple cross-check for existence of the first message to return early if already processed
			if i == 0 {
				var existing models.Message
				if err := db.Get(ctx).Where("tenant_id = ? AND idempotency_key = ?", tenantID, idempotencyKey).First(&existing).Error; err == nil {
					// Return empty or existing - for simplicity we just proceed but GORM will fail on unique constraint if we don't handle it
					// But we should ideally return the already staged batch
				}
			}
		}

		// Phase 4: Generate Custom IDs for each message in the batch
		var customID, eventCustomID string
		if s.redisClient != nil {
			idGen := redis.NewIDGenerator(s.redisClient)
			// For now, let's generate a unique event ID per bulk call if not provided
			eventCustomID, _ = idGen.GenerateEventID(ctx, tenantID, "BULK_API")
			customID, _ = idGen.GenerateMessageID(ctx, eventCustomID)
		} else {
			eventCustomID = "EV_BULK_FALLBACK_" + uuid.New().String()[:8]
			customID = "MS_BULK_FALLBACK_" + uuid.New().String()[:8]
		}

		// Calculate pricing for bulk item
		charCount, partsCount, encoding, cost := s.calculatePricing(ctx, m.Content)

		msg := models.Message{
			ID:             uuid.New(),
			CustomID:       customID,
			EventCustomID:  eventCustomID,
			TenantID:       tenantID,
			PhoneNumber:    m.PhoneNumber,
			Content:        m.Content,
			CharCount:      charCount,
			PartsCount:     partsCount,
			Encoding:       encoding,
			Cost:           cost,
			Status:         models.MessagePending,
			Source:         "WEB_UI", // Bulk staged is usually from Web UI
			IdempotencyKey: idempotencyKey,
			CreatedAt:      time.Now(),
		}
		if m.ScheduledAt != nil {
			msg.ScheduledAt = *m.ScheduledAt
		} else {
			msg.ScheduledAt = time.Now()
		}
		stagedMessages = append(stagedMessages, msg)
	}

	// Use GORM bulk insert for performance
	if err := db.Get(ctx).CreateInBatches(stagedMessages, 100).Error; err != nil {
		return nil, err
	}

	return stagedMessages, nil
}

// IngestGlobalEvent saves a raw JSON bulk message payload securely and quickly
func (s *Service) IngestGlobalEvent(ctx context.Context, tenantID uuid.UUID, senderIP string, rawJSON string) (*models.GlobalIncomingEvent, error) {
	// Generate custom Event ID using Redis
	var customID string
	if s.redisClient != nil {
		idGen := redis.NewIDGenerator(s.redisClient)
		var err error
		customID, err = idGen.GenerateEventID(ctx, tenantID, senderIP)
		if err != nil {
			return nil, fmt.Errorf("failed to generate event ID: %w", err)
		}
	} else {
		// Fallback for tests or when Redis is down
		customID = "EV_FB_" + uuid.New().String()[:8]
	}
	
	if senderIP == "" {
		senderIP = "127.0.0.1"
	}

	source, _ := ctx.Value("request_source").(string)
	if source == "" {
		source = "UNKNOWN"
	}

	event := &models.GlobalIncomingEvent{
		CustomID:   customID,
		TenantID:   &tenantID,
		SenderIP:   senderIP,
		RawPayload: rawJSON,
		Status:     "RECEIVED",
		Source:     source,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := db.Get(ctx).Create(event).Error; err != nil {
		return nil, err
	}

	// 3. Trigger Phase 2 Parser Worker (GlobalEventWorker)
	if dispatcher.GlobalManager != nil {
		// Use a general client for global events
		client, err := dispatcher.GlobalManager.GetRiverClient(tenantID)
		if err == nil && client != nil {
			_, _ = client.Insert(ctx, queue.GlobalEventArgs{
				GlobalEventID: event.ID,
				TenantID:      tenantID,
			}, nil)
		}
	}
	
	return event, nil
}

// IngestIncomingMessage saves a raw JSON message from an accounting system
func (s *Service) IngestIncomingMessage(ctx context.Context, tenantIDStr string, senderName string, rawJSON string) (*models.IncomingMessage, error) {
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return nil, errors.New("invalid tenant id format")
	}

	// Extract phone_number/content from the raw payload (if present at top level)
	// so the downstream AnalysisWorker can create a valid Message row.
	phoneNumber, content := extractRawMessageFields(rawJSON)

	incoming := &models.IncomingMessage{
		ID:             uuid.New(),
		TenantID:       tenantID,
		SenderName:     senderName,
		RawPayload:     rawJSON,
		PhoneNumber:    phoneNumber,
		MessageContent: content,
		ReceivedAt:     time.Now(),
	}

	if err := db.Get(ctx).Create(incoming).Error; err != nil {
		return nil, err
	}

	// Trigger background analysis via River (Sign 2: Analysis)
	s.TriggerAnalysis(ctx, tenantID, incoming.ID)

	return incoming, nil
}

// extractRawMessageFields attempts to read top-level phone_number/content from a raw
// JSON payload. Used by the legacy single-message ingestion path so that the
// AnalysisWorker never receives empty values.
func extractRawMessageFields(rawJSON string) (phoneNumber string, content string) {
	if rawJSON == "" {
		return "", ""
	}
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(rawJSON), &raw); err != nil {
		return "", ""
	}
	if v, ok := raw["phone_number"].(string); ok {
		phoneNumber = v
	}
	if v, ok := raw["content"].(string); ok {
		content = v
	}
	return phoneNumber, content
}

// IngestBulkIncomingMessages handles bulk ingestion of messages (e.g. from Excel)
// so they follow the same movement (Received -> Analyzed -> Staged) as accounting messages.
func (s *Service) IngestBulkIncomingMessages(ctx context.Context, tenantIDStr string, senderName string, messages []BulkMessageItem) (int, error) {
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return 0, errors.New("invalid tenant id format")
	}

	var incomingMessages []models.IncomingMessage
	for _, m := range messages {
		// Serialize content to raw payload for AnalysisWorker to parse
		payload := map[string]string{
			"phone_number": m.PhoneNumber,
			"content":      m.Content,
		}
		rawJSON, _ := json.Marshal(payload)

		incomingMessages = append(incomingMessages, models.IncomingMessage{
			ID:         uuid.New(),
			TenantID:   tenantID,
			SenderName: senderName,
			RawPayload: string(rawJSON),
			ReceivedAt: time.Now(),
		})
	}

	if err := db.Get(ctx).CreateInBatches(incomingMessages, 100).Error; err != nil {
		return 0, err
	}

	// Trigger analysis for each
	for _, msg := range incomingMessages {
		s.TriggerAnalysis(ctx, tenantID, msg.ID)
	}

	return len(incomingMessages), nil
}

func (s *Service) TriggerAnalysis(ctx context.Context, tenantID uuid.UUID, incomingID uuid.UUID) {
	if dispatcher.GlobalManager != nil {
		client, err := dispatcher.GlobalManager.GetRiverClient(tenantID)
		if err == nil && client != nil {
			_, err = client.Insert(ctx, queue.AnalysisArgs{
				IncomingMessageID: incomingID,
				TenantID:          tenantID,
			}, nil)
			if err != nil {
				logger.GetLogger().Error("Failed to insert Analysis job",
					zap.String("tenant_id", tenantID.String()),
					zap.String("incoming_id", incomingID.String()),
					zap.Error(err),
				)
			}
		} else {
			logger.GetLogger().Warn("River client not available for tenant analysis",
				zap.String("tenant_id", tenantID.String()),
			)
		}
	}
}

// ListMessages retrieves messages for a tenant with comprehensive filtering and pagination
func (s *Service) ListMessages(ctx context.Context, tenantIDStr string, filters MessageFilters, page, pageSize int) (*PaginatedMessages, error) {
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return nil, errors.New("invalid tenant id format")
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}

	var messages []models.Message
	var totalCount int64

	query := db.Get(ctx).Model(&models.Message{}).Where("tenant_id = ?", tenantID)

	// Apply optional filters
	if filters.Status != "" {
		if filters.Status == "WAITING_GROUP" {
			query = query.Where("status IN ?", []string{"PENDING", "QUEUED", "WAITING"})
		} else if filters.Status == "PROCESSING_GROUP" {
			query = query.Where("status IN ?", []string{"SENT", "PROCESSING"})
		} else {
			query = query.Where("status = ?", filters.Status)
		}
	}
	if filters.PhoneNumber != "" {
		query = query.Where("phone_number LIKE ?", "%"+filters.PhoneNumber+"%")
	}
	if filters.SenderName != "" {
		query = query.Where("sender_name ILIKE ?", "%"+filters.SenderName+"%")
	}
	if filters.Source != "" {
		if filters.Source == "SYSTEM" {
			query = query.Where("source != ?", "WEB_UI")
		} else {
			query = query.Where("source = ?", filters.Source)
		}
	}
	if filters.MessageType != "" {
		query = query.Where("message_type = ?", filters.MessageType)
	}
	if filters.ChannelID != "" {
		query = query.Where("channel_id = ?", filters.ChannelID)
	}
	if filters.DateFrom != "" {
		query = query.Where("created_at >= ?", filters.DateFrom+" 00:00:00")
	}
	if filters.DateTo != "" {
		query = query.Where("created_at <= ?", filters.DateTo+" 23:59:59")
	}

	// Get total count
	if err := query.Count(&totalCount).Error; err != nil {
		return nil, err
	}

	totalPages := int(totalCount) / pageSize
	if int(totalCount)%pageSize != 0 {
		totalPages++
	}

	// Get paginated results ordered newest first
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&messages).Error; err != nil {
		return nil, err
	}

	return &PaginatedMessages{
		Messages:   messages,
		TotalCount: totalCount,
		TotalPages: totalPages,
		Page:       page,
		PageSize:   pageSize,
	}, nil
}

// ListIncomingMessages retrieves raw incoming messages for a tenant
func (s *Service) ListIncomingMessages(ctx context.Context, tenantIDStr string, limit int) ([]models.IncomingMessage, error) {
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return nil, errors.New("invalid tenant id format")
	}

	var incoming []models.IncomingMessage
	if err := db.Get(ctx).Where("tenant_id = ?", tenantID).Order("received_at DESC").Limit(limit).Find(&incoming).Error; err != nil {
		return nil, err
	}

	return incoming, nil
}

// GetBulkEvent retrieves the progress of a global ingestion event by custom ID
func (s *Service) GetBulkEvent(ctx context.Context, tenantID uuid.UUID, eventCustomID string) (*models.GlobalIncomingEvent, error) {
	var event models.GlobalIncomingEvent
	if err := db.Get(ctx).Where("custom_id = ? AND tenant_id = ?", eventCustomID, tenantID).First(&event).Error; err != nil {
		return nil, err
	}
	return &event, nil
}

// calculatePricing calculates character count, parts count, encoding, and cost for a message
func (s *Service) calculatePricing(ctx context.Context, content string) (int, int, string, float64) {
	var charCount, partsCount int
	var encoding string
	var cost float64
	
	if s.pricingCache != nil {
		pricingInfo, err := s.pricingCache.GetOrCalculate(ctx, content, 0.05)
		if err == nil {
			return pricingInfo.CharCount, pricingInfo.PartsCount, pricingInfo.Encoding, pricingInfo.Cost
		}
	}
	
	// Fallback to direct calculation
	charCount = len([]rune(content))
	isUnicode := false
	for _, r := range content {
		if r > 127 {
			isUnicode = true
			break
		}
	}
	
	if isUnicode {
		encoding = "UNICODE"
		if charCount <= 70 {
			partsCount = 1
		} else {
			partsCount = (charCount + 66) / 67
		}
	} else {
		encoding = "GSM7"
		if charCount <= 160 {
			partsCount = 1
		} else {
			partsCount = (charCount + 152) / 153
		}
	}
	cost = float64(partsCount) * 0.05
	
	return charCount, partsCount, encoding, cost
}

// BulkDeleteMessages deletes multiple messages for a tenant
func (s *Service) BulkDeleteMessages(ctx context.Context, tenantIDStr string, ids []uuid.UUID) error {
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return errors.New("invalid tenant id format")
	}

	return db.Get(ctx).Where("tenant_id = ? AND id IN ?", tenantID, ids).Delete(&models.Message{}).Error
}

// BulkResendMessages attempts to resend multiple failed messages
func (s *Service) BulkResendMessages(ctx context.Context, tenantIDStr string, ids []uuid.UUID) error {
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return errors.New("invalid tenant id format")
	}

	// 1. Fetch messages that are in FAILED status
	var messages []models.Message
	if err := db.Get(ctx).Where("tenant_id = ? AND id IN ? AND status = ?", tenantID, ids, models.MessageFailed).Find(&messages).Error; err != nil {
		return err
	}

	if len(messages) == 0 {
		return nil
	}

	// 2. Reset status to PENDING and enqueue
	for _, msg := range messages {
		msg.Status = models.MessagePending
		msg.RetryCount++
		db.Get(ctx).Save(&msg)

		if s.messageQueue != nil {
			queueItem := &redis.MessageQueueItem{
				MessageID:   msg.ID,
				TenantID:    tenantID,
				PhoneNumber: msg.PhoneNumber,
				Content:     msg.Content,
				PartsCount:  msg.PartsCount,
				Cost:        msg.Cost,
				Priority:    3, // Higher priority for resends
				EnqueuedAt:  time.Now(),
			}
			s.messageQueue.Enqueue(ctx, queueItem)
		}
	}

	return nil
}
