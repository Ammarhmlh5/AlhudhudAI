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

func TestNewBalanceMonitor(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	alertManager := &AlertManager{db: db}
	monitor := NewBalanceMonitor(db, alertManager)

	assert.NotNil(t, monitor)
	assert.NotNil(t, monitor.db)
	assert.NotNil(t, monitor.alertManager)
	assert.NotNil(t, monitor.telecomClients)
	assert.Equal(t, 1*time.Hour, monitor.checkInterval)
}

func TestRegisterTelecomClient(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	monitor := NewBalanceMonitor(db, nil)
	mockClient := &MockTelecomClient{MockBalance: 1000.0}

	monitor.RegisterTelecomClient("YemenMobile", mockClient)

	assert.Len(t, monitor.telecomClients, 1)
	assert.NotNil(t, monitor.telecomClients["YemenMobile"])
}

func TestSetCheckInterval(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	monitor := NewBalanceMonitor(db, nil)
	newInterval := 30 * time.Minute

	monitor.SetCheckInterval(newInterval)

	assert.Equal(t, newInterval, monitor.checkInterval)
}

func TestCheckTelecomBalance_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()
	gatewayID := 1
	tenantID := uuid.New()
	currentBalance := 1500.0
	threshold := 1000.0

	// Mock set tenant context
	mock.ExpectBegin()
	mock.ExpectExec("SELECT set_config\\('app.current_tenant', \\$1, true\\)").
		WithArgs(tenantID.String()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Mock get gateway info
	rows := sqlmock.NewRows([]string{
		"id", "tenant_id", "gateway_name", "telecom_provider",
		"telecom_balance", "balance_alert_threshold", "last_balance_check", "balance_check_enabled",
	}).AddRow(
		gatewayID, tenantID, "Test Gateway", "YemenMobile",
		500.0, threshold, time.Now().Add(-2*time.Hour), true,
	)
	mock.ExpectQuery("SELECT (.+) FROM dedicated_gateways WHERE id").
		WithArgs(gatewayID, tenantID).
		WillReturnRows(rows)
	mock.ExpectCommit()

	// Mock get credentials
	mock.ExpectBegin()
	mock.ExpectExec("SELECT set_config\\('app.current_tenant', \\$1, true\\)").
		WithArgs(tenantID.String()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	credRows := sqlmock.NewRows([]string{
		"telecom_provider", "telecom_api_endpoint", "telecom_username",
		"telecom_password_encrypted", "telecom_api_key_encrypted",
	}).AddRow(
		"YemenMobile", "https://api.yemenmobile.com", "testuser",
		"encrypted_password", "encrypted_api_key",
	)
	mock.ExpectQuery("SELECT (.+) FROM dedicated_gateways WHERE id").
		WithArgs(gatewayID, tenantID).
		WillReturnRows(credRows)
	mock.ExpectCommit()

	// Mock update balance
	mock.ExpectBegin()
	mock.ExpectExec("SELECT set_config\\('app.current_tenant', \\$1, true\\)").
		WithArgs(tenantID.String()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	mock.ExpectExec("UPDATE dedicated_gateways SET telecom_balance").
		WithArgs(currentBalance, sqlmock.AnyArg(), gatewayID, tenantID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	// Create monitor with mock client
	alertManager := &AlertManager{db: db}
	monitor := NewBalanceMonitor(db, alertManager)
	mockClient := &MockTelecomClient{MockBalance: currentBalance}
	monitor.RegisterTelecomClient("YemenMobile", mockClient)

	// Execute
	err = monitor.CheckTelecomBalance(ctx, gatewayID, tenantID)

	// Assert
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckTelecomBalance_BalanceCheckDisabled(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()
	gatewayID := 1
	tenantID := uuid.New()

	// Mock set tenant context
	mock.ExpectBegin()
	mock.ExpectExec("SELECT set_config\\('app.current_tenant', \\$1, true\\)").
		WithArgs(tenantID.String()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Mock get gateway info with balance_check_enabled = false
	rows := sqlmock.NewRows([]string{
		"id", "tenant_id", "gateway_name", "telecom_provider",
		"telecom_balance", "balance_alert_threshold", "last_balance_check", "balance_check_enabled",
	}).AddRow(
		gatewayID, tenantID, "Test Gateway", "YemenMobile",
		500.0, 1000.0, time.Now(), false, // balance_check_enabled = false
	)
	mock.ExpectQuery("SELECT (.+) FROM dedicated_gateways WHERE id").
		WithArgs(gatewayID, tenantID).
		WillReturnRows(rows)
	mock.ExpectCommit()

	// Create monitor
	monitor := NewBalanceMonitor(db, nil)

	// Execute
	err = monitor.CheckTelecomBalance(ctx, gatewayID, tenantID)

	// Assert - should return nil without checking balance
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckTelecomBalance_NoTelecomClient(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()
	gatewayID := 1
	tenantID := uuid.New()

	// Mock set tenant context
	mock.ExpectBegin()
	mock.ExpectExec("SELECT set_config\\('app.current_tenant', \\$1, true\\)").
		WithArgs(tenantID.String()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Mock get gateway info
	rows := sqlmock.NewRows([]string{
		"id", "tenant_id", "gateway_name", "telecom_provider",
		"telecom_balance", "balance_alert_threshold", "last_balance_check", "balance_check_enabled",
	}).AddRow(
		gatewayID, tenantID, "Test Gateway", "UnknownProvider",
		500.0, 1000.0, time.Now(), true,
	)
	mock.ExpectQuery("SELECT (.+) FROM dedicated_gateways WHERE id").
		WithArgs(gatewayID, tenantID).
		WillReturnRows(rows)
	mock.ExpectCommit()

	// Create monitor without registering client for UnknownProvider
	monitor := NewBalanceMonitor(db, nil)

	// Execute
	err = monitor.CheckTelecomBalance(ctx, gatewayID, tenantID)

	// Assert - should return error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no telecom client registered")
}

func TestCheckTelecomBalance_LowBalanceAlert(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()
	gatewayID := 1
	tenantID := uuid.New()
	currentBalance := 500.0  // Below threshold
	threshold := 1000.0

	// Mock set tenant context
	mock.ExpectBegin()
	mock.ExpectExec("SELECT set_config\\('app.current_tenant', \\$1, true\\)").
		WithArgs(tenantID.String()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Mock get gateway info
	rows := sqlmock.NewRows([]string{
		"id", "tenant_id", "gateway_name", "telecom_provider",
		"telecom_balance", "balance_alert_threshold", "last_balance_check", "balance_check_enabled",
	}).AddRow(
		gatewayID, tenantID, "Test Gateway", "YemenMobile",
		800.0, threshold, time.Now().Add(-2*time.Hour), true,
	)
	mock.ExpectQuery("SELECT (.+) FROM dedicated_gateways WHERE id").
		WithArgs(gatewayID, tenantID).
		WillReturnRows(rows)
	mock.ExpectCommit()

	// Mock get credentials
	mock.ExpectBegin()
	mock.ExpectExec("SELECT set_config\\('app.current_tenant', \\$1, true\\)").
		WithArgs(tenantID.String()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	credRows := sqlmock.NewRows([]string{
		"telecom_provider", "telecom_api_endpoint", "telecom_username",
		"telecom_password_encrypted", "telecom_api_key_encrypted",
	}).AddRow(
		"YemenMobile", "https://api.yemenmobile.com", "testuser",
		"encrypted_password", "encrypted_api_key",
	)
	mock.ExpectQuery("SELECT (.+) FROM dedicated_gateways WHERE id").
		WithArgs(gatewayID, tenantID).
		WillReturnRows(credRows)
	mock.ExpectCommit()

	// Mock update balance
	mock.ExpectBegin()
	mock.ExpectExec("SELECT set_config\\('app.current_tenant', \\$1, true\\)").
		WithArgs(tenantID.String()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	mock.ExpectExec("UPDATE dedicated_gateways SET telecom_balance").
		WithArgs(currentBalance, sqlmock.AnyArg(), gatewayID, tenantID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	// Mock alert logging
	mock.ExpectBegin()
	mock.ExpectExec("SELECT set_config\\('app.current_tenant', \\$1, true\\)").
		WithArgs(tenantID.String()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	mock.ExpectExec("INSERT INTO monitoring_alerts").
		WithArgs(
			sqlmock.AnyArg(), // id
			gatewayID,
			tenantID,
			"TELECOM_BALANCE",
			"WARNING",
			sqlmock.AnyArg(), // message
			currentBalance,
			threshold,
			sqlmock.AnyArg(), // timestamp
			false,
			nil,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	// Create monitor with mock client
	alertManager := &AlertManager{db: db}
	monitor := NewBalanceMonitor(db, alertManager)
	mockClient := &MockTelecomClient{MockBalance: currentBalance}
	monitor.RegisterTelecomClient("YemenMobile", mockClient)

	// Execute
	err = monitor.CheckTelecomBalance(ctx, gatewayID, tenantID)

	// Assert
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateBalance(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()
	gatewayID := 1
	tenantID := uuid.New()
	balance := 1500.0

	// Mock set tenant context
	mock.ExpectBegin()
	mock.ExpectExec("SELECT set_config\\('app.current_tenant', \\$1, true\\)").
		WithArgs(tenantID.String()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Mock update
	mock.ExpectExec("UPDATE dedicated_gateways SET telecom_balance").
		WithArgs(balance, sqlmock.AnyArg(), gatewayID, tenantID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	// Create monitor
	monitor := NewBalanceMonitor(db, nil)

	// Execute
	err = monitor.updateBalance(ctx, gatewayID, tenantID, balance)

	// Assert
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateBalance_GatewayNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()
	gatewayID := 999
	tenantID := uuid.New()
	balance := 1500.0

	// Mock set tenant context
	mock.ExpectBegin()
	mock.ExpectExec("SELECT set_config\\('app.current_tenant', \\$1, true\\)").
		WithArgs(tenantID.String()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Mock update with no rows affected
	mock.ExpectExec("UPDATE dedicated_gateways SET telecom_balance").
		WithArgs(balance, sqlmock.AnyArg(), gatewayID, tenantID).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectRollback()

	// Create monitor
	monitor := NewBalanceMonitor(db, nil)

	// Execute
	err = monitor.updateBalance(ctx, gatewayID, tenantID, balance)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no gateway found")
}

func TestGetActiveGatewaysForBalanceCheck(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()
	tenantID1 := uuid.New()
	tenantID2 := uuid.New()

	// Mock query
	rows := sqlmock.NewRows([]string{
		"id", "tenant_id", "gateway_name", "telecom_provider",
		"telecom_balance", "balance_alert_threshold", "last_balance_check", "balance_check_enabled",
	}).
		AddRow(1, tenantID1, "Gateway 1", "YemenMobile", 1000.0, 500.0, time.Now().Add(-2*time.Hour), true).
		AddRow(2, tenantID2, "Gateway 2", "Sabafon", 2000.0, 1000.0, time.Now().Add(-1*time.Hour), true)

	mock.ExpectQuery("SELECT (.+) FROM dedicated_gateways WHERE status").
		WillReturnRows(rows)

	// Create monitor
	monitor := NewBalanceMonitor(db, nil)

	// Execute
	gateways, err := monitor.getActiveGatewaysForBalanceCheck(ctx)

	// Assert
	assert.NoError(t, err)
	assert.Len(t, gateways, 2)
	assert.Equal(t, 1, gateways[0].GatewayID)
	assert.Equal(t, "Gateway 1", gateways[0].GatewayName)
	assert.Equal(t, 2, gateways[1].GatewayID)
	assert.Equal(t, "Gateway 2", gateways[1].GatewayName)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetGatewayInfo(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()
	gatewayID := 1
	tenantID := uuid.New()

	// Mock set tenant context
	mock.ExpectBegin()
	mock.ExpectExec("SELECT set_config\\('app.current_tenant', \\$1, true\\)").
		WithArgs(tenantID.String()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Mock query
	rows := sqlmock.NewRows([]string{
		"id", "tenant_id", "gateway_name", "telecom_provider",
		"telecom_balance", "balance_alert_threshold", "last_balance_check", "balance_check_enabled",
	}).AddRow(
		gatewayID, tenantID, "Test Gateway", "YemenMobile",
		1500.0, 1000.0, time.Now(), true,
	)

	mock.ExpectQuery("SELECT (.+) FROM dedicated_gateways WHERE id").
		WithArgs(gatewayID, tenantID).
		WillReturnRows(rows)
	mock.ExpectCommit()

	// Create monitor
	monitor := NewBalanceMonitor(db, nil)

	// Execute
	info, err := monitor.getGatewayInfo(ctx, gatewayID, tenantID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, info)
	assert.Equal(t, gatewayID, info.GatewayID)
	assert.Equal(t, "Test Gateway", info.GatewayName)
	assert.Equal(t, "YemenMobile", info.TelecomProvider)
	assert.Equal(t, 1500.0, info.CurrentBalance)
	assert.Equal(t, 1000.0, info.BalanceAlertThreshold)
	assert.True(t, info.BalanceCheckEnabled)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetGatewayInfo_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()
	gatewayID := 999
	tenantID := uuid.New()

	// Mock set tenant context
	mock.ExpectBegin()
	mock.ExpectExec("SELECT set_config\\('app.current_tenant', \\$1, true\\)").
		WithArgs(tenantID.String()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Mock query with no rows
	mock.ExpectQuery("SELECT (.+) FROM dedicated_gateways WHERE id").
		WithArgs(gatewayID, tenantID).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	// Create monitor
	monitor := NewBalanceMonitor(db, nil)

	// Execute
	info, err := monitor.getGatewayInfo(ctx, gatewayID, tenantID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, info)
	assert.Contains(t, err.Error(), "gateway not found")
}

func TestMockTelecomClient(t *testing.T) {
	ctx := context.Background()
	creds := TelecomCredentials{
		Provider:    "YemenMobile",
		APIEndpoint: "https://api.yemenmobile.com",
		Username:    "testuser",
		Password:    "testpass",
		APIKey:      "testapikey",
	}

	t.Run("Success", func(t *testing.T) {
		mockClient := &MockTelecomClient{MockBalance: 1500.0}
		balance, err := mockClient.CheckBalance(ctx, creds)

		assert.NoError(t, err)
		assert.Equal(t, 1500.0, balance)
	})

	t.Run("Error", func(t *testing.T) {
		mockClient := &MockTelecomClient{MockError: assert.AnError}
		balance, err := mockClient.CheckBalance(ctx, creds)

		assert.Error(t, err)
		assert.Equal(t, 0.0, balance)
	})
}
