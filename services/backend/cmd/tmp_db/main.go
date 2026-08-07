package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/queue"
	_ "github.com/lib/pq"
)

var extraMigrations = []string{
	"002_balance_history_table.sql",
	"004_dead_letter_queue_table.sql",
	"007_audit_trail_table.sql",
	"056_recreate_dedicated_gateways_uuid.sql",
	"032_company_ratings_table.sql",
}

// Tables used by tests that come from migrations with invalid SQL or RLS
// policies that would block raw-SQL test inserts. Cleaned up DDL here.
var inlineDDL = []string{
	`CREATE TABLE IF NOT EXISTS services (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    code VARCHAR(50) NOT NULL,
    type VARCHAR(50) NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_services_tenant_id ON services(tenant_id);
CREATE INDEX IF NOT EXISTS idx_services_tenant_type ON services(tenant_id, type);
CREATE INDEX IF NOT EXISTS idx_services_tenant_active ON services(tenant_id, is_active);
CREATE INDEX IF NOT EXISTS idx_services_tenant_code ON services(tenant_id, code);
CREATE UNIQUE INDEX IF NOT EXISTS idx_services_tenant_code_unique ON services(tenant_id, code);`,
	`CREATE TABLE IF NOT EXISTS packages (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL,
    service_id INTEGER NOT NULL,
    name VARCHAR(255) NOT NULL,
    code VARCHAR(50) NOT NULL,
    price_usd DECIMAL(10, 2) NOT NULL,
    included_messages INTEGER,
    overage_rate DECIMAL(10, 4),
    is_unlimited BOOLEAN DEFAULT false,
    billing_cycle VARCHAR(20) DEFAULT 'monthly',
    rollover_enabled BOOLEAN DEFAULT false,
    rollover_limit INTEGER,
    rollover_expiry_months INTEGER,
    is_active BOOLEAN DEFAULT true,
    display_order INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT check_price_positive CHECK (price_usd > 0),
    CONSTRAINT check_messages_positive CHECK (included_messages IS NULL OR included_messages > 0),
    CONSTRAINT check_billing_cycle CHECK (billing_cycle IN ('monthly', 'yearly'))
);
CREATE INDEX IF NOT EXISTS idx_packages_tenant_id ON packages(tenant_id);
CREATE INDEX IF NOT EXISTS idx_packages_tenant_service ON packages(tenant_id, service_id);
CREATE INDEX IF NOT EXISTS idx_packages_tenant_active ON packages(tenant_id, is_active);
CREATE INDEX IF NOT EXISTS idx_packages_tenant_order ON packages(tenant_id, display_order);
CREATE UNIQUE INDEX IF NOT EXISTS idx_packages_tenant_code_unique ON packages(tenant_id, code);`,
	`CREATE TABLE IF NOT EXISTS subscriptions (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL,
    package_id INTEGER NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    messages_used INTEGER DEFAULT 0,
    messages_remaining INTEGER,
    overage_messages INTEGER DEFAULT 0,
    overage_cost DECIMAL(10, 2) DEFAULT 0.00,
    status VARCHAR(20) DEFAULT 'active',
    auto_renew BOOLEAN DEFAULT true,
    grace_period_days INTEGER DEFAULT 3,
    last_renewed_at TIMESTAMP,
    next_renewal_date DATE,
    renewal_attempts INTEGER DEFAULT 0,
    last_renewal_attempt TIMESTAMP,
    idempotency_key VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT check_messages_used_non_negative CHECK (messages_used >= 0),
    CONSTRAINT check_overage_non_negative CHECK (overage_messages >= 0),
    CONSTRAINT check_overage_cost_non_negative CHECK (overage_cost >= 0),
    CONSTRAINT check_grace_period_positive CHECK (grace_period_days > 0),
    CONSTRAINT check_status_valid CHECK (status IN ('active', 'grace_period', 'suspended', 'cancelled', 'expired')),
    CONSTRAINT check_dates_valid CHECK (end_date >= start_date)
);
CREATE INDEX IF NOT EXISTS idx_subscriptions_tenant ON subscriptions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_tenant_package ON subscriptions(tenant_id, package_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_tenant_status ON subscriptions(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_subscriptions_tenant_renewal ON subscriptions(tenant_id, next_renewal_date);
CREATE INDEX IF NOT EXISTS idx_subscriptions_tenant_dates ON subscriptions(tenant_id, start_date, end_date);`,
	`CREATE TABLE IF NOT EXISTS rollover_messages (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL,
    subscription_id INTEGER NOT NULL,
    messages_count INTEGER NOT NULL,
    source_month DATE NOT NULL,
    expiry_date DATE,
    messages_used INTEGER DEFAULT 0,
    messages_remaining INTEGER NOT NULL,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT check_rollover_positive CHECK (messages_count > 0),
    CONSTRAINT check_rollover_remaining CHECK (messages_remaining >= 0),
    CONSTRAINT check_rollover_used_valid CHECK (messages_used >= 0 AND messages_used <= messages_count),
    CONSTRAINT check_rollover_status_valid CHECK (status IN ('active', 'expired', 'consumed'))
);
CREATE INDEX IF NOT EXISTS idx_rollover_tenant ON rollover_messages(tenant_id);
CREATE INDEX IF NOT EXISTS idx_rollover_tenant_subscription ON rollover_messages(tenant_id, subscription_id);
CREATE INDEX IF NOT EXISTS idx_rollover_tenant_status ON rollover_messages(tenant_id, status);`,
	`CREATE TABLE IF NOT EXISTS billing_invoices (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL,
    invoice_number VARCHAR(50) NOT NULL,
    subscription_id INTEGER,
    transaction_id INTEGER,
    subtotal DECIMAL(10, 2) NOT NULL,
    tax_rate DECIMAL(5, 2) DEFAULT 0.00,
    tax_amount DECIMAL(10, 2) DEFAULT 0.00,
    total_amount DECIMAL(10, 2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',
    description TEXT,
    service_name VARCHAR(255),
    issue_date DATE NOT NULL DEFAULT CURRENT_DATE,
    due_date DATE,
    paid_date DATE,
    status VARCHAR(20) DEFAULT 'pending',
    pdf_generated BOOLEAN DEFAULT false,
    pdf_path VARCHAR(500),
    email_sent BOOLEAN DEFAULT false,
    email_sent_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT check_invoice_subtotal_positive CHECK (subtotal > 0),
    CONSTRAINT check_invoice_tax_rate_valid CHECK (tax_rate >= 0 AND tax_rate <= 100),
    CONSTRAINT check_invoice_tax_amount_non_negative CHECK (tax_amount >= 0),
    CONSTRAINT check_invoice_total_positive CHECK (total_amount > 0),
    CONSTRAINT check_invoice_status_valid CHECK (status IN ('pending', 'paid', 'failed', 'cancelled')),
    CONSTRAINT check_invoice_dates_valid CHECK (due_date IS NULL OR due_date >= issue_date)
);
CREATE INDEX IF NOT EXISTS idx_invoices_tenant ON billing_invoices(tenant_id);
CREATE INDEX IF NOT EXISTS idx_invoices_tenant_status ON billing_invoices(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_invoices_tenant_subscription ON billing_invoices(tenant_id, subscription_id);
CREATE INDEX IF NOT EXISTS idx_invoices_tenant_transaction ON billing_invoices(tenant_id, transaction_id);
CREATE INDEX IF NOT EXISTS idx_invoices_tenant_issue_date ON billing_invoices(tenant_id, issue_date DESC);
CREATE INDEX IF NOT EXISTS idx_invoices_tenant_due_date ON billing_invoices(tenant_id, due_date);
CREATE UNIQUE INDEX IF NOT EXISTS idx_invoices_tenant_number_unique ON billing_invoices(tenant_id, invoice_number);`,
	`CREATE TABLE IF NOT EXISTS promo_codes (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    code VARCHAR(50) NOT NULL,
    description TEXT,
    discount_type VARCHAR(20) NOT NULL CHECK (discount_type IN ('percentage', 'fixed_amount')),
    discount_value DECIMAL(10, 2) NOT NULL CHECK (discount_value > 0),
    max_discount_amount DECIMAL(10, 2),
    min_purchase_amount DECIMAL(10, 2),
    valid_from TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    valid_until TIMESTAMP NOT NULL,
    usage_limit INT,
    usage_count INT NOT NULL DEFAULT 0,
    per_user_limit INT DEFAULT 1,
    applicable_to VARCHAR(20) NOT NULL DEFAULT 'all' CHECK (applicable_to IN ('all', 'packages', 'topup', 'gateway')),
    applicable_package_ids BIGINT[],
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by BIGINT,
    CONSTRAINT unique_promo_code_per_tenant UNIQUE (tenant_id, code),
    CONSTRAINT valid_date_range CHECK (valid_until > valid_from),
    CONSTRAINT valid_usage_limit CHECK (usage_limit IS NULL OR usage_limit > 0),
    CONSTRAINT valid_per_user_limit CHECK (per_user_limit > 0)
);
CREATE INDEX IF NOT EXISTS idx_promo_codes_tenant ON promo_codes(tenant_id);
CREATE INDEX IF NOT EXISTS idx_promo_codes_tenant_code ON promo_codes(tenant_id, code);
CREATE INDEX IF NOT EXISTS idx_promo_codes_tenant_active ON promo_codes(tenant_id, is_active) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_promo_codes_tenant_valid ON promo_codes(tenant_id, valid_from, valid_until) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_promo_codes_applicable_to ON promo_codes(tenant_id, applicable_to) WHERE is_active = TRUE;
CREATE TABLE IF NOT EXISTS promo_code_usage (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    promo_code_id BIGINT NOT NULL REFERENCES promo_codes(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL,
    order_type VARCHAR(20) NOT NULL CHECK (order_type IN ('subscription', 'topup', 'gateway')),
    order_id BIGINT,
    original_amount DECIMAL(10, 2) NOT NULL,
    discount_amount DECIMAL(10, 2) NOT NULL,
    final_amount DECIMAL(10, 2) NOT NULL,
    used_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_promo_usage_per_order UNIQUE (tenant_id, promo_code_id, order_type, order_id),
    CONSTRAINT valid_discount_amount CHECK (discount_amount >= 0 AND discount_amount <= original_amount),
    CONSTRAINT valid_final_amount CHECK (final_amount = original_amount - discount_amount)
);
CREATE INDEX IF NOT EXISTS idx_promo_usage_tenant ON promo_code_usage(tenant_id);
CREATE INDEX IF NOT EXISTS idx_promo_usage_promo_code ON promo_code_usage(tenant_id, promo_code_id);
CREATE INDEX IF NOT EXISTS idx_promo_usage_user ON promo_code_usage(tenant_id, user_id);
CREATE INDEX IF NOT EXISTS idx_promo_usage_order ON promo_code_usage(tenant_id, order_type, order_id);
CREATE INDEX IF NOT EXISTS idx_promo_usage_used_at ON promo_code_usage(tenant_id, used_at);
CREATE OR REPLACE FUNCTION update_promo_codes_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
DROP TRIGGER IF EXISTS promo_codes_updated_at_trigger ON promo_codes;
CREATE TRIGGER promo_codes_updated_at_trigger
    BEFORE UPDATE ON promo_codes
    FOR EACH ROW
    EXECUTE FUNCTION update_promo_codes_updated_at();
CREATE OR REPLACE FUNCTION increment_promo_code_usage()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE promo_codes
    SET usage_count = usage_count + 1
    WHERE id = NEW.promo_code_id AND tenant_id = NEW.tenant_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
DROP TRIGGER IF EXISTS increment_promo_usage_trigger ON promo_code_usage;
CREATE TRIGGER increment_promo_usage_trigger
    AFTER INSERT ON promo_code_usage
    FOR EACH ROW
    EXECUTE FUNCTION increment_promo_code_usage();`,
}

