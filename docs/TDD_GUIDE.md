# دليل التطوير المدفوع بالاختبارات (TDD) - ALHudhudAI

**تاريخ الإنشاء:** 24 مايو 2026  
**مشروع:** ALHudhudAI SMS Gateway Platform  
**إصدار:** 1.0

---

## نظرة عامة

هذا الدليل يوثق منهجية التطوير المدفوع بالاختبارات المخصصة لمشروع ALHudhudAI. يهدف إلى ضمان جودة الكود وموثوقيته من خلال كتابة الاختبارات قبل كتابة الكود الفعلي.

## لماذا TDD؟

### الفوائد
- **كود نظيف** - تصميم أفضل من البداية
- **تغطية عالية** - 80%+ تغطية الاختبارات
- **وثيقة حية** - الاختبارات توثق السلوك المتوقع
- **ثقة في التغييرات** - refactoring آمن
- **أخطاء مبكرة** - اكتشاف المشاكل قبل الإنتاج

---

## هيكل الاختبارات

### Backend (Go)

```
services/backend/
├── internal/
│   ├── balance/
│   │   ├── balance.go
│   │   ├── balance_test.go        # Unit tests
│   │   └── balance_integration_test.go
│   ├── api/
│   │   ├── handlers/
│   │   │   ├── message_handler.go
│   │   │   └── message_handler_test.go
│   │   └── middleware/
│   │       ├── auth_test.go
│   │       └── rate_limit_test.go
│   └── ...
├── tests/
│   ├── e2e/
│   │   ├── messaging_e2e_test.go
│   │   └── auth_e2e_test.go
│   └── integration/
│       └── db_integration_test.go
└── go.mod
```

### Frontend (React)

```
apps/admin-web/
├── src/
│   ├── components/
│   │   ├── Button/
│   │   │   ├── Button.tsx
│   │   │   ├── Button.test.tsx
│   │   │   └── Button.stories.tsx
│   │   └── Dashboard/
│   │       ├── Dashboard.tsx
│   │       └── Dashboard.test.tsx
│   ├── pages/
│   │   ├── Overview/
│   │   │   ├── Overview.tsx
│   │   │   └── Overview.test.tsx
│   │   └── ...
│   └── api/
│       └── __tests__/
│           └── api.test.ts
└── package.json
```

---

## أدوات الاختبار المتاحة

### Backend (Go)

| الأداة | الغرض | الإصدار |
|--------|-------|---------|
| **gopter** | Property-based testing | v0.2.11 |
| **rapid** | Property-based testing | v1.2.0 |
| **testify** | Unit testing assertions | v1.11.1 |
| **sqlmock** | Mock database | v1.5.2 |
| **miniredis** | Mock Redis | v2.36.1 |
| **gomock** | Mock interfaces | v0.6.0 |

### Frontend (React)

| الأداة | الغرض | الإصدار |
|--------|-------|---------|
| **Vitest** | Unit testing | Latest |
| **React Testing Library** | Component testing | Latest |
| **Playwright** | E2E testing | Latest |

---

## أنواع الاختبارات

### 1. اختبارات الوحدة (Unit Tests)

#### Backend - Go

