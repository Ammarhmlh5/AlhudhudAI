# Developer Guide - Tariff & Pricing System

## Table of Contents

1. [Introduction](#introduction)
2. [System Architecture](#system-architecture)
3. [Multi-Tenancy Isolation](#multi-tenancy-isolation)
4. [Adding New Tariffs](#adding-new-tariffs)
5. [Adding New Packages](#adding-new-packages)
6. [Working with Dedicated Gateways](#working-with-dedicated-gateways)
7. [Development Workflow](#development-workflow)
8. [Testing Guidelines](#testing-guidelines)
9. [Performance Considerations](#performance-considerations)
10. [Security Best Practices](#security-best-practices)

---

## Introduction

This guide provides comprehensive information for developers working on the Tariff & Pricing System. The system supports three service models:

1. **Hudud Gateway Service**: Pay-as-you-go model
2. **Mobile Gateway Service**: Monthly subscription with rollover
3. **Dedicated Gateway Service**: Gateway-as-a-Service for enterprises

### Key Design Principles

- **Multi-Tenancy Isolation**: Complete data and resource isolation between tenants
- **Financial Accuracy**: Atomic operations, idempotency, and audit trails
- **Scalability**: Horizontal and vertical scaling support
- **Security**: Encryption, RLS policies, and access control

### Technology Stack

- **Backend**: Go 1.21+
- **Database**: PostgreSQL 15+ with Row Level Security (RLS)
- **Cache**: Redis 7+ with tenant-specific namespacing
- **Queue**: River Queue for background jobs
- **Monitoring**: Prometheus + Grafana
- **Frontend**: React + TypeScript

---

## System Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Load Balancer (AWS ALB)                   │
└─────────────────────────────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
┌───────▼────────┐   ┌───────▼────────┐   ┌───────▼────────┐
│  App Server 1  │   │  App Server 2  │   │  App Server 3  │
│                │   │                │   │                │
│  ┌──────────┐  │   │  ┌──────────┐  │   │  ┌──────────┐  │
│  │ Gateway  │  │   │  │ Gateway  │  │   │  │ Gateway  │  │
│  │ Handler  │  │   │  │ Handler  │  │   │  │ Handler  │  │
│  └──────────┘  │   │  └──────────┘  │   │  └──────────┘  │
│  ┌──────────┐  │   │  ┌──────────┐  │   │  ┌──────────┐  │
│  │ Tariff   │  │   │  │ Tariff   │  │   │  │ Tariff   │  │
│  │ Engine   │  │   │  │ Engine   │  │   │  │ Engine   │  │
│  └──────────┘  │   │  └──────────┘  │   │  └──────────┘  │
│  ┌──────────┐  │   │  ┌──────────┐  │   │  ┌──────────┐  │
│  │ Balance  │  │   │  │ Balance  │  │   │  │ Balance  │  │
│  │ Manager  │  │   │  │ Manager  │  │   │  │ Manager  │  │
│  └──────────┘  │   │  └──────────┘  │   │  └──────────┘  │
└────────┬───────┘   └────────┬───────┘   └────────┬───────┘
         │                    │                    │
         └────────────────────┼────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
┌───────▼────────┐   ┌───────▼────────┐   ┌───────▼────────┐
│   PostgreSQL   │   │  Redis Cluster │   │  River Queue   │
│   (RDS Multi-  │   │  (ElastiCache) │   │  (Background   │
│   AZ + Read    │   │                │   │   Workers)     │
│   Replicas)    │   │                │   │                │
└────────────────┘   └────────────────┘   └────────────────┘
```

### Core Components

#### 1. Tariff Engine (`internal/tariff/`)

Handles dynamic tariff calculation and message segmentation.

**Key Files:**
- `engine.go`: Core tariff calculation logic
- `repository.go`: Database operations for tariffs
- `cache.go`: Redis caching for tariff lookups

**Responsibilities:**
- Calculate message cost based on network and country
- Handle message segmentation (70, 65, 62 characters)
- Cache tariffs for performance (TTL: 1 hour)

#### 2. Balance Manager (`internal/balance/`)

Manages atomic balance operations with full audit trail.

**Key Files:**
- `manager.go`: Atomic balance deduction and refund
- `history_logger.go`: Audit trail logging
- `notification.go`: Low balance alerts

**Responsibilities:**
- Atomic balance deduction with SERIALIZABLE isolation
- Idempotent refund operations
- Complete audit trail in `balance_history` table
- Low balance notifications

#### 3. Subscription Manager (`internal/subscription/`)

Handles monthly packages, rollover, and overage.

**Key Files:**
- `manager.go`: Subscription lifecycle management
- `types.go`: Subscription data structures

**Responsibilities:**
- Package subscription with atomic payment
- Message deduction from subscription quota
- Overage calculation and billing
- Usage reporting

#### 4. Rollover Manager (`internal/rollover/`)

Manages conditional message rollover based on payment success.

**Key Files:**
- `manager.go`: Rollover calculation and activation

**Responsibilities:**
- Calculate remaining messages at month end
- Save messages with 'pending' status
- Activate rollover on successful payment
- Forfeit rollover on payment failure
- FIFO usage of rolled-over messages

#### 5. Gateway Manager (`internal/gateway/`)

Manages dedicated gateways for enterprise clients.

**Key Files:**
- `manager.go`: Gateway lifecycle management
- `user_manager.go`: Gateway user management
- `api_key_manager.go`: API key management

**Responsibilities:**
- Create and configure dedicated gateways
- Manage resource quotas (CPU, memory, bandwidth)
- Monitor telecom balance (read-only)
- Bill subscription and server rental fees

#### 6. Resource Limiter (`internal/resources/`)

Enforces resource quotas per dedicated gateway.

**Key Files:**
- `limiter.go`: Resource quota checking
- `tenant_pool.go`: Tenant-specific worker pools

**Responsibilities:**
- Check CPU, memory, bandwidth quotas
- Rate limiting per gateway
- Worker pool isolation
- Enforcement policies (soft/hard limits)

---

## Multi-Tenancy Isolation

### Overview

Complete isolation between tenants is critical for security and data privacy. The system implements isolation at multiple layers:

1. **Database Layer**: Row Level Security (RLS)
2. **Cache Layer**: Tenant-prefixed keys
3. **Application Layer**: Tenant context validation
4. **Resource Layer**: Quota enforcement

### Database Isolation with RLS

#### Adding tenant_id to Tables

Every table must include `tenant_id`:

```sql
CREATE TABLE tariffs (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    service_id INTEGER NOT NULL REFERENCES services(id),
    -- other fields...
);
```

#### Enabling Row Level Security

```sql
-- Enable RLS on the table
ALTER TABLE tariffs ENABLE ROW LEVEL SECURITY;

-- Create policy for SELECT
CREATE POLICY tenant_isolation_policy ON tariffs
    USING (tenant_id = current_setting('app.current_tenant')::uuid);

-- Create policy for INSERT
CREATE POLICY tenant_isolation_insert_policy ON tariffs
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant')::uuid);

-- Create policy for UPDATE
CREATE POLICY tenant_isolation_update_policy ON tariffs
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant')::uuid);

-- Create policy for DELETE
CREATE POLICY tenant_isolation_delete_policy ON tariffs
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant')::uuid);
```

#### Creating Composite Indexes

All indexes must start with `tenant_id`:

```sql
CREATE INDEX idx_tariffs_tenant_service ON tariffs(tenant_id, service_id);
CREATE INDEX idx_tariffs_tenant_country ON tariffs(tenant_id, country_code);
CREATE INDEX idx_tariffs_tenant_active ON tariffs(tenant_id, is_active);
```

### Application-Level Isolation

#### Middleware Setup

```go
// internal/middleware/tenant_isolation.go

func TenantIsolationMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Extract tenant_id from JWT or API Key
        tenantID, err := extractTenantID(c)
        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid tenant"})
            c.Abort()
            return
        }
        
        // Set tenant_id in context
        c.Set("tenant_id", tenantID)
        
        // Set tenant_id in database session
        db := getDB(c)
        db.Exec("SET app.current_tenant = ?", tenantID.String())
        
        c.Next()
    }
}
```

#### Tenant-Aware Repositories

```go
// internal/tariff/repository.go

type TenantTariffRepository struct {
    db       *gorm.DB
    tenantID uuid.UUID
}

func NewTenantTariffRepository(db *gorm.DB, tenantID uuid.UUID) *TenantTariffRepository {
    return &TenantTariffRepository{
        db:       db,
        tenantID: tenantID,
    }
}

func (r *TenantTariffRepository) GetActiveTariff(serviceID int, countryCode, operator string) (*models.Tariff, error) {
    var tariff models.Tariff
    
    // RLS automatically filters by tenant_id
    err := r.db.Where("service_id = ? AND country_code = ? AND network_operator = ? AND is_active = true", 
        serviceID, countryCode, operator).
        First(&tariff).Error
    
    return &tariff, err
}
```

### Redis Isolation

#### Key Prefixing Strategy

```go
// internal/cache/keys.go

type CacheKeyBuilder struct {
    tenantID uuid.UUID
}

func NewCacheKeyBuilder(tenantID uuid.UUID) *CacheKeyBuilder {
    return &CacheKeyBuilder{tenantID: tenantID}
}

// All keys start with tenant_id
func (ckb *CacheKeyBuilder) TariffKey(serviceID int, countryCode, operator string) string {
    return fmt.Sprintf("tenant:%s:tariff:%d:%s:%s", 
        ckb.tenantID.String(), serviceID, countryCode, operator)
}

func (ckb *CacheKeyBuilder) SubscriptionKey(subscriptionID int) string {
    return fmt.Sprintf("tenant:%s:subscription:%d", 
        ckb.tenantID.String(), subscriptionID)
}
```

#### Tenant Redis Client

```go
// internal/cache/redis.go

type TenantRedisClient struct {
    client     *redis.Client
    tenantID   uuid.UUID
    keyBuilder *CacheKeyBuilder
}

func NewTenantRedisClient(client *redis.Client, tenantID uuid.UUID) *TenantRedisClient {
    return &TenantRedisClient{
        client:     client,
        tenantID:   tenantID,
        keyBuilder: NewCacheKeyBuilder(tenantID),
    }
}
```

---

## Adding New Tariffs

### Step 1: Database Migration

Create a migration to add the new tariff:

```sql
-- migrations/XXX_add_new_tariff.sql

INSERT INTO tariffs (
    tenant_id,
    service_id,
    country_code,
    network_operator,
    cost_per_message,
    currency,
    effective_from,
    is_active
) VALUES (
    'your-tenant-id',
    1, -- service_id for Hudud Gateway
    'SA', -- Saudi Arabia
    'STC',
    0.15, -- 15 cents per message
    'SAR',
    CURRENT_DATE,
    true
);
```

### Step 2: API Endpoint

Use the existing API to create tariffs:

```bash
POST /api/admin/tariffs
Authorization: Bearer <admin-jwt-token>
Content-Type: application/json

{
  "service_id": 1,
  "country_code": "SA",
  "network_operator": "STC",
  "cost_per_message": 0.15,
  "currency": "SAR",
  "effective_from": "2026-03-01",
  "is_active": true
}
```

### Step 3: Tariff Engine Integration

The tariff engine automatically picks up new tariffs:

```go
// internal/tariff/engine.go

func (te *TariffEngine) CalculateCost(ctx context.Context, serviceID int, countryCode, operator string, segments int) (float64, error) {
    // Check cache first
    tariff, err := te.cache.GetTariff(ctx, serviceID, countryCode, operator)
    if err == nil {
        return tariff.CostPerMessage * float64(segments), nil
    }
    
    // Fetch from database (RLS automatically filters by tenant)
    tariff, err = te.repo.GetActiveTariff(serviceID, countryCode, operator)
    if err != nil {
        return 0, fmt.Errorf("tariff not found: %w", err)
    }
    
    // Cache for 1 hour
    te.cache.SetTariff(ctx, serviceID, countryCode, operator, tariff, time.Hour)
    
    return tariff.CostPerMessage * float64(segments), nil
}
```

### Step 4: Cache Invalidation

When updating tariffs, invalidate the cache:

```go
// internal/tariff/repository.go

func (r *TenantTariffRepository) UpdateTariff(tariff *models.Tariff) error {
    err := r.db.Save(tariff).Error
    if err != nil {
        return err
    }
    
    // Invalidate cache
    key := r.keyBuilder.TariffKey(tariff.ServiceID, tariff.CountryCode, tariff.NetworkOperator)
    r.cache.Del(context.Background(), key)
    
    return nil
}
```

### Step 5: Testing

Create tests for the new tariff:

```go
// internal/tariff/engine_test.go

func TestCalculateCost_NewTariff(t *testing.T) {
    // Setup
    engine := setupTariffEngine(t)
    
    // Test
    cost, err := engine.CalculateCost(context.Background(), 1, "SA", "STC", 1)
    
    // Assert
    assert.NoError(t, err)
    assert.Equal(t, 0.15, cost)
}
```

---

## Adding New Packages

### Step 1: Database Migration

Create a migration for the new package:

```sql
-- migrations/XXX_add_premium_package.sql

INSERT INTO packages (
    tenant_id,
    service_id,
    name,
    code,
    price,
    currency,
    billing_cycle,
    included_messages,
    rollover_enabled,
    rollover_limit,
    rollover_expiry_months,
    is_active
) VALUES (
    'your-tenant-id',
    2, -- Mobile Gateway Service
    'Premium',
    'PREMIUM',
    75.00,
    'USD',
    'monthly',
    20000,
    true,
    5000, -- Max 5000 messages can rollover
    2, -- Rollover expires after 2 months
    true
);
```

### Step 2: API Endpoint

Use the existing API to create packages:

```bash
POST /api/admin/packages
Authorization: Bearer <admin-jwt-token>
Content-Type: application/json

{
  "service_id": 2,
  "name": "Premium",
  "code": "PREMIUM",
  "price": 75.00,
  "currency": "USD",
  "billing_cycle": "monthly",
  "included_messages": 20000,
  "rollover_enabled": true,
  "rollover_limit": 5000,
  "rollover_expiry_months": 2,
  "is_active": true
}
```

### Step 3: Subscription Manager Integration

The subscription manager handles the new package automatically:

```go
// internal/subscription/manager.go

func (sm *SubscriptionManager) Subscribe(ctx context.Context, tenantID uuid.UUID, packageID int) (*models.Subscription, error) {
    // Fetch package (RLS filters by tenant)
    pkg, err := sm.packageRepo.GetByID(packageID)
    if err != nil {
        return nil, fmt.Errorf("package not found: %w", err)
    }
    
    // Check balance
    tenant, err := sm.tenantRepo.GetByID(tenantID)
    if err != nil {
        return nil, err
    }
    
    if tenant.Balance < pkg.Price {
        return nil, ErrInsufficientBalance
    }
    
    // Atomic transaction
    return sm.db.Transaction(func(tx *gorm.DB) (*models.Subscription, error) {
        // Deduct balance
        err := sm.balanceManager.DeductBalance(ctx, pkg.Price, "subscription", nil)
        if err != nil {
            return nil, err
        }
        
        // Create subscription
        subscription := &models.Subscription{
            TenantID:          tenantID,
            PackageID:         packageID,
            StartDate:         time.Now(),
            EndDate:           time.Now().AddDate(0, 1, 0),
            MessagesIncluded:  pkg.IncludedMessages,
            MessagesRemaining: pkg.IncludedMessages,
            Status:            "active",
        }
        
        err = tx.Create(subscription).Error
        return subscription, err
    })
}
```

### Step 4: Frontend Integration

Update the package list in the frontend:

```typescript
// apps/client-web/src/hooks/usePackages.ts

export function usePackages() {
  return useQuery({
    queryKey: ['packages'],
    queryFn: async () => {
      const response = await fetch('/api/packages');
      return response.json();
    }
  });
}
```

### Step 5: Testing

Create comprehensive tests:

```go
// internal/subscription/manager_test.go

func TestSubscribe_PremiumPackage(t *testing.T) {
    // Setup
    manager := setupSubscriptionManager(t)
    tenantID := createTestTenant(t, 100.00) // $100 balance
    packageID := createTestPackage(t, "Premium", 75.00, 20000)
    
    // Test
    subscription, err := manager.Subscribe(context.Background(), tenantID, packageID)
    
    // Assert
    assert.NoError(t, err)
    assert.Equal(t, "active", subscription.Status)
    assert.Equal(t, 20000, subscription.MessagesRemaining)
    
    // Verify balance deduction
    tenant := getTenant(t, tenantID)
    assert.Equal(t, 25.00, tenant.Balance)
}
```

---

## Working with Dedicated Gateways

### Creating a Dedicated Gateway

```go
// internal/gateway/manager.go

func (gm *GatewayManager) CreateGateway(ctx context.Context, req *CreateGatewayRequest) (*models.DedicatedGateway, error) {
    // Encrypt credentials
    encryptedPassword, err := gm.encryptCredentials(req.TelecomPassword)
    if err != nil {
        return nil, err
    }
    
    encryptedAPIKey, err := gm.encryptCredentials(req.TelecomAPIKey)
    if err != nil {
        return nil, err
    }
    
    gateway := &models.DedicatedGateway{
        TenantID:                req.TenantID,
        Name:                    req.Name,
        Code:                    req.Code,
        TelecomProvider:         req.TelecomProvider,
        TelecomUsername:         req.TelecomUsername,
        TelecomPasswordEncrypted: encryptedPassword,
        TelecomAPIKeyEncrypted:  encryptedAPIKey,
        SubscriptionType:        req.SubscriptionType,
        SubscriptionFee:         req.SubscriptionFee,
        ServerRentalFee:         req.ServerRentalFee,
        Status:                  "active",
    }
    
    err = gm.db.Create(gateway).Error
    if err != nil {
        return nil, err
    }
    
    // Create default resource quotas
    err = gm.createDefaultQuotas(gateway.ID)
    if err != nil {
        return nil, err
    }
    
    return gateway, nil
}
```

### Resource Quota Management

```go
// internal/resources/limiter.go

type ResourceLimiter struct {
    tenantID     uuid.UUID
    gatewayID    int
    cpuQuota     float64  // vCPU cores
    memoryQuota  int64    // bytes
    currentUsage *ResourceUsage
}

func (rl *ResourceLimiter) CheckCPUQuota(ctx context.Context) error {
    usage := rl.getCurrentCPUUsage()
    
    if usage >= rl.cpuQuota {
        return fmt.Errorf("CPU quota exceeded for gateway %d: %.2f/%.2f", 
            rl.gatewayID, usage, rl.cpuQuota)
    }
    
    return nil
}

func (rl *ResourceLimiter) CheckMemoryQuota(ctx context.Context) error {
    usage := rl.getCurrentMemoryUsage()
    
    if usage >= rl.memoryQuota {
        return fmt.Errorf("memory quota exceeded for gateway %d: %d/%d", 
            rl.gatewayID, usage, rl.memoryQuota)
    }
    
    return nil
}
```

### Balance Monitoring

```go
// internal/monitoring/balance_monitor.go

func (bm *BalanceMonitor) CheckTelecomBalance(ctx context.Context, gatewayID int) error {
    gateway, err := bm.gatewayRepo.GetByID(gatewayID)
    if err != nil {
        return err
    }
    
    // Query telecom provider API (read-only)
    balance, err := bm.queryTelecomBalance(gateway)
    if err != nil {
        return err
    }
    
    // Update gateway record
    gateway.TelecomBalance = balance
    gateway.LastBalanceCheck = time.Now()
    err = bm.gatewayRepo.Update(gateway)
    if err != nil {
        return err
    }
    
    // Send alert if below threshold
    if balance < gateway.BalanceAlertThreshold {
        err = bm.alertManager.SendLowBalanceAlert(gateway)
        if err != nil {
            return err
        }
    }
    
    return nil
}
```

### Gateway Billing

```go
// internal/billing/gateway_billing.go

func (gb *GatewayBilling) BillGatewaySubscription(ctx context.Context, gatewayID int) error {
    gateway, err := gb.gatewayRepo.GetByID(gatewayID)
    if err != nil {
        return err
    }
    
    // Calculate total fee (subscription + server rental)
    totalFee := gateway.SubscriptionFee + gateway.ServerRentalFee
    
    // Deduct from tenant balance
    err = gb.balanceManager.DeductBalance(ctx, totalFee, "gateway_subscription", nil)
    if err != nil {
        return err
    }
    
    // Record transaction
    transaction := &models.AccountingTransaction{
        TenantID:        gateway.TenantID,
        TransactionType: "gateway_subscription",
        Amount:          totalFee,
        Currency:        "USD",
        Description:     fmt.Sprintf("Gateway subscription for %s", gateway.Name),
    }
    
    err = gb.transactionRepo.Create(transaction)
    if err != nil {
        return err
    }
    
    // Generate invoice
    err = gb.invoiceGenerator.GenerateInvoice(gateway.TenantID, transaction)
    if err != nil {
        return err
    }
    
    return nil
}
```

---

## Development Workflow

### Setting Up Development Environment

1. **Clone the repository**:
```bash
git clone <repository-url>
cd sms-gateway
```

2. **Install dependencies**:
```bash
# Backend
cd services/backend
go mod download

# Frontend
cd apps/admin-web
npm install

cd ../client-web
npm install
```

3. **Start infrastructure**:
```bash
docker-compose up -d postgres redis
```

4. **Run migrations**:
```bash
cd services/backend
./run-migrations.sh
```

5. **Start development servers**:
```bash
# Backend
cd services/backend
go run main.go

# Admin Web
cd apps/admin-web
npm run dev

# Client Web
cd apps/client-web
npm run dev
```

### Creating a New Feature

1. **Create a feature branch**:
```bash
git checkout -b feature/new-tariff-model
```

2. **Write tests first** (TDD approach):
```go
// internal/tariff/new_model_test.go

func TestNewTariffModel(t *testing.T) {
    // Test implementation
}
```

3. **Implement the feature**:
```go
// internal/tariff/new_model.go

func CalculateNewModel(/* params */) (/* result */, error) {
    // Implementation
}
```

4. **Run tests**:
```bash
go test ./internal/tariff/...
```

5. **Create migration if needed**:
```sql
-- migrations/XXX_add_new_tariff_model.sql

ALTER TABLE tariffs ADD COLUMN new_field VARCHAR(255);
```

6. **Update documentation**:
```markdown
# Update relevant docs
- API_REFERENCE_TARIFF.md
- DEVELOPER_GUIDE_TARIFF.md
```

7. **Commit and push**:
```bash
git add .
git commit -m "feat: add new tariff model"
git push origin feature/new-tariff-model
```

### Code Review Checklist

- [ ] Tests pass (`go test ./...`)
- [ ] Code follows Go conventions
- [ ] RLS policies applied to new tables
- [ ] Indexes include `tenant_id` first
- [ ] Cache keys include tenant prefix
- [ ] Audit trail logging added
- [ ] Error handling is comprehensive
- [ ] Documentation updated
- [ ] No hardcoded credentials
- [ ] Security best practices followed

---

## Testing Guidelines

### Unit Tests

Test individual functions in isolation:

```go
// internal/tariff/engine_test.go

func TestCalculateCost(t *testing.T) {
    tests := []struct {
        name     string
        segments int
        tariff   float64
        want     float64
    }{
        {"single segment", 1, 9.0, 9.0},
        {"two segments", 2, 9.0, 18.0},
        {"three segments", 3, 9.0, 27.0},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := tt.segments * tt.tariff
            assert.Equal(t, tt.want, got)
        })
    }
}
```

### Integration Tests

Test complete workflows:

```go
// internal/integration/tariff_system_test.go

func TestSubscriptionWorkflow(t *testing.T) {
    // Setup
    db := setupTestDB(t)
    defer cleanupTestDB(t, db)
    
    tenantID := createTestTenant(t, db, 100.00)
    packageID := createTestPackage(t, db, "Basic", 10.00, 1000)
    
    // Test subscription
    subscription, err := subscriptionManager.Subscribe(context.Background(), tenantID, packageID)
    assert.NoError(t, err)
    assert.Equal(t, "active", subscription.Status)
    
    // Test message sending
    err = subscriptionManager.DeductMessages(context.Background(), subscription.ID, 1)
    assert.NoError(t, err)
    
    // Verify usage
    usage, err := subscriptionManager.GetUsage(context.Background(), subscription.ID)
    assert.NoError(t, err)
    assert.Equal(t, 1, usage.MessagesUsed)
    assert.Equal(t, 999, usage.MessagesRemaining)
}
```

### Property-Based Tests

Test invariants with random inputs:

```go
// internal/balance/manager_property_test.go

func TestBalanceOperations_Property(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        initialBalance := rapid.Float64Range(0, 10000).Draw(t, "initial_balance")
        deductAmount := rapid.Float64Range(0, initialBalance).Draw(t, "deduct_amount")
        
        // Setup
        manager := setupBalanceManager(t, initialBalance)
        
        // Deduct
        err := manager.DeductBalance(context.Background(), deductAmount, "test", nil)
        assert.NoError(t, err)
        
        // Verify
        balance := manager.GetBalance(context.Background())
        expected := initialBalance - deductAmount
        assert.InDelta(t, expected, balance, 0.01)
    })
}
```

### Load Tests

Test system under high load:

```javascript
// performance-tests/tariff-load-test.js

import http from 'k6/http';
import { check, sleep } from 'k6';

export let options = {
  stages: [
    { duration: '2m', target: 100 },  // Ramp up to 100 users
    { duration: '5m', target: 100 },  // Stay at 100 users
    { duration: '2m', target: 0 },    // Ramp down
  ],
};

export default function () {
  // Test subscription
  let subscribeRes = http.post('http://localhost:8080/api/subscriptions', JSON.stringify({
    package_id: 1
  }), {
    headers: { 'Content-Type': 'application/json', 'Authorization': 'Bearer <token>' },
  });
  
  check(subscribeRes, {
    'subscription created': (r) => r.status === 201,
  });
  
  sleep(1);
}
```

### Security Tests

Test security vulnerabilities:

```go
// internal/security/security_test.go

func TestCrossTenantAccessPrevention(t *testing.T) {
    // Setup two tenants
    tenant1ID := createTestTenant(t, "tenant1")
    tenant2ID := createTestTenant(t, "tenant2")
    
    // Create tariff for tenant1
    tariff1 := createTestTariff(t, tenant1ID)
    
    // Try to access tenant1's tariff as tenant2
    repo := NewTenantTariffRepository(db, tenant2ID)
    _, err := repo.GetByID(tariff1.ID)
    
    // Should fail due to RLS
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "not found")
}

func TestSQLInjectionPrevention(t *testing.T) {
    maliciousInput := "'; DROP TABLE tariffs; --"
    
    _, err := tariffRepo.GetByCountry(maliciousInput)
    
    // Should not execute SQL injection
    assert.Error(t, err)
    
    // Verify table still exists
    var count int64
    db.Model(&models.Tariff{}).Count(&count)
    assert.Greater(t, count, int64(0))
}
```

---

## Performance Considerations

### Database Optimization

#### Use Composite Indexes

```sql
-- Good: Index starts with tenant_id
CREATE INDEX idx_tariffs_tenant_service ON tariffs(tenant_id, service_id);

-- Bad: Index doesn't start with tenant_id
CREATE INDEX idx_tariffs_service ON tariffs(service_id);
```

#### Use Prepared Statements

```go
// Good: Prepared statement
stmt, err := db.Prepare("SELECT * FROM tariffs WHERE tenant_id = ? AND service_id = ?")
defer stmt.Close()

result, err := stmt.Query(tenantID, serviceID)

// Bad: String concatenation (SQL injection risk)
query := fmt.Sprintf("SELECT * FROM tariffs WHERE tenant_id = '%s'", tenantID)
```

#### Optimize Queries

```go
// Good: Select only needed columns
db.Select("id, name, price").Where("tenant_id = ?", tenantID).Find(&packages)

// Bad: Select all columns
db.Where("tenant_id = ?", tenantID).Find(&packages)
```

### Redis Caching Strategy

#### Cache Hot Data

```go
// Cache frequently accessed tariffs
func (te *TariffEngine) GetTariff(ctx context.Context, serviceID int, country, operator string) (*models.Tariff, error) {
    // Check cache first
    key := te.keyBuilder.TariffKey(serviceID, country, operator)
    cached, err := te.cache.Get(ctx, key).Result()
    if err == nil {
        var tariff models.Tariff
        json.Unmarshal([]byte(cached), &tariff)
        return &tariff, nil
    }
    
    // Fetch from database
    tariff, err := te.repo.GetActiveTariff(serviceID, country, operator)
    if err != nil {
        return nil, err
    }
    
    // Cache for 1 hour
    data, _ := json.Marshal(tariff)
    te.cache.Set(ctx, key, data, time.Hour)
    
    return tariff, nil
}
```

#### Set Appropriate TTLs

```go
// Short TTL for frequently changing data
te.cache.Set(ctx, balanceKey, balance, 5*time.Minute)

// Long TTL for rarely changing data
te.cache.Set(ctx, tariffKey, tariff, 1*time.Hour)

// Very long TTL for static data
te.cache.Set(ctx, packageKey, package, 24*time.Hour)
```

### Connection Pooling

```go
// Configure database connection pool
db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
sqlDB, err := db.DB()

// Set max open connections
sqlDB.SetMaxOpenConns(100)

// Set max idle connections
sqlDB.SetMaxIdleConns(10)

// Set connection lifetime
sqlDB.SetConnMaxLifetime(time.Hour)
```

### Rate Limiting

```go
// internal/resources/limiter.go

type TenantRateLimiter struct {
    tenantID uuid.UUID
    limiters map[string]*rate.Limiter
}

func NewTenantRateLimiter(tenantID uuid.UUID, config *RateLimitConfig) *TenantRateLimiter {
    return &TenantRateLimiter{
        tenantID: tenantID,
        limiters: map[string]*rate.Limiter{
            "api":      rate.NewLimiter(rate.Limit(config.APIRatePerMinute), config.APIRatePerMinute),
            "messages": rate.NewLimiter(rate.Limit(config.MessagesPerHour), config.MessagesPerHour),
        },
    }
}

func (trl *TenantRateLimiter) AllowAPI(ctx context.Context) error {
    if !trl.limiters["api"].Allow() {
        return fmt.Errorf("API rate limit exceeded for tenant %s", trl.tenantID)
    }
    return nil
}
```

---

## Security Best Practices

### Encryption

#### Encrypt Sensitive Data

```go
// internal/security/encryption.go

func EncryptCredentials(plaintext string) (string, error) {
    key := []byte(os.Getenv("ENCRYPTION_KEY")) // 32 bytes for AES-256
    
    block, err := aes.NewCipher(key)
    if err != nil {
        return "", err
    }
    
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }
    
    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return "", err
    }
    
    ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
    return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func DecryptCredentials(ciphertext string) (string, error) {
    key := []byte(os.Getenv("ENCRYPTION_KEY"))
    
    data, err := base64.StdEncoding.DecodeString(ciphertext)
    if err != nil {
        return "", err
    }
    
    block, err := aes.NewCipher(key)
    if err != nil {
        return "", err
    }
    
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }
    
    nonceSize := gcm.NonceSize()
    nonce, ciphertext := data[:nonceSize], data[nonceSize:]
    
    plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return "", err
    }
    
    return string(plaintext), nil
}
```

### API Key Hashing

```go
// internal/gateway/api_key_manager.go

