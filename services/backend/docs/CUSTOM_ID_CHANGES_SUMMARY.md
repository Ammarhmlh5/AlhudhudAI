# Custom ID System - Changes Summary

## تلخيص التغييرات (Summary in Arabic)

تم تنفيذ نظام توليد المعرفات المخصصة بالكامل باستخدام Redis. النظام يولد معرفات فريدة للأحداث والرسائل بناءً على الوقت ومعرف الشركة.

### صيغة المعرفات

**معرف الحدث (Event ID)** - 19 حرف:
```
EV2601012312345000100
│ │       │    │
│ │       │    └─ رقم تسلسل الحدث (0001-9999)
│ │       └────── هاش معرف الشركة (5 أرقام)
│ └────────────── الطابع الزمني (سنة شهر يوم ساعة دقيقة ثانية)
└──────────────── نوع الكيان (EV = حدث)
```

**معرف الرسالة (Message ID)** - 22 حرف:
```
MS26010123123450001001
│ │       │    │   │
│ │       │    │   └─ رقم تسلسل الرسالة (001-999)
│ │       │    └───── رقم تسلسل الحدث (0001-9999)
│ │       └────────── هاش معرف الشركة (5 أرقام)
│ └────────────────── الطابع الزمني
└──────────────────── نوع الكيان (MS = رسالة)
```

### الحد الأقصى
- **999 رسالة لكل حدث**: عند الوصول للحد، يتم إنشاء حدث جديد تلقائياً
- **9999 حدث لكل دقيقة**: لكل شركة

## Files Modified

### 1. Core Implementation Files (New)
- ✅ `internal/redis/id_generator.go` - ID generation logic
- ✅ `internal/redis/analytics.go` - Load monitoring
- ✅ `internal/redis/keys.go` - Updated with new key patterns
- ✅ `migrations/077_add_custom_ids.sql` - Database schema changes

### 2. Model Files (Modified)
- ✅ `internal/models/global_event.go`
  - Added `CustomID string` field
  - Updated JSON tags

- ✅ `internal/models/models.go`
  - Added `CustomID string` field to Message
  - Added `EventCustomID string` field to Message
  - Updated JSON tags and GORM tags

### 3. Service Layer (Modified)
- ✅ `internal/receiver/service.go`
  - Added `redisClient *redis.Client` field to Service struct
  - Updated `NewService(redisClient *redis.Client)` signature
  - Added redis import
  - Updated `IngestGlobalEvent()` to generate custom event IDs
  - Updated `GetBulkEvent()` to query by custom_id

### 4. Handler Layer (Modified)
- ✅ `internal/receiver/handler.go`
  - Updated `NewHandler(redisClient *redis.Client)` signature
  - Added redis import
  - Updated `BulkSendMessageHandler` response to return custom_id
  - Updated `BulkProgressHandler` to accept custom_id parameter
  - Updated `BulkIngestExcelHandler` response to return custom_id

### 5. Worker Layer (Modified)
- ✅ `internal/queue/jobs.go`
  - Updated `GlobalEventWorker.Work()` to generate message custom IDs
  - Added redis.GetGlobalClient() call
  - Added CustomID and EventCustomID to message creation

### 6. Redis Client (Modified)
- ✅ `internal/redis/redis.go`
  - Added `globalClient *Client` variable
  - Updated `NewRedisClient()` to set global client
  - Added `GetGlobalClient()` function

### 7. Main Application (Modified)
- ✅ `main.go`
  - Updated `receiver.NewHandler(redisClient)` call

### 8. Documentation (New)
- ✅ `docs/CUSTOM_ID_SYSTEM.md` - Complete system documentation
- ✅ `docs/CUSTOM_ID_INTEGRATION.md` - Integration guide
- ✅ `docs/CUSTOM_ID_CHANGES_SUMMARY.md` - This file

## Code Changes Summary

### Before
```go
// Service initialization
receiverHandler := receiver.NewHandler()

// Event creation
event := &models.GlobalIncomingEvent{
    TenantID:   &tenantID,
    RawPayload: rawJSON,
    Status:     "RECEIVED",
}

// API Response
{
  "event_id": 12345,
  "status": "RECEIVED"
}
```

### After
```go
// Service initialization
receiverHandler := receiver.NewHandler(redisClient)

// Event creation with custom ID
idGen := redis.NewIDGenerator(s.redisClient)
customID, err := idGen.GenerateEventID(ctx, tenantID, senderIP)

event := &models.GlobalIncomingEvent{
    CustomID:   customID,
    TenantID:   &tenantID,
    RawPayload: rawJSON,
    Status:     "RECEIVED",
}

// API Response
{
  "event_id": "EV2601012312345000100",
  "status": "RECEIVED",
  "_internal": 12345
}
```

## Database Schema Changes

### global_incoming_events Table
```sql
-- New columns
ALTER TABLE global_incoming_events 
ADD COLUMN custom_id VARCHAR(20) UNIQUE;

CREATE UNIQUE INDEX idx_global_events_custom_id 
ON global_incoming_events(custom_id);
```

