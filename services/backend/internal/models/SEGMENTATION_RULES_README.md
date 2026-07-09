# Segmentation Rules Implementation

## Overview

This document describes the implementation of the `segmentation_rules` table and related functionality for managing SMS message segmentation in the Al-Hudhud SMS Gateway system.

## Requirements

This implementation satisfies the following requirements from the tariff-pricing-system spec:

- **Requirement 7.1**: Store segmentation rule: First segment 70 characters
- **Requirement 7.2**: Store segmentation rule: Second segment 65 characters
- **Requirement 7.3**: Store segmentation rule: Third segment 62 characters
- **Requirement 7.4**: Support different segmentation rules for each gateway and network
- **Requirement 7.6**: Automatically calculate the number of segments required

## Database Schema

### Table: `segmentation_rules`

```sql
CREATE TABLE segmentation_rules (
    id SERIAL PRIMARY KEY,
    gateway_id INTEGER REFERENCES dedicated_gateways(id), -- NULL = default rules
    country_code VARCHAR(2),
    network_operator VARCHAR(100),
    encoding_type VARCHAR(20) NOT NULL, -- 'gsm7', 'unicode'
    segment_1_chars INTEGER NOT NULL, -- 70 for first segment
    segment_2_chars INTEGER NOT NULL, -- 65 for second segment
    segment_3_chars INTEGER NOT NULL, -- 62 for third segment
    segment_n_chars INTEGER NOT NULL, -- for subsequent segments
    header_overhead INTEGER DEFAULT 0, -- characters used for header
    description TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Indexes

- `idx_segmentation_gateway`: Index on `gateway_id` for fast lookup by gateway
- `idx_segmentation_country`: Index on `country_code` for country-specific rules
- `idx_segmentation_encoding`: Index on `encoding_type` for encoding-specific rules
- `idx_segmentation_active`: Index on `is_active` for filtering active rules

### Default Rules

Two default segmentation rules are inserted during migration:

1. **GSM-7 Encoding**: 70, 65, 62, 62 characters
2. **Unicode Encoding**: 70, 65, 62, 62 characters

## Go Model

### SegmentationRule Struct

```go
type SegmentationRule struct {
    ID               int       `json:"id"`
    GatewayID        *int      `json:"gateway_id,omitempty"`
    CountryCode      *string   `json:"country_code,omitempty"`
    NetworkOperator  *string   `json:"network_operator,omitempty"`
    EncodingType     string    `json:"encoding_type"`
    Segment1Chars    int       `json:"segment_1_chars"`
    Segment2Chars    int       `json:"segment_2_chars"`
    Segment3Chars    int       `json:"segment_3_chars"`
    SegmentNChars    int       `json:"segment_n_chars"`
    HeaderOverhead   int       `json:"header_overhead"`
    Description      *string   `json:"description,omitempty"`
    IsActive         bool      `json:"is_active"`
    CreatedAt        time.Time `json:"created_at"`
    UpdatedAt        time.Time `json:"updated_at"`
}
```

### Key Methods

#### CalculateSegments

Calculates the number of segments needed for a message of given length:

```go
func (sr *SegmentationRule) CalculateSegments(messageLength int) int
```

**Algorithm:**

1. If message length ≤ 0, return 0
2. Adjust for header overhead
3. If fits in first segment (≤ segment_1_chars), return 1
4. If fits in first + second (≤ segment_1_chars + segment_2_chars), return 2
5. If fits in first + second + third, return 3
6. Calculate additional segments needed using segment_n_chars

**Example:**

```go
rule := SegmentationRule{
    Segment1Chars: 70,
    Segment2Chars: 65,
    Segment3Chars: 62,
    SegmentNChars: 62,
}

segments := rule.CalculateSegments(150) // Returns 3
```

#### Validate

Validates the segmentation rule:

```go
func (sr *SegmentationRule) Validate() error
```

**Validation Rules:**

- All segment sizes must be positive (> 0)
- Header overhead must be non-negative (≥ 0)
- Encoding type must be 'gsm7' or 'unicode'

## Usage Examples

### Creating a Custom Segmentation Rule

```go
rule := &SegmentationRule{
    GatewayID:      &gatewayID,
    CountryCode:    stringPtr("YE"),
    NetworkOperator: stringPtr("Yemen Mobile"),
    EncodingType:   EncodingTypeGSM7,
    Segment1Chars:  70,
    Segment2Chars:  65,
    Segment3Chars:  62,
    SegmentNChars:  62,
    HeaderOverhead: 0,
    IsActive:       true,
}

if err := rule.Validate(); err != nil {
    return err
}

