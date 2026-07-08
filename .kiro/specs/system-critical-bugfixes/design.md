# System Critical Bugfixes - Technical Design Document

## Overview

هذا المستند يحدد التصميم الفني لإصلاح 30 مشكلة حرجة في نظام SMS Gateway. المشاكل موزعة على 6 فئات: حرجة (4)، عالية الأولوية (6)، متوسطة الأولوية (5)، أمنية (5)، أداء (5)، وتكامل (5).

الاستراتيجية: إصلاح منهجي يركز على الأخطاء الحرجة أولاً، مع ضمان عدم كسر الوظائف الموجودة. كل إصلاح يتضمن:
- تحديد شرط الخطأ (Bug Condition)
- السلوك المتوقع (Expected Behavior)
- التغييرات المطلوبة في الكود
- استراتيجية الاختبار (Exploratory → Fix → Preservation)

## Glossary

- **Bug_Condition (C)**: الشرط الذي يؤدي لظهور الخطأ
- **Property (P)**: السلوك المطلوب عند حدوث شرط الخطأ
- **Preservation**: السلوكيات الموجودة التي يجب أن تبقى دون تغيير
- **Fast Processor**: معالج الرسائل عالي الأداء في `fast_processor.go`
- **Message Queue**: نظام قوائم الانتظار في Redis لإدارة الرسائل
- **PgxPool**: مجموعة اتصالات PostgreSQL للأداء العالي
- **Channel Cache**: التخزين المؤقت لقنوات الإرسال النشطة
- **Pricing Cache**: التخزين المؤقت لحسابات تكلفة الرسائل
- **Custom ID System**: نظام المعرفات المخصصة المولدة من Redis
- **RLS (Row Level Security)**: أمان مستوى الصف في PostgreSQL
- **CSRF**: Cross-Site Request Forgery
- **XSS**: Cross-Site Scripting

---

## 🔴 Critical Bugs (1-4)

### Bug 1: Empty Active Tenants List

#### Bug Condition


المشكلة تحدث عندما يتم استدعاء `getActiveTenants()` في `fast_processor.go` (line 329).

**Formal Specification:**
```
FUNCTION isBugCondition_1(input)
  INPUT: context for getActiveTenants() call
  OUTPUT: boolean
  
  RETURN getActiveTenants() is called
         AND Redis set "alhudhud:active:tenants" exists with tenant UUIDs
         AND function returns empty list []
END FUNCTION
```

**Examples:**
- Tenant A has 100 pending messages in queue → getActiveTenants() returns [] → messages never processed
- Tenant B enqueues new message → tenant not added to active set → processor never picks it up
- System has 5 active tenants with messages → all return empty → complete processing halt

#### Expected Behavior

**Correct Behavior:**
عندما يتم استدعاء `getActiveTenants()`، يجب أن يقوم النظام بالاستعلام عن Redis set `alhudhud:active:tenants` ويعيد قائمة بمعرفات المستأجرين الذين لديهم رسائل معلقة.

**Preservation Requirements:**
- معالجة الرسائل الناجحة للمستأجرين النشطين يجب أن تستمر بنفس الطريقة
- آلية إضافة المستأجرين إلى القائمة النشطة عند إضافة رسائل جديدة يجب أن تبقى كما هي
- أداء الاستعلام يجب أن يبقى سريعًا (< 1ms)

#### Hypothesized Root Cause

1. **Function Not Implemented**: الدالة تعيد قائمة فارغة مباشرة (line 329: `return []uuid.UUID{}, nil`)
2. **Redis Set Not Populated**: عند إضافة رسائل للقائمة، لا يتم تحديث set المستأجرين النشطين
3. **Wrong Redis Key**: الدالة تبحث عن مفتاح خاطئ في Redis

#### Fix Implementation

**File**: `services/backend/internal/worker/fast_processor.go`

**Function**: `getActiveTenants()`

**Changes Required:**


1. **Implement Redis Query**: استبدال `return []uuid.UUID{}, nil` بالاستعلام الفعلي:
```go
func (fp *FastMessageProcessor) getActiveTenants(ctx context.Context) ([]uuid.UUID, error) {
    key := fmt.Sprintf("%s:active:tenants", redis.AppNamespace)
    members, err := fp.messageQueue.client.Client.SMembers(ctx, key).Result()
    if err != nil {
        return nil, fmt.Errorf("get active tenants: %w", err)
    }
    
    tenants := make([]uuid.UUID, 0, len(members))
    for _, member := range members {
        tenantID, err := uuid.Parse(member)
        if err != nil {
            fp.logger.Warn("Invalid tenant UUID in active set", zap.String("value", member))
            continue
        }
        tenants = append(tenants, tenantID)
    }
    
    return tenants, nil
}
```

2. **Update Enqueue Logic**: في `message_queue.go`، تحديث `Enqueue()` لإضافة المستأجر للقائمة النشطة:
```go
func (mq *MessageQueue) Enqueue(ctx context.Context, item *MessageQueueItem) error {
    // ... existing code ...
    
    // Add tenant to active set
    activeKey := fmt.Sprintf("%s:active:tenants", AppNamespace)
    _ = mq.client.Client.SAdd(ctx, activeKey, item.TenantID.String()).Err()
    
    return mq.client.Client.LPush(ctx, queueKey, data).Err()
}
```

3. **Cleanup Logic**: إزالة المستأجر من القائمة النشطة عندما تصبح قوائمه فارغة:
```go
// في نهاية processBatch()
if queueSize == 0 {
    activeKey := fmt.Sprintf("%s:active:tenants", redis.AppNamespace)
    _ = fp.messageQueue.client.Client.SRem(ctx, activeKey, tenantID.String()).Err()
}
```

---

### Bug 2: Nil PgxPool Crash

#### Bug Condition

المشكلة تحدث عندما يتم الوصول إلى `db.PgxPool` في أي مكان بالنظام.

**Formal Specification:**
```
FUNCTION isBugCondition_2(input)
  INPUT: code attempting to access db.PgxPool
  OUTPUT: boolean
  
  RETURN db.PgxPool is accessed
         AND db.PgxPool == nil
         AND system crashes with nil pointer dereference
END FUNCTION
```

**Examples:**
- `fast_processor.go` line 138: `tx, err := db.PgxPool.Begin(ctx)` → panic: nil pointer
- `fast_processor.go` line 244: `db.PgxPool.Exec(ctx, query, messageIDs)` → panic: nil pointer
- Any transaction operation → immediate crash


#### Expected Behavior

**Correct Behavior:**
عندما يتم تهيئة قاعدة البيانات في `db/gorm.go`، يجب أن يقوم النظام بتهيئة كل من `GormDB` و `PgxPool` باستخدام نفس إعدادات الاتصال.

**Preservation Requirements:**
- اتصالات GORM الموجودة يجب أن تستمر بالعمل بنفس الطريقة
- استعلامات GORM الموجودة يجب ألا تتأثر
- أداء الاتصالات يجب أن يبقى كما هو أو يتحسن

#### Hypothesized Root Cause

1. **Missing Initialization**: `PgxPool` غير مهيأ في `InitGormDB()` function
2. **Wrong Package**: `PgxPool` معرف في package خاطئ أو غير مستورد
3. **Initialization Order**: `PgxPool` يتم تهيئته بعد استخدامه

#### Fix Implementation

**File**: `services/backend/internal/db/gorm.go`

**Changes Required:**

1. **Add PgxPool Variable**:
```go
import (
    "github.com/jackc/pgx/v5/pgxpool"
    // ... existing imports
)

var GormDB *gorm.DB
var PgxPool *pgxpool.Pool  // Add this
```

2. **Initialize PgxPool in InitGormDB()**:
```go
func InitGormDB() {
    if GormDB != nil {
        return
    }
    log := logger.GetLogger()

    // ... existing .env loading code ...

    dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s timezone=UTC",
        os.Getenv("DB_HOST"),
        os.Getenv("DB_USER"),
        os.Getenv("DB_PASSWORD"),
        os.Getenv("DB_NAME"),
        os.Getenv("DB_PORT"),
        os.Getenv("DB_SSLMODE"),
    )

    // Initialize GORM
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    if err != nil {
        log.Fatal("Failed to connect to database", zap.Error(err))
    }
    GormDB = db
    log.Info("Connection to PostgreSQL established successfully via GORM")

    // Initialize PgxPool
    pgxDSN := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
        os.Getenv("DB_USER"),
        os.Getenv("DB_PASSWORD"),
        os.Getenv("DB_HOST"),
        os.Getenv("DB_PORT"),
        os.Getenv("DB_NAME"),
        os.Getenv("DB_SSLMODE"),
    )
    
    poolConfig, err := pgxpool.ParseConfig(pgxDSN)
    if err != nil {
        log.Fatal("Failed to parse pgxpool config", zap.Error(err))
    }
    
    // Configure pool settings
    poolConfig.MaxConns = 50
    poolConfig.MinConns = 10
    poolConfig.MaxConnLifetime = time.Hour
    poolConfig.MaxConnIdleTime = 30 * time.Minute
    
    pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
    if err != nil {
        log.Fatal("Failed to create pgxpool", zap.Error(err))
    }
    
    PgxPool = pool
    log.Info("PgxPool initialized successfully",
        zap.Int32("max_conns", poolConfig.MaxConns),
        zap.Int32("min_conns", poolConfig.MinConns),
    )

    // Run Migrations
    migrate()
}
```

3. **Add Cleanup Function**:
```go
func Close() {
    if PgxPool != nil {
        PgxPool.Close()
    }
}
```

---

### Bug 3: Mock Channel Selection

#### Bug Condition

المشكلة تحدث عندما يتم استدعاء `getOrSelectChannel()` في `fast_processor.go` (lines 248-262).

**Formal Specification:**
```
FUNCTION isBugCondition_3(input)
  INPUT: tenantID for channel selection
  OUTPUT: boolean
  
  RETURN getOrSelectChannel(tenantID) is called
         AND real channels exist in database for tenant
         AND function returns mock channel with uuid.New()
         AND messages are routed to non-existent channel
END FUNCTION
```

**Examples:**
- Tenant has 3 active channels in DB → function returns random UUID → messages fail to send
- Channel priority and load balancing ignored → all messages go to mock channel
- Channel online status not checked → messages sent to offline channels


#### Expected Behavior

**Correct Behavior:**
عندما يتم استدعاء `getOrSelectChannel()`، يجب أن يقوم النظام بالاستعلام عن قاعدة البيانات للحصول على القنوات المتاحة والنشطة، التحقق من الذاكرة المؤقتة أولاً، والاختيار بناءً على الأولوية والحمل.

**Preservation Requirements:**
- آلية التخزين المؤقت للقنوات يجب أن تستمر بالعمل
- أداء اختيار القنوات يجب أن يبقى سريعًا
- منطق الأولوية الموجود يجب أن يُحترم

#### Hypothesized Root Cause

1. **Incomplete Implementation**: الدالة تحتوي على تعليق "simplified - should use routing engine" وتعيد بيانات وهمية
2. **Missing Database Query**: لا يوجد استعلام فعلي لقاعدة البيانات
3. **Routing Engine Not Integrated**: محرك التوجيه غير مدمج بعد

#### Fix Implementation

**File**: `services/backend/internal/worker/fast_processor.go`

**Function**: `getOrSelectChannel()`

**Changes Required:**

```go
func (fp *FastMessageProcessor) getOrSelectChannel(ctx context.Context, tenantID uuid.UUID) (*redis.CachedChannel, error) {
    // Try cache first
    if fp.channelCache != nil {
        cached, err := fp.channelCache.Get(ctx, tenantID)
        if err == nil && cached.IsOnline {
            return cached, nil
        }
    }

    // Query database for available channels
    query := `
        SELECT c.id, c.type, c.priority, c.is_online, c.gateway_id, c.current_load
        FROM channels c
        INNER JOIN gateways g ON c.gateway_id = g.id
        WHERE g.tenant_id = $1 
          AND c.is_online = true 
          AND c.status = 'ACTIVE'
        ORDER BY c.priority ASC, c.current_load ASC
        LIMIT 1
    `
    
    var channel struct {
        ID          uuid.UUID
        Type        string
        Priority    int
        IsOnline    bool
        GatewayID   uuid.UUID
        CurrentLoad int
    }
    
    err := db.PgxPool.QueryRow(ctx, query, tenantID).Scan(
        &channel.ID,
        &channel.Type,
        &channel.Priority,
        &channel.IsOnline,
        &channel.GatewayID,
        &channel.CurrentLoad,
    )
    
    if err != nil {
        return nil, fmt.Errorf("no available channels for tenant %s: %w", tenantID, err)
    }
    
    cachedChannel := &redis.CachedChannel{
        ChannelID: channel.ID,
        GatewayID: channel.GatewayID,
        Type:      channel.Type,
        Priority:  channel.Priority,
        IsOnline:  channel.IsOnline,
        CachedAt:  time.Now(),
    }
    
    // Cache it
    if fp.channelCache != nil {
        _ = fp.channelCache.Set(ctx, tenantID, cachedChannel)
    }
    
    return cachedChannel, nil
}
```

---

### Bug 4: Incomplete Analytics Calculation

#### Bug Condition

المشكلة تحدث عندما يتم استدعاء `GetCurrentLoad()` في `redis/analytics.go`.

**Formal Specification:**
```
FUNCTION isBugCondition_4(input)
  INPUT: context for GetCurrentLoad() call
  OUTPUT: boolean
  
  RETURN GetCurrentLoad() is called
         AND actual messages exist in system
         AND function does not count actual messages
         AND returns incomplete metrics (totalMessages = 0)
END FUNCTION
```

**Examples:**
- System has 10,000 messages processed → GetCurrentLoad() returns MessageCount: 0
- Analytics dashboard shows 0 messages → misleading metrics
- Load detection fails → no pressure alerts triggered

#### Expected Behavior

**Correct Behavior:**
عندما يتم استدعاء `GetCurrentLoad()`، يجب أن يقوم النظام بمسح مفاتيح Redis للعدادات والرسائل، تجميعها، وإرجاع مقاييس دقيقة للحمل.

**Preservation Requirements:**
- حساب عدد الأحداث يجب أن يستمر بالعمل
- أداء الاستعلام يجب أن يبقى سريعًا
- صيغة مفاتيح Redis يجب أن تبقى متوافقة

#### Hypothesized Root Cause

1. **Incomplete Implementation**: الكود يحتوي على تعليق "We need to check all events created with this counter" لكن لا ينفذه
2. **Missing Message Count Query**: لا يوجد استعلام لعدد الرسائل الفعلية
3. **Wrong Key Pattern**: نمط المفاتيح المستخدم لا يطابق البيانات الفعلية


#### Fix Implementation

**File**: `services/backend/internal/redis/analytics.go`

**Function**: `GetCurrentLoad()`

**Changes Required:**

```go
func (la *LoadAnalytics) GetCurrentLoad(ctx context.Context) (*LoadMetrics, error) {
    now := time.Now()
    timestamp := formatTimestamp(now)
    
    var totalEvents int64
    var totalMessages int64
    var maxMessages int64
    var nearLimit int64
    
    // 1. Count events from counters
    counterPattern := fmt.Sprintf("%s:event:counter:%s:*", AppNamespace, timestamp)
    iter := la.client.Client.Scan(ctx, 0, counterPattern, 100).Iterator()
    for iter.Next(ctx) {
        count, err := la.client.Client.Get(ctx, iter.Val()).Int64()
        if err == nil {
            totalEvents += count
        }
    }
    if err := iter.Err(); err != nil {
        return nil, err
    }
    
    // 2. Count messages from event metadata
    metadataPattern := fmt.Sprintf("%s:event:metadata:EV%s*", AppNamespace, timestamp)
    iter = la.client.Client.Scan(ctx, 0, metadataPattern, 100).Iterator()
    for iter.Next(ctx) {
        key := iter.Val()
        msgCount, err := la.client.Client.HGet(ctx, key, "message_count").Int64()
        if err == nil {
            totalMessages += msgCount
            if msgCount > maxMessages {
                maxMessages = msgCount
            }
            if msgCount >= 900 {
                nearLimit++
            }
        }
    }
    if err := iter.Err(); err != nil {
        return nil, err
    }
    
    // 3. Count messages in queues
    queuePattern := fmt.Sprintf("%s:queue:messages:*", AppNamespace)
    iter = la.client.Client.Scan(ctx, 0, queuePattern, 100).Iterator()
    for iter.Next(ctx) {
        queueSize, err := la.client.Client.LLen(ctx, iter.Val()).Result()
        if err == nil {
            totalMessages += queueSize
        }
    }
    if err := iter.Err(); err != nil {
        return nil, err
    }
    
    avgMsgPerEvent := float64(0)
    if totalEvents > 0 {
        avgMsgPerEvent = float64(totalMessages) / float64(totalEvents)
    }
    
    return &LoadMetrics{
        Timestamp:      now,
        EventCount:     totalEvents,
        MessageCount:   totalMessages,
        AvgMsgPerEvent: avgMsgPerEvent,
        MaxMsgPerEvent: maxMessages,
        NearLimitCount: nearLimit,
    }, nil
}
```

---

## 🟠 High Priority Bugs (5-10)

### Bug 5: Silent Error Ignoring

#### Bug Condition

المشكلة تحدث عندما تحدث أخطاء في وظائف مختلفة ويتم تجاهلها باستخدام نمط `_ = `.

**Formal Specification:**
```
FUNCTION isBugCondition_5(input)
  INPUT: error-returning function call
  OUTPUT: boolean
  
  RETURN function returns error
         AND error is assigned to _ (ignored)
         AND critical failure is hidden from logs and monitoring
END FUNCTION
```

**Examples:**
- `_ = fp.wsHub.SendMessage()` → WebSocket failure ignored → message lost silently
- `_ = fp.channelCache.Set()` → cache update fails → stale data served
- `_ = pipe.Exec(ctx)` → Redis pipeline fails → data inconsistency

#### Expected Behavior

