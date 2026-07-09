package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/al-hudhud/backend/internal/cache"
	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockCacheClient - Mock for cache operations
type MockCacheClient struct {
	mock.Mock
}

func (m *MockCacheClient) Get(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.Get(0).(string), args.Error(1)
}

func (m *MockCacheClient) Set(ctx context.Context, key string, value string, expiration time.Duration) error {
	args := m.Called(ctx, key, value, expiration)
	return args.Error(0)
}

func (m *MockCacheClient) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockCacheClient) DeletePattern(ctx context.Context, pattern string) error {
	args := m.Called(ctx, pattern)
	return args.Error(0)
}

func (m *MockCacheClient) Exists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Get(0).(bool), args.Error(1)
}

func (m *MockCacheClient) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	args := m.Called(ctx, key)
	return args.Get(0).(time.Duration), args.Error(1)
}

func (m *MockCacheClient) Increment(ctx context.Context, key string) (int64, error) {
	args := m.Called(ctx, key)
	return args.Get(0).(int64), args.Error(1)
}

func TestCacheManager_SetAndGet(t *testing.T) {
	// Arrange
	key := "test:key"
	value := "test value"
	expiration := time.Hour

	mockClient := new(MockCacheClient)
	mockClient.On("Set", mock.Anything, key, value, expiration).Return(nil)
	mockClient.On("Get", mock.Anything, key).Return(value, nil)

	manager := cache.NewManager(mockClient)

	// Act
	err := manager.Set(context.Background(), key, value, expiration)
	assert.NoError(t, err)

	result, err := manager.Get(context.Background(), key)
	assert.NoError(t, err)
	assert.Equal(t, value, result)
	mockClient.AssertExpectations(t)
}

func TestCacheManager_Get_NotFound(t *testing.T) {
	// Arrange
	key := "nonexistent:key"

	mockClient := new(MockCacheClient)
	mockClient.On("Get", mock.Anything, key).Return("", cache.ErrCacheMiss)

	manager := cache.NewManager(mockClient)

	// Act
	result, err := manager.Get(context.Background(), key)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, cache.ErrCacheMiss, err)
	assert.Empty(t, result)
	mockClient.AssertExpectations(t)
}

func TestCacheManager_Delete(t *testing.T) {
	// Arrange
	key := "test:key"

	mockClient := new(MockCacheClient)
	mockClient.On("Delete", mock.Anything, key).Return(nil)

	manager := cache.NewManager(mockClient)

	// Act
	err := manager.Delete(context.Background(), key)

	// Assert
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestCacheManager_DeletePattern(t *testing.T) {
	// Arrange
	pattern := "tenant:123:*"

	mockClient := new(MockCacheClient)
	mockClient.On("DeletePattern", mock.Anything, pattern).Return(nil)

	manager := cache.NewManager(mockClient)

	// Act
	err := manager.DeletePattern(context.Background(), pattern)

	// Assert
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestCacheManager_Exists(t *testing.T) {
	// Arrange
	key := "test:key"

	mockClient := new(MockCacheClient)
	mockClient.On("Exists", mock.Anything, key).Return(true, nil)

	manager := cache.NewManager(mockClient)

	// Act
	exists, err := manager.Exists(context.Background(), key)

	// Assert
	assert.NoError(t, err)
	assert.True(t, exists)
	mockClient.AssertExpectations(t)
}

func TestCacheManager_Increment(t *testing.T) {
	// Arrange
	key := "counter:key"

	mockClient := new(MockCacheClient)
	mockClient.On("Increment", mock.Anything, key).Return(int64(5), nil)

	manager := cache.NewManager(mockClient)

	// Act
	result, err := manager.Increment(context.Background(), key)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, int64(5), result)
	mockClient.AssertExpectations(t)
}

func TestCacheManager_WithMiniredis(t *testing.T) {
	// Setup Miniredis
	mr, err := miniredis.Run()
	if err != nil {
		t.Skip("Miniredis not available")
	}
	defer mr.Close()

	// Create real cache client
	cacheClient := cache.NewRedisClient(mr.Addr())
	manager := cache.NewManager(cacheClient)

	// Test Set/Get
	err = manager.Set(context.Background(), "user:1", "John Doe", time.Hour)
	assert.NoError(t, err)

	result, err := manager.Get(context.Background(), "user:1")
	assert.NoError(t, err)
	assert.Equal(t, "John Doe", result)

	// Test Exists
	exists, err := manager.Exists(context.Background(), "user:1")
	assert.NoError(t, err)
	assert.True(t, exists)

	// Test Delete
	err = manager.Delete(context.Background(), "user:1")
	assert.NoError(t, err)

	exists, err = manager.Exists(context.Background(), "user:1")
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestCacheManager_TTL(t *testing.T) {
	// Arrange
	key := "test:key"
	expectedTTL := time.Hour

	mockClient := new(MockCacheClient)
	mockClient.On("GetTTL", mock.Anything, key).Return(expectedTTL, nil)

	manager := cache.NewManager(mockClient)

	// Act
	ttl, err := manager.GetTTL(context.Background(), key)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, expectedTTL, ttl)
	mockClient.AssertExpectations(t)
}

func TestCacheManager_ChannelCache(t *testing.T) {
	// Arrange
	tenantID := "tenant-123"
	channelData := `{"id": "1", "name": "Test Channel", "status": "ONLINE"}`

	mockClient := new(MockCacheClient)
	mockClient.On("Set", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).Return(nil)
	mockClient.On("Get", mock.Anything, mock.AnythingOfType("string")).Return(channelData, nil)

	manager := cache.NewManager(mockClient)

	// Act - Set channel cache
	err := manager.SetChannelCache(context.Background(), tenantID, channelData)
	assert.NoError(t, err)

	// Act - Get channel cache
	result, err := manager.GetChannelCache(context.Background(), tenantID)
	assert.NoError(t, err)
	assert.Equal(t, channelData, result)
	mockClient.AssertExpectations(t)
}

func TestCacheManager_InvalidateTenantCache(t *testing.T) {
	// Arrange
	tenantID := "tenant-123"

	mockClient := new(MockCacheClient)
	mockClient.On("DeletePattern", mock.Anything, "tenant:"+tenantID+":*").Return(nil)

	manager := cache.NewManager(mockClient)

	// Act
	err := manager.InvalidateTenantCache(context.Background(), tenantID)

	// Assert
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}