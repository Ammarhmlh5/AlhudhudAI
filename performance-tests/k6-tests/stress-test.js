import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';

// Custom metrics
const totalRequests = new Counter('total_requests');
const successfulRequests = new Counter('successful_requests');
const failedRequests = new Counter('failed_requests');
const systemErrors = new Counter('system_errors');
const responseTime = new Trend('response_time_ms');

// Stress Test Configuration - Target: 10,000 messages/minute
export const options = {
  stages: [
    { duration: '1m', target: 50 },    // Warm up
    { duration: '2m', target: 100 },   // Normal load
    { duration: '2m', target: 200 },   // High load
    { duration: '2m', target: 300 },   // Stress load (10,000+ msg/min)
    { duration: '1m', target: 0 },     // Cool down
  ],
  thresholds: {
    'http_req_failed': ['rate<0.05'],     // Error rate < 5% under stress
    'response_time_ms': ['p(95)<100'],    // 95% responses < 100ms
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
  return `Stress test message ${Date.now()}`;
}

export default function () {
  const tenantId = TENANT_IDS[Math.floor(Math.random() * TENANT_IDS.length)];
  
  const payload = JSON.stringify({
    tenant_id: tenantId,
    phone_number: randomPhone(),
    content: randomMessage(),
    priority: Math.random() > 0.8 ? 'HIGH' : 'NORMAL', // 20% high priority
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
    timeout: '30s',
  };

  totalRequests.add(1);
  
  const startTime = Date.now();
  const response = http.post(`${BASE_URL}/api/messages`, payload, params);
  const duration = Date.now() - startTime;
  
  responseTime.add(duration);

  if (response.status === 200) {
    successfulRequests.add(1);
  } else if (response.status >= 500) {
    systemErrors.add(1);
    failedRequests.add(1);
  } else {
    failedRequests.add(1);
  }

  const checkResult = check(response, {
    'status is 200 or 429': (r) => r.status === 200 || r.status === 429,
    'response time acceptable': () => duration < 100,
    'has valid response': (r) => {
      try {
        JSON.parse(r.body);
        return true;
      } catch (e) {
        return false;
      }
    },
  });

  // Minimal sleep to maximize load
  sleep(0.01);
}

export function handleSummary(data) {
  const durationSeconds = data.state.testRunDurationMs / 1000;
  const totalReqs = data.metrics.total_requests?.values.count || 0;
  const successReqs = data.metrics.successful_requests?.values.count || 0;
  const failedReqs = data.metrics.failed_requests?.values.count || 0;
  const sysErrors = data.metrics.system_errors?.values.count || 0;
  
  const avgResponseTime = data.metrics.response_time_ms?.values.avg || 0;
  const p95ResponseTime = data.metrics.response_time_ms?.values['p(95)'] || 0;
  const p99ResponseTime = data.metrics.response_time_ms?.values['p(99)'] || 0;
  const maxResponseTime = data.metrics.response_time_ms?.values.max || 0;
  
  const requestsPerSecond = totalReqs / durationSeconds;
  const requestsPerMinute = requestsPerSecond * 60;
  const successRate = totalReqs > 0 ? (successReqs / totalReqs) * 100 : 0;
  const errorRate = totalReqs > 0 ? (failedReqs / totalReqs) * 100 : 0;
  
  const summary = {
    test_name: 'Stress Test - 10,000 Messages/Minute',
    duration_seconds: durationSeconds,
    total_requests: totalReqs,
    successful_requests: successReqs,
    failed_requests: failedReqs,
    system_errors: sysErrors,
    requests_per_second: requestsPerSecond,
    requests_per_minute: requestsPerMinute,
    success_rate_percent: successRate,
    error_rate_percent: errorRate,
    avg_response_time_ms: avgResponseTime,
    p95_response_time_ms: p95ResponseTime,
    p99_response_time_ms: p99ResponseTime,
    max_response_time_ms: maxResponseTime,
    target_achieved: requestsPerMinute >= 10000,
    thresholds_passed: Object.values(data.thresholds || {}).every(t => t.ok),
  };

  return {
    'results/stress-test-summary.json': JSON.stringify(summary, null, 2),
    stdout: formatSummary(summary),
  };
}

function formatSummary(summary) {
  let output = '\n';
  output += '╔════════════════════════════════════════════════════════╗\n';
  output += '║     Stress Test - 10,000 Messages/Minute Summary      ║\n';
  output += '╚════════════════════════════════════════════════════════╝\n\n';
  
  output += `Test Duration: ${summary.duration_seconds.toFixed(2)}s\n\n`;
  
  output += '📊 Request Statistics:\n';
  output += `   Total Requests:      ${summary.total_requests.toLocaleString()}\n`;
  output += `   Successful:          ${summary.successful_requests.toLocaleString()} (${summary.success_rate_percent.toFixed(2)}%)\n`;
  output += `   Failed:              ${summary.failed_requests.toLocaleString()} (${summary.error_rate_percent.toFixed(2)}%)\n`;
  output += `   System Errors (5xx): ${summary.system_errors.toLocaleString()}\n\n`;
  
  output += '🚀 Throughput:\n';
  output += `   Requests/Second:     ${summary.requests_per_second.toFixed(2)}\n`;
  output += `   Requests/Minute:     ${summary.requests_per_minute.toFixed(2)}\n`;
  output += `   Target (10,000/min): ${summary.target_achieved ? '✓ ACHIEVED' : '✗ NOT ACHIEVED'}\n\n`;
  
  output += '⏱️  Response Time:\n';
  output += `   Average:             ${summary.avg_response_time_ms.toFixed(2)}ms\n`;
  output += `   P95:                 ${summary.p95_response_time_ms.toFixed(2)}ms\n`;
  output += `   P99:                 ${summary.p99_response_time_ms.toFixed(2)}ms\n`;
  output += `   Max:                 ${summary.max_response_time_ms.toFixed(2)}ms\n\n`;
  
  output += '🎯 Overall Result:\n';
  output += `   ${summary.thresholds_passed ? '✓ PASS - System handled stress successfully!' : '✗ FAIL - System struggled under stress'}\n\n`;
  
  if (!summary.target_achieved) {
    output += '⚠️  Warning: Target throughput not achieved. Consider:\n';
    output += '   - Increasing worker pool size\n';
    output += '   - Optimizing database queries\n';
    output += '   - Scaling horizontally\n\n';
  }
  
  if (summary.error_rate_percent > 5) {
    output += '⚠️  Warning: High error rate detected. Check:\n';
    output += '   - System logs for errors\n';
    output += '   - Database connection pool\n';
    output += '   - Redis memory usage\n\n';
  }
  
  return output;
}
