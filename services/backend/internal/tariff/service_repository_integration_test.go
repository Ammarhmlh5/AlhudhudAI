// +build integration

package tariff

import (
	"context"
	"os"
	"testing"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestServiceRepository_Integration runs integration tests against a real PostgreSQL database
// Run with: go test -tags=integration -v ./internal/tariff/...
func TestServiceRepository_Integration(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration tests. Set RUN_INTEGRATION_TESTS=true to run.")
	}

	// Initialize database connection
	db.InitDB()
	defer func() {
		if db.GormDB != nil {
			sqlDB, _ := db.GormDB.DB()
			sqlDB.Close()
		}
	}()

	repo := NewServiceRepository(db.GormDB)
	ctx := context.Background()
	tenantID := uuid.New()

	// Clean up after tests
	defer func() {
		db.GormDB.Where("tenant_id = ?", tenantID).Delete(&models.Service{})
	}()

	t.Run("create and retrieve service", func(t *testing.T) {
		service := &models.Service{
			TenantID:    tenantID,
			Name:        "Integration Test Service",
			Code:        models.ServiceCodeHududGateway,
			Type:        models.ServiceTypeGateway,
			Description: "Integration test description",
			IsActive:    true,
		}

		err := repo.Create(ctx, service)
		require.NoError(t, err)
		assert.NotZero(t, service.ID)

		// Retrieve and verify
		retrieved, err := repo.GetByID(ctx, tenantID, service.ID)
		require.NoError(t, err)
		assert.Equal(t, service.Name, retrieved.Name)
		assert.Equal(t, service.Code, retrieved.Code)
	})

	t.Run("insert default services", func(t *testing.T) {
		newTenantID := uuid.New()
		defer func() {
			db.GormDB.Where("tenant_id = ?", newTenantID).Delete(&models.Service{})
		}()

		err := repo.InsertDefaultServices(ctx, newTenantID)
		require.NoError(t, err)

		services, err := repo.GetAll(ctx, newTenantID)
		require.NoError(t, err)
		assert.Len(t, services, 3)

		// Verify service codes
		codes := make(map[models.ServiceCode]bool)
		for _, service := range services {
			codes[service.Code] = true
		}

		assert.True(t, codes[models.ServiceCodeHududGateway])
		assert.True(t, codes[models.ServiceCodeMobileGateway])
		assert.True(t, codes[models.ServiceCodeDedicatedGateway])
	})
}
