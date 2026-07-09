# Bugfix Requirements Document

## Introduction

**Bug Summary:** Messages that are saved in the database (via `/api/v1/messaging/ingest` or other endpoints) are not being delivered to the mobile application on Android/iOS. The mobile app uses WebSocket connection (`/ws/pair`) to receive messages, but messages stored in the database never reach the connected devices.

**Impact:** Users of the mobile application (alhomely5@gmail.com account) do not receive messages that should be forwarded from the accounting system through the web platform to their mobile devices.

## Bug Analysis

### Current Behavior (Defect)

1.1 WHEN a message is ingested via `/api/v1/messaging/ingest` and saved to the `incoming_messages` table THEN the system does NOT forward the message to connected mobile devices via WebSocket

1.2 WHEN a message is created in the `messages` table (via AnalysisWorker processing) THEN the system does NOT dispatch the message to the mobile application even when the device is connected via WebSocket

1.3 WHEN a mobile device is connected to `/ws/pair` WebSocket endpoint THEN the system fails to deliver pending messages that exist in the database to that device

### Expected Behavior (Correct)

2.1 WHEN a message is ingested via `/api/v1/messaging/ingest` THEN the system SHALL parse the message, create a corresponding entry in the `messages` table, and dispatch it to all connected mobile devices for that tenant via WebSocket

2.2 WHEN a message exists in the `messages` table with status `PENDING` or `QUEUED` AND a mobile device connects to WebSocket THEN the system SHALL deliver all pending messages to that device

2.3 WHEN a message is successfully processed by the AnalysisWorker THEN the system SHALL trigger the MessageActivationWorker to dispatch the message to mobile devices via `ws.GlobalHub.SendMessage()`

### Unchanged Behavior (Regression Prevention)

3.1 WHEN a message is sent via the standard SMS dispatch path (PHONES, PRIVATE channels) THEN the system SHALL CONTINUE to function as before without affecting mobile delivery

3.2 WHEN a WebSocket connection is established between mobile app and server THEN the heartbeat mechanism (PING/PONG) SHALL CONTINUE to work correctly

3.3 WHEN a device disconnects from WebSocket THEN the system SHALL CONTINUE to mark the connection status as disconnected in the hub

3.4 WHEN status reports are received from mobile devices THEN the system SHALL CONTINUE to process them and update message status in the database

---

## Bug Condition Derivation

### Bug Condition Function (C(X))

```pascal
FUNCTION isBugCondition(X)
  INPUT: X of type Message
  OUTPUT: boolean

  // Returns true when a message is saved but not delivered to mobile
  RETURN X.status IN ('PENDING', 'QUEUED', 'SAVED')
     AND X.source = 'ACCOUNTING_SYSTEM'
     AND X.id NOT IN (SELECT message_id FROM message_delivery_logs WHERE delivered = true)
END FUNCTION
```

### Property Specification (Fix Checking)

```pascal
// Property: Fix Checking - Message Delivery to Mobile
FOR ALL X WHERE isBugCondition(X) DO
  result ← dispatchMessageToMobile(X)
  ASSERT result.success = true AND result.error = null
END FOR
```

### Preservation Goal

```pascal
// Property: Preservation Checking
FOR ALL X WHERE NOT isBugCondition(X) DO
  originalResult ← F(X)      // Original behavior
  fixedResult ← F'(X)        // Fixed behavior
  ASSERT originalResult = fixedResult
END FOR
```

---

## Message Flow Analysis

### Current Flow (Defective)
```
System External → POST /api/v1/messaging/ingest
                    ↓
              IngestIncomingMessage()
                    ↓
              incoming_messages table (saved)
                    ↓
              TriggerAnalysis() → AnalysisWorker (async)
                    ↓
              messages table (created)
                    ↓
              [MISSING: No dispatch to mobile devices]
```

### Expected Flow (Fixed)
```
System External → POST /api/v1/messaging/ingest
                    ↓
              IngestIncomingMessage()
                    ↓
              incoming_messages table (saved)
                    ↓
              TriggerAnalysis() → AnalysisWorker
                    ↓
              messages table (created)
                    ↓
              MessageActivationWorker
                    ↓
              ws.GlobalHub.SendMessage() → WebSocket
                    ↓
              Mobile App (Android/iOS)
```

---

## Key Files Involved

| File | Role |
|------|------|
| `services/backend/internal/receiver/handler.go` | Ingest endpoint handler |
| `services/backend/internal/receiver/service.go` | Message staging and analysis triggering |
| `services/backend/internal/queue/jobs.go` | AnalysisWorker and MessageActivationWorker |
| `services/backend/internal/ws/hub.go` | WebSocket hub for message dispatch |
| `apps/mobile_gateway/src/services/WebSocketService.ts` | Mobile app WebSocket client |

---

## Reproduction Steps

1. Ensure mobile application is connected to WebSocket at `/ws/pair`
2. Send a test message via the accounting system simulation tool to `/api/v1/messaging/ingest`
3. Verify the message is saved in `incoming_messages` table
4. Wait for AnalysisWorker to process the message (creates entry in `messages` table)
5. Check if the message appears in the mobile application
6. **Expected Result:** Message should appear in mobile app
7. **Actual Result:** Message does NOT appear in mobile app