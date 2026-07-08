# Bug 15 Preservation Tests - تعليمات التشغيل

## نظرة عامة

هذا المستند يوضح كيفية تشغيل **اختبارات الحفاظ (Preservation Tests)** لـ Bug 15.

**Validates: Requirements 3.9, 3.15**

## ما هي اختبارات الحفاظ؟

اختبارات الحفاظ تتحقق من أن الوظائف الموجودة تستمر بالعمل بشكل صحيح:
- ✓ **قبل تطبيق الإصلاح** (على الكود غير المُصلح)
- ✓ **بعد تطبيق الإصلاح** (على الكود المُصلح)

الهدف: التأكد من أن إصلاح Bug 15 لا يكسر الوظائف الموجودة.

## متطلبات الحفاظ

### Requirement 3.9: Successful API Requests Continue

```
WHEN valid JWT tokens are provided
THEN the system SHALL continue to authenticate users
AND extract tenant context correctly
```

**ما يجب أن يستمر بالعمل:**
- طلبات API الناجحة مع tokens صالحة
- إضافة Authorization header بشكل صحيح
- معالجة الاستجابات الناجحة (200, 201, etc.)
- معالجة الأخطاء غير 401 (400, 403, 404, 500, etc.)

### Requirement 3.15: Authentication Flow Continues

```
WHEN users log in with correct credentials
THEN the system SHALL continue to issue tokens
AND establish sessions
```

**ما يجب أن يستمر بالعمل:**
- تسجيل الدخول وتخزين token
- إرسال طلبات مصادق عليها
- تسجيل الخروج وإزالة token
- إرسال طلبات بدون مصادقة

## الاختبارات المتضمنة

### Test 1: Successful Requests Continue

**الهدف:** التحقق من أن الطلبات الناجحة تستمر بالعمل

**الخطوات:**
1. تخزين token صالح في localStorage
2. إرسال طلب API ناجح (200 OK)
3. التحقق من استلام الاستجابة بشكل صحيح
4. التحقق من عدم إزالة token
5. التحقق من عدم إعادة التوجيه

**النتيجة المتوقعة:**
- ✓ الاستجابة مستلمة بنجاح
- ✓ Authorization header مضاف
- ✓ Token محفوظ في localStorage
- ✓ لا يوجد إعادة توجيه

### Test 2: Non-401 Errors Handled Correctly

**الهدف:** التحقق من أن الأخطاء غير 401 لا تسبب logout

**الخطوات:**
1. تخزين token صالح في localStorage
2. إرسال طلبات تعيد أخطاء مختلفة (400, 403, 404, 500, 503)
3. التحقق من استلام الأخطاء بشكل صحيح
4. التحقق من عدم إزالة token
5. التحقق من عدم إعادة التوجيه

**النتيجة المتوقعة:**
- ✓ الأخطاء مستلمة بشكل صحيح
- ✓ Token محفوظ في localStorage
- ✓ لا يوجد إعادة توجيه
- ✓ فقط 401 يسبب logout (بعد الإصلاح)

### Test 3: Authorization Header Added

**الهدف:** التحقق من إضافة Authorization header بشكل صحيح

**الخطوات:**
1. تخزين token صالح في localStorage
2. إرسال طلبات لعدة endpoints
3. التحقق من إضافة Authorization header لكل طلب
4. التحقق من صيغة header الصحيحة (Bearer token)

**النتيجة المتوقعة:**
- ✓ Authorization header موجود في كل طلب
- ✓ الصيغة صحيحة: `Bearer <token>`
- ✓ Token المستخدم هو نفسه المخزن

### Test 4: Authentication Flow Works

**الهدف:** التحقق من أن دورة المصادقة الكاملة تعمل

**الخطوات:**
1. محاكاة تسجيل الدخول (تخزين token)
2. إرسال طلب مصادق عليه
3. التحقق من نجاح الطلب مع token
4. محاكاة تسجيل الخروج (إزالة token)
5. إرسال طلب بدون token
6. التحقق من إرسال الطلب بدون Authorization header

**النتيجة المتوقعة:**
- ✓ تخزين token يعمل
- ✓ الطلبات المصادق عليها تعمل
- ✓ إزالة token يعمل
- ✓ الطلبات بدون مصادقة تعمل

## كيفية التشغيل

### الطريقة 1: عبر npm script

```bash
cd apps/client-web
npm run test:bug15:preservation
```

### الطريقة 2: عبر node مباشرة

```bash
cd apps/client-web
node src/api/__tests__/axios.bug15.preservation.test.js
```

### الطريقة 3: من workspace root

