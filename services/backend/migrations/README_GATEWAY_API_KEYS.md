# جدول Gateway API Keys - التوثيق الكامل

## نظرة عامة

جدول `gateway_api_keys` يدير مفاتيح API للبوابات المخصصة، مما يسمح للشركات المؤسسية بالتكامل مع أنظمتها الداخلية بشكل آمن.

## المتطلبات المغطاة

- **Requirement 12.1**: السماح بإنشاء API Keys متعددة لكل بوابة مخصصة
- **Requirement 12.2**: توليد مفتاح فريد ومشفّر عند الإنشاء
- **Requirement 12.3**: حفظ اسم المفتاح ووصفه
- **Requirement 12.4**: دعم تحديد صلاحيات مخصصة لكل API Key
- **Requirement 12.5**: دعم تحديد Rate Limit لكل API Key
- **Requirement 12.6**: دعم IP Whitelisting لكل API Key

## بنية الجدول

### الحقول الأساسية

```sql
id SERIAL PRIMARY KEY                    -- المعرف الفريد
tenant_id UUID NOT NULL                  -- معرف الشركة (للعزل)
gateway_id INTEGER NOT NULL              -- معرف البوابة المخصصة
```

### معلومات المفتاح

```sql
key_name VARCHAR(255) NOT NULL           -- اسم المفتاح (للتعريف)
description TEXT                         -- وصف المفتاح والغرض منه
api_key_hash VARCHAR(255) NOT NULL       -- Hash آمن لمفتاح API (bcrypt)
api_secret_hash VARCHAR(255)             -- Hash آمن للسر الإضافي (اختياري)
```

### الصلاحيات والحدود

```sql
permissions JSONB DEFAULT '{}'::jsonb    -- الصلاحيات المخصصة
rate_limit INTEGER DEFAULT 60            -- الحد الأقصى للطلبات/دقيقة
allowed_ips TEXT[]                       -- قائمة IP addresses المسموح بها
```

### الحالة والاستخدام

```sql
is_active BOOLEAN DEFAULT true           -- حالة المفتاح (نشط/معطل)
expires_at TIMESTAMP                     -- تاريخ انتهاء ال��لاحية (NULL = لا ينتهي)
last_used_at TIMESTAMP                   -- تاريخ آخر استخدام
usage_count INTEGER DEFAULT 0            -- عدد مرات الاستخدام
```

## الأمان والتشفير

### 1. تشفير API Keys

المفاتيح يتم تشفيرها باستخدام **bcrypt** قبل التخزين:

```go
// مثال على توليد وتشفير API Key
import "golang.org/x/crypto/bcrypt"

func GenerateAPIKey() (string, string, error) {
    // توليد مفتاح عشوائي
    apiKey := generateRandomString(32)
    
    // تشفير المفتاح باستخدام bcrypt
    hashedKey, err := bcrypt.GenerateFromPassword([]byte(apiKey), bcrypt.DefaultCost)
    if err != nil {
        return "", "", err
    }
    
    return apiKey, string(hashedKey), nil
}

// التحقق من المفتاح
func ValidateAPIKey(providedKey, hashedKey string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hashedKey), []byte(providedKey))
    return err == nil
}
```

### 2. IP Whitelisting

يدعم الجدول IP Whitelisting بصيغ متعددة:

- **IPv4**: `192.168.1.1`
- **IPv6**: `2001:db8::1`
- **CIDR Notation**: `10.0.0.0/24`, `2001:db8::/32`

```sql
-- مثال على إضافة IPs مسموحة
UPDATE gateway_api_keys
SET allowed_ips = ARRAY['192.168.1.1', '10.0.0.0/24', '2001:db8::/32']
WHERE id = 1;
```

### 3. Row Level Security (RLS)

تم تطبيق RLS للعزل التام بين الشركات:

```sql
-- Policy للقراءة
CREATE POLICY tenant_isolation_policy ON gateway_api_keys
    FOR SELECT
    USING (tenant_id = current_setting('app.current_tenant')::uuid);
```

## الصلاحيات (Permissions)

الصلاحيات يتم تخزينها بصيغة JSONB:

### الصلاحيات الافتراضية

```json
{
  "send_sms": true,
  "view_stats": true,
  "view_reports": true,
  "manage_gateway": false,
  "manage_users": false,
  "manage_api_keys": false
}
```

### أمثلة على الصلاحيات المخصصة

