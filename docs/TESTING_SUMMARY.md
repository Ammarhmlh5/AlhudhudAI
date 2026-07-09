# ملخص هيكل الاختبارات - ALHudhudAI

**تاريخ الإنشاء:** 24 مايو 2026  
**مشروع:** ALHudhudAI SMS Gateway Platform

---

## نظرة عامة

تم إعداد بنية شاملة للاختبارات تشمل:
- **Backend (Go):** اختبارات الوحدة، التكامل، Property-Based
- **Frontend (React):** اختبارات الوحدة مع Vitest، E2E مع Playwright
- **CI/CD:** GitHub Actions للتشغيل التلقائي

---

## هيكل الملفات

### Backend Tests

```
services/backend/
├── internal/
│   ├── balance/
│   │   ├── balance.go
│   │   └── tests/
│   │       └── balance_unit_test.go      # اختبارات نظام الرصيد
│   ├── channel/
│   │   ├── channel.go
│   │   └── channel_test.go               # اختبارات القنوات
│   ├── messages/
│   │   ├── messages.go
│   │   └── message_test.go               # اختبارات الرسائل
│   ├── auth/
│   │   ├── auth.go
│   │   └── auth_test.go                  # اختبارات المصادقة
│   ├── tenant/
│   │   ├── tenant.go
│   │   └── tenant_test.go                # اختبارات المستأجرين
│   └── db/
│       ├── db.go
│       ├── test_db.go                    # قاعدة بيانات الاختبار
│       └── mock_repository.go            # Mock repositories
└── go.mod
```

### Frontend Tests

```
apps/admin-web/
├── src/
│   ├── components/
│   │   └── Button/
│   │       ├── Button.tsx
│   │       └── Button.test.tsx           # مثال اختبار مكون
│   ├── api/
│   │   ├── index.ts
│   │   └── __tests__/
│   │       └── api.test.ts               # اختبارات API
│   ├── e2e/
│   │   ├── auth.spec.ts                  # اختبارات E2E للمصادقة
│   │   └── messaging.spec.ts             # اختبارات E2E للرسائل
│   └── test/
│       └── setup.ts                      # إعدادات الاختبارات
├── vitest.config.ts
├── playwright.config.ts
└── package.json
```

### CI/CD

```
.github/workflows/
└── test.yml                              # GitHub Actions workflow
```

---

## أدوات الاختبار

### Backend (Go)

| الأداة | الغرض | الاستخدام |
|--------|-------|-----------|
| **testify** | Unit testing assertions | `assert.NoError(t, err)` |
| **sqlmock** | Mock database | `NewMockBalanceRepository()` |
| **miniredis** | Mock Redis | اختبارات الـ caching |
| **gopter** | Property-based testing | `TestBalanceManager_PropertyBased` |
| **rapid** | Property-based testing | اختبارات سريعة |

### Frontend (React)

| الأداة | الغرض | الاستخدام |
|--------|-------|-----------|
| **Vitest** | Unit testing | `npm run test` |
| **React Testing Library** | Component testing | `render(<Component />)` |
| **Playwright** | E2E testing | `npm run test:e2e` |

---

## تشغيل الاختبارات

### Backend

```bash
# تشغيل جميع الاختبارات
cd services/backend
go test ./... -v

# تشغيل اختبارات محددة
go test ./internal/balance/... -v

# تشغيل مع التغطية
go test ./... -coverprofile=coverage.out

# تشغيل Property-based tests
go test ./... -run TestBalanceManager_PropertyBased

# تشغيل في وضع المراقبة
go test ./... -watch
```

### Frontend

```bash
# تشغيل الاختبارات
cd apps/admin-web
npm run test

# تشغيل مرة واحدة (بدون watch)
npm run test:run

# تشغيل مع التغطية
npm run test:coverage

# تشغيل E2E tests
npm run test:e2e
```

### سكريبت شامل

```powershell
# تشغيل جميع الاختبارات
.\run-tests.ps1 all

# تشغيل Backend فقط
.\run-tests.ps1 backend

# تشغيل Frontend فقط
.\run-tests.ps1 frontend

# تشغيل مع التغطية
.\run-tests.ps1 coverage
```

---

## أمثلة الاختبارات

### 1. اختبار وحدة (Backend)

```go
func TestBalanceManager_DeductBalance_SufficientFunds(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	amount := decimal.NewFromInt(100)
	
	mockRepo := new(MockBalanceRepository)
	mockRepo.On("DeductBalance", mock.Anything, tenantID, amount).Return(nil)
	
	manager := balance.NewManager(mockRepo)
	
	// Act
	err := manager.DeductBalance(context.Background(), tenantID, amount)
	
	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}
```

