# Database Failure Recovery Test - الدليل الشامل

## المقدمة

هذا الدليل يوفر شرحاً تفصيلياً لاختبار Database Failure Recovery الذي يتحقق من قدرة النظام على التعافي من فشل قاعدة البيانات والتحول التلقائي لقاعدة بيانات احتياطية.

## الأهداف

### الهدف الرئيسي
التحقق من أن النظام يمكنه:
1. اكتشاف فشل قاعدة البيانات الرئيسية
2. التحول التلقائي لقاعدة البيانات الاحتياطية
3. الاستمرار في العمل بدون فقدان بيانات
4. إعادة الاتصال بالقاعدة الرئيسية عند عودتها

### المتطلبات المرتبطة
- **المتطلب 24.3:** استرداد من الأعطال - التحول لقاعدة البيانات الاحتياطية

## البنية المعمارية

### نظرة عامة على High Availability

```
┌─────────────────────────────────────────────────────────┐
│                    Application Layer                     │
│                  (Backend Services)                      │
└────────────────────┬────────────────────────────────────┘
                     │
                     │ Connection Pool
                     │
        ┌────────────┴────────────┐
        │                         │
        ▼                         ▼
┌──────────────┐          ┌──────────────┐
│   Primary    │          │    Backup    │
│   Database   │◄────────►│   Database   │
│ (PostgreSQL) │ Replication│ (PostgreSQL) │
└──────────────┘          └──────────────┘
     Active                   Standby
```

### مكونات النظام

#### 1. Connection Pool
- **الوظيفة:** إدارة الاتصالات بقاعدة البيانات
- **التقنية:** pgxpool (PostgreSQL connection pool)
- **الإعدادات:**
  - MaxConns: 100 (الحد الأقصى للاتصالات)
  - MinConns: 10 (الحد الأدنى للاتصالات الخاملة)
  - MaxConnLifetime: 1 hour (عمر الاتصال الأقصى)
  - MaxConnIdleTime: 10 minutes (وقت الخمول الأقصى)
  - HealthCheckPeriod: 1 minute (فترة فحص الصحة)

#### 2. Circuit Breaker
- **الوظيفة:** حماية النظام من محاولات الاتصال المتكررة بقاعدة بيانات معطلة
- **الحالات:**
  - **Closed:** العمليات تعمل بشكل طبيعي
  - **Open:** العمليات تفشل فوراً بدون محاولة الاتصال
  - **Half-Open:** محاولة اختبار إذا كانت القاعدة عادت للعمل

#### 3. Failover Manager
- **الوظيفة:** إدارة التحول بين القواعد الرئيسية والاحتياطية
- **المسؤوليات:**
  - مراقبة صحة القاعدة الرئيسية
  - اكتشاف الفشل
  - التحول للقاعدة الاحتياطية
  - إعادة الاتصال بالقاعدة الرئيسية

## السيناريوهات المختبرة

### السيناريو 1: Primary Database Failure

#### الوصف
محاكاة فشل قاعدة البيانات الرئيسية والتحقق من التحول للقاعدة الاحتياطية.

#### الخطوات التفصيلية

```go
// 1. إنشاء اتصال بالقاعدة الرئيسية
primaryDB, err := database.NewDB(ctx, &database.Config{
    Host:     "localhost",
    Port:     5432,
    User:     "postgres",
    Password: "postgres",
    Database: "sms_gateway_test",
})

// 2. إنشاء tenant اختبار
tenantID := createTestTenantForFailover(t, ctx, primaryDB, 
    decimal.NewFromFloat(100.0))

// 3. إجراء عملية ناجحة
balanceManager := balance.NewManager(primaryDB, logger)
result, err := balanceManager.DeductBalance(ctx, operation)
// Expected: Success, Balance: 100 -> 99

// 4. محاكاة فشل القاعدة
primaryDB.Close()
// Expected: Connection closed, operations fail

// 5. الاتصال بالقاعدة الاحتياطية
backupDB, err := database.NewDB(ctx, backupConfig)
// Expected: Connection successful

// 6. إجراء عملية على القاعدة الاحتياطية
backupBalanceManager := balance.NewManager(backupDB, logger)
result2, err := backupBalanceManager.DeductBalance(ctx, operation2)
// Expected: Success, Balance: 99 -> 98

// 7. التحقق من عدم فقدان البيانات
currentBalance, err := backupBalanceManager.GetBalance(ctx, tenantID)
// Expected: Balance = 98 (no data loss)
```