func (akm *APIKeyManager) CreateAPIKey(ctx context.Context, gatewayID int, name string) (*models.GatewayAPIKey, error) {
    // Generate random API key
    apiKey := generateRandomKey(32)
    
    // Hash the key using bcrypt
    hashedKey, err := bcrypt.GenerateFromPassword([]byte(apiKey), bcrypt.DefaultCost)
    if err != nil {
        return nil, err
    }
    
    // Store only the hash
    keyRecord := &models.GatewayAPIKey{
        GatewayID: gatewayID,
        Name:      name,
        KeyHash:   string(hashedKey),
        IsActive:  true,
    }
    
    err = akm.db.Create(keyRecord).Error
    if err != nil {
        return nil, err
    }
    
    // Return the plain key only once
    keyRecord.PlainKey = apiKey
    return keyRecord, nil
}

func (akm *APIKeyManager) ValidateAPIKey(ctx context.Context, plainKey string) (*models.GatewayAPIKey, error) {
    // Find all active keys
    var keys []models.GatewayAPIKey
    err := akm.db.Where("is_active = true").Find(&keys).Error
    if err != nil {
        return nil, err
    }
    
    // Check each key
    for _, key := range keys {
        err := bcrypt.CompareHashAndPassword([]byte(key.KeyHash), []byte(plainKey))
        if err == nil {
            return &key, nil
        }
    }
    
    return nil, errors.New("invalid API key")
}
```

### Input Validation

```go
// internal/gateway/validator.go

