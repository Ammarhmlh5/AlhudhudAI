# Gateway API Keys - دليل البدء السريع

## نظرة عامة سريعة

جدول `gateway_api_keys` يدير مفاتيح API للبوابات المخصصة بشكل آمن مع دعم:
- ✅ تشفير bcrypt للمفاتيح
- ✅ IP Whitelisting (IPv4/IPv6/CIDR)
- ✅ Rate Limiting قابل للتخصيص
- ✅ صلاحيات مخصصة
- ✅ انتهاء صلاحية تلقائي
- ✅ Row Level Security للعزل

## التثبيت السريع

### 1. تشغيل Migration

```powershell
# Windows
.\run-gateway-api-keys-migration.ps1

# أو يدوياً
psql -U postgres -d sms_gateway -f migrations/024_gateway_api_keys_table.sql
```

### 2. تشغيل الاختبارات

```powershell
psql -U postgres -d sms_gateway -f test_gateway_api_keys_migration.sql
```

## الاستخدام السريع

### إنشاء API Key

```sql
-- 1. توليد المفتاح وتشفيره (في Go)
-- apiKey := generateRandomString(32)
-- hashedKey := bcrypt.GenerateFromPassword([]byte(apiKey), bcrypt.DefaultCost)

-- 2. حفظ في قاعدة البيانات
INSERT INTO gateway_api_keys (
    tenant_id,
    gateway_id,
    key_name,
    description,
    api_key_hash,
    rate_limit,
    allowed_ips,
    expires_at
) VALUES (
    '123e4567-e89b-12d3-a456-426614174000',
    1,
    'Production API Key',
    'مفتاح API للإنتاج',
    '$2a$10$...',  -- bcrypt hash
    100,  -- 100 طلب/دقيقة
    ARRAY['192.168.1.0/24', '10.0.0.5'],
    '2025-12-31 23:59:59'
);
```

### التحقق من صلاحية المفتاح

```sql
SELECT * FROM validate_api_key(
    'hashed_api_key_here',
    '192.168.1.100'::inet
);
```

**الإرجاع:**
- `is_valid`: true/false
- `permissions`: الصلاحيات
- `rate_limit`: الحد الأقصى للطلبات
- `error_message`: رسالة الخطأ (إن وجد)

### عرض جميع المفاتيح

```sql
SELECT 
    id,
    key_name,
    permissions,
    rate_limit,
    allowed_ips,
    is_active,
    last_used_at,
    usage_count,
    expires_at
FROM gateway_api_keys
WHERE gateway_id = 1
ORDER BY created_at DESC;
```

### تعطيل مفتاح

```sql
UPDATE gateway_api_keys
SET is_active = false
WHERE id = 1;
```

## التكامل مع Go

### نموذج البيانات

```go
type GatewayAPIKey struct {
    ID            int            `json:"id"`
    TenantID      uuid.UUID      `json:"tenant_id"`
    GatewayID     int            `json:"gateway_id"`
    KeyName       string         `json:"key_name"`
    Description   *string        `json:"description,omitempty"`
    APIKeyHash    string         `json:"-"` // لا يُعرض
    Permissions   map[string]any `json:"permissions"`
    RateLimit     int            `json:"rate_limit"`
    AllowedIPs    []string       `json:"allowed_ips"`
    IsActive      bool           `json:"is_active"`
    ExpiresAt     *time.Time     `json:"expires_at,omitempty"`
    LastUsedAt    *time.Time     `json:"last_used_at,omitempty"`
    UsageCount    int            `json:"usage_count"`
    CreatedAt     time.Time      `json:"created_at"`
    UpdatedAt     time.Time      `json:"updated_at"`
}
```

### إنشاء مفتاح جديد

```go
import "golang.org/x/crypto/bcrypt"

func CreateAPIKey(ctx context.Context, gatewayID int, keyName string) (string, error) {
    // 1. توليد مفتاح عشوائي
    apiKey := generateRandomString(32)
    
    // 2. تشفير المفتاح
    hashedKey, err := bcrypt.GenerateFromPassword([]byte(apiKey), bcrypt.DefaultCost)
    if err != nil {
        return "", err
    }
    
    // 3. حفظ في قاعدة البيانات
    query := `
        INSERT INTO gateway_api_keys (
            tenant_id, gateway_id, key_name, api_key_hash
        ) VALUES ($1, $2, $3, $4)
        RETURNING id
    `
    
    var keyID int
    err = db.QueryRowContext(ctx, query,
        tenantID, gatewayID, keyName, string(hashedKey),
    ).Scan(&keyID)
    
    if err != nil {
        return "", err
    }
    
    // 4. إرجاع المفتاح (يُعرض مرة واحدة فقط!)
    return apiKey, nil
}
```

