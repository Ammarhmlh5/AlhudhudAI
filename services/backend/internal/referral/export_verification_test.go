package referral

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExportFunctions_JSON verifies that exportToJSON works correctly
func TestExportFunctions_JSON(t *testing.T) {
	config := &ReferralConfig{
		Enabled: true,
	}
	manager := NewManager(nil, config)
	
	report := createSampleReport()
	ctx := context.Background()
	
	// Export to JSON
	jsonData, err := manager.ExportReferralReport(ctx, report, "json")
	require.NoError(t, err, "JSON export should not return error")
	require.NotNil(t, jsonData, "JSON data should not be nil")
	
	// Verify it's valid JSON
	var decoded ReferralReport
	err = json.Unmarshal(jsonData, &decoded)
	require.NoError(t, err, "JSON should be valid and parseable")
	
	// Verify data integrity
	assert.Equal(t, report.Period, decoded.Period)
	assert.Equal(t, report.Performance.TotalReferrals, decoded.Performance.TotalReferrals)
	
	t.Logf("✓ JSON export successful, size: %d bytes", len(jsonData))
}

// TestExportFunctions_CSV verifies that exportToCSV works correctly
func TestExportFunctions_CSV(t *testing.T) {
	config := &ReferralConfig{
		Enabled: true,
	}
	manager := NewManager(nil, config)
	
	report := createSampleReport()
	ctx := context.Background()
	
	// Export to CSV
	csvData, err := manager.ExportReferralReport(ctx, report, "csv")
	require.NoError(t, err, "CSV export should not return error")
	require.NotNil(t, csvData, "CSV data should not be nil")
	
	// Verify it's valid CSV (contains headers and data)
	csvString := string(csvData)
	assert.Contains(t, csvString, "Period", "CSV should contain header")
	assert.Contains(t, csvString, "Metric", "CSV should contain metric column")
	assert.Contains(t, csvString, "Value", "CSV should contain value column")
	assert.Contains(t, csvString, "Total Referrals", "CSV should contain performance metrics")
	
	// Verify it has multiple lines
	lines := strings.Split(csvString, "\n")
	assert.Greater(t, len(lines), 5, "CSV should have multiple lines")
	
	t.Logf("✓ CSV export successful, size: %d bytes, lines: %d", len(csvData), len(lines))
}

// TestExportFunctions_PDF verifies that exportToPDF works correctly
func TestExportFunctions_PDF(t *testing.T) {
	config := &ReferralConfig{
		Enabled: true,
	}
	manager := NewManager(nil, config)
	
	report := createSampleReport()
	ctx := context.Background()
	
	// Export to PDF
	pdfData, err := manager.ExportReferralReport(ctx, report, "pdf")
	require.NoError(t, err, "PDF export should not return error")
	require.NotNil(t, pdfData, "PDF data should not be nil")
	
	// Verify it's a PDF (starts with PDF magic bytes)
	assert.True(t, len(pdfData) > 4, "PDF should have content")
	assert.Equal(t, "%PDF", string(pdfData[:4]), "PDF should start with PDF magic bytes")
	
	t.Logf("✓ PDF export successful, size: %d bytes", len(pdfData))
}

// TestExportFunctions_Performance verifies export performance meets requirements
func TestExportFunctions_Performance(t *testing.T) {
	config := &ReferralConfig{
		Enabled: true,
	}
	manager := NewManager(nil, config)
	
	// Create report with < 1000 records (as per requirement)
	report := createLargeReport(500)
	ctx := context.Background()
	
	// Test JSON performance
	start := time.Now()
	_, err := manager.ExportReferralReport(ctx, report, "json")
	jsonDuration := time.Since(start)
	require.NoError(t, err)
	assert.Less(t, jsonDuration, 2*time.Second, "JSON export should complete in < 2 seconds for < 1000 records")
	
	// Test CSV performance
	start = time.Now()
	_, err = manager.ExportReferralReport(ctx, report, "csv")
	csvDuration := time.Since(start)
	require.NoError(t, err)
	assert.Less(t, csvDuration, 2*time.Second, "CSV export should complete in < 2 seconds for < 1000 records")
	
	// Test PDF performance
	start = time.Now()
	_, err = manager.ExportReferralReport(ctx, report, "pdf")
	pdfDuration := time.Since(start)
	require.NoError(t, err)
	assert.Less(t, pdfDuration, 2*time.Second, "PDF export should complete in < 2 seconds for < 1000 records")
	
	t.Logf("✓ Performance requirements met:")
	t.Logf("  JSON: %v", jsonDuration)
	t.Logf("  CSV: %v", csvDuration)
	t.Logf("  PDF: %v", pdfDuration)
}

// Helper function to create a sample report
func createSampleReport() *ReferralReport {
	return &ReferralReport{
		Period:    "2024-01-01 to 2024-01-31",
		StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC),
		Performance: &ReferralPerformance{
			TotalReferrals:      100,
			CompletedReferrals:  80,
			RewardedReferrals:   75,
			PendingReferrals:    15,
			CancelledReferrals:  5,
			ConversionRate:      80.0,
			RewardRate:          93.75,
			TotalRewardsGranted: 1500.0,
			AverageReward:       20.0,
		},
		TopReferrers: []*TopReferrer{
			{
				TenantID:          uuid.New(),
				TenantName:        "Test Tenant 1",
				TotalReferrals:    25,
				RewardedReferrals: 20,
				TotalRewards:      400.0,
				ConversionRate:    80.0,
			},
			{
				TenantID:          uuid.New(),
				TenantName:        "Test Tenant 2",
				TotalReferrals:    20,
				RewardedReferrals: 18,
				TotalRewards:      360.0,
				ConversionRate:    90.0,
			},
		},
	}
}

// Helper function to create a large report for performance testing
func createLargeReport(numReferrers int) *ReferralReport {
	report := createSampleReport()
	report.TopReferrers = make([]*TopReferrer, numReferrers)
	
	for i := 0; i < numReferrers; i++ {
		report.TopReferrers[i] = &TopReferrer{
			TenantID:          uuid.New(),
			TenantName:        "Test Tenant " + string(rune(i)),
			TotalReferrals:    10 + i,
			RewardedReferrals: 8 + i,
			TotalRewards:      float64(160 + i*20),
			ConversionRate:    80.0 + float64(i%20),
		}
	}
	
	return report
}