```go
// services/backend/internal/balance/balance_test.go

package balance_test

import (
	"context"
	"testing"
	"time"

	"github.com/al-hudhud/backend/internal/balance"
	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/tenant"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockBalanceRepository - Mock for database operations
type MockBalanceRepository struct {
	mock.Mock
}

func (m *MockBalanceRepository) GetBalance(ctx context.Context, tenantID uuid.UUID) (decimal.Decimal, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).(decimal.Decimal), args.Error(1)
}

func (m *MockBalanceRepository) DeductBalance(ctx context.Context, tenantID uuid.UUID, amount decimal.Decimal) error {
	args := m.Called(ctx, tenantID, amount)
	return args.Error(0)
}

func (m *MockBalanceRepository) AddBalance(ctx context.Context, tenantID uuid.UUID, amount decimal.Decimal) error {
	args := m.Called(ctx, tenantID, amount)
	return args.Error(0)
}

func TestBalanceManager_GetBalance_Success(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	expectedBalance := decimal.NewFromInt(1000)
	
	mockRepo := new(MockBalanceRepository)
	mockRepo.On("GetBalance", mock.Anything, tenantID).Return(expectedBalance, nil)
	
	manager := balance.NewManager(mockRepo)
	
	// Act
	result, err := manager.GetBalance(context.Background(), tenantID)
	
	// Assert
	assert.NoError(t, err)
	assert.True(t, result.Equal(expectedBalance))
	mockRepo.AssertExpectations(t)
}

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

func TestBalanceManager_DeductBalance_InsufficientFunds(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	amount := decimal.NewFromInt(10000) // More than available
	
	mockRepo := new(MockBalanceRepository)
	mockRepo.On("DeductBalance", mock.Anything, tenantID, amount).Return(balance.ErrInsufficientFunds)
	
	manager := balance.NewManager(mockRepo)
	
	// Act
	err := manager.DeductBalance(context.Background(), tenantID, amount)
	
	// Assert
	assert.Error(t, err)
	assert.Equal(t, balance.ErrInsufficientFunds, err)
	mockRepo.AssertExpectations(t)
}

func TestBalanceManager_DeductBalance_Atomicity(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	amount := decimal.NewFromInt(50)
	
	mockRepo := new(MockBalanceRepository)
	mockRepo.On("DeductBalance", mock.Anything, tenantID, amount).Return(nil)
	
	manager := balance.NewManager(mockRepo)
	
	// Act - Call multiple times concurrently
	errChan := make(chan error, 10)
	for i := 0; i < 10; i++ {
		go func() {
			err := manager.DeductBalance(context.Background(), tenantID, amount)
			errChan <- err
		}()
	}
	
	// Assert - All should succeed (atomic operations)
	for i := 0; i < 10; i++ {
		assert.NoError(t, <-errChan)
	}
}
```

#### Frontend - React (Vitest)

```tsx
// apps/admin-web/src/components/Dashboard/Dashboard.test.tsx

import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { Dashboard } from './Dashboard'
import * as api from '../../api/tenants'

// Mock API calls
vi.mock('../../api/tenants')

describe('Dashboard Component', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders dashboard with tenant statistics', async () => {
    // Arrange
    const mockStats = {
      totalTenants: 150,
      activeTenants: 120,
      messagesToday: 50000,
      revenue: 25000.50
    }
    vi.spyOn(api, 'getTenantStats').mockResolvedValue(mockStats)

    // Act
    render(<Dashboard />)

    // Assert
    await waitFor(() => {
      expect(screen.getByText('150')).toBeInTheDocument()
      expect(screen.getByText('120')).toBeInTheDocument()
      expect(screen.getByText('50,000')).toBeInTheDocument()
    })
  })

  it('shows loading state initially', () => {
    // Arrange
    vi.spyOn(api, 'getTenantStats').mockImplementation(() => 
      new Promise(resolve => setTimeout(resolve, 1000))
    )

    // Act
    render(<Dashboard />)

    // Assert
    expect(screen.getByTestId('loading-spinner')).toBeInTheDocument()
  })

  it('handles API error gracefully', async () => {
    // Arrange
    vi.spyOn(api, 'getTenantStats').mockRejectedValue(new Error('Network error'))

    // Act
    render(<Dashboard />)

    // Assert
    await waitFor(() => {
      expect(screen.getByText('Failed to load statistics')).toBeInTheDocument()
    })
  })

  it('refreshes data when refresh button clicked', async () => {
    // Arrange
    const mockStats = { totalTenants: 100, activeTenants: 80, messagesToday: 30000, revenue: 15000 }
    const getStats = vi.spyOn(api, 'getTenantStats').mockResolvedValue(mockStats)
    
    render(<Dashboard />)
    
    // Wait for initial load
    await waitFor(() => expect(getStats).toHaveBeenCalledTimes(1))
    
    // Act
    await userEvent.click(screen.getByRole('button', { name: /refresh/i }))
    
    // Assert
    await waitFor(() => expect(getStats).toHaveBeenCalledTimes(2))
  })
})
```

### 2. اختبارات التكامل (Integration Tests)

