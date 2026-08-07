package notifications

import (
	"context"
	"time"

	"github.com/al-hudhud/backend/internal/ws"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

// PgxQuerier is the minimal pgx query interface needed to send notifications.
// It is satisfied by *pgxpool.Pool and pgx.Tx.
type PgxQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// SendTenantNotification creates a notification for the tenant owner and
// broadcasts it in real time via the WebSocket hub. It is best-effort:
// failures are logged and never returned, so callers can fire-and-forget
// from background jobs without risking the core operation.
func SendTenantNotification(ctx context.Context, q PgxQuerier, tenantID uuid.UUID, ntype, title, body string, data map[string]any, logger *zap.Logger) {
	if logger == nil {
		return
	}

	// Resolve the tenant owner user (notifications are per-user).
	var userID uuid.UUID
	err := q.QueryRow(ctx, `SELECT user_id FROM tenants WHERE id = $1`, tenantID).Scan(&userID)
	if err != nil {
		logger.Warn("Failed to resolve tenant owner for notification",
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err))
		return
	}

	if _, err := q.Exec(ctx, `
		INSERT INTO notifications (tenant_id, user_id, type, title, body, data)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, tenantID, userID, ntype, title, body, data); err != nil {
		logger.Warn("Failed to create notification",
			zap.String("tenant_id", tenantID.String()),
			zap.String("type", ntype),
			zap.Error(err))
		return
	}

	if ws.GlobalHub != nil {
		_ = ws.GlobalHub.BroadcastToTenant(ctx, tenantID, ws.WebSocketMessage{
			Type:      "notification",
			TenantID:  tenantID,
			Timestamp: time.Now(),
			Payload: map[string]any{
				"type":  ntype,
				"title": title,
				"body":  body,
				"data":  data,
			},
		})
	}

	logger.Info("Tenant notification sent",
		zap.String("tenant_id", tenantID.String()),
		zap.String("type", ntype),
		zap.String("title", title))
}
