package promo

import (
	"time"
)

// DiscountType نوع الخصم
type DiscountType string

const (
	DiscountTypePercentage  DiscountType = "percentage"
	DiscountTypeFixedAmount DiscountType = "fixed_amount"
)

// ApplicableTo نطاق تطبيق الكود
type ApplicableTo string

const (
	ApplicableToAll      ApplicableTo = "all"
	ApplicableToPackages ApplicableTo = "packages"
	ApplicableToTopup    ApplicableTo = "topup"
	ApplicableToGateway  ApplicableTo = "gateway"
)

// OrderType نوع الطلب
type OrderType string

const (
	OrderTypeSubscription OrderType = "subscription"
	OrderTypeTopup        OrderType = "topup"
	OrderTypeGateway      OrderType = "gateway"
)

// PromoCode يمثل كود خصم
type PromoCode struct {
	ID                   int64        `db:"id" json:"id"`
	TenantID             int64        `db:"tenant_id" json:"tenant_id"`
	Code                 string       `db:"code" json:"code"`
	Description          string       `db:"description" json:"description"`
	DiscountType         DiscountType `db:"discount_type" json:"discount_type"`
	DiscountValue        float64      `db:"discount_value" json:"discount_value"`
	MaxDiscountAmount    *float64     `db:"max_discount_amount" json:"max_discount_amount,omitempty"`
	MinPurchaseAmount    *float64     `db:"min_purchase_amount" json:"min_purchase_amount,omitempty"`
	ValidFrom            time.Time    `db:"valid_from" json:"valid_from"`
	ValidUntil           time.Time    `db:"valid_until" json:"valid_until"`
	UsageLimit           *int         `db:"usage_limit" json:"usage_limit,omitempty"`
	UsageCount           int          `db:"usage_count" json:"usage_count"`
	PerUserLimit         int          `db:"per_user_limit" json:"per_user_limit"`
	ApplicableTo         ApplicableTo `db:"applicable_to" json:"applicable_to"`
	ApplicablePackageIDs []int64      `db:"applicable_package_ids" json:"applicable_package_ids,omitempty"`
	IsActive             bool         `db:"is_active" json:"is_active"`
	CreatedAt            time.Time    `db:"created_at" json:"created_at"`
	UpdatedAt            time.Time    `db:"updated_at" json:"updated_at"`
	CreatedBy            *int64       `db:"created_by" json:"created_by,omitempty"`
}

// PromoCodeUsage يمثل سجل استخدام كود خصم
type PromoCodeUsage struct {
	ID             int64     `db:"id" json:"id"`
	TenantID       int64     `db:"tenant_id" json:"tenant_id"`
	PromoCodeID    int64     `db:"promo_code_id" json:"promo_code_id"`
	UserID         int64     `db:"user_id" json:"user_id"`
	OrderType      OrderType `db:"order_type" json:"order_type"`
	OrderID        *int64    `db:"order_id" json:"order_id,omitempty"`
	OriginalAmount float64   `db:"original_amount" json:"original_amount"`
	DiscountAmount float64   `db:"discount_amount" json:"discount_amount"`
	FinalAmount    float64   `db:"final_amount" json:"final_amount"`
	UsedAt         time.Time `db:"used_at" json:"used_at"`
}

// CreatePromoCodeRequest طلب إنشاء كود خصم
type CreatePromoCodeRequest struct {
	Code                 string       `json:"code" binding:"required,min=3,max=50"`
	Description          string       `json:"description"`
	DiscountType         DiscountType `json:"discount_type" binding:"required,oneof=percentage fixed_amount"`
	DiscountValue        float64      `json:"discount_value" binding:"required,gt=0"`
	MaxDiscountAmount    *float64     `json:"max_discount_amount,omitempty"`
	MinPurchaseAmount    *float64     `json:"min_purchase_amount,omitempty"`
	ValidFrom            time.Time    `json:"valid_from" binding:"required"`
	ValidUntil           time.Time    `json:"valid_until" binding:"required"`
	UsageLimit           *int         `json:"usage_limit,omitempty"`
	PerUserLimit         int          `json:"per_user_limit" binding:"required,gt=0"`
	ApplicableTo         ApplicableTo `json:"applicable_to" binding:"required,oneof=all packages topup gateway"`
	ApplicablePackageIDs []int64      `json:"applicable_package_ids,omitempty"`
}

// ApplyPromoCodeRequest طلب تطبيق كود خصم
type ApplyPromoCodeRequest struct {
	Code          string    `json:"code" binding:"required"`
	UserID        int64     `json:"user_id" binding:"required"`
	OrderType     OrderType `json:"order_type" binding:"required,oneof=subscription topup gateway"`
	OrderID       *int64    `json:"order_id,omitempty"`
	Amount        float64   `json:"amount" binding:"required,gt=0"`
	PackageID     *int64    `json:"package_id,omitempty"`
}

// ApplyPromoCodeResponse استجابة تطبيق كود خصم
type ApplyPromoCodeResponse struct {
	PromoCodeID    int64   `json:"promo_code_id"`
	OriginalAmount float64 `json:"original_amount"`
	DiscountAmount float64 `json:"discount_amount"`
	FinalAmount    float64 `json:"final_amount"`
	Message        string  `json:"message"`
}

// PromoCodePerformance أداء كود خصم
type PromoCodePerformance struct {
	PromoCodeID       int64   `json:"promo_code_id"`
	Code              string  `json:"code"`
	Description       string  `json:"description"`
	TotalUsageCount   int     `json:"total_usage_count"`
	TotalDiscountGiven float64 `json:"total_discount_given"`
	TotalRevenue      float64 `json:"total_revenue"`
	UniqueUsers       int     `json:"unique_users"`
}

// ValidationError خطأ التحقق من صحة الكود
type ValidationError struct {
	Code    string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// خطأ شائعة
var (
	ErrPromoCodeNotFound      = &ValidationError{Code: "PROMO_CODE_NOT_FOUND", Message: "كود الخصم غير موجود"}
	ErrPromoCodeExpired       = &ValidationError{Code: "PROMO_CODE_EXPIRED", Message: "كود الخصم منتهي الصلاحية"}
	ErrPromoCodeNotStarted    = &ValidationError{Code: "PROMO_CODE_NOT_STARTED", Message: "كود الخصم لم يبدأ بعد"}
	ErrPromoCodeInactive      = &ValidationError{Code: "PROMO_CODE_INACTIVE", Message: "كود الخصم غير نشط"}
	ErrPromoCodeUsageLimitReached = &ValidationError{Code: "USAGE_LIMIT_REACHED", Message: "تم الوصول للحد الأقصى لاستخدام الكود"}
	ErrPromoCodeUserLimitReached  = &ValidationError{Code: "USER_LIMIT_REACHED", Message: "لقد استخدمت هذا الكود الحد الأقصى المسموح"}
	ErrPromoCodeNotApplicable     = &ValidationError{Code: "NOT_APPLICABLE", Message: "كود الخصم غير قابل للتطبيق على هذا الطلب"}
	ErrMinPurchaseNotMet          = &ValidationError{Code: "MIN_PURCHASE_NOT_MET", Message: "المبلغ أقل من الحد الأدنى المطلوب"}
	ErrPromoCodeAlreadyExists     = &ValidationError{Code: "PROMO_CODE_EXISTS", Message: "كود الخصم موجود بالفعل"}
)