```go
// services/backend/internal/api/handlers/message_handler_integration_test.go

package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/al-hudhud/backend/internal/api"
	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/tenant"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// MessageHandlerIntegrationSuite - Integration test suite
type MessageHandlerIntegrationSuite struct {
	suite.Suite
	router     *gin.Engine
	db         *db.TestDB
	tenantID   uuid.UUID
	testUserID uuid.UUID
}

func (s *MessageHandlerIntegrationSuite) SetupSuite() {
	// Setup test database
	s.db = db.NewTestDB()
	s.db.Migrate()
	
	// Create test tenant
	s.tenantID = s.db.CreateTestTenant()
	s.testUserID = s.db.CreateTestUser(s.tenantID)
	
	// Setup router
	s.router = api.SetupRouter(s.db)
}

func (s *MessageHandlerIntegrationSuite) TearDownSuite() {
	s.db.Cleanup()
}

func (s *MessageHandlerIntegrationSuite) TestSendMessage_Success() {
	// Arrange
	message := api.SendMessageRequest{
		To:      "+966501234567",
		Content: "Test message",
		Channel: "direct",
	}
	body, _ := json.Marshal(message)
	
	req, _ := http.NewRequest("POST", "/api/v1/messages/send", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", s.tenantID.String())
	
	// Act
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	
	// Assert
	assert.Equal(s.T(), http.StatusOK, w.Code)
	
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(s.T(), "success", response["status"])
	assert.NotEmpty(s.T(), response["message_id"])
}

func (s *MessageHandlerIntegrationSuite) TestSendMessage_InvalidRecipient() {
	// Arrange
	message := api.SendMessageRequest{
		To:      "invalid-phone",
		Content: "Test message",
		Channel: "direct",
	}
	body, _ := json.Marshal(message)
	
	req, _ := http.NewRequest("POST", "/api/v1/messages/send", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", s.tenantID.String())
	
	// Act
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	
	// Assert
	assert.Equal(s.T(), http.StatusBadRequest, w.Code)
}

func (s *MessageHandlerIntegrationSuite) TestSendMessage_InsufficientBalance() {
	// Arrange - Set tenant balance to 0
	s.db.SetTenantBalance(s.tenantID, 0)
	
	message := api.SendMessageRequest{
		To:      "+966501234567",
		Content: "Test message",
		Channel: "direct",
	}
	body, _ := json.Marshal(message)
	
	req, _ := http.NewRequest("POST", "/api/v1/messages/send", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", s.tenantID.String())
	
	// Act
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	
	// Assert
	assert.Equal(s.T(), http.StatusPaymentRequired, w.Code)
}

func (s *MessageHandlerIntegrationSuite) TestGetMessageHistory_Pagination() {
	// Arrange - Create test messages
	for i := 0; i < 25; i++ {
		s.db.CreateTestMessage(s.tenantID)
	}
	
	req, _ := http.NewRequest("GET", "/api/v1/messages?page=1&limit=10", nil)
	req.Header.Set("X-Tenant-ID", s.tenantID.String())
	
	// Act
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	
	// Assert
	assert.Equal(s.T(), http.StatusOK, w.Code)
	
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	
	messages := response["data"].([]interface{})
	assert.Equal(s.T(), 10, len(messages))
	
	pagination := response["pagination"].(map[string]interface{})
	assert.Equal(s.T(), 25, pagination["total"])
	assert.Equal(s.T(), 3, pagination["total_pages"])
}
```

### 3. اختبارات Property-Based (Go)

