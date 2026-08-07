package currency

import (
	"context"
	"testing"
	"time"

	"github.com/al-hudhud/backend/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// :memory: sqlite is per-connection; pin the pool to a single connection
	// so tables created by AutoMigrate are visible to later queries.
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	// Auto migrate tables
	err = db.AutoMigrate(&models.Currency{}, &models.ExchangeRate{})
	require.NoError(t, err)

	return db
}

func seedCurrencies(t *testing.T, db *gorm.DB) {
	currencies := []models.Currency{
		{Code: "USD", Name: "US Dollar", Symbol: "$", DecimalPlaces: 2, IsActive: true},
		{Code: "YER", Name: "Yemeni Rial", Symbol: "ر.ي", DecimalPlaces: 2, IsActive: true},
		{Code: "SAR", Name: "Saudi Riyal", Symbol: "ر.س", DecimalPlaces: 2, IsActive: true},
	}

	for _, currency := range currencies {
		err := db.Create(&currency).Error
		require.NoError(t, err)
	}
}

func seedExchangeRates(t *testing.T, db *gorm.DB) {
	rates := []models.ExchangeRate{
		{
			FromCurrency:  "USD",
			ToCurrency:    "YER",
			Rate:          250.0,
			EffectiveFrom: time.Now().Add(-24 * time.Hour),
			Source:        "manual",
		},
		{
			FromCurrency:  "YER",
			ToCurrency:    "USD",
			Rate:          0.004,
			EffectiveFrom: time.Now().Add(-24 * time.Hour),
			Source:        "manual",
		},
	}

	for _, rate := range rates {
		err := db.Create(&rate).Error
		require.NoError(t, err)
	}
}

func TestGetAllCurrencies(t *testing.T) {
	db := setupTestDB(t)
	seedCurrencies(t, db)
	repo := NewRepository(db)

	currencies, err := repo.GetAllCurrencies(context.Background())
	require.NoError(t, err)
	assert.Len(t, currencies, 3)
	assert.Equal(t, "SAR", currencies[0].Code) // Ordered by code ASC
	assert.Equal(t, "USD", currencies[1].Code)
	assert.Equal(t, "YER", currencies[2].Code)
}

func TestGetCurrencyByCode(t *testing.T) {
	db := setupTestDB(t)
	seedCurrencies(t, db)
	repo := NewRepository(db)

	t.Run("existing currency", func(t *testing.T) {
		currency, err := repo.GetCurrencyByCode(context.Background(), "USD")
		require.NoError(t, err)
		assert.Equal(t, "USD", currency.Code)
		assert.Equal(t, "US Dollar", currency.Name)
		assert.Equal(t, "$", currency.Symbol)
	})

	t.Run("non-existing currency", func(t *testing.T) {
		_, err := repo.GetCurrencyByCode(context.Background(), "EUR")
		assert.Error(t, err)
	})
}

func TestGetCurrentExchangeRate(t *testing.T) {
	db := setupTestDB(t)
	seedCurrencies(t, db)
	seedExchangeRates(t, db)
	repo := NewRepository(db)

	t.Run("existing rate", func(t *testing.T) {
		rate, err := repo.GetCurrentExchangeRate(context.Background(), "USD", "YER")
		require.NoError(t, err)
		assert.Equal(t, "USD", rate.FromCurrency)
		assert.Equal(t, "YER", rate.ToCurrency)
		assert.Equal(t, 250.0, rate.Rate)
	})

	t.Run("non-existing rate", func(t *testing.T) {
		_, err := repo.GetCurrentExchangeRate(context.Background(), "USD", "EUR")
		assert.Error(t, err)
	})
}

func TestConvertAmount(t *testing.T) {
	db := setupTestDB(t)
	seedCurrencies(t, db)
	seedExchangeRates(t, db)
	repo := NewRepository(db)

	t.Run("same currency", func(t *testing.T) {
		converted, err := repo.ConvertAmount(context.Background(), 100.0, "USD", "USD")
		require.NoError(t, err)
		assert.Equal(t, 100.0, converted)
	})

	t.Run("USD to YER", func(t *testing.T) {
		converted, err := repo.ConvertAmount(context.Background(), 10.0, "USD", "YER")
		require.NoError(t, err)
		assert.Equal(t, 2500.0, converted) // 10 * 250
	})

	t.Run("YER to USD", func(t *testing.T) {
		converted, err := repo.ConvertAmount(context.Background(), 1000.0, "YER", "USD")
		require.NoError(t, err)
		assert.Equal(t, 4.0, converted) // 1000 * 0.004
	})

	t.Run("non-existing rate", func(t *testing.T) {
		_, err := repo.ConvertAmount(context.Background(), 100.0, "USD", "EUR")
		assert.Error(t, err)
	})
}

