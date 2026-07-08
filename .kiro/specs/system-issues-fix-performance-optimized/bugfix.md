# Bugfix Requirements Document

## Introduction

هذا المستند يحدد متطلبات إصلاح 8 مشاكل تم اكتشافها في مشروع Al-Hudhud SMS Gateway من خلال فحص دقيق للكود. المشاكل تتراوح بين مشاكل أمنية حرجة (JWT_SECRET غير آمن، REDIS_PASSWORD فارغ) ومشاكل وظيفية عالية الأهمية (نظام الإحالة معطّل، وظائف التصدير غير مطبقة، Rate Limiting معطّل عند فشل Redis) ومشاكل متوسطة الأهمية (Active Invalidation للقنوات، متغيرات Grafana مفقودة، عدم تطابق PORT).

الهدف من هذا الإصلاح هو معالجة جميع المشاكل دون التأثير على سرعة الأداء الحالية، مع الحفاظ على الكود الموجود وعدم كسر الوظائف الحالية، واستخدام best practices للأداء.

## Bug Analysis

### Current Behavior (Defect)

#### 1. مشاكل الأمان الحرجة

1.1 WHEN يتم تشغيل النظام في بيئة الإنتاج THEN النظام يستخدم JWT_SECRET بقيمة "change-me-in-production-to-a-secure-random-string" وهي قيمة واضحة وقابلة للتخمين مما يشكل ثغرة أمنية خطيرة

1.2 WHEN يتم الاتصال بـ Redis THEN النظام يستخدم REDIS_PASSWORD فارغ مما يعني أن Redis يعمل بدون حماية ويمكن لأي شخص الوصول إليه على localhost

#### 2. مشاكل الوظائف المعطلة

1.3 WHEN يحاول المستخدم استخدام نظام الإحالة (Referral System) THEN النظام لا يستجيب لأن الكود موجود بالكامل في services/backend/internal/referral/ لكنه غير مفعّل في main.go ولا توجد API endpoints

1.4 WHEN يحاول المستخدم تصدير تقارير الإحالة بصيغة JSON أو CSV أو PDF THEN النظام يرجع خطأ "not implemented" لأن الوظائف exportToJSON, exportToCSV, exportToPDF في services/backend/internal/referral/reports.go غير مطبقة

1.5 WHEN يفشل الاتصال بـ Redis THEN النظام يعطّل Rate Limiting بالكامل (السطر 416-419 في main.go) مما يجعل النظام عرضة لهجمات DDoS

#### 3. مشاكل الأداء والتكوين

1.6 WHEN تنقطع قناة إرسال (channel goes offline) THEN النظام لا يحذف الـ cache فوراً ويستمر في استخدام بيانات قديمة لمدة 5 دقائق (TTL) مما يؤدي لفشل إرسال الرسائل

1.7 WHEN يحاول المستخدم الوصول إلى Grafana monitoring THEN النظام لا يعمل بشكل صحيح لأن المتغيرات GRAFANA_ADMIN_USER, GRAFANA_ADMIN_PASSWORD, GRAFANA_ROOT_URL, GRAFANA_PLUGINS مفقودة من ملف .env

1.8 WHEN يحاول المطور الجديد الاتصال بالـ API THEN يحدث ارتباك لأن ملف .env يحدد PORT=8080 بينما README.md يذكر localhost:8081 وملف main.go يستخدم 8081 كقيمة افتراضية

### Expected Behavior (Correct)

#### 1. إصلاح مشاكل الأمان الحرجة

2.1 WHEN يتم تشغيل النظام في أي بيئة THEN النظام SHALL يستخدم JWT_SECRET عشوائي وآمن بطول لا يقل عن 48 حرف base64 ويتم توليده باستخدام openssl rand -base64 48

2.2 WHEN يتم الاتصال بـ Redis THEN النظام SHALL يستخدم REDIS_PASSWORD قوي ومطابق للقيمة المحددة في redis.conf ويتم توليده باستخدام openssl rand -base64 32

#### 2. تفعيل الوظائف المعطلة