```sql
-- مفتاح للإرسال فقط
UPDATE gateway_api_keys
SET permissions = '{"send_sms": true, "view_stats": false, "view_reports": false}'::jsonb
WHERE id = 1;

-- مفتاح للمراقبة فقط
UPDATE gateway_api_keys
SET permissions = '{"send_sms": false, "view_stats": true, "view_reports": true}'::jsonb
WHERE id = 2;

-- مفتاح بصلاحيات كاملة
UPDATE gateway_api_keys
SET permissions = '{"send_sms": true, "view_stats": true, "view_reports": true, "manage_gateway": true}'::jsonb
WHERE id = 3;
```

## Rate Limiting

كل API Key له حد أقصى للطلبات في الدقيقة:

```sql
-- تحديد Rate Limit مخصص
UPDATE gateway_api_keys
SET rate_limit = 120  -- 120 طلب/دقيقة
WHERE id = 1;
```

## الوظائف المساعدة (Functions)

### 1. validate_api_key()

التحقق من صلاحية API Key مع فحص شامل:

```sql
SELECT * FROM validate_api_key(
    'hashed_api_key_here',
    '192.168.1.100'::inet
);
```

**الإرجاع:**
- `is_valid`: هل المفتاح صالح؟
- `key_id`: معرف المفتاح
- `gateway_id`: معرف البوابة
- `tenant_id`: معرف الشركة
- `permissions`: الصلاحيات
- `rate_limit`: الحد الأقصى للطلبات
- `error_message`: رسالة الخطأ (إن وجد)

**الفحوصات:**
1. ✅ وجود المفتاح
2. ✅ حالة المفتاح (is_active)
3. ✅ تاريخ الانتهاء (expires_at)
4. ✅ IP Whitelisting
5. ✅ تحديث آخر استخدام وعداد الاستخدام

### 2. cleanup_expired_api_keys()

تعطيل المفاتيح المنتهية تلقائياً:

```sql
-- تشغيل يدوي
SELECT cleanup_expired_api_keys();

-- يمكن جدولته كـ Cron Job
```

## Triggers

### 1. update_gateway_api_keys_updated_at

تحديث `updated_at` تلقائياً عند أي تعديل.

### 2. check_gateway_api_key_tenant_consistency

التحقق من أن `gateway_id` ينتمي لنفس `tenant_id`.

### 3. validate_gateway_api_key_ips

التحقق من صحة IP addresses في `allowed_ips`.

### 4. set_default_gateway_api_key_permissions

تعيين الصلاحيات الافتراضية عند الإنشاء.

## Indexes

تم إنشاء Indexes محسّنة للأداء:

```sql
-- Indexes للعزل
idx_gateway_api_keys_tenant              -- (tenant_id)
idx_gateway_api_keys_tenant_gateway      -- (tenant_id, gateway_id)
idx_gateway_api_keys_tenant_active       -- (tenant_id, is_active)
idx_gateway_api_keys_tenant_expires      -- (tenant_id, expires_at) WHERE expires_at IS NOT NULL

-- Indexes للبحث السريع
idx_gateway_api_keys_gateway             -- (gateway_id)
idx_gateway_api_keys_hash                -- (api_key_hash)
idx_gateway_api_keys_active              -- (is_active)
idx_gateway_api_keys_expires             -- (expires_at) WHERE expires_at IS NOT NULL
```

## أمثلة الاستخدام

### 1. إنشاء API Key جديد

```sql
INSERT INTO gateway_api_keys (
    tenant_id,
    gateway_id,
    key_name,
    description,
    api_key_hash,
    permissions,
    rate_limit,
    allowed_ips,
    expires_at
) VALUES (
    '123e4567-e89b-12d3-a456-426614174000',
    1,
    'Production API Key',
    'مفتاح API للإنتاج - إرسال الرسائل فقط',
    '$2a$10$...',  -- bcrypt hash
    '{"send_sms": true, "view_stats": true}'::jsonb,
    100,
    ARRAY['192.168.1.0/24', '10.0.0.5'],
    '2025-12-31 23:59:59'
);
```

### 2. عرض جميع المفاتيح النشطة لبوابة

```sql
SELECT 
    id,
    key_name,
    description,
    permissions,
    rate_limit,
    allowed_ips,
    last_used_at,
    usage_count,
    expires_at
FROM gateway_api_keys
WHERE gateway_id = 1
  AND is_active = true
ORDER BY created_at DESC;
```

### 3. تعطيل مفتاح API

```sql
UPDATE gateway_api_keys
SET is_active = false,
    updated_at = CURRENT_TIMESTAMP
WHERE id = 1;
```

### 4. تحديث Rate Limit

```sql
UPDATE gateway_api_keys
SET rate_limit = 200,
    updated_at = CURRENT_TIMESTAMP
WHERE id = 1;
```

### 5. إضافة IP جديد للقائمة المسموحة

```sql
UPDATE gateway_api_keys
SET allowed_ips = array_append(allowed_ips, '172.16.0.0/16'),
    updated_at = CURRENT_TIMESTAMP
WHERE id = 1;
```

