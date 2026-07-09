package tariff

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/al-hudhud/backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Property 12: حساب أجزاء الرسالة بدقة
// **Validates: Requirements 7.6**
//
// لأي نص رسالة، يجب أن يحسب النظام عدد الأجزاء بناءً على قواعد التقسيم (segmentation_rules)
// المطابقة لنوع الترميز (encoding_type)
//
// This property test validates:
// 1. Correct segment calculation for GSM-7 encoding
// 2. Correct segment calculation for Unicode encoding
// 3. Segments never less than 1
// 4. Segments increase monotonically with message length
// 5. Segmentation rules are applied correctly (160/153 for GSM-7, 70/67 for Unicode)

func TestProperty_CorrectSegmentCalculation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	// Setup
	ctx := context.Background()
	db, pool, cache := setupTestEnvironment(t)
	defer cleanupTestEnvironment(pool, cache)

	logger, _ := zap.NewDevelopment()
	engine := NewEngine(db, cache, logger)

	// Create segmentation rules
	createSegmentationRules(t, db)

	// Test cases with known expected results
	testCases := []struct {
		name            string
		messageLength   int
		encodingType    string
		expectedSegments int
	}{
		// GSM-7 encoding
		{"GSM7_Single_Short", 50, "GSM-7", 1},
		{"GSM7_Single_Max", 160, "GSM-7", 1},
		{"GSM7_Two_Min", 161, "GSM-7", 2},
		{"GSM7_Two_Max", 313, "GSM-7", 2}, // 160 + 153
		{"GSM7_Three_Min", 314, "GSM-7", 3},
		{"GSM7_Three_Max", 466, "GSM-7", 3}, // 160 + 153 + 153
		{"GSM7_Four", 467, "GSM-7", 4},
		{"GSM7_Five", 620, "GSM-7", 5},

		// Unicode encoding
		{"Unicode_Single_Short", 30, "Unicode", 1},
		{"Unicode_Single_Max", 70, "Unicode", 1},
		{"Unicode_Two_Min", 71, "Unicode", 2},
		{"Unicode_Two_Max", 137, "Unicode", 2}, // 70 + 67
		{"Unicode_Three_Min", 138, "Unicode", 3},
		{"Unicode_Three_Max", 204, "Unicode", 3}, // 70 + 67 + 67
		{"Unicode_Four", 205, "Unicode", 4},
		{"Unicode_Five", 272, "Unicode", 5},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			segments, err := engine.CalculateSegments(ctx, tc.messageLength, tc.encodingType)
			require.NoError(t, err, "Segment calculation should succeed")

			// Property: Calculated segments must match expected
			assert.Equal(t, tc.expectedSegments, segments,
				"Property violated: Incorrect segment count\n"+
					"Message length: %d, Encoding: %s\n"+
					"Expected: %d segments, Got: %d segments",
				tc.messageLength, tc.encodingType, tc.expectedSegments, segments)

			t.Logf("%s: Length=%d, Encoding=%s, Segments=%d ✓",
				tc.name, tc.messageLength, tc.encodingType, segments)
		})
	}
}

// Property Test: Segments Never Less Than One
// Tests that segment count is always at least 1 for any positive message length
// **Validates: Requirements 7.6**
func TestProperty_SegmentsNeverLessThanOne(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	// Setup
	ctx := context.Background()
	db, pool, cache := setupTestEnvironment(t)
	defer cleanupTestEnvironment(pool, cache)

	logger, _ := zap.NewDevelopment()
	engine := NewEngine(db, cache, logger)

	createSegmentationRules(t, db)

	numIterations := 50
	encodings := []string{"GSM-7", "Unicode"}

	for iteration := 0; iteration < numIterations; iteration++ {
		encoding := encodings[rand.Intn(len(encodings))]
		messageLength := rand.Intn(1000) + 1 // 1 to 1000

		t.Run(fmt.Sprintf("Iteration_%d_%s_%d", iteration+1, encoding, messageLength), func(t *testing.T) {
			segments, err := engine.CalculateSegments(ctx, messageLength, encoding)
			require.NoError(t, err)

			// Property: Segments must be at least 1
			assert.GreaterOrEqual(t, segments, 1,
				"Property violated: Segments should never be less than 1\n"+
					"Message length: %d, Encoding: %s, Got: %d segments",
				messageLength, encoding, segments)

			t.Logf("Iteration %d: Length=%d, Encoding=%s, Segments=%d ✓",
				iteration+1, messageLength, encoding, segments)
		})
	}
}

