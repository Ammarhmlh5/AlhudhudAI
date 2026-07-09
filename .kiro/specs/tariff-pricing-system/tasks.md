# خطة التنفيذ - نظام التعرفات والتسعير

## نظرة عامة

هذه الخطة تغطي تنفيذ نظام التعرفات والتسعير الشامل الذي يدعم ثلاثة نماذج خدمات:
1. **خدمة بوابة الهدهد**: نموذج الدفع حسب الاستخدام
2. **خدمة الإرسال عبر الهاتف**: نموذج الاشتراك الشهري مع Rollover
3. **خدمة إدارة البوابات المخصصة**: نموذج Gateway-as-a-Service للشركات الكبيرة

### الأهداف الرئيسية

- معالجة جميع المشاكل الحرجة في النظام الحالي (Race Conditions، Idempotency، إلخ)
- تطبيق العزل التام بين الشركات (Multi-Tenancy Isolation)
- بناء نظام تعرفات ديناميكي ومرن
- تطوير نظام باقات شهرية مع Rollover
- إنشاء بنية تحتية مشتركة للبوابات المخصصة

### لغة البرمجة

**Go (Golang)** - للاستمرارية مع النظام الحالي والأداء العالي

### المدة المتوقعة

14 أسبوع (7 مراحل × أسبوعين لكل مرحلة)

---

## المهام


## المرحلة 1: الأساسيات والبنية التحتية (الأسابيع 1-2)

- [x] 1. إنشاء جداول قاعدة البيانات الأساسية
  - [x] 1.1 إنشاء جدول services مع tenant_id وRow Level Security
    - إنشاء Migration لجدول services
    - تطبيق RLS Policies (SELECT، INSERT، UPDATE، DELETE)
    - إضافة Indexes مركبة تبدأ بـ tenant_id
    - إدراج البيانات الأولية للخدمات الثلاثة
    - _Requirements: 1.1, 1.2, 1.3_
  
  - [x] 1.2 إنشاء جدول tariffs مع العزل الكامل
    - إنشاء Migration لجدول tariffs مع tenant_id
    - تطبيق RLS Policies لعزل التعرفات بين الشركات
    - إضافة Indexes: idx_tariffs_tenant_service، idx_tariffs_tenant_country، idx_tariffs_tenant_operator
    - إدراج التعرفات الأولية لبوابة الهدهد (9 ريال، 13 ريال)
    - _Requirements: 2.1, 2.2, 2.6_
  
  - [x] 1.3 إنشاء جدول segmentation_rules
    - إنشاء Migration لجدول segmentation_rules
    - إضافة قواعد التقسيم الافتراضية (70، 65، 62 حرف)
    - دعم GSM-7 و Unicode
    - _Requirements: 7.1, 7.2, 7.3, 7.4_
  
  - [x] 1.4 إنشاء جداول العملات وأسعار الصرف
    - إنشاء Migration لجدول currencies
    - إنشاء Migration لجدول exchange_rates
    - إدراج العملات الأولية (USD، YER، SAR)
    - إدراج أسعار الصرف الأولية
    - _Requirements: 31.1, 31.2, 31.3_
  
  - [x] 1.5 إنشاء جدول balance_history للـ Audit Trail
    - إنشاء Migration لجدول balance_history مع tenant_id
    - تطبيق RLS Policies
    - إضافة Indexes: idx_balance_history_tenant، idx_balance_history_created
    - _Requirements: 16.4, 16.5, 16.10_

- [x] 2. تطوير Balance Manager المحسّن (معالجة المشاكل الحرجة)
  - [x] 2.1 تطوير Atomic Balance Deduction
    - إنشاء ملف internal/balance/manager.go
    - تطبيق DeductBalance مع Transaction Isolation Level SERIALIZABLE
    - استخدام Atomic Update مع WHERE condition (balance >= amount)
    - تسجيل في balance_history بعد كل عملية
    - _Requirements: 16.1, 16.2, 16.4_
  
  - [x] 2.2 كتابة Property Test لـ Atomic Balance Operations
    - **Property 15: Atomic Balance Operations**
    - **Validates: Requirements 16.1, 16.2**
    - إنشاء ملف internal/balance/manager_property_test.go
    - اختبار عمليات متزامنة على نفس الحساب
    - التحقق من عدم وجود Race Conditions
  
  - [x] 2.3 تطوير Idempotent Refund
    - تطبيق RefundBalance مع فحص الحالة (Already Refunded)
    - استخدام Atomic Update لتحديث حالة المعاملة
    - تسجيل في balance_history
    - _Requirements: 16.8, 16.9_
  
  - [x] 2.4 كتابة Unit Tests لـ Balance Manager
    - اختبار DeductBalance مع رصيد كافي
    - اختبار DeductBalance مع رصيد غير كافي
    - اختبار RefundBalance مع Idempotency
    - اختبار Concurrent Operations
  
  - [x] 2.5 تطبيق Context with Timeout
    - إضافة DeductBalanceWithTimeout مع timeout 5 ثواني
    - معالجة Context Cancellation
    - _Requirements: 27.1, 27.2_

- [x] 3. تطوير Tariff Engine (نظام التعرفات الديناميكي)
  - [x] 3.1 تطوير محرك حساب التكلفة
    - إنشاء ملف internal/tariff/engine.go
    - تطبيق CalculateCost مع دعم التعرفات الديناميكية
    - دعم حساب التكلفة حسب الشبكة والدولة
    - تكامل مع Redis للـ Caching (TTL: 1 ساعة)
    - _Requirements: 2.5, 2.6_
  
  - [x] 3.2 كتابة Property Test لاستخدام التعرفة النشطة
    - **Property 3: استخدام التعرفة النشطة الصحيحة**
    - **Validates: Requirements 2.5**
    - التحقق من استخدام التعرفة النشطة فقط
    - التحقق من تطبيق effective_from و effective_to
  
  - [x] 3.3 تطوير محرك حساب أجزاء الرسائل
    - تطبيق CalculateSegments حسب قواعد التقسيم
    - دعم GSM-7 و Unicode
    - تطبيق القواعد: الجزء الأول 70، الثاني 65، الثالث 62
    - _Requirements: 7.1, 7.2, 7.3, 7.6_
  
  - [x] 3.4 كتابة Property Test لحساب أجزاء الرسالة
    - **Property 12: حساب أجزاء الرسالة بدقة**
    - **Validates: Requirements 7.6**
    - اختبار رسائل بأطوال مختلفة
    - التحقق من تطبيق القواعد الصحيحة
  
  - [x] 3.5 تطوير Tariff Cache Manager
    - إنشاء ملف internal/tariff/cache.go
    - تطبيق GetTariffFromCache و SetTariffToCache
    - تطبيق Cache Invalidation عند تحديث التعرفات
    - _Requirements: 27.9_

- [x] 4. تطوير Tenant Isolation Middleware
  - [x] 4.1 تطوير Tenant Validation Middleware
    - إنشاء ملف internal/middleware/tenant_isolation.go
    - استخراج tenant_id من JWT أو API Key
    - تعيين tenant_id في Context
    - تعيين app.current_tenant في Database Session
    - _Requirements: 26.1, 26.2, 26.3_
  
  - [x] 4.2 تطوير Cross-Tenant Prevention Middleware
    - فحص جميع المعاملات في الطلب
    - منع الوصول لموارد tenant آخر
    - تسجيل محاولات Cross-Tenant Access
    - _Requirements: 26.7, 26.11_
  
  - [x] 4.3 كتابة Integration Tests للعزل
    - **Property 16: Tenant Isolation في قاعدة البيانات**
    - **Validates: Requirements 26.5**
    - اختبار عدم تسرب البيانات بين Tenants
    - اختبار RLS Policies

- [x] 5. Checkpoint - التحقق من الأساسيات
  - التأكد من نجاح جميع الاختبارات
  - التحقق من عمل RLS بشكل صحيح
  - التحقق من عدم وجود Race Conditions
  - مراجعة الكود مع الفريق
  - سؤال المستخدم عن أي استفسارات


## المرحلة 2: الباقات الشهرية (الأسابيع 3-4)

- [x] 6. إنشاء جداول الباقات والاشتراكات
  - [x] 6.1 إنشاء جدول packages مع العزل الكامل
    - إنشاء Migration لجدول packages مع tenant_id
    - تطبيق RLS Policies
    - إضافة Indexes: idx_packages_tenant_service، idx_packages_tenant_active
    - إدراج الباقات الأولية (Basic، Standard، Professional، Business، Enterprise)
    - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5, 3.6_
  
  - [x] 6.2 إنشاء جدول subscriptions مع tenant_id
    - إنشاء Migration لجدول subscriptions
    - تطبيق RLS Policies
    - إضافة Indexes: idx_subscriptions_tenant، idx_subscriptions_renewal
    - _Requirements: 4.3, 4.4, 4.5_
  
  - [x] 6.3 إنشاء جدول rollover_messages
    - إنشاء Migration لجدول rollover_messages مع tenant_id
    - تطبيق RLS Policies
    - إضافة Indexes: idx_rollover_tenant_subscription، idx_rollover_tenant_expiry
    - _Requirements: 6.2, 6.3, 6.7_