func ValidateCreateGatewayRequest(req *CreateGatewayRequest) error {
    if req.Name == "" {
        return errors.New("name is required")
    }
    
    if len(req.Name) > 255 {
        return errors.New("name too long")
    }
    
    if req.Code == "" {
        return errors.New("code is required")
    }
    
    // Validate code format (alphanumeric, underscore, hyphen only)
    matched, _ := regexp.MatchString("^[a-zA-Z0-9_-]+$", req.Code)
    if !matched {
        return errors.New("invalid code format")
    }
    
    if req.SubscriptionFee < 0 {
        return errors.New("subscription fee cannot be negative")
    }
    
    return nil
}
```

### Audit Logging

```go
// internal/audit/logger.go

type AuditLogger struct {
    db *gorm.DB
}

func (al *AuditLogger) LogAction(ctx context.Context, action string, resourceType string, resourceID string, details map[string]interface{}) error {
    tenantID := getTenantIDFromContext(ctx)
    userID := getUserIDFromContext(ctx)
    ipAddress := getIPFromContext(ctx)
    
    log := &models.AuditLog{
        TenantID:     tenantID,
        UserID:       userID,
        Action:       action,
        ResourceType: resourceType,
        ResourceID:   resourceID,
        Details:      details,
        IPAddress:    ipAddress,
        Timestamp:    time.Now(),
    }
    
    return al.db.Create(log).Error
}

