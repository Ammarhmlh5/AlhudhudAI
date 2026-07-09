# مستند التصميم - نظام التعرفات والتسعير

## المقدمة

هذا المستند يحدد التصميم الفني الشامل لنظام التعرفات والتسعير لمنصة SMS Gateway. النظام يدعم ثلاثة نماذج خدمات مختلفة مع معالجة شاملة للمشاكل الحرجة في النظام الحالي.

### نطاق التصميم

يغطي هذا التصميم:
- المعمارية العامة للنظام
- تصميم قاعدة البيانات التفصيلي
- تصميم APIs والواجهات
- معالجة المشاكل الحرجة في النظام الحالي
- استراتيجية التنفيذ على 7 مراحل

### الأهداف الرئيسية

1. **المرونة**: دعم ثلاثة نماذج خدمات مختلفة بسلاسة
2. **الأمان المالي**: معالجة جميع المشاكل الحرجة في عمليات المحاسبة
3. **القابلية للتوسع**: دعم النمو المستقبلي والتوسع الجغرافي
4. **الموثوقية**: ضمان دقة العمليات المالية بنسبة 100%

---

## المعمارية العامة

### نماذج الخدمات الثلاثة

#### 1. خدمة بوابة الهدهد (Hudud Gateway Service)

```
النموذج: Pay-as-you-go (الدفع حسب الاستخدام)

المكونات:
├── نظام التعرفات الديناميكي
│   ├── تعرفة يمن موبايل: 9 ريال/رسالة
│   ├── تعرفة شركات أخرى: 13 ريال/رسالة
│   └── دعم التوسع لدول جديدة
├── محرك حساب التكلفة
│   ├── تقسيم الرسائل (70، 65، 62 حرف)
│   ├── حساب عدد الأجزاء تلقائياً
│   └── حساب التكلفة الإجمالية
└── نظام الخصم الفوري
    ├── Atomic Balance Deduction
    ├── Transaction Isolation
    └── Rollback عند الفشل
```

#### 2. خدمة الإرسال عبر الهاتف (Mobile Gateway Service)

```
النموذج: Subscription + Overage (اشتراك شهري + استخدام إضافي)

الباقات الشهرية:
├── Basic: $10 - 1,000 رسالة
├── Standard: $20 - 3,000 رسالة
├── Professional: $40 - 8,000 رسالة
├── Business: $50 - 15,000 رسالة
└── Enterprise: $100 - Unlimited

المزايا:
├── نظام Rollover المشروط (ترحيل الرسائل المشروط)
│   ├── قابل للتفعيل/التعطيل لكل باقة
│   ├── سياسة ترحيل مشروط (يعتمد على نجاح الدفع)
│   ├── حفظ مؤقت للرسائل المتبقية عند نهاية الشهر
│   ├── تفعيل الترحيل عند نجاح الدفع التلقائي
│   ├── حذف الرسائل المرحّلة عند فشل الدفع (لمنع الخسائر)
│   ├── حد أقصى قابل للتخصيص
│   ├── مدة صلاحية قابلة للتخصيص
│   └── زر تحكم في واجهة الإدارة لتفعيل/تعطيل السياسة
├── الاستخدام الإضافي (Overage)
│   ├── بنفس سعر الباقة
│   ├── خصم من الرصيد
│   └── تتبع دقيق للتكلفة
└── الفوترة التلقائية
    ├── فحص يومي للاشتراكات
    ├── خصم تلقائي عند التجديد
    ├── تفعيل الترحيل عند نجاح الدفع
    ├── حذف الرسائل المرحّلة عند فشل الدفع
    └── فترة سماح عند الفشل
```

#### 3. خدمة إدارة البوابات (Gateway-as-a-Service)

```
النموذج: Infrastructure as a Service للشركات الكبيرة

نموذج العمل المالي:
├── الشركة المؤسسية تدفع لشركة الاتصالات مباشرة لتكلفة الرسائل
├── المنصة تتقاضى فقط:
│   ├── رسوم الاشتراك (شهري أو سنوي)
│   └── رسوم إيجار السيرفر
└── المنصة تراقب رصيد الشركة لدى شركة الاتصالات وترسل تنبيهات

المكونات:
├── البوابة المخصصة
│   ├── اتصال مباشر بشركة الاتصالات
│   ├── معرفات وبيانات اعتماد خاصة
│   ├── مراقبة الرصيد (للقراءة فقط)
│   └── إحصائيات منفصلة
├── البنية التحتية المشتركة
│   ├── Shared Infrastructure على AWS
│   ├── Resource Quotas لكل بوابة
│   ├── عزل منطقي قوي
│   └── مراقبة مستمرة
├── إدارة الموارد
│   ├── CPU Quota
│   ├── Memory Quota
│   ├── Bandwidth Quota
│   ├── Database Connections
│   ├── Redis Memory
│   ├── Message Queue Quota
│   └── API Rate Limit
├── نظام الفوترة
│   ├── خصم رسوم الاشتراك (شهري/سنوي)
│   ├── خصم رسوم إيجار السيرفر
│   └── لا يتم خصم تكلفة الرسائل (الشركة تدفع مباشرة)
└── واجهة إدارة مخصصة
    ├── لوحة تحكم شاملة
    ├── إدارة المستخدمين
    ├── إدارة API Keys
    ├── مراقبة الرصيد مع التنبيهات
    └── تقارير مفصلة
```

### المعمارية الفنية

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

---

## العزل التام بين الشركات (Multi-Tenancy Isolation)

### المبدأ الأساسي

كل شركة (tenant) يجب أن تعمل وكأنها المستخدم الوحيد للنظام. العزل التام يضمن:
- عدم تسرب البيانات بين الشركات
- عدم تأثير أداء شركة على الأخرى
- عدم إمكانية الوصول لبيانات شركة أخرى بأي طريقة

### 1. عزل قاعدة البيانات (Database Isolation)

#### 1.1 إضافة tenant_id لجميع الجداول

```sql
-- كل جدول يجب أن يحتوي على
tenant_id UUID NOT NULL REFERENCES tenants(id)

-- مثال على جدول محدّث
CREATE TABLE tariffs (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),  -- إضافة إلزامية
    service_id INTEGER NOT NULL REFERENCES services(id),
    country_code VARCHAR(2) NOT NULL,
    -- ... بقية الحقول
);
```

#### 1.2 Row Level Security (RLS)

```sql
-- تفعيل RLS على كل جدول
ALTER TABLE tariffs ENABLE ROW LEVEL SECURITY;

-- إنشاء Policy للعزل
CREATE POLICY tenant_isolation_policy ON tariffs
    USING (tenant_id = current_setting('app.current_tenant')::uuid);

-- Policy للإدراج
CREATE POLICY tenant_isolation_insert_policy ON tariffs
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant')::uuid);

-- Policy للتحديث
CREATE POLICY tenant_isolation_update_policy ON tariffs
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant')::uuid);

-- Policy للحذف
CREATE POLICY tenant_isolation_delete_policy ON tariffs
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant')::uuid);
```

#### 1.3 تطبيق RLS على جميع الجداول

```sql
-- قائمة الجداول التي تحتاج RLS
DO $$
DECLARE
    table_name TEXT;
    tables TEXT[] := ARRAY[
        'tariffs',
        'packages',
        'subscriptions',
        'rollover_messages',
        'dedicated_gateways',
        'resource_quotas',
        'resource_usage',
        'gateway_users',
        'gateway_api_keys',
        'gateway_statistics',
        'accounting_transactions',
        'balance_history',
        'billing_invoices'
    ];
BEGIN
    FOREACH table_name IN ARRAY tables
    LOOP
        -- تفعيل RLS
        EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', table_name);
        
        -- Policy للقراءة
        EXECUTE format('
            CREATE POLICY tenant_isolation_policy ON %I
            USING (tenant_id = current_setting(''app.current_tenant'')::uuid)
        ', table_name);
        
        -- Policy للإدراج
        EXECUTE format('
            CREATE POLICY tenant_isolation_insert_policy ON %I
            FOR INSERT
            WITH CHECK (tenant_id = current_setting(''app.current_tenant'')::uuid)
        ', table_name);
        
        -- Policy للتحديث
        EXECUTE format('
            CREATE POLICY tenant_isolation_update_policy ON %I
            FOR UPDATE
            USING (tenant_id = current_setting(''app.current_tenant'')::uuid)
        ', table_name);
        
        -- Policy للحذف
        EXECUTE format('
            CREATE POLICY tenant_isolation_delete_policy ON %I
            FOR DELETE
            USING (tenant_id = current_setting(''app.current_tenant'')::uuid)
        ', table_name);
    END LOOP;
END $$;
```

#### 1.4 Indexes مركبة تبدأ بـ tenant_id

```sql
-- جميع الـ Indexes يجب أن تبدأ بـ tenant_id
CREATE INDEX idx_tariffs_tenant_service ON tariffs(tenant_id, service_id);
CREATE INDEX idx_tariffs_tenant_country ON tariffs(tenant_id, country_code);
CREATE INDEX idx_tariffs_tenant_active ON tariffs(tenant_id, is_active);

CREATE INDEX idx_subscriptions_tenant_status ON subscriptions(tenant_id, status);
CREATE INDEX idx_subscriptions_tenant_renewal ON subscriptions(tenant_id, next_renewal_date);

CREATE INDEX idx_accounting_tenant_created ON accounting_transactions(tenant_id, created_at DESC);
CREATE INDEX idx_accounting_tenant_type ON accounting_transactions(tenant_id, transaction_type);

-- وهكذا لجميع الجداول
```

#### 1.5 Foreign Keys مع Tenant

```sql
-- التأكد من أن جميع العلاقات تحترم tenant_id
-- مثال: عند ربط subscription بـ package
ALTER TABLE subscriptions
ADD CONSTRAINT fk_subscriptions_package_tenant
FOREIGN KEY (tenant_id, package_id)
REFERENCES packages(tenant_id, id);

-- يتطلب إنشاء unique constraint على packages
ALTER TABLE packages
ADD CONSTRAINT uq_packages_tenant_id
UNIQUE (tenant_id, id);
```

### 2. عزل Redis (Cache Isolation)

#### 2.1 Key Prefixing Strategy

```go
// internal/cache/keys.go

type CacheKeyBuilder struct {
    tenantID uuid.UUID
}

func NewCacheKeyBuilder(tenantID uuid.UUID) *CacheKeyBuilder {
    return &CacheKeyBuilder{tenantID: tenantID}
}

// جميع المفاتيح تبدأ بـ tenant_id
func (ckb *CacheKeyBuilder) TariffKey(serviceID int, countryCode, operator string) string {
    return fmt.Sprintf("tenant:%s:tariff:%d:%s:%s", 
        ckb.tenantID.String(), serviceID, countryCode, operator)
}

func (ckb *CacheKeyBuilder) SubscriptionKey(subscriptionID int) string {
    return fmt.Sprintf("tenant:%s:subscription:%d", 
        ckb.tenantID.String(), subscriptionID)
}

func (ckb *CacheKeyBuilder) BalanceKey() string {
    return fmt.Sprintf("tenant:%s:balance", ckb.tenantID.String())
}

func (ckb *CacheKeyBuilder) GatewayStatsKey(gatewayID int, date string) string {
    return fmt.Sprintf("tenant:%s:gateway:%d:stats:%s", 
        ckb.tenantID.String(), gatewayID, date)
}
```

#### 2.2 Redis Namespace Separation

```go
// internal/cache/redis.go

type TenantRedisClient struct {
    client   *redis.Client
    tenantID uuid.UUID
    keyBuilder *CacheKeyBuilder
}

func NewTenantRedisClient(client *redis.Client, tenantID uuid.UUID) *TenantRedisClient {
    return &TenantRedisClient{
        client:     client,
        tenantID:   tenantID,
        keyBuilder: NewCacheKeyBuilder(tenantID),
    }
}

// جميع العمليات تستخدم المفاتيح المعزولة
func (trc *TenantRedisClient) GetTariff(ctx context.Context, serviceID int, countryCode, operator string) (*models.Tariff, error) {
    key := trc.keyBuilder.TariffKey(serviceID, countryCode, operator)
    
    val, err := trc.client.Get(ctx, key).Result()
    if err != nil {
        return nil, err
    }
    
    var tariff models.Tariff
    if err := json.Unmarshal([]byte(val), &tariff); err != nil {
        return nil, err
    }
    
    return &tariff, nil
}

func (trc *TenantRedisClient) SetTariff(ctx context.Context, serviceID int, countryCode, operator string, tariff *models.Tariff, ttl time.Duration) error {
    key := trc.keyBuilder.TariffKey(serviceID, countryCode, operator)
    
    data, err := json.Marshal(tariff)
    if err != nil {
        return err
    }
    
    return trc.client.Set(ctx, key, data, ttl).Err()
}
```

#### 2.3 TTL Management per Tenant

```go
// إدارة TTL منفصلة لكل tenant
func (trc *TenantRedisClient) SetWithTenantTTL(ctx context.Context, key string, value interface{}, baseTTL time.Duration) error {
    // يمكن تخصيص TTL حسب نوع الـ tenant
    // مثلاً: Enterprise tenants قد يحصلون على TTL أطول
    
    tenantTTL := baseTTL
    
    // تخصيص TTL حسب نوع الـ tenant
    if trc.isEnterpriseTenant() {
        tenantTTL = baseTTL * 2
    }
    
    fullKey := trc.keyBuilder.buildKey(key)
    return trc.client.Set(ctx, fullKey, value, tenantTTL).Err()
}
```

### 3. عزل Worker Pools

#### 3.1 Worker Pool per Tenant

