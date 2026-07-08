package dispatcher

import (
	"context"
	"sync"
	"time"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/limiter"
	"github.com/al-hudhud/backend/internal/metrics"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/al-hudhud/backend/internal/queue"
	"github.com/al-hudhud/backend/internal/redis"
	"github.com/al-hudhud/backend/internal/ws"
	"github.com/al-hudhud/backend/pkg/logger"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"go.uber.org/zap"
)

// TenantDispatcher handles message dispatching for a single tenant
type TenantDispatcher struct {
	tenantID    uuid.UUID
	tenant      *models.Tenant
	riverClient *river.Client[pgx.Tx]
	riverCtx    context.Context
	riverCancel context.CancelFunc
	stopChan    chan struct{}
	rateLimiter *limiter.TenantLimiter
	redisClient *redis.Client
	mu          sync.RWMutex
	isRunning   bool
}

// NewTenantDispatcher creates a new dispatcher for a specific tenant
func NewTenantDispatcher(tenantID uuid.UUID, redisClient *redis.Client) *TenantDispatcher {
	return &TenantDispatcher{
		tenantID:    tenantID,
		redisClient: redisClient,
		stopChan:    make(chan struct{}),
		rateLimiter: limiter.NewTenantLimiter(500),
		isRunning:   false,
	}
}

// Start begins the dispatcher for this tenant
func (td *TenantDispatcher) Start() error {
	td.mu.Lock()
	defer td.mu.Unlock()

	if td.isRunning {
		return nil // Already running
	}

	log := logger.GetLogger()

	// Load tenant information
	if err := db.GormDB.First(&td.tenant, td.tenantID).Error; err != nil {
		log.Error("Failed to load tenant",
			zap.String("tenant_id", td.tenantID.String()),
			zap.Error(err))
		return err
	}

	// Ensure tenant schema and migrations (System and River) are up to date
	if err := db.CreateTenantSchema(td.tenantID); err != nil {
		log.Error("Failed to ensure tenant schema",
			zap.String("tenant_id", td.tenantID.String()),
			zap.Error(err))
		return err
	}
	if err := db.MigrateTenantSchema(td.tenantID); err != nil {
		log.Warn("Tenant schema migration encountered errors (continuing)",
			zap.String("tenant_id", td.tenantID.String()),
			zap.Error(err))
	}

	// Initialize River client for this tenant (Isolated Queue)
	td.riverCtx, td.riverCancel = context.WithCancel(context.Background())
	client, err := queue.InitTenantClient(td.riverCtx, td.tenantID, td.redisClient)
	if err != nil {
		log.Error("Failed to initialize River client for tenant",
			zap.String("tenant_id", td.tenantID.String()),
			zap.Error(err))
		return err
	}
	td.riverClient = client

	// Start River client
	if err := td.riverClient.Start(td.riverCtx); err != nil {
		log.Error("Failed to start River client for tenant",
			zap.String("tenant_id", td.tenantID.String()),
			zap.Error(err))
		return err
	}

	// Start message fetcher
	go td.fetchMessages()

	td.isRunning = true

	log.Info("TenantDispatcher started (Isolated River Queue)",
		zap.String("tenant_id", td.tenantID.String()),
		zap.String("business_name", td.tenant.BusinessName))

	return nil
}

// Stop stops the dispatcher and the River client
func (td *TenantDispatcher) Stop() {
	td.mu.Lock()
	defer td.mu.Unlock()

	if !td.isRunning {
		return
	}

	log := logger.GetLogger()

	// Graceful stop for River client
	if td.riverClient != nil {
		if err := td.riverClient.Stop(context.Background()); err != nil {
			log.Error("Error stopping River client", zap.Error(err))
		}
		td.riverCancel()
	}

	// Close stop channel for the fetcher loop
	close(td.stopChan)

	td.isRunning = false

	log.Info("TenantDispatcher stopped",
		zap.String("tenant_id", td.tenantID.String()))
}

// fetchMessages continuously fetches pending messages for this tenant
func (td *TenantDispatcher) fetchMessages() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Adaptive interval
			cpu, mem := metrics.GetServerLoad()
			if cpu > 90 || mem > 90 {
				ticker.Reset(10 * time.Second)
			} else if cpu > 70 || mem > 70 {
				ticker.Reset(5 * time.Second)
			} else if cpu < 30 && mem < 30 {
				ticker.Reset(1 * time.Second)
			} else {
				ticker.Reset(2 * time.Second)
			}
			td.fetchAndQueueMessages()
		case <-td.stopChan:
			return
		}
	}
}

