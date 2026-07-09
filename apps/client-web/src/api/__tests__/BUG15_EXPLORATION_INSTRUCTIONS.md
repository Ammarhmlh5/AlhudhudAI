# Bug 15 Exploration Test - تعليمات التشغيل

## نظرة عامة

هذا الاختبار يستكشف **Bug 15: No Auto-Logout on 401** - عدم تسجيل الخروج التلقائي عند استجابة 401.

**Validates: Requirements 1.15, 2.15**

## شرط الخطأ (Bug Condition)

**Requirement 1.15:**
```
WHEN a 401 Unauthorized response is received in axios.js
THEN the system does not automatically log out the user,
leaving them in an inconsistent state
```

## السلوك المتوقع (Expected Behavior)

**Requirement 2.15:**
```
WHEN a 401 Unauthorized response is received
THEN the system SHALL clear the authentication token,
redirect to the login page, and optionally show a notification
```

## الكود الحالي (Buggy Code)

في الملف `apps/client-web/src/api/axios.js`، السطور 26-30:

```javascript
apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response && error.response.status === 401) {
      // Handle logout or refresh token logic if needed
      console.warn('Unauthorized access - potential session expiry');
      // localStorage.removeItem('token');  // ← معطل (BUG!)
      // window.location.href = '/login';   // ← معطل (BUG!)
    }
    return Promise.reject(error);
  }
);
```

**المشكلة:** السطران المسؤولان عن تسجيل الخروج معطلان (commented out).

## كيفية تشغيل الاختبار

### الطريقة 1: عبر npm (موصى بها)

```bash
cd apps/client-web
npm run test:bug15
```

### الطريقة 2: عبر node مباشرة

```bash
cd apps/client-web
node src/api/__tests__/axios.bug15.test.js
```

### الطريقة 3: من المجلد الجذر

```bash
node apps/client-web/src/api/__tests__/axios.bug15.test.js
```

## النتائج المتوقعة

### على الكود غير المصلح (UNFIXED CODE):

```
=== Bug 15 Exploration Test: No Auto-Logout on 401 ===

Initial State:
  - Token in localStorage: test-jwt-token-12345
  - Current location: /dashboard

Simulating 401 Unauthorized Response...

Received Error: Request failed with status code 401

State After 401 Response:
  - Token in localStorage: test-jwt-token-12345
  - Current location: /dashboard

=== Test Results ===
Token still exists: true ✓ (Bug Confirmed)
Not redirected to login: true ✓ (Bug Confirmed)

✓ EXPLORATION TEST PASSED: Bug condition confirmed!
  The system does NOT automatically log out on 401.
  This test will FAIL after the bug is fixed.
```

**Exit Code: 1** (يشير إلى وجود الخطأ)

### بعد إصلاح الكود (FIXED CODE):

```
=== Bug 15 Exploration Test: No Auto-Logout on 401 ===

Initial State:
  - Token in localStorage: test-jwt-token-12345
  - Current location: /dashboard

Simulating 401 Unauthorized Response...

Received Error: Request failed with status code 401

State After 401 Response:
  - Token in localStorage: (removed)
  - Current location: /login

=== Test Results ===
Token still exists: false ✗ (Bug Not Found)
Not redirected to login: false ✗ (Bug Not Found)

✗ EXPLORATION TEST FAILED: Bug not reproduced!
  The system appears to handle 401 correctly.
  Either the bug is already fixed or the test needs adjustment.

Status: BUG FIXED - Code is working correctly
```

**Exit Code: 0** (يشير إلى إصلاح الخطأ)

## استراتيجية الاختبار

الاختبار يحتوي على جزئين:

### 1. Bug Condition Test (اختبار شرط الخطأ)
- يحاكي الكود الحالي (مع التعليقات)
- يتحقق من أن المستخدم **لا يتم** تسجيل خروجه
- **يجب أن ينجح** على الكود غير المصلح

### 2. Expected Behavior Test (اختبار السلوك المتوقع)
- يحاكي الكود المصلح (بدون تعليقات)
- يتحقق من أن المستخدم **يتم** تسجيل خروجه
- **يجب أن ينجح** على الكود المصلح

## الإصلاح المطلوب

في الملف `apps/client-web/src/api/axios.js`، قم بإزالة التعليقات من السطرين 28-29:

```javascript
// قبل الإصلاح:
if (error.response && error.response.status === 401) {
  console.warn('Unauthorized access - potential session expiry');
  // localStorage.removeItem('token');
  // window.location.href = '/login';
}

// بعد الإصلاح:
if (error.response && error.response.status === 401) {
  console.warn('Unauthorized access - logging out user');
  localStorage.removeItem('token');
  window.location.href = '/login';
}
```

## ملاحظات مهمة

1. **هذا اختبار استكشافي (Exploration Test):**
   - الهدف هو تأكيد وجود الخطأ
   - يجب أن **ينجح** على الكود غير المصلح (يؤكد وجود الخطأ)
   - يجب أن **يفشل** بعد إصلاح الكود (يؤكد الإصلاح)

2. **Exit Codes:**
   - `1`: الخطأ موجود (حالة متوقعة قبل الإصلاح)
   - `0`: الخطأ مصلح (حالة متوقعة بعد الإصلاح)
   - `2`: نتائج غير متوقعة (يحتاج مراجعة)
   - `3`: فشل تنفيذ الاختبار

3. **التبعيات:**
   - الاختبار يستخدم `axios` المثبت بالفعل
   - لا يحتاج إلى تبعيات إضافية
   - يعمل مع Node.js مباشرة

## الخطوات التالية

بعد تشغيل هذا الاختبار:

1. **إذا نجح الاختبار (Exit Code 1):**
   - الخطأ موجود ومؤكد ✓
   - انتقل إلى Task 15.2: تطبيق الإصلاح

2. **إذا فشل الاختبار (Exit Code 0):**
   - الخطأ مصلح بالفعل
   - راجع الكود للتأكد من الإصلاح

3. **إذا كانت النتائج غير متوقعة (Exit Code 2):**
   - راجع الاختبار والكود
   - قد يحتاج الاختبار إلى تعديل

## الملفات ذات الصلة

- **الكود المعطوب:** `apps/client-web/src/api/axios.js`
- **الاختبار:** `apps/client-web/src/api/__tests__/axios.bug15.test.js`
- **المتطلبات:** `.kiro/specs/system-critical-bugfixes/bugfix.md`
- **التصميم:** `.kiro/specs/system-critical-bugfixes/design.md`
- **المهام:** `.kiro/specs/system-critical-bugfixes/tasks.md`

## نفس المشكلة في admin-web

**ملاحظة:** نفس المشكلة موجودة في `apps/admin-web` لكن لا يوجد ملف `axios.js` هناك حالياً. قد يحتاج إلى إنشاء ملف مشابه أو استخدام fetch API مع معالجة 401.
