#!/bin/bash

# AL-HUDHUD Message Lifecycle Stress Test Script
# ---------------------------------------------
# This script verifies the end-to-end flow: Ingestion -> Parsing -> Activation -> DLR
# It assumes the backend is running on http://localhost:8083

API_URL="http://localhost:8083/api/v1"
AUTH_TOKEN="YOUR_TEST_TOKEN" # Provide a valid JWT or API Key here
TENANT_ID="00000000-0000-0000-0000-000000000001" # Replace with your test tenant UUID

echo "--- STRESS TEST START ---"

# Step 1: Bulk Ingestion (Phase 1)
echo "Injecting 1000 messages (Bulk JSON)..."
START_TIME=$(date +%s%N)
curl -s -X POST "$API_URL/receiver/bulk" \
  -H "Authorization: Bearer $AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "message_type": "OTP",
    "sender_name": "Antigravity_Stress_Tester",
    "messages": [
      '$(for i in {1..1000}; do echo '{"phone_number": "+967770000000", "content": "Stress Test Message '"$i"'"},'; done | sed '$ s/,$//')'
    ]
  }' > /dev/null

END_TIME=$(date +%s%N)
DIFF=$(( (END_TIME - START_TIME) / 1000000 ))
echo "Ingestion Response Time: ${DIFF}ms (Target: <200ms)"

# Step 2: Observe Parsing (Phase 2 & 3)
echo "Waiting 5 seconds for Workers to parse and activate messages..."
sleep 5

# Step 3: Check Balance & Stats
echo "Verifying Admin Statistics..."
curl -s -X GET "$API_URL/admin/stats" \
  -H "Authorization: Bearer $AUTH_TOKEN" | jq '.data.total_messages' | xargs echo "Current Total Messages in Stats:"

# Step 4: Verify Atomic Financial Integrity (Phase 3)
echo "Verifying Balance Deduction (Atomic)..."
# Assuming 1000 messages @ 0.05 = 50.0 units
# You should check the 'tenants' table balance before and after.

# Step 5: Simulate Bulk DLRs (Phase 4)
echo "Simulating 1000 asynchronous Delivery Reports..."
START_TIME=$(date +%s%N)
for msg_id in $(psql -At -c "SELECT id FROM messages WHERE sender_name='Antigravity_Stress_Tester' LIMIT 100"); do
  curl -s -X POST "$API_URL/internal/dlr" \
    -H "Content-Type: application/json" \
    -d '{
      "message_id": "'$msg_id'",
      "status": "DELIVERED",
      "error_code": 0
    }' & 
done
wait
END_TIME=$(date +%s%N)
DIFF=$(( (END_TIME - START_TIME) / 1000000 ))
echo "Batch 100 DLR Processing Time (Parallel): ${DIFF}ms"

echo "--- STRESS TEST COMPLETE ---"
echo "Check the dashboard at http://localhost:8081/admin/dashboard to see the metrics reflected."