- [x] 7. تطوير Subscription Manager
  - [x] 7.1 تطوير وظيفة الاشتراك في الباقات
    - إنشاء ملف internal/subscription/manager.go
    - تطبيق Subscribe مع التحقق من كفاية الرصيد
    - خصم قيمة الباقة من الرصيد بشكل ذري (Atomic)
    - تسجيل في accounting_transactions
    - _Requirements: 4.1, 4.2, 4.3, 4.4_
  
  - [x] 7.2 كتابة Property Test للاشتراك الآمن
    - **Property 5: الاشتراك الآمن في الباقة**
    - **Validates: Requirements 4.1, 4.2**
    - اختبار الاشتراك مع رصيد كافي
    - اختبار الاشتراك مع رصيد غير كافي
    - التحقق من Atomicity
  
  - [x] 7.3 تطوير وظيفة خصم من عداد الرسائل
    - تطبيق DeductMessages مع Atomic Update
    - دعم خصم عدة أجزاء للرسائل الطويلة
    - تحديث messages_used و messages_remaining
    - _Requirements: 4.5, 4.6_
  
  - [x] 7.4 كتابة Property Test لخصم من العداد
    - **Property 6: خصم من عداد الرسائل عند الإرسال**
    - **Validates: Requirements 4.5**
    - التحقق من انخفاض messages_remaining
    - التحقق من زيادة messages_used
  
  - [x] 7.5 تطوير وظيفة حساب Overage
    - تطبيق CalculateOverageCost
    - خصم تكلفة الرسائل الإضافية من الرصيد
    - تسجيل overage_messages و overage_cost
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5_
  
  - [x] 7.6 كتابة Property Test لخصم Overage
    - **Property 8: خصم تكلفة الرسائل الإضافية من الرصيد**
    - **Validates: Requirements 5.3, 5.4**
    - اختبار Overage مع رصيد كافي
    - اختبار Overage مع رصيد غير كافي
  
  - [x] 7.7 تطوير وظيفة عرض الاستخدام
    - تطبيق GetUsage لعرض الرسائل المستخدمة والمتبقية
    - حساب الاستخدام الإضافي والتكلفة
    - _Requirements: 4.6, 5.7_

- [x] 8. تطوير Rollover Manager (مع سياسة الترحيل المشروط)
  - [x] 8.1 تطوير وظيفة حساب الرسائل المتبقية
    - إنشاء ملف internal/rollover/manager.go
    - تطبيق CalculateRemainingMessages عند نهاية الشهر
    - التحقق من rollover_enabled للباقة
    - **ملاحظة:** يتم حفظ الرسائل بحالة 'pending' إذا كانت سياسة الترحيل المشروط مفعّلة
    - _Requirements: 6.2, 6.3_
  
  - [x] 8.2 كتابة Property Test لحساب المتبقي
    - **Property 9: حساب الرسائل المتبقية عند نهاية الشهر**
    - **Validates: Requirements 6.2, 6.3**
    - التحقق من حساب messages_remaining بدقة
  
  - [x] 8.3 تطوير وظيفة ترحيل الرسائل
    - تطبيق RolloverMessages مع احترام rollover_limit
    - تسجيل في جدول rollover_messages
    - تحديد expiry_date حسب rollover_expiry_months
    - **تحديث:** دعم سياسة الترحيل المشروط (حفظ بحالة 'pending' أو 'active' حسب الإعدادات)
    - _Requirements: 6.3, 6.4, 6.5, 6.6, 6.7, 6.8_
  
  - [x] 8.4 كتابة Property Test لاحترام الحد الأقصى
    - **Property 10: احترام الحد الأقصى للترحيل**
    - **Validates: Requirements 6.5, 6.9**
    - التحقق من عدم تجاوز rollover_limit
  
  - [x] 8.5 تطوير وظيفة استخدام الرسائل المرحّلة (FIFO)
    - تطبيق UseRolloverMessages مع FIFO
    - استخدام الرسائل الأقدم أولاً
    - تحديث messages_used و messages_remaining
    - **ملاحظة:** يتم استخدام الرسائل بحالة 'active' فقط
    - _Requirements: 6.12_
  
  - [x] 8.6 كتابة Property Test لـ FIFO
    - **Property 11: استخدام الرسائل المرحّلة بترتيب FIFO**
    - **Validates: Requirements 6.12**
    - التحقق من استخدام الأقدم أولاً
  
  - [x] 8.7 تطوير وظيفة حذف الرسائل المنتهية
    - تطبيق CleanupExpiredRollover
    - Cron Job يومي لحذف الرسائل المنتهية
    - إرسال إشعار للعميل قبل الانتهاء
    - **تحديث:** حذف الرسائل بحالة 'forfeited' أيضاً
    - _Requirements: 6.11, 6.15_
  
  - [x] 8.8 تطوير وظيفة تفعيل الترحيل عند نجاح الدفع (جديد)
    - تطبيق ActivateRolloverOnPaymentSuccess
    - تحديث status من 'pending' إلى 'active'
    - تحديث payment_status إلى 'success'
    - تسجيل payment_verified_at
    - _Requirements: 6.5_
  
  - [x] 8.9 تطوير وظيفة مصادرة الترحيل عند فشل الدفع (جديد)
    - تطبيق ForfeitRolloverOnPaymentFailure
    - تحديث status من 'pending' إلى 'forfeited'
    - تحديث payment_status إلى 'failed'
    - تسجيل forfeited_reason
    - إرسال إشعار للعميل
    - _Requirements: 6.6, 6.15_

- [x] 9. تطوير Package Management APIs
  - [x] 9.1 تطوير API لإنشاء باقة جديدة
    - إنشاء ملف internal/api/package_handler.go
    - تطبيق CreatePackage مع Tenant Validation
    - التحقق من صلاحيات المستخدم (Admin فقط)
    - _Requirements: 3.6, 18.6_
  
  - [x] 9.2 تطوير API لعرض جميع الباقات
    - تطبيق ListPackages مع Tenant Isolation
    - دعم Filtering و Pagination
    - _Requirements: 18.5, 21.1_
  
  - [x] 9.3 تطوير API للاشتراك في باقة
    - إنشاء ملف internal/api/subscription_handler.go
    - تطبيق Subscribe مع Tenant Validation
    - معالجة الأخطاء (رصيد غير كافي، باقة غير نشطة)
    - _Requirements: 4.1, 4.2, 21.3_
  
  - [x] 9.4 تطوير API لعرض الاستخدام
    - تطبيق GetUsage مع Tenant Isolation
    - عرض الرسائل المستخدمة والمتبقية والمرحّلة
    - _Requirements: 4.6, 21.5, 21.6, 21.9_

- [x] 10. Checkpoint - التحقق من الباقات والاشتراكات
  - التأكد من نجاح جميع الاختبارات
  - التحقق من عمل Rollover بشكل صحيح
  - التحقق من عمل Overage بشكل صحيح
  - اختبار APIs مع Postman
  - سؤال المستخدم عن أي استفسارات


## المرحلة 3: الفوترة التلقائية (الأسبوع 5)

- [x] 11. إنشاء جدول الفواتير
  - [x] 11.1 إنشاء جدول billing_invoices مع tenant_id
    - إنشاء Migration لجدول billing_invoices
    - تطبيق RLS Policies
    - إضافة Indexes: idx_invoices_tenant، idx_invoices_status
    - _Requirements: 40.1, 40.2, 40.3_

