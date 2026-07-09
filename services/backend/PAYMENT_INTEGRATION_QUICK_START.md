# Payment Integration Quick Start Guide

## نظرة سريعة

دليل سريع لبدء استخدام نظام التكامل مع بوابات الدفع.

## التثبيت السريع

### 1. تثبيت المكتبات المطلوبة

```bash
cd services/backend

# Stripe SDK
go get github.com/stripe/stripe-go/v76

# لا حاجة لمكتبة PayPal - نستخدم HTTP مباشرة
```

### 2. تشغيل Migration

```powershell
# Windows
.\run-payment-transactions-migration.ps1

# أو يدوياً
psql -h localhost -U sms_user -d sms_gateway -f migrations/028_payment_transactions_table.sql
```

### 3. إعداد المتغيرات البيئية

أضف إلى ملف `.env`:

```env
# Stripe
STRIPE_API_KEY=sk_test_your_stripe_key
STRIPE_WEBHOOK_SECRET=whsec_your_webhook_secret

# PayPal
PAYPAL_CLIENT_ID=your_paypal_client_id
PAYPAL_CLIENT_SECRET=your_paypal_client_secret
PAYPAL_SANDBOX=true
```

## الاستخدام السريع

### 1. تهيئة Payment System

```go
package main

import (
    "sms-gateway/internal/payment"
    "os"
)

func main() {
    // Initialize Stripe
    stripeClient := payment.NewStripeClient(
        os.Getenv("STRIPE_API_KEY"),
        os.Getenv("STRIPE_WEBHOOK_SECRET"),
    )

    // Initialize PayPal
    paypalClient := payment.NewPayPalClient(
        os.Getenv("PAYPAL_CLIENT_ID"),
        os.Getenv("PAYPAL_CLIENT_SECRET"),
        os.Getenv("PAYPAL_SANDBOX") == "true",
    )

    // Initialize Payment Manager
    paymentManager := payment.NewManager(
        db,
        stripeClient,
        paypalClient,
        balanceTopupFunc,
        notificationFunc,
    )

    // Initialize Payment Handler
    paymentHandler := api.NewPaymentHandler(
        paymentManager,
        stripeClient,
        paypalClient,
    )

    // Register routes
    registerPaymentRoutes(router, paymentHandler)
}
```

### 2. تسجيل Routes

```go
func registerPaymentRoutes(router *gin.Engine, handler *api.PaymentHandler) {
    v1 := router.Group("/api/v1")
    {
        // Payment routes (require authentication)
        v1.POST("/payments/initiate", handler.InitiatePayment)
        v1.GET("/payments/:transaction_id", handler.GetPaymentStatus)
        v1.GET("/payments", handler.ListPaymentTransactions)
        v1.POST("/payments/paypal/capture/:order_id", handler.PayPalCapturePayment)
    }

    // Webhook routes (no authentication)
    router.POST("/api/v1/webhooks/stripe", handler.StripeWebhook)
    router.POST("/api/v1/webhooks/paypal", handler.PayPalWebhook)
}
```

### 3. تطبيق Balance Topup Function

```go
func balanceTopupFunc(ctx context.Context, tenantID uuid.UUID, amount float64, currency, reason string) error {
    // Set tenant context
    db.ExecContext(ctx, "SET app.current_tenant = $1", tenantID.String())

    // Update tenant balance
    query := `
        UPDATE tenants
        SET balance = balance + $1
        WHERE id = $2
    `
    _, err := db.ExecContext(ctx, query, amount, tenantID)
    if err != nil {
        return fmt.Errorf("failed to update balance: %w", err)
    }

    // Log in balance_history
    historyQuery := `
        INSERT INTO balance_history (
            tenant_id, amount, change_type, reason
        ) VALUES ($1, $2, 'TOPUP', $3)
    `
    _, err = db.ExecContext(ctx, historyQuery, tenantID, amount, reason)
    if err != nil {
        return fmt.Errorf("failed to log balance history: %w", err)
    }

    return nil
}
```

### 4. تطبيق Notification Function

```go
func notificationFunc(ctx context.Context, tenantID uuid.UUID, notificationType, message string) error {
    // Get tenant email
    var email string
    query := `SELECT email FROM tenants WHERE id = $1`
    err := db.GetContext(ctx, &email, query, tenantID)
    if err != nil {
        return fmt.Errorf("failed to get tenant email: %w", err)
    }

    // Send email notification
    // (استخدم مكتبة email مثل gomail)
    
    return nil
}
```

## أمثلة الاستخدام

### Frontend - Stripe Payment

