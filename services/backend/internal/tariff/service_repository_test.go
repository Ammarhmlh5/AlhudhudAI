package tariff

import (
	"context"
	"testing"

	"github.com/al-hudhud/backend/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupServiceTestDB creates an in-memory SQLite database for testing
func setupServiceTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// :memory: sqlite is per-connection; pin the pool to a single connection
	// so tables created by AutoMigrate are visible to later queries.
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	// Auto-migrate is not possible for models.Tenant on SQLite (Postgres-only
	// `DEFAULT public.uuid_generate_v4()`), so create a portable schema instead.
	createPortableSchema(t, db)

	return db
}

func TestServiceRepository_Create(t *testing.T) {
	db := setupServiceTestDB(t)
	repo := NewServiceRepository(db)
	ctx := context.Background()
	tenantID := uuid.New()

	t.Run("create valid service", func(t *testing.T) {
		service := &models.Service{
			TenantID:    tenantID,
			Name:        "Test Service",
			Code:        models.ServiceCodeHududGateway,
			Type:        models.ServiceTypeGateway,
			Description: "Test description",
			IsActive:    true,
		}

		err := repo.Create(ctx, service)
		assert.NoError(t, err)
		assert.NotZero(t, service.ID)
	})

	t.Run("create duplicate service code", func(t *testing.T) {
		service := &models.Service{
			TenantID:    tenantID,
			Name:        "Duplicate Service",
			Code:        models.ServiceCodeHududGateway, // Same code as above
			Type:        models.ServiceTypeGateway,
			Description: "Test description",
			IsActive:    true,
		}

		err := repo.Create(ctx, service)
		assert.Error(t, err)
		assert.Equal(t, models.ErrServiceAlreadyExists, err)
	})

	t.Run("create service with empty tenant_id", func(t *testing.T) {
		service := &models.Service{
			TenantID:    uuid.Nil,
			Name:        "Invalid Service",
			Code:        models.ServiceCodeMobileGateway,
			Type:        models.ServiceTypeMobile,
			Description: "Test description",
			IsActive:    true,
		}

		err := repo.Create(ctx, service)
		assert.Error(t, err)
		assert.Equal(t, models.ErrInvalidTenantID, err)
	})

	t.Run("create service with invalid type", func(t *testing.T) {
		service := &models.Service{
			TenantID:    tenantID,
			Name:        "Invalid Type Service",
			Code:        "INVALID_CODE",
			Type:        "invalid_type",
			Description: "Test description",
			IsActive:    true,
		}

		err := repo.Create(ctx, service)
		assert.Error(t, err)
		assert.Equal(t, models.ErrInvalidServiceType, err)
	})
}

func TestServiceRepository_GetByID(t *testing.T) {
	db := setupServiceTestDB(t)
	repo := NewServiceRepository(db)
	ctx := context.Background()
	tenantID := uuid.New()

	// Create a test service
	service := &models.Service{
		TenantID:    tenantID,
		Name:        "Test Service",
		Code:        models.ServiceCodeHududGateway,
		Type:        models.ServiceTypeGateway,
		Description: "Test description",
		IsActive:    true,
	}
	err := repo.Create(ctx, service)
	require.NoError(t, err)

	t.Run("get existing service", func(t *testing.T) {
		retrieved, err := repo.GetByID(ctx, tenantID, service.ID)
		assert.NoError(t, err)
		assert.Equal(t, service.ID, retrieved.ID)
		assert.Equal(t, service.Name, retrieved.Name)
		assert.Equal(t, service.Code, retrieved.Code)
	})

	t.Run("get non-existing service", func(t *testing.T) {
		_, err := repo.GetByID(ctx, tenantID, 99999)
		assert.Error(t, err)
		assert.Equal(t, models.ErrServiceNotFound, err)
	})

	t.Run("get service with wrong tenant_id", func(t *testing.T) {
		wrongTenantID := uuid.New()
		_, err := repo.GetByID(ctx, wrongTenantID, service.ID)
		assert.Error(t, err)
		assert.Equal(t, models.ErrServiceNotFound, err)
	})
}

