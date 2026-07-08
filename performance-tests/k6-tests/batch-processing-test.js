import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const batchProcessingTime = new Trend('batch_processing_time');
const messagesProcessed = new Counter('messages_processed');
const batchErrors = new Rate('batch_errors');

// Test configuration
export const options = {
  scenarios: {
    batch_test: {
      executor: 'constant-vus',
      vus: 10,
      duration: '2m',
    },
  },
  thresholds: {
    'batch_processing_time': ['p(95)<200'],  // 95% of batches processed in < 200ms
    'batch_errors': ['rate<0.01'],           // Error rate < 1%
  },
};

const BASE_URL = __ENV.API_URL || 'http://localhost:8080';
const TENANT_IDS = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10];

function randomPhone() {
  const prefix = '+96650';
  const number = Math.floor(Math.random() * 10000000).toString().padStart(7, '0');
  return prefix + number;
}

function randomMessage() {
  const messages = [
    'Batch test message',
    'Performance testing batch processing',
    'Load test for batch processor',
  ];
  return messages[Math.floor(Math.random() * messages.length)];
}

// Send a batch of messages
export default function () {
  const tenantId = TENANT_IDS[Math.floor(Math.random() * TENANT_IDS.length)];
  const batchSize = 100; // Test with 100 messages per batch
  
  // Create batch payload
  const messages = [];
  for (let i = 0; i < batchSize; i++) {
    messages.push({
      tenant_id: tenantId,
      phone_number: randomPhone(),
      content: randomMessage(),
      priority: 'NORMAL',
    });
  }

  const payload = JSON.stringify({ messages });
  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
  };

  // Send batch request
  const startTime = Date.now();
  const response = http.post(`${BASE_URL}/api/messages/bulk`, payload, params);
  const duration = Date.now() - startTime;

  // Record metrics
  batchProcessingTime.add(duration);
  batchErrors.add(response.status !== 200);
  
  if (response.status === 200) {
    messagesProcessed.add(batchSize);
  }

  // Validate response
  check(response, {
    'status is 200': (r) => r.status === 200,
    'batch processing time < 200ms': () => duration < 200,
    'has batch_id': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.batch_id !== undefined;
      } catch (e) {
        return false;
      }
    },
    'status is PROCESSING': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.status === 'PROCESSING';
      } catch (e) {
        return false;
      }
    },
  });

  // Wait before next batch
  sleep(1);
}

export function handleSummary(data) {
  const summary = {
    test_name: 'Batch Processing Performance Test',
    duration_seconds: data.state.testRunDurationMs / 1000,
    total_batches: data.metrics.http_reqs?.values.count || 0,
    total_messages: data.metrics.messages_processed?.values.count || 0,
    avg_batch_time_ms: data.metrics.batch_processing_time?.values.avg || 0,
    p95_batch_time_ms: data.metrics.batch_processing_time?.values['p(95)'] || 0,
    error_rate: (data.metrics.batch_errors?.values.rate || 0) * 100,
    thresholds_passed: Object.values(data.thresholds || {}).every(t => t.ok),
  };

  return {
    'results/batch-processing-test-summary.json': JSON.stringify(summary, null, 2),
    stdout: formatSummary(summary),
  };
}

function formatSummary(summary) {
  let output = '\n';
  output += '=== Batch Processing Performance Test Summary ===\n\n';
  output += `Test Duration: ${summary.duration_seconds.toFixed(2)}s\n`;
  output += `Total Batches: ${summary.total_batches}\n`;
  output += `Total Messages: ${summary.total_messages}\n`;
  output += `Avg Batch Processing Time: ${summary.avg_batch_time_ms.toFixed(2)}ms\n`;
  output += `P95 Batch Processing Time: ${summary.p95_batch_time_ms.toFixed(2)}ms\n`;
  output += `Error Rate: ${summary.error_rate.toFixed(2)}%\n`;
  output += `Thresholds: ${summary.thresholds_passed ? '✓ PASS' : '✗ FAIL'}\n\n`;
  
  // Calculate messages per second
  const messagesPerSecond = summary.total_messages / summary.duration_seconds;
  output += `Messages/Second: ${messagesPerSecond.toFixed(2)}\n`;
  output += `Messages/Minute: ${(messagesPerSecond * 60).toFixed(2)}\n\n`;
  
  return output;
}