```go
// internal/worker/tenant_pool.go

type TenantWorkerPool struct {
    tenantID    uuid.UUID
    workers     []*Worker
    jobQueue    chan Job
    maxWorkers  int
    rateLimiter *rate.Limiter
}

func NewTenantWorkerPool(tenantID uuid.UUID, maxWorkers int, rateLimit int) *TenantWorkerPool {
    return &TenantWorkerPool{
        tenantID:    tenantID,
        workers:     make([]*Worker, 0, maxWorkers),
        jobQueue:    make(chan Job, maxWorkers*10),
        maxWorkers:  maxWorkers,
        rateLimiter: rate.NewLimiter(rate.Limit(rateLimit), rateLimit),
    }
}

func (twp *TenantWorkerPool) Start(ctx context.Context) {
    for i := 0; i < twp.maxWorkers; i++ {
        worker := NewWorker(i, twp.tenantID, twp.jobQueue)
        twp.workers = append(twp.workers, worker)
        go worker.Start(ctx)
    }
}

func (twp *TenantWorkerPool) Submit(ctx context.Context, job Job) error {
    // تطبيق Rate Limiting
    if err := twp.rateLimiter.Wait(ctx); err != nil {
        return fmt.Errorf("rate limit exceeded for tenant %s: %w", twp.tenantID, err)
    }
    
    select {
    case twp.jobQueue <- job:
        return nil
    case <-ctx.Done():
        return ctx.Err()
    default:
        return fmt.Errorf("job queue full for tenant %s", twp.tenantID)
    }
}
```

#### 3.2 Queue Isolation

```go
// internal/queue/tenant_queue.go

type TenantQueueManager struct {
    tenantID uuid.UUID
    client   *river.Client
}

func (tqm *TenantQueueManager) EnqueueJob(ctx context.Context, job river.JobArgs) error {
    // إضافة tenant_id للـ metadata
    metadata := map[string]interface{}{
        "tenant_id": tqm.tenantID.String(),
    }
    
    _, err := tqm.client.Insert(ctx, job, &river.InsertOpts{
        Queue:    fmt.Sprintf("tenant_%s", tqm.tenantID.String()),
        Metadata: metadata,
    })
    
    return err
}

// Worker يعالج فقط jobs من tenant محدد
func (tqm *TenantQueueManager) StartWorker(ctx context.Context) {
    workers := river.NewWorkers()
    
    // تسجيل worker لـ queue محدد
    river.AddWorker(workers, &MessageWorker{
        tenantID: tqm.tenantID,
    })
    
    tqm.client.Start(ctx)
}
```

### 4. عزل API Keys و Authentication

#### 4.1 API Keys مرتبطة بـ Tenant

```go
// internal/auth/api_key.go

type APIKey struct {
    ID           uuid.UUID
    TenantID     uuid.UUID  // مرتبط بـ tenant واحد فقط
    KeyHash      string
    Permissions  []string
    IsActive     bool
    ExpiresAt    *time.Time
}

func (ak *APIKey) ValidateForTenant(tenantID uuid.UUID) error {
    if ak.TenantID != tenantID {
        return fmt.Errorf("API key does not belong to tenant %s", tenantID)
    }
    
    if !ak.IsActive {
        return errors.New("API key is not active")
    }
    
    if ak.ExpiresAt != nil && time.Now().After(*ak.ExpiresAt) {
        return errors.New("API key has expired")
    }
    
    return nil
}
```

#### 4.2 JWT Claims مع tenant_id

```go
// internal/auth/jwt.go

type JWTClaims struct {
    UserID   uuid.UUID `json:"user_id"`
    TenantID uuid.UUID `json:"tenant_id"`  // إلزامي
    Role     string    `json:"role"`
    jwt.RegisteredClaims
}

func GenerateToken(userID, tenantID uuid.UUID, role string) (string, error) {
    claims := JWTClaims{
        UserID:   userID,
        TenantID: tenantID,
        Role:     role,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
        },
    }
    
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(jwtSecret))
}
```

#### 4.3 Middleware Validation

```go
// internal/middleware/tenant_isolation.go

func TenantIsolationMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // استخراج tenant_id من JWT أو API Key
        tenantID, err := extractTenantID(c)
        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid tenant"})
            c.Abort()
            return
        }
        
        // تعيين tenant_id في السياق
        c.Set("tenant_id", tenantID)
        
        // تعيين tenant_id في database session
        db := getDB(c)
        db.Exec("SET app.current_tenant = ?", tenantID.String())
        
        c.Next()
    }
}

func extractTenantID(c *gin.Context) (uuid.UUID, error) {
    // من JWT
    if claims, exists := c.Get("jwt_claims"); exists {
        jwtClaims := claims.(*JWTClaims)
        return jwtClaims.TenantID, nil
    }
    
    // من API Key
    if apiKey, exists := c.Get("api_key"); exists {
        key := apiKey.(*APIKey)
        return key.TenantID, nil
    }
    
    return uuid.Nil, errors.New("tenant_id not found")
}
```

### 5. عزل الموارد (Resource Isolation)

#### 5.1 CPU/Memory Quotas

```go
// internal/resources/limiter.go

type ResourceLimiter struct {
    tenantID     uuid.UUID
    cpuQuota     float64  // vCPU cores
    memoryQuota  int64    // bytes
    currentUsage *ResourceUsage
}

func (rl *ResourceLimiter) CheckCPUQuota(ctx context.Context) error {
    usage := rl.getCurrentCPUUsage()
    
    if usage >= rl.cpuQuota {
        return fmt.Errorf("CPU quota exceeded for tenant %s: %.2f/%.2f", 
            rl.tenantID, usage, rl.cpuQuota)
    }
    
    return nil
}

func (rl *ResourceLimiter) CheckMemoryQuota(ctx context.Context) error {
    usage := rl.getCurrentMemoryUsage()
    
    if usage >= rl.memoryQuota {
        return fmt.Errorf("memory quota exceeded for tenant %s: %d/%d", 
            rl.tenantID, usage, rl.memoryQuota)
    }
    
    return nil
}
```

#### 5.2 Rate Limiting per Tenant

```go
// internal/ratelimit/tenant_limiter.go

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
            "queries":  rate.NewLimiter(rate.Limit(config.QueriesPerSecond), config.QueriesPerSecond),
        },
    }
}

func (trl *TenantRateLimiter) AllowAPI(ctx context.Context) error {
    if !trl.limiters["api"].Allow() {
        return fmt.Errorf("API rate limit exceeded for tenant %s", trl.tenantID)
    }
    return nil
}

func (trl *TenantRateLimiter) AllowMessage(ctx context.Context) error {
    if !trl.limiters["messages"].Allow() {
        return fmt.Errorf("message rate limit exceeded for tenant %s", trl.tenantID)
    }
    return nil
}
```

#### 5.3 Database Connection Pools

```go
// internal/database/tenant_pool.go

type TenantConnectionPool struct {
    tenantID    uuid.UUID
    pool        *pgxpool.Pool
    maxConns    int32
    activeConns atomic.Int32
}

func (tcp *TenantConnectionPool) Acquire(ctx context.Context) (*pgxpool.Conn, error) {
    // فحص الحد الأقصى
    if tcp.activeConns.Load() >= tcp.maxConns {
        return nil, fmt.Errorf("connection pool exhausted for tenant %s", tcp.tenantID)
    }
    
    conn, err := tcp.pool.Acquire(ctx)
    if err != nil {
        return nil, err
    }
    
    // تعيين tenant_id في الجلسة
    _, err = conn.Exec(ctx, "SET app.current_tenant = $1", tcp.tenantID.String())
    if err != nil {
        conn.Release()
        return nil, err
    }
    
    tcp.activeConns.Add(1)
    
    return conn, nil
}

func (tcp *TenantConnectionPool) Release(conn *pgxpool.Conn) {
    tcp.activeConns.Add(-1)
    conn.Release()
}
```

### 6. عزل البيانات المالية

#### 6.1 Balance Isolation

```go
// internal/balance/tenant_manager.go

type TenantBalanceManager struct {
    db       *gorm.DB
    tenantID uuid.UUID
}

func (tbm *TenantBalanceManager) DeductBalance(ctx context.Context, amount float64, reason string, messageID *uuid.UUID) error {
    return tbm.db.Transaction(func(tx *gorm.DB) error {
        // تعيين tenant_id في الجلسة
        tx.Exec("SET app.current_tenant = ?", tbm.tenantID.String())
        
        // Atomic Update مع WHERE condition تتضمن tenant_id
        result := tx.Model(&models.Tenant{}).
            Where("id = ? AND balance >= ?", tbm.tenantID, amount).
            Update("balance", gorm.Expr("balance - ?", amount))
        
        if result.RowsAffected == 0 {
            return ErrInsufficientBalance
        }
        
        // تسجيل في balance_history مع tenant_id
        balanceHistory := &models.BalanceHistory{
            TenantID:   tbm.tenantID,
            MessageID:  messageID,
            Amount:     amount,
            ChangeType: "DEDUCT",
            Reason:     reason,
        }
        
        return tx.Create(balanceHistory).Error
    })
}
```

#### 6.2 Transaction Isolation

```go
// جميع المعاملات المالية يجب أن تتضمن tenant_id
func (tbm *TenantBalanceManager) RecordTransaction(ctx context.Context, tx *AccountingTransaction) error {
    // التأكد من أن tenant_id صحيح
    if tx.TenantID != tbm.tenantID {
        return fmt.Errorf("transaction tenant_id mismatch")
    }
    
    return tbm.db.Create(tx).Error
}
```

### 7. عزل الإحصائيات والتقارير

#### 7.1 Statistics Isolation

```go
// internal/stats/tenant_stats.go

type TenantStatistics struct {
    tenantID uuid.UUID
    db       *gorm.DB
    cache    *redis.Client
}

func (ts *TenantStatistics) GetDailyStats(ctx context.Context, date time.Time) (*DailyStats, error) {
    // جميع الاستعلامات تتضمن tenant_id
    var stats DailyStats
    
    err := ts.db.WithContext(ctx).
        Where("tenant_id = ? AND date = ?", ts.tenantID, date).
        First(&stats).Error
    
    return &stats, err
}

func (ts *TenantStatistics) GetMonthlyRevenue(ctx context.Context, year, month int) (float64, error) {
    var total float64
    
    err := ts.db.WithContext(ctx).
        Model(&models.AccountingTransaction{}).
        Where("tenant_id = ? AND EXTRACT(YEAR FROM created_at) = ? AND EXTRACT(MONTH FROM created_at) = ?", 
            ts.tenantID, year, month).
        Select("COALESCE(SUM(amount), 0)").
        Scan(&total).Error
    
    return total, err
}
```

### 8. عزل الإشعارات

#### 8.1 Notification Queues

```go
// internal/notifications/tenant_notifier.go

type TenantNotifier struct {
    tenantID uuid.UUID
    queue    *NotificationQueue
}

func (tn *TenantNotifier) SendNotification(ctx context.Context, notification *Notification) error {
    // إضافة tenant_id للإشعار
    notification.TenantID = tn.tenantID
    
    // إرسال لـ queue مخصص
    queueName := fmt.Sprintf("notifications:tenant:%s", tn.tenantID.String())
    
    return tn.queue.Enqueue(ctx, queueName, notification)
}
```

#### 8.2 Email Templates per Tenant

```go
// قوالب بريد مخصصة لكل tenant
func (tn *TenantNotifier) GetEmailTemplate(templateName string) (*EmailTemplate, error) {
    // البحث عن قالب مخصص للـ tenant أولاً
    template, err := tn.loadTenantTemplate(templateName)
    if err == nil {
        return template, nil
    }
    
    // استخدام القالب الافتراضي
    return tn.loadDefaultTemplate(templateName)
}
```

### 9. Cross-Tenant Prevention

#### 9.1 Validation Middleware

```go
// internal/middleware/cross_tenant_prevention.go

func PreventCrossTenantAccess() gin.HandlerFunc {
    return func(c *gin.Context) {
        tenantID := c.GetString("tenant_id")
        
        // فحص جميع المعاملات في الطلب
        if resourceTenantID := c.Param("tenant_id"); resourceTenantID != "" {
            if resourceTenantID != tenantID {
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

#### 9.2 Error Responses

```go
// عند محاولة cross-tenant access
{
    "success": false,
    "error": {
        "code": "CROSS_TENANT_ACCESS_DENIED",
        "message": "You do not have permission to access resources from another tenant",
        "details": {
            "your_tenant_id": "123e4567-e89b-12d3-a456-426614174000",
            "requested_tenant_id": "987fcdeb-51a2-43f7-b890-123456789abc"
        }
    }
}
```

### 10. Audit Trail للعزل

```go
// internal/audit/tenant_audit.go

type TenantAuditLogger struct {
    tenantID uuid.UUID
    db       *gorm.DB
}

func (tal *TenantAuditLogger) LogAccess(ctx context.Context, action string, resourceType string, resourceID string) error {
    log := &AuditLog{
        TenantID:     tal.tenantID,
        Action:       action,
        ResourceType: resourceType,
        ResourceID:   resourceID,
        Timestamp:    time.Now(),
        UserID:       getUserIDFromContext(ctx),
        IPAddress:    getIPFromContext(ctx),
    }
    
    return tal.db.Create(log).Error
}

// تسجيل محاولات cross-tenant access
func (tal *TenantAuditLogger) LogCrossTenantAttempt(ctx context.Context, targetTenantID uuid.UUID) error {
    log := &SecurityLog{
        TenantID:       tal.tenantID,
        EventType:      "CROSS_TENANT_ACCESS_ATTEMPT",
        TargetTenantID: targetTenantID,
        Timestamp:      time.Now(),
        UserID:         getUserIDFromContext(ctx),
        IPAddress:      getIPFromContext(ctx),
        Blocked:        true,
    }
    
    return tal.db.Create(log).Error
}
```

---

## تصميم قاعدة البيانات

### المخطط العام (Schema Overview)

```
قاعدة البيانات الرئيسية: sms_gateway