// Save to database
db.Create(rule)
```

### Calculating Message Segments

```go
// Get the appropriate segmentation rule
var rule SegmentationRule
db.Where("encoding_type = ? AND is_active = true", EncodingTypeGSM7).
   Where("gateway_id IS NULL"). // Default rule
   First(&rule)

// Calculate segments for a message
messageText := "Your message here..."
segments := rule.CalculateSegments(len(messageText))

fmt.Printf("Message will be split into %d segments\n", segments)
```

### Querying Rules

```go
// Get default GSM-7 rule
var defaultGSM7 SegmentationRule
db.Where("encoding_type = ? AND gateway_id IS NULL AND is_active = true", 
         EncodingTypeGSM7).First(&defaultGSM7)

// Get gateway-specific rule
var gatewayRule SegmentationRule
db.Where("gateway_id = ? AND encoding_type = ? AND is_active = true", 
         gatewayID, EncodingTypeUnicode).First(&gatewayRule)

// Get country-specific rule
var countryRule SegmentationRule
db.Where("country_code = ? AND encoding_type = ? AND is_active = true", 
         "YE", EncodingTypeGSM7).First(&countryRule)
```

## Testing

### Unit Tests

The implementation includes comprehensive unit tests:

- `TestSegmentationRule_CalculateSegments`: Tests segment calculation for various message lengths
- `TestSegmentationRule_Validate`: Tests validation logic
- `TestDefaultSegmentationRules`: Tests default rule generation

Run tests:

```bash
cd services/backend
go test -v ./internal/models -run TestSegmentationRule
```

### Test Cases

#### Segment Calculation Tests

| Message Length | Expected Segments | Notes |
|----------------|-------------------|-------|
| 0              | 0                 | Empty message |
| 50             | 1                 | Fits in first segment |
| 70             | 1                 | Exactly first segment |
| 100            | 2                 | Needs second segment |
| 135            | 2                 | Exactly two segments (70+65) |
| 150            | 3                 | Needs third segment |
| 197            | 3                 | Exactly three segments (70+65+62) |
| 250            | 4                 | Needs fourth segment |
| 321            | 5                 | Five segments (70+65+62+62+62) |

#### Validation Tests

- Valid GSM-7 rule
- Valid Unicode rule
- Invalid segment sizes (≤ 0)
- Invalid header overhead (< 0)
- Invalid encoding type

## Migration

### Running the Migration

```bash
# Using the migration script
cd services/backend
go run ./cmd/migrate/main.go

# Or using psql directly
psql -U hudhud_admin -d al_hudhud -f migrations/011_segmentation_rules_table.sql
```

### Rollback

```sql
-- Drop indexes
DROP INDEX IF EXISTS idx_segmentation_active;
DROP INDEX IF EXISTS idx_segmentation_encoding;
DROP INDEX IF EXISTS idx_segmentation_country;
DROP INDEX IF EXISTS idx_segmentation_gateway;

-- Drop table
DROP TABLE IF EXISTS segmentation_rules;
```

## Integration with Tariff System

The segmentation rules integrate with the tariff system to calculate accurate message costs:

1. **Message Submission**: When a message is submitted, the system:
   - Detects the encoding type (GSM-7 or Unicode)
   - Retrieves the appropriate segmentation rule
   - Calculates the number of segments
   - Multiplies by the tariff cost per message

2. **Cost Calculation**:
   ```go
   segments := rule.CalculateSegments(len(messageText))
   totalCost := tariff.CostPerMessage * float64(segments)
   ```

3. **Rule Priority**: Rules are selected in this order:
   - Gateway + Country + Operator specific
   - Gateway + Country specific
   - Gateway specific
   - Country + Operator specific
   - Country specific
   - Default rule (gateway_id IS NULL)

## Future Enhancements

1. **Dynamic Rule Updates**: Support for updating rules without system restart
2. **Rule Versioning**: Track historical changes to segmentation rules
3. **A/B Testing**: Test different segmentation strategies
4. **Analytics**: Track segment distribution and optimization opportunities
5. **Custom Encoding**: Support for additional encoding types beyond GSM-7 and Unicode

## References

- Spec: `.kiro/specs/tariff-pricing-system/requirements.md` (Requirements 7.1-7.4, 7.6)
- Design: `.kiro/specs/tariff-pricing-system/design.md` (Section 14)
- Migration: `services/backend/migrations/011_segmentation_rules_table.sql`
- Model: `services/backend/internal/models/segmentation_rule.go`
- Tests: `services/backend/internal/models/segmentation_rule_test.go`
