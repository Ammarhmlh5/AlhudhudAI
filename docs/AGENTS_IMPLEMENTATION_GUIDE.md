# دليل تنفيذ نظام الوكلاء والموظفين

## نظرة عامة

هذا الدليل يشرح كيفية دمج نظام الوكلاء والموظفين في منصة الهدهد.

---

## الملفات المُنشأة

### 1. قاعدة البيانات
- `migrations/092_agents_system.sql` - جداول الوكلاء والموظفين

### 2. النماذج (Models)
- `internal/models/agent.go` - نماذج الوكلاء والموظفين

### 3. الخدمة (Service)
- `internal/agent/service.go` - منطق الأعمال

### 4. المعالج (Handler)
- `internal/agent/handler.go` - نقاط النهاية

### 5. المصادقة (Auth)
- `internal/agent/auth.go` - JWT Middleware

### 6. العزل والصلاحيات (Scope)
- `internal/agent/scope.go` - Guard Middleware

### 7. التوجيه (Router)
- `internal/agent/router.go` - إعداد المسارات
- `internal/agent/setup.go` - دالة التهيئة

---

## كيفية التكامل

### الخطوة 1: تشغيل Migration

```bash
# تشغيل Migration
psql -h localhost -d hudhud -U postgres -f migrations/092_agents_system.sql
```

### الخطوة 2: إضافة التهيئة في main.go

```go
package main

import (
    "github.com/al-hudhud/backend/internal/agent"
    "github.com/gin-gonic/gin"
    "gorm.io/gorm"
)

func main() {
    // ... existing code ...
    
    // Initialize agent system
    agent.Initialize(router, db)
    
    // ... rest of main.go ...
}
```

### الخطوة 3: استخدام النماذج

```go
import (
    "github.com/al-hudhud/backend/internal/models"
)

// إنشاء وكيل جديد
agent := models.Agent{
    CompanyName: "شركة البرمجة",
    Email: "company@example.com",
    PasswordHash: hash, // bcrypt
}

// إنشاء موظف
employee := models.AgentEmployee{
    AgentID: agent.ID,
    Name: "أحمد",
    Email: "ahmed@company.com",
    Role: "staff", // or "admin"
}
```

---

## نقاط النهاية (API Endpoints)

### المصادقة
| الطريقة | المسار | الوصف |
|---------|--------|-------|
| POST | `/api/v1/agents/auth/login` | تسجيل دخول الوكيل |
| POST | `/api/v1/agents/auth/employees/login` | تسجيل دخول الموظف |
| POST | `/api/v1/agents/auth/change-password` | تغيير كلمة المرور |

### الموظفين (للوكيل فقط)
| الطريقة | المسار | الوصف |
|---------|--------|-------|
| POST | `/api/v1/agents/employees` | إنشاء موظف |
| GET | `/api/v1/agents/employees` | قائمة الموظفين |
| GET | `/api/v1/agents/employees/:id` | تفاصيل موظف |
| DELETE | `/api/v1/agents/employees/:id` | حذف موظف |

### المستأجرين
| الطريقة | المسار | الوصف |
|---------|--------|-------|
| POST | `/api/v1/agents/tenants` | إنشاء مستأجر |
| GET | `/api/v1/agents/tenants` | قائمة المستأجرين |
| GET | `/api/v1/agents/tenants/:id` | تفاصيل مستأجر |
| DELETE | `/api/v1/agents/tenants/:id` | ❌ محظور |
| PUT | `/api/v1/agents/tenants/:id/suspend` | ❌ محظور |
| PUT | `/api/v1/agents/tenants/:id/activate` | ❌ محظور |

### المدفوعات
| الطريقة | المسار | الوصف |
|---------|--------|-------|
| POST | `/api/v1/agents/payments` | سداد فاتورة |
| GET | `/api/v1/agents/payments` | قائمة المدفوعات |

---

## أمثلة الاستخدام