func main() {
	pass := os.Getenv("PGPASSWORD")
	dbname := os.Getenv("DB_NAME")
	if dbname == "" {
		dbname = "sms_gateway_test"
	}

	dsn := fmt.Sprintf("host=localhost user=postgres password=%s dbname=%s sslmode=disable", pass, dbname)
	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		panic(err)
	}

	scaffold := []string{
		"DROP SCHEMA IF EXISTS public CASCADE",
		"CREATE SCHEMA public",
		`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`,
		"CREATE EXTENSION IF NOT EXISTS pgcrypto",
		"CREATE TYPE message_category AS ENUM ('INVOICE','PAYMENT','ALERT','REVERSAL','PROMO','OTP','REMINDER','OTHER')",
	}
	// Force session timezone to UTC for all connections to this database.
	// lib/pq parses timestamp columns using the session TimeZone; on Windows
	// Go cannot load the IANA zone name (e.g. Asia/Riyadh) so it falls back to
	// UTC, skewing comparisons by the session offset. UTC keeps both the
	// lib/pq and pgx paths consistent.
	scaffold = append(scaffold, fmt.Sprintf("ALTER DATABASE %s SET timezone = 'UTC'", dbname))
	for _, s := range scaffold {
		if _, err := conn.Exec(s); err != nil {
			fmt.Println("scaffold err:", err)
		}
	}
	conn.Close()
	fmt.Println("scaffold done:", dbname)

	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", "5432")
	os.Setenv("DB_USER", "postgres")
	os.Setenv("DB_PASSWORD", pass)
	os.Setenv("DB_NAME", dbname)
	os.Setenv("DB_SSLMODE", "disable")

	db.InitGormDB()

	ctx := context.Background()

	// Add legacy messages columns used by production batch/routing code
	legacyCols := []string{
		"ALTER TABLE messages ADD COLUMN IF NOT EXISTS message_id VARCHAR(255)",
		"ALTER TABLE messages ADD COLUMN IF NOT EXISTS message_content TEXT",
		"ALTER TABLE messages ADD COLUMN IF NOT EXISTS priority VARCHAR(20)",
		"ALTER TABLE messages ADD COLUMN IF NOT EXISTS queued_at TIMESTAMP",
		// The status handler writes failed_at/fail_reason on FAILED reports but the
		// GORM model has no such fields, so AutoMigrate never creates them.
		"ALTER TABLE messages ADD COLUMN IF NOT EXISTS failed_at TIMESTAMP",
		"ALTER TABLE messages ADD COLUMN IF NOT EXISTS fail_reason TEXT",
		// The model marks content NOT NULL, but production batch code writes the
		// legacy message_content column only; the later-added content column is
		// nullable in the production (migration-created) schema.
		"ALTER TABLE messages ALTER COLUMN content DROP NOT NULL",
	}
	for _, s := range legacyCols {
		if _, err := db.PgxPool.Exec(ctx, s); err != nil {
			fmt.Println("legacy col err:", err)
		}
	}

	// Apply migration-only tables (not part of GORM models)
	migDir := filepath.Join("migrations")
	for _, file := range extraMigrations {
		data, err := os.ReadFile(filepath.Join(migDir, file))
		if err != nil {
			fmt.Println("read err", file, err)
			continue
		}
		if _, err := db.PgxPool.Exec(ctx, string(data)); err != nil {
			fmt.Println("migration err:", file, err)
		} else {
			fmt.Println("applied:", file)
		}
	}
	for _, ddl := range inlineDDL {
		if _, err := db.PgxPool.Exec(ctx, ddl); err != nil {
			fmt.Println("inline ddl err:", err)
		} else {
			fmt.Println("applied inline ddl")
		}
	}

	// Post-migration adjustments: production balance/accounting code writes
	// gateway_id even though the migrations (and GORM model) predate the column,
	// and migration 002's FK on balance_history.message_id blocks logging for
	// messages not (yet) persisted.
	postMigrations := []string{
		"ALTER TABLE balance_history ADD COLUMN IF NOT EXISTS gateway_id UUID",
		"ALTER TABLE accounting_transactions ADD COLUMN IF NOT EXISTS gateway_id UUID",
		// The refund flow (manager.go) stamps settled_at when a deduction is
		// marked REFUNDED, but the GORM model/migrations predate the column.
		"ALTER TABLE accounting_transactions ADD COLUMN IF NOT EXISTS settled_at TIMESTAMP",
		// CreateSubscription (subscription/manager.go) stamps idempotency_key on
		// every insert; the inline DDL predates the column.
		"ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS idempotency_key VARCHAR(255)",
		"ALTER TABLE balance_history DROP CONSTRAINT IF EXISTS balance_history_message_id_fkey",
		// Production code treats ip_address as a plain string (scanning INET into
		// string fails in binary format), so use TEXT in test env.
		"ALTER TABLE balance_history ALTER COLUMN ip_address TYPE TEXT USING ip_address::text",
		// Same for audit_trail: the audit logger inserts a plain string (and the
		// status handler passes ""), which INET rejects with 22P02.
		"ALTER TABLE audit_trail ALTER COLUMN ip_address TYPE TEXT USING ip_address::text",
		// Audit tests log entries for throwaway user/tenant UUIDs, so the FKs
		// added in migration 007 must not block inserts in test env.
		"ALTER TABLE audit_trail DROP CONSTRAINT IF EXISTS audit_trail_user_id_fkey",
		"ALTER TABLE audit_trail DROP CONSTRAINT IF EXISTS audit_trail_tenant_id_fkey",
	}
	for _, s := range postMigrations {
		if _, err := db.PgxPool.Exec(ctx, s); err != nil {
			fmt.Println("post-migration err:", err)
		}
	}

	// lib/pq writes timestamp (without time zone) columns using Go's local
	// wall-clock and reads them back in the session timezone, which skews
	// time comparisons when the two differ (e.g. Windows + Asia/Riyadh).
	// timestamptz stores absolute instants regardless of session timezone, so
	// convert every timestamp column to timestamptz in the test env.
	if _, err := db.PgxPool.Exec(ctx, `
		DO $$
		DECLARE
			r record;
		BEGIN
			FOR r IN
				SELECT table_name, column_name
				FROM information_schema.columns
				WHERE table_schema = 'public'
				  AND data_type = 'timestamp without time zone'
			LOOP
				EXECUTE format('ALTER TABLE public.%I ALTER COLUMN %I TYPE timestamptz USING %I::timestamptz',
					r.table_name, r.column_name, r.column_name);
			END LOOP;
		END $$;`); err != nil {
		fmt.Println("timestamp conversion err:", err)
	} else {
		fmt.Println("timestamp columns converted to timestamptz")
	}

	// Apply River schema (river_job, river_queue, river_leader)
	if err := queue.MigrateSchema(ctx, "public"); err != nil {
		fmt.Println("river migrate err:", err)
	} else {
		fmt.Println("river schema done")
	}

	// Seed fixed tenants used by batch persistence tests
	seedTenants(ctx)

	fmt.Println("AutoMigrate done:", dbname)
	db.Close()
}

