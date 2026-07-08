# Currency Management Module

## Overview

This module provides currency and exchange rate management functionality for the tariff pricing system. It supports multiple currencies with historical exchange rate tracking.

## Requirements

This module implements the following requirements:
- **31.1**: Support for USD as primary currency
- **31.2**: Support for YER as secondary currency
- **31.3**: Store current exchange rates between currencies
- **31.4**: Allow updating exchange rates from admin interface
- **31.5**: Maintain historical record of exchange rates
- **31.6**: Convert amounts between currencies using current exchange rate
- **31.9**: Support adding new currencies in the future

## Database Schema

### currencies table
- `id`: Primary key
- `code`: 3-letter currency code (USD, YER, SAR)
- `name`: Full currency name
- `symbol`: Currency symbol
- `decimal_places`: Number of decimal places (default: 2)
- `is_active`: Whether the currency is active
- `created_at`, `updated_at`: Timestamps

### exchange_rates table
- `id`: Primary key
- `from_currency`: Source currency code
- `to_currency`: Target currency code
- `rate`: Exchange rate (decimal 15,6)
- `effective_from`: When this rate becomes active
- `effective_to`: When this rate expires (NULL for current rate)
- `source`: Source of the rate (manual, api, central_bank)
- `notes`: Additional notes
- `created_at`, `updated_at`: Timestamps

## Features

### Currency Operations
- Get all active currencies
- Get currency by code
- Create new currency
- Update currency details

### Exchange Rate Operations
- Get current exchange rate between two currencies
- Convert amount from one currency to another
- Get historical exchange rates
- Create new exchange rate (automatically expires previous rate)

## Usage

### Initialize Repository

```go
import (
    "sms-gateway/services/backend/internal/currency"
    "gorm.io/gorm"
)

repo := currency.NewRepository(db)
```

### Get All Currencies

```go
currencies, err := repo.GetAllCurrencies(ctx)
if err != nil {
    // Handle error
}
```

### Convert Amount

```go
// Convert 100 USD to YER
convertedAmount, err := repo.ConvertAmount(ctx, 100.0, "USD", "YER")
if err != nil {
    // Handle error
}
// convertedAmount will be 25000.0 (assuming rate is 250)
```

### Get Current Exchange Rate

```go
rate, err := repo.GetCurrentExchangeRate(ctx, "USD", "YER")
if err != nil {
    // Handle error
}
fmt.Printf("1 %s = %.6f %s\n", rate.FromCurrency, rate.Rate, rate.ToCurrency)
```

### Create New Exchange Rate

```go
newRate := &models.ExchangeRate{
    FromCurrency:  "USD",
    ToCurrency:    "YER",
    Rate:          260.0,
    EffectiveFrom: time.Now(),
    Source:        "manual",
    Notes:         "Updated market rate",
}

err := repo.CreateExchangeRate(ctx, newRate)
if err != nil {
    // Handle error
}
// This will automatically set effective_to on the previous rate
```

### Get Exchange Rate History

```go
// Get last 10 exchange rates for USD to YER
history, err := repo.GetExchangeRateHistory(ctx, "USD", "YER", 10)
if err != nil {
    // Handle error
}

for _, rate := range history {
    fmt.Printf("Rate: %.6f, Effective: %s\n", rate.Rate, rate.EffectiveFrom)
}
```

## Testing

Run the tests:

```bash
cd services/backend
go test -v ./internal/currency
```

The test suite includes:
- Currency retrieval tests
- Exchange rate retrieval tests
- Amount conversion tests
- Exchange rate creation with history tracking
- Currency creation and update tests

## Initial Data

The migrations include initial data for:

### Currencies
- USD (US Dollar, $)
- YER (Yemeni Rial, ر.ي)
- SAR (Saudi Riyal, ر.س)

### Exchange Rates (approximate)
- USD to YER: 250.0
- YER to USD: 0.004
- USD to SAR: 3.75
- SAR to USD: 0.266667
- YER to SAR: 0.015
- SAR to YER: 66.666667

**Note**: These are initial approximate rates and should be updated based on current market rates.

## Integration with Tariff System

This module integrates with the tariff system to:
1. Store tariff costs in different currencies
2. Convert costs between currencies for display
3. Process payments in customer's preferred currency
4. Maintain accurate financial records with original currency

## Future Enhancements

- Automatic exchange rate updates from external APIs
- Support for cryptocurrency
- Multi-currency wallet support
- Currency conversion fees
- Real-time rate alerts
