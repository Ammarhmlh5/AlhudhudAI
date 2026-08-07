package tariff

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/al-hudhud/backend/internal/models"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Engine defines the tariff calculation engine interface
type Engine interface {
	// CalculateCost calculates the cost of a message based on tariffs
	// Implements Requirements 2.5, 2.6
	CalculateCost(ctx context.Context, req CostCalculationRequest) (*CostCalculationResult, error)

	// CalculateSegments calculates the number of message segments
	// Implements Requirements 7.1, 7.2, 7.3, 7.6
	CalculateSegments(ctx context.Context, messageLength int, encodingType string) (int, error)

	// GetActiveTariff retrieves the active tariff for given parameters
	GetActiveTariff(ctx context.Context, req TariffRequest) (*models.Tariff, error)

	// InvalidateCache invalidates the tariff cache
	InvalidateCache(ctx context.Context, serviceID int, countryCode, networkOperator, messageType string) error
}

// engine implements the Engine interface
type engine struct {
	db     *gorm.DB
	cache  *redis.Client
	logger *zap.Logger
}

type CostCalculationRequest struct {
	TenantID        uuid.UUID
	ServiceID       int
	CountryCode     string
	NetworkOperator string
	MessageType     string // "GSM-7" or "Unicode"
	MessageLength   int
}

// CostCalculationResult represents the result of cost calculation
type CostCalculationResult struct {
	TotalCost       decimal.Decimal
	CostPerSegment  decimal.Decimal
	Segments        int
	TariffID        int
	CurrencyCode    string
}

// TariffRequest represents a tariff lookup request
type TariffRequest struct {
	TenantID        uuid.UUID
	ServiceID       int
	CountryCode     string
	NetworkOperator string
	MessageType     string
}

// NewEngine creates a new tariff engine
func NewEngine(db *gorm.DB, cache *redis.Client, logger *zap.Logger) Engine {
	return &engine{
		db:     db,
		cache:  cache,
		logger: logger,
	}
}

// CalculateCost calculates the cost of a message based on tariffs
// Implements Requirements 2.5, 2.6
func (e *engine) CalculateCost(ctx context.Context, req CostCalculationRequest) (*CostCalculationResult, error) {
	// Validate request
	if req.ServiceID <= 0 {
		return nil, fmt.Errorf("invalid service_id: %d", req.ServiceID)
	}
	if req.CountryCode == "" {
		return nil, fmt.Errorf("country_code is required")
	}
	if req.MessageLength <= 0 {
		return nil, fmt.Errorf("invalid message_length: %d", req.MessageLength)
	}

	// 1. Calculate number of segments
	segments, err := e.CalculateSegments(ctx, req.MessageLength, req.MessageType)
	if err != nil {
		e.logger.Error("Failed to calculate segments",
			zap.Error(err),
			zap.Int("message_length", req.MessageLength),
			zap.String("message_type", req.MessageType),
		)
		return nil, fmt.Errorf("failed to calculate segments: %w", err)
	}

	// 2. Get active tariff (with caching)
	tariff, err := e.GetActiveTariff(ctx, TariffRequest{
		TenantID:        req.TenantID,
		ServiceID:       req.ServiceID,
		CountryCode:     req.CountryCode,
		NetworkOperator: req.NetworkOperator,
		MessageType:     req.MessageType,
	})
	if err != nil {
		e.logger.Error("Failed to get active tariff",
			zap.Error(err),
			zap.String("tenant_id", req.TenantID.String()),
			zap.Int("service_id", req.ServiceID),
			zap.String("country_code", req.CountryCode),
			zap.String("network_operator", req.NetworkOperator),
		)
		return nil, fmt.Errorf("failed to get active tariff: %w", err)
	}

	// 3. Calculate total cost
	costPerSegment := decimal.NewFromFloat(tariff.CostPerMessage)
	totalCost := costPerSegment.Mul(decimal.NewFromInt(int64(segments)))

	e.logger.Debug("Cost calculated successfully",
		zap.String("tenant_id", req.TenantID.String()),
		zap.Int("service_id", req.ServiceID),
		zap.String("country_code", req.CountryCode),
		zap.String("network_operator", req.NetworkOperator),
		zap.Int("segments", segments),
		zap.String("cost_per_segment", costPerSegment.String()),
		zap.String("total_cost", totalCost.String()),
	)

	return &CostCalculationResult{
		TotalCost:      totalCost,
		CostPerSegment: costPerSegment,
		Segments:       segments,
		TariffID:       tariff.ID,
		CurrencyCode:   tariff.Currency,
	}, nil
}

// CalculateSegments calculates the number of message segments
// Implements Requirements 7.1, 7.2, 7.3, 7.6
func (e *engine) CalculateSegments(ctx context.Context, messageLength int, encodingType string) (int, error) {
	if messageLength <= 0 {
		return 0, fmt.Errorf("invalid message_length: %d", messageLength)
	}

	// Default to GSM-7 if not specified
	if encodingType == "" {
		encodingType = "GSM-7"
	}

	// Get segmentation rules from database
	var rule models.SegmentationRule
	err := e.db.WithContext(ctx).
		Where("encoding_type = ? AND is_active = true", encodingType).
		First(&rule).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			e.logger.Warn("Segmentation rule not found, using defaults",
				zap.String("encoding_type", encodingType),
			)
			// Use default values
			if encodingType == "GSM-7" {
				rule = models.SegmentationRule{
					EncodingType:  "GSM-7",
					Segment1Chars: 160,
					Segment2Chars: 153,
					Segment3Chars: 153,
					SegmentNChars: 153,
				}
			} else {
				rule = models.SegmentationRule{
					EncodingType:  "Unicode",
					Segment1Chars: 70,
					Segment2Chars: 67,
					Segment3Chars: 67,
					SegmentNChars: 67,
				}
			}
		} else {
			return 0, fmt.Errorf("failed to get segmentation rule: %w", err)
		}
	}

	// Calculate segments based on rules
	segments := calculateSegmentsFromRules(messageLength, rule)

	e.logger.Debug("Segments calculated",
		zap.Int("message_length", messageLength),
		zap.String("encoding_type", encodingType),
		zap.Int("segments", segments),
	)

	return segments, nil
}

