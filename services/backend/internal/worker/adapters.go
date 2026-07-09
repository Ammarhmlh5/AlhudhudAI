package worker

import (
	"context"
	"time"

	"github.com/al-hudhud/backend/internal/balance"
	"github.com/al-hudhud/backend/internal/routing"
	"github.com/al-hudhud/backend/internal/ws"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
)

// BalanceManagerAdapter adapts balance.Manager to worker interface
type BalanceManagerAdapter struct {
	manager balance.Manager
}

// NewBalanceManagerAdapter creates a new balance manager adapter
func NewBalanceManagerAdapter(manager balance.Manager) *BalanceManagerAdapter {
	return &BalanceManagerAdapter{
		manager: manager,
	}
}

// DeductBalance adapts the balance manager's DeductBalance method
func (a *BalanceManagerAdapter) DeductBalance(ctx context.Context, balanceOp balance.BalanceOperation) (*balance.BalanceResult, error) {
	return a.manager.DeductBalance(ctx, balanceOp)
}

// DeductBalanceTx adapts the balance manager's DeductBalanceTx method
func (a *BalanceManagerAdapter) DeductBalanceTx(ctx context.Context, tx pgx.Tx, balanceOp balance.BalanceOperation) (*balance.BalanceResult, error) {
	return a.manager.DeductBalanceTx(ctx, tx, balanceOp)
}

// DeductBalanceWithTimeout adapts the balance manager's DeductBalanceWithTimeout method
func (a *BalanceManagerAdapter) DeductBalanceWithTimeout(ctx context.Context, op balance.BalanceOperation, timeout time.Duration) (*balance.BalanceResult, error) {
	return a.manager.DeductBalanceWithTimeout(ctx, op, timeout)
}

// RefundBalance adapts the balance manager's RefundBalance method
func (a *BalanceManagerAdapter) RefundBalance(ctx context.Context, balanceOp balance.BalanceOperation) (*balance.BalanceResult, error) {
	return a.manager.RefundBalance(ctx, balanceOp)
}

// RefundBalanceTx adapts the balance manager's RefundBalanceTx method
func (a *BalanceManagerAdapter) RefundBalanceTx(ctx context.Context, tx pgx.Tx, balanceOp balance.BalanceOperation) (*balance.BalanceResult, error) {
	return a.manager.RefundBalanceTx(ctx, tx, balanceOp)
}

// GetBalance adapts the balance manager's GetBalance method
func (a *BalanceManagerAdapter) GetBalance(ctx context.Context, tenantID uuid.UUID) (decimal.Decimal, error) {
	return a.manager.GetBalance(ctx, tenantID)
}

// CheckSufficient adapts the balance manager's CheckSufficient method
func (a *BalanceManagerAdapter) CheckSufficient(ctx context.Context, tenantID uuid.UUID, amount decimal.Decimal) (bool, error) {
	return a.manager.CheckSufficient(ctx, tenantID, amount)
}

// RoutingEngineAdapter adapts routing.Engine to worker interface
type RoutingEngineAdapter struct {
	engine *routing.Engine
}

// NewRoutingEngineAdapter creates a new routing engine adapter
func NewRoutingEngineAdapter(engine *routing.Engine) *RoutingEngineAdapter {
	return &RoutingEngineAdapter{
		engine: engine,
	}
}

// SelectChannel adapts the routing engine's SelectChannel method
func (a *RoutingEngineAdapter) SelectChannel(ctx context.Context, tenantID uuid.UUID) (*routing.Channel, error) {
	return a.engine.SelectChannel(ctx, tenantID)
}

// IncrementChannelLoad adapts the routing engine's IncrementChannelLoad method
func (a *RoutingEngineAdapter) IncrementChannelLoad(ctx context.Context, channelID uuid.UUID, tenantID uuid.UUID) error {
	return a.engine.IncrementChannelLoad(ctx, channelID, tenantID)
}

// VerifyChannelOnline adapts the routing engine's VerifyChannelOnline method
func (a *RoutingEngineAdapter) VerifyChannelOnline(ctx context.Context, channelID uuid.UUID, tenantID uuid.UUID) (bool, error) {
	return a.engine.VerifyChannelOnline(ctx, channelID, tenantID)
}

// WebSocketHubAdapter adapts ws.Hub to worker interface
type WebSocketHubAdapter struct {
	hub *ws.Hub
}

// NewWebSocketHubAdapter creates a new WebSocket hub adapter
func NewWebSocketHubAdapter(hub *ws.Hub) *WebSocketHubAdapter {
	return &WebSocketHubAdapter{
		hub: hub,
	}
}

// SendMessage adapts the WebSocket hub's SendMessage method
func (a *WebSocketHubAdapter) SendMessage(ctx context.Context, msg WebSocketMessage) error {
	wsMsg := ws.WebSocketMessage{
		Type:        msg.Type,
		MessageID:   msg.MessageID,
		TenantID:    msg.TenantID,
		ChannelID:   msg.ChannelID,
		PhoneNumber: msg.PhoneNumber,
		Content:     msg.Content,
		Timestamp:   msg.Timestamp,
	}

	return a.hub.SendMessage(ctx, wsMsg)
}

// BalanceOperationBuilder helps build balance operations
type BalanceOperationBuilder struct {
	tenantID  uuid.UUID
	messageID *uuid.UUID
	gatewayID *uuid.UUID
	amount    decimal.Decimal
	opType    balance.OperationType
	reason    string
	createdBy string
}

// NewBalanceOperationBuilder creates a new balance operation builder
func NewBalanceOperationBuilder() *BalanceOperationBuilder {
	return &BalanceOperationBuilder{}
}

// WithTenantID sets the tenant ID
func (b *BalanceOperationBuilder) WithTenantID(tenantID uuid.UUID) *BalanceOperationBuilder {
	b.tenantID = tenantID
	return b
}

// WithMessageID sets the message ID
func (b *BalanceOperationBuilder) WithMessageID(messageID *uuid.UUID) *BalanceOperationBuilder {
	b.messageID = messageID
	return b
}

// WithGatewayID sets the gateway ID
func (b *BalanceOperationBuilder) WithGatewayID(gatewayID *uuid.UUID) *BalanceOperationBuilder {
	b.gatewayID = gatewayID
	return b
}

// WithAmount sets the amount
func (b *BalanceOperationBuilder) WithAmount(amount decimal.Decimal) *BalanceOperationBuilder {
	b.amount = amount
	return b
}

// WithType sets the operation type
func (b *BalanceOperationBuilder) WithType(opType balance.OperationType) *BalanceOperationBuilder {
	b.opType = opType
	return b
}

// WithReason sets the reason
func (b *BalanceOperationBuilder) WithReason(reason string) *BalanceOperationBuilder {
	b.reason = reason
	return b
}

// WithCreatedBy sets the creator
func (b *BalanceOperationBuilder) WithCreatedBy(createdBy string) *BalanceOperationBuilder {
	b.createdBy = createdBy
	return b
}

// Build creates the balance operation
func (b *BalanceOperationBuilder) Build() balance.BalanceOperation {
	return balance.BalanceOperation{
		TenantID:  b.tenantID,
		MessageID: b.messageID,
		GatewayID: b.gatewayID,
		Amount:    b.amount,
		Type:      b.opType,
		Reason:    b.reason,
		CreatedBy: b.createdBy,
	}
}
