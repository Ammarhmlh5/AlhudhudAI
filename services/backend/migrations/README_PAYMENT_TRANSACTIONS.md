# Payment Transactions Table Migration

## نظرة عامة

هذا الـ Migration ينشئ جدول `payment_transactions` لتسجيل جميع عمليات الدفع من بوابات الدفع المختلفة (Stripe، PayPal، بوابات محلية).

## المتطلبات المنفذة

- **Requirement 39.8**: تسجيل جميع عمليات الدفع في سجل المعاملات
- **Requirement 39.9**: حفظ معرّف المعاملة من بوابة الدفع

## هيكل الجدول

### الحقول الأساسية

- `id`: المعرف الفريد للسجل
- `tenant_id`: معرف الشركة (للعزل التام)
- `transaction_id`: معرف فريد للمعاملة في النظام
- `external_transaction_id`: معرف المعاملة من بوابة الدفع الخارجية

### معلومات الدفع

- `payment_gateway`: بوابة الدفع المستخدمة (stripe, paypal, local)
- `payment_method`: طريقة الدفع (card, bank_transfer, wallet)
- `amount`: المبلغ
- `currency`: العملة (USD, YER, SAR)

### الحالة

- `status`: حالة المعاملة
  - `pending`: قيد الانتظار
  - `processing`: قيد المعالجة
  - `completed`: مكتملة
  - `failed`: فاشلة
  - `refunded`: مستردة

### معلومات الاسترجاع

- `refund_amount`: مبلغ الاسترجاع
- `refund_reason`: سبب الاسترجاع
- `refunded_at`: تاريخ الاسترجاع

### معلومات الخطأ

- `error_code`: كود الخطأ
- `error_message`: رسالة الخطأ

### البيانات الإضافية

- `metadata`: بيانات إضافية من بوابة الدفع بصيغة JSONB

## الأمان

### Row Level Security (RLS)

تم تفعيل RLS على الجدول مع Policies التالية:

1. **SELECT Policy**: يسمح للشركة برؤية معاملاتها فقط
2. **INSERT Policy**: يسمح للشركة بإضافة معاملات لها فقط
3. **UPDATE Policy**: يسمح للشركة بتحديث معاملاتها فقط
4. **DELETE Policy**: محظور تماماً للحفاظ على سجل المعاملات

### Indexes

تم إنشاء Indexes مركبة تبدأ بـ `tenant_id` لتحسين الأداء:

- `idx_payment_transactions_tenant`: للبحث حسب الشركة
- `idx_payment_transactions_tenant_status`: للبحث حسب الشركة والحالة
- `idx_payment_transactions_tenant_created`: للبحث حسب الشركة والتاريخ
- `idx_payment_transactions_tenant_gateway`: للبحث حسب الشركة وبوابة الدفع
- `idx_payment_transactions_external_id`: للبحث حسب معرف المعاملة الخارجي
- `idx_payment_transactions_transaction_id`: للبحث حسب معرف المعاملة

## تشغيل الـ Migration

### على Linux/Mac:

```bash
cd services/backend
./run-payment-transactions-migration.sh
```

### على Windows:

```powershell
cd services/backend
.\run-payment-transactions-migration.ps1
```

### يدوياً:

```bash
psql -h localhost -U sms_user -d sms_gateway -f migrations/028_payment_transactions_table.sql
```

## اختبار الـ Migration

```bash
# اختبار إنشاء الجدول
psql -h localhost -U sms_user -d sms_gateway -f test_payment_transactions_migration.sql
```

## أمثلة الاستخدام

### إضافة معاملة دفع جديدة

```sql
-- تعيين tenant_id
SET app.current_tenant = 'your-tenant-uuid';

-- إضافة معاملة
INSERT INTO payment_transactions (
    tenant_id,
    transaction_id,
    external_transaction_id,
    payment_gateway,
    payment_method,
    amount,
    currency,
    status,
    description
) VALUES (
    'your-tenant-uuid',
    'TXN-' || gen_random_uuid(),
    'pi_1234567890',
    'stripe',
    'card',
    50.00,
    'USD',
    'pending',
    'شحن رصيد'
);
```

### الاستعلام عن المعاملات

```sql
-- تعيين tenant_id
SET app.current_tenant = 'your-tenant-uuid';

-- عرض جميع المعاملات
SELECT * FROM payment_transactions
ORDER BY created_at DESC;

-- عرض المعاملات المكتملة
SELECT * FROM payment_transactions
WHERE status = 'completed'
ORDER BY created_at DESC;

-- عرض المعاملات حسب بوابة الدفع
SELECT 
    payment_gateway,
    COUNT(*) as total_transactions,
    SUM(amount) as total_amount,
    AVG(amount) as avg_amount
FROM payment_transactions
WHERE status = 'completed'
GROUP BY payment_gateway;
```

### تحديث حالة المعاملة

```sql
-- تعيين tenant_id
SET app.current_tenant = 'your-tenant-uuid';

-- تحديث حالة المعاملة إلى مكتملة
UPDATE payment_transactions
SET 
    status = 'completed',
    completed_at = CURRENT_TIMESTAMP
WHERE transaction_id = 'TXN-xxx';

-- تحديث حالة المعاملة إلى فاشلة
UPDATE payment_transactions
SET 
    status = 'failed',
    error_code = 'card_declined',
    error_message = 'البطاقة مرفوضة'
WHERE transaction_id = 'TXN-xxx';
```

### معالجة الاسترجاع

```sql
-- تعيين tenant_id
SET app.current_tenant = 'your-tenant-uuid';

-- تسجيل استرجاع
UPDATE payment_transactions
SET 
    status = 'refunded',
    refund_amount = amount,
    refund_reason = 'طلب العميل',
    refunded_at = CURRENT_TIMESTAMP
WHERE transaction_id = 'TXN-xxx'
AND status = 'completed';
```

## الملاحظات

1. **عدم الحذف**: لا يمكن حذف أي سجل من الجدول للحفاظ على سجل المعاملات الكامل
2. **Tenant Isolation**: جميع العمليات معزولة تماماً بين الشركات
3. **Audit Trail**: يتم تسجيل جميع التحديثات مع `updated_at`
4. **External IDs**: يتم حفظ معرفات المعاملات من بوابات الدفع الخارجية
5. **Metadata**: يمكن حفظ بيانات إضافية من بوابة الدفع بصيغة JSON

## التكامل مع النظام

هذا الجدول يتكامل مع:

- **Balance Manager**: لإضافة الرصيد عند نجاح الدفع
- **Accounting Transactions**: لتسجيل المعاملة المالية
- **Notification Service**: لإرسال إشعارات للعميل
- **Invoice Generator**: لتوليد فاتورة للدفع

## الخطوات التالية

بعد تشغيل هذا الـ Migration:

1. تطوير Stripe Integration (30.2)
2. تطوير PayPal Integration (30.3)
3. تطوير Payment Manager (30.4)
4. تطوير Payment APIs (30.5)
5. تطبيق PCI DSS Compliance (30.6)
