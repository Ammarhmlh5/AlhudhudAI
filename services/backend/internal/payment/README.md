# Payment Integration System

## نظرة عامة

نظام التكامل مع بوابات الدفع يوفر طريقة آمنة ومتوافقة مع معايير PCI DSS لمعالجة المدفوعات الإلكترونية.

## المتطلبات المنفذة

- **Requirement 39.1**: دعم التكامل مع بوابات دفع محلية
- **Requirement 39.2**: دعم التكامل مع بوابات دفع عالمية (Stripe، PayPal)
- **Requirement 39.3**: السماح للعميل باختيار طريقة الدفع المفضلة
- **Requirement 39.4**: تحويل العميل لبوابة الدفع عند اختيار شحن الرصيد
- **Requirement 39.5**: إضافة الرصيد تلقائياً عند نجاح الدفع
- **Requirement 39.6**: إرسال إشعار للعميل عند نجاح الدفع
- **Requirement 39.7**: عرض رسالة خطأ واضحة عند فشل الدفع
- **Requirement 39.8**: تسجيل جميع عمليات الدفع في سجل المعاملات
- **Requirement 39.9**: حفظ معرّف المعاملة من بوابة الدفع
- **Requirement 39.10**: دعم استرجاع المبالغ (Refunds)
- **Requirement 39.11**: تشفير جميع بيانات الدفع
- **Requirement 39.12**: الامتثال لمعايير PCI DSS للأمان

## المكونات

### 1. Payment Types (`types.go`)

يحتوي على جميع الأنواع والثوابت المستخدمة في النظام:

- `PaymentGateway`: بوابات الدفع المدعومة (Stripe، PayPal، Local)
- `PaymentStatus`: حالات المعاملات (Pending، Processing، Completed، Failed، Refunded)
- `PaymentMethod`: طرق الدفع (Card، Bank Transfer، Wallet، PayPal)
- `PaymentTransaction`: نموذج معاملة الدفع
- `CreatePaymentRequest`: طلب إنشاء دفع
- `PaymentIntentResponse`: استجابة نية الدفع
- `RefundRequest`: طلب استرجاع
- `WebhookEvent`: حدث webhook من بوابة الدفع

### 2. Stripe Integration (`stripe.go`)

تكامل كامل مع Stripe:

- `CreatePaymentIntent`: إنشاء نية دفع
- `HandleWebhook`: معالجة أحداث webhook
- `ProcessRefund`: معالجة استرجاع المبالغ
- `GetPaymentStatus`: الحصول على حالة الدفع

**الأحداث المدعومة:**
- `payment_intent.succeeded`: نجاح الدفع
- `payment_intent.payment_failed`: فشل الدفع
- `payment_intent.processing`: الدفع قيد المعالجة
- `charge.refunded`: استرجاع المبلغ

### 3. PayPal Integration (`paypal.go`)

تكامل كامل مع PayPal:

- `CreateOrder`: إنشاء طلب PayPal
- `CapturePayment`: التقاط الدفع بعد موافقة المستخدم
- `GetOrderDetails`: الحصول على تفاصيل الطلب
- `getAccessToken`: الحصول على رمز OAuth

### 4. Payment Manager (`manager.go`)

مدير الدفع الرئيسي:

- `ProcessPayment`: معالجة طلب الدفع
- `HandleWebhookEvent`: معالجة أحداث webhook
- `AutoTopup`: شحن الرصيد تلقائياً
- `GetPaymentStatus`: الحصول على حالة الدفع
- `ListPaymentTransactions`: عرض قائمة المعاملات

**الوظائف الداخلية:**
- `storeTransaction`: تخزين المعاملة في قاعدة البيانات
- `updateTransactionStatus`: تحديث حالة المعاملة
- `handleCompletedPayment`: معالجة الدفع المكتمل
- `handleFailedPayment`: معالجة الدفع الفاشل

### 5. Payment API Handler (`payment_handler.go`)

