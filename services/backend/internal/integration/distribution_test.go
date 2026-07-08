//go:build mock

package integration

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMessageDistribution tests message distribution across multiple devices
func TestMessageDistribution(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	t.Run("Round Robin distribution", func(t *testing.T) {
		tenantID := createTestTenant(t, db, "round-robin-tenant")
		
		// Create 3 devices
		device1 := createTestDevice(t, db, tenantID, "device-1", 1)
		device2 := createTestDevice(t, db, tenantID, "device-2", 1)
		device3 := createTestDevice(t, db, tenantID, "device-3", 1)

		// Configure round robin strategy
		configureDistributionStrategy(t, db, tenantID, "round_robin", nil)

		// Distribute 30 messages
		deviceCounts := make(map[string]int)
		for i := 0; i < 30; i++ {
			device, err := selectDeviceForMessage(ctx, db, tenantID, "message")
			require.NoError(t, err)
			deviceCounts[device.ID]++
		}

		// Verify fair distribution (each device should get ~10 messages)
		assert.Equal(t, 10, deviceCounts[device1])
		assert.Equal(t, 10, deviceCounts[device2])
		assert.Equal(t, 10, deviceCounts[device3])
	})

	t.Run("Load Balancing distribution", func(t *testing.T) {
		tenantID := createTestTenant(t, db, "load-balance-tenant")
		
		// Create devices with different loads
		device1 := createTestDevice(t, db, tenantID, "device-1", 1)
		device2 := createTestDevice(t, db, tenantID, "device-2", 1)
		device3 := createTestDevice(t, db, tenantID, "device-3", 1)
		
		// Set different loads
		setDeviceLoad(t, db, device1, 10)
		setDeviceLoad(t, db, device2, 5)
		setDeviceLoad(t, db, device3, 0)

		// Configure load balancing strategy
		configureDistributionStrategy(t, db, tenantID, "load_balancing", nil)

		// Distribute 10 messages
		deviceCounts := make(map[string]int)
		for i := 0; i < 10; i++ {
			device, err := selectDeviceForMessage(ctx, db, tenantID, "message")
			require.NoError(t, err)
			deviceCounts[device.ID]++
		}

		// Device with lowest load (device3) should get most messages
		assert.Greater(t, deviceCounts[device3], deviceCounts[device1], 
			"Device with lowest load should get more messages")
		assert.Greater(t, deviceCounts[device3], deviceCounts[device2],
			"Device with lowest load should get more messages")
	})

	t.Run("Priority-Based distribution", func(t *testing.T) {
		tenantID := createTestTenant(t, db, "priority-tenant")
		
		// Create devices with different priorities
		device1 := createTestDevice(t, db, tenantID, "device-1", 3) // High priority
		device2 := createTestDevice(t, db, tenantID, "device-2", 2) // Medium priority
		device3 := createTestDevice(t, db, tenantID, "device-3", 1) // Low priority

		// Configure priority strategy
		configureDistributionStrategy(t, db, tenantID, "priority", nil)

		// Distribute 30 messages
		deviceCounts := make(map[string]int)
		for i := 0; i < 30; i++ {
			device, err := selectDeviceForMessage(ctx, db, tenantID, "message")
			require.NoError(t, err)
			deviceCounts[device.ID]++
			
			// Simulate some load
			if deviceCounts[device.ID]%5 == 0 {
				setDeviceLoad(t, db, device.ID, deviceCounts[device.ID])
			}
		}

		// Higher priority device should get more messages
		assert.Greater(t, deviceCounts[device1], deviceCounts[device3],
			"Higher priority device should get more messages")
	})
}

// TestDeviceFailureHandling tests device failure handling
func TestDeviceFailureHandling(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	t.Run("Exclude failed device", func(t *testing.T) {
		tenantID := createTestTenant(t, db, "failure-tenant")
		
		// Create 3 devices
		device1 := createTestDevice(t, db, tenantID, "device-1", 1)
		device2 := createTestDevice(t, db, tenantID, "device-2", 1)
		device3 := createTestDevice(t, db, tenantID, "device-3", 1)

		// Configure round robin
		configureDistributionStrategy(t, db, tenantID, "round_robin", nil)

		// Mark device2 as failed
		markDeviceAsFailed(t, db, device2)

		// Distribute 20 messages
		deviceCounts := make(map[string]int)
		for i := 0; i < 20; i++ {
			device, err := selectDeviceForMessage(ctx, db, tenantID, "message")
			require.NoError(t, err)
			deviceCounts[device.ID]++
		}

		// Failed device should not receive messages
		assert.Equal(t, 0, deviceCounts[device2], "Failed device should not receive messages")
		assert.Equal(t, 10, deviceCounts[device1])
		assert.Equal(t, 10, deviceCounts[device3])
	})

	t.Run("Re-include device after timeout", func(t *testing.T) {
		tenantID := createTestTenant(t, db, "timeout-tenant")
		
		device1 := createTestDevice(t, db, tenantID, "device-1", 1)
		device2 := createTestDevice(t, db, tenantID, "device-2", 1)

		configureDistributionStrategy(t, db, tenantID, "round_robin", nil)

		// Mark device2 as failed with short timeout (1 second for testing)
		markDeviceAsFailedWithTimeout(t, db, device2, 1*time.Second)

		// Distribute messages immediately - device2 should be excluded
		device, err := selectDeviceForMessage(ctx, db, tenantID, "message-1")
		require.NoError(t, err)
		assert.Equal(t, device1, device.ID)

		// Wait for timeout
		time.Sleep(1500 * time.Millisecond)

		// Device2 should now be available again
		deviceCounts := make(map[string]int)
		for i := 0; i < 10; i++ {
			device, err := selectDeviceForMessage(ctx, db, tenantID, "message")
			require.NoError(t, err)
			deviceCounts[device.ID]++
		}

		// Both devices should receive messages
		assert.Greater(t, deviceCounts[device2], 0, "Device should be re-included after timeout")
	})
}

