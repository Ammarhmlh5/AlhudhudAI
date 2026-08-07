package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/al-hudhud/backend/internal/accounting"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AdminAccountingHandler handles admin accounting (cashboxes & vouchers) endpoints.
type AdminAccountingHandler struct {
	manager *accounting.Manager
}

// NewAdminAccountingHandler creates a new accounting handler.
func NewAdminAccountingHandler(manager *accounting.Manager) *AdminAccountingHandler {
	return &AdminAccountingHandler{manager: manager}
}

// userIdFromContext returns the authenticated admin's UUID, or nil.
func userIdFromContext(c *gin.Context) *uuid.UUID {
	if userIDVal, exists := c.Get("user_id"); exists {
		if userIDStr, ok := userIDVal.(string); ok {
			if uid, err := uuid.Parse(userIDStr); err == nil {
				return &uid
			}
		}
	}
	return nil
}

// ListCashboxes handles GET /admin/accounting/cashboxes
func (h *AdminAccountingHandler) ListCashboxes(c *gin.Context) {
	cashboxes, err := h.manager.ListCashboxes(c.Request.Context())
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "internal_error", "Failed to retrieve cashboxes", nil)
		return
	}
	RespondWithSuccess(c, http.StatusOK, cashboxes, "")
}

// CreateCashbox handles POST /admin/accounting/cashboxes
func (h *AdminAccountingHandler) CreateCashbox(c *gin.Context) {
	var req accounting.CreateCashboxRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "invalid_json", "Invalid request body", nil)
		return
	}

	cashbox, err := h.manager.CreateCashbox(c.Request.Context(), &req, userIdFromContext(c))
	if err != nil {
		RespondWithAccountingError(c, err)
		return
	}
	RespondWithSuccess(c, http.StatusCreated, cashbox, "تم إنشاء الصندوق بنجاح")
}

// GetCashbox handles GET /admin/accounting/cashboxes/:id
func (h *AdminAccountingHandler) GetCashbox(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	cashbox, err := h.manager.GetCashbox(c.Request.Context(), id)
	if err != nil {
		RespondWithAccountingError(c, err)
		return
	}
	RespondWithSuccess(c, http.StatusOK, cashbox, "")
}

// UpdateCashbox handles PUT /admin/accounting/cashboxes/:id
func (h *AdminAccountingHandler) UpdateCashbox(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	var req accounting.CreateCashboxRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "invalid_json", "Invalid request body", nil)
		return
	}

	cashbox, err := h.manager.UpdateCashbox(c.Request.Context(), id, &req)
	if err != nil {
		RespondWithAccountingError(c, err)
		return
	}
	RespondWithSuccess(c, http.StatusOK, cashbox, "تم تعديل الصندوق بنجاح")
}

// SetCashboxStatus handles PUT /admin/accounting/cashboxes/:id/status
func (h *AdminAccountingHandler) SetCashboxStatus(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	var req struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "invalid_json", "Invalid request body", nil)
		return
	}

	cashbox, err := h.manager.SetCashboxStatus(c.Request.Context(), id, req.Status)
	if err != nil {
		RespondWithAccountingError(c, err)
		return
	}
	RespondWithSuccess(c, http.StatusOK, cashbox, "تم تحديث حالة الصندوق")
}

