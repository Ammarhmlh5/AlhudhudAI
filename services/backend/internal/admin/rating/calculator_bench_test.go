package rating

import (
	"testing"
)

// BenchmarkRatingCalculation benchmarks the rating calculation performance
// Target: < 10ms per calculation
func BenchmarkRatingCalculation(b *testing.B) {
	// Setup test data
	metrics := &PerformanceMetrics{
		TotalMessages:      10000,
		SuccessfulMessages: 9500,
		FailedMessages:     500,
		TotalPayments:      10,
		OnTimePayments:     9,
	}

	weights := RatingWeights{
		SuccessRate:       0.40,
		SendingVolume:     0.20,
		PaymentCompliance: 0.25,
		ServiceQuality:    0.15,
	}

	calculator, err := NewCalculator(weights)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = calculator.CalculateRating(metrics)
	}
}

// BenchmarkRatingCalculationParallel benchmarks parallel rating calculations
func BenchmarkRatingCalculationParallel(b *testing.B) {
	metrics := &PerformanceMetrics{
		TotalMessages:      10000,
		SuccessfulMessages: 9500,
		FailedMessages:     500,
		TotalPayments:      10,
		OnTimePayments:     9,
	}

	weights := RatingWeights{
		SuccessRate:       0.40,
		SendingVolume:     0.20,
		PaymentCompliance: 0.25,
		ServiceQuality:    0.15,
	}

	calculator, err := NewCalculator(weights)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = calculator.CalculateRating(metrics)
		}
	})
}

// BenchmarkRatingClassification benchmarks rating classification
func BenchmarkRatingClassification(b *testing.B) {
	calculator := NewDefaultCalculator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = calculator.ClassifyRating(85.5)
	}
}

// BenchmarkDerivedMetricsCalculation benchmarks derived metrics calculation
func BenchmarkDerivedMetricsCalculation(b *testing.B) {
	collector := &Collector{}
	metrics := &PerformanceMetrics{
		TotalMessages:      10000,
		SuccessfulMessages: 9500,
		FailedMessages:     500,
		TotalPayments:      10,
		OnTimePayments:     9,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.calculateDerivedMetrics(metrics)
	}
}