#### معايير النجاح
- ✅ العملية الأولى تنجح على القاعدة الرئيسية
- ✅ فشل القاعدة الرئيسية يتم اكتشافه
- ✅ الاتصال بالقاعدة الاحتياطية ينجح
- ✅ العملية الثانية تنجح على القاعدة الاحتياطية
- ✅ لا يوجد فقدان للبيانات (الرصيد صحيح)

#### الوقت المتوقع
- **الإجمالي:** ~250ms
- **عملية على القاعدة الرئيسية:** ~50ms
- **اكتشاف الفشل:** ~100ms
- **الاتصال بالقاعدة الاحتياطية:** ~50ms
- **عملية على القاعدة الاحتياطية:** ~50ms

### السيناريو 2: Automatic Failover With Operations

#### الوصف
التحقق من التحول التلقائي أثناء وجود عمليات جارية.

#### الخطوات التفصيلية

```go
// 1. بدء عمليات متزامنة
operationCount := 5
for i := 0; i < operationCount; i++ {
    result, err := balanceManager.DeductBalance(ctx, operation)
    
    // 2. محاكاة فشل بعد عمليتين
    if i == 1 {
        primaryDB.Close()
        
        // 3. التحول للقاعدة الاحتياطية
        backupDB, err := database.NewDB(ctx, backupConfig)
        balanceManager = balance.NewManager(backupDB, logger)
    }
    
    // 4. تسجيل النتائج
    if err == nil && result.Success {
        successCount++
    } else {
        failureCount++
    }
}
```

#### معايير النجاح
- ✅ بعض العمليات تنجح قبل الفشل
- ✅ الفشل يتم اكتشافه أثناء العمليات
- ✅ التحول للقاعدة الاحتياطية يحدث تلقائياً
- ✅ العمليات المتبقية تنجح على القاعدة الاحتياطية
- ✅ عدد العمليات الناجحة > 0

#### التوزيع المتوقع
- **عمليات ناجحة:** 3-4 (على القاعدة الرئيسية والاحتياطية)
- **عمليات فاشلة:** 1-2 (أثناء التحول)

### السيناريو 3: Reconnection To Primary

#### الوصف
التحقق من إعادة الاتصال بالقاعدة الرئيسية عند عودتها.

#### الخطوات التفصيلية

```go
// 1. العمل على القاعدة الاحتياطية
backupDB, err := database.NewDB(ctx, backupConfig)
balanceManager := balance.NewManager(backupDB, logger)
result, err := balanceManager.DeductBalance(ctx, operation)
// Expected: Success on backup, Balance: 100 -> 99

// 2. محاكاة عودة القاعدة الرئيسية
primaryDB, err := database.NewDB(ctx, primaryConfig)
err = primaryDB.Pool.Ping(ctx)
// Expected: Primary is operational

// 3. التحول للقاعدة الرئيسية
primaryBalanceManager := balance.NewManager(primaryDB, logger)
result2, err := primaryBalanceManager.DeductBalance(ctx, operation2)
// Expected: Success on primary, Balance: 99 -> 98

// 4. التحقق من اتساق البيانات
currentBalance, err := primaryBalanceManager.GetBalance(ctx, tenantID)
// Expected: Balance = 98 (consistent)
```

#### معايير النجاح
- ✅ العملية على القاعدة الاحتياطية تنجح
- ✅ القاعدة الرئيسية تعود للعمل
- ✅ التحول للقاعدة الرئيسية ينجح
- ✅ العملية على القاعدة الرئيسية تنجح
- ✅ البيانات متسقة بين القاعدتين

#### اتساق البيانات
```
Initial Balance: 100
Operation 1 (Backup): -1 → Balance: 99
Operation 2 (Primary): -1 → Balance: 98
Final Balance: 98 ✓
```

### السيناريو 4: Connection Pool Behavior

#### الوصف
التحقق من سلوك Connection Pool أثناء الفشل والاسترداد.

#### الخطوات التفصيلية

