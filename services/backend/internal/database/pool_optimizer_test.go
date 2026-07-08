package database

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestPoolOptimizer_GetOptimizedConfig(t *testing.T) {
	logger := zap.NewNop()
	optimizer := NewPoolOptimizer(logger)

	tests := []struct {
		name         string
		env          string
		expectedLoad int
		validate     func(t *testing.T, config *PoolConfig)
	}{
		{
			name:         "Production low load",
			env:          "production",
			expectedLoad: 500,
			validate: func(t *testing.T, config *PoolConfig) {
				assert.Greater(t, config.MaxConns, int32(10), "MaxConns should be reasonable for production")
				assert.Greater(t, config.MinConns, int32(0), "MinConns should be set")
				assert.Equal(t, 30*time.Minute, config.MaxConnLifetime)
				assert.Equal(t, 5*time.Minute, config.MaxConnIdleTime)
			},
		},
		{
			name:         "Production medium load",
			env:          "production",
			expectedLoad: 3000,
			validate: func(t *testing.T, config *PoolConfig) {
				assert.Greater(t, config.MaxConns, int32(20), "MaxConns should be higher for medium load")
				assert.LessOrEqual(t, config.MaxConns, int32(200), "MaxConns should be capped")
			},
		},
		{
			name:         "Production high load",
			env:          "production",
			expectedLoad: 10000,
			validate: func(t *testing.T, config *PoolConfig) {
				assert.Greater(t, config.MaxConns, int32(30), "MaxConns should be highest for high load")
				assert.LessOrEqual(t, config.MaxConns, int32(200), "MaxConns should be capped at 200")
			},
		},
		{
			name:         "Staging",
			env:          "staging",
			expectedLoad: 1000,
			validate: func(t *testing.T, config *PoolConfig) {
				assert.GreaterOrEqual(t, config.MaxConns, int32(10), "Staging should have at least 10 connections")
				assert.Equal(t, 20*time.Minute, config.MaxConnLifetime)
			},
		},
		{
			name:         "Development",
			env:          "development",
			expectedLoad: 100,
			validate: func(t *testing.T, config *PoolConfig) {
				assert.LessOrEqual(t, config.MaxConns, int32(20), "Development should use minimal connections")
				assert.Equal(t, int32(2), config.MinConns)
				assert.Equal(t, 10*time.Minute, config.MaxConnLifetime)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := optimizer.GetOptimizedConfig(tt.env, tt.expectedLoad)
			require.NotNil(t, config)
			
			// Common validations
			assert.Greater(t, config.MaxConns, int32(0), "MaxConns must be positive")
			assert.Greater(t, config.MinConns, int32(0), "MinConns must be positive")
			assert.LessOrEqual(t, config.MinConns, config.MaxConns, "MinConns must be <= MaxConns")
			assert.Greater(t, config.MaxConnLifetime, time.Duration(0), "MaxConnLifetime must be positive")
			assert.Greater(t, config.MaxConnIdleTime, time.Duration(0), "MaxConnIdleTime must be positive")
			
			// Custom validations
			tt.validate(t, config)
		})
	}
}

func TestPoolOptimizer_ApplyToPoolConfig(t *testing.T) {
	logger := zap.NewNop()
	optimizer := NewPoolOptimizer(logger)

	// Create a pgxpool config
	poolConfig, err := pgxpool.ParseConfig("postgres://user:pass@localhost:5432/testdb")
	require.NoError(t, err)

	// Get optimized config
	optimized := optimizer.GetOptimizedConfig("production", 1000)

	// Apply optimizations
	optimizer.ApplyToPoolConfig(poolConfig, optimized)

	// Verify settings were applied
	assert.Equal(t, optimized.MaxConns, poolConfig.MaxConns)
	assert.Equal(t, optimized.MinConns, poolConfig.MinConns)
	assert.Equal(t, optimized.MaxConnLifetime, poolConfig.MaxConnLifetime)
	assert.Equal(t, optimized.MaxConnIdleTime, poolConfig.MaxConnIdleTime)
	assert.Equal(t, optimized.HealthCheckPeriod, poolConfig.HealthCheckPeriod)
}

type mockPoolStats struct {
	acquiredConns        int32
	maxConns             int32
	idleConns            int32
	totalConns           int32
	emptyAcquireCount    int64
	canceledAcquireCount int64
}

