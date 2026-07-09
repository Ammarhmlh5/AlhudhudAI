package tariff

import (
	"context"
	"testing"
	"time"

	"github.com/al-hudhud/backend/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Auto-migrate tables
	err = db.AutoMigrate(&models.Tenant{}, &models.Service{}, &models.Tariff{})
	require.NoError(t, err)

	return db
}

func createTestTenant(t *testing.T, db *gorm.DB) *models.Tenant {
	tenant := &models.Tenant{
		ID:           uuid.New(),
		BusinessName: "Test Tenant",
		Balance:      1000.0,
	}

	err := db.Create(tenant).Error
	require.NoError(t, err)

	return tenant
}

func createTestService(t *testing.T, db *gorm.DB, tenantID uuid.UUID) *models.Service {
	service := &models.Service{
		TenantID:    tenantID,
		Name:        "خدمة بوابة الهدهد",
		Code:        models.ServiceCodeHududGateway,
		Type:        models.ServiceTypeGateway,
		Description: "Test service",
		IsActive:    true,
	}

	err := db.Create(service).Error
	require.NoError(t, err)

	return service
}

func TestRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	tenant := createTestTenant(t, db)
	service := createTestService(t, db, tenant.ID)

	ctx := context.Background()

	t.Run("successful creation", func(t *testing.T) {
		tariff := &models.Tariff{
			TenantID:        tenant.ID,
			ServiceID:       service.ID,
			CountryCode:     "YE",
			NetworkOperator: "Yemen Mobile",
			MessageType:     "standard",
			CostPerMessage:  9.0000,
			Currency:        "YER",
			EffectiveFrom:   time.Now(),
			IsActive:        true,
		}

		err := repo.Create(ctx, tariff)
		assert.NoError(t, err)
		assert.NotZero(t, tariff.ID)
	})

	t.Run("validation error - invalid tenant ID", func(t *testing.T) {
		tariff := &models.Tariff{
			TenantID:        uuid.Nil,
			ServiceID:       service.ID,
			CountryCode:     "YE",
			NetworkOperator: "Yemen Mobile",
			CostPerMessage:  9.0000,
			Currency:        "YER",
			EffectiveFrom:   time.Now(),
		}

		err := repo.Create(ctx, tariff)
		assert.Error(t, err)
	})

	t.Run("validation error - negative cost", func(t *testing.T) {
		tariff := &models.Tariff{
			TenantID:        tenant.ID,
			ServiceID:       service.ID,
			CountryCode:     "YE",
			NetworkOperator: "Yemen Mobile",
			CostPerMessage:  -1.0,
			Currency:        "YER",
			EffectiveFrom:   time.Now(),
		}

		err := repo.Create(ctx, tariff)
		assert.Error(t, err)
	})
}

func TestRepository_GetActiveTariff(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	tenant := createTestTenant(t, db)
	service := createTestService(t, db, tenant.ID)

	ctx := context.Background()
	now := time.Now()

	// Create test tariffs
	yemenMobileTariff := &models.Tariff{
		TenantID:        tenant.ID,
		ServiceID:       service.ID,
		CountryCode:     "YE",
		NetworkOperator: "Yemen Mobile",
		MessageType:     "standard",
		CostPerMessage:  9.0000,
		Currency:        "YER",
		EffectiveFrom:   now.AddDate(0, 0, -1),
		IsActive:        true,
	}
	err := repo.Create(ctx, yemenMobileTariff)
	require.NoError(t, err)

	otherTariff := &models.Tariff{
		TenantID:        tenant.ID,
		ServiceID:       service.ID,
		CountryCode:     "YE",
		NetworkOperator: "Other",
		MessageType:     "standard",
		CostPerMessage:  13.0000,
		Currency:        "YER",
		EffectiveFrom:   now.AddDate(0, 0, -1),
		IsActive:        true,
	}
	err = repo.Create(ctx, otherTariff)
	require.NoError(t, err)

	t.Run("get specific operator tariff", func(t *testing.T) {
		tariff, err := repo.GetActiveTariff(
			ctx,
			tenant.ID,
			service.ID,
			"YE",
			"Yemen Mobile",
			"standard",
			now,
		)

		assert.NoError(t, err)
		assert.NotNil(t, tariff)
		assert.Equal(t, "Yemen Mobile", tariff.NetworkOperator)
		assert.Equal(t, 9.0000, tariff.CostPerMessage)
	})

	t.Run("fallback to Other operator", func(t *testing.T) {
		tariff, err := repo.GetActiveTariff(
			ctx,
			tenant.ID,
			service.ID,
			"YE",
			"Sabafon",
			"standard",
			now,
		)

		assert.NoError(t, err)
		assert.NotNil(t, tariff)
		assert.Equal(t, "Other", tariff.NetworkOperator)
		assert.Equal(t, 13.0000, tariff.CostPerMessage)
	})

	t.Run("no tariff found for different country", func(t *testing.T) {
		_, err := repo.GetActiveTariff(
			ctx,
			tenant.ID,
			service.ID,
			"SA",
			"STC",
			"standard",
			now,
		)

		assert.Error(t, err)
	})
}