الجداول الأساسية:
├── services (الخدمات الثلاثة)
├── tariffs (التعرفات)
├── packages (الباقات الشهرية)
├── subscriptions (اشتراكات العملاء)
├── rollover_messages (الرسائل المرحّلة)
├── dedicated_gateways (البوابات المخصصة)
├── resource_quotas (الحصص الموردية)
├── resource_usage (الاستخدام الفعلي)
├── gateway_users (مستخدمو البوابات)
├── gateway_api_keys (مفاتيح API)
├── gateway_statistics (إحصائيات البوابات)
├── accounting_transactions (المعاملات المالية)
├── balance_history (سجل تغييرات الرصيد)
├── segmentation_rules (قواعد تقسيم الرسائل)
├── currencies (العملات)
├── exchange_rates (أسعار الصرف)
└── billing_invoices (الفواتير)
```

### الجداول التفصيلية

#### 1. جدول الخدمات (services)

```sql
CREATE TABLE services (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    code VARCHAR(50) UNIQUE NOT NULL,
    type VARCHAR(50) NOT NULL, -- 'gateway', 'mobile', 'dedicated'
    description TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_services_type ON services(type);
CREATE INDEX idx_services_active ON services(is_active);

-- البيانات الأولية
INSERT INTO services (name, code, type, description) VALUES
('خدمة بوابة الهدهد', 'HUDUD_GATEWAY', 'gateway', 'إرسال الرسائل عبر بوابة الهدهد بنموذج الدفع حسب الاستخدام'),
('خدمة الإرسال عبر الهاتف', 'MOBILE_GATEWAY', 'mobile', 'إرسال الرسائل عبر الهاتف بنموذج الاشتراك الشهري'),
('خدمة إدارة البوابات', 'DEDICATED_GATEWAY', 'dedicated', 'بوابات مخصصة للشركات المؤسسية');
```

#### 2. جدول التعرفات (tariffs)

```sql
CREATE TABLE tariffs (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),  -- إضافة للعزل
    service_id INTEGER NOT NULL REFERENCES services(id),
    country_code VARCHAR(2) NOT NULL, -- 'YE', 'SA', etc.
    network_operator VARCHAR(100), -- 'Yemen Mobile', 'Sabafon', etc.
    message_type VARCHAR(50) DEFAULT 'standard', -- 'standard', 'unicode'
    cost_per_message DECIMAL(10, 4) NOT NULL,
    currency VARCHAR(3) DEFAULT 'YER', -- 'YER', 'USD', 'SAR'
    effective_from DATE NOT NULL,
    effective_to DATE,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT check_cost_positive CHECK (cost_per_message > 0)
);

-- Indexes مركبة تبدأ بـ tenant_id
CREATE INDEX idx_tariffs_tenant_service ON tariffs(tenant_id, service_id);
CREATE INDEX idx_tariffs_tenant_country ON tariffs(tenant_id, country_code);
CREATE INDEX idx_tariffs_tenant_operator ON tariffs(tenant_id, network_operator);
CREATE INDEX idx_tariffs_tenant_active ON tariffs(tenant_id, is_active);
CREATE INDEX idx_tariffs_tenant_effective ON tariffs(tenant_id, effective_from, effective_to);

-- تفعيل Row Level Security
ALTER TABLE tariffs ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON tariffs
    USING (tenant_id = current_setting('app.current_tenant')::uuid);

CREATE POLICY tenant_isolation_insert_policy ON tariffs
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant')::uuid);

-- البيانات الأولية (بوابة الهدهد)
INSERT INTO tariffs (service_id, country_code, network_operator, cost_per_message, currency, effective_from) VALUES
(1, 'YE', 'Yemen Mobile', 9.0000, 'YER', CURRENT_DATE),
(1, 'YE', 'Other', 13.0000, 'YER', CURRENT_DATE);
```

#### 3. جدول الباقات الشهرية (packages)

```sql
CREATE TABLE packages (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),  -- إضافة للعزل
    service_id INTEGER NOT NULL REFERENCES services(id),
    name VARCHAR(255) NOT NULL,
    code VARCHAR(50) NOT NULL,
    price_usd DECIMAL(10, 2) NOT NULL,
    included_messages INTEGER, -- NULL = unlimited
    overage_rate DECIMAL(10, 4), -- السعر لكل رسالة إضافية
    is_unlimited BOOLEAN DEFAULT false,
    billing_cycle VARCHAR(20) DEFAULT 'monthly', -- 'monthly', 'yearly'
    
    -- إعدادات Rollover
    rollover_enabled BOOLEAN DEFAULT false,
    rollover_limit INTEGER, -- NULL = غير محدود
    rollover_expiry_months INTEGER, -- NULL = لا تنتهي
    
    -- الحالة
    is_active BOOLEAN DEFAULT true,
    display_order INTEGER DEFAULT 0,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT check_price_positive CHECK (price_usd > 0),
    CONSTRAINT check_messages_positive CHECK (included_messages IS NULL OR included_messages > 0),
    CONSTRAINT uq_packages_tenant_code UNIQUE (tenant_id, code)
);

-- Indexes مركبة تبدأ بـ tenant_id
CREATE INDEX idx_packages_tenant_service ON packages(tenant_id, service_id);
CREATE INDEX idx_packages_tenant_active ON packages(tenant_id, is_active);
CREATE INDEX idx_packages_tenant_order ON packages(tenant_id, display_order);

-- تفعيل Row Level Security
ALTER TABLE packages ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON packages
    USING (tenant_id = current_setting('app.current_tenant')::uuid);

CREATE POLICY tenant_isolation_insert_policy ON packages
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant')::uuid);

-- البيانات الأولية
INSERT INTO packages (service_id, name, code, price_usd, included_messages, overage_rate, is_unlimited, rollover_enabled, display_order) VALUES
(2, 'Basic', 'PKG_BASIC', 10.00, 1000, 0.0100, false, true, 1),
(2, 'Standard', 'PKG_STANDARD', 20.00, 3000, 0.0067, false, true, 2),
(2, 'Professional', 'PKG_PROFESSIONAL', 40.00, 8000, 0.0050, false, true, 3),
(2, 'Business', 'PKG_BUSINESS', 50.00, 15000, 0.0033, false, true, 4),
(2, 'Enterprise', 'PKG_ENTERPRISE', 100.00, NULL, 0.0000, true, false, 5);
```

#### 4. جدول اشتراكات العملاء (subscriptions)

```sql
CREATE TABLE subscriptions (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    package_id INTEGER NOT NULL REFERENCES packages(id),
    
    -- فترة الاشتراك
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    
    -- الاستخدام
    messages_used INTEGER DEFAULT 0,
    messages_remaining INTEGER,
    overage_messages INTEGER DEFAULT 0,
    overage_cost DECIMAL(10, 2) DEFAULT 0.00,
    
    -- الحالة
    status VARCHAR(20) DEFAULT 'active', -- 'active', 'expired', 'cancelled', 'suspended'
    auto_renew BOOLEAN DEFAULT true,
    grace_period_days INTEGER DEFAULT 3,
    
    -- التجديد
    last_renewed_at TIMESTAMP,
    next_renewal_date DATE,
    renewal_attempts INTEGER DEFAULT 0,
    last_renewal_attempt TIMESTAMP,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT check_messages_non_negative CHECK (messages_used >= 0),
    CONSTRAINT check_overage_non_negative CHECK (overage_messages >= 0)
);

CREATE INDEX idx_subscriptions_tenant ON subscriptions(tenant_id);
CREATE INDEX idx_subscriptions_package ON subscriptions(package_id);
CREATE INDEX idx_subscriptions_status ON subscriptions(status);
CREATE INDEX idx_subscriptions_renewal ON subscriptions(next_renewal_date);
CREATE INDEX idx_subscriptions_dates ON subscriptions(start_date, end_date);
```


#### 5. جدول الرسائل المرحّلة (rollover_messages)

```sql
CREATE TABLE rollover_messages (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),  -- إضافة للعزل
    subscription_id INTEGER NOT NULL REFERENCES subscriptions(id),
    
    -- تفاصيل الترحيل
    messages_count INTEGER NOT NULL,
    source_month DATE NOT NULL,
    expiry_date DATE,
    
    -- الاستخدام
    messages_used INTEGER DEFAULT 0,
    messages_remaining INTEGER NOT NULL,
    
    -- الحالة والترحيل المشروط
    status VARCHAR(20) DEFAULT 'pending', -- 'pending', 'active', 'expired', 'consumed', 'forfeited'
    payment_status VARCHAR(20), -- 'pending', 'success', 'failed' - لتتبع حالة الدفع
    payment_verified_at TIMESTAMP, -- تاريخ التحقق من الدفع
    forfeited_reason VARCHAR(255), -- سبب المصادرة (مثل: 'payment_failed')
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT check_rollover_positive CHECK (messages_count > 0),
    CONSTRAINT check_rollover_remaining CHECK (messages_remaining >= 0),
    CONSTRAINT check_rollover_status_valid CHECK (status IN ('pending', 'active', 'expired', 'consumed', 'forfeited')),
    CONSTRAINT check_payment_status_valid CHECK (payment_status IN ('pending', 'success', 'failed') OR payment_status IS NULL)
);

-- Indexes مركبة تبدأ بـ tenant_id
CREATE INDEX idx_rollover_tenant_subscription ON rollover_messages(tenant_id, subscription_id);
CREATE INDEX idx_rollover_tenant_status ON rollover_messages(tenant_id, status);
CREATE INDEX idx_rollover_tenant_expiry ON rollover_messages(tenant_id, expiry_date);
CREATE INDEX idx_rollover_tenant_payment ON rollover_messages(tenant_id, payment_status);

-- تفعيل Row Level Security
ALTER TABLE rollover_messages ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON rollover_messages
    USING (tenant_id = current_setting('app.current_tenant')::uuid);

CREATE POLICY tenant_isolation_insert_policy ON rollover_messages
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant')::uuid);
```

**ملاحظات مهمة حول الترحيل المشروط:**
- عند نهاية الشهر، يتم إنشاء سجل rollover بحالة 'pending' و payment_status = 'pending'
- عند نجاح الدفع التلقائي، يتم تحديث status إلى 'active' و payment_status إلى 'success'
- عند فشل الدفع التلقائي، يتم تحديث status إلى 'forfeited' و payment_status إلى 'failed'
- الرسائل المرحّلة بحالة 'forfeited' لا يمكن استخدامها ويتم حذفها
- يمكن تعطيل هذه السياسة من واجهة الإدارة للعودة للسلوك القديم (ترحيل مباشر)
```

#### 6. جدول البوابات المخصصة (dedicated_gateways)

```sql
CREATE TABLE dedicated_gateways (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    
    -- معلومات البوابة
    gateway_name VARCHAR(255) NOT NULL,
    gateway_code VARCHAR(50) UNIQUE NOT NULL,
    description TEXT,
    
    -- معلومات الاتصال بشركة الاتصالات
    telecom_provider VARCHAR(100),
    telecom_api_endpoint TEXT,
    telecom_username VARCHAR(255),
    telecom_password_encrypted TEXT,
    telecom_api_key_encrypted TEXT,
    
    -- معلومات السيرفر
    server_type VARCHAR(50), -- 'dedicated', 'vps', 'container', 'shared'
    server_ip VARCHAR(45),
    server_hostname VARCHAR(255),
    server_specs JSONB, -- {cpu: 4, ram: 8GB, storage: 100GB}
    
    -- الحدود
    max_throughput INTEGER, -- رسائل/ثانية
    daily_limit INTEGER,
    monthly_limit INTEGER,
    
    -- نموذج الاشتراك (الشركة تدفع لشركة الاتصالات مباشرة، المنصة تتقاضى رسوم الاشتراك والإيجار فقط)
    subscription_type VARCHAR(20) DEFAULT 'monthly', -- 'monthly', 'yearly'
    subscription_fee DECIMAL(10, 2), -- رسوم الاشتراك الشهري/السنوي للمنصة
    server_rental_fee DECIMAL(10, 2), -- رسوم إيجار السيرفر الشهرية
    setup_fee DECIMAL(10, 2), -- رسوم الإعداد الأولية
    billing_cycle VARCHAR(20) DEFAULT 'monthly',
    
    -- مراقبة الرصيد (للقراءة فقط - الشركة تدفع مباشرة لشركة الاتصالات)
    telecom_balance DECIMAL(10, 2), -- رصيد الشركة لدى شركة الاتصالات (للمراقبة فقط)
    balance_alert_threshold DECIMAL(10, 2), -- عتبة التنبيه عند انخفاض الرصيد
    last_balance_check TIMESTAMP, -- آخر فحص للرصيد
    
    -- العقد
    contract_start_date DATE NOT NULL,
    contract_end_date DATE,
    
    -- الحالة
    status VARCHAR(20) DEFAULT 'active', -- 'active', 'suspended', 'maintenance', 'terminated'
    last_health_check TIMESTAMP,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_dedicated_gateways_tenant ON dedicated_gateways(tenant_id);
CREATE INDEX idx_dedicated_gateways_code ON dedicated_gateways(gateway_code);
CREATE INDEX idx_dedicated_gateways_status ON dedicated_gateways(status);
```


#### 7. جدول الحصص الموردية (resource_quotas)

```sql
CREATE TABLE resource_quotas (
    id SERIAL PRIMARY KEY,
    gateway_id INTEGER NOT NULL REFERENCES dedicated_gateways(id),
    
    -- CPU & Memory
    cpu_cores DECIMAL(4, 2), -- 2.00 = 2 vCPU
    memory_mb INTEGER, -- 4096 = 4 GB
    
    -- Network
    bandwidth_mbps INTEGER,
    max_concurrent_connections INTEGER,
    
    -- Storage
    storage_gb INTEGER,
    
    -- Database
    max_db_connections INTEGER,
    max_queries_per_second INTEGER,
    
    -- Redis/Cache
    redis_memory_mb INTEGER,
    redis_max_keys INTEGER,
    
    -- Message Processing
    max_messages_per_hour INTEGER,
    max_messages_per_day INTEGER,
    max_queue_depth INTEGER,
    
    -- API
    api_rate_limit_per_minute INTEGER,
    api_rate_limit_per_hour INTEGER,
    
    -- Workers
    max_worker_threads INTEGER,
    
    -- الحالة
    is_active BOOLEAN DEFAULT true,
    enforcement_mode VARCHAR(20) DEFAULT 'soft', -- 'soft', 'hard'
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(gateway_id)
);

CREATE INDEX idx_resource_quotas_gateway ON resource_quotas(gateway_id);
```

#### 8. جدول الاستخدام الفعلي (resource_usage)