- [x] 12. تطوير Auto Renewal Service (مع تكامل الترحيل المشروط)
  - [x] 12.1 تطوير Cron Job للفحص اليومي
    - إنشاء ملف internal/billing/auto_renewal.go
    - تطبيق CheckExpiredSubscriptions
    - جدولة يومية باستخدام River Queue
    - _Requirements: 14.1_
  
  - [x] 12.2 تطوير وظيفة التجديد التلقائي
    - تطبيق RenewSubscription مع التحقق من الرصيد
    - خصم قيمة الباقة بشكل ذري
    - تحديث start_date و end_date
    - إعادة تعيين messages_used و messages_remaining
    - **تحديث:** عند نجاح الدفع، استدعاء ActivateRolloverOnPaymentSuccess لتفعيل الرسائل المرحّلة
    - _Requirements: 14.2, 14.3, 14.7, 6.5_
  
  - [x] 12.3 كتابة Property Test للتجديد التلقائي
    - **Property 13: التجديد التلقائي عند نهاية الشهر**
    - **Validates: Requirements 14.2, 14.3**
    - اختبار التجديد مع رصيد كافي
    - اختبار التجديد مع رصيد غير كافي
  
  - [x] 12.4 تطوير وظيفة فترة السماح
    - تطبيق ApplyGracePeriod عند فشل التجديد
    - تحديث حالة الاشتراك إلى 'grace_period'
    - جدولة إعادة المحاولة
    - **تحديث:** الرسائل المرحّلة تبقى بحالة 'pending' خلال فترة السماح
    - _Requirements: 14.4, 14.5_
  
  - [x] 12.5 تطوير وظيفة تعليق الاشتراك
    - تطبيق SuspendSubscription بعد انتهاء فترة السماح
    - تحديث الحالة إلى 'suspended'
    - منع إرسال رسائل جديدة
    - **تحديث:** عند التعليق النهائي، استدعاء ForfeitRolloverOnPaymentFailure لمصادرة الرسائل المرحّلة
    - _Requirements: 14.6, 14.11, 6.6_
  
  - [x] 12.6 تطوير وظيفة إعادة التفعيل التلقائي
    - تطبيق AutoReactivate عند توريد رصيد كافي
    - التحقق من الاشتراكات المعلقة
    - محاولة التجديد تلقائياً
    - **تحديث:** عند نجاح إعادة التفعيل، تفعيل الرسائل المرحّلة إذا كانت لا تزال بحالة 'pending'
    - _Requirements: 14.12, 6.5_

- [x] 13. تطوير Notification Service
  - [x] 13.1 تطوير إشعارات قبل التجديد
    - إنشاء ملف internal/notifications/service.go
    - تطبيق SendRenewalReminder قبل 3 أيام
    - دعم البريد الإلكتروني و SMS
    - _Requirements: 14.8, 17.1_
  
  - [x] 13.2 تطوير إشعارات نجاح التجديد
    - تطبيق SendRenewalSuccess
    - عرض تفاصيل الاشتراك الجديد
    - _Requirements: 14.9, 17.2_
  
  - [x] 13.3 تطوير إشعارات فشل التجديد
    - تطبيق SendRenewalFailure
    - عرض سبب الفشل وفترة السماح
    - _Requirements: 14.10, 17.3_
  
  - [x] 13.4 تطوير إشعارات إضافة الرصيد
    - تطبيق SendBalanceTopup
    - عرض المبلغ المضاف والرصيد الجديد
    - _Requirements: 17.4_
  
  - [x] 13.5 تطوير إشعارات انخفاض الرصيد
    - تطبيق SendLowBalanceWarning
    - تحديد عتبة التحذير (قابلة للتخصيص)
    - _Requirements: 17.5_
  
  - [x] 13.6 تطوير إشعارات اقتراب نفاد الرسائل
    - تطبيق SendLowMessagesWarning
    - التحذير عند الوصول لـ 10% من الرسائل
    - _Requirements: 17.6_

- [x] 14. تطوير Invoice Generator
  - [x] 14.1 تطوير وظيفة توليد الفواتير
    - إنشاء ملف internal/billing/invoice_generator.go
    - تطبيق GenerateInvoice لكل عملية دفع
    - توليد رقم فاتورة فريد
    - حساب الضريبة تلقائياً
    - _Requirements: 40.1, 40.2, 40.3, 40.4, 40.5_
  
  - [x] 14.2 تطوير PDF Generator
    - تكامل مع مكتبة PDF (مثل gofpdf)
    - تصميم قالب الفاتورة
    - إضافة Logo وألوان الشركة
    - _Requirements: 40.6, 40.9, 40.11_
  
  - [x] 14.3 تطوير وظيفة إرسال الفاتورة
    - تطبيق SendInvoice عبر البريد الإلكتروني
    - إرفاق PDF
    - _Requirements: 40.7_
  
  - [x] 14.4 تطوير API لعرض الفواتير
    - تطبيق ListInvoices مع Tenant Isolation
    - دعم Filtering حسب التاريخ والحالة
    - _Requirements: 40.8_

- [x] 15. Checkpoint - التحقق من الفوترة التلقائية
  - التأكد من عمل Auto Renewal بشكل صحيح
  - اختبار فترة السماح والتعليق
  - التحقق من إرسال الإشعارات
  - اختبار توليد الفواتير
  - سؤال المستخدم عن أي استفسارات


## المرحلة 4: البوابات المخصصة (الأسابيع 6-8)

- [x] 16. إنشاء جداول البوابات المخصصة
  - [x] 16.1 إنشاء جدول dedicated_gateways مع tenant_id ونموذج الاشتراك
    - إنشاء Migration لجدول dedicated_gateways
    - تطبيق RLS Policies
    - إضافة حقول: subscription_type، subscription_fee، server_rental_fee
    - إضافة حقول مراقبة الرصيد: telecom_balance، balance_alert_threshold، last_balance_check
    - إزالة حقول: cost_per_message، selling_price (الشركة تدفع مباشرة لشركة الاتصالات)
    - إضافة Indexes: idx_dedicated_gateways_tenant، idx_dedicated_gateways_status
    - تشفير بيانات الاعتماد (telecom_password_encrypted، telecom_api_key_encrypted)
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.10, 8.11, 8.13, 8.14_
  
  - [x] 16.2 إنشاء جدول resource_quotas
    - إنشاء Migration لجدول resource_quotas
    - تحديد الحصص الافتراضية
    - _Requirements: 9.1, 9.2, 9.3, 9.4, 9.5, 9.6, 9.7, 9.8, 9.9_
  
  - [x] 16.3 إنشاء جدول resource_usage
    - إنشاء Migration لجدول resource_usage
    - إضافة Partitioning حسب الشهر للأداء
    - إضافة Indexes: idx_resource_usage_gateway_time
    - _Requirements: 10.1, 10.2, 10.3, 10.4, 10.5, 10.6, 10.7, 10.8, 10.9, 10.10_
  
  - [x] 16.4 إنشاء جدول gateway_users
    - إنشاء Migration لجدول gateway_users
    - دعم الأدوار (admin، operator، viewer)
    - _Requirements: 11.1, 11.2, 11.3_
  
  - [x] 16.5 إنشاء جدول gateway_api_keys
    - إنشاء Migration لجدول gateway_api_keys
    - تشفير API Keys باستخدام Hash آمن
    - دعم IP Whitelisting
    - _Requirements: 12.1, 12.2, 12.3, 12.4, 12.5, 12.6_
  
  - [x] 16.6 إنشاء جدول gateway_statistics
    - إنشاء Migration لجدول gateway_statistics
    - إضافة Unique Constraint على (gateway_id، date)
    - _Requirements: 13.1, 13.2, 13.3, 13.4, 13.5, 13.6, 13.7_

- [x] 17. تطوير Gateway Manager
  - [x] 17.1 تطوير وظيفة إنشاء بوابة مخصصة
    - إنشاء ملف internal/gateway/manager.go
    - تطبيق CreateGateway مع Tenant Validation
    - تشفير بيانات الاعتماد باستخدام AES-256
    - إنشاء Resource Quotas افتراضية
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 26.1_
  
  - [x] 17.2 تطوير وظيفة تحديث إعدادات البوابة
    - تطبيق UpdateGateway مع Tenant Validation
    - دعم تحديث الحدود ورسوم الاشتراك والإيجار
    - دعم تحديث عتبة تنبيه الرصيد
    - _Requirements: 8.7, 8.8, 8.9, 8.10, 8.11, 8.14_
  
  - [x] 17.3 تطوير وظيفة تعليق/إيقاف البوابة
    - تطبيق SuspendGateway و TerminateGateway
    - تحديث الحالة
    - إرسال إشعار للشركة المؤسسية
    - _Requirements: 8.7_
  
  - [x] 17.4 تطوير وظيفة Health Check
    - تطبيق PerformHealthCheck
    - فحص الاتصال بشركة الاتصالات
    - تحديث last_health_check
    - _Requirements: 28.5_

- [x] 18. تطوير Resource Limiter
  - [x] 18.1 تطوير وظيفة فحص الحصص
    - إنشاء ملف internal/resources/limiter.go
    - تطبيق CheckCPUQuota و CheckMemoryQuota
    - تطبيق CheckBandwidthQuota
    - _Requirements: 9.1, 9.2, 9.3_
  
  - [x] 18.2 تطوير Rate Limiter per Gateway
    - تطبيق TenantRateLimiter
    - دعم API Rate Limit، Messages Rate Limit، Queries Rate Limit
    - استخدام Token Bucket Algorithm
    - _Requirements: 9.8, 27.10_
  
  - [x] 18.3 تطوير Worker Pool محدود
    - تطبيق TenantWorkerPool
    - تحديد max_worker_threads لكل بوابة
    - تطبيق Job Queue محدود
    - _Requirements: 9.9_
  
  - [x] 18.4 تطوير Enforcement Policies
    - تطبيق Soft Limit (تحذير فقط)
    - تطبيق Hard Limit (رفض الطلبات)
    - _Requirements: 9.11, 9.12_
  
  - [x] 18.5 كتابة Property Test لـ Rate Limiting per Tenant
    - **Property 20: Rate Limiting per Tenant**
    - **Validates: Requirements 9.8**
    - التحقق من عدم تجاوز الحدود
    - اختبار عزل Rate Limits بين Tenants

