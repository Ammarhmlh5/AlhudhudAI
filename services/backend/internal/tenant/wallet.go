package tenant

import (
	"errors"
	"net/http"

	"github.com/al-hudhud/backend/internal/balance"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// AdminAddBalanceRequest defines the input for admin wallet funding.
type AdminAddBalanceRequest struct {
	Amount decimal.Decimal `json:"amount" binding:"required"`
	Reason string          `json:"reason"`
}

// AdminAddBalance funds a tenant's wallet (OperationTypeCharge).
// POST /api/v1/admin/tenants/:id/balance
func (h *Handler) AdminAddBalance(c *gin.Context) {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant id"})
		return
	}

	var req AdminAddBalanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Amount.LessThanOrEqual(decimal.Zero) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "amount must be positive"})
		return
	}

	createdBy := c.GetString("email")
	if createdBy == "" {
		createdBy = c.GetString("user_id")
	}

	result, err := h.balance.CreditBalance(c.Request.Context(), balance.BalanceOperation{
		TenantID:  tenantID,
		Amount:    req.Amount,
		Reason:    req.Reason,
		CreatedBy: createdBy,
		IPAddress: c.ClientIP(),
	})
	if err != nil {
		var balErr *balance.BalanceError
		if errors.As(err, &balErr) && balErr.Type == balance.ErrorTenantNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "tenant not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "balance added successfully",
		"tenant_id":   tenantID,
		"amount":      req.Amount,
		"old_balance": result.OldBalance,
		"new_balance": result.NewBalance,
	})
}