// Property Test: Monotonic Increase
// Tests that segment count increases monotonically with message length
// **Validates: Requirements 7.6**
func TestProperty_SegmentsIncreaseMonotonically(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	// Setup
	ctx := context.Background()
	db, pool, cache := setupTestEnvironment(t)
	defer cleanupTestEnvironment(pool, cache)

	logger, _ := zap.NewDevelopment()
	engine := NewEngine(db, cache, logger)

	createSegmentationRules(t, db)

	encodings := []string{"GSM-7", "Unicode"}

	for _, encoding := range encodings {
		t.Run(fmt.Sprintf("Monotonic_%s", encoding), func(t *testing.T) {
			previousSegments := 0
			previousLength := 0

			// Test increasing message lengths
			for length := 1; length <= 1000; length += 10 {
				segments, err := engine.CalculateSegments(ctx, length, encoding)
				require.NoError(t, err)

				// Property: Segments should never decrease as length increases
				assert.GreaterOrEqual(t, segments, previousSegments,
					"Property violated: Segments should increase monotonically\n"+
						"Encoding: %s\n"+
						"Previous: length=%d, segments=%d\n"+
						"Current: length=%d, segments=%d",
					encoding, previousLength, previousSegments, length, segments)

				previousSegments = segments
				previousLength = length
			}

			t.Logf("Monotonic test for %s: Passed for lengths 1-1000", encoding)
		})
	}
}

// Property Test: GSM-7 Segmentation Rules
// Tests that GSM-7 segmentation follows the correct rules (160/153/153/...)
// **Validates: Requirements 7.1, 7.2, 7.3, 7.6**
func TestProperty_GSM7SegmentationRules(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	// Setup
	ctx := context.Background()
	db, pool, cache := setupTestEnvironment(t)
	defer cleanupTestEnvironment(pool, cache)

	logger, _ := zap.NewDevelopment()
	engine := NewEngine(db, cache, logger)

	createSegmentationRules(t, db)

	// Test boundary conditions for GSM-7
	boundaries := []struct {
		length           int
		expectedSegments int
		description      string
	}{
		{1, 1, "Minimum length"},
		{160, 1, "Single segment max"},
		{161, 2, "Two segments min"},
		{313, 2, "Two segments max (160+153)"},
		{314, 3, "Three segments min"},
		{466, 3, "Three segments max (160+153+153)"},
		{467, 4, "Four segments min"},
		{619, 4, "Four segments max (160+153+153+153)"},
		{620, 5, "Five segments min"},
	}

	for _, boundary := range boundaries {
		t.Run(fmt.Sprintf("GSM7_Boundary_%s", boundary.description), func(t *testing.T) {
			segments, err := engine.CalculateSegments(ctx, boundary.length, "GSM-7")
			require.NoError(t, err)

			// Property: Segments must match expected for boundary conditions
			assert.Equal(t, boundary.expectedSegments, segments,
				"Property violated: GSM-7 segmentation rule not followed\n"+
					"Length: %d (%s)\n"+
					"Expected: %d segments, Got: %d segments",
				boundary.length, boundary.description, boundary.expectedSegments, segments)

			t.Logf("GSM-7 boundary test: %s (length=%d) = %d segments ✓",
				boundary.description, boundary.length, segments)
		})
	}
}