**Correct Behavior:**
عندما تحدث أخطاء في العمليات الحرجة، يجب أن يقوم النظام بتسجيل الأخطاء بشكل صحيح ومعالجتها بشكل مناسب (إرجاع، إعادة محاولة، أو فشل بشكل آمن).

**Preservation Requirements:**
- معالجة الأخطاء الموجودة التي تعمل بشكل صحيح يجب أن تبقى كما هي
- أداء النظام يجب ألا يتأثر بالتسجيل الإضافي
- تدفق التنفيذ الطبيعي يجب أن يستمر

#### Hypothesized Root Cause

1. **Quick Development**: الأخطاء تم تجاهلها أثناء التطوير السريع
2. **Non-Critical Assumption**: افتراض أن بعض الأخطاء غير حرجة
3. **Missing Error Handling Strategy**: عدم وجود استراتيجية واضحة لمعالجة الأخطاء


#### Fix Implementation

**Files**: Multiple files with `_ = ` pattern

**Changes Required:**

1. **fast_processor.go** - WebSocket errors:
```go
// Before:
_ = fp.wsHub.SendMessage(ctx, wsMessage)

// After:
if err := fp.wsHub.SendMessage(ctx, wsMessage); err != nil {
    fp.logger.Error("Failed to send message via WebSocket",
        zap.String("message_id", msg.MessageID.String()),
        zap.Error(err),
    )
    failedMessages = append(failedMessages, msg)
}
```

2. **fast_processor.go** - Cache errors:
```go
// Before:
_ = fp.channelCache.Set(ctx, tenantID, channel)

// After:
if err := fp.channelCache.Set(ctx, tenantID, channel); err != nil {
    fp.logger.Warn("Failed to cache channel",
        zap.String("tenant_id", tenantID.String()),
        zap.Error(err),
    )
    // Continue - cache failure is not critical
}
```

3. **pricing_cache.go** - Cache set errors:
```go
// Before:
_ = pc.Set(ctx, encoding, charCount, info)

// After:
if err := pc.Set(ctx, encoding, charCount, info); err != nil {
    // Log but don't fail - cache miss is acceptable
    // Could add metrics here for cache failure rate
}
```

4. **message_queue.go** - Active tenant tracking:
```go
// Before:
_ = mq.client.Client.SAdd(ctx, activeKey, item.TenantID.String()).Err()

// After:
if err := mq.client.Client.SAdd(ctx, activeKey, item.TenantID.String()).Err(); err != nil {
    // Log error but don't fail enqueue
    // Worst case: tenant not in active set, will be added on next enqueue
}
```

**Strategy**: تصنيف الأخطاء إلى:
- **Critical**: يجب إيقاف العملية (return error)
- **Important**: يجب التسجيل والمتابعة (log.Error + continue)
- **Minor**: يمكن التجاهل مع تسجيل اختياري (log.Debug)

---

### Bug 6: Transaction Rollback After Commit

#### Bug Condition

المشكلة تحدث عندما يتم استخدام معاملات قاعدة البيانات مع `defer tx.Rollback()` و `tx.Commit()` صريح.

**Formal Specification:**
```
FUNCTION isBugCondition_6(input)
  INPUT: database transaction
  OUTPUT: boolean
  
  RETURN transaction uses defer tx.Rollback()
         AND transaction calls tx.Commit() explicitly
         AND Rollback() executes after successful Commit()
         AND causes transaction management issues
END FUNCTION
```

**Examples:**
- Transaction commits successfully → defer Rollback() runs → error logged unnecessarily
- Commit fails → Rollback() runs → correct behavior but error not checked
- Multiple deferred operations → execution order issues

#### Expected Behavior

**Correct Behavior:**
عندما يتم استخدام معاملات قاعدة البيانات، يجب أن يقوم النظام باستخدام نمط `defer tx.Rollback(ctx)` والتحقق من أخطاء الـ commit، مع ضمان تنفيذ الـ rollback فقط إذا فشل الـ commit.

**Preservation Requirements:**
- المعاملات الناجحة يجب أن تُحفظ بشكل ذري
- المعاملات الفاشلة يجب أن يتم التراجع عنها بالكامل
- أداء المعاملات يجب أن يبقى كما هو

#### Hypothesized Root Cause

1. **Standard Pattern Misuse**: استخدام نمط defer الشائع بشكل خاطئ
2. **Missing Error Check**: عدم التحقق من خطأ Commit
3. **Pgx Behavior**: pgx يتجاهل Rollback بعد Commit ناجح، لكن يسجل تحذير

#### Fix Implementation

**File**: `services/backend/internal/worker/fast_processor.go`

**Function**: `processBatchWithTransaction()`

**Changes Required:**

```go
func (fp *FastMessageProcessor) processBatchWithTransaction(ctx context.Context, tenantID uuid.UUID, messages []*redis.MessageQueueItem) error {
    startTime := time.Now()

    channel, err := fp.getOrSelectChannel(ctx, tenantID)
    if err != nil {
        return fmt.Errorf("select channel: %w", err)
    }

    tx, err := db.PgxPool.Begin(ctx)
    if err != nil {
        return fmt.Errorf("begin transaction: %w", err)
    }
    
    // Proper defer pattern: rollback only if commit not called
    committed := false
    defer func() {
        if !committed {
            if rbErr := tx.Rollback(ctx); rbErr != nil {
                fp.logger.Error("Failed to rollback transaction", zap.Error(rbErr))
            }
        }
    }()

    // ... process messages ...

    // Commit and mark as committed
    if err := tx.Commit(ctx); err != nil {
        return fmt.Errorf("commit transaction: %w", err)
    }
    committed = true

    // ... rest of function ...
}
```

**Alternative Pattern** (cleaner):
```go
defer func() {
    if err := tx.Rollback(ctx); err != nil && err != pgx.ErrTxClosed {
        fp.logger.Error("Failed to rollback transaction", zap.Error(err))
    }
}()

// ... process ...

if err := tx.Commit(ctx); err != nil {
    return fmt.Errorf("commit transaction: %w", err)
}
```


---

### Bug 7: No Balance Verification Before Deduction

#### Bug Condition

المشكلة تحدث عندما يتم خصم الرصيد في `processBatchWithTransaction()` دون التحقق من كفاية الرصيد.

**Formal Specification:**
```
FUNCTION isBugCondition_7(input)
  INPUT: balance deduction operation
  OUTPUT: boolean
  
  RETURN DeductBalanceTx() is called
         AND tenant balance < required amount
         AND deduction proceeds anyway
         AND results in negative balance
END FUNCTION
```

**Examples:**
- Tenant has 10 SAR balance → batch costs 15 SAR → deduction succeeds → balance = -5 SAR
- Multiple concurrent batches → race condition → multiple negative balances
- No validation → accounting inconsistency

#### Expected Behavior

**Correct Behavior:**
عندما يتم محاولة خصم الرصيد، يجب أن يقوم النظام بالتحقق من وجود رصيد كافٍ قبل الخصم، وإرجاع خطأ إذا كان الرصيد غير كافٍ.

**Preservation Requirements:**
- خصم الرصيد الناجح مع رصيد كافٍ يجب أن يستمر بالعمل
- إنشاء معاملات المحاسبة يجب أن يبقى كما هو
- أداء عمليات الرصيد يجب أن يبقى سريعًا

#### Hypothesized Root Cause

1. **Missing Validation**: دالة `DeductBalanceTx()` لا تتحقق من الرصيد قبل الخصم
2. **Assumed Pre-Check**: افتراض أن التحقق يتم في مكان آخر
3. **Race Condition**: عدم استخدام قفل للتحقق والخصم الذري

#### Fix Implementation

**File**: `services/backend/internal/balance/manager.go` (assumed location)

**Function**: `DeductBalanceTx()`

**Changes Required:**

