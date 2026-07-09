# Integration Tests - Admin Interface Enhancements

## Overview

This directory contains comprehensive integration tests for the Admin Interface Enhancement system.

## Test Coverage

### 1. Rating System Tests (`rating_system_test.go`)
- End-to-end rating calculation flow
- Rating updates trigger alerts
- Rating history maintenance
- Rating calculation accuracy across different performance levels

### 2. Routing Tests (`routing_test.go`)
- Dedicated server routing
- Health monitoring integration
- No fallback policy enforcement
- Routing metrics tracking

### 3. Distribution Tests (`distribution_test.go`)
- Round Robin distribution fairness
- Load Balancing strategy
- Priority-Based distribution
- Device failure handling
- Concurrent distribution safety
- Distribution metrics tracking

### 4. Signal/Push Tests (`signalpush_test.go`)
- Timing compliance monitoring
- Anomaly detection (Z-Score, Peak Ratio)
- Auto-enforcement system
- Adaptive sending based on server load
- Delivery report processing

## Running Tests

### Run all integration tests:
```bash
go test ./services/backend/internal/integration/... -v
```

### Run specific test file:
```bash
go test ./services/backend/internal/integration/rating_system_test.go -v
go test ./services/backend/internal/integration/routing_test.go -v
go test ./services/backend/internal/integration/distribution_test.go -v
go test ./services/backend/internal/integration/signalpush_test.go -v
```

### Run with coverage:
```bash
go test ./services/backend/internal/integration/... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Test Database Setup

Integration tests require a test database. Set up using:

```bash
# Create test database
createdb admin_test

# Run migrations
migrate -path services/backend/migrations -database "postgresql://localhost/admin_test" up
```

## Environment Variables

```bash
export TEST_DB_URL="postgresql://localhost/admin_test"
export TEST_REDIS_URL="redis://localhost:6379/1"
```

## Test Data Cleanup

Tests automatically clean up test data after execution. If manual cleanup is needed:

```bash
# Drop test database
dropdb admin_test

# Recreate
createdb admin_test
```

## Performance Expectations

- Rating calculation: < 10ms
- Routing decision: < 50ms at p95
- API response time: < 200ms average
- Distribution fairness: ±5% variance for Round Robin

## Notes

- Tests use isolated test database
- Mock external dependencies where appropriate
- Tests are idempotent and can be run multiple times
- Concurrent tests use proper synchronization
