# Quick Start Guide

Get your Chimera Mining Pool up and running in minutes with this step-by-step guide.

## 🚀 Prerequisites

Before you begin, ensure you have:

- **Operating System**: Linux (Ubuntu 20.04+), macOS, or Windows
- **Docker**: Version 20.10+ ([Install Docker](https://docs.docker.com/get-docker/))
- **Docker Compose**: Version 2.0+ (included with Docker Desktop)
- **Git**: For cloning the repository
- **Minimum Hardware**: 4 CPU cores, 8GB RAM, 100GB storage

## 📥 Installation Methods

### Method 1: One-Click Installation (Recommended)

The fastest way to get started:

```bash
# Download and run the installer
curl -sSL https://raw.githubusercontent.com/your-org/chimera-mining-pool/main/install.sh | bash
```

This script will:
- Detect your system configuration
- Install required dependencies
- Configure the mining pool
- Start all services
- Provide access URLs and credentials

### Method 2: Docker Compose (Manual)

For more control over the installation:

```bash
# 1. Clone the repository
git clone https://github.com/your-org/chimera-mining-pool.git
cd chimera-mining-pool/chimera-pool-core

# 2. Copy environment configuration
cp .env.example .env

# 3. Generate secure secrets
./scripts/generate-secrets.sh

# 4. Start the services
docker-compose up -d

# 5. Wait for services to be ready (2-3 minutes)
./scripts/wait-for-services.sh

# 6. Initialize the database
./scripts/init-database.sh
```

### Method 3: Development Setup

For developers who want to run from source:

```bash
# 1. Clone and enter directory
git clone https://github.com/your-org/chimera-mining-pool.git
cd chimera-mining-pool/chimera-pool-core

# 2. Install dependencies
make install-deps

# 3. Set up development environment
make dev-setup

# 4. Start development services
make dev-start

# 5. Run the application
make run
```

## 🔧 Initial Configuration

### 1. Environment Variables

Edit the `.env` file to configure your pool:

```bash
# Pool Configuration
POOL_NAME="My Chimera Pool"
POOL_FEE=1.0
PAYOUT_THRESHOLD=1000000

# Database Configuration
POSTGRES_PASSWORD=your_secure_password
DATABASE_URL=postgres://chimera:your_secure_password@postgres:5432/chimera_pool

# Security Configuration
JWT_SECRET=your_jwt_secret_key_here
ENCRYPTION_KEY=your_32_character_encryption_key

# Network Configuration
API_PORT=8080
STRATUM_PORT=18332
WEB_PORT=3000
```

### 2. Generate Secure Secrets

```bash
# Generate all required secrets automatically
./scripts/generate-secrets.sh

# Or generate manually:
# JWT Secret (64 characters)
openssl rand -hex 32

# Encryption Key (32 characters)
openssl rand -hex 16
```

### 3. Configure Mining Algorithm

```bash
# Set your preferred BlockDAG algorithm
export MINING_ALGORITHM=blake2s
export DIFFICULTY_TARGET=0x1e0fffff
export BLOCK_TIME=60  # seconds
```

## 🌐 Accessing Your Pool

Once installation is complete, you can access:

### Web Dashboard
- **URL**: http://localhost:3000
- **Default Admin**: admin@chimera-pool.com
- **Default Password**: (generated during setup)

### API Endpoints
- **Base URL**: http://localhost:8080/api
- **Health Check**: http://localhost:8080/health
- **Metrics**: http://localhost:8080/metrics
- **Documentation**: http://localhost:8080/docs

### Stratum Server
- **Host**: localhost
- **Port**: 18332
- **Protocol**: Stratum v1

### Monitoring
- **Grafana**: http://localhost:3000/grafana
- **Prometheus**: http://localhost:9090

## ⛏️ Connecting Miners

### Configuration for Popular Miners

#### CGMiner / BFGMiner
```bash
cgminer --url stratum+tcp://localhost:18332 \
        --user your_username \
        --pass your_password \
        --algorithm blake2s
```

#### T-Rex Miner
```bash
t-rex -a blake2s \
      -o stratum+tcp://localhost:18332 \
      -u your_username \
      -p your_password
```

#### Custom Miner Configuration
```json
{
  "pools": [
    {
      "url": "stratum+tcp://localhost:18332",
      "user": "your_username",
      "pass": "your_password",
      "algorithm": "blake2s"
    }
  ]
}
```

### Creating Mining Accounts

1. **Via Web Interface**:
   - Go to http://localhost:3000/register
   - Create your account
   - Enable 2FA for security

2. **Via API**:
   ```bash
   curl -X POST http://localhost:8080/api/auth/register \
     -H "Content-Type: application/json" \
     -d '{
       "username": "miner1",
       "email": "miner1@example.com",
       "password": "SecurePassword123!"
     }'
   ```

## 📊 Monitoring Your Pool

### Key Metrics to Watch

1. **Pool Statistics**:
   - Active miners count
   - Total hashrate
   - Shares per second
   - Block discovery rate

2. **System Health**:
   - CPU and memory usage
   - Database performance
   - Network connectivity
   - Service availability

3. **Security Metrics**:
   - Failed login attempts
   - Rate limiting triggers
   - Suspicious activities

### Setting Up Alerts

```bash
# Configure email alerts
export ALERT_EMAIL=admin@yourpool.com
export SMTP_SERVER=smtp.gmail.com
export SMTP_PORT=587
export SMTP_USERNAME=your_email@gmail.com
export SMTP_PASSWORD=your_app_password

# Restart services to apply changes
docker-compose restart
```

## 🔒 Security Setup

### 1. Enable HTTPS

```bash
# Generate SSL certificates
./scripts/generate-ssl.sh

# Update docker-compose.yml to use HTTPS
# Uncomment SSL configuration sections
```

### 2. Configure Firewall

```bash
# Allow only necessary ports
sudo ufw allow 22    # SSH
sudo ufw allow 80    # HTTP
sudo ufw allow 443   # HTTPS
sudo ufw allow 18332 # Stratum
sudo ufw enable
```

### 3. Set Up Backup

```bash
# Configure automated backups
./scripts/setup-backup.sh

# Test backup and restore
./scripts/test-backup.sh
```

## 🚨 Troubleshooting

### Common Issues

#### Services Won't Start
```bash
# Check service logs
docker-compose logs

# Check system resources
docker system df
free -h
df -h
```

#### Database Connection Issues
```bash
# Check database status
docker-compose exec postgres pg_isready

# Reset database
docker-compose down -v
docker-compose up -d postgres
./scripts/init-database.sh
```

#### Miners Can't Connect
```bash
# Check Stratum server logs
docker-compose logs stratum-server

# Test Stratum connection
telnet localhost 18332

# Verify firewall settings
sudo ufw status
```

#### Performance Issues
```bash
# Check resource usage
docker stats

# Monitor system performance
htop
iotop
```

### Getting Help

1. **Check Logs**:
   ```bash
   # View all service logs
   docker-compose logs -f
   
   # View specific service logs
   docker-compose logs -f api-server
   ```

2. **Run Diagnostics**:
   ```bash
   # Run built-in diagnostics
   ./scripts/diagnose.sh
   
   # Check system health
   curl http://localhost:8080/health
   ```

3. **Community Support**:
   - GitHub Issues: [Report a bug](https://github.com/your-org/chimera-mining-pool/issues)
   - Discord: [Join our community](https://discord.gg/chimera-pool)
   - Documentation: [Full documentation](https://docs.chimera-pool.com)

## 🎯 Next Steps

Now that your pool is running:

1. **Configure Pool Settings**: Adjust fees, payout thresholds, and algorithms
2. **Set Up Monitoring**: Configure alerts and dashboards
3. **Invite Miners**: Share your pool details with miners
4. **Optimize Performance**: Tune settings based on your hardware
5. **Plan Scaling**: Prepare for growth with load balancing and clustering

### Recommended Reading

- [Configuration Reference](./configuration.md) - Detailed configuration options
- [Security Guide](./security-guide.md) - Comprehensive security setup
- [Monitoring Guide](./monitoring-guide.md) - Advanced monitoring and alerting
- [Deployment Guide](./deployment-guide.md) - Production deployment best practices

## 🎉 Congratulations!

Your Chimera Mining Pool is now ready to accept miners and start mining BlockDAG blocks. Welcome to the future of decentralized mining!

---

**Need help?** Join our [Discord community](https://discord.gg/chimera-pool) or check our [troubleshooting guide](./troubleshooting.md).