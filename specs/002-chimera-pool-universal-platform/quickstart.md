# Quick Start Guide: Chimera Pool Universal Platform

## Overview

This quick start guide will help you get the Chimera Pool Universal Platform up and running in under 10 minutes. By the end of this guide, you'll have a fully functional mining pool supporting multiple cryptocurrencies.

## Prerequisites

### System Requirements
- **Operating System**: Linux (Ubuntu 20.04+), macOS (10.15+), or Windows 10+
- **Memory**: Minimum 8GB RAM (16GB recommended)
- **Storage**: Minimum 50GB free space (SSD recommended)
- **Network**: Stable internet connection with public IP (for production)
- **CPU**: 4+ cores recommended

### Required Software
The installer will automatically install these if missing:
- Docker and Docker Compose
- Git
- curl/wget

## Installation Methods

### Method 1: One-Click Installation (Recommended)

#### For Pool Operators
```bash
# Download and run the universal installer
curl -fsSL https://install.chimera-pool.com | bash

# Or with wget
wget -qO- https://install.chimera-pool.com | bash
```

The installer will:
1. Detect your system configuration
2. Install all required dependencies
3. Download and configure Chimera Pool
4. Start all services
5. Display your pool dashboard URL

#### For Miners
```bash
# Download and run the miner installer
curl -fsSL https://miner.chimera-pool.com | bash -s -- --pool YOUR_POOL_URL

# Or specify your wallet address
curl -fsSL https://miner.chimera-pool.com | bash -s -- --pool YOUR_POOL_URL --wallet YOUR_WALLET_ADDRESS
```

### Method 2: Docker Installation

```bash
# Clone the repository
git clone https://github.com/chimera-pool/chimera-pool-core.git
cd chimera-pool-core

# Run the setup script
./scripts/setup-simple.sh

# Start with Docker Compose
docker-compose up -d

# Check status
docker-compose ps
```

### Method 3: Manual Installation

```bash
# Clone the repository
git clone https://github.com/chimera-pool/chimera-pool-core.git
cd chimera-pool-core

# Install dependencies (Ubuntu/Debian)
sudo apt update
sudo apt install -y postgresql redis-server nginx golang-go nodejs npm

# Build the application
make build

# Initialize the database
make init-db

# Start services
make start
```

## Initial Configuration

### 1. Access the Dashboard

After installation, open your web browser and navigate to:
- **Local installation**: http://localhost:8080
- **Cloud installation**: http://YOUR_SERVER_IP:8080

### 2. Complete Setup Wizard

The setup wizard will guide you through:

1. **Administrator Account Creation**
   - Create your admin username and password
   - Set up multi-factor authentication (recommended)

2. **Pool Configuration**
   - Choose which cryptocurrencies to support
   - Configure payout settings
   - Set pool fees and thresholds

3. **Network Configuration**
   - Configure Stratum ports for each cryptocurrency
   - Set up SSL certificates (optional but recommended)
   - Configure firewall rules

4. **Wallet Configuration**
   - Add wallet addresses for each supported cryptocurrency
   - Configure payout methods and schedules
   - Set minimum payout thresholds

### 3. Verify Installation

Check that all services are running:

```bash
# Check service status
curl http://localhost:8080/api/health

# Check Stratum servers
telnet localhost 3333  # Bitcoin
telnet localhost 3334  # Ethereum Classic
telnet localhost 3335  # BlockDAG
```

## Supported Cryptocurrencies

### Default Supported Coins
- **Bitcoin (BTC)**: Port 3333, SHA-256 algorithm
- **Ethereum Classic (ETC)**: Port 3334, Ethash algorithm
- **BlockDAG (BDAG)**: Port 3335, Blake3 algorithm
- **Litecoin (LTC)**: Port 3336, Scrypt algorithm

### Adding New Cryptocurrencies

1. **Through Dashboard**:
   - Navigate to "Algorithm Management"
   - Click "Add New Cryptocurrency"
   - Upload algorithm package or select from marketplace

2. **Through Command Line**:
   ```bash
   # Install new algorithm package
   ./chimera-pool algorithm install --package dash-x11-v1.0.0.zip
   
   # Create new pool
   ./chimera-pool pool create --coin dash --algorithm x11 --port 3337
   ```

## Connecting Miners