```bash
node apps/client-web/src/api/__tests__/axios.bug15.preservation.test.js
```

## النتيجة المتوقعة

### على الكود غير المُصلح (قبل التعديل)

```
╔════════════════════════════════════════════════════════════╗
║  Bug 15: Preservation Tests - Existing Functionality      ║
╚════════════════════════════════════════════════════════════╝

These tests verify that existing functionality continues to work
correctly BOTH before and after the bug fix is applied.

=== Preservation Test 1: Successful Requests Continue ===

Initial State:
  - Token: valid-jwt-token-abc123
  - Location: /dashboard

Simulating Successful API Request (200 OK)...
✓ Request succeeded
  - Status: 200
  - Data: {"messages":[{"id":1,"content":"Test message 1"},{"id":2,"content":"Test message 2"}]}

State After Successful Request:
  - Token: valid-jwt-token-abc123
  - Location: /dashboard

=== Test Results ===
Response received: true ✓
Authorization header added: true ✓
Token preserved: true ✓
No redirect: true ✓

✓ PRESERVATION TEST PASSED
  Successful requests continue to work correctly.

=== Preservation Test 2: Non-401 Errors Handled Correctly ===

Initial State:
  - Token: valid-jwt-token-xyz789
  - Location: /dashboard

Testing Error Code: 400
  ✓ Request failed as expected: Request failed with status code 400
    - Token preserved: ✓
    - No redirect: ✓

Testing Error Code: 403
  ✓ Request failed as expected: Request failed with status code 403
    - Token preserved: ✓
    - No redirect: ✓

Testing Error Code: 404
  ✓ Request failed as expected: Request failed with status code 404
    - Token preserved: ✓
    - No redirect: ✓

Testing Error Code: 500
  ✓ Request failed as expected: Request failed with status code 500
    - Token preserved: ✓
    - No redirect: ✓

Testing Error Code: 503
  ✓ Request failed as expected: Request failed with status code 503
    - Token preserved: ✓
    - No redirect: ✓

Final State:
  - Token: valid-jwt-token-xyz789
  - Location: /dashboard

=== Test Results ===
All non-401 errors handled correctly: true ✓

✓ PRESERVATION TEST PASSED
  Non-401 errors do not trigger logout.

=== Preservation Test 3: Authorization Header Added ===

Initial State:
  - Token: valid-jwt-token-header-test

Testing Endpoint: /api/messages
  - Authorization header present: ✓
  - Correct format: ✓
  - Header value: Bearer valid-jwt-token-header-test

Testing Endpoint: /api/users
  - Authorization header present: ✓
  - Correct format: ✓
  - Header value: Bearer valid-jwt-token-header-test

Testing Endpoint: /api/settings
  - Authorization header present: ✓
  - Correct format: ✓
  - Header value: Bearer valid-jwt-token-header-test

=== Test Results ===
All requests have correct Authorization header: true ✓

✓ PRESERVATION TEST PASSED
  Authorization header is added correctly to all requests.

=== Preservation Test 4: Authentication Flow Works ===

Step 1: User logs in
  - Token stored: login-jwt-token-flow-test
  - Redirected to: /dashboard

Step 2: Make authenticated request
  - Request succeeded: ✓
  - Authorization header present: ✓

Step 3: User logs out
  - Token removed: ✓
  - Redirected to: /login

Step 4: Make request without token
  - Request succeeded: ✓
  - No Authorization header: ✓

=== Test Results ===
Token storage works: ✓
Authenticated requests work: ✓
Token removal works: ✓
Unauthenticated requests work: ✓

✓ PRESERVATION TEST PASSED
  Complete authentication flow works correctly.

╔════════════════════════════════════════════════════════════╗
║                  PRESERVATION TEST SUMMARY                 ║
╚════════════════════════════════════════════════════════════╝

1. Successful Requests Continue: ✓ PASS
2. Non-401 Errors Handled: ✓ PASS
3. Authorization Header Added: ✓ PASS
4. Authentication Flow Works: ✓ PASS

Total: 4/4 tests passed

✓ ALL PRESERVATION TESTS PASSED
  Existing functionality is preserved correctly.
  The bug fix will not break existing features.

Exit Code: 0
```

### على الكود المُصلح (بعد التعديل)

**النتيجة:** نفس النتيجة تماماً! ✓

جميع الاختبارات يجب أن تنجح على الكود المُصلح أيضاً، لأن:
- الطلبات الناجحة لا تزال تعمل
- الأخطاء غير 401 لا تزال لا تسبب logout
- Authorization header لا يزال يُضاف بشكل صحيح
- دورة المصادقة لا تزال تعمل

