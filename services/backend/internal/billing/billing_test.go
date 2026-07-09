package billing_test

import (
	"context"
	"testing"
	"time"

	"github.com/al-hudhud/backend/internal/billing"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockBillingRepository - Mock for billing operations
type MockBillingRepository struct {
	mock.Mock
}

func (m *MockBillingRepository) CreateInvoice(ctx context.Context, invoice *billing.Invoice) error {
	args := m.Called(ctx, invoice)
	return args.Error(0)
}

func (m *MockBillingRepository) GetInvoiceByID(ctx context.Context, invoiceID uuid.UUID) (*billing.Invoice, error) {
	args := m.Called(ctx, invoiceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*billing.Invoice), args.Error(1)
}

func (m *MockBillingRepository) GetInvoicesByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]billing.Invoice, error) {
	args := m.Called(ctx, tenantID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]billing.Invoice), args.Error(1)
}

func (m *MockBillingRepository) UpdateInvoiceStatus(ctx context.Context, invoiceID uuid.UUID, status string) error {
	args := m.Called(ctx, invoiceID, status)
	return args.Error(0)
}

func (m *MockBillingRepository) CreatePaymentTransaction(ctx context.Context, payment *billing.PaymentTransaction) error {
	args := m.Called(ctx, payment)
	return args.Error(0)
}

func (m *MockBillingRepository) ValidatePromoCode(ctx context.Context, code string) (*billing.PromoCode, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*billing.PromoCode), args.Error(1)
}

// MockTenantRepository - Mock for tenant operations used by billing
type MockTenantRepository struct {
	mock.Mock
}

func (m *MockTenantRepository) GetTenantByID(ctx context.Context, tenantID uuid.UUID) (*billing.Tenant, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*billing.Tenant), args.Error(1)
}

func (m *MockBillingRepository) GetPaymentByID(ctx context.Context, paymentID uuid.UUID) (*billing.PaymentTransaction, error) {
	args := m.Called(ctx, paymentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*billing.PaymentTransaction), args.Error(1)
}

func TestBillingService_CreateInvoice_Success(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	invoice := &billing.Invoice{
		TenantID:     tenantID,
		InvoiceNumber: "INV-2024-001",
		TotalAmount:  decimal.NewFromFloat(1500.00),
		Status:       "DRAFT",
		DueDate:      time.Now().Add(30 * 24 * time.Hour),
	}

	mockRepo := new(MockBillingRepository)
	mockRepo.On("CreateInvoice", mock.Anything, mock.AnythingOfType("*billing.Invoice")).Return(nil)

	service := billing.NewService(mockRepo)

	// Act
	result, err := service.CreateInvoice(context.Background(), invoice)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result.ID)
	assert.Equal(t, "DRAFT", result.Status)
	mockRepo.AssertExpectations(t)
}

func TestBillingService_GenerateMonthlyInvoice(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	tenant := &billing.Tenant{
		ID:      tenantID,
		Name:    "Test Company",
		Email:   "billing@test.com",
		Balance: decimal.NewFromFloat(1000),
	}

	usage := []billing.UsageRecord{
		{Service: "SMS", Quantity: 1000, UnitPrice: decimal.NewFromFloat(0.05)},
		{Service: "API", Quantity: 500, UnitPrice: decimal.NewFromFloat(0.01)},
	}

	mockTenantRepo := new(MockTenantRepository)
	mockTenantRepo.On("GetTenantByID", mock.Anything, tenantID).Return(tenant, nil)

	mockBillingRepo := new(MockBillingRepository)
	mockBillingRepo.On("CreateInvoice", mock.Anything, mock.AnythingOfType("*billing.Invoice")).Return(nil)

	service := billing.NewService(mockBillingRepo)
	service.SetTenantRepository(mockTenantRepo)

	// Act
	invoice, err := service.GenerateMonthlyInvoice(context.Background(), tenantID, time.Now(), usage)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, invoice)
	assert.Equal(t, decimal.NewFromFloat(55.00), invoice.TotalAmount) // 1000*0.05 + 500*0.01
	mockBillingRepo.AssertExpectations(t)
}

func TestBillingService_ProcessPayment_Success(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	payment := &billing.PaymentTransaction{
		TenantID:     tenantID,
		Amount:       decimal.NewFromFloat(500.00),
		PaymentMethod: "credit_card",
		Status:       "PENDING",
	}

	mockRepo := new(MockBillingRepository)
	mockRepo.On("CreatePaymentTransaction", mock.Anything, mock.AnythingOfType("*billing.PaymentTransaction")).Return(nil)

	service := billing.NewService(mockRepo)

	// Act
	result, err := service.ProcessPayment(context.Background(), payment)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result.ID)
	assert.Equal(t, "COMPLETED", result.Status)
	mockRepo.AssertExpectations(t)
}

