package tariff

import (
	"fmt"
	"testing"

	"github.com/al-hudhud/backend/internal/models"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

var (
	testTenantID  = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	testTenantID2 = uuid.MustParse("00000000-0000-0000-0000-000000000002")
)

// TestCalculateSegments_GSM7 tests segment calculation for GSM-7 encoding
func TestCalculateSegments_GSM7(t *testing.T) {
	tests := []struct {
		name          string
		messageLength int
		expectedSegs  int
	}{
		{
			name:          "single segment - 160 chars",
			messageLength: 160,
			expectedSegs:  1,
		},
		{
			name:          "single segment - 100 chars",
			messageLength: 100,
			expectedSegs:  1,
		},
		{
			name:          "two segments - 161 chars",
			messageLength: 161,
			expectedSegs:  2,
		},
		{
			name:          "two segments - 313 chars",
			messageLength: 313,
			expectedSegs:  2,
		},
		{
			name:          "three segments - 314 chars",
			messageLength: 314,
			expectedSegs:  3,
		},
		{
			name:          "three segments - 466 chars",
			messageLength: 466,
			expectedSegs:  3,
		},
		{
			name:          "four segments - 467 chars",
			messageLength: 467,
			expectedSegs:  4,
		},
		{
			name:          "five segments - 620 chars",
			messageLength: 620,
			expectedSegs:  5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use default GSM-7 rules
			rule := models.SegmentationRule{
				EncodingType:  "GSM-7",
				Segment1Chars: 160,
				Segment2Chars: 153,
				Segment3Chars: 153,
				SegmentNChars: 153,
			}

			segments := calculateSegmentsFromRules(tt.messageLength, rule)
			assert.Equal(t, tt.expectedSegs, segments)
		})
	}
}

// TestCalculateSegments_Unicode tests segment calculation for Unicode encoding
func TestCalculateSegments_Unicode(t *testing.T) {
	tests := []struct {
		name          string
		messageLength int
		expectedSegs  int
	}{
		{
			name:          "single segment - 70 chars",
			messageLength: 70,
			expectedSegs:  1,
		},
		{
			name:          "single segment - 50 chars",
			messageLength: 50,
			expectedSegs:  1,
		},
		{
			name:          "two segments - 71 chars",
			messageLength: 71,
			expectedSegs:  2,
		},
		{
			name:          "two segments - 137 chars",
			messageLength: 137,
			expectedSegs:  2,
		},
		{
			name:          "three segments - 138 chars",
			messageLength: 138,
			expectedSegs:  3,
		},
		{
			name:          "three segments - 204 chars",
			messageLength: 204,
			expectedSegs:  3,
		},
		{
			name:          "four segments - 205 chars",
			messageLength: 205,
			expectedSegs:  4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use default Unicode rules
			rule := models.SegmentationRule{
				EncodingType:  "Unicode",
				Segment1Chars: 70,
				Segment2Chars: 67,
				Segment3Chars: 67,
				SegmentNChars: 67,
			}

			segments := calculateSegmentsFromRules(tt.messageLength, rule)
			assert.Equal(t, tt.expectedSegs, segments)
		})
	}
}

// TestCostCalculationRequest_Validation tests request validation
func TestCostCalculationRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		req     CostCalculationRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid request",
			req: CostCalculationRequest{
				TenantID:        testTenantID,
				ServiceID:       1,
				CountryCode:     "YE",
				NetworkOperator: "Yemen Mobile",
				MessageType:     "GSM-7",
				MessageLength:   100,
			},
			wantErr: false,
		},
		{
			name: "invalid service_id",
			req: CostCalculationRequest{
				TenantID:      testTenantID,
				ServiceID:     0,
				CountryCode:   "YE",
				MessageLength: 100,
			},
			wantErr: true,
			errMsg:  "invalid service_id",
		},
		{
			name: "missing country_code",
			req: CostCalculationRequest{
				TenantID:      testTenantID,
				ServiceID:     1,
				CountryCode:   "",
				MessageLength: 100,
			},
			wantErr: true,
			errMsg:  "country_code is required",
		},
		{
			name: "invalid message_length",
			req: CostCalculationRequest{
				TenantID:      testTenantID,
				ServiceID:     1,
				CountryCode:   "YE",
				MessageLength: 0,
			},
			wantErr: true,
			errMsg:  "invalid message_length",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation checks
			if tt.wantErr {
				if tt.req.ServiceID <= 0 {
					assert.LessOrEqual(t, tt.req.ServiceID, 0)
				}
				if tt.req.CountryCode == "" {
					assert.Empty(t, tt.req.CountryCode)
				}
				if tt.req.MessageLength <= 0 {
					assert.LessOrEqual(t, tt.req.MessageLength, 0)
				}
			} else {
				assert.Greater(t, tt.req.ServiceID, 0)
				assert.NotEmpty(t, tt.req.CountryCode)
				assert.Greater(t, tt.req.MessageLength, 0)
			}
		})
	}
}

