# وثيقة متطلبات الإصلاح: النقاط المتبقية من التدقيق الشامل

## المقدمة

بناءً على تقرير التدقيق الشامل للمشروع، تم تحديد 7 نقاط متبقية تحتاج إلى إصلاح. هذه الوثيقة تحدد المشاكل الموجودة والسلوك المتوقع والسلوك الذي يجب الحفاظ عليه.

**نطاق الإصلاح:** إصلاح النقاط المتبقية فقط بدون إضافة تحسينات جديدة.

**الأولوية:** الأخطاء الحرجة أولاً (CSRF Protection)، ثم المكونات المتبقية حسب التبعيات.

---

## تحليل المشاكل

### 1. السلوك الحالي (العيوب)

#### 1.1 Bug 17: CSRF Protection غير مكتمل (Frontend)

**الحالة:** Backend مكتمل، Frontend غير مكتمل

**الموقع:** 
- `apps/admin-web/src/api/axios.js`
- `apps/admin-web/src/App.jsx`
- `apps/client-web/src/api/axios.js`
- `apps/client-web/src/App.jsx`

**المهام المتبقية:**
- 17.3.1: إنشاء CSRF middleware (Backend) - مكتمل
- 17.3.2: إضافة CSRF endpoint - مكتمل
- 17.3.3: تطبيق middleware على routes - مكتمل
- 17.3.4: تحديث frontend axios.js - **غير مكتمل**
- 17.3.5: تهيئة CSRF في App.jsx - **غير مكتمل**

1.1 WHEN يتم إرسال طلب POST/PUT/DELETE من واجهة الإدارة أو العميل THEN النظام لا يرسل CSRF token في الـ headers

1.2 WHEN يتم تحميل التطبيق لأول مرة THEN النظام لا يقوم بجلب CSRF token من الـ Backend

1.3 WHEN يتم إرسال طلب API بدون CSRF token THEN الـ Backend يرفض الطلب لكن الـ Frontend لا يعالج الخطأ بشكل صحيح

#### 1.2 Preservation Tests مفقودة

**الحالة:** 25 من 30 مكتملة، 5 اختبارات مفقودة

**الموقع:** ملفات الاختبارات في المشاريع الثلاثة

1.4 WHEN يتم البحث عن preservation tests في المشروع THEN يتم العثور على 25 اختبار فقط بدلاً من 30

1.5 WHEN يتم مراجعة ملفات tasks.md THEN بعض المهام المتعلقة بـ preservation tests غير مكتملة

#### 1.3 مكونات واجهة الإدارة المتبقية من Tariff Pricing System

**الحالة:** المهام 23.6-23.8 غير مكتملة

**الموقع:** `apps/admin-web/src/`

**المهام المتبقية:**
- 23.6: إضافة صفحة إدارة البوابات المخصصة
- 23.7: إنشاء مكون GatewayDetails
- 23.8: إضافة صفحة التقارير المالية

1.6 WHEN يحاول المسؤول الوصول إلى صفحة إدارة البوابات المخصصة THEN الصفحة غير موجودة

1.7 WHEN يحاول المسؤول عرض تفاصيل بوابة مخصصة THEN المكون غير موجود

1.8 WHEN يحاول المسؤول الوصول إلى التقارير المالية THEN الصفحة غير مكتملة أو غير موجودة

#### 1.4 مكونات واجهة العميل المتبقية من Tariff Pricing System

**الحالة:** المهام 24.4-24.6 غير مكتملة

**الموقع:** `apps/client-web/src/`

**المهام المتبقية:**
- 24.4: تعديل صفحة الرسائل لإضافة حساب التكلفة
- 24.5: إضافة صفحة الباقات
- 24.6: إضافة صفحة السجل والتقارير

1.9 WHEN يكتب العميل رسالة في صفحة الرسائل THEN لا يتم عرض التكلفة المتوقعة وعدد الأجزاء

1.10 WHEN يحاول العميل الوصول إلى صفحة الباقات THEN الصفحة غير موجودة

1.11 WHEN يحاول العميل الوصول إلى صفحة السجل والتقارير THEN الصفحة غير مكتملة أو غير موجودة

#### 1.5 تبويب الشركات المؤسسية

**الحالة:** المهام 25.1-25.8 غير مكتملة

**الموقع:** `apps/admin-web/src/components/`

**المهام المتبقية:**
- 25.1: إضافة تبويب Enterprise في TenantManager
- 25.2: إنشاء مكون EnterpriseGatewayDashboard
- 25.3: إنشاء مكون EnterpriseGatewaySettings
- 25.4: إنشاء مكون EnterpriseUsers
- 25.5: إنشاء مكون EnterpriseAPIKeys
- 25.6: إنشاء مكون EnterpriseReports
- 25.7: إنشاء مكون EnterpriseAPIDocs
- 25.8: تحديث TenantManager لدعم التبويبات الجديدة

1.12 WHEN يحاول المسؤول عرض معلومات شركة مؤسسية لديها بوابة مخصصة THEN لا يوجد تبويب Enterprise Gateway

