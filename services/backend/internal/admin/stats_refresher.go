package admin

import (
	"context"
	"time"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/pkg/logger"
	"go.uber.org/zap"
)

// StartStatisticsRefresher starts a background goroutine that periodically
// calls refresh_admin_statistics() to keep the admin_statistics table current.
// interval specifies how often to refresh (e.g., 1*time.Minute).
func StartStatisticsRefresher(ctx context.Context, interval time.Duration) {
	log := logger.GetLogger()
	log.Info("Admin statistics refresher started", zap.Duration("interval", interval))

	// Do an initial refresh immediately on startup
	if err := refreshAdminStats(); err != nil {
		log.Warn("Initial admin statistics refresh failed", zap.Error(err))
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("Admin statistics refresher stopped")
			return
		case <-ticker.C:
			if err := refreshAdminStats(); err != nil {
				log.Error("Admin statistics refresh failed", zap.Error(err))
			}
		}
	}
}

// refreshAdminStats calls the SQL function that atomically updates admin_statistics
func refreshAdminStats() error {
	return db.GormDB.Exec("SELECT refresh_admin_statistics()").Error
}