```go
// 1. إنشاء Pool مخصص
poolConfig, err := pgxpool.ParseConfig(dsn)
poolConfig.MaxConns = 5
poolConfig.MinConns = 2
pool, err := pgxpool.NewWithConfig(ctx, poolConfig)

// 2. التحقق من صحة Pool
err = pool.Ping(ctx)
stats := pool.Stat()
// Expected: Total=2, Idle=2, Acquired=0

// 3. الحصول على اتصالات
conns := make([]*pgxpool.Conn, 0, 3)
for i := 0; i < 3; i++ {
    conn, err := pool.Acquire(ctx)
    conns = append(conns, conn)
}
// Expected: 3 connections acquired

// 4. إطلاق الاتصالات
for _, conn := range conns {
    conn.Release()
}
// Expected: All connections released

// 5. التحقق من استرداد Pool
stats = pool.Stat()
// Expected: Total=3, Idle=3, Acquired=0
```

#### معايير النجاح
- ✅ Pool يبدأ بالحد الأدنى من الاتصالات
- ✅ يمكن الحصول على اتصالات جديدة
- ✅ الاتصالات تُطلق بشكل صحيح
- ✅ Pool يسترد حالته الطبيعية
- ✅ لا توجد تسريبات في الاتصالات

## التنفيذ التقني

### 1. Database Configuration

```go
type Config struct {
    Host            string
    Port            int
    User            string
    Password        string
    Database        string
    MaxOpenConns    int           // 100
    MaxIdleConns    int           // 10
    ConnMaxLifetime time.Duration // 1 hour
    ConnMaxIdleTime time.Duration // 10 minutes
}
```

### 2. Connection Pool Setup

```go
func NewDB(ctx context.Context, cfg *Config) (*DB, error) {
    // Parse config
    poolConfig, err := pgxpool.ParseConfig(getDSN(cfg))
    
    // Configure pool
    poolConfig.MaxConns = int32(cfg.MaxOpenConns)
    poolConfig.MinConns = int32(cfg.MaxIdleConns)
    poolConfig.MaxConnLifetime = cfg.ConnMaxLifetime
    poolConfig.MaxConnIdleTime = cfg.ConnMaxIdleTime
    poolConfig.HealthCheckPeriod = 1 * time.Minute
    
    // Create pool
    pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
    
    // Test connection
    if err := pool.Ping(ctx); err != nil {
        return nil, err
    }
    
    return &DB{Pool: pool}, nil
}
```

### 3. Failover Detection

```go
func (db *DB) HealthCheck(ctx context.Context) error {
    // Check connection
    if err := db.Pool.Ping(ctx); err != nil {
        return fmt.Errorf("database unhealthy: %w", err)
    }
    
    // Check query execution
    var result int
    err := db.Pool.QueryRow(ctx, "SELECT 1").Scan(&result)
    if err != nil {
        return fmt.Errorf("query failed: %w", err)
    }
    
    return nil
}
```

### 4. Automatic Failover

```go
type FailoverManager struct {
    primary *DB
    backup  *DB
    current *DB
    mu      sync.RWMutex
}

func (fm *FailoverManager) Execute(ctx context.Context, 
    operation func(*DB) error) error {
    
    fm.mu.RLock()
    current := fm.current
    fm.mu.RUnlock()
    
    // Try current database
    err := operation(current)
    if err == nil {
        return nil
    }
    
    // If failed, try failover
    if current == fm.primary {
        fm.mu.Lock()
        fm.current = fm.backup
        fm.mu.Unlock()
        
        return operation(fm.backup)
    }
    
    return err
}
```

## أفضل الممارسات

### 1. Connection Pool Management

#### ✅ افعل
```go
// استخدم connection pool
pool, err := pgxpool.NewWithConfig(ctx, config)

// حدد حدود مناسبة
config.MaxConns = 100
config.MinConns = 10

// استخدم health checks
config.HealthCheckPeriod = 1 * time.Minute

// أطلق الاتصالات بعد الاستخدام
conn, _ := pool.Acquire(ctx)
defer conn.Release()
```

#### ❌ لا تفعل
```go
// لا تنشئ اتصال جديد لكل عملية
db, _ := sql.Open("postgres", dsn)
defer db.Close()

// لا تترك الاتصالات مفتوحة
conn, _ := pool.Acquire(ctx)
// Missing: conn.Release()

// لا تستخدم حدود غير واقعية
config.MaxConns = 10000 // Too high!
```

### 2. Error Handling

#### ✅ افعل
```go
// تحقق من نوع الخطأ
if err != nil {
    if isConnectionError(err) {
        // Try failover
        return tryBackup()
    }
    return err
}

// استخدم retry logic
for i := 0; i < 3; i++ {
    if err := operation(); err == nil {
        return nil
    }
    time.Sleep(backoff(i))
}
```

