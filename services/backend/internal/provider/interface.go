package provider

import (
	"context"

	"github.com/al-hudhud/backend/internal/models"
	"github.com/google/uuid"
)

// Provider defines the interface for different message dispatch paths
type Provider interface {
	// Send dispatches a message through the provider
	// Returns a provider-specific job ID and an error
	Send(ctx context.Context, tenantID uuid.UUID, msg *models.Message, config string) (string, error)

	// GetType returns the type of the provider
	GetType() models.ChannelType
}

// Registry holds all registered providers
type Registry struct {
	providers map[models.ChannelType]Provider
}

func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[models.ChannelType]Provider),
	}
}

func (r *Registry) Register(p Provider) {
	r.providers[p.GetType()] = p
}

func (r *Registry) Get(t models.ChannelType) Provider {
	return r.providers[t]
}
