# Routing Engine

## نظرة عامة

Routing Engine هو المكون المسؤول عن اختيار أفضل قناة إرسال متاحة لكل رسالة. يضمن المحرك العزل التام بين الشركات (Tenants) ويطبق خوارزمية ذكية لتوزيع الحمل.

## المتطلبات المُنفذة

- **المتطلب 5.3**: فلترة القنوات بحالة ONLINE فقط
- **المتطلب 5.4**: ترتيب القنوات حسب الأولوية (Priority DESC) ثم الحمل الحالي (CurrentLoad ASC)
- **المتطلب 5.5**: تحديث حالة الرسالة إلى WAITING عند عدم توفر قنوات

## المعمارية

### المكونات الرئيسية

1. **RoutingEngine Interface**: واجهة تحدد عقد اختيار القناة
2. **Engine**: التطبيق الأساسي للمحرك
3. **Channel**: نموذج البيانات للقناة

### تدفق العمل

```
1. استقبال طلب اختيار قناة لـ tenant محدد
2. البحث في قاعدة البيانات عن القنوات المتاحة
   - فلترة بـ tenant_id (عزل تام)
   - فلترة بـ status = 'ONLINE'
3. حساب الحمل الحالي لكل قناة
4. ترتيب حسب Priority DESC, CurrentLoad ASC
5. إرجاع أفضل قناة متاحة
```

## الاستخدام

### إنشاء محرك التوجيه

```go
import (
    "database/sql"
    "github.com/al-hudhud/backend/internal/routing"
    "go.uber.org/zap"
)

// إنشاء المحرك
db, _ := sql.Open("postgres", dsn)
logger := zap.NewProduction()
engine := routing.NewEngine(db, logger)
```

### اختيار أفضل قناة

```go
import (
    "context"
    "github.com/google/uuid"
)

// اختيار قناة لـ tenant محدد
tenantID := uuid.MustParse("...")
ctx := context.Background()

channel, err := engine.SelectChannel(ctx, tenantID)
if err != nil {
    if err == routing.ErrNoChannelAvailable {
        // لا توجد قنوات متاحة - يجب الدخول في وضع الانتظار
        log.Warn("no channels available")
    } else {
        // خطأ آخر
        log.Error("failed to select channel", zap.Error(err))
    }
    return
}

// استخدام القناة المختارة
log.Info("channel selected",
    zap.String("channel_id", channel.ID.String()),
    zap.Int("priority", channel.Priority),
    zap.Int("current_load", channel.CurrentLoad),
)
```

### الحصول على جميع القنوات المتاحة

```go
// الحصول على جميع القنوات المتاحة لـ tenant
channels, err := engine.GetAvailableChannels(ctx, tenantID)
if err != nil {
    log.Error("failed to get available channels", zap.Error(err))
    return
}

log.Info("available channels",
    zap.Int("count", len(channels)),
)

for _, ch := range channels {
    log.Info("channel",
        zap.String("id", ch.ID.String()),
        zap.String("type", ch.Type),
        zap.Int("priority", ch.Priority),
        zap.Int("load", ch.CurrentLoad),
    )
}
```

## خوارزمية الاختيار

### 1. فلترة القنوات

```sql
WHERE tenant_id = $1 
  AND status = 'ONLINE'
```

- **tenant_id**: يضمن العزل التام بين الشركات
- **status = 'ONLINE'**: يضمن اختيار القنوات المتاحة فقط (المتطلب 5.3)

### 2. حساب الحمل الحالي

```sql
COALESCE(
    (SELECT COUNT(*) 
     FROM messages 
     WHERE channel_id = dispatch_channels.id 
       AND status IN ('PROCESSING', 'PENDING')
       AND tenant_id = $1
    ), 0
) as current_load
```

يتم حساب عدد الرسائل قيد المعالجة أو الانتظار لكل قناة.

### 3. الترتيب

```sql
ORDER BY priority DESC, current_load ASC
```

- **priority DESC**: القنوات ذات الأولوية الأعلى أولاً
- **current_load ASC**: عند تساوي الأولوية، القناة الأقل حملاً أولاً (المتطلب 5.4)

### 4. الاختيار

```sql
LIMIT 1
```

يتم اختيار أول قناة من النتائج المرتبة.

## العزل بين الشركات (Tenant Isolation)

### الضمانات

1. **جميع الاستعلامات تحتوي على `WHERE tenant_id = $1`**
   - لا يمكن لشركة الوصول لقنوات شركة أخرى
   
2. **حساب الحمل معزول**
   - يتم حساب الحمل فقط من رسائل نفس الشركة
   
3. **لا توجد استعلامات عامة**
   - جميع العمليات تتطلب tenant_id صريح

### مثال على العزل

```go
// Tenant A
channelA, _ := engine.SelectChannel(ctx, tenantA_ID)
// يحصل فقط على قنوات Tenant A

// Tenant B
channelB, _ := engine.SelectChannel(ctx, tenantB_ID)
// يحصل فقط على قنوات Tenant B

// لا يمكن لـ Tenant A رؤية قنوات Tenant B والعكس
```

## معالجة الأخطاء

