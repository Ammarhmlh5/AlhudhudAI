package finance

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// FinanceBridge is a singleton that wraps FinanceServiceClient.
// It is initialized once at application startup and used throughout the backend.
type FinanceBridge struct {
	client *FinanceServiceClient
}

var (
	instance *FinanceBridge
	once     sync.Once
)

// Init initialises the finance bridge. Call once from main.go.
func Init(financeServiceURL string, mainDB *sql.DB) {
	once.Do(func() {
		instance = &FinanceBridge{
			client: NewFinanceServiceClient(financeServiceURL, mainDB),
		}
		log.Println("[Finance] Bridge initialised → " + financeServiceURL)
	})
}

// Bridge returns the singleton instance or nil if Init was not called.
// This allows tests and lightweight backend workflows to operate without
// a configured finance microservice.
func Bridge() *FinanceBridge {
	return instance
}

// ─── Hook: OnTenantRegistered ─────────────────────────────────────────────────
// Call this immediately after a tenant is persisted in the main DB.
// It creates the matching liability account in the Finance Microservice.
func (b *FinanceBridge) OnTenantRegistered(tenantID uuid.UUID, name string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := b.client.OnTenantCreated(ctx, tenantID, name, "TENANT")
	if err != nil {
		// Non-blocking: log the error but don't fail the registration.
		// A reconciliation job can repair missing accounts later.
		log.Printf("[Finance] WARNING: Failed to create wallet for tenant %s: %v", tenantID, err)
	} else {
		log.Printf("[Finance] Wallet account created for tenant %s (%s)", tenantID, name)
	}
}

// ─── Hook: OnResellerRegistered ───────────────────────────────────────────────
func (b *FinanceBridge) OnResellerRegistered(resellerID uuid.UUID, name string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := b.client.OnTenantCreated(ctx, resellerID, name, "RESELLER")
	if err != nil {
		log.Printf("[Finance] WARNING: Failed to create wallet for reseller %s: %v", resellerID, err)
	} else {
		log.Printf("[Finance] Wallet account created for reseller %s (%s)", resellerID, name)
	}
}

// ─── Hook: OnPackagePurchased ─────────────────────────────────────────────────
// Called after subscription checkout is confirmed.
// Deducts the financial amount and credits the SMS quota in the main DB.
func (b *FinanceBridge) OnPackagePurchased(tenantID, walletAccountID uuid.UUID, amount float64, smsCount int64, packageName, customerName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	return b.client.PurchasePackage(ctx, tenantID, walletAccountID, amount, smsCount, packageName, customerName)
}

// GetWalletAccountID resolves a backend tenant or reseller reference to the finance wallet account ID.
func (b *FinanceBridge) GetWalletAccountID(ctx context.Context, refType string, refID uuid.UUID) (uuid.UUID, error) {
	return b.client.GetWalletAccountID(ctx, refType, refID)
}

func (b *FinanceBridge) GetOrCreateWalletAccountID(ctx context.Context, refType string, refID uuid.UUID) (uuid.UUID, error) {
	return b.client.GetOrCreateWalletAccountID(ctx, refType, refID)
}

// ─── Hook: OnSMSSent ─────────────────────────────────────────────────────────
// Called on every outbound SMS. Deducts from the fast SMS quota table only.
// Returns false if the tenant has insufficient quota.
func (b *FinanceBridge) OnSMSSent(tenantID uuid.UUID, count int) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	ok, err := b.client.DeductSMSQuota(ctx, tenantID, count)
	if err != nil {
		return false, fmt.Errorf("SMS quota deduction failed: %w", err)
	}
	return ok, nil
}

// ─── Hook: MonthlyReset ───────────────────────────────────────────────────────
// Called by a Cron Job at month end to zero out expired quotas.
func (b *FinanceBridge) MonthlyReset() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	affected, err := b.client.MonthlyReset(ctx)
	if err != nil {
		log.Printf("[Finance] Monthly reset ERROR: %v", err)
		return
	}
	log.Printf("[Finance] Monthly reset complete — %d quota records cleared", affected)
}
