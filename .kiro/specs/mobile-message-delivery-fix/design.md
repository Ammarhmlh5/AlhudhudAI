# Mobile Message Delivery Bugfix Design

## Overview

This design document addresses the bug where messages saved in the database are not being delivered to mobile devices via WebSocket. The focus is on testing the following mobile app operations:

1. **استلام الرسائل** (Message Reception): Receiving messages from the website via WebSocket
2. **تخزين الرسائل** (Message Storage): Storing received messages locally on the device
3. **ارسال الرسائل** (Message Sending): Sending messages from the phone via SIM card
4. **تقارير التسليم** (Delivery Reports): Sending delivery status reports back to the server

The root cause is that the `MessageActivationWorker` exists but does not dispatch messages to connected mobile devices through `ws.GlobalHub.SendMessage()`. The fix will connect the message processing pipeline to the WebSocket dispatch mechanism, ensuring that when messages are created or updated in the `messages` table, they are immediately delivered to all connected mobile devices for the corresponding tenant.

## Glossary

- **Bug_Condition (C)**: The condition that triggers the bug - when a message is saved in the database but not delivered to mobile devices via WebSocket
- **Property (P)**: The desired behavior when messages are processed - they should be dispatched to all connected mobile devices for the tenant
- **Preservation**: Existing SMS dispatch, heartbeat, and status report processing that must remain unchanged by the fix
- **MessageActivationWorker**: The background worker in `services/backend/internal/queue/jobs.go` responsible for activating/mobile-dispatching messages
- **ws.GlobalHub**: The WebSocket hub singleton in `services/backend/internal/ws/hub.go` that manages connections and message dispatch
- **Tenant**: The account/organization context (e.g., alhomely5@gmail.com) that determines which devices receive messages
- **Pending Messages**: Messages in the `messages` table with status `PENDING` or `QUEUED` that have not been delivered to mobile devices

## Bug Details

### Bug Condition

The bug manifests when a message is ingested via the accounting system integration and saved to the database, but the `MessageActivationWorker` fails to dispatch it to connected mobile devices. The worker either does not query for messages to dispatch, does not find the correct WebSocket hub method, or does not properly identify connected devices for the tenant.

**Formal Specification:**
```
FUNCTION isBugCondition(input)
  INPUT: input of type Message
  OUTPUT: boolean

  // Returns true when a message is saved but not delivered to mobile
  RETURN input.status IN ('PENDING', 'QUEUED', 'SAVED')
         AND input.source = 'ACCOUNTING_SYSTEM'
         AND input.created_at > NOW() - 24 HOURS
         AND input.id NOT IN (
           SELECT message_id FROM message_delivery_logs
           WHERE delivered = true AND delivery_channel = 'MOBILE'
         )
END FUNCTION
```

### Examples

- **Example 1 - New Message Ingestion**: A message arrives from the accounting system at 10:00 AM, gets saved to `incoming_messages`, processed by AnalysisWorker, and creates a PENDING entry in `messages` table at 10:01 AM. Expected: Message dispatched to mobile devices at 10:01 AM. Actual: Message remains undispatched.

- **Example 2 - Device Reconnection**: A mobile device was offline from 9:00 AM to 11:00 AM. During that time, 5 messages were ingested and saved. At 11:00 AM the device reconnects. Expected: All 5 pending messages are delivered. Actual: No messages are delivered on reconnect.

- **Example 3 - Multiple Tenants**: Two tenants (alhomely5@gmail.com and another-account@gmail.com) have mobile devices connected. A message for tenant A arrives. Expected: Only tenant A's devices receive the message. Actual: No devices receive the message.

- **Edge Case - No Connected Devices**: A message is ingested for a tenant that has no mobile devices connected. Expected: Message is saved and marked for later delivery. Actual: Message is saved but no delivery attempt is recorded.

## Expected Behavior

### Preservation Requirements

**Unchanged Behaviors:**
- SMS dispatch via PHONES and PRIVATE channels continues to function exactly as before
- WebSocket heartbeat mechanism (PING/PONG) continues to work correctly for connection keep-alive
- Device disconnection handling continues to mark connections as disconnected in the hub
- Status report processing from mobile devices continues to update message status in the database
- Message analysis and classification by AnalysisWorker continues to work unchanged

