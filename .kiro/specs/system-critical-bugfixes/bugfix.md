# Bugfix Requirements Document: System Critical Bugfixes

## Introduction

هذا المستند يحدد متطلبات إصلاح 30 مشكلة حرجة تم اكتشافها في نظام SMS Gateway. المشاكل موزعة على عدة فئات: مشاكل حرجة (4)، عالية الأولوية (6)، متوسطة الأولوية (5)، أمنية (5)، أداء (5)، وتكامل (5). الهدف هو إصلاح جميع المشاكل التي تمنع النظام من العمل بشكل صحيح، مع التركيز على الأداء العالي والأمان والاستقرار.

النطاق: إصلاح الأخطاء الموجودة فقط - لا ميزات جديدة.

## Bug Analysis

### Current Behavior (Defect)

#### 🔴 المشاكل الحرجة

1.1 WHEN `getActiveTenants()` is called in `fast_processor.go` THEN the system returns an empty list always, causing message processing to stop completely

1.2 WHEN `db.PgxPool` is accessed in `main.go` or `fast_processor.go` THEN the system crashes with nil pointer error because PgxPool is not initialized in `db/gorm.go`

1.3 WHEN `getOrSelectChannel()` is called in `fast_processor.go` (lines 248-262) THEN the system returns mock/dummy data instead of selecting real channels from the database

1.4 WHEN `GetCurrentLoad()` is called in `redis/analytics.go` THEN the system does not calculate actual message counts, returning incomplete analytics data

#### 🟠 المشاكل عالية الأولوية

1.5 WHEN errors occur in various functions THEN the system silently ignores them using `_ = ` pattern, hiding critical failures

1.6 WHEN database transactions are used with `defer tx.Rollback()` and explicit `tx.Commit()` THEN the system may execute rollback after commit, causing transaction management issues

1.7 WHEN balance deduction occurs in `processBatchWithTransaction()` THEN the system does not verify sufficient balance before deduction, potentially causing negative balances

1.8 WHEN `fp.wsHub.SendMessage()` is called in `fast_processor.go` THEN the system may crash with nil pointer if wsHub is not initialized

1.9 WHEN Redis operations fail THEN the system does not retry, causing data loss or inconsistent state

1.10 WHEN messages are dequeued from priority queues in `message_queue.go` THEN the system does not properly enforce priority ordering across all three queues

#### 🟡 المشاكل متوسطة الأولوية

1.11 WHEN migration `077_add_custom_ids.sql` runs THEN the system does not backfill existing records with custom_id values, leaving NULL values in production data

1.12 WHEN `InvalidateAll()` is called in `pricing_cache.go` THEN the system uses SCAN without COUNT parameter, potentially causing performance issues

1.13 WHEN caching balance and usage data THEN the system uses inconsistent TTL values (Balance: 1 minute, Usage: 30 seconds), causing cache coherence issues

1.14 WHEN analytics are calculated in `redis/analytics.go` THEN the system does not count actual messages, providing incomplete metrics

1.15 WHEN 401 Unauthorized response is received in `axios.js` THEN the system does not automatically log out the user, leaving expired sessions active

#### 🔵 المشاكل الأمنية

1.16 WHEN deployment scripts run THEN the system exposes database credentials in plain text in script files

1.17 WHEN API requests are made with `withCredentials: true` in `axios.js` THEN the system has no CSRF token protection, vulnerable to CSRF attacks

1.18 WHEN authentication tokens are stored THEN the system stores them in localStorage, making them vulnerable to XSS attacks

1.19 WHEN dynamic SQL queries are constructed THEN the system may be vulnerable to SQL injection if parameterization is not used consistently

1.20 WHEN the application starts THEN the system does not validate required environment variables, potentially running with missing configuration

#### 📊 مشاكل الأداء

1.21 WHEN batch processing runs with 100 messages × 10 workers THEN the system processes 1000 messages per cycle, which may not be optimized for high load

1.22 WHEN Redis cache is cleared using SCAN THEN the system does not use COUNT parameter for pagination, causing performance degradation

1.23 WHEN channel cache uses 5-minute TTL THEN the system may serve stale channel data, causing routing to offline channels

