//go:build benchmark

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/al-hudhud/backend/internal/admin/distribution"
	"github.com/al-hudhud/backend/internal/admin/rating"
	"github.com/al-hudhud/backend/internal/admin/routing"
	"github.com/al-hudhud/backend/internal/models"
)

// BenchmarkRatingCalculation benchmarks rating calculation latency
// Target: < 10ms per calculation
func BenchmarkRatingCalculation(b *testing.B) {
	db := setupBenchmarkDB(b)
	defer cleanupBenchmarkDB(b, db)

	ctx := context.Background()
	ratingEngine := rating.NewRatingEngine(db)

	// Setup test tenant
	tenantID := uuid.New()
	tenant := &models.Tenant{
		ID:          tenantID,
		CompanyName: "Benchmark Company",
		Email:       "benchmark@example.com",
		PhoneNumber: "+1234567890",
		Balance:     1000.0,
		IsActive:    true,
	}
	require.NoError(b, db.Create(tenant).Error)

	// Prepare test metrics
	metrics := &rating.PerformanceMetrics{
		TenantID:           tenantID,
		TotalMessages:      1000,
		SuccessfulMessages: 900,
		FailedMessages:     100,
		TotalPayments:      10,
		OnTimePayments:     9,
		AverageDeliveryTime: 150,
	}

	// Reset timer before benchmark
	b.ResetTimer()

	// Run benchmark
	for i := 0; i < b.N; i++ {
		_, err := ratingEngine.CalculateRating(ctx, metrics)
		if err != nil {
			b.Fatal(err)
		}
	}

	// Report results
	b.StopTimer()
	avgDuration := time.Duration(b.Elapsed().Nanoseconds() / int64(b.N))
	b.Logf("Average rating calculation time: %v", avgDuration)

	// Verify target: < 10ms
	if avgDuration > 10*time.Millisecond {
		b.Errorf("Rating calculation too slow: %v (target: < 10ms)", avgDuration)
	}
}

// BenchmarkRoutingDecision benchmarks routing decision latency
// Target: < 50ms at p95
func BenchmarkRoutingDecision(b *testing.B) {
	db := setupBenchmarkDB(b)
	defer cleanupBenchmarkDB(b, db)

	ctx := context.Background()
	routingEngine := routing.NewRoutingEngine(db)

	// Setup test tenant with dedicated server
	tenantID := uuid.New()
	tenant := &models.Tenant{
		ID:          tenantID,
		CompanyName: "Routing Benchmark Company",
		Email:       "routing-bench@example.com",
		PhoneNumber: "+1234567891",
		Balance:     1000.0,
		IsActive:    true,
	}
	require.NoError(b, db.Create(tenant).Error)

	routingConfig := &models.RoutingConfiguration{
		TenantID:             tenantID,
		RoutingType:          "dedicated",
		DedicatedServerURL:   "http://test-server.local",
		DedicatedServerAPIKey: "test-key",
		HealthStatus:         "healthy",
	}
	require.NoError(b, db.Create(routingConfig).Error)

	// Prepare test message
	message := &routing.Message{
		ID:       uuid.New(),
		TenantID: tenantID,
		Content:  "Benchmark message",
	}

	// Collect latencies for p95 calculation
	latencies := make([]time.Duration, b.N)

	// Reset timer before benchmark
	b.ResetTimer()

	// Run benchmark
	for i := 0; i < b.N; i++ {
		start := time.Now()
		err := routingEngine.RouteMessage(ctx, message)
		latencies[i] = time.Since(start)
		
		if err != nil {
			b.Fatal(err)
		}
	}

	b.StopTimer()

	// Calculate p95 latency
	p95Index := int(float64(b.N) * 0.95)
	if p95Index >= len(latencies) {
		p95Index = len(latencies) - 1
	}
	
	// Sort latencies to find p95
	sortedLatencies := make([]time.Duration, len(latencies))
	copy(sortedLatencies, latencies)
	// Simple bubble sort for small datasets
	for i := 0; i < len(sortedLatencies); i++ {
		for j := i + 1; j < len(sortedLatencies); j++ {
			if sortedLatencies[i] > sortedLatencies[j] {
				sortedLatencies[i], sortedLatencies[j] = sortedLatencies[j], sortedLatencies[i]
			}
		}
	}
	
	p95Latency := sortedLatencies[p95Index]
	avgLatency := time.Duration(b.Elapsed().Nanoseconds() / int64(b.N))

	b.Logf("Average routing decision time: %v", avgLatency)
	b.Logf("P95 routing decision time: %v", p95Latency)

	// Verify target: p95 < 50ms
	if p95Latency > 50*time.Millisecond {
		b.Errorf("P95 routing decision too slow: %v (target: < 50ms)", p95Latency)
	}
}

