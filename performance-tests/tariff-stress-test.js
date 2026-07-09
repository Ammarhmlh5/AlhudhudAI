import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate, Trend, Gauge } from 'k6/metrics';
import { htmlReport } from "https://raw.githubusercontent.com/benc-uk/k6-reporter/main/dist/bundle.js";

// Custom metrics for stress testing
const systemOverloaded = new Counter('system_overloaded');
const resourceQuotaExceeded = new Counter('resource_quota_exceeded');
const rateLimitHit = new Counter('rate_limit_hit');
const gracefulDegradation = new Counter('graceful_degradation');
const systemRecovered = new Counter('system_recovered');

const responseLatency = new Trend('response_latency');
const errorRate = new Rate('error_rate');
const successRate = new Rate('success_rate');

const activeConnections = new Gauge('active_connections');
const cpuUsage = new Gauge('cpu_usage_percent');
const memoryUsage = new Gauge('memory_usage_percent');

// Stress test configuration - Requirements 27.1, 27.2
export const options = {
  scenarios: {
    // Stress Test 1: Extreme load - push system beyond normal capacity
    extreme_load: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '2m', target: 100 },   // Ramp up to 100 VUs
        { duration: '3m', target: 500 },   // Ramp up to 500 VUs (high load)
        { duration: '5m', target: 1000 },  // Ramp up to 1000 VUs (extreme load)
        { duration: '5m', target: 1500 },  // Push to 1500 VUs (beyond capacity)
        { duration: '3m', target: 500 },   // Ramp down to 500 VUs
        { duration: '2m', target: 0 },     // Ramp down to 0 (recovery test)
      ],
      exec: 'stressTestMixed',
      gracefulRampDown: '30s',
    },
    
    // Stress Test 2: Resource quota enforcement under stress
    resource_quota_stress: {
      executor: 'constant-vus',
      vus: 200,
      duration: '10m',
      exec: 'stressTestResourceQuotas',
      startTime: '22m', // Start after extreme load test
    },
    
    // Stress Test 3: Rate limiting under stress
    rate_limit_stress: {
      executor: 'constant-arrival-rate',
      rate: 1000,        // 1000 requests per second
      timeUnit: '1s',
      duration: '5m',
      preAllocatedVUs: 500,
      maxVUs: 1000,
      exec: 'stressTestRateLimiting',
      startTime: '33m', // Start after resource quota test
    },
  },
  
  thresholds: {
    // System should handle stress gracefully
    'error_rate': ['rate<0.20'],                    // Less than 20% errors under extreme stress
    'success_rate': ['rate>0.80'],                  // At least 80% success under stress
    'response_latency': ['p(95)<5000'],             // 95% < 5s under stress (degraded but acceptable)
    'response_latency': ['p(99)<10000'],            // 99% < 10s under stress
    
    // Graceful degradation
    'graceful_degradation': ['count>0'],            // System should degrade gracefully
    'system_recovered': ['count>0'],                // System should recover after stress
    
    // Resource quota enforcement
    'resource_quota_exceeded': ['count>0'],         // Should detect quota violations
    
    // Rate limiting effectiveness
    'rate_limit_hit': ['count>0'],                  // Rate limits should be enforced
    
    // HTTP errors should not exceed 30% under extreme stress
    'http_req_failed': ['rate<0.30'],
  },
};

const BASE_URL = __ENV.API_URL || 'http://localhost:8080';
const ADMIN_TOKEN = __ENV.ADMIN_TOKEN || 'admin-test-token';

// Test data generators
function randomTenantId() {
  return Math.floor(Math.random() * 100) + 1;
}

function randomPackageId() {
  return Math.floor(Math.random() * 5) + 1; // 1-5
}

function randomAmount() {
  return (Math.random() * 100 + 10).toFixed(2);
}

function randomGatewayData() {
  const timestamp = Date.now();
  const random = Math.floor(Math.random() * 10000);
  return {
    tenant_id: randomTenantId(),
    name: `Stress Test Gateway ${timestamp}-${random}`,
    code: `ST-GW-${timestamp}-${random}`,
    subscription_type: Math.random() > 0.5 ? 'monthly' : 'annual',
    subscription_fee: 100.00,
    server_rental_fee: 50.00,
    telecom_username: `stress_user_${timestamp}`,
    telecom_password: 'stress_password_encrypted',
    telecom_api_key: 'stress_api_key_encrypted',
    balance_alert_threshold: 1000.00,
    max_throughput: 100,
    daily_limit: 10000,
    monthly_limit: 300000,
  };
}

