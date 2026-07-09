import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate, Trend, Gauge } from 'k6/metrics';
import { htmlReport } from "https://raw.githubusercontent.com/benc-uk/k6-reporter/main/dist/bundle.js";

// Custom metrics
const messagesSent = new Counter('messages_sent');
const messagesQueued = new Counter('messages_queued');
const messagesDelivered = new Counter('messages_delivered');
const messagesFailed = new Counter('messages_failed');
const endToEndLatency = new Trend('end_to_end_latency');
const queueLatency = new Trend('queue_latency');
const processingLatency = new Trend('processing_latency');
const successRate = new Rate('success_rate');
const apiResponseTime = new Trend('api_response_time');
const currentQueueSize = new Gauge('current_queue_size');

// Test configuration - Target: 5000+ messages/minute (83+ messages/second)
export const options = {
  scenarios: {
    load_test: {
      executor: 'ramping-arrival-rate',
      startRate: 50,           // Start with 50 req/sec
      timeUnit: '1s',
      preAllocatedVUs: 100,
      maxVUs: 200,
      stages: [
        { duration: '1m', target: 50 },    // Warm up: 50 req/sec (3000/min)
        { duration: '2m', target: 85 },    // Ramp up: 85 req/sec (5100/min)
        { duration: '3m', target: 85 },    // Sustain: 85 req/sec (5100/min)
        { duration: '1m', target: 100 },   // Peak: 100 req/sec (6000/min)
        { duration: '2m', target: 100 },   // Sustain peak
        { duration: '1m', target: 0 },     // Ramp down
      ],
    },
  },
  thresholds: {
    'messages_sent': ['count>50000'],           // Should send 50000+ messages in 10 minutes
    'success_rate': ['rate>0.99'],              // 99%+ success rate
    'end_to_end_latency': ['p(95)<2000'],       // 95% complete in < 2 seconds
    'api_response_time': ['p(95)<10'],          // 95% API responses < 10ms
    'http_req_failed': ['rate<0.01'],           // Less than 1% HTTP errors
  },
};

const BASE_URL = __ENV.API_URL || 'http://localhost:8080';
const TENANT_IDS = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10];
const POLL_INTERVAL = 500; // Poll every 500ms
const MAX_POLL_ATTEMPTS = 20; // Max 10 seconds

function randomPhone() {
  const prefix = '+96650';
  const number = Math.floor(Math.random() * 10000000).toString().padStart(7, '0');
  return prefix + number;
}

function randomMessage() {
  const templates = [
    'Load test message',
    'Performance testing in progress',
    'System capacity test',
    'Throughput validation message',
    'End-to-end latency test',
  ];
  const template = templates[Math.floor(Math.random() * templates.length)];
  return `${template} - ${Date.now()}`;
}

function randomPriority() {
  const priorities = ['HIGH', 'NORMAL', 'NORMAL', 'NORMAL', 'LOW']; // 20% HIGH, 60% NORMAL, 20% LOW
  return priorities[Math.floor(Math.random() * priorities.length)];
}

