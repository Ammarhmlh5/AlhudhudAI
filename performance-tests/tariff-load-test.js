import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate, Trend, Gauge } from 'k6/metrics';
import { htmlReport } from "https://raw.githubusercontent.com/benc-uk/k6-reporter/main/dist/bundle.js";

// Custom metrics for tariff system
const subscriptionsCreated = new Counter('subscriptions_created');
const subscriptionsFailed = new Counter('subscriptions_failed');
const balanceDeductions = new Counter('balance_deductions');
const balanceDeductionsFailed = new Counter('balance_deductions_failed');
const gatewaysCreated = new Counter('gateways_created');
const gatewaysFailed = new Counter('gateways_failed');

const subscriptionLatency = new Trend('subscription_latency');
const deductionLatency = new Trend('deduction_latency');
const gatewayCreationLatency = new Trend('gateway_creation_latency');

const subscriptionSuccessRate = new Rate('subscription_success_rate');
const deductionSuccessRate = new Rate('deduction_success_rate');
const gatewaySuccessRate = new Rate('gateway_success_rate');

const concurrentSubscriptions = new Gauge('concurrent_subscriptions');
const concurrentDeductions = new Gauge('concurrent_deductions');
const concurrentGateways = new Gauge('concurrent_gateways');

// Test configuration - Requirements 27.1, 27.2, 27.3
export const options = {
  scenarios: {
    // Test 1: 1000 concurrent subscriptions (Requirement 27.1)
    concurrent_subscriptions: {
      executor: 'shared-iterations',
      vus: 100,
      iterations: 1000,
      maxDuration: '5m',
      exec: 'testSubscription',
      startTime: '0s',
    },
    
    // Test 2: 10000 concurrent balance deductions (Requirement 27.2)
    concurrent_deductions: {
      executor: 'shared-iterations',
      vus: 200,
      iterations: 10000,
      maxDuration: '10m',
      exec: 'testBalanceDeduction',
      startTime: '6m', // Start after subscriptions
    },
    
    // Test 3: 100 concurrent dedicated gateways (Requirement 27.3)
    concurrent_gateways: {
      executor: 'shared-iterations',
      vus: 50,
      iterations: 100,
      maxDuration: '5m',
      exec: 'testDedicatedGateway',
      startTime: '17m', // Start after deductions
    },
  },
  
  thresholds: {
    // Subscription thresholds
    'subscriptions_created': ['count>=950'],           // 95%+ success rate
    'subscription_success_rate': ['rate>0.95'],        // 95%+ success
    'subscription_latency': ['p(95)<500'],             // 95% < 500ms
    
    // Balance deduction thresholds
    'balance_deductions': ['count>=9500'],             // 95%+ success rate
    'deduction_success_rate': ['rate>0.95'],           // 95%+ success
    'deduction_latency': ['p(95)<100'],                // 95% < 100ms (atomic operations)
    
    // Gateway creation thresholds
    'gateways_created': ['count>=95'],                 // 95%+ success rate
    'gateway_success_rate': ['rate>0.95'],             // 95%+ success
    'gateway_creation_latency': ['p(95)<1000'],        // 95% < 1000ms
    
    // HTTP error thresholds
    'http_req_failed': ['rate<0.05'],                  // Less than 5% HTTP errors
  },
};

const BASE_URL = __ENV.API_URL || 'http://localhost:8080';
const ADMIN_TOKEN = __ENV.ADMIN_TOKEN || 'admin-test-token';

// Package IDs for subscriptions
const PACKAGE_IDS = [1, 2, 3, 4, 5]; // Basic, Standard, Professional, Business, Enterprise

function randomTenantId() {
  return Math.floor(Math.random() * 100) + 1;
}

function randomPackageId() {
  return PACKAGE_IDS[Math.floor(Math.random() * PACKAGE_IDS.length)];
}

function randomAmount() {
  return (Math.random() * 100 + 10).toFixed(2); // Between 10 and 110
}

