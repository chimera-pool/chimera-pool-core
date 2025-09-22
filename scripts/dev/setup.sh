#!/bin/bash
set -e

echo "🔧 Setting up Chimera Pool development environment..."

# Check prerequisites
check_prerequisites() {
    echo "📋 Checking prerequisites..."
    
    # Check Go
    if ! command -v go &> /dev/null; then
        echo "❌ Go is not installed. Please install Go 1.21+"
        exit 1
    fi
    
    GO_VERSION=$(go version | grep -o 'go[0-9]\+\.[0-9]\+' | sed 's/go//')
    if [[ $(echo "$GO_VERSION < 1.21" | bc -l) -eq 1 ]]; then
        echo "❌ Go version $GO_VERSION is too old. Please install Go 1.21+"
        exit 1
    fi
    echo "✅ Go $GO_VERSION"
    
    # Check Rust
    if ! command -v rustc &> /dev/null; then
        echo "❌ Rust is not installed. Please install Rust 1.70+"
        exit 1
    fi
    
    RUST_VERSION=$(rustc --version | grep -o '[0-9]\+\.[0-9]\+\.[0-9]\+')
    echo "✅ Rust $RUST_VERSION"
    
    # Check Node.js
    if ! command -v node &> /dev/null; then
        echo "❌ Node.js is not installed. Please install Node.js 18+"
        exit 1
    fi
    
    NODE_VERSION=$(node --version | sed 's/v//')
    echo "✅ Node.js $NODE_VERSION"
    
    # Check Docker
    if ! command -v docker &> /dev/null; then
        echo "❌ Docker is not installed. Please install Docker"
        exit 1
    fi
    echo "✅ Docker $(docker --version | grep -o '[0-9]\+\.[0-9]\+\.[0-9]\+')"
    
    # Check Docker Compose
    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
        echo "❌ Docker Compose is not installed"
        exit 1
    fi
    echo "✅ Docker Compose"
}

# Install Go dependencies
setup_go() {
    echo "📦 Installing Go dependencies..."
    go mod download
    go mod tidy
    echo "✅ Go dependencies installed"
}

# Install Rust dependencies
setup_rust() {
    echo "🦀 Installing Rust dependencies..."
    cargo fetch
    echo "✅ Rust dependencies installed"
}

# Install Node.js dependencies
setup_node() {
    echo "📦 Installing Node.js dependencies..."
    npm install
    echo "✅ Node.js dependencies installed"
}

# Setup development database
setup_database() {
    echo "🗄️ Setting up development database..."
    docker-compose -f deployments/docker/docker-compose.dev.yml up -d postgres redis
    
    # Wait for PostgreSQL to be ready
    echo "⏳ Waiting for PostgreSQL to be ready..."
    until docker-compose -f deployments/docker/docker-compose.dev.yml exec -T postgres pg_isready -U chimera; do
        sleep 1
    done
    echo "✅ Database is ready"
}

# Create project structure
create_structure() {
    echo "📁 Creating project structure..."
    
    # Create Go directories
    mkdir -p src/pool-manager/{cmd,internal/{auth,stratum,database,payout}}
    mkdir -p src/auth-service/{cmd,internal/{auth,middleware}}
    mkdir -p src/stratum-server/{cmd,internal/{protocol,connection}}
    mkdir -p src/web-dashboard/src/{components,pages,services,types}
    
    # Create test directories
    mkdir -p tests/{unit,integration,e2e}
    mkdir -p tests/fixtures
    
    # Create algorithm engine structure
    mkdir -p src/algorithm-engine/src
    
    echo "✅ Project structure created"
}

# Run setup
main() {
    check_prerequisites
    create_structure
    setup_go
    setup_rust
    setup_node
    setup_database
    
    echo ""
    echo "🎉 Development environment setup complete!"
    echo ""
    echo "Next steps:"
    echo "  1. Run tests: ./scripts/test.sh"
    echo "  2. Start development: ./scripts/dev/start.sh"
    echo "  3. Access database: http://localhost:8080 (adminer)"
    echo ""
}

main "$@"
