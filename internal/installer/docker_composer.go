package installer

import (
	"fmt"
	"strings"
)

type DockerComposer struct{}

func NewDockerComposer() *DockerComposer {
	return &DockerComposer{}
}

func (dc *DockerComposer) Generate(config PoolConfig) (string, error) {
	template := `version: '3.8'

services:
  chimera-pool-core:
    image: chimera-pool/core:latest
    container_name: chimera-pool-core
    restart: unless-stopped
    ports:
      - "8080:8080"   # Web dashboard
      - "4444:4444"   # Stratum server
      - "9090:9090"   # Metrics endpoint
    environment:
      - DATABASE_URL=postgresql://pool_user:pool_password@postgresql:5432/chimera_pool
      - REDIS_URL=redis://redis:6379
      - STRATUM_PORT=4444
      - STRATUM_MAX_CONNECTIONS={{.StratumMaxConnections}}
      - STRATUM_WORKER_THREADS={{.StratumWorkerThreads}}
      - MAX_MINERS={{.MaxMiners}}
      - LOG_LEVEL={{.LogLevel}}
      - METRICS_ENABLED={{.MetricsEnabled}}
      - RATE_LIMIT_ENABLED={{.RateLimitEnabled}}
      - MFA_REQUIRED={{.MFARequired}}
    volumes:
      - ./config:/app/config:ro
      - ./logs:/app/logs
      - ./data:/app/data
    depends_on:
      - postgresql
      - redis
    networks:
      - chimera-pool-network
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s

  postgresql:
    image: postgres:15-alpine
    container_name: chimera-pool-db
    restart: unless-stopped
    environment:
      - POSTGRES_DB=chimera_pool
      - POSTGRES_USER=pool_user
      - POSTGRES_PASSWORD=pool_password
      - POSTGRES_MAX_CONNECTIONS={{.DatabaseMaxConnections}}
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./migrations:/docker-entrypoint-initdb.d:ro
    networks:
      - chimera-pool-network
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U pool_user -d chimera_pool"]
      interval: 10s
      timeout: 5s
      retries: 5
    command: >
      postgres
      -c max_connections={{.DatabaseMaxConnections}}
      -c shared_buffers=256MB
      -c effective_cache_size=1GB
      -c maintenance_work_mem=64MB
      -c checkpoint_completion_target=0.9
      -c wal_buffers=16MB
      -c default_statistics_target=100

  redis:
    image: redis:7-alpine
    container_name: chimera-pool-redis
    restart: unless-stopped
    command: >
      redis-server
      --maxmemory 512mb
      --maxmemory-policy allkeys-lru
      --maxclients {{.RedisMaxConnections}}
      --save 900 1
      --save 300 10
      --save 60 10000
    volumes:
      - redis_data:/data
    networks:
      - chimera-pool-network
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 3s
      retries: 3

  nginx:
    image: nginx:alpine
    container_name: chimera-pool-nginx
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf:ro
      - ./nginx/ssl:/etc/nginx/ssl:ro
      - ./logs/nginx:/var/log/nginx
    depends_on:
      - chimera-pool-core
    networks:
      - chimera-pool-network
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost/health"]
      interval: 30s
      timeout: 10s
      retries: 3

{{if .MetricsEnabled}}
  prometheus:
    image: prom/prometheus:latest
    container_name: chimera-pool-prometheus
    restart: unless-stopped
    ports:
      - "9091:9090"
    volumes:
      - ./monitoring/prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - prometheus_data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.console.libraries=/etc/prometheus/console_libraries'
      - '--web.console.templates=/etc/prometheus/consoles'
      - '--storage.tsdb.retention.time=200h'
      - '--web.enable-lifecycle'
    networks:
      - chimera-pool-network

  grafana:
    image: grafana/grafana:latest
    container_name: chimera-pool-grafana
    restart: unless-stopped
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
      - GF_USERS_ALLOW_SIGN_UP=false
    volumes:
      - grafana_data:/var/lib/grafana
      - ./monitoring/grafana/dashboards:/etc/grafana/provisioning/dashboards:ro
      - ./monitoring/grafana/datasources:/etc/grafana/provisioning/datasources:ro
    networks:
      - chimera-pool-network
{{end}}

volumes:
  postgres_data:
    driver: local
  redis_data:
    driver: local
{{if .MetricsEnabled}}
  prometheus_data:
    driver: local
  grafana_data:
    driver: local
{{end}}

networks:
  chimera-pool-network:
    driver: bridge
    ipam:
      config:
        - subnet: 172.20.0.0/16
`

	// Replace template variables
	result := strings.ReplaceAll(template, "{{.StratumMaxConnections}}", fmt.Sprintf("%d", config.StratumConfig.MaxConnections))
	result = strings.ReplaceAll(result, "{{.StratumWorkerThreads}}", fmt.Sprintf("%d", config.StratumConfig.WorkerThreads))
	result = strings.ReplaceAll(result, "{{.MaxMiners}}", fmt.Sprintf("%d", config.MaxMiners))
	result = strings.ReplaceAll(result, "{{.LogLevel}}", config.MonitoringConfig.LogLevel)
	result = strings.ReplaceAll(result, "{{.MetricsEnabled}}", fmt.Sprintf("%t", config.MonitoringConfig.MetricsEnabled))
	result = strings.ReplaceAll(result, "{{.RateLimitEnabled}}", fmt.Sprintf("%t", config.SecurityConfig.RateLimitEnabled))
	result = strings.ReplaceAll(result, "{{.MFARequired}}", fmt.Sprintf("%t", config.SecurityConfig.MFARequired))
	result = strings.ReplaceAll(result, "{{.DatabaseMaxConnections}}", fmt.Sprintf("%d", config.DatabaseConfig.MaxConnections))
	result = strings.ReplaceAll(result, "{{.RedisMaxConnections}}", fmt.Sprintf("%d", config.RedisConfig.MaxConnections))

	// Handle conditional sections
	if config.MonitoringConfig.MetricsEnabled {
		result = strings.ReplaceAll(result, "{{if .MetricsEnabled}}", "")
		result = strings.ReplaceAll(result, "{{end}}", "")
	} else {
		// Remove metrics services
		lines := strings.Split(result, "\n")
		var filteredLines []string
		skipSection := false
		
		for _, line := range lines {
			if strings.Contains(line, "{{if .MetricsEnabled}}") {
				skipSection = true
				continue
			}
			if strings.Contains(line, "{{end}}") {
				skipSection = false
				continue
			}
			if !skipSection {
				filteredLines = append(filteredLines, line)
			}
		}
		result = strings.Join(filteredLines, "\n")
	}

	return result, nil
}

