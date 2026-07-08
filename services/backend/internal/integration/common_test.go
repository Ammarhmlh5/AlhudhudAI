package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/al-hudhud/backend/internal/database"
	"github.com/al-hudhud/backend/internal/gateway"
	goredis "github.com/redis/go-redis/v9"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

// MockRateLimiter implements gateway.RateLimiter for testing
type MockRateLimiter struct{}

func (m *MockRateLimiter) Allow(ctx context.Context, tenantID uuid.UUID) (bool, *gateway.RateLimitInfo, error) {
	return true, &gateway.RateLimitInfo{
		Limit:     100000,
		Remaining: 99999,
		Reset:     time.Now().Add(time.Hour).Unix(),
	}, nil
}

func cleanupTestData(t *testing.T, ctx context.Context, db *database.DB, redisClient *goredis.Client) {
	if redisClient != nil {
		redisClient.FlushDB(ctx)
	}
	
	tables := []string{
		"balance_history", 
		"accounting_transactions", 
		"river_job", 
		"messages", 
		"dispatch_channels", 
		"tenants",
	}
	
	for _, table := range tables {
		_, err := db.Pool.Exec(ctx, fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			t.Logf("Warning: failed to clean %s: %v", table, err)
		}
	}
}

func createTestTenant(t *testing.T, ctx context.Context, db *database.DB, initialBalance decimal.Decimal) uuid.UUID {
	tenantID := uuid.New()
	_, err := db.Pool.Exec(ctx, "INSERT INTO tenants (id, name, email, balance, status, created_at, updated_at) VALUES ($1, $2, $3, $4, 'ACTIVE', NOW(), NOW())", 
		tenantID, "Test Tenant", "test@example.com", initialBalance)
	require.NoError(t, err)
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
