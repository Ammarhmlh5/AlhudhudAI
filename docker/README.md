# Docker Deployment Guide

This directory contains Docker configurations for deploying the admin system.

## Prerequisites

- Docker 20.10+
- Docker Compose 2.0+
- At least 4GB RAM
- At least 20GB disk space

## Quick Start

### 1. Environment Setup

Create a `.env` file in the project root:

```bash
# Database
DB_USER=hudhud
DB_PASSWORD=your_secure_password
DB_NAME=hudhud_sms

# Redis
REDIS_PASSWORD=your_redis_password
REDIS_PORT=6379

# JWT
JWT_SECRET=your_jwt_secret_key

# Grafana
GRAFANA_ADMIN_USER=admin
GRAFANA_ADMIN_PASSWORD=your_grafana_password
GRAFANA_ROOT_URL=http://localhost:3000

# Admin System
ADMIN_API_URL=http://localhost:8081
ADMIN_WS_URL=ws://localhost:8081

# Environment
ENVIRONMENT=production
LOG_LEVEL=info
```

### 2. Build Images

```bash
# Build all images
docker-compose build

# Build specific service
docker-compose build admin-backend
docker-compose build admin-web
```

### 3. Start Services

```bash
# Start all services
docker-compose up -d

# Start specific services
docker-compose up -d admin-backend admin-web

# View logs
docker-compose logs -f admin-backend
docker-compose logs -f admin-web
```

### 4. Verify Deployment

```bash
# Check service status
docker-compose ps

# Check admin backend health
curl http://localhost:8081/health

# Check admin web health
curl http://localhost:3001/health

# Check metrics endpoint
curl http://localhost:9091/metrics
```

## Services

### Admin Backend
- **Port**: 8081 (HTTP API)
- **Port**: 9091 (Metrics)
- **Health Check**: http://localhost:8081/health
- **Metrics**: http://localhost:9091/metrics

### Admin Web
- **Port**: 3001
- **Health Check**: http://localhost:3001/health
- **URL**: http://localhost:3001

### Database (PostgreSQL)
- **Port**: 5432
- **User**: Set in `.env`
- **Database**: Set in `.env`

### Redis
- **Port**: 6379
- **Password**: Set in `.env`

### Prometheus
- **Port**: 9090
- **URL**: http://localhost:9090

### Grafana
- **Port**: 3000
- **URL**: http://localhost:3000
- **Default Credentials**: Set in `.env`

## Database Migrations

### Run Migrations

```bash
# Run migrations on startup (automatic)
docker-compose up -d admin-backend

# Run migrations manually
docker-compose exec admin-backend /app/admin-backend migrate up

# Check migration status
docker-compose exec admin-backend /app/admin-backend migrate status
```

### Rollback Migrations

```bash
# Rollback last migration
docker-compose exec admin-backend /app/admin-backend migrate down

# Rollback to specific version
docker-compose exec admin-backend /app/admin-backend migrate goto <version>
```

## Monitoring

### Prometheus

Access Prometheus at http://localhost:9090

**Useful Queries:**
```promql
# Admin API request rate
rate(admin_api_request_total[5m])

# Admin API response time
histogram_quantile(0.95, rate(admin_api_request_duration_seconds_bucket[5m]))

# Active alerts
sum(admin_active_alerts_count)
```

### Grafana

1. Access Grafana at http://localhost:3000
2. Login with credentials from `.env`
3. Navigate to **Dashboards** → **Admin System Dashboard**

## Logs

### View Logs

```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f admin-backend
docker-compose logs -f admin-web

# Last 100 lines
docker-compose logs --tail=100 admin-backend

# Since specific time
docker-compose logs --since 2024-01-01T00:00:00 admin-backend
```

### Log Levels

Set log level in `.env`:
```bash
LOG_LEVEL=debug  # debug, info, warn, error
```

## Scaling

### Scale Services

```bash
# Scale admin backend to 3 instances
docker-compose up -d --scale admin-backend=3

# Scale admin web to 2 instances
docker-compose up -d --scale admin-web=2
```

### Load Balancing

For production, use a load balancer (nginx, HAProxy, etc.) in front of scaled services.

## Backup and Restore

### Database Backup