```go
// services/backend/internal/balance/balance_property_test.go

package balance_test

import (
	"context"
	"testing"

	"github.com/al-hudhud/backend/internal/balance"
	"github.com/al-hudhud/backend/internal/tenant"
	"github.com/google/uuid"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/commands"
	"github.com/leanovate/gopter/convey"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

// BalanceSystemModel - Model for property-based testing
type BalanceSystemModel struct {
	tenantID    uuid.UUID
	initialBalance decimal.Decimal
	operations   []balanceOperation
}

type balanceOperation struct {
	opType string // "add", "deduct", "get"
	amount  decimal.Decimal
}

func TestBalanceManager_PropertyBased(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	parameters.MaxSize = 10

	props := gopter.NewProperties(parameters)

	props.Property("balance never goes negative", convey.Settle(func(gs *gopter.State) {
		// Setup
		tenantID := uuid.New()
		initialBalance := decimal.NewFromInt(1000)
		
		mockRepo := new(MockBalanceRepository)
		mockRepo.On("GetBalance", mock.Anything, tenantID).Return(initialBalance, nil)
		mockRepo.On("DeductBalance", mock.Anything, tenantID, mock.Anything).Return(nil)
		
		manager := balance.NewManager(mockRepo)
		
		// Generate random operations
		numOps := gs.GetInt(1, 10)
		totalDeducted := decimal.NewFromInt(0)
		
		for i := 0; i < numOps; i++ {
			amount := decimal.NewFromFloat(gs.GetFloat64(0, 500))
			
			// Try to deduct
			err := manager.DeductBalance(context.Background(), tenantID, amount)
			
			// If no error, track the deduction
			if err == nil {
				totalDeducted = totalDeducted.Add(amount)
			}
		}
		
		// Final balance should be >= 0
		finalBalance := initialBalance.Sub(totalDeducted)
		assert.True(t, finalBalance.GreaterThanOrEqual(decimal.Zero),
			"Balance should never be negative. Initial: %s, Deducted: %s, Final: %s",
			initialBalance, totalDeducted, finalBalance)
	}))

	props.Property("deduct amount equals balance change", convey.Settle(func(gs *gopter.State) {
		// Setup
		tenantID := uuid.New()
		initialBalance := decimal.NewFromInt(1000)
		deductAmount := decimal.NewFromFloat(gs.GetFloat64(1, 500))
		
		mockRepo := new(MockBalanceRepository)
		mockRepo.On("GetBalance", mock.Anything, tenantID).Return(initialBalance, nil)
		mockRepo.On("DeductBalance", mock.Anything, tenantID, deductAmount).Return(nil)
		
		manager := balance.NewManager(mockRepo)
		
		// Get initial balance
		oldBalance, _ := manager.GetBalance(context.Background(), tenantID)
		
		// Deduct
		err := manager.DeductBalance(context.Background(), tenantID, deductAmount)
		assert.NoError(t, err)
		
		// Get new balance
		newBalance, _ := manager.GetBalance(context.Background(), tenantID)
		
		// Verify
		expectedNewBalance := oldBalance.Sub(deductAmount)
		assert.True(t, newBalance.Equal(expectedNewBalance),
			"Balance should decrease by exact amount. Old: %s, Amount: %s, New: %s, Expected: %s",
			oldBalance, deductAmount, newBalance, expectedNewBalance)
	}))

	props.TestingT(t)
}
```

### 4. اختبارات E2E (End-to-End)

```typescript
// apps/admin-web/src/e2e/messaging.spec.ts

import { test, expect } from '@playwright/test'

test.describe('Messaging Flow', () => {
  test.beforeEach(async ({ page }) => {
    // Login as admin
    await page.goto('/login')
    await page.fill('input[name="email"]', 'admin@alhudhud.com')
    await page.fill('input[name="password"]', 'password123')
    await page.click('button[type="submit"]')
    
    // Wait for dashboard to load
    await expect(page).toHaveURL(/\/dashboard/)
  })

  test('admin can send bulk messages', async ({ page }) => {
    // Navigate to messaging page
    await page.click('a[href="/messaging"]')
    await expect(page).toHaveURL(/\/messaging/)
    
    // Fill message form
    await page.fill('textarea[name="message"]', 'Test bulk message from E2E test')
    
    // Upload recipients file
    const fileInput = page.locator('input[type="file"]')
    await fileInput.setInputFiles('tests/fixtures/recipients.csv')
    
    // Select channel
    await page.selectOption('select[name="channel"]', 'direct')
    
    // Verify recipient count
    await expect(page.locator('text=100 recipients')).toBeVisible()
    
    // Schedule for now
    await page.click('button:has-text("Send Now")')
    
    // Verify confirmation dialog
    await expect(page.locator('text=Send 100 messages?')).toBeVisible()
    
    // Confirm send
    await page.click('button:has-text("Confirm")')
    
    // Verify success message
    await expect(page.locator('text=Messages queued successfully')).toBeVisible()
    
    // Navigate to history
    await page.click('a[href="/messaging/history"]')
    
    // Verify message appears in history
    await expect(page.locator('text=Test bulk message from E2E test')).toBeVisible()
  })

  test('validates phone numbers before sending', async ({ page }) => {
    await page.click('a[href="/messaging"]')
    
    // Enter invalid phone number
    await page.fill('input[name="recipients"]', 'invalid-phone-number')
    
    // Trigger validation
    await page.click('button:has-text("Validate")')
    
    // Verify error message
    await expect(page.locator('text=Invalid phone number format')).toBeVisible()
  })

  test('shows delivery reports', async ({ page }) => {
    await page.click('a[href="/messaging/reports"]')
    
    // Select date range
    await page.fill('input[name="startDate"]', '2024-01-01')
    await page.fill('input[name="endDate"]', '2024-01-31')
    
    // Click generate report
    await page.click('button:has-text("Generate Report")')
    
    // Wait for report to load
    await page.waitForSelector('table[data-testid="delivery-report"]')
    
    // Verify report data
    const rows = page.locator('table[data-testid="delivery-report"] tbody tr')
    await expect(rows).toHaveCount(10) // 10 entries per page
    
    // Check statistics
    await expect(page.locator('text=95% Delivery Rate')).toBeVisible()
  })
})
```

