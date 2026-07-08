# دليل ربط البوابات الخاصة (Private Gateway Integration Guide)

يسمح هذا المسار للشركات بربط خوادم الإرسال الخاصة بها (Private SMS Gateways) بنظام الهدهد. عند تفعيل مسار "البوابة الخاصة" للمستأجر، يقوم الهدهد بإرسال بيانات الرسالة فوراً إلى الرابط (URL) المخصص للشركة، ويقوم بعدها سيرفر الشركة بإعادة تقرير حالة التسليم (DLR) إلى الهدهد.

## 1. ما يصل إليك من الهدهد (Incoming Request to Your Server)

عند إرسال مستخدم لرسالة، سيقوم الهدهد بعمل `POST Request` غير متزامن (Asynchronous) إلى الرابط الخاص بك `PrivateServerURL`.

**شكل الطلب (HTTP POST):**
```http
POST /your-custom-endpoint
Content-Type: application/json
X-Message-ID: <UUID>

{
  "message_id": "123e4567-e89b-12d3-a456-426614174000",
  "tenant_id": "987fcdeb-51a2-43d7-9012-345678901234",
  "phone_number": "+966500000000",
  "content": "مرحبا بك في نظام الهدهد",
  "custom_id": "optional-client-id"
}
```

*ملاحظة: النظام لن ينتظر رد السيرفر الخاص بك لتحديث الحالة. هو نظام (Fire-and-Forget).*

---

## 2. ما يجب عليك إرساله للهدهد (Sending DLR Webhook to Alhudhud)

بمجرد قيام سيرفرك الخاص بإرسال الرسالة إلى مزود الاتصالات واستلام الرد منه (نجاح أو فشل)، يجب عليك استدعاء الـ API الخاص بالهدهد لتحديث حالة الرسالة.

**معلومات الـ Endpoint:**
* **الرابط:** `https://api.alhudhud.com/api/v1/private-gateway/dlr`
* **النوع:** `POST`
* **المصادقة:** يجب إرسال مفتاح `X-API-Key` الخاص بحسابك في الترويسة (Headers).

**شكل الطلب (HTTP POST):**
```http
POST /api/v1/private-gateway/dlr
Content-Type: application/json
X-API-Key: <YOUR_TENANT_API_KEY>

{
  "message_id": "123e4567-e89b-12d3-a456-426614174000",
  "status": "DELIVERED", 
  "error_code": ""
}
```

**الحالات المقبولة للحقل `status`:**
* `DELIVERED` : تم الإرسال/التسليم بنجاح.
* `FAILED` : فشل الإرسال (في هذه الحالة يُفضل ملء حقل `error_code` بسبب الفشل).

**مثال على رد الهدهد عند نجاح تحديث الحالة:**
```json
{
  "message": "Delivery report processed successfully",
  "status": "success"
}
```
