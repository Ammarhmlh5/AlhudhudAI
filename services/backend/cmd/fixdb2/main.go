package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
)

func main() {
	// Try loading from backend dir first, then parent dirs
	_ = godotenv.Load(".env")
	// Note: cmd path is backend/cmd/fixdb2 so parent is backend root

	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "hudhud_admin")
	password := getEnv("DB_PASSWORD", "hudhud_pass")
	dbname := getEnv("DB_NAME", "al_hudhud")
	sslmode := getEnv("DB_SSLMODE", "disable")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)

	log.Printf("Connecting: host=%s port=%s user=%s dbname=%s", host, port, user, dbname)

	// ── Standard DB ─────────────────────────────────────────────────────────────
	dbConn, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	defer dbConn.Close()

	if err = dbConn.Ping(); err != nil {
		log.Fatalf("DB connection failed: %v", err)
	}
	log.Println("✓ Connected to PostgreSQL")

	// ── Step 1: Create admin_statistics table + refresh function ────────────────
	log.Println("\n── Step 1: admin_statistics + refresh_admin_statistics() ──")
	applyAdminStatsMigration(dbConn)

	// ── Step 2: Get all tenants from the 'tenants' table ────────────────────────
	tenants := getAllTenants(dbConn)
	log.Printf("✓ Found %d tenants in 'tenants' table", len(tenants))

	if len(tenants) == 0 {
		log.Println("No tenants found - done.")
		return
	}

	// ── Step 3: Apply River migrations to each tenant schema ────────────────────
	log.Println("\n── Step 2: Ensure schemas and River migrations for each tenant ──")
	ctx := context.Background()
	pgxPool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		log.Fatalf("Failed to create pgxpool: %v", err)
	}
	defer pgxPool.Close()

	successCount := 0
	skipCount := 0
	failCount := 0

	for _, t := range tenants {
		schemaName := getSchemaName(t.ID)
		
		// 1. Ensure schema exists
		_, err := dbConn.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", schemaName))
		if err != nil {
			log.Printf("  ✗ %s: Failed to create schema: %v", schemaName, err)
			failCount++
			continue
		}

		// 2. Check if river_job already exists (basic check for River)
		var exists bool
		// Use parameterized query to prevent SQL injection
		query := "SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = $1 AND table_name = 'river_job')"
		err = dbConn.QueryRow(query, schemaName).Scan(&exists)
		if err != nil {
			log.Printf("  ⚠ Error checking %s: %v", schemaName, err)
			failCount++
			continue
		}

		if exists {
			log.Printf("  - %s: River tables already exist. Skipping.", schemaName)
			skipCount++
			continue
		}

		log.Printf("  → %s: Migrating River...", schemaName)
		if err := migrateRiverSchema(ctx, pgxPool, schemaName); err != nil {
			if strings.Contains(err.Error(), "already exists") {
				log.Printf("  ⚠ %s: Partial migration detected, but continuing.", schemaName)
				successCount++
			} else {
				log.Printf("  ✗ %s FAILED: %v", schemaName, err)
				failCount++
			}
		} else {
			log.Printf("  ✓ %s SUCCESS", schemaName)
			successCount++
		}
	}

	log.Printf("\nSummary: %d Migrated, %d Skipped, %d Failed", successCount, skipCount, failCount)
	log.Println("✓ All done! Restart the backend.")
}

type TenantInfo struct {
	ID   string
	Name string
}

func getAllTenants(db *sql.DB) []TenantInfo {
	rows, err := db.Query("SELECT id, business_name FROM tenants WHERE deleted_at IS NULL")
	if err != nil {
		log.Printf("Failed to query tenants: %v", err)
		return nil
	}
	defer rows.Close()

	var tenants []TenantInfo
	for rows.Next() {
		var t TenantInfo
		if err := rows.Scan(&t.ID, &t.Name); err == nil {
			tenants = append(tenants, t)
		}
	}
	return tenants
}

func getSchemaName(tenantID string) string {
	safeID := strings.ReplaceAll(tenantID, "-", "_")
	return fmt.Sprintf("tenant_%s", safeID)
}