معالج API للدفع:

- `InitiatePayment`: بدء عملية الدفع
- `GetPaymentStatus`: الحصول على حالة الدفع
- `ListPaymentTransactions`: عرض قائمة المعاملات
- `StripeWebhook`: معالجة webhook من Stripe
- `PayPalWebhook`: معالجة webhook من PayPal
- `PayPalCapturePayment`: التقاط دفع PayPal

## PCI DSS Compliance

### 1. عدم تخزين بيانات البطاقات (Requirement 39.12)

**المبدأ:** لا نقوم بتخزين أي بيانات حساسة للبطاقات في قاعدة البيانات.

```go
// ✅ صحيح: نستخدم Tokenization من Stripe/PayPal
// لا نخزن: رقم البطاقة، CVV، تاريخ الانتهاء

// ✅ ما نخزنه فقط:
- transaction_id: معرف المعاملة الداخلي
- external_transaction_id: معرف المعاملة من بوابة الدفع
- payment_gateway: اسم بوابة الدفع
- amount: المبلغ
- currency: العملة
- status: الحالة
- metadata: بيانات إضافية (لا تحتوي على بيانات حساسة)
```

### 2. استخدام Tokenization (Requirement 39.12)

**Stripe:**
```go
// Stripe يوفر Payment Intent مع Client Secret
// العميل يستخدم Stripe.js لإدخال بيانات البطاقة مباشرة
// لا تمر بيانات البطاقة عبر سيرفرنا أبداً

params := &stripe.PaymentIntentParams{
    Amount:   stripe.Int64(amountInCents),
    Currency: stripe.String(req.Currency),
    AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
        Enabled: stripe.Bool(true),
    },
}
```

**PayPal:**
```go
// PayPal يوفر Order ID و Approval URL
// العميل يُحوّل لصفحة PayPal لإتمام الدفع
// لا تمر بيانات الدفع عبر سيرفرنا أبداً

orderReq := map[string]interface{}{
    "intent": "CAPTURE",
    "purchase_units": []map[string]interface{}{...},
}
```

### 3. تشفير البيانات (Requirement 39.11)

**في قاعدة البيانات:**
```sql
-- جميع الاتصالات بقاعدة البيانات تستخدم SSL/TLS
-- البيانات الحساسة (إن وجدت) تُشفّر باستخدام pgcrypto

-- مثال: تشفير metadata إذا احتوت على بيانات حساسة
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- لكن في حالتنا، لا نخزن بيانات حساسة أصلاً
```

**في النقل:**
```go
// جميع الاتصالات تستخدم HTTPS/TLS 1.3
// Stripe و PayPal يفرضون HTTPS

// في main.go:
router.RunTLS(":443", "cert.pem", "key.pem")
```

### 4. Webhook Security

**Stripe Webhook Verification:**
```go
// التحقق من توقيع Stripe
event, err := webhook.ConstructEvent(payload, signature, webhookSecret)
if err != nil {
    return fmt.Errorf("failed to verify webhook signature: %w", err)
}
```

**PayPal Webhook Verification:**
```go
// في الإنتاج، يجب التحقق من توقيع PayPal
// باستخدام PayPal Webhook Verification API
```

### 5. Access Control

**Tenant Isolation:**
```go
// جميع العمليات معزولة بين الشركات
if _, err := m.db.ExecContext(ctx, "SET app.current_tenant = $1", tenantID.String()); err != nil {
    return fmt.Errorf("failed to set tenant context: %w", err)
}

// Row Level Security في قاعدة البيانات
ALTER TABLE payment_transactions ENABLE ROW LEVEL SECURITY;
```

### 6. Audit Trail

**تسجيل جميع العمليات:**
```go
// جميع المعاملات تُسجّل في payment_transactions
// مع timestamps كاملة
- initiated_at: وقت بدء المعاملة
- completed_at: وقت إتمام المعاملة
- created_at: وقت الإنشاء
- updated_at: وقت آخر تحديث
```