func (dc *DockerComposer) GenerateNginxConfig() string {
	return `events {
    worker_connections 1024;
}

http {
    include       /etc/nginx/mime.types;
    default_type  application/octet-stream;

    log_format main '$remote_addr - $remote_user [$time_local] "$request" '
                    '$status $body_bytes_sent "$http_referer" '
                    '"$http_user_agent" "$http_x_forwarded_for"';

    access_log /var/log/nginx/access.log main;
    error_log /var/log/nginx/error.log warn;

    sendfile on;
    tcp_nopush on;
    tcp_nodelay on;
    keepalive_timeout 65;
    types_hash_max_size 2048;

    # Rate limiting
    limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;
    limit_req_zone $binary_remote_addr zone=dashboard:10m rate=30r/s;

    # Upstream for pool core service
    upstream chimera_pool_core {
        server chimera-pool-core:8080;
        keepalive 32;
    }

    server {
        listen 80;
        server_name _;

        # Health check endpoint
        location /health {
            access_log off;
            return 200 "healthy\n";
            add_header Content-Type text/plain;
        }

        # API endpoints with rate limiting
        location /api/ {
            limit_req zone=api burst=20 nodelay;
            
            proxy_pass http://chimera_pool_core;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection 'upgrade';
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
            proxy_cache_bypass $http_upgrade;
            
            # CORS headers
            add_header Access-Control-Allow-Origin *;
            add_header Access-Control-Allow-Methods "GET, POST, PUT, DELETE, OPTIONS";
            add_header Access-Control-Allow-Headers "DNT,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Range,Authorization";
        }

        # WebSocket for real-time updates
        location /ws {
            proxy_pass http://chimera_pool_core;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
            proxy_read_timeout 86400;
        }

        # Dashboard with rate limiting
        location / {
            limit_req zone=dashboard burst=50 nodelay;
            
            proxy_pass http://chimera_pool_core;
            proxy_http_version 1.1;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
            
            # Security headers
            add_header X-Frame-Options DENY;
            add_header X-Content-Type-Options nosniff;
            add_header X-XSS-Protection "1; mode=block";
            add_header Referrer-Policy strict-origin-when-cross-origin;
        }
    }

    # HTTPS server (when SSL is enabled)
    # server {
    #     listen 443 ssl http2;
    #     server_name your-domain.com;
    #
    #     ssl_certificate /etc/nginx/ssl/cert.pem;
    #     ssl_certificate_key /etc/nginx/ssl/key.pem;
    #     ssl_protocols TLSv1.2 TLSv1.3;
    #     ssl_ciphers ECDHE-RSA-AES128-GCM-SHA256:ECDHE-RSA-AES256-GCM-SHA384;
    #     ssl_prefer_server_ciphers off;
    #
    #     # Same location blocks as HTTP server
    # }
}

# TCP load balancing for Stratum protocol
stream {
    upstream stratum_backend {
        server chimera-pool-core:4444;
    }

    server {
        listen 4444;
        proxy_pass stratum_backend;
        proxy_timeout 1s;
        proxy_responses 1;
        error_log /var/log/nginx/stratum.log;
    }
}`
}