### 1. تسجيل دخول الوكيل

```bash
curl -X POST http://localhost:8085/api/v1/agents/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "company@example.com",
    "password": "password123"
  }'
```

**الاستجابة:**
```json
{
  "success": true,
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
    "user": { ... },
    "user_type": "agent",
    "agent_id": 1
  }
}
```

### 2. إنشاء موظف

```bash
curl -X POST http://localhost:8085/api/v1/agents/employees \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <agent_token>" \
  -d '{
    "name": "أحمد محمد",
    "email": "ahmed@company.com",
    "password": "securePassword123",
    "role": "staff"
  }'
```

### 3. إنشاء مستأجر

```bash
curl -X POST http://localhost:8085/api/v1/agents/tenants \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "name": "عميل جديد",
    "email": "client@company.com",
    "phone": "+966501234567"
  }'
```

### 4. سداد فاتورة

```bash
curl -X POST http://localhost:8085/api/v1/agents/payments \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "invoice_id": "550e8400-e29b-41d4-a716-446655440000",
    "amount": 500.00,
    "payment_method": "bank_transfer",
    "transaction_id": "TX123456"
  }'
```

---

## هيكل قاعدة البيانات

### agents
```sql
CREATE TABLE agents (
    id UUID PRIMARY KEY,
    company_name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    phone VARCHAR(50),
    api_key VARCHAR(255) UNIQUE,
    api_secret VARCHAR(255),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
);
```

### agent_employees
```sql
CREATE TABLE agent_employees (
    id UUID PRIMARY KEY,
    agent_id UUID NOT NULL REFERENCES agents(id),
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(20) DEFAULT 'staff',
    is_active BOOLEAN DEFAULT true,
    last_login TIMESTAMPTZ,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
);
```

### agent_tenants
```sql
CREATE TABLE agent_tenants (
    id UUID PRIMARY KEY,
    agent_id UUID NOT NULL REFERENCES agents(id),
    tenant_id UUID NOT NULL,
    created_by UUID,
    created_at TIMESTAMPTZ
);
```

### agent_payments
```sql
CREATE TABLE agent_payments (
    id UUID PRIMARY KEY,
    invoice_id UUID,
    employee_id UUID REFERENCES agent_employees(id),
    agent_id UUID NOT NULL REFERENCES agents(id),
    tenant_id UUID,
    amount DECIMAL(10,2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'SAR',
    payment_method VARCHAR(50),
    transaction_id VARCHAR(255),
    status VARCHAR(20) DEFAULT 'pending',
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
);
```

---

## متطلبات الأمان

### 1. JWT Configuration
```go
const JWTKey = "hudhud-agent-jwt-secret-key" // تغيير في الإنتاج
```

### 2. Rate Limiting
```go
var defaultRateLimitConfig = rateLimitConfig{
    Login: rateLimitRule{
        Requests:       5,
        WindowSeconds:  60,
        BlockDuration:  900, // 15 minutes
    },
    API: rateLimitRule{
        Requests:       100,
        WindowSeconds:  60,
        BlockDuration:  60,
    },
    Payment: rateLimitRule{
        Requests:       10,
        WindowSeconds:  60,
        BlockDuration:  300,
    },
}
```

---

## الاختبار

### تشغيل Build Test
```bash
cd services/backend
go build
```

### تشغيل الاختبارات
```bash
cd services/backend
go test ./internal/agent/...
```

---

## المتبقي للتنفيذ

- [ ] تشغيل Migration على قاعدة البيانات
- [ ] إضافة `agent.Initialize()` في main.go
- [ ] دمج مع نظام الفواتير الحالي
- [ ] اختبارات التكامل
- [ ] توثيق API بالـ Swagger

---

## الدعم

للاستفسارات أو المشاكل، راجع:
- `docs/AGENTS_TECHNICAL_REQUIREMENTS.md` - المتطلبات التقنية
- `services/backend/AGENTS.md` - دليل الوكيل