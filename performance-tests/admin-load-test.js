import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const apiResponseTime = new Trend('api_response_time');

// Test configuration
export const options = {
  stages: [
    { duration: '2m', target: 20 },   // Ramp up to 20 users
    { duration: '5m', target: 100 },  // Ramp up to 100 users
    { duration: '10m', target: 100 }, // Stay at 100 users
    { duration: '2m', target: 0 },    // Ramp down to 0 users
  ],
  thresholds: {
    http_req_duration: ['p(95)<200'], // 95% of requests should be below 200ms
    http_req_failed: ['rate<0.01'],   // Error rate should be below 1%
    errors: ['rate<0.01'],
  },
};

// Base URL - should be configured via environment variable
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const API_KEY = __ENV.API_KEY || 'test-api-key';

// Test data
const testTenantIDs = [
  'tenant-1', 'tenant-2', 'tenant-3', 'tenant-4', 'tenant-5',
  'tenant-6', 'tenant-7', 'tenant-8', 'tenant-9', 'tenant-10'
];

export default function () {
  const tenantID = testTenantIDs[Math.floor(Math.random() * testTenantIDs.length)];
  
  // Test 1: Get Dashboard Overview
  testDashboardOverview();
  
  // Test 2: Get Company Ratings
  testCompanyRatings();
  
  // Test 3: Get Company Details
  testCompanyDetails(tenantID);
  
  // Test 4: Get Server Health
  testServerHealth();
  
  // Test 5: Get Alerts
  testAlerts();
  
  // Test 6: Get Distribution Config
  testDistributionConfig(tenantID);
  
  // Test 7: Get Signal/Push Stats
  testSignalPushStats(tenantID);
  
  sleep(1); // Think time between iterations
}

function testDashboardOverview() {
  const res = http.get(`${BASE_URL}/api/admin/dashboard/overview`, {
    headers: {
      'Authorization': `Bearer ${API_KEY}`,
      'Content-Type': 'application/json',
    },
  });
  
  const success = check(res, {
    'dashboard overview status is 200': (r) => r.status === 200,
    'dashboard overview has data': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.total_companies !== undefined;
      } catch (e) {
        return false;
      }
    },
  });
  
  errorRate.add(!success);
  apiResponseTime.add(res.timings.duration);
}

function testCompanyRatings() {
  const res = http.get(`${BASE_URL}/api/admin/companies/ratings`, {
    headers: {
      'Authorization': `Bearer ${API_KEY}`,
      'Content-Type': 'application/json',
    },
  });
  
  const success = check(res, {
    'company ratings status is 200': (r) => r.status === 200,
    'company ratings is array': (r) => {
      try {
        const body = JSON.parse(r.body);
        return Array.isArray(body);
      } catch (e) {
        return false;
      }
    },
  });
  
  errorRate.add(!success);
  apiResponseTime.add(res.timings.duration);
}

function testCompanyDetails(tenantID) {
  const res = http.get(`${BASE_URL}/api/admin/companies/${tenantID}/rating`, {
    headers: {
      'Authorization': `Bearer ${API_KEY}`,
      'Content-Type': 'application/json',
    },
  });
  
  const success = check(res, {
    'company details status is 200': (r) => r.status === 200,
    'company details has rating': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.overall_rating !== undefined;
      } catch (e) {
        return false;
      }
    },
  });
  
  errorRate.add(!success);
  apiResponseTime.add(res.timings.duration);
}

function testServerHealth() {
  const res = http.get(`${BASE_URL}/api/admin/servers`, {
    headers: {
      'Authorization': `Bearer ${API_KEY}`,
      'Content-Type': 'application/json',
    },
  });
  
  const success = check(res, {
    'server health status is 200': (r) => r.status === 200,
    'server health is array': (r) => {
      try {
        const body = JSON.parse(r.body);
        return Array.isArray(body);
      } catch (e) {
        return false;
      }
    },
  });
  
  errorRate.add(!success);
  apiResponseTime.add(res.timings.duration);
}

function testAlerts() {
  const res = http.get(`${BASE_URL}/api/admin/alerts?status=active`, {
    headers: {
      'Authorization': `Bearer ${API_KEY}`,
      'Content-Type': 'application/json',
    },
  });
  
  const success = check(res, {
    'alerts status is 200': (r) => r.status === 200,
    'alerts is array': (r) => {
      try {
        const body = JSON.parse(r.body);
        return Array.isArray(body);
      } catch (e) {
        return false;
      }
    },
  });
  
  errorRate.add(!success);
  apiResponseTime.add(res.timings.duration);
}

function testDistributionConfig(tenantID) {
  const res = http.get(`${BASE_URL}/api/admin/distribution/${tenantID}`, {
    headers: {
      'Authorization': `Bearer ${API_KEY}`,
      'Content-Type': 'application/json',
    },
  });
  
  const success = check(res, {
    'distribution config status is 200': (r) => r.status === 200,
    'distribution config has strategy': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.strategy_type !== undefined;
      } catch (e) {
        return false;
      }
    },
  });
  
  errorRate.add(!success);
  apiResponseTime.add(res.timings.duration);
}

function testSignalPushStats(tenantID) {
  const res = http.get(`${BASE_URL}/api/admin/signalpush/${tenantID}/statistics`, {
    headers: {
      'Authorization': `Bearer ${API_KEY}`,
      'Content-Type': 'application/json',
    },
  });
  
  const success = check(res, {
    'signal/push stats status is 200': (r) => r.status === 200,
    'signal/push stats has data': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.current_mpm !== undefined;
      } catch (e) {
        return false;
      }
    },
  });
  
  errorRate.add(!success);
  apiResponseTime.add(res.timings.duration);
}

// Setup function - runs once before the test
export function setup() {
  console.log('Starting Admin Load Test');
  console.log(`Base URL: ${BASE_URL}`);
  console.log(`Target: 100 concurrent users`);
  console.log(`Duration: 19 minutes`);
}

// Teardown function - runs once after the test
export function teardown(data) {
  console.log('Admin Load Test completed');
}
