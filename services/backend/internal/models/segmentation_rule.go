package models

import "time"

// SegmentationRule represents SMS message segmentation rules
// Requirements: 7.1, 7.2, 7.3, 7.4
type SegmentationRule struct {
	ID               int       `json:"id" gorm:"primaryKey"`
	GatewayID        *int      `json:"gateway_id,omitempty" gorm:"index"` // NULL = default rules
	CountryCode      *string   `json:"country_code,omitempty" gorm:"type:varchar(2);index"`
	NetworkOperator  *string   `json:"network_operator,omitempty" gorm:"type:varchar(100)"`
	EncodingType     string    `json:"encoding_type" gorm:"type:varchar(20);not null;index"` // 'gsm7', 'unicode'
	Segment1Chars    int       `json:"segment_1_chars" gorm:"not null"`                       // 70 for first segment
	Segment2Chars    int       `json:"segment_2_chars" gorm:"not null"`                       // 65 for second segment
	Segment3Chars    int       `json:"segment_3_chars" gorm:"not null"`                       // 62 for third segment
	SegmentNChars    int       `json:"segment_n_chars" gorm:"not null"`                       // for subsequent segments
	HeaderOverhead   int       `json:"header_overhead" gorm:"default:0"`                      // characters used for header
	Description      *string   `json:"description,omitempty" gorm:"type:text"`
	IsActive         bool      `json:"is_active" gorm:"default:true;index"`
	CreatedAt        time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt        time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName specifies the table name for GORM
func (SegmentationRule) TableName() string {
	return "segmentation_rules"
}

// EncodingType constants
const (
	EncodingTypeGSM7    = "gsm7"
	EncodingTypeUnicode = "unicode"
)

// DefaultSegmentationRules returns the default segmentation rules
func DefaultSegmentationRules() []SegmentationRule {
	gsm7Desc := "Default GSM-7 segmentation rules"
	unicodeDesc := "Default Unicode segmentation rules"

	return []SegmentationRule{
		{
			EncodingType:   EncodingTypeGSM7,
			Segment1Chars:  70,
			Segment2Chars:  65,
			Segment3Chars:  62,
			SegmentNChars:  62,
			HeaderOverhead: 0,
			Description:    &gsm7Desc,
			IsActive:       true,
		},
		{
			EncodingType:   EncodingTypeUnicode,
			Segment1Chars:  70,
			Segment2Chars:  65,
			Segment3Chars:  62,
			SegmentNChars:  62,
			HeaderOverhead: 0,
			Description:    &unicodeDesc,
			IsActive:       true,
		},
	}
}

// CalculateSegments calculates the number of segments needed for a message
// Requirements: 7.6
func (sr *SegmentationRule) CalculateSegments(messageLength int) int {
	if messageLength <= 0 {
		return 0
	}

	// Adjust for header overhead
	effectiveLength := messageLength + sr.HeaderOverhead

	// First segment
	if effectiveLength <= sr.Segment1Chars {
		return 1
	}

	// Second segment
	remainingAfterFirst := effectiveLength - sr.Segment1Chars
	if remainingAfterFirst <= sr.Segment2Chars {
		return 2
	}

	// Third segment
	remainingAfterSecond := remainingAfterFirst - sr.Segment2Chars
	if remainingAfterSecond <= sr.Segment3Chars {
		return 3
	}

	// Subsequent segments
	remainingAfterThird := remainingAfterSecond - sr.Segment3Chars
	additionalSegments := (remainingAfterThird + sr.SegmentNChars - 1) / sr.SegmentNChars

	return 3 + additionalSegments
}

// Validate checks if the segmentation rule is valid
func (sr *SegmentationRule) Validate() error {
	if sr.Segment1Chars <= 0 {
		return ErrInvalidSegmentSize
	}
	if sr.Segment2Chars <= 0 {
		return ErrInvalidSegmentSize
	}
	if sr.Segment3Chars <= 0 {
		return ErrInvalidSegmentSize
	}
	if sr.SegmentNChars <= 0 {
		return ErrInvalidSegmentSize
	}
	if sr.HeaderOverhead < 0 {
		return ErrInvalidHeaderOverhead
	}
	if sr.EncodingType != EncodingTypeGSM7 && sr.EncodingType != EncodingTypeUnicode {
		return ErrInvalidEncodingType
	}
	return nil
}


// Custom errors for segmentation rule validation
var (
	ErrInvalidSegmentSize     = &ValidationError{Field: "segment_chars", Message: "segment size must be positive"}
	ErrInvalidHeaderOverhead  = &ValidationError{Field: "header_overhead", Message: "header overhead must be non-negative"}
	ErrInvalidEncodingType    = &ValidationError{Field: "encoding_type", Message: "encoding type must be 'gsm7' or 'unicode'"}
	ErrSegmentationRuleNotFound = &NotFoundError{Resource: "segmentation_rule"}
)