// GetCashboxStatement handles POST /admin/accounting/cashboxes/:id/statement
func (h *AdminAccountingHandler) GetCashboxStatement(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	var req struct {
		From *time.Time `json:"from"`
		To   *time.Time `json:"to"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "invalid_json", "Invalid request body", nil)
		return
	}

	statement, err := h.manager.GetStatement(c.Request.Context(), id, req.From, req.To)
	if err != nil {
		RespondWithAccountingError(c, err)
		return
	}
	RespondWithSuccess(c, http.StatusOK, statement, "")
}

// ListReceipts handles GET /admin/accounting/receipts
func (h *AdminAccountingHandler) ListReceipts(c *gin.Context) {
	filter, ok := parseVoucherFilter(c)
	if !ok {
		return
	}
	vouchers, total, err := h.manager.ListReceipts(c.Request.Context(), filter)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "internal_error", "Failed to retrieve receipts", nil)
		return
	}
	RespondWithSuccess(c, http.StatusOK, map[string]interface{}{
		"vouchers": vouchers,
		"count":    total,
	}, "")
}

// CreateReceipt handles POST /admin/accounting/receipts
func (h *AdminAccountingHandler) CreateReceipt(c *gin.Context) {
	var req accounting.ReceiptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "invalid_json", "Invalid request body", nil)
		return
	}

	voucher, err := h.manager.CreateReceipt(c.Request.Context(), &req, userIdFromContext(c))
	if err != nil {
		RespondWithAccountingError(c, err)
		return
	}
	RespondWithSuccess(c, http.StatusCreated, voucher, "تم إنشاء سند القبض بنجاح")
}

// GetReceipt handles GET /admin/accounting/receipts/:id
func (h *AdminAccountingHandler) GetReceipt(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	voucher, err := h.manager.GetReceipt(c.Request.Context(), id)
	if err != nil {
		RespondWithAccountingError(c, err)
		return
	}
	RespondWithSuccess(c, http.StatusOK, voucher, "")
}

// PostReceipt handles POST /admin/accounting/receipts/:id/post
func (h *AdminAccountingHandler) PostReceipt(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	voucher, err := h.manager.PostReceipt(c.Request.Context(), id, userIdFromContext(c))
	if err != nil {
		RespondWithAccountingError(c, err)
		return
	}
	RespondWithSuccess(c, http.StatusOK, voucher, "تم ترحيل سند القبض بنجاح")
}

// ReverseReceipt handles POST /admin/accounting/receipts/:id/reverse
func (h *AdminAccountingHandler) ReverseReceipt(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	voucher, err := h.manager.ReverseReceipt(c.Request.Context(), id, userIdFromContext(c))
	if err != nil {
		RespondWithAccountingError(c, err)
		return
	}
	RespondWithSuccess(c, http.StatusOK, voucher, "تم عكس سند القبض بنجاح")
}

// ListPayments handles GET /admin/accounting/payments
func (h *AdminAccountingHandler) ListPayments(c *gin.Context) {
	filter, ok := parseVoucherFilter(c)
	if !ok {
		return
	}
	vouchers, total, err := h.manager.ListPayments(c.Request.Context(), filter)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "internal_error", "Failed to retrieve payments", nil)
		return
	}
	RespondWithSuccess(c, http.StatusOK, map[string]interface{}{
		"vouchers": vouchers,
		"count":    total,
	}, "")
}

// CreatePayment handles POST /admin/accounting/payments
func (h *AdminAccountingHandler) CreatePayment(c *gin.Context) {
	var req accounting.PaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "invalid_json", "Invalid request body", nil)
		return
	}

	voucher, err := h.manager.CreatePayment(c.Request.Context(), &req, userIdFromContext(c))
	if err != nil {
		RespondWithAccountingError(c, err)
		return
	}
	RespondWithSuccess(c, http.StatusCreated, voucher, "تم إنشاء سند الصرف بنجاح")
}

// GetPayment handles GET /admin/accounting/payments/:id
func (h *AdminAccountingHandler) GetPayment(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	voucher, err := h.manager.GetPayment(c.Request.Context(), id)
	if err != nil {
		RespondWithAccountingError(c, err)
		return
	}
	RespondWithSuccess(c, http.StatusOK, voucher, "")
}

// PostPayment handles POST /admin/accounting/payments/:id/post
func (h *AdminAccountingHandler) PostPayment(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	voucher, err := h.manager.PostPayment(c.Request.Context(), id, userIdFromContext(c))
	if err != nil {
		RespondWithAccountingError(c, err)
		return
	}
	RespondWithSuccess(c, http.StatusOK, voucher, "تم ترحيل سند الصرف بنجاح")
}

// ReversePayment handles POST /admin/accounting/payments/:id/reverse
func (h *AdminAccountingHandler) ReversePayment(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	voucher, err := h.manager.ReversePayment(c.Request.Context(), id, userIdFromContext(c))
	if err != nil {
		RespondWithAccountingError(c, err)
		return
	}
	RespondWithSuccess(c, http.StatusOK, voucher, "تم عكس سند الصرف بنجاح")
}

// PreviewNextNumber handles GET /admin/accounting/numbering/:type
func (h *AdminAccountingHandler) PreviewNextNumber(c *gin.Context) {
	number, err := h.manager.PreviewNextNumber(c.Request.Context(), c.Param("type"))
	if err != nil {
		RespondWithAccountingError(c, err)
		return
	}
	RespondWithSuccess(c, http.StatusOK, map[string]string{"number": number}, "")
}

func parseUUIDParam(c *gin.Context, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, "invalid_id", "Invalid "+name+" format", nil)
		return uuid.Nil, false
	}
	return id, true
}

func parseVoucherFilter(c *gin.Context) (*accounting.VoucherFilter, bool) {
	filter := &accounting.VoucherFilter{
		Limit:  50,
		Offset: 0,
	}

	if s := c.Query("cashbox_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			RespondWithError(c, http.StatusBadRequest, "invalid_cashbox_id", "Invalid cashbox ID format", nil)
			return nil, false
		}
		filter.CashboxID = &id
	}
	if s := c.Query("status"); s != "" {
		filter.Status = &s
	}
	if s := c.Query("from"); s != "" {
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			RespondWithError(c, http.StatusBadRequest, "invalid_from", "Invalid from date format (use YYYY-MM-DD)", nil)
			return nil, false
		}
		filter.From = &t
	}
	if s := c.Query("to"); s != "" {
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			RespondWithError(c, http.StatusBadRequest, "invalid_to", "Invalid to date format (use YYYY-MM-DD)", nil)
			return nil, false
		}
		filter.To = &t
	}
	if s := c.Query("limit"); s != "" {
		var n int
		if _, err := fmtSscanf(s, &n); err == nil && n > 0 && n <= 500 {
			filter.Limit = n
		}
	}
	if s := c.Query("offset"); s != "" {
		var n int
		if _, err := fmtSscanf(s, &n); err == nil && n >= 0 {
			filter.Offset = n
		}
	}
	return filter, true
}

func fmtSscanf(s string, dst *int) (int, error) {
	_, err := fmt.Sscanf(s, "%d", dst)
	return 1, err
}

func RespondWithAccountingError(c *gin.Context, err error) {
	switch err {
	case accounting.ErrCashboxNotFound:
		RespondWithError(c, http.StatusNotFound, "cashbox_not_found", "الصندوق غير موجود", nil)
	case accounting.ErrCashboxInactive:
		RespondWithError(c, http.StatusConflict, "cashbox_inactive", "الصندوق غير نشط", nil)
	case accounting.ErrVoucherNotFound:
		RespondWithError(c, http.StatusNotFound, "voucher_not_found", "السند غير موجود", nil)
	case accounting.ErrVoucherNotPostable:
		RespondWithError(c, http.StatusConflict, "voucher_not_postable", "السند ليس في حالة مسودة", nil)
	case accounting.ErrVoucherNotReversable:
		RespondWithError(c, http.StatusConflict, "voucher_not_reversable", "لا يمكن عكس سند غير مرحّل", nil)
	case accounting.ErrVoucherAlreadyReversed:
		RespondWithError(c, http.StatusConflict, "voucher_already_reversed", "السند معكوس مسبقاً", nil)
	case accounting.ErrInsufficientBalance:
		RespondWithError(c, http.StatusUnprocessableEntity, "insufficient_balance", "رصيد الصندوق غير كافٍ", nil)
	default:
		RespondWithError(c, http.StatusBadRequest, "validation_error", err.Error(), nil)
	}
}
