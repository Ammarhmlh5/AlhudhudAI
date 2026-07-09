# Database Schema Reference

## Overview

This document describes the ALHudhudAI database schema, including tables, relationships, constraints, and Row Level Security (RLS) policies.

## Core Tables

### tenants

Multi-tenant isolation table.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Tenant identifier |
| name | VARCHAR(255) | NOT NULL | Company/organization name |
| email | VARCHAR(255) | UNIQUE, NOT NULL | Contact email |
| phone | VARCHAR(50) | | Phone number |
| balance | DECIMAL(12,4) | DEFAULT 0 | Account balance |
| status | VARCHAR(50) | DEFAULT 'active' | Account status |
| created_at | TIMESTAMP | DEFAULT NOW() | Creation timestamp |
| updated_at | TIMESTAMP | | Last update timestamp |

### users

User accounts with tenant isolation.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | User identifier |
| tenant_id | UUID | REFERENCES tenants(id), NOT NULL | Tenant reference |
| name | VARCHAR(255) | NOT NULL | User's full name |
| email | VARCHAR(255) | UNIQUE, NOT NULL | User's email |
| phone | VARCHAR(50) | | Phone number |
| password_hash | VARCHAR(255) | NOT NULL | Bcrypt hashed password |
| role | VARCHAR(50) | DEFAULT 'user' | User role |
| status | VARCHAR(50) | DEFAULT 'active' | Account status |
| last_login | TIMESTAMP | | Last login time |
| created_at | TIMESTAMP | DEFAULT NOW() | Creation timestamp |
| updated_at | TIMESTAMP | | Last update timestamp |

### channels

Communication channels for sending messages.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Channel identifier |
| tenant_id | UUID | REFERENCES tenants(id), NOT NULL | Tenant reference |
| name | VARCHAR(255) | NOT NULL | Channel name |
| type | VARCHAR(50) | NOT NULL | Channel type (whatsapp/sms/email) |
| config | JSONB | | Channel configuration |
| status | VARCHAR(50) | DEFAULT 'active' | Channel status |
| created_at | TIMESTAMP | DEFAULT NOW() | Creation timestamp |
| updated_at | TIMESTAMP | | Last update timestamp |

### devices

Mobile devices registered for push notifications.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Device identifier |
| tenant_id | UUID | REFERENCES tenants(id), NOT NULL | Tenant reference |
| user_id | UUID | REFERENCES users(id) | User reference |
| device_token | VARCHAR(500) | NOT NULL | FCM device token |
| platform | VARCHAR(20) | NOT NULL | ios/android |
| status | VARCHAR(50) | DEFAULT 'active' | Device status |
| created_at | TIMESTAMP | DEFAULT NOW() | Creation timestamp |
| updated_at | TIMESTAMP | | Last update timestamp |

### messages

Message records with delivery tracking.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Message identifier |
| tenant_id | UUID | REFERENCES tenants(id), NOT NULL | Tenant reference |
| channel_id | UUID | REFERENCES channels(id), NOT NULL | Channel reference |
| device_id | UUID | REFERENCES devices(id), NOT NULL | Device reference |
| phone_number | VARCHAR(50) | NOT NULL | Recipient phone |
| content | TEXT | | Message content |
| status | VARCHAR(50) | DEFAULT 'pending' | Message status |
| cost | DECIMAL(12,4) | | Message cost |
| sent_at | TIMESTAMP | | Send timestamp |
| delivered_at | TIMESTAMP | | Delivery timestamp |
| created_at | TIMESTAMP | DEFAULT NOW() | Creation timestamp |
| updated_at | TIMESTAMP | | Last update timestamp |

## Financial Tables

### accounting_transactions

Financial transaction records.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Transaction identifier |
| tenant_id | UUID | REFERENCES tenants(id), NOT NULL | Tenant reference |
| type | VARCHAR(50) | NOT NULL | Transaction type |
| amount | DECIMAL(12,4) | NOT NULL, CHECK (amount > 0) | Transaction amount |
| balance_before | DECIMAL(12,4) | | Balance before transaction |
| balance_after | DECIMAL(12,4) | | Balance after transaction |
| description | TEXT | | Transaction description |
| reference_id | UUID | | Related record ID |
| status | VARCHAR(50) | NOT NULL | Transaction status |
| created_at | TIMESTAMP | DEFAULT NOW() | Creation timestamp |

### balance_history