func TestCreateExchangeRate(t *testing.T) {
	db := setupTestDB(t)
	seedCurrencies(t, db)
	seedExchangeRates(t, db)
	repo := NewRepository(db)

	// Create a new rate
	newRate := &models.ExchangeRate{
		FromCurrency:  "USD",
		ToCurrency:    "YER",
		Rate:          260.0, // New rate
		EffectiveFrom: time.Now(),
		Source:        "manual",
		Notes:         "Updated rate",
	}

	err := repo.CreateExchangeRate(context.Background(), newRate)
	require.NoError(t, err)

	// Verify the new rate is active
	currentRate, err := repo.GetCurrentExchangeRate(context.Background(), "USD", "YER")
	require.NoError(t, err)
	assert.Equal(t, 260.0, currentRate.Rate)

	// Verify the old rate has effective_to set
	var oldRates []models.ExchangeRate
	err = db.Where("from_currency = ? AND to_currency = ? AND effective_to IS NOT NULL", "USD", "YER").
		Find(&oldRates).Error
	require.NoError(t, err)
	assert.Len(t, oldRates, 1)
	assert.NotNil(t, oldRates[0].EffectiveTo)
}

func TestGetExchangeRateHistory(t *testing.T) {
	db := setupTestDB(t)
	seedCurrencies(t, db)
	repo := NewRepository(db)

	// Create multiple rates
	rates := []models.ExchangeRate{
		{
			FromCurrency:  "USD",
			ToCurrency:    "YER",
			Rate:          240.0,
			EffectiveFrom: time.Now().Add(-72 * time.Hour),
			EffectiveTo:   ptrTime(time.Now().Add(-48 * time.Hour)),
			Source:        "manual",
		},
		{
			FromCurrency:  "USD",
			ToCurrency:    "YER",
			Rate:          250.0,
			EffectiveFrom: time.Now().Add(-48 * time.Hour),
			EffectiveTo:   ptrTime(time.Now().Add(-24 * time.Hour)),
			Source:        "manual",
		},
		{
			FromCurrency:  "USD",
			ToCurrency:    "YER",
			Rate:          260.0,
			EffectiveFrom: time.Now().Add(-24 * time.Hour),
			Source:        "manual",
		},
	}

	for _, rate := range rates {
		err := db.Create(&rate).Error
		require.NoError(t, err)
	}

	history, err := repo.GetExchangeRateHistory(context.Background(), "USD", "YER", 10)
	require.NoError(t, err)
	assert.Len(t, history, 3)
	// Should be ordered by effective_from DESC
	assert.Equal(t, 260.0, history[0].Rate)
	assert.Equal(t, 250.0, history[1].Rate)
	assert.Equal(t, 240.0, history[2].Rate)
}

func TestCreateCurrency(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	currency := &models.Currency{
		Code:          "EUR",
		Name:          "Euro",
		Symbol:        "€",
		DecimalPlaces: 2,
		IsActive:      true,
	}

	err := repo.CreateCurrency(context.Background(), currency)
	require.NoError(t, err)
	assert.NotZero(t, currency.ID)

	// Verify it was created
	retrieved, err := repo.GetCurrencyByCode(context.Background(), "EUR")
	require.NoError(t, err)
	assert.Equal(t, "Euro", retrieved.Name)
}

func TestUpdateCurrency(t *testing.T) {
	db := setupTestDB(t)
	seedCurrencies(t, db)
	repo := NewRepository(db)

	updates := map[string]interface{}{
		"name":   "Updated Dollar",
		"symbol": "US$",
	}

	err := repo.UpdateCurrency(context.Background(), "USD", updates)
	require.NoError(t, err)

	// Verify the update
	currency, err := repo.GetCurrencyByCode(context.Background(), "USD")
	require.NoError(t, err)
	assert.Equal(t, "Updated Dollar", currency.Name)
	assert.Equal(t, "US$", currency.Symbol)
}

// Helper function to create a pointer to time.Time
func ptrTime(t time.Time) *time.Time {
	return &t
}
