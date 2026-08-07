package accounting

import (
	"context"
	"fmt"

	"github.com/al-hudhud/backend/internal/balance"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// creditTenantWallet credits a tenant's wallet when a receipt voucher is posted
// (OperationTypeCharge). The balance manager runs its own transaction, so this
// is a compensating operation rather than a 2PC; on failure the caller's voucher
// transaction is rolled back.
func (m *Manager) creditTenantWallet(ctx context.Context, tenantID uuid.UUID, amount decimal.Decimal, reason string, createdBy *uuid.UUID) error {
	if _, err := m.balance.CreditBalance(ctx, balance.BalanceOperation{
		TenantID:  tenantID,
		Amount:    amount,
		Type:      balance.OperationTypeCharge,
		Reason:    reason,
		CreatedBy: walletActor(createdBy),
	}); err != nil {
		return fmt.Errorf("failed to credit tenant wallet: %w", err)
	}
	return nil
}

// debitTenantWallet deducts a tenant's wallet when a payment voucher is posted
// (OperationTypeDeduct). It fails with ErrInsufficientWalletBalance when the
// tenant does not have enough balance.
func (m *Manager) debitTenantWallet(ctx context.Context, tenantID uuid.UUID, amount decimal.Decimal, reason string, createdBy *uuid.UUID) error {
	sufficient, err := m.balance.CheckSufficient(ctx, tenantID, amount)
	if err != nil {
		return fmt.Errorf("failed to check tenant wallet balance: %w", err)
	}
	if !sufficient {
		return ErrInsufficientWalletBalance
	}
	if _, err := m.balance.DeductBalance(ctx, balance.BalanceOperation{
		TenantID:  tenantID,
		Amount:    amount,
		Type:      balance.OperationTypeDeduct,
		Reason:    reason,
		CreatedBy: walletActor(createdBy),
	}); err != nil {
		return fmt.Errorf("failed to debit tenant wallet: %w", err)
	}
	return nil
}

// walletReason builds the balance history reason for a voucher posting.
func walletReason(voucherType, voucherNumber string) string {
	if voucherType == VoucherTypePayment {
		return "سند صرف " + voucherNumber
	}
	return "سند قبض " + voucherNumber
}

// walletReversalReason builds the balance history reason for a voucher reversal.
func walletReversalReason(voucherType, voucherNumber string) string {
	if voucherType == VoucherTypePayment {
		return "عكس سند صرف " + voucherNumber
	}
	return "عكس سند قبض " + voucherNumber
}

// walletActor returns a stable actor label for balance history entries.
func walletActor(createdBy *uuid.UUID) string {
	if createdBy == nil {
		return "accounting"
	}
	return createdBy.String()
}
