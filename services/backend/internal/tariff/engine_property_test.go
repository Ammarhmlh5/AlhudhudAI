package tariff

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/al-hudhud/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Property 3: استخدام التعرفة النشطة الصحيحة
// **Validates: Requirements 2.5**
//
// لأي رسالة مع شبكة محددة، عند حساب التكلفة، يجب أن يستخدم النظام التعرفة النشطة
// (is_active = true AND effective_from <= CURRENT_DATE AND (effective_to IS NULL OR effective_to >= CURRENT_DATE))
// المطابقة للشبكة
//
// This property test validates:
// 1. Only active tariffs are used for cost calculation
// 2. Tariffs respect effective_from and effective_to dates
// 3. Most specific tariff is selected (network_operator > country)
// 4. Inactive tariffs are never used

func TestProperty_UseActiveTariffOnly(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	// Setup
	ctx := context.Background()
	db, pool, cache := setupTestEnvironment(t)
	defer cleanupTestEnvironment(pool, cache)

	logger, _ := zap.NewDevelopment()
	engine := NewEngine(db, cache, logger)

	// Test parameters
	numIterations := 10

	for iteration := 0; iteration < numIterations; iteration++ {
		t.Run(fmt.Sprintf("Iteration_%d", iteration+1), func(t *testing.T) {
			// Create test tenant
			tenantID := createTestTenantForTariff(t, db)

			// Create test service
			serviceID := createTestServiceForProperty(t, db, tenantID, "TEST_SERVICE", "gateway")

			// Create multiple tariffs with different states
			countryCode := "YE"
			networkOperator := "Yemen Mobile"

			// 1. Active tariff (should be used)
			activeTariff := createTestTariff(t, db, TariffParams{
				TenantID:        tenantID,
				ServiceID:       serviceID,
				CountryCode:     countryCode,
				NetworkOperator: networkOperator,
				CostPerMessage:  decimal.NewFromFloat(9.0),
				IsActive:        true,
				EffectiveFrom:   time.Now().AddDate(0, 0, -7), // 7 days ago
				EffectiveTo:     nil,                          // no end date
			})

			// 2. Inactive tariff (should NOT be used)
			createTestTariff(t, db, TariffParams{
				TenantID:        tenantID,
				ServiceID:       serviceID,
				CountryCode:     countryCode,
				NetworkOperator: networkOperator,
				CostPerMessage:  decimal.NewFromFloat(15.0),
				IsActive:        false, // INACTIVE
				EffectiveFrom:   time.Now().AddDate(0, 0, -7),
				EffectiveTo:     nil,
			})

			// 3. Future tariff (should NOT be used yet)
			createTestTariff(t, db, TariffParams{
				TenantID:        tenantID,
				ServiceID:       serviceID,
				CountryCode:     countryCode,
				NetworkOperator: networkOperator,
				CostPerMessage:  decimal.NewFromFloat(12.0),
				IsActive:        true,
				EffectiveFrom:   time.Now().AddDate(0, 0, 7), // 7 days in future
				EffectiveTo:     nil,
			})

			// 4. Expired tariff (should NOT be used)
			createTestTariff(t, db, TariffParams{
				TenantID:        tenantID,
				ServiceID:       serviceID,
				CountryCode:     countryCode,
				NetworkOperator: networkOperator,
				CostPerMessage:  decimal.NewFromFloat(8.0),
				IsActive:        true,
				EffectiveFrom:   time.Now().AddDate(0, 0, -30),
				EffectiveTo:     ptrTime(time.Now().AddDate(0, 0, -1)), // expired yesterday
			})

			// Calculate cost
			result, err := engine.CalculateCost(ctx, CostCalculationRequest{
				TenantID:        tenantID,
				ServiceID:       serviceID,
				CountryCode:     countryCode,
				NetworkOperator: networkOperator,
				MessageType:     "GSM-7",
				MessageLength:   100,
			})

			require.NoError(t, err, "Cost calculation should succeed")

			// Property: Only the active tariff should be used
			assert.Equal(t, activeTariff.ID, result.TariffID,
				"Property violated: Should use active tariff ID\n"+
					"Expected: %d (active), Got: %d",
				activeTariff.ID, result.TariffID)

			assert.True(t, result.CostPerSegment.Equal(decimal.NewFromFloat(activeTariff.CostPerMessage)),
				"Property violated: Should use active tariff cost\n"+
				"Expected: %.4f, Got: %s",
				activeTariff.CostPerMessage, result.CostPerSegment.String())

			t.Logf("Iteration %d: Active tariff used correctly (ID=%d, Cost=%s)",
				iteration+1, result.TariffID, result.CostPerSegment.String())
		})
	}
}

