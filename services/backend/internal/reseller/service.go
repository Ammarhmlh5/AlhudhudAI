package reseller

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/al-hudhud/backend/internal/balance"
	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/al-hudhud/backend/internal/subscription"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// maxCreditEnvKey controls the maximum amount a single reseller funding
// operation may move. 0 or unset means unlimited.
const maxCreditEnvKey = "RESELLER_MAX_CREDIT"

// ErrCreditLimitExceeded is returned by CreditReseller when the requested
// amount exceeds the configured RESELLER_MAX_CREDIT limit.
var ErrCreditLimitExceeded = errors.New("credit amount exceeds maximum allowed")

type Service struct {
	db           *gorm.DB
	pool         *pgxpool.Pool
	balance      balance.Manager
	subscription subscription.Manager
	logger       *zap.Logger
}

func NewService(db *gorm.DB, pool *pgxpool.Pool, balance balance.Manager, subscription subscription.Manager, logger *zap.Logger) *Service {
	return &Service{
		db:           db,
		pool:         pool,
		balance:      balance,
		subscription: subscription,
		logger:       logger,
	}
}

// ResellerSummary describes a software company (reseller) for the admin interface
// along with the number of customer companies connected to it.
type ResellerSummary struct {
	models.Tenant
	ConnectedCompanies int64 `json:"connected_companies"`
}

// ListResellers returns all reseller tenants (is_reseller = true) for the admin interface,
// including a count of customer companies linked via parent_id.
func (s *Service) ListResellers(ctx context.Context) ([]ResellerSummary, error) {
	var resellers []models.Tenant
	if err := s.db.WithContext(ctx).Where("is_reseller = ?", true).Order("created_at DESC").Find(&resellers).Error; err != nil {
		return nil, err
	}

	summaries := make([]ResellerSummary, 0, len(resellers))
	for _, r := range resellers {
		var count int64
		s.db.WithContext(ctx).Model(&models.Tenant{}).Where("parent_id = ?", r.ID).Count(&count)
		summaries = append(summaries, ResellerSummary{Tenant: r, ConnectedCompanies: count})
	}
	return summaries, nil
}

// ListTenants returns the customer companies linked to a reseller.
func (s *Service) ListTenants(ctx context.Context, resellerID uuid.UUID) ([]models.Tenant, error) {
	var tenants []models.Tenant
	q := s.db.WithContext(ctx)
	if resellerID != uuid.Nil {
		q = q.Where("parent_id = ?", resellerID)
	}
	err := q.Find(&tenants).Error
	return tenants, err
}

// TenantWithDetails enriches a customer company with its wallet balance and
// current subscription status for the reseller portal.
type TenantWithDetails struct {
	models.Tenant
	Balance           string `json:"balance"`
	Subscription      string `json:"subscription"` // "active", "grace_period", "none"
	SubscriptionLabel string `json:"subscription_label"`
}

// ListTenantsWithDetails returns the reseller's customer companies enriched
// with wallet balance and subscription status.
func (s *Service) ListTenantsWithDetails(ctx context.Context, resellerID uuid.UUID) ([]TenantWithDetails, error) {
	tenants, err := s.ListTenants(ctx, resellerID)
	if err != nil {
		return nil, err
	}

	details := make([]TenantWithDetails, 0, len(tenants))
	for i := range tenants {
		td := TenantWithDetails{Tenant: tenants[i], Subscription: "none"}

		if bal, err := s.balance.GetBalance(ctx, tenants[i].ID); err == nil {
			td.Balance = bal.String()
		} else {
			td.Balance = "0"
		}

		if sub, err := s.subscription.GetActiveSubscription(ctx, tenants[i].ID); err == nil {
			td.Subscription = string(sub.Status)
		}

		details = append(details, td)
	}
	return details, nil
}

// CreateTenant creates a customer company (tenant) under a reseller.
// It generates the ACC/PHO/WEB API keys, a company serial, and provisions the
// tenant's database schema exactly like the admin creation flow.
func (s *Service) CreateTenant(ctx context.Context, resellerID uuid.UUID, tenant *models.Tenant, user *models.User) error {
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Create User
		user.UserType = models.TenantType
		if err := tx.Create(user).Error; err != nil {
			return err
		}

		// 2. Setup Tenant
		tenant.UserID = user.ID
		// Super admins (uuid.Nil) create top-level tenants without a parent
		if resellerID != uuid.Nil {
			tenant.ParentID = &resellerID
		}
		tenant.IsReseller = false
		if tenant.CompanySerial == "" {
			var count int64
			tx.Model(&models.Tenant{}).Count(&count)
			tenant.CompanySerial = fmt.Sprintf("HUD-%d-%05d", time.Now().Year(), count+1)
		}

		// 3. Generate API keys for the tenant
		tenant.APIKeyAccounting = s.generateAPIKey("ACC")
		tenant.APIKeyPhone = s.generateAPIKey("PHO")
		tenant.APIKeyWeb = s.generateAPIKey("WEB")

		if err := tx.Create(tenant).Error; err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	// 4. Create and migrate the tenant schema (best-effort, matches admin flow)
	if err := db.CreateTenantSchema(tenant.ID); err != nil {
		s.logger.Error("Failed to create tenant schema",
			zap.String("tenant_id", tenant.ID.String()),
			zap.Error(err),
		)
	} else if err := db.MigrateTenantSchema(tenant.ID); err != nil {
		s.logger.Error("Failed to migrate tenant schema",
			zap.String("tenant_id", tenant.ID.String()),
			zap.Error(err),
		)
	}

	return nil
}

func (s *Service) generateAPIKey(prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, uuid.New().String())
}

