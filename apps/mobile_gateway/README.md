# 📱 Mobile Gateway - بوابة الهدهد للرسائل

تطبيق أندرويد لإرسال الرسائل النصية (SMS) عبر شبكة موزعة من الأجهزة.

---

## 🎯 الميزات

### الميزات الأساسية
- ✅ استقبال الرسائل من الخادم عبر WebSocket
- ✅ إرسال الرسائل عبر SMS
- ✅ تتبع حالة كل رسالة
- ✅ إعادة المحاولة التلقائية (حتى 3 مرات)
- ✅ إرسال تقارير الحالة للخادم

### الميزات المتقدمة
- ✅ نظام Heartbeat للحفاظ على الاتصال
- ✅ مزامنة تلقائية للرسائل غير المرسلة
- ✅ إعادة اتصال تلقائي عند الانقطاع
- ✅ خدمة خلفية تعمل حتى عند إغلاق التطبيق
- ✅ إعادة تشغيل تلقائي عند إعادة تشغيل الجهاز

### ميزات المراقبة
- ✅ Dashboard مع إحصائيات حية
- ✅ رسوم بيانية للنشاط (آخر 24 ساعة)
- ✅ شاشة Logs لعرض السجلات
- ✅ شاشة Failed Messages للرسائل الفاشلة
- ✅ معلومات شرائح SIM

---

## 🛠️ التقنيات المستخدمة

- **React Native 0.76.0** - إطار العمل الأساسي
- **TypeScript** - لغة البرمجة
- **Zustand** - إدارة الحالة
- **SQLite** - قاعدة البيانات المحلية
- **WebSocket** - الاتصال بالخادم
- **Kotlin** - Native Modules

---

## 📋 المتطلبات

- Node.js >= 18
- Java JDK 17 أو 21
- Android Studio
- Android SDK (API Level 24-35)
- جهاز أندرويد أو محاكي

---

## 🚀 التثبيت

### 1. تثبيت Dependencies
```bash
npm install
```

### 2. البناء والتشغيل
```bash
# للأندرويد
npm run android

# للـ iOS (إذا كنت على Mac)
npm run ios
```

---

## ⚙️ الإعداد

### 1. إدخال بيانات الخادم
1. افتح التطبيق
2. اذهب إلى Settings (⚙️)
3. أدخل:
   - رابط الخادم: `http://YOUR_SERVER_IP:8082`
   - مفتاح الوصول: `YOUR_API_KEY`
4. احفظ الإعدادات

### 2. منح الصلاحيات
- إرسال الرسائل (SMS)
- قراءة حالة الهاتف
- الإشعارات
- تجاهل تحسين البطارية

### 3. تشغيل البوابة
1. ارجع للـ Dashboard
2. اضغط "تشغيل البوابة"

---

## 📁 هيكل المشروع

```
apps/mobile_gateway/
├── android/                    # ملفات Android Native
│   └── app/src/main/java/
│       └── com/mobile_gateway/
│           ├── GatewayService.kt      # الخدمة الخلفية
│           ├── SmsModule.kt           # وحدة إرسال SMS
│           └── BootReceiver.kt        # إعادة التشغيل التلقائي
├── src/
│   ├── screens/               # الشاشات
│   │   ├── DashboardScreen.tsx
│   │   ├── SettingsScreen.tsx
│   │   ├── LogsScreen.tsx
│   │   ├── FailedMessagesScreen.tsx
│   │   └── SupportScreen.tsx
│   └── services/              # الخدمات
│       ├── WebSocketService.ts       # إدارة WebSocket
│       ├── DatabaseService.ts        # قاعدة البيانات
│       ├── SmsQueueManager.ts        # معالجة الطابور
│       ├── NativeSms.ts              # واجهة SMS
│       └── BackgroundServiceControl.ts
├── App.tsx                    # نقطة الدخول
└── package.json
```

---

## 🔄 دورة حياة الرسالة

```
1. استقبال من الخادم (WebSocket)
   ↓
2. حفظ في قاعدة البيانات (SQLite)
   ↓
3. إرسال تقرير BUFFERED للخادم
   ↓
4. إضافة للطابور (Queue)
   ↓
5. إرسال عبر SMS (Native Module)
   ↓
6. تحديث الحالة (SENT/FAILED)
   ↓
7. إرسال تقرير نهائي للخادم
```

---

## 🧪 الاختبار

### اختبار سريع
```bash
# 1. ابنِ التطبيق
npm run android

# 2. أرسل رسالة من الخادم
POST http://YOUR_SERVER_IP:8082/api/messages
{
  "phone_number": "+966XXXXXXXXX",
  "content": "رسالة اختبار",
  "device_id": "YOUR_DEVICE_UUID"
}

# 3. راقب التطبيق
# - عداد "قيد الانتظار" يزيد
# - بعد 3 ثوانٍ، عداد "تم الإرسال" يزيد
# - الرسالة تصل للجهاز المستقبل
```

---

## 🚨 حل المشاكل

### التطبيق لا يبني
```bash
rm -rf node_modules
rm -rf android/build
rm -rf android/app/build
npm install
npm run android
```

### WebSocket لا يتصل
- تحقق من API URL (يجب أن يبدأ بـ http:// أو https://)
- تحقق من API Key
- تحقق من أن الخادم يعمل

### الرسائل لا تُرسل
- تحقق من صلاحية SEND_SMS
- تحقق من أن الخدمة الخلفية تعمل
- راجع السجلات للأخطاء

---

## 📊 الإحصائيات

- **عدد الشاشات:** 6
- **عدد الخدمات:** 6
- **عدد الملفات Native:** 9
- **Dependencies:** 17 مكتبة

---

## 🔐 الأمان

- API Key authentication
- Device UUID للتعريف
- نظام حظر الأجهزة
- Idempotency keys لمنع التكرار

---

## 📝 الإصدار

**النسخة الحالية:** 1.0.0-PRO  
**آخر تحديث:** 23 فبراير 2026

---

## 📞 الدعم

للمساعدة أو الإبلاغ عن مشاكل، راجع:
- `تقرير_جاهزية_التطبيق_للاختبار.md`
- `دليل_الاختبار_السريع.md`
- `خارطة_الطريق_لتثبيت_التطبيق.md`

---

## 📄 الترخيص

جميع الحقوق محفوظة © 2026