// Property Test: Tariff Effective Date Boundaries
// Tests that tariffs are correctly selected based on effective_from and effective_to dates
// **Validates: Requirements 2.5**
func TestProperty_TariffEffectiveDateBoundaries(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	// Setup
	ctx := context.Background()
	db, pool, cache := setupTestEnvironment(t)
	defer cleanupTestEnvironment(pool, cache)

	logger, _ := zap.NewDevelopment()
	engine := NewEngine(db, cache, logger)

	numIterations := 10

	for iteration := 0; iteration < numIterations; iteration++ {
		t.Run(fmt.Sprintf("DateBoundaries_Iteration_%d", iteration+1), func(t *testing.T) {
			tenantID := createTestTenantForTariff(t, db)
			serviceID := createTestServiceForProperty(t, db, tenantID, fmt.Sprintf("TEST_SERVICE_%d", iteration), "gateway")
			countryCode := "YE"
			networkOperator := "Yemen Mobile"

			// Create tariff that becomes active today
			todayTariff := createTestTariff(t, db, TariffParams{
				TenantID:        tenantID,
				ServiceID:       serviceID,
				CountryCode:     countryCode,
				NetworkOperator: networkOperator,
				CostPerMessage:  decimal.NewFromFloat(10.0),
				IsActive:        true,
				EffectiveFrom:   time.Now().Truncate(24 * time.Hour), // today at 00:00
				EffectiveTo:     nil,
			})

			// Calculate cost
			result, err := engine.CalculateCost(ctx, CostCalculationRequest{
				TenantID:        tenantID,
				ServiceID:       serviceID,
				CountryCode:     countryCode,
				NetworkOperator: networkOperator,
				MessageType:     "GSM-7",
				MessageLength:   100,
			})

			require.NoError(t, err)

			// Property: Tariff effective today should be used
			assert.Equal(t, todayTariff.ID, result.TariffID,
				"Property violated: Tariff effective today should be used")

			t.Logf("Date boundaries test %d: Tariff effective today used correctly",
				iteration+1)
		})
	}
}

// Property Test: Most Specific Tariff Selection
// Tests that the most specific tariff (with network_operator) is selected over generic ones
// **Validates: Requirements 2.5, 2.6**
func TestProperty_MostSpecificTariffSelection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	// Setup
	ctx := context.Background()
	db, pool, cache := setupTestEnvironment(t)
	defer cleanupTestEnvironment(pool, cache)

	logger, _ := zap.NewDevelopment()
	engine := NewEngine(db, cache, logger)

	numIterations := 10

	for iteration := 0; iteration < numIterations; iteration++ {
		t.Run(fmt.Sprintf("Specificity_Iteration_%d", iteration+1), func(t *testing.T) {
			tenantID := createTestTenantForTariff(t, db)
			serviceID := createTestServiceForProperty(t, db, tenantID, fmt.Sprintf("TEST_SERVICE_SPEC_%d", iteration), "gateway")
			countryCode := "YE"
			networkOperator := "Yemen Mobile"

			// 1. Generic tariff (country only, no operator)
			createTestTariff(t, db, TariffParams{
				TenantID:        tenantID,
				ServiceID:       serviceID,
				CountryCode:     countryCode,
				NetworkOperator: "", // NO OPERATOR (generic)
				CostPerMessage:  decimal.NewFromFloat(15.0),
				IsActive:        true,
				EffectiveFrom:   time.Now().AddDate(0, 0, -7),
				EffectiveTo:     nil,
			})

			// 2. Specific tariff (country + operator)
			specificTariff := createTestTariff(t, db, TariffParams{
				TenantID:        tenantID,
				ServiceID:       serviceID,
				CountryCode:     countryCode,
				NetworkOperator: networkOperator, // WITH OPERATOR (specific)
				CostPerMessage:  decimal.NewFromFloat(9.0),
				IsActive:        true,
				EffectiveFrom:   time.Now().AddDate(0, 0, -7),
				EffectiveTo:     nil,
			})

			// Calculate cost with specific operator
			result, err := engine.CalculateCost(ctx, CostCalculationRequest{
				TenantID:        tenantID,
				ServiceID:       serviceID,
				CountryCode:     countryCode,
				NetworkOperator: networkOperator,
				MessageType:     "GSM-7",
				MessageLength:   100,
			})

			require.NoError(t, err)

			// Property: Most specific tariff (with operator) should be used
			assert.Equal(t, specificTariff.ID, result.TariffID,
				"Property violated: Most specific tariff should be selected\n"+
					"Expected: %d (specific with operator), Got: %d",
				specificTariff.ID, result.TariffID)

			expectedCost := decimal.NewFromFloat(specificTariff.CostPerMessage).String()
			actualCost := result.CostPerSegment.String()
			assert.Equal(t, expectedCost, actualCost,
				"Property violated: Should use specific tariff cost\n"+
					"Expected: %s, Got: %s",
				expectedCost, actualCost)

			t.Logf("Specificity test %d: Most specific tariff selected correctly (Cost=%s)",
				iteration+1, result.CostPerSegment.String())
		})
	}
}