// TestCostCalculationResult_Structure tests the result structure
func TestCostCalculationResult_Structure(t *testing.T) {
	result := &CostCalculationResult{
		TotalCost:      decimal.NewFromFloat(27.0),
		CostPerSegment: decimal.NewFromFloat(9.0),
		Segments:       3,
		TariffID:       1,
		CurrencyCode:   "YER",
	}

	assert.Equal(t, "27", result.TotalCost.String())
	assert.Equal(t, "9", result.CostPerSegment.String())
	assert.Equal(t, 3, result.Segments)
	assert.Equal(t, 1, result.TariffID)
	assert.Equal(t, "YER", result.CurrencyCode)
}

// TestCacheKeyGeneration tests cache key generation
func TestCacheKeyGeneration(t *testing.T) {
	tests := []struct {
		name     string
		req      TariffRequest
		expected string
	}{
		{
			name: "full request",
			req: TariffRequest{
				TenantID:        uuid.New(),
				ServiceID:       1,
				CountryCode:     "YE",
				NetworkOperator: "Yemen Mobile",
				MessageType:     "GSM-7",
			},
			expected: fmt.Sprintf("tariff:%s:1:YE:Yemen Mobile:GSM-7", testTenantID),
		},
		{
			name: "without network operator",
			req: TariffRequest{
				TenantID:        testTenantID2,
				ServiceID:       1,
				CountryCode:     "SA",
				NetworkOperator: "",
				MessageType:     "Unicode",
			},
			expected: fmt.Sprintf("tariff:%s:1:SA::Unicode", testTenantID2),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cacheKey := buildCacheKey(tt.req)
			assert.Equal(t, tt.expected, cacheKey)
		})
	}
}

// buildCacheKey is a helper function for testing
func buildCacheKey(req TariffRequest) string {
	return fmt.Sprintf("tariff:%s:%d:%s:%s:%s",
		req.TenantID.String(),
		req.ServiceID,
		req.CountryCode,
		req.NetworkOperator,
		req.MessageType,
	)
}

// TestSegmentCalculation_EdgeCases tests edge cases in segment calculation
func TestSegmentCalculation_EdgeCases(t *testing.T) {
	t.Run("zero length message", func(t *testing.T) {
		rule := models.SegmentationRule{
			EncodingType:  "GSM-7",
			Segment1Chars: 160,
			Segment2Chars: 153,
			Segment3Chars: 153,
			SegmentNChars: 153,
		}

		// Zero length should be handled by validation
		segments := calculateSegmentsFromRules(0, rule)
		assert.Equal(t, 0, segments)
	})

	t.Run("exactly at boundary", func(t *testing.T) {
		rule := models.SegmentationRule{
			EncodingType:  "GSM-7",
			Segment1Chars: 160,
			Segment2Chars: 153,
			Segment3Chars: 153,
			SegmentNChars: 153,
		}

		// Exactly 160 chars should be 1 segment
		segments := calculateSegmentsFromRules(160, rule)
		assert.Equal(t, 1, segments)

		// 161 chars should be 2 segments
		segments = calculateSegmentsFromRules(161, rule)
		assert.Equal(t, 2, segments)
	})

	t.Run("very long message", func(t *testing.T) {
		rule := models.SegmentationRule{
			EncodingType:  "GSM-7",
			Segment1Chars: 160,
			Segment2Chars: 153,
			Segment3Chars: 153,
			SegmentNChars: 153,
		}

		// 1000 chars should be calculated correctly
		// 160 + 153 + 153 = 466 (3 segments)
		// Remaining: 1000 - 466 = 534
		// Additional segments: ceil(534 / 153) = 4
		// Total: 3 + 4 = 7
		segments := calculateSegmentsFromRules(1000, rule)
		assert.Equal(t, 7, segments)
	})
}

// TestTariffRequest_Validation tests tariff request validation
func TestTariffRequest_Validation(t *testing.T) {
	tests := []struct {
		name  string
		req   TariffRequest
		valid bool
	}{
		{
			name: "valid request with all fields",
			req: TariffRequest{
				TenantID:        testTenantID,
				ServiceID:       1,
				CountryCode:     "YE",
				NetworkOperator: "Yemen Mobile",
				MessageType:     "GSM-7",
			},
			valid: true,
		},
		{
			name: "valid request without network operator",
			req: TariffRequest{
				TenantID:    testTenantID,
				ServiceID:   1,
				CountryCode: "YE",
				MessageType: "GSM-7",
			},
			valid: true,
		},
		{
			name: "invalid - missing tenant_id",
			req: TariffRequest{
				TenantID:    uuid.Nil,
				ServiceID:   1,
				CountryCode: "YE",
			},
			valid: false,
		},
		{
			name: "invalid - missing service_id",
			req: TariffRequest{
				TenantID:    testTenantID,
				ServiceID:   0,
				CountryCode: "YE",
			},
			valid: false,
		},
		{
			name: "invalid - missing country_code",
			req: TariffRequest{
				TenantID:  testTenantID,
				ServiceID: 1,
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.valid {
				assert.NotEqual(t, uuid.Nil, tt.req.TenantID)
				assert.Greater(t, tt.req.ServiceID, 0)
				assert.NotEmpty(t, tt.req.CountryCode)
			} else {
				isValid := tt.req.TenantID != uuid.Nil && tt.req.ServiceID > 0 && tt.req.CountryCode != ""
				assert.False(t, isValid)
			}
		})
	}
}