**Scope:**
All inputs that do NOT involve mobile message dispatch should be completely unaffected by this fix. This includes:
- SMS gateway dispatch operations
- WebSocket connection establishment and heartbeat
- Device status updates and disconnection handling
- Message status report processing
- Message analysis and classification workflows

## Hypothesized Root Cause

Based on the bug description and message flow analysis, the most likely issues are:

1. **MessageActivationWorker Not Invoked**: The AnalysisWorker creates messages in the `messages` table but does not trigger the MessageActivationWorker to dispatch them. The worker may exist but is never called after message creation.

2. **WebSocket Dispatch Method Not Called**: Even if MessageActivationWorker runs, it may not be calling `ws.GlobalHub.SendMessage()` correctly. The worker might be querying for messages but not actually invoking the dispatch method, or the method signature may have changed.

3. **Tenant/Device Matching Missing**: The dispatch logic may not correctly identify which mobile devices should receive a message based on tenant context. Messages might be dispatched to all devices regardless of tenant, or to no devices at all.

4. **Pending Message Queue Not Implemented**: When a device connects, the system may not query for and deliver messages that were queued while the device was offline. The reconnection handler may only handle new messages.

5. **Missing Delivery Logging**: Even if messages are dispatched, the `message_delivery_logs` table may not be updated, making it appear as though messages were never delivered.

## Correctness Properties

Property 1: Bug Condition - Mobile Message Delivery

_For any_ message where the bug condition holds (isBugCondition returns true), the fixed MessageActivationWorker SHALL dispatch the message to all connected mobile devices for the corresponding tenant via `ws.GlobalHub.SendMessage()` and record successful delivery in `message_delivery_logs`.

**Validates: Requirements 2.1, 2.2, 2.3**

Property 2: Preservation - Non-Mobile Delivery Behavior

_For any_ message or operation that does NOT involve mobile WebSocket dispatch (SMS dispatch, heartbeat, status reports, message analysis), the fixed code SHALL produce exactly the same behavior as the original code, preserving all existing functionality for non-mobile delivery paths.

**Validates: Requirements 3.1, 3.2, 3.3, 3.4**

## Fix Implementation

### Changes Required

**File**: `services/backend/internal/queue/jobs.go`

**Function**: `MessageActivationWorker` and `TriggerMessageActivation`

**Specific Changes:**

1. **Invoke MessageActivationWorker After Analysis**: Modify the AnalysisWorker completion handler to call `TriggerMessageActivation(messageID)` after a message is successfully created in the `messages` table.

2. **Implement MessageActivationWorker Logic**: Complete the MessageActivationWorker implementation to:
   - Query the `messages` table for PENDING/QUEUED messages that have not been delivered to mobile
   - For each message, identify all connected mobile devices for the tenant
   - Call `ws.GlobalHub.SendMessage()` for each device with the message payload
   - Record delivery in `message_delivery_logs` table

3. **Add Tenant-Aware Device Lookup**: Create a helper function to get all connected device IDs for a specific tenant from the WebSocket hub.

4. **Implement Reconnection Delivery**: Modify the WebSocket hub connection handler to query and deliver pending messages when a new device connects.

5. **Add Delivery Logging**: Ensure all dispatch attempts (successful and failed) are recorded in `message_delivery_logs`.

**File**: `services/backend/internal/ws/hub.go`

**Function**: `GlobalHub.SendMessage()` and connection management

**Specific Changes:**

1. **Expose Connected Devices by Tenant**: Add a method to query all connections for a specific tenant, returning device IDs and connection references.

2. **Ensure SendMessage Handles Multiple Recipients**: Verify `SendMessage()` can broadcast to multiple devices or be called per-device efficiently.

3. **Add Connection Hook for Pending Messages**: Add a callback or hook that gets triggered on new connections to deliver pending messages.

## Testing Strategy

### Mobile App Operations Testing

The testing strategy focuses on four core mobile app operations:

#### 1. استلام الرسائل (Message Reception)
- Verify WebSocket connection receives messages from website
- Test message parsing and deserialization
- Verify message appears in mobile app UI
- Test with multiple concurrent messages

#### 2. تخزين الرسائل (Message Storage)
- Verify received messages are stored locally
- Test message persistence across app restarts
- Verify message ordering and timestamps
- Test storage limits and cleanup

#### 3. ارسال الرسائل (Message Sending via SIM)
- Test sending messages through SIM card
- Verify message status updates (SENT, DELIVERED, FAILED)
- Test message queue management
- Verify delivery confirmation handling

