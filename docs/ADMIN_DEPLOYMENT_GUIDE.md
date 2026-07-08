# Admin Interface Deployment Guide

**Version:** 1.0.0  
**Last Updated:** April 9, 2026

---

## Table of Contents

1. [Infrastructure Requirements](#infrastructure-requirements)
2. [Pre-Deployment Checklist](#pre-deployment-checklist)
3. [Database Setup](#database-setup)
4. [Backend Deployment](#backend-deployment)
5. [Frontend Deployment](#frontend-deployment)
6. [Monitoring Setup](#monitoring-setup)
7. [Security Configuration](#security-configuration)
8. [Backup & Recovery](#backup--recovery)
9. [Troubleshooting](#troubleshooting)

---

## Infrastructure Requirements

### Minimum Requirements

**Backend Server:**
- CPU: 4 cores
- RAM: 8 GB
- Storage: 50 GB SSD
- OS: Ubuntu 22.04 LTS or similar

**Database Server:**
- CPU: 4 cores
- RAM: 16 GB
- Storage: 200 GB SSD (with room for growth)
- PostgreSQL 15+

**Redis Server:**
- CPU: 2 cores
- RAM: 4 GB
- Storage: 20 GB SSD
- Redis 7+

**Frontend Server:**
- CPU: 2 cores
- RAM: 4 GB
- Storage: 20 GB SSD
- Nginx or similar web server

### Recommended Production Setup

**Backend:**
- CPU: 8 cores
- RAM: 16 GB
- Storage: 100 GB SSD
- Load Balancer: 2+ instances

**Database:**
- CPU: 8 cores
- RAM: 32 GB
- Storage: 500 GB SSD
- Replication: Primary + 1 Replica

**Redis:**
- CPU: 4 cores
- RAM: 8 GB
- Storage: 50 GB SSD
- Replication: Primary + 1 Replica

**Frontend:**
- CDN: CloudFlare or similar
- Multiple edge locations
- SSL/TLS certificates

### Network Requirements

- **Bandwidth**: Minimum 100 Mbps, recommended 1 Gbps
- **Latency**: < 50ms between services
- **Firewall**: Configured for required ports only
- **SSL/TLS**: Required for all external connections

---

## Pre-Deployment Checklist

### Code Preparation

- [ ] All tests passing (unit, integration, security)
- [ ] Code reviewed and approved
- [ ] Version tagged in Git
- [ ] Changelog updated
- [ ] Documentation updated

### Infrastructure

- [ ] Servers provisioned
- [ ] Network configured
- [ ] Firewall rules set
- [ ] SSL certificates obtained
- [ ] DNS records configured
- [ ] Load balancer configured

### Database

- [ ] Database created
- [ ] User accounts created
- [ ] Permissions configured
- [ ] Backup system configured
- [ ] Replication configured (if applicable)

### Monitoring

- [ ] Prometheus installed
- [ ] Grafana installed
- [ ] Alert rules configured
- [ ] Log aggregation configured
- [ ] Status page setup

### Security

- [ ] Secrets management configured
- [ ] API keys generated
- [ ] JWT secrets generated
- [ ] CORS configured
- [ ] Rate limiting configured
- [ ] Security headers configured

---

## Database Setup

### 1. Install PostgreSQL

```bash
# Ubuntu/Debian
sudo apt update
sudo apt install postgresql-15 postgresql-contrib

# Start service
sudo systemctl start postgresql
sudo systemctl enable postgresql
```

### 2. Create Database and User

```sql
-- Connect as postgres user
sudo -u postgres psql

-- Create database
CREATE DATABASE admin_prod;

-- Create user
CREATE USER admin_user WITH ENCRYPTED PASSWORD 'secure_password';

-- Grant privileges
GRANT ALL PRIVILEGES ON DATABASE admin_prod TO admin_user;

-- Exit
\q
```

### 3. Configure PostgreSQL

Edit `/etc/postgresql/15/main/postgresql.conf`:

```conf
# Connection settings
listen_addresses = '*'
max_connections = 200

# Memory settings
shared_buffers = 4GB
effective_cache_size = 12GB
maintenance_work_mem = 1GB
work_mem = 20MB

# WAL settings
wal_buffers = 16MB
checkpoint_completion_target = 0.9
```

Edit `/etc/postgresql/15/main/pg_hba.conf`:

```conf
# Allow connections from application servers
host    admin_prod    admin_user    10.0.0.0/24    md5
```

Restart PostgreSQL:

```bash
sudo systemctl restart postgresql
```

### 4. Run Migrations

```bash
# Install migrate tool
curl -L https://github.com/golang-migrate/migrate/releases/download/v4.16.2/migrate.linux-amd64.tar.gz | tar xvz
sudo mv migrate /usr/local/bin/

# Run migrations
migrate -path services/backend/migrations \
        -database "postgresql://admin_user:secure_password@localhost:5432/admin_prod?sslmode=require" \
        up
```

### 5. Setup Replication (Optional but Recommended)

**On Primary:**

```sql
-- Create replication user
CREATE USER replicator WITH REPLICATION ENCRYPTED PASSWORD 'repl_password';
```

Edit `postgresql.conf`:

```conf
wal_level = replica
max_wal_senders = 3
wal_keep_size = 1GB
```

**On Replica:**

```bash
# Stop PostgreSQL
sudo systemctl stop postgresql

# Remove data directory
sudo rm -rf /var/lib/postgresql/15/main/*

# Create base backup
pg_basebackup -h primary_ip -D /var/lib/postgresql/15/main -U replicator -P -v -R

# Start PostgreSQL
sudo systemctl start postgresql
```

---

## Backend Deployment

### 1. Build Application

```bash
# Clone repository
git clone https://github.com/your-org/al-hudhud-gateway.git
cd al-hudhud-gateway/services/backend

# Build binary
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o admin-server cmd/server/main.go
```

### 2. Create Systemd Service

Create `/etc/systemd/system/admin-backend.service`:

```ini
[Unit]
Description=Admin Backend Service
After=network.target postgresql.service redis.service

[Service]
Type=simple
User=admin
WorkingDirectory=/opt/admin-backend
ExecStart=/opt/admin-backend/admin-server
Restart=always
RestartSec=10

# Environment variables
Environment="DATABASE_URL=postgresql://admin_user:secure_password@localhost:5432/admin_prod"
Environment="REDIS_URL=redis://localhost:6379"
Environment="PORT=8080"
Environment="ENV=production"
Environment="JWT_SECRET=your-jwt-secret"

# Security
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/log/admin-backend

[Install]
WantedBy=multi-user.target
```

### 3. Deploy and Start

```bash
# Create directory
sudo mkdir -p /opt/admin-backend
sudo mkdir -p /var/log/admin-backend

# Copy binary
sudo cp admin-server /opt/admin-backend/

# Set permissions
sudo chown -R admin:admin /opt/admin-backend
sudo chown -R admin:admin /var/log/admin-backend

# Enable and start service
sudo systemctl daemon-reload
sudo systemctl enable admin-backend
sudo systemctl start admin-backend

# Check status
sudo systemctl status admin-backend
```

### 4. Configure Nginx Reverse Proxy

Create `/etc/nginx/sites-available/admin-backend`:

```nginx
upstream admin_backend {
    server localhost:8080;
    # Add more servers for load balancing
    # server localhost:8081;
}

server {
    listen 80;
    server_name api.admin.example.com;
    
    # Redirect to HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name api.admin.example.com;

    # SSL configuration
    ssl_certificate /etc/letsencrypt/live/api.admin.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/api.admin.example.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;

    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;

    # Proxy settings
    location /api/ {
        proxy_pass http://admin_backend;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
        
        # Timeouts
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }

    # Rate limiting
    limit_req_zone $binary_remote_addr zone=api_limit:10m rate=100r/m;
    limit_req zone=api_limit burst=20 nodelay;
}
```

Enable and restart Nginx:

```bash
sudo ln -s /etc/nginx/sites-available/admin-backend /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl restart nginx
```

---

## Frontend Deployment

### 1. Build Frontend

```bash
cd apps/admin-web

# Install dependencies
npm ci --production

# Build for production
npm run build

# Output in dist/ directory
```

### 2. Configure Nginx for Frontend

Create `/etc/nginx/sites-available/admin-frontend`:

```nginx
server {
    listen 80;
    server_name admin.example.com;
    
    # Redirect to HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name admin.example.com;

    # SSL configuration
    ssl_certificate /etc/letsencrypt/live/admin.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/admin.example.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;

    # Root directory
    root /var/www/admin-frontend;
    index index.html;

    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;

    # Gzip compression
    gzip on;
    gzip_vary on;
    gzip_min_length 1024;
    gzip_types text/plain text/css text/xml text/javascript application/x-javascript application/xml+rss application/json;

    # Cache static assets
    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)$ {
        expires 1y;
        add_header Cache-Control "public, immutable";
    }

    # SPA routing
    location / {
        try_files $uri $uri/ /index.html;
    }
}
```

### 3. Deploy Frontend

```bash
# Create directory
sudo mkdir -p /var/www/admin-frontend

# Copy build files
sudo cp -r dist/* /var/www/admin-frontend/

# Set permissions
sudo chown -R www-data:www-data /var/www/admin-frontend

# Enable site
sudo ln -s /etc/nginx/sites-available/admin-frontend /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl restart nginx
```

---

## Monitoring Setup

### 1. Install Prometheus

```bash
# Download Prometheus
wget https://github.com/prometheus/prometheus/releases/download/v2.45.0/prometheus-2.45.0.linux-amd64.tar.gz
tar xvfz prometheus-2.45.0.linux-amd64.tar.gz
sudo mv prometheus-2.45.0.linux-amd64 /opt/prometheus

# Create configuration
sudo nano /opt/prometheus/prometheus.yml
```

Prometheus configuration:

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'admin-backend'
    static_configs:
      - targets: ['localhost:8080']
  
  - job_name: 'postgres'
    static_configs:
      - targets: ['localhost:9187']
  
  - job_name: 'redis'
    static_configs:
      - targets: ['localhost:9121']
```

Create systemd service `/etc/systemd/system/prometheus.service`:

```ini
[Unit]
Description=Prometheus
After=network.target

[Service]
Type=simple
User=prometheus
ExecStart=/opt/prometheus/prometheus --config.file=/opt/prometheus/prometheus.yml --storage.tsdb.path=/var/lib/prometheus
Restart=always

[Install]
WantedBy=multi-user.target
```

Start Prometheus:

```bash
sudo systemctl daemon-reload
sudo systemctl enable prometheus
sudo systemctl start prometheus
```

### 2. Install Grafana

```bash
# Add Grafana repository
sudo apt-get install -y software-properties-common
sudo add-apt-repository "deb https://packages.grafana.com/oss/deb stable main"
wget -q -O - https://packages.grafana.com/gpg.key | sudo apt-key add -

# Install Grafana
sudo apt-get update
sudo apt-get install grafana

# Start Grafana
sudo systemctl enable grafana-server
sudo systemctl start grafana-server
```

Access Grafana at `http://localhost:3000` (default credentials: admin/admin)

### 3. Import Dashboards

Import the pre-configured dashboards from `grafana/dashboards/` directory.

---

## Security Configuration

### 1. Firewall Rules

```bash
# Allow SSH
sudo ufw allow 22/tcp

# Allow HTTP/HTTPS
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

# Allow PostgreSQL (from app servers only)
sudo ufw allow from 10.0.0.0/24 to any port 5432

# Allow Redis (from app servers only)
sudo ufw allow from 10.0.0.0/24 to any port 6379

# Enable firewall
sudo ufw enable
```

### 2. SSL/TLS Certificates

```bash
# Install Certbot
sudo apt install certbot python3-certbot-nginx

# Obtain certificates
sudo certbot --nginx -d admin.example.com -d api.admin.example.com

# Auto-renewal
sudo certbot renew --dry-run
```

### 3. Secrets Management

Use environment variables or a secrets manager (e.g., HashiCorp Vault, AWS Secrets Manager).

Never commit secrets to version control!

---

## Backup & Recovery

### Database Backup

```bash
# Create backup script
sudo nano /usr/local/bin/backup-admin-db.sh
```

```bash
#!/bin/bash
BACKUP_DIR="/var/backups/admin-db"
DATE=$(date +%Y%m%d_%H%M%S)
FILENAME="admin_prod_$DATE.sql.gz"

mkdir -p $BACKUP_DIR

pg_dump -U admin_user admin_prod | gzip > $BACKUP_DIR/$FILENAME

# Keep only last 30 days
find $BACKUP_DIR -name "*.sql.gz" -mtime +30 -delete

echo "Backup completed: $FILENAME"
```

```bash
# Make executable
sudo chmod +x /usr/local/bin/backup-admin-db.sh

# Add to crontab (daily at 2 AM)
sudo crontab -e
0 2 * * * /usr/local/bin/backup-admin-db.sh
```

### Recovery

```bash
# Restore from backup
gunzip < /var/backups/admin-db/admin_prod_20260409_020000.sql.gz | psql -U admin_user admin_prod
```

---

## Troubleshooting

### Backend Not Starting

```bash
# Check logs
sudo journalctl -u admin-backend -f

# Check if port is in use
sudo netstat -tulpn | grep 8080

# Check database connection
psql -U admin_user -h localhost admin_prod
```

### High Memory Usage

```bash
# Check memory usage
free -h
top

# Check PostgreSQL connections
psql -U admin_user admin_prod -c "SELECT count(*) FROM pg_stat_activity;"

# Restart services if needed
sudo systemctl restart admin-backend
```

### Slow Queries

```sql
-- Enable slow query logging
ALTER SYSTEM SET log_min_duration_statement = 1000; -- 1 second
SELECT pg_reload_conf();

-- Check slow queries
SELECT query, calls, total_time, mean_time
FROM pg_stat_statements
ORDER BY mean_time DESC
LIMIT 10;
```

---

## Post-Deployment Checklist

- [ ] All services running
- [ ] Health checks passing
- [ ] Monitoring dashboards showing data
- [ ] Alerts configured and tested
- [ ] Backups running successfully
- [ ] SSL certificates valid
- [ ] DNS resolving correctly
- [ ] Load testing completed
- [ ] Security scan completed
- [ ] Documentation updated

---

**Deployment Guide Version:** 1.0.0  
**Last Updated:** April 9, 2026  
**Support:** devops@example.com
