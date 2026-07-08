package batch

import (
	"context"
	"fmt"
	"time"

	"github.com/al-hudhud/backend/internal/database"
	"github.com/al-hudhud/backend/internal/gateway"
	"github.com/al-hudhud/backend/internal/metrics"
	"github.com/al-hudhud/backend/internal/queue"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// MessageBuffer interface for dependency injection
type MessageBuffer interface {
	Push(ctx context.Context, tenantID uuid.UUID, msg gateway.BufferedMessage) error
	PopBatch(ctx context.Context, tenantID uuid.UUID, batchSize int) ([]gateway.BufferedMessage, error)
	GetSize(ctx context.Context, tenantID uuid.UUID) (int64, error)
	GetAllActiveTenants(ctx context.Context) ([]uuid.UUID, error)
	RemoveActiveTenant(ctx context.Context, tenantID uuid.UUID) error
	Clear(ctx context.Context, tenantID uuid.UUID) error
	SetTTL(ctx context.Context, tenantID uuid.UUID, ttl time.Duration) error
}

// BatchProcessor handles batch processing of messages from Redis to PostgreSQL
type BatchProcessor struct {
	buffer      MessageBuffer
	db          *database.DB
	config      *BatchConfig
	logger      *zap.Logger
	stopCh      chan struct{}
	riverClient *river.Client[pgx.Tx]
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(
	buffer MessageBuffer,
	db *database.DB,
	config *BatchConfig,
	logger *zap.Logger,
	riverClient *river.Client[pgx.Tx],
) *BatchProcessor {
	if config == nil {
		config = DefaultBatchConfig()
	}

	return &BatchProcessor{
		buffer:      buffer,
		db:          db,
		config:      config,
		logger:      logger,
		stopCh:      make(chan struct{}),
		riverClient: riverClient,
	}
}

// ProcessBatch processes a batch of messages for a specific tenant
// Implements Requirements 4.1, 4.2
func (bp *BatchProcessor) ProcessBatch(ctx context.Context, tenantID uuid.UUID) (*BatchResult, error) {
	startTime := time.Now()

	result := &BatchResult{
		TenantID: tenantID,
	}

	// 1. Determine batch size based on buffer size (adaptive mode)
	batchSize := bp.AdaptBatchSize(ctx, tenantID)
	if batchSize == 0 {
		// No messages to process
		return result, nil
	}

	// 2. Extract messages from Redis buffer
	bufferedMessages, err := bp.buffer.PopBatch(ctx, tenantID, batchSize)
	if err != nil {
		bp.logger.Error("Failed to pop batch from buffer",
			zap.Error(err),
			zap.String("tenant_id", tenantID.String()),
			zap.Int("batch_size", batchSize),
		)
		result.Error = fmt.Errorf("pop batch: %w", err)
		
		// Record failed batch
		tenantIDStr := tenantID.String()
		metrics.RecordBatchProcessed(tenantIDStr, "failure")
		
		return result, err
	}

	if len(bufferedMessages) == 0 {
		// No messages retrieved
		return result, nil
	}

	result.ProcessedCount = len(bufferedMessages)

	// Record batch size metric
	metrics.ObserveBatchSize(float64(len(bufferedMessages)))

	// 3. Convert to database format
	messages := bp.convertToDBFormat(tenantID, bufferedMessages)

	// 4. Insert batch into PostgreSQL within a single transaction
	err = bp.insertBatch(ctx, tenantID, messages)
	if err != nil {
		bp.logger.Warn("Batch insert failed, attempting individual retry",
			zap.Error(err),
			zap.String("tenant_id", tenantID.String()),
			zap.Int("count", len(messages)),
		)
		
		// Implements Requirement 4.7: retry messages individually on batch failure
		successCount, failedCount := bp.retryIndividually(ctx, tenantID, messages)
		result.SuccessCount = successCount
		result.FailedCount = failedCount
		
		if failedCount > 0 {
			result.Error = fmt.Errorf("batch insert failed, individual retry: %d succeeded, %d failed", successCount, failedCount)
		}
		
		// Return success if at least some messages were processed
		if successCount > 0 {
			bp.logger.Info("Partial batch recovery successful",
				zap.String("tenant_id", tenantID.String()),
				zap.Int("success_count", successCount),
				zap.Int("failed_count", failedCount),
			)
		}
		
		return result, nil
	}

	result.SuccessCount = len(messages)
	result.ProcessingTimeMs = time.Since(startTime).Milliseconds()

	// Record successful batch processing
	tenantIDStr := tenantID.String()
	metrics.RecordBatchProcessed(tenantIDStr, "success")
	metrics.ObserveMessageLatency("batch_processor", time.Since(startTime).Seconds())

	bp.logger.Info("Batch processed successfully",
		zap.String("tenant_id", tenantID.String()),
		zap.Int("count", result.SuccessCount),
		zap.Int64("processing_time_ms", result.ProcessingTimeMs),
	)

	// 5. Remove tenant from active set if no more messages
	if err := bp.buffer.RemoveActiveTenant(ctx, tenantID); err != nil {
		bp.logger.Warn("Failed to remove tenant from active set",
			zap.Error(err),
			zap.String("tenant_id", tenantID.String()),
		)
	}

	return result, nil
}

// ProcessAllTenants processes batches for all active tenants
func (bp *BatchProcessor) ProcessAllTenants(ctx context.Context) error {
	// Get all tenants with pending messages
	tenantIDs, err := bp.buffer.GetAllActiveTenants(ctx)
	if err != nil {
		return fmt.Errorf("get active tenants: %w", err)
	}

	if len(tenantIDs) == 0 {
		return nil
	}

	bp.logger.Debug("Processing batches for active tenants",
		zap.Int("tenant_count", len(tenantIDs)),
	)

	// Process each tenant separately for isolation
	for _, tenantID := range tenantIDs {
		_, err := bp.ProcessBatch(ctx, tenantID)
		if err != nil {
			bp.logger.Error("Failed to process batch for tenant",
				zap.Error(err),
				zap.String("tenant_id", tenantID.String()),
			)
			// Continue processing other tenants
			continue
		}
	}

	return nil
}

// AdaptBatchSize adjusts batch size based on buffer pressure
// Implements Requirement 9 (Smart Adaptation)
func (bp *BatchProcessor) AdaptBatchSize(ctx context.Context, tenantID uuid.UUID) int {
	if !bp.config.AdaptiveMode {
		return bp.config.DefaultBatchSize
	}

	bufferSize, err := bp.buffer.GetSize(ctx, tenantID)
	if err != nil {
		bp.logger.Warn("Failed to get buffer size, using default",
			zap.Error(err),
			zap.String("tenant_id", tenantID.String()),
		)
		return bp.config.DefaultBatchSize
	}

	// Adaptive sizing based on pressure
	switch {
	case bufferSize == 0:
		return 0
	case bufferSize < int64(bp.config.MinBatchSize):
		// Low pressure: process immediately with small batch
		return int(bufferSize)
	case bufferSize >= int64(bp.config.MinBatchSize) && bufferSize < int64(bp.config.DefaultBatchSize):
		// Medium pressure: process with medium batch
		return min(int(bufferSize), bp.config.DefaultBatchSize/2)
	case bufferSize >= int64(bp.config.DefaultBatchSize):
		// High pressure: process with large batch
		return min(int(bufferSize), bp.config.DefaultBatchSize)
	default:
		return bp.config.DefaultBatchSize
	}
}

// convertToDBFormat converts buffered messages to database format
func (bp *BatchProcessor) convertToDBFormat(tenantID uuid.UUID, bufferedMessages []gateway.BufferedMessage) []MessageForInsert {
	messages := make([]MessageForInsert, 0, len(bufferedMessages))
	now := time.Now()

	for _, msg := range bufferedMessages {
		// Calculate cost (simplified - should use pricing engine)
		cost := bp.calculateCost(msg.Content)

		// Calculate parts count for long messages
		partsCount := bp.calculatePartsCount(msg.Content)

		messages = append(messages, MessageForInsert{
			TenantID:    tenantID.String(),
			MessageID:   msg.MessageID,
			PhoneNumber: msg.PhoneNumber,
			Content:     msg.Content,
			Status:      "PENDING",
			Priority:    msg.Priority,
			Cost:        cost,
			PartsCount:  partsCount,
			QueuedAt:    msg.QueuedAt,
			CreatedAt:   now,
			UpdatedAt:   now,
		})
	}

	return messages
}

// insertBatch inserts a batch of messages into PostgreSQL within a single transaction
// and creates River Queue jobs for each message
// Implements Requirement 4.2 (Batch Insert in Single Transaction)
// Implements Requirement 4.4 (River Queue Jobs Creation)
func (bp *BatchProcessor) insertBatch(ctx context.Context, tenantID uuid.UUID, messages []MessageForInsert) error {
	// Begin transaction
	tx, err := bp.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Prepare batch insert using COPY for maximum performance
	// This is much faster than individual INSERTs
	_, err = tx.CopyFrom(
		ctx,
		pgx.Identifier{"messages"},
		[]string{
			"tenant_id",
			"message_id",
			"phone_number",
			"message_content",
			"status",
			"priority",
			"cost",
			"parts_count",
			"queued_at",
			"created_at",
			"updated_at",
		},
		pgx.CopyFromSlice(len(messages), func(i int) ([]interface{}, error) {
			msg := messages[i]
			return []interface{}{
				msg.TenantID,
				msg.MessageID,
				msg.PhoneNumber,
				msg.Content,
				msg.Status,
				msg.Priority,
				msg.Cost,
				msg.PartsCount,
				msg.QueuedAt,
				msg.CreatedAt,
				msg.UpdatedAt,
			}, nil
		}),
	)

	if err != nil {
		return fmt.Errorf("copy from: %w", err)
	}

	// Create River Queue jobs for each message
	// Implements Requirement 4.4
	if bp.riverClient != nil {
		for _, msg := range messages {
			// Parse tenant_id and message_id to UUID
			tenantUUID, err := uuid.Parse(msg.TenantID)
			if err != nil {
				bp.logger.Warn("Failed to parse tenant_id as UUID, skipping River job",
					zap.String("tenant_id", msg.TenantID),
					zap.Error(err),
				)
				continue
			}

			messageUUID, err := uuid.Parse(msg.MessageID)
			if err != nil {
				bp.logger.Warn("Failed to parse message_id as UUID, skipping River job",
					zap.String("message_id", msg.MessageID),
					zap.Error(err),
				)
				continue
			}

			// Create River job with tenant_id and message_id
			// max_attempts = 3 as per requirement
			_, err = bp.riverClient.InsertTx(ctx, tx, queue.SendSMSArgs{
				TenantID:  tenantUUID,
				MessageID: messageUUID,
			}, &river.InsertOpts{
				MaxAttempts: 3,
				Queue:       river.QueueDefault,
				Priority:    bp.getPriority(msg.Priority),
			})

			if err != nil {
				bp.logger.Error("Failed to create River job",
					zap.Error(err),
					zap.String("tenant_id", msg.TenantID),
					zap.String("message_id", msg.MessageID),
				)
				return fmt.Errorf("create river job: %w", err)
			}
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
// retryIndividually attempts to insert messages individually when batch insert failure occurs
// Implements Requirement 4.7: WHEN يفشل إدخال دفعة، THE Batch_Processor SHALL إعادة محاولة الرسائل الفردية
func (bp *BatchProcessor) retryIndividually(ctx context.Context, tenantID uuid.UUID, messages []MessageForInsert) (int, int) {
	successCount := 0
	failedCount := 0

	bp.logger.Info("Starting individual retry for failed batch",
		zap.String("tenant_id", tenantID.String()),
		zap.Int("total_messages", len(messages)),
	)

	for _, msg := range messages {
		// Begin individual transaction
		tx, err := bp.db.Pool.Begin(ctx)
		if err != nil {
			bp.logger.Error("Failed to begin transaction for individual retry",
				zap.Error(err),
				zap.String("tenant_id", tenantID.String()),
				zap.String("message_id", msg.MessageID),
			)
			failedCount++
			continue
		}

		// Insert single message
		_, err = tx.Exec(ctx, `
			INSERT INTO messages (
				tenant_id, message_id, phone_number, message_content,
				status, priority, cost, parts_count,
				queued_at, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		`,
			msg.TenantID,
			msg.MessageID,
			msg.PhoneNumber,
			msg.Content,
			msg.Status,
			msg.Priority,
			msg.Cost,
			msg.PartsCount,
			msg.QueuedAt,
			msg.CreatedAt,
			msg.UpdatedAt,
		)

		if err != nil {
			tx.Rollback(ctx)
			bp.logger.Error("Failed to insert individual message",
				zap.Error(err),
				zap.String("tenant_id", tenantID.String()),
				zap.String("message_id", msg.MessageID),
				zap.String("phone_number", msg.PhoneNumber),
			)
			failedCount++
			continue
		}

		// Create River Queue job for this message
		if bp.riverClient != nil {
			tenantUUID, err := uuid.Parse(msg.TenantID)
			if err != nil {
				tx.Rollback(ctx)
				bp.logger.Error("Failed to parse tenant_id as UUID",
					zap.Error(err),
					zap.String("tenant_id", msg.TenantID),
					zap.String("message_id", msg.MessageID),
				)
				failedCount++
				continue
			}

			messageUUID, err := uuid.Parse(msg.MessageID)
			if err != nil {
				tx.Rollback(ctx)
				bp.logger.Error("Failed to parse message_id as UUID",
					zap.Error(err),
					zap.String("message_id", msg.MessageID),
				)
				failedCount++
				continue
			}

			_, err = bp.riverClient.InsertTx(ctx, tx, queue.SendSMSArgs{
				TenantID:  tenantUUID,
				MessageID: messageUUID,
			}, &river.InsertOpts{
				MaxAttempts: 3,
				Queue:       river.QueueDefault,
				Priority:    bp.getPriority(msg.Priority),
			})

			if err != nil {
				tx.Rollback(ctx)
				bp.logger.Error("Failed to create River job for individual message",
					zap.Error(err),
					zap.String("tenant_id", msg.TenantID),
					zap.String("message_id", msg.MessageID),
				)
				failedCount++
				continue
			}
		}

		// Commit individual transaction
		if err := tx.Commit(ctx); err != nil {
			bp.logger.Error("Failed to commit individual transaction",
				zap.Error(err),
				zap.String("tenant_id", tenantID.String()),
				zap.String("message_id", msg.MessageID),
			)
			failedCount++
			continue
		}

		successCount++
		bp.logger.Debug("Successfully inserted individual message",
			zap.String("tenant_id", tenantID.String()),
			zap.String("message_id", msg.MessageID),
		)
	}

	bp.logger.Info("Individual retry completed",
		zap.String("tenant_id", tenantID.String()),
		zap.Int("success_count", successCount),
		zap.Int("failed_count", failedCount),
	)

	return successCount, failedCount
}


// getPriority converts string priority to River priority integer
func (bp *BatchProcessor) getPriority(priority string) int {
	switch priority {
	case "HIGH":
		return 1
	case "NORMAL":
		return 2
	case "LOW":
		return 3
	default:
		return 2
	}
}

// calculateCost calculates the cost of sending a message
// Simplified version - should be replaced with proper pricing engine
func (bp *BatchProcessor) calculateCost(content string) decimal.Decimal {
	// Base cost per message part
	baseCost := decimal.NewFromFloat(0.05)

	// Calculate parts
	parts := bp.calculatePartsCount(content)

	// Total cost = base cost * parts
	return baseCost.Mul(decimal.NewFromInt(int64(parts)))
}

// calculatePartsCount calculates how many SMS parts are needed
func (bp *BatchProcessor) calculatePartsCount(content string) int {
	// Standard SMS: 160 characters per part
	// Multipart SMS: 153 characters per part (7 chars for header)
	length := len([]rune(content))

	if length <= 160 {
		return 1
	}

	// Multipart message
	parts := (length + 152) / 153
	return parts
}

// Start starts the batch processor
func (bp *BatchProcessor) Start(ctx context.Context) error {
	bp.logger.Info("Starting batch processor",
		zap.Int("default_batch_size", bp.config.DefaultBatchSize),
		zap.Duration("process_interval", bp.config.ProcessInterval),
		zap.Bool("adaptive_mode", bp.config.AdaptiveMode),
	)

	ticker := time.NewTicker(bp.config.ProcessInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			bp.logger.Info("Batch processor stopped by context")
			return ctx.Err()
		case <-bp.stopCh:
			bp.logger.Info("Batch processor stopped")
			return nil
		case <-ticker.C:
			// Process all active tenants
			if err := bp.ProcessAllTenants(ctx); err != nil {
				bp.logger.Error("Failed to process all tenants",
					zap.Error(err),
				)
			}
		}
	}
}

// Stop stops the batch processor gracefully
func (bp *BatchProcessor) Stop(ctx context.Context) error {
	bp.logger.Info("Stopping batch processor")
	close(bp.stopCh)
	return nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