// Property Test: Inactive Tariffs Never Used
// Stress test to ensure inactive tariffs are never selected under any circumstances
// **Validates: Requirements 2.5**
func TestProperty_InactiveTariffsNeverUsed(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	// Setup
	ctx := context.Background()
	db, pool, cache := setupTestEnvironment(t)
	defer cleanupTestEnvironment(pool, cache)

	logger, _ := zap.NewDevelopment()
	engine := NewEngine(db, cache, logger)

	numIterations := 20

	for iteration := 0; iteration < numIterations; iteration++ {
		t.Run(fmt.Sprintf("InactiveNeverUsed_Iteration_%d", iteration+1), func(t *testing.T) {
			tenantID := createTestTenantForTariff(t, db)
			serviceID := createTestServiceForProperty(t, db, tenantID, fmt.Sprintf("TEST_SERVICE_INACTIVE_%d", iteration), "gateway")
			countryCode := "YE"
			networkOperator := fmt.Sprintf("Operator_%d", rand.Intn(5))

			// Create only inactive tariffs
			numInactiveTariffs := 5
			for i := 0; i < numInactiveTariffs; i++ {
				createTestTariff(t, db, TariffParams{
					TenantID:        tenantID,
					ServiceID:       serviceID,
					CountryCode:     countryCode,
					NetworkOperator: networkOperator,
					CostPerMessage:  decimal.NewFromFloat(float64(10 + i)),
					IsActive:        false, // ALL INACTIVE
					EffectiveFrom:   time.Now().AddDate(0, 0, -7),
					EffectiveTo:     nil,
				})
			}

			// Try to calculate cost
			_, err := engine.CalculateCost(ctx, CostCalculationRequest{
				TenantID:        tenantID,
				ServiceID:       serviceID,
				CountryCode:     countryCode,
				NetworkOperator: networkOperator,
				MessageType:     "GSM-7",
				MessageLength:   100,
			})

			// Property: Should fail because no active tariff exists
			assert.Error(t, err,
				"Property violated: Should fail when no active tariff exists")

			assert.Contains(t, err.Error(), "active tariff not found",
				"Property violated: Error should indicate no active tariff found")

			t.Logf("Inactive test %d: Correctly rejected when only inactive tariffs exist",
				iteration+1)
		})
	}
}

// Property Test: Tariff Cache Consistency
// Tests that cached tariffs match database tariffs
// **Validates: Requirements 2.5, 27.9**
func TestProperty_TariffCacheConsistency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	// Setup
	ctx := context.Background()
	db, pool, cache := setupTestEnvironment(t)
	defer cleanupTestEnvironment(pool, cache)

	logger, _ := zap.NewDevelopment()
	engine := NewEngine(db, cache, logger)

	numIterations := 10

	for iteration := 0; iteration < numIterations; iteration++ {
		t.Run(fmt.Sprintf("CacheConsistency_Iteration_%d", iteration+1), func(t *testing.T) {
			tenantID := createTestTenantForTariff(t, db)
			serviceID := createTestServiceForProperty(t, db, tenantID, fmt.Sprintf("TEST_SERVICE_CACHE_%d", iteration), "gateway")
			countryCode := "YE"
			networkOperator := "Yemen Mobile"

			// Create active tariff
			tariff := createTestTariff(t, db, TariffParams{
				TenantID:        tenantID,
				ServiceID:       serviceID,
				CountryCode:     countryCode,
				NetworkOperator: networkOperator,
				CostPerMessage:  decimal.NewFromFloat(9.0),
				IsActive:        true,
				EffectiveFrom:   time.Now().AddDate(0, 0, -7),
				EffectiveTo:     nil,
			})

			// First call - should fetch from database and cache
			result1, err := engine.CalculateCost(ctx, CostCalculationRequest{
				TenantID:        tenantID,
				ServiceID:       serviceID,
				CountryCode:     countryCode,
				NetworkOperator: networkOperator,
				MessageType:     "GSM-7",
				MessageLength:   100,
			})
			require.NoError(t, err)

			// Second call - should fetch from cache
			result2, err := engine.CalculateCost(ctx, CostCalculationRequest{
				TenantID:        tenantID,
				ServiceID:       serviceID,
				CountryCode:     countryCode,
				NetworkOperator: networkOperator,
				MessageType:     "GSM-7",
				MessageLength:   100,
			})
			require.NoError(t, err)

			// Property: Cached result should match database result
			assert.Equal(t, result1.TariffID, result2.TariffID,
				"Property violated: Cached tariff ID should match database tariff ID")

			assert.Equal(t, result1.CostPerSegment.String(), result2.CostPerSegment.String(),
				"Property violated: Cached cost should match database cost")

			assert.Equal(t, tariff.ID, result2.TariffID,
				"Property violated: Cached tariff should be the correct one")

			t.Logf("Cache consistency test %d: Cache matches database correctly",
				iteration+1)
		})
	}
}

