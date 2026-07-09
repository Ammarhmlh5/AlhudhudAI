# دليل التوثيق الشامل - المهمة 36

## نظرة عامة

هذا الدليل يغطي إنشاء التوثيق الشامل لنظام التعرفات والتسعير:
- API Documentation
- User Guide
- Admin Guide
- Enterprise Guide
- Developer Guide
- Database Schema Documentation

## المتطلبات

### الأدوات المطلوبة
- Markdown Editor
- Postman (لتوليد API Collection)
- Draw.io أو Mermaid (للرسوم البيانية)

## المهمة 36.1: API Documentation

### الهدف
توثيق جميع APIs الجديدة بشكل شامل.

### خطوات التنفيذ

#### 1. إنشاء ملف API Reference

```markdown
# docs/API_REFERENCE_TARIFF.md

# API Reference - نظام التعرفات والتسعير

## نظرة عامة

هذا المستند يوثق جميع APIs الخاصة بنظام التعرفات والتسعير.

**Base URL:** `https://api.example.com/v1`

**Authentication:** Bearer Token

## الباقات (Packages)

### 1. عرض جميع الباقات

**Endpoint:** `GET /packages`

**Description:** عرض جميع الباقات المتاحة

**Headers:**
```
Authorization: Bearer {token}
```

**Query Parameters:**
- `service_id` (optional): تصفية حسب الخدمة
- `is_active` (optional): تصفية حسب الحالة
- `page` (optional): رقم الصفحة (default: 1)
- `limit` (optional): عدد النتائج (default: 10)

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "packages": [
      {
        "id": 1,
        "name": "Basic",
        "service_id": 2,
        "price": 10.00,
        "currency": "USD",
        "messages_included": 1000,
        "billing_cycle": "monthly",
        "rollover_enabled": false,
        "is_active": true
      }
    ],
    "pagination": {
      "page": 1,
      "limit": 10,
      "total": 5
    }
  }
}
```

**Example (cURL):**
```bash
curl -X GET "https://api.example.com/v1/packages?is_active=true" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

**Example (JavaScript):**
```javascript
const response = await fetch('https://api.example.com/v1/packages', {
  headers: {
    'Authorization': 'Bearer YOUR_TOKEN'
  }
});
const data = await response.json();
```

**Example (Python):**
```python
import requests

response = requests.get(
    'https://api.example.com/v1/packages',
    headers={'Authorization': 'Bearer YOUR_TOKEN'}
)
data = response.json()
```

### 2. الاشتراك في باقة

**Endpoint:** `POST /subscriptions`

**Description:** الاشتراك في باقة شهرية

**Headers:**
```
Authorization: Bearer {token}
Content-Type: application/json
```

**Request Body:**
```json
{
  "package_id": 1,
  "auto_renewal": true,
  "promo_code": "WELCOME10"
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "subscription": {
      "id": 123,
      "package_id": 1,
      "status": "active",
      "start_date": "2024-01-01",
      "end_date": "2024-02-01",
      "messages_included": 1000,
      "messages_used": 0,
      "messages_remaining": 1000,
      "auto_renewal": true
    },
    "balance_deducted": 10.00,
    "new_balance": 90.00
  }
}
```

**Error Responses:**

- **400 Bad Request:** بيانات غير صحيحة
```json
{
  "success": false,
  "error": {
    "code": "INVALID_PACKAGE",
    "message": "Package not found or inactive"
  }
}
```

- **402 Payment Required:** رصيد غير كافي
```json
{
  "success": false,
  "error": {
    "code": "INSUFFICIENT_BALANCE",
    "message": "Insufficient balance to subscribe",
    "details": {
      "required": 10.00,
      "available": 5.00
    }
  }
}
```

## الاستخدام (Usage)

### 3. عرض الاستخدام الحالي

**Endpoint:** `GET /subscriptions/usage`

