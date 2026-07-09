package promo

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecordUsage(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	tracker := NewUsageTracker(db)
	ctx := context.Background()
	tenantID := int64(1)
	
	defer cleanupTestData(t, db, tenantID)
	
	usage := &PromoCodeUsage{
		TenantID:       tenantID,
		PromoCodeID:    1,
		UserID:         1,
		OrderType:      OrderTypeSubscription,
		OrderID:        nil,
		OriginalAmount: 100.0,
		DiscountAmount: 20.0,
		FinalAmount:    80.0,
	}
	
	err := tracker.RecordUsage(ctx, usage)
	require.NoError(t, err)
	assert.NotZero(t, usage.ID)
	assert.NotZero(t, usage.UsedAt)
}

func TestGetUserUsageCount(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	manager := NewManager(db)
	tracker := NewUsageTracker(db)
	ctx := context.Background()
	tenantID := int64(1)
	userID := int64(1)
	
	defer cleanupTestData(t, db, tenantID)
	
	// إنشاء كود خصم
	createReq := &CreatePromoCodeRequest{
		Code:          "COUNTTEST",
		Description:   "اختبار العد",
		DiscountType:  DiscountTypePercentage,
		DiscountValue: 10.0,
		ValidFrom:     time.Now(),
		ValidUntil:    time.Now().Add(30 * 24 * time.Hour),
		PerUserLimit:  3,
		ApplicableTo:  ApplicableToAll,
	}
	
	promoCode, err := manager.CreatePromoCode(ctx, tenantID, createReq)
	require.NoError(t, err)
	
	// تسجيل استخدامات متعددة
	for i := 0; i < 3; i++ {
		usage := &PromoCodeUsage{
			TenantID:       tenantID,
			PromoCodeID:    promoCode.ID,
			UserID:         userID,
			OrderType:      OrderTypeSubscription,
			OriginalAmount: 100.0,
			DiscountAmount: 10.0,
			FinalAmount:    90.0,
		}
		
		err := tracker.RecordUsage(ctx, usage)
		require.NoError(t, err)
	}
	
	// التحقق من العد
	count, err := tracker.GetUserUsageCount(ctx, tenantID, promoCode.ID, userID)
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestHasUserUsedPromoCode(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	manager := NewManager(db)
	tracker := NewUsageTracker(db)
	ctx := context.Background()
	tenantID := int64(1)
	userID := int64(1)
	
	defer cleanupTestData(t, db, tenantID)
	
	// إنشاء كود خصم
	createReq := &CreatePromoCodeRequest{
		Code:          "HASUSED",
		Description:   "اختبار الاستخدام",
		DiscountType:  DiscountTypePercentage,
		DiscountValue: 10.0,
		ValidFrom:     time.Now(),
		ValidUntil:    time.Now().Add(30 * 24 * time.Hour),
		PerUserLimit:  1,
		ApplicableTo:  ApplicableToAll,
	}
	
	promoCode, err := manager.CreatePromoCode(ctx, tenantID, createReq)
	require.NoError(t, err)
	
	// التحقق قبل الاستخدام
	hasUsed, err := tracker.HasUserUsedPromoCode(ctx, tenantID, promoCode.ID, userID)
	require.NoError(t, err)
	assert.False(t, hasUsed)
	
	// تسجيل استخدام
	usage := &PromoCodeUsage{
		TenantID:       tenantID,
		PromoCodeID:    promoCode.ID,
		UserID:         userID,
		OrderType:      OrderTypeSubscription,
		OriginalAmount: 100.0,
		DiscountAmount: 10.0,
		FinalAmount:    90.0,
	}
	
	err = tracker.RecordUsage(ctx, usage)
	require.NoError(t, err)
	
	// التحقق بعد الاستخدام
	hasUsed, err = tracker.HasUserUsedPromoCode(ctx, tenantID, promoCode.ID, userID)
	require.NoError(t, err)
	assert.True(t, hasUsed)
}

func TestCanUserUsePromoCode(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	manager := NewManager(db)
	tracker := NewUsageTracker(db)
	ctx := context.Background()
	tenantID := int64(1)
	userID := int64(1)
	
	defer cleanupTestData(t, db, tenantID)
	
	// إنشاء كود خصم بحد 2 استخدامات لكل مستخدم
	createReq := &CreatePromoCodeRequest{
		Code:          "CANUSE",
		Description:   "اختبار الإمكانية",
		DiscountType:  DiscountTypePercentage,
		DiscountValue: 10.0,
		ValidFrom:     time.Now(),
		ValidUntil:    time.Now().Add(30 * 24 * time.Hour),
		PerUserLimit:  2,
		ApplicableTo:  ApplicableToAll,
	}
	
	promoCode, err := manager.CreatePromoCode(ctx, tenantID, createReq)
	require.NoError(t, err)
	
	// التحقق قبل أي استخدام
	canUse, err := tracker.CanUserUsePromoCode(ctx, tenantID, promoCode.ID, userID, 2)
	require.NoError(t, err)
	assert.True(t, canUse)
	
	// تسجيل استخدام واحد
	usage1 := &PromoCodeUsage{
		TenantID:       tenantID,
		PromoCodeID:    promoCode.ID,
		UserID:         userID,
		OrderType:      OrderTypeSubscription,
		OriginalAmount: 100.0,
		DiscountAmount: 10.0,
		FinalAmount:    90.0,
	}
	err = tracker.RecordUsage(ctx, usage1)
	require.NoError(t, err)
	
	// التحقق بعد استخدام واحد
	canUse, err = tracker.CanUserUsePromoCode(ctx, tenantID, promoCode.ID, userID, 2)
	require.NoError(t, err)
	assert.True(t, canUse)
	
	// تسجيل استخدام ثاني
	usage2 := &PromoCodeUsage{
		TenantID:       tenantID,
		PromoCodeID:    promoCode.ID,
		UserID:         userID,
		OrderType:      OrderTypeTopup,
		OriginalAmount: 200.0,
		DiscountAmount: 20.0,
		FinalAmount:    180.0,
	}
	err = tracker.RecordUsage(ctx, usage2)
	require.NoError(t, err)
	
	// التحقق بعد استخدامين (الحد الأقصى)
	canUse, err = tracker.CanUserUsePromoCode(ctx, tenantID, promoCode.ID, userID, 2)
	require.NoError(t, err)
	assert.False(t, canUse)
}

func TestGetPromoCodeUsageStats(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	manager := NewManager(db)
	tracker := NewUsageTracker(db)
	ctx := context.Background()
	tenantID := int64(1)
	
	defer cleanupTestData(t, db, tenantID)
	
	// إنشاء كود خصم
	createReq := &CreatePromoCodeRequest{
		Code:          "STATS",
		Description:   "اختبار الإحصائيات",
		DiscountType:  DiscountTypePercentage,
		DiscountValue: 15.0,
		ValidFrom:     time.Now(),
		ValidUntil:    time.Now().Add(30 * 24 * time.Hour),
		PerUserLimit:  5,
		ApplicableTo:  ApplicableToAll,
	}
	
	promoCode, err := manager.CreatePromoCode(ctx, tenantID, createReq)
	require.NoError(t, err)
	
	// تسجيل استخدامات من مستخدمين مختلفين
	for userID := int64(1); userID <= 3; userID++ {
		for i := 0; i < 2; i++ {
			usage := &PromoCodeUsage{
				TenantID:       tenantID,
				PromoCodeID:    promoCode.ID,
				UserID:         userID,
				OrderType:      OrderTypeSubscription,
				OriginalAmount: 100.0,
				DiscountAmount: 15.0,
				FinalAmount:    85.0,
			}
			
			err := tracker.RecordUsage(ctx, usage)
			require.NoError(t, err)
		}
	}
	
	// الحصول على الإحصائيات
	stats, err := tracker.GetPromoCodeUsageStats(ctx, tenantID, promoCode.ID)
	require.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, 6, stats.TotalUses)           // 3 مستخدمين × 2 استخدامات
	assert.Equal(t, 3, stats.UniqueUsers)         // 3 مستخدمين فريدين
	assert.Equal(t, 600.0, stats.TotalOriginalAmount) // 100 × 6
	assert.Equal(t, 90.0, stats.TotalDiscountGiven)   // 15 × 6
	assert.Equal(t, 510.0, stats.TotalFinalAmount)    // 85 × 6
	assert.Equal(t, 15.0, stats.AvgDiscountAmount)
}

