import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const apiResponseTime = new Trend('api_response_time');
const requestCount = new Counter('total_requests');

// Stress test configuration - gradually increase load beyond normal capacity
export const options = {
  stages: [
    { duration: '2m', target: 100 },   // Normal load
    { duration: '5m', target: 500 },   // Stress load
    { duration: '5m', target: 1000 },  // High stress
    { duration: '5m', target: 1500 },  // Peak stress
    { duration: '5m', target: 1000 },  // Scale down
    { duration: '2m', target: 0 },     // Recovery
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'], // 95% under 500ms (relaxed for stress test)
    http_req_failed: ['rate<0.05'],   // Allow up to 5% errors under stress
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const API_KEY = __ENV.API_KEY || 'test-api-key';

// Generate large dataset for stress testing
const tenantIDs = Array.from({ length: 1000 }, (_, i) => `tenant-${i + 1}`);

export default function () {
  const tenantID = tenantIDs[Math.floor(Math.random() * tenantIDs.length)];
  
  // Simulate heavy admin operations
  stressTestRatingCalculation(tenantID);
  stressTestMetricsAggregation(tenantID);
  stressTestDistributionSystem(tenantID);
  stressTestAlertSystem();
  
  sleep(0.1); // Minimal think time for stress test
}

function stressTestRatingCalculation(tenantID) {
  // Test rating calculation with large dataset
  const res = http.get(`${BASE_URL}/api/admin/companies/${tenantID}/rating`, {
    headers: {
      'Authorization': `Bearer ${API_KEY}`,
      'Content-Type': 'application/json',
    },
    timeout: '10s',
  });
  
  requestCount.add(1);
  
  const success = check(res, {
    'rating calculation status is 200 or 503': (r) => r.status === 200 || r.status === 503,
    'rating calculation response time < 1s': (r) => r.timings.duration < 1000,
  });
  
  errorRate.add(!success);
  if (res.status === 200) {
    apiResponseTime.add(res.timings.duration);
  }
}

function stressTestMetricsAggregation(tenantID) {
  // Test metrics aggregation with complex queries
  const periodStart = new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString();
  const periodEnd = new Date().toISOString();
  
  const res = http.get(
    `${BASE_URL}/api/admin/companies/${tenantID}/metrics?period_type=daily&period_start=${periodStart}&period_end=${periodEnd}`,
    {
      headers: {
        'Authorization': `Bearer ${API_KEY}`,
        'Content-Type': 'application/json',
      },
      timeout: '10s',
    }
  );
  
  requestCount.add(1);
  
  const success = check(res, {
    'metrics aggregation status is 200 or 503': (r) => r.status === 200 || r.status === 503,
    'metrics aggregation response time < 2s': (r) => r.timings.duration < 2000,
  });
  
  errorRate.add(!success);
  if (res.status === 200) {
    apiResponseTime.add(res.timings.duration);
  }
}

function stressTestDistributionSystem(tenantID) {
  // Test distribution system with 10,000 messages/minute simulation
  const batch = http.batch([
    ['GET', `${BASE_URL}/api/admin/distribution/${tenantID}`, null, {
      headers: { 'Authorization': `Bearer ${API_KEY}` },
      timeout: '5s',
    }],
    ['GET', `${BASE_URL}/api/admin/distribution/${tenantID}/devices`, null, {
      headers: { 'Authorization': `Bearer ${API_KEY}` },
      timeout: '5s',
    }],
    ['GET', `${BASE_URL}/api/admin/distribution/${tenantID}/metrics`, null, {
      headers: { 'Authorization': `Bearer ${API_KEY}` },
      timeout: '5s',
    }],
  ]);
  
  requestCount.add(batch.length);
  
  batch.forEach((res) => {
    const success = check(res, {
      'distribution status is 200 or 503': (r) => r.status === 200 || r.status === 503,
    });
    
    errorRate.add(!success);
    if (res.status === 200) {
      apiResponseTime.add(res.timings.duration);
    }
  });
}

function stressTestAlertSystem() {
  // Test alert system with high volume
  const res = http.get(`${BASE_URL}/api/admin/alerts?limit=1000`, {
    headers: {
      'Authorization': `Bearer ${API_KEY}`,
      'Content-Type': 'application/json',
    },
    timeout: '10s',
  });
  
  requestCount.add(1);
  
  const success = check(res, {
    'alerts status is 200 or 503': (r) => r.status === 200 || r.status === 503,
    'alerts response time < 1s': (r) => r.timings.duration < 1000,
  });
  
  errorRate.add(!success);
  if (res.status === 200) {
    apiResponseTime.add(res.timings.duration);
  }
}

export function setup() {
  console.log('Starting Admin Stress Test');
  console.log(`Base URL: ${BASE_URL}`);
  console.log(`Peak Target: 1500 concurrent users`);
  console.log(`Test Duration: 24 minutes`);
  console.log(`Dataset: 1000 tenants`);
  console.log('');
  console.log('Stress Test Stages:');
  console.log('  1. Normal load: 100 users (2 min)');
  console.log('  2. Stress load: 500 users (5 min)');
  console.log('  3. High stress: 1000 users (5 min)');
  console.log('  4. Peak stress: 1500 users (5 min)');
  console.log('  5. Scale down: 1000 users (5 min)');
  console.log('  6. Recovery: 0 users (2 min)');
}

export function teardown(data) {
  console.log('');
  console.log('Admin Stress Test completed');
  console.log('Review metrics to identify breaking points and bottlenecks');
}
