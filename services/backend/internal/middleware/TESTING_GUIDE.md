# دليل اختبار Tenant Isolation Middleware

## نظرة عامة

هذا الدليل يشرح كيفية تشغيل واختبار Tenant Isolation Middleware و Cross-Tenant Prevention Middleware.

## الاختبارات المتوفرة

### 1. Unit Tests للـ Cross-Tenant Prevention
**الملف**: `cross_tenant_prevention_test.go`

**الاختبارات**:
- ✅ الوصول لنفس الشركة (Same Tenant)
- ✅ الوصول لشركة مختلفة (Different Tenant)
- ✅ وصول Super Admin
- ✅ وصول Staff
- ✅ فحص Query Parameters
- ✅ فحص Request Body
- ✅ tenant_id مفقود
- ✅ tenant_id بصيغة غير صحيحة
- ✅ تسجيل المحاولات

### 2. Integration Tests للعزل
**الملف**: `tenant_isolation_integration_test.go`

**الاختبارات**:
- ✅ Property 16: Tenant Isolation في قاعدة البيانات
- ✅ إنشاء موارد مع العزل
- ✅ تحديث موارد مع العزل
- ✅ حذف موارد مع العزل
- ✅ وصول Super Admin لجميع الشركات

## تشغيل الاختبارات

### تشغيل جميع الاختبارات

```bash
cd services/backend
go test ./internal/middleware -v
```

### تشغيل اختبارات محددة

#### Unit Tests فقط
```bash
go test ./internal/middleware -v -run TestPreventCrossTenantAccess
```

#### Integration Tests فقط
```bash
go test ./internal/middleware -v -run TestIntegration
```

#### Property Tests فقط
```bash
go test ./internal/middleware -v -run TestProperty16
```

### تشغيل مع تغطية الكود (Code Coverage)

```bash
go test ./internal/middleware -v -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

ثم افتح `coverage.html` في المتصفح لرؤية التغطية.

### تشغيل مع Race Detector

```bash
go test ./internal/middleware -v -race
```

## متطلبات الاختبار

### المكتبات المطلوبة

```bash
go get github.com/stretchr/testify/assert
go get github.com/stretchr/testify/require
go get github.com/gin-gonic/gin
go get github.com/google/uuid
go get gorm.io/gorm
go get gorm.io/driver/sqlite
```

### قاعدة البيانات

الاختبارات تستخدم SQLite in-memory database، لذلك لا تحتاج لإعداد قاعدة بيانات خارجية.

## فهم الاختبارات

### Property 16: Tenant Isolation في قاعدة البيانات

هذا الاختبار يتحقق من:

1. **عزل القراءة**: كل شركة ترى فقط مواردها
2. **عزل الكتابة**: كل شركة تستطيع إنشاء موارد لنفسها فقط
3. **منع Cross-Tenant Access**: لا يمكن لشركة الوصول لموارد شركة أخرى
4. **استثناءات Super Admin**: Super Admin يمكنه الوصول لجميع الشركات

### مثال على الاختبار

```go
func TestProperty16_TenantIsolationInDatabase(t *testing.T) {
    // إعداد قاعدة البيانات
    db := setupIntegrationTestDB(t)
    
    // إنشاء شركتين
    tenant1ID, tenant2ID := createTestTenants(t, db)
    
    // إنشاء موارد لكل شركة
    createTestResources(t, db, tenant1ID, tenant2ID)
    
    // إعداد Router
    router := setupIntegrationTestRouter(db)
    
    // الاختبار 1: Tenant1 يرى فقط موارده
    t.Run("Tenant1 can only see their own resources", func(t *testing.T) {
        req, _ := http.NewRequest("GET", fmt.Sprintf("/api/tenants/%s/resources", tenant1ID.String()), nil)
        req.Header.Set("X-Tenant-ID", tenant1ID.String())
        req.Header.Set("X-User-Type", "TENANT")
        
        w := httptest.NewRecorder()
        router.ServeHTTP(w, req)
        
        assert.Equal(t, http.StatusOK, w.Code)
        // التحقق من أن جميع الموارد تنتمي لـ tenant1
    })
    
    // الاختبار 2: Tenant1 لا يستطيع الوصول لموارد Tenant2
    t.Run("Tenant1 cannot access Tenant2 resources", func(t *testing.T) {
        req, _ := http.NewRequest("GET", fmt.Sprintf("/api/tenants/%s/resources", tenant2ID.String()), nil)
        req.Header.Set("X-Tenant-ID", tenant1ID.String())
        req.Header.Set("X-User-Type", "TENANT")
        
        w := httptest.NewRecorder()
        router.ServeHTTP(w, req)
        
        assert.Equal(t, http.StatusForbidden, w.Code)
        // التحقق من رسالة الخطأ
    })
}
```

## استكشاف الأخطاء

### الخطأ: "Failed to connect to test database"

**الحل**: تأكد من تثبيت SQLite driver:
```bash
go get gorm.io/driver/sqlite
```

### الخطأ: "Failed to migrate test database"

**الحل**: تأكد من أن جميع Models محددة بشكل صحيح في `AutoMigrate()`.

### الخطأ: "Test timeout"

**الحل**: زد من timeout الاختبار:
```bash
go test ./internal/middleware -v -timeout 30s
```

### الخطأ: "Race condition detected"

**الحل**: هذا يعني وجود مشكلة في الكود. راجع الكود وأصلح المشكلة.

## أفضل الممارسات

### 1. استخدم t.Run() للاختبارات الفرعية

```go
func TestSomething(t *testing.T) {
    t.Run("scenario 1", func(t *testing.T) {
        // test code
    })
    
    t.Run("scenario 2", func(t *testing.T) {
        // test code
    })
}
```

### 2. استخدم require للتحقق من الشروط الأساسية

```go
// استخدم require للأخطاء الحرجة (توقف الاختبار)
require.NoError(t, err, "Failed to setup database")