### ErrNoChannelAvailable

يُرجع هذا الخطأ عندما لا توجد قنوات متاحة (ONLINE) للشركة.

**الإجراء المطلوب**: تحديث حالة الرسالة إلى WAITING باستخدام `SetMessageWaiting`

```go
channel, err := engine.SelectChannel(ctx, tenantID)
if err == routing.ErrNoChannelAvailable {
    // تحديث حالة الرسالة إلى WAITING
    err = engine.SetMessageWaiting(ctx, messageID, tenantID)
    if err != nil {
        log.Error("failed to set message to WAITING", zap.Error(err))
        return err
    }
    
    log.Info("message set to WAITING, will retry later")
    return nil
}
```

### SetMessageWaiting Method

يستخدم هذا الـ method لتحديث حالة الرسالة إلى WAITING عند عدم توفر قنوات.

**الميزات**:
- ✅ تحديث ذري (Atomic) للحالة
- ✅ عزل تام بين الشركات (Tenant Isolation)
- ✅ يعمل فقط على رسائل PENDING أو PROCESSING
- ✅ تسجيل مفصل للأحداث

**مثال الاستخدام**:

```go
// عند عدم توفر قنوات
if err == routing.ErrNoChannelAvailable {
    err = engine.SetMessageWaiting(ctx, messageID, tenantID)
    if err != nil {
        // معالجة الخطأ
        log.Error("failed to update message status", zap.Error(err))
    }
}
```

### أخطاء قاعدة البيانات

```go
channel, err := engine.SelectChannel(ctx, tenantID)
if err != nil && err != routing.ErrNoChannelAvailable {
    // خطأ في قاعدة البيانات
    // تسجيل الخطأ وإعادة المحاولة
}
```

## الأداء

### استخدام Prepared Statements

المحرك يستخدم `QueryRowContext` و `QueryContext` التي تدعم prepared statements تلقائياً في PostgreSQL.

### Indexes المطلوبة

```sql
-- Index للبحث السريع بـ tenant_id و status
CREATE INDEX idx_dispatch_channels_tenant_status 
ON dispatch_channels(tenant_id, status) 
WHERE status = 'ONLINE';

-- Index للترتيب
CREATE INDEX idx_dispatch_channels_priority 
ON dispatch_channels(priority DESC);

-- Index لحساب الحمل
CREATE INDEX idx_messages_channel_status 
ON messages(channel_id, status) 
WHERE status IN ('PROCESSING', 'PENDING');
```

### الأداء المتوقع

- **SelectChannel**: < 10ms (مع indexes)
- **GetAvailableChannels**: < 20ms (مع indexes)

## الاختبارات

### Unit Tests

```bash
cd services/backend/internal/routing
go test -v
```

### الاختبارات المتوفرة

1. **TestSelectChannel_OnlineOnly**: يتحقق من اختيار القنوات ONLINE فقط
2. **TestSelectChannel_PriorityOrdering**: يتحقق من الترتيب حسب الأولوية
3. **TestSelectChannel_LoadBalancing**: يتحقق من توزيع الحمل
4. **TestSelectChannel_NoChannelAvailable**: يتحقق من معالجة حالة عدم وجود قنوات
5. **TestSelectChannel_TenantIsolation**: يتحقق من العزل بين الشركات
6. **TestGetAvailableChannels_***: اختبارات شاملة لجميع السيناريوهات

### متطلبات الاختبار

- PostgreSQL test database
- Connection string: `postgres://postgres:postgres@localhost:5432/sms_gateway_test?sslmode=disable`

## التكامل مع المكونات الأخرى

### Worker Pool

```go
// في Worker
channel, err := routingEngine.SelectChannel(ctx, message.TenantID)
if err != nil {
    if err == routing.ErrNoChannelAvailable {
        // تحديث حالة الرسالة إلى WAITING
        return
    }
    return err
}

// إرسال الرسالة عبر القناة المختارة
err = websocketHub.SendMessage(ctx, channel.ID, message)
```

### Status Report Handler

```go
// عند تغيير حالة القناة
if channel.Status == "OFFLINE" {
    // إبطال الـ cache
    routingEngine.InvalidateCache(ctx, tenantID)
}
```

## الخطوات القادمة

### المرحلة التالية (Task 5.2)

- تطبيق Redis Caching للقنوات
- TTL = 5 دقائق
- البحث في Cache أولاً، ثم Database عند Cache miss
- تحقيق 99%+ Cache Hit Rate

### التحسينات المستقبلية

1. **Health Checks**: فحص صحة القنوات دورياً
2. **Auto-scaling**: زيادة/تقليل القنوات تلقائياً
3. **Predictive Routing**: اختيار القناة بناءً على التنبؤ بالحمل
4. **Geographic Routing**: اختيار القناة بناءً على موقع المستلم

## المراجع

- [وثيقة المتطلبات](.kiro/specs/message-lifecycle-rebuild/requirements.md)
- [وثيقة التصميم](.kiro/specs/message-lifecycle-rebuild/design.md)
- [قائمة المهام](.kiro/specs/message-lifecycle-rebuild/tasks.md)
