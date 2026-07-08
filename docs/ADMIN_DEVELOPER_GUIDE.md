# Admin Interface Developer Guide

**Version:** 1.0.0  
**Last Updated:** April 9, 2026

---

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Development Setup](#development-setup)
3. [Backend Development](#backend-development)
4. [Frontend Development](#frontend-development)
5. [Database Schema](#database-schema)
6. [Testing](#testing)
7. [Deployment](#deployment)
8. [Contributing](#contributing)

---

## Architecture Overview

### System Components

```
┌─────────────────────────────────────────────────────────┐
│                     Admin Frontend                       │
│              (React 18+ with TypeScript)                 │
└────────────────────┬────────────────────────────────────┘
                     │ HTTPS/REST
┌────────────────────▼────────────────────────────────────┐
│                   Admin API Layer                        │
│                  (Go HTTP Handlers)                      │
└────────────────────┬────────────────────────────────────┘
                     │
        ┌────────────┼────────────┐
        │            │            │
┌───────▼──────┐ ┌──▼──────┐ ┌──▼──────────┐
│ Rating       │ │ Health  │ │ Distribution│
│ Engine       │ │ Monitor │ │ System      │
└───────┬──────┘ └──┬──────┘ └──┬──────────┘
        │           │            │
        └───────────┼────────────┘
                    │
        ┌───────────▼────────────┐
        │   PostgreSQL Database   │
        │   + Redis Cache         │
        └────────────────────────┘
```

### Technology Stack

**Backend:**
- Language: Go 1.21+
- Framework: net/http (standard library)
- Database: PostgreSQL 15+
- Cache: Redis 7+
- Testing: testing, testify, gopter

**Frontend:**
- Framework: React 18+
- Language: JavaScript (JSX)
- Routing: React Router DOM
- HTTP Client: Axios
- i18n: i18next
- Styling: Tailwind CSS
- Icons: Lucide React

---

## Development Setup

### Prerequisites

- Go 1.21+
- Node.js 18+
- PostgreSQL 15+
- Redis 7+
- Git

### Backend Setup

```bash
# Clone repository
git clone https://github.com/your-org/al-hudhud-gateway.git
cd al-hudhud-gateway

# Install Go dependencies
cd services/backend
go mod download

# Setup environment variables
cp .env.example .env
# Edit .env with your configuration

# Run database migrations
migrate -path migrations -database "postgresql://localhost/admin_dev" up

# Run backend
go run cmd/server/main.go
```

### Frontend Setup

```bash
# Navigate to frontend directory
cd apps/admin-web

# Install dependencies
npm install

# Setup environment variables
cp .env.example .env
# Edit .env with your configuration

# Run development server
npm run dev
```

### Database Setup

```bash
# Create database
createdb admin_dev

# Run migrations
migrate -path services/backend/migrations -database "postgresql://localhost/admin_dev" up

# Seed test data (optional)
go run services/backend/cmd/seed/main.go
```

---

## Backend Development

### Project Structure

```
services/backend/
├── cmd/
│   └── server/
│       └── main.go              # Application entry point
├── internal/
│   ├── admin/
│   │   ├── rating/              # Rating engine
│   │   ├── health/              # Health monitoring
│   │   ├── routing/             # Routing system
│   │   ├── distribution/        # Distribution strategies
│   │   ├── signalpush/          # Signal/Push monitoring
│   │   ├── alerts/              # Alert management
│   │   └── metrics/             # Performance metrics
│   ├── api/                     # HTTP handlers
│   ├── db/                      # Database layer
│   └── middleware/              # HTTP middleware
├── migrations/                  # Database migrations
└── tests/
    ├── integration/             # Integration tests
    └── unit/                    # Unit tests
```

### Adding a New Feature

#### 1. Define Data Models

```go
// internal/admin/feature/types.go
package feature

type Feature struct {
    ID        string    `json:"id"`
    TenantID  string    `json:"tenant_id"`
    Name      string    `json:"name"`
    CreatedAt time.Time `json:"created_at"`
}
```

#### 2. Implement Business Logic

```go
// internal/admin/feature/service.go
package feature

type Service struct {
    db *sql.DB
}

func NewService(db *sql.DB) *Service {
    return &Service{db: db}
}

func (s *Service) GetFeature(ctx context.Context, id string) (*Feature, error) {
    // Implementation
    return &Feature{}, nil
}
```

#### 3. Create API Handler

```go
// internal/api/feature_handler.go
package api

func (h *Handler) GetFeature(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    
    feature, err := h.featureService.GetFeature(r.Context(), id)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    json.NewEncoder(w).Encode(feature)
}
```

#### 4. Register Route

```go
// internal/api/routes.go
r.Get("/api/admin/features/{id}", h.GetFeature)
```

#### 5. Write Tests

```go
// internal/admin/feature/service_test.go
func TestGetFeature(t *testing.T) {
    // Test implementation
}
```

### Database Migrations

#### Creating a Migration

```bash
# Create new migration
migrate create -ext sql -dir services/backend/migrations -seq feature_table
```

#### Migration File Structure

```sql
-- migrations/XXX_feature_table.up.sql
CREATE TABLE features (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_features_tenant ON features(tenant_id);
```

```sql
-- migrations/XXX_feature_table.down.sql
DROP TABLE IF EXISTS features;
```

### Error Handling

```go
// Use custom error types
type AppError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Details map[string]interface{} `json:"details,omitempty"`
}

func (e *AppError) Error() string {
    return e.Message
}

// Return errors consistently
func (s *Service) GetFeature(ctx context.Context, id string) (*Feature, error) {
    if id == "" {
        return nil, &AppError{
            Code:    "INVALID_INPUT",
            Message: "Feature ID is required",
        }
    }
    
    // ... implementation
}
```

### Logging

```go
import "log/slog"

// Use structured logging
slog.Info("Feature retrieved",
    "feature_id", feature.ID,
    "tenant_id", feature.TenantID,
)

slog.Error("Failed to retrieve feature",
    "error", err,
    "feature_id", id,
)
```

---

## Frontend Development

### Project Structure

```
apps/admin-web/
├── public/
├── src/
│   ├── api/
│   │   └── axios.js             # API client configuration
│   ├── components/
│   │   ├── Companies/           # Company components
│   │   ├── Servers/             # Server components
│   │   ├── Alerts/              # Alert components
│   │   └── ...
│   ├── pages/
│   │   ├── Overview.jsx
│   │   ├── Companies/
│   │   ├── Servers/
│   │   └── ...
│   ├── hooks/
│   │   ├── useWebSocket.js
│   │   ├── useAlerts.js
│   │   └── ...
│   ├── contexts/
│   │   └── AuthContext.jsx
│   ├── locales/
│   │   ├── ar.json
│   │   └── en.json
│   ├── App.jsx
│   └── main.jsx
└── package.json
```

### Creating a New Page

#### 1. Create Page Component

```jsx
// src/pages/Feature/FeatureList.jsx
import React, { useState, useEffect } from 'react';
import axios from '../../api/axios';

function FeatureList() {
  const [features, setFeatures] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchFeatures();
  }, []);

  const fetchFeatures = async () => {
    try {
      const response = await axios.get('/api/admin/features');
      setFeatures(response.data);
    } catch (error) {
      console.error('Failed to fetch features', error);
    } finally {
      setLoading(false);
    }
  };

  if (loading) return <div>Loading...</div>;

  return (
    <div className="p-8">
      <h1 className="text-2xl font-bold mb-4">Features</h1>
      {/* Feature list implementation */}
    </div>
  );
}

export default FeatureList;
```

#### 2. Add Route

```jsx
// src/App.jsx
import FeatureList from './pages/Feature/FeatureList';

// In Routes
<Route path="/features" element={<FeatureList />} />
```

#### 3. Add Navigation Item

```jsx
// In MainLayout component
<NavItem id="features" label={t('nav.features')} />
```

#### 4. Add Translations

```json
// src/locales/en.json
{
  "nav": {
    "features": "Features"
  }
}

// src/locales/ar.json
{
  "nav": {
    "features": "الميزات"
  }
}
```

### Creating Custom Hooks

```jsx
// src/hooks/useFeatures.js
import { useState, useEffect } from 'react';
import axios from '../api/axios';

export function useFeatures() {
  const [features, setFeatures] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    fetchFeatures();
  }, []);

  const fetchFeatures = async () => {
    try {
      setLoading(true);
      const response = await axios.get('/api/admin/features');
      setFeatures(response.data);
      setError(null);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const refetch = () => {
    fetchFeatures();
  };

  return { features, loading, error, refetch };
}
```

### Styling Guidelines

```jsx
// Use Tailwind CSS classes
<div className="bg-slate-900 border border-slate-800 p-6 rounded-xl">
  <h2 className="text-xl font-bold text-white mb-4">Title</h2>
  <p className="text-slate-400">Description</p>
</div>

// Responsive design
<div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
  {/* Content */}
</div>

// RTL support
<div className="text-left rtl:text-right">
  {/* Content */}
</div>
```

---

## Database Schema

### Key Tables

#### company_ratings
```sql
CREATE TABLE company_ratings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    success_rate DECIMAL(5,2),
    sending_volume INTEGER,
    payment_compliance DECIMAL(5,2),
    service_quality_score DECIMAL(5,2),
    overall_rating DECIMAL(5,2),
    rating_category VARCHAR(20),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

#### server_health_logs
```sql
CREATE TABLE server_health_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL,
    tenant_id UUID,
    check_timestamp TIMESTAMP NOT NULL,
    response_time_ms INTEGER,
    status_code INTEGER,
    health_status VARCHAR(20),
    created_at TIMESTAMP DEFAULT NOW()
);
```

#### alerts
```sql
CREATE TABLE alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID,
    alert_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    title VARCHAR(255) NOT NULL,
    message TEXT,
    status VARCHAR(20) DEFAULT 'active',
    context JSONB,
    acknowledged_at TIMESTAMP,
    acknowledged_by VARCHAR(255),
    resolved_at TIMESTAMP,
    dedup_key VARCHAR(255),
    created_at TIMESTAMP DEFAULT NOW()
);
```

### Indexes

```sql
-- Performance indexes
CREATE INDEX idx_company_ratings_tenant ON company_ratings(tenant_id);
CREATE INDEX idx_server_health_logs_server ON server_health_logs(server_id);
CREATE INDEX idx_alerts_tenant_status ON alerts(tenant_id, status);
CREATE INDEX idx_alerts_dedup ON alerts(dedup_key);
```

---

## Testing

### Running Tests

```bash
# Backend tests
cd services/backend

# Unit tests
go test ./internal/... -v

# Integration tests
go test ./tests/integration/... -v

# With coverage
go test ./... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out

# Benchmarks
go test -bench=. -benchmem ./internal/admin/rating/
```

```bash
# Frontend tests (if implemented)
cd apps/admin-web
npm test
```

### Performance Tests

```bash
# Load test
k6 run performance-tests/admin-load-test.js

# Stress test
k6 run performance-tests/admin-stress-test.js
```

### Writing Tests

#### Unit Test Example

```go
func TestCalculateRating(t *testing.T) {
    calculator := NewRatingCalculator(defaultWeights)
    
    metrics := &PerformanceMetrics{
        TotalMessages:      10000,
        SuccessfulMessages: 9500,
        FailedMessages:     500,
    }
    
    rating, err := calculator.CalculateRating(metrics)
    
    assert.NoError(t, err)
    assert.InDelta(t, 85.0, rating, 1.0)
}
```

#### Integration Test Example

```go
func TestRatingSystemEndToEnd(t *testing.T) {
    db := setupTestDB(t)
    defer cleanupTestDB(t, db)
    
    tenantID := createTestTenant(t, db, "test-company")
    simulateMessageActivity(t, db, tenantID, 1000, 950)
    
    rating, err := calculateRating(context.Background(), db, tenantID)
    
    require.NoError(t, err)
    assert.Greater(t, rating.OverallRating, 80.0)
}
```

---

## Deployment

### Building for Production

#### Backend

```bash
# Build binary
cd services/backend
go build -o admin-server cmd/server/main.go

# Or with Docker
docker build -t admin-backend:latest .
```

#### Frontend

```bash
# Build production bundle
cd apps/admin-web
npm run build

# Output in dist/ directory
```

### Environment Variables

#### Backend (.env)

```env
# Database
DATABASE_URL=postgresql://user:pass@localhost:5432/admin_prod
REDIS_URL=redis://localhost:6379

# Server
PORT=8080
ENV=production

# JWT
JWT_SECRET=your-secret-key
JWT_EXPIRY=24h

# CORS
ALLOWED_ORIGINS=https://admin.example.com
```

#### Frontend (.env.production)

```env
VITE_API_BASE_URL=https://api.example.com
VITE_WS_URL=wss://api.example.com/ws
```

### Docker Deployment

```dockerfile
# Backend Dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
RUN go build -o server cmd/server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/server .
EXPOSE 8080
CMD ["./server"]
```

```dockerfile
# Frontend Dockerfile
FROM node:18-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/nginx.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
```

### Docker Compose

```yaml
version: '3.8'

services:
  backend:
    build: ./services/backend
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=postgresql://postgres:password@db:5432/admin
      - REDIS_URL=redis://redis:6379
    depends_on:
      - db
      - redis

  frontend:
    build: ./apps/admin-web
    ports:
      - "80:80"
    depends_on:
      - backend

  db:
    image: postgres:15-alpine
    environment:
      - POSTGRES_DB=admin
      - POSTGRES_PASSWORD=password
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    volumes:
      - redis_data:/data

volumes:
  postgres_data:
  redis_data:
```

---

## Contributing

### Code Style

**Go:**
- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for formatting
- Run `golint` before committing

**JavaScript:**
- Follow [Airbnb JavaScript Style Guide](https://github.com/airbnb/javascript)
- Use ESLint
- Use Prettier for formatting

### Git Workflow

```bash
# Create feature branch
git checkout -b feature/new-feature

# Make changes and commit
git add .
git commit -m "feat: add new feature"

# Push and create PR
git push origin feature/new-feature
```

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add new rating algorithm
fix: resolve health check timeout issue
docs: update API documentation
test: add integration tests for routing
refactor: simplify distribution logic
```

---

## Additional Resources

- [Go Documentation](https://golang.org/doc/)
- [React Documentation](https://react.dev/)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [Tailwind CSS Documentation](https://tailwindcss.com/docs)

---

**Developer Guide Version:** 1.0.0  
**Last Updated:** April 9, 2026  
**Maintainers:** Development Team