// استخدم assert للتحققات العادية (يستمر الاختبار)
assert.Equal(t, expected, actual)
```

### 3. نظف البيانات بعد كل اختبار

```go
func TestSomething(t *testing.T) {
    db := setupTestDB(t)
    
    // Cleanup
    defer func() {
        // Clean up test data
    }()
    
    // Test code
}
```

### 4. استخدم Table-Driven Tests للحالات المتشابهة

```go
func TestValidation(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected bool
    }{
        {"valid UUID", "123e4567-e89b-12d3-a456-426614174000", true},
        {"invalid UUID", "invalid", false},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := validate(tt.input)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

## التكامل المستمر (CI/CD)

### GitHub Actions

```yaml
name: Test Middleware

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      
      - name: Set up Go
        uses: actions/setup-go@v2
        with:
          go-version: 1.21
      
      - name: Install dependencies
        run: |
          cd services/backend
          go mod download
      
      - name: Run tests
        run: |
          cd services/backend
          go test ./internal/middleware -v -race -coverprofile=coverage.out
      
      - name: Upload coverage
        uses: codecov/codecov-action@v2
        with:
          files: ./services/backend/coverage.out
```

## المراجع

- [Testing in Go](https://golang.org/pkg/testing/)
- [Testify Documentation](https://github.com/stretchr/testify)
- [Gin Testing](https://github.com/gin-gonic/gin#testing)
- [GORM Testing](https://gorm.io/docs/testing.html)

## الخلاصة

الاختبارات الشاملة تضمن:
- ✅ عزل تام بين الشركات
- ✅ منع Cross-Tenant Access
- ✅ عمل صحيح للاستثناءات (Super Admin، Staff)
- ✅ تسجيل شامل للمحاولات
- ✅ رسائل خطأ واضحة

قم بتشغيل الاختبارات بانتظام للتأكد من استمرار عمل العزل بشكل صحيح.