// TestConcurrentDistribution tests concurrent message distribution
func TestConcurrentDistribution(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	t.Run("Concurrent distribution is thread-safe", func(t *testing.T) {
		tenantID := createTestTenant(t, db, "concurrent-tenant")
		
		// Create 5 devices
		for i := 1; i <= 5; i++ {
			createTestDevice(t, db, tenantID, "device-"+string(rune(i)), 1)
		}

		configureDistributionStrategy(t, db, tenantID, "round_robin", nil)

		// Distribute 1000 messages concurrently
		var wg sync.WaitGroup
		deviceCounts := &sync.Map{}
		
		for i := 0; i < 1000; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				device, err := selectDeviceForMessage(ctx, db, tenantID, "message")
				if err == nil {
					count, _ := deviceCounts.LoadOrStore(device.ID, 0)
					deviceCounts.Store(device.ID, count.(int)+1)
				}
			}()
		}

		wg.Wait()

		// Verify all messages were distributed
		totalDistributed := 0
		deviceCounts.Range(func(key, value interface{}) bool {
			totalDistributed += value.(int)
			return true
		})

		assert.Equal(t, 1000, totalDistributed, "All messages should be distributed")

		// Verify fair distribution (each device should get ~200 messages, allow 10% variance)
		deviceCounts.Range(func(key, value interface{}) bool {
			count := value.(int)
			assert.InDelta(t, 200, count, 20, "Distribution should be relatively fair")
			return true
		})
	})
}

// TestDistributionMetrics tests distribution metrics tracking
func TestDistributionMetrics(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	t.Run("Track distribution metrics", func(t *testing.T) {
		tenantID := createTestTenant(t, db, "metrics-tenant")
		
		device1 := createTestDevice(t, db, tenantID, "device-1", 1)
		device2 := createTestDevice(t, db, tenantID, "device-2", 1)

		configureDistributionStrategy(t, db, tenantID, "round_robin", nil)

		// Distribute and track messages
		for i := 0; i < 100; i++ {
			device, err := selectDeviceForMessage(ctx, db, tenantID, "message")
			require.NoError(t, err)
			
			// Simulate message sent
			recordMessageSent(t, db, device.ID, true)
		}

		// Get metrics
		metrics1 := getDeviceMetrics(t, db, device1)
		metrics2 := getDeviceMetrics(t, db, device2)

		// Verify metrics
		assert.Equal(t, 50, metrics1.MessagesAssigned)
		assert.Equal(t, 50, metrics1.MessagesSent)
		assert.Equal(t, 50, metrics2.MessagesAssigned)
		assert.Equal(t, 50, metrics2.MessagesSent)
	})
}

// Helper functions

func createTestDevice(t *testing.T, db *DB, tenantID, deviceName string, priority int) string {
	t.Helper()
	// Create test device and return ID
	return "device-id"
}

func configureDistributionStrategy(t *testing.T, db *DB, tenantID, strategy string, config map[string]interface{}) {
	t.Helper()
	// Configure distribution strategy
}

func selectDeviceForMessage(ctx context.Context, db *DB, tenantID, message string) (*Device, error) {
	// Call actual distribution service
	return &Device{}, nil
}

func setDeviceLoad(t *testing.T, db *DB, deviceID string, load int) {
	t.Helper()
	// Set device load
}

func markDeviceAsFailed(t *testing.T, db *DB, deviceID string) {
	t.Helper()
	// Mark device as failed
}

func markDeviceAsFailedWithTimeout(t *testing.T, db *DB, deviceID string, timeout time.Duration) {
	t.Helper()
	// Mark device as failed with timeout
}

func recordMessageSent(t *testing.T, db *DB, deviceID string, success bool) {
	t.Helper()
	// Record message sent
}

func getDeviceMetrics(t *testing.T, db *DB, deviceID string) *DeviceMetrics {
	t.Helper()
	// Get device metrics
	return &DeviceMetrics{}
}

// Test types

type Device struct {
	ID       string
	Name     string
	Priority int
	Load     int
}

type DeviceMetrics struct {
	MessagesAssigned int
	MessagesSent     int
	SuccessRate      float64
}