- [x] 19. تطوير Monitoring Service
  - [x] 19.1 تطوير Metrics Collector
    - إنشاء ملف internal/monitoring/collector.go
    - جمع مقاييس CPU، Memory، Bandwidth
    - جمع مقاييس Database Connections، Queries/sec
    - جمع مقاييس Redis Memory، Keys، Hit Rate
    - _Requirements: 10.1, 10.2, 10.3, 10.4, 10.5, 10.6_
  
  - [x] 19.2 تطوير Usage Logger
    - تطبيق LogResourceUsage كل دقيقة
    - تسجيل في جدول resource_usage
    - _Requirements: 10.1, 10.2, 10.3_
  
  - [x] 19.3 تطوير Alert Manager
    - تطبيق CheckThresholds
    - إرسال تنبيهات عند الاقتراب من الحدود (90%)
    - إرسال تنبيهات حرجة عند التجاوز
    - _Requirements: 9.10, 17.8, 24.3_
  
  - [x] 19.4 تطوير Statistics Aggregator
    - تطبيق AggregateDaily لحساب الإحصائيات اليومية
    - حساب messages_sent، messages_delivered، messages_failed
    - حساب avg_response_time، max_response_time
    - حساب uptime_percentage
    - _Requirements: 13.2, 13.3, 13.4, 13.5, 13.6_
  
  - [x] 19.5 تطوير Balance Monitor للبوابات المخصصة
    - تطبيق CheckTelecomBalance لفحص رصيد الشركة لدى شركة الاتصالات
    - تطبيق Cron Job لفحص الرصيد دورياً (كل ساعة)
    - إرسال تنبيه عند انخفاض الرصيد عن balance_alert_threshold
    - تحديث telecom_balance و last_balance_check
    - ملاحظة: النظام يراقب الرصيد فقط ولا يعالج أي دفعات
    - _Requirements: 8.13, 8.14, 8.15, 17.8_
  
  - [x] 19.6 تطوير Subscription Billing للبوابات المخصصة
    - تطبيق BillGatewaySubscription لخصم رسوم الاشتراك والإيجار
    - خصم subscription_fee + server_rental_fee شهرياً/سنوياً
    - تسجيل في accounting_transactions
    - إرسال فاتورة للشركة المؤسسية
    - ملاحظة: لا يتم خصم أي تكلفة متعلقة بالرسائل (الشركة تدفع مباشرة لشركة الاتصالات)
    - _Requirements: 8.10, 8.11, 8.15_

- [x] 20. تطوير User Management للبوابات
  - [x] 20.1 تطوير وظيفة إضافة مستخدم
    - إنشاء ملف internal/gateway/user_manager.go
    - تطبيق AddUser مع تحديد الدور
    - تطبيق RBAC (Role-Based Access Control)
    - _Requirements: 11.1, 11.2, 11.3, 11.4_
  
  - [x] 20.2 تطوير وظيفة تخصيص الصلاحيات
    - تطبيق UpdatePermissions
    - دعم صلاحيات دقيقة (Fine-grained Permissions)
    - _Requirements: 11.4_
  
  - [x] 20.3 تطوير Middleware للتحقق من الصلاحيات
    - تطبيق CheckGatewayPermission
    - التحقق من دور المستخدم وصلاحياته
    - _Requirements: 11.7, 11.8, 11.9_

- [x] 21. تطوير API Key Management
  - [x] 21.1 تطوير وظيفة إنشاء API Key
    - إنشاء ملف internal/gateway/api_key_manager.go
    - تطبيق CreateAPIKey مع توليد مفتاح فريد
    - تشفير المفتاح باستخدام Hash آمن (bcrypt)
    - _Requirements: 12.1, 12.2, 12.3_
  
  - [x] 21.2 تطوير وظيفة تحديد الصلاحيات
    - تطبيق SetPermissions للـ API Key
    - تحديد Rate Limit لكل مفتاح
    - _Requirements: 12.4, 12.5_
  
  - [x] 21.3 تطوير IP Whitelisting
    - تطبيق AddAllowedIP و RemoveAllowedIP
    - التحقق من IP عند استخدام المفتاح
    - _Requirements: 12.6, 12.10_
  
  - [x] 21.4 تطوير وظيفة انتهاء الصلاحية
    - تطبيق SetExpiry
    - التحقق من expires_at عند الاستخدام
    - _Requirements: 12.7, 12.11_
  
  - [x] 21.5 تطوير Middleware للتحقق من API Key
    - تطبيق ValidateAPIKey
    - التحقق من الصلاحيات، IP، Expiry
    - تحديث last_used_at و usage_count
    - _Requirements: 12.8, 12.10, 12.11_

- [x] 22. Checkpoint - التحقق من البوابات المخصصة
  - التأكد من عمل Resource Quotas بشكل صحيح
  - اختبار Rate Limiting per Gateway
  - التحقق من عمل Monitoring
  - اختبار User Management و API Keys
  - سؤال المستخدم عن أي استفسارات


## المرحلة 5: الواجهات (الأسابيع 9-10)

**ملاحظة مهمة:** جميع التعديلات في هذه المرحلة هي تعديلات طفيفة على الواجهات الموجودة. لا يُسمح بإعادة تصميم الواجهات من الصفر.

- [-] 23. تعديلات واجهة الإدارة (apps/admin-web)
  - [x] 23.1 إضافة قسم التعرفات في Dashboard
    - تعديل ملف apps/admin-web/src/pages/Dashboard.tsx
    - إضافة import لمكون TariffSummary
    - إضافة المكون في الصفحة الموجودة
    - _Requirements: 18.1_
  
  - [x] 23.2 إنشاء مكون TariffSummary (بسيط)
    - إنشاء ملف apps/admin-web/src/components/TariffSummary.tsx
    - عرض ملخص التعرفات النشطة
    - استخدام useTariffs hook
    - _Requirements: 18.1_
  
  - [x] 23.3 إضافة تبويب التعرفات في Settings
    - تعديل ملف apps/admin-web/src/pages/Settings.tsx
    - إضافة Tab جديد للتعرفات
    - إضافة Tab جديد للباقات
    - _Requirements: 18.2, 18.5_
  
  - [x] 23.4 إنشاء مكون TariffSettings (بسيط)
    - إنشاء ملف apps/admin-web/src/components/TariffSettings.tsx
    - عرض جدول التعرفات مع إمكانية التعديل
    - نموذج لإضافة تعرفة جديدة
    - _Requirements: 18.2, 18.3, 18.4_
  
  - [x] 23.5 إنشاء مكون PackageSettings (بسيط)
    - إنشاء ملف apps/admin-web/src/components/PackageSettings.tsx
    - عرض جدول الباقات مع إمكانية التعديل
    - نموذج لإنشاء باقة جديدة
    - عرض عدد المشتركين لكل باقة
    - **تحديث:** إضافة زر تحكم لتفعيل/تعطيل سياسة الترحيل المشروط (Conditional Rollover Policy)
    - **تحديث:** إضافة شرح توضيحي للسياسة: "عند التفعيل، يتم ترحيل الرسائل فقط عند نجاح الدفع التلقائي"
    - _Requirements: 18.5, 18.6, 18.7, 18.8, 6.2_
  
  - [x] 23.6 إضافة صفحة إدارة البوابات المخصصة
    - إنشاء ملف apps/admin-web/src/pages/Gateways.tsx (صفحة جديدة)
    - عرض جدول البوابات مع الحالة والإحصائيات
    - نموذج لإنشاء بوابة جديدة
    - عرض التنبيهات النشطة
    - _Requirements: 19.1, 19.2, 19.3, 19.4, 19.5, 19.10, 19.11_
  
  - [x] 23.7 إنشاء مكون GatewayDetails (بسيط)
    - إنشاء ملف apps/admin-web/src/components/GatewayDetails.tsx
    - عرض تفاصيل البوابة
    - إدارة الحصص الموردية
    - عرض الاستخدام الحالي
    - _Requirements: 19.6, 19.7, 19.8_
  
  - [x] 23.8 إضافة صفحة التقارير المالية
    - تعديل ملف apps/admin-web/src/pages/Reports.tsx (إن وجد)
    - إضافة تقارير الإيرادات والتكاليف
    - إضافة رسوم بيانية
    - دعم تصدير التقارير (PDF، Excel، CSV)
    - _Requirements: 22.1, 22.2, 22.3, 22.4, 22.11, 22.12_