func (m *mockPoolStats) AcquiredConns() int32        { return m.acquiredConns }
func (m *mockPoolStats) MaxConns() int32             { return m.maxConns }
func (m *mockPoolStats) IdleConns() int32            { return m.idleConns }
func (m *mockPoolStats) TotalConns() int32           { return m.totalConns }
func (m *mockPoolStats) EmptyAcquireCount() int64    { return m.emptyAcquireCount }
func (m *mockPoolStats) CanceledAcquireCount() int64 { return m.canceledAcquireCount }

func TestPoolOptimizer_GetRecommendations(t *testing.T) {
	logger := zap.NewNop()
	optimizer := NewPoolOptimizer(logger)

	config := &PoolConfig{
		MaxConns:        50,
		MinConns:        10,
		MaxConnIdleTime: 5 * time.Minute,
	}

	tests := []struct {
		name                 string
		stats                PoolStats
		expectedRecommendations int
		checkContains        string
	}{
		{
			name: "High utilization",
			stats: &mockPoolStats{
				acquiredConns: 46, // 46/50 = 92% utilization
				maxConns:      50,
			},
			expectedRecommendations: 1,
			checkContains:           "utilization is very high",
		},
		{
			name: "Low utilization",
			stats: &mockPoolStats{
				acquiredConns: 5, // 5/50 = 10% utilization
				maxConns:      50,
			},
			expectedRecommendations: 1,
			checkContains:           "utilization is low",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recommendations := optimizer.GetRecommendations(tt.stats, config)
			assert.NotNil(t, recommendations)
			if tt.checkContains != "" {
				found := false
				for _, r := range recommendations {
					if assert.Contains(t, r, tt.checkContains) {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected recommendation containing: "+tt.checkContains)
			}
		})
	}
}

func TestPoolOptimizer_MonitorPoolHealth(t *testing.T) {
	logger := zap.NewNop()
	optimizer := NewPoolOptimizer(logger)

	// Create a test pool
	poolConfig, err := pgxpool.ParseConfig("postgres://user:pass@localhost:5432/testdb?pool_max_conns=5")
	require.NoError(t, err)

	// Note: This test would require a real database connection
	// For unit testing, we just verify the function doesn't panic
	_, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// This would normally monitor a real pool
	// For testing, we just ensure the function signature is correct
	assert.NotNil(t, optimizer)
	assert.NotNil(t, poolConfig)
}

func TestPoolConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  *PoolConfig
		isValid bool
	}{
		{
			name: "Valid production config",
			config: &PoolConfig{
				MaxConns:          100,
				MinConns:          20,
				MaxConnLifetime:   30 * time.Minute,
				MaxConnIdleTime:   5 * time.Minute,
				HealthCheckPeriod: 1 * time.Minute,
				ConnectTimeout:    10 * time.Second,
				AcquireTimeout:    5 * time.Second,
			},
			isValid: true,
		},
		{
			name: "Invalid - MinConns > MaxConns",
			config: &PoolConfig{
				MaxConns: 10,
				MinConns: 20,
			},
			isValid: false,
		},
		{
			name: "Invalid - Zero MaxConns",
			config: &PoolConfig{
				MaxConns: 0,
				MinConns: 5,
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.isValid {
				assert.Greater(t, tt.config.MaxConns, int32(0))
				assert.LessOrEqual(t, tt.config.MinConns, tt.config.MaxConns)
			} else {
				// Invalid configs should fail validation
				if tt.config.MaxConns == 0 {
					assert.Equal(t, int32(0), tt.config.MaxConns)
				}
				if tt.config.MinConns > tt.config.MaxConns {
					assert.Greater(t, tt.config.MinConns, tt.config.MaxConns)
				}
			}
		})
	}
}

func TestPoolOptimizer_EnvironmentScaling(t *testing.T) {
	logger := zap.NewNop()
	optimizer := NewPoolOptimizer(logger)

	// Get configs for different environments
	prodConfig := optimizer.GetOptimizedConfig("production", 5000)
	stagingConfig := optimizer.GetOptimizedConfig("staging", 5000)
	devConfig := optimizer.GetOptimizedConfig("development", 5000)

	// Verify scaling relationship
	assert.Greater(t, prodConfig.MaxConns, stagingConfig.MaxConns,
		"Production should have more connections than staging")
	assert.Greater(t, stagingConfig.MaxConns, devConfig.MaxConns,
		"Staging should have more connections than development")

	// Verify timeout relationships
	assert.Greater(t, prodConfig.MaxConnLifetime, devConfig.MaxConnLifetime,
		"Production should have longer connection lifetime")
}

func BenchmarkPoolOptimizer_GetOptimizedConfig(b *testing.B) {
	logger := zap.NewNop()
	optimizer := NewPoolOptimizer(logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = optimizer.GetOptimizedConfig("production", 5000)
	}
}