1.24 WHEN critical system issues occur THEN the system has no alerting mechanism to notify administrators

1.25 WHEN database queries filter by `event_custom_id` THEN the system performs slow queries because no index exists on this column

#### 📋 مشاكل التكامل

1.26 WHEN API endpoints are configured in `axios.js` THEN the system uses hardcoded localhost:8081 addresses instead of environment variables

1.27 WHEN WebSocket connection drops THEN the system does not automatically reconnect, requiring manual page refresh

1.28 WHEN message lists are loaded THEN the system loads all messages at once without pagination, causing memory and performance issues

1.29 WHEN admin panel makes API requests THEN the system has no rate limiting, allowing potential abuse

1.30 WHEN Mobile Gateway encounters errors THEN the system has no recovery mechanism, requiring manual intervention

### Expected Behavior (Correct)

#### 🔴 المشاكل الحرجة - الإصلاحات المتوقعة

2.1 WHEN `getActiveTenants()` is called THEN the system SHALL query Redis set `alhudhud:active:tenants` to return list of tenant UUIDs that have pending messages in queues

2.2 WHEN database connections are initialized in `db/gorm.go` THEN the system SHALL initialize both `GormDB` and `PgxPool` connections using the same DSN configuration

2.3 WHEN `getOrSelectChannel()` is called THEN the system SHALL query the database for available online channels, check cache first, and select based on priority and load

2.4 WHEN `GetCurrentLoad()` is called THEN the system SHALL scan Redis keys for event counters and message counts, aggregate them, and return accurate load metrics

#### 🟠 المشاكل عالية الأولوية - الإصلاحات المتوقعة

2.5 WHEN errors occur in critical operations THEN the system SHALL log errors properly and handle them appropriately (return, retry, or fail gracefully)

2.6 WHEN database transactions are used THEN the system SHALL use `defer tx.Rollback(ctx)` pattern and check commit errors, ensuring rollback only executes if commit fails

2.7 WHEN balance deduction is attempted THEN the system SHALL verify sufficient balance exists before deduction, returning error if insufficient

2.8 WHEN `fp.wsHub.SendMessage()` is called THEN the system SHALL check if wsHub is not nil before calling methods on it

2.9 WHEN Redis operations fail THEN the system SHALL implement retry logic with exponential backoff for transient failures

2.10 WHEN messages are dequeued from priority queues THEN the system SHALL properly drain high-priority queue completely before moving to normal priority, then low priority

#### 🟡 المشاكل متوسطة الأولوية - الإصلاحات المتوقعة

2.11 WHEN migration `077_add_custom_ids.sql` runs THEN the system SHALL include UPDATE statements to backfill custom_id for existing records using a generation function

2.12 WHEN `InvalidateAll()` is called THEN the system SHALL use SCAN with COUNT parameter (e.g., 1000) for efficient pagination

2.13 WHEN caching balance and usage data THEN the system SHALL use consistent TTL values (e.g., both 1 minute) to maintain cache coherence

2.14 WHEN analytics are calculated THEN the system SHALL query actual message counts from Redis or database and aggregate them correctly

2.15 WHEN 401 Unauthorized response is received THEN the system SHALL clear localStorage token and redirect to login page automatically

#### 🔵 المشاكل الأمنية - الإصلاحات المتوقعة

2.16 WHEN deployment scripts need credentials THEN the system SHALL read credentials from environment variables or secure vaults, not hardcoded values

2.17 WHEN API requests are made with credentials THEN the system SHALL implement CSRF token validation using double-submit cookie or synchronizer token pattern

2.18 WHEN authentication tokens are stored THEN the system SHALL use httpOnly secure cookies instead of localStorage to prevent XSS attacks

2.19 WHEN SQL queries are constructed THEN the system SHALL use parameterized queries exclusively, validating all dynamic query construction

2.20 WHEN the application starts THEN the system SHALL validate all required environment variables (DB_HOST, DB_PORT, JWT_SECRET, etc.) and fail fast with clear error messages if missing

#### 📊 مشاكل الأداء - الإصلاحات المتوقعة

2.21 WHEN batch processing configuration is set THEN the system SHALL use optimized batch size based on load testing (e.g., 50 messages × 20 workers for better parallelism)