func applyAdminStatsMigration(db *sql.DB) {
	// Create table
	createTable := `
CREATE TABLE IF NOT EXISTS admin_statistics (
    id                      INT PRIMARY KEY DEFAULT 1,
    total_tenants           INT NOT NULL DEFAULT 0,
    active_tenants          INT NOT NULL DEFAULT 0,
    total_devices           INT NOT NULL DEFAULT 0,
    active_devices          INT NOT NULL DEFAULT 0,
    pending_messages        INT NOT NULL DEFAULT 0,
    queued_messages         INT NOT NULL DEFAULT 0,
    processing_messages     INT NOT NULL DEFAULT 0,
    total_messages_today    INT NOT NULL DEFAULT 0,
    sent_messages_today     INT NOT NULL DEFAULT 0,
    failed_messages_today   INT NOT NULL DEFAULT 0,
    total_balance           DECIMAL(14, 4) NOT NULL DEFAULT 0,
    total_events_today      INT NOT NULL DEFAULT 0,
    parsed_events_today     INT NOT NULL DEFAULT 0,
    failed_events_today     INT NOT NULL DEFAULT 0,
    last_updated            TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT single_row CHECK (id = 1)
);`
	if _, err := db.Exec(createTable); err != nil {
		log.Printf("  ⚠ Create table: %v", err)
	} else {
		log.Println("  ✓ admin_statistics table OK")
	}

	if _, err := db.Exec("INSERT INTO admin_statistics (id) VALUES (1) ON CONFLICT (id) DO NOTHING"); err != nil {
		log.Printf("  ⚠ Insert row: %v", err)
	}

	// Check if global_incoming_events exists
	var eventsTableExists bool
	db.QueryRow(`SELECT EXISTS (
		SELECT 1 FROM information_schema.tables 
		WHERE table_schema = 'public' AND table_name = 'global_incoming_events'
	)`).Scan(&eventsTableExists)

	var fn string
	if eventsTableExists {
		fn = `CREATE OR REPLACE FUNCTION refresh_admin_statistics()
RETURNS void AS $$
BEGIN
    UPDATE admin_statistics SET
        total_tenants         = (SELECT COUNT(*) FROM tenants WHERE deleted_at IS NULL),
        active_tenants        = (SELECT COUNT(*) FROM tenants WHERE is_active = true AND deleted_at IS NULL),
        total_devices         = (SELECT COUNT(*) FROM paired_devices),
        active_devices        = (SELECT COUNT(*) FROM paired_devices WHERE status = 'ONLINE'),
        pending_messages      = (SELECT COUNT(*) FROM messages WHERE status = 'PENDING'),
        queued_messages       = (SELECT COUNT(*) FROM messages WHERE status = 'QUEUED'),
        processing_messages   = (SELECT COUNT(*) FROM messages WHERE status = 'PROCESSING'),
        total_messages_today  = (SELECT COUNT(*) FROM messages WHERE created_at >= CURRENT_DATE),
        sent_messages_today   = (SELECT COUNT(*) FROM messages WHERE status IN ('SENT','DELIVERED') AND created_at >= CURRENT_DATE),
        failed_messages_today = (SELECT COUNT(*) FROM messages WHERE status = 'FAILED' AND created_at >= CURRENT_DATE),
        total_balance         = (SELECT COALESCE(SUM(balance), 0) FROM tenants WHERE deleted_at IS NULL),
        total_events_today    = (SELECT COUNT(*) FROM global_incoming_events WHERE created_at >= CURRENT_DATE),
        parsed_events_today   = (SELECT COUNT(*) FROM global_incoming_events WHERE status = 'PARSED' AND created_at >= CURRENT_DATE),
        failed_events_today   = (SELECT COUNT(*) FROM global_incoming_events WHERE status = 'FAILED' AND created_at >= CURRENT_DATE),
        last_updated          = NOW()
    WHERE id = 1;
END;
$$ LANGUAGE plpgsql;`
	} else {
		log.Println("  ⚠ global_incoming_events not found - creating function without event metrics")
		fn = `CREATE OR REPLACE FUNCTION refresh_admin_statistics()
RETURNS void AS $$
BEGIN
    UPDATE admin_statistics SET
        total_tenants         = (SELECT COUNT(*) FROM tenants WHERE deleted_at IS NULL),
        active_tenants        = (SELECT COUNT(*) FROM tenants WHERE is_active = true AND deleted_at IS NULL),
        total_devices         = (SELECT COUNT(*) FROM paired_devices),
        active_devices        = (SELECT COUNT(*) FROM paired_devices WHERE status = 'ONLINE'),
        pending_messages      = (SELECT COUNT(*) FROM messages WHERE status = 'PENDING'),
        queued_messages       = (SELECT COUNT(*) FROM messages WHERE status = 'QUEUED'),
        processing_messages   = (SELECT COUNT(*) FROM messages WHERE status = 'PROCESSING'),
        total_messages_today  = (SELECT COUNT(*) FROM messages WHERE created_at >= CURRENT_DATE),
        sent_messages_today   = (SELECT COUNT(*) FROM messages WHERE status IN ('SENT','DELIVERED') AND created_at >= CURRENT_DATE),
        failed_messages_today = (SELECT COUNT(*) FROM messages WHERE status = 'FAILED' AND created_at >= CURRENT_DATE),
        total_balance         = (SELECT COALESCE(SUM(balance), 0) FROM tenants WHERE deleted_at IS NULL),
        total_events_today    = 0,
        parsed_events_today   = 0,
        failed_events_today   = 0,
        last_updated          = NOW()
    WHERE id = 1;
END;
$$ LANGUAGE plpgsql;`
	}

	if _, err := db.Exec(fn); err != nil {
		if strings.Contains(err.Error(), "paired_devices") || strings.Contains(err.Error(), "messages") {
			log.Printf("  ✗ Function creation failed (missing cross-schema tables): %v", err)
			log.Println("  ℹ The function references tenant tables - this may require search_path adjustments")
		} else {
			log.Printf("  ✗ Function creation failed: %v", err)
		}
	} else {
		log.Println("  ✓ refresh_admin_statistics() created/updated successfully")
	}
}

func migrateRiverSchema(ctx context.Context, pool *pgxpool.Pool, schemaName string) error {
	migrator, err := rivermigrate.New[pgx.Tx](riverpgxv5.New(pool), &rivermigrate.Config{
		Schema: schemaName,
	})
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	if _, err = migrator.Migrate(ctx, rivermigrate.DirectionUp, &rivermigrate.MigrateOpts{}); err != nil {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