func TestGetUsageByDateRange(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	manager := NewManager(db)
	tracker := NewUsageTracker(db)
	ctx := context.Background()
	tenantID := int64(1)
	
	defer cleanupTestData(t, db, tenantID)
	
	// إنشاء كود خصم
	createReq := &CreatePromoCodeRequest{
		Code:          "DATERANGE",
		Description:   "اختبار النطاق الزمني",
		DiscountType:  DiscountTypePercentage,
		DiscountValue: 10.0,
		ValidFrom:     time.Now().Add(-7 * 24 * time.Hour),
		ValidUntil:    time.Now().Add(30 * 24 * time.Hour),
		PerUserLimit:  10,
		ApplicableTo:  ApplicableToAll,
	}
	
	promoCode, err := manager.CreatePromoCode(ctx, tenantID, createReq)
	require.NoError(t, err)
	
	// تسجيل استخدامات
	for i := 0; i < 5; i++ {
		usage := &PromoCodeUsage{
			TenantID:       tenantID,
			PromoCodeID:    promoCode.ID,
			UserID:         int64(i + 1),
			OrderType:      OrderTypeSubscription,
			OriginalAmount: 100.0,
			DiscountAmount: 10.0,
			FinalAmount:    90.0,
		}
		
		err := tracker.RecordUsage(ctx, usage)
		require.NoError(t, err)
	}
	
	// الحصول على الاستخدامات في نطاق زمني
	startDate := time.Now().Add(-1 * time.Hour)
	endDate := time.Now().Add(1 * time.Hour)
	
	usages, err := tracker.GetUsageByDateRange(ctx, tenantID, promoCode.ID, startDate, endDate)
	require.NoError(t, err)
	assert.Len(t, usages, 5)
}

