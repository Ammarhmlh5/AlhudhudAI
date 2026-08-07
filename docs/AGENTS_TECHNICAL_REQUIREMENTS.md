# خطة المتطلبات الفنية والتقنية - نظام الوكلاء والموظفين

## الملخص التنفيذي

هذا المستند يحدد المتطلبات الفنية والتقنية الكاملة لنظام إدارة الوكلاء والموظفين في منصة الهدهد. يهدف النظام إلى تمكين شركات البرمجة (الوكلاء) من إدارة عملائهم (المستأجرين) من خلال واجهة برمجية محددة الصلاحيات.

---

## 1. متطلبات قاعدة البيانات

### 1.1 الجداول الأساسية

```sql
-- ============================================================================
-- جدول الوكلاء (شركات البرمجة)
-- ============================================================================
CREATE TABLE agents (
    id                  BIGSERIAL PRIMARY KEY,
    company_name        VARCHAR(255) NOT NULL,
    email               VARCHAR(255) UNIQUE NOT NULL,
    password_hash       VARCHAR(255) NOT NULL,
    phone               VARCHAR(50),
    api_key             VARCHAR(255) UNIQUE,
    api_secret          VARCHAR(255),  -- مشفر
    is_active           BOOLEAN DEFAULT true,
    created_at          TIMESTAMPTZ DEFAULT NOW(),
    updated_at          TIMESTAMPTZ DEFAULT NOW()
);

-- فهارس
CREATE INDEX idx_agents_email ON agents(email);
CREATE INDEX idx_agents_api_key ON agents(api_key) WHERE api_key IS NOT NULL;

-- ============================================================================
-- جدول موظفي الوكلاء
-- ============================================================================
CREATE TABLE agent_employees (
    id                  BIGSERIAL PRIMARY KEY,
    agent_id            BIGINT NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    name                VARCHAR(255) NOT NULL,
    email               VARCHAR(255) UNIQUE NOT NULL,
    password_hash       VARCHAR(255) NOT NULL,
    role                VARCHAR(20) DEFAULT 'staff',  -- 'admin', 'staff'
    is_active           BOOLEAN DEFAULT true,
    last_login          TIMESTAMPTZ,
    created_at          TIMESTAMPTZ DEFAULT NOW(),
    updated_at          TIMESTAMPTZ DEFAULT NOW()
);

-- فهارس
CREATE INDEX idx_agent_employees_agent_id ON agent_employees(agent_id);
CREATE INDEX idx_agent_employees_email ON agent_employees(email);

-- ============================================================================
-- جدول ربط الوكلاء بمستأجرينهم
-- ============================================================================
CREATE TABLE agent_tenants (
    id                  BIGSERIAL PRIMARY KEY,
    agent_id            BIGINT NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    tenant_id           BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    created_by          BIGINT,  -- employee_id who created
    created_at          TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(agent_id, tenant_id)
);

-- فهارس
CREATE INDEX idx_agent_tenants_agent_id ON agent_tenants(agent_id);
CREATE INDEX idx_agent_tenants_tenant_id ON agent_tenants(tenant_id);

-- ============================================================================
-- جدول الفواتير
-- ============================================================================
CREATE TABLE invoices (
    id                  BIGSERIAL PRIMARY KEY,
    tenant_id           BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    amount              DECIMAL(10,2) NOT NULL,
    currency            VARCHAR(3) DEFAULT 'SAR',
    description         TEXT,
    status              VARCHAR(20) DEFAULT 'pending',  -- pending, paid, overdue, cancelled
    due_date            TIMESTAMPTZ NOT NULL,
    paid_at             TIMESTAMPTZ,
    created_at          TIMESTAMPTZ DEFAULT NOW(),
    updated_at          TIMESTAMPTZ DEFAULT NOW()
);

-- فهارس
CREATE INDEX idx_invoices_tenant_id ON invoices(tenant_id);
CREATE INDEX idx_invoices_status ON invoices(status);
CREATE INDEX idx_invoices_due_date ON invoices(due_date);

-- ============================================================================
-- جدول المدفوعات
-- ============================================================================
CREATE TABLE payments (
    id                  BIGSERIAL PRIMARY KEY,
    invoice_id          BIGINT NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    employee_id         BIGINT REFERENCES agent_employees(id) ON DELETE SET NULL,
    agent_id            BIGINT NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    amount              DECIMAL(10,2) NOT NULL,
    currency            VARCHAR(3) DEFAULT 'SAR',
    payment_method      VARCHAR(50),  -- credit_card, bank_transfer, apple_pay, etc.
    transaction_id      VARCHAR(255),  -- من بوابة الدفع
    notes               TEXT,
    created_at          TIMESTAMPTZ DEFAULT NOW()
);

-- فهارس
CREATE INDEX idx_payments_invoice_id ON payments(invoice_id);
CREATE INDEX idx_payments_employee_id ON payments(employee_id);
CREATE INDEX idx_payments_agent_id ON payments(agent_id);
CREATE INDEX idx_payments_created_at ON payments(created_at);

-- ============================================================================
-- جدول جلسات تسجيل الدخول
-- ============================================================================
CREATE TABLE agent_sessions (
    id                  BIGSERIAL PRIMARY KEY,
    user_id             BIGINT NOT NULL,  -- agent_id or employee_id
    user_type           VARCHAR(20) NOT NULL,  -- 'agent' or 'employee'
    token               VARCHAR(500) NOT NULL,
    refresh_token       VARCHAR(500),
    expires_at          TIMESTAMPTZ NOT NULL,
    ip_address          VARCHAR(45),
    user_agent          TEXT,
    created_at          TIMESTAMPTZ DEFAULT NOW()
);

-- فهارس
CREATE INDEX idx_agent_sessions_token ON agent_sessions(token);
CREATE INDEX idx_agent_sessions_user ON agent_sessions(user_id, user_type);

-- ============================================================================
-- جدول سجل الأنشطة
-- ============================================================================
CREATE TABLE agent_activity_log (
    id                  BIGSERIAL PRIMARY KEY,
    user_id             BIGINT NOT NULL,
    user_type           VARCHAR(20) NOT NULL,  -- 'agent' or 'employee'
    agent_id            BIGINT,  -- مرتبط بالوكيل
    action              VARCHAR(100) NOT NULL,
    entity_type         VARCHAR(50),  -- tenant, invoice, payment, employee
    entity_id           BIGINT,
    details             JSONB,
    ip_address          VARCHAR(45),
    created_at          TIMESTAMPTZ DEFAULT NOW()
);

-- فهارس
CREATE INDEX idx_agent_activity_log_user ON agent_activity_log(user_id, user_type);
CREATE INDEX idx_agent_activity_log_agent_id ON agent_activity_log(agent_id);
CREATE INDEX idx_agent_activity_log_created_at ON agent_activity_log(created_at);
```

