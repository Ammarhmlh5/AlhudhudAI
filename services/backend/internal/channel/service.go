package channel

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/al-hudhud/backend/internal/redis"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Service handles DispatchChannel CRUD operations with full tenant isolation.
type Service struct {
	channelCache *redis.ChannelCache
	logger       *zap.Logger
	stopChan     chan struct{}
}

func NewService(channelCache *redis.ChannelCache, logger *zap.Logger) *Service {
	return &Service{
		channelCache: channelCache,
		logger:       logger,
		stopChan:     make(chan struct{}),
	}
}

// CreateChannelRequest defines the input for creating a new dispatch channel.
type CreateChannelRequest struct {
	Type       string `json:"type" binding:"required"`
	ChannelKey string `json:"channel_key" binding:"required"`
	Config     string `json:"config"`
	Priority   int    `json:"priority"`
}

// UpdateChannelRequest defines the input for updating a dispatch channel.
type UpdateChannelRequest struct {
	Priority *int    `json:"priority"`
	Status   *string `json:"status"`
	Config   *string `json:"config"`
}

// ListChannels returns all dispatch channels for a tenant, ordered by priority DESC.
// Uses the composite index (tenant_id, status, priority) for maximum speed.
func (s *Service) ListChannels(ctx context.Context, tenantID uuid.UUID) ([]models.DispatchChannel, error) {
	var channels []models.DispatchChannel
	err := db.RunInTenantScope(tenantID, false, func(tx *gorm.DB) error {
		return tx.Where("tenant_id = ?", tenantID).
			Order("priority DESC, created_at ASC").
			Find(&channels).Error
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list channels: %w", err)
	}
	return channels, nil
}

// CreateChannel adds a new dispatch channel for a tenant.
func (s *Service) CreateChannel(ctx context.Context, tenantID uuid.UUID, req CreateChannelRequest) (*models.DispatchChannel, error) {
	config := req.Config
	if config == "" {
		config = "{}"
	}
	priority := req.Priority
	if priority == 0 {
		priority = 5
	}

	ch := &models.DispatchChannel{
		TenantID:   tenantID,
		Type:       models.ChannelType(req.Type),
		ChannelKey: req.ChannelKey,
		Config:     config,
		Priority:   priority,
		Status:     "ONLINE",
	}

	err := db.RunInTenantScope(tenantID, false, func(tx *gorm.DB) error {
		// Use ON CONFLICT to avoid duplicate key errors.
		result := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "tenant_id"}, {Name: "channel_key"}}, DoNothing: true}).Create(ch)
		if result.Error != nil {
			return result.Error
		}
		// If no row was inserted, the channel already exists – fetch it.
		if result.RowsAffected == 0 {
			return tx.Where("tenant_id = ? AND channel_key = ?", tenantID, req.ChannelKey).First(ch).Error
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create channel: %w", err)
	}
	return ch, nil
}

// UpdateChannel updates priority, status, or config of an existing channel.
// This is a non-blocking operation: it does not affect in-flight messages.
func (s *Service) UpdateChannel(ctx context.Context, tenantID uuid.UUID, channelID uuid.UUID, req UpdateChannelRequest) (*models.DispatchChannel, error) {
	var ch models.DispatchChannel
	var statusChanged bool
	var newStatus string

	err := db.RunInTenantScope(tenantID, false, func(tx *gorm.DB) error {
		// Fetch with tenant isolation guard
		if err := tx.Where("id = ? AND tenant_id = ?", channelID, tenantID).First(&ch).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("channel not found")
			}
			return err
		}

		updates := map[string]interface{}{}
		if req.Priority != nil {
			updates["priority"] = *req.Priority
		}
		if req.Status != nil {
			newStatus = *req.Status
			// Check if status is changing to offline
			if ch.Status != newStatus && newStatus == "OFFLINE" {
				statusChanged = true
			}
			updates["status"] = newStatus
		}
		if req.Config != nil {
			updates["config"] = *req.Config
		}

		if len(updates) == 0 {
			return nil
		}

		return tx.Model(&ch).Updates(updates).Error
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update channel: %w", err)
	}

	// Invalidate cache if status changed to offline (Bug 13 - Task 15.2)
	if statusChanged && s.channelCache != nil {
		if err := s.channelCache.InvalidateChannelOnOffline(ctx, tenantID); err != nil {
			// Log error but don't fail the update - cache invalidation is best-effort
			if s.logger != nil {
				s.logger.Warn("failed to invalidate channel cache on offline",
					zap.String("tenant_id", tenantID.String()),
					zap.String("channel_id", channelID.String()),
					zap.Error(err),
				)
			}
		} else {
			// Log successful cache invalidation
			if s.logger != nil {
				s.logger.Info("channel cache invalidated on offline",
					zap.String("tenant_id", tenantID.String()),
					zap.String("channel_id", channelID.String()),
				)
			}
		}
	}

	return &ch, nil
}

