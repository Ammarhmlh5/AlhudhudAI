# قائمة مهام إصلاح النقاط المتبقية

## ملخص المهام

بعد المراجعة الشاملة، تبين أن معظم المكونات مكتملة بالفعل:
- ✅ Bug 17 (CSRF Protection) - مكتمل
- ✅ مكونات واجهة الإدارة (Gateways, GatewayDetails, Reports) - مكتملة
- ✅ مكونات واجهة العميل (SendMessage, Packages, HistoryReports) - مكتملة
- ❌ usePackages hook - مفقود (المهمة الرئيسية)
- ❓ Preservation Tests - يحتاج تحقق

## المرحلة 1: التحقق من الحالة الحالية

- [x] 1.1 مراجعة Preservation Tests
  - [x] 1.1.1 البحث عن جميع ملفات preservation tests
  - [x] 1.1.2 مراجعة ملفات tasks.md في جميع المشاريع
  - [x] 1.1.3 تحديد الاختبارات المفقودة (إن وجدت)
  - [x] 1.1.4 توثيق الاختبارات المطلوبة

- [x] 1.2 التحقق من API Hooks
  - [x] 1.2.1 التحقق من useTariffs.ts في admin-web
  - [x] 1.2.2 التحقق من useGateways.ts في admin-web
  - [x] 1.2.3 التأكد من عدم وجود usePackages.ts
  - [x] 1.2.4 توثيق المتطلبات لـ usePackages hook

## المرحلة 2: إنشاء usePackages Hook في admin-web

- [x] 2.1 إنشاء ملف usePackages.ts
  - [x] 2.1.1 إنشاء apps/admin-web/src/hooks/usePackages.ts
  - [x] 2.1.2 إضافة imports (React, axios, types)
  - [x] 2.1.3 تعريف TypeScript interfaces (Package, PackageFilters, CreatePackageData, UpdatePackageData)

- [x] 2.2 تنفيذ usePackages() - جلب جميع الباقات
  - [x] 2.2.1 إنشاء state (packages, loading, error)
  - [x] 2.2.2 تنفيذ useEffect لجلب البيانات
  - [x] 2.2.3 معالجة filters (اختيارية)
  - [x] 2.2.4 معالجة الأخطاء
  - [x] 2.2.5 إرجاع { packages, loading, error }

- [x] 2.3 تنفيذ usePackage(packageId) - جلب باقة واحدة
  - [x] 2.3.1 إنشاء state (package, loading, error)
  - [x] 2.3.2 تنفيذ useEffect لجلب البيانات
  - [x] 2.3.3 معالجة حالة packageId === null
  - [x] 2.3.4 معالجة الأخطاء
  - [x] 2.3.5 إرجاع { package, loading, error }

- [x] 2.4 تنفيذ useCreatePackage() - إنشاء باقة
  - [x] 2.4.1 إنشاء state (loading, error)
  - [x] 2.4.2 تنفيذ دالة createPackage
  - [x] 2.4.3 إرسال POST request إلى /api/packages
  - [x] 2.4.4 معالجة الأخطاء
  - [x] 2.4.5 إرجاع { createPackage, loading, error }

- [x] 2.5 تنفيذ useUpdatePackage() - تحديث باقة
  - [x] 2.5.1 إنشاء state (loading, error)
  - [x] 2.5.2 تنفيذ دالة updatePackage
  - [x] 2.5.3 إرسال PUT request إلى /api/packages/:id
  - [x] 2.5.4 معالجة الأخطاء
  - [x] 2.5.5 إرجاع { updatePackage, loading, error }

- [x] 2.6 تنفيذ useDeletePackage() - حذف باقة
  - [x] 2.6.1 إنشاء state (loading, error)
  - [x] 2.6.2 تنفيذ دالة deletePackage
  - [x] 2.6.3 إرسال DELETE request إلى /api/packages/:id
  - [x] 2.6.4 معالجة الأخطاء
  - [x] 2.6.5 إرجاع { deletePackage, loading, error }

## المرحلة 3: إنشاء usePackages Hook في client-web

- [x] 3.1 إنشاء ملف usePackages.ts
  - [x] 3.1.1 إنشاء apps/client-web/src/hooks/usePackages.ts
  - [x] 3.1.2 إضافة imports (React, axios, types)
  - [x] 3.1.3 تعريف TypeScript interfaces (Package, SubscribeData)

- [x] 3.2 تنفيذ usePackages() - جلب جميع الباقات المتاحة
  - [x] 3.2.1 إنشاء state (packages, loading, error)
  - [x] 3.2.2 تنفيذ useEffect لجلب البيانات
  - [x] 3.2.3 جلب الباقات النشطة فقط (is_active = true)
  - [x] 3.2.4 معالجة الأخطاء
  - [x] 3.2.5 إرجاع { packages, loading, error, refetch }

- [x] 3.3 تنفيذ usePackage(packageId) - جلب باقة واحدة
  - [x] 3.3.1 إنشاء state (package, loading, error)
  - [x] 3.3.2 تنفيذ useEffect لجلب البيانات
  - [x] 3.3.3 معالجة حالة packageId === null
  - [x] 3.3.4 معالجة الأخطاء
  - [x] 3.3.5 إرجاع { package, loading, error }