```go
func (bm *BalanceManager) DeductBalanceTx(ctx context.Context, tx pgx.Tx, op BalanceOperation) (*BalanceTransaction, error) {
    // 1. Get current balance with row lock
    var currentBalance decimal.Decimal
    query := `
        SELECT balance 
        FROM tenant_balances 
        WHERE tenant_id = $1 
        FOR UPDATE
    `
    err := tx.QueryRow(ctx, query, op.TenantID).Scan(&currentBalance)
    if err != nil {
        return nil, fmt.Errorf("get balance: %w", err)
    }
    
    // 2. Verify sufficient balance
    if currentBalance.LessThan(op.Amount) {
        return nil, fmt.Errorf("insufficient balance: have %s, need %s", 
            currentBalance.String(), op.Amount.String())
    }
    
    // 3. Deduct balance
    newBalance := currentBalance.Sub(op.Amount)
    updateQuery := `
        UPDATE tenant_balances 
        SET balance = $1, updated_at = NOW() 
        WHERE tenant_id = $2
    `
    _, err = tx.Exec(ctx, updateQuery, newBalance, op.TenantID)
    if err != nil {
        return nil, fmt.Errorf("update balance: %w", err)
    }
    
    // 4. Create transaction record
    txRecord := &BalanceTransaction{
        ID:        uuid.New(),
        TenantID:  op.TenantID,
        GatewayID: op.GatewayID,
        Type:      "DEDUCTION",
        Amount:    op.Amount,
        Balance:   newBalance,
        Reason:    op.Reason,
        CreatedBy: op.CreatedBy,
        CreatedAt: time.Now(),
    }
    
    insertQuery := `
        INSERT INTO balance_transactions 
        (id, tenant_id, gateway_id, type, amount, balance, reason, created_by, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
    `
    _, err = tx.Exec(ctx, insertQuery,
        txRecord.ID, txRecord.TenantID, txRecord.GatewayID,
        txRecord.Type, txRecord.Amount, txRecord.Balance,
        txRecord.Reason, txRecord.CreatedBy, txRecord.CreatedAt,
    )
    if err != nil {
        return nil, fmt.Errorf("insert transaction: %w", err)
    }
    
    return txRecord, nil
}
```

**Update fast_processor.go** to handle insufficient balance:
```go
_, err = fp.balanceManager.DeductBalanceTx(ctx, tx, balanceOp)
if err != nil {
    fp.logger.Warn("Insufficient balance for batch",
        zap.String("tenant_id", tenantID.String()),
        zap.Error(err),
    )
    // Update all messages to FAILED with reason
    fp.updateBatchStatus(ctx, tx, messageIDs, "FAILED")
    tx.Commit(ctx) // Commit the FAILED status
    return fmt.Errorf("insufficient balance: %w", err)
}
```

---

### Bug 8: Nil wsHub Crash

#### Bug Condition

المشكلة تحدث عندما يتم استدعاء `fp.wsHub.SendMessage()` في `fast_processor.go` وقد يكون wsHub غير مهيأ.

**Formal Specification:**
```
FUNCTION isBugCondition_8(input)
  INPUT: WebSocket message send operation
  OUTPUT: boolean
  
  RETURN fp.wsHub.SendMessage() is called
         AND fp.wsHub == nil
         AND system crashes with nil pointer dereference
END FUNCTION
```

**Examples:**
- Fast processor initialized without wsHub → SendMessage() called → panic
- wsHub initialization fails → processor continues → crash on first message
- Testing environment without WebSocket → crash


#### Expected Behavior

**Correct Behavior:**
عندما يتم استدعاء `fp.wsHub.SendMessage()`، يجب أن يقوم النظام بالتحقق من أن wsHub ليس nil قبل استدعاء الدوال عليه.

**Preservation Requirements:**
- إرسال الرسائل عبر WebSocket الناجح يجب أن يستمر بالعمل
- أداء إرسال الرسائل يجب أن يبقى كما هو
- معالجة الأخطاء الموجودة يجب أن تبقى

#### Hypothesized Root Cause

1. **Optional Dependency**: wsHub اختياري في بعض البيئات
2. **Initialization Order**: wsHub يتم تهيئته بعد FastProcessor
3. **Missing Nil Check**: عدم وجود فحص nil قبل الاستخدام

#### Fix Implementation

**File**: `services/backend/internal/worker/fast_processor.go`

**Changes Required:**

1. **Add nil check in SendMessage calls**:
```go
// In processBatchWithTransaction()
for _, msg := range messages {
    wsMessage := ws.WebSocketMessage{
        Type:        "MESSAGE",
        MessageID:   msg.MessageID.String(),
        TenantID:    tenantID,
        ChannelID:   channel.ChannelID,
        PhoneNumber: msg.PhoneNumber,
        Content:     msg.Content,
        Timestamp:   time.Now(),
    }

    // Add nil check
    if fp.wsHub == nil {
        fp.logger.Warn("WebSocket hub not initialized, message cannot be sent",
            zap.String("message_id", msg.MessageID.String()),
        )
        failedMessages = append(failedMessages, msg)
        continue
    }

    if err := fp.wsHub.SendMessage(ctx, wsMessage); err != nil {
        fp.logger.Error("Failed to send message via WebSocket",
            zap.String("message_id", msg.MessageID.String()),
            zap.Error(err),
        )
        failedMessages = append(failedMessages, msg)
    } else {
        successCount++
    }
}
```

2. **Add validation in constructor**:
```go
func NewFastMessageProcessor(
    messageQueue *redis.MessageQueue,
    channelCache *redis.ChannelCache,
    balanceManager balance.Manager,
    wsHub *ws.Hub,
    logger *zap.Logger,
) *FastMessageProcessor {
    if logger == nil {
        panic("logger is required")
    }
    if messageQueue == nil {
        panic("messageQueue is required")
    }
    if balanceManager == nil {
        panic("balanceManager is required")
    }
    // wsHub is optional - log warning if nil
    if wsHub == nil {
        logger.Warn("WebSocket hub not provided - messages will not be sent to devices")
    }
    
    return &FastMessageProcessor{
        messageQueue:   messageQueue,
        channelCache:   channelCache,
        balanceManager: balanceManager,
        wsHub:          wsHub,
        logger:         logger,
        batchSize:      100,
        workerCount:    10,
        stopChan:       make(chan struct{}),
    }
}
```

---

### Bug 9: No Redis Retry Logic

#### Bug Condition

المشكلة تحدث عندما تفشل عمليات Redis ولا يتم إعادة المحاولة.

**Formal Specification:**
```
FUNCTION isBugCondition_9(input)
  INPUT: Redis operation
  OUTPUT: boolean
  
  RETURN Redis operation fails with transient error
         AND no retry is attempted
         AND data is lost or state becomes inconsistent
END FUNCTION
```

**Examples:**
- Network blip → Redis LPUSH fails → message lost
- Redis temporarily overloaded → cache set fails → stale data served
- Connection timeout → enqueue fails → message never processed

#### Expected Behavior

**Correct Behavior:**
عندما تفشل عمليات Redis، يجب أن يقوم النظام بتنفيذ منطق إعادة المحاولة مع تأخير أسي للأخطاء العابرة.

**Preservation Requirements:**
- عمليات Redis الناجحة يجب أن تستمر بنفس الأداء
- عمليات Redis الفاشلة بشكل دائم يجب أن تفشل بسرعة
- ترتيب العمليات يجب أن يُحفظ

#### Hypothesized Root Cause

1. **No Retry Strategy**: عدم وجود استراتيجية إعادة محاولة مطبقة
2. **Assumed Reliability**: افتراض أن Redis دائمًا متاح
3. **Performance Concern**: قلق من تأثير إعادة المحاولة على الأداء

#### Fix Implementation

**File**: `services/backend/internal/redis/client.go` (create if not exists)

**Add Retry Wrapper**:

```go
package redis

import (
    "context"
    "fmt"
    "time"
    
    "github.com/redis/go-redis/v9"
    "go.uber.org/zap"
)

// RetryConfig holds retry configuration
type RetryConfig struct {
    MaxRetries     int
    InitialBackoff time.Duration
    MaxBackoff     time.Duration
    Multiplier     float64
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() RetryConfig {
    return RetryConfig{
        MaxRetries:     3,
        InitialBackoff: 10 * time.Millisecond,
        MaxBackoff:     1 * time.Second,
        Multiplier:     2.0,
    }
}

// WithRetry wraps a Redis operation with retry logic
func (c *Client) WithRetry(ctx context.Context, operation string, fn func() error) error {
    config := DefaultRetryConfig()
    backoff := config.InitialBackoff
    
    for attempt := 0; attempt <= config.MaxRetries; attempt++ {
        err := fn()
        
        if err == nil {
            return nil
        }
        
        // Check if error is retryable
        if !isRetryableError(err) {
            return err
        }
        
        // Last attempt - return error
        if attempt == config.MaxRetries {
            return fmt.Errorf("operation %s failed after %d attempts: %w", 
                operation, config.MaxRetries+1, err)
        }
        
        // Log retry
        if c.logger != nil {
            c.logger.Warn("Redis operation failed, retrying",
                zap.String("operation", operation),
                zap.Int("attempt", attempt+1),
                zap.Duration("backoff", backoff),
                zap.Error(err),
            )
        }
        
        // Wait before retry
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(backoff):
        }
        
        // Increase backoff
        backoff = time.Duration(float64(backoff) * config.Multiplier)
        if backoff > config.MaxBackoff {
            backoff = config.MaxBackoff
        }
    }
    
    return fmt.Errorf("operation %s exhausted retries", operation)
}

// isRetryableError checks if error is transient and retryable
func isRetryableError(err error) bool {
    if err == nil {
        return false
    }
    
    // Network errors are retryable
    if err == redis.Nil {
        return false // Not found is not retryable
    }
    
    errStr := err.Error()
    retryableErrors := []string{
        "connection refused",
        "connection reset",
        "timeout",
        "temporary failure",
        "LOADING",
        "BUSY",
    }
    
    for _, retryable := range retryableErrors {
        if contains(errStr, retryable) {
            return true
        }
    }
    
    return false
}

func contains(s, substr string) bool {
    return len(s) >= len(substr) && (s == substr || 
        (len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr)))
}
```

**Update message_queue.go to use retry**:
```go
func (mq *MessageQueue) Enqueue(ctx context.Context, item *MessageQueueItem) error {
    if item.EnqueuedAt.IsZero() {
        item.EnqueuedAt = time.Now()
    }

    data, err := json.Marshal(item)
    if err != nil {
        return fmt.Errorf("marshal message: %w", err)
    }

    queueKey := mq.getQueueKey(item.TenantID, item.Priority)
    
    // Use retry wrapper
    return mq.client.WithRetry(ctx, "enqueue", func() error {
        return mq.client.Client.LPush(ctx, queueKey, data).Err()
    })
}
```


---

### Bug 10: Improper Priority Queue Ordering

#### Bug Condition

المشكلة تحدث عندما يتم سحب الرسائل من قوائم الأولوية في `message_queue.go`.

**Formal Specification:**
```
FUNCTION isBugCondition_10(input)
  INPUT: dequeue operation across priority queues
  OUTPUT: boolean
  
  RETURN DequeueBatch() is called
         AND high priority queue has messages
         AND function returns mix of priorities before draining high queue
         AND priority ordering is not properly enforced
END FUNCTION
```

**Examples:**
- High queue has 50 messages, Normal has 100 → DequeueBatch(100) returns 50 high + 50 normal → correct
- High queue has 150 messages → DequeueBatch(100) returns 100 high → correct
- But implementation may not fully drain high before moving to normal

#### Expected Behavior

**Correct Behavior:**
عندما يتم سحب الرسائل من قوائم الأولوية، يجب أن يقوم النظام بتفريغ قائمة الأولوية العالية بالكامل قبل الانتقال إلى الأولوية العادية، ثم المنخفضة.

**Preservation Requirements:**
- ترتيب FIFO داخل كل مستوى أولوية يجب أن يُحفظ
- أداء سحب الرسائل يجب أن يبقى سريعًا
- دفعات الرسائل يجب أن تُعالج بكفاءة

#### Hypothesized Root Cause

1. **Current Implementation Correct**: الكود الحالي يبدو صحيحًا، لكن يحتاج تأكيد
2. **Edge Case**: قد توجد حالات حافة لم تُختبر
3. **Documentation**: قد يكون الكود صحيحًا لكن يحتاج توثيق أفضل

#### Fix Implementation

**File**: `services/backend/internal/redis/message_queue.go`

**Function**: `DequeueBatch()`

**Review and Enhance**:

```go
// DequeueBatch retrieves multiple messages at once with strict priority ordering
// Priority order: 1 (high) → 2 (normal) → 3 (low)
// Fully drains higher priority queues before moving to lower priority
func (mq *MessageQueue) DequeueBatch(ctx context.Context, tenantID uuid.UUID, count int) ([]*MessageQueueItem, error) {
    if count <= 0 {
        count = 100
    }

    items := make([]*MessageQueueItem, 0, count)
    
    // Process priorities in strict order: high → normal → low
    for priority := 1; priority <= 3 && len(items) < count; priority++ {
        queueKey := mq.getQueueKey(tenantID, priority)
        remaining := count - len(items)
        
        // Get remaining count from this priority queue
        results, err := mq.client.Client.RPopCount(ctx, queueKey, remaining).Result()
        if err == redis.Nil {
            // Queue empty, try next priority
            continue
        }
        if err != nil {
            // Return what we have so far
            return items, fmt.Errorf("dequeue from priority %d: %w", priority, err)
        }

        // Parse and add items
        for _, result := range results {
            var item MessageQueueItem
            if err := json.Unmarshal([]byte(result), &item); err != nil {
                // Log invalid item but continue
                continue
            }
            items = append(items, &item)
        }
    }

    return items, nil
}
```

**Add Test to Verify Priority Ordering**:
```go
// Test case to add in message_queue_test.go
func TestDequeueBatch_PriorityOrdering(t *testing.T) {
    // Setup: Enqueue messages with different priorities
    // High: 150 messages
    // Normal: 100 messages
    // Low: 50 messages
    
    // Test 1: Dequeue 100 → should get 100 high priority
    batch1 := DequeueBatch(ctx, tenantID, 100)
    assert.Len(t, batch1, 100)
    for _, msg := range batch1 {
        assert.Equal(t, 1, msg.Priority) // All high priority
    }
    
    // Test 2: Dequeue 100 → should get 50 high + 50 normal
    batch2 := DequeueBatch(ctx, tenantID, 100)
    assert.Len(t, batch2, 100)
    assert.Equal(t, 1, batch2[0].Priority) // First 50 are high
    assert.Equal(t, 2, batch2[50].Priority) // Next 50 are normal
    
    // Test 3: Dequeue 100 → should get 50 normal + 50 low
    batch3 := DequeueBatch(ctx, tenantID, 100)
    assert.Len(t, batch3, 100)
    assert.Equal(t, 2, batch3[0].Priority) // First 50 are normal
    assert.Equal(t, 3, batch3[50].Priority) // Next 50 are low
}
```

---

## 🟡 Medium Priority Bugs (11-15)

### Bug 11: Missing Custom ID Backfill

#### Bug Condition

المشكلة تحدث عندما يتم تشغيل migration `077_add_custom_ids.sql`.

**Formal Specification:**
```
FUNCTION isBugCondition_11(input)
  INPUT: migration execution
  OUTPUT: boolean
  
  RETURN migration 077 runs on existing database
         AND existing records have NULL custom_id values
         AND migration does not backfill these values
         AND production data has incomplete custom_id coverage
END FUNCTION
```

**Examples:**
- Database has 1M existing messages → migration adds column → all have custom_id = NULL
- New messages get custom_id → old messages don't → inconsistent data
- Queries by custom_id fail for old records

#### Expected Behavior

**Correct Behavior:**
عندما يتم تشغيل migration `077_add_custom_ids.sql`، يجب أن يتضمن عبارات UPDATE لملء custom_id للسجلات الموجودة باستخدام دالة توليد.

**Preservation Requirements:**
- السجلات الجديدة يجب أن تحصل على custom_id عند الإنشاء
- الفهارس يجب أن تُنشأ بشكل صحيح
- أداء الاستعلامات يجب أن يبقى سريعًا


#### Hypothesized Root Cause

1. **Migration Incomplete**: Migration يضيف الأعمدة فقط دون ملء البيانات
2. **Assumed Empty Database**: افتراض أن قاعدة البيانات فارغة عند التشغيل
3. **Manual Backfill Expected**: توقع أن يتم الملء يدويًا لاحقًا

#### Fix Implementation

**File**: `services/backend/migrations/077_add_custom_ids.sql`

**Add Backfill Logic**:

```sql
-- After adding columns, backfill existing records

-- 9. Backfill custom_id for existing events
-- Generate custom_id based on created_at timestamp and id
DO $
DECLARE
    batch_size INT := 1000;
    processed INT := 0;
    total_count INT;
BEGIN
    -- Get total count
    SELECT COUNT(*) INTO total_count 
    FROM global_incoming_events 
    WHERE custom_id IS NULL;
    
    RAISE NOTICE 'Backfilling custom_id for % events', total_count;
    
    -- Process in batches
    WHILE EXISTS (SELECT 1 FROM global_incoming_events WHERE custom_id IS NULL LIMIT 1) LOOP
        UPDATE global_incoming_events
        SET custom_id = 'EV' || 
            TO_CHAR(created_at, 'YYMMDD') ||
            LPAD(EXTRACT(HOUR FROM created_at)::TEXT, 2, '0') ||
            LPAD(SUBSTR(MD5(tenant_id::TEXT), 1, 5), 5, '0') ||
            LPAD(id::TEXT, 6, '0')
        WHERE id IN (
            SELECT id 
            FROM global_incoming_events 
            WHERE custom_id IS NULL 
            LIMIT batch_size
        );
        
        processed := processed + batch_size;
        
        IF processed % 10000 = 0 THEN
            RAISE NOTICE 'Processed % / % events', processed, total_count;
        END IF;
    END LOOP;
    
    RAISE NOTICE 'Backfill complete: % events updated', total_count;
END $;

-- 10. Backfill custom_id for existing messages
DO $
DECLARE
    batch_size INT := 1000;
    processed INT := 0;
    total_count INT;
BEGIN
    SELECT COUNT(*) INTO total_count 
    FROM messages 
    WHERE custom_id IS NULL;
    
    RAISE NOTICE 'Backfilling custom_id for % messages', total_count;
    
    WHILE EXISTS (SELECT 1 FROM messages WHERE custom_id IS NULL LIMIT 1) LOOP
        UPDATE messages m
        SET custom_id = 'MS' || 
            TO_CHAR(m.created_at, 'YYMMDD') ||
            LPAD(EXTRACT(HOUR FROM m.created_at)::TEXT, 2, '0') ||
            LPAD(SUBSTR(MD5(m.tenant_id::TEXT), 1, 5), 5, '0') ||
            LPAD(COALESCE(m.event_id::TEXT, '0'), 4, '0') ||
            LPAD(m.id::TEXT, 3, '0'),
        event_custom_id = (
            SELECT ge.custom_id 
            FROM global_incoming_events ge 
            WHERE ge.id = m.event_id
        )
        WHERE m.id IN (
            SELECT id 
            FROM messages 
            WHERE custom_id IS NULL 
            LIMIT batch_size
        );
        
        processed := processed + batch_size;
        
        IF processed % 10000 = 0 THEN
            RAISE NOTICE 'Processed % / % messages', processed, total_count;
        END IF;
    END LOOP;
    
    RAISE NOTICE 'Backfill complete: % messages updated', total_count;
END $;

-- 11. Make custom_id NOT NULL after backfill (optional, for data integrity)
-- ALTER TABLE global_incoming_events ALTER COLUMN custom_id SET NOT NULL;
-- ALTER TABLE messages ALTER COLUMN custom_id SET NOT NULL;
```

---

### Bug 12: SCAN Without COUNT Parameter

#### Bug Condition

المشكلة تحدث عندما يتم استدعاء `InvalidateAll()` في `pricing_cache.go`.

**Formal Specification:**
```
FUNCTION isBugCondition_12(input)
  INPUT: cache invalidation operation
  OUTPUT: boolean
  
  RETURN InvalidateAll() is called
         AND SCAN is used without COUNT parameter
         AND may cause performance issues with large key sets
END FUNCTION
```

**Examples:**
- 100K cached pricing entries → SCAN without COUNT → slow iteration
- Redis blocked during scan → other operations delayed
- Large result sets → memory pressure

#### Expected Behavior

**Correct Behavior:**
عندما يتم استدعاء `InvalidateAll()`، يجب أن يقوم النظام باستخدام SCAN مع معامل COUNT (مثل 1000) للتصفح الفعال.

**Preservation Requirements:**
- تنظيف الذاكرة المؤقتة يجب أن يحذف جميع المفاتيح
- عمليات Redis الأخرى يجب ألا تتأثر
- أداء التنظيف يجب أن يكون مقبولاً

#### Hypothesized Root Cause

1. **Default COUNT**: الكود الحالي يستخدم COUNT=1000 بالفعل في السطر `Scan(ctx, 0, pattern, 1000)`
2. **Documentation Issue**: قد يكون هذا خطأ في التوثيق
3. **Verification Needed**: يحتاج التحقق من أن COUNT يُستخدم بشكل صحيح

#### Fix Implementation

**File**: `services/backend/internal/redis/pricing_cache.go`

**Function**: `InvalidateAll()`

**Current Code Review**:
```go
// Current implementation already uses COUNT=1000
iter := pc.client.Client.Scan(ctx, 0, pattern, 1000).Iterator()
```

**Enhancement - Add Batch Processing**:
```go
func (pc *PricingCache) InvalidateAll(ctx context.Context) error {
    pattern := fmt.Sprintf("%s:cache:pricing:*", AppNamespace)
    
    var cursor uint64
    var totalDeleted int
    
    for {
        // Scan with COUNT parameter for efficient pagination
        keys, nextCursor, err := pc.client.Client.Scan(ctx, cursor, pattern, 1000).Result()
        if err != nil {
            return fmt.Errorf("scan keys: %w", err)
        }
        
        // Delete keys in pipeline if any found
        if len(keys) > 0 {
            pipe := pc.client.Client.Pipeline()
            for _, key := range keys {
                pipe.Del(ctx, key)
            }
            
            _, err = pipe.Exec(ctx)
            if err != nil {
                return fmt.Errorf("delete keys: %w", err)
            }
            
            totalDeleted += len(keys)
        }
        
        // Move to next cursor
        cursor = nextCursor
        
        // Break if scan complete
        if cursor == 0 {
            break
        }
    }
    
    return nil
}
```


---

### Bug 13: Inconsistent Cache TTL Values

#### Bug Condition

المشكلة تحدث عندما يتم تخزين بيانات الرصيد والاستخدام مؤقتًا.

**Formal Specification:**
```
FUNCTION isBugCondition_13(input)
  INPUT: cache operations for balance and usage
  OUTPUT: boolean
  
  RETURN balance cache uses TTL = 1 minute
         AND usage cache uses TTL = 30 seconds
         AND inconsistent TTLs cause cache coherence issues
END FUNCTION
```

**Examples:**
- Balance cached at T=0 (TTL 60s) → Usage cached at T=0 (TTL 30s)
- At T=35s: Balance from cache (old), Usage from DB (new) → inconsistent view
- User sees old balance with new usage → confusion

#### Expected Behavior

**Correct Behavior:**
عندما يتم تخزين بيانات الرصيد والاستخدام مؤقتًا، يجب أن يقوم النظام باستخدام قيم TTL متسقة (مثل كلاهما دقيقة واحدة) للحفاظ على تماسك الذاكرة المؤقتة.

**Preservation Requirements:**
- أداء الذاكرة المؤقتة يجب أن يبقى سريعًا
- معدل إصابة الذاكرة المؤقتة يجب أن يبقى مرتفعًا
- استعلامات قاعدة البيانات يجب أن تبقى محدودة

#### Hypothesized Root Cause

1. **Different Requirements**: افتراض أن الرصيد والاستخدام لهما متطلبات نضارة مختلفة
2. **Performance Tuning**: محاولة تحسين الأداء بـ TTL مختلفة
3. **Incremental Development**: تطوير تدريجي بدون توحيد

#### Fix Implementation

**Files**: Multiple cache implementations

**Changes Required:**

1. **Define Standard TTL Constants**:
```go
// In redis/constants.go or redis/client.go
const (
    // Cache TTL values - keep consistent for related data
    CacheTTLBalance      = 1 * time.Minute
    CacheTTLUsage        = 1 * time.Minute  // Changed from 30s
    CacheTTLPricing      = 1 * time.Hour
    CacheTTLChannel      = 1 * time.Minute  // Changed from 5 minutes
    CacheTTLTenant       = 5 * time.Minute
)
```

2. **Update Balance Cache** (if exists):
```go
func (bc *BalanceCache) Set(ctx context.Context, tenantID uuid.UUID, balance decimal.Decimal) error {
    key := fmt.Sprintf("%s:cache:balance:%s", AppNamespace, tenantID)
    return bc.client.Client.Set(ctx, key, balance.String(), CacheTTLBalance).Err()
}
```

3. **Update Usage Cache** (if exists):
```go
func (uc *UsageCache) Set(ctx context.Context, tenantID uuid.UUID, usage *UsageInfo) error {
    key := fmt.Sprintf("%s:cache:usage:%s", AppNamespace, tenantID)
    data, _ := json.Marshal(usage)
    return uc.client.Client.Set(ctx, key, data, CacheTTLUsage).Err()  // Now 1 minute
}
```

4. **Update Channel Cache TTL**:
```go
// In redis/channel_cache.go
func NewChannelCache(client *Client) *ChannelCache {
    return &ChannelCache{
        client: client,
        ttl:    CacheTTLChannel,  // Now 1 minute instead of 5
    }
}
```

**Rationale**: 
- Balance and Usage are tightly coupled → same TTL ensures consistency
- Shorter Channel TTL (1 min vs 5 min) reduces stale channel data
- All financial data uses 1-minute TTL for consistency

---

### Bug 14: Incomplete Analytics Message Count

#### Bug Condition

المشكلة تحدث عندما يتم حساب التحليلات في `redis/analytics.go`.

**Formal Specification:**
```
FUNCTION isBugCondition_14(input)
  INPUT: analytics calculation request
  OUTPUT: boolean
  
  RETURN analytics are calculated
         AND system does not count actual messages
         AND provides incomplete metrics
END FUNCTION
```

**Examples:**
- 10,000 messages in system → analytics shows 0 messages
- Dashboard displays incorrect load metrics
- Capacity planning based on wrong data

#### Expected Behavior

**Correct Behavior:**
عندما يتم حساب التحليلات، يجب أن يقوم النظام بالاستعلام عن عدد الرسائل الفعلية من Redis أو قاعدة البيانات وتجميعها بشكل صحيح.

**Preservation Requirements:**
- حساب عدد الأحداث يجب أن يستمر بالعمل
- أداء التحليلات يجب أن يبقى سريعًا
- دقة البيانات يجب أن تتحسن

#### Hypothesized Root Cause

**Already Fixed in Bug 4** - هذا الخطأ تم معالجته في Bug 4 (GetCurrentLoad implementation).

#### Fix Implementation

**Status**: Fixed in Bug 4 implementation above.

**Additional Enhancement** - Add Database Fallback:
```go
func (la *LoadAnalytics) GetMessageCountFromDB(ctx context.Context, since time.Time) (int64, error) {
    query := `
        SELECT COUNT(*) 
        FROM messages 
        WHERE created_at >= $1
    `
    
    var count int64
    err := db.PgxPool.QueryRow(ctx, query, since).Scan(&count)
    return count, err
}

// Use in GetCurrentLoad as fallback
func (la *LoadAnalytics) GetCurrentLoad(ctx context.Context) (*LoadMetrics, error) {
    // ... existing Redis-based counting ...
    
    // If Redis data is incomplete, fallback to DB
    if totalMessages == 0 {
        dbCount, err := la.GetMessageCountFromDB(ctx, time.Now().Add(-1*time.Hour))
        if err == nil {
            totalMessages = dbCount
        }
    }
    
    // ... rest of function ...
}
```

---

### Bug 15: No Auto-Logout on 401

#### Bug Condition

المشكلة تحدث عندما يتم استلام استجابة 401 Unauthorized في `axios.js`.

**Formal Specification:**
```
FUNCTION isBugCondition_15(input)
  INPUT: API response with status 401
  OUTPUT: boolean
  
  RETURN API returns 401 Unauthorized
         AND token is expired or invalid
         AND system does not automatically log out user
         AND expired session remains active in UI
END FUNCTION
```

**Examples:**
- Token expires after 24h → API returns 401 → user sees errors but stays "logged in"
- User clicks around → multiple 401 errors → confusing UX
- Token stolen/revoked → 401 → attacker still sees UI


#### Expected Behavior

**Correct Behavior:**
عندما يتم استلام استجابة 401 Unauthorized، يجب أن يقوم النظام بمسح token من localStorage وإعادة التوجيه إلى صفحة تسجيل الدخول تلقائيًا.

**Preservation Requirements:**
- معالجة الاستجابات الناجحة يجب أن تستمر بالعمل
- إضافة Authorization header يجب أن يستمر
- أداء الطلبات يجب أن يبقى كما هو

#### Hypothesized Root Cause

1. **Commented Code**: الكود موجود لكن معلق (lines 27-28 في axios.js)
2. **Testing Phase**: تم تعليقه أثناء الاختبار ونُسي
3. **Intentional**: قد يكون مقصودًا لسبب ما

#### Fix Implementation

**File**: `apps/client-web/src/api/axios.js`

**Changes Required:**

```javascript
// Response interceptor to handle unauthorized errors
apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response && error.response.status === 401) {
      // Clear authentication state
      localStorage.removeItem('token');
      localStorage.removeItem('user');
      localStorage.removeItem('tenant');
      
      // Log for debugging
      console.warn('Session expired - redirecting to login');
      
      // Redirect to login page
      // Check if not already on login page to avoid redirect loop
      if (!window.location.pathname.includes('/login')) {
        window.location.href = '/login';
      }
    }
    return Promise.reject(error);
  }
);
```

**Also Update admin-web**:
```javascript
// In apps/admin-web/src/api/axios.js (if exists)
// Apply same fix
```

**Add User Notification** (optional enhancement):
```javascript
import { toast } from 'react-toastify'; // or your notification library

apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response && error.response.status === 401) {
      // Show user-friendly message
      toast.error('جلستك انتهت. يرجى تسجيل الدخول مرة أخرى.');
      
      // Clear and redirect
      localStorage.removeItem('token');
      localStorage.removeItem('user');
      localStorage.removeItem('tenant');
      
      if (!window.location.pathname.includes('/login')) {
        window.location.href = '/login';
      }
    }
    return Promise.reject(error);
  }
);
```

---

## 🔵 Security Bugs (16-20)

### Bug 16: Exposed Database Credentials

#### Bug Condition

المشكلة تحدث عندما يتم تشغيل سكريبتات النشر.

**Formal Specification:**
```
FUNCTION isBugCondition_16(input)
  INPUT: deployment script execution
  OUTPUT: boolean
  
  RETURN deployment scripts contain hardcoded credentials
         AND credentials are exposed in plain text
         AND security vulnerability exists
END FUNCTION
```

**Examples:**
- Script contains `DB_PASSWORD=secret123` → exposed in git history
- Credentials in docker-compose.yml → visible to all developers
- Production credentials in scripts → major security risk

#### Expected Behavior

**Correct Behavior:**
عندما تحتاج سكريبتات النشر إلى بيانات الاعتماد، يجب أن يقوم النظام بقراءة بيانات الاعتماد من متغيرات البيئة أو خزائن آمنة، وليس قيم مشفرة.

**Preservation Requirements:**
- سكريبتات النشر يجب أن تستمر بالعمل
- عملية النشر يجب أن تبقى آلية
- التوافق مع CI/CD يجب أن يُحفظ

#### Hypothesized Root Cause

1. **Quick Setup**: بيانات اعتماد مؤقتة للإعداد السريع
2. **Development Environment**: مخصصة للتطوير فقط
3. **Missing Security Review**: لم تتم مراجعة أمنية

#### Fix Implementation

**Files**: Deployment scripts, docker-compose files

**Changes Required:**

1. **Remove Hardcoded Credentials**:
```bash
# Before (in deploy.sh or similar):
DB_PASSWORD=secret123
REDIS_PASSWORD=redis456

# After:
if [ -z "$DB_PASSWORD" ]; then
    echo "Error: DB_PASSWORD environment variable not set"
    exit 1
fi

if [ -z "$REDIS_PASSWORD" ]; then
    echo "Error: REDIS_PASSWORD environment variable not set"
    exit 1
fi
```

2. **Update docker-compose.yml**:
```yaml
# Before:
services:
  postgres:
    environment:
      POSTGRES_PASSWORD: secret123

# After:
services:
  postgres:
    environment:
      POSTGRES_PASSWORD: ${DB_PASSWORD:?DB_PASSWORD not set}
```

3. **Create .env.example**:
```bash
# .env.example - Template for environment variables
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_secure_password_here
DB_NAME=alhudhud
DB_SSLMODE=disable

REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=your_redis_password_here

JWT_SECRET=your_jwt_secret_here
```

4. **Add .env to .gitignore**:
```
# .gitignore
.env
.env.local
.env.production
*.pem
*.key
```

5. **Document in README**:
```markdown
## Environment Setup

1. Copy `.env.example` to `.env`
2. Fill in your credentials
3. Never commit `.env` to git

For production, use:
- AWS Secrets Manager
- HashiCorp Vault
- Kubernetes Secrets
```


---

### Bug 17: Missing CSRF Protection

#### Bug Condition

المشكلة تحدث عندما يتم إجراء طلبات API مع `withCredentials: true` في `axios.js`.

**Formal Specification:**
```
FUNCTION isBugCondition_17(input)
  INPUT: API request with credentials
  OUTPUT: boolean
  
  RETURN request uses withCredentials: true
         AND no CSRF token is included
         AND system is vulnerable to CSRF attacks
END FUNCTION
```

**Examples:**
- Attacker creates malicious site → user visits while logged in → attacker makes requests using user's session
- Form submission without CSRF token → accepted by server
- State-changing operations (POST/PUT/DELETE) without CSRF protection

#### Expected Behavior

**Correct Behavior:**
عندما يتم إجراء طلبات API مع بيانات الاعتماد، يجب أن يقوم النظام بتنفيذ التحقق من رمز CSRF باستخدام نمط double-submit cookie أو synchronizer token.

**Preservation Requirements:**
- الطلبات الشرعية يجب أن تستمر بالعمل
- أداء الطلبات يجب أن يبقى سريعًا
- تجربة المستخدم يجب ألا تتأثر

#### Hypothesized Root Cause

1. **JWT-Based Auth**: افتراض أن JWT يوفر حماية كافية
2. **Missing Implementation**: CSRF protection لم يتم تنفيذه بعد
3. **Development Phase**: مخطط للتنفيذ لاحقًا

#### Fix Implementation

**Strategy**: استخدام Double-Submit Cookie Pattern

**Backend Changes**:

1. **Add CSRF Middleware** (in Go):
```go
// internal/middleware/csrf.go
package middleware

import (
    "crypto/rand"
    "encoding/base64"
    "net/http"
    "time"
    
    "github.com/gin-gonic/gin"
)

const (
    CSRFTokenLength = 32
    CSRFCookieName  = "csrf_token"
    CSRFHeaderName  = "X-CSRF-Token"
)

// GenerateCSRFToken generates a random CSRF token
func GenerateCSRFToken() (string, error) {
    bytes := make([]byte, CSRFTokenLength)
    if _, err := rand.Read(bytes); err != nil {
        return "", err
    }
    return base64.URLEncoding.EncodeToString(bytes), nil
}

// CSRFProtection middleware validates CSRF tokens
func CSRFProtection() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Skip for GET, HEAD, OPTIONS
        if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
            c.Next()
            return
        }
        
        // Get token from cookie
        cookieToken, err := c.Cookie(CSRFCookieName)
        if err != nil {
            c.JSON(http.StatusForbidden, gin.H{"error": "CSRF token missing"})
            c.Abort()
            return
        }
        
        // Get token from header
        headerToken := c.GetHeader(CSRFHeaderName)
        if headerToken == "" {
            c.JSON(http.StatusForbidden, gin.H{"error": "CSRF token missing in header"})
            c.Abort()
            return
        }
        
        // Validate tokens match
        if cookieToken != headerToken {
            c.JSON(http.StatusForbidden, gin.H{"error": "CSRF token mismatch"})
            c.Abort()
            return
        }
        
        c.Next()
    }
}

// SetCSRFToken sets CSRF token in cookie
func SetCSRFToken(c *gin.Context) {
    token, err := GenerateCSRFToken()
    if err != nil {
        return
    }
    
    c.SetCookie(
        CSRFCookieName,
        token,
        3600*24, // 24 hours
        "/",
        "",
        true,  // Secure
        false, // HttpOnly = false (needs to be readable by JS)
    )
}
```

2. **Apply Middleware**:
```go
// In main.go or router setup
router.Use(middleware.CSRFProtection())

// Add endpoint to get CSRF token
router.GET("/api/csrf-token", func(c *gin.Context) {
    middleware.SetCSRFToken(c)
    token, _ := c.Cookie(middleware.CSRFCookieName)
    c.JSON(200, gin.H{"csrf_token": token})
})
```

**Frontend Changes**:

1. **Update axios.js**:
```javascript
import axios from 'axios';
import { API_BASE_URL } from '../config';

const apiClient = axios.create({
  baseURL: API_BASE_URL,
  withCredentials: true,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Fetch CSRF token on app initialization
let csrfToken = null;

export const initializeCSRF = async () => {
  try {
    const response = await axios.get(`${API_BASE_URL}/api/csrf-token`, {
      withCredentials: true
    });
    csrfToken = response.data.csrf_token;
  } catch (error) {
    console.error('Failed to fetch CSRF token:', error);
  }
};

// Request interceptor to add Authorization and CSRF headers
apiClient.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token');
    if (token) {
      config.headers['Authorization'] = `Bearer ${token}`;
    }
    
    // Add CSRF token for state-changing requests
    if (csrfToken && ['POST', 'PUT', 'DELETE', 'PATCH'].includes(config.method.toUpperCase())) {
      config.headers['X-CSRF-Token'] = csrfToken;
    }
    
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// ... rest of interceptors ...

export default apiClient;
```

2. **Initialize in App.jsx**:
```javascript
import { useEffect } from 'react';
import { initializeCSRF } from './api/axios';

function App() {
  useEffect(() => {
    initializeCSRF();
  }, []);
  
  // ... rest of app
}
```

---

### Bug 18: Tokens in localStorage (XSS Vulnerability)

#### Bug Condition

المشكلة تحدث عندما يتم تخزين رموز المصادقة.

**Formal Specification:**
```
FUNCTION isBugCondition_18(input)
  INPUT: authentication token storage
  OUTPUT: boolean
  
  RETURN tokens are stored in localStorage
         AND localStorage is accessible to JavaScript
         AND system is vulnerable to XSS attacks
END FUNCTION
```

**Examples:**
- XSS vulnerability in app → attacker steals token from localStorage
- Third-party script compromised → token leaked
- Browser extension malicious → token stolen


#### Expected Behavior

**Correct Behavior:**
عندما يتم تخزين رموز المصادقة، يجب أن يقوم النظام باستخدام httpOnly secure cookies بدلاً من localStorage لمنع هجمات XSS.

**Preservation Requirements:**
- المصادقة يجب أن تستمر بالعمل
- تجربة المستخدم يجب ألا تتأثر
- أداء الطلبات يجب أن يبقى كما هو

#### Hypothesized Root Cause

1. **Common Pattern**: localStorage شائع في تطبيقات SPA
2. **Simplicity**: أسهل في التنفيذ من cookies
3. **CORS Considerations**: تجنب تعقيدات CORS مع cookies

#### Fix Implementation

**Strategy**: Migrate to httpOnly Cookies

**Backend Changes**:

1. **Update Login Endpoint**:
```go
// In auth handler
func (h *AuthHandler) Login(c *gin.Context) {
    // ... authentication logic ...
    
    // Generate JWT token
    token, err := generateJWT(user)
    if err != nil {
        c.JSON(500, gin.H{"error": "Failed to generate token"})
        return
    }
    
    // Set token in httpOnly cookie
    c.SetCookie(
        "auth_token",
        token,
        3600*24*7, // 7 days
        "/",
        "",
        true,  // Secure (HTTPS only)
        true,  // HttpOnly (not accessible to JavaScript)
    )
    
    // Return user info (without token)
    c.JSON(200, gin.H{
        "user": user,
        "message": "Login successful",
    })
}
```

2. **Update Auth Middleware**:
```go
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Get token from cookie instead of Authorization header
        token, err := c.Cookie("auth_token")
        if err != nil {
            c.JSON(401, gin.H{"error": "Unauthorized"})
            c.Abort()
            return
        }
        
        // Validate token
        claims, err := validateJWT(token)
        if err != nil {
            c.JSON(401, gin.H{"error": "Invalid token"})
            c.Abort()
            return
        }
        
        // Set user context
        c.Set("user_id", claims.UserID)
        c.Set("tenant_id", claims.TenantID)
        
        c.Next()
    }
}
```

3. **Add Logout Endpoint**:
```go
func (h *AuthHandler) Logout(c *gin.Context) {
    // Clear cookie
    c.SetCookie(
        "auth_token",
        "",
        -1, // Expire immediately
        "/",
        "",
        true,
        true,
    )
    
    c.JSON(200, gin.H{"message": "Logged out successfully"})
}
```

**Frontend Changes**:

1. **Update axios.js**:
```javascript
// Remove Authorization header logic
apiClient.interceptors.request.use(
  (config) => {
    // Token is now sent automatically via cookie
    // No need to add Authorization header
    
    // Only add CSRF token
    if (csrfToken && ['POST', 'PUT', 'DELETE', 'PATCH'].includes(config.method.toUpperCase())) {
      config.headers['X-CSRF-Token'] = csrfToken;
    }
    
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);
```

2. **Update Login Component**:
```javascript
// In Login.jsx
const handleLogin = async (credentials) => {
  try {
    const response = await apiClient.post('/api/auth/login', credentials);
    
    // Don't store token in localStorage anymore
    // Store only user info
    localStorage.setItem('user', JSON.stringify(response.data.user));
    
    // Redirect to dashboard
    navigate('/dashboard');
  } catch (error) {
    setError('Login failed');
  }
};
```

3. **Update Logout**:
```javascript
const handleLogout = async () => {
  try {
    await apiClient.post('/api/auth/logout');
    localStorage.removeItem('user');
    navigate('/login');
  } catch (error) {
    console.error('Logout failed:', error);
  }
};
```

4. **Update CORS Configuration**:
```go
// In main.go
router.Use(cors.New(cors.Config{
    AllowOrigins:     []string{"http://localhost:3000", "https://yourdomain.com"},
    AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
    AllowHeaders:     []string{"Content-Type", "X-CSRF-Token"},
    AllowCredentials: true, // Important for cookies
    MaxAge:           12 * time.Hour,
}))
```

**Migration Plan**:
1. Deploy backend with cookie support (keep header support for backward compatibility)
2. Update frontend to use cookies
3. After all clients updated, remove header-based auth

---

### Bug 19: SQL Injection Risk

#### Bug Condition

المشكلة تحدث عندما يتم بناء استعلامات SQL ديناميكية.

**Formal Specification:**
```
FUNCTION isBugCondition_19(input)
  INPUT: dynamic SQL query construction
  OUTPUT: boolean
  
  RETURN SQL query is constructed with string concatenation
         AND user input is not parameterized
         AND system is vulnerable to SQL injection
END FUNCTION
```

**Examples:**
- `query := "SELECT * FROM users WHERE name = '" + userName + "'"` → SQL injection
- Dynamic ORDER BY without validation → injection risk
- Dynamic table names without whitelist → injection risk

#### Expected Behavior

**Correct Behavior:**
عندما يتم بناء استعلامات SQL، يجب أن يقوم النظام باستخدام الاستعلامات المعلمة حصريًا، والتحقق من صحة جميع بناء الاستعلامات الديناميكية.

**Preservation Requirements:**
- الاستعلامات الموجودة يجب أن تستمر بالعمل
- أداء الاستعلامات يجب أن يبقى كما هو
- وظائف البحث والتصفية يجب أن تعمل

#### Hypothesized Root Cause

1. **Code Review Needed**: يحتاج مراجعة شاملة للكود
2. **Dynamic Queries**: بعض الاستعلامات تحتاج بناء ديناميكي
3. **Legacy Code**: كود قديم قد يحتوي على ممارسات غير آمنة


#### Fix Implementation

**Strategy**: Audit and Fix All SQL Queries

**1. Search for Unsafe Patterns**:
```bash
# Search for potential SQL injection vulnerabilities
grep -r "fmt.Sprintf.*SELECT" services/backend/
grep -r "fmt.Sprintf.*INSERT" services/backend/
grep -r "fmt.Sprintf.*UPDATE" services/backend/
grep -r "fmt.Sprintf.*DELETE" services/backend/
grep -r '+ "SELECT' services/backend/
```

**2. Safe Query Patterns**:

**Bad** (Vulnerable):
```go
// NEVER DO THIS
query := fmt.Sprintf("SELECT * FROM users WHERE email = '%s'", email)
db.Query(query)
```

**Good** (Safe):
```go
// ALWAYS DO THIS
query := "SELECT * FROM users WHERE email = $1"
db.Query(query, email)
```

**3. Dynamic ORDER BY** (Safe Implementation):
```go
func buildOrderByClause(sortField string, sortOrder string) (string, error) {
    // Whitelist allowed fields
    allowedFields := map[string]bool{
        "created_at": true,
        "updated_at": true,
        "status":     true,
        "cost":       true,
    }
    
    if !allowedFields[sortField] {
        return "", fmt.Errorf("invalid sort field: %s", sortField)
    }
    
    // Validate sort order
    if sortOrder != "ASC" && sortOrder != "DESC" {
        return "", fmt.Errorf("invalid sort order: %s", sortOrder)
    }
    
    // Safe to use in query now
    return fmt.Sprintf("ORDER BY %s %s", sortField, sortOrder), nil
}

// Usage:
orderBy, err := buildOrderByClause(req.SortField, req.SortOrder)
if err != nil {
    return err
}
query := "SELECT * FROM messages WHERE tenant_id = $1 " + orderBy
```

**4. Dynamic Filters** (Safe Implementation):
```go
func buildFilterQuery(filters map[string]interface{}) (string, []interface{}, error) {
    allowedFilters := map[string]string{
        "status":       "status = $",
        "phone_number": "phone_number = $",
        "created_after": "created_at >= $",
    }
    
    var conditions []string
    var args []interface{}
    paramIndex := 1
    
    for key, value := range filters {
        template, ok := allowedFilters[key]
        if !ok {
            return "", nil, fmt.Errorf("invalid filter: %s", key)
        }
        
        condition := strings.Replace(template, "$", fmt.Sprintf("$%d", paramIndex), 1)
        conditions = append(conditions, condition)
        args = append(args, value)
        paramIndex++
    }
    
    whereClause := ""
    if len(conditions) > 0 {
        whereClause = "WHERE " + strings.Join(conditions, " AND ")
    }
    
    return whereClause, args, nil
}

// Usage:
whereClause, args, err := buildFilterQuery(filters)
if err != nil {
    return err
}
query := "SELECT * FROM messages " + whereClause
db.Query(query, args...)
```

**5. Review Existing Queries**:

Check `fast_processor.go`:
```go
// Line 195 - SAFE (uses parameterized query)
query := `
    UPDATE messages 
    SET status = $1, updated_at = NOW()
    WHERE id = ANY($2)
`
_, err := tx.Exec(ctx, query, status, messageIDs)
```

Check all handlers and services for similar patterns.

**6. Add SQL Injection Tests**:
```go
func TestSQLInjectionPrevention(t *testing.T) {
    // Test malicious input
    maliciousEmail := "admin@example.com' OR '1'='1"
    
    // Should not return all users
    users, err := GetUserByEmail(maliciousEmail)
    assert.NoError(t, err)
    assert.Empty(t, users) // Should find no users
    
    // Test malicious sort field
    _, err = GetMessages(filters, "id; DROP TABLE messages--", "ASC")
    assert.Error(t, err) // Should reject invalid field
}
```

---

### Bug 20: Missing Environment Variable Validation

#### Bug Condition

المشكلة تحدث عندما يبدأ التطبيق.

**Formal Specification:**
```
FUNCTION isBugCondition_20(input)
  INPUT: application startup
  OUTPUT: boolean
  
  RETURN application starts
         AND required environment variables are missing
         AND application runs with missing configuration
         AND causes runtime errors
END FUNCTION
```

**Examples:**
- JWT_SECRET not set → token generation fails at runtime
- DB_HOST not set → connection fails after startup
- REDIS_HOST not set → cache operations fail silently

#### Expected Behavior

**Correct Behavior:**
عندما يبدأ التطبيق، يجب أن يقوم النظام بالتحقق من جميع متغيرات البيئة المطلوبة (DB_HOST, DB_PORT, JWT_SECRET, إلخ) والفشل بسرعة مع رسائل خطأ واضحة إذا كانت مفقودة.

**Preservation Requirements:**
- بدء التطبيق الناجح مع تكوين صحيح يجب أن يستمر
- أداء بدء التشغيل يجب أن يبقى سريعًا
- رسائل الخطأ يجب أن تكون واضحة

#### Hypothesized Root Cause

1. **Missing Validation**: لا يوجد فحص للمتغيرات المطلوبة
2. **Fail Late**: الأخطاء تظهر عند الاستخدام وليس عند البدء
3. **Development Focus**: التركيز على التطوير بدلاً من الإنتاج

#### Fix Implementation

**File**: `services/backend/cmd/main.go` or `internal/config/config.go`

**Create Config Validation**:

```go
// internal/config/config.go
package config

import (
    "fmt"
    "os"
    "strconv"
    
    "go.uber.org/zap"
)

type Config struct {
    // Database
    DBHost     string
    DBPort     int
    DBUser     string
    DBPassword string
    DBName     string
    DBSSLMode  string
    
    // Redis
    RedisHost     string
    RedisPort     int
    RedisPassword string
    RedisDB       int
    
    // JWT
    JWTSecret     string
    JWTExpiration int
    
    // Server
    ServerPort int
    ServerHost string
    
    // CORS
    AllowedOrigins []string
}

// LoadConfig loads and validates configuration from environment
func LoadConfig() (*Config, error) {
    config := &Config{}
    
    // Required variables
    required := map[string]*string{
        "DB_HOST":     &config.DBHost,
        "DB_USER":     &config.DBUser,
        "DB_PASSWORD": &config.DBPassword,
        "DB_NAME":     &config.DBName,
        "REDIS_HOST":  &config.RedisHost,
        "JWT_SECRET":  &config.JWTSecret,
    }
    
    var missing []string
    for key, target := range required {
        value := os.Getenv(key)
        if value == "" {
            missing = append(missing, key)
        } else {
            *target = value
        }
    }
    
    if len(missing) > 0 {
        return nil, fmt.Errorf("missing required environment variables: %v", missing)
    }
    
    // Optional with defaults
    config.DBPort = getEnvAsInt("DB_PORT", 5432)
    config.DBSSLMode = getEnvOrDefault("DB_SSLMODE", "disable")
    config.RedisPort = getEnvAsInt("REDIS_PORT", 6379)
    config.RedisPassword = os.Getenv("REDIS_PASSWORD") // Optional
    config.RedisDB = getEnvAsInt("REDIS_DB", 0)
    config.JWTExpiration = getEnvAsInt("JWT_EXPIRATION", 3600*24*7) // 7 days
    config.ServerPort = getEnvAsInt("SERVER_PORT", 8080)
    config.ServerHost = getEnvOrDefault("SERVER_HOST", "0.0.0.0")
    
    // Validate JWT secret length
    if len(config.JWTSecret) < 32 {
        return nil, fmt.Errorf("JWT_SECRET must be at least 32 characters")
    }
    
    return config, nil
}

func getEnvOrDefault(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
    if value := os.Getenv(key); value != "" {
        if intValue, err := strconv.Atoi(value); err == nil {
            return intValue
        }
    }
    return defaultValue
}

// Validate performs additional validation checks
func (c *Config) Validate() error {
    if c.DBPort < 1 || c.DBPort > 65535 {
        return fmt.Errorf("invalid DB_PORT: %d", c.DBPort)
    }
    
    if c.RedisPort < 1 || c.RedisPort > 65535 {
        return fmt.Errorf("invalid REDIS_PORT: %d", c.RedisPort)
    }
    
    if c.ServerPort < 1 || c.ServerPort > 65535 {
        return fmt.Errorf("invalid SERVER_PORT: %d", c.ServerPort)
    }
    
    return nil
}
```

**Update main.go**:
```go
func main() {
    // Initialize logger first
    logger := logger.GetLogger()
    
    // Load and validate configuration
    config, err := config.LoadConfig()
    if err != nil {
        logger.Fatal("Configuration error", zap.Error(err))
    }
    
    if err := config.Validate(); err != nil {
        logger.Fatal("Configuration validation failed", zap.Error(err))
    }
    
    logger.Info("Configuration loaded successfully",
        zap.String("db_host", config.DBHost),
        zap.Int("db_port", config.DBPort),
        zap.String("redis_host", config.RedisHost),
        zap.Int("server_port", config.ServerPort),
    )
    
    // Continue with initialization...
}
```


---

## 📊 Performance Bugs (21-25)

### Bug 21: Suboptimal Batch Processing Configuration

#### Bug Condition

المشكلة تحدث عندما يتم تكوين معالجة الدفعات.

**Formal Specification:**
```
FUNCTION isBugCondition_21(input)
  INPUT: batch processing configuration
  OUTPUT: boolean
  
  RETURN batch size = 100 messages
         AND worker count = 10
         AND total throughput = 1000 messages/cycle
         AND configuration may not be optimized for high load
END FUNCTION
```

**Examples:**
- 100 msg × 10 workers = 1000 msg/cycle → may be too low for peak load
- Workers may be underutilized with small batches
- Could process 50 msg × 20 workers = 1000 msg/cycle with better parallelism

#### Expected Behavior

**Correct Behavior:**
عندما يتم تعيين تكوين معالجة الدفعات، يجب أن يقوم النظام باستخدام حجم دفعة محسّن بناءً على اختبار الحمل (مثل 50 رسالة × 20 عامل لتوازي أفضل).

**Preservation Requirements:**
- معالجة الرسائل يجب أن تستمر بالعمل
- ترتيب الرسائل يجب أن يُحفظ
- استهلاك الموارد يجب أن يكون معقولاً

#### Hypothesized Root Cause

1. **Default Values**: قيم افتراضية لم يتم ضبطها
2. **No Load Testing**: لم يتم اختبار الحمل لتحديد القيم المثلى
3. **Conservative Approach**: نهج محافظ لتجنب الحمل الزائد

#### Fix Implementation

**File**: `services/backend/internal/worker/fast_processor.go`

**Changes Required:**

1. **Make Configuration Tunable**:
```go
type FastMessageProcessor struct {
    messageQueue   *redis.MessageQueue
    channelCache   *redis.ChannelCache
    balanceManager balance.Manager
    wsHub          *ws.Hub
    logger         *zap.Logger
    batchSize      int
    workerCount    int
    stopChan       chan struct{}
}

// NewFastMessageProcessor with configurable parameters
func NewFastMessageProcessor(
    messageQueue *redis.MessageQueue,
    channelCache *redis.ChannelCache,
    balanceManager balance.Manager,
    wsHub *ws.Hub,
    logger *zap.Logger,
    batchSize int,
    workerCount int,
) *FastMessageProcessor {
    // Set defaults if not provided
    if batchSize <= 0 {
        batchSize = 50 // Reduced from 100
    }
    if workerCount <= 0 {
        workerCount = 20 // Increased from 10
    }
    
    return &FastMessageProcessor{
        messageQueue:   messageQueue,
        channelCache:   channelCache,
        balanceManager: balanceManager,
        wsHub:          wsHub,
        logger:         logger,
        batchSize:      batchSize,
        workerCount:    workerCount,
        stopChan:       make(chan struct{}),
    }
}
```

2. **Add Configuration from Environment**:
```go
// In config.go
type WorkerConfig struct {
    BatchSize   int
    WorkerCount int
    TickerMs    int
}

func LoadWorkerConfig() WorkerConfig {
    return WorkerConfig{
        BatchSize:   getEnvAsInt("WORKER_BATCH_SIZE", 50),
        WorkerCount: getEnvAsInt("WORKER_COUNT", 20),
        TickerMs:    getEnvAsInt("WORKER_TICKER_MS", 100),
    }
}
```

3. **Add Dynamic Scaling** (Advanced):
```go
// Adjust worker count based on queue size
func (fp *FastMessageProcessor) adjustWorkerCount(ctx context.Context) {
    // Get total queue size across all tenants
    totalQueueSize := fp.getTotalQueueSize(ctx)
    
    optimalWorkers := 20 // Default
    if totalQueueSize > 10000 {
        optimalWorkers = 50 // Scale up for high load
    } else if totalQueueSize > 5000 {
        optimalWorkers = 30
    } else if totalQueueSize < 1000 {
        optimalWorkers = 10 // Scale down for low load
    }
    
    if optimalWorkers != fp.workerCount {
        fp.logger.Info("Adjusting worker count",
            zap.Int("old", fp.workerCount),
            zap.Int("new", optimalWorkers),
            zap.Int64("queue_size", totalQueueSize),
        )
        // Implement worker scaling logic
    }
}
```

4. **Add Performance Metrics**:
```go
type ProcessorMetrics struct {
    MessagesProcessed int64
    BatchesProcessed  int64
    AverageLatency    time.Duration
    Throughput        float64 // messages per second
}

func (fp *FastMessageProcessor) GetMetrics() ProcessorMetrics {
    // Return current metrics
}
```

---

### Bug 22: Redis SCAN Performance Issue

#### Bug Condition

**Already addressed in Bug 12** - SCAN with COUNT parameter.

#### Fix Implementation

**Status**: Fixed in Bug 12 implementation.

**Additional Enhancement** - Add Monitoring:
```go
func (pc *PricingCache) InvalidateAll(ctx context.Context) error {
    startTime := time.Now()
    pattern := fmt.Sprintf("%s:cache:pricing:*", AppNamespace)
    
    var cursor uint64
    var totalDeleted int
    
    for {
        keys, nextCursor, err := pc.client.Client.Scan(ctx, cursor, pattern, 1000).Result()
        if err != nil {
            return fmt.Errorf("scan keys: %w", err)
        }
        
        if len(keys) > 0 {
            pipe := pc.client.Client.Pipeline()
            for _, key := range keys {
                pipe.Del(ctx, key)
            }
            
            _, err = pipe.Exec(ctx)
            if err != nil {
                return fmt.Errorf("delete keys: %w", err)
            }
            
            totalDeleted += len(keys)
        }
        
        cursor = nextCursor
        if cursor == 0 {
            break
        }
    }
    
    duration := time.Since(startTime)
    // Log performance metrics
    if pc.logger != nil {
        pc.logger.Info("Cache invalidation complete",
            zap.Int("keys_deleted", totalDeleted),
            zap.Duration("duration", duration),
        )
    }
    
    return nil
}
```

---

### Bug 23: Stale Channel Cache

#### Bug Condition

المشكلة تحدث عندما يتم تكوين ذاكرة التخزين المؤقت للقنوات.

**Formal Specification:**
```
FUNCTION isBugCondition_23(input)
  INPUT: channel cache configuration
  OUTPUT: boolean
  
  RETURN channel cache uses TTL = 5 minutes
         AND channel goes offline during TTL period
         AND messages are routed to offline channel
END FUNCTION
```

**Examples:**
- Channel cached at T=0 (online) → Channel goes offline at T=1min → Cache still shows online until T=5min
- Messages sent to offline channel for 4 minutes → delivery failures
- User experience degraded

#### Expected Behavior

**Correct Behavior:**
عندما يتم تكوين ذاكرة التخزين المؤقت للقنوات، يجب أن يقوم النظام باستخدام TTL أقصر (مثل دقيقة واحدة) لتقليل البيانات القديمة مع الحفاظ على الأداء.

**Preservation Requirements:**
- أداء اختيار القنوات يجب أن يبقى سريعًا
- استعلامات قاعدة البيانات يجب أن تبقى محدودة
- توجيه الرسائل يجب أن يعمل بشكل صحيح


#### Hypothesized Root Cause

1. **Performance Optimization**: TTL طويل لتقليل استعلامات قاعدة البيانات
2. **Assumed Stability**: افتراض أن القنوات تبقى نشطة لفترات طويلة
3. **No Monitoring**: عدم وجود مراقبة لحالة القنوات

#### Fix Implementation

**File**: `services/backend/internal/redis/channel_cache.go`

**Changes Required:**

1. **Reduce TTL** (Already addressed in Bug 13):
```go
func NewChannelCache(client *Client) *ChannelCache {
    return &ChannelCache{
        client: client,
        ttl:    1 * time.Minute, // Reduced from 5 minutes
    }
}
```

2. **Add Active Invalidation**:
```go
// InvalidateChannel removes a specific channel from cache
func (cc *ChannelCache) InvalidateChannel(ctx context.Context, tenantID uuid.UUID) error {
    key := cc.getCacheKey(tenantID)
    return cc.client.Client.Del(ctx, key).Err()
}

// Call this when channel goes offline
func (h *ChannelHandler) UpdateChannelStatus(channelID uuid.UUID, isOnline bool) error {
    // Update database
    err := h.db.UpdateChannelStatus(channelID, isOnline)
    if err != nil {
        return err
    }
    
    // Invalidate cache if channel went offline
    if !isOnline {
        channel, _ := h.db.GetChannel(channelID)
        if channel != nil {
            _ = h.channelCache.InvalidateChannel(ctx, channel.TenantID)
        }
    }
    
    return nil
}
```

3. **Add Health Check**:
```go
// VerifyChannelOnline checks if cached channel is actually online
func (cc *ChannelCache) VerifyChannelOnline(ctx context.Context, channelID uuid.UUID) (bool, error) {
    query := `SELECT is_online FROM channels WHERE id = $1`
    
    var isOnline bool
    err := db.PgxPool.QueryRow(ctx, query, channelID).Scan(&isOnline)
    if err != nil {
        return false, err
    }
    
    return isOnline, nil
}

// Use in getOrSelectChannel
func (fp *FastMessageProcessor) getOrSelectChannel(ctx context.Context, tenantID uuid.UUID) (*redis.CachedChannel, error) {
    // Try cache first
    if fp.channelCache != nil {
        cached, err := fp.channelCache.Get(ctx, tenantID)
        if err == nil {
            // Verify channel is still online (every 10th request to reduce DB load)
            if rand.Intn(10) == 0 {
                isOnline, _ := fp.channelCache.VerifyChannelOnline(ctx, cached.ChannelID)
                if !isOnline {
                    // Invalidate and select new channel
                    _ = fp.channelCache.InvalidateChannel(ctx, tenantID)
                    cached = nil
                }
            }
            
            if cached != nil && cached.IsOnline {
                return cached, nil
            }
        }
    }
    
    // ... rest of function ...
}
```

---

### Bug 24: No Alerting Mechanism

#### Bug Condition

المشكلة تحدث عندما تحدث مشاكل حرجة في النظام.

**Formal Specification:**
```
FUNCTION isBugCondition_24(input)
  INPUT: critical system issue
  OUTPUT: boolean
  
  RETURN critical issue occurs (high error rate, queue backup, low balance)
         AND no alert is sent to administrators
         AND issue goes unnoticed until user reports
END FUNCTION
```

**Examples:**
- Error rate spikes to 50% → no alert → users affected for hours
- Queue backs up to 100K messages → no alert → processing delays
- Tenant balance reaches 0 → no alert → service interruption

#### Expected Behavior

**Correct Behavior:**
عندما تحدث مشاكل حرجة (معدل خطأ عالٍ، احتياطي قائمة الانتظار، رصيد منخفض)، يجب أن يقوم النظام بإرسال تنبيهات عبر القنوات المكونة (بريد إلكتروني، webhook، SMS).

**Preservation Requirements:**
- العمليات العادية يجب ألا تولد تنبيهات
- أداء النظام يجب ألا يتأثر بالمراقبة
- التنبيهات يجب أن تكون قابلة للتكوين

#### Hypothesized Root Cause

1. **Not Implemented**: نظام التنبيهات لم يتم تنفيذه بعد
2. **Development Phase**: مخطط للتنفيذ لاحقًا
3. **External Monitoring**: افتراض استخدام أدوات خارجية

#### Fix Implementation

**Create Alerting System**:

1. **Define Alert Types**:
```go
// internal/alerting/types.go
package alerting

type AlertLevel string

const (
    AlertLevelInfo     AlertLevel = "INFO"
    AlertLevelWarning  AlertLevel = "WARNING"
    AlertLevelCritical AlertLevel = "CRITICAL"
)

type Alert struct {
    Level       AlertLevel
    Title       string
    Message     string
    Timestamp   time.Time
    Metadata    map[string]interface{}
}

type AlertChannel interface {
    Send(alert Alert) error
}
```

2. **Implement Email Channel**:
```go
// internal/alerting/email.go
package alerting

import (
    "fmt"
    "net/smtp"
)

type EmailChannel struct {
    smtpHost     string
    smtpPort     int
    smtpUser     string
    smtpPassword string
    recipients   []string
}

func NewEmailChannel(host string, port int, user, password string, recipients []string) *EmailChannel {
    return &EmailChannel{
        smtpHost:     host,
        smtpPort:     port,
        smtpUser:     user,
        smtpPassword: password,
        recipients:   recipients,
    }
}

func (ec *EmailChannel) Send(alert Alert) error {
    subject := fmt.Sprintf("[%s] %s", alert.Level, alert.Title)
    body := fmt.Sprintf("Time: %s\n\nMessage: %s\n\nMetadata: %v",
        alert.Timestamp.Format(time.RFC3339),
        alert.Message,
        alert.Metadata,
    )
    
    message := fmt.Sprintf("Subject: %s\r\n\r\n%s", subject, body)
    
    auth := smtp.PlainAuth("", ec.smtpUser, ec.smtpPassword, ec.smtpHost)
    addr := fmt.Sprintf("%s:%d", ec.smtpHost, ec.smtpPort)
    
    return smtp.SendMail(addr, auth, ec.smtpUser, ec.recipients, []byte(message))
}
```

3. **Implement Webhook Channel**:
```go
// internal/alerting/webhook.go
package alerting

import (
    "bytes"
    "encoding/json"
    "net/http"
)

type WebhookChannel struct {
    url string
}

func NewWebhookChannel(url string) *WebhookChannel {
    return &WebhookChannel{url: url}
}

func (wc *WebhookChannel) Send(alert Alert) error {
    payload, err := json.Marshal(alert)
    if err != nil {
        return err
    }
    
    resp, err := http.Post(wc.url, "application/json", bytes.NewBuffer(payload))
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode >= 400 {
        return fmt.Errorf("webhook returned status %d", resp.StatusCode)
    }
    
    return nil
}
```

4. **Create Alert Manager**:
```go
// internal/alerting/manager.go
package alerting

import (
    "sync"
    "time"
    
    "go.uber.org/zap"
)

type Manager struct {
    channels []AlertChannel
    logger   *zap.Logger
    mu       sync.RWMutex
}

func NewManager(logger *zap.Logger) *Manager {
    return &Manager{
        channels: make([]AlertChannel, 0),
        logger:   logger,
    }
}

func (m *Manager) AddChannel(channel AlertChannel) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.channels = append(m.channels, channel)
}

func (m *Manager) SendAlert(level AlertLevel, title, message string, metadata map[string]interface{}) {
    alert := Alert{
        Level:     level,
        Title:     title,
        Message:   message,
        Timestamp: time.Now(),
        Metadata:  metadata,
    }
    
    m.mu.RLock()
    channels := m.channels
    m.mu.RUnlock()
    
    for _, channel := range channels {
        go func(ch AlertChannel) {
            if err := ch.Send(alert); err != nil {
                m.logger.Error("Failed to send alert",
                    zap.String("title", title),
                    zap.Error(err),
                )
            }
        }(channel)
    }
}
```

5. **Integrate with System**:
```go
// In fast_processor.go
func (fp *FastMessageProcessor) monitorQueueHealth(ctx context.Context) {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            fp.checkQueueBackup(ctx)
        }
    }
}

func (fp *FastMessageProcessor) checkQueueBackup(ctx context.Context) {
    activeTenants, _ := fp.getActiveTenants(ctx)
    
    for _, tenantID := range activeTenants {
        queueSize, err := fp.messageQueue.GetQueueSize(ctx, tenantID)
        if err != nil {
            continue
        }
        
        // Alert if queue is backing up
        if queueSize > 10000 {
            fp.alertManager.SendAlert(
                alerting.AlertLevelCritical,
                "Queue Backup Detected",
                fmt.Sprintf("Tenant %s has %d messages in queue", tenantID, queueSize),
                map[string]interface{}{
                    "tenant_id":  tenantID.String(),
                    "queue_size": queueSize,
                },
            )
        } else if queueSize > 5000 {
            fp.alertManager.SendAlert(
                alerting.AlertLevelWarning,
                "High Queue Size",
                fmt.Sprintf("Tenant %s has %d messages in queue", tenantID, queueSize),
                map[string]interface{}{
                    "tenant_id":  tenantID.String(),
                    "queue_size": queueSize,
                },
            )
        }
    }
}
```


---

### Bug 25: Missing Database Index

#### Bug Condition

المشكلة تحدث عندما يتم الاستعلام عن قاعدة البيانات بواسطة `event_custom_id`.

**Formal Specification:**
```
FUNCTION isBugCondition_25(input)
  INPUT: database query filtering by event_custom_id
  OUTPUT: boolean
  
  RETURN query filters by event_custom_id
         AND no index exists on messages.event_custom_id
         AND query performs full table scan
         AND performance is slow
END FUNCTION
```

**Examples:**
- Query: `SELECT * FROM messages WHERE event_custom_id = 'EV...'` → full table scan on 10M rows
- Dashboard loads event messages → 5 second delay
- API timeout on event queries

#### Expected Behavior

**Correct Behavior:**
عندما يتم إنشاء مخطط قاعدة البيانات، يجب أن يتضمن النظام فهرسًا على `messages.event_custom_id` للاستعلامات السريعة المستندة إلى الأحداث.

**Preservation Requirements:**
- الاستعلامات الموجودة يجب أن تستمر بالعمل
- أداء الاستعلامات الأخرى يجب ألا يتأثر
- عمليات الإدراج يجب أن تبقى سريعة

#### Hypothesized Root Cause

**Already Fixed in Migration 077** - الفهرس موجود في السطر 26:
```sql
CREATE INDEX IF NOT EXISTS idx_messages_event_custom_id 
    ON messages(event_custom_id);
```

#### Fix Implementation

**Status**: Index already exists in migration 077.

**Verification**:
```sql
-- Verify index exists
SELECT indexname, indexdef 
FROM pg_indexes 
WHERE tablename = 'messages' 
AND indexname = 'idx_messages_event_custom_id';

-- Test query performance
EXPLAIN ANALYZE 
SELECT * FROM messages 
WHERE event_custom_id = 'EV2601012312345000100';
```

**Additional Optimization** - Add Composite Index:
```sql
-- If queries often filter by event_custom_id AND status
CREATE INDEX IF NOT EXISTS idx_messages_event_status 
    ON messages(event_custom_id, status);

-- If queries often filter by event_custom_id AND created_at
CREATE INDEX IF NOT EXISTS idx_messages_event_created 
    ON messages(event_custom_id, created_at DESC);
```

---

## 📋 Integration Bugs (26-30)

### Bug 26: Hardcoded API URLs

#### Bug Condition

المشكلة تحدث عندما يتم تكوين نقاط نهاية API في `axios.js`.

**Formal Specification:**
```
FUNCTION isBugCondition_26(input)
  INPUT: API client configuration
  OUTPUT: boolean
  
  RETURN API base URL is hardcoded to localhost:8081
         AND not configurable via environment variables
         AND deployment to production requires code changes
END FUNCTION
```

**Examples:**
- Development: works with localhost:8081
- Production: needs https://api.example.com → requires code change
- Staging: needs https://staging-api.example.com → requires code change

#### Expected Behavior

**Correct Behavior:**
عندما يتم تكوين عميل API، يجب أن يقوم النظام بقراءة عنوان URL الأساسي للـ API من متغير البيئة (VITE_API_BASE_URL) بدلاً من القيم المشفرة.

**Preservation Requirements:**
- طلبات API يجب أن تستمر بالعمل
- أداء الطلبات يجب أن يبقى كما هو
- التوافق مع البيئات المختلفة يجب أن يتحسن

#### Hypothesized Root Cause

1. **Development Setup**: تكوين للتطوير المحلي
2. **Missing Config**: لم يتم إعداد متغيرات البيئة
3. **Quick Start**: إعداد سريع دون تكوين مناسب

#### Fix Implementation

**File**: `apps/client-web/src/config.js`

**Changes Required:**

1. **Update config.js**:
```javascript
// apps/client-web/src/config.js
export const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8081';
export const WS_BASE_URL = import.meta.env.VITE_WS_BASE_URL || 'ws://localhost:8081';

// Other configuration
export const APP_NAME = import.meta.env.VITE_APP_NAME || 'Al-Hudhud SMS';
export const APP_VERSION = import.meta.env.VITE_APP_VERSION || '1.0.0';
```

2. **Create .env files**:
```bash
# apps/client-web/.env.development
VITE_API_BASE_URL=http://localhost:8081
VITE_WS_BASE_URL=ws://localhost:8081
VITE_APP_NAME=Al-Hudhud SMS (Dev)

# apps/client-web/.env.production
VITE_API_BASE_URL=https://api.alhudhud.com
VITE_WS_BASE_URL=wss://api.alhudhud.com
VITE_APP_NAME=Al-Hudhud SMS

# apps/client-web/.env.staging
VITE_API_BASE_URL=https://staging-api.alhudhud.com
VITE_WS_BASE_URL=wss://staging-api.alhudhud.com
VITE_APP_NAME=Al-Hudhud SMS (Staging)
```

3. **Update axios.js** (if needed):
```javascript
import axios from 'axios';
import { API_BASE_URL } from '../config';

const apiClient = axios.create({
  baseURL: API_BASE_URL, // Already using config
  withCredentials: true,
  headers: {
    'Content-Type': 'application/json',
  },
});
```

4. **Same for admin-web**:
```javascript
// apps/admin-web/src/config.js
export const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8081';
```

5. **Update package.json scripts**:
```json
{
  "scripts": {
    "dev": "vite --mode development",
    "build": "vite build --mode production",
    "build:staging": "vite build --mode staging",
    "preview": "vite preview"
  }
}
```

---

### Bug 27: No WebSocket Auto-Reconnect

#### Bug Condition

المشكلة تحدث عندما ينقطع اتصال WebSocket.

**Formal Specification:**
```
FUNCTION isBugCondition_27(input)
  INPUT: WebSocket connection drop
  OUTPUT: boolean
  
  RETURN WebSocket connection drops
         AND no automatic reconnection is attempted
         AND user must manually refresh page
END FUNCTION
```

**Examples:**
- Network blip → WebSocket disconnects → no reconnect → user sees no updates
- Server restart → all connections drop → users must refresh
- Mobile switches networks → connection lost → no recovery

#### Expected Behavior

**Correct Behavior:**
عندما ينقطع اتصال WebSocket، يجب أن يقوم النظام بتنفيذ إعادة الاتصال التلقائي مع تأخير أسي (1s، 2s، 4s، 8s، max 30s).

**Preservation Requirements:**
- اتصالات WebSocket الناجحة يجب أن تستمر بالعمل
- معالجة الرسائل يجب أن تبقى كما هي
- أداء الاتصال يجب أن يبقى سريعًا

#### Hypothesized Root Cause

1. **Basic Implementation**: تنفيذ أساسي بدون إعادة اتصال
2. **Missing Feature**: لم يتم تنفيذه بعد
3. **Development Phase**: مخطط للتنفيذ لاحقًا

#### Fix Implementation

**File**: `apps/client-web/src/contexts/SocketContext.jsx`

**Changes Required:**

```javascript
import React, { createContext, useContext, useEffect, useRef, useState } from 'react';
import { WS_BASE_URL } from '../config';

const SocketContext = createContext(null);

export const useSocket = () => useContext(SocketContext);

export const SocketProvider = ({ children }) => {
  const [socket, setSocket] = useState(null);
  const [isConnected, setIsConnected] = useState(false);
  const [reconnectAttempt, setReconnectAttempt] = useState(0);
  const reconnectTimeoutRef = useRef(null);
  const socketRef = useRef(null);

  const connect = () => {
    try {
      const token = localStorage.getItem('token');
      if (!token) {
        console.log('No token available, skipping WebSocket connection');
        return;
      }

      const ws = new WebSocket(`${WS_BASE_URL}/ws?token=${token}`);
      socketRef.current = ws;

      ws.onopen = () => {
        console.log('WebSocket connected');
        setIsConnected(true);
        setReconnectAttempt(0); // Reset reconnect attempts on successful connection
        setSocket(ws);
      };

      ws.onclose = (event) => {
        console.log('WebSocket disconnected', event.code, event.reason);
        setIsConnected(false);
        setSocket(null);
        socketRef.current = null;

        // Attempt reconnection with exponential backoff
        scheduleReconnect();
      };

      ws.onerror = (error) => {
        console.error('WebSocket error:', error);
        ws.close();
      };

      ws.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data);
          handleMessage(message);
        } catch (error) {
          console.error('Failed to parse WebSocket message:', error);
        }
      };

    } catch (error) {
      console.error('Failed to create WebSocket connection:', error);
      scheduleReconnect();
    }
  };

  const scheduleReconnect = () => {
    // Clear any existing reconnect timeout
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
    }

    // Calculate backoff delay: 1s, 2s, 4s, 8s, 16s, 30s (max)
    const baseDelay = 1000;
    const maxDelay = 30000;
    const delay = Math.min(baseDelay * Math.pow(2, reconnectAttempt), maxDelay);

    console.log(`Scheduling reconnect in ${delay}ms (attempt ${reconnectAttempt + 1})`);

    reconnectTimeoutRef.current = setTimeout(() => {
      setReconnectAttempt(prev => prev + 1);
      connect();
    }, delay);
  };

  const disconnect = () => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }

    if (socketRef.current) {
      socketRef.current.close();
      socketRef.current = null;
    }

    setSocket(null);
    setIsConnected(false);
    setReconnectAttempt(0);
  };

  const handleMessage = (message) => {
    // Dispatch custom event for message handling
    window.dispatchEvent(new CustomEvent('websocket-message', { detail: message }));
  };

  const sendMessage = (message) => {
    if (socketRef.current && socketRef.current.readyState === WebSocket.OPEN) {
      socketRef.current.send(JSON.stringify(message));
      return true;
    } else {
      console.warn('WebSocket not connected, cannot send message');
      return false;
    }
  };

  useEffect(() => {
    connect();

    return () => {
      disconnect();
    };
  }, []);

  const value = {
    socket,
    isConnected,
    sendMessage,
    reconnect: connect,
  };

  return (
    <SocketContext.Provider value={value}>
      {children}
    </SocketContext.Provider>
  );
};
```

**Add Connection Status Indicator**:
```javascript
// In App.jsx or Layout component
import { useSocket } from './contexts/SocketContext';

function ConnectionStatus() {
  const { isConnected } = useSocket();

  if (isConnected) {
    return null; // Don't show anything when connected
  }

  return (
    <div className="fixed top-0 left-0 right-0 bg-yellow-500 text-white text-center py-2 z-50">
      <span>⚠️ Connection lost. Reconnecting...</span>
    </div>
  );
}
```


---

### Bug 28: No Message List Pagination

#### Bug Condition

المشكلة تحدث عندما يتم تحميل قوائم الرسائل.

**Formal Specification:**
```
FUNCTION isBugCondition_28(input)
  INPUT: message list load request
  OUTPUT: boolean
  
  RETURN message list is loaded
         AND all messages are loaded at once
         AND no pagination is implemented
         AND causes memory and performance issues
END FUNCTION
```

**Examples:**
- Tenant has 100K messages → all loaded at once → browser freezes
- API returns 100K records → 50MB response → slow network transfer
- Frontend renders 100K rows → UI unresponsive

#### Expected Behavior

**Correct Behavior:**
عندما يتم تحميل قوائم الرسائل، يجب أن يقوم النظام بتنفيذ التصفح مع حجم صفحة قابل للتكوين (افتراضي 50) ووظيفة تحميل المزيد.

**Preservation Requirements:**
- عرض الرسائل يجب أن يستمر بالعمل
- ترتيب الرسائل يجب أن يُحفظ
- أداء التحميل الأولي يجب أن يتحسن

#### Hypothesized Root Cause

1. **Simple Implementation**: تنفيذ بسيط للنموذج الأولي
2. **Small Dataset**: تم الاختبار مع بيانات صغيرة
3. **Missing Feature**: لم يتم تنفيذ التصفح بعد

#### Fix Implementation

**Backend Changes**:

1. **Add Pagination to API**:
```go
// internal/handlers/messages.go
type GetMessagesRequest struct {
    Page     int    `form:"page" binding:"min=1"`
    PageSize int    `form:"page_size" binding:"min=1,max=100"`
    Status   string `form:"status"`
    SortBy   string `form:"sort_by"`
    SortOrder string `form:"sort_order"`
}

type PaginatedResponse struct {
    Data       interface{} `json:"data"`
    Page       int         `json:"page"`
    PageSize   int         `json:"page_size"`
    TotalCount int64       `json:"total_count"`
    TotalPages int         `json:"total_pages"`
    HasNext    bool        `json:"has_next"`
    HasPrev    bool        `json:"has_prev"`
}

func (h *MessageHandler) GetMessages(c *gin.Context) {
    var req GetMessagesRequest
    if err := c.ShouldBindQuery(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    // Set defaults
    if req.Page == 0 {
        req.Page = 1
    }
    if req.PageSize == 0 {
        req.PageSize = 50
    }
    if req.SortBy == "" {
        req.SortBy = "created_at"
    }
    if req.SortOrder == "" {
        req.SortOrder = "DESC"
    }
    
    tenantID := c.GetString("tenant_id")
    
    // Get total count
    var totalCount int64
    countQuery := "SELECT COUNT(*) FROM messages WHERE tenant_id = $1"
    if req.Status != "" {
        countQuery += " AND status = $2"
        db.PgxPool.QueryRow(c, countQuery, tenantID, req.Status).Scan(&totalCount)
    } else {
        db.PgxPool.QueryRow(c, countQuery, tenantID).Scan(&totalCount)
    }
    
    // Calculate pagination
    offset := (req.Page - 1) * req.PageSize
    totalPages := int((totalCount + int64(req.PageSize) - 1) / int64(req.PageSize))
    
    // Build query
    query := `
        SELECT id, custom_id, phone_number, content, status, cost, created_at, updated_at
        FROM messages
        WHERE tenant_id = $1
    `
    args := []interface{}{tenantID}
    argIndex := 2
    
    if req.Status != "" {
        query += fmt.Sprintf(" AND status = $%d", argIndex)
        args = append(args, req.Status)
        argIndex++
    }
    
    // Add sorting
    orderBy, err := buildOrderByClause(req.SortBy, req.SortOrder)
    if err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    query += " " + orderBy
    
    // Add pagination
    query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
    args = append(args, req.PageSize, offset)
    
    // Execute query
    rows, err := db.PgxPool.Query(c, query, args...)
    if err != nil {
        c.JSON(500, gin.H{"error": "Failed to fetch messages"})
        return
    }
    defer rows.Close()
    
    messages := make([]Message, 0, req.PageSize)
    for rows.Next() {
        var msg Message
        err := rows.Scan(&msg.ID, &msg.CustomID, &msg.PhoneNumber, &msg.Content, 
            &msg.Status, &msg.Cost, &msg.CreatedAt, &msg.UpdatedAt)
        if err != nil {
            continue
        }
        messages = append(messages, msg)
    }
    
    // Build response
    response := PaginatedResponse{
        Data:       messages,
        Page:       req.Page,
        PageSize:   req.PageSize,
        TotalCount: totalCount,
        TotalPages: totalPages,
        HasNext:    req.Page < totalPages,
        HasPrev:    req.Page > 1,
    }
    
    c.JSON(200, response)
}
```

**Frontend Changes**:

2. **Update MessageHistory Component**:
```javascript
// apps/client-web/src/components/MessageHistory.jsx
import React, { useState, useEffect } from 'react';
import apiClient from '../api/axios';

function MessageHistory() {
  const [messages, setMessages] = useState([]);
  const [page, setPage] = useState(1);
  const [pageSize] = useState(50);
  const [totalCount, setTotalCount] = useState(0);
  const [totalPages, setTotalPages] = useState(0);
  const [loading, setLoading] = useState(false);
  const [hasNext, setHasNext] = useState(false);

  const loadMessages = async (pageNum = 1) => {
    setLoading(true);
    try {
      const response = await apiClient.get('/api/messages', {
        params: {
          page: pageNum,
          page_size: pageSize,
          sort_by: 'created_at',
          sort_order: 'DESC',
        },
      });

      setMessages(response.data.data);
      setPage(response.data.page);
      setTotalCount(response.data.total_count);
      setTotalPages(response.data.total_pages);
      setHasNext(response.data.has_next);
    } catch (error) {
      console.error('Failed to load messages:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadMessages();
  }, []);

  const handleNextPage = () => {
    if (hasNext) {
      loadMessages(page + 1);
    }
  };

  const handlePrevPage = () => {
    if (page > 1) {
      loadMessages(page - 1);
    }
  };

  return (
    <div className="message-history">
      <h2>Message History</h2>
      
      <div className="message-stats">
        <p>Total Messages: {totalCount}</p>
        <p>Page {page} of {totalPages}</p>
      </div>

      {loading ? (
        <div>Loading...</div>
      ) : (
        <>
          <table className="message-table">
            <thead>
              <tr>
                <th>ID</th>
                <th>Phone Number</th>
                <th>Content</th>
                <th>Status</th>
                <th>Cost</th>
                <th>Created At</th>
              </tr>
            </thead>
            <tbody>
              {messages.map((msg) => (
                <tr key={msg.id}>
                  <td>{msg.custom_id || msg.id}</td>
                  <td>{msg.phone_number}</td>
                  <td>{msg.content.substring(0, 50)}...</td>
                  <td>{msg.status}</td>
                  <td>{msg.cost}</td>
                  <td>{new Date(msg.created_at).toLocaleString()}</td>
                </tr>
              ))}
            </tbody>
          </table>

          <div className="pagination">
            <button 
              onClick={handlePrevPage} 
              disabled={page === 1 || loading}
            >
              Previous
            </button>
            <span>Page {page} of {totalPages}</span>
            <button 
              onClick={handleNextPage} 
              disabled={!hasNext || loading}
            >
              Next
            </button>
          </div>
        </>
      )}
    </div>
  );
}

export default MessageHistory;
```

**Alternative: Infinite Scroll**:
```javascript
// Using react-infinite-scroll-component
import InfiniteScroll from 'react-infinite-scroll-component';

function MessageHistoryInfinite() {
  const [messages, setMessages] = useState([]);
  const [page, setPage] = useState(1);
  const [hasMore, setHasMore] = useState(true);

  const loadMore = async () => {
    const response = await apiClient.get('/api/messages', {
      params: { page: page + 1, page_size: 50 },
    });

    setMessages([...messages, ...response.data.data]);
    setPage(page + 1);
    setHasMore(response.data.has_next);
  };

  return (
    <InfiniteScroll
      dataLength={messages.length}
      next={loadMore}
      hasMore={hasMore}
      loader={<h4>Loading...</h4>}
    >
      {messages.map((msg) => (
        <MessageCard key={msg.id} message={msg} />
      ))}
    </InfiniteScroll>
  );
}
```


---

### Bug 29: No API Rate Limiting

#### Bug Condition

المشكلة تحدث عندما تقوم لوحة الإدارة بإجراء طلبات API.

**Formal Specification:**
```
FUNCTION isBugCondition_29(input)
  INPUT: API requests from admin panel
  OUTPUT: boolean
  
  RETURN admin panel makes API requests
         AND no rate limiting is implemented
         AND allows potential abuse
END FUNCTION
```

**Examples:**
- Malicious user makes 10,000 requests/second → server overloaded
- Buggy script in infinite loop → API overwhelmed
- DDoS attack → service unavailable

#### Expected Behavior

**Correct Behavior:**
عندما تقوم لوحة الإدارة بإجراء طلبات API، يجب أن يقوم النظام بتنفيذ تحديد المعدل (مثل 100 طلب في الدقيقة لكل IP).

**Preservation Requirements:**
- الاستخدام الشرعي يجب أن يستمر بالعمل
- أداء API يجب أن يبقى سريعًا
- تجربة المستخدم يجب ألا تتأثر

#### Hypothesized Root Cause

1. **Not Implemented**: تحديد المعدل لم يتم تنفيذه بعد
2. **Development Phase**: مخطط للتنفيذ لاحقًا
3. **Trusted Environment**: افتراض بيئة موثوقة

#### Fix Implementation

**Backend Changes**:

1. **Install Rate Limiting Middleware**:
```bash
go get github.com/ulule/limiter/v3
go get github.com/ulule/limiter/v3/drivers/store/redis
```

2. **Create Rate Limiter Middleware**:
```go
// internal/middleware/rate_limiter.go
package middleware

import (
    "net/http"
    "time"
    
    "github.com/gin-gonic/gin"
    "github.com/ulule/limiter/v3"
    "github.com/ulule/limiter/v3/drivers/middleware/gin"
    "github.com/ulule/limiter/v3/drivers/store/redis"
)

// NewRateLimiter creates a rate limiter middleware
func NewRateLimiter(redisClient *redis.Client) gin.HandlerFunc {
    // Define rate: 100 requests per minute per IP
    rate := limiter.Rate{
        Period: 1 * time.Minute,
        Limit:  100,
    }
    
    // Create store
    store, err := redis.NewStoreWithOptions(redisClient, limiter.StoreOptions{
        Prefix:   "rate_limiter",
        MaxRetry: 3,
    })
    if err != nil {
        panic(err)
    }
    
    // Create limiter instance
    instance := limiter.New(store, rate)
    
    // Create middleware
    middleware := gin.NewMiddleware(instance)
    
    return middleware
}

// NewRateLimiterWithTiers creates tiered rate limiting
func NewRateLimiterWithTiers(redisClient *redis.Client) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Get user role from context
        role := c.GetString("role")
        
        var rate limiter.Rate
        switch role {
        case "admin":
            rate = limiter.Rate{Period: 1 * time.Minute, Limit: 1000}
        case "user":
            rate = limiter.Rate{Period: 1 * time.Minute, Limit: 100}
        default:
            rate = limiter.Rate{Period: 1 * time.Minute, Limit: 20}
        }
        
        store, _ := redis.NewStore(redisClient)
        instance := limiter.New(store, rate)
        
        // Get client IP
        clientIP := c.ClientIP()
        
        // Check rate limit
        context, err := instance.Get(c, clientIP)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Rate limiter error"})
            c.Abort()
            return
        }
        
        // Set rate limit headers
        c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", context.Limit))
        c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", context.Remaining))
        c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", context.Reset))
        
        // Check if limit exceeded
        if context.Reached {
            c.JSON(http.StatusTooManyRequests, gin.H{
                "error": "Rate limit exceeded",
                "retry_after": context.Reset,
            })
            c.Abort()
            return
        }
        
        c.Next()
    }
}
```

3. **Apply Middleware**:
```go
// In main.go or router setup
import "github.com/al-hudhud/backend/internal/middleware"

func setupRouter(redisClient *redis.Client) *gin.Engine {
    router := gin.Default()
    
    // Apply rate limiter to all API routes
    api := router.Group("/api")
    api.Use(middleware.NewRateLimiter(redisClient))
    
    // Or use tiered rate limiting
    // api.Use(middleware.NewRateLimiterWithTiers(redisClient))
    
    // Define routes
    api.GET("/messages", messageHandler.GetMessages)
    api.POST("/messages", messageHandler.CreateMessage)
    // ... other routes
    
    return router
}
```

4. **Add Rate Limit Bypass for Internal Services**:
```go
func RateLimiterWithBypass(redisClient *redis.Client) gin.HandlerFunc {
    rateLimiter := NewRateLimiter(redisClient)
    
    return func(c *gin.Context) {
        // Bypass rate limiting for internal services
        if c.GetHeader("X-Internal-Service") == "true" {
            internalSecret := c.GetHeader("X-Internal-Secret")
            if internalSecret == os.Getenv("INTERNAL_SERVICE_SECRET") {
                c.Next()
                return
            }
        }
        
        // Apply rate limiting
        rateLimiter(c)
    }
}
```

**Frontend Changes**:

5. **Handle Rate Limit Errors**:
```javascript
// In axios.js
apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response && error.response.status === 429) {
      const retryAfter = error.response.headers['x-ratelimit-reset'];
      const message = `Too many requests. Please try again in ${retryAfter} seconds.`;
      
      // Show user-friendly message
      toast.error(message);
      
      // Optionally implement automatic retry
      // return new Promise((resolve) => {
      //   setTimeout(() => {
      //     resolve(apiClient.request(error.config));
      //   }, retryAfter * 1000);
      // });
    }
    
    if (error.response && error.response.status === 401) {
      // ... existing 401 handling
    }
    
    return Promise.reject(error);
  }
);
```

---

### Bug 30: No Mobile Gateway Error Recovery

#### Bug Condition

المشكلة تحدث عندما يواجه Mobile Gateway أخطاء.

**Formal Specification:**
```
FUNCTION isBugCondition_30(input)
  INPUT: Mobile Gateway error
  OUTPUT: boolean
  
  RETURN Mobile Gateway encounters error
         AND no recovery mechanism exists
         AND requires manual intervention
END FUNCTION
```

**Examples:**
- Network error → gateway stops → requires app restart
- SMS send fails → no retry → message lost
- Connection timeout → gateway hangs → manual restart needed

#### Expected Behavior

**Correct Behavior:**
عندما يواجه Mobile Gateway أخطاء، يجب أن يقوم النظام بتنفيذ منطق إعادة المحاولة، والإبلاغ عن الأخطاء، والتدهور الرشيق.

**Preservation Requirements:**
- إرسال الرسائل الناجح يجب أن يستمر بالعمل
- أداء الإرسال يجب أن يبقى سريعًا
- تجربة المستخدم يجب أن تتحسن

#### Hypothesized Root Cause

1. **Basic Implementation**: تنفيذ أساسي بدون معالجة أخطاء متقدمة
2. **Mobile Constraints**: قيود منصة الهاتف المحمول
3. **Development Phase**: مخطط للتحسين لاحقًا

#### Fix Implementation

**Mobile Gateway Changes** (React Native):

1. **Add Error Recovery Service**:
```javascript
// apps/mobile_gateway/src/services/ErrorRecoveryService.js
class ErrorRecoveryService {
  constructor() {
    this.retryQueue = [];
    this.maxRetries = 3;
    this.retryDelay = 1000; // Start with 1 second
  }

  async handleError(error, operation, context) {
    console.error('Error occurred:', error, 'Operation:', operation);

    // Categorize error
    if (this.isNetworkError(error)) {
      return this.handleNetworkError(operation, context);
    } else if (this.isAuthError(error)) {
      return this.handleAuthError(operation, context);
    } else if (this.isSMSError(error)) {
      return this.handleSMSError(operation, context);
    } else {
      return this.handleUnknownError(error, operation, context);
    }
  }

  isNetworkError(error) {
    return error.message?.includes('network') || 
           error.message?.includes('timeout') ||
           error.code === 'ECONNREFUSED';
  }

  isAuthError(error) {
    return error.response?.status === 401 || 
           error.response?.status === 403;
  }

  isSMSError(error) {
    return error.code?.startsWith('SMS_');
  }

  async handleNetworkError(operation, context) {
    // Add to retry queue
    this.retryQueue.push({
      operation,
      context,
      attempts: 0,
      nextRetry: Date.now() + this.retryDelay,
    });

    // Start retry processor if not running
    this.startRetryProcessor();

    return { recovered: false, queued: true };
  }

  async handleAuthError(operation, context) {
    // Try to refresh token
    try {
      await this.refreshAuth();
      // Retry operation immediately
      return await operation(context);
    } catch (refreshError) {
      // Redirect to login
      NavigationService.navigate('Login');
      return { recovered: false, requiresLogin: true };
    }
  }

  async handleSMSError(operation, context) {
    // Log SMS error for debugging
    await this.logError('SMS_ERROR', {
      operation,
      context,
      timestamp: new Date().toISOString(),
    });

    // Check if SMS permission is still granted
    const hasPermission = await this.checkSMSPermission();
    if (!hasPermission) {
      // Request permission again
      await this.requestSMSPermission();
    }

    return { recovered: false, permissionIssue: !hasPermission };
  }

  async handleUnknownError(error, operation, context) {
    // Log error for analysis
    await this.logError('UNKNOWN_ERROR', {
      error: error.toString(),
      operation,
      context,
      timestamp: new Date().toISOString(),
    });

    return { recovered: false };
  }

  startRetryProcessor() {
    if (this.retryProcessorRunning) return;

    this.retryProcessorRunning = true;
    this.processRetryQueue();
  }

  async processRetryQueue() {
    while (this.retryQueue.length > 0) {
      const now = Date.now();
      const item = this.retryQueue[0];

      if (item.nextRetry > now) {
        // Wait until next retry time
        await new Promise(resolve => setTimeout(resolve, item.nextRetry - now));
        continue;
      }

      // Remove from queue
      this.retryQueue.shift();

      // Attempt retry
      try {
        await item.operation(item.context);
        console.log('Retry successful for operation:', item.operation.name);
      } catch (error) {
        item.attempts++;

        if (item.attempts < this.maxRetries) {
          // Re-queue with exponential backoff
          item.nextRetry = Date.now() + (this.retryDelay * Math.pow(2, item.attempts));
          this.retryQueue.push(item);
          console.log(`Retry ${item.attempts} failed, re-queuing`);
        } else {
          console.error('Max retries exceeded for operation:', item.operation.name);
          // Report permanent failure
          await this.reportPermanentFailure(item);
        }
      }
    }

    this.retryProcessorRunning = false;
  }

  async reportPermanentFailure(item) {
    // Send failure report to server
    try {
      await api.post('/api/gateway/report-failure', {
        operation: item.operation.name,
        context: item.context,
        attempts: item.attempts,
        timestamp: new Date().toISOString(),
      });
    } catch (error) {
      console.error('Failed to report permanent failure:', error);
    }
  }

  async logError(type, data) {
    // Log to local storage for later sync
    const errorLog = {
      type,
      data,
      timestamp: new Date().toISOString(),
    };

    try {
      const existingLogs = await AsyncStorage.getItem('error_logs');
      const logs = existingLogs ? JSON.parse(existingLogs) : [];
      logs.push(errorLog);

      // Keep only last 100 errors
      if (logs.length > 100) {
        logs.shift();
      }

      await AsyncStorage.setItem('error_logs', JSON.stringify(logs));
    } catch (error) {
      console.error('Failed to log error:', error);
    }
  }

  async syncErrorLogs() {
    try {
      const logs = await AsyncStorage.getItem('error_logs');
      if (!logs) return;

      await api.post('/api/gateway/error-logs', {
        logs: JSON.parse(logs),
      });

      // Clear logs after successful sync
      await AsyncStorage.removeItem('error_logs');
    } catch (error) {
      console.error('Failed to sync error logs:', error);
    }
  }
}

export default new ErrorRecoveryService();
```

2. **Integrate with SMS Sending**:
```javascript
// In SMSService.js
import ErrorRecoveryService from './ErrorRecoveryService';

async function sendSMS(phoneNumber, message) {
  try {
    // Attempt to send SMS
    await SmsAndroid.autoSend(
      phoneNumber,
      message,
      (fail) => {
        throw new Error(`SMS_SEND_FAILED: ${fail}`);
      },
      (success) => {
        console.log('SMS sent successfully');
      }
    );
  } catch (error) {
    // Handle error with recovery service
    const result = await ErrorRecoveryService.handleError(
      error,
      sendSMS,
      { phoneNumber, message }
    );

    if (!result.recovered) {
      throw error;
    }
  }
}
```

3. **Add Health Check**:
```javascript
// Add periodic health check
setInterval(async () => {
  try {
    await api.get('/api/gateway/health');
    // Sync error logs if connected
    await ErrorRecoveryService.syncErrorLogs();
  } catch (error) {
    console.log('Health check failed, will retry');
  }
}, 60000); // Every minute
```

---

## Correctness Properties


Property 1: Critical Bugs Fixed - Active Tenant Processing

_For any_ system state where tenants have pending messages in queues, the fixed getActiveTenants() function SHALL return the list of active tenant UUIDs from Redis, enabling message processing to proceed.

**Validates: Requirements 2.1, 2.2, 2.3, 2.4**

Property 2: Critical Bugs Fixed - Database Connection Stability

_For any_ database operation requiring PgxPool, the fixed initialization SHALL ensure PgxPool is properly initialized alongside GormDB, preventing nil pointer crashes.

**Validates: Requirements 2.2**

Property 3: Critical Bugs Fixed - Real Channel Selection

_For any_ message routing operation, the fixed getOrSelectChannel() function SHALL query the database for real available channels instead of returning mock data, ensuring messages are routed correctly.

**Validates: Requirements 2.3**

Property 4: Critical Bugs Fixed - Accurate Analytics

_For any_ analytics calculation request, the fixed GetCurrentLoad() function SHALL count actual messages from Redis and database, providing accurate load metrics.

**Validates: Requirements 2.4**

Property 5: High Priority Bugs Fixed - Error Handling

_For any_ critical operation that returns an error, the fixed code SHALL log the error appropriately and handle it (return, retry, or fail gracefully) instead of silently ignoring it.

**Validates: Requirements 2.5**

Property 6: High Priority Bugs Fixed - Transaction Management

_For any_ database transaction, the fixed code SHALL use proper defer patterns ensuring rollback only executes if commit fails, preventing transaction management issues.

**Validates: Requirements 2.6**

Property 7: High Priority Bugs Fixed - Balance Verification

_For any_ balance deduction attempt, the fixed DeductBalanceTx() function SHALL verify sufficient balance exists before deduction, preventing negative balances.

**Validates: Requirements 2.7**

Property 8: High Priority Bugs Fixed - Nil Pointer Safety

_For any_ WebSocket operation, the fixed code SHALL check if wsHub is not nil before calling methods, preventing nil pointer crashes.

**Validates: Requirements 2.8**

Property 9: High Priority Bugs Fixed - Redis Reliability

_For any_ Redis operation that fails with a transient error, the fixed code SHALL implement retry logic with exponential backoff, improving reliability.

**Validates: Requirements 2.9**

Property 10: High Priority Bugs Fixed - Priority Queue Ordering

_For any_ batch dequeue operation, the fixed DequeueBatch() function SHALL fully drain high-priority queue before moving to normal priority, then low priority, ensuring proper priority enforcement.

**Validates: Requirements 2.10**

Property 11: Medium Priority Bugs Fixed - Data Completeness

_For any_ existing database records when migration 077 runs, the fixed migration SHALL backfill custom_id values using a generation function, ensuring data completeness.

**Validates: Requirements 2.11**

Property 12: Medium Priority Bugs Fixed - Cache Performance

_For any_ cache invalidation operation, the fixed InvalidateAll() function SHALL use SCAN with COUNT parameter for efficient pagination, preventing performance issues.

**Validates: Requirements 2.12**

Property 13: Medium Priority Bugs Fixed - Cache Coherence

_For any_ balance and usage cache operations, the fixed code SHALL use consistent TTL values (both 1 minute) to maintain cache coherence.

**Validates: Requirements 2.13**

Property 14: Medium Priority Bugs Fixed - Complete Analytics

_For any_ analytics calculation, the fixed code SHALL query actual message counts from Redis or database and aggregate them correctly, providing complete metrics.

**Validates: Requirements 2.14**

Property 15: Medium Priority Bugs Fixed - Session Management

_For any_ 401 Unauthorized response, the fixed axios interceptor SHALL clear localStorage token and redirect to login page automatically, improving security and UX.

**Validates: Requirements 2.15**

Property 16: Security Bugs Fixed - Credential Protection

_For any_ deployment script execution, the fixed scripts SHALL read credentials from environment variables or secure vaults instead of hardcoded values, preventing credential exposure.

**Validates: Requirements 2.16**

Property 17: Security Bugs Fixed - CSRF Protection

_For any_ API request with credentials, the fixed system SHALL implement CSRF token validation using double-submit cookie pattern, preventing CSRF attacks.

**Validates: Requirements 2.17**

Property 18: Security Bugs Fixed - XSS Protection

_For any_ authentication token storage, the fixed system SHALL use httpOnly secure cookies instead of localStorage, preventing XSS attacks.

**Validates: Requirements 2.18**

Property 19: Security Bugs Fixed - SQL Injection Prevention

_For any_ SQL query construction, the fixed code SHALL use parameterized queries exclusively with validated dynamic query construction, preventing SQL injection.

**Validates: Requirements 2.19**

Property 20: Security Bugs Fixed - Configuration Validation

_For any_ application startup, the fixed code SHALL validate all required environment variables and fail fast with clear error messages if missing, preventing runtime configuration errors.

**Validates: Requirements 2.20**

Property 21: Performance Bugs Fixed - Optimized Throughput

_For any_ batch processing configuration, the fixed system SHALL use optimized batch size and worker count based on load testing, improving parallelism and throughput.

**Validates: Requirements 2.21**

Property 22: Performance Bugs Fixed - Efficient Cache Operations

_For any_ Redis cache clearing operation, the fixed code SHALL use SCAN with COUNT=1000 and process in batches, avoiding Redis blocking.

**Validates: Requirements 2.22**

Property 23: Performance Bugs Fixed - Fresh Channel Data

_For any_ channel cache configuration, the fixed system SHALL use shorter TTL (1 minute) to reduce stale data while maintaining performance.

**Validates: Requirements 2.23**

Property 24: Performance Bugs Fixed - Proactive Monitoring

_For any_ critical system issue (high error rate, queue backup, low balance), the fixed system SHALL send alerts via configured channels, enabling proactive response.

**Validates: Requirements 2.24**

Property 25: Performance Bugs Fixed - Query Optimization

_For any_ database schema creation, the fixed migration SHALL include index on messages.event_custom_id for fast event-based queries.

**Validates: Requirements 2.25**

Property 26: Integration Bugs Fixed - Environment Configuration

_For any_ API client configuration, the fixed code SHALL read API base URL from environment variable instead of hardcoded values, enabling proper deployment.

**Validates: Requirements 2.26**

Property 27: Integration Bugs Fixed - Connection Resilience

_For any_ WebSocket connection drop, the fixed system SHALL implement automatic reconnection with exponential backoff, improving reliability.

**Validates: Requirements 2.27**

Property 28: Integration Bugs Fixed - Scalable Data Loading

_For any_ message list load operation, the fixed system SHALL implement pagination with configurable page size and load-more functionality, preventing memory issues.

**Validates: Requirements 2.28**

Property 29: Integration Bugs Fixed - API Protection

_For any_ admin panel API requests, the fixed system SHALL implement rate limiting (100 requests per minute per IP), preventing abuse.

**Validates: Requirements 2.29**

Property 30: Integration Bugs Fixed - Error Recovery

_For any_ Mobile Gateway error, the fixed system SHALL implement retry logic, error reporting, and graceful degradation, improving reliability.

**Validates: Requirements 2.30**

Property 31: Preservation - Message Processing

_For any_ message that is successfully enqueued and processed, the fixed system SHALL continue to maintain FIFO order within each priority level, update status correctly, and deliver via WebSocket as before.

**Validates: Requirements 3.1, 3.2, 3.3**

Property 32: Preservation - Financial Operations

_For any_ successful balance deduction or refund operation, the fixed system SHALL continue to create accounting transactions, update balances, and log transactions correctly as before.

**Validates: Requirements 3.4, 3.5**

Property 33: Preservation - Caching Behavior

_For any_ cache hit operation (pricing, channel, etc.), the fixed system SHALL continue to return cached results without recalculation or database query as before.

**Validates: Requirements 3.6, 3.7, 3.8**

Property 34: Preservation - Authentication

_For any_ valid JWT token or API key, the fixed system SHALL continue to authenticate users, extract tenant context, and allow access as before.

**Validates: Requirements 3.9, 3.10, 3.11**

Property 35: Preservation - Database Operations

_For any_ GORM query or migration, the fixed system SHALL continue to enforce RLS policies, create schema correctly, and persist changes atomically as before.

**Validates: Requirements 3.12, 3.13, 3.14**

Property 36: Preservation - User Interfaces

_For any_ user login, dashboard load, or Excel upload, the fixed system SHALL continue to function correctly as before.

**Validates: Requirements 3.15, 3.16, 3.17**

Property 37: Preservation - Monitoring

_For any_ metrics collection, health check, or audit log creation, the fixed system SHALL continue to function correctly as before.

**Validates: Requirements 3.18, 3.19, 3.20**

---

## Testing Strategy

### Validation Approach

The testing strategy follows a three-phase approach for each bug fix:

1. **Exploratory Testing**: Surface counterexamples on unfixed code to confirm root cause
2. **Fix Verification**: Verify the fix resolves the bug condition
3. **Preservation Testing**: Ensure existing functionality remains unchanged

### Test Organization

Tests are organized by bug category:
- **Unit Tests**: Test individual functions and components
- **Integration Tests**: Test interactions between components
- **Property-Based Tests**: Generate random inputs to verify properties
- **End-to-End Tests**: Test complete user flows

### Exploratory Bug Condition Checking

**Goal**: Confirm root cause analysis by demonstrating bugs on unfixed code.

**Critical Bugs (1-4)**:
- Test 1.1: Call getActiveTenants() with populated Redis set → expect empty list (bug)
- Test 1.2: Access db.PgxPool → expect nil pointer crash (bug)
- Test 1.3: Call getOrSelectChannel() → expect mock UUID returned (bug)
- Test 1.4: Call GetCurrentLoad() → expect MessageCount = 0 despite messages (bug)

**High Priority Bugs (5-10)**:
- Test 2.1: Trigger error in critical operation → expect silent failure (bug)
- Test 2.2: Commit transaction → expect rollback executed after (bug)
- Test 2.3: Deduct balance with insufficient funds → expect negative balance (bug)
- Test 2.4: Call wsHub.SendMessage() with nil wsHub → expect crash (bug)
- Test 2.5: Fail Redis operation → expect no retry (bug)
- Test 2.6: Dequeue from mixed priority queues → verify strict ordering

**Medium Priority Bugs (11-15)**:
- Test 3.1: Run migration 077 on existing data → expect NULL custom_id (bug)
- Test 3.2: Call InvalidateAll() → measure performance with large key set
- Test 3.3: Cache balance and usage → verify TTL inconsistency
- Test 3.4: Calculate analytics → expect incomplete message count (bug)
- Test 3.5: Receive 401 response → expect no auto-logout (bug)

**Security Bugs (16-20)**:
- Test 4.1: Check deployment scripts → expect hardcoded credentials (bug)
- Test 4.2: Make API request → expect no CSRF validation (bug)
- Test 4.3: Check token storage → expect localStorage usage (bug)
- Test 4.4: Audit SQL queries → identify injection vulnerabilities
- Test 4.5: Start app without env vars → expect runtime errors (bug)

**Performance Bugs (21-25)**:
- Test 5.1: Measure throughput with current config → compare to optimized
- Test 5.2: Measure SCAN performance → verify COUNT usage
- Test 5.3: Test channel cache with offline channel → expect stale data (bug)
- Test 5.4: Trigger critical issue → expect no alert (bug)
- Test 5.5: Query by event_custom_id → measure performance

**Integration Bugs (26-30)**:
- Test 6.1: Check API URL config → expect hardcoded value (bug)
- Test 6.2: Drop WebSocket connection → expect no reconnect (bug)
- Test 6.3: Load large message list → expect memory issues (bug)
- Test 6.4: Make many API requests → expect no rate limiting (bug)
- Test 6.5: Trigger Mobile Gateway error → expect no recovery (bug)

### Fix Checking

**Goal**: Verify each fix resolves its bug condition.

**Approach**: For each bug, write tests that:
1. Set up the bug condition
2. Execute the fixed code
3. Assert the expected correct behavior

**Example for Bug 1**:
```go
func TestGetActiveTenants_Fixed(t *testing.T) {
    // Setup: Add tenants to Redis set
    ctx := context.Background()
    redisClient.SAdd(ctx, "alhudhud:active:tenants", 
        "tenant-uuid-1", "tenant-uuid-2")
    
    // Execute: Call fixed function
    tenants, err := processor.getActiveTenants(ctx)
    
    // Assert: Returns tenant list
    assert.NoError(t, err)
    assert.Len(t, tenants, 2)
    assert.Contains(t, tenants, uuid.MustParse("tenant-uuid-1"))
}
```

### Preservation Checking

**Goal**: Verify existing functionality remains unchanged.

**Approach**: For each category of preserved behavior:
1. Identify critical existing functionality
2. Write tests that verify behavior matches original
3. Run tests on both original and fixed code

**Property-Based Testing Recommended** for:
- Message processing order preservation
- Balance calculation correctness
- Cache hit behavior
- Authentication flow
- Database query results

**Example Preservation Test**:
```go
func TestMessageProcessing_Preserved(t *testing.T) {
    // Test that FIFO order is maintained
    messages := generateTestMessages(100)
    
    for _, msg := range messages {
        messageQueue.Enqueue(ctx, msg)
    }
    
    dequeued, _ := messageQueue.DequeueBatch(ctx, tenantID, 100)
    
    // Assert: Order preserved (FIFO within priority)
    for i := 0; i < len(dequeued)-1; i++ {
        assert.True(t, dequeued[i].EnqueuedAt.Before(dequeued[i+1].EnqueuedAt))
    }
}
```

### Test Coverage Goals

- **Unit Tests**: 80% code coverage minimum
- **Integration Tests**: All critical paths covered
- **Property-Based Tests**: Key invariants verified
- **End-to-End Tests**: Major user flows tested

### Continuous Testing

- Run unit tests on every commit
- Run integration tests on every PR
- Run full test suite nightly
- Monitor test execution time and optimize slow tests

---

## Implementation Plan

### Phase 1: Critical Bugs (Week 1)
- Fix bugs 1-4 (Active Tenants, PgxPool, Channel Selection, Analytics)
- Write and run exploratory tests
- Implement fixes
- Verify with fix checking tests
- Run preservation tests

### Phase 2: High Priority Bugs (Week 2)
- Fix bugs 5-10 (Error Handling, Transactions, Balance, wsHub, Redis Retry, Priority Queues)
- Follow same testing approach
- Deploy to staging for integration testing

### Phase 3: Medium Priority Bugs (Week 3)
- Fix bugs 11-15 (Custom ID Backfill, SCAN, Cache TTL, Analytics, Auto-Logout)
- Test thoroughly
- Monitor staging performance

### Phase 4: Security Bugs (Week 4)
- Fix bugs 16-20 (Credentials, CSRF, XSS, SQL Injection, Env Validation)
- Security audit
- Penetration testing

### Phase 5: Performance Bugs (Week 5)
- Fix bugs 21-25 (Batch Config, SCAN, Channel Cache, Alerting, Index)
- Load testing
- Performance benchmarking

### Phase 6: Integration Bugs (Week 6)
- Fix bugs 26-30 (API URLs, WebSocket, Pagination, Rate Limiting, Mobile Recovery)
- End-to-end testing
- User acceptance testing

### Phase 7: Production Deployment (Week 7)
- Final testing
- Gradual rollout
- Monitoring and alerting
- Rollback plan ready

---

## Success Criteria

- All 30 bugs fixed and verified
- All tests passing (unit, integration, property-based, e2e)
- No regressions in existing functionality
- Performance metrics improved or maintained
- Security vulnerabilities addressed
- System stable in production for 1 week

---

## Rollback Plan

If critical issues are discovered after deployment:

1. **Immediate**: Revert to previous version
2. **Investigate**: Identify root cause of issue
3. **Fix**: Apply targeted fix
4. **Test**: Verify fix in staging
5. **Redeploy**: Gradual rollout with monitoring

---

## Monitoring and Alerting

Post-deployment monitoring:

- Error rate by endpoint
- Response time percentiles (p50, p95, p99)
- Queue sizes and processing rates
- Database connection pool usage
- Redis memory usage
- WebSocket connection count
- Message delivery success rate
- Balance deduction accuracy
- Cache hit rates

Alerts configured for:
- Error rate > 5%
- Response time p95 > 1s
- Queue size > 10,000
- Database connections > 80% of pool
- Redis memory > 80%
- WebSocket disconnections > 10/min
- Message delivery failure rate > 1%