1.13 WHEN يحاول المسؤول إدارة مستخدمي بوابة مخصصة THEN المكون غير موجود

1.14 WHEN يحاول المسؤول إدارة API Keys لبوابة مخصصة THEN المكون غير موجود

#### 1.6 API Hooks و Services

**الحالة:** المهام 26.1-26.2 غير مكتملة

**الموقع:** `apps/admin-web/src/hooks/` و `apps/client-web/src/hooks/`

**المهام المتبقية:**
- 26.1: تطوير Tariff Hooks
- 26.2: تطوير Package Hooks

1.15 WHEN يحاول مكون React استخدام useTariffs hook THEN الـ hook غير موجود

1.16 WHEN يحاول مكون React استخدام usePackages hook THEN الـ hook غير موجود

#### 1.7 Frontend Development لـ Admin Interface Enhancements

**الحالة:** المرحلة 7 كاملة غير مكتملة (إعداد مشروع + 6 صفحات)

**الموقع:** `apps/admin-web-v2/` (مجلد جديد)

**المهام المتبقية:**
- 20.1-20.3: إعداد مشروع Frontend
- 21.1-21.3: تطوير صفحة Overview
- 22.1-22.4: تطوير صفحة Companies
- 23.1-23.4: تطوير صفحة Servers
- 24.1-24.4: تطوير صفحة Distribution
- 25.1-25.3: تطوير صفحة Signal/Push
- 26.1-26.3: تطوير صفحة Alerts

1.17 WHEN يحاول المسؤول الوصول إلى واجهة الإدارة المحسّنة (v2) THEN المشروع غير موجود

1.18 WHEN يحاول المسؤول عرض نظرة عامة على النظام THEN الصفحة غير موجودة

1.19 WHEN يحاول المسؤول عرض تقييمات الشركات THEN الصفحة غير موجودة

1.20 WHEN يحاول المسؤول مراقبة صحة السيرفرات THEN الصفحة غير موجودة

1.21 WHEN يحاول المسؤول إدارة توزيع الرسائل THEN الصفحة غير موجودة

1.22 WHEN يحاول المسؤول مراقبة نظام Signal/Push THEN الصفحة غير موجودة

1.23 WHEN يحاول المسؤول إدارة التنبيهات THEN الصفحة غير موجودة

---

### 2. السلوك المتوقع (الصحيح)

#### 2.1 CSRF Protection مكتمل

2.1 WHEN يتم إرسال طلب POST/PUT/DELETE من واجهة الإدارة أو العميل THEN النظام SHALL يرسل CSRF token في الـ headers تلقائياً

2.2 WHEN يتم تحميل التطبيق لأول مرة THEN النظام SHALL يقوم بجلب CSRF token من endpoint `/api/csrf-token` وحفظه

2.3 WHEN يتم إرسال طلب API بدون CSRF token THEN الـ Backend SHALL يرفض الطلب والـ Frontend SHALL يعرض رسالة خطأ واضحة

2.4 WHEN ينتهي صلاحية CSRF token THEN النظام SHALL يقوم بتحديث الـ token تلقائياً

#### 2.2 Preservation Tests مكتملة

2.5 WHEN يتم البحث عن preservation tests في المشروع THEN النظام SHALL يحتوي على 30 اختبار preservation كاملة

2.6 WHEN يتم تشغيل preservation tests THEN جميع الاختبارات SHALL تنجح وتؤكد عدم وجود regressions

#### 2.3 مكونات واجهة الإدارة مكتملة

2.7 WHEN يحاول المسؤول الوصول إلى صفحة إدارة البوابات المخصصة THEN النظام SHALL يعرض الصفحة مع جدول البوابات والإحصائيات

2.8 WHEN يحاول المسؤول عرض تفاصيل بوابة مخصصة THEN النظام SHALL يعرض مكون GatewayDetails مع جميع المعلومات

2.9 WHEN يحاول المسؤول الوصول إلى التقارير المالية THEN النظام SHALL يعرض الصفحة مع التقارير والرسوم البيانية

#### 2.4 مكونات واجهة العميل مكتملة

2.10 WHEN يكتب العميل رسالة في صفحة الرسائل THEN النظام SHALL يعرض التكلفة المتوقعة وعدد الأجزاء تلقائياً

2.11 WHEN يحاول العميل الوصول إلى صفحة الباقات THEN النظام SHALL يعرض جميع الباقات المتاحة مع التفاصيل

2.12 WHEN يحاول العميل الوصول إلى صفحة السجل والتقارير THEN النظام SHALL يعرض سجل الرسائل والمعاملات المالية

#### 2.5 تبويب الشركات المؤسسية مكتمل

2.13 WHEN يحاول المسؤول عرض معلومات شركة مؤسسية لديها بوابة مخصصة THEN النظام SHALL يعرض تبويب Enterprise Gateway

2.14 WHEN يحاول المسؤول إدارة مستخدمي بوابة مخصصة THEN النظام SHALL يعرض مكون EnterpriseUsers مع جميع الوظائف

2.15 WHEN يحاول المسؤول إدارة API Keys لبوابة مخصصة THEN النظام SHALL يعرض مكون EnterpriseAPIKeys مع جميع الوظائف