### 1.2 قيود التكامل المرجعي

```sql
-- ضمان أن الموظف ينتمي لوكيل موجود
ALTER TABLE agent_employees 
ADD CONSTRAINT fk_agent_employees_agent 
FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE CASCADE;

-- ضمان أن المستأجر المرتبط بالوكيل موجود
ALTER TABLE agent_tenants 
ADD CONSTRAINT fk_agent_tenants_agent 
FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE CASCADE;

-- ضمان أن المدفوعات مرتبطة بوكيل صحيح
ALTER TABLE payments 
ADD CONSTRAINT fk_payments_agent 
FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE CASCADE;
```

---

## 2. متطلبات API

### 2.1 نقاط النهاية

#### Authentication Endpoints

| الطريقة | المسار | الوصف | الوصول |
|---------|--------|-------|--------|
| POST | `/api/v1/agents/auth/login` | تسجيل دخول الوكيل | عام |
| POST | `/api/v1/agents/auth/employees/login` | تسجيل دخول الموظف | عام |
| POST | `/api/v1/agents/auth/refresh` | تجديد التوكن | عام |
| POST | `/api/v1/agents/auth/logout` | تسجيل خروج | الوكيل/الموظف |
| POST | `/api/v1/agents/auth/change-password` | تغيير كلمة مرور الوكيل | الوكيل |
| POST | `/api/v1/agents/auth/employees/{id}/change-password` | تغيير كلمة مرور الموظف | الموظف |

