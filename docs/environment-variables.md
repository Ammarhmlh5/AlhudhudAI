# Environment Variables Reference

This document describes all environment variables used in ALHudhudAI.

## Backend Variables

### Database Configuration

| Variable | Required | Description | Example |
|----------|----------|-------------|---------|
| `DB_HOST` | Yes | PostgreSQL host | `localhost` |
| `DB_PORT` | Yes | PostgreSQL port | `5432` |
| `DB_NAME` | Yes | Database name | `alhudhud` |
| `DB_USER` | Yes | Database username | `alhudhud_user` |
| `DB_PASSWORD` | Yes | Database password | `${DB_PASSWORD}` |
| `DB_SSLMODE` | Yes | SSL mode (require/verify-full) | `require` |
| `DB_SSLCA` | No | Path to CA certificate | `/path/to/ca.crt` |

### Redis Configuration

| Variable | Required | Description | Example |
|----------|----------|-------------|---------|
| `REDIS_HOST` | Yes | Redis host | `localhost` |
| `REDIS_PORT` | Yes | Redis port | `6379` |
| `REDIS_PASSWORD` | Yes | Redis password | `${REDIS_PASSWORD}` |
| `REDIS_DB` | No | Redis database number | `0` |

### Application Configuration

| Variable | Required | Description | Example |
|----------|----------|-------------|---------|
| `APP_ENV` | Yes | Environment (development/staging/production) | `production` |
| `APP_PORT` | Yes | Application port | `8080` |
| `JWT_SECRET` | Yes | JWT signing secret | `${JWT_SECRET}` |
| `JWT_EXPIRY` | No | JWT expiry time | `24h` |
| `ADMIN_PASSWORD` | Yes | Admin panel password | `${ADMIN_PASSWORD}` |

### Rate Limiting

| Variable | Required | Description | Default |
|----------|----------|-------------|---------|
| `RATE_LIMIT_REQUESTS` | No | Requests per minute | `100` |
| `RATE_LIMIT_WINDOW` | No | Time window | `1m` |

### Logging

| Variable | Required | Description | Default |
|----------|----------|-------------|---------|
| `LOG_LEVEL` | No | Log level (debug/info/warn/error) | `info` |
| `LOG_FORMAT` | No | Log format (json/console) | `json` |

## Frontend Variables (Admin Web & Client Web)

| Variable | Required | Description | Example |
|----------|----------|-------------|---------|
| `VITE_API_BASE_URL` | Yes | Backend API URL | `http://localhost:8080` |
| `VITE_APP_NAME` | No | Application name | `ALHudhud` |

## Mobile Gateway Variables

| Variable | Required | Description | Example |
|----------|----------|-------------|---------|
| `API_URL` | Yes | Backend API URL | `http://localhost:8080` |
| `WS_URL` | Yes | WebSocket URL | `ws://localhost:8080` |

## Security Considerations

1. **Never commit `.env` files** - Use `.env.example` with placeholders only
2. **Use strong passwords** - Minimum 16 characters with mixed case, numbers, and symbols
3. **Rotate secrets regularly** - Change JWT secrets and passwords periodically
4. **Use different secrets** - Don't use the same password for multiple services
5. **Enable SSL/TLS** - Always use `DB_SSLMODE=require` in production

## Example Configuration

```bash
# .env.example (safe to commit)
DB_HOST=localhost
DB_PORT=5432
DB_NAME=alhudhud
DB_USER=alhudhud_user
DB_PASSWORD=${DB_PASSWORD}
DB_SSLMODE=require

REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=${REDIS_PASSWORD}
REDIS_DB=0

APP_ENV=production
APP_PORT=8080
JWT_SECRET=${JWT_SECRET}
ADMIN_PASSWORD=${ADMIN_PASSWORD}

VITE_API_BASE_URL=http://localhost:8080
```