func seedTenants(ctx context.Context) {
	seeds := []struct {
		userID, tenantID, serial, suffix string
	}{
		{"00000000-0000-0000-0000-0000000000a1", "00000000-0000-0000-0000-000000000001", "SERIAL-001", "001"},
		{"00000000-0000-0000-0000-0000000000a2", "00000000-0000-0000-0000-000000000002", "SERIAL-002", "002"},
	}
	for _, s := range seeds {
		_, err := db.PgxPool.Exec(ctx, `
			INSERT INTO users (id, email, password_hash, user_type, is_active, is_verified, created_at, updated_at)
			VALUES ($1, $2, 'x', 'ADMIN', true, true, now(), now())
			ON CONFLICT (id) DO NOTHING`,
			s.userID, "tenant"+s.suffix+"@test.local")
		if err != nil {
			fmt.Println("seed user err:", err)
		}

		_, err = db.PgxPool.Exec(ctx, `
			INSERT INTO tenants (id, user_id, business_name, contact_name, email, company_serial,
				api_key_accounting, api_key_phone, api_key_web, balance, status, is_active, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 1000, 'ACTIVE', true, now(), now())
			ON CONFLICT (id) DO NOTHING`,
			s.tenantID, s.userID, "Test Tenant "+s.suffix, "Contact "+s.suffix,
			"tenant"+s.suffix+"@test.local", s.serial,
			"key-accounting-"+s.suffix, "key-phone-"+s.suffix, "key-web-"+s.suffix)
		if err != nil {
			fmt.Println("seed tenant err:", err)
		}
	}
}
