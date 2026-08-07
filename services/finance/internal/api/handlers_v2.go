package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"finance/internal/models"
	"finance/internal/services"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// ─── Full Finance Handler ─────────────────────────────────────────────────────

type FinanceHandlerV2 struct {
	db             *gorm.DB
	accService     *services.AccountingService
	accManager     *services.AccountManager
	invoiceService *services.InvoiceService
}

func NewFinanceHandlerV2(db *gorm.DB, accSvc *services.AccountingService, accMgr *services.AccountManager, invSvc *services.InvoiceService) *FinanceHandlerV2 {
	return &FinanceHandlerV2{db: db, accService: accSvc, accManager: accMgr, invoiceService: invSvc}
}

// ─── Op 1: Charge Wallet (شحن رصيد شركة) POST /api/finance/charge ─────────────
func (h *FinanceHandlerV2) ChargeWalletHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CashboxCode     string          `json:"cashbox_code"`
		WalletAccountID uuid.UUID       `json:"wallet_account_id"`
		Amount          decimal.Decimal `json:"amount"`
		PayerName       string          `json:"payer_name"`
		Description     string          `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "بيانات غير صالحة", http.StatusBadRequest)
		return
	}
	entry, err := h.accService.ChargeWallet(r.Context(), req.CashboxCode, req.WalletAccountID, req.Amount, req.PayerName, req.Description, nil)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Auto-generate invoice (15% VAT)
	vatRate := decimal.NewFromFloat(0.15)
	h.invoiceService.GenerateInvoice(r.Context(), entry.ID, req.PayerName, req.Amount, vatRate)
	jsonOK(w, entry)
}

// ─── Op 2: Debit Cashbox (سند صرف) POST /api/finance/debit-cashbox ────────────
func (h *FinanceHandlerV2) DebitCashboxHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CashboxCode        string          `json:"cashbox_code"`
		ExpenseAccountCode string          `json:"expense_account_code"`
		Amount             decimal.Decimal `json:"amount"`
		PayeeName          string          `json:"payee_name"`
		Description        string          `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "بيانات غير صالحة", http.StatusBadRequest)
		return
	}
	entry, err := h.accService.DebitCashbox(r.Context(), req.CashboxCode, req.ExpenseAccountCode, req.Amount, req.PayeeName, req.Description, nil)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, entry)
}

// ─── Op 3: Create Cashbox (إضافة صندوق) POST /api/finance/cashboxes ────────────
func (h *FinanceHandlerV2) CreateCashboxHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code string `json:"code"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "بيانات غير صالحة", http.StatusBadRequest)
		return
	}
	cashbox, err := h.accManager.CreateCashbox(r.Context(), req.Code, req.Name, nil)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, cashbox)
}

// ─── Op 4: Get Wallet Balance (رصيد المحفظة) GET /api/finance/balance ──────────
func (h *FinanceHandlerV2) GetBalanceHandler(w http.ResponseWriter, r *http.Request) {
	idParam := r.URL.Query().Get("id")
	accID, err := uuid.Parse(idParam)
	if err != nil {
		jsonError(w, "معرف المحفظة غير صالح", http.StatusBadRequest)
		return
	}
	balance, err := h.accService.GetAccountBalance(r.Context(), accID)
	if err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}
	jsonOK(w, map[string]string{"balance": balance.StringFixed(2)})
}

// ─── Op 5: Cashbox Statement (كشف حساب) GET /api/finance/cashboxes/{id}/statement
func (h *FinanceHandlerV2) CashboxStatementHandler(w http.ResponseWriter, r *http.Request) {
	idParam := r.URL.Query().Get("id")
	cashboxID, err := uuid.Parse(idParam)
	if err != nil {
		jsonError(w, "معرف الصندوق غير صالح", http.StatusBadRequest)
		return
	}
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	from, _ := time.Parse("2006-01-02", fromStr)
	to, _ := time.Parse("2006-01-02", toStr)
	if to.IsZero() {
		to = time.Now()
	}
	lines, err := h.accService.GetCashboxStatement(r.Context(), cashboxID, from, to)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, lines)
}

// ─── Op 6: Reverse Entry (عكس سند) POST /api/finance/reverse ───────────────────
func (h *FinanceHandlerV2) ReverseEntryHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		EntryID uuid.UUID `json:"entry_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "بيانات غير صالحة", http.StatusBadRequest)
		return
	}
	reversal, err := h.accService.ReverseJournalEntry(r.Context(), req.EntryID, nil)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	jsonOK(w, reversal)
}

