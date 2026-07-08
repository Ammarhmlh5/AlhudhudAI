package currency

import (
	"context"
	"fmt"
	"time"

	"github.com/al-hudhud/backend/internal/models"

	"gorm.io/gorm"
)

// Repository handles currency and exchange rate operations
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new currency repository
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// GetAllCurrencies retrieves all currencies
func (r *Repository) GetAllCurrencies(ctx context.Context) ([]models.Currency, error) {
	var currencies []models.Currency
	err := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Order("code ASC").
		Find(&currencies).Error
	return currencies, err
}

// GetCurrencyByCode retrieves a currency by its code
func (r *Repository) GetCurrencyByCode(ctx context.Context, code string) (*models.Currency, error) {
	var currency models.Currency
	err := r.db.WithContext(ctx).
		Where("code = ? AND is_active = ?", code, true).
		First(&currency).Error
	if err != nil {
		return nil, err
	}
	return &currency, nil
}

// GetCurrentExchangeRate retrieves the current active exchange rate between two currencies
func (r *Repository) GetCurrentExchangeRate(ctx context.Context, fromCurrency, toCurrency string) (*models.ExchangeRate, error) {
	var rate models.ExchangeRate
	now := time.Now()
	
	err := r.db.WithContext(ctx).
		Where("from_currency = ? AND to_currency = ?", fromCurrency, toCurrency).
		Where("effective_from <= ?", now).
		Where("effective_to IS NULL OR effective_to > ?", now).
		Order("effective_from DESC").
		First(&rate).Error
	
	if err != nil {
		return nil, err
	}
	
	return &rate, nil
}

// ConvertAmount converts an amount from one currency to another
func (r *Repository) ConvertAmount(ctx context.Context, amount float64, fromCurrency, toCurrency string) (float64, error) {
	// If same currency, no conversion needed
	if fromCurrency == toCurrency {
		return amount, nil
	}
	
	// Get exchange rate
	rate, err := r.GetCurrentExchangeRate(ctx, fromCurrency, toCurrency)
	if err != nil {
		return 0, fmt.Errorf("exchange rate not found for %s to %s: %w", fromCurrency, toCurrency, err)
	}
	
	// Convert amount
	convertedAmount := amount * rate.Rate
	
	return convertedAmount, nil
}

// GetExchangeRateHistory retrieves historical exchange rates for a currency pair
func (r *Repository) GetExchangeRateHistory(ctx context.Context, fromCurrency, toCurrency string, limit int) ([]models.ExchangeRate, error) {
	var rates []models.ExchangeRate
	
	query := r.db.WithContext(ctx).
		Where("from_currency = ? AND to_currency = ?", fromCurrency, toCurrency).
		Order("effective_from DESC")
	
	if limit > 0 {
		query = query.Limit(limit)
	}
	
	err := query.Find(&rates).Error
	return rates, err
}

// CreateExchangeRate creates a new exchange rate
func (r *Repository) CreateExchangeRate(ctx context.Context, rate *models.ExchangeRate) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Set effective_to for the previous active rate
		now := time.Now()
		err := tx.Model(&models.ExchangeRate{}).
			Where("from_currency = ? AND to_currency = ?", rate.FromCurrency, rate.ToCurrency).
			Where("effective_to IS NULL").
			Update("effective_to", now).Error
		if err != nil {
			return fmt.Errorf("failed to update previous rate: %w", err)
		}
		
		// Create new rate
		if err := tx.Create(rate).Error; err != nil {
			return fmt.Errorf("failed to create exchange rate: %w", err)
		}
		
		return nil
	})
}

// UpdateCurrency updates a currency
func (r *Repository) UpdateCurrency(ctx context.Context, code string, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).
		Model(&models.Currency{}).
		Where("code = ?", code).
		Updates(updates).Error
}

// CreateCurrency creates a new currency
func (r *Repository) CreateCurrency(ctx context.Context, currency *models.Currency) error {
	return r.db.WithContext(ctx).Create(currency).Error
}
