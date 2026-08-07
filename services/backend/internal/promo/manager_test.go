package promo

import (
	"context"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestDB ينشئ اتصال قاعدة بيانات للاختبار
func setupTestDB(t *testing.T) *sqlx.DB {
	// استخدام قاعدة بيانات الاختبار
	db, err := sqlx.Connect("postgres", "postgres://postgres@localhost:5432/sms_gateway_test?sslmode=disable")
	require.NoError(t, err)
	
	// تعيين tenant_id للاختبار
	_, err = db.Exec("SET app.current_tenant = '1'")
	require.NoError(t, err)
	
	return db
}

// cleanupTestData ينظف بيانات الاختبار
func cleanupTestData(t *testing.T, db *sqlx.DB, tenantID int64) {
	_, err := db.Exec("DELETE FROM promo_code_usage WHERE tenant_id = $1", tenantID)
	require.NoError(t, err)
	
	_, err = db.Exec("DELETE FROM promo_codes WHERE tenant_id = $1", tenantID)
	require.NoError(t, err)
}

func TestCreatePromoCode(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	manager := NewManager(db)
	ctx := context.Background()
	tenantID := int64(1)
	
	defer cleanupTestData(t, db, tenantID)
	
	t.Run("Create percentage discount code", func(t *testing.T) {
		req := &CreatePromoCodeRequest{
			Code:          "SUMMER2026",
			Description:   "خصم صيفي 20%",
			DiscountType:  DiscountTypePercentage,
			DiscountValue: 20.0,
			ValidFrom:     time.Now(),
			ValidUntil:    time.Now().Add(30 * 24 * time.Hour),
			PerUserLimit:  1,
			ApplicableTo:  ApplicableToAll,
		}
		
		promoCode, err := manager.CreatePromoCode(ctx, tenantID, req)
		require.NoError(t, err)
		assert.NotNil(t, promoCode)
		assert.Equal(t, "SUMMER2026", promoCode.Code)
		assert.Equal(t, DiscountTypePercentage, promoCode.DiscountType)
		assert.Equal(t, 20.0, promoCode.DiscountValue)
		assert.True(t, promoCode.IsActive)
	})
	
	t.Run("Create fixed amount discount code", func(t *testing.T) {
		maxDiscount := 50.0
		minPurchase := 100.0
		usageLimit := 100
		
		req := &CreatePromoCodeRequest{
			Code:              "FIXED50",
			Description:       "خصم 50 ريال",
			DiscountType:      DiscountTypeFixedAmount,
			DiscountValue:     50.0,
			MaxDiscountAmount: &maxDiscount,
			MinPurchaseAmount: &minPurchase,
			ValidFrom:         time.Now(),
			ValidUntil:        time.Now().Add(30 * 24 * time.Hour),
			UsageLimit:        &usageLimit,
			PerUserLimit:      2,
			ApplicableTo:      ApplicableToPackages,
			ApplicablePackageIDs: []int64{1, 2, 3},
		}
		
		promoCode, err := manager.CreatePromoCode(ctx, tenantID, req)
		require.NoError(t, err)
		assert.NotNil(t, promoCode)
		assert.Equal(t, "FIXED50", promoCode.Code)
		assert.Equal(t, DiscountTypeFixedAmount, promoCode.DiscountType)
		assert.Equal(t, 50.0, promoCode.DiscountValue)
		assert.NotNil(t, promoCode.MaxDiscountAmount)
		assert.Equal(t, 50.0, *promoCode.MaxDiscountAmount)
		assert.NotNil(t, promoCode.MinPurchaseAmount)
		assert.Equal(t, 100.0, *promoCode.MinPurchaseAmount)
		assert.NotNil(t, promoCode.UsageLimit)
		assert.Equal(t, 100, *promoCode.UsageLimit)
		assert.Equal(t, 2, promoCode.PerUserLimit)
		assert.Equal(t, ApplicableToPackages, promoCode.ApplicableTo)
		assert.Len(t, promoCode.ApplicablePackageIDs, 3)
	})
	
	t.Run("Duplicate code should fail", func(t *testing.T) {
		req := &CreatePromoCodeRequest{
			Code:          "DUPLICATE",
			Description:   "كود مكرر",
			DiscountType:  DiscountTypePercentage,
			DiscountValue: 10.0,
			ValidFrom:     time.Now(),
			ValidUntil:    time.Now().Add(30 * 24 * time.Hour),
			PerUserLimit:  1,
			ApplicableTo:  ApplicableToAll,
		}
		
		// إنشاء الكود الأول
		_, err := manager.CreatePromoCode(ctx, tenantID, req)
		require.NoError(t, err)
		
		// محاولة إنشاء كود مكرر
		_, err = manager.CreatePromoCode(ctx, tenantID, req)
		assert.Error(t, err)
		assert.Equal(t, ErrPromoCodeAlreadyExists, err)
	})
	
	t.Run("Invalid date range should fail", func(t *testing.T) {
		req := &CreatePromoCodeRequest{
			Code:          "INVALID",
			Description:   "تاريخ غير صحيح",
			DiscountType:  DiscountTypePercentage,
			DiscountValue: 10.0,
			ValidFrom:     time.Now().Add(30 * 24 * time.Hour),
			ValidUntil:    time.Now(), // تاريخ انتهاء قبل البداية
			PerUserLimit:  1,
			ApplicableTo:  ApplicableToAll,
		}
		
		_, err := manager.CreatePromoCode(ctx, tenantID, req)
		assert.Error(t, err)
	})
}

func TestValidatePromoCode(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	manager := NewManager(db)
	ctx := context.Background()
	tenantID := int64(1)
	userID := int64(1)
	
	defer cleanupTestData(t, db, tenantID)
	
	// إنشاء كود خصم للاختبار
	createReq := &CreatePromoCodeRequest{
		Code:          "VALID2026",
		Description:   "كود صالح",
		DiscountType:  DiscountTypePercentage,
		DiscountValue: 15.0,
		ValidFrom:     time.Now().Add(-1 * time.Hour),
		ValidUntil:    time.Now().Add(30 * 24 * time.Hour),
		PerUserLimit:  2,
		ApplicableTo:  ApplicableToAll,
	}
	
	promoCode, err := manager.CreatePromoCode(ctx, tenantID, createReq)
	require.NoError(t, err)
	
	t.Run("Valid promo code", func(t *testing.T) {
		req := &ApplyPromoCodeRequest{
			Code:      "VALID2026",
			UserID:    userID,
			OrderType: OrderTypeSubscription,
			Amount:    100.0,
		}
		
		validated, err := manager.ValidatePromoCode(ctx, tenantID, req)
		require.NoError(t, err)
		assert.NotNil(t, validated)
		assert.Equal(t, promoCode.ID, validated.ID)
	})
	
	t.Run("Non-existent code", func(t *testing.T) {
		req := &ApplyPromoCodeRequest{
			Code:      "NOTEXIST",
			UserID:    userID,
			OrderType: OrderTypeSubscription,
			Amount:    100.0,
		}
		
		_, err := manager.ValidatePromoCode(ctx, tenantID, req)
		assert.Error(t, err)
		assert.Equal(t, ErrPromoCodeNotFound, err)
	})
	
	t.Run("Expired code", func(t *testing.T) {
		// إنشاء كود منتهي
		expiredReq := &CreatePromoCodeRequest{
			Code:          "EXPIRED",
			Description:   "كود منتهي",
			DiscountType:  DiscountTypePercentage,
			DiscountValue: 10.0,
			ValidFrom:     time.Now().Add(-48 * time.Hour),
			ValidUntil:    time.Now().Add(-24 * time.Hour),
			PerUserLimit:  1,
			ApplicableTo:  ApplicableToAll,
		}
		
		_, err := manager.CreatePromoCode(ctx, tenantID, expiredReq)
		require.NoError(t, err)
		
		req := &ApplyPromoCodeRequest{
			Code:      "EXPIRED",
			UserID:    userID,
			OrderType: OrderTypeSubscription,
			Amount:    100.0,
		}
		
		_, err = manager.ValidatePromoCode(ctx, tenantID, req)
		assert.Error(t, err)
		assert.Equal(t, ErrPromoCodeExpired, err)
	})
	
	t.Run("Inactive code", func(t *testing.T) {
		// إنشاء كود وتعطيله
		inactiveReq := &CreatePromoCodeRequest{
			Code:          "INACTIVE",
			Description:   "كود معطل",
			DiscountType:  DiscountTypePercentage,
			DiscountValue: 10.0,
			ValidFrom:     time.Now(),
			ValidUntil:    time.Now().Add(30 * 24 * time.Hour),
			PerUserLimit:  1,
			ApplicableTo:  ApplicableToAll,
		}
		
		inactive, err := manager.CreatePromoCode(ctx, tenantID, inactiveReq)
		require.NoError(t, err)
		
		// تعطيل الكود
		err = manager.DeletePromoCode(ctx, tenantID, inactive.ID)
		require.NoError(t, err)
		
		req := &ApplyPromoCodeRequest{
			Code:      "INACTIVE",
			UserID:    userID,
			OrderType: OrderTypeSubscription,
			Amount:    100.0,
		}
		
		_, err = manager.ValidatePromoCode(ctx, tenantID, req)
		assert.Error(t, err)
		assert.Equal(t, ErrPromoCodeInactive, err)
	})
}

func TestApplyPromoCode(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	manager := NewManager(db)
	ctx := context.Background()
	tenantID := int64(1)
	userID := int64(1)
	
	defer cleanupTestData(t, db, tenantID)
	
	t.Run("Apply percentage discount", func(t *testing.T) {
		// إنشاء كود خصم 20%
		createReq := &CreatePromoCodeRequest{
			Code:          "PERCENT20",
			Description:   "خصم 20%",
			DiscountType:  DiscountTypePercentage,
			DiscountValue: 20.0,
			ValidFrom:     time.Now(),
			ValidUntil:    time.Now().Add(30 * 24 * time.Hour),
			PerUserLimit:  1,
			ApplicableTo:  ApplicableToAll,
		}
		
		_, err := manager.CreatePromoCode(ctx, tenantID, createReq)
		require.NoError(t, err)
		
		// تطبيق الكود
		applyReq := &ApplyPromoCodeRequest{
			Code:      "PERCENT20",
			UserID:    userID,
			OrderType: OrderTypeSubscription,
			Amount:    100.0,
		}
		
		response, err := manager.ApplyPromoCode(ctx, tenantID, applyReq)
		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, 100.0, response.OriginalAmount)
		assert.Equal(t, 20.0, response.DiscountAmount)
		assert.Equal(t, 80.0, response.FinalAmount)
	})
	
	t.Run("Apply fixed amount discount", func(t *testing.T) {
		// إنشاء كود خصم 50 ريال
		createReq := &CreatePromoCodeRequest{
			Code:          "FIXED50TEST",
			Description:   "خصم 50 ريال",
			DiscountType:  DiscountTypeFixedAmount,
			DiscountValue: 50.0,
			ValidFrom:     time.Now(),
			ValidUntil:    time.Now().Add(30 * 24 * time.Hour),
			PerUserLimit:  1,
			ApplicableTo:  ApplicableToAll,
		}
		
		_, err := manager.CreatePromoCode(ctx, tenantID, createReq)
		require.NoError(t, err)
		
		// تطبيق الكود
		applyReq := &ApplyPromoCodeRequest{
			Code:      "FIXED50TEST",
			UserID:    userID + 1, // مستخدم مختلف
			OrderType: OrderTypeTopup,
			Amount:    200.0,
		}
		
		response, err := manager.ApplyPromoCode(ctx, tenantID, applyReq)
		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, 200.0, response.OriginalAmount)
		assert.Equal(t, 50.0, response.DiscountAmount)
		assert.Equal(t, 150.0, response.FinalAmount)
	})
	
	t.Run("Apply with max discount limit", func(t *testing.T) {
		maxDiscount := 30.0
		
		// إنشاء كود خصم 50% مع حد أقصى 30 ريال
		createReq := &CreatePromoCodeRequest{
			Code:              "MAXDISCOUNT",
			Description:       "خصم 50% بحد أقصى 30 ريال",
			DiscountType:      DiscountTypePercentage,
			DiscountValue:     50.0,
			MaxDiscountAmount: &maxDiscount,
			ValidFrom:         time.Now(),
			ValidUntil:        time.Now().Add(30 * 24 * time.Hour),
			PerUserLimit:      1,
			ApplicableTo:      ApplicableToAll,
		}
		
		_, err := manager.CreatePromoCode(ctx, tenantID, createReq)
		require.NoError(t, err)
		
		// تطبيق الكود على مبلغ 100 ريال
		// الخصم المحسوب = 50 ريال، لكن الحد الأقصى = 30 ريال
		applyReq := &ApplyPromoCodeRequest{
			Code:      "MAXDISCOUNT",
			UserID:    userID + 2,
			OrderType: OrderTypeSubscription,
			Amount:    100.0,
		}
		
		response, err := manager.ApplyPromoCode(ctx, tenantID, applyReq)
		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, 100.0, response.OriginalAmount)
		assert.Equal(t, 30.0, response.DiscountAmount) // الحد الأقصى
		assert.Equal(t, 70.0, response.FinalAmount)
	})
	
	t.Run("User limit reached", func(t *testing.T) {
		// إنشاء كود بحد استخدام 1 لكل مستخدم
		createReq := &CreatePromoCodeRequest{
			Code:          "ONCEPERUSER",
			Description:   "مرة واحدة لكل مستخدم",
			DiscountType:  DiscountTypePercentage,
			DiscountValue: 10.0,
			ValidFrom:     time.Now(),
			ValidUntil:    time.Now().Add(30 * 24 * time.Hour),
			PerUserLimit:  1,
			ApplicableTo:  ApplicableToAll,
		}
		
		_, err := manager.CreatePromoCode(ctx, tenantID, createReq)
		require.NoError(t, err)
		
		testUserID := userID + 3
		
		// الاستخدام الأول - يجب أن ينجح
		applyReq := &ApplyPromoCodeRequest{
			Code:      "ONCEPERUSER",
			UserID:    testUserID,
			OrderType: OrderTypeSubscription,
			Amount:    100.0,
		}
		
		_, err = manager.ApplyPromoCode(ctx, tenantID, applyReq)
		require.NoError(t, err)
		
		// الاستخدام الثاني - يجب أن يفشل
		_, err = manager.ApplyPromoCode(ctx, tenantID, applyReq)
		assert.Error(t, err)
		assert.Equal(t, ErrPromoCodeUserLimitReached, err)
	})
}