### التحقق من المفتاح

```go
func ValidateAPIKey(ctx context.Context, providedKey, clientIP string) (*APIKeyValidationResult, error) {
    // 1. تشفير المفتاح المقدم
    hashedKey, err := bcrypt.GenerateFromPassword([]byte(providedKey), bcrypt.DefaultCost)
    if err != nil {
        return nil, err
    }
    
    // 2. استدعاء Function للتحقق
    query := `SELECT * FROM validate_api_key($1, $2::inet)`
    
    var result APIKeyValidationResult
    err = db.QueryRowContext(ctx, query, string(hashedKey), clientIP).Scan(
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
```

### Middleware للتحقق

```go
func APIKeyAuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. استخراج API Key من Header
        apiKey := c.GetHeader("X-API-Key")
        if apiKey == "" {
            c.JSON(401, gin.H{"error": "API key required"})
            c.Abort()
            return
        }
        
        // 2. الحصول على IP العميل
        clientIP := c.ClientIP()
        
        // 3. التحقق من صلاحية المفتاح
        result, err := ValidateAPIKey(c.Request.Context(), apiKey, clientIP)
        if err != nil || !result.IsValid {
            c.JSON(401, gin.H{"error": result.ErrorMessage})
            c.Abort()
            return
        }
        
        // 4. تعيين المعلومات في Context
        c.Set("gateway_id", result.GatewayID)
        c.Set("tenant_id", result.TenantID)
        c.Set("permissions", result.Permissions)
        c.Set("rate_limit", result.RateLimit)
        
        c.Next()
    }
}
```

## الصلاحيات

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

### التحقق من الصلاحيات

```go
func CheckPermission(c *gin.Context, permission string) bool {
    permissions, exists := c.Get("permissions")
    if !exists {
        return false
    }
    
    perms := permissions.(map[string]any)
    value, ok := perms[permission]
    if !ok {
        return false
    }
    
    return value.(bool)
}

// الاستخدام
if !CheckPermission(c, "send_sms") {
    c.JSON(403, gin.H{"error": "Permission denied"})
    return
}
```

## IP Whitelisting

### إضافة IPs مسموحة

```sql
-- IPv4
UPDATE gateway_api_keys
SET allowed_ips = ARRAY['192.168.1.1', '10.0.0.0/24']
WHERE id = 1;

-- IPv6
UPDATE gateway_api_keys
SET allowed_ips = ARRAY['2001:db8::1', '2001:db8::/32']
WHERE id = 1;

-- Mixed
UPDATE gateway_api_keys
SET allowed_ips = ARRAY['192.168.1.1', '2001:db8::1', '10.0.0.0/24']
WHERE id = 1;
```

### إضافة IP جديد

```sql
UPDATE gateway_api_keys
SET allowed_ips = array_append(allowed_ips, '172.16.0.0/16')
WHERE id = 1;
```

### إزالة IP

```sql
UPDATE gateway_api_keys
SET allowed_ips = array_remove(allowed_ips, '10.0.0.5')
WHERE id = 1;
```

## Rate Limiting

### تحديد Rate Limit

```sql
UPDATE gateway_api_keys
SET rate_limit = 200  -- 200 طلب/دقيقة
WHERE id = 1;
```

### تطبيق Rate Limiting في Go

```go
import "golang.org/x/time/rate"

var rateLimiters = make(map[int]*rate.Limiter)

func RateLimitMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        keyID := c.GetInt("key_id")
        rateLimit := c.GetInt("rate_limit")
        
        // الحصول على Rate Limiter
        limiter, exists := rateLimiters[keyID]
        if !exists {
            limiter = rate.NewLimiter(rate.Limit(rateLimit), rateLimit)
            rateLimiters[keyID] = limiter
        }
        
        // التحقق من الحد
        if !limiter.Allow() {
            c.JSON(429, gin.H{"error": "Rate limit exceeded"})
            c.Abort()
            return
        }
        
        c.Next()
    }
}
```

## انتهاء الصلاحية

### تحديد تاريخ الانتهاء

```sql
UPDATE gateway_api_keys
SET expires_at = '2025-12-31 23:59:59'
WHERE id = 1;
```

### تنظيف المفاتيح المنتهية

```sql
-- تشغيل يدوي
SELECT cleanup_expired_api_keys();

-- جدولة كـ Cron Job (يومياً)
-- 0 0 * * * psql -U postgres -d sms_gateway -c "SELECT cleanup_expired_api_keys();"
```

### البحث عن المفاتيح التي ستنتهي قريباً