Balance change history.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | History record ID |
| tenant_id | UUID | REFERENCES tenants(id), NOT NULL | Tenant reference |
| type | VARCHAR(50) | NOT NULL | Change type |
| amount | DECIMAL(12,4) | NOT NULL, CHECK (amount != 0) | Change amount |
| balance_before | DECIMAL(12,4) | | Previous balance |
| balance_after | DECIMAL(12,4) | | New balance |
| reason | TEXT | | Change reason |
| created_at | TIMESTAMP | DEFAULT NOW() | Creation timestamp |

## RBAC Tables

### roles

User roles for access control.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Role identifier |
| name | VARCHAR(100) | UNIQUE, NOT NULL | Role name |
| description | TEXT | | Role description |
| created_at | TIMESTAMP | DEFAULT NOW() | Creation timestamp |

### permissions

Available permissions.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Permission identifier |
| name | VARCHAR(100) | UNIQUE, NOT NULL | Permission name |
| resource | VARCHAR(100) | NOT NULL | Resource type |
| action | VARCHAR(50) | NOT NULL | Allowed action |
| description | TEXT | | Permission description |

### role_permissions

Role-permission mappings.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Mapping ID |
| role_id | UUID | REFERENCES roles(id), NOT NULL | Role reference |
| permission_id | UUID | REFERENCES permissions(id), NOT NULL | Permission reference |
| created_at | TIMESTAMP | DEFAULT NOW() | Creation timestamp |

**Unique constraint:** (role_id, permission_id)

### user_roles

User-role assignments.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Assignment ID |
| user_id | UUID | REFERENCES users(id), NOT NULL | User reference |
| role_id | UUID | REFERENCES roles(id), NOT NULL | Role reference |
| tenant_id | UUID | REFERENCES tenants(id), NOT NULL | Tenant context |
| created_at | TIMESTAMP | DEFAULT NOW() | Creation timestamp |

**Unique constraint:** (user_id, role_id, tenant_id)

## Audit Tables

### audit_trail

System audit log.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Audit record ID |
| tenant_id | UUID | REFERENCES tenants(id), NOT NULL | Tenant reference |
| user_id | UUID | REFERENCES users(id) | User reference |
| action | VARCHAR(100) | NOT NULL | Action performed |
| resource | VARCHAR(100) | NOT NULL | Resource affected |
| resource_id | UUID | | Resource identifier |
| details | JSONB | | Additional details |
| ip_address | VARCHAR(50) | | Client IP |
| created_at | TIMESTAMP | DEFAULT NOW() | Creation timestamp |

## Row Level Security (RLS)

All multi-tenant tables have RLS enabled with the following policies:

### messages
- Users can only access messages belonging to their tenant
- Admins can access all tenant messages

### balance_history
- Users can only view their tenant's balance history
- Admins can view all tenant history

### devices
- Users can only access devices registered to their tenant
- Device tokens are tenant-isolated

### channels
- Users can only access channels belonging to their tenant
- Channel configuration is tenant-specific

### users
- Users can only view users within their tenant
- Cross-tenant user access is blocked

### audit_trail
- Users can only view audit records for their tenant
- Audit data is tenant-isolated

## Indexes

Key indexes for query performance:

| Table | Index | Columns | Purpose |
|-------|-------|---------|---------|
| messages | idx_messages_tenant_status_created | tenant_id, status, created_at | Message queries |
| balance_history | idx_balance_tenant_type_created | tenant_id, type, created_at | Balance history |
| accounting_transactions | idx_transactions_tenant_status | tenant_id, status | Transaction queries |
| subscriptions | idx_subscriptions_tenant_expires | tenant_id, expires_at | Subscription queries |
| users | idx_users_tenant_email | tenant_id, email | User lookup |
| devices | idx_devices_tenant_user | tenant_id, user_id | Device queries |

## Constraints Summary

### NOT NULL Constraints
- `messages.channel_id`
- `messages.device_id`
- `dispatch_channels.name`
- `dispatch_channels.type`
- `accounting_transactions.type`
- `accounting_transactions.status`

### CHECK Constraints
- `messages.cost >= 0`
- `balance_history.amount != 0`
- `accounting_transactions.amount > 0`

### Unique Constraints
- `users.email` (per tenant)
- `channels.name` (per tenant)
- `role_permissions(role_id, permission_id)`
- `user_roles(user_id, role_id, tenant_id)`