**Description:** عرض الرسائل المستخدمة والمتبقية

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "subscription": {
      "id": 123,
      "package_name": "Standard",
      "messages_included": 3000,
      "messages_used": 1500,
      "messages_remaining": 1500,
      "overage_messages": 0,
      "overage_cost": 0.00,
      "rollover_messages": [
        {
          "id": 1,
          "messages_count": 500,
          "source_month": "2023-12",
          "expiry_date": "2024-02-28",
          "status": "active"
        }
      ]
    }
  }
}
```

## التعرفات (Tariffs)

### 4. حساب تكلفة رسالة

**Endpoint:** `POST /tariffs/calculate`

**Description:** حساب تكلفة رسالة قبل الإرسال

**Request Body:**
```json
{
  "to": "+967777777777",
  "message": "Hello, this is a test message",
  "service_id": 1
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "segments": 1,
    "cost_per_segment": 9.00,
    "total_cost": 9.00,
    "currency": "YER",
    "network_operator": "Yemen Mobile",
    "encoding": "GSM-7"
  }
}
```
```

#### 2. إنشاء Postman Collection

```json
// postman/Tariff_System_API.postman_collection.json
{
  "info": {
    "name": "Tariff System API",
    "description": "Complete API collection for Tariff System",
    "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
  },
  "item": [
    {
      "name": "Packages",
      "item": [
        {
          "name": "List Packages",
          "request": {
            "method": "GET",
            "header": [
              {
                "key": "Authorization",
                "value": "Bearer {{token}}"
              }
            ],
            "url": {
              "raw": "{{base_url}}/packages",
              "host": ["{{base_url}}"],
              "path": ["packages"]
            }
          }
        },
        {
          "name": "Subscribe to Package",
          "request": {
            "method": "POST",
            "header": [
              {
                "key": "Authorization",
                "value": "Bearer {{token}}"
              },
              {
                "key": "Content-Type",
                "value": "application/json"
              }
            ],
            "body": {
              "mode": "raw",
              "raw": "{\n  \"package_id\": 1,\n  \"auto_renewal\": true\n}"
            },
            "url": {
              "raw": "{{base_url}}/subscriptions",
              "host": ["{{base_url}}"],
              "path": ["subscriptions"]
            }
          }
        }
      ]
    }
  ]
}
```

### Checklist للتحقق

- [ ] جميع APIs موثقة
- [ ] أمثلة كود بلغات مختلفة
- [ ] Error Responses موثقة
- [ ] Postman Collection جاهزة

## المهمة 36.2: User Guide للعملاء

### الهدف
دليل شامل للعملاء لاستخدام النظام.

### خطوات التنفيذ

