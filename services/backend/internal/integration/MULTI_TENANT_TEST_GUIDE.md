# Multi-Tenant Scenarios Test - Complete Guide

## المقدمة

هذا الدليل الشامل يغطي جميع جوانب اختبار Multi-Tenant Scenarios (المهمة 15.4)، بما في ذلك التصميم، التنفيذ، والتحقق من النتائج.

## أهمية العزل بين الشركات (Multi-Tenancy Isolation)

### لماذا العزل مهم؟

1. **الأمان المالي**: منع tenant من الوصول لرصيد tenant آخر
2. **الخصوصية**: منع تسرب البيانات بين الشركات
3. **العدالة**: ضمان عدم تأثير tenant على أداء tenant آخر
4. **الامتثال**: الالتزام بمتطلبات حماية البيانات (GDPR, etc.)

### مستويات العزل في النظام

#### 1. Database Level Isolation
```sql
-- جميع الـ queries تحتوي على WHERE tenant_id = ?
SELECT * FROM messages WHERE tenant_id = $1;
UPDATE tenants SET balance = balance - $1 WHERE id = $2;
INSERT INTO balance_history (tenant_id, ...) VALUES ($1, ...);
```

#### 2. Redis Level Isolation
```
# Keys منفصلة لكل tenant
messages:pending:{tenant_id}
rate_limit:{tenant_id}
channels:cache:{tenant_id}
```

#### 3. Application Level Isolation
```go
// جميع الـ operations تتحقق من tenant_id
func (bm *BalanceManager) DeductBalance(ctx context.Context, op BalanceOperation) {
    // Verify tenant_id in operation
    // Execute with WHERE tenant_id = ?
}
```

#### 4. Rate Limiting Isolation
```go
// Rate limiting منفصل لكل tenant
rateLimitKey := fmt.Sprintf("rate_limit:%d", tenantID)
```

## بنية الاختبار

### Test 1: TestMultiTenantScenarios

#### الهدف
اختبار شامل للعزل في جميع مكونات النظام.

#### السيناريو
```
3 Tenants × 50 Messages = 150 Total Messages
+ Concurrent sending
+ Independent processing
+ Isolated storage
```

#### المكونات المختبرة

1. **API Gateway**
   - استقبال رسائل من tenants مختلفة
   - Rate limiting منفصل لكل tenant
   - Validation منفصلة

2. **Redis Buffer**
   - Queue منفصلة لكل tenant
   - عدم تداخل البيانات
   - Keys معزولة

3. **Batch Processor**
   - معالجة منفصلة لكل tenant
   - عدم تأثير tenant على آخر
   - Throughput مستقل

4. **Database**
   - رسائل معزولة لكل tenant
   - عدم cross-tenant queries
   - Indexes تدعم العزل

5. **Balance Manager**
   - رصيد منفصل لكل tenant
   - عمليات ذرية معزولة
   - History منفصلة

6. **Routing Engine**
   - قنوات معزولة لكل tenant
   - Cache منفصل
   - Load balancing مستقل

#### مراحل الاختبار

**Phase 1: Setup (الإعداد)**
```go
// إنشاء 3 tenants مع أرصدة مختلفة
tenants := []struct {
    ID      int64
    Balance decimal.Decimal
}{
    {1, decimal.NewFromFloat(100.0)},
    {2, decimal.NewFromFloat(150.0)},
    {3, decimal.NewFromFloat(200.0)},
}
```

**Phase 2: Concurrent Sending (الإرسال المتزامن)**
```go
// إرسال 50 رسالة من كل tenant بشكل متزامن
var wg sync.WaitGroup
for _, tenant := range tenants {
    wg.Add(1)
    go func(t Tenant) {
        defer wg.Done()
        for i := 0; i < 50; i++ {
            apiGateway.ReceiveMessage(ctx, MessageRequest{
                TenantID: t.ID,
                // ...
            })
        }
    }(tenant)
}
wg.Wait()
```

**Phase 3: Buffer Isolation (عزل Buffer)**
```go
// التحقق من أن كل tenant له buffer منفصل
for _, tenant := range tenants {
    size := messageBuffer.GetSize(ctx, tenant.ID)
    assert.Equal(t, 50, size) // كل tenant له 50 رسالة
}
```

**Phase 4: Processing (المعالجة)**
```go
// معالجة الرسائل لكل tenant بشكل منفصل
for _, tenant := range tenants {
    batchProcessor.ProcessBatch(ctx, tenant.ID)
}
```

