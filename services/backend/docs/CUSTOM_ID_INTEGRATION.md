# Custom ID System Integration Guide

## Overview
This document describes the complete integration of the Redis-based Custom ID generation system into the Al-Hudhud backend.

## What Was Implemented

### 1. Core ID Generation System
- **File**: `internal/redis/id_generator.go`
- **Features**:
  - Event ID generation: `EV{timestamp}{tenant_hash}{sequence}` (19 chars)
  - Message ID generation: `MS{timestamp}{tenant_hash}{event_seq}{msg_seq}` (22 chars)
  - Automatic sequence management with Redis counters
  - 999 message limit per event (auto-creates new event if exceeded)
  - 9999 event limit per tenant per minute

### 2. Database Schema Updates
- **File**: `migrations/077_add_custom_ids.sql`
- **Changes**:
  - Added `custom_id` column to `global_incoming_events` table (VARCHAR(20), UNIQUE)
  - Added `custom_id` column to `messages` table (VARCHAR(23), UNIQUE)
  - Added `event_custom_id` column to `messages` table (VARCHAR(20), INDEXED)
  - Created indexes for fast lookups
  - Created helper functions for parsing IDs
  - Created views for analytics

### 3. Model Updates
- **Files**: 
  - `internal/models/global_event.go`
  - `internal/models/models.go`
- **Changes**:
  - Added `CustomID` field to `GlobalIncomingEvent` struct
  - Added `CustomID` and `EventCustomID` fields to `Message` struct
  - Internal `ID` fields remain for admin statistics only

### 4. Service Layer Integration
- **File**: `internal/receiver/service.go`
- **Changes**:
  - Added `redisClient` field to Service struct
  - Updated `NewService()` to accept redis client parameter
  - Updated `IngestGlobalEvent()` to generate custom event IDs using Redis
  - Updated `GetBulkEvent()` to query by custom_id instead of internal id

### 5. Handler Layer Updates
- **File**: `internal/receiver/handler.go`
- **Changes**:
  - Updated `NewHandler()` to accept redis client parameter
  - Updated API responses to return `custom_id` for tracking
  - Added `_internal` or `_internal_id` fields for admin-only internal IDs
  - Updated `BulkProgressHandler` to accept custom_id in URL parameter

### 6. Worker Integration
- **File**: `internal/queue/jobs.go`
- **Changes**:
  - Updated `GlobalEventWorker` to generate message custom IDs
  - Each message gets a unique ID linked to its parent event
  - Uses `redis.GetGlobalClient()` to access ID generator

### 7. Main Application Updates
- **File**: `main.go`
- **Changes**:
  - Updated `receiver.NewHandler()` call to pass `redisClient`
  - Redis client is now available globally for ID generation

### 8. Redis Client Enhancement
- **File**: `internal/redis/redis.go`
- **Changes**:
  - Added global `globalClient` variable
  - Added `GetGlobalClient()` function for worker access
  - Automatically sets global client when `NewRedisClient()` is called

## API Response Changes

### Before
```json
{
  "event_id": 12345,
  "status": "RECEIVED"
}
```

### After
```json
{
  "event_id": "EV2601012312345000100",
  "status": "RECEIVED",
  "_internal": 12345
}
```

## ID Format Specifications

### Event ID Format (19 characters)
```
EV 26010123 12345 0001
│  │        │     │
│  │        │     └─ Event Sequence (0001-9999)
│  │        └─────── Tenant Hash (5 digits)
│  └──────────────── Timestamp (YYMMDDHHmmss)
└─────────────────── Entity Type (EV = Event)
```

### Message ID Format (22 characters)
```
MS 26010123 12345 0001 001
│  │        │     │    │
│  │        │     │    └─ Message Sequence (001-999)
│  │        │     └────── Event Sequence (0001-9999)
│  │        └──────────── Tenant Hash (5 digits)
│  └───────────────────── Timestamp (YYMMDDHHmmss)
└──────────────────────── Entity Type (MS = Message)
```

## Redis Keys Used

### ID Generation
- `alhudhud:id:event:{tenant_id}:{minute}` - Event sequence counter
- `alhudhud:id:message:{event_custom_id}` - Message sequence counter per event

