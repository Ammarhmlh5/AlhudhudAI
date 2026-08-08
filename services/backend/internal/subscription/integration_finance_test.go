package subscription

import (
    "context"
    "testing"
    "time"

    "github.com/al-hudhud/backend/internal/finance"
    "github.com/google/uuid"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/stretchr/testify/require"
    "go.uber.org/zap"
)

// fakeBridge updates tenant_sms_quotas in the main DB to simulate the Finance service.
type fakeBridge struct{
    pool *pgxpool.Pool
    wallet uuid.UUID
}

func (f *fakeBridge) OnTenantRegistered(tenantID uuid.UUID, name string) {}
func (f *fakeBridge) OnResellerRegistered(resellerID uuid.UUID, name string) {}
func (f *fakeBridge) GetWalletAccountID(ctx context.Context, refType string, refID uuid.UUID) (uuid.UUID, error) {
    return f.wallet, nil
}
func (f *fakeBridge) GetOrCreateWalletAccountID(ctx context.Context, refType string, refID uuid.UUID) (uuid.UUID, error) {
    return f.wallet, nil
}
func (f *fakeBridge) OnPackagePurchased(tenantID, walletAccountID uuid.UUID, amount float64, smsCount int64, packageName, customerName string) error {
    if smsCount <= 0 {
        return nil
    }
    now := time.Now()
    firstOfNext := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())
    endOfMonth := firstOfNext.Add(-time.Second)

    _, err := f.pool.Exec(context.Background(), `
        INSERT INTO tenant_sms_quotas (tenant_id, available_sms, quota_expires_at, last_reset_at)
        VALUES ($1, $2, $3, NOW())
        ON CONFLICT (tenant_id) DO UPDATE
          SET available_sms = tenant_sms_quotas.available_sms + $2,
              quota_expires_at = EXCLUDED.quota_expires_at,
              updated_at = NOW()
    `, tenantID, smsCount, endOfMonth)
    return err
}
func (f *fakeBridge) OnSMSSent(tenantID uuid.UUID, count int) (bool, error) { return true, nil }
func (f *fakeBridge) MonthlyReset() {}

func TestSubscribe_Integration_MockedFinance(t *testing.T) {
    // Reuse existing test DB helper; will skip if DB not available.
    pool, cleanup := setupTestDB(t)
    if cleanup != nil {
        defer cleanup()
    }

    // Ensure fast SMS quota table exists in test DB
    _, _ = pool.Exec(context.Background(), `
        CREATE TABLE IF NOT EXISTS tenant_sms_quotas (
            tenant_id uuid PRIMARY KEY,
            available_sms bigint DEFAULT 0,
            quota_expires_at timestamp,
            last_reset_at timestamp,
            total_consumed_this_month bigint DEFAULT 0,
            updated_at timestamp
        )
    `)

    // Prepare tenant
    tenantID := uuid.New()
    _, err := pool.Exec(context.Background(), `
        INSERT INTO tenants (id, business_name, contact_name, email, balance, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
        ON CONFLICT (id) DO UPDATE SET balance = $5
    `, tenantID, "Test Tenant", "Test Contact", "test@example.com", 100.0)
    require.NoError(t, err)

    // Create a package with included messages
    _, err = pool.Exec(context.Background(), "SELECT set_config('app.current_tenant', $1, false)", tenantID.String())
    require.NoError(t, err)

    var packageID int64
    err = pool.QueryRow(context.Background(), `
        INSERT INTO packages (tenant_id, service_id, name, code, price_usd, included_messages, overage_rate, is_unlimited, billing_cycle, rollover_enabled, is_active)
        VALUES ($1, 2, 'Integration Package', $2, $3, 1000, 0.01, false, 'monthly', true, true)
        RETURNING id
    `, tenantID, "INT_TEST", 10.0).Scan(&packageID)
    require.NoError(t, err)

    // Inject fake bridge
    fake := &fakeBridge{pool: pool, wallet: uuid.New()}
    finance.SetBridgeForTests(fake)
    defer finance.SetBridgeForTests(nil)

    logger, _ := zap.NewDevelopment()
    mgr := NewManager(pool, logger)

    // Execute subscribe which should call the injected bridge and credit SMS quota
    res, err := mgr.Subscribe(context.Background(), SubscribeRequest{TenantID: tenantID, PackageID: packageID, AutoRenew: true})
    require.NoError(t, err)
    require.True(t, res.Success)

    // Verify tenant_sms_quotas was updated by fake bridge
    var available int64
    err = pool.QueryRow(context.Background(), `SELECT available_sms FROM tenant_sms_quotas WHERE tenant_id = $1`, tenantID).Scan(&available)
    require.NoError(t, err)
    require.Equal(t, int64(1000), available)

    // Cleanup
    _, _ = pool.Exec(context.Background(), `DELETE FROM accounting_transactions WHERE tenant_id = $1`, tenantID)
    _, _ = pool.Exec(context.Background(), `DELETE FROM subscriptions WHERE tenant_id = $1`, tenantID)
    _, _ = pool.Exec(context.Background(), `DELETE FROM packages WHERE tenant_id = $1`, tenantID)
    _, _ = pool.Exec(context.Background(), `DELETE FROM tenant_sms_quotas WHERE tenant_id = $1`, tenantID)
    _, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id = $1`, tenantID)
}