**Phase 5: Database Isolation (عزل قاعدة البيانات)**
```go
// التحقق من عدم وجود cross-tenant data
for _, tenant := range tenants {
    // عدد رسائل هذا الـ tenant
    count := countMessages(tenant.ID)
    assert.Equal(t, 50, count)
    
    // التحقق من عدم وجود رسائل من tenants أخرى
    for _, other := range tenants {
        if other.ID != tenant.ID {
            crossCount := countCrossTenantMessages(tenant.ID, other.ID)
            assert.Equal(t, 0, crossCount)
        }
    }
}
```

**Phase 6: Balance Isolation (عزل الرصيد)**
```go
// التحقق من أن الرصيد لم يتأثر بـ tenants أخرى
for _, tenant := range tenants {
    balance := balanceManager.GetBalance(ctx, tenant.ID)
    assert.Equal(t, tenant.InitialBalance, balance)
}
```

**Phase 7: Channel Isolation (عزل القنوات)**
```go
// التحقق من أن كل tenant له قنواته الخاصة
for _, tenant := range tenants {
    channels := routingEngine.GetAvailableChannels(ctx, tenant.ID)
    assert.Len(t, channels, 1)
    assert.Equal(t, tenant.ID, channels[0].TenantID)
}
```

**Phase 8: Rate Limiting Isolation (عزل Rate Limiting)**
```go
// إرسال رسائل كثيرة من tenant واحد لتجاوز الحد
for i := 0; i < 150; i++ {
    _, err := apiGateway.ReceiveMessage(ctx, MessageRequest{
        TenantID: tenants[0].ID,
        // ...
    })
    if err != nil {
        // Rate limit hit
        break
    }
}

// التحقق من أن tenants الأخرى لم تتأثر
for _, tenant := range tenants[1:] {
    _, err := apiGateway.ReceiveMessage(ctx, MessageRequest{
        TenantID: tenant.ID,
        // ...
    })
    assert.NoError(t, err) // يجب أن ينجح
}
```

**Phase 9: Redis Key Isolation (عزل مفاتيح Redis)**
```go
// التحقق من أن كل tenant له مفاتيح منفصلة
for _, tenant := range tenants {
    key := fmt.Sprintf("messages:pending:%d", tenant.ID)
    exists := redisClient.Exists(ctx, key)
    assert.True(t, exists)
}
```

### Test 2: TestMultiTenantBalanceIsolation

#### الهدف
اختبار مكثف لعزل العمليات المالية.

#### السيناريو
```
5 Tenants × 100 Operations = 500 Total Operations
+ Concurrent execution
+ Atomic operations
+ Isolated history
```

#### المكونات المختبرة

1. **Atomic Balance Operations**
   ```sql
   UPDATE tenants 
   SET balance = balance - $1
   WHERE id = $2 AND balance >= $1
   RETURNING balance;
   ```

2. **Balance History Logging**
   ```sql
   INSERT INTO balance_history (
       tenant_id, message_id, old_balance, new_balance,
       amount, type, reason, created_by
   ) VALUES (...);
   ```

3. **Concurrent Safety**
   - عمليات متزامنة من عدة tenants
   - عدم race conditions
   - نتائج صحيحة دائماً

#### مراحل الاختبار

**Phase 1: Setup**
```go
// إنشاء 5 tenants مع رصيد متساوي
tenants := make([]*TenantBalance, 5)
for i := 0; i < 5; i++ {
    tenants[i] = &TenantBalance{
        ID:             createTestTenant(t, ctx, db, decimal.NewFromFloat(100.0)),
        InitialBalance: decimal.NewFromFloat(100.0),
    }
}
```

**Phase 2: Concurrent Operations**
```go
// تنفيذ 100 عملية خصم لكل tenant بشكل متزامن
var wg sync.WaitGroup
for _, tenant := range tenants {
    wg.Add(1)
    go func(t *TenantBalance) {
        defer wg.Done()
        for i := 0; i < 100; i++ {
            op := BalanceOperation{
                TenantID:  t.ID,
                MessageID: fmt.Sprintf("msg-%d-%d", t.ID, i),
                Amount:    decimal.NewFromFloat(0.05),
                Type:      "DEDUCT",
            }
            _, err := balanceManager.DeductBalance(ctx, op)
            if err == nil {
                t.SuccessOps.Add(1)
            } else {
                t.FailedOps.Add(1)
            }
        }
    }(tenant)
}
wg.Wait()
```

**Phase 3: Verify Balance Correctness**
```go
// التحقق من صحة الرصيد النهائي
for _, tenant := range tenants {
    currentBalance := balanceManager.GetBalance(ctx, tenant.ID)
    
    // الرصيد المتوقع = الرصيد الأولي - (عدد العمليات الناجحة × المبلغ)
    expectedBalance := tenant.InitialBalance.Sub(
        decimal.NewFromInt(int64(tenant.SuccessOps.Load())).
            Mul(decimal.NewFromFloat(0.05)),
    )
    
    assert.True(t, currentBalance.Equal(expectedBalance))
}
```