// Helper types and functions

type TariffParams struct {
	TenantID        uuid.UUID
	ServiceID       int
	CountryCode     string
	NetworkOperator string
	CostPerMessage  decimal.Decimal
	IsActive        bool
	EffectiveFrom   time.Time
	EffectiveTo     *time.Time
}

func createTestTariff(t *testing.T, db *gorm.DB, params TariffParams) *models.Tariff {
	tariff := &models.Tariff{
		TenantID:        params.TenantID,
		ServiceID:       params.ServiceID,
		CountryCode:     params.CountryCode,
		NetworkOperator: params.NetworkOperator,
		MessageType:     "GSM-7",
		CostPerMessage:  params.CostPerMessage.InexactFloat64(),
		Currency:        "YER",
		EffectiveFrom:   params.EffectiveFrom,
		EffectiveTo:     params.EffectiveTo,
		IsActive:        params.IsActive,
	}

	err := db.Create(tariff).Error
	require.NoError(t, err, "Failed to create test tariff")

	if !params.IsActive {
		err = db.Exec(`UPDATE tariffs SET is_active = false WHERE id = ?`, tariff.ID).Error
		require.NoError(t, err, "Failed to force inactive tariff")
	}

	return tariff
}

func createTestTenantForTariff(t *testing.T, db *gorm.DB) uuid.UUID {
	// Use raw SQL to insert tenant
	tenantID := uuid.New()
	err := db.Exec(`
		INSERT INTO tenants (id, business_name, contact_name, email, balance, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, NOW(), NOW())
	`, tenantID, "Property Test Tenant", "Test Contact", "test@property.com", 10000.0).Error
	
	require.NoError(t, err, "Failed to create test tenant")
	return tenantID
}

func createTestServiceForProperty(t *testing.T, db *gorm.DB, tenantID uuid.UUID, code, serviceType string) int {
	var serviceID int
	err := db.Raw(`
		INSERT INTO services (tenant_id, name, code, type, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, true, NOW(), NOW())
		RETURNING id
	`, tenantID, code, code, serviceType).Scan(&serviceID).Error
	
	require.NoError(t, err, "Failed to create test service")
	return serviceID
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

func setupTestEnvironment(t *testing.T) (*gorm.DB, *pgxpool.Pool, *redis.Client) {
	// Database connection
	connString := "postgres://postgres@localhost:5432/sms_gateway_test?sslmode=disable"
	
	// GORM connection
	gormDB, err := gorm.Open(postgres.Open(connString), &gorm.Config{})
	require.NoError(t, err, "Failed to connect to test database with GORM")

	// pgxpool connection
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, connString)
	require.NoError(t, err, "Failed to connect to test database with pgxpool")

	// Redis connection
	cache := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   3, // Dedicated test DB so FlushDB does not collide with routing (DB 1)
	})

	// Verify Redis connection. Under the full `go test ./...` parallel sweep
	// the WSL2 Redis relay can be saturated enough to blow the default 3s read
	// timeout (observed "i/o timeout" at this ping), so retry with a generous
	// per-attempt budget before giving up.
	var pingErr error
	for attempt := 0; attempt < 5; attempt++ {
		pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		pingErr = cache.Ping(pingCtx).Err()
		cancel()
		if pingErr == nil {
			break
		}
		time.Sleep(300 * time.Millisecond)
	}
	require.NoError(t, pingErr, "Failed to connect to Redis")

	// Clean Redis test database
	cache.FlushDB(ctx)

	return gormDB, pool, cache
}

func cleanupTestEnvironment(pool *pgxpool.Pool, cache *redis.Client) {
	if pool != nil {
		pool.Close()
	}
	if cache != nil {
		cache.Close()
	}
}