export default function () {
  const tenantId = TENANT_IDS[Math.floor(Math.random() * TENANT_IDS.length)];
  
  const payload = JSON.stringify({
    tenant_id: tenantId,
    phone_number: randomPhone(),
    content: randomMessage(),
    priority: randomPriority(),
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
    timeout: '30s',
  };

  // Step 1: Send message via API
  const apiStartTime = Date.now();
  const response = http.post(`${BASE_URL}/api/messages`, payload, params);
  const apiDuration = Date.now() - apiStartTime;
  
  apiResponseTime.add(apiDuration);

  const apiCheck = check(response, {
    'API: status is 200': (r) => r.status === 200,
    'API: response time < 10ms': () => apiDuration < 10,
    'API: has message_id': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.message_id !== undefined && body.message_id !== '';
      } catch (e) {
        return false;
      }
    },
    'API: status is QUEUED': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.status === 'QUEUED';
      } catch (e) {
        return false;
      }
    },
  });

  if (response.status === 200) {
    messagesSent.add(1);
    
    try {
      const body = JSON.parse(response.body);
      const messageId = body.message_id;
      const queuedAt = Date.now();
      
      messagesQueued.add(1);
      
      // Step 2: Poll for message status (end-to-end flow)
      let finalStatus = null;
      let attempts = 0;
      let processingStartTime = null;
      
      while (finalStatus === null && attempts < MAX_POLL_ATTEMPTS) {
        sleep(POLL_INTERVAL / 1000); // Convert to seconds
        attempts++;
        
        const statusResponse = http.get(
          `${BASE_URL}/api/messages/${messageId}?tenant_id=${tenantId}`,
          params
        );
        
        if (statusResponse.status === 200) {
          const statusBody = JSON.parse(statusResponse.body);
          const status = statusBody.status;
          
          // Track when processing started
          if (processingStartTime === null && status !== 'QUEUED' && status !== 'PENDING') {
            processingStartTime = Date.now();
            const queueTime = processingStartTime - queuedAt;
            queueLatency.add(queueTime);
          }
          
          // Check for final status
          if (status === 'SENT' || status === 'DELIVERED') {
            finalStatus = 'SUCCESS';
            const endTime = Date.now();
            const totalLatency = endTime - apiStartTime;
            
            endToEndLatency.add(totalLatency);
            messagesDelivered.add(1);
            successRate.add(1);
            
            if (processingStartTime) {
              const procTime = endTime - processingStartTime;
              processingLatency.add(procTime);
            }
            
            break;
          } else if (status === 'FAILED') {
            finalStatus = 'FAILED';
            messagesFailed.add(1);
            successRate.add(0);
            break;
          }
        }
      }
      
      // Timeout case
      if (finalStatus === null && attempts >= MAX_POLL_ATTEMPTS) {
        successRate.add(0);
      }
      
    } catch (e) {
      console.error(`Error processing message: ${e.message}`);
      successRate.add(0);
    }
  } else {
    successRate.add(0);
  }

  // Small sleep to prevent overwhelming the system
  sleep(0.01);
}

