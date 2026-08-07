package integration

import (
	"context"
	"testing"
	"time"

	"github.com/al-hudhud/backend/internal/balance"
	"github.com/al-hudhud/backend/internal/billing"
	"github.com/al-hudhud/backend/internal/config"
	"github.com/al-hudhud/backend/internal/database"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/al-hudhud/backend/internal/rollover"
	"github.com/al-hudhud/backend/internal/subscription"
	"github.com/al-hudhud/backend/internal/tariff"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type TariffSystemTestSuite struct {
	suite.Suite
	gormDB            *gorm.DB
	db                *database.DB
	redisClient       *goredis.Client
	ctx               context.Context
	cancel            context.CancelFunc
	logger            *zap.Logger
	balanceManager    balance.Manager
	subscriptionMgr   subscription.Manager
	rolloverManager   rollover.Manager
	autoRenewal       *billing.AutoRenewalService
	tariffEngine      tariff.Engine
	createdTenants    []uuid.UUID
}

func (suite *TariffSystemTestSuite) SetupSuite() {
	suite.ctx, suite.cancel = context.WithTimeout(context.Background(), 10*time.Minute)
	suite.logger = zap.NewNop()

	cfg := &config.Config{}
	cfg.Database.Host = "localhost"
	cfg.Database.Port = "5432"
	cfg.Database.User = "postgres"
	cfg.Database.Password = testDBPassword()
	cfg.Database.Name = "sms_gateway_test"
	cfg.Database.SSLMode = "disable"
	cfg.Database.MaxOpenConns = 10
	cfg.Database.MaxIdleConns = 5
	cfg.Database.ConnMaxLifetime = time.Hour
	cfg.Database.ConnMaxIdleTime = 30 * time.Minute

	db, err := database.NewDatabase(cfg, suite.logger)
	require.NoError(suite.T(), err)
	suite.db = db

	connStr := "postgres://postgres@localhost:5432/sms_gateway_test?sslmode=disable"
	gormDB, err := gorm.Open(postgres.Open(connStr), &gorm.Config{})
	require.NoError(suite.T(), err)
	suite.gormDB = gormDB

	suite.redisClient = goredis.NewClient(&goredis.Options{
		Addr: "localhost:6379",
		DB:   13, // Dedicated test DB so FlushDB does not collide with other packages
	})

	suite.balanceManager = balance.NewManager(db.Pool, suite.logger)
	suite.subscriptionMgr = subscription.NewManager(db.Pool, suite.logger)
	suite.rolloverManager = rollover.NewManager(db.Pool, suite.logger)
	suite.autoRenewal = billing.NewAutoRenewalService(db.Pool, nil, suite.logger)
	suite.tariffEngine = tariff.NewEngine(suite.gormDB, suite.redisClient, suite.logger)
}

func (suite *TariffSystemTestSuite) TearDownSuite() {
	suite.cancel()
	if suite.db != nil {
		suite.db.Close()
	}
	if suite.redisClient != nil {
		suite.redisClient.Close()
	}
}

func (suite *TariffSystemTestSuite) SetupTest() {
	suite.cleanupTestData()
}

func (suite *TariffSystemTestSuite) cleanupTestData() {
	// Scope cleanup to the tenants this suite created so parallel packages
	// sharing sms_gateway_test are not wiped.
	for _, tenantID := range suite.createdTenants {
		suite.gormDB.Exec("DELETE FROM rollover_messages WHERE tenant_id = ?", tenantID)
		suite.gormDB.Exec("DELETE FROM subscriptions WHERE tenant_id = ?", tenantID)
		suite.gormDB.Exec("DELETE FROM packages WHERE tenant_id = ?", tenantID)
		suite.gormDB.Exec("DELETE FROM balance_history WHERE tenant_id = ?", tenantID)
		suite.gormDB.Exec("DELETE FROM accounting_transactions WHERE tenant_id = ?", tenantID)
		suite.gormDB.Exec("DELETE FROM messages WHERE tenant_id = ?", tenantID)
		suite.gormDB.Exec("DELETE FROM dispatch_channels WHERE tenant_id = ?", tenantID)
		suite.gormDB.Exec("DELETE FROM tenants WHERE id = ?", tenantID)
	}
	suite.redisClient.FlushDB(suite.ctx)
}

func (suite *TariffSystemTestSuite) TestCompleteSubscriptionFlow() {
	t := suite.T()
	tenantID := suite.createTestTenant("test_flow@example.com", 100.0)
	packageID := suite.createTestPackage(tenantID, "Standard", 20.0, 3000, false, 0, 0)
	
	res, err := suite.subscriptionMgr.Subscribe(suite.ctx, subscription.SubscribeRequest{
		TenantID:  tenantID,
		PackageID: packageID,
		AutoRenew: true,
	})
	require.NoError(t, err)
	require.True(t, res.Success)
	subscriptionID := res.SubscriptionID
	
	bal, _ := suite.balanceManager.GetBalance(suite.ctx, tenantID)
	assert.True(t, bal.Equal(decimal.NewFromFloat(80.0)))
	
	for i := 0; i < 100; i++ {
		_, err := suite.subscriptionMgr.DeductMessages(suite.ctx, subscription.DeductMessagesRequest{
			SubscriptionID: subscriptionID,
			MessageCount:   1,
			MessageID:      uuid.New().String(),
		})
		require.NoError(t, err)
	}
	
	usage, err := suite.subscriptionMgr.GetUsage(suite.ctx, tenantID)
	require.NoError(t, err)
	assert.Equal(t, int64(100), usage.MessagesUsed)
	
	sub, _ := suite.subscriptionMgr.GetActiveSubscription(suite.ctx, tenantID)
	tenant := suite.getTenant(tenantID)
	err = suite.autoRenewal.RenewSubscription(suite.ctx, sub.ID, tenantID, packageID, decimal.Zero, decimal.NewFromFloat(tenant.Balance))
	require.NoError(t, err)
}

func (suite *TariffSystemTestSuite) createTestTenant(email string, balanceVal float64) uuid.UUID {
	tenantID := uuid.New()
	userID := uuid.New()
	err := suite.gormDB.Exec(`
		INSERT INTO users (id, email, password_hash, user_type, is_active, is_verified, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, NOW(), NOW())
	`, userID, "int-"+userID.String()+"@test.com", "hash", "TENANT", true, true).Error
	require.NoError(suite.T(), err)
	err = suite.gormDB.Exec(`
		INSERT INTO tenants (id, user_id, business_name, contact_name, email, balance, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, 'ACTIVE', NOW(), NOW())
	`, tenantID, userID, "Test Business", "Test Contact", email, balanceVal).Error
	require.NoError(suite.T(), err)
	suite.createdTenants = append(suite.createdTenants, tenantID)
	return tenantID
}

func (suite *TariffSystemTestSuite) createTestPackage(tenantID uuid.UUID, name string, price float64, messages int, rolloverEnabled bool, rolloverLimit int, maxRolloverMonths int) int64 {
	msg64 := int64(messages)
	limit64 := int64(rolloverLimit)
	pkg := &subscription.Package{
		TenantID:             tenantID,
		ServiceID:            2,
		Name:                 name,
		PriceUSD:             decimal.NewFromFloat(price),
		IncludedMessages:     &msg64,
		BillingCycle:         "monthly",
		RolloverEnabled:      rolloverEnabled,
		RolloverLimit:        &limit64,
		RolloverExpiryMonths: &maxRolloverMonths,
		IsActive:             true,
	}
	err := suite.gormDB.Table("packages").Create(pkg).Error
	require.NoError(suite.T(), err)
	return pkg.ID
}

func (suite *TariffSystemTestSuite) getTenant(tenantID uuid.UUID) *models.Tenant {
	var tenant models.Tenant
	err := suite.gormDB.First(&tenant, "id = ?", tenantID).Error
	require.NoError(suite.T(), err)
	return &tenant
}

func TestTariffSystemSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests")
	}
	suite.Run(t, new(TariffSystemTestSuite))
}