- [x] 24. تعديلات واجهة العميل (apps/client-web)
  - [x] 24.1 إضافة معلومات الباقة في Dashboard
    - تعديل ملف apps/client-web/src/pages/Dashboard.tsx
    - إضافة import لمكونات SubscriptionInfo و MessageUsage
    - إضافة المكونات في الصفحة الموجودة
    - _Requirements: 21.4, 21.5, 21.6_
  
  - [x] 24.2 إنشاء مكون SubscriptionInfo (بسيط)
    - إنشاء ملف apps/client-web/src/components/SubscriptionInfo.tsx
    - عرض الباقة الحالية والسعر
    - عرض تاريخ التجديد
    - عرض الرصيد الحالي
    - _Requirements: 21.4, 21.6, 21.7_
  
  - [x] 24.3 إنشاء مكون MessageUsage (بسيط)
    - إنشاء ملف apps/client-web/src/components/MessageUsage.tsx
    - عرض الرسائل المستخدمة والمتبقية
    - عرض الرسائل المرحّلة مع تواريخ الانتهاء
    - عرض الاستخدام الإضافي والتكلفة
    - _Requirements: 21.5, 21.8, 21.9_
  
  - [x] 24.4 تعديل صفحة الرسائل لإضافة حساب التكلفة
    - تعديل ملف apps/client-web/src/pages/Messages.tsx
    - إضافة useEffect لحساب التكلفة عند تغيير النص
    - عرض التكلفة المتوقعة وعدد الأجزاء
    - عرض تحذير إذا كان الرصيد غير كافي
    - _Requirements: 21.12, 21.13_
  
  - [x] 24.5 إضافة صفحة الباقات
    - إنشاء ملف apps/client-web/src/pages/Packages.tsx (صفحة جديدة)
    - عرض جميع الباقات المتاحة
    - عرض تفاصيل كل باقة (السعر، الرسائل، المزايا)
    - زر الاشتراك لكل باقة
    - _Requirements: 21.1, 21.2, 21.3_
  
  - [x] 24.6 إضافة صفحة السجل والتقارير
    - تعديل ملف apps/client-web/src/pages/History.tsx (إن وجد)
    - عرض سجل الرسائل المرسلة
    - عرض سجل المعاملات المالية
    - دعم Filtering و Pagination
    - _Requirements: 21.14, 21.15_

- [x] 25. إضافة تبويب الشركات المؤسسية في واجهة الإدارة
  - [x] 25.1 إضافة تبويب Enterprise في TenantManager
    - تعديل ملف apps/admin-web/src/components/TenantManager.jsx
    - إضافة تبويب "Enterprise Gateway" في قسم التفاصيل
    - عرض التبويب فقط للشركات التي لديها بوابة مخصصة
    - _Requirements: 20.1_
  
  - [x] 25.2 إنشاء مكون EnterpriseGatewayDashboard
    - إنشاء ملف apps/admin-web/src/components/EnterpriseGatewayDashboard.jsx
    - عرض حالة البوابة ووقت التشغيل
    - عرض إحصائيات اليوم (رسائل مرسلة، معدل النجاح، وقت الاستجابة)
    - عرض الاستخدام الشهري مقابل الحدود (CPU، Memory، Bandwidth)
    - عرض التنبيهات النشطة
    - عرض رصيد الشركة لدى شركة الاتصالات مع تنبيه إذا كان منخفضاً
    - _Requirements: 20.1, 20.2, 20.3, 20.4, 20.5_
  
  - [x] 25.3 إنشاء مكون EnterpriseGatewaySettings
    - إنشاء ملف apps/admin-web/src/components/EnterpriseGatewaySettings.jsx
    - عرض إعدادات البوابة (معلومات الاتصال، حدود الموارد)
    - تعديل حدود الاستخدام (CPU، Memory، Bandwidth، Rate Limits)
    - تعديل رسوم الاشتراك والإيجار
    - تعديل عتبة تنبيه الرصيد
    - _Requirements: 20.6, 20.7_
  
  - [x] 25.4 إنشاء مكون EnterpriseUsers
    - إنشاء ملف apps/admin-web/src/components/EnterpriseUsers.jsx
    - عرض جدول مستخدمي البوابة
    - إضافة/تعديل/حذف مستخدمين
    - تحديد الأدوار (admin، operator، viewer)
    - تحديد الصلاحيات الدقيقة
    - _Requirements: 20.8_
  
  - [x] 25.5 إنشاء مكون EnterpriseAPIKeys
    - إنشاء ملف apps/admin-web/src/components/EnterpriseAPIKeys.jsx
    - عرض جدول API Keys للبوابة
    - إنشاء/تعطيل/حذف مفاتيح
    - تحديد الصلاحيات لكل مفتاح
    - إدارة IP Whitelisting
    - تحديد تاريخ انتهاء الصلاحية
    - _Requirements: 20.9, 20.10_
  
  - [x] 25.6 إنشاء مكون EnterpriseReports
    - إنشاء ملف apps/admin-web/src/components/EnterpriseReports.jsx
    - عرض تقارير الأداء (وقت التشغيل، معدل النجاح، وقت الاستجابة)
    - عرض تقارير الاستخدام (CPU، Memory، Bandwidth، عدد الرسائل)
    - رسوم بيانية للاتجاهات
    - دعم تصدير التقارير (PDF، Excel، CSV)
    - _Requirements: 20.12, 20.13_
  
  - [x] 25.7 إنشاء مكون EnterpriseAPIDocs
    - إنشاء ملف apps/admin-web/src/components/EnterpriseAPIDocs.jsx
    - عرض وثائق API كاملة مع أمثلة
    - عرض أمثلة كود بلغات مختلفة (cURL، JavaScript، Python، PHP)
    - دعم تجربة APIs مباشرة (API Playground)
    - _Requirements: 20.11, 30.12_
  
  - [x] 25.8 تحديث TenantManager لدعم التبويبات الجديدة
    - تعديل ملف apps/admin-web/src/components/TenantManager.jsx
    - إضافة تبويب "Enterprise Gateway" بجانب التبويبات الموجودة
    - عرض التبويب فقط إذا كان للشركة بوابة مخصصة (has_dedicated_gateway = true)
    - تحميل المكونات الجديدة حسب التبويب النشط
    - _Requirements: 20.1_

- [x] 26. تطوير API Hooks و Services
  - [x] 26.1 تطوير Tariff Hooks
    - إنشاء ملف apps/admin-web/src/hooks/useTariffs.ts
    - تطبيق useTariffs، useCreateTariff، useUpdateTariff
    - _Requirements: 18.1, 18.2, 18.3_
  
  - [x] 26.2 تطوير Package Hooks
    - إنشاء ملف apps/client-web/src/hooks/usePackages.ts
    - تطبيق usePackages، useSubscribe، useSubscription
    - _Requirements: 21.1, 21.3, 21.4_
  
  - [x] 26.3 تطوير Gateway Hooks
    - إنشاء ملف apps/admin-web/src/hooks/useGateways.ts
    - تطبيق useGateways، useCreateGateway، useGatewayStats
    - _Requirements: 19.1, 19.2, 19.9_
  
  - [x] 26.4 تحديث Message Service لحساب التكلفة
    - تعديل ملف apps/client-web/src/api/messages.ts
    - إضافة calculateCost function
    - تحديث sendMessage لتضمين estimated_cost
    - _Requirements: 21.12_

- [x] 27. Checkpoint - التحقق من الواجهات
  - اختبار جميع الواجهات على أجهزة مختلفة
  - التحقق من Responsive Design
  - اختبار جميع APIs من الواجهات
  - مراجعة UX/UI
  - سؤال المستخدم عن أي استفسارات


## المرحلة 6: المزايا المتقدمة (الأسابيع 11-12)

- [x] 28. تطوير نظام الخصومات والعروض
  - [x] 28.1 إنشاء جدول promo_codes
    - إنشاء Migration لجدول promo_codes مع tenant_id
    - تطبيق RLS Policies
    - دعم خصومات بنسبة مئوية أو مبلغ ثابت
    - _Requirements: 32.1, 32.2_
  
  - [x] 28.2 تطوير Promo Code Manager
    - إنشاء ملف internal/promo/manager.go
    - تطبيق CreatePromoCode مع تحديد الشروط
    - تطبيق ValidatePromoCode للتحقق من الصلاحية
    - تطبيق ApplyPromoCode لتطبيق الخصم
    - _Requirements: 32.3, 32.4, 32.5, 32.6, 32.7, 32.8_
  
  - [x] 28.3 تطوير Usage Tracker
    - تسجيل استخدام كل كود خصم
    - منع الاستخدام المتكرر لنفس العميل
    - _Requirements: 32.9, 32.10_
  
  - [x] 28.4 تطوير API لإدارة الخصومات
    - تطبيق CreatePromoCode، ListPromoCodes، UpdatePromoCode
    - تطبيق ApplyPromoCode في Subscription API
    - _Requirements: 32.1, 32.8_
  
  - [x] 28.5 إضافة تقرير أداء العروض
    - تطبيق GetPromoPerformance
    - عرض عدد الاستخدامات والإيرادات
    - _Requirements: 32.11_

