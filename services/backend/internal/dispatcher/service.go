package dispatcher

import (
	"time"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/limiter"
	"github.com/al-hudhud/backend/internal/metrics"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/al-hudhud/backend/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Dispatcher struct {
	stopChan    chan struct{}
	limiter     *limiter.TenantLimiter
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		stopChan:    make(chan struct{}),
		limiter:     limiter.NewTenantLimiter(500), // Global limit 500 msgs/min
	}
}

// Start begins the heart of the messaging system: background scanning and distribution
func (d *Dispatcher) Start() {
	log := logger.GetLogger()
	log.Info("Message Dispatcher service started - Global Standard Edition")

	ticker := time.NewTicker(2 * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				d.dispatchNextBatch()
			case <-d.stopChan:
				ticker.Stop()
				return
			}
		}
	}()
}

func (d *Dispatcher) Stop() {
	close(d.stopChan)
}

func (d *Dispatcher) dispatchNextBatch() {
	log := logger.GetLogger()

	// 1. Use a standard/constant batch size
	batchSize := 50
	
	// Phase 3.3: Export to Prometheus
	metrics.SetAdaptiveBatchSize(float64(batchSize))
	
	log.Debug("Standard batch size configured", 
		zap.Int("batch_size", batchSize))

	// 2. Get all active tenants with pending messages
	var tenantIDs []string
	// Optimized query: GROUP BY is better than DISTINCT for large tables
	result := db.GormDB.Raw(`
		SELECT tenant_id 
		FROM messages 
		WHERE status = ? AND scheduled_at <= ?
		GROUP BY tenant_id
		LIMIT ?
	`, models.MessagePending, time.Now(), batchSize).Scan(&tenantIDs)

	if result.Error != nil {
		log.Error("Dispatcher database lookup error", zap.Error(result.Error))
		return
	}

	if len(tenantIDs) == 0 {
		return
	}

	// 2. Process each tenant separately to ensure isolation
	for _, tenantIDStr := range tenantIDs {
		d.processTenantMessages(tenantIDStr)
	}
}

// processTenantMessages processes pending messages for a specific tenant
func (d *Dispatcher) processTenantMessages(tenantIDStr string) {
	log := logger.GetLogger()

	// 1. Fetch PENDING messages for this tenant only
	var messages []models.Message
	result := db.GormDB.Where("tenant_id = ? AND status = ? AND scheduled_at <= ?",
		tenantIDStr, models.MessagePending, time.Now()).
		Limit(20). // Limit per tenant to ensure fairness
		Find(&messages)

	if result.Error != nil {
		log.Error("Error fetching tenant messages",
			zap.String("tenant_id", tenantIDStr),
			zap.Error(result.Error))
		return
	}

	if len(messages) == 0 {
		return
	}

	// 2. Process each message
	for _, msg := range messages {
		// Rate limiting check per tenant
		if !d.limiter.Allow(msg.TenantID, 60) {
			continue // Skip this message for now, will retry next cycle
		}

		// 3. Find an ONLINE device for this tenant
		var device models.PairedDevice
		err := db.GormDB.Where("tenant_id = ? AND status = ?", msg.TenantID, "ONLINE").
			Order("last_seen DESC").
			First(&device).Error

		if err != nil {
			if !errorsIs(err, gorm.ErrRecordNotFound) {
				log.Error("Error finding device for tenant",
					zap.String("tenant_id", msg.TenantID.String()),
					zap.Error(err))
			}
			continue
		}

		// 4. Mark as PROCESSING and assign device
		db.GormDB.Model(&msg).Updates(map[string]interface{}{
			"device_id": device.ID,
			"status":    models.MessageProcessing,
		})

		log.Debug("Message dispatched for processing",
			zap.String("msg_id", msg.ID.String()),
			zap.String("tenant_id", msg.TenantID.String()),
			zap.String("device_id", device.ID.String()))
	}
}

// Helper to avoid circular imports or direct errors
func errorsIs(err, target error) bool {
	return err == target
}
