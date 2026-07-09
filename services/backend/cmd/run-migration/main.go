package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println(".env file not found, using system environment variables")
	}

	// Database connection string - use 127.0.0.1 instead of localhost
	connStr := fmt.Sprintf(
		"host=127.0.0.1 port=%s user=%s password=%s dbname=%s sslmode=disable connect_timeout=30",
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
	)

	// Connect to database with retry
	var db *sql.DB
	var err error
	
	fmt.Println("Attempting to connect to database...")
	for i := 0; i < 5; i++ {
		db, err = sql.Open("postgres", connStr)
		if err != nil {
			fmt.Printf("Connection attempt %d failed: %v\n", i+1, err)
			time.Sleep(2 * time.Second)
			continue
		}
		
		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)
		db.SetConnMaxLifetime(5 * time.Minute)
		
		if err = db.Ping(); err != nil {
			fmt.Printf("Ping attempt %d failed: %v\n", i+1, err)
			db.Close()
			time.Sleep(2 * time.Second)
			continue
		}
		
		fmt.Println("✓ Connected to database successfully")
		break
	}
	
	if err != nil {
		log.Fatalf("Failed to connect to database after 5 attempts: %v", err)
	}
	defer db.Close()

	// Migration files to run
	migrations := []string{
		"050_fix_rbac_tables.sql",
		"051_add_not_null_constraints.sql",
		"052_unify_decimal_precision.sql",
	}

	migrationsDir := "./migrations"

	for _, migrationFile := range migrations {
		filePath := filepath.Join(migrationsDir, migrationFile)
		
		// Check if file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			fmt.Printf("⚠ Migration file not found: %s\n", migrationFile)
			continue
		}

		fmt.Printf("\n📄 Running migration: %s\n", migrationFile)

		// Read migration file
		content, err := os.ReadFile(filePath)
		if err != nil {
			log.Fatalf("Failed to read migration file: %v", err)
		}

		// Execute migration
		sql := string(content)
		
		// Split by semicolons but handle BEGIN; and END; blocks
		// Execute the entire script at once (it contains BEGIN; and COMMIT;)
		_, err = db.Exec(sql)
		if err != nil {
			// Check if it's a "already exists" error
			if strings.Contains(err.Error(), "already exists") || 
			   strings.Contains(err.Error(), "duplicate key") {
				fmt.Printf("⚠ Migration %s: Some objects may already exist\n", migrationFile)
			} else {
				log.Fatalf("Failed to execute migration %s: %v", migrationFile, err)
			}
		} else {
			fmt.Printf("✓ Migration %s completed successfully\n", migrationFile)
		}
	}

	fmt.Println("\n✅ All migrations completed!")
	
	// ================================================================
	// التحقق من نجاح الإصلاحات
	// ================================================================
	fmt.Println("\n📊 Verification Results:")
	fmt.Println("========================================")
	
	// 1. التحقق من جداول RBAC
	fmt.Println("\n1. Checking RBAC tables...")
	var tableCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name IN ('role_permissions', 'user_roles')`).Scan(&tableCount)
	if err != nil {
		log.Printf("Error checking RBAC tables: %v", err)
	} else if tableCount == 2 {
		fmt.Println("   ✓ role_permissions and user_roles tables created")
	} else {
		fmt.Printf("   ⚠ Only %d/2 RBAC tables found\n", tableCount)
	}
	
	// 2. التحقق من قيود NOT NULL
	fmt.Println("\n2. Checking NOT NULL constraints...")
	notNullCols := []struct {
		table  string
		column string
	}{
		{"tenants", "balance"},
		{"messages", "tenant_id"},
		{"messages", "phone_number"},
		{"messages", "content"},
	}
	
	for _, col := range notNullCols {
		var isNotNull string
		err = db.QueryRow(`SELECT is_nullable FROM information_schema.columns WHERE table_name = $1 AND column_name = $2`, col.table, col.column).Scan(&isNotNull)
		if err != nil {
			fmt.Printf("   ⚠ %s.%s: Error checking\n", col.table, col.column)
		} else if isNotNull == "NO" {
			fmt.Printf("   ✓ %s.%s is NOT NULL\n", col.table, col.column)
		} else {
			fmt.Printf("   ⚠ %s.%s still allows NULL\n", col.table, col.column)
		}
	}
	
	// 3. التحقق من دقة الأعمدة المالية
	fmt.Println("\n3. Checking decimal precision...")
	decimalCols := []struct {
		table  string
		column string
	}{
		{"accounting_transactions", "amount"},
		{"payment_transactions", "amount"},
		{"tenants", "balance"},
	}
	
	for _, col := range decimalCols {
		var precision, scale int
		err = db.QueryRow(`SELECT numeric_precision, numeric_scale FROM information_schema.columns WHERE table_name = $1 AND column_name = $2`, col.table, col.column).Scan(&precision, &scale)
		if err != nil {
			fmt.Printf("   ⚠ %s.%s: Error checking\n", col.table, col.column)
		} else {
			fmt.Printf("   ✓ %s.%s: DECIMAL(%d,%d)\n", col.table, col.column, precision, scale)
		}
	}
	
	// 4. التحقق من قيود CHECK
	fmt.Println("\n4. Checking CHECK constraints...")
	var checkCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM pg_constraint WHERE conname LIKE '%_non_negative' OR conname LIKE '%_check'`).Scan(&checkCount)
	if err != nil {
		log.Printf("Error checking CHECK constraints: %v", err)
	} else {
		fmt.Printf("   ✓ Found %d CHECK constraints\n", checkCount)
	}
	
	fmt.Println("\n========================================")
	fmt.Println("✅ Database fixes verification completed!")
	
	// ================================================================
	// فحص إضافي لـ RLS والفهارس
	// ================================================================
	fmt.Println("\n📊 Additional Database Checks:")
	fmt.Println("========================================")
	
	// التحقق من RLS
	fmt.Println("\n5. Checking Row Level Security (RLS)...")
	var rlsCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM pg_class WHERE relrowsecurity = true AND relkind = 'r'`).Scan(&rlsCount)
	if err != nil {
		fmt.Printf("   ⚠ Error checking RLS: %v\n", err)
	} else if rlsCount > 0 {
		fmt.Printf("   ✓ Found %d tables with RLS enabled\n", rlsCount)
	} else {
		fmt.Println("   ⚠ No tables with RLS found - this is a security issue!")
	}
	
	// التحقق من الفهارس على tenant_id
	fmt.Println("\n6. Checking tenant_id indexes...")
	var indexCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM pg_indexes WHERE indexname LIKE '%tenant%'`).Scan(&indexCount)
	if err != nil {
		fmt.Printf("   ⚠ Error checking indexes: %v\n", err)
	} else if indexCount > 0 {
		fmt.Printf("   ✓ Found %d indexes on tenant-related columns\n", indexCount)
	} else {
		fmt.Println("   ⚠ No tenant-related indexes found")
	}
	
	// التحقق من الـ triggers لـ updated_at
	fmt.Println("\n7. Checking updated_at triggers...")
	var triggerCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM pg_trigger WHERE tgname LIKE '%updated_at%'`).Scan(&triggerCount)
	if err != nil {
		fmt.Printf("   ⚠ Error checking triggers: %v\n", err)
	} else if triggerCount > 0 {
		fmt.Printf("   ✓ Found %d updated_at triggers\n", triggerCount)
	} else {
		fmt.Println("   ⚠ No updated_at triggers found")
	}
	
	fmt.Println("\n========================================")
}