func TestServiceRepository_GetByCode(t *testing.T) {
	db := setupServiceTestDB(t)
	repo := NewServiceRepository(db)
	ctx := context.Background()
	tenantID := uuid.New()

	// Create a test service
	service := &models.Service{
		TenantID:    tenantID,
		Name:        "Test Service",
		Code:        models.ServiceCodeHududGateway,
		Type:        models.ServiceTypeGateway,
		Description: "Test description",
		IsActive:    true,
	}
	err := repo.Create(ctx, service)
	require.NoError(t, err)

	t.Run("get existing service by code", func(t *testing.T) {
		retrieved, err := repo.GetByCode(ctx, tenantID, models.ServiceCodeHududGateway)
		assert.NoError(t, err)
		assert.Equal(t, service.Code, retrieved.Code)
	})

	t.Run("get non-existing service by code", func(t *testing.T) {
		_, err := repo.GetByCode(ctx, tenantID, "NON_EXISTING_CODE")
		assert.Error(t, err)
		assert.Equal(t, models.ErrServiceNotFound, err)
	})
}

func TestServiceRepository_GetByType(t *testing.T) {
	db := setupServiceTestDB(t)
	repo := NewServiceRepository(db)
	ctx := context.Background()
	tenantID := uuid.New()

	// Create test services
	services := []*models.Service{
		{
			TenantID:    tenantID,
			Name:        "Gateway Service",
			Code:        models.ServiceCodeHududGateway,
			Type:        models.ServiceTypeGateway,
			Description: "Gateway description",
			IsActive:    true,
		},
		{
			TenantID:    tenantID,
			Name:        "Mobile Service",
			Code:        models.ServiceCodeMobileGateway,
			Type:        models.ServiceTypeMobile,
			Description: "Mobile description",
			IsActive:    true,
		},
	}

	for _, service := range services {
		err := repo.Create(ctx, service)
		require.NoError(t, err)
	}

	t.Run("get services by type", func(t *testing.T) {
		gatewayServices, err := repo.GetByType(ctx, tenantID, models.ServiceTypeGateway)
		assert.NoError(t, err)
		assert.Len(t, gatewayServices, 1)
		assert.Equal(t, models.ServiceTypeGateway, gatewayServices[0].Type)

		mobileServices, err := repo.GetByType(ctx, tenantID, models.ServiceTypeMobile)
		assert.NoError(t, err)
		assert.Len(t, mobileServices, 1)
		assert.Equal(t, models.ServiceTypeMobile, mobileServices[0].Type)
	})
}

func TestServiceRepository_GetAll(t *testing.T) {
	db := setupServiceTestDB(t)
	repo := NewServiceRepository(db)
	ctx := context.Background()
	tenantID := uuid.New()

	// Create test services
	err := repo.InsertDefaultServices(ctx, tenantID)
	require.NoError(t, err)

	t.Run("get all services", func(t *testing.T) {
		services, err := repo.GetAll(ctx, tenantID)
		assert.NoError(t, err)
		assert.Len(t, services, 3)
	})
}

func TestServiceRepository_GetActive(t *testing.T) {
	db := setupServiceTestDB(t)
	repo := NewServiceRepository(db)
	ctx := context.Background()
	tenantID := uuid.New()

	// Create test services
	err := repo.InsertDefaultServices(ctx, tenantID)
	require.NoError(t, err)

	// Deactivate one service
	services, err := repo.GetAll(ctx, tenantID)
	require.NoError(t, err)
	err = repo.Deactivate(ctx, tenantID, services[0].ID)
	require.NoError(t, err)

	t.Run("get active services only", func(t *testing.T) {
		activeServices, err := repo.GetActive(ctx, tenantID)
		assert.NoError(t, err)
		assert.Len(t, activeServices, 2)

		for _, service := range activeServices {
			assert.True(t, service.IsActive)
		}
	})
}

func TestServiceRepository_Update(t *testing.T) {
	db := setupServiceTestDB(t)
	repo := NewServiceRepository(db)
	ctx := context.Background()
	tenantID := uuid.New()

	// Create a test service
	service := &models.Service{
		TenantID:    tenantID,
		Name:        "Original Name",
		Code:        models.ServiceCodeHududGateway,
		Type:        models.ServiceTypeGateway,
		Description: "Original description",
		IsActive:    true,
	}
	err := repo.Create(ctx, service)
	require.NoError(t, err)

	t.Run("update service", func(t *testing.T) {
		service.Name = "Updated Name"
		service.Description = "Updated description"

		err := repo.Update(ctx, service)
		assert.NoError(t, err)

		// Verify update
		updated, err := repo.GetByID(ctx, tenantID, service.ID)
		assert.NoError(t, err)
		assert.Equal(t, "Updated Name", updated.Name)
		assert.Equal(t, "Updated description", updated.Description)
	})

	t.Run("update non-existing service", func(t *testing.T) {
		nonExisting := &models.Service{
			ID:          99999,
			TenantID:    tenantID,
			Name:        "Non-existing",
			Code:        models.ServiceCodeMobileGateway,
			Type:        models.ServiceTypeMobile,
			Description: "Test",
			IsActive:    true,
		}

		err := repo.Update(ctx, nonExisting)
		assert.Error(t, err)
		assert.Equal(t, models.ErrServiceNotFound, err)
	})
}

