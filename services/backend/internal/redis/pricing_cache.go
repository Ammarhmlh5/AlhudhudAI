package redis

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"go.uber.org/zap"
)

// PricingInfo represents cached pricing information
type PricingInfo struct {
	CharCount  int
	PartsCount int
	Encoding   string
	Cost       float64
}

// PricingCache provides fast pricing calculation caching
type PricingCache struct {
	client *Client
	ttl    time.Duration
}

// NewPricingCache creates a new pricing cache
func NewPricingCache(client *Client) *PricingCache {
	return &PricingCache{
		client: client,
		ttl:    1 * time.Hour, // Cache for 1 hour
	}
}

// Get retrieves cached pricing info
func (pc *PricingCache) Get(ctx context.Context, encoding string, length int) (*PricingInfo, error) {
	key := pc.getCacheKey(encoding, length)
	
	var result map[string]string
	var err error
	
	// Use retry wrapper for cache get operation
	retryErr := pc.client.WithRetry(ctx, func() error {
		result, err = pc.client.Client.HGetAll(ctx, key).Result()
		return err
	})
	
	if retryErr != nil {
		return nil, retryErr
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("pricing not found")
	}

	charCount, _ := strconv.Atoi(result["char_count"])
	partsCount, _ := strconv.Atoi(result["parts_count"])
	cost, _ := strconv.ParseFloat(result["cost"], 64)

	return &PricingInfo{
		CharCount:  charCount,
		PartsCount: partsCount,
		Encoding:   result["encoding"],
		Cost:       cost,
	}, nil
}

// Set stores pricing info in cache
func (pc *PricingCache) Set(ctx context.Context, encoding string, length int, info *PricingInfo) error {
	key := pc.getCacheKey(encoding, length)
	
	data := map[string]interface{}{
		"char_count":  info.CharCount,
		"parts_count": info.PartsCount,
		"encoding":    info.Encoding,
		"cost":        info.Cost,
	}

	// Use retry wrapper for cache set operation
	return pc.client.WithRetry(ctx, func() error {
		pipe := pc.client.Client.Pipeline()
		pipe.HSet(ctx, key, data)
		pipe.Expire(ctx, key, pc.ttl)
		
		_, err := pipe.Exec(ctx)
		return err
	})
}

// GetOrCalculate retrieves from cache or calculates and caches
func (pc *PricingCache) GetOrCalculate(ctx context.Context, content string, costPerPart float64) (*PricingInfo, error) {
	charCount := len([]rune(content))
	isUnicode := false
	
	for _, r := range content {
		if r > 127 {
			isUnicode = true
			break
		}
	}

	encoding := "GSM7"
	if isUnicode {
		encoding = "UNICODE"
	}

	// Try to get from cache
	cached, err := pc.Get(ctx, encoding, charCount)
	if err == nil {
		return cached, nil
	}

	// Calculate
	var partsCount int
	if isUnicode {
		if charCount <= 70 {
			partsCount = 1
		} else {
			partsCount = (charCount + 66) / 67
		}
	} else {
		if charCount <= 160 {
			partsCount = 1
		} else {
			partsCount = (charCount + 152) / 153
		}
	}

	info := &PricingInfo{
		CharCount:  charCount,
		PartsCount: partsCount,
		Encoding:   encoding,
		Cost:       float64(partsCount) * costPerPart,
	}

	// Cache it
	if err := pc.Set(ctx, encoding, charCount, info); err != nil {
		// Log at WARN level - cache failure is not critical but should be visible
		if pc.client != nil && pc.client.Logger != nil {
			pc.client.Logger.Warn("Failed to cache pricing info",
				zap.String("encoding", encoding),
				zap.Int("charCount", charCount),
				zap.Error(err),
			)
		}
	}

	return info, nil
}

// getCacheKey returns the Redis key for pricing cache
func (pc *PricingCache) getCacheKey(encoding string, length int) string {
	return fmt.Sprintf("%s:cache:pricing:%s:%d", AppNamespace, encoding, length)
}

