# دليل النشر - نظام إدارة الرسائل SMS

## نظرة عامة

هذا الدليل يوضح خطوات نشر نظام إدارة الرسائل SMS في بيئات مختلفة.

## متطلبات البنية التحتية

### الحد الأدنى للمتطلبات

**Server Specifications:**
- CPU: 4 cores
- RAM: 8 GB
- Storage: 100 GB SSD
- Network: 100 Mbps

**Software Requirements:**
- Docker 24.0+
- Docker Compose 2.20+
- Git 2.40+

### المتطلبات الموصى بها (Production)

**Server Specifications:**
- CPU: 8 cores
- RAM: 16 GB
- Storage: 500 GB SSD (with backup)
- Network: 1 Gbps

**Additional Services:**
- Load Balancer (Nginx/HAProxy)
- SSL Certificate (Let's Encrypt)
- Monitoring (Prometheus + Grafana)
- Backup Solution

---

## البيئات المتاحة

### 1. Development Environment

للتطوير المحلي.

```bash
# Clone repository
git clone https://github.com/your-org/sms-gateway.git
cd sms-gateway

# Copy environment file
cp .env.example .env.development

# Start services
docker-compose up -d

# Run migrations
./run-migrations.sh

# Access application
# Backend: http://localhost:8080
# Admin Web: http://localhost:3000
# Client Web: http://localhost:3001
```

---

### 2. Staging Environment

لاختبار ما قبل الإنتاج.

```bash
# Clone repository
git clone https://github.com/your-org/sms-gateway.git
cd sms-gateway

# Copy environment file
cp .env.example .env.staging

# Edit environment variables
nano .env.staging

# Start services
docker-compose -f docker-compose.yml -f docker-compose.staging.yml up -d

# Run migrations
./run-migrations.sh

# Verify deployment
curl http://staging.example.com/health
```

---

### 3. Production Environment

للإنتاج.

```bash
# Clone repository
git clone https://github.com/your-org/sms-gateway.git
cd sms-gateway

# Copy environment file
cp .env.example .env.production

# Edit environment variables (CRITICAL)
nano .env.production

# Start services
docker-compose -f docker-compose.yml -f docker-compose.production.yml up -d

# Run migrations
./run-migrations.sh

# Verify deployment
curl https://api.example.com/health
```

---

## إعداد Environment Variables

### Backend (.env)

```bash
# Application
APP_ENV=production
APP_PORT=8080
APP_DEBUG=false

# Database
DB_HOST=postgres
DB_PORT=5432
DB_NAME=sms_gateway
DB_USER=postgres
DB_PASSWORD=your_secure_password_here
DB_MAX_CONNECTIONS=100
DB_IDLE_CONNECTIONS=10

# Redis
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=your_redis_password_here
REDIS_DB=0

# JWT
JWT_SECRET=your_jwt_secret_here_min_32_chars
JWT_EXPIRY=24h

# Rate Limiting
RATE_LIMIT_PER_MINUTE=100

# Worker Pool
WORKER_COUNT=50

# Batch Processing
BATCH_SIZE=100
BATCH_INTERVAL=100ms

# Monitoring
PROMETHEUS_ENABLED=true
PROMETHEUS_PORT=9090

# Logging
LOG_LEVEL=info
LOG_FORMAT=json
```

### Frontend (Admin Web)

```bash
VITE_API_URL=https://api.example.com
VITE_WS_URL=wss://api.example.com/ws
```

### Frontend (Client Web)

```bash
VITE_API_URL=https://api.example.com
VITE_WS_URL=wss://api.example.com/ws
```

---

## خطوات النشر التفصيلية

### الخطوة 1: إعداد الخادم

```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# Install Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose

# Verify installation
docker --version
docker-compose --version
```

---

### الخطوة 2: إعداد Firewall

```bash
# Allow SSH
sudo ufw allow 22/tcp

# Allow HTTP/HTTPS
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

# Allow Backend API (if needed)
sudo ufw allow 8080/tcp

# Enable firewall
sudo ufw enable
```

---

### الخطوة 3: Clone Repository

```bash
# Create application directory
sudo mkdir -p /opt/sms-gateway
sudo chown $USER:$USER /opt/sms-gateway
cd /opt/sms-gateway

# Clone repository
git clone https://github.com/your-org/sms-gateway.git .

# Checkout production branch
git checkout production
```

---

### الخطوة 4: إعداد Environment Variables

```bash
# Copy environment file
cp .env.example .env.production

# Edit environment variables
nano .env.production

# IMPORTANT: Update these values:
# - DB_PASSWORD
# - REDIS_PASSWORD
# - JWT_SECRET
# - All URLs and domains
```

---

### الخطوة 5: إعداد SSL Certificate

```bash
# Install Certbot
sudo apt install certbot python3-certbot-nginx -y

# Obtain certificate
sudo certbot --nginx -d api.example.com -d admin.example.com -d client.example.com

# Verify auto-renewal
sudo certbot renew --dry-run
```

---

### الخطوة 6: بناء Docker Images

```bash
# Build backend
cd services/backend
docker build -t sms-gateway-backend:latest .

# Build admin web
cd ../../apps/admin-web
docker build -t sms-gateway-admin:latest .

# Build client web
cd ../client-web
docker build -t sms-gateway-client:latest .

# Return to root
cd ../..
```

---

### الخطوة 7: تشغيل الخدمات

```bash
# Start infrastructure services
docker-compose -f docker-compose.yml up -d postgres redis prometheus grafana

# Wait for services to be ready
sleep 10

# Run database migrations
./run-migrations.sh

# Start application services
docker-compose -f docker-compose.yml -f docker-compose.production.yml up -d backend admin-web client-web

# Verify all services are running
docker-compose ps
```

---

### الخطوة 8: التحقق من النشر

```bash
# Check backend health
curl https://api.example.com/health

# Expected response:
# {"status":"healthy","timestamp":"2026-02-26T10:30:00Z"}

# Check database connection
curl https://api.example.com/health/db

# Check Redis connection
curl https://api.example.com/health/redis

# Check Prometheus metrics
curl https://api.example.com/metrics
```

---

### الخطوة 9: إعداد Nginx Reverse Proxy

```nginx
# /etc/nginx/sites-available/sms-gateway

# Backend API
server {
    listen 443 ssl http2;
    server_name api.example.com;

    ssl_certificate /etc/letsencrypt/live/api.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/api.example.com/privkey.pem;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location /ws {
        proxy_pass http://localhost:8080/ws;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}

# Admin Web
server {
    listen 443 ssl http2;
    server_name admin.example.com;

    ssl_certificate /etc/letsencrypt/live/admin.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/admin.example.com/privkey.pem;

    location / {
        proxy_pass http://localhost:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}

# Client Web
server {
    listen 443 ssl http2;
    server_name client.example.com;

    ssl_certificate /etc/letsencrypt/live/client.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/client.example.com/privkey.pem;

    location / {
        proxy_pass http://localhost:3001;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

```bash
# Enable site
sudo ln -s /etc/nginx/sites-available/sms-gateway /etc/nginx/sites-enabled/

# Test configuration
sudo nginx -t

# Reload Nginx
sudo systemctl reload nginx
```

---

### الخطوة 10: إعداد Monitoring

```bash
# Access Grafana
# URL: https://grafana.example.com
# Default credentials: admin/admin (change immediately)

# Import dashboards
# 1. Performance Dashboard (ID: 1)
# 2. Resources Dashboard (ID: 2)
# 3. Financial Dashboard (ID: 3)

# Configure Prometheus data source
# URL: http://prometheus:9090
```

---

## إعداد Backup

### Database Backup

```bash
# Create backup script
cat > /opt/sms-gateway/backup-db.sh << 'EOF'
#!/bin/bash
BACKUP_DIR="/opt/backups/database"
DATE=$(date +%Y%m%d_%H%M%S)
mkdir -p $BACKUP_DIR

docker exec postgres pg_dump -U postgres sms_gateway | gzip > $BACKUP_DIR/backup_$DATE.sql.gz

# Keep only last 7 days
find $BACKUP_DIR -name "backup_*.sql.gz" -mtime +7 -delete
EOF

chmod +x /opt/sms-gateway/backup-db.sh

# Add to crontab (daily at 2 AM)
(crontab -l 2>/dev/null; echo "0 2 * * * /opt/sms-gateway/backup-db.sh") | crontab -
```

### Redis Backup

```bash
# Redis automatically creates RDB snapshots
# Configure in redis.conf:
# save 900 1
# save 300 10
# save 60 10000

# Backup RDB file
cat > /opt/sms-gateway/backup-redis.sh << 'EOF'
#!/bin/bash
BACKUP_DIR="/opt/backups/redis"
DATE=$(date +%Y%m%d_%H%M%S)
mkdir -p $BACKUP_DIR

docker exec redis redis-cli BGSAVE
sleep 5
docker cp redis:/data/dump.rdb $BACKUP_DIR/dump_$DATE.rdb

# Keep only last 7 days
find $BACKUP_DIR -name "dump_*.rdb" -mtime +7 -delete
EOF

chmod +x /opt/sms-gateway/backup-redis.sh

# Add to crontab (daily at 3 AM)
(crontab -l 2>/dev/null; echo "0 3 * * * /opt/sms-gateway/backup-redis.sh") | crontab -
```

---

## Rollback Plan

### خطة التراجع السريع

```bash
# Stop current version
docker-compose down

# Checkout previous version
git checkout <previous-tag>

# Restore database backup (if needed)
gunzip < /opt/backups/database/backup_YYYYMMDD_HHMMSS.sql.gz | docker exec -i postgres psql -U postgres sms_gateway

# Start previous version
docker-compose up -d

# Verify
curl https://api.example.com/health
```

---

## Health Checks

### Automated Health Checks

```bash
# Create health check script
cat > /opt/sms-gateway/health-check.sh << 'EOF'
#!/bin/bash

# Check backend
if ! curl -f https://api.example.com/health > /dev/null 2>&1; then
    echo "Backend health check failed"
    # Send alert
    exit 1
fi

# Check database
if ! docker exec postgres pg_isready -U postgres > /dev/null 2>&1; then
    echo "Database health check failed"
    # Send alert
    exit 1
fi

# Check Redis
if ! docker exec redis redis-cli ping > /dev/null 2>&1; then
    echo "Redis health check failed"
    # Send alert
    exit 1
fi

echo "All health checks passed"
EOF

chmod +x /opt/sms-gateway/health-check.sh

# Add to crontab (every 5 minutes)
(crontab -l 2>/dev/null; echo "*/5 * * * * /opt/sms-gateway/health-check.sh") | crontab -
```

---

## Scaling

### Horizontal Scaling

```bash
# Scale backend workers
docker-compose up -d --scale backend=3

# Configure load balancer
# Update Nginx upstream:
upstream backend {
    server localhost:8080;
    server localhost:8081;
    server localhost:8082;
}
```

### Vertical Scaling

```yaml
# docker-compose.production.yml
services:
  backend:
    deploy:
      resources:
        limits:
          cpus: '4'
          memory: 8G
        reservations:
          cpus: '2'
          memory: 4G
```

---

## Security Checklist

- [ ] تغيير جميع كلمات المرور الافتراضية
- [ ] تفعيل SSL/TLS لجميع الاتصالات
- [ ] تكوين Firewall بشكل صحيح
- [ ] تفعيل Database encryption at rest
- [ ] تكوين Rate limiting
- [ ] تفعيل Audit logging
- [ ] إعداد Backup منتظم
- [ ] تحديث النظام والتطبيقات بانتظام
- [ ] مراجعة Access logs
- [ ] تكوين Monitoring و Alerting

---

## Post-Deployment Tasks

1. **إنشاء مستخدم Admin الأول**
```bash
docker exec -it backend ./bin/create-admin --email admin@example.com --password secure_password
```

2. **إنشاء Tenant الأول**
```bash
docker exec -it backend ./bin/create-tenant --name "شركة تجريبية" --email tenant@example.com
```

3. **شحن رصيد تجريبي**
```bash
docker exec -it backend ./bin/add-balance --tenant-id 1 --amount 1000
```

4. **اختبار إرسال رسالة**
```bash
curl -X POST https://api.example.com/api/messages \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "phone_number": "+966501234567",
    "content": "رسالة تجريبية"
  }'
```

---

## Troubleshooting

### الخدمة لا تعمل

```bash
# Check logs
docker-compose logs backend
docker-compose logs postgres
docker-compose logs redis

# Check service status
docker-compose ps

# Restart services
docker-compose restart
```

### Database Connection Error

```bash
# Check database is running
docker exec postgres pg_isready -U postgres

# Check connection from backend
docker exec backend nc -zv postgres 5432

# Check credentials in .env
```

### High Memory Usage

```bash
# Check memory usage
docker stats

# Restart services
docker-compose restart

# Consider scaling up
```

---

## Support

للدعم الفني:
- Email: support@example.com
- Documentation: https://docs.example.com
- Status Page: https://status.example.com

---

**تاريخ الإنشاء**: 2026-02-26  
**الإصدار**: 1.0  
**آخر تحديث**: 2026-02-26
