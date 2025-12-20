# Chimeria Pool Deployment Guide

## Production Server Information

- **Server IP**: 206.162.80.230
- **Server Location**: Local machine (this computer, port-forwarded to public IP)
- **Access**: https://206.162.80.230 via browser
- **GitHub Repo**: https://github.com/chimera-pool/chimera-pool-core.git

## Quick Deployment Commands

### Deploy ALL Services (Full Rebuild)
```powershell
cd c:\Users\Reid Davis\CascadeProjects\ChimeriaPool\chimera-pool-core
docker-compose -f deployments/docker/docker-compose.yml up --build -d
```

### Deploy Specific Services

**API Only (Backend changes):**
```powershell
docker-compose -f deployments/docker/docker-compose.yml up --build -d chimera-pool-api
```

**Web Dashboard Only (Frontend changes):**
```powershell
docker-compose -f deployments/docker/docker-compose.yml up --build -d chimera-pool-web
```

**Nginx (Reverse Proxy):**
```powershell
docker-compose -f deployments/docker/docker-compose.yml up --build -d nginx
```

**Stratum Server (Mining):**
```powershell
docker-compose -f deployments/docker/docker-compose.yml up --build -d chimera-pool-stratum
```

### Restart Services
```powershell
docker restart docker-chimera-pool-api-1
docker restart docker-chimera-pool-web-1
docker restart docker-nginx-1
docker restart docker-chimera-pool-stratum-1
```

## Git Workflow

### Before Deploying
1. Make code changes
2. Test locally: `go build -o api.exe ./cmd/api`
3. Commit changes:
```powershell
git add .
git commit -m "Description of changes"
git push origin main
```

### After Pushing to GitHub
Deploy the relevant containers (see commands above).

## Container Status Check
```powershell
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
```

## Logs
```powershell
# All logs
docker-compose -f deployments/docker/docker-compose.yml logs -f

# Specific service
docker logs docker-chimera-pool-api-1 -f
docker logs docker-chimera-pool-web-1 -f
docker logs docker-nginx-1 -f
```

## Health Checks
```powershell
# API health
Invoke-RestMethod -Uri "http://localhost:8080/health" -Method Get

# Check all container health
docker ps --format "table {{.Names}}\t{{.Status}}"
```

## Database

**Location**: PostgreSQL running in Docker container `docker-postgres-1`

**Connect to DB:**
```powershell
docker exec -it docker-postgres-1 psql -U chimera -d chimera_pool
```

**Run Migrations:**
Migrations are in `migrations/` folder and run automatically on container start via `init-db.sql`.

## Troubleshooting

### Container Unhealthy
```powershell
# Check logs
docker logs docker-chimera-pool-web-1

# Restart container
docker restart docker-chimera-pool-web-1

# Full rebuild
docker-compose -f deployments/docker/docker-compose.yml up --build -d chimera-pool-web
```

### Port Already in Use
```powershell
netstat -ano | findstr :8080
taskkill /PID <PID> /F
```

### Clean Rebuild
```powershell
docker-compose -f deployments/docker/docker-compose.yml down
docker-compose -f deployments/docker/docker-compose.yml up --build -d
```

## Service Ports

| Service | Internal Port | External Port |
|---------|--------------|---------------|
| API | 8080 | 8080 |
| Web Dashboard | 80 | 3000 |
| Nginx (HTTP) | 80 | 80 |
| Nginx (HTTPS) | 443 | 443 |
| Stratum | 3333, 18332 | 3333, 18332 |
| PostgreSQL | 5432 | 5432 |
| Redis | 6379 | 6379 |
| Prometheus | 9090 | 9090 |

## Important Files

- **Docker Compose**: `deployments/docker/docker-compose.yml`
- **API Dockerfile**: `deployments/docker/Dockerfile.api`
- **Web Dockerfile**: `deployments/docker/Dockerfile.web`
- **Nginx Config**: `deployments/docker/nginx.conf`
- **Environment**: `.env`
- **Migrations**: `migrations/`