---

## دورة TDD

### الخطوة 1: اكتب الاختبار أولاً

```go
// services/backend/internal/routing/routing_test.go

func TestRoutingEngine_SelectOptimalRoute(t *testing.T) {
	// Arrange
	engine := routing.NewEngine()
	routes := []Route{
		{ID: "route1", Cost: 0.05, SuccessRate: 0.99, Latency: 100},
		{ID: "route2", Cost: 0.03, SuccessRate: 0.95, Latency: 200},
		{ID: "route3", Cost: 0.04, SuccessRate: 0.97, Latency: 150},
	}
	
	// Act
	selectedRoute := engine.SelectOptimalRoute(routes, routing.OptimizeCost)
	
	// Assert
	assert.Equal(t, "route2", selectedRoute.ID)
}
```

### الخطوة 2: شغّل الاختبار (يجب أن يفشل)

```bash
cd services/backend
go test ./internal/routing/... -v
# Expected: FAIL - function not implemented yet
```

### الخطوة 3: اكتب الكود

```go
// services/backend/internal/routing/engine.go

type Route struct {
	ID         string
	Cost       float64
	SuccessRate float64
	Latency    int
}

type OptimizationStrategy int

const (
	OptimizeCost OptimizationStrategy = iota
	OptimizeSpeed
	OptimizeReliability
)

type Engine struct {
	// ...
}

func (e *Engine) SelectOptimalRoute(routes []Route, strategy OptimizationStrategy) Route {
	if len(routes) == 0 {
		return Route{}
	}
	
	switch strategy {
	case OptimizeCost:
		return e.selectByCost(routes)
	case OptimizeSpeed:
		return e.selectByLatency(routes)
	case OptimizeReliability:
		return e.selectBySuccessRate(routes)
	default:
		return routes[0]
	}
}

func (e *Engine) selectByCost(routes []Route) Route {
	best := routes[0]
	for _, route := range routes[1:] {
		if route.Cost < best.Cost {
			best = route
		}
	}
	return best
}
```

### الخطوة 4: شغّل الاختبار (يجب أن ينجح)

```bash
go test ./internal/routing/... -v
# Expected: PASS
```

### الخطوة 5: أصلح الكود

```go
// تحسين الأداء وإضافة معالجة الحالات الحدية
func (e *Engine) SelectOptimalRoute(routes []Route, strategy OptimizationStrategy) Route {
	if len(routes) == 0 {
		return Route{}
	}
	
	// تصفية المسارات غير النشطة
	activeRoutes := filterActiveRoutes(routes)
	if len(activeRoutes) == 0 {
		return Route{}
	}
	
	// ... rest of implementation
}

func filterActiveRoutes(routes []Route) []Route {
	var active []Route
	for _, r := range routes {
		if r.SuccessRate >= 0.9 { // فقط المسارات بنسبة نجاح 90%+
			active = append(active, r)
		}
	}
	return active
}
```

### الخطوة 6: تحقق من التغطية

```bash
go test ./internal/routing/... -coverprofile=coverage.out
go tool cover -func=coverage.out
# Verify 80%+ coverage
```

---

## تغطية الاختبارات المطلوبة

### متطلبات التغطية

| نوع الكود | الحد الأدنى | ملاحظات |
|-----------|-------------|---------|
| Business Logic | 90% | التوازن، التوجيه، الفوترة |
| API Handlers | 85% | جميع نقاط النهاية |
| Middleware | 80% | المصادقة، Rate Limiting |
| Utilities | 75% | Helpers، Transformers |
| Frontend Components | 80% | المكونات الرئيسية |

