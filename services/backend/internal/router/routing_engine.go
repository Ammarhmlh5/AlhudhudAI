package router

import (
	"context"
	"errors"
	"fmt"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RoutingEngine struct{}

func NewRoutingEngine() *RoutingEngine {
	return &RoutingEngine{}
}

// GetBestChannel selects the most appropriate channel for a message
func (re *RoutingEngine) GetBestChannel(ctx context.Context, tenantID uuid.UUID, message *models.Message) (*models.DispatchChannel, error) {
	var channel models.DispatchChannel

	err := db.RunInTenantScope(tenantID, false, func(tx *gorm.DB) error {
		// Defense in depth: explicitly filter by tenant_id
		baseQuery := tx.Where("tenant_id = ?", tenantID)

		// 1. Check if a specific channel was requested via the message (future-proofing)
		if message.ChannelID != nil {
			if err := baseQuery.Where("id = ?", message.ChannelID).
				First(&channel).Error; err == nil {
				if channel.Status == "ONLINE" {
					return nil
				}
			}
		}

		// 2. Find the highest priority ONLINE channel for this tenant
		err := baseQuery.Where("status = ?", "ONLINE").
			Order("priority DESC, created_at ASC").
			First(&channel).Error

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("WAIT_MODE_REQUIRED") // Signal that we need to enter Wait Mode
			}
			return err
		}

		return nil
	})

	if err != nil {
		if err.Error() == "WAIT_MODE_REQUIRED" {
			return nil, err
		}
		return nil, fmt.Errorf("failed to route message: %w", err)
	}

	return &channel, nil
}
