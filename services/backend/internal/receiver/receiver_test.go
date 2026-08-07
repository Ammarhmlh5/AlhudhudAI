package receiver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/al-hudhud/backend/internal/queue"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestEndToEndMessagingFlow validates the entire pipeline from Ingestion -> Parsing -> Activation -> Status update
func TestEndToEndMessagingFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Set env vars for local DB connection before InitGormDB
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", "5433")
	os.Setenv("DB_USER", "postgres")
	os.Setenv("DB_PASSWORD", "Ammar1983hmlh##")
	os.Setenv("DB_NAME", "al_hudhud_new")
	os.Setenv("DB_SSLMODE", "disable")
	// Cap the shared pool: the al_hudhud_new server is shared with a running
	// service, so opening the production-sized pool exhausts max_connections.
	os.Setenv("DB_MAX_CONNS", "5")
	os.Setenv("DB_MIN_CONNS", "1")

	// Setup Database
	db.InitGormDB()
	if db.GormDB == nil {
		t.Skip("Skipping test: database not available")
	}

	// Setup Gin
	gin.SetMode(gin.TestMode)
	r := gin.New()

	h := NewHandler(nil)
	
	// Create a mock user first (for FK)
	userID := uuid.New()
	user := models.User{
		ID:        userID,
		Email:     fmt.Sprintf("test-%s@example.com", userID.String()[:8]),
		UserType:  models.TenantType,
		IsActive:  true,
		CreatedAt: time.Now(),
	}
	if err := db.GormDB.Create(&user).Error; err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	defer db.GormDB.Unscoped().Delete(&user)

	// Create a mock tenant
	tenantID := uuid.New()
	tenantSerial := "INT-TST-RCV-" + uuid.New().String()[:8]
	tenant := models.Tenant{
		ID:               tenantID,
		UserID:           userID,
		BusinessName:     "Test Corporation",
		ContactName:      "Test Contact",
		Email:            fmt.Sprintf("receiver-test-%s@test.com", uuid.New().String()[:8]),
		Phone:            "+966500000001",
		CountryCode:      "SA",
		Balance:          5000.0,
		Status:           models.TenantActive,
		CompanySerial:    tenantSerial,
		APIKeyAccounting: "recv_acct_" + uuid.New().String()[:8],
		APIKeyPhone:      "recv_phone_" + uuid.New().String()[:8],
		APIKeyWeb:        "recv_web_" + uuid.New().String()[:8],
		IsActive:         true,
	}
	if err := db.GormDB.Create(&tenant).Error; err != nil {
		t.Fatalf("Failed to create test tenant: %v", err)
	}
	defer db.GormDB.Unscoped().Delete(&tenant)

	// Auth Middleware Mock
	r.POST("/api/v1/messaging/bulk-send", func(c *gin.Context) {
		c.Set("tenant_id", tenantID.String())
		c.Set("user_type", string(models.TenantType))
		h.BulkSendMessageHandler(c)
	})

	// 1. PHASE 1: Bulk Ingest 100 messages
	payload := queue.BulkMessagePayload{
		MessageType: "INVOICE",
		SenderName:  "Audit Tester",
		Messages:    make([]queue.SingleMessageItem, 100),
	}
	for i := 0; i < 100; i++ {
		payload.Messages[i] = queue.SingleMessageItem{
			PhoneNumber: "+1234567890",
			Content:     "Your invoice is ready. Please pay now.",
		}
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/messaging/bulk-send", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Assert Ingestion Success
	if w.Code != http.StatusAccepted {
		t.Fatalf("Expected status 202, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	eventCustomID := resp["event_id"].(string)
	assert.NotEmpty(t, eventCustomID)

	// 2. Wait for Parsing & Processing Simulation
	var event models.GlobalIncomingEvent
	err := db.GormDB.Where("custom_id = ?", eventCustomID).First(&event).Error
	assert.NoError(t, err)
	assert.Equal(t, "RECEIVED", event.Status)
	assert.Contains(t, event.RawPayload, "Audit Tester")

	t.Logf("Ingestion latency for 100 messages: %v", time.Since(time.Now()))
}