### 2. اختبار مكون (Frontend)

```tsx
it('renders dashboard with tenant statistics', async () => {
	const mockStats = {
		totalTenants: 150,
		activeTenants: 120,
		messagesToday: 50000,
		revenue: 25000.50
	}
	vi.spyOn(api, 'getTenantStats').mockResolvedValue(mockStats)

	render(<Dashboard />)

	await waitFor(() => {
		expect(screen.getByText('150')).toBeInTheDocument()
		expect(screen.getByText('120')).toBeInTheDocument()
	})
})
```

### 3. اختبار E2E

```typescript
test('admin can send bulk messages', async ({ page }) => {
	await page.click('a[href="/messaging"]')
	await page.fill('textarea[name="message"]', 'Test message')
	await page.selectOption('select[name="channel"]', 'direct')
	await page.click('button:has-text("Send Now")')
	
	await expect(page.locator('text=Messages queued successfully')).toBeVisible()
})
```

### 4. Property-Based Test

```go
props.Property("balance never goes negative", convey.Settle(func(gs *gopter.State) {
	tenantID := uuid.New()
	initialBalance := decimal.NewFromInt(1000)
	
	mockRepo := new(MockBalanceRepository)
	mockRepo.On("GetBalance", mock.Anything, tenantID).Return(initialBalance, nil)
	
	manager := balance.NewManager(mockRepo)
	
	// Generate random operations
	numOps := gs.GetInt(1, 10)
	
	for i := 0; i < numOps; i++ {
		amount := decimal.NewFromFloat(gs.GetFloat64(0, 500))
		manager.DeductBalance(context.Background(), tenantID, amount)
	}
	
	// Final balance should be >= 0
	finalBalance, _ := manager.GetBalance(context.Background(), tenantID)
	assert.True(t, finalBalance.GreaterThanOrEqual(decimal.Zero))
}))
```

---

## متطلبات التغطية

| نوع الكود | الحد الأدنى | الهدف |
|-----------|-------------|-------|
| Business Logic | 80% | 90% |
| API Handlers | 75% | 85% |
| Middleware | 70% | 80% |
| Frontend Components | 70% | 80% |

---

## GitHub Actions Workflow

### المراحل:

1. **backend-test** - تشغيل اختبارات Backend
   - إعداد PostgreSQL و Redis
   - تشغيل Migrations
   - تشغيل Unit Tests
   - تشغيل Integration Tests
   - تشغيل Property-Based Tests

2. **frontend-test** - تشغيل اختبارات Frontend
   - تثبيت Dependencies
   - تشغيل Unit Tests مع التغطية

3. **e2e-test** - تشغيل اختبارات E2E
   - تشغيل Backend
   - تشغيل Frontend
   - تشغيل Playwright tests

4. **test-summary** - ملخص النتائج

---

## أفضل الممارسات

### ✅ افعل

- اكتب الاختبار قبل ال��ود (TDD)
- استخدم descriptive names للاختبارات
- اختبر الحالات الحدية (null, empty, large values)
- اختبر مسارات الأخطاء
- استخدم mocks للخدمات الخارجية
- اجعل الاختبارات سريعة (< 50ms)

### ❌ لا تفعل

- تخطي كتابة الاختبارات
- اختبار تفاصيل التنفيذ
- اختبارات هشة (تتغير بسهولة)
- اختبارات مترابطة
- تجاهل التغطية المنخفضة

---

## الأوامر السريعة

```powershell
# التحقق من التغطية
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out

# تشغيل اختبار محدد
go test ./internal/balance -run TestBalanceManager_DeductBalance -v

# تشغيل جميع اختبارات التكامل
go test ./tests/... -v

# تشغيل Property-based tests
go test ./... -run TestBalanceManager_PropertyBased -v

# التحقق منlinting
cd apps/admin-web
npm run lint
```

---

## الموارد

- **الدليل الكامل:** [TDD_GUIDE.md](./TDD_GUIDE.md)
- **مراجعة الكود:** [.kiro/CODE_REVIEW_REPORT.md](../.kiro/CODE_REVIEW_REPORT.md)
- **مراجعة قاعدة البيانات:** [.kiro/DATABASE_REVIEW_REPORT.md](../.kiro/DATABASE_REVIEW_REPORT.md)

---

**تم إنشاء هذا الملف:** 24 مايو 2026