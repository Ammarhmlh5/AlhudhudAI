package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSegmentationRule_CalculateSegments(t *testing.T) {
	tests := []struct {
		name          string
		rule          SegmentationRule
		messageLength int
		expected      int
	}{
		{
			name: "Empty message",
			rule: SegmentationRule{
				Segment1Chars: 70,
				Segment2Chars: 65,
				Segment3Chars: 62,
				SegmentNChars: 62,
			},
			messageLength: 0,
			expected:      0,
		},
		{
			name: "Single segment - GSM7",
			rule: SegmentationRule{
				Segment1Chars: 70,
				Segment2Chars: 65,
				Segment3Chars: 62,
				SegmentNChars: 62,
			},
			messageLength: 50,
			expected:      1,
		},
		{
			name: "Exactly first segment - GSM7",
			rule: SegmentationRule{
				Segment1Chars: 70,
				Segment2Chars: 65,
				Segment3Chars: 62,
				SegmentNChars: 62,
			},
			messageLength: 70,
			expected:      1,
		},
		{
			name: "Two segments - GSM7",
			rule: SegmentationRule{
				Segment1Chars: 70,
				Segment2Chars: 65,
				Segment3Chars: 62,
				SegmentNChars: 62,
			},
			messageLength: 100,
			expected:      2,
		},
		{
			name: "Exactly two segments - GSM7",
			rule: SegmentationRule{
				Segment1Chars: 70,
				Segment2Chars: 65,
				Segment3Chars: 62,
				SegmentNChars: 62,
			},
			messageLength: 135, // 70 + 65
			expected:      2,
		},
		{
			name: "Three segments - GSM7",
			rule: SegmentationRule{
				Segment1Chars: 70,
				Segment2Chars: 65,
				Segment3Chars: 62,
				SegmentNChars: 62,
			},
			messageLength: 150,
			expected:      3,
		},
		{
			name: "Exactly three segments - GSM7",
			rule: SegmentationRule{
				Segment1Chars: 70,
				Segment2Chars: 65,
				Segment3Chars: 62,
				SegmentNChars: 62,
			},
			messageLength: 197, // 70 + 65 + 62
			expected:      3,
		},
		{
			name: "Four segments - GSM7",
			rule: SegmentationRule{
				Segment1Chars: 70,
				Segment2Chars: 65,
				Segment3Chars: 62,
				SegmentNChars: 62,
			},
			messageLength: 250,
			expected:      4,
		},
		{
			name: "Five segments - GSM7",
			rule: SegmentationRule{
				Segment1Chars: 70,
				Segment2Chars: 65,
				Segment3Chars: 62,
				SegmentNChars: 62,
			},
			messageLength: 321, // 70 + 65 + 62 + 62 + 62
			expected:      5,
		},
		{
			name: "With header overhead",
			rule: SegmentationRule{
				Segment1Chars:  70,
				Segment2Chars:  65,
				Segment3Chars:  62,
				SegmentNChars:  62,
				HeaderOverhead: 5,
			},
			messageLength: 66, // 66 + 5 = 71, needs 2 segments
			expected:      2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.rule.CalculateSegments(tt.messageLength)
			assert.Equal(t, tt.expected, result, "Segment calculation mismatch")
		})
	}
}

func TestSegmentationRule_Validate(t *testing.T) {
	tests := []struct {
		name      string
		rule      SegmentationRule
		expectErr bool
		errType   error
	}{
		{
			name: "Valid GSM7 rule",
			rule: SegmentationRule{
				EncodingType:  EncodingTypeGSM7,
				Segment1Chars: 70,
				Segment2Chars: 65,
				Segment3Chars: 62,
				SegmentNChars: 62,
			},
			expectErr: false,
		},
		{
			name: "Valid Unicode rule",
			rule: SegmentationRule{
				EncodingType:  EncodingTypeUnicode,
				Segment1Chars: 70,
				Segment2Chars: 65,
				Segment3Chars: 62,
				SegmentNChars: 62,
			},
			expectErr: false,
		},
		{
			name: "Invalid segment 1 size",
			rule: SegmentationRule{
				EncodingType:  EncodingTypeGSM7,
				Segment1Chars: 0,
				Segment2Chars: 65,
				Segment3Chars: 62,
				SegmentNChars: 62,
			},
			expectErr: true,
			errType:   ErrInvalidSegmentSize,
		},
		{
			name: "Invalid segment 2 size",
			rule: SegmentationRule{
				EncodingType:  EncodingTypeGSM7,
				Segment1Chars: 70,
				Segment2Chars: -1,
				Segment3Chars: 62,
				SegmentNChars: 62,
			},
			expectErr: true,
			errType:   ErrInvalidSegmentSize,
		},
		{
			name: "Invalid segment 3 size",
			rule: SegmentationRule{
				EncodingType:  EncodingTypeGSM7,
				Segment1Chars: 70,
				Segment2Chars: 65,
				Segment3Chars: 0,
				SegmentNChars: 62,
			},
			expectErr: true,
			errType:   ErrInvalidSegmentSize,
		},
		{
			name: "Invalid segment N size",
			rule: SegmentationRule{
				EncodingType:  EncodingTypeGSM7,
				Segment1Chars: 70,
				Segment2Chars: 65,
				Segment3Chars: 62,
				SegmentNChars: 0,
			},
			expectErr: true,
			errType:   ErrInvalidSegmentSize,
		},
		{
			name: "Invalid header overhead",
			rule: SegmentationRule{
				EncodingType:   EncodingTypeGSM7,
				Segment1Chars:  70,
				Segment2Chars:  65,
				Segment3Chars:  62,
				SegmentNChars:  62,
				HeaderOverhead: -1,
			},
			expectErr: true,
			errType:   ErrInvalidHeaderOverhead,
		},
		{
			name: "Invalid encoding type",
			rule: SegmentationRule{
				EncodingType:  "invalid",
				Segment1Chars: 70,
				Segment2Chars: 65,
				Segment3Chars: 62,
				SegmentNChars: 62,
			},
			expectErr: true,
			errType:   ErrInvalidEncodingType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rule.Validate()
			if tt.expectErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.Equal(t, tt.errType, err)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDefaultSegmentationRules(t *testing.T) {
	rules := DefaultSegmentationRules()

	assert.Len(t, rules, 2, "Should have 2 default rules")

	// Check GSM7 rule
	gsm7Rule := rules[0]
	assert.Equal(t, EncodingTypeGSM7, gsm7Rule.EncodingType)
	assert.Equal(t, 70, gsm7Rule.Segment1Chars)
	assert.Equal(t, 65, gsm7Rule.Segment2Chars)
	assert.Equal(t, 62, gsm7Rule.Segment3Chars)
	assert.Equal(t, 62, gsm7Rule.SegmentNChars)
	assert.True(t, gsm7Rule.IsActive)

	// Check Unicode rule
	unicodeRule := rules[1]
	assert.Equal(t, EncodingTypeUnicode, unicodeRule.EncodingType)
	assert.Equal(t, 70, unicodeRule.Segment1Chars)
	assert.Equal(t, 65, unicodeRule.Segment2Chars)
	assert.Equal(t, 62, unicodeRule.Segment3Chars)
	assert.Equal(t, 62, unicodeRule.SegmentNChars)
	assert.True(t, unicodeRule.IsActive)
}

func TestSegmentationRule_TableName(t *testing.T) {
	rule := SegmentationRule{}
	assert.Equal(t, "segmentation_rules", rule.TableName())
}