```markdown
# docs/USER_GUIDE_TARIFF.md

# دليل المستخدم - نظام التعرفات والتسعير

## مقدمة

مرحباً بك في نظام التعرفات والتسعير! هذا الدليل سيساعدك على:
- اختيار الباقة المناسبة
- فهم نظام الترحيل (Rollover)
- إدارة الاستخدام الإضافي (Overage)
- شحن الرصيد

## الباقات الشهرية

### ما هي الباقات الشهرية؟

الباقات الشهرية هي اشتراكات تمنحك عدداً محدداً من الرسائل شهرياً بسعر ثابت.

### الباقات المتاحة

#### 1. باقة Basic
- **السعر:** 10 دولار شهرياً
- **الرسائل:** 1,000 رسالة
- **سعر الرسالة:** 0.01 دولار
- **مناسبة لـ:** الأفراد والشركات الصغيرة

#### 2. باقة Standard
- **السعر:** 20 دولار شهرياً
- **الرسائل:** 3,000 رسالة
- **سعر الرسالة:** 0.0067 دولار
- **مناسبة لـ:** الشركات المتوسطة

#### 3. باقة Professional
- **السعر:** 40 دولار شهرياً
- **الرسائل:** 8,000 رسالة
- **سعر الرسالة:** 0.005 دولار
- **مناسبة لـ:** الشركات الكبيرة

### كيفية الاشتراك

1. سجّل الدخول لحسابك
2. اذهب إلى صفحة "الباقات"
3. اختر الباقة المناسبة
4. انقر على "اشترك الآن"
5. تأكد من كفاية رصيدك
6. أكّد الاشتراك

**ملاحظة:** سيتم خصم قيمة الباقة من رصيدك فوراً.

## نظام الترحيل (Rollover)

### ما هو الترحيل؟

الترحيل يسمح لك بنقل الرسائل المتبقية من الشهر الحالي للشهر التالي.

### كيف يعمل؟

1. في نهاية الشهر، يتم حساب الرسائل المتبقية
2. إذا كان الترحيل مفعّلاً، تُنقل الرسائل للشهر التالي
3. الرسائل المرحّلة لها تاريخ انتهاء صلاحية
4. تُستخدم الرسائل المرحّلة الأقدم أولاً (FIFO)

### مثال

- اشتركت في باقة Standard (3,000 رسالة)
- استخدمت 2,000 رسالة فقط
- تبقى 1,000 رسالة
- في الشهر التالي، سيكون لديك: 3,000 (جديدة) + 1,000 (مرحّلة) = 4,000 رسالة

### الحد الأقصى للترحيل

كل باقة لها حد أقصى للترحيل:
- Basic: 500 رسالة
- Standard: 1,000 رسالة
- Professional: 2,000 رسالة

## الاستخدام الإضافي (Overage)

### ما هو Overage؟

عندما تنفد رسائلك المضمنة، يمكنك الاستمرار في الإرسال بنفس سعر الباقة.

### كيف يعمل؟

1. عند نفاد الرسائل المضمنة، يتم تفعيل Overage تلقائياً
2. كل رسالة إضافية تُخصم من رصيدك
3. السعر = سعر الرسالة في باقتك
4. يمكنك متابعة التكلفة الإضافية في لوحة التحكم

### مثال

- باقة Basic: 1,000 رسالة بـ 10 دولار (0.01 دولار/رسالة)
- استخدمت 1,200 رسالة
- Overage: 200 رسالة × 0.01 = 2 دولار
- التكلفة الإجمالية: 10 (باقة) + 2 (overage) = 12 دولار

## شحن الرصيد

### طرق الشحن

1. **بطاقة ائتمان:** Visa, Mastercard
2. **PayPal**
3. **تحويل بنكي:** اتصل بالدعم

### خطوات الشحن

1. اذهب إلى "الرصيد"
2. انقر على "شحن الرصيد"
3. أدخل المبلغ
4. اختر طريقة الدفع
5. أكمل عملية الدفع

## الأسئلة الشائعة

### متى يتم تجديد اشتراكي؟

يتم التجديد تلقائياً في نفس تاريخ الاشتراك كل شهر.

### ماذا يحدث إذا لم يكن رصيدي كافياً للتجديد؟

- ستحصل على فترة سماح 3 أيام
- سيتم إرسال إشعار لك
- إذا لم تشحن رصيدك، سيتم تعليق الاشتراك

### هل يمكنني إلغاء الاشتراك؟

نعم، يمكنك إلغاء التجديد التلقائي في أي وقت من إعدادات الاشتراك.

### هل يمكنني تغيير الباقة؟

نعم، يمكنك الترقية أو التخفيض في أي وقت. سيتم احتساب الفرق.
```

### Checklist للتحقق

- [ ] الدليل شامل وواضح
- [ ] أمثلة عملية
- [ ] أسئلة شائعة مجابة
- [ ] لغة بسيطة ومفهومة

## المهمة 36.3: Admin Guide

### الهدف
دليل للمديرين لإدارة النظام.

### خطوات التنفيذ