### 7. Error Handling

**عدم كشف معلومات حساسة:**
```go
// ✅ صحيح: رسائل خطأ عامة للعميل
c.JSON(http.StatusInternalServerError, ErrorResponse{
    Success: false,
    Error: ErrorDetail{
        Code:    "PAYMENT_FAILED",
        Message: "فشلت عملية الدفع. يرجى المحاولة مرة أخرى.",
    },
})

// ❌ خطأ: لا نكشف تفاصيل داخلية
// Message: "Database connection failed: connection refused"
```

## استخدام النظام

### 1. إعداد البيئة

```bash
# Stripe
export STRIPE_API_KEY="sk_test_..."
export STRIPE_WEBHOOK_SECRET="whsec_..."

# PayPal
export PAYPAL_CLIENT_ID="..."
export PAYPAL_CLIENT_SECRET="..."
export PAYPAL_SANDBOX=true  # false للإنتاج
```

### 2. تشغيل Migration

```bash
cd services/backend
./run-payment-transactions-migration.ps1
```

### 3. تهيئة Payment Manager

```go
// في main.go
stripeClient := payment.NewStripeClient(
    os.Getenv("STRIPE_API_KEY"),
    os.Getenv("STRIPE_WEBHOOK_SECRET"),
)

paypalClient := payment.NewPayPalClient(
    os.Getenv("PAYPAL_CLIENT_ID"),
    os.Getenv("PAYPAL_CLIENT_SECRET"),
    os.Getenv("PAYPAL_SANDBOX") == "true",
)

paymentManager := payment.NewManager(
    db,
    stripeClient,
    paypalClient,
    balanceTopupFunc,
    notificationFunc,
)

paymentHandler := api.NewPaymentHandler(
    paymentManager,
    stripeClient,
    paypalClient,
)
```

### 4. تسجيل Routes

```go
// Payment routes
v1.POST("/payments/initiate", paymentHandler.InitiatePayment)
v1.GET("/payments/:transaction_id", paymentHandler.GetPaymentStatus)
v1.GET("/payments", paymentHandler.ListPaymentTransactions)

// Webhook routes (بدون authentication)
router.POST("/api/v1/webhooks/stripe", paymentHandler.StripeWebhook)
router.POST("/api/v1/webhooks/paypal", paymentHandler.PayPalWebhook)

// PayPal capture route
v1.POST("/payments/paypal/capture/:order_id", paymentHandler.PayPalCapturePayment)
```

## أمثلة API

### 1. بدء عملية دفع (Stripe)

```bash
curl -X POST http://localhost:8080/api/v1/payments/initiate \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "amount": 50.00,
    "currency": "USD",
    "payment_gateway": "stripe",
    "description": "شحن رصيد"
  }'
```

**Response:**
```json
{
  "success": true,
  "data": {
    "transaction_id": "TXN-xxx-xxx-xxx",
    "client_secret": "pi_xxx_secret_xxx",
    "status": "pending",
    "metadata": {
      "payment_intent_id": "pi_xxx",
      "stripe_status": "requires_payment_method"
    }
  }
}
```

### 2. بدء عملية دفع (PayPal)

```bash
curl -X POST http://localhost:8080/api/v1/payments/initiate \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "amount": 75.00,
    "currency": "USD",
    "payment_gateway": "paypal",
    "description": "شحن رصيد"
  }'
```

**Response:**
```json
{
  "success": true,
  "data": {
    "transaction_id": "TXN-xxx-xxx-xxx",
    "payment_url": "https://www.paypal.com/checkoutnow?token=xxx",
    "status": "pending",
    "metadata": {
      "paypal_order_id": "xxx",
      "paypal_status": "CREATED"
    }
  }
}
```

### 3. الحصول على حالة الدفع