### messages Table
```sql
-- New columns
ALTER TABLE messages 
ADD COLUMN custom_id VARCHAR(23) UNIQUE,
ADD COLUMN event_custom_id VARCHAR(20);

CREATE UNIQUE INDEX idx_messages_custom_id 
ON messages(custom_id);

CREATE INDEX idx_messages_event_custom_id 
ON messages(event_custom_id);
```

## API Changes

### Endpoint: POST /api/v1/messages/bulk

**Response Before:**
```json
{
  "message": "Bulk messages received successfully",
  "event_id": 12345,
  "status": "RECEIVED"
}
```

**Response After:**
```json
{
  "message": "Bulk messages received successfully",
  "event_id": "EV2601012312345000100",
  "status": "RECEIVED",
  "_internal": 12345
}
```

### Endpoint: GET /api/v1/messages/bulk/:id

**URL Before:**
```
GET /api/v1/messages/bulk/12345
```

**URL After:**
```
GET /api/v1/messages/bulk/EV2601012312345000100
```

**Response After:**
```json
{
  "event_id": "EV2601012312345000100",
  "status": "PROCESSING",
  "total_items": 1000,
  "processed_items": 500,
  "failed_items": 0,
  "progress": "50.00%",
  "_internal_id": 12345
}
```

## Redis Keys Added

### ID Generation Keys
```
alhudhud:id:event:{tenant_id}:{minute}
alhudhud:id:message:{event_custom_id}
```

### Analytics Keys
```
alhudhud:analytics:load:{minute}
alhudhud:analytics:tenant:{tenant_id}:{minute}
```

## Testing Requirements

### Unit Tests Needed
- [ ] ID generation format validation
- [ ] Sequence counter increment
- [ ] 999 message limit handling
- [ ] Tenant hash generation
- [ ] Timestamp formatting

### Integration Tests Needed
- [ ] End-to-end bulk ingestion with custom IDs
- [ ] Event progress tracking with custom IDs
- [ ] Message creation with event linkage
- [ ] API response format validation
- [ ] Database query performance

### Load Tests Needed
- [ ] High-volume ID generation (10k+ events/min)
- [ ] Redis performance under load
- [ ] Database query performance with custom IDs
- [ ] Concurrent ID generation

## Deployment Checklist

### Pre-Deployment
- [ ] Backup database
- [ ] Test migration on staging
- [ ] Verify Redis is running
- [ ] Review rollback plan

### Deployment Steps
1. [ ] Stop application
2. [ ] Run database migration `077_add_custom_ids.sql`
3. [ ] Verify indexes created
4. [ ] Deploy new code
5. [ ] Start application
6. [ ] Test ID generation
7. [ ] Monitor logs for errors

### Post-Deployment
- [ ] Verify API responses contain custom IDs
- [ ] Check Redis key creation
- [ ] Monitor database performance
- [ ] Test bulk ingestion
- [ ] Verify analytics tracking

## Performance Impact

### Expected Improvements
- ✅ Faster event/message tracking (human-readable IDs)
- ✅ Better load monitoring (Redis analytics)
- ✅ Easier debugging (IDs contain timestamp and tenant info)

### Potential Concerns
- ⚠️ Additional Redis operations (2 per ID generation)
- ⚠️ Slightly larger database storage (custom_id columns)
- ⚠️ API response size increase (custom IDs are longer)

### Mitigation
- Redis operations are pipelined for efficiency
- Custom IDs are indexed for fast lookups
- Internal IDs remain for admin queries

## Rollback Instructions

If issues occur:

1. **Stop application**
2. **Revert code changes:**
   ```bash
   git revert HEAD
   ```

3. **Drop new columns (optional):**
   ```sql
   ALTER TABLE global_incoming_events DROP COLUMN IF EXISTS custom_id;
   ALTER TABLE messages DROP COLUMN IF EXISTS custom_id;
   ALTER TABLE messages DROP COLUMN IF EXISTS event_custom_id;
   ```

4. **Restart application**

## Next Steps

1. ✅ Complete integration (DONE)
2. ⏳ Run database migration
3. ⏳ Deploy to staging
4. ⏳ Run integration tests
5. ⏳ Deploy to production
6. ⏳ Monitor performance
7. ⏳ Update API documentation
8. ⏳ Update client applications

## Questions & Answers

**Q: Why use Redis for ID generation instead of database sequences?**
A: Redis provides faster atomic operations and allows us to embed metadata (timestamp, tenant) in the ID itself.

**Q: What happens if Redis is unavailable?**
A: The system will return an error. Redis availability is critical for ID generation. Consider Redis Sentinel or Cluster for high availability.

**Q: Can we change the ID format later?**
A: Yes, but it requires migration. The format is designed to be extensible (can add region prefix, etc.).

**Q: How do we handle ID collisions?**
A: Redis INCR is atomic, preventing collisions. The combination of timestamp + tenant + sequence ensures uniqueness.

**Q: What about existing messages without custom IDs?**
A: They will have NULL custom_id. New messages will have custom IDs. You can backfill if needed.

---

**Status**: ✅ Integration Complete
**Ready for**: Database Migration & Testing
**Next Action**: Run migration `077_add_custom_ids.sql`
