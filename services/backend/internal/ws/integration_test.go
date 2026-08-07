package ws

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/al-hudhud/backend/internal/config"
	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// initIntegrationDB connects to the real database if db.GormDB is not yet initialized.
// Skips the test if the database is unreachable.
func initIntegrationDB(t *testing.T) {
	if db.GormDB != nil {
		return
	}
	dsn := "host=localhost port=5433 user=postgres password=Ammar1983hmlh## dbname=al_hudhud_new sslmode=disable"
	gormDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		SkipDefaultTransaction:                   true,
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Skipf("Skipping integration test: cannot connect to database: %v", err)
	}
	db.GormDB = gormDB
}

// TestIntegration_DeliverPendingMessages_WEBUISource verifies that a message
// with source="WEB_UI" (NOT ACCOUNTING_SYSTEM) is delivered via WebSocket
// to a mobile device through deliverPendingMessagesToDevice.
//
// This validates the fix that removed the ACCOUNTING_SYSTEM filter so that
// ALL message sources (WEB_UI, API, etc.) get dispatched via WebSocket.
func TestIntegration_DeliverPendingMessages_WEBUISource(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	initIntegrationDB(t)

	ctx := context.Background()
	now := time.Now()

	// --- Generate unique IDs for this test run ---
	tenantID := uuid.New()
	userID := uuid.New()
	channelID := uuid.New()
	hardwareID := "int_test_android_" + uuid.New().String()[:8]

	// --- 1. Create tenant schema ---
	schemaName := db.GetSchemaName(tenantID)
	require.NoError(t, db.GormDB.Exec(
		fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", schemaName),
	).Error, "Create tenant schema")

	// --- 2. Create User in public schema (required for Tenant FK) ---
	user := models.User{
		ID:           userID,
		Email:        fmt.Sprintf("int-test-%s@alhudhud.com", uuid.New().String()[:8]),
		PasswordHash: "test_hash",
		UserType:     models.TenantType,
		IsActive:     true,
		IsVerified:   true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	require.NoError(t, db.GormDB.Create(&user).Error, "Create test user")

	// --- 3. Create Tenant in public schema ---
	tenant := models.Tenant{
		ID:               tenantID,
		UserID:           userID,
		BusinessName:     "Integration Test Tenant",
		ContactName:      "Test Contact",
		Email:            fmt.Sprintf("tenant-int-%s@alhudhud.com", uuid.New().String()[:8]),
		Phone:            "+966500000000",
		CountryCode:      "SA",
		Balance:          10000,
		Status:           models.TenantActive,
		CompanySerial:    "INT-TST-" + uuid.New().String()[:8],
		APIKeyAccounting: "int_acct_" + uuid.New().String()[:8],
		APIKeyPhone:      "int_phone_" + uuid.New().String()[:8],
		APIKeyWeb:        "int_web_" + uuid.New().String()[:8],
		IsActive:         true,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	require.NoError(t, db.GormDB.Create(&tenant).Error, "Create test tenant")

	// --- 4. Ensure tenant-scoped tables (PairedDevice, Message, DispatchChannel) exist ---
	require.NoError(t, db.GormDB.Transaction(func(tx *gorm.DB) error {
		require.NoError(t, tx.Exec(
			fmt.Sprintf("SET LOCAL search_path TO %s, public", schemaName),
		).Error)
		return tx.AutoMigrate(&models.PairedDevice{}, &models.Message{}, &models.DispatchChannel{})
	}), "Migrate tenant-scoped tables")

	// --- 5. Create PairedDevice in tenant schema ---
	pairKey := "test_pair_key_" + uuid.New().String()[:8]
	require.NoError(t, db.RunInTenantScope(tenantID, false, func(tx *gorm.DB) error {
		return tx.Create(&models.PairedDevice{
			ID:            channelID,
			TenantID:      tenantID,
			ChannelID:     &channelID,
			DeviceName:    "Test Android Device",
			HardwareID:    hardwareID,
			DevicePairKey: pairKey,
			PhoneNumber:   "+966500000000",
			Status:        "ONLINE",
			CreatedAt:     now,
			UpdatedAt:     now,
		}).Error
	}), "Create paired device")

	// --- 6. Create Message with source="WEB_UI" (the fix being verified) ---
	msgContent := "Integration test message from WEB_UI source"
	msgID := uuid.New()
	msgStatus := models.MessagePending
	require.NoError(t, db.RunInTenantScope(tenantID, false, func(tx *gorm.DB) error {
		return tx.Create(&models.Message{
			ID:          msgID,
			TenantID:    tenantID,
			Source:      "WEB_UI",
			SenderName:  "Integration Test",
			PhoneNumber: "+966501234567",
			Content:     msgContent,
			MessageType: models.MsgCatOther,
			Status:      msgStatus,
			CreatedAt:   now,
			UpdatedAt:   now,
		}).Error
	}), "Create test message with source=WEB_UI")

	// --- Cleanup: drop schema and remove public records ---
	t.Cleanup(func() {
		db.GormDB.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", schemaName))
		db.GormDB.Where("id = ?", tenantID).Delete(&models.Tenant{})
		db.GormDB.Where("id = ?", userID).Delete(&models.User{})
	})

	// ========================================================================
	// TEST 1: deliverPendingMessagesToDevice
	// ========================================================================
	t.Run("deliverPendingMessagesToDevice delivers WEB_UI message", func(t *testing.T) {
		hub := NewHub(config.WebSocketConfig{
			BufferSize:        1000,
			HeartbeatInterval: 30 * time.Second,
			WriteWait:         10 * time.Second,
			PongWait:          60 * time.Second,
		}, nil, nil)
		go hub.Run()
		t.Cleanup(func() { hub.cancel() })

		// Create a connection mimicking a paired mobile device.
		// DeviceID is a non-UUID string so isWebSession returns false,
		// which means deliverPendingMessagesToDevice will process it.
		// We set Conn to nil and read from Send channel directly.
		conn := &Connection{
			TenantID:  tenantID,
			ChannelID: channelID,
			DeviceID:  hardwareID,
			IsMobile:  true,
			Status:    StatusConnected,
			Send:      make(chan []byte, 1000),
			Hub:       hub,
		}
		hub.mu.Lock()
		hub.connections[ConnectionKey{TenantID: tenantID, ChannelID: channelID}] = conn
		hub.mu.Unlock()

		// Invoke the method that queries the DB and dispatches pending messages.
		hub.deliverPendingMessagesToDevice(conn)

		// Verify the WEB_UI message is received on the WebSocket send channel.
		select {
		case data := <-conn.Send:
			assert.NotNil(t, data, "WebSocket message should not be nil")
			msgStr := string(data)
			assert.Contains(t, msgStr, "MESSAGE_DELIVERY",
				"Message type should be MESSAGE_DELIVERY")
			assert.Contains(t, msgStr, msgContent,
				"Message content should be delivered")
			assert.Contains(t, msgStr, msgID.String(),
				"Message ID should be present in payload")
			t.Logf("SUCCESS: deliverPendingMessagesToDevice delivered: %s", msgStr)

		case <-time.After(5 * time.Second):
			t.Fatal("FAIL: No message received on WebSocket send channel within timeout")
		}
	})

	// ========================================================================
	// TEST 2: SendMessageToTenant (simulates what MessageActivationWorker does)
	// ========================================================================
	t.Run("SendMessageToTenant dispatches WEB_UI message to mobile devices", func(t *testing.T) {
		hub := NewHub(config.WebSocketConfig{
			BufferSize:        1000,
			HeartbeatInterval: 30 * time.Second,
			WriteWait:         10 * time.Second,
			PongWait:          60 * time.Second,
		}, nil, nil)
		go hub.Run()
		t.Cleanup(func() { hub.cancel() })

		conn := &Connection{
			TenantID:  tenantID,
			ChannelID: channelID,
			DeviceID:  hardwareID,
			IsMobile:  true,
			Status:    StatusConnected,
			Send:      make(chan []byte, 1000),
			Hub:       hub,
		}
		hub.mu.Lock()
		hub.connections[ConnectionKey{TenantID: tenantID, ChannelID: channelID}] = conn
		hub.mu.Unlock()

		// Simulate what MessageActivationWorker does:
		// Build a WebSocketMessage and call SendMessageToTenant.
		wsMsg := WebSocketMessage{
			Type:      "MESSAGE_DELIVERY",
			MessageID: msgID.String(),
			TenantID:  tenantID,
			Payload: map[string]interface{}{
				"message_id":   msgID.String(),
				"content":      msgContent,
				"sender_name":  "Integration Test",
				"phone_number": "+966501234567",
				"message_type": string(models.MsgCatOther),
				"timestamp":    time.Now(),
			},
			Timestamp: time.Now(),
		}

		err := hub.SendMessageToTenant(ctx, tenantID, wsMsg)
		require.NoError(t, err, "SendMessageToTenant should succeed when device is connected")

		select {
		case data := <-conn.Send:
			assert.NotNil(t, data)
			msgStr := string(data)
			assert.Contains(t, msgStr, "MESSAGE_DELIVERY")
			assert.Contains(t, msgStr, msgContent)
			assert.Contains(t, msgStr, msgID.String())
			t.Logf("SUCCESS: SendMessageToTenant delivered: %s", msgStr)

		case <-time.After(5 * time.Second):
			t.Fatal("FAIL: No message received via SendMessageToTenant")
		}
	})
}
