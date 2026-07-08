package redis

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// AppNamespace is the global namespace prefix for all Redis keys
const AppNamespace = "alhudhud"

// Key generation helpers with unified namespace for tenant isolation

// MessageQueueKey returns Redis key for message queue per tenant
// Format: alhudhud:messages:pending:{tenant_id}
func MessageQueueKey(tenantID uuid.UUID) string {
	return fmt.Sprintf("%s:messages:pending:%s", AppNamespace, tenantID.String())
}

// MessageCountKey returns Redis key for message count per tenant
// Format: alhudhud:messages:count:{tenant_id}
func MessageCountKey(tenantID uuid.UUID) string {
	return fmt.Sprintf("%s:messages:count:%s", AppNamespace, tenantID.String())
}

// RateLimitKey returns Redis key for rate limiting per tenant
// Format: alhudhud:rate_limit:{tenant_id}
func RateLimitKey(tenantID uuid.UUID) string {
	return fmt.Sprintf("%s:rate_limit:%s", AppNamespace, tenantID.String())
}

// ChannelCacheKey returns Redis key for channel cache per tenant
// Format: alhudhud:channels:cache:{tenant_id}
func ChannelCacheKey(tenantID uuid.UUID) string {
	return fmt.Sprintf("%s:channels:cache:%s", AppNamespace, tenantID.String())
}

// TariffCacheKey returns Redis key for tariff cache per tenant
// Format: alhudhud:tariff:cache:{tenant_id}
func TariffCacheKey(tenantID uuid.UUID) string {
	return fmt.Sprintf("%s:tariff:cache:%s", AppNamespace, tenantID.String())
}

// IdempotencyKey returns Redis key for idempotency check
// Format: alhudhud:idempotency:{tenant_id}:{message_id}:{operation}
func IdempotencyKey(tenantID uuid.UUID, messageID, operation string) string {
	return fmt.Sprintf("%s:idempotency:%s:%s:%s", AppNamespace, tenantID.String(), messageID, operation)
}

// ActiveTenantsKey returns Redis key for set of active tenants
// Format: alhudhud:tenants:active
func ActiveTenantsKey() string {
	return fmt.Sprintf("%s:tenants:active", AppNamespace)
}

// TenantCacheKey returns Redis key for tenant data cache
// Format: alhudhud:tenant:{tenant_id}
func TenantCacheKey(tenantID uuid.UUID) string {
	return fmt.Sprintf("%s:tenant:%s", AppNamespace, tenantID.String())
}

// TenantUserCacheKey returns Redis key for tenant lookup by user ID
// Format: alhudhud:tenant:user:{user_id}
func TenantUserCacheKey(userID uuid.UUID) string {
	return fmt.Sprintf("%s:tenant:user:%s", AppNamespace, userID.String())
}

// TenantAPIKeyCacheKey returns Redis key for tenant lookup by API key
// Format: alhudhud:tenant:apikey:{prefix}:{key}
func TenantAPIKeyCacheKey(prefix, key string) string {
	return fmt.Sprintf("%s:tenant:apikey:%s:%s", AppNamespace, prefix, key)
}

// StaffCacheKey returns Redis key for staff profile cache
// Format: alhudhud:staff:{tenant_id}:{staff_id}
func StaffCacheKey(tenantID, staffID uuid.UUID) string {
	return fmt.Sprintf("%s:staff:%s:%s", AppNamespace, tenantID.String(), staffID.String())
}

// DispatchChannelKey returns Redis key for device dispatch channel
// Format: alhudhud:dispatch:{device_id}
func DispatchChannelKey(deviceID uuid.UUID) string {
	return fmt.Sprintf("%s:dispatch:%s", AppNamespace, deviceID.String())
}

// PackageCacheKey returns Redis key for package cache
// Format: alhudhud:package:{package_id}
func PackageCacheKey(packageID int64) string {
	return fmt.Sprintf("%s:package:%d", AppNamespace, packageID)
}

// SubscriptionCacheKey returns Redis key for subscription cache
// Format: alhudhud:subscription:{subscription_id}
func SubscriptionCacheKey(subscriptionID int64) string {
	return fmt.Sprintf("%s:subscription:%d", AppNamespace, subscriptionID)
}

// GatewayCacheKey returns Redis key for gateway cache
// Format: alhudhud:gateway:{gateway_id}
func GatewayCacheKey(gatewayID uuid.UUID) string {
	return fmt.Sprintf("%s:gateway:%s", AppNamespace, gatewayID.String())
}

// TenantNamespacePattern returns pattern for all keys belonging to a tenant
// Format: alhudhud:*:{tenant_id}:*
func TenantNamespacePattern(tenantID uuid.UUID) string {
	return fmt.Sprintf("%s:*:%s:*", AppNamespace, tenantID.String())
}

// AllKeysPattern returns pattern for all application keys
// Format: alhudhud:*
func AllKeysPattern() string {
	return fmt.Sprintf("%s:*", AppNamespace)
}

// MessageCounterKey returns Redis key for message counter within an event
func MessageCounterKey(eventID string) string {
	return fmt.Sprintf("%s:event:%s:msg:counter", AppNamespace, eventID)
}

// formatTimestamp formats time to YYMMDDHHmm (10 digits)
func formatTimestamp(t time.Time) string {
	return fmt.Sprintf("%02d%02d%02d%02d%02d",
		t.Year()%100,
		t.Month(),
		t.Day(),
		t.Hour(),
		t.Minute())
}