// ─── Op 7: Purchase Package (شراء باقة) POST /api/finance/purchase ─────────────
func (h *FinanceHandlerV2) PurchasePackageHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		WalletAccountID uuid.UUID       `json:"wallet_account_id"`
		Amount          decimal.Decimal `json:"amount"`
		PackageName     string          `json:"package_name"`
		CustomerName    string          `json:"customer_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "بيانات غير صالحة", http.StatusBadRequest)
		return
	}
	balance, err := h.accService.GetAccountBalance(r.Context(), req.WalletAccountID)
	if err != nil || balance.LessThan(req.Amount) {
		jsonError(w, "رصيد المحفظة غير كافٍ لشراء الباقة", http.StatusPaymentRequired)
		return
	}
	var revAccount models.Account
	if err := h.db.WithContext(r.Context()).Where("code = ?", "4100").First(&revAccount).Error; err != nil {
		jsonError(w, "حساب الإيرادات غير موجود", http.StatusInternalServerError)
		return
	}
	lines := []models.JournalEntryLine{
		{AccountID: req.WalletAccountID, Debit: req.Amount, Credit: decimal.Zero, Description: "استقطاع قيمة باقة: " + req.PackageName},
		{AccountID: revAccount.ID, Debit: decimal.Zero, Credit: req.Amount, Description: "إيراد بيع باقة: " + req.PackageName},
	}
	entry, err := h.accService.CreateJournalEntry(r.Context(), "شراء باقة: "+req.PackageName, lines, nil)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Auto-generate invoice (15% VAT)
	h.invoiceService.GenerateInvoice(r.Context(), entry.ID, req.CustomerName, req.Amount, decimal.NewFromFloat(0.15))
	jsonOK(w, entry)
}

// ─── Op 8: List Transactions (قائمة المعاملات) GET /api/finance/transactions ─────────────
func (h *FinanceHandlerV2) ListTransactionsHandler(w http.ResponseWriter, r *http.Request) {
	limit := 20
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	// Get journal entries (which represent transactions)
	var entries []models.JournalEntry
	query := h.db.WithContext(r.Context()).Preload("Lines").Preload("Lines.Account").
		Order("created_at DESC").Offset(offset).Limit(limit)

	// Filter by status if provided
	if status := r.URL.Query().Get("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Find(&entries).Error; err != nil {
		jsonError(w, "فشل تحميل المعاملات", http.StatusInternalServerError)
		return
	}

	// Transform to transaction format for frontend
	type Transaction struct {
		ID          string          `json:"id"`
		EntryNumber string          `json:"entry_number"`
		Description string          `json:"description"`
		Amount      decimal.Decimal `json:"amount"`
		Type        string          `json:"type"`
		Status      string          `json:"status"`
		Date        time.Time       `json:"date"`
	}
	var transactions []Transaction
	for _, entry := range entries {
		var amount decimal.Decimal
		var txnType string
		for _, line := range entry.Lines {
			if line.Debit.IsPositive() {
				amount = amount.Add(line.Debit)
				txnType = "DEBIT"
			}
			if line.Credit.IsPositive() {
				amount = amount.Add(line.Credit)
				txnType = "CREDIT"
			}
		}
		transactions = append(transactions, Transaction{
			ID:          entry.ID.String(),
			EntryNumber: entry.EntryNumber,
			Description: entry.Description,
			Amount:      amount,
			Type:        txnType,
			Status:      string(entry.Status),
			Date:        entry.CreatedAt,
		})
	}

	jsonOK(w, transactions)
}

// ─── Op 8b: List Cashboxes (قائمة الصناديق) GET /api/finance/cashboxes ─────────────
func (h *FinanceHandlerV2) ListCashboxesHandler(w http.ResponseWriter, r *http.Request) {
	var accounts []models.Account
	// Get cashbox-type accounts (1100 series)
	if err := h.db.WithContext(r.Context()).Where("code LIKE ?", "11%").
		Order("code ASC").Find(&accounts).Error; err != nil {
		jsonError(w, "فشل تحميل الصناديق", http.StatusInternalServerError)
		return
	}

	type Cashbox struct {
		ID     string `json:"id"`
		Code   string `json:"code"`
		Name   string `json:"bank"`
		IBAN   string `json:"iban"`
		Status string `json:"status"`
	}
	var cashboxes []Cashbox
	for _, acc := range accounts {
		balance, _ := h.accService.GetAccountBalance(r.Context(), acc.ID)
		_ = balance // يمكن إرساله إذا احتاجFrontend
		cashboxes = append(cashboxes, Cashbox{
			ID:     acc.ID.String(),
			Code:   acc.Code,
			Name:   acc.Name,
			IBAN:   "", // يمكن إضافة إذا كان موجود في الحقل
			Status: "Active",
		})
	}

	jsonOK(w, cashboxes)
}

// ─── Op 9: List Invoices (قائمة الفواتير) GET /api/finance/invoices ─────────────
func (h *FinanceHandlerV2) ListInvoicesHandler(w http.ResponseWriter, r *http.Request) {
	var invoices []models.Invoice
	query := h.db.WithContext(r.Context()).Order("created_at DESC")
	if tenantName := r.URL.Query().Get("customer"); tenantName != "" {
		query = query.Where("customer_name ILIKE ?", "%"+tenantName+"%")
	}
	if err := query.Find(&invoices).Error; err != nil {
		jsonError(w, "فشل تحميل الفواتير", http.StatusInternalServerError)
		return
	}
	jsonOK(w, invoices)
}

// ─── Op 10: Invoice HTML (طباعة فاتورة) GET /api/finance/invoices/{id}/html ─────
func (h *FinanceHandlerV2) InvoiceHTMLHandler(w http.ResponseWriter, r *http.Request) {
	idParam := r.URL.Query().Get("id")
	invID, err := uuid.Parse(idParam)
	if err != nil {
		http.Error(w, "معرف الفاتورة غير صالح", http.StatusBadRequest)
		return
	}
	html, err := h.invoiceService.GetInvoiceHTML(r.Context(), invID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

// ─── Op 11: Auto-create wallet (عند تسجيل مستخدم جديد) POST /api/finance/wallets
func (h *FinanceHandlerV2) CreateWalletHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ReferenceID   uuid.UUID `json:"reference_id"`
		ReferenceType string    `json:"reference_type"` // TENANT or RESELLER
		Name          string    `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "بيانات غير صالحة", http.StatusBadRequest)
		return
	}
	var account *models.Account
	var err error
	if req.ReferenceType == "TENANT" {
		account, err = h.accManager.CreateUserWalletAccount(r.Context(), req.ReferenceID, req.Name)
	} else if req.ReferenceType == "RESELLER" {
		account, err = h.accManager.CreateResellerWalletAccount(r.Context(), req.ReferenceID, req.Name)
	} else {
		jsonError(w, "نوع المرجع غير مدعوم", http.StatusBadRequest)
		return
	}
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, account)
}