```sql
SELECT 
    key_name,
    expires_at,
    EXTRACT(DAY FROM (expires_at - CURRENT_TIMESTAMP)) as days_remaining
FROM gateway_api_keys
WHERE expires_at IS NOT NULL
  AND expires_at > CURRENT_TIMESTAMP
  AND expires_at < CURRENT_TIMESTAMP + INTERVAL '30 days'
  AND is_active = true
ORDER BY expires_at ASC;
```

## الإحصائيات

### المفاتيح الأكثر استخداماً

```sql
SELECT 
    key_name,
    usage_count,
    last_used_at,
    rate_limit
FROM gateway_api_keys
WHERE is_active = true
ORDER BY usage_count DESC
LIMIT 10;
```

### المفاتيح غير المستخدمة

```sql
SELECT 
    key_name,
    created_at,
    last_used_at
FROM gateway_api_keys
WHERE is_active = true
  AND (last_used_at IS NULL OR last_used_at < CURRENT_TIMESTAMP - INTERVAL '30 days')
ORDER BY created_at ASC;
```

## الأمان

### ✅ أفضل الممارسات

1. **تشفير المفاتيح**: استخدام bcrypt دائماً
2. **عرض المفتاح مرة واحدة**: عند الإنشاء فقط
3. **IP Whitelisting**: تحديد IPs مسموحة
4. **Rate Limiting**: تحديد حدود معقولة
5. **انتهاء الصلاحية**: تحديد تاريخ انتهاء
6. **المراقبة**: تتبع الاستخدام والمحاولات الفاشلة

### ❌ تجنب

1. تخزين المفاتيح بشكل نصي
2. مشاركة المفاتيح بين البوابات
3. عدم تحديد Rate Limit
4. عدم مراقبة الاستخدام
5. عدم تعطيل المفاتيح المخترقة

## استكشاف الأخطاء

### المفتاح غير صالح

```sql
-- التحقق من وجود المفتاح
SELECT * FROM gateway_api_keys WHERE api_key_hash = 'hash_here';

-- التحقق من الحالة
SELECT key_name, is_active, expires_at
FROM gateway_api_keys
WHERE api_key_hash = 'hash_here';
```

### IP غير مسموح

```sql
-- عرض IPs المسموحة
SELECT key_name, allowed_ips
FROM gateway_api_keys
WHERE id = 1;

-- إضافة IP جديد
UPDATE gateway_api_keys
SET allowed_ips = array_append(allowed_ips, 'new_ip')
WHERE id = 1;
```

### تجاوز Rate Limit

```sql
-- عرض Rate Limit الحالي
SELECT key_name, rate_limit, usage_count
FROM gateway_api_keys
WHERE id = 1;

-- زيادة Rate Limit
UPDATE gateway_api_keys
SET rate_limit = 500
WHERE id = 1;
```

## الموارد

### الوثائق

- 📚 **README الكامل**: `migrations/README_GATEWAY_API_KEYS.md`
- 📝 **Migration**: `migrations/024_gateway_api_keys_table.sql`
- 🧪 **الاختبارات**: `test_gateway_api_keys_migration.sql`

### الأوامر المفيدة

```sql
-- عرض جميع المفاتيح
SELECT * FROM gateway_api_keys;

-- التحقق من صلاحية مفتاح
SELECT * FROM validate_api_key('hash', '192.168.1.1'::inet);

-- تنظيف المفاتيح المنتهية
SELECT cleanup_expired_api_keys();

-- عرض الإحصائيات
SELECT 
    COUNT(*) as total,
    COUNT(*) FILTER (WHERE is_active) as active,
    AVG(rate_limit)::INT as avg_rate_limit
FROM gateway_api_keys;
```

## المهام القادمة

في سياق مشروع tariff-pricing-system:

1. ✅ **تم**: جدول `dedicated_gateways` (المهمة 16.1)
2. ✅ **تم**: جدول `resource_quotas` (المهمة 16.2)
3. ✅ **تم**: جدول `resource_usage` (المهمة 16.3)
4. ✅ **تم**: جدول `gateway_users` (المهمة 16.4)
5. ✅ **الحالي**: جدول `gateway_api_keys` (المهمة 16.5)
6. ⏳ **التالي**: جدول `gateway_statistics` (المهمة 16.6)
7. ⏳ تطوير Gateway Manager (المهمة 17)
8. ⏳ تطوير API Key Management APIs (المهمة 21)

---

**تاريخ الإنشاء**: 2024  
**الحالة**: ✅ جاهز للاستخدام  
**المهمة**: 16.5 - إنشاء جدول gateway_api_keys
