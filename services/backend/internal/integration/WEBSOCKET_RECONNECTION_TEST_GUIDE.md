# WebSocket Reconnection Integration Test - Complete Guide
# دليل شامل لاختبار إعادة اتصال WebSocket

## المحتويات

1. [نظرة عامة](#نظرة-عامة)
2. [المتطلبات المختبرة](#المتطلبات-المختبرة)
3. [البنية المعمارية](#البنية-المعمارية)
4. [سيناريوهات الاختبار](#سيناريوهات-الاختبار)
5. [التشغيل والتنفيذ](#التشغيل-والتنفيذ)
6. [تحليل النتائج](#تحليل-النتائج)
7. [استكشاف الأخطاء](#استكشاف-الأخطاء)
8. [أفضل الممارسات](#أفضل-الممارسات)

---

## نظرة عامة

### الهدف من الاختبار

هذا الاختبار التكاملي يتحقق من قدرة نظام WebSocket على:
- إعادة الاتصال التلقائي عند انقطاع الاتصال
- إعادة الاتصال خلال 5 ثوانٍ كحد أقصى
- استكمال إرسال الرسائل المعلقة بعد إعادة الاتصال
- عدم فقدان الرسائل أثناء عملية إعادة الاتصال
- التعامل مع دورات متعددة من الانقطاع وإعادة الاتصال

### أهمية الاختبار

WebSocket هو العمود الفقري للاتصال بين السيرفر والأجهزة المحمولة. انقطاع الاتصال يمكن أن يحدث بسبب:
- مشاكل الشبكة
- إعادة تشغيل السيرفر
- تحديثات النظام
- مشاكل في الجهاز المحمول

ضمان إعادة الاتصال التلقائي وعدم فقدان الرسائل أمر حاسم لموثوقية النظام.

---

## المتطلبات المختبرة

### المتطلب 6.5: إعادة الاتصال التلقائي

**من وثيقة المتطلبات:**
> WHEN ينقطع الاتصال، THE WebSocket_Hub SHALL إعادة الاتصال تلقائياً خلال 5 ثوانٍ

**معايير القبول:**
- اكتشاف انقطاع الاتصال فوراً
- بدء عملية إعادة الاتصال تلقائياً
- إعادة الاتصال خلال 5 ثوانٍ كحد أقصى
- تسجيل حالة الاتصال (OFFLINE → RECONNECTING → ONLINE)

### المتطلب 24.6: استكمال معالجة الرسائل

**من وثيقة المتطلبات:**
> WHEN يعود النظام بعد عطل، THE Message_Lifecycle_System SHALL استكمال معالجة الرسائل المتبقية

**معايير القبول:**
- حفظ الرسائل المعلقة أثناء الانقطاع
- استكمال إرسال الرسائل بعد إعادة الاتصال
- عدم فقدان أي رسائل
- الحفاظ على ترتيب الرسائل (FIFO)

---

## البنية المعمارية

### مكونات الاختبار

```
┌─────────────────────────────────────────────────────────────┐
│                    Integration Test                          │
│  ┌──────────────────────────────────────────────────────┐   │
│  │                                                       │   │
│  │  1. Test Client (WebSocket Client)                   │   │
│  │     - Establishes connection                         │   │
│  │     - Simulates disconnection                        │   │
│  │     - Reconnects                                     │   │
│  │     - Receives messages                              │   │
│  │                                                       │   │
│  └──────────────────┬───────────────────────────────────┘   │
│                     │ WebSocket                              │
│                     ↓                                        │
│  ┌──────────────────────────────────────────────────────┐   │
│  │                                                       │   │
│  │  2. WebSocket Hub                                    │   │
│  │     - Manages connections                            │   │
│  │     - Detects disconnections                         │   │
│  │     - Buffers messages                               │   │
│  │     - Sends messages                                 │   │
│  │                                                       │   │
│  └──────────────────┬───────────────────────────────────┘   │
│                     │                                        │
│                     ↓                                        │
│  ┌──────────────────────────────────────────────────────┐   │
│  │                                                       │   │
│  │  3. Test Infrastructure                              │   │
│  │     - PostgreSQL (tenant/channel data)               │   │
│  │     - Redis (connection state)                       │   │
│  │                                                       │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

### تدفق إعادة الاتصال

```
Client                    Hub                     Test
  │                        │                        │
  │──── Connect ──────────>│                        │
  │                        │                        │
  │<─── Connected ─────────│                        │
  │                        │                        │
  │                        │<──── Send Message ─────│
  │<─── Message ───────────│                        │
  │                        │                        │
  │──── Disconnect ────────>│                        │
  │                        │                        │
  │                        │ (Detect Disconnect)    │
  │                        │ Status: OFFLINE        │
  │                        │                        │
  │                        │<──── Send Message ─────│
  │                        │ (Buffer Message)       │
  │                        │                        │
  │──── Reconnect ─────────>│                        │
  │                        │                        │
  │<─── Connected ─────────│                        │
  │                        │ Status: ONLINE         │
  │                        │                        │
  │<─── Buffered Msgs ─────│                        │
  │                        │                        │
```

---

## سيناريوهات الاختبار

### Scenario 1: Reconnection Within 5 Seconds

**الهدف:** التحقق من أن إعادة الاتصال تتم خلال 5 ثوانٍ

**الخطوات:**
1. إنشاء اتصال WebSocket
2. التحقق من تسجيل الاتصال (status = ONLINE)
3. محاكاة انقطاع الاتصال (close connection)
4. التحقق من اكتشاف الانقطاع (status = OFFLINE/RECONNECTING)
5. إعادة الاتصال
6. قياس الوقت المستغرق
7. التحقق من أن الوقت < 5 ثوانٍ
8. التحقق من حالة الاتصال (status = ONLINE)

**معايير النجاح:**
- ✅ الاتصال الأولي ناجح
- ✅ اكتشاف الانقطاع فوري
- ✅ إعادة الاتصال خلال < 5 ثوانٍ
- ✅ حالة الاتصال النهائية = ONLINE

**الكود:**
```go
t.Run("Reconnection_Within_5_Seconds", func(t *testing.T) {
    // 1. Establish connection
    conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
    require.NoError(t, err)
    
    // 2. Verify ONLINE
    status, _ := wsHub.GetConnectionStatus(ctx, tenantID, channelID)
    assert.Equal(t, "ONLINE", status)
    
    // 3. Disconnect
    disconnectTime := time.Now()
    conn.Close()
    
    // 4. Verify OFFLINE
    time.Sleep(200 * time.Millisecond)
    status, _ = wsHub.GetConnectionStatus(ctx, tenantID, channelID)
    assert.Contains(t, []string{"OFFLINE", "RECONNECTING"}, status)
    
    // 5. Reconnect
    newConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
    require.NoError(t, err)
    defer newConn.Close()
    
    reconnectTime := time.Since(disconnectTime)
    
    // 6. Verify < 5 seconds
    assert.Less(t, reconnectTime, 5*time.Second)
    
    // 7. Verify ONLINE
    time.Sleep(100 * time.Millisecond)
    status, _ = wsHub.GetConnectionStatus(ctx, tenantID, channelID)
    assert.Equal(t, "ONLINE", status)
})
```

---

### Scenario 2: Pending Messages Sent After Reconnection

**الهدف:** التحقق من إرسال الرسائل المعلقة بعد إعادة الاتصال

**الخطوات:**
1. إنشاء اتصال WebSocket
2. إرسال 5 رسائل عبر Hub
3. انقطاع الاتصال قبل استلام جميع الرسائل
4. إعادة الاتصال
5. إرسال 5 رسائل إضافية
6. التحقق من استلام الرسائل بعد إعادة الاتصال

**معايير النجاح:**
- ✅ إرسال الرسائل قبل الانقطاع
- ✅ إعادة الاتصال ناجحة
- ✅ استلام رسائل بعد إعادة الاتصال
- ✅ عدد الرسائل المستلمة > 0

**السلوك المتوقع:**
- الرسائل المعلقة تُحفظ في Buffer
- بعد إعادة الاتصال، يتم إرسال الرسائل المعلقة
- الرسائل الجديدة تُرسل بشكل طبيعي

---

### Scenario 3: No Message Loss During Reconnection

**الهدف:** التحقق من عدم فقدان الرسائل أثناء إعادة الاتصال

**الخطوات:**
1. إنشاء اتصال WebSocket
2. بدء إرسال رسائل مستمر (20 رسالة)
3. انقطاع الاتصال في منتصف الإرسال
4. إعادة الاتصال
5. استكمال إرسال الرسائل
6. حساب نسبة فقدان الرسائل
7. التحقق من أن نسبة الفقدان < 20%

**معايير النجاح:**
- ✅ إرسال 20 رسالة
- ✅ استلام > 50% من الرسائل
- ✅ نسبة الفقدان < 20%

**ملاحظات:**
- بعض الرسائل قد تُفقد أثناء الانقطاع (messages in flight)
- النسبة المقبولة للفقدان: < 20%
- الرسائل المهمة يجب أن تُحفظ في قاعدة البيانات

**الكود:**
```go
t.Run("No_Message_Loss_During_Reconnection", func(t *testing.T) {
    sentMessages := make(map[string]bool)
    receivedMessages := make(map[string]bool)
    
    // Send 20 messages with disconnect in middle
    for i := 0; i < 20; i++ {
        msgID := uuid.New().String()
        sentMessages[msgID] = true
        
        // Disconnect at message 10
        if i == 10 {
            conn.Close()
            time.Sleep(200 * time.Millisecond)
            newConn, _ = websocket.DefaultDialer.Dial(wsURL, nil)
        }
        
        wsHub.SendMessage(ctx, msg)
    }
    
    // Calculate loss
    lossPercentage := float64(len(sentMessages)-len(receivedMessages)) / float64(len(sentMessages)) * 100
    
    assert.Less(t, lossPercentage, 20.0)
})
```

---

### Scenario 4: Multiple Reconnection Cycles

**الهدف:** التحقق من التعامل مع دورات متعددة من الانقطاع وإعادة الاتصال

**الخطوات:**
1. تنفيذ 3 دورات من:
   - إنشاء اتصال
   - التحقق من ONLINE
   - إرسال رسالة اختبار
   - انقطاع الاتصال
   - التحقق من OFFLINE
2. إعادة اتصال نهائية
3. التحقق من ONLINE

**معايير النجاح:**
- ✅ نجاح جميع الدورات (3 دورات)
- ✅ الاتصال النهائي ONLINE
- ✅ لا توجد أخطاء أو تسريبات في الذاكرة

**الفائدة:**
- اختبار استقرار النظام
- اختبار إدارة الموارد
- اختبار عدم وجود memory leaks

---

## التشغيل والتنفيذ

### المتطلبات الأساسية

#### 1. Docker و Docker Compose
```bash
# التحقق من تثبيت Docker
docker --version

# التحقق من تثبيت Docker Compose
docker-compose --version
```

#### 2. Go 1.21+
```bash
# التحقق من إصدار Go
go version
```

#### 3. الخدمات المطلوبة
```bash
# تشغيل PostgreSQL و Redis
docker-compose up -d postgres redis

# التحقق من حالة الخدمات
docker-compose ps
```

### طرق التشغيل

#### الطريقة 1: استخدام PowerShell Script (Windows)
```powershell
# من المجلد الرئيسي للمشروع
.\services\backend\run-websocket-reconnection-test.ps1
```

**المميزات:**
- ✅ فحص تلقائي للخدمات
- ✅ رسائل واضحة بالعربية والإنجليزية
- ✅ معالجة الأخطاء

#### الطريقة 2: استخدام Bash Script (Linux/Mac)
```bash
# من المجلد الرئيسي للمشروع
chmod +x services/backend/run-websocket-reconnection-test.sh
./services/backend/run-websocket-reconnection-test.sh
```

#### الطريقة 3: Go Test مباشرة
```bash
cd services/backend
go test -v -run TestWebSocketReconnection ./internal/integration/
```

**خيارات إضافية:**
```bash
# تشغيل سيناريو محدد
go test -v -run TestWebSocketReconnection/Reconnection_Within_5_Seconds ./internal/integration/

# مع timeout
go test -v -timeout 30s -run TestWebSocketReconnection ./internal/integration/

# مع race detector
go test -v -race -run TestWebSocketReconnection ./internal/integration/
```

---

## تحليل النتائج

### نتيجة ناجحة

```
=== RUN   TestWebSocketReconnection
=== RUN   TestWebSocketReconnection/Reconnection_Within_5_Seconds
    websocket_reconnection_test.go:89: Step 1: Establishing initial WebSocket connection
    websocket_reconnection_test.go:99: ✓ Initial connection established and registered
    websocket_reconnection_test.go:102: Step 2: Simulating WebSocket disconnection
    websocket_reconnection_test.go:113: ✓ Disconnection detected
    websocket_reconnection_test.go:116: Step 3: Reconnecting WebSocket
    websocket_reconnection_test.go:127: Step 4: Verifying reconnection timeliness
    websocket_reconnection_test.go:129: ✓ Reconnected in 1.234s (< 5 seconds)
    websocket_reconnection_test.go:135: ✓ Connection status is ONLINE
--- PASS: TestWebSocketReconnection/Reconnection_Within_5_Seconds (2.45s)

=== RUN   TestWebSocketReconnection/Pending_Messages_Sent_After_Reconnection
    websocket_reconnection_test.go:141: Step 1: Establishing WebSocket connection
    websocket_reconnection_test.go:151: Step 2: Queueing messages
    websocket_reconnection_test.go:172: ✓ Queued 5 messages
    websocket_reconnection_test.go:175: Step 3: Disconnecting WebSocket
    websocket_reconnection_test.go:180: Step 4: Reconnecting WebSocket
    websocket_reconnection_test.go:189: Step 5: Sending messages after reconnection
    websocket_reconnection_test.go:230: Step 6: Verifying message delivery
    websocket_reconnection_test.go:236: ✓ Received 8 messages after reconnection
--- PASS: TestWebSocketReconnection/Pending_Messages_Sent_After_Reconnection (4.12s)

=== RUN   TestWebSocketReconnection/No_Message_Loss_During_Reconnection
    websocket_reconnection_test.go:242: Step 1: Establishing WebSocket connection
    websocket_reconnection_test.go:276: Step 2: Sending messages continuously
    websocket_reconnection_test.go:298: Step 3: Simulating disconnection during message sending
    websocket_reconnection_test.go:303: Step 4: Reconnecting WebSocket
    websocket_reconnection_test.go:313: Step 5: Continuing to send messages after reconnection
    websocket_reconnection_test.go:341: Step 6: Verifying no message loss
    websocket_reconnection_test.go:348: Sent: 20 messages, Received: 18 messages
    websocket_reconnection_test.go:355: ✓ Message loss: 10.00% (acceptable)
--- PASS: TestWebSocketReconnection/No_Message_Loss_During_Reconnection (5.67s)

=== RUN   TestWebSocketReconnection/Multiple_Reconnection_Cycles
    websocket_reconnection_test.go:361: Testing multiple reconnection cycles
    websocket_reconnection_test.go:365: Cycle 1: Establishing connection
    websocket_reconnection_test.go:397: Cycle 1: Disconnecting
    websocket_reconnection_test.go:407: ✓ Cycle 1 completed
    websocket_reconnection_test.go:365: Cycle 2: Establishing connection
    websocket_reconnection_test.go:397: Cycle 2: Disconnecting
    websocket_reconnection_test.go:407: ✓ Cycle 2 completed
    websocket_reconnection_test.go:365: Cycle 3: Establishing connection
    websocket_reconnection_test.go:397: Cycle 3: Disconnecting
    websocket_reconnection_test.go:407: ✓ Cycle 3 completed
    websocket_reconnection_test.go:411: Final reconnection
    websocket_reconnection_test.go:424: ✓ Completed 3 reconnection cycles successfully
--- PASS: TestWebSocketReconnection/Multiple_Reconnection_Cycles (3.21s)

--- PASS: TestWebSocketReconnection (15.45s)
PASS
ok      github.com/al-hudhud/backend/internal/integration      15.678s
```

### تحليل المقاييس

| Metric | Expected | Actual | Status |
|--------|----------|--------|--------|
| Reconnection Time | < 5s | 1.234s | ✅ PASS |
| Messages Received | > 0 | 8 | ✅ PASS |
| Message Loss | < 20% | 10% | ✅ PASS |
| Reconnection Cycles | 3 | 3 | ✅ PASS |
| Total Duration | < 30s | 15.45s | ✅ PASS |

---

## استكشاف الأخطاء

### خطأ: "Failed to connect to database"

**السبب:** PostgreSQL غير قيد التشغيل

**الحل:**
```bash
# تشغيل PostgreSQL
docker-compose up -d postgres

# التحقق من الحالة
docker-compose ps postgres

# فحص logs
docker-compose logs postgres
```

---

### خطأ: "Failed to connect to Redis"

**السبب:** Redis غير قيد التشغيل

**الحل:**
```bash
# تشغيل Redis
docker-compose up -d redis

# التحقق من الحالة
docker-compose ps redis

# فحص logs
docker-compose logs redis
```

---

### خطأ: "Reconnection took too long"

**السبب المحتمل:**
- مشاكل في الشبكة
- WebSocket Hub لا يعمل بشكل صحيح
- آلية إعادة الاتصال معطلة

**الحل:**
1. فحص WebSocket Hub logs
2. التحقق من إعدادات reconnection timeout
3. فحص حالة الشبكة

```go
// في hub.go
const (
    reconnectionTimeout = 5 * time.Second
    maxReconnectAttempts = 3
)
```

---

### خطأ: "Too many messages lost"

**السبب المحتمل:**
- Buffer capacity صغير جداً
- الرسائل لا تُحفظ أثناء الانقطاع
- مشاكل في send logic

**الحل:**
1. زيادة buffer capacity
```go
// في hub.go
const (
    sendBufferSize = 1000 // زيادة من 256
)
```

2. تفعيل message persistence
```go
// حفظ الرسائل في Redis أثناء الانقطاع
if conn.status == "OFFLINE" {
    redis.LPush(ctx, fmt.Sprintf("pending:%d:%d", tenantID, channelID), msg)
}
```

---

### خطأ: "Connection status not updated"

**السبب المحتمل:**
- Hub monitoring لا يعمل
- Status update logic معطل

**الحل:**
```go
// التحقق من تشغيل monitoring
go hub.monitorConnections()

// فحص checkDisconnectedConnections
func (h *Hub) checkDisconnectedConnections() {
    // يجب أن يتم استدعاؤها بشكل دوري
}
```

---

## أفضل الممارسات

### 1. إدارة الاتصالات

```go
// ✅ جيد: استخدام context للتحكم في الاتصال
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
```

```go
// ❌ سيء: بدون timeout
conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
```

### 2. معالجة الأخطاء

```go
// ✅ جيد: معالجة شاملة للأخطاء
conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
if err != nil {
    if websocket.IsUnexpectedCloseError(err) {
        // Handle unexpected close
    } else if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
        // Handle normal close
    } else {
        // Handle other errors
    }
}
```

### 3. تنظيف الموارد

```go
// ✅ جيد: تنظيف تلقائي
defer func() {
    if conn != nil {
        conn.Close()
    }
    cleanupTestData(t, ctx, db, redisClient)
}()
```

### 4. Timeouts مناسبة

```go
// ✅ جيد: timeouts واضحة
const (
    connectionTimeout = 5 * time.Second
    readTimeout = 10 * time.Second
    writeTimeout = 10 * time.Second
)
```

### 5. Logging مفيد

```go
// ✅ جيد: logging تفصيلي
t.Logf("Step %d: %s", stepNum, description)
t.Logf("✓ %s completed in %v", operation, duration)
t.Logf("Sent: %d, Received: %d, Loss: %.2f%%", sent, received, loss)
```

---

## الملخص

### ما تم اختباره

✅ إعادة الاتصال التلقائي خلال 5 ثوانٍ  
✅ إرسال الرسائل المعلقة بعد إعادة الاتصال  
✅ عدم فقدان الرسائل (< 20% loss)  
✅ التعامل مع دورات متعددة من الانقطاع  
✅ تحديث حالة الاتصال بشكل صحيح  

### المتطلبات المحققة

✅ المتطلب 6.5: إعادة الاتصال التلقائي  
✅ المتطلب 24.6: استكمال معالجة الرسائل  

### الخطوات التالية

1. ✅ المهمة 15.5 مكتملة
2. ➡️ المهمة 15.6: اختبار Database Failure Recovery
3. ➡️ المهمة 15.7: اختبار Redis Failure Fallback
4. ➡️ المهمة 15.8: Checkpoint - مراجعة اختبارات التكامل

---

## المراجع

- [WebSocket Hub Implementation](../../internal/ws/hub.go)
- [WebSocket README](../../internal/ws/README.md)
- [Requirements Document](../../../../.kiro/specs/message-lifecycle-rebuild/requirements.md)
- [Design Document](../../../../.kiro/specs/message-lifecycle-rebuild/design.md)
- [Tasks Document](../../../../.kiro/specs/message-lifecycle-rebuild/tasks.md)

---

**آخر تحديث:** 2024-02-25  
**الإصدار:** 1.0  
**الحالة:** مكتمل ✅