export function handleSummary(data) {
  const durationSeconds = data.state.testRunDurationMs / 1000;
  const durationMinutes = durationSeconds / 60;
  
  const totalSent = data.metrics.messages_sent?.values.count || 0;
  const totalQueued = data.metrics.messages_queued?.values.count || 0;
  const totalDelivered = data.metrics.messages_delivered?.values.count || 0;
  const totalFailed = data.metrics.messages_failed?.values.count || 0;
  
  const messagesPerSecond = totalSent / durationSeconds;
  const messagesPerMinute = messagesPerSecond * 60;
  
  const successRateValue = data.metrics.success_rate?.values.rate || 0;
  
  const apiResponseTimeAvg = data.metrics.api_response_time?.values.avg || 0;
  const apiResponseTimeP95 = data.metrics.api_response_time?.values['p(95)'] || 0;
  const apiResponseTimeP99 = data.metrics.api_response_time?.values['p(99)'] || 0;
  
  const endToEndLatencyAvg = data.metrics.end_to_end_latency?.values.avg || 0;
  const endToEndLatencyP95 = data.metrics.end_to_end_latency?.values['p(95)'] || 0;
  const endToEndLatencyP99 = data.metrics.end_to_end_latency?.values['p(99)'] || 0;
  const endToEndLatencyMax = data.metrics.end_to_end_latency?.values.max || 0;
  
  const queueLatencyAvg = data.metrics.queue_latency?.values.avg || 0;
  const queueLatencyP95 = data.metrics.queue_latency?.values['p(95)'] || 0;
  
  const processingLatencyAvg = data.metrics.processing_latency?.values.avg || 0;
  const processingLatencyP95 = data.metrics.processing_latency?.values['p(95)'] || 0;
  
  const httpReqDuration = data.metrics.http_req_duration?.values.avg || 0;
  const httpReqFailed = data.metrics.http_req_failed?.values.rate || 0;
  
  const targetAchieved = messagesPerMinute >= 5000;
  const successRateAchieved = successRateValue >= 0.99;
  const latencyAchieved = endToEndLatencyP95 < 2000;
  const apiResponseAchieved = apiResponseTimeP95 < 10;
  
  const allThresholdsPassed = Object.values(data.thresholds || {}).every(t => t.ok);
  
  const summary = {
    test_name: 'Load Testing - 5000+ Messages/Minute',
    test_date: new Date().toISOString(),
    duration_seconds: durationSeconds,
    duration_minutes: durationMinutes,
    
    // Message statistics
    total_messages_sent: totalSent,
    total_messages_queued: totalQueued,
    total_messages_delivered: totalDelivered,
    total_messages_failed: totalFailed,
    
    // Throughput
    messages_per_second: messagesPerSecond,
    messages_per_minute: messagesPerMinute,
    target_throughput: 5000,
    throughput_achieved: targetAchieved,
    
    // Success rate
    success_rate_percent: successRateValue * 100,
    success_rate_target: 99,
    success_rate_achieved: successRateAchieved,
    
    // API response time
    api_response_time_avg_ms: apiResponseTimeAvg,
    api_response_time_p95_ms: apiResponseTimeP95,
    api_response_time_p99_ms: apiResponseTimeP99,
    api_response_time_target_ms: 10,
    api_response_achieved: apiResponseAchieved,
    
    // End-to-end latency
    end_to_end_latency_avg_ms: endToEndLatencyAvg,
    end_to_end_latency_p95_ms: endToEndLatencyP95,
    end_to_end_latency_p99_ms: endToEndLatencyP99,
    end_to_end_latency_max_ms: endToEndLatencyMax,
    end_to_end_latency_target_ms: 2000,
    latency_achieved: latencyAchieved,
    
    // Queue latency
    queue_latency_avg_ms: queueLatencyAvg,
    queue_latency_p95_ms: queueLatencyP95,
    
    // Processing latency
    processing_latency_avg_ms: processingLatencyAvg,
    processing_latency_p95_ms: processingLatencyP95,
    
    // HTTP metrics
    http_req_duration_avg_ms: httpReqDuration,
    http_req_failed_rate: httpReqFailed * 100,
    
    // Overall result
    all_targets_achieved: targetAchieved && successRateAchieved && latencyAchieved && apiResponseAchieved,
    thresholds_passed: allThresholdsPassed,
  };

  return {
    'results/load-test-summary.json': JSON.stringify(summary, null, 2),
    'results/load-test-full.json': JSON.stringify(data, null, 2),
    'results/load-test-report.html': htmlReport(data),
    stdout: formatSummary(summary),
  };
}