**Phase 4: Verify History Isolation**
```go
// التحقق من عدم وجود cross-tenant history
for _, tenant := range tenants {
    // عدد سجلات هذا الـ tenant
    historyCount := countBalanceHistory(tenant.ID)
    assert.Equal(t, tenant.SuccessOps.Load(), historyCount)
    
    // التحقق من عدم وجود سجلات من tenants أخرى
    crossHistory := countCrossTenantHistory(tenant.ID)
    assert.Equal(t, 0, crossHistory)
}
```

## التحقق من النتائج

### 1. Database Verification

#### Messages Table
```sql
-- عدد الرسائل لكل tenant
SELECT 
    tenant_id,
    COUNT(*) as message_count,
    COUNT(DISTINCT phone_number) as unique_phones,
    MIN(created_at) as first_message,
    MAX(created_at) as last_message
FROM messages
WHERE deleted_at IS NULL
GROUP BY tenant_id
ORDER BY tenant_id;

-- Expected output:
-- tenant_id | message_count | unique_phones | first_message | last_message
-- ----------|---------------|---------------|---------------|-------------
--         1 |            50 |            50 | 2024-01-...   | 2024-01-...
--         2 |            50 |            50 | 2024-01-...   | 2024-01-...
--         3 |            50 |            50 | 2024-01-...   | 2024-01-...
```

#### Cross-Tenant Data Check
```sql
-- يجب أن يكون النتيجة فارغة (no cross-tenant data)
SELECT 
    m1.tenant_id as tenant_1,
    m2.tenant_id as tenant_2,
    COUNT(*) as cross_tenant_count
FROM messages m1
JOIN messages m2 ON m1.phone_number = m2.phone_number
WHERE m1.tenant_id != m2.tenant_id
  AND m1.deleted_at IS NULL
  AND m2.deleted_at IS NULL
GROUP BY m1.tenant_id, m2.tenant_id;

-- Expected: No rows (empty result)
```

#### Balance History
```sql
-- سجلات الرصيد لكل tenant
SELECT 
    tenant_id,
    COUNT(*) as history_count,
    SUM(CASE WHEN type = 'DEDUCT' THEN amount ELSE 0 END) as total_deducted,
    SUM(CASE WHEN type = 'REFUND' THEN amount ELSE 0 END) as total_refunded,
    MIN(created_at) as first_operation,
    MAX(created_at) as last_operation
FROM balance_history
GROUP BY tenant_id
ORDER BY tenant_id;
```

#### Tenants Balance
```sql
-- الرصيد الحالي لكل tenant
SELECT 
    id as tenant_id,
    balance,
    created_at,
    updated_at
FROM tenants
WHERE deleted_at IS NULL
ORDER BY id;
```

### 2. Redis Verification

```bash
# الاتصال بـ Redis
docker exec -it redis redis-cli

# فحص جميع المفاتيح
KEYS *

# Expected keys:
# messages:pending:1
# messages:pending:2
# messages:pending:3
# rate_limit:1
# rate_limit:2
# rate_limit:3
# channels:cache:1
# channels:cache:2
# channels:cache:3

# فحص حجم كل buffer
LLEN messages:pending:1
LLEN messages:pending:2
LLEN messages:pending:3

# فحص rate limiting
GET rate_limit:1
GET rate_limit:2
GET rate_limit:3

# فحص TTL
TTL rate_limit:1
TTL channels:cache:1
```

### 3. Application Logs Verification

```bash
# فحص logs للتحقق من العزل
docker logs backend | grep "tenant_id"

# Expected patterns:
# Processing message for tenant_id=1
# Balance deducted for tenant_id=1
# Channel selected for tenant_id=1
# Rate limit checked for tenant_id=1
```

## مقاييس الأداء

### Throughput
```
Total Messages: 150 (3 tenants × 50 messages)
Total Time: ~5-10 seconds
Throughput: 900-1800 messages/minute
```

### Latency
```
API Response Time: < 10ms per message
Batch Processing: ~100ms per batch
Total E2E Time: < 2 seconds per message
```

### Concurrency
```
Concurrent Tenants: 3
Concurrent Operations: 150
No race conditions
No deadlocks
```

## استكشاف الأخطاء

### Problem 1: Cross-Tenant Data Leakage

**الأعراض:**
```
❌ Tenant-1 should have exactly 50 messages in database
   Expected: 50
   Actual: 75
```

**الأسباب المحتملة:**
1. Query لا يحتوي على `WHERE tenant_id = ?`
2. Bug في Batch Processor
3. خطأ في Buffer isolation

**الحل:**
```go
// تأكد من أن جميع الـ queries تحتوي على tenant_id
db.Query(ctx, `
    SELECT * FROM messages 
    WHERE tenant_id = $1  -- CRITICAL
    AND deleted_at IS NULL
