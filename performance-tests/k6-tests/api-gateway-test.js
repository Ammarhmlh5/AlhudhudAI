import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const responseTime = new Trend('response_time');

// Test configuration
export const options = {
  stages: [
    { duration: '30s', target: 50 },   // Ramp up to 50 users
    { duration: '1m', target: 100 },   // Stay at 100 users
    { duration: '30s', target: 0 },    // Ramp down to 0 users
  ],
  thresholds: {
    'http_req_duration': ['p(95)<10'],  // 95% of requests must complete below 10ms
    'errors': ['rate<0.01'],            // Error rate must be below 1%
    'http_reqs': ['rate>1000'],         // Must handle 1000+ requests/second
  },
};

// Test data
const BASE_URL = __ENV.API_URL || 'http://localhost:8080';
const TENANT_IDS = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]; // Test tenant IDs

// Generate random phone number
function randomPhone() {
  const prefix = '+96650';
  const number = Math.floor(Math.random() * 10000000).toString().padStart(7, '0');
  return prefix + number;
}

// Generate random message content
function randomMessage() {
  const messages = [
    'Test message for performance testing',
    'Hello from k6 load test',
    'Performance test message',
    'Load testing in progress',
    'API Gateway performance test',
  ];
  return messages[Math.floor(Math.random() * messages.length)];
}

// Main test function
export default function () {
  // Select random tenant
  const tenantId = TENANT_IDS[Math.floor(Math.random() * TENANT_IDS.length)];
  
  // Prepare request payload
  const payload = JSON.stringify({
    tenant_id: tenantId,
    phone_number: randomPhone(),
    content: randomMessage(),
    priority: 'NORMAL',
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
  };

  // Send request
  const startTime = Date.now();
  const response = http.post(`${BASE_URL}/api/messages`, payload, params);
  const duration = Date.now() - startTime;

  // Record custom metrics
  responseTime.add(duration);
  errorRate.add(response.status !== 200);

  // Validate response
  const checkResult = check(response, {
    'status is 200': (r) => r.status === 200,
    'response time < 10ms': () => duration < 10,
    'has message_id': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.message_id !== undefined;
      } catch (e) {
        return false;
      }
    },
    'status is QUEUED': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.status === 'QUEUED';
      } catch (e) {
        return false;
      }
    },
  });

  // Small sleep to simulate real user behavior
  sleep(0.1);
}

// Summary handler
export function handleSummary(data) {
  return {
    'results/api-gateway-test-summary.json': JSON.stringify(data, null, 2),
    stdout: textSummary(data, { indent: ' ', enableColors: true }),
  };
}

function textSummary(data, options) {
  const indent = options.indent || '';
  const enableColors = options.enableColors || false;
  
  let summary = '\n';
  summary += indent + '=== API Gateway Performance Test Summary ===\n\n';
  
  // Test duration
  summary += indent + `Test Duration: ${data.state.testRunDurationMs / 1000}s\n\n`;
  
  // HTTP metrics
  const httpReqs = data.metrics.http_reqs;
  const httpReqDuration = data.metrics.http_req_duration;
  
  summary += indent + 'HTTP Metrics:\n';
  summary += indent + `  Total Requests: ${httpReqs.values.count}\n`;
  summary += indent + `  Requests/sec: ${httpReqs.values.rate.toFixed(2)}\n`;
  summary += indent + `  Avg Response Time: ${httpReqDuration.values.avg.toFixed(2)}ms\n`;
  summary += indent + `  Min Response Time: ${httpReqDuration.values.min.toFixed(2)}ms\n`;
  summary += indent + `  Max Response Time: ${httpReqDuration.values.max.toFixed(2)}ms\n`;
  summary += indent + `  P95 Response Time: ${httpReqDuration.values['p(95)'].toFixed(2)}ms\n`;
  summary += indent + `  P99 Response Time: ${httpReqDuration.values['p(99)'].toFixed(2)}ms\n\n`;
  
  // Checks
  const checks = data.metrics.checks;
  summary += indent + 'Checks:\n';
  summary += indent + `  Success Rate: ${(checks.values.rate * 100).toFixed(2)}%\n`;
  summary += indent + `  Passed: ${checks.values.passes}\n`;
  summary += indent + `  Failed: ${checks.values.fails}\n\n`;
  
  // Custom metrics
  if (data.metrics.errors) {
    const errors = data.metrics.errors;
    summary += indent + 'Errors:\n';
    summary += indent + `  Error Rate: ${(errors.values.rate * 100).toFixed(2)}%\n\n`;
  }
  
  // Thresholds
  summary += indent + 'Thresholds:\n';
  for (const [name, threshold] of Object.entries(data.thresholds || {})) {
    const status = threshold.ok ? '✓ PASS' : '✗ FAIL';
    summary += indent + `  ${status}: ${name}\n`;
  }
  
  return summary;
}