- [x] 29. تطوير نظام الإحالة (Referral System)
  - [x] 29.1 إنشاء جدول referrals
    - إنشاء Migration لجدول referrals مع tenant_id
    - تطبيق RLS Policies
    - تسجيل العلاقة بين المُحيل والمُحال
    - _Requirements: 33.1, 33.3_
  
  - [x] 29.2 تطوير Referral Manager
    - إنشاء ملف internal/referral/manager.go
    - تطبيق GenerateReferralCode لكل عميل
    - تطبيق ProcessReferral عند اشتراك عميل جديد
    - تطبيق GrantReward لمنح المكافأة
    - _Requirements: 33.1, 33.2, 33.3, 33.4, 33.5, 33.6_
  
  - [x] 29.3 تطوير API للإحالة
    - تطبيق GetReferralCode، GetReferralStats
    - تكامل مع Subscription API
    - _Requirements: 33.2, 33.8_
  
  - [x] 29.4 إضافة تقرير أداء الإحالة
    - تطبيق GetReferralPerformance
    - عرض عدد الإحالات والمكافآت
    - _Requirements: 33.9_

- [x] 30. التكامل مع بوابات الدفع
  - [x] 30.1 إنشاء جدول payment_transactions
    - إنشاء Migration لجدول payment_transactions مع tenant_id
    - تطبيق RLS Policies
    - تسجيل جميع عمليات الدفع
    - _Requirements: 39.8, 39.9_
  
  - [x] 30.2 تطوير Stripe Integration
    - إنشاء ملف internal/payment/stripe.go
    - تطبيق CreatePaymentIntent
    - تطبيق HandleWebhook لمعالجة الأحداث
    - تطبيق ProcessRefund
    - _Requirements: 39.2, 39.4, 39.5, 39.6, 39.10_
  
  - [x] 30.3 تطوير PayPal Integration
    - إنشاء ملف internal/payment/paypal.go
    - تطبيق CreateOrder
    - تطبيق CapturePayment
    - _Requirements: 39.2_
  
  - [x] 30.4 تطوير Payment Manager
    - إنشاء ملف internal/payment/manager.go
    - تطبيق ProcessPayment مع دعم بوابات متعددة
    - تطبيق AutoTopup عند نجاح الدفع
    - إرسال إشعار للعميل
    - _Requirements: 39.3, 39.5, 39.6, 39.7_
  
  - [x] 30.5 تطوير API للدفع
    - تطبيق InitiatePayment، GetPaymentStatus
    - تطبيق Webhook Endpoints
    - _Requirements: 39.4, 39.5_
  
  - [x] 30.6 تطبيق PCI DSS Compliance
    - تشفير جميع بيانات الدفع
    - عدم تخزين بيانات البطاقات
    - استخدام Tokenization
    - _Requirements: 39.11, 39.12_

- [x] 31. دعم اللغات المتعددة (i18n)
  - [x] 31.1 إعداد i18n في Frontend
    - تثبيت مكتبة i18next
    - إنشاء ملفات الترجمة (ar.json، en.json)
    - تكوين i18n في التطبيق
    - _Requirements: 34.1, 34.2, 34.10_
  
  - [x] 31.2 ترجمة واجهة الإدارة
    - ترجمة جميع النصوص في apps/admin-web
    - ترجمة رسائل الخطأ
    - _Requirements: 34.5, 34.7, 34.8_
  
  - [x] 31.3 ترجمة واجهة العميل
    - ترجمة جميع النصوص في apps/client-web
    - ترجمة رسائل الخطأ والإشعارات
    - _Requirements: 34.5, 34.7, 34.8_
  
  - [x] 31.4 تطبيق RTL Support
    - إضافة دعم اتجاه النص من اليمين لليسار
    - تعديل CSS للدعم RTL
    - _Requirements: 34.6_
  
  - [x] 31.5 تطوير Language Selector
    - إضافة مكون لاختيار اللغة
    - حفظ تفضيل اللغة للمستخدم
    - _Requirements: 34.3, 34.4_

- [x] 32. تحسينات الأداء والتوافق
  - [x] 32.1 تطبيق Responsive Design
    - التأكد من عمل جميع الواجهات على الهواتف
    - التأكد من عمل جميع الواجهات على الأجهزة اللوحية
    - _Requirements: 35.1, 35.2, 35.3, 35.4_
  
  - [x] 32.2 اختبار المتصفحات المختلفة
    - اختبار على Chrome، Firefox، Safari، Edge
    - إصلاح مشاكل التوافق
    - _Requirements: 35.5_
  
  - [x] 32.3 تحسين الأداء على الاتصالات البطيئة
    - تطبيق Code Splitting
    - تطبيق Lazy Loading
    - تحسين حجم الصور
    - _Requirements: 35.9_
  
  - [x] 32.4 تطبيق Progressive Web App (PWA)
    - إضافة Service Worker
    - إضافة Manifest
    - تطبيق Offline Support
    - _Requirements: 35.10_

- [x] 33. Checkpoint - التحقق من المزايا المتقدمة
  - اختبار نظام الخصومات والإحالة
  - اختبار التكامل مع بوابات الدفع
  - التحقق من دعم اللغات المتعددة
  - اختبار الأداء على أجهزة مختلفة
  - سؤال المستخدم عن أي استفسارات


## المرحلة 7: الاستقرار والتحسين (الأسابيع 13-14)

- [x] 34. اختبارات شاملة
  - [x] 34.1 Integration Tests للنظام الكامل
    - إنشاء ملف internal/integration/tariff_system_test.go
    - اختبار تدفق كامل: الاشتراك → الإرسال → الخصم → التجديد
    - اختبار تدفق Overage
    - اختبار تدفق Rollover
    - _Requirements: جميع المتطلبات_
  
  - [x] 34.2 Property-Based Tests الشاملة
    - اختبار جميع الـ Properties المحددة في التصميم
    - Property 1-20 (التعرفات، الباقات، Rollover، العزل)
    - استخدام rapid/gopter للـ Property Testing
  
  - [x] 34.3 Load Tests
    - إنشاء ملف performance-tests/tariff-load-test.js
    - اختبار 1000 اشتراك متزامن
    - اختبار 10000 عملية خصم متزامنة
    - اختبار 100 بوابة مخصصة متزامنة
    - _Requirements: 27.1, 27.2, 27.3_
  
  - [x] 34.4 Stress Tests
    - اختبار النظام تحت ضغط عالي
    - اختبار حدود Resource Quotas
    - اختبار Rate Limiting
    - _Requirements: 27.1, 27.2_
  
  - [x] 34.5 Security Tests
    - اختبار Cross-Tenant Access Prevention
    - اختبار SQL Injection
    - اختبار XSS
    - اختبار CSRF
    - اختبار API Authentication
    - _Requirements: 26.1, 26.2, 26.3, 26.7, 26.8_
  
  - [x] 34.6 Concurrent Tests للعزل
    - اختبار عمليات متزامنة من tenants مختلفة
    - التحقق من عدم تأثير tenant على آخر
    - التحقق من عدم تسرب البيانات
    - _Requirements: 26.5_

- [x] 35. تحسين الأداء
  - [x] 35.1 Database Optimization
    - مراجعة جميع الاستعلامات
    - إضافة Indexes المفقودة
    - تحسين الاستعلامات البطيئة
    - تطبيق Query Caching
    - _Requirements: 27.10_
  
  - [x] 35.2 Redis Optimization
    - مراجعة استراتيجية الـ Caching
    - تحسين TTL للمفاتيح المختلفة
    - تطبيق Cache Warming
    - _Requirements: 27.9_
  
  - [x] 35.3 Connection Pooling
    - تحسين إعدادات Database Connection Pool
    - تحسين إعدادات Redis Connection Pool
    - _Requirements: 27.7_
  
  - [x] 35.4 Query Optimization
    - استخدام EXPLAIN ANALYZE لتحليل الاستعلامات
    - إعادة كتابة الاستعلامات البطيئة
    - تطبيق Prepared Statements
    - _Requirements: 27.10_
  
  - [x] 35.5 Monitoring Setup
    - إعداد Prometheus لجمع المقاييس
    - إعداد Grafana لعرض المقاييس
    - إنشاء Dashboards للمراقبة
    - _Requirements: 29.1, 29.9, 29.10, 29.12_
  
  - [x] 35.6 Alerting Setup
    - إعداد Alert Rules في Prometheus
    - تكامل مع Slack/Email للتنبيهات
    - _Requirements: 24.1, 24.2, 24.3, 24.4, 24.5, 24.6, 24.7, 24.9_