### 6. عرض إحصائيات الاستخدام

```sql
SELECT 
    gak.key_name,
    gak.usage_count,
    gak.last_used_at,
    gak.rate_limit,
    dg.gateway_name
FROM gateway_api_keys gak
JOIN dedicated_gateways dg ON gak.gateway_id = dg.id
WHERE gak.gateway_id = 1
ORDER BY gak.usage_count DESC;
```

### 7. البحث عن المفاتيح المنتهية قريباً

```sql
SELECT 
    id,
    key_name,
    gateway_id,
    expires_at,
    EXTRACT(DAY FROM (expires_at - CURRENT_TIMESTAMP)) as days_remaining
FROM gateway_api_keys
WHERE expires_at IS NOT NULL
  AND expires_at > CURRENT_TIMESTAMP
  AND expires_at < CURRENT_TIMESTAMP + INTERVAL '30 days'
  AND is_active = true
ORDER BY expires_at ASC;
```

## التكامل مع Go

### نموذج البيانات (Model)

```go
package models

import (
    "time"
    "github.com/google/uuid"
    "github.com/lib/pq"
)

type GatewayAPIKey struct {
    ID            int            `json:"id" db:"id"`
    TenantID      uuid.UUID      `json:"tenant_id" db:"tenant_id"`
    GatewayID     int            `json:"gateway_id" db:"gateway_id"`
    KeyName       string         `json:"key_name" db:"key_name"`
    Description   *string        `json:"description,omitempty" db:"description"`
    APIKeyHash    string         `json:"-" db:"api_key_hash"` // لا يُعرض في JSON
    APISecretHash *string        `json:"-" db:"api_secret_hash"`
    Permissions   map[string]any `json:"permissions" db:"permissions"`
    RateLimit     int            `json:"rate_limit" db:"rate_limit"`
    AllowedIPs    pq.StringArray `json:"allowed_ips" db:"allowed_ips"`
    IsActive      bool           `json:"is_active" db:"is_active"`
    ExpiresAt     *time.Time     `json:"expires_at,omitempty" db:"expires_at"`
    LastUsedAt    *time.Time     `json:"last_used_at,omitempty" db:"last_used_at"`
    UsageCount    int            `json:"usage_count" db:"usage_count"`
    CreatedAt     time.Time      `json:"created_at" db:"created_at"`
    UpdatedAt     time.Time      `json:"updated_at" db:"updated_at"`
}

type APIKeyValidationResult struct {
    IsValid      bool           `db:"is_valid"`
    KeyID        *int           `db:"key_id"`
    GatewayID    *int           `db:"gateway_id"`
    TenantID     *uuid.UUID     `db:"tenant_id"`
    Permissions  map[string]any `db:"permissions"`
    RateLimit    *int           `db:"rate_limit"`
    ErrorMessage *string        `db:"error_message"`
}
```

### Repository

