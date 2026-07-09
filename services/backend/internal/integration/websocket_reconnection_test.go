package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/al-hudhud/backend/internal/config"
	"github.com/al-hudhud/backend/internal/database"
	"github.com/al-hudhud/backend/internal/ws"
	goredis "github.com/redis/go-redis/v9"
	"github.com/gorilla/websocket"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestWebSocketReconnectionFlow(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. Setup real Database and Redis for integration test
	cfg := &config.Config{}
	cfg.Database.Host = "localhost"
	cfg.Database.Port = "5432"
	cfg.Database.User = "postgres"
	cfg.Database.Password = "postgres"
	cfg.Database.Name = "sms_gateway_test"
	cfg.Database.MaxOpenConns = 10
	cfg.Database.MaxIdleConns = 5
	cfg.Database.ConnMaxLifetime = time.Hour
	cfg.Database.ConnMaxIdleTime = 30 * time.Minute

	db, err := database.NewDatabase(cfg, zap.NewNop())
	require.NoError(t, err)
	defer db.Close()

	redisClient := goredis.NewClient(&goredis.Options{
		Addr: "localhost:6379",
		DB:   1,
	})
	require.NoError(t, redisClient.Ping(ctx).Err())
	defer redisClient.Close()

	// 2. Setup WebSocket Hub
	wsHub := ws.NewHub(config.WebSocketConfig{
		BufferSize:        1000,
		HeartbeatInterval: 30 * time.Second,
		WriteWait:         10 * time.Second,
		PongWait:          60 * time.Second,
	}, nil, nil)
	go wsHub.Run()

	// 3. Setup test data
	cleanupTestData(t, ctx, db, redisClient)
	tenantID := createTestTenant(t, ctx, db, decimal.NewFromFloat(100.0))
	channelID := createTestChannel(t, ctx, db, tenantID)

	// 4. Create WebSocket server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws.ServeWs(wsHub, w, r, tenantID.String(), channelID.String())
	}))
	defer server.Close()

	// 5. Connect initial client
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn1.Close()

	// 6. Verify connection in Hub
	status, err := wsHub.GetConnectionStatus(ctx, tenantID, channelID)
	require.NoError(t, err)
	assert.Equal(t, string(ws.StatusConnected), status)

	// 7. Simulate disconnection
	conn1.Close()
	time.Sleep(100 * time.Millisecond)

	// 8. Verify offline status
	status, err = wsHub.GetConnectionStatus(ctx, tenantID, channelID)
	require.NoError(t, err)
	assert.Equal(t, string(ws.StatusDisconnected), status)

	// 9. Reconnect
	conn2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn2.Close()

	// 10. Verify online again
	status, err = wsHub.GetConnectionStatus(ctx, tenantID, channelID)
	require.NoError(t, err)
	assert.Equal(t, string(ws.StatusConnected), status)
}