- [x] 36. التوثيق الشامل
  - [x] 36.1 API Documentation
    - إنشاء ملف docs/API_REFERENCE_TARIFF.md
    - توثيق جميع APIs الجديدة
    - إضافة أمثلة لكل API
    - إنشاء Postman Collection
    - _Requirements: 30.12, 30.13_
  
  - [x] 36.2 User Guide للعملاء
    - إنشاء ملف docs/USER_GUIDE_TARIFF.md
    - شرح كيفية الاشتراك في الباقات
    - شرح نظام Rollover
    - شرح نظام Overage
    - شرح كيفية شحن الرصيد
    - _Requirements: 37.10, 37.11_
  
  - [x] 36.3 Admin Guide
    - إنشاء ملف docs/ADMIN_GUIDE_TARIFF.md
    - شرح كيفية إدارة التعرفات
    - شرح كيفية إدارة الباقات
    - شرح كيفية إدارة البوابات المخصصة
    - شرح كيفية التوريد المباشر
    - _Requirements: 37.10, 37.11_
  
  - [x] 36.4 Enterprise Guide
    - إنشاء ملف docs/ENTERPRISE_GUIDE.md
    - شرح كيفية إدارة البوابة المخصصة
    - شرح كيفية إدارة المستخدمين
    - شرح كيفية إدارة API Keys
    - شرح كيفية قراءة التقارير
    - _Requirements: 37.10, 37.11_
  
  - [x] 36.5 Developer Guide
    - إنشاء ملف docs/DEVELOPER_GUIDE_TARIFF.md
    - شرح معمارية النظام
    - شرح كيفية إضافة تعرفات جديدة
    - شرح كيفية إضافة باقات جديدة
    - شرح Multi-Tenancy Isolation
    - _Requirements: 37.10, 37.11_
  
  - [x] 36.6 Database Schema Documentation
    - تحديث ملف docs/DATABASE_SCHEMA.md
    - توثيق جميع الجداول الجديدة
    - توثيق العلاقات بين الجداول
    - توثيق RLS Policies
    - _Requirements: 37.10_

- [x] 37. النسخ الاحتياطي والاسترجاع
  - [x] 37.1 تطوير Backup Strategy
    - إنشاء ملف scripts/backup-tariff-data.sh
    - نسخ احتياطي يومي لجداول التعرفات والباقات
    - نسخ احتياطي يومي لجداول الاشتراكات
    - نسخ احتياطي يومي لجداول البوابات المخصصة
    - _Requirements: 25.1, 25.2_
  
  - [x] 37.2 تطوير Retention Policy
    - الاحتفاظ بالنسخ اليومية لمدة 30 يوم
    - الاحتفاظ بالنسخ الأسبوعية لمدة 3 أشهر
    - الاحتفاظ بالنسخ الشهرية لمدة سنة
    - _Requirements: 25.2, 25.3, 25.4_
  
  - [x] 37.3 تطوير Backup Encryption
    - تشفير جميع النسخ الاحتياطية
    - تخزين المفاتيح بشكل آمن
    - _Requirements: 25.6_
  
  - [x] 37.4 تطوير Backup Storage
    - تخزين النسخ في S3 أو مكافئ
    - تطبيق Versioning
    - _Requirements: 25.5_
  
  - [x] 37.5 تطوير Backup Validation
    - اختبار صحة النسخ الاحتياطية تلقائياً
    - إرسال تنبيه عند فشل النسخ
    - _Requirements: 25.7, 25.10_
  
  - [x] 37.6 تطوير Recovery Procedures
    - إنشاء ملف scripts/restore-tariff-data.sh
    - توثيق خطوات الاسترجاع
    - اختبار الاسترجاع دورياً
    - _Requirements: 25.8, 25.9, 25.11_
  
  - [x] 37.7 إنشاء Disaster Recovery Plan
    - إنشاء ملف docs/DISASTER_RECOVERY_PLAN.md
    - توثيق السيناريوهات المختلفة
    - توثيق خطوات الاسترجاع لكل سيناريو
    - تحديد RTO و RPO
    - _Requirements: 25.11, 28.8, 28.9_

- [x] 38. الامتثال والتوافق القانوني
  - [x] 38.1 إنشاء صفحة شروط الخدمة
    - إنشاء ملف apps/client-web/src/pages/TermsOfService.tsx
    - كتابة شروط الخدمة
    - _Requirements: 36.1_
  
  - [x] 38.2 إنشاء صفحة سياسة الخصوصية
    - إنشاء ملف apps/client-web/src/pages/PrivacyPolicy.tsx
    - كتابة سياسة الخصوصية
    - _Requirements: 36.2_
  
  - [x] 38.3 تطوير Consent Management
    - تطبيق طلب الموافقة عند التسجيل
    - تسجيل الموافقة مع التاريخ
    - _Requirements: 36.3, 36.4_
  
  - [x] 38.4 تطوير Data Deletion
    - تطبيق DeleteAccount API
    - حذف جميع البيانات الشخصية
    - الاحتفاظ بالبيانات المالية للمراجعة
    - _Requirements: 36.5, 36.6, 36.7_
  
  - [x] 38.5 تطوير Data Export
    - تطبيق ExportData API
    - تصدير جميع بيانات المستخدم
    - _Requirements: 36.8_
  
  - [x] 38.6 تطبيق Fraud Prevention
    - تطبيق قواعد كشف الاحتيال
    - تسجيل الأنشطة المشبوهة
    - _Requirements: 36.9_
  
  - [x] 38.7 تطبيق Audit Logging
    - تسجيل جميع العمليات الحساسة
    - الاحتفاظ بالسجلات للمراجعة
    - _Requirements: 36.10, 36.11_

- [x] 39. نظام الدعم الفني
  - [x] 39.1 إنشاء جدول support_tickets
    - إنشاء Migration لجدول support_tickets مع tenant_id
    - تطبيق RLS Policies
    - دعم الأولويات والأنواع المختلفة
    - _Requirements: 37.1, 37.2, 37.3, 37.4_
  
  - [x] 39.2 تطوير Ticket Manager
    - إنشاء ملف internal/support/ticket_manager.go
    - تطبيق CreateTicket، UpdateTicket، CloseTicket
    - تطبيق AddComment، AddAttachment
    - _Requirements: 37.2, 37.6, 37.7_
  
  - [x] 39.3 تطوير Notification System للتذاكر
    - إرسال إشعار عند إنشاء التذكرة
    - إرسال إشعار عند الرد
    - _Requirements: 37.5, 37.8_
  
  - [x] 39.4 تطوير SLA Tracking
    - تسجيل وقت الاستجابة ووقت الحل
    - تنبيهات عند تجاوز SLA
    - _Requirements: 37.9_
  
  - [x] 39.5 إنشاء Knowledge Base
    - إنشاء جدول knowledge_base
    - إضافة مقالات للأسئلة الشائعة
    - _Requirements: 37.10_
  
  - [x] 39.6 تطوير Priority Support للشركات
    - تطبيق أولوية أعلى لتذاكر الشركات المؤسسية
    - SLA أقصر للشركات
    - _Requirements: 37.12_

- [x] 40. اختبار الرسائل (Test Mode)
  - [x] 40.1 تطوير Test Mode Environment
    - إنشاء ملف internal/testmode/manager.go
    - تطبيق IsTestMode للتحقق من الوضع
    - تطبيق MockSendMessage
    - _Requirements: 38.1, 38.2, 38.3, 38.4_
  
  - [x] 40.2 تطوير Test Phone Numbers
    - تحديد أرقام اختبار خاصة
    - إرجاع حالات مختلفة (نجاح، فشل، معلق)
    - _Requirements: 38.5, 38.6_
  
  - [x] 40.3 تطوير Test API Keys
    - توليد API Keys منفصلة لوضع الاختبار
    - التمييز الواضح بين الوضعين
    - _Requirements: 38.7, 38.8, 38.9_
  
  - [x] 40.4 تطوير Test Logging
    - تسجيل جميع عمليات الاختبار في سجل منفصل
    - عدم التأثير على الإحصائيات الحقيقية
    - _Requirements: 38.10_

- [x] 41. Checkpoint النهائي - التحقق من الاستقرار
  - التأكد من نجاح جميع الاختبارات
  - التحقق من الأداء تحت الضغط
  - مراجعة جميع التوثيق
  - اختبار النسخ الاحتياطي والاسترجاع
  - مراجعة الأمان والامتثال
  - سؤال المستخدم عن أي استفسارات نهائية


---

## ملاحظات التنفيذ

### الأولويات

1. **المرحلة 1 (حرجة)**: معالجة المشاكل الحرجة في النظام الحالي أولاً
2. **المرحلة 2-3**: بناء الوظائف الأساسية (الباقات والفوترة)
3. **المرحلة 4**: البوابات المخصصة (للشركات الكبيرة)
4. **المرحلة 5**: الواجهات (تعديلات طفيفة فقط)
5. **المرحلة 6**: المزايا المتقدمة (اختيارية)
6. **المرحلة 7**: الاستقرار والتحسين (ضرورية)

