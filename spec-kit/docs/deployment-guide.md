# Deployment Guide

This comprehensive guide covers all deployment scenarios for the Chimera Mining Pool, from development to enterprise production environments.

## 📋 Table of Contents

1. [Prerequisites](#prerequisites)
2. [Development Deployment](#development-deployment)
3. [Staging Deployment](#staging-deployment)
4. [Production Deployment](#production-deployment)
5. [Cloud Deployments](#cloud-deployments)
6. [Kubernetes Deployment](#kubernetes-deployment)
7. [High Availability Setup](#high-availability-setup)
8. [Security Hardening](#security-hardening)
9. [Monitoring & Observability](#monitoring--observability)
10. [Backup & Recovery](#backup--recovery)
11. [Troubleshooting](#troubleshooting)

## Prerequisites

### System Requirements

#### Minimum Requirements (Development)
- **CPU**: 4 cores, 2.4GHz
- **RAM**: 8GB
- **Storage**: 100GB SSD
- **Network**: 100Mbps
- **OS**: Linux (Ubuntu 20.04+), macOS 10.15+, Windows 10+

#### Recommended Production
- **CPU**: 8+ cores, 3.0GHz+
- **RAM**: 32GB+
- **Storage**: 1TB+ NVMe SSD
- **Network**: 1Gbps+
- **OS**: Linux (Ubuntu 22.04 LTS)

#### Enterprise Production
- **CPU**: 16+ cores, 3.5GHz+
- **RAM**: 64GB+
- **Storage**: 2TB+ NVMe SSD (RAID 10)
- **Network**: 10Gbps+
- **OS**: Linux (Ubuntu 22.04 LTS)

### Software Dependencies

```bash
# Required
- Docker 20.10+
- Docker Compose 2.0+
- Git 2.30+

# Optional (for manual deployment)
- Go 1.21+
- Node.js 18+
- PostgreSQL 15+
- Redis 7+
- Nginx 1.20+
```

## Development Deployment

### Quick Start

```bash
# Clone repository
git clone https://github.com/your-org/chimera-mining-pool.git
cd chimera-mining-pool/chimera-pool-core

# Start development environment
make dev-start

# Or using Docker Compose
docker-compose -f docker-compose.dev.yml up -d
```

### Development Configuration

```bash
# Copy development environment
cp .env.development .env

# Key development settings
NODE_ENV=development
LOG_LEVEL=debug
DEBUG=true
HOT_RELOAD=true
```

### Development Services

| Service | URL | Purpose |
|---------|-----|---------|
| Web Dashboard | http://localhost:3000 | Main interface |
| API Server | http://localhost:8080 | REST API |
| Stratum Server | localhost:18332 | Mining protocol |
| Grafana | http://localhost:3001 | Monitoring |
| Prometheus | http://localhost:9090 | Metrics |

## Staging Deployment

### Environment Setup

```bash
# Create staging environment
cp .env.staging .env

# Configure staging-specific settings
POOL_NAME="Chimera Pool Staging"
NODE_ENV=staging
LOG_LEVEL=info
ENABLE_DEBUG_ENDPOINTS=true
```

### Staging Docker Compose

```bash
# Deploy staging environment
docker-compose -f docker-compose.staging.yml up -d

# Initialize database
./scripts/init-staging-database.sh

# Run integration tests
./scripts/test-staging.sh
```

## Production Deployment

### Pre-Deployment Checklist

- [ ] System requirements met
- [ ] SSL certificates obtained
- [ ] DNS records configured
- [ ] Firewall rules configured
- [ ] Backup strategy implemented
- [ ] Monitoring configured
- [ ] Security hardening completed

### Production Environment Setup

```bash
# 1. Create production directories
sudo mkdir -p /opt/chimera-pool/{data,logs,backups,ssl}
sudo chown -R $USER:$USER /opt/chimera-pool

# 2. Clone and configure
git clone https://github.com/your-org/chimera-mining-pool.git /opt/chimera-pool/app
cd /opt/chimera-pool/app/chimera-pool-core

# 3. Production configuration
cp spec-kit/examples/configs/production.env .env
# Edit .env with your production values

# 4. Generate secrets
./scripts/generate-production-secrets.sh

# 5. SSL certificates
./scripts/setup-ssl.sh

# 6. Deploy
docker-compose -f spec-kit/examples/deployments/docker-compose.production.yml up -d
```

### Production Configuration

Key production settings in `.env`:

```bash
# Pool Configuration
POOL_NAME="Your Pool Name"
POOL_FEE=1.0
PUBLIC_API_URL=https://api.yourpool.com
PUBLIC_WEB_URL=https://yourpool.com

# Security
SSL_ENABLED=true
HSTS_ENABLED=true
MFA_ENABLED=true
RATE_LIMIT_ENABLED=true

# Performance
MAX_CONNECTIONS=2000
WORKER_PROCESSES=auto
CACHE_STRATEGY=redis

# Monitoring
METRICS_ENABLED=true
ALERTS_ENABLED=true
LOG_LEVEL=info
```

### Database Setup

```bash
# Initialize production database
./scripts/init-production-database.sh

# Run migrations
./scripts/migrate-production.sh

# Create admin user
./scripts/create-admin-user.sh
```

## Cloud Deployments

### AWS Deployment

#### Using Terraform

```bash
# 1. Configure AWS credentials
aws configure

# 2. Deploy infrastructure
cd deployments/terraform/aws
terraform init
terraform plan -var-file="production.tfvars"
terraform apply

# 3. Deploy application
./scripts/deploy-to-aws.sh
```

#### AWS Services Used
- **EC2**: Application servers
- **RDS**: PostgreSQL database
- **ElastiCache**: Redis cache
- **ALB**: Load balancer
- **S3**: Backup storage
- **CloudWatch**: Monitoring
- **Route53**: DNS management

### Google Cloud Platform

```bash
# 1. Setup GCP project
gcloud config set project your-project-id

# 2. Deploy using Cloud Run
./scripts/deploy-to-gcp.sh

# 3. Configure Cloud SQL
./scripts/setup-gcp-database.sh
```

### Azure Deployment

```bash
# 1. Login to Azure
az login

# 2. Create resource group
az group create --name chimera-pool --location eastus

# 3. Deploy using ARM templates
az deployment group create \
  --resource-group chimera-pool \
  --template-file deployments/azure/template.json
```

## Kubernetes Deployment

### Prerequisites

```bash
# Install kubectl and helm
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
```

### Helm Deployment

```bash
# 1. Add Helm repository
helm repo add chimera-pool https://charts.chimera-pool.com
helm repo update

# 2. Create namespace
kubectl create namespace chimera-pool

# 3. Install with Helm
helm install chimera-pool chimera-pool/chimera-mining-pool \
  --namespace chimera-pool \
  --values deployments/kubernetes/values.production.yaml

# 4. Verify deployment
kubectl get pods -n chimera-pool
```

### Manual Kubernetes Deployment

```bash
# 1. Apply configurations
kubectl apply -f deployments/kubernetes/namespace.yaml
kubectl apply -f deployments/kubernetes/configmap.yaml
kubectl apply -f deployments/kubernetes/secrets.yaml
kubectl apply -f deployments/kubernetes/postgres.yaml
kubectl apply -f deployments/kubernetes/redis.yaml
kubectl apply -f deployments/kubernetes/api-server.yaml
kubectl apply -f deployments/kubernetes/stratum-server.yaml
kubectl apply -f deployments/kubernetes/web-dashboard.yaml
kubectl apply -f deployments/kubernetes/ingress.yaml

# 2. Check status
kubectl get all -n chimera-pool
```

## High Availability Setup

### Load Balancer Configuration

```nginx
# /etc/nginx/sites-available/chimera-pool
upstream api_servers {
    least_conn;
    server api-server-1:8080 max_fails=3 fail_timeout=30s;
    server api-server-2:8080 max_fails=3 fail_timeout=30s;
    server api-server-3:8080 max_fails=3 fail_timeout=30s;
}

upstream stratum_servers {
    ip_hash;
    server stratum-server-1:18332 max_fails=3 fail_timeout=30s;
    server stratum-server-2:18332 max_fails=3 fail_timeout=30s;
}

server {
    listen 443 ssl http2;
    server_name api.yourpool.com;
    
    ssl_certificate /etc/ssl/certs/yourpool.com.crt;
    ssl_certificate_key /etc/ssl/private/yourpool.com.key;
    
    location / {
        proxy_pass http://api_servers;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### Database High Availability

#### PostgreSQL Streaming Replication

```bash
# Master configuration (postgresql.conf)
wal_level = replica
max_wal_senders = 3
wal_keep_segments = 64
archive_mode = on
archive_command = 'cp %p /var/lib/postgresql/archive/%f'

# Replica configuration
standby_mode = 'on'
primary_conninfo = 'host=postgres-master port=5432 user=replicator'
restore_command = 'cp /var/lib/postgresql/archive/%f %p'
```

#### Redis Sentinel

```bash
# Sentinel configuration
sentinel monitor mymaster redis-master 6379 2
sentinel down-after-milliseconds mymaster 5000
sentinel failover-timeout mymaster 10000
sentinel parallel-syncs mymaster 1
```

### Auto-Scaling Configuration

```yaml
# Kubernetes HPA
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: api-server-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: api-server
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

## Security Hardening

### SSL/TLS Configuration

```bash
# Generate SSL certificates
./scripts/generate-ssl-certs.sh

# Or use Let's Encrypt
certbot certonly --webroot -w /var/www/html -d yourpool.com -d api.yourpool.com
```

### Firewall Configuration

```bash
# UFW (Ubuntu)
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow ssh
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw allow 18332/tcp
sudo ufw enable

# iptables
iptables -A INPUT -p tcp --dport 22 -j ACCEPT
iptables -A INPUT -p tcp --dport 80 -j ACCEPT
iptables -A INPUT -p tcp --dport 443 -j ACCEPT
iptables -A INPUT -p tcp --dport 18332 -j ACCEPT
iptables -A INPUT -j DROP
```

### Security Headers

```nginx
# Security headers in Nginx
add_header X-Frame-Options "SAMEORIGIN" always;
add_header X-Content-Type-Options "nosniff" always;
add_header X-XSS-Protection "1; mode=block" always;
add_header Referrer-Policy "no-referrer-when-downgrade" always;
add_header Content-Security-Policy "default-src 'self'" always;
add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
```

### Fail2Ban Configuration

```ini
# /etc/fail2ban/jail.local
[chimera-api]
enabled = true
port = 80,443
filter = chimera-api
logpath = /opt/chimera-pool/logs/api.log
maxretry = 5
bantime = 3600

[chimera-stratum]
enabled = true
port = 18332
filter = chimera-stratum
logpath = /opt/chimera-pool/logs/stratum.log
maxretry = 3
bantime = 7200
```

## Monitoring & Observability

### Prometheus Configuration

```yaml
# prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  - "alert_rules.yml"

scrape_configs:
  - job_name: 'chimera-api'
    static_configs:
      - targets: ['api-server:8080']
    metrics_path: /metrics
    scrape_interval: 30s

  - job_name: 'chimera-stratum'
    static_configs:
      - targets: ['stratum-server:9091']

  - job_name: 'postgres'
    static_configs:
      - targets: ['postgres-exporter:9187']

  - job_name: 'redis'
    static_configs:
      - targets: ['redis-exporter:9121']

alerting:
  alertmanagers:
    - static_configs:
        - targets:
          - alertmanager:9093
```

### Grafana Dashboards

Key dashboards to import:
- Pool Overview Dashboard
- System Metrics Dashboard
- Mining Performance Dashboard
- Security Events Dashboard
- Database Performance Dashboard

### Log Aggregation

```yaml
# Logstash pipeline
input {
  file {
    path => "/logs/api/*.log"
    type => "api"
    codec => "json"
  }
  file {
    path => "/logs/stratum/*.log"
    type => "stratum"
    codec => "json"
  }
}

filter {
  if [type] == "api" {
    mutate {
      add_field => { "service" => "api-server" }
    }
  }
}

output {
  elasticsearch {
    hosts => ["elasticsearch:9200"]
    index => "chimera-pool-%{+YYYY.MM.dd}"
  }
}
```

## Backup & Recovery

### Automated Backup Script

```bash
#!/bin/bash
# backup.sh

BACKUP_DIR="/opt/chimera-pool/backups"
DATE=$(date +%Y%m%d_%H%M%S)
DB_BACKUP="$BACKUP_DIR/db_backup_$DATE.sql"
CONFIG_BACKUP="$BACKUP_DIR/config_backup_$DATE.tar.gz"

# Database backup
docker exec chimera-postgres pg_dump -U chimera chimera_pool > "$DB_BACKUP"

# Configuration backup
tar -czf "$CONFIG_BACKUP" /opt/chimera-pool/app/.env /opt/chimera-pool/ssl

# Upload to S3
aws s3 cp "$DB_BACKUP" s3://your-backup-bucket/database/
aws s3 cp "$CONFIG_BACKUP" s3://your-backup-bucket/config/

# Cleanup old backups (keep 30 days)
find "$BACKUP_DIR" -name "*.sql" -mtime +30 -delete
find "$BACKUP_DIR" -name "*.tar.gz" -mtime +30 -delete
```

### Recovery Procedures

```bash
# Database recovery
docker exec -i chimera-postgres psql -U chimera chimera_pool < backup.sql

# Configuration recovery
tar -xzf config_backup.tar.gz -C /

# Service restart
docker-compose restart
```

## Troubleshooting

### Common Issues

#### Services Won't Start

```bash
# Check logs
docker-compose logs -f

# Check system resources
df -h
free -h
docker system df

# Check port conflicts
netstat -tulpn | grep :8080
```

#### Database Connection Issues

```bash
# Test database connection
docker exec chimera-postgres pg_isready -U chimera

# Check database logs
docker logs chimera-postgres

# Reset database
docker-compose down -v
docker-compose up -d postgres
./scripts/init-database.sh
```

#### Performance Issues

```bash
# Monitor resource usage
htop
iotop
docker stats

# Check database performance
docker exec chimera-postgres psql -U chimera -c "SELECT * FROM pg_stat_activity;"

# Analyze slow queries
docker exec chimera-postgres psql -U chimera -c "SELECT query, mean_time FROM pg_stat_statements ORDER BY mean_time DESC LIMIT 10;"
```

#### SSL Certificate Issues

```bash
# Check certificate validity
openssl x509 -in /etc/ssl/certs/yourpool.com.crt -text -noout

# Test SSL configuration
openssl s_client -connect yourpool.com:443

# Renew Let's Encrypt certificates
certbot renew --dry-run
```

### Health Checks

```bash
# API health check
curl -f http://localhost:8080/health

# Database health check
docker exec chimera-postgres pg_isready -U chimera

# Redis health check
docker exec chimera-redis redis-cli ping

# Stratum server check
nc -z localhost 18332
```

### Performance Tuning

#### Database Optimization

```sql
-- PostgreSQL tuning
ALTER SYSTEM SET shared_buffers = '256MB';
ALTER SYSTEM SET effective_cache_size = '1GB';
ALTER SYSTEM SET maintenance_work_mem = '64MB';
ALTER SYSTEM SET checkpoint_completion_target = 0.9;
ALTER SYSTEM SET wal_buffers = '16MB';
ALTER SYSTEM SET default_statistics_target = 100;
SELECT pg_reload_conf();
```

#### Application Tuning

```bash
# Go application tuning
export GOGC=100
export GOMAXPROCS=8

# Node.js tuning
export NODE_OPTIONS="--max-old-space-size=4096"
```

### Disaster Recovery

#### Complete System Recovery

```bash
# 1. Restore infrastructure
terraform apply

# 2. Restore database
./scripts/restore-database.sh backup_20231219_120000.sql

# 3. Restore configuration
tar -xzf config_backup_20231219_120000.tar.gz -C /

# 4. Start services
docker-compose up -d

# 5. Verify functionality
./scripts/health-check.sh
```

This deployment guide provides comprehensive coverage for all deployment scenarios. For specific questions or advanced configurations, refer to the individual component documentation or contact support.