package api

import (
	"context"
	"encoding/json"
	"net/http"

	"finance/internal/models"
	"finance/internal/services"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type FinanceHandler struct {
	db          *gorm.DB
	accService  *services.AccountingService
	accManager  *services.AccountManager
}

func NewFinanceHandler(db *gorm.DB, accService *services.AccountingService, accManager *services.AccountManager) *FinanceHandler {
	return &FinanceHandler{
		db:          db,
		accService:  accService,
		accManager:  accManager,
	}
}

// CreateWalletReq payload
type CreateWalletReq struct {
	ReferenceID   uuid.UUID `json:"reference_id"`
	ReferenceType string    `json:"reference_type"` // TENANT or RESELLER
	Name          string    `json:"name"`
}

// CreateWalletHandler handles POST /api/wallets
func (h *FinanceHandler) CreateWalletHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateWalletReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "بيانات غير صالحة", http.StatusBadRequest)
		return
	}

	var account *models.Account
	var err error

	if req.ReferenceType == "TENANT" {
		account, err = h.accManager.CreateUserWalletAccount(r.Context(), req.ReferenceID, req.Name)
	} else if req.ReferenceType == "RESELLER" {
		account, err = h.accManager.CreateResellerWalletAccount(r.Context(), req.ReferenceID, req.Name)
	} else {
		http.Error(w, "نوع المرجع غير مدعوم", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(account)
}

// TransactionReq payload for charging or deducting
type TransactionReq struct {
	WalletAccountID uuid.UUID       `json:"wallet_account_id"`
	Amount          decimal.Decimal `json:"amount"`
	Description     string          `json:"description"`
}

// ChargeWalletHandler handles POST /api/charge
// Debit: System Cash (1100), Credit: User Wallet (2100-XXX)
func (h *FinanceHandler) ChargeWalletHandler(w http.ResponseWriter, r *http.Request) {
	var req TransactionReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "بيانات غير صالحة", http.StatusBadRequest)
		return
	}

	// Retrieve System Cash Account (1100)
	var cashAccount models.Account
	if err := h.db.WithContext(r.Context()).Where("code = ?", "1100").First(&cashAccount).Error; err != nil {
		http.Error(w, "حساب الصندوق الأساسي غير موجود", http.StatusInternalServerError)
		return
	}

	// Create Double Entry
	lines := []models.JournalEntryLine{
		{
			AccountID:   cashAccount.ID,
			Debit:       req.Amount,
			Credit:      decimal.Zero,
			Description: "إيداع نقدي في الصندوق",
		},
		{
			AccountID:   req.WalletAccountID,
			Debit:       decimal.Zero,
			Credit:      req.Amount,
			Description: "شحن رصيد المحفظة " + req.Description,
		},
	}

	entry, err := h.accService.CreateJournalEntry(r.Context(), "عملية شحن رصيد", lines, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entry)
}

// PurchasePackageHandler handles POST /api/purchase_package (e.g. buying SMS quotas/subscriptions)
// Debit: User Wallet, Credit: Revenue (4100)
func (h *FinanceHandler) PurchasePackageHandler(w http.ResponseWriter, r *http.Request) {
	var req TransactionReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "بيانات غير صالحة", http.StatusBadRequest)
		return
	}

	// Check balance first to ensure no overdraft (unless credit limit allows it, simple check here)
	balance, err := h.accService.GetAccountBalance(r.Context(), req.WalletAccountID)
	if err != nil {
		http.Error(w, "الحساب غير موجود", http.StatusNotFound)
		return
	}
	if balance.LessThan(req.Amount) {
		http.Error(w, "رصيد المحفظة غير كافٍ لشراء الباقة", http.StatusPaymentRequired)
		return
	}

	// Retrieve Revenue Account (4100)
	var revAccount models.Account
	if err := h.db.WithContext(r.Context()).Where("code = ?", "4100").First(&revAccount).Error; err != nil {
		http.Error(w, "حساب الإيرادات الأساسي غير موجود", http.StatusInternalServerError)
		return
	}

	lines := []models.JournalEntryLine{
		{
			AccountID:   req.WalletAccountID,
			Debit:       req.Amount,
			Credit:      decimal.Zero,
			Description: "استقطاع قيمة شراء باقة/اشتراك: " + req.Description,
		},
		{
			AccountID:   revAccount.ID,
			Debit:       decimal.Zero,
			Credit:      req.Amount,
			Description: "إثبات إيراد بيع باقات",
		},
	}

	entry, err := h.accService.CreateJournalEntry(r.Context(), "شراء باقة رسائل", lines, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entry)
}

// GetBalanceHandler GET /api/wallets/{id}/balance
func (h *FinanceHandler) GetBalanceHandler(w http.ResponseWriter, r *http.Request) {
	// In a real router, you'd extract {id} from URL parameters.
	// For simplicity, reading from query string ?id=UUID
	idParam := r.URL.Query().Get("id")
	accID, err := uuid.Parse(idParam)
	if err != nil {
		http.Error(w, "معرف المحفظة غير صالح", http.StatusBadRequest)
		return
	}

	balance, err := h.accService.GetAccountBalance(context.Background(), accID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	resp := map[string]string{"balance": balance.String()}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// RegisterRoutes registers all finance HTTP routes
func RegisterRoutes(mux *http.ServeMux, handler *FinanceHandler) {
	mux.HandleFunc("/api/finance/wallets", handler.CreateWalletHandler) // POST
	mux.HandleFunc("/api/finance/charge", handler.ChargeWalletHandler)  // POST
	mux.HandleFunc("/api/finance/purchase", handler.PurchasePackageHandler) // POST
	mux.HandleFunc("/api/finance/balance", handler.GetBalanceHandler)   // GET
}