### Analytics
- `alhudhud:analytics:load:{minute}` - Load tracking per minute
- `alhudhud:analytics:tenant:{tenant_id}:{minute}` - Per-tenant load tracking

## Migration Steps

### 1. Run Database Migration
```bash
psql -U your_user -d your_database -f migrations/077_add_custom_ids.sql
```

### 2. Verify Indexes
```sql
-- Check that indexes were created
\d global_incoming_events
\d messages
```

### 3. Test ID Generation
```bash
# Start Redis
redis-server

# Start backend
go run main.go

# Test bulk ingestion
curl -X POST http://localhost:8080/api/v1/messages/bulk \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "message_type": "INVOICE",
    "sender_name": "Test System",
    "messages": [
      {"phone_number": "+966501234567", "content": "Test message"}
    ]
  }'
```

### 4. Verify Response
```json
{
  "message": "Bulk messages received successfully",
  "event_id": "EV2601012312345000100",
  "status": "RECEIVED",
  "_internal": 1
}
```

## Monitoring and Analytics

### Check System Load
```go
analytics := redis.NewAnalytics(redisClient)
load, err := analytics.GetCurrentLoad(ctx)
// Returns: events/min, messages/min, tenants active
```

### Check Tenant Load
```go
tenantLoad, err := analytics.GetTenantLoad(ctx, tenantID)
// Returns: events and messages for specific tenant
```

### Parse Custom IDs
```sql
-- Parse event ID
SELECT parse_event_id('EV2601012312345000100');

-- Parse message ID
SELECT parse_message_id('MS26010123123450001001');
```

## Error Handling

### Event Limit Exceeded (9999 events/minute)
- **Error**: "event sequence limit exceeded for this minute"
- **Solution**: Wait for next minute or implement rate limiting

### Message Limit Exceeded (999 messages/event)
- **Behavior**: Automatically creates new event and continues
- **Tracking**: New event ID is generated, messages continue seamlessly

### Redis Unavailable
- **Error**: "redis client not available for ID generation"
- **Solution**: Ensure Redis is running and connected

## Performance Considerations

### Redis Operations
- Event ID generation: 2 Redis operations (INCR + EXPIRE)
- Message ID generation: 2 Redis operations (INCR + EXPIRE)
- All operations use pipelining for efficiency

### Database Queries
- Custom ID lookups use UNIQUE indexes (O(log n))
- Internal ID remains primary key for admin queries
- Event-Message relationship uses indexed `event_custom_id`

## Testing Checklist

- [ ] Single message ingestion generates custom IDs
- [ ] Bulk message ingestion generates sequential message IDs
- [ ] Event progress tracking works with custom IDs
- [ ] API responses return custom IDs
- [ ] Database queries by custom_id work correctly
- [ ] 999 message limit triggers new event creation
- [ ] Analytics tracking works correctly
- [ ] Parsing functions extract correct components

## Rollback Plan

If issues occur, you can rollback by:

1. **Revert code changes** (git revert)
2. **Drop new columns** (if needed):
```sql
ALTER TABLE global_incoming_events DROP COLUMN IF EXISTS custom_id;
ALTER TABLE messages DROP COLUMN IF EXISTS custom_id;
ALTER TABLE messages DROP COLUMN IF EXISTS event_custom_id;
```

3. **Update API clients** to use internal IDs again

## Future Enhancements

1. **Custom ID Validation Middleware**: Add API middleware to validate custom_id format
2. **ID Reservation System**: Pre-generate ID batches for ultra-high throughput
3. **Multi-Region Support**: Add region prefix to IDs for global deployment
4. **Compression**: Use base62 encoding to reduce ID length
5. **Audit Trail**: Track ID generation history for compliance

## Support

For issues or questions:
- Check Redis connection: `redis-cli ping`
- Check database migration: `\d messages`
- Review logs: Look for "failed to generate" errors
- Test ID generation: Use `redis.NewIDGenerator(client).GenerateEventID()`

---

**Status**: ✅ Fully Integrated
**Version**: 1.0
**Last Updated**: 2026-01-23
