package integration

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/al-hudhud/backend/internal/database"
	"github.com/al-hudhud/backend/internal/gateway"
	goredis "github.com/redis/go-redis/v9"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func testDBPassword() string {
	if p := os.Getenv("PGPASSWORD"); p != "" {
		return p
	}
	return "postgres"
}

// requireRedisPing verifies connectivity to Redis. Under the parallel load of a
// full `go test ./...` sweep the WSL2 Redis relay can be saturated enough to
// blow go-redis' default 3s read timeout (observed "i/o timeout" at these setup
// pings), so it retries with a generous per-attempt budget instead of failing
// the whole package on a transient hiccup.
func requireRedisPing(t *testing.T, client *goredis.Client) {
	t.Helper()
	ctx := context.Background()
	var pingErr error
	for attempt := 0; attempt < 5; attempt++ {
		pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		pingErr = client.Ping(pingCtx).Err()
		cancel()
		if pingErr == nil {
			return
		}
		time.Sleep(300 * time.Millisecond)
	}
	require.NoError(t, pingErr, "Failed to ping Redis")
}

// MockRateLimiter implements gateway.RateLimiter for testing
type MockRateLimiter struct{}

func (m *MockRateLimiter) Allow(ctx context.Context, tenantID uuid.UUID) (bool, *gateway.RateLimitInfo, error) {
	return true, &gateway.RateLimitInfo{
		Limit:     100000,
		Remaining: 99999,
		Reset:     time.Now().Add(time.Hour).Unix(),
	}, nil
}

// createdTenants tracks every tenant this package creates so cleanup can be
// scoped to this package's own data instead of wiping shared tables that other
// packages (running in parallel via go test ./...) rely on.
var (
	tenantMu       sync.Mutex
	createdTenants = make(map[uuid.UUID]struct{})
)

func trackTenant(id uuid.UUID) {
	tenantMu.Lock()
	createdTenants[id] = struct{}{}
	tenantMu.Unlock()
}

func trackedTenantIDs() []uuid.UUID {
	tenantMu.Lock()
	defer tenantMu.Unlock()
	ids := make([]uuid.UUID, 0, len(createdTenants))
	for id := range createdTenants {
		ids = append(ids, id)
	}
	return ids
}

func cleanupTestData(t *testing.T, ctx context.Context, db *database.DB, redisClient *goredis.Client) {
	if redisClient != nil {
		// Contained to this package's dedicated Redis test database (DB 13).
		redisClient.FlushDB(ctx)
	}

	childTables := []string{
		"balance_history",
		"accounting_transactions",
		"river_job",
		"messages",
		"dispatch_channels",
	}

	for _, tenantID := range trackedTenantIDs() {
		for _, table := range childTables {
			_, err := db.Pool.Exec(ctx, fmt.Sprintf("DELETE FROM %s WHERE tenant_id = $1", table), tenantID)
			if err != nil {
				t.Logf("Warning: failed to clean %s: %v", table, err)
			}
		}
		_, err := db.Pool.Exec(ctx, "DELETE FROM tenants WHERE id = $1", tenantID)
		if err != nil {
			t.Logf("Warning: failed to clean tenant: %v", err)
		}
	}
}

func createTestTenant(t *testing.T, ctx context.Context, db *database.DB, initialBalance decimal.Decimal) uuid.UUID {
	t.Helper()
	tenantID := uuid.New()
	userID := uuid.New()
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, user_type, is_active, is_verified, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
	`, userID, "int-"+userID.String()+"@test.com", "hash", "ADMIN", true, true)
	require.NoError(t, err)
	_, err = db.Pool.Exec(ctx, "INSERT INTO tenants (id, user_id, business_name, contact_name, email, balance, status, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, 'ACTIVE', NOW(), NOW())",
		tenantID, userID, "Test Tenant", "Test Contact", "test@example.com", initialBalance)
	require.NoError(t, err)
	trackTenant(tenantID)
	return tenantID
}

func createTestChannel(t *testing.T, ctx context.Context, db *database.DB, tenantID uuid.UUID) uuid.UUID {
	channelID := uuid.New()
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO dispatch_channels (id, tenant_id, type, channel_key, status, priority, current_load, config, created_at, updated_at) 
		VALUES ($1, $2, 'PHONE', $3, 'ONLINE', 1, 0, '{}', NOW(), NOW())`, 
		channelID, tenantID, uuid.New().String())
	require.NoError(t, err)
	return channelID
}