#### 4. تقارير التسليم (Delivery Reports)
- Test sending delivery status reports to server
- Verify report includes message ID, status, timestamp
- Test report acknowledgment handling
- Verify offline report queuing

### Validation Approach

The testing strategy follows a two-phase approach: first, surface counterexamples that demonstrate the bug on unfixed code, then verify the fix works correctly and preserves existing behavior.

### Exploratory Bug Condition Checking

**Goal**: Surface counterexamples that demonstrate the bug BEFORE implementing the fix. Confirm or refute the root cause analysis. If we refute, we will need to re-hypothesize.

**Test Plan**: Write integration tests that simulate the full message flow from ingestion to mobile delivery. Run these tests on the UNFIXED code to observe failures and understand the root cause.

**Test Cases:**
1. **Message Ingestion to Mobile Delivery Test**: Simulate message ingestion via `/api/v1/messaging/ingest` with a connected mobile device. Assert the message appears on the device. (Will fail on unfixed code)
2. **Multiple Messages Single Device Test**: Send 5 messages while device is connected, verify all 5 are received. (Will fail on unfixed code)
3. **Device Reconnection Test**: Disconnect device, send messages, reconnect device, verify pending messages are delivered. (Will fail on unfixed code)
4. **Multi-Tenant Isolation Test**: Connect devices for two tenants, send message for tenant A, verify only tenant A's device receives it. (Will fail on unfixed code)

**Expected Counterexamples:**
- Message is saved to database but never dispatched to WebSocket
- WebSocket hub receives no message when AnalysisWorker completes
- Connected devices are not identified correctly by tenant
- Reconnecting devices do not receive queued messages

### Fix Checking

**Goal**: Verify that for all inputs where the bug condition holds, the fixed function produces the expected behavior.

**Pseudocode:**
```
FOR ALL message WHERE isBugCondition(message) DO
  result := MessageActivationWorker(message)
  ASSERT result.success = true
  ASSERT ws.GlobalHub.SendMessage was called with message
  ASSERT message_delivery_logs shows delivery for mobile channel
END FOR
```

### Preservation Checking

**Goal**: Verify that for all inputs where the bug condition does NOT hold, the fixed function produces the same result as the original function.

**Pseudocode:**
```
FOR ALL operation WHERE operation DOES NOT involve mobile dispatch DO
  ASSERT originalBehavior(operation) = fixedBehavior(operation)
END FOR
```

**Testing Approach**: Property-based testing is recommended for preservation checking because:
- It generates many test cases automatically across the input domain
- It catches edge cases that manual unit tests might miss
- It provides strong guarantees that behavior is unchanged for non-mobile delivery paths

**Test Plan**: Observe behavior on UNFIXED code first for SMS dispatch, heartbeat, and status reports, then write property-based tests capturing that behavior.

**Test Cases:**
1. **SMS Dispatch Preservation**: Observe that SMS messages continue to be dispatched via PHONES/PRIVATE channels, then write test to verify this continues after fix
2. **Heartbeat Preservation**: Observe that WebSocket PING/PONG continues to work, then write test to verify this continues after fix
3. **Status Report Preservation**: Observe that mobile status reports update message status, then write test to verify this continues after fix
4. **Message Analysis Preservation**: Observe that AnalysisWorker continues to classify messages correctly, then write test to verify this continues after fix

### Unit Tests

- Test `isBugCondition()` function with various message states
- Test tenant-aware device lookup from WebSocket hub
- Test message dispatch loop handles empty device lists gracefully
- Test delivery logging records success and failure correctly
- Test reconnection handler queries pending messages correctly

### Property-Based Tests

- Generate random messages with various statuses and sources, verify only accounting system messages trigger mobile dispatch
- Generate random tenant/device configurations, verify messages go only to correct tenant's devices
- Generate random WebSocket connection states, verify preservation of heartbeat and disconnect behavior
- Generate random message flows, verify SMS dispatch path is unaffected by mobile changes

### Integration Tests

- Full end-to-end test from `/api/v1/messaging/ingest` to mobile device message receipt
- Multi-tenant message routing test
- Device disconnection and reconnection with pending messages test
- Concurrent message ingestion with multiple connected devices test
- Failure handling test (WebSocket disconnect during message dispatch)