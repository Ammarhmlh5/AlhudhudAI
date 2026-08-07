# قائمة العمليات المالية المطلوب تنفيذها (Finance Operations Plan)

## الرابط بين النظام المالي (finance_db) والنظام الأساسي (backend)

---

## العمليات من واجهة الإدارة (admin-web)

| # | الحالة | العملية | الواجهة | API الخلفية | ملف التنفيذ |
|---|---|---|---|---|---|
| 1 | ✅ | **شحن رصيد شركة/مستخدم (Charge Wallet)** | `/accounting/receipts` → سند قبض | `POST /api/finance/charge` | `handlers_v2.go` + `operations.go` |
| 2 | ✅ | **إنشاء سند صرف (Payment Voucher)** | `/accounting/payments` → سند صرف | `POST /api/finance/debit-cashbox` | `handlers_v2.go` + `operations.go` |
| 3 | ✅ | **إضافة صندوق مالي جديد (Cashbox)** | `/accounting/cashboxes` | `POST /api/finance/cashboxes` | `handlers_v2.go` + `account_manager.go` |
| 3b | ✅ | **عرض قائمة الصناديق** | `/accounting/cashboxes` | `GET /api/finance/cashboxes` | `handlers_v2.go` + `account_manager.go` |
| 4 | ✅ | **عرض رصيد المحفظة لشركة** | `/companies/:id` → تبويب مالي | `GET /api/finance/balance?id=...` | `handlers_v2.go` |
| 5 | ✅ | **استعراض سجل حركات الصندوق (Statement)** | `/accounting/cashboxes` → كشف حساب | `GET /api/finance/statement?id=&from=&to=` | `handlers_v2.go` + `operations.go` |
| 6 | ✅ | **عكس/إلغاء سند (Reversal)** | `/accounting/receipts` أو `/accounting/payments` | `POST /api/finance/reverse` | `handlers_v2.go` + `operations.go` |
| 8 | ✅ | **استعراض سجل المعاملات** | `/accounting` | `GET /api/finance/transactions` | `handlers_v2.go` + `operations.go` |

---

## العمليات من واجهة الشركة (company-portal)

| # | الحالة | العملية | الواجهة | API الخلفية | ملف التنفيذ |
|---|---|---|---|---|---|
| 7 | ✅ | **شراء باقة رسائل (Buy Package)** | صفحة الاشتراكات | `POST /api/finance/purchase` ثم UPDATE sms_quotas | `handlers_v2.go` + `finance_client.go` |
| 8 | ✅ | **عرض رصيد الرسائل المتبقي** | لوحة التحكم الرئيسية | `GET /api/tenant/sms-quota` (النظام الأساسي) | `finance_client.go` |
| 9 | ✅ | **عرض الفواتير السابقة** | صفحة الفواتير | `GET /api/finance/invoices` | `handlers_v2.go` |
| 10 | ✅ | **طباعة/تنزيل فاتورة (Invoice HTML)** | صفحة الفواتير → زر تنزيل | `GET /api/finance/invoices/print?id=...` | `handlers_v2.go` + `invoice_service.go` |

---

## العمليات الآلية (System Operations)

| # | الحالة | العملية | المُشغّل | الآلية | ملف التنفيذ |
|---|---|---|---|---|---|
| 11 | ✅ | **إنشاء حساب محفظة تلقائياً عند تسجيل مستخدم جديد** | Hook في Backend | `POST /api/finance/wallets` | `finance_client.go → OnTenantCreated` |
| 12 | ✅ | **تصفير رصيد الرسائل نهاية الشهر (Monthly Reset)** | Cron Job شهري | `UPDATE tenant_sms_quotas SET available_sms=0` | `finance_client.go → MonthlyReset` |
| 13 | ✅ | **خصم رسائل (Deduct SMS quota)** | كل عملية إرسال | `UPDATE tenant_sms_quotas SET available_sms - N` | `finance_client.go → DeductSMSQuota` |

---

## ملخص الملفات المُنشأة (Backend)

| الملف | الموقع | المحتوى |
|---|---|---|
| `main.go` | `cmd/main.go` | ✅ نقطة بدء الخدمة مع الـ routes |
| `handlers_v2.go` | `internal/api/` | ✅ 16 HTTP Handler |
| `operations.go` | `internal/services/` | ✅ 5 عمليات محاسبية |
| `accounting_service.go` | `internal/services/` | ✅ القيد المزدوج |
| `account_manager.go` | `internal/services/` | ✅ إدارة الصناديق والحسابات |
| `invoice_service.go` | `internal/services/` | ✅ الفوترة والضريبة |
| `models.go` | `internal/models/` | ✅ نماذج قاعدة البيانات |

---

## Frontend - admin-web

| الملف | الموقع | المحتوى |
|---|---|---|
| `finance.js` | `admin-web/src/api/` | ✅ API Client - 15 function |
| `FinanceManager.jsx` | `admin-web/src/components/` | ✅ محدث - يتصل بـ API حقيقي |

---

## تشغيل الخدمة

```bash
# تشغيل خدمة المالية (Finance Service)
cd services/finance
go run cmd/main.go -db="postgresql://user:pass@localhost:5432/finance_db" -port=8081

# الخدمة تعمل على: http://localhost:8081
```

---

## نقاط النهاية (API Endpoints)

| الطريقة | المسار | الوصف |
|---------|--------|-------|
| POST | `/api/finance/charge` | شحن رصيد شركة |
| POST | `/api/finance/debit-cashbox` | سند صرف |
| POST | `/api/finance/cashboxes` | إنشاء صندوق |
| GET | `/api/finance/cashboxes` | قائمة الصناديق |
| GET | `/api/finance/balance` | رصيد المحفظة |
| GET | `/api/finance/statement` | كشف حساب |
| POST | `/api/finance/reverse` | عكس سند |
| POST | `/api/finance/purchase` | شراء باقة |
| GET | `/api/finance/transactions` | قائمة المعاملات |
| GET | `/api/finance/invoices` | قائمة الفواتير |
| GET | `/api/finance/invoices/print` | طباعة فاتورة |
| POST | `/api/finance/wallets` | إنشاء محفظة |
| GET | `/health` | فحص الصحة |

---

## ✅ الحالة النهائية

**جميع العمليات مكتملة وجاهزة للاستخدام!**

- ✅ Backend: 13 عملية
- ✅ Frontend: متصل بالـ API
- ✅ Database: الجداول موجودة (GORM Auto-Migrate)