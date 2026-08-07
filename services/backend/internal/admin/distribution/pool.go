package distribution

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	// ErrDeviceNotFound is returned when a device is not found
	ErrDeviceNotFound = errors.New("device not found")
)

// DevicePoolManager manages the pool of available devices for message distribution
type DevicePoolManager struct {
	db             *gorm.DB
	failureHandler *DeviceFailureHandler
}

// NewDevicePoolManager creates a new device pool manager
func NewDevicePoolManager(db *gorm.DB, failureHandler *DeviceFailureHandler) *DevicePoolManager {
	return &DevicePoolManager{
		db:             db,
		failureHandler: failureHandler,
	}
}

// GetAvailableDevices fetches all available devices for a tenant
// Filters out offline, failed, and excluded devices
func (dpm *DevicePoolManager) GetAvailableDevices(ctx context.Context, tenantID uuid.UUID) ([]*Device, error) {
	var devices []*Device

	// Query devices with status 'online' only
	err := dpm.db.WithContext(ctx).
		Where("tenant_id = ? AND status = ?", tenantID, DeviceStatusOnline).
		Order("priority_level DESC, current_load ASC").
		Find(&devices).Error

	if err != nil {
		return nil, fmt.Errorf("failed to fetch devices: %w", err)
	}

	// Filter out excluded devices (those in failure handler)
	availableDevices := dpm.failureHandler.FilterAvailableDevices(devices)

	return availableDevices, nil
}

// GetAllDevices fetches all devices for a tenant regardless of status
func (dpm *DevicePoolManager) GetAllDevices(ctx context.Context, tenantID uuid.UUID) ([]*Device, error) {
	var devices []*Device

	err := dpm.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("created_at DESC").
		Find(&devices).Error

	if err != nil {
		return nil, fmt.Errorf("failed to fetch all devices: %w", err)
	}

	return devices, nil
}

// GetDeviceByID fetches a specific device by ID
func (dpm *DevicePoolManager) GetDeviceByID(ctx context.Context, deviceID uuid.UUID) (*Device, error) {
	var device Device

	err := dpm.db.WithContext(ctx).
		Where("id = ?", deviceID).
		First(&device).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDeviceNotFound
		}
		return nil, fmt.Errorf("failed to fetch device: %w", err)
	}

	return &device, nil
}

