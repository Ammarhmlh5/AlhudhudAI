package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	
	"github.com/al-hudhud/backend/internal/promo"
)

func setupPromoTestRouter(t *testing.T) (*gin.Engine, *sqlx.DB) {
	if testing.Short() {
		t.Skip("Skipping promo API tests in short mode")
	}
	gin.SetMode(gin.TestMode)
	
	db, err := sqlx.Connect("postgres", "postgres://postgres:postgres@localhost:5432/sms_gateway_test?sslmode=disable")
	require.NoError(t, err)
	
	router := gin.New()
	
	// Middleware لتعيين tenant_id
	router.Use(func(c *gin.Context) {
		c.Set("tenant_id", int64(1))
		c.Next()
	})
	
	RegisterPromoRoutes(router, db)
	
	return router, db
}

func cleanupPromoTestData(t *testing.T, db *sqlx.DB) {
	_, err := db.Exec("DELETE FROM promo_code_usage WHERE tenant_id = 1")
	require.NoError(t, err)
	
	_, err = db.Exec("DELETE FROM promo_codes WHERE tenant_id = 1")
	require.NoError(t, err)
}

func TestCreatePromoCodeAPI(t *testing.T) {
	router, db := setupPromoTestRouter(t)
	defer db.Close()
	defer cleanupPromoTestData(t, db)
	
	reqBody := promo.CreatePromoCodeRequest{
		Code:          "APITEST",
		Description:   "اختبار API",
		DiscountType:  promo.DiscountTypePercentage,
		DiscountValue: 20.0,
		ValidFrom:     time.Now(),
		ValidUntil:    time.Now().Add(30 * 24 * time.Hour),
		PerUserLimit:  1,
		ApplicableTo:  promo.ApplicableToAll,
	}
	
	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/admin/promo-codes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusCreated, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])
}