- [x] 3.4 تنفيذ useSubscribeToPackage() - الاشتراك في باقة
  - [x] 3.4.1 إنشاء state (loading, error)
  - [x] 3.4.2 تنفيذ دالة subscribeToPackage
  - [x] 3.4.3 إرسال POST request إلى /api/v1/client/subscribe
  - [x] 3.4.4 معالجة الأخطاء
  - [x] 3.4.5 إرجاع { subscribeToPackage, loading, error }

- [x] 3.5 تنفيذ useCurrentPackage() - جلب الباقة الحالية
  - [x] 3.5.1 إنشاء state (currentPackage, loading, error)
  - [x] 3.5.2 تنفيذ useEffect لجلب البيانات
  - [x] 3.5.3 جلب الباقة النشطة للعميل الحالي
  - [x] 3.5.4 معالجة الأخطاء
  - [x] 3.5.5 إرجاع { currentPackage, loading, error, refetch }

## المرحلة 4: إصلاح Preservation Tests المفقودة (إن وجدت)

- [x] 4.1 تحديد الاختبارات المفقودة بدقة
  - [x] 4.1.1 مراجعة .kiro/specs/*/tasks.md
  - [x] 4.1.2 إنشاء قائمة بالاختبارات المطلوبة
  - [x] 4.1.3 مقارنة مع الاختبارات الموجودة

- [x] 4.2 إنشاء الاختبارات المفقودة (إن وجدت)
  - [x] 4.2.1 إنشاء اختبار preservation 1 (إذا كان مفقوداً)
  - [x] 4.2.2 إنشاء اختبار preservation 2 (إذا كان مفقوداً)
  - [x] 4.2.3 إنشاء اختبار preservation 3 (إذا كان مفقوداً)
  - [x] 4.2.4 إنشاء اختبار preservation 4 (إذا كان مفقوداً)
  - [x] 4.2.5 إنشاء اختبار preservation 5 (إذا كان مفقوداً)

- [x] 4.3 تشغيل الاختبارات الجديدة
  - [x] 4.3.1 تشغيل جميع preservation tests
  - [x] 4.3.2 التحقق من نجاح الاختبارات
  - [x] 4.3.3 إصلاح أي أخطاء

## المرحلة 5: الاختبار والتحقق

- [x] 5.1 اختبار usePackages hooks
  - [x] 5.1.1 اختبار usePackages في admin-web
  - [x] 5.1.2 اختبار usePackage في admin-web
  - [x] 5.1.3 اختبار useCreatePackage في admin-web
  - [x] 5.1.4 اختبار useUpdatePackage في admin-web
  - [x] 5.1.5 اختبار useDeletePackage في admin-web
  - [x] 5.1.6 اختبار usePackages في client-web
  - [x] 5.1.7 اختبار useSubscribeToPackage في client-web
  - [x] 5.1.8 اختبار useCurrentPackage في client-web

- [x] 5.2 اختبار التكامل
  - [x] 5.2.1 اختبار استخدام usePackages في Packages.jsx (client-web)
  - [x] 5.2.2 اختبار استخدام usePackages في PackageSettings.jsx (admin-web)
  - [x] 5.2.3 التحقق من عمل جميع الوظائف

- [x] 5.3 اختبار الحفاظ على السلوك (Preservation Tests)
  - [x] 5.3.1 تشغيل جميع preservation tests الموجودة
  - [x] 5.3.2 تشغيل جميع bug condition tests الموجودة
  - [x] 5.3.3 التحقق من نجاح جميع الاختبارات

- [x] 5.4 الاختبار اليدوي
  - [x] 5.4.1 اختبار واجهة الإدارة (إدارة الباقات)
  - [x] 5.4.2 اختبار واجهة العميل (عرض الباقات، الاشتراك)
  - [x] 5.4.3 التحقق من عدم وجود regressions

## المرحلة 6: التوثيق والتسليم

- [x] 6.1 تحديث التوثيق
  - [x] 6.1.1 توثيق usePackages hook في admin-web
  - [x] 6.1.2 توثيق usePackages hook في client-web
  - [x] 6.1.3 إضافة أمثلة استخدام

- [x] 6.2 مراجعة نهائية
  - [x] 6.2.1 مراجعة جميع التغييرات
  - [x] 6.2.2 التحقق من عدم وجود أخطاء
  - [x] 6.2.3 التحقق من عدم وجود regressions
  - [x] 6.2.4 التحقق من اتباع نفس النمط المستخدم في الكود الموجود

- [x] 6.3 التسليم
  - [x] 6.3.1 إنشاء تقرير الإصلاح
  - [x] 6.3.2 تحديث ملف tasks.md
  - [x] 6.3.3 إغلاق المهمة

## ملاحظات

- جميع المهام يجب أن تكتمل بالترتيب
- يجب اختبار كل مرحلة قبل الانتقال إلى المرحلة التالية
- يجب التأكد من عدم حدوث regressions في الوظائف الموجودة
- يجب التركيز على الإصلاح فقط بدون إضافة تحسينات جديدة
- يجب اتباع نفس النمط المستخدم في useTariffs.ts و useGateways.ts