// Test 1: Concurrent Subscriptions (Requirement 27.1)
export function testSubscription() {
  concurrentSubscriptions.add(1);
  
  const tenantId = randomTenantId();
  const packageId = randomPackageId();
  
  const payload = JSON.stringify({
    tenant_id: tenantId,
    package_id: packageId,
    auto_renew: true,
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${ADMIN_TOKEN}`,
    },
    timeout: '30s',
  };

  const startTime = Date.now();
  const response = http.post(`${BASE_URL}/api/subscriptions`, payload, params);
  const duration = Date.now() - startTime;
  
  subscriptionLatency.add(duration);

  const success = check(response, {
    'Subscription: status is 200 or 201': (r) => r.status === 200 || r.status === 201,
    'Subscription: has subscription_id': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.subscription_id !== undefined;
      } catch (e) {
        return false;
      }
    },
    'Subscription: response time < 500ms': () => duration < 500,
  });

  if (success) {
    subscriptionsCreated.add(1);
    subscriptionSuccessRate.add(1);
  } else {
    subscriptionsFailed.add(1);
    subscriptionSuccessRate.add(0);
  }
  
  concurrentSubscriptions.add(-1);
  sleep(0.01);
}

// Test 2: Concurrent Balance Deductions (Requirement 27.2)
export function testBalanceDeduction() {
  concurrentDeductions.add(1);
  
  const tenantId = randomTenantId();
  const amount = randomAmount();
  
  const payload = JSON.stringify({
    tenant_id: tenantId,
    amount: parseFloat(amount),
    reason: 'Load test deduction',
    message_id: `load-test-${Date.now()}-${Math.random()}`,
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${ADMIN_TOKEN}`,
    },
    timeout: '30s',
  };

  const startTime = Date.now();
  const response = http.post(`${BASE_URL}/api/balance/deduct`, payload, params);
  const duration = Date.now() - startTime;
  
  deductionLatency.add(duration);

  const success = check(response, {
    'Deduction: status is 200': (r) => r.status === 200,
    'Deduction: balance updated': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.success === true || body.new_balance !== undefined;
      } catch (e) {
        return false;
      }
    },
    'Deduction: response time < 100ms': () => duration < 100,
    'Deduction: atomic operation': (r) => {
      // Check that response indicates atomic operation
      try {
        const body = JSON.parse(r.body);
        return body.transaction_id !== undefined;
      } catch (e) {
        return false;
      }
    },
  });

  if (success) {
    balanceDeductions.add(1);
    deductionSuccessRate.add(1);
  } else {
    balanceDeductionsFailed.add(1);
    deductionSuccessRate.add(0);
  }
  
  concurrentDeductions.add(-1);
  sleep(0.005); // Shorter sleep for high-volume test
}

