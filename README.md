# Chimera Pool - Universal Mining Pool Platform

🚀 **The next-generation universal mining pool platform supporting multiple cryptocurrencies**

## Overview

Chimera Pool is a revolutionary mining pool platform that supports multiple cryptocurrencies through its hot-swappable algorithm engine. Built with enterprise-grade performance, security, and ease of use in mind.

### Supported Cryptocurrencies

- **Bitcoin** (SHA-256)
- **Ethereum Classic** (Ethash)
- **BlockDAG** (Blake3)
- **Litecoin** (Scrypt)
- **Dash** (X11)
- **Monero** (RandomX)
- **Zcash** (Equihash)
- **And many more...**

## Key Features

### 🎯 Universal Support
- Support ANY proof-of-work cryptocurrency
- Hot-swappable algorithms for instant coin additions
- One platform, infinite possibilities

### ⚡ Enterprise Performance
- Handle 10,000+ concurrent miners per pool
- Sub-100ms response times
- 99.9% uptime guarantee

### 🛡️ Security First
- Enterprise-grade security
- Multi-factor authentication
- Comprehensive audit logging

### 🎛️ Easy Management
- One-click deployment
- Universal dashboard for all pools
- Automated optimization

## Quick Start

```bash
# Clone the repository
git clone https://github.com/chimera-pool/chimera-pool-core.git
cd chimera-pool-core

# Run one-click installer
./scripts/install.sh

# Access dashboard
open http://localhost:8080
```

## Technology Stack

- **Go**: Pool management, Stratum server, API services
- **Rust**: High-performance algorithm engine
- **React + TypeScript**: Modern web dashboard with cyber-minimal theme
- **PostgreSQL**: Primary data store
- **Redis**: High-performance caching
- **Docker**: Containerized deployment

## Development

### Prerequisites

- Go 1.21+
- Rust 1.70+
- Node.js 18+
- Docker & Docker Compose
- PostgreSQL 15+
- Redis 7+

### Setup Development Environment

```bash
# Install dependencies
./scripts/dev/setup.sh

# Start development services
docker-compose -f deployments/docker/docker-compose.dev.yml up -d

# Run tests
./scripts/test.sh

# Start development server
./scripts/dev/start.sh
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

**Built with ❤️ by the Chimera Pool Team**
