# Staging Deployment Guide - System Critical Bugfixes

**Date**: 2025-01-XX  
**Task**: 31.2 Staging deployment  
**Status**: 📋 **DEPLOYMENT GUIDE**

---

## Overview

This guide provides step-by-step instructions for deploying the system critical bugfixes to the staging environment. The staging deployment serves as the final validation before production rollout.

**Deployment Scope**: 29 bug fixes across 6 phases
**Expected Duration**: 2-4 hours (deployment + initial verification)
**Monitoring Period**: 48 hours

---

## Pre-Deployment Verification

### ✅ Prerequisites Checklist

Before starting staging deployment, verify:

- [x] Pre-deployment checklist complete (Task 31.1)
- [x] All 29 bugs fixed and verified
- [x] All 215+ tests passing
- [x] No regressions detected
- [x] Documentation updated
- [x] Rollback plan prepared

---

## Staging Environment Configuration

### 1. Environment Variables

Create `.env.staging` file with the following configuration:

```bash
# Database Configuration
DB_HOST=staging-db.example.com
DB_PORT=5432
DB_USER=staging_user
DB_PASSWORD=<STAGING_DB_PASSWORD>
DB_NAME=alhudhud_staging
DB_SSLMODE=require

# Redis Configuration
REDIS_HOST=staging-redis.example.com
REDIS_PORT=6379
REDIS_PASSWORD=<STAGING_REDIS_PASSWORD>
REDIS_DB=0

# JWT Configuration
JWT_SECRET=<STAGING_JWT_SECRET_MIN_32_CHARS>
JWT_EXPIRATION=24h

# Server Configuration
PORT=8081
GIN_MODE=release
ENVIRONMENT=staging

# CORS Configuration
CORS_ALLOWED_ORIGINS=https://staging-client.example.com,https://staging-admin.example.com

# Alert Email Configuration (Optional)
ALERT_EMAIL_ENABLED=true
ALERT_EMAIL_SMTP_HOST=smtp.gmail.com
ALERT_EMAIL_SMTP_PORT=587
ALERT_EMAIL_USERNAME=alerts-staging@example.com
ALERT_EMAIL_PASSWORD=<STAGING_EMAIL_PASSWORD>
ALERT_EMAIL_FROM=alerts-staging@example.com
ALERT_EMAIL_TO=devops@example.com,qa@example.com
ALERT_EMAIL_MIN_LEVEL=WARNING

# Alert Webhook Configuration (Optional)
ALERT_WEBHOOK_ENABLED=true
ALERT_WEBHOOK_URL=https://hooks.slack.com/services/YOUR/STAGING/WEBHOOK
ALERT_WEBHOOK_MIN_LEVEL=INFO

# Worker Configuration
WORKER_BATCH_SIZE=50
WORKER_COUNT=20
WORKER_TICKER_MS=100

# Frontend API URL
VITE_API_BASE_URL=https://staging-api.example.com
```

### 2. Database Migration

**Run database migrations**:

```bash
cd services/backend
go run main.go migrate
```

**Verify migrations**:
- Check that migration 077 (custom IDs) is applied
- Verify custom_id backfill completed
- Verify all indexes created

### 3. Redis Setup

**Verify Redis connectivity**:

```bash
redis-cli -h staging-redis.example.com -p 6379 -a <PASSWORD> ping
```

**Expected response**: `PONG`

**Clear staging Redis cache** (optional, for clean start):

```bash
redis-cli -h staging-redis.example.com -p 6379 -a <PASSWORD> FLUSHDB
```

---

## Deployment Steps

### Step 1: Backend Deployment

**1.1 Build Backend**:

```bash
cd services/backend
go build -o alhudhud-backend main.go
```

**1.2 Deploy to Staging Server**:

```bash
# Copy binary to staging server
scp alhudhud-backend user@staging-server:/opt/alhudhud/

# Copy .env.staging
scp .env.staging user@staging-server:/opt/alhudhud/.env

# SSH to staging server
ssh user@staging-server

# Stop existing backend (if running)
sudo systemctl stop alhudhud-backend

# Replace binary
sudo mv /opt/alhudhud/alhudhud-backend /opt/alhudhud/alhudhud-backend.bak
sudo mv alhudhud-backend /opt/alhudhud/
sudo chmod +x /opt/alhudhud/alhudhud-backend

# Start backend
sudo systemctl start alhudhud-backend

# Check status
sudo systemctl status alhudhud-backend
```

**1.3 Verify Backend Startup**:

```bash
# Check logs
sudo journalctl -u alhudhud-backend -f

# Expected log messages:
# - "Configuration loaded successfully"
# - "System alerting initialized successfully"
# - "Alert manager started with 5 workers"
# - "Fast Message Processor started successfully"
# - "Server listening on port 8081"
```

**1.4 Health Check**:

```bash
curl https://staging-api.example.com/api/v1/health
```

**Expected response**:
```json
{
  "status": "healthy",
  "timestamp": "2025-01-XX...",
  "version": "1.0.0"
}
```

---

### Step 2: Frontend Deployment (Client Web)

**2.1 Build Client Web**:

```bash
cd apps/client-web

# Install dependencies (if needed)
npm install

# Build for staging
npm run build -- --mode staging
```

**2.2 Deploy to Staging Server**:

```bash
# Copy dist folder to staging server
scp -r dist/* user@staging-server:/var/www/staging-client/

# Or use deployment tool (e.g., Netlify, Vercel)
# netlify deploy --prod --dir=dist
```

**2.3 Verify Client Web**:

```bash
# Open in browser
https://staging-client.example.com

# Verify:
# - Page loads successfully
# - API connection works
# - WebSocket connection established
# - Login functionality works
```

---

### Step 3: Frontend Deployment (Admin Web)

**3.1 Build Admin Web**:

```bash
cd apps/admin-web

# Install dependencies (if needed)
npm install

# Build for staging
npm run build -- --mode staging
```

**3.2 Deploy to Staging Server**:

```bash
# Copy dist folder to staging server
scp -r dist/* user@staging-server:/var/www/staging-admin/

# Or use deployment tool
# netlify deploy --prod --dir=dist
```

**3.3 Verify Admin Web**:

```bash
# Open in browser
https://staging-admin.example.com

# Verify:
# - Page loads successfully
# - API connection works
# - Admin login works
# - Dashboard displays correctly
```

---

## Post-Deployment Verification

### Immediate Verification (0-2 hours)

#### 1. System Health Checks

**Backend Health**:
```bash
curl https://staging-api.example.com/api/v1/health
```

**Database Connectivity**:
```bash
# Check database connection in logs
sudo journalctl -u alhudhud-backend | grep "Database connected"
```

**Redis Connectivity**:
```bash
# Check Redis connection in logs
sudo journalctl -u alhudhud-backend | grep "Redis connected"
```

#### 2. Bug Fix Verification

**Critical Bugs (Phase 1)**:
- [ ] Bug 1: Verify active tenants list populated
- [ ] Bug 2: Verify PgxPool initialized (check logs)
- [ ] Bug 3: Verify real channels selected (not mock)
- [ ] Bug 4: Verify analytics show accurate message counts

**High Priority Bugs (Phase 2)**:
- [ ] Bug 5: Verify errors are logged (not ignored)
- [ ] Bug 6: Verify transaction management working
- [ ] Bug 7: Verify balance verification prevents negative balances
- [ ] Bug 8: Verify no wsHub crashes
- [ ] Bug 9: Verify Redis retry logic working
- [ ] Bug 10: Verify priority queue ordering

**Medium Priority Bugs (Phase 3)**:
- [ ] Bug 11: Verify custom IDs present in database
- [ ] Bug 12: Verify SCAN operations efficient
- [ ] Bug 13: Verify consistent cache TTL (1 minute)
- [ ] Bug 15: Verify auto-logout on 401

**Security Bugs (Phase 4)**:
- [ ] Bug 16: Verify no hardcoded credentials
- [ ] Bug 17: Verify CSRF protection active
- [ ] Bug 18: Verify tokens in httpOnly cookies
- [ ] Bug 19: Verify parameterized queries
- [ ] Bug 20: Verify environment validation

**Performance Bugs (Phase 5)**:
- [ ] Bug 21: Verify worker configuration (50×20)
- [ ] Bug 23: Verify channel cache TTL (1 minute)
- [ ] Bug 24: Verify alert system operational
- [ ] Bug 25: Verify database indexes active

**Integration Bugs (Phase 6)**:
- [ ] Bug 26: Verify environment-based API URLs
- [ ] Bug 27: Verify WebSocket auto-reconnect
- [ ] Bug 28: Verify message pagination
- [ ] Bug 29: Verify API rate limiting

#### 3. Functional Testing

**Authentication**:
```bash
# Test login
curl -X POST https://staging-api.example.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password"}'

# Expected: 200 OK with Set-Cookie header (httpOnly)
```

**CSRF Protection**:
```bash
# Test CSRF token
curl https://staging-api.example.com/api/csrf-token

# Expected: 200 OK with CSRF token
```

**Rate Limiting**:
```bash
# Test rate limiting (make 101 requests)
for i in {1..101}; do
  curl -s -o /dev/null -w "%{http_code}\n" https://staging-api.example.com/api/v1/health
done

# Expected: First 100 return 200, 101st returns 429
```

**Message Pagination**:
```bash
# Test pagination
curl "https://staging-api.example.com/api/v1/messaging/list?page=1&page_size=50" \
  -H "Cookie: token=<TOKEN>"

# Expected: 200 OK with pagination metadata
```