// Property Test: Unicode Segmentation Rules
// Tests that Unicode segmentation follows the correct rules (70/67/67/...)
// **Validates: Requirements 7.1, 7.2, 7.3, 7.6**
func TestProperty_UnicodeSegmentationRules(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	// Setup
	ctx := context.Background()
	db, pool, cache := setupTestEnvironment(t)
	defer cleanupTestEnvironment(pool, cache)

	logger, _ := zap.NewDevelopment()
	engine := NewEngine(db, cache, logger)

	createSegmentationRules(t, db)

	// Test boundary conditions for Unicode
	boundaries := []struct {
		length           int
		expectedSegments int
		description      string
	}{
		{1, 1, "Minimum length"},
		{70, 1, "Single segment max"},
		{71, 2, "Two segments min"},
		{137, 2, "Two segments max (70+67)"},
		{138, 3, "Three segments min"},
		{204, 3, "Three segments max (70+67+67)"},
		{205, 4, "Four segments min"},
		{271, 4, "Four segments max (70+67+67+67)"},
		{272, 5, "Five segments min"},
	}

	for _, boundary := range boundaries {
		t.Run(fmt.Sprintf("Unicode_Boundary_%s", boundary.description), func(t *testing.T) {
			segments, err := engine.CalculateSegments(ctx, boundary.length, "Unicode")
			require.NoError(t, err)

			// Property: Segments must match expected for boundary conditions
			assert.Equal(t, boundary.expectedSegments, segments,
				"Property violated: Unicode segmentation rule not followed\n"+
					"Length: %d (%s)\n"+
					"Expected: %d segments, Got: %d segments",
				boundary.length, boundary.description, boundary.expectedSegments, segments)

			t.Logf("Unicode boundary test: %s (length=%d) = %d segments ✓",
				boundary.description, boundary.length, segments)
		})
	}
}

// Property Test: Random Message Lengths
// Stress test with random message lengths to ensure consistency
// **Validates: Requirements 7.6**
func TestProperty_RandomMessageLengths(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	// Setup
	ctx := context.Background()
	db, pool, cache := setupTestEnvironment(t)
	defer cleanupTestEnvironment(pool, cache)

	logger, _ := zap.NewDevelopment()
	engine := NewEngine(db, cache, logger)

	createSegmentationRules(t, db)

	numIterations := 100
	encodings := []string{"GSM-7", "Unicode"}

	for iteration := 0; iteration < numIterations; iteration++ {
		encoding := encodings[rand.Intn(len(encodings))]
		messageLength := rand.Intn(2000) + 1 // 1 to 2000

		t.Run(fmt.Sprintf("Random_Iteration_%d", iteration+1), func(t *testing.T) {
			segments, err := engine.CalculateSegments(ctx, messageLength, encoding)
			require.NoError(t, err)

			// Property: Segments must be positive
			assert.Greater(t, segments, 0,
				"Property violated: Segments must be positive\n"+
					"Length: %d, Encoding: %s, Got: %d segments",
				messageLength, encoding, segments)

			// Property: Verify calculation manually
			expectedSegments := calculateExpectedSegments(messageLength, encoding)
			assert.Equal(t, expectedSegments, segments,
				"Property violated: Segment calculation mismatch\n"+
					"Length: %d, Encoding: %s\n"+
					"Expected: %d, Got: %d",
				messageLength, encoding, expectedSegments, segments)

			t.Logf("Random test %d: Length=%d, Encoding=%s, Segments=%d ✓",
				iteration+1, messageLength, encoding, segments)
		})
	}
}

// Property Test: Default Encoding
// Tests that empty encoding defaults to GSM-7
// **Validates: Requirements 7.1**
func TestProperty_DefaultEncodingIsGSM7(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	// Setup
	ctx := context.Background()
	db, pool, cache := setupTestEnvironment(t)
	defer cleanupTestEnvironment(pool, cache)

	logger, _ := zap.NewDevelopment()
	engine := NewEngine(db, cache, logger)

	createSegmentationRules(t, db)

	testLengths := []int{50, 160, 161, 313, 314}

	for _, length := range testLengths {
		t.Run(fmt.Sprintf("DefaultEncoding_Length_%d", length), func(t *testing.T) {
			// Calculate with empty encoding (should default to GSM-7)
			segmentsDefault, err := engine.CalculateSegments(ctx, length, "")
			require.NoError(t, err)

			// Calculate with explicit GSM-7
			segmentsGSM7, err := engine.CalculateSegments(ctx, length, "GSM-7")
			require.NoError(t, err)

			// Property: Empty encoding should behave like GSM-7
			assert.Equal(t, segmentsGSM7, segmentsDefault,
				"Property violated: Empty encoding should default to GSM-7\n"+
					"Length: %d\n"+
					"GSM-7 segments: %d, Default segments: %d",
				length, segmentsGSM7, segmentsDefault)

			t.Logf("Default encoding test: Length=%d, Segments=%d (matches GSM-7) ✓",
				length, segmentsDefault)
		})
	}
}

