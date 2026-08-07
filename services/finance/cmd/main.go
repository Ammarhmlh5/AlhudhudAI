package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"finance/internal/api"
	"finance/internal/models"
	"finance/internal/services"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	// Parse flags
	port := flag.String("port", "8081", "HTTP server port")
	dbURL := flag.String("db", "", "PostgreSQL connection URL (optional, uses env if empty)")
	flag.Parse()

	// Load environment variables
	_ = godotenv.Load()

	// Database connection
	dsn := *dbURL
	if dsn == "" {
		dsn = os.Getenv("FINANCE_DB_URL")
	}
	if dsn == "" {
		log.Fatal("FINANCE_DB_URL environment variable or -db flag is required")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("فشل الاتصال بقاعدة البيانات: %v", err)
	}

	// Auto-migrate models
	log.Println("جاري إنشاء الجداول...")
	if err := db.AutoMigrate(
		&models.Account{},
		&models.JournalEntry{},
		&models.JournalEntryLine{},
		&models.Invoice{},
	); err != nil {
		log.Fatalf("فشل إنشاء الجداول: %v", err)
	}

	// Initialize services
	accService := services.NewAccountingService(db)
	accManager := services.NewAccountManager(db)
	invoiceService := services.NewInvoiceService(db)

	// Seed default accounts (if not exist)
	log.Println("جاري إنشاء الحسابات الأساسية...")
	if err := accManager.SeedDefaultAccounts(nil); err != nil {
		log.Printf("تحذير: فشل إنشاء الحسابات الأساسية: %v", err)
	}

	// Initialize handler
	handler := api.NewFinanceHandlerV2(db, accService, accManager, invoiceService)

	// Register routes
	mux := http.NewServeMux()
	api.RegisterAllRoutes(mux, handler)

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	// Start server
	addr := fmt.Sprintf(":%s", *port)
	log.Printf("خدمة المالية تعمل على: http://localhost%s", addr)
	log.Printf("Available endpoints:")
	log.Printf("  POST   /api/finance/charge          - شحن رصيد")
	log.Printf("  POST   /api/finance/debit-cashbox   - سند صرف")
	log.Printf("  POST   /api/finance/cashboxes       - إنشاء صندوق")
	log.Printf("  GET    /api/finance/cashboxes       - قائمة الصناديق")
	log.Printf("  GET    /api/finance/balance         - رصيد المحفظة")
	log.Printf("  GET    /api/finance/statement       - كشف حساب")
	log.Printf("  POST   /api/finance/reverse         - عكس سند")
	log.Printf("  POST   /api/finance/purchase        - شراء باقة")
	log.Printf("  GET    /api/finance/transactions    - قائمة المعاملات")
	log.Printf("  GET    /api/finance/invoices        - قائمة الفواتير")
	log.Printf("  GET    /api/finance/invoices/print  - طباعة فاتورة")
	log.Printf("  POST   /api/finance/wallets         - إنشاء محفظة")
	log.Printf("  GET    /health                      - فحص الصحة")

	// Graceful shutdown
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		log.Println("إيقاف الخدمة...")
		os.Exit(0)
	}()

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("فشل تشغيل الخادم: %v", err)
	}
}