`, tenantID)
```

### Problem 2: Balance Calculation Error

**الأعراض:**
```
❌ Tenant-1 balance should be 95.00, got 90.00
```

**الأسباب المحتملة:**
1. Race condition في balance deduction
2. عمليات من tenant آخر تؤثر على الرصيد
3. خطأ في حساب التكلفة

**الحل:**
```sql
-- استخدم Atomic UPDATE مع WHERE condition
UPDATE tenants 
SET balance = balance - $1
WHERE id = $2 
  AND balance >= $1  -- Prevent negative balance
RETURNING balance;
```

### Problem 3: Rate Limiting Affecting Other Tenants

**الأعراض:**
```
❌ Tenant-2 should not be affected by Tenant-1 rate limit
   Expected: No error
   Actual: Rate limit exceeded
```

**الأسباب المحتملة:**
1. Rate limit key لا يحتوي على tenant_id
2. Global rate limiting بدلاً من per-tenant
3. خطأ في Redis key format

**الحل:**
```go
// استخدم tenant-specific rate limit key
rateLimitKey := fmt.Sprintf("rate_limit:%d", tenantID)
```

### Problem 4: Redis Key Conflicts

**الأعراض:**
```
❌ Tenant-1: Buffer size = 150 (expected: 50)
```

**الأسباب المحتملة:**
1. Redis keys لا تحتوي على tenant_id
2. Key collision بين tenants
3. خطأ في Buffer implementation

**الحل:**
```go
// استخدم tenant-specific keys
bufferKey := fmt.Sprintf("messages:pending:%d", tenantID)
```

## Best Practices للعزل

### 1. Database Level

```go
// ✅ GOOD: Always include tenant_id in WHERE clause
db.Query(ctx, `
    SELECT * FROM messages 
    WHERE tenant_id = $1 AND id = $2
`, tenantID, messageID)

// ❌ BAD: Missing tenant_id check
db.Query(ctx, `
    SELECT * FROM messages 
    WHERE id = $1
`, messageID)
```

### 2. Redis Level

```go
// ✅ GOOD: Tenant-specific keys
key := fmt.Sprintf("messages:pending:%d", tenantID)

// ❌ BAD: Global keys
key := "messages:pending"
```

### 3. Application Level

```go
// ✅ GOOD: Verify tenant_id in all operations
func (s *Service) ProcessMessage(ctx context.Context, tenantID int64, messageID string) error {
    // Verify tenant owns this message
    var ownerID int64
    err := s.db.QueryRow(ctx, `
        SELECT tenant_id FROM messages WHERE id = $1
    `, messageID).Scan(&ownerID)
    
    if ownerID != tenantID {
        return ErrUnauthorized
    }
    
    // Process...
}

// ❌ BAD: No tenant verification
func (s *Service) ProcessMessage(ctx context.Context, messageID string) error {
    // Process without checking tenant...
}
```

### 4. Rate Limiting Level

```go
// ✅ GOOD: Per-tenant rate limiting
func (rl *RateLimiter) CheckLimit(ctx context.Context, tenantID int64) error {
    key := fmt.Sprintf("rate_limit:%d", tenantID)
    count, err := rl.redis.Incr(ctx, key).Result()
    if count == 1 {
        rl.redis.Expire(ctx, key, time.Minute)
    }
    if count > rl.limit {
        return ErrRateLimitExceeded
    }
    return nil
}

// ❌ BAD: Global rate limiting
func (rl *RateLimiter) CheckLimit(ctx context.Context) error {
    key := "rate_limit"
    // All tenants share same limit
}
```

## الخلاصة

اختبار Multi-Tenant Scenarios يضمن:

1. ✅ **Data Isolation**: كل tenant له بياناته الخاصة
2. ✅ **Balance Isolation**: عمليات مالية معزولة تماماً
3. ✅ **Rate Limiting Isolation**: حدود مستقلة لكل tenant
4. ✅ **Channel Isolation**: قنوات منفصلة لكل tenant
5. ✅ **Redis Isolation**: مفاتيح منفصلة لكل tenant
6. ✅ **No Cross-Tenant Effects**: عدم تأثير tenant على آخر

هذا الاختبار حاسم لضمان:
- الأمان المالي
- الخصوصية
- العدالة
- الامتثال للمعايير

## المراجع

- `.kiro/specs/message-lifecycle-rebuild/requirements.md` - المتطلبات الكاملة
- `.kiro/specs/message-lifecycle-rebuild/design.md` - التصميم التفصيلي
- `.kiro/specs/message-lifecycle-rebuild/tasks.md` - خطة التنفيذ
- `services/backend/internal/integration/multi_tenant_test.go` - كود الاختبار