### تشغيل التغطية

```bash
# Backend
cd services/backend
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
open coverage.html

# Frontend
cd apps/admin-web
npm run test:coverage
```

---

## الاختبارات المطلوبة حسب المكون

### 1. نظام الرسائل

```go
// يجب اختبار:
// 1. إرسال رسالة ناجح
// 2. إرسال رسالة مع رصيد غير كافٍ
// 3. التحقق من صحة رقم الهاتف
// 4. اختيار المسار الأمثل
// 5. معالجة الفشل وإعادة المحاولة
// 6. تحديث حالة التسليم (DLR)
// 7. الرسائل المتكررة (Idempotency)
```

### 2. نظام الرصيد

```go
// يجب اختبار:
// 1. خصم الرصيد (Atomic)
// 2. إضافة الرصيد
// 3. رصيد غير كافٍ
// 4. Race conditions
// 5. تاريخ المعاملات
// 6. تحويل الرصيد بين الشركات
```

### 3. نظام المصادقة

```go
// يجب اختبار:
// 1. تسجيل الدخول ناجح
// 2. تسجيل الدخول بفشل
// 3. JWT Token generation
// 4. Token refresh
// 5. Logout
// 6. Rate limiting على Auth
```

### 4. واجهة Admin

```tsx
// يجب اختبار:
// 1. عرض لوحة التحكم
// 2. إدارة الشركات
// 3. إدارة القنوات
// 4. التقارير والإحصائيات
// 5. إعدادات التنبيهات
```

---

## أفضل الممارسات

### ✅ افعل

- اكتب الاختبار قبل الكود
- اختبر السلوك وليس التنفيذ
- استخدم descriptive names للاختبارات
- اختبر الحالات الحدية (null, empty, large values)
- اختبر مسارات الأخطاء
- استخدم mocks للخدمات الخارجية
- اجعل الاختبارات سريعة (< 50ms للوحدة)

### ❌ لا تفعل

- تخطي كتابة الاختبارات
- اختبار تفاصيل التنفيذ الداخلية
- اختبارات هشة (تتكسر بسهولة)
- اختبارات مترابطة (تعتمد على بعضها)
- تجاهل التغطية المنخفضة
- استخدام sleep/wait في الاختبارات

---

## أوامر التشغيل

```bash
# Backend Tests
cd services/backend

# تشغيل جميع الاختبارات
go test ./... -v

# تشغيل اختبارات محددة
go test ./internal/balance/... -v

# تشغيل مع تغطية
go test ./... -coverprofile=coverage.out

# تشغيل E2E tests
go test ./tests/e2e/... -v

# تشغيل في وضع المراقبة
go test ./... -watch

# Property-based tests
go test ./... -run TestBalanceManager_PropertyBased

# Frontend Tests
cd apps/admin-web

# تشغيل الاختبارات
npm test

# تشغيل مع تغطية
npm run test:coverage

# تشغيل E2E tests
npm run test:e2e
```

---

## CI/CD Integration

```yaml
# .github/workflows/test.yml

name: Tests

on: [push, pull_request]

jobs:
  backend-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.25'
      - name: Run Tests
        run: |
          cd services/backend
          go test ./... -coverprofile=coverage.out
      - name: Upload Coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./services/backend/coverage.out

  frontend-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Setup Node
        uses: actions/setup-node@v3
        with:
          node-version: '20'
      - name: Install Dependencies
        run: |
          cd apps/admin-web
          npm ci
      - name: Run Tests
        run: npm test -- --coverage
      - name: Upload Coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./apps/admin-web/coverage/lcov.info
```

---

## ملخص

TDD هو منهجية أساسية لضمان جودة كود ALHudhudAI. باتباع هذا الدليل:

1. **ابدأ بكتابة الاختبار** قبل أي كود جديد
2. **استخدم أنواع الاختبارات المناسبة** (وحدة، تكامل، E2E)
3. **حافظ على تغطية 80%+** لجميع المكونات
4. **أتمت الاختبارات** في CI/CD pipeline
5. **راجع التغطية بانتظام** وحدّث الاختبارات

---

**تذكر:** الاختبارات ليست اختيارية - هي الشبكة الأمان التي تمكّنك من التطوير بثقة.