func TestGetUsageByOrderType(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	manager := NewManager(db)
	tracker := NewUsageTracker(db)
	ctx := context.Background()
	tenantID := int64(1)
	
	defer cleanupTestData(t, db, tenantID)
	
	// إنشاء كود خصم
	createReq := &CreatePromoCodeRequest{
		Code:          "ORDERTYPE",
		Description:   "اختبار نوع الطلب",
		DiscountType:  DiscountTypePercentage,
		DiscountValue: 10.0,
		ValidFrom:     time.Now(),
		ValidUntil:    time.Now().Add(30 * 24 * time.Hour),
		PerUserLimit:  10,
		ApplicableTo:  ApplicableToAll,
	}
	
	promoCode, err := manager.CreatePromoCode(ctx, tenantID, createReq)
	require.NoError(t, err)
	
	// تسجيل استخدامات من أنواع مختلفة
	orderTypes := []OrderType{
		OrderTypeSubscription,
		OrderTypeSubscription,
		OrderTypeTopup,
		OrderTypeGateway,
	}
	
	for i, orderType := range orderTypes {
		usage := &PromoCodeUsage{
			TenantID:       tenantID,
			PromoCodeID:    promoCode.ID,
			UserID:         int64(i + 1),
			OrderType:      orderType,
			OriginalAmount: 100.0,
			DiscountAmount: 10.0,
			FinalAmount:    90.0,
		}
		
		err := tracker.RecordUsage(ctx, usage)
		require.NoError(t, err)
	}
	
	// الحصول على استخدامات الاشتراكات فقط
	subscriptionUsages, err := tracker.GetUsageByOrderType(ctx, tenantID, promoCode.ID, OrderTypeSubscription)
	require.NoError(t, err)
	assert.Len(t, subscriptionUsages, 2)
	
	// الحصول على استخدامات التوريد فقط
	topupUsages, err := tracker.GetUsageByOrderType(ctx, tenantID, promoCode.ID, OrderTypeTopup)
	require.NoError(t, err)
	assert.Len(t, topupUsages, 1)
}