```markdown
# docs/ADMIN_GUIDE_TARIFF.md

# دليل المدير - نظام التعرفات والتسعير

## نظرة عامة

هذا الدليل موجه لمديري المنصة لإدارة:
- التعرفات
- الباقات
- البوابات المخصصة
- التوريد المباشر

## إدارة التعرفات

### إضافة تعرفة جديدة

1. اذهب إلى "الإعدادات" → "التعرفات"
2. انقر على "إضافة تعرفة"
3. املأ البيانات:
   - الخدمة
   - الدولة
   - الشبكة
   - التكلفة
   - العملة
   - تاريخ السريان
4. احفظ

### تحديث تعرفة موجودة

**مهم:** لا تحذف التعرفات القديمة! أنشئ تعرفة جديدة بتاريخ سريان جديد.

1. اذهب إلى "الإعدادات" → "التعرفات"
2. انقر على "تحديث" بجانب التعرفة
3. أدخل التكلفة الجديدة
4. حدد تاريخ السريان
5. احفظ

### عرض سجل التعرفات

يمكنك عرض جميع التعرفات السابقة لكل شبكة من صفحة "سجل التعرفات".

## إدارة الباقات

### إنشاء باقة جديدة

1. اذهب إلى "الإعدادات" → "الباقات"
2. انقر على "إنشاء باقة"
3. املأ البيانات:
   - الاسم
   - الخدمة
   - السعر
   - عدد الرسائل
   - دورة الفوترة
   - إعدادات Rollover
4. احفظ

### تفعيل/تعطيل باقة

- لتعطيل باقة: انقر على زر "تعطيل"
- الباقات المعطلة لا تظهر للعملاء الجدد
- العملاء الحاليون يستمرون في اشتراكهم

### إعدادات Rollover

لكل باقة، يمكنك:
- تفعيل/تعطيل Rollover
- تحديد الحد الأقصى للترحيل
- تحديد مدة الصلاحية (بالأشهر)

## إدارة البوابات المخصصة

### إنشاء بوابة مخصصة

1. اذهب إلى "البوابات المخصصة"
2. انقر على "إنشاء بوابة"
3. املأ البيانات:
   - اسم البوابة
   - الشركة المؤسسية
   - معلومات الاتصال بشركة الاتصالات
   - نوع السيرفر
   - الحصص الموردية
4. احفظ

### مراقبة البوابات

من لوحة "البوابات المخصصة"، يمكنك:
- عرض حالة كل بوابة
- مراقبة الاستخدام
- عرض التنبيهات
- عرض الإحصائيات

### تعليق/إيقاف بوابة

في حالات الطوارئ:
1. اذهب إلى تفاصيل البوابة
2. انقر على "تعليق" أو "إيقاف"
3. أدخل السبب
4. أكّد

## التوريد المباشر

### إضافة رصيد لعميل

1. اذهب إلى "العملاء"
2. ابحث عن العميل
3. انقر على "توريد رصيد"
4. أدخل:
   - المبلغ
   - العملة
   - السبب
5. أكّد العملية

**ملاحظة:** جميع عمليات التوريد تُسجل في سجل التدقيق.

## التقارير

### تقرير الإيرادات الشهرية

يعرض:
- إيرادات الباقات
- إيرادات Overage
- إيرادات البوابات المخصصة
- الإجمالي

### تقرير أداء الباقات

يعرض:
- عدد المشتركين لكل باقة
- معدل التجديد
- معدل الإلغاء

### تقرير البوابات المخصصة

يعرض:
- وقت التشغيل
- الإنتاجية
- الاستخدام الموردي
```

### Checklist للتحقق

- [ ] جميع الوظائف الإدارية موثقة
- [ ] خطوات واضحة ومفصلة
- [ ] تحذيرات للعمليات الحرجة
- [ ] أمثلة عملية

## المهمة 36.4: Enterprise Guide

### الهدف
دليل للشركات المؤسسية لإدارة بوابتها المخصصة.

### المهمة 36.5: Developer Guide

### الهدف
دليل للمطورين لفهم معمارية النظام.

### المهمة 36.6: Database Schema Documentation

### الهدف
توثيق جميع الجداول والعلاقات.

## الخلاصة

هذا الدليل يغطي جميع جوانب التوثيق للمهمة 36.

**Requirements Covered:** 30.12, 30.13, 37.10, 37.11