#### 2.6 API Hooks و Services مكتملة

2.16 WHEN يحاول مكون React استخدام useTariffs hook THEN الـ hook SHALL يكون متاحاً ويعمل بشكل صحيح

2.17 WHEN يحاول مكون React استخدام usePackages hook THEN الـ hook SHALL يكون متاحاً ويعمل بشكل صحيح

#### 2.7 Frontend Development لـ Admin Interface Enhancements مكتمل

2.18 WHEN يحاول المسؤول الوصول إلى واجهة الإدارة المحسّنة (v2) THEN النظام SHALL يعرض المشروع الجديد

2.19 WHEN يحاول المسؤول عرض نظرة عامة على النظام THEN النظام SHALL يعرض صفحة Overview مع الإحصائيات

2.20 WHEN يحاول المسؤول عرض تقييمات الشركات THEN النظام SHALL يعرض صفحة Companies مع جدول التقييمات

2.21 WHEN يحاول المسؤول مراقبة صحة السيرفرات THEN النظام SHALL يعرض صفحة Servers مع حالة السيرفرات

2.22 WHEN يحاول المسؤول إدارة توزيع الرسائل THEN النظام SHALL يعرض صفحة Distribution مع الاستراتيجيات

2.23 WHEN يحاول المسؤول مراقبة نظام Signal/Push THEN النظام SHALL يعرض صفحة Signal/Push مع الإحصائيات

2.24 WHEN يحاول المسؤول إدارة التنبيهات THEN النظام SHALL يعرض صفحة Alerts مع التنبيهات النشطة

---

### 3. السلوك الذي يجب الحفاظ عليه (منع الانحدار)

#### 3.1 CSRF Protection Backend

3.1 WHEN يتم إرسال طلب GET/HEAD/OPTIONS THEN النظام SHALL CONTINUE TO يسمح بالطلب بدون CSRF token

3.2 WHEN يتم إرسال طلب من API مع API Key صحيح THEN النظام SHALL CONTINUE TO يسمح بالطلب بدون CSRF token

#### 3.2 الوظائف الموجودة في واجهة الإدارة

3.3 WHEN يستخدم المسؤول الوظائف الموجودة في واجهة الإدارة (TenantManager، GatewayManager، إلخ) THEN جميع الوظائف SHALL CONTINUE TO تعمل بشكل صحيح

3.4 WHEN يستخدم المسؤول التعرفات والباقات الموجودة THEN جميع الوظائف SHALL CONTINUE TO تعمل بشكل صحيح

#### 3.3 الوظائف الموجودة في واجهة العميل

3.5 WHEN يستخدم العميل الوظائف الموجودة (إرسال رسائل، عرض الرصيد، إلخ) THEN جميع الوظائف SHALL CONTINUE TO تعمل بشكل صحيح

3.6 WHEN يستخدم العميل معلومات الباقة الموجودة THEN جميع الوظائف SHALL CONTINUE TO تعمل بشكل صحيح

#### 3.4 Backend APIs

3.7 WHEN يتم استدعاء Backend APIs الموجودة THEN جميع الـ APIs SHALL CONTINUE TO تعمل بشكل صحيح

3.8 WHEN يتم استخدام نظام التقييم والمقاييس THEN جميع الوظائف SHALL CONTINUE TO تعمل بشكل صحيح

#### 3.5 الاختبارات الموجودة

3.9 WHEN يتم تشغيل الاختبارات الموجودة (Bug Condition Tests، Unit Tests) THEN جميع الاختبارات SHALL CONTINUE TO تنجح

3.10 WHEN يتم تشغيل Property Tests الموجودة THEN جميع الاختبارات SHALL CONTINUE TO تنجح

---

## الخلاصة

هذه الوثيقة تحدد 7 نقاط متبقية تحتاج إلى إصلاح:

1. **Bug 17: CSRF Protection غير مكتمل** (Frontend) - أولوية حرجة
2. **Preservation Tests مفقودة** - 5 اختبارات
3. **مكونات واجهة الإدارة المتبقية** - 3 مهام (23.6-23.8)
4. **مكونات واجهة العميل المتبقية** - 3 مهام (24.4-24.6)
5. **تبويب الشركات المؤسسية** - 8 مهام (25.1-25.8)
6. **API Hooks و Services** - 2 مهام (26.1-26.2)
7. **Frontend Development لـ Admin Interface Enhancements** - المرحلة 7 كاملة

**الأولوية:**
- حرجة: Bug 17 (CSRF Protection)
- عالية: Preservation Tests، API Hooks
- متوسطة: مكونات واجهة الإدارة والعميل، تبويب الشركات المؤسسية
- منخفضة: Frontend Development لـ Admin Interface Enhancements (واجهة جديدة)

**التبعيات:**
- API Hooks يجب أن تكتمل قبل المكونات التي تستخدمها
- CSRF Protection يجب أن يكتمل أولاً لأنه يؤثر على الأمان
- Preservation Tests يجب أن تكتمل للتأكد من عدم وجود regressions
