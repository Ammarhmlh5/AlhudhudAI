package promo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// Manager مدير أكواد الخصم
type Manager struct {
	db *sqlx.DB
}

// NewManager ينشئ مدير أكواد خصم جديد
func NewManager(db *sqlx.DB) *Manager {
	return &Manager{
		db: db,
	}
}

// CreatePromoCode ينشئ كود خصم جديد
// Requirements: 32.1, 32.2, 32.3, 32.4, 32.5, 32.6, 32.7
func (m *Manager) CreatePromoCode(ctx context.Context, tenantID int64, req *CreatePromoCodeRequest) (*PromoCode, error) {
	// التحقق من صحة البيانات
	if err := m.validateCreateRequest(req); err != nil {
		return nil, err
	}

	// التحقق من عدم وجود كود بنفس الاسم
	exists, err := m.promoCodeExists(ctx, tenantID, req.Code)
	if err != nil {
		return nil, fmt.Errorf("failed to check promo code existence: %w", err)
	}
	if exists {
		return nil, ErrPromoCodeAlreadyExists
	}

	// إنشاء الكود
	query := `
		INSERT INTO promo_codes (
			tenant_id, code, description, discount_type, discount_value,
			max_discount_amount, min_purchase_amount, valid_from, valid_until,
			usage_limit, per_user_limit, applicable_to, applicable_package_ids,
			is_active
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		) RETURNING id, created_at, updated_at, usage_count
	`

	var promoCode PromoCode
	promoCode.TenantID = tenantID
	promoCode.Code = strings.ToUpper(req.Code) // تحويل الكود لأحرف كبيرة
	promoCode.Description = req.Description
	promoCode.DiscountType = req.DiscountType
	promoCode.DiscountValue = req.DiscountValue
	promoCode.MaxDiscountAmount = req.MaxDiscountAmount
	promoCode.MinPurchaseAmount = req.MinPurchaseAmount
	promoCode.ValidFrom = req.ValidFrom
	promoCode.ValidUntil = req.ValidUntil
	promoCode.UsageLimit = req.UsageLimit
	promoCode.PerUserLimit = req.PerUserLimit
	promoCode.ApplicableTo = req.ApplicableTo
	promoCode.ApplicablePackageIDs = req.ApplicablePackageIDs
	promoCode.IsActive = true

	err = m.db.QueryRowContext(
		ctx, query,
		promoCode.TenantID, promoCode.Code, promoCode.Description,
		promoCode.DiscountType, promoCode.DiscountValue,
		promoCode.MaxDiscountAmount, promoCode.MinPurchaseAmount,
		promoCode.ValidFrom, promoCode.ValidUntil,
		promoCode.UsageLimit, promoCode.PerUserLimit,
		promoCode.ApplicableTo, pq.Array(promoCode.ApplicablePackageIDs),
		promoCode.IsActive,
	).Scan(&promoCode.ID, &promoCode.CreatedAt, &promoCode.UpdatedAt, &promoCode.UsageCount)

	if err != nil {
		return nil, fmt.Errorf("failed to create promo code: %w", err)
	}

	return &promoCode, nil
}

// ValidatePromoCode يتحقق من صلاحية كود الخصم
// Requirements: 32.3, 32.4, 32.6, 32.7, 32.8
func (m *Manager) ValidatePromoCode(ctx context.Context, tenantID int64, req *ApplyPromoCodeRequest) (*PromoCode, error) {
	// الحصول على الكود
	promoCode, err := m.getPromoCodeByCode(ctx, tenantID, req.Code)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrPromoCodeNotFound
		}
		return nil, fmt.Errorf("failed to get promo code: %w", err)
	}

	// التحقق من أن الكود نشط
	if !promoCode.IsActive {
		return nil, ErrPromoCodeInactive
	}

	// التحقق من تاريخ الصلاحية
	now := time.Now()
	if now.Before(promoCode.ValidFrom) {
		return nil, ErrPromoCodeNotStarted
	}
	if now.After(promoCode.ValidUntil) {
		return nil, ErrPromoCodeExpired
	}

	// التحقق من حد الاستخدام الإجمالي
	if promoCode.UsageLimit != nil && promoCode.UsageCount >= *promoCode.UsageLimit {
		return nil, ErrPromoCodeUsageLimitReached
	}

	// التحقق من حد الاستخدام للمستخدم
	userUsageCount, err := m.getUserUsageCount(ctx, tenantID, promoCode.ID, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user usage count: %w", err)
	}
	if userUsageCount >= promoCode.PerUserLimit {
		return nil, ErrPromoCodeUserLimitReached
	}

	// التحقق من نطاق التطبيق
	if !m.isApplicable(promoCode, req) {
		return nil, ErrPromoCodeNotApplicable
	}

	// التحقق من الحد الأدنى للشراء
	if promoCode.MinPurchaseAmount != nil && req.Amount < *promoCode.MinPurchaseAmount {
		return nil, ErrMinPurchaseNotMet
	}

	return promoCode, nil
}