```javascript
// 1. طلب Payment Intent من Backend
const response = await fetch('/api/v1/payments/initiate', {
  method: 'POST',
  headers: {
    'Authorization': `Bearer ${token}`,
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    amount: 50.00,
    currency: 'USD',
    payment_gateway: 'stripe',
    description: 'شحن رصيد'
  })
});

const { data } = await response.json();
const { client_secret, transaction_id } = data;

// 2. استخدام Stripe.js لإتمام الدفع
const stripe = Stripe('pk_test_your_publishable_key');

const { error } = await stripe.confirmCardPayment(client_secret, {
  payment_method: {
    card: cardElement,
    billing_details: {
      name: 'Customer Name'
    }
  }
});

if (error) {
  console.error('Payment failed:', error.message);
} else {
  console.log('Payment successful!');
  // Stripe سيرسل webhook تلقائياً
  // Backend سيضيف الرصيد تلقائياً
}
```

### Frontend - PayPal Payment

```javascript
// 1. طلب PayPal Order من Backend
const response = await fetch('/api/v1/payments/initiate', {
  method: 'POST',
  headers: {
    'Authorization': `Bearer ${token}`,
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    amount: 75.00,
    currency: 'USD',
    payment_gateway: 'paypal',
    description: 'شحن رصيد'
  })
});

const { data } = await response.json();
const { payment_url, transaction_id } = data;

// 2. تحويل المستخدم لصفحة PayPal
window.location.href = payment_url;

// 3. بعد موافقة المستخدم، PayPal يحوّله لصفحة النجاح
// في صفحة النجاح، استدعي capture endpoint:
const orderID = new URLSearchParams(window.location.search).get('token');

const captureResponse = await fetch(`/api/v1/payments/paypal/capture/${orderID}`, {
  method: 'POST',
  headers: {
    'Authorization': `Bearer ${token}`
  }
});

if (captureResponse.ok) {
  console.log('Payment captured successfully!');
  // Backend سيضيف الرصيد تلقائياً
}
```

## إعداد Webhooks

### Stripe Webhook

1. اذهب إلى [Stripe Dashboard](https://dashboard.stripe.com/webhooks)
2. أضف endpoint جديد: `https://your-domain.com/api/v1/webhooks/stripe`
3. اختر الأحداث:
   - `payment_intent.succeeded`
   - `payment_intent.payment_failed`
   - `payment_intent.processing`
   - `charge.refunded`
4. انسخ Webhook Secret وأضفه لـ `.env`

### PayPal Webhook

1. اذهب إلى [PayPal Developer Dashboard](https://developer.paypal.com/dashboard/applications)
2. اختر تطبيقك
3. أضف webhook URL: `https://your-domain.com/api/v1/webhooks/paypal`
4. اختر الأحداث:
   - `PAYMENT.CAPTURE.COMPLETED`
   - `PAYMENT.CAPTURE.DENIED`

## الاختبار

### اختبار Stripe

```bash
# استخدم Stripe CLI للاختبار المحلي
stripe listen --forward-to localhost:8080/api/v1/webhooks/stripe

# في terminal آخر، أرسل test webhook
stripe trigger payment_intent.succeeded
```

### اختبار PayPal

استخدم PayPal Sandbox:
- [PayPal Sandbox](https://www.sandbox.paypal.com/)
- أنشئ حسابات تجريبية (buyer & seller)
- استخدم بيانات الاعتماد التجريبية

## PCI DSS Compliance Checklist

- ✅ لا نخزن بيانات البطاقات
- ✅ نستخدم Tokenization (Stripe/PayPal)
- ✅ جميع الاتصالات عبر HTTPS
- ✅ التحقق من توقيع Webhooks
- ✅ Tenant Isolation مع RLS
- ✅ Audit Trail كامل
- ✅ رسائل خطأ عامة (لا تكشف تفاصيل)

## الأخطاء الشائعة

### 1. Webhook Signature Verification Failed

```
Error: failed to verify webhook signature
```

**الحل:**
- تأكد من صحة `STRIPE_WEBHOOK_SECRET`
- تأكد من إرسال `Stripe-Signature` header

### 2. Transaction Not Found

```
Error: transaction not found
```

**الحل:**
- تأكد من تعيين `app.current_tenant` قبل الاستعلام
- تأكد من صحة `transaction_id`

### 3. Balance Topup Failed

```
Error: failed to update balance
```

**الحل:**
- تأكد من وجود `balanceTopupFunc`
- تأكد من صحة `tenant_id`
- تحقق من صلاحيات قاعدة البيانات

## الموارد

- [Stripe Documentation](https://stripe.com/docs)
- [PayPal Documentation](https://developer.paypal.com/docs)
- [PCI DSS Standards](https://www.pcisecuritystandards.org/)
- [Payment Integration README](internal/payment/README.md)

## الدعم

للمساعدة أو الإبلاغ عن مشاكل:
1. راجع [Payment Integration README](internal/payment/README.md)
2. تحقق من logs في `payment_transactions` table
3. استخدم Stripe/PayPal Dashboard للتحقق من المعاملات

## الخطوات التالية

1. ✅ إعداد البيئة
2. ✅ تشغيل Migration
3. ✅ تهيئة Payment System
4. ⏳ تطوير واجهة المستخدم
5. ⏳ اختبار التكامل الكامل
6. ⏳ إعداد Webhooks في الإنتاج
7. ⏳ النشر