// Usage example
func (gm *GatewayManager) UpdateGateway(ctx context.Context, gatewayID int, updates map[string]interface{}) error {
    // Update gateway
    err := gm.db.Model(&models.DedicatedGateway{}).Where("id = ?", gatewayID).Updates(updates).Error
    if err != nil {
        return err
    }
    
    // Log the action
    err = gm.auditLogger.LogAction(ctx, "UPDATE_GATEWAY", "dedicated_gateway", fmt.Sprintf("%d", gatewayID), updates)
    if err != nil {
        // Log error but don't fail the operation
        log.Printf("Failed to log audit: %v", err)
    }
    
    return nil
}
```

### Cross-Tenant Access Prevention

```go
// internal/middleware/cross_tenant_prevention.go

func PreventCrossTenantAccess() gin.HandlerFunc {
    return func(c *gin.Context) {
        tenantID := c.GetString("tenant_id")
        
        // Check resource tenant_id in URL params
        if resourceTenantID := c.Param("tenant_id"); resourceTenantID != "" {
            if resourceTenantID != tenantID {
                // Log the attempt
                logCrossTenantAttempt(c, tenantID, resourceTenantID)
                
                c.JSON(http.StatusForbidden, gin.H{
                    "error": "cross-tenant access denied",
                })
                c.Abort()
                return
            }
        }
        
        c.Next()
    }
}
```

---

## Common Patterns

### Atomic Operations

Always use transactions for operations that modify multiple records:

```go
func (sm *SubscriptionManager) Subscribe(ctx context.Context, tenantID uuid.UUID, packageID int) error {
    return sm.db.Transaction(func(tx *gorm.DB) error {
        // 1. Deduct balance
        err := sm.balanceManager.DeductBalance(ctx, amount, "subscription", nil)
        if err != nil {
            return err // Rollback
        }
        
        // 2. Create subscription
        subscription := &models.Subscription{/* ... */}
        err = tx.Create(subscription).Error
        if err != nil {
            return err // Rollback
        }
        
        // 3. Record transaction
        transaction := &models.AccountingTransaction{/* ... */}
        err = tx.Create(transaction).Error
        if err != nil {
            return err // Rollback
        }
        
        return nil // Commit
    })
}
```

### Idempotency

Ensure operations can be safely retried:

```go
func (bm *BalanceManager) RefundBalance(ctx context.Context, transactionID uuid.UUID) error {
    return bm.db.Transaction(func(tx *gorm.DB) error {
        // Check if already refunded
        var transaction models.AccountingTransaction
        err := tx.Where("id = ?", transactionID).First(&transaction).Error
        if err != nil {
            return err
        }
        
        if transaction.Status == "refunded" {
            return nil // Already refunded, idempotent
        }
        
        // Add balance back
        err = tx.Model(&models.Tenant{}).
            Where("id = ?", transaction.TenantID).
            Update("balance", gorm.Expr("balance + ?", transaction.Amount)).Error
        if err != nil {
            return err
        }
        
        // Mark as refunded
        transaction.Status = "refunded"
        err = tx.Save(&transaction).Error
        if err != nil {
            return err
        }
        
        return nil
    })
}
```

### Error Handling

Use structured errors with context:

```go
// internal/errors/types.go

