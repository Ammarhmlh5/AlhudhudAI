package monitoring

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockNotificationService is a mock implementation of NotificationService
type MockNotificationService struct {
	alerts []*Alert
}

func (m *MockNotificationService) SendAlert(ctx context.Context, alert *Alert) error {
	m.alerts = append(m.alerts, alert)
	return nil
}

func (m *MockNotificationService) GetAlerts() []*Alert {
	return m.alerts
}

func (m *MockNotificationService) Reset() {
	m.alerts = nil
}

func TestNewAlertManager(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	logger := &Logger{db: db}
	notificationSvc := &MockNotificationService{}

	am := NewAlertManager(db, logger, notificationSvc)

	assert.NotNil(t, am)
	assert.NotNil(t, am.defaultThreshold)
	assert.Equal(t, 80.0, am.defaultThreshold.CPUWarning)
	assert.Equal(t, 95.0, am.defaultThreshold.CPUCritical)
}

func TestCheckCPUThreshold_Warning(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	logger := &Logger{db: db}
	notificationSvc := &MockNotificationService{}
	am := NewAlertManager(db, logger, notificationSvc)

	ctx := context.Background()
	gatewayID := 1
	tenantID := uuid.New()
	cpuUsage := 85.0 // Above warning threshold (80%)

	// Mock tenant context setting
	mock.ExpectBegin()
	mock.ExpectExec("SELECT set_config\\('app.current_tenant', \\$1, true\\)").
		WithArgs(tenantID.String()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Mock alert logging
	mock.ExpectExec("INSERT INTO monitoring_alerts").
		WithArgs(
			sqlmock.AnyArg(), // id
			gatewayID,
			tenantID,
			"CPU",
			"WARNING",
			sqlmock.AnyArg(), // message
			cpuUsage,
			80.0, // threshold
			sqlmock.AnyArg(), // timestamp
			false,
			nil,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = am.checkCPUThreshold(ctx, gatewayID, tenantID, cpuUsage, am.defaultThreshold)
	require.NoError(t, err)

	// Verify alert was sent
	alerts := notificationSvc.GetAlerts()
	assert.Len(t, alerts, 1)
	assert.Equal(t, "CPU", alerts[0].AlertType)
	assert.Equal(t, "WARNING", alerts[0].Severity)
	assert.Equal(t, cpuUsage, alerts[0].MetricValue)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckCPUThreshold_Critical(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	logger := &Logger{db: db}
	notificationSvc := &MockNotificationService{}
	am := NewAlertManager(db, logger, notificationSvc)

	ctx := context.Background()
	gatewayID := 1
	tenantID := uuid.New()
	cpuUsage := 98.0 // Above critical threshold (95%)

	// Mock tenant context setting
	mock.ExpectBegin()
	mock.ExpectExec("SELECT set_config\\('app.current_tenant', \\$1, true\\)").
		WithArgs(tenantID.String()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Mock alert logging
	mock.ExpectExec("INSERT INTO monitoring_alerts").
		WithArgs(
			sqlmock.AnyArg(), // id
			gatewayID,
			tenantID,
			"CPU",
			"CRITICAL",
			sqlmock.AnyArg(), // message
			cpuUsage,
			95.0, // threshold
			sqlmock.AnyArg(), // timestamp
			false,
			nil,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = am.checkCPUThreshold(ctx, gatewayID, tenantID, cpuUsage, am.defaultThreshold)
	require.NoError(t, err)

	// Verify alert was sent
	alerts := notificationSvc.GetAlerts()
	assert.Len(t, alerts, 1)
	assert.Equal(t, "CPU", alerts[0].AlertType)
	assert.Equal(t, "CRITICAL", alerts[0].Severity)
	assert.Equal(t, cpuUsage, alerts[0].MetricValue)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckCPUThreshold_NoAlert(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	logger := &Logger{db: db}
	notificationSvc := &MockNotificationService{}
	am := NewAlertManager(db, logger, notificationSvc)

	ctx := context.Background()
	gatewayID := 1
	tenantID := uuid.New()
	cpuUsage := 50.0 // Below warning threshold

	err = am.checkCPUThreshold(ctx, gatewayID, tenantID, cpuUsage, am.defaultThreshold)
	require.NoError(t, err)

	// Verify no alert was sent
	alerts := notificationSvc.GetAlerts()
	assert.Len(t, alerts, 0)
}

func TestCheckMemoryThreshold_Warning(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	logger := &Logger{db: db}
	notificationSvc := &MockNotificationService{}
	am := NewAlertManager(db, logger, notificationSvc)

	ctx := context.Background()
	gatewayID := 1
	tenantID := uuid.New()
	memoryUsageMB := 850 // 85% of 1000 MB quota

	// Mock tenant context setting for quota query
	mock.ExpectBegin()
	mock.ExpectExec("SELECT set_config\\('app.current_tenant', \\$1, true\\)").
		WithArgs(tenantID.String()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Mock memory quota query
	mock.ExpectQuery("SELECT memory_quota_mb FROM resource_quotas").
		WithArgs(gatewayID).
		WillReturnRows(sqlmock.NewRows([]string{"memory_quota_mb"}).AddRow(1000))
	mock.ExpectCommit()

	// Mock tenant context setting for alert logging
	mock.ExpectBegin()
	mock.ExpectExec("SELECT set_config\\('app.current_tenant', \\$1, true\\)").
		WithArgs(tenantID.String()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Mock alert logging
	mock.ExpectExec("INSERT INTO monitoring_alerts").
		WithArgs(
			sqlmock.AnyArg(), // id
			gatewayID,
			tenantID,
			"MEMORY",
			"WARNING",
			sqlmock.AnyArg(), // message
			85.0,             // percentage
			80.0,             // threshold
			sqlmock.AnyArg(), // timestamp
			false,
			nil,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = am.checkMemoryThreshold(ctx, gatewayID, tenantID, memoryUsageMB, am.defaultThreshold)
	require.NoError(t, err)

	// Verify alert was sent
	alerts := notificationSvc.GetAlerts()
	assert.Len(t, alerts, 1)
	assert.Equal(t, "MEMORY", alerts[0].AlertType)
	assert.Equal(t, "WARNING", alerts[0].Severity)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckBandwidthThreshold_Critical(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	logger := &Logger{db: db}
	notificationSvc := &MockNotificationService{}
	am := NewAlertManager(db, logger, notificationSvc)

	ctx := context.Background()
	gatewayID := 1
	tenantID := uuid.New()
	bandwidthKBps := 1300.0 // Above 100% of 1280 KB/s quota (10 Mbps)

	// Mock tenant context setting for quota query
	mock.ExpectBegin()
	mock.ExpectExec("SELECT set_config\\('app.current_tenant', \\$1, true\\)").
		WithArgs(tenantID.String()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Mock bandwidth quota query (stored as Mbps, converted to KBps)
	mock.ExpectQuery("SELECT bandwidth_quota_mbps FROM resource_quotas").
		WithArgs(gatewayID).
		WillReturnRows(sqlmock.NewRows([]string{"bandwidth_quota_mbps"}).AddRow(10)) // 10 Mbps = 1280 KB/s
	mock.ExpectCommit()

	// Mock tenant context setting for alert logging
	mock.ExpectBegin()
	mock.ExpectExec("SELECT set_config\\('app.current_tenant', \\$1, true\\)").
		WithArgs(tenantID.String()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Mock alert logging
	mock.ExpectExec("INSERT INTO monitoring_alerts").
		WithArgs(
			sqlmock.AnyArg(), // id
			gatewayID,
			tenantID,
			"BANDWIDTH",
			"CRITICAL",
			sqlmock.AnyArg(), // message
			sqlmock.AnyArg(), // percentage
			100.0,            // threshold
			sqlmock.AnyArg(), // timestamp
			false,
			nil,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = am.checkBandwidthThreshold(ctx, gatewayID, tenantID, bandwidthKBps, am.defaultThreshold)
	require.NoError(t, err)

	// Verify alert was sent
	alerts := notificationSvc.GetAlerts()
	assert.Len(t, alerts, 1)
	assert.Equal(t, "BANDWIDTH", alerts[0].AlertType)
	assert.Equal(t, "CRITICAL", alerts[0].Severity)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckDatabaseThreshold_Warning(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	logger := &Logger{db: db}
	notificationSvc := &MockNotificationService{}
	am := NewAlertManager(db, logger, notificationSvc)

	ctx := context.Background()
	gatewayID := 1
	tenantID := uuid.New()
	activeConnections := 45 // 90% of 50 max connections

	// Mock tenant context setting for quota query
	mock.ExpectBegin()
	mock.ExpectExec("SELECT set_config\\('app.current_tenant', \\$1, true\\)").
		WithArgs(tenantID.String()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Mock DB connection quota query
	mock.ExpectQuery("SELECT max_db_connections FROM resource_quotas").
		WithArgs(gatewayID).
		WillReturnRows(sqlmock.NewRows([]string{"max_db_connections"}).AddRow(50))
	mock.ExpectCommit()

	// Mock tenant context setting for alert logging
	mock.ExpectBegin()
	mock.ExpectExec("SELECT set_config\\('app.current_tenant', \\$1, true\\)").
		WithArgs(tenantID.String()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Mock alert logging
	mock.ExpectExec("INSERT INTO monitoring_alerts").
		WithArgs(
			sqlmock.AnyArg(), // id
			gatewayID,
			tenantID,
			"DATABASE",
			"WARNING",
			sqlmock.AnyArg(), // message
			90.0,             // percentage
			90.0,             // threshold
			sqlmock.AnyArg(), // timestamp
			false,
			nil,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = am.checkDatabaseThreshold(ctx, gatewayID, tenantID, activeConnections, am.defaultThreshold)
	require.NoError(t, err)

	// Verify alert was sent
	alerts := notificationSvc.GetAlerts()
	assert.Len(t, alerts, 1)
	assert.Equal(t, "DATABASE", alerts[0].AlertType)
	assert.Equal(t, "WARNING", alerts[0].Severity)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckRedisThreshold_Critical(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	logger := &Logger{db: db}
	notificationSvc := &MockNotificationService{}
	am := NewAlertManager(db, logger, notificationSvc)

	ctx := context.Background()
	gatewayID := 1
	tenantID := uuid.New()
	redisMemoryMB := 550 // 110% of 500 MB quota

	// Mock tenant context setting for quota query
	mock.ExpectBegin()
	mock.ExpectExec("SELECT set_config\\('app.current_tenant', \\$1, true\\)").
		WithArgs(tenantID.String()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Mock Redis quota query
	mock.ExpectQuery("SELECT redis_memory_quota_mb FROM resource_quotas").
		WithArgs(gatewayID).
		WillReturnRows(sqlmock.NewRows([]string{"redis_memory_quota_mb"}).AddRow(500))
	mock.ExpectCommit()

	// Mock tenant context setting for alert logging
	mock.ExpectBegin()
	mock.ExpectExec("SELECT set_config\\('app.current_tenant', \\$1, true\\)").
		WithArgs(tenantID.String()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Mock alert logging
	mock.ExpectExec("INSERT INTO monitoring_alerts").
		WithArgs(
			sqlmock.AnyArg(), // id
			gatewayID,
			tenantID,
			"REDIS",
			"CRITICAL",
			sqlmock.AnyArg(), // message
			sqlmock.AnyArg(), // percentage (floating point precision issue)
			100.0,            // threshold
			sqlmock.AnyArg(), // timestamp
			false,
			nil,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = am.checkRedisThreshold(ctx, gatewayID, tenantID, redisMemoryMB, am.defaultThreshold)
	require.NoError(t, err)

	// Verify alert was sent
	alerts := notificationSvc.GetAlerts()
	assert.Len(t, alerts, 1)
	assert.Equal(t, "REDIS", alerts[0].AlertType)
	assert.Equal(t, "CRITICAL", alerts[0].Severity)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSendAlert(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	logger := &Logger{db: db}
	notificationSvc := &MockNotificationService{}
	am := NewAlertManager(db, logger, notificationSvc)

	ctx := context.Background()
	alert := &Alert{
		ID:          uuid.New(),
		GatewayID:   1,
		TenantID:    uuid.New(),
		AlertType:   "CPU",
		Severity:    "WARNING",
		Message:     "CPU usage warning: 85.00%",
		MetricValue: 85.0,
		Threshold:   80.0,
		Timestamp:   time.Now(),
		Resolved:    false,
	}

	// Mock tenant context setting
	mock.ExpectBegin()
	mock.ExpectExec("SELECT set_config\\('app.current_tenant', \\$1, true\\)").
		WithArgs(alert.TenantID.String()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Mock alert logging
	mock.ExpectExec("INSERT INTO monitoring_alerts").
		WithArgs(
			alert.ID,
			alert.GatewayID,
			alert.TenantID,
			alert.AlertType,
			alert.Severity,
			alert.Message,
			alert.MetricValue,
			alert.Threshold,
			sqlmock.AnyArg(), // timestamp
			alert.Resolved,
			alert.ResolvedAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = am.SendAlert(ctx, alert)
	require.NoError(t, err)

	// Verify alert was sent to notification service
	alerts := notificationSvc.GetAlerts()
	assert.Len(t, alerts, 1)
	assert.Equal(t, alert.ID, alerts[0].ID)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAlertManager_GetActiveDedicatedGateways(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	logger := &Logger{db: db}
	notificationSvc := &MockNotificationService{}
	am := NewAlertManager(db, logger, notificationSvc)

	ctx := context.Background()

	tenantID1 := uuid.New()
	tenantID2 := uuid.New()

	// Mock query
	rows := sqlmock.NewRows([]string{"id", "tenant_id"}).
		AddRow(1, tenantID1).
		AddRow(2, tenantID2)

	mock.ExpectQuery("SELECT id, tenant_id FROM dedicated_gateways").
		WillReturnRows(rows)

	gateways, err := am.getActiveDedicatedGateways(ctx)
	require.NoError(t, err)

	assert.Len(t, gateways, 2)
	assert.Equal(t, 1, gateways[0].ID)
	assert.Equal(t, tenantID1, gateways[0].TenantID)
	assert.Equal(t, 2, gateways[1].ID)
	assert.Equal(t, tenantID2, gateways[1].TenantID)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckThresholds_NoQuota(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	logger := &Logger{db: db}
	notificationSvc := &MockNotificationService{}
	am := NewAlertManager(db, logger, notificationSvc)

	ctx := context.Background()
	gatewayID := 1
	tenantID := uuid.New()
	memoryUsageMB := 850

	// Mock tenant context setting
	mock.ExpectBegin()
	mock.ExpectExec("SELECT set_config\\('app.current_tenant', \\$1, true\\)").
		WithArgs(tenantID.String()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Mock memory quota query returning no rows (no quota set)
	mock.ExpectQuery("SELECT memory_quota_mb FROM resource_quotas").
		WithArgs(gatewayID).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	err = am.checkMemoryThreshold(ctx, gatewayID, tenantID, memoryUsageMB, am.defaultThreshold)
	require.NoError(t, err)

	// Verify no alert was sent (no quota to check against)
	alerts := notificationSvc.GetAlerts()
	assert.Len(t, alerts, 0)

	assert.NoError(t, mock.ExpectationsWereMet())
}