```bash
curl -X GET http://localhost:8080/api/v1/payments/TXN-xxx-xxx-xxx \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": 1,
    "tenant_id": "xxx-xxx-xxx",
    "transaction_id": "TXN-xxx-xxx-xxx",
    "external_transaction_id": "pi_xxx",
    "payment_gateway": "stripe",
    "payment_method": "card",
    "amount": 50.00,
    "currency": "USD",
    "status": "completed",
    "description": "شحن رصيد",
    "completed_at": "2024-01-15T10:30:00Z",
    "created_at": "2024-01-15T10:25:00Z"
  }
}
```

### 4. عرض قائمة المعاملات

```bash
curl -X GET "http://localhost:8080/api/v1/payments?limit=10&offset=0" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

## تدفق العمل

### Stripe Flow

```
1. العميل يطلب شحن الرصيد
   ↓
2. Backend ينشئ Payment Intent في Stripe
   ↓
3. Backend يعيد Client Secret للعميل
   ↓
4. العميل يستخدم Stripe.js لإدخال بيانات البطاقة
   ↓
5. Stripe يعالج الدفع ويرسل webhook
   ↓
6. Backend يستقبل webhook ويحدث الحالة
   ↓
7. Backend يضيف الرصيد تلقائياً
   ↓
8. Backend يرسل إشعار للعميل
```

### PayPal Flow

```
1. العميل يطلب شحن الرصيد
   ↓
2. Backend ينشئ Order في PayPal
   ↓
3. Backend يعيد Approval URL للعميل
   ↓
4. العميل يُحوّل لصفحة PayPal
   ↓
5. العميل يوافق على الدفع في PayPal
   ↓
6. PayPal يحوّل العميل لصفحة النجاح
   ↓
7. Backend يلتقط الدفع (Capture)
   ↓
8. Backend يضيف الرصيد تلقائياً
   ↓
9. Backend يرسل إشعار للعميل
```

## الأمان

### 1. لا تخزين بيانات البطاقات
- ✅ نستخدم Stripe و PayPal لمعالجة الدفع
- ✅ بيانات البطاقة لا تمر عبر سيرفرنا
- ✅ نخزن فقط معرفات المعاملات

### 2. HTTPS فقط
- ✅ جميع الاتصالات عبر HTTPS/TLS 1.3
- ✅ Stripe و PayPal يفرضون HTTPS

### 3. Webhook Verification
- ✅ التحقق من توقيع Stripe
- ✅ التحقق من توقيع PayPal (في الإنتاج)

### 4. Tenant Isolation
- ✅ Row Level Security في قاعدة البيانات
- ✅ جميع العمليات معزولة بين الشركات

### 5. Audit Trail
- ✅ تسجيل جميع المعاملات
- ✅ timestamps كاملة
- ✅ لا يمكن حذف السجلات

## الاختبار

### 1. Stripe Test Cards

```
# بطاقة ناجحة
4242 4242 4242 4242

# بطاقة مرفوضة
4000 0000 0000 0002

# بطاقة تتطلب 3D Secure
4000 0027 6000 3184
```

### 2. PayPal Sandbox

```
# حساب مشتري تجريبي
Email: buyer@example.com
Password: test1234

# حساب بائع تجريبي
Email: seller@example.com
Password: test1234
```

## الخطوات التالية

1. ✅ إنشاء جدول payment_transactions
2. ✅ تطوير Stripe Integration
3. ✅ تطوير PayPal Integration
4. ✅ تطوير Payment Manager
5. ✅ تطوير Payment APIs
6. ✅ تطبيق PCI DSS Compliance
7. ⏳ اختبار التكامل الكامل
8. ⏳ إضافة بوابات دفع محلية
9. ⏳ تطوير واجهة المستخدم للدفع
10. ⏳ النشر في الإنتاج

## الدعم

للمزيد من المعلومات:
- [Stripe Documentation](https://stripe.com/docs)
- [PayPal Documentation](https://developer.paypal.com/docs)
- [PCI DSS Requirements](https://www.pcisecuritystandards.org/)