function formatSummary(summary) {
  let output = '\n';
  output += '╔════════════════════════════════════════════════════════════════╗\n';
  output += '║       Load Testing - 5000+ Messages/Minute                    ║\n';
  output += '╚════════════════════════════════════════════════════════════════╝\n\n';
  
  output += `📅 Test Date: ${summary.test_date}\n`;
  output += `⏱️  Duration: ${summary.duration_minutes.toFixed(2)} minutes (${summary.duration_seconds.toFixed(0)}s)\n\n`;
  
  output += '━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n';
  output += '📊 MESSAGE STATISTICS\n';
  output += '━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n';
  output += `  Total Sent:      ${summary.total_messages_sent.toLocaleString()}\n`;
  output += `  Total Queued:    ${summary.total_messages_queued.toLocaleString()}\n`;
  output += `  Total Delivered: ${summary.total_messages_delivered.toLocaleString()}\n`;
  output += `  Total Failed:    ${summary.total_messages_failed.toLocaleString()}\n`;
  output += `  Success Rate:    ${summary.success_rate_percent.toFixed(2)}% ${summary.success_rate_achieved ? '✅' : '❌'} (target: ${summary.success_rate_target}%)\n\n`;
  
  output += '━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n';
  output += '🚀 THROUGHPUT (Requirement 13.6)\n';
  output += '━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n';
  output += `  Messages/Second: ${summary.messages_per_second.toFixed(2)}\n`;
  output += `  Messages/Minute: ${summary.messages_per_minute.toFixed(2)} ${summary.throughput_achieved ? '✅' : '❌'}\n`;
  output += `  Target:          ${summary.target_throughput.toLocaleString()}/minute\n`;
  output += `  Achievement:     ${((summary.messages_per_minute / summary.target_throughput) * 100).toFixed(1)}%\n\n`;
  
  output += '━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n';
  output += '⚡ API RESPONSE TIME (Requirement 3.1)\n';
  output += '━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n';
  output += `  Average:  ${summary.api_response_time_avg_ms.toFixed(2)}ms\n`;
  output += `  P95:      ${summary.api_response_time_p95_ms.toFixed(2)}ms ${summary.api_response_achieved ? '✅' : '❌'}\n`;
  output += `  P99:      ${summary.api_response_time_p99_ms.toFixed(2)}ms\n`;
  output += `  Target:   < ${summary.api_response_time_target_ms}ms\n\n`;
  
  output += '━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n';
  output += '⏳ END-TO-END LATENCY (Requirement 21.6)\n';
  output += '━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n';
  output += `  Average:  ${summary.end_to_end_latency_avg_ms.toFixed(2)}ms\n`;
  output += `  P95:      ${summary.end_to_end_latency_p95_ms.toFixed(2)}ms ${summary.latency_achieved ? '✅' : '❌'}\n`;
  output += `  P99:      ${summary.end_to_end_latency_p99_ms.toFixed(2)}ms\n`;
  output += `  Max:      ${summary.end_to_end_latency_max_ms.toFixed(2)}ms\n`;
  output += `  Target:   < ${summary.end_to_end_latency_target_ms}ms\n\n`;
  
  output += '━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n';
  output += '📈 LATENCY BREAKDOWN\n';
  output += '━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n';
  output += `  Queue Latency (Avg):      ${summary.queue_latency_avg_ms.toFixed(2)}ms\n`;
  output += `  Queue Latency (P95):      ${summary.queue_latency_p95_ms.toFixed(2)}ms\n`;
  output += `  Processing Latency (Avg): ${summary.processing_latency_avg_ms.toFixed(2)}ms\n`;
  output += `  Processing Latency (P95): ${summary.processing_latency_p95_ms.toFixed(2)}ms\n\n`;
  
  output += '━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n';
  output += '🌐 HTTP METRICS\n';
  output += '━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n';
  output += `  Avg Request Duration: ${summary.http_req_duration_avg_ms.toFixed(2)}ms\n`;
  output += `  Failed Requests:      ${summary.http_req_failed_rate.toFixed(2)}%\n\n`;
  
  output += '━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n';
  output += '🎯 OVERALL RESULT\n';
  output += '━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n';
  output += `  Throughput Target:    ${summary.throughput_achieved ? '✅ ACHIEVED' : '❌ NOT ACHIEVED'}\n`;
  output += `  Success Rate Target:  ${summary.success_rate_achieved ? '✅ ACHIEVED' : '❌ NOT ACHIEVED'}\n`;
  output += `  Latency Target:       ${summary.latency_achieved ? '✅ ACHIEVED' : '❌ NOT ACHIEVED'}\n`;
  output += `  API Response Target:  ${summary.api_response_achieved ? '✅ ACHIEVED' : '❌ NOT ACHIEVED'}\n`;
  output += `  All Thresholds:       ${summary.thresholds_passed ? '✅ PASSED' : '❌ FAILED'}\n\n`;
  
  if (summary.all_targets_achieved) {
    output += '🎉 SUCCESS: All performance targets achieved!\n';
  } else {
    output += '⚠️  WARNING: Some performance targets not achieved.\n';
  }
  
  output += '\n📁 Detailed results saved to:\n';
  output += '   - results/load-test-summary.json\n';
  output += '   - results/load-test-full.json\n';
  output += '   - results/load-test-report.html\n\n';
  
  return output;
}
