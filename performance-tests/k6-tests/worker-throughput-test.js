import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';

// Custom metrics
const messagesSent = new Counter('messages_sent');
const messagesDelivered = new Counter('messages_delivered');
const messagesFailed = new Counter('messages_failed');
const endToEndLatency = new Trend('end_to_end_latency');
const successRate = new Rate('success_rate');

// Test configuration - Target: 5000+ messages/minute
export const options = {
  scenarios: {
    throughput_test: {
      executor: 'constant-arrival-rate',
      rate: 100,              // 100 requests per second
      timeUnit: '1s',
      duration: '2m',
      preAllocatedVUs: 50,
      maxVUs: 100,
    },
  },
  thresholds: {
    'messages_sent': ['count>10000'],        // Should send 10000+ messages in 2 minutes
    'success_rate': ['rate>0.99'],           // 99%+ success rate
    'end_to_end_latency': ['p(95)<2000'],    // 95% complete in < 2 seconds
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
  return `Throughput test message ${Date.now()}`;
}

export default function () {
  const tenantId = TENANT_IDS[Math.floor(Math.random() * TENANT_IDS.length)];
  
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

  // Send message
  const startTime = Date.now();
  const response = http.post(`${BASE_URL}/api/messages`, payload, params);
  
  if (response.status === 200) {
    messagesSent.add(1);
    
    try {
      const body = JSON.parse(response.body);
      const messageId = body.message_id;
      
      // Poll for message status (simulating end-to-end flow)
      let delivered = false;
      let attempts = 0;
      const maxAttempts = 20; // Max 10 seconds (20 * 500ms)
      
      while (!delivered && attempts < maxAttempts) {
        sleep(0.5);
        attempts++;
        
        const statusResponse = http.get(
          `${BASE_URL}/api/messages/${messageId}?tenant_id=${tenantId}`,
          params
        );
        
        if (statusResponse.status === 200) {
          const statusBody = JSON.parse(statusResponse.body);
          const status = statusBody.status;
          
          if (status === 'SENT' || status === 'DELIVERED') {
            delivered = true;
            const latency = Date.now() - startTime;
            endToEndLatency.add(latency);
            messagesDelivered.add(1);
            successRate.add(1);
            break;
          } else if (status === 'FAILED') {
            messagesFailed.add(1);
            successRate.add(0);
            break;
          }
        }
      }
      
      if (!delivered && attempts >= maxAttempts) {
        // Timeout
        successRate.add(0);
      }
    } catch (e) {
      successRate.add(0);
    }
  } else {
    successRate.add(0);
  }

  check(response, {
    'message sent successfully': (r) => r.status === 200,
  });
}

export function handleSummary(data) {
  const durationSeconds = data.state.testRunDurationMs / 1000;
  const totalSent = data.metrics.messages_sent?.values.count || 0;
  const totalDelivered = data.metrics.messages_delivered?.values.count || 0;
  const totalFailed = data.metrics.messages_failed?.values.count || 0;
  const avgLatency = data.metrics.end_to_end_latency?.values.avg || 0;
  const p95Latency = data.metrics.end_to_end_latency?.values['p(95)'] || 0;
  const successRateValue = data.metrics.success_rate?.values.rate || 0;
  
  const messagesPerSecond = totalSent / durationSeconds;
  const messagesPerMinute = messagesPerSecond * 60;
  
  const summary = {
    test_name: 'Worker Throughput Performance Test',
    duration_seconds: durationSeconds,
    total_messages_sent: totalSent,
    total_messages_delivered: totalDelivered,
    total_messages_failed: totalFailed,
    messages_per_second: messagesPerSecond,
    messages_per_minute: messagesPerMinute,
    success_rate_percent: successRateValue * 100,
    avg_latency_ms: avgLatency,
    p95_latency_ms: p95Latency,
    target_achieved: messagesPerMinute >= 5000,
    thresholds_passed: Object.values(data.thresholds || {}).every(t => t.ok),
  };

  return {
    'results/worker-throughput-test-summary.json': JSON.stringify(summary, null, 2),
    stdout: formatSummary(summary),
  };
}

function formatSummary(summary) {
  let output = '\n';
  output += '=== Worker Throughput Performance Test Summary ===\n\n';
  output += `Test Duration: ${summary.duration_seconds.toFixed(2)}s\n\n`;
  
  output += 'Message Statistics:\n';
  output += `  Total Sent: ${summary.total_messages_sent}\n`;
  output += `  Total Delivered: ${summary.total_messages_delivered}\n`;
  output += `  Total Failed: ${summary.total_messages_failed}\n`;
  output += `  Success Rate: ${summary.success_rate_percent.toFixed(2)}%\n\n`;
  
  output += 'Throughput:\n';
  output += `  Messages/Second: ${summary.messages_per_second.toFixed(2)}\n`;
  output += `  Messages/Minute: ${summary.messages_per_minute.toFixed(2)}\n`;
  output += `  Target (5000/min): ${summary.target_achieved ? '✓ ACHIEVED' : '✗ NOT ACHIEVED'}\n\n`;
  
  output += 'Latency:\n';
  output += `  Average: ${summary.avg_latency_ms.toFixed(2)}ms\n`;
  output += `  P95: ${summary.p95_latency_ms.toFixed(2)}ms\n\n`;
  
  output += `Overall: ${summary.thresholds_passed ? '✓ PASS' : '✗ FAIL'}\n\n`;
  
  return output;
}
