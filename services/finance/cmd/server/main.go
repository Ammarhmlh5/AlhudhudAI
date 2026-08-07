package main

import (
	"finance/internal/models"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// DSN should be configured via environment variables in production
	// Example DSN for local testing:
	dsn := "host=localhost user=postgres password=postgres dbname=finance_db port=5432 sslmode=disable TimeZone=UTC"
	
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("فشل الاتصال بقاعدة البيانات المحاسبية: %v", err)
	}

	log.Println("تم الاتصال بقاعدة البيانات المحاسبية بنجاح.")

	// AutoMigrate the schema
	err = db.AutoMigrate(
		&models.AccountingPeriod{},
		&models.Account{},
		&models.JournalEntry{},
		&models.JournalEntryLine{},
		&models.Invoice{},
	)
	if err != nil {
		log.Fatalf("فشل ترحيل الجداول (AutoMigrate): %v", err)
	}

	log.Println("تم إنشاء/تحديث جداول القيد المزدوج بنجاح.")
}