#### Employee Management (Agent Only)

| الطريقة | المسار | الوصف | الوصول |
|---------|--------|-------|--------|
| POST | `/api/v1/agents/employees` | إنشاء موظف جديد | الوكيل فقط |
| GET | `/api/v1/agents/employees` | قائمة الموظفين | الوكيل فقط |
| GET | `/api/v1/agents/employees/{id}` | تفاصيل موظف | الوكيل فقط |
| PUT | `/api/v1/agents/employees/{id}` | تحديث موظف | الوكيل فقط |
| DELETE | `/api/v1/agents/employees/{id}` | حذف موظف | الوكيل فقط |

#### Tenant Management

| الطريقة | المسار | الوصف | الوصول |
|---------|--------|-------|--------|
| POST | `/api/v1/agents/tenants` | إنشاء مستأجر | الوكيل/الموظف |
| GET | `/api/v1/agents/tenants` | قائمة المستأجرين | الوكيل/الموظف |
| GET | `/api/v1/agents/tenants/{id}` | تفاصيل مستأجر | الوكيل/الموظف |
| GET | `/api/v1/agents/tenants/{id}/invoices` | فواتير المستأجر | الوكيل/الموظف |

#### Payment Management

| الطريقة | المسار | الوصف | الوصول |
|---------|--------|-------|--------|
| GET | `/api/v1/agents/invoices` | قائمة الفواتير | الوكيل/الموظف |
| GET | `/api/v1/agents/invoices/{id}` | تفاصيل فاتورة | الوكيل/الموظف |
| POST | `/api/v1/agents/payments` | سداد فاتورة | الوكيل/الموظف |
| GET | `/api/v1/agents/payments` | سجل المدفوعات | الوكيل/الموظف |

#### Protected (Blocked)

| الطريقة | المسار | الاستجابة |
|---------|--------|----------|
| DELETE | `/api/v1/agents/tenants/{id}` | 403 Forbidden |
| PUT | `/api/v1/agents/tenants/{id}/suspend` | 403 Forbidden |
| PUT | `/api/v1/agents/tenants/{id}/activate` | 403 Forbidden |
| PUT | `/api/v1/agents/tenants/{id}` | 403 Forbidden |

### 2.2 تنسيق الاستجابات

```json
// نجاح مع بيانات
{
    "success": true,
    "data": { ... },
    "meta": {
        "page": 1,
        "limit": 20,
        "total": 100
    }
}

// نجاح بدون بيانات
{
    "success": true,
    "message": "تم الإجراء بنجاح"
}

// خطأ
{
    "success": false,
    "error": {
        "code": "VALIDATION_ERROR",
        "message": "الرسالة العربية",
        "details": [...]
    }
}

// فشل المصادقة
{
    "success": false,
    "error": {
        "code": "UNAUTHORIZED",
        "message": "يرجى تسجيل الدخول"
    }
}

// محظور
{
    "success": false,
    "error": {
        "code": "FORBIDDEN",
        "message": "غير مسموح - هذا الإجراء يتم من المنصة الرئيسية"
    }
}
```

### 2.3 حدود الطلبات (Rate Limiting)

```yaml
rate_limits:
  login:
    requests: 5
    window: 1m
    block_duration: 15m
  
  api:
    requests: 100
    window: 1m
  
  payment:
    requests: 10
    window: 1m
```