### المهام الاختيارية (Optional)

جميع المهام المميزة بـ `*` هي اختيارية ويمكن تخطيها لتسريع التطوير:
- Property-Based Tests (مهمة لكن يمكن إضافتها لاحقاً)
- Unit Tests (يمكن الاكتفاء بـ Integration Tests)

### المهام الإلزامية (Required)

جميع المهام الأخرى إلزامية ويجب تنفيذها لضمان:
- معالجة المشاكل الحرجة
- العزل التام بين الشركات
- الأمان المالي
- الموثوقية

### التبعيات

- **المرحلة 2** تعتمد على **المرحلة 1** (Balance Manager و Tariff Engine)
- **المرحلة 3** تعتمد على **المرحلة 2** (Subscription Manager)
- **المرحلة 4** مستقلة نسبياً لكن تحتاج **المرحلة 1** (Tenant Isolation)
- **المرحلة 5** تعتمد على **المراحل 1-4** (جميع APIs)
- **المرحلة 6** مستقلة نسبياً
- **المرحلة 7** تعتمد على جميع المراحل السابقة

### نقاط التحقق (Checkpoints)

يوجد 7 نقاط تحقق (Checkpoint) في نهاية كل مرحلة:
1. بعد المرحلة 1: التحقق من الأساسيات
2. بعد المرحلة 2: التحقق من الباقات والاشتراكات
3. بعد المرحلة 3: التحقق من الفوترة التلقائية
4. بعد المرحلة 4: التحقق من البوابات المخصصة
5. بعد المرحلة 5: التحقق من الواجهات
6. بعد المرحلة 6: التحقق من المزايا المتقدمة
7. بعد المرحلة 7: Checkpoint النهائي

في كل نقطة تحقق:
- التأكد من نجاح جميع الاختبارات
- مراجعة الكود
- اختبار الوظائف الجديدة
- سؤال المستخدم عن أي استفسارات

### استراتيجية الاختبار

#### Property-Based Testing
- استخدام مكتبة `rapid` أو `gopter` في Go
- اختبار الخصائص العامة (Properties) بدلاً من الحالات المحددة
- التركيز على:
  - Atomic Operations
  - Idempotency
  - Tenant Isolation
  - Business Rules

#### Integration Testing
- اختبار التدفقات الكاملة
- استخدام قاعدة بيانات اختبار منفصلة
- تنظيف البيانات بعد كل اختبار

#### Load Testing
- استخدام k6 للـ Load Tests
- اختبار 1000+ عملية متزامنة
- قياس الأداء تحت الضغط

#### Security Testing
- اختبار Cross-Tenant Access Prevention
- اختبار SQL Injection، XSS، CSRF
- اختبار Authentication و Authorization

### معايير النجاح

#### للمرحلة 1
- ✅ لا توجد Race Conditions في Balance Operations
- ✅ جميع العمليات المالية Atomic
- ✅ Audit Trail كامل لجميع التغييرات
- ✅ نظام تعرفات ديناميكي يعمل بشكل صحيح

#### للمرحلة 2
- ✅ نظام باقات شهرية كامل
- ✅ نظام Rollover يعمل بشكل صحيح
- ✅ نظام Overage يعمل بشكل صحيح
- ✅ جميع APIs تعمل بشكل صحيح

#### للمرحلة 3
- ✅ الفوترة التلقائية تعمل بشكل موثوق
- ✅ الإشعارات تُرسل في الوقت المناسب
- ✅ الفواتير تُولّد تلقائياً

#### للمرحلة 4
- ✅ البوابات المخصصة معزولة تماماً
- ✅ Resource Quotas تُطبّق بشكل صحيح
- ✅ المراقبة تعمل بشكل مستمر
- ✅ التنبيهات تُرسل عند الحاجة

#### للمرحلة 5
- ✅ جميع الواجهات تعمل بشكل صحيح
- ✅ Responsive Design على جميع الأجهزة
- ✅ تجربة مستخدم ممتازة

#### للمرحلة 6
- ✅ نظام الخصومات يعمل بشكل صحيح
- ✅ نظام الإحالة يعمل بشكل صحيح
- ✅ التكامل مع بوابات الدفع يعمل بشكل آمن
- ✅ دعم اللغات المتعددة يعمل بشكل كامل

#### للمرحلة 7
- ✅ جميع الاختبارات تنجح
- ✅ الأداء ممتاز تحت الضغط
- ✅ التوثيق شامل وواضح
- ✅ النسخ الاحتياطي والاسترجاع يعملان بشكل موثوق
- ✅ الامتثال للمتطلبات القانونية

### الموارد المطلوبة

#### للتطوير
- Go 1.21+
- PostgreSQL 15+
- Redis 7+
- Node.js 18+ (للواجهات)
- Docker و Docker Compose

#### للاختبار
- k6 (Load Testing)
- Postman (API Testing)
- rapid/gopter (Property Testing)

#### للمراقبة
- Prometheus
- Grafana
- AlertManager

### التقديرات الزمنية

- **المرحلة 1**: أسبوعان (حرجة)
- **المرحلة 2**: أسبوعان
- **المرحلة 3**: أسبوع واحد
- **المرحلة 4**: 3 أسابيع (معقدة)
- **المرحلة 5**: أسبوعان
- **المرحلة 6**: أسبوعان
- **المرحلة 7**: أسبوعان

**الإجمالي**: 14 أسبوع (حوالي 3.5 شهر)

### المخاطر المحتملة

#### مخاطر تقنية
1. **صعوبة تطبيق RLS**: قد يكون تطبيق Row Level Security معقداً
   - **الحل**: اختبار شامل، مراجعة دقيقة للـ Policies
   
2. **مشاكل الأداء**: قد يؤثر RLS على الأداء
   - **الحل**: Indexes مناسبة، Caching، Query Optimization

3. **تعقيد Worker Pool Isolation**: صعوبة عزل Worker Pools
   - **الحل**: استخدام River Queue مع Tenant-specific Queues

#### مخاطر الأعمال
1. **تغيير المتطلبات**: قد تتغير المتطلبات أثناء التطوير
   - **الحل**: Checkpoints منتظمة، مرونة في التصميم

2. **تأخير التسليم**: قد يتأخر التسليم بسبب التعقيد
   - **الحل**: تقسيم العمل لمراحل، تسليم تدريجي

### التوصيات

1. **ابدأ بالمرحلة 1 فوراً**: معالجة المشاكل الحرجة أولوية قصوى
2. **لا تتخطى الاختبارات**: الاختبارات ضرورية للموثوقية
3. **راجع الكود بانتظام**: مراجعة الكود تقلل الأخطاء
4. **استخدم Checkpoints**: نقاط التحقق تضمن الجودة
5. **وثّق أثناء التطوير**: التوثيق المتأخر صعب ومكلف
6. **اختبر على بيئة مشابهة للإنتاج**: تجنب المفاجآت عند النشر

---

## الخلاصة

هذه الخطة تغطي تنفيذ نظام التعرفات والتسعير الشامل على 7 مراحل. النظام يعالج جميع المشاكل الحرجة في النظام الحالي ويوفر:

- **نظام تعرفات ديناميكي** يدعم التوسع الجغرافي
- **نظام باقات شهرية** مع Rollover و Overage
- **فوترة تلقائية** موثوقة
- **بوابات مخصصة** للشركات الكبيرة مع عزل تام
- **واجهات مستخدم** محسّنة (تعديلات طفيفة)
- **مزايا متقدمة** (خصومات، إحالة، دفع إلكتروني)
- **استقرار وموثوقية** عالية

### المتطلبات المغطاة

هذه الخطة تغطي جميع المتطلبات الـ 42:
- ✅ المتطلبات 1-7: الخدمات والتعرفات والتقسيم
- ✅ المتطلبات 8-13: البوابات المخصصة
- ✅ المتطلبات 14-17: الفوترة والإشعارات
- ✅ المتطلبات 18-24: الواجهات والتقارير والتنبيهات
- ✅ المتطلبات 25-30: البنية التحتية والأمان والـ APIs
- ✅ المتطلبات 31-36: العملات والخصومات والامتثال
- ✅ المتطلبات 37-42: الدعم والاختبار والتحليلات

### الخطوات التالية

1. **مراجعة الخطة** مع الفريق
2. **تخصيص الموارد** للمراحل المختلفة
3. **إعداد البيئة** (Development، Staging، Production)
4. **البدء بالمرحلة 1** فوراً

---

**تاريخ الإنشاء:** 2026-02-27  
**الإصدار:** 1.0  
**الحالة:** جاهز للتنفيذ  
**لغة البرمجة:** Go (Golang)  
**المدة المتوقعة:** 14 أسبوع