2.22 WHEN Redis cache is cleared THEN the system SHALL use SCAN with COUNT=1000 and process in batches to avoid blocking Redis

2.23 WHEN channel cache is configured THEN the system SHALL use shorter TTL (e.g., 1 minute) to reduce stale data while maintaining performance

2.24 WHEN critical issues occur (high error rate, queue backup, low balance) THEN the system SHALL send alerts via configured channels (email, webhook, SMS)

2.25 WHEN database schema is created THEN the system SHALL include index on `messages.event_custom_id` for fast event-based queries

#### 📋 مشاكل التكامل - الإصلاحات المتوقعة

2.26 WHEN API client is configured THEN the system SHALL read API base URL from environment variable (VITE_API_BASE_URL) instead of hardcoded values

2.27 WHEN WebSocket connection drops THEN the system SHALL implement automatic reconnection with exponential backoff (1s, 2s, 4s, 8s, max 30s)

2.28 WHEN message lists are loaded THEN the system SHALL implement pagination with configurable page size (default 50) and load-more functionality

2.29 WHEN admin panel makes API requests THEN the system SHALL implement rate limiting (e.g., 100 requests per minute per IP)

2.30 WHEN Mobile Gateway encounters errors THEN the system SHALL implement retry logic, error reporting, and graceful degradation

### Unchanged Behavior (Regression Prevention)

#### معالجة الرسائل والقوائم

3.1 WHEN messages are successfully enqueued to Redis THEN the system SHALL CONTINUE TO maintain FIFO order within each priority level

3.2 WHEN batch processing completes successfully THEN the system SHALL CONTINUE TO update message status to PROCESSING and commit transactions atomically

3.3 WHEN WebSocket messages are sent successfully THEN the system SHALL CONTINUE TO deliver them to connected devices in real-time

#### إدارة الرصيد والمعاملات

3.4 WHEN balance deduction succeeds with sufficient funds THEN the system SHALL CONTINUE TO create accounting transactions and update tenant balance correctly

3.5 WHEN refunds are processed for failed messages THEN the system SHALL CONTINUE TO restore balance and log refund transactions

#### التخزين المؤقت والأداء

3.6 WHEN pricing cache hits occur for common message lengths THEN the system SHALL CONTINUE TO return cached results without recalculation

3.7 WHEN channel cache hits occur for active tenants THEN the system SHALL CONTINUE TO return cached channel information without database query

3.8 WHEN Redis operations succeed THEN the system SHALL CONTINUE TO provide fast queue operations and caching

#### المصادقة والأمان

3.9 WHEN valid JWT tokens are provided THEN the system SHALL CONTINUE TO authenticate users and extract tenant context correctly

3.10 WHEN API keys are used for authentication THEN the system SHALL CONTINUE TO validate them and allow programmatic access

3.11 WHEN CORS requests are made from allowed origins THEN the system SHALL CONTINUE TO accept them with proper headers

#### قاعدة البيانات والاستعلامات

3.12 WHEN GORM queries are executed with proper tenant isolation THEN the system SHALL CONTINUE TO enforce RLS policies and prevent cross-tenant access

3.13 WHEN migrations run on fresh database THEN the system SHALL CONTINUE TO create all tables, indexes, and constraints correctly

3.14 WHEN database transactions commit successfully THEN the system SHALL CONTINUE TO persist all changes atomically

#### واجهات المستخدم والتكامل

3.15 WHEN users log in with correct credentials THEN the system SHALL CONTINUE TO issue tokens and establish sessions

3.16 WHEN admin panel loads dashboard data THEN the system SHALL CONTINUE TO display metrics, charts, and statistics correctly

3.17 WHEN Excel files are uploaded for bulk messaging THEN the system SHALL CONTINUE TO parse and validate them correctly

#### المراقبة والتحليلات

3.18 WHEN metrics are collected every 10 seconds THEN the system SHALL CONTINUE TO store them in Redis and database

3.19 WHEN health checks are performed THEN the system SHALL CONTINUE TO return accurate system status

3.20 WHEN audit logs are created for sensitive operations THEN the system SHALL CONTINUE TO record them with full context