func TestServiceRepository_ActivateDeactivate(t *testing.T) {
	db := setupServiceTestDB(t)
	repo := NewServiceRepository(db)
	ctx := context.Background()
	tenantID := uuid.New()

	// Create a test service
	service := &models.Service{
		TenantID:    tenantID,
		Name:        "Test Service",
		Code:        models.ServiceCodeHududGateway,
		Type:        models.ServiceTypeGateway,
		Description: "Test description",
		IsActive:    true,
	}
	err := repo.Create(ctx, service)
	require.NoError(t, err)

	t.Run("deactivate service", func(t *testing.T) {
		err := repo.Deactivate(ctx, tenantID, service.ID)
		assert.NoError(t, err)

		// Verify deactivation
		retrieved, err := repo.GetByID(ctx, tenantID, service.ID)
		assert.NoError(t, err)
		assert.False(t, retrieved.IsActive)
	})

	t.Run("activate service", func(t *testing.T) {
		err := repo.Activate(ctx, tenantID, service.ID)
		assert.NoError(t, err)

		// Verify activation
		retrieved, err := repo.GetByID(ctx, tenantID, service.ID)
		assert.NoError(t, err)
		assert.True(t, retrieved.IsActive)
	})
}

func TestServiceRepository_InsertDefaultServices(t *testing.T) {
	db := setupServiceTestDB(t)
	repo := NewServiceRepository(db)
	ctx := context.Background()
	tenantID := uuid.New()

	t.Run("insert default services", func(t *testing.T) {
		err := repo.InsertDefaultServices(ctx, tenantID)
		assert.NoError(t, err)

		// Verify all three services were created
		services, err := repo.GetAll(ctx, tenantID)
		assert.NoError(t, err)
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

	t.Run("insert default services idempotent", func(t *testing.T) {
		// Insert again - should not create duplicates
		err := repo.InsertDefaultServices(ctx, tenantID)
		assert.NoError(t, err)

		// Verify still only 3 services
		services, err := repo.GetAll(ctx, tenantID)
		assert.NoError(t, err)
		assert.Len(t, services, 3)
	})
}

func TestServiceRepository_TenantIsolation(t *testing.T) {
	db := setupServiceTestDB(t)
	repo := NewServiceRepository(db)
	ctx := context.Background()

	tenant1ID := uuid.New()
	tenant2ID := uuid.New()

	// Insert default services for both tenants
	err := repo.InsertDefaultServices(ctx, tenant1ID)
	require.NoError(t, err)

	err = repo.InsertDefaultServices(ctx, tenant2ID)
	require.NoError(t, err)

	t.Run("tenant isolation - each tenant has their own services", func(t *testing.T) {
		tenant1Services, err := repo.GetAll(ctx, tenant1ID)
		assert.NoError(t, err)
		assert.Len(t, tenant1Services, 3)

		tenant2Services, err := repo.GetAll(ctx, tenant2ID)
		assert.NoError(t, err)
		assert.Len(t, tenant2Services, 3)

		// Verify tenant IDs
		for _, service := range tenant1Services {
			assert.Equal(t, tenant1ID, service.TenantID)
		}

		for _, service := range tenant2Services {
			assert.Equal(t, tenant2ID, service.TenantID)
		}
	})

	t.Run("tenant isolation - cannot access other tenant's services", func(t *testing.T) {
		tenant1Services, err := repo.GetAll(ctx, tenant1ID)
		require.NoError(t, err)

		// Try to get tenant1's service using tenant2's ID
		_, err = repo.GetByID(ctx, tenant2ID, tenant1Services[0].ID)
		assert.Error(t, err)
		assert.Equal(t, models.ErrServiceNotFound, err)
	})
}