// InvalidateAll clears all pricing cache (for maintenance)
// Uses SCAN with COUNT=1000 for efficient pagination and batch deletion with pipeline
func (pc *PricingCache) InvalidateAll(ctx context.Context) error {
	pattern := fmt.Sprintf("%s:cache:pricing:*", AppNamespace)
	
	var cursor uint64
	var totalDeleted int
	startTime := time.Now()
	
	for {
		// SCAN with COUNT parameter for efficient pagination
		// COUNT=1000 ensures we don't block Redis for too long
		keys, nextCursor, err := pc.client.Client.Scan(ctx, cursor, pattern, 1000).Result()
		if err != nil {
			return fmt.Errorf("scan keys: %w", err)
		}
		
		// Delete keys in pipeline if any found
		if len(keys) > 0 {
			pipe := pc.client.Client.Pipeline()
			for _, key := range keys {
				pipe.Del(ctx, key)
			}
			
			_, err = pipe.Exec(ctx)
			if err != nil {
				// Log error but continue processing remaining keys
				if pc.client != nil && pc.client.Logger != nil {
					pc.client.Logger.Error("Failed to execute pipeline batch during cache invalidation",
						zap.Int("batch_size", len(keys)),
						zap.Error(err),
					)
				}
				return fmt.Errorf("delete keys batch: %w", err)
			}
			
			totalDeleted += len(keys)
		}
		
		// Move to next cursor
		cursor = nextCursor
		
		// Break if scan complete
		if cursor == 0 {
			break
		}
	}
	
	// Performance logging
	duration := time.Since(startTime)
	if pc.client != nil && pc.client.Logger != nil {
		pc.client.Logger.Info("Cache invalidation completed",
			zap.Int("keys_deleted", totalDeleted),
			zap.Duration("duration", duration),
			zap.Float64("keys_per_second", float64(totalDeleted)/duration.Seconds()),
		)
	}
	
	return nil
}

// Warmup pre-calculates and caches common message lengths
func (pc *PricingCache) Warmup(ctx context.Context, costPerPart float64) error {
	// Common lengths for GSM7
	gsmLengths := []int{50, 100, 160, 200, 300, 320, 480}
	
	// Common lengths for UNICODE
	unicodeLengths := []int{35, 70, 100, 134, 200, 268}

	pipe := pc.client.Client.Pipeline()

	for _, length := range gsmLengths {
		var partsCount int
		if length <= 160 {
			partsCount = 1
		} else {
			partsCount = (length + 152) / 153
		}

		info := &PricingInfo{
			CharCount:  length,
			PartsCount: partsCount,
			Encoding:   "GSM7",
			Cost:       float64(partsCount) * costPerPart,
		}

		key := pc.getCacheKey("GSM7", length)
		data := map[string]interface{}{
			"char_count":  info.CharCount,
			"parts_count": info.PartsCount,
			"encoding":    info.Encoding,
			"cost":        info.Cost,
		}
		pipe.HSet(ctx, key, data)
		pipe.Expire(ctx, key, pc.ttl)
	}

	for _, length := range unicodeLengths {
		var partsCount int
		if length <= 70 {
			partsCount = 1
		} else {
			partsCount = (length + 66) / 67
		}

		info := &PricingInfo{
			CharCount:  length,
			PartsCount: partsCount,
			Encoding:   "UNICODE",
			Cost:       float64(partsCount) * costPerPart,
		}

		key := pc.getCacheKey("UNICODE", length)
		data := map[string]interface{}{
			"char_count":  info.CharCount,
			"parts_count": info.PartsCount,
			"encoding":    info.Encoding,
			"cost":        info.Cost,
		}
		pipe.HSet(ctx, key, data)
		pipe.Expire(ctx, key, pc.ttl)
	}

	_, err := pipe.Exec(ctx)
	return err
}
