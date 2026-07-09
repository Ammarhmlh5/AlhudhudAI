package integration

import (
	"context"
	"testing"
	"time"

	"github.com/al-hudhud/backend/internal/balance"
	"github.com/al-hudhud/backend/internal/config"
	"github.com/al-hudhud/backend/internal/database"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestDatabaseFailureRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database failure recovery test")
	}

	ctx := context.Background()
	logger := zap.NewNop()

	cfg := &config.Config{}
	cfg.Database.Host = "localhost"
	cfg.Database.Port = "5432"
	cfg.Database.User = "postgres"
	cfg.Database.Password = "postgres"
	cfg.Database.Name = "sms_gateway_test"

	t.Run("PrimaryDatabaseFailure", func(t *testing.T) {
		primaryDB, err := database.NewDatabase(cfg, logger)
		require.NoError(t, err)
		defer primaryDB.Close()

		tenantID := uuid.New()
		_, _ = primaryDB.Pool.Exec(ctx, "INSERT INTO tenants (id, name, balance, status, created_at, updated_at) VALUES ($1, $2, $3, $4, NOW(), NOW())", tenantID, "Failover Test", 100.0, "ACTIVE")

		balanceManager := balance.NewManager(primaryDB.Pool, logger)

		messageID := uuid.New()
		result, err := balanceManager.DeductBalance(ctx, balance.BalanceOperation{
			TenantID:  tenantID,
			MessageID: &messageID,
			Amount:    decimal.NewFromFloat(1.0),
			Type:      balance.OperationTypeDeduct,
			Reason:    "Test before failure",
		})
		require.NoError(t, err)
		assert.True(t, result.Success)

		primaryDB.Close()
		time.Sleep(100 * time.Millisecond)

		backupDB, err := database.NewDatabase(cfg, logger)
		require.NoError(t, err)
		defer backupDB.Close()

		backupBalanceManager := balance.NewManager(backupDB.Pool, logger)
		messageID2 := uuid.New()
		result2, err := backupBalanceManager.DeductBalance(ctx, balance.BalanceOperation{
			TenantID:  tenantID,
			MessageID: &messageID2,
			Amount:    decimal.NewFromFloat(1.0),
			Type:      balance.OperationTypeDeduct,
			Reason:    "Test on backup",
		})
		require.NoError(t, err)
		assert.True(t, result2.Success)

		currentBalance, _ := backupBalanceManager.GetBalance(ctx, tenantID)
		assert.True(t, currentBalance.Equal(decimal.NewFromFloat(98.0)))
	})
}