// Property Test: Invalid Message Length
// Tests that invalid message lengths are handled correctly
// **Validates: Requirements 7.6**
func TestProperty_InvalidMessageLength(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	// Setup
	ctx := context.Background()
	db, pool, cache := setupTestEnvironment(t)
	defer cleanupTestEnvironment(pool, cache)

	logger, _ := zap.NewDevelopment()
	engine := NewEngine(db, cache, logger)

	createSegmentationRules(t, db)

	invalidLengths := []int{0, -1, -10, -100}

	for _, length := range invalidLengths {
		t.Run(fmt.Sprintf("InvalidLength_%d", length), func(t *testing.T) {
			_, err := engine.CalculateSegments(ctx, length, "GSM-7")

			// Property: Invalid lengths should return error
			assert.Error(t, err,
				"Property violated: Invalid message length should return error\n"+
					"Length: %d", length)

			assert.Contains(t, err.Error(), "invalid message_length",
				"Property violated: Error should indicate invalid message length")

			t.Logf("Invalid length test: Length=%d correctly rejected ✓", length)
		})
	}
}

// Helper functions

func createSegmentationRules(t *testing.T, db *gorm.DB) {
	// GSM-7 rule
	gsm7Rule := &models.SegmentationRule{
		EncodingType:  "GSM-7",
		Segment1Chars: 160,
		Segment2Chars: 153,
		Segment3Chars: 153,
		SegmentNChars: 153,
		IsActive:      true,
	}
	err := db.Create(gsm7Rule).Error
	require.NoError(t, err, "Failed to create GSM-7 segmentation rule")

	// Unicode rule
	unicodeRule := &models.SegmentationRule{
		EncodingType:  "Unicode",
		Segment1Chars: 70,
		Segment2Chars: 67,
		Segment3Chars: 67,
		SegmentNChars: 67,
		IsActive:      true,
	}
	err = db.Create(unicodeRule).Error
	require.NoError(t, err, "Failed to create Unicode segmentation rule")
}

// calculateExpectedSegments manually calculates expected segments for verification
func calculateExpectedSegments(messageLength int, encodingType string) int {
	if messageLength <= 0 {
		return 0
	}

	var segment1, segment2, segment3, segmentN int

	if encodingType == "Unicode" {
		segment1 = 70
		segment2 = 67
		segment3 = 67
		segmentN = 67
	} else {
		// Default to GSM-7
		segment1 = 160
		segment2 = 153
		segment3 = 153
		segmentN = 153
	}

	// Single segment
	if messageLength <= segment1 {
		return 1
	}

	// Two segments
	if messageLength <= segment1+segment2 {
		return 2
	}

	// Three segments
	if messageLength <= segment1+segment2+segment3 {
		return 3
	}

	// More than three segments
	remaining := messageLength - segment1 - segment2 - segment3
	additionalSegments := (remaining + segmentN - 1) / segmentN // Ceiling division

	return 3 + additionalSegments
}

// generateRandomMessage generates a random message of specified length
func generateRandomMessage(length int, encoding string) string {
	var charset string
	if encoding == "Unicode" {
		// Include Arabic characters for Unicode
		charset = "أبتثجحخدذرزسشصضطظعغفقكلمنهويابتثجحخدذرزسشصضطظعغفقكلمنهوي"
	} else {
		// GSM-7 characters
		charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789 "
	}

	var sb strings.Builder
	for i := 0; i < length; i++ {
		sb.WriteByte(charset[rand.Intn(len(charset))])
	}

	return sb.String()
}
