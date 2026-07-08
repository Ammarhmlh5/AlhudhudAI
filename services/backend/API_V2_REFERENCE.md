# AL-HUDHUD API V2 REFERENCE (Hardened Edition)

This document provides a comprehensive guide to the yeni messaging and administration API, featuring the high-performance 12-stage pipeline and enhanced security layers.

## 🚀 Unified Messaging Ingestion

### [POST] /api/v1/messaging/bulk-send
Ingests a bulk set of messages into the raw event store for durable processing.

**Headers:**
- `X-API-KEY`: Your tenant API Key (Cached, Rate-limited: 100 req/10s)

**Request Body:**
```json
{
  "message_type": "INVOICE",
  "sender_name": "MyBusiness",
  "messages": [
    {
      "phone_number": "+1234567890",
      "content": "Message 1",
      "scheduled_at": "2026-03-22T12:00:00Z"
    }
  ]
}
```

**Response (O(1) - Fast Path):**
```json
{
  "event_id": 12345,
  "status": "RECEIVED",
  "total_items": 1
}
```

---

## 📊 Monitoring & Observability

### [GET] /api/v1/messaging/progress/:event_id
Returns real-time processing statistics for a bulk ingestion event.

---

## 🛡️ Administrative Audit

### [GET] /api/v1/admin/health
Returns system health status (Database, Redis, Worker Pool).

### [GET] /api/v1/admin/events
Lists all raw ingestion events for auditing.
- Query Params: `status`, `tenant_id`

### [POST] /api/v1/admin/events/:id/retry
Manually re-queues a failed event for parsing.

### [GET] /api/v1/admin/alerts
Lists system-wide security and performance alerts (e.g., Massive Bulk detection).

---

## 🔒 Security Policy
- **Rate Limiting**: Enforced per-tenant via Redis.
- **RLS**: All data is strictly isolated via PostgreSQL Row Level Security.
- **Durable Delivery**: All events are stored raw first, then parsed via River Queue.
