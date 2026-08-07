package finance

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// FinanceServiceClient is a thin client used by the main backend to communicate
// with the isolated Finance Microservice.
type FinanceServiceClient struct {
	baseURL    string
	httpClient *http.Client
	db         *sql.DB // main backend DB (for tenant_sms_quotas)
}

func NewFinanceServiceClient(financeServiceURL string, db *sql.DB) *FinanceServiceClient {
	return &FinanceServiceClient{
		baseURL:    financeServiceURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		db:         db,
	}
}

type walletResponse struct {
	ID uuid.UUID `json:"id"`
}

// ─── Op 11: OnTenantCreated ────────────────────────────────────────────────────
// Called automatically by the main backend when a new tenant/reseller registers.
func (c *FinanceServiceClient) OnTenantCreated(ctx context.Context, tenantID uuid.UUID, name string, refType string) (uuid.UUID, error) {
        payload := map[string]any{
                "reference_id":   tenantID.String(),
                "reference_type": refType, // "TENANT" or "RESELLER"
                "name":           name,
        }

        var resp walletResponse
        if err := c.post(ctx, "/api/finance/wallets", payload, &resp); err != nil {
                return uuid.Nil, err
        }

        if err := c.upsertWalletMapping(ctx, refType, tenantID, resp.ID); err != nil {
                return uuid.Nil, err
        }
        return resp.ID, nil
}

func (c *FinanceServiceClient) upsertWalletMapping(ctx context.Context, refType string, referenceID, walletAccountID uuid.UUID) error {
        _, err := c.db.ExecContext(ctx, `
                INSERT INTO finance_wallet_accounts (reference_type, reference_id, wallet_account_id, created_at, updated_at)
                VALUES ($1, $2, $3, NOW(), NOW())
                ON CONFLICT (reference_type, reference_id) DO UPDATE
                  SET wallet_account_id = EXCLUDED.wallet_account_id,
                      updated_at = NOW()
        `, refType, referenceID, walletAccountID)
        if err != nil {
                return fmt.Errorf("failed to persist finance wallet mapping: %w", err)
        }
        return nil
}

func (c *FinanceServiceClient) queryWalletAccountID(ctx context.Context, refType string, referenceID uuid.UUID) (uuid.UUID, error) {
        var walletID uuid.UUID
        row := c.db.QueryRowContext(ctx, `
                SELECT wallet_account_id
                FROM finance_wallet_accounts
                WHERE reference_type = $1 AND reference_id = $2
        `, refType, referenceID)
        if err := row.Scan(&walletID); err != nil {
                return uuid.Nil, err
        }
        return walletID, nil
}

func (c *FinanceServiceClient) GetWalletAccountID(ctx context.Context, refType string, referenceID uuid.UUID) (uuid.UUID, error) {
        walletID, err := c.queryWalletAccountID(ctx, refType, referenceID)
        if err != nil {
                if err == sql.ErrNoRows {
                        return uuid.Nil, fmt.Errorf("finance wallet mapping not found for %s %s", refType, referenceID)
                }
                return uuid.Nil, err
        }
        return walletID, nil
}

func (c *FinanceServiceClient) GetOrCreateWalletAccountID(ctx context.Context, refType string, referenceID uuid.UUID) (uuid.UUID, error) {
        walletID, err := c.queryWalletAccountID(ctx, refType, referenceID)
        if err == nil {
                return walletID, nil
        }
        if err != nil && err != sql.ErrNoRows {
                return uuid.Nil, err
        }

        name, err := c.resolveReferenceName(ctx, refType, referenceID)
        if err != nil {
                return uuid.Nil, fmt.Errorf("failed to rebuild finance wallet mapping for %s %s: %w", refType, referenceID, err)
        }

        return c.OnTenantCreated(ctx, referenceID, name, refType)
}