func TestGetPromoPerformance(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	manager := NewManager(db)
	ctx := context.Background()
	tenantID := int64(1)
	
	defer cleanupTestData(t, db, tenantID)
	
	// إنشاء كود خصم
	createReq := &CreatePromoCodeRequest{
		Code:          "PERFORMANCE",
		Description:   "اختبار الأداء",
		DiscountType:  DiscountTypePercentage,
		DiscountValue: 15.0,
		ValidFrom:     time.Now(),
		ValidUntil:    time.Now().Add(30 * 24 * time.Hour),
		PerUserLimit:  5,
		ApplicableTo:  ApplicableToAll,
	}
	
	promoCode, err := manager.CreatePromoCode(ctx, tenantID, createReq)
	require.NoError(t, err)
	
	// تطبيق الكود عدة مرات
	for i := int64(1); i <= 3; i++ {
		applyReq := &ApplyPromoCodeRequest{
			Code:      "PERFORMANCE",
			UserID:    i,
			OrderType: OrderTypeSubscription,
			Amount:    100.0,
		}
		
		_, err := manager.ApplyPromoCode(ctx, tenantID, applyReq)
		require.NoError(t, err)
	}
	
	// الحصول على الأداء
	performance, err := manager.GetPromoPerformance(ctx, tenantID, promoCode.ID)
	require.NoError(t, err)
	assert.NotNil(t, performance)
	assert.Equal(t, "PERFORMANCE", performance.Code)
	assert.Equal(t, 3, performance.TotalUsageCount)
	assert.Equal(t, 45.0, performance.TotalDiscountGiven) // 15% من 100 = 15، مضروب في 3
	assert.Equal(t, 255.0, performance.TotalRevenue)      // 85 * 3
	assert.Equal(t, 3, performance.UniqueUsers)
}