```sql
CREATE TABLE resource_usage (
    id SERIAL PRIMARY KEY,
    gateway_id INTEGER NOT NULL REFERENCES dedicated_gateways(id),
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- CPU & Memory
    cpu_usage_percent DECIMAL(5, 2),
    memory_usage_mb INTEGER,
    
    -- Network
    bandwidth_usage_mbps DECIMAL(10, 2),
    active_connections INTEGER,
    
    -- Database
    db_connections_active INTEGER,
    queries_per_second INTEGER,
    
    -- Redis
    redis_memory_usage_mb INTEGER,
    redis_keys_count INTEGER,
    
    -- Messages
    messages_processed_last_hour INTEGER,
    messages_processed_today INTEGER,
    current_queue_depth INTEGER,
    
    -- API
    api_requests_last_minute INTEGER,
    api_requests_last_hour INTEGER,
    
    -- Workers
    active_worker_threads INTEGER,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_resource_usage_gateway_time ON resource_usage(gateway_id, timestamp DESC);
-- Partition by month for better performance
CREATE INDEX idx_resource_usage_timestamp ON resource_usage(timestamp DESC);
```


#### 9. جدول مستخدمي البوابات (gateway_users)

```sql
CREATE TABLE gateway_users (
    id SERIAL PRIMARY KEY,
    gateway_id INTEGER NOT NULL REFERENCES dedicated_gateways(id),
    user_id UUID NOT NULL REFERENCES users(id),
    
    -- الصلاحيات
    role VARCHAR(50) NOT NULL, -- 'admin', 'operator', 'viewer'
    permissions JSONB, -- {send_messages: true, view_reports: true, manage_users: false}
    
    -- الحالة
    is_active BOOLEAN DEFAULT true,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(gateway_id, user_id)
);

CREATE INDEX idx_gateway_users_gateway ON gateway_users(gateway_id);
CREATE INDEX idx_gateway_users_user ON gateway_users(user_id);
CREATE INDEX idx_gateway_users_role ON gateway_users(role);
```

#### 10. جدول مفاتيح API للبوابات (gateway_api_keys)

```sql
CREATE TABLE gateway_api_keys (
    id SERIAL PRIMARY KEY,
    gateway_id INTEGER NOT NULL REFERENCES dedicated_gateways(id),
    
    -- معلومات المفتاح
    key_name VARCHAR(255) NOT NULL,
    api_key_hash VARCHAR(255) NOT NULL UNIQUE,
    api_secret_hash VARCHAR(255),
    
    -- الصلاحيات
    permissions JSONB, -- {send_sms: true, view_stats: true, manage_gateway: false}
    rate_limit INTEGER, -- requests per minute
    
    -- IP Whitelisting
    allowed_ips TEXT[], -- ['192.168.1.1', '10.0.0.0/24']
    
    -- الحالة
    is_active BOOLEAN DEFAULT true,
    expires_at TIMESTAMP,
    last_used_at TIMESTAMP,
    usage_count INTEGER DEFAULT 0,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_gateway_api_keys_gateway ON gateway_api_keys(gateway_id);
CREATE INDEX idx_gateway_api_keys_hash ON gateway_api_keys(api_key_hash);
CREATE INDEX idx_gateway_api_keys_active ON gateway_api_keys(is_active);
```

#### 11. جدول إحصائيات البوابات (gateway_statistics)

```sql
CREATE TABLE gateway_statistics (
    id SERIAL PRIMARY KEY,
    gateway_id INTEGER NOT NULL REFERENCES dedicated_gateways(id),
    date DATE NOT NULL,
    
    -- إحصائيات الرسائل
    messages_sent INTEGER DEFAULT 0,
    messages_delivered INTEGER DEFAULT 0,
    messages_failed INTEGER DEFAULT 0,
    messages_pending INTEGER DEFAULT 0,
    
    -- إحصائيات الأداء
    avg_response_time_ms INTEGER,
    max_response_time_ms INTEGER,
    min_response_time_ms INTEGER,
    uptime_percentage DECIMAL(5, 2),
    downtime_minutes INTEGER DEFAULT 0,
    
    -- إحصائيات مالية
    total_cost DECIMAL(10, 2),
    total_revenue DECIMAL(10, 2),
    profit_margin DECIMAL(10, 2),
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(gateway_id, date)
);

CREATE INDEX idx_gateway_stats_gateway_date ON gateway_statistics(gateway_id, date DESC);
CREATE INDEX idx_gateway_stats_date ON gateway_statistics(date DESC);
```


#### 12. جدول المعاملات المالية المحسّن (accounting_transactions)

```sql
-- تحديث الجدول الحالي لمعالجة المشاكل الحرجة
CREATE TABLE accounting_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    message_id UUID REFERENCES messages(id),
    
    -- نوع المعاملة
    transaction_type VARCHAR(50) NOT NULL, -- 'deduction', 'refund', 'topup', 'subscription', 'overage'
    
    -- المبالغ
    amount DECIMAL(10, 4) NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',
    
    -- الرصيد
    balance_before DECIMAL(10, 4) NOT NULL,
    balance_after DECIMAL(10, 4) NOT NULL,
    
    -- الحالة
    status VARCHAR(20) DEFAULT 'pending', -- 'pending', 'completed', 'refunded', 'failed', 'cancelled'
    
    -- معلومات إضافية
    description TEXT,
    metadata JSONB, -- {service_id: 1, package_id: 2, gateway_id: 3}
    
    -- التدقيق
    created_by UUID REFERENCES users(id),
    refunded_at TIMESTAMP,
    refund_reason TEXT,
    
    -- Idempotency
    idempotency_key VARCHAR(255) UNIQUE,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT check_amount_positive CHECK (amount > 0)
);

CREATE INDEX idx_accounting_tenant ON accounting_transactions(tenant_id);
CREATE INDEX idx_accounting_message ON accounting_transactions(message_id);
CREATE INDEX idx_accounting_type ON accounting_transactions(transaction_type);
CREATE INDEX idx_accounting_status ON accounting_transactions(status);
CREATE INDEX idx_accounting_created ON accounting_transactions(created_at DESC);
CREATE INDEX idx_accounting_idempotency ON accounting_transactions(idempotency_key);
```

#### 13. جدول سجل تغييرات الرصيد (balance_history)

```sql
-- جدول جديد لحل مشكلة عدم وجود Audit Trail
CREATE TABLE balance_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    message_id UUID REFERENCES messages(id),
    transaction_id UUID REFERENCES accounting_transactions(id),
    
    -- التغيير
    old_balance DECIMAL(10, 4) NOT NULL,
    new_balance DECIMAL(10, 4) NOT NULL,
    amount DECIMAL(10, 4) NOT NULL,
    change_type VARCHAR(50) NOT NULL, -- 'DEDUCT', 'REFUND', 'TOPUP', 'SUBSCRIPTION'
    
    -- السبب
    reason TEXT,
    
    -- التدقيق
    created_by VARCHAR(255), -- 'system', 'admin:user_id', 'api:api_key_id'
    ip_address INET,
    user_agent TEXT,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_balance_history_tenant ON balance_history(tenant_id);
CREATE INDEX idx_balance_history_message ON balance_history(message_id);
CREATE INDEX idx_balance_history_transaction ON balance_history(transaction_id);
CREATE INDEX idx_balance_history_created ON balance_history(created_at DESC);
CREATE INDEX idx_balance_history_type ON balance_history(change_type);
```


#### 14. جدول قواعد تقسيم الرسائل (segmentation_rules)

```sql
CREATE TABLE segmentation_rules (
    id SERIAL PRIMARY KEY,
    gateway_id INTEGER REFERENCES dedicated_gateways(id), -- NULL = default
    country_code VARCHAR(2),
    network_operator VARCHAR(100),
    
    -- نوع الترميز
    encoding_type VARCHAR(20) NOT NULL, -- 'gsm7', 'unicode'
    
    -- قواعد التقسيم
    segment_1_chars INTEGER NOT NULL, -- 70 للجزء الأول
    segment_2_chars INTEGER NOT NULL, -- 65 للجزء الثاني
    segment_3_chars INTEGER NOT NULL, -- 62 للجزء الثالث
    segment_n_chars INTEGER NOT NULL, -- للأجزاء التالية
    
    -- معلومات إضافية
    header_overhead INTEGER DEFAULT 0, -- عدد الأحرف المستخدمة للترويسة
    description TEXT,
    
    -- الحالة
    is_active BOOLEAN DEFAULT true,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_segmentation_gateway ON segmentation_rules(gateway_id);
CREATE INDEX idx_segmentation_country ON segmentation_rules(country_code);
CREATE INDEX idx_segmentation_encoding ON segmentation_rules(encoding_type);

-- القواعد الافتراضية
INSERT INTO segmentation_rules (gateway_id, country_code, network_operator, encoding_type, segment_1_chars, segment_2_chars, segment_3_chars, segment_n_chars) VALUES
(NULL, NULL, NULL, 'gsm7', 70, 65, 62, 62),
(NULL, NULL, NULL, 'unicode', 70, 65, 62, 62);
```

#### 15. جدول العملات (currencies)

```sql
CREATE TABLE currencies (
    id SERIAL PRIMARY KEY,
    code VARCHAR(3) UNIQUE NOT NULL, -- 'USD', 'YER', 'SAR'
    name VARCHAR(100) NOT NULL,
    symbol VARCHAR(10),
    decimal_places INTEGER DEFAULT 2,
    is_active BOOLEAN DEFAULT true,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_currencies_code ON currencies(code);

-- البيانات الأولية
INSERT INTO currencies (code, name, symbol, decimal_places) VALUES
('USD', 'US Dollar', '$', 2),
('YER', 'Yemeni Rial', 'ر.ي', 2),
('SAR', 'Saudi Riyal', 'ر.س', 2);
```

#### 16. جدول أسعار الصرف (exchange_rates)

```sql
CREATE TABLE exchange_rates (
    id SERIAL PRIMARY KEY,
    from_currency VARCHAR(3) NOT NULL REFERENCES currencies(code),
    to_currency VARCHAR(3) NOT NULL REFERENCES currencies(code),
    rate DECIMAL(15, 6) NOT NULL,
    effective_from TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    effective_to TIMESTAMP,
    source VARCHAR(100), -- 'manual', 'api', 'central_bank'
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT check_rate_positive CHECK (rate > 0),
    CONSTRAINT check_different_currencies CHECK (from_currency != to_currency)
);

CREATE INDEX idx_exchange_rates_from ON exchange_rates(from_currency);
CREATE INDEX idx_exchange_rates_to ON exchange_rates(to_currency);
CREATE INDEX idx_exchange_rates_effective ON exchange_rates(effective_from, effective_to);

-- أسعار الصرف الأولية (تقريبية)
INSERT INTO exchange_rates (from_currency, to_currency, rate, effective_from) VALUES
('USD', 'YER', 250.0000, CURRENT_TIMESTAMP),
('YER', 'USD', 0.0040, CURRENT_TIMESTAMP);
```


#### 17. جدول الفواتير (billing_invoices)

```sql
CREATE TABLE billing_invoices (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    invoice_number VARCHAR(50) UNIQUE NOT NULL,
    
    -- نوع الفاتورة
    invoice_type VARCHAR(50) NOT NULL, -- 'subscription', 'topup', 'overage', 'gateway_rental'
    
    -- المبالغ
    subtotal DECIMAL(10, 2) NOT NULL,
    tax_rate DECIMAL(5, 2) DEFAULT 0.00,
    tax_amount DECIMAL(10, 2) DEFAULT 0.00,
    total_amount DECIMAL(10, 2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',
    
    -- التفاصيل
    line_items JSONB, -- [{description: "Basic Package", quantity: 1, unit_price: 10.00, total: 10.00}]
    
    -- التواريخ
    issue_date DATE NOT NULL,
    due_date DATE,
    paid_date DATE,
    
    -- الحالة
    status VARCHAR(20) DEFAULT 'pending', -- 'pending', 'paid', 'overdue', 'cancelled'
    
    -- معلومات الدفع
    payment_method VARCHAR(50),
    payment_reference VARCHAR(255),
    
    -- ملاحظات
    notes TEXT,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT check_amounts_positive CHECK (subtotal >= 0 AND total_amount >= 0)
);

CREATE INDEX idx_invoices_tenant ON billing_invoices(tenant_id);
CREATE INDEX idx_invoices_number ON billing_invoices(invoice_number);
CREATE INDEX idx_invoices_status ON billing_invoices(status);
CREATE INDEX idx_invoices_dates ON billing_invoices(issue_date, due_date);
```

---

## معالجة المشاكل الحرجة في النظام الحالي

### المشكلة 1: Race Condition في خصم الرصيد

#### الحل المقترح: Atomic Balance Deduction

```go
// internal/balance/manager.go

type BalanceManager struct {
    db *gorm.DB
}

// DeductBalance - خصم رصيد بطريقة آمنة (Atomic)
func (bm *BalanceManager) DeductBalance(ctx context.Context, tenantID uuid.UUID, amount float64, reason string, messageID *uuid.UUID) error {
    return bm.db.Transaction(func(tx *gorm.DB) error {
        // 1. Set Transaction Isolation Level
        tx.Exec("SET TRANSACTION ISOLATION LEVEL SERIALIZABLE")
        
        // 2. Atomic Update مع WHERE condition
        result := tx.Model(&models.Tenant{}).
            Where("id = ? AND balance >= ?", tenantID, amount).
            Update("balance", gorm.Expr("balance - ?", amount))
        
        if result.RowsAffected == 0 {
            return ErrInsufficientBalance
        }
        
        // 3. Get updated balance for audit trail
        var tenant models.Tenant
        if err := tx.First(&tenant, "id = ?", tenantID).Error; err != nil {
            return err
        }
        
        // 4. Log balance change
        balanceHistory := &models.BalanceHistory{
            TenantID:    tenantID,
            MessageID:   messageID,
            OldBalance:  tenant.Balance + amount,
            NewBalance:  tenant.Balance,
            Amount:      amount,
            ChangeType:  "DEDUCT",
            Reason:      reason,
            CreatedBy:   "system",
        }
        
        if err := tx.Create(balanceHistory).Error; err != nil {
            return err
        }
        
        return nil
    })
}
```


### المشكلة 2: عدم وجود Idempotency في Refund

#### الحل المقترح: Idempotent Refund