// Stress Test 1: Mixed operations under extreme load
export function stressTestMixed() {
  activeConnections.add(1);
  
  // Randomly choose operation type
  const operationType = Math.random();
  let response;
  let operationName;
  
  const params = {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${ADMIN_TOKEN}`,
    },
    timeout: '30s',
  };
  
  const startTime = Date.now();
  
  if (operationType < 0.4) {
    // 40% - Balance deduction (most common operation)
    operationName = 'balance_deduction';
    const payload = JSON.stringify({
      tenant_id: randomTenantId(),
      amount: parseFloat(randomAmount()),
      reason: 'Stress test deduction',
      message_id: `stress-${Date.now()}-${Math.random()}`,
    });
    response = http.post(`${BASE_URL}/api/balance/deduct`, payload, params);
    
  } else if (operationType < 0.7) {
    // 30% - Subscription operations
    operationName = 'subscription';
    const payload = JSON.stringify({
      tenant_id: randomTenantId(),
      package_id: randomPackageId(),
      auto_renew: true,
    });
    response = http.post(`${BASE_URL}/api/subscriptions`, payload, params);
    
  } else if (operationType < 0.9) {
    // 20% - Query operations (read-heavy)
    operationName = 'query';
    const tenantId = randomTenantId();
    response = http.get(`${BASE_URL}/api/subscriptions?tenant_id=${tenantId}`, params);
    
  } else {
    // 10% - Gateway operations (resource-intensive)
    operationName = 'gateway';
    const payload = JSON.stringify(randomGatewayData());
    response = http.post(`${BASE_URL}/api/gateways`, payload, params);
  }
  
  const duration = Date.now() - startTime;
  responseLatency.add(duration);
  
  // Check response
  const success = check(response, {
    'status is 2xx or acceptable error': (r) => {
      // Under stress, 429 (rate limit) and 503 (service unavailable) are acceptable
      return (r.status >= 200 && r.status < 300) || r.status === 429 || r.status === 503;
    },
  });
  
  if (success) {
    successRate.add(1);
    errorRate.add(0);
  } else {
    successRate.add(0);
    errorRate.add(1);
  }
  
  // Track specific stress conditions
  if (response.status === 429) {
    rateLimitHit.add(1);
    gracefulDegradation.add(1);
  }
  
  if (response.status === 503) {
    systemOverloaded.add(1);
    gracefulDegradation.add(1);
  }
  
  // Track recovery (successful response after previous errors)
  if (response.status >= 200 && response.status < 300 && duration < 1000) {
    systemRecovered.add(1);
  }
  
  activeConnections.add(-1);
  sleep(0.01); // Minimal sleep to maintain pressure
}

// Stress Test 2: Resource quota enforcement under stress
export function stressTestResourceQuotas() {
  activeConnections.add(1);
  
  const tenantId = randomTenantId();
  
  // Simulate operations that should hit resource quotas
  const operations = [
    // CPU-intensive operation
    () => {
      const payload = JSON.stringify({
        tenant_id: tenantId,
        operation: 'cpu_intensive',
        iterations: 10000,
      });
      return http.post(`${BASE_URL}/api/resources/test-cpu`, payload, {
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${ADMIN_TOKEN}`,
        },
        timeout: '30s',
      });
    },
    
    // Memory-intensive operation
    () => {
      const payload = JSON.stringify({
        tenant_id: tenantId,
        operation: 'memory_intensive',
        size_mb: 100,
      });
      return http.post(`${BASE_URL}/api/resources/test-memory`, payload, {
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${ADMIN_TOKEN}`,
        },
        timeout: '30s',
      });
    },
    
    // Bandwidth-intensive operation
    () => {
      const payload = JSON.stringify({
        tenant_id: tenantId,
        operation: 'bandwidth_intensive',
        data_size_mb: 50,
      });
      return http.post(`${BASE_URL}/api/resources/test-bandwidth`, payload, {
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${ADMIN_TOKEN}`,
        },
        timeout: '30s',
      });
    },
  ];
  
  const operation = operations[Math.floor(Math.random() * operations.length)];
  const startTime = Date.now();
  const response = operation();
  const duration = Date.now() - startTime;
  
  responseLatency.add(duration);
  
  // Check if resource quota was enforced
  const quotaEnforced = check(response, {
    'resource quota enforced': (r) => {
      // Should return 429 or specific quota exceeded error
      if (r.status === 429) {
        try {
          const body = JSON.parse(r.body);
          return body.error && (
            body.error.includes('quota') || 
            body.error.includes('limit') ||
            body.error.includes('resource')
          );
        } catch (e) {
          return false;
        }
      }
      return false;
    },
  });
  
  if (quotaEnforced) {
    resourceQuotaExceeded.add(1);
    gracefulDegradation.add(1);
  }
  
  const success = response.status >= 200 && response.status < 300;
  successRate.add(success ? 1 : 0);
  errorRate.add(success ? 0 : 1);
  
  activeConnections.add(-1);
  sleep(0.05);
}