// ApplyPromoCode يطبق كود الخصم ويحسب المبلغ النهائي
// Requirements: 32.5, 32.8, 32.9, 32.10
func (m *Manager) ApplyPromoCode(ctx context.Context, tenantID int64, req *ApplyPromoCodeRequest) (*ApplyPromoCodeResponse, error) {
	// التحقق من صلاحية الكود
	promoCode, err := m.ValidatePromoCode(ctx, tenantID, req)
	if err != nil {
		return nil, err
	}

	// حساب قيمة الخصم
	discountAmount := m.calculateDiscount(promoCode, req.Amount)

	// حساب المبلغ النهائي
	finalAmount := req.Amount - discountAmount

	// تسجيل الاستخدام في transaction
	tx, err := m.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// تسجيل الاستخدام
	usageQuery := `
		INSERT INTO promo_code_usage (
			tenant_id, promo_code_id, user_id, order_type, order_id,
			original_amount, discount_amount, final_amount
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err = tx.ExecContext(
		ctx, usageQuery,
		tenantID, promoCode.ID, req.UserID, req.OrderType, req.OrderID,
		req.Amount, discountAmount, finalAmount,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to record promo code usage: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &ApplyPromoCodeResponse{
		PromoCodeID:    promoCode.ID,
		OriginalAmount: req.Amount,
		DiscountAmount: discountAmount,
		FinalAmount:    finalAmount,
		Message:        fmt.Sprintf("تم تطبيق كود الخصم %s بنجاح", promoCode.Code),
	}, nil
}

// GetPromoPerformance يحصل على أداء كود خصم
// Requirements: 32.11
func (m *Manager) GetPromoPerformance(ctx context.Context, tenantID int64, promoCodeID int64) (*PromoCodePerformance, error) {
	query := `
		SELECT 
			pc.id,
			pc.code,
			pc.description,
			pc.usage_count as total_usage_count,
			COALESCE(SUM(pcu.discount_amount), 0) as total_discount_given,
			COALESCE(SUM(pcu.final_amount), 0) as total_revenue,
			COUNT(DISTINCT pcu.user_id) as unique_users
		FROM promo_codes pc
		LEFT JOIN promo_code_usage pcu ON pc.id = pcu.promo_code_id AND pc.tenant_id = pcu.tenant_id
		WHERE pc.tenant_id = $1 AND pc.id = $2
		GROUP BY pc.id, pc.code, pc.description, pc.usage_count
	`

	var performance PromoCodePerformance
	err := m.db.GetContext(ctx, &performance, query, tenantID, promoCodeID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrPromoCodeNotFound
		}
		return nil, fmt.Errorf("failed to get promo code performance: %w", err)
	}

	return &performance, nil
}

// ListPromoCodes يعرض قائمة أكواد الخصم
func (m *Manager) ListPromoCodes(ctx context.Context, tenantID int64, activeOnly bool) ([]*PromoCode, error) {
	query := `
		SELECT 
			id, tenant_id, code, description, discount_type, discount_value,
			max_discount_amount, min_purchase_amount, valid_from, valid_until,
			usage_limit, usage_count, per_user_limit, applicable_to,
			applicable_package_ids, is_active, created_at, updated_at, created_by
		FROM promo_codes
		WHERE tenant_id = $1
	`

	if activeOnly {
		query += " AND is_active = TRUE AND valid_until > NOW()"
	}

	query += " ORDER BY created_at DESC"

	var promoCodes []*PromoCode
	err := m.db.SelectContext(ctx, &promoCodes, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list promo codes: %w", err)
	}

	return promoCodes, nil
}

// UpdatePromoCode يحدث كود خصم
func (m *Manager) UpdatePromoCode(ctx context.Context, tenantID int64, promoCodeID int64, updates map[string]interface{}) error {
	// بناء استعلام التحديث ديناميكياً
	setClauses := []string{}
	args := []interface{}{tenantID, promoCodeID}
	argIndex := 3

	allowedFields := map[string]bool{
		"description":          true,
		"valid_until":          true,
		"usage_limit":          true,
		"per_user_limit":       true,
		"is_active":            true,
		"max_discount_amount":  true,
		"min_purchase_amount":  true,
	}

	for field, value := range updates {
		if !allowedFields[field] {
			continue
		}
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", field, argIndex))
		args = append(args, value)
		argIndex++
	}

	if len(setClauses) == 0 {
		return fmt.Errorf("no valid fields to update")
	}

	query := fmt.Sprintf(`
		UPDATE promo_codes
		SET %s, updated_at = NOW()
		WHERE tenant_id = $1 AND id = $2
	`, strings.Join(setClauses, ", "))

	result, err := m.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update promo code: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrPromoCodeNotFound
	}

	return nil
}

// DeletePromoCode يحذف كود خصم (soft delete)
func (m *Manager) DeletePromoCode(ctx context.Context, tenantID int64, promoCodeID int64) error {
	query := `
		UPDATE promo_codes
		SET is_active = FALSE, updated_at = NOW()
		WHERE tenant_id = $1 AND id = $2
	`

	result, err := m.db.ExecContext(ctx, query, tenantID, promoCodeID)
	if err != nil {
		return fmt.Errorf("failed to delete promo code: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrPromoCodeNotFound
	}

	return nil
}

// ============================================================================
// Helper Functions
// ============================================================================

// validateCreateRequest يتحقق من صحة طلب إنشاء كود خصم
func (m *Manager) validateCreateRequest(req *CreatePromoCodeRequest) error {
	// التحقق من نطاق التاريخ
	if req.ValidUntil.Before(req.ValidFrom) {
		return fmt.Errorf("valid_until must be after valid_from")
	}

	// التحقق من قيمة الخصم
	if req.DiscountType == DiscountTypePercentage {
		if req.DiscountValue <= 0 || req.DiscountValue > 100 {
			return fmt.Errorf("percentage discount must be between 0 and 100")
		}
	}

	// التحقق من الباقات المحددة
	if req.ApplicableTo == ApplicableToPackages && len(req.ApplicablePackageIDs) == 0 {
		return fmt.Errorf("applicable_package_ids required when applicable_to is 'packages'")
	}

	return nil
}

// promoCodeExists يتحقق من وجود كود خصم بنفس الاسم
func (m *Manager) promoCodeExists(ctx context.Context, tenantID int64, code string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM promo_codes WHERE tenant_id = $1 AND code = $2)`
	var exists bool
	err := m.db.GetContext(ctx, &exists, query, tenantID, strings.ToUpper(code))
	return exists, err
}