**الفرق الوحيد:** بعد الإصلاح، استجابة 401 ستسبب logout تلقائي (وهذا ما نريده).

## التحقق اليدوي

إذا لم تتمكن من تشغيل الاختبارات تلقائياً، يمكنك التحقق يدوياً:

### 1. التحقق من الكود الحالي

افتح `apps/client-web/src/api/axios.js` وتحقق من السطور 28-29:

```javascript
// Response interceptor to handle unauthorized errors
apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response && error.response.status === 401) {
      // Handle logout or refresh token logic if needed
      console.warn('Unauthorized access - potential session expiry');
      // localStorage.removeItem('token');  // ← السطر 28: معلق
      // window.location.href = '/login';   // ← السطر 29: معلق
    }
    return Promise.reject(error);
  }
);
```

### 2. اختبار الطلبات الناجحة

1. افتح Developer Console في المتصفح
2. سجل الدخول إلى التطبيق
3. افتح Network tab
4. قم بأي عملية (مثل تحميل الرسائل)
5. تحقق من:
   - ✓ الطلب نجح (200 OK)
   - ✓ Authorization header موجود
   - ✓ Token لا يزال في localStorage
   - ✓ لا يوجد إعادة توجيه

### 3. اختبار الأخطاء غير 401

1. افتح Developer Console
2. في Console، نفذ:
   ```javascript
   fetch('/api/nonexistent', {
     headers: { 'Authorization': 'Bearer ' + localStorage.getItem('token') }
   })
   ```
3. تحقق من:
   - ✓ الطلب فشل (404 Not Found)
   - ✓ Token لا يزال في localStorage
   - ✓ لا يوجد إعادة توجيه

### 4. اختبار Authorization Header

1. افتح Developer Console
2. افتح Network tab
3. قم بأي طلب API
4. انقر على الطلب في Network tab
5. افتح Headers tab
6. تحقق من:
   - ✓ Request Headers يحتوي على `Authorization: Bearer <token>`
   - ✓ Token صحيح

## الملفات المتعلقة

- ✓ **Test File:** `apps/client-web/src/api/__tests__/axios.bug15.preservation.test.js`
- ✓ **Source File:** `apps/client-web/src/api/axios.js`
- ✓ **Package.json:** `apps/client-web/package.json` (script: `test:bug15:preservation`)
- ✓ **Instructions:** هذا الملف

## الخطوات التالية

1. ✓ **Task 15.1 COMPLETE:** Bug exploration test
2. ✓ **Task 15.2 COMPLETE:** Preservation tests (هذا الملف)
3. ⏳ **Task 15.3 PENDING:** Implement fix (uncomment lines 28-29)
4. ⏳ **Task 15.4 PENDING:** Verify fix works correctly

## ملاحظات مهمة

### لماذا يجب أن تنجح الاختبارات قبل وبعد الإصلاح؟

اختبارات الحفاظ تختبر **الوظائف الموجودة** التي يجب أن تبقى كما هي:

- ✓ الطلبات الناجحة (200 OK) → لا تتأثر بالإصلاح
- ✓ الأخطاء غير 401 (400, 403, 404, 500) → لا تتأثر بالإصلاح
- ✓ Authorization header → لا يتأثر بالإصلاح
- ✓ دورة المصادقة → لا تتأثر بالإصلاح

**الإصلاح يؤثر فقط على:** استجابة 401 (وهذا ما نريده)

### الفرق بين Exploration Test و Preservation Test

| Aspect | Exploration Test | Preservation Test |
|--------|------------------|-------------------|
| **الهدف** | تأكيد وجود Bug | تأكيد عدم كسر الوظائف |
| **قبل الإصلاح** | ✓ PASS (Bug موجود) | ✓ PASS (الوظائف تعمل) |
| **بعد الإصلاح** | ✗ FAIL (Bug مُصلح) | ✓ PASS (الوظائف لا تزال تعمل) |
| **يختبر** | السلوك الخاطئ | السلوك الصحيح |

## الخلاصة

اختبارات الحفاظ جاهزة وتتحقق من:

1. ✓ الطلبات الناجحة تستمر بالعمل
2. ✓ الأخطاء غير 401 لا تسبب logout
3. ✓ Authorization header يُضاف بشكل صحيح
4. ✓ دورة المصادقة الكاملة تعمل

**الحالة:** ✓ READY TO RUN

**التوقع:** جميع الاختبارات يجب أن تنجح على الكود غير المُصلح وعلى الكود المُصلح.