func TestBillingService_ProcessPayment_InsufficientFunds(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	payment := &billing.PaymentTransaction{
		TenantID:     tenantID,
		Amount:       decimal.NewFromFloat(500.00),
		PaymentMethod: "credit_card",
		Status:       "PENDING",
	}

	mockRepo := new(MockBillingRepository)
	mockRepo.On("CreatePaymentTransaction", mock.Anything, mock.AnythingOfType("*billing.PaymentTransaction")).Return(nil)

	service := billing.NewService(mockRepo)

	// Act
	result, err := service.ProcessPayment(context.Background(), payment)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestBillingService_GetInvoiceHistory(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	invoices := []billing.Invoice{
		{ID: uuid.New(), InvoiceNumber: "INV-2024-001", TotalAmount: decimal.NewFromFloat(1000)},
		{ID: uuid.New(), InvoiceNumber: "INV-2024-002", TotalAmount: decimal.NewFromFloat(1500)},
		{ID: uuid.New(), InvoiceNumber: "INV-2024-003", TotalAmount: decimal.NewFromFloat(2000)},
	}

	mockRepo := new(MockBillingRepository)
	mockRepo.On("GetInvoicesByTenant", mock.Anything, tenantID, 10, 0).Return(invoices, nil)

	service := billing.NewService(mockRepo)

	// Act
	result, err := service.GetInvoiceHistory(context.Background(), tenantID, 10, 0)

	// Assert
	assert.NoError(t, err)
	assert.Len(t, result, 3)
	mockRepo.AssertExpectations(t)
}

func TestBillingService_ApplyPromoCode_Valid(t *testing.T) {
	// Arrange
	promoCode := "SAVE20"
	originalAmount := decimal.NewFromFloat(1000.00)
	expectedAmount := decimal.NewFromFloat(800.00) // 20% discount

	mockRepo := new(MockBillingRepository)
	mockRepo.On("ValidatePromoCode", mock.Anything, promoCode).Return(&billing.PromoCode{
		Code:      promoCode,
		Discount:  20,
		Type:      "PERCENTAGE",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}, nil)

	service := billing.NewService(mockRepo)

	// Act
	discountedAmount, err := service.ApplyPromoCode(context.Background(), promoCode, originalAmount)

	// Assert
	assert.NoError(t, err)
	assert.True(t, discountedAmount.Equal(expectedAmount))
	mockRepo.AssertExpectations(t)
}

func TestBillingService_ApplyPromoCode_Expired(t *testing.T) {
	// Arrange
	promoCode := "EXPIRED"
	originalAmount := decimal.NewFromFloat(1000.00)

	mockRepo := new(MockBillingRepository)
	mockRepo.On("ValidatePromoCode", mock.Anything, promoCode).Return(nil, billing.ErrPromoCodeExpired)

	service := billing.NewService(mockRepo)

	// Act
	discountedAmount, err := service.ApplyPromoCode(context.Background(), promoCode, originalAmount)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, billing.ErrPromoCodeExpired, err)
	assert.Equal(t, originalAmount, discountedAmount)
	mockRepo.AssertExpectations(t)
}

func TestBillingService_GenerateInvoiceNumber(t *testing.T) {
	// Arrange
	service := billing.NewService(nil)

	// Act
	invoiceNumber1 := service.GenerateInvoiceNumber()
	invoiceNumber2 := service.GenerateInvoiceNumber()

	// Assert
	assert.NotEmpty(t, invoiceNumber1)
	assert.NotEqual(t, invoiceNumber1, invoiceNumber2)
	assert.Contains(t, invoiceNumber1, "INV-")
}

func TestBillingService_CalculateLateFee(t *testing.T) {
	// Arrange
	service := billing.NewService(nil)
	amount := decimal.NewFromFloat(1000.00)
	daysOverdue := 15

	// Act
	lateFee, err := service.CalculateLateFee(amount, daysOverdue)

	// Assert
	assert.NoError(t, err)
	assert.True(t, lateFee.GreaterThan(decimal.Zero))
}

func TestBillingService_RefundPayment(t *testing.T) {
	// Arrange
	paymentID := uuid.New()
	amount := decimal.NewFromFloat(500.00)

	mockRepo := new(MockBillingRepository)
	mockRepo.On("GetPaymentByID", mock.Anything, paymentID).Return(&billing.PaymentTransaction{
		ID:     paymentID,
		Amount: amount,
		Status: "COMPLETED",
	}, nil)
	mockRepo.On("UpdateInvoiceStatus", mock.Anything, mock.Anything, "REFUNDED").Return(nil)

	service := billing.NewService(mockRepo)

	// Act
	err := service.RefundPayment(context.Background(), paymentID, amount, "Customer request")

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}