```go
// internal/balance/manager.go

// RefundBalance - استرجاع رصيد بطريقة آمنة (Idempotent)
func (bm *BalanceManager) RefundBalance(ctx context.Context, messageID uuid.UUID, reason string) error {
    return bm.db.Transaction(func(tx *gorm.DB) error {
        // 1. Set Transaction Isolation Level
        tx.Exec("SET TRANSACTION ISOLATION LEVEL SERIALIZABLE")
        
        // 2. Get transaction details
        var transaction models.AccountingTransaction
        if err := tx.Where("message_id = ?", messageID).First(&transaction).Error; err != nil {
            return err
        }
        
        // 3. Check if already refunded (Idempotency)
        if transaction.Status == models.TxRefunded {
            return nil // Already refunded, skip
        }
        
        // 4. Update transaction status atomically
        result := tx.Model(&models.AccountingTransaction{}).
            Where("message_id = ? AND status = ?", messageID, models.TxPending).
            Updates(map[string]interface{}{
                "status":        models.TxRefunded,
                "refunded_at":   time.Now(),
                "refund_reason": reason,
            })
        
        if result.RowsAffected == 0 {
            return nil // Already refunded by another process
        }
        
        // 5. Atomic Refund
        result = tx.Model(&models.Tenant{}).
            Where("id = ?", transaction.TenantID).
            Update("balance", gorm.Expr("balance + ?", transaction.Amount))
        
        if result.RowsAffected == 0 {
            return ErrTenantNotFound
        }
        
        // 6. Get updated balance for audit trail
        var tenant models.Tenant
        if err := tx.First(&tenant, "id = ?", transaction.TenantID).Error; err != nil {
            return err
        }
        
        // 7. Log balance change
        balanceHistory := &models.BalanceHistory{
            TenantID:      transaction.TenantID,
            MessageID:     &messageID,
            TransactionID: &transaction.ID,
            OldBalance:    tenant.Balance - transaction.Amount,
            NewBalance:    tenant.Balance,
            Amount:        transaction.Amount,
            ChangeType:    "REFUND",
            Reason:        reason,
            CreatedBy:     "system",
        }
        
        if err := tx.Create(balanceHistory).Error; err != nil {
            return err
        }
        
        return nil
    })
}
```

### المشكلة 3: عدم وجود Transaction Timeout

#### الحل المقترح: Context with Timeout

```go
// internal/balance/manager.go

// DeductBalanceWithTimeout - خصم رصيد مع Timeout
func (bm *BalanceManager) DeductBalanceWithTimeout(tenantID uuid.UUID, amount float64, reason string, messageID *uuid.UUID) error {
    // Create context with 5 second timeout
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    return bm.DeductBalance(ctx, tenantID, amount, reason, messageID)
}
```


### المشكلة 4: Fixed Cost (1.0) - نظام التعرفات الديناميكي

#### الحل المقترح: Tariff Engine

```go
// internal/tariff/engine.go

type TariffEngine struct {
    db    *gorm.DB
    cache *redis.Client
}

// CalculateCost - حساب تكلفة الرسالة بناءً على التعرفات
func (te *TariffEngine) CalculateCost(ctx context.Context, serviceID int, countryCode, networkOperator, messageType string, messageLength int) (float64, error) {
    // 1. Get segmentation rules
    segments := te.CalculateSegments(messageLength, messageType)
    
    // 2. Get tariff (with caching)
    cacheKey := fmt.Sprintf("tariff:%d:%s:%s:%s", serviceID, countryCode, networkOperator, messageType)
    
    var tariff models.Tariff
    
    // Try cache first
    cached, err := te.cache.Get(ctx, cacheKey).Result()
    if err == nil {
        json.Unmarshal([]byte(cached), &tariff)
    } else {
        // Get from database
        err = te.db.Where(`
            service_id = ? AND 
            country_code = ? AND 
            network_operator = ? AND 
            message_type = ? AND 
            is_active = true AND 
            effective_from <= CURRENT_DATE AND 
            (effective_to IS NULL OR effective_to >= CURRENT_DATE)
        `, serviceID, countryCode, networkOperator, messageType).
            First(&tariff).Error
        
        if err != nil {
            return 0, ErrTariffNotFound
        }
        
        // Cache for 1 hour
        tariffJSON, _ := json.Marshal(tariff)
        te.cache.Set(ctx, cacheKey, tariffJSON, time.Hour)
    }
    
    // 3. Calculate total cost
    totalCost := float64(segments) * tariff.CostPerMessage
    
    return totalCost, nil
}

// CalculateSegments - حساب عدد أجزاء الرسالة
func (te *TariffEngine) CalculateSegments(messageLength int, messageType string) int {
    // Get segmentation rules (simplified)
    var rules models.SegmentationRules
    te.db.Where("encoding_type = ? AND is_active = true", messageType).First(&rules)
    
    if messageLength <= rules.Segment1Chars {
        return 1
    }
    
    remaining := messageLength - rules.Segment1Chars
    segments := 1
    
    if remaining <= rules.Segment2Chars {
        return 2
    }
    
    remaining -= rules.Segment2Chars
    segments = 2
    
    if remaining <= rules.Segment3Chars {
        return 3
    }
    
    remaining -= rules.Segment3Chars
    segments = 3
    
    // Calculate remaining segments
    segments += int(math.Ceil(float64(remaining) / float64(rules.SegmentNChars)))
    
    return segments
}
```


---

## Correctness Properties

*الخاصية (Property) هي سمة أو سلوك يجب أن يكون صحيحاً عبر جميع عمليات التنفيذ الصالحة للنظام - بشكل أساسي، بيان رسمي حول ما يجب أن يفعله النظام. الخصائص تعمل كجسر بين المواصفات المقروءة للإنسان وضمانات الصحة القابلة للتحقق آلياً.*

### Property 1: تحديث التعرفة يحفظ التاريخ

*لأي* تعرفة موجودة، عند تحديثها بقيمة جديدة، يجب أن يحفظ النظام تاريخ السريان (effective_from) للتعرفة الجديدة

**Validates: Requirements 2.3**

### Property 2: الاحتفاظ بالسجل التاريخي للتعرفات

*لأي* تعرفة يتم تحديثها، يجب أن تبقى النسخة القديمة في قاعدة البيانات مع تحديد تاريخ الانتهاء (effective_to)

**Validates: Requirements 2.4**

### Property 3: استخدام التعرفة النشطة الصحيحة

*لأي* رسالة مع شبكة محددة، عند حساب التكلفة، يجب أن يستخدم النظام التعرفة النشطة (is_active = true AND effective_from <= CURRENT_DATE AND (effective_to IS NULL OR effective_to >= CURRENT_DATE)) المطابقة للشبكة

**Validates: Requirements 2.5**

### Property 4: حساب سعر الرسالة الواحدة تلقائياً

*لأي* باقة غير محدودة (is_unlimited = false)، يجب أن يكون سعر الرسالة الواحدة = price_usd / included_messages

**Validates: Requirements 3.8**

### Property 5: الاشتراك الآمن في الباقة

*لأي* باقة وعميل، عند محاولة الاشتراك، إذا كان الرصيد >= سعر الباقة، يجب أن ينجح الاشتراك ويخصم السعر من الرصيد بشكل ذري (atomic)، وإذا كان الرصيد < سعر الباقة، يجب أن يفشل الاشتراك دون تغيير الرصيد

**Validates: Requirements 4.1, 4.2**

### Property 6: خصم من عداد الرسائل عند الإرسال

*لأي* اشتراك نشط مع رسائل متاحة > 0، عند إرسال رسالة، يجب أن ينخفض عداد messages_remaining بمقدار 1 (أو عدد الأجزاء للرسائل الطويلة)

**Validates: Requirements 4.5**

### Property 7: السماح بالاستمرار عند نفاد الرسائل (Overage)

*لأي* اشتراك نشط مع messages_remaining = 0، يجب أن يسمح النظام بإرسال رسائل إضافية (لا يرفض الإرسال)

**Validates: Requirements 5.1**

### Property 8: خصم تكلفة الرسائل الإضافية من الرصيد

*لأي* رسالة إضافية (overage) مع رصيد >= تكلفة الرسالة، يجب أن يخصم النظام التكلفة من الرصيد بشكل ذري، وإذا كان الرصيد < التكلفة، يجب أن يرفض الإرسال

**Validates: Requirements 5.3, 5.4**

### Property 9: حساب الرسائل المتبقية عند نهاية الشهر

*لأي* اشتراك نشط مع rollover_enabled = true، عند انتهاء الشهر (end_date)، يجب أن يحسب النظام messages_remaining ويسجلها في جدول rollover_messages

**Validates: Requirements 6.2**

### Property 10: احترام الحد الأقصى للترحيل

*لأي* باقة مع rollover_limit محدد، عند ترحيل الرسائل، يجب أن يكون عدد الرسائل المرحّلة <= rollover_limit

**Validates: Requirements 6.5**

### Property 11: استخدام FIFO للرسائل المرحّلة

*لأي* عميل مع رسائل مرحّلة من عدة أشهر، عند إرسال رسالة، يجب أن يستخدم النظام الرسائل من أقدم source_month أولاً

**Validates: Requirements 6.8**

### Property 12: حساب أجزاء الرسالة تلقائياً

*لأي* نص رسالة، يجب أن يحسب النظام عدد الأجزاء بناءً على قواعد التقسيم (segmentation_rules) المطابقة لنوع الترميز (encoding_type)

**Validates: Requirements 7.6**

### Property 13: التجديد التلقائي للاشتراك

*لأي* اشتراك مع auto_renew = true وend_date <= CURRENT_DATE، إذا كان الرصيد >= سعر الباقة، يجب أن يجدد النظام الاشتراك تلقائياً ويخصم السعر، وإذا كان الرصيد < السعر، يجب أن يعلّق الاشتراك بعد فترة السماح

**Validates: Requirements 14.2, 14.3**

### Property 14: تسجيل كل عملية مالية

*لأي* عملية تغيّر الرصيد (خصم، استرجاع، توريد، اشتراك)، يجب أن يسجل النظام معاملة في accounting_transactions مع جميع التفاصيل المطلوبة

**Validates: Requirements 16.1, 16.3, 16.4**

### Property 15: Atomic Balance Operations (معالجة Race Condition)

*لأي* عمليتي خصم متزامنتين على نفس الحساب، يجب أن تنجح واحدة فقط إذا كان الرصيد كافياً لواحدة فقط، ويجب ألا يصبح الرصيد سالباً أبداً

**Validates: معالجة المشكلة الحرجة 1 في النظام الحالي**

### Property 16: Tenant Isolation في قاعدة البيانات

*لأي* استعلام على أي جدول، يجب أن يعيد النظام فقط البيانات التي تخص الـ tenant الحالي (tenant_id = current_setting('app.current_tenant'))

**Validates: Multi-Tenancy Isolation - Database Level**

### Property 17: Tenant Isolation في Redis

*لأي* عملية cache، يجب أن يبدأ المفتاح بـ tenant_id، ويجب ألا يتمكن tenant من قراءة أو كتابة مفاتيح tenant آخر

**Validates: Multi-Tenancy Isolation - Cache Level**

### Property 18: Cross-Tenant Access Prevention

*لأي* محاولة للوصول لمورد (resource) من tenant مختلف، يجب أن يرفض النظام الطلب ويعيد خطأ CROSS_TENANT_ACCESS_DENIED

**Validates: Multi-Tenancy Isolation - API Level**

### Property 19: Worker Pool Isolation

*لأي* عمليتين متزامنتين من tenants مختلفين، يجب ألا تؤثر إحداهما على الأخرى في استخدام الموارد (CPU، Memory، Queue)

**Validates: Multi-Tenancy Isolation - Worker Level**

### Property 20: Rate Limiting per Tenant

*لأي* tenant يتجاوز حد الـ rate limit الخاص به، يجب أن يرفض النظام طلباته دون التأثير على tenants أخرى

**Validates: Multi-Tenancy Isolation - Rate Limiting**


---

## Error Handling

### استراتيجية معالجة الأخطاء

#### 1. أخطاء الأعمال (Business Errors)

```go
// internal/errors/business.go

var (
    ErrInsufficientBalance   = errors.New("insufficient balance")
    ErrTariffNotFound        = errors.New("tariff not found")
    ErrPackageNotFound       = errors.New("package not found")
    ErrSubscriptionNotActive = errors.New("subscription not active")
    ErrInvalidPackage        = errors.New("invalid package")
    ErrGatewayNotFound       = errors.New("gateway not found")
    ErrQuotaExceeded         = errors.New("resource quota exceeded")
)
```

#### 2. أخطاء النظام (System Errors)

```go
// internal/errors/system.go

var (
    ErrDatabaseConnection = errors.New("database connection failed")
    ErrCacheConnection    = errors.New("cache connection failed")
    ErrTransactionTimeout = errors.New("transaction timeout")
    ErrConcurrencyConflict = errors.New("concurrency conflict")
)
```

#### 3. معالجة الأخطاء في المعاملات المالية

```go
// internal/balance/manager.go

func (bm *BalanceManager) DeductBalance(ctx context.Context, tenantID uuid.UUID, amount float64, reason string, messageID *uuid.UUID) error {
    err := bm.db.Transaction(func(tx *gorm.DB) error {
        // ... transaction logic ...
    })
    
    if err != nil {
        // Log error with context
        log.Error().
            Err(err).
            Str("tenant_id", tenantID.String()).
            Float64("amount", amount).
            Str("reason", reason).
            Msg("Failed to deduct balance")
        
        // Return appropriate error
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return ErrTenantNotFound
        }
        
        if strings.Contains(err.Error(), "insufficient balance") {
            return ErrInsufficientBalance
        }
        
        return fmt.Errorf("balance deduction failed: %w", err)
    }
    
    return nil
}
```

#### 4. Retry Logic للعمليات الحرجة

```go
// internal/billing/auto_renewal.go

func (ar *AutoRenewal) RenewSubscription(subscriptionID int) error {
    maxRetries := 3
    backoff := time.Second
    
    for attempt := 1; attempt <= maxRetries; attempt++ {
        err := ar.attemptRenewal(subscriptionID)
        
        if err == nil {
            return nil
        }
        
        // Don't retry business errors
        if errors.Is(err, ErrInsufficientBalance) {
            return err
        }
        
        // Retry system errors with exponential backoff
        if attempt < maxRetries {
            time.Sleep(backoff)
            backoff *= 2
        }
    }
    
    return fmt.Errorf("renewal failed after %d attempts", maxRetries)
}
```

---

## Testing Strategy

### استراتيجية الاختبار المزدوجة

يستخدم النظام نهجاً مزدوجاً للاختبار:

1. **Unit Tests**: للأمثلة المحددة، الحالات الحدية، وشروط الأخطاء
2. **Property-Based Tests**: للخصائص العامة عبر جميع المدخلات

### Property-Based Testing

#### المكتبة المستخدمة: gopter

```go
// go.mod
require (
    github.com/leanovate/gopter v0.2.9
)
```

#### مثال: اختبار Property 5 (الاشتراك الآمن)

```go
// internal/subscription/subscription_property_test.go

import (
    "testing"
    "github.com/leanovate/gopter"
    "github.com/leanovate/gopter/gen"
    "github.com/leanovate/gopter/prop"
)

// Feature: tariff-pricing-system, Property 5: الاشتراك الآمن في الباقة
func TestProperty_SafeSubscription(t *testing.T) {
    properties := gopter.NewProperties(nil)
    
    properties.Property("subscription succeeds with sufficient balance", prop.ForAll(
        func(balance float64, packagePrice float64) bool {
            // Setup
            tenant := createTestTenant(balance)
            pkg := createTestPackage(packagePrice)
            
            // Act
            err := subscriptionService.Subscribe(tenant.ID, pkg.ID)
            
            // Assert
            if balance >= packagePrice {
                // Should succeed
                if err != nil {
                    return false
                }
                
                // Check balance deducted
                updatedTenant := getTenant(tenant.ID)
                expectedBalance := balance - packagePrice
                
                return math.Abs(updatedTenant.Balance - expectedBalance) < 0.0001
            } else {
                // Should fail
                return err == ErrInsufficientBalance
            }
        },
        gen.Float64Range(0, 1000),    // balance
        gen.Float64Range(10, 100),    // package price
    ))
    
    properties.TestingRun(t, gopter.ConsoleReporter(false))
}
```

#### مثال: اختبار Property 16 (Tenant Isolation في قاعدة البيانات)

```go
// internal/database/tenant_isolation_property_test.go

// Feature: tariff-pricing-system, Property 16: Tenant Isolation في قاعدة البيانات
func TestProperty_DatabaseTenantIsolation(t *testing.T) {
    properties := gopter.NewProperties(nil)
    
    properties.Property("queries return only current tenant data", prop.ForAll(
        func(tenant1Data, tenant2Data []TariffData) bool {
            // Setup: إنشاء بيانات لـ tenant1
            tenant1 := createTestTenant()
            for _, data := range tenant1Data {
                createTariff(tenant1.ID, data)
            }
            
            // Setup: إنشاء بيانات لـ tenant2
            tenant2 := createTestTenant()
            for _, data := range tenant2Data {
                createTariff(tenant2.ID, data)
            }
            
            // Act: الاستعلام كـ tenant1
            setCurrentTenant(tenant1.ID)
            results1 := getAllTariffs()
            
            // Act: الاستعلام كـ tenant2
            setCurrentTenant(tenant2.ID)
            results2 := getAllTariffs()
            
            // Assert: tenant1 يرى بياناته فقط
            if len(results1) != len(tenant1Data) {
                return false
            }
            
            for _, result := range results1 {
                if result.TenantID != tenant1.ID {
                    return false
                }
            }
            
            // Assert: tenant2 يرى بياناته فقط
            if len(results2) != len(tenant2Data) {
                return false
            }
            
            for _, result := range results2 {
                if result.TenantID != tenant2.ID {
                    return false
                }
            }
            
            return true
        },
        gen.SliceOf(genTariffData()),
        gen.SliceOf(genTariffData()),
    ))
    
    properties.TestingRun(t, gopter.ConsoleReporter(false))
}
```

#### مثال: اختبار Property 18 (Cross-Tenant Access Prevention)

```go
// internal/api/cross_tenant_property_test.go

// Feature: tariff-pricing-system, Property 18: Cross-Tenant Access Prevention
func TestProperty_CrossTenantAccessPrevention(t *testing.T) {
    properties := gopter.NewProperties(nil)
    
    properties.Property("cross-tenant access is denied", prop.ForAll(
        func(resourceType string) bool {
            // Setup: إنشاء موارد لـ tenant1
            tenant1 := createTestTenant()
            resource1 := createResource(tenant1.ID, resourceType)
            
            // Setup: إنشاء tenant2
            tenant2 := createTestTenant()
            
            // Act: محاولة tenant2 الوصول لمورد tenant1
            token := generateToken(tenant2.ID)
            response := makeAPIRequest(token, fmt.Sprintf("/api/%s/%s", resourceType, resource1.ID))
            
            // Assert: يجب أن يُرفض الطلب
            if response.StatusCode != http.StatusForbidden {
                return false
            }
            
            if response.Error.Code != "CROSS_TENANT_ACCESS_DENIED" {
                return false
            }
            
            // Assert: يجب تسجيل المحاولة في audit log
            auditLogs := getAuditLogs(tenant2.ID)
            found := false
            for _, log := range auditLogs {
                if log.EventType == "CROSS_TENANT_ACCESS_ATTEMPT" {
                    found = true
                    break
                }
            }
            
            return found
        },
        gen.OneConstOf("tariffs", "packages", "subscriptions", "gateways"),
    ))
    
    properties.TestingRun(t, gopter.ConsoleReporter(false))
}
```

#### التكوين الموصى به

```go
// كل اختبار property يجب أن يعمل 100 تكرار على الأقل
properties := gopter.NewProperties(&gopter.TestParameters{
    MinSuccessfulTests: 100,
    MaxSize: 100,
    Rng: rand.New(rand.NewSource(time.Now().UnixNano())),
})
```

### Unit Testing

#### مثال: اختبار حالة محددة

```go
// internal/tariff/engine_test.go

func TestTariffEngine_CalculateCost_YemenMobile(t *testing.T) {
    // Feature: tariff-pricing-system, Example: تعرفة يمن موبايل
    
    engine := NewTariffEngine(db, cache)
    
    cost, err := engine.CalculateCost(
        context.Background(),
        1,              // service_id (Hudud Gateway)
        "YE",           // country_code
        "Yemen Mobile", // network_operator
        "standard",     // message_type
        50,             // message_length (1 segment)
    )
    
    assert.NoError(t, err)
    assert.Equal(t, 9.0, cost) // 9 YER for Yemen Mobile
}
```

### Integration Testing

```go
// internal/integration/subscription_flow_test.go

func TestSubscriptionFlow_EndToEnd(t *testing.T) {
    // Test complete subscription flow
    // 1. Create tenant with balance
    // 2. Subscribe to package
    // 3. Send messages
    // 4. Check overage
    // 5. Auto-renewal
    // 6. Rollover
}

// internal/integration/tenant_isolation_test.go

func TestTenantIsolation_EndToEnd(t *testing.T) {
    // Test complete tenant isolation
    // 1. Create two tenants
    // 2. Create data for each tenant
    // 3. Verify each tenant sees only their data
    // 4. Attempt cross-tenant access
    // 5. Verify access is denied
    // 6. Verify audit logs
}
```

### Concurrent Testing للعزل

```go
// internal/isolation/concurrent_test.go

func TestConcurrentOperations_TenantIsolation(t *testing.T) {
    // Feature: tariff-pricing-system, Property 19: Worker Pool Isolation
    
    tenant1 := createTestTenant()
    tenant2 := createTestTenant()
    
    var wg sync.WaitGroup
    errors := make(chan error, 200)
    
    // 100 عملية متزامنة لـ tenant1
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            if err := performOperation(tenant1.ID); err != nil {
                errors <- err
            }
        }()
    }
    
    // 100 عملية متزامنة لـ tenant2
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            if err := performOperation(tenant2.ID); err != nil {
                errors <- err
            }
        }()
    }
    
    wg.Wait()
    close(errors)
    
    // التحقق من عدم وجود أخطاء
    for err := range errors {
        t.Errorf("Operation failed: %v", err)
    }
    
    // التحقق من أن كل tenant حصل على موارده
    stats1 := getResourceStats(tenant1.ID)
    stats2 := getResourceStats(tenant2.ID)
    
    assert.Equal(t, 100, stats1.CompletedOperations)
    assert.Equal(t, 100, stats2.CompletedOperations)
}
```

### Performance Testing

```go
// internal/balance/manager_benchmark_test.go

func BenchmarkBalanceManager_ConcurrentDeductions(b *testing.B) {
    // Test concurrent balance deductions
    // Verify no race conditions
    // Measure throughput
}
```

---

## APIs

### Tenant Validation في جميع APIs

**قاعدة أساسية:** جميع APIs يجب أن تتحقق من tenant_id وتطبق العزل

```go
// internal/middleware/tenant.go

func TenantMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // استخراج tenant_id من JWT أو API Key
        tenantID, err := extractTenantID(c)
        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{
                "success": false,
                "error": map[string]interface{}{
                    "code":    "INVALID_TENANT",
                    "message": "Unable to identify tenant",
                },
            })
            c.Abort()
            return
        }
        
        // تعيين tenant_id في السياق
        c.Set("tenant_id", tenantID)
        
        // تعيين tenant_id في database session
        db := getDB(c)
        if _, err := db.Exec("SET app.current_tenant = $1", tenantID.String()); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{
                "success": false,
                "error": map[string]interface{}{
                    "code":    "DATABASE_ERROR",
                    "message": "Failed to set tenant context",
                },
            })
            c.Abort()
            return
        }
        
        c.Next()
    }
}

func CrossTenantPreventionMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        currentTenantID := c.GetString("tenant_id")
        
        // فحص tenant_id في المعاملات
        if paramTenantID := c.Param("tenant_id"); paramTenantID != "" {
            if paramTenantID != currentTenantID {
                // تسجيل المحاولة
                logCrossTenantAttempt(c, currentTenantID, paramTenantID)
                
                c.JSON(http.StatusForbidden, gin.H{
                    "success": false,
                    "error": map[string]interface{}{
                        "code":    "CROSS_TENANT_ACCESS_DENIED",
                        "message": "You do not have permission to access resources from another tenant",
                        "details": map[string]interface{}{
                            "your_tenant_id":      currentTenantID,
                            "requested_tenant_id": paramTenantID,
                        },
                    },
                })
                c.Abort()
                return
            }
        }
        
        c.Next()
    }
}
```

### REST APIs

#### 1. Tariff Management APIs

```
POST   /api/admin/tariffs              - إنشاء تعرفة جديدة
GET    /api/admin/tariffs              - عرض جميع التعرفات (للـ tenant الحالي فقط)
GET    /api/admin/tariffs/:id          - عرض تعرفة محددة (مع التحقق من tenant_id)
PUT    /api/admin/tariffs/:id          - تحديث تعرفة (مع التحقق من tenant_id)
DELETE /api/admin/tariffs/:id          - حذف تعرفة (مع التحقق من tenant_id)
GET    /api/admin/tariffs/history      - عرض السجل التاريخي (للـ tenant الحالي فقط)
```

**مثال على التطبيق:**

```go
// internal/api/tariff_handler.go

func (h *TariffHandler) GetTariff(c *gin.Context) {
    tenantID := c.GetString("tenant_id")
    tariffID := c.Param("id")
    
    var tariff models.Tariff
    
    // RLS سيطبق تلقائياً، لكن نضيف فحص إضافي للوضوح
    err := h.db.Where("id = ? AND tenant_id = ?", tariffID, tenantID).First(&tariff).Error
    
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            c.JSON(http.StatusNotFound, gin.H{
                "success": false,
                "error": map[string]interface{}{
                    "code":    "TARIFF_NOT_FOUND",
                    "message": "Tariff not found or you don't have access to it",
                },
            })
            return
        }
        
        c.JSON(http.StatusInternalServerError, gin.H{
            "success": false,
            "error": map[string]interface{}{
                "code":    "DATABASE_ERROR",
                "message": "Failed to retrieve tariff",
            },
        })
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "data":    tariff,
    })
}

func (h *TariffHandler) CreateTariff(c *gin.Context) {
    tenantID := c.GetString("tenant_id")
    
    var req CreateTariffRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "success": false,
            "error": map[string]interface{}{
                "code":    "INVALID_REQUEST",
                "message": err.Error(),
            },
        })
        return
    }
    
    tariff := &models.Tariff{
        TenantID:        uuid.MustParse(tenantID),  // إلزامي
        ServiceID:       req.ServiceID,
        CountryCode:     req.CountryCode,
        NetworkOperator: req.NetworkOperator,
        CostPerMessage:  req.CostPerMessage,
        // ... بقية الحقول
    }
    
    if err := h.db.Create(tariff).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "success": false,
            "error": map[string]interface{}{
                "code":    "CREATE_FAILED",
                "message": "Failed to create tariff",
            },
        })
        return
    }
    
    c.JSON(http.StatusCreated, gin.H{
        "success": true,
        "data":    tariff,
    })
}
```

#### 2. Package Management APIs

```
POST   /api/admin/packages             - إنشاء باقة جديدة (مع tenant_id)
GET    /api/admin/packages             - عرض جميع الباقات (للـ tenant الحالي فقط)
GET    /api/admin/packages/:id         - عرض باقة محددة (مع التحقق من tenant_id)
PUT    /api/admin/packages/:id         - تحديث باقة (مع التحقق من tenant_id)
DELETE /api/admin/packages/:id         - حذف باقة (مع التحقق من tenant_id)
```

#### 3. Subscription APIs

```
POST   /api/subscriptions              - الاشتراك في باقة (مع tenant_id)
GET    /api/subscriptions              - عرض اشتراكات العميل (للـ tenant الحالي فقط)
GET    /api/subscriptions/:id          - عرض اشتراك محدد (مع التحقق من tenant_id)
PUT    /api/subscriptions/:id/cancel   - إلغاء اشتراك (مع التحقق من tenant_id)
PUT    /api/subscriptions/:id/renew    - تجديد اشتراك يدوياً (مع التحقق من tenant_id)
GET    /api/subscriptions/:id/usage    - عرض الاستخدام (مع التحقق من tenant_id)
```

#### 4. Gateway Management APIs