// DeleteChannel removes a dispatch channel for a tenant.
func (s *Service) DeleteChannel(ctx context.Context, tenantID uuid.UUID, channelID uuid.UUID) error {
	err := db.RunInTenantScope(tenantID, false, func(tx *gorm.DB) error {
		result := tx.Where("id = ? AND tenant_id = ?", channelID, tenantID).Delete(&models.DispatchChannel{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errors.New("channel not found")
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to delete channel: %w", err)
	}
	return nil
}

// StartHealthCheck starts a background goroutine that checks channel health every 1 minute
// and triggers cache invalidation for offline channels (Bug 13 - Task 15.3)
func (s *Service) StartHealthCheck(ctx context.Context) {
	if s.channelCache == nil {
		if s.logger != nil {
			s.logger.Warn("channel health check disabled: cache not available")
		}
		return
	}

	if s.logger != nil {
		s.logger.Info("starting channel health check background task")
	}

	ticker := time.NewTicker(1 * time.Minute)
	
	go func() {
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				s.performHealthCheck(ctx)
			case <-s.stopChan:
				if s.logger != nil {
					s.logger.Info("channel health check stopped")
				}
				return
			case <-ctx.Done():
				if s.logger != nil {
					s.logger.Info("channel health check stopped due to context cancellation")
				}
				return
			}
		}
	}()
}

// StopHealthCheck gracefully stops the health check goroutine
func (s *Service) StopHealthCheck() {
	close(s.stopChan)
}

// performHealthCheck checks all channels and invalidates cache for offline ones
func (s *Service) performHealthCheck(ctx context.Context) {
	if s.logger != nil {
		s.logger.Debug("performing channel health check")
	}

	// Get all tenants with channels
	var channels []models.DispatchChannel
	err := db.GormDB.Where("status = ?", "OFFLINE").Find(&channels).Error
	if err != nil {
		if s.logger != nil {
			s.logger.Error("failed to query offline channels during health check",
				zap.Error(err),
			)
		}
		return
	}

	if len(channels) == 0 {
		if s.logger != nil {
			s.logger.Debug("health check: no offline channels found")
		}
		return
	}

	// Track unique tenant IDs to avoid duplicate invalidations
	tenantMap := make(map[uuid.UUID]bool)
	for _, ch := range channels {
		tenantMap[ch.TenantID] = true
	}

	// Invalidate cache for each tenant with offline channels
	invalidatedCount := 0
	for tenantID := range tenantMap {
		err := s.channelCache.InvalidateChannelOnOffline(ctx, tenantID)
		if err != nil {
			if s.logger != nil {
				s.logger.Warn("failed to invalidate cache during health check",
					zap.String("tenant_id", tenantID.String()),
					zap.Error(err),
				)
			}
		} else {
			invalidatedCount++
		}
	}

	if s.logger != nil {
		s.logger.Info("channel health check completed",
			zap.Int("offline_channels", len(channels)),
			zap.Int("tenants_invalidated", invalidatedCount),
		)
	}
}

// RegenerateChannelKey generates a new unique key for the channel
func (s *Service) RegenerateChannelKey(ctx context.Context, tenantID uuid.UUID, channelID uuid.UUID) (*models.DispatchChannel, error) {
	var ch models.DispatchChannel
	
	// Generate a new 16-char hex key (similar to generatePairKey but shorter)
	newKey := generateShortKey()

	err := db.RunInTenantScope(tenantID, false, func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND tenant_id = ?", channelID, tenantID).First(&ch).Error; err != nil {
			return err
		}
		
		ch.ChannelKey = newKey
		return tx.Save(&ch).Error
	})
	
	if err != nil {
		return nil, err
	}
	
	return &ch, nil
}

func generateShortKey() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}