// UpdateDeviceLoad updates the current load for a device
// This should be called after assigning or completing a message
func (dpm *DevicePoolManager) UpdateDeviceLoad(ctx context.Context, deviceID uuid.UUID, newLoad int) error {
	result := dpm.db.WithContext(ctx).
		Model(&Device{}).
		Where("id = ?", deviceID).
		Updates(map[string]interface{}{
			"current_load": newLoad,
			"updated_at":   time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update device load: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrDeviceNotFound
	}

	return nil
}

// IncrementDeviceLoad increments the current load for a device by 1
func (dpm *DevicePoolManager) IncrementDeviceLoad(ctx context.Context, deviceID uuid.UUID) error {
	result := dpm.db.WithContext(ctx).
		Model(&Device{}).
		Where("id = ?", deviceID).
		Updates(map[string]interface{}{
			"current_load": gorm.Expr("current_load + ?", 1),
			"updated_at":   time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to increment device load: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrDeviceNotFound
	}

	return nil
}

// DecrementDeviceLoad decrements the current load for a device by 1
func (dpm *DevicePoolManager) DecrementDeviceLoad(ctx context.Context, deviceID uuid.UUID) error {
	result := dpm.db.WithContext(ctx).
		Model(&Device{}).
		Where("id = ? AND current_load > 0", deviceID).
		Updates(map[string]interface{}{
			"current_load": gorm.Expr("current_load - ?", 1),
			"updated_at":   time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to decrement device load: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrDeviceNotFound
	}

	return nil
}

// UpdateDeviceMetrics updates the metrics for a device
func (dpm *DevicePoolManager) UpdateDeviceMetrics(ctx context.Context, deviceID uuid.UUID, metrics DeviceMetricsUpdate) error {
	updates := make(map[string]interface{})

	if metrics.TotalMessagesSent != nil {
		updates["total_messages_sent"] = gorm.Expr("total_messages_sent + ?", *metrics.TotalMessagesSent)
	}

	if metrics.SuccessfulMessages != nil {
		updates["successful_messages"] = gorm.Expr("successful_messages + ?", *metrics.SuccessfulMessages)
	}

	if metrics.FailedMessages != nil {
		updates["failed_messages"] = gorm.Expr("failed_messages + ?", *metrics.FailedMessages)
	}

	if metrics.AverageProcessingTimeMs != nil {
		updates["average_processing_time_ms"] = *metrics.AverageProcessingTimeMs
	}

	updates["updated_at"] = time.Now()

	result := dpm.db.WithContext(ctx).
		Model(&Device{}).
		Where("id = ?", deviceID).
		Updates(updates)

	if result.Error != nil {
		return fmt.Errorf("failed to update device metrics: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrDeviceNotFound
	}

	return nil
}

// UpdateDeviceStatus updates the status of a device
func (dpm *DevicePoolManager) UpdateDeviceStatus(ctx context.Context, deviceID uuid.UUID, status string) error {
	result := dpm.db.WithContext(ctx).
		Model(&Device{}).
		Where("id = ?", deviceID).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update device status: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrDeviceNotFound
	}

	return nil
}

// MarkDeviceAsOnline marks a device as online and updates last_seen timestamp
func (dpm *DevicePoolManager) MarkDeviceAsOnline(ctx context.Context, deviceID uuid.UUID) error {
	result := dpm.db.WithContext(ctx).
		Model(&Device{}).
		Where("id = ?", deviceID).
		Updates(map[string]interface{}{
			"status":     DeviceStatusOnline,
			"last_seen":  time.Now(),
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to mark device as online: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrDeviceNotFound
	}

	return nil
}

// MarkDeviceAsFailed marks a device as failed and updates last_failure timestamp
// Also triggers the failure handler to exclude the device
func (dpm *DevicePoolManager) MarkDeviceAsFailed(ctx context.Context, deviceID uuid.UUID) error {
	now := time.Now()

	result := dpm.db.WithContext(ctx).
		Model(&Device{}).
		Where("id = ?", deviceID).
		Updates(map[string]interface{}{
			"status":       DeviceStatusFailed,
			"last_failure": now,
			"updated_at":   now,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to mark device as failed: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrDeviceNotFound
	}

	// Trigger failure handler to exclude device
	dpm.failureHandler.OnDeviceFailure(deviceID)

	return nil
}

// SaveDistributionMetrics saves distribution metrics for a device
func (dpm *DevicePoolManager) SaveDistributionMetrics(ctx context.Context, metrics *DistributionMetrics) error {
	err := dpm.db.WithContext(ctx).Create(metrics).Error
	if err != nil {
		return fmt.Errorf("failed to save distribution metrics: %w", err)
	}

	return nil
}

// GetDistributionMetrics retrieves distribution metrics for a device within a time range
func (dpm *DevicePoolManager) GetDistributionMetrics(ctx context.Context, deviceID uuid.UUID, startTime, endTime time.Time) ([]*DistributionMetrics, error) {
	var metrics []*DistributionMetrics

	err := dpm.db.WithContext(ctx).
		Where("device_id = ? AND period_start >= ? AND period_end <= ?", deviceID, startTime, endTime).
		Order("period_start DESC").
		Find(&metrics).Error

	if err != nil {
		return nil, fmt.Errorf("failed to fetch distribution metrics: %w", err)
	}

	return metrics, nil
}

// GetTenantMetrics retrieves aggregated distribution metrics for all devices of a
// tenant within a time range, joined with device names from phone_devices.
func (dpm *DevicePoolManager) GetTenantMetrics(ctx context.Context, tenantID uuid.UUID, startTime, endTime time.Time) ([]*TenantDeviceMetric, error) {
	var metrics []*TenantDeviceMetric

	err := dpm.db.WithContext(ctx).Raw(`
		SELECT
			dm.device_id AS device_id,
			COALESCE(pd.device_name, '') AS device_name,
			COALESCE(SUM(dm.messages_assigned), 0) AS messages_assigned,
			COALESCE(SUM(dm.messages_sent), 0) AS messages_sent,
			COALESCE(SUM(dm.messages_sent), 0) AS successful_messages,
			COALESCE(AVG(dm.average_processing_time_ms), 0)::int AS average_processing_time_ms
		FROM distribution_metrics dm
		LEFT JOIN phone_devices pd ON pd.id = dm.device_id
		WHERE dm.tenant_id = $1 AND dm.period_start >= $2 AND dm.period_end <= $3
		GROUP BY dm.device_id, pd.device_name
		ORDER BY pd.device_name
	`, tenantID, startTime, endTime).Scan(&metrics).Error

	if err != nil {
		return nil, fmt.Errorf("failed to fetch tenant distribution metrics: %w", err)
	}

	return metrics, nil
}

// GetDeviceCount returns the total number of devices for a tenant
func (dpm *DevicePoolManager) GetDeviceCount(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	var count int64

	err := dpm.db.WithContext(ctx).
		Model(&Device{}).
		Where("tenant_id = ?", tenantID).
		Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("failed to count devices: %w", err)
	}

	return count, nil
}

// GetOnlineDeviceCount returns the number of online devices for a tenant
func (dpm *DevicePoolManager) GetOnlineDeviceCount(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	var count int64

	err := dpm.db.WithContext(ctx).
		Model(&Device{}).
		Where("tenant_id = ? AND status = ?", tenantID, DeviceStatusOnline).
		Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("failed to count online devices: %w", err)
	}

	return count, nil
}

// DeviceMetricsUpdate represents metrics updates for a device
type DeviceMetricsUpdate struct {
	TotalMessagesSent       *int64
	SuccessfulMessages      *int64
	FailedMessages          *int64
	AverageProcessingTimeMs *int
}

// Helper function to create nullable int64 pointer
func Int64Ptr(v int64) *int64 {
	return &v
}

// Helper function to create nullable int pointer
func IntPtr(v int) *int {
	return &v
}

// Helper function to create nullable time pointer
func TimePtr(v time.Time) *time.Time {
	return &v
}

// Helper function to create nullable sql.NullTime
func NullTime(v time.Time) sql.NullTime {
	return sql.NullTime{Time: v, Valid: true}
}