#### 4. Alert System Testing

**Test Queue Health Alert**:
```bash
# Manually trigger queue backup (if possible)
# Or wait for natural queue buildup

# Check alert delivery:
# - Email: Check devops@example.com inbox
# - Webhook: Check Slack channel
```

**Test Error Rate Alert**:
```bash
# Trigger errors (if possible in staging)
# Or wait for natural errors

# Check alert delivery
```

**Test Low Balance Alert**:
```bash
# Set tenant balance below threshold
# Check alert delivery
```

---

### Extended Verification (2-48 hours)

#### 1. Monitoring Metrics

**Message Processing**:
- [ ] Throughput: > 1000 messages/second
- [ ] Latency: < 100ms average
- [ ] Error rate: < 1%
- [ ] Queue size: Stable

**API Performance**:
- [ ] Response time: < 200ms average
- [ ] Rate limit violations: < 1%
- [ ] 4xx errors: < 5%
- [ ] 5xx errors: < 1%

**Database Performance**:
- [ ] Query time: < 50ms average
- [ ] Connection pool: Stable
- [ ] Index usage: Verified
- [ ] No slow queries

**Redis Performance**:
- [ ] Cache hit rate: > 90%
- [ ] Response time: < 5ms
- [ ] Memory usage: Stable
- [ ] No connection errors

**WebSocket Stability**:
- [ ] Connection uptime: > 99%
- [ ] Reconnection time: < 5s
- [ ] Message delivery: 100%
- [ ] No connection leaks

#### 2. Load Testing

**Simulate Production Load**:

```bash
# Use load testing tool (e.g., k6, Apache Bench)
k6 run load-test.js

# Test scenarios:
# - Normal load (100 req/min)
# - Peak load (1000 req/min)
# - Sustained load (500 req/min for 1 hour)
```

**Monitor During Load Test**:
- [ ] CPU usage: < 70%
- [ ] Memory usage: < 80%
- [ ] Response times: Acceptable
- [ ] Error rates: < 1%
- [ ] No crashes or restarts

#### 3. User Acceptance Testing

**Client Web Testing**:
- [ ] User registration works
- [ ] Login/logout works
- [ ] Message sending works
- [ ] Message history displays correctly
- [ ] Pagination works
- [ ] Filters work
- [ ] Export functionality works
- [ ] WebSocket updates work
- [ ] Auto-logout on 401 works

**Admin Web Testing**:
- [ ] Admin login works
- [ ] Dashboard displays correctly
- [ ] Tenant management works
- [ ] Gateway management works
- [ ] Statistics accurate
- [ ] Reports generate correctly
- [ ] Settings update correctly

#### 4. Security Testing

**CSRF Protection**:
- [ ] POST requests without CSRF token rejected
- [ ] POST requests with valid CSRF token accepted
- [ ] CSRF token refreshes correctly

**Rate Limiting**:
- [ ] Excessive requests return 429
- [ ] Rate limit headers present
- [ ] Retry-After header correct
- [ ] Legitimate users not blocked

**Authentication**:
- [ ] Invalid tokens rejected
- [ ] Expired tokens trigger auto-logout
- [ ] httpOnly cookies prevent XSS
- [ ] Session management secure

**SQL Injection**:
- [ ] Malicious inputs sanitized
- [ ] Parameterized queries used
- [ ] No SQL errors exposed

---

## Monitoring and Alerting

### 1. Log Monitoring

**Backend Logs**:
```bash
# Real-time logs
sudo journalctl -u alhudhud-backend -f

# Error logs
sudo journalctl -u alhudhud-backend | grep ERROR

# Alert logs
sudo journalctl -u alhudhud-backend | grep "Alert sent"
```

**Key Log Messages to Monitor**:
- Configuration loaded successfully
- Alert manager started
- Queue health monitoring started
- Error rate monitoring started
- Balance monitoring started
- Any ERROR or CRITICAL messages

### 2. Alert Monitoring

**Email Alerts**:
- Check devops@example.com inbox
- Verify alert format and content
- Verify alert levels (INFO, WARNING, CRITICAL)

**Webhook Alerts**:
- Check Slack channel
- Verify alert format and content
- Verify alert metadata

**Alert Types to Monitor**:
- Queue backup alerts (> 5,000 messages)
- Error rate alerts (> 25%)
- Low balance alerts (< 100 SAR)

### 3. Metrics Dashboard

**Create Monitoring Dashboard** (e.g., Grafana):

**Panels**:
- Message processing throughput
- API response times
- Error rates by endpoint
- Queue sizes by tenant
- Database query performance
- Redis cache hit rates
- WebSocket connection count
- Rate limit violations
- Alert counts by type

---

## Issue Tracking

### Known Issues

**None identified during pre-deployment testing**

### Issue Reporting

**If issues are discovered**:

1. **Document the issue**:
   - Description
   - Steps to reproduce
   - Expected vs actual behavior
   - Logs and screenshots
   - Impact assessment

2. **Assess severity**:
   - **Critical**: System down, data loss, security breach
   - **High**: Major functionality broken, many users affected
   - **Medium**: Minor functionality broken, few users affected
   - **Low**: Cosmetic issues, no functional impact

3. **Decide on action**:
   - **Critical/High**: Consider rollback
   - **Medium**: Fix in hotfix
   - **Low**: Fix in next release

4. **Communicate**:
   - Notify stakeholders
   - Update status page
   - Provide ETA for fix

---

## Rollback Procedure

### When to Rollback

**Rollback triggers**:
- Critical bugs discovered
- Performance degradation > 50%
- Security vulnerabilities detected
- Data integrity issues
- User-facing errors > 5%
- System instability

### Rollback Steps

**1. Stop Current Deployment**:
```bash
ssh user@staging-server
sudo systemctl stop alhudhud-backend
```

**2. Restore Previous Version**:
```bash
# Restore backend binary
sudo mv /opt/alhudhud/alhudhud-backend.bak /opt/alhudhud/alhudhud-backend

# Restore previous .env (if needed)
sudo mv /opt/alhudhud/.env.bak /opt/alhudhud/.env
```

**3. Rollback Database** (if needed):
```bash
# Rollback migrations
cd services/backend
go run main.go migrate down
```

**4. Restart Backend**:
```bash
sudo systemctl start alhudhud-backend
sudo systemctl status alhudhud-backend
```

**5. Verify Rollback**:
```bash
curl https://staging-api.example.com/api/v1/health
```

**6. Rollback Frontend** (if needed):
```bash
# Restore previous frontend build
scp -r previous-dist/* user@staging-server:/var/www/staging-client/
```

---

## Success Criteria

### Staging Deployment Success

**Deployment successful if**:
- [x] All services deployed without errors
- [x] Health checks passing
- [x] All 29 bug fixes verified
- [x] No regressions detected
- [x] Performance metrics acceptable
- [x] Security features working
- [x] Alert system operational
- [x] No critical issues discovered

**Monitoring period**: 48 hours

**Approval for production**: After 48 hours of stable operation

---

## Next Steps

### After Successful Staging Deployment

1. **Monitor for 48 hours**:
   - Check logs daily
   - Review metrics
   - Respond to alerts
   - Document any issues

2. **Conduct final review**:
   - Review all metrics
   - Review all alerts
   - Review user feedback
   - Review test results

3. **Prepare for production**:
   - Update production configuration
   - Schedule production deployment
   - Notify stakeholders
   - Prepare production rollback plan

4. **Proceed to Task 31.3**: Production deployment preparation

---

## Contact Information

**DevOps Team**: devops@example.com  
**QA Team**: qa@example.com  
**On-Call Engineer**: oncall@example.com  
**Slack Channel**: #staging-deployment

---

## Appendix

### A. Useful Commands

**Check Backend Status**:
```bash
sudo systemctl status alhudhud-backend
```

**View Logs**:
```bash
sudo journalctl -u alhudhud-backend -f
```

**Check Database Connection**:
```bash
psql -h staging-db.example.com -U staging_user -d alhudhud_staging -c "SELECT 1;"
```

**Check Redis Connection**:
```bash
redis-cli -h staging-redis.example.com -p 6379 -a <PASSWORD> ping
```

**Test API Endpoint**:
```bash
curl https://staging-api.example.com/api/v1/health
```

### B. Troubleshooting

**Backend won't start**:
- Check logs: `sudo journalctl -u alhudhud-backend -n 100`
- Verify .env file exists and is readable
- Verify database connectivity
- Verify Redis connectivity
- Check port 8081 is not in use

**Database connection errors**:
- Verify DB_HOST, DB_PORT, DB_USER, DB_PASSWORD
- Check database is running
- Check firewall rules
- Verify SSL/TLS configuration

**Redis connection errors**:
- Verify REDIS_HOST, REDIS_PORT, REDIS_PASSWORD
- Check Redis is running
- Check firewall rules
- Verify Redis authentication

**Frontend not loading**:
- Check VITE_API_BASE_URL is correct
- Verify CORS configuration
- Check browser console for errors
- Verify API is accessible

**Rate limiting not working**:
- Check Redis connection
- Verify rate limiter middleware applied
- Check logs for rate limiter initialization
- Test with curl (make 101 requests)

**Alerts not sending**:
- Check ALERT_EMAIL_* or ALERT_WEBHOOK_* configuration
- Verify SMTP credentials
- Verify webhook URL
- Check logs for alert manager errors
- Test alert delivery manually

---

**Deployment Guide Version**: 1.0  
**Last Updated**: 2025-01-XX  
**Status**: Ready for staging deployment
