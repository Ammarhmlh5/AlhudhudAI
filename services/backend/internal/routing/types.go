package routing

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Channel represents a sending channel (device or gateway)
type Channel struct {
	ID          uuid.UUID  `json:"id"`
	TenantID     uuid.UUID  `json:"tenant_id"`
	GatewayID   *uuid.UUID `json:"gateway_id"` // Link to dedicated gateway
	Type        string     `json:"type"`        // PHONE, GATEWAY_HUDHUD, GATEWAY_PRIVATE
	ChannelKey  string     `json:"channel_key"` // Unique identifier for the channel
	Status      string     `json:"status"`      // ONLINE, OFFLINE, WAIT_MODE
	Priority    int        `json:"priority"`    // Higher value = higher priority
	CurrentLoad int        `json:"current_load"` // Current number of messages being processed
	Config      string     `json:"config"`      // JSONB configuration
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// RoutingEngine interface defines the contract for channel selection
// Requirements: 5.1, 5.2, 5.3, 5.4, 5.6
type RoutingEngine interface {
	// SelectChannel selects the best available channel for a tenant
	// Searches in Cache first, then Database on cache miss (Requirements 5.1, 5.2)
	// Filters channels by ONLINE status only (Requirement 5.3)
	// Orders by Priority DESC, then CurrentLoad ASC (Requirement 5.4)
	SelectChannel(ctx context.Context, tenantID uuid.UUID) (*Channel, error)

	// GetAvailableChannels returns all available channels for a tenant
	// Uses Redis cache with 5-minute TTL (Requirements 5.1, 5.2)
	// Filtered by ONLINE status only
	GetAvailableChannels(ctx context.Context, tenantID uuid.UUID) ([]*Channel, error)

	// IncrementChannelLoad increments the current load of a channel
	// Used when a message is assigned to a channel (Requirement 5.6)
	// Invalidates cache to ensure accurate load balancing
	IncrementChannelLoad(ctx context.Context, channelID uuid.UUID, tenantID uuid.UUID) error

	// DecrementChannelLoad decrements the current load of a channel
	// Used when a message is completed or failed (Requirement 5.6)
	// Invalidates cache to ensure accurate load balancing
	DecrementChannelLoad(ctx context.Context, channelID uuid.UUID, tenantID uuid.UUID) error

	// UpdateChannelLoad updates the current load of a channel to a specific value
	// Invalidates cache to ensure accurate load balancing (Requirement 5.6)
	UpdateChannelLoad(ctx context.Context, channelID uuid.UUID, tenantID uuid.UUID, load int) error

	// SetMessageWaiting updates message status to WAITING when no channels available
	// Used when SelectChannel returns ErrNoChannelAvailable (Requirement 5.5)
	SetMessageWaiting(ctx context.Context, messageID uuid.UUID, tenantID uuid.UUID) error

	// VerifyChannelOnline checks if a channel is actually online in the database
	// Used for health check verification (Bug 23 - Task 23.2.2)
	// Returns true if channel is online, false otherwise
	VerifyChannelOnline(ctx context.Context, channelID uuid.UUID, tenantID uuid.UUID) (bool, error)
}

// ChannelStatus constants
const (
	ChannelStatusOnline   = "ONLINE"
	ChannelStatusOffline  = "OFFLINE"
	ChannelStatusWaitMode = "WAIT_MODE"
)

// Error definitions
var (
	ErrNoChannelAvailable = &RoutingError{
		Code:    "NO_CHANNEL_AVAILABLE",
		Message: "no online channels available for tenant",
	}
	ErrTenantNotFound = &RoutingError{
		Code:    "TENANT_NOT_FOUND",
		Message: "tenant not found",
	}
)

// RoutingError represents a routing-specific error
type RoutingError struct {
	Code    string
	Message string
}

func (e *RoutingError) Error() string {
	return e.Message
}