func TestListPromoCodesAPI(t *testing.T) {
	router, db := setupPromoTestRouter(t)
	defer db.Close()
	defer cleanupPromoTestData(t, db)
	
	// إنشاء بعض الأكواد
	manager := promo.NewManager(db)
	ctx := context.Background()
	
	for i := 1; i <= 3; i++ {
		req := &promo.CreatePromoCodeRequest{
			Code:          "LIST" + string(rune('0'+i)),
			Description:   "كود للاختبار",
			DiscountType:  promo.DiscountTypePercentage,
			DiscountValue: 10.0,
			ValidFrom:     time.Now(),
			ValidUntil:    time.Now().Add(30 * 24 * time.Hour),
			PerUserLimit:  1,
			ApplicableTo:  promo.ApplicableToAll,
		}
		
		_, err := manager.CreatePromoCode(ctx, 1, req)
		require.NoError(t, err)
	}
	
	req, _ := http.NewRequest("GET", "/api/admin/promo-codes", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.True(t, response["success"].(bool))
	assert.GreaterOrEqual(t, int(response["count"].(float64)), 3)
}

func TestValidatePromoCodeAPI(t *testing.T) {
	router, db := setupPromoTestRouter(t)
	defer db.Close()
	defer cleanupPromoTestData(t, db)
	
	// إنشاء كود خصم
	manager := promo.NewManager(db)
	ctx := context.Background()
	
	createReq := &promo.CreatePromoCodeRequest{
		Code:          "VALIDATE",
		Description:   "اختبار التحقق",
		DiscountType:  promo.DiscountTypePercentage,
		DiscountValue: 15.0,
		ValidFrom:     time.Now(),
		ValidUntil:    time.Now().Add(30 * 24 * time.Hour),
		PerUserLimit:  1,
		ApplicableTo:  promo.ApplicableToAll,
	}
	
	_, err := manager.CreatePromoCode(ctx, 1, createReq)
	require.NoError(t, err)
	
	// التحقق من الكود
	validateReq := promo.ApplyPromoCodeRequest{
		Code:      "VALIDATE",
		UserID:    1,
		OrderType: promo.OrderTypeSubscription,
		Amount:    100.0,
	}
	
	body, _ := json.Marshal(validateReq)
	req, _ := http.NewRequest("POST", "/api/promo-codes/validate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.True(t, response["success"].(bool))
	assert.True(t, response["valid"].(bool))
	
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "VALIDATE", data["code"])
	assert.Equal(t, 100.0, data["original_amount"])
	assert.Equal(t, 15.0, data["discount_amount"])
	assert.Equal(t, 85.0, data["final_amount"])
}

func TestApplyPromoCodeAPI(t *testing.T) {
	router, db := setupPromoTestRouter(t)
	defer db.Close()
	defer cleanupPromoTestData(t, db)
	
	// إنشاء كود خصم
	manager := promo.NewManager(db)
	ctx := context.Background()
	
	createReq := &promo.CreatePromoCodeRequest{
		Code:          "APPLY",
		Description:   "اختبار التطبيق",
		DiscountType:  promo.DiscountTypePercentage,
		DiscountValue: 20.0,
		ValidFrom:     time.Now(),
		ValidUntil:    time.Now().Add(30 * 24 * time.Hour),
		PerUserLimit:  1,
		ApplicableTo:  promo.ApplicableToAll,
	}
	
	_, err := manager.CreatePromoCode(ctx, 1, createReq)
	require.NoError(t, err)
	
	// تطبيق الكود
	applyReq := promo.ApplyPromoCodeRequest{
		Code:      "APPLY",
		UserID:    1,
		OrderType: promo.OrderTypeSubscription,
		Amount:    100.0,
	}
	
	body, _ := json.Marshal(applyReq)
	req, _ := http.NewRequest("POST", "/api/promo-codes/apply", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.True(t, response["success"].(bool))
	
	data := response["data"].(map[string]interface{})
	assert.Equal(t, 100.0, data["original_amount"])
	assert.Equal(t, 20.0, data["discount_amount"])
	assert.Equal(t, 80.0, data["final_amount"])
}

func TestGetPromoPerformanceAPI(t *testing.T) {
	router, db := setupPromoTestRouter(t)
	defer db.Close()
	defer cleanupPromoTestData(t, db)
	
	// إنشاء كود خصم
	manager := promo.NewManager(db)
	ctx := context.Background()
	
	createReq := &promo.CreatePromoCodeRequest{
		Code:          "PERFORMANCE",
		Description:   "اختبار الأداء",
		DiscountType:  promo.DiscountTypePercentage,
		DiscountValue: 10.0,
		ValidFrom:     time.Now(),
		ValidUntil:    time.Now().Add(30 * 24 * time.Hour),
		PerUserLimit:  5,
		ApplicableTo:  promo.ApplicableToAll,
	}
	
	promoCode, err := manager.CreatePromoCode(ctx, 1, createReq)
	require.NoError(t, err)
	
	// تطبيق الكود عدة مرات
	for i := int64(1); i <= 3; i++ {
		applyReq := &promo.ApplyPromoCodeRequest{
			Code:      "PERFORMANCE",
			UserID:    i,
			OrderType: promo.OrderTypeSubscription,
			Amount:    100.0,
		}
		
		_, err := manager.ApplyPromoCode(ctx, 1, applyReq)
		require.NoError(t, err)
	}
	
	// الحصول على الأداء
	req, _ := http.NewRequest("GET", "/api/admin/promo-codes/"+string(rune(promoCode.ID))+"/performance", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])
}

func TestUpdatePromoCodeAPI(t *testing.T) {
	router, db := setupPromoTestRouter(t)
	defer db.Close()
	defer cleanupPromoTestData(t, db)
	
	// إنشاء كود خصم
	manager := promo.NewManager(db)
	ctx := context.Background()
	
	createReq := &promo.CreatePromoCodeRequest{
		Code:          "UPDATE",
		Description:   "كود للتحديث",
		DiscountType:  promo.DiscountTypePercentage,
		DiscountValue: 10.0,
		ValidFrom:     time.Now(),
		ValidUntil:    time.Now().Add(30 * 24 * time.Hour),
		PerUserLimit:  1,
		ApplicableTo:  promo.ApplicableToAll,
	}
	
	promoCode, err := manager.CreatePromoCode(ctx, 1, createReq)
	require.NoError(t, err)
	
	// تحديث الكود
	updates := map[string]interface{}{
		"description":   "وصف محدث",
		"per_user_limit": 3,
	}
	
	body, _ := json.Marshal(updates)
	req, _ := http.NewRequest("PUT", "/api/admin/promo-codes/"+string(rune(promoCode.ID)), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.True(t, response["success"].(bool))
}

func TestDeletePromoCodeAPI(t *testing.T) {
	router, db := setupPromoTestRouter(t)
	defer db.Close()
	defer cleanupPromoTestData(t, db)
	
	// إنشاء كود خصم
	manager := promo.NewManager(db)
	ctx := context.Background()
	
	createReq := &promo.CreatePromoCodeRequest{
		Code:          "DELETE",
		Description:   "كود للحذف",
		DiscountType:  promo.DiscountTypePercentage,
		DiscountValue: 10.0,
		ValidFrom:     time.Now(),
		ValidUntil:    time.Now().Add(30 * 24 * time.Hour),
		PerUserLimit:  1,
		ApplicableTo:  promo.ApplicableToAll,
	}
	
	promoCode, err := manager.CreatePromoCode(ctx, 1, createReq)
	require.NoError(t, err)
	
	// حذف الكود
	req, _ := http.NewRequest("DELETE", "/api/admin/promo-codes/"+string(rune(promoCode.ID)), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.True(t, response["success"].(bool))
}

func TestInvalidPromoCodeAPI(t *testing.T) {
	router, db := setupPromoTestRouter(t)
	defer db.Close()
	defer cleanupPromoTestData(t, db)
	
	// محاولة التحقق من كود غير موجود
	validateReq := promo.ApplyPromoCodeRequest{
		Code:      "NOTEXIST",
		UserID:    1,
		OrderType: promo.OrderTypeSubscription,
		Amount:    100.0,
	}
	
	body, _ := json.Marshal(validateReq)
	req, _ := http.NewRequest("POST", "/api/promo-codes/validate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusBadRequest, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.False(t, response["success"].(bool))
	assert.NotEmpty(t, response["error"])
}
