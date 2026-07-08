package auth

import (
	"context"
	"time"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/models"
	"go.uber.org/zap"
)

// StartSessionCleanupJob starts a background goroutine that periodically
// deletes expired sessions from the database.
func StartSessionCleanupJob(ctx context.Context, logger *zap.Logger, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				cleanupExpiredSessions(logger)
			}
		}
	}()
}

func cleanupExpiredSessions(logger *zap.Logger) {
	result := db.GormDB.Where("expires_at < ?", time.Now()).Delete(&models.Session{})
	if result.Error != nil {
		logger.Error("Failed to cleanup expired sessions", zap.Error(result.Error))
		return
	}
	
	if result.RowsAffected > 0 {
		logger.Info("Cleaned up expired sessions", zap.Int64("count", result.RowsAffected))
	}
}