// BenchmarkDistributionSelection benchmarks device selection latency
// Target: < 5ms per selection
func BenchmarkDistributionSelection(b *testing.B) {
	db := setupBenchmarkDB(b)
	defer cleanupBenchmarkDB(b, db)

	ctx := context.Background()
	distManager := distribution.NewDistributionManager(db)

	// Setup test tenant
	tenantID := uuid.New()
	tenant := &models.Tenant{
		ID:          tenantID,
		CompanyName: "Distribution Benchmark Company",
		Email:       "dist-bench@example.com",
		PhoneNumber: "+1234567892",
		Balance:     1000.0,
		IsActive:    true,
	}
	require.NoError(b, db.Create(tenant).Error)

	// Create multiple devices
	for i := 0; i < 10; i++ {
		device := &models.PhoneDevice{
			ID:         uuid.New(),
			TenantID:   tenantID,
			DeviceName: "Device " + string(rune('A'+i)),
			Status:     "active",
		}
		require.NoError(b, db.Create(device).Error)
	}

	// Create distribution strategy
	strategy := &models.DistributionStrategy{
		TenantID:     tenantID,
		StrategyType: "round_robin",
	}
	require.NoError(b, db.Create(strategy).Error)

	// Prepare test message
	message := &distribution.Message{
		ID:       uuid.New(),
		TenantID: tenantID,
		Content:  "Benchmark message",
	}

	// Reset timer before benchmark
	b.ResetTimer()

	// Run benchmark
	for i := 0; i < b.N; i++ {
		_, err := distManager.SelectDevice(ctx, tenantID, message)
		if err != nil {
			b.Fatal(err)
		}
	}

	b.StopTimer()

	avgDuration := time.Duration(b.Elapsed().Nanoseconds() / int64(b.N))
	b.Logf("Average device selection time: %v", avgDuration)

	// Verify target: < 5ms
	if avgDuration > 5*time.Millisecond {
		b.Errorf("Device selection too slow: %v (target: < 5ms)", avgDuration)
	}
}

// BenchmarkAPIResponseTime benchmarks API handler response time
// Target: < 200ms average
func BenchmarkAPIResponseTime(b *testing.B) {
	db := setupBenchmarkDB(b)
	defer cleanupBenchmarkDB(b, db)

	ctx := context.Background()

	// Setup test data
	tenantID := uuid.New()
	tenant := &models.Tenant{
		ID:          tenantID,
		CompanyName: "API Benchmark Company",
		Email:       "api-bench@example.com",
		PhoneNumber: "+1234567893",
		Balance:     1000.0,
		IsActive:    true,
	}
	require.NoError(b, db.Create(tenant).Error)

	// Create rating data
	companyRating := &models.CompanyRating{
		TenantID:          tenantID,
		SuccessRate:       95.0,
		SendingVolume:     1000,
		PaymentCompliance: 100.0,
		ServiceQualityScore: 90.0,
		OverallRating:     93.0,
		RatingCategory:    "excellent",
		CalculationTimestamp: time.Now(),
	}
	require.NoError(b, db.Create(companyRating).Error)

	// Reset timer before benchmark
	b.ResetTimer()

	// Benchmark: Fetch company rating (simulating API call)
	for i := 0; i < b.N; i++ {
		var rating models.CompanyRating
		err := db.Where("tenant_id = ?", tenantID).First(&rating).Error
		if err != nil {
			b.Fatal(err)
		}
	}

	b.StopTimer()

	avgDuration := time.Duration(b.Elapsed().Nanoseconds() / int64(b.N))
	b.Logf("Average API response time: %v", avgDuration)

	// Verify target: < 200ms (database query should be much faster)
	if avgDuration > 200*time.Millisecond {
		b.Errorf("API response too slow: %v (target: < 200ms)", avgDuration)
	}
}