func TestListPromoCodes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	manager := NewManager(db)
	ctx := context.Background()
	tenantID := int64(1)
	
	defer cleanupTestData(t, db, tenantID)
	
	// إنشاء عدة أكواد
	codes := []string{"LIST1", "LIST2", "LIST3"}
	for _, code := range codes {
		req := &CreatePromoCodeRequest{
			Code:          code,
			Description:   "كود للاختبار",
			DiscountType:  DiscountTypePercentage,
			DiscountValue: 10.0,
			ValidFrom:     time.Now(),
			ValidUntil:    time.Now().Add(30 * 24 * time.Hour),
			PerUserLimit:  1,
			ApplicableTo:  ApplicableToAll,
		}
		
		_, err := manager.CreatePromoCode(ctx, tenantID, req)
		require.NoError(t, err)
	}
	
	// الحصول على جميع الأكواد
	promoCodes, err := manager.ListPromoCodes(ctx, tenantID, false)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(promoCodes), 3)
	
	// الحصول على الأكواد النشطة فقط
	activePromoCodes, err := manager.ListPromoCodes(ctx, tenantID, true)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(activePromoCodes), 3)
}

func TestUpdatePromoCode(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	manager := NewManager(db)
	ctx := context.Background()
	tenantID := int64(1)
	
	defer cleanupTestData(t, db, tenantID)
	
	// إنشاء كود
	createReq := &CreatePromoCodeRequest{
		Code:          "UPDATE",
		Description:   "كود للتحديث",
		DiscountType:  DiscountTypePercentage,
		DiscountValue: 10.0,
		ValidFrom:     time.Now(),
		ValidUntil:    time.Now().Add(30 * 24 * time.Hour),
		PerUserLimit:  1,
		ApplicableTo:  ApplicableToAll,
	}
	
	promoCode, err := manager.CreatePromoCode(ctx, tenantID, createReq)
	require.NoError(t, err)
	
	// تحديث الكود
	updates := map[string]interface{}{
		"description":   "وصف محدث",
		"per_user_limit": 3,
		"is_active":     false,
	}
	
	err = manager.UpdatePromoCode(ctx, tenantID, promoCode.ID, updates)
	require.NoError(t, err)
	
	// التحقق من التحديث
	updated, err := manager.getPromoCodeByCode(ctx, tenantID, "UPDATE")
	require.NoError(t, err)
	assert.Equal(t, "وصف محدث", updated.Description)
	assert.Equal(t, 3, updated.PerUserLimit)
	assert.False(t, updated.IsActive)
}

func TestDeletePromoCode(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	manager := NewManager(db)
	ctx := context.Background()
	tenantID := int64(1)
	
	defer cleanupTestData(t, db, tenantID)
	
	// إنشاء كود
	createReq := &CreatePromoCodeRequest{
		Code:          "DELETE",
		Description:   "كود للحذف",
		DiscountType:  DiscountTypePercentage,
		DiscountValue: 10.0,
		ValidFrom:     time.Now(),
		ValidUntil:    time.Now().Add(30 * 24 * time.Hour),
		PerUserLimit:  1,
		ApplicableTo:  ApplicableToAll,
	}
	
	promoCode, err := manager.CreatePromoCode(ctx, tenantID, createReq)
	require.NoError(t, err)
	
	// حذف الكود
	err = manager.DeletePromoCode(ctx, tenantID, promoCode.ID)
	require.NoError(t, err)
	
	// التحقق من الحذف (soft delete)
	deleted, err := manager.getPromoCodeByCode(ctx, tenantID, "DELETE")
	require.NoError(t, err)
	assert.False(t, deleted.IsActive)
}