func (c *FinanceServiceClient) resolveReferenceName(ctx context.Context, refType string, referenceID uuid.UUID) (string, error) {
        var name string
        err := c.db.QueryRowContext(ctx, `
                SELECT business_name
                FROM tenants
                WHERE id = $1
                  AND deleted_at IS NULL
        `, referenceID).Scan(&name)
        if err != nil {
                if err == sql.ErrNoRows {
                        return "", fmt.Errorf("reference %s %s does not exist", refType, referenceID)
                }
                return "", err
        }
        return name, nil
}

// ─── Op 7: PurchasePackage ─────────────────────────────────────────────────────
// Called when a tenant buys a package: first charges the financial system,
// then adds the SMS quota to the fast table in the main DB.
func (c *FinanceServiceClient) PurchasePackage(ctx context.Context, tenantID, walletAccountID uuid.UUID, amount float64, smsCount int64, packageName, customerName string) error {
	// Step 1: Deduct from financial wallet
	payload := map[string]any{
		"wallet_account_id": walletAccountID.String(),
		"amount":            amount,
		"package_name":      packageName,
		"customer_name":     customerName,
	}
	if err := c.post(ctx, "/api/finance/purchase", payload, nil); err != nil {
		return fmt.Errorf("فشل الاستقطاع المالي للباقة: %v", err)
	}

	// Step 2: Add SMS quota to the fast table in main DB for this tenant.
	if smsCount > 0 {
		_, err := c.db.ExecContext(ctx, `
		INSERT INTO tenant_sms_quotas (tenant_id, available_sms, quota_expires_at, last_reset_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (tenant_id) DO UPDATE
		  SET available_sms = tenant_sms_quotas.available_sms + $2,
		      quota_expires_at = EXCLUDED.quota_expires_at,
		      updated_at = NOW()
		`, tenantID, smsCount, endOfMonth())
		if err != nil {
			return fmt.Errorf("تم الاستقطاع المالي لكن فشل تحديث رصيد الرسائل: %v", err)
		}
	}

	return nil
}

// ─── Op 13: DeductSMSQuota ────────────────────────────────────────────────────
// Called on every message send. Fast path — no financial system involved.
func (c *FinanceServiceClient) DeductSMSQuota(ctx context.Context, tenantID uuid.UUID, smsCount int) (bool, error) {
	row := c.db.QueryRowContext(ctx, `
		UPDATE tenant_sms_quotas
		SET available_sms = available_sms - $1,
		    total_consumed_this_month = total_consumed_this_month + $1,
		    updated_at = NOW()
		WHERE tenant_id = $2 AND available_sms >= $1
		RETURNING available_sms
	`, smsCount, tenantID)

	var remaining int64
	if err := row.Scan(&remaining); err == sql.ErrNoRows {
		return false, fmt.Errorf("رصيد الرسائل غير كافٍ أو لا توجد باقة نشطة")
	} else if err != nil {
		return false, err
	}
	return true, nil
}

// ─── Op 12: MonthlyReset ──────────────────────────────────────────────────────
// Called by a monthly Cron Job. Resets SMS quotas for all tenants.
func (c *FinanceServiceClient) MonthlyReset(ctx context.Context) (int64, error) {
	result, err := c.db.ExecContext(ctx, `
		UPDATE tenant_sms_quotas
		SET available_sms = 0,
		    total_consumed_this_month = 0,
		    last_reset_at = NOW(),
		    updated_at = NOW()
		WHERE quota_expires_at <= NOW()
	`)
	if err != nil {
		return 0, fmt.Errorf("فشل التصفير الشهري: %v", err)
	}
	affected, _ := result.RowsAffected()
	return affected, nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────
func (c *FinanceServiceClient) post(ctx context.Context, path string, payload any, result any) error {
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Service", "backend")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		var errBody map[string]string
		json.NewDecoder(resp.Body).Decode(&errBody)
		return fmt.Errorf("Finance API error %d: %s", resp.StatusCode, errBody["error"])
	}
	if result != nil {
		json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

func endOfMonth() time.Time {
	now := time.Now()
	firstOfNext := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())
	return firstOfNext.Add(-time.Second)
}