---

## 3. متطلبات الأمان

### 3.1 تشفير البيانات

| البند | المتطلب | الطريقة |
|-------|---------|---------|
| كلمات المرور | تشفير بإتجاه واحد | bcrypt (cost: 14) |
| مفاتيح API | تشفير في قاعدة البيانات | AES-256 |
| رموز Refresh | تخزين مشفر | AES-256 |
| البيانات الحساسة في السجلات | عدم التخزين | - |

### 3.2 المصادقة والتفويض

```go
type JWTClaims struct {
    UserID    int64  `json:"user_id"`    // agent_id or employee_id
    AgentID   int64  `json:"agent_id"`   // ALWAYS present
    Role      string `json:"role"`       // "agent", "employee_admin", "employee_staff"
    UserType  string `json:"user_type"`  // "agent" or "employee"
    Type      string `json:"type"`       // "access" or "refresh"
    Exp       int64  `json:"exp"`
    Iat       int64  `json:"iat"`
}
```

#### متطلبات التوكن
- **Access Token:** مدة صلاحية 1 ساعة
- **Refresh Token:** مدة صلاحية 7 أيام
- **التمييز:** `user_type` يحدد نوع المستخدم
- **العزل:** `agent_id` يُستخدم في جميع الاستعلامات

### 3.3 سياسات الأمان

```yaml
security:
  password_policy:
    min_length: 8
    require_uppercase: true
    require_lowercase: true
    require_numbers: true
    require_special_chars: false
    max_age_days: 90
  
  session:
    max_concurrent: 3
    idle_timeout: 30m
    absolute_timeout: 24h
  
  api_key:
    length: 32
    prefix: "hud_"
    rotation_days: 90
```

### 3.4 سجل التدقيق

كل إجراء يتم تسجيله:
```json
{
    "user_id": 123,
    "user_type": "employee",
    "agent_id": 45,
    "action": "create_tenant",
    "entity_type": "tenant",
    "entity_id": 789,
    "details": {
        "tenant_name": "شركة جديدة",
        "tenant_email": "new@company.com"
    },
    "ip_address": "192.168.1.100",
    "timestamp": "2026-08-06T10:30:00Z"
}
```

---

## 4. متطلبات الأداء

### 4.1 مؤشرات الأداء المستهدفة

| المقياس | الهدف |
|---------|-------|
| وقت الاستجابة (P95) | < 200ms |
| وقت الاستجابة (P99) | < 500ms |
| إمكانية التوسع | 100+ وكيل متزامن |
| وقت الاستجابة API | < 100ms للمعدلات |
| وقت استعلام قاعدة البيانات | < 50ms |

### 4.2 التخزين المؤقت

```yaml
caching:
  # تخزين مؤقت لبيانات الوكيل والموظف
  user_data:
    ttl: 5m
    max_size: 10000
  
  # تخزين مؤقت لقائمة المستأجرين
  tenant_list:
    ttl: 1m
    invalidate_on_change: true
  
  # تخزين مؤقت للفواتير
  invoices:
    ttl: 30s
```

### 4.3 تحسين قاعدة البيانات

- فهارس على الأعمدة المستخدمة في WHERE و JOIN
-_partitioning لجدول السجلات حسب التاريخ
- استخدام CONNECTION POOLing
- تحسين الاستعلامات المتكررة

---

## 5. متطلبات التكامل

### 5.1 التكامل مع بوابات الدفع

```yaml
payment_gateways:
  # بوابة الدفع الافتراضية
  default: "mada"
  
  gateways:
    mada:
      api_url: "https://api.mada.com.sa/v1"
      merchant_id: "${MADA_MERCHANT_ID}"
      test_mode: true
    
    apple_pay:
      api_url: "https://apple-pay-gateway.example.com"
      certificate_path: "/certs/apple-pay.p12"
    
    bank_transfer:
      enabled: true
      auto_verify: false
```