// GetTenant returns a customer company if it belongs to the reseller.
func (s *Service) GetTenant(ctx context.Context, resellerID, tenantID uuid.UUID) (*models.Tenant, error) {
	var tenant models.Tenant
	q := s.db.WithContext(ctx)
	if resellerID != uuid.Nil {
		q = q.Where("id = ? AND parent_id = ?", tenantID, resellerID)
	} else {
		q = q.Where("id = ?", tenantID)
	}
	err := q.First(&tenant).Error
	if err != nil {
		return nil, err
	}
	return &tenant, nil
}

// UpdateTenantAPIKeys updates the accounting API key of a customer company.
func (s *Service) UpdateTenantAPIKeys(ctx context.Context, resellerID, tenantID uuid.UUID, accountingKey string) error {
	q := s.db.WithContext(ctx).Model(&models.Tenant{})
	if resellerID != uuid.Nil {
		q = q.Where("id = ? AND parent_id = ?", tenantID, resellerID)
	} else {
		q = q.Where("id = ?", tenantID)
	}
	result := q.Update("api_key_accounting", accountingKey)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("tenant not found or not owned by reseller")
	}
	return nil
}

// ListStaff returns the reseller's staff profiles.
func (s *Service) ListStaff(ctx context.Context, resellerID uuid.UUID) ([]models.StaffProfile, error) {
	var staff []models.StaffProfile
	q := s.db.WithContext(ctx).Preload("User").Preload("Role")
	if resellerID != uuid.Nil {
		q = q.Where("tenant_id = ?", resellerID)
	}
	err := q.Find(&staff).Error
	return staff, err
}

// GetTenantBalance returns the wallet balance of a customer company.
func (s *Service) GetTenantBalance(ctx context.Context, resellerID, tenantID uuid.UUID) (decimal.Decimal, error) {
	if _, err := s.GetTenant(ctx, resellerID, tenantID); err != nil {
		return decimal.Zero, err
	}
	return s.balance.GetBalance(ctx, tenantID)
}

// GetWalletBalance returns the reseller's own wallet balance.
func (s *Service) GetWalletBalance(ctx context.Context, resellerID uuid.UUID) (decimal.Decimal, error) {
	if resellerID == uuid.Nil {
		return decimal.Zero, errors.New("reseller identity is required")
	}
	return s.balance.GetBalance(ctx, resellerID)
}

// TransferToTenant moves funds from the reseller wallet to a customer company wallet.
func (s *Service) TransferToTenant(ctx context.Context, resellerID, tenantID uuid.UUID, amount decimal.Decimal, reason, createdBy, ipAddress string) (*balance.TransferResult, error) {
	if resellerID == uuid.Nil {
		return nil, errors.New("reseller identity is required for transfer")
	}
	if _, err := s.GetTenant(ctx, resellerID, tenantID); err != nil {
		return nil, err
	}
	return s.balance.TransferBalance(ctx, resellerID, tenantID, amount, reason, createdBy, ipAddress)
}

// SubscribeTenant subscribes a customer company to a package, charging the
// customer company's wallet (exactly like the tenant subscribing themselves).
func (s *Service) SubscribeTenant(ctx context.Context, resellerID, tenantID uuid.UUID, packageID int64, autoRenew bool) (*subscription.SubscribeResult, error) {
	if _, err := s.GetTenant(ctx, resellerID, tenantID); err != nil {
		return nil, err
	}

	result, err := s.subscription.Subscribe(ctx, subscription.SubscribeRequest{
		TenantID:  tenantID,
		PackageID: packageID,
		AutoRenew: autoRenew,
	})
	if err != nil {
		return nil, err
	}

	// Deactivate any previous active/grace subscriptions so only the new one counts.
	if result.Success {
		err = db.RunInTenantPgxTx(ctx, tenantID, true, s.pool, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx,
				`UPDATE subscriptions
				 SET status = 'expired', updated_at = NOW()
				 WHERE tenant_id = $1 AND id != $2 AND status IN ('active', 'grace_period')`,
				tenantID, result.SubscriptionID,
			)
			return err
		})
		if err != nil {
			s.logger.Warn("Failed to deactivate previous subscriptions",
				zap.String("tenant_id", tenantID.String()),
				zap.Error(err),
			)
		}
	}

	return result, nil
}

// maxCreditAllowed returns the configured single-operation funding cap for
// reseller wallets. A non-positive value means unlimited.
func (s *Service) maxCreditAllowed() decimal.Decimal {
	if v := os.Getenv(maxCreditEnvKey); v != "" {
		if d, err := decimal.NewFromString(v); err == nil && d.IsPositive() {
			return d
		}
	}
	return decimal.Zero
}

// CreditReseller funds a reseller's wallet (admin operation).
func (s *Service) CreditReseller(ctx context.Context, resellerID uuid.UUID, amount decimal.Decimal, reason, createdBy, ipAddress string) (*balance.BalanceResult, error) {
	if resellerID == uuid.Nil {
		return nil, errors.New("reseller id is required")
	}

	var reseller models.Tenant
	if err := s.db.WithContext(ctx).Where("id = ? AND is_reseller = ?", resellerID, true).First(&reseller).Error; err != nil {
		return nil, err
	}

	if max := s.maxCreditAllowed(); max.IsPositive() && amount.GreaterThan(max) {
		s.logger.Warn("reseller credit rejected: exceeds configured limit",
			zap.String("reseller_id", resellerID.String()),
			zap.String("amount", amount.String()),
			zap.String("max", max.String()),
			zap.String("actor", createdBy),
			zap.String("ip", ipAddress),
		)
		return nil, ErrCreditLimitExceeded
	}

	return s.balance.CreditBalance(ctx, balance.BalanceOperation{
		TenantID:  resellerID,
		Amount:    amount,
		Reason:    reason,
		CreatedBy: createdBy,
		IPAddress: ipAddress,
	})
}
