# Tariff and Pricing System

This package implements the tariff and pricing system for the SMS Gateway platform, supporting three distinct service models.

## Service Models

### 1. Hudud Gateway Service (Pay-as-you-go)
- **Code**: `HUDUD_GATEWAY`
- **Type**: `gateway`
- **Model**: Pay per message sent
- **Pricing**: Based on network operator tariffs
  - Yemen Mobile: 9 YER per message
  - Other networks: 13 YER per message

### 2. Mobile Gateway Service (Monthly Subscription)
- **Code**: `MOBILE_GATEWAY`
- **Type**: `mobile`
- **Model**: Monthly subscription with included messages
- **Packages**:
  - Basic: $10/month - 1,000 messages
  - Standard: $20/month - 3,000 messages
  - Professional: $40/month - 8,000 messages
  - Business: $50/month - 15,000 messages
  - Enterprise: $100/month - Unlimited messages
- **Features**:
  - Rollover unused messages to next month
  - Overage charges for messages beyond included amount

### 3. Dedicated Gateway Service (Gateway-as-a-Service)
- **Code**: `DEDICATED_GATEWAY`
- **Type**: `dedicated`
- **Model**: Enterprise dedicated gateway infrastructure
- **Pricing**: 
  - Monthly/yearly subscription fees
  - Server rental fees
  - Note: Enterprise pays telecom provider directly for message costs
- **Features**:
  - Dedicated gateway infrastructure
  - Custom resource quotas
  - Direct telecom provider integration
  - Balance monitoring and alerts

## Database Schema

### Services Table

```sql
CREATE TABLE services (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    code VARCHAR(50) NOT NULL,
    type VARCHAR(50) NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Row Level Security (RLS)

The services table implements Row Level Security to ensure complete tenant isolation:

```sql
ALTER TABLE services ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_select_policy ON services
    FOR SELECT
    USING (tenant_id = current_setting('app.current_tenant')::uuid);
```

All policies (SELECT, INSERT, UPDATE, DELETE) enforce tenant isolation.

### Indexes

All indexes start with `tenant_id` for optimal performance:

```sql
CREATE INDEX idx_services_tenant_id ON services(tenant_id);
CREATE INDEX idx_services_tenant_type ON services(tenant_id, type);
CREATE INDEX idx_services_tenant_active ON services(tenant_id, is_active);
CREATE INDEX idx_services_tenant_code ON services(tenant_id, code);
CREATE UNIQUE INDEX idx_services_tenant_code_unique ON services(tenant_id, code);
```

## Usage

### Creating a Service Repository

```go
import (
    "github.com/al-hudhud/backend/internal/tariff"
    "github.com/al-hudhud/backend/internal/db"
)

repo := tariff.NewServiceRepository(db.GormDB)
```

### Inserting Default Services for a Tenant

```go
tenantID := uuid.New()
err := repo.InsertDefaultServices(ctx, tenantID)
if err != nil {
    log.Fatal(err)
}
```

### Retrieving Services

```go
// Get all services for a tenant
services, err := repo.GetAll(ctx, tenantID)

// Get active services only
activeServices, err := repo.GetActive(ctx, tenantID)

// Get service by ID
service, err := repo.GetByID(ctx, tenantID, serviceID)

// Get service by code
service, err := repo.GetByCode(ctx, tenantID, models.ServiceCodeHududGateway)

// Get services by type
gatewayServices, err := repo.GetByType(ctx, tenantID, models.ServiceTypeGateway)
```

### Updating a Service

```go
service.Name = "Updated Name"
service.Description = "Updated description"
err := repo.Update(ctx, service)
```

### Activating/Deactivating a Service

```go
// Deactivate
err := repo.Deactivate(ctx, tenantID, serviceID)

// Activate
err := repo.Activate(ctx, tenantID, serviceID)
```

## Running Migrations

### Apply Migrations

```bash
# From project root
./run-migrations.sh
```

Or manually:

```bash
cd services/backend/migrations
psql -h localhost -U hudhud_admin -d al_hudhud -f 008_tariff_services_table.sql
psql -h localhost -U hudhud_admin -d al_hudhud -f 009_insert_default_services.sql
```

### Insert Default Services for Existing Tenants

The migration `009_insert_default_services.sql` automatically inserts default services for all existing tenants.

To manually insert for a specific tenant:

```sql
SELECT insert_default_services_for_tenant('<tenant-uuid>');
```

## Testing

### Unit Tests

```bash
cd services/backend
go test -v ./internal/tariff/...
```

Note: Unit tests use SQLite in-memory database and require CGO to be enabled.

### Integration Tests

```bash
cd services/backend
RUN_INTEGRATION_TESTS=true go test -tags=integration -v ./internal/tariff/...
```

Integration tests require a running PostgreSQL database.

## Tenant Isolation

The services table implements complete tenant isolation through:

1. **tenant_id column**: Every service belongs to exactly one tenant
2. **Row Level Security (RLS)**: PostgreSQL policies enforce tenant isolation at the database level
3. **Composite indexes**: All indexes start with tenant_id for optimal query performance
4. **Repository layer**: All queries include tenant_id in WHERE clauses
5. **Unique constraints**: Service codes are unique per tenant (not globally)

### Testing Tenant Isolation

```go
tenant1ID := uuid.New()
tenant2ID := uuid.New()

// Insert services for both tenants
repo.InsertDefaultServices(ctx, tenant1ID)
repo.InsertDefaultServices(ctx, tenant2ID)

// Each tenant has their own services
tenant1Services, _ := repo.GetAll(ctx, tenant1ID) // Returns 3 services
tenant2Services, _ := repo.GetAll(ctx, tenant2ID) // Returns 3 services

// Cannot access other tenant's services
_, err := repo.GetByID(ctx, tenant2ID, tenant1Services[0].ID)
// Returns: ErrServiceNotFound
```

## Error Handling

The repository returns specific errors for different scenarios:

- `ErrInvalidTenantID`: tenant_id is required
- `ErrInvalidServiceName`: service name is required
- `ErrInvalidServiceCode`: service code is required
- `ErrInvalidServiceType`: invalid service type
- `ErrServiceNotFound`: service not found
- `ErrServiceAlreadyExists`: service with this code already exists for this tenant

## Next Steps

After implementing the services table, the next tasks in the tariff and pricing system are:

1. **Task 1.2**: Create tariffs table with RLS
2. **Task 1.3**: Create packages table with RLS
3. **Task 1.4**: Create subscriptions table with RLS
4. **Task 2.1**: Implement tariff calculation engine
5. **Task 2.2**: Implement package management service
6. **Task 2.3**: Implement subscription management service

## Related Documentation

- [Requirements Document](../../../../.kiro/specs/tariff-pricing-system/requirements.md)
- [Design Document](../../../../.kiro/specs/tariff-pricing-system/design.md)
- [Tasks Document](../../../../.kiro/specs/tariff-pricing-system/tasks.md)
