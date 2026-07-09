package dedicated_gateway

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
)

type Gateway struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	APIKey      string    `json:"api_key"`
	TenantID    uuid.UUID `json:"tenant_id"`
	CreatedAt   time.Time `json:"created_at"`
}

type ResourceQuota struct {
	MaxMessagesPerDay    int `json:"max_messages_per_day"`
	MaxAPIcallsPerMinute int `json:"max_api_calls_per_minute"`
	MaxConcurrentSessions int `json:"max_concurrent_sessions"`
}

type ResourceUsage struct {
	GatewayID    uuid.UUID `json:"gateway_id"`
	Date         time.Time `json:"date"`
	MessagesSent int       `json:"messages_sent"`
	APICalls     int       `json:"api_calls"`
}

type UsageStats struct {
	TotalMessages  int     `json:"total_messages"`
	TotalAPICalls  int     `json:"total_api_calls"`
	SuccessRate    float64 `json:"success_rate"`
	AverageLatency int     `json:"average_latency"`
}

type Repository interface {
	CreateGateway(ctx context.Context, gw *Gateway) error
	GetGatewayByID(ctx context.Context, gwID uuid.UUID) (*Gateway, error)
	GetGatewaysByTenant(ctx context.Context, tenantID uuid.UUID) ([]Gateway, error)
	UpdateGatewayStatus(ctx context.Context, gwID uuid.UUID, status string) error
	UpdateGatewayQuota(ctx context.Context, gwID uuid.UUID, quota *ResourceQuota) error
	RecordUsage(ctx context.Context, usage *ResourceUsage) error
	GetUsageStats(ctx context.Context, gwID uuid.UUID, start, end time.Time) (*UsageStats, error)
}

type Manager struct {
	repo Repository
}

func NewManager(repo Repository) *Manager {
	return &Manager{repo: repo}
}

func (m *Manager) CreateGateway(ctx context.Context, tenantID uuid.UUID, gw *Gateway) (*Gateway, error) {
	gw.ID = uuid.New()
	gw.TenantID = tenantID
	gw.APIKey = m.GenerateAPIKey()
	gw.CreatedAt = time.Now()
	if gw.Status == "" {
		gw.Status = "PENDING"
	}

	if m.repo != nil {
		if err := m.repo.CreateGateway(ctx, gw); err != nil {
			return nil, err
		}
	}

	return gw, nil
}

func (m *Manager) ActivateGateway(ctx context.Context, gwID uuid.UUID) error {
	if m.repo != nil {
		return m.repo.UpdateGatewayStatus(ctx, gwID, "ACTIVE")
	}
	return nil
}

func (m *Manager) GetGatewayByID(ctx context.Context, gwID uuid.UUID) (*Gateway, error) {
	if m.repo != nil {
		return m.repo.GetGatewayByID(ctx, gwID)
	}
	return nil, nil
}

func (m *Manager) SetResourceQuota(ctx context.Context, gwID uuid.UUID, quota *ResourceQuota) error {
	if m.repo != nil {
		return m.repo.UpdateGatewayQuota(ctx, gwID, quota)
	}
	return nil
}

func (m *Manager) RecordUsage(ctx context.Context, usage *ResourceUsage) error {
	if m.repo != nil {
		return m.repo.RecordUsage(ctx, usage)
	}
	return nil
}

func (m *Manager) GetUsageStats(ctx context.Context, gwID uuid.UUID, start, end time.Time) (*UsageStats, error) {
	if m.repo != nil {
		return m.repo.GetUsageStats(ctx, gwID, start, end)
	}
	return nil, nil
}

func (m *Manager) GenerateAPIKey() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (m *Manager) ValidateQuota(quota *ResourceQuota, usage *ResourceUsage) error {
	if usage.MessagesSent > quota.MaxMessagesPerDay {
		return errors.New("messages quota exceeded")
	}
	return nil
}

func (m *Manager) SuspendGateway(ctx context.Context, gwID uuid.UUID, reason string) error {
	if m.repo != nil {
		return m.repo.UpdateGatewayStatus(ctx, gwID, "SUSPENDED")
	}
	return nil
}

func (m *Manager) GetGatewaysByTenant(ctx context.Context, tenantID uuid.UUID) ([]Gateway, error) {
	if m.repo != nil {
		return m.repo.GetGatewaysByTenant(ctx, tenantID)
	}
	return nil, nil
}