// BenchmarkConcurrentRatingCalculations benchmarks concurrent rating calculations
func BenchmarkConcurrentRatingCalculations(b *testing.B) {
	db := setupBenchmarkDB(b)
	defer cleanupBenchmarkDB(b, db)

	ctx := context.Background()
	ratingEngine := rating.NewRatingEngine(db)

	// Setup multiple test tenants
	numTenants := 100
	tenantIDs := make([]uuid.UUID, numTenants)

	for i := 0; i < numTenants; i++ {
		tenantID := uuid.New()
		tenantIDs[i] = tenantID

		tenant := &models.Tenant{
			ID:          tenantID,
			CompanyName: "Concurrent Tenant " + string(rune(i)),
			Email:       "concurrent" + string(rune(i)) + "@example.com",
			PhoneNumber: "+123456789" + string(rune('0'+i%10)),
			Balance:     1000.0,
			IsActive:    true,
		}
		require.NoError(b, db.Create(tenant).Error)
	}

	// Reset timer before benchmark
	b.ResetTimer()

	// Run benchmark with concurrency
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Select random tenant
			tenantID := tenantIDs[b.N%numTenants]

			metrics := &rating.PerformanceMetrics{
				TenantID:           tenantID,
				TotalMessages:      1000,
				SuccessfulMessages: 900,
				FailedMessages:     100,
				TotalPayments:      10,
				OnTimePayments:     9,
			}

			_, err := ratingEngine.CalculateRating(ctx, metrics)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.StopTimer()

	avgDuration := time.Duration(b.Elapsed().Nanoseconds() / int64(b.N))
	b.Logf("Average concurrent rating calculation time: %v", avgDuration)
}

// BenchmarkDatabaseQueryPerformance benchmarks database query performance
func BenchmarkDatabaseQueryPerformance(b *testing.B) {
	db := setupBenchmarkDB(b)
	defer cleanupBenchmarkDB(b, db)

	// Setup test data - create 1000 company ratings
	for i := 0; i < 1000; i++ {
		tenantID := uuid.New()
		
		tenant := &models.Tenant{
			ID:          tenantID,
			CompanyName: "Query Test Company " + string(rune(i)),
			Email:       "query" + string(rune(i)) + "@example.com",
			PhoneNumber: "+123456789" + string(rune('0'+i%10)),
			Balance:     1000.0,
			IsActive:    true,
		}
		require.NoError(b, db.Create(tenant).Error)

		rating := &models.CompanyRating{
			TenantID:          tenantID,
			SuccessRate:       float64(80 + i%20),
			SendingVolume:     int64(1000 + i*10),
			PaymentCompliance: float64(90 + i%10),
			ServiceQualityScore: float64(85 + i%15),
			OverallRating:     float64(85 + i%15),
			RatingCategory:    "good",
			CalculationTimestamp: time.Now(),
		}
		require.NoError(b, db.Create(rating).Error)
	}

	// Reset timer before benchmark
	b.ResetTimer()

	// Benchmark: Query with filtering and sorting
	for i := 0; i < b.N; i++ {
		var ratings []models.CompanyRating
		err := db.Where("rating_category = ?", "good").
			Order("overall_rating DESC").
			Limit(50).
			Find(&ratings).Error
		if err != nil {
			b.Fatal(err)
		}
	}

	b.StopTimer()

	avgDuration := time.Duration(b.Elapsed().Nanoseconds() / int64(b.N))
	b.Logf("Average database query time (1000 records): %v", avgDuration)

	// Verify reasonable performance
	if avgDuration > 100*time.Millisecond {
		b.Errorf("Database query too slow: %v (target: < 100ms)", avgDuration)
	}
}

// setupBenchmarkDB creates a test database connection for benchmarks
func setupBenchmarkDB(b *testing.B) *gorm.DB {
	// This should use the actual database connection from your config
	// For now, skipping - implement based on your setup
	b.Skip("Database setup not implemented - implement based on your configuration")
	return nil
}

// cleanupBenchmarkDB cleans up benchmark test data
func cleanupBenchmarkDB(b *testing.B, db *gorm.DB) {
	if db != nil {
		// Clean up test data
		db.Exec("DELETE FROM company_ratings WHERE tenant_id IN (SELECT id FROM tenants WHERE email LIKE '%bench%@example.com')")
		db.Exec("DELETE FROM tenants WHERE email LIKE '%bench%@example.com'")
	}
}