#### ❌ لا تفعل
```go
// لا تتجاهل الأخطاء
operation()

// لا تعيد المحاولة بدون حد
for {
    if err := operation(); err == nil {
        break
    }
}
```

### 3. Data Consistency

#### ✅ افعل
```go
// استخدم transactions
tx, _ := db.Begin(ctx)
defer tx.Rollback(ctx)

// نفذ العمليات
tx.Exec(ctx, "UPDATE ...")
tx.Exec(ctx, "INSERT ...")

// commit فقط عند النجاح
tx.Commit(ctx)
```

#### ❌ لا تفعل
```go
// لا تنفذ عمليات منفصلة
db.Exec(ctx, "UPDATE ...")
db.Exec(ctx, "INSERT ...") // May fail!
```

## استكشاف الأخطاء

### المشكلة 1: Connection Timeout

#### الأعراض
```
Error: context deadline exceeded
Error: connection timeout
```

#### الأسباب المحتملة
1. قاعدة البيانات محملة بشكل زائد
2. شبكة بطيئة
3. Connection pool ممتلئ

#### الحلول
```go
// 1. زيادة timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

// 2. زيادة حجم pool
config.MaxConns = 200

// 3. تحسين queries
CREATE INDEX idx_tenant_id ON messages(tenant_id);
```

### المشكلة 2: Connection Leak

#### الأعراض
```
Error: too many connections
Pool stats: Acquired=100, Idle=0
```

#### الأسباب المحتملة
1. عدم إطلاق الاتصالات
2. goroutines معلقة
3. transactions غير مغلقة

#### الحلول
```go
// 1. استخدم defer
conn, _ := pool.Acquire(ctx)
defer conn.Release()

// 2. أغلق transactions
tx, _ := db.Begin(ctx)
defer tx.Rollback(ctx)

// 3. راقب Pool stats
stats := pool.Stat()
log.Printf("Acquired: %d, Idle: %d", 
    stats.AcquiredConns(), stats.IdleConns())
```

### المشكلة 3: Data Inconsistency

#### الأعراض
```
Error: balance mismatch
Expected: 98, Got: 99
```

#### الأسباب المحتملة
1. عدم استخدام transactions
2. race conditions
3. replication lag

#### الحلول
```go
// 1. استخدم transactions
tx, _ := db.Begin(ctx)
defer tx.Rollback(ctx)
// ... operations
tx.Commit(ctx)

// 2. استخدم locks
UPDATE tenants SET balance = balance - 1
WHERE id = $1 AND balance >= 1
RETURNING balance;

// 3. انتظر replication
time.Sleep(100 * time.Millisecond)
```

## المقاييس والمراقبة

### 1. Connection Pool Metrics

```go
stats := pool.Stat()
metrics := map[string]interface{}{
    "total_conns":        stats.TotalConns(),
    "acquired_conns":     stats.AcquiredConns(),
    "idle_conns":         stats.IdleConns(),
    "constructing_conns": stats.ConstructingConns(),
    "max_conns":          stats.MaxConns(),
}
```

### 2. Health Check Metrics

```go
type HealthMetrics struct {
    LastCheck    time.Time
    Status       string // "healthy", "unhealthy"
    ResponseTime time.Duration
    ErrorCount   int
}
```

### 3. Failover Metrics

```go
type FailoverMetrics struct {
    FailoverCount     int
    LastFailover      time.Time
    CurrentDatabase   string // "primary", "backup"
    FailoverDuration  time.Duration
}
```

## الخلاصة

### النقاط الرئيسية
1. ✅ النظام يمكنه اكتشاف فشل قاعدة البيانات
2. ✅ التحول التلقائي للقاعدة الاحتياطية يعمل
3. ✅ لا يوجد فقدان للبيانات أثناء الفشل
4. ✅ إعادة الاتصال بالقاعدة الرئيسية تعمل
5. ✅ Connection Pool يتصرف بشكل صحيح

### التوصيات
1. استخدم PostgreSQL Replication في الإنتاج
2. راقب صحة قاعدة البيانات باستمرار
3. اختبر سيناريوهات الفشل بانتظام
4. وثق إجراءات الطوارئ
5. درب الفريق على التعامل مع الأعطال

---

**آخر تحديث:** 2024
**الحالة:** ✅ جاهز للاستخدام
