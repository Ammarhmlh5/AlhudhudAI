package tenant

import (
	"context"
	"errors"
	"regexp"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	ErrTenantNotFound = errors.New("tenant not found")
	ErrInvalidBalance = errors.New("invalid balance: cannot be negative")
)

type Tenant struct {
	ID      uuid.UUID       `json:"id"`
	Name    string          `json:"name"`
	Email   string          `json:"email"`
	Balance decimal.Decimal `json:"balance"`
	Status  string          `json:"status"`
}

type TenantRepository interface {
	GetTenantByID(ctx context.Context, tenantID uuid.UUID) (*Tenant, error)
	GetTenantByEmail(ctx context.Context, email string) (*Tenant, error)
	CreateTenant(ctx context.Context, tenant *Tenant) error
	UpdateTenantStatus(ctx context.Context, tenantID uuid.UUID, status string) error
	UpdateTenantBalance(ctx context.Context, tenantID uuid.UUID, newBalance decimal.Decimal) error
	DeleteTenant(ctx context.Context, tenantID uuid.UUID) error
	ListTenants(ctx context.Context, limit, offset int) ([]Tenant, error)
}

type Manager struct {
	repo TenantRepository
}

func NewManager(repo TenantRepository) *Manager {
	return &Manager{repo: repo}
}

func (m *Manager) GetTenantByID(ctx context.Context, tenantID uuid.UUID) (*Tenant, error) {
	return m.repo.GetTenantByID(ctx, tenantID)
}

func (m *Manager) GetTenantByEmail(ctx context.Context, email string) (*Tenant, error) {
	return m.repo.GetTenantByEmail(ctx, email)
}

func (m *Manager) CreateTenant(ctx context.Context, tenant *Tenant) error {
	if tenant.ID == uuid.Nil {
		tenant.ID = uuid.New()
	}
	return m.repo.CreateTenant(ctx, tenant)
}

func (m *Manager) ValidateTenantData(name, email string) error {
	if name == "" {
		return errors.New("name cannot be empty")
	}
	if email == "" {
		return errors.New("email cannot be empty")
	}
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	if !re.MatchString(email) {
		return errors.New("invalid email format")
	}
	return nil
}

func (m *Manager) SuspendTenant(ctx context.Context, tenantID uuid.UUID) error {
	return m.repo.UpdateTenantStatus(ctx, tenantID, "SUSPENDED")
}

func (m *Manager) ActivateTenant(ctx context.Context, tenantID uuid.UUID) error {
	return m.repo.UpdateTenantStatus(ctx, tenantID, "ACTIVE")
}

func (m *Manager) ListTenants(ctx context.Context, limit, offset int) ([]Tenant, error) {
	return m.repo.ListTenants(ctx, limit, offset)
}

func (m *Manager) DeleteTenant(ctx context.Context, tenantID uuid.UUID) error {
	return m.repo.DeleteTenant(ctx, tenantID)
}

func (m *Manager) UpdateBalance(ctx context.Context, tenantID uuid.UUID, newBalance decimal.Decimal) error {
	if newBalance.IsNegative() {
		return ErrInvalidBalance
	}
	return m.repo.UpdateTenantBalance(ctx, tenantID, newBalance)
}
