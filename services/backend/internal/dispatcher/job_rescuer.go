package dispatcher

import (
	"context"
	"fmt"
	"time"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/al-hudhud/backend/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	// rescueInterval is how often stuck "running" jobs are scanned and rescued.
	rescueInterval = 30 * time.Second

	// staleRunningMinutes is how long a job may stay in the "running" state
	// before it is considered orphaned (its owning worker process died) and is
	// reset to "available" for retry. Message activation jobs normally finish
	// in well under a minute, so a 10-minute threshold is safe against
	// double-execution of jobs that a live worker is genuinely executing.
	staleRunningMinutes = 10
)

// startStuckJobRescuer runs a background loop that rescues River jobs left in
// the "running" state by workers that crashed. River's built-in JobRescuer
// repeatedly fails in this multi-tenant setup (context deadline exceeded), so
// we implement our own rescue to guarantee jobs are never stranded forever.
func (m *Manager) startStuckJobRescuer() {
	go func() {
		ticker := time.NewTicker(rescueInterval)
		defer ticker.Stop()

		// Initial pass right away so jobs stranded by a recent crash are
		// rescued as soon as they exceed the staleness threshold.
		m.rescueStuckJobs()

		for {
			select {
			case <-ticker.C:
				m.rescueStuckJobs()
			case <-m.stopChan:
				return
			}
		}
	}()
}

// rescueStuckJobs resets orphaned "running" jobs back to "available" across
// all active tenant schemas so their consumers pick them up again.
func (m *Manager) rescueStuckJobs() {
	log := logger.GetLogger()

	var tenants []models.Tenant
	if err := db.GormDB.Select("id", "is_active").Where("is_active = ?", true).Find(&tenants).Error; err != nil {
		log.Error("StuckJobRescuer: failed to load tenants", zap.Error(err))
		return
	}

	for _, tenant := range tenants {
		if err := m.rescueStuckJobsForSchema(tenant.ID); err != nil {
			log.Warn("StuckJobRescuer: rescue query failed",
				zap.String("tenant_id", tenant.ID.String()),
				zap.Error(err))
		}
	}
}

// rescueStuckJobsForSchema rescues stuck "running" jobs in a single tenant's
// River schema. It only touches jobs whose attempted_at is older than the
// staleness threshold, so jobs actively being executed by a live worker are
// left alone.
func (m *Manager) rescueStuckJobsForSchema(tenantID uuid.UUID) error {
	schema := db.GetSchemaName(tenantID)

	sql := fmt.Sprintf(
		"UPDATE %s.river_job SET state = 'available', attempted_at = NULL, finalized_at = NULL "+
			"WHERE state = 'running' AND finalized_at IS NULL "+
			"AND attempted_at < now() - interval '%d minutes'",
		schema, staleRunningMinutes,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	tag, err := db.PgxPool.Exec(ctx, sql)
	if err != nil {
		return err
	}

	if n := tag.RowsAffected(); n > 0 {
		logger.GetLogger().Info("StuckJobRescuer: rescued stuck running jobs",
			zap.String("schema", schema),
			zap.Int64("rescued", n))
	}
	return nil
}