```bash
# Create backup
docker-compose exec db pg_dump -U $DB_USER $DB_NAME > backup_$(date +%Y%m%d_%H%M%S).sql

# Restore backup
docker-compose exec -T db psql -U $DB_USER $DB_NAME < backup.sql
```

### Redis Backup

```bash
# Create backup
docker-compose exec redis redis-cli SAVE
docker cp hudhud-redis:/data/dump.rdb ./redis_backup_$(date +%Y%m%d_%H%M%S).rdb

# Restore backup
docker cp redis_backup.rdb hudhud-redis:/data/dump.rdb
docker-compose restart redis
```

### Volume Backup

```bash
# Backup all volumes
docker run --rm \
  -v hudhud_db_data:/data \
  -v $(pwd):/backup \
  alpine tar czf /backup/db_data_backup.tar.gz -C /data .

# Restore volume
docker run --rm \
  -v hudhud_db_data:/data \
  -v $(pwd):/backup \
  alpine tar xzf /backup/db_data_backup.tar.gz -C /data
```

## Troubleshooting

### Service Won't Start

```bash
# Check logs
docker-compose logs admin-backend

# Check service status
docker-compose ps

# Restart service
docker-compose restart admin-backend
```

### Database Connection Issues

```bash
# Check database is running
docker-compose ps db

# Check database logs
docker-compose logs db

# Test connection
docker-compose exec admin-backend nc -zv db 5432
```

### High Memory Usage

```bash
# Check resource usage
docker stats

# Limit memory for service (in docker-compose.yml)
services:
  admin-backend:
    deploy:
      resources:
        limits:
          memory: 512M
```

### Disk Space Issues

```bash
# Check disk usage
docker system df

# Clean up unused resources
docker system prune -a

# Remove old images
docker image prune -a

# Remove unused volumes
docker volume prune
```

## Security

### Best Practices

1. **Use Strong Passwords**: Set strong passwords in `.env`
2. **Limit Exposed Ports**: Only expose necessary ports
3. **Use Secrets**: Use Docker secrets for sensitive data
4. **Regular Updates**: Keep images updated
5. **Network Isolation**: Use Docker networks for isolation

### Docker Secrets (Production)

```yaml
services:
  admin-backend:
    secrets:
      - db_password
      - jwt_secret

secrets:
  db_password:
    file: ./secrets/db_password.txt
  jwt_secret:
    file: ./secrets/jwt_secret.txt
```

### SSL/TLS

For production, use a reverse proxy (nginx, Traefik) with SSL/TLS:

```yaml
services:
  nginx:
    image: nginx:alpine
    ports:
      - "443:443"
    volumes:
      - ./nginx/ssl:/etc/nginx/ssl
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf
```

## Production Deployment

### Recommended Configuration

```yaml
services:
  admin-backend:
    deploy:
      replicas: 3
      resources:
        limits:
          cpus: '1'
          memory: 512M
        reservations:
          cpus: '0.5'
          memory: 256M
      restart_policy:
        condition: on-failure
        delay: 5s
        max_attempts: 3
```

### Health Checks

All services have health checks configured. Monitor them:

```bash
# Check health status
docker-compose ps

# View health check logs
docker inspect --format='{{json .State.Health}}' hudhud-admin-backend
```

### Monitoring Alerts

Configure Alertmanager for production alerts:

```yaml
alertmanager:
  image: prom/alertmanager:latest
  ports:
    - "9093:9093"
  volumes:
    - ./alertmanager.yml:/etc/alertmanager/alertmanager.yml
```

## Maintenance

### Update Services

```bash
# Pull latest images
docker-compose pull

# Rebuild and restart
docker-compose up -d --build

# Rolling update (zero downtime)
docker-compose up -d --no-deps --build admin-backend
```

### Clean Up

```bash
# Stop all services
docker-compose down

# Stop and remove volumes
docker-compose down -v

# Remove all data (WARNING: destructive)
docker-compose down -v --rmi all
```

## Additional Resources

- [Docker Documentation](https://docs.docker.com/)
- [Docker Compose Documentation](https://docs.docker.com/compose/)
- [Prometheus Documentation](https://prometheus.io/docs/)
- [Grafana Documentation](https://grafana.com/docs/)