func TestRepository_GetByService(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	tenant := createTestTenant(t, db)
	service := createTestService(t, db, tenant.ID)

	ctx := context.Background()
	now := time.Now()

	// Create multiple tariffs
	tariffs := []*models.Tariff{
		{
			TenantID:        tenant.ID,
			ServiceID:       service.ID,
			CountryCode:     "YE",
			NetworkOperator: "Yemen Mobile",
			MessageType:     "standard",
			CostPerMessage:  9.0000,
			Currency:        "YER",
			EffectiveFrom:   now,
			IsActive:        true,
		},
		{
			TenantID:        tenant.ID,
			ServiceID:       service.ID,
			CountryCode:     "YE",
			NetworkOperator: "Other",
			MessageType:     "standard",
			CostPerMessage:  13.0000,
			Currency:        "YER",
			EffectiveFrom:   now,
			IsActive:        true,
		},
	}

	for _, tariff := range tariffs {
		err := repo.Create(ctx, tariff)
		require.NoError(t, err)
	}

	t.Run("get all tariffs for service", func(t *testing.T) {
		result, err := repo.GetByService(ctx, tenant.ID, service.ID)

		assert.NoError(t, err)
		assert.Len(t, result, 2)
	})
}

func TestRepository_InsertDefaultTariffs(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	tenant := createTestTenant(t, db)
	service := createTestService(t, db, tenant.ID)

	ctx := context.Background()

	t.Run("insert default tariffs", func(t *testing.T) {
		err := repo.InsertDefaultTariffs(ctx, tenant.ID)
		assert.NoError(t, err)

		// Verify tariffs were created
		tariffs, err := repo.GetByService(ctx, tenant.ID, service.ID)
		assert.NoError(t, err)
		assert.Len(t, tariffs, 2)

		// Verify Yemen Mobile tariff
		yemenMobile := findTariffByOperator(tariffs, "Yemen Mobile")
		assert.NotNil(t, yemenMobile)
		assert.Equal(t, 9.0000, yemenMobile.CostPerMessage)
		assert.Equal(t, "YER", yemenMobile.Currency)

		// Verify Other tariff
		other := findTariffByOperator(tariffs, "Other")
		assert.NotNil(t, other)
		assert.Equal(t, 13.0000, other.CostPerMessage)
		assert.Equal(t, "YER", other.Currency)
	})

	t.Run("idempotent - don't duplicate tariffs", func(t *testing.T) {
		// Insert again
		err := repo.InsertDefaultTariffs(ctx, tenant.ID)
		assert.NoError(t, err)

		// Verify still only 2 tariffs
		tariffs, err := repo.GetByService(ctx, tenant.ID, service.ID)
		assert.NoError(t, err)
		assert.Len(t, tariffs, 2)
	})
}

func TestRepository_TenantIsolation(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	ctx := context.Background()
	now := time.Now()

	// Create two tenants
	tenant1 := createTestTenant(t, db)
	tenant2 := createTestTenant(t, db)

	// Create services for both tenants
	service1 := createTestService(t, db, tenant1.ID)
	service2 := createTestService(t, db, tenant2.ID)

	// Create tariff for tenant1
	tariff1 := &models.Tariff{
		TenantID:        tenant1.ID,
		ServiceID:       service1.ID,
		CountryCode:     "YE",
		NetworkOperator: "Yemen Mobile",
		MessageType:     "standard",
		CostPerMessage:  9.0000,
		Currency:        "YER",
		EffectiveFrom:   now,
		IsActive:        true,
	}
	err := repo.Create(ctx, tariff1)
	require.NoError(t, err)

	// Create tariff for tenant2
	tariff2 := &models.Tariff{
		TenantID:        tenant2.ID,
		ServiceID:       service2.ID,
		CountryCode:     "YE",
		NetworkOperator: "Yemen Mobile",
		MessageType:     "standard",
		CostPerMessage:  10.0000,
		Currency:        "YER",
		EffectiveFrom:   now,
		IsActive:        true,
	}
	err = repo.Create(ctx, tariff2)
	require.NoError(t, err)

	t.Run("tenant1 can only see their tariffs", func(t *testing.T) {
		tariffs, err := repo.GetByService(ctx, tenant1.ID, service1.ID)
		assert.NoError(t, err)
		assert.Len(t, tariffs, 1)
		assert.Equal(t, tenant1.ID, tariffs[0].TenantID)
		assert.Equal(t, 9.0000, tariffs[0].CostPerMessage)
	})

	t.Run("tenant2 can only see their tariffs", func(t *testing.T) {
		tariffs, err := repo.GetByService(ctx, tenant2.ID, service2.ID)
		assert.NoError(t, err)
		assert.Len(t, tariffs, 1)
		assert.Equal(t, tenant2.ID, tariffs[0].TenantID)
		assert.Equal(t, 10.0000, tariffs[0].CostPerMessage)
	})

	t.Run("tenant1 cannot access tenant2's tariff", func(t *testing.T) {
		_, err := repo.GetByID(ctx, tenant1.ID, tariff2.ID)
		assert.Error(t, err)
	})
}

// Helper function to find tariff by operator
func findTariffByOperator(tariffs []*models.Tariff, operator string) *models.Tariff {
	for _, t := range tariffs {
		if t.NetworkOperator == operator {
			return t
		}
	}
	return nil
}
