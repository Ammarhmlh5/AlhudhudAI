package rating

import (
	"fmt"
	"math"
)

// Calculator handles rating calculations for companies
type Calculator struct {
	weights RatingWeights
}

// NewCalculator creates a new Calculator with the given weights
func NewCalculator(weights RatingWeights) (*Calculator, error) {
	if !weights.Validate() {
		return nil, fmt.Errorf("invalid weights: sum must equal 1.0")
	}

	return &Calculator{
		weights: weights,
	}, nil
}

// NewDefaultCalculator creates a new Calculator with default weights
func NewDefaultCalculator() *Calculator {
	return &Calculator{
		weights: DefaultRatingWeights(),
	}
}

// CalculateRating calculates the overall rating based on performance metrics
// Returns a value between 0 and 100
func (c *Calculator) CalculateRating(metrics *PerformanceMetrics) float64 {
	rating := 0.0

	// Success Rate (0-100)
	successRate := 0.0
	if metrics.TotalMessages > 0 {
		successRate = (float64(metrics.SuccessfulMessages) / float64(metrics.TotalMessages)) * 100
	}
	rating += successRate * c.weights.SuccessRate

	// Sending Volume (normalized to 0-100)
	volumeScore := c.NormalizeVolume(metrics.TotalMessages)
	rating += volumeScore * c.weights.SendingVolume

	// Payment Compliance (0-100)
	paymentScore := 0.0
	if metrics.TotalPayments > 0 {
		paymentScore = (float64(metrics.OnTimePayments) / float64(metrics.TotalPayments)) * 100
	}
	rating += paymentScore * c.weights.PaymentCompliance

	// Service Quality (0-100)
	qualityScore := c.CalculateQualityScore(metrics)
	rating += qualityScore * c.weights.ServiceQuality

	// Ensure rating is within bounds
	if rating < 0 {
		rating = 0
	}
	if rating > 100 {
		rating = 100
	}

	return rating
}

// ClassifyRating classifies a rating value into a category
func (c *Calculator) ClassifyRating(rating float64) RatingCategory {
	switch {
	case rating >= 90:
		return Excellent
	case rating >= 75:
		return Good
	case rating >= 60:
		return Average
	default:
		return Poor
	}
}

// NormalizeVolume normalizes sending volume to a 0-100 scale
// Uses logarithmic scaling to handle wide range of volumes
func (c *Calculator) NormalizeVolume(volume int64) float64 {
	if volume <= 0 {
		return 0
	}

	// Define volume thresholds
	const (
		minVolume = 100      // Minimum expected volume
		maxVolume = 10000000 // Maximum expected volume (10M)
	)

	// Use logarithmic scale for better distribution
	logVolume := math.Log10(float64(volume))
	logMin := math.Log10(float64(minVolume))
	logMax := math.Log10(float64(maxVolume))

	// Normalize to 0-100
	normalized := ((logVolume - logMin) / (logMax - logMin)) * 100

	// Clamp to 0-100 range
	if normalized < 0 {
		normalized = 0
	}
	if normalized > 100 {
		normalized = 100
	}

	return normalized
}

// CalculateQualityScore calculates service quality score based on delivery speed and error rate
func (c *Calculator) CalculateQualityScore(metrics *PerformanceMetrics) float64 {
	qualityScore := 100.0

	// Penalize based on error rate (0-50 points penalty)
	// Error rate of 0% = no penalty, 10% = 25 points penalty, 20%+ = 50 points penalty
	errorPenalty := 0.0
	if metrics.TotalMessages > 0 {
		errorRate := (float64(metrics.ErrorCount) / float64(metrics.TotalMessages)) * 100
		if errorRate > 0 {
			// Linear penalty up to 20% error rate
			errorPenalty = math.Min(errorRate*2.5, 50)
		}
	}
	qualityScore -= errorPenalty

	// Penalize based on delivery time (0-30 points penalty)
	// Fast delivery (< 1s) = no penalty
	// Medium delivery (1-5s) = 10-20 points penalty
	// Slow delivery (> 5s) = 30 points penalty
	deliveryPenalty := 0.0
	if metrics.AverageDeliveryTimeMs != nil {
		avgDeliveryMs := *metrics.AverageDeliveryTimeMs
		switch {
		case avgDeliveryMs < 1000:
			deliveryPenalty = 0
		case avgDeliveryMs < 5000:
			// Linear penalty from 0 to 20 points
			deliveryPenalty = float64(avgDeliveryMs-1000) / 4000 * 20
		default:
			deliveryPenalty = 30
		}
	}
	qualityScore -= deliveryPenalty

	// Bonus for consistent performance (0-20 points bonus)
	// If P95 and P99 are close to average, give bonus
	consistencyBonus := 0.0
	if metrics.AverageDeliveryTimeMs != nil && metrics.P95DeliveryTimeMs != nil && metrics.P99DeliveryTimeMs != nil {
		avgMs := float64(*metrics.AverageDeliveryTimeMs)
		p95Ms := float64(*metrics.P95DeliveryTimeMs)
		p99Ms := float64(*metrics.P99DeliveryTimeMs)

		if avgMs > 0 {
			// Calculate variance ratio
			p95Ratio := p95Ms / avgMs
			p99Ratio := p99Ms / avgMs

			// If P95 is within 2x of average and P99 within 3x, give bonus
			if p95Ratio <= 2.0 && p99Ratio <= 3.0 {
				// Better consistency = higher bonus
				consistencyBonus = 20 * (1 - (p95Ratio-1)/1.0)
				if consistencyBonus < 0 {
					consistencyBonus = 0
				}
			}
		}
	}
	qualityScore += consistencyBonus

	// Ensure score is within bounds
	if qualityScore < 0 {
		qualityScore = 0
	}
	if qualityScore > 100 {
		qualityScore = 100
	}

	return qualityScore
}

// CalculateRatingWithWeights calculates rating with custom weights
func (c *Calculator) CalculateRatingWithWeights(metrics *PerformanceMetrics, weights RatingWeights) (float64, error) {
	if !weights.Validate() {
		return 0, fmt.Errorf("invalid weights: sum must equal 1.0")
	}

	// Temporarily use custom weights
	originalWeights := c.weights
	c.weights = weights
	rating := c.CalculateRating(metrics)
	c.weights = originalWeights

	return rating, nil
}