func TestGetTopUsers(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	manager := NewManager(db)
	tracker := NewUsageTracker(db)
	ctx := context.Background()
	tenantID := int64(1)
	
	defer cleanupTestData(t, db, tenantID)
	
	// إنشاء كود خصم
	createReq := &CreatePromoCodeRequest{
		Code:          "TOPUSERS",
		Description:   "اختبار أكثر المستخدمين",
		DiscountType:  DiscountTypePercentage,
		DiscountValue: 10.0,
		ValidFrom:     time.Now(),
		ValidUntil:    time.Now().Add(30 * 24 * time.Hour),
		PerUserLimit:  10,
		ApplicableTo:  ApplicableToAll,
	}
	
	promoCode, err := manager.CreatePromoCode(ctx, tenantID, createReq)
	require.NoError(t, err)
	
	// تسجيل استخدامات بأعداد مختلفة
	usageCounts := map[int64]int{
		1: 5, // المستخدم 1: 5 استخدامات
		2: 3, // المستخدم 2: 3 استخدامات
		3: 1, // المستخدم 3: 1 استخدام
	}
	
	for userID, count := range usageCounts {
		for i := 0; i < count; i++ {
			usage := &PromoCodeUsage{
				TenantID:       tenantID,
				PromoCodeID:    promoCode.ID,
				UserID:         userID,
				OrderType:      OrderTypeSubscription,
				OriginalAmount: 100.0,
				DiscountAmount: 10.0,
				FinalAmount:    90.0,
			}
			
			err := tracker.RecordUsage(ctx, usage)
			require.NoError(t, err)
		}
	}
	
	// الحصول على أكثر المستخدمين استخداماً
	topUsers, err := tracker.GetTopUsers(ctx, tenantID, promoCode.ID, 3)
	require.NoError(t, err)
	assert.Len(t, topUsers, 3)
	
	// التحقق من الترتيب
	assert.Equal(t, int64(1), topUsers[0].UserID)
	assert.Equal(t, 5, topUsers[0].UsageCount)
	assert.Equal(t, int64(2), topUsers[1].UserID)
	assert.Equal(t, 3, topUsers[1].UsageCount)
	assert.Equal(t, int64(3), topUsers[2].UserID)
	assert.Equal(t, 1, topUsers[2].UsageCount)
}

func TestPreventDuplicateUsage(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	manager := NewManager(db)
	tracker := NewUsageTracker(db)
	ctx := context.Background()
	tenantID := int64(1)
	userID := int64(1)
	orderID := int64(123)
	
	defer cleanupTestData(t, db, tenantID)
	
	// إنشاء كود خصم
	createReq := &CreatePromoCodeRequest{
		Code:          "NODUP",
		Description:   "منع التكرار",
		DiscountType:  DiscountTypePercentage,
		DiscountValue: 10.0,
		ValidFrom:     time.Now(),
		ValidUntil:    time.Now().Add(30 * 24 * time.Hour),
		PerUserLimit:  10,
		ApplicableTo:  ApplicableToAll,
	}
	
	promoCode, err := manager.CreatePromoCode(ctx, tenantID, createReq)
	require.NoError(t, err)
	
	// التحقق قبل الاستخدام
	canUse, err := tracker.PreventDuplicateUsage(ctx, tenantID, promoCode.ID, OrderTypeSubscription, orderID)
	require.NoError(t, err)
	assert.True(t, canUse)
	
	// تسجيل استخدام
	usage := &PromoCodeUsage{
		TenantID:       tenantID,
		PromoCodeID:    promoCode.ID,
		UserID:         userID,
		OrderType:      OrderTypeSubscription,
		OrderID:        &orderID,
		OriginalAmount: 100.0,
		DiscountAmount: 10.0,
		FinalAmount:    90.0,
	}
	
	err = tracker.RecordUsage(ctx, usage)
	require.NoError(t, err)
	
	// التحقق بعد الاستخدام
	canUse, err = tracker.PreventDuplicateUsage(ctx, tenantID, promoCode.ID, OrderTypeSubscription, orderID)
	require.NoError(t, err)
	assert.False(t, canUse) // لا يمكن استخدام نفس الكود لنفس الطلب
}