// getPromoCodeByCode يحصل على كود خصم بالكود
func (m *Manager) getPromoCodeByCode(ctx context.Context, tenantID int64, code string) (*PromoCode, error) {
	query := `
		SELECT 
			id, tenant_id, code, description, discount_type, discount_value,
			max_discount_amount, min_purchase_amount, valid_from, valid_until,
			usage_limit, usage_count, per_user_limit, applicable_to,
			applicable_package_ids, is_active, created_at, updated_at, created_by
		FROM promo_codes
		WHERE tenant_id = $1 AND code = $2
	`

	var promoCode PromoCode
	err := m.db.GetContext(ctx, &promoCode, query, tenantID, strings.ToUpper(code))
	if err != nil {
		return nil, err
	}

	return &promoCode, nil
}

// getUserUsageCount يحصل على عدد مرات استخدام المستخدم للكود
func (m *Manager) getUserUsageCount(ctx context.Context, tenantID int64, promoCodeID int64, userID int64) (int, error) {
	query := `
		SELECT COUNT(*) 
		FROM promo_code_usage 
		WHERE tenant_id = $1 AND promo_code_id = $2 AND user_id = $3
	`

	var count int
	err := m.db.GetContext(ctx, &count, query, tenantID, promoCodeID, userID)
	return count, err
}

// isApplicable يتحقق من إمكانية تطبيق الكود على الطلب
func (m *Manager) isApplicable(promoCode *PromoCode, req *ApplyPromoCodeRequest) bool {
	// إذا كان الكود قابل للتطبيق على الكل
	if promoCode.ApplicableTo == ApplicableToAll {
		return true
	}

	// التحقق من نوع الطلب
	switch promoCode.ApplicableTo {
	case ApplicableToPackages:
		if req.OrderType != OrderTypeSubscription {
			return false
		}
		// التحقق من الباقة المحددة
		if req.PackageID == nil {
			return false
		}
		if len(promoCode.ApplicablePackageIDs) > 0 {
			found := false
			for _, pkgID := range promoCode.ApplicablePackageIDs {
				if pkgID == *req.PackageID {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	case ApplicableToTopup:
		if req.OrderType != OrderTypeTopup {
			return false
		}
	case ApplicableToGateway:
		if req.OrderType != OrderTypeGateway {
			return false
		}
	}

	return true
}

// calculateDiscount يحسب قيمة الخصم
func (m *Manager) calculateDiscount(promoCode *PromoCode, amount float64) float64 {
	var discount float64

	if promoCode.DiscountType == DiscountTypePercentage {
		discount = amount * (promoCode.DiscountValue / 100.0)
		
		// تطبيق الحد الأقصى للخصم إن وجد
		if promoCode.MaxDiscountAmount != nil && discount > *promoCode.MaxDiscountAmount {
			discount = *promoCode.MaxDiscountAmount
		}
	} else {
		discount = promoCode.DiscountValue
		
		// التأكد من أن الخصم لا يتجاوز المبلغ الأصلي
		if discount > amount {
			discount = amount
		}
	}

	return discount
}