```
POST   /api/admin/gateways             - إنشاء بوابة مخصصة (مع tenant_id)
GET    /api/admin/gateways             - عرض جميع البوابات (للـ tenant الحالي فقط)
GET    /api/admin/gateways/:id         - عرض بوابة محددة (مع التحقق من tenant_id)
PUT    /api/admin/gateways/:id         - تحديث بوابة (مع التحقق من tenant_id)
DELETE /api/admin/gateways/:id         - حذف بوابة (مع التحقق من tenant_id)
GET    /api/admin/gateways/:id/stats   - إحصائيات البوابة (مع التحقق من tenant_id)
```

#### 5. Balance Management APIs

```
POST   /api/admin/balance/topup        - توريد رصيد مباشر (مع tenant_id)
GET    /api/balance                    - عرض الرصيد الحالي (للـ tenant الحالي فقط)
GET    /api/balance/history            - سجل تغييرات الرصيد (للـ tenant الحالي فقط)
GET    /api/transactions               - سجل المعاملات المالية (للـ tenant الحالي فقط)
```


### API Request/Response Examples

#### مثال: الاشتراك في باقة

**Request:**
```http
POST /api/subscriptions
Content-Type: application/json
Authorization: Bearer {token}

{
  "package_id": 2,
  "auto_renew": true
}
```

**Response (Success):**
```http
HTTP/1.1 201 Created
Content-Type: application/json

{
  "success": true,
  "data": {
    "subscription_id": 123,
    "package": {
      "id": 2,
      "name": "Standard",
      "price": 20.00,
      "included_messages": 3000
    },
    "start_date": "2026-02-27",
    "end_date": "2026-03-27",
    "messages_remaining": 3000,
    "balance_after": 30.00
  }
}
```

**Response (Insufficient Balance):**
```http
HTTP/1.1 400 Bad Request
Content-Type: application/json

{
  "success": false,
  "error": {
    "code": "INSUFFICIENT_BALANCE",
    "message": "رصيدك الحالي (5.00 USD) غير كافي للاشتراك في هذه الباقة (20.00 USD)",
    "details": {
      "current_balance": 5.00,
      "required_amount": 20.00,
      "shortfall": 15.00
    }
  }
}
```

---

## واجهات المستخدم

### مبدأ أساسي: استخدام الواجهات الموجودة

**قيد حرج:** يُمنع إعادة تصميم واجهات المستخدم الموجودة من الصفر. جميع التعديلات يجب أن تكون:
- تعديلات طفيفة على الواجهات الموجودة
- إضافة حقول جديدة للنماذج الموجودة
- إضافة أزرار أو روابط جديدة للصفحات الموجودة
- تعديل API calls لتتوافق مع النظام الجديد
- **لا** تُنشئ صفحات جديدة إلا إذا كانت ضرورية جداً

### استراتيجية التكامل مع الكود الموجود

#### 1. تحديد الواجهات الموجودة

```
apps/admin-web/
├── src/pages/
│   ├── Dashboard.tsx          (موجود - نضيف إحصائيات التعرفات)
│   ├── Settings.tsx           (موجود - نضيف إعدادات التعرفات)
│   └── Users.tsx              (موجود - نستخدمه كما هو)

apps/client-web/
├── src/pages/
│   ├── Dashboard.tsx          (موجود - نضيف معلومات الباقة)
│   ├── Messages.tsx           (موجود - نضيف حساب التكلفة)
│   └── Profile.tsx            (موجود - نضيف معلومات الاشتراك)
```

#### 2. التعديلات المطلوبة على الواجهات الموجودة

##### أ. واجهة الإدارة (apps/admin-web)

**صفحة Dashboard الموجودة - إضافة قسم التعرفات:**

```typescript
// apps/admin-web/src/pages/Dashboard.tsx
// إضافة import جديد
import { TariffSummary } from '../components/TariffSummary';

// إضافة في الـ component الموجود
function Dashboard() {
  // الكود الموجود...
  
  return (
    <div className="dashboard">
      {/* المكونات الموجودة */}
      <StatisticsCards />
      <RecentActivity />
      
      {/* إضافة مكون جديد */}
      <TariffSummary />
    </div>
  );
}
```

**صفحة Settings الموجودة - إضافة تبويب التعرفات:**

```typescript
// apps/admin-web/src/pages/Settings.tsx
// إضافة تبويب جديد للتعرفات في الصفحة الموجودة

function Settings() {
  const [activeTab, setActiveTab] = useState('general');
  
  return (
    <div className="settings">
      <Tabs value={activeTab} onChange={setActiveTab}>
        <Tab value="general">عام</Tab>
        <Tab value="users">المستخدمون</Tab>
        {/* إضافة تبويب جديد */}
        <Tab value="tariffs">التعرفات</Tab>
        <Tab value="packages">الباقات</Tab>
      </Tabs>
      
      {activeTab === 'general' && <GeneralSettings />}
      {activeTab === 'users' && <UserSettings />}
      {/* إضافة محتوى التبويبات الجديدة */}
      {activeTab === 'tariffs' && <TariffSettings />}
      {activeTab === 'packages' && <PackageSettings />}
    </div>
  );
}
```

##### ب. واجهة العميل (apps/client-web)

**صفحة Dashboard الموجودة - إضافة معلومات الباقة:**

```typescript
// apps/client-web/src/pages/Dashboard.tsx
// إضافة import جديد
import { SubscriptionInfo } from '../components/SubscriptionInfo';
import { MessageUsage } from '../components/MessageUsage';

function Dashboard() {
  return (
    <div className="dashboard">
      {/* المكونات الموجودة */}
      <WelcomeCard />
      <QuickActions />
      
      {/* إضافة مكونات جديدة */}
      <SubscriptionInfo />
      <MessageUsage />
      
      {/* المكونات الموجودة */}
      <RecentMessages />
    </div>
  );
}
```

**صفحة Messages الموجودة - إضافة حساب التكلفة:**

```typescript
// apps/client-web/src/pages/Messages.tsx
// تعديل النموذج الموجود لإضافة حساب التكلفة

function MessageForm() {
  const [message, setMessage] = useState('');
  const [cost, setCost] = useState(0);
  
  // إضافة useEffect لحساب التكلفة
  useEffect(() => {
    if (message) {
      calculateCost(message).then(setCost);
    }
  }, [message]);
  
  return (
    <form className="message-form">
      {/* الحقول الموجودة */}
      <textarea 
        value={message}
        onChange={(e) => setMessage(e.target.value)}
        placeholder="اكتب رسالتك هنا"
      />
      
      {/* إضافة عرض التكلفة */}
      {cost > 0 && (
        <div className="cost-info">
          <span>التكلفة المتوقعة: {cost} ريال</span>
          <span>عدد الأجزاء: {calculateSegments(message)}</span>
        </div>
      )}
      
      {/* الأزرار الموجودة */}
      <button type="submit">إرسال</button>
    </form>
  );
}
```

#### 3. المكونات الجديدة المطلوبة (Minimal)

```typescript
// apps/admin-web/src/components/TariffSummary.tsx
// مكون بسيط لعرض ملخص التعرفات
export function TariffSummary() {
  const { data: tariffs } = useTariffs();
  
  return (
    <Card>
      <CardHeader>التعرفات النشطة</CardHeader>
      <CardContent>
        <TariffList tariffs={tariffs} />
      </CardContent>
    </Card>
  );
}

// apps/client-web/src/components/SubscriptionInfo.tsx
// مكون بسيط لعرض معلومات الاشتراك
export function SubscriptionInfo() {
  const { data: subscription } = useSubscription();
  
  if (!subscription) return null;
  
  return (
    <Card>
      <CardHeader>اشتراكك الحالي</CardHeader>
      <CardContent>
        <div>الباقة: {subscription.package.name}</div>
        <div>الرسائل المتبقية: {subscription.messages_remaining}</div>
        <div>تاريخ التجديد: {subscription.next_renewal_date}</div>
      </CardContent>
    </Card>
  );
}
```

#### 4. تعديل API Calls

```typescript
// apps/client-web/src/api/messages.ts
// تحديث دالة إرسال الرسالة لتتوافق مع النظام الجديد

export async function sendMessage(message: MessageData) {
  // إضافة حساب التكلفة قبل الإرسال
  const cost = await calculateCost(message.text);
  
  return api.post('/api/messages/send', {
    ...message,
    estimated_cost: cost,
    // الحقول الموجودة الأخرى
  });
}

// إضافة دالة جديدة لحساب التكلفة
export async function calculateCost(text: string): Promise<number> {
  const response = await api.post('/api/tariffs/calculate', { text });
  return response.data.cost;
}
```

### 1. واجهة الإدارة (Admin Interface)

#### التعديلات على الصفحات الموجودة

**ملاحظة:** جميع الواجهات التالية هي تعديلات على الصفحات الموجودة في `apps/admin-web`

#### صفحة إدارة التعرفات (تبويب جديد في Settings)

```
┌─────────────────────────────────────────────────────────────┐
│ إدارة التعرفات                                    [+ إضافة] │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│ البحث: [_______________] [بحث]                             │
│                                                             │
│ ┌─────────────────────────────────────────────────────────┐ │
│ │ الخدمة    │ الدولة │ الشبكة      │ التكلفة │ الحالة   │ │
│ ├─────────────────────────────────────────────────────────┤ │
│ │ الهدهد    │ YE     │ Yemen Mobile│ 9.00 YER│ ✓ نشط   │ │
│ │ الهدهد    │ YE     │ Other       │ 13.00 YER│ ✓ نشط  │ │
│ │ الهدهد    │ SA     │ STC         │ 0.15 SAR│ ✗ معطل │ │
│ └─────────────────────────────────────────────────────────┘ │
│                                                             │
│ [السابق] صفحة 1 من 5 [التالي]                             │
└─────────────────────────────────────────────────────────────┘
```

#### صفحة إدارة الباقات

```
┌─────────────────────────────────────────────────────────────┐
│ إدارة الباقات الشهرية                            [+ إضافة] │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│ ┌─────────────────────────────────────────────────────────┐ │
│ │ الباقة        │ السعر  │ الرسائل  │ Rollover│ الحالة  │ │
│ ├─────────────────────────────────────────────────────────┤ │
│ │ Basic        │ $10    │ 1,000   │ ✓      │ ✓ نشط   │ │
│ │ Standard     │ $20    │ 3,000   │ ✓      │ ✓ نشط   │ │
│ │ Professional │ $40    │ 8,000   │ ✓      │ ✓ نشط   │ │
│ │ Business     │ $50    │ 15,000  │ ✓      │ ✓ نشط   │ │
│ │ Enterprise   │ $100   │ Unlimited│ ✗     │ ✓ نشط   │ │
│ └─────────────────────────────────────────────────────────┘ │
│                                                             │
│ المشتركون الحاليون:                                        │
│ Basic: 45 | Standard: 123 | Professional: 67 | ...         │
└─────────────────────────────────────────────────────────────┘
```

#### صفحة إدارة البوابات المخصصة

```
┌─────────────────────────────────────────────────────────────┐
│ إدارة البوابات المخصصة                           [+ إضافة] │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│ ┌─────────────────────────────────────────────────────────┐ │
│ │ البوابة      │ الشركة    │ الحالة  │ Uptime │ الرسائل/يوم│ │
│ ├─────────────────────────────────────────────────────────┤ │
│ │ GW-BANK-001 │ البنك الوطني│ ✓ نشط │ 99.9% │ 15,234    │ │
│ │ GW-TELCO-001│ شركة الاتصالات│ ✓ نشط│ 99.8% │ 45,678    │ │
│ │ GW-RETAIL-01│ سلسلة متاجر │ ⚠ صيانة│ 98.5% │ 8,901     │ │
│ └─────────────────────────────────────────────────────────┘ │
│                                                             │
│ التنبيهات النشطة:                                          │
│ ⚠️ GW-RETAIL-01: اقتراب من الحد اليومي (90%)              │
│ ✓ جميع البوابات الأخرى تعمل بشكل طبيعي                    │
└─────────────────────────────────────────────────────────────┘
```

### 2. واجهة العميل العادي (Client Interface)

#### صفحة الباقات

```
┌─────────────────────────────────────────────────────────────┐
│ اختر الباقة المناسبة لك                                    │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│ ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐    │
│ │  Basic   │  │ Standard │  │Professional│ │ Business │    │
│ │          │  │          │  │          │  │          │    │
│ │  $10     │  │  $20     │  │  $40     │  │  $50     │    │
│ │  /شهر    │  │  /شهر    │  │  /شهر    │  │  /شهر    │    │
│ │          │  │          │  │          │  │          │    │
│ │ 1,000    │  │ 3,000    │  │ 8,000    │  │ 15,000   │    │
│ │ رسالة    │  │ رسالة    │  │ رسالة    │  │ رسالة    │    │
│ │          │  │          │  │          │  │          │    │
│ │ ✓ Rollover│ │ ✓ Rollover│ │ ✓ Rollover│ │ ✓ Rollover│   │
│ │          │  │          │  │          │  │          │    │
│ │ [اشترك]  │  │ [اشترك]  │  │ [اشترك]  │  │ [اشترك]  │    │
│ └──────────┘  └──────────┘  └──────────┘  └──────────┘    │
│                                                             │
│ رصيدك الحالي: $50.00                                       │
└─────────────────────────────────────────────────────────────┘
```

#### لوحة التحكم

```
┌─────────────────────────────────────────────────────────────┐
│ لوحة التحكم                                                │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│ الباقة الحالية: Standard ($20/شهر)                        │
│                                                             │
│ ┌─────────────────────────────────────────────────────────┐ │
│ │ الرسائل المتاحة                                         │ │
│ │                                                         │ │
│ │ رسائل الشهر الحالي:  2,500 / 3,000  [████████░░] 83%  │ │
│ │                                                         │ │
│ │ رسائل مرحّلة:                                           │ │
│ │   • من فبراير: 300 (تنتهي في 27 أبريل)                │ │
│ │   • من يناير: 150 (تنتهي في 27 مارس)                  │ │
│ │                                                         │ │
│ │ الإجمالي المتاح: 2,950 رسالة                           │ │
│ └─────────────────────────────────────────────────────────┘ │
│                                                             │
│ تاريخ التجديد: 27 مارس 2026                                │
│ الرصيد الحالي: $50.00                                      │
│ الاستخدام الإضافي هذا الشهر: $0.00                        │
│                                                             │
│ [إرسال رسالة] [عرض السجل] [إدارة الاشتراك]                │
└─────────────────────────────────────────────────────────────┘
```

