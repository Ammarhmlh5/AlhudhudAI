# سكربتات تشغيل مشروع ALHudhudAI

## نظرة عامة

هذا المجلد يحتوي على سكربتات تشغيل وإيقاف جميع مكونات مشروع ALHudhudAI.

## المكونات المدعومة

### 1. البنية التحتية (Infrastructure)
- Docker Containers (PostgreSQL, Redis, إلخ)

### 2. الخدمات الخلفية (Backend Services)
- **Backend Service**: `services/backend`
- **Company API Service**: `services/company-api`

### 3. واجهات الويب (Web Interfaces)
- **Admin Web**: `apps/admin-web` - المنفذ 5174
- **Client Web**: `apps/client-web` - المنفذ 5173
- **Company Portal**: `apps/company-portal` - المنفذ 5175

### 4. تطبيق الهاتف (Mobile Application)
- **AlHudhud Mobile**: `C:\ALHudhud\ALHudhud\ALHudhudMobile` - Expo (SDK 54)

---

## ملفات التشغيل

### ملفات Windows Batch (.bat)

#### `0-تشغيل_الكل.bat`
**تشغيل النظام الكامل بضغطة واحدة**
- يشغل Docker
- يشغل الخدمات الخلفية (Backend + Company API)
- يشغل جميع واجهات الويب (3 واجهات)
- يشغل Metro Bundler لتطبيق الهاتف

#### `1-إيقاف_الكل.bat`
**إيقاف جميع الخدمات**
- يوقف Docker containers
- يوقف جميع عمليات Go (Backend services)
- يوقف جميع عمليات Node.js (Web interfaces + Mobile)

#### `2-تشغيل_السيرفر.bat`
**تشغيل الخدمات الخلفية فقط**
- Docker + Backend Service + Company API

#### `3-تشغيل_الواجهات.bat`
**تشغيل واجهات الويب وتطبيق الهاتف فقط**
- Admin Web + Client Web + Company Portal + Mobile Gateway

---

### ملفات PowerShell (.ps1)

#### `Start_All.ps1`
سكربت شامل لتشغيل النظام الكامل بالترتيب:
1. Docker Infrastructure
2. Backend Services
3. Web Interfaces
4. Mobile Gateway

#### `Start_Backend.ps1`
تشغيل الخدمات الخلفية فقط

#### `Start_Web_Interfaces.ps1`
تشغيل واجهات الويب وتطبيق الهاتف

#### `Stop_All.ps1`
إيقاف جميع الخدمات

---

## طريقة الاستخدام

### الطريقة السريعة (موصى بها)

1. **تشغيل النظام الكامل:**
   - انقر نقراً مزدوجاً على `0-تشغيل_الكل.bat`
   - انتظر حتى تبدأ جميع الخدمات

2. **إيقاف النظام:**
   - انقر نقراً مزدوجاً على `1-إيقاف_الكل.bat`

### الطريقة المتقدمة (تشغيل منفصل)

1. **تشغيل السيرفر أولاً:**
   ```
   2-تشغيل_السيرفر.bat
   ```

2. **ثم تشغيل الواجهات:**
   ```
   3-تشغيل_الواجهات.bat
   ```

---

## روابط الوصول

بعد تشغيل النظام، يمكنك الوصول إلى:

- **Admin Web**: http://localhost:5174
- **Client Web**: http://localhost:5173
- **Company Portal**: http://localhost:5175

---

## تطبيق الهاتف (Mobile Gateway)

### خطوات تشغيل تطبيق React Native:

1. **بعد تشغيل السكربتات**، سيفتح نافذة PowerShell لـ Metro Bundler

2. **في نافذة Metro Bundler**، قم بتشغيل أحد الأوامر التالية:
   ```bash
   # لتشغيل على Android
   npm run android
   
   # لتشغيل على iOS
   npm run ios
   ```

3. **تأكد من:**
   - تشغيل محاكي Android/iOS أو توصيل جهاز حقيقي
   - تفعيل USB Debugging على الجهاز (Android)

---

## المتطلبات

### البرامج المطلوبة:
- ✅ Docker Desktop
- ✅ Go (للخدمات الخلفية)
- ✅ Node.js (للواجهات وتطبيق الهاتف)
- ✅ Android Studio / Xcode (لتطبيق الهاتف)

### المسار المطلوب:
```
d:\ALHudhudAI
```

---

## استكشاف الأخطاء

### المشكلة: "Project not found"
**الحل:** تأكد من أن المشروع موجود في المسار `d:\ALHudhudAI`

### المشكلة: "Docker failed to start"
**الحل:** 
- تأكد من تشغيل Docker Desktop
- تحقق من أن Docker يعمل بشكل صحيح: `docker ps`

### المشكلة: "Port already in use"
**الحل:**
- أوقف جميع الخدمات باستخدام `1-إيقاف_الكل.bat`
- تحقق من عدم وجود عمليات Node.js أو Go قيد التشغيل

### المشكلة: "Expo Dev Server not starting"
**الحل:**
- تأكد من تثبيت dependencies: `cd C:\ALHudhud\ALHudhud\ALHudhudMobile && npm install`
- امسح cache: `cd C:\ALHudhud\ALHudhud\ALHudhudMobile && npx expo start -c`

---

## ملاحظات مهمة

1. **الترتيب مهم**: يجب تشغيل Docker والخدمات الخلفية قبل الواجهات
2. **الانتظار**: انتظر 5-10 ثوانٍ بين تشغيل كل خدمة
3. **النوافذ المتعددة**: كل خدمة تعمل في نافذة PowerShell منفصلة
4. **الإيقاف الصحيح**: استخدم دائماً `1-إيقاف_الكل.bat` لإيقاف النظام بشكل آمن

---

## الدعم

إذا واجهت أي مشاكل، تحقق من:
- سجلات Docker: `docker compose logs`
- نوافذ PowerShell لكل خدمة
- ملفات السجلات في مجلدات الخدمات

---

**آخر تحديث:** 2024
**الإصدار:** 1.0