```go
package gateway

import (
    "context"
    "database/sql"
    "github.com/google/uuid"
    "golang.org/x/crypto/bcrypt"
)

type APIKeyRepository struct {
    db *sql.DB
}

func NewAPIKeyRepository(db *sql.DB) *APIKeyRepository {
    return &APIKeyRepository{db: db}
}

// CreateAPIKey إنشاء مفتاح API جديد
func (r *APIKeyRepository) CreateAPIKey(ctx context.Context, key *GatewayAPIKey, plainKey string) error {
    // تشفير المفتاح
    hashedKey, err := bcrypt.GenerateFromPassword([]byte(plainKey), bcrypt.DefaultCost)
    if err != nil {
        return err
    }
    
    key.APIKeyHash = string(hashedKey)
    
    query := `
        INSERT INTO gateway_api_keys (
            tenant_id, gateway_id, key_name, description,
            api_key_hash, permissions, rate_limit, allowed_ips, expires_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
        RETURNING id, created_at, updated_at
    `
    
    return r.db.QueryRowContext(ctx, query,
        key.TenantID, key.GatewayID, key.KeyName, key.Description,
        key.APIKeyHash, key.Permissions, key.RateLimit, key.AllowedIPs, key.ExpiresAt,
    ).Scan(&key.ID, &key.CreatedAt, &key.UpdatedAt)
}

// ValidateAPIKey التحقق من صلاحية مفتاح API
func (r *APIKeyRepository) ValidateAPIKey(ctx context.Context, apiKeyHash string, clientIP string) (*APIKeyValidationResult, error) {
    query := `SELECT * FROM validate_api_key($1, $2::inet)`
    
    var result APIKeyValidationResult
    err := r.db.QueryRowContext(ctx, query, apiKeyHash, clientIP).Scan(
        &result.IsValid,
        &result.KeyID,
        &result.GatewayID,
        &result.TenantID,
        &result.Permissions,
        &result.RateLimit,
        &result.ErrorMessage,
    )
    
    return &result, err
}

// GetAPIKeysByGateway عرض جميع مفاتيح API لبوابة
func (r *APIKeyRepository) GetAPIKeysByGateway(ctx context.Context, gatewayID int) ([]*GatewayAPIKey, error) {
    query := `
        SELECT id, tenant_id, gateway_id, key_name, description,
               permissions, rate_limit, allowed_ips, is_active,
               expires_at, last_used_at, usage_count, created_at, updated_at
        FROM gateway_api_keys
        WHERE gateway_id = $1
        ORDER BY created_at DESC
    `
    
    rows, err := r.db.QueryContext(ctx, query, gatewayID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var keys []*GatewayAPIKey
    for rows.Next() {
        var key GatewayAPIKey
        err := rows.Scan(
            &key.ID, &key.TenantID, &key.GatewayID, &key.KeyName, &key.Description,
            &key.Permissions, &key.RateLimit, &key.AllowedIPs, &key.IsActive,
            &key.ExpiresAt, &key.LastUsedAt, &key.UsageCount, &key.CreatedAt, &key.UpdatedAt,
        )
        if err != nil {
            return nil, err
        }
        keys = append(keys, &key)
    }
    
    return keys, rows.Err()
}

// DeactivateAPIKey تعطيل مفتاح API
func (r *APIKeyRepository) DeactivateAPIKey(ctx context.Context, keyID int) error {
    query := `
        UPDATE gateway_api_keys
        SET is_active = false, updated_at = CURRENT_TIMESTAMP
        WHERE id = $1
    `
    
    _, err := r.db.ExecContext(ctx, query, keyID)
    return err
}
```

## الأمان والأفضليات

### 1. تشفير المفاتيح

- ✅ استخدام bcrypt لتشفير API Keys
- ✅ عدم تخزين المفاتيح بشكل نصي
- ✅ عرض المفتاح للمستخدم مرة واحدة فقط عند الإنشاء

### 2. IP Whitelisting

- ✅ دعم IPv4 و IPv6
- ✅ دعم CIDR notation
- ✅ التحقق من صحة IP addresses
- ✅ تسجيل محاولات الوصول غير المصرح بها

### 3. Rate Limiting

- ✅ تحديد حد أقصى للطلبات لكل مفتاح
- ✅ قابل للتخصيص حسب الحاجة
- ✅ منع إساءة الاستخدام

### 4. انتهاء الصلاحية

- ✅ دعم تحديد تاريخ انتهاء
- ✅ تعطيل تلقائي للمفاتيح المنتهية
- ✅ إشعارات قبل الانتهاء

### 5. Audit Trail

- ✅ تسجيل آخر استخدام
- ✅ عداد الاستخدام
- ✅ تسجيل محاولات الوصول غير المصرح بها

## الصيانة

### تنظيف المفاتيح المنتهية

يُنصح بجدولة تنظيف دوري:

```sql
-- Cron Job يومي
SELECT cleanup_expired_api_keys();
```

### مراقبة الاستخدام

```sql
-- المفاتيح الأكثر استخداماً
SELECT key_name, usage_count, last_used_at
FROM gateway_api_keys
WHERE is_active = true
ORDER BY usage_count DESC
LIMIT 10;

-- المفاتيح غير المستخدمة
SELECT key_name, created_at
FROM gateway_api_keys
WHERE is_active = true
  AND (last_used_at IS NULL OR last_used_at < CURRENT_TIMESTAMP - INTERVAL '30 days')
ORDER BY created_at ASC;
```

## الخلاصة

جدول `gateway_api_keys` يوفر:

✅ **أمان قوي**: تشفير bcrypt، IP Whitelisting، انتهاء الصلاحية  
✅ **مرونة**: صلاحيات مخصصة، Rate Limiting قابل للتخصيص  
✅ **عزل تام**: RLS للعزل بين الشركات  
✅ **مراقبة شاملة**: تتبع الاستخدام، Audit Trail  
✅ **سهولة الإدارة**: Triggers تلقائية، Functions مساعدة  

## المهام القادمة

6. ⏳ **التالي**: جدول `gateway_statistics` (المهمة 16.6)
7. ⏳ تطوير Gateway Manager (المهمة 17)
8. ⏳ تطوير API Key Management APIs (المهمة 21)

---

**تاريخ الإنشاء**: 2024  
**الحالة**: ✅ مكتمل  
**المهمة التالية**: 16.6 - إنشاء جدول gateway_statistics