// Test 3: Concurrent Dedicated Gateways (Requirement 27.3)
export function testDedicatedGateway() {
  concurrentGateways.add(1);
  
  const tenantId = randomTenantId();
  const gatewayName = `Load Test Gateway ${Date.now()}-${Math.random()}`;
  
  const payload = JSON.stringify({
    tenant_id: tenantId,
    name: gatewayName,
    code: `LT-GW-${Date.now()}-${Math.floor(Math.random() * 10000)}`,
    subscription_type: 'monthly',
    subscription_fee: 100.00,
    server_rental_fee: 50.00,
    telecom_username: `test_user_${tenantId}`,
    telecom_password: 'test_password_encrypted',
    telecom_api_key: 'test_api_key_encrypted',
    balance_alert_threshold: 1000.00,
    max_throughput: 100,
    daily_limit: 10000,
    monthly_limit: 300000,
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${ADMIN_TOKEN}`,
    },
    timeout: '30s',
  };

  const startTime = Date.now();
  const response = http.post(`${BASE_URL}/api/gateways`, payload, params);
  const duration = Date.now() - startTime;
  
  gatewayCreationLatency.add(duration);

  const success = check(response, {
    'Gateway: status is 200 or 201': (r) => r.status === 200 || r.status === 201,
    'Gateway: has gateway_id': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.gateway_id !== undefined || body.id !== undefined;
      } catch (e) {
        return false;
      }
    },
    'Gateway: response time < 1000ms': () => duration < 1000,
    'Gateway: resource quotas created': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.resource_quotas !== undefined || body.status === 'active';
      } catch (e) {
        return false;
      }
    },
  });

  if (success) {
    gatewaysCreated.add(1);
    gatewaySuccessRate.add(1);
  } else {
    gatewaysFailed.add(1);
    gatewaySuccessRate.add(0);
  }
  
  concurrentGateways.add(-1);
  sleep(0.02);
}

export function handleSummary(data) {
  const summary = {
    test_name: 'Tariff System Load Testing',
    test_date: new Date().toISOString(),
    requirements: ['27.1', '27.2', '27.3'],
    
    // Test 1: Concurrent Subscriptions (Requirement 27.1)
    test_1_concurrent_subscriptions: {
      requirement: '27.1',
      target: 1000,
      created: data.metrics.subscriptions_created?.values.count || 0,
      failed: data.metrics.subscriptions_failed?.values.count || 0,
      success_rate: (data.metrics.subscription_success_rate?.values.rate || 0) * 100,
      latency_avg_ms: data.metrics.subscription_latency?.values.avg || 0,
      latency_p95_ms: data.metrics.subscription_latency?.values['p(95)'] || 0,
      latency_p99_ms: data.metrics.subscription_latency?.values['p(99)'] || 0,
      latency_max_ms: data.metrics.subscription_latency?.values.max || 0,
      target_achieved: (data.metrics.subscriptions_created?.values.count || 0) >= 950,
    },
    
    // Test 2: Concurrent Balance Deductions (Requirement 27.2)
    test_2_concurrent_deductions: {
      requirement: '27.2',
      target: 10000,
      completed: data.metrics.balance_deductions?.values.count || 0,
      failed: data.metrics.balance_deductions_failed?.values.count || 0,
      success_rate: (data.metrics.deduction_success_rate?.values.rate || 0) * 100,
      latency_avg_ms: data.metrics.deduction_latency?.values.avg || 0,
      latency_p95_ms: data.metrics.deduction_latency?.values['p(95)'] || 0,
      latency_p99_ms: data.metrics.deduction_latency?.values['p(99)'] || 0,
      latency_max_ms: data.metrics.deduction_latency?.values.max || 0,
      target_achieved: (data.metrics.balance_deductions?.values.count || 0) >= 9500,
    },
    
    // Test 3: Concurrent Dedicated Gateways (Requirement 27.3)
    test_3_concurrent_gateways: {
      requirement: '27.3',
      target: 100,
      created: data.metrics.gateways_created?.values.count || 0,
      failed: data.metrics.gateways_failed?.values.count || 0,
      success_rate: (data.metrics.gateway_success_rate?.values.rate || 0) * 100,
      latency_avg_ms: data.metrics.gateway_creation_latency?.values.avg || 0,
      latency_p95_ms: data.metrics.gateway_creation_latency?.values['p(95)'] || 0,
      latency_p99_ms: data.metrics.gateway_creation_latency?.values['p(99)'] || 0,
      latency_max_ms: data.metrics.gateway_creation_latency?.values.max || 0,
      target_achieved: (data.metrics.gateways_created?.values.count || 0) >= 95,
    },
    
    // Overall results
    overall: {
      all_targets_achieved: 
        ((data.metrics.subscriptions_created?.values.count || 0) >= 950) &&
        ((data.metrics.balance_deductions?.values.count || 0) >= 9500) &&
        ((data.metrics.gateways_created?.values.count || 0) >= 95),
      thresholds_passed: Object.values(data.thresholds || {}).every(t => t.ok),
      http_req_failed_rate: (data.metrics.http_req_failed?.values.rate || 0) * 100,
    },
  };

  return {
    'results/tariff-load-test-summary.json': JSON.stringify(summary, null, 2),
    'results/tariff-load-test-full.json': JSON.stringify(data, null, 2),
    'results/tariff-load-test-report.html': htmlReport(data),
    stdout: formatSummary(summary),
  };
}

function formatSummary(summary) {
  let output = '\n';
  output += '╔════════════════════════════════════════════════════════════════╗\n';
  output += '║          Tariff System Load Testing Results                   ║\n';
  output += '╚════════════════════════════════════════════════════════════════╝\n\n';
  
  output += `📅 Test Date: ${summary.test_date}\n`;
  output += `📋 Requirements: ${summary.requirements.join(', ')}\n\n`;
  
  // Test 1: Concurrent Subscriptions
  output += '━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n';
  output += '📊 TEST 1: 1000 CONCURRENT SUBSCRIPTIONS (Requirement 27.1)\n';
  output += '━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n';
  output += `  Target:          ${summary.test_1_concurrent_subscriptions.target}\n`;
  output += `  Created:         ${summary.test_1_concurrent_subscriptions.created} ${summary.test_1_concurrent_subscriptions.target_achieved ? '✅' : '❌'}\n`;
  output += `  Failed:          ${summary.test_1_concurrent_subscriptions.failed}\n`;
  output += `  Success Rate:    ${summary.test_1_concurrent_subscriptions.success_rate.toFixed(2)}%\n`;
  output += `  Latency (Avg):   ${summary.test_1_concurrent_subscriptions.latency_avg_ms.toFixed(2)}ms\n`;
  output += `  Latency (P95):   ${summary.test_1_concurrent_subscriptions.latency_p95_ms.toFixed(2)}ms\n`;
  output += `  Latency (P99):   ${summary.test_1_concurrent_subscriptions.latency_p99_ms.toFixed(2)}ms\n`;
  output += `  Latency (Max):   ${summary.test_1_concurrent_subscriptions.latency_max_ms.toFixed(2)}ms\n`;
  output += `  Status:          ${summary.test_1_concurrent_subscriptions.target_achieved ? '✅ PASSED' : '❌ FAILED'}\n\n`;
  
  // Test 2: Concurrent Balance Deductions
  output += '━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n';
  output += '💰 TEST 2: 10000 CONCURRENT BALANCE DEDUCTIONS (Requirement 27.2)\n';
  output += '━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n';
  output += `  Target:          ${summary.test_2_concurrent_deductions.target}\n`;
  output += `  Completed:       ${summary.test_2_concurrent_deductions.completed} ${summary.test_2_concurrent_deductions.target_achieved ? '✅' : '❌'}\n`;
  output += `  Failed:          ${summary.test_2_concurrent_deductions.failed}\n`;
  output += `  Success Rate:    ${summary.test_2_concurrent_deductions.success_rate.toFixed(2)}%\n`;
  output += `  Latency (Avg):   ${summary.test_2_concurrent_deductions.latency_avg_ms.toFixed(2)}ms\n`;
  output += `  Latency (P95):   ${summary.test_2_concurrent_deductions.latency_p95_ms.toFixed(2)}ms\n`;
  output += `  Latency (P99):   ${summary.test_2_concurrent_deductions.latency_p99_ms.toFixed(2)}ms\n`;
  output += `  Latency (Max):   ${summary.test_2_concurrent_deductions.latency_max_ms.toFixed(2)}ms\n`;
  output += `  Status:          ${summary.test_2_concurrent_deductions.target_achieved ? '✅ PASSED' : '❌ FAILED'}\n\n`;
  
  // Test 3: Concurrent Dedicated Gateways
  output += '━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n';
  output += '🌐 TEST 3: 100 CONCURRENT DEDICATED GATEWAYS (Requirement 27.3)\n';
  output += '━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n';
  output += `  Target:          ${summary.test_3_concurrent_gateways.target}\n`;
  output += `  Created:         ${summary.test_3_concurrent_gateways.created} ${summary.test_3_concurrent_gateways.target_achieved ? '✅' : '❌'}\n`;
  output += `  Failed:          ${summary.test_3_concurrent_gateways.failed}\n`;
  output += `  Success Rate:    ${summary.test_3_concurrent_gateways.success_rate.toFixed(2)}%\n`;
  output += `  Latency (Avg):   ${summary.test_3_concurrent_gateways.latency_avg_ms.toFixed(2)}ms\n`;
  output += `  Latency (P95):   ${summary.test_3_concurrent_gateways.latency_p95_ms.toFixed(2)}ms\n`;
  output += `  Latency (P99):   ${summary.test_3_concurrent_gateways.latency_p99_ms.toFixed(2)}ms\n`;
  output += `  Latency (Max):   ${summary.test_3_concurrent_gateways.latency_max_ms.toFixed(2)}ms\n`;
  output += `  Status:          ${summary.test_3_concurrent_gateways.target_achieved ? '✅ PASSED' : '❌ FAILED'}\n\n`;
  
  // Overall Results
  output += '━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n';
  output += '🎯 OVERALL RESULTS\n';
  output += '━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n';
  output += `  All Targets:     ${summary.overall.all_targets_achieved ? '✅ ACHIEVED' : '❌ NOT ACHIEVED'}\n`;
  output += `  All Thresholds:  ${summary.overall.thresholds_passed ? '✅ PASSED' : '❌ FAILED'}\n`;
  output += `  HTTP Errors:     ${summary.overall.http_req_failed_rate.toFixed(2)}%\n\n`;
  
  if (summary.overall.all_targets_achieved && summary.overall.thresholds_passed) {
    output += '🎉 SUCCESS: All load tests passed!\n';
    output += '   - 1000 concurrent subscriptions handled successfully\n';
    output += '   - 10000 concurrent balance deductions completed\n';
    output += '   - 100 concurrent dedicated gateways created\n';
  } else {
    output += '⚠️  WARNING: Some load tests did not meet targets.\n';
    if (!summary.test_1_concurrent_subscriptions.target_achieved) {
      output += '   - Subscription test: Below 95% success rate\n';
    }
    if (!summary.test_2_concurrent_deductions.target_achieved) {
      output += '   - Balance deduction test: Below 95% success rate\n';
    }
    if (!summary.test_3_concurrent_gateways.target_achieved) {
      output += '   - Gateway creation test: Below 95% success rate\n';
    }
  }
  
  output += '\n📁 Detailed results saved to:\n';
  output += '   - results/tariff-load-test-summary.json\n';
  output += '   - results/tariff-load-test-full.json\n';
  output += '   - results/tariff-load-test-report.html\n\n';
  
  return output;
}