// ─── Register All Routes ──────────────────────────────────────────────────────
func RegisterAllRoutes(mux *http.ServeMux, h *FinanceHandlerV2) {
	// Op 1: شحن رصيد
	mux.HandleFunc("/api/finance/charge", h.ChargeWalletHandler)
	// Op 2: سند صرف
	mux.HandleFunc("/api/finance/debit-cashbox", h.DebitCashboxHandler)
	// Op 3 & 3b: إنشاء/قائمة الصناديق (GET/POST)
	mux.HandleFunc("/api/finance/cashboxes", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.ListCashboxesHandler(w, r)
		} else if r.Method == http.MethodPost {
			h.CreateCashboxHandler(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	// Op 4: رصيد المحفظة
	mux.HandleFunc("/api/finance/balance", h.GetBalanceHandler)
	// Op 5: كشف حساب
	mux.HandleFunc("/api/finance/statement", h.CashboxStatementHandler)
	// Op 6: عكس سند
	mux.HandleFunc("/api/finance/reverse", h.ReverseEntryHandler)
	// Op 7: شراء باقة
	mux.HandleFunc("/api/finance/purchase", h.PurchasePackageHandler)
	// Op 8: قائمة المعاملات
	mux.HandleFunc("/api/finance/transactions", h.ListTransactionsHandler)
	// Op 9: قائمة الفواتير
	mux.HandleFunc("/api/finance/invoices", h.ListInvoicesHandler)
	// Op 10: طباعة فاتورة
	mux.HandleFunc("/api/finance/invoices/print", h.InvoiceHTMLHandler)
	// Op 11: إنشاء محفظة
	mux.HandleFunc("/api/finance/wallets", h.CreateWalletHandler)
}

// ─── Helpers ──────────────────────────────────────────────────────────────────
func jsonOK(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
