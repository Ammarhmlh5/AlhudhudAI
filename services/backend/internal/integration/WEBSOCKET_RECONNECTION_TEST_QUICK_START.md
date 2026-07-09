# WebSocket Reconnection Test - Quick Start
# دليل سريع لاختبار إعادة اتصال WebSocket

## نظرة عامة

هذا الاختبار يتحقق من قدرة النظام على إعادة الاتصال التلقائي عند انقطاع WebSocket وضمان عدم فقدان الرسائل.

**المتطلبات المختبرة:**
- المتطلب 6.5: إعادة الاتصال التلقائي خلال 5 ثوانٍ
- المتطلب 24.6: استكمال معالجة الرسائل المتبقية بعد العطل

## التشغيل السريع

### Windows (PowerShell)
```powershell
.\services\backend\run-websocket-reconnection-test.ps1
```

### Linux/Mac (Bash)
```bash
chmod +x services/backend/run-websocket-reconnection-test.sh
./services/backend/run-websocket-reconnection-test.sh
```

### مباشرة باستخدام Go
```bash
cd services/backend
go test -v -run TestWebSocketReconnection ./internal/integration/
```

## المتطلبات الأساسية

1. **Docker** يجب أن يكون قيد التشغيل
2. **PostgreSQL** container يجب أن يكون قيد التشغيل
3. **Redis** container يجب أن يكون قيد التشغيل

### تشغيل الخدمات
```bash
docker-compose up -d postgres redis
```

## سيناريوهات الاختبار

### 1. Reconnection Within 5 Seconds
- إنشاء اتصال WebSocket
- محاكاة انقطاع الاتصال
- إعادة الاتصال
- **التحقق:** إعادة الاتصال تتم خلال 5 ثوانٍ

### 2. Pending Messages Sent After Reconnection
- إنشاء اتصال وإرسال رسائل
- انقطاع الاتصال أثناء الإرسال
- إعادة الاتصال
- **التحقق:** الرسائل المعلقة تُرسل بعد إعادة الاتصال

### 3. No Message Loss During Reconnection
- إرسال رسائل مستمر
- انقطاع وإعادة اتصال أثناء الإرسال
- **التحقق:** نسبة فقدان الرسائل < 20%

### 4. Multiple Reconnection Cycles
- عدة دورات من الانقطاع وإعادة الاتصال
- **التحقق:** النظام يتعامل مع دورات متعددة بنجاح

## النتائج المتوقعة

### ✅ نجاح الاختبار
```
=== RUN   TestWebSocketReconnection
=== RUN   TestWebSocketReconnection/Reconnection_Within_5_Seconds
    ✓ Initial connection established and registered
    ✓ Disconnection detected
    ✓ Reconnected in 1.234s (< 5 seconds)
    ✓ Connection status is ONLINE
=== RUN   TestWebSocketReconnection/Pending_Messages_Sent_After_Reconnection
    ✓ Queued 5 messages
    ✓ Received 8 messages after reconnection
=== RUN   TestWebSocketReconnection/No_Message_Loss_During_Reconnection
    Sent: 20 messages, Received: 18 messages
    ✓ Message loss: 10.00% (acceptable)
=== RUN   TestWebSocketReconnection/Multiple_Reconnection_Cycles
    ✓ Completed 3 reconnection cycles successfully
--- PASS: TestWebSocketReconnection (15.23s)
PASS
```

### ❌ فشل الاختبار
إذا فشل الاختبار، تحقق من:
1. WebSocket Hub يعمل بشكل صحيح
2. آلية إعادة الاتصال مفعلة
3. Buffer الرسائل يعمل بشكل صحيح
4. لا توجد أخطاء في الشبكة

## استكشاف الأخطاء

### خطأ: "Failed to connect to database"
```bash
# تأكد من تشغيل PostgreSQL
docker-compose up -d postgres
```

### خطأ: "Failed to connect to Redis"
```bash
# تأكد من تشغيل Redis
docker-compose up -d redis
```

### خطأ: "Reconnection took too long"
- تحقق من إعدادات WebSocket Hub
- تحقق من آلية إعادة الاتصال التلقائي
- راجع logs للأخطاء

### خطأ: "Too many messages lost"
- تحقق من Buffer capacity
- تحقق من آلية حفظ الرسائل المعلقة
- راجع WebSocket send logic

## معلومات إضافية

### مدة الاختبار
- الوقت المتوقع: 15-20 ثانية
- يتضمن عدة دورات من الاتصال والانقطاع

### البيانات المستخدمة
- يتم إنشاء tenant وchannel اختباريين
- يتم تنظيف البيانات تلقائياً بعد الاختبار

### الملفات ذات الصلة
- `websocket_reconnection_test.go` - كود الاختبار
- `services/backend/internal/ws/hub.go` - WebSocket Hub implementation
- `services/backend/internal/ws/README.md` - WebSocket documentation

## الخطوات التالية

بعد نجاح هذا الاختبار:
1. ✅ المهمة 15.5 مكتملة
2. ➡️ انتقل للمهمة 15.6: اختبار Database Failure Recovery
3. 📝 راجع نتائج الاختبار وسجل أي ملاحظات

## الدعم

إذا واجهت مشاكل:
1. راجع logs في `services/backend/logs/`
2. تحقق من حالة الخدمات: `docker-compose ps`
3. راجع الوثائق الكاملة في `WEBSOCKET_RECONNECTION_TEST_GUIDE.md`