### 5.2 تكامل الإشعارات

```yaml
notifications:
  email:
    enabled: true
    provider: "ses"  # or smtp
    
  sms:
    enabled: true
    provider: "hudhud"  # منصة الهدهد
    
  push:
    enabled: false  # مستقبلاً
```

---

## 6. متطلبات الاختبار

### 6.1 أنواع الاختبارات

| النوع | التغطية المستهدفة | الأدوات |
|-------|-------------------|---------|
| Unit Tests | 80%+ | go test |
| Integration Tests | 100% endpoints | go test + testdb |
| E2E Tests | User flows | Playwright |
| Security Tests | OWASP Top 10 | Manual + automated |
| Load Tests | Performance | k6 / locust |

### 6.2 سيناريوهات الاختبار الحرجة

```gherkin
# Scenario 1: عزل المستأجرين
Given موظف يتبع لوكيل 1
When يطلب قائمة المستأجرين
Then يجب أن يرى فقط مستأجرين وكيل 1
And لا يرى مستأجرين وكيل 2

# Scenario 2: منع الحذف
Given موظف أو وكيل
When يحاول حذف مستأجر
Then يجب أن يرفض الطلب
And يرجع 403 Forbidden

# Scenario 3: تتبع المدفوعات
Given موظف يسدد فاتورة
When يتم السداد بنجاح
Then يجب تسجيل employee_id في سجل المدفوعات
And يمكن للوكيل رؤية من قام بالسداد
```

---

## 7. متطلبات النشر

### 7.1 البيئة

```yaml
environments:
  development:
    replicas: 1
    resources:
      cpu: 500m
      memory: 512Mi
  
  staging:
    replicas: 2
    resources:
      cpu: 1000m
      memory: 1Gi
  
  production:
    replicas: 3+
    resources:
      cpu: 2000m
      memory: 2Gi
    autoscaling:
      enabled: true
      min_replicas: 3
      max_replicas: 10
      target_cpu: 70%
```

### 7.2 المراقبة

```yaml
monitoring:
  metrics:
    - request_duration
    - request_count
    - error_rate
    - active_connections
    - db_query_duration
  
  alerts:
    - error_rate > 5%
    - response_time_p99 > 1s
    - cpu_usage > 80%
    - memory_usage > 85%
```

---

## 8. قائمة التحقق للتنفيذ

### 8.1 قاعدة البيانات
- [ ] إنشاء جدول agents
- [ ] إنشاء جدول agent_employees
- [ ] إنشاء جدول agent_tenants
- [ ] إنشاء جدول invoices
- [ ] إنشاء جدول payments
- [ ] إنشاء جدول agent_sessions
- [ ] إنشاء جدول agent_activity_log
- [ ] إضافة الفهارس اللازمة
- [ ] إضافة قيود التكامل المرجعي

### 8.2 API
- [ ] تنفيذ نقاط نهاية المصادقة
- [ ] تنفيذ إدارة الموظفين (للوكيل فقط)
- [ ] تنفيذ إدارة المستأجرين
- [ ] تنفيذ نظام المدفوعات
- [ ] تنفيذ Scope Guard للمحظورات

### 8.3 الأمان
- [ ] إعداد JWT مع Claims كاملة
- [ ] تنفيذ عزل المستأجرين
- [ ] إعداد Rate Limiting
- [ ] إعداد تشفير كلمات المرور
- [ ] إعداد سجل التدقيق

### 8.4 الاختبار
- [ ] اختبارات_UNIT للـ services
- [ ] اختبارات تكامل للـ endpoints
- [ ] اختبارات الأمان
- [ ] اختبارات الحمل

---

## 9. ملاحظات

- جميع الأوقات بالتوقيت العالمي (UTC)
- اللغة العربية للاستخدامين النهائيين
- اللغة الإنجليزية للكود والتعليقات
- استخدام تنسيق ISO 8601 للتواريخ