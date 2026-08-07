package reseller_test

import (
	"context"
	"testing"

	"github.com/al-hudhud/backend/internal/balance"
	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/al-hudhud/backend/internal/reseller"
	"github.com/al-hudhud/backend/internal/subscription"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// createResellerTenant inserts a reseller tenant (is_reseller = true) with a zero
// wallet balance and returns its tenant ID.
func createResellerTenant(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	ctx := context.Background()

	userID := uuid.New()
	email := "reseller-integration-" + uuid.New().String() + "@test.local"
	_, err := pool.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, user_type, is_active, is_verified, created_at, updated_at)
		VALUES ($1, $2, 'x', 'RESELLER', true, true, NOW(), NOW())
	`, userID, email)
	require.NoError(t, err)

	tenantID := uuid.New()
	_, err = pool.Exec(ctx, `
		INSERT INTO tenants (id, user_id, business_name, contact_name, email, company_serial,
			api_key_accounting, api_key_phone, api_key_web, balance, status, is_active,
			is_reseller, parent_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 0, 'ACTIVE', true, true, NULL, NOW(), NOW())
	`,
		tenantID, userID,
		"Integration Reseller "+uuid.New().String()[:8],
		"Integration Contact",
		email,
		"RSL-INT-"+tenantID.String(),
		"acc-"+tenantID.String(),
		"phone-"+tenantID.String(),
		"web-"+tenantID.String(),
	)
	require.NoError(t, err)

	return tenantID
}

// createIntegrationPackage inserts an active monthly package and returns its ID.
func createIntegrationPackage(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) int64 {
	t.Helper()
	ctx := context.Background()

	var pkgID int64
	err := pool.QueryRow(ctx, `
		INSERT INTO packages (tenant_id, service_id, name, code, price_usd, included_messages,
			overage_rate, is_unlimited, billing_cycle, rollover_enabled, is_active, display_order)
		VALUES ($1, 1, 'Integration Pkg', $2, 50.00, 1000, 0.01, false, 'monthly', true, true, 1)
		RETURNING id
	`, tenantID, "PKG_INT_"+uuid.New().String()[:8]).Scan(&pkgID)
	require.NoError(t, err)

	return pkgID
}

// cleanupIntegrationFlow removes every record created by the full-flow test.
func cleanupIntegrationFlow(t *testing.T, pool *pgxpool.Pool, resellerID, otherID, tenantID uuid.UUID, pkgID int64) {
	t.Helper()
	ctx := context.Background()

	ids := []uuid.UUID{tenantID, resellerID, otherID}
	for _, id := range ids {
		_, _ = pool.Exec(ctx, "DELETE FROM balance_history WHERE tenant_id = $1", id)
		_, _ = pool.Exec(ctx, "DELETE FROM accounting_transactions WHERE tenant_id = $1", id)
		_, _ = pool.Exec(ctx, "DELETE FROM subscriptions WHERE tenant_id = $1", id)
	}

	var userIDs []uuid.UUID
	rows, err := pool.Query(ctx, "SELECT user_id FROM tenants WHERE id = ANY($1)", ids)
	if err == nil {
		for rows.Next() {
			var uid uuid.UUID
			if err := rows.Scan(&uid); err == nil {
				userIDs = append(userIDs, uid)
			}
		}
		rows.Close()
	}

	_, _ = pool.Exec(ctx, "DELETE FROM packages WHERE id = $1", pkgID)
	_, _ = pool.Exec(ctx, "DELETE FROM tenants WHERE id = ANY($1)", ids)
	if len(userIDs) > 0 {
		_, _ = pool.Exec(ctx, "DELETE FROM users WHERE id = ANY($1)", userIDs)
	}
	_, _ = pool.Exec(ctx, "DROP SCHEMA IF EXISTS "+db.GetSchemaName(tenantID)+" CASCADE")
}

// TestResellerFullFlow_Integration exercises the complete reseller lifecycle with
// real balance + subscription managers against the test database:
//   admin funds reseller -> reseller creates tenant -> transfers -> subscribes.
//   It also verifies cross-reseller isolation.
func TestResellerFullFlow_Integration(t *testing.T) {
	ctx := context.Background()
	pool, gdb := setupResellerTestDB(t)
	defer pool.Close()

	// Route schema provisioning (db.CreateTenantSchema) to the test database.
	prevGormDB := db.GormDB
	db.GormDB = gdb
	defer func() { db.GormDB = prevGormDB }()

	balanceManager := balance.NewManager(pool, zap.NewNop())
	subscriptionManager := subscription.NewManager(pool, zap.NewNop())
	svc := reseller.NewService(gdb, pool, balanceManager, subscriptionManager, zap.NewNop())

	// 1. Two resellers: the acting one and an unrelated one (for isolation checks).
	resellerID := createResellerTenant(t, pool)
	otherResellerID := createResellerTenant(t, pool)
	var pkgID int64
	var tenantID uuid.UUID
	defer func() {
		cleanupIntegrationFlow(t, pool, resellerID, otherResellerID, tenantID, pkgID)
	}()

	// 2. Admin funds the reseller wallet (CreditReseller).
	creditResult, err := svc.CreditReseller(ctx, resellerID, decimal.NewFromFloat(500), "admin funding", "admin@test.local", "127.0.0.1")
	require.NoError(t, err)
	require.Equal(t, "500", creditResult.NewBalance.String())

	wallet, err := svc.GetWalletBalance(ctx, resellerID)
	require.NoError(t, err)
	require.Equal(t, "500", wallet.String())

	// 3. Reseller creates a customer company (API keys + serial + parent link).
	tenant := &models.Tenant{
		BusinessName: "Customer Co " + uuid.New().String()[:8],
		ContactName:  "Customer Contact",
		Email:        "customer-" + uuid.New().String() + "@test.local",
		Phone:        "771234567",
		CountryCode:  "YE",
	}
	user := &models.User{
		Email:        "customer-user-" + uuid.New().String() + "@test.local",
		PasswordHash: "x",
	}
	require.NoError(t, svc.CreateTenant(ctx, resellerID, tenant, user))
	tenantID = tenant.ID
	require.NotEqual(t, uuid.Nil, tenantID)
	require.NotEmpty(t, tenant.APIKeyAccounting)
	require.NotEmpty(t, tenant.APIKeyPhone)
	require.NotEmpty(t, tenant.APIKeyWeb)
	require.False(t, tenant.IsReseller)
	require.NotNil(t, tenant.ParentID)
	require.Equal(t, resellerID, *tenant.ParentID)
	require.NotEmpty(t, tenant.CompanySerial)

	// 4. Reseller transfers funds to the customer wallet.
	transfer, err := svc.TransferToTenant(ctx, resellerID, tenantID, decimal.NewFromFloat(200), "customer funding", "reseller@test.local", "127.0.0.1")
	require.NoError(t, err)
	require.Equal(t, "300", transfer.FromNewBalance.String())
	require.Equal(t, "200", transfer.ToNewBalance.String())

	tenantBal, err := svc.GetTenantBalance(ctx, resellerID, tenantID)
	require.NoError(t, err)
	require.Equal(t, "200", tenantBal.String())

	// 5. Reseller subscribes the customer (charged from the customer wallet).
	pkgID = createIntegrationPackage(t, pool, tenantID)
	subResult, err := svc.SubscribeTenant(ctx, resellerID, tenantID, pkgID, false)
	require.NoError(t, err)
	require.True(t, subResult.Success)
	require.Equal(t, "150", subResult.NewBalance.String())

	// 6. Detail list reflects balance + subscription status.
	details, err := svc.ListTenantsWithDetails(ctx, resellerID)
	require.NoError(t, err)
	require.Len(t, details, 1)
	require.Equal(t, tenantID, details[0].ID)
	require.Equal(t, "150", details[0].Balance)
	require.Equal(t, "active", details[0].Subscription)

	// 7. Ownership isolation: another reseller cannot read, fund, transfer or subscribe.
	_, err = svc.GetTenant(ctx, otherResellerID, tenantID)
	require.Error(t, err)

	_, err = svc.TransferToTenant(ctx, otherResellerID, tenantID, decimal.NewFromFloat(10), "hijack", "other@test.local", "127.0.0.1")
	require.Error(t, err)

	_, err = svc.SubscribeTenant(ctx, otherResellerID, tenantID, pkgID, false)
	require.Error(t, err)

	_, err = svc.GetTenantBalance(ctx, otherResellerID, tenantID)
	require.Error(t, err)

	// The customer wallet was not touched by the failed attempts.
	bal, err := balanceManager.GetBalance(ctx, tenantID)
	require.NoError(t, err)
	require.Equal(t, "150", bal.String())
}

// TestResellerCreditRejectsNonReseller verifies that admin credit refuses
// non-reseller tenants (isolated from the happy path).
func TestResellerCreditRejectsNonReseller_Integration(t *testing.T) {
	ctx := context.Background()
	pool, gdb := setupResellerTestDB(t)
	defer pool.Close()

	balanceManager := balance.NewManager(pool, zap.NewNop())
	subscriptionManager := subscription.NewManager(pool, zap.NewNop())
	svc := reseller.NewService(gdb, pool, balanceManager, subscriptionManager, zap.NewNop())

	// A regular customer tenant (not a reseller).
	customerID := createTestTenant(t, pool, false, nil)
	defer cleanupTestTenant(t, pool, customerID)

	_, err := svc.CreditReseller(ctx, customerID, decimal.NewFromFloat(100), "should fail", "admin@test.local", "127.0.0.1")
	require.Error(t, err)
}

// TestResellerCreateTenantGeneratesSchema verifies schema provisioning executes
// without error through db.GormDB for a created customer tenant.
func TestResellerCreateTenantGeneratesSchema_Integration(t *testing.T) {
	ctx := context.Background()
	pool, gdb := setupResellerTestDB(t)
	defer pool.Close()

	prevGormDB := db.GormDB
	db.GormDB = gdb
	defer func() { db.GormDB = prevGormDB }()

	svc := reseller.NewService(gdb, pool,
		balance.NewManager(pool, zap.NewNop()),
		subscription.NewManager(pool, zap.NewNop()),
		zap.NewNop())

	resellerID := createResellerTenant(t, pool)
	tenant := &models.Tenant{
		BusinessName: "Schema Co " + uuid.New().String()[:8],
		ContactName:  "Schema Contact",
		Email:        "schema-" + uuid.New().String() + "@test.local",
		Phone:        "771122233",
		CountryCode:  "YE",
	}
	user := &models.User{
		Email:        "schema-user-" + uuid.New().String() + "@test.local",
		PasswordHash: "x",
	}

	var tenantID uuid.UUID
	defer func() {
		cleanupIntegrationFlow(t, pool, resellerID, uuid.Nil, tenantID, 0)
	}()

	require.NoError(t, svc.CreateTenant(ctx, resellerID, tenant, user))
	tenantID = tenant.ID
	require.NotEmpty(t, tenant.APIKeyAccounting)
	require.NotEmpty(t, tenant.CompanySerial)

	// The tenant schema must now exist in the test database.
	var exists bool
	err := pool.QueryRow(ctx,
		"SELECT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = $1)",
		db.GetSchemaName(tenantID)).Scan(&exists)
	require.NoError(t, err)
	require.True(t, exists, "tenant schema should be provisioned")

	// Ownership still enforced for a reseller that did not create the tenant.
	otherResellerID := createResellerTenant(t, pool)
	defer func() {
		// re-invoke cleanup for the second reseller without a tenant schema
		_, _ = pool.Exec(ctx, "DELETE FROM tenants WHERE id = $1", otherResellerID)
	}()
	_, err = svc.GetTenant(ctx, otherResellerID, tenantID)
	require.Error(t, err)
}
