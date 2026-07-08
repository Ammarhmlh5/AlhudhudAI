package orchestrator

import (
	"time"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/al-hudhud/backend/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Orchestrator struct {
	stopChan chan struct{}
}

func NewOrchestrator() *Orchestrator {
	return &Orchestrator{
		stopChan: make(chan struct{}),
	}
}

func (o *Orchestrator) Start() {
	log := logger.GetLogger()
	log.Info("Message Orchestrator service started - Ensuring Reliability")

	ticker := time.NewTicker(30 * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				o.reconcileMessages()
			case <-o.stopChan:
				ticker.Stop()
				return
			}
		}
	}()
}

func (o *Orchestrator) Stop() {
	close(o.stopChan)
}

func (o *Orchestrator) reconcileMessages() {
	log := logger.GetLogger()

	// 1. Get all active tenants
	var tenants []models.Tenant
	if err := db.GormDB.Where("is_active = ?", true).Find(&tenants).Error; err != nil {
		log.Error("Orchestrator: Failed to fetch active tenants", zap.Error(err))
		return
	}

	timeout := time.Now().Add(-2 * time.Minute)

	// 2. Process each tenant separately in its own schema scope
	for _, t := range tenants {
		err := db.RunInTenantScope(t.ID, false, func(tx *gorm.DB) error {
			var stuckMessages []models.Message
			// Find messages stuck in PROCESSING for too long
			// Use explicit Select to avoid "cached plan must not change result type" errors 
			// when switching between tenant schemas with slightly different column orders.
			result := tx.Select("id", "status", "retry_count", "updated_at").
				Where("status = ? AND updated_at < ?",
					models.MessageProcessing, timeout).Find(&stuckMessages)

			if result.Error != nil {
				return result.Error
			}

			for _, msg := range stuckMessages {
				if msg.RetryCount >= 5 {
					// Mark as FAILED if retry limit reached
					tx.Model(&msg).
						Where("id = ? AND status = ?", msg.ID, models.MessageProcessing).
						Update("status", models.MessageFailed)
					log.Warn("Message failed after maximum retries",
						zap.String("msg_id", msg.ID.String()),
						zap.String("tenant_id", t.ID.String()))
				} else {
					// Increment retry count and put back to PENDING for dispatcher to pick up
					tx.Model(&msg).
						Where("id = ? AND status = ?", msg.ID, models.MessageProcessing).
						Updates(map[string]interface{}{
							"status":      models.MessagePending,
							"retry_count": msg.RetryCount + 1,
						})
					log.Info("Message reconciled and put back to PENDING",
						zap.String("msg_id", msg.ID.String()),
						zap.String("tenant_id", t.ID.String()))
				}
			}
			return nil
		})

		if err != nil {
			log.Error("Orchestrator reconciliation error for tenant",
				zap.String("tenant_id", t.ID.String()),
				zap.Error(err))
		}
	}
}