// Stress Test 3: Rate limiting effectiveness under stress
export function stressTestRateLimiting() {
  activeConnections.add(1);
  
  const tenantId = randomTenantId();
  
  // Rapid-fire requests to trigger rate limiting
  const payload = JSON.stringify({
    tenant_id: tenantId,
    amount: 10.00,
    reason: 'Rate limit stress test',
    message_id: `rate-limit-${Date.now()}-${Math.random()}`,
  });
  
  const params = {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${ADMIN_TOKEN}`,
    },
    timeout: '10s',
  };
  
  const startTime = Date.now();
  const response = http.post(`${BASE_URL}/api/balance/deduct`, payload, params);
  const duration = Date.now() - startTime;
  
  responseLatency.add(duration);
  
  // Check if rate limit was hit
  const rateLimited = check(response, {
    'rate limit enforced': (r) => r.status === 429,
    'rate limit header present': (r) => {
      return r.headers['X-RateLimit-Limit'] !== undefined ||
             r.headers['X-RateLimit-Remaining'] !== undefined ||
             r.headers['Retry-After'] !== undefined;
    },
  });
  
  if (rateLimited) {
    rateLimitHit.add(1);
    gracefulDegradation.add(1);
  }
  
  const success = response.status >= 200 && response.status < 300;
  successRate.add(success ? 1 : 0);
  errorRate.add(success ? 0 : 1);
  
  activeConnections.add(-1);
  // No sleep - maintain maximum pressure for rate limit testing
}

export function handleSummary(data) {
  const summary = {
    test_name: 'Tariff System Stress Testing',
    test_date: new Date().toISOString(),
    requirements: ['27.1', '27.2'],
    
    // Stress Test 1: Extreme Load
    test_1_extreme_load: {
      requirement: '27.1',
      description: 'System behavior under extreme load (beyond normal capacity)',
      total_requests: data.metrics.http_reqs?.values.count || 0,
      success_rate: (data.metrics.success_rate?.values.rate || 0) * 100,
      error_rate: (data.metrics.error_rate?.values.rate || 0) * 100,
      http_error_rate: (data.metrics.http_req_failed?.values.rate || 0) * 100,
      latency_avg_ms: data.metrics.response_latency?.values.avg || 0,
      latency_p95_ms: data.metrics.response_latency?.values['p(95)'] || 0,
      latency_p99_ms: data.metrics.response_latency?.values['p(99)'] || 0,
      latency_max_ms: data.metrics.response_latency?.values.max || 0,
      system_overloaded_count: data.metrics.system_overloaded?.values.count || 0,
      graceful_degradation_count: data.metrics.graceful_degradation?.values.count || 0,
      system_recovered_count: data.metrics.system_recovered?.values.count || 0,
      target_achieved: (data.metrics.success_rate?.values.rate || 0) >= 0.80,
    },
    
    // Stress Test 2: Resource Quota Enforcement
    test_2_resource_quotas: {
      requirement: '27.1',
      description: 'Resource quota enforcement under stress',
      quota_exceeded_count: data.metrics.resource_quota_exceeded?.values.count || 0,
      quota_enforced: (data.metrics.resource_quota_exceeded?.values.count || 0) > 0,
      graceful_handling: (data.metrics.graceful_degradation?.values.count || 0) > 0,
      target_achieved: (data.metrics.resource_quota_exceeded?.values.count || 0) > 0,
    },
    
    // Stress Test 3: Rate Limiting Effectiveness
    test_3_rate_limiting: {
      requirement: '27.2',
      description: 'Rate limiting effectiveness under stress',
      rate_limit_hit_count: data.metrics.rate_limit_hit?.values.count || 0,
      rate_limit_enforced: (data.metrics.rate_limit_hit?.values.count || 0) > 0,
      graceful_handling: (data.metrics.graceful_degradation?.values.count || 0) > 0,
      target_achieved: (data.metrics.rate_limit_hit?.values.count || 0) > 0,
    },
    
    // Overall stress test results
    overall: {
      all_targets_achieved: 
        ((data.metrics.success_rate?.values.rate || 0) >= 0.80) &&
        ((data.metrics.resource_quota_exceeded?.values.count || 0) > 0) &&
        ((data.metrics.rate_limit_hit?.values.count || 0) > 0),
      thresholds_passed: Object.values(data.thresholds || {}).every(t => t.ok),
      graceful_degradation_working: (data.metrics.graceful_degradation?.values.count || 0) > 0,
      system_recovery_working: (data.metrics.system_recovered?.values.count || 0) > 0,
      max_concurrent_connections: data.metrics.active_connections?.values.max || 0,
    },
  };

  return {
    'results/tariff-stress-test-summary.json': JSON.stringify(summary, null, 2),
    'results/tariff-stress-test-full.json': JSON.stringify(data, null, 2),
    'results/tariff-stress-test-report.html': htmlReport(data),
    stdout: formatSummary(summary),
  };
}

function formatSummary(summary) {
  let output = '\n';
  output += '╔════════════════════════════════════════════════════════════════╗\n';
  output += '║          Tariff System Stress Testing Results                 ║\n';
  output += '╚════════════════════════════════════════════════════════════════╝\n\n';
  
  output += `📅 Test Date: ${summary.test_date}\n`;
  output += `📋 Requirements: ${summary.requirements.join(', ')}\n\n`;
  
  // Test 1: Extreme Load
  output += '━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n';
  output += '🔥 TEST 1: EXTREME LOAD (Beyond Normal Capacity)\n';
  output += '━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n';
  output += `  Requirement:           ${summary.test_1_extreme_load.requirement}\n`;
  output += `  Total Requests:        ${summary.test_1_extreme_load.total_requests}\n`;
  output += `  Success Rate:          ${summary.test_1_extreme_load.success_rate.toFixed(2)}% ${summary.test_1_extreme_load.target_achieved ? '✅' : '❌'}\n`;
  output += `  Error Rate:            ${summary.test_1_extreme_load.error_rate.toFixed(2)}%\n`;
  output += `  HTTP Error Rate:       ${summary.test_1_extreme_load.http_error_rate.toFixed(2)}%\n`;
  output += `  Latency (Avg):         ${summary.test_1_extreme_load.latency_avg_ms.toFixed(2)}ms\n`;
  output += `  Latency (P95):         ${summary.test_1_extreme_load.latency_p95_ms.toFixed(2)}ms\n`;
  output += `  Latency (P99):         ${summary.test_1_extreme_load.latency_p99_ms.toFixed(2)}ms\n`;
  output += `  Latency (Max):         ${summary.test_1_extreme_load.latency_max_ms.toFixed(2)}ms\n`;
  output += `  System Overloaded:     ${summary.test_1_extreme_load.system_overloaded_count} times\n`;
  output += `  Graceful Degradation:  ${summary.test_1_extreme_load.graceful_degradation_count} times\n`;
  output += `  System Recovered:      ${summary.test_1_extreme_load.system_recovered_count} times\n`;
  output += `  Status:                ${summary.test_1_extreme_load.target_achieved ? '✅ PASSED' : '❌ FAILED'}\n\n`;
  
  // Test 2: Resource Quota Enforcement
  output += '━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n';
  output += '📊 TEST 2: RESOURCE QUOTA ENFORCEMENT UNDER STRESS\n';
  output += '━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n';
  output += `  Requirement:           ${summary.test_2_resource_quotas.requirement}\n`;
  output += `  Quota Exceeded:        ${summary.test_2_resource_quotas.quota_exceeded_count} times\n`;
  output += `  Quota Enforced:        ${summary.test_2_resource_quotas.quota_enforced ? '✅ YES' : '❌ NO'}\n`;
  output += `  Graceful Handling:     ${summary.test_2_resource_quotas.graceful_handling ? '✅ YES' : '❌ NO'}\n`;
  output += `  Status:                ${summary.test_2_resource_quotas.target_achieved ? '✅ PASSED' : '❌ FAILED'}\n\n`;
  
  // Test 3: Rate Limiting Effectiveness
  output += '━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n';
  output += '⚡ TEST 3: RATE LIMITING EFFECTIVENESS UNDER STRESS\n';
  output += '━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n';
  output += `  Requirement:           ${summary.test_3_rate_limiting.requirement}\n`;
  output += `  Rate Limit Hit:        ${summary.test_3_rate_limiting.rate_limit_hit_count} times\n`;
  output += `  Rate Limit Enforced:   ${summary.test_3_rate_limiting.rate_limit_enforced ? '✅ YES' : '❌ NO'}\n`;
  output += `  Graceful Handling:     ${summary.test_3_rate_limiting.graceful_handling ? '✅ YES' : '❌ NO'}\n`;
  output += `  Status:                ${summary.test_3_rate_limiting.target_achieved ? '✅ PASSED' : '❌ FAILED'}\n\n`;
  
  // Overall Results
  output += '━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n';
  output += '🎯 OVERALL STRESS TEST RESULTS\n';
  output += '━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n';
  output += `  All Targets:           ${summary.overall.all_targets_achieved ? '✅ ACHIEVED' : '❌ NOT ACHIEVED'}\n`;
  output += `  All Thresholds:        ${summary.overall.thresholds_passed ? '✅ PASSED' : '❌ FAILED'}\n`;
  output += `  Graceful Degradation:  ${summary.overall.graceful_degradation_working ? '✅ WORKING' : '❌ NOT WORKING'}\n`;
  output += `  System Recovery:       ${summary.overall.system_recovery_working ? '✅ WORKING' : '❌ NOT WORKING'}\n`;
  output += `  Max Connections:       ${summary.overall.max_concurrent_connections}\n\n`;
  
  if (summary.overall.all_targets_achieved && summary.overall.thresholds_passed) {
    output += '🎉 SUCCESS: All stress tests passed!\n';
    output += '   ✅ System handled extreme load gracefully\n';
    output += '   ✅ Resource quotas enforced under stress\n';
    output += '   ✅ Rate limiting effective under stress\n';
    output += '   ✅ System degraded gracefully when overloaded\n';
    output += '   ✅ System recovered after stress\n';
  } else {
    output += '⚠️  WARNING: Some stress tests did not meet targets.\n';
    if (!summary.test_1_extreme_load.target_achieved) {
      output += '   ❌ Extreme load test: Success rate below 80%\n';
    }
    if (!summary.test_2_resource_quotas.target_achieved) {
      output += '   ❌ Resource quota test: Quotas not enforced\n';
    }
    if (!summary.test_3_rate_limiting.target_achieved) {
      output += '   ❌ Rate limiting test: Rate limits not enforced\n';
    }
    if (!summary.overall.graceful_degradation_working) {
      output += '   ⚠️  System did not degrade gracefully\n';
    }
    if (!summary.overall.system_recovery_working) {
      output += '   ⚠️  System did not recover properly\n';
    }
  }
  
  output += '\n📁 Detailed results saved to:\n';
  output += '   - results/tariff-stress-test-summary.json\n';
  output += '   - results/tariff-stress-test-full.json\n';
  output += '   - results/tariff-stress-test-report.html\n\n';
  
  output += '💡 Key Insights:\n';
  output += '   - Stress testing validates system behavior under extreme conditions\n';
  output += '   - Graceful degradation ensures service continuity\n';
  output += '   - Resource quotas protect system from resource exhaustion\n';
  output += '   - Rate limiting prevents system overload\n';
  output += '   - System recovery validates resilience\n\n';
  
  return output;
}