// fetchAndQueueMessages fetches pending messages and enqueues them into River
func (td *TenantDispatcher) fetchAndQueueMessages() {
	log := logger.GetLogger()

	err := db.RunInTenantPgxTx(context.Background(), td.tenantID, false, db.PgxPool, func(tx pgx.Tx) error {
		// Adaptive limit and rate
		cpu, mem := metrics.GetServerLoad()
		limit := 100
		rate := 60.0

		if cpu > 90 || mem > 90 {
			limit = 10
			rate = 10.0
		} else if cpu > 70 || mem > 70 {
			limit = 50
			rate = 30.0
		} else if cpu < 30 && mem < 30 {
			limit = 200
			rate = 120.0
		}

		metrics.SetAdaptiveBatchSize(float64(limit))

		// 1. Fetch pending messages with FOR UPDATE SKIP LOCKED (Requirement 12.879)
		// This prevents multiple dispatchers from fetching the same messages.
		rows, err := tx.Query(context.Background(), `
			SELECT id, channel_id FROM messages 
		WHERE status IN ('PENDING','WAITING') AND scheduled_at <= $1
		ORDER BY scheduled_at ASC
		LIMIT $2
		FOR UPDATE SKIP LOCKED`, time.Now(), limit)
		if err != nil {
			return err
		}
		defer rows.Close()

		var messageIDs []uuid.UUID
		var channelIDs []uuid.UUID
		for rows.Next() {
			var msgID uuid.UUID
			var chID uuid.UUID
			if err := rows.Scan(&msgID, &chID); err != nil {
				return err
			}
			messageIDs = append(messageIDs, msgID)
			channelIDs = append(channelIDs, chID)
		}

		if len(messageIDs) == 0 {
			return nil
		}

		// 2. Process each message respecting channel status
		for i, msgID := range messageIDs {
			chID := channelIDs[i]

			// Rate limiting check
			if !td.rateLimiter.Allow(td.tenantID, int(rate)) {
				log.Debug("Rate limit reached for tenant",
					zap.String("tenant_id", td.tenantID.String()))
				break
			}

			// Check connection status via GlobalHub
			status, err := ws.GlobalHub.GetConnectionStatus(context.Background(), td.tenantID, chID)
			if err != nil {
				log.Error("Failed to get connection status", zap.Error(err))
				continue
			}

			if status != string(ws.StatusConnected) {
				// Mark message as WAITING if channel offline
				if _, err := tx.Exec(context.Background(), `UPDATE messages SET status = 'WAITING', updated_at = NOW() WHERE id = $1`, msgID); err != nil {
					log.Error("Failed to update message status to WAITING", zap.Error(err))
				}
				continue
			}

			// Update status FIRST to QUEUED (Atomic within tx)
			commandTag, err := tx.Exec(context.Background(), `
				UPDATE messages SET status = 'QUEUED', updated_at = NOW() 
				WHERE id = $1 AND status IN ('PENDING','WAITING')`, msgID)

			if err != nil {
				return err
			}

			if commandTag.RowsAffected() == 0 {
				// Already claimed by another worker/transaction
				continue
			}

			// Then insert into the target queue (River) within the SAME transaction
			_, err = td.riverClient.InsertTx(td.riverCtx, tx, queue.SendSMSArgs{
				MessageID: msgID,
				TenantID:  td.tenantID,
			}, nil)

			if err != nil {
				log.Error("Failed to insert job into River",
					zap.String("tenant_id", td.tenantID.String()),
					zap.String("msg_id", msgID.String()),
					zap.Error(err))
				return err // Rollback the whole batch if queueing fails to ensure consistency
			}
		}

		return nil
	})

	if err != nil {
		log.Error("Error processing messages for tenant",
			zap.String("tenant_id", td.tenantID.String()),
			zap.Error(err))
	}
}

// ProcessWaitingMessages processes messages in WAITING state for a specific channel when it becomes ONLINE.
func (td *TenantDispatcher) ProcessWaitingMessages(channelID uuid.UUID) error {
	log := logger.GetLogger()

	err := db.RunInTenantPgxTx(context.Background(), td.tenantID, false, db.PgxPool, func(tx pgx.Tx) error {
		// Fetch waiting messages for the channel with row lock
		rows, err := tx.Query(context.Background(), `
			SELECT id FROM messages 
			WHERE status = 'WAITING' AND channel_id = $1 
			ORDER BY scheduled_at ASC 
			FOR UPDATE SKIP LOCKED`, channelID)
		if err != nil {
			return err
		}
		defer rows.Close()

		var msgIDs []uuid.UUID
		for rows.Next() {
			var msgID uuid.UUID
			if err := rows.Scan(&msgID); err != nil {
				return err
			}
			msgIDs = append(msgIDs, msgID)
		}

		if len(msgIDs) == 0 {
			return nil
		}

		for _, msgID := range msgIDs {
			// Update status to QUEUED
			commandTag, err := tx.Exec(context.Background(), `
				UPDATE messages SET status = 'QUEUED', updated_at = NOW() 
				WHERE id = $1 AND status = 'WAITING'`, msgID)
			if err != nil {
				return err
			}
			if commandTag.RowsAffected() == 0 {
				continue
			}

			// Insert into River queue
			_, err = td.riverClient.InsertTx(td.riverCtx, tx, queue.SendSMSArgs{MessageID: msgID, TenantID: td.tenantID}, nil)
			if err != nil {
				log.Error("Failed to insert waiting job into River", zap.String("tenant_id", td.tenantID.String()), zap.String("msg_id", msgID.String()), zap.Error(err))
				return err
			}
		}

		return nil
	})

	if err != nil {
		log.Error("Error processing waiting messages for channel", zap.String("tenant_id", td.tenantID.String()), zap.String("channel_id", channelID.String()), zap.Error(err))
	}
	return err
}

func (td *TenantDispatcher) GetStats() map[string]interface{} {
	td.mu.RLock()
	defer td.mu.RUnlock()

	return map[string]interface{}{
		"tenant_id":  td.tenantID.String(),
		"is_running": td.isRunning,
		// Stats for River could be added here
	}
}