type AppError struct {
    Code    string
    Message string
    Details map[string]interface{}
    Err     error
}

func (e *AppError) Error() string {
    if e.Err != nil {
        return fmt.Sprintf("%s: %v", e.Message, e.Err)
    }
    return e.Message
}

// Usage
func (sm *SubscriptionManager) Subscribe(ctx context.Context, tenantID uuid.UUID, packageID int) error {
    pkg, err := sm.packageRepo.GetByID(packageID)
    if err != nil {
        return &AppError{
            Code:    "PACKAGE_NOT_FOUND",
            Message: "Package not found",
            Details: map[string]interface{}{"package_id": packageID},
            Err:     err,
        }
    }
    
    // ... rest of implementation
}
```

---

## Troubleshooting

### Common Issues

#### Issue: RLS Policy Not Working

**Symptom**: Can see data from other tenants

**Solution**:
1. Check if RLS is enabled: `ALTER TABLE table_name ENABLE ROW LEVEL SECURITY;`
2. Verify policy exists: `\d+ table_name` in psql
3. Ensure `app.current_tenant` is set in session
4. Check policy conditions match your query

#### Issue: Cache Not Invalidating

**Symptom**: Stale data returned from cache

**Solution**:
1. Check TTL is appropriate for data volatility
2. Ensure cache invalidation on updates
3. Verify Redis connection is stable
4. Check key naming consistency

#### Issue: Performance Degradation

**Symptom**: Slow queries under load

**Solution**:
1. Run `EXPLAIN ANALYZE` on slow queries
2. Check if indexes exist and are being used
3. Verify connection pool settings
4. Monitor Redis hit rate
5. Check for N+1 query problems

---

## Additional Resources

- [API Reference](./API_REFERENCE_TARIFF.md)
- [Database Schema](./DATABASE_SCHEMA_TARIFF.md)
- [Admin Guide](./ADMIN_GUIDE_TARIFF.md)
- [User Guide](./USER_GUIDE_TARIFF.md)
- [Enterprise Guide](./ENTERPRISE_GUIDE.md)

---

**Last Updated**: 2026-03-02  
**Version**: 1.0  
**Maintainer**: Development Team