// calculateSegmentsFromRules calculates segments based on segmentation rules
func calculateSegmentsFromRules(messageLength int, rule models.SegmentationRule) int {
	// Zero/negative lengths are invalid; they are rejected by validation
	// before this helper is reached.
	if messageLength <= 0 {
		return 0
	}

	// Single segment
	if messageLength <= rule.Segment1Chars {
		return 1
	}

	// Two segments
	if messageLength <= rule.Segment1Chars+rule.Segment2Chars {
		return 2
	}

	// Three segments
	if messageLength <= rule.Segment1Chars+rule.Segment2Chars+rule.Segment3Chars {
		return 3
	}

	// More than three segments
	remaining := messageLength - rule.Segment1Chars - rule.Segment2Chars - rule.Segment3Chars
	additionalSegments := int(math.Ceil(float64(remaining) / float64(rule.SegmentNChars)))

	return 3 + additionalSegments
}

// GetActiveTariff retrieves the active tariff for given parameters
// Implements Requirements 2.5, 2.6 with caching (TTL: 1 hour)
func (e *engine) GetActiveTariff(ctx context.Context, req TariffRequest) (*models.Tariff, error) {
	// Build cache key
	cacheKey := fmt.Sprintf("tariff:%d:%d:%s:%s:%s",
		req.TenantID,
		req.ServiceID,
		req.CountryCode,
		req.NetworkOperator,
		req.MessageType,
	)

	// Try to get from cache first
	cached, err := e.cache.Get(ctx, cacheKey).Result()
	if err == nil {
		var tariff models.Tariff
		if err := json.Unmarshal([]byte(cached), &tariff); err == nil {
			e.logger.Debug("Tariff retrieved from cache",
				zap.String("cache_key", cacheKey),
			)
			return &tariff, nil
		}
		// If unmarshal fails, continue to database lookup
		e.logger.Warn("Failed to unmarshal cached tariff",
			zap.Error(err),
			zap.String("cache_key", cacheKey),
		)
	}

	// Get from database
	var tariff models.Tariff
	query := e.db.WithContext(ctx).
		Where("tenant_id = ?", req.TenantID).
		Where("service_id = ?", req.ServiceID).
		Where("country_code = ?", req.CountryCode).
		Where("is_active = true").
		Where("effective_from <= CURRENT_DATE").
		Where("effective_to IS NULL OR effective_to >= CURRENT_DATE")

	// Add network operator filter if specified
	if req.NetworkOperator != "" {
		query = query.Where("network_operator = ?", req.NetworkOperator)
	}

	// Add message type filter if specified
	if req.MessageType != "" {
		query = query.Where("message_type = ?", req.MessageType)
	}

	// Order by specificity (more specific first)
	query = query.Order("network_operator DESC NULLS LAST").
		Order("message_type DESC NULLS LAST")

	err = query.First(&tariff).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			e.logger.Warn("Active tariff not found",
				zap.String("tenant_id", req.TenantID.String()),
				zap.Int("service_id", req.ServiceID),
				zap.String("country_code", req.CountryCode),
				zap.String("network_operator", req.NetworkOperator),
			)
			return nil, fmt.Errorf("active tariff not found for service_id=%d, country=%s, operator=%s",
				req.ServiceID, req.CountryCode, req.NetworkOperator)
		}
		return nil, fmt.Errorf("failed to get tariff from database: %w", err)
	}

	// Cache for 1 hour (Requirement 27.9)
	tariffJSON, err := json.Marshal(tariff)
	if err == nil {
		err = e.cache.Set(ctx, cacheKey, tariffJSON, time.Hour).Err()
		if err != nil {
			e.logger.Warn("Failed to cache tariff",
				zap.Error(err),
				zap.String("cache_key", cacheKey),
			)
		} else {
			e.logger.Debug("Tariff cached successfully",
				zap.String("cache_key", cacheKey),
			)
		}
	}

	return &tariff, nil
}

// InvalidateCache invalidates the tariff cache for specific parameters
// Implements Requirement 27.9 (Cache Invalidation)
func (e *engine) InvalidateCache(ctx context.Context, serviceID int, countryCode, networkOperator, messageType string) error {
	// Build cache key pattern
	pattern := fmt.Sprintf("tariff:*:%d:%s:%s:%s", serviceID, countryCode, networkOperator, messageType)

	// Get all matching keys
	keys, err := e.cache.Keys(ctx, pattern).Result()
	if err != nil {
		e.logger.Error("Failed to get cache keys for invalidation",
			zap.Error(err),
			zap.String("pattern", pattern),
		)
		return fmt.Errorf("failed to get cache keys: %w", err)
	}

	// Delete all matching keys
	if len(keys) > 0 {
		err = e.cache.Del(ctx, keys...).Err()
		if err != nil {
			e.logger.Error("Failed to delete cache keys",
				zap.Error(err),
				zap.Strings("keys", keys),
			)
			return fmt.Errorf("failed to delete cache keys: %w", err)
		}

		e.logger.Info("Cache invalidated successfully",
			zap.Int("keys_deleted", len(keys)),
			zap.String("pattern", pattern),
		)
	}

	return nil
}