### 3. واجهة الشركة المؤسسية (Enterprise Portal)

#### لوحة التحكم الرئيسية

```
┌─────────────────────────────────────────────────────────────┐
│ بوابة GW-BANK-001                                          │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│ ┌──────────────────┐  ┌──────────────────┐                 │
│ │ الحالة           │  │ إحصائيات اليوم   │                 │
│ │                  │  │                  │                 │
│ │ ✓ نشطة          │  │ الرسائل: 15,234  │                 │
│ │ Uptime: 99.9%   │  │ النجاح: 98.5%    │                 │
│ │ آخر فحص: 5 دقائق│  │ الاستجابة: 120ms │                 │
│ └──────────────────┘  └──────────────────┘                 │
│                                                             │
│ ┌─────────────────────────────────────────────────────────┐ │
│ │ الاستخدام الشهري                                        │ │
│ │                                                         │ │
│ │ الرسائل:  450,000 / 1,000,000  [████████░░░░] 45%     │ │
│ │ CPU:      2.1 / 4.0 vCPU        [█████░░░░░░] 52%      │ │
│ │ Memory:   3.2 / 8.0 GB          [████░░░░░░░] 40%      │ │
│ │ Bandwidth: 15 / 50 Mbps         [███░░░░░░░░] 30%      │ │
│ └─────────────────────────────────────────────────────────┘ │
│                                                             │
│ التنبيهات:                                                 │
│ ⚠️ اقتراب من الحد اليومي للرسائل (90%)                    │
│ ✓ جميع الأنظمة الأخرى تعمل بشكل طبيعي                     │
│                                                             │
│ [إدارة البوابة] [API Keys] [التقارير] [الإعدادات]        │
└─────────────────────────────────────────────────────────────┘
```


---

## خطة التنفيذ على 7 مراحل

### المرحلة 1: الأساسيات والبنية التحتية (الأسابيع 1-2)

#### الأهداف
- إنشاء البنية التحتية الأساسية
- معالجة المشاكل الحرجة في النظام الحالي
- إنشاء نظام التعرفات الأساسي

#### المهام
1. إنشاء جداول قاعدة البيانات الأساسية
   - services
   - tariffs
   - segmentation_rules
   - currencies
   - exchange_rates

2. تطوير Balance Manager المحسّن
   - Atomic Balance Deduction
   - Idempotent Refund
   - Transaction Isolation
   - Audit Trail (balance_history)

3. تطوير Tariff Engine
   - حساب التكلفة الديناميكي
   - حساب أجزاء الرسائل
   - Caching للتعرفات

4. اختبارات Property-Based
   - Property 15: Atomic Balance Operations
   - Property 3: استخدام التعرفة النشطة
   - Property 12: حساب أجزاء الرسالة

#### المخرجات
- ✅ نظام محاسبة آمن بدون Race Conditions
- ✅ نظام تعرفات ديناميكي
- ✅ Audit Trail كامل

---

### المرحلة 2: الباقات الشهرية (الأسابيع 3-4)

#### الأهداف
- تطوير نظام الباقات الشهرية
- تطوير نظام الاشتراكات
- تطوير نظام Rollover

#### المهام
1. إنشاء جداول قاعدة البيانات
   - packages
   - subscriptions
   - rollover_messages

2. تطوير Subscription Manager
   - الاشتراك في الباقات
   - خصم من عداد الرسائل
   - حساب Overage

3. تطوير Rollover Manager
   - حساب الرسائل المتبقية
   - ترحيل الرسائل
   - FIFO للاستخدام
   - انتهاء الصلاحية

4. اختبارات Property-Based
   - Property 5: الاشتراك الآمن
   - Property 6: خصم من العداد
   - Property 8: خصم Overage
   - Property 9: حساب المتبقي
   - Property 10: احترام الحد الأقصى
   - Property 11: FIFO

#### المخرجات
- ✅ نظام باقات شهرية كامل
- ✅ نظام Rollover مرن
- ✅ نظام Overage

---

### المرحلة 3: الفوترة التلقائية (الأسبوع 5)

#### الأهداف
- تطوير نظام الفوترة التلقائية
- تطوير نظام الإشعارات
- تطوير نظام الفواتير

#### المهام
1. إنشاء جداول قاعدة البيانات
   - billing_invoices

2. تطوير Auto Renewal Service
   - Cron Job يومي
   - فحص الاشتراكات المنتهية
   - التجديد التلقائي
   - فترة السماح

3. تطوير Notification Service
   - إشعارات قبل التجديد
   - إشعارات بعد التجديد
   - إشعارات الفشل

4. تطوير Invoice Generator
   - توليد الفواتير تلقائياً
   - PDF Generation
   - إرسال عبر البريد

5. اختبارات Property-Based
   - Property 13: التجديد التلقائي

#### المخرجات
- ✅ فوترة تلقائية موثوقة
- ✅ نظام إشعارات شامل
- ✅ فواتير احترافية

---

### المرحلة 4: البوابات المخصصة (الأسابيع 6-8)

#### الأهداف
- تطوير نظام البوابات المخصصة
- تطوير نظام Resource Quotas
- تطوير نظام المراقبة

#### المهام
1. إنشاء جداول قاعدة البيانات
   - dedicated_gateways
   - resource_quotas
   - resource_usage
   - gateway_users
   - gateway_api_keys
   - gateway_statistics

2. تطوير Gateway Manager
   - إنشاء بوابات مخصصة
   - إدارة الإعدادات
   - إدارة المستخدمين
   - إدارة API Keys

3. تطوير Resource Limiter
   - فحص الحصص
   - Rate Limiting
   - Worker Pool محدود
   - Enforcement (Soft/Hard)

4. تطوير Monitoring Service
   - جمع المقاييس
   - تسجيل الاستخدام
   - التنبيهات التلقائية
   - Health Checks

5. تطوير Statistics Aggregator
   - إحصائيات يومية
   - إحصائيات شهرية
   - حساب Uptime
   - حساب الأرباح

#### المخرجات
- ✅ نظام بوابات مخصصة كامل
- ✅ عزل موارد قوي
- ✅ مراقبة شاملة

---

### المرحلة 5: الواجهات (الأسابيع 9-10)

#### الأهداف
- تطوير واجهة الإدارة
- تطوير واجهة العميل
- تطوير واجهة الشركات المؤسسية

#### المهام
1. واجهة الإدارة (Admin Interface)
   - إدارة التعرفات
   - إدارة الباقات
   - إدارة البوابات المخصصة
   - التوريد المباشر
   - التقارير المالية

2. واجهة العميل (Client Interface)
   - عرض الباقات
   - الاشتراك
   - لوحة التحكم
   - إرسال الرسائل
   - السجل والتقارير

3. واجهة الشركات (Enterprise Portal)
   - لوحة التحكم
   - إدارة البوابة
   - إدارة المستخدمين
   - إدارة API Keys
   - التقارير المفصلة

#### المخرجات
- ✅ واجهات مستخدم كاملة
- ✅ تجربة مستخدم ممتازة
- ✅ Responsive Design

---

### المرحلة 6: المزايا المتقدمة (الأسابيع 11-12)

#### الأهداف
- تطوير نظام الخصومات والعروض
- تطوير نظام الإحالة
- تطوير التكامل مع بوابات الدفع

#### المهام
1. نظام الخصومات
   - أكواد الخصم
   - العروض الترويجية
   - التسعير المتدرج

2. نظام الإحالة
   - أكواد الإحالة
   - المكافآت
   - التتبع

3. التكامل مع بوابات الدفع
   - Stripe Integration
   - PayPal Integration
   - بوابات محلية

4. دعم اللغات المتعددة
   - العربية
   - الإنجليزية
   - RTL Support

#### المخرجات
- ✅ نظام خصومات مرن
- ✅ برنامج إحالة فعال
- ✅ دفع إلكتروني آمن

---

### المرحلة 7: الاستقرار والتحسين (الأسابيع 13-14)

#### الأهداف
- اختبارات شاملة
- تحسين الأداء
- التوثيق الكامل

#### المهام
1. اختبارات شاملة
   - Integration Tests
   - Load Tests
   - Stress Tests
   - Security Tests

2. تحسين الأداء
   - Database Optimization
   - Caching Strategy
   - Query Optimization
   - Connection Pooling

3. التوثيق
   - API Documentation
   - User Guides
   - Admin Guides
   - Developer Guides

4. النسخ الاحتياطي والاسترجاع
   - Backup Strategy
   - Disaster Recovery Plan
   - Testing Recovery

#### المخرجات
- ✅ نظام مستقر وموثوق
- ✅ أداء ممتاز
- ✅ توثيق شامل

---

## الخلاصة

هذا التصميم يوفر نظام تعرفات وتسعير شامل يدعم ثلاثة نماذج خدمات مختلفة مع معالجة كاملة للمشاكل الحرجة في النظام الحالي. النظام مصمم ليكون:

- **آمن مالياً**: معالجة جميع Race Conditions والمشاكل الحرجة
- **مرن**: دعم نماذج خدمات متعددة
- **قابل للتوسع**: بنية تحتية مشتركة مع عزل قوي
- **موثوق**: Audit Trail كامل واختبارات شاملة
- **قابل للصيانة**: كود نظيف ومنظم مع توثيق شامل
- **معزول تماماً**: عزل تام بين الشركات (Multi-Tenancy Isolation) على جميع المستويات

### التحديثات الرئيسية في هذا الإصدار

#### 1. العزل التام بين الشركات (Multi-Tenancy Isolation)

تم إضافة قسم شامل يغطي:
- **عزل قاعدة البيانات**: Row Level Security (RLS) على جميع الجداول، tenant_id في كل جدول، Indexes مركبة
- **عزل Redis**: Key Prefixing Strategy، Namespace Separation، TTL Management per Tenant
- **عزل Worker Pools**: Worker Pool per Tenant، Queue Isolation، Resource Limits
- **عزل API Keys**: API Keys مرتبطة بـ Tenant، JWT Claims مع tenant_id، Middleware Validation
- **عزل الموارد**: CPU/Memory Quotas، Rate Limiting per Tenant، Database Connection Pools
- **عزل البيانات المالية**: Balance Isolation، Transaction Isolation، Accounting Isolation
- **عزل الإحصائيات**: Statistics Isolation، Reports Isolation، Dashboard Isolation
- **عزل الإشعارات**: Notification Queues، Email Templates per Tenant
- **منع Cross-Tenant Access**: Validation Middleware، Error Responses، Audit Trail

#### 2. تحديث تصميم قاعدة البيانات

- إضافة `tenant_id UUID NOT NULL REFERENCES tenants(id)` لجميع الجداول
- تطبيق Row Level Security (RLS) على جميع الجداول
- إنشاء Policies للعزل (SELECT، INSERT، UPDATE، DELETE)
- تحديث جميع الـ Indexes لتبدأ بـ tenant_id
- إضافة Foreign Keys مع Tenant للحفاظ على سلامة البيانات

#### 3. تحديث قسم الواجهات

- **قيد حرج**: يُمنع إعادة تصميم واجهات المستخدم من الصفر
- **استراتيجية التكامل**: استخدام الواجهات الموجودة في apps/admin-web و apps/client-web
- **التعديلات المسموحة**:
  - إضافة حقول جديدة للنماذج الموجودة
  - إضافة أزرار أو روابط جديدة
  - تعديل API calls لتتوافق مع النظام الجديد
  - إضافة مكونات جديدة بسيطة (Minimal Components)
- **أمثلة عملية**: تعديلات محددة على Dashboard.tsx، Settings.tsx، Messages.tsx

#### 4. تحديث قسم APIs

- إضافة Tenant Validation Middleware لجميع APIs
- إضافة Cross-Tenant Prevention Middleware
- تحديث جميع API Handlers للتحقق من tenant_id
- إضافة Error Responses لمحاولات Cross-Tenant Access
- أمثلة كود كاملة لـ Tenant-Aware API Handlers

#### 5. تحديث Correctness Properties

إضافة 6 Properties جديدة للعزل:
- **Property 16**: Tenant Isolation في قاعدة البيانات
- **Property 17**: Tenant Isolation في Redis
- **Property 18**: Cross-Tenant Access Prevention
- **Property 19**: Worker Pool Isolation
- **Property 20**: Rate Limiting per Tenant

#### 6. تحديث Testing Strategy

- إضافة Property-Based Tests للعزل
- إضافة Integration Tests للعزل
- إضافة Concurrent Tests للتحقق من عدم تأثير tenants على بعضهم
- أمثلة كود كاملة لاختبارات العزل

### ضمانات العزل

النظام يضمن:

1. **عزل البيانات**: كل tenant يرى بياناته فقط
2. **عزل الأداء**: عمليات tenant لا تؤثر على tenants أخرى
3. **عزل الموارد**: كل tenant له حصة موارد محددة
4. **عزل الأمان**: لا يمكن لـ tenant الوصول لبيانات tenant آخر
5. **عزل المالية**: رصيد ومعاملات كل tenant معزولة تماماً

### التنفيذ على 7 مراحل

التنفيذ على 7 مراحل يضمن تسليم تدريجي مع اختبار مستمر وتقليل المخاطر، مع التركيز على:
- المرحلة 1: الأساسيات والبنية التحتية (مع العزل)
- المرحلة 2: الباقات الشهرية (مع العزل)
- المرحلة 3: الفوترة التلقائية (مع العزل)
- المرحلة 4: البوابات المخصصة (مع العزل الكامل)
- المرحلة 5: الواجهات (تعديلات طفيفة فقط)
- المرحلة 6: المزايا المتقدمة
- المرحلة 7: الاستقرار والتحسين

---

**تاريخ الإنشاء:** 2026-02-27  
**تاريخ التحديث:** 2026-02-27  
**الإصدار:** 2.0  
**الحالة:** محدّث ومراجع - جاهز للتنفيذ

**التحديثات الرئيسية:**
- إضافة قسم شامل عن Multi-Tenancy Isolation
- تحديث تصميم قاعدة البيانات لإضافة tenant_id وRLS
- تحديث قسم الواجهات للتأكيد على استخدام الواجهات الموجودة
- تحديث قسم APIs لإضافة Tenant Validation
- إضافة Correctness Properties للعزل
- تحديث Testing Strategy لتشمل اختبارات العزل