2.3 WHEN يحاول المستخدم استخدام نظام الإحالة THEN النظام SHALL يوفر API endpoints كاملة (/api/v1/referrals/*) ويفعّل ReferralManager في main.go مع التكوين المناسب في config.go

2.4 WHEN يحاول المستخدم تصدير تقارير الإحالة بصيغة JSON أو CSV أو PDF THEN النظام SHALL ينفذ الوظائف exportToJSON (باستخدام encoding/json), exportToCSV (باستخدام encoding/csv), exportToPDF (باستخدام github.com/jung-kurt/gofpdf) ويرجع البيانات بالصيغة المطلوبة

2.5 WHEN يفشل الاتصال بـ Redis THEN النظام SHALL يستخدم rate limiter بديل يعمل في الذاكرة (in-memory) باستخدام golang.org/x/time/rate مع cleanup دوري للحفاظ على الأداء

#### 3. إصلاح مشاكل الأداء والتكوين

2.6 WHEN تنقطع قناة إرسال (channel status changes to offline) THEN النظام SHALL يحذف الـ cache فوراً باستخدام دالة InvalidateChannelOnOffline() ويضيف health check دوري للقنوات

2.7 WHEN يتم تشغيل النظام THEN ملف .env SHALL يحتوي على جميع متغيرات Grafana المطلوبة (GRAFANA_ADMIN_USER, GRAFANA_ADMIN_PASSWORD, GRAFANA_ROOT_URL, GRAFANA_PLUGINS) مع قيم افتراضية آمنة

2.8 WHEN يحاول المطور الاتصال بالـ API THEN جميع الملفات (.env, README.md, main.go) SHALL تستخدم نفس رقم PORT (8081) لتجنب الارتباك

### Unchanged Behavior (Regression Prevention)

#### 1. الحفاظ على الأداء الحالي

3.1 WHEN يتم إصلاح المشاكل الأمنية (JWT_SECRET, REDIS_PASSWORD) THEN النظام SHALL CONTINUE TO يعمل بنفس سرعة الأداء الحالية دون أي تأخير في المصادقة أو الوصول للـ cache

3.2 WHEN يتم تفعيل نظام الإحالة THEN النظام SHALL CONTINUE TO يعمل بنفس الأداء للوظائف الأخرى (إرسال الرسائل، إدارة المستخدمين، التقارير) دون أي تأثير على سرعة الاستجابة

3.3 WHEN يتم إضافة rate limiter بديل (in-memory) THEN النظام SHALL CONTINUE TO يطبق نفس حدود الطلبات (100 request/minute) بنفس الدقة عند توفر Redis

#### 2. الحفاظ على الوظائف الحالية

3.4 WHEN يتم إصلاح Active Invalidation للقنوات THEN النظام SHALL CONTINUE TO يستخدم الـ cache بشكل طبيعي للقنوات النشطة مع نفس TTL (5 دقائق) ونفس آلية التحديث

3.5 WHEN يتم إضافة متغيرات Grafana THEN النظام SHALL CONTINUE TO يعمل بشكل طبيعي حتى لو لم يتم تكوين Grafana (optional feature)

3.6 WHEN يتم توحيد PORT في جميع الملفات THEN النظام SHALL CONTINUE TO يقبل الاتصالات على نفس المنفذ المستخدم حالياً دون الحاجة لتغيير تكوينات العملاء الحاليين

#### 3. الحفاظ على سلامة البيانات

3.7 WHEN يتم تطبيق وظائف تصدير التقارير THEN النظام SHALL CONTINUE TO يحافظ على نفس بنية البيانات في ReferralReport دون أي تغيير في الـ schema أو الـ API responses

3.8 WHEN يتم إصلاح جميع المشاكل THEN النظام SHALL CONTINUE TO يحافظ على جميع البيانات الموجودة في قاعدة البيانات (referrals, tenants, messages, channels) دون أي فقدان أو تلف

3.9 WHEN يتم تشغيل النظام بعد الإصلاحات THEN جميع الاختبارات الموجودة (bug15, bug17, bug23, bug26, bug28, bug29) SHALL CONTINUE TO تنجح بنفس النتائج السابقة
