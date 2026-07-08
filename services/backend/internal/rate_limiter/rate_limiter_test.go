package rate_limiter_test

import (
	"context"
	"testing"
	"time"

	"github.com/al-hudhud/backend/internal/rate_limiter"
	"github.com/al-hudhud/backend/internal/redis"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRedisClient - Mock for Redis operations
type MockRedisClient struct {
	mock.Mock
}

func (m *MockRedisClient) Get(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.Get(0).(string), args.Error(1)
}

func (m *MockRedisClient) Set(ctx context.Context, key string, value string, expiration time.Duration) error {
	args := m.Called(ctx, key, value, expiration)
	return args.Error(0)
}

func (m *MockRedisClient) Incr(ctx context.Context, key string) (int64, error) {
	args := m.Called(ctx, key)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockRedisClient) Expire(ctx context.Context, key string, expiration time.Duration) error {
	args := m.Called(ctx, key, expiration)
	return args.Error(0)
}

func (m *MockRedisClient) TTL(ctx context.Context, key string) (time.Duration, error) {
	args := m.Called(ctx, key)
	return args.Get(0).(time.Duration), args.Error(1)
}

func (m *MockRedisClient) Delete(ctx context.Context, keys ...string) error {
	args := m.Called(ctx, keys)
	return args.Error(0)
}

func TestRateLimiter_AllowRequest_UnderLimit(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	limit := 100
	window := time.Minute

	mockRedis := new(MockRedisClient)
	mockRedis.On("Get", mock.Anything, mock.AnythingOfType("string")).Return("0", nil)
	mockRedis.On("Incr", mock.Anything, mock.AnythingOfType("string")).Return(int64(1), nil)
	mockRedis.On("Expire", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).Return(nil)

	limiter := rate_limiter.NewRateLimiter(mockRedis, limit, window)

	// Act
	allowed, remaining, err := limiter.AllowRequest(context.Background(), tenantID)

	// Assert
	assert.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, 99, remaining)
	mockRedis.AssertExpectations(t)
}

func TestRateLimiter_AllowRequest_OverLimit(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	limit := 100
	window := time.Minute

	mockRedis := new(MockRedisClient)
	mockRedis.On("Get", mock.Anything, mock.AnythingOfType("string")).Return("100", nil)

	limiter := rate_limiter.NewRateLimiter(mockRedis, limit, window)

	// Act
	allowed, remaining, err := limiter.AllowRequest(context.Background(), tenantID)

	// Assert
	assert.NoError(t, err)
	assert.False(t, allowed)
	assert.Equal(t, 0, remaining)
	mockRedis.AssertExpectations(t)
}

func TestRateLimiter_Reset(t *testing.T) {
	// Arrange
	tenantID := uuid.New()

	mockRedis := new(MockRedisClient)
	mockRedis.On("Delete", mock.Anything, mock.AnythingOfType("[]string")).Return(nil)

	limiter := rate_limiter.NewRateLimiter(mockRedis, 100, time.Minute)

	// Act
	err := limiter.Reset(context.Background(), tenantID)

	// Assert
	assert.NoError(t, err)
	mockRedis.AssertExpectations(t)
}

func TestRateLimiter_GetRemainingRequests(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	limit := 100

	mockRedis := new(MockRedisClient)
	mockRedis.On("Get", mock.Anything, mock.AnythingOfType("string")).Return("75", nil)

	limiter := rate_limiter.NewRateLimiter(mockRedis, limit, time.Minute)

	// Act
	remaining, err := limiter.GetRemainingRequests(context.Background(), tenantID)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 25, remaining)
	mockRedis.AssertExpectations(t)
}

func TestRateLimiter_WithMiniredis(t *testing.T) {
	// Setup Miniredis
	mr, err := miniredis.Run()
	if err != nil {
		t.Skip("Miniredis not available")
	}
	defer mr.Close()

	// Create real Redis client
	redisClient := redis.NewClient(mr.Addr())

	// Create rate limiter
	limiter := rate_limiter.NewRateLimiter(redisClient, 10, time.Minute)
	tenantID := uuid.New()

	// Test under limit
	for i := 0; i < 10; i++ {
		allowed, remaining, err := limiter.AllowRequest(context.Background(), tenantID)
		assert.NoError(t, err)
		assert.True(t, allowed)
		assert.Equal(t, 9-i, remaining)
	}

	// Test over limit
	allowed, remaining, err := limiter.AllowRequest(context.Background(), tenantID)
	assert.NoError(t, err)
	assert.False(t, allowed)
	assert.Equal(t, 0, remaining)
}

func TestRateLimiter_DifferentTenants(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Skip("Miniredis not available")
	}
	defer mr.Close()

	redisClient := redis.NewClient(mr.Addr())
	tenant1 := uuid.New()
	tenant2 := uuid.New()
	limit := 5
	window := time.Minute

	limiter := rate_limiter.NewRateLimiter(redisClient, limit, window)

	for i := 0; i < limit; i++ {
		allowed, remaining, err := limiter.AllowRequest(context.Background(), tenant1)
		assert.NoError(t, err)
		assert.True(t, allowed)
		assert.Equal(t, limit-1-i, remaining)
	}

	allowed, remaining, err := limiter.AllowRequest(context.Background(), tenant1)
	assert.NoError(t, err)
	assert.False(t, allowed)
	assert.Equal(t, 0, remaining)

	allowed, remaining, err = limiter.AllowRequest(context.Background(), tenant2)
	assert.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, limit-1, remaining)
}