### For GPU Miners (T-Rex Example)
```bash
# Bitcoin mining
./t-rex -a sha256 -o stratum+tcp://YOUR_POOL_IP:3333 -u YOUR_WALLET.WORKER_NAME

# BlockDAG mining
./t-rex -a blake3 -o stratum+tcp://YOUR_POOL_IP:3335 -u YOUR_WALLET.WORKER_NAME
```

### For CPU Miners (cpuminer-opt Example)
```bash
# Litecoin mining
./cpuminer -a scrypt -o stratum+tcp://YOUR_POOL_IP:3336 -u YOUR_WALLET.WORKER_NAME
```

### For ASIC Miners
Configure your ASIC miner with:
- **URL**: stratum+tcp://YOUR_POOL_IP:PORT
- **Username**: YOUR_WALLET_ADDRESS.WORKER_NAME
- **Password**: x (or any value)

## Monitoring Your Pool

### Dashboard Features
- **Real-time Statistics**: Pool hashrate, active miners, blocks found
- **Miner Management**: Individual miner statistics and configuration
- **Payout Tracking**: Automatic payout calculations and history
- **Algorithm Management**: Hot-swap algorithms without downtime

### API Access
```bash
# Get pool statistics
curl http://localhost:8080/api/pools/bitcoin/stats

# Get miner information
curl http://localhost:8080/api/miners/YOUR_MINER_ID

# Get payout history
curl http://localhost:8080/api/payouts?user=YOUR_USER_ID
```

## Common Configuration Examples

### High-Performance Setup
```yaml
# docker-compose.override.yml
version: '3.8'
services:
  pool-manager:
    deploy:
      resources:
        limits:
          memory: 4G
        reservations:
          memory: 2G
    environment:
      - MAX_CONNECTIONS=25000
      - WORKER_THREADS=16
```

### Multi-Region Setup
```bash
# Deploy to multiple regions
./scripts/deploy-cloud.sh --provider aws --regions us-east-1,eu-west-1,ap-southeast-1
```

### SSL/TLS Configuration
```bash
# Enable SSL for Stratum connections
./chimera-pool config set --stratum-ssl true --cert-path /path/to/cert.pem --key-path /path/to/key.pem
```

## Troubleshooting

### Common Issues

#### Services Not Starting
```bash
# Check logs
docker-compose logs pool-manager
docker-compose logs stratum-server

# Restart services
docker-compose restart
```

#### Miners Can't Connect
```bash
# Check firewall
sudo ufw status
sudo ufw allow 3333:3340/tcp

# Check Stratum server
telnet localhost 3333
```

#### Database Connection Issues
```bash
# Check PostgreSQL status
sudo systemctl status postgresql

# Reset database
make reset-db
```

### Getting Help

- **Documentation**: https://docs.chimera-pool.com
- **Community Forum**: https://community.chimera-pool.com
- **GitHub Issues**: https://github.com/chimera-pool/chimera-pool-core/issues
- **Discord**: https://discord.gg/chimera-pool

## Next Steps

### For Pool Operators
1. **Customize Your Pool**: Configure branding, fees, and payout methods
2. **Add More Cryptocurrencies**: Expand your pool's offerings
3. **Set Up Monitoring**: Configure alerts and monitoring dashboards
4. **Optimize Performance**: Tune settings for your specific hardware

### For Miners
1. **Optimize Mining Settings**: Fine-tune your mining software configuration
2. **Monitor Performance**: Track your hashrate and earnings
3. **Join the Community**: Connect with other miners and pool operators
4. **Provide Feedback**: Help improve the platform

### For Developers
1. **Explore APIs**: Build custom tools and integrations
2. **Contribute Algorithms**: Develop support for new cryptocurrencies
3. **Create Plugins**: Extend the platform's functionality
4. **Join Development**: Contribute to the open-source project

## Security Best Practices

### For Pool Operators
- Enable multi-factor authentication
- Use strong, unique passwords
- Keep software updated
- Monitor logs for suspicious activity
- Use SSL/TLS for all connections
- Regular security audits

### For Miners
- Verify pool authenticity before connecting
- Use dedicated mining wallets
- Monitor your mining statistics regularly
- Keep mining software updated
- Use secure network connections

---

**Congratulations!** You now have a fully functional universal mining pool platform. The Chimera Pool platform will continue to evolve with new features, cryptocurrencies, and optimizations. Join our community to stay updated and contribute to the project's growth.

